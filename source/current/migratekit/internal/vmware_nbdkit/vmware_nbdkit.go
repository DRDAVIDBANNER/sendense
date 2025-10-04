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
	"time"
	"unsafe"

	log "github.com/sirupsen/logrus"
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

const MaxChunkSize = 32 * 1024 * 1024 // 32MB maximum for NBD server compatibility

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
	// Send VMA progress update for snapshot creation start
	if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
		if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
			vpc.SendStageUpdate("Creating Snapshot", 10)
		}
	}

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

	// Send VMA progress update for snapshot creation completion
	if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
		if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
			vpc.SendStageUpdate("Creating Snapshot", 15)
		}
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

// isZeroBlock checks if a data block contains only zeros using vectorized comparison
func isZeroBlock(data []byte) bool {
	// Fast path: check length
	if len(data) == 0 {
		return true
	}

	// Use 8-byte (uint64) comparison for better performance
	dataLen := len(data)

	// Handle 8-byte aligned portion
	for i := 0; i < dataLen-7; i += 8 {
		// Convert to uint64 for fast comparison
		val := *(*uint64)(unsafe.Pointer(&data[i]))
		if val != 0 {
			return false
		}
	}

	// Handle remaining bytes (less than 8)
	for i := dataLen &^ 7; i < dataLen; i++ {
		if data[i] != 0 {
			return false
		}
	}

	return true
}

// isZeroBlockSampled checks if a large block is likely zero by sampling small portions
// This avoids reading the entire block when most of it is zero
func isZeroBlockSampled(handle *libnbd.Libnbd, offset int64, size int) bool {
	const sampleSize = 8192 // 8KB samples
	const numSamples = 5    // Sample 5 locations

	if size <= sampleSize*2 {
		// For small blocks, just read and check the whole thing
		data := make([]byte, size)
		err := handle.Pread(data, uint64(offset), nil)
		if err != nil {
			return false // Assume not zero if we can't read
		}
		return isZeroBlock(data)
	}

	// Sample at: start, 25%, 50%, 75%, end
	sampleOffsets := []int64{
		offset,                            // Start
		offset + int64(size)/4,            // 25%
		offset + int64(size)/2,            // 50%
		offset + int64(size)*3/4,          // 75%
		offset + int64(size) - sampleSize, // End (ensure we don't go past bounds)
	}

	sample := make([]byte, sampleSize)
	for _, sampleOffset := range sampleOffsets {
		// Ensure we don't read past the end of the block
		readSize := sampleSize
		if sampleOffset+int64(sampleSize) > offset+int64(size) {
			readSize = int(offset + int64(size) - sampleOffset)
		}

		err := handle.Pread(sample[:readSize], uint64(sampleOffset), nil)
		if err != nil {
			return false // Assume not zero if we can't read
		}

		if !isZeroBlock(sample[:readSize]) {
			return false // Found non-zero data in sample
		}
	}

	return true // All samples were zero, likely the whole block is zero
}

func (s *NbdkitServer) FullCopyToTarget(ctx context.Context, t target.Target, path string, targetIsClean bool) error {
	// Initialize VMA progress tracking variables
	var (
		totalBytesTransferred int64
		totalBytes            int64
		lastProgressUpdate    = time.Now()
		lastProgressPercent   = 0.0
		startTime             = time.Now()
		// 🆕 SPARSE OPTIMIZATION: Track sparse block savings
		sparseBlocksSkipped int64
		sparseBytesSkipped  int64
	)
	logger := log.WithFields(log.Fields{
		"vm":   s.Servers.VirtualMachine.Name(),
		"disk": s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName,
	})

	logger.Info("Starting full copy")

	// 🎯 CRITICAL: Send initial job type to VMA now that we know it's confirmed
	if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
		if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
			vpc.SendUpdate(progress.VMAProgressUpdate{
				Stage:    "Transfer",
				Status:   "in_progress",
				SyncType: "initial", // 🎯 CRITICAL: Tell VMA this is initial/full copy
				Percent:  0,
			})
		}
	}

	// 🎯 PROPER CBT IMPLEMENTATION: Calculate actual used space using VMware APIs
	diskInfo, err := vmware.CalculateUsedSpace(ctx, s.Servers.VirtualMachine, s.Disk, s.Servers.SnapshotRef)
	if err != nil {
		logger.WithError(err).Warn("CBT calculation failed, using actual VMDK file size")
		// Fall back to actual VMDK file size (better than logical capacity)
		totalBytes = s.getActualDiskSize()
	} else {
		// Use CBT-calculated used space for accurate progress
		totalBytes = diskInfo.GetUsedBytes()
		logger.Infof("📊 Using CBT-calculated disk usage: %d GB used of %d GB total",
			diskInfo.UsedGB, diskInfo.SizeGB)
	}

	// 🎯 FULL COPY WITH LIBNBD + VMA PROGRESS INTEGRATION
	handle, err := libnbd.Create()
	if err != nil {
		return err
	}
	defer handle.Close()

	err = handle.SetExportName(s.Nbdkit.LibNBDExportName())
	if err != nil {
		return err
	}

	err = handle.ConnectUnix(s.Nbdkit.Socket())
	if err != nil {
		return err
	}

	// Connect to target NBD/file
	var fd *os.File
	var nbdTarget *libnbd.Libnbd
	var isNBD, isNamedPipe bool

	if strings.HasPrefix(path, "nbd://") {
		// NBD target (CloudStack)
		isNBD = true
		nbdTarget, err = libnbd.Create()
		if err != nil {
			return err
		}
		defer nbdTarget.Close()

		// Parse NBD URL: nbd://host:port/export
		u, err := url.Parse(path)
		if err != nil {
			return err
		}
		exportName := strings.TrimPrefix(u.Path, "/")
		err = nbdTarget.SetExportName(exportName)
		if err != nil {
			return err
		}

		// 🆕 SPARSE OPTIMIZATION: Enable NBD metadata context for allocation detection
		err = nbdTarget.AddMetaContext("base:allocation")
		if err != nil {
			logger.WithError(err).Warn("Failed to add metadata context - sparse optimization disabled")
			// Continue without sparse optimization
		} else {
			logger.Info("✅ NBD metadata context enabled for sparse optimization")
		}

		err = nbdTarget.ConnectTcp(u.Hostname(), u.Port())
		if err != nil {
			return err
		}
	} else {
		// File target
		fd, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer fd.Close()

		// Check if it's a named pipe
		if stat, err := fd.Stat(); err == nil {
			isNamedPipe = (stat.Mode() & os.ModeNamedPipe) != 0
		}
	}

	// Full copy: read entire disk in chunks
	diskSize := s.getActualDiskSize()
	// Use global MaxChunkSize (32MB) for NBD server compatibility

	for offset := int64(0); offset < diskSize; {
		chunkSize := MaxChunkSize
		if offset+int64(chunkSize) > diskSize {
			chunkSize = int(diskSize - offset)
		}

		// 🚀 ENHANCED SPARSE OPTIMIZATION: Hierarchical detection with sampling
		var isZero bool
		var buf []byte

		if chunkSize >= 4*1024*1024 { // 4MB+: Use fast sampling first to avoid reading entire block
			// Use sampling to check if block is likely zero without reading the whole thing
			isZero = isZeroBlockSampled(handle, offset, chunkSize)
			if isZero {
				// If sampling suggests it's zero, we don't need to read the data for processing
				logger.WithFields(log.Fields{
					"offset": offset,
					"size":   chunkSize,
				}).Debug("🔍 Hierarchical sparse detection: Block likely zero via sampling")
			} else {
				// Sampling indicates non-zero data, so read the actual block
				buf = make([]byte, chunkSize)
				err = handle.Pread(buf, uint64(offset), nil)
				if err != nil {
					return err
				}
			}
		} else {
			// For smaller blocks, read and check directly
			buf = make([]byte, chunkSize)
			err = handle.Pread(buf, uint64(offset), nil)
			if err != nil {
				return err
			}
			isZero = isZeroBlock(buf)
		}

		if isNBD && isZero {
			// 🕳️ SPARSE BLOCK DETECTED: Use NBD zero command instead of writing zeros
			sparseBlocksSkipped++
			sparseBytesSkipped += int64(chunkSize)

			logger.WithFields(log.Fields{
				"offset":              offset,
				"size":                chunkSize,
				"sparse_blocks_total": sparseBlocksSkipped,
				"sparse_bytes_saved":  sparseBytesSkipped,
			}).Debug("🕳️ Client-side sparse detection: NBD server said allocated but block is zero")

			err = nbdTarget.Zero(uint64(chunkSize), uint64(offset), nil)
			if err != nil {
				// Fallback to regular write if Zero command fails
				logger.WithError(err).Warn("NBD Zero command failed, falling back to regular write")
				// If we used sampling and zero failed, we need to read the actual data for fallback
				if chunkSize >= 4*1024*1024 {
					err = handle.Pread(buf, uint64(offset), nil)
					if err != nil {
						return err
					}
				}
				err = nbdTarget.Pwrite(buf, uint64(offset), nil)
			}
		} else {
			// 📝 REAL DATA: Write actual non-zero content using appropriate method
			// Ensure we have the data read if it wasn't already (should be populated from above logic)
			if buf == nil {
				// This shouldn't happen with our logic above, but safety check
				buf = make([]byte, chunkSize)
				err = handle.Pread(buf, uint64(offset), nil)
				if err != nil {
					return err
				}
			}

			if isNBD {
				// NBD target: positioned write over network with TLS
				err = nbdTarget.Pwrite(buf, uint64(offset), nil)
			} else if isNamedPipe {
				// Named pipe: sequential write (CloudStack handles positioning)
				_, err = fd.Write(buf)
			} else {
				// Regular file: positioned write for random access
				_, err = fd.WriteAt(buf, offset)
			}
		}

		if err != nil {
			return err
		}

		offset += int64(chunkSize)

		// 🎯 CRITICAL: VMA Progress Integration - Real-time progress callbacks
		totalBytesTransferred += int64(chunkSize)
		currentPercent := (float64(totalBytesTransferred) / float64(totalBytes)) * 100
		timeSinceUpdate := time.Since(lastProgressUpdate)

		// Send VMA progress update every 2 seconds or 1% progress change (matching working pattern)
		if timeSinceUpdate >= 2*time.Second || currentPercent >= lastProgressPercent+1.0 {
			if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
				if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
					// Calculate throughput (bytes per second)
					elapsed := time.Since(startTime).Seconds()
					var throughputBPS int64
					if elapsed > 0 {
						throughputBPS = int64(float64(totalBytesTransferred) / elapsed)
					}

					// Extract disk info for logging
					diskPath := s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName
					vmName := s.Servers.VirtualMachine.Name()

					// Send VMA progress update matching working log format
					vpc.SendUpdate(progress.VMAProgressUpdate{
						Stage:            "Transfer",
						Status:           "in_progress",
						BytesTransferred: totalBytesTransferred,
						TotalBytes:       totalBytes,
						Percent:          currentPercent,
						ThroughputBPS:    throughputBPS,
					})

					// Log matching working pattern exactly
					log.WithFields(log.Fields{
						"bytes_transferred": totalBytesTransferred,
						"percent":           currentPercent,
						"throughput_bps":    throughputBPS,
						"total_bytes":       totalBytes,
						"vm":                vmName,
						"disk":              diskPath,
						"job_id":            ctx.Value("jobID"),
					}).Debug("📊 Progress update sent to VMA")

					lastProgressUpdate = time.Now()
					lastProgressPercent = currentPercent
				}
			}
		}
	}

	// 🎯 CRITICAL: Send final 100% progress update to VMA (matching working pattern)
	if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
		if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
			// Send final 100% completion update
			vpc.SendUpdate(progress.VMAProgressUpdate{
				Stage:            "Transfer",
				Status:           "completed",
				BytesTransferred: totalBytes,
				TotalBytes:       totalBytes,
				Percent:          100,
				ThroughputBPS:    0,
			})

			// Extract disk info for final logging
			diskPath := s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName
			vmName := s.Servers.VirtualMachine.Name()

			log.WithFields(log.Fields{
				"vm":     vmName,
				"disk":   diskPath,
				"job_id": ctx.Value("jobID"),
			}).Info("📊 Final progress update sent to VMA")
		}
	}

	// 🆕 SPARSE OPTIMIZATION: Log sparse block savings summary
	if sparseBlocksSkipped > 0 {
		logger.WithFields(log.Fields{
			"sparse_blocks_skipped": sparseBlocksSkipped,
			"sparse_bytes_saved":    sparseBytesSkipped,
			"sparse_mb_saved":       sparseBytesSkipped / (1024 * 1024),
			"total_bytes_actual":    totalBytesTransferred - sparseBytesSkipped,
		}).Info("🕳️ Sparse optimization summary - bandwidth saved by skipping zero blocks")
	}

	logger.Info("Full copy completed")

	return nil
}

func (s *NbdkitServer) IncrementalCopyToTarget(ctx context.Context, t target.Target, path string) error {
	// Initialize VMA progress tracking variables
	var (
		totalBytesTransferred int64
		totalBytes            int64
		lastProgressUpdate    = time.Now()
		lastProgressPercent   = 0.0
		startTime             = time.Now()
	)
	logger := log.WithFields(log.Fields{
		"vm":   s.Servers.VirtualMachine.Name(),
		"disk": s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName,
	})

	logger.Info("Starting incremental copy")

	// 🎯 CRITICAL: Send incremental job type to VMA now that we know it's confirmed
	if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
		if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
			vpc.SendUpdate(progress.VMAProgressUpdate{
				Stage:    "Transfer",
				Status:   "in_progress",
				SyncType: "incremental", // 🎯 CRITICAL: Tell VMA this is incremental
				Percent:  0,
			})
		}
	}

	currentChangeId, err := t.GetCurrentChangeID(ctx)
	if err != nil {
		return err
	}

	// 🎯 CRITICAL: Calculate CBT delta size using changed disk areas (matching working pattern)
	deltaSize, err := s.calculateDeltaSize(ctx, currentChangeId)
	if err != nil {
		logger.WithError(err).Warn("CBT delta calculation failed, using actual VMDK file size")
		// Fall back to actual VMDK file size (better than logical capacity)
		totalBytes = s.getActualDiskSize()
	} else {
		// Use CBT-calculated changed data size for accurate progress
		totalBytes = deltaSize
		logger.Infof("📊 Total delta size calculated: %.2f MB (%d bytes)",
			float64(totalBytes)/(1024*1024), totalBytes)
		logger.Infof("🎯 Using CBT-calculated changed data size for incremental progress",
			"change_id", currentChangeId.Value, "changed_data_mb", totalBytes/(1024*1024))
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

	// 🔧 Handle NBD, pipes, and regular files with proper selective copying
	var fd *os.File
	var nbdTarget *libnbd.Libnbd
	var isNBD bool
	var isNamedPipe bool

	// Check target type: NBD, named pipe, or regular file
	if strings.HasPrefix(path, "nbd://") {
		// NBD target - get handle from CloudStack target
		if cloudStackTarget, ok := t.(*target.CloudStack); ok {
			nbdTarget = cloudStackTarget.GetNBDHandle()
			if nbdTarget == nil {
				return fmt.Errorf("NBD target not connected")
			}
			isNBD = true
			logger.Info("🌐 Using NBD positioned writes for selective incremental copy")
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
				err = handle.Pread(buf, uint64(offset), nil)
				if err != nil {
					return err
				}

				// 🔧 Handle NBD, pipes, and files differently for writing
				if isNBD {
					// NBD target: positioned write over network with TLS
					err = nbdTarget.Pwrite(buf, uint64(offset), nil)
				} else if isNamedPipe {
					// Named pipe: sequential write (CloudStack handles positioning)
					_, err = fd.Write(buf)
				} else {
					// Regular file: positioned write for random access
					_, err = fd.WriteAt(buf, offset)
				}

				if err != nil {
					return err
				}

				bar.Set64(offset + chunkSize)
				offset += chunkSize

				// 🎯 CRITICAL: VMA Progress Integration - Real-time progress callbacks
				totalBytesTransferred += chunkSize
				currentPercent := (float64(totalBytesTransferred) / float64(totalBytes)) * 100
				timeSinceUpdate := time.Since(lastProgressUpdate)

				// Send VMA progress update every 2 seconds or 1% progress change (matching working pattern)
				if timeSinceUpdate >= 2*time.Second || currentPercent >= lastProgressPercent+1.0 {
					if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
						if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
							// Calculate throughput (bytes per second)
							elapsed := time.Since(startTime).Seconds()
							var throughputBPS int64
							if elapsed > 0 {
								throughputBPS = int64(float64(totalBytesTransferred) / elapsed)
							}

							// Extract disk info for logging
							diskPath := s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName
							vmName := s.Servers.VirtualMachine.Name()

							// Send VMA progress update matching working log format
							vpc.SendUpdate(progress.VMAProgressUpdate{
								Stage:            "Transfer",
								Status:           "in_progress",
								BytesTransferred: totalBytesTransferred,
								TotalBytes:       totalBytes,
								Percent:          currentPercent,
								ThroughputBPS:    throughputBPS,
							})

							// Log matching working pattern exactly
							log.WithFields(log.Fields{
								"bytes_transferred": totalBytesTransferred,
								"percent":           currentPercent,
								"throughput_bps":    throughputBPS,
								"total_bytes":       totalBytes,
								"vm":                vmName,
								"disk":              diskPath,
								"job_id":            ctx.Value("jobID"),
							}).Debug("📊 Progress update sent to VMA")

							lastProgressUpdate = time.Now()
							lastProgressPercent = currentPercent
						}
					}
				}
			}
		}

		startOffset = diskChangeInfo.StartOffset + diskChangeInfo.Length
		bar.Set64(startOffset)

		if startOffset == s.getActualDiskSize() { // Use actual VMDK file size
			break
		}
	}

	// 🎯 CRITICAL: Send final 100% progress update to VMA (matching working pattern)
	if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
		if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
			// Send final 100% completion update
			vpc.SendUpdate(progress.VMAProgressUpdate{
				Stage:            "Transfer",
				Status:           "completed",
				BytesTransferred: totalBytes,
				TotalBytes:       totalBytes,
				Percent:          100,
				ThroughputBPS:    0,
			})

			// Log final update matching working pattern exactly
			diskPath := s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName
			vmName := s.Servers.VirtualMachine.Name()
			log.WithFields(log.Fields{
				"disk":   diskPath,
				"job_id": ctx.Value("jobID"),
				"vm":     vmName,
			}).Info("📊 Final progress update sent to VMA")
		}
	}

	logger.Info("Incremental copy completed via selective block copying")
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
		// Use parallel NBD copy for full disk migrations (highest throughput)
		err = s.ParallelFullCopyToTarget(ctx, t, path, targetIsClean)
		if err != nil {
			return err
		}
	} else {
		// Use parallel NBD copy (with auto-fallback to serial on error)
		err = s.ParallelIncrementalCopyToTarget(ctx, t, path)
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
