package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/models"
)

// StateRecoveryService provides mechanisms to recover lost Volume Daemon device mappings
type StateRecoveryService struct {
	repo                VolumeRepository
	cloudStackClient    CloudStackClient
	deviceMonitor       DeviceMonitor
	maxRecoveryAttempts int
	recoveryTimeout     time.Duration
}

// NewStateRecoveryService creates a new state recovery service
func NewStateRecoveryService(
	repo VolumeRepository,
	cloudStackClient CloudStackClient,
	deviceMonitor DeviceMonitor,
) *StateRecoveryService {
	return &StateRecoveryService{
		repo:                repo,
		cloudStackClient:    cloudStackClient,
		deviceMonitor:       deviceMonitor,
		maxRecoveryAttempts: 3,
		recoveryTimeout:     2 * time.Minute,
	}
}

// RecoveryResult represents the result of a state recovery operation
type RecoveryResult struct {
	VolumesRecovered  int                    `json:"volumes_recovered"`
	VolumesOrphaned   int                    `json:"volumes_orphaned"`
	MappingsCreated   int                    `json:"mappings_created"`
	MappingsFixed     int                    `json:"mappings_fixed"`
	RecoveredMappings []models.DeviceMapping `json:"recovered_mappings"`
	OrphanedVolumes   []OrphanedVolume       `json:"orphaned_volumes"`
	Errors            []string               `json:"errors"`
	Duration          time.Duration          `json:"duration"`
}

// OrphanedVolume represents a volume that couldn't be recovered
type OrphanedVolume struct {
	VolumeID     string `json:"volume_id"`
	VolumeName   string `json:"volume_name"`
	AttachedToVM string `json:"attached_to_vm"`
	State        string `json:"state"`
	Reason       string `json:"reason"`
}

// RecoverLostMappings attempts to recover all lost device mappings for a specific VM
func (srs *StateRecoveryService) RecoverLostMappings(ctx context.Context, vmID string) (*RecoveryResult, error) {
	startTime := time.Now()
	result := &RecoveryResult{
		RecoveredMappings: make([]models.DeviceMapping, 0),
		OrphanedVolumes:   make([]OrphanedVolume, 0),
		Errors:            make([]string, 0),
	}

	log.WithField("vm_id", vmID).Info("🔄 Starting state recovery for VM")

	// Step 1: Get all volumes attached to this VM from CloudStack
	cloudStackVolumes, err := srs.getCloudStackVolumesForVM(ctx, vmID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get CloudStack volumes: %v", err))
		return result, fmt.Errorf("failed to get CloudStack volumes for VM %s: %w", vmID, err)
	}

	log.WithFields(log.Fields{
		"vm_id":        vmID,
		"volume_count": len(cloudStackVolumes),
	}).Info("📋 Found CloudStack volumes for VM")

	// Step 2: Get current device mappings from Volume Daemon database
	existingMappings, err := srs.repo.ListMappingsForVM(ctx, vmID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get existing mappings: %v", err))
		return result, fmt.Errorf("failed to get existing mappings for VM %s: %w", vmID, err)
	}

	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"mapping_count": len(existingMappings),
	}).Info("📊 Found existing device mappings")

	// Step 3: Create mapping index for quick lookup
	mappingIndex := make(map[string]*models.DeviceMapping)
	for i := range existingMappings {
		mappingIndex[existingMappings[i].VolumeUUID] = &existingMappings[i]
	}

	// Step 4: Process each CloudStack volume
	for _, volume := range cloudStackVolumes {
		log.WithFields(log.Fields{
			"volume_id":   volume.VolumeID,
			"volume_name": volume.VolumeName,
			"device_id":   volume.DeviceID,
			"vm_id":       vmID,
		}).Debug("Processing CloudStack volume")

		if existingMapping, exists := mappingIndex[volume.VolumeID]; exists {
			// Mapping exists - verify and fix if needed
			if err := srs.verifyAndFixMapping(ctx, existingMapping, &volume); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to verify mapping for volume %s: %v", volume.VolumeID, err))
				continue
			}
			result.MappingsFixed++
		} else {
			// Missing mapping - attempt recovery
			recoveredMapping, err := srs.recoverMissingMapping(ctx, &volume, vmID)
			if err != nil {
				log.WithError(err).WithField("volume_id", volume.VolumeID).Warn("Failed to recover mapping")
				result.OrphanedVolumes = append(result.OrphanedVolumes, OrphanedVolume{
					VolumeID:     volume.VolumeID,
					VolumeName:   volume.VolumeName,
					AttachedToVM: vmID,
					State:        volume.State,
					Reason:       err.Error(),
				})
				result.VolumesOrphaned++
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to recover volume %s: %v", volume.VolumeID, err))
				continue
			}

			result.RecoveredMappings = append(result.RecoveredMappings, *recoveredMapping)
			result.MappingsCreated++
			result.VolumesRecovered++

			log.WithFields(log.Fields{
				"volume_id":   volume.VolumeID,
				"device_path": recoveredMapping.DevicePath,
			}).Info("✅ Successfully recovered device mapping")
		}
	}

	result.Duration = time.Since(startTime)

	log.WithFields(log.Fields{
		"vm_id":             vmID,
		"volumes_recovered": result.VolumesRecovered,
		"volumes_orphaned":  result.VolumesOrphaned,
		"mappings_created":  result.MappingsCreated,
		"mappings_fixed":    result.MappingsFixed,
		"duration":          result.Duration,
		"error_count":       len(result.Errors),
	}).Info("🎯 State recovery completed")

	return result, nil
}

// RecoverSingleVolume attempts to recover a single lost volume mapping
func (srs *StateRecoveryService) RecoverSingleVolume(ctx context.Context, volumeID string) (*models.DeviceMapping, error) {
	log.WithField("volume_id", volumeID).Info("🔄 Starting single volume recovery")

	// Get volume information from CloudStack
	cloudStackVolume, err := srs.getCloudStackVolume(ctx, volumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get CloudStack volume: %w", err)
	}

	if cloudStackVolume.AttachedToVM == "" {
		return nil, fmt.Errorf("volume %s is not attached to any VM", volumeID)
	}

	// Attempt recovery
	recoveredMapping, err := srs.recoverMissingMapping(ctx, cloudStackVolume, cloudStackVolume.AttachedToVM)
	if err != nil {
		return nil, fmt.Errorf("failed to recover mapping: %w", err)
	}

	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"device_path": recoveredMapping.DevicePath,
		"vm_id":       cloudStackVolume.AttachedToVM,
	}).Info("✅ Successfully recovered single volume mapping")

	return recoveredMapping, nil
}

// PerformFullSystemRecovery performs comprehensive system-wide state recovery
func (srs *StateRecoveryService) PerformFullSystemRecovery(ctx context.Context) (*RecoveryResult, error) {
	startTime := time.Now()
	totalResult := &RecoveryResult{
		RecoveredMappings: make([]models.DeviceMapping, 0),
		OrphanedVolumes:   make([]OrphanedVolume, 0),
		Errors:            make([]string, 0),
	}

	log.Info("🔄 Starting full system state recovery")

	// Get all attached volumes from CloudStack
	allVolumes, err := srs.getAllAttachedVolumes(ctx)
	if err != nil {
		return totalResult, fmt.Errorf("failed to get all attached volumes: %w", err)
	}

	log.WithField("total_volumes", len(allVolumes)).Info("📋 Found attached volumes system-wide")

	// Group volumes by VM
	volumesByVM := make(map[string][]CloudStackVolume)
	for _, volume := range allVolumes {
		if volume.AttachedToVM != "" {
			volumesByVM[volume.AttachedToVM] = append(volumesByVM[volume.AttachedToVM], volume)
		}
	}

	// Process each VM
	for vmID, volumes := range volumesByVM {
		log.WithFields(log.Fields{
			"vm_id":        vmID,
			"volume_count": len(volumes),
		}).Info("🔄 Processing VM for recovery")

		vmResult, err := srs.RecoverLostMappings(ctx, vmID)
		if err != nil {
			totalResult.Errors = append(totalResult.Errors, fmt.Sprintf("VM %s recovery failed: %v", vmID, err))
			continue
		}

		// Aggregate results
		totalResult.VolumesRecovered += vmResult.VolumesRecovered
		totalResult.VolumesOrphaned += vmResult.VolumesOrphaned
		totalResult.MappingsCreated += vmResult.MappingsCreated
		totalResult.MappingsFixed += vmResult.MappingsFixed
		totalResult.RecoveredMappings = append(totalResult.RecoveredMappings, vmResult.RecoveredMappings...)
		totalResult.OrphanedVolumes = append(totalResult.OrphanedVolumes, vmResult.OrphanedVolumes...)
		totalResult.Errors = append(totalResult.Errors, vmResult.Errors...)
	}

	totalResult.Duration = time.Since(startTime)

	log.WithFields(log.Fields{
		"total_vms":         len(volumesByVM),
		"volumes_recovered": totalResult.VolumesRecovered,
		"volumes_orphaned":  totalResult.VolumesOrphaned,
		"mappings_created":  totalResult.MappingsCreated,
		"mappings_fixed":    totalResult.MappingsFixed,
		"duration":          totalResult.Duration,
		"error_count":       len(totalResult.Errors),
	}).Info("🎯 Full system recovery completed")

	return totalResult, nil
}

// Private helper methods

// recoverMissingMapping attempts to recover a missing device mapping
func (srs *StateRecoveryService) recoverMissingMapping(ctx context.Context, volume *CloudStackVolume, vmID string) (*models.DeviceMapping, error) {
	log.WithFields(log.Fields{
		"volume_id": volume.VolumeID,
		"vm_id":     vmID,
		"device_id": volume.DeviceID,
	}).Debug("Attempting to recover missing device mapping")

	// Step 1: Try to correlate with current devices using size and timing
	devicePath, err := srs.correlateDeviceBySize(ctx, volume.SizeBytes)
	if err != nil {
		log.WithError(err).Debug("Size-based correlation failed, trying device enumeration")

		// Step 2: Fallback to device enumeration based on CloudStack device ID
		devicePath, err = srs.correlateDeviceByCloudStackID(ctx, volume.DeviceID, vmID)
		if err != nil {
			return nil, fmt.Errorf("failed to correlate device: %w", err)
		}
	}

	// Step 3: Create the recovered mapping
	deviceIDPtr := volume.DeviceID
	mapping := &models.DeviceMapping{
		VolumeUUID:                volume.VolumeID,
		VolumeIDNumeric:           &volume.VolumeIDNumeric,
		VMID:                      vmID,
		DevicePath:                devicePath,
		CloudStackState:           "attached",
		LinuxState:                "detected",
		OperationMode:             "oma",
		CloudStackDeviceID:        &deviceIDPtr,
		RequiresDeviceCorrelation: false, // Already correlated
		Size:                      volume.SizeBytes,
		LastSync:                  time.Now(),
		CreatedAt:                 time.Now(),
		UpdatedAt:                 time.Now(),
	}

	// Step 4: Store the recovered mapping
	if err := srs.repo.CreateMapping(ctx, mapping); err != nil {
		return nil, fmt.Errorf("failed to create recovered mapping: %w", err)
	}

	return mapping, nil
}

// correlateDeviceBySize attempts to find a device by matching size
func (srs *StateRecoveryService) correlateDeviceBySize(ctx context.Context, expectedSize int64) (string, error) {
	devices, err := srs.deviceMonitor.GetDevices(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get devices: %w", err)
	}

	// Look for devices with matching size (with 3GB tolerance for CloudStack overhead)
	tolerance := int64(3 * 1024 * 1024 * 1024) // 3GB

	for _, deviceInfo := range devices {
		if deviceInfo.Size >= expectedSize-tolerance && deviceInfo.Size <= expectedSize+tolerance {
			// Additional validation: device should be virtio and not already mapped
			if strings.Contains(deviceInfo.Path, "/dev/vd") {
				// Check if this device is already mapped
				existingMapping, err := srs.repo.GetMappingByDevice(ctx, deviceInfo.Path)
				if err == nil && existingMapping != nil {
					continue // Device already mapped
				}

				log.WithFields(log.Fields{
					"device_path":   deviceInfo.Path,
					"device_size":   deviceInfo.Size,
					"expected_size": expectedSize,
					"size_diff":     deviceInfo.Size - expectedSize,
				}).Debug("Found potential device match by size")

				return deviceInfo.Path, nil
			}
		}
	}

	return "", fmt.Errorf("no device found matching size %d bytes", expectedSize)
}

// correlateDeviceByCloudStackID attempts to find device using CloudStack device ID mapping
func (srs *StateRecoveryService) correlateDeviceByCloudStackID(ctx context.Context, deviceID int, vmID string) (string, error) {
	// CloudStack device IDs typically map to Linux devices as follows:
	// Device ID 0 = /dev/vda (root)
	// Device ID 1 = /dev/vdb
	// Device ID 2 = /dev/vdc
	// etc.

	var devicePath string
	if deviceID == 0 {
		devicePath = "/dev/vda"
	} else {
		// Convert device ID to letter (1=b, 2=c, 3=d, etc.)
		deviceLetter := rune('a' + deviceID)
		devicePath = fmt.Sprintf("/dev/vd%c", deviceLetter)
	}

	log.WithFields(log.Fields{
		"cloudstack_device_id":  deviceID,
		"predicted_device_path": devicePath,
		"vm_id":                 vmID,
	}).Debug("Attempting device correlation by CloudStack ID")

	// Verify the device actually exists
	devices, err := srs.deviceMonitor.GetDevices(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get devices: %w", err)
	}

	var deviceInfo *DeviceInfo
	var exists bool
	for _, device := range devices {
		if device.Path == devicePath {
			deviceInfo = &device
			exists = true
			break
		}
	}

	if exists {
		// Check if this device is already mapped
		existingMapping, err := srs.repo.GetMappingByDevice(ctx, devicePath)
		if err == nil && existingMapping != nil {
			return "", fmt.Errorf("device %s is already mapped to volume %s", devicePath, existingMapping.VolumeUUID)
		}

		log.WithFields(log.Fields{
			"device_path": devicePath,
			"device_size": deviceInfo.Size,
		}).Debug("Successfully correlated device by CloudStack ID")

		return devicePath, nil
	}

	return "", fmt.Errorf("predicted device %s does not exist", devicePath)
}

// verifyAndFixMapping verifies an existing mapping and fixes inconsistencies
func (srs *StateRecoveryService) verifyAndFixMapping(ctx context.Context, mapping *models.DeviceMapping, volume *CloudStackVolume) error {
	log.WithFields(log.Fields{
		"volume_id":   mapping.VolumeUUID,
		"device_path": mapping.DevicePath,
	}).Debug("Verifying existing device mapping")

	// Check if the device still exists
	devices, err := srs.deviceMonitor.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	var deviceInfo *DeviceInfo
	var deviceExists bool
	for _, device := range devices {
		if device.Path == mapping.DevicePath {
			deviceInfo = &device
			deviceExists = true
			break
		}
	}

	if !deviceExists {
		// Device no longer exists - need to re-correlate
		log.WithFields(log.Fields{
			"volume_id":   mapping.VolumeUUID,
			"device_path": mapping.DevicePath,
		}).Warn("Device no longer exists, attempting re-correlation")

		// Try to find the new device path
		newDevicePath, err := srs.correlateDeviceBySize(ctx, volume.SizeBytes)
		if err != nil {
			newDevicePath, err = srs.correlateDeviceByCloudStackID(ctx, volume.DeviceID, mapping.VMID)
			if err != nil {
				return fmt.Errorf("failed to re-correlate device: %w", err)
			}
		}

		// Update the mapping with new device path
		mapping.DevicePath = newDevicePath
		mapping.UpdatedAt = time.Now()
		mapping.LastSync = time.Now()

		if err := srs.repo.UpdateMapping(ctx, mapping); err != nil {
			return fmt.Errorf("failed to update mapping with new device path: %w", err)
		}

		log.WithFields(log.Fields{
			"volume_id":       mapping.VolumeUUID,
			"old_device_path": mapping.DevicePath,
			"new_device_path": newDevicePath,
		}).Info("✅ Updated mapping with new device path")
	} else {
		// Device exists - verify size consistency
		tolerance := int64(3 * 1024 * 1024 * 1024) // 3GB tolerance
		if deviceInfo.Size < volume.SizeBytes-tolerance || deviceInfo.Size > volume.SizeBytes+tolerance {
			log.WithFields(log.Fields{
				"volume_id":     mapping.VolumeUUID,
				"device_path":   mapping.DevicePath,
				"device_size":   deviceInfo.Size,
				"expected_size": volume.SizeBytes,
				"size_diff":     deviceInfo.Size - volume.SizeBytes,
			}).Warn("Device size mismatch detected")
		}

		// Update last sync time
		mapping.LastSync = time.Now()
		mapping.UpdatedAt = time.Now()
		if err := srs.repo.UpdateMapping(ctx, mapping); err != nil {
			return fmt.Errorf("failed to update mapping sync time: %w", err)
		}
	}

	return nil
}

// CloudStack data structures for recovery

type CloudStackVolume struct {
	VolumeID        string `json:"volume_id"`
	VolumeName      string `json:"volume_name"`
	VolumeIDNumeric int64  `json:"volume_id_numeric"`
	AttachedToVM    string `json:"attached_to_vm"`
	DeviceID        int    `json:"device_id"`
	SizeBytes       int64  `json:"size_bytes"`
	State           string `json:"state"`
}

// Helper methods to interface with CloudStack

func (srs *StateRecoveryService) getCloudStackVolumesForVM(ctx context.Context, vmID string) ([]CloudStackVolume, error) {
	// This would call the CloudStack client to get volumes for a specific VM
	// Implementation depends on the CloudStack client interface
	log.WithField("vm_id", vmID).Debug("Getting CloudStack volumes for VM")

	// Placeholder - would be implemented with actual CloudStack API calls
	return []CloudStackVolume{}, nil
}

func (srs *StateRecoveryService) getCloudStackVolume(ctx context.Context, volumeID string) (*CloudStackVolume, error) {
	// This would call the CloudStack client to get a specific volume
	log.WithField("volume_id", volumeID).Debug("Getting CloudStack volume")

	// Placeholder - would be implemented with actual CloudStack API calls
	return &CloudStackVolume{}, nil
}

func (srs *StateRecoveryService) getAllAttachedVolumes(ctx context.Context) ([]CloudStackVolume, error) {
	// This would call the CloudStack client to get all attached volumes
	log.Debug("Getting all attached CloudStack volumes")

	// Placeholder - would be implemented with actual CloudStack API calls
	return []CloudStackVolume{}, nil
}
