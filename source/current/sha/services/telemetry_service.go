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
	
	return nil
}

