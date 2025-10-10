package services

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
)

// TelemetryUpdate represents a telemetry update from SBC (avoiding circular import)
type TelemetryUpdate struct {
	JobID            string            `json:"job_id"`
	JobType          string            `json:"job_type"`
	Status           string            `json:"status"`
	CurrentPhase     string            `json:"current_phase"`
	BytesTransferred int64             `json:"bytes_transferred"`
	TotalBytes       int64             `json:"total_bytes"`
	TransferSpeedBps int64             `json:"transfer_speed_bps"`
	ETASeconds       int               `json:"eta_seconds"`
	ProgressPercent  float64           `json:"progress_percent"`
	Disks            []DiskTelemetry   `json:"disks"`
	Error            *TelemetryError   `json:"error,omitempty"`
	Timestamp        string            `json:"timestamp,omitempty"`
}

type DiskTelemetry struct {
	DiskIndex        int     `json:"disk_index"`
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes"`
	Status           string  `json:"status"`
	ProgressPercent  float64 `json:"progress_percent"`
}

type TelemetryError struct {
	Message   string `json:"message"`
	Code      string `json:"code,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// TelemetryService processes real-time telemetry updates from SBC
type TelemetryService struct {
	db database.Connection
}

// NewTelemetryService creates a new telemetry service
func NewTelemetryService(db database.Connection) *TelemetryService {
	return &TelemetryService{db: db}
}

// ProcessTelemetryUpdate processes a telemetry update from SBC
// Updates both backup_jobs and backup_disks tables with real-time progress
func (ts *TelemetryService) ProcessTelemetryUpdate(
	ctx context.Context,
	jobType string,
	jobID string,
	update *TelemetryUpdate,
) error {
	now := time.Now()
	
	log.WithFields(log.Fields{
		"job_id":  jobID,
		"status":  update.Status,
		"phase":   update.CurrentPhase,
		"progress": update.ProgressPercent,
		"bytes":   update.BytesTransferred,
	}).Debug("Processing telemetry update")
	
	// Update backup_jobs table with aggregate progress
	// ⚠️ SMART UPDATE: Only update data fields if they contain meaningful values
	// This prevents completion status updates from zeroing out good telemetry data
	updates := map[string]interface{}{
		"last_telemetry_at": now,
	}
	
	// Only update bytes_transferred if non-zero (preserve existing good data)
	if update.BytesTransferred > 0 {
		updates["bytes_transferred"] = update.BytesTransferred
	}
	
	// Only update total_bytes if non-zero
	if update.TotalBytes > 0 {
		updates["total_bytes"] = update.TotalBytes
	}
	
	// Always update phase (important for status tracking)
	if update.CurrentPhase != "" {
		updates["current_phase"] = update.CurrentPhase
	}
	
	// Only update speed if non-zero
	if update.TransferSpeedBps > 0 {
		updates["transfer_speed_bps"] = update.TransferSpeedBps
	}
	
	// Only update ETA if non-zero
	if update.ETASeconds > 0 {
		updates["eta_seconds"] = update.ETASeconds
	}
	
	// Only update progress if non-zero
	if update.ProgressPercent > 0 {
		updates["progress_percent"] = update.ProgressPercent
	}
	
	// Update status if provided
	if update.Status != "" && update.Status != "running" {
		updates["status"] = update.Status
		if update.Status == "completed" {
			updates["completed_at"] = now
		}
	}
	
	// Handle errors
	if update.Error != nil {
		updates["error_message"] = update.Error.Message
		updates["status"] = "failed"
		updates["completed_at"] = now
		
		log.WithFields(log.Fields{
			"job_id": jobID,
			"error":  update.Error.Message,
		}).Error("Backup job failed - error reported via telemetry")
	}
	
	log.WithFields(log.Fields{
		"job_id":            jobID,
		"bytes_transferred": update.BytesTransferred,
		"progress_percent":  update.ProgressPercent,
		"current_phase":     update.CurrentPhase,
	}).Info("💾 ATTEMPTING database update for backup_jobs")
	
	// Update backup_jobs record
	result := ts.db.GetGormDB().
		Model(&database.BackupJob{}).
		Where("id = ?", jobID).
		Updates(updates)
	
	if result.Error != nil {
		log.WithError(result.Error).Error("❌ Database update FAILED")
		return fmt.Errorf("failed to update backup job: %w", result.Error)
	}
	
	log.WithFields(log.Fields{
		"job_id":        jobID,
		"rows_affected": result.RowsAffected,
	}).Info("💾 Database UPDATE executed")
	
	if result.RowsAffected == 0 {
		log.WithField("job_id", jobID).Warn("⚠️  RowsAffected = 0 - job not found or values unchanged")
		return fmt.Errorf("backup job not found: %s", jobID)
	}
	
	log.WithFields(log.Fields{
		"job_id":        jobID,
		"rows_affected": result.RowsAffected,
	}).Info("✅ Database update SUCCESS - rows modified")
	
	// Update per-disk progress in backup_disks table
	for _, disk := range update.Disks {
		diskUpdates := map[string]interface{}{
			"bytes_transferred": disk.BytesTransferred,
			"progress_percent":  disk.ProgressPercent,
		}
		
		// Update disk status if changed
		if disk.Status != "" {
			diskUpdates["status"] = disk.Status
			if disk.Status == "completed" {
				diskUpdates["completed_at"] = now
			}
		}
		
		result := ts.db.GetGormDB().
			Model(&database.BackupDisk{}).
			Where("backup_job_id = ? AND disk_index = ?", jobID, disk.DiskIndex).
			Updates(diskUpdates)
		
		if result.Error != nil {
			log.WithError(result.Error).WithFields(log.Fields{
				"job_id":     jobID,
				"disk_index": disk.DiskIndex,
			}).Warn("Failed to update disk progress")
			// Don't fail - partial update is acceptable
		} else if result.RowsAffected > 0 {
			log.WithFields(log.Fields{
				"job_id":     jobID,
				"disk_index": disk.DiskIndex,
				"progress":   disk.ProgressPercent,
				"bytes":      disk.BytesTransferred,
			}).Debug("Updated disk progress")
		}
	}
	
	log.WithFields(log.Fields{
		"job_id":           jobID,
		"progress_percent": update.ProgressPercent,
		"disks_updated":    len(update.Disks),
	}).Debug("✅ Telemetry update persisted to database")
	
	// 🆕 EVENT-DRIVEN FLOW EXECUTION UPDATE
	// When a backup job completes or fails, check if its parent flow execution is now complete
	if update.Status == "completed" || update.Status == "failed" {
		ts.checkAndUpdateFlowExecution(ctx, jobID)
	}
	
	return nil
}

// checkAndUpdateFlowExecution checks if a backup job belongs to a flow execution
// and updates the execution status if all jobs are complete
func (ts *TelemetryService) checkAndUpdateFlowExecution(ctx context.Context, jobID string) {
	// Find all flow executions that reference this job_id in their created_job_ids
	var executions []struct {
		ID             string
		FlowID         string
		CreatedJobIDs  *string
		Status         string
		StartedAt      time.Time
	}
	
	err := ts.db.GetGormDB().
		Table("protection_flow_executions").
		Select("id, flow_id, created_job_ids, status, started_at").
		Where("status = 'running'").
		Where("created_job_ids LIKE ?", "%"+jobID+"%").
		Find(&executions).Error
	
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Warn("Failed to query flow executions")
		return
	}
	
	if len(executions) == 0 {
		// This job doesn't belong to a flow execution (manual backup)
		return
	}
	
	// Check each execution to see if all its jobs are complete
	for _, execution := range executions {
		ts.updateExecutionIfComplete(ctx, &execution)
	}
}

// updateExecutionIfComplete checks if all jobs for an execution are complete
// and updates the execution status if they are
func (ts *TelemetryService) updateExecutionIfComplete(ctx context.Context, execution *struct {
	ID             string
	FlowID         string
	CreatedJobIDs  *string
	Status         string
	StartedAt      time.Time
}) {
	if execution.CreatedJobIDs == nil || *execution.CreatedJobIDs == "" {
		return
	}
	
	// Parse job IDs from JSON array
	var jobIDs []string
	err := ts.db.GetGormDB().Raw(
		"SELECT JSON_UNQUOTE(JSON_EXTRACT(?, CONCAT('$[', idx, ']'))) FROM "+
		"(SELECT 0 AS idx UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4) AS numbers "+
		"WHERE JSON_EXTRACT(?, CONCAT('$[', idx, ']')) IS NOT NULL",
		*execution.CreatedJobIDs, *execution.CreatedJobIDs,
	).Scan(&jobIDs).Error
	
	if err != nil || len(jobIDs) == 0 {
		return
	}
	
	// Check status of all jobs
	var jobStatuses []struct {
		ID     string
		Status string
	}
	
	err = ts.db.GetGormDB().
		Table("backup_jobs").
		Select("id, status").
		Where("id IN ?", jobIDs).
		Scan(&jobStatuses).Error
	
	if err != nil {
		log.WithError(err).WithField("execution_id", execution.ID).Warn("Failed to query job statuses")
		return
	}
	
	// Count completed/failed jobs
	var completed, failed int
	for _, job := range jobStatuses {
		switch job.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}
	
	totalJobs := len(jobStatuses)
	
	// Check if all jobs are done
	if completed+failed < totalJobs {
		// Still running
		return
	}
	
	// All jobs are done - update execution
	finalStatus := "success"
	if failed > 0 {
		if completed == 0 {
			finalStatus = "error"
		} else {
			finalStatus = "warning"
		}
	}
	
	now := time.Now()
	executionTime := int(now.Sub(execution.StartedAt).Seconds())
	
	log.WithFields(log.Fields{
		"execution_id":   execution.ID,
		"flow_id":        execution.FlowID,
		"final_status":   finalStatus,
		"jobs_completed": completed,
		"jobs_failed":    failed,
		"trigger":        "telemetry_event",
	}).Info("🎉 Flow execution complete (triggered by SBC telemetry)")
	
	// Update execution status
	flowRepo := database.NewFlowRepository(ts.db)
	err = flowRepo.UpdateExecutionStatus(ctx, execution.ID, finalStatus, map[string]interface{}{
		"jobs_completed":         completed,
		"jobs_failed":            failed,
		"completed_at":           now,
		"execution_time_seconds": executionTime,
	})
	
	if err != nil {
		log.WithError(err).WithField("execution_id", execution.ID).Error("Failed to update execution status")
		return
	}
	
	// Update flow statistics
	flow, err := flowRepo.GetFlowByID(ctx, execution.FlowID)
	if err != nil {
		log.WithError(err).WithField("flow_id", execution.FlowID).Error("Failed to get flow for statistics")
		return
	}
	
	successIncrement := 0
	failureIncrement := 0
	if finalStatus == "success" {
		successIncrement = 1
	} else if finalStatus == "error" {
		failureIncrement = 1
	}
	
	err = flowRepo.UpdateFlowStatistics(ctx, execution.FlowID, database.FlowStatistics{
		LastExecutionID:      &execution.ID,
		LastExecutionStatus:  finalStatus,
		LastExecutionTime:    &now,
		TotalExecutions:      flow.TotalExecutions,
		SuccessfulExecutions: flow.SuccessfulExecutions + successIncrement,
		FailedExecutions:     flow.FailedExecutions + failureIncrement,
	})
	
	if err != nil {
		log.WithError(err).WithField("flow_id", execution.FlowID).Error("Failed to update flow statistics")
	}
	
	log.WithFields(log.Fields{
		"execution_id": execution.ID,
		"flow_id":      execution.FlowID,
		"status":       finalStatus,
	}).Info("✅ Flow execution and statistics updated (event-driven)")
}

