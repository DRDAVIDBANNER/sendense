package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/nbd"
)

// NBDCleanupService provides comprehensive cleanup for orphaned NBD exports
type NBDCleanupService struct {
	db               *sqlx.DB
	nbdExportManager *nbd.ExportManager
}

// OrphanedExport represents an NBD export that should be cleaned up
type OrphanedExport struct {
	ID                  int       `db:"id" json:"id"`
	VolumeID            string    `db:"volume_id" json:"volume_id"`
	ExportName          string    `db:"export_name" json:"export_name"`
	DevicePath          string    `db:"device_path" json:"device_path"`
	DeviceMappingUUID   *string   `db:"device_mapping_uuid" json:"device_mapping_uuid"`
	Status              string    `db:"status" json:"status"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	OrphanReason        string    `json:"orphan_reason"`
	DeviceMappingExists bool      `json:"device_mapping_exists"`
	VolumeExists        bool      `json:"volume_exists"`
	DeviceExists        bool      `json:"device_exists"`
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	TotalExports            int              `json:"total_exports"`
	OrphanedExports         int              `json:"orphaned_exports"`
	CleanedExports          int              `json:"cleaned_exports"`
	FailedCleanups          int              `json:"failed_cleanups"`
	OrphanedExportDetails   []OrphanedExport `json:"orphaned_export_details"`
	CleanupErrors           []string         `json:"cleanup_errors"`
	ConfigFilesValidated    bool             `json:"config_files_validated"`
	DatabaseInconsistencies []string         `json:"database_inconsistencies"`
	OperationTime           time.Duration    `json:"operation_time"`
}

// NewNBDCleanupService creates a new NBD cleanup service
func NewNBDCleanupService(db *sqlx.DB, nbdExportManager *nbd.ExportManager) *NBDCleanupService {
	return &NBDCleanupService{
		db:               db,
		nbdExportManager: nbdExportManager,
	}
}

// PerformComprehensiveCleanup performs a comprehensive cleanup of orphaned NBD exports
func (ncs *NBDCleanupService) PerformComprehensiveCleanup(ctx context.Context, dryRun bool) (*CleanupResult, error) {
	startTime := time.Now()

	log.WithField("dry_run", dryRun).Info("🧹 Starting comprehensive NBD export cleanup")

	result := &CleanupResult{
		OrphanedExportDetails:   make([]OrphanedExport, 0),
		CleanupErrors:           make([]string, 0),
		DatabaseInconsistencies: make([]string, 0),
	}

	// Step 1: Get all NBD exports from database
	exports, err := ncs.getAllNBDExports(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to get NBD exports: %w", err)
	}
	result.TotalExports = len(exports)

	log.WithField("total_exports", result.TotalExports).Info("📊 Found NBD exports in database")

	// Step 2: Analyze each export for orphan status
	for _, export := range exports {
		orphanStatus := ncs.analyzeExportOrphanStatus(ctx, &export)
		if orphanStatus.OrphanReason != "" {
			result.OrphanedExports++
			result.OrphanedExportDetails = append(result.OrphanedExportDetails, orphanStatus)

			log.WithFields(log.Fields{
				"export_name":   export.ExportName,
				"volume_id":     export.VolumeID,
				"orphan_reason": orphanStatus.OrphanReason,
			}).Warn("🔍 Orphaned NBD export detected")
		}
	}

	// Step 3: Validate NBD configuration consistency
	configValidationResult := ncs.validateConfigurationConsistency(ctx)
	result.ConfigFilesValidated = configValidationResult.Success
	if !configValidationResult.Success {
		result.DatabaseInconsistencies = append(result.DatabaseInconsistencies,
			configValidationResult.Inconsistencies...)
	}

	// Step 4: Perform cleanup if not dry run
	if !dryRun && result.OrphanedExports > 0 {
		log.WithField("orphaned_count", result.OrphanedExports).Info("🔧 Starting cleanup of orphaned exports")

		for _, orphanedExport := range result.OrphanedExportDetails {
			err := ncs.cleanupOrphanedExport(ctx, &orphanedExport)
			if err != nil {
				result.FailedCleanups++
				errorMsg := fmt.Sprintf("Failed to cleanup export %s: %v", orphanedExport.ExportName, err)
				result.CleanupErrors = append(result.CleanupErrors, errorMsg)
				log.WithError(err).WithField("export_name", orphanedExport.ExportName).Error("❌ Failed to cleanup orphaned export")
			} else {
				result.CleanedExports++
				log.WithField("export_name", orphanedExport.ExportName).Info("✅ Successfully cleaned up orphaned export")
			}
		}
	}

	result.OperationTime = time.Since(startTime)

	log.WithFields(log.Fields{
		"total_exports":    result.TotalExports,
		"orphaned_exports": result.OrphanedExports,
		"cleaned_exports":  result.CleanedExports,
		"failed_cleanups":  result.FailedCleanups,
		"operation_time":   result.OperationTime,
		"dry_run":          dryRun,
	}).Info("🎉 NBD export cleanup completed")

	return result, nil
}

// getAllNBDExports retrieves all NBD exports from the database
func (ncs *NBDCleanupService) getAllNBDExports(ctx context.Context) ([]OrphanedExport, error) {
	query := `
		SELECT id, volume_id, export_name, device_path, device_mapping_uuid, status, created_at
		FROM nbd_exports 
		ORDER BY created_at DESC
	`

	var exports []OrphanedExport
	err := ncs.db.SelectContext(ctx, &exports, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query NBD exports: %w", err)
	}

	return exports, nil
}

// analyzeExportOrphanStatus determines if an NBD export is orphaned and why
func (ncs *NBDCleanupService) analyzeExportOrphanStatus(ctx context.Context, export *OrphanedExport) OrphanedExport {
	result := *export // Copy the export
	result.OrphanReason = ""

	// Check 1: Does the device mapping exist?
	if export.DeviceMappingUUID != nil {
		exists, err := ncs.checkDeviceMappingExists(ctx, *export.DeviceMappingUUID)
		if err != nil {
			result.OrphanReason = fmt.Sprintf("Failed to check device mapping: %v", err)
			return result
		}
		result.DeviceMappingExists = exists

		if !exists {
			result.OrphanReason = "Device mapping no longer exists"
			return result
		}
	} else {
		result.DeviceMappingExists = false
		result.OrphanReason = "No device mapping UUID specified"
		return result
	}

	// Check 2: Does the OSSEA volume exist?
	volumeExists, err := ncs.checkOSSEAVolumeExists(ctx, export.VolumeID)
	if err != nil {
		result.OrphanReason = fmt.Sprintf("Failed to check OSSEA volume: %v", err)
		return result
	}
	result.VolumeExists = volumeExists

	if !volumeExists {
		result.OrphanReason = "OSSEA volume no longer exists"
		return result
	}

	// Check 3: Does the device path exist on the filesystem?
	deviceExists := ncs.checkDevicePathExists(export.DevicePath)
	result.DeviceExists = deviceExists

	if !deviceExists {
		result.OrphanReason = "Device path no longer exists on filesystem"
		return result
	}

	// Check 4: Is the export for an SHA volume but device is not in SHA mode?
	if export.DeviceMappingUUID != nil {
		operationMode, err := ncs.getDeviceMappingOperationMode(ctx, *export.DeviceMappingUUID)
		if err != nil {
			result.OrphanReason = fmt.Sprintf("Failed to get operation mode: %v", err)
			return result
		}

		// NBD exports should only exist for SHA mode volumes
		if operationMode != "oma" {
			result.OrphanReason = fmt.Sprintf("Export exists for %s mode volume (should only be oma mode)", operationMode)
			return result
		}
	}

	// If all checks pass, the export is not orphaned
	return result
}

// checkDeviceMappingExists verifies if a device mapping exists
func (ncs *NBDCleanupService) checkDeviceMappingExists(ctx context.Context, deviceMappingUUID string) (bool, error) {
	query := `SELECT COUNT(*) FROM device_mappings WHERE volume_uuid = ?`
	var count int
	err := ncs.db.QueryRowContext(ctx, query, deviceMappingUUID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// checkOSSEAVolumeExists verifies if an OSSEA volume exists
func (ncs *NBDCleanupService) checkOSSEAVolumeExists(ctx context.Context, volumeID string) (bool, error) {
	query := `SELECT COUNT(*) FROM ossea_volumes WHERE volume_id = ?`
	var count int
	err := ncs.db.QueryRowContext(ctx, query, volumeID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// checkDevicePathExists verifies if a device path exists on the filesystem
func (ncs *NBDCleanupService) checkDevicePathExists(devicePath string) bool {
	// For remote VM device paths (from failover VMs), they won't exist locally
	if strings.HasPrefix(devicePath, "remote-vm-") {
		return false
	}

	// TODO: Add actual filesystem check
	// For now, assume device paths starting with /dev/ might exist
	return strings.HasPrefix(devicePath, "/dev/")
}

// getDeviceMappingOperationMode gets the operation mode for a device mapping
func (ncs *NBDCleanupService) getDeviceMappingOperationMode(ctx context.Context, deviceMappingUUID string) (string, error) {
	query := `SELECT operation_mode FROM device_mappings WHERE volume_uuid = ?`
	var operationMode string
	err := ncs.db.QueryRowContext(ctx, query, deviceMappingUUID).Scan(&operationMode)
	if err != nil {
		return "", err
	}
	return operationMode, nil
}

// validateConfigurationConsistency checks for consistency between database and NBD config files
func (ncs *NBDCleanupService) validateConfigurationConsistency(ctx context.Context) struct {
	Success         bool
	Inconsistencies []string
} {
	result := struct {
		Success         bool
		Inconsistencies []string
	}{
		Success:         true,
		Inconsistencies: make([]string, 0),
	}

	// Use the NBD export manager to validate configuration
	if ncs.nbdExportManager != nil {
		err := ncs.nbdExportManager.ValidateExports(ctx)
		if err != nil {
			result.Success = false
			result.Inconsistencies = append(result.Inconsistencies, fmt.Sprintf("NBD configuration validation failed: %v", err))
		}
	}

	return result
}

// cleanupOrphanedExport removes an orphaned NBD export
func (ncs *NBDCleanupService) cleanupOrphanedExport(ctx context.Context, orphanedExport *OrphanedExport) error {
	log.WithFields(log.Fields{
		"export_name":   orphanedExport.ExportName,
		"volume_id":     orphanedExport.VolumeID,
		"orphan_reason": orphanedExport.OrphanReason,
	}).Info("🧹 Cleaning up orphaned NBD export")

	// Step 1: Remove from NBD configuration files via export manager
	if ncs.nbdExportManager != nil {
		err := ncs.nbdExportManager.DeleteExport(ctx, orphanedExport.VolumeID)
		if err != nil {
			log.WithError(err).Warn("Failed to remove NBD export from configuration - continuing with database cleanup")
		}
	}

	// Step 2: Remove from database
	query := `DELETE FROM nbd_exports WHERE id = ?`
	_, err := ncs.db.ExecContext(ctx, query, orphanedExport.ID)
	if err != nil {
		return fmt.Errorf("failed to delete NBD export from database: %w", err)
	}

	log.WithField("export_name", orphanedExport.ExportName).Info("✅ Orphaned NBD export cleaned up successfully")
	return nil
}

// GetOrphanedExportsCount returns the count of orphaned exports without performing cleanup
func (ncs *NBDCleanupService) GetOrphanedExportsCount(ctx context.Context) (int, error) {
	result, err := ncs.PerformComprehensiveCleanup(ctx, true) // Dry run
	if err != nil {
		return 0, err
	}
	return result.OrphanedExports, nil
}

// CleanupExportsByAge removes NBD exports older than specified duration that are orphaned
func (ncs *NBDCleanupService) CleanupExportsByAge(ctx context.Context, maxAge time.Duration, dryRun bool) (*CleanupResult, error) {
	log.WithFields(log.Fields{
		"max_age": maxAge,
		"dry_run": dryRun,
	}).Info("🕐 Starting age-based NBD export cleanup")

	// First perform comprehensive cleanup to identify orphaned exports
	result, err := ncs.PerformComprehensiveCleanup(ctx, true) // Always dry run first
	if err != nil {
		return result, err
	}

	// Filter by age
	cutoffTime := time.Now().Add(-maxAge)
	var ageFilteredOrphans []OrphanedExport

	for _, orphan := range result.OrphanedExportDetails {
		if orphan.CreatedAt.Before(cutoffTime) {
			ageFilteredOrphans = append(ageFilteredOrphans, orphan)
		}
	}

	// Update result with age-filtered orphans
	result.OrphanedExports = len(ageFilteredOrphans)
	result.OrphanedExportDetails = ageFilteredOrphans

	// Perform actual cleanup if not dry run
	if !dryRun && len(ageFilteredOrphans) > 0 {
		result.CleanedExports = 0
		result.FailedCleanups = 0
		result.CleanupErrors = make([]string, 0)

		for _, orphan := range ageFilteredOrphans {
			err := ncs.cleanupOrphanedExport(ctx, &orphan)
			if err != nil {
				result.FailedCleanups++
				result.CleanupErrors = append(result.CleanupErrors, fmt.Sprintf("Failed to cleanup %s: %v", orphan.ExportName, err))
			} else {
				result.CleanedExports++
			}
		}
	}

	log.WithFields(log.Fields{
		"total_orphaned":     len(result.OrphanedExportDetails),
		"age_filtered_count": len(ageFilteredOrphans),
		"cleaned":            result.CleanedExports,
		"failed":             result.FailedCleanups,
	}).Info("🎉 Age-based NBD export cleanup completed")

	return result, nil
}
