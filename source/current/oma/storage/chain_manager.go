package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ChainManager handles backup chain tracking and validation.
// Refactored to use repository pattern (PROJECT_RULES lines 469-470).
// Note: Retains db field for complex transaction operations (SELECT FOR UPDATE patterns).
type ChainManager struct {
	repo BackupChainRepository // Repository pattern for simple queries
	db   *sql.DB               // Direct DB access for transactions only
}

// NewChainManager creates a new ChainManager using repository pattern.
// Note: db parameter retained for transaction support (AddBackupToChain, RemoveBackupFromChain).
func NewChainManager(repo BackupChainRepository, db *sql.DB) *ChainManager {
	return &ChainManager{
		repo: repo,
		db:   db,
	}
}

// GetOrCreateChain retrieves an existing chain or creates a new one for a VM disk.
func (cm *ChainManager) GetOrCreateChain(ctx context.Context, vmContextID string, diskID int) (*BackupChain, error) {
	// Try to get existing chain
	chain, err := cm.GetChain(ctx, vmContextID, diskID)
	if err == nil {
		return chain, nil
	}
	if err != ErrBackupChainNotFound {
		return nil, err
	}

	// Create new chain using repository
	chainID := GenerateChainID(vmContextID, diskID)
	now := time.Now()

	newChain := &BackupChain{
		ID:             chainID,
		VMContextID:    vmContextID,
		DiskID:         diskID,
		FullBackupID:   "",
		LatestBackupID: "",
		Backups:        []*Backup{},
		TotalBackups:   0,
		TotalSizeBytes: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err = cm.repo.CreateBackupChain(ctx, newChain)
	if err != nil {
		return nil, &ChainError{
			ChainID: chainID,
			Op:      "create",
			Err:     fmt.Errorf("failed to create chain: %w", err),
		}
	}

	return newChain, nil
}

// GetChain retrieves the complete backup chain for a VM disk.
func (cm *ChainManager) GetChain(ctx context.Context, vmContextID string, diskID int) (*BackupChain, error) {
	// Get chain record using repository
	chain, err := cm.repo.GetBackupChain(ctx, vmContextID, diskID)
	if err != nil {
		return nil, err
	}

	// Get all backups for this chain using repository
	backups, err := cm.repo.ListBackupsForChain(ctx, vmContextID, diskID)
	if err != nil {
		return nil, &ChainError{
			ChainID: chain.ID,
			Op:      "get_backups",
			Err:     fmt.Errorf("failed to query backups: %w", err),
		}
	}

	chain.Backups = backups
	return chain, nil
}

// AddBackupToChain adds a backup to the chain and updates metadata.
func (cm *ChainManager) AddBackupToChain(ctx context.Context, chainID string, backup *Backup) error {
	// Start transaction (using db for complex transaction pattern)
	tx, err := cm.db.BeginTx(ctx, nil)
	if err != nil {
		return &ChainError{
			ChainID: chainID,
			Op:      "begin_transaction",
			Err:     fmt.Errorf("failed to begin transaction: %w", err),
		}
	}
	defer tx.Rollback()

	// Get current chain state
	var fullBackupID, latestBackupID string
	var totalBackups int
	var totalSizeBytes int64

	query := `
		SELECT full_backup_id, latest_backup_id, total_backups, total_size_bytes
		FROM backup_chains
		WHERE id = ?
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, query, chainID).Scan(
		&fullBackupID, &latestBackupID, &totalBackups, &totalSizeBytes,
	)
	if err != nil {
		return &ChainError{
			ChainID: chainID,
			Op:      "get_chain_state",
			Err:     fmt.Errorf("failed to get chain state: %w", err),
		}
	}

	// Update full_backup_id if this is the first full backup
	if backup.BackupType == BackupTypeFull && fullBackupID == "" {
		fullBackupID = backup.ID
	}

	// Update latest backup
	latestBackupID = backup.ID
	totalBackups++
	totalSizeBytes += backup.SizeBytes

	// Update chain
	updateQuery := `
		UPDATE backup_chains
		SET full_backup_id = ?,
			latest_backup_id = ?,
			total_backups = ?,
			total_size_bytes = ?,
			updated_at = ?
		WHERE id = ?
	`

	_, err = tx.ExecContext(ctx, updateQuery,
		fullBackupID, latestBackupID,
		totalBackups, totalSizeBytes,
		time.Now(), chainID,
	)
	if err != nil {
		return &ChainError{
			ChainID: chainID,
			Op:      "update_chain",
			Err:     fmt.Errorf("failed to update chain: %w", err),
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return &ChainError{
			ChainID: chainID,
			Op:      "commit",
			Err:     fmt.Errorf("failed to commit transaction: %w", err),
		}
	}

	return nil
}

// ValidateChain checks if a backup chain is valid and intact.
func (cm *ChainManager) ValidateChain(ctx context.Context, chainID string) error {
	// Get chain with all backups
	chain, err := cm.GetChainByID(ctx, chainID)
	if err != nil {
		return err
	}

	if len(chain.Backups) == 0 {
		return nil // Empty chain is valid
	}

	// First backup must be full
	if chain.Backups[0].BackupType != BackupTypeFull {
		return &ChainError{
			ChainID: chainID,
			Op:      "validate",
			Err:     fmt.Errorf("chain must start with full backup"),
		}
	}

	// Verify parent relationships
	backupMap := make(map[string]*Backup)
	for _, backup := range chain.Backups {
		backupMap[backup.ID] = backup
	}

	for i, backup := range chain.Backups {
		if i == 0 {
			// First backup (full) should have no parent
			if backup.ParentBackupID != "" {
				return &ChainError{
					ChainID: chainID,
					Op:      "validate",
					Err:     fmt.Errorf("full backup %s should not have parent", backup.ID),
				}
			}
			continue
		}

		// Incremental backups must have parent
		if backup.BackupType == BackupTypeIncremental {
			if backup.ParentBackupID == "" {
				return &ChainError{
					ChainID: chainID,
					Op:      "validate",
					Err:     fmt.Errorf("incremental backup %s missing parent", backup.ID),
				}
			}

			// Parent must exist in chain
			if _, exists := backupMap[backup.ParentBackupID]; !exists {
				return &ChainError{
					ChainID: chainID,
					Op:      "validate",
					Err:     fmt.Errorf("parent backup %s not found for %s", backup.ParentBackupID, backup.ID),
				}
			}
		}
	}

	return nil
}

// GetChainByID retrieves a chain by its ID.
func (cm *ChainManager) GetChainByID(ctx context.Context, chainID string) (*BackupChain, error) {
	// Use repository to get chain by ID
	chain, err := cm.repo.GetBackupChainByID(ctx, chainID)
	if err != nil {
		return nil, err
	}

	// Get all backups for this chain
	backups, err := cm.repo.ListBackupsForChain(ctx, chain.VMContextID, chain.DiskID)
	if err != nil {
		return nil, &ChainError{
			ChainID: chainID,
			Op:      "get_backups",
			Err:     fmt.Errorf("failed to query backups: %w", err),
		}
	}

	chain.Backups = backups
	return chain, nil
}

// CalculateChainSize calculates the total actual size of all files in a chain.
func (cm *ChainManager) CalculateChainSize(ctx context.Context, chainID string) (int64, error) {
	chain, err := cm.GetChainByID(ctx, chainID)
	if err != nil {
		return 0, err
	}

	var totalSize int64
	for _, backup := range chain.Backups {
		totalSize += backup.SizeBytes
	}

	return totalSize, nil
}

// CanDeleteBackup checks if a backup can be safely deleted (no dependents).
func (cm *ChainManager) CanDeleteBackup(ctx context.Context, backupID string) (bool, error) {
	// Use repository to count dependencies
	count, err := cm.repo.CountBackupDependencies(ctx, backupID)
	if err != nil {
		return false, &BackupError{
			BackupID: backupID,
			Op:       "check_dependents",
			Err:      fmt.Errorf("failed to check dependents: %w", err),
		}
	}

	return count == 0, nil
}

// RemoveBackupFromChain removes a backup from chain tracking and updates metadata.
func (cm *ChainManager) RemoveBackupFromChain(ctx context.Context, chainID, backupID string) error {
	// Start transaction (using db for complex transaction pattern)
	tx, err := cm.db.BeginTx(ctx, nil)
	if err != nil {
		return &ChainError{
			ChainID: chainID,
			Op:      "begin_transaction",
			Err:     fmt.Errorf("failed to begin transaction: %w", err),
		}
	}
	defer tx.Rollback()

	// Get backup size
	var sizeBytes int64
	err = tx.QueryRowContext(ctx, "SELECT bytes_transferred FROM backup_jobs WHERE id = ?", backupID).Scan(&sizeBytes)
	if err != nil {
		return &BackupError{
			BackupID: backupID,
			Op:       "get_size",
			Err:      fmt.Errorf("failed to get backup size: %w", err),
		}
	}

	// Update chain totals
	updateQuery := `
		UPDATE backup_chains
		SET total_backups = total_backups - 1,
			total_size_bytes = total_size_bytes - ?,
			updated_at = ?
		WHERE id = ?
	`

	_, err = tx.ExecContext(ctx, updateQuery, sizeBytes, time.Now(), chainID)
	if err != nil {
		return &ChainError{
			ChainID: chainID,
			Op:      "update_chain",
			Err:     fmt.Errorf("failed to update chain: %w", err),
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return &ChainError{
			ChainID: chainID,
			Op:      "commit",
			Err:     fmt.Errorf("failed to commit transaction: %w", err),
		}
	}

	return nil
}