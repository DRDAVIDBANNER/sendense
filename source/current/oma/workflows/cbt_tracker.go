// Package workflows - CBT (Change Block Tracking) management for incremental migrations
package workflows

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
)

// initializeCBTTracking sets up CBT tracking for the migration
func (m *MigrationEngine) initializeCBTTracking(req *MigrationRequest) error {
	log.WithField("job_id", req.JobID).Info("Initializing CBT tracking")

	// Get VM disks for this job
	vmDisks, err := m.vmDiskRepo.GetByJobID(req.JobID)
	if err != nil {
		return fmt.Errorf("failed to get VM disks for CBT tracking: %w", err)
	}

	// Get VM context ID for this job
	vmContextID, err := m.getVMContextIDForJob(req.JobID)
	if err != nil {
		log.WithError(err).WithField("job_id", req.JobID).Warn("Failed to get VM context ID for CBT tracking, continuing without it")
		vmContextID = "" // Continue without VM context for backward compatibility
	}

	for _, vmDisk := range vmDisks {
		// Create initial CBT history record
		cbtHistory := &database.CBTHistory{
			JobID:            req.JobID,
			VMContextID:      vmContextID, // VM-Centric Architecture integration
			DiskID:           vmDisk.DiskID,
			ChangeID:         req.ChangeID,
			PreviousChangeID: req.PreviousChangeID,
			SyncType:         req.ReplicationType,
			SyncSuccess:      false, // Will be updated when sync completes
			CreatedAt:        time.Now(),
		}

		if err := m.cbtHistoryRepo.Create(cbtHistory); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"job_id":  req.JobID,
				"disk_id": vmDisk.DiskID,
			}).Warn("Failed to create CBT history record")
			continue
		}

		// Update VM disk with change ID
		if req.ChangeID != "" {
			if err := m.vmDiskRepo.UpdateChangeID(vmDisk.ID, req.ChangeID); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"job_id":    req.JobID,
					"disk_id":   vmDisk.DiskID,
					"change_id": req.ChangeID,
				}).Warn("Failed to update VM disk change ID")
			}
		}

		log.WithFields(log.Fields{
			"job_id":             req.JobID,
			"disk_id":            vmDisk.DiskID,
			"change_id":          req.ChangeID,
			"previous_change_id": req.PreviousChangeID,
			"sync_type":          req.ReplicationType,
		}).Info("CBT tracking initialized for disk")
	}

	return nil
}

// ValidateCBTChangeID checks if the provided change ID is valid for incremental sync
func (m *MigrationEngine) ValidateCBTChangeID(jobID, diskID, changeID string) (bool, error) {
	log.WithFields(log.Fields{
		"job_id":    jobID,
		"disk_id":   diskID,
		"change_id": changeID,
	}).Debug("Validating CBT change ID")

	if changeID == "" {
		log.Debug("Empty change ID, assuming full sync")
		return true, nil // Empty change ID means full sync
	}

	// Get latest CBT history for the disk
	history, err := m.cbtHistoryRepo.GetLatestByDiskID(diskID)
	if err != nil {
		log.WithError(err).Debug("No previous CBT history found, allowing change ID")
		return true, nil // No history means this is acceptable
	}

	// Check if the change ID matches the latest successful sync
	if history.SyncSuccess && history.ChangeID == changeID {
		log.Debug("Change ID matches latest successful sync")
		return true, nil
	}

	// Check if this is a reasonable incremental change ID
	// In a real implementation, this would validate the change ID format
	// and ensure it's newer than the previous one
	log.WithFields(log.Fields{
		"current_change_id":  changeID,
		"previous_change_id": history.ChangeID,
		"previous_success":   history.SyncSuccess,
	}).Debug("Change ID validation result")

	return true, nil // For now, accept all change IDs
}

// InvalidateCBTTracking marks CBT tracking as invalid, forcing a full resync
func (m *MigrationEngine) InvalidateCBTTracking(jobID, diskID, reason string) error {
	log.WithFields(log.Fields{
		"job_id":  jobID,
		"disk_id": diskID,
		"reason":  reason,
	}).Info("Invalidating CBT tracking")

	// Get the latest CBT history for the disk
	history, err := m.cbtHistoryRepo.GetLatestByDiskID(diskID)
	if err != nil {
		log.WithError(err).Debug("No CBT history found to invalidate")
		return nil // No history to invalidate
	}

	// Get VM context ID for this job
	vmContextID, err := m.getVMContextIDForJob(jobID)
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Warn("Failed to get VM context ID for CBT invalidation, continuing without it")
		vmContextID = "" // Continue without VM context for backward compatibility
	}

	// Create a new CBT history record indicating invalidation
	invalidationHistory := &database.CBTHistory{
		JobID:               jobID,
		VMContextID:         vmContextID, // VM-Centric Architecture integration
		DiskID:              diskID,
		ChangeID:            "", // Empty change ID indicates full sync needed
		PreviousChangeID:    history.ChangeID,
		SyncType:            "full", // Force full sync
		SyncSuccess:         false,
		BlocksChanged:       0, // Will be updated during sync
		BytesTransferred:    0, // Will be updated during sync
		SyncDurationSeconds: 0, // Will be updated during sync
		CreatedAt:           time.Now(),
	}

	if err := m.cbtHistoryRepo.Create(invalidationHistory); err != nil {
		return fmt.Errorf("failed to create CBT invalidation record: %w", err)
	}

	// Update VM disk to clear change ID (forcing full sync)
	vmDisks, err := m.vmDiskRepo.GetByJobID(jobID)
	if err != nil {
		return fmt.Errorf("failed to get VM disks: %w", err)
	}

	for _, vmDisk := range vmDisks {
		if vmDisk.DiskID == diskID {
			if err := m.vmDiskRepo.UpdateChangeID(vmDisk.ID, ""); err != nil {
				log.WithError(err).Warn("Failed to clear VM disk change ID")
			}
			break
		}
	}

	log.WithFields(log.Fields{
		"job_id":  jobID,
		"disk_id": diskID,
		"reason":  reason,
	}).Info("CBT tracking invalidated, full sync required")

	return nil
}

// UpdateCBTProgress updates the progress of a CBT sync operation
func (m *MigrationEngine) UpdateCBTProgress(jobID, diskID string, bytesTransferred int64) error {
	log.WithFields(log.Fields{
		"job_id":            jobID,
		"disk_id":           diskID,
		"bytes_transferred": bytesTransferred,
	}).Debug("Updating CBT sync progress")

	// Get the latest CBT history for the disk
	history, err := m.cbtHistoryRepo.GetLatestByDiskID(diskID)
	if err != nil {
		return fmt.Errorf("failed to get CBT history for progress update: %w", err)
	}

	// Update the bytes transferred (sync completion will be handled separately)
	updates := map[string]interface{}{
		"bytes_transferred": bytesTransferred,
	}

	if err := m.db.GetGormDB().Model(&database.CBTHistory{}).Where("id = ?", history.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update CBT progress: %w", err)
	}

	return nil
}

// CompleteCBTSync marks a CBT sync as completed
func (m *MigrationEngine) CompleteCBTSync(jobID, diskID string, success bool, bytesTransferred int64, blocksChanged int, durationSeconds int) error {
	log.WithFields(log.Fields{
		"job_id":            jobID,
		"disk_id":           diskID,
		"success":           success,
		"bytes_transferred": bytesTransferred,
		"blocks_changed":    blocksChanged,
		"duration_seconds":  durationSeconds,
	}).Info("Completing CBT sync")

	// Get the latest CBT history for the disk
	history, err := m.cbtHistoryRepo.GetLatestByDiskID(diskID)
	if err != nil {
		return fmt.Errorf("failed to get CBT history for completion: %w", err)
	}

	// Mark sync as completed
	if err := m.cbtHistoryRepo.MarkSyncCompleted(history.ID, success, bytesTransferred, durationSeconds); err != nil {
		return fmt.Errorf("failed to mark CBT sync as completed: %w", err)
	}

	// Update blocks changed if provided
	if blocksChanged > 0 {
		updates := map[string]interface{}{
			"blocks_changed": blocksChanged,
		}
		if err := m.db.GetGormDB().Model(&database.CBTHistory{}).Where("id = ?", history.ID).Updates(updates).Error; err != nil {
			log.WithError(err).Warn("Failed to update blocks changed")
		}
	}

	// Update VM disk sync status
	vmDisks, err := m.vmDiskRepo.GetByJobID(jobID)
	if err != nil {
		log.WithError(err).Warn("Failed to get VM disks for status update")
		return nil // Non-fatal
	}

	for _, vmDisk := range vmDisks {
		if vmDisk.DiskID == diskID {
			status := "completed"
			if !success {
				status = "failed"
			}

			progressPercent := 100.0
			if !success {
				progressPercent = vmDisk.SyncProgressPercent // Keep existing progress
			}

			if err := m.vmDiskRepo.UpdateSyncStatus(vmDisk.ID, status, progressPercent, bytesTransferred); err != nil {
				log.WithError(err).Warn("Failed to update VM disk sync status")
			}
			break
		}
	}

	log.WithFields(log.Fields{
		"job_id":            jobID,
		"disk_id":           diskID,
		"success":           success,
		"bytes_transferred": bytesTransferred,
	}).Info("CBT sync completed")

	return nil
}

// GetCBTHistory retrieves CBT history for a job or disk
func (m *MigrationEngine) GetCBTHistory(jobID string, diskID string) ([]database.CBTHistory, error) {
	if diskID != "" {
		return m.cbtHistoryRepo.GetByDiskID(diskID)
	}
	return m.cbtHistoryRepo.GetByJobID(jobID)
}

// DetectCBTReset detects if a CBT reset has occurred in VMware
func (m *MigrationEngine) DetectCBTReset(jobID, diskID, currentChangeID string) (bool, error) {
	log.WithFields(log.Fields{
		"job_id":            jobID,
		"disk_id":           diskID,
		"current_change_id": currentChangeID,
	}).Debug("Detecting CBT reset")

	// Get the latest CBT history
	history, err := m.cbtHistoryRepo.GetLatestByDiskID(diskID)
	if err != nil {
		log.WithError(err).Debug("No CBT history found, no reset detected")
		return false, nil // No history means no reset
	}

	// Check for CBT reset indicators:
	// 1. Change ID format changed (different length/pattern)
	// 2. Change ID is unexpectedly different from expected sequence
	// 3. VMware reports a reset through other mechanisms

	// Simple heuristic: if change IDs are completely different format
	if len(currentChangeID) != len(history.ChangeID) && history.ChangeID != "" {
		log.WithFields(log.Fields{
			"current_change_id":  currentChangeID,
			"previous_change_id": history.ChangeID,
		}).Warn("CBT change ID format changed, possible reset detected")
		return true, nil
	}

	// More sophisticated reset detection would be implemented here
	// For now, assume no reset unless explicitly detected
	return false, nil
}
