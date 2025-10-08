// Package workflows provides orchestration for VMware backup operations
// Following project rules: modular design, clean interfaces, comprehensive error handling
package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/nbd"
	"github.com/vexxhost/migratekit-sha/services"
	"github.com/vexxhost/migratekit-sha/storage"
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

	// SNA client for triggering replications
	snaAPIEndpoint string
	snaClient      *http.Client
}

// NewBackupEngine creates a new backup workflow orchestration engine
func NewBackupEngine(
	db database.Connection,
	repositoryManager *storage.RepositoryManager,
	snaAPIEndpoint string,
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
		snaAPIEndpoint:    snaAPIEndpoint,
		snaClient:         &http.Client{Timeout: 30 * time.Second},
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

	// Trigger SNA replication
	snaJobID, err := be.triggerVMAReplication(ctx, req, backup, exportInfo)
	if err != nil {
		// Cleanup: Remove NBD export and delete backup
		nbd.RemoveFileExport(exportInfo.ExportName)
		repo.DeleteBackup(ctx, backup.ID)
		return nil, fmt.Errorf("failed to trigger SNA replication: %w", err)
	}

	log.WithFields(log.Fields{
		"backup_id":  backup.ID,
		"vma_job_id": snaJobID,
	}).Info("✅ SNA replication triggered")

	// Update backup job status to 'running' (record already created by storage layer)
	if err := be.updateBackupJobStatus(ctx, backup.ID, "running"); err != nil {
		log.WithError(err).Warn("Failed to update backup job status - continuing")
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
	// ALL backups need read-write - SNA writes VMware disk data INTO the QCOW2 file
	readWrite := true // Both full and incremental backups write to QCOW2

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

// triggerVMAReplication triggers the SNA (Capture Agent) to perform replication
func (be *BackupEngine) triggerVMAReplication(
	ctx context.Context,
	req *BackupRequest,
	backup *storage.Backup,
	exportInfo *nbd.ExportInfo,
) (string, error) {
	log.WithFields(log.Fields{
		"vm_context_id": req.VMContextID,
		"export_name":   exportInfo.ExportName,
	}).Info("📡 Triggering SNA replication for backup")

	// Get VM context for vCenter info
	var vmContext database.VMReplicationContext
	if err := be.db.GetGormDB().Where("context_id = ?", req.VMContextID).First(&vmContext).Error; err != nil {
		return "", fmt.Errorf("failed to get VM context: %w", err)
	}

	// Get vCenter credentials
	var vcenterUsername, vcenterPassword string
	encryptionService, err := services.NewCredentialEncryptionService()
	if err != nil {
		log.WithError(err).Warn("Failed to initialize encryption service, using fallback credentials")
		vcenterUsername = "administrator@vsphere.local"
		vcenterPassword = "EmyGVoBFesGQc47-"
	} else {
		credentialService := services.NewVMwareCredentialService(&be.db, encryptionService)
		creds, err := credentialService.GetDefaultCredentials(ctx)
		if err != nil {
			log.WithError(err).Warn("Failed to get default credentials, using fallback")
			vcenterUsername = "administrator@vsphere.local"
			vcenterPassword = "EmyGVoBFesGQc47-"
		} else {
			vcenterUsername = creds.Username
			vcenterPassword = creds.Password
		}
	}

	// Build NBD target
	shaNbdHost := os.Getenv("SHA_NBD_HOST")
	if shaNbdHost == "" {
		shaNbdHost = "localhost" // Default for tunnel
	}
	devicePath := fmt.Sprintf("nbd://%s:%d/%s", shaNbdHost, exportInfo.Port, exportInfo.ExportName)

	// Build SNA replication request (same format as migrations)
	snaRequest := map[string]interface{}{
		"job_id":   backup.ID,
		"vcenter":  vmContext.VCenterHost,
		"username": vcenterUsername,
		"password": vcenterPassword,
		"vm_paths": []string{vmContext.VMPath},
		"oma_url":  "http://localhost:8082",
		"nbd_targets": []map[string]interface{}{
			{
				"device_path":     devicePath,
				"vmware_disk_key": fmt.Sprintf("%d", req.DiskID+2000), // disk-2000, disk-2001, etc.
			},
		},
	}

	// Serialize request
	requestBody, err := json.Marshal(snaRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal SNA request: %w", err)
	}

	// Call SNA API to start replication
	snaURL := fmt.Sprintf("%s/api/v1/replicate", be.snaAPIEndpoint)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", snaURL, bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create SNA request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := be.snaClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call SNA API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("SNA API returned error status: %d", resp.StatusCode)
	}

	// Parse SNA response
	var snaResponse struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snaResponse); err != nil {
		return "", fmt.Errorf("failed to decode SNA response: %w", err)
	}

	log.WithFields(log.Fields{
		"vma_job_id": snaResponse.JobID,
		"status":     snaResponse.Status,
	}).Info("✅ SNA replication started")

	return snaResponse.JobID, nil
}

// updateBackupJobStatus updates the backup job status to 'running' with started_at timestamp
// Note: backup_jobs record is already created by storage layer with status='pending'
func (be *BackupEngine) updateBackupJobStatus(
	ctx context.Context,
	backupID string,
	status string,
) error {
	now := time.Now()
	
	// Update existing backup_job record (created by storage layer)
	err := be.db.GetGormDB().Model(&database.BackupJob{}).
		Where("id = ?", backupID).
		Updates(map[string]interface{}{
			"status":     status,
			"started_at": now,
		}).Error
	
	if err != nil {
		return fmt.Errorf("failed to update backup job status: %w", err)
	}

	log.WithFields(log.Fields{
		"backup_id": backupID,
		"status":    status,
	}).Info("✅ Backup job status updated")
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
