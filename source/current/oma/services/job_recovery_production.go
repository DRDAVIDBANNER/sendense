// Package services provides production-ready job recovery functionality
package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vexxhost/migratekit-oma/database"
)

// VMAStatusResult represents the result of querying VMA for job status
type VMAStatusResult struct {
	Status        string  // running, completed, failed, not_found
	Percentage    float64 // Progress percentage
	ErrorMessage  string  // Error message if failed
	IsReachable   bool    // Whether VMA API was reachable
	Phase         string  // Current phase (for running jobs)
	SyncType      string  // Sync type (full/incremental)
	ResponseValid bool    // Whether we got a valid response
}

// ProductionJobRecovery provides job recovery with minimal dependencies
type ProductionJobRecovery struct {
	db                database.Connection
	vmaClient         *VMAProgressClient // VMA API client for status validation
	vmaProgressPoller *VMAProgressPoller // Poller to restart for active jobs
	maxJobAge         time.Duration
	recoveryEnabled   bool
}

// NewProductionJobRecovery creates a new production job recovery service
func NewProductionJobRecovery(
	db database.Connection,
	vmaClient *VMAProgressClient,
	vmaProgressPoller *VMAProgressPoller,
) *ProductionJobRecovery {
	return &ProductionJobRecovery{
		db:                db,
		vmaClient:         vmaClient,
		vmaProgressPoller: vmaProgressPoller,
		maxJobAge:         30 * time.Minute, // Jobs older than 30 minutes are suspicious
		recoveryEnabled:   true,
	}
}

// RecoverOrphanedJobsOnStartup scans for and recovers orphaned jobs during OMA startup
// Uses VMA validation to make intelligent recovery decisions
func (pjr *ProductionJobRecovery) RecoverOrphanedJobsOnStartup(ctx context.Context) error {
	if !pjr.recoveryEnabled {
		log.Println("Job recovery disabled - skipping startup recovery")
		return nil
	}

	log.Println("🔍 Starting intelligent job recovery with VMA validation on OMA startup")

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
		VMAUnreachable  int
		PollingRestarted int
		Errors          int
	}{Total: len(activeJobs)}

	// Process each active job with VMA validation
	for _, job := range activeJobs {
		ageMinutes := time.Since(job.UpdatedAt).Minutes()
		totalAgeMinutes := time.Since(job.CreatedAt).Minutes()
		
		log.Printf("🔄 Processing job: %s (%s) - stagnant: %.1f min, total age: %.1f min",
			job.ID, job.SourceVMName, ageMinutes, totalAgeMinutes)

		// Query VMA for actual status
		vmaStatus, err := pjr.checkVMAStatus(ctx, &job)
		if err != nil && vmaStatus == nil {
			log.Printf("❌ Failed to check VMA status for job %s: %v", job.ID, err)
			stats.Errors++
			continue
		}

		// Make intelligent recovery decision based on VMA status
		if err := pjr.recoverJobWithVMAValidation(&job, vmaStatus, ageMinutes); err != nil {
			log.Printf("❌ Failed to recover job %s: %v", job.ID, err)
			stats.Errors++
			continue
		}

		// Update statistics
		switch vmaStatus.Status {
		case "running":
			stats.StillRunning++
			if pjr.vmaProgressPoller != nil {
				stats.PollingRestarted++
			}
		case "completed":
			stats.Completed++
		case "failed":
			stats.Failed++
		case "unreachable":
			stats.VMAUnreachable++
		}
	}

	log.Printf(`✅ Job recovery completed:
	Total processed: %d
	Still running (polling restarted): %d
	Completed: %d
	Failed: %d
	VMA unreachable: %d
	Errors: %d`,
		stats.Total, stats.StillRunning, stats.Completed, 
		stats.Failed, stats.VMAUnreachable, stats.Errors)

	return nil
}

// recoverJobWithVMAValidation makes intelligent recovery decision based on VMA status
func (pjr *ProductionJobRecovery) recoverJobWithVMAValidation(
	job *database.ReplicationJob,
	vmaStatus *VMAStatusResult,
	stagnantMinutes float64,
) error {
	log.Printf("🎯 Recovery decision for job %s: VMA status=%s, stagnant=%.1f min",
		job.ID, vmaStatus.Status, stagnantMinutes)

	switch vmaStatus.Status {
	case "running":
		// Job is still actively running on VMA - restart polling
		log.Printf("✅ Job %s still running on VMA (%.1f%%) - restarting polling",
			job.ID, vmaStatus.Percentage)
		return pjr.restartPollingForRunningJob(job, vmaStatus)

	case "completed":
		// Job completed during OMA downtime
		log.Printf("✅ Job %s completed on VMA (%.1f%%) - finalizing",
			job.ID, vmaStatus.Percentage)
		return pjr.markAsCompleted(job, vmaStatus)

	case "failed":
		// Job failed on VMA
		log.Printf("❌ Job %s failed on VMA - marking as failed with VMA error",
			job.ID)
		return pjr.markAsFailed(job, vmaStatus.ErrorMessage, "vma_reported_failure")

	case "not_found":
		// Job not found on VMA - decide based on progress
		if job.ProgressPercent > 90.0 {
			log.Printf("✅ Job %s not found on VMA but was >90%% complete - assuming completed",
				job.ID)
			return pjr.markAsCompleted(job, vmaStatus)
		} else {
			log.Printf("❌ Job %s not found on VMA and was <90%% complete - marking as lost",
				job.ID)
			return pjr.markAsFailed(job, "Job lost on VMA (not found after restart)", "job_lost")
		}

	case "unreachable":
		// VMA is unreachable - decide based on job age
		if stagnantMinutes > pjr.maxJobAge.Minutes() {
			log.Printf("❌ Job %s - VMA unreachable and job is old (%.1f min) - marking as failed",
				job.ID, stagnantMinutes)
			return pjr.markAsFailed(job, vmaStatus.ErrorMessage, "vma_unreachable_timeout")
		} else {
			log.Printf("⏳ Job %s - VMA unreachable but job is recent (%.1f min) - leaving for retry",
				job.ID, stagnantMinutes)
			// Don't mark as failed yet - VMA might be starting up
			// Health monitor will catch it later if needed
			return nil
		}

	default:
		log.Printf("⚠️ Job %s - unknown VMA status: %s - leaving unchanged",
			job.ID, vmaStatus.Status)
		return nil
	}
}

// restartPollingForRunningJob restarts VMA progress polling for a job still running on VMA
func (pjr *ProductionJobRecovery) restartPollingForRunningJob(
	job *database.ReplicationJob,
	vmaStatus *VMAStatusResult,
) error {
	// Update job with latest VMA progress
	updates := map[string]interface{}{
		"progress_percent":  vmaStatus.Percentage,
		"vma_current_phase": vmaStatus.Phase,
		"vma_sync_type":     vmaStatus.SyncType,
		"updated_at":        time.Now(),
	}

	if err := pjr.db.GetGormDB().Model(&database.ReplicationJob{}).
		Where("id = ?", job.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	// Restart polling if poller is available
	if pjr.vmaProgressPoller != nil {
		if err := pjr.vmaProgressPoller.StartPolling(job.ID); err != nil {
			log.Printf("⚠️ Failed to restart polling for job %s: %v", job.ID, err)
			// Don't fail the recovery - job data is updated
		} else {
			log.Printf("🚀 Successfully restarted VMA progress polling for job %s", job.ID)
		}
	} else {
		log.Printf("⚠️ VMA progress poller not available - cannot restart polling for job %s", job.ID)
	}

	return nil
}

// markAsCompleted marks a job as completed with final progress
func (pjr *ProductionJobRecovery) markAsCompleted(
	job *database.ReplicationJob,
	vmaStatus *VMAStatusResult,
) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":            "completed",
		"progress_percent":  100.0,
		"current_operation": "Completed",
		"completed_at":      now,
		"updated_at":        now,
	}

	// Add VMA data if available
	if vmaStatus != nil && vmaStatus.ResponseValid {
		updates["vma_current_phase"] = vmaStatus.Phase
		updates["vma_sync_type"] = vmaStatus.SyncType
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
	// When OMA restarts, the polling map is lost, so we need to check ALL active jobs
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

// checkVMAStatus queries VMA API to determine actual job status
// Returns comprehensive status information for recovery decision making
func (pjr *ProductionJobRecovery) checkVMAStatus(ctx context.Context, job *database.ReplicationJob) (*VMAStatusResult, error) {
	if pjr.vmaClient == nil {
		return &VMAStatusResult{
			Status:        "unknown",
			IsReachable:   false,
			ResponseValid: false,
			ErrorMessage:  "VMA client not initialized",
		}, fmt.Errorf("VMA client not available")
	}

	log.Printf("🔍 Checking VMA status for job %s (%s)", job.ID, job.SourceVMName)

	// Phase 1: Try NBD export name method first (primary method)
	nbdExportNames, err := pjr.getNBDExportNamesForJob(job.ID)
	if err == nil && len(nbdExportNames) > 0 {
		log.Printf("🔗 Found %d NBD export names for job %s, trying progress API", len(nbdExportNames), job.ID)
		
		for _, exportName := range nbdExportNames {
			progress, err := pjr.vmaClient.GetProgress(exportName)
			if err == nil {
				log.Printf("✅ Got VMA response via NBD export name %s", exportName)
				return pjr.parseVMAResponse(progress, true), nil
			}
			
			// Check if it's a "not found" error
			if vmaErr, ok := err.(*VMAProgressError); ok {
				if vmaErr.StatusCode == 404 {
					log.Printf("⚠️ NBD export %s not found on VMA (404)", exportName)
					continue
				}
				// Check for HTTP 200 with "not found" message (known bug)
				if vmaErr.StatusCode == 200 && strings.Contains(strings.ToLower(vmaErr.Message), "not found") {
					log.Printf("⚠️ NBD export %s not found on VMA (HTTP 200 'not found')", exportName)
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
	progress, err := pjr.vmaClient.GetProgress(job.ID)
	if err != nil {
		// Check if it's a connection error (VMA unreachable)
		if strings.Contains(err.Error(), "connection refused") || 
		   strings.Contains(err.Error(), "no such host") ||
		   strings.Contains(err.Error(), "timeout") {
			log.Printf("❌ VMA unreachable for job %s: %v", job.ID, err)
			return &VMAStatusResult{
				Status:        "unreachable",
				IsReachable:   false,
				ResponseValid: false,
				ErrorMessage:  fmt.Sprintf("VMA API unreachable: %v", err),
			}, nil
		}

		// Check for "not found" errors
		if vmaErr, ok := err.(*VMAProgressError); ok {
			if vmaErr.StatusCode == 404 {
				log.Printf("📋 Job %s not found on VMA (404) - likely completed or lost", job.ID)
				return &VMAStatusResult{
					Status:        "not_found",
					IsReachable:   true,
					ResponseValid: true,
					ErrorMessage:  "Job not found on VMA",
				}, nil
			}
			// HTTP 200 with "not found" (known bug)
			if vmaErr.StatusCode == 200 && strings.Contains(strings.ToLower(vmaErr.Message), "not found") {
				log.Printf("📋 Job %s not found on VMA (HTTP 200 'not found') - likely completed or lost", job.ID)
				return &VMAStatusResult{
					Status:        "not_found",
					IsReachable:   true,
					ResponseValid: true,
					ErrorMessage:  "Job not found on VMA",
				}, nil
			}
		}

		log.Printf("❌ Failed to get VMA status for job %s: %v", job.ID, err)
		return &VMAStatusResult{
			Status:        "error",
			IsReachable:   true, // We got a response, just not what we expected
			ResponseValid: false,
			ErrorMessage:  fmt.Sprintf("VMA API error: %v", err),
		}, fmt.Errorf("failed to query VMA: %w", err)
	}

	log.Printf("✅ Got VMA response via job ID for %s", job.ID)
	return pjr.parseVMAResponse(progress, true), nil
}

// parseVMAResponse converts VMA progress response to recovery status result
func (pjr *ProductionJobRecovery) parseVMAResponse(progress *VMAProgressResponse, isReachable bool) *VMAStatusResult {
	result := &VMAStatusResult{
		IsReachable:   isReachable,
		ResponseValid: true,
		Percentage:    progress.Percentage,
		Phase:         progress.Phase,
		SyncType:      progress.SyncType,
	}

	// Determine status from VMA response
	switch {
	case progress.Status == "completed" || progress.Phase == "Completed" || progress.Phase == "completed":
		result.Status = "completed"
		result.Percentage = 100.0
		log.Printf("✅ VMA reports job completed (status=%s, phase=%s)", progress.Status, progress.Phase)
		
	case progress.Status == "failed" || progress.Phase == "Error" || len(progress.Errors) > 0:
		result.Status = "failed"
		if len(progress.Errors) > 0 {
			result.ErrorMessage = fmt.Sprintf("VMA reported errors: %v", progress.Errors)
		} else if progress.LastError != nil {
			result.ErrorMessage = fmt.Sprintf("VMA reported error: %v", progress.LastError)
		} else {
			result.ErrorMessage = "VMA reported failure status"
		}
		log.Printf("❌ VMA reports job failed: %s", result.ErrorMessage)
		
	case progress.Phase == "Copying Data" || progress.Phase == "Initializing" || progress.Phase == "Snapshot Creation":
		result.Status = "running"
		log.Printf("🔄 VMA reports job running (phase=%s, progress=%.1f%%)", progress.Phase, progress.Percentage)
		
	default:
		// If we have progress updates, assume it's running
		if progress.Percentage > 0 && progress.Percentage < 100 {
			result.Status = "running"
			log.Printf("🔄 VMA reports job in progress (%.1f%%)", progress.Percentage)
		} else {
			result.Status = "unknown"
			log.Printf("⚠️ VMA status unclear (status=%s, phase=%s, progress=%.1f%%)", 
				progress.Status, progress.Phase, progress.Percentage)
		}
	}

	return result
}

// getNBDExportNamesForJob queries database to construct NBD export names for a job
// This enables the primary method of querying VMA progress via NBD export names
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








