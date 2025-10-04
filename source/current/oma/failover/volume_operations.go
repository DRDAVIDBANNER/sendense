// Package failover provides volume operations for enhanced test failover
package failover

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-oma/common"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
	"github.com/vexxhost/migratekit-oma/ossea"
)

// VolumeOperations handles all volume-related operations for test failover
type VolumeOperations struct {
	volumeClient *common.VolumeClient
	jobTracker   *joblog.Tracker
	db           *database.Connection
	helpers      *FailoverHelpers
}

// NewVolumeOperations creates a new volume operations handler
// Note: No longer requires pre-initialized osseaClient - credentials fetched fresh per operation
func NewVolumeOperations(jobTracker *joblog.Tracker, db *database.Connection, osseaClient *ossea.Client) *VolumeOperations {
	// Initialize helpers for credential management
	helpers := &FailoverHelpers{
		db:         db,
		jobTracker: jobTracker,
		// osseaClient is NOT cached - will be initialized fresh per operation
	}

	return &VolumeOperations{
		volumeClient: common.NewVolumeClient("http://localhost:8090"),
		jobTracker:   jobTracker,
		db:           db,
		helpers:      helpers,
	}
}

// DeleteTestVMRootVolume deletes the default root volume of a test VM using direct CloudStack SDK
func (vo *VolumeOperations) DeleteTestVMRootVolume(ctx context.Context, testVMID string) error {
	logger := vo.jobTracker.Logger(ctx)
	logger.Info("🗑️ Deleting test VM's default root volume via direct CloudStack SDK", "test_vm_id", testVMID)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := vo.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for volume deletion", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Step 1: List volumes for test VM using CloudStack SDK
	volumes, err := osseaClient.ListVMVolumes(testVMID)
	if err != nil {
		return fmt.Errorf("failed to list volumes for test VM %s: %w", testVMID, err)
	}

	// Step 2: Find the root volume (Type = "ROOT")
	var rootVolumeID string
	for _, volume := range volumes {
		if volume.Type == "ROOT" {
			rootVolumeID = volume.ID
			logger.Info("📝 Found test VM root volume",
				"test_vm_id", testVMID,
				"root_volume_id", volume.ID,
				"volume_size_gb", volume.SizeGB,
			)
			break
		}
	}

	if rootVolumeID == "" {
		return fmt.Errorf("no root volume found for test VM %s", testVMID)
	}

	// Step 3: Detach root volume using CloudStack SDK
	logger.Info("🔓 Detaching root volume from test VM",
		"test_vm_id", testVMID,
		"root_volume_id", rootVolumeID,
	)

	err = osseaClient.DetachVolumeFromVM(rootVolumeID, testVMID)
	if err != nil {
		return fmt.Errorf("failed to detach root volume %s from test VM %s: %w", rootVolumeID, testVMID, err)
	}

	// Step 4: Delete the detached volume using CloudStack SDK
	logger.Info("🗑️ Deleting detached root volume", "root_volume_id", rootVolumeID)

	err = osseaClient.DeleteVolume(rootVolumeID)
	if err != nil {
		return fmt.Errorf("failed to delete root volume %s: %w", rootVolumeID, err)
	}

	logger.Info("✅ Test VM root volume deleted successfully", "root_volume_id", rootVolumeID)
	return nil
}

// DetachVolumeFromOMA detaches volume from OMA using Volume Daemon
func (vo *VolumeOperations) DetachVolumeFromOMA(ctx context.Context, volumeID string) error {
	logger := vo.jobTracker.Logger(ctx)
	logger.Info("🔌 Detaching volume from OMA via Volume Daemon", "volume_id", volumeID)

	// Use Volume Management Daemon for detachment
	operation, err := vo.volumeClient.DetachVolume(context.Background(), volumeID)
	if err != nil {
		return fmt.Errorf("failed to start volume detachment via daemon: %w", err)
	}

	// Wait for completion
	finalOp, err := vo.volumeClient.WaitForCompletionWithTimeout(context.Background(), operation.ID, 180*time.Second)
	if err != nil {
		return fmt.Errorf("volume detachment operation failed: %w", err)
	}

	logger.Info("✅ Volume detached from OMA successfully",
		"volume_id", volumeID,
		"operation_id", finalOp.ID,
	)

	return nil
}

// ReattachVolumeToOMA emergency recovery method using Volume Daemon
func (vo *VolumeOperations) ReattachVolumeToOMA(ctx context.Context, volumeID string) error {
	logger := vo.jobTracker.Logger(ctx)
	logger.Warn("🚨 Emergency: Reattaching volume to OMA via Volume Daemon", "volume_id", volumeID)

	// Get OMA VM ID from configuration
	omaVMID, err := vo.getOMAVMID(ctx, volumeID)
	if err != nil {
		return fmt.Errorf("failed to get OMA VM ID: %w", err)
	}

	operation, err := vo.volumeClient.AttachVolume(context.Background(), volumeID, omaVMID)
	if err != nil {
		return fmt.Errorf("failed to reattach volume to OMA: %w", err)
	}

	// Wait for reattachment to complete
	finalOp, err := vo.volumeClient.WaitForCompletionWithTimeout(context.Background(), operation.ID, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("volume reattachment failed or timed out: %w", err)
	}

	devicePath := finalOp.Response["device_path"]
	logger.Info("✅ Volume reattached to OMA successfully",
		"volume_id", volumeID,
		"device_path", devicePath,
		"oma_vm_id", omaVMID,
	)
	return nil
}

// getOMAVMID retrieves the OMA VM ID for volume reattachment
func (vo *VolumeOperations) getOMAVMID(ctx context.Context, volumeID string) (string, error) {
	// Use helper method to get OMA VM ID from configuration
	return vo.helpers.GetOMAVMID(ctx)
}
