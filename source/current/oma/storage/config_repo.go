package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ConfigRepository defines the interface for repository configuration database operations.
// This follows PROJECT_RULES lines 469-470: "ALL database queries via repository pattern"
type ConfigRepository interface {
	// Repository configuration CRUD
	ListEnabled(ctx context.Context) ([]*RepositoryConfig, error)
	GetByID(ctx context.Context, id string) (*RepositoryConfig, error)
	Create(ctx context.Context, config *RepositoryConfig) error
	Update(ctx context.Context, config *RepositoryConfig) error
	Delete(ctx context.Context, id string) error
	
	// Repository statistics
	CountBackupsForRepository(ctx context.Context, repoID string) (int, error)
	UpdateStorageStats(ctx context.Context, repoID string, total, used, available int64) error
}

// SQLConfigRepository implements ConfigRepository using database/sql.
type SQLConfigRepository struct {
	db *sql.DB
}

// NewConfigRepository creates a new SQL-based configuration repository.
func NewConfigRepository(db *sql.DB) ConfigRepository {
	return &SQLConfigRepository{db: db}
}

// ListEnabled retrieves all enabled backup repositories from the database.
func (r *SQLConfigRepository) ListEnabled(ctx context.Context) ([]*RepositoryConfig, error) {
	query := `
		SELECT id, name, repository_type, enabled, config,
			is_immutable, immutable_config, min_retention_days,
			total_size_bytes, used_size_bytes, available_size_bytes,
			last_check_at, created_at, updated_at
		FROM backup_repositories
		WHERE enabled = TRUE
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled repositories: %w", err)
	}
	defer rows.Close()

	var configs []*RepositoryConfig
	for rows.Next() {
		config, err := r.scanRepositoryConfig(rows)
		if err != nil {
			// Log warning but continue loading other repositories
			fmt.Printf("Warning: failed to scan repository: %v\n", err)
			continue
		}
		configs = append(configs, config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating repository rows: %w", err)
	}

	return configs, nil
}

// GetByID retrieves a specific repository configuration by ID.
func (r *SQLConfigRepository) GetByID(ctx context.Context, id string) (*RepositoryConfig, error) {
	query := `
		SELECT id, name, repository_type, enabled, config,
			is_immutable, immutable_config, min_retention_days,
			total_size_bytes, used_size_bytes, available_size_bytes,
			last_check_at, created_at, updated_at
		FROM backup_repositories
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	config, err := r.scanRepositoryConfigRow(row)
	if err == sql.ErrNoRows {
		return nil, ErrRepositoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query repository: %w", err)
	}

	return config, nil
}

// Create inserts a new repository configuration into the database.
func (r *SQLConfigRepository) Create(ctx context.Context, config *RepositoryConfig) error {
	// Serialize main config to JSON
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Serialize immutable config if present
	var immutableConfigJSON sql.NullString
	if config.IsImmutable && config.ImmutableConfig != nil {
		data, err := json.Marshal(config.ImmutableConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal immutable config: %w", err)
		}
		immutableConfigJSON = sql.NullString{String: string(data), Valid: true}
	}

	now := time.Now()
	query := `
		INSERT INTO backup_repositories (
			id, name, repository_type, enabled, config,
			is_immutable, immutable_config, min_retention_days,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		config.ID, config.Name, config.Type, config.Enabled, configJSON,
		config.IsImmutable, immutableConfigJSON, config.MinRetentionDays,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert repository: %w", err)
	}

	config.CreatedAt = now
	config.UpdatedAt = now
	return nil
}

// Update modifies an existing repository configuration in the database.
func (r *SQLConfigRepository) Update(ctx context.Context, config *RepositoryConfig) error {
	// Serialize main config to JSON
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Serialize immutable config if present
	var immutableConfigJSON sql.NullString
	if config.IsImmutable && config.ImmutableConfig != nil {
		data, err := json.Marshal(config.ImmutableConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal immutable config: %w", err)
		}
		immutableConfigJSON = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		UPDATE backup_repositories
		SET name = ?, enabled = ?, config = ?,
			is_immutable = ?, immutable_config = ?, min_retention_days = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		config.Name, config.Enabled, configJSON,
		config.IsImmutable, immutableConfigJSON, config.MinRetentionDays,
		now, config.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrRepositoryNotFound
	}

	config.UpdatedAt = now
	return nil
}

// Delete removes a repository configuration from the database.
func (r *SQLConfigRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM backup_repositories WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrRepositoryNotFound
	}

	return nil
}

// CountBackupsForRepository counts how many backups are stored in this repository.
func (r *SQLConfigRepository) CountBackupsForRepository(ctx context.Context, repoID string) (int, error) {
	query := `SELECT COUNT(*) FROM backup_jobs WHERE repository_id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count backups: %w", err)
	}

	return count, nil
}

// UpdateStorageStats updates storage statistics for a repository.
func (r *SQLConfigRepository) UpdateStorageStats(ctx context.Context, repoID string, total, used, available int64) error {
	query := `
		UPDATE backup_repositories
		SET total_size_bytes = ?,
			used_size_bytes = ?,
			available_size_bytes = ?,
			last_check_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, total, used, available, now, repoID)
	if err != nil {
		return fmt.Errorf("failed to update storage stats: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrRepositoryNotFound
	}

	return nil
}

// scanRepositoryConfig scans a database row into a RepositoryConfig.
func (r *SQLConfigRepository) scanRepositoryConfig(rows *sql.Rows) (*RepositoryConfig, error) {
	var config RepositoryConfig
	var configJSON []byte
	var immutableConfigJSON sql.NullString
	var lastCheckAt sql.NullTime

	err := rows.Scan(
		&config.ID, &config.Name, &config.Type, &config.Enabled, &configJSON,
		&config.IsImmutable, &immutableConfigJSON, &config.MinRetentionDays,
		&config.TotalBytes, &config.UsedBytes, &config.AvailableBytes,
		&lastCheckAt, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan repository: %w", err)
	}

	// Parse config JSON based on type
	if err := r.parseConfig(&config, configJSON); err != nil {
		return nil, err
	}

	if lastCheckAt.Valid {
		config.LastCheckAt = &lastCheckAt.Time
	}

	// Parse immutable config if present
	if immutableConfigJSON.Valid && config.IsImmutable {
		var immutableConfig ImmutableConfig
		if err := json.Unmarshal([]byte(immutableConfigJSON.String), &immutableConfig); err != nil {
			return nil, fmt.Errorf("failed to parse immutable config: %w", err)
		}
		config.ImmutableConfig = &immutableConfig
	}

	return &config, nil
}

// scanRepositoryConfigRow scans a single row into a RepositoryConfig.
func (r *SQLConfigRepository) scanRepositoryConfigRow(row *sql.Row) (*RepositoryConfig, error) {
	var config RepositoryConfig
	var configJSON []byte
	var immutableConfigJSON sql.NullString
	var lastCheckAt sql.NullTime

	err := row.Scan(
		&config.ID, &config.Name, &config.Type, &config.Enabled, &configJSON,
		&config.IsImmutable, &immutableConfigJSON, &config.MinRetentionDays,
		&config.TotalBytes, &config.UsedBytes, &config.AvailableBytes,
		&lastCheckAt, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse config JSON based on type
	if err := r.parseConfig(&config, configJSON); err != nil {
		return nil, err
	}

	if lastCheckAt.Valid {
		config.LastCheckAt = &lastCheckAt.Time
	}

	// Parse immutable config if present
	if immutableConfigJSON.Valid && config.IsImmutable {
		var immutableConfig ImmutableConfig
		if err := json.Unmarshal([]byte(immutableConfigJSON.String), &immutableConfig); err != nil {
			return nil, fmt.Errorf("failed to parse immutable config: %w", err)
		}
		config.ImmutableConfig = &immutableConfig
	}

	return &config, nil
}

// parseConfig parses the JSON config based on repository type.
func (r *SQLConfigRepository) parseConfig(config *RepositoryConfig, configJSON []byte) error {
	switch config.Type {
	case RepositoryTypeLocal:
		var localConfig LocalRepositoryConfig
		if err := json.Unmarshal(configJSON, &localConfig); err != nil {
			return fmt.Errorf("failed to parse local config: %w", err)
		}
		config.Config = localConfig
	case RepositoryTypeNFS:
		var nfsConfig NFSRepositoryConfig
		if err := json.Unmarshal(configJSON, &nfsConfig); err != nil {
			return fmt.Errorf("failed to parse NFS config: %w", err)
		}
		config.Config = nfsConfig
	case RepositoryTypeCIFS, RepositoryTypeSMB:
		var cifsConfig CIFSRepositoryConfig
		if err := json.Unmarshal(configJSON, &cifsConfig); err != nil {
			return fmt.Errorf("failed to parse CIFS config: %w", err)
		}
		config.Config = cifsConfig
	default:
		return fmt.Errorf("unsupported repository type: %s", config.Type)
	}

	return nil
}

