package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/nbd"
)

// NBDExportRepository implements the ExportRepository interface
type NBDExportRepository struct {
	db *sqlx.DB
}

// NewNBDExportRepository creates a new NBD export repository
func NewNBDExportRepository(db *sqlx.DB) nbd.ExportRepository {
	return &NBDExportRepository{db: db}
}

// CreateExport creates a new NBD export record
func (r *NBDExportRepository) CreateExport(ctx context.Context, export *nbd.ExportInfo) error {
	metadataJSON, err := json.Marshal(export.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO nbd_exports (
			id, volume_id, vm_disk_id, export_name, device_path, port, status, 
			created_at, updated_at, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	log.WithFields(log.Fields{
		"export_id":   export.ID,
		"volume_id":   export.VolumeID,
		"export_name": export.ExportName,
		"device_path": export.DevicePath,
	}).Debug("Creating NBD export database record")

	_, err = r.db.ExecContext(ctx, query,
		export.ID,
		export.VolumeID,
		export.VMDiskID, // Include vm_disk_id correlation
		export.ExportName,
		export.DevicePath,
		export.Port,
		string(export.Status),
		export.CreatedAt,
		export.UpdatedAt,
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to create NBD export record: %w", err)
	}

	log.WithField("export_id", export.ID).Info("✅ Created NBD export database record")
	return nil
}

// UpdateExport updates an existing NBD export record
func (r *NBDExportRepository) UpdateExport(ctx context.Context, export *nbd.ExportInfo) error {
	metadataJSON, err := json.Marshal(export.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE nbd_exports 
		SET volume_id = ?, export_name = ?, device_path = ?, port = ?, 
		    status = ?, updated_at = ?, metadata = ?
		WHERE id = ?
	`

	log.WithFields(log.Fields{
		"export_id": export.ID,
		"status":    export.Status,
	}).Debug("Updating NBD export database record")

	result, err := r.db.ExecContext(ctx, query,
		export.VolumeID,
		export.ExportName,
		export.DevicePath,
		export.Port,
		string(export.Status),
		export.UpdatedAt,
		string(metadataJSON),
		export.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update NBD export record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("NBD export record not found: %s", export.ID)
	}

	log.WithField("export_id", export.ID).Debug("✅ Updated NBD export database record")
	return nil
}

// DeleteExport deletes an NBD export record
func (r *NBDExportRepository) DeleteExport(ctx context.Context, exportID string) error {
	query := `DELETE FROM nbd_exports WHERE id = ?`

	log.WithField("export_id", exportID).Debug("Deleting NBD export database record")

	result, err := r.db.ExecContext(ctx, query, exportID)
	if err != nil {
		return fmt.Errorf("failed to delete NBD export record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.WithField("export_id", exportID).Warn("NBD export record not found for deletion")
		return nil // Idempotent operation
	}

	log.WithField("export_id", exportID).Info("✅ Deleted NBD export database record")
	return nil
}

// GetExport retrieves an NBD export by ID
func (r *NBDExportRepository) GetExport(ctx context.Context, exportID string) (*nbd.ExportInfo, error) {
	query := `
		SELECT id, volume_id, export_name, device_path, port, status, 
		       created_at, updated_at, metadata
		FROM nbd_exports 
		WHERE id = ?
	`

	var export nbd.ExportInfo
	var metadataJSON string

	err := r.db.QueryRowContext(ctx, query, exportID).Scan(
		&export.ID,
		&export.VolumeID,
		&export.ExportName,
		&export.DevicePath,
		&export.Port,
		&export.Status,
		&export.CreatedAt,
		&export.UpdatedAt,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("NBD export not found: %s", exportID)
		}
		return nil, fmt.Errorf("failed to get NBD export: %w", err)
	}

	// Unmarshal metadata
	if err := json.Unmarshal([]byte(metadataJSON), &export.Metadata); err != nil {
		log.WithError(err).Warn("Failed to unmarshal export metadata")
		export.Metadata = make(map[string]string)
	}

	return &export, nil
}

// GetExportByVolumeID retrieves an NBD export by volume ID
func (r *NBDExportRepository) GetExportByVolumeID(ctx context.Context, volumeID string) (*nbd.ExportInfo, error) {
	query := `
		SELECT id, volume_id, export_name, device_path, port, status, 
		       created_at, updated_at, metadata
		FROM nbd_exports 
		WHERE volume_id = ?
	`

	var export nbd.ExportInfo
	var metadataJSON string

	err := r.db.QueryRowContext(ctx, query, volumeID).Scan(
		&export.ID,
		&export.VolumeID,
		&export.ExportName,
		&export.DevicePath,
		&export.Port,
		&export.Status,
		&export.CreatedAt,
		&export.UpdatedAt,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("NBD export not found for volume: %s", volumeID)
		}
		return nil, fmt.Errorf("failed to get NBD export by volume ID: %w", err)
	}

	// Unmarshal metadata
	if err := json.Unmarshal([]byte(metadataJSON), &export.Metadata); err != nil {
		log.WithError(err).Warn("Failed to unmarshal export metadata")
		export.Metadata = make(map[string]string)
	}

	return &export, nil
}

// ListExports lists NBD exports with optional filtering
func (r *NBDExportRepository) ListExports(ctx context.Context, filter nbd.ExportFilter) ([]*nbd.ExportInfo, error) {
	query := `
		SELECT id, volume_id, export_name, device_path, port, status, 
		       created_at, updated_at, metadata
		FROM nbd_exports 
		WHERE 1=1
	`
	args := []interface{}{}

	// Apply filters
	if filter.VolumeID != nil {
		query += " AND volume_id = ?"
		args = append(args, *filter.VolumeID)
	}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, string(*filter.Status))
	}

	if filter.VMName != nil {
		query += " AND JSON_EXTRACT(metadata, '$.vm_name') = ?"
		args = append(args, *filter.VMName)
	}

	// Add ordering and limit
	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	log.WithFields(log.Fields{
		"filter": filter,
		"query":  query,
	}).Debug("Listing NBD exports")

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list NBD exports: %w", err)
	}
	defer rows.Close()

	var exports []*nbd.ExportInfo

	for rows.Next() {
		var export nbd.ExportInfo
		var metadataJSON string

		err := rows.Scan(
			&export.ID,
			&export.VolumeID,
			&export.ExportName,
			&export.DevicePath,
			&export.Port,
			&export.Status,
			&export.CreatedAt,
			&export.UpdatedAt,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan NBD export row: %w", err)
		}

		// Unmarshal metadata
		if err := json.Unmarshal([]byte(metadataJSON), &export.Metadata); err != nil {
			log.WithError(err).Warn("Failed to unmarshal export metadata")
			export.Metadata = make(map[string]string)
		}

		exports = append(exports, &export)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating NBD export rows: %w", err)
	}

	log.WithField("count", len(exports)).Debug("✅ Listed NBD exports")
	return exports, nil
}
