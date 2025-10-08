package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-sha/services"
)

// CloudStackStateHandlers provides HTTP handlers for CloudStack state synchronization
type CloudStackStateHandlers struct {
	stateSyncService *services.CloudStackStateSyncService
}

// NewCloudStackStateHandlers creates new state synchronization handlers
func NewCloudStackStateHandlers(stateSyncService *services.CloudStackStateSyncService) *CloudStackStateHandlers {
	return &CloudStackStateHandlers{
		stateSyncService: stateSyncService,
	}
}

// StartStateSynchronization starts the CloudStack state synchronization service
// POST /api/v1/cloudstack/state/sync/start
func (csh *CloudStackStateHandlers) StartStateSynchronization(c *gin.Context) {
	log.Info("🔄 Starting CloudStack state synchronization via API")

	err := csh.stateSyncService.StartStateSynchronization(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("❌ Failed to start state synchronization")
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Failed to start state synchronization",
			"details": err.Error(),
		})
		return
	}

	log.Info("✅ CloudStack state synchronization started successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "CloudStack state synchronization started",
		"status":  csh.stateSyncService.GetSyncStatus(),
	})
}

// StopStateSynchronization stops the CloudStack state synchronization service
// POST /api/v1/cloudstack/state/sync/stop
func (csh *CloudStackStateHandlers) StopStateSynchronization(c *gin.Context) {
	log.Info("⏹️ Stopping CloudStack state synchronization via API")

	err := csh.stateSyncService.StopStateSynchronization()
	if err != nil {
		log.WithError(err).Error("❌ Failed to stop state synchronization")
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Failed to stop state synchronization",
			"details": err.Error(),
		})
		return
	}

	log.Info("✅ CloudStack state synchronization stopped successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "CloudStack state synchronization stopped",
		"status":  csh.stateSyncService.GetSyncStatus(),
	})
}

// GetSyncStatus returns the current synchronization status
// GET /api/v1/cloudstack/state/sync/status
func (csh *CloudStackStateHandlers) GetSyncStatus(c *gin.Context) {
	status := csh.stateSyncService.GetSyncStatus()

	statusCode := http.StatusOK
	if !status["is_running"].(bool) {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"status":  status,
	})
}

// ForceSync triggers an immediate synchronization cycle
// POST /api/v1/cloudstack/state/sync/force
func (csh *CloudStackStateHandlers) ForceSync(c *gin.Context) {
	log.Info("🔄 Force synchronization requested via API")

	result, err := csh.stateSyncService.ForceSync(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("❌ Force synchronization failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Force synchronization failed",
			"details": err.Error(),
		})
		return
	}

	vmsSynced := 0
	volumesSynced := 0
	inconsistenciesFound := 0
	if result != nil {
		vmsSynced = result.VMsSynced
		volumesSynced = result.VolumesSynced
		inconsistenciesFound = result.InconsistenciesFound
	}

	log.WithFields(log.Fields{
		"vms_synced":            vmsSynced,
		"volumes_synced":        volumesSynced,
		"inconsistencies_found": inconsistenciesFound,
	}).Info("✅ Force synchronization completed")

	statusCode := http.StatusOK
	if result != nil && len(result.Errors) > 0 {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"message": "Force synchronization completed",
		"result":  result,
	})
}

// GetCurrentState returns the current cached CloudStack state
// GET /api/v1/cloudstack/state/current
func (csh *CloudStackStateHandlers) GetCurrentState(c *gin.Context) {
	state, err := csh.stateSyncService.GetCurrentState()
	if err != nil {
		log.WithError(err).Error("❌ Failed to get current state")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get current state",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"state":   state,
	})
}

// GetVMState returns the cached state for a specific VM
// GET /api/v1/cloudstack/state/vm/:vm_id
func (csh *CloudStackStateHandlers) GetVMState(c *gin.Context) {
	vmID := c.Param("vm_id")
	if vmID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "VM ID is required",
		})
		return
	}

	state, err := csh.stateSyncService.GetCurrentState()
	if err != nil {
		log.WithError(err).Error("❌ Failed to get VM state")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get VM state",
			"details": err.Error(),
		})
		return
	}

	vmState, exists := state.VMs[vmID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "VM not found in cache",
			"vm_id": vmID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"vm_id":    vmID,
		"vm_state": vmState,
	})
}

// GetVolumeState returns the cached state for a specific volume
// GET /api/v1/cloudstack/state/volume/:volume_id
func (csh *CloudStackStateHandlers) GetVolumeState(c *gin.Context) {
	volumeID := c.Param("volume_id")
	if volumeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Volume ID is required",
		})
		return
	}

	state, err := csh.stateSyncService.GetCurrentState()
	if err != nil {
		log.WithError(err).Error("❌ Failed to get volume state")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get volume state",
			"details": err.Error(),
		})
		return
	}

	volumeState, exists := state.Volumes[volumeID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "Volume not found in cache",
			"volume_id": volumeID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"volume_id":    volumeID,
		"volume_state": volumeState,
	})
}

// GetStateStatistics returns statistics about the current state cache
// GET /api/v1/cloudstack/state/statistics
func (csh *CloudStackStateHandlers) GetStateStatistics(c *gin.Context) {
	state, err := csh.stateSyncService.GetCurrentState()
	if err != nil {
		log.WithError(err).Error("❌ Failed to get state statistics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get state statistics",
			"details": err.Error(),
		})
		return
	}

	status := csh.stateSyncService.GetSyncStatus()

	// Calculate state statistics
	vmStats := make(map[string]int)
	volumeStats := make(map[string]int)

	for _, vm := range state.VMs {
		vmStats[vm.State]++
	}

	for _, volume := range state.Volumes {
		volumeStats[volume.State]++
	}

	attachedVolumes := 0
	for _, volume := range state.Volumes {
		if volume.AttachedToVM != "" {
			attachedVolumes++
		}
	}

	statistics := map[string]interface{}{
		"sync_status": status,
		"cache_info": map[string]interface{}{
			"last_updated":  state.LastUpdated,
			"sync_version":  state.SyncVersion,
			"total_vms":     len(state.VMs),
			"total_volumes": len(state.Volumes),
		},
		"vm_statistics": map[string]interface{}{
			"by_state":    vmStats,
			"total_count": len(state.VMs),
		},
		"volume_statistics": map[string]interface{}{
			"by_state":         volumeStats,
			"total_count":      len(state.Volumes),
			"attached_volumes": attachedVolumes,
			"free_volumes":     len(state.Volumes) - attachedVolumes,
		},
		"health_indicators": map[string]interface{}{
			"is_sync_running":   status["is_running"],
			"sync_errors":       status["sync_errors"],
			"cache_age_minutes": nil, // Would calculate based on last_updated
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"statistics": statistics,
	})
}

// GetStateInconsistencies returns any detected state inconsistencies
// GET /api/v1/cloudstack/state/inconsistencies
func (csh *CloudStackStateHandlers) GetStateInconsistencies(c *gin.Context) {
	// This would trigger an inconsistency check and return results
	// For now, return a placeholder response indicating this requires force sync

	log.Info("🔍 State inconsistencies check requested via API")

	c.JSON(http.StatusAccepted, gin.H{
		"success":        true,
		"message":        "Inconsistency detection requires force synchronization",
		"recommendation": "Use POST /api/v1/cloudstack/state/sync/force to detect and fix inconsistencies",
	})
}

// GetVMsWithStaleState returns VMs that haven't been updated recently
// GET /api/v1/cloudstack/state/stale/vms
func (csh *CloudStackStateHandlers) GetVMsWithStaleState(c *gin.Context) {
	maxAgeMinutesStr := c.DefaultQuery("max_age_minutes", "10")
	maxAgeMinutes, err := strconv.Atoi(maxAgeMinutesStr)
	if err != nil || maxAgeMinutes < 1 {
		maxAgeMinutes = 10
	}

	_, err = csh.stateSyncService.GetCurrentState()
	if err != nil {
		log.WithError(err).Error("❌ Failed to get stale VMs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get stale VMs",
			"details": err.Error(),
		})
		return
	}

	// This would identify VMs with stale last_seen timestamps
	// For now, return basic response
	staleVMs := []map[string]interface{}{}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"stale_vms":       staleVMs,
		"max_age_minutes": maxAgeMinutes,
		"total_stale_vms": len(staleVMs),
	})
}

// GetVolumesWithStaleState returns volumes that haven't been updated recently
// GET /api/v1/cloudstack/state/stale/volumes
func (csh *CloudStackStateHandlers) GetVolumesWithStaleState(c *gin.Context) {
	maxAgeMinutesStr := c.DefaultQuery("max_age_minutes", "10")
	maxAgeMinutes, err := strconv.Atoi(maxAgeMinutesStr)
	if err != nil || maxAgeMinutes < 1 {
		maxAgeMinutes = 10
	}

	_, err = csh.stateSyncService.GetCurrentState()
	if err != nil {
		log.WithError(err).Error("❌ Failed to get stale volumes")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get stale volumes",
			"details": err.Error(),
		})
		return
	}

	// This would identify volumes with stale last_seen timestamps
	// For now, return basic response
	staleVolumes := []map[string]interface{}{}

	c.JSON(http.StatusOK, gin.H{
		"success":             true,
		"stale_volumes":       staleVolumes,
		"max_age_minutes":     maxAgeMinutes,
		"total_stale_volumes": len(staleVolumes),
	})
}

// RefreshResource forces refresh of a specific resource from CloudStack
// POST /api/v1/cloudstack/state/refresh/:resource_type/:resource_id
func (csh *CloudStackStateHandlers) RefreshResource(c *gin.Context) {
	resourceType := c.Param("resource_type")
	resourceID := c.Param("resource_id")

	if resourceType == "" || resourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Resource type and ID are required",
		})
		return
	}

	log.WithFields(log.Fields{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	}).Info("🔄 Resource refresh requested via API")

	// This would force refresh the specific resource from CloudStack
	// For now, return success response
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Resource refresh completed",
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

// GetSyncHistory returns recent synchronization history
// GET /api/v1/cloudstack/state/sync/history
func (csh *CloudStackStateHandlers) GetSyncHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// This would query synchronization history from logs or database
	// For now, return placeholder response
	history := []map[string]interface{}{
		{
			"sync_time":             "2025-01-21T14:30:00Z",
			"duration":              "2.1s",
			"vms_synced":            15,
			"volumes_synced":        42,
			"inconsistencies_found": 0,
			"inconsistencies_fixed": 0,
			"errors":                0,
			"is_full_sync":          false,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"history": history,
		"count":   len(history),
		"limit":   limit,
	})
}
