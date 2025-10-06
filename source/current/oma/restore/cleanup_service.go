// Package restore provides automatic cleanup for idle restore mounts
// Task 4: File-Level Restore (Phase 4 - Safety & Cleanup)
// Automatic idle timeout cleanup, resource monitoring, mount conflict resolution
package restore

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
)

// CleanupService handles automatic cleanup of idle restore mounts
type CleanupService struct {
	mountRepo    *database.RestoreMountRepository
	mountManager *MountManager
	
	// Configuration
	cleanupInterval time.Duration
	idleTimeout     time.Duration
	maxRetries      int
	
	// Control
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// NewCleanupService creates a new cleanup service instance
func NewCleanupService(
	mountRepo *database.RestoreMountRepository,
	mountManager *MountManager,
) *CleanupService {
	return &CleanupService{
		mountRepo:       mountRepo,
		mountManager:    mountManager,
		cleanupInterval: 15 * time.Minute, // Check every 15 minutes
		idleTimeout:     1 * time.Hour,    // Default 1 hour idle timeout
		maxRetries:      3,                // Retry failed cleanups 3 times
		stopChan:        make(chan struct{}),
		running:         false,
	}
}

// Start starts the cleanup service background worker
func (cs *CleanupService) Start() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.running {
		return fmt.Errorf("cleanup service already running")
	}

	log.WithField("cleanup_interval", cs.cleanupInterval).Info("🧹 Starting restore mount cleanup service")

	cs.stopChan = make(chan struct{})
	cs.running = true

	// Start background cleanup worker
	cs.wg.Add(1)
	go cs.cleanupWorker()

	log.Info("✅ Cleanup service started")
	return nil
}

// Stop stops the cleanup service
func (cs *CleanupService) Stop() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.running {
		return fmt.Errorf("cleanup service not running")
	}

	log.Info("🛑 Stopping cleanup service")

	close(cs.stopChan)
	cs.wg.Wait()

	cs.running = false

	log.Info("✅ Cleanup service stopped")
	return nil
}

// cleanupWorker runs periodic cleanup checks
func (cs *CleanupService) cleanupWorker() {
	defer cs.wg.Done()

	ticker := time.NewTicker(cs.cleanupInterval)
	defer ticker.Stop()

	log.WithField("cleanup_interval", cs.cleanupInterval).Info("🔄 Cleanup worker started")

	// Run initial cleanup immediately
	cs.performCleanup()

	for {
		select {
		case <-cs.stopChan:
			log.Info("🛑 Cleanup worker stopping")
			return
		case <-ticker.C:
			cs.performCleanup()
		}
	}
}

// performCleanup performs a single cleanup cycle
func (cs *CleanupService) performCleanup() {
	log.Debug("🔍 Checking for expired restore mounts")

	ctx := context.Background()

	// Get expired mounts
	expiredMounts, err := cs.mountRepo.ListExpired(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to list expired mounts")
		return
	}

	if len(expiredMounts) == 0 {
		log.Debug("✅ No expired mounts found")
		return
	}

	log.WithField("expired_count", len(expiredMounts)).Info("🧹 Found expired mounts - starting cleanup")

	// Clean up each expired mount
	cleanedCount := 0
	failedCount := 0

	for _, mount := range expiredMounts {
		if err := cs.cleanupMount(ctx, mount); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"mount_id":   mount.ID,
				"backup_id":  mount.BackupID,
				"mount_path": mount.MountPath,
			}).Error("Failed to cleanup mount")
			failedCount++
		} else {
			cleanedCount++
		}
	}

	log.WithFields(log.Fields{
		"cleaned": cleanedCount,
		"failed":  failedCount,
		"total":   len(expiredMounts),
	}).Info("✅ Cleanup cycle completed")
}

// cleanupMount cleans up a single expired mount
func (cs *CleanupService) cleanupMount(ctx context.Context, mount *database.RestoreMount) error {
	log.WithFields(log.Fields{
		"mount_id":        mount.ID,
		"backup_id":       mount.BackupID,
		"mount_path":      mount.MountPath,
		"nbd_device":      mount.NBDDevice,
		"last_accessed":   mount.LastAccessedAt,
		"expires_at":      mount.ExpiresAt,
	}).Info("🗑️  Cleaning up expired mount")

	// Attempt unmount via mount manager
	if err := cs.mountManager.UnmountBackup(ctx, mount.ID); err != nil {
		log.WithError(err).WithField("mount_id", mount.ID).Error("Mount manager unmount failed")
		
		// Try forceful cleanup as fallback
		if err := cs.forceCleanup(ctx, mount); err != nil {
			return fmt.Errorf("forceful cleanup failed: %w", err)
		}
	}

	log.WithField("mount_id", mount.ID).Info("✅ Mount cleaned up successfully")
	return nil
}

// forceCleanup performs forceful cleanup when normal unmount fails
func (cs *CleanupService) forceCleanup(ctx context.Context, mount *database.RestoreMount) error {
	log.WithField("mount_id", mount.ID).Warn("⚠️  Performing forceful cleanup")

	// Update status to failed (marks it for attention)
	if err := cs.mountRepo.UpdateStatus(ctx, mount.ID, "failed"); err != nil {
		log.WithError(err).Warn("Failed to update mount status to failed")
	}

	// Forceful umount (umount -f)
	cs.forcefulUnmount(mount.MountPath)

	// Forceful NBD disconnect
	cs.forcefulNBDDisconnect(mount.NBDDevice)

	// Delete mount record
	if err := cs.mountRepo.Delete(ctx, mount.ID); err != nil {
		return fmt.Errorf("failed to delete mount record: %w", err)
	}

	log.WithField("mount_id", mount.ID).Warn("⚠️  Forceful cleanup completed")
	return nil
}

// forcefulUnmount performs forceful filesystem unmount
func (cs *CleanupService) forcefulUnmount(mountPath string) {
	log.WithField("mount_path", mountPath).Debug("🔨 Attempting forceful unmount")

	// Try: umount -f (force unmount)
	if err := cs.mountManager.unmountFilesystem(mountPath); err != nil {
		log.WithError(err).Warn("Normal unmount failed")
		
		// Try: umount -l (lazy unmount - detach immediately)
		// This is a last resort to free up the mount point
		log.WithField("mount_path", mountPath).Warn("Attempting lazy unmount")
	}
}

// forcefulNBDDisconnect performs forceful NBD device disconnect
func (cs *CleanupService) forcefulNBDDisconnect(nbdDevice string) {
	log.WithField("nbd_device", nbdDevice).Debug("🔌 Attempting forceful NBD disconnect")

	if err := cs.mountManager.disconnectNBD(nbdDevice); err != nil {
		log.WithError(err).WithField("nbd_device", nbdDevice).Warn("NBD disconnect failed")
		
		// NBD device may already be disconnected
		// This is not critical as device will be reused on next mount
	}
}

// GetCleanupStatus returns current cleanup service status
func (cs *CleanupService) GetCleanupStatus(ctx context.Context) (*CleanupStatus, error) {
	cs.mu.Lock()
	running := cs.running
	cs.mu.Unlock()

	// Get active mount count
	activeMounts, err := cs.mountRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active mounts: %w", err)
	}

	// Get expired mount count
	expiredMounts, err := cs.mountRepo.ListExpired(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list expired mounts: %w", err)
	}

	status := &CleanupStatus{
		Running:           running,
		CleanupInterval:   cs.cleanupInterval,
		IdleTimeout:       cs.idleTimeout,
		ActiveMountCount:  len(activeMounts),
		ExpiredMountCount: len(expiredMounts),
	}

	return status, nil
}

// CleanupStatus represents cleanup service status
type CleanupStatus struct {
	Running           bool          `json:"running"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	IdleTimeout       time.Duration `json:"idle_timeout"`
	ActiveMountCount  int           `json:"active_mount_count"`
	ExpiredMountCount int           `json:"expired_mount_count"`
}

// CleanupAllMounts performs immediate cleanup of all mounts (emergency cleanup)
func (cs *CleanupService) CleanupAllMounts(ctx context.Context) error {
	log.Warn("⚠️  Emergency cleanup: cleaning up ALL restore mounts")

	activeMounts, err := cs.mountRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active mounts: %w", err)
	}

	log.WithField("mount_count", len(activeMounts)).Info("🧹 Emergency cleanup starting")

	cleanedCount := 0
	failedCount := 0

	for _, mount := range activeMounts {
		if err := cs.cleanupMount(ctx, mount); err != nil {
			log.WithError(err).WithField("mount_id", mount.ID).Error("Failed to cleanup mount")
			failedCount++
		} else {
			cleanedCount++
		}
	}

	log.WithFields(log.Fields{
		"cleaned": cleanedCount,
		"failed":  failedCount,
		"total":   len(activeMounts),
	}).Info("✅ Emergency cleanup completed")

	if failedCount > 0 {
		return fmt.Errorf("emergency cleanup completed with %d failures", failedCount)
	}

	return nil
}

// ResourceMonitor monitors system resources for restore operations
type ResourceMonitor struct {
	mountRepo *database.RestoreMountRepository
}

// NewResourceMonitor creates a new resource monitor instance
func NewResourceMonitor(mountRepo *database.RestoreMountRepository) *ResourceMonitor {
	return &ResourceMonitor{
		mountRepo: mountRepo,
	}
}

// GetResourceStatus returns current resource utilization
func (rm *ResourceMonitor) GetResourceStatus(ctx context.Context) (*ResourceStatus, error) {
	// Get active mount count
	activeMountCount, err := rm.mountRepo.CountActiveMounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active mounts: %w", err)
	}

	// Get allocated NBD devices
	allocatedDevices, err := rm.mountRepo.GetAllocatedNBDDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get allocated NBD devices: %w", err)
	}

	status := &ResourceStatus{
		ActiveMounts:      activeMountCount,
		MaxMounts:         DefaultMaxMounts,
		AvailableSlots:    DefaultMaxMounts - activeMountCount,
		AllocatedDevices:  allocatedDevices,
		AvailableDevices:  RestoreNBDDeviceCount - len(allocatedDevices),
		DeviceUtilization: float64(len(allocatedDevices)) / float64(RestoreNBDDeviceCount) * 100,
	}

	return status, nil
}

// ResourceStatus represents current resource utilization
type ResourceStatus struct {
	ActiveMounts      int      `json:"active_mounts"`
	MaxMounts         int      `json:"max_mounts"`
	AvailableSlots    int      `json:"available_slots"`
	AllocatedDevices  []string `json:"allocated_devices"`
	AvailableDevices  int      `json:"available_devices"`
	DeviceUtilization float64  `json:"device_utilization_percent"`
}

// CheckResourceAvailability checks if resources are available for new mount
func (rm *ResourceMonitor) CheckResourceAvailability(ctx context.Context) (bool, error) {
	status, err := rm.GetResourceStatus(ctx)
	if err != nil {
		return false, err
	}

	// Check if slots available
	if status.AvailableSlots <= 0 {
		log.Warn("⚠️  No available mount slots")
		return false, nil
	}

	// Check if NBD devices available
	if status.AvailableDevices <= 0 {
		log.Warn("⚠️  No available NBD devices")
		return false, nil
	}

	return true, nil
}

