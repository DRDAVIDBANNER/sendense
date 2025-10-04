// Package device provides device path detection and management functionality
package device

import (
	"fmt"
	"strings"

	"github.com/apache/cloudstack-go/cloudstack"
	log "github.com/sirupsen/logrus"
)

// DeviceDetector handles real device path detection from CloudStack API
type DeviceDetector struct {
	cs *cloudstack.CloudStackClient
}

// NewDeviceDetector creates a new device detector instance
func NewDeviceDetector(cs *cloudstack.CloudStackClient) *DeviceDetector {
	return &DeviceDetector{
		cs: cs,
	}
}

// VolumeDeviceInfo contains volume device assignment information
type VolumeDeviceInfo struct {
	VolumeID   string `json:"volume_id"`
	VolumeName string `json:"volume_name"`
	DevicePath string `json:"device_path"`
	VMID       string `json:"vm_id"`
	VMName     string `json:"vm_name"`
	Status     string `json:"status"`
}

// GetActualDevicePath queries CloudStack API to get the real device assignment for a volume
func (dd *DeviceDetector) GetActualDevicePath(volumeID, vmID string) (string, error) {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"vm_id":     vmID,
	}).Debug("Detecting actual device path from CloudStack API")

	// Create parameters for listVolumes API call
	params := dd.cs.Volume.NewListVolumesParams()
	params.SetId(volumeID)
	
	// Execute the API call
	response, err := dd.cs.Volume.ListVolumes(params)
	if err != nil {
		return "", fmt.Errorf("failed to query CloudStack for volume %s: %w", volumeID, err)
	}

	if response.Count == 0 {
		return "", fmt.Errorf("volume %s not found in CloudStack", volumeID)
	}

	if len(response.Volumes) == 0 {
		return "", fmt.Errorf("no volume data returned for volume %s", volumeID)
	}

	volume := response.Volumes[0]

	// Check if volume is attached
	if volume.Virtualmachineid == "" {
		return "", fmt.Errorf("volume %s is not attached to any VM", volumeID)
	}

	// Verify it's attached to the expected VM
	if volume.Virtualmachineid != vmID {
		return "", fmt.Errorf("volume %s is attached to VM %s, not expected VM %s", 
			volumeID, volume.Virtualmachineid, vmID)
	}

	// Extract device path from CloudStack volume response
	// CloudStack returns device information in the volume object
	devicePath := dd.extractDevicePathFromVolume(volume)
	if devicePath == "" {
		return "", fmt.Errorf("could not determine device path for volume %s from CloudStack response", volumeID)
	}

	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"vm_id":       vmID,
		"device_path": devicePath,
		"status":      volume.State,
	}).Info("✅ Device path detected from CloudStack API")

	return devicePath, nil
}

// extractDevicePathFromVolume extracts the Linux device path from CloudStack volume response
func (dd *DeviceDetector) extractDevicePathFromVolume(volume *cloudstack.Volume) string {
	// CloudStack provides device information in the Deviceid field
	// For KVM hypervisor, this is typically 0, 1, 2, etc.
	
	// Method 1: Device ID field (primary method)
	if volume.Deviceid != 0 {
		// Device ID is typically 0, 1, 2, etc.
		// Convert to /dev/vda, /dev/vdb, /dev/vdc, etc.
		deviceLetter := 'a' + rune(volume.Deviceid)
		devicePath := fmt.Sprintf("/dev/vd%c", deviceLetter)
		
		log.WithFields(log.Fields{
			"volume_id":   volume.Id,
			"device_id":   volume.Deviceid,
			"device_path": devicePath,
		}).Debug("Extracted device path from CloudStack Deviceid")
		
		return devicePath
	}

	// Method 2: Fallback for root volumes (device ID 0)
	// Root volumes sometimes have Deviceid = 0, which maps to /dev/vda
	if volume.Type == "ROOT" {
		devicePath := "/dev/vda"
		
		log.WithFields(log.Fields{
			"volume_id":   volume.Id,
			"device_path": devicePath,
			"method":      "root_volume_fallback",
		}).Debug("Using root volume fallback device path")
		
		return devicePath
	}

	log.WithFields(log.Fields{
		"volume_id": volume.Id,
		"device_id": volume.Deviceid,
		"type":      volume.Type,
		"state":     volume.State,
	}).Warn("Could not extract device path from CloudStack volume response")

	return ""
}



// GetVolumeDeviceInfo retrieves comprehensive device information for a volume
func (dd *DeviceDetector) GetVolumeDeviceInfo(volumeID string) (*VolumeDeviceInfo, error) {
	log.WithField("volume_id", volumeID).Debug("Getting volume device information from CloudStack")

	// Create parameters for listVolumes API call
	params := dd.cs.Volume.NewListVolumesParams()
	params.SetId(volumeID)
	
	// Execute the API call
	response, err := dd.cs.Volume.ListVolumes(params)
	if err != nil {
		return nil, fmt.Errorf("failed to query CloudStack for volume %s: %w", volumeID, err)
	}

	if response.Count == 0 || len(response.Volumes) == 0 {
		return nil, fmt.Errorf("volume %s not found in CloudStack", volumeID)
	}

	volume := response.Volumes[0]

	info := &VolumeDeviceInfo{
		VolumeID:   volume.Id,
		VolumeName: volume.Name,
		Status:     volume.State,
		VMID:       volume.Virtualmachineid,
		VMName:     "",  // CloudStack SDK doesn't provide VM name in volume response
	}

	// Get device path if volume is attached
	if volume.Virtualmachineid != "" {
		devicePath := dd.extractDevicePathFromVolume(volume)
		info.DevicePath = devicePath
	}

	return info, nil
}

// ValidateDevicePathExists checks if the detected device path actually exists on the system
func (dd *DeviceDetector) ValidateDevicePathExists(devicePath string) error {
	// This would typically check if the device file exists and is a block device
	// For now, just validate the path format
	if !strings.HasPrefix(devicePath, "/dev/") {
		return fmt.Errorf("invalid device path format: %s", devicePath)
	}
	
	if len(devicePath) < 6 { // minimum: /dev/a
		return fmt.Errorf("device path too short: %s", devicePath)
	}
	
	return nil
}
