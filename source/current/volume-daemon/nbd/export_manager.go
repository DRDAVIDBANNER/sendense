// Package nbd provides NBD export management integrated with Volume Daemon
package nbd

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/models"
)

// ExportManager handles NBD export lifecycle integrated with volume operations
type ExportManager struct {
	configManager *ConfigManager
	repository    ExportRepository
}

// ExportInfo represents NBD export information
type ExportInfo struct {
	ID         string            `json:"id"`
	VolumeID   string            `json:"volume_id"`
	VMDiskID   *int              `json:"vm_disk_id,omitempty"` // Correlation to vm_disks.id
	ExportName string            `json:"export_name"`
	DevicePath string            `json:"device_path"`
	Port       int               `json:"port"`
	Status     ExportStatus      `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Metadata   map[string]string `json:"metadata"`
}

// ExportStatus represents the status of an NBD export
type ExportStatus string

const (
	ExportStatusPending ExportStatus = "pending"
	ExportStatusActive  ExportStatus = "active"
	ExportStatusFailed  ExportStatus = "failed"
)

// ExportRequest represents a request to create an NBD export
type ExportRequest struct {
	VolumeID   string            `json:"volume_id"`
	VMName     string            `json:"vm_name"`
	VMID       string            `json:"vm_id"`
	VMDiskID   *int              `json:"vm_disk_id,omitempty"` // Correlation to vm_disks.id
	DiskNumber int               `json:"disk_number"`
	DevicePath string            `json:"device_path"`
	ReadOnly   bool              `json:"read_only"`
	Metadata   map[string]string `json:"metadata"`
}

// ExportRepository defines the interface for NBD export persistence
type ExportRepository interface {
	CreateExport(ctx context.Context, export *ExportInfo) error
	UpdateExport(ctx context.Context, export *ExportInfo) error
	DeleteExport(ctx context.Context, exportID string) error
	GetExport(ctx context.Context, exportID string) (*ExportInfo, error)
	GetExportByVolumeID(ctx context.Context, volumeID string) (*ExportInfo, error)
	ListExports(ctx context.Context, filter ExportFilter) ([]*ExportInfo, error)
}

// ExportFilter represents filters for listing exports
type ExportFilter struct {
	VolumeID *string       `json:"volume_id,omitempty"`
	Status   *ExportStatus `json:"status,omitempty"`
	VMName   *string       `json:"vm_name,omitempty"`
	Limit    int           `json:"limit,omitempty"`
}

// NewExportManager creates a new NBD export manager
func NewExportManager(configPath, confDir string, repository ExportRepository) *ExportManager {
	return &ExportManager{
		configManager: NewConfigManager(configPath, confDir),
		repository:    repository,
	}
}

// GetConfigManager returns the NBD configuration manager (for atomic operations)
func (em *ExportManager) GetConfigManager() *ConfigManager {
	return em.configManager
}

// CreateExport creates a new NBD export for a volume
func (em *ExportManager) CreateExport(ctx context.Context, req *ExportRequest) (*ExportInfo, error) {
	log.WithFields(log.Fields{
		"volume_id":   req.VolumeID,
		"vm_name":     req.VMName,
		"vm_id":       req.VMID,
		"disk_number": req.DiskNumber,
		"device_path": req.DevicePath,
	}).Info("🔗 Creating NBD export via Volume Daemon")

	// Generate export name using volume ID for uniqueness (not VM ID)
	// Using volume ID ensures each export is unique, even when multiple volumes
	// from different VMs are attached to the same target VM (OMA)
	exportName := fmt.Sprintf("migration-vol-%s", req.VolumeID)

	// Check if export already exists
	if exists, err := em.configManager.ExportExists(exportName); err != nil {
		return nil, fmt.Errorf("failed to check if export exists: %w", err)
	} else if exists {
		// Return existing export info
		return em.getExistingExport(ctx, req.VolumeID, exportName)
	}

	// Create export info record
	exportInfo := &ExportInfo{
		ID:         generateExportID(),
		VolumeID:   req.VolumeID,
		VMDiskID:   req.VMDiskID, // Include vm_disk_id correlation
		ExportName: exportName,
		DevicePath: req.DevicePath,
		Port:       10809, // Shared NBD port
		Status:     ExportStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata:   req.Metadata,
	}

	// Enhance metadata with VM information
	if exportInfo.Metadata == nil {
		exportInfo.Metadata = make(map[string]string)
	}
	exportInfo.Metadata["vm_name"] = req.VMName
	exportInfo.Metadata["vm_id"] = req.VMID
	exportInfo.Metadata["disk_number"] = fmt.Sprintf("%d", req.DiskNumber)
	exportInfo.Metadata["created_by"] = "volume-daemon"

	// Create database record first (for rollback capability)
	if err := em.repository.CreateExport(ctx, exportInfo); err != nil {
		return nil, fmt.Errorf("failed to create export database record: %w", err)
	}

	// Create NBD configuration export
	configExport := &Export{
		Name:       exportName,
		DevicePath: req.DevicePath,
		ReadOnly:   req.ReadOnly,
		Metadata:   exportInfo.Metadata,
	}

	if err := em.configManager.AddExport(configExport); err != nil {
		// Rollback database record on configuration failure
		if deleteErr := em.repository.DeleteExport(ctx, exportInfo.ID); deleteErr != nil {
			log.WithError(deleteErr).Error("Failed to rollback export database record after config failure")
		}

		exportInfo.Status = ExportStatusFailed
		if updateErr := em.repository.UpdateExport(ctx, exportInfo); updateErr != nil {
			log.WithError(updateErr).Error("Failed to update export status to failed")
		}

		return nil, fmt.Errorf("failed to add export to NBD configuration: %w", err)
	}

	// Update status to active
	exportInfo.Status = ExportStatusActive
	exportInfo.UpdatedAt = time.Now()

	if err := em.repository.UpdateExport(ctx, exportInfo); err != nil {
		log.WithError(err).Error("Failed to update export status to active - export created but status not updated")
	}

	log.WithFields(log.Fields{
		"export_id":   exportInfo.ID,
		"export_name": exportInfo.ExportName,
		"device_path": exportInfo.DevicePath,
		"port":        exportInfo.Port,
	}).Info("✅ NBD export created successfully via Volume Daemon")

	return exportInfo, nil
}

// DeleteExport removes an NBD export
func (em *ExportManager) DeleteExport(ctx context.Context, volumeID string) error {
	log.WithField("volume_id", volumeID).Info("🗑️ Deleting NBD export via Volume Daemon")

	// Get existing export
	exportInfo, err := em.repository.GetExportByVolumeID(ctx, volumeID)
	if err != nil {
		log.WithField("volume_id", volumeID).Warn("NBD export not found in database - already deleted")
		return nil // Idempotent operation
	}

	// Remove from NBD configuration
	if err := em.configManager.RemoveExport(exportInfo.ExportName); err != nil {
		log.WithError(err).Warn("Failed to remove export from NBD configuration - continuing with database cleanup")
	}

	// Remove database record using volume_id (what the repository expects)
	if err := em.repository.DeleteExport(ctx, volumeID); err != nil {
		return fmt.Errorf("failed to delete export database record: %w", err)
	}

	log.WithFields(log.Fields{
		"export_id":   exportInfo.ID,
		"export_name": exportInfo.ExportName,
		"volume_id":   volumeID,
	}).Info("✅ NBD export deleted successfully via Volume Daemon")

	return nil
}

// GetExport retrieves export information by volume ID
func (em *ExportManager) GetExport(ctx context.Context, volumeID string) (*ExportInfo, error) {
	return em.repository.GetExportByVolumeID(ctx, volumeID)
}

// ListExports lists all exports with optional filtering
func (em *ExportManager) ListExports(ctx context.Context, filter ExportFilter) ([]*ExportInfo, error) {
	return em.repository.ListExports(ctx, filter)
}

// ValidateExports checks that all database exports exist in NBD configuration
func (em *ExportManager) ValidateExports(ctx context.Context) error {
	log.Info("🔍 Validating NBD exports consistency between database and configuration")

	// Get all exports from database
	dbExports, err := em.repository.ListExports(ctx, ExportFilter{})
	if err != nil {
		return fmt.Errorf("failed to list database exports: %w", err)
	}

	// Get all exports from configuration
	configExports, err := em.configManager.ListExports()
	if err != nil {
		return fmt.Errorf("failed to list configuration exports: %w", err)
	}

	// Create maps for quick lookup
	dbExportMap := make(map[string]*ExportInfo)
	for _, export := range dbExports {
		dbExportMap[export.ExportName] = export
	}

	configExportMap := make(map[string]*Export)
	for _, export := range configExports {
		configExportMap[export.Name] = export
	}

	// Check for exports in database but not in configuration
	for exportName, dbExport := range dbExportMap {
		if _, exists := configExportMap[exportName]; !exists {
			log.WithFields(log.Fields{
				"export_name": exportName,
				"volume_id":   dbExport.VolumeID,
			}).Warn("Export exists in database but not in NBD configuration")
		}
	}

	// Check for exports in configuration but not in database
	for exportName, configExport := range configExportMap {
		// Skip dummy exports - these are maintained by the config manager
		if exportName == "dummy" || configExport.DevicePath == "/dev/null" {
			continue
		}
		if _, exists := dbExportMap[exportName]; !exists {
			log.WithFields(log.Fields{
				"export_name": exportName,
				"device_path": configExport.DevicePath,
			}).Warn("Export exists in NBD configuration but not in database")
		}
	}

	log.WithFields(log.Fields{
		"database_exports":      len(dbExports),
		"configuration_exports": len(configExports),
	}).Info("✅ NBD export validation completed")

	return nil
}

// CleanupOrphanedExports removes exports that no longer have valid volumes
func (em *ExportManager) CleanupOrphanedExports(ctx context.Context, activeVolumeIDs []string) error {
	log.Info("🧹 Cleaning up orphaned NBD exports")

	// Create map of active volume IDs for quick lookup
	activeVolumes := make(map[string]bool)
	for _, volumeID := range activeVolumeIDs {
		activeVolumes[volumeID] = true
	}

	// Get all exports
	allExports, err := em.repository.ListExports(ctx, ExportFilter{})
	if err != nil {
		return fmt.Errorf("failed to list exports for cleanup: %w", err)
	}

	orphanedCount := 0
	for _, export := range allExports {
		if !activeVolumes[export.VolumeID] {
			log.WithFields(log.Fields{
				"export_name": export.ExportName,
				"volume_id":   export.VolumeID,
			}).Info("Removing orphaned NBD export")

			if err := em.DeleteExport(ctx, export.VolumeID); err != nil {
				log.WithError(err).WithField("volume_id", export.VolumeID).Error("Failed to delete orphaned export")
			} else {
				orphanedCount++
			}
		}
	}

	log.WithField("orphaned_count", orphanedCount).Info("✅ Orphaned NBD export cleanup completed")
	return nil
}

// GetHealth returns the health status of the NBD export system
func (em *ExportManager) GetHealth(ctx context.Context) (*models.HealthStatus, error) {
	health := &models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   make(map[string]string),
	}

	// Check NBD configuration validity
	if err := em.configManager.ValidateConfig(); err != nil {
		health.Status = "unhealthy"
		health.Details["nbd_config_error"] = err.Error()
	}

	// Check NBD server status
	if pid, err := em.configManager.GetNBDServerPID(); err != nil {
		health.Status = "unhealthy"
		health.Details["nbd_server_error"] = err.Error()
	} else {
		health.Details["nbd_server_pid"] = fmt.Sprintf("%d", pid)
	}

	// Get export statistics
	if exports, err := em.repository.ListExports(ctx, ExportFilter{}); err != nil {
		health.Details["export_count_error"] = err.Error()
	} else {
		health.Details["total_exports"] = fmt.Sprintf("%d", len(exports))

		// Count exports by status
		statusCounts := make(map[ExportStatus]int)
		for _, export := range exports {
			statusCounts[export.Status]++
		}

		// Convert to string representation
		for status, count := range statusCounts {
			health.Details[fmt.Sprintf("exports_%s", string(status))] = fmt.Sprintf("%d", count)
		}
	}

	health.Details["implementation_status"] = "volume_daemon_integrated"
	return health, nil
}

// Private helper methods

func (em *ExportManager) getExistingExport(ctx context.Context, volumeID, exportName string) (*ExportInfo, error) {
	export, err := em.repository.GetExportByVolumeID(ctx, volumeID)
	if err != nil {
		return nil, fmt.Errorf("export exists in configuration but not in database: %s", exportName)
	}

	log.WithFields(log.Fields{
		"export_name": exportName,
		"volume_id":   volumeID,
	}).Info("✅ NBD export already exists - returning existing")

	return export, nil
}

func generateExportID() string {
	return fmt.Sprintf("nbd-export-%d", time.Now().UnixNano())
}

// ExportNameFromVMInfo generates a consistent export name from VM information
func ExportNameFromVMInfo(vmID string, diskNumber int) string {
	return fmt.Sprintf("migration-vm-%s-disk%d", vmID, diskNumber)
}

// ParseExportName extracts VM ID and disk number from export name
func ParseExportName(exportName string) (vmID string, diskNumber int, err error) {
	// Format: migration-vm-{vmID}-disk{number}
	if !strings.HasPrefix(exportName, "migration-vm-") {
		return "", 0, fmt.Errorf("invalid export name format: %s", exportName)
	}

	parts := strings.Split(exportName, "-")
	if len(parts) < 4 || !strings.HasPrefix(parts[len(parts)-1], "disk") {
		return "", 0, fmt.Errorf("invalid export name format: %s", exportName)
	}

	// Extract VM ID (everything between "migration-vm-" and "-disk{number}")
	vmIDParts := parts[2 : len(parts)-1]
	vmID = strings.Join(vmIDParts, "-")

	// Extract disk number
	diskPart := parts[len(parts)-1]
	if _, err := fmt.Sscanf(diskPart, "disk%d", &diskNumber); err != nil {
		return "", 0, fmt.Errorf("invalid disk number in export name: %s", exportName)
	}

	return vmID, diskNumber, nil
}
