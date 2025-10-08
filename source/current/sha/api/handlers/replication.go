// Package handlers provides HTTP handlers for SHA API endpoints
// Replication job management following project rules: modular design, clean interfaces
package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/vexxhost/migratekit-sha/common"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/models"
	"github.com/vexxhost/migratekit-sha/services"
	"github.com/vexxhost/migratekit-sha/volume"
	"github.com/vexxhost/migratekit-sha/workflows"
)

// ReplicationHandler handles replication job endpoints with automated workflow
type ReplicationHandler struct {
	db              database.Connection
	migrationEngine *workflows.MigrationEngine
	replicationRepo *database.ReplicationJobRepository // Repository for database operations
	jobs            []ReplicationJob                   // Legacy in-memory storage (being phased out)
}

// CreateMigrationRequest represents a simplified migration creation request
type CreateMigrationRequest struct {
	// Required: Source VM information
	SourceVM models.VMInfo `json:"source_vm" binding:"required"`

	// Required: OSSEA configuration to use
	OSSEAConfigID int `json:"ossea_config_id" binding:"required"`

	// Optional: Migration configuration (defaults applied)
	ReplicationType string `json:"replication_type,omitempty"` // "initial" or "incremental"
	TargetNetwork   string `json:"target_network,omitempty"`
	VCenterHost     string `json:"vcenter_host,omitempty"` // Auto-detected if not provided
	Datacenter      string `json:"datacenter,omitempty"`   // Auto-detected if not provided

	// Optional: CBT configuration for incremental migrations
	ChangeID         string `json:"change_id,omitempty"`
	PreviousChangeID string `json:"previous_change_id,omitempty"`
	SnapshotID       string `json:"snapshot_id,omitempty"`

	// Optional: Scheduler metadata (only used when called by scheduler)
	ScheduleExecutionID string `json:"schedule_execution_id,omitempty"`
	VMGroupID           string `json:"vm_group_id,omitempty"`
	ScheduledBy         string `json:"scheduled_by,omitempty"`

	// Optional: Control replication start (defaults to true for backward compatibility)
	StartReplication *bool `json:"start_replication,omitempty"`
}

// StoreChangeIDRequest represents a request to store a ChangeID for a migration job
type StoreChangeIDRequest struct {
	ChangeID         string `json:"change_id" binding:"required"` // VMware ChangeID from migration
	DiskID           string `json:"disk_id,omitempty"`            // Disk identifier (defaults to "disk-2000")
	PreviousChangeID string `json:"previous_change_id,omitempty"` // Previous ChangeID for tracking
}

// MigrationStartResult represents the result of starting an automated migration
type MigrationStartResult struct {
	JobID           string        `json:"job_id"`
	Status          string        `json:"status"`
	ProgressPercent float64       `json:"progress_percent"`
	SourceVM        models.VMInfo `json:"source_vm"`
	CreatedVolumes  []VolumeInfo  `json:"created_volumes"`
	MountedVolumes  []MountInfo   `json:"mounted_volumes"`
	StartedAt       time.Time     `json:"started_at"`
	Message         string        `json:"message"`
}

// VolumeInfo represents information about a created volume
type VolumeInfo struct {
	VolumeID   string `json:"volume_id"`
	VolumeName string `json:"volume_name"`
	SizeGB     int    `json:"size_gb"`
	Status     string `json:"status"`
}

// MountInfo represents information about a mounted volume
type MountInfo struct {
	DevicePath string `json:"device_path"`
	MountPoint string `json:"mount_point"`
	Status     string `json:"status"`
}

// VMContextResult represents the result of adding a VM to management without starting replication
type VMContextResult struct {
	ContextID     string    `json:"context_id"`
	VMName        string    `json:"vm_name"`
	VMwareVMID    string    `json:"vmware_vm_id"`
	CurrentStatus string    `json:"current_status"`
	Message       string    `json:"message"`
	CreatedAt     time.Time `json:"created_at"`
}

// ReplicationJob represents a legacy replication job (being phased out)
type ReplicationJob struct {
	ID               string        `json:"id"`
	SourceVM         models.VMInfo `json:"source_vm"`
	Status           string        `json:"status"`
	Progress         float64       `json:"progress"`
	ReplicationType  string        `json:"replication_type"`
	TargetNetwork    string        `json:"target_network"`
	BytesTransferred int64         `json:"bytes_transferred"`
	TotalBytes       int64         `json:"total_bytes"`
	TransferSpeedBPS int64         `json:"transfer_speed_bps"`
	ChangeID         string        `json:"change_id"`
	ErrorMessage     string        `json:"error_message,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// generateUniqueJobID creates a collision-resistant job ID using millisecond precision + random suffix
// This prevents concurrent scheduler executions from generating duplicate job IDs
func generateUniqueJobID() string {
	// Use millisecond precision timestamp (1000x better than second precision)
	timestamp := time.Now().Format("20060102-150405.000")

	// Add 6-character random suffix to eliminate any remaining collision risk
	randomBytes := make([]byte, 3) // 3 bytes = 6 hex characters
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to nanosecond if random generation fails (extremely rare)
		return fmt.Sprintf("job-%s-%d", timestamp, time.Now().UnixNano()%1000000)
	}

	randomSuffix := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("job-%s-%s", timestamp, randomSuffix)
}

// checkExistingVMContext checks if a VM already exists in vm_replication_contexts by vmware_vm_id
func (h *ReplicationHandler) checkExistingVMContext(vmwareVMID string) (*database.VMReplicationContext, error) {
	// Use GORM to query for existing VM context
	gormDB := h.db.GetGormDB()

	var context database.VMReplicationContext
	err := gormDB.Where("vmware_vm_id = ?", vmwareVMID).First(&context).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// VM not found - this is expected for new VMs
			return nil, nil
		}
		// Database error
		return nil, fmt.Errorf("failed to query VM context: %w", err)
	}

	// VM found - return the existing context
	return &context, nil
}

// createVMContextOnly creates a VM context without starting replication job
func (h *ReplicationHandler) createVMContextOnly(ctx context.Context, req CreateMigrationRequest) (*VMContextResult, error) {
	// Generate context ID
	contextID := fmt.Sprintf("ctx-%s-%s", req.SourceVM.Name, time.Now().Format("20060102-150405"))

	// Set defaults for optional fields
	// 🆕 ENHANCED: Use credential service for vCenter host
	vcenterHost := req.VCenterHost
	if vcenterHost == "" {
		// Get vCenter host from credential service
		if defaultHost := h.getDefaultVCenterHost(ctx); defaultHost != "" {
			vcenterHost = defaultHost
		} else {
			// Fallback to hardcoded during transition
			vcenterHost = "quad-vcenter-01.quadris.local"
		}
	}
	datacenter := req.Datacenter
	if datacenter == "" {
		datacenter = "DatabanxDC"
	}

	// Create VM context record using GORM
	gormDB := h.db.GetGormDB()
	now := time.Now()

	vmContext := &database.VMReplicationContext{
		ContextID:        contextID,
		VMName:           req.SourceVM.Name,
		VMwareVMID:       req.SourceVM.ID,
		VMPath:           req.SourceVM.Path,
		VCenterHost:      vcenterHost,
		Datacenter:       datacenter,
		CurrentStatus:    "discovered",
		CPUCount:         &req.SourceVM.CPUs,
		MemoryMB:         &req.SourceVM.MemoryMB,
		OSType:           &req.SourceVM.OSType,
		PowerState:       &req.SourceVM.PowerState,
		CreatedAt:        now,
		UpdatedAt:        now,
		LastStatusChange: now,
		AutoAdded:        true,
		SchedulerEnabled: true,
	}

	err := gormDB.Create(vmContext).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create VM context: %w", err)
	}

	log.WithFields(log.Fields{
		"context_id":     contextID,
		"vm_name":        req.SourceVM.Name,
		"vmware_vm_id":   req.SourceVM.ID,
		"current_status": "discovered",
	}).Info("📋 VM context created successfully")

	return &VMContextResult{
		ContextID:     contextID,
		VMName:        req.SourceVM.Name,
		VMwareVMID:    req.SourceVM.ID,
		CurrentStatus: "discovered",
		Message:       "VM added to management successfully",
		CreatedAt:     now,
	}, nil
}

// NewReplicationHandler creates a new replication handler with migration workflow engine
func NewReplicationHandler(db database.Connection, mountManager *volume.MountManager, snaProgressPoller workflows.SNAProgressPoller) *ReplicationHandler {
	return &ReplicationHandler{
		db:              db,
		migrationEngine: workflows.NewMigrationEngine(db, mountManager, snaProgressPoller),
		replicationRepo: database.NewReplicationJobRepository(db), // Initialize repository
		jobs:            make([]ReplicationJob, 0),
	}
}

// List handles replication job list requests
// @Summary List replication jobs
// @Description Get list of all replication jobs with their current status
// @Tags replications
// @Security BearerAuth
// @Produce json
// @Success 200 {array} ReplicationJob
// @Router /api/v1/replications [get]
func (h *ReplicationHandler) List(w http.ResponseWriter, r *http.Request) {
	h.writeJSONResponse(w, http.StatusOK, h.jobs)
}

// Create handles automated replication job creation with full OSSEA integration
// @Summary Create automated migration job
// @Description Start a new VM replication job with automatic volume creation, mounting, and CBT tracking
// @Tags replications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param job body CreateMigrationRequest true "Migration job configuration"
// @Success 201 {object} MigrationStartResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/replications [post]
func (h *ReplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// Validate required fields
	if req.SourceVM.ID == "" || req.SourceVM.Name == "" || req.SourceVM.Path == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Source VM information is required", "Missing VM ID, name, or path")
		return
	}

	if req.OSSEAConfigID == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "OSSEA configuration ID is required", "Must specify target OSSEA configuration")
		return
	}

	// Determine if we should start replication (defaults to true for backward compatibility)
	startReplication := true
	if req.StartReplication != nil {
		startReplication = *req.StartReplication
	}

	// Check for existing VM context (by vmware_vm_id)
	existingContext, err := h.checkExistingVMContext(req.SourceVM.ID)
	if err != nil {
		log.WithError(err).WithField("vmware_vm_id", req.SourceVM.ID).Error("Failed to check for existing VM context")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to check VM status", err.Error())
		return
	}

	// Apply different duplicate protection logic based on operation type
	if existingContext != nil {
		if !startReplication {
			// Add to Management: Block all duplicates
			h.writeErrorResponse(w, http.StatusConflict, "VM already exists in management",
				fmt.Sprintf("VM '%s' (%s) is already managed with context ID: %s",
					req.SourceVM.Name, req.SourceVM.ID, existingContext.ContextID))
			return
		} else if existingContext.CurrentJobID != nil && *existingContext.CurrentJobID != "" {
			// Start Replication: Block if VM has active job
			h.writeErrorResponse(w, http.StatusConflict, "VM has active replication job",
				fmt.Sprintf("VM '%s' (%s) already has active job: %s",
					req.SourceVM.Name, req.SourceVM.ID, *existingContext.CurrentJobID))
			return
		}
		// Start Replication on existing VM with no active job: Allow (continue)
		log.WithFields(log.Fields{
			"vm_name":        req.SourceVM.Name,
			"context_id":     existingContext.ContextID,
			"current_status": existingContext.CurrentStatus,
		}).Info("🔄 Starting replication on existing managed VM")
	}

	// Generate unique job ID with collision protection (only if starting replication)
	var jobID string
	if startReplication {
		jobID = generateUniqueJobID()
	}

	// Convert to workflow request
	migrationReq := &workflows.MigrationRequest{
		SourceVM:         req.SourceVM,
		VCenterHost:      req.VCenterHost,
		Datacenter:       req.Datacenter,
		JobID:            jobID,
		OSSEAConfigID:    req.OSSEAConfigID,
		ReplicationType:  req.ReplicationType,
		TargetNetwork:    req.TargetNetwork,
		ChangeID:         req.ChangeID,
		PreviousChangeID: req.PreviousChangeID,
		SnapshotID:       req.SnapshotID,
		// Pass scheduler metadata to Migration Engine
		ScheduleExecutionID: req.ScheduleExecutionID,
		VMGroupID:           req.VMGroupID,
		ScheduledBy:         req.ScheduledBy,
	}

	// If we have an existing context, pass it to the migration engine
	if existingContext != nil {
		migrationReq.ExistingContextID = existingContext.ContextID
		log.WithFields(log.Fields{
			"existing_context_id": existingContext.ContextID,
			"vm_name":             req.SourceVM.Name,
		}).Info("🔄 Using existing VM context for replication")
	}

	// Set defaults - check for previous successful migrations
	if migrationReq.ReplicationType == "" {
		// Check if there are previous successful migrations for this VM
		replicationType, previousChangeID, err := h.determineReplicationType(req.SourceVM.Path)
		if err != nil {
			log.WithError(err).Warn("Failed to determine replication type, defaulting to initial")
			migrationReq.ReplicationType = "initial"
		} else {
			migrationReq.ReplicationType = replicationType
			// Set previous change ID for incremental sync
			if replicationType == "incremental" && previousChangeID != "" {
				migrationReq.PreviousChangeID = previousChangeID
				log.WithFields(log.Fields{
					"vm_path":            req.SourceVM.Path,
					"previous_change_id": previousChangeID,
				}).Info("🔄 Setting up incremental sync with previous change ID")
			}
		}
	}
	// 🆕 ENHANCED: Use credential service for vCenter host
	ctx := context.Background()
	if migrationReq.VCenterHost == "" {
		// Get vCenter host from credential service
		if defaultHost := h.getDefaultVCenterHost(ctx); defaultHost != "" {
			migrationReq.VCenterHost = defaultHost
		} else {
			// Fallback to hardcoded during transition
			migrationReq.VCenterHost = "quad-vcenter-01.quadris.local"
		}
	}
	if migrationReq.Datacenter == "" {
		migrationReq.Datacenter = "DatabanxDC" // Default for current environment
	}

	if startReplication {
		// 🚫 TASK 7: Pre-flight CloudStack validation for INITIAL replications only
		// Incremental replications reuse existing volumes and don't need CloudStack resources
		if migrationReq.ReplicationType == "initial" {
			log.WithField("vm_name", req.SourceVM.Name).Info("🔍 Validating CloudStack prerequisites for initial replication")
			
			// 🆕 Get the detected config ID (may be auto-detected if invalid)
			detectedConfigID, err := h.validateAndGetConfigID(req.OSSEAConfigID)
			if err != nil {
				h.writeErrorResponse(w, http.StatusBadRequest,
					"Cannot start initial replication - CloudStack prerequisites not met",
					fmt.Sprintf("%s\n\nInitial replications require CloudStack resources (volumes will be provisioned). "+
						"Please complete CloudStack configuration in Settings page.", err.Error()))
				return
			}
			
			// 🆕 Update the migration request with the detected config ID
			if detectedConfigID != req.OSSEAConfigID {
				log.WithFields(log.Fields{
					"requested_config_id": req.OSSEAConfigID,
					"detected_config_id":  detectedConfigID,
				}).Info("🔄 Using auto-detected OSSEA config ID for migration")
				migrationReq.OSSEAConfigID = detectedConfigID
			}
			
			log.Info("✅ CloudStack prerequisites validated - proceeding with initial replication")
		} else {
			log.WithField("replication_type", migrationReq.ReplicationType).Info("⏩ Skipping CloudStack validation for incremental replication (reuses existing volumes)")
		}

		// Full replication workflow (existing behavior)
		log.WithFields(log.Fields{
			"job_id":           jobID,
			"source_vm":        req.SourceVM.Name,
			"source_vm_path":   req.SourceVM.Path,
			"ossea_config_id":  req.OSSEAConfigID,
			"replication_type": migrationReq.ReplicationType,
			"vcenter_host":     migrationReq.VCenterHost,
			"datacenter":       migrationReq.Datacenter,
		}).Info("🚀 Starting automated migration workflow")

		// Start the automated migration workflow
		result, err := h.migrationEngine.StartMigration(ctx, migrationReq)
		if err != nil {
			log.WithError(err).WithField("job_id", jobID).Error("Migration workflow failed")
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to start migration workflow", err.Error())
			return
		}

		// Convert to API response (existing job creation response)
		response := &MigrationStartResult{
			JobID:           result.JobID,
			Status:          result.Status,
			ProgressPercent: result.ProgressPercent,
			SourceVM:        req.SourceVM,
			CreatedVolumes:  make([]VolumeInfo, len(result.CreatedVolumes)),
			MountedVolumes:  make([]MountInfo, len(result.MountedVolumes)),
			StartedAt:       time.Now(),
			Message:         "Migration workflow started successfully - volumes created and mounted automatically",
		}

		// Convert volume information
		for i, vol := range result.CreatedVolumes {
			response.CreatedVolumes[i] = VolumeInfo{
				VolumeID:   vol.OSSEAVolumeID,
				VolumeName: vol.VolumeName,
				SizeGB:     vol.SizeGB,
				Status:     vol.Status,
			}
		}

		for i, mount := range result.MountedVolumes {
			response.MountedVolumes[i] = MountInfo{
				DevicePath: mount.DevicePath,
				MountPoint: mount.MountPoint,
				Status:     mount.Status,
			}
		}

		log.WithFields(log.Fields{
			"job_id":          result.JobID,
			"status":          result.Status,
			"progress":        result.ProgressPercent,
			"volumes_created": len(result.CreatedVolumes),
			"volumes_mounted": len(result.MountedVolumes),
		}).Info("✅ Migration workflow started - VMware replication initiated")

		h.writeJSONResponse(w, http.StatusCreated, response)
	} else {
		// Context-only creation (new "Add to Management" behavior)
		log.WithFields(log.Fields{
			"source_vm":      req.SourceVM.Name,
			"source_vm_path": req.SourceVM.Path,
			"vmware_vm_id":   req.SourceVM.ID,
		}).Info("📋 Adding VM to management without starting replication")

		// Create VM context without job
		contextResult, err := h.createVMContextOnly(ctx, req)
		if err != nil {
			log.WithError(err).WithField("vm_name", req.SourceVM.Name).Error("Failed to create VM context")
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to add VM to management", err.Error())
			return
		}

		log.WithFields(log.Fields{
			"context_id":     contextResult.ContextID,
			"vm_name":        contextResult.VMName,
			"current_status": contextResult.CurrentStatus,
		}).Info("✅ VM added to management successfully")

		h.writeJSONResponse(w, http.StatusCreated, contextResult)
	}
}

// GetByID handles migration job status lookup by ID
// @Summary Get migration job status
// @Description Get detailed real-time status of a migration job including volume and CBT information
// @Tags replications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Migration job ID"
// @Success 200 {object} workflows.MigrationStatusResult
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/replications/{id} [get]
func (h *ReplicationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	log.WithField("job_id", jobID).Debug("Getting migration job status")

	// Get real-time status from migration engine
	status, err := h.migrationEngine.GetMigrationStatus(jobID)
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Warn("Failed to get migration status")

		// Fallback to legacy in-memory storage for old jobs
		for _, job := range h.jobs {
			if job.ID == jobID {
				h.writeJSONResponse(w, http.StatusOK, job)
				return
			}
		}

		h.writeErrorResponse(w, http.StatusNotFound, "Job not found", "Migration job with ID "+jobID+" does not exist")
		return
	}

	log.WithFields(log.Fields{
		"job_id":       jobID,
		"status":       status.Status,
		"progress":     status.ProgressPercent,
		"disks_count":  len(status.Disks),
		"mounts_count": len(status.Mounts),
		"cbt_records":  len(status.CBTHistory),
	}).Debug("Retrieved migration status")

	h.writeJSONResponse(w, http.StatusOK, status)
}

// Update handles replication job updates
// @Summary Update replication job
// @Description Update replication job status, progress, or configuration
// @Tags replications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Replication job ID"
// @Param job body ReplicationJob true "Updated job data"
// @Success 200 {object} ReplicationJob
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/replications/{id} [put]
func (h *ReplicationHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	var updatedJob ReplicationJob
	if err := json.NewDecoder(r.Body).Decode(&updatedJob); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// Find and update job
	for i, job := range h.jobs {
		if job.ID == jobID {
			updatedJob.ID = jobID
			updatedJob.UpdatedAt = time.Now()
			h.jobs[i] = updatedJob

			log.WithField("job_id", jobID).Info("Updated replication job")
			h.writeJSONResponse(w, http.StatusOK, updatedJob)
			return
		}
	}

	h.writeErrorResponse(w, http.StatusNotFound, "Replication job not found", "")
}

// Delete handles replication job deletion
// @Summary Delete replication job
// @Description Cancel and delete a replication job
// @Tags replications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Replication job ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/replications/{id} [delete]
// Delete implements comprehensive job deletion with volume cleanup and database integrity
// DELETE /api/v1/replications/{id} - Safely removes replication jobs with all associated resources
func (h *ReplicationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]
	ctx := r.Context()

	log.WithField("job_id", jobID).Info("Starting comprehensive replication job deletion")

	// Step 1: Initialize JobLog tracking for deletion operation
	jobTracker, err := h.initializeJobTracker()
	if err != nil {
		log.WithError(err).Error("Failed to initialize JobLog tracker")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to initialize deletion tracking", err.Error())
		return
	}

	// Step 2: Create deletion tracking job
	ctx, deletionJobID, err := jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "cleanup",
		Operation: "delete-replication-job",
		Owner:     func() *string { s := "api"; return &s }(),
		Metadata: map[string]interface{}{
			"target_job_id": jobID,
			"user_agent":    r.Header.Get("User-Agent"),
		},
	})
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("Failed to create deletion tracking job")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create deletion tracking", err.Error())
		return
	}

	// Ensure job completion tracking
	defer func() {
		if err != nil {
			jobTracker.EndJob(ctx, deletionJobID, joblog.StatusFailed, err)
		} else {
			jobTracker.EndJob(ctx, deletionJobID, joblog.StatusCompleted, nil)
		}
	}()

	// Step 3: Get VM context information (VM-centric enhancement)
	vmContextID, err := h.getVMContextForJob(ctx, jobID)
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("Failed to get VM context for job")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get job context", err.Error())
		return
	}

	// Step 4: Validate job deletion is safe
	err = h.validateJobDeletion(ctx, jobTracker, deletionJobID, jobID)
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Warn("Job deletion validation failed")
		h.writeErrorResponse(w, http.StatusConflict, "Cannot delete job", err.Error())
		return
	}

	// Step 4: Delete volumes via Volume Daemon (continue on failure for orphaned volumes)
	volumeDeleteErr := h.deleteJobVolumes(ctx, jobTracker, deletionJobID, jobID)
	if volumeDeleteErr != nil {
		log.WithError(volumeDeleteErr).WithField("job_id", jobID).Warn("Volume deletion failed - continuing with database cleanup for orphaned job")
		// Continue with database cleanup even if volume deletion fails
		// This handles cases where volumes were already deleted manually or are corrupted
	}

	// Step 5: Clean up database records (always attempt this)
	err = h.deleteJobFromDatabase(ctx, jobTracker, deletionJobID, jobID)
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("Database cleanup failed")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Database cleanup failed", err.Error())
		return
	}

	// Step 6: Update VM context after job deletion (VM-centric enhancement)
	if vmContextID != "" {
		err = h.updateVMContextAfterJobDeletion(ctx, jobTracker, deletionJobID, vmContextID, jobID)
		if err != nil {
			log.WithError(err).WithField("vm_context_id", vmContextID).Warn("Failed to update VM context after job deletion - job deleted but context may be inconsistent")
			// Don't fail the request - job is already deleted
		}
	}

	// Step 7: Remove from legacy in-memory storage (backward compatibility)
	h.removeJobFromMemory(jobID)

	// Determine final status based on volume deletion result
	volumesDeleted := (volumeDeleteErr == nil)
	message := "Replication job deleted successfully"
	if volumeDeleteErr != nil {
		message = "Replication job deleted (volumes were already missing or corrupted)"
	}

	log.WithField("job_id", jobID).WithField("deletion_job_id", deletionJobID).WithField("volumes_deleted", volumesDeleted).Info("✅ Replication job deletion completed")

	response := map[string]interface{}{
		"message":              message,
		"job_id":               jobID,
		"deletion_tracking_id": deletionJobID,
		"timestamp":            time.Now().Format(time.RFC3339),
		"operations": map[string]interface{}{
			"volumes_deleted":     volumesDeleted,
			"database_cleaned":    true,
			"nbd_exports_removed": volumesDeleted, // NBD exports removed if volumes were successfully processed
		},
	}

	// Add volume deletion error details if relevant
	if volumeDeleteErr != nil {
		response["volume_deletion_warning"] = volumeDeleteErr.Error()
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// Helper functions

// writeJSONResponse writes a standardized JSON response
func (h *ReplicationHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeErrorResponse writes a standardized error response
func (h *ReplicationHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	response := map[string]interface{}{
		"error":     message,
		"details":   details,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}

// determineReplicationType checks for previous successful migrations to determine replication type
// Returns replication type and previous change ID (if available)
func (h *ReplicationHandler) determineReplicationType(vmPath string) (string, string, error) {
	// Query vm_disks table for previous successful ChangeID storage for this VM path
	// Join with replication_jobs to get VM path information
	var vmDisks []database.VMDisk
	query := `
		SELECT vm_disks.* FROM vm_disks 
		JOIN replication_jobs ON vm_disks.job_id = replication_jobs.id 
		WHERE replication_jobs.source_vm_path = ? 
		AND vm_disks.disk_change_id IS NOT NULL 
		AND vm_disks.disk_change_id != ''
		ORDER BY vm_disks.updated_at DESC
	`

	if err := h.db.GetGormDB().Raw(query, vmPath).Find(&vmDisks).Error; err != nil {
		return "initial", "", fmt.Errorf("failed to check previous migrations: %w", err)
	}

	// If we found any disks with stored ChangeIDs, use incremental
	if len(vmDisks) > 0 {
		latestDisk := vmDisks[0]

		log.WithFields(log.Fields{
			"vm_path":          vmPath,
			"previous_disks":   len(vmDisks),
			"latest_job":       latestDisk.JobID,
			"latest_change_id": latestDisk.DiskChangeID,
			"latest_updated":   latestDisk.UpdatedAt,
		}).Info("Found previous successful migration with ChangeID, using incremental sync")

		// Return incremental type with the change ID from the most recent successful migration
		return "incremental", latestDisk.DiskChangeID, nil
	}

	log.WithFields(log.Fields{
		"vm_path": vmPath,
	}).Info("No previous successful migration with ChangeID found, using initial sync")

	return "initial", "", nil
}

// getDiskSpecificChangeID retrieves change ID for a specific disk of a VM
func (h *ReplicationHandler) getDiskSpecificChangeID(vmPath, diskID string) (string, error) {
	var vmDisk database.VMDisk
	query := `
		SELECT vm_disks.* FROM vm_disks 
		JOIN replication_jobs ON vm_disks.job_id = replication_jobs.id 
		WHERE replication_jobs.source_vm_path = ? 
		AND vm_disks.disk_id = ?
		AND vm_disks.disk_change_id IS NOT NULL 
		AND vm_disks.disk_change_id != ''
		ORDER BY vm_disks.updated_at DESC
		LIMIT 1
	`

	if err := h.db.GetGormDB().Raw(query, vmPath, diskID).First(&vmDisk).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.WithFields(log.Fields{
				"vm_path": vmPath,
				"disk_id": diskID,
			}).Info("No previous change ID found for specific disk")
			return "", nil // No change ID found - not an error
		}
		return "", fmt.Errorf("failed to check previous disk change ID: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_path":    vmPath,
		"disk_id":    diskID,
		"change_id":  vmDisk.DiskChangeID,
		"job_id":     vmDisk.JobID,
		"updated_at": vmDisk.UpdatedAt,
	}).Info("Found previous change ID for specific disk")

	return vmDisk.DiskChangeID, nil
}

// GetPreviousChangeID retrieves the last successful change ID for a VM path
// @Summary Get previous change ID for VM
// @Description Get the change ID from the last successful migration for incremental sync
// @Tags replications
// @Security BearerAuth
// @Produce json
// @Param vm_path query string true "VM path (e.g., /DatabanxDC/vm/PGWINTESTBIOS)"
// @Param disk_id query string false "Disk ID for multi-disk VMs (e.g., disk-2000)"
// @Success 200 {object} map[string]string
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/replications/changeid [get]
func (h *ReplicationHandler) GetPreviousChangeID(w http.ResponseWriter, r *http.Request) {
	vmPath := r.URL.Query().Get("vm_path")
	diskID := r.URL.Query().Get("disk_id") // NEW: disk-specific query

	if vmPath == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing vm_path parameter", "vm_path query parameter is required")
		return
	}

	var changeID string
	var err error

	if diskID != "" {
		// NEW: Disk-specific change ID lookup for multi-disk VMs
		changeID, err = h.getDiskSpecificChangeID(vmPath, diskID)
		log.WithFields(log.Fields{
			"vm_path": vmPath,
			"disk_id": diskID,
		}).Info("Using disk-specific change ID lookup for multi-disk VM")
	} else {
		// BACKWARD COMPATIBILITY: Use existing logic for single-disk VMs
		_, changeID, err = h.determineReplicationType(vmPath)
		log.WithField("vm_path", vmPath).Info("Using legacy change ID lookup (backward compatibility)")
	}

	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"vm_path": vmPath,
			"disk_id": diskID,
		}).Error("Failed to get previous change ID")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get change ID", err.Error())
		return
	}

	response := map[string]string{
		"vm_path":   vmPath,
		"change_id": changeID,
	}

	if diskID != "" {
		response["disk_id"] = diskID
	}

	if changeID == "" {
		response["message"] = "No previous successful migration found"
	} else {
		response["message"] = "Previous change ID found"
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// StoreChangeID stores a ChangeID for a specific job and disk in the database
// @Summary Store change ID for migration job
// @Description Store the VMware ChangeID after migration completion for future incremental sync
// @Tags replications
// @Accept json
// @Produce json
// @Param job_id path string true "Migration job ID"
// @Param request body StoreChangeIDRequest true "ChangeID data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/replications/{job_id}/changeid [post]
func (h *ReplicationHandler) StoreChangeID(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from URL path
	jobID := mux.Vars(r)["job_id"]
	if jobID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing job_id parameter", "job_id path parameter is required")
		return
	}

	// Parse request body
	var req StoreChangeIDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate required fields
	if req.ChangeID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing change_id", "change_id field is required")
		return
	}

	if req.DiskID == "" {
		req.DiskID = "disk-2000" // Default disk ID if not provided
	}

	log.WithFields(log.Fields{
		"job_id":    jobID,
		"disk_id":   req.DiskID,
		"change_id": req.ChangeID,
	}).Info("Storing ChangeID for migration job")

	// 🎯 CRITICAL FIX: Use VM-centric lookup for stable vm_disks compatibility
	// First, find the VM context from the job ID
	var replicationJob database.ReplicationJob
	err := h.db.GetGormDB().Where("id = ?", jobID).First(&replicationJob).Error
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("Failed to find replication job")
		h.writeErrorResponse(w, http.StatusNotFound, "Job not found", "Replication job not found")
		return
	}

	// Use VM context to find current vm_disks records (stable vm_disks approach)
	var vmDisks []database.VMDisk
	err = h.db.GetGormDB().Where("vm_context_id = ?", replicationJob.VMContextID).Find(&vmDisks).Error
	if err != nil {
		log.WithError(err).WithField("vm_context_id", replicationJob.VMContextID).Error("Failed to get VM disks for context")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Database error", "Failed to retrieve VM disk records")
		return
	}

	if len(vmDisks) == 0 {
		h.writeErrorResponse(w, http.StatusNotFound, "VM disks not found", "No VM disks found for VM context")
		return
	}

	// Find the specific disk by disk_id (required for multi-disk support)
	var targetDisk *database.VMDisk
	for _, disk := range vmDisks {
		if disk.DiskID == req.DiskID {
			targetDisk = &disk
			log.WithFields(log.Fields{
				"job_id":        jobID,
				"vm_context_id": replicationJob.VMContextID,
				"disk_id":       req.DiskID,
				"vm_disk_id":    disk.ID,
			}).Info("✅ Found target disk for change ID storage using VM-centric lookup")
			break
		}
	}

	// If specific disk not found, use first disk
	if targetDisk == nil {
		if len(vmDisks) > 0 {
			targetDisk = &vmDisks[0]
			log.WithFields(log.Fields{
				"job_id":         jobID,
				"requested_disk": req.DiskID,
				"using_disk":     targetDisk.DiskID,
			}).Info("Disk ID not found, using first available disk")
		} else {
			h.writeErrorResponse(w, http.StatusNotFound, "Disk not found", "No disks available for this job")
			return
		}
	}

	// Update the disk's ChangeID
	err = h.migrationEngine.UpdateVMDiskChangeID(targetDisk.ID, req.ChangeID)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"job_id":    jobID,
			"disk_id":   req.DiskID,
			"change_id": req.ChangeID,
		}).Error("Failed to update ChangeID in database")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Database error", "Failed to store ChangeID")
		return
	}

	// Also create a CBT history record for tracking
	err = h.migrationEngine.StoreCBTHistory(jobID, req.DiskID, req.ChangeID, req.PreviousChangeID, "completed", true)
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Warn("Failed to create CBT history record")
		// Don't fail the request for history record failure
	}

	log.WithFields(log.Fields{
		"job_id":    jobID,
		"disk_id":   req.DiskID,
		"change_id": req.ChangeID,
	}).Info("✅ Successfully stored ChangeID in database")

	// Return success response
	response := map[string]string{
		"job_id":    jobID,
		"disk_id":   req.DiskID,
		"change_id": req.ChangeID,
		"message":   "ChangeID stored successfully",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetReplicationProgress handles real-time progress requests with SNA integration
// @Summary Get real-time replication progress
// @Description Get detailed replication progress including SNA throughput and phase tracking
// @Tags replications
// @Security BearerAuth
// @Produce json
// @Param job_id path string true "Job ID"
// @Success 200 {object} ReplicationProgressResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/replications/{job_id}/progress [get]
func (h *ReplicationHandler) GetReplicationProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]

	log.WithField("job_id", jobID).Debug("Getting replication progress with SNA integration")

	// Get job from database
	repo := database.NewOSSEAConfigRepository(h.db)
	job, err := repo.GetReplicationJob(jobID)
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("Failed to get replication job")
		h.writeErrorResponse(w, http.StatusNotFound, "Job not found", err.Error())
		return
	}

	// Build enhanced progress response
	progressResp := ReplicationProgressResponse{
		JobID:            job.ID,
		Status:           job.Status,
		ProgressPercent:  job.ProgressPercent,
		CurrentOperation: job.CurrentOperation,
		BytesTransferred: job.BytesTransferred,
		TotalBytes:       job.TotalBytes,
		TransferSpeedBps: job.TransferSpeedBps,
		ErrorMessage:     job.ErrorMessage,
		CreatedAt:        job.CreatedAt,
		UpdatedAt:        job.UpdatedAt,
		StartedAt:        job.StartedAt,
		CompletedAt:      job.CompletedAt,

		// SNA Progress Integration
		SNAProgress: SNAProgressInfo{
			SyncType:            job.SNASyncType,
			CurrentPhase:        job.SNACurrentPhase,
			ThroughputMBps:      job.SNAThroughputMBps,
			ETASeconds:          job.SNAETASeconds,
			LastPolled:          job.SNALastPollAt,
			ErrorClassification: job.SNAErrorClassification,
			ErrorDetails:        job.SNAErrorDetails,
		},
	}

	// Calculate ETA string
	if job.SNAETASeconds != nil && *job.SNAETASeconds > 0 {
		eta := time.Duration(*job.SNAETASeconds) * time.Second
		progressResp.SNAProgress.ETAFormatted = eta.String()
	}

	// Add real-time status
	progressResp.SNAProgress.IsLive = job.SNALastPollAt != nil &&
		time.Since(*job.SNALastPollAt) < 30*time.Second

	log.WithFields(log.Fields{
		"job_id":           jobID,
		"progress_percent": job.ProgressPercent,
		"vma_phase":        job.SNACurrentPhase,
		"vma_throughput":   job.SNAThroughputMBps,
		"vma_last_poll":    job.SNALastPollAt,
	}).Debug("✅ Successfully retrieved replication progress with SNA data")

	h.writeJSONResponse(w, http.StatusOK, progressResp)
}

// ReplicationProgressResponse represents enhanced progress response with SNA integration
type ReplicationProgressResponse struct {
	JobID            string     `json:"job_id"`
	Status           string     `json:"status"`
	ProgressPercent  float64    `json:"progress_percent"`
	CurrentOperation string     `json:"current_operation"`
	BytesTransferred int64      `json:"bytes_transferred"`
	TotalBytes       int64      `json:"total_bytes"`
	TransferSpeedBps int64      `json:"transfer_speed_bps"`
	ErrorMessage     string     `json:"error_message"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	StartedAt        *time.Time `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`

	// SNA Progress Integration
	SNAProgress SNAProgressInfo `json:"vma_progress"`
}

// SNAProgressInfo represents SNA-specific progress information
type SNAProgressInfo struct {
	SyncType            string     `json:"sync_type"`
	CurrentPhase        string     `json:"current_phase"`
	ThroughputMBps      float64    `json:"throughput_mbps"`
	ETASeconds          *int       `json:"eta_seconds,omitempty"`
	ETAFormatted        string     `json:"eta_formatted,omitempty"`
	LastPolled          *time.Time `json:"last_polled,omitempty"`
	ErrorClassification string     `json:"error_classification"`
	ErrorDetails        string     `json:"error_details"`
	IsLive              bool       `json:"is_live"` // True if data is fresh (< 30s old)
}

// GetVMAProgressProxy proxies SNA progress requests through the tunnel
// @Summary Get SNA progress via tunnel
// @Description Proxy SNA progress API through SHA tunnel following project rules (port 443 only)
// @Tags progress
// @Security BearerAuth
// @Produce json
// @Param job_id path string true "Job ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/progress/{job_id} [get]
func (h *ReplicationHandler) GetVMAProgressProxy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]

	log.WithField("job_id", jobID).Debug("Proxying SNA progress request through tunnel")

	// Create HTTP client for SNA API call (following project rules: all traffic via tunnel)
	snaURL := fmt.Sprintf("http://localhost:9081/api/v1/progress/%s", jobID)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request to SNA API
	resp, err := client.Get(snaURL)
	if err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("Failed to connect to SNA progress API")
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "SNA API unavailable", err.Error())
		return
	}
	defer resp.Body.Close()

	// Handle SNA API response codes
	if resp.StatusCode == http.StatusNotFound {
		log.WithField("job_id", jobID).Debug("Job not found in SNA API")
		h.writeErrorResponse(w, http.StatusNotFound, "Job not found", "Migration job not found in SNA")
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"job_id":      jobID,
			"status_code": resp.StatusCode,
		}).Warn("SNA API returned non-200 status")
		h.writeErrorResponse(w, resp.StatusCode, "SNA API error", fmt.Sprintf("SNA returned status %d", resp.StatusCode))
		return
	}

	// Read and forward SNA response
	body := make([]byte, 0, 4096)
	buffer := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			body = append(body, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Set content type and forward response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)

	log.WithFields(log.Fields{
		"job_id":        jobID,
		"response_size": len(body),
	}).Debug("✅ Successfully proxied SNA progress response")
}

// =============================================================================
// Job Deletion Helper Methods
// =============================================================================

// initializeJobTracker creates a JobLog tracker for deletion operations
func (h *ReplicationHandler) initializeJobTracker() (*joblog.Tracker, error) {
	// Get sql.DB from GORM for JobLog integration
	sqlDB, err := h.db.GetGormDB().DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from GORM for joblog: %w", err)
	}

	// Create JobLog handlers following project patterns
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	dbHandler := joblog.NewDBHandler(sqlDB, joblog.DefaultDBHandlerConfig())

	return joblog.New(sqlDB, stdoutHandler, dbHandler), nil
}

// validateJobDeletion checks if a job can be safely deleted
func (h *ReplicationHandler) validateJobDeletion(ctx context.Context, tracker *joblog.Tracker, deletionJobID, jobID string) error {
	return tracker.RunStep(ctx, deletionJobID, "validate-deletion", func(ctx context.Context) error {
		logger := tracker.Logger(ctx)
		logger.Info("Validating job deletion safety", "job_id", jobID)

		// Check if job exists in database
		job, err := h.getReplicationJob(ctx, jobID)
		if err != nil {
			return fmt.Errorf("job not found: %w", err)
		}

		// Prevent deletion of running jobs
		if job.Status == "running" || job.Status == "replicating" {
			return fmt.Errorf("cannot delete job in status: %s", job.Status)
		}

		// Check for active failover operations
		activeFailover, err := h.checkActiveFailover(ctx, jobID)
		if err != nil {
			return fmt.Errorf("failed to check failover status: %w", err)
		}

		if activeFailover {
			return fmt.Errorf("cannot delete job with active failover operations")
		}

		logger.Info("✅ Job deletion validation passed", "job_id", jobID)
		return nil
	})
}

// deleteJobVolumes removes all volumes associated with a job via Volume Daemon
func (h *ReplicationHandler) deleteJobVolumes(ctx context.Context, tracker *joblog.Tracker, deletionJobID, jobID string) error {
	return tracker.RunStep(ctx, deletionJobID, "delete-volumes", func(ctx context.Context) error {
		logger := tracker.Logger(ctx)
		logger.Info("Starting volume deletion via Volume Daemon", "job_id", jobID)

		// Get all volumes for this job
		volumes, err := h.getJobVolumes(ctx, jobID)
		if err != nil {
			return fmt.Errorf("failed to get job volumes: %w", err)
		}

		if len(volumes) == 0 {
			logger.Info("No volumes found for job - skipping volume deletion", "job_id", jobID)
			return nil
		}

		logger.Info("Found volumes to delete", "job_id", jobID, "volume_count", len(volumes))

		// Initialize Volume Daemon client
		volumeClient := common.NewVolumeClient("http://localhost:8090")

		// Delete each volume via Volume Daemon
		for _, volumeID := range volumes {
			logger.Info("Deleting volume via Volume Daemon", "volume_id", volumeID)

			// Delete volume (NBD exports are handled automatically by Volume Daemon)
			operation, err := volumeClient.DeleteVolume(ctx, volumeID)
			if err != nil {
				logger.Error("Volume deletion failed", "volume_id", volumeID, "error", err)
				return fmt.Errorf("failed to delete volume %s: %w", volumeID, err)
			}

			// Wait for completion with timeout
			_, err = volumeClient.WaitForCompletionWithTimeout(ctx, operation.ID, 300*time.Second)
			if err != nil {
				return fmt.Errorf("volume deletion operation failed for %s: %w", volumeID, err)
			}

			logger.Info("✅ Volume deleted successfully", "volume_id", volumeID)
		}

		logger.Info("✅ All job volumes deleted successfully", "job_id", jobID, "deleted_count", len(volumes))
		return nil
	})
}

// deleteJobFromDatabase removes job records from database using CASCADE DELETE
func (h *ReplicationHandler) deleteJobFromDatabase(ctx context.Context, tracker *joblog.Tracker, deletionJobID, jobID string) error {
	return tracker.RunStep(ctx, deletionJobID, "database-cleanup", func(ctx context.Context) error {
		logger := tracker.Logger(ctx)
		logger.Info("Starting database cleanup", "job_id", jobID)

		// Get GORM DB for transaction handling
		gormDB := h.db.GetGormDB()

		// Start database transaction for atomic operations
		tx := gormDB.Begin()
		if tx.Error != nil {
			return fmt.Errorf("failed to start database transaction: %w", tx.Error)
		}
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Manual cleanup for SET NULL relationships (failover_jobs)
		result := tx.Exec("UPDATE failover_jobs SET replication_job_id = NULL WHERE replication_job_id = ?", jobID)
		if result.Error != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update failover jobs: %w", result.Error)
		}

		if result.RowsAffected > 0 {
			logger.Info("Updated failover jobs to NULL reference", "job_id", jobID, "failover_jobs_updated", result.RowsAffected)
		}

		// Delete main job record (triggers CASCADE DELETE for dependent records)
		// This will automatically delete: vm_disks, volume_mounts, cbt_history
		result = tx.Exec("DELETE FROM replication_jobs WHERE id = ?", jobID)
		if result.Error != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete replication job: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			tx.Rollback()
			return fmt.Errorf("job not found in database")
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit database transaction: %w", err)
		}

		logger.Info("✅ Job and related records deleted from database", "job_id", jobID)
		return nil
	})
}

// removeJobFromMemory removes job from legacy in-memory storage (backward compatibility)
func (h *ReplicationHandler) removeJobFromMemory(jobID string) {
	for i, job := range h.jobs {
		if job.ID == jobID {
			h.jobs = append(h.jobs[:i], h.jobs[i+1:]...)
			log.WithField("job_id", jobID).Debug("Removed job from legacy in-memory storage")
			return
		}
	}
}

// =============================================================================
// Database Query Helper Methods
// =============================================================================

// getReplicationJob retrieves a replication job from database via repository
func (h *ReplicationHandler) getReplicationJob(ctx context.Context, jobID string) (*database.ReplicationJob, error) {
	return h.replicationRepo.GetByID(ctx, jobID)
}

// getJobVolumes retrieves all volume IDs associated with a job via repository
func (h *ReplicationHandler) getJobVolumes(ctx context.Context, jobID string) ([]string, error) {
	return h.replicationRepo.GetJobVolumes(ctx, jobID)
}

// checkActiveFailover checks if job has any active failover operations via repository
func (h *ReplicationHandler) checkActiveFailover(ctx context.Context, jobID string) (bool, error) {
	return h.replicationRepo.CheckActiveFailover(ctx, jobID)
}

// VM-Centric Helper Methods

// getVMContextForJob retrieves the VM context ID for a given replication job
func (h *ReplicationHandler) getVMContextForJob(ctx context.Context, jobID string) (string, error) {
	var vmContextID string
	err := h.db.GetGormDB().Raw(
		"SELECT COALESCE(vm_context_id, '') FROM replication_jobs WHERE id = ?",
		jobID).Scan(&vmContextID).Error

	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("job not found: %s", jobID)
		}
		return "", fmt.Errorf("failed to get VM context for job %s: %w", jobID, err)
	}

	return vmContextID, nil
}

// updateVMContextAfterJobDeletion updates VM context statistics and status after a job is deleted
func (h *ReplicationHandler) updateVMContextAfterJobDeletion(ctx context.Context, jobTracker *joblog.Tracker, deletionJobID string, vmContextID, deletedJobID string) error {
	if vmContextID == "" {
		// No context to update (legacy job or context not set)
		return nil
	}

	return jobTracker.RunStep(ctx, deletionJobID, "update-vm-context", func(ctx context.Context) error {
		log := jobTracker.Logger(ctx)

		// Start transaction for atomic context update
		tx := h.db.GetGormDB().Begin()
		if tx.Error != nil {
			return fmt.Errorf("failed to start transaction: %w", tx.Error)
		}
		defer tx.Rollback()

		// Get current context state using simpler approach
		type ContextState struct {
			CurrentJobID        *string `gorm:"column:current_job_id"`
			LastSuccessfulJobID *string `gorm:"column:last_successful_job_id"`
			TotalJobsRun        int     `gorm:"column:total_jobs_run"`
			SuccessfulJobs      int     `gorm:"column:successful_jobs"`
			FailedJobs          int     `gorm:"column:failed_jobs"`
			CurrentStatus       string  `gorm:"column:current_status"`
		}

		var contextState ContextState
		err := tx.Table("vm_replication_contexts").
			Where("context_id = ?", vmContextID).
			First(&contextState).Error

		if err != nil {
			return fmt.Errorf("failed to get current context state: %w", err)
		}

		// Update statistics - assume deleted job was successful if it was completed
		// (Only failed or stuck jobs typically need manual deletion)
		var jobWasSuccessful bool
		err = tx.Raw(
			"SELECT status = 'completed' FROM replication_jobs WHERE id = ?",
			deletedJobID).Scan(&jobWasSuccessful).Error

		// If job doesn't exist anymore (already CASCADE deleted), assume it was problematic
		if err != nil {
			// Job likely already deleted via CASCADE - assume it was problematic
			jobWasSuccessful = false
		}

		// Calculate new statistics
		newTotalJobs := contextState.TotalJobsRun - 1
		newSuccessfulJobs := contextState.SuccessfulJobs
		newFailedJobs := contextState.FailedJobs

		if jobWasSuccessful {
			newSuccessfulJobs = max(0, contextState.SuccessfulJobs-1)
		} else {
			newFailedJobs = max(0, contextState.FailedJobs-1)
		}

		// Determine new status based on remaining jobs
		var remainingJobs int64
		err = tx.Table("replication_jobs").
			Where("vm_context_id = ?", vmContextID).
			Count(&remainingJobs).Error

		if err != nil {
			return fmt.Errorf("failed to count remaining jobs: %w", err)
		}

		newStatus := "discovered" // Default if no jobs remain
		var newCurrentJobID *string = nil

		if remainingJobs > 0 {
			// Find the most recent job
			type LatestJob struct {
				ID     string `gorm:"column:id"`
				Status string `gorm:"column:status"`
			}

			var latestJob LatestJob
			err = tx.Table("replication_jobs").
				Where("vm_context_id = ?", vmContextID).
				Order("created_at DESC").
				First(&latestJob).Error

			if err != nil {
				return fmt.Errorf("failed to get latest job: %w", err)
			}

			newCurrentJobID = &latestJob.ID

			// Map job status to context status
			switch latestJob.Status {
			case "replicating", "provisioning":
				newStatus = "replicating"
			case "completed":
				newStatus = "ready_for_failover"
			case "failed":
				newStatus = "failed"
			default:
				newStatus = "discovered"
			}
		}

		// Clear last_successful_job_id if it was the deleted job
		newLastSuccessfulJobID := contextState.LastSuccessfulJobID
		if contextState.LastSuccessfulJobID != nil && *contextState.LastSuccessfulJobID == deletedJobID {
			// Find the next most recent successful job
			type SuccessfulJob struct {
				ID string `gorm:"column:id"`
			}

			var successfulJob SuccessfulJob
			err = tx.Table("replication_jobs").
				Where("vm_context_id = ? AND status = 'completed'", vmContextID).
				Order("completed_at DESC").
				First(&successfulJob).Error

			if err != nil {
				// No other successful jobs found
				newLastSuccessfulJobID = nil
			} else {
				newLastSuccessfulJobID = &successfulJob.ID
			}
		}

		// Update the VM context
		err = tx.Table("vm_replication_contexts").
			Where("context_id = ?", vmContextID).
			Updates(map[string]interface{}{
				"current_job_id":         newCurrentJobID,
				"last_successful_job_id": newLastSuccessfulJobID,
				"total_jobs_run":         newTotalJobs,
				"successful_jobs":        newSuccessfulJobs,
				"failed_jobs":            newFailedJobs,
				"current_status":         newStatus,
				"last_status_change":     "CURRENT_TIMESTAMP",
				"updated_at":             "CURRENT_TIMESTAMP",
			}).Error

		if err != nil {
			return fmt.Errorf("failed to update VM context: %w", err)
		}

		err = tx.Commit().Error
		if err != nil {
			return fmt.Errorf("failed to commit context update: %w", err)
		}

		log.Info("VM context updated after job deletion", map[string]interface{}{
			"vm_context_id":   vmContextID,
			"deleted_job_id":  deletedJobID,
			"new_status":      newStatus,
			"remaining_jobs":  remainingJobs,
			"total_jobs":      newTotalJobs,
			"successful_jobs": newSuccessfulJobs,
			"failed_jobs":     newFailedJobs,
		})

		return nil
	})
}

// getDefaultVCenterHost retrieves the default vCenter host from credential service
func (h *ReplicationHandler) getDefaultVCenterHost(ctx context.Context) string {
	encryptionService, err := services.NewCredentialEncryptionService()
	if err != nil {
		log.WithError(err).Debug("Failed to initialize encryption service for vCenter host lookup")
		return ""
	}

	credentialService := services.NewVMwareCredentialService(&h.db, encryptionService)
	creds, err := credentialService.GetDefaultCredentials(ctx)
	if err != nil {
		log.WithError(err).Debug("Failed to get default credentials for vCenter host lookup")
		return ""
	}

	log.WithField("vcenter_host", creds.VCenterHost).Debug("✅ Retrieved vCenter host from credential service")
	return creds.VCenterHost
}

// Helper function since max() is not available in older Go versions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// validateAndGetConfigID validates CloudStack config and returns the valid config ID (may auto-detect)
// Returns the config ID to use (original or auto-detected) and error if validation fails
// 🚫 TASK 7: Replication Blocker Logic
func (h *ReplicationHandler) validateAndGetConfigID(osseaConfigID int) (int, error) {
	// 🆕 Auto-detect active config if not provided (or if ID doesn't exist)
	if osseaConfigID == 0 || osseaConfigID == 1 {
		log.WithField("requested_config_id", osseaConfigID).Debug("No valid config ID provided, attempting auto-detection")
		activeConfigID := h.getActiveOSSEAConfigID()
		if activeConfigID > 0 {
			log.WithFields(log.Fields{
				"old_config_id": osseaConfigID,
				"new_config_id": activeConfigID,
			}).Info("🔄 Auto-detected active OSSEA config for validation")
			osseaConfigID = activeConfigID
		}
	}
	
	// Now validate the config
	err := h.validateCloudStackForProvisioning(osseaConfigID)
	if err != nil {
		return 0, err
	}
	
	return osseaConfigID, nil
}

// validateCloudStackForProvisioning validates CloudStack prerequisites for volume provisioning
// This is called ONLY for INITIAL replications (not incremental)
// 🚫 TASK 7: Replication Blocker Logic
func (h *ReplicationHandler) validateCloudStackForProvisioning(osseaConfigID int) error {
	
	// Get OSSEA configuration
	repo := database.NewOSSEAConfigRepository(h.db)
	
	// Initialize encryption service if available
	encryptionService, err := services.NewCredentialEncryptionService()
	if err != nil {
		log.WithError(err).Debug("Encryption service not available for CloudStack validation")
	} else {
		repo.SetEncryptionService(encryptionService)
	}
	
	config, err := repo.GetByID(osseaConfigID)
	if err != nil {
		return fmt.Errorf("no CloudStack configuration found (ID: %d)", osseaConfigID)
	}

	// HARD BLOCKS - Required for volume provisioning phase
	var errors []string

	if config.SHAVMID == "" {
		errors = append(errors, "❌ SHA VM ID not configured - volume attachment will fail")
	}

	if config.DiskOfferingID == "" {
		errors = append(errors, "❌ Disk Offering not selected - volume creation will fail")
	}

	if config.Zone == "" {
		errors = append(errors, "❌ CloudStack Zone not configured - volume creation will fail")
	}

	// Return hard block errors if any
	if len(errors) > 0 {
		return fmt.Errorf("Missing required CloudStack prerequisites:\n%s", strings.Join(errors, "\n"))
	}

	// WARNINGS - Needed for failover later, but not for replication
	if config.NetworkID == "" {
		log.Warn("⚠️  Network not configured - failover will not be possible until network is selected")
	}

	if config.ServiceOfferingID == "" {
		log.Warn("⚠️  Compute offering not configured - failover will not be possible until offering is selected")
	}

	log.WithFields(log.Fields{
		"oma_vm_id":           config.SHAVMID,
		"disk_offering_id":    config.DiskOfferingID,
		"zone":                config.Zone,
		"network_id":          config.NetworkID,
		"service_offering_id": config.ServiceOfferingID,
	}).Debug("✅ CloudStack prerequisites validated for volume provisioning")

	return nil
}

// 🆕 getActiveOSSEAConfigID retrieves the active OSSEA configuration ID (where is_active = 1)
// Returns 0 if no active config found
func (h *ReplicationHandler) getActiveOSSEAConfigID() int {
	// Query database for active OSSEA config
	var configID int
	
	var result struct {
		ID int
	}
	err := h.db.GetGormDB().Raw(
		"SELECT id FROM ossea_configs WHERE is_active = 1 ORDER BY id DESC LIMIT 1").Scan(&result).Error
	configID = result.ID
	
	if err != nil {
		log.WithError(err).Debug("No active OSSEA config found during auto-detection")
		return 0
	}
	
	log.WithField("config_id", configID).Info("✅ Found active OSSEA config via auto-detection")
	return configID
}
