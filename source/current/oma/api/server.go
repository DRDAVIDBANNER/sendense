// Package api provides the OMA API server implementation
// Following project rules: minimal endpoints, modular design, clean interfaces
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/vexxhost/migratekit-oma/api/handlers"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/middleware"
)

// Server represents the main OMA API server
// Follows project rules: modular, well-structured, no monster code
type Server struct {
	config         *Config
	router         *mux.Router
	handlers       *handlers.Handlers
	rateLimiter    *middleware.RateLimiter
	inputValidator *middleware.InputValidator
}

// Config contains server configuration
type Config struct {
	Port        int                 `json:"port"`
	AuthEnabled bool                `json:"auth_enabled"`
	Database    database.Connection `json:"-"` // Don't serialize DB connection
	Debug       bool                `json:"debug"`
}

// NewServer creates a new OMA API server instance
// Following project rules: clean interfaces, modular design
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("server config is required")
	}

	server := &Server{
		config:         config,
		router:         mux.NewRouter(),
		rateLimiter:    middleware.NewRateLimiter(),
		inputValidator: middleware.NewInputValidator(),
	}

	// Initialize handlers with database connection
	var err error
	server.handlers, err = handlers.NewHandlers(config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all API endpoints following minimal endpoint design
// PROJECT RULE: Simple API with minimal endpoints to avoid sprawl
func (s *Server) setupRoutes() {
	// Enable CORS for development
	s.router.Use(s.corsMiddleware)

	// Add request logging middleware
	if s.config.Debug {
		s.router.Use(s.loggingMiddleware)
	}

	// Swagger documentation endpoints
	s.router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Health check endpoint
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Authentication endpoint
	api.HandleFunc("/auth/login", s.handlers.Auth.Login).Methods("POST")

	// VM inventory management endpoints
	api.HandleFunc("/vms", s.requireAuth(s.handlers.VM.List)).Methods("GET")
	api.HandleFunc("/vms/inventory", s.requireAuth(s.handlers.VM.ReceiveInventory)).Methods("POST")
	api.HandleFunc("/vms/{id}", s.requireAuth(s.handlers.VM.GetByID)).Methods("GET")

	// Replication job management endpoints
	api.HandleFunc("/replications", s.requireAuth(s.handlers.Replication.List)).Methods("GET")
	api.HandleFunc("/replications", s.requireAuth(s.handlers.Replication.Create)).Methods("POST")
	api.HandleFunc("/replications/changeid", s.requireAuth(s.handlers.Replication.GetPreviousChangeID)).Methods("GET")
	api.HandleFunc("/replications/{job_id}/changeid", s.requireAuth(s.handlers.Replication.StoreChangeID)).Methods("POST")
	api.HandleFunc("/replications/{id}", s.requireAuth(s.handlers.Replication.GetByID)).Methods("GET")
	api.HandleFunc("/replications/{id}", s.requireAuth(s.handlers.Replication.Update)).Methods("PUT")
	api.HandleFunc("/replications/{id}", s.requireAuth(s.handlers.Replication.Delete)).Methods("DELETE")

	// VM Context endpoints for GUI integration (VM-Centric Architecture)
	api.HandleFunc("/vm-contexts", s.requireAuth(s.handlers.VMContext.ListVMContexts)).Methods("GET")
	api.HandleFunc("/vm-contexts/{vm_name}", s.requireAuth(s.handlers.VMContext.GetVMContext)).Methods("GET")
	api.HandleFunc("/vm-contexts/{context_id}/recent-jobs", s.requireAuth(s.handlers.VMContext.GetRecentJobs)).Methods("GET")

	// OSSEA configuration - SINGLE UNIFIED ENDPOINT (following project rules)
	api.HandleFunc("/ossea/config", s.requireAuth(s.handlers.OSSEA.HandleConfig)).Methods("POST")
	// 🆕 NEW: Streamlined OSSEA configuration with auto-discovery
	api.HandleFunc("/ossea/discover-resources", s.requireAuth(s.handlers.StreamlinedOSSEA.DiscoverResources)).Methods("POST")
	api.HandleFunc("/ossea/config-streamlined", s.requireAuth(s.handlers.StreamlinedOSSEA.SaveStreamlinedConfig)).Methods("POST")

	// Linstor configuration - SINGLE UNIFIED ENDPOINT (following project rules)
	api.HandleFunc("/linstor/config", s.requireAuth(s.handlers.Linstor.HandleConfig)).Methods("POST")

	// Network mapping endpoints for VM failover system
	api.HandleFunc("/network-mappings", s.requireAuth(s.handlers.NetworkMapping.CreateNetworkMapping)).Methods("POST")
	api.HandleFunc("/network-mappings", s.requireAuth(s.handlers.NetworkMapping.ListAllNetworkMappings)).Methods("GET")
	api.HandleFunc("/network-mappings/{vm_id}", s.requireAuth(s.handlers.NetworkMapping.GetNetworkMappingsByVM)).Methods("GET")
	api.HandleFunc("/network-mappings/{vm_id}/status", s.requireAuth(s.handlers.NetworkMapping.GetNetworkMappingStatus)).Methods("GET")
	api.HandleFunc("/network-mappings/{vm_id}/{source_network_name}", s.requireAuth(s.handlers.NetworkMapping.DeleteNetworkMapping)).Methods("DELETE")

	// Network discovery and resolution endpoints
	api.HandleFunc("/networks/available", s.requireAuth(s.handlers.NetworkMapping.ListAvailableNetworks)).Methods("GET")
	api.HandleFunc("/networks/resolve", s.requireAuth(s.handlers.NetworkMapping.ResolveNetworkID)).Methods("POST")

	// Service offering discovery endpoints
	api.HandleFunc("/service-offerings/available", s.requireAuth(s.handlers.NetworkMapping.ListServiceOfferings)).Methods("GET")

	// VM Failover Management endpoints (enhanced with Linstor snapshots and VirtIO injection)
	api.HandleFunc("/failover/live", s.requireAuth(s.handlers.Failover.InitiateEnhancedLiveFailover)).Methods("POST")
	api.HandleFunc("/failover/test", s.requireAuth(s.handlers.Failover.InitiateEnhancedTestFailover)).Methods("POST")
	api.HandleFunc("/failover/test/{job_id}", s.requireAuth(s.handlers.Failover.EndTestFailover)).Methods("DELETE")
	api.HandleFunc("/failover/cleanup/{vm_name}", s.requireAuth(s.handlers.Failover.CleanupTestFailover)).Methods("POST")
	// 🆕 NEW: Failed execution cleanup for stuck operations
	api.HandleFunc("/failover/{vm_name}/cleanup-failed", s.requireAuth(s.handlers.Failover.CleanupFailedExecution)).Methods("POST")

	// UNIFIED FAILOVER SYSTEM ENDPOINTS (Phase 4 Implementation)
	// Register all unified failover routes including pre-flight configuration and enhanced rollback
	handlers.RegisterFailoverRoutes(s.router, s.handlers.Failover)

	// VMA Progress Proxy endpoints (tunneled via port 443)
	api.HandleFunc("/progress/{job_id}", s.requireAuth(s.handlers.Replication.GetVMAProgressProxy)).Methods("GET")
	api.HandleFunc("/failover/{job_id}/status", s.requireAuth(s.handlers.Failover.GetFailoverJobStatus)).Methods("GET")
	api.HandleFunc("/failover/{vm_id}/readiness", s.requireAuth(s.handlers.Failover.ValidateFailoverReadiness)).Methods("GET")
	api.HandleFunc("/failover/jobs", s.requireAuth(s.handlers.Failover.ListFailoverJobs)).Methods("GET")

	// VM Validation and Status endpoints
	api.HandleFunc("/vms/{vm_id}/failover-readiness", s.requireAuth(s.handlers.Validation.GetVMFailoverReadiness)).Methods("GET")
	api.HandleFunc("/vms/{vm_id}/sync-status", s.requireAuth(s.handlers.Validation.GetVMSyncStatus)).Methods("GET")
	api.HandleFunc("/vms/{vm_id}/network-mapping-status", s.requireAuth(s.handlers.Validation.GetVMNetworkMappingStatus)).Methods("GET")
	api.HandleFunc("/vms/{vm_id}/volume-status", s.requireAuth(s.handlers.Validation.GetVMVolumeStatus)).Methods("GET")
	api.HandleFunc("/vms/{vm_id}/active-jobs", s.requireAuth(s.handlers.Validation.GetVMActiveJobs)).Methods("GET")
	api.HandleFunc("/vms/{vm_id}/configuration-check", s.requireAuth(s.handlers.Validation.ValidateVMConfiguration)).Methods("GET")

	// Debug and troubleshooting endpoints (authentication optional for health checks)
	api.HandleFunc("/debug/health", s.handlers.Debug.GetSystemHealth).Methods("GET")
	api.HandleFunc("/debug/failover-jobs", s.requireAuth(s.handlers.Debug.GetFailoverJobsDebug)).Methods("GET")
	api.HandleFunc("/debug/endpoints", s.handlers.Debug.GetAPIEndpointsDebug).Methods("GET")
	api.HandleFunc("/debug/logs", s.requireAuth(s.handlers.Debug.GetRecentLogs)).Methods("GET")

	// Scheduler Management endpoints
	api.HandleFunc("/schedules", s.requireAuth(s.handlers.ScheduleManagement.CreateSchedule)).Methods("POST")
	api.HandleFunc("/schedules", s.requireAuth(s.handlers.ScheduleManagement.ListSchedules)).Methods("GET")
	api.HandleFunc("/schedules/{id}", s.requireAuth(s.handlers.ScheduleManagement.GetScheduleByID)).Methods("GET")
	api.HandleFunc("/schedules/{id}", s.requireAuth(s.handlers.ScheduleManagement.UpdateSchedule)).Methods("PUT")
	api.HandleFunc("/schedules/{id}", s.requireAuth(s.handlers.ScheduleManagement.DeleteSchedule)).Methods("DELETE")
	api.HandleFunc("/schedules/{id}/enable", s.requireAuth(s.handlers.ScheduleManagement.EnableSchedule)).Methods("POST")
	api.HandleFunc("/schedules/{id}/trigger", s.requireAuth(s.handlers.ScheduleManagement.TriggerSchedule)).Methods("POST")
	api.HandleFunc("/schedules/{id}/executions", s.requireAuth(s.handlers.ScheduleManagement.GetScheduleExecutions)).Methods("GET")

	// Machine Group Management endpoints
	api.HandleFunc("/machine-groups", s.requireAuth(s.handlers.MachineGroupManagement.CreateGroup)).Methods("POST")
	api.HandleFunc("/machine-groups", s.requireAuth(s.handlers.MachineGroupManagement.ListGroups)).Methods("GET")
	api.HandleFunc("/machine-groups/{id}", s.requireAuth(s.handlers.MachineGroupManagement.GetGroup)).Methods("GET")
	api.HandleFunc("/machine-groups/{id}", s.requireAuth(s.handlers.MachineGroupManagement.UpdateGroup)).Methods("PUT")
	api.HandleFunc("/machine-groups/{id}", s.requireAuth(s.handlers.MachineGroupManagement.DeleteGroup)).Methods("DELETE")

	// VM Group Assignment endpoints
	api.HandleFunc("/machine-groups/{id}/vms", s.requireAuth(s.handlers.VMGroupAssignment.AssignVMToGroup)).Methods("POST")
	api.HandleFunc("/machine-groups/{id}/vms/{vmId}", s.requireAuth(s.handlers.VMGroupAssignment.RemoveVMFromGroup)).Methods("DELETE")
	api.HandleFunc("/machine-groups/{id}/vms", s.requireAuth(s.handlers.VMGroupAssignment.ListGroupVMs)).Methods("GET")
	api.HandleFunc("/vm-contexts/{id}/group", s.requireAuth(s.handlers.VMGroupAssignment.AssignVMToGroupByContext)).Methods("PUT")

	// Enhanced Discovery endpoints
	api.HandleFunc("/discovery/discover-vms", s.requireAuth(s.handlers.EnhancedDiscovery.DiscoverVMs)).Methods("POST") // 🆕 NEW: Primary discovery endpoint with credential_id support
	api.HandleFunc("/discovery/add-vms", s.requireAuth(s.handlers.EnhancedDiscovery.AddVMs)).Methods("POST")
	api.HandleFunc("/discovery/bulk-add", s.requireAuth(s.handlers.EnhancedDiscovery.BulkAddVMs)).Methods("POST")
	api.HandleFunc("/discovery/ungrouped-vms", s.requireAuth(s.handlers.EnhancedDiscovery.GetUngroupedVMs)).Methods("GET")
	api.HandleFunc("/vm-contexts/ungrouped", s.requireAuth(s.handlers.EnhancedDiscovery.GetUngroupedVMContexts)).Methods("GET")

	// 🆕 NEW: VMware Credentials Management endpoints (complete CRUD)
	api.HandleFunc("/vmware-credentials", s.requireAuth(s.handlers.VMwareCredentials.ListCredentials)).Methods("GET")
	api.HandleFunc("/vmware-credentials", s.requireAuth(s.handlers.VMwareCredentials.CreateCredentials)).Methods("POST")
	api.HandleFunc("/vmware-credentials/{id}", s.requireAuth(s.handlers.VMwareCredentials.GetCredentials)).Methods("GET")
	api.HandleFunc("/vmware-credentials/{id}", s.requireAuth(s.handlers.VMwareCredentials.UpdateCredentials)).Methods("PUT")
	api.HandleFunc("/vmware-credentials/{id}", s.requireAuth(s.handlers.VMwareCredentials.DeleteCredentials)).Methods("DELETE")
	api.HandleFunc("/vmware-credentials/{id}/set-default", s.requireAuth(s.handlers.VMwareCredentials.SetDefaultCredentials)).Methods("PUT")
	api.HandleFunc("/vmware-credentials/{id}/test", s.requireAuth(s.handlers.VMwareCredentials.TestCredentials)).Methods("POST")
	api.HandleFunc("/vmware-credentials/default", s.requireAuth(s.handlers.VMwareCredentials.GetDefaultCredentials)).Methods("GET")

	// 🆕 NEW: CloudStack Settings and Validation endpoints
	api.HandleFunc("/settings/cloudstack/test-connection", s.requireAuth(s.handlers.CloudStackSettings.TestConnection)).Methods("POST")
	api.HandleFunc("/settings/cloudstack/detect-oma-vm", s.requireAuth(s.handlers.CloudStackSettings.DetectOMAVM)).Methods("POST")
	api.HandleFunc("/settings/cloudstack/networks", s.requireAuth(s.handlers.CloudStackSettings.ListNetworks)).Methods("GET")
	api.HandleFunc("/settings/cloudstack/validate", s.requireAuth(s.handlers.CloudStackSettings.ValidateSettings)).Methods("POST")
	api.HandleFunc("/settings/cloudstack/discover-all", s.requireAuth(s.handlers.CloudStackSettings.DiscoverAllResources)).Methods("POST")

	// 🆕 NEW: VMA Enrollment System endpoints (real implementation with security hardening)
	// Admin endpoints (authenticated, basic rate limiting)
	api.HandleFunc("/admin/vma/pairing-code", s.requireAuth(s.handlers.VMAReal.GeneratePairingCode)).Methods("POST", "OPTIONS")
	api.HandleFunc("/admin/vma/pending", s.requireAuth(s.handlers.VMAReal.ListPendingEnrollments)).Methods("GET", "OPTIONS")
	api.HandleFunc("/admin/vma/approve/{id}", s.requireAuth(s.handlers.VMAReal.ApproveEnrollment)).Methods("POST", "OPTIONS")
	api.HandleFunc("/admin/vma/active", s.requireAuth(s.handlers.VMAReal.ListActiveVMAs)).Methods("GET", "OPTIONS")
	api.HandleFunc("/admin/vma/reject/{id}", s.requireAuth(s.handlers.VMAReal.RejectEnrollment)).Methods("POST", "OPTIONS")
	api.HandleFunc("/admin/vma/revoke/{id}", s.requireAuth(s.handlers.VMAReal.RevokeVMAAccess)).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/admin/vma/audit", s.requireAuth(s.handlers.VMAReal.GetAuditLog)).Methods("GET", "OPTIONS")

	// Public enrollment endpoints (internet-exposed) - security middleware will be added in port 443 setup
	api.HandleFunc("/vma/enroll", s.handlers.VMAReal.EnrollVMA).Methods("POST", "OPTIONS")
	api.HandleFunc("/vma/enroll/verify", s.handlers.VMAReal.VerifyChallenge).Methods("POST", "OPTIONS")
	api.HandleFunc("/vma/enroll/result", s.handlers.VMAReal.GetEnrollmentResult).Methods("GET", "OPTIONS")

	log.WithField("endpoints", 71).Info("OMA API routes configured - includes basic VMA enrollment")
}

// Middleware functions

// corsMiddleware adds CORS headers for development
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs all requests with enhanced details for troubleshooting
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Capture response details
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Log request details
		log.WithFields(log.Fields{
			"method":       r.Method,
			"path":         r.URL.Path,
			"query":        r.URL.RawQuery,
			"remote":       r.RemoteAddr,
			"user_agent":   r.UserAgent(),
			"content_type": r.Header.Get("Content-Type"),
			"request_id":   generateRequestID(),
		}).Info("🔄 API request started")

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		logLevel := log.InfoLevel
		statusPrefix := "✅"

		// Adjust log level and prefix based on response status
		if wrapped.statusCode >= 400 && wrapped.statusCode < 500 {
			logLevel = log.WarnLevel
			statusPrefix = "⚠️"
		} else if wrapped.statusCode >= 500 {
			logLevel = log.ErrorLevel
			statusPrefix = "❌"
		}

		log.WithFields(log.Fields{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status_code": wrapped.statusCode,
			"duration":    duration,
			"duration_ms": duration.Milliseconds(),
			"remote":      r.RemoteAddr,
		}).Logf(logLevel, "%s API request completed", statusPrefix)

		// Log slow requests as warnings
		if duration > 5*time.Second {
			log.WithFields(log.Fields{
				"method":   r.Method,
				"path":     r.URL.Path,
				"duration": duration,
			}).Warn("🐌 Slow API request detected")
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// requireAuth middleware for protected endpoints
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.config.AuthEnabled {
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.writeErrorResponse(w, http.StatusUnauthorized, "Missing authorization header", "")
			return
		}

		// Parse Bearer token
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			s.writeErrorResponse(w, http.StatusUnauthorized, "Invalid authorization format", "")
			return
		}

		token := authHeader[7:]
		if !s.handlers.Auth.ValidateToken(token) {
			s.writeErrorResponse(w, http.StatusUnauthorized, "Invalid or expired token", "")
			return
		}

		next(w, r)
	}
}

// Route handlers

// handleHealth provides health check endpoint
// @Summary OMA Health Check
// @Description Check if the OMA API server is running and healthy
// @Tags health
// @Produce json
// @Success 200 {object} object
// @Router /health [get]
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service":   "OMA API",
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"database":  s.getDatabaseStatus(),
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// Helper functions following project standards: small, focused, well-documented

// getDatabaseStatus returns database connection status
func (s *Server) getDatabaseStatus() string {
	if s.config.Database == nil {
		return "memory"
	}
	return "connected"
}

// writeJSONResponse writes a standardized JSON response
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeErrorResponse writes a standardized error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	response := map[string]interface{}{
		"error":     message,
		"details":   details,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	s.writeJSONResponse(w, statusCode, response)
}

// Start starts the OMA API server with graceful shutdown support
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.config.Port)

	server := &http.Server{
		Addr:    addr,
		Handler: s.router,
		// Security timeouts
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.WithFields(log.Fields{
			"port":      s.config.Port,
			"endpoints": 9,
		}).Info("Starting OMA API server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Server failed to start")
		}
	}()

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Info("Shutting down OMA API server gracefully...")
	return server.Shutdown(shutdownCtx)
}

// GetHandlers returns the handlers instance for accessing internal services
// Used by job recovery and other system components that need access to VMA services
func (s *Server) GetHandlers() *handlers.Handlers {
	return s.handlers
}
