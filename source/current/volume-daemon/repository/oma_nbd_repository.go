package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/nbd"
)

// Helper function to safely convert nullable string pointer to string
func stringPtrToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// OMANBDRepository implements NBD export operations using the existing migratekit_oma database
type OMANBDRepository struct {
	db *sqlx.DB
}

// OMANBDExport represents the existing nbd_exports table structure in migratekit_oma
type OMANBDExport struct {
	ID                uint      `db:"id" json:"id"`
	JobID             *string   `db:"job_id" json:"job_id,omitempty"`
	VMContextID       *string   `db:"vm_context_id" json:"vm_context_id,omitempty"` // VM-Centric Architecture integration
	VolumeID          string    `db:"volume_id" json:"volume_id"`
	VMDiskID          *uint     `db:"vm_disk_id" json:"vm_disk_id,omitempty"`
	DeviceMappingUUID *string   `db:"device_mapping_uuid" json:"device_mapping_uuid,omitempty"`
	ExportName        string    `db:"export_name" json:"export_name"`
	Port              int       `db:"port" json:"port"`
	DevicePath        string    `db:"device_path" json:"device_path"`
	ConfigPath        string    `db:"config_path" json:"config_path"`
	Status            string    `db:"status" json:"status"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

// NewOMANBDRepository creates a repository for the existing migratekit_oma nbd_exports table
func NewOMANBDRepository(db *sqlx.DB) nbd.ExportRepository {
	return &OMANBDRepository{db: db}
}

// CreateExport creates a new NBD export record in the existing migratekit_oma database
func (r *OMANBDRepository) CreateExport(ctx context.Context, export *nbd.ExportInfo) error {
	// Metadata is stored in our export metadata field, not in the OMA table
	// The OMA table doesn't have a metadata column, so we'll store key info in our interface

	// Extract job_id from metadata - allow NULL for standalone exports
	var jobID *string
	if export.Metadata != nil {
		if jid, exists := export.Metadata["job_id"]; exists && jid != "" {
			jobID = &jid
		}
	}

	query := `
		INSERT INTO nbd_exports (
			job_id, vm_context_id, volume_id, export_name, port, device_path, 
			config_path, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	log.WithFields(log.Fields{
		"export_name": export.ExportName,
		"volume_id":   export.VolumeID,
		"device_path": export.DevicePath,
		"job_id":      jobID,
	}).Debug("Creating NBD export in migratekit_oma database")

	// Calculate the actual config file path in conf.d directory
	configFilePath := fmt.Sprintf("/etc/nbd-server/conf.d/%s.conf", export.ExportName)

	log.WithField("volume_id", export.VolumeID).Info("🔍 NBD export creation - checking VM context ID")

	// Get VM context ID from job ID if available, otherwise try volume ID
	var vmContextID *string
	if jobID != nil {
		log.WithField("job_id", *jobID).Info("🔍 NBD export has job_id, trying job lookup first")
		vmCtxID, err := r.getVMContextIDForJob(*jobID)
		if err != nil {
			log.WithError(err).WithField("job_id", *jobID).Warn("Failed to get VM context ID for NBD export via job_id, trying volume_id")
			// Fall back to volume ID lookup
			if vmCtxID, err := r.getVMContextIDFromVolumeID(export.VolumeID); err != nil {
				log.WithError(err).WithField("volume_id", export.VolumeID).Debug("Failed to get VM context ID for NBD export via volume_id, continuing without it")
			} else if vmCtxID != "" {
				vmContextID = &vmCtxID
			}
		} else if vmCtxID != "" {
			vmContextID = &vmCtxID
		}
	} else {
		log.WithField("volume_id", export.VolumeID).Info("🔍 NBD export has no job_id, trying volume_id lookup")
		// No job ID available, try volume ID lookup
		if vmCtxID, err := r.getVMContextIDFromVolumeID(export.VolumeID); err != nil {
			log.WithError(err).WithField("volume_id", export.VolumeID).Warn("❌ Failed to get VM context ID for NBD export via volume_id, continuing without it")
		} else if vmCtxID != "" {
			vmContextID = &vmCtxID
			log.WithFields(log.Fields{
				"volume_id":     export.VolumeID,
				"vm_context_id": vmCtxID,
			}).Info("✅ Successfully set VM context ID for NBD export")
		} else {
			log.WithField("volume_id", export.VolumeID).Warn("⚠️ VM context lookup returned empty string for NBD export")
		}
	}

	_, err := r.db.ExecContext(ctx, query,
		jobID,
		vmContextID, // VM-Centric Architecture integration
		export.VolumeID,
		export.ExportName,
		export.Port,
		export.DevicePath,
		configFilePath, // Actual individual config file path
		string(export.Status),
		export.CreatedAt,
		export.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create NBD export in migratekit_oma: %w", err)
	}

	log.WithField("export_name", export.ExportName).Info("✅ Created NBD export in migratekit_oma database")
	return nil
}

// UpdateExport updates an existing NBD export record
func (r *OMANBDRepository) UpdateExport(ctx context.Context, export *nbd.ExportInfo) error {
	query := `
		UPDATE nbd_exports 
		SET volume_id = ?, export_name = ?, port = ?, device_path = ?, 
		    status = ?, updated_at = ?
		WHERE export_name = ?
	`

	log.WithFields(log.Fields{
		"export_name": export.ExportName,
		"status":      export.Status,
	}).Debug("Updating NBD export in migratekit_oma database")

	result, err := r.db.ExecContext(ctx, query,
		export.VolumeID,
		export.ExportName,
		export.Port,
		export.DevicePath,
		string(export.Status),
		export.UpdatedAt,
		export.ExportName, // WHERE clause
	)

	if err != nil {
		return fmt.Errorf("failed to update NBD export in migratekit_oma: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("NBD export not found: %s", export.ExportName)
	}

	log.WithField("export_name", export.ExportName).Debug("✅ Updated NBD export in migratekit_oma database")
	return nil
}

// DeleteExport deletes an NBD export record by volume ID
func (r *OMANBDRepository) DeleteExport(ctx context.Context, exportID string) error {
	// The exportID in our interface is actually the volume_id in most cases
	// First try to find by export name (if it looks like an export name)
	// Otherwise use volume_id

	var query string
	var param string

	if len(exportID) > 20 && (exportID[:10] == "migration-" || exportID[:4] == "nbd-") {
		// Looks like an export name
		query = `DELETE FROM nbd_exports WHERE export_name = ?`
		param = exportID
	} else {
		// Treat as volume_id
		query = `DELETE FROM nbd_exports WHERE volume_id = ?`
		param = exportID
	}

	log.WithField("identifier", exportID).Debug("Deleting NBD export from migratekit_oma database")

	result, err := r.db.ExecContext(ctx, query, param)
	if err != nil {
		return fmt.Errorf("failed to delete NBD export from migratekit_oma: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.WithField("identifier", exportID).Warn("NBD export not found for deletion - may already be deleted")
		return nil // Idempotent operation
	}

	log.WithField("identifier", exportID).Info("✅ Deleted NBD export from migratekit_oma database")
	return nil
}

// GetExport retrieves an NBD export by export name (using exportID parameter)
func (r *OMANBDRepository) GetExport(ctx context.Context, exportID string) (*nbd.ExportInfo, error) {
	query := `
		SELECT job_id, volume_id, export_name, port, device_path, 
		       config_path, status, created_at, updated_at
		FROM nbd_exports 
		WHERE export_name = ?
	`

	var omaExport OMANBDExport
	err := r.db.QueryRowContext(ctx, query, exportID).Scan(
		&omaExport.JobID,
		&omaExport.VolumeID,
		&omaExport.ExportName,
		&omaExport.Port,
		&omaExport.DevicePath,
		&omaExport.ConfigPath,
		&omaExport.Status,
		&omaExport.CreatedAt,
		&omaExport.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("NBD export not found: %s", exportID)
		}
		return nil, fmt.Errorf("failed to get NBD export from migratekit_oma: %w", err)
	}

	// Convert to our export info format
	exportInfo := &nbd.ExportInfo{
		ID:         fmt.Sprintf("%d", omaExport.ID),
		VolumeID:   omaExport.VolumeID,
		ExportName: omaExport.ExportName,
		DevicePath: omaExport.DevicePath,
		Port:       omaExport.Port,
		Status:     nbd.ExportStatus(omaExport.Status),
		CreatedAt:  omaExport.CreatedAt,
		UpdatedAt:  omaExport.UpdatedAt,
		Metadata: map[string]string{
			"job_id":      stringPtrToString(omaExport.JobID),
			"config_path": omaExport.ConfigPath,
		},
	}

	return exportInfo, nil
}

// GetExportByVolumeID retrieves an NBD export by volume ID
func (r *OMANBDRepository) GetExportByVolumeID(ctx context.Context, volumeID string) (*nbd.ExportInfo, error) {
	query := `
		SELECT job_id, volume_id, export_name, port, device_path, 
		       config_path, status, created_at, updated_at
		FROM nbd_exports 
		WHERE volume_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	var omaExport OMANBDExport
	err := r.db.QueryRowContext(ctx, query, volumeID).Scan(
		&omaExport.JobID,
		&omaExport.VolumeID,
		&omaExport.ExportName,
		&omaExport.Port,
		&omaExport.DevicePath,
		&omaExport.ConfigPath,
		&omaExport.Status,
		&omaExport.CreatedAt,
		&omaExport.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("NBD export not found for volume: %s", volumeID)
		}
		return nil, fmt.Errorf("failed to get NBD export by volume ID from migratekit_oma: %w", err)
	}

	// Convert to our export info format
	exportInfo := &nbd.ExportInfo{
		ID:         fmt.Sprintf("%d", omaExport.ID),
		VolumeID:   omaExport.VolumeID,
		ExportName: omaExport.ExportName,
		DevicePath: omaExport.DevicePath,
		Port:       omaExport.Port,
		Status:     nbd.ExportStatus(omaExport.Status),
		CreatedAt:  omaExport.CreatedAt,
		UpdatedAt:  omaExport.UpdatedAt,
		Metadata: map[string]string{
			"job_id":      stringPtrToString(omaExport.JobID),
			"config_path": omaExport.ConfigPath,
		},
	}

	return exportInfo, nil
}

// ListExports lists NBD exports with optional filtering
func (r *OMANBDRepository) ListExports(ctx context.Context, filter nbd.ExportFilter) ([]*nbd.ExportInfo, error) {
	query := `
		SELECT job_id, volume_id, export_name, port, device_path, 
		       config_path, status, created_at, updated_at
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
		// For VM name, we'll search in the export_name pattern
		query += " AND export_name LIKE ?"
		args = append(args, "%"+*filter.VMName+"%")
	}

	// Add ordering and limit
	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	log.WithFields(log.Fields{
		"filter": filter,
	}).Debug("Listing NBD exports from migratekit_oma database")

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list NBD exports from migratekit_oma: %w", err)
	}
	defer rows.Close()

	var exports []*nbd.ExportInfo

	for rows.Next() {
		var omaExport OMANBDExport
		err := rows.Scan(
			&omaExport.JobID,
			&omaExport.VolumeID,
			&omaExport.ExportName,
			&omaExport.Port,
			&omaExport.DevicePath,
			&omaExport.ConfigPath,
			&omaExport.Status,
			&omaExport.CreatedAt,
			&omaExport.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan NBD export row from migratekit_oma: %w", err)
		}

		// Convert to our export info format
		exportInfo := &nbd.ExportInfo{
			ID:         fmt.Sprintf("%d", omaExport.ID),
			VolumeID:   omaExport.VolumeID,
			ExportName: omaExport.ExportName,
			DevicePath: omaExport.DevicePath,
			Port:       omaExport.Port,
			Status:     nbd.ExportStatus(omaExport.Status),
			CreatedAt:  omaExport.CreatedAt,
			UpdatedAt:  omaExport.UpdatedAt,
			Metadata: map[string]string{
				"job_id":      stringPtrToString(omaExport.JobID),
				"config_path": omaExport.ConfigPath,
			},
		}

		exports = append(exports, exportInfo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating NBD export rows from migratekit_oma: %w", err)
	}

	log.WithField("count", len(exports)).Debug("✅ Listed NBD exports from migratekit_oma database")
	return exports, nil
}

// getVMContextIDForJob retrieves the VM context ID for a given job ID from migratekit_oma database
func (r *OMANBDRepository) getVMContextIDForJob(jobID string) (string, error) {
	var vmContextID string
	query := `SELECT vm_context_id FROM replication_jobs WHERE id = ?`

	err := r.db.QueryRow(query, jobID).Scan(&vmContextID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("job ID %s not found", jobID)
		}
		return "", fmt.Errorf("failed to get VM context ID for job %s: %w", jobID, err)
	}

	return vmContextID, nil
}

// getVMContextIDFromVolumeID retrieves the VM context ID for a given volume ID from migratekit_oma database
func (r *OMANBDRepository) getVMContextIDFromVolumeID(volumeID string) (string, error) {
	var vmContextID sql.NullString
	query := `SELECT vm_context_id FROM ossea_volumes WHERE volume_id = ?`

	log.WithField("volume_id", volumeID).Debug("Looking up VM context ID from volume ID")

	err := r.db.QueryRow(query, volumeID).Scan(&vmContextID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.WithField("volume_id", volumeID).Debug("Volume ID not found in ossea_volumes")
			return "", fmt.Errorf("volume ID %s not found in ossea_volumes", volumeID)
		}
		log.WithError(err).WithField("volume_id", volumeID).Debug("Failed to scan VM context ID")
		return "", fmt.Errorf("failed to get VM context ID for volume %s: %w", volumeID, err)
	}

	if !vmContextID.Valid {
		log.WithField("volume_id", volumeID).Debug("VM context ID is NULL for volume")
		return "", nil // Return empty string for NULL values
	}

	log.WithFields(log.Fields{
		"volume_id":     volumeID,
		"vm_context_id": vmContextID.String,
	}).Debug("Successfully retrieved VM context ID from volume ID")

	return vmContextID.String, nil
}
