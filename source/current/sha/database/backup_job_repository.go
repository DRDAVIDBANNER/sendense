// Package database provides database operations for backup job tracking
package database

import (
	"context"
	"fmt"
	"time"
)

// BackupJob represents a backup job record in the database
// Maps to backup_jobs table (created in 20251004120000_add_backup_tables.up.sql)
type BackupJob struct {
	ID                 string     `gorm:"column:id;primaryKey" json:"id"`
	VMContextID        string     `gorm:"column:vm_context_id;not null;index" json:"vm_context_id"`
	VMName             string     `gorm:"column:vm_name;not null" json:"vm_name"`
	DiskID             int        `gorm:"column:disk_id;not null;default:0" json:"disk_id"` // Added in Task 4 migration
	RepositoryID       string     `gorm:"column:repository_id;not null;index" json:"repository_id"`
	PolicyID           *string    `gorm:"column:policy_id;index" json:"policy_id"` // Pointer for NULL support
	BackupType         string     `gorm:"column:backup_type;not null" json:"backup_type"` // full, incremental, differential
	Status             string     `gorm:"column:status;not null;default:'pending'" json:"status"`
	RepositoryPath     string     `gorm:"column:repository_path;not null" json:"repository_path"`
	ParentBackupID     *string    `gorm:"column:parent_backup_id" json:"parent_backup_id"` // Pointer for NULL support
	ChangeID           string     `gorm:"column:change_id" json:"change_id"`
	BytesTransferred   int64      `gorm:"column:bytes_transferred;default:0" json:"bytes_transferred"`
	TotalBytes         int64      `gorm:"column:total_bytes;default:0" json:"total_bytes"`
	CompressionEnabled bool       `gorm:"column:compression_enabled;default:true" json:"compression_enabled"`
	ErrorMessage       string     `gorm:"column:error_message" json:"error_message"`
	CreatedAt          time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	StartedAt          *time.Time `gorm:"column:started_at" json:"started_at"`
	CompletedAt        *time.Time `gorm:"column:completed_at" json:"completed_at"`
}

// TableName returns the table name for BackupJob
func (BackupJob) TableName() string {
	return "backup_jobs"
}

// BackupJobRepository provides database operations for backup jobs
type BackupJobRepository struct {
	conn Connection
}

// NewBackupJobRepository creates a new backup job repository
func NewBackupJobRepository(conn Connection) *BackupJobRepository {
	return &BackupJobRepository{
		conn: conn,
	}
}

// Create creates a new backup job record
func (r *BackupJobRepository) Create(ctx context.Context, job *BackupJob) error {
	result := r.conn.GetGormDB().WithContext(ctx).Create(job)
	if result.Error != nil {
		return fmt.Errorf("failed to create backup job: %w", result.Error)
	}
	return nil
}

// GetByID retrieves a backup job by ID
func (r *BackupJobRepository) GetByID(ctx context.Context, id string) (*BackupJob, error) {
	var job BackupJob
	result := r.conn.GetGormDB().WithContext(ctx).Where("id = ?", id).First(&job)
	if result.Error != nil {
		return nil, fmt.Errorf("backup job not found: %w", result.Error)
	}
	return &job, nil
}

// Update updates a backup job with the provided fields
func (r *BackupJobRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	result := r.conn.GetGormDB().WithContext(ctx).Model(&BackupJob{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update backup job: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("backup job not found: %s", id)
	}
	return nil
}

// Delete deletes a backup job
func (r *BackupJobRepository) Delete(ctx context.Context, id string) error {
	result := r.conn.GetGormDB().WithContext(ctx).Delete(&BackupJob{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete backup job: %w", result.Error)
	}
	return nil
}

// ListByVMContext lists all backup jobs for a VM context
func (r *BackupJobRepository) ListByVMContext(ctx context.Context, vmContextID string) ([]*BackupJob, error) {
	var jobs []*BackupJob
	result := r.conn.GetGormDB().WithContext(ctx).
		Where("vm_context_id = ?", vmContextID).
		Order("created_at DESC").
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list backup jobs: %w", result.Error)
	}
	return jobs, nil
}

// ListByRepository lists all backup jobs for a repository
func (r *BackupJobRepository) ListByRepository(ctx context.Context, repositoryID string) ([]*BackupJob, error) {
	var jobs []*BackupJob
	result := r.conn.GetGormDB().WithContext(ctx).
		Where("repository_id = ?", repositoryID).
		Order("created_at DESC").
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list backup jobs: %w", result.Error)
	}
	return jobs, nil
}

// ListByStatus lists all backup jobs with a specific status
func (r *BackupJobRepository) ListByStatus(ctx context.Context, status string) ([]*BackupJob, error) {
	var jobs []*BackupJob
	result := r.conn.GetGormDB().WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list backup jobs by status: %w", result.Error)
	}
	return jobs, nil
}

// GetBackupChain retrieves the backup chain (full + incrementals) for a VM
func (r *BackupJobRepository) GetBackupChain(ctx context.Context, vmContextID string, diskID int) ([]*BackupJob, error) {
	var jobs []*BackupJob
	result := r.conn.GetGormDB().WithContext(ctx).
		Where("vm_context_id = ? AND status = 'completed'", vmContextID).
		Order("created_at ASC").
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get backup chain: %w", result.Error)
	}

	// Build chain: find full backup first, then all incrementals based on parent_backup_id
	var chain []*BackupJob

	// Find full backup
	var fullBackup *BackupJob
	for _, job := range jobs {
		if job.BackupType == "full" && (job.ParentBackupID == nil || *job.ParentBackupID == "") {
			fullBackup = job
			break
		}
	}

	if fullBackup == nil {
		return []*BackupJob{}, nil // No chain yet
	}

	chain = append(chain, fullBackup)

	// Build incremental chain
	currentParent := fullBackup.ID
	for {
		foundIncremental := false
		for _, job := range jobs {
			if job.ParentBackupID != nil && *job.ParentBackupID == currentParent {
				chain = append(chain, job)
				currentParent = job.ID
				foundIncremental = true
				break
			}
		}
		if !foundIncremental {
			break
		}
	}

	return chain, nil
}

// GetLatestBackup retrieves the most recent completed backup for a VM
func (r *BackupJobRepository) GetLatestBackup(ctx context.Context, vmContextID string) (*BackupJob, error) {
	var job BackupJob
	result := r.conn.GetGormDB().WithContext(ctx).
		Where("vm_context_id = ? AND status = 'completed'", vmContextID).
		Order("created_at DESC").
		First(&job)
	if result.Error != nil {
		return nil, fmt.Errorf("no completed backups found: %w", result.Error)
	}
	return &job, nil
}

// GetBackupStatistics retrieves backup statistics for a VM context
func (r *BackupJobRepository) GetBackupStatistics(ctx context.Context, vmContextID string) (map[string]interface{}, error) {
	var stats struct {
		TotalBackups     int64
		CompletedBackups int64
		FailedBackups    int64
		TotalBytes       int64
		LatestBackupAt   *time.Time
	}

	// Count total backups
	r.conn.GetGormDB().WithContext(ctx).
		Model(&BackupJob{}).
		Where("vm_context_id = ?", vmContextID).
		Count(&stats.TotalBackups)

	// Count completed backups
	r.conn.GetGormDB().WithContext(ctx).
		Model(&BackupJob{}).
		Where("vm_context_id = ? AND status = 'completed'", vmContextID).
		Count(&stats.CompletedBackups)

	// Count failed backups
	r.conn.GetGormDB().WithContext(ctx).
		Model(&BackupJob{}).
		Where("vm_context_id = ? AND status = 'failed'", vmContextID).
		Count(&stats.FailedBackups)

	// Sum total bytes transferred
	r.conn.GetGormDB().WithContext(ctx).
		Model(&BackupJob{}).
		Where("vm_context_id = ? AND status = 'completed'", vmContextID).
		Select("SUM(bytes_transferred)").
		Scan(&stats.TotalBytes)

	// Get latest backup timestamp
	var latestJob BackupJob
	result := r.conn.GetGormDB().WithContext(ctx).
		Where("vm_context_id = ?", vmContextID).
		Order("created_at DESC").
		First(&latestJob)
	if result.Error == nil {
		stats.LatestBackupAt = &latestJob.CreatedAt
	}

	return map[string]interface{}{
		"total_backups":     stats.TotalBackups,
		"completed_backups": stats.CompletedBackups,
		"failed_backups":    stats.FailedBackups,
		"total_bytes":       stats.TotalBytes,
		"latest_backup_at":  stats.LatestBackupAt,
	}, nil
}
