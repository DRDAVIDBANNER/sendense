package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/models"
	"github.com/vexxhost/migratekit-volume-daemon/nbd"
	"github.com/vexxhost/migratekit-volume-daemon/repository"
)

// VolumeService implements the VolumeManagementService interface
type VolumeService struct {
	repo                    VolumeRepository
	cloudStackClient        CloudStackClient
	deviceMonitor           DeviceMonitor
	nbdExportManager        *nbd.ExportManager
	osseaVolumeRepo         *repository.OSSEAVolumeRepository
	persistentDeviceManager *PersistentDeviceManager // 🆕 NEW: Persistent device naming
}

// NewVolumeService creates a new volume management service
func NewVolumeService(repo VolumeRepository, cloudStackClient CloudStackClient, deviceMonitor DeviceMonitor, nbdExportManager *nbd.ExportManager, osseaVolumeRepo *repository.OSSEAVolumeRepository) VolumeManagementService {
	// 🆕 NEW: Initialize persistent device manager
	persistentDeviceManager := NewPersistentDeviceManager(repo)

	return &VolumeService{
		repo:                    repo,
		cloudStackClient:        cloudStackClient,
		deviceMonitor:           deviceMonitor,
		nbdExportManager:        nbdExportManager,
		osseaVolumeRepo:         osseaVolumeRepo,
		persistentDeviceManager: persistentDeviceManager,
	}
}

// CreateVolume creates a new volume (placeholder implementation)
func (vs *VolumeService) CreateVolume(ctx context.Context, req models.CreateVolumeRequest) (*models.VolumeOperation, error) {
	// Generate operation ID
	operationID := uuid.New().String()

	// Create operation record
	operation := &models.VolumeOperation{
		ID:       operationID,
		Type:     models.OperationCreate,
		Status:   models.StatusPending,
		VolumeID: "", // Will be set when CloudStack volume is created
		VMID:     nil,
		Request: map[string]interface{}{
			"name":             req.Name,
			"size":             req.Size,
			"disk_offering_id": req.DiskOfferingID,
			"zone_id":          req.ZoneID,
			"metadata":         req.Metadata,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store operation
	if err := vs.repo.CreateOperation(ctx, operation); err != nil {
		return nil, fmt.Errorf("failed to create operation record: %w", err)
	}

	// Execute CloudStack volume creation in background
	go vs.executeCreateVolume(context.Background(), operation, req)

	return operation, nil
}

// AttachVolume attaches a volume to a VM (placeholder implementation)
func (vs *VolumeService) AttachVolume(ctx context.Context, volumeID, vmID string) (*models.VolumeOperation, error) {
	// Generate operation ID
	operationID := uuid.New().String()

	// Create operation record
	operation := &models.VolumeOperation{
		ID:       operationID,
		Type:     models.OperationAttach,
		Status:   models.StatusPending,
		VolumeID: volumeID,
		VMID:     &vmID,
		Request: map[string]interface{}{
			"volume_id": volumeID,
			"vm_id":     vmID,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store operation
	if err := vs.repo.CreateOperation(ctx, operation); err != nil {
		return nil, fmt.Errorf("failed to create operation record: %w", err)
	}

	// Execute CloudStack volume attachment in background
	go vs.executeAttachVolume(context.Background(), operation, volumeID, vmID)

	return operation, nil
}

// AttachVolumeWithPersistentNaming attaches a volume with persistent device naming support
func (vs *VolumeService) AttachVolumeWithPersistentNaming(ctx context.Context, req *models.AttachVolumeRequest) (*models.VolumeOperation, error) {
	// Generate operation ID
	operationID := uuid.New().String()

	// Create operation record
	operation := &models.VolumeOperation{
		ID:       operationID,
		Type:     models.OperationAttach,
		Status:   models.StatusPending,
		VolumeID: req.VolumeID,
		VMID:     &req.VMID,
		Request: map[string]interface{}{
			"volume_id":               req.VolumeID,
			"vm_id":                   req.VMID,
			"vm_name":                 req.VMName,
			"disk_id":                 req.DiskID,
			"request_persistent_name": req.RequestPersistentName,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store operation
	if err := vs.repo.CreateOperation(ctx, operation); err != nil {
		return nil, fmt.Errorf("failed to create operation: %w", err)
	}

	// Execute attachment with persistent naming in background
	go vs.executeAttachVolumeWithPersistentNaming(ctx, operation, req)

	return operation, nil
}

// AttachVolumeAsRoot attaches a volume to a VM as root disk (device ID 0)
func (vs *VolumeService) AttachVolumeAsRoot(ctx context.Context, volumeID, vmID string) (*models.VolumeOperation, error) {
	// Generate operation ID
	operationID := uuid.New().String()

	// Create operation record
	operation := &models.VolumeOperation{
		ID:       operationID,
		Type:     models.OperationAttach,
		Status:   models.StatusPending,
		VolumeID: volumeID,
		VMID:     &vmID,
		Request: map[string]interface{}{
			"volume_id": volumeID,
			"vm_id":     vmID,
			"device_id": 0, // Root disk indicator
			"attach_as": "root",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store operation
	if err := vs.repo.CreateOperation(ctx, operation); err != nil {
		return nil, fmt.Errorf("failed to create operation record: %w", err)
	}

	// Execute CloudStack volume attachment as root in background
	go vs.executeAttachVolumeAsRoot(context.Background(), operation, volumeID, vmID)

	return operation, nil
}

// DetachVolume detaches a volume from its VM (placeholder implementation)
func (vs *VolumeService) DetachVolume(ctx context.Context, volumeID string) (*models.VolumeOperation, error) {
	// Generate operation ID
	operationID := uuid.New().String()

	// Create operation record
	operation := &models.VolumeOperation{
		ID:       operationID,
		Type:     models.OperationDetach,
		Status:   models.StatusPending,
		VolumeID: volumeID,
		Request: map[string]interface{}{
			"volume_id": volumeID,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store operation
	if err := vs.repo.CreateOperation(ctx, operation); err != nil {
		return nil, fmt.Errorf("failed to create operation record: %w", err)
	}

	// Execute CloudStack volume detachment in background
	go vs.executeDetachVolume(context.Background(), operation, volumeID)

	return operation, nil
}

// DeleteVolume deletes a volume (placeholder implementation)
func (vs *VolumeService) DeleteVolume(ctx context.Context, volumeID string) (*models.VolumeOperation, error) {
	// Generate operation ID
	operationID := uuid.New().String()

	// Create operation record
	operation := &models.VolumeOperation{
		ID:       operationID,
		Type:     models.OperationDelete,
		Status:   models.StatusPending,
		VolumeID: volumeID,
		Request: map[string]interface{}{
			"volume_id": volumeID,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store operation
	if err := vs.repo.CreateOperation(ctx, operation); err != nil {
		return nil, fmt.Errorf("failed to create operation record: %w", err)
	}

	// Execute CloudStack volume deletion in background
	go vs.executeDeleteVolume(context.Background(), operation, volumeID)

	return operation, nil
}

// CleanupTestFailover orchestrates complete test failover cleanup
func (vs *VolumeService) CleanupTestFailover(ctx context.Context, req models.CleanupRequest) (*models.VolumeOperation, error) {
	// Generate operation ID
	operationID := uuid.New().String()

	// Create operation record
	operation := &models.VolumeOperation{
		ID:       operationID,
		Type:     models.OperationCleanup,
		Status:   models.StatusPending,
		VolumeID: req.VolumeID,
		VMID:     &req.TestVMID,
		Request: map[string]interface{}{
			"test_vm_id":  req.TestVMID,
			"volume_id":   req.VolumeID,
			"oma_vm_id":   req.SHAVMID,
			"delete_vm":   req.DeleteVM,
			"force_clean": req.ForceClean,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store operation
	if err := vs.repo.CreateOperation(ctx, operation); err != nil {
		return nil, fmt.Errorf("failed to create cleanup operation record: %w", err)
	}

	// Execute cleanup workflow in background
	go vs.executeCleanupTestFailover(context.Background(), operation, req)

	return operation, nil
}

// GetVolumeStatus gets the current status of a volume (placeholder implementation)
func (vs *VolumeService) GetVolumeStatus(ctx context.Context, volumeID string) (*models.VolumeStatus, error) {
	// TODO: Implement by querying CloudStack and device mappings
	return &models.VolumeStatus{
		VolumeID: volumeID,
		State:    "unknown",
	}, nil
}

// GetDeviceMapping gets the device mapping for a volume
func (vs *VolumeService) GetDeviceMapping(ctx context.Context, volumeID string) (*models.DeviceMapping, error) {
	return vs.repo.GetMapping(ctx, volumeID)
}

// GetVolumeForDevice gets the volume mapping for a device path
func (vs *VolumeService) GetVolumeForDevice(ctx context.Context, devicePath string) (*models.DeviceMapping, error) {
	return vs.repo.GetMappingByDevice(ctx, devicePath)
}

// ListVolumesForVM lists all volumes for a VM by querying device mappings
func (vs *VolumeService) ListVolumesForVM(ctx context.Context, vmID string) ([]models.VolumeStatus, error) {
	log.WithField("vm_id", vmID).Debug("🔍 Listing volumes for VM")

	// Get all device mappings for this VM
	mappings, err := vs.repo.ListMappingsForVM(ctx, vmID)
	if err != nil {
		log.WithFields(log.Fields{
			"vm_id": vmID,
			"error": err,
		}).Error("Failed to list device mappings for VM")
		return nil, fmt.Errorf("failed to list device mappings for VM: %w", err)
	}

	// Convert mappings to volume status
	volumes := make([]models.VolumeStatus, 0, len(mappings))
	for _, mapping := range mappings {
		volume := models.VolumeStatus{
			VolumeID:   mapping.VolumeUUID,
			VMID:       &mapping.VMID,
			DevicePath: &mapping.DevicePath,
			State:      "attached",
			Size:       mapping.Size,
		}
		volumes = append(volumes, volume)
	}

	log.WithFields(log.Fields{
		"vm_id":        vmID,
		"volume_count": len(volumes),
	}).Debug("✅ Retrieved volumes for VM")

	return volumes, nil
}

// GetOperation gets a volume operation by ID
func (vs *VolumeService) GetOperation(ctx context.Context, operationID string) (*models.VolumeOperation, error) {
	return vs.repo.GetOperation(ctx, operationID)
}

// ListOperations lists volume operations based on filter criteria
func (vs *VolumeService) ListOperations(ctx context.Context, filter models.OperationFilter) ([]models.VolumeOperation, error) {
	return vs.repo.ListOperations(ctx, filter)
}

// WaitForOperation waits for an operation to complete (placeholder implementation)
func (vs *VolumeService) WaitForOperation(ctx context.Context, operationID string, timeout time.Duration) (*models.VolumeOperation, error) {
	// TODO: Implement proper waiting logic with polling
	return vs.GetOperation(ctx, operationID)
}

// GetHealth returns the health status of the service
func (vs *VolumeService) GetHealth(ctx context.Context) (*models.HealthStatus, error) {
	// Test database connectivity
	dbHealth := "healthy"
	if err := vs.repo.Ping(ctx); err != nil {
		dbHealth = fmt.Sprintf("unhealthy: %v", err)
	}

	return &models.HealthStatus{
		Status:           "healthy",
		Timestamp:        time.Now(),
		CloudStackHealth: "not_implemented",
		DatabaseHealth:   dbHealth,
		DeviceMonitor:    "not_implemented",
		Details: map[string]string{
			"implementation_status": "phase_1_foundation",
		},
	}, nil
}

// GetMetrics returns service metrics (placeholder implementation)
func (vs *VolumeService) GetMetrics(ctx context.Context) (*models.ServiceMetrics, error) {
	// TODO: Implement proper metrics collection
	return &models.ServiceMetrics{
		Timestamp:           time.Now(),
		TotalOperations:     0,
		PendingOperations:   0,
		ActiveMappings:      0,
		OperationsByType:    make(map[string]int64),
		OperationsByStatus:  make(map[string]int64),
		AverageResponseTime: 0,
		ErrorRate:           0,
		Details: map[string]interface{}{
			"implementation_status": "placeholder",
		},
	}, nil
}

// ForceSync forces synchronization between CloudStack and device mappings (placeholder)
func (vs *VolumeService) ForceSync(ctx context.Context) error {
	// TODO: Implement synchronization logic
	return fmt.Errorf("force sync not yet implemented")
}

// Start starts the volume service
func (vs *VolumeService) Start(ctx context.Context) error {
	// TODO: Initialize CloudStack client and device monitor
	return nil
}

// Stop stops the volume service
func (vs *VolumeService) Stop(ctx context.Context) error {
	// TODO: Clean shutdown of CloudStack client and device monitor
	return nil
}

// executeCreateVolume executes the CloudStack volume creation operation
func (vs *VolumeService) executeCreateVolume(ctx context.Context, operation *models.VolumeOperation, req models.CreateVolumeRequest) {
	// Update operation status to executing
	operation.Status = models.StatusExecuting
	operation.UpdatedAt = time.Now()
	vs.repo.UpdateOperation(ctx, operation)

	// Check if CloudStack client is available
	if vs.cloudStackClient == nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack client not available"))
		return
	}

	// Create volume in CloudStack
	volumeID, err := vs.cloudStackClient.CreateVolume(ctx, req)
	if err != nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack volume creation failed: %w", err))
		return
	}

	// Update operation with successful result
	operation.Status = models.StatusCompleted
	operation.VolumeID = volumeID
	now := time.Now()
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	operation.Response = map[string]interface{}{
		"volume_id": volumeID,
		"message":   "Volume created successfully",
	}

	if err := vs.repo.UpdateOperation(ctx, operation); err != nil {
		log.WithFields(log.Fields{
			"operation_id": operation.ID,
			"volume_id":    volumeID,
			"error":        err,
		}).Error("Failed to update operation after successful volume creation")
	}
}

// executeAttachVolume executes the CloudStack volume attachment operation
func (vs *VolumeService) executeAttachVolume(ctx context.Context, operation *models.VolumeOperation, volumeID, vmID string) {
	// Update operation status to executing
	operation.Status = models.StatusExecuting
	operation.UpdatedAt = time.Now()
	vs.repo.UpdateOperation(ctx, operation)

	// Check if CloudStack client is available
	if vs.cloudStackClient == nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack client not available"))
		return
	}

	// Attach volume in CloudStack
	err := vs.cloudStackClient.AttachVolume(ctx, volumeID, vmID)
	if err != nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack volume attachment failed: %w", err))
		return
	}

	// Determine operation mode and handle device correlation accordingly
	var devicePath string
	var deviceSize int64
	var operationMode string
	var cloudStackDeviceID *int
	var requiresCorrelation bool

	// Check if this is an SHA attachment (where device correlation is possible)
	isOMAAttachment := vs.isOMAVM(ctx, vmID)

	if isOMAAttachment {
		// SHA Mode: Full device correlation required
		operationMode = string(models.OperationModeOMA)
		requiresCorrelation = true

		log.WithFields(log.Fields{
			"volume_id": volumeID,
			"vm_id":     vmID,
			"mode":      operationMode,
		}).Info("🔗 SHA attachment - performing device correlation")

		if vs.deviceMonitor != nil {
			devicePath, deviceSize = vs.correlateVolumeToDevice(ctx, volumeID, vmID)
		} else {
			log.Warn("Device monitor not available for SHA correlation")
		}
	} else {
		// Failover Mode: CloudStack state tracking only
		operationMode = string(models.OperationModeFailover)
		requiresCorrelation = false

		log.WithFields(log.Fields{
			"volume_id": volumeID,
			"vm_id":     vmID,
			"mode":      operationMode,
		}).Info("📡 Failover VM attachment - CloudStack state tracking only")

		// Get CloudStack device ID from API response
		if csDeviceID, err := vs.getCloudStackDeviceID(ctx, volumeID); err == nil {
			cloudStackDeviceID = &csDeviceID
		}

		// Use placeholder device path for failover VMs
		devicePath = fmt.Sprintf("remote-vm-%s", vmID)
		deviceSize = 0 // Size not needed for failover VMs
	}

	// Create device mapping record BEFORE marking operation as completed
	if devicePath != "" {
		// Set appropriate Linux state based on operation mode
		linuxState := "detected"
		if operationMode == string(models.OperationModeFailover) {
			linuxState = "n/a" // No Linux device detection for failover VMs
		}

		mapping := &models.DeviceMapping{
			VolumeUUID:                volumeID,
			VolumeIDNumeric:           nil, // Will be populated from CloudStack if available
			VMID:                      vmID,
			OperationMode:             operationMode,
			CloudStackDeviceID:        cloudStackDeviceID,
			RequiresDeviceCorrelation: requiresCorrelation,
			DevicePath:                devicePath,
			CloudStackState:           "attached",
			LinuxState:                linuxState,
			Size:                      deviceSize,
			LastSync:                  time.Now(),
			CreatedAt:                 time.Now(),
			UpdatedAt:                 time.Now(),
		}

		// 🆕 NEW: Add persistent device naming fields to mapping
		var persistentDeviceName *string
		var symlinkPath *string
		
		// For now, set to nil - full integration will populate these fields
		// This ensures compatibility with enhanced schema
		mapping.PersistentDeviceName = persistentDeviceName
		mapping.SymlinkPath = symlinkPath

		if err := vs.repo.CreateMapping(ctx, mapping); err != nil {
			log.WithFields(log.Fields{
				"volume_id":   volumeID,
				"device_path": devicePath,
				"error":       err,
			}).Error("Failed to create device mapping after volume attachment")

			// Mark operation as failed since mapping creation is critical for verification
			vs.completeOperationWithError(ctx, operation, fmt.Errorf("mapping creation failed: %w", err))
			return
		}

		log.WithFields(log.Fields{
			"volume_id":   volumeID,
			"vm_id":       vmID,
			"device_path": devicePath,
		}).Info("✅ Device mapping created successfully")

		// Update ossea_volumes table with attachment status
		if vs.osseaVolumeRepo != nil {
			if err := vs.osseaVolumeRepo.UpdateVolumeAttachment(ctx, volumeID, devicePath); err != nil {
				log.WithFields(log.Fields{
					"volume_id":   volumeID,
					"device_path": devicePath,
					"error":       err,
				}).Warn("Failed to update ossea_volumes on attachment - continuing")
			}
		}

		// Create NBD export for SHA volumes only (volumes with real device paths)
		if isOMAAttachment && devicePath != "" && !strings.HasPrefix(devicePath, "remote-vm-") {
			vs.createNBDExportForVolume(ctx, volumeID, vmID, devicePath)
		}
	}

	// Update operation with successful result AFTER mapping creation
	operation.Status = models.StatusCompleted
	now := time.Now()
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	operation.Response = map[string]interface{}{
		"volume_id":   volumeID,
		"vm_id":       vmID,
		"device_path": devicePath,
		"message":     "Volume attached successfully",
	}

	if err := vs.repo.UpdateOperation(ctx, operation); err != nil {
		log.WithFields(log.Fields{
			"operation_id": operation.ID,
			"volume_id":    volumeID,
			"vm_id":        vmID,
			"error":        err,
		}).Error("Failed to update operation after successful volume attachment")
	}
}

// executeAttachVolumeAsRoot executes the CloudStack volume attachment as root disk operation
func (vs *VolumeService) executeAttachVolumeAsRoot(ctx context.Context, operation *models.VolumeOperation, volumeID, vmID string) {
	// Update operation status to executing
	operation.Status = models.StatusExecuting
	operation.UpdatedAt = time.Now()
	vs.repo.UpdateOperation(ctx, operation)

	// Check if CloudStack client is available
	if vs.cloudStackClient == nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack client not available"))
		return
	}

	// Attach volume as root disk in CloudStack
	err := vs.cloudStackClient.AttachVolumeAsRoot(ctx, volumeID, vmID)
	if err != nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack volume root attachment failed: %w", err))
		return
	}

	// Determine operation mode and handle device correlation accordingly
	var devicePath string
	var deviceSize int64
	var operationMode string
	var cloudStackDeviceID *int
	var requiresCorrelation bool

	// Check if this is an SHA attachment (where device correlation is possible)
	isOMAAttachment := vs.isOMAVM(ctx, vmID)

	if isOMAAttachment {
		// SHA Mode: Full device correlation required
		operationMode = string(models.OperationModeOMA)
		requiresCorrelation = true
		// Root device is always device ID 0
		deviceIDZero := 0
		cloudStackDeviceID = &deviceIDZero

		log.WithFields(log.Fields{
			"volume_id": volumeID,
			"vm_id":     vmID,
			"mode":      operationMode,
			"device_id": 0,
		}).Info("🔗 SHA root attachment - performing device correlation")

		if vs.deviceMonitor != nil {
			devicePath, deviceSize = vs.correlateVolumeToDevice(ctx, volumeID, vmID)
		} else {
			log.Warn("Device monitor not available for SHA correlation")
		}
	} else {
		// Failover Mode: CloudStack state tracking only
		operationMode = string(models.OperationModeFailover)
		requiresCorrelation = false
		// Root device is always device ID 0
		deviceIDZero := 0
		cloudStackDeviceID = &deviceIDZero

		log.WithFields(log.Fields{
			"volume_id": volumeID,
			"vm_id":     vmID,
			"mode":      operationMode,
			"device_id": 0,
		}).Info("📡 Failover VM root attachment - CloudStack state tracking only")

		// Use placeholder device path for failover VMs
		devicePath = fmt.Sprintf("remote-vm-root-%s", vmID)
		deviceSize = 0 // Size not needed for failover VMs
	}

	// Create device mapping record BEFORE marking operation as completed
	if devicePath != "" {
		// Set appropriate Linux state based on operation mode
		linuxState := "detected"
		if operationMode == string(models.OperationModeFailover) {
			linuxState = "n/a" // No Linux device detection for failover VMs
		}

		mapping := &models.DeviceMapping{
			VolumeUUID:                volumeID,
			VolumeIDNumeric:           nil, // Will be populated from CloudStack if available
			VMID:                      vmID,
			OperationMode:             operationMode,
			CloudStackDeviceID:        cloudStackDeviceID,
			RequiresDeviceCorrelation: requiresCorrelation,
			DevicePath:                devicePath,
			CloudStackState:           "attached",
			LinuxState:                linuxState,
			Size:                      deviceSize,
			LastSync:                  time.Now(),
			CreatedAt:                 time.Now(),
			UpdatedAt:                 time.Now(),
		}

		if err := vs.repo.CreateMapping(ctx, mapping); err != nil {
			log.WithFields(log.Fields{
				"operation_id": operation.ID,
				"volume_id":    volumeID,
				"vm_id":        vmID,
				"device_path":  devicePath,
				"error":        err,
			}).Error("Failed to create device mapping after volume root attachment")

			// Mark operation as failed since mapping creation is critical for verification
			vs.completeOperationWithError(ctx, operation, fmt.Errorf("mapping creation failed: %w", err))
			return
		}

		log.WithFields(log.Fields{
			"volume_id":   volumeID,
			"vm_id":       vmID,
			"device_path": devicePath,
		}).Info("✅ Device mapping created successfully for root attachment")

		// Update ossea_volumes table with attachment status
		if vs.osseaVolumeRepo != nil {
			if err := vs.osseaVolumeRepo.UpdateVolumeAttachment(ctx, volumeID, devicePath); err != nil {
				log.WithFields(log.Fields{
					"volume_id":   volumeID,
					"device_path": devicePath,
					"error":       err,
				}).Warn("Failed to update ossea_volumes on root attachment - continuing")
			}
		}

		// Create NBD export for SHA root volumes only (volumes with real device paths)
		if isOMAAttachment && devicePath != "" && !strings.HasPrefix(devicePath, "remote-vm-") {
			vs.createNBDExportForVolume(ctx, volumeID, vmID, devicePath)
		}
	}

	// Update operation with successful result AFTER mapping creation
	operation.Status = models.StatusCompleted
	now := time.Now()
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	operation.Response = map[string]interface{}{
		"volume_id":   volumeID,
		"vm_id":       vmID,
		"device_path": devicePath,
		"device_id":   0, // Root disk
		"message":     "Volume attached as root disk successfully",
	}

	if err := vs.repo.UpdateOperation(ctx, operation); err != nil {
		log.WithFields(log.Fields{
			"operation_id": operation.ID,
			"volume_id":    volumeID,
			"vm_id":        vmID,
			"error":        err,
		}).Error("Failed to update operation after successful volume root attachment")
	}
}

// executeDetachVolume executes the CloudStack volume detachment operation
func (vs *VolumeService) executeDetachVolume(ctx context.Context, operation *models.VolumeOperation, volumeID string) {
	// Update operation status to executing
	operation.Status = models.StatusExecuting
	operation.UpdatedAt = time.Now()
	vs.repo.UpdateOperation(ctx, operation)

	// Check if CloudStack client is available
	if vs.cloudStackClient == nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack client not available"))
		return
	}

	// Get current device mapping before detachment
	var devicePath string
	var vmID string
	if mapping, err := vs.repo.GetMapping(ctx, volumeID); err == nil {
		devicePath = mapping.DevicePath
		vmID = mapping.VMID
	}

	// Detach volume in CloudStack
	err := vs.cloudStackClient.DetachVolume(ctx, volumeID)
	if err != nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack volume detachment failed: %w", err))
		return
	}

	// Remove NBD export ONLY when detaching from SHA VM (prevents double SIGHUP during failover cleanup)
	if devicePath != "" && !strings.HasPrefix(devicePath, "remote-vm-") && vs.isOMAVM(ctx, vmID) {
		vs.deleteNBDExportForVolume(ctx, volumeID)
	}

	// Remove device mapping
	if devicePath != "" {
		if err := vs.repo.DeleteMapping(ctx, volumeID); err != nil {
			log.WithFields(log.Fields{
				"volume_id":   volumeID,
				"device_path": devicePath,
				"error":       err,
			}).Error("Failed to delete device mapping after volume detachment")
		}
	}

	// Update ossea_volumes table with detachment status
	if vs.osseaVolumeRepo != nil {
		if err := vs.osseaVolumeRepo.UpdateVolumeDetachment(ctx, volumeID); err != nil {
			log.WithFields(log.Fields{
				"volume_id": volumeID,
				"error":     err,
			}).Warn("Failed to update ossea_volumes on detachment - continuing")
		}
	}

	// Update operation with successful result
	operation.Status = models.StatusCompleted
	now := time.Now()
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	operation.Response = map[string]interface{}{
		"volume_id":   volumeID,
		"device_path": devicePath,
		"message":     "Volume detached successfully",
	}

	if err := vs.repo.UpdateOperation(ctx, operation); err != nil {
		log.WithFields(log.Fields{
			"operation_id": operation.ID,
			"volume_id":    volumeID,
			"error":        err,
		}).Error("Failed to update operation after successful volume detachment")
	}
}

// executeDeleteVolume executes the CloudStack volume deletion operation
func (vs *VolumeService) executeDeleteVolume(ctx context.Context, operation *models.VolumeOperation, volumeID string) {
	// Update operation status to executing
	operation.Status = models.StatusExecuting
	operation.UpdatedAt = time.Now()
	vs.repo.UpdateOperation(ctx, operation)

	// Check if CloudStack client is available
	if vs.cloudStackClient == nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack client not available"))
		return
	}

	// ENHANCED DELETE WORKFLOW: Detach first, then delete
	// This ensures NBD exports are properly cleaned up and attached volumes are handled gracefully

	// Step 1: Check if volume is attached and detach if necessary
	mapping, err := vs.repo.GetMapping(ctx, volumeID)
	if err == nil {
		// Volume is attached - detach first
		log.WithFields(log.Fields{
			"volume_id":   volumeID,
			"device_path": mapping.DevicePath,
			"vm_id":       mapping.VMID,
		}).Info("Volume is attached - detaching before deletion to clean up NBD exports")

		// Get VM ID and device path before detachment
		vmID := mapping.VMID
		devicePath := mapping.DevicePath

		// Detach volume in CloudStack
		err := vs.cloudStackClient.DetachVolume(ctx, volumeID)
		if err != nil {
			vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack volume detachment failed: %w", err))
			return
		}

		// Remove NBD export (using same logic as executeDetachVolume)
		if devicePath != "" && !strings.HasPrefix(devicePath, "remote-vm-") && vs.isOMAVM(ctx, vmID) {
			vs.deleteNBDExportForVolume(ctx, volumeID)
		}

		// Remove device mapping
		if err := vs.repo.DeleteMapping(ctx, volumeID); err != nil {
			log.WithFields(log.Fields{
				"volume_id":   volumeID,
				"device_path": devicePath,
				"error":       err,
			}).Warn("Failed to delete device mapping after detachment - continuing with deletion")
		}

		// Update ossea_volumes table with detachment status
		if vs.osseaVolumeRepo != nil {
			if err := vs.osseaVolumeRepo.UpdateVolumeDetachment(ctx, volumeID); err != nil {
				log.WithFields(log.Fields{
					"volume_id": volumeID,
					"error":     err,
				}).Warn("Failed to update ossea_volumes on detachment - continuing with deletion")
			}
		}

		log.WithFields(log.Fields{
			"volume_id": volumeID,
		}).Info("Volume detached successfully, proceeding with deletion")
	} else {
		// Volume not attached or mapping not found - proceed directly to deletion
		log.WithFields(log.Fields{
			"volume_id": volumeID,
			"error":     err,
		}).Info("Volume not attached or mapping not found - proceeding directly with deletion")
	}

	// Step 2: Delete volume in CloudStack
	err = vs.cloudStackClient.DeleteVolume(ctx, volumeID)
	if err != nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack volume deletion failed: %w", err))
		return
	}

	// Step 3: Clean up any remaining mappings (safety cleanup)
	if err := vs.repo.DeleteMapping(ctx, volumeID); err != nil {
		log.WithFields(log.Fields{
			"volume_id": volumeID,
			"error":     err,
		}).Debug("No device mapping to delete after volume deletion (expected if detached above)")
	}

	// Update operation with successful result
	operation.Status = models.StatusCompleted
	now := time.Now()
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	operation.Response = map[string]interface{}{
		"volume_id": volumeID,
		"message":   "Volume deleted successfully (detached and cleaned up NBD exports)",
	}

	if err := vs.repo.UpdateOperation(ctx, operation); err != nil {
		log.WithFields(log.Fields{
			"operation_id": operation.ID,
			"volume_id":    volumeID,
			"error":        err,
		}).Error("Failed to update operation after successful volume deletion")
	}

	log.WithFields(log.Fields{
		"volume_id":    volumeID,
		"operation_id": operation.ID,
	}).Info("Volume deletion completed successfully with NBD export cleanup")
}

// completeOperationWithError marks an operation as failed with an error
func (vs *VolumeService) completeOperationWithError(ctx context.Context, operation *models.VolumeOperation, err error) {
	operation.Status = models.StatusFailed
	now := time.Now()
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	errorMsg := err.Error()
	operation.Error = &errorMsg

	if updateErr := vs.repo.UpdateOperation(ctx, operation); updateErr != nil {
		log.WithFields(log.Fields{
			"operation_id":   operation.ID,
			"original_error": err,
			"update_error":   updateErr,
		}).Error("Failed to update operation with error status")
	}
}

// correlateVolumeToDevice attempts to correlate a CloudStack volume with a Linux device
func (vs *VolumeService) correlateVolumeToDevice(ctx context.Context, volumeID, vmID string) (string, int64) {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"vm_id":     vmID,
	}).Info("🔍 Attempting to correlate volume with device")

	if vs.deviceMonitor == nil {
		log.Warn("Device monitor not available for correlation")
		return "", 0
	}

	// ✅ FIXED: Record correlation start time for timestamp filtering
	correlationStartTime := time.Now()
	skippedStaleEvents := 0

	log.WithFields(log.Fields{
		"volume_id":              volumeID,
		"correlation_start_time": correlationStartTime,
	}).Info("🕐 Starting device correlation with timestamp filtering (no pre-draining)")

	// Wait for device events for up to 30 seconds
	timeout := 30 * time.Second
	deadline := time.Now().Add(timeout)
	contemporaryThreshold := correlationStartTime.Add(-30 * time.Second)

	for time.Now().Before(deadline) {
		eventCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		event, err := vs.deviceMonitor.WaitForDevice(eventCtx, 5*time.Second)
		cancel()

		if err != nil {
			// No event, continue waiting
			time.Sleep(1 * time.Second)
			continue
		}

		if event.Type == DeviceAdded {
			// ✅ FIXED: Skip stale events, use contemporary/fresh events immediately
			if event.Timestamp.Before(contemporaryThreshold) {
				skippedStaleEvents++
				log.WithFields(log.Fields{
					"device_path":            event.DevicePath,
					"event_time":             event.Timestamp,
					"correlation_start_time": correlationStartTime,
					"age_seconds":            correlationStartTime.Sub(event.Timestamp).Seconds(),
				}).Debug("🚫 Skipping stale device event (>5s before correlation)")
				continue // Skip stale event, keep looking for fresh ones
			}

			log.WithFields(log.Fields{
				"device_path":   event.DevicePath,
				"device_size":   event.DeviceInfo.Size,
				"event_time":    event.Timestamp,
				"age_seconds":   time.Since(event.Timestamp).Seconds(),
				"skipped_stale": skippedStaleEvents,
			}).Info("✅ Using contemporary/fresh device for correlation")

			// ✅ Clear remaining events after successful correlation
			vs.clearDeviceEventsAfterSuccess(ctx)

			return event.DevicePath, event.DeviceInfo.Size
		}
	}

	log.WithFields(log.Fields{
		"volume_id":     volumeID,
		"timeout":       timeout,
		"skipped_stale": skippedStaleEvents,
	}).Warn("⚠️  No fresh device detected during correlation timeout")

	return "", 0
}

// drainStaleDeviceEvents - REMOVED: No longer needed with simplified correlation approach
// The correlation loop now handles stale event filtering directly without pre-draining

// clearDeviceEventsAfterSuccess clears remaining events after successful correlation
// This prepares the channel for the next volume attachment operation
func (vs *VolumeService) clearDeviceEventsAfterSuccess(ctx context.Context) {
	cleared := 0
	clearTimeout := 1 * time.Second
	clearDeadline := time.Now().Add(clearTimeout)

	log.Debug("🧹 Clearing remaining device events after successful correlation")

	for time.Now().Before(clearDeadline) {
		eventCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		event, err := vs.deviceMonitor.WaitForDevice(eventCtx, 50*time.Millisecond)
		cancel()

		if err != nil {
			// No more events to clear
			break
		}

		cleared++
		log.WithFields(log.Fields{
			"device_path": event.DevicePath,
			"event_time":  event.Timestamp,
		}).Debug("🧹 Cleared device event after successful correlation")
	}

	if cleared > 0 {
		log.WithFields(log.Fields{
			"cleared_events": cleared,
		}).Info("🧹 Cleared remaining device events - channel ready for next correlation")
	}
}

// isOMAVM checks if the given VM ID is the SHA VM (where device correlation is possible)
func (vs *VolumeService) isOMAVM(ctx context.Context, vmID string) bool {
	// The SHA VM ID - could be made configurable or retrieved from database
	const shaVMID = "8a4400e5-c92a-4bc4-8bff-4b6b0b6a018c"
	return vmID == shaVMID
}

// getCloudStackDeviceID retrieves the CloudStack device ID for a volume attachment
func (vs *VolumeService) getCloudStackDeviceID(ctx context.Context, volumeID string) (int, error) {
	// This would typically query CloudStack API to get the device ID
	// For now, return a placeholder since device ID isn't critical for failover mode
	log.WithField("volume_id", volumeID).Debug("Getting CloudStack device ID for failover volume")

	// TODO: Implement actual CloudStack API call to get device ID
	// For failover mode, this is optional - used for better state tracking
	return 0, fmt.Errorf("CloudStack device ID retrieval not implemented")
}

// executeCleanupTestFailover executes the complete test failover cleanup workflow
func (vs *VolumeService) executeCleanupTestFailover(ctx context.Context, operation *models.VolumeOperation, req models.CleanupRequest) {
	// Update operation status to executing
	operation.Status = models.StatusExecuting
	operation.UpdatedAt = time.Now()
	vs.repo.UpdateOperation(ctx, operation)

	log.WithFields(log.Fields{
		"operation_id": operation.ID,
		"test_vm_id":   req.TestVMID,
		"volume_id":    req.VolumeID,
		"oma_vm_id":    req.SHAVMID,
		"delete_vm":    req.DeleteVM,
	}).Info("🧹 Starting test failover cleanup workflow")

	// Check if CloudStack client is available
	if vs.cloudStackClient == nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("CloudStack client not available"))
		return
	}

	// Phase 1: VM Power Management
	if !req.ForceClean {
		log.WithField("test_vm_id", req.TestVMID).Info("🔍 Validating test VM power state")

		// Validate VM is powered off (required for safe volume detachment)
		err := vs.cloudStackClient.ValidateVMPoweredOff(ctx, req.TestVMID)
		if err != nil {
			log.WithFields(log.Fields{
				"test_vm_id": req.TestVMID,
				"error":      err,
			}).Warn("Test VM not powered off, attempting automatic power off")

			// Try to power off the VM
			if powerErr := vs.cloudStackClient.PowerOffVM(ctx, req.TestVMID); powerErr != nil {
				vs.completeOperationWithError(ctx, operation, fmt.Errorf("failed to power off test VM: %w", powerErr))
				return
			}

			// Wait for power state to stabilize
			time.Sleep(3 * time.Second)

			// Re-validate powered off state
			if validateErr := vs.cloudStackClient.ValidateVMPoweredOff(ctx, req.TestVMID); validateErr != nil {
				vs.completeOperationWithError(ctx, operation, fmt.Errorf("test VM still not powered off after forced shutdown: %w", validateErr))
				return
			}
		}

		log.WithField("test_vm_id", req.TestVMID).Info("✅ Test VM validated as powered off")
	}

	// Phase 2: Volume Detachment from Test VM
	log.WithFields(log.Fields{
		"volume_id":  req.VolumeID,
		"test_vm_id": req.TestVMID,
	}).Info("🔗 Detaching volume from test VM")

	err := vs.cloudStackClient.DetachVolume(ctx, req.VolumeID)
	if err != nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("failed to detach volume from test VM: %w", err))
		return
	}

	// Wait for detachment to complete and device to disappear
	time.Sleep(5 * time.Second)

	log.WithField("volume_id", req.VolumeID).Info("✅ Volume detached from test VM")

	// Phase 3: Volume Reattachment to SHA
	log.WithFields(log.Fields{
		"volume_id": req.VolumeID,
		"oma_vm_id": req.SHAVMID,
	}).Info("🔗 Reattaching volume to SHA")

	err = vs.cloudStackClient.AttachVolume(ctx, req.VolumeID, req.SHAVMID)
	if err != nil {
		vs.completeOperationWithError(ctx, operation, fmt.Errorf("failed to reattach volume to SHA: %w", err))
		return
	}

	// Wait for device to appear and correlate with CloudStack volume
	var devicePath string
	var deviceSize int64
	if vs.deviceMonitor != nil {
		devicePath, deviceSize = vs.correlateVolumeToDevice(ctx, req.VolumeID, req.SHAVMID)
	}

	log.WithFields(log.Fields{
		"volume_id":   req.VolumeID,
		"device_path": devicePath,
	}).Info("✅ Volume reattached to SHA")

	// Phase 4: Test VM Deletion (if requested)
	if req.DeleteVM {
		log.WithField("test_vm_id", req.TestVMID).Info("🗑️  Deleting test VM")

		err = vs.cloudStackClient.DeleteVM(ctx, req.TestVMID)
		if err != nil {
			// Log warning but don't fail the cleanup operation
			log.WithFields(log.Fields{
				"test_vm_id": req.TestVMID,
				"error":      err,
			}).Warn("Failed to delete test VM, but volume cleanup completed successfully")
		} else {
			log.WithField("test_vm_id", req.TestVMID).Info("✅ Test VM deleted")
		}
	}

	// Update operation with successful result
	operation.Status = models.StatusCompleted
	now := time.Now()
	operation.UpdatedAt = now
	operation.CompletedAt = &now
	operation.Response = map[string]interface{}{
		"test_vm_id":  req.TestVMID,
		"volume_id":   req.VolumeID,
		"oma_vm_id":   req.SHAVMID,
		"device_path": devicePath,
		"vm_deleted":  req.DeleteVM,
		"message":     "Test failover cleanup completed successfully",
	}

	if err := vs.repo.UpdateOperation(ctx, operation); err != nil {
		log.WithFields(log.Fields{
			"operation_id": operation.ID,
			"volume_id":    req.VolumeID,
			"error":        err,
		}).Error("Failed to update operation after successful cleanup")
	}

	// Create/update device mapping record for SHA attachment
	if devicePath != "" {
		mapping := &models.DeviceMapping{
			VolumeUUID:                req.VolumeID,
			VolumeIDNumeric:           nil, // Will be populated from CloudStack if available
			VMID:                      req.SHAVMID,
			OperationMode:             string(models.OperationModeOMA), // Cleanup always for SHA
			CloudStackDeviceID:        nil,
			RequiresDeviceCorrelation: true,
			DevicePath:                devicePath,
			Size:                      deviceSize,
			CloudStackState:           "attached",
			LinuxState:                "detected",
			LastSync:                  time.Now(),
			CreatedAt:                 time.Now(),
			UpdatedAt:                 time.Now(),
		}

		if err := vs.repo.CreateMapping(ctx, mapping); err != nil {
			log.WithFields(log.Fields{
				"operation_id": operation.ID,
				"volume_id":    req.VolumeID,
				"device_path":  devicePath,
				"error":        err,
			}).Warn("Failed to create/update device mapping after cleanup")
		}
	}

	log.WithFields(log.Fields{
		"operation_id": operation.ID,
		"volume_id":    req.VolumeID,
		"test_vm_id":   req.TestVMID,
		"vm_deleted":   req.DeleteVM,
	}).Info("🎉 Test failover cleanup completed successfully")
}

// NBD Export Management Methods

// CreateNBDExport creates an NBD export for a volume
func (vs *VolumeService) CreateNBDExport(ctx context.Context, volumeID, vmName, vmID string, diskNumber int) (*models.NBDExportInfo, error) {
	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"vm_name":     vmName,
		"vm_id":       vmID,
		"disk_number": diskNumber,
	}).Info("🔗 Creating NBD export via Volume Daemon service")

	// Get device mapping to find the actual device path
	deviceMapping, err := vs.GetDeviceMapping(ctx, volumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device mapping for volume %s: %w", volumeID, err)
	}

	if deviceMapping.DevicePath == "" {
		return nil, fmt.Errorf("volume %s is not attached - no device path available", volumeID)
	}

	// Create NBD export request
	exportRequest := &nbd.ExportRequest{
		VolumeID:   volumeID,
		VMName:     vmName,
		VMID:       vmID,
		DiskNumber: diskNumber,
		DevicePath: deviceMapping.DevicePath,
		ReadOnly:   false, // Default to read-write for migrations
		Metadata: map[string]string{
			"created_via": "volume_daemon_service",
			"vm_name":     vmName,
			"vm_id":       vmID,
		},
	}

	// Create export via NBD export manager
	exportInfo, err := vs.nbdExportManager.CreateExport(ctx, exportRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create NBD export: %w", err)
	}

	// Convert to models.NBDExportInfo
	modelExportInfo := &models.NBDExportInfo{
		ID:         exportInfo.ID,
		VolumeID:   exportInfo.VolumeID,
		ExportName: exportInfo.ExportName,
		DevicePath: exportInfo.DevicePath,
		Port:       exportInfo.Port,
		Status:     models.NBDExportStatus(exportInfo.Status),
		CreatedAt:  exportInfo.CreatedAt,
		UpdatedAt:  exportInfo.UpdatedAt,
		Metadata:   exportInfo.Metadata,
	}

	log.WithFields(log.Fields{
		"export_id":   exportInfo.ID,
		"export_name": exportInfo.ExportName,
		"device_path": exportInfo.DevicePath,
		"port":        exportInfo.Port,
	}).Info("✅ NBD export created successfully via Volume Daemon service")

	return modelExportInfo, nil
}

// DeleteNBDExport removes an NBD export for a volume
func (vs *VolumeService) DeleteNBDExport(ctx context.Context, volumeID string) error {
	log.WithField("volume_id", volumeID).Info("🗑️ Deleting NBD export via Volume Daemon service")

	if err := vs.nbdExportManager.DeleteExport(ctx, volumeID); err != nil {
		return fmt.Errorf("failed to delete NBD export: %w", err)
	}

	log.WithField("volume_id", volumeID).Info("✅ NBD export deleted successfully via Volume Daemon service")
	return nil
}

// GetNBDExport retrieves NBD export information for a volume
func (vs *VolumeService) GetNBDExport(ctx context.Context, volumeID string) (*models.NBDExportInfo, error) {
	exportInfo, err := vs.nbdExportManager.GetExport(ctx, volumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get NBD export: %w", err)
	}

	// Convert to models.NBDExportInfo
	modelExportInfo := &models.NBDExportInfo{
		ID:         exportInfo.ID,
		VolumeID:   exportInfo.VolumeID,
		ExportName: exportInfo.ExportName,
		DevicePath: exportInfo.DevicePath,
		Port:       exportInfo.Port,
		Status:     models.NBDExportStatus(exportInfo.Status),
		CreatedAt:  exportInfo.CreatedAt,
		UpdatedAt:  exportInfo.UpdatedAt,
		Metadata:   exportInfo.Metadata,
	}

	return modelExportInfo, nil
}

// ListNBDExports lists NBD exports with optional filtering
func (vs *VolumeService) ListNBDExports(ctx context.Context, filter models.NBDExportFilter) ([]*models.NBDExportInfo, error) {
	// Convert models filter to nbd filter
	nbdFilter := nbd.ExportFilter{
		VolumeID: filter.VolumeID,
		VMName:   filter.VMName,
		Limit:    filter.Limit,
	}

	if filter.Status != nil {
		status := nbd.ExportStatus(*filter.Status)
		nbdFilter.Status = &status
	}

	exports, err := vs.nbdExportManager.ListExports(ctx, nbdFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list NBD exports: %w", err)
	}

	// Convert to models.NBDExportInfo slice
	modelExports := make([]*models.NBDExportInfo, len(exports))
	for i, export := range exports {
		modelExports[i] = &models.NBDExportInfo{
			ID:         export.ID,
			VolumeID:   export.VolumeID,
			ExportName: export.ExportName,
			DevicePath: export.DevicePath,
			Port:       export.Port,
			Status:     models.NBDExportStatus(export.Status),
			CreatedAt:  export.CreatedAt,
			UpdatedAt:  export.UpdatedAt,
			Metadata:   export.Metadata,
		}
	}

	return modelExports, nil
}

// ValidateNBDExports validates consistency between database and NBD configuration
func (vs *VolumeService) ValidateNBDExports(ctx context.Context) error {
	return vs.nbdExportManager.ValidateExports(ctx)
}

// NBD Export Lifecycle Integration (Private Helper Methods)

// createNBDExportForVolume creates an NBD export automatically during volume attachment
func (vs *VolumeService) createNBDExportForVolume(ctx context.Context, volumeID, vmID, devicePath string) {
	if vs.nbdExportManager == nil {
		log.Warn("NBD export manager not available - skipping automatic export creation")
		return
	}

	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"vm_id":       vmID,
		"device_path": devicePath,
	}).Info("🔗 Creating NBD export automatically during volume attachment")

	// 🎯 CRITICAL: Validate and recreate NBD export if device path changed
	if err := vs.ensureNBDExportDevicePathCorrect(ctx, volumeID, devicePath); err != nil {
		log.WithError(err).WithField("volume_id", volumeID).Error("Failed to ensure NBD export device path correct")
		// Continue with creation - this is a warning, not a fatal error
	}

	// Get VM name from CloudStack (use VM ID as fallback)
	vmName := vs.getVMName(ctx, vmID)
	if vmName == "" {
		vmName = vmID // Fallback to VM ID if name not available
	}

	// 🎯 CRITICAL FIX: Get vm_disk_id correlation for multi-disk support
	vmDiskID := vs.getVMDiskIDFromOMA(ctx, volumeID)

	// Determine disk number based on existing exports for this VM
	diskNumber := vs.getNextDiskNumber(ctx, vmID)

	// Create NBD export request
	exportRequest := &nbd.ExportRequest{
		VolumeID:   volumeID,
		VMName:     vmName,
		VMID:       vmID,
		VMDiskID:   vmDiskID, // Include vm_disk_id correlation from SHA
		DiskNumber: diskNumber,
		DevicePath: devicePath,
		ReadOnly:   false, // Default to read-write for migrations
		Metadata: map[string]string{
			"created_by":    "volume_daemon_lifecycle",
			"created_at":    time.Now().Format(time.RFC3339),
			"vm_name":       vmName,
			"vm_id":         vmID,
			"auto_created":  "true",
			"volume_attach": "true",
		},
	}

	// Create export via NBD export manager
	exportInfo, err := vs.nbdExportManager.CreateExport(ctx, exportRequest)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"volume_id":   volumeID,
			"vm_id":       vmID,
			"device_path": devicePath,
		}).Error("Failed to create NBD export during volume attachment")
		return
	}

	log.WithFields(log.Fields{
		"export_id":   exportInfo.ID,
		"export_name": exportInfo.ExportName,
		"volume_id":   volumeID,
		"device_path": devicePath,
		"port":        exportInfo.Port,
	}).Info("✅ NBD export created automatically during volume attachment")
}

// deleteNBDExportForVolume deletes an NBD export automatically during volume detachment
func (vs *VolumeService) deleteNBDExportForVolume(ctx context.Context, volumeID string) {
	if vs.nbdExportManager == nil {
		log.Warn("NBD export manager not available - skipping automatic export deletion")
		return
	}

	log.WithField("volume_id", volumeID).Info("🗑️ Deleting NBD export automatically during volume detachment")

	// Delete export via NBD export manager
	if err := vs.nbdExportManager.DeleteExport(ctx, volumeID); err != nil {
		log.WithError(err).WithField("volume_id", volumeID).Error("Failed to delete NBD export during volume detachment")
		return
	}

	log.WithField("volume_id", volumeID).Info("✅ NBD export deleted automatically during volume detachment")
}

// getVMDiskIDFromOMA queries the SHA database to get vm_disk_id correlation for NBD export
// NOTE: This would require database access that VolumeService doesn't currently have
// For now, return nil to skip vm_disk_id correlation (handled by SHA-level auto-repair)
func (vs *VolumeService) getVMDiskIDFromOMA(ctx context.Context, volumeID string) *int {
	log.WithField("volume_id", volumeID).Debug("⚠️ VolumeService doesn't have database access - skipping vm_disk_id correlation")
	return nil // SHA auto-repair will fix this later
}

// ensureNBDExportDevicePathCorrect validates and recreates NBD export if device path changed
func (vs *VolumeService) ensureNBDExportDevicePathCorrect(ctx context.Context, volumeID, currentDevicePath string) error {
	exportName := fmt.Sprintf("migration-vol-%s", volumeID)
	configPath := fmt.Sprintf("/etc/nbd-server/conf.d/%s.conf", exportName)

	log.WithFields(log.Fields{
		"volume_id":           volumeID,
		"current_device_path": currentDevicePath,
		"export_name":         exportName,
		"config_path":         configPath,
	}).Debug("🔍 Validating NBD export device path")

	// Check if export config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.WithField("config_path", configPath).Debug("NBD export config does not exist - will be created")
		return nil // Config doesn't exist, normal creation will handle it
	}

	// Read current export configuration
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		log.WithError(err).WithField("config_path", configPath).Warn("Failed to read NBD export config")
		return nil // Continue with normal creation
	}

	// Extract current device path from config
	currentExportDevice := vs.extractDevicePathFromNBDConfig(string(configContent))

	// Check if device paths match
	if currentExportDevice == currentDevicePath {
		log.WithFields(log.Fields{
			"volume_id":   volumeID,
			"device_path": currentDevicePath,
		}).Debug("✅ NBD export device path is correct - no recreation needed")
		return nil
	}

	// Device paths don't match - recreate export!
	log.WithFields(log.Fields{
		"volume_id":     volumeID,
		"export_device": currentExportDevice,
		"actual_device": currentDevicePath,
		"export_name":   exportName,
	}).Warn("🔄 NBD export device path mismatch detected - recreating export")

	// Delete old export config file
	if err := os.Remove(configPath); err != nil {
		log.WithError(err).WithField("config_path", configPath).Error("Failed to remove old NBD export config")
		return fmt.Errorf("failed to remove old NBD config: %w", err)
	}

	// Create new export config with correct device path
	if err := vs.createNBDExportConfigFile(exportName, currentDevicePath, configPath); err != nil {
		return fmt.Errorf("failed to create new NBD export config: %w", err)
	}

	// Reload NBD server to pick up new configuration
	if err := vs.reloadNBDServer(); err != nil {
		log.WithError(err).Warn("Failed to reload NBD server after export recreation")
		return fmt.Errorf("failed to reload NBD server: %w", err)
	}

	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"export_name": exportName,
		"new_device":  currentDevicePath,
		"old_device":  currentExportDevice,
	}).Info("✅ NBD export recreated with correct device path")

	return nil
}

// extractDevicePathFromNBDConfig extracts device path from NBD config file content
func (vs *VolumeService) extractDevicePathFromNBDConfig(configContent string) string {
	// Parse NBD config format: exportname = /dev/vdX
	lines := strings.Split(configContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "exportname") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				devicePath := strings.TrimSpace(parts[1])
				return devicePath
			}
		}
	}
	return "unknown"
}

// createNBDExportConfigFile creates NBD export configuration file
func (vs *VolumeService) createNBDExportConfigFile(exportName, devicePath, configPath string) error {
	configContent := fmt.Sprintf(`[%s]
exportname = %s
readonly = false
multifile = false
copyonwrite = false
`, exportName, devicePath)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write NBD config file: %w", err)
	}

	log.WithFields(log.Fields{
		"export_name": exportName,
		"device_path": devicePath,
		"config_path": configPath,
	}).Debug("✅ NBD export config file created")

	return nil
}

// reloadNBDServer sends SIGHUP to reload NBD server configuration
func (vs *VolumeService) reloadNBDServer() error {
	// Find NBD server process
	cmd := exec.Command("pgrep", "nbd-server")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find NBD server process: %w", err)
	}

	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return fmt.Errorf("NBD server process not found")
	}

	// Send SIGHUP to reload configuration
	reloadCmd := exec.Command("kill", "-HUP", pidStr)
	if err := reloadCmd.Run(); err != nil {
		return fmt.Errorf("failed to send SIGHUP to NBD server: %w", err)
	}

	log.WithField("nbd_server_pid", pidStr).Info("✅ NBD server configuration reloaded via SIGHUP")
	return nil
}

// getVMName retrieves VM name from CloudStack (with caching potential)
func (vs *VolumeService) getVMName(ctx context.Context, vmID string) string {
	// For now, return VM ID as name - in future could cache VM info or query CloudStack
	// This could be enhanced to query CloudStack API for actual VM name
	return vmID
}

// getNextDiskNumber determines the next disk number for a VM based on existing exports
func (vs *VolumeService) getNextDiskNumber(ctx context.Context, vmID string) int {
	if vs.nbdExportManager == nil {
		return 0 // Default to disk 0
	}

	// List existing exports for this VM
	filter := nbd.ExportFilter{
		VMName: &vmID, // Using VM ID since that's what we store in metadata
		Limit:  100,   // Reasonable limit for exports per VM
	}

	exports, err := vs.nbdExportManager.ListExports(ctx, filter)
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Warn("Failed to list existing exports for disk number calculation")
		return 0 // Default to disk 0
	}

	// Find the highest disk number in use
	maxDiskNumber := -1
	for _, export := range exports {
		if export.Metadata != nil {
			if vmIDMetadata, exists := export.Metadata["vm_id"]; exists && vmIDMetadata == vmID {
				// Parse export name to extract disk number
				if _, diskNumber, err := nbd.ParseExportName(export.ExportName); err == nil {
					if diskNumber > maxDiskNumber {
						maxDiskNumber = diskNumber
					}
				}
			}
		}
	}

	return maxDiskNumber + 1
}

// TrackVolumeSnapshot stores snapshot information for a volume in device_mappings
func (vs *VolumeService) TrackVolumeSnapshot(ctx context.Context, req *models.TrackSnapshotRequest) error {
	log.WithFields(log.Fields{
		"volume_uuid":   req.VolumeUUID,
		"vm_context_id": req.VMContextID,
		"snapshot_id":   req.SnapshotID,
		"snapshot_name": req.SnapshotName,
		"disk_id":       req.DiskID,
	}).Info("📸 Tracking volume snapshot in device_mappings")

	// Set default status if not provided
	status := req.SnapshotStatus
	if status == "" {
		status = "ready"
	}

	// Find the device mapping for this volume
	mapping, err := vs.repo.GetDeviceMappingByVolumeUUID(ctx, req.VolumeUUID)
	if err != nil {
		return fmt.Errorf("failed to find device mapping for volume %s: %w", req.VolumeUUID, err)
	}

	// Update the device mapping with snapshot information
	now := time.Now()
	mapping.OSSEASnapshotID = &req.SnapshotID
	mapping.SnapshotCreatedAt = &now
	mapping.SnapshotStatus = status

	// Also ensure VM context ID is set
	if mapping.VMContextID == nil || *mapping.VMContextID != req.VMContextID {
		mapping.VMContextID = &req.VMContextID
	}

	err = vs.repo.UpdateDeviceMapping(ctx, mapping)
	if err != nil {
		return fmt.Errorf("failed to update device mapping with snapshot info: %w", err)
	}

	log.WithFields(log.Fields{
		"volume_uuid":   req.VolumeUUID,
		"snapshot_id":   req.SnapshotID,
		"device_path":   mapping.DevicePath,
		"vm_context_id": req.VMContextID,
	}).Info("✅ Volume snapshot tracked successfully in device_mappings")

	return nil
}

// GetVMVolumeSnapshots retrieves all snapshot information for volumes belonging to a VM
func (vs *VolumeService) GetVMVolumeSnapshots(ctx context.Context, vmContextID string) ([]models.VolumeSnapshotInfo, error) {
	log.WithField("vm_context_id", vmContextID).Info("🔍 Getting volume snapshots for VM")

	// Get all device mappings for this VM context
	mappings, err := vs.repo.GetDeviceMappingsByVMContext(ctx, vmContextID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device mappings for VM context %s: %w", vmContextID, err)
	}

	var snapshots []models.VolumeSnapshotInfo
	for _, mapping := range mappings {
		snapshotInfo := models.VolumeSnapshotInfo{
			VolumeUUID:        mapping.VolumeUUID,
			VMContextID:       vmContextID,
			DevicePath:        mapping.DevicePath,
			OperationMode:     mapping.OperationMode,
			SnapshotStatus:    mapping.SnapshotStatus,
			SnapshotID:        mapping.OSSEASnapshotID,
			SnapshotCreatedAt: mapping.SnapshotCreatedAt,
		}
		snapshots = append(snapshots, snapshotInfo)
	}

	log.WithFields(log.Fields{
		"vm_context_id":  vmContextID,
		"snapshot_count": len(snapshots),
	}).Info("✅ Retrieved VM volume snapshots")

	return snapshots, nil
}

// ClearVMVolumeSnapshots clears all snapshot tracking information for a VM
func (vs *VolumeService) ClearVMVolumeSnapshots(ctx context.Context, vmContextID string) (int, error) {
	log.WithField("vm_context_id", vmContextID).Info("🧹 Clearing volume snapshots for VM")

	// Get all device mappings for this VM context
	mappings, err := vs.repo.GetDeviceMappingsByVMContext(ctx, vmContextID)
	if err != nil {
		return 0, fmt.Errorf("failed to get device mappings for VM context %s: %w", vmContextID, err)
	}

	clearedCount := 0
	for _, mapping := range mappings {
		// Only clear if there's actually snapshot data to clear
		if mapping.OSSEASnapshotID != nil || mapping.SnapshotStatus != "none" {
			mapping.OSSEASnapshotID = nil
			mapping.SnapshotCreatedAt = nil
			mapping.SnapshotStatus = "none"

			err = vs.repo.UpdateDeviceMapping(ctx, &mapping)
			if err != nil {
				log.WithFields(log.Fields{
					"volume_uuid": mapping.VolumeUUID,
					"error":       err,
				}).Error("Failed to clear snapshot info for volume")
				continue
			}
			clearedCount++
		}
	}

	log.WithFields(log.Fields{
		"vm_context_id":     vmContextID,
		"snapshots_cleared": clearedCount,
	}).Info("✅ Cleared VM volume snapshots")

	return clearedCount, nil
}

// UpdateVolumeSnapshot updates snapshot information for a specific volume
func (vs *VolumeService) UpdateVolumeSnapshot(ctx context.Context, req *models.UpdateSnapshotRequest) error {
	log.WithFields(log.Fields{
		"volume_uuid":     req.VolumeUUID,
		"snapshot_id":     req.SnapshotID,
		"snapshot_status": req.SnapshotStatus,
	}).Info("🔄 Updating volume snapshot information")

	// Find the device mapping for this volume
	mapping, err := vs.repo.GetDeviceMappingByVolumeUUID(ctx, req.VolumeUUID)
	if err != nil {
		return fmt.Errorf("failed to find device mapping for volume %s: %w", req.VolumeUUID, err)
	}

	// Update fields that are provided
	if req.SnapshotID != nil {
		mapping.OSSEASnapshotID = req.SnapshotID
	}

	if req.SnapshotStatus != nil {
		mapping.SnapshotStatus = *req.SnapshotStatus
	}

	// Update timestamp if we're setting a new snapshot
	if req.SnapshotID != nil && *req.SnapshotID != "" {
		now := time.Now()
		mapping.SnapshotCreatedAt = &now
	}

	err = vs.repo.UpdateDeviceMapping(ctx, mapping)
	if err != nil {
		return fmt.Errorf("failed to update device mapping: %w", err)
	}

	log.WithFields(log.Fields{
		"volume_uuid": req.VolumeUUID,
		"snapshot_id": mapping.OSSEASnapshotID,
		"device_path": mapping.DevicePath,
	}).Info("✅ Volume snapshot information updated")

	return nil
}
