// Package middleware provides HTTP and CLI integration examples for the joblog package
package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/vexxhost/migratekit/internal/joblog"
)

// HTTPJobMiddleware provides HTTP middleware that creates a job per request
type HTTPJobMiddleware struct {
	tracker *joblog.Tracker
	config  *HTTPJobConfig
}

// HTTPJobConfig configures the HTTP job middleware
type HTTPJobConfig struct {
	// JobType is the job type for HTTP requests
	JobType string
	
	// OperationFromPath extracts operation name from request path
	OperationFromPath func(*http.Request) string
	
	// OwnerFromRequest extracts owner from request (e.g., from headers)
	OwnerFromRequest func(*http.Request) *string
	
	// MetadataFromRequest extracts metadata from request
	MetadataFromRequest func(*http.Request) map[string]any
	
	// SkipPaths are paths that should not create jobs
	SkipPaths []string
	
	// RecoverPanics enables panic recovery with job status updates
	RecoverPanics bool
}

// DefaultHTTPJobConfig returns a sensible default configuration
func DefaultHTTPJobConfig() *HTTPJobConfig {
	return &HTTPJobConfig{
		JobType: "http_request",
		OperationFromPath: func(r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		},
		OwnerFromRequest: func(r *http.Request) *string {
			if user := r.Header.Get("X-User-ID"); user != "" {
				return &user
			}
			if auth := r.Header.Get("Authorization"); auth != "" {
				// Extract user from Authorization header if needed
				return nil
			}
			return nil
		},
		MetadataFromRequest: func(r *http.Request) map[string]any {
			return map[string]any{
				"method":      r.Method,
				"path":        r.URL.Path,
				"query":       r.URL.RawQuery,
				"user_agent":  r.Header.Get("User-Agent"),
				"remote_addr": r.RemoteAddr,
			}
		},
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
		},
		RecoverPanics: true,
	}
}

// NewHTTPJobMiddleware creates a new HTTP job middleware
func NewHTTPJobMiddleware(tracker *joblog.Tracker, config *HTTPJobConfig) *HTTPJobMiddleware {
	if config == nil {
		config = DefaultHTTPJobConfig()
	}
	
	return &HTTPJobMiddleware{
		tracker: tracker,
		config:  config,
	}
}

// Middleware returns an HTTP middleware function
func (hjm *HTTPJobMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path should be skipped
		for _, skipPath := range hjm.config.SkipPaths {
			if r.URL.Path == skipPath {
				next.ServeHTTP(w, r)
				return
			}
		}
		
		// Start a job for this request
		jobStart := joblog.JobStart{
			JobType:   hjm.config.JobType,
			Operation: hjm.config.OperationFromPath(r),
			Owner:     hjm.config.OwnerFromRequest(r),
			Metadata:  hjm.config.MetadataFromRequest(r),
		}
		
		ctx, jobID, err := hjm.tracker.StartJob(r.Context(), jobStart)
		if err != nil {
			// Log error but don't fail the request
			slog.Error("Failed to start job for HTTP request",
				slog.String("error", err.Error()),
				slog.String("path", r.URL.Path),
			)
			next.ServeHTTP(w, r)
			return
		}
		
		// Add job ID to response headers for tracking
		w.Header().Set("X-Job-ID", jobID)
		
		// Create a response writer wrapper to capture status
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     200, // Default to 200
		}
		
		// Set up panic recovery if enabled
		if hjm.config.RecoverPanics {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					
					// Log the panic
					logger := hjm.tracker.Logger(ctx)
					logger.Error("Panic in HTTP request handler",
						slog.String("panic", fmt.Sprintf("%v", r)),
						slog.String("stack", string(stack)),
					)
					
					// End job with failure
					panicErr := fmt.Errorf("handler panic: %v", r)
					if endErr := hjm.tracker.EndJob(ctx, jobID, joblog.StatusFailed, panicErr); endErr != nil {
						logger.Error("Failed to end job after panic",
							slog.String("error", endErr.Error()),
						)
					}
					
					// Return 500 error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
		}
		
		// Execute the request with context
		start := time.Now()
		r = r.WithContext(ctx)
		next.ServeHTTP(wrapper, r)
		duration := time.Since(start)
		
		// Determine job status based on HTTP status code
		var status joblog.Status
		var jobErr error
		
		if wrapper.statusCode >= 200 && wrapper.statusCode < 400 {
			status = joblog.StatusCompleted
		} else {
			status = joblog.StatusFailed
			jobErr = fmt.Errorf("HTTP request failed with status %d", wrapper.statusCode)
		}
		
		// Log request completion
		logger := hjm.tracker.Logger(ctx)
		logger.Info("HTTP request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status_code", wrapper.statusCode),
			slog.Duration("duration", duration),
		)
		
		// End the job
		if err := hjm.tracker.EndJob(ctx, jobID, status, jobErr); err != nil {
			logger.Error("Failed to end job for HTTP request",
				slog.String("error", err.Error()),
			)
		}
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Example HTTP server setup
func ExampleHTTPServer() {
	// Database setup (use your actual DSN)
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(localhost:3306)/migratekit?parseTime=true"
	}
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer db.Close()
	
	// Create tracker
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	dbHandler := joblog.NewDBHandler(db, joblog.DefaultDBHandlerConfig())
	tracker := joblog.New(db, stdoutHandler, dbHandler)
	defer tracker.Close()
	
	// Create middleware
	jobMiddleware := NewHTTPJobMiddleware(tracker, DefaultHTTPJobConfig())
	
	// Create HTTP server
	mux := http.NewServeMux()
	
	// Health endpoint (skipped by middleware)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// API endpoints with job tracking
	mux.HandleFunc("/api/migrate", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := tracker.Logger(ctx)
		
		// Use RunStep for sub-operations
		err := tracker.RunStep(ctx, getJobIDFromContext(ctx), "validate-request", func(stepCtx context.Context) error {
			logger.Info("Validating migration request")
			time.Sleep(100 * time.Millisecond) // Simulate validation
			return nil
		})
		if err != nil {
			http.Error(w, "Validation failed", http.StatusBadRequest)
			return
		}
		
		err = tracker.RunStep(ctx, getJobIDFromContext(ctx), "queue-migration", func(stepCtx context.Context) error {
			logger.Info("Queueing migration job")
			time.Sleep(200 * time.Millisecond) // Simulate queueing
			return nil
		})
		if err != nil {
			http.Error(w, "Failed to queue migration", http.StatusInternalServerError)
			return
		}
		
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status": "migration queued"}`))
	})
	
	mux.HandleFunc("/api/failover", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := tracker.Logger(ctx)
		
		// Simulate a failing operation
		err := tracker.RunStep(ctx, getJobIDFromContext(ctx), "check-prerequisites", func(stepCtx context.Context) error {
			logger.Info("Checking failover prerequisites")
			time.Sleep(150 * time.Millisecond)
			return fmt.Errorf("VM not ready for failover")
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "failover initiated"}`))
	})
	
	// Apply middleware
	handler := jobMiddleware.Middleware(mux)
	
	slog.Info("Starting HTTP server with job tracking",
		slog.String("address", ":8080"),
	)
	
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}
	
	if err := server.ListenAndServe(); err != nil {
		slog.Error("HTTP server failed", slog.String("error", err.Error()))
	}
}

// Helper function to extract job ID from context (you'll need to implement this)
func getJobIDFromContext(ctx context.Context) string {
	if jobID, ok := joblog.JobIDFromCtx(ctx); ok {
		return jobID
	}
	return ""
}

// StepMiddleware provides middleware for creating steps within existing jobs
func StepMiddleware(tracker *joblog.Tracker, stepName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Get job ID from context
			jobID := getJobIDFromContext(ctx)
			if jobID == "" {
				// No job context, just pass through
				next.ServeHTTP(w, r)
				return
			}
			
			// Create step
			stepStart := joblog.StepStart{
				Name: stepName,
				Metadata: map[string]any{
					"endpoint": r.URL.Path,
					"method":   r.Method,
				},
			}
			
			stepCtx, stepID, err := tracker.StartStep(ctx, jobID, stepStart)
			if err != nil {
				// Log error but don't fail request
				slog.Error("Failed to start step",
					slog.String("error", err.Error()),
					slog.String("step_name", stepName),
				)
				next.ServeHTTP(w, r)
				return
			}
			
			// Response wrapper to capture status
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     200,
			}
			
			// Execute request
			start := time.Now()
			r = r.WithContext(stepCtx)
			next.ServeHTTP(wrapper, r)
			duration := time.Since(start)
			
			// End step based on HTTP status
			var stepStatus joblog.Status
			var stepErr error
			
			if wrapper.statusCode >= 200 && wrapper.statusCode < 400 {
				stepStatus = joblog.StatusCompleted
			} else {
				stepStatus = joblog.StatusFailed
				stepErr = fmt.Errorf("HTTP error %d", wrapper.statusCode)
			}
			
			// Log step completion
			logger := tracker.Logger(stepCtx)
			logger.Info("Step completed",
				slog.String("step_name", stepName),
				slog.Int("status_code", wrapper.statusCode),
				slog.Duration("duration", duration),
			)
			
			// End the step
			if err := tracker.EndStep(stepID, stepStatus, stepErr); err != nil {
				logger.Error("Failed to end step",
					slog.String("error", err.Error()),
				)
			}
		})
	}
}
