package storage

import (
	"time"
)

// RepositoryType defines the type of storage backend.
type RepositoryType string

const (
	RepositoryTypeLocal RepositoryType = "local"
	RepositoryTypeNFS   RepositoryType = "nfs"
	RepositoryTypeCIFS  RepositoryType = "cifs"
	RepositoryTypeSMB   RepositoryType = "smb"
	RepositoryTypeS3    RepositoryType = "s3"    // Future
	RepositoryTypeAzure RepositoryType = "azure" // Future
)

// RepositoryConfig defines configuration for a backup repository.
type RepositoryConfig struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Type               RepositoryType         `json:"type"`
	Enabled            bool                   `json:"enabled"`
	Config             interface{}            `json:"config"` // Type-specific config
	IsImmutable        bool                   `json:"is_immutable"`
	ImmutableConfig    *ImmutableConfig       `json:"immutable_config,omitempty"`
	MinRetentionDays   int                    `json:"min_retention_days"`
	TotalBytes         int64                  `json:"total_bytes"`
	UsedBytes          int64                  `json:"used_bytes"`
	AvailableBytes     int64                  `json:"available_bytes"`
	LastCheckAt        *time.Time             `json:"last_check_at,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// LocalRepositoryConfig defines configuration for local disk storage.
type LocalRepositoryConfig struct {
	Path         string `json:"path"`          // /var/lib/sendense/backups
	AutoMount    bool   `json:"auto_mount"`    // Mount at startup
	MountOptions string `json:"mount_options"` // defaults,noatime
}

// NFSRepositoryConfig defines configuration for NFS storage.
type NFSRepositoryConfig struct {
	Server       string `json:"server"`        // 10.0.100.50
	ExportPath   string `json:"export_path"`   // /exports/backups
	MountPoint   string `json:"mount_point"`   // /mnt/nfs-backups
	NFSVersion   string `json:"nfs_version"`   // 4.1
	MountOptions string `json:"mount_options"` // defaults,soft,timeo=30
}

// CIFSRepositoryConfig defines configuration for CIFS/SMB storage.
type CIFSRepositoryConfig struct {
	Server         string `json:"server"`           // fileserver.local
	ShareName      string `json:"share_name"`       // backups
	MountPoint     string `json:"mount_point"`      // /mnt/smb-backups
	Username       string `json:"username"`         // Optional
	PasswordSecret string `json:"password_secret"`  // Encrypted reference
	Domain         string `json:"domain"`           // Optional
	MountOptions   string `json:"mount_options"`    // vers=3.0,iocharset=utf8
}

// S3RepositoryConfig defines configuration for S3 storage (future).
type S3RepositoryConfig struct {
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix"`
	AccessKeyID     string `json:"access_key_id"`
	SecretKeySecret string `json:"secret_key_secret"`
	UseObjectLock   bool   `json:"use_object_lock"` // Immutability
}

// AzureRepositoryConfig defines configuration for Azure Blob storage (future).
type AzureRepositoryConfig struct {
	AccountName        string `json:"account_name"`
	ContainerName      string `json:"container_name"`
	AccountKeySecret   string `json:"account_key_secret"`
	UseImmutableBlobs  bool   `json:"use_immutable_blobs"` // WORM
}

// ImmutableConfig defines immutability settings for a repository.
type ImmutableConfig struct {
	Type             ImmutableType     `json:"type"`
	MinRetentionDays int               `json:"min_retention_days"`
	Config           interface{}       `json:"config"`
}

// ImmutableType defines the type of immutability implementation.
type ImmutableType string

const (
	ImmutableTypeLinuxChattr ImmutableType = "linux_chattr"    // chattr +i flag
	ImmutableTypeS3Lock      ImmutableType = "s3_object_lock"  // S3 Object Lock
	ImmutableTypeAzureWORM   ImmutableType = "azure_worm"      // Azure immutable blob
)

// LinuxImmutableConfig defines Linux filesystem immutability config.
type LinuxImmutableConfig struct {
	GracePeriodDays int    `json:"grace_period_days"` // Days before applying immutability
	UnlockCommand   string `json:"unlock_command"`    // Command to unlock (admin only)
}

// BackupPolicy defines a backup policy with copy rules.
type BackupPolicy struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Enabled             bool              `json:"enabled"`
	PrimaryRepositoryID string            `json:"primary_repository_id"`
	CopyRules           []*BackupCopyRule `json:"copy_rules"`
	RetentionDays       int               `json:"retention_days"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// BackupCopyRule defines a rule for copying backups to another repository.
type BackupCopyRule struct {
	ID                      string   `json:"id"`
	DestinationRepositoryID string   `json:"destination_repository_id"`
	CopyMode                CopyMode `json:"copy_mode"`
	Priority                int      `json:"priority"` // Order of copies (1, 2, 3)
	Enabled                 bool     `json:"enabled"`
	VerifyAfterCopy         bool     `json:"verify_after_copy"`
}

// CopyMode defines when backup copies are created.
type CopyMode string

const (
	CopyModeImmediate CopyMode = "immediate"  // Copy as soon as backup completes
	CopyModeScheduled CopyMode = "scheduled"  // Copy during off-peak hours
	CopyModeManual    CopyMode = "manual"     // User-triggered
)

// BackupCopy tracks a copy of a backup to another repository.
type BackupCopy struct {
	ID                   string             `json:"id"`
	SourceBackupID       string             `json:"source_backup_id"`
	RepositoryID         string             `json:"repository_id"`
	CopyRuleID           string             `json:"copy_rule_id,omitempty"`
	Status               BackupCopyStatus   `json:"status"`
	FilePath             string             `json:"file_path"`
	SizeBytes            int64              `json:"size_bytes"`
	CopyStartedAt        *time.Time         `json:"copy_started_at,omitempty"`
	CopyCompletedAt      *time.Time         `json:"copy_completed_at,omitempty"`
	VerifiedAt           *time.Time         `json:"verified_at,omitempty"`
	VerificationStatus   VerificationStatus `json:"verification_status"`
	ErrorMessage         string             `json:"error_message,omitempty"`
	CreatedAt            time.Time          `json:"created_at"`
}

// BackupCopyStatus defines the state of a backup copy operation.
type BackupCopyStatus string

const (
	BackupCopyStatusPending   BackupCopyStatus = "pending"
	BackupCopyStatusCopying   BackupCopyStatus = "copying"
	BackupCopyStatusVerifying BackupCopyStatus = "verifying"
	BackupCopyStatusCompleted BackupCopyStatus = "completed"
	BackupCopyStatusFailed    BackupCopyStatus = "failed"
)

// VerificationStatus defines the verification state of a backup copy.
type VerificationStatus string

const (
	VerificationStatusPending VerificationStatus = "pending"
	VerificationStatusPassed  VerificationStatus = "passed"
	VerificationStatusFailed  VerificationStatus = "failed"
)
