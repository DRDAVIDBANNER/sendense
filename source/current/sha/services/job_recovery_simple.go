// Package services provides simplified job recovery functionality
package services

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-sha/database"
)

// SimpleJobRecovery provides basic job recovery without complex dependencies
type SimpleJobRecovery struct {
	db              database.Connection
	maxJobAge       time.Duration
	recoveryEnabled bool
}

// NewSimpleJobRecovery creates a new simple job recovery service
func NewSimpleJobRecovery(db database.Connection) *SimpleJobRecovery {
	return &SimpleJobRecovery{
		db:              db,
		maxJobAge:       30 * time.Minute,
		recoveryEnabled: true,
	}
}

// RecoverOrphanedJobs scans for and recovers orphaned replication jobs
func (sjr *SimpleJobRecovery) RecoverOrphanedJobs(ctx context.Context) error {
	if !sjr.recoveryEnabled {
		log.Info("📋 Job recovery disabled")
		return nil
	}

	log.Info("🔍 Scanning for orphaned replication jobs")

	// Find jobs stuck in 'replicating' status
	var orphanedJobs []database.ReplicationJob
	query := `
		SELECT * FROM replication_jobs 
		WHERE status = 'replicating' 
		AND updated_at < NOW() - INTERVAL ? MINUTE
		ORDER BY updated_at ASC
	`

	err := sjr.db.GetGormDB().Raw(query, int(sjr.maxJobAge.Minutes())).Find(&orphanedJobs).Error
	if err != nil {
		return fmt.Errorf("failed to find orphaned jobs: %w", err)
	}

	if len(orphanedJobs) == 0 {
		log.Info("✅ No orphaned jobs found")
		return nil
	}

	log.WithField("orphaned_count", len(orphanedJobs)).Warn("🔍 Found potentially orphaned jobs")

	for _, job := range orphanedJobs {
		log.WithFields(log.Fields{
			"job_id":      job.ID,
			"vm_name":     job.SourceVMName,
			"age_minutes": time.Since(job.UpdatedAt).Minutes(),
		}).Warn("🔄 Recovering orphaned job")

		// Mark job as failed and update VM context
		if err := sjr.recoverJob(&job); err != nil {
			log.WithError(err).WithField("job_id", job.ID).Error("Failed to recover job")
			continue
		}

		log.WithField("job_id", job.ID).Info("✅ Job recovered successfully")
	}

	return nil
}

// recoverJob recovers a specific orphaned job
func (sjr *SimpleJobRecovery) recoverJob(job *database.ReplicationJob) error {
	// Mark job as failed
	updates := map[string]interface{}{
		"status":                   "failed",
		"error_message":            "Job recovery: Process orphaned during service restart",
		"vma_error_classification": "startup_recovery",
		"completed_at":             time.Now(),
		"updated_at":               time.Now(),
	}

	err := sjr.db.GetGormDB().Model(&database.ReplicationJob{}).Where("id = ?", job.ID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Update VM context to allow new operations
	if job.VMContextID != "" {
		contextUpdates := map[string]interface{}{
			"current_status": "ready_for_failover",
			"current_job_id": nil,
			"failed_jobs":    "failed_jobs + 1",
			"updated_at":     time.Now(),
		}

		err = sjr.db.GetGormDB().Model(&database.VMReplicationContext{}).
			Where("context_id = ?", job.VMContextID).Updates(contextUpdates).Error
		if err != nil {
			log.WithError(err).WithField("vm_context_id", job.VMContextID).Warn("Failed to update VM context")
		}
	}

	return nil
}

// GetOrphanedJobStatus returns status of potentially orphaned jobs
func (sjr *SimpleJobRecovery) GetOrphanedJobStatus() ([]map[string]interface{}, error) {
	var stuckJobs []database.ReplicationJob
	query := `
		SELECT id, source_vm_name, status, created_at, updated_at, progress_percent, vma_last_poll_at
		FROM replication_jobs 
		WHERE status = 'replicating' 
		ORDER BY updated_at ASC
	`

	err := sjr.db.GetGormDB().Raw(query).Find(&stuckJobs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get stuck jobs: %w", err)
	}

	var jobStatuses []map[string]interface{}
	for _, job := range stuckJobs {
		status := map[string]interface{}{
			"job_id":           job.ID,
			"vm_name":          job.SourceVMName,
			"status":           job.Status,
			"age_minutes":      time.Since(job.CreatedAt).Minutes(),
			"stagnant_minutes": time.Since(job.UpdatedAt).Minutes(),
			"progress_percent": job.ProgressPercent,
			"last_update":      job.UpdatedAt,
		}

		if job.SNALastPollAt != nil {
			status["last_vma_poll"] = job.SNALastPollAt
			status["vma_poll_age_minutes"] = time.Since(*job.SNALastPollAt).Minutes()
		}

		jobStatuses = append(jobStatuses, status)
	}

	return jobStatuses, nil
}








