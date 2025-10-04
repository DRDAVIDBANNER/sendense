// Package services provides enhanced VM discovery capabilities for scheduler system
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
)

// EnhancedDiscoveryService provides VM discovery without immediate job creation
type EnhancedDiscoveryService struct {
	vmContextRepo *database.VMReplicationContextRepository
	db            *database.SchedulerRepository
	tracker       *joblog.Tracker
	vmaBaseURL    string // VMA API URL via tunnel (e.g., "http://localhost:9081")
}

// NewEnhancedDiscoveryService creates a new enhanced discovery service
func NewEnhancedDiscoveryService(vmContextRepo *database.VMReplicationContextRepository,
	db *database.SchedulerRepository, tracker *joblog.Tracker, vmaBaseURL string) *EnhancedDiscoveryService {
	return &EnhancedDiscoveryService{
		vmContextRepo: vmContextRepo,
		db:            db,
		tracker:       tracker,
		vmaBaseURL:    vmaBaseURL,
	}
}

// DiscoveryRequest represents a VM discovery request to VMA
type DiscoveryRequest struct {
	VCenter    string `json:"vcenter" binding:"required"`
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Datacenter string `json:"datacenter" binding:"required"`
	Filter     string `json:"filter,omitempty"` // Optional VM name filter
}

// VMADiscoveryResponse represents the response from VMA discovery endpoint
type VMADiscoveryResponse struct {
	VCenter struct {
		Host       string `json:"host"`
		Datacenter string `json:"datacenter"`
	} `json:"vcenter"`
	VMs []VMAVMInfo `json:"vms"`
}

// VMAVMInfo represents VM info from VMA discovery
type VMAVMInfo struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	PowerState string        `json:"power_state"`
	GuestOS    string        `json:"guest_os"`
	MemoryMB   int           `json:"memory_mb"`
	NumCPU     int           `json:"num_cpu"`
	VMXVersion string        `json:"vmx_version,omitempty"`
	Disks      []VMADiskInfo `json:"disks"`
	Networks   []VMANetworkInfo `json:"networks"`
}

// VMADiskInfo represents disk information from VMA
type VMADiskInfo struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Path          string `json:"path"`
	SizeGB        int    `json:"size_gb"`
	CapacityBytes int64  `json:"capacity_bytes"`
	Datastore     string `json:"datastore"`
}

// VMANetworkInfo represents network information from VMA
type VMANetworkInfo struct {
	Label       string `json:"label"`
	NetworkName string `json:"network_name"`
	MACAddress  string `json:"mac_address"`
}

// BulkAddResult represents the result of bulk VM addition
type BulkAddResult struct {
	TotalRequested     int                `json:"total_requested"`
	SuccessfullyAdded  int                `json:"successfully_added"`
	Skipped            int                `json:"skipped"`
	Failed             int                `json:"failed"`
	AddedVMs           []VMContextSummary `json:"added_vms"`
	SkippedVMs         []VMSkipReason     `json:"skipped_vms"`
	FailedVMs          []VMFailureReason  `json:"failed_vms"`
	DiscoveryDuration  time.Duration      `json:"discovery_duration"`
	ProcessingDuration time.Duration      `json:"processing_duration"`
}

// VMContextSummary represents a summary of created VM context
type VMContextSummary struct {
	ContextID  string `json:"context_id"`
	VMName     string `json:"vm_name"`
	VMPath     string `json:"vm_path"`
	Status     string `json:"status"`
	AutoAdded  bool   `json:"auto_added"`
	CPUCount   int    `json:"cpu_count"`
	MemoryMB   int    `json:"memory_mb"`
	OSType     string `json:"os_type"`
	PowerState string `json:"power_state"`
}

// VMSkipReason represents why a VM was skipped
type VMSkipReason struct {
	VMName   string `json:"vm_name"`
	VMPath   string `json:"vm_path"`
	Reason   string `json:"reason"`
	Existing string `json:"existing_context_id,omitempty"`
}

// VMFailureReason represents why VM addition failed
type VMFailureReason struct {
	VMName string `json:"vm_name"`
	VMPath string `json:"vm_path"`
	Error  string `json:"error"`
}

// DiscoverVMsFromVMA calls VMA discovery endpoint and returns VM information
func (eds *EnhancedDiscoveryService) DiscoverVMsFromVMA(ctx context.Context, request DiscoveryRequest) (*VMADiscoveryResponse, error) {
	log := eds.tracker.Logger(ctx)

	log.Info("Starting VM discovery from VMA",
		"vcenter", request.VCenter,
		"datacenter", request.Datacenter,
		"filter", request.Filter)

	// Prepare request to VMA
	vmaURL := fmt.Sprintf("%s/api/v1/discover", eds.vmaBaseURL)

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal discovery request: %w", err)
	}

	// Make HTTP request to VMA
	httpReq, err := http.NewRequestWithContext(ctx, "POST", vmaURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call VMA discovery endpoint: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read VMA response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("VMA discovery failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse VMA response
	var vmaResponse VMADiscoveryResponse
	if err := json.Unmarshal(body, &vmaResponse); err != nil {
		return nil, fmt.Errorf("failed to parse VMA discovery response: %w", err)
	}

	log.Info("Successfully discovered VMs from VMA",
		"vm_count", len(vmaResponse.VMs),
		"vcenter", vmaResponse.VCenter.Host,
		"datacenter", vmaResponse.VCenter.Datacenter)

	return &vmaResponse, nil
}

// AddVMsToOMAWithoutJobs adds discovered VMs to OMA as VM contexts without creating replication jobs
func (eds *EnhancedDiscoveryService) AddVMsToOMAWithoutJobs(ctx context.Context,
	discoveryRequest DiscoveryRequest, selectedVMNames []string) (*BulkAddResult, error) {

	// Start job tracking
	ctx, jobID, err := eds.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "discovery",
		Operation: "bulk-add-vms",
		Owner:     stringPtr("scheduler-system"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start discovery job: %w", err)
	}

	log := eds.tracker.Logger(ctx)
	result := &BulkAddResult{
		TotalRequested: len(selectedVMNames),
		AddedVMs:       make([]VMContextSummary, 0),
		SkippedVMs:     make([]VMSkipReason, 0),
		FailedVMs:      make([]VMFailureReason, 0),
	}

	// Step 1: Discover VMs from VMA
	err = eds.tracker.RunStep(ctx, jobID, "vma-discovery", func(ctx context.Context) error {
		log := eds.tracker.Logger(ctx)
		start := time.Now()

		vmaResponse, err := eds.DiscoverVMsFromVMA(ctx, discoveryRequest)
		if err != nil {
			return fmt.Errorf("VMA discovery failed: %w", err)
		}

		result.DiscoveryDuration = time.Since(start)
		log.Info("VMA discovery completed",
			"duration", result.DiscoveryDuration,
			"discovered_vms", len(vmaResponse.VMs))

		// Step 2: Process discovered VMs
		start = time.Now()
		return eds.processDiscoveredVMs(ctx, vmaResponse, selectedVMNames, result)
	})

	// End job tracking
	if err != nil {
		eds.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return result, err
	}

	eds.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	log.Info("Bulk VM addition completed",
		"total_requested", result.TotalRequested,
		"successfully_added", result.SuccessfullyAdded,
		"skipped", result.Skipped,
		"failed", result.Failed,
		"discovery_duration", result.DiscoveryDuration,
		"processing_duration", result.ProcessingDuration)

	return result, nil
}

// processDiscoveredVMs processes VMs from VMA discovery and creates VM contexts
func (eds *EnhancedDiscoveryService) processDiscoveredVMs(ctx context.Context, vmaResponse *VMADiscoveryResponse,
	selectedVMNames []string, result *BulkAddResult) error {

	log := eds.tracker.Logger(ctx)
	start := time.Now()
	defer func() {
		result.ProcessingDuration = time.Since(start)
	}()

	// Create map for efficient lookup
	selectedMap := make(map[string]bool)
	for _, vmName := range selectedVMNames {
		selectedMap[vmName] = true
	}

	// Process each discovered VM
	for _, vm := range vmaResponse.VMs {
		// Skip if not in selected list (for filtered addition)
		if len(selectedVMNames) > 0 && !selectedMap[vm.Name] {
			continue
		}

		// Check if VM already exists
		existing, err := eds.vmContextRepo.GetVMContextByName(vm.Name)
		if err == nil && existing != nil {
			result.Skipped++
			result.SkippedVMs = append(result.SkippedVMs, VMSkipReason{
				VMName:   vm.Name,
				VMPath:   vm.Path,
				Reason:   "VM context already exists",
				Existing: existing.ContextID,
			})
			log.Info("Skipping existing VM", "vm_name", vm.Name, "existing_context", existing.ContextID)
			continue
		}

		// Create new VM context
		vcenterInfo := struct{ Host, Datacenter string }{
			Host:       vmaResponse.VCenter.Host,
			Datacenter: vmaResponse.VCenter.Datacenter,
		}
		contextID, err := eds.createVMContext(ctx, vm, vcenterInfo)
		if err != nil {
			result.Failed++
			result.FailedVMs = append(result.FailedVMs, VMFailureReason{
				VMName: vm.Name,
				VMPath: vm.Path,
				Error:  err.Error(),
			})
			log.Error("Failed to create VM context", "vm_name", vm.Name, "error", err)
			continue
		}

		// Success
		result.SuccessfullyAdded++
		result.AddedVMs = append(result.AddedVMs, VMContextSummary{
			ContextID:  contextID,
			VMName:     vm.Name,
			VMPath:     vm.Path,
			Status:     "discovered",
			AutoAdded:  true,
			CPUCount:   vm.NumCPU,
			MemoryMB:   vm.MemoryMB,
			OSType:     vm.GuestOS,
			PowerState: vm.PowerState,
		})

		log.Info("Successfully added VM context",
			"vm_name", vm.Name,
			"context_id", contextID,
			"auto_added", true)
	}

	return nil
}

// createVMContext creates a new VM context from discovered VM information
func (eds *EnhancedDiscoveryService) createVMContext(ctx context.Context, vm VMAVMInfo,
	vcenter struct{ Host, Datacenter string }) (string, error) {

	log := eds.tracker.Logger(ctx)

	// Generate context ID
	contextID := fmt.Sprintf("ctx-%s-%s", vm.Name, time.Now().Format("20060102-150405"))

	// Determine OS type from guest OS
	osType := eds.determineOSType(vm.GuestOS)

	// 🆕 Get active OSSEA config ID (if available)
	osseaConfigID := eds.getActiveOSSEAConfigID(ctx)
	if osseaConfigID > 0 {
		log.Info("Auto-assigning active OSSEA config to VM context", 
			"vm_name", vm.Name, "ossea_config_id", osseaConfigID)
	} else {
		log.Warn("No active OSSEA config found - VM will need config before replication", 
			"vm_name", vm.Name)
	}

	// Create VM context
	vmContext := database.VMReplicationContext{
		ContextID:        contextID,
		VMName:           vm.Name,
		VMwareVMID:       vm.ID,
		VMPath:           vm.Path,
		VCenterHost:      vcenter.Host,
		Datacenter:       vcenter.Datacenter,
		CurrentStatus:    "discovered",
		OSSEAConfigID:    &osseaConfigID, // 🆕 Auto-assign active config
		CPUCount:         &vm.NumCPU,
		MemoryMB:         &vm.MemoryMB,
		OSType:           &osType,
		PowerState:       &vm.PowerState,
		VMToolsVersion:   &vm.VMXVersion,
		AutoAdded:        true,
		SchedulerEnabled: true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		LastStatusChange: time.Now(),
	}

	// Save to database
	if err := eds.vmContextRepo.CreateVMContext(&vmContext); err != nil {
		return "", fmt.Errorf("failed to save VM context to database: %w", err)
	}

	log.Info("Created VM context",
		"context_id", contextID,
		"vm_name", vm.Name,
		"vm_path", vm.Path,
		"vcenter", vcenter.Host,
		"auto_added", true)

	return contextID, nil
}

// determineOSType determines OS type from VMware guest OS string
func (eds *EnhancedDiscoveryService) determineOSType(guestOS string) string {
	if guestOS == "" {
		return "unknown"
	}

	// Convert VMware guest OS to simplified OS type (case-insensitive)
	guestOSLower := strings.ToLower(guestOS)
	switch {
	case strings.Contains(guestOSLower, "windows"):
		return "windows"
	case strings.Contains(guestOSLower, "linux") ||
		strings.Contains(guestOSLower, "ubuntu") ||
		strings.Contains(guestOSLower, "rhel") ||
		strings.Contains(guestOSLower, "centos"):
		return "linux"
	case strings.Contains(guestOSLower, "darwin") ||
		strings.Contains(guestOSLower, "mac"):
		return "macos"
	default:
		return "other"
	}
}

// 🆕 getActiveOSSEAConfigID retrieves the active OSSEA configuration ID (where is_active = 1)
// Returns 0 if no active config found
func (eds *EnhancedDiscoveryService) getActiveOSSEAConfigID(ctx context.Context) int {
	log := eds.tracker.Logger(ctx)
	
	// Query database for active OSSEA config
	var result struct {
		ID int `db:"id"`
	}
	
	err := eds.db.GetGormDB().Raw(
		"SELECT id FROM ossea_configs WHERE is_active = 1 ORDER BY id DESC LIMIT 1").Scan(&result).Error
	
	if err != nil {
		log.Debug("No active OSSEA config found", "error", err)
		return 0
	}
	
	log.Debug("Found active OSSEA config", "config_id", result.ID)
	return result.ID
}

// GetDiscoveredVMsWithoutGroups returns VMs that have been discovered but not assigned to any machine group
func (eds *EnhancedDiscoveryService) GetDiscoveredVMsWithoutGroups(ctx context.Context) ([]database.VMReplicationContext, error) {
	log := eds.tracker.Logger(ctx)

	log.Info("Retrieving discovered VMs without group assignments")

	return eds.db.GetVMContextsWithoutGroups()
}
