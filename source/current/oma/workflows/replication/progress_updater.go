package replication

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/joblog"
)

// ProgressUpdater handles database updates and joblog integration for replication progress
type ProgressUpdater struct {
	db      *sql.DB
	tracker *joblog.Tracker

	// Throttling state
	lastUpdates map[string]*LastUpdateInfo
}

// LastUpdateInfo tracks the last update information for throttling
type LastUpdateInfo struct {
	LastWriteTime      time.Time
	LastProgressPercent float64
	LastCurrentOperation string
	LastStatus         string
}

// NewProgressUpdater creates a new progress updater instance
func NewProgressUpdater(db *sql.DB, tracker *joblog.Tracker) *ProgressUpdater {
	return &ProgressUpdater{
		db:          db,
		tracker:     tracker,
		lastUpdates: make(map[string]*LastUpdateInfo),
	}
}

// UpdateFromVMAProgress processes VMA progress and updates database with throttling
func (u *ProgressUpdater) UpdateFromVMAProgress(ctx context.Context, jobID string, vmaProgress *VMAProgressResponse) error {
	// Check if we should write to database (throttling logic)
	if !u.shouldWriteToDatabase(jobID, vmaProgress) {
		log.WithFields(log.Fields{
			"job_id":  jobID,
			"stage":   vmaProgress.Stage,
			"percent": vmaProgress.Aggregate.Percent,
		}).Debug("Skipping database write due to throttling")
		return nil
	}

	// Update replication_jobs table
	if err := u.updateReplicationJob(ctx, jobID, vmaProgress); err != nil {
		return fmt.Errorf("failed to update replication job: %w", err)
	}

	// Update vm_disks table for each disk
	if err := u.updateVMDisks(ctx, jobID, vmaProgress.Disks); err != nil {
		return fmt.Errorf("failed to update VM disks: %w", err)
	}

	// Update CBT history if job is completed
	if u.isJobCompleted(vmaProgress.Status) {
		if err := u.updateCBTHistory(ctx, jobID, vmaProgress); err != nil {
			log.WithError(err).Warn("Failed to update CBT history") // Non-fatal
		}
	}

	// Update joblog with stage transitions
	if err := u.updateJoblog(ctx, jobID, vmaProgress); err != nil {
		log.WithError(err).Warn("Failed to update joblog") // Non-fatal
	}

	// Update throttling state
	u.updateThrottlingState(jobID, vmaProgress)

	log.WithFields(log.Fields{
		"job_id":           jobID,
		"stage":            vmaProgress.Stage,
		"status":           vmaProgress.Status,
		"progress_percent": vmaProgress.Aggregate.Percent,
		"bytes_transferred": vmaProgress.Aggregate.BytesTransferred,
		"throughput_bps":   vmaProgress.Aggregate.ThroughputBPS,
	}).Debug("Progress update completed")

	return nil
}

// shouldWriteToDatabase determines if we should write to database based on throttling rules
func (u *ProgressUpdater) shouldWriteToDatabase(jobID string, vmaProgress *VMAProgressResponse) bool {
	lastUpdate, exists := u.lastUpdates[jobID]
	if !exists {
		// First update - always write
		return true
	}

	now := time.Now()
	timeSinceLastWrite := now.Sub(lastUpdate.LastWriteTime)

	// Always write if ≥2 seconds since last write
	if timeSinceLastWrite >= 2*time.Second {
		return true
	}

	// Always write if current_operation changed
	if vmaProgress.Stage != lastUpdate.LastCurrentOperation {
		return true
	}

	// Always write if status changed
	if vmaProgress.Status != lastUpdate.LastStatus {
		return true
	}

	// Write if progress changed by ≥1%
	progressDelta := vmaProgress.Aggregate.Percent - lastUpdate.LastProgressPercent
	if progressDelta >= 1.0 {
		return true
	}

	// Don't write - throttled
	return false
}

// updateReplicationJob updates the replication_jobs table
func (u *ProgressUpdater) updateReplicationJob(ctx context.Context, jobID string, vmaProgress *VMAProgressResponse) error {
	query := `
		UPDATE replication_jobs 
		SET 
			status = ?,
			current_operation = ?,
			progress_percent = ?,
			bytes_transferred = ?,
			total_bytes = ?,
			transfer_speed_bps = ?,
			updated_at = NOW()
		WHERE id = ?
	`

	// Map VMA status to database status
	dbStatus := u.mapVMAStatusToDBStatus(vmaProgress.Status)

	result, err := u.db.ExecContext(ctx, query,
		dbStatus,
		vmaProgress.Stage,
		vmaProgress.Aggregate.Percent,
		vmaProgress.Aggregate.BytesTransferred,
		vmaProgress.Aggregate.TotalBytes,
		vmaProgress.Aggregate.ThroughputBPS,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("database update failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated for job_id: %s", jobID)
	}

	return nil
}

// updateVMDisks updates the vm_disks table for individual disk progress
func (u *ProgressUpdater) updateVMDisks(ctx context.Context, jobID string, disks []DiskProgress) error {
	if len(disks) == 0 {
		return nil // No disks to update
	}

	for _, disk := range disks {
		// Map VMA disk status to database status
		dbStatus := u.mapVMADiskStatusToDBStatus(disk.Status)

		query := `
			UPDATE vm_disks 
			SET 
				sync_status = ?,
				sync_progress_percent = ?,
				bytes_synced = ?,
				updated_at = NOW()
			WHERE job_id = ? AND disk_id = ?
		`

		result, err := u.db.ExecContext(ctx, query,
			dbStatus,
			disk.Percent,
			disk.BytesTransferred,
			jobID,
			disk.ID,
		)
		if err != nil {
			log.WithFields(log.Fields{
				"job_id":  jobID,
				"disk_id": disk.ID,
				"error":   err,
			}).Error("Failed to update VM disk progress")
			continue // Continue with other disks
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.WithError(err).Warn("Failed to get rows affected for disk update")
			continue
		}

		if rowsAffected == 0 {
			log.WithFields(log.Fields{
				"job_id":  jobID,
				"disk_id": disk.ID,
			}).Warn("No disk rows updated - disk may not exist in database")
		}
	}

	return nil
}

// updateCBTHistory creates CBT history records when job completes
func (u *ProgressUpdater) updateCBTHistory(ctx context.Context, jobID string, vmaProgress *VMAProgressResponse) error {
	if vmaProgress.CBT.ChangeID == "" {
		log.WithField("job_id", jobID).Debug("No change ID available, skipping CBT history update")
		return nil
	}

	// Calculate sync duration from job start time
	var syncDurationSeconds int64
	if !vmaProgress.StartedAt.IsZero() {
		syncDurationSeconds = int64(time.Since(vmaProgress.StartedAt).Seconds())
	}

	// Create CBT history record for each disk
	for _, disk := range vmaProgress.Disks {
		query := `
			INSERT INTO cbt_history (
				job_id, disk_id, change_id, previous_change_id, sync_type,
				blocks_changed, bytes_transferred, sync_duration_seconds, sync_success,
				created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
			ON DUPLICATE KEY UPDATE
				bytes_transferred = VALUES(bytes_transferred),
				sync_duration_seconds = VALUES(sync_duration_seconds),
				sync_success = VALUES(sync_success)
		`

		syncSuccess := disk.Status == "Completed"
		syncType := vmaProgress.CBT.Type
		if syncType == "" {
			syncType = "full" // Default to full if not specified
		}

		_, err := u.db.ExecContext(ctx, query,
			jobID,
			disk.ID,
			vmaProgress.CBT.ChangeID,
			vmaProgress.CBT.PreviousChangeID,
			syncType,
			nil, // blocks_changed - not available from VMA progress
			disk.BytesTransferred,
			syncDurationSeconds,
			syncSuccess,
		)
		if err != nil {
			log.WithFields(log.Fields{
				"job_id":    jobID,
				"disk_id":   disk.ID,
				"change_id": vmaProgress.CBT.ChangeID,
				"error":     err,
			}).Error("Failed to create CBT history record")
			// Continue with other disks
		}
	}

	return nil
}

// updateJoblog updates joblog with stage transitions
func (u *ProgressUpdater) updateJoblog(ctx context.Context, jobID string, vmaProgress *VMAProgressResponse) error {
	if u.tracker == nil {
		return nil // No tracker available
	}

	// Check if stage changed from last update
	lastUpdate, exists := u.lastUpdates[jobID]
	if exists && lastUpdate.LastCurrentOperation == vmaProgress.Stage {
		return nil // No stage change
	}

	// Run step for stage transition
	stepName := fmt.Sprintf("replication-%s", vmaProgress.Stage)
	return u.tracker.RunStep(ctx, jobID, stepName, func(ctx context.Context) error {
		logger := u.tracker.Logger(ctx)
		logger.Info("Replication stage transition",
			"stage", vmaProgress.Stage,
			"status", vmaProgress.Status,
			"progress_percent", vmaProgress.Aggregate.Percent,
			"throughput_bps", vmaProgress.Aggregate.ThroughputBPS,
		)
		return nil
	})
}

// MarkJobAsFailed marks a job as failed due to timeout or other issues
func (u *ProgressUpdater) MarkJobAsFailed(ctx context.Context, jobID, errorMessage string) error {
	query := `
		UPDATE replication_jobs 
		SET 
			status = 'failed',
			error_message = ?,
			updated_at = NOW(),
			completed_at = NOW()
		WHERE id = ?
	`

	result, err := u.db.ExecContext(ctx, query, errorMessage, jobID)
	if err != nil {
		return fmt.Errorf("failed to mark job as failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated for job_id: %s", jobID)
	}

	log.WithFields(log.Fields{
		"job_id":       jobID,
		"error_message": errorMessage,
	}).Error("Job marked as failed")

	return nil
}

// updateThrottlingState updates the internal throttling state
func (u *ProgressUpdater) updateThrottlingState(jobID string, vmaProgress *VMAProgressResponse) {
	u.lastUpdates[jobID] = &LastUpdateInfo{
		LastWriteTime:        time.Now(),
		LastProgressPercent:  vmaProgress.Aggregate.Percent,
		LastCurrentOperation: vmaProgress.Stage,
		LastStatus:          vmaProgress.Status,
	}
}

// mapVMAStatusToDBStatus maps VMA status values to database status values
func (u *ProgressUpdater) mapVMAStatusToDBStatus(vmaStatus string) string {
	switch vmaStatus {
	case "Queued":
		return "pending"
	case "Preparing":
		return "preparing"
	case "Snapshotting":
		return "snapshotting"
	case "Streaming":
		return "streaming"
	case "Finalizing":
		return "finalizing"
	case "Succeeded":
		return "completed"
	case "Failed":
		return "failed"
	default:
		return "running" // Default fallback
	}
}

// mapVMADiskStatusToDBStatus maps VMA disk status values to database status values
func (u *ProgressUpdater) mapVMADiskStatusToDBStatus(vmaDiskStatus string) string {
	switch vmaDiskStatus {
	case "Queued":
		return "pending"
	case "Snapshotting":
		return "snapshotting"
	case "Streaming":
		return "syncing"
	case "Completed":
		return "completed"
	case "Failed":
		return "failed"
	default:
		return "pending" // Default fallback
	}
}

// isJobCompleted checks if a job status indicates completion
func (u *ProgressUpdater) isJobCompleted(status string) bool {
	switch status {
	case "Succeeded", "Failed":
		return true
	default:
		return false
	}
}

// CleanupThrottlingState removes throttling state for completed jobs
func (u *ProgressUpdater) CleanupThrottlingState(jobID string) {
	delete(u.lastUpdates, jobID)
	log.WithField("job_id", jobID).Debug("Cleaned up throttling state for completed job")
}
