package storage

import (
	"context"
	"time"
)

// Repository defines the interface for backup storage backends.
// Implementations can store backups on local disk, NFS, CIFS, S3, Azure Blob, etc.
type Repository interface {
	// CreateBackup creates a new backup in the repository.
	// Returns a Backup object with ID and file path.
	CreateBackup(ctx context.Context, req BackupRequest) (*Backup, error)

	// GetBackup retrieves backup metadata by ID.
	GetBackup(ctx context.Context, backupID string) (*Backup, error)

	// ListBackups lists all backups for a VM context.
	ListBackups(ctx context.Context, vmContextID string) ([]*Backup, error)

	// DeleteBackup removes a backup from the repository.
	// Will fail if backup is part of a chain and has dependents.
	DeleteBackup(ctx context.Context, backupID string) error

	// GetBackupChain retrieves the complete backup chain for a VM disk.
	GetBackupChain(ctx context.Context, vmContextID string, diskID int) (*BackupChain, error)

	// GetStorageInfo returns current storage capacity and usage.
	GetStorageInfo(ctx context.Context) (*StorageInfo, error)

	// GetExportPath returns the file system path for a backup (for NBD export).
	GetExportPath(ctx context.Context, backupID string) (string, error)
}

// BackupRequest encapsulates parameters for creating a backup.
type BackupRequest struct {
	VMContextID    string         `json:"vm_context_id"`
	VMName         string         `json:"vm_name"`
	DiskID         int            `json:"disk_id"`
	BackupType     BackupType     `json:"backup_type"`
	ParentBackupID string         `json:"parent_backup_id,omitempty"` // For incrementals
	TotalBytes     int64          `json:"total_bytes"`
	ChangeID       string         `json:"change_id,omitempty"` // VMware CBT change ID
	Metadata       BackupMetadata `json:"metadata"`
}

// Backup represents a single backup in the repository.
type Backup struct {
	ID             string       `json:"id"`
	VMContextID    string       `json:"vm_context_id"`
	VMName         string       `json:"vm_name"`
	DiskID         int          `json:"disk_id"`
	BackupType     BackupType   `json:"backup_type"`
	Status         BackupStatus `json:"status"`
	ParentBackupID string       `json:"parent_backup_id,omitempty"`
	ChangeID       string       `json:"change_id,omitempty"`
	FilePath       string       `json:"file_path"`        // Actual file path on disk
	SizeBytes      int64        `json:"size_bytes"`       // Actual file size
	TotalBytes     int64        `json:"total_bytes"`      // VM disk total size
	CreatedAt      time.Time    `json:"created_at"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	ErrorMessage   string       `json:"error_message,omitempty"`
}

// BackupChain represents a full backup plus its incrementals.
type BackupChain struct {
	ID              string    `json:"id"`
	VMContextID     string    `json:"vm_context_id"`
	DiskID          int       `json:"disk_id"`
	FullBackupID    string    `json:"full_backup_id"`
	LatestBackupID  string    `json:"latest_backup_id"`
	Backups         []*Backup `json:"backups"` // Ordered: full first, then incrementals
	TotalBackups    int       `json:"total_backups"`
	TotalSizeBytes  int64     `json:"total_size_bytes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// StorageInfo provides repository capacity information.
type StorageInfo struct {
	RepositoryID   string    `json:"repository_id"`
	TotalBytes     int64     `json:"total_bytes"`
	UsedBytes      int64     `json:"used_bytes"`
	AvailableBytes int64     `json:"available_bytes"`
	UsedPercent    float64   `json:"used_percent"`
	BackupCount    int       `json:"backup_count"`
	LastCheckAt    time.Time `json:"last_check_at"`
}

// BackupType defines the type of backup operation.
type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
	BackupTypeDifferential BackupType = "differential" // Future
)

// BackupStatus defines the current state of a backup.
type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
	BackupStatusCancelled BackupStatus = "cancelled"
)

// BackupMetadata contains platform-specific backup information.
type BackupMetadata struct {
	VMwareInfo      *VMwareMetadata      `json:"vmware_info,omitempty"`
	CloudStackInfo  *CloudStackMetadata  `json:"cloudstack_info,omitempty"`
	HyperVInfo      *HyperVMetadata      `json:"hyperv_info,omitempty"`
	AWSInfo         *AWSMetadata         `json:"aws_info,omitempty"`
	CustomMetadata  map[string]string    `json:"custom_metadata,omitempty"`
}

// VMwareMetadata contains VMware-specific backup information.
type VMwareMetadata struct {
	VCenterUUID   string `json:"vcenter_uuid"`
	VMwareVMID    string `json:"vmware_vm_id"`
	VMwareDiskID  string `json:"vmware_disk_id"`
	DiskPath      string `json:"disk_path"`
	DatastoreName string `json:"datastore_name"`
	ChangeID      string `json:"change_id"`      // CBT change ID
	CBTEnabled    bool   `json:"cbt_enabled"`
}

// CloudStackMetadata contains CloudStack-specific information.
type CloudStackMetadata struct {
	ZoneID      string `json:"zone_id"`
	VMID        string `json:"vm_id"`
	VolumeID    string `json:"volume_id"`
	VolumeName  string `json:"volume_name"`
	VolumeType  string `json:"volume_type"`
}

// HyperVMetadata contains Hyper-V-specific information.
type HyperVMetadata struct {
	VMID         string `json:"vm_id"`
	VHDID        string `json:"vhd_id"`
	VHDPath      string `json:"vhd_path"`
	CheckpointID string `json:"checkpoint_id,omitempty"` // RCT checkpoint
}

// AWSMetadata contains AWS EC2-specific information.
type AWSMetadata struct {
	Region          string `json:"region"`
	InstanceID      string `json:"instance_id"`
	VolumeID        string `json:"volume_id"`
	SnapshotID      string `json:"snapshot_id,omitempty"`
	EBSChangeToken  string `json:"ebs_change_token,omitempty"` // EBS direct APIs
}

