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

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
)

// EnhancedDiscoveryService provides VM discovery without immediate job creation
type EnhancedDiscoveryService struct {
	vmContextRepo *database.VMReplicationContextRepository
	db            database.Connection // 🆕 Changed to proper Connection type for vm_disks creation
	tracker       *joblog.Tracker
	snaBaseURL    string // SNA API URL via tunnel (e.g., "http://localhost:9081")
}

// NewEnhancedDiscoveryService creates a new enhanced discovery service
func NewEnhancedDiscoveryService(vmContextRepo *database.VMReplicationContextRepository,
	db database.Connection, tracker *joblog.Tracker, snaBaseURL string) *EnhancedDiscoveryService {
	return &EnhancedDiscoveryService{
		vmContextRepo: vmContextRepo,
		db:            db,
		tracker:       tracker,
		snaBaseURL:    snaBaseURL,
	}
}

// DiscoveryRequest represents a VM discovery request to SNA
type DiscoveryRequest struct {
	VCenter    string `json:"vcenter" binding:"required"`
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Datacenter string `json:"datacenter" binding:"required"`
	Filter     string `json:"filter,omitempty"` // Optional VM name filter
}

// SNADiscoveryResponse represents the response from SNA discovery endpoint
type SNADiscoveryResponse struct {
	VCenter struct {
		Host       string `json:"host"`
		Datacenter string `json:"datacenter"`
	} `json:"vcenter"`
	VMs []SNAVMInfo `json:"vms"`
}

// SNAVMInfo represents VM info from SNA discovery
type SNAVMInfo struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	PowerState string        `json:"power_state"`
	GuestOS    string        `json:"guest_os"`
	MemoryMB   int           `json:"memory_mb"`
	NumCPU     int           `json:"num_cpu"`
	VMXVersion string        `json:"vmx_version,omitempty"`
	Disks      []SNADiskInfo `json:"disks"`
	Networks   []SNANetworkInfo `json:"networks"`
}

// SNADiskInfo represents disk information from SNA
type SNADiskInfo struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Path          string `json:"path"`
	SizeGB        int    `json:"size_gb"`
	CapacityBytes int64  `json:"capacity_bytes"`
	Datastore     string `json:"datastore"`
}

// SNANetworkInfo represents network information from SNA
type SNANetworkInfo struct {
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

// DiscoverVMsFromVMA calls SNA discovery endpoint and returns VM information
func (eds *EnhancedDiscoveryService) DiscoverVMsFromVMA(ctx context.Context, request DiscoveryRequest) (*SNADiscoveryResponse, error) {
	log := eds.tracker.Logger(ctx)

	log.Info("Starting VM discovery from SNA",
		"vcenter", request.VCenter,
		"datacenter", request.Datacenter,
		"filter", request.Filter)

	// Prepare request to SNA
	snaURL := fmt.Sprintf("%s/api/v1/discover", eds.snaBaseURL)

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal discovery request: %w", err)
	}

	// Make HTTP request to SNA
	httpReq, err := http.NewRequestWithContext(ctx, "POST", snaURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call SNA discovery endpoint: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read SNA response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SNA discovery failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SNA response
	var snaResponse SNADiscoveryResponse
	if err := json.Unmarshal(body, &snaResponse); err != nil {
		return nil, fmt.Errorf("failed to parse SNA discovery response: %w", err)
	}

	log.Info("Successfully discovered VMs from SNA",
		"vm_count", len(snaResponse.VMs),
		"vcenter", snaResponse.VCenter.Host,
		"datacenter", snaResponse.VCenter.Datacenter)

	return &snaResponse, nil
}

// AddVMsToOMAWithoutJobs adds discovered VMs to SHA as VM contexts without creating replication jobs
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

	// Step 1: Discover VMs from SNA
	err = eds.tracker.RunStep(ctx, jobID, "vma-discovery", func(ctx context.Context) error {
		log := eds.tracker.Logger(ctx)
		start := time.Now()

		snaResponse, err := eds.DiscoverVMsFromVMA(ctx, discoveryRequest)
		if err != nil {
			return fmt.Errorf("SNA discovery failed: %w", err)
		}

		result.DiscoveryDuration = time.Since(start)
		log.Info("SNA discovery completed",
			"duration", result.DiscoveryDuration,
			"discovered_vms", len(snaResponse.VMs))

		// Step 2: Process discovered VMs
		start = time.Now()
		return eds.processDiscoveredVMs(ctx, snaResponse, selectedVMNames, result)
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

// processDiscoveredVMs processes VMs from SNA discovery and creates VM contexts
func (eds *EnhancedDiscoveryService) processDiscoveredVMs(ctx context.Context, snaResponse *SNADiscoveryResponse,
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
	for _, vm := range snaResponse.VMs {
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
			Host:       snaResponse.VCenter.Host,
			Datacenter: snaResponse.VCenter.Datacenter,
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
func (eds *EnhancedDiscoveryService) createVMContext(ctx context.Context, vm SNAVMInfo,
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
		"auto_added", true,
		"disk_count", len(vm.Disks))

	// 🆕 NEW: Create vm_disks records from discovery (without job_id)
	// This allows backup workflow to access disk information without creating replication job
	if err := eds.createVMDisksFromDiscovery(ctx, contextID, vm.Disks); err != nil {
		// Log error but don't fail - disks can be populated later during replication
		log.Error("Failed to create VM disk records from discovery",
			"error", err,
			"context_id", contextID)
	} else {
		log.Info("Created VM disk records from discovery",
			"context_id", contextID,
			"disk_count", len(vm.Disks))
	}

	return contextID, nil
}

// createVMDisksFromDiscovery creates vm_disks records from SNA discovery information
// This populates disk metadata WITHOUT requiring a replication job, enabling backup workflow
func (eds *EnhancedDiscoveryService) createVMDisksFromDiscovery(
	ctx context.Context, contextID string, disks []SNADiskInfo) error {

	log := eds.tracker.Logger(ctx)

	// Get vmDiskRepo from database connection
	vmDiskRepo := database.NewVMDiskRepository(eds.db)

	for _, disk := range disks {
		vmDisk := &database.VMDisk{
			VMContextID:   contextID,
			JobID:         nil, // NULL - no replication job yet (will be set when replication starts)
			DiskID:        disk.ID,
			VMDKPath:      disk.Path,
			SizeGB:        disk.SizeGB, // Already an int
			CapacityBytes: disk.CapacityBytes,
			Datastore:     disk.Datastore,
			Label:         disk.Label,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := vmDiskRepo.Create(vmDisk); err != nil {
			return fmt.Errorf("failed to create disk record for disk %s: %w", disk.ID, err)
		}

		log.Info("Created VM disk record from discovery",
			"context_id", contextID,
			"disk_id", disk.ID,
			"size_gb", disk.SizeGB,
			"capacity_bytes", disk.CapacityBytes,
			"datastore", disk.Datastore)
	}

	return nil
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

	// Query VM contexts that are not in any group
	var contexts []database.VMReplicationContext
	err := eds.db.GetGormDB().Raw(`
		SELECT vrc.* FROM vm_replication_contexts vrc
		WHERE vrc.context_id NOT IN (
			SELECT DISTINCT vm_context_id FROM vm_group_memberships
		)
	`).Scan(&contexts).Error
	
	return contexts, err
}
