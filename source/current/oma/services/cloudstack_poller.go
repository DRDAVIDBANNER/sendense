package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/vexxhost/migratekit-oma/models"
	"github.com/vexxhost/migratekit-oma/ossea"
)

// CloudStackPoller manages polling of CloudStack async jobs
type CloudStackPoller struct {
	db                 *gorm.DB
	jobTrackingService *JobTrackingService
	osseaClient        *ossea.Client
	logger             *log.Logger

	// Polling configuration
	pollInterval       time.Duration
	maxConcurrentPolls int
	isRunning          bool
	stopChan           chan struct{}
	wg                 sync.WaitGroup
	mutex              sync.RWMutex
}

// NewCloudStackPoller creates a new CloudStack job poller
func NewCloudStackPoller(db *gorm.DB, jobTrackingService *JobTrackingService, osseaClient *ossea.Client) *CloudStackPoller {
	return &CloudStackPoller{
		db:                 db,
		jobTrackingService: jobTrackingService,
		osseaClient:        osseaClient,
		logger:             log.StandardLogger(),
		pollInterval:       2 * time.Second, // Default 2-second polling
		maxConcurrentPolls: 10,              // Maximum concurrent polls
		stopChan:           make(chan struct{}),
	}
}

// Start begins the polling process
func (csp *CloudStackPoller) Start(ctx context.Context) error {
	csp.mutex.Lock()
	defer csp.mutex.Unlock()

	if csp.isRunning {
		return fmt.Errorf("poller is already running")
	}

	csp.isRunning = true
	csp.logger.Info("🚀 Starting CloudStack async job poller")

	// Start the main polling loop
	csp.wg.Add(1)
	go csp.pollLoop(ctx)

	return nil
}

// Stop gracefully stops the polling process
func (csp *CloudStackPoller) Stop() error {
	csp.mutex.Lock()
	defer csp.mutex.Unlock()

	if !csp.isRunning {
		return fmt.Errorf("poller is not running")
	}

	csp.logger.Info("🛑 Stopping CloudStack async job poller")
	close(csp.stopChan)
	csp.wg.Wait()
	csp.isRunning = false

	return nil
}

// IsRunning returns whether the poller is currently running
func (csp *CloudStackPoller) IsRunning() bool {
	csp.mutex.RLock()
	defer csp.mutex.RUnlock()
	return csp.isRunning
}

// pollLoop is the main polling loop that runs continuously
func (csp *CloudStackPoller) pollLoop(ctx context.Context) {
	defer csp.wg.Done()

	ticker := time.NewTicker(csp.pollInterval)
	defer ticker.Stop()

	csp.logger.WithFields(log.Fields{
		"poll_interval":        csp.pollInterval,
		"max_concurrent_polls": csp.maxConcurrentPolls,
	}).Info("📋 CloudStack job poller started")

	for {
		select {
		case <-ctx.Done():
			csp.logger.Info("📋 CloudStack job poller stopped due to context cancellation")
			return
		case <-csp.stopChan:
			csp.logger.Info("📋 CloudStack job poller stopped")
			return
		case <-ticker.C:
			csp.processPendingJobs(ctx)
		}
	}
}

// processPendingJobs retrieves and processes jobs that need polling
func (csp *CloudStackPoller) processPendingJobs(ctx context.Context) {
	jobs, err := csp.jobTrackingService.GetJobsForPolling(ctx, csp.maxConcurrentPolls)
	if err != nil {
		csp.logger.WithError(err).Error("❌ Failed to get jobs for polling")
		return
	}

	if len(jobs) == 0 {
		return
	}

	csp.logger.WithField("job_count", len(jobs)).Debug("📋 Processing pending jobs")

	// Process jobs concurrently with a semaphore to limit concurrency
	semaphore := make(chan struct{}, csp.maxConcurrentPolls)
	var wg sync.WaitGroup

	for _, job := range jobs {
		wg.Add(1)
		go func(j models.CloudStackJobTracking) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			csp.pollSingleJob(ctx, j)
		}(job)
	}

	wg.Wait()
}

// pollSingleJob polls a single CloudStack job for completion
func (csp *CloudStackPoller) pollSingleJob(ctx context.Context, job models.CloudStackJobTracking) {
	if job.CloudStackJobID == nil {
		csp.logger.WithField("job_id", job.ID).Error("❌ Job has no CloudStack job ID")
		csp.markJobAsFailed(ctx, job.ID, "No CloudStack job ID available")
		return
	}

	logger := csp.logger.WithFields(log.Fields{
		"job_id":            job.ID,
		"cloudstack_job_id": *job.CloudStackJobID,
		"operation_type":    job.OperationType,
		"correlation_id":    job.CorrelationID,
	})

	logger.Debug("🔍 Polling CloudStack job")

	// Query CloudStack for job status
	jobResult, err := csp.osseaClient.QueryAsyncJobResult(*job.CloudStackJobID)
	if err != nil {
		logger.WithError(err).Error("❌ Failed to query CloudStack job")
		csp.handlePollingError(ctx, job.ID, err)
		return
	}

	// Process the job result
	csp.processJobResult(ctx, job, jobResult, logger)
}

// processJobResult processes the CloudStack job result and updates the tracking record
func (csp *CloudStackPoller) processJobResult(ctx context.Context, job models.CloudStackJobTracking, result *ossea.AsyncJobResult, logger *log.Entry) {
	// Convert CloudStack status to our enum
	var cloudStackStatus models.CloudStackJobStatus
	var trackingStatus models.JobTrackingStatus
	var errorMessage *string

	switch result.JobStatus {
	case 0: // Pending
		cloudStackStatus = models.CloudStackStatusPending
		trackingStatus = models.JobStatusPolling
		// Update next poll time
		nextPollAt := time.Now().Add(csp.pollInterval)
		csp.jobTrackingService.UpdatePollQueue(ctx, job.ID, 0, nextPollAt)
		logger.Debug("⏳ Job still pending")
		return

	case 1: // In Progress
		cloudStackStatus = models.CloudStackStatusInProgress
		trackingStatus = models.JobStatusPolling
		// Update next poll time
		nextPollAt := time.Now().Add(csp.pollInterval)
		csp.jobTrackingService.UpdatePollQueue(ctx, job.ID, 0, nextPollAt)
		logger.Debug("🔄 Job in progress")
		return

	case 2: // Success
		cloudStackStatus = models.CloudStackStatusSuccess
		trackingStatus = models.JobStatusCompleted
		logger.Info("✅ Job completed successfully")

	case 3: // Failure
		cloudStackStatus = models.CloudStackStatusFailure
		trackingStatus = models.JobStatusFailed
		if result.JobResultCode != nil {
			errorMsg := fmt.Sprintf("CloudStack job failed with code %d", *result.JobResultCode)
			if result.JobResult != nil {
				if errorText, ok := result.JobResult["errortext"]; ok {
					errorMsg = fmt.Sprintf("%s: %v", errorMsg, errorText)
				}
			}
			errorMessage = &errorMsg
		}
		logger.WithField("result_code", result.JobResultCode).Error("❌ Job failed")

	default:
		// Unknown status - treat as failure
		cloudStackStatus = models.CloudStackStatusFailure
		trackingStatus = models.JobStatusFailed
		errorMsg := fmt.Sprintf("Unknown CloudStack job status: %d", result.JobStatus)
		errorMessage = &errorMsg
		logger.WithField("job_status", result.JobStatus).Error("❌ Unknown job status")
	}

	// Convert result to JSON
	var responseJSON models.JSON
	if result.JobResult != nil {
		responseJSON = models.JSON(result.JobResult)
	}

	// Update the job status
	err := csp.jobTrackingService.UpdateJobStatus(
		ctx,
		job.ID,
		trackingStatus,
		cloudStackStatus,
		responseJSON,
		errorMessage,
	)

	if err != nil {
		logger.WithError(err).Error("❌ Failed to update job status")
		return
	}

	// If this was a parent job and it completed, check if all children are complete
	if job.ParentJobID == nil && trackingStatus == models.JobStatusCompleted {
		csp.checkChildJobsCompletion(ctx, job.ID, job.CorrelationID, logger)
	}
}

// handlePollingError handles errors that occur during polling
func (csp *CloudStackPoller) handlePollingError(ctx context.Context, jobID string, err error) {
	// Get current job to check retry count
	job, getErr := csp.jobTrackingService.GetJob(ctx, jobID)
	if getErr != nil {
		csp.logger.WithError(getErr).Error("❌ Failed to get job for error handling")
		return
	}

	// Increment consecutive failures in poll queue
	nextPollAt := time.Now().Add(csp.pollInterval * 2) // Double the interval on error
	updateErr := csp.jobTrackingService.UpdatePollQueue(ctx, jobID, job.RetryCount+1, nextPollAt)
	if updateErr != nil {
		csp.logger.WithError(updateErr).Error("❌ Failed to update poll queue after error")
	}

	// If we've exceeded max retries, mark the job as failed
	if !job.ShouldRetry() {
		errorMsg := fmt.Sprintf("Max polling retries exceeded: %v", err)
		csp.markJobAsFailed(ctx, jobID, errorMsg)
		return
	}

	// Increment retry count
	if retryErr := csp.jobTrackingService.IncrementRetryCount(ctx, jobID); retryErr != nil {
		csp.logger.WithError(retryErr).Error("❌ Failed to increment retry count")
	}

	csp.logger.WithFields(log.Fields{
		"job_id":      jobID,
		"retry_count": job.RetryCount + 1,
		"max_retries": job.MaxRetries,
		"error":       err,
	}).Warn("⚠️ Polling error, will retry")
}

// markJobAsFailed marks a job as failed with an error message
func (csp *CloudStackPoller) markJobAsFailed(ctx context.Context, jobID, errorMsg string) {
	err := csp.jobTrackingService.UpdateJobStatus(
		ctx,
		jobID,
		models.JobStatusFailed,
		models.CloudStackStatusFailure,
		nil,
		&errorMsg,
	)

	if err != nil {
		csp.logger.WithError(err).Error("❌ Failed to mark job as failed")
	}
}

// checkChildJobsCompletion checks if all child jobs in a correlation group are complete
func (csp *CloudStackPoller) checkChildJobsCompletion(ctx context.Context, parentJobID, correlationID string, logger *log.Entry) {
	jobs, err := csp.jobTrackingService.GetJobsByCorrelationID(ctx, correlationID)
	if err != nil {
		logger.WithError(err).Error("❌ Failed to get jobs by correlation ID")
		return
	}

	allComplete := true
	anyFailed := false

	for _, job := range jobs {
		if !job.IsCompleted() {
			allComplete = false
			break
		}
		if job.Status == string(models.JobStatusFailed) {
			anyFailed = true
		}
	}

	if allComplete {
		status := "success"
		if anyFailed {
			status = "partial_failure"
		}

		logger.WithFields(log.Fields{
			"correlation_id": correlationID,
			"total_jobs":     len(jobs),
			"final_status":   status,
		}).Info("🎯 All jobs in correlation group completed")

		// Could trigger completion callbacks here
		csp.handleCorrelationGroupCompletion(ctx, correlationID, status, jobs)
	}
}

// handleCorrelationGroupCompletion handles the completion of an entire correlation group
func (csp *CloudStackPoller) handleCorrelationGroupCompletion(ctx context.Context, correlationID, status string, jobs []models.CloudStackJobTracking) {
	// This is where we could trigger webhooks, notifications, or other completion logic
	// For now, just log the completion

	completedCount := 0
	failedCount := 0

	for _, job := range jobs {
		switch job.Status {
		case string(models.JobStatusCompleted):
			completedCount++
		case string(models.JobStatusFailed):
			failedCount++
		}
	}

	csp.logger.WithFields(log.Fields{
		"correlation_id": correlationID,
		"total_jobs":     len(jobs),
		"completed_jobs": completedCount,
		"failed_jobs":    failedCount,
		"overall_status": status,
	}).Info("🎉 Correlation group completed")

	// Record metrics
	csp.recordCompletionMetrics(ctx, jobs)
}

// recordCompletionMetrics records metrics for completed job groups
func (csp *CloudStackPoller) recordCompletionMetrics(ctx context.Context, jobs []models.CloudStackJobTracking) {
	if len(jobs) == 0 {
		return
	}

	// Calculate metrics
	var totalDuration time.Duration
	operationCounts := make(map[string]int)

	for _, job := range jobs {
		if duration := job.Duration(); duration != nil {
			totalDuration += *duration
		}
		operationCounts[job.OperationType]++
	}

	avgDuration := float64(totalDuration.Seconds()) / float64(len(jobs))

	// Create metrics record
	windowStart := jobs[0].InitiatedAt
	windowEnd := time.Now()

	// Convert operationCounts to JSON format
	var operationsByType models.JSON
	if len(operationCounts) > 0 {
		operationsByType = make(models.JSON)
		for k, v := range operationCounts {
			operationsByType[k] = v
		}
	}

	metrics := &models.CloudStackJobMetrics{
		WindowStart:                  windowStart,
		WindowEnd:                    windowEnd,
		TotalJobsInitiated:           len(jobs),
		TotalJobsCompleted:           len(jobs), // All jobs in this group are complete
		AverageCompletionTimeSeconds: avgDuration,
		OperationsByType:             operationsByType,
	}

	if err := csp.db.WithContext(ctx).Create(metrics).Error; err != nil {
		csp.logger.WithError(err).Warn("Failed to record job completion metrics")
	}
}

// GetPollerStatus returns the current status of the poller
func (csp *CloudStackPoller) GetPollerStatus() map[string]interface{} {
	csp.mutex.RLock()
	defer csp.mutex.RUnlock()

	return map[string]interface{}{
		"is_running":            csp.isRunning,
		"poll_interval_seconds": int(csp.pollInterval.Seconds()),
		"max_concurrent_polls":  csp.maxConcurrentPolls,
	}
}
