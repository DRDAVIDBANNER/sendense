// Package failover provides helper utilities for enhanced test failover
package failover

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/vexxhost/migratekit-oma/common/logging"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
	"github.com/vexxhost/migratekit-oma/ossea"
	"github.com/vexxhost/migratekit-oma/services"
)

// FailoverHelpers provides utility functions for enhanced test failover
type FailoverHelpers struct {
	db              *database.Connection
	osseaClient     *ossea.Client
	jobTracker      *joblog.Tracker
	failoverJobRepo *database.FailoverJobRepository
}

// NewFailoverHelpers creates a new failover helpers instance
func NewFailoverHelpers(
	db *database.Connection,
	osseaClient *ossea.Client,
	jobTracker *joblog.Tracker,
	failoverJobRepo *database.FailoverJobRepository,
) *FailoverHelpers {
	return &FailoverHelpers{
		db:              db,
		osseaClient:     osseaClient,
		jobTracker:      jobTracker,
		failoverJobRepo: failoverJobRepo,
	}
}

// GatherVMSpecifications retrieves VM specifications from the database for test VM creation
func (fh *FailoverHelpers) GatherVMSpecifications(ctx context.Context, vmID string) (*VMSpecification, error) {
	logger := fh.jobTracker.Logger(ctx)
	logger.Info("🔍 Gathering VM specifications for enhanced test failover", "vm_id", vmID)

	// Use the same method as original failover: vmInfoService.GetVMDetails()
	// This properly queries vm_disks table which has the actual VM specs

	// Get VM info service based on configuration
	vmInfoService, err := fh.getVMInfoService()
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info service: %w", err)
	}

	// Cast to the specific type to access GetVMDetails method (with fallback support)
	if dbService, ok := vmInfoService.(interface {
		GetVMDetails(string) (map[string]interface{}, error)
	}); ok {
		details, err := dbService.GetVMDetails(vmID)
		if err != nil {
			return nil, fmt.Errorf("failed to get VM details from database: %w", err)
		}

		// Build VM specification with unique test naming
		timestamp := time.Now().Unix()
		vmSpec := &VMSpecification{
			Name:        fmt.Sprintf("%s-test-%d", details["name"].(string), timestamp),
			DisplayName: details["display_name"].(string) + " (Enhanced Test Failover)",
			CPUs:        details["cpu_count"].(int),
			MemoryMB:    details["memory_mb"].(int),
			OSType:      "", // Template defines OS type
		}

		// 🎯 MULTI-DISK ENHANCEMENT: Gather all disk specifications
		diskSpecs, err := fh.gatherDiskSpecifications(ctx, vmID)
		if err != nil {
			logger.Warn("⚠️ Failed to gather disk specifications - using single-disk fallback", "error", err)
			// Fallback to single-disk behavior for backward compatibility
			vmSpec.RootDiskSizeGB = 0 // Will be determined by volume query
		} else {
			vmSpec.Disks = diskSpecs
			vmSpec.TotalDiskCount = len(diskSpecs)

			// Set root disk size from OS disk for backward compatibility
			for _, disk := range diskSpecs {
				if disk.IsRoot {
					vmSpec.RootDiskSizeGB = disk.SizeGB
					break
				}
			}

			logger.Info("✅ Multi-disk VM specifications gathered",
				"vm_id", vmID,
				"total_disks", vmSpec.TotalDiskCount,
				"root_disk_size_gb", vmSpec.RootDiskSizeGB)
		}

		if vmSpec.Name == "" || vmSpec.CPUs == 0 || vmSpec.MemoryMB == 0 {
			logger.Warn("⚠️ Incomplete VM specifications found",
				"vm_id", vmID,
				"name", vmSpec.Name,
				"cpus", vmSpec.CPUs,
				"memory_mb", vmSpec.MemoryMB,
			)
		} else {
			logger.Info("✅ VM specifications gathered from database for enhanced test failover",
				"vm_id", vmID,
				"vm_name", vmSpec.Name,
				"cpu_count", vmSpec.CPUs,
				"memory_mb", vmSpec.MemoryMB,
			)
		}

		return vmSpec, nil
	}

	// Fallback error if service type is not supported
	return nil, fmt.Errorf("unsupported VM info service type for enhanced test failover")
}

// gatherDiskSpecifications retrieves all disk specifications for multi-disk VM support
func (fh *FailoverHelpers) gatherDiskSpecifications(ctx context.Context, vmID string) ([]DiskSpecification, error) {
	logger := fh.jobTracker.Logger(ctx)
	logger.Info("🔍 Gathering disk specifications for multi-disk failover", "vm_id", vmID)

	// Query vm_disks using VM context for stable records
	// First, find the VM context ID from VMware UUID
	var vmContext database.VMReplicationContext
	err := (*fh.db).GetGormDB().Where("vmware_vm_id = ?", vmID).First(&vmContext).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find VM context for vmware_vm_id %s: %w", vmID, err)
	}

	// Get all vm_disks for this VM context (using our stable vm_disks architecture)
	var vmDisks []database.VMDisk
	err = (*fh.db).GetGormDB().Where("vm_context_id = ?", vmContext.ContextID).Find(&vmDisks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get vm_disks for context %s: %w", vmContext.ContextID, err)
	}

	if len(vmDisks) == 0 {
		return nil, fmt.Errorf("no vm_disks found for VM context %s", vmContext.ContextID)
	}

	// Build disk specifications for each disk
	var diskSpecs []DiskSpecification
	for _, vmDisk := range vmDisks {
		// Get OSSEA volume information
		var osseaVolume database.OSSEAVolume
		err := (*fh.db).GetGormDB().Where("id = ?", vmDisk.OSSEAVolumeID).First(&osseaVolume).Error
		if err != nil {
			logger.Error("Failed to get OSSEA volume for vm_disk", "error", err, "vm_disk_id", vmDisk.ID)
			continue // Skip disks without valid volumes
		}

		// Get device path from device mappings (direct query to avoid model issues)
		var devicePath string
		err = (*fh.db).GetGormDB().Raw("SELECT device_path FROM device_mappings WHERE volume_uuid = ? LIMIT 1", osseaVolume.VolumeID).Scan(&devicePath).Error
		if err != nil || devicePath == "" {
			devicePath = "unknown" // Fallback if device mapping not found
		}

		diskSpec := DiskSpecification{
			DiskID:     vmDisk.DiskID,
			VolumeID:   osseaVolume.VolumeID,
			SizeGB:     vmDisk.SizeGB,
			IsRoot:     vmDisk.DiskID == "disk-2000",
			DevicePath: devicePath,
		}

		diskSpecs = append(diskSpecs, diskSpec)

		logger.Info("📀 Gathered disk specification",
			"disk_id", diskSpec.DiskID,
			"volume_id", diskSpec.VolumeID,
			"size_gb", diskSpec.SizeGB,
			"is_root", diskSpec.IsRoot,
			"device_path", diskSpec.DevicePath)
	}

	logger.Info("✅ All disk specifications gathered",
		"vm_id", vmID,
		"context_id", vmContext.ContextID,
		"disk_count", len(diskSpecs))

	return diskSpecs, nil
}

// GetOSSEAConfig retrieves OSSEA configuration from database
func (fh *FailoverHelpers) GetOSSEAConfig() (*database.OSSEAConfig, error) {
	// Query database for OSSEA configuration
	var config database.OSSEAConfig
	err := (*fh.db).GetGormDB().First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get OSSEA configuration: %w", err)
	}
	return &config, nil
}

// ResolveZoneID resolves zone name to ID
func (fh *FailoverHelpers) ResolveZoneID(ctx context.Context, zoneName string) (string, error) {
	logger := fh.jobTracker.Logger(ctx)
	logger.Info("🔍 Resolving zone name to ID", "zone_name", zoneName)

	zones, err := fh.osseaClient.ListZones()
	if err != nil {
		return "", fmt.Errorf("failed to list zones: %w", err)
	}

	for _, zone := range zones {
		if zone.Name == zoneName {
			logger.Info("✅ Resolved zone name to ID",
				"zone_name", zoneName,
				"zone_id", zone.ID,
			)
			return zone.ID, nil
		}
	}

	return "", fmt.Errorf("zone not found: %s", zoneName)
}

// CalculateDiskSize calculates appropriate disk size for VM (using working archive logic)
func (fh *FailoverHelpers) CalculateDiskSize(vmSpec *VMSpecification) int {
	if vmSpec.RootDiskSizeGB > 0 {
		return vmSpec.RootDiskSizeGB
	}
	// Default minimum size for test VMs (from working archive)
	return 20
}

// SanitizeVMName sanitizes VM name for CloudStack compatibility
func (fh *FailoverHelpers) SanitizeVMName(originalName string) string {
	// CloudStack VM name requirements:
	// - Only lowercase letters, numbers, and hyphens
	// - 1-63 characters
	// - No leading/trailing hyphens
	// - No leading digits

	// Convert to lowercase
	name := strings.ToLower(originalName)

	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	name = reg.ReplaceAllString(name, "-")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	name = reg.ReplaceAllString(name, "-")

	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")

	// Ensure it doesn't start with a digit
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "vm-" + name
	}

	// Truncate to 63 characters if needed
	if len(name) > 63 {
		name = name[:63]
		// Remove trailing hyphen if truncation created one
		name = strings.TrimRight(name, "-")
	}

	// Ensure minimum length
	if len(name) == 0 {
		name = "test-vm"
	}

	return name
}

// GetOMAVMID gets the OMA VM ID from system configuration
func (fh *FailoverHelpers) GetOMAVMID(ctx context.Context) (string, error) {
	logger := fh.jobTracker.Logger(ctx)

	// Get OMA VM ID from OSSEA configuration
	config, err := fh.GetOSSEAConfig()
	if err != nil {
		logger.Error("Failed to get OSSEA config for OMA VM ID", "error", err)
		return "", fmt.Errorf("failed to get OSSEA configuration: %w", err)
	}

	if config.OMAVMID == "" {
		logger.Error("OMA VM ID not configured in OSSEA config")
		return "", fmt.Errorf("OMA VM ID not configured in OSSEA config")
	}

	logger.Info("Retrieved OMA VM ID from configuration", "oma_vm_id", config.OMAVMID)
	return config.OMAVMID, nil
}

// CreateTestFailoverJob creates a failover job record in the database
func (fh *FailoverHelpers) CreateTestFailoverJob(ctx context.Context, request *EnhancedTestFailoverRequest) error {
	logger := logging.NewOperationLogger(request.FailoverJobID)
	opCtx := logger.StartOperation("create-failover-job", request.FailoverJobID)
	defer opCtx.EndOperation("completed", map[string]interface{}{
		"vm_id":           request.VMID,
		"failover_job_id": request.FailoverJobID,
	})

	// Get VM specifications for the job record
	vmSpec, err := fh.GatherVMSpecifications(ctx, request.VMID)
	if err != nil {
		return fmt.Errorf("failed to gather VM specifications: %w", err)
	}

	// Marshal VM specifications to JSON
	vmSpecJSON, err := json.Marshal(vmSpec)
	if err != nil {
		opCtx.LogError("marshal-vm-spec", "Failed to marshal VM specification", err, map[string]interface{}{})
		return fmt.Errorf("failed to marshal VM specification: %w", err)
	}

	// Create failover job record
	failoverJob := &database.FailoverJob{
		JobID:            request.FailoverJobID,
		VMID:             request.VMID,
		ReplicationJobID: request.VMID, // Use VM ID as replication job reference
		JobType:          "test",
		Status:           "pending",
		SourceVMName:     request.VMName, // Add the missing source VM name
		SourceVMSpec:     string(vmSpecJSON),
		CreatedAt:        request.Timestamp,
		UpdatedAt:        request.Timestamp,
	}

	err = fh.failoverJobRepo.Create(failoverJob)
	if err != nil {
		opCtx.LogError("create-failover-job", "Failed to create failover job record", err, map[string]interface{}{
			"job_id": request.FailoverJobID,
		})
		return fmt.Errorf("failed to create failover job record: %w", err)
	}

	opCtx.LogSuccess("create-failover-job", "Successfully created failover_jobs record", map[string]interface{}{
		"job_id":  request.FailoverJobID,
		"vm_id":   request.VMID,
		"vm_name": request.VMName,
	})

	return nil
}

// InitializeOSSEAClient initializes a fresh OSSEA client from database configuration
// This method fetches credentials from the database on every call, ensuring
// that credential updates are immediately available without service restart.
// Pattern: Follows cleanup_helpers.go:InitializeOSSEAClient()
func (fh *FailoverHelpers) InitializeOSSEAClient(ctx context.Context) (*ossea.Client, error) {
	logger := fh.jobTracker.Logger(ctx)
	logger.Info("🔧 Initializing OSSEA client from database configuration")

	// Query active OSSEA configuration from database
	var config database.OSSEAConfig
	err := (*fh.db).GetGormDB().Where("is_active = ?", true).First(&config).Error
	if err != nil {
		logger.Error("❌ Failed to get active OSSEA configuration from database", "error", err.Error())
		return nil, fmt.Errorf("failed to get active OSSEA config: %w", err)
	}

	// Create fresh OSSEA client with database credentials
	client := ossea.NewClient(
		config.APIURL,
		config.APIKey,
		config.SecretKey,
		config.Domain,
		config.Zone,
	)

	logger.Info("✅ OSSEA client initialized successfully",
		"config_name", config.Name,
		"api_url", config.APIURL,
		"zone", config.Zone,
		"credential_source", "database")

	return client, nil
}

// getVMInfoService returns the appropriate VM info service
func (fh *FailoverHelpers) getVMInfoService() (interface{}, error) {
	// Return the database-based VM info service
	return services.NewSimpleDatabaseVMInfoService(*fh.db), nil
}
