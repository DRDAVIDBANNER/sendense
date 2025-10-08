// Package services provides simple failed execution cleanup for stuck failover operations
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-sha/common"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/ossea"
)

// FailedExecutionCleanupService handles cleanup of stuck failover operations
type FailedExecutionCleanupService struct {
	db           *database.Connection
	volumeClient *common.VolumeClient
	osseaClient  *ossea.Client
	jobTracker   *joblog.Tracker
}

// NewFailedExecutionCleanupService creates a new cleanup service with pre-initialized OSSEA client
func NewFailedExecutionCleanupService(db *database.Connection, jobTracker *joblog.Tracker, osseaClient *ossea.Client) *FailedExecutionCleanupService {
	volumeClient := common.NewVolumeClient("http://localhost:8090")

	return &FailedExecutionCleanupService{
		db:           db,
		volumeClient: volumeClient,
		osseaClient:  osseaClient,
		jobTracker:   jobTracker,
	}
}

// CleanupFailedExecution performs simple cleanup for a stuck failover operation
func (fecs *FailedExecutionCleanupService) CleanupFailedExecution(ctx context.Context, vmName string) error {
	// Start cleanup job
	ctx, jobID, err := fecs.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "cleanup",
		Operation: "failed-execution-cleanup",
		Owner:     &[]string{"system"}[0],
	})
	if err != nil {
		return fmt.Errorf("failed to start cleanup job: %w", err)
	}

	logger := fecs.jobTracker.Logger(ctx)
	logger.Info("🧹 Starting simple failed execution cleanup", "vm_name", vmName)

	// Get VM context
	var vmContext database.VMReplicationContext
	err = (*fecs.db).GetGormDB().Where("vm_name = ?", vmName).First(&vmContext).Error
	if err != nil {
		fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return fmt.Errorf("VM context not found: %w", err)
	}

	// Get volumes for this VM
	var volumes []database.OSSEAVolume
	err = (*fecs.db).GetGormDB().Where("vm_context_id = ?", vmContext.ContextID).Find(&volumes).Error
	if err != nil {
		fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return fmt.Errorf("failed to get volumes: %w", err)
	}

	// Phase 1: Intelligent volume detachment based on current state
	var attachedVolumes []database.OSSEAVolume
	var detachedVolumes []database.OSSEAVolume

	err = fecs.jobTracker.RunStep(ctx, jobID, "analyze-volume-state", func(ctx context.Context) error {
		logger := fecs.jobTracker.Logger(ctx)
		logger.Info("🔍 Analyzing volume attachment state", "vm_name", vmName)

		// Categorize volumes by attachment status using Volume Daemon
		for _, volume := range volumes {
			// Check volume attachment status via Volume Daemon
			deviceInfo, err := fecs.volumeClient.GetVolumeDevice(ctx, volume.VolumeID)
			if err != nil || deviceInfo == nil {
				// No device info = volume is detached
				logger.Info("📋 Volume detached (no device info)", "volume_id", volume.VolumeID)
				detachedVolumes = append(detachedVolumes, volume)
			} else {
				// Has device info = volume is attached
				logger.Info("📋 Volume attached (has device info)", "volume_id", volume.VolumeID, "device_path", deviceInfo.DevicePath)
				attachedVolumes = append(attachedVolumes, volume)
			}
		}

		logger.Info("📊 Volume state analysis complete",
			"attached_volumes", len(attachedVolumes),
			"detached_volumes", len(detachedVolumes))

		return nil
	})
	if err != nil {
		fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return fmt.Errorf("volume state analysis failed: %w", err)
	}

	// Phase 2: Detach attached volumes only
	if len(attachedVolumes) > 0 {
		err = fecs.jobTracker.RunStep(ctx, jobID, "detach-attached-volumes", func(ctx context.Context) error {
			logger := fecs.jobTracker.Logger(ctx)
			logger.Info("🔌 Detaching attached volumes", "count", len(attachedVolumes))

			for _, volume := range attachedVolumes {
				logger.Info("🔌 Detaching volume from SHA", "volume_id", volume.VolumeID)

				operation, detachErr := fecs.volumeClient.DetachVolume(ctx, volume.VolumeID)
				if detachErr != nil {
					return fmt.Errorf("failed to start volume detachment for %s: %w", volume.VolumeID, detachErr)
				}

				// Wait for detach operation to complete
				logger.Info("⏳ Waiting for volume detachment to complete", "volume_id", volume.VolumeID, "operation_id", operation.ID)
				_, err := fecs.volumeClient.WaitForCompletionWithTimeout(ctx, operation.ID, 60*time.Second)
				if err != nil {
					return fmt.Errorf("volume detachment completion failed for %s: %w", volume.VolumeID, err)
				}

				logger.Info("✅ Volume detached successfully", "volume_id", volume.VolumeID)
			}
			return nil
		})
		if err != nil {
			fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
			return fmt.Errorf("volume detachment failed: %w", err)
		}
	} else {
		logger.Info("ℹ️ All volumes already detached, skipping detach phase")
	}

	// Phase 3: Proper snapshot cleanup (revert → delete → clear database)
	// CRITICAL: All volumes are now detached (either were already or just detached)
	err = fecs.jobTracker.RunStep(ctx, jobID, "cleanup-snapshots", func(ctx context.Context) error {
		logger := fecs.jobTracker.Logger(ctx)
		logger.Info("📸 Starting multi-volume snapshot cleanup", "vm_context_id", vmContext.ContextID)

		// Fail fast if OSSEA client not available
		if fecs.osseaClient == nil {
			return fmt.Errorf("OSSEA client not available - cannot perform snapshot operations")
		}

		// Use direct CloudStack snapshot operations (same logic as multi-volume service)
		err := fecs.cleanupSnapshotsDirectly(ctx, vmContext.ContextID)
		if err != nil {
			return fmt.Errorf("multi-volume snapshot cleanup failed: %w", err)
		}

		logger.Info("✅ Multi-volume snapshot cleanup completed", "vm_name", vmName)
		return nil
	})
	if err != nil {
		fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return fmt.Errorf("snapshot cleanup failed: %w", err)
	}

	// Phase 4: Reattach ALL volumes to SHA (both originally attached and detached)
	err = fecs.jobTracker.RunStep(ctx, jobID, "reattach-all-volumes", func(ctx context.Context) error {
		logger := fecs.jobTracker.Logger(ctx)
		
		// Get SHA VM ID from database (following Volume Daemon v2.1.2 pattern)
		shaVMID, err := fecs.getOMAVMIDFromDatabase(ctx)
		if err != nil {
			return fmt.Errorf("failed to get SHA VM ID from database: %w", err)
		}
		logger.Info("Retrieved SHA VM ID from database for volume reattachment", "oma_vm_id", shaVMID)

		for _, volume := range volumes {
			logger.Info("🔗 Reattaching volume to SHA", "volume_id", volume.VolumeID, "oma_vm_id", shaVMID)

			// Send attach request and get operation ID
			operation, attachErr := fecs.volumeClient.AttachVolume(ctx, volume.VolumeID, shaVMID)
			if attachErr != nil {
				return fmt.Errorf("failed to start volume attachment for %s: %w", volume.VolumeID, attachErr)
			}

			// Wait for attach operation to complete
			logger.Info("⏳ Waiting for volume attachment to complete", "volume_id", volume.VolumeID, "operation_id", operation.ID)
			_, err := fecs.volumeClient.WaitForCompletionWithTimeout(ctx, operation.ID, 60*time.Second)
			if err != nil {
				return fmt.Errorf("volume attachment completion failed for %s: %w", volume.VolumeID, err)
			}

			logger.Info("✅ Volume attached successfully", "volume_id", volume.VolumeID)
		}
		return nil
	})
	if err != nil {
		fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return fmt.Errorf("volume reattachment failed: %w", err)
	}

	// Phase 5: Reset VM state and database cleanup
	err = fecs.jobTracker.RunStep(ctx, jobID, "reset-vm-state", func(ctx context.Context) error {
		logger := fecs.jobTracker.Logger(ctx)

		// Mark failover job as failed
		(*fecs.db).GetGormDB().Model(&database.FailoverJob{}).
			Where("vm_context_id = ? AND status IN ('pending', 'running')", vmContext.ContextID).
			Updates(map[string]interface{}{
				"status":     "failed",
				"updated_at": time.Now(),
			})

		// Reset VM context to ready_for_failover
		err := (*fecs.db).GetGormDB().Model(&database.VMReplicationContext{}).
			Where("context_id = ?", vmContext.ContextID).
			Updates(map[string]interface{}{
				"current_status": "ready_for_failover",
				"current_job_id": nil,
				"updated_at":     time.Now(),
			}).Error
		if err != nil {
			return fmt.Errorf("failed to reset VM context: %w", err)
		}

		logger.Info("📋 Reset VM state to ready_for_failover", "vm_name", vmName)
		return nil
	})
	if err != nil {
		fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return fmt.Errorf("VM state reset failed: %w", err)
	}

	fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	logger.Info("✅ Failed execution cleanup completed successfully", "vm_name", vmName)
	return nil
}

// VolumeSnapshotInfo represents snapshot information for cleanup (same as multi-volume service)
type VolumeSnapshotInfo struct {
	VolumeID   string `json:"volume_id"`
	SnapshotID string `json:"snapshot_id"`
	VolumeName string `json:"volume_name"`
}

// cleanupSnapshotsDirectly performs direct CloudStack snapshot cleanup (exact same logic as multi-volume service)
func (fecs *FailedExecutionCleanupService) cleanupSnapshotsDirectly(ctx context.Context, vmContextID string) error {
	logger := fecs.jobTracker.Logger(ctx)
	logger.Info("🧹 Cleaning up ALL volume snapshots", "vm_context_id", vmContextID)

	// Step 1: Get snapshots using RAW SQL (same as multi-volume service)
	snapshots, err := fecs.getVMSnapshotsFromDatabase(ctx, vmContextID)
	if err != nil {
		return fmt.Errorf("failed to get VM snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		logger.Info("No snapshots found for cleanup", "vm_context_id", vmContextID)
		return nil
	}

	logger.Info("🔍 Found snapshots for cleanup", "vm_context_id", vmContextID, "snapshot_count", len(snapshots))

	// Step 2: Revert each volume to snapshot (CRITICAL: Undo test failover changes)
	revertedCount := 0
	for _, snapshot := range snapshots {
		if snapshot.SnapshotID == "" {
			continue
		}

		logger.Info("⏪ Reverting volume to snapshot", "volume_id", snapshot.VolumeID, "snapshot_id", snapshot.SnapshotID)

		if fecs.osseaClient != nil {
			err := fecs.osseaClient.RevertVolumeSnapshot(snapshot.SnapshotID)
			if err != nil {
				logger.Error("❌ Volume revert failed", "volume_id", snapshot.VolumeID, "snapshot_id", snapshot.SnapshotID, "error", err)
				return fmt.Errorf("failed to revert volume %s to snapshot: %w", snapshot.VolumeID, err)
			}
			logger.Info("✅ Volume reverted to snapshot", "volume_id", snapshot.VolumeID)
			revertedCount++
		}
	}

	// Step 3: Delete each CloudStack snapshot after revert
	deletedCount := 0
	for _, snapshot := range snapshots {
		if snapshot.SnapshotID == "" {
			continue
		}

		logger.Info("🗑️ Deleting CloudStack snapshot", "volume_id", snapshot.VolumeID, "snapshot_id", snapshot.SnapshotID)

		if fecs.osseaClient != nil {
			err := fecs.osseaClient.DeleteVolumeSnapshot(snapshot.SnapshotID)
			if err != nil {
				logger.Error("❌ Snapshot deletion failed", "snapshot_id", snapshot.SnapshotID, "error", err)
				return fmt.Errorf("failed to delete snapshot %s: %w", snapshot.SnapshotID, err)
			}
			logger.Info("✅ CloudStack snapshot deleted", "snapshot_id", snapshot.SnapshotID)
			deletedCount++
		}
	}

	// Step 4: Clear database tracking (same as multi-volume service)
	if deletedCount > 0 {
		updateResult := (*fecs.db).GetGormDB().Model(&database.OSSEAVolume{}).
			Where("vm_context_id = ?", vmContextID).
			Updates(map[string]interface{}{
				"snapshot_id":         "",
				"snapshot_created_at": nil,
				"snapshot_status":     "none",
			})

		if updateResult.Error != nil {
			return fmt.Errorf("failed to clear snapshot database tracking: %w", updateResult.Error)
		}

		logger.Info("✅ Snapshot tracking data cleared from ossea_volumes successfully",
			"vm_context_id", vmContextID,
			"database_records_updated", updateResult.RowsAffected)
	}

	logger.Info("🎉 Multi-volume snapshot cleanup completed",
		"vm_context_id", vmContextID,
		"snapshots_reverted", revertedCount,
		"snapshots_deleted", deletedCount)

	return nil
}

// getVMSnapshotsFromDatabase retrieves snapshot info directly from ossea_volumes table (same as multi-volume service)
func (fecs *FailedExecutionCleanupService) getVMSnapshotsFromDatabase(ctx context.Context, vmContextID string) ([]VolumeSnapshotInfo, error) {
	logger := fecs.jobTracker.Logger(ctx)
	logger.Info("🔍 Getting VM snapshots from ossea_volumes table (stable storage)", "vm_context_id", vmContextID)

	// Query ossea_volumes for snapshot information (same query as multi-volume service)
	query := `
		SELECT ov.volume_id, ov.volume_name, ov.snapshot_id
		FROM ossea_volumes ov
		WHERE ov.vm_context_id = ? AND ov.snapshot_id IS NOT NULL AND ov.snapshot_id != ''
	`

	rows, err := (*fecs.db).GetGormDB().Raw(query, vmContextID).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to query ossea_volumes: %w", err)
	}
	defer rows.Close()

	var snapshots []VolumeSnapshotInfo
	for rows.Next() {
		var snapshot VolumeSnapshotInfo
		err := rows.Scan(&snapshot.VolumeID, &snapshot.VolumeName, &snapshot.SnapshotID)
		if err != nil {
			logger.Error("Failed to scan snapshot row", "error", err)
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	logger.Info("📊 Retrieved snapshots from database",
		"vm_context_id", vmContextID,
		"snapshot_count", len(snapshots))

	return snapshots, nil
}

// getOMAVMIDFromDatabase retrieves the SHA VM ID from the active ossea_configs record
// Following Volume Daemon v2.1.2 pattern for dynamic SHA VM ID lookup
func (fecs *FailedExecutionCleanupService) getOMAVMIDFromDatabase(ctx context.Context) (string, error) {
	logger := fecs.jobTracker.Logger(ctx)
	
	var shaVMID string
	err := (*fecs.db).GetGormDB().Raw("SELECT oma_vm_id FROM ossea_configs WHERE is_active = 1 LIMIT 1").Scan(&shaVMID).Error
	if err != nil {
		logger.Error("Failed to query SHA VM ID from database", "error", err)
		return "", fmt.Errorf("failed to query SHA VM ID from ossea_configs: %w", err)
	}
	
	if shaVMID == "" {
		logger.Error("SHA VM ID is empty in database")
		return "", fmt.Errorf("SHA VM ID is empty in ossea_configs table")
	}
	
	logger.Debug("Successfully retrieved SHA VM ID from database", "oma_vm_id", shaVMID)
	return shaVMID, nil
}
