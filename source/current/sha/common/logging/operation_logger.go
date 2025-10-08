// Package logging provides centralized, structured logging for all MigrateKit operations
// MANDATORY: All component updates and operations MUST use this centralized logging system
package logging

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// OperationLogger provides centralized logging interface for all operations
// MANDATORY: All operations MUST implement this interface
type OperationLogger interface {
	StartOperation(operation, jobID string) OperationContext
	GetCorrelationID() string
	GetJobID() string
}

// OperationContext provides context for a specific operation with step tracking
type OperationContext interface {
	LogStep(step, message string, fields log.Fields)
	LogError(step, message string, err error, fields log.Fields)
	LogDuration(step string, start time.Time)
	LogSuccess(step, message string, fields log.Fields)
	StartStep(stepName string) StepContext
	EndOperation(status string, summary log.Fields)
	GetCorrelationID() string
	GetJobID() string
	CreateChildContext(childOperation string) OperationContext
}

// StepContext provides context for individual operation steps
type StepContext interface {
	Info(message string, fields log.Fields)
	Warn(message string, fields log.Fields)
	Error(message string, err error, fields log.Fields)
	Success(message string, fields log.Fields)
	EndStep(status string, fields log.Fields)
	LogDuration(start time.Time)
	GetStepName() string
}

// operationLogger implements OperationLogger interface
type operationLogger struct {
	correlationID string
	jobID         string
	baseLogger    *log.Logger
}

// operationContext implements OperationContext interface
type operationContext struct {
	operation     string
	correlationID string
	jobID         string
	baseLogger    *log.Logger
	startTime     time.Time
}

// stepContext implements StepContext interface
type stepContext struct {
	stepName      string
	operation     string
	correlationID string
	jobID         string
	baseLogger    *log.Logger
	startTime     time.Time
}

// NewOperationLogger creates a new centralized operation logger
// MANDATORY: Use this for ALL operation logging
func NewOperationLogger(jobID string) OperationLogger {
	return &operationLogger{
		correlationID: uuid.New().String(),
		jobID:         jobID,
		baseLogger:    log.StandardLogger(),
	}
}

// NewOperationLoggerWithCorrelation creates logger with existing correlation ID
// Used for child operations and service-to-service calls
func NewOperationLoggerWithCorrelation(jobID, correlationID string) OperationLogger {
	return &operationLogger{
		correlationID: correlationID,
		jobID:         jobID,
		baseLogger:    log.StandardLogger(),
	}
}

// StartOperation begins a new operation with full context tracking
func (ol *operationLogger) StartOperation(operation, jobID string) OperationContext {
	if jobID != "" {
		ol.jobID = jobID
	}

	ctx := &operationContext{
		operation:     operation,
		correlationID: ol.correlationID,
		jobID:         ol.jobID,
		baseLogger:    ol.baseLogger,
		startTime:     time.Now(),
	}

	// Log operation start
	ctx.baseLogger.WithFields(log.Fields{
		"correlation_id": ctx.correlationID,
		"job_id":         ctx.jobID,
		"operation":      ctx.operation,
		"event":          "operation_start",
		"timestamp":      ctx.startTime.Format(time.RFC3339),
	}).Info(fmt.Sprintf("🚀 Starting operation: %s", operation))

	return ctx
}

// GetCorrelationID returns the correlation ID for tracing
func (ol *operationLogger) GetCorrelationID() string {
	return ol.correlationID
}

// GetJobID returns the job ID
func (ol *operationLogger) GetJobID() string {
	return ol.jobID
}

// OperationContext Implementation

// LogStep logs a major step in the operation
func (oc *operationContext) LogStep(step, message string, fields log.Fields) {
	if fields == nil {
		fields = log.Fields{}
	}

	fields["correlation_id"] = oc.correlationID
	fields["job_id"] = oc.jobID
	fields["operation"] = oc.operation
	fields["step"] = step
	fields["event"] = "step_info"

	oc.baseLogger.WithFields(fields).Info(fmt.Sprintf("🔄 %s: %s", step, message))
}

// LogError logs an error with full context
func (oc *operationContext) LogError(step, message string, err error, fields log.Fields) {
	if fields == nil {
		fields = log.Fields{}
	}

	fields["correlation_id"] = oc.correlationID
	fields["job_id"] = oc.jobID
	fields["operation"] = oc.operation
	fields["step"] = step
	fields["event"] = "step_error"
	fields["error"] = err.Error()

	oc.baseLogger.WithFields(fields).Error(fmt.Sprintf("❌ %s ERROR: %s - %v", step, message, err))
}

// LogDuration logs the duration of a step
func (oc *operationContext) LogDuration(step string, start time.Time) {
	duration := time.Since(start)

	oc.baseLogger.WithFields(log.Fields{
		"correlation_id": oc.correlationID,
		"job_id":         oc.jobID,
		"operation":      oc.operation,
		"step":           step,
		"event":          "step_duration",
		"duration_ms":    duration.Milliseconds(),
		"duration_str":   duration.String(),
	}).Info(fmt.Sprintf("⏱️ %s completed in %v", step, duration))
}

// LogSuccess logs successful completion of a step
func (oc *operationContext) LogSuccess(step, message string, fields log.Fields) {
	if fields == nil {
		fields = log.Fields{}
	}

	fields["correlation_id"] = oc.correlationID
	fields["job_id"] = oc.jobID
	fields["operation"] = oc.operation
	fields["step"] = step
	fields["event"] = "step_success"

	oc.baseLogger.WithFields(fields).Info(fmt.Sprintf("✅ %s SUCCESS: %s", step, message))
}

// StartStep creates a context for detailed step logging
func (oc *operationContext) StartStep(stepName string) StepContext {
	stepCtx := &stepContext{
		stepName:      stepName,
		operation:     oc.operation,
		correlationID: oc.correlationID,
		jobID:         oc.jobID,
		baseLogger:    oc.baseLogger,
		startTime:     time.Now(),
	}

	stepCtx.baseLogger.WithFields(log.Fields{
		"correlation_id": stepCtx.correlationID,
		"job_id":         stepCtx.jobID,
		"operation":      stepCtx.operation,
		"step":           stepCtx.stepName,
		"event":          "step_start",
		"timestamp":      stepCtx.startTime.Format(time.RFC3339),
	}).Info(fmt.Sprintf("🔄 Starting step: %s", stepName))

	return stepCtx
}

// EndOperation logs the completion of an operation
func (oc *operationContext) EndOperation(status string, summary log.Fields) {
	duration := time.Since(oc.startTime)

	if summary == nil {
		summary = log.Fields{}
	}

	summary["correlation_id"] = oc.correlationID
	summary["job_id"] = oc.jobID
	summary["operation"] = oc.operation
	summary["status"] = status
	summary["event"] = "operation_end"
	summary["total_duration_ms"] = duration.Milliseconds()
	summary["total_duration_str"] = duration.String()

	var logMessage string
	var logLevel log.Level
	switch status {
	case "completed":
		logMessage = fmt.Sprintf("✅ Operation completed: %s", oc.operation)
		logLevel = log.InfoLevel
	case "failed":
		logMessage = fmt.Sprintf("❌ Operation failed: %s", oc.operation)
		logLevel = log.ErrorLevel
	default:
		logMessage = fmt.Sprintf("🔄 Operation ended: %s (status: %s)", oc.operation, status)
		logLevel = log.InfoLevel
	}

	oc.baseLogger.WithFields(summary).Log(logLevel, logMessage)
}

// GetCorrelationID returns correlation ID from operation context
func (oc *operationContext) GetCorrelationID() string {
	return oc.correlationID
}

// GetJobID returns job ID from operation context
func (oc *operationContext) GetJobID() string {
	return oc.jobID
}

// CreateChildContext creates a child operation context with inherited correlation
func (oc *operationContext) CreateChildContext(childOperation string) OperationContext {
	childCtx := &operationContext{
		operation:     childOperation,
		correlationID: oc.correlationID, // Inherit parent correlation ID
		jobID:         oc.jobID,
		baseLogger:    oc.baseLogger,
		startTime:     time.Now(),
	}

	// Log child operation start
	childCtx.baseLogger.WithFields(log.Fields{
		"correlation_id":   childCtx.correlationID,
		"job_id":           childCtx.jobID,
		"parent_operation": oc.operation,
		"child_operation":  childOperation,
		"event":            "child_operation_start",
		"timestamp":        childCtx.startTime.Format(time.RFC3339),
	}).Info(fmt.Sprintf("🔄 Starting child operation: %s", childOperation))

	return childCtx
}

// StepContext Implementation

// Info logs informational message for the step
func (sc *stepContext) Info(message string, fields log.Fields) {
	if fields == nil {
		fields = log.Fields{}
	}

	fields["correlation_id"] = sc.correlationID
	fields["job_id"] = sc.jobID
	fields["operation"] = sc.operation
	fields["step"] = sc.stepName
	fields["event"] = "step_info"

	sc.baseLogger.WithFields(fields).Info(fmt.Sprintf("💬 %s: %s", sc.stepName, message))
}

// Warn logs warning message for the step
func (sc *stepContext) Warn(message string, fields log.Fields) {
	if fields == nil {
		fields = log.Fields{}
	}

	fields["correlation_id"] = sc.correlationID
	fields["job_id"] = sc.jobID
	fields["operation"] = sc.operation
	fields["step"] = sc.stepName
	fields["event"] = "step_warning"

	sc.baseLogger.WithFields(fields).Warn(fmt.Sprintf("⚠️ %s WARNING: %s", sc.stepName, message))
}

// Error logs error message for the step
func (sc *stepContext) Error(message string, err error, fields log.Fields) {
	if fields == nil {
		fields = log.Fields{}
	}

	fields["correlation_id"] = sc.correlationID
	fields["job_id"] = sc.jobID
	fields["operation"] = sc.operation
	fields["step"] = sc.stepName
	fields["event"] = "step_error"
	fields["error"] = err.Error()

	sc.baseLogger.WithFields(fields).Error(fmt.Sprintf("❌ %s ERROR: %s - %v", sc.stepName, message, err))
}

// Success logs success message for the step
func (sc *stepContext) Success(message string, fields log.Fields) {
	if fields == nil {
		fields = log.Fields{}
	}

	fields["correlation_id"] = sc.correlationID
	fields["job_id"] = sc.jobID
	fields["operation"] = sc.operation
	fields["step"] = sc.stepName
	fields["event"] = "step_success"

	sc.baseLogger.WithFields(fields).Info(fmt.Sprintf("✅ %s SUCCESS: %s", sc.stepName, message))
}

// EndStep logs the completion of a step
func (sc *stepContext) EndStep(status string, fields log.Fields) {
	duration := time.Since(sc.startTime)

	if fields == nil {
		fields = log.Fields{}
	}

	fields["correlation_id"] = sc.correlationID
	fields["job_id"] = sc.jobID
	fields["operation"] = sc.operation
	fields["step"] = sc.stepName
	fields["status"] = status
	fields["event"] = "step_end"
	fields["step_duration_ms"] = duration.Milliseconds()
	fields["step_duration_str"] = duration.String()

	var logMessage string
	switch status {
	case "completed":
		logMessage = fmt.Sprintf("✅ Step completed: %s", sc.stepName)
	case "failed":
		logMessage = fmt.Sprintf("❌ Step failed: %s", sc.stepName)
	default:
		logMessage = fmt.Sprintf("🔄 Step ended: %s (status: %s)", sc.stepName, status)
	}

	sc.baseLogger.WithFields(fields).Info(logMessage)
}

// LogDuration logs the duration of the step
func (sc *stepContext) LogDuration(start time.Time) {
	duration := time.Since(start)

	sc.baseLogger.WithFields(log.Fields{
		"correlation_id": sc.correlationID,
		"job_id":         sc.jobID,
		"operation":      sc.operation,
		"step":           sc.stepName,
		"event":          "step_duration",
		"duration_ms":    duration.Milliseconds(),
		"duration_str":   duration.String(),
	}).Info(fmt.Sprintf("⏱️ %s duration: %v", sc.stepName, duration))
}

// GetStepName returns the step name
func (sc *stepContext) GetStepName() string {
	return sc.stepName
}

// Utility Functions

// WithContext adds correlation ID to existing context
func WithContext(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, "correlation_id", correlationID)
}

// GetCorrelationFromContext extracts correlation ID from context
func GetCorrelationFromContext(ctx context.Context) string {
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}
	return ""
}

