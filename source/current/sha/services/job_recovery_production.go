// Package services provides production-ready job recovery functionality
package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vexxhost/migratekit-sha/database"
)

// SNAStatusResult represents the result of querying SNA for job status
type SNAStatusResult struct {
	Status        string  // running, completed, failed, not_found
	Percentage    float64 // Progress percentage
	ErrorMessage  string  // Error message if failed
	IsReachable   bool    // Whether SNA API was reachable
	Phase         string  // Current phase (for running jobs)
	SyncType      string  // Sync type (full/incremental)
	ResponseValid bool    // Whether we got a valid response
}

// ProductionJobRecovery provides job recovery with minimal dependencies
type ProductionJobRecovery struct {
	db                database.Connection
	snaClient         *SNAProgressClient // SNA API client for status validation
	snaProgressPoller *SNAProgressPoller // Poller to restart for active jobs
	maxJobAge         time.Duration
	recoveryEnabled   bool
}

// NewProductionJobRecovery creates a new production job recovery service
func NewProductionJobRecovery(
	db database.Connection,
	snaClient *SNAProgressClient,
	snaProgressPoller *SNAProgressPoller,
) *ProductionJobRecovery {
	return &ProductionJobRecovery{
		db:                db,
		snaClient:         snaClient,
		snaProgressPoller: snaProgressPoller,
		maxJobAge:         30 * time.Minute, // Jobs older than 30 minutes are suspicious
		recoveryEnabled:   true,
	}
}

// RecoverOrphanedJobsOnStartup scans for and recovers orphaned jobs during SHA startup
// Uses SNA validation to make intelligent recovery decisions
func (pjr *ProductionJobRecovery) RecoverOrphanedJobsOnStartup(ctx context.Context) error {
	if !pjr.recoveryEnabled {
		log.Println("Job recovery disabled - skipping startup recovery")
		return nil
	}

	log.Println("🔍 Starting intelligent job recovery with SNA validation on SHA startup")

	// Find ALL jobs in active states (not just "replicating")
	activeJobs, err := pjr.findAllActiveJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to find active jobs: %w", err)
	}

	if len(activeJobs) == 0 {
		log.Println("✅ No active jobs found - system is clean")
		return nil
	}

	log.Printf("🔍 Found %d active jobs requiring recovery validation", len(activeJobs))

	// Recovery statistics
	stats := struct {
		Total           int
		StillRunning    int
		Completed       int
		Failed          int
		SNAUnreachable  int
		PollingRestarted int
		Errors          int
	}{Total: len(activeJobs)}

	// Process each active job with SNA validation
	for _, job := range activeJobs {
		ageMinutes := time.Since(job.UpdatedAt).Minutes()
		totalAgeMinutes := time.Since(job.CreatedAt).Minutes()
		
		log.Printf("🔄 Processing job: %s (%s) - stagnant: %.1f min, total age: %.1f min",
			job.ID, job.SourceVMName, ageMinutes, totalAgeMinutes)

		// Query SNA for actual status
		snaStatus, err := pjr.checkVMAStatus(ctx, &job)
		if err != nil && snaStatus == nil {
			log.Printf("❌ Failed to check SNA status for job %s: %v", job.ID, err)
			stats.Errors++
			continue
		}

		// Make intelligent recovery decision based on SNA status
		if err := pjr.recoverJobWithVMAValidation(&job, snaStatus, ageMinutes); err != nil {
			log.Printf("❌ Failed to recover job %s: %v", job.ID, err)
			stats.Errors++
			continue
		}

		// Update statistics
		switch snaStatus.Status {
		case "running":
			stats.StillRunning++
			if pjr.snaProgressPoller != nil {
				stats.PollingRestarted++
			}
		case "completed":
			stats.Completed++
		case "failed":
			stats.Failed++
		case "unreachable":
			stats.SNAUnreachable++
		}
	}

	log.Printf(`✅ Job recovery completed:
	Total processed: %d
	Still running (polling restarted): %d
	Completed: %d
	Failed: %d
	SNA unreachable: %d
	Errors: %d`,
		stats.Total, stats.StillRunning, stats.Completed, 
		stats.Failed, stats.SNAUnreachable, stats.Errors)

	return nil
}

// recoverJobWithVMAValidation makes intelligent recovery decision based on SNA status
func (pjr *ProductionJobRecovery) recoverJobWithVMAValidation(
	job *database.ReplicationJob,
	snaStatus *SNAStatusResult,
	stagnantMinutes float64,
) error {
	log.Printf("🎯 Recovery decision for job %s: SNA status=%s, stagnant=%.1f min",
		job.ID, snaStatus.Status, stagnantMinutes)

	switch snaStatus.Status {
	case "running":
		// Job is still actively running on SNA - restart polling
		log.Printf("✅ Job %s still running on SNA (%.1f%%) - restarting polling",
			job.ID, snaStatus.Percentage)
		return pjr.restartPollingForRunningJob(job, snaStatus)

	case "completed":
		// Job completed during SHA downtime
		log.Printf("✅ Job %s completed on SNA (%.1f%%) - finalizing",
			job.ID, snaStatus.Percentage)
		return pjr.markAsCompleted(job, snaStatus)

	case "failed":
		// Job failed on SNA
		log.Printf("❌ Job %s failed on SNA - marking as failed with SNA error",
			job.ID)
		return pjr.markAsFailed(job, snaStatus.ErrorMessage, "vma_reported_failure")

	case "not_found":
		// Job not found on SNA - decide based on progress
		if job.ProgressPercent > 90.0 {
			log.Printf("✅ Job %s not found on SNA but was >90%% complete - assuming completed",
				job.ID)
			return pjr.markAsCompleted(job, snaStatus)
		} else {
			log.Printf("❌ Job %s not found on SNA and was <90%% complete - marking as lost",
				job.ID)
			return pjr.markAsFailed(job, "Job lost on SNA (not found after restart)", "job_lost")
		}

	case "unreachable":
		// SNA is unreachable - decide based on job age
		if stagnantMinutes > pjr.maxJobAge.Minutes() {
			log.Printf("❌ Job %s - SNA unreachable and job is old (%.1f min) - marking as failed",
				job.ID, stagnantMinutes)
			return pjr.markAsFailed(job, snaStatus.ErrorMessage, "vma_unreachable_timeout")
		} else {
			log.Printf("⏳ Job %s - SNA unreachable but job is recent (%.1f min) - leaving for retry",
				job.ID, stagnantMinutes)
			// Don't mark as failed yet - SNA might be starting up
			// Health monitor will catch it later if needed
			return nil
		}

	default:
		log.Printf("⚠️ Job %s - unknown SNA status: %s - leaving unchanged",
			job.ID, snaStatus.Status)
		return nil
	}
}

// restartPollingForRunningJob restarts SNA progress polling for a job still running on SNA
func (pjr *ProductionJobRecovery) restartPollingForRunningJob(
	job *database.ReplicationJob,
	snaStatus *SNAStatusResult,
) error {
	// Update job with latest SNA progress
	updates := map[string]interface{}{
		"progress_percent":  snaStatus.Percentage,
		"vma_current_phase": snaStatus.Phase,
		"vma_sync_type":     snaStatus.SyncType,
		"updated_at":        time.Now(),
	}

	if err := pjr.db.GetGormDB().Model(&database.ReplicationJob{}).
		Where("id = ?", job.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	// Restart polling if poller is available
	if pjr.snaProgressPoller != nil {
		if err := pjr.snaProgressPoller.StartPolling(job.ID); err != nil {
			log.Printf("⚠️ Failed to restart polling for job %s: %v", job.ID, err)
			// Don't fail the recovery - job data is updated
		} else {
			log.Printf("🚀 Successfully restarted SNA progress polling for job %s", job.ID)
		}
	} else {
		log.Printf("⚠️ SNA progress poller not available - cannot restart polling for job %s", job.ID)
	}

	return nil
}

// markAsCompleted marks a job as completed with final progress
func (pjr *ProductionJobRecovery) markAsCompleted(
	job *database.ReplicationJob,
	snaStatus *SNAStatusResult,
) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":            "completed",
		"progress_percent":  100.0,
		"current_operation": "Completed",
		"completed_at":      now,
		"updated_at":        now,
	}

	// Add SNA data if available
	if snaStatus != nil && snaStatus.ResponseValid {
		updates["vma_current_phase"] = snaStatus.Phase
		updates["vma_sync_type"] = snaStatus.SyncType
	}

	if err := pjr.db.GetGormDB().Model(&database.ReplicationJob{}).
		Where("id = ?", job.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to mark job as completed: %w", err)
	}

	// Update VM context
	if job.VMContextID != "" {
		contextUpdates := map[string]interface{}{
			"current_status":      "ready_for_failover",
			"last_replication_at": now,
			"successful_jobs":     gorm.Expr("successful_jobs + 1"),
			"updated_at":          now,
		}

		if err := pjr.db.GetGormDB().Model(&database.VMReplicationContext{}).
			Where("context_id = ?", job.VMContextID).Updates(contextUpdates).Error; err != nil {
			log.Printf("⚠️ Failed to update VM context after completion: %v", err)
		}
	}

	log.Printf("✅ Job %s marked as completed", job.ID)
	return nil
}

// markAsFailed marks a job as failed with error classification
func (pjr *ProductionJobRecovery) markAsFailed(
	job *database.ReplicationJob,
	errorMessage string,
	classification string,
) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":                   "failed",
		"error_message":            errorMessage,
		"vma_error_classification": classification,
		"current_operation":        "Failed",
		"completed_at":             now,
		"updated_at":               now,
	}

	if err := pjr.db.GetGormDB().Model(&database.ReplicationJob{}).
		Where("id = ?", job.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to mark job as failed: %w", err)
	}

	// Update VM context and clear current_job_id to allow new operations
	if job.VMContextID != "" {
		contextUpdates := map[string]interface{}{
			"current_status": "ready_for_failover",
			"current_job_id": nil, // 🚨 CRITICAL: Clear job reference so new jobs can start
			"failed_jobs":    gorm.Expr("failed_jobs + 1"),
			"updated_at":     now,
		}

		if err := pjr.db.GetGormDB().Model(&database.VMReplicationContext{}).
			Where("context_id = ?", job.VMContextID).Updates(contextUpdates).Error; err != nil {
			log.Printf("⚠️ Failed to update VM context after failure: %v", err)
		} else {
			log.Printf("✅ Cleared current_job_id for VM context %s - new operations now allowed", job.VMContextID)
		}
	}

	log.Printf("❌ Job %s marked as failed: %s", job.ID, errorMessage)
	return nil
}

// findAllActiveJobs finds ALL jobs in active states requiring recovery validation
func (pjr *ProductionJobRecovery) findAllActiveJobs(ctx context.Context) ([]database.ReplicationJob, error) {
	var activeJobs []database.ReplicationJob

	// Find jobs in any active state (not just "replicating")
	// Include: replicating, initializing, ready_for_sync, attaching, configuring
	activeStatuses := []string{
		"replicating",
		"initializing",
		"ready_for_sync",
		"attaching",
		"configuring",
		"provisioning",
		"analyzing",
	}

	// 🚨 CRITICAL FIX: Find ALL active jobs regardless of update time
	// When SHA restarts, the polling map is lost, so we need to check ALL active jobs
	// to see if they need polling restarted, even if they were recently updated
	query := pjr.db.GetGormDB().Where("status IN ?", activeStatuses)

	if err := query.Find(&activeJobs).Error; err != nil {
		return nil, fmt.Errorf("failed to query active jobs: %w", err)
	}

	log.Printf("📊 Found %d active jobs in states: %v", len(activeJobs), activeStatuses)
	return activeJobs, nil
}

// GetOrphanedJobStatus returns status of potentially orphaned jobs
func (pjr *ProductionJobRecovery) GetOrphanedJobStatus() ([]map[string]interface{}, error) {
	var stuckJobs []database.ReplicationJob
	query := `
		SELECT id, source_vm_name, status, created_at, updated_at, progress_percent, vm_context_id
		FROM replication_jobs 
		WHERE status = 'replicating' 
		ORDER BY updated_at ASC
	`

	err := pjr.db.GetGormDB().Raw(query).Find(&stuckJobs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get stuck jobs: %w", err)
	}

	var jobStatuses []map[string]interface{}
	for _, job := range stuckJobs {
		ageMinutes := time.Since(job.CreatedAt).Minutes()
		stagnantMinutes := time.Since(job.UpdatedAt).Minutes()

		status := map[string]interface{}{
			"job_id":            job.ID,
			"vm_name":           job.SourceVMName,
			"status":            job.Status,
			"age_minutes":       ageMinutes,
			"stagnant_minutes":  stagnantMinutes,
			"progress_percent":  job.ProgressPercent,
			"vm_context_id":     job.VMContextID,
			"last_update":       job.UpdatedAt,
			"requires_recovery": stagnantMinutes > pjr.maxJobAge.Minutes(),
		}

		jobStatuses = append(jobStatuses, status)
	}

	return jobStatuses, nil
}

// EnableRecovery enables or disables automatic job recovery
func (pjr *ProductionJobRecovery) EnableRecovery(enabled bool) {
	pjr.recoveryEnabled = enabled
	if enabled {
		log.Println("✅ Job recovery enabled")
	} else {
		log.Println("⚠️ Job recovery disabled")
	}
}

// SetMaxJobAge configures the maximum job age for recovery
func (pjr *ProductionJobRecovery) SetMaxJobAge(maxAge time.Duration) {
	pjr.maxJobAge = maxAge
	log.Printf("🔧 Job recovery max age set to %v", maxAge)
}

// checkVMAStatus queries SNA API to determine actual job status
// Returns comprehensive status information for recovery decision making
func (pjr *ProductionJobRecovery) checkVMAStatus(ctx context.Context, job *database.ReplicationJob) (*SNAStatusResult, error) {
	if pjr.snaClient == nil {
		return &SNAStatusResult{
			Status:        "unknown",
			IsReachable:   false,
			ResponseValid: false,
			ErrorMessage:  "SNA client not initialized",
		}, fmt.Errorf("SNA client not available")
	}

	log.Printf("🔍 Checking SNA status for job %s (%s)", job.ID, job.SourceVMName)

	// Phase 1: Try NBD export name method first (primary method)
	nbdExportNames, err := pjr.getNBDExportNamesForJob(job.ID)
	if err == nil && len(nbdExportNames) > 0 {
		log.Printf("🔗 Found %d NBD export names for job %s, trying progress API", len(nbdExportNames), job.ID)
		
		for _, exportName := range nbdExportNames {
			progress, err := pjr.snaClient.GetProgress(exportName)
			if err == nil {
				log.Printf("✅ Got SNA response via NBD export name %s", exportName)
				return pjr.parseVMAResponse(progress, true), nil
			}
			
			// Check if it's a "not found" error
			if snaErr, ok := err.(*SNAProgressError); ok {
				if snaErr.StatusCode == 404 {
					log.Printf("⚠️ NBD export %s not found on SNA (404)", exportName)
					continue
				}
				// Check for HTTP 200 with "not found" message (known bug)
				if snaErr.StatusCode == 200 && strings.Contains(strings.ToLower(snaErr.Message), "not found") {
					log.Printf("⚠️ NBD export %s not found on SNA (HTTP 200 'not found')", exportName)
					continue
				}
			}
			
			log.Printf("⚠️ NBD export %s query failed: %v", exportName, err)
		}
		log.Printf("⚠️ All NBD export names failed, falling back to job ID")
	} else if err != nil {
		log.Printf("⚠️ Failed to get NBD export names: %v", err)
	}

	// Phase 2: Fallback to job ID method
	log.Printf("🔄 Trying traditional job ID method for %s", job.ID)
	progress, err := pjr.snaClient.GetProgress(job.ID)
	if err != nil {
		// Check if it's a connection error (SNA unreachable)
		if strings.Contains(err.Error(), "connection refused") || 
		   strings.Contains(err.Error(), "no such host") ||
		   strings.Contains(err.Error(), "timeout") {
			log.Printf("❌ SNA unreachable for job %s: %v", job.ID, err)
			return &SNAStatusResult{
				Status:        "unreachable",
				IsReachable:   false,
				ResponseValid: false,
				ErrorMessage:  fmt.Sprintf("SNA API unreachable: %v", err),
			}, nil
		}

		// Check for "not found" errors
		if snaErr, ok := err.(*SNAProgressError); ok {
			if snaErr.StatusCode == 404 {
				log.Printf("📋 Job %s not found on SNA (404) - likely completed or lost", job.ID)
				return &SNAStatusResult{
					Status:        "not_found",
					IsReachable:   true,
					ResponseValid: true,
					ErrorMessage:  "Job not found on SNA",
				}, nil
			}
			// HTTP 200 with "not found" (known bug)
			if snaErr.StatusCode == 200 && strings.Contains(strings.ToLower(snaErr.Message), "not found") {
				log.Printf("📋 Job %s not found on SNA (HTTP 200 'not found') - likely completed or lost", job.ID)
				return &SNAStatusResult{
					Status:        "not_found",
					IsReachable:   true,
					ResponseValid: true,
					ErrorMessage:  "Job not found on SNA",
				}, nil
			}
		}

		log.Printf("❌ Failed to get SNA status for job %s: %v", job.ID, err)
		return &SNAStatusResult{
			Status:        "error",
			IsReachable:   true, // We got a response, just not what we expected
			ResponseValid: false,
			ErrorMessage:  fmt.Sprintf("SNA API error: %v", err),
		}, fmt.Errorf("failed to query SNA: %w", err)
	}

	log.Printf("✅ Got SNA response via job ID for %s", job.ID)
	return pjr.parseVMAResponse(progress, true), nil
}

// parseVMAResponse converts SNA progress response to recovery status result
func (pjr *ProductionJobRecovery) parseVMAResponse(progress *SNAProgressResponse, isReachable bool) *SNAStatusResult {
	result := &SNAStatusResult{
		IsReachable:   isReachable,
		ResponseValid: true,
		Percentage:    progress.Percentage,
		Phase:         progress.Phase,
		SyncType:      progress.SyncType,
	}

	// Determine status from SNA response
	switch {
	case progress.Status == "completed" || progress.Phase == "Completed" || progress.Phase == "completed":
		result.Status = "completed"
		result.Percentage = 100.0
		log.Printf("✅ SNA reports job completed (status=%s, phase=%s)", progress.Status, progress.Phase)
		
	case progress.Status == "failed" || progress.Phase == "Error" || len(progress.Errors) > 0:
		result.Status = "failed"
		if len(progress.Errors) > 0 {
			result.ErrorMessage = fmt.Sprintf("SNA reported errors: %v", progress.Errors)
		} else if progress.LastError != nil {
			result.ErrorMessage = fmt.Sprintf("SNA reported error: %v", progress.LastError)
		} else {
			result.ErrorMessage = "SNA reported failure status"
		}
		log.Printf("❌ SNA reports job failed: %s", result.ErrorMessage)
		
	case progress.Phase == "Copying Data" || progress.Phase == "Initializing" || progress.Phase == "Snapshot Creation":
		result.Status = "running"
		log.Printf("🔄 SNA reports job running (phase=%s, progress=%.1f%%)", progress.Phase, progress.Percentage)
		
	default:
		// If we have progress updates, assume it's running
		if progress.Percentage > 0 && progress.Percentage < 100 {
			result.Status = "running"
			log.Printf("🔄 SNA reports job in progress (%.1f%%)", progress.Percentage)
		} else {
			result.Status = "unknown"
			log.Printf("⚠️ SNA status unclear (status=%s, phase=%s, progress=%.1f%%)", 
				progress.Status, progress.Phase, progress.Percentage)
		}
	}

	return result
}

// getNBDExportNamesForJob queries database to construct NBD export names for a job
// This enables the primary method of querying SNA progress via NBD export names
func (pjr *ProductionJobRecovery) getNBDExportNamesForJob(jobID string) ([]string, error) {
	// Query: replication_jobs → vm_disks → ossea_volumes → volume_id
	// Construct: migration-vol-{volume_id}
	
	query := `
		SELECT DISTINCT ov.volume_id 
		FROM replication_jobs rj
		JOIN vm_disks vd ON rj.id = vd.job_id
		JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
		WHERE rj.id = ?
		ORDER BY vd.unit_number ASC
	`

	var volumeIDs []string
	rows, err := pjr.db.GetGormDB().Raw(query, jobID).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to query volume IDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var volumeID string
		if err := rows.Scan(&volumeID); err != nil {
			log.Printf("⚠️ Failed to scan volume ID: %v", err)
			continue
		}
		volumeIDs = append(volumeIDs, volumeID)
	}

	if len(volumeIDs) == 0 {
		return []string{}, fmt.Errorf("no volume IDs found for job %s", jobID)
	}

	// Construct NBD export names: migration-vol-{volume_uuid}
	var nbdExportNames []string
	for _, volumeID := range volumeIDs {
		exportName := fmt.Sprintf("migration-vol-%s", volumeID)
		nbdExportNames = append(nbdExportNames, exportName)
	}

	log.Printf("🔗 Constructed %d NBD export names for job %s", len(nbdExportNames), jobID)
	return nbdExportNames, nil
}








