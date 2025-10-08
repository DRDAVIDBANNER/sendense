// Package device provides device path integrity validation functionality
package device

import (
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-sha/database"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// IntegrityViolation represents a device path integrity violation
type IntegrityViolation struct {
	ViolationType   string                 `json:"violation_type"`
	Severity        string                 `json:"severity"`        // critical, warning, info
	Description     string                 `json:"description"`
	RecommendedFix  string                 `json:"recommended_fix"`
	AffectedTables  []string               `json:"affected_tables"`
	ConflictDetails map[string]interface{} `json:"conflict_details"`
}

// IntegrityValidationResult contains the full validation results
type IntegrityValidationResult struct {
	HasViolations     bool                  `json:"has_violations"`
	CriticalCount     int                   `json:"critical_count"`
	WarningCount      int                   `json:"warning_count"`
	TotalViolations   int                   `json:"total_violations"`
	Violations        []IntegrityViolation  `json:"violations"`
	ValidationTime    time.Time             `json:"validation_time"`
	DatabaseSnapshot  DatabaseSnapshot      `json:"database_snapshot"`
}

// DatabaseSnapshot captures current state for reference
type DatabaseSnapshot struct {
	OSSEAVolumes      []database.OSSEAVolume        `json:"ossea_volumes"`
	VMExportMappings  []database.VMExportMapping    `json:"vm_export_mappings"`
	NBDExports        []database.NBDExport          `json:"nbd_exports"`
}

// IntegrityValidator provides comprehensive device path integrity validation
type IntegrityValidator struct {
	db       *gorm.DB
	detector *DeviceDetector
}

// NewIntegrityValidator creates a new integrity validator instance
func NewIntegrityValidator(db *gorm.DB, detector *DeviceDetector) *IntegrityValidator {
	return &IntegrityValidator{
		db:       db,
		detector: detector,
	}
}

// ValidateDevicePathConsistency performs comprehensive integrity validation
func (iv *IntegrityValidator) ValidateDevicePathConsistency() (*IntegrityValidationResult, error) {
	log.Info("🔍 Starting comprehensive device path integrity validation")

	result := &IntegrityValidationResult{
		ValidationTime: time.Now(),
		Violations:     make([]IntegrityViolation, 0),
	}

	// Capture database snapshot
	snapshot, err := iv.captureDBSnapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to capture database snapshot: %w", err)
	}
	result.DatabaseSnapshot = *snapshot

	// Run all validation checks
	violations := make([]IntegrityViolation, 0)

	// Check 1: Duplicate device paths in ossea_volumes
	duplicateViolations, err := iv.checkDuplicateOSSEAVolumePaths()
	if err != nil {
		log.WithError(err).Warn("Failed to check duplicate OSSEA volume paths")
	} else {
		violations = append(violations, duplicateViolations...)
	}

	// Check 2: Device path mismatches with CloudStack reality
	cloudStackViolations, err := iv.checkCloudStackDevicePathMismatches()
	if err != nil {
		log.WithError(err).Warn("Failed to check CloudStack device path mismatches")
	} else {
		violations = append(violations, cloudStackViolations...)
	}

	// Check 3: Cross-table device path conflicts
	crossTableViolations, err := iv.checkCrossTableDevicePathConflicts()
	if err != nil {
		log.WithError(err).Warn("Failed to check cross-table device path conflicts")
	} else {
		violations = append(violations, crossTableViolations...)
	}

	// Check 4: Orphaned NBD exports
	orphanedViolations, err := iv.checkOrphanedNBDExports()
	if err != nil {
		log.WithError(err).Warn("Failed to check orphaned NBD exports")
	} else {
		violations = append(violations, orphanedViolations...)
	}

	// Analyze violations
	result.Violations = violations
	result.TotalViolations = len(violations)
	result.HasViolations = len(violations) > 0

	for _, violation := range violations {
		switch violation.Severity {
		case "critical":
			result.CriticalCount++
		case "warning":
			result.WarningCount++
		}
	}

	log.WithFields(log.Fields{
		"total_violations": result.TotalViolations,
		"critical_count":   result.CriticalCount,
		"warning_count":    result.WarningCount,
	}).Info("✅ Device path integrity validation completed")

	return result, nil
}

// checkDuplicateOSSEAVolumePaths finds duplicate device paths in ossea_volumes table
func (iv *IntegrityValidator) checkDuplicateOSSEAVolumePaths() ([]IntegrityViolation, error) {
	log.Debug("Checking for duplicate device paths in ossea_volumes table")

	// Query for device paths that appear multiple times
	var duplicates []struct {
		DevicePath string `json:"device_path"`
		Count      int    `json:"count"`
		VolumeIDs  string `json:"volume_ids"`
	}

	query := `
		SELECT 
			device_path,
			COUNT(*) as count,
			GROUP_CONCAT(volume_id) as volume_ids
		FROM ossea_volumes 
		WHERE device_path != '' AND device_path IS NOT NULL
		GROUP BY device_path 
		HAVING COUNT(*) > 1
	`

	if err := iv.db.Raw(query).Scan(&duplicates).Error; err != nil {
		return nil, fmt.Errorf("failed to query duplicate device paths: %w", err)
	}

	violations := make([]IntegrityViolation, 0)
	for _, dup := range duplicates {
		violation := IntegrityViolation{
			ViolationType:  "duplicate_device_path",
			Severity:       "critical",
			Description:    fmt.Sprintf("Device path '%s' is assigned to %d volumes simultaneously", dup.DevicePath, dup.Count),
			RecommendedFix: "Use DeviceDetector to query CloudStack for actual device assignments and update database",
			AffectedTables: []string{"ossea_volumes"},
			ConflictDetails: map[string]interface{}{
				"device_path":   dup.DevicePath,
				"volume_count":  dup.Count,
				"volume_ids":    dup.VolumeIDs,
			},
		}
		violations = append(violations, violation)
	}

	return violations, nil
}

// checkCloudStackDevicePathMismatches compares database paths with CloudStack reality
func (iv *IntegrityValidator) checkCloudStackDevicePathMismatches() ([]IntegrityViolation, error) {
	if iv.detector == nil {
		log.Warn("DeviceDetector not available, skipping CloudStack device path validation")
		return []IntegrityViolation{}, nil
	}

	log.Debug("Checking device path mismatches with CloudStack API")

	// Get all OSSEA volumes
	var volumes []database.OSSEAVolume
	if err := iv.db.Find(&volumes).Error; err != nil {
		return nil, fmt.Errorf("failed to get OSSEA volumes: %w", err)
	}

	violations := make([]IntegrityViolation, 0)

	for _, volume := range volumes {
		if volume.VolumeID == "" || volume.DevicePath == "" {
			continue // Skip volumes without device paths
		}

		// Get actual device path from CloudStack
		info, err := iv.detector.GetVolumeDeviceInfo(volume.VolumeID)
		if err != nil {
			// Create a warning for volumes we can't validate
			violation := IntegrityViolation{
				ViolationType:  "cloudstack_query_failed",
				Severity:       "warning",
				Description:    fmt.Sprintf("Could not query CloudStack for volume %s: %v", volume.VolumeID, err),
				RecommendedFix: "Check CloudStack connectivity and volume existence",
				AffectedTables: []string{"ossea_volumes"},
				ConflictDetails: map[string]interface{}{
					"volume_id":     volume.VolumeID,
					"volume_name":   volume.VolumeName,
					"database_path": volume.DevicePath,
					"error":         err.Error(),
				},
			}
			violations = append(violations, violation)
			continue
		}

		// Compare paths
		if info.DevicePath != "" && info.DevicePath != volume.DevicePath {
			violation := IntegrityViolation{
				ViolationType:  "cloudstack_database_mismatch",
				Severity:       "critical",
				Description:    fmt.Sprintf("Volume %s: Database shows %s but CloudStack shows %s", volume.VolumeName, volume.DevicePath, info.DevicePath),
				RecommendedFix: fmt.Sprintf("Update database device_path from '%s' to '%s'", volume.DevicePath, info.DevicePath),
				AffectedTables: []string{"ossea_volumes"},
				ConflictDetails: map[string]interface{}{
					"volume_id":        volume.VolumeID,
					"volume_name":      volume.VolumeName,
					"database_path":    volume.DevicePath,
					"cloudstack_path":  info.DevicePath,
					"cloudstack_vm_id": info.VMID,
					"cloudstack_status": info.Status,
				},
			}
			violations = append(violations, violation)
		}
	}

	return violations, nil
}

// checkCrossTableDevicePathConflicts finds conflicts between tables
func (iv *IntegrityValidator) checkCrossTableDevicePathConflicts() ([]IntegrityViolation, error) {
	log.Debug("Checking cross-table device path conflicts")

	violations := make([]IntegrityViolation, 0)

	// Check ossea_volumes vs vm_export_mappings
	var conflicts []struct {
		DevicePath        string `json:"device_path"`
		OSSEAVolumeID     string `json:"ossea_volume_id"`
		OSSEAVolumeName   string `json:"ossea_volume_name"`
		VMExportVMID      string `json:"vm_export_vm_id"`
		VMExportName      string `json:"vm_export_name"`
	}

	query := `
		SELECT 
			ov.device_path,
			ov.volume_id as ossea_volume_id,
			ov.volume_name as ossea_volume_name,
			vem.vm_id as vm_export_vm_id,
			vem.export_name as vm_export_name
		FROM ossea_volumes ov
		JOIN vm_export_mappings vem ON ov.device_path = vem.device_path
		WHERE ov.device_path != '' AND vem.device_path != ''
		AND ov.volume_id != vem.export_name  -- They should be related but different tables
	`

	if err := iv.db.Raw(query).Scan(&conflicts).Error; err != nil {
		return nil, fmt.Errorf("failed to query cross-table conflicts: %w", err)
	}

	for _, conflict := range conflicts {
		violation := IntegrityViolation{
			ViolationType:  "cross_table_device_conflict",
			Severity:       "warning",
			Description:    fmt.Sprintf("Device path '%s' used by both OSSEA volume '%s' and VM export mapping for VM '%s'", conflict.DevicePath, conflict.OSSEAVolumeName, conflict.VMExportVMID),
			RecommendedFix: "Verify that these assignments are intentional and related to the same physical device",
			AffectedTables: []string{"ossea_volumes", "vm_export_mappings"},
			ConflictDetails: map[string]interface{}{
				"device_path":         conflict.DevicePath,
				"ossea_volume_id":     conflict.OSSEAVolumeID,
				"ossea_volume_name":   conflict.OSSEAVolumeName,
				"vm_export_vm_id":     conflict.VMExportVMID,
				"vm_export_name":      conflict.VMExportName,
			},
		}
		violations = append(violations, violation)
	}

	return violations, nil
}

// checkOrphanedNBDExports finds NBD exports without corresponding volumes
func (iv *IntegrityValidator) checkOrphanedNBDExports() ([]IntegrityViolation, error) {
	log.Debug("Checking for orphaned NBD exports")

	// Query for NBD exports that don't have corresponding OSSEA volumes
	var orphaned []database.NBDExport
	
	query := `
		SELECT ne.* FROM nbd_exports ne
		LEFT JOIN ossea_volumes ov ON ne.volume_id = ov.volume_id
		WHERE ov.volume_id IS NULL
	`

	if err := iv.db.Raw(query).Scan(&orphaned).Error; err != nil {
		return nil, fmt.Errorf("failed to query orphaned NBD exports: %w", err)
	}

	violations := make([]IntegrityViolation, 0)
	for _, export := range orphaned {
		violation := IntegrityViolation{
			ViolationType:  "orphaned_nbd_export",
			Severity:       "warning",
			Description:    fmt.Sprintf("NBD export '%s' references non-existent volume '%s'", export.ExportName, export.VolumeID),
			RecommendedFix: "Remove orphaned NBD export or verify volume existence",
			AffectedTables: []string{"nbd_exports"},
			ConflictDetails: map[string]interface{}{
				"export_id":    export.ID,
				"export_name":  export.ExportName,
				"volume_id":    export.VolumeID,
				"device_path":  export.DevicePath,
				"job_id":       export.JobID,
				"status":       export.Status,
			},
		}
		violations = append(violations, violation)
	}

	return violations, nil
}

// captureDBSnapshot captures current database state for reference
func (iv *IntegrityValidator) captureDBSnapshot() (*DatabaseSnapshot, error) {
	snapshot := &DatabaseSnapshot{}

	// Get OSSEA volumes
	if err := iv.db.Find(&snapshot.OSSEAVolumes).Error; err != nil {
		return nil, fmt.Errorf("failed to get OSSEA volumes: %w", err)
	}

	// Get VM export mappings
	if err := iv.db.Find(&snapshot.VMExportMappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get VM export mappings: %w", err)
	}

	// Get NBD exports
	if err := iv.db.Find(&snapshot.NBDExports).Error; err != nil {
		return nil, fmt.Errorf("failed to get NBD exports: %w", err)
	}

	return snapshot, nil
}

// GenerateRepairScript creates SQL commands to fix detected violations
func (iv *IntegrityValidator) GenerateRepairScript(result *IntegrityValidationResult) []string {
	commands := make([]string, 0)
	
	for _, violation := range result.Violations {
		switch violation.ViolationType {
		case "cloudstack_database_mismatch":
			if details, ok := violation.ConflictDetails["volume_id"].(string); ok {
				if cloudStackPath, pathOK := violation.ConflictDetails["cloudstack_path"].(string); pathOK {
					cmd := fmt.Sprintf("UPDATE ossea_volumes SET device_path = '%s' WHERE volume_id = '%s';", 
						cloudStackPath, details)
					commands = append(commands, cmd)
				}
			}
		case "orphaned_nbd_export":
			if exportID, ok := violation.ConflictDetails["export_id"].(int); ok {
				cmd := fmt.Sprintf("DELETE FROM nbd_exports WHERE id = %d;", exportID)
				commands = append(commands, cmd)
			}
		}
	}
	
	return commands
}
