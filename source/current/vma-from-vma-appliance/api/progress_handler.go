package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

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

// GetJobProgress handles GET /progress/{jobId} requests
func (h *ProgressHandler) GetJobProgress(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from URL path
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	if jobID == "" {
		http.Error(w, "job ID is required", http.StatusBadRequest)
		return
	}

	// Get progress from service
	progress, err := h.progressService.GetJobProgress(r.Context(), jobID)
	if err != nil {
		if err.Error() == "job not found: "+jobID {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	// Encode and send response
	if err := json.NewEncoder(w).Encode(progress); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
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
	}

	// Update progress via service
	err := h.progressService.UpdateJobProgressFromMigratekit(r.Context(), jobID, updateReq)
	if err != nil {
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

// RegisterRoutes registers the progress handler routes with the router
func (h *ProgressHandler) RegisterRoutes(router *mux.Router) {
	// GET /progress/{jobId} - Get progress for a specific job
	router.HandleFunc("/progress/{jobId}", h.GetJobProgress).Methods("GET")

	// POST /progress/{jobId}/update - Update progress from migratekit
	router.HandleFunc("/progress/{jobId}/update", h.UpdateJobProgress).Methods("POST")
}
