package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// LocalRepository implements the Repository interface for local filesystem storage.
type LocalRepository struct {
	config      *RepositoryConfig
	basePath    string
	db          *sql.DB
	qcowManager *QCOW2Manager
	chainMgr    *ChainManager
}

// NewLocalRepository creates a new LocalRepository instance.
func NewLocalRepository(config *RepositoryConfig, db *sql.DB) (*LocalRepository, error) {
	// Parse local config
	var localConfig LocalRepositoryConfig
	if err := parseConfig(config.Config, &localConfig); err != nil {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "parse_config",
			Err:          fmt.Errorf("invalid local repository config: %w", err),
		}
	}

	// Validate path exists and is writable
	if err := validatePath(localConfig.Path); err != nil {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "validate_path",
			Err:          err,
		}
	}

	// Initialize QCOW2Manager
	qcowManager, err := NewQCOW2Manager()
	if err != nil {
		return nil, &RepositoryError{
			RepositoryID: config.ID,
			Op:           "init_qcow2_manager",
			Err:          err,
		}
	}

	return &LocalRepository{
		config:      config,
		basePath:    localConfig.Path,
		db:          db,
		qcowManager: qcowManager,
		chainMgr:    NewChainManager(NewBackupChainRepository(db), db),
	}, nil
}

// CreateBackup creates a new backup in the local repository.
func (lr *LocalRepository) CreateBackup(ctx context.Context, req BackupRequest) (*Backup, error) {
	// Generate backup ID
	backupID := GenerateBackupID(req.VMName, req.DiskID)

	// Get or create chain (TODO: use returned chain for validation)
	_, err := lr.chainMgr.GetOrCreateChain(ctx, req.VMContextID, req.DiskID)
	if err != nil {
		return nil, err
	}

	// Determine backup file path
	backupPath := GetBackupFilePath(lr.basePath, req.VMContextID, req.DiskID, backupID)

	// Create QCOW2 file
	if req.BackupType == BackupTypeFull {
		// Full backup
		if err := lr.qcowManager.CreateFull(ctx, backupPath, req.TotalBytes); err != nil {
			return nil, &BackupError{
				BackupID: backupID,
				Op:       "create_full",
				Err:      err,
			}
		}
	} else if req.BackupType == BackupTypeIncremental {
		// Incremental backup requires parent
		if req.ParentBackupID == "" {
			return nil, &BackupError{
				BackupID: backupID,
				Op:       "create_incremental",
				Err:      ErrParentBackupRequired,
			}
		}

		// Get parent backup path
		parentBackup, err := lr.GetBackup(ctx, req.ParentBackupID)
		if err != nil {
			return nil, &BackupError{
				BackupID: backupID,
				Op:       "get_parent",
				Err:      fmt.Errorf("parent backup not found: %w", err),
			}
		}

		// Create incremental with backing file
		if err := lr.qcowManager.CreateIncremental(ctx, backupPath, parentBackup.FilePath); err != nil {
			return nil, &BackupError{
				BackupID: backupID,
				Op:       "create_incremental",
				Err:      err,
			}
		}
	} else {
		return nil, &BackupError{
			BackupID: backupID,
			Op:       "create_backup",
			Err:      ErrInvalidBackupType,
		}
	}

	// Create backup record
	now := time.Now()
	backup := &Backup{
		ID:             backupID,
		VMContextID:    req.VMContextID,
		VMName:         req.VMName,
		DiskID:         req.DiskID,
		BackupType:     req.BackupType,
		Status:         BackupStatusPending,
		ParentBackupID: req.ParentBackupID,
		ChangeID:       req.ChangeID,
		FilePath:       backupPath,
		SizeBytes:      0, // Will be updated when backup completes
		TotalBytes:     req.TotalBytes,
		CreatedAt:      now,
	}

	// Insert into database
	query := `
		INSERT INTO backup_jobs (
			id, vm_context_id, vm_name, repository_id,
			backup_type, status, repository_path,
			parent_backup_id, change_id,
			bytes_transferred, total_bytes,
			compression_enabled, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = lr.db.ExecContext(ctx, query,
		backup.ID, backup.VMContextID, backup.VMName, lr.config.ID,
		backup.BackupType, backup.Status, backup.FilePath,
		nullString(backup.ParentBackupID), nullString(backup.ChangeID),
		backup.SizeBytes, backup.TotalBytes,
		true, backup.CreatedAt,
	)
	if err != nil {
		// Clean up QCOW2 file
		os.Remove(backupPath)
		return nil, &BackupError{
			BackupID: backupID,
			Op:       "insert_database",
			Err:      fmt.Errorf("failed to insert backup: %w", err),
		}
	}

	// Get QCOW2 info and save metadata
	qcow2Info, err := lr.qcowManager.GetInfo(ctx, backupPath)
	if err == nil {
		SaveBackupMetadata(backup, req.Metadata, qcow2Info)
	}

	return backup, nil
}

// GetBackup retrieves backup metadata by ID.
func (lr *LocalRepository) GetBackup(ctx context.Context, backupID string) (*Backup, error) {
	var backup Backup
	var completedAt sql.NullTime
	var errorMessage, parentBackupID, changeID sql.NullString

	query := `
		SELECT id, vm_context_id, vm_name, disk_id,
			backup_type, status, parent_backup_id,
			change_id, repository_path, bytes_transferred,
			total_bytes, created_at, completed_at, error_message
		FROM backup_jobs
		WHERE id = ? AND repository_id = ?
	`

	err := lr.db.QueryRowContext(ctx, query, backupID, lr.config.ID).Scan(
		&backup.ID, &backup.VMContextID, &backup.VMName, &backup.DiskID,
		&backup.BackupType, &backup.Status, &parentBackupID,
		&changeID, &backup.FilePath, &backup.SizeBytes,
		&backup.TotalBytes, &backup.CreatedAt, &completedAt, &errorMessage,
	)
	if err == sql.ErrNoRows {
		return nil, ErrBackupNotFound
	}
	if err != nil {
		return nil, &BackupError{
			BackupID: backupID,
			Op:       "get_backup",
			Err:      fmt.Errorf("failed to query backup: %w", err),
		}
	}

	if completedAt.Valid {
		backup.CompletedAt = &completedAt.Time
	}
	if errorMessage.Valid {
		backup.ErrorMessage = errorMessage.String
	}
	if parentBackupID.Valid {
		backup.ParentBackupID = parentBackupID.String
	}
	if changeID.Valid {
		backup.ChangeID = changeID.String
	}

	return &backup, nil
}

// ListBackups lists all backups for a VM context.
func (lr *LocalRepository) ListBackups(ctx context.Context, vmContextID string) ([]*Backup, error) {
	query := `
		SELECT id, vm_context_id, vm_name, disk_id,
			backup_type, status, parent_backup_id,
			change_id, repository_path, bytes_transferred,
			total_bytes, created_at, completed_at, error_message
		FROM backup_jobs
		WHERE vm_context_id = ? AND repository_id = ?
		ORDER BY created_at DESC
	`

	rows, err := lr.db.QueryContext(ctx, query, vmContextID, lr.config.ID)
	if err != nil {
		return nil, &RepositoryError{
			RepositoryID: lr.config.ID,
			Op:           "list_backups",
			Err:          fmt.Errorf("failed to query backups: %w", err),
		}
	}
	defer rows.Close()

	backups := []*Backup{}
	for rows.Next() {
		var backup Backup
		var completedAt sql.NullTime
		var errorMessage, parentBackupID, changeID sql.NullString

		err := rows.Scan(
			&backup.ID, &backup.VMContextID, &backup.VMName, &backup.DiskID,
			&backup.BackupType, &backup.Status, &parentBackupID,
			&changeID, &backup.FilePath, &backup.SizeBytes,
			&backup.TotalBytes, &backup.CreatedAt, &completedAt, &errorMessage,
		)
		if err != nil {
			return nil, &RepositoryError{
				RepositoryID: lr.config.ID,
				Op:           "scan_backup",
				Err:          fmt.Errorf("failed to scan backup: %w", err),
			}
		}

		if completedAt.Valid {
			backup.CompletedAt = &completedAt.Time
		}
		if errorMessage.Valid {
			backup.ErrorMessage = errorMessage.String
		}
		if parentBackupID.Valid {
			backup.ParentBackupID = parentBackupID.String
		}
		if changeID.Valid {
			backup.ChangeID = changeID.String
		}

		backups = append(backups, &backup)
	}

	if err := rows.Err(); err != nil {
		return nil, &RepositoryError{
			RepositoryID: lr.config.ID,
			Op:           "iterate_backups",
			Err:          fmt.Errorf("failed to iterate backups: %w", err),
		}
	}

	return backups, nil
}

// DeleteBackup removes a backup from the repository.
func (lr *LocalRepository) DeleteBackup(ctx context.Context, backupID string) error {
	// Get backup details
	backup, err := lr.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}

	// Check if backup has dependents
	canDelete, err := lr.chainMgr.CanDeleteBackup(ctx, backupID)
	if err != nil {
		return err
	}
	if !canDelete {
		return ErrBackupHasDependents
	}

	// Start transaction
	tx, err := lr.db.BeginTx(ctx, nil)
	if err != nil {
		return &BackupError{
			BackupID: backupID,
			Op:       "begin_transaction",
			Err:      fmt.Errorf("failed to begin transaction: %w", err),
		}
	}
	defer tx.Rollback()

	// Delete from database
	_, err = tx.ExecContext(ctx, "DELETE FROM backup_jobs WHERE id = ?", backupID)
	if err != nil {
		return &BackupError{
			BackupID: backupID,
			Op:       "delete_database",
			Err:      fmt.Errorf("failed to delete from database: %w", err),
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return &BackupError{
			BackupID: backupID,
			Op:       "commit",
			Err:      fmt.Errorf("failed to commit transaction: %w", err),
		}
	}

	// Delete files
	if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
		// File deletion failed but database is already updated
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to delete backup file %s: %v\n", backup.FilePath, err)
	}

	// Delete metadata sidecar
	metadataPath := backup.FilePath + ".json"
	os.Remove(metadataPath) // Ignore errors

	return nil
}

// GetBackupChain retrieves the complete backup chain for a VM disk.
func (lr *LocalRepository) GetBackupChain(ctx context.Context, vmContextID string, diskID int) (*BackupChain, error) {
	return lr.chainMgr.GetChain(ctx, vmContextID, diskID)
}

// GetStorageInfo returns current storage capacity and usage.
func (lr *LocalRepository) GetStorageInfo(ctx context.Context) (*StorageInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(lr.basePath, &stat); err != nil {
		return nil, &RepositoryError{
			RepositoryID: lr.config.ID,
			Op:           "stat_filesystem",
			Err:          fmt.Errorf("failed to stat filesystem: %w", err),
		}
	}

	totalBytes := int64(stat.Blocks) * int64(stat.Bsize)
	availableBytes := int64(stat.Bavail) * int64(stat.Bsize)
	usedBytes := totalBytes - availableBytes

	// Count backups in this repository
	var backupCount int
	err := lr.db.QueryRowContext(ctx, 
		"SELECT COUNT(*) FROM backup_jobs WHERE repository_id = ?", 
		lr.config.ID).Scan(&backupCount)
	if err != nil {
		return nil, &RepositoryError{
			RepositoryID: lr.config.ID,
			Op:           "count_backups",
			Err:          fmt.Errorf("failed to count backups: %w", err),
		}
	}

	return &StorageInfo{
		RepositoryID:   lr.config.ID,
		TotalBytes:     totalBytes,
		UsedBytes:      usedBytes,
		AvailableBytes: availableBytes,
		UsedPercent:    float64(usedBytes) / float64(totalBytes) * 100,
		BackupCount:    backupCount,
		LastCheckAt:    time.Now(),
	}, nil
}

// GetExportPath returns the file system path for a backup (for NBD export).
func (lr *LocalRepository) GetExportPath(ctx context.Context, backupID string) (string, error) {
	backup, err := lr.GetBackup(ctx, backupID)
	if err != nil {
		return "", err
	}

	// Verify file exists
	if _, err := os.Stat(backup.FilePath); err != nil {
		return "", &BackupError{
			BackupID: backupID,
			Op:       "verify_file",
			Err:      fmt.Errorf("backup file not found: %w", err),
		}
	}

	return backup.FilePath, nil
}

// validatePath checks if a path exists and is writable.
func validatePath(path string) error {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create it
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("path does not exist and cannot be created: %w", err)
			}
			return nil
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}

	// Check if writable by creating a temp file
	testFile := filepath.Join(path, ".sendense_test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("path is not writable: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

// parseConfig parses interface{} config into a specific config type.
func parseConfig(src interface{}, dst interface{}) error {
	// This is a simplified version - in production you'd use mapstructure or similar
	// For now, assume src is already the correct type or can be type-asserted
	switch v := src.(type) {
	case LocalRepositoryConfig:
		if ptr, ok := dst.(*LocalRepositoryConfig); ok {
			*ptr = v
			return nil
		}
	case map[string]interface{}:
		// Handle JSON unmarshaling case
		if ptr, ok := dst.(*LocalRepositoryConfig); ok {
			if path, ok := v["path"].(string); ok {
				ptr.Path = path
			}
			if autoMount, ok := v["auto_mount"].(bool); ok {
				ptr.AutoMount = autoMount
			}
			if mountOpts, ok := v["mount_options"].(string); ok {
				ptr.MountOptions = mountOpts
			}
			return nil
		}
	}
	return fmt.Errorf("invalid config type")
}

// nullString returns a sql.NullString for the given string.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
