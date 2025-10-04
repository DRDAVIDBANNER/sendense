package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// CloudStackJobTracking represents a CloudStack async job with full lifecycle tracking
type CloudStackJobTracking struct {
	ID string `gorm:"primaryKey;column:id" json:"id"`

	// CloudStack async job details
	CloudStackJobID      *string `gorm:"column:cloudstack_job_id" json:"cloudstack_job_id,omitempty"`
	CloudStackCommand    string  `gorm:"column:cloudstack_command;not null" json:"cloudstack_command"`
	CloudStackStatus     string  `gorm:"column:cloudstack_status;default:pending" json:"cloudstack_status"`
	CloudStackResultCode *int    `gorm:"column:cloudstack_result_code" json:"cloudstack_result_code,omitempty"`
	CloudStackResponse   JSON    `gorm:"column:cloudstack_response;type:json" json:"cloudstack_response,omitempty"`

	// Operation correlation
	OperationType string                  `gorm:"column:operation_type;not null" json:"operation_type"`
	CorrelationID string                  `gorm:"column:correlation_id;not null" json:"correlation_id"`
	ParentJobID   *string                 `gorm:"column:parent_job_id" json:"parent_job_id,omitempty"`
	ParentJob     *CloudStackJobTracking  `gorm:"foreignKey:ParentJobID" json:"parent_job,omitempty"`
	ChildJobs     []CloudStackJobTracking `gorm:"foreignKey:ParentJobID" json:"child_jobs,omitempty"`

	// Request/Response tracking
	RequestData      JSON    `gorm:"column:request_data;type:json;not null" json:"request_data"`
	LocalOperationID *string `gorm:"column:local_operation_id" json:"local_operation_id,omitempty"`

	// Execution tracking
	Status     string     `gorm:"column:status;default:initiated" json:"status"`
	RetryCount int        `gorm:"column:retry_count;default:0" json:"retry_count"`
	MaxRetries int        `gorm:"column:max_retries;default:3" json:"max_retries"`
	NextPollAt *time.Time `gorm:"column:next_poll_at" json:"next_poll_at,omitempty"`

	// Error handling
	ErrorMessage *string `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	ErrorDetails JSON    `gorm:"column:error_details;type:json" json:"error_details,omitempty"`

	// Audit trail
	InitiatedBy string     `gorm:"column:initiated_by;not null" json:"initiated_by"`
	InitiatedAt time.Time  `gorm:"column:initiated_at;autoCreateTime" json:"initiated_at"`
	SubmittedAt *time.Time `gorm:"column:submitted_at" json:"submitted_at,omitempty"`
	CompletedAt *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Related records
	ExecutionLogs []CloudStackJobExecutionLog `gorm:"foreignKey:JobTrackingID" json:"execution_logs,omitempty"`
	PollQueue     *CloudStackJobPollQueue     `gorm:"foreignKey:JobTrackingID" json:"poll_queue,omitempty"`
}

// CloudStackJobExecutionLog provides detailed audit trail for job execution
type CloudStackJobExecutionLog struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	JobTrackingID string `gorm:"column:job_tracking_id;not null" json:"job_tracking_id"`

	// Log entry details
	LogLevel string `gorm:"column:log_level;not null" json:"log_level"`
	Message  string `gorm:"column:message;type:text;not null" json:"message"`
	Details  JSON   `gorm:"column:details;type:json" json:"details,omitempty"`

	// Context
	OperationPhase      *string `gorm:"column:operation_phase" json:"operation_phase,omitempty"`
	CloudStackJobStatus *string `gorm:"column:cloudstack_job_status" json:"cloudstack_job_status,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Relationships
	JobTracking CloudStackJobTracking `gorm:"foreignKey:JobTrackingID" json:"-"`
}

// CloudStackJobPollQueue manages active polling for CloudStack jobs
type CloudStackJobPollQueue struct {
	ID            string `gorm:"primaryKey;column:id" json:"id"`
	JobTrackingID string `gorm:"column:job_tracking_id;not null;unique" json:"job_tracking_id"`

	// Polling configuration
	PollIntervalSeconds    int       `gorm:"column:poll_interval_seconds;default:2" json:"poll_interval_seconds"`
	NextPollAt             time.Time `gorm:"column:next_poll_at;not null" json:"next_poll_at"`
	ConsecutiveFailures    int       `gorm:"column:consecutive_failures;default:0" json:"consecutive_failures"`
	MaxConsecutiveFailures int       `gorm:"column:max_consecutive_failures;default:5" json:"max_consecutive_failures"`

	// Queue management
	IsActive bool `gorm:"column:is_active;default:true" json:"is_active"`
	Priority int  `gorm:"column:priority;default:0" json:"priority"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	JobTracking CloudStackJobTracking `gorm:"foreignKey:JobTrackingID" json:"-"`
}

// CloudStackJobMetrics stores operational metrics for performance monitoring
type CloudStackJobMetrics struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Timestamp   time.Time `gorm:"column:timestamp;autoCreateTime" json:"timestamp"`
	WindowStart time.Time `gorm:"column:window_start;not null" json:"window_start"`
	WindowEnd   time.Time `gorm:"column:window_end;not null" json:"window_end"`

	// Job statistics
	TotalJobsInitiated int `gorm:"column:total_jobs_initiated;default:0" json:"total_jobs_initiated"`
	TotalJobsCompleted int `gorm:"column:total_jobs_completed;default:0" json:"total_jobs_completed"`
	TotalJobsFailed    int `gorm:"column:total_jobs_failed;default:0" json:"total_jobs_failed"`
	TotalJobsCancelled int `gorm:"column:total_jobs_cancelled;default:0" json:"total_jobs_cancelled"`

	// Performance metrics
	AverageCompletionTimeSeconds float64 `gorm:"column:average_completion_time_seconds;default:0" json:"average_completion_time_seconds"`
	AveragePollCycles            float32 `gorm:"column:average_poll_cycles;default:0" json:"average_poll_cycles"`
	LongestCompletionTimeSeconds int     `gorm:"column:longest_completion_time_seconds;default:0" json:"longest_completion_time_seconds"`

	// Operation breakdown
	OperationsByType JSON `gorm:"column:operations_by_type;type:json" json:"operations_by_type,omitempty"`
	CommonErrors     JSON `gorm:"column:common_errors;type:json" json:"common_errors,omitempty"`
}

// JSON is a custom type for handling JSON fields in GORM
type JSON map[string]interface{}

// Value implements the driver.Valuer interface for JSON
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSON
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSON", value)
	}

	if len(bytes) == 0 {
		*j = nil
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// JobTrackingStatus represents the possible statuses for job tracking
type JobTrackingStatus string

const (
	JobStatusInitiated JobTrackingStatus = "initiated"
	JobStatusSubmitted JobTrackingStatus = "submitted"
	JobStatusPolling   JobTrackingStatus = "polling"
	JobStatusCompleted JobTrackingStatus = "completed"
	JobStatusFailed    JobTrackingStatus = "failed"
	JobStatusCancelled JobTrackingStatus = "cancelled"
)

// CloudStackJobStatus represents CloudStack async job statuses
type CloudStackJobStatus string

const (
	CloudStackStatusPending    CloudStackJobStatus = "pending"
	CloudStackStatusInProgress CloudStackJobStatus = "in-progress"
	CloudStackStatusSuccess    CloudStackJobStatus = "success"
	CloudStackStatusFailure    CloudStackJobStatus = "failure"
)

// OperationType represents different types of operations being tracked
type OperationType string

const (
	OpTypeTestFailoverCleanup OperationType = "test-failover-cleanup"
	OpTypeVMDelete            OperationType = "vm-delete"
	OpTypeVMPowerOff          OperationType = "vm-power-off"
	OpTypeVolumeAttach        OperationType = "volume-attach"
	OpTypeVolumeDetach        OperationType = "volume-detach"
	OpTypeVolumeCreate        OperationType = "volume-create"
	OpTypeVolumeDelete        OperationType = "volume-delete"
	OpTypeVMCreate            OperationType = "vm-create"
	OpTypeVMStart             OperationType = "vm-start"
	OpTypeVMStop              OperationType = "vm-stop"
	OpTypeVMSnapshot          OperationType = "vm-snapshot"
	OpTypeVMSnapshotRevert    OperationType = "vm-snapshot-revert"
)

// LogLevel represents log levels for execution logs
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// Table names for GORM
func (CloudStackJobTracking) TableName() string     { return "cloudstack_job_tracking" }
func (CloudStackJobExecutionLog) TableName() string { return "cloudstack_job_execution_log" }
func (CloudStackJobPollQueue) TableName() string    { return "cloudstack_job_poll_queue" }
func (CloudStackJobMetrics) TableName() string      { return "cloudstack_job_metrics" }

// Helper methods for CloudStackJobTracking

// IsCompleted returns true if the job has reached a final state
func (j *CloudStackJobTracking) IsCompleted() bool {
	return j.Status == string(JobStatusCompleted) ||
		j.Status == string(JobStatusFailed) ||
		j.Status == string(JobStatusCancelled)
}

// IsActive returns true if the job is still being processed
func (j *CloudStackJobTracking) IsActive() bool {
	return !j.IsCompleted()
}

// ShouldRetry returns true if the job should be retried based on retry count
func (j *CloudStackJobTracking) ShouldRetry() bool {
	return j.RetryCount < j.MaxRetries
}

// Duration returns the total duration of the job execution
func (j *CloudStackJobTracking) Duration() *time.Duration {
	if j.CompletedAt == nil {
		return nil
	}
	duration := j.CompletedAt.Sub(j.InitiatedAt)
	return &duration
}

// Helper methods for CloudStackJobPollQueue

// ShouldPoll returns true if the queue item should be polled now
func (q *CloudStackJobPollQueue) ShouldPoll() bool {
	return q.IsActive && time.Now().After(q.NextPollAt) &&
		q.ConsecutiveFailures < q.MaxConsecutiveFailures
}

// CalculateNextPoll calculates the next poll time based on interval
func (q *CloudStackJobPollQueue) CalculateNextPoll() time.Time {
	return time.Now().Add(time.Duration(q.PollIntervalSeconds) * time.Second)
}
