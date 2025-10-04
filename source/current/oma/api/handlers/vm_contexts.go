// Package handlers - VM Context API endpoints for GUI integration
package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/failover"
	"github.com/vexxhost/migratekit-oma/joblog"
)

// VMContextHandler handles VM context API endpoints
type VMContextHandler struct {
	db            database.Connection
	vmContextRepo *database.VMReplicationContextRepository
	jobTracker    *joblog.Tracker
}

// NewVMContextHandler creates a new VM context handler
func NewVMContextHandler(db database.Connection, jobTracker *joblog.Tracker) *VMContextHandler {
	return &VMContextHandler{
		db:            db,
		vmContextRepo: database.NewVMReplicationContextRepository(db),
		jobTracker:    jobTracker,
	}
}

// OperationSummary represents a sanitized summary of the last operation
type OperationSummary struct {
	JobID           string    `json:"job_id"`
	ExternalJobID   string    `json:"external_job_id,omitempty"`
	OperationType   string    `json:"operation_type"` // "replication", "test_failover", "live_failover", "rollback"
	Status          string    `json:"status"`
	Progress        float64   `json:"progress"`
	FailedStep      string    `json:"failed_step,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"` // Sanitized, user-friendly
	ErrorCategory   string    `json:"error_category,omitempty"`
	ErrorSeverity   string    `json:"error_severity,omitempty"`
	ActionableSteps []string  `json:"actionable_steps,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	DurationSeconds int64     `json:"duration_seconds"`
	StepsCompleted  int       `json:"steps_completed,omitempty"`
	StepsTotal      int       `json:"steps_total,omitempty"`
}

// GetVMContext retrieves complete VM context information for GUI
// GET /api/v1/vm-contexts/{vm_name}
func (h *VMContextHandler) GetVMContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmName := vars["vm_name"]

	if vmName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "VM name is required")
		return
	}

	log.WithField("vm_name", vmName).Info("Getting VM context with full details")

	vmContextDetails, err := h.vmContextRepo.GetVMContextWithFullDetails(vmName)
	if err != nil {
		log.WithError(err).WithField("vm_name", vmName).Error("Failed to get VM context details")
		
		// Check if it's a not found error
		if err.Error() == "VM context not found for: "+vmName {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve VM context")
		return
	}

	// Parse and include last operation summary if present
	if vmContextDetails.Context.LastOperationSummary != nil && *vmContextDetails.Context.LastOperationSummary != "" {
		var opSummary map[string]interface{}
		if err := json.Unmarshal([]byte(*vmContextDetails.Context.LastOperationSummary), &opSummary); err == nil {
			vmContextDetails.LastOperation = opSummary
		} else {
			log.WithError(err).Warn("Failed to parse last_operation_summary JSON")
		}
	}

	log.WithFields(log.Fields{
		"vm_name":           vmName,
		"context_id":        vmContextDetails.Context.ContextID,
		"current_job":       vmContextDetails.Context.CurrentJobID,
		"job_history_count": len(vmContextDetails.JobHistory),
		"disks_count":       len(vmContextDetails.Disks),
		"cbt_history_count": len(vmContextDetails.CBTHistory),
		"has_operation_summary": vmContextDetails.Context.LastOperationSummary != nil,
	}).Info("Successfully retrieved VM context details with operation summary")

	h.writeJSONResponse(w, http.StatusOK, vmContextDetails)
}

// ListVMContexts retrieves all VM contexts with summary information
// GET /api/v1/vm-contexts
func (h *VMContextHandler) ListVMContexts(w http.ResponseWriter, r *http.Request) {
	log.Info("Listing all VM contexts")

	contexts, err := h.vmContextRepo.ListVMContexts()
	if err != nil {
		log.WithError(err).Error("Failed to list VM contexts")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve VM contexts")
		return
	}

	response := map[string]interface{}{
		"vm_contexts": contexts,
		"count":       len(contexts),
	}

	log.WithField("count", len(contexts)).Info("Successfully retrieved VM contexts")
	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse writes a JSON response
func (h *VMContextHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to write JSON response")
	}
}

// writeErrorResponse writes an error response
func (h *VMContextHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}

// UnifiedJobItem represents a job from any source (replication, failover, rollback) with sanitized display
type UnifiedJobItem struct {
	JobID           string     `json:"job_id"`
	ExternalJobID   string     `json:"external_job_id,omitempty"`
	JobType         string     `json:"job_type"` // "replication", "test_failover", "live_failover", "rollback"
	Status          string     `json:"status"`
	Progress        float64    `json:"progress"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	
	// User-friendly display fields
	DisplayName      string   `json:"display_name"` // "Incremental Replication", "Test Failover", etc.
	CurrentStep      string   `json:"current_step,omitempty"`
	ErrorMessage     string   `json:"error_message,omitempty"` // Sanitized
	ErrorCategory    string   `json:"error_category,omitempty"`
	ActionableSteps  []string `json:"actionable_steps,omitempty"`
	
	// Metadata
	DataSource       string `json:"data_source"` // "replication_jobs" or "job_tracking"
	DurationSeconds  int64  `json:"duration_seconds,omitempty"`
}

// GetRecentJobs retrieves all recent operations (replication + failover + rollback) for a VM
// GET /api/v1/vm-contexts/{context_id}/recent-jobs
func (h *VMContextHandler) GetRecentJobs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contextID := vars["context_id"]
	
	if contextID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Context ID is required")
		return
	}
	
	log.WithField("context_id", contextID).Info("📋 Getting unified recent jobs for VM context")
	
	var allJobs []UnifiedJobItem
	
	// 1. Get replication jobs from replication_jobs table
	var replJobs []database.ReplicationJob
	err := h.db.GetGormDB().
		Where("vm_context_id = ?", contextID).
		Order("created_at DESC").
		Limit(10).
		Find(&replJobs).Error
	
	if err != nil {
		log.WithError(err).Warn("Failed to query replication jobs")
	} else {
		for _, job := range replJobs {
			item := UnifiedJobItem{
				JobID:        job.ID,
				JobType:      "replication",
				Status:       job.Status,
				Progress:     job.ProgressPercent,
				StartedAt:    job.CreatedAt,
				CompletedAt:  job.CompletedAt,
				DisplayName:  getReplicationDisplayName(job.ReplicationType, job.Status),
				CurrentStep:  job.CurrentOperation,
				DataSource:   "replication_jobs",
			}
			
			// Add duration if completed
			if job.CompletedAt != nil {
				item.DurationSeconds = int64(job.CompletedAt.Sub(job.CreatedAt).Seconds())
			}
			
			// Sanitize error message if present
			if job.ErrorMessage != "" {
				item.ErrorMessage = sanitizeReplicationError(job.ErrorMessage)
			}
			
			allJobs = append(allJobs, item)
		}
	}
	
	// 2. Get failover/rollback jobs from job_tracking via JobLog
	if h.jobTracker != nil {
		jobSummaries, err := h.jobTracker.GetJobByContextID(contextID)
		if err != nil {
			log.WithError(err).Warn("Failed to query job_tracking")
		} else {
			for _, summary := range jobSummaries {
				// Only include failover and cleanup (rollback) jobs
				if summary.Job.JobType == "failover" || summary.Job.JobType == "cleanup" {
					item := UnifiedJobItem{
						JobID:           summary.Job.ID,
						ExternalJobID:   getStringPtrValue(summary.Job.ExternalJobID),
						JobType:         getDisplayJobType(summary.Job.Operation),
						Status:          string(summary.Job.Status),
						Progress:        summary.Progress.StepCompletion,
						StartedAt:       summary.Job.StartedAt,
						CompletedAt:     summary.Job.CompletedAt,
						DisplayName:     getOperationDisplayName(summary.Job.Operation),
						CurrentStep:     getCurrentStepFriendly(summary.Steps),
						DataSource:      "job_tracking",
						DurationSeconds: summary.Progress.RuntimeSeconds,
					}
					
					// Add sanitized error if failed
					if summary.Job.Status == joblog.StatusFailed && summary.Job.ErrorMessage != nil {
						item.ErrorMessage, item.ErrorCategory, item.ActionableSteps = 
							extractSanitizedError(summary.Steps)
					}
					
					allJobs = append(allJobs, item)
				}
			}
		}
	}
	
	// 3. Sort all jobs by timestamp (most recent first)
	sort.Slice(allJobs, func(i, j int) bool {
		return allJobs[i].StartedAt.After(allJobs[j].StartedAt)
	})
	
	// 4. Limit to 20 most recent
	if len(allJobs) > 20 {
		allJobs = allJobs[:20]
	}
	
	log.WithFields(log.Fields{
		"context_id": contextID,
		"total_jobs": len(allJobs),
	}).Info("✅ Retrieved unified recent jobs")
	
	response := map[string]interface{}{
		"context_id": contextID,
		"jobs":       allJobs,
		"count":      len(allJobs),
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

// Helper functions

func getReplicationDisplayName(replicationType string, status string) string {
	if status == "completed" {
		return "Replication Completed"
	}
	if replicationType == "incremental" {
		return "Incremental Replication"
	}
	return "Initial Replication"
}

func getDisplayJobType(operation string) string {
	switch {
	case strings.Contains(operation, "test-failover"):
		return "test_failover"
	case strings.Contains(operation, "live-failover"):
		return "live_failover"
	case strings.Contains(operation, "rollback"):
		return "rollback"
	default:
		return "unknown"
	}
}

func getOperationDisplayName(operation string) string {
	switch {
	case strings.Contains(operation, "test-failover"):
		return "Test Failover"
	case strings.Contains(operation, "live-failover"):
		return "Live Failover"
	case strings.Contains(operation, "rollback"):
		return "Rollback"
	default:
		return "Operation"
	}
}

func getCurrentStepFriendly(steps []joblog.StepRecord) string {
	// Find the current running step or last completed step
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status == joblog.StatusRunning {
			return failover.GetUserFriendlyStepName(step.Name)
		}
		if step.Status == joblog.StatusCompleted && i == len(steps)-1 {
			return failover.GetUserFriendlyStepName(step.Name)
		}
	}
	return ""
}

func extractSanitizedError(steps []joblog.StepRecord) (string, string, []string) {
	// Find the failed step
	for _, step := range steps {
		if step.Status == joblog.StatusFailed && step.ErrorMessage != nil {
			// Sanitize the error
			sanitized := failover.SanitizeFailoverError(step.Name, 
				&simpleError{msg: *step.ErrorMessage})
			
			return sanitized.UserMessage, sanitized.Category, sanitized.ActionableSteps
		}
	}
	return "", "", nil
}

func getStringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func sanitizeReplicationError(errMsg string) string {
	if errMsg == "" {
		return ""
	}
	// Basic sanitization for replication errors
	// Can be enhanced later if needed
	return errMsg
}

// simpleError implements error interface for sanitization
type simpleError struct {
	msg string
}

func (e *simpleError) Error() string {
	return e.msg
}
