// Package failover provides VirtIO driver injection for enhanced test failover
package failover

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/vexxhost/migratekit-sha/common"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
)

// VirtIOInjection handles VirtIO driver injection for KVM compatibility
type VirtIOInjection struct {
	db           *database.Connection
	volumeClient *common.VolumeClient
	jobTracker   *joblog.Tracker
}

// NewVirtIOInjection creates a new VirtIO injection handler
func NewVirtIOInjection(db *database.Connection, jobTracker *joblog.Tracker) *VirtIOInjection {
	return &VirtIOInjection{
		db:           db,
		volumeClient: common.NewVolumeClient("http://localhost:8090"),
		jobTracker:   jobTracker,
	}
}

// InjectVirtIODrivers injects VirtIO drivers for KVM compatibility using real implementation
func (vi *VirtIOInjection) InjectVirtIODrivers(ctx context.Context, request *EnhancedTestFailoverRequest, snapshotName string) (string, error) {
	logger := vi.jobTracker.Logger(ctx)

	// Get device path for VirtIO injection (need real device, not volume UUID)
	volumeUUID, err := vi.getVolumeUUIDForVM(ctx, request.VMID)
	if err != nil {
		return "", fmt.Errorf("failed to get volume UUID for VM %s: %w", request.VMID, err)
	}

	// Get device path from Volume Daemon
	deviceInfo, err := vi.volumeClient.GetVolumeDevice(context.Background(), volumeUUID)
	if err != nil {
		return "", fmt.Errorf("failed to get device path for volume %s: %w", volumeUUID, err)
	}

	devicePath := deviceInfo.DevicePath
	logger.Info("🔍 Found device for VirtIO injection",
		"device_path", devicePath,
		"vm_id", request.VMID,
	)

	// Generate job ID for injection script logging
	injectionJobID := fmt.Sprintf("virtio-%s-%d", request.VMID, time.Now().Unix())

	// Call VirtIO injection script
	injectionScript := "/opt/migratekit/bin/inject-virtio-drivers.sh"
	logger.Info("🚀 Executing VirtIO driver injection script",
		"script", injectionScript,
		"device_path", devicePath,
		"injection_job_id", injectionJobID,
	)

	cmd := exec.CommandContext(ctx, "sudo", injectionScript, devicePath, injectionJobID)

	output, err := cmd.CombinedOutput()
	if err != nil {
		errorMsg := fmt.Sprintf("VirtIO injection failed: %s", string(output))
		logger.Error("❌ VirtIO driver injection failed",
			"error", err.Error(),
			"injection_output", string(output),
			"device_path", devicePath,
			"injection_job_id", injectionJobID,
		)

		// Attempt to rollback to snapshot if injection failed
		if snapshotName != "" {
			logger.Warn("🔄 Attempting rollback to snapshot due to VirtIO injection failure",
				"snapshot_name", snapshotName,
				"device_path", devicePath,
			)

			rollbackErr := vi.performCloudStackVolumeRollback(ctx, request.VMID, snapshotName)
			if rollbackErr != nil {
				logger.Error("❌ CloudStack snapshot rollback failed - manual intervention required",
					"error", rollbackErr.Error(),
					"snapshot_id", snapshotName,
				)
				errorMsg = fmt.Sprintf("%s; Rollback failed: %s", errorMsg, rollbackErr.Error())
			} else {
				logger.Info("✅ Successfully rolled back to CloudStack snapshot", "snapshot_id", snapshotName)
				errorMsg = fmt.Sprintf("%s; Volume rolled back to snapshot %s", errorMsg, snapshotName)
			}
		} else {
			logger.Warn("⚠️ No snapshot available for rollback")
		}

		return "", fmt.Errorf("failed to inject VirtIO drivers: %w", err)
	}

	status := "drivers-injected-successfully"

	logger.Info("✅ VirtIO injection step: Successfully injected drivers for KVM compatibility",
		"virtio_status", status,
		"device_path", devicePath,
		"injection_output", string(output),
		"injection_job_id", injectionJobID,
	)

	// Verify injection log file was created
	logFile := fmt.Sprintf("/var/log/migratekit/virtv2v-%s.log", injectionJobID)
	if _, err := exec.CommandContext(ctx, "test", "-f", logFile).CombinedOutput(); err == nil {
		logger.Info("📄 VirtIO injection log file created", "log_file", logFile)
	} else {
		logger.Warn("⚠️ VirtIO injection log file not found", "log_file", logFile)
	}

	return status, nil
}

// ExecuteVirtIOInjectionStep injects VirtIO drivers for test VM using JobLog.RunStep pattern
func (vi *VirtIOInjection) ExecuteVirtIOInjectionStep(
	ctx context.Context,
	parentJobID string,
	request *EnhancedTestFailoverRequest,
	snapshotName string,
) (string, error) {
	logger := vi.jobTracker.Logger(ctx)
	logger.Info("🖥️ Starting VirtIO injection step for KVM compatibility",
		"vm_id", request.VMID,
		"vm_name", request.VMName,
		"parent_job_id", parentJobID,
		"purpose", "test-vm-kvm-compatibility",
	)

	// Use JobLog.RunStep for proper step tracking (replaces GenericJobTrackingService)
	var injectionStatus string
	err := vi.jobTracker.RunStep(ctx, parentJobID, "virtio-driver-injection", func(ctx context.Context) error {
		var err error
		injectionStatus, err = vi.InjectVirtIODrivers(ctx, request, snapshotName)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("VirtIO driver injection failed: %w", err)
	}

	logger.Info("✅ VirtIO injection step completed successfully",
		"vm_id", request.VMID,
		"injection_status", injectionStatus,
	)

	return injectionStatus, nil
}

// getVolumeUUIDForVM retrieves the CloudStack volume UUID for a VM
func (vi *VirtIOInjection) getVolumeUUIDForVM(ctx context.Context, vmID string) (string, error) {
	logger := vi.jobTracker.Logger(ctx)
	logger.Info("🔍 Querying database for volume UUID for VirtIO injection", "source_vm_id", vmID)

	// Step 1: Find replication job for this source VM
	var replicationJob database.ReplicationJob
	err := (*vi.db).GetGormDB().Where("source_vm_id = ?", vmID).
		Order("created_at DESC").
		First(&replicationJob).Error
	if err != nil {
		logger.Error("Failed to find replication job", "error", err, "source_vm_id", vmID)
		return "", fmt.Errorf("no replication job found for VM %s: %w", vmID, err)
	}

	// Step 2: Find VM disks for this job
	var vmDisks []database.VMDisk
	err = (*vi.db).GetGormDB().Where("job_id = ?", replicationJob.ID).Find(&vmDisks).Error
	if err != nil {
		logger.Error("Failed to find VM disks", "error", err, "job_id", replicationJob.ID)
		return "", fmt.Errorf("no VM disks found for job %s: %w", replicationJob.ID, err)
	}

	if len(vmDisks) == 0 {
		logger.Error("No VM disks found for replication job", "job_id", replicationJob.ID)
		return "", fmt.Errorf("no VM disks found for job %s", replicationJob.ID)
	}

	// 🎯 CRITICAL FIX: Find the OS disk (disk-2000) specifically for VirtIO injection
	// VirtIO drivers must only be injected into the Windows OS disk, not data disks
	var osDisk *database.VMDisk
	for _, disk := range vmDisks {
		if disk.DiskID == "disk-2000" {
			osDisk = &disk
			logger.Info("🎯 Found OS disk for VirtIO injection",
				"disk_id", disk.DiskID,
				"ossea_volume_id", disk.OSSEAVolumeID,
				"vmware_uuid", disk.VMwareUUID)
			break
		}
	}

	if osDisk == nil {
		// Log all available disks for debugging
		logger.Error("❌ OS disk (disk-2000) not found for VirtIO injection",
			"job_id", replicationJob.ID,
			"available_disks", len(vmDisks))
		for i, disk := range vmDisks {
			logger.Info("Available disk",
				"index", i,
				"disk_id", disk.DiskID,
				"vmware_uuid", disk.VMwareUUID)
		}
		return "", fmt.Errorf("OS disk (disk-2000) not found for VM %s - VirtIO injection requires Windows OS disk", vmID)
	}

	vmDisk := *osDisk

	// 🪟 WINDOWS DETECTION: Only inject VirtIO drivers for Windows VMs
	// 🆕 ENHANCED: More permissive OS type handling for VirtIO injection
	if vmDisk.OSType == "" || vmDisk.OSType == "unknown" {
		logger.Warn("⚠️ OS type not detected or unknown - proceeding with VirtIO injection (assuming Windows)",
			"os_type", vmDisk.OSType,
			"disk_id", vmDisk.DiskID,
			"vm_id", vmID)
	} else if vmDisk.OSType == "windows" {
		logger.Info("🪟 Windows VM detected - proceeding with VirtIO injection",
			"os_type", vmDisk.OSType,
			"disk_id", vmDisk.DiskID,
			"vm_id", vmID)
	} else if vmDisk.OSType == "linux" || vmDisk.OSType == "other" {
		logger.Info("⏭️ Skipping VirtIO injection for non-Windows VM",
			"os_type", vmDisk.OSType,
			"disk_id", vmDisk.DiskID,
			"vm_id", vmID)
		return "", fmt.Errorf("VirtIO injection skipped - VM is not Windows (os_type: %s)", vmDisk.OSType)
	} else {
		// Unknown OS type - proceed with caution
		logger.Warn("⚠️ Unrecognized OS type - proceeding with VirtIO injection (manual verification recommended)",
			"os_type", vmDisk.OSType,
			"disk_id", vmDisk.DiskID,
			"vm_id", vmID)
	}

	// Step 3: Find OSSEA volume
	var osseaVolume database.OSSEAVolume
	err = (*vi.db).GetGormDB().Where("id = ?", vmDisk.OSSEAVolumeID).First(&osseaVolume).Error
	if err != nil {
		logger.Error("Failed to find OSSEA volume", "error", err, "ossea_volume_id", vmDisk.OSSEAVolumeID)
		return "", fmt.Errorf("no OSSEA volume found for ID %d: %w", vmDisk.OSSEAVolumeID, err)
	}

	logger.Info("Found volume UUID for VirtIO injection",
		"source_vm_id", vmID,
		"volume_uuid", osseaVolume.VolumeID,
		"volume_name", osseaVolume.VolumeName)

	return osseaVolume.VolumeID, nil
}

// performCloudStackVolumeRollback rolls back a volume to a CloudStack volume snapshot
func (vi *VirtIOInjection) performCloudStackVolumeRollback(ctx context.Context, vmID, snapshotID string) error {
	// This method is not currently used in the workflow
	// VirtIO injection uses CloudStack snapshots for rollback protection, not rollback itself
	// If rollback is needed in the future, this would call snapshot revert operations
	return fmt.Errorf("volume rollback feature disabled - VirtIO injection uses snapshots for protection only")
}
