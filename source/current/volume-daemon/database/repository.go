package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/models"
	"github.com/vexxhost/migratekit-volume-daemon/service"
)

// Repository implements the VolumeRepository interface
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new volume repository
func NewRepository(db *sqlx.DB) service.VolumeRepository {
	return &Repository{db: db}
}

// CreateOperation creates a new volume operation record
func (r *Repository) CreateOperation(ctx context.Context, op *models.VolumeOperation) error {
	requestJSON, err := json.Marshal(op.Request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	var responseJSON []byte
	if op.Response != nil {
		responseJSON, err = json.Marshal(op.Response)
		if err != nil {
			return fmt.Errorf("failed to marshal response: %w", err)
		}
	}

	query := `
		INSERT INTO volume_operations (
			id, type, status, volume_id, vm_id, request, response, error, created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		op.ID, op.Type, op.Status, op.VolumeID, op.VMID,
		requestJSON, responseJSON, op.Error,
		op.CreatedAt, op.UpdatedAt, op.CompletedAt)

	if err != nil {
		return fmt.Errorf("failed to create operation: %w", err)
	}

	return nil
}

// UpdateOperation updates an existing volume operation record
func (r *Repository) UpdateOperation(ctx context.Context, op *models.VolumeOperation) error {
	var responseJSON []byte
	var err error
	if op.Response != nil {
		responseJSON, err = json.Marshal(op.Response)
		if err != nil {
			return fmt.Errorf("failed to marshal response: %w", err)
		}
	}

	query := `
		UPDATE volume_operations 
		SET status = ?, response = ?, error = ?, updated_at = ?, completed_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		op.Status, responseJSON, op.Error, op.UpdatedAt, op.CompletedAt, op.ID)

	if err != nil {
		return fmt.Errorf("failed to update operation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("operation not found: %s", op.ID)
	}

	return nil
}

// GetOperation retrieves a volume operation by ID
func (r *Repository) GetOperation(ctx context.Context, operationID string) (*models.VolumeOperation, error) {
	query := `
		SELECT id, type, status, volume_id, vm_id, request, response, error, 
		       created_at, updated_at, completed_at
		FROM volume_operations 
		WHERE id = ?
	`

	var op models.VolumeOperation
	var requestJSON, responseJSON sql.NullString

	err := r.db.QueryRowContext(ctx, query, operationID).Scan(
		&op.ID, &op.Type, &op.Status, &op.VolumeID, &op.VMID,
		&requestJSON, &responseJSON, &op.Error,
		&op.CreatedAt, &op.UpdatedAt, &op.CompletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("operation not found: %s", operationID)
		}
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}

	// Unmarshal JSON fields
	if requestJSON.Valid {
		err = json.Unmarshal([]byte(requestJSON.String), &op.Request)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal request: %w", err)
		}
	}

	if responseJSON.Valid {
		err = json.Unmarshal([]byte(responseJSON.String), &op.Response)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return &op, nil
}

// ListOperations lists volume operations based on filter criteria
func (r *Repository) ListOperations(ctx context.Context, filter models.OperationFilter) ([]models.VolumeOperation, error) {
	query := `
		SELECT id, type, status, volume_id, vm_id, request, response, error,
		       created_at, updated_at, completed_at
		FROM volume_operations 
		WHERE 1=1
	`

	args := []interface{}{}

	if filter.Type != nil {
		query += " AND type = ?"
		args = append(args, *filter.Type)
	}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}

	if filter.VolumeID != nil {
		query += " AND volume_id = ?"
		args = append(args, *filter.VolumeID)
	}

	if filter.VMID != nil {
		query += " AND vm_id = ?"
		args = append(args, *filter.VMID)
	}

	if filter.Since != nil {
		query += " AND created_at >= ?"
		args = append(args, *filter.Since)
	}

	if filter.Until != nil {
		query += " AND created_at <= ?"
		args = append(args, *filter.Until)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list operations: %w", err)
	}
	defer rows.Close()

	var operations []models.VolumeOperation
	for rows.Next() {
		var op models.VolumeOperation
		var requestJSON, responseJSON sql.NullString

		err := rows.Scan(
			&op.ID, &op.Type, &op.Status, &op.VolumeID, &op.VMID,
			&requestJSON, &responseJSON, &op.Error,
			&op.CreatedAt, &op.UpdatedAt, &op.CompletedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan operation: %w", err)
		}

		// Unmarshal JSON fields
		if requestJSON.Valid {
			err = json.Unmarshal([]byte(requestJSON.String), &op.Request)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal request: %w", err)
			}
		}

		if responseJSON.Valid {
			err = json.Unmarshal([]byte(responseJSON.String), &op.Response)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		operations = append(operations, op)
	}

	return operations, nil
}

// CreateMapping creates a new device mapping record
func (r *Repository) CreateMapping(ctx context.Context, mapping *models.DeviceMapping) error {
	log.WithField("volume_uuid", mapping.VolumeUUID).Info("🔍 CreateMapping called - checking VM context ID")

	// Get VM context ID from volume ID if not already set
	if mapping.VMContextID == nil || *mapping.VMContextID == "" {
		log.WithField("volume_uuid", mapping.VolumeUUID).Info("🔍 VM context ID is nil/empty, looking up from volume ID")
		if vmCtxID, err := r.getVMContextIDFromVolumeID(mapping.VolumeUUID); err != nil {
			log.WithError(err).WithField("volume_uuid", mapping.VolumeUUID).Warn("❌ Failed to get VM context ID for device mapping, continuing without it")
		} else if vmCtxID != "" {
			mapping.VMContextID = &vmCtxID
			log.WithFields(log.Fields{
				"volume_uuid":   mapping.VolumeUUID,
				"vm_context_id": vmCtxID,
			}).Info("✅ Successfully set VM context ID for device mapping")
		} else {
			log.WithField("volume_uuid", mapping.VolumeUUID).Warn("⚠️ VM context lookup returned empty string")
		}
	} else {
		log.WithFields(log.Fields{
			"volume_uuid":   mapping.VolumeUUID,
			"vm_context_id": *mapping.VMContextID,
		}).Info("✅ VM context ID already set for device mapping")
	}
	query := `
		INSERT INTO device_mappings (
			vm_context_id, volume_uuid, volume_id_numeric, vm_id, device_path, cloudstack_state, linux_state, 
			operation_mode, cloudstack_device_id, requires_device_correlation,
			size, last_sync, created_at, updated_at,
			persistent_device_name, symlink_path
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		mapping.VMContextID, mapping.VolumeUUID, mapping.VolumeIDNumeric, mapping.VMID, mapping.DevicePath,
		mapping.CloudStackState, mapping.LinuxState, mapping.OperationMode,
		mapping.CloudStackDeviceID, mapping.RequiresDeviceCorrelation,
		mapping.Size, mapping.LastSync, mapping.CreatedAt, mapping.UpdatedAt,
		mapping.PersistentDeviceName, mapping.SymlinkPath)

	if err != nil {
		return fmt.Errorf("failed to create mapping: %w", err)
	}

	return nil
}

// getVMContextIDFromVolumeID retrieves the VM context ID for a given volume ID from migratekit_oma database
func (r *Repository) getVMContextIDFromVolumeID(volumeID string) (string, error) {
	var vmContextID string
	query := `SELECT vm_context_id FROM ossea_volumes WHERE volume_id = ?`

	err := r.db.QueryRow(query, volumeID).Scan(&vmContextID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("volume ID %s not found in ossea_volumes", volumeID)
		}
		return "", fmt.Errorf("failed to get VM context ID for volume %s: %w", volumeID, err)
	}

	return vmContextID, nil
}

// UpdateMapping updates an existing device mapping record
func (r *Repository) UpdateMapping(ctx context.Context, mapping *models.DeviceMapping) error {
	query := `
		UPDATE device_mappings 
		SET vm_id = ?, device_path = ?, cloudstack_state = ?, linux_state = ?, 
		    operation_mode = ?, cloudstack_device_id = ?, requires_device_correlation = ?,
		    size = ?, last_sync = ?, updated_at = ?
		WHERE volume_uuid = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		mapping.VMID, mapping.DevicePath, mapping.CloudStackState,
		mapping.LinuxState, mapping.OperationMode, mapping.CloudStackDeviceID,
		mapping.RequiresDeviceCorrelation, mapping.Size, mapping.LastSync,
		mapping.UpdatedAt, mapping.VolumeUUID)

	if err != nil {
		return fmt.Errorf("failed to update mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("mapping not found for volume: %s", mapping.VolumeUUID)
	}

	return nil
}

// DeleteMapping deletes a device mapping record
func (r *Repository) DeleteMapping(ctx context.Context, volumeID string) error {
	query := `DELETE FROM device_mappings WHERE volume_uuid = ?`

	result, err := r.db.ExecContext(ctx, query, volumeID)
	if err != nil {
		return fmt.Errorf("failed to delete mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("mapping not found for volume: %s", volumeID)
	}

	return nil
}

// GetMapping retrieves a device mapping by volume ID
func (r *Repository) GetMapping(ctx context.Context, volumeID string) (*models.DeviceMapping, error) {
	query := `
		SELECT volume_uuid, volume_id_numeric, vm_id, device_path, cloudstack_state, linux_state,
		       operation_mode, cloudstack_device_id, requires_device_correlation,
		       size, last_sync, created_at, updated_at
		FROM device_mappings 
		WHERE volume_uuid = ?
	`

	var mapping models.DeviceMapping
	err := r.db.QueryRowContext(ctx, query, volumeID).Scan(
		&mapping.VolumeUUID, &mapping.VolumeIDNumeric, &mapping.VMID, &mapping.DevicePath,
		&mapping.CloudStackState, &mapping.LinuxState, &mapping.OperationMode,
		&mapping.CloudStackDeviceID, &mapping.RequiresDeviceCorrelation,
		&mapping.Size, &mapping.LastSync, &mapping.CreatedAt, &mapping.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mapping not found for volume: %s", volumeID)
		}
		return nil, fmt.Errorf("failed to get mapping: %w", err)
	}

	return &mapping, nil
}

// GetMappingByDevice retrieves a device mapping by device path
func (r *Repository) GetMappingByDevice(ctx context.Context, devicePath string) (*models.DeviceMapping, error) {
	query := `
		SELECT volume_uuid, volume_id_numeric, vm_id, device_path, cloudstack_state, linux_state,
		       operation_mode, cloudstack_device_id, requires_device_correlation,
		       size, last_sync, created_at, updated_at
		FROM device_mappings 
		WHERE device_path = ?
	`

	var mapping models.DeviceMapping
	err := r.db.QueryRowContext(ctx, query, devicePath).Scan(
		&mapping.VolumeUUID, &mapping.VolumeIDNumeric, &mapping.VMID, &mapping.DevicePath,
		&mapping.CloudStackState, &mapping.LinuxState, &mapping.OperationMode,
		&mapping.CloudStackDeviceID, &mapping.RequiresDeviceCorrelation,
		&mapping.Size, &mapping.LastSync, &mapping.CreatedAt, &mapping.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mapping not found for device: %s", devicePath)
		}
		return nil, fmt.Errorf("failed to get mapping by device: %w", err)
	}

	return &mapping, nil
}

// ListMappingsForVM lists all device mappings for a specific VM
func (r *Repository) ListMappingsForVM(ctx context.Context, vmID string) ([]models.DeviceMapping, error) {
	query := `
		SELECT volume_uuid, volume_id_numeric, vm_id, device_path, cloudstack_state, linux_state,
		       operation_mode, cloudstack_device_id, requires_device_correlation,
		       size, last_sync, created_at, updated_at
		FROM device_mappings 
		WHERE vm_id = ?
		ORDER BY device_path
	`

	rows, err := r.db.QueryContext(ctx, query, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to list mappings for VM: %w", err)
	}
	defer rows.Close()

	var mappings []models.DeviceMapping
	for rows.Next() {
		var mapping models.DeviceMapping
		err := rows.Scan(
			&mapping.VolumeUUID, &mapping.VolumeIDNumeric, &mapping.VMID, &mapping.DevicePath,
			&mapping.CloudStackState, &mapping.LinuxState, &mapping.OperationMode,
			&mapping.CloudStackDeviceID, &mapping.RequiresDeviceCorrelation,
			&mapping.Size, &mapping.LastSync, &mapping.CreatedAt, &mapping.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan mapping: %w", err)
		}

		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// UpdateDeviceMapping updates a device mapping record with all fields including snapshot info
func (r *Repository) UpdateDeviceMapping(ctx context.Context, mapping *models.DeviceMapping) error {
	query := `
		UPDATE device_mappings 
		SET vm_context_id = ?, vm_id = ?, device_path = ?, cloudstack_state = ?, linux_state = ?, 
		    operation_mode = ?, cloudstack_device_id = ?, requires_device_correlation = ?,
		    size = ?, last_sync = ?, updated_at = ?,
		    ossea_snapshot_id = ?, snapshot_created_at = ?, snapshot_status = ?
		WHERE volume_uuid = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		mapping.VMContextID, mapping.VMID, mapping.DevicePath, mapping.CloudStackState,
		mapping.LinuxState, mapping.OperationMode, mapping.CloudStackDeviceID,
		mapping.RequiresDeviceCorrelation, mapping.Size, mapping.LastSync,
		mapping.UpdatedAt, mapping.OSSEASnapshotID, mapping.SnapshotCreatedAt,
		mapping.SnapshotStatus, mapping.VolumeUUID)

	if err != nil {
		return fmt.Errorf("failed to update device mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no device mapping found for volume UUID %s", mapping.VolumeUUID)
	}

	return nil
}

// GetDeviceMappingByVolumeUUID gets a device mapping by volume UUID
func (r *Repository) GetDeviceMappingByVolumeUUID(ctx context.Context, volumeUUID string) (*models.DeviceMapping, error) {
	query := `
		SELECT id, vm_context_id, volume_uuid, volume_id_numeric, vm_id, operation_mode,
		       cloudstack_device_id, requires_device_correlation, device_path,
		       cloudstack_state, linux_state, size, last_sync, created_at, updated_at,
		       ossea_snapshot_id, snapshot_created_at, snapshot_status
		FROM device_mappings 
		WHERE volume_uuid = ?
	`

	var mapping models.DeviceMapping
	err := r.db.GetContext(ctx, &mapping, query, volumeUUID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no device mapping found for volume UUID %s", volumeUUID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device mapping: %w", err)
	}

	return &mapping, nil
}

// GetDeviceMappingsByVMContext gets all device mappings for a VM context
func (r *Repository) GetDeviceMappingsByVMContext(ctx context.Context, vmContextID string) ([]models.DeviceMapping, error) {
	query := `
		SELECT id, vm_context_id, volume_uuid, volume_id_numeric, vm_id, operation_mode,
		       cloudstack_device_id, requires_device_correlation, device_path,
		       cloudstack_state, linux_state, size, last_sync, created_at, updated_at,
		       ossea_snapshot_id, snapshot_created_at, snapshot_status
		FROM device_mappings 
		WHERE vm_context_id = ?
		ORDER BY created_at ASC
	`

	var mappings []models.DeviceMapping
	err := r.db.SelectContext(ctx, &mappings, query, vmContextID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device mappings for VM context %s: %w", vmContextID, err)
	}

	return mappings, nil
}

// Ping checks database connectivity
func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// GetOMAVMID retrieves the SHA VM ID from the active ossea_configs record
func (r *Repository) GetOMAVMID(ctx context.Context) (string, error) {
	var shaVMID string
	query := "SELECT oma_vm_id FROM ossea_configs WHERE is_active = 1 LIMIT 1"

	err := r.db.QueryRowContext(ctx, query).Scan(&shaVMID)
	if err != nil {
		return "", fmt.Errorf("failed to query SHA VM ID from ossea_configs: %w", err)
	}

	if shaVMID == "" {
		return "", fmt.Errorf("SHA VM ID is empty in ossea_configs table")
	}

	return shaVMID, nil
}
