// Package failover provides unified failover engine for both live and test scenarios
package failover

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/common"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
	"github.com/vexxhost/migratekit-oma/ossea"
	"github.com/vexxhost/migratekit-oma/services"
)

// MultiDiskVolumeInfo represents volume information for multi-disk failover operations
type MultiDiskVolumeInfo struct {
	OSVolume    VolumeDetails   `json:"os_volume"`    // Root volume (disk-2000)
	DataVolumes []VolumeDetails `json:"data_volumes"` // Additional volumes (disk-2001, disk-2002, etc.)
	TotalCount  int             `json:"total_count"`  // Total number of volumes
}

// VolumeDetails represents individual volume information
type VolumeDetails struct {
	VolumeID   string `json:"volume_id"`   // OSSEA volume UUID
	VolumeName string `json:"volume_name"` // Volume display name
	DevicePath string `json:"device_path"` // OMA device path (/dev/vdb, /dev/vdc)
	SizeGB     int    `json:"size_gb"`     // Volume size in GB
	DiskID     string `json:"disk_id"`     // VMware disk ID (disk-2000, disk-2001)
	VMDiskID   int    `json:"vm_disk_id"`  // vm_disks.id for correlation
}

// UnifiedFailoverEngine orchestrates both live and test failover operations using configuration-based differences
// This replaces both EnhancedTestFailoverEngine and EnhancedLiveFailoverEngine with a single, configurable engine
type UnifiedFailoverEngine struct {
	// Core dependencies (shared by both live and test failover)
	db                 database.Connection
	jobTracker         *joblog.Tracker
	failoverJobRepo    *database.FailoverJobRepository
	vmContextRepo      *database.VMReplicationContextRepository
	networkMappingRepo *database.NetworkMappingRepository

	// Modular components (reused from existing engines)
	vmOperations       *VMOperations
	volumeOperations   *VolumeOperations
	virtioInjection    *VirtIOInjection
	snapshotOperations *SnapshotOperations
	validation         *FailoverValidation
	helpers            *FailoverHelpers

	// 🆕 NEW: Multi-volume snapshot service for complete VM protection
	multiVolumeSnapshotService *MultiVolumeSnapshotService

	// Enhanced services
	networkMappingService *services.NetworkMappingService
	networkConfigProvider *NetworkConfigProvider
	volumeClient          *common.VolumeClient
	osseaClient           *ossea.Client
	networkClient         *ossea.NetworkClient
	vmInfoService         services.VMInfoProvider
	validator             *PreFailoverValidator
	vmaClient             VMAClient
}

// UnifiedFailoverResult represents the result of a unified failover operation
type UnifiedFailoverResult struct {
	FailoverJobID   string                 `json:"failover_job_id"`
	DestinationVMID string                 `json:"destination_vm_id"`
	Status          string                 `json:"status"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	Duration        time.Duration          `json:"duration"`
	SnapshotName    string                 `json:"snapshot_name,omitempty"`
	NetworkMappings map[string]string      `json:"network_mappings"`
	VolumeMappings  []VolumeInfo           `json:"volume_mappings"`
	Metadata        map[string]interface{} `json:"metadata"`
	Error           string                 `json:"error,omitempty"`
}

// NewUnifiedFailoverEngine creates a new unified failover engine with all required dependencies
func NewUnifiedFailoverEngine(
	db database.Connection,
	jobTracker *joblog.Tracker,
	osseaClient *ossea.Client,
	networkClient *ossea.NetworkClient,
	vmInfoService services.VMInfoProvider,
	networkMappingService *services.NetworkMappingService,
	volumeClient *common.VolumeClient,
	vmaClient VMAClient,
) *UnifiedFailoverEngine {
	// Initialize repositories
	failoverJobRepo := database.NewFailoverJobRepository(db)
	vmContextRepo := database.NewVMReplicationContextRepository(db)
	networkMappingRepo := database.NewNetworkMappingRepository(db)

	// Initialize modular components (reuse existing implementations)
	// Note: Order matters due to dependencies
	helpers := NewFailoverHelpers(&db, osseaClient, jobTracker, failoverJobRepo)
	vmOperations := NewVMOperations(osseaClient, jobTracker, &db)
	volumeOperations := NewVolumeOperations(jobTracker, &db, osseaClient)
	virtioInjection := NewVirtIOInjection(&db, jobTracker)
	snapshotOperations := NewSnapshotOperations(&db, osseaClient, jobTracker)
	validation := NewFailoverValidation(jobTracker, helpers)
	validator := NewPreFailoverValidator(db, vmInfoService, networkMappingService)

	// 🆕 NEW: Initialize multi-volume snapshot service for complete VM protection
	multiVolumeSnapshotService := NewMultiVolumeSnapshotService(&db, osseaClient, jobTracker)

	// 🌐 ENHANCED: Initialize NetworkConfigProvider with default network fallback
	defaultNetworkID := getDefaultNetworkID(db)
	networkConfigProvider := NewNetworkConfigProvider(networkMappingRepo, defaultNetworkID)

	return &UnifiedFailoverEngine{
		db:                         db,
		jobTracker:                 jobTracker,
		failoverJobRepo:            failoverJobRepo,
		vmContextRepo:              vmContextRepo,
		networkMappingRepo:         networkMappingRepo,
		vmOperations:               vmOperations,
		volumeOperations:           volumeOperations,
		virtioInjection:            virtioInjection,
		snapshotOperations:         snapshotOperations,
		validation:                 validation,
		helpers:                    helpers,
		multiVolumeSnapshotService: multiVolumeSnapshotService, // 🆕 NEW: Multi-volume snapshot support
		networkMappingService:      networkMappingService,
		networkConfigProvider:      networkConfigProvider,
		volumeClient:               volumeClient,
		osseaClient:                osseaClient,
		networkClient:              networkClient,
		vmInfoService:              vmInfoService,
		validator:                  validator,
		vmaClient:                  vmaClient,
	}
}

// getDefaultNetworkID retrieves the default network ID from OSSEA configuration
func getDefaultNetworkID(db database.Connection) string {
	var config database.OSSEAConfig
	err := db.GetGormDB().Where("is_active = ?", true).First(&config).Error
	if err != nil {
		log.WithError(err).Warn("Failed to get default network ID from OSSEA config, using empty fallback")
		return ""
	}
	return config.NetworkID
}

// ExecuteUnifiedFailover performs a unified failover operation based on the provided configuration
// This method replaces both ExecuteEnhancedTestFailover and ExecuteEnhancedFailover methods
func (ufe *UnifiedFailoverEngine) ExecuteUnifiedFailover(ctx context.Context, config *UnifiedFailoverConfig) (*UnifiedFailoverResult, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid failover configuration: %w", err)
	}

	// Generate GUI-compatible external job ID for correlation
	externalJobID := fmt.Sprintf("unified-%s-failover-%s-%d",
		config.FailoverType, config.VMName, time.Now().Unix())

	// Start the unified failover job with enhanced JobLog tracking
	ctx, jobID, err := ufe.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:       "failover",
		Operation:     fmt.Sprintf("unified-%s-failover", config.FailoverType),
		Owner:         stringPtr("system"),
		ContextID:     &config.ContextID,     // Enhanced: Direct VM context correlation
		ExternalJobID: &externalJobID,        // Enhanced: GUI job ID correlation
		JobCategory:   stringPtr("failover"), // Enhanced: High-level categorization
		Metadata: map[string]interface{}{
			"context_id":      config.ContextID, // Backward compatibility
			"vmware_vm_id":    config.VMwareVMID,
			"vm_name":         config.VMName,
			"failover_type":   config.FailoverType,
			"failover_job_id": config.FailoverJobID,
			"external_job_id": externalJobID, // Track external ID in metadata too
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start unified failover job: %w", err)
	}

	startTime := time.Now()
	result := &UnifiedFailoverResult{
		FailoverJobID:   config.FailoverJobID,
		Status:          "running",
		StartTime:       startTime,
		NetworkMappings: make(map[string]string),
		VolumeMappings:  make([]VolumeInfo, 0),
		Metadata:        make(map[string]interface{}),
	}

	// Ensure job completion is tracked with sanitized error messages
	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		if result.Error != "" {
			// Sanitize error for JobLog (keeps technical details but adds user message)
			ufe.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, fmt.Errorf(result.Error))
			
			// Store operation summary for persistent GUI visibility
			ufe.storeOperationSummary(ctx, config, jobID, externalJobID, "failed", result)
		} else {
			ufe.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
			
			// Store success summary
			ufe.storeOperationSummary(ctx, config, jobID, externalJobID, "completed", result)
		}
	}()

	// Execute the 9-phase unified failover workflow
	if err := ufe.executeUnifiedWorkflow(ctx, jobID, config, result); err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	result.Status = "completed"
	return result, nil
}

// executeUnifiedWorkflow executes the 9-phase unified failover workflow
func (ufe *UnifiedFailoverEngine) executeUnifiedWorkflow(ctx context.Context, jobID string, config *UnifiedFailoverConfig, result *UnifiedFailoverResult) error {
	logger := ufe.jobTracker.Logger(ctx)
	logger.Info("🚀 Starting unified failover workflow",
		"failover_type", config.FailoverType,
		"context_id", config.ContextID,
		"vm_name", config.VMName)

	// Phase 1: Validation
	if !config.SkipValidation {
		if err := ufe.executeValidationPhase(ctx, jobID, config); err != nil {
			return fmt.Errorf("validation phase failed: %w", err)
		}
	}

	// CRITICAL FIX: Create failover job record (was missing from unified system)
	// This mirrors the enhanced test failover system exactly
	if err := ufe.createUnifiedFailoverJob(ctx, jobID, config); err != nil {
		return fmt.Errorf("failed to create failover job record: %w", err)
	}

	// Phase 2: Source VM Power Management (live failover only)
	if config.RequiresSourceVMPowerOff() {
		if err := ufe.executeSourceVMPowerOffPhase(ctx, jobID, config); err != nil {
			return fmt.Errorf("source VM power-off phase failed: %w", err)
		}
	}

	// Phase 3: Final Sync (live failover only)
	if config.RequiresFinalSync() {
		if err := ufe.executeFinalSyncPhase(ctx, jobID, config); err != nil {
			return fmt.Errorf("final sync phase failed: %w", err)
		}
	}

	// CRITICAL FIX: Update VM context status AFTER final sync completion
	// This prevents blocking replication API during final sync
	statusValue := "failed_over_test"
	if config.FailoverType == FailoverTypeLive {
		statusValue = "failed_over_live"
	}
	if err := ufe.updateVMContextStatus(ctx, config.ContextID, statusValue); err != nil {
		// Error is logged but doesn't fail the operation (matches enhanced system)
	}

	// Phase 3.5: Switch volumes to 'failover' mode for multi-volume snapshot detection
	// 🆕 CRITICAL: MultiVolumeSnapshotService requires volumes in 'failover' mode
	if err := ufe.executeVolumeModeSwitch(ctx, jobID, config.ContextID, "failover"); err != nil {
		return fmt.Errorf("volume mode switch to failover failed: %w", err)
	}

	// Phase 4: Multi-Volume Snapshot Creation (for complete VM protection)
	// 🆕 ENHANCED: Create snapshots for ALL volumes, not just first disk
	var legacySnapshotID string // For backward compatibility with VirtIO injection
	if config.SnapshotType != SnapshotTypeNone {
		snapshotResult, err := ufe.executeMultiVolumeSnapshotCreationPhase(ctx, jobID, config)
		if err != nil {
			return fmt.Errorf("multi-volume snapshot creation phase failed: %w", err)
		}

		// Set legacy snapshot ID for backward compatibility (use first snapshot)
		if len(snapshotResult.SnapshotsCreated) > 0 {
			legacySnapshotID = snapshotResult.SnapshotsCreated[0].SnapshotID
			result.SnapshotName = legacySnapshotID
		}

		// Store complete snapshot information in metadata
		result.Metadata["multi_volume_snapshots"] = snapshotResult
	}

	// Phase 5: VirtIO Injection (if not skipped) - MUST happen after snapshot
	if !config.SkipVirtIO {
		virtioStatus, err := ufe.executeVirtIOInjectionPhase(ctx, jobID, config, legacySnapshotID)
		if err != nil {
			// 🆕 ENHANCED: Make VirtIO injection non-fatal for live failover
			// Live failover may have OS detection issues after VM shutdown
			logger := ufe.jobTracker.Logger(ctx)
			if config.FailoverType == FailoverTypeLive {
				logger.Warn("⚠️ VirtIO injection failed during live failover - continuing with failover (VM may need manual driver installation)",
					"error", err.Error(),
					"context_id", config.ContextID,
					"failover_type", "live")
				result.Metadata["virtio_status"] = "failed_non_fatal"
				result.Metadata["virtio_error"] = err.Error()
			} else {
				// Test failover should still fail on VirtIO injection errors (VM is running, OS type should be detectable)
				return fmt.Errorf("VirtIO injection phase failed: %w", err)
			}
		} else {
			result.Metadata["virtio_status"] = virtioStatus
		}
	}

	// Phase 6: VM Creation
	destinationVMID, err := ufe.executeVMCreationPhase(ctx, jobID, config)
	if err != nil {
		return fmt.Errorf("VM creation phase failed: %w", err)
	}
	result.DestinationVMID = destinationVMID

	// CRITICAL FIX: Update failover job with destination VM ID (was missing from unified system)
	if err := ufe.updateDestinationVMID(ctx, jobID, destinationVMID); err != nil {
		return fmt.Errorf("failed to update destination VM ID: %w", err)
	}

	// Phase 7: Volume Attachment (matches enhanced_test_failover.go exactly)
	if err := ufe.executeVolumeAttachmentPhase(ctx, jobID, config, destinationVMID); err != nil {
		return fmt.Errorf("volume attachment phase failed: %w", err)
	}

	// Phase 8: VM Startup and Validation (matches enhanced_test_failover.go exactly)
	if err := ufe.executeVMStartupAndValidationPhase(ctx, jobID, config, destinationVMID); err != nil {
		return fmt.Errorf("VM startup and validation phase failed: %w", err)
	}

	// Phase 9: Status Updates
	if err := ufe.executeStatusUpdatePhase(ctx, jobID, config, destinationVMID); err != nil {
		return fmt.Errorf("status update phase failed: %w", err)
	}

	// CRITICAL FIX: Mark failover job as completed (was missing from unified system)
	if err := ufe.markFailoverJobCompleted(ctx, jobID); err != nil {
		return fmt.Errorf("failed to mark failover job as completed: %w", err)
	}

	logger.Info("✅ Unified failover workflow completed successfully",
		"failover_type", config.FailoverType,
		"destination_vm_id", destinationVMID,
		"duration", time.Since(result.StartTime))

	return nil
}

// getVolumeInfoForVM retrieves volume information for a VM (adapted from enhanced test failover)
func (ufe *UnifiedFailoverEngine) getVolumeInfoForVM(ctx context.Context, contextID string) (*VolumeInfo, error) {
	// Query the database chain: replication_jobs -> vm_disks -> ossea_volumes
	volumeInfo, err := ufe.queryVolumeInfoFromDatabase(ctx, contextID)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume info from database: %w", err)
	}
	return volumeInfo, nil
}

// getMultiDiskVolumeInfoForVM retrieves volume information for all disks in a multi-disk VM
func (ufe *UnifiedFailoverEngine) getMultiDiskVolumeInfoForVM(ctx context.Context, contextID string) (*MultiDiskVolumeInfo, error) {
	logger := ufe.jobTracker.Logger(ctx)
	logger.Info("🔍 Getting multi-disk volume information for VM", "context_id", contextID)

	// Get all vm_disks for this VM context (using our stable vm_disks architecture)
	var vmDisks []database.VMDisk
	err := ufe.db.GetGormDB().Where("vm_context_id = ?", contextID).Find(&vmDisks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get vm_disks for context %s: %w", contextID, err)
	}

	if len(vmDisks) == 0 {
		return nil, fmt.Errorf("no vm_disks found for VM context %s", contextID)
	}

	multiDiskInfo := &MultiDiskVolumeInfo{
		TotalCount: len(vmDisks),
	}

	// Process each disk and categorize as OS or data disk
	for _, vmDisk := range vmDisks {
		// Get OSSEA volume information
		var osseaVolume database.OSSEAVolume
		err := ufe.db.GetGormDB().Where("id = ?", vmDisk.OSSEAVolumeID).First(&osseaVolume).Error
		if err != nil {
			logger.Error("Failed to get OSSEA volume for vm_disk", "error", err, "vm_disk_id", vmDisk.ID)
			continue // Skip disks without valid volumes
		}

		// Get device path from device mappings
		var devicePath string
		err = ufe.db.GetGormDB().Raw("SELECT device_path FROM device_mappings WHERE volume_uuid = ? LIMIT 1", osseaVolume.VolumeID).Scan(&devicePath).Error
		if err != nil || devicePath == "" {
			devicePath = "unknown" // Fallback if device mapping not found
		}

		volumeDetails := VolumeDetails{
			VolumeID:   osseaVolume.VolumeID,
			VolumeName: osseaVolume.VolumeName,
			DevicePath: devicePath,
			SizeGB:     vmDisk.SizeGB,
			DiskID:     vmDisk.DiskID,
			VMDiskID:   vmDisk.ID,
		}

		// Categorize as OS or data disk
		if vmDisk.DiskID == "disk-2000" {
			multiDiskInfo.OSVolume = volumeDetails
			logger.Info("📀 Found OS volume",
				"disk_id", volumeDetails.DiskID,
				"volume_id", volumeDetails.VolumeID,
				"size_gb", volumeDetails.SizeGB,
				"device_path", volumeDetails.DevicePath)
		} else {
			multiDiskInfo.DataVolumes = append(multiDiskInfo.DataVolumes, volumeDetails)
			logger.Info("📀 Found data volume",
				"disk_id", volumeDetails.DiskID,
				"volume_id", volumeDetails.VolumeID,
				"size_gb", volumeDetails.SizeGB,
				"device_path", volumeDetails.DevicePath)
		}
	}

	// Validate that we found an OS volume
	if multiDiskInfo.OSVolume.VolumeID == "" {
		return nil, fmt.Errorf("no OS volume (disk-2000) found for VM context %s", contextID)
	}

	logger.Info("✅ Multi-disk volume information gathered",
		"context_id", contextID,
		"os_volume", multiDiskInfo.OSVolume.VolumeID,
		"data_volumes", len(multiDiskInfo.DataVolumes),
		"total_count", multiDiskInfo.TotalCount)

	return multiDiskInfo, nil
}

// createUnifiedFailoverJob creates a failover job record in the database (mirrors enhanced system)
// This is the critical missing component that was causing database inconsistencies
func (ufe *UnifiedFailoverEngine) createUnifiedFailoverJob(ctx context.Context, jobID string, config *UnifiedFailoverConfig) error {
	logger := ufe.jobTracker.Logger(ctx)
	logger.Info("📝 Creating unified failover job record",
		"job_id", jobID,
		"context_id", config.ContextID,
		"vm_name", config.VMName,
		"failover_type", config.FailoverType)

	// Get VM specifications for the job record (reuse enhanced system logic)
	vmSpec, err := ufe.helpers.GatherVMSpecifications(ctx, config.VMwareVMID)
	if err != nil {
		return fmt.Errorf("failed to gather VM specifications: %w", err)
	}

	// Marshal VM specifications to JSON
	vmSpecJSON, err := json.Marshal(vmSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal VM specification: %w", err)
	}

	// 🎯 Get the actual replication job ID for proper correlation
	var replicationJob database.ReplicationJob
	err = ufe.db.GetGormDB().Where("vm_context_id = ?", config.ContextID).
		Order("created_at DESC").First(&replicationJob).Error
	if err != nil {
		logger.Error("Failed to find replication job for failover job record", "error", err, "context_id", config.ContextID)
		return fmt.Errorf("failed to find replication job for context %s: %w", config.ContextID, err)
	}

	// Create failover job record with CRITICAL FIX: populate VMContextID
	failoverJob := &database.FailoverJob{
		VMContextID:      config.ContextID,            // CRITICAL FIX: Was missing in both systems
		JobID:            jobID,                       // JobLog UUID for correlation
		VMID:             config.VMwareVMID,           // VMware UUID (not VM name)
		ReplicationJobID: replicationJob.ID,           // 🎯 FIXED: Use actual replication job ID
		JobType:          string(config.FailoverType), // "live" or "test"
		Status:           "pending",
		SourceVMName:     config.VMName, // VM display name
		SourceVMSpec:     string(vmSpecJSON),
		CreatedAt:        config.Timestamp,
		UpdatedAt:        config.Timestamp,
	}

	err = ufe.failoverJobRepo.Create(failoverJob)
	if err != nil {
		logger.Error("Failed to create unified failover job record", "error", err, "job_id", jobID)
		return fmt.Errorf("failed to create unified failover job record: %w", err)
	}

	logger.Info("✅ Unified failover job record created successfully",
		"job_id", jobID,
		"vm_context_id", config.ContextID,
		"vm_id", config.VMwareVMID,
		"job_type", config.FailoverType)

	return nil
}

// updateVMContextStatus updates the VM context status (mirrors enhanced system)
// This is the second critical missing component
func (ufe *UnifiedFailoverEngine) updateVMContextStatus(ctx context.Context, contextID string, status string) error {
	logger := ufe.jobTracker.Logger(ctx)
	logger.Info("🔄 Updating VM context status",
		"context_id", contextID,
		"new_status", status)

	if contextID == "" {
		logger.Warn("Empty context_id provided - skipping VM context status update")
		return nil
	}

	if err := ufe.vmContextRepo.UpdateVMContextStatus(contextID, status); err != nil {
		// Log error but don't fail the operation (matches enhanced system behavior)
		logger.Error("Failed to update VM context status", "error", err, "context_id", contextID, "status", status)
		return nil // Don't fail the operation
	}

	logger.Info("✅ VM context status updated successfully",
		"context_id", contextID,
		"status", status)

	return nil
}

// updateDestinationVMID updates the failover job with the destination VM ID (mirrors enhanced system)
func (ufe *UnifiedFailoverEngine) updateDestinationVMID(ctx context.Context, jobID string, destinationVMID string) error {
	logger := ufe.jobTracker.Logger(ctx)
	logger.Info("🔄 Updating failover job with destination VM ID",
		"job_id", jobID,
		"destination_vm_id", destinationVMID)

	if err := ufe.failoverJobRepo.UpdateDestinationVM(jobID, destinationVMID); err != nil {
		logger.Error("Failed to update destination VM ID", "error", err, "job_id", jobID)
		return fmt.Errorf("failed to update destination VM ID: %w", err)
	}

	logger.Info("✅ Destination VM ID updated successfully",
		"job_id", jobID,
		"destination_vm_id", destinationVMID)

	return nil
}

// markFailoverJobCompleted marks the failover job as completed (mirrors enhanced system)
func (ufe *UnifiedFailoverEngine) markFailoverJobCompleted(ctx context.Context, jobID string) error {
	logger := ufe.jobTracker.Logger(ctx)
	logger.Info("✅ Marking failover job as completed", "job_id", jobID)

	if err := ufe.failoverJobRepo.MarkCompleted(jobID); err != nil {
		logger.Error("Failed to mark failover job as completed", "error", err, "job_id", jobID)
		return fmt.Errorf("failed to mark failover job as completed: %w", err)
	}

	logger.Info("✅ Failover job marked as completed successfully", "job_id", jobID)
	return nil
}

// queryVolumeInfoFromDatabase queries the database following the normalized schema
func (ufe *UnifiedFailoverEngine) queryVolumeInfoFromDatabase(ctx context.Context, contextID string) (*VolumeInfo, error) {
	logger := ufe.jobTracker.Logger(ctx)
	logger.Info("🔍 Querying database for volume info", "context_id", contextID)

	// DEBUG: Test database connection first
	logger.Info("🔧 DEBUG: Testing database connection")
	var connectionTest int64
	err := ufe.db.GetGormDB().Raw("SELECT 1 as test").Scan(&connectionTest).Error
	if err != nil {
		logger.Error("❌ DEBUG: Database connection test failed", "error", err)
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	logger.Info("✅ DEBUG: Database connection test passed", "result", connectionTest)

	// DEBUG: Count total replication jobs first
	var totalJobs int64
	err = ufe.db.GetGormDB().Model(&database.ReplicationJob{}).Count(&totalJobs).Error
	if err != nil {
		logger.Error("❌ DEBUG: Failed to count replication jobs", "error", err)
	} else {
		logger.Info("📊 DEBUG: Total replication jobs in database", "count", totalJobs)
	}

	// DEBUG: Check if specific context exists
	var contextCount int64
	err = ufe.db.GetGormDB().Model(&database.ReplicationJob{}).Where("vm_context_id = ?", contextID).Count(&contextCount).Error
	if err != nil {
		logger.Error("❌ DEBUG: Failed to count jobs for context", "error", err, "context_id", contextID)
	} else {
		logger.Info("📊 DEBUG: Jobs found for context", "context_id", contextID, "count", contextCount)
	}

	// Step 1: Find replication job for this VM context
	logger.Info("🔍 DEBUG: Executing GORM query", "query", "vm_context_id = ?", "parameter", contextID)
	var replicationJob database.ReplicationJob
	err = ufe.db.GetGormDB().Where("vm_context_id = ?", contextID).First(&replicationJob).Error
	if err != nil {
		logger.Error("❌ GORM query failed", "error", err, "context_id", contextID)

		// DEBUG: Try raw SQL as fallback
		logger.Info("🔍 DEBUG: Attempting raw SQL fallback")
		var rawJob database.ReplicationJob
		rawErr := ufe.db.GetGormDB().Raw("SELECT * FROM replication_jobs WHERE vm_context_id = ? LIMIT 1", contextID).Scan(&rawJob).Error
		if rawErr != nil {
			logger.Error("❌ DEBUG: Raw SQL also failed", "error", rawErr, "context_id", contextID)
		} else {
			logger.Info("✅ DEBUG: Raw SQL succeeded", "job_id", rawJob.ID, "context_id", contextID)
			logger.Error("🚨 DEBUG: GORM vs Raw SQL inconsistency detected!", "gorm_error", err, "raw_success", true)
		}

		return nil, fmt.Errorf("no replication job found for context %s: %w", contextID, err)
	}

	logger.Info("✅ Found replication job via GORM", "job_id", replicationJob.ID, "context_id", contextID)

	// Step 2: Find VM disks for this VM context (FIXED: Use VM-centric query for stable vm_disks)
	var vmDisks []database.VMDisk
	err = ufe.db.GetGormDB().Where("vm_context_id = ?", contextID).Find(&vmDisks).Error
	if err != nil {
		logger.Error("Failed to find VM disks", "error", err, "vm_context_id", contextID)
		return nil, fmt.Errorf("no VM disks found for VM context %s: %w", contextID, err)
	}

	if len(vmDisks) == 0 {
		return nil, fmt.Errorf("no VM disks found for VM context %s", contextID)
	}

	// 🎯 CRITICAL FIX: Find the OS disk (disk-2000) specifically for volume attachment
	// For multi-disk VMs, the OS disk MUST be attached as device ID 0 (root)
	var osDisk *database.VMDisk
	for _, disk := range vmDisks {
		if disk.DiskID == "disk-2000" {
			osDisk = &disk
			logger.Info("🎯 Found OS disk for root volume attachment",
				"disk_id", disk.DiskID,
				"ossea_volume_id", disk.OSSEAVolumeID,
				"vmware_uuid", disk.VMwareUUID)
			break
		}
	}

	if osDisk == nil {
		// Log all available disks for debugging
		logger.Error("❌ OS disk (disk-2000) not found for volume attachment",
			"job_id", replicationJob.ID,
			"available_disks", len(vmDisks))
		for i, disk := range vmDisks {
			logger.Info("Available disk",
				"index", i,
				"disk_id", disk.DiskID,
				"vmware_uuid", disk.VMwareUUID)
		}
		return nil, fmt.Errorf("OS disk (disk-2000) not found for VM context %s - volume attachment requires OS disk as root", contextID)
	}

	vmDisk := *osDisk
	logger.Info("✅ Using OS disk for root volume attachment", "disk_id", vmDisk.DiskID, "ossea_volume_id", vmDisk.OSSEAVolumeID)

	// Step 3: Find OSSEA volume
	var osseaVolume database.OSSEAVolume
	err = ufe.db.GetGormDB().Where("id = ?", vmDisk.OSSEAVolumeID).First(&osseaVolume).Error
	if err != nil {
		logger.Error("Failed to find OSSEA volume", "error", err, "ossea_volume_id", vmDisk.OSSEAVolumeID)
		return nil, fmt.Errorf("no OSSEA volume found for ID %d: %w", vmDisk.OSSEAVolumeID, err)
	}

	logger.Info("Found OSSEA volume",
		"volume_id", osseaVolume.VolumeID,
		"volume_name", osseaVolume.VolumeName,
		"size_gb", osseaVolume.SizeGB,
		"device_path", osseaVolume.DevicePath)

	return &VolumeInfo{
		VolumeID:   osseaVolume.VolumeID,
		VolumeName: osseaVolume.VolumeName,
		DevicePath: osseaVolume.DevicePath,
		SizeGB:     osseaVolume.SizeGB,
	}, nil
}

// attachVolumeToDestinationVMAsRoot attaches a volume to the destination VM as root disk
func (ufe *UnifiedFailoverEngine) attachVolumeToDestinationVMAsRoot(ctx context.Context, volumeID, destinationVMID string) error {
	logger := ufe.jobTracker.Logger(ctx)
	logger.Info("🔗 Attaching volume to destination VM as root disk via Volume Daemon",
		"volume_id", volumeID,
		"destination_vm_id", destinationVMID,
	)

	// Use Volume Daemon for root volume attachment
	operation, err := ufe.volumeClient.AttachVolumeAsRoot(ctx, volumeID, destinationVMID)
	if err != nil {
		return fmt.Errorf("failed to start volume attachment via daemon: %w", err)
	}

	logger.Info("⏳ Waiting for volume attachment completion via Volume Daemon",
		"operation_id", operation.ID,
		"volume_id", volumeID,
		"destination_vm_id", destinationVMID,
	)

	// Wait for completion with device correlation
	finalOp, err := ufe.volumeClient.WaitForCompletionWithTimeout(ctx, operation.ID, 300*time.Second)
	if err != nil {
		logger.Error("Volume attachment operation failed", "error", err, "operation_id", operation.ID)
		return fmt.Errorf("volume attachment operation failed: %w", err)
	}

	devicePath := finalOp.Response["device_path"]
	logger.Info("✅ Volume attached to destination VM as root disk successfully",
		"volume_id", volumeID,
		"destination_vm_id", destinationVMID,
		"device_path", devicePath,
	)

	return nil
}

// attachVolumeToDestinationVM attaches a data volume to the destination VM (non-root)
func (ufe *UnifiedFailoverEngine) attachVolumeToDestinationVM(ctx context.Context, volumeID, destinationVMID string) error {
	logger := ufe.jobTracker.Logger(ctx)
	logger.Info("🔗 Attaching data volume to destination VM via Volume Daemon",
		"volume_id", volumeID,
		"destination_vm_id", destinationVMID,
	)

	// Use Volume Daemon for data volume attachment (auto device ID assignment)
	operation, err := ufe.volumeClient.AttachVolume(ctx, volumeID, destinationVMID)
	if err != nil {
		return fmt.Errorf("failed to start data volume attachment via daemon: %w", err)
	}

	logger.Info("⏳ Waiting for data volume attachment completion via Volume Daemon",
		"operation_id", operation.ID,
		"volume_id", volumeID,
		"destination_vm_id", destinationVMID,
	)

	// Wait for completion with device correlation
	finalOp, err := ufe.volumeClient.WaitForCompletionWithTimeout(ctx, operation.ID, 300*time.Second)
	if err != nil {
		logger.Error("Data volume attachment operation failed", "error", err, "operation_id", operation.ID)
		return fmt.Errorf("data volume attachment operation failed: %w", err)
	}

	devicePath := finalOp.Response["device_path"]
	logger.Info("✅ Data volume attached to destination VM successfully",
		"volume_id", volumeID,
		"destination_vm_id", destinationVMID,
		"device_path", devicePath,
	)

	return nil
}

// executeValidationPhase performs pre-failover validation
func (ufe *UnifiedFailoverEngine) executeValidationPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig) error {
	return ufe.jobTracker.RunStep(ctx, jobID, "validation", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("🔍 Executing validation phase", "context_id", config.ContextID)

		// Use existing validation component
		_, err := ufe.validator.ValidateFailoverReadiness(config.VMName, string(config.FailoverType))
		return err
	})
}

// executeSourceVMPowerOffPhase powers off the source VM for live failover
func (ufe *UnifiedFailoverEngine) executeSourceVMPowerOffPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig) error {
	return ufe.jobTracker.RunStep(ctx, jobID, "source-vm-power-off", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("🔌 Executing source VM power-off phase", "vmware_vm_id", config.VMwareVMID)

		if ufe.vmaClient == nil {
			return fmt.Errorf("VMA client not available for power management")
		}

		// 🔧 CREDENTIAL FIX: Get vCenter credentials from secure credential service (NO FALLBACKS)
		// Initialize credential service
		encryptionService, err := services.NewCredentialEncryptionService()
		if err != nil {
			logger.Error("❌ Failed to initialize VMware credential encryption service", "error", err.Error())
			return fmt.Errorf("failed to initialize VMware credential encryption service: %w. Please ensure encryption service is configured", err)
		}

		credentialService := services.NewVMwareCredentialService(&ufe.db, encryptionService)
		creds, err := credentialService.GetDefaultCredentials(ctx)
		if err != nil {
			logger.Error("❌ Failed to retrieve VMware credentials from database", "error", err.Error())
			return fmt.Errorf("failed to retrieve VMware credentials from database: %w. "+
				"Please ensure VMware credentials are configured in GUI (Settings → VMware Credentials)", err)
		}

		// Use service-managed credentials (NO fallback to hardcoded values)
		vcenterHost := creds.VCenterHost
		vcenterUsername := creds.Username
		vcenterPassword := creds.Password

		logger.Info("✅ Using fresh VMware credentials from database",
			"vcenter_host", vcenterHost,
			"username", vcenterUsername,
			"credential_source", "database")

		// Check current power state before attempting power-off
		powerState, err := ufe.vmaClient.GetVMPowerState(ctx, config.VMwareVMID, vcenterHost, vcenterUsername, vcenterPassword)
		if err != nil {
			logger.Warn("Failed to get VM power state, proceeding with power-off", "error", err)
		} else {
			logger.Info("Current VM power state", "power_state", powerState)
			if powerState == "poweredOff" {
				logger.Info("✅ VM already powered off, skipping power-off phase")
				return nil
			}
		}

		// Execute graceful power-off
		logger.Info("🔄 Initiating graceful source VM power-off")
		if err := ufe.vmaClient.PowerOffSourceVM(ctx, config.VMwareVMID, vcenterHost, vcenterUsername, vcenterPassword); err != nil {
			return fmt.Errorf("failed to power off source VM: %w", err)
		}

		// Verify final power state
		finalState, err := ufe.vmaClient.GetVMPowerState(ctx, config.VMwareVMID, vcenterHost, vcenterUsername, vcenterPassword)
		if err != nil {
			logger.Warn("Failed to verify final power state", "error", err)
		} else {
			logger.Info("✅ Source VM power-off completed", "final_state", finalState)
		}

		return nil
	})
}

// executeFinalSyncPhase performs final incremental sync for live failover
func (ufe *UnifiedFailoverEngine) executeFinalSyncPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig) error {
	return ufe.jobTracker.RunStep(ctx, jobID, "final-sync", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("🔄 Starting final sync phase", "context_id", config.ContextID)

		// Step 1: Get VM context for discovery parameters
		vmContext, err := ufe.getVMContextByContextID(ctx, config.ContextID)
		if err != nil {
			return fmt.Errorf("failed to get VM context: %w", err)
		}

		// Step 2: Fresh VM discovery from VMA (following standard pattern)
		discoveredVM, err := ufe.discoverVMFromVMA(ctx, vmContext.VMName, vmContext.VCenterHost, vmContext.Datacenter)
		if err != nil {
			return fmt.Errorf("final sync VM discovery failed: %w", err)
		}

		logger.Info("✅ VM discovered for final sync",
			"vm_name", vmContext.VMName,
			"datacenter", vmContext.Datacenter)

		// Step 3: Create replication request with proper field mapping (following GUI pattern)
		sourceVM := map[string]interface{}{
			// ✅ EXACT FIELD MAPPING (critical for consistency)
			"id":           discoveredVM["id"],                                           // VMware VM UUID
			"name":         discoveredVM["name"],                                         // VM display name
			"path":         discoveredVM["path"],                                         // VMware inventory path
			"vm_id":        discoveredVM["id"],                                           // Duplicate for compatibility
			"vm_name":      discoveredVM["name"],                                         // Duplicate for compatibility
			"vm_path":      discoveredVM["path"],                                         // Duplicate for compatibility
			"datacenter":   discoveredVM["datacenter"],                                   // VMware datacenter
			"vcenter_host": vmContext.VCenterHost,                                        // vCenter server
			"cpus":         getIntOrDefault(discoveredVM, "num_cpu", 2),                  // CPU count fix
			"memory_mb":    getIntOrDefault(discoveredVM, "memory_mb", 4096),             // Memory in MB
			"power_state":  getStringOrDefault(discoveredVM, "power_state", "poweredOn"), // Power state
			"os_type":      getStringOrDefault(discoveredVM, "guest_os", "otherGuest"),   // OS type fix
			"vmx_version":  discoveredVM["vmx_version"],                                  // VMware version
			"disks":        discoveredVM["disks"],                                        // ⚠️ CRITICAL: Fresh disk array
			"networks":     discoveredVM["networks"],                                     // Network configuration
		}

		// Get active OSSEA config ID dynamically
		var activeConfigID int
		err = ufe.db.GetGormDB().Raw("SELECT id FROM ossea_configs WHERE is_active = 1 LIMIT 1").Scan(&activeConfigID).Error
		if err != nil {
			return fmt.Errorf("failed to get active OSSEA config ID: %w", err)
		}

		finalSyncRequest := map[string]interface{}{
			"source_vm":         sourceVM,
			"ossea_config_id":   activeConfigID,
			"vcenter_host":      vmContext.VCenterHost,
			"datacenter":        vmContext.Datacenter,
			"start_replication": true,
			// ✅ NO replication_type - let migration engine auto-detect incremental
			// ✅ NO scheduler metadata - this is not a scheduled job
		}

		// Step 4: Call standard OMA replication API
		finalSyncJobID, err := ufe.callOMAReplicationAPI(ctx, finalSyncRequest)
		if err != nil {
			return fmt.Errorf("failed to start final sync: %w", err)
		}

		logger.Info("✅ Final sync started", "replication_job_id", finalSyncJobID)

		// Step 5: Wait for completion using existing VMA poller pattern
		err = ufe.waitForReplicationCompletion(ctx, finalSyncJobID)
		if err != nil {
			// ✅ CRITICAL: If final sync fails, mark failover as failed
			ufe.failoverJobRepo.UpdateStatus(config.FailoverJobID, "failed")
			return fmt.Errorf("final sync failed: %w", err)
		}

		logger.Info("✅ Final sync completed successfully", "replication_job_id", finalSyncJobID)
		return nil
	})
}

// executeMultiVolumeSnapshotCreationPhase creates snapshots for ALL volumes in VM for complete protection
// 🆕 NEW: Replaces legacy single-snapshot approach with multi-volume support
func (ufe *UnifiedFailoverEngine) executeMultiVolumeSnapshotCreationPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig) (*VolumeSnapshotResult, error) {
	var result *VolumeSnapshotResult
	err := ufe.jobTracker.RunStep(ctx, jobID, "multi-volume-snapshot-creation", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("📸 Creating multi-volume snapshots for complete VM protection",
			"vm_context_id", config.ContextID,
			"vm_name", config.VMName)

		// Use NEW multi-volume snapshot service for complete protection
		var err error
		result, err = ufe.multiVolumeSnapshotService.CreateAllVolumeSnapshots(ctx, config.ContextID)
		if err != nil {
			return fmt.Errorf("failed to create multi-volume snapshots: %w", err)
		}

		logger.Info("✅ Multi-volume snapshots created successfully",
			"vm_context_id", config.ContextID,
			"snapshots_created", result.SuccessCount,
			"total_volumes", result.TotalVolumes)

		return nil
	})

	return result, err
}

// executeVolumeModeSwitch switches all volumes for a VM context between 'oma' and 'failover' modes
// 🆕 NEW: Critical for multi-volume snapshot detection during test failover
func (ufe *UnifiedFailoverEngine) executeVolumeModeSwitch(ctx context.Context, jobID string, vmContextID string, mode string) error {
	return ufe.jobTracker.RunStep(ctx, jobID, fmt.Sprintf("volume-mode-switch-%s", mode), func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("🔄 Switching volume operation mode",
			"vm_context_id", vmContextID,
			"target_mode", mode)

		// Update all device mappings for this VM context to the target mode
		// Note: device_mappings table is managed by Volume Daemon, using raw SQL query
		result := ufe.db.GetGormDB().Exec(
			"UPDATE device_mappings SET operation_mode = ? WHERE vm_context_id = ?",
			mode, vmContextID)

		if result.Error != nil {
			return fmt.Errorf("failed to update operation mode: %w", result.Error)
		}

		logger.Info("✅ Volume operation mode switched successfully",
			"vm_context_id", vmContextID,
			"target_mode", mode,
			"volumes_updated", result.RowsAffected)

		return nil
	})
}

// executeMultiVolumeCleanupPhase cleans up ALL volume snapshots after test failover completion
// 🆕 NEW: Complete multi-volume snapshot cleanup for comprehensive VM cleanup
func (ufe *UnifiedFailoverEngine) executeMultiVolumeCleanupPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig) error {
	// Step 1: Clean up multi-volume snapshots
	err := ufe.jobTracker.RunStep(ctx, jobID, "multi-volume-cleanup", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("🧹 Executing multi-volume snapshot cleanup",
			"vm_context_id", config.ContextID,
			"vm_name", config.VMName)

		// Use multi-volume snapshot service for complete cleanup
		err := ufe.multiVolumeSnapshotService.CleanupAllVolumeSnapshots(ctx, config.ContextID)
		if err != nil {
			return fmt.Errorf("failed to cleanup multi-volume snapshots: %w", err)
		}

		logger.Info("✅ Multi-volume cleanup completed successfully",
			"vm_context_id", config.ContextID)

		return nil
	})
	if err != nil {
		return err
	}

	// Step 2: Switch volumes back to 'oma' mode after cleanup
	// 🆕 CRITICAL: Restore volumes to normal operation mode
	if err := ufe.executeVolumeModeSwitch(ctx, jobID, config.ContextID, "oma"); err != nil {
		return fmt.Errorf("volume mode switch back to oma failed: %w", err)
	}

	return nil
}

// executeCloudStackSnapshotCreationPhase creates CloudStack volume snapshots for rollback protection
// ⚠️ LEGACY: This method only protects first disk - use executeMultiVolumeSnapshotCreationPhase for complete protection
func (ufe *UnifiedFailoverEngine) executeCloudStackSnapshotCreationPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig) (string, error) {
	var snapshotID string
	err := ufe.jobTracker.RunStep(ctx, jobID, "cloudstack-volume-snapshot-creation", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("📸 Creating CloudStack volume snapshot for rollback protection",
			"context_id", config.ContextID,
			"vm_name", config.VMName)

		// Create request object exactly like enhanced test failover
		request := &EnhancedTestFailoverRequest{
			ContextID:     config.ContextID,
			VMID:          config.VMwareVMID,
			VMName:        config.VMName,
			FailoverJobID: config.FailoverJobID,
			Timestamp:     config.Timestamp,
		}

		// Use existing snapshot operations for CloudStack volume snapshots
		var err error
		snapshotID, err = ufe.snapshotOperations.CreateCloudStackVolumeSnapshot(ctx, request)
		return err
	})

	// Update failover_jobs with snapshot ID for cleanup operations (like enhanced test failover)
	// CRITICAL FIX: Use the actual JobLog UUID from context, not the constructed config.FailoverJobID
	actualJobID := ufe.getJobIDFromContext(ctx)
	if err == nil && snapshotID != "" && actualJobID != "" {
		if updateErr := ufe.failoverJobRepo.UpdateSnapshot(actualJobID, snapshotID); updateErr != nil {
			logger := ufe.jobTracker.Logger(ctx)
			logger.Error("Failed to update failover job with snapshot ID",
				"error", updateErr,
				"actual_job_id", actualJobID,
				"config_job_id", config.FailoverJobID,
				"snapshot_id", snapshotID,
			)
		} else {
			logger := ufe.jobTracker.Logger(ctx)
			logger.Info("✅ Updated failover_jobs.ossea_snapshot_id",
				"actual_job_id", actualJobID,
				"config_job_id", config.FailoverJobID,
				"snapshot_id", snapshotID,
			)
		}
	}

	return snapshotID, err
}

// getJobIDFromContext extracts the JobLog UUID from the context
// This is the actual database job_id that should be used for database operations
func (ufe *UnifiedFailoverEngine) getJobIDFromContext(ctx context.Context) string {
	// The JobLog package stores the job ID in the context using JobIDFromCtx
	if jobID, ok := joblog.JobIDFromCtx(ctx); ok {
		return jobID
	}
	return ""
}

// executeVirtIOInjectionPhase injects VirtIO drivers for Windows VMs
// This matches the exact pattern from enhanced_test_failover.go
func (ufe *UnifiedFailoverEngine) executeVirtIOInjectionPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig, snapshotID string) (string, error) {
	var virtioStatus string
	err := ufe.jobTracker.RunStep(ctx, jobID, "virtio-driver-injection", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("💾 Executing VirtIO driver injection for KVM compatibility", "context_id", config.ContextID)

		// Create request object exactly like enhanced test failover
		request := &EnhancedTestFailoverRequest{
			ContextID:     config.ContextID,
			VMID:          config.VMwareVMID,
			VMName:        config.VMName,
			FailoverJobID: config.FailoverJobID,
			Timestamp:     config.Timestamp,
		}

		// Use existing VirtIO injection component with exact same method signature
		var err error
		virtioStatus, err = ufe.virtioInjection.ExecuteVirtIOInjectionStep(ctx, jobID, request, snapshotID)
		return err
	})
	return virtioStatus, err
}

// executeVMCreationPhase creates the destination VM
func (ufe *UnifiedFailoverEngine) executeVMCreationPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig) (string, error) {
	var destinationVMID string
	err := ufe.jobTracker.RunStep(ctx, jobID, "vm-creation", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		destinationVMName := config.GetDestinationVMName()
		logger.Info("🖥️ Executing VM creation phase",
			"destination_vm_name", destinationVMName,
			"network_strategy", config.NetworkStrategy)

		// 🌐 ENHANCED: Resolve network configuration for failover type
		// For now, use the first VMware network as the primary network
		// TODO: In the future, this could be enhanced to handle multiple networks
		vmwareNetworkName := "default" // This will be improved in a future iteration

		networkID, err := ufe.networkConfigProvider.GetNetworkIDForFailover(
			config.ContextID,
			config.FailoverType,
			vmwareNetworkName,
		)
		if err != nil {
			logger.Error("Failed to resolve network configuration", "error", err)
			return fmt.Errorf("failed to resolve network configuration: %w", err)
		}

		logger.Info("🌐 Resolved network configuration for VM creation",
			"context_id", config.ContextID,
			"failover_type", config.FailoverType,
			"network_id", networkID,
			"vmware_network", vmwareNetworkName)

		// Create request object for existing VM operations component
		request := &EnhancedTestFailoverRequest{
			ContextID:     config.ContextID,
			VMID:          config.VMwareVMID,
			VMName:        destinationVMName, // Use the calculated destination name
			FailoverJobID: config.FailoverJobID,
			Timestamp:     config.Timestamp,
		}

		// Use existing VM operations component with dynamic network configuration
		vmID, err := ufe.vmOperations.CreateTestVM(ctx, request, networkID)
		if err != nil {
			return err
		}
		destinationVMID = vmID
		return nil
	})
	return destinationVMID, err
}

// executeVolumeAttachmentPhase handles volume attachment exactly like enhanced_test_failover.go
func (ufe *UnifiedFailoverEngine) executeVolumeAttachmentPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig, destinationVMID string) error {
	return ufe.jobTracker.RunStep(ctx, jobID, "volume-attachment", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("🔗 Starting multi-disk volume attachment phase",
			"vm_id", config.VMName,
			"destination_vm_id", destinationVMID,
		)

		// 🎯 MULTI-DISK FIX: Get ALL volume information for complete VM failover
		multiDiskInfo, err := ufe.getMultiDiskVolumeInfoForVM(ctx, config.ContextID)
		if err != nil {
			return fmt.Errorf("failed to get multi-disk volume info: %w", err)
		}

		logger.Info("🔍 Multi-disk attachment plan",
			"os_volume", multiDiskInfo.OSVolume.DiskID,
			"data_volumes", len(multiDiskInfo.DataVolumes),
			"total_volumes", multiDiskInfo.TotalCount)

		// Step 1: Delete destination VM's default root volume
		if err := ufe.volumeOperations.DeleteTestVMRootVolume(ctx, destinationVMID); err != nil {
			return fmt.Errorf("failed to delete destination VM root volume: %w", err)
		}

		// Step 2: Detach OS volume from OMA
		logger.Info("🔗 Detaching OS volume from OMA", "volume_id", multiDiskInfo.OSVolume.VolumeID)
		if err := ufe.volumeOperations.DetachVolumeFromOMA(ctx, multiDiskInfo.OSVolume.VolumeID); err != nil {
			return fmt.Errorf("failed to detach OS volume from OMA: %w", err)
		}

		// Step 3: Attach OS volume to destination VM as root disk (device ID 0)
		logger.Info("🔗 Attaching OS volume as root disk",
			"volume_id", multiDiskInfo.OSVolume.VolumeID,
			"disk_id", multiDiskInfo.OSVolume.DiskID)
		if err := ufe.attachVolumeToDestinationVMAsRoot(ctx, multiDiskInfo.OSVolume.VolumeID, destinationVMID); err != nil {
			// Critical error - try to reattach OS volume to OMA
			logger.Error("❌ Critical: Failed to attach OS volume to destination VM - attempting recovery")
			if reattachErr := ufe.volumeOperations.ReattachVolumeToOMA(ctx, multiDiskInfo.OSVolume.VolumeID); reattachErr != nil {
				logger.Error("❌ Recovery failed: Could not reattach OS volume to OMA", "error", reattachErr)
			}
			return fmt.Errorf("failed to attach OS volume to destination VM: %w", err)
		}

		// Step 4: Process all data volumes (NEW MULTI-DISK LOGIC)
		for i, dataVolume := range multiDiskInfo.DataVolumes {
			logger.Info("🔗 Processing data volume",
				"index", i+1,
				"volume_id", dataVolume.VolumeID,
				"disk_id", dataVolume.DiskID,
				"size_gb", dataVolume.SizeGB)

			// Detach data volume from OMA
			logger.Info("🔗 Detaching data volume from OMA", "volume_id", dataVolume.VolumeID)
			if err := ufe.volumeOperations.DetachVolumeFromOMA(ctx, dataVolume.VolumeID); err != nil {
				logger.Error("Failed to detach data volume from OMA", "error", err, "disk_id", dataVolume.DiskID)
				return fmt.Errorf("failed to detach data volume %s from OMA: %w", dataVolume.DiskID, err)
			}

			// Attach data volume to destination VM (auto device ID assignment)
			logger.Info("🔗 Attaching data volume to test VM",
				"volume_id", dataVolume.VolumeID,
				"disk_id", dataVolume.DiskID)
			if err := ufe.attachVolumeToDestinationVM(ctx, dataVolume.VolumeID, destinationVMID); err != nil {
				logger.Error("❌ Failed to attach data volume to destination VM", "error", err, "disk_id", dataVolume.DiskID)
				// Try to reattach to OMA for recovery
				if reattachErr := ufe.volumeOperations.ReattachVolumeToOMA(ctx, dataVolume.VolumeID); reattachErr != nil {
					logger.Error("❌ Recovery failed: Could not reattach data volume to OMA", "error", reattachErr)
				}
				return fmt.Errorf("failed to attach data volume %s to destination VM: %w", dataVolume.DiskID, err)
			}

			logger.Info("✅ Data volume attached successfully",
				"volume_id", dataVolume.VolumeID,
				"disk_id", dataVolume.DiskID)
		}

		logger.Info("✅ Multi-disk volume attachment phase completed successfully",
			"os_volume", multiDiskInfo.OSVolume.VolumeID,
			"data_volumes", len(multiDiskInfo.DataVolumes),
			"total_volumes", multiDiskInfo.TotalCount,
			"destination_vm_id", destinationVMID,
		)

		return nil
	})
}

// executeVMStartupAndValidationPhase handles VM startup and validation exactly like enhanced_test_failover.go
func (ufe *UnifiedFailoverEngine) executeVMStartupAndValidationPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig, destinationVMID string) error {
	return ufe.jobTracker.RunStep(ctx, jobID, "vm-startup-and-validation", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)
		logger.Info("🚀 Starting VM startup and validation phase", "destination_vm_id", destinationVMID)

		// Step 1: Power on the destination VM
		if err := ufe.vmOperations.PowerOnTestVM(ctx, destinationVMID); err != nil {
			return fmt.Errorf("failed to power on destination VM: %w", err)
		}

		// Step 2: Validate the destination VM (exactly like enhanced_test_failover.go)
		vmSpec, err := ufe.helpers.GatherVMSpecifications(ctx, config.VMName)
		if err != nil {
			logger.Warn("⚠️ Could not gather VM specs for validation, skipping detailed validation", "error", err)
			vmSpec = nil // Continue with basic validation
		}

		validationResults, err := ufe.vmOperations.ValidateTestVM(ctx, destinationVMID, vmSpec)
		if err != nil {
			logger.Warn("⚠️ Destination VM validation failed - VM may still be functional", "error", err)
			// Don't fail the entire operation for validation issues, just log them
		} else {
			logger.Info("✅ Destination VM validation completed successfully",
				"destination_vm_id", destinationVMID,
				"validation_results", validationResults,
			)
		}

		return nil
	})
}

// executeStatusUpdatePhase updates VM context status
func (ufe *UnifiedFailoverEngine) executeStatusUpdatePhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig, destinationVMID string) error {
	return ufe.jobTracker.RunStep(ctx, jobID, "status-update", func(ctx context.Context) error {
		logger := ufe.jobTracker.Logger(ctx)

		// Determine the appropriate status based on failover type
		var newStatus string
		if config.IsLiveFailover() {
			newStatus = "failed_over_live"
		} else {
			newStatus = "failed_over_test"
		}

		logger.Info("📊 Executing status update phase",
			"context_id", config.ContextID,
			"new_status", newStatus,
			"destination_vm_id", destinationVMID)

		// Update VM context status
		if err := ufe.vmContextRepo.UpdateVMContextStatus(config.ContextID, newStatus); err != nil {
			return fmt.Errorf("failed to update VM context status: %w", err)
		}

		// Update failover job with destination VM ID
		if err := ufe.failoverJobRepo.UpdateDestinationVM(config.FailoverJobID, destinationVMID); err != nil {
			logger.Warn("Failed to update failover job destination VM ID", "error", err)
			// Don't fail the entire operation for this
		}

		// Mark failover job as completed with timestamp (exactly like enhanced_test_failover.go)
		if err := ufe.failoverJobRepo.MarkCompleted(config.FailoverJobID); err != nil {
			logger.Error("Failed to mark failover job as completed", "error", err)
		} else {
			logger.Info("✅ Marked failover job as completed with timestamp", "failover_job_id", config.FailoverJobID)
		}

		return nil
	})
}

// Helper methods for final sync implementation

// discoverVMFromVMA performs fresh VM discovery via VMA API using HTTP call
func (ufe *UnifiedFailoverEngine) discoverVMFromVMA(ctx context.Context, vmName, vCenterHost, datacenter string) (map[string]interface{}, error) {
	logger := ufe.jobTracker.Logger(ctx)

	// 🔧 CREDENTIAL FIX: Get vCenter credentials from secure credential service (NO FALLBACKS)
	// Initialize credential service
	encryptionService, err := services.NewCredentialEncryptionService()
	if err != nil {
		logger.Error("❌ Failed to initialize VMware credential encryption service for VMA discovery", "error", err.Error())
		return nil, fmt.Errorf("failed to initialize VMware credential encryption service: %w. Please ensure encryption service is configured", err)
	}

	credentialService := services.NewVMwareCredentialService(&ufe.db, encryptionService)
	creds, err := credentialService.GetDefaultCredentials(ctx)
	if err != nil {
		logger.Error("❌ Failed to retrieve VMware credentials from database for VMA discovery", "error", err.Error())
		return nil, fmt.Errorf("failed to retrieve VMware credentials from database: %w. "+
			"Please ensure VMware credentials are configured in GUI (Settings → VMware Credentials)", err)
	}

	// Use service-managed credentials (NO fallback to hardcoded values)
	discoveryUsername := creds.Username
	discoveryPassword := creds.Password

	logger.Info("✅ Using fresh VMware credentials from database for VMA discovery",
		"username", discoveryUsername,
		"credential_source", "database")

	requestBody := map[string]interface{}{
		"vcenter":    vCenterHost,
		"username":   discoveryUsername,
		"password":   discoveryPassword,
		"datacenter": datacenter,
		"filter":     vmName,
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal VMA discovery request: %w", err)
	}

	// Make HTTP POST call to VMA discovery API (same pattern as GUI)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post("http://localhost:9081/api/v1/discover", "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("VMA discovery HTTP call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("VMA discovery API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse discovery response
	var discoveryResponse struct {
		VMs []map[string]interface{} `json:"vms"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&discoveryResponse); err != nil {
		return nil, fmt.Errorf("failed to decode VMA discovery response: %w", err)
	}

	if len(discoveryResponse.VMs) == 0 {
		return nil, fmt.Errorf("VM not found in discovery results: %s", vmName)
	}

	// Return the first VM (should be exact match due to filter)
	vmSpecs := discoveryResponse.VMs[0]
	logger.Info("✅ Fresh VM discovery completed", "vm_name", vmName)
	return vmSpecs, nil
}

// callOMAReplicationAPI makes HTTP call to local OMA replication API
func (ufe *UnifiedFailoverEngine) callOMAReplicationAPI(ctx context.Context, request map[string]interface{}) (string, error) {
	logger := ufe.jobTracker.Logger(ctx)

	// Marshal request to JSON
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal replication request: %w", err)
	}

	// Make HTTP POST to local OMA API (following documented pattern)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post("http://localhost:8082/api/v1/replications", "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		return "", fmt.Errorf("HTTP call to replication API failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("replication API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response to get job ID
	var response struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode replication API response: %w", err)
	}

	logger.Info("✅ Replication API call successful", "job_id", response.JobID, "status", response.Status)
	return response.JobID, nil
}

// waitForReplicationCompletion waits for replication completion using database polling
// This follows the same pattern as the migration engine and GUI
func (ufe *UnifiedFailoverEngine) waitForReplicationCompletion(ctx context.Context, jobID string) error {
	logger := ufe.jobTracker.Logger(ctx)

	// Start VMA polling for this job (VMA poller updates database automatically)
	logger.Info("🚀 Starting VMA progress polling for final sync", "job_id", jobID)

	// Poll database for completion (same pattern as migration engine)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute) // Reasonable timeout for final sync

	for {
		select {
		case <-timeout:
			logger.Error("❌ Final sync timeout after 30 minutes")
			return fmt.Errorf("final sync timeout after 30 minutes")

		case <-ticker.C:
			// Check job status in database (updated by VMA poller)
			job, err := ufe.getReplicationJobStatus(ctx, jobID)
			if err != nil {
				logger.Debug("Retrying job status check", "error", err)
				continue // Keep trying
			}

			switch job.Status {
			case "completed":
				logger.Info("✅ Final sync completed", "job_id", jobID)
				return nil

			case "failed":
				logger.Error("❌ Final sync failed", "job_id", jobID, "error", job.ErrorMessage)
				return fmt.Errorf("final sync failed: %s", job.ErrorMessage)

			case "replicating":
				// Still in progress, continue polling
				logger.Debug("Final sync in progress",
					"progress", job.ProgressPercent,
					"operation", job.CurrentOperation)
				continue

			default:
				// Unknown status, continue polling
				logger.Debug("Unknown job status, continuing", "status", job.Status)
				continue
			}
		}
	}
}

// getReplicationJobStatus retrieves job status from database
func (ufe *UnifiedFailoverEngine) getReplicationJobStatus(ctx context.Context, jobID string) (*JobStatus, error) {
	// Create a simple replication job repository to query status
	replicationRepo := database.NewReplicationJobRepository(ufe.db)

	job, err := replicationRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job status: %w", err)
	}

	return &JobStatus{
		JobID:            job.ID,
		Status:           job.Status,
		ProgressPercent:  job.ProgressPercent,
		CurrentOperation: job.CurrentOperation,
		ErrorMessage:     job.ErrorMessage,
	}, nil
}

// getVMContextByContextID retrieves VM context by context ID
func (ufe *UnifiedFailoverEngine) getVMContextByContextID(ctx context.Context, contextID string) (*database.VMReplicationContext, error) {
	// Query VM context by context_id
	var vmContext database.VMReplicationContext
	if err := ufe.db.GetGormDB().Where("context_id = ?", contextID).First(&vmContext).Error; err != nil {
		return nil, fmt.Errorf("failed to get VM context by context_id %s: %w", contextID, err)
	}

	return &vmContext, nil
}

// JobStatus represents replication job status for polling
type JobStatus struct {
	JobID            string  `json:"job_id"`
	Status           string  `json:"status"`
	ProgressPercent  float64 `json:"progress_percent"`
	CurrentOperation string  `json:"current_operation"`
	ErrorMessage     string  `json:"error_message"`
}

// Helper functions for field mapping (same pattern as GUI)

// getIntOrDefault safely extracts integer values with fallback
func getIntOrDefault(data map[string]interface{}, key string, defaultValue int) int {
	if val, exists := data[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			// Try to parse string to int
			if parsed, err := strconv.Atoi(v); err == nil {
				return parsed
			}
		}
	}
	return defaultValue
}

// getStringOrDefault safely extracts string values with fallback
func getStringOrDefault(data map[string]interface{}, key string, defaultValue string) string {
	if val, exists := data[key]; exists {
		if str, ok := val.(string); ok && str != "" {
			return str
		}
	}
	return defaultValue
}

// storeOperationSummary saves a sanitized summary of the failover operation for persistent GUI visibility
func (ufe *UnifiedFailoverEngine) storeOperationSummary(
	ctx context.Context,
	config *UnifiedFailoverConfig,
	jobID string,
	externalJobID string,
	status string,
	result *UnifiedFailoverResult,
) {
	logger := ufe.jobTracker.Logger(ctx)
	
	// Get complete job details from JobLog
	jobSummary, err := ufe.jobTracker.FindJobByAnyID(jobID)
	if err != nil {
		logger.Warn("Could not retrieve job summary for operation storage", "error", err)
		return
	}
	
	// Build sanitized summary
	summary := map[string]interface{}{
		"job_id":           jobID,
		"external_job_id":  externalJobID,
		"operation_type":   fmt.Sprintf("%s_failover", config.FailoverType),
		"status":           status,
		"progress":         jobSummary.Progress.StepCompletion,
		"timestamp":        time.Now(),
		"steps_completed":  jobSummary.Progress.CompletedSteps,
		"steps_total":      jobSummary.Progress.TotalSteps,
		"duration_seconds": jobSummary.Progress.RuntimeSeconds,
	}
	
	// Add sanitized error information if failed
	if status == "failed" && result.Error != "" {
		// Find the failed step
		var failedStepName string
		for _, step := range jobSummary.Steps {
			if step.Status == joblog.StatusFailed {
				failedStepName = step.Name
				break
			}
		}
		
		if failedStepName != "" {
			// Sanitize the error for user display
			sanitized := SanitizeFailoverError(failedStepName, fmt.Errorf(result.Error))
			
			summary["failed_step"] = GetUserFriendlyStepName(failedStepName)
			summary["failed_step_internal"] = failedStepName
			summary["error_message"] = sanitized.UserMessage
			summary["error_category"] = sanitized.Category
			summary["error_severity"] = sanitized.Severity
			summary["actionable_steps"] = sanitized.ActionableSteps
			// Note: Technical details NOT included in summary (stays in JobLog only)
			
			logger.Info("📋 Sanitized error for GUI display",
				"failed_step", failedStepName,
				"user_message", sanitized.UserMessage,
				"category", sanitized.Category)
		}
	}
	
	// Serialize to JSON
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		logger.Error("Failed to marshal operation summary", "error", err)
		return
	}
	
	// Store in VM context for persistent visibility
	updates := map[string]interface{}{
		"last_operation_summary": string(summaryJSON),
		"updated_at":             time.Now(),
	}
	
	err = ufe.db.GetGormDB().Model(&database.VMReplicationContext{}).
		Where("context_id = ?", config.ContextID).
		Updates(updates).Error
	
	if err != nil {
		logger.Error("Failed to store operation summary", "error", err)
		return
	}
	
	logger.Info("✅ Stored sanitized operation summary for persistent GUI visibility",
		"context_id", config.ContextID,
		"operation", config.FailoverType,
		"status", status,
		"sanitized", status == "failed")
}
