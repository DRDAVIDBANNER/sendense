// Package ossea provides volume snapshot operations for OSSEA (CloudStack) failover functionality
package ossea

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// VolumeSnapshot represents an OSSEA volume snapshot
type VolumeSnapshot struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	VolumeID     string `json:"volumeid"`
	VolumeName   string `json:"volumename"`
	VolumeType   string `json:"volumetype"`   // ROOT, DATADISK
	SnapshotType string `json:"snapshottype"` // MANUAL, RECURRING, etc.
	State        string `json:"state"`        // Created, Creating, BackedUp, etc.
	IntervalType string `json:"intervaltype"` // MANUAL, HOURLY, DAILY, etc.
	Account      string `json:"account"`
	DomainID     string `json:"domainid"`
	Domain       string `json:"domain"`
	Created      string `json:"created"`
	ZoneID       string `json:"zoneid"`
	ZoneName     string `json:"zonename"`

	// Size information
	PhysicalSize int64 `json:"physicalsize"` // Physical size in bytes
	Size         int64 `json:"size"`         // Logical size in bytes

	// Snapshot metadata
	Tags []SnapshotTag `json:"tags,omitempty"`
}

// SnapshotTag represents snapshot metadata tags
type SnapshotTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CreateSnapshotRequest represents parameters for creating a snapshot
type CreateSnapshotRequest struct {
	VolumeID  string            `json:"volumeid" binding:"required"`
	Name      string            `json:"name,omitempty"`
	QuiesceVM bool              `json:"quiescevm,omitempty"` // Quiesce VM before snapshot
	Tags      map[string]string `json:"tags,omitempty"`
}

// CreateVolumeSnapshot creates a snapshot of the specified volume
func (c *Client) CreateVolumeSnapshot(req *CreateSnapshotRequest) (*VolumeSnapshot, error) {
	log.WithFields(log.Fields{
		"volume_id":  req.VolumeID,
		"name":       req.Name,
		"quiesce_vm": req.QuiesceVM,
	}).Info("📸 Creating OSSEA volume snapshot")

	// Build CloudStack createSnapshot parameters
	params := c.cs.Snapshot.NewCreateSnapshotParams(req.VolumeID)

	log.WithFields(log.Fields{
		"volume_id":        req.VolumeID,
		"cloudstack_param": params,
	}).Info("🔍 CloudStack snapshot parameters prepared")

	// Set snapshot name if provided
	if req.Name != "" {
		params.SetName(req.Name)
	}

	// Set VM quiescing if requested
	if req.QuiesceVM {
		params.SetQuiescevm(req.QuiesceVM)
	}

	// Create the snapshot
	resp, err := c.cs.Snapshot.CreateSnapshot(params)
	if err != nil {
		return nil, fmt.Errorf("failed to submit volume snapshot creation: %w", err)
	}

	log.WithFields(log.Fields{
		"volume_id":     req.VolumeID,
		"snapshot_name": req.Name,
		"job_id":        resp.JobID,
	}).Info("✅ Volume snapshot creation initiated successfully")

	snapshot := &VolumeSnapshot{
		ID:           resp.Id,
		Name:         resp.Name,
		VolumeID:     resp.Volumeid,
		VolumeName:   resp.Volumename,
		VolumeType:   resp.Volumetype,
		SnapshotType: resp.Snapshottype,
		State:        resp.State,
		IntervalType: resp.Intervaltype,
		Account:      resp.Account,
		DomainID:     resp.Domainid,
		Domain:       resp.Domain,
		Created:      resp.Created,
		ZoneID:       resp.Zoneid,
		ZoneName:     resp.Zoneid, // Use ZoneID as ZoneName not always available
		PhysicalSize: resp.Physicalsize,
		Size:         resp.Physicalsize, // Use physical size as logical size fallback
	}

	log.WithFields(log.Fields{
		"snapshot_id":   snapshot.ID,
		"snapshot_name": snapshot.Name,
		"volume_id":     snapshot.VolumeID,
		"state":         snapshot.State,
	}).Info("✅ OSSEA volume snapshot creation initiated")

	return snapshot, nil
}

// GetVolumeSnapshot retrieves snapshot information by ID
func (c *Client) GetVolumeSnapshot(snapshotID string) (*VolumeSnapshot, error) {
	log.WithField("snapshot_id", snapshotID).Debug("🔍 Retrieving OSSEA snapshot details")

	params := c.cs.Snapshot.NewListSnapshotsParams()
	params.SetId(snapshotID)

	resp, err := c.cs.Snapshot.ListSnapshots(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	if resp.Count == 0 {
		return nil, fmt.Errorf("snapshot with ID %s not found", snapshotID)
	}

	snapResp := resp.Snapshots[0]
	snapshot := &VolumeSnapshot{
		ID:           snapResp.Id,
		Name:         snapResp.Name,
		VolumeID:     snapResp.Volumeid,
		VolumeName:   snapResp.Volumename,
		VolumeType:   snapResp.Volumetype,
		SnapshotType: snapResp.Snapshottype,
		State:        snapResp.State,
		IntervalType: snapResp.Intervaltype,
		Account:      snapResp.Account,
		DomainID:     snapResp.Domainid,
		Domain:       snapResp.Domain,
		Created:      snapResp.Created,
		ZoneID:       snapResp.Zoneid,
		ZoneName:     snapResp.Zoneid, // Use ZoneID if ZoneName not available
		PhysicalSize: snapResp.Physicalsize,
		Size:         snapResp.Physicalsize, // Use physical size as logical size fallback
	}

	log.WithFields(log.Fields{
		"snapshot_id":   snapshot.ID,
		"snapshot_name": snapshot.Name,
		"volume_id":     snapshot.VolumeID,
		"state":         snapshot.State,
	}).Debug("✅ Retrieved OSSEA snapshot details")

	return snapshot, nil
}

// ListVolumeSnapshots lists all snapshots for a specific volume
func (c *Client) ListVolumeSnapshots(volumeID string) ([]VolumeSnapshot, error) {
	log.WithField("volume_id", volumeID).Debug("📋 Listing OSSEA volume snapshots")

	params := c.cs.Snapshot.NewListSnapshotsParams()
	params.SetVolumeid(volumeID)

	resp, err := c.cs.Snapshot.ListSnapshots(params)
	if err != nil {
		return nil, fmt.Errorf("failed to list volume snapshots: %w", err)
	}

	snapshots := make([]VolumeSnapshot, len(resp.Snapshots))
	for i, snap := range resp.Snapshots {
		snapshots[i] = VolumeSnapshot{
			ID:           snap.Id,
			Name:         snap.Name,
			VolumeID:     snap.Volumeid,
			VolumeName:   snap.Volumename,
			VolumeType:   snap.Volumetype,
			SnapshotType: snap.Snapshottype,
			State:        snap.State,
			IntervalType: snap.Intervaltype,
			Account:      snap.Account,
			DomainID:     snap.Domainid,
			Domain:       snap.Domain,
			Created:      snap.Created,
			ZoneID:       snap.Zoneid,
			ZoneName:     snap.Zoneid, // Use ZoneID if ZoneName not available
			PhysicalSize: snap.Physicalsize,
			Size:         snap.Physicalsize, // Use physical size as logical size fallback
		}
	}

	log.WithFields(log.Fields{
		"volume_id":      volumeID,
		"snapshot_count": len(snapshots),
	}).Debug("✅ Listed OSSEA volume snapshots")

	return snapshots, nil
}

// DeleteVolumeSnapshot deletes a volume snapshot
func (c *Client) DeleteVolumeSnapshot(snapshotID string) error {
	log.WithField("snapshot_id", snapshotID).Info("🗑️ Deleting OSSEA volume snapshot")

	params := c.cs.Snapshot.NewDeleteSnapshotParams(snapshotID)

	resp, err := c.cs.Snapshot.DeleteSnapshot(params)
	if err != nil {
		return fmt.Errorf("failed to submit snapshot deletion: %w", err)
	}

	log.WithFields(log.Fields{
		"snapshot_id": snapshotID,
		"job_id":      resp.JobID,
	}).Info("✅ Volume snapshot deletion submitted successfully, waiting for completion...")

	// Wait for async job completion
	err = c.WaitForAsyncJob(resp.JobID, 180*time.Second)
	if err != nil {
		return fmt.Errorf("volume snapshot deletion async job failed: %w", err)
	}

	log.WithField("snapshot_id", snapshotID).Info("✅ OSSEA volume snapshot deleted successfully")
	return nil
}

// RevertVolumeSnapshot reverts a volume to a previous snapshot state and waits for completion
func (c *Client) RevertVolumeSnapshot(snapshotID string) error {
	log.WithField("snapshot_id", snapshotID).Info("⏪ Reverting OSSEA volume to snapshot")

	params := c.cs.Snapshot.NewRevertSnapshotParams(snapshotID)

	resp, err := c.cs.Snapshot.RevertSnapshot(params)
	if err != nil {
		return fmt.Errorf("failed to submit volume snapshot revert: %w", err)
	}

	log.WithFields(log.Fields{
		"snapshot_id": snapshotID,
		"job_id":      resp.JobID,
	}).Info("✅ Volume snapshot revert submitted successfully, waiting for completion...")

	// Wait for async job completion
	err = c.WaitForAsyncJob(resp.JobID, 300*time.Second)
	if err != nil {
		return fmt.Errorf("volume snapshot revert async job failed: %w", err)
	}

	log.WithField("snapshot_id", snapshotID).Info("✅ OSSEA volume reverted to snapshot successfully")
	return nil
}

// WaitForSnapshotState waits for a snapshot to reach the specified state
func (c *Client) WaitForSnapshotState(snapshotID string, targetState string, timeout time.Duration) error {
	log.WithFields(log.Fields{
		"snapshot_id":  snapshotID,
		"target_state": targetState,
		"timeout":      timeout,
	}).Info("⏳ Waiting for OSSEA snapshot state change")

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		snapshot, err := c.GetVolumeSnapshot(snapshotID)
		if err != nil {
			return fmt.Errorf("failed to check snapshot state: %w", err)
		}

		if strings.EqualFold(snapshot.State, targetState) {
			log.WithFields(log.Fields{
				"snapshot_id":    snapshotID,
				"snapshot_state": snapshot.State,
			}).Info("✅ OSSEA snapshot reached target state")
			return nil
		}

		log.WithFields(log.Fields{
			"snapshot_id":   snapshotID,
			"current_state": snapshot.State,
			"target_state":  targetState,
		}).Debug("⏳ Snapshot state transition in progress")

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for snapshot %s to reach state %s", snapshotID, targetState)
}

// CreateVolumeFromSnapshot creates a new volume from an existing snapshot
func (c *Client) CreateVolumeFromSnapshot(snapshotID, name string, sizeGB int) (*Volume, error) {
	log.WithFields(log.Fields{
		"snapshot_id": snapshotID,
		"volume_name": name,
		"size_gb":     sizeGB,
	}).Info("💾 Creating OSSEA volume from snapshot")

	// Get snapshot details to determine zone
	snapshot, err := c.GetVolumeSnapshot(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot details: %w", err)
	}

	// Build CloudStack createVolume parameters
	params := c.cs.Volume.NewCreateVolumeParams()
	params.SetName(name)
	params.SetSnapshotid(snapshotID)
	params.SetZoneid(snapshot.ZoneID)

	if sizeGB > 0 {
		params.SetSize(int64(sizeGB))
	}

	resp, err := c.cs.Volume.CreateVolume(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume from snapshot: %w", err)
	}

	volume := &Volume{
		ID:               resp.Id,
		Name:             resp.Name,
		Size:             resp.Size,
		SizeGB:           int(resp.Size / (1024 * 1024 * 1024)), // Convert bytes to GB
		Type:             resp.Type,
		State:            resp.State,
		ZoneID:           resp.Zoneid,
		ZoneName:         resp.Zonename,
		DiskOfferingID:   resp.Diskofferingid,
		DiskOfferingName: resp.Diskofferingname,
		VirtualMachineID: resp.Virtualmachineid,
		DeviceID:         int(resp.Deviceid),
		Created:          resp.Created,
		Attached:         resp.Attached,
		IsExtractable:    resp.Isextractable,
		StorageType:      resp.Storagetype,
		ProvisioningType: resp.Provisioningtype,
	}

	log.WithFields(log.Fields{
		"volume_id":   volume.ID,
		"volume_name": volume.Name,
		"snapshot_id": snapshotID,
		"state":       volume.State,
	}).Info("✅ OSSEA volume created from snapshot")

	return volume, nil
}

// CleanupFailoverSnapshots removes snapshots created during failover operations
func (c *Client) CleanupFailoverSnapshots(volumeID string, maxAge time.Duration) error {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"max_age":   maxAge,
	}).Info("🧹 Cleaning up old failover snapshots")

	snapshots, err := c.ListVolumeSnapshots(volumeID)
	if err != nil {
		return fmt.Errorf("failed to list snapshots for cleanup: %w", err)
	}

	cutoffTime := time.Now().Add(-maxAge)
	deletedCount := 0

	for _, snapshot := range snapshots {
		// Check if snapshot is a failover snapshot (by name pattern or tags)
		if strings.Contains(snapshot.Name, "failover") || strings.Contains(snapshot.Name, "test") {
			// Parse creation time
			createdTime, err := time.Parse("2006-01-02T15:04:05-0700", snapshot.Created)
			if err != nil {
				log.WithError(err).WithField("snapshot_id", snapshot.ID).Warn("Failed to parse snapshot creation time")
				continue
			}

			// Delete if older than cutoff
			if createdTime.Before(cutoffTime) {
				if err := c.DeleteVolumeSnapshot(snapshot.ID); err != nil {
					log.WithError(err).WithField("snapshot_id", snapshot.ID).Error("Failed to delete old failover snapshot")
					continue
				}
				deletedCount++
			}
		}
	}

	log.WithFields(log.Fields{
		"volume_id":       volumeID,
		"deleted_count":   deletedCount,
		"total_snapshots": len(snapshots),
	}).Info("✅ Failover snapshot cleanup completed")

	return nil
}
