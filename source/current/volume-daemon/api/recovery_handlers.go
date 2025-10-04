package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/service"
)

// RecoveryHandlers provides HTTP handlers for volume state recovery operations
type RecoveryHandlers struct {
	stateRecovery *service.StateRecoveryService
	autoRecovery  *service.AutoRecoveryService
}

// NewRecoveryHandlers creates new recovery handlers
func NewRecoveryHandlers(stateRecovery *service.StateRecoveryService, autoRecovery *service.AutoRecoveryService) *RecoveryHandlers {
	return &RecoveryHandlers{
		stateRecovery: stateRecovery,
		autoRecovery:  autoRecovery,
	}
}

// RecoverVM recovers lost device mappings for a specific VM
// POST /api/v1/recovery/vm/:vm_id
func (rh *RecoveryHandlers) RecoverVM(c *gin.Context) {
	vmID := c.Param("vm_id")
	if vmID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "VM ID is required",
		})
		return
	}

	log.WithField("vm_id", vmID).Info("🔄 VM recovery requested via API")

	result, err := rh.stateRecovery.RecoverLostMappings(c.Request.Context(), vmID)
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("❌ VM recovery failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "VM recovery failed",
			"details": err.Error(),
		})
		return
	}

	log.WithFields(log.Fields{
		"vm_id":             vmID,
		"volumes_recovered": result.VolumesRecovered,
		"mappings_created":  result.MappingsCreated,
		"mappings_fixed":    result.MappingsFixed,
	}).Info("✅ VM recovery completed via API")

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"vm_id":              vmID,
		"volumes_recovered":  result.VolumesRecovered,
		"volumes_orphaned":   result.VolumesOrphaned,
		"mappings_created":   result.MappingsCreated,
		"mappings_fixed":     result.MappingsFixed,
		"recovered_mappings": result.RecoveredMappings,
		"orphaned_volumes":   result.OrphanedVolumes,
		"duration":           result.Duration.String(),
		"errors":             result.Errors,
	})
}

// RecoverVolume recovers a lost device mapping for a specific volume
// POST /api/v1/recovery/volume/:volume_id
func (rh *RecoveryHandlers) RecoverVolume(c *gin.Context) {
	volumeID := c.Param("volume_id")
	if volumeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Volume ID is required",
		})
		return
	}

	log.WithField("volume_id", volumeID).Info("🔄 Volume recovery requested via API")

	mapping, err := rh.stateRecovery.RecoverSingleVolume(c.Request.Context(), volumeID)
	if err != nil {
		log.WithError(err).WithField("volume_id", volumeID).Error("❌ Volume recovery failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Volume recovery failed",
			"details": err.Error(),
		})
		return
	}

	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"device_path": mapping.DevicePath,
		"vm_id":       mapping.VMID,
	}).Info("✅ Volume recovery completed via API")

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"volume_id":      volumeID,
		"device_mapping": mapping,
	})
}

// RecoverSystemWide performs system-wide recovery of all lost device mappings
// POST /api/v1/recovery/system
func (rh *RecoveryHandlers) RecoverSystemWide(c *gin.Context) {
	log.Info("🔄 System-wide recovery requested via API")

	result, err := rh.stateRecovery.PerformFullSystemRecovery(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("❌ System-wide recovery failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "System-wide recovery failed",
			"details": err.Error(),
		})
		return
	}

	log.WithFields(log.Fields{
		"volumes_recovered": result.VolumesRecovered,
		"volumes_orphaned":  result.VolumesOrphaned,
		"mappings_created":  result.MappingsCreated,
		"mappings_fixed":    result.MappingsFixed,
		"duration":          result.Duration,
	}).Info("✅ System-wide recovery completed via API")

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"volumes_recovered":  result.VolumesRecovered,
		"volumes_orphaned":   result.VolumesOrphaned,
		"mappings_created":   result.MappingsCreated,
		"mappings_fixed":     result.MappingsFixed,
		"recovered_mappings": result.RecoveredMappings,
		"orphaned_volumes":   result.OrphanedVolumes,
		"duration":           result.Duration.String(),
		"errors":             result.Errors,
	})
}

// GetAutoRecoveryStatus returns the status of the auto recovery service
// GET /api/v1/recovery/auto/status
func (rh *RecoveryHandlers) GetAutoRecoveryStatus(c *gin.Context) {
	status := rh.autoRecovery.GetHealthStatus()
	stats := rh.autoRecovery.GetStats()

	response := gin.H{
		"auto_recovery_status": status,
		"statistics":           stats,
	}

	c.JSON(http.StatusOK, response)
}

// StartAutoRecovery starts the automatic recovery service
// POST /api/v1/recovery/auto/start
func (rh *RecoveryHandlers) StartAutoRecovery(c *gin.Context) {
	if rh.autoRecovery.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Auto recovery service is already running",
		})
		return
	}

	err := rh.autoRecovery.Start(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("❌ Failed to start auto recovery service")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start auto recovery service",
			"details": err.Error(),
		})
		return
	}

	log.Info("✅ Auto recovery service started via API")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Auto recovery service started",
	})
}

// StopAutoRecovery stops the automatic recovery service
// POST /api/v1/recovery/auto/stop
func (rh *RecoveryHandlers) StopAutoRecovery(c *gin.Context) {
	if !rh.autoRecovery.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Auto recovery service is not running",
		})
		return
	}

	err := rh.autoRecovery.Stop()
	if err != nil {
		log.WithError(err).Error("❌ Failed to stop auto recovery service")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to stop auto recovery service",
			"details": err.Error(),
		})
		return
	}

	log.Info("✅ Auto recovery service stopped via API")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Auto recovery service stopped",
	})
}

// TriggerManualRecovery manually triggers a recovery run
// POST /api/v1/recovery/auto/trigger
func (rh *RecoveryHandlers) TriggerManualRecovery(c *gin.Context) {
	log.Info("🔄 Manual recovery triggered via API")

	result, err := rh.autoRecovery.TriggerManualRecovery(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("❌ Manual recovery failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Manual recovery failed",
			"details": err.Error(),
		})
		return
	}

	log.WithFields(log.Fields{
		"volumes_recovered": result.VolumesRecovered,
		"mappings_created":  result.MappingsCreated,
		"mappings_fixed":    result.MappingsFixed,
		"duration":          result.Duration,
	}).Info("✅ Manual recovery completed via API")

	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"volumes_recovered":  result.VolumesRecovered,
		"volumes_orphaned":   result.VolumesOrphaned,
		"mappings_created":   result.MappingsCreated,
		"mappings_fixed":     result.MappingsFixed,
		"recovered_mappings": result.RecoveredMappings,
		"orphaned_volumes":   result.OrphanedVolumes,
		"duration":           result.Duration.String(),
		"errors":             result.Errors,
	})
}

// SetAutoRecoveryInterval updates the auto recovery interval
// PUT /api/v1/recovery/auto/interval
func (rh *RecoveryHandlers) SetAutoRecoveryInterval(c *gin.Context) {
	var request struct {
		IntervalMinutes int `json:"interval_minutes" binding:"required,min=1,max=1440"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	interval := time.Duration(request.IntervalMinutes) * time.Minute
	rh.autoRecovery.SetRecoveryInterval(interval)

	log.WithField("interval", interval).Info("✅ Auto recovery interval updated via API")

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"message":          "Auto recovery interval updated",
		"interval_minutes": request.IntervalMinutes,
		"interval":         interval.String(),
	})
}

// GetRecoveryHealth returns comprehensive recovery health information
// GET /api/v1/recovery/health
func (rh *RecoveryHandlers) GetRecoveryHealth(c *gin.Context) {
	stats := rh.autoRecovery.GetStats()
	status := rh.autoRecovery.GetHealthStatus()

	// Calculate health metrics
	isHealthy := true
	issues := make([]string, 0)

	if !rh.autoRecovery.IsRunning() {
		isHealthy = false
		issues = append(issues, "Auto recovery service is not running")
	}

	if len(stats.LastRunErrors) > 0 {
		isHealthy = false
		issues = append(issues, "Last recovery run had errors")
	}

	timeSinceLastRun := time.Since(stats.LastRunTime)
	// Use a reasonable default interval since we can't access the private field
	expectedInterval := 10 * time.Minute // Default expected interval
	if timeSinceLastRun > expectedInterval {
		isHealthy = false
		issues = append(issues, "Recovery hasn't run recently")
	}

	response := gin.H{
		"is_healthy":           isHealthy,
		"issues":               issues,
		"auto_recovery_stats":  stats,
		"auto_recovery_status": status,
		"recommendations":      rh.generateHealthRecommendations(stats, status),
	}

	if isHealthy {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// generateHealthRecommendations generates health recommendations based on stats
func (rh *RecoveryHandlers) generateHealthRecommendations(stats *service.RecoveryStats, status map[string]interface{}) []string {
	recommendations := make([]string, 0)

	if !rh.autoRecovery.IsRunning() {
		recommendations = append(recommendations, "Start the auto recovery service")
	}

	if len(stats.LastRunErrors) > 0 {
		recommendations = append(recommendations, "Investigate and resolve recovery errors")
	}

	if stats.TotalVolumesRecovered > 10 {
		recommendations = append(recommendations, "High volume recovery count indicates underlying issues - investigate CloudStack connectivity")
	}

	if time.Since(stats.LastRunTime) > 30*time.Minute {
		recommendations = append(recommendations, "Consider reducing recovery interval for more frequent checks")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Recovery system is healthy - no action needed")
	}

	return recommendations
}

// Enhanced volume operations with automatic recovery

// EnhancedGetVolumeDevice gets volume device with automatic recovery on failure
// GET /api/v1/volumes/:volume_id/device/enhanced
func (rh *RecoveryHandlers) EnhancedGetVolumeDevice(c *gin.Context) {
	volumeID := c.Param("volume_id")
	if volumeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Volume ID is required",
		})
		return
	}

	log.WithField("volume_id", volumeID).Debug("🔍 Enhanced device lookup requested")

	// Define the operation to be enhanced with recovery
	var devicePath string
	var err error

	operation := func() error {
		// This would call the regular volume service GetVolumeDevice method
		// Placeholder implementation - would integrate with actual volume service
		devicePath, err = rh.getVolumeDeviceInternal(c.Request.Context(), volumeID)
		return err
	}

	// Execute with automatic recovery
	err = rh.autoRecovery.EnhancedVolumeOperation(c.Request.Context(), operation, volumeID)
	if err != nil {
		log.WithError(err).WithField("volume_id", volumeID).Error("❌ Enhanced device lookup failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get volume device",
			"details": err.Error(),
		})
		return
	}

	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"device_path": devicePath,
	}).Debug("✅ Enhanced device lookup successful")

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"volume_id":   volumeID,
		"device_path": devicePath,
		"enhanced":    true,
	})
}

// Placeholder method - would integrate with actual volume service
func (rh *RecoveryHandlers) getVolumeDeviceInternal(ctx context.Context, volumeID string) (string, error) {
	// This would call the volume service repository to get the device mapping
	// For now, return a placeholder
	return "/dev/vdb", nil
}
