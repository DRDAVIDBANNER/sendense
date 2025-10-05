package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// NFSRepository implements Repository interface for NFS-backed storage.
// Design: Embeds LocalRepository for all backup operations, adds NFS mount management.
type NFSRepository struct {
	*LocalRepository              // Embedded for backup operations
	mountManager     *MountManager
	nfsConfig        *NFSRepositoryConfig
	repositoryConfig *RepositoryConfig
	mountPoint       string
	mounted          bool
	mu               sync.RWMutex // Protects mount state
}

// NewNFSRepository creates a new NFS-backed repository.
// The repository will mount the NFS share on first use and manage the mount lifecycle.
func NewNFSRepository(config *RepositoryConfig, db *sql.DB, mountManager *MountManager) (*NFSRepository, error) {
	// Parse NFS config
	nfsConfig, ok := config.Config.(NFSRepositoryConfig)
	if !ok {
		// Try type assertion from map[string]interface{} (JSON unmarshaling case)
		if err := parseConfig(config.Config, &nfsConfig); err != nil {
			return nil, &RepositoryError{
				RepositoryID: config.ID,
				Op:           "parse_config",
				Err:          fmt.Errorf("invalid NFS repository config: %w", err),
			}
		}
	}

	// Validate NFS config
	if nfsConfig.Server == "" {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "validate_config",
			Err:          fmt.Errorf("NFS server cannot be empty"),
		}
	}
	if nfsConfig.ExportPath == "" {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "validate_config",
			Err:          fmt.Errorf("NFS export path cannot be empty"),
		}
	}
	if nfsConfig.MountPoint == "" {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "validate_config",
			Err:          fmt.Errorf("NFS mount point cannot be empty"),
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
			Path: nfsConfig.MountPoint,
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

	nfsRepo := &NFSRepository{
		LocalRepository:  localRepo,
		mountManager:     mountManager,
		nfsConfig:        &nfsConfig,
		repositoryConfig: config,
		mountPoint:       nfsConfig.MountPoint,
		mounted:          false,
	}

	return nfsRepo, nil
}

// ensureMounted checks if NFS is mounted and mounts it if not.
// This is called before any operation that requires storage access.
func (nr *NFSRepository) ensureMounted(ctx context.Context) error {
	// Fast path: check if already mounted without lock
	nr.mu.RLock()
	if nr.mounted {
		nr.mu.RUnlock()
		return nil
	}
	nr.mu.RUnlock()

	// Slow path: acquire write lock and mount
	nr.mu.Lock()
	defer nr.mu.Unlock()

	// Double-check after acquiring lock
	if nr.mounted {
		return nil
	}

	// Check if already mounted in system (could have been mounted externally)
	isMounted := nr.mountManager.IsMounted(nr.mountPoint)

	if isMounted {
		// Already mounted (externally), just mark as mounted
		nr.mounted = true
		return nil
	}

	// Parse mount options
	mountOpts := []string{"rw", "hard"} // Default options
	if nr.nfsConfig.MountOptions != "" {
		// Parse comma-separated options
		mountOpts = parseMountOptions(nr.nfsConfig.MountOptions)
	}
	if nr.nfsConfig.NFSVersion != "" {
		mountOpts = append(mountOpts, fmt.Sprintf("vers=%s", nr.nfsConfig.NFSVersion))
	}

	// Prepare mount configuration
	mountConfig := NFSMountConfig{
		Server:       nr.nfsConfig.Server,
		ExportPath:   nr.nfsConfig.ExportPath,
		MountPoint:   nr.mountPoint,
		Options:      mountOpts,
		RepositoryID: nr.repositoryConfig.ID,
	}

	// Perform mount
	if err := nr.mountManager.MountNFS(ctx, mountConfig); err != nil {
		return &RepositoryError{
			RepositoryID: nr.repositoryConfig.ID,
			Op:           "mount_nfs",
			Err:          fmt.Errorf("failed to mount NFS share: %w", err),
		}
	}

	nr.mounted = true
	return nil
}

// CreateBackup creates a new backup on NFS storage.
// Ensures NFS is mounted before delegating to LocalRepository.
func (nr *NFSRepository) CreateBackup(ctx context.Context, req BackupRequest) (*Backup, error) {
	if err := nr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return nr.LocalRepository.CreateBackup(ctx, req)
}

// GetBackup retrieves backup metadata from NFS storage.
// Ensures NFS is mounted before delegating to LocalRepository.
func (nr *NFSRepository) GetBackup(ctx context.Context, backupID string) (*Backup, error) {
	if err := nr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return nr.LocalRepository.GetBackup(ctx, backupID)
}

// ListBackups lists all backups for a VM context on NFS storage.
// Ensures NFS is mounted before delegating to LocalRepository.
func (nr *NFSRepository) ListBackups(ctx context.Context, vmContextID string) ([]*Backup, error) {
	if err := nr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return nr.LocalRepository.ListBackups(ctx, vmContextID)
}

// DeleteBackup removes a backup from NFS storage.
// Ensures NFS is mounted before delegating to LocalRepository.
func (nr *NFSRepository) DeleteBackup(ctx context.Context, backupID string) error {
	if err := nr.ensureMounted(ctx); err != nil {
		return err
	}
	return nr.LocalRepository.DeleteBackup(ctx, backupID)
}

// GetBackupChain retrieves the complete backup chain from NFS storage.
// Ensures NFS is mounted before delegating to LocalRepository.
func (nr *NFSRepository) GetBackupChain(ctx context.Context, vmContextID string, diskID int) (*BackupChain, error) {
	if err := nr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return nr.LocalRepository.GetBackupChain(ctx, vmContextID, diskID)
}

// GetStorageInfo returns storage capacity information for NFS storage.
// Ensures NFS is mounted before delegating to LocalRepository.
func (nr *NFSRepository) GetStorageInfo(ctx context.Context) (*StorageInfo, error) {
	if err := nr.ensureMounted(ctx); err != nil {
		return nil, err
	}
	return nr.LocalRepository.GetStorageInfo(ctx)
}

// GetExportPath returns the file system path for a backup on NFS storage.
// Ensures NFS is mounted before delegating to LocalRepository.
func (nr *NFSRepository) GetExportPath(ctx context.Context, backupID string) (string, error) {
	if err := nr.ensureMounted(ctx); err != nil {
		return "", err
	}
	return nr.LocalRepository.GetExportPath(ctx, backupID)
}

// Unmount unmounts the NFS share.
// Should only be called when no backup operations are in progress.
// Returns error if unmount fails or if repository is not mounted.
func (nr *NFSRepository) Unmount(ctx context.Context) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	if !nr.mounted {
		return &RepositoryError{
			RepositoryID: nr.repositoryConfig.ID,
			Op:           "unmount",
			Err:          fmt.Errorf("repository is not mounted"),
		}
	}

	if err := nr.mountManager.Unmount(ctx, nr.mountPoint, false); err != nil {
		return &RepositoryError{
			RepositoryID: nr.repositoryConfig.ID,
			Op:           "unmount",
			Err:          fmt.Errorf("failed to unmount NFS share: %w", err),
		}
	}

	nr.mounted = false
	return nil
}

// IsMounted returns true if the NFS share is currently mounted.
func (nr *NFSRepository) IsMounted() bool {
	nr.mu.RLock()
	defer nr.mu.RUnlock()
	return nr.mounted
}

// parseMountOptions parses comma-separated mount options string.
func parseMountOptions(opts string) []string {
	if opts == "" {
		return []string{}
	}
	
	result := []string{}
	for _, opt := range splitAndTrim(opts, ',') {
		if opt != "" {
			result = append(result, opt)
		}
	}
	return result
}

// splitAndTrim splits a string by delimiter and trims whitespace.
func splitAndTrim(s string, delim rune) []string {
	result := []string{}
	current := ""
	
	for _, char := range s {
		if char == delim {
			trimmed := trimSpace(current)
			if trimmed != "" {
				result = append(result, trimmed)
			}
			current = ""
		} else {
			current += string(char)
		}
	}
	
	// Add last segment
	trimmed := trimSpace(current)
	if trimmed != "" {
		result = append(result, trimmed)
	}
	
	return result
}

// trimSpace removes leading and trailing whitespace.
func trimSpace(s string) string {
	start := 0
	end := len(s)
	
	// Trim leading whitespace
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	
	// Trim trailing whitespace
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	
	return s[start:end]
}
