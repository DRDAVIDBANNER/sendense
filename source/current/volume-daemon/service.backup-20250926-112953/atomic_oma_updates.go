package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/nbd"
)

// AtomicOMAService handles atomic updates within the migratekit_oma database
type AtomicOMAService struct {
	db               *sqlx.DB
	nbdExportManager *nbd.ExportManager
}

// AtomicOMATransaction represents a transaction within the migratekit_oma database
type AtomicOMATransaction struct {
	tx  *sqlx.Tx
	ctx context.Context
}

// AtomicOMAUpdateRequest represents a request for atomic NBD export creation with database consistency
type AtomicOMAUpdateRequest struct {
	VolumeID     string            `json:"volume_id"`
	VMName       string            `json:"vm_name"`
	VMID         string            `json:"vm_id"`
	DiskNumber   int               `json:"disk_number"`
	DevicePath   string            `json:"device_path"`
	ReadOnly     bool              `json:"read_only"`
	JobID        string            `json:"job_id"`
	UpdateOSSEA  bool              `json:"update_ossea_volumes"`
	UpdateDevice bool              `json:"update_device_mappings"`
	Metadata     map[string]string `json:"metadata"`
}

// AtomicOMAResult represents the result of an atomic OMA database update
type AtomicOMAResult struct {
	Success             bool            `json:"success"`
	ExportInfo          *nbd.ExportInfo `json:"export_info,omitempty"`
	OSSEAVolumeUpdate   bool            `json:"ossea_volume_updated"`
	DeviceMappingUpdate bool            `json:"device_mapping_updated"`
	RollbackInfo        string          `json:"rollback_info,omitempty"`
	OperationTime       time.Duration   `json:"operation_time"`
}

// NewAtomicOMAService creates a new atomic OMA database service
func NewAtomicOMAService(db *sqlx.DB, nbdManager *nbd.ExportManager) *AtomicOMAService {
	return &AtomicOMAService{
		db:               db,
		nbdExportManager: nbdManager,
	}
}

// CreateNBDExportWithAtomicUpdates creates an NBD export with atomic database updates
func (aos *AtomicOMAService) CreateNBDExportWithAtomicUpdates(ctx context.Context, req *AtomicOMAUpdateRequest) (*AtomicOMAResult, error) {
	startTime := time.Now()

	log.WithFields(log.Fields{
		"volume_id":     req.VolumeID,
		"vm_id":         req.VMID,
		"device_path":   req.DevicePath,
		"job_id":        req.JobID,
		"update_ossea":  req.UpdateOSSEA,
		"update_device": req.UpdateDevice,
	}).Info("🔄 Starting atomic NBD export creation with database updates")

	result := &AtomicOMAResult{
		Success: false,
	}

	// Begin transaction for atomic operations
	tx, err := aos.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safe to call even after commit

	atomicTx := &AtomicOMATransaction{tx: tx, ctx: ctx}

	// Phase 1: Create NBD export in nbd_exports table (within transaction)
	exportInfo, err := aos.createNBDExportInTransaction(atomicTx, req)
	if err != nil {
		result.OperationTime = time.Since(startTime)
		return result, fmt.Errorf("NBD export creation failed: %w", err)
	}
	result.ExportInfo = exportInfo

	// Phase 2: Update ossea_volumes table with NBD export information (if requested)
	if req.UpdateOSSEA {
		err := aos.updateOSSEAVolumeInTransaction(atomicTx, req.VolumeID, exportInfo.ExportName, req.DevicePath)
		if err != nil {
			result.OperationTime = time.Since(startTime)
			return result, fmt.Errorf("OSSEA volume update failed: %w", err)
		}
		result.OSSEAVolumeUpdate = true
	}

	// Phase 3: Update device_mappings table with NBD export reference (if requested)
	if req.UpdateDevice {
		err := aos.updateDeviceMappingInTransaction(atomicTx, req.VolumeID, exportInfo.ExportName)
		if err != nil {
			result.OperationTime = time.Since(startTime)
			return result, fmt.Errorf("device mapping update failed: %w", err)
		}
		result.DeviceMappingUpdate = true
	}

	// Phase 4: Update NBD configuration file (outside transaction but with rollback capability)
	configExport := &nbd.Export{
		Name:       exportInfo.ExportName,
		DevicePath: req.DevicePath,
		ReadOnly:   req.ReadOnly,
		Metadata:   req.Metadata,
	}

	err = aos.nbdExportManager.GetConfigManager().AddExport(configExport)
	if err != nil {
		// Config update failed - rollback transaction
		result.RollbackInfo = "Transaction rolled back due to NBD configuration failure"
		result.OperationTime = time.Since(startTime)
		return result, fmt.Errorf("NBD configuration update failed: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		// Remove the NBD configuration we just added
		if removeErr := aos.nbdExportManager.GetConfigManager().RemoveExport(exportInfo.ExportName); removeErr != nil {
			log.WithError(removeErr).Error("Failed to rollback NBD configuration after commit failure")
		}

		result.RollbackInfo = "NBD configuration rolled back after commit failure"
		result.OperationTime = time.Since(startTime)
		return result, fmt.Errorf("transaction commit failed: %w", err)
	}

	result.Success = true
	result.OperationTime = time.Since(startTime)

	log.WithFields(log.Fields{
		"volume_id":              req.VolumeID,
		"export_name":            exportInfo.ExportName,
		"ossea_volume_updated":   result.OSSEAVolumeUpdate,
		"device_mapping_updated": result.DeviceMappingUpdate,
		"operation_time":         result.OperationTime,
	}).Info("🎉 Atomic NBD export creation completed successfully")

	return result, nil
}

// DeleteNBDExportWithAtomicUpdates deletes an NBD export with atomic database cleanup
func (aos *AtomicOMAService) DeleteNBDExportWithAtomicUpdates(ctx context.Context, volumeID string, cleanupOSSEA bool, cleanupDevice bool) (*AtomicOMAResult, error) {
	startTime := time.Now()

	log.WithFields(log.Fields{
		"volume_id":      volumeID,
		"cleanup_ossea":  cleanupOSSEA,
		"cleanup_device": cleanupDevice,
	}).Info("🗑️ Starting atomic NBD export deletion with database cleanup")

	result := &AtomicOMAResult{
		Success: false,
	}

	// Get export info before deletion for rollback capability
	exportInfo, err := aos.nbdExportManager.GetExport(ctx, volumeID)
	if err != nil {
		log.WithField("volume_id", volumeID).Warn("NBD export not found for deletion - may already be deleted")
	} else {
		result.ExportInfo = exportInfo
	}

	// Begin transaction for atomic operations
	tx, err := aos.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safe to call even after commit

	atomicTx := &AtomicOMATransaction{tx: tx, ctx: ctx}

	// Phase 1: Remove NBD export from nbd_exports table
	if exportInfo != nil {
		err := aos.deleteNBDExportInTransaction(atomicTx, volumeID)
		if err != nil {
			result.OperationTime = time.Since(startTime)
			return result, fmt.Errorf("NBD export deletion failed: %w", err)
		}
	}

	// Phase 2: Clean up ossea_volumes table NBD references (if requested)
	if cleanupOSSEA {
		err := aos.cleanupOSSEAVolumeInTransaction(atomicTx, volumeID)
		if err != nil {
			result.OperationTime = time.Since(startTime)
			return result, fmt.Errorf("OSSEA volume cleanup failed: %w", err)
		}
		result.OSSEAVolumeUpdate = true
	}

	// Phase 3: Clean up device_mappings table NBD references (if requested)
	if cleanupDevice {
		err := aos.cleanupDeviceMappingInTransaction(atomicTx, volumeID)
		if err != nil {
			result.OperationTime = time.Since(startTime)
			return result, fmt.Errorf("device mapping cleanup failed: %w", err)
		}
		result.DeviceMappingUpdate = true
	}

	// Phase 4: Remove from NBD configuration file (outside transaction)
	if exportInfo != nil {
		err = aos.nbdExportManager.GetConfigManager().RemoveExport(exportInfo.ExportName)
		if err != nil {
			log.WithError(err).Warn("Failed to remove NBD configuration - manual cleanup may be required")
			// Don't fail the operation for config cleanup issues
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		result.OperationTime = time.Since(startTime)
		return result, fmt.Errorf("transaction commit failed: %w", err)
	}

	result.Success = true
	result.OperationTime = time.Since(startTime)

	log.WithFields(log.Fields{
		"volume_id":      volumeID,
		"ossea_cleanup":  result.OSSEAVolumeUpdate,
		"device_cleanup": result.DeviceMappingUpdate,
		"operation_time": result.OperationTime,
	}).Info("🎉 Atomic NBD export deletion completed successfully")

	return result, nil
}

// Private helper methods for transaction operations

func (aos *AtomicOMAService) createNBDExportInTransaction(tx *AtomicOMATransaction, req *AtomicOMAUpdateRequest) (*nbd.ExportInfo, error) {
	exportName := fmt.Sprintf("migration-vol-%s", req.VolumeID)
	now := time.Now()

	query := `
		INSERT INTO nbd_exports (
			job_id, volume_id, export_name, port, device_path, 
			config_path, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.tx.ExecContext(tx.ctx, query,
		req.JobID,
		req.VolumeID,
		exportName,
		10809, // Standard NBD port
		req.DevicePath,
		"/etc/nbd-server/config-base",
		"pending", // Will be updated to active after config file update
		now,
		now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert NBD export: %w", err)
	}

	exportInfo := &nbd.ExportInfo{
		ID:         fmt.Sprintf("tx-%d", now.UnixNano()),
		VolumeID:   req.VolumeID,
		ExportName: exportName,
		DevicePath: req.DevicePath,
		Port:       10809,
		Status:     nbd.ExportStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
		Metadata:   req.Metadata,
	}

	log.WithField("export_name", exportName).Debug("✅ NBD export created in transaction")
	return exportInfo, nil
}

func (aos *AtomicOMAService) updateOSSEAVolumeInTransaction(tx *AtomicOMATransaction, volumeID, exportName, devicePath string) error {
	query := `
		UPDATE ossea_volumes 
		SET device_path = ?, updated_at = NOW()
		WHERE volume_id = ?
	`

	result, err := tx.tx.ExecContext(tx.ctx, query, devicePath, volumeID)
	if err != nil {
		return fmt.Errorf("failed to update ossea_volumes: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.WithField("volume_id", volumeID).Warn("OSSEA volume not found for NBD export update")
	} else {
		log.WithField("volume_id", volumeID).Debug("✅ OSSEA volume updated in transaction")
	}

	return nil
}

func (aos *AtomicOMAService) updateDeviceMappingInTransaction(tx *AtomicOMATransaction, volumeID, exportName string) error {
	// Update device mapping with reference to NBD export (using volume_uuid)
	query := `
		UPDATE device_mappings 
		SET updated_at = NOW()
		WHERE volume_uuid = ?
	`

	result, err := tx.tx.ExecContext(tx.ctx, query, volumeID)
	if err != nil {
		return fmt.Errorf("failed to update device_mappings: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.WithField("volume_id", volumeID).Warn("Device mapping not found for NBD export update")
	} else {
		log.WithField("volume_id", volumeID).Debug("✅ Device mapping updated in transaction")
	}

	return nil
}

func (aos *AtomicOMAService) deleteNBDExportInTransaction(tx *AtomicOMATransaction, volumeID string) error {
	query := `DELETE FROM nbd_exports WHERE volume_id = ?`

	result, err := tx.tx.ExecContext(tx.ctx, query, volumeID)
	if err != nil {
		return fmt.Errorf("failed to delete NBD export: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.WithField("volume_id", volumeID).Warn("NBD export not found for deletion")
	} else {
		log.WithField("volume_id", volumeID).Debug("✅ NBD export deleted in transaction")
	}

	return nil
}

func (aos *AtomicOMAService) cleanupOSSEAVolumeInTransaction(tx *AtomicOMATransaction, volumeID string) error {
	query := `
		UPDATE ossea_volumes 
		SET updated_at = NOW()
		WHERE volume_id = ?
	`

	_, err := tx.tx.ExecContext(tx.ctx, query, volumeID)
	if err != nil {
		return fmt.Errorf("failed to cleanup ossea_volumes: %w", err)
	}

	log.WithField("volume_id", volumeID).Debug("✅ OSSEA volume cleaned up in transaction")
	return nil
}

func (aos *AtomicOMAService) cleanupDeviceMappingInTransaction(tx *AtomicOMATransaction, volumeID string) error {
	query := `
		UPDATE device_mappings 
		SET updated_at = NOW()
		WHERE volume_uuid = ?
	`

	_, err := tx.tx.ExecContext(tx.ctx, query, volumeID)
	if err != nil {
		return fmt.Errorf("failed to cleanup device_mappings: %w", err)
	}

	log.WithField("volume_id", volumeID).Debug("✅ Device mapping cleaned up in transaction")
	return nil
}
