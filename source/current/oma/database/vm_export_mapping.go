package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// VMExportMapping represents the persistent mapping between VMware VMs and NBD exports
// This enables export reuse to prevent unnecessary SIGHUP operations
type VMExportMapping struct {
	ID               int64     `db:"id" json:"id"`
	VMID             string    `db:"vm_id" json:"vm_id"`                         // VMware VM UUID
	DiskUnitNumber   int       `db:"disk_unit_number" json:"disk_unit_number"`   // SCSI unit number (0,1,2...)
	VMName           string    `db:"vm_name" json:"vm_name"`                     // VMware VM name for reference
	ExportName       string    `db:"export_name" json:"export_name"`             // NBD export name
	DevicePath       string    `db:"device_path" json:"device_path"`             // Linux device path (/dev/vdb, etc.)
	Status           string    `db:"status" json:"status"`                       // active/inactive
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

// VMExportMappingRepository provides database operations for VM export mappings
type VMExportMappingRepository struct {
	db *gorm.DB
}

// NewVMExportMappingRepository creates a new repository instance
func NewVMExportMappingRepository(conn Connection) *VMExportMappingRepository {
	return &VMExportMappingRepository{db: conn.GetGormDB()}
}

// FindByVMAndDisk finds an existing mapping for a specific VM and disk
func (r *VMExportMappingRepository) FindByVMAndDisk(vmID string, diskUnitNumber int) (*VMExportMapping, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var mapping VMExportMapping
	err := r.db.Where("vm_id = ? AND disk_unit_number = ? AND status = ?", vmID, diskUnitNumber, "active").First(&mapping).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No record found is not an error in this context
		}
		return nil, fmt.Errorf("failed to find VM export mapping: %w", err)
	}
	return &mapping, nil
}

// Create creates a new VM export mapping
func (r *VMExportMappingRepository) Create(mapping *VMExportMapping) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	if err := r.db.Create(mapping).Error; err != nil {
		return fmt.Errorf("failed to create VM export mapping: %w", err)
	}
	return nil
}

// GetNextAvailableDevice finds the next available device path
func (r *VMExportMappingRepository) GetNextAvailableDevice() (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("database not available")
	}

	// Query for all currently used devices
	var usedDevices []string
	err := r.db.Model(&VMExportMapping{}).Where("status = ?", "active").Pluck("device_path", &usedDevices).Error
	if err != nil {
		return "", fmt.Errorf("failed to query used devices: %w", err)
	}
	
	// Available device sequence: /dev/vdb, /dev/vdc, /dev/vdd, /dev/vde, ...
	deviceLetters := "bcdefghijklmnopqrstuvwxyz"
	
	for _, letter := range deviceLetters {
		candidate := "/dev/vd" + string(letter)
		
		// Check if this device is already in use
		inUse := false
		for _, used := range usedDevices {
			if used == candidate {
				inUse = true
				break
			}
		}
		
		if !inUse {
			return candidate, nil
		}
	}
	
	return "", fmt.Errorf("no available device paths - all devices in use")
}

// FindConflictingExports finds exports on the same device from different VMs
func (r *VMExportMappingRepository) FindConflictingExports(vmID, devicePath string) ([]VMExportMapping, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var conflicts []VMExportMapping
	err := r.db.Where("device_path = ? AND vm_id != ? AND status = ?", devicePath, vmID, "active").Find(&conflicts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find conflicting exports: %w", err)
	}
	return conflicts, nil
}

// DeactivateMapping marks a mapping as inactive
func (r *VMExportMappingRepository) DeactivateMapping(id int64) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	err := r.db.Model(&VMExportMapping{}).Where("id = ?", id).Update("status", "inactive").Error
	if err != nil {
		return fmt.Errorf("failed to deactivate mapping: %w", err)
	}
	return nil
}
