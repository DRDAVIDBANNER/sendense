// Package database provides repository layer for OMA database operations
// Following project rules: modular design, clean interfaces, small focused functions
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/vexxhost/migratekit-oma/ossea"
)

// OSSEAConfigRepository handles OSSEA configuration database operations
// Follows project rules: clean interfaces, no monster code
type OSSEAConfigRepository struct {
	db                *gorm.DB
	encryptionService CredentialEncryptor
}

// CredentialEncryptor defines the interface for credential encryption/decryption
// This allows for dependency injection and testing
type CredentialEncryptor interface {
	EncryptPassword(password string) (string, error)
	DecryptPassword(encryptedPassword string) (string, error)
}

// NewOSSEAConfigRepository creates a new OSSEA config repository
func NewOSSEAConfigRepository(conn Connection) *OSSEAConfigRepository {
	return &OSSEAConfigRepository{
		db:                conn.GetGormDB(),
		encryptionService: nil, // Will be set via SetEncryptionService
	}
}

// SetEncryptionService sets the credential encryption service
// This allows for optional encryption - if not set, credentials stored as-is
func (r *OSSEAConfigRepository) SetEncryptionService(service CredentialEncryptor) {
	r.encryptionService = service
	log.Info("Credential encryption enabled for OSSEA config repository")
}

// Create saves a new OSSEA configuration to the database
func (r *OSSEAConfigRepository) Create(config *OSSEAConfig) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	// Validate configuration before saving
	if err := r.ValidateConfig(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	log.WithField("config_name", config.Name).Info("Creating/updating OSSEA configuration in database")

	// Encrypt credentials if encryption service is available
	encryptedConfig := *config // Create a copy to avoid modifying original
	if r.encryptionService != nil {
		if err := r.encryptCredentials(&encryptedConfig); err != nil {
			log.WithError(err).Error("Failed to encrypt credentials")
			return fmt.Errorf("failed to encrypt credentials: %w", err)
		}
		log.Debug("Credentials encrypted before database storage")
	} else {
		log.Warn("Encryption service not available - credentials will be stored in plaintext")
	}

	// Start a transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Deactivate existing configurations with the same name
	if err := tx.Model(&OSSEAConfig{}).Where("name = ?", encryptedConfig.Name).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		log.WithError(err).Error("Failed to deactivate existing configurations")
		return fmt.Errorf("failed to deactivate existing configurations: %w", err)
	}

	// Create new configuration
	encryptedConfig.IsActive = true
	if err := tx.Create(&encryptedConfig).Error; err != nil {
		tx.Rollback()
		log.WithError(err).Error("Failed to create OSSEA configuration")
		return fmt.Errorf("failed to create OSSEA configuration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.WithError(err).Error("Failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.WithFields(log.Fields{
		"config_id":   encryptedConfig.ID,
		"config_name": encryptedConfig.Name,
	}).Info("OSSEA configuration created/updated successfully")

	// Update the original config with the generated ID
	config.ID = encryptedConfig.ID
	config.CreatedAt = encryptedConfig.CreatedAt
	config.UpdatedAt = encryptedConfig.UpdatedAt

	return nil
}

// GetByID retrieves an OSSEA configuration by ID
func (r *OSSEAConfigRepository) GetByID(id int) (*OSSEAConfig, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var config OSSEAConfig
	if err := r.db.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("OSSEA configuration with ID %d not found", id)
		}
		log.WithError(err).WithField("config_id", id).Error("Failed to retrieve OSSEA configuration")
		return nil, fmt.Errorf("failed to retrieve OSSEA configuration: %w", err)
	}

	// Decrypt credentials if encryption service is available
	if r.encryptionService != nil {
		if err := r.decryptCredentials(&config); err != nil {
			log.WithError(err).Warn("Failed to decrypt credentials - returning encrypted values")
			// Don't fail the entire operation, but log the error
		}
	}

	return &config, nil
}

// GetAll retrieves all OSSEA configurations
func (r *OSSEAConfigRepository) GetAll() ([]OSSEAConfig, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var configs []OSSEAConfig
	if err := r.db.Where("is_active = ?", true).Find(&configs).Error; err != nil {
		log.WithError(err).Error("Failed to retrieve OSSEA configurations")
		return nil, fmt.Errorf("failed to retrieve OSSEA configurations: %w", err)
	}

	// Decrypt credentials for all configurations
	if r.encryptionService != nil {
		for i := range configs {
			if err := r.decryptCredentials(&configs[i]); err != nil {
				log.WithError(err).WithField("config_id", configs[i].ID).Warn("Failed to decrypt credentials")
				// Continue with other configs
			}
		}
	}

	log.WithField("count", len(configs)).Info("Retrieved OSSEA configurations")
	return configs, nil
}

// GetByName retrieves an OSSEA configuration by name
func (r *OSSEAConfigRepository) GetByName(name string) (*OSSEAConfig, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var config OSSEAConfig
	if err := r.db.Where("name = ? AND is_active = ?", name, true).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("OSSEA configuration '%s' not found", name)
		}
		log.WithError(err).WithField("config_name", name).Error("Failed to retrieve OSSEA configuration")
		return nil, fmt.Errorf("failed to retrieve OSSEA configuration: %w", err)
	}

	return &config, nil
}

// Update updates an existing OSSEA configuration
func (r *OSSEAConfigRepository) Update(id int, config *OSSEAConfig) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	// Verify the configuration exists
	existing, err := r.GetByID(id)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"config_id":   id,
		"config_name": config.Name,
	}).Info("Updating OSSEA configuration")

	// Update the configuration
	config.ID = existing.ID // Ensure we keep the same ID
	if err := r.db.Save(config).Error; err != nil {
		log.WithError(err).WithField("config_id", id).Error("Failed to update OSSEA configuration")
		return fmt.Errorf("failed to update OSSEA configuration: %w", err)
	}

	log.WithField("config_id", id).Info("OSSEA configuration updated successfully")
	return nil
}

// Delete marks an OSSEA configuration as inactive (soft delete)
func (r *OSSEAConfigRepository) Delete(id int) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	// Verify the configuration exists
	_, err := r.GetByID(id)
	if err != nil {
		return err
	}

	log.WithField("config_id", id).Info("Deleting OSSEA configuration")

	// Soft delete by marking as inactive
	if err := r.db.Model(&OSSEAConfig{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		log.WithError(err).WithField("config_id", id).Error("Failed to delete OSSEA configuration")
		return fmt.Errorf("failed to delete OSSEA configuration: %w", err)
	}

	log.WithField("config_id", id).Info("OSSEA configuration deleted successfully")
	return nil
}

// TestConnection tests the OSSEA connection using the configuration
func (r *OSSEAConfigRepository) TestConnection(config *OSSEAConfig) (bool, string, error) {
	log.WithField("api_url", config.APIURL).Info("Testing OSSEA connection")

	// Create OSSEA client
	client := ossea.NewClient(
		config.APIURL,
		config.APIKey,
		config.SecretKey,
		config.Domain,
		config.Zone,
	)

	// Test connection by listing zones
	zones, err := client.ListZones()
	if err != nil {
		log.WithError(err).Error("OSSEA connection test failed")
		return false, fmt.Sprintf("OSSEA connection failed: %v", err), err
	}

	// Verify the configured zone exists
	zoneFound := false
	var foundZones []string
	for _, zone := range zones {
		foundZones = append(foundZones, zone.Name)
		if zone.Name == config.Zone {
			zoneFound = true
			break
		}
	}

	if !zoneFound {
		message := fmt.Sprintf("Configured zone '%s' not found in OSSEA. Available zones: %v", config.Zone, foundZones)
		log.WithField("zone", config.Zone).WithField("available_zones", foundZones).Warn("Zone not found")
		return false, message, nil
	}

	successMessage := fmt.Sprintf("OSSEA connection successful! Zone '%s' verified. Found %d total zones.", config.Zone, len(zones))
	log.WithFields(log.Fields{
		"zone":        config.Zone,
		"total_zones": len(zones),
		"api_url":     config.APIURL,
	}).Info("OSSEA connection test successful")

	return true, successMessage, nil
}

// AutoMigrate creates or updates the database schema for all models
func (r *OSSEAConfigRepository) AutoMigrate() error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.Info("Running database auto-migration for all models")

	// Migrate all database models in dependency order
	models := []interface{}{
		&OSSEAConfig{},     // Base configuration (no dependencies)
		&ReplicationJob{},  // References OSSEAConfig
		&OSSEAVolume{},     // References OSSEAConfig
		&VMDisk{},          // References ReplicationJob and OSSEAVolume
		&VolumeMount{},     // References OSSEAVolume and ReplicationJob
		&CBTHistory{},      // References VMDisk - MUST come after VMDisk
		&NBDExport{},       // References ReplicationJob and OSSEAVolume
		&VMExportMapping{}, // VM to NBD export persistence mapping
		&FailoverJob{},     // References ReplicationJob - failover operations
		&NetworkMapping{},  // Network mappings for VMs (independent)
	}

	for _, model := range models {
		if err := r.db.AutoMigrate(model); err != nil {
			log.WithError(err).Errorf("Failed to auto-migrate table for %T", model)
			return fmt.Errorf("failed to migrate table for %T: %w", model, err)
		}
		log.Infof("Successfully migrated table for %T", model)
	}

	// Add custom indexes after migration
	if err := r.addCustomIndexes(); err != nil {
		log.WithError(err).Error("Failed to add custom indexes")
		return fmt.Errorf("failed to add custom indexes: %w", err)
	}

	log.Info("Database auto-migration completed successfully for all models")
	return nil
}

// addCustomIndexes adds custom indexes that aren't handled by GORM tags
func (r *OSSEAConfigRepository) addCustomIndexes() error {
	log.Info("Adding custom database indexes")

	// Add unique constraint on network mappings (vm_id, source_network_name)
	if err := r.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_network_mappings_unique_vm_network ON network_mappings (vm_id, source_network_name)").Error; err != nil {
		log.WithError(err).Error("Failed to create unique index on network_mappings")
		return fmt.Errorf("failed to create network mappings unique index: %w", err)
	}

	log.Info("Custom indexes added successfully")
	return nil
}

// UpdateVolumeStatus updates ossea_volumes status and device_path after Volume Daemon operations
// DEPRECATED: Volume Daemon now automatically handles ossea_volumes updates during attach/detach operations
func (r *OSSEAConfigRepository) UpdateVolumeStatus(volumeID, status, devicePath string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"status":      status,
		"device_path": devicePath,
	}).Info("Updating volume status and device path")

	result := r.db.Model(&OSSEAVolume{}).
		Where("volume_id = ?", volumeID).
		Updates(map[string]interface{}{
			"status":      status,
			"device_path": devicePath,
			"updated_at":  time.Now(),
		})

	if result.Error != nil {
		log.WithError(result.Error).WithField("volume_id", volumeID).Error("Failed to update volume status")
		return fmt.Errorf("failed to update volume status for %s: %w", volumeID, result.Error)
	}

	if result.RowsAffected == 0 {
		log.WithField("volume_id", volumeID).Warn("No rows affected - volume not found")
		return fmt.Errorf("volume not found: %s", volumeID)
	}

	log.WithFields(log.Fields{
		"volume_id":     volumeID,
		"rows_affected": result.RowsAffected,
	}).Info("✅ Volume status updated successfully")

	return nil
}

// VMDiskRepository handles VM disk database operations
type VMDiskRepository struct {
	db *gorm.DB
}

// NewVMDiskRepository creates a new VM disk repository
func NewVMDiskRepository(conn Connection) *VMDiskRepository {
	return &VMDiskRepository{
		db: conn.GetGormDB(),
	}
}

// Create saves a new VM disk to the database
func (r *VMDiskRepository) Create(disk *VMDisk) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"job_id":    disk.JobID,
		"disk_id":   disk.DiskID,
		"vmdk_path": disk.VMDKPath,
		"size_gb":   disk.SizeGB,
	}).Info("Creating VM disk record in database")

	if err := r.db.Create(disk).Error; err != nil {
		log.WithError(err).Error("Failed to create VM disk record")
		return fmt.Errorf("failed to create VM disk: %w", err)
	}

	log.WithFields(log.Fields{
		"id":      disk.ID,
		"job_id":  disk.JobID,
		"disk_id": disk.DiskID,
	}).Info("VM disk record created successfully")

	return nil
}

// GetByJobID retrieves all VM disks for a specific job
func (r *VMDiskRepository) GetByJobID(jobID string) ([]VMDisk, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var disks []VMDisk
	if err := r.db.Where("job_id = ?", jobID).Find(&disks).Error; err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("Failed to retrieve VM disks")
		return nil, fmt.Errorf("failed to get VM disks for job %s: %w", jobID, err)
	}

	log.WithFields(log.Fields{
		"job_id":     jobID,
		"disk_count": len(disks),
	}).Debug("Retrieved VM disks for job")

	return disks, nil
}

// GetByID retrieves a VM disk by ID
func (r *VMDiskRepository) GetByID(id int) (*VMDisk, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var disk VMDisk
	if err := r.db.First(&disk, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("VM disk not found: %d", id)
		}
		log.WithError(err).WithField("id", id).Error("Failed to retrieve VM disk")
		return nil, fmt.Errorf("failed to get VM disk %d: %w", id, err)
	}

	return &disk, nil
}

// UpdateSyncStatus updates the sync status and progress for a VM disk
func (r *VMDiskRepository) UpdateSyncStatus(id int, status string, progressPercent float64, bytesSynced int64) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	updates := map[string]interface{}{
		"sync_status":           status,
		"sync_progress_percent": progressPercent,
		"bytes_synced":          bytesSynced,
		"updated_at":            time.Now(),
	}

	if err := r.db.Model(&VMDisk{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		log.WithError(err).WithField("id", id).Error("Failed to update VM disk sync status")
		return fmt.Errorf("failed to update VM disk sync status: %w", err)
	}

	log.WithFields(log.Fields{
		"id":               id,
		"status":           status,
		"progress_percent": progressPercent,
		"bytes_synced":     bytesSynced,
	}).Info("VM disk sync status updated")

	return nil
}

// UpdateChangeID updates the CBT change ID for a VM disk
func (r *VMDiskRepository) UpdateChangeID(id int, changeID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	if err := r.db.Model(&VMDisk{}).Where("id = ?", id).Updates(map[string]interface{}{
		"disk_change_id": changeID,
		"updated_at":     time.Now(),
	}).Error; err != nil {
		log.WithError(err).WithField("id", id).Error("Failed to update VM disk change ID")
		return fmt.Errorf("failed to update VM disk change ID: %w", err)
	}

	log.WithFields(log.Fields{
		"id":        id,
		"change_id": changeID,
	}).Info("VM disk change ID updated")

	return nil
}

// FindByContextAndDiskID finds existing vm_disk record for stable ID management
func (r *VMDiskRepository) FindByContextAndDiskID(vmContextID, diskID string) (*VMDisk, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var vmDisk VMDisk
	err := r.db.Where("vm_context_id = ? AND disk_id = ?", vmContextID, diskID).First(&vmDisk).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error - allows CREATE path
		}
		log.WithError(err).WithFields(log.Fields{
			"vm_context_id": vmContextID,
			"disk_id":       diskID,
		}).Error("Failed to find VM disk by context and disk ID")
		return nil, fmt.Errorf("failed to find vm_disk: %w", err)
	}

	log.WithFields(log.Fields{
		"id":            vmDisk.ID,
		"vm_context_id": vmContextID,
		"disk_id":       diskID,
		"job_id":        vmDisk.JobID,
	}).Debug("Found existing VM disk record")

	return &vmDisk, nil
}

// Update updates existing vm_disk record for stable ID management
func (r *VMDiskRepository) Update(disk *VMDisk) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"id":      disk.ID,
		"job_id":  disk.JobID,
		"disk_id": disk.DiskID,
		"size_gb": disk.SizeGB,
	}).Info("Updating existing VM disk record")

	if err := r.db.Save(disk).Error; err != nil {
		log.WithError(err).Error("Failed to update VM disk record")
		return fmt.Errorf("failed to update VM disk: %w", err)
	}

	log.WithFields(log.Fields{
		"id":      disk.ID,
		"job_id":  disk.JobID,
		"disk_id": disk.DiskID,
	}).Info("VM disk record updated successfully")

	return nil
}

// OSSEAVolumeRepository handles OSSEA volume database operations
type OSSEAVolumeRepository struct {
	db *gorm.DB
}

// NewOSSEAVolumeRepository creates a new OSSEA volume repository
func NewOSSEAVolumeRepository(conn Connection) *OSSEAVolumeRepository {
	return &OSSEAVolumeRepository{
		db: conn.GetGormDB(),
	}
}

// Create saves a new OSSEA volume to the database
func (r *OSSEAVolumeRepository) Create(volume *OSSEAVolume) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"volume_id":       volume.VolumeID,
		"volume_name":     volume.VolumeName,
		"size_gb":         volume.SizeGB,
		"ossea_config_id": volume.OSSEAConfigID,
	}).Info("Creating OSSEA volume record in database")

	if err := r.db.Create(volume).Error; err != nil {
		log.WithError(err).Error("Failed to create OSSEA volume record")
		return fmt.Errorf("failed to create OSSEA volume: %w", err)
	}

	log.WithFields(log.Fields{
		"id":          volume.ID,
		"volume_id":   volume.VolumeID,
		"volume_name": volume.VolumeName,
	}).Info("OSSEA volume record created successfully")

	return nil
}

// GetByVolumeID retrieves an OSSEA volume by its CloudStack volume ID
func (r *OSSEAVolumeRepository) GetByVolumeID(volumeID string) (*OSSEAVolume, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var volume OSSEAVolume
	if err := r.db.Where("volume_id = ?", volumeID).First(&volume).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("OSSEA volume not found: %s", volumeID)
		}
		log.WithError(err).WithField("volume_id", volumeID).Error("Failed to retrieve OSSEA volume")
		return nil, fmt.Errorf("failed to get OSSEA volume %s: %w", volumeID, err)
	}

	return &volume, nil
}

// GetByID retrieves an OSSEA volume by database ID
func (r *OSSEAVolumeRepository) GetByID(id int) (*OSSEAVolume, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var volume OSSEAVolume
	if err := r.db.First(&volume, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("OSSEA volume not found: %d", id)
		}
		log.WithError(err).WithField("id", id).Error("Failed to retrieve OSSEA volume")
		return nil, fmt.Errorf("failed to get OSSEA volume %d: %w", id, err)
	}

	return &volume, nil
}

// UpdateStatus updates the status of an OSSEA volume
func (r *OSSEAVolumeRepository) UpdateStatus(id int, status string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	if err := r.db.Model(&OSSEAVolume{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		log.WithError(err).WithField("id", id).Error("Failed to update OSSEA volume status")
		return fmt.Errorf("failed to update OSSEA volume status: %w", err)
	}

	log.WithFields(log.Fields{
		"id":     id,
		"status": status,
	}).Info("OSSEA volume status updated")

	return nil
}

// UpdateMountInfo updates the device path and mount point for an OSSEA volume
func (r *OSSEAVolumeRepository) UpdateMountInfo(id int, devicePath, mountPoint string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	if err := r.db.Model(&OSSEAVolume{}).Where("id = ?", id).Updates(map[string]interface{}{
		"device_path": devicePath,
		"mount_point": mountPoint,
		"updated_at":  time.Now(),
	}).Error; err != nil {
		log.WithError(err).WithField("id", id).Error("Failed to update OSSEA volume mount info")
		return fmt.Errorf("failed to update OSSEA volume mount info: %w", err)
	}

	log.WithFields(log.Fields{
		"id":          id,
		"device_path": devicePath,
		"mount_point": mountPoint,
	}).Info("OSSEA volume mount info updated")

	return nil
}

// VolumeMountRepository handles volume mount database operations
type VolumeMountRepository struct {
	db *gorm.DB
}

// NewVolumeMountRepository creates a new volume mount repository
func NewVolumeMountRepository(conn Connection) *VolumeMountRepository {
	return &VolumeMountRepository{
		db: conn.GetGormDB(),
	}
}

// Create saves a new volume mount to the database
func (r *VolumeMountRepository) Create(mount *VolumeMount) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"ossea_volume_id": mount.OSSEAVolumeID,
		"job_id":          mount.JobID,
		"device_path":     mount.DevicePath,
		"mount_point":     mount.MountPoint,
	}).Info("Creating volume mount record in database")

	if err := r.db.Create(mount).Error; err != nil {
		log.WithError(err).Error("Failed to create volume mount record")
		return fmt.Errorf("failed to create volume mount: %w", err)
	}

	log.WithFields(log.Fields{
		"id":              mount.ID,
		"ossea_volume_id": mount.OSSEAVolumeID,
		"job_id":          mount.JobID,
	}).Info("Volume mount record created successfully")

	return nil
}

// GetByJobID retrieves all volume mounts for a specific job
func (r *VolumeMountRepository) GetByJobID(jobID string) ([]VolumeMount, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var mounts []VolumeMount
	if err := r.db.Where("job_id = ?", jobID).Find(&mounts).Error; err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("Failed to retrieve volume mounts")
		return nil, fmt.Errorf("failed to get volume mounts for job %s: %w", jobID, err)
	}

	log.WithFields(log.Fields{
		"job_id":      jobID,
		"mount_count": len(mounts),
	}).Debug("Retrieved volume mounts for job")

	return mounts, nil
}

// GetByVolumeID retrieves volume mount for a specific OSSEA volume
func (r *VolumeMountRepository) GetByVolumeID(volumeID int) (*VolumeMount, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var mount VolumeMount
	if err := r.db.Where("ossea_volume_id = ?", volumeID).First(&mount).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("volume mount not found for volume: %d", volumeID)
		}
		log.WithError(err).WithField("volume_id", volumeID).Error("Failed to retrieve volume mount")
		return nil, fmt.Errorf("failed to get volume mount for volume %d: %w", volumeID, err)
	}

	return &mount, nil
}

// UpdateMountStatus updates the mount status and timestamps
func (r *VolumeMountRepository) UpdateMountStatus(id int, status string, mounted bool) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	updates := map[string]interface{}{
		"mount_status": status,
		"updated_at":   time.Now(),
	}

	if mounted {
		updates["mounted_at"] = time.Now()
		updates["unmounted_at"] = nil
	} else {
		updates["unmounted_at"] = time.Now()
	}

	if err := r.db.Model(&VolumeMount{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		log.WithError(err).WithField("id", id).Error("Failed to update volume mount status")
		return fmt.Errorf("failed to update volume mount status: %w", err)
	}

	log.WithFields(log.Fields{
		"id":      id,
		"status":  status,
		"mounted": mounted,
	}).Info("Volume mount status updated")

	return nil
}

// CBTHistoryRepository handles CBT history database operations
type CBTHistoryRepository struct {
	db *gorm.DB
}

// NewCBTHistoryRepository creates a new CBT history repository
func NewCBTHistoryRepository(conn Connection) *CBTHistoryRepository {
	return &CBTHistoryRepository{
		db: conn.GetGormDB(),
	}
}

// Create saves a new CBT history record to the database
func (r *CBTHistoryRepository) Create(history *CBTHistory) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"job_id":         history.JobID,
		"disk_id":        history.DiskID,
		"change_id":      history.ChangeID,
		"sync_type":      history.SyncType,
		"blocks_changed": history.BlocksChanged,
	}).Info("Creating CBT history record in database")

	if err := r.db.Create(history).Error; err != nil {
		log.WithError(err).Error("Failed to create CBT history record")
		return fmt.Errorf("failed to create CBT history: %w", err)
	}

	log.WithFields(log.Fields{
		"id":        history.ID,
		"job_id":    history.JobID,
		"disk_id":   history.DiskID,
		"change_id": history.ChangeID,
	}).Info("CBT history record created successfully")

	return nil
}

// GetByDiskID retrieves CBT history for a specific disk
func (r *CBTHistoryRepository) GetByDiskID(diskID string) ([]CBTHistory, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var history []CBTHistory
	if err := r.db.Where("disk_id = ?", diskID).Order("created_at DESC").Find(&history).Error; err != nil {
		log.WithError(err).WithField("disk_id", diskID).Error("Failed to retrieve CBT history")
		return nil, fmt.Errorf("failed to get CBT history for disk %s: %w", diskID, err)
	}

	log.WithFields(log.Fields{
		"disk_id":       diskID,
		"history_count": len(history),
	}).Debug("Retrieved CBT history for disk")

	return history, nil
}

// GetLatestByDiskID retrieves the most recent CBT history for a specific disk
func (r *CBTHistoryRepository) GetLatestByDiskID(diskID string) (*CBTHistory, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var history CBTHistory
	if err := r.db.Where("disk_id = ?", diskID).Order("created_at DESC").First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no CBT history found for disk: %s", diskID)
		}
		log.WithError(err).WithField("disk_id", diskID).Error("Failed to retrieve latest CBT history")
		return nil, fmt.Errorf("failed to get latest CBT history for disk %s: %w", diskID, err)
	}

	return &history, nil
}

// GetByJobID retrieves all CBT history for a specific job
func (r *CBTHistoryRepository) GetByJobID(jobID string) ([]CBTHistory, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var history []CBTHistory
	if err := r.db.Where("job_id = ?", jobID).Order("created_at DESC").Find(&history).Error; err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("Failed to retrieve CBT history for job")
		return nil, fmt.Errorf("failed to get CBT history for job %s: %w", jobID, err)
	}

	log.WithFields(log.Fields{
		"job_id":        jobID,
		"history_count": len(history),
	}).Debug("Retrieved CBT history for job")

	return history, nil
}

// MarkSyncCompleted marks a CBT sync as completed with success status
func (r *CBTHistoryRepository) MarkSyncCompleted(id int, success bool, bytesTransferred int64, durationSeconds int) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	updates := map[string]interface{}{
		"sync_success":          success,
		"bytes_transferred":     bytesTransferred,
		"sync_duration_seconds": durationSeconds,
	}

	if err := r.db.Model(&CBTHistory{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		log.WithError(err).WithField("id", id).Error("Failed to mark CBT sync as completed")
		return fmt.Errorf("failed to mark CBT sync as completed: %w", err)
	}

	log.WithFields(log.Fields{
		"id":                id,
		"success":           success,
		"bytes_transferred": bytesTransferred,
		"duration_seconds":  durationSeconds,
	}).Info("CBT sync marked as completed")

	return nil
}

// NBD Export Repository Methods

// CreateNBDExport creates a new NBD export record
func (r *OSSEAConfigRepository) CreateNBDExport(export *NBDExport) error {
	log.WithFields(log.Fields{
		"job_id":      export.JobID,
		"volume_id":   export.VolumeID,
		"export_name": export.ExportName,
		"port":        export.Port,
	}).Info("Creating NBD export record")

	if err := r.db.Create(export).Error; err != nil {
		log.WithError(err).Error("Failed to create NBD export record")
		return fmt.Errorf("failed to create NBD export: %w", err)
	}

	log.WithFields(log.Fields{
		"id":          export.ID,
		"job_id":      export.JobID,
		"export_name": export.ExportName,
		"port":        export.Port,
	}).Info("✅ NBD export record created")

	return nil
}

// GetNBDExportByJobID retrieves NBD exports for a specific job
func (r *OSSEAConfigRepository) GetNBDExportByJobID(jobID string) ([]NBDExport, error) {
	var exports []NBDExport

	if err := r.db.Where("job_id = ?", jobID).Find(&exports).Error; err != nil {
		return nil, fmt.Errorf("failed to get NBD exports for job %s: %w", jobID, err)
	}

	return exports, nil
}

// UpdateNBDExportStatus updates the status of an NBD export
func (r *OSSEAConfigRepository) UpdateNBDExportStatus(id uint, status string) error {
	log.WithFields(log.Fields{
		"id":     id,
		"status": status,
	}).Info("Updating NBD export status")

	if err := r.db.Model(&NBDExport{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update NBD export status: %w", err)
	}

	return nil
}

// DeleteNBDExport removes an NBD export record
func (r *OSSEAConfigRepository) DeleteNBDExport(id uint) error {
	log.WithField("id", id).Info("Deleting NBD export record")

	if err := r.db.Delete(&NBDExport{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete NBD export: %w", err)
	}

	return nil
}

// FailoverJobRepository handles failover job database operations
type FailoverJobRepository struct {
	db *gorm.DB
}

// NewFailoverJobRepository creates a new failover job repository
func NewFailoverJobRepository(conn Connection) *FailoverJobRepository {
	return &FailoverJobRepository{
		db: conn.GetGormDB(),
	}
}

// Create saves a new failover job to the database
func (r *FailoverJobRepository) Create(job *FailoverJob) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"job_id":   job.JobID,
		"vm_id":    job.VMID,
		"job_type": job.JobType,
	}).Info("Creating failover job in database")

	if err := r.db.Create(job).Error; err != nil {
		log.WithError(err).Error("Failed to create failover job")
		return fmt.Errorf("failed to create failover job: %w", err)
	}

	log.WithField("job_id", job.JobID).Info("✅ Failover job created")
	return nil
}

// GetByJobID retrieves a failover job by its job ID
func (r *FailoverJobRepository) GetByJobID(jobID string) (*FailoverJob, error) {
	var job FailoverJob

	if err := r.db.Where("job_id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failover job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get failover job: %w", err)
	}

	return &job, nil
}

// GetByVMID retrieves failover jobs for a specific VM
func (r *FailoverJobRepository) GetByVMID(vmID string) ([]FailoverJob, error) {
	var jobs []FailoverJob

	if err := r.db.Where("vm_id = ?", vmID).Order("created_at DESC").Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to get failover jobs for VM %s: %w", vmID, err)
	}

	return jobs, nil
}

// UpdateStatus updates the status of a failover job
func (r *FailoverJobRepository) UpdateStatus(jobID, status string) error {
	log.WithFields(log.Fields{
		"job_id": jobID,
		"status": status,
	}).Info("Updating failover job status")

	if err := r.db.Model(&FailoverJob{}).Where("job_id = ?", jobID).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update failover job status: %w", err)
	}

	return nil
}

// UpdateDestinationVM updates the destination VM ID for a failover job
func (r *FailoverJobRepository) UpdateDestinationVM(jobID, destinationVMID string) error {
	log.WithFields(log.Fields{
		"job_id":            jobID,
		"destination_vm_id": destinationVMID,
	}).Info("Updating failover job destination VM")

	if err := r.db.Model(&FailoverJob{}).Where("job_id = ?", jobID).Update("destination_vm_id", destinationVMID).Error; err != nil {
		return fmt.Errorf("failed to update failover job destination VM: %w", err)
	}

	return nil
}

// UpdateSnapshot updates the snapshot ID for a failover job
func (r *FailoverJobRepository) UpdateSnapshot(jobID, snapshotID string) error {
	log.WithFields(log.Fields{
		"job_id":      jobID,
		"snapshot_id": snapshotID,
	}).Info("Updating failover job snapshot")

	if err := r.db.Model(&FailoverJob{}).Where("job_id = ?", jobID).Update("ossea_snapshot_id", snapshotID).Error; err != nil {
		return fmt.Errorf("failed to update failover job snapshot: %w", err)
	}

	return nil
}

// UpdateLinstorSnapshot updates the Linstor snapshot name for a failover job
func (r *FailoverJobRepository) UpdateLinstorSnapshot(jobID, linstorSnapshotName string) error {
	log.WithFields(log.Fields{
		"job_id":                jobID,
		"linstor_snapshot_name": linstorSnapshotName,
	}).Info("Updating failover job Linstor snapshot")

	if err := r.db.Model(&FailoverJob{}).Where("job_id = ?", jobID).Update("linstor_snapshot_name", linstorSnapshotName).Error; err != nil {
		return fmt.Errorf("failed to update failover job Linstor snapshot: %w", err)
	}

	return nil
}

// MarkStarted updates the started_at timestamp for a failover job
func (r *FailoverJobRepository) MarkStarted(jobID string) error {
	now := time.Now()
	if err := r.db.Model(&FailoverJob{}).Where("job_id = ?", jobID).Update("started_at", &now).Error; err != nil {
		return fmt.Errorf("failed to mark failover job as started: %w", err)
	}

	return nil
}

// MarkCompleted updates the completed_at timestamp and status for a failover job
func (r *FailoverJobRepository) MarkCompleted(jobID string) error {
	now := time.Now()
	if err := r.db.Model(&FailoverJob{}).Where("job_id = ?", jobID).Updates(map[string]interface{}{
		"status":       "completed",
		"completed_at": &now,
	}).Error; err != nil {
		return fmt.Errorf("failed to mark failover job as completed: %w", err)
	}

	return nil
}

// SetError updates the error message and status for a failover job
func (r *FailoverJobRepository) SetError(jobID, errorMessage string) error {
	log.WithFields(log.Fields{
		"job_id": jobID,
		"error":  errorMessage,
	}).Error("Setting failover job error")

	if err := r.db.Model(&FailoverJob{}).Where("job_id = ?", jobID).Updates(map[string]interface{}{
		"status":        "failed",
		"error_message": errorMessage,
	}).Error; err != nil {
		return fmt.Errorf("failed to set failover job error: %w", err)
	}

	return nil
}

// NetworkMappingRepository handles network mapping database operations
type NetworkMappingRepository struct {
	db *gorm.DB
}

// NewNetworkMappingRepository creates a new network mapping repository
func NewNetworkMappingRepository(conn Connection) *NetworkMappingRepository {
	return &NetworkMappingRepository{
		db: conn.GetGormDB(),
	}
}

// CreateOrUpdate saves or updates a network mapping
func (r *NetworkMappingRepository) CreateOrUpdate(mapping *NetworkMapping) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"vm_id":               mapping.VMID,
		"source_network":      mapping.SourceNetworkName,
		"destination_network": mapping.DestinationNetworkName,
		"is_test_network":     mapping.IsTestNetwork,
	}).Info("Creating/updating network mapping")

	// Check if record exists first
	var existingMapping NetworkMapping
	err := r.db.Where("vm_id = ? AND source_network_name = ? AND is_test_network = ?",
		mapping.VMID, mapping.SourceNetworkName, mapping.IsTestNetwork).
		First(&existingMapping).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithError(err).Error("Failed to query existing network mapping")
		return fmt.Errorf("failed to query existing network mapping: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new record
		if err := r.db.Create(mapping).Error; err != nil {
			log.WithError(err).Error("Failed to create network mapping")
			return fmt.Errorf("failed to create network mapping: %w", err)
		}
		log.WithField("mapping_id", mapping.ID).Info("✅ Created new network mapping")
	} else {
		// Update existing record
		mapping.ID = existingMapping.ID // Preserve the ID
		if err := r.db.Save(mapping).Error; err != nil {
			log.WithError(err).Error("Failed to update network mapping")
			return fmt.Errorf("failed to update network mapping: %w", err)
		}
		log.WithField("mapping_id", mapping.ID).Info("✅ Updated existing network mapping")
	}

	log.WithField("mapping_id", mapping.ID).Info("✅ Network mapping saved")
	return nil
}

// GetByVMID retrieves all network mappings for a specific VM
func (r *NetworkMappingRepository) GetByVMID(vmID string) ([]NetworkMapping, error) {
	var mappings []NetworkMapping

	if err := r.db.Where("vm_id = ?", vmID).Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get network mappings for VM %s: %w", vmID, err)
	}

	return mappings, nil
}

// GetByVMAndNetwork retrieves a specific network mapping
func (r *NetworkMappingRepository) GetByVMAndNetwork(vmID, sourceNetworkName string) (*NetworkMapping, error) {
	var mapping NetworkMapping

	if err := r.db.Where("vm_id = ? AND source_network_name = ?", vmID, sourceNetworkName).First(&mapping).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("network mapping not found for VM %s, network %s", vmID, sourceNetworkName)
		}
		return nil, fmt.Errorf("failed to get network mapping: %w", err)
	}

	return &mapping, nil
}

// DeleteByVMID removes all network mappings for a specific VM
func (r *NetworkMappingRepository) DeleteByVMID(vmID string) error {
	log.WithField("vm_id", vmID).Info("Deleting network mappings for VM")

	if err := r.db.Where("vm_id = ?", vmID).Delete(&NetworkMapping{}).Error; err != nil {
		return fmt.Errorf("failed to delete network mappings for VM %s: %w", vmID, err)
	}

	return nil
}

// GetTestNetworkMappings retrieves all test network mappings
func (r *NetworkMappingRepository) GetTestNetworkMappings() ([]NetworkMapping, error) {
	var mappings []NetworkMapping

	if err := r.db.Where("is_test_network = ?", true).Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get test network mappings: %w", err)
	}

	return mappings, nil
}

// VM-Centric Network Mapping Methods (Phase 5.5 Enhancement)
// These methods support the unified failover system's VM-centric architecture

// GetByContextID retrieves all network mappings for a specific VM context
// This method supports the VM-centric architecture by using context_id as the primary identifier
// ENHANCED: Now uses direct context_id lookup with backward compatibility fallback
func (r *NetworkMappingRepository) GetByContextID(contextID string) ([]NetworkMapping, error) {
	var mappings []NetworkMapping

	// PHASE 1: Try direct context_id lookup (new schema - preferred method)
	if err := r.db.Where("vm_context_id = ?", contextID).Find(&mappings).Error; err == nil && len(mappings) > 0 {
		log.WithFields(log.Fields{
			"context_id":    contextID,
			"mapping_count": len(mappings),
			"lookup_method": "direct_context_id",
		}).Debug("Retrieved network mappings by direct context ID lookup")
		return mappings, nil
	}

	// PHASE 2: Fallback to vm_name resolution (backward compatibility)
	log.WithField("context_id", contextID).Debug("Direct context_id lookup yielded no results, trying fallback")

	var result struct {
		VMName string `gorm:"column:vm_name"`
	}
	if err := r.db.Table("vm_replication_contexts").
		Select("vm_name").
		Where("context_id = ?", contextID).
		First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// No VM context found - return empty mappings (not an error)
			return []NetworkMapping{}, nil
		}
		return nil, fmt.Errorf("failed to resolve context_id to vm_name: %w", err)
	}

	// Use the resolved vm_name to get network mappings (legacy method)
	vmName := result.VMName
	if err := r.db.Where("vm_id = ?", vmName).Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get network mappings for context %s: %w", contextID, err)
	}

	log.WithFields(log.Fields{
		"context_id":    contextID,
		"vm_name":       vmName,
		"mapping_count": len(mappings),
		"lookup_method": "fallback_vm_name",
	}).Debug("Retrieved network mappings by fallback vm_name lookup")

	return mappings, nil
}

// CreateForContext creates a network mapping for a specific VM context
// This method supports VM-centric network mapping creation
func (r *NetworkMappingRepository) CreateForContext(contextID string, mapping *NetworkMapping) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	// Resolve context_id to vm_name for backward compatibility
	var result struct {
		VMName string `gorm:"column:vm_name"`
	}
	if err := r.db.Table("vm_replication_contexts").
		Select("vm_name").
		Where("context_id = ?", contextID).
		First(&result).Error; err != nil {
		return fmt.Errorf("failed to resolve context_id to vm_name: %w", err)
	}

	// Set the VM ID from the resolved name
	vmName := result.VMName
	mapping.VMID = vmName

	log.WithFields(log.Fields{
		"context_id":          contextID,
		"vm_name":             vmName,
		"source_network":      mapping.SourceNetworkName,
		"destination_network": mapping.DestinationNetworkName,
	}).Info("Creating network mapping for context")

	// Use existing CreateOrUpdate method
	return r.CreateOrUpdate(mapping)
}

// DeleteSpecificMapping removes a specific network mapping by context and source network
// This replaces the inefficient delete-all-recreate pattern
func (r *NetworkMappingRepository) DeleteSpecificMapping(contextID, sourceNetworkName string) error {
	// Resolve context_id to vm_name
	var result struct {
		VMName string `gorm:"column:vm_name"`
	}
	if err := r.db.Table("vm_replication_contexts").
		Select("vm_name").
		Where("context_id = ?", contextID).
		First(&result).Error; err != nil {
		return fmt.Errorf("failed to resolve context_id to vm_name: %w", err)
	}

	vmName := result.VMName

	log.WithFields(log.Fields{
		"context_id":     contextID,
		"vm_name":        vmName,
		"source_network": sourceNetworkName,
	}).Info("Deleting specific network mapping")

	if err := r.db.Where("vm_id = ? AND source_network_name = ?", vmName, sourceNetworkName).
		Delete(&NetworkMapping{}).Error; err != nil {
		return fmt.Errorf("failed to delete network mapping for context %s, network %s: %w",
			contextID, sourceNetworkName, err)
	}

	return nil
}

// UpdateMappingDestination updates the destination network for a specific mapping
// This enables targeted mapping updates without recreating all mappings
func (r *NetworkMappingRepository) UpdateMappingDestination(contextID, sourceNetworkName, newDestinationID string) error {
	// Resolve context_id to vm_name
	var vmResult struct {
		VMName string `gorm:"column:vm_name"`
	}
	if err := r.db.Table("vm_replication_contexts").
		Select("vm_name").
		Where("context_id = ?", contextID).
		First(&vmResult).Error; err != nil {
		return fmt.Errorf("failed to resolve context_id to vm_name: %w", err)
	}

	vmName := vmResult.VMName

	log.WithFields(log.Fields{
		"context_id":         contextID,
		"vm_name":            vmName,
		"source_network":     sourceNetworkName,
		"new_destination_id": newDestinationID,
	}).Info("Updating network mapping destination")

	updateResult := r.db.Model(&NetworkMapping{}).
		Where("vm_id = ? AND source_network_name = ?", vmName, sourceNetworkName).
		Update("destination_network_id", newDestinationID)

	if updateResult.Error != nil {
		return fmt.Errorf("failed to update network mapping destination: %w", updateResult.Error)
	}

	if updateResult.RowsAffected == 0 {
		return fmt.Errorf("no network mapping found for context %s, network %s", contextID, sourceNetworkName)
	}

	return nil
}

// Enhanced Network Mapping Repository Methods (Phase 2 Implementation)
// These methods leverage the new schema fields while maintaining backward compatibility

// UpdateValidationStatus updates the validation status for a specific network mapping
func (r *NetworkMappingRepository) UpdateValidationStatus(contextID string, sourceNetwork string, status string) error {
	log.WithFields(log.Fields{
		"context_id":        contextID,
		"source_network":    sourceNetwork,
		"validation_status": status,
	}).Info("Updating network mapping validation status")

	// Try direct context_id update first (new schema)
	updateResult := r.db.Model(&NetworkMapping{}).
		Where("vm_context_id = ? AND source_network_name = ?", contextID, sourceNetwork).
		Updates(map[string]interface{}{
			"validation_status": status,
			"last_validated":    time.Now(),
		})

	if updateResult.Error != nil {
		return fmt.Errorf("failed to update validation status: %w", updateResult.Error)
	}

	if updateResult.RowsAffected > 0 {
		log.WithField("rows_affected", updateResult.RowsAffected).Debug("Updated via direct context_id")
		return nil
	}

	// Fallback to vm_name resolution for backward compatibility
	var vmResult struct {
		VMName string `gorm:"column:vm_name"`
	}
	if err := r.db.Table("vm_replication_contexts").
		Select("vm_name").
		Where("context_id = ?", contextID).
		First(&vmResult).Error; err != nil {
		return fmt.Errorf("failed to resolve context_id for validation update: %w", err)
	}

	updateResult = r.db.Model(&NetworkMapping{}).
		Where("vm_id = ? AND source_network_name = ?", vmResult.VMName, sourceNetwork).
		Updates(map[string]interface{}{
			"validation_status": status,
			"last_validated":    time.Now(),
		})

	if updateResult.Error != nil {
		return fmt.Errorf("failed to update validation status via fallback: %w", updateResult.Error)
	}

	if updateResult.RowsAffected == 0 {
		return fmt.Errorf("no network mapping found for context %s, network %s", contextID, sourceNetwork)
	}

	log.WithField("rows_affected", updateResult.RowsAffected).Debug("Updated via vm_name fallback")
	return nil
}

// GetMappingsByStrategy retrieves all network mappings with a specific strategy
func (r *NetworkMappingRepository) GetMappingsByStrategy(strategy string) ([]NetworkMapping, error) {
	var mappings []NetworkMapping

	if err := r.db.Where("network_strategy = ?", strategy).Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get network mappings for strategy %s: %w", strategy, err)
	}

	log.WithFields(log.Fields{
		"strategy":      strategy,
		"mapping_count": len(mappings),
	}).Debug("Retrieved network mappings by strategy")

	return mappings, nil
}

// SetNetworkStrategy updates the network strategy for all mappings of a specific VM context
func (r *NetworkMappingRepository) SetNetworkStrategy(contextID string, strategy string) error {
	log.WithFields(log.Fields{
		"context_id":       contextID,
		"network_strategy": strategy,
	}).Info("Setting network strategy for VM context")

	// Try direct context_id update first (new schema)
	updateResult := r.db.Model(&NetworkMapping{}).
		Where("vm_context_id = ?", contextID).
		Update("network_strategy", strategy)

	if updateResult.Error != nil {
		return fmt.Errorf("failed to set network strategy: %w", updateResult.Error)
	}

	if updateResult.RowsAffected > 0 {
		log.WithField("rows_affected", updateResult.RowsAffected).Debug("Strategy updated via direct context_id")
		return nil
	}

	// Fallback to vm_name resolution for backward compatibility
	var vmResult struct {
		VMName string `gorm:"column:vm_name"`
	}
	if err := r.db.Table("vm_replication_contexts").
		Select("vm_name").
		Where("context_id = ?", contextID).
		First(&vmResult).Error; err != nil {
		return fmt.Errorf("failed to resolve context_id for strategy update: %w", err)
	}

	updateResult = r.db.Model(&NetworkMapping{}).
		Where("vm_id = ?", vmResult.VMName).
		Update("network_strategy", strategy)

	if updateResult.Error != nil {
		return fmt.Errorf("failed to set network strategy via fallback: %w", updateResult.Error)
	}

	log.WithField("rows_affected", updateResult.RowsAffected).Debug("Strategy updated via vm_name fallback")
	return nil
}

// GetMappingsByValidationStatus retrieves network mappings by validation status
func (r *NetworkMappingRepository) GetMappingsByValidationStatus(status string) ([]NetworkMapping, error) {
	var mappings []NetworkMapping

	if err := r.db.Where("validation_status = ?", status).Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to get network mappings for validation status %s: %w", status, err)
	}

	log.WithFields(log.Fields{
		"validation_status": status,
		"mapping_count":     len(mappings),
	}).Debug("Retrieved network mappings by validation status")

	return mappings, nil
}

// GetReplicationJob retrieves a replication job by ID
func (r *OSSEAConfigRepository) GetReplicationJob(jobID string) (*ReplicationJob, error) {
	var job ReplicationJob

	if err := r.db.Where("id = ?", jobID).First(&job).Error; err != nil {
		return nil, fmt.Errorf("failed to get replication job %s: %w", jobID, err)
	}

	return &job, nil
}

// UpdateReplicationJob updates a replication job with given fields
func (r *OSSEAConfigRepository) UpdateReplicationJob(jobID string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"job_id": jobID,
		"fields": len(updates),
	}).Debug("Updating replication job")

	result := r.db.Model(&ReplicationJob{}).
		Where("id = ?", jobID).
		Updates(updates)

	if result.Error != nil {
		log.WithError(result.Error).WithField("job_id", jobID).Error("Failed to update replication job")
		return fmt.Errorf("failed to update replication job %s: %w", jobID, result.Error)
	}

	if result.RowsAffected == 0 {
		log.WithField("job_id", jobID).Warn("No rows affected - replication job not found")
		return fmt.Errorf("replication job not found: %s", jobID)
	}

	log.WithFields(log.Fields{
		"job_id":        jobID,
		"rows_affected": result.RowsAffected,
	}).Debug("✅ Replication job updated successfully")

	return nil
}

// GetActiveReplicationJobs retrieves jobs that are currently being replicated
func (r *OSSEAConfigRepository) GetActiveReplicationJobs() ([]ReplicationJob, error) {
	var jobs []ReplicationJob

	if err := r.db.Where("status IN ?", []string{"replicating", "snapshotting", "finalizing"}).Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to get active replication jobs: %w", err)
	}

	return jobs, nil
}

// GetVolumeIDsForJob returns volume UUIDs associated with a replication job
// Used by VMAProgressPoller to construct NBD export names
func (r *OSSEAConfigRepository) GetVolumeIDsForJob(jobID string) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `
		SELECT ov.volume_id 
		FROM replication_jobs rj
		JOIN vm_disks vd ON rj.id = vd.job_id  
		JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
		WHERE rj.id = ?`

	var volumeIDs []string
	err := r.db.Raw(query, jobID).Scan(&volumeIDs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get volume IDs for job %s: %w", jobID, err)
	}

	return volumeIDs, nil
}

// UpdateVMContextAfterJobCompletion updates VM context status when jobs complete/fail
// This method bridges OSSEAConfigRepository to ReplicationJobRepository functionality
func (r *OSSEAConfigRepository) UpdateVMContextAfterJobCompletion(jobID string) error {
	// Create a ReplicationJobRepository using the same DB connection
	replicationRepo := &ReplicationJobRepository{db: r.db}
	return replicationRepo.UpdateVMContextAfterJobCompletion(jobID)
}

// =============================================================================
// ReplicationJobRepository - Job Deletion Support
// =============================================================================

// ReplicationJobRepository handles replication job database operations
// Following project rules: clean interfaces, no monster code, modular design
// Schema verified from: source/current/oma/database/models.go ReplicationJob struct
type ReplicationJobRepository struct {
	db *gorm.DB
}

// NewReplicationJobRepository creates a new replication job repository
func NewReplicationJobRepository(conn Connection) *ReplicationJobRepository {
	return &ReplicationJobRepository{
		db: conn.GetGormDB(),
	}
}

// GetByID retrieves a replication job by ID using VERIFIED schema field names
func (r *ReplicationJobRepository) GetByID(ctx context.Context, jobID string) (*ReplicationJob, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	log.WithField("job_id", jobID).Debug("Retrieving replication job from database")

	var job ReplicationJob
	err := r.db.WithContext(ctx).Where("id = ?", jobID).First(&job).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("replication job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to retrieve replication job: %w", err)
	}

	log.WithField("job_id", jobID).Debug("✅ Replication job retrieved successfully")
	return &job, nil
}

// GetJobVolumes retrieves all volume IDs associated with a job using VERIFIED schema
// Query verified from existing handler implementation and schema documentation
func (r *ReplicationJobRepository) GetJobVolumes(ctx context.Context, jobID string) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	log.WithField("job_id", jobID).Debug("Retrieving job volumes from database")

	var volumes []string
	// VERIFIED query from existing handler implementation + schema verification
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ov.volume_id
		FROM vm_disks vd
		JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
		WHERE vd.job_id = ?
	`, jobID).Scan(&volumes).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get job volumes: %w", err)
	}

	log.WithField("job_id", jobID).WithField("volume_count", len(volumes)).Debug("✅ Job volumes retrieved successfully")
	return volumes, nil
}

// CheckActiveFailover checks if job has any active failover operations using VERIFIED schema
// Query verified from existing handler implementation and schema documentation
func (r *ReplicationJobRepository) CheckActiveFailover(ctx context.Context, jobID string) (bool, error) {
	if r.db == nil {
		return false, fmt.Errorf("database not available")
	}

	log.WithField("job_id", jobID).Debug("Checking for active failover operations")

	var count int64
	// VERIFIED query from existing handler implementation + schema verification
	// Using verified field name: replication_job_id (NOT replication_job_uuid)
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM failover_jobs 
		WHERE replication_job_id = ? 
		AND status IN ('pending', 'executing', 'validating', 'creating_vm', 'switching_volume')
	`, jobID).Scan(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check active failover: %w", err)
	}

	hasActiveFailover := count > 0
	log.WithField("job_id", jobID).WithField("active_failover", hasActiveFailover).Debug("✅ Active failover check completed")
	return hasActiveFailover, nil
}

// Delete removes a replication job and handles CASCADE DELETE relationships
// Uses VERIFIED schema field names and CASCADE DELETE behavior
func (r *ReplicationJobRepository) Delete(ctx context.Context, jobID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithField("job_id", jobID).Info("Deleting replication job from database")

	// Start database transaction for atomic operations
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start database transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Manual cleanup for SET NULL relationships (failover_jobs)
	// Using VERIFIED field name: replication_job_id
	result := tx.WithContext(ctx).Exec("UPDATE failover_jobs SET replication_job_id = NULL WHERE replication_job_id = ?", jobID)
	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update failover jobs: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log.WithField("job_id", jobID).WithField("failover_jobs_updated", result.RowsAffected).Info("Updated failover jobs to NULL reference")
	}

	// Delete main job record (triggers CASCADE DELETE for dependent records)
	// This will automatically delete: vm_disks, volume_mounts (per VERIFIED FK constraints)
	result = tx.WithContext(ctx).Exec("DELETE FROM replication_jobs WHERE id = ?", jobID)
	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete replication job: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("job not found in database")
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit database transaction: %w", err)
	}

	log.WithField("job_id", jobID).Info("✅ Replication job and related records deleted from database")
	return nil
}

// Create saves a new replication job to the database using VERIFIED schema
// VM-Centric Enhancement: Also creates/updates VM context automatically
func (r *ReplicationJobRepository) Create(ctx context.Context, job *ReplicationJob) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithField("job_id", job.ID).Info("Creating replication job in database with VM context")

	// Start transaction for atomic job + context creation
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer tx.Rollback()

	// Step 1: Create or update VM context
	vmContextID, err := r.createOrUpdateVMContext(tx, job)
	if err != nil {
		return fmt.Errorf("failed to create/update VM context: %w", err)
	}

	// Step 2: Link job to VM context
	job.VMContextID = vmContextID

	// Step 3: Create the replication job
	err = tx.Create(job).Error
	if err != nil {
		return fmt.Errorf("failed to create replication job: %w", err)
	}

	// Step 4: Update VM context with new job reference
	err = r.updateVMContextAfterJobCreation(tx, vmContextID, job)
	if err != nil {
		return fmt.Errorf("failed to update VM context after job creation: %w", err)
	}

	// Commit transaction
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id":        job.ID,
		"vm_context_id": vmContextID,
		"vm_name":       job.SourceVMName,
	}).Info("✅ Replication job and VM context created successfully")

	return nil
}

// Update saves changes to an existing replication job using VERIFIED schema
func (r *ReplicationJobRepository) Update(ctx context.Context, job *ReplicationJob) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithField("job_id", job.ID).Debug("Updating replication job in database")

	err := r.db.WithContext(ctx).Save(job).Error
	if err != nil {
		return fmt.Errorf("failed to update replication job: %w", err)
	}

	log.WithField("job_id", job.ID).Debug("✅ Replication job updated successfully")
	return nil
}

// VM-Centric Helper Methods

// createOrUpdateVMContext finds or creates a VM context for the given job
func (r *ReplicationJobRepository) createOrUpdateVMContext(tx *gorm.DB, job *ReplicationJob) (string, error) {
	// Try to find existing context by VM identifier and vCenter
	var existingContext struct {
		ContextID string `gorm:"column:context_id"`
	}

	err := tx.Table("vm_replication_contexts").
		Where("vm_name = ? AND vcenter_host = ?", job.SourceVMName, job.VCenterHost).
		First(&existingContext).Error

	if err == nil {
		// Context exists, return its ID
		log.WithFields(log.Fields{
			"vm_name":      job.SourceVMName,
			"vcenter_host": job.VCenterHost,
			"context_id":   existingContext.ContextID,
		}).Debug("Found existing VM context")
		return existingContext.ContextID, nil
	}

	// Context doesn't exist, create new one
	contextID := "ctx-" + job.SourceVMName + "-" + time.Now().Format("20060102-150405")

	// Extract VM specifications from job if available
	vmContext := map[string]interface{}{
		"context_id":      contextID,
		"vm_name":         job.SourceVMName,
		"vmware_vm_id":    job.SourceVMID,
		"vm_path":         job.SourceVMPath,
		"vcenter_host":    job.VCenterHost,
		"datacenter":      job.Datacenter,
		"current_status":  "discovered",
		"total_jobs_run":  0,
		"successful_jobs": 0,
		"failed_jobs":     0,
		"created_at":      time.Now(),
		"updated_at":      time.Now(),
		"first_job_at":    time.Now(),
	}

	// Note: VM specifications (CPU, memory, etc.) will be populated
	// from VMDisk records during VM discovery process

	err = tx.Table("vm_replication_contexts").Create(vmContext).Error
	if err != nil {
		return "", fmt.Errorf("failed to create VM context: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_name":      job.SourceVMName,
		"vcenter_host": job.VCenterHost,
		"context_id":   contextID,
	}).Info("✅ Created new VM context")

	return contextID, nil
}

// updateVMContextAfterJobCreation updates VM context statistics after a new job is created
func (r *ReplicationJobRepository) updateVMContextAfterJobCreation(tx *gorm.DB, vmContextID string, job *ReplicationJob) error {
	// Update VM context with job reference and increment counters
	updates := map[string]interface{}{
		"current_job_id":     job.ID,
		"total_jobs_run":     gorm.Expr("total_jobs_run + 1"),
		"current_status":     "replicating", // New job starts in replicating state
		"last_job_at":        time.Now(),
		"last_status_change": time.Now(),
		"updated_at":         time.Now(),
	}

	err := tx.Table("vm_replication_contexts").
		Where("context_id = ?", vmContextID).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update VM context after job creation: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_context_id": vmContextID,
		"job_id":        job.ID,
	}).Debug("Updated VM context after job creation")

	return nil
}

// UpdateVMContextAfterJobCompletion updates VM context status when jobs complete/fail
func (r *ReplicationJobRepository) UpdateVMContextAfterJobCompletion(jobID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	// Start transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer tx.Rollback()

	// Find the job and its VM context
	var job ReplicationJob
	err := tx.Where("id = ?", jobID).First(&job).Error
	if err != nil {
		return fmt.Errorf("failed to find job %s: %w", jobID, err)
	}

	if job.VMContextID == "" {
		// No context to update (legacy job)
		log.WithField("job_id", jobID).Debug("Job has no VM context - skipping context update")
		return nil
	}

	// Get all jobs for this VM context
	var allJobs []ReplicationJob
	err = tx.Where("vm_context_id = ?", job.VMContextID).
		Order("created_at DESC").
		Find(&allJobs).Error
	if err != nil {
		return fmt.Errorf("failed to get jobs for context: %w", err)
	}

	// Calculate new context state
	var newCurrentJobID *string = nil
	var newStatus string = "discovered"
	var newLastSuccessfulJobID *string = nil
	var totalJobs = len(allJobs)
	var successfulJobs = 0
	var failedJobs = 0

	// Find latest active job, count job types, and find last successful job
	for _, j := range allJobs {
		switch j.Status {
		case "completed":
			successfulJobs++
			// Track the most recent successful job (jobs are ordered by created_at DESC)
			if newLastSuccessfulJobID == nil {
				newLastSuccessfulJobID = &j.ID
			}
		case "failed":
			failedJobs++
		case "replicating", "provisioning":
			if newCurrentJobID == nil { // First active job found
				newCurrentJobID = &j.ID
				newStatus = "replicating"
			}
		}
	}

	// If no active jobs, determine status based on latest job
	if newCurrentJobID == nil && len(allJobs) > 0 {
		latestJob := allJobs[0] // Already ordered by created_at DESC
		switch latestJob.Status {
		case "completed":
			newStatus = "ready_for_failover"
		case "failed":
			newStatus = "failed"
		default:
			newStatus = "discovered"
		}
	}

	// Update the VM context
	err = tx.Table("vm_replication_contexts").
		Where("context_id = ?", job.VMContextID).
		Updates(map[string]interface{}{
			"current_job_id":         newCurrentJobID,
			"last_successful_job_id": newLastSuccessfulJobID,
			"total_jobs_run":         totalJobs,
			"successful_jobs":        successfulJobs,
			"failed_jobs":            failedJobs,
			"current_status":         newStatus,
			"last_status_change":     time.Now(),
			"updated_at":             time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update VM context: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_context_id":          job.VMContextID,
		"job_id":                 jobID,
		"new_status":             newStatus,
		"total_jobs":             totalJobs,
		"successful":             successfulJobs,
		"failed":                 failedJobs,
		"current_job_id":         newCurrentJobID,
		"last_successful_job_id": newLastSuccessfulJobID,
	}).Info("Updated VM context after job completion")

	return tx.Commit().Error
}

// =============================================================================
// VM CONTEXT REPOSITORY - GUI Integration
// =============================================================================

// VMReplicationContextRepository handles VM context operations
type VMReplicationContextRepository struct {
	db *gorm.DB
}

// NewVMReplicationContextRepository creates a new VM context repository
func NewVMReplicationContextRepository(conn Connection) *VMReplicationContextRepository {
	return &VMReplicationContextRepository{
		db: conn.GetGormDB(),
	}
}

// VMContextDetails represents complete VM context information for GUI
type VMContextDetails struct {
	Context         VMReplicationContext `json:"context"`
	CurrentJob      *ReplicationJob      `json:"current_job,omitempty"`
	JobHistory      []ReplicationJob     `json:"job_history"`
	Disks           []VMDisk             `json:"disks"`
	CBTHistory      []CBTHistory         `json:"cbt_history"`
	LastOperation   map[string]interface{} `json:"last_operation,omitempty"` // Parsed operation summary
}

// GetVMContextWithFullDetails retrieves complete VM context information for GUI
func (r *VMReplicationContextRepository) GetVMContextWithFullDetails(vmName string) (*VMContextDetails, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Get the VM context
	var context VMReplicationContext
	if err := r.db.Where("vm_name = ?", vmName).First(&context).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("VM context not found for: %s", vmName)
		}
		return nil, fmt.Errorf("failed to get VM context for %s: %w", vmName, err)
	}

	result := &VMContextDetails{
		Context: context,
	}

	// Get current job if exists
	if context.CurrentJobID != nil && *context.CurrentJobID != "" {
		var currentJob ReplicationJob
		if err := r.db.Where("id = ?", *context.CurrentJobID).First(&currentJob).Error; err == nil {
			result.CurrentJob = &currentJob
		}
	}

	// Get job history (last 10 jobs)
	var jobHistory []ReplicationJob
	if err := r.db.Where("vm_context_id = ?", context.ContextID).
		Order("created_at DESC").
		Limit(10).
		Find(&jobHistory).Error; err != nil {
		log.WithError(err).Warn("Failed to get job history")
	} else {
		result.JobHistory = jobHistory
	}

	// Get VM disks for the most recent job
	if len(jobHistory) > 0 {
		var disks []VMDisk
		if err := r.db.Where("vm_context_id = ?", context.ContextID).
			Order("created_at DESC").
			Find(&disks).Error; err != nil {
			log.WithError(err).Warn("Failed to get VM disks")
		} else {
			result.Disks = disks
		}
	}

	// Get CBT history (last 20 records)
	var cbtHistory []CBTHistory
	if err := r.db.Where("vm_context_id = ?", context.ContextID).
		Order("created_at DESC").
		Limit(20).
		Find(&cbtHistory).Error; err != nil {
		log.WithError(err).Warn("Failed to get CBT history")
	} else {
		result.CBTHistory = cbtHistory
	}

	log.WithFields(log.Fields{
		"vm_name":           vmName,
		"context_id":        context.ContextID,
		"current_job":       context.CurrentJobID,
		"job_history_count": len(result.JobHistory),
		"disks_count":       len(result.Disks),
		"cbt_history_count": len(result.CBTHistory),
	}).Debug("Retrieved VM context with full details")

	return result, nil
}

// ListVMContexts retrieves all VM contexts with summary information
func (r *VMReplicationContextRepository) ListVMContexts() ([]VMReplicationContext, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var contexts []VMReplicationContext
	if err := r.db.Order("vm_name ASC").Find(&contexts).Error; err != nil {
		return nil, fmt.Errorf("failed to list VM contexts: %w", err)
	}

	log.WithField("count", len(contexts)).Debug("Listed VM contexts")
	return contexts, nil
}

// =============================================================================
// REPLICATION JOB REPOSITORY - Additional Methods for Phantom Detection
// =============================================================================

// GetJobsByStatus retrieves jobs by their current status
func (r *ReplicationJobRepository) GetJobsByStatus(statuses []string) ([]ReplicationJob, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var jobs []ReplicationJob
	err := r.db.Where("status IN ?", statuses).Find(&jobs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs by status: %w", err)
	}

	log.WithFields(log.Fields{
		"statuses": statuses,
		"count":    len(jobs),
	}).Debug("Retrieved jobs by status")

	return jobs, nil
}

// UpdateJobFields updates specific fields of a replication job
func (r *ReplicationJobRepository) UpdateJobFields(jobID string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	err := r.db.Model(&ReplicationJob{}).Where("id = ?", jobID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update job fields: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id": jobID,
		"fields": len(updates),
	}).Debug("Updated job fields")

	return nil
}

// CreateVMContext creates a new VM replication context
func (r *VMReplicationContextRepository) CreateVMContext(context *VMReplicationContext) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	if context == nil {
		return fmt.Errorf("VM context cannot be nil")
	}

	// Validate required fields
	if context.VMName == "" {
		return fmt.Errorf("VM name is required")
	}
	if context.VMwareVMID == "" {
		return fmt.Errorf("VMware VM ID is required")
	}
	if context.VMPath == "" {
		return fmt.Errorf("VM path is required")
	}
	if context.VCenterHost == "" {
		return fmt.Errorf("vCenter host is required")
	}
	if context.Datacenter == "" {
		return fmt.Errorf("datacenter is required")
	}

	// Set defaults if not provided
	if context.CurrentStatus == "" {
		context.CurrentStatus = "discovered"
	}

	// Create the context in database
	if err := r.db.Create(context).Error; err != nil {
		return fmt.Errorf("failed to create VM context: %w", err)
	}

	return nil
}

// GetVMContextByName retrieves a VM context by VM name
func (r *VMReplicationContextRepository) GetVMContextByName(vmName string) (*VMReplicationContext, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	if vmName == "" {
		return nil, fmt.Errorf("VM name is required")
	}

	var context VMReplicationContext
	if err := r.db.Where("vm_name = ?", vmName).First(&context).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Not found, but not an error
		}
		return nil, fmt.Errorf("failed to get VM context for %s: %w", vmName, err)
	}

	return &context, nil
}

// UpdateVMContextStatus updates the current status of a VM context
func (r *VMReplicationContextRepository) UpdateVMContextStatus(contextID, newStatus string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	if contextID == "" {
		return fmt.Errorf("context ID is required")
	}

	if newStatus == "" {
		return fmt.Errorf("new status is required")
	}

	// Update the status and last_status_change timestamp
	result := r.db.Model(&VMReplicationContext{}).
		Where("context_id = ?", contextID).
		Updates(map[string]interface{}{
			"current_status":     newStatus,
			"last_status_change": time.Now(),
			"updated_at":         time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update VM context status for %s: %w", contextID, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("VM context not found: %s", contextID)
	}

	return nil
}

// =============================================================================
// OSSEA CONFIG REPOSITORY - Encryption and Validation Helpers
// =============================================================================

// encryptCredentials encrypts sensitive fields in the OSSEA configuration
func (r *OSSEAConfigRepository) encryptCredentials(config *OSSEAConfig) error {
	if config.APIKey != "" {
		encrypted, err := r.encryptionService.EncryptPassword(config.APIKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt API key: %w", err)
		}
		config.APIKey = encrypted
	}

	if config.SecretKey != "" {
		encrypted, err := r.encryptionService.EncryptPassword(config.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret key: %w", err)
		}
		config.SecretKey = encrypted
	}

	return nil
}

// decryptCredentials decrypts sensitive fields in the OSSEA configuration
func (r *OSSEAConfigRepository) decryptCredentials(config *OSSEAConfig) error {
	if config.APIKey != "" {
		decrypted, err := r.encryptionService.DecryptPassword(config.APIKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt API key: %w", err)
		}
		config.APIKey = decrypted
	}

	if config.SecretKey != "" {
		decrypted, err := r.encryptionService.DecryptPassword(config.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt secret key: %w", err)
		}
		config.SecretKey = decrypted
	}

	return nil
}

// ValidateConfig validates OSSEA configuration fields
func (r *OSSEAConfigRepository) ValidateConfig(config *OSSEAConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	// Validate required fields
	if config.Name == "" {
		return fmt.Errorf("configuration name is required")
	}

	if config.APIURL == "" {
		return fmt.Errorf("API URL is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if config.SecretKey == "" {
		return fmt.Errorf("secret key is required")
	}

	if config.Zone == "" {
		return fmt.Errorf("zone is required")
	}

	// Validate API URL format (basic check for http/https)
	if config.APIURL != "" {
		if !isValidURL(config.APIURL) {
			return fmt.Errorf("invalid API URL format: must start with http:// or https://")
		}
	}

	// Note: Network ID, Service Offering ID, and Disk Offering ID validation
	// can be optionally empty during initial configuration

	log.WithField("config_name", config.Name).Debug("Configuration validation passed")
	return nil
}

// isValidURL performs basic URL validation
func isValidURL(url string) bool {
	if url == "" {
		return false
	}
	// Basic check: must start with http:// or https://
	if len(url) >= 7 && (url[:7] == "http://" || url[:8] == "https://") {
		return true
	}
	return false
}
