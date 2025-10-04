// Package services provides core failed execution cleanup for stuck failover operations
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-oma/common"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
)

// CoreFailedExecutionCleanupService handles core cleanup of stuck failover operations
type CoreFailedExecutionCleanupService struct {
	db           *database.Connection
	volumeClient *common.VolumeClient
	jobTracker   *joblog.Tracker
}

// NewCoreFailedExecutionCleanupService creates a new core cleanup service
func NewCoreFailedExecutionCleanupService(db *database.Connection, jobTracker *joblog.Tracker) *CoreFailedExecutionCleanupService {
	volumeClient := common.NewVolumeClient("http://localhost:8090")

	return &CoreFailedExecutionCleanupService{
		db:           db,
		volumeClient: volumeClient,
		jobTracker:   jobTracker,
	}
}

// CleanupFailedExecution performs core cleanup for a stuck failover operation
func (fecs *CoreFailedExecutionCleanupService) CleanupFailedExecution(ctx context.Context, vmName string) error {
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
	logger.Info("🧹 Starting core failed execution cleanup", "vm_name", vmName)

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

	// Phase 1: Detach volumes from OMA
	err = fecs.jobTracker.RunStep(ctx, jobID, "detach-volumes", func(ctx context.Context) error {
		logger := fecs.jobTracker.Logger(ctx)
		for _, volume := range volumes {
			logger.Info("🔌 Detaching volume from OMA", "volume_id", volume.VolumeID)
			_, detachErr := fecs.volumeClient.DetachVolume(ctx, volume.VolumeID)
			if detachErr != nil {
				logger.Warn("Volume detachment failed or already detached", "volume_id", volume.VolumeID, "error", detachErr)
			}
		}
		return nil
	})
	if err != nil {
		fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return fmt.Errorf("volume detachment failed: %w", err)
	}

	// Phase 2: Reattach volumes to OMA
	err = fecs.jobTracker.RunStep(ctx, jobID, "reattach-volumes", func(ctx context.Context) error {
		logger := fecs.jobTracker.Logger(ctx)
		
		// Get OMA VM ID from database (following Volume Daemon v2.1.2 pattern)
		omaVMID, err := fecs.getOMAVMIDFromDatabase(ctx)
		if err != nil {
			return fmt.Errorf("failed to get OMA VM ID from database: %w", err)
		}
		logger.Info("Retrieved OMA VM ID from database for volume reattachment", "oma_vm_id", omaVMID)

		for _, volume := range volumes {
			logger.Info("🔗 Reattaching volume to OMA", "volume_id", volume.VolumeID)
			_, attachErr := fecs.volumeClient.AttachVolume(ctx, volume.VolumeID, omaVMID)
			if attachErr != nil {
				return fmt.Errorf("failed to reattach volume %s: %w", volume.VolumeID, attachErr)
			}
			time.Sleep(2 * time.Second) // Wait for attachment
		}
		return nil
	})
	if err != nil {
		fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return fmt.Errorf("volume reattachment failed: %w", err)
	}

	// Phase 3: Reset VM state
	err = fecs.jobTracker.RunStep(ctx, jobID, "reset-vm-state", func(ctx context.Context) error {
		logger := fecs.jobTracker.Logger(ctx)

		// Mark failover jobs as failed
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
	logger.Info("✅ Core failed execution cleanup completed successfully", "vm_name", vmName)
	return nil
}

// getOMAVMIDFromDatabase retrieves the OMA VM ID from the active ossea_configs record
// Following Volume Daemon v2.1.2 pattern for dynamic OMA VM ID lookup
func (fecs *CoreFailedExecutionCleanupService) getOMAVMIDFromDatabase(ctx context.Context) (string, error) {
	logger := fecs.jobTracker.Logger(ctx)
	
	var omaVMID string
	err := (*fecs.db).GetGormDB().Raw("SELECT oma_vm_id FROM ossea_configs WHERE is_active = 1 LIMIT 1").Scan(&omaVMID).Error
	if err != nil {
		logger.Error("Failed to query OMA VM ID from database", "error", err)
		return "", fmt.Errorf("failed to query OMA VM ID from ossea_configs: %w", err)
	}
	
	if omaVMID == "" {
		logger.Error("OMA VM ID is empty in database")
		return "", fmt.Errorf("OMA VM ID is empty in ossea_configs table")
	}
	
	logger.Debug("Successfully retrieved OMA VM ID from database", "oma_vm_id", omaVMID)
	return omaVMID, nil
}






