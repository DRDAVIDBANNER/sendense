// Package handlers provides REST API endpoints for backup operations
// Task 5: Backup API Endpoints - Expose BackupEngine via REST API
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/services"
	"github.com/vexxhost/migratekit-sha/storage"
	"github.com/vexxhost/migratekit-sha/workflows"
)

// BackupHandler provides REST API endpoints for backup operations
// Integrates with BackupEngine (Task 3) to provide API-driven backup automation
type BackupHandler struct {
	backupEngine      *workflows.BackupEngine
	backupJobRepo     *database.BackupJobRepository
	vmContextRepo     *database.VMReplicationContextRepository
	vmDiskRepo        *database.VMDiskRepository      // 🆕 NEW: For querying discovery-populated disk info
	portAllocator     *services.NBDPortAllocator       // 🆕 NEW: Dynamic NBD port allocation
	qemuManager       *services.QemuNBDManager         // 🆕 NEW: qemu-nbd process management
	credentialService *services.VMwareCredentialService // 🆕 NEW: For getting decrypted vCenter credentials
	db                database.Connection
}

// NewBackupHandler creates a new backup API handler
func NewBackupHandler(
	db database.Connection,
	backupEngine *workflows.BackupEngine,
	portAllocator *services.NBDPortAllocator,
	qemuManager *services.QemuNBDManager,
	credentialService *services.VMwareCredentialService,
) *BackupHandler {
	return &BackupHandler{
		backupEngine:      backupEngine,
		backupJobRepo:     database.NewBackupJobRepository(db),
		vmContextRepo:     database.NewVMReplicationContextRepository(db),
		vmDiskRepo:        database.NewVMDiskRepository(db),                // 🆕 NEW: For discovery-populated disk info
		portAllocator:     portAllocator,                                   // 🆕 NEW: NBD port allocation
		qemuManager:       qemuManager,                                     // 🆕 NEW: qemu-nbd management
		credentialService: credentialService,                               // 🆕 NEW: VMware credential service
		db:                db,
	}
}

// ========================================================================
// REQUEST/RESPONSE MODELS
// ========================================================================

// BackupStartRequest represents a request to start a VM backup (ALL disks)
// ⚠️ IMPORTANT: Backups are VM-level, not disk-level, to maintain VMware snapshot consistency
type BackupStartRequest struct {
	VMName       string            `json:"vm_name"`                  // Required: VM name (ALL disks backed up)
	BackupType   string            `json:"backup_type"`              // Required: "full" or "incremental"
	RepositoryID string            `json:"repository_id"`            // Required: Target repository ID
	PolicyID     string            `json:"policy_id,omitempty"`      // Optional: Backup policy ID
	Tags         map[string]string `json:"tags,omitempty"`           // Optional: Custom tags
	// NO disk_id field - backups are VM-level to prevent data corruption from multiple snapshots
}

// DiskBackupResult represents backup result for a single disk in a multi-disk VM
type DiskBackupResult struct {
	DiskID       int    `json:"disk_id"`              // Disk number (0, 1, 2...)
	NBDPort      int    `json:"nbd_port"`             // Allocated NBD port
	ExportName   string `json:"nbd_export_name"`      // NBD export name
	QCOW2Path    string `json:"qcow2_path"`           // QCOW2 file path
	QemuNBDPID   int    `json:"qemu_nbd_pid"`         // qemu-nbd process ID
	Status       string `json:"status"`               // "port_allocated", "qemu_started", "failed"
	ErrorMessage string `json:"error_message,omitempty"` // Error details if status=failed
}

// BackupResponse represents a VM backup job response (ALL disks)
type BackupResponse struct {
	BackupID         string              `json:"backup_id"`
	VMContextID      string              `json:"vm_context_id"`
	VMName           string              `json:"vm_name"`
	DiskResults      []DiskBackupResult  `json:"disk_results"`           // 🆕 NEW: Results for ALL disks
	NBDTargetsString string              `json:"nbd_targets_string"`     // 🆕 NEW: Multi-disk NBD targets for SBC
	BackupType       string              `json:"backup_type"`
	RepositoryID     string              `json:"repository_id"`
	PolicyID         string              `json:"policy_id,omitempty"`
	Status           string              `json:"status"`
	FilePath         string              `json:"file_path,omitempty"`       // Deprecated - use disk_results
	BytesTransferred int64               `json:"bytes_transferred"`
	TotalBytes       int64               `json:"total_bytes"`
	DisksCount       int                 `json:"disks_count"`             // 🆕 NEW: Number of disks in this backup
	ChangeID         string              `json:"change_id,omitempty"`
	ErrorMessage     string              `json:"error_message,omitempty"`
	CreatedAt        string              `json:"created_at"`
	StartedAt        string              `json:"started_at,omitempty"`
	CompletedAt      string              `json:"completed_at,omitempty"`
	Tags             map[string]string   `json:"tags,omitempty"`
}

// BackupListResponse represents a list of backups
type BackupListResponse struct {
	Backups []*BackupResponse `json:"backups"`
	Total   int               `json:"total"`
}

// BackupChainResponse represents a backup chain
type BackupChainResponse struct {
	ChainID       string            `json:"chain_id"`
	VMContextID   string            `json:"vm_context_id"`
	VMName        string            `json:"vm_name"`
	DiskID        int               `json:"disk_id"`
	RepositoryID  string            `json:"repository_id"`
	FullBackupID  string            `json:"full_backup_id"`
	Backups       []*BackupResponse `json:"backups"`
	TotalSizeBytes int64            `json:"total_size_bytes"`
	BackupCount   int               `json:"backup_count"`
}

// Note: ErrorResponse is defined in auth.go and reused here

// ========================================================================
// API ENDPOINTS
// ========================================================================

// StartBackup handles POST /api/v1/backup/start
// Triggers a full or incremental backup of a VM disk
func (bh *BackupHandler) StartBackup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = ctx // Reserved for future database operations

	// Parse request
	var req BackupStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		bh.sendError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Validate required fields
	if req.VMName == "" {
		bh.sendError(w, http.StatusBadRequest, "vm_name is required", "")
		return
	}
	if req.RepositoryID == "" {
		bh.sendError(w, http.StatusBadRequest, "repository_id is required", "")
		return
	}
	if req.BackupType != "full" && req.BackupType != "incremental" {
		bh.sendError(w, http.StatusBadRequest, "backup_type must be 'full' or 'incremental'", "")
		return
	}

	log.WithFields(log.Fields{
		"vm_name":       req.VMName,
		"backup_type":   req.BackupType,
		"repository_id": req.RepositoryID,
	}).Info("🎯 Starting VM backup (multi-disk)")

	// ========================================================================
	// STEP 1: Get VM context
	// ========================================================================
	vmContext, err := bh.vmContextRepo.GetVMContextByName(req.VMName)
	if err != nil {
		log.WithError(err).Error("Failed to get VM context")
		bh.sendError(w, http.StatusNotFound, "VM not found", err.Error())
		return
	}

	// ========================================================================
	// STEP 2: Get ALL disks for VM
	// ========================================================================
	vmDisks, err := bh.vmDiskRepo.GetByVMContextID(vmContext.ContextID)
	if err != nil {
		log.WithError(err).Error("Failed to get VM disks")
		bh.sendError(w, http.StatusInternalServerError, "failed to get VM disks", err.Error())
		return
	}

	if len(vmDisks) == 0 {
		log.Error("No disks found for VM")
		bh.sendError(w, http.StatusNotFound, "No disks found for VM",
			"VM has no disks in database - ensure VM discovery completed")
		return
	}

	log.WithFields(log.Fields{
		"vm_name":    req.VMName,
		"disk_count": len(vmDisks),
	}).Info("📀 Found disks for multi-disk backup")

	// ========================================================================
	// STEP 2.5: Find or create vm_backup_contexts record (NEW ARCHITECTURE!)
	// ========================================================================
	var vmBackupContext database.VMBackupContext
	err = bh.db.GetGormDB().
		Where("vm_name = ? AND repository_id = ?", req.VMName, req.RepositoryID).
		First(&vmBackupContext).Error
	
	if err != nil {
		// Context doesn't exist - create it
		vmBackupContext = database.VMBackupContext{
			ContextID:    fmt.Sprintf("ctx-backup-%s-%d", req.VMName, time.Now().Unix()),
			VMName:       req.VMName,
			VMwareVMID:   vmContext.VMwareVMID,
			VMPath:       vmContext.VMPath,
			VCenterHost:  vmContext.VCenterHost,
			Datacenter:   vmContext.Datacenter,
			RepositoryID: req.RepositoryID,
		}
		
		if err := bh.db.GetGormDB().Create(&vmBackupContext).Error; err != nil {
			log.WithError(err).Error("Failed to create vm_backup_contexts record")
			bh.sendError(w, http.StatusInternalServerError, "failed to create backup context", err.Error())
			return
		}
		
		log.WithField("context_id", vmBackupContext.ContextID).Info("✅ Created new vm_backup_context")
	} else {
		log.WithField("context_id", vmBackupContext.ContextID).Info("📋 Using existing vm_backup_context")
	}

	// ========================================================================
	// STEP 3: Prepare backup for each disk using BackupEngine
	// ========================================================================
	backupJobID := fmt.Sprintf("backup-%s-%d", req.VMName, time.Now().Unix())
	diskResults := make([]DiskBackupResult, len(vmDisks))
	var preparationErr error

	// STEP 3.1: Create parent backup_jobs record FIRST (for backup_disks FK constraint)
	// This ensures the FK reference exists before per-disk records are created
	now := time.Now()
	parentJobInsert := `
		INSERT INTO backup_jobs (
			id, vm_backup_context_id, vm_context_id, vm_name, repository_id,
			backup_type, status, repository_path, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	err = bh.db.GetGormDB().Exec(parentJobInsert,
		backupJobID, vmBackupContext.ContextID, vmContext.ContextID, req.VMName, req.RepositoryID,
		req.BackupType, "running", "/multi-disk-parent", now,
	).Error
	
	if err != nil {
		log.WithError(err).Error("Failed to create parent backup_jobs record")
		bh.sendError(w, http.StatusInternalServerError, "failed to create parent backup job", err.Error())
		return
	}
	
	log.WithField("backup_job_id", backupJobID).Info("✅ Created parent backup_jobs record for multi-disk backup")

	// Cleanup function for failure scenarios
	defer func() {
		if preparationErr != nil {
			log.WithField("backup_job_id", backupJobID).Warn("🧹 CLEANUP: Failure detected, cleaning up resources")
			
			// BackupEngine handles cleanup internally (qemu-nbd stop, port release, QCOW2 delete)
			// Only need to stop qemu-nbd processes that were successfully started
			for i := range diskResults {
				if diskResults[i].NBDPort > 0 {
					bh.qemuManager.Stop(diskResults[i].NBDPort)
					bh.portAllocator.Release(diskResults[i].NBDPort)
				}
			}
			
			// Also delete parent backup_jobs record on failure
			bh.db.GetGormDB().Exec("DELETE FROM backup_jobs WHERE id = ?", backupJobID)
		}
	}()

	// Prepare each disk backup using BackupEngine
	for i, vmDisk := range vmDisks {
		// Use loop index as disk ID since unit_number can be duplicated (VMware bug)
		diskIndex := i
		
		// For incremental backups, look up previous change_id
		var previousChangeID string
		if req.BackupType == "incremental" {
			// NEW ARCHITECTURE: Query backup_disks table with JOIN to vm_backup_contexts
			// This uses the vm_backup_contexts + backup_disks architecture (v2.16.0+)
			var prevDisk database.BackupDisk
			err := bh.db.GetGormDB().
				Table("backup_disks bd").
				Select("bd.*").
				Joins("JOIN vm_backup_contexts vbc ON bd.vm_backup_context_id = vbc.context_id").
				Where("vbc.vm_name = ? AND bd.disk_index = ? AND bd.status = ? AND bd.disk_change_id IS NOT NULL", req.VMName, diskIndex, "completed").
				Order("bd.completed_at DESC").
				First(&prevDisk).Error
			
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"vm_name":    req.VMName,
					"disk_index": diskIndex,
				}).Error("❌ No previous backup found for incremental - full backup required first")
				preparationErr = fmt.Errorf("no previous backup found for incremental - full backup required first")
				bh.sendError(w, http.StatusBadRequest, "no previous backup found", "full backup required before incremental")
				return
			}
			
			if prevDisk.DiskChangeID != nil {
				previousChangeID = *prevDisk.DiskChangeID
			}
			log.WithFields(log.Fields{
				"disk_index":         diskIndex,
				"previous_backup_id": prevDisk.BackupJobID,
				"previous_change_id": previousChangeID,
			}).Info("📎 Found previous backup for incremental")
		}

		// Build BackupRequest for this disk
		backupReq := &workflows.BackupRequest{
			VMContextID:       vmContext.ContextID,                // Legacy replication context
			VMBackupContextID: vmBackupContext.ContextID,          // NEW: Backup context for proper parent-child relationships
			ParentJobID:       backupJobID,                        // NEW: Parent job ID that backup client knows about
			VMName:            req.VMName,
			DiskID:            diskIndex, // Use loop index (0, 1, 2...) to avoid unit_number duplicates
			BackupType:        storage.BackupType(req.BackupType),
			RepositoryID:      req.RepositoryID,
			TotalBytes:        int64(vmDisk.SizeGB) * 1024 * 1024 * 1024,
			PreviousChangeID:  previousChangeID, // For incremental backups
			Tags:              req.Tags,
		}

		// Call BackupEngine to prepare disk (parent lookup + QCOW2 create + qemu-nbd)
		result, err := bh.backupEngine.PrepareBackupDisk(ctx, backupReq)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"unit_number": vmDisk.UnitNumber,
				"disk_id":     vmDisk.DiskID,
			}).Error("❌ Failed to prepare backup disk")
			preparationErr = err
			bh.sendError(w, http.StatusInternalServerError, "failed to prepare backup disk", err.Error())
			return
		}

		// Store result
		diskResults[i] = DiskBackupResult{
			DiskID:        diskIndex, // Use loop index consistently
			NBDPort:       result.NBDPort,
			ExportName:    result.NBDExportName,
			QCOW2Path:     result.FilePath,
			QemuNBDPID:    result.QemuNBDPID,
			Status:        "prepared",
			ErrorMessage:  "",
		}

		log.WithFields(log.Fields{
			"disk_id":      vmDisk.UnitNumber,
			"backup_id":    result.BackupID,
			"backup_type":  result.BackupType,
			"nbd_port":     result.NBDPort,
			"qemu_nbd_pid": result.QemuNBDPID,
		}).Info("✅ Disk backup prepared successfully")
	}

	// ========================================================================
	// STEP 6: Build NBD targets string for SendenseBackupClient
	// ========================================================================
	// Format: "vmware_disk_key:nbd://host:port/export,vmware_disk_key:nbd://..."
	nbdTargets := []string{}
	for i, result := range diskResults {
		// Calculate VMware disk key (use loop index since unit_number broken - both disks = 0)
		diskKey := i + 2000
		
		// 🔍 DEBUG: Log disk key calculation
		log.WithFields(log.Fields{
			"loop_index":   i,
			"disk_key":     diskKey,
			"result_index": result.DiskID,
			"nbd_port":     result.NBDPort,
			"export_name":  result.ExportName,
		}).Info("🔍 DEBUG: Building NBD target for disk")
		
		nbdURL := fmt.Sprintf("nbd://127.0.0.1:%d/%s", result.NBDPort, result.ExportName)
		nbdTargets = append(nbdTargets, fmt.Sprintf("%d:%s", diskKey, nbdURL))
	}
	nbdTargetsString := strings.Join(nbdTargets, ",")

	log.WithFields(log.Fields{
		"nbd_targets":  nbdTargetsString,
		"target_count": len(nbdTargets),
	}).Info("🎯 Built multi-disk NBD targets string")
	
	// 🔍 DEBUG: Final NBD targets string for verification
	log.WithField("final_nbd_targets", nbdTargetsString).Info("🔍 DEBUG: Final NBD targets string to be sent to SNA")

	// ========================================================================
	// STEP 6.5: Get VMware credentials using credential service
	// ========================================================================
	if vmContext.CredentialID == nil {
		log.Error("VM context has no credential_id set")
		bh.sendError(w, http.StatusBadRequest, "VM context missing credential_id", "")
		return
	}
	
	creds, err := bh.credentialService.GetCredentials(r.Context(), *vmContext.CredentialID)
	if err != nil {
		log.WithError(err).Error("Failed to get VMware credentials")
		bh.sendError(w, http.StatusInternalServerError, "failed to get VMware credentials", err.Error())
		return
	}
	
	// ========================================================================
	// STEP 7: Call SNA VMA API (via reverse tunnel on port 9081)
	// ========================================================================
	snaReq := map[string]interface{}{
		"vm_name":           req.VMName,
		"vcenter_host":      creds.VCenterHost,
		"vcenter_user":      creds.Username,
		"vcenter_password":  creds.Password, // Already decrypted by credential service
		"vm_path":           vmContext.VMPath,
		"nbd_host":          "127.0.0.1",      // Via SSH tunnel
		"nbd_targets":       nbdTargetsString, // ← Multi-disk NBD targets!
		"job_id":            backupJobID,
		"backup_type":       req.BackupType,
		"previous_change_id": "PLACEHOLDER",  // ✅ NEW: Backup client queries SHA database per-disk for actual change_ids
	}

	jsonData, _ := json.Marshal(snaReq)
	snaURL := "http://localhost:9081/api/v1/backup/start"

	resp, err := http.Post(snaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.WithError(err).Error("❌ Failed to call SNA VMA API")
		preparationErr = err
		bh.sendError(w, http.StatusInternalServerError, "failed to call SNA API", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		log.WithFields(log.Fields{
			"status": resp.StatusCode,
		}).Error("❌ SNA VMA API returned error")
		preparationErr = fmt.Errorf("SNA API error: %d", resp.StatusCode)
		bh.sendError(w, http.StatusInternalServerError, "SNA API error", fmt.Sprintf("status: %d", resp.StatusCode))
		return
	}

	log.WithFields(log.Fields{
		"vm_name":       req.VMName,
		"disk_count":    len(diskResults),
		"sna_url":       snaURL,
		"backup_job_id": backupJobID,
	}).Info("✅ SNA VMA API called successfully for multi-disk backup")

	// ========================================================================
	// NOTE: Parent backup_jobs record already created via RAW SQL at line ~243
	// No need for duplicate GORM Create() here - it was causing duplicate key errors
	// ========================================================================
	// STEP 8: Return response with ALL disk details
	// ========================================================================
	response := BackupResponse{
		BackupID:         backupJobID,
		VMContextID:      vmContext.ContextID,
		VMName:           req.VMName,
		DiskResults:      diskResults,
		NBDTargetsString: nbdTargetsString,
		BackupType:       req.BackupType,
		RepositoryID:     req.RepositoryID,
		PolicyID:         req.PolicyID,
		Status:           "started",
		CreatedAt:        time.Now().Format(time.RFC3339),
		Tags:             req.Tags,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.WithFields(log.Fields{
		"backup_id":  backupJobID,
		"vm_name":    req.VMName,
		"disk_count": len(diskResults),
	}).Info("🎉 Multi-disk VM backup started successfully")
}

// ListBackups handles GET /api/v1/backup/list
// Returns a list of backups with optional filtering
func (bh *BackupHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	vmName := r.URL.Query().Get("vm_name")
	vmContextID := r.URL.Query().Get("vm_context_id")
	repositoryID := r.URL.Query().Get("repository_id")
	status := r.URL.Query().Get("status")
	backupType := r.URL.Query().Get("backup_type")

	log.WithFields(log.Fields{
		"vm_name":       vmName,
		"vm_context_id": vmContextID,
		"repository_id": repositoryID,
		"status":        status,
		"backup_type":   backupType,
	}).Info("📋 Listing backups")

	var backups []*database.BackupJob
	var err error

	// Query based on filters
	if vmContextID != "" {
		backups, err = bh.backupJobRepo.ListByVMContext(ctx, vmContextID)
	} else if vmName != "" {
		// Get VM context by name
		vmContext, contextErr := bh.vmContextRepo.GetVMContextByName(vmName)
		if contextErr != nil {
			bh.sendError(w, http.StatusNotFound, "VM not found", contextErr.Error())
			return
		}
		backups, err = bh.backupJobRepo.ListByVMContext(ctx, vmContext.ContextID)
	} else if repositoryID != "" {
		backups, err = bh.backupJobRepo.ListByRepository(ctx, repositoryID)
	} else if status != "" {
		backups, err = bh.backupJobRepo.ListByStatus(ctx, status)
	} else {
		// No filter - return empty list (could implement paginated full list if needed)
		backups = []*database.BackupJob{}
	}

	if err != nil {
		log.WithError(err).Error("Failed to list backups")
		bh.sendError(w, http.StatusInternalServerError, "failed to list backups", err.Error())
		return
	}

	// Apply additional filters
	filteredBackups := bh.filterBackups(backups, backupType, status)

	// Filter out per-disk records (only show parent records)
	parentBackups := make([]*database.BackupJob, 0, len(filteredBackups))
	for _, backup := range filteredBackups {
		// Skip per-disk records (they contain "-disk" in the ID)
		if !strings.Contains(backup.ID, "-disk") {
			parentBackups = append(parentBackups, backup)
		}
	}

	// Convert to API responses
	responses := make([]*BackupResponse, 0, len(parentBackups))
	for _, backup := range parentBackups {
		responses = append(responses, bh.convertToBackupResponse(backup))
	}

	response := &BackupListResponse{
		Backups: responses,
		Total:   len(responses),
	}

	log.WithField("count", response.Total).Info("✅ Backups listed successfully")
	bh.sendJSON(w, http.StatusOK, response)
}

// GetBackupDetails handles GET /api/v1/backup/{backup_id}
// Returns detailed information about a specific backup
func (bh *BackupHandler) GetBackupDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get backup ID from URL
	vars := mux.Vars(r)
	backupID := vars["backup_id"]

	if backupID == "" {
		bh.sendError(w, http.StatusBadRequest, "backup_id is required", "")
		return
	}

	log.WithField("backup_id", backupID).Info("🔍 Getting backup details")

	// Get backup from database
	backup, err := bh.backupJobRepo.GetByID(ctx, backupID)
	if err != nil {
		bh.sendError(w, http.StatusNotFound, "backup not found", err.Error())
		return
	}

	// Convert to API response
	response := bh.convertToBackupResponse(backup)

	log.WithField("backup_id", backupID).Info("✅ Backup details retrieved")
	bh.sendJSON(w, http.StatusOK, response)
}

// DeleteBackup handles DELETE /api/v1/backup/{backup_id}
// Deletes a backup from the repository and database
func (bh *BackupHandler) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get backup ID from URL
	vars := mux.Vars(r)
	backupID := vars["backup_id"]

	if backupID == "" {
		bh.sendError(w, http.StatusBadRequest, "backup_id is required", "")
		return
	}

	log.WithField("backup_id", backupID).Info("🗑️  Deleting backup")

	// Get backup details first
	backup, err := bh.backupJobRepo.GetByID(ctx, backupID)
	if err != nil {
		bh.sendError(w, http.StatusNotFound, "backup not found", err.Error())
		return
	}

	// TODO: Check immutability settings from repository
	// For now, allow deletion

	// Delete from database (CASCADE will handle related records)
	if err := bh.backupJobRepo.Delete(ctx, backupID); err != nil {
		log.WithError(err).Error("Failed to delete backup")
		bh.sendError(w, http.StatusInternalServerError, "failed to delete backup", err.Error())
		return
	}

	// TODO: Delete physical backup file from repository
	// This should be handled by repository.DeleteBackup() method

	log.WithFields(log.Fields{
		"backup_id": backupID,
		"vm_name":   backup.VMName,
	}).Info("✅ Backup deleted successfully")

	bh.sendJSON(w, http.StatusOK, map[string]string{
		"message":   "backup deleted successfully",
		"backup_id": backupID,
	})
}

// GetBackupChain handles GET /api/v1/backup/chain
// Returns the backup chain for a VM disk (full + incrementals)
func (bh *BackupHandler) GetBackupChain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	vmContextID := r.URL.Query().Get("vm_context_id")
	vmName := r.URL.Query().Get("vm_name")
	diskIDStr := r.URL.Query().Get("disk_id")

	if vmContextID == "" && vmName == "" {
		bh.sendError(w, http.StatusBadRequest, "vm_context_id or vm_name is required", "")
		return
	}

	diskID := 0 // Default to first disk
	if diskIDStr != "" {
		var err error
		diskID, err = strconv.Atoi(diskIDStr)
		if err != nil {
			bh.sendError(w, http.StatusBadRequest, "invalid disk_id", err.Error())
			return
		}
	}

	// Get VM context if vm_name provided
	if vmContextID == "" && vmName != "" {
		vmContext, err := bh.vmContextRepo.GetVMContextByName(vmName)
		if err != nil {
			bh.sendError(w, http.StatusNotFound, "VM not found", err.Error())
			return
		}
		vmContextID = vmContext.ContextID
	}

	log.WithFields(log.Fields{
		"vm_context_id": vmContextID,
		"disk_id":       diskID,
	}).Info("🔗 Getting backup chain")

	// Get backup chain from repository
	backups, err := bh.backupJobRepo.GetBackupChain(ctx, vmContextID, diskID)
	if err != nil {
		log.WithError(err).Error("Failed to get backup chain")
		bh.sendError(w, http.StatusInternalServerError, "failed to get backup chain", err.Error())
		return
	}

	if len(backups) == 0 {
		bh.sendJSON(w, http.StatusOK, &BackupChainResponse{
			Backups:     []*BackupResponse{},
			BackupCount: 0,
		})
		return
	}

	// Build chain response
	var fullBackupID string
	var totalSize int64
	responses := make([]*BackupResponse, 0, len(backups))

	for _, backup := range backups {
		responses = append(responses, bh.convertToBackupResponse(backup))
		totalSize += backup.TotalBytes

		if backup.BackupType == "full" {
			fullBackupID = backup.ID
		}
	}

	chainID := fmt.Sprintf("%s-disk%d-chain", vmContextID, diskID)

	response := &BackupChainResponse{
		ChainID:        chainID,
		VMContextID:    vmContextID,
		VMName:         backups[0].VMName,
		DiskID:         diskID,
		RepositoryID:   backups[0].RepositoryID,
		FullBackupID:   fullBackupID,
		Backups:        responses,
		TotalSizeBytes: totalSize,
		BackupCount:    len(backups),
	}

	log.WithFields(log.Fields{
		"chain_id":     chainID,
		"backup_count": response.BackupCount,
	}).Info("✅ Backup chain retrieved")

	bh.sendJSON(w, http.StatusOK, response)
}

// CompleteBackup handles POST /api/v1/backups/{backup_id}/complete
// Called by sendense-backup-client when backup finishes to record change_id
func (bh *BackupHandler) CompleteBackup(w http.ResponseWriter, r *http.Request) {
	// Extract backup ID from URL path
	backupID := mux.Vars(r)["backup_id"]
	if backupID == "" {
		bh.sendError(w, http.StatusBadRequest, "missing backup_id", "backup_id path parameter is required")
		return
	}

	// Parse request body
	var req struct {
		ChangeID         string `json:"change_id"`
		DiskID           int    `json:"disk_id"`           // NEW: numeric disk ID for multi-disk VMs
		BytesTransferred int64  `json:"bytes_transferred"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		bh.sendError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Validate required fields
	if req.ChangeID == "" {
		bh.sendError(w, http.StatusBadRequest, "missing change_id", "change_id field is required")
		return
	}

	log.WithFields(log.Fields{
		"backup_id":         backupID,
		"disk_id":           req.DiskID,
		"change_id":         req.ChangeID,
		"bytes_transferred": req.BytesTransferred,
	}).Info("📝 Completing backup job and storing change_id")

	// Call BackupEngine.CompleteBackup()
	err := bh.backupEngine.CompleteBackup(r.Context(), backupID, req.DiskID, req.ChangeID, req.BytesTransferred)
	if err != nil {
		// Check if backup job not found
		if strings.Contains(err.Error(), "not found") {
			bh.sendError(w, http.StatusNotFound, "backup job not found", err.Error())
			return
		}
		bh.sendError(w, http.StatusInternalServerError, "failed to complete backup", err.Error())
		return
	}

	// Success response
	response := map[string]interface{}{
		"status":     "completed",
		"backup_id":  backupID,
		"change_id":  req.ChangeID,
		"message":    "Backup completed successfully, change_id recorded",
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	log.WithField("backup_id", backupID).Info("✅ Backup completed and change_id stored")
	bh.sendJSON(w, http.StatusOK, response)
}

// GetChangeID retrieves the last successful change_id for a VM disk
// This enables incremental backups by providing the previous snapshot point
// @Summary Get previous change ID for backup
// @Description Get the change ID from the last successful backup for incremental support
// @Tags backups
// @Produce json
// @Param vm_name query string true "VM name (e.g., pgtest1)"
// @Param disk_id query int false "Disk ID (numeric 0, 1, 2..., defaults to 0)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/backups/changeid [get]
func (bh *BackupHandler) GetChangeID(w http.ResponseWriter, r *http.Request) {
	vmName := r.URL.Query().Get("vm_name")
	diskIDStr := r.URL.Query().Get("disk_id")
	
	if vmName == "" {
		bh.sendError(w, http.StatusBadRequest, "missing vm_name", "vm_name query parameter is required")
		return
	}
	
	diskID := 0 // Default to first disk
	if diskIDStr != "" {
		var err error
		diskID, err = strconv.Atoi(diskIDStr)
		if err != nil {
			bh.sendError(w, http.StatusBadRequest, "invalid disk_id", "disk_id must be numeric")
			return
		}
	}
	
	log.WithFields(log.Fields{
		"vm_name": vmName,
		"disk_id": diskID,
	}).Info("📡 Querying previous change_id for incremental backup")
	
	// NEW ARCHITECTURE: Query backup_disks with JOIN to vm_backup_contexts
	// This eliminates any time-window matching - direct FK relationship
	var backupDisk database.BackupDisk
	err := bh.db.GetGormDB().
		Table("backup_disks").
		Select("backup_disks.*").
		Joins("JOIN vm_backup_contexts ON backup_disks.vm_backup_context_id = vm_backup_contexts.context_id").
		Where("vm_backup_contexts.vm_name = ? AND backup_disks.disk_index = ? AND backup_disks.status = ? AND backup_disks.disk_change_id IS NOT NULL AND backup_disks.disk_change_id != ''",
			vmName, diskID, "completed").
		Order("backup_disks.completed_at DESC").
		First(&backupDisk).Error
	
	if err != nil {
		// Check if it's a "not found" error (not an actual error condition)
		if strings.Contains(err.Error(), "record not found") {
			// No previous backup found - return empty (not an error for first backup)
			log.WithFields(log.Fields{
				"vm_name": vmName,
				"disk_id": diskID,
			}).Info("📋 No previous backup found - this will be a full backup")
			
			bh.sendJSON(w, http.StatusOK, map[string]string{
				"vm_name":   vmName,
				"disk_id":   fmt.Sprintf("%d", diskID),
				"change_id": "",
				"message":   "No previous backup found",
			})
			return
		}
		
		// Actual database error
		log.WithError(err).WithFields(log.Fields{
			"vm_name": vmName,
			"disk_id": diskID,
		}).Error("Failed to query previous backup")
		bh.sendError(w, http.StatusInternalServerError, "database error", err.Error())
		return
	}
	
	// Return change_id from backup_disks
	changeID := ""
	if backupDisk.DiskChangeID != nil {
		changeID = *backupDisk.DiskChangeID
	}
	
	log.WithFields(log.Fields{
		"vm_name":      vmName,
		"disk_id":      diskID,
		"change_id":    changeID,
		"backup_job_id": backupDisk.BackupJobID,
	}).Info("✅ Previous change_id found from backup_disks")
	
	bh.sendJSON(w, http.StatusOK, map[string]string{
		"vm_name":      vmName,
		"disk_id":      fmt.Sprintf("%d", diskID), // Return as string for JSON compatibility
		"change_id":    changeID,
		"backup_job_id": backupDisk.BackupJobID,
		"message":      "Previous change_id found",
	})
}

// ========================================================================
// HELPER METHODS
// ========================================================================

// convertToBackupResponse converts a database BackupJob to API BackupResponse
func (bh *BackupHandler) convertToBackupResponse(job *database.BackupJob) *BackupResponse {
	// Dereference policy_id pointer
	policyID := ""
	if job.PolicyID != nil {
		policyID = *job.PolicyID
	}

	// Count associated disks for this backup
	var disksCount int64
	err := bh.db.GetGormDB().Table("backup_disks").
		Where("backup_job_id = ?", job.ID).
		Count(&disksCount).Error
	if err != nil {
		log.WithError(err).Warn("Failed to count disks for backup")
		disksCount = 1 // Default fallback
	}

	response := &BackupResponse{
		BackupID:         job.ID,
		VMContextID:      job.VMContextID,
		VMName:           job.VMName,
		BackupType:       job.BackupType,
		RepositoryID:     job.RepositoryID,
		PolicyID:         policyID,
		Status:           job.Status,
		FilePath:         job.RepositoryPath,
		BytesTransferred: job.BytesTransferred,
		TotalBytes:       job.TotalBytes,
		DisksCount:       int(disksCount), // 🆕 NEW: Count of disks in this backup
		ChangeID:         job.ChangeID,
		ErrorMessage:     job.ErrorMessage,
		CreatedAt:        job.CreatedAt.Format("2006-01-02T15:04:05Z"),
		// Note: DiskResults will be populated by caller for multi-disk backups
	}

	if job.StartedAt != nil {
		response.StartedAt = job.StartedAt.Format("2006-01-02T15:04:05Z")
	}
	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt.Format("2006-01-02T15:04:05Z")
	}

	return response
}

// filterBackups applies additional filtering to backup list
func (bh *BackupHandler) filterBackups(backups []*database.BackupJob, backupType, status string) []*database.BackupJob {
	if backupType == "" && status == "" {
		return backups
	}

	filtered := make([]*database.BackupJob, 0, len(backups))
	for _, backup := range backups {
		// Apply backup_type filter
		if backupType != "" && backup.BackupType != backupType {
			continue
		}
		// Apply status filter (if not already filtered by query)
		if status != "" && backup.Status != status {
			continue
		}
		filtered = append(filtered, backup)
	}

	return filtered
}

// timePtr returns a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}

// sendJSON sends a JSON response
func (bh *BackupHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (bh *BackupHandler) sendError(w http.ResponseWriter, status int, message string, details string) {
	response := &ErrorResponse{
		Error:   message,
		Details: details,
	}
	bh.sendJSON(w, status, response)
}

// RegisterRoutes registers backup API routes
// RegisterRoutes registers all backup API endpoints following REST conventions
// PROJECT RULE: RESTful resource naming, consistent with project standards
func (bh *BackupHandler) RegisterRoutes(r *mux.Router) {
	log.Info("🔗 Registering backup API routes (RESTful resource-based)")

	// Backup resource endpoints (following REST conventions: /backups not /backup)
	// Route order matters: specific routes BEFORE parameterized routes to avoid conflicts
	
	// 1. POST /api/v1/backups - Start new backup
	r.HandleFunc("/backups", bh.StartBackup).Methods("POST")

	// 2. GET /api/v1/backups/stats - Get backup statistics for VM (MUST come before /backups)
	r.HandleFunc("/backups/stats", bh.GetBackupStats).Methods("GET")

	// 3. GET /api/v1/backups - List all backups (with optional filters)
	r.HandleFunc("/backups", bh.ListBackups).Methods("GET")

	// 4. GET /api/v1/backups/changeid - Get previous change_id for incremental (MUST come before parameterized routes)
	r.HandleFunc("/backups/changeid", bh.GetChangeID).Methods("GET")

	// 5. GET /api/v1/backups/{vm_name}/chain - Get backup chain for VM (MUST come before /{backup_id})
	r.HandleFunc("/backups/{vm_name}/chain", bh.GetBackupChain).Methods("GET")

	// 6. POST /api/v1/backups/{backup_id}/complete - Complete backup and record change_id (MUST come before /{backup_id})
	r.HandleFunc("/backups/{backup_id}/complete", bh.CompleteBackup).Methods("POST")

	// 7. GET /api/v1/backups/{backup_id} - Get backup details
	r.HandleFunc("/backups/{backup_id}", bh.GetBackupDetails).Methods("GET")

	// 8. DELETE /api/v1/backups/{backup_id} - Delete backup
	r.HandleFunc("/backups/{backup_id}", bh.DeleteBackup).Methods("DELETE")

	log.Info("✅ Backup API routes registered - 8 RESTful endpoints (start, stats, complete, changeid, list, get, delete, chain)")
}

// GetBackupStats returns backup statistics for a VM in a specific repository
// GET /api/v1/backups/stats?vm_name={name}&repository_id={repo}
func (bh *BackupHandler) GetBackupStats(w http.ResponseWriter, r *http.Request) {
	vmName := r.URL.Query().Get("vm_name")
	repoID := r.URL.Query().Get("repository_id")

	if vmName == "" || repoID == "" {
		bh.sendError(w, http.StatusBadRequest, "vm_name and repository_id are required", "")
		return
	}

	var stats struct {
		BackupCount      int    `json:"backup_count"`
		TotalSizeBytes   int64  `json:"total_size_bytes"`
		LastBackupAt     string `json:"last_backup_at"`
	}

	// Query backup statistics (CRITICAL: filter out per-disk records with id NOT LIKE '%-disk%')
	err := bh.db.GetGormDB().
		Table("backup_jobs").
		Select("COUNT(*) as backup_count, SUM(IFNULL(bytes_transferred, 0)) as total_size_bytes, MAX(completed_at) as last_backup_at").
		Where("vm_name = ? AND repository_id = ? AND status = ? AND id NOT LIKE ?",
			vmName, repoID, "completed", "%-disk%").
		Scan(&stats).Error

	if err != nil {
		log.WithError(err).Error("Failed to get backup stats")
		bh.sendError(w, http.StatusInternalServerError, "Failed to get backup stats", err.Error())
		return
	}

	log.WithFields(log.Fields{
		"vm_name":       vmName,
		"repository_id": repoID,
		"backup_count":  stats.BackupCount,
		"total_size":    stats.TotalSizeBytes,
	}).Info("✅ Backup stats retrieved")

	bh.sendJSON(w, http.StatusOK, stats)
}
