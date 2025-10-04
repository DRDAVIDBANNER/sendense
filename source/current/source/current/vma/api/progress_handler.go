package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/vexxhost/migratekit/source/current/vma/services"
)

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

	// Parse the JSON request body
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Extract progress fields
	stage, hasStage := updateData["stage"].(string)
	bytesTransferred, hasBytesTransferred := updateData["bytes_transferred"].(float64)
	percentComplete, hasPercentComplete := updateData["percent_complete"].(float64)
	transferRate, hasTransferRate := updateData["transfer_rate"].(float64)

	// Get current job progress
	jobProgress, err := h.progressService.GetJobProgress(r.Context(), jobID)
	if err != nil {
		// Job not found - initialize it
		if err.Error() == "job not found: "+jobID {
			err = h.progressService.StartJobTracking(r.Context(), jobID)
			if err != nil {
				http.Error(w, "failed to start job tracking", http.StatusInternalServerError)
				return
			}
			jobProgress, err = h.progressService.GetJobProgress(r.Context(), jobID)
			if err != nil {
				http.Error(w, "failed to get job progress", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Update fields if provided
	if hasStage {
		// Convert string to ReplicationStage (defined in progress package)
		jobProgress.Stage = progress.ReplicationStage(stage)
	}
	if hasBytesTransferred {
		jobProgress.Aggregate.BytesTransferred = int64(bytesTransferred)
	}
	if hasPercentComplete {
		jobProgress.Aggregate.Percent = percentComplete
	}
	if hasTransferRate {
		jobProgress.Aggregate.ThroughputBPS = int64(transferRate)
	}

	// Update the job progress
	err = h.progressService.UpdateJobProgress(r.Context(), jobID, jobProgress)
	if err != nil {
		http.Error(w, "failed to update job progress", http.StatusInternalServerError)
		return
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// RegisterRoutes registers the progress handler routes with the router
func (h *ProgressHandler) RegisterRoutes(router *mux.Router) {
	// GET /progress/{jobId} - Get progress for a specific job
	router.HandleFunc("/progress/{jobId}", h.GetJobProgress).Methods("GET")
	// POST /progress/{jobId}/update - Update progress from migratekit
	router.HandleFunc("/progress/{jobId}/update", h.UpdateJobProgress).Methods("POST")
}
