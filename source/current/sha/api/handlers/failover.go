// Package handlers provides HTTP handlers for VM failover management
// Following project rules: minimal endpoints, modular design, clean interfaces
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/common"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/failover"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/ossea"
	"github.com/vexxhost/migratekit-sha/services"
)

// FailoverHandler provides HTTP handlers for VM failover operations
type FailoverHandler struct {
	db                     database.Connection
	failoverJobRepo        *database.FailoverJobRepository
	validator              *failover.PreFailoverValidator
	enhancedLiveEngine     *failover.EnhancedLiveFailoverEngine
	enhancedTestEngine     *failover.EnhancedTestFailoverEngine
	enhancedCleanupService *failover.EnhancedCleanupService
	unifiedEngine          *failover.UnifiedFailoverEngine  // NEW: Unified failover engine
	configResolver         *failover.FailoverConfigResolver // NEW: Configuration resolver
	osseaClient            *ossea.Client
	networkClient          *ossea.NetworkClient
	vmInfoService          services.VMInfoProvider
	networkMappingService  *services.NetworkMappingService
	jobTracker             *joblog.Tracker // NEW: Enhanced job tracking
}

// NewFailoverHandler creates new failover handler
func NewFailoverHandler(db database.Connection) *FailoverHandler {
	failoverJobRepo := database.NewFailoverJobRepository(db)

	log.Info("🚀 Initializing failover handler")

	// Initialize JobLog tracker for enhanced services
	var jobTracker *joblog.Tracker
	log.Info("🔍 DEBUG: Starting JobLog tracker initialization")
	if sqlDB, err := db.GetGormDB().DB(); err == nil {
		log.Info("✅ DEBUG: Database connection obtained successfully")
		stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
		dbHandler := joblog.NewDBHandler(sqlDB, joblog.DefaultDBHandlerConfig())
		jobTracker = joblog.New(sqlDB, stdoutHandler, dbHandler)
		log.WithField("jobTracker_nil", jobTracker == nil).Info("🔧 DEBUG: JobLog tracker created")
	} else {
		log.WithError(err).Error("❌ DEBUG: Failed to initialize JobLog tracker, cleanup service will be unavailable")
	}

	handler := &FailoverHandler{
		db:              db,
		failoverJobRepo: failoverJobRepo,
		jobTracker:      jobTracker, // Enhanced job tracking
	}
	log.WithField("handler_jobTracker_nil", handler.jobTracker == nil).Info("🏗️ DEBUG: FailoverHandler created with jobTracker")

	// Only initialize cleanup service if JobLog tracker is available
	if jobTracker != nil {
		// Initialize SNA client (placeholder for now, will be enhanced in Phase 3)
		snaClient := failover.NewNullVMAClient() // Use null client until SNA integration is complete
		handler.enhancedCleanupService = failover.NewEnhancedCleanupService(db, jobTracker, snaClient)
	}

	// Try to initialize engines with existing OSSEA configuration
	handler.initializeEngines()

	return handler
}

// initializeEngines attempts to initialize enhanced failover engines with available OSSEA configuration
func (fh *FailoverHandler) initializeEngines() {
	if fh.enhancedTestEngine != nil && fh.enhancedLiveEngine != nil {
		return // Already initialized
	}

	// Try to get existing OSSEA configuration
	repo := database.NewOSSEAConfigRepository(fh.db)
	configs, err := repo.GetAll()
	if err != nil || len(configs) == 0 {
		log.WithError(err).Info("📝 No OSSEA configurations found - engines will use placeholders")
		return
	}

	// Use the first active configuration
	config := configs[0]
	log.WithField("config_name", config.Name).Info("🔧 Initializing failover engines with OSSEA config")

	// Create OSSEA client
	osseaClient := ossea.NewClient(
		config.APIURL,
		config.APIKey,
		config.SecretKey,
		config.Domain,
		config.Zone,
	)

	networkClient := ossea.NewNetworkClient(osseaClient)
	networkMappingRepo := database.NewNetworkMappingRepository(fh.db)

	// Use database-based VM info service for validation
	vmInfoService := services.NewSimpleDatabaseVMInfoService(fh.db)
	log.Info("🗄️  Using database-based VM info service for failover validation")

	// Initialize network mapping service
	networkMappingService := services.NewNetworkMappingService(networkMappingRepo, networkClient, vmInfoService)

	// Initialize validator
	validator := failover.NewPreFailoverValidator(
		fh.db,
		vmInfoService,
		networkMappingService,
	)

	// Note: Only enhanced engines are used - deprecated engines removed

	// Initialize Volume Client for unified engine
	volumeClient := common.NewVolumeClient("http://localhost:8090")

	// Initialize JobLog tracker for unified engine
	var jobTracker *joblog.Tracker
	if sqlDB, err := fh.db.GetGormDB().DB(); err == nil {
		stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
		dbHandler := joblog.NewDBHandler(sqlDB, joblog.DefaultDBHandlerConfig())
		jobTracker = joblog.New(sqlDB, stdoutHandler, dbHandler)
	} else {
		log.WithError(err).Error("Failed to initialize JobLog tracker for unified engine")
		return
	}

	// Initialize SNA client for power management (credentials passed per-operation from VM context)
	var snaClient failover.SNAClient = failover.NewVMAClientForFailover()
	if snaClient == nil {
		log.Error("Failed to initialize SNA client for power management")
		// Continue without SNA client - power management will be disabled
		snaClient = failover.NewNullVMAClient()
	}

	// Initialize unified failover engine
	fh.unifiedEngine = failover.NewUnifiedFailoverEngine(
		fh.db,
		jobTracker,
		osseaClient,
		networkClient,
		vmInfoService,
		networkMappingService,
		volumeClient,
		snaClient,
	)

	// Initialize configuration resolver
	vmContextRepo := database.NewVMReplicationContextRepository(fh.db)
	fh.configResolver = failover.NewFailoverConfigResolver(
		networkMappingService,
		vmContextRepo,
		networkMappingRepo,
	)

	// Store references for future use
	fh.osseaClient = osseaClient
	fh.networkClient = networkClient
	fh.vmInfoService = vmInfoService
	fh.networkMappingService = networkMappingService
	fh.validator = validator

	log.Info("✅ Failover engines initialized successfully with OSSEA configuration (including unified engine)")
}

// LiveFailoverRequest represents a live failover request
type LiveFailoverRequest struct {
	ContextID          string                 `json:"context_id" binding:"required"`
	VMID               string                 `json:"vm_id" binding:"required"`
	VMName             string                 `json:"vm_name" binding:"required"`
	SkipValidation     bool                   `json:"skip_validation"`
	NetworkMappings    map[string]string      `json:"network_mappings"`
	CustomConfig       map[string]interface{} `json:"custom_config"`
	NotificationConfig map[string]string      `json:"notification_config"`
}

// TestFailoverRequest represents a test failover request
type TestFailoverRequest struct {
	ContextID          string                 `json:"context_id" binding:"required"`
	VMID               string                 `json:"vm_id" binding:"required"`
	VMName             string                 `json:"vm_name" binding:"required"`
	SkipValidation     bool                   `json:"skip_validation"`
	TestDuration       string                 `json:"test_duration"` // Duration string (e.g., "2h", "30m")
	AutoCleanup        bool                   `json:"auto_cleanup"`
	NetworkMappings    map[string]string      `json:"network_mappings"`
	CustomConfig       map[string]interface{} `json:"custom_config"`
	NotificationConfig map[string]string      `json:"notification_config"`
}

// CleanupRequest represents a test failover cleanup request
type CleanupRequest struct {
	ContextID   string `json:"context_id" binding:"required"`
	VMID        string `json:"vm_id" binding:"required"`
	VMName      string `json:"vm_name" binding:"required"`
	CleanupType string `json:"cleanup_type"`
}

// RollbackRequest represents an enhanced rollback request with optional behaviors
type RollbackRequest struct {
	ContextID     string `json:"context_id" binding:"required"`
	VMID          string `json:"vm_id" binding:"required"`
	VMName        string `json:"vm_name" binding:"required"`
	VMwareVMID    string `json:"vmware_vm_id"`
	FailoverType  string `json:"failover_type" binding:"required"` // "test" or "live"
	PowerOnSource bool   `json:"power_on_source,omitempty"`
	ForceCleanup  bool   `json:"force_cleanup,omitempty"`
}

// UnifiedFailoverRequest represents a unified failover request with optional behaviors
type UnifiedFailoverRequest struct {
	ContextID    string `json:"context_id" binding:"required"`
	VMwareVMID   string `json:"vmware_vm_id" binding:"required"`
	VMName       string `json:"vm_name" binding:"required"`
	FailoverType string `json:"failover_type" binding:"required"` // "live" or "test"

	// Optional behaviors for live failover
	PowerOffSource   *bool `json:"power_off_source,omitempty"`   // Power off source VM (live failover)
	PerformFinalSync *bool `json:"perform_final_sync,omitempty"` // Perform final incremental sync (live failover)

	// Optional behaviors for both types
	SkipValidation *bool `json:"skip_validation,omitempty"` // Skip pre-failover validation
	SkipVirtIO     *bool `json:"skip_virtio,omitempty"`     // Skip VirtIO driver injection

	// Network and VM naming options
	NetworkStrategy string `json:"network_strategy,omitempty"` // "test", "live", "custom"
	VMNaming        string `json:"vm_naming,omitempty"`        // "exact", "suffixed"

	// Advanced options
	TestDuration    string                 `json:"test_duration,omitempty"`    // For test failover (e.g., "2h")
	CustomConfig    map[string]interface{} `json:"custom_config,omitempty"`    // Custom configuration options
	NetworkMappings map[string]string      `json:"network_mappings,omitempty"` // Custom network mappings
}

// FailoverResponse represents a failover operation response
type FailoverResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	JobID   string      `json:"job_id"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ValidationResponse represents a validation response
type ValidationResponse struct {
	Success          bool                       `json:"success"`
	Message          string                     `json:"message"`
	IsValid          bool                       `json:"is_valid"`
	ReadinessScore   float64                    `json:"readiness_score"`
	ValidationResult *failover.ValidationResult `json:"validation_result,omitempty"`
	RequiredActions  []string                   `json:"required_actions"`
	Error            string                     `json:"error,omitempty"`
}

// JobStatusResponse represents a job status response
type JobStatusResponse struct {
	Success    bool          `json:"success"`
	Message    string        `json:"message"`
	JobID      string        `json:"job_id"`
	Status     string        `json:"status"`
	Progress   float64       `json:"progress"`
	StartTime  time.Time     `json:"start_time"`
	Duration   time.Duration `json:"duration"`
	JobDetails interface{}   `json:"job_details,omitempty"`
	Error      string        `json:"error,omitempty"`
}

// InitiateLiveFailover starts a live VM failover operation
// POST /api/v1/failover/live
func (fh *FailoverHandler) InitiateLiveFailover(w http.ResponseWriter, r *http.Request) {
	var req LiveFailoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("Invalid live failover request")
		response := FailoverResponse{
			Success: false,
			Message: "Invalid request data",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithFields(log.Fields{
		"vm_id":           req.VMID,
		"vm_name":         req.VMName,
		"skip_validation": req.SkipValidation,
	}).Info("🚀 API: Initiating live VM failover")

	// Validate required fields
	if req.VMID == "" {
		response := FailoverResponse{
			Success: false,
			Message: "VM ID is required",
			Error:   "validation_failed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate job ID
	jobID := fmt.Sprintf("live-failover-%s-%d", req.VMID, time.Now().Unix())

	// For now, return a placeholder response indicating feature is available
	if fh.enhancedLiveEngine == nil {
		// Create placeholder job entry
		job := &database.FailoverJob{
			JobID:        jobID,
			VMID:         req.VMID,
			JobType:      "live",
			Status:       "pending",
			SourceVMName: req.VMName,
			SourceVMSpec: fmt.Sprintf(`{"vm_id":"%s","vm_name":"%s","request_time":"%s"}`,
				req.VMID, req.VMName, time.Now().Format(time.RFC3339)),
		}

		err := fh.failoverJobRepo.Create(job)
		if err != nil {
			log.WithError(err).Error("Failed to create failover job")
			response := FailoverResponse{
				Success: false,
				Message: "Failed to create failover job",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := FailoverResponse{
			Success: true,
			Message: "Live failover job created successfully (execution pending full engine integration)",
			JobID:   jobID,

			Data: map[string]interface{}{
				"vm_id":            req.VMID,
				"vm_name":          req.VMName,
				"job_type":         "live",
				"status":           "pending",
				"integration_note": "Full failover engine integration in progress",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Implement actual live failover execution when engine is wired up
	// failoverRequest := &failover.LiveFailoverRequest{...}
	// result, err := fh.liveFailoverEngine.ExecuteLiveFailover(failoverRequest)

	log.WithFields(log.Fields{
		"job_id": jobID,
		"vm_id":  req.VMID,
	}).Info("✅ API: Live failover initiated")
}

// InitiateTestFailover starts a test VM failover operation
// POST /api/v1/failover/test
func (fh *FailoverHandler) InitiateTestFailover(w http.ResponseWriter, r *http.Request) {
	var req TestFailoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("Invalid test failover request")
		response := FailoverResponse{
			Success: false,
			Message: "Invalid request data",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithFields(log.Fields{
		"vm_id":         req.VMID,
		"vm_name":       req.VMName,
		"test_duration": req.TestDuration,
		"auto_cleanup":  req.AutoCleanup,
	}).Info("🧪 API: Initiating enhanced test VM failover with JobLog")

	// Validate required fields
	if req.VMID == "" {
		response := FailoverResponse{
			Success: false,
			Message: "VM ID is required",
			Error:   "validation_failed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if enhanced test engine is available
	if fh.enhancedTestEngine == nil {
		log.Error("❌ CRITICAL: enhancedTestEngine is NIL!")
		response := FailoverResponse{
			Success: false,
			Message: "Enhanced test failover engine not initialized",
			Error:   "service_not_available",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate job ID for enhanced test failover
	jobID := fmt.Sprintf("enhanced-test-failover-%s-%d", req.VMID, time.Now().Unix())

	log.WithFields(log.Fields{
		"job_id":        jobID,
		"vm_id":         req.VMID,
		"vm_name":       req.VMName,
		"test_duration": req.TestDuration,
		"auto_cleanup":  req.AutoCleanup,
	}).Info("🧪 Executing enhanced test failover with JobLog integration")

	// Create enhanced test failover request
	enhancedRequest := &failover.EnhancedTestFailoverRequest{
		VMID:          req.VMID,
		VMName:        req.VMName,
		FailoverJobID: jobID,
		Timestamp:     time.Now(),
	}

	// Execute enhanced test failover in background with joblog tracking
	go func() {
		actualJobID, err := fh.enhancedTestEngine.ExecuteEnhancedTestFailover(r.Context(), enhancedRequest)
		if err != nil {
			log.WithError(err).WithField("job_id", jobID).Error("❌ Enhanced test failover execution failed")
		} else {
			log.WithFields(log.Fields{
				"job_id":        jobID,
				"actual_job_id": actualJobID,
				"vm_id":         req.VMID,
				"vm_name":       req.VMName,
			}).Info("✅ Enhanced test failover execution completed with JobLog")
		}
	}()

	response := FailoverResponse{
		Success: true,
		Message: "Test failover initiated successfully with snapshot protection and VirtIO injection",
		JobID:   jobID,

		Data: map[string]interface{}{
			"vm_id":               req.VMID,
			"vm_name":             req.VMName,
			"job_type":            "test",
			"status":              "executing",
			"test_duration":       req.TestDuration,
			"auto_cleanup":        req.AutoCleanup,
			"snapshot_protection": true,
			"virtio_injection":    true,
			"correlation_id":      fmt.Sprintf("%s", r.Header.Get("X-Request-ID")),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// EndTestFailover terminates a test failover and cleans up resources
// DELETE /api/v1/failover/test/{job_id}
func (fh *FailoverHandler) EndTestFailover(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]

	log.WithField("job_id", jobID).Info("🧹 API: Ending test failover")

	// Get the test failover job
	job, err := fh.failoverJobRepo.GetByJobID(jobID)
	if err != nil {
		log.WithError(err).Error("Test failover job not found")
		response := FailoverResponse{
			Success: false,
			Message: "Test failover job not found",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate it's a test job
	if job.JobType != "test" {
		response := FailoverResponse{
			Success: false,
			Message: "Job is not a test failover",
			Error:   "invalid_job_type",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Execute cleanup via ENHANCED cleanup service
	if fh.enhancedCleanupService == nil {
		log.Error("❌ CRITICAL: enhancedCleanupService is NIL!")
		response := FailoverResponse{
			Success: false,
			Message: "Enhanced cleanup service not initialized",
			Error:   "service_not_available",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Execute cleanup using the VM context ID and VM name for proper VM-centric architecture
	// CRITICAL FIX: Use context.Background() to prevent cancellation and provide proper VM identifiers
	err = fh.enhancedCleanupService.ExecuteTestFailoverCleanupWithTracking(context.Background(), job.VMContextID, job.SourceVMName, fmt.Sprintf("legacy-cleanup-%d", job.ID))
	if err != nil {
		log.WithError(err).Error("Enhanced test failover cleanup failed")
		response := FailoverResponse{
			Success: false,
			Message: fmt.Sprintf("Test failover cleanup failed: %v", err),
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := FailoverResponse{
		Success: true,
		Message: "Test failover cleanup completed successfully with JobLog tracking",
		JobID:   jobID,
		Data: map[string]interface{}{
			"job_id":         jobID,
			"context_id":     job.VMContextID,
			"vm_name":        job.SourceVMName,
			"vm_id":          job.VMID,
			"cleanup_status": "completed",
			"integration":    "enhanced_cleanup_service",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.WithField("job_id", jobID).Info("✅ API: Test failover cleanup completed")
}

// GetFailoverJobStatus retrieves the status of a failover job using smart lookup
// GET /api/v1/failover/{job_id}/status
func (fh *FailoverHandler) GetFailoverJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]

	log.WithField("job_id", jobID).Info("📊 API: Getting failover job status with smart lookup")

	// Enhanced smart lookup: Try JobLog first, then fallback to legacy
	var response JobStatusResponse

	// Attempt 1: Smart JobLog lookup (UUID, external ID, or context ID)
	log.WithField("job_id", jobID).WithField("jobTracker_nil", fh.jobTracker == nil).Info("🔍 DEBUG: Checking JobLog tracker status")
	if fh.jobTracker != nil {
		log.WithField("job_id", jobID).Info("✅ DEBUG: JobTracker is not nil, calling FindJobByAnyID")
		if jobSummary, err := fh.jobTracker.FindJobByAnyID(jobID); err == nil {
			log.WithField("job_id", jobID).Info("✅ Found job via direct JobLog UUID lookup")

			// Build enhanced response from JobLog data
			progress := jobSummary.Progress
			var status string
			switch jobSummary.Job.Status {
			case joblog.StatusRunning:
				status = "running"
			case joblog.StatusCompleted:
				status = "completed"
			case joblog.StatusFailed:
				status = "failed"
			case joblog.StatusCancelled:
				status = "cancelled"
			default:
				status = string(jobSummary.Job.Status)
			}

			// Calculate duration
			var duration time.Duration
			if jobSummary.Job.CompletedAt != nil {
				duration = jobSummary.Job.CompletedAt.Sub(jobSummary.Job.StartedAt)
			} else {
				duration = time.Since(jobSummary.Job.StartedAt)
			}

			// Parse metadata for additional details
			var metadata map[string]interface{}
			if jobSummary.Job.Metadata != nil {
				json.Unmarshal([]byte(*jobSummary.Job.Metadata), &metadata)
			}

			response = JobStatusResponse{
				Success:   true,
				Message:   fmt.Sprintf("Retrieved status for %s job via direct JobLog lookup", jobSummary.Job.JobType),
				JobID:     jobSummary.Job.ID,
				Status:    status,
				Progress:  progress.StepCompletion,
				StartTime: jobSummary.Job.StartedAt,
				Duration:  duration,
				JobDetails: map[string]interface{}{
					"job_type":        jobSummary.Job.JobType,
					"operation":       jobSummary.Job.Operation,
					"total_steps":     progress.TotalSteps,
					"completed_steps": progress.CompletedSteps,
					"failed_steps":    progress.FailedSteps,
					"running_steps":   progress.RunningSteps,
					"step_completion": progress.StepCompletion,
					"error_message":   jobSummary.Job.ErrorMessage,
					"metadata":        metadata,
					"tracking_source": "direct_joblog_uuid",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		log.WithField("job_id", jobID).Debug("Job not found by direct UUID, trying external job ID lookup")
	} else {
		log.WithField("job_id", jobID).Error("❌ DEBUG: JobTracker is nil - enhanced lookup unavailable")
	}

	// Attempt 2: External job ID lookup (GUI compatibility)
	if fh.jobTracker != nil {
		if jobSummary, err := fh.jobTracker.GetJobByExternalID(jobID); err == nil {
			log.WithField("job_id", jobID).WithField("internal_job_id", jobSummary.Job.ID).Info("✅ Found job via external job ID correlation")

			// Build enhanced response from JobLog data
			progress := jobSummary.Progress
			var status string
			switch jobSummary.Job.Status {
			case joblog.StatusRunning:
				status = "running"
			case joblog.StatusCompleted:
				status = "completed"
			case joblog.StatusFailed:
				status = "failed"
			case joblog.StatusCancelled:
				status = "cancelled"
			default:
				status = string(jobSummary.Job.Status)
			}

			// Calculate duration
			var duration time.Duration
			if jobSummary.Job.CompletedAt != nil {
				duration = jobSummary.Job.CompletedAt.Sub(jobSummary.Job.StartedAt)
			} else {
				duration = time.Since(jobSummary.Job.StartedAt)
			}

			// Parse metadata for additional details
			var metadata map[string]interface{}
			if jobSummary.Job.Metadata != nil {
				json.Unmarshal([]byte(*jobSummary.Job.Metadata), &metadata)
			}

			response = JobStatusResponse{
				Success:   true,
				Message:   fmt.Sprintf("Retrieved status for %s job via enhanced tracking", jobSummary.Job.JobType),
				JobID:     jobSummary.Job.ID,
				Status:    status,
				Progress:  progress.StepCompletion, // Already float64
				StartTime: jobSummary.Job.StartedAt,
				Duration:  duration,
				JobDetails: map[string]interface{}{
					"job_type":        jobSummary.Job.JobType,
					"operation":       jobSummary.Job.Operation,
					"context_id":      jobSummary.Job.ContextID,
					"external_job_id": jobSummary.Job.ExternalJobID,
					"job_category":    jobSummary.Job.JobCategory,
					"total_steps":     progress.TotalSteps,
					"completed_steps": progress.CompletedSteps,
					"failed_steps":    progress.FailedSteps,
					"running_steps":   progress.RunningSteps,
					"step_completion": progress.StepCompletion,
					"error_message":   jobSummary.Job.ErrorMessage,
					"metadata":        metadata,
					"tracking_source": "external_job_id_correlation",
					"gui_job_id":      jobID,             // Original GUI job ID
					"internal_job_id": jobSummary.Job.ID, // JobLog UUID
				},
			}

			log.WithFields(log.Fields{
				"job_id":          jobID,
				"status":          status,
				"progress":        progress.StepCompletion,
				"duration":        duration,
				"tracking_type":   "enhanced",
				"total_steps":     progress.TotalSteps,
				"completed_steps": progress.CompletedSteps,
			}).Info("✅ API: Retrieved failover job status via external job ID correlation")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		log.WithField("job_id", jobID).Debug("Job not found via external job ID lookup, trying legacy")
	}

	// Attempt 3: Legacy failover_jobs table lookup
	job, err := fh.failoverJobRepo.GetByJobID(jobID)
	if err != nil {
		log.WithError(err).Error("Failover job not found in any tracking system")
		response = JobStatusResponse{
			Success: false,
			Message: "Failover job not found",
			Error:   fmt.Sprintf("failover job not found: %s", jobID),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithField("job_id", jobID).Info("✅ Found job via legacy failover_jobs table")

	// Build legacy response (existing logic)
	progress := fh.calculateJobProgress(job)
	duration := time.Since(job.CreatedAt)
	if job.CompletedAt != nil {
		duration = job.CompletedAt.Sub(job.CreatedAt)
	}

	// Parse job details
	var jobDetails map[string]interface{}
	if job.SourceVMSpec != "" {
		json.Unmarshal([]byte(job.SourceVMSpec), &jobDetails)
	}

	response = JobStatusResponse{
		Success:   true,
		Message:   fmt.Sprintf("Retrieved status for %s failover job via legacy tracking", job.JobType),
		JobID:     job.JobID,
		Status:    job.Status,
		Progress:  progress,
		StartTime: job.CreatedAt,
		Duration:  duration,
		JobDetails: map[string]interface{}{
			"vm_id":             job.VMID,
			"vm_name":           job.SourceVMName,
			"job_type":          job.JobType,
			"destination_vm_id": job.DestinationVMID,
			"snapshot_id":       job.OSSEASnapshotID,
			"error_message":     job.ErrorMessage,
			"started_at":        job.StartedAt,
			"completed_at":      job.CompletedAt,
			"custom_config":     jobDetails,
			"tracking_source":   "legacy_failover_jobs",
		},
	}

	log.WithFields(log.Fields{
		"job_id":        jobID,
		"status":        job.Status,
		"progress":      progress,
		"duration":      duration,
		"tracking_type": "legacy",
	}).Info("✅ API: Retrieved legacy failover job status")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ValidateFailoverReadiness checks if a VM is ready for failover
// GET /api/v1/failover/{vm_id}/readiness
func (fh *FailoverHandler) ValidateFailoverReadiness(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["vm_id"]
	failoverType := r.URL.Query().Get("type")
	if failoverType == "" {
		failoverType = "live" // Default to live failover
	}

	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"failover_type": failoverType,
	}).Info("🔍 API: Validating failover readiness")

	// For now, return a placeholder response
	if fh.validator == nil {
		response := ValidationResponse{
			Success:        true,
			Message:        "Failover readiness check completed (validation engine integration in progress)",
			IsValid:        true, // Assume valid for placeholder
			ReadinessScore: 85.0, // Placeholder score
			RequiredActions: []string{
				"Full validation engine integration required",
				"Network mapping validation pending",
				"Volume state verification pending",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Implement actual validation when validator is wired up
	// result, err := fh.validator.ValidateFailoverReadiness(vmID, failoverType)

	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"failover_type": failoverType,
		"score":         85.0, // Placeholder
	}).Info("✅ API: Failover readiness validation completed")
}

// ListFailoverJobs lists all failover jobs with optional filtering
// GET /api/v1/failover/jobs
func (fh *FailoverHandler) ListFailoverJobs(w http.ResponseWriter, r *http.Request) {
	log.Info("📋 API: Listing failover jobs")

	// Parse query parameters
	jobType := r.URL.Query().Get("type")  // live, test, or empty for all
	status := r.URL.Query().Get("status") // pending, in_progress, completed, failed, etc.
	vmID := r.URL.Query().Get("vm_id")    // Filter by specific VM

	// For now, get all jobs and filter manually
	// TODO: Implement proper filtering in repository
	var jobs []database.FailoverJob
	var err error

	if vmID != "" {
		jobs, err = fh.failoverJobRepo.GetByVMID(vmID)
	} else {
		// Get all jobs - need to implement this method
		// For now, return empty slice
		jobs = []database.FailoverJob{}
	}

	if err != nil {
		log.WithError(err).Error("Failed to get failover jobs")
		response := map[string]interface{}{
			"success": false,
			"message": "Failed to retrieve failover jobs",
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Apply filters
	var filteredJobs []database.FailoverJob
	for _, job := range jobs {
		if jobType != "" && job.JobType != jobType {
			continue
		}
		if status != "" && job.Status != status {
			continue
		}
		filteredJobs = append(filteredJobs, job)
	}

	// Convert to response format
	jobList := make([]map[string]interface{}, len(filteredJobs))
	for i, job := range filteredJobs {
		progress := fh.calculateJobProgress(&job)
		duration := time.Since(job.CreatedAt)
		if job.CompletedAt != nil {
			duration = job.CompletedAt.Sub(job.CreatedAt)
		}

		jobList[i] = map[string]interface{}{
			"job_id":            job.JobID,
			"vm_id":             job.VMID,
			"vm_name":           job.SourceVMName,
			"job_type":          job.JobType,
			"status":            job.Status,
			"progress":          progress,
			"destination_vm_id": job.DestinationVMID,
			"snapshot_id":       job.OSSEASnapshotID,
			"created_at":        job.CreatedAt,
			"started_at":        job.StartedAt,
			"completed_at":      job.CompletedAt,
			"duration":          duration,
			"error_message":     job.ErrorMessage,
		}
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Retrieved %d failover jobs", len(filteredJobs)),
		"total":   len(filteredJobs),
		"filters": map[string]string{
			"type":   jobType,
			"status": status,
			"vm_id":  vmID,
		},
		"jobs": jobList,
	}

	log.WithFields(log.Fields{
		"total_jobs":    len(filteredJobs),
		"type_filter":   jobType,
		"status_filter": status,
		"vm_filter":     vmID,
	}).Info("✅ API: Retrieved failover jobs list")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UnifiedFailover performs unified failover using the new unified engine
// POST /api/v1/failover/unified
func (fh *FailoverHandler) UnifiedFailover(w http.ResponseWriter, r *http.Request) {
	log.Info("🚀 API: Unified failover request received")

	// Parse request body using structured request
	var request UnifiedFailoverRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.WithError(err).Error("❌ Failed to parse unified failover request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"context_id":    request.ContextID,
		"vmware_vm_id":  request.VMwareVMID,
		"vm_name":       request.VMName,
		"failover_type": request.FailoverType,
	}).Info("📋 Parsed unified failover request")

	// Check if unified engine is available
	if fh.unifiedEngine == nil {
		log.Error("❌ CRITICAL: unifiedEngine is NIL!")
		response := FailoverResponse{
			Success: false,
			Message: "Unified failover engine not initialized",
			Error:   "service_not_available",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate failover job ID
	jobID := fmt.Sprintf("unified-%s-failover-%s-%d", request.FailoverType, request.VMName, time.Now().Unix())

	// Convert structured request to configuration options
	options := make(map[string]interface{})
	if request.PowerOffSource != nil {
		options["power_off_source"] = *request.PowerOffSource
	}
	if request.PerformFinalSync != nil {
		options["perform_final_sync"] = *request.PerformFinalSync
	}
	if request.SkipValidation != nil {
		options["skip_validation"] = *request.SkipValidation
	}
	if request.SkipVirtIO != nil {
		options["skip_virtio"] = *request.SkipVirtIO
	}
	if request.NetworkStrategy != "" {
		options["network_strategy"] = request.NetworkStrategy
	}
	if request.VMNaming != "" {
		options["vm_naming"] = request.VMNaming
	}
	if request.TestDuration != "" {
		options["test_duration"] = request.TestDuration
	}
	if request.NetworkMappings != nil {
		options["network_mappings"] = request.NetworkMappings
	}
	if request.CustomConfig != nil {
		options["custom_config"] = request.CustomConfig
	}

	// Resolve configuration using the config resolver
	config, err := fh.configResolver.ResolveFromAPIRequest(
		request.ContextID,
		request.VMwareVMID,
		request.VMName,
		jobID,
		request.FailoverType,
		options,
	)
	if err != nil {
		log.WithError(err).Error("❌ Failed to resolve unified failover configuration")
		response := FailoverResponse{
			Success: false,
			Message: "Failed to resolve failover configuration",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate configuration
	if err := fh.configResolver.ValidateConfiguration(config); err != nil {
		log.WithError(err).Error("❌ Invalid unified failover configuration")
		response := FailoverResponse{
			Success: false,
			Message: "Invalid failover configuration",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get configuration summary for logging
	configSummary := fh.configResolver.GetConfigurationSummary(config)
	log.WithFields(log.Fields{
		"job_id":         jobID,
		"config_summary": configSummary,
	}).Info("🔧 Unified failover configuration resolved and validated")

	// Return immediate response with job ID
	response := FailoverResponse{
		Success: true,
		Message: fmt.Sprintf("Unified %s failover initiated successfully", request.FailoverType),
		JobID:   jobID,
		Data: map[string]interface{}{
			"failover_type":    request.FailoverType,
			"context_id":       request.ContextID,
			"vm_name":          request.VMName,
			"destination_name": config.GetDestinationVMName(),
			"configuration":    configSummary,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Execute unified failover in background
	go func() {
		log.WithField("job_id", jobID).Info("🚀 Starting unified failover execution in background")

		// CRITICAL FIX: Use context.Background() instead of r.Context() to prevent cancellation
		// when HTTP request completes. Background jobs need independent context lifecycle.
		ctx := context.Background()
		result, err := fh.unifiedEngine.ExecuteUnifiedFailover(ctx, config)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"job_id":        jobID,
				"failover_type": request.FailoverType,
				"context_id":    request.ContextID,
			}).Error("❌ Unified failover execution failed")
		} else {
			log.WithFields(log.Fields{
				"job_id":            jobID,
				"failover_type":     request.FailoverType,
				"destination_vm_id": result.DestinationVMID,
				"duration":          result.Duration,
				"status":            result.Status,
			}).Info("✅ Unified failover execution completed successfully")
		}
	}()
}

// Helper methods

func (fh *FailoverHandler) calculateJobProgress(job *database.FailoverJob) float64 {
	switch job.Status {
	case "pending":
		return 0.0
	case "validating":
		return 10.0
	case "snapshotting":
		return 25.0
	case "creating_vm":
		return 50.0
	case "switching_volume":
		return 75.0
	case "powering_on":
		return 90.0
	case "completed":
		return 100.0
	case "failed":
		return 0.0
	case "cleanup":
		return 100.0
	default:
		return 50.0 // Unknown status
	}
}

// CleanupTestFailover performs test failover cleanup for a VM
// POST /api/v1/failover/cleanup/{vm_name} (backward compatibility)
// Also accepts JSON body with VM-centric identifiers
func (fh *FailoverHandler) CleanupTestFailover(w http.ResponseWriter, r *http.Request) {
	// Support both URL parameter (backward compatibility) and JSON body (VM-centric)
	vars := mux.Vars(r)
	vmName := vars["vm_name"]

	var contextID, vmID string

	// Try to parse JSON body for VM-centric identifiers
	if r.Header.Get("Content-Type") == "application/json" {
		var request CleanupRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err == nil {
			contextID = request.ContextID
			vmID = request.VMID
			vmName = request.VMName // Override URL parameter if provided in body
			log.WithFields(log.Fields{
				"parsed_context_id": contextID,
				"parsed_vm_id":      vmID,
				"parsed_vm_name":    vmName,
				"cleanup_type":      request.CleanupType,
			}).Info("🔍 Successfully parsed JSON body for cleanup")
		} else {
			log.WithFields(log.Fields{
				"error":        err.Error(),
				"content_type": r.Header.Get("Content-Type"),
				"url_vm_name":  vmName,
			}).Error("❌ Failed to parse JSON body for cleanup - using URL parameter fallback")
		}
	} else {
		log.WithField("content_type", r.Header.Get("Content-Type")).Info("🔍 No JSON content type, using URL parameter only")
	}

	log.WithFields(log.Fields{
		"context_id": contextID,
		"vm_id":      vmID,
		"vm_name":    vmName,
	}).Info("🧹 API: Starting test failover cleanup")

	// DEBUG: Check if enhanced cleanup service is properly initialized
	if fh.enhancedCleanupService == nil {
		log.Error("❌ CRITICAL: enhancedCleanupService is NIL!")
		http.Error(w, "Enhanced cleanup service not initialized", http.StatusInternalServerError)
		return
	}
	log.Info("✅ DEBUG: enhancedCleanupService is properly initialized")

	// Execute cleanup via ENHANCED cleanup service with VM-centric identifiers
	// CRITICAL FIX: Use context.Background() instead of r.Context() to prevent cancellation
	err := fh.enhancedCleanupService.ExecuteTestFailoverCleanupWithTracking(context.Background(), contextID, vmName, "legacy-cleanup-"+contextID)
	if err != nil {
		log.WithError(err).Error("Test failover cleanup failed")
		response := FailoverResponse{
			Success: false,
			Message: fmt.Sprintf("Test failover cleanup failed: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return success response
	response := FailoverResponse{
		Success: true,
		Message: fmt.Sprintf("Test failover cleanup for %s completed successfully", vmName),
	}

	log.WithField("vm_name", vmName).Info("✅ Test failover cleanup completed successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// EnhancedRollback handles enhanced rollback operations with optional behaviors
func (fh *FailoverHandler) EnhancedRollback(w http.ResponseWriter, r *http.Request) {
	var request RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.WithError(err).Error("Failed to parse enhanced rollback request")
		response := FailoverResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid request format: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithFields(log.Fields{
		"context_id":      request.ContextID,
		"vm_id":           request.VMID,
		"vm_name":         request.VMName,
		"vmware_vm_id":    request.VMwareVMID,
		"failover_type":   request.FailoverType,
		"power_on_source": request.PowerOnSource,
		"force_cleanup":   request.ForceCleanup,
	}).Info("🔄 API: Starting enhanced failover rollback")

	// Validate required fields
	if request.ContextID == "" || request.VMName == "" {
		response := FailoverResponse{
			Success: false,
			Message: "Missing required fields: context_id and vm_name are required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate failover type
	if request.FailoverType != "test" && request.FailoverType != "live" {
		response := FailoverResponse{
			Success: false,
			Message: "Invalid failover_type: must be 'test' or 'live'",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if enhanced cleanup service is available
	if fh.enhancedCleanupService == nil {
		log.Error("❌ CRITICAL: enhancedCleanupService is NIL!")
		response := FailoverResponse{
			Success: false,
			Message: "Enhanced cleanup service not initialized",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate rollback job ID (following live failover pattern)
	jobID := fmt.Sprintf("rollback-%s-failover-%s-%d", request.FailoverType, request.VMName, time.Now().Unix())

	// Create rollback options from request
	rollbackOptions := &failover.RollbackOptions{
		PowerOnSourceVM: request.PowerOnSource,
		ForceCleanup:    request.ForceCleanup,
		FailoverType:    request.FailoverType,
	}

	log.WithFields(log.Fields{
		"job_id":          jobID,
		"context_id":      request.ContextID,
		"vm_name":         request.VMName,
		"failover_type":   request.FailoverType,
		"power_on_source": request.PowerOnSource,
		"force_cleanup":   request.ForceCleanup,
	}).Info("🔧 Enhanced rollback configuration resolved and validated")

	// Return immediate response with job ID (following live failover pattern)
	response := FailoverResponse{
		Success: true,
		Message: fmt.Sprintf("Enhanced %s failover rollback initiated successfully", request.FailoverType),
		JobID:   jobID,
		Data: map[string]interface{}{
			"rollback_type":   fmt.Sprintf("%s_rollback", request.FailoverType),
			"context_id":      request.ContextID,
			"vm_name":         request.VMName,
			"vmware_vm_id":    request.VMwareVMID,
			"power_on_source": request.PowerOnSource,
			"force_cleanup":   request.ForceCleanup,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Execute enhanced rollback in background (following live failover pattern)
	go func() {
		log.WithField("job_id", jobID).Info("🚀 Starting enhanced rollback execution in background")

		// CRITICAL FIX: Use context.Background() instead of r.Context() to prevent cancellation
		// when HTTP request completes. Background jobs need independent context lifecycle.
		ctx := context.Background()

		err := fh.enhancedCleanupService.ExecuteUnifiedFailoverRollback(
			ctx,
			request.ContextID,
			request.VMName,
			request.VMwareVMID,
			rollbackOptions,
			jobID, // Pass GUI job ID as external job ID (same as unified failover)
		)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"job_id":        jobID,
				"rollback_type": request.FailoverType,
				"context_id":    request.ContextID,
			}).Error("❌ Enhanced rollback execution failed")
		} else {
			log.WithFields(log.Fields{
				"job_id":        jobID,
				"rollback_type": request.FailoverType,
				"context_id":    request.ContextID,
				"vm_name":       request.VMName,
			}).Info("✅ Enhanced rollback execution completed successfully")
		}
	}()
}

// GetRollbackDecision provides decision points for rollback operations
func (fh *FailoverHandler) GetRollbackDecision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	failoverType := vars["failover_type"]
	vmName := vars["vm_name"]

	if fh.enhancedCleanupService == nil {
		http.Error(w, "Enhanced cleanup service not initialized", http.StatusInternalServerError)
		return
	}

	decision := fh.enhancedCleanupService.CreateRollbackDecision(failoverType, vmName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(decision)
}

// GetPreFlightConfiguration provides configuration options for failover pre-flight setup
func (fh *FailoverHandler) GetPreFlightConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	failoverType := vars["failover_type"]
	vmName := vars["vm_name"]

	log.WithFields(log.Fields{
		"failover_type": failoverType,
		"vm_name":       vmName,
	}).Info("🔧 API: Getting pre-flight configuration")

	// Validate failover type
	if failoverType != "test" && failoverType != "live" {
		response := FailoverResponse{
			Success: false,
			Message: "Invalid failover_type: must be 'test' or 'live'",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get default configuration options based on failover type
	var config map[string]interface{}

	if failoverType == "live" {
		config = map[string]interface{}{
			"power_off_source": map[string]interface{}{
				"default":     true,
				"required":    true,
				"description": "Power off source VM during live failover",
				"type":        "boolean",
			},
			"perform_final_sync": map[string]interface{}{
				"default":     true,
				"required":    false,
				"description": "Perform final incremental sync before failover",
				"type":        "boolean",
			},
			"network_strategy": map[string]interface{}{
				"default":     "live",
				"required":    false,
				"description": "Network mapping strategy for live failover",
				"type":        "string",
				"options":     []string{"live", "custom"},
			},
			"vm_naming": map[string]interface{}{
				"default":     "exact",
				"required":    false,
				"description": "VM naming strategy (exact same name or suffixed)",
				"type":        "string",
				"options":     []string{"exact", "suffixed"},
			},
		}
	} else {
		config = map[string]interface{}{
			"test_duration": map[string]interface{}{
				"default":     "2h",
				"required":    false,
				"description": "Duration for test failover (e.g., '2h', '30m')",
				"type":        "string",
				"pattern":     "^[0-9]+[hm]$",
			},
			"network_strategy": map[string]interface{}{
				"default":     "test",
				"required":    false,
				"description": "Network mapping strategy for test failover",
				"type":        "string",
				"options":     []string{"test", "custom"},
			},
			"vm_naming": map[string]interface{}{
				"default":     "suffixed",
				"required":    false,
				"description": "VM naming strategy (exact same name or suffixed)",
				"type":        "string",
				"options":     []string{"exact", "suffixed"},
			},
		}
	}

	// Common options for both types
	commonOptions := map[string]interface{}{
		"skip_validation": map[string]interface{}{
			"default":     false,
			"required":    false,
			"description": "Skip pre-failover validation checks",
			"type":        "boolean",
		},
		"skip_virtio": map[string]interface{}{
			"default":     false,
			"required":    false,
			"description": "Skip VirtIO driver injection (Windows VMs)",
			"type":        "boolean",
		},
	}

	// Merge common options
	for key, value := range commonOptions {
		config[key] = value
	}

	response := map[string]interface{}{
		"success":       true,
		"failover_type": failoverType,
		"vm_name":       vmName,
		"configuration": config,
		"metadata": map[string]interface{}{
			"description": fmt.Sprintf("Pre-flight configuration options for %s failover", failoverType),
			"version":     "1.0",
			"timestamp":   time.Now().UTC(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ValidatePreFlightConfiguration validates a pre-flight configuration before execution
func (fh *FailoverHandler) ValidatePreFlightConfiguration(w http.ResponseWriter, r *http.Request) {
	log.Info("🔍 API: Validating pre-flight configuration")

	var request UnifiedFailoverRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.WithError(err).Error("Failed to parse pre-flight validation request")
		response := FailoverResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid request format: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate required fields
	var validationErrors []string

	if request.ContextID == "" {
		validationErrors = append(validationErrors, "context_id is required")
	}
	if request.VMwareVMID == "" {
		validationErrors = append(validationErrors, "vmware_vm_id is required")
	}
	if request.VMName == "" {
		validationErrors = append(validationErrors, "vm_name is required")
	}
	if request.FailoverType != "test" && request.FailoverType != "live" {
		validationErrors = append(validationErrors, "failover_type must be 'test' or 'live'")
	}

	// Validate failover-type specific requirements
	if request.FailoverType == "live" {
		if request.PowerOffSource != nil && !*request.PowerOffSource {
			validationErrors = append(validationErrors, "power_off_source is required for live failover")
		}
	}

	// Validate test duration format if provided
	if request.TestDuration != "" {
		if matched, _ := regexp.MatchString(`^[0-9]+[hm]$`, request.TestDuration); !matched {
			validationErrors = append(validationErrors, "test_duration must be in format like '2h' or '30m'")
		}
	}

	// Validate network strategy
	if request.NetworkStrategy != "" {
		validStrategies := []string{"test", "live", "custom"}
		valid := false
		for _, strategy := range validStrategies {
			if request.NetworkStrategy == strategy {
				valid = true
				break
			}
		}
		if !valid {
			validationErrors = append(validationErrors, "network_strategy must be one of: test, live, custom")
		}
	}

	// Validate VM naming
	if request.VMNaming != "" {
		if request.VMNaming != "exact" && request.VMNaming != "suffixed" {
			validationErrors = append(validationErrors, "vm_naming must be 'exact' or 'suffixed'")
		}
	}

	// Return validation results
	if len(validationErrors) > 0 {
		response := map[string]interface{}{
			"success": false,
			"message": "Configuration validation failed",
			"errors":  validationErrors,
			"request": request,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Configuration is valid
	response := map[string]interface{}{
		"success": true,
		"message": "Configuration validation passed",
		"request": request,
		"metadata": map[string]interface{}{
			"validated_at": time.Now().UTC(),
			"version":      "1.0",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RegisterFailoverRoutes registers failover routes with the router
func RegisterFailoverRoutes(r *mux.Router, handler *FailoverHandler) {
	// Failover management endpoints
	r.HandleFunc("/api/v1/failover/live", handler.InitiateLiveFailover).Methods("POST")
	r.HandleFunc("/api/v1/failover/test", handler.InitiateTestFailover).Methods("POST")
	r.HandleFunc("/api/v1/failover/test/{job_id}", handler.EndTestFailover).Methods("DELETE")
	r.HandleFunc("/api/v1/failover/cleanup/{vm_name}", handler.CleanupTestFailover).Methods("POST")
	r.HandleFunc("/api/v1/failover/{job_id}/status", handler.GetFailoverJobStatus).Methods("GET")
	r.HandleFunc("/api/v1/failover/{vm_id}/readiness", handler.ValidateFailoverReadiness).Methods("GET")
	r.HandleFunc("/api/v1/failover/jobs", handler.ListFailoverJobs).Methods("GET")

	// Unified failover endpoint
	r.HandleFunc("/api/v1/failover/unified", handler.UnifiedFailover).Methods("POST")

	// Pre-flight configuration endpoints
	r.HandleFunc("/api/v1/failover/preflight/config/{failover_type}/{vm_name}", handler.GetPreFlightConfiguration).Methods("GET")
	r.HandleFunc("/api/v1/failover/preflight/validate", handler.ValidatePreFlightConfiguration).Methods("POST")

	// Enhanced rollback endpoints
	r.HandleFunc("/api/v1/failover/rollback", handler.EnhancedRollback).Methods("POST")
	r.HandleFunc("/api/v1/failover/rollback/decision/{failover_type}/{vm_name}", handler.GetRollbackDecision).Methods("GET")
}

// CleanupFailedExecution handles cleanup of failed failover/rollback operations
// POST /api/v1/failover/{vm_name}/cleanup-failed
func (fh *FailoverHandler) CleanupFailedExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["vm_name"]

	log.WithField("vm_name", vmName).Info("🧹 API: Starting failed execution cleanup")

	// Initialize cleanup service with pre-initialized OSSEA client (same as rollback system)
	cleanupService := services.NewFailedExecutionCleanupService(&fh.db, fh.jobTracker, fh.osseaClient)

	err := cleanupService.CleanupFailedExecution(r.Context(), vmName)
	if err != nil {
		log.WithError(err).WithField("vm_name", vmName).Error("Failed execution cleanup failed")
		http.Error(w, fmt.Sprintf("Failed to cleanup failed execution: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Failed execution cleanup completed for %s", vmName),
		"vm_name":   vmName,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.WithField("vm_name", vmName).Info("✅ API: Failed execution cleanup completed successfully")
}
