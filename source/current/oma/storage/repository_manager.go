package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// RepositoryManager manages multiple backup repositories.
// Refactored to use repository pattern (PROJECT_RULES lines 469-470).
// Note: Retains db field for creating repository instances (LocalRepository needs db for ChainManager).
type RepositoryManager struct {
	configRepo   ConfigRepository          // Repository pattern for config operations
	backupRepo   BackupChainRepository     // Backup chain operations
	db           *sql.DB                   // Direct DB access for creating repository instances only
	mountManager *MountManager             // Mount management for NFS/CIFS repositories
	repositories map[string]Repository      // Active repository instances
	configs      map[string]*RepositoryConfig // Repository configurations
	mu           sync.RWMutex               // Protects repositories and configs
}

// NewRepositoryManager creates a new RepositoryManager using repository pattern.
// Note: db parameter retained for creating LocalRepository instances (they need db for ChainManager).
// mountManager is optional - if nil, NFS/CIFS repositories will not be available.
func NewRepositoryManager(configRepo ConfigRepository, backupRepo BackupChainRepository, db *sql.DB, mountManager *MountManager) (*RepositoryManager, error) {
	// Create default MountManager if not provided
	if mountManager == nil {
		mountManager = NewMountManager()
	}

	rm := &RepositoryManager{
		configRepo:   configRepo,
		backupRepo:   backupRepo,
		db:           db,
		mountManager: mountManager,
		repositories: make(map[string]Repository),
		configs:      make(map[string]*RepositoryConfig),
	}

	// Load existing repositories from database
	if err := rm.loadRepositories(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load repositories: %w", err)
	}

	return rm, nil
}

// loadRepositories loads all enabled repositories from database.
func (rm *RepositoryManager) loadRepositories(ctx context.Context) error {
	// Use repository pattern instead of direct SQL
	configs, err := rm.configRepo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to query repositories: %w", err)
	}

	for _, config := range configs {
		// Initialize repository instance
		if err := rm.initializeRepository(ctx, config); err != nil {
			fmt.Printf("Warning: failed to initialize repository %s: %v\n", config.ID, err)
			continue
		}
	}

	return nil
}

// initializeRepository creates a repository instance from config.
func (rm *RepositoryManager) initializeRepository(ctx context.Context, config *RepositoryConfig) error {
	var repo Repository
	var err error

	switch config.Type {
	case RepositoryTypeLocal:
		// LocalRepository needs db for ChainManager - this is legitimate use
		repo, err = NewLocalRepository(config, rm.db)
	case RepositoryTypeNFS:
		// NFSRepository uses MountManager for NFS mounting
		repo, err = NewNFSRepository(config, rm.db, rm.mountManager)
	case RepositoryTypeCIFS, RepositoryTypeSMB:
		// CIFSRepository uses MountManager for CIFS/SMB mounting with authentication
		repo, err = NewCIFSRepository(config, rm.db, rm.mountManager)
	default:
		return fmt.Errorf("unsupported repository type: %s", config.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	// Wrap with immutable repository if needed
	if config.IsImmutable && config.ImmutableConfig != nil {
		// TODO: Wrap with ImmutableRepository in Job Sheet 3
		// For now, just use base repository
	}

	// Store in maps
	rm.mu.Lock()
	rm.repositories[config.ID] = repo
	rm.configs[config.ID] = config
	rm.mu.Unlock()

	return nil
}

// RegisterRepository registers a new backup repository.
func (rm *RepositoryManager) RegisterRepository(ctx context.Context, config *RepositoryConfig) error {
	// Generate ID if not provided
	if config.ID == "" {
		config.ID = fmt.Sprintf("repo-%s-%d", config.Type, time.Now().Unix())
	}

	// Validate configuration
	if err := rm.validateConfig(ctx, config); err != nil {
		return &RepositoryError{
			RepositoryID: config.ID,
			Op:           "validate",
			Err:          err,
		}
	}

	// Test repository connection
	if err := rm.testRepositoryConnection(ctx, config); err != nil {
		return &RepositoryError{
			RepositoryID: config.ID,
			Op:           "test_connection",
			Err:          err,
		}
	}

	// Use repository pattern to create config
	err := rm.configRepo.Create(ctx, config)
	if err != nil {
		return &RepositoryError{
			RepositoryID: config.ID,
			Op:           "insert_database",
			Err:          fmt.Errorf("failed to insert repository: %w", err),
		}
	}

	// Initialize repository instance
	if err := rm.initializeRepository(ctx, config); err != nil {
		// Rollback database insert using repository pattern
		rm.configRepo.Delete(ctx, config.ID)
		return err
	}

	return nil
}

// GetRepository retrieves a repository instance by ID.
func (rm *RepositoryManager) GetRepository(ctx context.Context, repoID string) (Repository, error) {
	rm.mu.RLock()
	repo, exists := rm.repositories[repoID]
	rm.mu.RUnlock()

	if !exists {
		return nil, ErrRepositoryNotFound
	}

	return repo, nil
}

// GetRepositoryConfig retrieves repository configuration by ID.
func (rm *RepositoryManager) GetRepositoryConfig(ctx context.Context, repoID string) (*RepositoryConfig, error) {
	rm.mu.RLock()
	config, exists := rm.configs[repoID]
	rm.mu.RUnlock()

	if !exists {
		return nil, ErrRepositoryNotFound
	}

	return config, nil
}

// ListRepositories returns all configured repositories.
func (rm *RepositoryManager) ListRepositories(ctx context.Context) ([]*RepositoryConfig, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	configs := make([]*RepositoryConfig, 0, len(rm.configs))
	for _, config := range rm.configs {
		configs = append(configs, config)
	}

	return configs, nil
}

// TestRepository tests a repository configuration without saving it.
func (rm *RepositoryManager) TestRepository(ctx context.Context, config *RepositoryConfig) error {
	// Validate configuration
	if err := rm.validateConfig(ctx, config); err != nil {
		return err
	}

	// Test connection
	if err := rm.testRepositoryConnection(ctx, config); err != nil {
		return err
	}

	return nil
}

// DeleteRepository removes a repository configuration.
func (rm *RepositoryManager) DeleteRepository(ctx context.Context, repoID string) error {
	// Check for existing backups using repository pattern
	backupCount, err := rm.configRepo.CountBackupsForRepository(ctx, repoID)
	if err != nil {
		return &RepositoryError{
			RepositoryID: repoID,
			Op:           "check_backups",
			Err:          fmt.Errorf("failed to check for backups: %w", err),
		}
	}

	if backupCount > 0 {
		return &RepositoryError{
			RepositoryID: repoID,
			Op:           "delete",
			Err:          fmt.Errorf("cannot delete repository with %d existing backups", backupCount),
		}
	}

	// Remove from database using repository pattern
	err = rm.configRepo.Delete(ctx, repoID)
	if err != nil {
		return &RepositoryError{
			RepositoryID: repoID,
			Op:           "delete_database",
			Err:          fmt.Errorf("failed to delete from database: %w", err),
		}
	}

	// Remove from active repositories
	rm.mu.Lock()
	delete(rm.repositories, repoID)
	delete(rm.configs, repoID)
	rm.mu.Unlock()

	return nil
}

// UpdateRepository updates a repository configuration.
func (rm *RepositoryManager) UpdateRepository(ctx context.Context, config *RepositoryConfig) error {
	// Validate exists
	if _, err := rm.GetRepositoryConfig(ctx, config.ID); err != nil {
		return err
	}

	// Validate new configuration
	if err := rm.validateConfig(ctx, config); err != nil {
		return err
	}

	// Update database using repository pattern
	err := rm.configRepo.Update(ctx, config)
	if err != nil {
		return &RepositoryError{
			RepositoryID: config.ID,
			Op:           "update_database",
			Err:          fmt.Errorf("failed to update repository: %w", err),
		}
	}

	// Reinitialize repository instance
	rm.mu.Lock()
	delete(rm.repositories, config.ID)
	delete(rm.configs, config.ID)
	rm.mu.Unlock()

	if err := rm.initializeRepository(ctx, config); err != nil {
		return err
	}

	return nil
}

// validateConfig validates a repository configuration.
func (rm *RepositoryManager) validateConfig(ctx context.Context, config *RepositoryConfig) error {
	if config.Name == "" {
		return fmt.Errorf("repository name is required")
	}

	if config.Config == nil {
		return fmt.Errorf("repository config is required")
	}

	switch config.Type {
	case RepositoryTypeLocal:
		localConfig, ok := config.Config.(LocalRepositoryConfig)
		if !ok {
			return fmt.Errorf("invalid local repository config")
		}
		if localConfig.Path == "" {
			return fmt.Errorf("local repository path is required")
		}
	case RepositoryTypeNFS:
		nfsConfig, ok := config.Config.(NFSRepositoryConfig)
		if !ok {
			return fmt.Errorf("invalid NFS repository config")
		}
		if nfsConfig.Server == "" || nfsConfig.ExportPath == "" {
			return fmt.Errorf("NFS server and export path are required")
		}
	case RepositoryTypeCIFS, RepositoryTypeSMB:
		cifsConfig, ok := config.Config.(CIFSRepositoryConfig)
		if !ok {
			return fmt.Errorf("invalid CIFS repository config")
		}
		if cifsConfig.Server == "" || cifsConfig.ShareName == "" {
			return fmt.Errorf("CIFS server and share name are required")
		}
	default:
		return fmt.Errorf("unsupported repository type: %s", config.Type)
	}

	return nil
}

// testRepositoryConnection tests if a repository is accessible.
func (rm *RepositoryManager) testRepositoryConnection(ctx context.Context, config *RepositoryConfig) error {
	switch config.Type {
	case RepositoryTypeLocal:
		localConfig, ok := config.Config.(LocalRepositoryConfig)
		if !ok {
			return fmt.Errorf("invalid local repository config")
		}
		return validatePath(localConfig.Path)
	case RepositoryTypeNFS:
		// TODO: Implement NFS connection test in Job Sheet 2
		return nil
	case RepositoryTypeCIFS, RepositoryTypeSMB:
		// TODO: Implement CIFS connection test in Job Sheet 2
		return nil
	default:
		return fmt.Errorf("unsupported repository type: %s", config.Type)
	}
}

// GetDefaultRepository returns the first enabled repository.
// Useful for systems with only one repository configured.
func (rm *RepositoryManager) GetDefaultRepository(ctx context.Context) (Repository, *RepositoryConfig, error) {
	configs, err := rm.ListRepositories(ctx)
	if err != nil {
		return nil, nil, err
	}

	for _, config := range configs {
		if config.Enabled {
			repo, err := rm.GetRepository(ctx, config.ID)
			if err != nil {
				continue
			}
			return repo, config, nil
		}
	}

	return nil, nil, fmt.Errorf("no enabled repositories found")
}

// RefreshStorageInfo updates storage information for all repositories.
func (rm *RepositoryManager) RefreshStorageInfo(ctx context.Context) error {
	rm.mu.RLock()
	repoIDs := make([]string, 0, len(rm.repositories))
	for id := range rm.repositories {
		repoIDs = append(repoIDs, id)
	}
	rm.mu.RUnlock()

	for _, repoID := range repoIDs {
		repo, err := rm.GetRepository(ctx, repoID)
		if err != nil {
			continue
		}

		info, err := repo.GetStorageInfo(ctx)
		if err != nil {
			fmt.Printf("Warning: failed to get storage info for %s: %v\n", repoID, err)
			continue
		}

		// Update database using repository pattern
		err = rm.configRepo.UpdateStorageStats(ctx, repoID, info.TotalBytes, info.UsedBytes, info.AvailableBytes)
		if err != nil {
			fmt.Printf("Warning: failed to update storage info for %s: %v\n", repoID, err)
		}
	}

	return nil
}