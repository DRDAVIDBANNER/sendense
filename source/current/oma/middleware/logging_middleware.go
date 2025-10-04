package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/services"
)

// LoggingMiddleware provides centralized request logging with correlation
type LoggingMiddleware struct {
	centralLogger *services.CentralLogger
	component     string
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(centralLogger *services.CentralLogger, component string) *LoggingMiddleware {
	if component == "" {
		component = "api"
	}

	return &LoggingMiddleware{
		centralLogger: centralLogger,
		component:     component,
	}
}

// GinMiddleware returns a Gin middleware function for request logging
func (lm *LoggingMiddleware) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate or extract correlation ID
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Extract job ID from request if present
		var jobID *string
		if jid := c.GetHeader("X-Job-ID"); jid != "" {
			jobID = &jid
		}
		// Also check in URL path for job operations
		if strings.Contains(c.Request.URL.Path, "/jobs/") {
			pathParts := strings.Split(c.Request.URL.Path, "/")
			for i, part := range pathParts {
				if part == "jobs" && i+1 < len(pathParts) {
					jobID = &pathParts[i+1]
					break
				}
			}
		}

		// Add correlation ID to response headers
		c.Header("X-Correlation-ID", correlationID)
		if jobID != nil {
			c.Header("X-Job-ID", *jobID)
		}

		// Store in context for downstream use
		ctx := context.WithValue(c.Request.Context(), "correlation_id", correlationID)
		if jobID != nil {
			ctx = context.WithValue(ctx, "job_id", *jobID)
		}
		c.Request = c.Request.WithContext(ctx)

		// Log request start
		requestContext := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"query":      c.Request.URL.RawQuery,
			"user_agent": c.Request.UserAgent(),
			"remote_ip":  c.ClientIP(),
		}

		if lm.centralLogger != nil {
			lm.centralLogger.LogInfo(
				ctx,
				lm.component,
				"http_request",
				"Request received",
				&correlationID,
				jobID,
				requestContext,
			)
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		responseContext := map[string]interface{}{
			"method":        c.Request.Method,
			"path":          c.Request.URL.Path,
			"status_code":   c.Writer.Status(),
			"duration_ms":   duration.Milliseconds(),
			"response_size": c.Writer.Size(),
		}

		// Determine log level based on status code
		var logLevel log.Level
		var message string
		var err error

		statusCode := c.Writer.Status()
		switch {
		case statusCode >= 500:
			logLevel = log.ErrorLevel
			message = "Request failed with server error"
			if len(c.Errors) > 0 {
				err = c.Errors.Last()
			}
		case statusCode >= 400:
			logLevel = log.WarnLevel
			message = "Request failed with client error"
			if len(c.Errors) > 0 {
				err = c.Errors.Last()
			}
		case statusCode >= 300:
			logLevel = log.InfoLevel
			message = "Request redirected"
		default:
			logLevel = log.InfoLevel
			message = "Request completed successfully"
		}

		if lm.centralLogger != nil {
			lm.centralLogger.LogWithCorrelation(
				ctx,
				logLevel,
				lm.component,
				"http_response",
				message,
				&correlationID,
				jobID,
				responseContext,
				err,
				&duration,
			)
		}
	}
}

// HTTPMiddleware returns a standard HTTP middleware for non-Gin applications
func (lm *LoggingMiddleware) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate or extract correlation ID
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Extract job ID from request if present
		var jobID *string
		if jid := r.Header.Get("X-Job-ID"); jid != "" {
			jobID = &jid
		}

		// Add correlation ID to response headers
		w.Header().Set("X-Correlation-ID", correlationID)
		if jobID != nil {
			w.Header().Set("X-Job-ID", *jobID)
		}

		// Store in context for downstream use
		ctx := context.WithValue(r.Context(), "correlation_id", correlationID)
		if jobID != nil {
			ctx = context.WithValue(ctx, "job_id", *jobID)
		}
		r = r.WithContext(ctx)

		// Wrap response writer to capture status code and size
		wrappedWriter := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		// Log request start
		requestContext := map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"query":      r.URL.RawQuery,
			"user_agent": r.UserAgent(),
			"remote_ip":  getClientIP(r),
		}

		if lm.centralLogger != nil {
			lm.centralLogger.LogInfo(
				ctx,
				lm.component,
				"http_request",
				"Request received",
				&correlationID,
				jobID,
				requestContext,
			)
		}

		// Process request
		next.ServeHTTP(wrappedWriter, r)

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		responseContext := map[string]interface{}{
			"method":        r.Method,
			"path":          r.URL.Path,
			"status_code":   wrappedWriter.statusCode,
			"duration_ms":   duration.Milliseconds(),
			"response_size": wrappedWriter.size,
		}

		// Determine log level based on status code
		var message string
		statusCode := wrappedWriter.statusCode
		switch {
		case statusCode >= 500:
			message = "Request failed with server error"
		case statusCode >= 400:
			message = "Request failed with client error"
		case statusCode >= 300:
			message = "Request redirected"
		default:
			message = "Request completed successfully"
		}

		if lm.centralLogger != nil {
			lm.centralLogger.LogInfo(
				ctx,
				lm.component,
				"http_response",
				message,
				&correlationID,
				jobID,
				responseContext,
			)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(data)
	rw.size += size
	return size, err
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to remote address
	if ip := strings.Split(r.RemoteAddr, ":"); len(ip) > 0 {
		return ip[0]
	}

	return r.RemoteAddr
}

// ExtractCorrelationID extracts correlation ID from context
func ExtractCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}
	return ""
}

// ExtractJobID extracts job ID from context
func ExtractJobID(ctx context.Context) *string {
	if jobID, ok := ctx.Value("job_id").(string); ok {
		return &jobID
	}
	return nil
}

// CreateChildContext creates a new context with correlation for child operations
func CreateChildContext(parentCtx context.Context, operation string) (context.Context, string) {
	// Generate new correlation ID for child operation
	childCorrelationID := uuid.New().String()

	// Preserve parent correlation ID as parent_correlation_id
	if parentCorrelationID := ExtractCorrelationID(parentCtx); parentCorrelationID != "" {
		ctx := context.WithValue(parentCtx, "parent_correlation_id", parentCorrelationID)
		ctx = context.WithValue(ctx, "correlation_id", childCorrelationID)
		ctx = context.WithValue(ctx, "operation", operation)
		return ctx, childCorrelationID
	}

	// No parent correlation, create fresh context
	ctx := context.WithValue(parentCtx, "correlation_id", childCorrelationID)
	ctx = context.WithValue(ctx, "operation", operation)
	return ctx, childCorrelationID
}

// CorrelatedLogger wraps the central logger with automatic correlation extraction
type CorrelatedLogger struct {
	centralLogger *services.CentralLogger
	component     string
}

// NewCorrelatedLogger creates a logger that automatically extracts correlation from context
func NewCorrelatedLogger(centralLogger *services.CentralLogger, component string) *CorrelatedLogger {
	return &CorrelatedLogger{
		centralLogger: centralLogger,
		component:     component,
	}
}

// Info logs an info message with automatic correlation extraction
func (cl *CorrelatedLogger) Info(ctx context.Context, operation, message string, context map[string]interface{}) {
	correlationID := ExtractCorrelationID(ctx)
	jobID := ExtractJobID(ctx)

	var cid, jid *string
	if correlationID != "" {
		cid = &correlationID
	}
	if jobID != nil {
		jid = jobID
	}

	cl.centralLogger.LogInfo(ctx, cl.component, operation, message, cid, jid, context)
}

// Error logs an error message with automatic correlation extraction
func (cl *CorrelatedLogger) Error(ctx context.Context, operation, message string, context map[string]interface{}, err error) {
	correlationID := ExtractCorrelationID(ctx)
	jobID := ExtractJobID(ctx)

	var cid, jid *string
	if correlationID != "" {
		cid = &correlationID
	}
	if jobID != nil {
		jid = jobID
	}

	cl.centralLogger.LogError(ctx, cl.component, operation, message, cid, jid, context, err)
}

// Warning logs a warning message with automatic correlation extraction
func (cl *CorrelatedLogger) Warning(ctx context.Context, operation, message string, context map[string]interface{}, err error) {
	correlationID := ExtractCorrelationID(ctx)
	jobID := ExtractJobID(ctx)

	var cid, jid *string
	if correlationID != "" {
		cid = &correlationID
	}
	if jobID != nil {
		jid = jobID
	}

	cl.centralLogger.LogWarning(ctx, cl.component, operation, message, cid, jid, context, err)
}

// Debug logs a debug message with automatic correlation extraction
func (cl *CorrelatedLogger) Debug(ctx context.Context, operation, message string, context map[string]interface{}) {
	correlationID := ExtractCorrelationID(ctx)
	jobID := ExtractJobID(ctx)

	var cid, jid *string
	if correlationID != "" {
		cid = &correlationID
	}
	if jobID != nil {
		jid = jobID
	}

	cl.centralLogger.LogDebug(ctx, cl.component, operation, message, cid, jid, context)
}

// StartOperation logs the start of an operation with automatic correlation extraction
func (cl *CorrelatedLogger) StartOperation(ctx context.Context, operation, message string, context map[string]interface{}) *services.OperationTimer {
	correlationID := ExtractCorrelationID(ctx)
	jobID := ExtractJobID(ctx)

	var cid, jid *string
	if correlationID != "" {
		cid = &correlationID
	}
	if jobID != nil {
		jid = jobID
	}

	return cl.centralLogger.LogOperationStart(ctx, cl.component, operation, message, cid, jid, context)
}



