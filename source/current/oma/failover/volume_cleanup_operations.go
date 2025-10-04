// Package failover provides volume cleanup operations for enhanced test failover cleanup
package failover

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-oma/common"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
)

// VolumeCleanupOperations handles volume-related cleanup operations
type VolumeCleanupOperations struct {
	volumeClient *common.VolumeClient
	jobTracker   *joblog.Tracker
	db           database.Connection
}

// NewVolumeCleanupOperations creates a new volume cleanup operations handler
func NewVolumeCleanupOperations(jobTracker *joblog.Tracker, db database.Connection) *VolumeCleanupOperations {
	return &VolumeCleanupOperations{
		volumeClient: common.NewVolumeClient("http://localhost:8090"),
		jobTracker:   jobTracker,
		db:           db,
	}
}

// DetachVolumesFromTestVM detaches all volumes from test VM using Volume Daemon
func (vco *VolumeCleanupOperations) DetachVolumesFromTestVM(ctx context.Context, testVMID string) ([]string, error) {
	logger := vco.jobTracker.Logger(ctx)
	logger.Info("🔌 Detaching volumes from test VM", "test_vm_id", testVMID)

	// Get list of volumes attached to the test VM
	volumes, err := vco.volumeClient.ListVolumes(context.Background(), testVMID)
	if err != nil {
		logger.Error("Failed to list volumes for test VM", "error", err, "test_vm_id", testVMID)
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	var volumeIDs []string
	var detachOperations []string

	// Find volumes attached to this test VM
	for _, volume := range volumes {
		if volume.VMID == testVMID {
			volumeIDs = append(volumeIDs, volume.VolumeID)
			logger.Info("Found volume attached to test VM",
				"volume_id", volume.VolumeID,
				"test_vm_id", testVMID,
				"device_path", volume.DevicePath,
			)
		}
	}

	if len(volumeIDs) == 0 {
		logger.Info("No volumes found attached to test VM", "test_vm_id", testVMID)
		return []string{}, nil
	}

	logger.Info("Starting volume detachment process",
		"test_vm_id", testVMID,
		"volume_count", len(volumeIDs),
	)

	// Detach each volume using Volume Daemon
	for _, volumeID := range volumeIDs {
		logger.Info("Detaching volume from test VM", "volume_id", volumeID, "test_vm_id", testVMID)

		operation, err := vco.volumeClient.DetachVolume(context.Background(), volumeID)
		if err != nil {
			logger.Error("Failed to initiate volume detachment", "error", err, "volume_id", volumeID)
			return nil, fmt.Errorf("failed to detach volume %s: %w", volumeID, err)
		}

		detachOperations = append(detachOperations, operation.ID)
		logger.Info("Volume detachment initiated", "volume_id", volumeID, "operation_id", operation.ID)
	}

	// Wait for all detachment operations to complete
	for i, operationID := range detachOperations {
		volumeID := volumeIDs[i]
		logger.Info("Waiting for volume detachment completion", "volume_id", volumeID, "operation_id", operationID)

		finalOp, err := vco.volumeClient.WaitForCompletionWithTimeout(context.Background(), operationID, 3*time.Minute)
		if err != nil {
			logger.Error("Volume detachment operation failed", "error", err, "volume_id", volumeID, "operation_id", operationID)
			return nil, fmt.Errorf("volume detachment operation failed for %s: %w", volumeID, err)
		}

		if finalOp.Status != "completed" {
			logger.Error("Volume detachment operation did not complete successfully", "volume_id", volumeID, "status", finalOp.Status)
			return nil, fmt.Errorf("volume detachment failed for %s with status: %s", volumeID, finalOp.Status)
		}

		logger.Info("Volume detached successfully", "volume_id", volumeID, "operation_id", operationID)
	}

	logger.Info("✅ All volumes detached from test VM successfully",
		"test_vm_id", testVMID,
		"volumes_detached", len(volumeIDs),
	)

	return volumeIDs, nil
}

// ReattachVolumesToOMA reattaches volumes to original OMA VM using Volume Daemon
func (vco *VolumeCleanupOperations) ReattachVolumesToOMA(ctx context.Context, volumeIDs []string, vmNameOrID string) error {
	logger := vco.jobTracker.Logger(ctx)
	logger.Info("🔗 Reattaching volumes to OMA via Volume Daemon", "volume_count", len(volumeIDs), "vm_name_or_id", vmNameOrID)

	if len(volumeIDs) == 0 {
		logger.Info("No volumes to reattach to OMA")
		return nil
	}

	// Get OMA VM ID from database (following Volume Daemon v2.1.2 pattern)
	omaVMID, err := vco.getOMAVMIDFromDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to get OMA VM ID from database: %w", err)
	}
	logger.Info("Retrieved OMA VM ID from database for volume reattachment via Volume Daemon", "oma_vm_id", omaVMID)

	var attachOperations []string

	// Reattach each volume to OMA using Volume Daemon
	for _, volumeID := range volumeIDs {
		logger.Info("🔗 Reattaching volume to OMA via Volume Daemon", "volume_id", volumeID, "oma_vm_id", omaVMID)

		operation, err := vco.volumeClient.AttachVolume(context.Background(), volumeID, omaVMID)
		if err != nil {
			logger.Error("Failed to initiate volume reattachment to OMA via Volume Daemon", "error", err, "volume_id", volumeID, "oma_vm_id", omaVMID)
			return fmt.Errorf("failed to reattach volume %s to OMA: %w", volumeID, err)
		}

		attachOperations = append(attachOperations, operation.ID)
		logger.Info("Volume reattachment to OMA initiated via Volume Daemon", "volume_id", volumeID, "operation_id", operation.ID)
	}

	// Wait for all reattachment operations to complete
	for i, operationID := range attachOperations {
		volumeID := volumeIDs[i]
		logger.Info("Waiting for volume reattachment completion", "volume_id", volumeID, "operation_id", operationID)

		finalOp, err := vco.volumeClient.WaitForCompletionWithTimeout(context.Background(), operationID, 3*time.Minute)
		if err != nil {
			logger.Error("Volume reattachment operation failed", "error", err, "volume_id", volumeID, "operation_id", operationID)
			return fmt.Errorf("volume reattachment operation failed for %s: %w", volumeID, err)
		}

		if finalOp.Status != "completed" {
			logger.Error("Volume reattachment operation did not complete successfully", "volume_id", volumeID, "status", finalOp.Status)
			return fmt.Errorf("volume reattachment failed for %s with status: %s", volumeID, finalOp.Status)
		}

		devicePath := finalOp.Response["device_path"]
		logger.Info("Volume reattached to OMA successfully",
			"volume_id", volumeID,
			"operation_id", operationID,
			"device_path", devicePath,
		)
	}

	logger.Info("✅ All volumes reattached to OMA successfully via Volume Daemon",
		"oma_vm_id", omaVMID,
		"volumes_reattached", len(volumeIDs),
	)

	return nil
}

// getOMAVMIDFromDatabase retrieves the OMA VM ID from the active ossea_configs record
// Following Volume Daemon v2.1.2 pattern for dynamic OMA VM ID lookup
func (vco *VolumeCleanupOperations) getOMAVMIDFromDatabase(ctx context.Context) (string, error) {
	logger := vco.jobTracker.Logger(ctx)
	
	var omaVMID string
	err := vco.db.GetGormDB().Raw("SELECT oma_vm_id FROM ossea_configs WHERE is_active = 1 LIMIT 1").Scan(&omaVMID).Error
	if err != nil {
		logger.Error("Failed to query OMA VM ID from database", "error", err)
		return "", fmt.Errorf("failed to query OMA VM ID from ossea_configs: %w", err)
	}
	
	if omaVMID == "" {
		logger.Error("OMA VM ID is empty in database")
		return "", fmt.Errorf("OMA VM ID is empty in ossea_configs table")
	}
	
	logger.Debug("Successfully retrieved OMA VM ID from database", "oma_vm_id", omaVMID)
	return omaVMID, nil
}
