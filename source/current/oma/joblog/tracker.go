package joblog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Tracker provides job and step lifecycle management with integrated logging
type Tracker struct {
	db      *sql.DB
	logger  *slog.Logger
	handler slog.Handler
	mu      sync.RWMutex
}

// New creates a new job tracker with the given database and log handlers
func New(db *sql.DB, handlers ...slog.Handler) *Tracker {
	var handler slog.Handler

	if len(handlers) == 0 {
		// Default to JSON handler to stdout
		handler = slog.NewJSONHandler(nil, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else if len(handlers) == 1 {
		handler = handlers[0]
	} else {
		// Multiple handlers - use fanout
		handler = NewFanoutHandler(handlers...)
	}

	logger := slog.New(handler)

	return &Tracker{
		db:      db,
		logger:  logger,
		handler: handler,
	}
}

// StartJob creates a new job and returns a context with the job ID
func (t *Tracker) StartJob(ctx context.Context, input JobStart) (context.Context, string, error) {
	if err := input.Validate(); err != nil {
		return ctx, "", fmt.Errorf("invalid job start input: %w", err)
	}

	jobID := uuid.New().String()
	now := time.Now()

	// Serialize metadata if provided
	var metadataJSON *string
	if input.Metadata != nil {
		if jsonBytes, err := json.Marshal(input.Metadata); err == nil {
			jsonStr := string(jsonBytes)
			metadataJSON = &jsonStr
		}
	}

	// Insert job record with enhanced fields
	query := `
		INSERT INTO job_tracking (
			id, parent_job_id, job_type, operation, status, 
			metadata, owner, started_at, created_at, updated_at,
			context_id, external_job_id, job_category
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := t.db.ExecContext(ctx, query,
		jobID,
		input.ParentJobID,
		input.JobType,
		input.Operation,
		StatusRunning,
		metadataJSON,
		input.Owner,
		now,
		now,
		now,
		input.ContextID,     // Enhanced field
		input.ExternalJobID, // Enhanced field
		input.JobCategory,   // Enhanced field
	)

	if err != nil {
		return ctx, "", fmt.Errorf("failed to create job record: %w", err)
	}

	// Add job ID to context
	ctxWithJob := WithJobID(ctx, jobID)

	// Add external job ID to context if provided (for log correlation)
	if input.ExternalJobID != nil && *input.ExternalJobID != "" {
		ctxWithJob = WithExternalJobID(ctxWithJob, *input.ExternalJobID)
	}

	// Log job start
	logger := t.Logger(ctxWithJob)
	logger.Info("Job started",
		slog.String("job_id", jobID),
		slog.String("job_type", input.JobType),
		slog.String("operation", input.Operation),
		slog.Any("parent_job_id", input.ParentJobID),
		slog.Any("owner", input.Owner),
	)

	return ctxWithJob, jobID, nil
}

// EndJob completes a job with the given status and optional error
func (t *Tracker) EndJob(ctx context.Context, jobID string, status Status, err error) error {
	now := time.Now()

	var errorMessage *string
	if err != nil {
		errStr := err.Error()
		errorMessage = &errStr
	}

	var completedAt *time.Time
	var canceledAt *time.Time

	if status.IsTerminal() {
		completedAt = &now
		if status == StatusCancelled {
			canceledAt = &now
		}
	}

	// Update job record
	query := `
		UPDATE job_tracking 
		SET status = ?, completed_at = ?, canceled_at = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := t.db.ExecContext(ctx, query,
		status,
		completedAt,
		canceledAt,
		errorMessage,
		now,
		jobID,
	)

	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Log job completion
	logger := t.Logger(WithJobID(ctx, jobID))
	logLevel := slog.LevelInfo
	message := "Job completed"

	if status == StatusFailed {
		logLevel = slog.LevelError
		message = "Job failed"
	} else if status == StatusCancelled {
		logLevel = slog.LevelWarn
		message = "Job cancelled"
	}

	logger.Log(ctx, logLevel, message,
		slog.String("job_id", jobID),
		slog.String("status", string(status)),
		slog.Any("error", errorMessage),
	)

	return nil
}

// MarkJobProgress updates the percent completion of a job
func (t *Tracker) MarkJobProgress(ctx context.Context, jobID string, percent uint8) error {
	if percent > 100 {
		return fmt.Errorf("invalid percentage: %d (must be 0-100)", percent)
	}

	query := `
		UPDATE job_tracking 
		SET percent_complete = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := t.db.ExecContext(ctx, query, percent, time.Now(), jobID)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Log progress update
	logger := t.Logger(WithJobID(ctx, jobID))
	logger.Debug("Job progress updated",
		slog.String("job_id", jobID),
		slog.Int("percent", int(percent)),
	)

	return nil
}

// StartStep creates a new step within a job
func (t *Tracker) StartStep(ctx context.Context, jobID string, input StepStart) (context.Context, int64, error) {
	if err := input.Validate(); err != nil {
		return ctx, 0, fmt.Errorf("invalid step start input: %w", err)
	}

	// Auto-generate sequence number if not provided
	if input.Seq == 0 {
		seq, err := t.getNextStepSequence(ctx, jobID)
		if err != nil {
			return ctx, 0, fmt.Errorf("failed to generate step sequence: %w", err)
		}
		input.Seq = seq
	}

	now := time.Now()

	// Serialize metadata if provided
	var metadataJSON *string
	if input.Metadata != nil {
		if jsonBytes, err := json.Marshal(input.Metadata); err == nil {
			jsonStr := string(jsonBytes)
			metadataJSON = &jsonStr
		}
	}

	// Insert step record
	query := `
		INSERT INTO job_steps (job_id, name, seq, status, started_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := t.db.ExecContext(ctx, query,
		jobID,
		input.Name,
		input.Seq,
		StatusRunning,
		now,
		metadataJSON,
	)

	if err != nil {
		return ctx, 0, fmt.Errorf("failed to create step record: %w", err)
	}

	stepID, err := result.LastInsertId()
	if err != nil {
		return ctx, 0, fmt.Errorf("failed to get step ID: %w", err)
	}

	// Add step ID to context
	ctxWithStep := WithStepID(WithJobID(ctx, jobID), stepID)

	// Log step start
	logger := t.Logger(ctxWithStep)
	logger.Info("Step started",
		slog.String("job_id", jobID),
		slog.Int64("step_id", stepID),
		slog.String("step_name", input.Name),
		slog.Int("sequence", input.Seq),
	)

	return ctxWithStep, stepID, nil
}

// EndStep completes a step with the given status and optional error
func (t *Tracker) EndStep(stepID int64, status Status, err error) error {
	now := time.Now()

	var errorMessage *string
	if err != nil {
		errStr := err.Error()
		errorMessage = &errStr
	}

	var completedAt *time.Time
	if status.IsTerminal() || status == StatusSkipped {
		completedAt = &now
	}

	// Update step record
	query := `
		UPDATE job_steps 
		SET status = ?, completed_at = ?, error_message = ?
		WHERE id = ?
	`

	result, err := t.db.ExecContext(context.Background(), query,
		status,
		completedAt,
		errorMessage,
		stepID,
	)

	if err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("step not found: %d", stepID)
	}

	// Get step details for logging
	var jobID string
	var stepName string
	query = `SELECT job_id, name FROM job_steps WHERE id = ?`
	err = t.db.QueryRowContext(context.Background(), query, stepID).Scan(&jobID, &stepName)
	if err == nil {
		// Log step completion
		ctx := WithStepID(WithJobID(context.Background(), jobID), stepID)
		logger := t.Logger(ctx)

		logLevel := slog.LevelInfo
		message := "Step completed"

		if status == StatusFailed {
			logLevel = slog.LevelError
			message = "Step failed"
		} else if status == StatusSkipped {
			logLevel = slog.LevelWarn
			message = "Step skipped"
		}

		logger.Log(context.Background(), logLevel, message,
			slog.String("job_id", jobID),
			slog.Int64("step_id", stepID),
			slog.String("step_name", stepName),
			slog.String("status", string(status)),
			slog.Any("error", errorMessage),
		)
	}

	return nil
}

// RunStep automatically manages step lifecycle and handles panics
func (t *Tracker) RunStep(ctx context.Context, jobID string, name string, fn func(ctx context.Context) error) error {
	stepInput := StepStart{Name: name}

	// Start the step
	stepCtx, stepID, err := t.StartStep(ctx, jobID, stepInput)
	if err != nil {
		return fmt.Errorf("failed to start step: %w", err)
	}

	// Set up panic recovery
	var stepErr error
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error
			var panicErr error
			switch v := r.(type) {
			case error:
				panicErr = fmt.Errorf("panic in step %s: %w", name, v)
			case string:
				panicErr = fmt.Errorf("panic in step %s: %s", name, v)
			default:
				panicErr = fmt.Errorf("panic in step %s: %v", name, v)
			}

			stepErr = panicErr

			// Log the panic
			logger := t.Logger(stepCtx)
			logger.Error("Panic recovered in step",
				slog.String("step_name", name),
				slog.String("panic", fmt.Sprintf("%v", r)),
			)
		}

		// End the step with appropriate status
		var status Status
		if stepErr != nil {
			status = StatusFailed
		} else {
			status = StatusCompleted
		}

		if endErr := t.EndStep(stepID, status, stepErr); endErr != nil {
			logger := t.Logger(stepCtx)
			logger.Error("Failed to end step",
				slog.String("step_name", name),
				slog.String("error", endErr.Error()),
			)
		}
	}()

	// Execute the step function
	stepErr = fn(stepCtx)

	return stepErr
}

// Logger returns a logger with job and step context
func (t *Tracker) Logger(ctx context.Context) *slog.Logger {
	logger := t.logger

	// Add job ID if present
	if jobID, ok := JobIDFromCtx(ctx); ok && jobID != "" {
		logger = logger.With(slog.String("job_id", jobID))
	}

	// Add step ID if present
	if stepID, ok := StepIDFromCtx(ctx); ok {
		logger = logger.With(slog.Int64("step_id", stepID))
	}

	return logger
}

// GetJob retrieves a job record by ID
func (t *Tracker) GetJob(ctx context.Context, jobID string) (*JobRecord, error) {
	query := `
		SELECT id, parent_job_id, job_type, operation, status, percent_complete,
		       cloudstack_job_id, external_job_id, metadata, error_message, owner,
		       started_at, completed_at, canceled_at, created_at, updated_at
		FROM job_tracking WHERE id = ?
	`

	var job JobRecord
	err := t.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID,
		&job.ParentJobID,
		&job.JobType,
		&job.Operation,
		&job.Status,
		&job.PercentComplete,
		&job.CloudStackJobID,
		&job.ExternalJobID,
		&job.Metadata,
		&job.ErrorMessage,
		&job.Owner,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CanceledAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// GetJobProgress returns progress information for a job
func (t *Tracker) GetJobProgress(ctx context.Context, jobID string) (*ProgressInfo, error) {
	query := `
		SELECT 
			jt.id,
			COUNT(js.id) as total_steps,
			COUNT(CASE WHEN js.status = 'completed' THEN 1 END) as completed_steps,
			COUNT(CASE WHEN js.status = 'failed' THEN 1 END) as failed_steps,
			COUNT(CASE WHEN js.status = 'running' THEN 1 END) as running_steps,
			COUNT(CASE WHEN js.status = 'skipped' THEN 1 END) as skipped_steps,
			jt.percent_complete,
			jt.started_at,
			MAX(js.started_at) as last_activity,
			TIMESTAMPDIFF(SECOND, jt.started_at, NOW()) as runtime_seconds
		FROM job_tracking jt
		LEFT JOIN job_steps js ON jt.id = js.job_id
		WHERE jt.id = ?
		GROUP BY jt.id, jt.percent_complete, jt.started_at
	`

	var info ProgressInfo
	var lastActivity sql.NullTime

	err := t.db.QueryRowContext(ctx, query, jobID).Scan(
		&info.JobID,
		&info.TotalSteps,
		&info.CompletedSteps,
		&info.FailedSteps,
		&info.RunningSteps,
		&info.SkippedSteps,
		&info.ManualCompletion,
		&info.StartedAt,
		&lastActivity,
		&info.RuntimeSeconds,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job progress: %w", err)
	}

	if lastActivity.Valid {
		info.LastActivity = &lastActivity.Time
	}

	// Calculate step completion percentage
	if info.TotalSteps > 0 {
		completedAndSkipped := info.CompletedSteps + info.SkippedSteps
		info.StepCompletion = float64(completedAndSkipped) / float64(info.TotalSteps) * 100
	}

	return &info, nil
}

// getNextStepSequence gets the next sequence number for a step in the given job
func (t *Tracker) getNextStepSequence(ctx context.Context, jobID string) (int, error) {
	query := `SELECT COALESCE(MAX(seq), 0) + 1 FROM job_steps WHERE job_id = ?`

	var nextSeq int
	err := t.db.QueryRowContext(ctx, query, jobID).Scan(&nextSeq)
	if err != nil {
		return 0, fmt.Errorf("failed to get next step sequence: %w", err)
	}

	return nextSeq, nil
}

// Close shuts down the tracker and any associated resources
func (t *Tracker) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If the handler is a DBHandler, close it
	if dbHandler, ok := t.handler.(*DBHandler); ok {
		return dbHandler.Close()
	}

	// If it's a fanout handler, try to close any DBHandlers within it
	if fanoutHandler, ok := t.handler.(*FanoutHandler); ok {
		for _, handler := range fanoutHandler.handlers {
			if dbHandler, ok := handler.(*DBHandler); ok {
				dbHandler.Close()
			}
		}
	}

	return nil
}

// GetJobByExternalID retrieves a job by its external job ID (e.g., GUI-constructed ID)
func (t *Tracker) GetJobByExternalID(externalJobID string) (*JobSummary, error) {
	log.WithField("external_job_id", externalJobID).Info("🔍 DEBUG: GetJobByExternalID called")
	t.mu.RLock()
	defer t.mu.RUnlock()

	query := `
		SELECT id, parent_job_id, job_type, operation, status, percent_complete,
			   cloudstack_job_id, external_job_id, metadata, error_message, owner,
			   started_at, completed_at, canceled_at, created_at, updated_at,
			   context_id, job_category
		FROM job_tracking 
		WHERE external_job_id = ?
		ORDER BY started_at DESC
		LIMIT 1
	`
	log.WithField("external_job_id", externalJobID).Info("🔍 DEBUG: Executing query for external job ID")

	var job JobRecord
	err := t.db.QueryRow(query, externalJobID).Scan(
		&job.ID, &job.ParentJobID, &job.JobType, &job.Operation, &job.Status,
		&job.PercentComplete, &job.CloudStackJobID, &job.ExternalJobID,
		&job.Metadata, &job.ErrorMessage, &job.Owner, &job.StartedAt,
		&job.CompletedAt, &job.CanceledAt, &job.CreatedAt, &job.UpdatedAt,
		&job.ContextID, &job.JobCategory,
	)

	if err != nil {
		log.WithField("external_job_id", externalJobID).WithError(err).Error("🔍 DEBUG: Query failed in GetJobByExternalID")
		return nil, fmt.Errorf("job not found with external ID %s: %w", externalJobID, err)
	}

	log.WithField("external_job_id", externalJobID).WithField("found_job_id", job.ID).Info("🔍 DEBUG: Successfully found job by external ID")

	// Get steps and progress for the job
	steps, err := t.getJobSteps(job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get steps for job %s: %w", job.ID, err)
	}

	progress := t.calculateJobProgress(job, steps)

	return &JobSummary{
		Job:      job,
		Steps:    steps,
		Progress: progress,
	}, nil
}

// GetJobByContextID retrieves all jobs associated with a VM context ID
func (t *Tracker) GetJobByContextID(contextID string) ([]JobSummary, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	query := `
		SELECT id, parent_job_id, job_type, operation, status, percent_complete,
			   cloudstack_job_id, external_job_id, metadata, error_message, owner,
			   started_at, completed_at, canceled_at, created_at, updated_at,
			   context_id, job_category
		FROM job_tracking 
		WHERE context_id = ?
		ORDER BY started_at DESC
	`

	rows, err := t.db.Query(query, contextID)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs for context %s: %w", contextID, err)
	}
	defer rows.Close()

	var summaries []JobSummary

	for rows.Next() {
		var job JobRecord
		err := rows.Scan(
			&job.ID, &job.ParentJobID, &job.JobType, &job.Operation, &job.Status,
			&job.PercentComplete, &job.CloudStackJobID, &job.ExternalJobID,
			&job.Metadata, &job.ErrorMessage, &job.Owner, &job.StartedAt,
			&job.CompletedAt, &job.CanceledAt, &job.CreatedAt, &job.UpdatedAt,
			&job.ContextID, &job.JobCategory,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job record: %w", err)
		}

		// Get steps and progress for each job
		steps, err := t.getJobSteps(job.ID)
		if err != nil {
			// Don't fail the whole query for missing steps
			steps = []StepRecord{}
		}

		progress := t.calculateJobProgress(job, steps)

		summaries = append(summaries, JobSummary{
			Job:      job,
			Steps:    steps,
			Progress: progress,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job rows: %w", err)
	}

	return summaries, nil
}

// FindJobByAnyID attempts to find a job by any ID type (JobLog UUID, external ID, or context ID)
func (t *Tracker) FindJobByAnyID(anyID string) (*JobSummary, error) {
	// Try 1: Direct JobLog UUID lookup
	if job, err := t.GetJob(context.Background(), anyID); err == nil {
		// Build summary from job record
		steps, stepErr := t.getJobSteps(job.ID)
		if stepErr != nil {
			steps = []StepRecord{} // Don't fail on missing steps
		}
		progress := t.calculateJobProgress(*job, steps)

		return &JobSummary{
			Job:      *job,
			Steps:    steps,
			Progress: progress,
		}, nil
	}

	// Try 2: External job ID lookup (GUI compatibility)
	if summary, err := t.GetJobByExternalID(anyID); err == nil {
		return summary, nil
	}

	// Try 3: Context ID lookup (return most recent job)
	if summaries, err := t.GetJobByContextID(anyID); err == nil && len(summaries) > 0 {
		return &summaries[0], nil // Most recent job for this context
	}

	return nil, fmt.Errorf("job not found with ID: %s", anyID)
}

// GetJobProgressByAnyID retrieves progress information for a job by any ID type
func (t *Tracker) GetJobProgressByAnyID(jobID string) (*ProgressInfo, error) {
	summary, err := t.FindJobByAnyID(jobID)
	if err != nil {
		return nil, err
	}

	return &summary.Progress, nil
}

// Helper method to get job steps (used by new methods)
func (t *Tracker) getJobSteps(jobID string) ([]StepRecord, error) {
	query := `
		SELECT id, job_id, name, seq, status, started_at, completed_at, error_message, metadata
		FROM job_steps 
		WHERE job_id = ?
		ORDER BY seq ASC
	`

	rows, err := t.db.Query(query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to query job steps: %w", err)
	}
	defer rows.Close()

	var steps []StepRecord

	for rows.Next() {
		var step StepRecord
		err := rows.Scan(
			&step.ID, &step.JobID, &step.Name, &step.Seq, &step.Status,
			&step.StartedAt, &step.CompletedAt, &step.ErrorMessage, &step.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step record: %w", err)
		}

		steps = append(steps, step)
	}

	return steps, rows.Err()
}

// Helper method to calculate job progress (used by new methods)
func (t *Tracker) calculateJobProgress(job JobRecord, steps []StepRecord) ProgressInfo {
	totalSteps := len(steps)
	completedSteps := 0
	failedSteps := 0
	runningSteps := 0
	skippedSteps := 0

	var lastActivity *time.Time

	for _, step := range steps {
		switch step.Status {
		case StatusCompleted:
			completedSteps++
		case StatusFailed:
			failedSteps++
		case StatusRunning:
			runningSteps++
		case StatusSkipped:
			skippedSteps++
		}

		// Track last activity
		if step.CompletedAt != nil && (lastActivity == nil || step.CompletedAt.After(*lastActivity)) {
			lastActivity = step.CompletedAt
		}
	}

	// Calculate step completion percentage
	var stepCompletion float64
	if totalSteps > 0 {
		stepCompletion = float64(completedSteps) / float64(totalSteps) * 100
	}

	// Calculate runtime
	var runtimeSeconds int64
	if job.CompletedAt != nil {
		runtimeSeconds = int64(job.CompletedAt.Sub(job.StartedAt).Seconds())
	} else {
		runtimeSeconds = int64(time.Since(job.StartedAt).Seconds())
	}

	return ProgressInfo{
		JobID:            job.ID,
		TotalSteps:       totalSteps,
		CompletedSteps:   completedSteps,
		FailedSteps:      failedSteps,
		RunningSteps:     runningSteps,
		SkippedSteps:     skippedSteps,
		StepCompletion:   stepCompletion,
		ManualCompletion: job.PercentComplete,
		StartedAt:        job.StartedAt,
		LastActivity:     lastActivity,
		RuntimeSeconds:   runtimeSeconds,
	}
}

// Common errors
var (
	ErrInvalidJobType   = fmt.Errorf("job type cannot be empty")
	ErrInvalidOperation = fmt.Errorf("operation cannot be empty")
	ErrInvalidStepName  = fmt.Errorf("step name cannot be empty")
)
