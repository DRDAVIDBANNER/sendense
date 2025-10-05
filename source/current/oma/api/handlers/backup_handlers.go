// Package handlers provides REST API endpoints for backup operations
// Task 5: Backup API Endpoints - Expose BackupEngine via REST API
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/storage"
	"github.com/vexxhost/migratekit-oma/workflows"
)

// BackupHandler provides REST API endpoints for backup operations
// Integrates with BackupEngine (Task 3) to provide API-driven backup automation
type BackupHandler struct {
	backupEngine    *workflows.BackupEngine
	backupJobRepo   *database.BackupJobRepository
	vmContextRepo   *database.VMReplicationContextRepository
	db              database.Connection
}

// NewBackupHandler creates a new backup API handler
func NewBackupHandler(
	db database.Connection,
	backupEngine *workflows.BackupEngine,
) *BackupHandler {
	return &BackupHandler{
		backupEngine:  backupEngine,
		backupJobRepo: database.NewBackupJobRepository(db),
		vmContextRepo: database.NewVMReplicationContextRepository(db),
		db:            db,
	}
}

// ========================================================================
// REQUEST/RESPONSE MODELS
// ========================================================================

// BackupStartRequest represents a request to start a backup
type BackupStartRequest struct {
	VMName       string            `json:"vm_name"`                  // Required: VM name
	DiskID       int               `json:"disk_id"`                  // Required: Disk number (0, 1, 2...)
	BackupType   string            `json:"backup_type"`              // Required: "full" or "incremental"
	RepositoryID string            `json:"repository_id"`            // Required: Target repository ID
	PolicyID     string            `json:"policy_id,omitempty"`      // Optional: Backup policy ID
	Tags         map[string]string `json:"tags,omitempty"`           // Optional: Custom tags
}

// BackupResponse represents a backup job response
type BackupResponse struct {
	BackupID         string            `json:"backup_id"`
	VMContextID      string            `json:"vm_context_id"`
	VMName           string            `json:"vm_name"`
	DiskID           int               `json:"disk_id"`
	BackupType       string            `json:"backup_type"`
	RepositoryID     string            `json:"repository_id"`
	PolicyID         string            `json:"policy_id,omitempty"`
	Status           string            `json:"status"`
	FilePath         string            `json:"file_path,omitempty"`
	NBDExportName    string            `json:"nbd_export_name,omitempty"`
	BytesTransferred int64             `json:"bytes_transferred"`
	TotalBytes       int64             `json:"total_bytes"`
	ChangeID         string            `json:"change_id,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
	CreatedAt        string            `json:"created_at"`
	StartedAt        string            `json:"started_at,omitempty"`
	CompletedAt      string            `json:"completed_at,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
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
		"disk_id":       req.DiskID,
		"backup_type":   req.BackupType,
		"repository_id": req.RepositoryID,
	}).Info("📥 Received backup start request")

	// Get VM context
	vmContext, err := bh.vmContextRepo.GetVMContextByName(req.VMName)
	if err != nil {
		bh.sendError(w, http.StatusNotFound, "VM not found", err.Error())
		return
	}

	// Build BackupEngine request
	backupReq := &workflows.BackupRequest{
		VMContextID:  vmContext.ContextID,
		VMName:       req.VMName,
		DiskID:       req.DiskID,
		RepositoryID: req.RepositoryID,
		BackupType:   storage.BackupType(req.BackupType),
		PolicyID:     req.PolicyID,
		Tags:         req.Tags,
		TotalBytes:   0, // Will be determined by BackupEngine
	}

	// Execute backup
	result, err := bh.backupEngine.ExecuteBackup(ctx, backupReq)
	if err != nil {
		log.WithError(err).Error("Failed to start backup")
		bh.sendError(w, http.StatusInternalServerError, "failed to start backup", err.Error())
		return
	}

	// Get full backup details
	backup, err := bh.backupJobRepo.GetByID(ctx, result.BackupID)
	if err != nil {
		log.WithError(err).Warn("Backup started but failed to retrieve details")
		// Return partial response
		response := &BackupResponse{
			BackupID:     result.BackupID,
			VMName:       req.VMName,
			DiskID:       req.DiskID,
			BackupType:   req.BackupType,
			RepositoryID: req.RepositoryID,
			Status:       string(result.Status),
		}
		bh.sendJSON(w, http.StatusAccepted, response)
		return
	}

	// Convert to API response
	response := bh.convertToBackupResponse(backup)

	log.WithFields(log.Fields{
		"backup_id": response.BackupID,
		"status":    response.Status,
	}).Info("✅ Backup started successfully")

	bh.sendJSON(w, http.StatusAccepted, response)
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

	// Convert to API responses
	responses := make([]*BackupResponse, 0, len(filteredBackups))
	for _, backup := range filteredBackups {
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

// ========================================================================
// HELPER METHODS
// ========================================================================

// convertToBackupResponse converts a database BackupJob to API BackupResponse
func (bh *BackupHandler) convertToBackupResponse(job *database.BackupJob) *BackupResponse {
	response := &BackupResponse{
		BackupID:         job.ID,
		VMContextID:      job.VMContextID,
		VMName:           job.VMName,
		DiskID:           job.DiskID,
		BackupType:       job.BackupType,
		RepositoryID:     job.RepositoryID,
		PolicyID:         job.PolicyID,
		Status:           job.Status,
		FilePath:         job.RepositoryPath,
		BytesTransferred: job.BytesTransferred,
		TotalBytes:       job.TotalBytes,
		ChangeID:         job.ChangeID,
		ErrorMessage:     job.ErrorMessage,
		CreatedAt:        job.CreatedAt.Format("2006-01-02T15:04:05Z"),
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
func (bh *BackupHandler) RegisterRoutes(r *mux.Router) {
	log.Info("🔗 Registering backup API routes")

	backup := r.PathPrefix("/backup").Subrouter()

	// Backup operations (specific routes BEFORE parameterized routes)
	backup.HandleFunc("/start", bh.StartBackup).Methods("POST")
	backup.HandleFunc("/list", bh.ListBackups).Methods("GET")
	backup.HandleFunc("/chain", bh.GetBackupChain).Methods("GET") // MUST come before /{backup_id}
	backup.HandleFunc("/{backup_id}", bh.GetBackupDetails).Methods("GET")
	backup.HandleFunc("/{backup_id}", bh.DeleteBackup).Methods("DELETE")

	log.Info("✅ Backup API routes registered (5 endpoints)")
}
