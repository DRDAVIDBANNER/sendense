package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// SchemaValidator provides database schema validation and constraint enforcement
type SchemaValidator struct {
	db                  *sql.DB
	validationInterval  time.Duration
	autoFixConstraints  bool
	maxDevicePathLength int
	maxDuplicateAge     time.Duration
}

// ValidationResult represents the result of schema validation
type ValidationResult struct {
	IsHealthy            bool                   `json:"is_healthy"`
	IssuesFound          int                    `json:"issues_found"`
	IssuesFixed          int                    `json:"issues_fixed"`
	DuplicateVolumes     []DuplicateVolumeIssue `json:"duplicate_volumes"`
	DuplicateDevicePaths []DuplicateDeviceIssue `json:"duplicate_device_paths"`
	LongDevicePaths      []LongDevicePathIssue  `json:"long_device_paths"`
	StaleOperations      []StaleOperationIssue  `json:"stale_operations"`
	OrphanedRecords      []OrphanedRecordIssue  `json:"orphaned_records"`
	ConstraintViolations []ConstraintViolation  `json:"constraint_violations"`
	ValidationTime       time.Time              `json:"validation_time"`
	ValidationDuration   time.Duration          `json:"validation_duration"`
	Recommendations      []string               `json:"recommendations"`
}

// Issue types for specific constraint problems
type DuplicateVolumeIssue struct {
	VolumeUUID   string    `json:"volume_uuid"`
	Count        int       `json:"count"`
	DevicePaths  []string  `json:"device_paths"`
	VMIDs        []string  `json:"vm_ids"`
	OldestRecord time.Time `json:"oldest_record"`
	NewestRecord time.Time `json:"newest_record"`
}

type DuplicateDeviceIssue struct {
	VMID        string   `json:"vm_id"`
	DevicePath  string   `json:"device_path"`
	Count       int      `json:"count"`
	VolumeUUIDs []string `json:"volume_uuids"`
}

type LongDevicePathIssue struct {
	VolumeUUID    string `json:"volume_uuid"`
	DevicePath    string `json:"device_path"`
	Length        int    `json:"length"`
	MaxLength     int    `json:"max_length"`
	TruncatedPath string `json:"truncated_path"`
}

type StaleOperationIssue struct {
	OperationID string    `json:"operation_id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	VolumeID    string    `json:"volume_id"`
	AgeMinutes  int       `json:"age_minutes"`
	CreatedAt   time.Time `json:"created_at"`
}

type OrphanedRecordIssue struct {
	Type        string `json:"type"`
	RecordID    string `json:"record_id"`
	VolumeID    string `json:"volume_id"`
	Description string `json:"description"`
}

type ConstraintViolation struct {
	TableName       string `json:"table_name"`
	ConstraintName  string `json:"constraint_name"`
	ViolationType   string `json:"violation_type"`
	AffectedRecords int    `json:"affected_records"`
	Description     string `json:"description"`
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(db *sql.DB) *SchemaValidator {
	return &SchemaValidator{
		db:                  db,
		validationInterval:  10 * time.Minute,
		autoFixConstraints:  true,
		maxDevicePathLength: 255,
		maxDuplicateAge:     5 * time.Minute,
	}
}

// ValidateSchema performs comprehensive schema validation
func (sv *SchemaValidator) ValidateSchema(ctx context.Context) (*ValidationResult, error) {
	startTime := time.Now()
	result := &ValidationResult{
		ValidationTime:       startTime,
		DuplicateVolumes:     make([]DuplicateVolumeIssue, 0),
		DuplicateDevicePaths: make([]DuplicateDeviceIssue, 0),
		LongDevicePaths:      make([]LongDevicePathIssue, 0),
		StaleOperations:      make([]StaleOperationIssue, 0),
		OrphanedRecords:      make([]OrphanedRecordIssue, 0),
		ConstraintViolations: make([]ConstraintViolation, 0),
		Recommendations:      make([]string, 0),
	}

	log.Info("🔍 Starting comprehensive schema validation")

	// Check for duplicate volumes
	if err := sv.checkDuplicateVolumes(ctx, result); err != nil {
		log.WithError(err).Error("Failed to check duplicate volumes")
	}

	// Check for duplicate device paths
	if err := sv.checkDuplicateDevicePaths(ctx, result); err != nil {
		log.WithError(err).Error("Failed to check duplicate device paths")
	}

	// Check for long device paths
	if err := sv.checkLongDevicePaths(ctx, result); err != nil {
		log.WithError(err).Error("Failed to check long device paths")
	}

	// Check for stale operations
	if err := sv.checkStaleOperations(ctx, result); err != nil {
		log.WithError(err).Error("Failed to check stale operations")
	}

	// Check for orphaned records
	if err := sv.checkOrphanedRecords(ctx, result); err != nil {
		log.WithError(err).Error("Failed to check orphaned records")
	}

	// Check for constraint violations
	if err := sv.checkConstraintViolations(ctx, result); err != nil {
		log.WithError(err).Error("Failed to check constraint violations")
	}

	// Auto-fix issues if enabled
	if sv.autoFixConstraints {
		result.IssuesFixed = sv.autoFixIssues(ctx, result)
	}

	// Calculate summary
	result.IssuesFound = len(result.DuplicateVolumes) + len(result.DuplicateDevicePaths) +
		len(result.LongDevicePaths) + len(result.StaleOperations) +
		len(result.OrphanedRecords) + len(result.ConstraintViolations)

	result.IsHealthy = result.IssuesFound == 0
	result.ValidationDuration = time.Since(startTime)

	// Generate recommendations
	sv.generateRecommendations(result)

	log.WithFields(log.Fields{
		"issues_found":        result.IssuesFound,
		"issues_fixed":        result.IssuesFixed,
		"validation_duration": result.ValidationDuration,
		"is_healthy":          result.IsHealthy,
	}).Info("✅ Schema validation completed")

	return result, nil
}

// checkDuplicateVolumes identifies volumes with multiple device mappings
func (sv *SchemaValidator) checkDuplicateVolumes(ctx context.Context, result *ValidationResult) error {
	query := `
		SELECT 
			volume_uuid,
			COUNT(*) as count,
			GROUP_CONCAT(device_path) as device_paths,
			GROUP_CONCAT(vm_id) as vm_ids,
			MIN(created_at) as oldest_record,
			MAX(created_at) as newest_record
		FROM device_mappings 
		GROUP BY volume_uuid 
		HAVING COUNT(*) > 1
	`

	rows, err := sv.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query duplicate volumes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var issue DuplicateVolumeIssue
		var devicePathsStr, vmIDsStr string

		err := rows.Scan(&issue.VolumeUUID, &issue.Count, &devicePathsStr,
			&vmIDsStr, &issue.OldestRecord, &issue.NewestRecord)
		if err != nil {
			continue
		}

		issue.DevicePaths = strings.Split(devicePathsStr, ",")
		issue.VMIDs = strings.Split(vmIDsStr, ",")

		result.DuplicateVolumes = append(result.DuplicateVolumes, issue)
	}

	return nil
}

// checkDuplicateDevicePaths identifies device paths mapped to multiple volumes
func (sv *SchemaValidator) checkDuplicateDevicePaths(ctx context.Context, result *ValidationResult) error {
	query := `
		SELECT 
			vm_id,
			device_path,
			COUNT(*) as count,
			GROUP_CONCAT(volume_uuid) as volume_uuids
		FROM device_mappings 
		GROUP BY vm_id, device_path 
		HAVING COUNT(*) > 1
	`

	rows, err := sv.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query duplicate device paths: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var issue DuplicateDeviceIssue
		var volumeUUIDsStr string

		err := rows.Scan(&issue.VMID, &issue.DevicePath, &issue.Count, &volumeUUIDsStr)
		if err != nil {
			continue
		}

		issue.VolumeUUIDs = strings.Split(volumeUUIDsStr, ",")
		result.DuplicateDevicePaths = append(result.DuplicateDevicePaths, issue)
	}

	return nil
}

// checkLongDevicePaths identifies device paths that exceed the maximum length
func (sv *SchemaValidator) checkLongDevicePaths(ctx context.Context, result *ValidationResult) error {
	query := `
		SELECT volume_uuid, device_path, LENGTH(device_path) as length
		FROM device_mappings 
		WHERE LENGTH(device_path) > ?
	`

	rows, err := sv.db.QueryContext(ctx, query, sv.maxDevicePathLength)
	if err != nil {
		return fmt.Errorf("failed to query long device paths: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var issue LongDevicePathIssue
		err := rows.Scan(&issue.VolumeUUID, &issue.DevicePath, &issue.Length)
		if err != nil {
			continue
		}

		issue.MaxLength = sv.maxDevicePathLength
		if len(issue.DevicePath) > sv.maxDevicePathLength {
			issue.TruncatedPath = issue.DevicePath[:sv.maxDevicePathLength]
		}

		result.LongDevicePaths = append(result.LongDevicePaths, issue)
	}

	return nil
}

// checkStaleOperations identifies operations that have been running too long
func (sv *SchemaValidator) checkStaleOperations(ctx context.Context, result *ValidationResult) error {
	query := `
		SELECT id, type, status, volume_id, 
			   TIMESTAMPDIFF(MINUTE, created_at, NOW()) as age_minutes,
			   created_at
		FROM volume_operations 
		WHERE status IN ('pending', 'executing') 
		  AND created_at < DATE_SUB(NOW(), INTERVAL 30 MINUTE)
	`

	rows, err := sv.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query stale operations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var issue StaleOperationIssue
		err := rows.Scan(&issue.OperationID, &issue.Type, &issue.Status,
			&issue.VolumeID, &issue.AgeMinutes, &issue.CreatedAt)
		if err != nil {
			continue
		}

		result.StaleOperations = append(result.StaleOperations, issue)
	}

	return nil
}

// checkOrphanedRecords identifies records that reference non-existent entities
func (sv *SchemaValidator) checkOrphanedRecords(ctx context.Context, result *ValidationResult) error {
	// Check for volume operations without corresponding device mappings (for completed attachments)
	query := `
		SELECT vo.id, vo.volume_id, 'volume_operation' as type,
			   'Operation completed but no device mapping exists' as description
		FROM volume_operations vo
		LEFT JOIN device_mappings dm ON vo.volume_id = dm.volume_uuid
		WHERE vo.type = 'attach' 
		  AND vo.status = 'completed' 
		  AND dm.volume_uuid IS NULL
	`

	rows, err := sv.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query orphaned records: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var issue OrphanedRecordIssue
		err := rows.Scan(&issue.RecordID, &issue.VolumeID, &issue.Type, &issue.Description)
		if err != nil {
			continue
		}

		result.OrphanedRecords = append(result.OrphanedRecords, issue)
	}

	return nil
}

// checkConstraintViolations checks for any constraint violations
func (sv *SchemaValidator) checkConstraintViolations(ctx context.Context, result *ValidationResult) error {
	// This would check for foreign key violations, unique constraint violations, etc.
	// For now, we'll check for data that would violate our expected constraints

	violations := []ConstraintViolation{}

	// Check for NULL values in NOT NULL columns
	nullChecks := map[string]string{
		"device_mappings.volume_uuid": "SELECT COUNT(*) FROM device_mappings WHERE volume_uuid IS NULL OR volume_uuid = ''",
		"device_mappings.vm_id":       "SELECT COUNT(*) FROM device_mappings WHERE vm_id IS NULL OR vm_id = ''",
		"device_mappings.device_path": "SELECT COUNT(*) FROM device_mappings WHERE device_path IS NULL OR device_path = ''",
	}

	for constraint, query := range nullChecks {
		var count int
		err := sv.db.QueryRowContext(ctx, query).Scan(&count)
		if err != nil {
			continue
		}

		if count > 0 {
			violations = append(violations, ConstraintViolation{
				TableName:       strings.Split(constraint, ".")[0],
				ConstraintName:  constraint,
				ViolationType:   "null_constraint",
				AffectedRecords: count,
				Description:     fmt.Sprintf("Found %d records with NULL values in required field %s", count, constraint),
			})
		}
	}

	result.ConstraintViolations = violations
	return nil
}

// autoFixIssues attempts to automatically fix identified issues
func (sv *SchemaValidator) autoFixIssues(ctx context.Context, result *ValidationResult) int {
	fixedCount := 0

	log.Info("🔧 Starting automatic issue resolution")

	// Fix duplicate volumes by keeping the newest record
	for _, issue := range result.DuplicateVolumes {
		if sv.fixDuplicateVolume(ctx, issue.VolumeUUID) {
			fixedCount++
		}
	}

	// Fix duplicate device paths by removing older mappings
	for _, issue := range result.DuplicateDevicePaths {
		if sv.fixDuplicateDevicePath(ctx, issue.VMID, issue.DevicePath) {
			fixedCount++
		}
	}

	// Fix long device paths by truncating
	for _, issue := range result.LongDevicePaths {
		if sv.fixLongDevicePath(ctx, issue.VolumeUUID, issue.TruncatedPath) {
			fixedCount++
		}
	}

	// Fix stale operations by marking them as failed
	for _, issue := range result.StaleOperations {
		if sv.fixStaleOperation(ctx, issue.OperationID) {
			fixedCount++
		}
	}

	log.WithField("issues_fixed", fixedCount).Info("✅ Automatic issue resolution completed")

	return fixedCount
}

// Individual fix methods

func (sv *SchemaValidator) fixDuplicateVolume(ctx context.Context, volumeUUID string) bool {
	query := `
		DELETE dm1 FROM device_mappings dm1
		INNER JOIN device_mappings dm2 
		WHERE dm1.volume_uuid = dm2.volume_uuid 
		  AND dm1.volume_uuid = ?
		  AND dm1.created_at < dm2.created_at
	`

	result, err := sv.db.ExecContext(ctx, query, volumeUUID)
	if err != nil {
		log.WithError(err).WithField("volume_uuid", volumeUUID).Error("Failed to fix duplicate volume")
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	log.WithFields(log.Fields{
		"volume_uuid":   volumeUUID,
		"rows_affected": rowsAffected,
	}).Info("🔧 Fixed duplicate volume mappings")

	return rowsAffected > 0
}

func (sv *SchemaValidator) fixDuplicateDevicePath(ctx context.Context, vmID, devicePath string) bool {
	// Keep the newest mapping for this device path
	query := `
		DELETE dm1 FROM device_mappings dm1
		INNER JOIN device_mappings dm2 
		WHERE dm1.vm_id = dm2.vm_id 
		  AND dm1.device_path = dm2.device_path
		  AND dm1.vm_id = ? AND dm1.device_path = ?
		  AND dm1.created_at < dm2.created_at
	`

	result, err := sv.db.ExecContext(ctx, query, vmID, devicePath)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"vm_id":       vmID,
			"device_path": devicePath,
		}).Error("Failed to fix duplicate device path")
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"device_path":   devicePath,
		"rows_affected": rowsAffected,
	}).Info("🔧 Fixed duplicate device path mappings")

	return rowsAffected > 0
}

func (sv *SchemaValidator) fixLongDevicePath(ctx context.Context, volumeUUID, truncatedPath string) bool {
	query := `UPDATE device_mappings SET device_path = ? WHERE volume_uuid = ?`

	result, err := sv.db.ExecContext(ctx, query, truncatedPath, volumeUUID)
	if err != nil {
		log.WithError(err).WithField("volume_uuid", volumeUUID).Error("Failed to fix long device path")
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	log.WithFields(log.Fields{
		"volume_uuid":    volumeUUID,
		"truncated_path": truncatedPath,
		"rows_affected":  rowsAffected,
	}).Info("🔧 Fixed long device path")

	return rowsAffected > 0
}

func (sv *SchemaValidator) fixStaleOperation(ctx context.Context, operationID string) bool {
	query := `
		UPDATE volume_operations 
		SET status = 'failed', 
		    error = 'Operation timed out - marked as failed by schema validator',
		    updated_at = NOW(),
		    completed_at = NOW()
		WHERE id = ?
	`

	result, err := sv.db.ExecContext(ctx, query, operationID)
	if err != nil {
		log.WithError(err).WithField("operation_id", operationID).Error("Failed to fix stale operation")
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	log.WithFields(log.Fields{
		"operation_id":  operationID,
		"rows_affected": rowsAffected,
	}).Info("🔧 Fixed stale operation")

	return rowsAffected > 0
}

// generateRecommendations generates actionable recommendations based on validation results
func (sv *SchemaValidator) generateRecommendations(result *ValidationResult) {
	if len(result.DuplicateVolumes) > 0 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Found %d volumes with duplicate mappings - consider running cleanup procedures", len(result.DuplicateVolumes)))
	}

	if len(result.DuplicateDevicePaths) > 0 {
		result.Recommendations = append(result.Recommendations,
			"Duplicate device paths detected - check for concurrent operations on the same VM")
	}

	if len(result.LongDevicePaths) > 0 {
		result.Recommendations = append(result.Recommendations,
			"Long device paths found - consider using shorter naming conventions")
	}

	if len(result.StaleOperations) > 10 {
		result.Recommendations = append(result.Recommendations,
			"High number of stale operations - check Volume Daemon health and CloudStack connectivity")
	}

	if len(result.OrphanedRecords) > 0 {
		result.Recommendations = append(result.Recommendations,
			"Orphaned records detected - run state recovery to re-establish proper relationships")
	}

	if result.IsHealthy {
		result.Recommendations = append(result.Recommendations,
			"Database schema is healthy - no action needed")
	}
}

// GetHealthStatus returns a summary of schema health
func (sv *SchemaValidator) GetHealthStatus(ctx context.Context) map[string]interface{} {
	result, err := sv.ValidateSchema(ctx)
	if err != nil {
		return map[string]interface{}{
			"is_healthy": false,
			"error":      err.Error(),
		}
	}

	return map[string]interface{}{
		"is_healthy":            result.IsHealthy,
		"issues_found":          result.IssuesFound,
		"last_validation":       result.ValidationTime,
		"validation_duration":   result.ValidationDuration.String(),
		"auto_fix_enabled":      sv.autoFixConstraints,
		"duplicate_volumes":     len(result.DuplicateVolumes),
		"duplicate_devices":     len(result.DuplicateDevicePaths),
		"long_device_paths":     len(result.LongDevicePaths),
		"stale_operations":      len(result.StaleOperations),
		"constraint_violations": len(result.ConstraintViolations),
		"recommendations":       result.Recommendations,
	}
}



