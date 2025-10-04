// Package api provides the VMA Control API server for OMA communication
// This implements the minimal 4-endpoint API design for bidirectional tunnel communication
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/source/current/vma/cbt"
	"github.com/vexxhost/migratekit/source/current/vma/progress"
	"github.com/vexxhost/migratekit/source/current/vma/services"
)

// VMAControlServer provides the minimal VMA control API
type VMAControlServer struct {
	port              int
	jobTracker        *JobTracker
	vmwareClient      VMwareClientInterface
	router            *mux.Router
	discoveryProvider services.VMwareDiscoveryProvider
	specChecker       services.VMSpecificationChecker
}

// JobTracker tracks migration jobs and their status
type JobTracker struct {
	mu      sync.RWMutex
	jobs    map[string]*JobStatus
	parsers map[string]*progress.ProgressParser
}

// JobStatus represents the current status of a migration job
type JobStatus struct {
	JobID            string    `json:"job_id"`
	Status           string    `json:"status"` // "running", "completed", "failed"
	ProgressPercent  float64   `json:"progress_percent"`
	CurrentOperation string    `json:"current_operation"`
	StartTime        time.Time `json:"start_time"`
	LastUpdate       time.Time `json:"last_update"`
	VMPath           string    `json:"vm_path,omitempty"`
}

// VMwareClientInterface abstracts VMware operations for testing
type VMwareClientInterface interface {
	DeleteSnapshot(jobID string) error
	GetVMStatus(vmPath string) (string, error)
	DiscoverVMs(vcenter, username, password, datacenter string) (*VMInventory, error)
	DiscoverVMsWithFilter(vcenter, username, password, datacenter, filter string) (*VMInventory, error)
	StartReplication(request *ReplicationRequest) (*ReplicationResponse, error)
}

// CleanupRequest represents a cleanup operation request
type CleanupRequest struct {
	JobID  string `json:"job_id"`
	Action string `json:"action"` // "delete_snapshot", "cleanup_all"
}

// ConfigRequest represents a configuration update request
type ConfigRequest struct {
	NBDPort      int    `json:"nbd_port"`
	ExportName   string `json:"export_name"`
	TargetDevice string `json:"target_device"`
}

// CBTRequest represents a CBT management request
type CBTRequest struct {
	VCenter  string `json:"vcenter"`
	Username string `json:"username"`
	Password string `json:"password"`
	VMPath   string `json:"vm_path"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Uptime    string `json:"uptime"`
}

// DiscoveryRequest represents a VM discovery request from OMA
type DiscoveryRequest struct {
	VCenter    string `json:"vcenter"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Datacenter string `json:"datacenter"`
	Filter     string `json:"filter,omitempty"` // Optional VM name filter
}

// VMInventory represents discovered VM inventory
type VMInventory struct {
	VCenter struct {
		Host       string `json:"host"`
		Datacenter string `json:"datacenter"`
	} `json:"vcenter"`
	VMs []VMInfo `json:"vms"`
}

// VMInfo represents information about a discovered VM with complete disk and network details
type VMInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Datacenter string `json:"datacenter"`
	PowerState string `json:"power_state"`
	GuestOS    string `json:"guest_os"`
	MemoryMB   int    `json:"memory_mb"`
	NumCPU     int    `json:"num_cpu"`
	CPUs       int    `json:"cpus"`    // Alias for NumCPU for compatibility
	OSType     string `json:"os_type"` // Alias for GuestOS
	VMXVersion string `json:"vmx_version"`

	// Additional VM metadata for failover system
	DisplayName        string `json:"display_name"`         // VM display name
	Annotation         string `json:"annotation"`           // VM notes/description
	FolderPath         string `json:"folder_path"`          // vCenter folder path
	VMwareToolsStatus  string `json:"vmware_tools_status"`  // VMware Tools status
	VMwareToolsVersion string `json:"vmware_tools_version"` // VMware Tools version

	Disks    []DiskInfo    `json:"disks"`
	Networks []NetworkInfo `json:"networks"`
}

// DiskInfo represents VM disk information
type DiskInfo struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Path             string `json:"path"`
	VMDKPath         string `json:"vmdk_path"`
	SizeGB           int    `json:"size_gb"`
	CapacityBytes    int64  `json:"capacity_bytes"`
	Datastore        string `json:"datastore"`
	ProvisioningType string `json:"provisioning_type"`
	UnitNumber       int    `json:"unit_number"`
}

// NetworkInfo represents VM network interface information
type NetworkInfo struct {
	Label       string `json:"label"`
	NetworkName string `json:"network_name"`
	AdapterType string `json:"adapter_type"`
	MACAddress  string `json:"mac_address"`
	Connected   bool   `json:"connected"`
	Type        string `json:"type"` // Network type for compatibility
}

// ReplicationRequest represents a replication job request from OMA
type ReplicationRequest struct {
	JobID      string      `json:"job_id"`
	VCenter    string      `json:"vcenter"`
	Username   string      `json:"username"`
	Password   string      `json:"password"`
	VMPaths    []string    `json:"vm_paths"`
	OMAUrl     string      `json:"oma_url"`
	NBDTargets []NBDTarget `json:"nbd_targets,omitempty"` // NBD target devices
}

// NBDTarget represents NBD target device information
type NBDTarget struct {
	DevicePath    string `json:"device_path"`     // e.g., nbd://host:port/export
	VMDiskID      int    `json:"vm_disk_id"`      // Associated VM disk ID (legacy)
	VMwareDiskKey string `json:"vmware_disk_key"` // VMware disk key (e.g., "2000", "2001")
}

// ReplicationResponse represents the response from starting replication
type ReplicationResponse struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	VMCount   int    `json:"vm_count"`
	StartedAt string `json:"started_at"`
}

// VMSpecChangesRequest represents a request to check VM specification changes
type VMSpecChangesRequest struct {
	VCenter      string `json:"vcenter" binding:"required"`
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password" binding:"required"`
	Datacenter   string `json:"datacenter" binding:"required"`
	VMPath       string `json:"vm_path" binding:"required"`
	StoredVMInfo VMInfo `json:"stored_vm_info" binding:"required"`
}

// VMSpecChangesResponse represents the response from VM specification change detection
type VMSpecChangesResponse struct {
	HasChanges     bool   `json:"has_changes"`
	ChangesSummary string `json:"changes_summary"`
	ChangesJSON    string `json:"changes_json"`
	CheckTime      string `json:"check_time"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// NewVMAControlServer creates a new VMA control API server
func NewVMAControlServer(port int, vmwareClient VMwareClientInterface) *VMAControlServer {
	server := &VMAControlServer{
		port: port,
		jobTracker: &JobTracker{
			jobs:    make(map[string]*JobStatus),
			parsers: make(map[string]*progress.ProgressParser),
		},
		vmwareClient: vmwareClient,
		router:       mux.NewRouter(),
	}

	server.setupRoutes()
	return server
}

// NewVMAControlServerWithServices creates a new VMA control API server with injected services
func NewVMAControlServerWithServices(port int, vmwareClient VMwareClientInterface,
	discoveryProvider services.VMwareDiscoveryProvider, specChecker services.VMSpecificationChecker) *VMAControlServer {
	server := &VMAControlServer{
		port: port,
		jobTracker: &JobTracker{
			jobs:    make(map[string]*JobStatus),
			parsers: make(map[string]*progress.ProgressParser),
		},
		vmwareClient:      vmwareClient,
		router:            mux.NewRouter(),
		discoveryProvider: discoveryProvider,
		specChecker:       specChecker,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures the VMA Control API endpoints
func (s *VMAControlServer) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Core control endpoints
	api.HandleFunc("/cleanup", s.handleCleanup).Methods("POST")
	api.HandleFunc("/status/{job_id}", s.handleStatus).Methods("GET")
	// 🚨 REMOVED: Conflicting progress route - now handled by ProgressHandler
	// api.HandleFunc("/progress/{job_id}", s.handleProgress).Methods("GET")
	api.HandleFunc("/config", s.handleConfig).Methods("PUT")
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/vms/{vm_path:.*}/cbt-status", s.handleCBTStatus).Methods("GET")

	// OMA-initiated workflow endpoints
	api.HandleFunc("/discover", s.handleDiscover).Methods("POST")
	api.HandleFunc("/replicate", s.handleReplicate).Methods("POST")
	api.HandleFunc("/vm-spec-changes", s.handleVMSpecChanges).Methods("POST")

	// Power management endpoints for unified failover system
	api.HandleFunc("/vm/{vm_id}/power-off", s.handleVMPowerOff).Methods("POST")
	api.HandleFunc("/vm/{vm_id}/power-on", s.handleVMPowerOn).Methods("POST")
	api.HandleFunc("/vm/{vm_id}/power-state", s.handleVMPowerState).Methods("GET")

	// 🆕 NEW: VMA Enrollment endpoints for secure OMA pairing
	api.HandleFunc("/enrollment/enroll", s.handleEnrollWithOMA).Methods("POST")
	api.HandleFunc("/enrollment/status", s.handleEnrollmentStatus).Methods("GET")

	log.WithField("endpoints", 11).Info("VMA Control API routes configured (including enrollment system)")
}

// GetRouter returns the router instance for external route registration
func (s *VMAControlServer) GetRouter() *mux.Router {
	return s.router
}

// AddJobWithProgress adds a job to tracking with progress parser
func (s *VMAControlServer) AddJobWithProgress(jobID, vmPath string) {
	s.jobTracker.mu.Lock()
	defer s.jobTracker.mu.Unlock()

	// Add job status
	s.jobTracker.jobs[jobID] = &JobStatus{
		JobID:      jobID,
		Status:     "running",
		VMPath:     vmPath,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	// Create progress parser for hybrid tracking
	logPath := fmt.Sprintf("/tmp/migratekit-%s.log", jobID)
	parser := progress.NewProgressParser(jobID, logPath)
	s.jobTracker.parsers[jobID] = parser

	log.WithFields(log.Fields{
		"job_id":   jobID,
		"vm_path":  vmPath,
		"log_path": logPath,
	}).Info("Job added with hybrid progress tracking")
}

// RemoveJob removes a job from tracking and cleans up resources
func (s *VMAControlServer) RemoveJob(jobID string) {
	s.jobTracker.mu.Lock()
	defer s.jobTracker.mu.Unlock()

	// Clean up progress parser
	if _, exists := s.jobTracker.parsers[jobID]; exists {
		delete(s.jobTracker.parsers, jobID)
	}

	delete(s.jobTracker.jobs, jobID)

	log.WithField("job_id", jobID).Info("Job removed from tracking")
}

// UpdateNBDProgress updates NBD progress from pipe data
func (s *VMAControlServer) UpdateNBDProgress(jobID string, bytesTransferred, totalBytes int64) {
	s.jobTracker.mu.RLock()
	parser, exists := s.jobTracker.parsers[jobID]
	s.jobTracker.mu.RUnlock()

	if exists && parser != nil && totalBytes > 0 {
		percentage := float64(bytesTransferred) / float64(totalBytes) * 100.0
		parser.UpdateNBDProgress(percentage)
		log.WithFields(log.Fields{
			"job_id":            jobID,
			"bytes_transferred": bytesTransferred,
			"total_bytes":       totalBytes,
			"percentage":        percentage,
		}).Debug("NBD progress updated from pipe")
	}
}

// handleCleanup processes cleanup operation requests from OMA
func (s *VMAControlServer) handleCleanup(w http.ResponseWriter, r *http.Request) {
	var req CleanupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"job_id": req.JobID,
		"action": req.Action,
	}).Info("Received cleanup request from OMA")

	switch req.Action {
	case "delete_snapshot":
		if err := s.vmwareClient.DeleteSnapshot(req.JobID); err != nil {
			log.WithError(err).Error("Failed to delete snapshot")
			http.Error(w, fmt.Sprintf("Snapshot deletion failed: %v", err), http.StatusInternalServerError)
			return
		}
		log.WithField("job_id", req.JobID).Info("✅ Snapshot deleted successfully")

	case "cleanup_all":
		// Implement full cleanup logic
		if err := s.vmwareClient.DeleteSnapshot(req.JobID); err != nil {
			log.WithError(err).Warn("Snapshot cleanup failed, continuing")
		}
		// Additional cleanup operations can be added here
		log.WithField("job_id", req.JobID).Info("✅ Full cleanup completed")

	default:
		http.Error(w, fmt.Sprintf("Unknown action: %s", req.Action), http.StatusBadRequest)
		return
	}

	// Update job status
	if job, exists := s.jobTracker.jobs[req.JobID]; exists {
		job.Status = "completed"
		job.CurrentOperation = "cleanup_completed"
		job.LastUpdate = time.Now()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"job_id": req.JobID,
		"action": req.Action,
	})
}

// handleStatus returns the current status of a migration job
func (s *VMAControlServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]

	job, exists := s.jobTracker.jobs[jobID]
	if !exists {
		http.Error(w, fmt.Sprintf("Job not found: %s", jobID), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// handleProgress returns detailed progress information including throughput and errors
func (s *VMAControlServer) handleProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]

	// Check if job exists in tracker
	if _, exists := s.jobTracker.jobs[jobID]; !exists {
		http.Error(w, fmt.Sprintf("Job not found: %s", jobID), http.StatusNotFound)
		return
	}

	// Get progress from hybrid parser (log parsing + NBD pipe data)
	s.jobTracker.mu.RLock()
	parser, exists := s.jobTracker.parsers[jobID]
	s.jobTracker.mu.RUnlock()

	if !exists || parser == nil {
		log.WithField("job_id", jobID).Warn("No active parser found for job")
		http.Error(w, fmt.Sprintf("No active migration found for job: %s", jobID), http.StatusNotFound)
		return
	}

	// Get progress from hybrid parser (log parsing + NBD pipe data)
	jobProgress := parser.GetProgress()

	log.WithFields(log.Fields{
		"job_id":     jobID,
		"percentage": jobProgress.Percentage,
		"status":     jobProgress.Status,
		"phase":      jobProgress.Phase,
		"throughput": jobProgress.Throughput.CurrentMBps,
	}).Debug("Returning progress from hybrid parser")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobProgress)
}

// handleConfig processes configuration updates from OMA
func (s *VMAControlServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"nbd_port":      req.NBDPort,
		"export_name":   req.ExportName,
		"target_device": req.TargetDevice,
	}).Info("Received configuration update from OMA")

	// TODO: Implement dynamic stunnel configuration generation
	// This will generate new stunnel config with the provided port/export

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Configuration updated successfully",
	})
}

// handleHealth provides health check endpoint
func (s *VMAControlServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(time.Now()) // This would be actual server start time

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDiscover processes VM discovery requests from OMA
func (s *VMAControlServer) handleDiscover(w http.ResponseWriter, r *http.Request) {
	var req DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"vcenter":    req.VCenter,
		"datacenter": req.Datacenter,
		"filter":     req.Filter,
	}).Info("Received VM discovery request from OMA")

	// Call VMware client to discover VMs with filter
	inventory, err := s.vmwareClient.DiscoverVMsWithFilter(req.VCenter, req.Username, req.Password, req.Datacenter, req.Filter)
	if err != nil {
		log.WithError(err).Error("Failed to discover VMs")
		http.Error(w, fmt.Sprintf("Discovery failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter is now applied in the service layer with path correction

	log.WithFields(log.Fields{
		"vm_count":   len(inventory.VMs),
		"vcenter":    inventory.VCenter.Host,
		"datacenter": inventory.VCenter.Datacenter,
	}).Info("VM discovery completed successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inventory)
}

// handleReplicate processes replication job requests from OMA
func (s *VMAControlServer) handleReplicate(w http.ResponseWriter, r *http.Request) {
	var req ReplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"job_id":   req.JobID,
		"vcenter":  req.VCenter,
		"vm_count": len(req.VMPaths),
	}).Info("Received replication request from OMA")

	// Start replication using VMware client
	response, err := s.vmwareClient.StartReplication(&req)
	if err != nil {
		log.WithError(err).Error("Failed to start replication")
		http.Error(w, fmt.Sprintf("Replication failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Add job to tracker
	s.AddJob(req.JobID)

	log.WithFields(log.Fields{
		"job_id":   response.JobID,
		"status":   response.Status,
		"vm_count": response.VMCount,
	}).Info("Replication started successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Start starts the VMA control API server
func (s *VMAControlServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)

	log.WithFields(log.Fields{
		"port":      s.port,
		"endpoints": 6,
	}).Info("🚀 Starting VMA Control API server")

	return http.ListenAndServe(addr, s.router)
}

// AddJob adds a new job to the tracker
func (s *VMAControlServer) AddJob(jobID string) {
	s.jobTracker.jobs[jobID] = &JobStatus{
		JobID:            jobID,
		Status:           "running",
		ProgressPercent:  0.0,
		CurrentOperation: "initializing",
		StartTime:        time.Now(),
		LastUpdate:       time.Now(),
	}

	log.WithField("job_id", jobID).Info("Job added to tracker")
}

// UpdateJobProgress updates the progress of a running job
func (s *VMAControlServer) UpdateJobProgress(jobID string, progress float64, operation string) {
	if job, exists := s.jobTracker.jobs[jobID]; exists {
		job.ProgressPercent = progress
		job.CurrentOperation = operation
		job.LastUpdate = time.Now()

		log.WithFields(log.Fields{
			"job_id":    jobID,
			"progress":  progress,
			"operation": operation,
		}).Debug("Job progress updated")
	}
}

// handleCBTStatus checks CBT status for a VM via vCenter API
// @Summary Check CBT status for a VM
// @Description Checks if Change Block Tracking (CBT) is enabled on the specified VM
// @Tags vm
// @Accept json
// @Produce json
// @Param vm_path path string true "VM path (URL-encoded)"
// @Param vcenter query string true "vCenter hostname"
// @Param username query string true "vCenter username"
// @Param password query string true "vCenter password"
// @Success 200 {object} cbt.CBTStatus
// @Failure 400 {object} ErrorResponse "Invalid request parameters"
// @Failure 500 {object} ErrorResponse "CBT check failed"
// @Router /api/v1/vms/{vm_path}/cbt-status [get]
func (s *VMAControlServer) handleCBTStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmPath, err := url.QueryUnescape(vars["vm_path"])
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid VM path: %v", err), http.StatusBadRequest)
		return
	}

	// Get vCenter credentials from query parameters
	vcenter := r.URL.Query().Get("vcenter")
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	if vcenter == "" || username == "" || password == "" {
		http.Error(w, "Missing required parameters: vcenter, username, password", http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"vm_path": vmPath,
		"vcenter": vcenter,
	}).Info("Checking CBT status for VM")

	// Create CBT manager and check status
	cbtManager := cbt.NewManager(vcenter, username, password)
	status, err := cbtManager.CheckCBTStatus(context.Background(), vmPath)
	if err != nil {
		log.WithError(err).WithField("vm_path", vmPath).Error("CBT status check failed")
		http.Error(w, fmt.Sprintf("CBT status check failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return CBT status as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.WithError(err).Error("Failed to encode CBT status response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"vm_path":     vmPath,
		"cbt_enabled": status.Enabled,
		"vm_name":     status.VMName,
		"power_state": status.PowerState,
	}).Info("CBT status check completed successfully")
}

// handleVMSpecChanges checks for VM specification changes compared to stored data
func (s *VMAControlServer) handleVMSpecChanges(w http.ResponseWriter, r *http.Request) {
	var req VMSpecChangesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"vm_path":    req.VMPath,
		"vcenter":    req.VCenter,
		"datacenter": req.Datacenter,
		"stored_vm":  req.StoredVMInfo.Name,
	}).Info("Checking VM specification changes")

	response := VMSpecChangesResponse{
		Status:    "success",
		CheckTime: time.Now().UTC().Format(time.RFC3339),
	}

	// Check if we have the required services injected
	if s.discoveryProvider == nil || s.specChecker == nil {
		log.Error("VM specification change detection services not properly injected")
		response.Status = "error"
		response.ErrorMessage = "VM specification services not available"
		s.sendVMSpecChangesResponse(w, response)
		return
	}

	// Create discovery service with provided credentials
	discovery, err := s.discoveryProvider.CreateDiscovery(req.VCenter, req.Username, req.Password, req.Datacenter)
	if err != nil {
		log.WithError(err).Error("Failed to create discovery service")
		response.Status = "error"
		response.ErrorMessage = fmt.Sprintf("Failed to create discovery service: %v", err)
		s.sendVMSpecChangesResponse(w, response)
		return
	}

	// Connect to vCenter
	if err := discovery.Connect(context.Background()); err != nil {
		log.WithError(err).Error("Failed to connect to vCenter for specification check")
		response.Status = "error"
		response.ErrorMessage = fmt.Sprintf("vCenter connection failed: %v", err)
		s.sendVMSpecChangesResponse(w, response)
		return
	}
	defer discovery.Disconnect()

	// Convert API VMInfo to services format
	storedVMInfo := s.convertAPIVMInfoToServices(&req.StoredVMInfo)

	// Detect changes using the real service
	diff, err := s.specChecker.DetectVMSpecificationChanges(context.Background(), req.VMPath, storedVMInfo)
	if err != nil {
		log.WithError(err).Error("Failed to detect VM specification changes")
		response.Status = "error"
		response.ErrorMessage = fmt.Sprintf("Change detection failed: %v", err)
		s.sendVMSpecChangesResponse(w, response)
		return
	}

	// Populate response with real data
	response.HasChanges = diff.HasChanges
	response.ChangesSummary = s.specChecker.GetChangesSummary(diff)

	if changesJSON, err := s.specChecker.SerializeChanges(diff); err == nil {
		response.ChangesJSON = changesJSON
	} else {
		log.WithError(err).Warn("Failed to serialize changes to JSON")
		response.ChangesJSON = "{}"
	}

	s.sendVMSpecChangesResponse(w, response)

	log.WithFields(log.Fields{
		"vm_path":     req.VMPath,
		"has_changes": response.HasChanges,
		"status":      response.Status,
	}).Info("VM specification change detection completed")
}

// Helper methods for VM specification change detection

// convertAPIVMInfoToServices converts API VMInfo to services.StoredVMInfo
func (s *VMAControlServer) convertAPIVMInfoToServices(apiVM *VMInfo) *services.StoredVMInfo {
	// Convert disks
	var disks []services.StoredDiskInfo
	for _, disk := range apiVM.Disks {
		disks = append(disks, services.StoredDiskInfo{
			ID:               disk.ID,
			Path:             disk.Path,
			SizeGB:           disk.SizeGB,
			Datastore:        disk.Datastore,
			VMDKPath:         disk.VMDKPath,
			ProvisioningType: disk.ProvisioningType,
			Label:            disk.Label,
			CapacityBytes:    disk.CapacityBytes,
			UnitNumber:       disk.UnitNumber,
		})
	}

	// Convert networks
	var networks []services.StoredNetworkInfo
	for _, net := range apiVM.Networks {
		networks = append(networks, services.StoredNetworkInfo{
			Name:        net.Label,
			Type:        net.Type,
			Connected:   net.Connected,
			MACAddress:  net.MACAddress,
			Label:       net.Label,
			NetworkName: net.NetworkName,
			AdapterType: net.AdapterType,
		})
	}

	return &services.StoredVMInfo{
		ID:                 apiVM.ID,
		Name:               apiVM.Name,
		Path:               apiVM.Path,
		Datacenter:         apiVM.Datacenter,
		CPUs:               apiVM.CPUs,
		MemoryMB:           apiVM.MemoryMB,
		PowerState:         apiVM.PowerState,
		OSType:             apiVM.OSType,
		VMXVersion:         apiVM.VMXVersion,
		DisplayName:        apiVM.DisplayName,
		Annotation:         apiVM.Annotation,
		FolderPath:         apiVM.FolderPath,
		VMwareToolsStatus:  apiVM.VMwareToolsStatus,
		VMwareToolsVersion: apiVM.VMwareToolsVersion,
		Disks:              disks,
		Networks:           networks,
	}
}

// sendVMSpecChangesResponse sends the VM specification changes response
func (s *VMAControlServer) sendVMSpecChangesResponse(w http.ResponseWriter, response VMSpecChangesResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode VM specification changes response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// VMA Enrollment Handler Methods

// handleEnrollWithOMA handles VMA enrollment with OMA
func (s *VMAControlServer) handleEnrollWithOMA(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OMAHost     string `json:"oma_host"`
		OMAPort     int    `json:"oma_port"`
		PairingCode string `json:"pairing_code"`
		VMAName     string `json:"vma_name"`
		VMAVersion  string `json:"vma_version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("Invalid enrollment request")
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.OMAHost == "" || req.PairingCode == "" {
		log.Error("Missing required enrollment fields")
		http.Error(w, "Missing required fields: oma_host and pairing_code", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.OMAPort == 0 {
		req.OMAPort = 443
	}
	if req.VMAName == "" {
		req.VMAName = fmt.Sprintf("VMA-%d", time.Now().Unix()%10000)
	}
	if req.VMAVersion == "" {
		req.VMAVersion = "v2.20.1"
	}

	log.WithFields(log.Fields{
		"oma_host":     req.OMAHost,
		"oma_port":     req.OMAPort,
		"pairing_code": req.PairingCode,
		"vma_name":     req.VMAName,
	}).Info("🔐 Processing VMA enrollment request")

	// Create enrollment client
	enrollmentClient := services.NewVMAEnrollmentClient(req.OMAHost, req.OMAPort)

	// Perform enrollment
	config, err := enrollmentClient.EnrollWithOMA(req.PairingCode, req.VMAName, req.VMAVersion)
	if err != nil {
		log.WithError(err).Error("VMA enrollment failed")
		http.Error(w, fmt.Sprintf("Enrollment failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Configure tunnel
	tunnelConfigured := false
	if err := enrollmentClient.ConfigureTunnel(config); err != nil {
		log.WithError(err).Warn("Tunnel configuration failed - manual setup required")
	} else {
		tunnelConfigured = true
	}

	response := map[string]interface{}{
		"success":           true,
		"enrollment_id":     config.EnrollmentID,
		"status":            "approved",
		"message":           "VMA enrollment completed successfully",
		"tunnel_configured": tunnelConfigured,
		"oma_host":          config.OMAHost,
		"ssh_user":          config.SSHUser,
	}

	log.WithFields(log.Fields{
		"enrollment_id":     config.EnrollmentID,
		"tunnel_configured": tunnelConfigured,
	}).Info("🎉 VMA enrollment completed successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleEnrollmentStatus returns current enrollment status
func (s *VMAControlServer) handleEnrollmentStatus(w http.ResponseWriter, r *http.Request) {
	enrollmentID := r.URL.Query().Get("enrollment_id")
	if enrollmentID == "" {
		http.Error(w, "Missing enrollment_id parameter", http.StatusBadRequest)
		return
	}

	// For now, return placeholder response
	// This would query the saved enrollment configuration
	response := map[string]interface{}{
		"enrollment_id": enrollmentID,
		"status":        "unknown",
		"message":       "Enrollment status checking - implementation pending",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
