// Package joblog provides unified job tracking and structured logging for MigrateKit operations
package joblog

import (
	"context"
	"time"
)

// Status represents the possible states of jobs and steps
type Status string

const (
	// StatusPending indicates a job/step is waiting to start
	StatusPending Status = "pending"
	
	// StatusRunning indicates a job/step is currently executing
	StatusRunning Status = "running"
	
	// StatusCompleted indicates a job/step finished successfully
	StatusCompleted Status = "completed"
	
	// StatusFailed indicates a job/step failed with an error
	StatusFailed Status = "failed"
	
	// StatusCancelled indicates a job/step was cancelled before completion
	StatusCancelled Status = "cancelled"
	
	// StatusSkipped indicates a step was skipped (only valid for steps)
	StatusSkipped Status = "skipped"
)

// JobStart contains the parameters for starting a new job
type JobStart struct {
	// ParentJobID links this job to a parent job for hierarchical tracking
	ParentJobID *string `json:"parent_job_id,omitempty"`
	
	// JobType categorizes the type of job being performed
	JobType string `json:"job_type"`
	
	// Operation describes the specific operation being performed
	Operation string `json:"operation"`
	
	// Owner identifies who or what initiated this job
	Owner *string `json:"owner,omitempty"`
	
	// Metadata contains arbitrary job-specific data
	Metadata any `json:"metadata,omitempty"`
}

// StepStart contains the parameters for starting a new step
type StepStart struct {
	// Name is the human-readable name of the step
	Name string `json:"name"`
	
	// Seq is the sequence number within the job (auto-generated if 0)
	Seq int `json:"seq"`
	
	// Metadata contains arbitrary step-specific data
	Metadata any `json:"metadata,omitempty"`
}

// JobRecord represents a job in the database
type JobRecord struct {
	ID              string     `db:"id"`
	ParentJobID     *string    `db:"parent_job_id"`
	JobType         string     `db:"job_type"`
	Operation       string     `db:"operation"`
	Status          Status     `db:"status"`
	PercentComplete *uint8     `db:"percent_complete"`
	CloudStackJobID *string    `db:"cloudstack_job_id"`
	ExternalJobID   *string    `db:"external_job_id"`
	Metadata        *string    `db:"metadata"` // JSON string
	ErrorMessage    *string    `db:"error_message"`
	Owner           *string    `db:"owner"`
	StartedAt       time.Time  `db:"started_at"`
	CompletedAt     *time.Time `db:"completed_at"`
	CanceledAt      *time.Time `db:"canceled_at"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

// StepRecord represents a step in the database
type StepRecord struct {
	ID           int64      `db:"id"`
	JobID        string     `db:"job_id"`
	Name         string     `db:"name"`
	Seq          int        `db:"seq"`
	Status       Status     `db:"status"`
	StartedAt    time.Time  `db:"started_at"`
	CompletedAt  *time.Time `db:"completed_at"`
	ErrorMessage *string    `db:"error_message"`
	Metadata     *string    `db:"metadata"` // JSON string
}

// LogRecord represents a log event in the database
type LogRecord struct {
	ID     int64  `db:"id"`
	JobID  *string `db:"job_id"`
	StepID *int64 `db:"step_id"`
	Level  string `db:"level"`
	Message string `db:"message"`
	Attrs  *string `db:"attrs"` // JSON string
	Ts     time.Time `db:"ts"`
}

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	// jobIDKey is the context key for job IDs
	jobIDKey contextKey = "joblog_job_id"
	
	// stepIDKey is the context key for step IDs
	stepIDKey contextKey = "joblog_step_id"
)

// WithJobID adds a job ID to the context
func WithJobID(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, jobIDKey, jobID)
}

// WithStepID adds a step ID to the context
func WithStepID(ctx context.Context, stepID int64) context.Context {
	return context.WithValue(ctx, stepIDKey, stepID)
}

// JobIDFromCtx extracts the job ID from the context
func JobIDFromCtx(ctx context.Context) (string, bool) {
	jobID, ok := ctx.Value(jobIDKey).(string)
	return jobID, ok
}

// StepIDFromCtx extracts the step ID from the context
func StepIDFromCtx(ctx context.Context) (int64, bool) {
	stepID, ok := ctx.Value(stepIDKey).(int64)
	return stepID, ok
}

// ProgressInfo provides a summary of job progress
type ProgressInfo struct {
	JobID              string    `json:"job_id"`
	TotalSteps         int       `json:"total_steps"`
	CompletedSteps     int       `json:"completed_steps"`
	FailedSteps        int       `json:"failed_steps"`
	RunningSteps       int       `json:"running_steps"`
	SkippedSteps       int       `json:"skipped_steps"`
	StepCompletion     float64   `json:"step_completion_percentage"`
	ManualCompletion   *uint8    `json:"manual_completion_percentage,omitempty"`
	StartedAt          time.Time `json:"started_at"`
	LastActivity       *time.Time `json:"last_activity,omitempty"`
	RuntimeSeconds     int64     `json:"runtime_seconds"`
}

// JobSummary provides a complete summary of a job
type JobSummary struct {
	Job      JobRecord      `json:"job"`
	Steps    []StepRecord   `json:"steps"`
	Progress ProgressInfo   `json:"progress"`
}

// LogFilter provides filtering options for log queries
type LogFilter struct {
	JobID    *string    `json:"job_id,omitempty"`
	StepID   *int64     `json:"step_id,omitempty"`
	Level    *string    `json:"level,omitempty"`
	Since    *time.Time `json:"since,omitempty"`
	Until    *time.Time `json:"until,omitempty"`
	Limit    int        `json:"limit,omitempty"`
}

// JobFilter provides filtering options for job queries
type JobFilter struct {
	ParentJobID *string   `json:"parent_job_id,omitempty"`
	JobType     *string   `json:"job_type,omitempty"`
	Status      *Status   `json:"status,omitempty"`
	Owner       *string   `json:"owner,omitempty"`
	Since       *time.Time `json:"since,omitempty"`
	Until       *time.Time `json:"until,omitempty"`
	Limit       int       `json:"limit,omitempty"`
}

// IsTerminal returns true if the status represents a terminal state
func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCancelled
}

// IsActive returns true if the status represents an active state
func (s Status) IsActive() bool {
	return s == StatusPending || s == StatusRunning
}

// String returns the string representation of the status
func (s Status) String() string {
	return string(s)
}

// Validate ensures the JobStart has required fields
func (js *JobStart) Validate() error {
	if js.JobType == "" {
		return ErrInvalidJobType
	}
	if js.Operation == "" {
		return ErrInvalidOperation
	}
	return nil
}

// Validate ensures the StepStart has required fields
func (ss *StepStart) Validate() error {
	if ss.Name == "" {
		return ErrInvalidStepName
	}
	return nil
}
