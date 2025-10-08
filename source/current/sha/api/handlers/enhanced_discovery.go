// Package handlers provides enhanced discovery API endpoints for scheduler system
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/services"
)

// EnhancedDiscoveryHandler handles enhanced discovery API endpoints
type EnhancedDiscoveryHandler struct {
	discoveryService *services.EnhancedDiscoveryService
	vmContextRepo    *database.VMReplicationContextRepository
	schedulerRepo    *database.SchedulerRepository
	tracker          *joblog.Tracker
	db               database.Connection // 🆕 NEW: Database connection for credential lookup
}

// NewEnhancedDiscoveryHandler creates a new enhanced discovery handler
func NewEnhancedDiscoveryHandler(discoveryService *services.EnhancedDiscoveryService,
	vmContextRepo *database.VMReplicationContextRepository,
	schedulerRepo *database.SchedulerRepository,
	tracker *joblog.Tracker,
	db database.Connection) *EnhancedDiscoveryHandler { // 🆕 NEW: Added db parameter
	return &EnhancedDiscoveryHandler{
		discoveryService: discoveryService,
		vmContextRepo:    vmContextRepo,
		schedulerRepo:    schedulerRepo,
		tracker:          tracker,
		db:               db, // 🆕 NEW: Store database connection
	}
}

// DiscoverVMsRequest represents a request to discover VMs without creating jobs
type DiscoverVMsRequest struct {
	// NEW: Optional credential ID for saved credentials (preferred method)
	CredentialID  *int     `json:"credential_id,omitempty"` // Use saved credentials from database
	
	// Existing fields (now optional when credential_id provided)
	VCenter       string   `json:"vcenter,omitempty"`       // vCenter hostname (required if no credential_id)
	Username      string   `json:"username,omitempty"`      // vCenter username (required if no credential_id)
	Password      string   `json:"password,omitempty"`      // vCenter password (required if no credential_id)
	Datacenter    string   `json:"datacenter,omitempty"`    // Datacenter name (required if no credential_id)
	
	// Unchanged
	Filter        string   `json:"filter,omitempty"`       // Optional VM name filter
	SelectedVMs   []string `json:"selected_vms,omitempty"` // Specific VMs to add (empty = all discovered)
	CreateContext bool     `json:"create_context"`         // Whether to create VM contexts immediately
}

// DiscoverVMsResponse represents the response from VM discovery
type DiscoverVMsResponse struct {
	DiscoveredVMs  []DiscoveredVMInfo      `json:"discovered_vms"`
	AdditionResult *services.BulkAddResult `json:"addition_result,omitempty"`
	DiscoveryCount int                     `json:"discovery_count"`
	ProcessingTime time.Duration           `json:"processing_time"`
	Status         string                  `json:"status"`
	Message        string                  `json:"message"`
}

// DiscoveredVMInfo represents a discovered VM
type DiscoveredVMInfo struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	Path       string                     `json:"path"`
	PowerState string                     `json:"power_state"`
	GuestOS    string                     `json:"guest_os"`
	MemoryMB   int                        `json:"memory_mb"`
	NumCPU     int                        `json:"num_cpu"`
	VMXVersion string                     `json:"vmx_version,omitempty"`
	Disks      []services.SNADiskInfo     `json:"disks"`
	Networks   []services.SNANetworkInfo  `json:"networks"`
	Existing   bool                       `json:"existing"`             // Whether VM context already exists
	ContextID  string                     `json:"context_id,omitempty"` // Existing context ID if applicable
}

// BulkAddVMsRequest represents a request to bulk add VMs to SHA
type BulkAddVMsRequest struct {
	VCenter     string   `json:"vcenter" binding:"required"`
	Username    string   `json:"username" binding:"required"`
	Password    string   `json:"password" binding:"required"`
	Datacenter  string   `json:"datacenter" binding:"required"`
	Filter      string   `json:"filter,omitempty"`
	SelectedVMs []string `json:"selected_vms" binding:"required"` // Must specify which VMs to add
}

// UngroupedVMsResponse represents VMs that are discovered but not in any group
type UngroupedVMsResponse struct {
	VMs         []UngroupedVMInfo `json:"vms"`
	Count       int               `json:"count"`
	RetrievedAt time.Time         `json:"retrieved_at"`
}

// UngroupedVMInfo represents an ungrouped VM
type UngroupedVMInfo struct {
	ContextID        string     `json:"context_id"`
	VMName           string     `json:"vm_name"`
	VMPath           string     `json:"vm_path"`
	VCenterHost      string     `json:"vcenter_host"`
	Datacenter       string     `json:"datacenter"`
	CurrentStatus    string     `json:"current_status"`
	AutoAdded        bool       `json:"auto_added"`
	SchedulerEnabled bool       `json:"scheduler_enabled"`
	CPUCount         *int       `json:"cpu_count"`
	MemoryMB         *int       `json:"memory_mb"`
	OSType           *string    `json:"os_type"`
	PowerState       *string    `json:"power_state"`
	CreatedAt        time.Time  `json:"created_at"`
	LastJobAt        *time.Time `json:"last_job_at"`
}

// AddVMsRequest represents a request to add specific VMs to SHA without jobs
type AddVMsRequest struct {
	// NEW: Optional credential ID for saved credentials (preferred method)
	CredentialID *int     `json:"credential_id,omitempty"`
	
	// Existing fields (now optional when credential_id provided)
	VCenter    string   `json:"vcenter,omitempty"`
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	Datacenter string   `json:"datacenter,omitempty"`
	VMNames    []string `json:"vm_names" binding:"required,min=1"`
	AddedBy    string   `json:"added_by,omitempty"`
}

// AddVMsResponse represents the response from adding VMs to SHA
type AddVMsResponse struct {
	Success      bool              `json:"success"`
	Message      string            `json:"message"`
	VMsAdded     int               `json:"vms_added"`
	VMsFailed    int               `json:"vms_failed"`
	TotalVMs     int               `json:"total_vms"`
	AddedAt      time.Time         `json:"added_at"`
	ProcessedVMs []ProcessedVMInfo `json:"processed_vms,omitempty"`
}

// ProcessedVMInfo represents information about a processed VM
type ProcessedVMInfo struct {
	VMName    string  `json:"vm_name"`
	Success   bool    `json:"success"`
	ContextID *string `json:"context_id,omitempty"`
	Error     *string `json:"error,omitempty"`
}

// DiscoverVMs discovers VMs from SNA with optional context creation
// POST /api/v1/discovery/discover-vms
func (h *EnhancedDiscoveryHandler) DiscoverVMs(w http.ResponseWriter, r *http.Request) {
	var request DiscoverVMsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	var vcenter, username, password, datacenter string
	
	// 🆕 NEW: Check if using credential_id or manual entry
	if request.CredentialID != nil && *request.CredentialID > 0 {
		// Load credentials from database
		encryptionService, err := services.NewCredentialEncryptionService()
		if err != nil {
			log.WithError(err).Error("Failed to initialize credential encryption service")
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to initialize encryption service: "+err.Error())
			return
		}
		credentialService := services.NewVMwareCredentialService(&h.db, encryptionService)
		
		creds, err := credentialService.GetCredentials(r.Context(), *request.CredentialID)
		if err != nil {
			log.WithError(err).WithField("credential_id", *request.CredentialID).Error("Failed to load VMware credentials")
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to load credentials: "+err.Error())
			return
		}
		
		vcenter = creds.VCenterHost
		username = creds.Username
		password = creds.Password
		datacenter = creds.Datacenter
		
		log.WithFields(log.Fields{
			"credential_id":   *request.CredentialID,
			"credential_name": creds.Name,
			"vcenter_host":    creds.VCenterHost,
			"datacenter":      creds.Datacenter,
		}).Info("✅ Using saved VMware credentials for discovery")
	} else {
		// Validate manual entry fields
		if request.VCenter == "" || request.Username == "" || request.Password == "" || request.Datacenter == "" {
			h.writeErrorResponse(w, http.StatusBadRequest, "Either credential_id OR (vcenter, username, password, datacenter) must be provided")
			return
		}
		
		vcenter = request.VCenter
		username = request.Username
		password = request.Password
		datacenter = request.Datacenter
		
		log.WithField("vcenter_host", vcenter).Info("Using manual VMware credentials for discovery")
	}

	log.WithFields(log.Fields{
		"vcenter":        vcenter,
		"datacenter":     datacenter,
		"filter":         request.Filter,
		"create_context": request.CreateContext,
		"selected_count": len(request.SelectedVMs),
	}).Info("Starting enhanced VM discovery")

	ctx := r.Context()
	start := time.Now()

	// Discover VMs from SNA (using resolved credentials)
	discoveryReq := services.DiscoveryRequest{
		VCenter:    vcenter,
		Username:   username,
		Password:   password,
		Datacenter: datacenter,
		Filter:     request.Filter,
	}

	snaResponse, err := h.discoveryService.DiscoverVMsFromVMA(ctx, discoveryReq)
	if err != nil {
		log.WithError(err).Error("Failed to discover VMs from SNA")
		h.writeErrorResponse(w, http.StatusInternalServerError, "VM discovery failed: "+err.Error())
		return
	}

	// Convert to response format and check for existing contexts
	discoveredVMs := make([]DiscoveredVMInfo, 0, len(snaResponse.VMs))
	for _, vm := range snaResponse.VMs {
		existing, _ := h.vmContextRepo.GetVMContextByName(vm.Name)

		discoveredVM := DiscoveredVMInfo{
			ID:         vm.ID,
			Name:       vm.Name,
			Path:       vm.Path,
			PowerState: vm.PowerState,
			GuestOS:    vm.GuestOS,
			MemoryMB:   vm.MemoryMB,
			NumCPU:     vm.NumCPU,
			VMXVersion: vm.VMXVersion,
			Disks:      vm.Disks,      // Include disk information
			Networks:   vm.Networks,   // Include network information
			Existing:   existing != nil,
		}

		if existing != nil {
			discoveredVM.ContextID = existing.ContextID
		}

		discoveredVMs = append(discoveredVMs, discoveredVM)
	}

	response := DiscoverVMsResponse{
		DiscoveredVMs:  discoveredVMs,
		DiscoveryCount: len(discoveredVMs),
		ProcessingTime: time.Since(start),
		Status:         "success",
		Message:        "VM discovery completed successfully",
	}

	// If create_context is true, add VMs to SHA
	if request.CreateContext {
		addResult, err := h.discoveryService.AddVMsToOMAWithoutJobs(ctx, discoveryReq, request.SelectedVMs)
		if err != nil {
			log.WithError(err).Error("Failed to add VMs to SHA")
			response.Status = "partial_success"
			response.Message = "Discovery succeeded but VM addition failed: " + err.Error()
		} else {
			response.AdditionResult = addResult
			response.Message = "VM discovery and addition completed successfully"
		}
	}

	log.WithFields(log.Fields{
		"discovered_vms":   len(discoveredVMs),
		"processing_time":  response.ProcessingTime,
		"context_creation": request.CreateContext,
		"status":           response.Status,
	}).Info("Enhanced VM discovery completed")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// BulkAddVMs adds multiple VMs to SHA without creating replication jobs
// POST /api/v1/discovery/bulk-add
func (h *EnhancedDiscoveryHandler) BulkAddVMs(w http.ResponseWriter, r *http.Request) {
	var request BulkAddVMsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate request
	if request.VCenter == "" || request.Username == "" || request.Password == "" || request.Datacenter == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "vCenter, username, password, and datacenter are required")
		return
	}

	if len(request.SelectedVMs) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "At least one VM must be selected for addition")
		return
	}

	log.WithFields(log.Fields{
		"vcenter":      request.VCenter,
		"datacenter":   request.Datacenter,
		"filter":       request.Filter,
		"selected_vms": len(request.SelectedVMs),
	}).Info("Starting bulk VM addition to SHA")

	ctx := r.Context()

	// Add VMs to SHA
	discoveryReq := services.DiscoveryRequest{
		VCenter:    request.VCenter,
		Username:   request.Username,
		Password:   request.Password,
		Datacenter: request.Datacenter,
		Filter:     request.Filter,
	}

	result, err := h.discoveryService.AddVMsToOMAWithoutJobs(ctx, discoveryReq, request.SelectedVMs)
	if err != nil {
		log.WithError(err).Error("Failed to bulk add VMs to SHA")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Bulk VM addition failed: "+err.Error())
		return
	}

	log.WithFields(log.Fields{
		"total_requested":     result.TotalRequested,
		"successfully_added":  result.SuccessfullyAdded,
		"skipped":             result.Skipped,
		"failed":              result.Failed,
		"discovery_duration":  result.DiscoveryDuration,
		"processing_duration": result.ProcessingDuration,
	}).Info("Bulk VM addition completed")

	h.writeJSONResponse(w, http.StatusOK, result)
}

// AddVMs adds specific VMs to SHA without creating replication jobs (simplified interface)
// POST /api/v1/discovery/add-vms
func (h *EnhancedDiscoveryHandler) AddVMs(w http.ResponseWriter, r *http.Request) {
	var request AddVMsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate request
	if len(request.VMNames) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "At least one VM name is required")
		return
	}

	var vcenter, username, password, datacenter string
	
	// Check if using credential_id or manual entry
	if request.CredentialID != nil && *request.CredentialID > 0 {
		// Load credentials from database
		encryptionService, err := services.NewCredentialEncryptionService()
		if err != nil {
			log.WithError(err).Error("Failed to initialize credential encryption service")
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to initialize encryption service: "+err.Error())
			return
		}
		credentialService := services.NewVMwareCredentialService(&h.db, encryptionService)
		
		creds, err := credentialService.GetCredentials(r.Context(), *request.CredentialID)
		if err != nil {
			log.WithError(err).WithField("credential_id", *request.CredentialID).Error("Failed to load VMware credentials")
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to load credentials: "+err.Error())
			return
		}
		
		vcenter = creds.VCenterHost
		username = creds.Username
		password = creds.Password
		datacenter = creds.Datacenter
		
		log.WithFields(log.Fields{
			"credential_id":   *request.CredentialID,
			"credential_name": creds.Name,
			"vcenter_host":    creds.VCenterHost,
			"datacenter":      creds.Datacenter,
		}).Info("✅ Using saved VMware credentials for VM addition")
	} else {
		// Validate manual entry fields
		if request.VCenter == "" || request.Username == "" || request.Password == "" || request.Datacenter == "" {
			h.writeErrorResponse(w, http.StatusBadRequest, "Either credential_id OR (vcenter, username, password, datacenter) must be provided")
			return
		}
		
		vcenter = request.VCenter
		username = request.Username
		password = request.Password
		datacenter = request.Datacenter
		
		log.WithField("vcenter_host", vcenter).Info("Using manual VMware credentials for VM addition")
	}

	log.WithField("vm_count", len(request.VMNames)).Info("Adding specific VMs to SHA")

	// Convert to service request format for internal processing
	discoveryRequest := services.DiscoveryRequest{
		VCenter:    vcenter,
		Username:   username,
		Password:   password,
		Datacenter: datacenter,
		Filter:     "", // No filter for specific VM add
	}

	// Use existing service method
	result, err := h.discoveryService.AddVMsToOMAWithoutJobs(r.Context(), discoveryRequest, request.VMNames)

	if err != nil {
		log.WithError(err).Error("Failed to add VMs to SHA")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to add VMs: "+err.Error())
		return
	}

	// Convert result to response format
	processedVMs := make([]ProcessedVMInfo, 0)

	// Add successful VMs
	for _, vm := range result.AddedVMs {
		processedVMs = append(processedVMs, ProcessedVMInfo{
			VMName:    vm.VMName,
			Success:   true,
			ContextID: &vm.ContextID,
		})
	}

	// Add failed VMs
	for _, vm := range result.FailedVMs {
		processedVMs = append(processedVMs, ProcessedVMInfo{
			VMName:  vm.VMName,
			Success: false,
			Error:   &vm.Error,
		})
	}

	response := AddVMsResponse{
		Success:      true,
		Message:      "VMs successfully processed",
		VMsAdded:     result.SuccessfullyAdded,
		VMsFailed:    result.Failed,
		TotalVMs:     len(request.VMNames),
		AddedAt:      time.Now().UTC(),
		ProcessedVMs: processedVMs,
	}

	log.WithFields(log.Fields{
		"vms_added":  result.SuccessfullyAdded,
		"vms_failed": result.Failed,
		"total_vms":  len(request.VMNames),
	}).Info("VMs added to SHA successfully")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetUngroupedVMs returns VMs that have been discovered but not assigned to any machine group
// GET /api/v1/discovery/ungrouped-vms
func (h *EnhancedDiscoveryHandler) GetUngroupedVMs(w http.ResponseWriter, r *http.Request) {
	log.Info("Retrieving ungrouped VMs")

	ctx := r.Context()
	vmContexts, err := h.discoveryService.GetDiscoveredVMsWithoutGroups(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get ungrouped VMs")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve ungrouped VMs: "+err.Error())
		return
	}

	// Convert to response format
	ungroupedVMs := make([]UngroupedVMInfo, 0, len(vmContexts))
	for _, vm := range vmContexts {
		ungroupedVM := UngroupedVMInfo{
			ContextID:        vm.ContextID,
			VMName:           vm.VMName,
			VMPath:           vm.VMPath,
			VCenterHost:      vm.VCenterHost,
			Datacenter:       vm.Datacenter,
			CurrentStatus:    vm.CurrentStatus,
			AutoAdded:        vm.AutoAdded,
			SchedulerEnabled: vm.SchedulerEnabled,
			CPUCount:         vm.CPUCount,
			MemoryMB:         vm.MemoryMB,
			OSType:           vm.OSType,
			PowerState:       vm.PowerState,
			CreatedAt:        vm.CreatedAt,
			LastJobAt:        vm.LastJobAt,
		}
		ungroupedVMs = append(ungroupedVMs, ungroupedVM)
	}

	response := UngroupedVMsResponse{
		VMs:         ungroupedVMs,
		Count:       len(ungroupedVMs),
		RetrievedAt: time.Now(),
	}

	log.WithField("count", len(ungroupedVMs)).Info("Retrieved ungrouped VMs")
	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetDiscoveryPreview provides a preview of VMs that would be discovered without actually adding them
// POST /api/v1/discovery/preview
func (h *EnhancedDiscoveryHandler) GetDiscoveryPreview(w http.ResponseWriter, r *http.Request) {
	var request services.DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate request
	if request.VCenter == "" || request.Username == "" || request.Password == "" || request.Datacenter == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "vCenter, username, password, and datacenter are required")
		return
	}

	log.WithFields(log.Fields{
		"vcenter":    request.VCenter,
		"datacenter": request.Datacenter,
		"filter":     request.Filter,
	}).Info("Getting discovery preview")

	ctx := r.Context()
	start := time.Now()

	// Discover VMs from SNA (preview only)
	snaResponse, err := h.discoveryService.DiscoverVMsFromVMA(ctx, request)
	if err != nil {
		log.WithError(err).Error("Failed to get discovery preview")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Discovery preview failed: "+err.Error())
		return
	}

	// Check which VMs already exist
	previewVMs := make([]DiscoveredVMInfo, 0, len(snaResponse.VMs))
	for _, vm := range snaResponse.VMs {
		existing, _ := h.vmContextRepo.GetVMContextByName(vm.Name)

		previewVM := DiscoveredVMInfo{
			ID:         vm.ID,
			Name:       vm.Name,
			Path:       vm.Path,
			PowerState: vm.PowerState,
			GuestOS:    vm.GuestOS,
			MemoryMB:   vm.MemoryMB,
			NumCPU:     vm.NumCPU,
			VMXVersion: vm.VMXVersion,
			Existing:   existing != nil,
		}

		if existing != nil {
			previewVM.ContextID = existing.ContextID
		}

		previewVMs = append(previewVMs, previewVM)
	}

	response := struct {
		VMs             []DiscoveredVMInfo `json:"vms"`
		TotalDiscovered int                `json:"total_discovered"`
		NewVMs          int                `json:"new_vms"`
		ExistingVMs     int                `json:"existing_vms"`
		ProcessingTime  time.Duration      `json:"processing_time"`
		VCenter         string             `json:"vcenter"`
		Datacenter      string             `json:"datacenter"`
		Filter          string             `json:"filter,omitempty"`
	}{
		VMs:             previewVMs,
		TotalDiscovered: len(previewVMs),
		ProcessingTime:  time.Since(start),
		VCenter:         request.VCenter,
		Datacenter:      request.Datacenter,
		Filter:          request.Filter,
	}

	// Count new vs existing
	for _, vm := range previewVMs {
		if vm.Existing {
			response.ExistingVMs++
		} else {
			response.NewVMs++
		}
	}

	log.WithFields(log.Fields{
		"total_discovered": response.TotalDiscovered,
		"new_vms":          response.NewVMs,
		"existing_vms":     response.ExistingVMs,
		"processing_time":  response.ProcessingTime,
	}).Info("Discovery preview completed")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse writes a JSON response
func (h *EnhancedDiscoveryHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to write JSON response")
	}
}

// writeErrorResponse writes an error response
func (h *EnhancedDiscoveryHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}

// GetUngroupedVMContexts returns VMs that have been discovered but not assigned to any machine group
// GET /api/v1/vm-contexts/ungrouped (alias for GetUngroupedVMs)
func (h *EnhancedDiscoveryHandler) GetUngroupedVMContexts(w http.ResponseWriter, r *http.Request) {
	// This is just an alias for the existing GetUngroupedVMs method
	h.GetUngroupedVMs(w, r)
}
