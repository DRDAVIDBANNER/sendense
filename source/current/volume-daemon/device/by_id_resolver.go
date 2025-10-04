// Package device provides by-id device path resolution for CloudStack volumes
// This replaces complex size-based correlation with deterministic UUID-based resolution
package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// ByIDResolver handles CloudStack volume UUID to /dev/disk/by-id path resolution
type ByIDResolver struct {
	defaultTimeout time.Duration
}

// NewByIDResolver creates a new by-id path resolver
func NewByIDResolver() *ByIDResolver {
	return &ByIDResolver{
		defaultTimeout: 10 * time.Second, // Default timeout for symlink appearance
	}
}

// Default resolver instance for package-level functions
var defaultResolver = NewByIDResolver()

// ConstructByIDPath builds /dev/disk/by-id path from CloudStack volume UUID
// 
// CloudStack volume UUIDs are embedded in virtio device identifiers:
// Volume UUID: b3bb9310-1b59-4f62-97e8-cefffdfe3804
// by-id path:  /dev/disk/by-id/virtio-b3bb93101b594f6297e8
//
// Pattern: Remove hyphens, take first 20 chars, prefix with virtio-
func ConstructByIDPath(volumeID string) string {
	// Remove all hyphens from UUID
	cleanUUID := strings.ReplaceAll(volumeID, "-", "")
	
	// Take first 20 characters (virtio device identifier limit)
	if len(cleanUUID) < 20 {
		log.WithFields(log.Fields{
			"volume_id":  volumeID,
			"clean_uuid": cleanUUID,
			"length":     len(cleanUUID),
		}).Warn("Volume UUID shorter than expected after cleaning")
		// Use what we have if UUID is shorter
	}
	
	shortID := cleanUUID
	if len(cleanUUID) >= 20 {
		shortID = cleanUUID[:20]
	}
	
	// Construct by-id path
	byIDPath := fmt.Sprintf("/dev/disk/by-id/virtio-%s", shortID)
	
	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"clean_uuid":  cleanUUID,
		"short_id":    shortID,
		"by_id_path":  byIDPath,
	}).Debug("Constructed by-id path from volume UUID")
	
	return byIDPath
}

// WaitForByIDSymlink waits for /dev/disk/by-id symlink to appear and resolves it
// Returns the actual device path that the symlink points to
func WaitForByIDSymlink(byIDPath string, timeout time.Duration) (string, error) {
	log.WithFields(log.Fields{
		"by_id_path": byIDPath,
		"timeout":    timeout,
	}).Info("🔍 Waiting for by-id symlink to appear")
	
	deadline := time.Now().Add(timeout)
	attempts := 0
	
	for time.Now().Before(deadline) {
		attempts++
		
		// Check if symlink exists
		if _, err := os.Lstat(byIDPath); err == nil {
			// Symlink exists, resolve it to actual device
			devicePath, err := filepath.EvalSymlinks(byIDPath)
			if err != nil {
				log.WithFields(log.Fields{
					"by_id_path": byIDPath,
					"attempts":   attempts,
					"error":      err,
				}).Debug("by-id symlink exists but resolution failed")
				
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			// Verify resolved device exists and is readable
			if _, err := os.Stat(devicePath); err != nil {
				log.WithFields(log.Fields{
					"by_id_path":  byIDPath,
					"device_path": devicePath,
					"attempts":    attempts,
					"error":       err,
				}).Debug("Resolved device path not accessible")
				
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			log.WithFields(log.Fields{
				"by_id_path":  byIDPath,
				"device_path": devicePath,
				"attempts":    attempts,
				"elapsed":     time.Since(deadline.Add(-timeout)),
			}).Info("✅ by-id symlink resolved successfully")
			
			return devicePath, nil
		}
		
		// Symlink doesn't exist yet, wait a bit
		if attempts%50 == 0 { // Log every 5 seconds
			log.WithFields(log.Fields{
				"by_id_path": byIDPath,
				"attempts":   attempts,
				"elapsed":    time.Since(deadline.Add(-timeout)),
			}).Debug("Still waiting for by-id symlink")
		}
		
		time.Sleep(100 * time.Millisecond)
	}
	
	log.WithFields(log.Fields{
		"by_id_path": byIDPath,
		"timeout":    timeout,
		"attempts":   attempts,
	}).Error("❌ Timeout waiting for by-id symlink")
	
	return "", fmt.Errorf("timeout waiting for by-id symlink: %s (waited %v, %d attempts)", 
		byIDPath, timeout, attempts)
}

// GetDeviceByVolumeID resolves CloudStack volume ID to Linux device path using by-id
// This is the main entry point that replaces complex size-based correlation
func GetDeviceByVolumeID(volumeID string, timeout time.Duration) (byIDPath string, devicePath string, err error) {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"timeout":   timeout,
	}).Info("🔍 Resolving volume ID to device path via by-id")
	
	// Step 1: Construct expected by-id path
	byIDPath = ConstructByIDPath(volumeID)
	
	// Step 2: Wait for symlink to appear and resolve it
	devicePath, err = WaitForByIDSymlink(byIDPath, timeout)
	if err != nil {
		return byIDPath, "", fmt.Errorf("failed to resolve device for volume %s: %w", volumeID, err)
	}
	
	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"by_id_path":  byIDPath,
		"device_path": devicePath,
	}).Info("✅ Volume resolved to device via by-id path")
	
	return byIDPath, devicePath, nil
}

// GetDeviceByVolumeIDWithDefault uses default timeout for convenience
func (r *ByIDResolver) GetDeviceByVolumeID(volumeID string) (byIDPath string, devicePath string, err error) {
	return GetDeviceByVolumeID(volumeID, r.defaultTimeout)
}

// ValidateByIDPath checks if a by-id path exists and is accessible
func ValidateByIDPath(byIDPath string) (string, error) {
	// Check if symlink exists
	if _, err := os.Lstat(byIDPath); err != nil {
		return "", fmt.Errorf("by-id path does not exist: %s", byIDPath)
	}
	
	// Resolve symlink
	devicePath, err := filepath.EvalSymlinks(byIDPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve by-id symlink %s: %w", byIDPath, err)
	}
	
	// Verify device is accessible
	if _, err := os.Stat(devicePath); err != nil {
		return "", fmt.Errorf("resolved device not accessible %s: %w", devicePath, err)
	}
	
	return devicePath, nil
}

// ListAllByIDPaths returns all virtio by-id paths currently available
func ListAllByIDPaths() (map[string]string, error) {
	byIDDir := "/dev/disk/by-id"
	
	entries, err := os.ReadDir(byIDDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read by-id directory: %w", err)
	}
	
	virtioDevices := make(map[string]string)
	
	for _, entry := range entries {
		name := entry.Name()
		
		// Only process virtio devices
		if !strings.HasPrefix(name, "virtio-") {
			continue
		}
		
		// Skip partition entries
		if strings.Contains(name, "-part") {
			continue
		}
		
		byIDPath := filepath.Join(byIDDir, name)
		
		// Resolve to actual device
		devicePath, err := filepath.EvalSymlinks(byIDPath)
		if err != nil {
			log.WithFields(log.Fields{
				"by_id_path": byIDPath,
				"error":      err,
			}).Debug("Failed to resolve by-id symlink")
			continue
		}
		
		virtioDevices[byIDPath] = devicePath
	}
	
	log.WithField("device_count", len(virtioDevices)).Debug("Listed all virtio by-id devices")
	return virtioDevices, nil
}

// ExtractVolumeIDFromByIDPath attempts to extract CloudStack volume ID from by-id path
// This is useful for reverse lookups and validation
func ExtractVolumeIDFromByIDPath(byIDPath string) (string, error) {
	// Extract the virtio identifier: /dev/disk/by-id/virtio-b3bb93101b594f6297e8
	basename := filepath.Base(byIDPath)
	
	if !strings.HasPrefix(basename, "virtio-") {
		return "", fmt.Errorf("not a virtio by-id path: %s", byIDPath)
	}
	
	shortID := strings.TrimPrefix(basename, "virtio-")
	
	if len(shortID) != 20 {
		return "", fmt.Errorf("invalid virtio ID length %d (expected 20): %s", len(shortID), shortID)
	}
	
	// This is a partial reconstruction - we only have first 20 chars
	// Full volume ID reconstruction would require database lookup
	log.WithFields(log.Fields{
		"by_id_path": byIDPath,
		"short_id":   shortID,
	}).Debug("Extracted partial volume ID from by-id path")
	
	return shortID, nil
}

// GetDeviceSize returns the size of a device in bytes
func GetDeviceSize(devicePath string) (int64, error) {
	// Use blockdev to get size in bytes
	file, err := os.Open(devicePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open device %s: %w", devicePath, err)
	}
	defer file.Close()
	
	// Seek to end to get size
	size, err := file.Seek(0, 2) // SEEK_END
	if err != nil {
		return 0, fmt.Errorf("failed to get device size %s: %w", devicePath, err)
	}
	
	return size, nil
}

// DeviceInfo represents information about a device resolved via by-id
type DeviceInfo struct {
	VolumeID     string `json:"volume_id"`      // CloudStack volume UUID (if known)
	ByIDPath     string `json:"by_id_path"`     // Stable by-id path
	DevicePath   string `json:"device_path"`    // Current device path (/dev/vdX)
	Size         int64  `json:"size"`           // Device size in bytes
	ResolvedAt   time.Time `json:"resolved_at"` // When resolution completed
	ResolutionTime time.Duration `json:"resolution_time"` // How long resolution took
}

// ResolveDeviceInfo gets complete device information via by-id resolution
func (r *ByIDResolver) ResolveDeviceInfo(volumeID string) (*DeviceInfo, error) {
	startTime := time.Now()
	
	byIDPath, devicePath, err := r.GetDeviceByVolumeID(volumeID)
	if err != nil {
		return nil, err
	}
	
	// Get device size
	size, err := GetDeviceSize(devicePath)
	if err != nil {
		log.WithError(err).WithField("device_path", devicePath).Warn("Failed to get device size")
		size = 0 // Continue without size if we can't get it
	}
	
	resolutionTime := time.Since(startTime)
	
	info := &DeviceInfo{
		VolumeID:       volumeID,
		ByIDPath:       byIDPath,
		DevicePath:     devicePath,
		Size:           size,
		ResolvedAt:     time.Now(),
		ResolutionTime: resolutionTime,
	}
	
	log.WithFields(log.Fields{
		"volume_id":       volumeID,
		"by_id_path":      byIDPath,
		"device_path":     devicePath,
		"size_gb":         size / (1024 * 1024 * 1024),
		"resolution_time": resolutionTime,
	}).Info("✅ Complete device info resolved via by-id")
	
	return info, nil
}
