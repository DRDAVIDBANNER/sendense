// Package failover provides multi-volume snapshot operations using Volume Daemon API
package failover

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
	"github.com/vexxhost/migratekit-oma/ossea"
)

// MultiVolumeSnapshotService provides multi-volume snapshot management via Volume Daemon API
// This service creates snapshots for ALL volumes in a VM and stores tracking data in device_mappings
type MultiVolumeSnapshotService struct {
	db              *database.Connection
	jobTracker      *joblog.Tracker
	volumeDaemonURL string
	helpers         *FailoverHelpers
}

// VolumeSnapshotResult represents the result of multi-volume snapshot operations
type VolumeSnapshotResult struct {
	VMContextID      string               `json:"vm_context_id"`
	SnapshotsCreated []VolumeSnapshotInfo `json:"snapshots_created"`
	TotalVolumes     int                  `json:"total_volumes"`
	SuccessCount     int                  `json:"success_count"`
	FailureCount     int                  `json:"failure_count"`
}

// VolumeSnapshotInfo represents snapshot information for a volume
type VolumeSnapshotInfo struct {
	VolumeUUID   string    `json:"volume_uuid"`
	VolumeName   string    `json:"volume_name"`
	SnapshotID   string    `json:"snapshot_id"`
	SnapshotName string    `json:"snapshot_name"`
	DiskID       string    `json:"disk_id"`
	DevicePath   string    `json:"device_path"`
	CreatedAt    time.Time `json:"created_at"`
}

// TrackSnapshotRequest represents a Volume Daemon API request to track snapshots
type TrackSnapshotRequest struct {
	VolumeUUID     string `json:"volume_uuid"`
	VMContextID    string `json:"vm_context_id"`
	SnapshotID     string `json:"snapshot_id"`
	SnapshotName   string `json:"snapshot_name,omitempty"`
	DiskID         string `json:"disk_id,omitempty"`
	SnapshotStatus string `json:"snapshot_status,omitempty"`
}

// VolumeSnapshot represents Volume Daemon snapshot info response
type VolumeSnapshot struct {
	VolumeUUID        string     `json:"volume_uuid"`
	VMContextID       string     `json:"vm_context_id"`
	VolumeName        string     `json:"volume_name"`
	DevicePath        string     `json:"device_path"`
	OperationMode     string     `json:"operation_mode"`
	SnapshotID        *string    `json:"snapshot_id,omitempty"`
	SnapshotCreatedAt *time.Time `json:"snapshot_created_at,omitempty"`
	SnapshotStatus    string     `json:"snapshot_status"`
}

// NewMultiVolumeSnapshotService creates a new multi-volume snapshot service
// Note: No longer requires pre-initialized osseaClient - credentials fetched fresh per operation
func NewMultiVolumeSnapshotService(db *database.Connection, osseaClient *ossea.Client, jobTracker *joblog.Tracker) *MultiVolumeSnapshotService {
	// Initialize helpers for credential management
	helpers := &FailoverHelpers{
		db:         db,
		jobTracker: jobTracker,
		// osseaClient is NOT cached - will be initialized fresh per operation
	}

	return &MultiVolumeSnapshotService{
		db:              db,
		jobTracker:      jobTracker,
		volumeDaemonURL: "http://localhost:8090",
		helpers:         helpers,
	}
}

// CreateAllVolumeSnapshots creates CloudStack snapshots for ALL volumes and tracks them in device_mappings
func (mvss *MultiVolumeSnapshotService) CreateAllVolumeSnapshots(
	ctx context.Context,
	vmContextID string,
) (*VolumeSnapshotResult, error) {
	logger := mvss.jobTracker.Logger(ctx)
	logger.Info("📸 Creating snapshots for ALL volumes in VM",
		"vm_context_id", vmContextID)

	// Step 1: Get all volumes for this VM context from ossea_volumes
	var volumes []database.OSSEAVolume
	err := (*mvss.db).GetGormDB().Where("vm_context_id = ?", vmContextID).Find(&volumes).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get volumes for VM context %s: %w", vmContextID, err)
	}

	if len(volumes) == 0 {
		return nil, fmt.Errorf("no volumes found for VM context %s", vmContextID)
	}

	logger.Info("🔍 Found volumes for snapshot creation",
		"vm_context_id", vmContextID,
		"volume_count", len(volumes))

	// Step 2: Get disk ID mapping for better snapshot naming
	diskIDMap, err := mvss.getVolumeToDiscIDMapping(ctx, vmContextID)
	if err != nil {
		logger.Warn("Could not get disk ID mapping, using basic naming", "error", err)
		diskIDMap = make(map[string]string) // Empty map as fallback
	}

	result := &VolumeSnapshotResult{
		VMContextID:  vmContextID,
		TotalVolumes: len(volumes),
	}

	// Step 3: Create snapshot for each volume and track in Volume Daemon
	for _, volume := range volumes {
		snapshotInfo, err := mvss.createAndTrackSingleSnapshot(ctx, volume, diskIDMap, vmContextID)
		if err != nil {
			logger.Error("❌ Failed to create and track snapshot for volume",
				"error", err,
				"volume_uuid", volume.VolumeID,
				"volume_name", volume.VolumeName)
			result.FailureCount++
			continue
		}

		result.SnapshotsCreated = append(result.SnapshotsCreated, *snapshotInfo)
		result.SuccessCount++

		logger.Info("✅ Volume snapshot created and tracked successfully",
			"volume_uuid", snapshotInfo.VolumeUUID,
			"snapshot_id", snapshotInfo.SnapshotID,
			"disk_id", snapshotInfo.DiskID,
			"device_path", snapshotInfo.DevicePath)
	}

	// Validate results
	if result.SuccessCount == 0 {
		return result, fmt.Errorf("failed to create any snapshots for VM context %s", vmContextID)
	}

	logger.Info("🎉 Multi-volume snapshot creation completed",
		"vm_context_id", vmContextID,
		"successful_snapshots", result.SuccessCount,
		"failed_snapshots", result.FailureCount,
		"total_volumes", result.TotalVolumes)

	return result, nil
}

// createAndTrackSingleSnapshot creates a CloudStack snapshot and tracks it via Volume Daemon API
func (mvss *MultiVolumeSnapshotService) createAndTrackSingleSnapshot(
	ctx context.Context,
	volume database.OSSEAVolume,
	diskIDMap map[string]string,
	vmContextID string,
) (*VolumeSnapshotInfo, error) {
	logger := mvss.jobTracker.Logger(ctx)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := mvss.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for multi-volume snapshot creation", "error", err.Error())
		return nil, fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Generate snapshot name with disk ID correlation
	diskID := "unknown"
	if mappedDiskID, exists := diskIDMap[volume.VolumeID]; exists {
		diskID = mappedDiskID
	}

	timestamp := time.Now().Unix()
	snapshotName := fmt.Sprintf("multi-test-failover-%s-%d", diskID, timestamp)

	logger.Info("📸 Creating CloudStack volume snapshot",
		"volume_uuid", volume.VolumeID,
		"volume_name", volume.VolumeName,
		"snapshot_name", snapshotName,
		"disk_id", diskID)

	// Step 1: Create CloudStack volume snapshot
	snapshotReq := &ossea.CreateSnapshotRequest{
		VolumeID:  volume.VolumeID,
		Name:      snapshotName,
		QuiesceVM: false, // Don't quiesce during test failover
	}

	snapshot, err := osseaClient.CreateVolumeSnapshot(snapshotReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudStack volume snapshot for %s: %w",
			volume.VolumeID, err)
	}

	logger.Info("✅ CloudStack snapshot created, now tracking in Volume Daemon",
		"volume_uuid", volume.VolumeID,
		"snapshot_id", snapshot.ID)

	// Step 2: Track snapshot in ossea_volumes table (stable storage)
	err = mvss.updateOSSEAVolumeSnapshot(ctx, volume.VolumeID, snapshot.ID, snapshotName, "ready")
	if err != nil {
		logger.Error("Failed to track snapshot in ossea_volumes, but keeping snapshot for manual cleanup",
			"error", err,
			"snapshot_id", snapshot.ID,
			"volume_uuid", volume.VolumeID)
		// DON'T delete the snapshot - let it exist for manual cleanup or retry
		return nil, fmt.Errorf("failed to track snapshot in ossea_volumes: %w", err)
	}

	// Step 3: Get device path from Volume Daemon
	devicePath, err := mvss.getVolumeDevicePath(ctx, volume.VolumeID)
	if err != nil {
		logger.Warn("Could not get device path", "error", err)
		devicePath = "unknown"
	}

	logger.Info("✅ Volume snapshot created and tracked in device_mappings",
		"volume_uuid", volume.VolumeID,
		"snapshot_id", snapshot.ID,
		"device_path", devicePath)

	return &VolumeSnapshotInfo{
		VolumeUUID:   volume.VolumeID,
		VolumeName:   volume.VolumeName,
		SnapshotID:   snapshot.ID,
		SnapshotName: snapshotName,
		DiskID:       diskID,
		DevicePath:   devicePath,
		CreatedAt:    time.Now(),
	}, nil
}

// RollbackAllVolumeSnapshots rolls back ALL volumes to their tracked snapshots
func (mvss *MultiVolumeSnapshotService) RollbackAllVolumeSnapshots(
	ctx context.Context,
	vmContextID string,
) error {
	logger := mvss.jobTracker.Logger(ctx)
	logger.Info("⏪ Rolling back ALL volumes to snapshots",
		"vm_context_id", vmContextID)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := mvss.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for multi-volume rollback", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Step 1: Get all snapshot information from Volume Daemon
	snapshots, err := mvss.getVMSnapshots(ctx, vmContextID)
	if err != nil {
		return fmt.Errorf("failed to get VM snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		logger.Warn("No snapshots found for rollback", "vm_context_id", vmContextID)
		return nil
	}

	logger.Info("🔍 Found snapshots for rollback",
		"vm_context_id", vmContextID,
		"snapshot_count", len(snapshots))

	successCount := 0
	errorCount := 0

	// Step 2: Rollback each volume to its snapshot
	for _, snapshot := range snapshots {
		if snapshot.SnapshotID == nil || *snapshot.SnapshotID == "" {
			logger.Warn("Skipping volume with no snapshot ID",
				"volume_uuid", snapshot.VolumeUUID)
			continue
		}

		logger.Info("⏪ Rolling back volume to snapshot",
			"volume_uuid", snapshot.VolumeUUID,
			"snapshot_id", *snapshot.SnapshotID,
			"device_path", snapshot.DevicePath)

		err = osseaClient.RevertVolumeSnapshot(*snapshot.SnapshotID)
		if err != nil {
			logger.Error("❌ Failed to rollback volume snapshot",
				"error", err,
				"volume_uuid", snapshot.VolumeUUID,
				"snapshot_id", *snapshot.SnapshotID)
			errorCount++
			continue
		}

		// Update snapshot status to indicate rollback completed
		err = mvss.updateSnapshotStatus(ctx, snapshot.VolumeUUID, "rollback_complete")
		if err != nil {
			logger.Error("Failed to update rollback status", "error", err)
		}

		successCount++
		logger.Info("✅ Volume rolled back successfully",
			"volume_uuid", snapshot.VolumeUUID,
			"snapshot_id", *snapshot.SnapshotID)
	}

	if errorCount > 0 {
		return fmt.Errorf("failed to rollback %d volumes (succeeded: %d)", errorCount, successCount)
	}

	logger.Info("🎉 Multi-volume rollback completed successfully",
		"vm_context_id", vmContextID,
		"volumes_rolled_back", successCount)

	return nil
}

// CleanupAllVolumeSnapshots deletes all snapshots and clears tracking data
func (mvss *MultiVolumeSnapshotService) CleanupAllVolumeSnapshots(
	ctx context.Context,
	vmContextID string,
) error {
	logger := mvss.jobTracker.Logger(ctx)
	logger.Info("🧹 Cleaning up ALL volume snapshots",
		"vm_context_id", vmContextID)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := mvss.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for multi-volume cleanup", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Step 1: Get all snapshot information directly from database (bypasses Volume Daemon API issues)
	snapshots, err := mvss.getVMSnapshotsFromDatabase(ctx, vmContextID)
	if err != nil {
		return fmt.Errorf("failed to get VM snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		logger.Info("No snapshots found for cleanup", "vm_context_id", vmContextID)
		return nil
	}

	// Filter snapshots that actually have snapshot IDs (ignore empty tracking records)
	var snapshotsToDelete []VolumeSnapshot
	for _, snapshot := range snapshots {
		if snapshot.SnapshotID != nil && *snapshot.SnapshotID != "" {
			snapshotsToDelete = append(snapshotsToDelete, snapshot)
		}
	}

	if len(snapshotsToDelete) == 0 {
		logger.Info("No actual snapshots to delete (only empty tracking records)",
			"vm_context_id", vmContextID,
			"tracking_records", len(snapshots))
		return nil
	}

	logger.Info("🔍 Found snapshots for cleanup",
		"vm_context_id", vmContextID,
		"snapshot_count", len(snapshotsToDelete))

	// Step 2: Revert each volume to its snapshot (CRITICAL: Undo test failover changes)
	revertedCount := 0
	for _, snapshot := range snapshotsToDelete {
		// Safety checks to prevent nil pointer panics
		if snapshot.SnapshotID == nil {
			logger.Error("❌ Snapshot ID is nil, skipping revert",
				"volume_uuid", snapshot.VolumeUUID)
			continue
		}

		logger.Info("⏪ Reverting volume to snapshot state",
			"volume_uuid", snapshot.VolumeUUID,
			"snapshot_id", *snapshot.SnapshotID)

		err = osseaClient.RevertVolumeSnapshot(*snapshot.SnapshotID)
		if err != nil {
			logger.Error("❌ Failed to revert volume to snapshot",
				"error", err,
				"snapshot_id", *snapshot.SnapshotID,
				"volume_uuid", snapshot.VolumeUUID)
			// Continue with other snapshots even if one fails
		} else {
			revertedCount++
			logger.Info("✅ Volume reverted to snapshot successfully",
				"snapshot_id", *snapshot.SnapshotID,
				"volume_uuid", snapshot.VolumeUUID)
		}
	}

	logger.Info("📊 Volume revert summary",
		"vm_context_id", vmContextID,
		"volumes_reverted", revertedCount,
		"volumes_attempted", len(snapshotsToDelete))

	// Step 3: Delete each CloudStack snapshot after revert
	deletedCount := 0
	for _, snapshot := range snapshotsToDelete {
		// Safety checks to prevent nil pointer panics
		if snapshot.SnapshotID == nil {
			logger.Error("❌ Snapshot ID is nil, skipping deletion",
				"volume_uuid", snapshot.VolumeUUID)
			continue
		}

		logger.Info("🗑️ Deleting CloudStack volume snapshot",
			"volume_uuid", snapshot.VolumeUUID,
			"snapshot_id", *snapshot.SnapshotID)

		err = osseaClient.DeleteVolumeSnapshot(*snapshot.SnapshotID)
		if err != nil {
			logger.Error("❌ Failed to delete snapshot",
				"error", err,
				"snapshot_id", *snapshot.SnapshotID,
				"volume_uuid", snapshot.VolumeUUID)
			// Continue with other snapshots even if one fails
		} else {
			deletedCount++
			logger.Info("✅ CloudStack snapshot deleted successfully",
				"snapshot_id", *snapshot.SnapshotID,
				"volume_uuid", snapshot.VolumeUUID)
		}
	}

	logger.Info("📊 CloudStack snapshot deletion summary",
		"vm_context_id", vmContextID,
		"snapshots_deleted", deletedCount,
		"snapshots_attempted", len(snapshotsToDelete))

	// Step 4: Clear tracking data ONLY if we actually processed snapshots
	if deletedCount > 0 {
		err = mvss.clearOSSEAVolumeSnapshots(ctx, vmContextID)
		if err != nil {
			logger.Error("Failed to clear snapshot tracking data from ossea_volumes", "error", err)
			// Don't fail the operation - tracking cleanup is non-critical
		} else {
			logger.Info("✅ Snapshot tracking data cleared from ossea_volumes successfully",
				"vm_context_id", vmContextID)
		}
	} else {
		logger.Info("ℹ️ Preserving snapshot tracking data - no CloudStack snapshots were deleted",
			"vm_context_id", vmContextID)
	}

	logger.Info("🎉 Multi-volume snapshot cleanup completed",
		"vm_context_id", vmContextID)

	return nil
}

// Helper methods for Volume Daemon API communication

// getVMSnapshotsFromDatabase retrieves snapshot info directly from ossea_volumes table (stable storage)
func (mvss *MultiVolumeSnapshotService) getVMSnapshotsFromDatabase(ctx context.Context, vmContextID string) ([]VolumeSnapshot, error) {
	logger := mvss.jobTracker.Logger(ctx)
	logger.Info("🔍 Getting VM snapshots from ossea_volumes table (stable storage)",
		"vm_context_id", vmContextID)

	// Query ossea_volumes for snapshot information (survives device mapping changes)
	query := `
		SELECT ov.volume_id, ov.vm_context_id, ov.volume_name,
		       ov.snapshot_id, ov.snapshot_created_at, ov.snapshot_status,
		       COALESCE(dm.device_path, 'unknown') as device_path,
		       COALESCE(dm.operation_mode, 'oma') as operation_mode
		FROM ossea_volumes ov
		LEFT JOIN device_mappings dm ON ov.volume_id = dm.volume_uuid
		WHERE ov.vm_context_id = ? AND ov.snapshot_id IS NOT NULL AND ov.snapshot_id != ''
	`

	rows, err := (*mvss.db).GetGormDB().Raw(query, vmContextID).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to query ossea_volumes: %w", err)
	}
	defer rows.Close()

	var snapshots []VolumeSnapshot
	for rows.Next() {
		var snapshot VolumeSnapshot
		var snapshotID, snapshotCreatedAtStr string

		err := rows.Scan(
			&snapshot.VolumeUUID,
			&snapshot.VMContextID,
			&snapshot.VolumeName,
			&snapshotID,
			&snapshotCreatedAtStr,
			&snapshot.SnapshotStatus,
			&snapshot.DevicePath,
			&snapshot.OperationMode,
		)
		if err != nil {
			logger.Error("Failed to scan snapshot row", "error", err)
			continue
		}

		// Set snapshot ID (convert string to *string)
		snapshot.SnapshotID = &snapshotID

		// Parse timestamp if present
		if snapshotCreatedAtStr != "" {
			if createdAt, parseErr := time.Parse("2006-01-02 15:04:05", snapshotCreatedAtStr); parseErr == nil {
				snapshot.SnapshotCreatedAt = &createdAt
			}
		}

		snapshots = append(snapshots, snapshot)
	}

	logger.Info("✅ Retrieved snapshots from ossea_volumes table",
		"vm_context_id", vmContextID,
		"snapshot_count", len(snapshots))

	return snapshots, nil
}

// GetVMSnapshots is a public method to retrieve snapshot information (used by cleanup coordination)
func (mvss *MultiVolumeSnapshotService) GetVMSnapshots(ctx context.Context, vmContextID string) ([]VolumeSnapshot, error) {
	return mvss.getVMSnapshotsFromDatabase(ctx, vmContextID)
}

// getVMSnapshots retrieves all snapshot info for a VM from Volume Daemon
func (mvss *MultiVolumeSnapshotService) getVMSnapshots(ctx context.Context, vmContextID string) ([]VolumeSnapshot, error) {
	url := fmt.Sprintf("%s/api/v1/snapshots/vm/%s", mvss.volumeDaemonURL, vmContextID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call Volume Daemon API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Volume Daemon API returned status %d", resp.StatusCode)
	}

	var response struct {
		Snapshots []VolumeSnapshot `json:"snapshots"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Volume Daemon response: %w", err)
	}

	return response.Snapshots, nil
}

// updateOSSEAVolumeSnapshot updates snapshot tracking in ossea_volumes table
func (mvss *MultiVolumeSnapshotService) updateOSSEAVolumeSnapshot(ctx context.Context, volumeID, snapshotID, snapshotName, status string) error {
	logger := mvss.jobTracker.Logger(ctx)

	now := time.Now()
	result := (*mvss.db).GetGormDB().Model(&database.OSSEAVolume{}).
		Where("volume_id = ?", volumeID).
		Updates(map[string]interface{}{
			"snapshot_id":         snapshotID,
			"snapshot_created_at": now,
			"snapshot_status":     status,
			"updated_at":          now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update ossea_volumes with snapshot info: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no ossea_volumes record found for volume_id %s", volumeID)
	}

	logger.Info("✅ Snapshot tracking updated in ossea_volumes",
		"volume_id", volumeID,
		"snapshot_id", snapshotID,
		"status", status)

	return nil
}

// clearOSSEAVolumeSnapshots clears snapshot tracking from ossea_volumes table
func (mvss *MultiVolumeSnapshotService) clearOSSEAVolumeSnapshots(ctx context.Context, vmContextID string) error {
	logger := mvss.jobTracker.Logger(ctx)

	result := (*mvss.db).GetGormDB().Model(&database.OSSEAVolume{}).
		Where("vm_context_id = ?", vmContextID).
		Updates(map[string]interface{}{
			"snapshot_id":         nil,
			"snapshot_created_at": nil,
			"snapshot_status":     "none",
			"updated_at":          time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to clear snapshot tracking from ossea_volumes: %w", result.Error)
	}

	logger.Info("✅ Snapshot tracking cleared from ossea_volumes",
		"vm_context_id", vmContextID,
		"volumes_updated", result.RowsAffected)

	return nil
}

// clearVMSnapshotTracking clears all snapshot tracking for a VM via Volume Daemon API
func (mvss *MultiVolumeSnapshotService) clearVMSnapshotTracking(ctx context.Context, vmContextID string) error {
	url := fmt.Sprintf("%s/api/v1/snapshots/vm/%s", mvss.volumeDaemonURL, vmContextID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Volume Daemon API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Volume Daemon API returned status %d", resp.StatusCode)
	}

	return nil
}

// updateSnapshotStatus updates the snapshot status for a volume via Volume Daemon API
func (mvss *MultiVolumeSnapshotService) updateSnapshotStatus(ctx context.Context, volumeUUID, status string) error {
	url := fmt.Sprintf("%s/api/v1/snapshots/%s", mvss.volumeDaemonURL, volumeUUID)

	updateReq := map[string]string{
		"snapshot_status": status,
	}

	return mvss.callVolumeDaemonAPI(ctx, "PUT", url, updateReq)
}

// getVolumeDevicePath gets the device path for a volume from Volume Daemon
func (mvss *MultiVolumeSnapshotService) getVolumeDevicePath(ctx context.Context, volumeUUID string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/volumes/%s/device", mvss.volumeDaemonURL, volumeUUID)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to call Volume Daemon API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Volume Daemon API returned status %d", resp.StatusCode)
	}

	var mapping struct {
		DevicePath string `json:"device_path"`
	}

	err = json.NewDecoder(resp.Body).Decode(&mapping)
	if err != nil {
		return "", fmt.Errorf("failed to decode Volume Daemon response: %w", err)
	}

	return mapping.DevicePath, nil
}

// callVolumeDaemonAPI makes a generic API call to Volume Daemon
func (mvss *MultiVolumeSnapshotService) callVolumeDaemonAPI(ctx context.Context, method, url string, data interface{}) error {
	var reqBody *bytes.Buffer
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal request data: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Volume Daemon API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Volume Daemon API returned status %d", resp.StatusCode)
	}

	return nil
}

// getVolumeToDiscIDMapping creates a mapping from OSSEA volume UUID to VMware disk ID
func (mvss *MultiVolumeSnapshotService) getVolumeToDiscIDMapping(
	ctx context.Context,
	vmContextID string,
) (map[string]string, error) {
	logger := mvss.jobTracker.Logger(ctx)

	// Query vm_disks and ossea_volumes to correlate volume UUIDs with disk IDs
	type VolumeMapping struct {
		VolumeUUID string `json:"volume_uuid"`
		DiskID     string `json:"disk_id"`
	}

	var mappings []VolumeMapping
	err := (*mvss.db).GetGormDB().Raw(`
		SELECT ov.volume_id as volume_uuid, vd.disk_id
		FROM vm_disks vd
		JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
		WHERE vd.vm_context_id = ?
	`, vmContextID).Scan(&mappings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get volume-to-disk mapping: %w", err)
	}

	volumeToDiskMap := make(map[string]string)
	for _, mapping := range mappings {
		volumeToDiskMap[mapping.VolumeUUID] = mapping.DiskID
		logger.Debug("Volume-to-disk mapping",
			"volume_uuid", mapping.VolumeUUID,
			"disk_id", mapping.DiskID)
	}

	logger.Info("✅ Volume-to-disk mapping created",
		"vm_context_id", vmContextID,
		"mappings_count", len(volumeToDiskMap))

	return volumeToDiskMap, nil
}
