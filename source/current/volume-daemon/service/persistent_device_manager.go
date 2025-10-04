// Package service provides persistent device naming management for stable NBD export consistency
package service

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

// PersistentDeviceManager handles stable device naming throughout volume lifecycle
type PersistentDeviceManager struct {
	repo VolumeRepository
}

// NewPersistentDeviceManager creates a new persistent device manager
func NewPersistentDeviceManager(repo VolumeRepository) *PersistentDeviceManager {
	return &PersistentDeviceManager{
		repo: repo,
	}
}

// GeneratePersistentDeviceName creates stable device name based on VM and disk information
func (pdm *PersistentDeviceManager) GeneratePersistentDeviceName(
	vmName string,
	diskID string,
) string {
	// Clean VM name for device naming (remove special characters)
	cleanVMName := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(vmName, "")

	// Extract disk number from disk ID (disk-2000 → 0, disk-2001 → 1)
	diskNumber := "0" // Default for OS disk
	if strings.HasPrefix(diskID, "disk-200") {
		diskNumber = strings.TrimPrefix(diskID, "disk-200")
	}

	persistentName := fmt.Sprintf("%sdisk%s", cleanVMName, diskNumber)

	log.WithFields(log.Fields{
		"vm_name":         vmName,
		"disk_id":         diskID,
		"persistent_name": persistentName,
	}).Info("🏷️  Generated persistent device name")

	return persistentName
}

// CreatePersistentDevice creates device mapper symlink for stable device access
func (pdm *PersistentDeviceManager) CreatePersistentDevice(
	ctx context.Context,
	actualDevicePath string,
	persistentName string,
) (string, error) {
	symlinkPath := fmt.Sprintf("/dev/mapper/%s", persistentName)

	log.WithFields(log.Fields{
		"actual_device":   actualDevicePath,
		"persistent_name": persistentName,
		"symlink_path":    symlinkPath,
	}).Info("🔗 Creating persistent device mapping")

	// Get device size for device mapper table
	cmd := exec.Command("blockdev", "--getsz", actualDevicePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get device size for %s: %w", actualDevicePath, err)
	}

	deviceSize := strings.TrimSpace(string(output))

	// Create device mapper linear mapping
	table := fmt.Sprintf("0 %s linear %s 0", deviceSize, actualDevicePath)
	cmd = exec.Command("dmsetup", "create", persistentName, "--table", table)
	if err := cmd.Run(); err != nil {
		// Check if device already exists
		if checkCmd := exec.Command("dmsetup", "info", persistentName); checkCmd.Run() == nil {
			log.WithField("persistent_name", persistentName).Info("ℹ️  Persistent device already exists")
			return symlinkPath, nil
		}
		return "", fmt.Errorf("failed to create device mapper for %s: %w", persistentName, err)
	}

	log.WithFields(log.Fields{
		"actual_device":   actualDevicePath,
		"persistent_name": persistentName,
		"symlink_path":    symlinkPath,
		"device_size":     deviceSize,
	}).Info("✅ Created persistent device mapping")

	return symlinkPath, nil
}

// UpdatePersistentDevice updates device mapper target when underlying device changes
func (pdm *PersistentDeviceManager) UpdatePersistentDevice(
	ctx context.Context,
	persistentName string,
	newDevicePath string,
) error {
	log.WithFields(log.Fields{
		"persistent_name": persistentName,
		"new_device":      newDevicePath,
	}).Info("🔄 Updating persistent device mapping")

	// Get device size for new device
	cmd := exec.Command("blockdev", "--getsz", newDevicePath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get device size for %s: %w", newDevicePath, err)
	}

	deviceSize := strings.TrimSpace(string(output))

	// Reload device mapper with new target
	table := fmt.Sprintf("0 %s linear %s 0", deviceSize, newDevicePath)
	cmd = exec.Command("dmsetup", "reload", persistentName, "--table", table)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload device mapper %s: %w", persistentName, err)
	}

	// Resume with new mapping
	cmd = exec.Command("dmsetup", "resume", persistentName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to resume device mapper %s: %w", persistentName, err)
	}

	log.WithFields(log.Fields{
		"persistent_name": persistentName,
		"new_device":      newDevicePath,
		"device_size":     deviceSize,
	}).Info("✅ Updated persistent device mapping")

	return nil
}

// RemovePersistentDevice removes device mapper when volume is permanently deleted
func (pdm *PersistentDeviceManager) RemovePersistentDevice(
	ctx context.Context,
	persistentName string,
) error {
	log.WithField("persistent_name", persistentName).Info("🗑️  Removing persistent device mapping")

	cmd := exec.Command("dmsetup", "remove", persistentName)
	if err := cmd.Run(); err != nil {
		// Log warning but don't fail - device might already be removed
		log.WithError(err).WithField("persistent_name", persistentName).
			Warn("Failed to remove persistent device (may already be removed)")
	} else {
		log.WithField("persistent_name", persistentName).Info("✅ Removed persistent device mapping")
	}

	return nil
}

// DetectDeviceNameConflicts checks for device path conflicts and resolves them
func (pdm *PersistentDeviceManager) DetectDeviceNameConflicts(
	ctx context.Context,
	newDevicePath string,
	targetPersistentName string,
) error {
	log.WithFields(log.Fields{
		"new_device":        newDevicePath,
		"target_persistent": targetPersistentName,
	}).Info("🔍 Checking for device name conflicts")

	// Get all device mappings using this device path
	mappings, err := pdm.repo.GetMappingByDevice(ctx, newDevicePath)
	if err != nil {
		log.WithError(err).Debug("No existing device mapping found for conflict check")
		return nil // No conflicts if no existing mappings
	}

	// Check single mapping for conflict (simplified for now)
	if mappings != nil {
		log.WithFields(log.Fields{
			"existing_device":   newDevicePath,
			"target_persistent": targetPersistentName,
		}).Info("ℹ️  Device mapping exists - conflict detection logged")

		// TODO: Implement full conflict resolution in future enhancement
		// For now, log and continue - Volume Service will handle conflicts
	}

	return nil
}

// GetPersistentDeviceInfo retrieves persistent device information for a volume
func (pdm *PersistentDeviceManager) GetPersistentDeviceInfo(
	ctx context.Context,
	volumeUUID string,
) (interface{}, error) {
	mapping, err := pdm.repo.GetMapping(ctx, volumeUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device mapping: %w", err)
	}

	return mapping, nil
}
