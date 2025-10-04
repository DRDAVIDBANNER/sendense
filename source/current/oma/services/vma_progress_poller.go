package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
)

// VMAProgressPoller manages background polling of VMA progress API
type VMAProgressPoller struct {
	vmaClient     *VMAProgressClient
	repository    *database.OSSEAConfigRepository
	pollInterval  time.Duration
	maxConcurrent int

	// Internal state
	activeJobs   map[string]*PollingContext
	jobsMutex    sync.RWMutex
	stopChan     chan struct{}
	wg           sync.WaitGroup
	isRunning    bool
	runningMutex sync.RWMutex
}

// PollingContext tracks polling state for individual jobs
type PollingContext struct {
	JobID              string
	StartedAt          time.Time
	LastPoll           time.Time
	ConsecutiveErrors  int
	MaxErrors          int
	StopChan           chan struct{}
	StartupGracePeriod time.Duration // Grace period to avoid premature failure detection
}

// NewVMAProgressPoller creates a new VMA progress poller
func NewVMAProgressPoller(vmaClient *VMAProgressClient, repository *database.OSSEAConfigRepository) *VMAProgressPoller {
	return &VMAProgressPoller{
		vmaClient:     vmaClient,
		repository:    repository,
		pollInterval:  5 * time.Second, // Poll every 5 seconds
		maxConcurrent: 10,              // Max 10 concurrent jobs
		activeJobs:    make(map[string]*PollingContext),
		stopChan:      make(chan struct{}),
	}
}

// Start begins the background polling service
func (vpp *VMAProgressPoller) Start(ctx context.Context) error {
	vpp.runningMutex.Lock()
	defer vpp.runningMutex.Unlock()

	if vpp.isRunning {
		return fmt.Errorf("VMA progress poller is already running")
	}

	vpp.isRunning = true

	log.WithFields(log.Fields{
		"poll_interval":  vpp.pollInterval,
		"max_concurrent": vpp.maxConcurrent,
	}).Info("🚀 Starting VMA progress poller")

	// Start the main polling loop
	vpp.wg.Add(1)
	go vpp.pollingLoop(ctx)

	return nil
}

// Stop gracefully stops the polling service
func (vpp *VMAProgressPoller) Stop() error {
	vpp.runningMutex.Lock()
	defer vpp.runningMutex.Unlock()

	if !vpp.isRunning {
		return fmt.Errorf("VMA progress poller is not running")
	}

	log.Info("🛑 Stopping VMA progress poller")

	// Stop all individual job polling
	vpp.jobsMutex.Lock()
	for jobID, ctx := range vpp.activeJobs {
		close(ctx.StopChan)
		log.WithField("job_id", jobID).Debug("Stopping job polling")
	}
	vpp.jobsMutex.Unlock()

	// Stop main loop
	close(vpp.stopChan)
	vpp.wg.Wait()

	vpp.isRunning = false
	return nil
}

// StartPolling begins polling for a specific job
func (vpp *VMAProgressPoller) StartPolling(jobID string) error {
	vpp.jobsMutex.Lock()
	defer vpp.jobsMutex.Unlock()

	// Check if already polling
	if _, exists := vpp.activeJobs[jobID]; exists {
		log.WithField("job_id", jobID).Debug("Already polling job")
		return nil
	}

	// Check concurrent limit
	if len(vpp.activeJobs) >= vpp.maxConcurrent {
		return fmt.Errorf("max concurrent polling jobs reached (%d)", vpp.maxConcurrent)
	}

	// Create polling context with startup grace period
	pollingCtx := &PollingContext{
		JobID:              jobID,
		StartedAt:          time.Now(),
		MaxErrors:          5, // Stop after 5 consecutive errors
		StopChan:           make(chan struct{}),
		StartupGracePeriod: 30 * time.Second, // Wait 30 seconds before assuming failure
	}

	vpp.activeJobs[jobID] = pollingCtx

	log.WithField("job_id", jobID).Info("📋 Started VMA progress polling")

	return nil
}

// StopPolling stops polling for a specific job
func (vpp *VMAProgressPoller) StopPolling(jobID string) error {
	vpp.jobsMutex.Lock()
	defer vpp.jobsMutex.Unlock()

	pollingCtx, exists := vpp.activeJobs[jobID]
	if !exists {
		return fmt.Errorf("job %s is not being polled", jobID)
	}

	close(pollingCtx.StopChan)
	delete(vpp.activeJobs, jobID)

	log.WithField("job_id", jobID).Info("🛑 Stopped VMA progress polling")

	return nil
}

// pollingLoop is the main background polling loop
func (vpp *VMAProgressPoller) pollingLoop(ctx context.Context) {
	defer vpp.wg.Done()

	ticker := time.NewTicker(vpp.pollInterval)
	defer ticker.Stop()

	log.WithField("poll_interval", vpp.pollInterval).Info("📋 VMA progress polling loop started")

	for {
		select {
		case <-ctx.Done():
			log.Info("📋 VMA progress poller stopped due to context cancellation")
			return
		case <-vpp.stopChan:
			log.Info("📋 VMA progress poller stopped")
			return
		case <-ticker.C:
			log.Debug("🔍 Polling ticker fired")
			vpp.pollAllActiveJobs()
		}
	}
}

// pollAllActiveJobs polls progress for all active jobs
func (vpp *VMAProgressPoller) pollAllActiveJobs() {
	vpp.jobsMutex.RLock()
	activeJobsCopy := make(map[string]*PollingContext)
	for k, v := range vpp.activeJobs {
		activeJobsCopy[k] = v
	}
	vpp.jobsMutex.RUnlock()

	log.WithField("active_jobs", len(activeJobsCopy)).Debug("🔍 pollAllActiveJobs called")

	if len(activeJobsCopy) == 0 {
		log.Debug("📋 No active jobs to poll")
		return
	}

	log.WithField("active_jobs", len(activeJobsCopy)).Debug("🔍 Polling VMA progress for active jobs")

	// Use semaphore to limit concurrent API calls
	semaphore := make(chan struct{}, vpp.maxConcurrent)

	for jobID, pollingCtx := range activeJobsCopy {
		semaphore <- struct{}{} // Acquire

		go func(jobID string, ctx *PollingContext) {
			defer func() { <-semaphore }() // Release

			vpp.pollSingleJob(jobID, ctx)
		}(jobID, pollingCtx)
	}
}

// pollSingleJob polls progress for a single job
func (vpp *VMAProgressPoller) pollSingleJob(jobID string, pollingCtx *PollingContext) {
	logger := log.WithField("job_id", jobID)

	// Update last poll time
	pollingCtx.LastPoll = time.Now()

	// Phase 1 Fix: Try NBD export names first
	nbdExportNames, err := vpp.getNBDExportNameForJob(jobID)
	if err == nil && len(nbdExportNames) > 0 {
		logger.WithField("nbd_export_count", len(nbdExportNames)).Debug("🔗 Trying NBD export names for progress")

		for _, nbdExportName := range nbdExportNames {
			progressData, err := vpp.vmaClient.GetProgress(nbdExportName)
			if err == nil {
				logger.WithField("nbd_export_name", nbdExportName).Info("✅ Found progress via NBD export name")
				pollingCtx.ConsecutiveErrors = 0

				// Update database with VMA progress
				if err := vpp.updateJobWithVMAData(jobID, progressData); err != nil {
					logger.WithError(err).Warn("Failed to update job with VMA data")
					return
				}

				// Check if job is complete
				if progressData.Phase == "completed" || progressData.Status == "completed" || progressData.Status == "failed" {
					logger.WithField("final_status", progressData.Status).Info("✅ Job completed - stopping polling")
					vpp.StopPolling(jobID)
				}

				logger.WithFields(log.Fields{
					"job_id":           jobID,
					"nbd_export_name":  nbdExportName,
					"progress_percent": progressData.Percentage,
					"current_phase":    progressData.Phase,
					"throughput_mbps":  progressData.Throughput.CurrentMBps,
				}).Debug("✅ Successfully updated job progress from VMA via NBD export name")
				return
			}
			logger.WithFields(log.Fields{
				"nbd_export_name": nbdExportName,
				"error":           err.Error(),
			}).Debug("⚠️ NBD export name failed, trying next")
		}
		logger.Debug("⚠️ All NBD export names failed, falling back to job ID")
	} else if err != nil {
		logger.WithError(err).Debug("⚠️ Failed to get NBD export names, falling back to job ID")
	}

	// Fallback: Get progress from VMA using traditional job ID
	logger.Debug("🔄 Trying traditional job ID for progress")
	progressData, err := vpp.vmaClient.GetProgress(jobID)
	if err != nil {
		vpp.handlePollingError(jobID, pollingCtx, err, logger)
		return
	}

	// Reset error count on success
	pollingCtx.ConsecutiveErrors = 0

	// Update database with VMA progress
	if err := vpp.updateJobWithVMAData(jobID, progressData); err != nil {
		logger.WithError(err).Warn("Failed to update job with VMA data")
		return
	}

	// Check if job is complete
	if progressData.Phase == "completed" || progressData.Status == "completed" || progressData.Status == "failed" {
		logger.WithField("final_status", progressData.Status).Info("✅ Job completed - stopping polling")
		vpp.StopPolling(jobID)
	}

	logger.WithFields(log.Fields{
		"job_id":           jobID,
		"progress_percent": progressData.Percentage,
		"current_phase":    progressData.Phase,
		"throughput_mbps":  progressData.Throughput.CurrentMBps,
	}).Debug("✅ Successfully updated job progress from VMA via job ID (legacy)")
}

// handlePollingError handles errors during polling
func (vpp *VMAProgressPoller) handlePollingError(jobID string, pollingCtx *PollingContext, err error, logger *log.Entry) {
	pollingCtx.ConsecutiveErrors++

	// Check if it's a "job not found" error - could be startup phase or completion
	if vmaErr, ok := err.(*VMAProgressError); ok && vmaErr.StatusCode == 404 {
		jobAge := time.Since(pollingCtx.StartedAt)

		// During startup grace period, don't assume completion
		if jobAge < pollingCtx.StartupGracePeriod {
			logger.WithField("job_age", jobAge).Debug("Job not found during startup grace period - continuing to poll")
			return // Continue polling during grace period
		}

		// After grace period, assume completion
		logger.WithField("job_age", jobAge).Info("📋 Job not found in VMA after grace period - likely completed")
		vpp.StopPolling(jobID)
		return
	}

	logger.WithError(err).WithField("consecutive_errors", pollingCtx.ConsecutiveErrors).Warn("⚠️ VMA polling error")

	// Stop polling if too many consecutive errors
	if pollingCtx.ConsecutiveErrors >= pollingCtx.MaxErrors {
		logger.WithField("max_errors", pollingCtx.MaxErrors).Error("❌ Max polling errors reached - stopping polling")
		vpp.StopPolling(jobID)

		// Mark job as failed
		vpp.repository.UpdateReplicationJob(jobID, map[string]interface{}{
			"status":                   "failed",
			"error_message":            fmt.Sprintf("VMA polling failed: %v", err),
			"vma_error_classification": "polling",
			"vma_error_details":        err.Error(),
			"completed_at":             time.Now(),
		})

		// 🎯 NEW: Update VM context after polling failure
		go func() {
			if err := vpp.repository.UpdateVMContextAfterJobCompletion(jobID); err != nil {
				log.WithError(err).WithField("job_id", jobID).Error("Failed to update VM context after polling failure")
			}
		}()
	}
}

// updateJobWithVMAData updates the database with VMA progress data
func (vpp *VMAProgressPoller) updateJobWithVMAData(jobID string, vmaData *VMAProgressResponse) error {
	updates := map[string]interface{}{
		"status":              MapVMAPhaseToStatus(vmaData.Phase),
		"progress_percent":    vmaData.Percentage,
		"current_operation":   vmaData.CurrentOperation,
		"bytes_transferred":   vmaData.BytesTransferred,
		"total_bytes":         vmaData.TotalBytes,
		"transfer_speed_bps":  ConvertThroughputToBps(vmaData.Throughput.CurrentMBps),
		"vma_sync_type":       vmaData.SyncType,
		"vma_current_phase":   vmaData.Phase,
		"vma_throughput_mbps": vmaData.Throughput.CurrentMBps,
		"vma_eta_seconds":     &vmaData.Timing.ETASeconds,
		"vma_last_poll_at":    time.Now(),
		"updated_at":          time.Now(),
	}

	// 🎯 UPDATE REPLICATION TYPE: Update replication_type based on VMA detected sync type
	if vmaData.SyncType != "" {
		updates["replication_type"] = mapVMASyncTypeToReplicationType(vmaData.SyncType)
	}

	// Handle completion
	if vmaData.Status == "completed" || vmaData.Phase == "completed" {
		updates["completed_at"] = time.Now()
		updates["progress_percent"] = 100.0
		updates["current_operation"] = "Completed" // 🎯 FIX: Update current_operation to Completed
		updates["status"] = "completed"            // 🚨 CRITICAL FIX: Update status to completed
	}

	// Handle errors - check for errors array or failed status
	if len(vmaData.Errors) > 0 || vmaData.Status == "failed" {
		var errorMessage string
		if len(vmaData.Errors) > 0 {
			errorJSON, _ := json.Marshal(vmaData.Errors)
			errorMessage = string(errorJSON)
		} else {
			errorMessage = "Job failed"
		}

		updates["error_message"] = errorMessage
		updates["vma_error_classification"] = "migration_error"
		if vmaData.LastError != nil {
			lastErrorJSON, _ := json.Marshal(vmaData.LastError)
			updates["vma_error_details"] = string(lastErrorJSON)
		}

		if vmaData.Status == "failed" {
			updates["status"] = "failed"
			updates["completed_at"] = time.Now()
			updates["current_operation"] = "Failed" // 🎯 FIX: Update current_operation to Failed
		}
	}

	// Update the job status first
	err := vpp.repository.UpdateReplicationJob(jobID, updates)
	if err != nil {
		return err
	}

	// 🎯 CRITICAL FIX: Update VM context AFTER job status is updated (fixes race condition)
	// Only update VM context if job is completed or failed
	if (vmaData.Status == "completed" || vmaData.Phase == "completed") || vmaData.Status == "failed" {
		if err := vpp.repository.UpdateVMContextAfterJobCompletion(jobID); err != nil {
			log.WithError(err).WithField("job_id", jobID).Error("Failed to update VM context after job completion")
			// Don't return error here - job update succeeded, VM context update is secondary
		}
	}

	return nil
}

// GetPollingStatus returns current polling status
func (vpp *VMAProgressPoller) GetPollingStatus() map[string]interface{} {
	vpp.jobsMutex.RLock()
	defer vpp.jobsMutex.RUnlock()

	vpp.runningMutex.RLock()
	defer vpp.runningMutex.RUnlock()

	activeJobs := make([]map[string]interface{}, 0, len(vpp.activeJobs))
	for jobID, ctx := range vpp.activeJobs {
		activeJobs = append(activeJobs, map[string]interface{}{
			"job_id":             jobID,
			"started_at":         ctx.StartedAt,
			"last_poll":          ctx.LastPoll,
			"consecutive_errors": ctx.ConsecutiveErrors,
		})
	}

	return map[string]interface{}{
		"is_running":            vpp.isRunning,
		"poll_interval_seconds": int(vpp.pollInterval.Seconds()),
		"max_concurrent":        vpp.maxConcurrent,
		"active_jobs_count":     len(vpp.activeJobs),
		"active_jobs":           activeJobs,
		"vma_api_healthy":       vpp.vmaClient.IsHealthy(),
	}
}

// getNBDExportNameForJob constructs NBD export names from OMA job ID
// Phase 1 fix: Maps job ID to NBD export names via database relationships
func (vpp *VMAProgressPoller) getNBDExportNameForJob(jobID string) ([]string, error) {
	logger := log.WithField("job_id", jobID)

	// Execute the database query using the repository method
	volumeIDs, err := vpp.repository.GetVolumeIDsForJob(jobID)
	if err != nil {
		logger.WithError(err).Error("Failed to get volume IDs for job")
		return nil, fmt.Errorf("failed to get volume IDs for job %s: %w", jobID, err)
	}

	if len(volumeIDs) == 0 {
		logger.Debug("No volume IDs found for job")
		return []string{}, nil
	}

	// Construct NBD export names from volume UUIDs
	var nbdExportNames []string
	for _, volumeID := range volumeIDs {
		nbdExportName := fmt.Sprintf("migration-vol-%s", volumeID)
		nbdExportNames = append(nbdExportNames, nbdExportName)
	}

	logger.WithFields(log.Fields{
		"volume_count":     len(volumeIDs),
		"nbd_export_names": nbdExportNames,
	}).Debug("🔗 Constructed NBD export names from job ID via database query")

	return nbdExportNames, nil
}

// mapVMASyncTypeToReplicationType maps VMA sync type to database replication type
func mapVMASyncTypeToReplicationType(vmaSyncType string) string {
	switch strings.ToLower(vmaSyncType) {
	case "incremental":
		return "incremental"
	case "full", "initial":
		return "initial"
	default:
		// Default to initial if unknown
		return "initial"
	}
}
