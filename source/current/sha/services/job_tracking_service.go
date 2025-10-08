package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/vexxhost/migratekit-sha/models"
)

// JobTrackingService provides centralized CloudStack async job tracking
type JobTrackingService struct {
	db     *gorm.DB
	logger *log.Logger
}

// NewJobTrackingService creates a new job tracking service instance
func NewJobTrackingService(db *gorm.DB) *JobTrackingService {
	return &JobTrackingService{
		db:     db,
		logger: log.StandardLogger(),
	}
}

// CreateJobTracking creates a new job tracking record
func (jts *JobTrackingService) CreateJobTracking(ctx context.Context, req CreateJobTrackingRequest) (*models.CloudStackJobTracking, error) {
	jobID := uuid.New().String()
	if req.CorrelationID == "" {
		req.CorrelationID = uuid.New().String()
	}

	job := &models.CloudStackJobTracking{
		ID:                jobID,
		CloudStackCommand: req.CloudStackCommand,
		OperationType:     string(req.OperationType),
		CorrelationID:     req.CorrelationID,
		ParentJobID:       req.ParentJobID,
		RequestData:       req.RequestData,
		LocalOperationID:  req.LocalOperationID,
		InitiatedBy:       req.InitiatedBy,
		Status:            string(models.JobStatusInitiated),
		CloudStackStatus:  string(models.CloudStackStatusPending),
		MaxRetries:        req.MaxRetries,
	}

	if req.MaxRetries == 0 {
		job.MaxRetries = 3 // Default retry count
	}

	err := jts.db.WithContext(ctx).Create(job).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create job tracking record: %w", err)
	}

	// Log the job creation
	jts.logJobExecution(ctx, jobID, models.LogLevelInfo, "Job tracking record created", models.JSON{
		"operation_type":     req.OperationType,
		"cloudstack_command": req.CloudStackCommand,
		"correlation_id":     req.CorrelationID,
		"initiated_by":       req.InitiatedBy,
	}, "initiation", string(models.CloudStackStatusPending))

	jts.logger.WithFields(log.Fields{
		"job_id":             jobID,
		"operation_type":     req.OperationType,
		"cloudstack_command": req.CloudStackCommand,
		"correlation_id":     req.CorrelationID,
		"parent_job_id":      req.ParentJobID,
	}).Info("🆕 Created new job tracking record")

	return job, nil
}

// UpdateJobWithCloudStackID updates a job with the CloudStack async job ID
func (jts *JobTrackingService) UpdateJobWithCloudStackID(ctx context.Context, jobID, cloudStackJobID string) error {
	now := time.Now()

	updates := map[string]interface{}{
		"cloudstack_job_id": cloudStackJobID,
		"status":            string(models.JobStatusSubmitted),
		"submitted_at":      &now,
	}

	err := jts.db.WithContext(ctx).
		Model(&models.CloudStackJobTracking{}).
		Where("id = ?", jobID).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update job with CloudStack ID: %w", err)
	}

	// Log the submission
	jts.logJobExecution(ctx, jobID, models.LogLevelInfo, "Job submitted to CloudStack", models.JSON{
		"cloudstack_job_id": cloudStackJobID,
	}, "submission", string(models.CloudStackStatusPending))

	jts.logger.WithFields(log.Fields{
		"job_id":            jobID,
		"cloudstack_job_id": cloudStackJobID,
	}).Info("✅ Updated job with CloudStack async job ID")

	return nil
}

// AddJobToPollQueue adds a job to the polling queue
func (jts *JobTrackingService) AddJobToPollQueue(ctx context.Context, jobID string, pollIntervalSeconds int) error {
	if pollIntervalSeconds == 0 {
		pollIntervalSeconds = 2 // Default 2-second polling
	}

	pollQueue := &models.CloudStackJobPollQueue{
		ID:                     uuid.New().String(),
		JobTrackingID:          jobID,
		PollIntervalSeconds:    pollIntervalSeconds,
		NextPollAt:             time.Now().Add(time.Duration(pollIntervalSeconds) * time.Second),
		ConsecutiveFailures:    0,
		MaxConsecutiveFailures: 5,
		IsActive:               true,
		Priority:               0,
	}

	err := jts.db.WithContext(ctx).Create(pollQueue).Error
	if err != nil {
		return fmt.Errorf("failed to add job to poll queue: %w", err)
	}

	// Update job status to polling
	err = jts.updateJobStatus(ctx, jobID, string(models.JobStatusPolling), nil)
	if err != nil {
		return fmt.Errorf("failed to update job status to polling: %w", err)
	}

	jts.logger.WithFields(log.Fields{
		"job_id":                jobID,
		"poll_interval_seconds": pollIntervalSeconds,
		"next_poll_at":          pollQueue.NextPollAt,
	}).Info("📋 Added job to polling queue")

	return nil
}

// UpdateJobStatus updates the job status and CloudStack response
func (jts *JobTrackingService) UpdateJobStatus(ctx context.Context, jobID string, status models.JobTrackingStatus, cloudStackStatus models.CloudStackJobStatus, response models.JSON, errorMsg *string) error {
	updates := map[string]interface{}{
		"status":            string(status),
		"cloudstack_status": string(cloudStackStatus),
	}

	if response != nil {
		updates["cloudstack_response"] = response
	}

	if errorMsg != nil {
		updates["error_message"] = *errorMsg
	}

	// Set completion time for final statuses
	if status == models.JobStatusCompleted || status == models.JobStatusFailed || status == models.JobStatusCancelled {
		now := time.Now()
		updates["completed_at"] = &now

		// Remove from poll queue
		jts.removeFromPollQueue(ctx, jobID)
	}

	err := jts.db.WithContext(ctx).
		Model(&models.CloudStackJobTracking{}).
		Where("id = ?", jobID).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Log the status update
	logLevel := models.LogLevelInfo
	if status == models.JobStatusFailed {
		logLevel = models.LogLevelError
	}

	logDetails := models.JSON{
		"new_status":        string(status),
		"cloudstack_status": string(cloudStackStatus),
	}
	if response != nil {
		logDetails["response"] = response
	}
	if errorMsg != nil {
		logDetails["error"] = *errorMsg
	}

	jts.logJobExecution(ctx, jobID, logLevel, fmt.Sprintf("Job status updated to %s", status), logDetails, "status-update", string(cloudStackStatus))

	jts.logger.WithFields(log.Fields{
		"job_id":            jobID,
		"status":            string(status),
		"cloudstack_status": string(cloudStackStatus),
		"has_error":         errorMsg != nil,
	}).Info("🔄 Updated job status")

	return nil
}

// GetJobsForPolling retrieves jobs that need to be polled
func (jts *JobTrackingService) GetJobsForPolling(ctx context.Context, limit int) ([]models.CloudStackJobTracking, error) {
	if limit == 0 {
		limit = 50 // Default limit
	}

	var jobs []models.CloudStackJobTracking

	err := jts.db.WithContext(ctx).
		Joins("JOIN cloudstack_job_poll_queue q ON q.job_tracking_id = cloudstack_job_tracking.id").
		Where("q.is_active = ? AND q.next_poll_at <= ? AND q.consecutive_failures < q.max_consecutive_failures", true, time.Now()).
		Order("q.next_poll_at ASC, q.priority DESC").
		Limit(limit).
		Find(&jobs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get jobs for polling: %w", err)
	}

	return jobs, nil
}

// UpdatePollQueue updates the polling queue for a job
func (jts *JobTrackingService) UpdatePollQueue(ctx context.Context, jobID string, consecutiveFailures int, nextPollAt time.Time) error {
	updates := map[string]interface{}{
		"consecutive_failures": consecutiveFailures,
		"next_poll_at":         nextPollAt,
	}

	err := jts.db.WithContext(ctx).
		Model(&models.CloudStackJobPollQueue{}).
		Where("job_tracking_id = ?", jobID).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update poll queue: %w", err)
	}

	return nil
}

// GetJobsByCorrelationID retrieves all jobs with the same correlation ID
func (jts *JobTrackingService) GetJobsByCorrelationID(ctx context.Context, correlationID string) ([]models.CloudStackJobTracking, error) {
	var jobs []models.CloudStackJobTracking

	err := jts.db.WithContext(ctx).
		Where("correlation_id = ?", correlationID).
		Order("created_at ASC").
		Find(&jobs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get jobs by correlation ID: %w", err)
	}

	return jobs, nil
}

// GetJobHierarchy retrieves a job and all its children
func (jts *JobTrackingService) GetJobHierarchy(ctx context.Context, rootJobID string) (*models.CloudStackJobTracking, error) {
	var rootJob models.CloudStackJobTracking

	err := jts.db.WithContext(ctx).
		Preload("ChildJobs").
		Preload("ExecutionLogs").
		Where("id = ?", rootJobID).
		First(&rootJob).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get job hierarchy: %w", err)
	}

	return &rootJob, nil
}

// Private helper methods

func (jts *JobTrackingService) updateJobStatus(ctx context.Context, jobID, status string, completedAt *time.Time) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if completedAt != nil {
		updates["completed_at"] = completedAt
	}

	return jts.db.WithContext(ctx).
		Model(&models.CloudStackJobTracking{}).
		Where("id = ?", jobID).
		Updates(updates).Error
}

func (jts *JobTrackingService) removeFromPollQueue(ctx context.Context, jobID string) error {
	return jts.db.WithContext(ctx).
		Model(&models.CloudStackJobPollQueue{}).
		Where("job_tracking_id = ?", jobID).
		Update("is_active", false).Error
}

func (jts *JobTrackingService) logJobExecution(ctx context.Context, jobID string, level models.LogLevel, message string, details models.JSON, phase, cloudStackStatus string) {
	logEntry := &models.CloudStackJobExecutionLog{
		JobTrackingID:       jobID,
		LogLevel:            string(level),
		Message:             message,
		Details:             details,
		OperationPhase:      &phase,
		CloudStackJobStatus: &cloudStackStatus,
	}

	// Don't fail the main operation if logging fails
	if err := jts.db.WithContext(ctx).Create(logEntry).Error; err != nil {
		jts.logger.WithError(err).Warn("Failed to create job execution log entry")
	}
}

// IncrementRetryCount increments the retry count for a job
func (jts *JobTrackingService) IncrementRetryCount(ctx context.Context, jobID string) error {
	err := jts.db.WithContext(ctx).
		Model(&models.CloudStackJobTracking{}).
		Where("id = ?", jobID).
		Update("retry_count", gorm.Expr("retry_count + 1")).Error

	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	return nil
}

// GetJob retrieves a single job by ID
func (jts *JobTrackingService) GetJob(ctx context.Context, jobID string) (*models.CloudStackJobTracking, error) {
	var job models.CloudStackJobTracking

	err := jts.db.WithContext(ctx).
		Where("id = ?", jobID).
		First(&job).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// Request structures

type CreateJobTrackingRequest struct {
	CloudStackCommand string               `json:"cloudstack_command" validate:"required"`
	OperationType     models.OperationType `json:"operation_type" validate:"required"`
	CorrelationID     string               `json:"correlation_id,omitempty"`
	ParentJobID       *string              `json:"parent_job_id,omitempty"`
	RequestData       models.JSON          `json:"request_data" validate:"required"`
	LocalOperationID  *string              `json:"local_operation_id,omitempty"`
	InitiatedBy       string               `json:"initiated_by" validate:"required"`
	MaxRetries        int                  `json:"max_retries,omitempty"`
}
