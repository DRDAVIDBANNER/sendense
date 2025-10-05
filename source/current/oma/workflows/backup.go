// Package workflows provides orchestration for VMware backup operations
// Following project rules: modular design, clean interfaces, comprehensive error handling
package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/nbd"
	"github.com/vexxhost/migratekit-oma/storage"
)

// BackupEngine orchestrates VMware backup operations to repository storage
// Integrates with NBD file export (Task 2) and storage infrastructure (Task 1)
type BackupEngine struct {
	// Repository dependencies
	db              database.Connection
	backupJobRepo   *database.BackupJobRepository
	backupChainRepo storage.BackupChainRepository
	vmContextRepo   *database.VMReplicationContextRepository

	// Storage infrastructure (Task 1)
	repositoryManager *storage.RepositoryManager

	// VMA client for triggering replications
	vmaAPIEndpoint string
	vmaClient      *http.Client
}

// NewBackupEngine creates a new backup workflow orchestration engine
func NewBackupEngine(
	db database.Connection,
	repositoryManager *storage.RepositoryManager,
	vmaAPIEndpoint string,
) *BackupEngine {
	// Get SQL DB for backup chain repo
	sqlDB, err := db.GetGormDB().DB()
	if err != nil {
		log.WithError(err).Fatal("Failed to get SQL DB for backup engine")
	}

	return &BackupEngine{
		db:                db,
		backupJobRepo:     database.NewBackupJobRepository(db),
		backupChainRepo:   storage.NewBackupChainRepository(sqlDB),
		vmContextRepo:     database.NewVMReplicationContextRepository(db),
		repositoryManager: repositoryManager,
		vmaAPIEndpoint:    vmaAPIEndpoint,
		vmaClient:         &http.Client{Timeout: 30 * time.Second},
	}
}

// BackupRequest represents a request to create a VM backup
type BackupRequest struct {
	// VM identification
	VMContextID string `json:"vm_context_id"` // Required: VM context identifier
	VMName      string `json:"vm_name"`       // Required: VM name
	DiskID      int    `json:"disk_id"`       // Required: Disk number (0, 1, 2...)

	// Backup configuration
	RepositoryID string             `json:"repository_id"` // Required: Target repository
	BackupType   storage.BackupType `json:"backup_type"`   // full or incremental
	PolicyID     string             `json:"policy_id"`     // Optional: Backup policy

	// VMware CBT configuration
	ChangeID         string `json:"change_id,omitempty"`          // Current CBT change ID
	PreviousChangeID string `json:"previous_change_id,omitempty"` // For incremental backups

	// Metadata
	TotalBytes int64                  `json:"total_bytes"`           // VM disk total size
	Metadata   storage.BackupMetadata `json:"metadata,omitempty"`    // Platform-specific metadata
	Tags       map[string]string      `json:"tags,omitempty"`        // Custom tags
}

// BackupResult represents the result of a backup operation
type BackupResult struct {
	BackupID       string             `json:"backup_id"`
	Status         storage.BackupStatus `json:"status"`
	BackupType     storage.BackupType `json:"backup_type"`
	FilePath       string             `json:"file_path"`
	NBDExportName  string             `json:"nbd_export_name,omitempty"`
	BytesTransferred int64            `json:"bytes_transferred"`
	TotalBytes     int64              `json:"total_bytes"`
	ChangeID       string             `json:"change_id,omitempty"`
	ErrorMessage   string             `json:"error_message,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	CompletedAt    *time.Time         `json:"completed_at,omitempty"`
}

// ExecuteBackup orchestrates a complete backup operation
// This is the main entry point for backup workflows
func (be *BackupEngine) ExecuteBackup(ctx context.Context, req *BackupRequest) (*BackupResult, error) {
	log.WithFields(log.Fields{
		"vm_context_id": req.VMContextID,
		"vm_name":       req.VMName,
		"disk_id":       req.DiskID,
		"backup_type":   req.BackupType,
		"repository_id": req.RepositoryID,
	}).Info("🚀 Starting backup workflow orchestration")

	// Validate request
	if err := be.validateBackupRequest(req); err != nil {
		return nil, fmt.Errorf("backup request validation failed: %w", err)
	}

	// Get repository
	repo, err := be.repositoryManager.GetRepository(ctx, req.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Create backup in repository (creates QCOW2 file)
	backupReq := storage.BackupRequest{
		VMContextID:    req.VMContextID,
		VMName:         req.VMName,
		DiskID:         req.DiskID,
		BackupType:     req.BackupType,
		ParentBackupID: "", // Will be set for incrementals
		TotalBytes:     req.TotalBytes,
		ChangeID:       req.ChangeID,
		Metadata:       req.Metadata,
	}

	// For incremental backups, find parent backup
	if req.BackupType == storage.BackupTypeIncremental {
		chain, err := repo.GetBackupChain(ctx, req.VMContextID, req.DiskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get backup chain for incremental: %w", err)
		}
		if chain.LatestBackupID == "" {
			return nil, fmt.Errorf("no parent backup found for incremental - full backup required first")
		}
		backupReq.ParentBackupID = chain.LatestBackupID
		log.WithField("parent_backup_id", chain.LatestBackupID).Info("📎 Using parent backup for incremental")
	}

	// Create backup (repository creates QCOW2 file with proper backing if incremental)
	backup, err := repo.CreateBackup(ctx, backupReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup in repository: %w", err)
	}

	log.WithFields(log.Fields{
		"backup_id": backup.ID,
		"file_path": backup.FilePath,
		"backup_type": backup.BackupType,
	}).Info("✅ Backup file created in repository")

	// Create NBD file export (Task 2 integration)
	exportInfo, err := be.createNBDExport(ctx, req, backup)
	if err != nil {
		// Cleanup: Delete backup file if NBD export fails
		repo.DeleteBackup(ctx, backup.ID)
		return nil, fmt.Errorf("failed to create NBD export: %w", err)
	}

	log.WithFields(log.Fields{
		"export_name": exportInfo.ExportName,
		"nbd_port":    exportInfo.Port,
	}).Info("✅ NBD file export created")

	// Trigger VMA replication
	vmaJobID, err := be.triggerVMAReplication(ctx, req, backup, exportInfo)
	if err != nil {
		// Cleanup: Remove NBD export and delete backup
		nbd.RemoveFileExport(exportInfo.ExportName)
		repo.DeleteBackup(ctx, backup.ID)
		return nil, fmt.Errorf("failed to trigger VMA replication: %w", err)
	}

	log.WithFields(log.Fields{
		"backup_id":  backup.ID,
		"vma_job_id": vmaJobID,
	}).Info("✅ VMA replication triggered")

	// Create backup job record in database
	if err := be.createBackupJobRecord(ctx, req, backup, exportInfo.ExportName); err != nil {
		log.WithError(err).Warn("Failed to create backup job record - continuing")
	}

	// Build result
	result := &BackupResult{
		BackupID:       backup.ID,
		Status:         backup.Status,
		BackupType:     backup.BackupType,
		FilePath:       backup.FilePath,
		NBDExportName:  exportInfo.ExportName,
		TotalBytes:     backup.TotalBytes,
		ChangeID:       backup.ChangeID,
		CreatedAt:      backup.CreatedAt,
	}

	log.WithFields(log.Fields{
		"backup_id":   backup.ID,
		"backup_type": backup.BackupType,
		"file_path":   backup.FilePath,
	}).Info("🎉 Backup workflow orchestration completed successfully")

	return result, nil
}

// validateBackupRequest validates a backup request
func (be *BackupEngine) validateBackupRequest(req *BackupRequest) error {
	if req.VMContextID == "" {
		return fmt.Errorf("vm_context_id is required")
	}
	if req.VMName == "" {
		return fmt.Errorf("vm_name is required")
	}
	if req.RepositoryID == "" {
		return fmt.Errorf("repository_id is required")
	}
	if req.BackupType != storage.BackupTypeFull && req.BackupType != storage.BackupTypeIncremental {
		return fmt.Errorf("invalid backup_type: %s", req.BackupType)
	}
	if req.TotalBytes <= 0 {
		return fmt.Errorf("total_bytes must be greater than 0")
	}
	if req.BackupType == storage.BackupTypeIncremental && req.PreviousChangeID == "" {
		return fmt.Errorf("previous_change_id required for incremental backups")
	}

	return nil
}

// createNBDExport creates an NBD file export for the backup (Task 2 integration)
func (be *BackupEngine) createNBDExport(ctx context.Context, req *BackupRequest, backup *storage.Backup) (*nbd.ExportInfo, error) {
	log.WithFields(log.Fields{
		"backup_id": backup.ID,
		"file_path": backup.FilePath,
	}).Info("🔗 Creating NBD file export for backup")

	// Determine read-write mode
	// Incremental backups need read-write for writing changed blocks
	readWrite := backup.BackupType == storage.BackupTypeIncremental

	// Get file system path for NBD export
	filePath := backup.FilePath
	if !filepath.IsAbs(filePath) {
		return nil, fmt.Errorf("backup file path must be absolute: %s", filePath)
	}

	// Create NBD file export using Task 2 implementation
	exportInfo, err := nbd.CreateFileExport(
		req.VMContextID,
		req.DiskID,
		string(backup.BackupType),
		filePath,
		readWrite,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create NBD file export: %w", err)
	}

	log.WithFields(log.Fields{
		"export_name": exportInfo.ExportName,
		"port":        exportInfo.Port,
		"read_write":  readWrite,
	}).Info("✅ NBD file export created via SIGHUP reload")

	return exportInfo, nil
}

// triggerVMAReplication triggers the VMA (Capture Agent) to perform replication
func (be *BackupEngine) triggerVMAReplication(
	ctx context.Context,
	req *BackupRequest,
	backup *storage.Backup,
	exportInfo *nbd.ExportInfo,
) (string, error) {
	log.WithFields(log.Fields{
		"vm_context_id": req.VMContextID,
		"export_name":   exportInfo.ExportName,
	}).Info("📡 Triggering VMA replication for backup")

	// Build VMA replication request
	vmaRequest := map[string]interface{}{
		"job_id":             backup.ID,
		"vm_name":            req.VMName,
		"vm_context_id":      req.VMContextID,
		"disk_id":            req.DiskID,
		"nbd_export_name":    exportInfo.ExportName,
		"nbd_port":           exportInfo.Port,
		"replication_type":   string(backup.BackupType), // "full" or "incremental"
		"change_id":          req.ChangeID,
		"previous_change_id": req.PreviousChangeID,
		"operation":          "backup", // Distinguish from migration
	}

	// Serialize request
	requestBody, err := json.Marshal(vmaRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal VMA request: %w", err)
	}

	// Call VMA API to start replication
	vmaURL := fmt.Sprintf("%s/api/v1/replicate", be.vmaAPIEndpoint)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", vmaURL, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create VMA request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := be.vmaClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call VMA API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("VMA API returned error status: %d", resp.StatusCode)
	}

	// Parse VMA response
	var vmaResponse struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vmaResponse); err != nil {
		return "", fmt.Errorf("failed to decode VMA response: %w", err)
	}

	log.WithFields(log.Fields{
		"vma_job_id": vmaResponse.JobID,
		"status":     vmaResponse.Status,
	}).Info("✅ VMA replication started")

	return vmaResponse.JobID, nil
}

// createBackupJobRecord creates a backup_jobs database record
func (be *BackupEngine) createBackupJobRecord(
	ctx context.Context,
	req *BackupRequest,
	backup *storage.Backup,
	nbdExportName string,
) error {
	now := time.Now()
	jobRecord := &database.BackupJob{
		ID:            backup.ID,
		VMContextID:   req.VMContextID,
		VMName:        req.VMName,
		RepositoryID:  req.RepositoryID,
		PolicyID:      req.PolicyID,
		BackupType:    string(backup.BackupType),
		Status:        "running",
		RepositoryPath: backup.FilePath,
		ParentBackupID: backup.ParentBackupID,
		ChangeID:      backup.ChangeID,
		TotalBytes:    backup.TotalBytes,
		CreatedAt:     backup.CreatedAt,
		StartedAt:     &now,
	}

	if err := be.backupJobRepo.Create(ctx, jobRecord); err != nil {
		return fmt.Errorf("failed to create backup job record: %w", err)
	}

	log.WithField("backup_id", backup.ID).Info("✅ Backup job record created in database")
	return nil
}

// GetBackupStatus retrieves the current status of a backup operation
func (be *BackupEngine) GetBackupStatus(ctx context.Context, backupID string) (*BackupResult, error) {
	// Get backup job from database
	job, err := be.backupJobRepo.GetByID(ctx, backupID)
	if err != nil {
		return nil, fmt.Errorf("backup job not found: %w", err)
	}

	result := &BackupResult{
		BackupID:         job.ID,
		Status:           storage.BackupStatus(job.Status),
		BackupType:       storage.BackupType(job.BackupType),
		FilePath:         job.RepositoryPath,
		BytesTransferred: job.BytesTransferred,
		TotalBytes:       job.TotalBytes,
		ChangeID:         job.ChangeID,
		ErrorMessage:     job.ErrorMessage,
		CreatedAt:        job.CreatedAt,
	}

	if job.CompletedAt != nil {
		result.CompletedAt = job.CompletedAt
	}

	return result, nil
}

// CompleteBackup marks a backup as completed and updates chain
func (be *BackupEngine) CompleteBackup(ctx context.Context, backupID string, changeID string, bytesTransferred int64) error {
	log.WithFields(log.Fields{
		"backup_id":         backupID,
		"bytes_transferred": bytesTransferred,
		"change_id":         changeID,
	}).Info("📝 Marking backup as completed")

	// Update backup job status
	now := time.Now()
	if err := be.backupJobRepo.Update(ctx, backupID, map[string]interface{}{
		"status":            "completed",
		"change_id":         changeID,
		"bytes_transferred": bytesTransferred,
		"completed_at":      now,
	}); err != nil {
		return fmt.Errorf("failed to update backup job: %w", err)
	}

	// Update backup chain
	job, err := be.backupJobRepo.GetByID(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup job: %w", err)
	}

	// Update backup chain (chain tracking is managed by repository layer)
	// The backup_chains table is automatically updated by the repository when backups are created/deleted
	log.WithField("vm_context_id", job.VMContextID).Debug("Backup chain management handled by repository layer")

	// Remove NBD export (backup complete)
	// Parse export name from backup ID if needed
	// For now, we'll need to track this separately or clean up periodically

	log.WithField("backup_id", backupID).Info("✅ Backup marked as completed")
	return nil
}

// FailBackup marks a backup as failed
func (be *BackupEngine) FailBackup(ctx context.Context, backupID string, errorMessage string) error {
	log.WithFields(log.Fields{
		"backup_id":     backupID,
		"error_message": errorMessage,
	}).Warn("❌ Marking backup as failed")

	now := time.Now()
	if err := be.backupJobRepo.Update(ctx, backupID, map[string]interface{}{
		"status":        "failed",
		"error_message": errorMessage,
		"completed_at":  now,
	}); err != nil {
		return fmt.Errorf("failed to update backup job: %w", err)
	}

	return nil
}

// ListBackups lists all backups for a VM context
func (be *BackupEngine) ListBackups(ctx context.Context, vmContextID string) ([]*BackupResult, error) {
	jobs, err := be.backupJobRepo.ListByVMContext(ctx, vmContextID)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	results := make([]*BackupResult, len(jobs))
	for i, job := range jobs {
		results[i] = &BackupResult{
			BackupID:         job.ID,
			Status:           storage.BackupStatus(job.Status),
			BackupType:       storage.BackupType(job.BackupType),
			FilePath:         job.RepositoryPath,
			BytesTransferred: job.BytesTransferred,
			TotalBytes:       job.TotalBytes,
			ChangeID:         job.ChangeID,
			ErrorMessage:     job.ErrorMessage,
			CreatedAt:        job.CreatedAt,
		}
		if job.CompletedAt != nil {
			results[i].CompletedAt = job.CompletedAt
		}
	}

	return results, nil
}
