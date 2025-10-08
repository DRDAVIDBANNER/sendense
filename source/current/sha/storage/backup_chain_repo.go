package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// BackupChainRepository defines the interface for backup chain database operations.
// This follows PROJECT_RULES lines 469-470: "ALL database queries via repository pattern"
type BackupChainRepository interface {
	// Chain CRUD operations
	CreateBackupChain(ctx context.Context, chain *BackupChain) error
	GetBackupChain(ctx context.Context, vmContextID string, diskID int) (*BackupChain, error)
	GetBackupChainByID(ctx context.Context, chainID string) (*BackupChain, error)
	UpdateBackupChain(ctx context.Context, chain *BackupChain) error
	DeleteBackupChain(ctx context.Context, chainID string) error

	// Backup operations for chain management
	GetBackup(ctx context.Context, backupID string) (*Backup, error)
	ListBackupsForChain(ctx context.Context, vmContextID string, diskID int) ([]*Backup, error)
	CountBackupDependencies(ctx context.Context, backupID string) (int, error)
}

// SQLBackupChainRepository implements BackupChainRepository using database/sql.
type SQLBackupChainRepository struct {
	db *sql.DB
}

// NewBackupChainRepository creates a new SQL-based backup chain repository.
func NewBackupChainRepository(db *sql.DB) BackupChainRepository {
	return &SQLBackupChainRepository{db: db}
}

// CreateBackupChain inserts a new backup chain into the database.
func (r *SQLBackupChainRepository) CreateBackupChain(ctx context.Context, chain *BackupChain) error {
	query := `
		INSERT INTO backup_chains (
			id, vm_context_id, disk_id,
			full_backup_id, latest_backup_id,
			total_backups, total_size_bytes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		chain.ID, chain.VMContextID, chain.DiskID,
		chain.FullBackupID, chain.LatestBackupID,
		chain.TotalBackups, chain.TotalSizeBytes,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert backup chain: %w", err)
	}

	chain.CreatedAt = now
	chain.UpdatedAt = now
	return nil
}

// GetBackupChain retrieves a backup chain for a specific VM disk.
func (r *SQLBackupChainRepository) GetBackupChain(ctx context.Context, vmContextID string, diskID int) (*BackupChain, error) {
	query := `
		SELECT id, vm_context_id, disk_id,
			full_backup_id, latest_backup_id,
			total_backups, total_size_bytes,
			created_at, updated_at
		FROM backup_chains
		WHERE vm_context_id = ? AND disk_id = ?
	`

	chain := &BackupChain{}
	err := r.db.QueryRowContext(ctx, query, vmContextID, diskID).Scan(
		&chain.ID, &chain.VMContextID, &chain.DiskID,
		&chain.FullBackupID, &chain.LatestBackupID,
		&chain.TotalBackups, &chain.TotalSizeBytes,
		&chain.CreatedAt, &chain.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrBackupChainNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query backup chain: %w", err)
	}

	return chain, nil
}

// GetBackupChainByID retrieves a backup chain by its ID.
func (r *SQLBackupChainRepository) GetBackupChainByID(ctx context.Context, chainID string) (*BackupChain, error) {
	query := `
		SELECT id, vm_context_id, disk_id,
			full_backup_id, latest_backup_id,
			total_backups, total_size_bytes,
			created_at, updated_at
		FROM backup_chains
		WHERE id = ?
	`

	chain := &BackupChain{}
	err := r.db.QueryRowContext(ctx, query, chainID).Scan(
		&chain.ID, &chain.VMContextID, &chain.DiskID,
		&chain.FullBackupID, &chain.LatestBackupID,
		&chain.TotalBackups, &chain.TotalSizeBytes,
		&chain.CreatedAt, &chain.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrBackupChainNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query backup chain by ID: %w", err)
	}

	return chain, nil
}

// UpdateBackupChain updates an existing backup chain in the database.
func (r *SQLBackupChainRepository) UpdateBackupChain(ctx context.Context, chain *BackupChain) error {
	query := `
		UPDATE backup_chains
		SET full_backup_id = ?,
			latest_backup_id = ?,
			total_backups = ?,
			total_size_bytes = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		chain.FullBackupID, chain.LatestBackupID,
		chain.TotalBackups, chain.TotalSizeBytes,
		now, chain.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update backup chain: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrBackupChainNotFound
	}

	chain.UpdatedAt = now
	return nil
}

// DeleteBackupChain removes a backup chain from the database.
func (r *SQLBackupChainRepository) DeleteBackupChain(ctx context.Context, chainID string) error {
	query := `DELETE FROM backup_chains WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, chainID)
	if err != nil {
		return fmt.Errorf("failed to delete backup chain: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrBackupChainNotFound
	}

	return nil
}

// GetBackup retrieves a single backup by ID.
func (r *SQLBackupChainRepository) GetBackup(ctx context.Context, backupID string) (*Backup, error) {
	query := `
		SELECT id, vm_context_id, vm_name, disk_id,
			backup_type, status, parent_backup_id,
			change_id, repository_path,
			bytes_transferred, total_bytes,
			created_at, completed_at, error_message
		FROM backup_jobs
		WHERE id = ?
	`

	backup := &Backup{}
	var completedAt sql.NullTime
	var errorMessage sql.NullString
	var parentBackupID sql.NullString
	var changeID sql.NullString

	err := r.db.QueryRowContext(ctx, query, backupID).Scan(
		&backup.ID, &backup.VMContextID, &backup.VMName, &backup.DiskID,
		&backup.BackupType, &backup.Status, &parentBackupID,
		&changeID, &backup.FilePath,
		&backup.SizeBytes, &backup.TotalBytes,
		&backup.CreatedAt, &completedAt, &errorMessage,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("backup not found: %s", backupID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query backup: %w", err)
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

	return backup, nil
}

// ListBackupsForChain retrieves all backups for a specific VM disk (chain).
func (r *SQLBackupChainRepository) ListBackupsForChain(ctx context.Context, vmContextID string, diskID int) ([]*Backup, error) {
	query := `
		SELECT id, vm_context_id, vm_name, disk_id,
			backup_type, status, parent_backup_id,
			change_id, repository_path,
			bytes_transferred, total_bytes,
			created_at, completed_at, error_message
		FROM backup_jobs
		WHERE vm_context_id = ? AND disk_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, vmContextID, diskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backups: %w", err)
	}
	defer rows.Close()

	var backups []*Backup
	for rows.Next() {
		backup := &Backup{}
		var completedAt sql.NullTime
		var errorMessage sql.NullString
		var parentBackupID sql.NullString
		var changeID sql.NullString

		err := rows.Scan(
			&backup.ID, &backup.VMContextID, &backup.VMName, &backup.DiskID,
			&backup.BackupType, &backup.Status, &parentBackupID,
			&changeID, &backup.FilePath,
			&backup.SizeBytes, &backup.TotalBytes,
			&backup.CreatedAt, &completedAt, &errorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan backup: %w", err)
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

		backups = append(backups, backup)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating backup rows: %w", err)
	}

	return backups, nil
}

// CountBackupDependencies counts how many backups depend on the given backup as a parent.
func (r *SQLBackupChainRepository) CountBackupDependencies(ctx context.Context, backupID string) (int, error) {
	query := `SELECT COUNT(*) FROM backup_jobs WHERE parent_backup_id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, backupID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count backup dependencies: %w", err)
	}

	return count, nil
}

