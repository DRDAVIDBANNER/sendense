// Package failover provides enhanced test failover orchestration with CloudStack snapshots and VirtIO injection
package failover

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-sha/common"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/ossea"
)

// EnhancedTestFailoverEngine orchestrates enhanced test failover with modular components
type EnhancedTestFailoverEngine struct {
	// Core dependencies
	db              *database.Connection
	osseaClient     *ossea.Client
	jobTracker      *joblog.Tracker
	failoverJobRepo *database.FailoverJobRepository
	vmContextRepo   *database.VMReplicationContextRepository

	// Modular components
	vmOperations       *VMOperations
	volumeOperations   *VolumeOperations
	virtioInjection    *VirtIOInjection
	snapshotOperations *SnapshotOperations
	validation         *FailoverValidation
	helpers            *FailoverHelpers
}

// EnhancedTestFailoverRequest represents a test failover request
type EnhancedTestFailoverRequest struct {
	ContextID     string    `json:"context_id"`
	VMID          string    `json:"vm_id"`
	VMName        string    `json:"vm_name"`
	FailoverJobID string    `json:"failover_job_id"`
	Timestamp     time.Time `json:"timestamp"`
}

// VolumeInfo represents volume information for failover operations
type VolumeInfo struct {
	VolumeID   string `json:"volume_id"`
	VolumeName string `json:"volume_name"`
	DevicePath string `json:"device_path"`
	SizeGB     int    `json:"size_gb"`
}

// NewEnhancedTestFailoverEngine creates a new enhanced test failover engine with modular architecture
func NewEnhancedTestFailoverEngine(
	db *database.Connection,
	osseaClient *ossea.Client,
	failoverJobRepo *database.FailoverJobRepository,
	validator *PreFailoverValidator,
	jobTracker *joblog.Tracker,
) *EnhancedTestFailoverEngine {
	// Create modular components
	helpers := NewFailoverHelpers(db, osseaClient, jobTracker, failoverJobRepo)
	vmOperations := NewVMOperations(osseaClient, jobTracker, db)
	volumeOperations := NewVolumeOperations(jobTracker, db, osseaClient)
	virtioInjection := NewVirtIOInjection(db, jobTracker)
	snapshotOperations := NewSnapshotOperations(db, osseaClient, jobTracker)
	validation := NewFailoverValidation(jobTracker, helpers)

	// Initialize VM context repository for status updates
	vmContextRepo := database.NewVMReplicationContextRepository(*db)

	return &EnhancedTestFailoverEngine{
		db:                 db,
		osseaClient:        osseaClient,
		jobTracker:         jobTracker,
		failoverJobRepo:    failoverJobRepo,
		vmContextRepo:      vmContextRepo,
		vmOperations:       vmOperations,
		volumeOperations:   volumeOperations,
		virtioInjection:    virtioInjection,
		snapshotOperations: snapshotOperations,
		validation:         validation,
		helpers:            helpers,
	}
}

// ExecuteEnhancedTestFailover orchestrates the complete enhanced test failover process
func (etfe *EnhancedTestFailoverEngine) ExecuteEnhancedTestFailover(
	ctx context.Context,
	request *EnhancedTestFailoverRequest,
) (string, error) {
	// START: Job creation with enhanced joblog
	externalJobID := fmt.Sprintf("enhanced-test-failover-%s-%d",
		request.VMName, time.Now().Unix())

	ctx, jobID, err := etfe.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:       "failover",
		Operation:     "enhanced-test-failover",
		Owner:         stringPtr("system"),
		ExternalJobID: &externalJobID,        // Enhanced: GUI job ID correlation
		JobCategory:   stringPtr("failover"), // Enhanced: High-level categorization
		Metadata: map[string]interface{}{
			"vm_id":                    request.VMID,
			"vm_name":                  request.VMName,
			"original_failover_job_id": request.FailoverJobID,
			"external_job_id":          externalJobID, // Track external ID in metadata too
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to start enhanced test failover job: %w", err)
	}
	defer etfe.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	// CORRELATION FIX: Use JobLog ID as the primary identifier for database consistency
	// Store original API ID in metadata, use JobLog UUID for all database operations
	originalAPIJobID := request.FailoverJobID
	request.FailoverJobID = jobID // Use JobLog UUID for database correlation

	// Create failover_jobs table entry for atomic status tracking
	if err := etfe.helpers.CreateTestFailoverJob(ctx, request); err != nil {
		return "", fmt.Errorf("failed to create failover job record: %w", err)
	}

	// Update VM context status to indicate test failover in progress
	if request.ContextID != "" {
		if err := etfe.vmContextRepo.UpdateVMContextStatus(request.ContextID, "failed_over_test"); err != nil {
			// Log error but don't fail the operation
			logger := etfe.jobTracker.Logger(ctx)
			logger.Error("Failed to update VM context status to failed_over_test", "error", err, "context_id", request.ContextID)
		}
	}

	// Get logger with job context
	logger := etfe.jobTracker.Logger(ctx)
	logger.Info("Enhanced test failover started",
		"context_id", request.ContextID,
		"vm_id", request.VMID,
		"vm_name", request.VMName,
		"failover_job_id", request.FailoverJobID,
		"original_api_job_id", originalAPIJobID,
		"correlation_fixed", true,
	)

	// PHASE 1: Pre-failover validation
	if err := etfe.failoverJobRepo.UpdateStatus(request.FailoverJobID, "validating"); err != nil {
		logger.Error("Failed to update failover job status to validating", "error", err)
	}

	if err := etfe.jobTracker.RunStep(ctx, jobID, "pre-failover-validation", func(ctx context.Context) error {
		return etfe.validation.ExecutePreFailoverValidation(ctx, request)
	}); err != nil {
		etfe.failoverJobRepo.UpdateStatus(request.FailoverJobID, "failed")
		return "", fmt.Errorf("pre-failover validation failed: %w", err)
	}

	// Update status to executing
	if err := etfe.failoverJobRepo.UpdateStatus(request.FailoverJobID, "executing"); err != nil {
		logger.Error("Failed to update failover job status to executing", "error", err)
	}

	// PHASE 2: Create CloudStack volume snapshot for rollback protection
	var snapshotID string
	if err := etfe.jobTracker.RunStep(ctx, jobID, "cloudstack-volume-snapshot-creation", func(ctx context.Context) error {
		var err error
		snapshotID, err = etfe.snapshotOperations.CreateCloudStackVolumeSnapshot(ctx, request)
		return err
	}); err != nil {
		return "", fmt.Errorf("CloudStack volume snapshot creation failed: %w", err)
	}

	// Update failover_jobs with snapshot ID for cleanup operations
	if snapshotID != "" {
		if err := etfe.failoverJobRepo.UpdateSnapshot(request.FailoverJobID, snapshotID); err != nil {
			logger.Error("Failed to update failover job with snapshot ID",
				"error", err,
				"failover_job_id", request.FailoverJobID,
				"snapshot_id", snapshotID,
			)
		} else {
			logger.Info("✅ Updated failover_jobs.ossea_snapshot_id",
				"failover_job_id", request.FailoverJobID,
				"snapshot_id", snapshotID,
			)
		}
	}

	// PHASE 3: VirtIO driver injection for KVM compatibility
	var virtioStatus string
	if err := etfe.jobTracker.RunStep(ctx, jobID, "virtio-driver-injection", func(ctx context.Context) error {
		var err error
		virtioStatus, err = etfe.virtioInjection.ExecuteVirtIOInjectionStep(ctx, jobID, request, snapshotID)
		return err
	}); err != nil {
		return "", fmt.Errorf("VirtIO driver injection failed: %w", err)
	}

	// PHASE 4: Test VM creation
	var testVMID string
	if err := etfe.jobTracker.RunStep(ctx, jobID, "test-vm-creation", func(ctx context.Context) error {
		// LEGACY COMPATIBILITY: Get default network ID for old enhanced test failover
		defaultNetworkID := getDefaultNetworkID(*etfe.db)
		if defaultNetworkID == "" {
			return fmt.Errorf("no default network configuration available")
		}

		var err error
		testVMID, err = etfe.vmOperations.CreateTestVM(ctx, request, defaultNetworkID)
		return err
	}); err != nil {
		etfe.failoverJobRepo.UpdateStatus(request.FailoverJobID, "failed")
		return "", fmt.Errorf("test VM creation failed: %w", err)
	}

	// Update failover_jobs with destination VM ID (CRITICAL for cleanup operations)
	if testVMID != "" {
		if err := etfe.failoverJobRepo.UpdateDestinationVM(request.FailoverJobID, testVMID); err != nil {
			logger.Error("Failed to update failover job with destination VM ID",
				"error", err,
				"failover_job_id", request.FailoverJobID,
				"destination_vm_id", testVMID,
			)
		} else {
			logger.Info("✅ Updated failover_jobs.destination_vm_id",
				"failover_job_id", request.FailoverJobID,
				"destination_vm_id", testVMID,
			)
		}
	}

	// PHASE 5: Volume attachment to test VM
	if err := etfe.jobTracker.RunStep(ctx, jobID, "volume-attachment", func(ctx context.Context) error {
		return etfe.executeVolumeAttachment(ctx, request, testVMID)
	}); err != nil {
		return "", fmt.Errorf("volume attachment failed: %w", err)
	}

	// PHASE 6: Test VM startup and validation
	if err := etfe.jobTracker.RunStep(ctx, jobID, "test-vm-startup", func(ctx context.Context) error {
		return etfe.executeVMStartupAndValidation(ctx, request, testVMID)
	}); err != nil {
		return "", fmt.Errorf("test VM startup failed: %w", err)
	}

	// Mark failover job as completed with timestamp (FINAL STATUS)
	if err := etfe.failoverJobRepo.MarkCompleted(request.FailoverJobID); err != nil {
		logger.Error("Failed to mark failover job as completed", "error", err)
	} else {
		logger.Info("✅ Marked failover job as completed with timestamp", "failover_job_id", request.FailoverJobID)
	}

	logger.Info("✅ Enhanced test failover completed successfully",
		"vm_id", request.VMID,
		"test_vm_id", testVMID,
		"virtio_status", virtioStatus,
		"snapshot_id", snapshotID,
	)

	return jobID, nil
}

// executeVolumeAttachment handles the volume attachment phase
func (etfe *EnhancedTestFailoverEngine) executeVolumeAttachment(
	ctx context.Context,
	request *EnhancedTestFailoverRequest,
	testVMID string,
) error {
	logger := etfe.jobTracker.Logger(ctx)
	logger.Info("🔗 Starting volume attachment phase",
		"vm_id", request.VMID,
		"test_vm_id", testVMID,
	)

	// Get volume information for the VM
	volumeInfo, err := etfe.getVolumeInfoForVM(ctx, request.VMID)
	if err != nil {
		return fmt.Errorf("failed to get volume info: %w", err)
	}

	// Step 1: Delete test VM's default root volume
	if err := etfe.volumeOperations.DeleteTestVMRootVolume(ctx, testVMID); err != nil {
		return fmt.Errorf("failed to delete test VM root volume: %w", err)
	}

	// Step 2: Detach volume from SHA
	if err := etfe.volumeOperations.DetachVolumeFromOMA(ctx, volumeInfo.VolumeID); err != nil {
		return fmt.Errorf("failed to detach volume from SHA: %w", err)
	}

	// Step 3: Attach volume to test VM as root disk
	if err := etfe.attachVolumeToTestVMAsRoot(ctx, volumeInfo.VolumeID, testVMID); err != nil {
		// Critical error - try to reattach to SHA
		logger.Error("❌ Critical: Failed to attach volume to test VM - attempting recovery")
		if reattachErr := etfe.volumeOperations.ReattachVolumeToOMA(ctx, volumeInfo.VolumeID); reattachErr != nil {
			logger.Error("🚨 CRITICAL: Volume orphaned - manual intervention required")
		}
		return fmt.Errorf("failed to attach volume to test VM: %w", err)
	}

	logger.Info("✅ Volume attachment phase completed successfully",
		"volume_id", volumeInfo.VolumeID,
		"test_vm_id", testVMID,
	)

	return nil
}

// executeVMStartupAndValidation handles the VM startup and validation phase
func (etfe *EnhancedTestFailoverEngine) executeVMStartupAndValidation(
	ctx context.Context,
	request *EnhancedTestFailoverRequest,
	testVMID string,
) error {
	logger := etfe.jobTracker.Logger(ctx)
	logger.Info("🚀 Starting VM startup and validation phase", "test_vm_id", testVMID)

	// Step 1: Power on the test VM
	if err := etfe.vmOperations.PowerOnTestVM(ctx, testVMID); err != nil {
		return fmt.Errorf("failed to power on test VM: %w", err)
	}

	// Step 2: Validate the test VM
	vmSpec, err := etfe.helpers.GatherVMSpecifications(ctx, request.VMID)
	if err != nil {
		logger.Warn("⚠️ Could not gather VM specs for validation, skipping detailed validation", "error", err)
		vmSpec = nil // Continue with basic validation
	}

	validationResults, err := etfe.vmOperations.ValidateTestVM(ctx, testVMID, vmSpec)
	if err != nil {
		logger.Warn("⚠️ Test VM validation failed - VM may still be functional", "error", err)
		// Don't fail the entire operation for validation issues, just log them
	} else {
		logger.Info("✅ Test VM validation completed successfully",
			"test_vm_id", testVMID,
			"validation_results", validationResults,
		)
	}

	logger.Info("✅ VM startup and validation phase completed successfully", "test_vm_id", testVMID)
	return nil
}

// getVolumeInfoForVM retrieves volume information for the specified VM
func (etfe *EnhancedTestFailoverEngine) getVolumeInfoForVM(ctx context.Context, vmID string) (*VolumeInfo, error) {
	// Query the database chain: replication_jobs -> vm_disks -> ossea_volumes
	volumeInfo, err := etfe.queryVolumeInfoFromDatabase(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume info from database: %w", err)
	}

	return volumeInfo, nil
}

// queryVolumeInfoFromDatabase queries the database following the normalized schema
// Flow: source_vm_id -> replication_jobs -> vm_disks -> ossea_volumes
func (etfe *EnhancedTestFailoverEngine) queryVolumeInfoFromDatabase(ctx context.Context, sourceVMID string) (*VolumeInfo, error) {
	logger := etfe.jobTracker.Logger(ctx)
	logger.Info("🔍 Querying database for volume info", "source_vm_id", sourceVMID)

	// Step 1: Find replication job for this source VM
	var replicationJob database.ReplicationJob
	err := (*etfe.db).GetGormDB().Where("source_vm_id = ?", sourceVMID).
		Order("created_at DESC").
		First(&replicationJob).Error
	if err != nil {
		logger.Error("Failed to find replication job", "error", err, "source_vm_id", sourceVMID)
		return nil, fmt.Errorf("no replication job found for VM %s: %w", sourceVMID, err)
	}

	logger.Info("Found replication job", "job_id", replicationJob.ID, "source_vm_name", replicationJob.SourceVMName)

	// Step 2: Find VM disks for this job
	var vmDisks []database.VMDisk
	err = (*etfe.db).GetGormDB().Where("job_id = ?", replicationJob.ID).Find(&vmDisks).Error
	if err != nil {
		logger.Error("Failed to find VM disks", "error", err, "job_id", replicationJob.ID)
		return nil, fmt.Errorf("no VM disks found for job %s: %w", replicationJob.ID, err)
	}

	if len(vmDisks) == 0 {
		logger.Error("No VM disks found for replication job", "job_id", replicationJob.ID)
		return nil, fmt.Errorf("no VM disks found for job %s", replicationJob.ID)
	}

	// For now, use the first disk (typically the root disk)
	// TODO: Add logic to select the correct disk if multiple disks exist
	vmDisk := vmDisks[0]
	logger.Info("Using VM disk", "disk_id", vmDisk.DiskID, "ossea_volume_id", vmDisk.OSSEAVolumeID)

	// Step 3: Find OSSEA volume
	var osseaVolume database.OSSEAVolume
	err = (*etfe.db).GetGormDB().Where("id = ?", vmDisk.OSSEAVolumeID).First(&osseaVolume).Error
	if err != nil {
		logger.Error("Failed to find OSSEA volume", "error", err, "ossea_volume_id", vmDisk.OSSEAVolumeID)
		return nil, fmt.Errorf("no OSSEA volume found for ID %d: %w", vmDisk.OSSEAVolumeID, err)
	}

	logger.Info("Found OSSEA volume",
		"volume_id", osseaVolume.VolumeID,
		"volume_name", osseaVolume.VolumeName,
		"size_gb", osseaVolume.SizeGB,
		"device_path", osseaVolume.DevicePath)

	// Return volume info
	return &VolumeInfo{
		VolumeID:   osseaVolume.VolumeID,
		VolumeName: osseaVolume.VolumeName,
		DevicePath: osseaVolume.DevicePath,
		SizeGB:     osseaVolume.SizeGB,
	}, nil
}

func (etfe *EnhancedTestFailoverEngine) attachVolumeToTestVMAsRoot(ctx context.Context, volumeID, testVMID string) error {
	logger := etfe.jobTracker.Logger(ctx)
	logger.Info("🔗 Attaching volume to test VM as root disk via Volume Daemon",
		"volume_id", volumeID,
		"test_vm_id", testVMID,
	)

	// Use Volume Daemon for root volume attachment
	volumeClient := common.NewVolumeClient("http://localhost:8090")
	operation, err := volumeClient.AttachVolumeAsRoot(ctx, volumeID, testVMID)
	if err != nil {
		return fmt.Errorf("failed to start volume attachment via daemon: %w", err)
	}

	logger.Info("⏳ Waiting for volume attachment completion via Volume Daemon",
		"operation_id", operation.ID,
		"volume_id", volumeID,
		"test_vm_id", testVMID,
	)

	// Wait for completion with device correlation
	finalOp, err := volumeClient.WaitForCompletionWithTimeout(ctx, operation.ID, 300*time.Second)
	if err != nil {
		logger.Error("Volume attachment operation failed", "error", err, "operation_id", operation.ID)
		return fmt.Errorf("volume attachment operation failed: %w", err)
	}

	devicePath := finalOp.Response["device_path"]
	logger.Info("✅ Volume attached to test VM as root disk successfully",
		"volume_id", volumeID,
		"test_vm_id", testVMID,
		"device_path", devicePath,
	)

	return nil
}
