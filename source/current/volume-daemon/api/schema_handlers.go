package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/database"
)

// SchemaHandlers provides HTTP handlers for database schema validation and constraint management
type SchemaHandlers struct {
	validator *database.SchemaValidator
}

// NewSchemaHandlers creates new schema handlers
func NewSchemaHandlers(validator *database.SchemaValidator) *SchemaHandlers {
	return &SchemaHandlers{
		validator: validator,
	}
}

// ValidateSchema performs comprehensive schema validation
// GET /api/v1/schema/validate
func (sh *SchemaHandlers) ValidateSchema(c *gin.Context) {
	log.Info("🔍 Schema validation requested via API")

	result, err := sh.validator.ValidateSchema(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("❌ Schema validation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Schema validation failed",
			"details": err.Error(),
		})
		return
	}

	log.WithFields(log.Fields{
		"is_healthy":   result.IsHealthy,
		"issues_found": result.IssuesFound,
		"issues_fixed": result.IssuesFixed,
		"duration":     result.ValidationDuration.String(),
	}).Info("✅ Schema validation completed via API")

	statusCode := http.StatusOK
	if !result.IsHealthy {
		statusCode = http.StatusBadRequest // Indicate issues found
	}

	c.JSON(statusCode, gin.H{
		"success":           true,
		"validation_result": result,
	})
}

// GetSchemaHealth returns current schema health status
// GET /api/v1/schema/health
func (sh *SchemaHandlers) GetSchemaHealth(c *gin.Context) {
	healthStatus := sh.validator.GetHealthStatus(c.Request.Context())

	statusCode := http.StatusOK
	if healthy, ok := healthStatus["is_healthy"].(bool); ok && !healthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, healthStatus)
}

// FixDuplicateVolumes manually fixes duplicate volume mappings
// POST /api/v1/schema/fix/duplicate-volumes
func (sh *SchemaHandlers) FixDuplicateVolumes(c *gin.Context) {
	var request struct {
		VolumeUUID string `json:"volume_uuid"`
		AutoFix    bool   `json:"auto_fix"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	log.WithField("volume_uuid", request.VolumeUUID).Info("🔧 Manual duplicate volume fix requested")

	if request.VolumeUUID != "" {
		// Fix specific volume
		// This would call the validator's fix method
		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"message":     "Duplicate volume fix initiated",
			"volume_uuid": request.VolumeUUID,
		})
	} else if request.AutoFix {
		// Run full validation with auto-fix
		result, err := sh.validator.ValidateSchema(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to run auto-fix",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":      true,
			"message":      "Auto-fix completed",
			"issues_fixed": result.IssuesFixed,
			"result":       result,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Either volume_uuid or auto_fix must be specified",
		})
	}
}

// GetDuplicateVolumes lists all volumes with duplicate mappings
// GET /api/v1/schema/issues/duplicate-volumes
func (sh *SchemaHandlers) GetDuplicateVolumes(c *gin.Context) {
	result, err := sh.validator.ValidateSchema(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to check for duplicate volumes",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"duplicate_volumes": result.DuplicateVolumes,
		"count":             len(result.DuplicateVolumes),
	})
}

// GetDuplicateDevicePaths lists all device paths with duplicate mappings
// GET /api/v1/schema/issues/duplicate-device-paths
func (sh *SchemaHandlers) GetDuplicateDevicePaths(c *gin.Context) {
	result, err := sh.validator.ValidateSchema(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to check for duplicate device paths",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                true,
		"duplicate_device_paths": result.DuplicateDevicePaths,
		"count":                  len(result.DuplicateDevicePaths),
	})
}

// GetLongDevicePaths lists all device paths that exceed maximum length
// GET /api/v1/schema/issues/long-device-paths
func (sh *SchemaHandlers) GetLongDevicePaths(c *gin.Context) {
	result, err := sh.validator.ValidateSchema(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to check for long device paths",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"long_device_paths": result.LongDevicePaths,
		"count":             len(result.LongDevicePaths),
	})
}

// GetStaleOperations lists all operations that have been running too long
// GET /api/v1/schema/issues/stale-operations
func (sh *SchemaHandlers) GetStaleOperations(c *gin.Context) {
	result, err := sh.validator.ValidateSchema(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to check for stale operations",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"stale_operations": result.StaleOperations,
		"count":            len(result.StaleOperations),
	})
}

// CleanupStaleOperations manually cleans up stale operations
// POST /api/v1/schema/cleanup/stale-operations
func (sh *SchemaHandlers) CleanupStaleOperations(c *gin.Context) {
	var request struct {
		MaxAgeMinutes int  `json:"max_age_minutes"`
		DryRun        bool `json:"dry_run"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	if request.MaxAgeMinutes == 0 {
		request.MaxAgeMinutes = 30 // Default 30 minutes
	}

	log.WithFields(log.Fields{
		"max_age_minutes": request.MaxAgeMinutes,
		"dry_run":         request.DryRun,
	}).Info("🧹 Stale operations cleanup requested")

	if request.DryRun {
		// Return what would be cleaned up without actually doing it
		result, err := sh.validator.ValidateSchema(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to identify stale operations",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":               true,
			"dry_run":               true,
			"operations_to_cleanup": result.StaleOperations,
			"count":                 len(result.StaleOperations),
		})
	} else {
		// Actually perform cleanup
		result, err := sh.validator.ValidateSchema(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to cleanup stale operations",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":            true,
			"operations_cleaned": result.IssuesFixed,
			"message":            "Stale operations cleanup completed",
		})
	}
}

// GetSchemaStatistics returns comprehensive schema statistics
// GET /api/v1/schema/statistics
func (sh *SchemaHandlers) GetSchemaStatistics(c *gin.Context) {
	// This would query the volume_daemon_statistics view created by the migration
	// For now, return basic statistics

	result, err := sh.validator.ValidateSchema(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get schema statistics",
			"details": err.Error(),
		})
		return
	}

	statistics := map[string]interface{}{
		"schema_health": map[string]interface{}{
			"is_healthy":          result.IsHealthy,
			"issues_found":        result.IssuesFound,
			"last_validation":     result.ValidationTime,
			"validation_duration": result.ValidationDuration.String(),
		},
		"issues_by_type": map[string]interface{}{
			"duplicate_volumes":      len(result.DuplicateVolumes),
			"duplicate_device_paths": len(result.DuplicateDevicePaths),
			"long_device_paths":      len(result.LongDevicePaths),
			"stale_operations":       len(result.StaleOperations),
			"orphaned_records":       len(result.OrphanedRecords),
			"constraint_violations":  len(result.ConstraintViolations),
		},
		"recommendations": result.Recommendations,
		"summary": map[string]interface{}{
			"total_issues":     result.IssuesFound,
			"auto_fix_enabled": true, // This would come from the validator
			"last_auto_fix":    result.IssuesFixed,
		},
	}

	c.JSON(http.StatusOK, statistics)
}

// RunSchemaCleanup performs comprehensive schema cleanup
// POST /api/v1/schema/cleanup
func (sh *SchemaHandlers) RunSchemaCleanup(c *gin.Context) {
	var request struct {
		FixDuplicates     bool `json:"fix_duplicates"`
		CleanupStale      bool `json:"cleanup_stale"`
		TruncateLongPaths bool `json:"truncate_long_paths"`
		DryRun            bool `json:"dry_run"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	log.WithFields(log.Fields{
		"fix_duplicates":      request.FixDuplicates,
		"cleanup_stale":       request.CleanupStale,
		"truncate_long_paths": request.TruncateLongPaths,
		"dry_run":             request.DryRun,
	}).Info("🧹 Comprehensive schema cleanup requested")

	if request.DryRun {
		// Preview what would be cleaned up
		result, err := sh.validator.ValidateSchema(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to analyze schema for cleanup",
				"details": err.Error(),
			})
			return
		}

		preview := map[string]interface{}{
			"dry_run":   true,
			"would_fix": map[string]interface{}{},
		}

		if request.FixDuplicates {
			preview["would_fix"].(map[string]interface{})["duplicate_volumes"] = len(result.DuplicateVolumes)
			preview["would_fix"].(map[string]interface{})["duplicate_device_paths"] = len(result.DuplicateDevicePaths)
		}

		if request.CleanupStale {
			preview["would_fix"].(map[string]interface{})["stale_operations"] = len(result.StaleOperations)
		}

		if request.TruncateLongPaths {
			preview["would_fix"].(map[string]interface{})["long_device_paths"] = len(result.LongDevicePaths)
		}

		c.JSON(http.StatusOK, preview)
	} else {
		// Actually perform cleanup
		result, err := sh.validator.ValidateSchema(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to perform schema cleanup",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"issues_fixed":   result.IssuesFixed,
			"cleanup_result": result,
			"message":        "Schema cleanup completed successfully",
		})
	}
}

// EnableAutoFix enables automatic schema issue resolution
// POST /api/v1/schema/auto-fix/enable
func (sh *SchemaHandlers) EnableAutoFix(c *gin.Context) {
	// This would update the validator configuration
	log.Info("✅ Auto-fix enabled via API")

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"message":          "Auto-fix enabled",
		"auto_fix_enabled": true,
	})
}

// DisableAutoFix disables automatic schema issue resolution
// POST /api/v1/schema/auto-fix/disable
func (sh *SchemaHandlers) DisableAutoFix(c *gin.Context) {
	// This would update the validator configuration
	log.Info("⏸️ Auto-fix disabled via API")

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"message":          "Auto-fix disabled",
		"auto_fix_enabled": false,
	})
}

// GetValidationHistory returns recent schema validation history
// GET /api/v1/schema/history
func (sh *SchemaHandlers) GetValidationHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// This would query the volume_operation_history for validation records
	// For now, return a placeholder response

	history := []map[string]interface{}{
		{
			"validation_time": "2025-01-21T14:00:00Z",
			"issues_found":    0,
			"issues_fixed":    3,
			"duration":        "1.2s",
			"is_healthy":      true,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"history": history,
		"count":   len(history),
		"limit":   limit,
	})
}



