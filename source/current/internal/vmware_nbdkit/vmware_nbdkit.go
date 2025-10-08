package vmware_nbdkit

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/internal/nbdcopy"
	"github.com/vexxhost/migratekit/internal/nbdkit"
	"github.com/vexxhost/migratekit/internal/progress"
	"github.com/vexxhost/migratekit/internal/target"
	vmware "github.com/vexxhost/migratekit/internal/vmware"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"libguestfs.org/libnbd"
)

const MaxChunkSize = 64 * 1024 * 1024

type VddkConfig struct {
	Debug       bool
	Endpoint    *url.URL
	Thumbprint  string
	Compression nbdkit.CompressionMethod
	Quiesce     bool
}

type NbdkitServers struct {
	VddkConfig     *VddkConfig
	VirtualMachine *object.VirtualMachine
	SnapshotRef    types.ManagedObjectReference
	Servers        []*NbdkitServer
}

type NbdkitServer struct {
	Servers *NbdkitServers
	Disk    *types.VirtualDisk
	Nbdkit  *nbdkit.NbdkitServer
}

func NewNbdkitServers(vddk *VddkConfig, vm *object.VirtualMachine) *NbdkitServers {
	return &NbdkitServers{
		VddkConfig:     vddk,
		VirtualMachine: vm,
		Servers:        []*NbdkitServer{},
	}
}

func (s *NbdkitServers) createSnapshot(ctx context.Context) error {
	task, err := s.VirtualMachine.CreateSnapshot(ctx, "migratekit", "Ephemeral snapshot for MigrateKit", false, s.VddkConfig.Quiesce)
	if err != nil {
		return err
	}

	bar := progress.NewVMwareProgressBar("Creating snapshot")
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		bar.Loop(ctx.Done())
	}()
	defer cancel()

	info, err := task.WaitForResult(ctx, bar)
	if err != nil {
		return err
	}

	s.SnapshotRef = info.Result.(types.ManagedObjectReference)
	return nil
}

func (s *NbdkitServers) Start(ctx context.Context) error {
	err := s.createSnapshot(ctx)
	if err != nil {
		return err
	}

	var snapshot mo.VirtualMachineSnapshot
	err = s.VirtualMachine.Properties(ctx, s.SnapshotRef, []string{"config.hardware"}, &snapshot)
	if err != nil {
		return err
	}

	for _, device := range snapshot.Config.Hardware.Device {
		switch disk := device.(type) {
		case *types.VirtualDisk:
			backing := disk.Backing.(types.BaseVirtualDeviceFileBackingInfo)
			info := backing.GetVirtualDeviceFileBackingInfo()

			password, _ := s.VddkConfig.Endpoint.User.Password()
			server, err := nbdkit.NewNbdkitBuilder().
				Server(s.VddkConfig.Endpoint.Host).
				Username(s.VddkConfig.Endpoint.User.Username()).
				Password(password).
				Thumbprint(s.VddkConfig.Thumbprint).
				VirtualMachine(s.VirtualMachine.Reference().Value).
				Snapshot(s.SnapshotRef.Value).
				Filename(info.FileName).
				Compression(s.VddkConfig.Compression).
				Build()
			if err != nil {
				return err
			}

			if err := server.Start(); err != nil {
				return err
			}

			s.Servers = append(s.Servers, &NbdkitServer{
				Servers: s,
				Disk:    disk,
				Nbdkit:  server,
			})
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Warn("Received interrupt signal, cleaning up...")

		err := s.Stop(ctx)
		if err != nil {
			log.WithError(err).Fatal("Failed to stop nbdkit servers")
		}

		os.Exit(1)
	}()

	return nil
}

func (s *NbdkitServers) removeSnapshot(ctx context.Context) error {
	consolidate := true
	task, err := s.VirtualMachine.RemoveSnapshot(ctx, s.SnapshotRef.Value, false, &consolidate)
	if err != nil {
		return err
	}

	bar := progress.NewVMwareProgressBar("Removing snapshot")
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		bar.Loop(ctx.Done())
	}()
	defer cancel()

	_, err = task.WaitForResult(ctx, bar)
	if err != nil {
		return err
	}

	return nil
}

func (s *NbdkitServers) Stop(ctx context.Context) error {
	for _, server := range s.Servers {
		if err := server.Nbdkit.Stop(); err != nil {
			return err
		}
	}

	err := s.removeSnapshot(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *NbdkitServers) MigrationCycle(ctx context.Context, runV2V bool) error {
	err := s.Start(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err := s.Stop(ctx)
		if err != nil {
			log.WithError(err).Fatal("Failed to stop nbdkit servers")
		}
	}()

	for index, server := range s.Servers {
		t, err := target.NewCloudStack(ctx, s.VirtualMachine, server.Disk)
		if err != nil {
			return err
		}

		if index != 0 {
			runV2V = false
		}

		err = server.SyncToTarget(ctx, t, runV2V)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *NbdkitServer) FullCopyToTarget(t target.Target, path string, targetIsClean bool) error {
	logger := log.WithFields(log.Fields{
		"vm":   s.Servers.VirtualMachine.Name(),
		"disk": s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName,
	})

	logger.Info("Starting full copy")

	// 🎯 PROPER CBT IMPLEMENTATION: Calculate actual used space using VMware APIs
	ctx := context.Background()
	diskInfo, err := vmware.CalculateUsedSpace(ctx, s.Servers.VirtualMachine, s.Disk, s.Servers.SnapshotRef)
	if err != nil {
		logger.WithError(err).Warn("CBT calculation failed, falling back to total capacity")
		// Fallback to original behavior for compatibility
		err = nbdcopy.Run(
			s.Nbdkit.LibNBDExportName(),
			path,
			s.getActualDiskSize(), // Use actual VMDK file size instead of logical capacity
			targetIsClean,
		)
	} else {
		// Use accurate progress tracking based on actual used space
		usedBytes := diskInfo.GetUsedBytes()
		logger.Infof("📊 Using CBT-calculated disk usage: %d GB used of %d GB total",
			diskInfo.UsedGB, diskInfo.SizeGB)

		err = nbdcopy.Run(
			s.Nbdkit.LibNBDExportName(),
			path,
			usedBytes, // 🎯 Use actual used space for accurate progress!
			targetIsClean,
		)
	}

	if err != nil {
		return err
	}

	logger.Info("Full copy completed")

	return nil
}

func (s *NbdkitServer) IncrementalCopyToTarget(ctx context.Context, t target.Target, path string) error {
	logger := log.WithFields(log.Fields{
		"vm":   s.Servers.VirtualMachine.Name(),
		"disk": s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName,
	})

	logger.Info("Starting incremental copy")

	// 🎯 INITIALIZE PROGRESS TRACKING FOR REAL DATA VOLUME
	jobID := progress.GetJobIDFromContext()
	progressEndpoint := progress.GetVMAProgressEndpoint()
	
	// Calculate planned bytes based on CBT change areas (will be updated after QueryChangedDiskAreas)
	plannedBytes := s.getActualDiskSize() // Initial estimate, refined during CBT query
	
	// Create progress tracker with real data volume tracking
	tracker := progress.NewDataTracker(jobID, plannedBytes, progressEndpoint)
	tracker.SetStage(progress.StageQueryCBT)
	
	// Start periodic SNA updates every 2 seconds
	notifier := progress.NewVMAProgressNotifier(tracker)
	notifier.StartPeriodicUpdates()
	
	logger.Info("🎯 Progress tracking initialized for job:", jobID)

	currentChangeId, err := t.GetCurrentChangeID(ctx)
	if err != nil {
		return err
	}

	// 🎯 PROPER CBT SELECTIVE INCREMENTAL IMPLEMENTATION: Copy only changed blocks
	handle, err := libnbd.Create()
	if err != nil {
		return err
	}

	err = handle.ConnectUri(s.Nbdkit.LibNBDExportName())
	if err != nil {
		return err
	}
	defer handle.Close()
	
	// 🚀 WRAP SOURCE HANDLE WITH PROGRESS TRACKING
	sourceHandle := progress.NewLibNBDProgressWrapper(handle, tracker)

	// 🔧 Handle NBD, pipes, and regular files with proper selective copying
	var fd *os.File
	var nbdTarget *progress.LibNBDProgressWrapper
	var isNBD bool
	var isNamedPipe bool

	// Check target type: NBD, named pipe, or regular file
	if strings.HasPrefix(path, "nbd://") {
		// NBD target - get handle from CloudStack target
		if cloudStackTarget, ok := t.(*target.CloudStack); ok {
			rawNbdTarget := cloudStackTarget.GetNBDHandle()
			if rawNbdTarget == nil {
				return fmt.Errorf("NBD target not connected")
			}
			// 🚀 WRAP TARGET HANDLE WITH PROGRESS TRACKING
			nbdTarget = progress.NewLibNBDProgressWrapper(rawNbdTarget, tracker)
			isNBD = true
			logger.Info("🌐 Using NBD positioned writes for selective incremental copy with progress tracking")
		} else {
			return fmt.Errorf("NBD path but not CloudStack target")
		}
	} else if strings.Contains(path, "cloudstack_stream_") {
		// Named pipe - use simple write-only flags
		fd, err = os.OpenFile(path, os.O_WRONLY, 0644)
		isNamedPipe = true
		logger.Info("📡 Opened CloudStack streaming pipe for selective incremental copy")
		if err != nil {
			return err
		}
		defer fd.Close()
	} else {
		// Regular file - use original flags for direct I/O
		fd, err = os.OpenFile(path, os.O_WRONLY|os.O_EXCL|syscall.O_DIRECT, 0644)
		isNamedPipe = false
		logger.Info("💾 Opened regular file for selective incremental copy")
		if err != nil {
			return err
		}
		defer fd.Close()
	}

	startOffset := int64(0)
	bar := progress.DataProgressBar("Incremental copy", s.getActualDiskSize()) // Use actual VMDK file size
	
	// 🎯 UPDATE STAGE TO TRANSFER
	tracker.SetStage(progress.StageTransfer)
	logger.Info("🚀 Starting data transfer stage with real-time progress tracking")

	for {
		req := types.QueryChangedDiskAreas{
			This:        s.Servers.VirtualMachine.Reference(),
			Snapshot:    &s.Servers.SnapshotRef,
			DeviceKey:   s.Disk.Key,
			StartOffset: startOffset,
			ChangeId:    currentChangeId.Value,
		}

		res, err := methods.QueryChangedDiskAreas(ctx, s.Servers.VirtualMachine.Client(), &req)
		if err != nil {
			return err
		}

		diskChangeInfo := res.Returnval

		// 🎯 SELECTIVE BLOCK COPYING: Only copy changed areas, not entire disk
		for _, area := range diskChangeInfo.ChangedArea {
			for offset := area.Start; offset < area.Start+area.Length; {
				chunkSize := area.Length - (offset - area.Start)
				if chunkSize > MaxChunkSize {
					chunkSize = MaxChunkSize
				}

				buf := make([]byte, chunkSize)
				// 🚀 READ WITH PROGRESS TRACKING: Real data bytes tracked automatically
				err = sourceHandle.Pread(buf, uint64(offset), nil)
				if err != nil {
					return err
				}

				// 🔧 Handle NBD, pipes, and files differently for writing
				if isNBD {
					// 🚀 NBD WRITE WITH PROGRESS TRACKING: Real data bytes tracked automatically
					err = nbdTarget.Pwrite(buf, uint64(offset), nil)
				} else if isNamedPipe {
					// Named pipe: sequential write (CloudStack handles positioning)
					// 🎯 MANUAL PROGRESS TRACKING for non-NBD writes
					_, err = fd.Write(buf)
					if err == nil {
						tracker.OnDataTransfer(int64(len(buf)))
					}
				} else {
					// Regular file: positioned write for random access
					// 🎯 MANUAL PROGRESS TRACKING for non-NBD writes
					_, err = fd.WriteAt(buf, offset)
					if err == nil {
						tracker.OnDataTransfer(int64(len(buf)))
					}
				}

				if err != nil {
					return err
				}

				bar.Set64(offset + chunkSize)
				offset += chunkSize
			}
		}

		startOffset = diskChangeInfo.StartOffset + diskChangeInfo.Length
		bar.Set64(startOffset)

		if startOffset == s.getActualDiskSize() { // Use actual VMDK file size
			break
		}
	}

	// 🎯 UPDATE STAGE TO FINALIZE
	tracker.SetStage(progress.StageFinalize)
	logger.Info("🎯 Incremental copy completed via selective block copying with progress tracking")
	
	// Send final progress update to SNA
	err = tracker.UpdateVMAEndpoint()
	if err != nil {
		logger.Warn("Failed to send final progress update:", err)
	}
	
	return nil
}

// calculateDeltaSize calculates the total size of changed areas using VMware CBT
func (s *NbdkitServer) calculateDeltaSize(ctx context.Context, currentChangeId *vmware.ChangeID) (int64, error) {
	logger := log.WithFields(log.Fields{
		"vm":   s.Servers.VirtualMachine.Name(),
		"disk": s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName,
	})

	logger.Infof("🔍 Calculating delta size since ChangeID: %s", currentChangeId.Value)

	var totalDeltaSize int64 = 0
	startOffset := int64(0)

	for {
		req := types.QueryChangedDiskAreas{
			This:        s.Servers.VirtualMachine.Reference(),
			Snapshot:    &s.Servers.SnapshotRef,
			DeviceKey:   s.Disk.Key,
			StartOffset: startOffset,
			ChangeId:    currentChangeId.Value,
		}

		res, err := methods.QueryChangedDiskAreas(ctx, s.Servers.VirtualMachine.Client(), &req)
		if err != nil {
			return 0, err
		}

		diskChangeInfo := res.Returnval

		// Add up the size of all changed areas
		for _, area := range diskChangeInfo.ChangedArea {
			totalDeltaSize += area.Length
			logger.Debugf("📊 Changed area: offset=%d, length=%d", area.Start, area.Length)
		}

		// Move to next batch
		startOffset = diskChangeInfo.StartOffset + diskChangeInfo.Length

		// Check if we've processed all areas
		if startOffset >= s.getActualDiskSize() { // Use actual VMDK file size
			break
		}
	}

	logger.Infof("📊 Total delta size calculated: %.2f MB (%d bytes)",
		float64(totalDeltaSize)/(1024*1024), totalDeltaSize)

	return totalDeltaSize, nil
}

func (s *NbdkitServer) SyncToTarget(ctx context.Context, t target.Target, runV2V bool) error {
	snapshotChangeId, err := vmware.GetChangeID(s.Disk)
	if err != nil {
		return err
	}

	needFullCopy, targetIsClean, err := target.NeedsFullCopy(ctx, t)
	if err != nil {
		return err
	}

	err = t.Connect(ctx)
	if err != nil {
		return err
	}
	defer t.Disconnect(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Warn("Received interrupt signal, cleaning up...")

		err := t.Disconnect(ctx)
		if err != nil {
			log.WithError(err).Fatal("Failed to disconnect from target")
		}

		os.Exit(1)
	}()

	path, err := t.GetPath(ctx)
	if err != nil {
		return err
	}

	if needFullCopy {
		err = s.FullCopyToTarget(t, path, targetIsClean)
		if err != nil {
			return err
		}
	} else {
		err = s.IncrementalCopyToTarget(ctx, t, path)
		if err != nil {
			return err
		}
	}

	if runV2V {
		log.Info("Running virt-v2v-in-place")

		os.Setenv("LIBGUESTFS_BACKEND", "direct")

		var cmd *exec.Cmd
		if s.Servers.VddkConfig.Debug {
			cmd = exec.Command("virt-v2v-in-place", "-v", "-x", "-i", "disk", path)
		} else {
			cmd = exec.Command("virt-v2v-in-place", "-i", "disk", path)
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return err
		}

		err = t.WriteChangeID(ctx, &vmware.ChangeID{})
		if err != nil {
			return err
		}
	} else {
		err = t.WriteChangeID(ctx, snapshotChangeId)
		if err != nil {
			return err
		}
	}

	return nil
}

// getActualDiskSize returns the actual VMDK file size from VM layout information
func (s *NbdkitServer) getActualDiskSize() int64 {
	// Get VM managed object with layout information
	var props []string
	props = append(props, "layoutEx")
	
	var vmMo mo.VirtualMachine
	ctx := context.Background()
	err := s.Servers.VirtualMachine.Properties(ctx, s.Servers.VirtualMachine.Reference(), props, &vmMo)
	if err != nil {
		log.WithError(err).Warn("Failed to get VM layout, falling back to logical capacity")
		return s.Disk.CapacityInBytes
	}

	if vmMo.LayoutEx == nil || vmMo.LayoutEx.File == nil {
		log.Warn("VM layout information not available, falling back to logical capacity")
		return s.Disk.CapacityInBytes
	}

	var totalSize int64 = 0

	// Look for files associated with this disk key
	for _, file := range vmMo.LayoutEx.File {
		if file.Key == s.Disk.Key {
			totalSize += file.Size
		}
	}

	if totalSize > 0 {
		log.WithFields(log.Fields{
			"disk_key":          s.Disk.Key,
			"logical_size_gb":   s.Disk.CapacityInBytes / 1073741824,
			"actual_size_gb":    float64(totalSize) / 1073741824,
			"actual_size_bytes": totalSize,
		}).Info("✅ Using actual VMDK file size instead of logical capacity")
		return totalSize
	}

	log.WithFields(log.Fields{
		"disk_key":        s.Disk.Key,
		"logical_size_gb": s.Disk.CapacityInBytes / 1073741824,
	}).Warn("Could not determine actual disk size from VM layout, falling back to logical capacity")
	return s.Disk.CapacityInBytes
}
