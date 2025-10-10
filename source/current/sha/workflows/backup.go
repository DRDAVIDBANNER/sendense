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
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/services"
	"github.com/vexxhost/migratekit-sha/storage"
)

// BackupEngine orchestrates VMware backup operations to repository storage
// Integrates with qemu-nbd process management and storage infrastructure
type BackupEngine struct {
	// Repository dependencies
	db              database.Connection
	backupJobRepo   *database.BackupJobRepository
	backupChainRepo storage.BackupChainRepository
	vmContextRepo   *database.VMReplicationContextRepository

	// Storage infrastructure
	repositoryManager *storage.RepositoryManager

	// NBD infrastructure (qemu-nbd approach)
	portAllocator *services.NBDPortAllocator
	qemuManager   *services.QemuNBDManager

	// SNA client for triggering replications
	snaAPIEndpoint string
	snaClient      *http.Client
}

// NewBackupEngine creates a new backup workflow orchestration engine
func NewBackupEngine(
	db database.Connection,
	repositoryManager *storage.RepositoryManager,
	portAllocator *services.NBDPortAllocator,
	qemuManager *services.QemuNBDManager,
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
		portAllocator:     portAllocator,
		qemuManager:       qemuManager,
		snaAPIEndpoint:    snaAPIEndpoint,
		snaClient:         &http.Client{Timeout: 30 * time.Second},
	}
}

// BackupRequest represents a request to create a VM backup
type BackupRequest struct {
	// VM identification
	VMContextID       string `json:"vm_context_id"`        // Required: VM replication context identifier (legacy)
	VMBackupContextID string `json:"vm_backup_context_id"` // Required: VM backup context identifier (NEW ARCHITECTURE!)
	ParentJobID       string `json:"parent_job_id"`        // Required: Parent backup job ID (for backup_disks FK)
	VMName            string `json:"vm_name"`              // Required: VM name
	DiskID            int    `json:"disk_id"`              // Required: Disk number (0, 1, 2...)

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
	BackupID         string               `json:"backup_id"`
	Status           storage.BackupStatus `json:"status"`
	BackupType       storage.BackupType   `json:"backup_type"`
	FilePath         string               `json:"file_path"`
	NBDExportName    string               `json:"nbd_export_name,omitempty"`
	NBDPort          int                  `json:"nbd_port,omitempty"`          // qemu-nbd port
	QemuNBDPID       int                  `json:"qemu_nbd_pid,omitempty"`      // qemu-nbd process ID
	BytesTransferred int64                `json:"bytes_transferred"`
	TotalBytes       int64                `json:"total_bytes"`
	ChangeID         string               `json:"change_id,omitempty"`
	ErrorMessage     string               `json:"error_message,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	CompletedAt      *time.Time           `json:"completed_at,omitempty"`
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
		VMContextID:       req.VMContextID,       // Legacy replication context
		VMBackupContextID: req.VMBackupContextID, // NEW: Backup context for proper FK relationships
		ParentJobID:       req.ParentJobID,       // NEW: Parent job ID for backup_disks FK
		VMName:            req.VMName,
		DiskID:            req.DiskID,
		BackupType:        req.BackupType,
		ParentBackupID:    "", // Will be set for incrementals (QCOW2 backing file)
		TotalBytes:        req.TotalBytes,
		ChangeID:          req.ChangeID,
		Metadata:          req.Metadata,
	}

	// For incremental backups, find parent backup
	if req.BackupType == storage.BackupTypeIncremental {
		// Use VMBackupContextID (new architecture) not VMContextID (legacy replication)
		contextID := req.VMBackupContextID
		if contextID == "" {
			contextID = req.VMContextID // Fallback for old code
		}
		chain, err := repo.GetBackupChain(ctx, contextID, req.DiskID)
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
		"backup_id":   backup.ID,
		"file_path":   backup.FilePath,
		"backup_type": backup.BackupType,
	}).Info("✅ Backup file created in repository")

	// Allocate NBD port
	exportName := fmt.Sprintf("%s-disk%d", req.VMName, req.DiskID)
	diskJobID := fmt.Sprintf("%s-disk%d", backup.ID, req.DiskID)
	
	nbdPort, err := be.portAllocator.Allocate(diskJobID, req.VMName, exportName)
	if err != nil {
		// Cleanup: Delete backup file if port allocation fails
		repo.DeleteBackup(ctx, backup.ID)
		return nil, fmt.Errorf("failed to allocate NBD port: %w", err)
	}

	log.WithFields(log.Fields{
		"nbd_port":    nbdPort,
		"export_name": exportName,
	}).Info("✅ NBD port allocated")

	// Start qemu-nbd process
	qemuProcess, err := be.qemuManager.Start(
		nbdPort,
		exportName,
		backup.FilePath,
		backup.ID,
		req.VMName,
		req.DiskID,
	)
	if err != nil {
		// Cleanup: Release port and delete backup file
		be.portAllocator.Release(nbdPort)
		repo.DeleteBackup(ctx, backup.ID)
		return nil, fmt.Errorf("failed to start qemu-nbd: %w", err)
	}

	log.WithFields(log.Fields{
		"qemu_nbd_pid": qemuProcess.PID,
		"nbd_port":     nbdPort,
		"export_name":  exportName,
	}).Info("✅ qemu-nbd process started")

	// Trigger SNA replication
	snaJobID, err := be.triggerSNAReplication(ctx, req, backup, nbdPort, exportName)
	if err != nil {
		// Cleanup: Stop qemu-nbd, release port, delete backup
		be.qemuManager.Stop(nbdPort)
		be.portAllocator.Release(nbdPort)
		repo.DeleteBackup(ctx, backup.ID)
		return nil, fmt.Errorf("failed to trigger SNA replication: %w", err)
	}

	log.WithFields(log.Fields{
		"backup_id":  backup.ID,
		"sna_job_id": snaJobID,
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
		NBDExportName:  exportName,
		NBDPort:        nbdPort,
		QemuNBDPID:     qemuProcess.PID,
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

// PrepareBackupDisk prepares a single disk for backup without triggering SNA replication
// This is used by handlers for multi-disk backups where SNA is called once for all disks
func (be *BackupEngine) PrepareBackupDisk(ctx context.Context, req *BackupRequest) (*BackupResult, error) {
	log.WithFields(log.Fields{
		"vm_context_id": req.VMContextID,
		"vm_name":       req.VMName,
		"disk_id":       req.DiskID,
		"backup_type":   req.BackupType,
		"repository_id": req.RepositoryID,
	}).Info("🚀 Preparing backup disk (without SNA trigger)")

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
		VMContextID:       req.VMContextID,       // Legacy replication context
		VMBackupContextID: req.VMBackupContextID, // NEW: Backup context for proper FK relationships
		ParentJobID:       req.ParentJobID,       // NEW: Parent job ID for backup_disks FK
		VMName:            req.VMName,
		DiskID:            req.DiskID,
		BackupType:        req.BackupType,
		ParentBackupID:    "", // Will be set for incrementals (QCOW2 backing file)
		TotalBytes:        req.TotalBytes,
		ChangeID:          req.ChangeID,
		Metadata:          req.Metadata,
	}

	// For incremental backups, find parent backup
	if req.BackupType == storage.BackupTypeIncremental {
		// Use VMBackupContextID (new architecture) not VMContextID (legacy replication)
		contextID := req.VMBackupContextID
		if contextID == "" {
			contextID = req.VMContextID // Fallback for old code
		}
		chain, err := repo.GetBackupChain(ctx, contextID, req.DiskID)
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
		"backup_id":   backup.ID,
		"file_path":   backup.FilePath,
		"backup_type": backup.BackupType,
	}).Info("✅ Backup file created in repository")

	// Allocate NBD port
	exportName := fmt.Sprintf("%s-disk%d", req.VMName, req.DiskID)
	diskJobID := fmt.Sprintf("%s-disk%d", backup.ID, req.DiskID)
	
	nbdPort, err := be.portAllocator.Allocate(diskJobID, req.VMName, exportName)
	if err != nil {
		// Cleanup: Delete backup file if port allocation fails
		repo.DeleteBackup(ctx, backup.ID)
		return nil, fmt.Errorf("failed to allocate NBD port: %w", err)
	}

	log.WithFields(log.Fields{
		"nbd_port":    nbdPort,
		"export_name": exportName,
	}).Info("✅ NBD port allocated")

	// Start qemu-nbd process
	qemuProcess, err := be.qemuManager.Start(
		nbdPort,
		exportName,
		backup.FilePath,
		backup.ID,
		req.VMName,
		req.DiskID,
	)
	if err != nil {
		// Cleanup: Release port and delete backup file
		be.portAllocator.Release(nbdPort)
		repo.DeleteBackup(ctx, backup.ID)
		return nil, fmt.Errorf("failed to start qemu-nbd: %w", err)
	}

	log.WithFields(log.Fields{
		"qemu_nbd_pid": qemuProcess.PID,
		"nbd_port":     nbdPort,
		"export_name":  exportName,
	}).Info("✅ qemu-nbd process started")

	// Build result (WITHOUT triggering SNA)
	result := &BackupResult{
		BackupID:       backup.ID,
		Status:         backup.Status,
		BackupType:     backup.BackupType,
		FilePath:       backup.FilePath,
		NBDExportName:  exportName,
		NBDPort:        nbdPort,
		QemuNBDPID:     qemuProcess.PID,
		TotalBytes:     backup.TotalBytes,
		ChangeID:       backup.ChangeID,
		CreatedAt:      backup.CreatedAt,
	}

	log.WithFields(log.Fields{
		"backup_id":   backup.ID,
		"backup_type": backup.BackupType,
		"nbd_port":    nbdPort,
	}).Info("🎉 Backup disk prepared successfully (awaiting SNA trigger)")

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

// createNBDExport - REMOVED: Replaced with qemu-nbd approach (portAllocator + qemuManager)
// NBD exports are now created using individual qemu-nbd processes per backup instead of
// the NBD-server config.d + SIGHUP approach. This provides better isolation and port management.

// triggerSNAReplication triggers the SNA (Capture Agent) to perform replication
func (be *BackupEngine) triggerSNAReplication(
	ctx context.Context,
	req *BackupRequest,
	backup *storage.Backup,
	nbdPort int,
	exportName string,
) (string, error) {
	log.WithFields(log.Fields{
		"vm_context_id": req.VMContextID,
		"export_name":   exportName,
		"nbd_port":      nbdPort,
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
		shaNbdHost = "127.0.0.1" // Default for tunnel
	}
	devicePath := fmt.Sprintf("nbd://%s:%d/%s", shaNbdHost, nbdPort, exportName)

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
func (be *BackupEngine) CompleteBackup(ctx context.Context, backupID string, diskID int, changeID string, bytesTransferred int64) error {
	log.WithFields(log.Fields{
		"backup_id":         backupID,
		"disk_id":           diskID,
		"bytes_transferred": bytesTransferred,
		"change_id":         changeID,
	}).Info("📝 Completing backup disk")

	now := time.Now()
	
	// NEW ARCHITECTURE: Update backup_disks table directly (no time-window hack!)
	result := be.db.GetGormDB().
		Model(&database.BackupDisk{}).
		Where("backup_job_id = ? AND disk_index = ?", backupID, diskID).
		Updates(map[string]interface{}{
			"status":            "completed",
			"disk_change_id":    changeID,
			"bytes_transferred": bytesTransferred,
			"completed_at":      now,
		})
	
	if result.Error != nil {
		return fmt.Errorf("failed to update backup disk: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("backup disk not found: job_id=%s disk_index=%d", backupID, diskID)
	}
	
	log.WithFields(log.Fields{
		"backup_id": backupID,
		"disk_id":   diskID,
	}).Info("✅ Backup disk marked completed")

	// Check if all disks for this backup job are completed
	var totalDisks, completedDisks int64
	be.db.GetGormDB().Model(&database.BackupDisk{}).
		Where("backup_job_id = ?", backupID).
		Count(&totalDisks)
	be.db.GetGormDB().Model(&database.BackupDisk{}).
		Where("backup_job_id = ? AND status = ?", backupID, "completed").
		Count(&completedDisks)
	
	log.WithFields(log.Fields{
		"backup_id":       backupID,
		"total_disks":     totalDisks,
		"completed_disks": completedDisks,
	}).Debug("Checking backup completion status")
	
	// If all disks completed, mark parent backup_jobs record as completed
	if totalDisks > 0 && totalDisks == completedDisks {
		// Get current job to check if telemetry already set bytes_transferred
		var currentJob database.BackupJob
		be.db.GetGormDB().Where("id = ?", backupID).First(&currentJob)
		
		// Preserve telemetry data if present, otherwise aggregate from disks
		var finalBytesTransferred int64
		if currentJob.BytesTransferred > 0 {
			// Telemetry already set this - keep it!
			finalBytesTransferred = currentJob.BytesTransferred
			log.WithFields(log.Fields{
				"backup_id":              backupID,
				"bytes_from_telemetry":   finalBytesTransferred,
			}).Info("✅ Using bytes_transferred from telemetry (real-time SBC data)")
		} else {
			// No telemetry - aggregate from disks as fallback
			be.db.GetGormDB().
				Model(&database.BackupDisk{}).
				Select("SUM(IFNULL(bytes_transferred, 0))").
				Where("backup_job_id = ?", backupID).
				Scan(&finalBytesTransferred)
			
			log.WithFields(log.Fields{
				"backup_id":              backupID,
				"bytes_from_disks":       finalBytesTransferred,
			}).Debug("Aggregated bytes_transferred from disks (fallback)")
		}
		
		result = be.db.GetGormDB().
			Model(&database.BackupJob{}).
			Where("id = ?", backupID).
			Updates(map[string]interface{}{
				"status":            "completed",
				"bytes_transferred": finalBytesTransferred,
				"completed_at":      now,
			})
		
		if result.Error != nil {
			log.WithError(result.Error).Warn("Failed to update parent backup job status")
			// Don't fail - disk completion is what matters
		} else {
			log.WithField("backup_id", backupID).Info("✅ All disks completed - backup job finished")
			
			// Update backup context statistics
			var backupJob database.BackupJob
			if err := be.db.GetGormDB().Where("id = ?", backupID).First(&backupJob).Error; err == nil {
				if backupJob.VMBackupContextID != nil {
					be.db.GetGormDB().
						Model(&database.VMBackupContext{}).
						Where("context_id = ?", *backupJob.VMBackupContextID).
						Updates(map[string]interface{}{
							"successful_backups": be.db.GetGormDB().Raw("successful_backups + 1"),
							"last_backup_id":     backupID,
							"last_backup_type":   backupJob.BackupType,
							"last_backup_at":     now,
						})
					
					// 🆕 FIX: Update backup_chains for each completed disk
					var completedDisksForChain []database.BackupDisk
					if err := be.db.GetGormDB().
						Where("backup_job_id = ? AND status = ?", backupID, "completed").
						Find(&completedDisksForChain).Error; err == nil {
						
						for _, disk := range completedDisksForChain {
							// Find the actual per-disk backup_jobs record by vm_backup_context_id and disk_index
							// Per-disk jobs have format: backup-{vm_name}-disk{index}-{timestamp}
							var perDiskJob database.BackupJob
							err := be.db.GetGormDB().
								Where("vm_backup_context_id = ? AND backup_type = ?", backupJob.VMBackupContextID, backupJob.BackupType).
								Where("id LIKE ?", fmt.Sprintf("%%disk%d%%", disk.DiskIndex)).
								Where("created_at >= ?", backupJob.CreatedAt.Add(-1*time.Minute)). // Within 1 minute
								Where("created_at <= ?", backupJob.CreatedAt.Add(1*time.Minute)).
								Order("created_at DESC").
								First(&perDiskJob).Error
							
							if err != nil {
								log.WithError(err).WithFields(log.Fields{
									"backup_id":  backupID,
									"disk_index": disk.DiskIndex,
								}).Warn("⚠️ Could not find per-disk backup_jobs record")
								continue
							}
							
							// Update per-disk backup_jobs record to "completed"
							be.db.GetGormDB().
								Model(&database.BackupJob{}).
								Where("id = ?", perDiskJob.ID).
								Updates(map[string]interface{}{
									"status":       "completed",
									"completed_at": now,
								})
							
							log.WithField("per_disk_job_id", perDiskJob.ID).Info("✅ Updated per-disk backup_jobs status")
							
							// Get backup file size for chain update
							var fileSize int64
							if disk.QCOW2Path != nil && *disk.QCOW2Path != "" {
								if fileInfo, err := os.Stat(*disk.QCOW2Path); err == nil {
									fileSize = fileInfo.Size()
								}
							}
							
							// Add backup to chain (updates total_backups and latest_backup_id)
							chainID := storage.GenerateChainID(*backupJob.VMBackupContextID, disk.DiskIndex)
							
							// Prepare ChangeID (handle *string type)
							changeIDStr := ""
							if disk.DiskChangeID != nil {
								changeIDStr = *disk.DiskChangeID
							}
							
							backupForChain := &storage.Backup{
								ID:             perDiskJob.ID, // Use actual per-disk job ID from database
								VMContextID:    *backupJob.VMBackupContextID,
								DiskID:         disk.DiskIndex,
								BackupType:     storage.BackupType(backupJob.BackupType),
								SizeBytes:      fileSize,
								Status:         "completed",
								ChangeID:       changeIDStr,
								CreatedAt:      perDiskJob.CreatedAt,
								CompletedAt:    &now,
							}
							
							// Use chain manager via repository (get *sql.DB from GORM)
							sqlDB, dbErr := be.db.GetGormDB().DB()
							if dbErr != nil {
								log.WithError(dbErr).Warn("⚠️ Failed to get SQL DB for chain manager")
								continue
							}
							chainMgr := storage.NewChainManager(be.backupChainRepo, sqlDB)
							if err := chainMgr.AddBackupToChain(context.Background(), chainID, backupForChain); err != nil {
								log.WithError(err).WithFields(log.Fields{
									"chain_id":   chainID,
									"backup_id":  perDiskJob.ID,
									"disk_index": disk.DiskIndex,
								}).Warn("⚠️ Failed to add backup to chain (non-fatal)")
							} else {
								log.WithFields(log.Fields{
									"chain_id":   chainID,
									"backup_id":  perDiskJob.ID,
									"disk_index": disk.DiskIndex,
									"file_size":  fileSize,
								}).Info("✅ Added backup to chain")
							}
						}
					}
				}
			}
			
			// 🆕 CLEANUP: Stop qemu-nbd processes and release NBD ports after backup completion
			// This fixes the stale qemu-nbd process bug
			ports := be.portAllocator.GetPortsForBackupJob(backupID)
			if len(ports) > 0 {
				log.WithFields(log.Fields{
					"backup_id":  backupID,
					"port_count": len(ports),
					"ports":      ports,
				}).Info("🧹 Cleaning up qemu-nbd processes for completed backup")
				
				for _, port := range ports {
					// Stop qemu-nbd process
					be.qemuManager.Stop(port)
					// Release port for reuse
					be.portAllocator.Release(port)
				}
				
				log.WithField("backup_id", backupID).Info("✅ qemu-nbd cleanup completed")
			}
		}
	}

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
