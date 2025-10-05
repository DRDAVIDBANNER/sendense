// Package database provides database operations using repository pattern
// Task 4: File-Level Restore - Restore Mount Repository
// PROJECT_RULES compliance: ALL database operations via repository pattern
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// RestoreMount represents an active QCOW2 backup mount
type RestoreMount struct {
	ID             string    `db:"id" json:"id"`
	BackupID       string    `db:"backup_id" json:"backup_id"`
	MountPath      string    `db:"mount_path" json:"mount_path"`
	NBDDevice      string    `db:"nbd_device" json:"nbd_device"`
	FilesystemType string    `db:"filesystem_type" json:"filesystem_type"`
	MountMode      string    `db:"mount_mode" json:"mount_mode"`
	Status         string    `db:"status" json:"status"` // mounting, mounted, unmounting, failed
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	LastAccessedAt time.Time `db:"last_accessed_at" json:"last_accessed_at"`
	ExpiresAt      *time.Time `db:"expires_at" json:"expires_at,omitempty"`
}

// RestoreMountRepository handles database operations for restore mounts
// Follows repository pattern as required by PROJECT_RULES
type RestoreMountRepository struct {
	db Connection
}

// NewRestoreMountRepository creates a new restore mount repository instance
func NewRestoreMountRepository(db Connection) *RestoreMountRepository {
	return &RestoreMountRepository{db: db}
}

// Create creates a new restore mount record
func (r *RestoreMountRepository) Create(ctx context.Context, mount *RestoreMount) error {
	log.WithFields(log.Fields{
		"mount_id":    mount.ID,
		"backup_id":   mount.BackupID,
		"mount_path":  mount.MountPath,
		"nbd_device":  mount.NBDDevice,
	}).Debug("Creating restore mount record")

	result := r.db.GetGormDB().WithContext(ctx).Create(mount)
	if result.Error != nil {
		return fmt.Errorf("failed to create restore mount: %w", result.Error)
	}

	log.WithField("mount_id", mount.ID).Info("✅ Restore mount record created")
	return nil
}

// GetByID retrieves a restore mount by ID
func (r *RestoreMountRepository) GetByID(ctx context.Context, mountID string) (*RestoreMount, error) {
	log.WithField("mount_id", mountID).Debug("Fetching restore mount by ID")

	query := `
		SELECT id, backup_id, mount_path, nbd_device, filesystem_type,
		       mount_mode, status, created_at, last_accessed_at, expires_at
		FROM restore_mounts
		WHERE id = ?
	`

	var mount RestoreMount
	err := r.db.GetGormDB().Raw(query, mountID).Scan(&mount).Error
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("restore mount not found: %s", mountID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get restore mount: %w", err)
	}

	return &mount, nil
}

// GetByBackupID retrieves active restore mounts for a backup
func (r *RestoreMountRepository) GetByBackupID(ctx context.Context, backupID string) ([]*RestoreMount, error) {
	log.WithField("backup_id", backupID).Debug("Fetching restore mounts by backup ID")

	query := `
		SELECT id, backup_id, mount_path, nbd_device, filesystem_type,
		       mount_mode, status, created_at, last_accessed_at, expires_at
		FROM restore_mounts
		WHERE backup_id = ? AND status IN ('mounting', 'mounted')
		ORDER BY created_at DESC
	`

	var mounts []*RestoreMount
	err := r.db.GetGormDB().Raw(query, backupID).Scan(&mounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get restore mounts by backup ID: %w", err)
	}

	return mounts, nil
}

// GetByNBDDevice retrieves a restore mount by NBD device path
func (r *RestoreMountRepository) GetByNBDDevice(ctx context.Context, nbdDevice string) (*RestoreMount, error) {
	log.WithField("nbd_device", nbdDevice).Debug("Fetching restore mount by NBD device")

	query := `
		SELECT id, backup_id, mount_path, nbd_device, filesystem_type,
		       mount_mode, status, created_at, last_accessed_at, expires_at
		FROM restore_mounts
		WHERE nbd_device = ? AND status IN ('mounting', 'mounted')
	`

	var mount RestoreMount
	err := r.db.GetGormDB().Raw(query, nbdDevice).Scan(&mount).Error
	if err == sql.ErrNoRows {
		return nil, nil // No mount found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get restore mount by NBD device: %w", err)
	}

	return &mount, nil
}

// ListActive lists all active restore mounts
func (r *RestoreMountRepository) ListActive(ctx context.Context) ([]*RestoreMount, error) {
	log.Debug("Listing active restore mounts")

	query := `
		SELECT id, backup_id, mount_path, nbd_device, filesystem_type,
		       mount_mode, status, created_at, last_accessed_at, expires_at
		FROM restore_mounts
		WHERE status IN ('mounting', 'mounted')
		ORDER BY created_at DESC
	`

	var mounts []*RestoreMount
	err := r.db.GetGormDB().Raw(query).Scan(&mounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list active restore mounts: %w", err)
	}

	return mounts, nil
}

// ListExpired lists all mounts that have exceeded their idle timeout
func (r *RestoreMountRepository) ListExpired(ctx context.Context) ([]*RestoreMount, error) {
	log.Debug("Listing expired restore mounts")

	query := `
		SELECT id, backup_id, mount_path, nbd_device, filesystem_type,
		       mount_mode, status, created_at, last_accessed_at, expires_at
		FROM restore_mounts
		WHERE status IN ('mounting', 'mounted')
		  AND expires_at IS NOT NULL
		  AND expires_at < NOW()
		ORDER BY expires_at ASC
	`

	var mounts []*RestoreMount
	err := r.db.GetGormDB().Raw(query).Scan(&mounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list expired restore mounts: %w", err)
	}

	return mounts, nil
}

// UpdateStatus updates the mount status
func (r *RestoreMountRepository) UpdateStatus(ctx context.Context, mountID string, status string) error {
	log.WithFields(log.Fields{
		"mount_id": mountID,
		"status":   status,
	}).Debug("Updating restore mount status")

	query := `
		UPDATE restore_mounts
		SET status = ?, last_accessed_at = NOW()
		WHERE id = ?
	`

	result := r.db.GetGormDB().Exec(query, status, mountID)
	if result.Error != nil {
		return fmt.Errorf("failed to update restore mount status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("restore mount not found: %s", mountID)
	}

	return nil
}

// UpdateLastAccessed updates the last accessed timestamp (for idle detection)
func (r *RestoreMountRepository) UpdateLastAccessed(ctx context.Context, mountID string) error {
	log.WithField("mount_id", mountID).Debug("Updating restore mount last accessed time")

	query := `
		UPDATE restore_mounts
		SET last_accessed_at = NOW()
		WHERE id = ?
	`

	result := r.db.GetGormDB().Exec(query, mountID)
	if result.Error != nil {
		return fmt.Errorf("failed to update last accessed time: %w", result.Error)
	}

	return nil
}

// Delete removes a restore mount record
func (r *RestoreMountRepository) Delete(ctx context.Context, mountID string) error {
	log.WithField("mount_id", mountID).Debug("Deleting restore mount record")

	query := `DELETE FROM restore_mounts WHERE id = ?`

	result := r.db.GetGormDB().Exec(query, mountID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete restore mount: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("restore mount not found: %s", mountID)
	}

	log.WithField("mount_id", mountID).Info("✅ Restore mount record deleted")
	return nil
}

// GetAllocatedNBDDevices returns a list of currently allocated NBD devices
func (r *RestoreMountRepository) GetAllocatedNBDDevices(ctx context.Context) ([]string, error) {
	log.Debug("Fetching allocated NBD devices")

	query := `
		SELECT nbd_device
		FROM restore_mounts
		WHERE status IN ('mounting', 'mounted')
		ORDER BY nbd_device
	`

	var devices []string
	err := r.db.GetGormDB().Raw(query).Scan(&devices).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get allocated NBD devices: %w", err)
	}

	return devices, nil
}

// CountActiveMounts returns the count of active mounts
func (r *RestoreMountRepository) CountActiveMounts(ctx context.Context) (int, error) {
	log.Debug("Counting active restore mounts")

	query := `
		SELECT COUNT(*)
		FROM restore_mounts
		WHERE status IN ('mounting', 'mounted')
	`

	var count int
	err := r.db.GetGormDB().Raw(query).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count active restore mounts: %w", err)
	}

	return count, nil
}

// UpdateFields updates multiple fields of a restore mount
func (r *RestoreMountRepository) UpdateFields(ctx context.Context, mountID string, fields map[string]interface{}) error {
	log.WithFields(log.Fields{
		"mount_id": mountID,
		"fields":   fields,
	}).Debug("Updating restore mount fields")

	result := r.db.GetGormDB().WithContext(ctx).Model(&RestoreMount{}).
		Where("id = ?", mountID).
		Updates(fields)

	if result.Error != nil {
		return fmt.Errorf("failed to update restore mount fields: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("restore mount not found: %s", mountID)
	}

	return nil
}
