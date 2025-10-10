package services

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
)

// StaleJobDetector monitors running jobs and marks stale jobs as failed
// Runs as a background worker to detect SBC crashes or network failures
type StaleJobDetector struct {
	db              database.Connection
	checkInterval   time.Duration
	staleThreshold  time.Duration
	failedThreshold time.Duration
}

// NewStaleJobDetector creates a new stale job detector
func NewStaleJobDetector(db database.Connection) *StaleJobDetector {
	return &StaleJobDetector{
		db:              db,
		checkInterval:   30 * time.Second,  // Check every 30 seconds
		staleThreshold:  60 * time.Second,  // No update in 60s = stalled warning
		failedThreshold: 300 * time.Second, // No update in 5min = mark as failed
	}
}

// Start begins monitoring for stale jobs
// Runs until context is cancelled
func (sjd *StaleJobDetector) Start(ctx context.Context) {
	ticker := time.NewTicker(sjd.checkInterval)
	defer ticker.Stop()
	
	log.WithFields(log.Fields{
		"check_interval":   sjd.checkInterval,
		"stale_threshold":  sjd.staleThreshold,
		"failed_threshold": sjd.failedThreshold,
	}).Info("🚀 Stale job detector started")
	
	for {
		select {
		case <-ctx.Done():
			log.Info("🛑 Stale job detector stopped")
			return
		case <-ticker.C:
			sjd.checkStaleJobs()
		}
	}
}

// checkStaleJobs scans for running jobs with no recent telemetry
func (sjd *StaleJobDetector) checkStaleJobs() {
	now := time.Now()
	
	// Find running jobs with stale telemetry
	var staleJobs []database.BackupJob
	err := sjd.db.GetGormDB().
		Where("status = ?", "running").
		Where("last_telemetry_at IS NOT NULL").
		Where("last_telemetry_at < ?", now.Add(-sjd.staleThreshold)).
		Find(&staleJobs).Error
	
	if err != nil {
		log.WithError(err).Error("Failed to query for stale jobs")
		return
	}
	
	if len(staleJobs) == 0 {
		return // No stale jobs
	}
	
	log.WithField("stale_jobs_count", len(staleJobs)).Debug("Found stale jobs")
	
	// Process each stale job
	for _, job := range staleJobs {
		staleDuration := now.Sub(*job.LastTelemetryAt)
		
		if staleDuration > sjd.failedThreshold {
			// Mark as failed - no telemetry for 5+ minutes
			err := sjd.db.GetGormDB().
				Model(&database.BackupJob{}).
				Where("id = ?", job.ID).
				Updates(map[string]interface{}{
					"status":        "failed",
					"error_message": fmt.Sprintf("Job stalled - no telemetry for %s (SBC may have crashed)", staleDuration.Round(time.Second)),
					"completed_at":  now,
				}).Error
			
			if err != nil {
				log.WithError(err).WithField("job_id", job.ID).Error("Failed to mark stale job as failed")
			} else {
				log.WithFields(log.Fields{
					"job_id":         job.ID,
					"vm_name":        job.VMName,
					"stale_duration": staleDuration.Round(time.Second),
				}).Warn("⚠️ Marked stale job as failed (no telemetry)")
			}
		} else {
			// Just log warning - not failed yet
			log.WithFields(log.Fields{
				"job_id":         job.ID,
				"vm_name":        job.VMName,
				"stale_duration": staleDuration.Round(time.Second),
			}).Debug("Job stalled but not yet failed - waiting for more time")
		}
	}
}

