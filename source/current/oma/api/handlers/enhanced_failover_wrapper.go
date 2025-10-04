// Package handlers provides a wrapper to integrate enhanced failover into existing endpoints
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/common"
	"github.com/vexxhost/migratekit-oma/common/logging"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/failover"
	"github.com/vexxhost/migratekit-oma/joblog"
	"github.com/vexxhost/migratekit-oma/ossea"
	"github.com/vexxhost/migratekit-oma/services"
)

// NewEnhancedFailoverHandler creates a handler that acts like FailoverHandler but uses enhanced functionality
// FIXED: Properly initializes all dependencies instead of passing nil
func NewEnhancedFailoverHandler(db database.Connection) *FailoverHandler {
	log.Info("🚀 Initializing enhanced failover handler (resilient mode)")

	// Skip centralized logging for initialization to avoid database dependencies
	// Use basic logging until system is fully initialized

	// Create enhanced engine
	var osseaClient *ossea.Client
	var networkClient *ossea.NetworkClient

	// Try to get active OSSEA configuration (resilient mode)
	var configs []database.OSSEAConfig
	err := db.GetGormDB().Where("is_active = true").Find(&configs).Error
	if err == nil && len(configs) > 0 {
		config := configs[0]
		osseaClient = ossea.NewClient(
			config.APIURL,
			config.APIKey,
			config.SecretKey,
			config.Domain,
			config.Zone,
		)
		networkClient = ossea.NewNetworkClient(osseaClient)
		log.WithFields(log.Fields{
			"config_name": config.Name,
			"api_url":     config.APIURL,
			"zone":        config.Zone,
		}).Info("✅ OSSEA client initialized successfully")
	} else {
		log.WithFields(log.Fields{
			"configs_found": len(configs),
			"error":         err,
		}).Warn("⚠️ No active OSSEA configuration found - will initialize with nil clients")
		// Continue with nil clients - this is OK for initialization
	}

	// Initialize all required services (FIX: No more nil dependencies)
	log.Info("🔧 Initializing required services")

	// Initialize repositories
	failoverJobRepo := database.NewFailoverJobRepository(db)
	networkMappingRepo := database.NewNetworkMappingRepository(db)

	// Initialize VM info service (database-based)
	vmInfoService := services.NewSimpleDatabaseVMInfoService(db)
	log.Info("✅ VM info service initialized (database-based)")

	// Initialize network mapping service
	networkMappingService := services.NewNetworkMappingService(networkMappingRepo, networkClient, vmInfoService)
	log.Info("✅ Network mapping service initialized")

	// Initialize validator
	validator := failover.NewPreFailoverValidator(
		db,
		vmInfoService,
		networkMappingService,
	)
	log.Info("✅ Pre-failover validator initialized")

	// Create JobLog tracker for enhanced failover
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	// Get sql.DB from GORM for joblog
	sqlDB, err := db.GetGormDB().DB()
	if err != nil {
		log.WithError(err).Fatal("Failed to get sql.DB from GORM for joblog")
	}
	dbHandler := joblog.NewDBHandler(sqlDB, joblog.DefaultDBHandlerConfig())
	jobTracker := joblog.New(sqlDB, stdoutHandler, dbHandler)

	// Create enhanced engines with JobLog integration
	enhancedLiveEngine := failover.NewEnhancedLiveFailoverEngine(
		db,                    // Database connection
		osseaClient,           // OSSEA client
		networkClient,         // Network client
		vmInfoService,         // VM info service
		networkMappingService, // Network mapping service
		validator,             // Pre-failover validator
		jobTracker,            // JobLog tracker for unified logging
	)
	log.Info("✅ Enhanced live failover engine initialized")

	// Create enhanced test failover engine with JobLog-only architecture
	enhancedTestEngine := failover.NewEnhancedTestFailoverEngine(
		&db,             // Database connection pointer
		osseaClient,     // OSSEA client
		failoverJobRepo, // Failover job repository
		validator,       // Pre-failover validator
		jobTracker,      // JobLog tracker for structured logging
	)
	log.Info("✅ Enhanced test failover engine initialized")
	log.Info("🐛 DEBUG: After enhanced test engine creation, before cleanup service")

	// Create enhanced cleanup service with JobLog integration
	log.Info("🐛 DEBUG: About to create enhanced cleanup service")
	// Initialize VMA client for power management (credentials passed per-operation from VM context)
	var vmaClient failover.VMAClient = failover.NewVMAClientForFailover()
	if vmaClient == nil {
		log.Error("Failed to initialize VMA client for power management - using null client")
		// Continue with null client - power management will be disabled
		vmaClient = failover.NewNullVMAClient()
	}
	enhancedCleanupService := failover.NewEnhancedCleanupService(db, jobTracker, vmaClient)
	log.Info("🐛 DEBUG: Enhanced cleanup service created successfully")
	if enhancedCleanupService == nil {
		log.Error("🐛 DEBUG ERROR: enhancedCleanupService is NIL after creation!")
	} else {
		log.Info("🐛 DEBUG: enhancedCleanupService is NOT nil")
	}
	log.Info("✅ Enhanced cleanup service initialized with JobLog")

	// Create a custom handler with enhanced functionality
	handler := &FailoverHandler{
		db:                     db,
		failoverJobRepo:        failoverJobRepo,
		jobTracker:             jobTracker, // ✅ CRITICAL FIX: Assign JobLog tracker for status endpoint correlation
		enhancedLiveEngine:     enhancedLiveEngine,
		enhancedTestEngine:     enhancedTestEngine,
		enhancedCleanupService: enhancedCleanupService, // ✅ JobLog-enabled cleanup service
		// Also store references for potential future use
		osseaClient:           osseaClient,
		networkClient:         networkClient,
		vmInfoService:         vmInfoService,
		networkMappingService: networkMappingService,
		validator:             validator,
	}

	log.WithFields(log.Fields{
		"handler_type":       "enhanced-failover",
		"live_engine_ready":  enhancedLiveEngine != nil,
		"test_engine_ready":  enhancedTestEngine != nil,
		"ossea_client_ready": osseaClient != nil,
	}).Info("✅ Enhanced failover handler initialization completed")

	// UNIFIED FAILOVER SYSTEM INITIALIZATION (Phase 4 Implementation)
	// Initialize unified failover engine and configuration resolver
	log.Info("🚀 Initializing unified failover system components")

	// Initialize volume client for unified engine
	volumeClient := common.NewVolumeClient("http://localhost:8090")

	// VMA client already initialized above for cleanup service

	// Initialize unified failover engine
	handler.unifiedEngine = failover.NewUnifiedFailoverEngine(
		db,
		jobTracker,
		osseaClient,
		networkClient,
		vmInfoService,
		networkMappingService,
		volumeClient,
		vmaClient,
	)

	if handler.unifiedEngine == nil {
		log.Error("❌ CRITICAL: Failed to initialize unified failover engine")
	} else {
		log.Info("✅ Unified failover engine initialized successfully")
	}

	// Initialize configuration resolver
	vmContextRepo := database.NewVMReplicationContextRepository(db)
	handler.configResolver = failover.NewFailoverConfigResolver(
		networkMappingService,
		vmContextRepo,
		networkMappingRepo,
	)

	if handler.configResolver == nil {
		log.Error("❌ CRITICAL: Failed to initialize failover config resolver")
	} else {
		log.Info("✅ Failover config resolver initialized successfully")
	}

	return handler
}

// InitiateEnhancedLiveFailover handles POST /api/v1/failover/live with enhanced functionality
func (fh *FailoverHandler) InitiateEnhancedLiveFailover(w http.ResponseWriter, r *http.Request) {
	// Initialize centralized logging for API request
	logger := logging.NewOperationLogger("api-live-failover")
	opCtx := logger.StartOperation("api-enhanced-live-failover", "api-request")

	var request LiveFailoverRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		if opCtx != nil {
			opCtx.LogError("request-parsing", "Failed to parse request payload", err, log.Fields{
				"content_type":   r.Header.Get("Content-Type"),
				"content_length": r.ContentLength,
			})
			opCtx.EndOperation("failed", log.Fields{"failure_reason": "invalid_request_payload"})
		}
		fh.writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// Generate failover job ID
	failoverJobID := fmt.Sprintf("enhanced-live-failover-%s-%d", request.VMID, time.Now().Unix())

	opCtx.LogStep("request-received", "Enhanced live failover request received", log.Fields{
		"vm_id":            request.VMID,
		"vm_name":          request.VMName,
		"failover_job_id":  failoverJobID,
		"with_snapshots":   true,
		"with_virtio":      true,
		"skip_validation":  request.SkipValidation,
		"network_mappings": len(request.NetworkMappings),
	})

	// Convert API request to engine request (enable snapshots and VirtIO injection by default)
	engineRequest := &failover.EnhancedFailoverRequest{
		VMID:                request.VMID,
		VMName:              request.VMName,
		FailoverJobID:       failoverJobID,
		SkipValidation:      request.SkipValidation,
		SkipSnapshot:        false, // Always enable snapshots for protection
		SkipVirtIOInjection: false, // Always enable VirtIO injection
		NetworkMappings:     request.NetworkMappings,
		CustomConfig:        request.CustomConfig,
		NotificationConfig:  request.NotificationConfig,
		LinstorConfigID:     nil, // Use default Linstor config
	}

	// Execute enhanced failover asynchronously with proper logging
	go func() {
		// Create child context for async execution
		asyncCtx := opCtx.CreateChildContext("async-execution")
		asyncCtx.LogStep("execution-start", "Starting async enhanced live failover execution", log.Fields{
			"failover_job_id": failoverJobID,
			"execution_mode":  "asynchronous",
		})

		result, err := fh.enhancedLiveEngine.ExecuteEnhancedFailover(
			logging.WithContext(context.Background(), asyncCtx.GetCorrelationID()),
			engineRequest,
		)
		if err != nil {
			asyncCtx.LogError("execution-failed", "Enhanced failover execution failed", err, log.Fields{
				"failover_job_id": failoverJobID,
				"vm_id":           request.VMID,
			})
			asyncCtx.EndOperation("failed", log.Fields{
				"failure_reason": "execution_error",
				"error_message":  err.Error(),
			})
		} else {
			asyncCtx.LogSuccess("execution-completed", "Enhanced failover completed successfully", log.Fields{
				"parent_job_id":     result.ParentJobID,
				"destination_vm_id": result.DestinationVMID,
				"snapshot_name":     result.LinstorSnapshotName,
				"virtio_status":     result.VirtIOInjectionStatus,
			})
			asyncCtx.EndOperation("completed", log.Fields{
				"total_duration":   result.Duration,
				"child_jobs_count": len(result.ChildJobs),
			})
		}
	}()

	// Return immediate response (same format as original for GUI compatibility)
	response := FailoverResponse{
		Success: true,
		Message: "Live failover initiated successfully with snapshot protection and VirtIO injection",
		JobID:   failoverJobID,

		Data: map[string]interface{}{
			"job_type":            "live",
			"status":              "executing",
			"vm_id":               request.VMID,
			"vm_name":             request.VMName,
			"snapshot_protection": true,
			"virtio_injection":    true,
			"correlation_id":      opCtx.GetCorrelationID(),
		},
	}

	opCtx.LogSuccess("api-response", "API response sent successfully", log.Fields{
		"response_status":    "accepted",
		"failover_job_id":    failoverJobID,
		"estimated_duration": "8-15 minutes",
	})
	opCtx.EndOperation("completed", log.Fields{
		"operation_type":          "api_request_handling",
		"async_execution_started": true,
	})

	fh.writeJSONResponse(w, http.StatusOK, response)
}

// InitiateEnhancedTestFailover handles POST /api/v1/failover/test with enhanced functionality
func (fh *FailoverHandler) InitiateEnhancedTestFailover(w http.ResponseWriter, r *http.Request) {
	// Initialize centralized logging for API request
	logger := logging.NewOperationLogger("api-test-failover")
	opCtx := logger.StartOperation("api-enhanced-test-failover", "api-request")

	var request TestFailoverRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		opCtx.LogError("request-parsing", "Failed to parse test failover request payload", err, log.Fields{
			"content_type":   r.Header.Get("Content-Type"),
			"content_length": r.ContentLength,
		})
		opCtx.EndOperation("failed", log.Fields{"failure_reason": "invalid_request_payload"})
		fh.writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// Generate failover job ID using UUID for JobLog correlation
	failoverJobID := uuid.New().String()

	opCtx.LogStep("request-received", "Enhanced test failover request received", log.Fields{
		"context_id":       request.ContextID,
		"vm_id":            request.VMID,
		"vm_name":          request.VMName,
		"failover_job_id":  failoverJobID,
		"test_duration":    request.TestDuration,
		"auto_cleanup":     request.AutoCleanup,
		"with_snapshots":   true,
		"with_virtio":      true,
		"skip_validation":  request.SkipValidation,
		"network_mappings": len(request.NetworkMappings),
	})

	// Convert API request to engine request with VM-centric context ID
	engineRequest := &failover.EnhancedTestFailoverRequest{
		ContextID:     request.ContextID,
		VMID:          request.VMID,
		VMName:        request.VMName,
		FailoverJobID: failoverJobID,
		Timestamp:     time.Now(),
	}

	// Execute enhanced test failover asynchronously with proper logging
	go func() {
		// Create child context for async execution
		asyncCtx := opCtx.CreateChildContext("async-test-execution")
		asyncCtx.LogStep("execution-start", "Starting async enhanced test failover execution", log.Fields{
			"failover_job_id": failoverJobID,
			"execution_mode":  "asynchronous",
			"test_duration":   request.TestDuration,
		})

		actualJobID, err := fh.enhancedTestEngine.ExecuteEnhancedTestFailover(
			logging.WithContext(context.Background(), asyncCtx.GetCorrelationID()),
			engineRequest,
		)
		if err != nil {
			asyncCtx.LogError("execution-failed", "Enhanced test failover execution failed", err, log.Fields{
				"failover_job_id": failoverJobID,
				"vm_id":           request.VMID,
				"test_duration":   request.TestDuration,
			})
			asyncCtx.EndOperation("failed", log.Fields{
				"failure_reason": "execution_error",
				"error_message":  err.Error(),
			})
		} else {
			asyncCtx.LogSuccess("execution-completed", "Enhanced test failover completed successfully", log.Fields{
				"failover_job_id": failoverJobID,
				"actual_job_id":   actualJobID,
				"vm_id":           request.VMID,
			})
			asyncCtx.EndOperation("completed", log.Fields{
				"execution_mode": "asynchronous",
				"result":         "success",
			})
		}
	}()

	// Return immediate response (same format as original for GUI compatibility)
	response := FailoverResponse{
		Success: true,
		Message: "Test failover initiated successfully with snapshot protection and VirtIO injection",
		JobID:   failoverJobID,

		Data: map[string]interface{}{
			"job_type":            "test",
			"status":              "executing",
			"vm_id":               request.VMID,
			"vm_name":             request.VMName,
			"test_duration":       request.TestDuration,
			"auto_cleanup":        request.AutoCleanup,
			"snapshot_protection": true,
			"virtio_injection":    true,
			"correlation_id":      opCtx.GetCorrelationID(),
		},
	}

	opCtx.LogSuccess("api-response", "Test failover API response sent successfully", log.Fields{
		"response_status": "accepted",
		"failover_job_id": failoverJobID,
		"test_duration":   request.TestDuration,
		"auto_cleanup":    request.AutoCleanup,
	})
	opCtx.EndOperation("completed", log.Fields{
		"operation_type":          "api_request_handling",
		"async_execution_started": true,
		"test_failover_mode":      true,
	})

	fh.writeJSONResponse(w, http.StatusOK, response)
}

// Helper functions

// writeJSONResponse writes a standardized JSON response
func (fh *FailoverHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeErrorResponse writes a standardized error response
func (fh *FailoverHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	response := map[string]interface{}{
		"success":   false,
		"message":   message,
		"error":     details,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	fh.writeJSONResponse(w, statusCode, response)
}
