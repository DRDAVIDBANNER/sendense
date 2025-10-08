package joblog

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"runtime/debug"
	"strings"
)

// RecoverAsFailure handles panics and converts them to failed job/step status
// This function should be called in a defer statement to capture panics
func RecoverAsFailure(ctx context.Context, tracker *Tracker, jobID string, stepID *int64) {
	if r := recover(); r != nil {
		// Get stack trace
		stack := debug.Stack()
		stackStr := string(stack)
		
		// Format the panic message
		var panicMsg string
		switch v := r.(type) {
		case error:
			panicMsg = v.Error()
		case string:
			panicMsg = v
		default:
			panicMsg = fmt.Sprintf("%v", v)
		}
		
		// Create error message with panic details
		errorMsg := fmt.Sprintf("Panic recovered: %s", panicMsg)
		
		// Log the panic with full context
		logger := tracker.Logger(ctx)
		logger.Error("Panic recovered in job/step execution",
			slog.String("panic_message", panicMsg),
			slog.String("job_id", jobID),
			slog.Any("step_id", stepID),
			slog.String("stack_trace", stackStr),
			slog.String("recovery_point", getRecoveryPoint()),
		)
		
		// Update step status if we have a step ID
		if stepID != nil {
			if err := tracker.EndStep(*stepID, StatusFailed, fmt.Errorf(errorMsg)); err != nil {
				logger.Error("Failed to update step status after panic recovery",
					slog.String("job_id", jobID),
					slog.Int64("step_id", *stepID),
					slog.String("error", err.Error()),
				)
			}
		}
		
		// Update job status - always mark as failed on panic
		if err := tracker.EndJob(ctx, jobID, StatusFailed, fmt.Errorf(errorMsg)); err != nil {
			logger.Error("Failed to update job status after panic recovery",
				slog.String("job_id", jobID),
				slog.String("error", err.Error()),
			)
		}
		
		// Re-panic to ensure the panic is not silently swallowed
		// This ensures the calling code can still handle the panic if needed
		panic(r)
	}
}

// SafeEndJob safely ends a job, handling any errors that occur
func SafeEndJob(ctx context.Context, tracker *Tracker, jobID string, status Status, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger := tracker.Logger(ctx)
			logger.Error("Panic occurred while ending job",
				slog.String("job_id", jobID),
				slog.String("status", string(status)),
				slog.String("panic", fmt.Sprintf("%v", r)),
			)
		}
	}()
	
	if endErr := tracker.EndJob(ctx, jobID, status, err); endErr != nil {
		logger := tracker.Logger(ctx)
		logger.Error("Failed to end job",
			slog.String("job_id", jobID),
			slog.String("status", string(status)),
			slog.String("error", endErr.Error()),
		)
	}
}

// SafeEndStep safely ends a step, handling any errors that occur
func SafeEndStep(ctx context.Context, tracker *Tracker, stepID int64, status Status, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger := tracker.Logger(ctx)
			logger.Error("Panic occurred while ending step",
				slog.Int64("step_id", stepID),
				slog.String("status", string(status)),
				slog.String("panic", fmt.Sprintf("%v", r)),
			)
		}
	}()
	
	if endErr := tracker.EndStep(stepID, status, err); endErr != nil {
		logger := tracker.Logger(ctx)
		logger.Error("Failed to end step",
			slog.Int64("step_id", stepID),
			slog.String("status", string(status)),
			slog.String("error", endErr.Error()),
		)
	}
}

// WithRecovery wraps a function with panic recovery that automatically updates job/step status
func WithRecovery(ctx context.Context, tracker *Tracker, jobID string, stepID *int64, fn func() error) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error
			var panicErr error
			switch v := r.(type) {
			case error:
				panicErr = fmt.Errorf("panic: %w", v)
			case string:
				panicErr = fmt.Errorf("panic: %s", v)
			default:
				panicErr = fmt.Errorf("panic: %v", v)
			}
			
			// Set return error
			retErr = panicErr
			
			// Log the panic
			logger := tracker.Logger(ctx)
			stack := debug.Stack()
			logger.Error("Panic recovered in wrapped function",
				slog.String("panic_error", panicErr.Error()),
				slog.String("job_id", jobID),
				slog.Any("step_id", stepID),
				slog.String("stack_trace", string(stack)),
			)
			
			// Update statuses
			if stepID != nil {
				SafeEndStep(ctx, tracker, *stepID, StatusFailed, panicErr)
			}
			// Note: We don't auto-fail the job here since the caller might want to continue
		}
	}()
	
	return fn()
}

// getRecoveryPoint returns information about where the recovery occurred
func getRecoveryPoint() string {
	// Skip getRecoveryPoint, RecoverAsFailure, and the defer function
	if _, file, line, ok := runtime.Caller(3); ok {
		// Extract just the filename from the full path
		parts := strings.Split(file, "/")
		filename := parts[len(parts)-1]
		return fmt.Sprintf("%s:%d", filename, line)
	}
	return "unknown"
}

// ErrorWithContext enriches an error with contextual information
func ErrorWithContext(ctx context.Context, err error, operation string, metadata map[string]any) error {
	if err == nil {
		return nil
	}
	
	// Build context string
	var contextParts []string
	
	if jobID, ok := JobIDFromCtx(ctx); ok {
		contextParts = append(contextParts, fmt.Sprintf("job_id=%s", jobID))
	}
	
	if stepID, ok := StepIDFromCtx(ctx); ok {
		contextParts = append(contextParts, fmt.Sprintf("step_id=%d", stepID))
	}
	
	if operation != "" {
		contextParts = append(contextParts, fmt.Sprintf("operation=%s", operation))
	}
	
	// Add metadata if provided
	if metadata != nil {
		for k, v := range metadata {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	
	contextStr := strings.Join(contextParts, ", ")
	
	if contextStr != "" {
		return fmt.Errorf("%w [%s]", err, contextStr)
	}
	
	return err
}

// HandleJobError provides standardized error handling for jobs
func HandleJobError(ctx context.Context, tracker *Tracker, jobID string, err error, operation string) error {
	if err == nil {
		return nil
	}
	
	// Enrich error with context
	enrichedErr := ErrorWithContext(ctx, err, operation, nil)
	
	// Log the error
	logger := tracker.Logger(ctx)
	logger.Error("Job operation failed",
		slog.String("job_id", jobID),
		slog.String("operation", operation),
		slog.String("error", enrichedErr.Error()),
	)
	
	return enrichedErr
}

// HandleStepError provides standardized error handling for steps
func HandleStepError(ctx context.Context, tracker *Tracker, stepID int64, err error, operation string) error {
	if err == nil {
		return nil
	}
	
	// Enrich error with context
	enrichedErr := ErrorWithContext(ctx, err, operation, nil)
	
	// Log the error
	logger := tracker.Logger(ctx)
	logger.Error("Step operation failed",
		slog.Int64("step_id", stepID),
		slog.String("operation", operation),
		slog.String("error", enrichedErr.Error()),
	)
	
	return enrichedErr
}

// IsRetryableError determines if an error might be retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Common retryable error patterns
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network unreachable",
		"connection reset",
		"no such host",
		"i/o timeout",
		"context deadline exceeded",
		"database is locked",
		"deadlock",
		"connection lost",
	}
	
	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	
	return false
}

// IsCriticalError determines if an error should immediately fail the entire job
func IsCriticalError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Critical error patterns that should fail the entire job
	criticalPatterns := []string{
		"permission denied",
		"authentication failed",
		"authorization failed",
		"invalid credentials",
		"access denied",
		"forbidden",
		"not found",
		"invalid configuration",
		"validation failed",
		"schema violation",
		"constraint violation",
		"panic:",
	}
	
	for _, pattern := range criticalPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	
	return false
}
