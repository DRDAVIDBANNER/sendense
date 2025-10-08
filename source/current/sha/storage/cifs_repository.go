package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// CIFSRepository implements Repository interface for CIFS/SMB-backed storage.
// Design: Embeds LocalRepository for all backup operations, adds CIFS mount management with authentication.
type CIFSRepository struct {
	*LocalRepository              // Embedded for backup operations
	mountManager     *MountManager
	cifsConfig       *CIFSRepositoryConfig
	repositoryConfig *RepositoryConfig
	mountPoint       string
	mounted          bool
	mu               sync.RWMutex // Protects mount state
}

// NewCIFSRepository creates a new CIFS/SMB-backed repository.
// The repository will mount the CIFS share on first use and manage the mount lifecycle.
// Handles authentication with username, password, and optional domain.
func NewCIFSRepository(config *RepositoryConfig, db *sql.DB, mountManager *MountManager) (*CIFSRepository, error) {
	// Parse CIFS config
	cifsConfig, ok := config.Config.(CIFSRepositoryConfig)
	if !ok {
		// Try type assertion from map[string]interface{} (JSON unmarshaling case)
		if err := parseConfig(config.Config, &cifsConfig); err != nil {
			return nil, &RepositoryError{
				RepositoryID: config.ID,
				Op:           "parse_config",
				Err:          fmt.Errorf("invalid CIFS repository config: %w", err),
			}
		}
	}

	// Validate CIFS config
	if cifsConfig.Server == "" {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "validate_config",
			Err:          fmt.Errorf("CIFS server cannot be empty"),
		}
	}
	if cifsConfig.ShareName == "" {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "validate_config",
			Err:          fmt.Errorf("CIFS share name cannot be empty"),
		}
	}
	if cifsConfig.MountPoint == "" {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "validate_config",
			Err:          fmt.Errorf("CIFS mount point cannot be empty"),
		}
	}

	// Create a modified config for LocalRepository with mount point as basePath
	localConfig := &RepositoryConfig{
		ID:               config.ID,
		Name:             config.Name,
		Type:             config.Type,
		Enabled:          config.Enabled,
		IsImmutable:      config.IsImmutable,
		ImmutableConfig:  config.ImmutableConfig,
		MinRetentionDays: config.MinRetentionDays,
		CreatedAt:        config.CreatedAt,
		UpdatedAt:        config.UpdatedAt,
		Config: LocalRepositoryConfig{
			Path: cifsConfig.MountPoint,
		},
	}

	// Create embedded LocalRepository (mount point must exist, will be created during mount)
	localRepo, err := NewLocalRepository(localConfig, db)
	if err != nil {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "create_local_repository",
			Err:          fmt.Errorf("failed to create local repository: %w", err),
		}
	}

	cifsRepo := &CIFSRepository{
		LocalRepository:  localRepo,
		mountManager:     mountManager,
		cifsConfig:       &cifsConfig,
		repositoryConfig: config,
		mountPoint:       cifsConfig.MountPoint,
		mounted:          false,
	}

	return cifsRepo, nil
}

// ensureMounted checks if CIFS is mounted and mounts it if not.
// This is called before any operation that requires storage access.
// Handles authentication with username, password, and optional domain.
func (cr *CIFSRepository) ensureMounted(ctx context.Context) error {
	// Fast path: check if already mounted without lock
	cr.mu.RLock()
	if cr.mounted {
		cr.mu.RUnlock()
		return nil
	}
	cr.mu.RUnlock()

	// Slow path: acquire write lock and mount
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Double-check after acquiring lock
	if cr.mounted {
		return nil
	}

	// Check if already mounted in system (could have been mounted externally)
	isMounted := cr.mountManager.IsMounted(cr.mountPoint)

	if isMounted {
		// Already mounted (externally), just mark as mounted
		cr.mounted = true
		return nil
	}

	// Parse mount options
	mountOpts := []string{"rw"} // Default options
	if cr.cifsConfig.MountOptions != "" {
		// Parse comma-separated options
		mountOpts = parseMountOptions(cr.cifsConfig.MountOptions)
	}

	// Prepare mount configuration with credentials
	mountConfig := CIFSMountConfig{
		Server:       cr.cifsConfig.Server,
		ShareName:    cr.cifsConfig.ShareName,
		MountPoint:   cr.mountPoint,
		Username:     cr.cifsConfig.Username,
		Password:     cr.resolvePassword(ctx), // Resolve password from secret
		Domain:       cr.cifsConfig.Domain,
		Options:      mountOpts,
		RepositoryID: cr.repositoryConfig.ID,
	}

	// Perform mount
	if err := cr.mountManager.MountCIFS(ctx, mountConfig); err != nil {
		return &RepositoryError{
			RepositoryID: cr.repositoryConfig.ID,
			Op:           "mount_cifs",
			Err:          fmt.Errorf("failed to mount CIFS share: %w", err),
		}
	}

	cr.mounted = true
	return nil
}

// resolvePassword resolves the password from the secret reference.
// In production, this would integrate with a secret management system (Vault, K8s secrets, etc.)
// For now, we use the PasswordSecret field directly (assuming it contains the actual password).
// TODO: Integrate with proper secret management system.
func (cr *CIFSRepository) resolvePassword(ctx context.Context) string {
	// SECURITY NOTE: In production, this should:
	// 1. Decrypt the PasswordSecret field
	// 2. Fetch from secret manager (Vault, AWS Secrets Manager, etc.)
	// 3. Never log or expose the password
	// 
	// For now, assuming PasswordSecret contains the password directly
	// This is a simplified implementation for the initial version
	return cr.cifsConfig.PasswordSecret
}

// CreateBackup creates a new backup on CIFS storage.
// Ensures CIFS is mounted before delegating to LocalRepository.
func (cr *CIFSRepository) CreateBackup(ctx context.Context, req BackupRequest) (*Backup, error) {
	if err := cr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return cr.LocalRepository.CreateBackup(ctx, req)
}

// GetBackup retrieves backup metadata from CIFS storage.
// Ensures CIFS is mounted before delegating to LocalRepository.
func (cr *CIFSRepository) GetBackup(ctx context.Context, backupID string) (*Backup, error) {
	if err := cr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return cr.LocalRepository.GetBackup(ctx, backupID)
}

// ListBackups lists all backups for a VM context on CIFS storage.
// Ensures CIFS is mounted before delegating to LocalRepository.
func (cr *CIFSRepository) ListBackups(ctx context.Context, vmContextID string) ([]*Backup, error) {
	if err := cr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return cr.LocalRepository.ListBackups(ctx, vmContextID)
}

// DeleteBackup removes a backup from CIFS storage.
// Ensures CIFS is mounted before delegating to LocalRepository.
func (cr *CIFSRepository) DeleteBackup(ctx context.Context, backupID string) error {
	if err := cr.ensureMounted(ctx); err != nil {
		return err
	}
	return cr.LocalRepository.DeleteBackup(ctx, backupID)
}

// GetBackupChain retrieves the complete backup chain from CIFS storage.
// Ensures CIFS is mounted before delegating to LocalRepository.
func (cr *CIFSRepository) GetBackupChain(ctx context.Context, vmContextID string, diskID int) (*BackupChain, error) {
	if err := cr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return cr.LocalRepository.GetBackupChain(ctx, vmContextID, diskID)
}

// GetStorageInfo returns storage capacity information for CIFS storage.
// Ensures CIFS is mounted before delegating to LocalRepository.
func (cr *CIFSRepository) GetStorageInfo(ctx context.Context) (*StorageInfo, error) {
	if err := cr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return cr.LocalRepository.GetStorageInfo(ctx)
}

// GetExportPath returns the file system path for a backup on CIFS storage.
// Ensures CIFS is mounted before delegating to LocalRepository.
func (cr *CIFSRepository) GetExportPath(ctx context.Context, backupID string) (string, error) {
	if err := cr.ensureMounted(ctx); err != nil {
		return "", err
	}
	return cr.LocalRepository.GetExportPath(ctx, backupID)
}

// Unmount unmounts the CIFS share.
// Should only be called when no backup operations are in progress.
// Returns error if unmount fails or if repository is not mounted.
func (cr *CIFSRepository) Unmount(ctx context.Context) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.mounted {
		return &RepositoryError{
			RepositoryID: cr.repositoryConfig.ID,
			Op:           "unmount",
			Err:          fmt.Errorf("repository is not mounted"),
		}
	}

	if err := cr.mountManager.Unmount(ctx, cr.mountPoint, false); err != nil {
		return &RepositoryError{
			RepositoryID: cr.repositoryConfig.ID,
			Op:           "unmount",
			Err:          fmt.Errorf("failed to unmount CIFS share: %w", err),
		}
	}

	cr.mounted = false
	return nil
}

// IsMounted returns true if the CIFS share is currently mounted.
func (cr *CIFSRepository) IsMounted() bool {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.mounted
}
