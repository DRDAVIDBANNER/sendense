// Package nbd provides NBD server management models
package nbd

import (
	"time"
)

// Export represents an NBD export record in the database
type Export struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	JobID      string    `gorm:"not null;index" json:"job_id"`             // Foreign key to replication_jobs.id
	VolumeID   string    `gorm:"not null;index" json:"volume_id"`          // Foreign key to ossea_volumes.volume_id
	ExportName string    `gorm:"not null;unique" json:"export_name"`       // NBD export name
	Port       int       `gorm:"not null" json:"port"`                     // NBD server port
	DevicePath string    `gorm:"not null" json:"device_path"`              // Block device path (e.g., /dev/vdb)
	ConfigPath string    `gorm:"not null" json:"config_path"`              // NBD config file path
	Status     string    `gorm:"not null;default:'pending'" json:"status"` // pending, active, stopped, error
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName returns the table name for NBD exports
func (Export) TableName() string {
	return "nbd_exports"
}

// NBDStatus constants for export status tracking
const (
	StatusPending = "pending" // Configuration created but server not started
	StatusActive  = "active"  // NBD server running and accessible
	StatusStopped = "stopped" // NBD server stopped/completed
	StatusError   = "error"   // NBD server failed or configuration error
)

// FileExport represents a file-based NBD export (for QCOW2 backups)
type FileExport struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	VMContextID   string    `gorm:"not null;index" json:"vm_context_id"`      // VM context identifier
	DiskID        int       `gorm:"not null" json:"disk_id"`                  // Disk number (0, 1, 2...)
	BackupType    string    `gorm:"not null" json:"backup_type"`              // full, incr
	ExportName    string    `gorm:"not null;unique" json:"export_name"`       // NBD export name (unique)
	FilePath      string    `gorm:"not null" json:"file_path"`                // QCOW2 file path
	FileSize      int64     `gorm:"not null" json:"file_size"`                // File size in bytes
	Port          int       `gorm:"not null" json:"port"`                     // NBD server port
	ReadWrite     bool      `gorm:"not null;default:false" json:"read_write"` // true for incremental writes
	Status        string    `gorm:"not null;default:'pending'" json:"status"` // pending, active, stopped, error
	BackupChainID string    `gorm:"index" json:"backup_chain_id"`             // Link to backup chain
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName returns the table name for file-based NBD exports
func (FileExport) TableName() string {
	return "nbd_file_exports"
}

