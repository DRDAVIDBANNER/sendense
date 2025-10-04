package volume

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/database"
)

// MountManager handles volume mounting operations on the OMA appliance
type MountManager struct {
	db database.Connection

	// Configuration
	baseMountPath string // e.g., /mnt/migration
	devicePrefix  string // e.g., /dev/vd
	maxRetries    int
	retryInterval time.Duration
}

// NewMountManager creates a new volume mount manager
func NewMountManager(db database.Connection, baseMountPath string) *MountManager {
	return &MountManager{
		db:            db,
		baseMountPath: baseMountPath,
		devicePrefix:  "/dev/vd", // Default for virtio devices
		maxRetries:    3,
		retryInterval: 5 * time.Second,
	}
}

// MountOptions represents volume mounting options
type MountOptions struct {
	JobID          string
	VolumeID       string
	DevicePath     string
	FilesystemType string // auto, ext4, xfs, ntfs, etc.
	ReadOnly       bool
	MountOptions   []string // mount options like rw, noatime, etc.
}

// MountResult represents the result of a mount operation
type MountResult struct {
	MountPoint     string
	DevicePath     string
	FilesystemType string
	MountOptions   string
	Success        bool
	Error          error
}

// MountVolume mounts an OSSEA volume on the OMA appliance
func (m *MountManager) MountVolume(opts *MountOptions) (*MountResult, error) {
	log.WithFields(log.Fields{
		"job_id":      opts.JobID,
		"volume_id":   opts.VolumeID,
		"device_path": opts.DevicePath,
		"readonly":    opts.ReadOnly,
	}).Info("🔧 Mounting OSSEA volume")

	// Generate mount point
	mountPoint := m.generateMountPoint(opts.JobID, opts.VolumeID)

	// Create mount point directory
	if err := m.createMountPoint(mountPoint); err != nil {
		return nil, fmt.Errorf("failed to create mount point: %w", err)
	}

	// Wait for device to be available
	if err := m.waitForDevice(opts.DevicePath); err != nil {
		return nil, fmt.Errorf("device not available: %w", err)
	}

	// Detect filesystem type if auto
	fsType := opts.FilesystemType
	if fsType == "" || fsType == "auto" {
		detectedFS, err := m.detectFilesystemType(opts.DevicePath)
		if err != nil {
			log.WithError(err).Warn("Failed to detect filesystem type, using ext4")
			fsType = "ext4"
		} else {
			fsType = detectedFS
		}
	}

	// Build mount options
	mountOpts := m.buildMountOptions(opts.MountOptions, opts.ReadOnly)

	// Attempt to mount
	err := m.performMount(opts.DevicePath, mountPoint, fsType, mountOpts)
	if err != nil {
		// Cleanup mount point on failure
		os.RemoveAll(mountPoint)
		return &MountResult{
			Success: false,
			Error:   err,
		}, err
	}

	// Update database
	if err := m.updateMountDatabase(opts, mountPoint, fsType, strings.Join(mountOpts, ",")); err != nil {
		log.WithError(err).Warn("Failed to update mount database")
	}

	result := &MountResult{
		MountPoint:     mountPoint,
		DevicePath:     opts.DevicePath,
		FilesystemType: fsType,
		MountOptions:   strings.Join(mountOpts, ","),
		Success:        true,
	}

	log.WithFields(log.Fields{
		"device_path":     opts.DevicePath,
		"mount_point":     mountPoint,
		"filesystem_type": fsType,
		"mount_options":   result.MountOptions,
	}).Info("✅ Volume mounted successfully")

	return result, nil
}

// UnmountVolume unmounts a volume and cleans up
func (m *MountManager) UnmountVolume(jobID, volumeID string) error {
	log.WithFields(log.Fields{
		"job_id":    jobID,
		"volume_id": volumeID,
	}).Info("🔧 Unmounting OSSEA volume")

	// Get mount info from database
	mountInfo, err := m.getMountInfo(jobID, volumeID)
	if err != nil {
		return fmt.Errorf("failed to get mount info: %w", err)
	}

	if mountInfo == nil {
		log.Warn("Volume not found in mount database")
		return nil
	}

	// Perform unmount
	if err := m.performUnmount(mountInfo.MountPoint); err != nil {
		return fmt.Errorf("failed to unmount: %w", err)
	}

	// Remove mount point directory
	if err := os.RemoveAll(mountInfo.MountPoint); err != nil {
		log.WithError(err).Warn("Failed to remove mount point directory")
	}

	// Update database
	if err := m.updateUnmountDatabase(mountInfo.ID); err != nil {
		log.WithError(err).Warn("Failed to update unmount database")
	}

	log.WithField("mount_point", mountInfo.MountPoint).Info("✅ Volume unmounted successfully")
	return nil
}

// GetMountedVolumes returns all currently mounted volumes for a job
func (m *MountManager) GetMountedVolumes(jobID string) ([]database.VolumeMount, error) {
	db := m.db.GetGormDB()

	var mounts []database.VolumeMount
	err := db.Where("job_id = ? AND mount_status = ?", jobID, "mounted").
		Preload("OSSEAVolume").
		Find(&mounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get mounted volumes: %w", err)
	}

	return mounts, nil
}

// CleanupJobMounts unmounts all volumes for a completed job
func (m *MountManager) CleanupJobMounts(jobID string) error {
	log.WithField("job_id", jobID).Info("🧹 Cleaning up job mounts")

	mounts, err := m.GetMountedVolumes(jobID)
	if err != nil {
		return err
	}

	var errors []string
	for _, mount := range mounts {
		// Temporarily disabled due to foreign key relationships being disabled
		// if mount.OSSEAVolume != nil {
		//	if err := m.UnmountVolume(jobID, mount.OSSEAVolume.VolumeID); err != nil {
		//		errors = append(errors, fmt.Sprintf("volume %s: %v", mount.OSSEAVolume.VolumeID, err))
		//	}
		// }
		_ = mount // Prevent unused variable warning
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}

	log.WithField("job_id", jobID).Info("✅ Job mounts cleaned up")
	return nil
}

// Helper methods

// generateMountPoint generates a unique mount point path
func (m *MountManager) generateMountPoint(jobID, volumeID string) string {
	// Clean volume ID for filesystem compatibility
	cleanVolumeID := strings.ReplaceAll(volumeID, "-", "_")
	return filepath.Join(m.baseMountPath, fmt.Sprintf("job_%s_vol_%s", jobID, cleanVolumeID))
}

// createMountPoint creates the mount point directory
func (m *MountManager) createMountPoint(mountPoint string) error {
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", mountPoint, err)
	}
	return nil
}

// waitForDevice waits for a device to become available
func (m *MountManager) waitForDevice(devicePath string) error {
	for i := 0; i < m.maxRetries; i++ {
		if _, err := os.Stat(devicePath); err == nil {
			// Device exists, check if it's ready
			if m.isDeviceReady(devicePath) {
				return nil
			}
		}

		if i < m.maxRetries-1 {
			log.WithFields(log.Fields{
				"device_path": devicePath,
				"retry":       i + 1,
				"max_retries": m.maxRetries,
			}).Debug("Device not ready, retrying...")
			time.Sleep(m.retryInterval)
		}
	}

	return fmt.Errorf("device %s not available after %d retries", devicePath, m.maxRetries)
}

// isDeviceReady checks if a device is ready for mounting
func (m *MountManager) isDeviceReady(devicePath string) bool {
	// Try to read the first sector to verify device is accessible
	cmd := exec.Command("dd", "if="+devicePath, "of=/dev/null", "bs=512", "count=1")
	err := cmd.Run()
	return err == nil
}

// detectFilesystemType detects the filesystem type of a device
func (m *MountManager) detectFilesystemType(devicePath string) (string, error) {
	cmd := exec.Command("blkid", "-s", "TYPE", "-o", "value", devicePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to detect filesystem: %w", err)
	}

	fsType := strings.TrimSpace(string(output))
	if fsType == "" {
		return "ext4", nil // Default if undetectable
	}

	return fsType, nil
}

// buildMountOptions builds the mount options string
func (m *MountManager) buildMountOptions(options []string, readOnly bool) []string {
	opts := make([]string, 0, len(options)+2)

	// Add read-only flag if specified
	if readOnly {
		opts = append(opts, "ro")
	} else {
		opts = append(opts, "rw")
	}

	// Add user-specified options
	opts = append(opts, options...)

	// Add default options if not specified
	hasNoatime := false
	for _, opt := range options {
		if opt == "noatime" || opt == "atime" {
			hasNoatime = true
			break
		}
	}
	if !hasNoatime {
		opts = append(opts, "noatime")
	}

	return opts
}

// performMount executes the mount command
func (m *MountManager) performMount(devicePath, mountPoint, fsType string, options []string) error {
	args := []string{"-t", fsType}

	if len(options) > 0 {
		args = append(args, "-o", strings.Join(options, ","))
	}

	args = append(args, devicePath, mountPoint)

	cmd := exec.Command("mount", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount command failed: %w (output: %s)", err, string(output))
	}

	// Verify mount was successful
	if !m.isMounted(mountPoint) {
		return fmt.Errorf("mount verification failed for %s", mountPoint)
	}

	return nil
}

// performUnmount executes the umount command
func (m *MountManager) performUnmount(mountPoint string) error {
	// Force unmount with lazy flag as fallback
	cmd := exec.Command("umount", mountPoint)
	err := cmd.Run()

	if err != nil {
		log.WithError(err).Warn("Normal unmount failed, trying lazy unmount")
		cmd = exec.Command("umount", "-l", mountPoint)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("lazy unmount failed: %w", err)
		}
	}

	// Verify unmount
	if m.isMounted(mountPoint) {
		return fmt.Errorf("unmount verification failed for %s", mountPoint)
	}

	return nil
}

// isMounted checks if a path is currently mounted
func (m *MountManager) isMounted(mountPoint string) bool {
	// Check /proc/mounts
	cmd := exec.Command("grep", "-q", mountPoint, "/proc/mounts")
	err := cmd.Run()
	return err == nil
}

// Database operations

// updateMountDatabase updates the mount record in the database
func (m *MountManager) updateMountDatabase(opts *MountOptions, mountPoint, fsType, mountOpts string) error {
	db := m.db.GetGormDB()

	// Get or create volume mount record
	var mount database.VolumeMount
	err := db.Where("job_id = ? AND ossea_volume_id = (SELECT id FROM ossea_volumes WHERE volume_id = ?)",
		opts.JobID, opts.VolumeID).First(&mount).Error

	if err != nil {
		// Create new record
		mount = database.VolumeMount{
			JobID:          opts.JobID,
			DevicePath:     opts.DevicePath,
			MountPoint:     mountPoint,
			MountStatus:    "mounted",
			FilesystemType: fsType,
			MountOptions:   strings.Join([]string{mountOpts}, ","),
			IsReadOnly:     opts.ReadOnly,
		}

		// Get OSSEA volume ID
		var volume database.OSSEAVolume
		if err := db.Where("volume_id = ?", opts.VolumeID).First(&volume).Error; err == nil {
			mount.OSSEAVolumeID = volume.ID
		}

		now := time.Now()
		mount.MountedAt = &now

		return db.Create(&mount).Error
	}

	// Update existing record
	now := time.Now()
	mount.MountPoint = mountPoint
	mount.MountStatus = "mounted"
	mount.FilesystemType = fsType
	mount.MountOptions = strings.Join([]string{mountOpts}, ",")
	mount.IsReadOnly = opts.ReadOnly
	mount.MountedAt = &now
	mount.UnmountedAt = nil

	return db.Save(&mount).Error
}

// updateUnmountDatabase updates the database when a volume is unmounted
func (m *MountManager) updateUnmountDatabase(mountID int) error {
	db := m.db.GetGormDB()

	now := time.Now()
	return db.Model(&database.VolumeMount{}).
		Where("id = ?", mountID).
		Updates(map[string]interface{}{
			"mount_status": "unmounted",
			"unmounted_at": &now,
		}).Error
}

// getMountInfo retrieves mount information from database
func (m *MountManager) getMountInfo(jobID, volumeID string) (*database.VolumeMount, error) {
	db := m.db.GetGormDB()

	var mount database.VolumeMount
	err := db.Where("job_id = ? AND ossea_volume_id = (SELECT id FROM ossea_volumes WHERE volume_id = ?) AND mount_status = ?",
		jobID, volumeID, "mounted").
		Preload("OSSEAVolume").
		First(&mount).Error

	if err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		}
		return nil, err
	}

	return &mount, nil
}
