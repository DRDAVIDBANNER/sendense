// Package workflows - Volume mounting and management for migration workflows
package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/common"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/volume"
)

// mountVolumesToOMA attaches and mounts volumes to the SHA appliance
func (m *MigrationEngine) mountVolumesToOMA(ctx context.Context, req *MigrationRequest) ([]VolumeMountResult, error) {
	log.WithField("job_id", req.JobID).Info("Mounting volumes to SHA appliance")

	// Get OSSEA configuration
	osseaConfig, err := m.osseaConfigRepo.GetByID(req.OSSEAConfigID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OSSEA configuration: %w", err)
	}

	// Note: Volume operations now use Volume Daemon HTTP API instead of direct CloudStack calls

	// Get VM disks for this job
	vmDisks, err := m.vmDiskRepo.GetByJobID(req.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM disks: %w", err)
	}

	var results []VolumeMountResult

	for _, vmDisk := range vmDisks {
		if vmDisk.OSSEAVolumeID == 0 {
			log.WithField("disk_id", vmDisk.DiskID).Warn("VM disk has no OSSEA volume, skipping mount")
			continue
		}

		// Get OSSEA volume record
		osseaVolume, err := m.osseaVolumeRepo.GetByID(vmDisk.OSSEAVolumeID)
		if err != nil {
			return results, fmt.Errorf("failed to get OSSEA volume for disk %s: %w", vmDisk.DiskID, err)
		}

		result := VolumeMountResult{
			OSSEAVolumeID: osseaVolume.VolumeID, // Use CloudStack volume UUID, not database row ID
			Status:        "attaching",
		}

		// Step 1: Attach volume to SHA VM via Volume Daemon
		volumeClient := common.NewVolumeClient("http://localhost:8090")
		operation, err := volumeClient.AttachVolume(ctx, osseaVolume.VolumeID, osseaConfig.SHAVMID)
		if err != nil {
			result.Status = "attach_failed"
			result.ErrorMessage = err.Error()
			results = append(results, result)
			return results, fmt.Errorf("failed to attach volume %s to SHA: %w", osseaVolume.VolumeID, err)
		}

		// Wait for volume attachment to complete
		_, err = volumeClient.WaitForCompletionWithTimeout(ctx, operation.ID, 2*time.Minute)
		if err != nil {
			result.Status = "attach_failed"
			result.ErrorMessage = err.Error()
			results = append(results, result)
			return results, fmt.Errorf("volume attachment operation failed for %s: %w", osseaVolume.VolumeID, err)
		}

		log.WithFields(log.Fields{
			"job_id":    req.JobID,
			"volume_id": osseaVolume.VolumeID,
			"oma_vm_id": osseaConfig.SHAVMID,
		}).Info("Volume attached to SHA VM")

		// Step 2: Wait for attachment and detect device path
		devicePath, err := m.waitForDeviceAttachment(osseaVolume.VolumeID, vmDisk.UnitNumber)
		if err != nil {
			result.Status = "device_detection_failed"
			result.ErrorMessage = err.Error()
			results = append(results, result)
			return results, fmt.Errorf("failed to detect device path for volume %s: %w", osseaVolume.VolumeID, err)
		}

		result.DevicePath = devicePath

		// Step 3: Mount the volume
		mountOpts := &volume.MountOptions{
			JobID:          req.JobID,
			VolumeID:       osseaVolume.VolumeID,
			DevicePath:     devicePath,
			FilesystemType: "auto", // Auto-detect filesystem
			ReadOnly:       false,
			MountOptions:   []string{"rw", "noatime"},
		}

		mountResult, err := m.mountManager.MountVolume(mountOpts)
		if err != nil {
			result.Status = "mount_failed"
			result.ErrorMessage = err.Error()
			results = append(results, result)
			return results, fmt.Errorf("failed to mount volume %s: %w", osseaVolume.VolumeID, err)
		}

		result.MountPoint = mountResult.MountPoint
		result.Status = "mounted"

		// Step 4: Create volume mount record
		volumeMount := &database.VolumeMount{
			OSSEAVolumeID:  osseaVolume.ID,
			JobID:          req.JobID,
			DevicePath:     devicePath,
			MountPoint:     mountResult.MountPoint,
			MountStatus:    "mounted",
			FilesystemType: mountResult.FilesystemType,
			MountOptions:   mountResult.MountOptions,
			IsReadOnly:     false,
			MountedAt:      &[]time.Time{time.Now()}[0],
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := m.volumeMountRepo.Create(volumeMount); err != nil {
			log.WithError(err).Warn("Failed to create volume mount record")
		}

		// Step 5: Update OSSEA volume with mount info
		if err := m.osseaVolumeRepo.UpdateMountInfo(osseaVolume.ID, devicePath, mountResult.MountPoint); err != nil {
			log.WithError(err).Warn("Failed to update OSSEA volume mount info")
		}

		results = append(results, result)

		log.WithFields(log.Fields{
			"job_id":      req.JobID,
			"volume_id":   osseaVolume.VolumeID,
			"device_path": devicePath,
			"mount_point": mountResult.MountPoint,
		}).Info("Volume mounted successfully")
	}

	return results, nil
}

// waitForDeviceAttachment waits for the attached volume to appear as a block device
func (m *MigrationEngine) waitForDeviceAttachment(volumeID string, unitNumber int) (string, error) {
	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"unit_number": unitNumber,
	}).Info("Detecting device attachment")

	// Wait a moment for the attachment to complete
	time.Sleep(2 * time.Second)

	// Get current block devices and find the highest available device
	// Since CloudStack attaches volumes sequentially, the latest device should be ours
	currentDevices, err := m.getBlockDevices()
	if err != nil {
		return "", fmt.Errorf("failed to get block device list: %w", err)
	}

	if len(currentDevices) == 0 {
		return "", fmt.Errorf("no block devices found on system")
	}

	// Get the last (highest) device - this should be the newly attached one
	latestDevice := currentDevices[len(currentDevices)-1]

	log.WithFields(log.Fields{
		"volume_id":       volumeID,
		"device_path":     latestDevice,
		"total_devices":   len(currentDevices),
		"detected_method": "latest_device",
	}).Info("Device attachment detected")

	return latestDevice, nil
}

// getBlockDevices returns a list of current block devices
func (m *MigrationEngine) getBlockDevices() ([]string, error) {
	devices := []string{}

	// Scan /dev for vd* devices
	files, err := filepath.Glob("/dev/vd*")
	if err != nil {
		return nil, fmt.Errorf("failed to scan /dev for block devices: %w", err)
	}

	for _, file := range files {
		// Check if it's a block device
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// Check if it's a block device (not a partition)
		if info.Mode()&os.ModeDevice != 0 && info.Mode()&os.ModeCharDevice == 0 {
			// Additional check: ensure it's not a partition (ends with digit)
			if matched, _ := regexp.MatchString(`/dev/vd[a-z]$`, file); matched {
				devices = append(devices, file)
			}
		}
	}

	sort.Strings(devices)
	return devices, nil
}

// CleanupVolumes removes volumes and mounts for a failed or cancelled migration
func (m *MigrationEngine) CleanupVolumes(ctx context.Context, jobID string) error {
	log.WithField("job_id", jobID).Info("Starting volume cleanup for migration")

	// Get all volume mounts for this job
	mounts, err := m.volumeMountRepo.GetByJobID(jobID)
	if err != nil {
		log.WithError(err).Error("Failed to get volume mounts for cleanup")
		return fmt.Errorf("failed to get volume mounts: %w", err)
	}

	// Unmount all volumes
	for _, mount := range mounts {
		if mount.MountStatus == "mounted" {
			log.WithFields(log.Fields{
				"job_id":      jobID,
				"mount_point": mount.MountPoint,
				"device_path": mount.DevicePath,
			}).Info("Unmounting volume")

			// TODO: Implement unmount logic in volume.MountManager
			// if err := m.mountManager.UnmountVolume(mount.MountPoint); err != nil {
			//     log.WithError(err).Warn("Failed to unmount volume")
			// }

			// Update mount status
			if err := m.volumeMountRepo.UpdateMountStatus(mount.ID, "unmounted", false); err != nil {
				log.WithError(err).Warn("Failed to update mount status")
			}
		}
	}

	// Get VM disks to find OSSEA volumes
	vmDisks, err := m.vmDiskRepo.GetByJobID(jobID)
	if err != nil {
		log.WithError(err).Error("Failed to get VM disks for cleanup")
		return fmt.Errorf("failed to get VM disks: %w", err)
	}

	// Get OSSEA configuration for the job
	var osseaConfigID int
	if len(vmDisks) > 0 {
		// Get replication job to find OSSEA config
		var job database.ReplicationJob
		if err := m.db.GetGormDB().Where("id = ?", jobID).First(&job).Error; err != nil {
			log.WithError(err).Warn("Failed to get replication job for cleanup")
			return nil // Non-fatal, continue cleanup
		}
		osseaConfigID = job.OSSEAConfigID
	}

	if osseaConfigID > 0 {
		// Note: OSSEA configuration no longer needed for volume operations (using Volume Daemon)

		// Note: Volume operations now use Volume Daemon HTTP API instead of direct CloudStack calls

		// Detach and delete volumes
		for _, vmDisk := range vmDisks {
			if vmDisk.OSSEAVolumeID > 0 {
				osseaVolume, err := m.osseaVolumeRepo.GetByID(vmDisk.OSSEAVolumeID)
				if err != nil {
					log.WithError(err).Warn("Failed to get OSSEA volume for cleanup")
					continue
				}

				log.WithFields(log.Fields{
					"job_id":    jobID,
					"volume_id": osseaVolume.VolumeID,
				}).Info("Detaching and deleting OSSEA volume")

				// Detach volume from SHA VM via Volume Daemon
				volumeClient := common.NewVolumeClient("http://localhost:8090")
				operation, err := volumeClient.DetachVolume(context.Background(), osseaVolume.VolumeID)
				if err != nil {
					log.WithError(err).Warn("Failed to detach volume")
				} else {
					// Wait for detachment to complete
					_, err = volumeClient.WaitForCompletionWithTimeout(context.Background(), operation.ID, 2*time.Minute)
					if err != nil {
						log.WithError(err).Warn("Volume detachment operation failed")
					}
				}

				// Delete volume via Volume Daemon
				operation, err = volumeClient.DeleteVolume(context.Background(), osseaVolume.VolumeID)
				if err != nil {
					log.WithError(err).Warn("Failed to delete volume")
				} else {
					// Wait for deletion to complete
					_, err = volumeClient.WaitForCompletionWithTimeout(context.Background(), operation.ID, 2*time.Minute)
					if err != nil {
						log.WithError(err).Warn("Volume deletion operation failed")
					}
				}

				// Update volume status
				if err := m.osseaVolumeRepo.UpdateStatus(osseaVolume.ID, "deleted"); err != nil {
					log.WithError(err).Warn("Failed to update volume status")
				}
			}
		}
	}

	log.WithField("job_id", jobID).Info("Volume cleanup completed")
	return nil
}
