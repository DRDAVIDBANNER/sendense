// Package failover provides VM operations for enhanced test failover
package failover

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
	"github.com/vexxhost/migratekit-oma/ossea"
	"github.com/vexxhost/migratekit-oma/services"
)

// VMOperations handles all VM-related operations for test failover
type VMOperations struct {
	jobTracker *joblog.Tracker
	db         *database.Connection
	helpers    *FailoverHelpers
}

// NewVMOperations creates a new VM operations handler
// Note: No longer requires pre-initialized osseaClient - credentials fetched fresh per operation
func NewVMOperations(osseaClient *ossea.Client, jobTracker *joblog.Tracker, db *database.Connection) *VMOperations {
	// Initialize helpers for credential management
	helpers := &FailoverHelpers{
		db:         db,
		jobTracker: jobTracker,
		// osseaClient is NOT cached - will be initialized fresh per operation
	}

	return &VMOperations{
		jobTracker: jobTracker,
		db:         db,
		helpers:    helpers,
	}
}

// VMSpecification represents VM configuration for test failover
type VMSpecification struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	CPUs           int    `json:"cpus"`
	MemoryMB       int    `json:"memory_mb"`
	OSType         string `json:"os_type"`
	RootDiskSizeGB int    `json:"root_disk_size_gb"`
	// 🎯 MULTI-DISK ENHANCEMENT: Support for multiple disks
	Disks          []DiskSpecification `json:"disks,omitempty"`  // All VM disks (OS + data)
	TotalDiskCount int                 `json:"total_disk_count"` // Total number of disks
}

// DiskSpecification represents individual disk configuration for multi-disk failover
type DiskSpecification struct {
	DiskID     string `json:"disk_id"`     // "disk-2000", "disk-2001", etc.
	VolumeID   string `json:"volume_id"`   // OSSEA volume UUID
	SizeGB     int    `json:"size_gb"`     // Disk size in GB
	IsRoot     bool   `json:"is_root"`     // True for OS disk (disk-2000)
	DevicePath string `json:"device_path"` // OMA device path (/dev/vdb, /dev/vdc)
}

// CreateTestVM creates a test VM in CloudStack with dynamic network configuration
func (vo *VMOperations) CreateTestVM(ctx context.Context, request *EnhancedTestFailoverRequest, networkID string) (string, error) {
	logger := vo.jobTracker.Logger(ctx)
	logger.Info("🖥️ Creating test CloudStack VM", "vm_name", request.VMName)

	// Validate network ID parameter
	if networkID == "" {
		return "", fmt.Errorf("network ID is required for VM creation")
	}

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := vo.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for VM creation", "error", err.Error())
		return "", fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}
	logger.Info("✅ Using fresh OSSEA credentials for VM creation")

	// Get OSSEA configuration for VM creation
	config, err := vo.getOSSEAConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get OSSEA configuration: %w", err)
	}

	// Get VM specifications for the source VM (using real database extraction)
	vmSpec, err := vo.getVMSpecifications(request.VMID)
	if err != nil {
		logger.Error("Failed to get VM specifications", "error", err)
		return "", fmt.Errorf("failed to get VM specifications for %s: %w", request.VMID, err)
	}

	// CRITICAL FIX: Use the calculated destination VM name from the request
	// This respects the unified failover config's GetDestinationVMName() logic:
	// - Live failover: exact name (e.g., "pgtest1")
	// - Test failover: suffixed name (e.g., "pgtest1-test-1234567890")
	testVMName := request.VMName

	// CRITICAL FIX: Use the calculated destination VM name for display name too
	// This ensures both Name and DisplayName follow the same naming pattern
	vmSpec.DisplayName = request.VMName

	// Calculate proper disk size using helper method
	rootDiskSizeGB := vmSpec.RootDiskSizeGB
	if rootDiskSizeGB <= 0 {
		rootDiskSizeGB = 20 // Minimum default from working code
	}

	// Resolve zone ID (copied from working archive logic)
	zoneID := config.Zone
	if len(zoneID) < 36 { // UUID length check - if not a UUID, try to resolve
		// Create helpers instance with fresh client for zone resolution
		zoneHelpers := &FailoverHelpers{
			db:          vo.db,
			osseaClient: osseaClient, // Use fresh client
			jobTracker:  vo.jobTracker,
		}

		resolvedZoneID, err := zoneHelpers.ResolveZoneID(ctx, config.Zone)
		if err != nil {
			logger.Warn("⚠️ Could not resolve zone name to ID, using as-is",
				"zone_name", config.Zone,
				"error", err,
			)
		} else {
			zoneID = resolvedZoneID
			logger.Info("✅ Resolved zone name to ID",
				"zone_name", config.Zone,
				"zone_id", zoneID,
			)
		}
	}

	// Sanitize VM name for CloudStack compatibility
	sanitizedName := vo.helpers.SanitizeVMName(testVMName)

	logger.Info("🚀 Creating test VM with complete OSSEA configuration",
		"vm_name", sanitizedName,
		"cpu_count", vmSpec.CPUs,
		"memory_mb", vmSpec.MemoryMB,
		"root_disk_size_gb", rootDiskSizeGB,
		"template_id", config.TemplateID,
		"service_offering_id", config.ServiceOfferingID,
		"disk_offering_id", config.DiskOfferingID,
		"network_id", networkID,
		"zone_id", zoneID,
		"start_vm", false,
	)

	// Create VM request with complete configuration (using working archive patterns)
	createReq := &ossea.CreateVMRequest{
		Name:              sanitizedName,
		DisplayName:       vmSpec.DisplayName,
		ServiceOfferingID: config.ServiceOfferingID,
		TemplateID:        config.TemplateID,
		ZoneID:            zoneID,
		NetworkID:         networkID, // 🌐 ENHANCED: Use dynamic network selection
		StartVM:           false,
		DiskOfferingID:    config.DiskOfferingID,
		RootDiskSize:      rootDiskSizeGB,
		CPUNumber:         vmSpec.CPUs,
		Memory:            vmSpec.MemoryMB,
	}

	// Create the VM using fresh OSSEA client (with our async polling fixes)
	vm, err := osseaClient.CreateVM(createReq)
	if err != nil {
		return "", fmt.Errorf("failed to create test VM: %w", err)
	}

	logger.Info("✅ Test CloudStack VM created successfully",
		"test_vm_id", vm.ID,
		"vm_name", testVMName,
	)

	return vm.ID, nil
}

// PowerOnTestVM powers on a test VM
func (vo *VMOperations) PowerOnTestVM(ctx context.Context, testVMID string) error {
	logger := vo.jobTracker.Logger(ctx)
	logger.Info("⚡ Powering on test VM", "test_vm_id", testVMID)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := vo.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for VM power-on", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	err = osseaClient.StartVM(testVMID)
	if err != nil {
		return fmt.Errorf("failed to start test VM: %w", err)
	}

	logger.Info("✅ Test VM started successfully", "test_vm_id", testVMID)
	return nil
}

// ValidateTestVM validates test VM configuration and status
func (vo *VMOperations) ValidateTestVM(ctx context.Context, testVMID string, vmSpec *VMSpecification) (map[string]interface{}, error) {
	logger := vo.jobTracker.Logger(ctx)
	logger.Info("🔍 Validating test VM", "test_vm_id", testVMID)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := vo.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for VM validation", "error", err.Error())
		return nil, fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Get VM details from CloudStack
	vm, err := osseaClient.GetVM(testVMID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM details: %w", err)
	}

	results := map[string]interface{}{
		"vm_id":   vm.ID,
		"vm_name": vm.Name,
		"state":   vm.State,
		"zone":    vm.ZoneID,
	}

	// Validate VM is running
	if vm.State != "Running" {
		return results, fmt.Errorf("VM not in running state, current state: %s", vm.State)
	}

	// If we have VM specs, validate configuration matches
	if vmSpec != nil {
		results["cpu_validation"] = fmt.Sprintf("Expected: %d, Actual: %d", vmSpec.CPUs, vm.CPUNumber)
		results["memory_validation"] = fmt.Sprintf("Expected: %d MB, Actual: %d MB", vmSpec.MemoryMB, vm.Memory)

		// Validate CPU and memory if available
		if vm.CPUNumber != vmSpec.CPUs {
			logger.Warn("⚠️ CPU count mismatch",
				"expected_cpu", vmSpec.CPUs,
				"actual_cpu", vm.CPUNumber,
			)
		}

		if vm.Memory != vmSpec.MemoryMB {
			logger.Warn("⚠️ Memory size mismatch",
				"expected_memory", vmSpec.MemoryMB,
				"actual_memory", vm.Memory,
			)
		}
	}

	// Check for network connectivity (basic validation)
	if vm.IPAddress != "" {
		results["network_configured"] = true
		results["primary_ip"] = vm.IPAddress
		results["mac_address"] = vm.MACAddress
		results["network_id"] = vm.NetworkID
	} else {
		results["network_configured"] = false
		logger.Warn("⚠️ No IP address assigned to test VM")
	}

	logger.Info("✅ Test VM validation completed",
		"test_vm_id", testVMID,
		"vm_state", vm.State,
		"cpu_count", vm.CPUNumber,
		"memory_mb", vm.Memory,
		"ip_address", vm.IPAddress,
	)

	return results, nil
}

// getOSSEAConfig retrieves OSSEA configuration from database
func (vo *VMOperations) getOSSEAConfig() (*database.OSSEAConfig, error) {
	var config database.OSSEAConfig
	err := (*vo.db).GetGormDB().Where("is_active = ?", true).First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get active OSSEA config: %w", err)
	}
	return &config, nil
}

// getVMSpecifications retrieves VM specifications from source VM via database
func (vo *VMOperations) getVMSpecifications(vmID string) (*VMSpecification, error) {
	// Use the database-based VM info service to get actual VM specifications
	vmInfoService := services.NewSimpleDatabaseVMInfoService(*vo.db)

	details, err := vmInfoService.GetVMDetails(vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM details from database: %w", err)
	}

	// Build VM specification with actual values from database
	timestamp := time.Now().Unix()
	vmSpec := &VMSpecification{
		Name:           fmt.Sprintf("%s-test-%d", details["name"].(string), timestamp),
		DisplayName:    details["display_name"].(string) + " (Enhanced Test Failover)",
		CPUs:           details["cpu_count"].(int),
		MemoryMB:       details["memory_mb"].(int),
		OSType:         "",  // Template defines OS type
		RootDiskSizeGB: 102, // Default, will be calculated properly
	}

	// Add disk info if available
	if capacityBytes, ok := details["capacity_bytes"].(int64); ok && capacityBytes > 0 {
		vmSpec.RootDiskSizeGB = int(capacityBytes / (1024 * 1024 * 1024))
	}

	return vmSpec, nil
}
