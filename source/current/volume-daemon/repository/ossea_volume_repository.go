package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// OSSEAVolumeRepository handles OSSEA volume database operations for the Volume Daemon
type OSSEAVolumeRepository struct {
	db *sqlx.DB
}

// OSSEAVolume represents the ossea_volumes table structure
type OSSEAVolume struct {
	ID            int       `db:"id" json:"id"`
	VolumeID      string    `db:"volume_id" json:"volume_id"`
	VolumeName    string    `db:"volume_name" json:"volume_name"`
	SizeGB        int       `db:"size_gb" json:"size_gb"`
	OSSEAConfigID *int      `db:"ossea_config_id" json:"ossea_config_id,omitempty"`
	VolumeType    *string   `db:"volume_type" json:"volume_type,omitempty"`
	DevicePath    *string   `db:"device_path" json:"device_path,omitempty"`
	MountPoint    *string   `db:"mount_point" json:"mount_point,omitempty"`
	Status        string    `db:"status" json:"status"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// NewOSSEAVolumeRepository creates a new OSSEA volume repository for Volume Daemon
func NewOSSEAVolumeRepository(db *sqlx.DB) *OSSEAVolumeRepository {
	return &OSSEAVolumeRepository{db: db}
}

// UpdateVolumeAttachment updates ossea_volumes table when a volume is attached
func (r *OSSEAVolumeRepository) UpdateVolumeAttachment(ctx context.Context, volumeID, devicePath string) error {
	query := `
		UPDATE ossea_volumes 
		SET device_path = ?, status = 'attached', updated_at = NOW()
		WHERE volume_id = ?
	`

	result, err := r.db.ExecContext(ctx, query, devicePath, volumeID)
	if err != nil {
		log.WithFields(log.Fields{
			"volume_id":   volumeID,
			"device_path": devicePath,
			"error":       err,
		}).Error("Failed to update ossea_volumes on volume attachment")
		return fmt.Errorf("failed to update ossea_volumes attachment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.WithFields(log.Fields{
			"volume_id":   volumeID,
			"device_path": devicePath,
		}).Warn("No ossea_volumes record found to update - volume may not be tracked")
		return nil // Non-fatal - volume might not be part of a migration
	}

	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"device_path": devicePath,
		"status":      "attached",
	}).Info("✅ Updated ossea_volumes on volume attachment")

	return nil
}

// UpdateVolumeDetachment updates ossea_volumes table when a volume is detached
func (r *OSSEAVolumeRepository) UpdateVolumeDetachment(ctx context.Context, volumeID string) error {
	query := `
		UPDATE ossea_volumes 
		SET device_path = NULL, status = 'detached', updated_at = NOW()
		WHERE volume_id = ?
	`

	result, err := r.db.ExecContext(ctx, query, volumeID)
	if err != nil {
		log.WithFields(log.Fields{
			"volume_id": volumeID,
			"error":     err,
		}).Error("Failed to update ossea_volumes on volume detachment")
		return fmt.Errorf("failed to update ossea_volumes detachment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.WithFields(log.Fields{
			"volume_id": volumeID,
		}).Warn("No ossea_volumes record found to update - volume may not be tracked")
		return nil // Non-fatal - volume might not be part of a migration
	}

	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"status":    "detached",
	}).Info("✅ Updated ossea_volumes on volume detachment")

	return nil
}

// GetVolumeByID retrieves an ossea_volumes record by volume_id
func (r *OSSEAVolumeRepository) GetVolumeByID(ctx context.Context, volumeID string) (*OSSEAVolume, error) {
	query := `
		SELECT id, volume_id, volume_name, size_gb, ossea_config_id, volume_type, 
		       device_path, mount_point, status, created_at, updated_at
		FROM ossea_volumes 
		WHERE volume_id = ?
	`

	var volume OSSEAVolume
	err := r.db.GetContext(ctx, &volume, query, volumeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Volume not found (not an error)
		}
		return nil, fmt.Errorf("failed to get ossea_volume: %w", err)
	}

	return &volume, nil
}

// CheckVolumeExists verifies if a volume exists in the ossea_volumes table
func (r *OSSEAVolumeRepository) CheckVolumeExists(ctx context.Context, volumeID string) (bool, error) {
	query := `SELECT COUNT(*) FROM ossea_volumes WHERE volume_id = ?`
	
	var count int
	err := r.db.GetContext(ctx, &count, query, volumeID)
	if err != nil {
		return false, fmt.Errorf("failed to check ossea_volume existence: %w", err)
	}

	return count > 0, nil
}
