// Package handlers provides REST API endpoints for protection flow management
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/services"
)

// ProtectionFlowHandler handles protection flow CRUD API endpoints
type ProtectionFlowHandler struct {
	flowService *services.ProtectionFlowService
	tracker     *joblog.Tracker
}

// NewProtectionFlowHandler creates a new protection flow handler
func NewProtectionFlowHandler(
	flowService *services.ProtectionFlowService,
	tracker *joblog.Tracker,
) *ProtectionFlowHandler {
	return &ProtectionFlowHandler{
		flowService: flowService,
		tracker:     tracker,
	}
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// CreateFlowRequest represents a request to create a new protection flow
type CreateFlowRequest struct {
	Name         string  `json:"name" validate:"required,min=1,max=255"`
	Description  *string `json:"description,omitempty"`
	FlowType     string  `json:"flow_type" validate:"required,oneof=backup replication"`
	TargetType   string  `json:"target_type" validate:"required,oneof=vm group"`
	TargetID     string  `json:"target_id" validate:"required"`
	RepositoryID *string `json:"repository_id,omitempty"`
	PolicyID     *string `json:"policy_id,omitempty"`
	ScheduleID   *string `json:"schedule_id,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
}

// UpdateFlowRequest represents a request to update an existing protection flow
type UpdateFlowRequest struct {
	Name         *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description  *string `json:"description,omitempty"`
	RepositoryID *string `json:"repository_id,omitempty"`
	PolicyID     *string `json:"policy_id,omitempty"`
	ScheduleID   *string `json:"schedule_id,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
}

// FlowResponse represents a protection flow in API responses
type FlowResponse struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description *string                 `json:"description,omitempty"`
	FlowType    string                  `json:"flow_type"`
	TargetType  string                  `json:"target_type"`
	TargetID    string                  `json:"target_id"`
	TargetName  *string                 `json:"target_name,omitempty"` // Resolved name
	RepositoryID *string                `json:"repository_id,omitempty"`
	RepositoryName *string              `json:"repository_name,omitempty"` // Resolved name
	PolicyID     *string                `json:"policy_id,omitempty"`
	PolicyName   *string                `json:"policy_name,omitempty"` // Resolved name
	ScheduleID   *string                `json:"schedule_id,omitempty"`
	ScheduleName *string                `json:"schedule_name,omitempty"` // Resolved name
	ScheduleCron *string                `json:"schedule_cron,omitempty"` // Cron expression
	Enabled      bool                   `json:"enabled"`
	Status       FlowStatusResponse     `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CreatedBy    string                 `json:"created_by"`
}

// FlowStatusResponse represents flow execution status
type FlowStatusResponse struct {
	LastExecutionID     *string    `json:"last_execution_id,omitempty"`
	LastExecutionStatus string     `json:"last_execution_status"`
	LastExecutionTime   *time.Time `json:"last_execution_time,omitempty"`
	NextExecutionTime   *time.Time `json:"next_execution_time,omitempty"`
	TotalExecutions     int        `json:"total_executions"`
	SuccessfulExecutions int       `json:"successful_executions"`
	FailedExecutions    int       `json:"failed_executions"`
}

// ExecutionResponse represents a flow execution in API responses
type ExecutionResponse struct {
	ID                   string     `json:"id"`
	FlowID               string     `json:"flow_id"`
	FlowName             string     `json:"flow_name"`
	Status               string     `json:"status"`
	ExecutionType        string     `json:"execution_type"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	ExecutionTimeSeconds int        `json:"execution_time_seconds,omitempty"`
	JobsCreated          int        `json:"jobs_created"`
	JobsCompleted        int        `json:"jobs_completed"`
	JobsFailed           int        `json:"jobs_failed"`
	JobsSkipped          int        `json:"jobs_skipped"`
	VMsProcessed         int        `json:"vms_processed"`
	BytesTransferred     int64      `json:"bytes_transferred"`
	ErrorMessage         *string    `json:"error_message,omitempty"`
	CreatedJobIDs        []string   `json:"created_job_ids,omitempty"`
	TriggeredBy          string     `json:"triggered_by"`
	CreatedAt            time.Time  `json:"created_at"`
}

// BulkOperationRequest represents a bulk operation request
type BulkOperationRequest struct {
	FlowIDs []string `json:"flow_ids" validate:"required,min=1"`
}

// FlowSummaryResponse represents aggregated flow statistics
type FlowSummaryResponse struct {
	TotalFlows           int `json:"total_flows"`
	EnabledFlows         int `json:"enabled_flows"`
	DisabledFlows        int `json:"disabled_flows"`
	BackupFlows          int `json:"backup_flows"`
	ReplicationFlows     int `json:"replication_flows"`
	TotalExecutionsToday int `json:"total_executions_today"`
	SuccessfulExecutionsToday int `json:"successful_executions_today"`
	FailedExecutionsToday int `json:"failed_executions_today"`
}

// =============================================================================
// CRUD OPERATIONS
// =============================================================================

// CreateFlow handles POST /api/v1/protection-flows
func (h *ProtectionFlowHandler) CreateFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Create the flow
	flow, err := h.flowService.CreateFlow(ctx, services.CreateFlowRequest{
		Name:         req.Name,
		Description:  req.Description,
		FlowType:     req.FlowType,
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		RepositoryID: req.RepositoryID,
		PolicyID:     req.PolicyID,
		ScheduleID:   req.ScheduleID,
		Enabled:      req.Enabled,
	})
	if err != nil {
		log.WithError(err).Error("Failed to create protection flow")
		h.sendError(w, http.StatusInternalServerError, "Failed to create flow", err.Error())
		return
	}

	// Convert to response format
	response := h.convertFlowToResponse(flow)

	log.WithFields(log.Fields{
		"flow_id":   flow.ID,
		"flow_name": flow.Name,
	}).Info("Protection flow created successfully")

	h.writeJSON(w, http.StatusCreated, response)
}

// ListFlows handles GET /api/v1/protection-flows
func (h *ProtectionFlowHandler) ListFlows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filters := database.FlowFilters{}
	if flowType := r.URL.Query().Get("flow_type"); flowType != "" {
		filters.FlowType = &flowType
	}
	if targetType := r.URL.Query().Get("target_type"); targetType != "" {
		filters.TargetType = &targetType
	}
	if enabledStr := r.URL.Query().Get("enabled"); enabledStr != "" {
		if enabled, err := strconv.ParseBool(enabledStr); err == nil {
			filters.Enabled = &enabled
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	flows, err := h.flowService.ListFlows(ctx, filters)
	if err != nil {
		log.WithError(err).Error("Failed to list protection flows")
		h.sendError(w, http.StatusInternalServerError, "Failed to list flows", err.Error())
		return
	}

	// Convert to response format
	response := make([]FlowResponse, len(flows))
	for i, flow := range flows {
		response[i] = h.convertFlowToResponse(flow)
	}

	total := len(response) // TODO: Implement proper pagination counting

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"flows": response,
		"total": total,
	})
}

// GetFlow handles GET /api/v1/protection-flows/{id}
func (h *ProtectionFlowHandler) GetFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flowID := mux.Vars(r)["id"]

	flow, err := h.flowService.GetFlow(ctx, flowID)
	if err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to get protection flow")
		h.sendError(w, http.StatusNotFound, "Flow not found", err.Error())
		return
	}

	response := h.convertFlowToResponse(flow)
	h.writeJSON(w, http.StatusOK, response)
}

// UpdateFlow handles PUT /api/v1/protection-flows/{id}
func (h *ProtectionFlowHandler) UpdateFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flowID := mux.Vars(r)["id"]

	var req UpdateFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.RepositoryID != nil {
		updates["repository_id"] = *req.RepositoryID
	}
	if req.PolicyID != nil {
		updates["policy_id"] = *req.PolicyID
	}
	if req.ScheduleID != nil {
		updates["schedule_id"] = *req.ScheduleID
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if err := h.flowService.UpdateFlow(ctx, flowID, updates); err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to update protection flow")
		h.sendError(w, http.StatusInternalServerError, "Failed to update flow", err.Error())
		return
	}

	// Return updated flow
	flow, err := h.flowService.GetFlow(ctx, flowID)
	if err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to get updated flow")
		h.sendError(w, http.StatusInternalServerError, "Failed to get updated flow", err.Error())
		return
	}

	response := h.convertFlowToResponse(flow)
	h.writeJSON(w, http.StatusOK, response)
}

// DeleteFlow handles DELETE /api/v1/protection-flows/{id}
func (h *ProtectionFlowHandler) DeleteFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flowID := mux.Vars(r)["id"]

	if err := h.flowService.DeleteFlow(ctx, flowID); err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to delete protection flow")
		h.sendError(w, http.StatusInternalServerError, "Failed to delete flow", err.Error())
		return
	}

	log.WithField("flow_id", flowID).Info("Protection flow deleted successfully")
	w.WriteHeader(http.StatusNoContent)
}

// =============================================================================
// CONTROL OPERATIONS
// =============================================================================

// EnableFlow handles PATCH /api/v1/protection-flows/{id}/enable
func (h *ProtectionFlowHandler) EnableFlow(w http.ResponseWriter, r *http.Request) {
	h.setFlowEnabled(w, r, true)
}

// DisableFlow handles PATCH /api/v1/protection-flows/{id}/disable
func (h *ProtectionFlowHandler) DisableFlow(w http.ResponseWriter, r *http.Request) {
	h.setFlowEnabled(w, r, false)
}

func (h *ProtectionFlowHandler) setFlowEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	ctx := r.Context()
	flowID := mux.Vars(r)["id"]

	updates := map[string]interface{}{"enabled": enabled}
	if err := h.flowService.UpdateFlow(ctx, flowID, updates); err != nil {
		action := "disable"
		if enabled {
			action = "enable"
		}
		log.WithError(err).WithField("flow_id", flowID).Error(fmt.Sprintf("Failed to %s flow", action))
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to %s flow", action), err.Error())
		return
	}

	// Return updated flow
	flow, err := h.flowService.GetFlow(ctx, flowID)
	if err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to get updated flow")
		h.sendError(w, http.StatusInternalServerError, "Failed to get updated flow", err.Error())
		return
	}

	response := h.convertFlowToResponse(flow)
	h.writeJSON(w, http.StatusOK, response)
}

// =============================================================================
// EXECUTION OPERATIONS
// =============================================================================

// ExecuteFlow handles POST /api/v1/protection-flows/{id}/execute
func (h *ProtectionFlowHandler) ExecuteFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flowID := mux.Vars(r)["id"]

	execution, err := h.flowService.ExecuteFlow(ctx, flowID, "manual")
	if err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to execute protection flow")
		h.sendError(w, http.StatusInternalServerError, "Failed to execute flow", err.Error())
		return
	}

	response := h.convertExecutionToResponse(execution)
	h.writeJSON(w, http.StatusOK, response)
}

// GetFlowExecutions handles GET /api/v1/protection-flows/{id}/executions
func (h *ProtectionFlowHandler) GetFlowExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flowID := mux.Vars(r)["id"]

	limit := 10 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	executions, err := h.flowService.GetFlowExecutions(ctx, flowID, limit)
	if err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to get flow executions")
		h.sendError(w, http.StatusInternalServerError, "Failed to get executions", err.Error())
		return
	}

	response := make([]ExecutionResponse, len(executions))
	for i, execution := range executions {
		response[i] = h.convertExecutionToResponse(execution)
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"executions": response,
		"total":      len(response),
	})
}

// GetFlowStatus handles GET /api/v1/protection-flows/{id}/status
func (h *ProtectionFlowHandler) GetFlowStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flowID := mux.Vars(r)["id"]

	status, err := h.flowService.GetFlowStatus(ctx, flowID)
	if err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to get flow status")
		h.sendError(w, http.StatusNotFound, "Flow not found", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, status)
}

// TestFlow handles POST /api/v1/protection-flows/{id}/test
func (h *ProtectionFlowHandler) TestFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flowID := mux.Vars(r)["id"]

	flow, err := h.flowService.GetFlow(ctx, flowID)
	if err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to get flow for testing")
		h.sendError(w, http.StatusNotFound, "Flow not found", err.Error())
		return
	}

	// Validate flow configuration
	if err := h.flowService.ValidateFlowConfiguration(ctx, flow); err != nil {
		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":  false,
			"errors": []string{err.Error()},
		})
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}

// =============================================================================
// BULK OPERATIONS
// =============================================================================

// BulkEnableFlows handles POST /api/v1/protection-flows/bulk-enable
func (h *ProtectionFlowHandler) BulkEnableFlows(w http.ResponseWriter, r *http.Request) {
	h.bulkSetEnabled(w, r, true)
}

// BulkDisableFlows handles POST /api/v1/protection-flows/bulk-disable
func (h *ProtectionFlowHandler) BulkDisableFlows(w http.ResponseWriter, r *http.Request) {
	h.bulkSetEnabled(w, r, false)
}

func (h *ProtectionFlowHandler) bulkSetEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	ctx := r.Context()

	var req BulkOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	action := "enable"
	if !enabled {
		action = "disable"
	}

	successful := 0
	failed := 0
	var errors []string

	for _, flowID := range req.FlowIDs {
		updates := map[string]interface{}{"enabled": enabled}
		if err := h.flowService.UpdateFlow(ctx, flowID, updates); err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Failed to %s flow %s: %v", action, flowID, err))
		} else {
			successful++
		}
	}

	log.WithFields(log.Fields{
		"action":     action,
		"requested":  len(req.FlowIDs),
		"successful": successful,
		"failed":     failed,
	}).Info("Bulk flow operation completed")

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"action":     action,
		"requested":  len(req.FlowIDs),
		"successful": successful,
		"failed":     failed,
		"errors":     errors,
	})
}

// BulkDeleteFlows handles POST /api/v1/protection-flows/bulk-delete
func (h *ProtectionFlowHandler) BulkDeleteFlows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req BulkOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	successful := 0
	failed := 0
	var errors []string

	for _, flowID := range req.FlowIDs {
		if err := h.flowService.DeleteFlow(ctx, flowID); err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Failed to delete flow %s: %v", flowID, err))
		} else {
			successful++
		}
	}

	log.WithFields(log.Fields{
		"requested":  len(req.FlowIDs),
		"successful": successful,
		"failed":     failed,
	}).Info("Bulk flow deletion completed")

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"action":     "delete",
		"requested":  len(req.FlowIDs),
		"successful": successful,
		"failed":     failed,
		"errors":     errors,
	})
}

// GetFlowSummary handles GET /api/v1/protection-flows/summary
func (h *ProtectionFlowHandler) GetFlowSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all flows
	flows, err := h.flowService.ListFlows(ctx, database.FlowFilters{})
	if err != nil {
		log.WithError(err).Error("Failed to get flows for summary")
		h.sendError(w, http.StatusInternalServerError, "Failed to get flow summary", err.Error())
		return
	}

	summary := FlowSummaryResponse{}

	for _, flow := range flows {
		summary.TotalFlows++

		if flow.Enabled {
			summary.EnabledFlows++
		} else {
			summary.DisabledFlows++
		}

		if flow.FlowType == "backup" {
			summary.BackupFlows++
		} else if flow.FlowType == "replication" {
			summary.ReplicationFlows++
		}

		// Simple execution counting (could be optimized with database aggregation)
		if flow.LastExecutionTime != nil {
			today := time.Now().Truncate(24 * time.Hour)
			if flow.LastExecutionTime.After(today) || flow.LastExecutionTime.Equal(today) {
				summary.TotalExecutionsToday++
				if flow.LastExecutionStatus == "success" {
					summary.SuccessfulExecutionsToday++
				} else if flow.LastExecutionStatus == "error" {
					summary.FailedExecutionsToday++
				}
			}
		}
	}

	h.writeJSON(w, http.StatusOK, summary)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// convertFlowToResponse converts a database flow to API response format
func (h *ProtectionFlowHandler) convertFlowToResponse(flow *database.ProtectionFlow) FlowResponse {
	response := FlowResponse{
		ID:         flow.ID,
		Name:       flow.Name,
		Description: flow.Description,
		FlowType:   flow.FlowType,
		TargetType: flow.TargetType,
		TargetID:   flow.TargetID,
		Enabled:    flow.Enabled,
		Status: FlowStatusResponse{
			LastExecutionID:     flow.LastExecutionID,
			LastExecutionStatus: flow.LastExecutionStatus,
			LastExecutionTime:   flow.LastExecutionTime,
			NextExecutionTime:   flow.NextExecutionTime,
			TotalExecutions:     flow.TotalExecutions,
			SuccessfulExecutions: flow.SuccessfulExecutions,
			FailedExecutions:    flow.FailedExecutions,
		},
		CreatedAt: flow.CreatedAt,
		UpdatedAt: flow.UpdatedAt,
		CreatedBy: flow.CreatedBy,
	}

	// Set optional fields
	if flow.RepositoryID != nil {
		response.RepositoryID = flow.RepositoryID
	}
	if flow.PolicyID != nil {
		response.PolicyID = flow.PolicyID
	}
	if flow.ScheduleID != nil {
		response.ScheduleID = flow.ScheduleID
	}

	// Resolve related names (simplified - could be enhanced with joins)
	if flow.Schedule != nil {
		response.ScheduleName = &flow.Schedule.Name
		response.ScheduleCron = &flow.Schedule.CronExpression
	}
	if flow.Repository != nil {
		response.RepositoryName = &flow.Repository.Name
	}
	if flow.Policy != nil {
		response.PolicyName = &flow.Policy.Name
	}

	// TODO: Resolve target name (VM name or group name)

	return response
}

// convertExecutionToResponse converts a database execution to API response format
func (h *ProtectionFlowHandler) convertExecutionToResponse(execution *database.ProtectionFlowExecution) ExecutionResponse {
	response := ExecutionResponse{
		ID:            execution.ID,
		FlowID:        execution.FlowID,
		Status:        execution.Status,
		ExecutionType: execution.ExecutionType,
		StartedAt:     execution.StartedAt,
		CompletedAt:   execution.CompletedAt,
		ExecutionTimeSeconds: execution.ExecutionTimeSeconds,
		JobsCreated:   execution.JobsCreated,
		JobsCompleted: execution.JobsCompleted,
		JobsFailed:    execution.JobsFailed,
		JobsSkipped:   execution.JobsSkipped,
		VMsProcessed:  execution.VMsProcessed,
		BytesTransferred: execution.BytesTransferred,
		ErrorMessage:  execution.ErrorMessage,
		TriggeredBy:   execution.TriggeredBy,
		CreatedAt:     execution.CreatedAt,
	}

	// Parse created job IDs if present
	if execution.CreatedJobIDs != nil {
		// Simple JSON parsing (could be enhanced)
		var jobIDs []string
		if err := json.Unmarshal([]byte(*execution.CreatedJobIDs), &jobIDs); err == nil {
			response.CreatedJobIDs = jobIDs
		}
	}

	// Get flow name (simplified - could be optimized)
	if execution.Flow != nil {
		response.FlowName = execution.Flow.Name
	}

	return response
}

// sendError sends a standardized error response
func (h *ProtectionFlowHandler) sendError(w http.ResponseWriter, statusCode int, message, details string) {
	errorResponse := map[string]interface{}{
		"error":   message,
		"details": details,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

// writeJSON sends a JSON response
func (h *ProtectionFlowHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
