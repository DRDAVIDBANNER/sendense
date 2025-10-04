// Package failover provides CloudStack snapshot operations for enhanced test failover
package failover

import (
	"context"
	"fmt"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
	"github.com/vexxhost/migratekit-oma/ossea"
)

// SnapshotOperations handles CloudStack volume snapshot operations
type SnapshotOperations struct {
	db         *database.Connection
	jobTracker *joblog.Tracker
	helpers    *FailoverHelpers
}

// NewSnapshotOperations creates a new snapshot operations handler
// Note: No longer requires pre-initialized osseaClient - credentials fetched fresh per operation
func NewSnapshotOperations(db *database.Connection, osseaClient *ossea.Client, jobTracker *joblog.Tracker) *SnapshotOperations {
	// Initialize helpers for credential management
	helpers := &FailoverHelpers{
		db:         db,
		jobTracker: jobTracker,
		// osseaClient is NOT cached - will be initialized fresh per operation
	}

	return &SnapshotOperations{
		db:         db,
		jobTracker: jobTracker,
		helpers:    helpers,
	}
}

// CreateCloudStackVolumeSnapshot creates a CloudStack volume snapshot for rollback protection
func (so *SnapshotOperations) CreateCloudStackVolumeSnapshot(ctx context.Context, request *EnhancedTestFailoverRequest) (string, error) {
	logger := so.jobTracker.Logger(ctx)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := so.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for snapshot creation", "error", err.Error())
		return "", fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Get volume UUID for CloudStack snapshot
	volumeUUID, err := so.getVolumeUUIDForVM(ctx, request.VMID)
	if err != nil {
		return "", fmt.Errorf("failed to get volume UUID for VM %s: %w", request.VMID, err)
	}

	logger.Info("📸 Creating CloudStack volume snapshot for rollback protection",
		"vm_id", request.VMID,
		"volume_uuid", volumeUUID,
	)

	// Create snapshot using CloudStack API
	snapshotName := fmt.Sprintf("test-failover-%s-%d", request.VMID, request.Timestamp)

	snapshotReq := &ossea.CreateSnapshotRequest{
		VolumeID:  volumeUUID,
		Name:      snapshotName,
		QuiesceVM: false,
	}

	snapshot, err := osseaClient.CreateVolumeSnapshot(snapshotReq)
	if err != nil {
		return "", fmt.Errorf("failed to create CloudStack volume snapshot: %w", err)
	}

	logger.Info("✅ CloudStack volume snapshot created successfully",
		"snapshot_id", snapshot.ID,
		"snapshot_name", snapshotName,
		"volume_uuid", volumeUUID,
	)

	return snapshot.ID, nil
}

// PerformCloudStackVolumeRollback rolls back a volume to a CloudStack volume snapshot
func (so *SnapshotOperations) PerformCloudStackVolumeRollback(ctx context.Context, vmID, snapshotID string) error {
	logger := so.jobTracker.Logger(ctx)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := so.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for snapshot rollback", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	logger.Info("Performing CloudStack volume snapshot rollback",
		"vm_id", vmID,
		"snapshot_id", snapshotID,
	)

	// Get volume snapshot details
	snapshot, err := osseaClient.GetVolumeSnapshot(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to get snapshot details: %w", err)
	}

	// Revert volume to snapshot
	err = osseaClient.RevertVolumeSnapshot(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to revert volume to snapshot: %w", err)
	}

	logger.Info("✅ CloudStack volume snapshot rollback completed successfully",
		"vm_id", vmID,
		"snapshot_id", snapshotID,
		"volume_id", snapshot.VolumeID,
	)

	return nil
}

// DeleteCloudStackVolumeSnapshot deletes a CloudStack volume snapshot
func (so *SnapshotOperations) DeleteCloudStackVolumeSnapshot(ctx context.Context, snapshotID string) error {
	logger := so.jobTracker.Logger(ctx)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := so.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for snapshot deletion", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	logger.Info("🗑️ Deleting CloudStack volume snapshot",
		"snapshot_id", snapshotID,
	)

	err = osseaClient.DeleteVolumeSnapshot(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to delete CloudStack volume snapshot: %w", err)
	}

	logger.Info("✅ CloudStack volume snapshot deleted successfully",
		"snapshot_id", snapshotID,
	)

	return nil
}

// getVolumeUUIDForVM retrieves the CloudStack volume UUID for a VM
func (so *SnapshotOperations) getVolumeUUIDForVM(ctx context.Context, vmID string) (string, error) {
	logger := so.jobTracker.Logger(ctx)
	logger.Info("🔍 Querying database for volume UUID", "source_vm_id", vmID)

	// Step 1: Find replication job for this source VM
	var replicationJob database.ReplicationJob
	err := (*so.db).GetGormDB().Where("source_vm_id = ?", vmID).
		Order("created_at DESC").
		First(&replicationJob).Error
	if err != nil {
		logger.Error("Failed to find replication job", "error", err, "source_vm_id", vmID)
		return "", fmt.Errorf("no replication job found for VM %s: %w", vmID, err)
	}

	// Step 2: Find VM disks for this job
	var vmDisks []database.VMDisk
	err = (*so.db).GetGormDB().Where("job_id = ?", replicationJob.ID).Find(&vmDisks).Error
	if err != nil {
		logger.Error("Failed to find VM disks", "error", err, "job_id", replicationJob.ID)
		return "", fmt.Errorf("no VM disks found for job %s: %w", replicationJob.ID, err)
	}

	if len(vmDisks) == 0 {
		logger.Error("No VM disks found for replication job", "job_id", replicationJob.ID)
		return "", fmt.Errorf("no VM disks found for job %s", replicationJob.ID)
	}

	// Use the first disk (typically the root disk)
	vmDisk := vmDisks[0]

	// Step 3: Find OSSEA volume
	var osseaVolume database.OSSEAVolume
	err = (*so.db).GetGormDB().Where("id = ?", vmDisk.OSSEAVolumeID).First(&osseaVolume).Error
	if err != nil {
		logger.Error("Failed to find OSSEA volume", "error", err, "ossea_volume_id", vmDisk.OSSEAVolumeID)
		return "", fmt.Errorf("no OSSEA volume found for ID %d: %w", vmDisk.OSSEAVolumeID, err)
	}

	logger.Info("Found volume UUID for VM", 
		"source_vm_id", vmID,
		"volume_uuid", osseaVolume.VolumeID,
		"volume_name", osseaVolume.VolumeName)

	return osseaVolume.VolumeID, nil
}
