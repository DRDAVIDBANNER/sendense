// Package workflows provides orchestration for VMware to OSSEA migrations
// Following project rules: modular design, clean interfaces, comprehensive error handling
package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/common"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/models"
	"github.com/vexxhost/migratekit-oma/nbd"
	"github.com/vexxhost/migratekit-oma/ossea"
	"github.com/vexxhost/migratekit-oma/services"
	"github.com/vexxhost/migratekit-oma/volume"
)

// VMAProgressPoller interface for VMA progress polling service
type VMAProgressPoller interface {
	StartPolling(jobID string) error
	StopPolling(jobID string) error
	GetPollingStatus() map[string]interface{}
}

// MigrationEngine orchestrates complete VMware to OSSEA migrations
type MigrationEngine struct {
	// Repository dependencies
	db                 database.Connection
	osseaConfigRepo    *database.OSSEAConfigRepository
	vmDiskRepo         *database.VMDiskRepository
	osseaVolumeRepo    *database.OSSEAVolumeRepository
	volumeMountRepo    *database.VolumeMountRepository
	cbtHistoryRepo     *database.CBTHistoryRepository
	replicationJobRepo *database.ReplicationJobRepository // VM-Centric Architecture support

	// Service dependencies
	mountManager      *volume.MountManager
	vmaProgressPoller VMAProgressPoller // Interface for VMA progress polling
}

// NewMigrationEngine creates a new migration workflow engine
func NewMigrationEngine(db database.Connection, mountManager *volume.MountManager, vmaProgressPoller VMAProgressPoller) *MigrationEngine {
	return &MigrationEngine{
		db:                 db,
		osseaConfigRepo:    database.NewOSSEAConfigRepository(db),
		vmDiskRepo:         database.NewVMDiskRepository(db),
		osseaVolumeRepo:    database.NewOSSEAVolumeRepository(db),
		volumeMountRepo:    database.NewVolumeMountRepository(db),
		cbtHistoryRepo:     database.NewCBTHistoryRepository(db),
		replicationJobRepo: database.NewReplicationJobRepository(db), // VM-Centric Architecture
		mountManager:       mountManager,
		vmaProgressPoller:  vmaProgressPoller,
	}
}

// MigrationRequest represents a complete migration request
type MigrationRequest struct {
	// Source VM information
	SourceVM    models.VMInfo `json:"source_vm"`
	VCenterHost string        `json:"vcenter_host"`
	Datacenter  string        `json:"datacenter"`

	// Migration configuration
	JobID           string `json:"job_id"`
	OSSEAConfigID   int    `json:"ossea_config_id"`
	ReplicationType string `json:"replication_type"` // initial, incremental
	TargetNetwork   string `json:"target_network"`

	// CBT configuration
	ChangeID         string `json:"change_id,omitempty"`
	PreviousChangeID string `json:"previous_change_id,omitempty"`
	SnapshotID       string `json:"snapshot_id,omitempty"`

	// Optional: Scheduler metadata
	ScheduleExecutionID string `json:"schedule_execution_id,omitempty"`
	VMGroupID           string `json:"vm_group_id,omitempty"`
	ScheduledBy         string `json:"scheduled_by,omitempty"`

	// Optional: Existing VM context for replication on managed VMs
	ExistingContextID string `json:"existing_context_id,omitempty"`
}

// MigrationResult represents the result of a migration workflow
type MigrationResult struct {
	JobID           string                  `json:"job_id"`
	Status          string                  `json:"status"`
	CreatedVolumes  []VolumeProvisionResult `json:"created_volumes"`
	MountedVolumes  []VolumeMountResult     `json:"mounted_volumes"`
	ErrorMessage    string                  `json:"error_message,omitempty"`
	ProgressPercent float64                 `json:"progress_percent"`
}

// VolumeProvisionResult represents the result of volume provisioning
type VolumeProvisionResult struct {
	VMDiskID      int    `json:"vm_disk_id"`
	OSSEAVolumeID string `json:"ossea_volume_id"`
	VolumeName    string `json:"volume_name"`
	SizeGB        int    `json:"size_gb"`
	Status        string `json:"status"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// VolumeMountResult represents the result of volume mounting
type VolumeMountResult struct {
	OSSEAVolumeID  string `json:"ossea_volume_id"` // CloudStack volume UUID (changed from int)
	DevicePath     string `json:"device_path"`
	MountPoint     string `json:"mount_point"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"error_message,omitempty"`
	DiskUnitNumber int    `json:"disk_unit_number"` // SCSI unit number for NBD export naming
}

// StartMigration initiates a complete VMware to OSSEA migration workflow
func (m *MigrationEngine) StartMigration(ctx context.Context, req *MigrationRequest) (*MigrationResult, error) {
	log.WithFields(log.Fields{
		"job_id":           req.JobID,
		"source_vm":        req.SourceVM.Name,
		"ossea_config_id":  req.OSSEAConfigID,
		"replication_type": req.ReplicationType,
	}).Info("🚀 Starting migration workflow")

	result := &MigrationResult{
		JobID:           req.JobID,
		Status:          "starting",
		CreatedVolumes:  []VolumeProvisionResult{},
		MountedVolumes:  []VolumeMountResult{},
		ProgressPercent: 0.0,
	}

	// Phase 1: Create replication job record
	if err := m.createReplicationJob(ctx, req); err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to create replication job: %v", err)
		return result, err
	}
	result.ProgressPercent = 10.0
	m.updateSetupProgress(req.JobID, "initializing", 10.0)

	// Phase 2: Analyze VM disks and create VM disk records
	if err := m.analyzeAndRecordVMDisks(req); err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to analyze VM disks: %v", err)
		return result, err
	}
	result.ProgressPercent = 20.0
	m.updateSetupProgress(req.JobID, "analyzing", 20.0)

	// Phase 3: Provision OSSEA volumes
	volumeResults, err := m.provisionOSSEAVolumes(ctx, req)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to provision volumes: %v", err)
		return result, err
	}
	result.CreatedVolumes = volumeResults
	result.ProgressPercent = 60.0
	m.updateSetupProgress(req.JobID, "provisioning", 60.0)

	// Phase 4: Attach volumes to OMA appliance (but don't mount as filesystems)
	// NBD needs raw block device access, so we attach but don't mount
	log.Info("Attaching OSSEA volumes to OMA appliance for NBD access")
	attachResults, err := m.attachOSSEAVolumes(ctx, req, volumeResults)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to attach volumes: %v", err)
		return result, err
	}
	result.MountedVolumes = attachResults
	result.ProgressPercent = 70.0
	m.updateSetupProgress(req.JobID, "attaching", 70.0)

	// Phase 4.5: Verify device paths exist (CloudStack API vs actual Linux devices)
	log.Info("Verifying device paths exist and are accessible")
	attachResults, err = m.verifyDevicePaths(attachResults)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to verify device paths: %v", err)
		return result, err
	}

	// Phase 5: Query NBD exports auto-created by Volume Daemon during volume attachment
	log.Info("Querying NBD exports auto-created by Volume Daemon")
	nbdExports, err := m.queryNBDExportsFromVolumeDaemon(req, attachResults)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to query NBD exports from Volume Daemon: %v", err)
		return result, err
	}
	result.ProgressPercent = 80.0
	m.updateSetupProgress(req.JobID, "configuring", 80.0)

	// Phase 6: Initialize CBT tracking
	if err := m.initializeCBTTracking(req); err != nil {
		log.WithError(err).Warn("Failed to initialize CBT tracking, continuing without CBT")
		// Non-fatal error - continue without CBT
	}
	result.ProgressPercent = 85.0

	// Phase 7: Update job status to ready for sync
	if err := m.updateSetupProgress(req.JobID, "ready_for_sync", 85.0); err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to update job status: %v", err)
		return result, err
	}

	// Phase 8: Initiate VMware replication via VMA API
	log.WithField("job_id", req.JobID).Info("Initiating VMware replication via VMA")
	if err := m.initiateVMwareReplication(req, nbdExports); err != nil {
		log.WithError(err).Error("Failed to initiate VMware replication")
		// Update status to failed
		m.updateSetupProgress(req.JobID, "replication_failed", 85.0)
		result.Status = "replication_failed"
		result.ErrorMessage = fmt.Sprintf("Failed to initiate VMware replication: %v", err)
		return result, err
	}

	// Update status to replicating with NBD export name and start VMA progress polling
	if err := m.updateJobStatusWithNBDExport(req.JobID, "replicating", 0.0, nbdExports); err != nil {
		log.WithError(err).Warn("Failed to update job status to replicating")
	}

	// Start VMA progress polling for real-time updates
	if m.vmaProgressPoller != nil {
		if err := m.vmaProgressPoller.StartPolling(req.JobID); err != nil {
			log.WithError(err).WithField("job_id", req.JobID).Warn("Failed to start VMA progress polling - continuing with static progress")
		} else {
			log.WithField("job_id", req.JobID).Info("🚀 Started VMA progress polling - real-time progress tracking active")
		}
	}

	result.Status = "replicating"
	result.ProgressPercent = 0.0 // VMA will update with real progress during replication

	log.WithFields(log.Fields{
		"job_id":          req.JobID,
		"volumes_created": len(result.CreatedVolumes),
		"volumes_mounted": len(result.MountedVolumes),
	}).Info("✅ Migration workflow started - VMware replication initiated")

	return result, nil
}

// stringPtrOrNil returns a pointer to the string if it's not empty, otherwise nil
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// createReplicationJob creates the main replication job record
func (m *MigrationEngine) createReplicationJob(ctx context.Context, req *MigrationRequest) error {
	log.WithField("job_id", req.JobID).Info("Creating replication job record")

	job := &database.ReplicationJob{
		ID:               req.JobID,
		SourceVMID:       req.SourceVM.ID,
		SourceVMName:     req.SourceVM.Name,
		SourceVMPath:     req.SourceVM.Path,
		VCenterHost:      req.VCenterHost,
		Datacenter:       req.Datacenter,
		ReplicationType:  req.ReplicationType,
		TargetNetwork:    req.TargetNetwork,
		Status:           "initializing",
		ProgressPercent:  0.0,
		OSSEAConfigID:    req.OSSEAConfigID,
		ChangeID:         req.ChangeID,
		PreviousChangeID: req.PreviousChangeID,
		SnapshotID:       req.SnapshotID,
		// ✅ NEW: Scheduler metadata (populated when called by scheduler)
		ScheduleExecutionID: stringPtrOrNil(req.ScheduleExecutionID),
		VMGroupID:           stringPtrOrNil(req.VMGroupID),
		ScheduledBy:         stringPtrOrNil(req.ScheduledBy),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// If we have an existing context ID, use it
	if req.ExistingContextID != "" {
		job.VMContextID = req.ExistingContextID
		log.WithFields(log.Fields{
			"job_id":     req.JobID,
			"context_id": req.ExistingContextID,
		}).Info("Creating replication job with existing VM context")
	}

	// Save to database using ReplicationJobRepository with VM-Centric Architecture support
	if err := m.replicationJobRepo.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create replication job: %w", err)
	}

	log.WithField("job_id", req.JobID).Info("Replication job record created with VM context")
	return nil
}

// analyzeAndRecordVMDisks analyzes VM disk configuration and creates VM disk records with VM specifications
func (m *MigrationEngine) analyzeAndRecordVMDisks(req *MigrationRequest) error {
	// Get VM context ID for this job
	vmContextID, err := m.getVMContextIDForJob(req.JobID)
	if err != nil {
		log.WithError(err).WithField("job_id", req.JobID).Warn("Failed to get VM context ID, continuing without it")
		vmContextID = "" // Continue without VM context for backward compatibility
	}
	log.WithFields(log.Fields{
		"job_id":     req.JobID,
		"vm_name":    req.SourceVM.Name,
		"disk_count": len(req.SourceVM.Disks),
		"cpu_count":  req.SourceVM.CPUs,
		"memory_mb":  req.SourceVM.MemoryMB,
	}).Info("Analyzing VM disks and storing VM specifications for migration")

	if len(req.SourceVM.Disks) == 0 {
		return fmt.Errorf("source VM has no disks to migrate")
	}

	// Serialize network configuration for storage
	networkConfigJSON := ""
	if len(req.SourceVM.Networks) > 0 {
		networkBytes, err := json.Marshal(req.SourceVM.Networks)
		if err != nil {
			log.WithError(err).Warn("Failed to serialize network configuration")
		} else {
			networkConfigJSON = string(networkBytes)
		}
	}

	for i, disk := range req.SourceVM.Disks {
		vmDisk := &database.VMDisk{
			JobID:            req.JobID,
			VMContextID:      vmContextID, // VM-Centric Architecture integration
			DiskID:           disk.ID,
			VMDKPath:         disk.Path,
			SizeGB:           int(disk.SizeGB),
			Datastore:        disk.Datastore,
			UnitNumber:       i,
			Label:            disk.Label,
			CapacityBytes:    disk.CapacityBytes,
			ProvisioningType: disk.ProvisioningType,
			SyncStatus:       "pending",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		// 🎯 CRITICAL FIX: Store VM specifications in ALL disk records for failover compatibility
		// This ensures VM info service gets correct specs regardless of which record it queries
		vmDisk.CPUCount = req.SourceVM.CPUs
		vmDisk.MemoryMB = req.SourceVM.MemoryMB
		vmDisk.OSType = req.SourceVM.OSType
		vmDisk.PowerState = req.SourceVM.PowerState
		vmDisk.DisplayName = req.SourceVM.DisplayName
		vmDisk.Annotation = req.SourceVM.Annotation
		vmDisk.VMwareUUID = req.SourceVM.ID
		vmDisk.NetworkConfig = networkConfigJSON
		vmDisk.VMToolsVersion = req.SourceVM.VMwareToolsVersion

		if i == 0 {
			log.WithFields(log.Fields{
				"job_id":        req.JobID,
				"vm_name":       req.SourceVM.Name,
				"cpu_count":     vmDisk.CPUCount,
				"memory_mb":     vmDisk.MemoryMB,
				"os_type":       vmDisk.OSType,
				"power_state":   vmDisk.PowerState,
				"network_count": len(req.SourceVM.Networks),
				"tools_version": vmDisk.VMToolsVersion,
			}).Info("💾 Storing VM specifications in ALL disk records for failover compatibility")
		}

		// 🎯 CRITICAL FIX: Implement stable vm_disks UPSERT logic
		// Check if vm_disk record already exists for this VM context + disk ID
		existingDisk, err := m.vmDiskRepo.FindByContextAndDiskID(vmContextID, disk.ID)
		if err != nil {
			return fmt.Errorf("failed to check existing vm_disk for disk %s: %w", disk.ID, err)
		}

		if existingDisk != nil {
			// UPDATE existing record to maintain stable vm_disks.id
			log.WithFields(log.Fields{
				"existing_id":   existingDisk.ID,
				"vm_context_id": vmContextID,
				"disk_id":       disk.ID,
				"old_job_id":    existingDisk.JobID,
				"new_job_id":    req.JobID,
			}).Info("🔄 Updating existing VM disk record to maintain stable ID")

			// Preserve stable ID, update with new job data
			existingDisk.JobID = req.JobID
			existingDisk.VMDKPath = disk.Path
			existingDisk.SizeGB = int(disk.SizeGB)
			existingDisk.Datastore = disk.Datastore
			existingDisk.UnitNumber = i
			existingDisk.Label = disk.Label
			existingDisk.CapacityBytes = disk.CapacityBytes
			existingDisk.ProvisioningType = disk.ProvisioningType
			existingDisk.SyncStatus = "pending"
			existingDisk.UpdatedAt = time.Now()

			// 🎯 CRITICAL FIX: Store VM specs in ALL disk records for failover compatibility
			// This ensures VM info service gets correct specs regardless of which record it queries
			existingDisk.CPUCount = req.SourceVM.CPUs
			existingDisk.MemoryMB = req.SourceVM.MemoryMB
			existingDisk.OSType = req.SourceVM.OSType
			existingDisk.PowerState = req.SourceVM.PowerState
			existingDisk.DisplayName = req.SourceVM.DisplayName
			existingDisk.Annotation = req.SourceVM.Annotation
			existingDisk.VMwareUUID = req.SourceVM.ID
			existingDisk.NetworkConfig = networkConfigJSON
			existingDisk.VMToolsVersion = req.SourceVM.VMwareToolsVersion

			if err := m.vmDiskRepo.Update(existingDisk); err != nil {
				return fmt.Errorf("failed to update VM disk record for disk %s: %w", disk.ID, err)
			}

			// Use existing disk for subsequent operations
			vmDisk = existingDisk
		} else {
			// CREATE new record (first time for this VM context + disk)
			log.WithFields(log.Fields{
				"vm_context_id": vmContextID,
				"disk_id":       disk.ID,
				"job_id":        req.JobID,
			}).Info("🆕 Creating new VM disk record for first-time disk")

			if err := m.vmDiskRepo.Create(vmDisk); err != nil {
				return fmt.Errorf("failed to create VM disk record for disk %s: %w", disk.ID, err)
			}
		}

		// 🎯 NEW: Update VM context with specs from first disk record
		if i == 0 {
			if err := m.updateVMContextWithSpecs(vmContextID, vmDisk); err != nil {
				log.WithError(err).Warn("Failed to update VM context with specifications - continuing migration")
			}
		}

		log.WithFields(log.Fields{
			"job_id":    req.JobID,
			"disk_id":   disk.ID,
			"size_gb":   disk.SizeGB,
			"datastore": disk.Datastore,
		}).Info("VM disk record created")
	}

	log.WithFields(log.Fields{
		"job_id":       req.JobID,
		"vm_name":      req.SourceVM.Name,
		"total_disks":  len(req.SourceVM.Disks),
		"specs_stored": true,
	}).Info("✅ VM specifications and disk analysis completed")

	return nil
}

// updateVMContextWithSpecs updates VM context with specifications from VM disk record
func (m *MigrationEngine) updateVMContextWithSpecs(vmContextID string, vmDisk *database.VMDisk) error {
	updates := map[string]interface{}{
		"cpu_count":        vmDisk.CPUCount,
		"memory_mb":        vmDisk.MemoryMB,
		"os_type":          vmDisk.OSType,
		"power_state":      vmDisk.PowerState,
		"vm_tools_version": vmDisk.VMToolsVersion,
		"updated_at":       time.Now(),
	}

	err := m.db.GetGormDB().Table("vm_replication_contexts").
		Where("context_id = ?", vmContextID).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update VM context with specs: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_context_id":    vmContextID,
		"cpu_count":        vmDisk.CPUCount,
		"memory_mb":        vmDisk.MemoryMB,
		"os_type":          vmDisk.OSType,
		"power_state":      vmDisk.PowerState,
		"vm_tools_version": vmDisk.VMToolsVersion,
	}).Info("✅ Updated VM context with specifications from first disk record")

	return nil
}

// provisionOSSEAVolumes creates volumes in OSSEA for each VM disk
func (m *MigrationEngine) provisionOSSEAVolumes(ctx context.Context, req *MigrationRequest) ([]VolumeProvisionResult, error) {
	log.WithField("job_id", req.JobID).Info("Provisioning OSSEA volumes")

	// Get VM context ID for this job
	vmContextID, err := m.getVMContextIDForJob(req.JobID)
	if err != nil {
		log.WithError(err).WithField("job_id", req.JobID).Warn("Failed to get VM context ID for volume provisioning, continuing without it")
		vmContextID = "" // Continue without VM context for backward compatibility
	}

	// Get OSSEA configuration
	osseaConfig, err := m.osseaConfigRepo.GetByID(req.OSSEAConfigID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OSSEA configuration: %w", err)
	}

	// NOTE: OSSEA client no longer needed - using Volume Management Daemon
	// osseaClient creation removed as part of Volume Daemon integration

	// Get VM disks for this job
	vmDisks, err := m.vmDiskRepo.GetByJobID(req.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM disks: %w", err)
	}

	var results []VolumeProvisionResult

	for _, vmDisk := range vmDisks {
		result := VolumeProvisionResult{
			VMDiskID: vmDisk.ID,
			SizeGB:   vmDisk.SizeGB,
			Status:   "creating",
		}

		// Calculate the same volume size that would be used for creation (CapacityBytes + 5GB buffer)
		volumeSizeBytes := vmDisk.CapacityBytes + (5 * 1024 * 1024 * 1024) // CapacityBytes + 5GB
		calculatedSizeGB := int(volumeSizeBytes / (1024 * 1024 * 1024))

		// Check for existing volume for this VM disk first using the calculated size
		existingVolume, err := m.findExistingVolumeForVMDisk(req.SourceVM.Path, vmDisk.UnitNumber, calculatedSizeGB)
		if err != nil {
			log.WithError(err).Warn("Failed to check for existing volume, creating new one")
		}

		var volume *ossea.Volume
		if existingVolume != nil {
			log.WithFields(log.Fields{
				"job_id":      req.JobID,
				"vm_path":     req.SourceVM.Path,
				"disk_unit":   vmDisk.UnitNumber,
				"volume_id":   existingVolume.VolumeID,
				"volume_name": existingVolume.VolumeName,
			}).Info("♻️  Reusing existing OSSEA volume for incremental sync")

			// Convert existing volume to OSSEA format
			volume = &ossea.Volume{
				ID:    existingVolume.VolumeID,
				Name:  existingVolume.VolumeName,
				State: "Allocated",                                       // Assume allocated if it exists
				Size:  int64(existingVolume.SizeGB * 1024 * 1024 * 1024), // Convert GB to bytes
			}
			result.Status = "reused"
		} else {
			// Create new volume in OSSEA
			volumeName := fmt.Sprintf("migration-%s-%s-disk-%d",
				req.SourceVM.Name, // Use VM name for persistence across jobs
				req.SourceVM.Path[strings.LastIndex(req.SourceVM.Path, "/")+1:], // Extract VM name from path
				vmDisk.UnitNumber)

			// Calculate precise volume size using actual VMDK bytes + 5GB buffer
			volumeSizeBytes := vmDisk.CapacityBytes + (5 * 1024 * 1024 * 1024) // CapacityBytes + 5GB
			volumeSizeGB := float64(volumeSizeBytes) / (1024 * 1024 * 1024)

			log.WithFields(log.Fields{
				"job_id":            req.JobID,
				"vm_path":           req.SourceVM.Path,
				"disk_unit":         vmDisk.UnitNumber,
				"volume_name":       volumeName,
				"capacity_bytes":    vmDisk.CapacityBytes,
				"size_gb_old":       vmDisk.SizeGB,
				"volume_size_gb":    volumeSizeGB,
				"volume_size_bytes": volumeSizeBytes,
			}).Info("🆕 Creating new OSSEA volume via Volume Daemon with CapacityBytes sizing")

			// Use Volume Management Daemon for centralized volume creation
			volumeClient := common.NewVolumeClient("http://localhost:8090")
			createReq := common.CreateVolumeRequest{
				Name:           volumeName,
				Size:           volumeSizeBytes, // Use actual CapacityBytes + 5GB buffer in bytes
				DiskOfferingID: osseaConfig.DiskOfferingID,
				ZoneID:         osseaConfig.Zone,
				Metadata: map[string]string{
					"migration_job_id":      req.JobID,
					"source_vm_name":        req.SourceVM.Name,
					"source_vm_path":        req.SourceVM.Path,
					"disk_unit_number":      fmt.Sprintf("%d", vmDisk.UnitNumber),
					"vmware_capacity_bytes": fmt.Sprintf("%d", vmDisk.CapacityBytes),
					"created_by":            "migratekit-migration-engine",
				},
			}

			operation, err := volumeClient.CreateVolume(ctx, createReq)
			if err != nil {
				result.Status = "failed"
				result.ErrorMessage = err.Error()
				results = append(results, result)
				return results, fmt.Errorf("failed to create volume via daemon for disk %s: %w", vmDisk.DiskID, err)
			}

			// Wait for volume creation completion
			completedOp, err := volumeClient.WaitForCompletionWithTimeout(ctx, operation.ID, 2*time.Minute)
			if err != nil {
				result.Status = "failed"
				result.ErrorMessage = fmt.Sprintf("Volume creation timeout: %v", err)
				results = append(results, result)
				return results, fmt.Errorf("volume creation failed for disk %s: %w", vmDisk.DiskID, err)
			}

			// Extract volume ID from daemon response
			volumeID, ok := completedOp.Response["volume_id"].(string)
			if !ok || volumeID == "" {
				result.Status = "failed"
				result.ErrorMessage = "Volume Daemon did not return volume ID"
				results = append(results, result)
				return results, fmt.Errorf("volume creation failed - no volume ID returned for disk %s", vmDisk.DiskID)
			}

			// Create volume object compatible with existing code
			volume = &ossea.Volume{
				ID:   volumeID,
				Name: volumeName,
				Size: volumeSizeBytes,
			}
			result.Status = "created"
		}

		// Create OSSEA volume record in database (only for new volumes)
		var osseaVolume *database.OSSEAVolume
		var volumeDBID int

		if result.Status == "created" {
			// Calculate volume size in GB from the precise byte calculation
			volumeSizeGBForDB := int((volume.Size + 1073741823) / 1073741824) // Round up to next GB

			osseaVolume = &database.OSSEAVolume{
				VMContextID:   vmContextID, // VM-Centric Architecture integration
				VolumeID:      volume.ID,
				VolumeName:    volume.Name,
				SizeGB:        volumeSizeGBForDB, // Use actual volume size created via daemon
				OSSEAConfigID: req.OSSEAConfigID,
				VolumeType:    "DATADISK",
				Status:        "creating",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			if err := m.osseaVolumeRepo.Create(osseaVolume); err != nil {
				result.Status = "failed"
				result.ErrorMessage = err.Error()
				results = append(results, result)
				return results, fmt.Errorf("failed to create OSSEA volume record: %w", err)
			}
			volumeDBID = osseaVolume.ID
		} else {
			// For reused volumes, just log that we're using existing database record
			log.WithFields(log.Fields{
				"job_id":      req.JobID,
				"volume_id":   volume.ID,
				"volume_name": volume.Name,
			}).Info("♻️  Using existing database record for reused volume")
			// For reused volumes, get the existing database ID
			volumeDBID = existingVolume.ID
		}

		// Update VM disk with OSSEA volume reference

		if err := m.db.GetGormDB().Model(&vmDisk).Update("ossea_volume_id", volumeDBID).Error; err != nil {
			log.WithError(err).Warn("Failed to update VM disk with OSSEA volume reference")
		}

		result.OSSEAVolumeID = volume.ID
		result.VolumeName = volume.Name
		// Status was already set above (either "created" or "reused")
		results = append(results, result)

		log.WithFields(log.Fields{
			"job_id":                req.JobID,
			"volume_id":             volume.ID,
			"volume_name":           volume.Name,
			"vmware_capacity_bytes": vmDisk.CapacityBytes,
			"volume_size_bytes":     volume.Size,
			"volume_size_gb":        int((volume.Size + 1073741823) / 1073741824),
			"size_gb_old":           vmDisk.SizeGB,
			"disk_id":               vmDisk.DiskID,
			"created_via":           "volume_daemon",
		}).Info("✅ OSSEA volume created successfully via Volume Daemon with CapacityBytes sizing")
	}

	return results, nil
}

// updateJobStatus updates the status and progress of a migration job
func (m *MigrationEngine) updateJobStatus(jobID, status string, progressPercent float64) error {
	log.WithFields(log.Fields{
		"job_id":           jobID,
		"status":           status,
		"progress_percent": progressPercent,
	}).Debug("Updating job status")

	updates := map[string]interface{}{
		"status":           status,
		"progress_percent": progressPercent,
		"updated_at":       time.Now(),
	}

	// Add completion timestamp if job is finished
	if status == "completed" || status == "failed" || status == "cancelled" {
		updates["completed_at"] = time.Now()
	}

	if err := m.db.GetGormDB().Model(&database.ReplicationJob{}).Where("id = ?", jobID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// updateSetupProgress updates the setup progress and status for OMA pre-replication phases
func (m *MigrationEngine) updateSetupProgress(jobID, status string, setupProgressPercent float64) error {
	log.WithFields(log.Fields{
		"job_id":                 jobID,
		"status":                 status,
		"setup_progress_percent": setupProgressPercent,
	}).Debug("Updating setup progress")

	updates := map[string]interface{}{
		"status":                 status,
		"setup_progress_percent": setupProgressPercent,
		"updated_at":             time.Now(),
	}

	// Add completion timestamp if job is finished
	if status == "completed" || status == "failed" || status == "cancelled" {
		updates["completed_at"] = time.Now()
	}

	if err := m.db.GetGormDB().Model(&database.ReplicationJob{}).Where("id = ?", jobID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update setup progress: %w", err)
	}

	return nil
}

// updateJobStatusWithNBDExport updates job status and populates nbd_export_name for VMA progress polling
func (m *MigrationEngine) updateJobStatusWithNBDExport(jobID, status string, progressPercent float64, nbdExports []*nbd.ExportInfo) error {
	log.WithFields(log.Fields{
		"job_id":           jobID,
		"status":           status,
		"progress_percent": progressPercent,
		"nbd_exports":      len(nbdExports),
	}).Debug("Updating job status with NBD export name")

	updates := map[string]interface{}{
		"status":           status,
		"progress_percent": progressPercent,
		"updated_at":       time.Now(),
	}

	// Populate nbd_export_name from the first NBD export for VMA progress polling
	// For multi-disk VMs, we use the primary disk's export name as the job identifier
	if len(nbdExports) > 0 {
		primaryExport := nbdExports[0]
		updates["nbd_export_name"] = primaryExport.ExportName

		log.WithFields(log.Fields{
			"job_id":        jobID,
			"export_name":   primaryExport.ExportName,
			"export_port":   primaryExport.Port,
			"export_device": primaryExport.DevicePath,
		}).Info("✅ Populated nbd_export_name for VMA progress polling")
	} else {
		log.WithField("job_id", jobID).Warn("No NBD exports available to populate nbd_export_name")
	}

	// Add completion timestamp if job is finished
	if status == "completed" || status == "failed" || status == "cancelled" {
		updates["completed_at"] = time.Now()
	}

	if err := m.db.GetGormDB().Model(&database.ReplicationJob{}).Where("id = ?", jobID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update job status with NBD export: %w", err)
	}

	return nil
}

// GetMigrationStatus retrieves the current status of a migration
func (m *MigrationEngine) GetMigrationStatus(jobID string) (*MigrationStatusResult, error) {
	log.WithField("job_id", jobID).Debug("Getting migration status")

	// Get replication job
	var job database.ReplicationJob
	if err := m.db.GetGormDB().Where("id = ?", jobID).First(&job).Error; err != nil {
		return nil, fmt.Errorf("failed to get replication job: %w", err)
	}

	// Get VM disks
	vmDisks, err := m.vmDiskRepo.GetByJobID(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM disks: %w", err)
	}

	// Get volume mounts
	mounts, err := m.volumeMountRepo.GetByJobID(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume mounts: %w", err)
	}

	// Get CBT history
	cbtHistory, err := m.cbtHistoryRepo.GetByJobID(jobID)
	if err != nil {
		log.WithError(err).Debug("Failed to get CBT history, continuing without it")
		cbtHistory = []database.CBTHistory{}
	}

	// Build status result
	result := &MigrationStatusResult{
		JobID:           job.ID,
		Status:          job.Status,
		ProgressPercent: job.ProgressPercent,
		CreatedAt:       job.CreatedAt,
		UpdatedAt:       job.UpdatedAt,
		StartedAt:       job.StartedAt,
		CompletedAt:     job.CompletedAt,
		SourceVM: SourceVMStatus{
			ID:          job.SourceVMID,
			Name:        job.SourceVMName,
			Path:        job.SourceVMPath,
			VCenterHost: job.VCenterHost,
			Datacenter:  job.Datacenter,
		},
		Configuration: MigrationConfigStatus{
			ReplicationType: job.ReplicationType,
			TargetNetwork:   job.TargetNetwork,
			OSSEAConfigID:   job.OSSEAConfigID,
		},
		Disks:      make([]DiskStatus, len(vmDisks)),
		Mounts:     make([]MountStatus, len(mounts)),
		CBTHistory: make([]CBTStatus, len(cbtHistory)),
	}

	// Populate disk status
	for i, disk := range vmDisks {
		result.Disks[i] = DiskStatus{
			ID:                  disk.ID,
			DiskID:              disk.DiskID,
			VMDKPath:            disk.VMDKPath,
			SizeGB:              disk.SizeGB,
			SyncStatus:          disk.SyncStatus,
			SyncProgressPercent: disk.SyncProgressPercent,
			BytesSynced:         disk.BytesSynced,
			ChangeID:            disk.DiskChangeID,
			OSSEAVolumeID:       disk.OSSEAVolumeID,
		}
	}

	// Populate mount status
	for i, mount := range mounts {
		result.Mounts[i] = MountStatus{
			ID:             mount.ID,
			OSSEAVolumeID:  mount.OSSEAVolumeID,
			DevicePath:     mount.DevicePath,
			MountPoint:     mount.MountPoint,
			MountStatus:    mount.MountStatus,
			FilesystemType: mount.FilesystemType,
			MountedAt:      mount.MountedAt,
		}
	}

	// Populate CBT history
	for i, cbt := range cbtHistory {
		result.CBTHistory[i] = CBTStatus{
			ID:                  cbt.ID,
			DiskID:              cbt.DiskID,
			ChangeID:            cbt.ChangeID,
			SyncType:            cbt.SyncType,
			SyncSuccess:         cbt.SyncSuccess,
			BlocksChanged:       cbt.BlocksChanged,
			BytesTransferred:    cbt.BytesTransferred,
			SyncDurationSeconds: cbt.SyncDurationSeconds,
			CreatedAt:           cbt.CreatedAt,
		}
	}

	return result, nil
}

// MigrationStatusResult represents the complete status of a migration
type MigrationStatusResult struct {
	JobID           string                `json:"job_id"`
	Status          string                `json:"status"`
	ProgressPercent float64               `json:"progress_percent"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	StartedAt       *time.Time            `json:"started_at,omitempty"`
	CompletedAt     *time.Time            `json:"completed_at,omitempty"`
	SourceVM        SourceVMStatus        `json:"source_vm"`
	Configuration   MigrationConfigStatus `json:"configuration"`
	Disks           []DiskStatus          `json:"disks"`
	Mounts          []MountStatus         `json:"mounts"`
	CBTHistory      []CBTStatus           `json:"cbt_history"`
}

// SourceVMStatus represents source VM information
type SourceVMStatus struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	VCenterHost string `json:"vcenter_host"`
	Datacenter  string `json:"datacenter"`
}

// MigrationConfigStatus represents migration configuration
type MigrationConfigStatus struct {
	ReplicationType string `json:"replication_type"`
	TargetNetwork   string `json:"target_network"`
	OSSEAConfigID   int    `json:"ossea_config_id"`
}

// DiskStatus represents the status of a VM disk
type DiskStatus struct {
	ID                  int     `json:"id"`
	DiskID              string  `json:"disk_id"`
	VMDKPath            string  `json:"vmdk_path"`
	SizeGB              int     `json:"size_gb"`
	SyncStatus          string  `json:"sync_status"`
	SyncProgressPercent float64 `json:"sync_progress_percent"`
	BytesSynced         int64   `json:"bytes_synced"`
	ChangeID            string  `json:"change_id"`
	OSSEAVolumeID       int     `json:"ossea_volume_id"`
}

// MountStatus represents the status of a volume mount
type MountStatus struct {
	ID             int        `json:"id"`
	OSSEAVolumeID  int        `json:"ossea_volume_id"`
	DevicePath     string     `json:"device_path"`
	MountPoint     string     `json:"mount_point"`
	MountStatus    string     `json:"mount_status"`
	FilesystemType string     `json:"filesystem_type"`
	MountedAt      *time.Time `json:"mounted_at,omitempty"`
}

// CBTStatus represents CBT tracking status
type CBTStatus struct {
	ID                  int       `json:"id"`
	DiskID              string    `json:"disk_id"`
	ChangeID            string    `json:"change_id"`
	SyncType            string    `json:"sync_type"`
	SyncSuccess         bool      `json:"sync_success"`
	BlocksChanged       int       `json:"blocks_changed"`
	BytesTransferred    int64     `json:"bytes_transferred"`
	SyncDurationSeconds int       `json:"sync_duration_seconds"`
	CreatedAt           time.Time `json:"created_at"`
}

// initiateVMwareReplication calls VMA API to start actual replication
func (m *MigrationEngine) initiateVMwareReplication(req *MigrationRequest, nbdExports []*nbd.ExportInfo) error {
	log.WithField("job_id", req.JobID).Info("Calling VMA API to initiate VMware replication")

	// Get vCenter credentials from secure credential service
	var vcenterUsername, vcenterPassword string

	encryptionService, err := services.NewCredentialEncryptionService()
	if err != nil {
		// Fallback to hardcoded during transition
		log.WithError(err).Warn("Failed to initialize encryption service, using fallback credentials")
		vcenterUsername = "administrator@vsphere.local"
		vcenterPassword = "EmyGVoBFesGQc47-"
	} else {
		credentialService := services.NewVMwareCredentialService(&m.db, encryptionService)
		creds, err := credentialService.GetDefaultCredentials(context.Background())
		if err != nil {
			// Fallback to hardcoded on error
			log.WithError(err).Warn("Failed to get default credentials, using fallback")
			vcenterUsername = "administrator@vsphere.local"
			vcenterPassword = "EmyGVoBFesGQc47-"
		} else {
			// Use service-managed credentials
			log.WithField("vcenter_host", creds.VCenterHost).Info("✅ Using credential service for VMA replication")
			vcenterUsername = creds.Username
			vcenterPassword = creds.Password
		}
	}

	// Build NBD target information for VMA (NBD connection details)
	var nbd_targets []map[string]interface{}
	// Resolve OMA NBD host from environment to avoid hardcoding
	omaNbdHost := os.Getenv("OMA_NBD_HOST")
	if omaNbdHost == "" {
		// If not set, prefer passing only port+export and let VMA resolve host via its own env
		log.Warn("OMA_NBD_HOST not set on OMA; sending NBD targets without host, VMA must compose using its environment")
	}

	for i, export := range nbdExports {
		var devicePath string
		if omaNbdHost != "" {
			// Use the actual unique export name with configured host
			devicePath = fmt.Sprintf("nbd://%s:%d/%s", omaNbdHost, export.Port, export.ExportName)
		} else {
			// Hostless form hint; VMA will substitute host using its OMA_NBD_HOST
			devicePath = fmt.Sprintf("nbd://:%d/%s", export.Port, export.ExportName)
		}

		// 🎯 CRITICAL FIX: Get VMware disk key directly from volume correlation
		// Extract volume_id from export_name (format: migration-vol-{uuid})
		volumeID := strings.TrimPrefix(export.ExportName, "migration-vol-")

		vmwareDiskKey, err := m.getVMwareDiskKeyFromVolume(volumeID)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"export_name": export.ExportName,
				"volume_id":   volumeID,
			}).Warn("Failed to get VMware disk key correlation, using array index fallback")
			vmwareDiskKey = fmt.Sprintf("200%d", i) // Fallback: disk-2000, disk-2001, etc.
		}

		nbd_targets = append(nbd_targets, map[string]interface{}{
			"device_path":     devicePath,
			"vmware_disk_key": vmwareDiskKey, // 🎯 NEW: Send VMware disk key directly (e.g., "2000", "2001")
		})

		log.WithFields(log.Fields{
			"export_name":     export.ExportName,
			"volume_id":       volumeID,
			"vmware_disk_key": vmwareDiskKey,
			"device_path":     devicePath,
		}).Debug("🎯 Built NBD target with VMware disk key correlation")
	}

	// Debug: Log NBD targets being sent to VMA
	log.WithFields(log.Fields{
		"job_id":            req.JobID,
		"nbd_exports_count": len(nbdExports),
		"nbd_targets_count": len(nbd_targets),
		"nbd_targets":       nbd_targets,
	}).Info("Built NBD targets for VMA API call")

	// VMA API request payload with NBD target information
	// Get OMA URL from environment, fallback to localhost for tunnel
	omaURL := os.Getenv("OMA_API_URL")
	if omaURL == "" {
		omaURL = "http://localhost:8082" // Default for reverse tunnel
	}

	vmaRequest := map[string]interface{}{
		"job_id":      req.JobID,
		"vcenter":     req.VCenterHost,
		"username":    vcenterUsername, // Using credential service
		"password":    vcenterPassword, // Using credential service
		"vm_paths":    []string{req.SourceVM.Path},
		"oma_url":     omaURL,
		"nbd_targets": nbd_targets, // NBD connection URLs for migratekit
	}

	// Convert to JSON
	jsonData, err := json.Marshal(vmaRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal VMA request: %w", err)
	}

	// Get VMA API URL from environment, fallback to localhost for tunnel
	vmaAPIURL := os.Getenv("VMA_API_URL")
	if vmaAPIURL == "" {
		vmaAPIURL = "http://localhost:9081" // Default for reverse tunnel
	}

	// Make HTTP request to VMA via reverse tunnel
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(vmaAPIURL+"/api/v1/replicate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to call VMA API: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("VMA API returned status %d", resp.StatusCode)
	}

	log.WithFields(log.Fields{
		"job_id":     req.JobID,
		"vm_path":    req.SourceVM.Path,
		"vma_status": resp.StatusCode,
	}).Info("✅ VMware replication initiated via VMA API")

	return nil
}

// getVMDiskUnitNumber retrieves the unit number for a VM disk by ID
func (m *MigrationEngine) getVMDiskUnitNumber(vmDiskID int) (int, error) {
	var vmDisk database.VMDisk
	if err := m.db.GetGormDB().Where("id = ?", vmDiskID).First(&vmDisk).Error; err != nil {
		return 0, fmt.Errorf("failed to get VM disk unit number for ID %d: %w", vmDiskID, err)
	}
	return vmDisk.UnitNumber, nil
}

// attachOSSEAVolumes attaches volumes to OMA appliance for NBD access (without mounting)
func (m *MigrationEngine) attachOSSEAVolumes(ctx context.Context, req *MigrationRequest, volumeResults []VolumeProvisionResult) ([]VolumeMountResult, error) {
	log.WithField("job_id", req.JobID).Info("Attaching OSSEA volumes to OMA appliance")

	// Get OSSEA configuration
	osseaConfig, err := m.osseaConfigRepo.GetByID(req.OSSEAConfigID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OSSEA configuration: %w", err)
	}

	// Validate OMA VM ID is configured
	if osseaConfig.OMAVMID == "" {
		return nil, fmt.Errorf("OMA VM ID not configured in OSSEA configuration")
	}

	var attachResults []VolumeMountResult

	for _, volumeResult := range volumeResults {
		log.WithFields(log.Fields{
			"job_id":      req.JobID,
			"volume_id":   volumeResult.OSSEAVolumeID,
			"volume_name": volumeResult.VolumeName,
			"oma_vm_id":   osseaConfig.OMAVMID,
			"status":      volumeResult.Status,
		}).Info("Processing OSSEA volume attachment")

		// Check if volume is already attached (for reused volumes)
		if volumeResult.Status == "reused" {
			// For reused volumes, verify device path with Volume Management Daemon
			log.WithFields(log.Fields{
				"job_id":    req.JobID,
				"volume_id": volumeResult.OSSEAVolumeID,
			}).Info("♻️  Reused volume - verifying device path with Volume Daemon")

			// Use Volume Management Daemon to get REAL device path
			volumeClient := common.NewVolumeClient("http://localhost:8090")

			mapping, err := volumeClient.GetVolumeDevice(ctx, volumeResult.OSSEAVolumeID)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"job_id":    req.JobID,
					"volume_id": volumeResult.OSSEAVolumeID,
				}).Warn("Failed to get device mapping from daemon for reused volume - will reattach")
				// Fall through to reattachment logic
			} else if mapping.DevicePath != "" {
				log.WithFields(log.Fields{
					"job_id":      req.JobID,
					"volume_id":   volumeResult.OSSEAVolumeID,
					"device_path": mapping.DevicePath,
				}).Info("✅ Reused volume device path verified by daemon")

				// Get disk unit number for NBD export naming
				unitNumber, err := m.getVMDiskUnitNumber(volumeResult.VMDiskID)
				if err != nil {
					log.WithError(err).WithField("vm_disk_id", volumeResult.VMDiskID).Warn("Failed to get disk unit number, using 0")
					unitNumber = 0
				}

				// Use daemon-verified device path
				attachResult := VolumeMountResult{
					OSSEAVolumeID:  volumeResult.OSSEAVolumeID, // Use CloudStack volume UUID, not VM disk ID
					DevicePath:     mapping.DevicePath,
					MountPoint:     "raw-block-device",
					Status:         "attached",
					ErrorMessage:   "",
					DiskUnitNumber: unitNumber,
				}
				attachResults = append(attachResults, attachResult)
				continue
			} else {
				log.WithFields(log.Fields{
					"job_id":    req.JobID,
					"volume_id": volumeResult.OSSEAVolumeID,
				}).Warn("Volume daemon reports no device mapping for reused volume - will reattach")
				// Fall through to reattachment logic
			}
		}

		// Attach volume to OMA appliance using Volume Management Daemon
		log.WithFields(log.Fields{
			"job_id":    req.JobID,
			"volume_id": volumeResult.OSSEAVolumeID,
			"oma_vm_id": osseaConfig.OMAVMID,
		}).Info("Attaching OSSEA volume to OMA appliance via Volume Daemon")

		// Use Volume Management Daemon for attachment with real device correlation
		volumeClient := common.NewVolumeClient("http://localhost:8090")

		operation, err := volumeClient.AttachVolume(ctx, volumeResult.OSSEAVolumeID, osseaConfig.OMAVMID)
		if err != nil {
			return nil, fmt.Errorf("failed to start volume attachment via daemon: %w", err)
		}

		// Wait for completion with device correlation
		completed, err := volumeClient.WaitForCompletionWithTimeout(ctx, operation.ID, 2*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("volume attachment failed: %w", err)
		}

		// Get REAL device path from daemon (no more arithmetic assumptions!)
		devicePath, ok := completed.Response["device_path"].(string)
		if !ok || devicePath == "" {
			return nil, fmt.Errorf("no device path returned from volume attachment for volume %s", volumeResult.OSSEAVolumeID)
		}

		log.WithFields(log.Fields{
			"job_id":       req.JobID,
			"volume_id":    volumeResult.OSSEAVolumeID,
			"device_path":  devicePath,
			"operation_id": operation.ID,
		}).Info("✅ Volume attached with REAL device correlation via daemon")

		// Note: ossea_volumes table is now automatically updated by Volume Daemon
		// during attach/detach operations - no manual UpdateVolumeStatus calls needed

		// Get disk unit number for NBD export naming
		unitNumber, err := m.getVMDiskUnitNumber(volumeResult.VMDiskID)
		if err != nil {
			log.WithError(err).WithField("vm_disk_id", volumeResult.VMDiskID).Warn("Failed to get disk unit number, using 0")
			unitNumber = 0
		}

		// Create mount result (device is attached as raw block device, not mounted as filesystem)
		attachResult := VolumeMountResult{
			OSSEAVolumeID:  volumeResult.OSSEAVolumeID, // Use CloudStack volume UUID, not VM disk ID
			DevicePath:     devicePath,
			MountPoint:     "raw-block-device", // Not mounted as filesystem - NBD will access directly
			Status:         "attached",
			ErrorMessage:   "",
			DiskUnitNumber: unitNumber,
		}

		attachResults = append(attachResults, attachResult)

		// Update the OSSEA volume record in database with device path and status
		if err := m.db.GetGormDB().Model(&database.OSSEAVolume{}).
			Where("volume_id = ?", volumeResult.OSSEAVolumeID).
			Updates(map[string]interface{}{
				"device_path": devicePath,
				"status":      "attached",
				"updated_at":  time.Now(),
			}).Error; err != nil {
			log.WithError(err).Warn("Failed to update OSSEA volume with device path")
		}

		log.WithFields(log.Fields{
			"job_id":      req.JobID,
			"volume_id":   volumeResult.OSSEAVolumeID,
			"device_path": devicePath,
		}).Info("✅ OSSEA volume attached to OMA appliance")
	}

	log.WithFields(log.Fields{
		"job_id":        req.JobID,
		"volumes_count": len(attachResults),
	}).Info("✅ All OSSEA volumes attached to OMA appliance")

	return attachResults, nil
}

// queryNBDExportsFromVolumeDaemon queries NBD exports auto-created by Volume Daemon during volume attachment
func (m *MigrationEngine) queryNBDExportsFromVolumeDaemon(req *MigrationRequest, attachResults []VolumeMountResult) ([]*nbd.ExportInfo, error) {
	log.WithFields(log.Fields{
		"job_id":         req.JobID,
		"attached_count": len(attachResults),
	}).Info("Querying NBD exports auto-created by Volume Daemon during volume attachment")

	// Initialize Volume Daemon client for NBD export querying
	volumeClient := common.NewVolumeClient("http://localhost:8090")
	var nbdExports []*nbd.ExportInfo

	for _, attachResult := range attachResults {
		volumeID := attachResult.OSSEAVolumeID // Already a CloudStack UUID string

		log.WithFields(log.Fields{
			"job_id":      req.JobID,
			"device_path": attachResult.DevicePath,
			"volume_id":   volumeID,
		}).Info("Querying NBD export auto-created for volume")

		// Query NBD export that was auto-created during volume attachment
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		exportInfo, err := volumeClient.GetNBDExport(ctx, volumeID)
		cancel()

		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"job_id":      req.JobID,
				"device_path": attachResult.DevicePath,
				"volume_id":   volumeID,
			}).Error("Failed to query NBD export from Volume Daemon - export should have been auto-created during volume attachment")

			return nil, fmt.Errorf("failed to query auto-created NBD export for volume %s: %w", volumeID, err)
		}

		// Convert Volume Daemon NBD export to our internal format
		internalExportInfo := &nbd.ExportInfo{
			JobID:      req.JobID,
			Port:       exportInfo.Port,
			ExportName: exportInfo.ExportName,
			DevicePath: exportInfo.DevicePath,
			Status:     exportInfo.Status,
			PID:        0, // Volume Daemon manages the process
			ConfigPath: "/etc/nbd-server/config-base",
		}

		nbdExports = append(nbdExports, internalExportInfo)

		log.WithFields(log.Fields{
			"job_id":      req.JobID,
			"export_name": exportInfo.ExportName,
			"port":        exportInfo.Port,
			"device_path": exportInfo.DevicePath,
			"status":      exportInfo.Status,
		}).Info("✅ NBD export found - auto-created by Volume Daemon during volume attachment")
	}

	log.WithFields(log.Fields{
		"job_id":       req.JobID,
		"export_count": len(nbdExports),
	}).Info("✅ All auto-created NBD exports queried successfully from Volume Daemon")

	// 🎯 CRITICAL FIX: Correlate NBD exports with stable vm_disks.id
	// This also auto-repairs any broken correlations from volume detach/reattach cycles
	if err := m.correlateNBDExportsWithVMDisks(req.JobID, nbdExports, attachResults); err != nil {
		log.WithError(err).Warn("Failed to correlate NBD exports with vm_disks - proceeding without correlation")
		// Don't fail the entire migration, but log the issue for debugging
	}

	return nbdExports, nil
}

// correlateNBDExportsWithVMDisks establishes correlation between NBD exports and stable vm_disks.id
// Also auto-repairs any broken correlations from volume detach/reattach cycles
func (m *MigrationEngine) correlateNBDExportsWithVMDisks(jobID string, nbdExports []*nbd.ExportInfo, attachResults []VolumeMountResult) error {
	log.WithFields(log.Fields{
		"job_id":       jobID,
		"export_count": len(nbdExports),
		"attach_count": len(attachResults),
	}).Info("🔗 Correlating NBD exports with stable vm_disks.id for multi-disk mapping (includes auto-repair)")

	// Get vm_disks records for this job
	vmDisks, err := m.vmDiskRepo.GetByJobID(jobID)
	if err != nil {
		return fmt.Errorf("failed to get vm_disks for correlation: %w", err)
	}

	// Create volume_uuid → vm_disks.id mapping
	// Need to join with ossea_volumes to get UUID from vm_disks.ossea_volume_id
	volumeToVMDiskMap := make(map[string]int)
	for _, vmDisk := range vmDisks {
		if vmDisk.OSSEAVolumeID != 0 {
			// Get volume UUID from ossea_volumes table
			var osseaVolume database.OSSEAVolume
			err := m.db.GetGormDB().Where("id = ?", vmDisk.OSSEAVolumeID).First(&osseaVolume).Error
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"vm_disk_id":      vmDisk.ID,
					"ossea_volume_id": vmDisk.OSSEAVolumeID,
				}).Warn("Failed to get volume UUID for vm_disk correlation")
				continue
			}

			volumeToVMDiskMap[osseaVolume.VolumeID] = vmDisk.ID

			log.WithFields(log.Fields{
				"vm_disk_id":  vmDisk.ID,
				"disk_id":     vmDisk.DiskID,
				"volume_uuid": osseaVolume.VolumeID,
				"volume_name": osseaVolume.VolumeName,
			}).Debug("🔗 Created volume UUID → vm_disk correlation mapping")
		}
	}

	// Update each NBD export with corresponding vm_disk_id
	for _, export := range nbdExports {
		// Extract volume_id from export_name (format: migration-vol-{uuid})
		volumeID := strings.TrimPrefix(export.ExportName, "migration-vol-")

		if vmDiskID, exists := volumeToVMDiskMap[volumeID]; exists {
			// Update nbd_exports table with vm_disk_id correlation
			updateQuery := `UPDATE nbd_exports SET vm_disk_id = ? WHERE export_name = ? AND volume_id = ?`
			err := m.db.GetGormDB().Exec(updateQuery, vmDiskID, export.ExportName, volumeID).Error
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"export_name": export.ExportName,
					"volume_id":   volumeID,
					"vm_disk_id":  vmDiskID,
				}).Error("Failed to update NBD export with vm_disk_id correlation")
				continue
			}

			log.WithFields(log.Fields{
				"export_name": export.ExportName,
				"volume_id":   volumeID,
				"vm_disk_id":  vmDiskID,
			}).Info("✅ NBD export correlated with stable vm_disk.id")
		} else {
			log.WithFields(log.Fields{
				"export_name": export.ExportName,
				"volume_id":   volumeID,
			}).Warn("⚠️ No vm_disk correlation found for NBD export")
		}
	}

	log.WithField("job_id", jobID).Info("✅ NBD export correlation with vm_disks completed")

	// 🔧 AUTO-REPAIR: Fix any other NBD exports for this VM context that have NULL vm_disk_id
	if err := m.autoRepairNBDCorrelations(jobID); err != nil {
		log.WithError(err).Warn("Failed to auto-repair NBD correlations - some exports may have NULL vm_disk_id")
	}

	return nil
}

// getVMDiskIDFromNBDExport retrieves the stable vm_disk_id for an NBD export
func (m *MigrationEngine) getVMDiskIDFromNBDExport(exportName, volumeID string) (int, error) {
	// Query nbd_exports table for vm_disk_id (should be populated by correlateNBDExportsWithVMDisks)
	var vmDiskID int
	query := `SELECT vm_disk_id FROM nbd_exports WHERE export_name = ? AND volume_id = ? AND vm_disk_id IS NOT NULL`
	err := m.db.GetGormDB().Raw(query, exportName, volumeID).Scan(&vmDiskID).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get vm_disk_id from NBD export: %w", err)
	}

	if vmDiskID == 0 {
		return 0, fmt.Errorf("NBD export has no vm_disk_id correlation")
	}

	log.WithFields(log.Fields{
		"export_name": exportName,
		"volume_id":   volumeID,
		"vm_disk_id":  vmDiskID,
	}).Debug("✅ Retrieved stable vm_disk_id from NBD export")

	return vmDiskID, nil
}

// getVMwareDiskKeyFromVolume retrieves the VMware disk key (e.g., "2000", "2001") from volume correlation
func (m *MigrationEngine) getVMwareDiskKeyFromVolume(volumeID string) (string, error) {
	// Query vm_disks table to find the disk_id for this volume
	// disk_id format is "disk-XXXX" where XXXX is the VMware disk key
	var vmDisk database.VMDisk
	query := `
		SELECT vm_disks.disk_id FROM vm_disks 
		JOIN ossea_volumes ON vm_disks.ossea_volume_id = ossea_volumes.id 
		WHERE ossea_volumes.volume_id = ?
		LIMIT 1
	`
	err := m.db.GetGormDB().Raw(query, volumeID).Scan(&vmDisk.DiskID).Error
	if err != nil {
		return "", fmt.Errorf("failed to get disk_id for volume %s: %w", volumeID, err)
	}

	if vmDisk.DiskID == "" {
		return "", fmt.Errorf("no disk_id found for volume %s", volumeID)
	}

	// Extract VMware disk key from disk_id (format: "disk-2000" → "2000")
	vmwareDiskKey := strings.TrimPrefix(vmDisk.DiskID, "disk-")

	log.WithFields(log.Fields{
		"volume_id":       volumeID,
		"disk_id":         vmDisk.DiskID,
		"vmware_disk_key": vmwareDiskKey,
	}).Debug("🔗 Extracted VMware disk key from volume correlation")

	return vmwareDiskKey, nil
}

// autoRepairNBDCorrelations automatically repairs NULL vm_disk_id correlations for the VM context
func (m *MigrationEngine) autoRepairNBDCorrelations(jobID string) error {
	// Get VM context ID for this job
	vmContextID, err := m.getVMContextIDForJob(jobID)
	if err != nil {
		return fmt.Errorf("failed to get VM context for auto-repair: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id":        jobID,
		"vm_context_id": vmContextID,
	}).Info("🔧 Auto-repairing NBD export correlations for VM context")

	// Find NBD exports with NULL vm_disk_id for this VM context
	var brokenExports []struct {
		ExportName string
		VolumeID   string
	}

	query := `SELECT export_name, volume_id FROM nbd_exports WHERE vm_context_id = ? AND vm_disk_id IS NULL`
	rows, err := m.db.GetGormDB().Raw(query, vmContextID).Rows()
	if err != nil {
		return fmt.Errorf("failed to find broken NBD exports: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var export struct {
			ExportName string
			VolumeID   string
		}
		if err := rows.Scan(&export.ExportName, &export.VolumeID); err != nil {
			continue
		}
		brokenExports = append(brokenExports, export)
	}

	if len(brokenExports) == 0 {
		log.WithField("vm_context_id", vmContextID).Info("✅ No broken NBD correlations found - all exports properly correlated")
		return nil
	}

	log.WithFields(log.Fields{
		"vm_context_id":  vmContextID,
		"broken_exports": len(brokenExports),
	}).Info("🔧 Found broken NBD correlations - auto-repairing")

	// Get volume → vm_disk mapping for this VM context
	volumeToVMDiskMap := make(map[string]int)
	vmDisks, err := m.db.GetGormDB().Where("vm_context_id = ?", vmContextID).Find(&[]database.VMDisk{}).Rows()
	if err != nil {
		return fmt.Errorf("failed to get vm_disks for auto-repair: %w", err)
	}
	defer vmDisks.Close()

	for vmDisks.Next() {
		var vmDisk database.VMDisk
		if err := m.db.GetGormDB().ScanRows(vmDisks, &vmDisk); err != nil {
			continue
		}

		if vmDisk.OSSEAVolumeID != 0 {
			// Get volume UUID from ossea_volumes
			var osseaVolume database.OSSEAVolume
			if err := m.db.GetGormDB().Where("id = ?", vmDisk.OSSEAVolumeID).First(&osseaVolume).Error; err == nil {
				volumeToVMDiskMap[osseaVolume.VolumeID] = vmDisk.ID
			}
		}
	}

	// Repair each broken export
	repairedCount := 0
	for _, brokenExport := range brokenExports {
		if vmDiskID, exists := volumeToVMDiskMap[brokenExport.VolumeID]; exists {
			updateQuery := `UPDATE nbd_exports SET vm_disk_id = ? WHERE export_name = ? AND volume_id = ? AND vm_disk_id IS NULL`
			err := m.db.GetGormDB().Exec(updateQuery, vmDiskID, brokenExport.ExportName, brokenExport.VolumeID).Error
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"export_name": brokenExport.ExportName,
					"volume_id":   brokenExport.VolumeID,
				}).Error("Failed to repair NBD export correlation")
				continue
			}

			log.WithFields(log.Fields{
				"export_name": brokenExport.ExportName,
				"volume_id":   brokenExport.VolumeID,
				"vm_disk_id":  vmDiskID,
			}).Info("🔧 Auto-repaired NBD export correlation")
			repairedCount++
		}
	}

	log.WithFields(log.Fields{
		"vm_context_id":  vmContextID,
		"repaired_count": repairedCount,
		"broken_count":   len(brokenExports),
	}).Info("✅ NBD correlation auto-repair completed")

	return nil
}

// NOTE: NBD export cleanup functions removed - Volume Daemon handles cleanup automatically
// When volumes are detached, the Volume Daemon automatically removes the corresponding NBD exports
// This eliminates the need for manual cleanup logic in the migration workflow

// verifyDevicePaths checks that device paths from CloudStack API actually exist
// and corrects them if necessary (CloudStack API paths don't always match Linux reality)
func (m *MigrationEngine) verifyDevicePaths(attachResults []VolumeMountResult) ([]VolumeMountResult, error) {
	log.WithField("volume_count", len(attachResults)).Info("Verifying device paths exist")

	for i, result := range attachResults {
		originalPath := result.DevicePath

		// Check if the reported device path actually exists
		if _, err := os.Stat(originalPath); err == nil {
			log.WithField("device_path", originalPath).Info("✅ Device path verified - exists as reported")
			continue
		}

		log.WithField("device_path", originalPath).Warn("⚠️  Device path from CloudStack API does not exist, searching for actual device")

		// Find the actual device by looking for recent block devices
		actualPath, err := m.findActualDevicePath(originalPath)
		if err != nil {
			return nil, fmt.Errorf("failed to find actual device for %s: %w", originalPath, err)
		}

		log.WithFields(log.Fields{
			"original_path": originalPath,
			"actual_path":   actualPath,
		}).Info("✅ Device path corrected")

		// Update the result with the correct path
		attachResults[i].DevicePath = actualPath
	}

	return attachResults, nil
}

// findActualDevicePath attempts to find the real Linux device path
// when CloudStack API returns incorrect path (common issue)
func (m *MigrationEngine) findActualDevicePath(reportedPath string) (string, error) {
	// Strategy 1: Look for the most recent block device that appeared
	// This works because we just attached the volume
	cmd := exec.Command("find", "/dev", "-name", "vd*", "-type", "b", "-newer", "/dev/vda")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to find recent block devices: %w", err)
	}

	devices := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(devices) == 0 || (len(devices) == 1 && devices[0] == "") {
		return "", fmt.Errorf("no recent block devices found")
	}

	// Sort by modification time to get the most recent
	sort.Slice(devices, func(i, j int) bool {
		statI, errI := os.Stat(devices[i])
		statJ, errJ := os.Stat(devices[j])
		if errI != nil || errJ != nil {
			return false
		}
		return statI.ModTime().After(statJ.ModTime())
	})

	// Return the most recent device
	actualPath := devices[0]
	log.WithFields(log.Fields{
		"reported_path": reportedPath,
		"actual_path":   actualPath,
		"search_count":  len(devices),
	}).Info("Found actual device path")

	return actualPath, nil
}

// findExistingVolumeForVMDisk checks for existing OSSEA volumes for this VM disk
func (m *MigrationEngine) findExistingVolumeForVMDisk(vmPath string, unitNumber int, sizeGB int) (*database.OSSEAVolume, error) {
	// Extract VM name from path for consistent volume naming
	vmName := vmPath[strings.LastIndex(vmPath, "/")+1:]

	log.WithFields(log.Fields{
		"vm_path":     vmPath,
		"vm_name":     vmName,
		"unit_number": unitNumber,
		"size_gb":     sizeGB,
	}).Info("🔍 DEBUG: Checking for existing volume to reuse")

	// Try new naming pattern first: migration-{VMName}-{VMName}-disk-{UnitNumber}
	newPattern := fmt.Sprintf("migration-%s-%s-disk-%d", vmName, vmName, unitNumber)

	// Query database for existing volumes with new pattern (reuse existing volume regardless of status)
	var volumes []database.OSSEAVolume
	log.WithFields(log.Fields{
		"pattern": newPattern,
		"size_gb": sizeGB,
	}).Info("🔍 DEBUG: Executing volume query")

	if err := m.db.GetGormDB().Where("volume_name = ? AND size_gb = ?", newPattern, sizeGB).Find(&volumes).Error; err != nil {
		log.WithError(err).Error("🚨 DEBUG: Database query failed")
		return nil, fmt.Errorf("failed to query existing volumes: %w", err)
	}

	log.WithFields(log.Fields{
		"volumes_found": len(volumes),
		"pattern":       newPattern,
	}).Info("🔍 DEBUG: Query results")

	if len(volumes) > 0 {
		volume := &volumes[0]
		log.WithFields(log.Fields{
			"vm_path":     vmPath,
			"unit_number": unitNumber,
			"volume_id":   volume.VolumeID,
			"volume_name": volume.VolumeName,
			"created_at":  volume.CreatedAt,
			"pattern":     "new",
		}).Info("♻️  Found existing volume with new naming pattern")
		return volume, nil
	}

	// Try old job-based pattern: migration-job-%-disk-{UnitNumber} for this VM
	// First, find completed jobs for this VM path
	var jobs []database.ReplicationJob
	if err := m.db.GetGormDB().Where("source_vm_path = ? AND status IN (?)", vmPath, []string{"completed", "replicating"}).Order("created_at DESC").Find(&jobs).Error; err != nil {
		log.WithError(err).Warn("Failed to query previous jobs for volume reuse")
	} else {
		// Look for volumes from previous jobs
		for _, job := range jobs {
			oldPattern := fmt.Sprintf("migration-%s-disk-%d", job.ID, unitNumber)
			if err := m.db.GetGormDB().Where("volume_name = ? AND size_gb = ?", oldPattern, sizeGB).Find(&volumes).Error; err == nil && len(volumes) > 0 {
				volume := &volumes[0]
				log.WithFields(log.Fields{
					"vm_path":     vmPath,
					"unit_number": unitNumber,
					"volume_id":   volume.VolumeID,
					"volume_name": volume.VolumeName,
					"created_at":  volume.CreatedAt,
					"pattern":     "old_job",
					"source_job":  job.ID,
				}).Info("♻️  Found existing volume with old job naming pattern")
				return volume, nil
			}
		}
	}

	log.WithFields(log.Fields{
		"vm_path":     vmPath,
		"vm_name":     vmName,
		"unit_number": unitNumber,
		"size_gb":     sizeGB,
		"new_pattern": newPattern,
	}).Debug("No existing volume found for VM disk")
	return nil, nil
}

// GetVMDisksByJobID retrieves VM disk records for a specific job
func (m *MigrationEngine) GetVMDisksByJobID(jobID string) ([]database.VMDisk, error) {
	return m.vmDiskRepo.GetByJobID(jobID)
}

// UpdateVMDiskChangeID updates the ChangeID for a specific VM disk
func (m *MigrationEngine) UpdateVMDiskChangeID(diskID int, changeID string) error {
	return m.vmDiskRepo.UpdateChangeID(diskID, changeID)
}

// StoreCBTHistory creates a CBT history record
func (m *MigrationEngine) StoreCBTHistory(jobID, diskID, changeID, previousChangeID, syncType string, syncSuccess bool) error {
	// Get VM context ID for this job
	vmContextID, err := m.getVMContextIDForJob(jobID)
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Warn("Failed to get VM context ID for CBT history, continuing without it")
		vmContextID = "" // Continue without VM context for backward compatibility
	}

	cbtHistory := &database.CBTHistory{
		JobID:            jobID,
		VMContextID:      vmContextID, // VM-Centric Architecture integration
		DiskID:           diskID,
		ChangeID:         changeID,
		PreviousChangeID: previousChangeID,
		SyncType:         syncType,
		SyncSuccess:      syncSuccess,
		CreatedAt:        time.Now(),
	}

	return m.cbtHistoryRepo.Create(cbtHistory)
}

// getVMContextIDForJob retrieves the VM context ID for a given job ID
func (m *MigrationEngine) getVMContextIDForJob(jobID string) (string, error) {
	var vmContextID string
	err := m.db.GetGormDB().Table("replication_jobs").
		Select("vm_context_id").
		Where("id = ?", jobID).
		Scan(&vmContextID).Error

	if err != nil {
		return "", fmt.Errorf("failed to get VM context ID for job %s: %w", jobID, err)
	}

	return vmContextID, nil
}
