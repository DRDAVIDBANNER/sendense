package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vexxhost/migratekit-volume-daemon/models"
	"github.com/vexxhost/migratekit-volume-daemon/service"
)

// Handler handles HTTP requests for volume management
type Handler struct {
	volumeService  service.VolumeManagementService
	cleanupService *service.NBDCleanupService
}

// NewHandler creates a new API handler
func NewHandler(volumeService service.VolumeManagementService, cleanupService *service.NBDCleanupService) *Handler {
	return &Handler{
		volumeService:  volumeService,
		cleanupService: cleanupService,
	}
}

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine, volumeService service.VolumeManagementService, cleanupService *service.NBDCleanupService) {
	handler := NewHandler(volumeService, cleanupService)

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Volume operations
		v1.POST("/volumes", handler.CreateVolume)
		v1.POST("/volumes/:id/attach", handler.AttachVolume)
		v1.POST("/volumes/:id/attach-root", handler.AttachVolumeAsRoot)
		v1.POST("/volumes/:id/detach", handler.DetachVolume)
		v1.DELETE("/volumes/:id", handler.DeleteVolume)

		// Cleanup operations
		v1.POST("/cleanup/test-failover", handler.CleanupTestFailover)

		// Snapshot tracking operations
		v1.POST("/snapshots/track", handler.TrackVolumeSnapshot)
		v1.GET("/snapshots/vm/:vm_context_id", handler.GetVMSnapshots)
		v1.DELETE("/snapshots/vm/:vm_context_id", handler.ClearVMSnapshots)
		v1.PUT("/snapshots/:volume_uuid", handler.UpdateVolumeSnapshot)

		// Volume status queries
		v1.GET("/volumes/:id", handler.GetVolumeStatus)
		v1.GET("/volumes/:id/device", handler.GetDeviceMapping)
		v1.GET("/devices/:path/volume", handler.GetVolumeForDevice)
		v1.GET("/vms/:id/volumes", handler.ListVolumesForVM)

		// Operation tracking
		v1.GET("/operations/:id", handler.GetOperation)
		v1.GET("/operations", handler.ListOperations)

		// NBD Export Management (NEW)
		v1.POST("/exports", handler.CreateNBDExport)
		v1.DELETE("/exports/:volume_id", handler.DeleteNBDExport)
		v1.GET("/exports/:volume_id", handler.GetNBDExport)
		v1.GET("/exports", handler.ListNBDExports)
		v1.POST("/exports/validate", handler.ValidateNBDExports)

		// NBD Export Cleanup Service
		v1.POST("/exports/cleanup", handler.CleanupOrphanedNBDExports)
		v1.GET("/exports/orphaned/count", handler.GetOrphanedExportsCount)
		v1.POST("/exports/cleanup/age", handler.CleanupNBDExportsByAge)

		// Administrative
		v1.POST("/admin/force-sync", handler.ForceSync)
		v1.GET("/health", handler.GetHealth)
		v1.GET("/metrics", handler.GetMetrics)
	}
}

// CreateVolume handles POST /api/v1/volumes
func (h *Handler) CreateVolume(c *gin.Context) {
	var req models.CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operation, err := h.volumeService.CreateVolume(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, operation)
}

// AttachVolume handles POST /api/v1/volumes/:id/attach
func (h *Handler) AttachVolume(c *gin.Context) {
	volumeID := c.Param("id")

	var req models.AttachVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operation, err := h.volumeService.AttachVolume(c.Request.Context(), volumeID, req.VMID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, operation)
}

// AttachVolumeAsRoot handles POST /api/v1/volumes/:id/attach-root
func (h *Handler) AttachVolumeAsRoot(c *gin.Context) {
	volumeID := c.Param("id")

	var req models.AttachVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operation, err := h.volumeService.AttachVolumeAsRoot(c.Request.Context(), volumeID, req.VMID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, operation)
}

// DetachVolume handles POST /api/v1/volumes/:id/detach
func (h *Handler) DetachVolume(c *gin.Context) {
	volumeID := c.Param("id")

	operation, err := h.volumeService.DetachVolume(c.Request.Context(), volumeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, operation)
}

// DeleteVolume handles DELETE /api/v1/volumes/:id
func (h *Handler) DeleteVolume(c *gin.Context) {
	volumeID := c.Param("id")

	operation, err := h.volumeService.DeleteVolume(c.Request.Context(), volumeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, operation)
}

// GetVolumeStatus handles GET /api/v1/volumes/:id
func (h *Handler) GetVolumeStatus(c *gin.Context) {
	volumeID := c.Param("id")

	status, err := h.volumeService.GetVolumeStatus(c.Request.Context(), volumeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetDeviceMapping handles GET /api/v1/volumes/:id/device
func (h *Handler) GetDeviceMapping(c *gin.Context) {
	volumeID := c.Param("id")

	mapping, err := h.volumeService.GetDeviceMapping(c.Request.Context(), volumeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, mapping)
}

// GetVolumeForDevice handles GET /api/v1/devices/:path/volume
func (h *Handler) GetVolumeForDevice(c *gin.Context) {
	devicePath := "/" + c.Param("path") // Re-add leading slash

	mapping, err := h.volumeService.GetVolumeForDevice(c.Request.Context(), devicePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, mapping)
}

// ListVolumesForVM handles GET /api/v1/vms/:id/volumes
func (h *Handler) ListVolumesForVM(c *gin.Context) {
	vmID := c.Param("id")

	volumes, err := h.volumeService.ListVolumesForVM(c.Request.Context(), vmID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, volumes)
}

// GetOperation handles GET /api/v1/operations/:id
func (h *Handler) GetOperation(c *gin.Context) {
	operationID := c.Param("id")

	operation, err := h.volumeService.GetOperation(c.Request.Context(), operationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, operation)
}

// ListOperations handles GET /api/v1/operations
func (h *Handler) ListOperations(c *gin.Context) {
	// Parse query parameters for filtering
	filter := models.OperationFilter{}

	if typeParam := c.Query("type"); typeParam != "" {
		opType := models.VolumeOperationType(typeParam)
		filter.Type = &opType
	}

	if statusParam := c.Query("status"); statusParam != "" {
		status := models.OperationStatus(statusParam)
		filter.Status = &status
	}

	if volumeID := c.Query("volume_id"); volumeID != "" {
		filter.VolumeID = &volumeID
	}

	if vmID := c.Query("vm_id"); vmID != "" {
		filter.VMID = &vmID
	}

	if limitParam := c.Query("limit"); limitParam != "" {
		if limit, err := strconv.Atoi(limitParam); err == nil {
			filter.Limit = limit
		}
	}

	operations, err := h.volumeService.ListOperations(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, operations)
}

// ForceSync handles POST /api/v1/admin/force-sync
func (h *Handler) ForceSync(c *gin.Context) {
	err := h.volumeService.ForceSync(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Force sync completed successfully",
		"timestamp": time.Now(),
	})
}

// CleanupTestFailover handles POST /api/v1/cleanup/test-failover
func (h *Handler) CleanupTestFailover(c *gin.Context) {
	var req models.CleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operation, err := h.volumeService.CleanupTestFailover(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, operation)
}

// GetHealth handles GET /api/v1/health
func (h *Handler) GetHealth(c *gin.Context) {
	health, err := h.volumeService.GetHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetMetrics handles GET /api/v1/metrics
func (h *Handler) GetMetrics(c *gin.Context) {
	metrics, err := h.volumeService.GetMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// NBD Export Management Handlers

// CreateNBDExport handles POST /api/v1/exports
func (h *Handler) CreateNBDExport(c *gin.Context) {
	var req models.NBDExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exportInfo, err := h.volumeService.CreateNBDExport(c.Request.Context(), req.VolumeID, req.VMName, req.VMID, req.DiskNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, exportInfo)
}

// DeleteNBDExport handles DELETE /api/v1/exports/:volume_id
func (h *Handler) DeleteNBDExport(c *gin.Context) {
	volumeID := c.Param("volume_id")
	if volumeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "volume_id parameter is required"})
		return
	}

	err := h.volumeService.DeleteNBDExport(c.Request.Context(), volumeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "NBD export deleted successfully"})
}

// GetNBDExport handles GET /api/v1/exports/:volume_id
func (h *Handler) GetNBDExport(c *gin.Context) {
	volumeID := c.Param("volume_id")
	if volumeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "volume_id parameter is required"})
		return
	}

	exportInfo, err := h.volumeService.GetNBDExport(c.Request.Context(), volumeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, exportInfo)
}

// ListNBDExports handles GET /api/v1/exports
func (h *Handler) ListNBDExports(c *gin.Context) {
	var filter models.NBDExportFilter

	// Parse query parameters
	if volumeID := c.Query("volume_id"); volumeID != "" {
		filter.VolumeID = &volumeID
	}

	if status := c.Query("status"); status != "" {
		exportStatus := models.NBDExportStatus(status)
		filter.Status = &exportStatus
	}

	if vmName := c.Query("vm_name"); vmName != "" {
		filter.VMName = &vmName
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	exports, err := h.volumeService.ListNBDExports(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exports": exports,
		"count":   len(exports),
	})
}

// ValidateNBDExports handles POST /api/v1/exports/validate
func (h *Handler) ValidateNBDExports(c *gin.Context) {
	err := h.volumeService.ValidateNBDExports(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "NBD exports validation completed successfully"})
}

// CleanupOrphanedNBDExports handles POST /api/v1/exports/cleanup
func (h *Handler) CleanupOrphanedNBDExports(c *gin.Context) {
	var req struct {
		DryRun bool `json:"dry_run"`
	}

	// Default to dry run if not specified
	req.DryRun = true
	if err := c.ShouldBindJSON(&req); err != nil {
		// Ignore binding errors for optional parameters
	}

	result, err := h.cleanupService.PerformComprehensiveCleanup(c.Request.Context(), req.DryRun)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetOrphanedExportsCount handles GET /api/v1/exports/orphaned/count
func (h *Handler) GetOrphanedExportsCount(c *gin.Context) {
	count, err := h.cleanupService.GetOrphanedExportsCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orphaned_exports_count": count,
		"timestamp":              time.Now(),
	})
}

// CleanupNBDExportsByAge handles POST /api/v1/exports/cleanup/age
func (h *Handler) CleanupNBDExportsByAge(c *gin.Context) {
	var req struct {
		MaxAgeHours int  `json:"max_age_hours" binding:"required"`
		DryRun      bool `json:"dry_run"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	maxAge := time.Duration(req.MaxAgeHours) * time.Hour
	result, err := h.cleanupService.CleanupExportsByAge(c.Request.Context(), maxAge, req.DryRun)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// TrackVolumeSnapshot handles POST /api/v1/snapshots/track
// Records snapshot information for a volume in device_mappings
func (h *Handler) TrackVolumeSnapshot(c *gin.Context) {
	var req models.TrackSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.volumeService.TrackVolumeSnapshot(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Volume snapshot tracked successfully",
		"volume_uuid":   req.VolumeUUID,
		"snapshot_id":   req.SnapshotID,
		"vm_context_id": req.VMContextID,
	})
}

// GetVMSnapshots handles GET /api/v1/snapshots/vm/:vm_context_id
// Returns all snapshot information for volumes belonging to a VM
func (h *Handler) GetVMSnapshots(c *gin.Context) {
	vmContextID := c.Param("vm_context_id")

	snapshots, err := h.volumeService.GetVMVolumeSnapshots(c.Request.Context(), vmContextID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vm_context_id": vmContextID,
		"snapshots":     snapshots,
		"count":         len(snapshots),
	})
}

// ClearVMSnapshots handles DELETE /api/v1/snapshots/vm/:vm_context_id
// Clears all snapshot tracking information for a VM
func (h *Handler) ClearVMSnapshots(c *gin.Context) {
	vmContextID := c.Param("vm_context_id")

	count, err := h.volumeService.ClearVMVolumeSnapshots(c.Request.Context(), vmContextID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "VM volume snapshots cleared successfully",
		"vm_context_id":     vmContextID,
		"snapshots_cleared": count,
	})
}

// UpdateVolumeSnapshot handles PUT /api/v1/snapshots/:volume_uuid
// Updates snapshot information for a specific volume
func (h *Handler) UpdateVolumeSnapshot(c *gin.Context) {
	volumeUUID := c.Param("volume_uuid")

	var req models.UpdateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.VolumeUUID = volumeUUID // Ensure consistency

	err := h.volumeService.UpdateVolumeSnapshot(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Volume snapshot updated successfully",
		"volume_uuid": volumeUUID,
	})
}
