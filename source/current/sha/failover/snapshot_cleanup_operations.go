// Package failover provides snapshot cleanup operations for enhanced test failover cleanup
package failover

import (
	"context"
	"fmt"

	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/ossea"
)

// SnapshotCleanupOperations handles CloudStack snapshot cleanup operations
type SnapshotCleanupOperations struct {
	jobTracker *joblog.Tracker
	helpers    *CleanupHelpers
}

// NewSnapshotCleanupOperations creates a new snapshot cleanup operations handler
// Note: Takes CleanupHelpers for dynamic credential initialization
func NewSnapshotCleanupOperations(osseaClient *ossea.Client, jobTracker *joblog.Tracker, helpers *CleanupHelpers) *SnapshotCleanupOperations {
	return &SnapshotCleanupOperations{
		jobTracker: jobTracker,
		helpers:    helpers,
	}
}

// RollbackCloudStackVolumeSnapshot rolls back to CloudStack volume snapshot for cleanup
func (sco *SnapshotCleanupOperations) RollbackCloudStackVolumeSnapshot(ctx context.Context, snapshotID string) error {
	logger := sco.jobTracker.Logger(ctx)
	logger.Info("🔄 Performing CloudStack volume snapshot rollback", "snapshot_id", snapshotID)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := sco.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for snapshot rollback", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Get snapshot details first to verify it exists and get volume information
	snapshot, err := osseaClient.GetVolumeSnapshot(snapshotID)
	if err != nil {
		logger.Error("Failed to get snapshot details for rollback", "error", err, "snapshot_id", snapshotID)
		return fmt.Errorf("failed to get snapshot details: %w", err)
	}

	logger.Info("Retrieved snapshot details for rollback",
		"snapshot_id", snapshotID,
		"volume_id", snapshot.VolumeID,
		"state", snapshot.State,
		"created", snapshot.Created,
	)

	// Verify snapshot is in a valid state for rollback
	if snapshot.State != "BackedUp" {
		logger.Warn("Snapshot may not be in optimal state for rollback",
			"snapshot_id", snapshotID,
			"current_state", snapshot.State,
		)
	}

	// Perform the rollback using CloudStack RevertSnapshot API
	logger.Info("Initiating CloudStack volume snapshot rollback",
		"snapshot_id", snapshotID,
		"volume_id", snapshot.VolumeID,
	)

	err = osseaClient.RevertVolumeSnapshot(snapshotID)
	if err != nil {
		logger.Error("Failed to revert volume snapshot", "error", err, "snapshot_id", snapshotID)
		return fmt.Errorf("failed to revert volume snapshot: %w", err)
	}

	logger.Info("✅ CloudStack volume snapshot rollback completed successfully",
		"snapshot_id", snapshotID,
		"volume_id", snapshot.VolumeID,
	)
	return nil
}

// DeleteCloudStackVolumeSnapshot deletes CloudStack volume snapshot after successful rollback
func (sco *SnapshotCleanupOperations) DeleteCloudStackVolumeSnapshot(ctx context.Context, snapshotID string) error {
	logger := sco.jobTracker.Logger(ctx)
	logger.Info("🗑️ Deleting CloudStack volume snapshot after rollback", "snapshot_id", snapshotID)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := sco.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for snapshot deletion", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Verify snapshot exists before deletion
	snapshot, err := osseaClient.GetVolumeSnapshot(snapshotID)
	if err != nil {
		logger.Error("Failed to get snapshot details for deletion", "error", err, "snapshot_id", snapshotID)
		return fmt.Errorf("failed to get snapshot details for deletion: %w", err)
	}

	logger.Info("Retrieved snapshot details for deletion",
		"snapshot_id", snapshotID,
		"volume_id", snapshot.VolumeID,
		"state", snapshot.State,
		"size_bytes", snapshot.PhysicalSize,
	)

	// Delete the snapshot using CloudStack API
	logger.Info("Initiating CloudStack volume snapshot deletion",
		"snapshot_id", snapshotID,
		"volume_id", snapshot.VolumeID,
	)

	err = osseaClient.DeleteVolumeSnapshot(snapshotID)
	if err != nil {
		logger.Error("Failed to delete CloudStack volume snapshot", "error", err, "snapshot_id", snapshotID)
		return fmt.Errorf("failed to delete CloudStack volume snapshot: %w", err)
	}

	logger.Info("✅ CloudStack volume snapshot deleted successfully",
		"snapshot_id", snapshotID,
		"volume_id", snapshot.VolumeID,
	)
	return nil
}
