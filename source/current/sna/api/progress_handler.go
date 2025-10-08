package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit/source/current/sna/progress"
	"github.com/vexxhost/migratekit/source/current/sna/services"
)

// ProgressUpdate represents a progress update from migratekit
type ProgressUpdate struct {
	Stage            progress.ReplicationStage  `json:"stage"`
	Status           progress.ReplicationStatus `json:"status,omitempty"`
	BytesTransferred int64                      `json:"bytes_transferred"`
	TotalBytes       int64                      `json:"total_bytes,omitempty"`
	ThroughputBPS    int64                      `json:"throughput_bps"`
	Percent          float64                    `json:"percent,omitempty"`
	DiskID           string                     `json:"disk_id,omitempty"`
	SyncType         string                     `json:"sync_type,omitempty"` // 🎯 FIX: Sync type from migratekit
}

// SNAProgressResponse represents the response format expected by SHA Progress Poller
type SNAProgressResponse struct {
	JobID            string  `json:"job_id"`
	Status           string  `json:"status"`
	SyncType         string  `json:"sync_type"`
	Phase            string  `json:"phase"`
	Percentage       float64 `json:"percentage"`
	CurrentOperation string  `json:"current_operation"`
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes"`

	// Throughput data
	Throughput struct {
		CurrentMBps float64 `json:"current_mbps"`
		AverageMBps float64 `json:"average_mbps"`
		PeakMBps    float64 `json:"peak_mbps"`
		LastUpdate  string  `json:"last_update"`
	} `json:"throughput"`

	// Timing information
	Timing struct {
		StartTime      string `json:"start_time"`
		LastUpdate     string `json:"last_update"`
		ElapsedMs      int64  `json:"elapsed_ms"`
		PhaseStart     string `json:"phase_start"`
		PhaseElapsedMs int64  `json:"phase_elapsed_ms"`
		ETASeconds     int    `json:"eta_seconds"`
	} `json:"timing"`

	// VM Information
	VMInfo struct {
		Name          string `json:"name"`
		Path          string `json:"path"`
		DiskSizeGB    int64  `json:"disk_size_gb"`
		DiskSizeBytes int64  `json:"disk_size_bytes"`
		CBTEnabled    bool   `json:"cbt_enabled"`
	} `json:"vm_info"`

	// Phase progression
	Phases []struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		StartTime  string `json:"start_time"`
		EndTime    string `json:"end_time"`
		DurationMs int64  `json:"duration_ms"`
	} `json:"phases"`

	// Error information
	Errors    []interface{} `json:"errors"`
	LastError interface{}   `json:"last_error"`

	// 🎯 MULTI-DISK PROGRESS INFORMATION
	DiskProgresses []DiskProgressInfo `json:"disk_progresses,omitempty"`
}

// DiskProgressInfo represents individual disk progress for API response
type DiskProgressInfo struct {
	DiskID           string  `json:"disk_id"`
	Label            string  `json:"label"`
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes"`
	Percent          float64 `json:"percent"`
	ThroughputMBps   float64 `json:"throughput_mbps"`
	Status           string  `json:"status"`
}

// ProgressHandler handles HTTP requests for replication progress tracking
type ProgressHandler struct {
	progressService *services.ProgressService
}

// NewProgressHandler creates a new progress handler
func NewProgressHandler(progressService *services.ProgressService) *ProgressHandler {
	return &ProgressHandler{
		progressService: progressService,
	}
}

// convertToVMAProgressResponse converts internal ReplicationProgress to SNAProgressResponse format
func convertToVMAProgressResponse(jobProgress *progress.ReplicationProgress) *SNAProgressResponse {
	response := &SNAProgressResponse{
		JobID:            jobProgress.JobID,
		Status:           string(jobProgress.Status),
		SyncType:         string(jobProgress.CBT.Type), // 🚨 FIX: Use actual CBT type instead of hardcoded "full"
		Phase:            string(jobProgress.Stage),
		Percentage:       jobProgress.Aggregate.Percent,
		CurrentOperation: mapStageToOperation(jobProgress.Stage, jobProgress.Status),
		BytesTransferred: jobProgress.Aggregate.BytesTransferred,
		TotalBytes:       jobProgress.Aggregate.TotalBytes,
		Errors:           []interface{}{},
		LastError:        nil,
	}

	// Timing information
	response.Timing.StartTime = jobProgress.StartedAt.Format(time.RFC3339)
	response.Timing.LastUpdate = jobProgress.UpdatedAt.Format(time.RFC3339)
	response.Timing.ElapsedMs = int64(jobProgress.UpdatedAt.Sub(jobProgress.StartedAt).Milliseconds())
	response.Timing.PhaseStart = jobProgress.StartedAt.Format(time.RFC3339)
	response.Timing.PhaseElapsedMs = response.Timing.ElapsedMs

	// 🎯 ETA CALCULATION: Based on current progress and throughput
	response.Timing.ETASeconds = calculateETA(jobProgress)

	// Throughput information (use aggregate throughput)
	response.Throughput.CurrentMBps = float64(jobProgress.Aggregate.ThroughputBPS) / (1024 * 1024)
	response.Throughput.AverageMBps = response.Throughput.CurrentMBps
	response.Throughput.PeakMBps = response.Throughput.CurrentMBps
	response.Throughput.LastUpdate = jobProgress.UpdatedAt.Format(time.RFC3339)

	// VM Information
	response.VMInfo.Name = "unknown" // TODO: Extract from job context
	response.VMInfo.Path = "unknown" // TODO: Extract from job context
	response.VMInfo.DiskSizeBytes = jobProgress.Aggregate.TotalBytes
	response.VMInfo.DiskSizeGB = jobProgress.Aggregate.TotalBytes / (1024 * 1024 * 1024)
	response.VMInfo.CBTEnabled = (jobProgress.CBT.Type == progress.CBTTypeFull || jobProgress.CBT.Type == progress.CBTTypeIncremental)

	// Phase progression
	response.Phases = []struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		StartTime  string `json:"start_time"`
		EndTime    string `json:"end_time"`
		DurationMs int64  `json:"duration_ms"`
	}{
		{
			Name:       string(jobProgress.Stage),
			Status:     string(jobProgress.Status),
			StartTime:  jobProgress.StartedAt.Format(time.RFC3339),
			EndTime:    "",
			DurationMs: response.Timing.ElapsedMs,
		},
	}

	// 🎯 MULTI-DISK PROGRESS INFORMATION
	// Convert disk progress to API format
	if len(jobProgress.Disks) > 0 {
		response.DiskProgresses = make([]DiskProgressInfo, len(jobProgress.Disks))
		for i, disk := range jobProgress.Disks {
			response.DiskProgresses[i] = DiskProgressInfo{
				DiskID:           disk.ID,
				Label:            disk.Label,
				BytesTransferred: disk.BytesTransferred,
				TotalBytes:       disk.PlannedBytes,
				Percent:          disk.Percent,
				ThroughputMBps:   float64(disk.ThroughputBPS) / (1024 * 1024),
				Status:           string(disk.Status),
			}
		}
	}

	return response
}

// GetJobProgress handles GET /progress/{jobId} requests for SHA Progress Poller
func (h *ProgressHandler) GetJobProgress(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from URL path
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	if jobID == "" {
		http.Error(w, "job ID is required", http.StatusBadRequest)
		return
	}

	log.WithField("job_id", jobID).Info("🔍 SNA API: GET progress request received")
	log.WithField("job_id", jobID).Debug("SNA API: Getting progress for SHA poller")

	// Get progress from service
	jobProgress, err := h.progressService.GetJobProgress(r.Context(), jobID)
	if err != nil {
		if err.Error() == "job not found: "+jobID {
			log.WithField("job_id", jobID).Debug("SNA API: Job not found for progress request")
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		log.WithError(err).WithField("job_id", jobID).Error("SNA API: Error getting job progress")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to SNAProgressResponse format for SHA
	snaResponse := convertToVMAProgressResponse(jobProgress)

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	// Encode and send response
	if err := json.NewEncoder(w).Encode(snaResponse); err != nil {
		log.WithError(err).WithField("job_id", jobID).Error("SNA API: Failed to encode progress response")
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"job_id":     jobID,
		"percentage": snaResponse.Percentage,
		"phase":      snaResponse.Phase,
		"status":     snaResponse.Status,
	}).Debug("SNA API: Successfully returned progress to SHA poller")
}

// UpdateJobProgress handles POST /progress/{jobId}/update requests from migratekit
func (h *ProgressHandler) UpdateJobProgress(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from URL path
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	if jobID == "" {
		http.Error(w, "job ID is required", http.StatusBadRequest)
		return
	}

	// Parse the progress update from request body
	var update ProgressUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert ProgressUpdate to ProgressUpdateRequest
	updateReq := &services.ProgressUpdateRequest{
		Stage:            update.Stage,
		Status:           update.Status,
		BytesTransferred: update.BytesTransferred,
		TotalBytes:       update.TotalBytes,
		ThroughputBPS:    update.ThroughputBPS,
		Percent:          update.Percent,
		DiskID:           update.DiskID,
		SyncType:         update.SyncType, // 🎯 FIX: Pass sync type to progress service
	}

	// Update progress via service - try original job ID first
	err := h.progressService.UpdateJobProgressFromMigratekit(r.Context(), jobID, updateReq)
	if err != nil {
		// If job not found, auto-initialize progress tracking for any job ID
		if err.Error() == "job not found: "+jobID {
			log.WithField("job_id", jobID).Info("🎯 Auto-initializing progress tracking for new job ID")

			// Auto-initialize job progress tracking
			initErr := h.progressService.InitializeJobWithNBDExport(r.Context(), jobID, jobID)
			if initErr == nil {
				// Retry with the newly initialized job ID
				err = h.progressService.UpdateJobProgressFromMigratekit(r.Context(), jobID, updateReq)
				if err == nil {
					log.WithField("job_id", jobID).Info("✅ Successfully auto-initialized and updated progress for job")
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"status":           "success",
						"auto_initialized": true,
						"job_id":           jobID,
					})
					return
				} else {
					log.WithError(err).WithField("job_id", jobID).Error("Failed to update progress after auto-initialization")
					http.Error(w, "failed to update progress after auto-initialization: "+err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				log.WithError(initErr).WithField("job_id", jobID).Error("Failed to auto-initialize job progress")
				http.Error(w, "failed to auto-initialize job progress: "+initErr.Error(), http.StatusInternalServerError)
				return
			}

			// If auto-initialization failed, try legacy mapping strategies
			var actualJobID string
			var mapErr error

			// Strategy 1: Check if jobID is an NBD export name (migration-vol-UUID format)
			if strings.HasPrefix(jobID, "migration-vol-") {
				log.WithField("export_name", jobID).Debug("Job ID appears to be NBD export name, attempting mapping")
				actualJobID, mapErr = h.progressService.FindJobByNBDExport(r.Context(), jobID)
				if mapErr == nil {
					log.WithFields(log.Fields{
						"export_name":   jobID,
						"mapped_job_id": actualJobID,
					}).Info("Successfully mapped NBD export name to job ID")
				}
			}

			// Strategy 2: Check if jobID is vm-disk format and use fallback mapping
			if mapErr != nil && strings.Contains(jobID, "-disk-") {
				log.WithField("original_job_id", jobID).Debug("Job ID not found, attempting VM-disk format mapping")
				actualJobID, mapErr = h.progressService.FindActiveJobForVMDisk(r.Context(), jobID)
				if mapErr == nil {
					log.WithFields(log.Fields{
						"original_job_id": jobID,
						"mapped_job_id":   actualJobID,
					}).Info("Successfully mapped VM-disk job ID to actual job ID")
				}
			}

			// If mapping succeeded, retry with the mapped job ID
			if mapErr == nil && actualJobID != "" {
				err = h.progressService.UpdateJobProgressFromMigratekit(r.Context(), actualJobID, updateReq)
				if err == nil {
					// Success with mapped ID
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"status":        "success",
						"mapped_from":   jobID,
						"actual_job_id": actualJobID,
					})
					return
				}
			}
		}

		if err.Error() == "job not found: "+jobID {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update progress: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"job_id": jobID,
	})
}

// calculateETA calculates estimated time to completion based on current progress and throughput
func calculateETA(jobProgress *progress.ReplicationProgress) int {
	// If job is completed or failed, ETA is 0
	if jobProgress.Status == progress.StatusSucceeded || jobProgress.Status == progress.StatusFailed {
		return 0
	}

	// If no progress yet, can't calculate ETA
	if jobProgress.Aggregate.Percent <= 0 || jobProgress.Aggregate.ThroughputBPS <= 0 {
		return 0
	}

	// Calculate remaining bytes
	remainingBytes := jobProgress.Aggregate.TotalBytes - jobProgress.Aggregate.BytesTransferred
	if remainingBytes <= 0 {
		return 0 // Almost done
	}

	// Calculate ETA in seconds based on current throughput
	etaSeconds := float64(remainingBytes) / float64(jobProgress.Aggregate.ThroughputBPS)

	// Cap ETA at 24 hours to avoid unrealistic estimates
	if etaSeconds > 86400 {
		etaSeconds = 86400
	}

	return int(etaSeconds)
}

// mapStageToOperation maps internal stage names to user-friendly operation descriptions
func mapStageToOperation(stage progress.ReplicationStage, status progress.ReplicationStatus) string {
	// Handle completion states first
	if status == progress.StatusSucceeded {
		return "Completed"
	}
	if status == progress.StatusFailed {
		return "Failed"
	}

	// Map stage to operation description
	switch stage {
	case progress.StageDiscover:
		return "Discovering VM"
	case progress.StageEnableCBT:
		return "Enabling CBT"
	case progress.StageQueryCBT:
		return "Querying CBT"
	case progress.StageSnapshot:
		return "Creating Snapshot"
	case progress.StagePrepareVolumes:
		return "Preparing Volumes"
	case progress.StageStartExports:
		return "Starting Exports"
	case progress.StageTransfer:
		return "Transferring Data"
	case progress.StageFinalize:
		return "Finalizing"
	case progress.StagePersistChangeIDs:
		return "Persisting Change IDs"
	default:
		// Fallback to stage name if no mapping found
		return string(stage)
	}
}

// RegisterRoutes registers the progress handler routes with the router
func (h *ProgressHandler) RegisterRoutes(router *mux.Router) {
	// Create API v1 subrouter to match server route structure
	api := router.PathPrefix("/api/v1").Subrouter()

	// GET /api/v1/progress/{jobId} - Get progress for SHA poller
	api.HandleFunc("/progress/{jobId}", h.GetJobProgress).Methods("GET")

	// POST /api/v1/progress/{jobId}/update - Update progress from migratekit
	api.HandleFunc("/progress/{jobId}/update", h.UpdateJobProgress).Methods("POST")
}
