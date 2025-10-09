// Package restore provides QCOW2 backup mount management for file-level restore
// Task 4: File-Level Restore (Phase 1 - QCOW2 Mount Management)
// COMPLIANCE: Uses repository pattern, qemu-nbd for QCOW2 mounting, NBD device allocation
package restore

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/storage"
)

const (
	// NBD device allocation strategy (Task 4 integration requirement)
	// Restore operations use /dev/nbd0-7 (8 devices for concurrent mounts)
	// Backup operations use /dev/nbd8+ (separate allocation pool)
	RestoreNBDDeviceStart = 0
	RestoreNBDDeviceEnd   = 7
	RestoreNBDDeviceCount = 8

	// Mount configuration
	RestoreMountBaseDir  = "/mnt/sendense/restore"
	DefaultIdleTimeout   = 1 * time.Hour
	DefaultMaxMounts     = 8
)

// MountManager handles QCOW2 backup mounting via qemu-nbd
type MountManager struct {
	// Dependencies (repository pattern)
	mountRepo         *database.RestoreMountRepository
	repositoryManager *storage.RepositoryManager
	db                database.Connection // v2.16.0+: Direct DB access for backup_disks queries

	// Configuration
	mountBaseDir string
	idleTimeout  time.Duration
	maxMounts    int
}

// NewMountManager creates a new mount manager instance
func NewMountManager(
	mountRepo *database.RestoreMountRepository,
	repositoryManager *storage.RepositoryManager,
	db database.Connection, // v2.16.0+: For backup_disks queries
) *MountManager {
	return &MountManager{
		mountRepo:         mountRepo,
		repositoryManager: repositoryManager,
		db:                db,
		mountBaseDir:      RestoreMountBaseDir,
		idleTimeout:       DefaultIdleTimeout,
		maxMounts:         DefaultMaxMounts,
	}
}

// MountRequest represents a request to mount a backup disk for browsing
// v2.16.0+: Multi-disk support requires disk_index parameter
type MountRequest struct {
	BackupID  string `json:"backup_id"`            // Required: Parent backup job ID
	DiskIndex int    `json:"disk_index,omitempty"` // Required: Which disk to mount (0, 1, 2...) - defaults to 0 for backward compat
}

// MountInfo contains information about an active mount
// v2.16.0+: Includes disk_index for multi-disk VM support
type MountInfo struct {
	MountID        string     `json:"mount_id"`
	BackupID       string     `json:"backup_id"`
	BackupDiskID   int64      `json:"backup_disk_id"`   // v2.16.0+: FK to backup_disks.id
	DiskIndex      int        `json:"disk_index"`       // v2.16.0+: Which disk (0, 1, 2...)
	MountPath      string     `json:"mount_path"`
	NBDDevice      string     `json:"nbd_device"`
	FilesystemType string     `json:"filesystem_type"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// MountBackup mounts a QCOW2 backup disk for file browsing
// v2.16.0+: Multi-disk support - mounts specific disk from multi-disk backups
// This is the main entry point for mounting backups
func (mm *MountManager) MountBackup(ctx context.Context, req *MountRequest) (*MountInfo, error) {
	log.WithFields(log.Fields{
		"backup_id":  req.BackupID,
		"disk_index": req.DiskIndex,
	}).Info("🔗 Starting QCOW2 backup mount operation (v2.16.0+ multi-disk support)")

	// v2.16.0+: Find the specific backup disk in the new schema
	backupDiskID, backupFile, err := mm.findBackupDiskFile(ctx, req.BackupID, req.DiskIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to find backup disk file: %w", err)
	}

	log.WithFields(log.Fields{
		"backup_disk_id": backupDiskID,
		"qcow2_path":     backupFile,
		"disk_index":     req.DiskIndex,
	}).Info("📁 Located backup disk file from backup_disks table")

	// v2.16.0+: Check if mount already exists for this specific disk
	existingMounts, err := mm.mountRepo.GetByBackupDiskID(ctx, backupDiskID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing mounts: %w", err)
	}
	if len(existingMounts) > 0 {
		existing := existingMounts[0]
		log.WithFields(log.Fields{
			"mount_id":       existing.ID,
			"mount_path":     existing.MountPath,
			"backup_disk_id": backupDiskID,
			"disk_index":     req.DiskIndex,
		}).Info("♻️  Reusing existing mount for backup disk")

		// Update last accessed time
		mm.mountRepo.UpdateLastAccessed(ctx, existing.ID)

		return &MountInfo{
			MountID:        existing.ID,
			BackupID:       req.BackupID,
			BackupDiskID:   backupDiskID,
			DiskIndex:      req.DiskIndex,
			MountPath:      existing.MountPath,
			NBDDevice:      existing.NBDDevice,
			FilesystemType: existing.FilesystemType,
			Status:         existing.Status,
			CreatedAt:      existing.CreatedAt,
			ExpiresAt:      existing.ExpiresAt,
		}, nil
	}

	// Check mount limits
	activeCount, err := mm.mountRepo.CountActiveMounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active mounts: %w", err)
	}
	if activeCount >= mm.maxMounts {
		return nil, fmt.Errorf("maximum concurrent mounts reached (%d/%d) - please wait for cleanup or unmount unused backups", activeCount, mm.maxMounts)
	}

	// Allocate NBD device
	nbdDevice, err := mm.allocateNBDDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate NBD device: %w", err)
	}

	log.WithField("nbd_device", nbdDevice).Info("📀 Allocated NBD device")

	// Generate mount ID and path
	mountID := uuid.New().String()
	mountPath := filepath.Join(mm.mountBaseDir, mountID)

	// v2.16.0+: Create mount record with backup_disk_id FK
	now := time.Now()
	expiresAt := now.Add(mm.idleTimeout)
	mount := &database.RestoreMount{
		ID:             mountID,
		BackupDiskID:   backupDiskID, // v2.16.0+: FK to backup_disks.id
		MountPath:      mountPath,
		NBDDevice:      nbdDevice,
		FilesystemType: "", // Will be detected
		MountMode:      "read-only",
		Status:         "mounting",
		CreatedAt:      now,
		LastAccessedAt: now,
		ExpiresAt:      &expiresAt,
	}

	if err := mm.mountRepo.Create(ctx, mount); err != nil {
		return nil, fmt.Errorf("failed to create mount record: %w", err)
	}

	// Perform actual mount operation (qemu-nbd + filesystem mount)
	if err := mm.performMount(ctx, backupFile, nbdDevice, mountPath, mountID); err != nil {
		// Cleanup: Update status to failed
		mm.mountRepo.UpdateStatus(ctx, mountID, "failed")
		return nil, fmt.Errorf("mount operation failed: %w", err)
	}

	// Detect filesystem type
	filesystemType, err := mm.detectFilesystem(nbdDevice)
	if err != nil {
		log.WithError(err).Warn("Failed to detect filesystem type - continuing")
		filesystemType = "unknown"
	}

	// Update mount record with filesystem type and status: mounted
	updateCtx := context.Background()
	updateData := map[string]interface{}{
		"filesystem_type": filesystemType,
		"status":          "mounted",
	}
	if err := mm.mountRepo.UpdateFields(updateCtx, mountID, updateData); err != nil {
		log.WithError(err).Warn("Failed to update filesystem type - continuing")
	}

	log.WithFields(log.Fields{
		"mount_id":        mountID,
		"mount_path":      mountPath,
		"nbd_device":      nbdDevice,
		"filesystem_type": filesystemType,
		"backup_disk_id":  backupDiskID,
		"disk_index":      req.DiskIndex,
	}).Info("✅ QCOW2 backup disk mounted successfully (v2.16.0+ multi-disk support)")

	return &MountInfo{
		MountID:        mountID,
		BackupID:       req.BackupID,
		BackupDiskID:   backupDiskID,
		DiskIndex:      req.DiskIndex,
		MountPath:      mountPath,
		NBDDevice:      nbdDevice,
		FilesystemType: filesystemType,
		Status:         "mounted",
		CreatedAt:      now,
		ExpiresAt:      &expiresAt,
	}, nil
}

// UnmountBackup unmounts a QCOW2 backup
func (mm *MountManager) UnmountBackup(ctx context.Context, mountID string) error {
	log.WithField("mount_id", mountID).Info("🔓 Starting backup unmount operation")

	// Get mount record
	mount, err := mm.mountRepo.GetByID(ctx, mountID)
	if err != nil {
		return fmt.Errorf("mount not found: %w", err)
	}

	// Update status to unmounting
	mm.mountRepo.UpdateStatus(ctx, mountID, "unmounting")

	// Perform actual unmount operation
	if err := mm.performUnmount(ctx, mount.MountPath, mount.NBDDevice); err != nil {
		return fmt.Errorf("unmount operation failed: %w", err)
	}

	// Delete mount record
	if err := mm.mountRepo.Delete(ctx, mountID); err != nil {
		log.WithError(err).Warn("Failed to delete mount record after successful unmount")
	}

	log.WithField("mount_id", mountID).Info("✅ Backup unmounted successfully")
	return nil
}

// ListMounts returns all active mounts
func (mm *MountManager) ListMounts(ctx context.Context) ([]*MountInfo, error) {
	log.Debug("📋 Listing active restore mounts")

	mounts, err := mm.mountRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active mounts: %w", err)
	}

	result := make([]*MountInfo, len(mounts))
	for i, mount := range mounts {
		result[i] = &MountInfo{
			MountID:        mount.ID,
			BackupID:       "", // v2.16.0+: Parent backup_id not stored, only backup_disk_id
			BackupDiskID:   mount.BackupDiskID,
			DiskIndex:      0, // TODO: Could query backup_disks to get disk_index
			MountPath:      mount.MountPath,
			NBDDevice:      mount.NBDDevice,
			FilesystemType: mount.FilesystemType,
			Status:         mount.Status,
			CreatedAt:      mount.CreatedAt,
			ExpiresAt:      mount.ExpiresAt,
		}
	}

	return result, nil
}

// findBackupDiskFile locates the QCOW2 file for a specific disk (v2.16.0+ schema)
// Returns: (backup_disk_id, qcow2_path, error)
func (mm *MountManager) findBackupDiskFile(ctx context.Context, backupID string, diskIndex int) (int64, string, error) {
	log.WithFields(log.Fields{
		"backup_id":  backupID,
		"disk_index": diskIndex,
	}).Debug("🔍 Querying backup_disks table for QCOW2 file")

	// v2.16.0+: Query backup_disks table directly
	// The new architecture stores QCOW2 paths in backup_disks, not backup_jobs
	var disk struct {
		ID         int64  `gorm:"column:id"`
		QCOW2Path  string `gorm:"column:qcow2_path"`
		Status     string `gorm:"column:status"`
		DiskIndex  int    `gorm:"column:disk_index"`
	}

	err := mm.db.GetGormDB().WithContext(ctx).
		Table("backup_disks").
		Select("id, qcow2_path, status, disk_index").
		Where("backup_job_id = ? AND disk_index = ? AND status = ?", backupID, diskIndex, "completed").
		First(&disk).Error

	if err != nil {
		return 0, "", fmt.Errorf("disk not found: backup_id=%s, disk_index=%d: %w", backupID, diskIndex, err)
	}

	log.WithFields(log.Fields{
		"backup_disk_id": disk.ID,
		"qcow2_path":     disk.QCOW2Path,
		"disk_index":     disk.DiskIndex,
	}).Debug("✅ Found QCOW2 file in backup_disks table")

	// Validate file exists on filesystem
	if _, err := os.Stat(disk.QCOW2Path); os.IsNotExist(err) {
		return 0, "", fmt.Errorf("QCOW2 file does not exist: %s", disk.QCOW2Path)
	}

	return disk.ID, disk.QCOW2Path, nil
}

// allocateNBDDevice finds an available NBD device from the restore pool (/dev/nbd0-7)
func (mm *MountManager) allocateNBDDevice(ctx context.Context) (string, error) {
	log.Debug("🎯 Allocating NBD device from restore pool (/dev/nbd0-7)")

	// Get currently allocated devices
	allocatedDevices, err := mm.mountRepo.GetAllocatedNBDDevices(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get allocated devices: %w", err)
	}

	// Build set of allocated devices for quick lookup
	allocated := make(map[string]bool)
	for _, device := range allocatedDevices {
		allocated[device] = true
	}

	// Find first available device in restore pool
	for i := RestoreNBDDeviceStart; i <= RestoreNBDDeviceEnd; i++ {
		device := fmt.Sprintf("/dev/nbd%d", i)
		if !allocated[device] {
			log.WithField("nbd_device", device).Debug("✅ Allocated available NBD device")
			return device, nil
		}
	}

	return "", fmt.Errorf("no available NBD devices in restore pool (/dev/nbd0-7) - %d mounts active", len(allocatedDevices))
}

// performMount executes the actual mount operation (qemu-nbd + filesystem mount)
func (mm *MountManager) performMount(ctx context.Context, backupFile, nbdDevice, mountPath, mountID string) error {
	log.WithFields(log.Fields{
		"backup_file": backupFile,
		"nbd_device":  nbdDevice,
		"mount_path":  mountPath,
	}).Info("🔧 Executing mount operation")

	// Step 1: Export QCOW2 via qemu-nbd
	if err := mm.exportQCOW2(backupFile, nbdDevice); err != nil {
		return fmt.Errorf("failed to export QCOW2: %w", err)
	}

	// Step 2: Wait for NBD device to be ready
	if err := mm.waitForNBDDevice(nbdDevice); err != nil {
		// Cleanup: Disconnect qemu-nbd
		mm.disconnectNBD(nbdDevice)
		return fmt.Errorf("NBD device not ready: %w", err)
	}

	// Step 3: Detect partition (usually nbd device + p1 for first partition)
	partition := mm.detectPartition(nbdDevice)

	// Step 4: Create mount point directory
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		mm.disconnectNBD(nbdDevice)
		return fmt.Errorf("failed to create mount directory: %w", err)
	}

	// Step 5: Mount filesystem (read-only)
	if err := mm.mountFilesystem(partition, mountPath); err != nil {
		os.RemoveAll(mountPath)
		mm.disconnectNBD(nbdDevice)
		return fmt.Errorf("failed to mount filesystem: %w", err)
	}

	log.WithField("mount_path", mountPath).Info("✅ Mount operation completed successfully")
	return nil
}

// exportQCOW2 exports a QCOW2 file via qemu-nbd
func (mm *MountManager) exportQCOW2(qcow2Path, nbdDevice string) error {
	log.WithFields(log.Fields{
		"qcow2_path":  qcow2Path,
		"nbd_device":  nbdDevice,
	}).Debug("📤 Exporting QCOW2 via qemu-nbd")

	// Command: sudo qemu-nbd --connect=/dev/nbdX --format=qcow2 --read-only /path/to/backup.qcow2
	cmd := exec.Command("sudo", "qemu-nbd",
		"--connect="+nbdDevice,
		"--format=qcow2",
		"--read-only",
		qcow2Path,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-nbd failed: %w, output: %s", err, string(output))
	}

	log.WithField("nbd_device", nbdDevice).Info("✅ QCOW2 exported via qemu-nbd")
	return nil
}

// waitForNBDDevice waits for NBD device to become available
func (mm *MountManager) waitForNBDDevice(nbdDevice string) error {
	log.WithField("nbd_device", nbdDevice).Debug("⏳ Waiting for NBD device to be ready")

	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for NBD device: %s", nbdDevice)
		case <-ticker.C:
			if _, err := os.Stat(nbdDevice); err == nil {
				log.WithField("nbd_device", nbdDevice).Debug("✅ NBD device ready")
				return nil
			}
		}
	}
}

// detectPartition detects the partition to mount (usually nbdXp1)
func (mm *MountManager) detectPartition(nbdDevice string) string {
	// Most common: first partition (nbdXp1)
	partition := nbdDevice + "p1"
	
	// Check if partition exists
	if _, err := os.Stat(partition); err == nil {
		log.WithField("partition", partition).Debug("✅ Detected partition")
		return partition
	}

	// Fallback: use the device itself (no partition table)
	log.WithField("device", nbdDevice).Debug("ℹ️  No partition detected, using device directly")
	return nbdDevice
}

// detectFilesystem detects the filesystem type
func (mm *MountManager) detectFilesystem(device string) (string, error) {
	cmd := exec.Command("blkid", "-o", "value", "-s", "TYPE", device)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to detect filesystem: %w", err)
	}

	fsType := strings.TrimSpace(string(output))
	log.WithField("filesystem_type", fsType).Debug("📋 Detected filesystem type")
	return fsType, nil
}

// mountFilesystem mounts the filesystem to the mount point (read-only)
func (mm *MountManager) mountFilesystem(device, mountPath string) error {
	log.WithFields(log.Fields{
		"device":     device,
		"mount_path": mountPath,
	}).Debug("🔨 Mounting filesystem (read-only)")

	// Command: sudo mount -o ro /dev/nbdXp1 /mnt/sendense/restore/uuid
	cmd := exec.Command("sudo", "mount", "-o", "ro", device, mountPath)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount failed: %w, output: %s", err, string(output))
	}

	log.WithField("mount_path", mountPath).Info("✅ Filesystem mounted")
	return nil
}

// performUnmount executes the unmount operation (umount + qemu-nbd disconnect)
func (mm *MountManager) performUnmount(ctx context.Context, mountPath, nbdDevice string) error {
	log.WithFields(log.Fields{
		"mount_path": mountPath,
		"nbd_device": nbdDevice,
	}).Info("🔧 Executing unmount operation")

	// Step 1: Unmount filesystem
	if err := mm.unmountFilesystem(mountPath); err != nil {
		log.WithError(err).Warn("Failed to unmount filesystem - continuing with cleanup")
	}

	// Step 2: Remove mount directory
	if err := os.RemoveAll(mountPath); err != nil {
		log.WithError(err).Warn("Failed to remove mount directory")
	}

	// Step 3: Disconnect qemu-nbd
	if err := mm.disconnectNBD(nbdDevice); err != nil {
		log.WithError(err).Warn("Failed to disconnect qemu-nbd")
	}

	log.Info("✅ Unmount operation completed")
	return nil
}

// unmountFilesystem unmounts the filesystem
func (mm *MountManager) unmountFilesystem(mountPath string) error {
	log.WithField("mount_path", mountPath).Debug("🔓 Unmounting filesystem")

	cmd := exec.Command("sudo", "umount", mountPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("umount failed: %w, output: %s", err, string(output))
	}

	log.WithField("mount_path", mountPath).Info("✅ Filesystem unmounted")
	return nil
}

// disconnectNBD disconnects qemu-nbd from NBD device
func (mm *MountManager) disconnectNBD(nbdDevice string) error {
	log.WithField("nbd_device", nbdDevice).Debug("🔌 Disconnecting qemu-nbd")

	// Command: sudo qemu-nbd --disconnect /dev/nbdX
	cmd := exec.Command("sudo", "qemu-nbd", "--disconnect", nbdDevice)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-nbd disconnect failed: %w, output: %s", err, string(output))
	}

	log.WithField("nbd_device", nbdDevice).Info("✅ qemu-nbd disconnected")
	return nil
}

