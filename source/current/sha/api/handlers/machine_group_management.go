// Package handlers provides REST API endpoints for machine group management
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/services"
)

// MachineGroupManagementHandler handles machine group CRUD API endpoints
type MachineGroupManagementHandler struct {
	machineGroupService *services.MachineGroupService
	schedulerRepo       *database.SchedulerRepository
	tracker             *joblog.Tracker
}

// NewMachineGroupManagementHandler creates a new machine group management handler
func NewMachineGroupManagementHandler(
	machineGroupService *services.MachineGroupService,
	schedulerRepo *database.SchedulerRepository,
	tracker *joblog.Tracker,
) *MachineGroupManagementHandler {
	return &MachineGroupManagementHandler{
		machineGroupService: machineGroupService,
		schedulerRepo:       schedulerRepo,
		tracker:             tracker,
	}
}

// CreateGroupRequest represents a request to create a new machine group
type CreateGroupRequest struct {
	Name             string  `json:"name" binding:"required,min=1,max=255"`
	Description      *string `json:"description,omitempty"`
	ScheduleID       *string `json:"schedule_id,omitempty"`
	MaxConcurrentVMs int     `json:"max_concurrent_vms" binding:"min=1,max=100"`
	Priority         int     `json:"priority" binding:"min=0,max=100"`
	CreatedBy        string  `json:"created_by,omitempty"`
}

// UpdateGroupRequest represents a request to update an existing machine group
type UpdateGroupRequest struct {
	Name             *string `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description      *string `json:"description,omitempty"`
	ScheduleID       *string `json:"schedule_id,omitempty"`
	MaxConcurrentVMs *int    `json:"max_concurrent_vms,omitempty" binding:"omitempty,min=1,max=100"`
	Priority         *int    `json:"priority,omitempty" binding:"omitempty,min=0,max=100"`
}

// GroupResponse represents a machine group with full details
type GroupResponse struct {
	ID               string                        `json:"id"`
	Name             string                        `json:"name"`
	Description      *string                       `json:"description,omitempty"`
	ScheduleID       *string                       `json:"schedule_id,omitempty"`
	ScheduleName     *string                       `json:"schedule_name,omitempty"`
	MaxConcurrentVMs int                           `json:"max_concurrent_vms"`
	Priority         int                           `json:"priority"`
	TotalVMs         int                           `json:"total_vms"`
	EnabledVMs       int                           `json:"enabled_vms"`
	DisabledVMs      int                           `json:"disabled_vms"`
	ActiveJobs       int                           `json:"active_jobs"`
	LastExecution    *time.Time                    `json:"last_execution,omitempty"`
	CreatedBy        string                        `json:"created_by"`
	CreatedAt        time.Time                     `json:"created_at"`
	UpdatedAt        time.Time                     `json:"updated_at"`
	Schedule         *database.ReplicationSchedule `json:"schedule,omitempty"`
	Memberships      []database.VMGroupMembership  `json:"memberships,omitempty"`
}

// GroupListResponse represents a list of machine groups
type GroupListResponse struct {
	Groups      []GroupResponse `json:"groups"`
	TotalCount  int             `json:"total_count"`
	FilteredBy  *string         `json:"filtered_by,omitempty"`
	RetrievedAt time.Time       `json:"retrieved_at"`
}

// CreateGroup creates a new machine group
// POST /api/v1/machine-groups
func (h *MachineGroupManagementHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var request CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx := context.Background()
	ctx, jobID, _ := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "create-group",
		Owner:     &[]string{"api"}[0],
	})

	logger := h.tracker.Logger(ctx)

	var group *database.VMMachineGroup

	err := h.tracker.RunStep(ctx, jobID, "create-group", func(ctx context.Context) error {
		logger.Info("Creating machine group", "name", request.Name, "schedule_id", request.ScheduleID)

		// Validate schedule exists if provided
		if request.ScheduleID != nil {
			schedule, err := h.schedulerRepo.GetScheduleByID(*request.ScheduleID)
			if err != nil {
				return fmt.Errorf("failed to validate schedule: %w", err)
			}
			if schedule == nil {
				return fmt.Errorf("schedule not found: %s", *request.ScheduleID)
			}
		}

		// Convert API request to service request
		serviceReq := &services.GroupCreateRequest{
			Name:             request.Name,
			Description:      request.Description,
			ScheduleID:       request.ScheduleID,
			MaxConcurrentVMs: request.MaxConcurrentVMs,
			Priority:         request.Priority,
			CreatedBy:        request.CreatedBy,
		}

		createdGroup, err := h.machineGroupService.CreateGroup(ctx, serviceReq)
		if err != nil {
			return fmt.Errorf("failed to create group: %w", err)
		}

		group = createdGroup
		logger.Info("Group created successfully", "group_id", group.ID, "name", group.Name)
		return nil
	})

	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create group: "+err.Error())
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	// Return created group
	response := h.convertToGroupResponse(group, nil)
	h.writeJSONResponse(w, http.StatusCreated, response)
}

// ListGroups lists all machine groups with optional filtering
// GET /api/v1/machine-groups
func (h *MachineGroupManagementHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	scheduleID := r.URL.Query().Get("schedule_id")
	var scheduleFilter *string
	if scheduleID != "" {
		scheduleFilter = &scheduleID
	}

	ctx := context.Background()
	ctx, jobID, _ := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "list-groups",
		Owner:     &[]string{"api"}[0],
	})

	logger := h.tracker.Logger(ctx)

	var listResponse *GroupListResponse

	err := h.tracker.RunStep(ctx, jobID, "list-groups", func(ctx context.Context) error {
		logger.Info("Listing machine groups", "schedule_filter", scheduleFilter)

		summaries, err := h.machineGroupService.ListGroups(ctx, scheduleFilter)
		if err != nil {
			return fmt.Errorf("failed to list groups: %w", err)
		}

		// Convert summaries to response format
		groupResponses := make([]GroupResponse, len(summaries))
		for i, summary := range summaries {
			groupResponses[i] = h.convertToGroupResponse(summary.Group, summary)
		}

		listResponse = &GroupListResponse{
			Groups:      groupResponses,
			TotalCount:  len(groupResponses),
			FilteredBy:  scheduleFilter,
			RetrievedAt: time.Now().UTC(),
		}

		logger.Info("Groups listed successfully", "count", len(groupResponses))
		return nil
	})

	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list groups: "+err.Error())
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	h.writeJSONResponse(w, http.StatusOK, listResponse)
}

// GetGroup retrieves a specific machine group with full details
// GET /api/v1/machine-groups/{id}
func (h *MachineGroupManagementHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	if groupID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID is required")
		return
	}

	ctx := context.Background()
	ctx, jobID, _ := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "get-group",
		Owner:     &[]string{"api"}[0],
	})

	logger := h.tracker.Logger(ctx)

	var groupResponse *GroupResponse

	err := h.tracker.RunStep(ctx, jobID, "get-group", func(ctx context.Context) error {
		logger.Info("Retrieving machine group", "group_id", groupID)

		summary, err := h.machineGroupService.GetGroup(ctx, groupID)
		if err != nil {
			return fmt.Errorf("failed to get group: %w", err)
		}
		if summary == nil || summary.Group == nil {
			return fmt.Errorf("group not found")
		}

		response := h.convertToGroupResponse(summary.Group, summary)
		groupResponse = &response

		logger.Info("Group retrieved successfully", "group_id", groupID, "name", summary.Group.Name)
		return nil
	})

	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		if err.Error() == "group not found" {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get group: "+err.Error())
		}
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	h.writeJSONResponse(w, http.StatusOK, groupResponse)
}

// UpdateGroup updates an existing machine group
// PUT /api/v1/machine-groups/{id}
func (h *MachineGroupManagementHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	if groupID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID is required")
		return
	}

	var request UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx := context.Background()
	ctx, jobID, _ := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "update-group",
		Owner:     &[]string{"api"}[0],
	})

	logger := h.tracker.Logger(ctx)

	var updatedGroup *database.VMMachineGroup

	err := h.tracker.RunStep(ctx, jobID, "update-group", func(ctx context.Context) error {
		logger.Info("Updating machine group", "group_id", groupID)

		// Validate schedule exists if provided
		if request.ScheduleID != nil {
			schedule, err := h.schedulerRepo.GetScheduleByID(*request.ScheduleID)
			if err != nil {
				return fmt.Errorf("failed to validate schedule: %w", err)
			}
			if schedule == nil {
				return fmt.Errorf("schedule not found: %s", *request.ScheduleID)
			}
		}

		// Convert API request to service request
		serviceReq := &services.GroupUpdateRequest{
			Name:             request.Name,
			Description:      request.Description,
			ScheduleID:       request.ScheduleID,
			MaxConcurrentVMs: request.MaxConcurrentVMs,
			Priority:         request.Priority,
		}

		group, err := h.machineGroupService.UpdateGroup(ctx, groupID, serviceReq)
		if err != nil {
			return fmt.Errorf("failed to update group: %w", err)
		}

		updatedGroup = group
		logger.Info("Group updated successfully", "group_id", groupID, "name", group.Name)
		return nil
	})

	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		if err.Error() == "group not found" {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update group: "+err.Error())
		}
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	// Return updated group
	response := h.convertToGroupResponse(updatedGroup, nil)
	h.writeJSONResponse(w, http.StatusOK, response)
}

// DeleteGroup deletes a machine group
// DELETE /api/v1/machine-groups/{id}
func (h *MachineGroupManagementHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	if groupID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID is required")
		return
	}

	ctx := context.Background()
	ctx, jobID, _ := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "delete-group",
		Owner:     &[]string{"api"}[0],
	})

	logger := h.tracker.Logger(ctx)

	err := h.tracker.RunStep(ctx, jobID, "delete-group", func(ctx context.Context) error {
		logger.Info("Deleting machine group", "group_id", groupID)

		// Get group details before deletion for logging
		summary, err := h.machineGroupService.GetGroup(ctx, groupID)
		if err != nil {
			return fmt.Errorf("failed to get group for deletion: %w", err)
		}
		if summary == nil || summary.Group == nil {
			return fmt.Errorf("group not found")
		}

		// Check if group has VMs
		if summary.TotalVMs > 0 {
			return fmt.Errorf("cannot delete group with %d VMs, remove VMs first", summary.TotalVMs)
		}

		if err := h.machineGroupService.DeleteGroup(ctx, groupID); err != nil {
			return fmt.Errorf("failed to delete group: %w", err)
		}

		logger.Info("Group deleted successfully", "group_id", groupID, "name", summary.Group.Name)
		return nil
	})

	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		if err.Error() == "group not found" {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else if err.Error() != "" && err.Error()[0:6] == "cannot" {
			h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete group: "+err.Error())
		}
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	// Return success response
	response := map[string]interface{}{
		"success":   true,
		"message":   "Group deleted successfully",
		"group_id":  groupID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// convertToGroupResponse converts a database group and optional summary to API response format
func (h *MachineGroupManagementHandler) convertToGroupResponse(group *database.VMMachineGroup, summary *services.GroupSummary) GroupResponse {
	response := GroupResponse{
		ID:               group.ID,
		Name:             group.Name,
		Description:      group.Description,
		ScheduleID:       group.ScheduleID,
		MaxConcurrentVMs: group.MaxConcurrentVMs,
		Priority:         group.Priority,
		CreatedBy:        group.CreatedBy,
		CreatedAt:        group.CreatedAt,
		UpdatedAt:        group.UpdatedAt,
	}

	// Add schedule name if schedule is loaded
	if group.Schedule != nil {
		response.ScheduleName = &group.Schedule.Name
		response.Schedule = group.Schedule
	}

	// Add summary statistics if provided
	if summary != nil {
		response.TotalVMs = summary.TotalVMs
		response.EnabledVMs = summary.EnabledVMs
		response.DisabledVMs = summary.DisabledVMs
		response.ActiveJobs = summary.ActiveJobs
		response.LastExecution = summary.LastExecution
		response.Memberships = summary.Memberships
	}

	return response
}

// writeJSONResponse writes a JSON response
func (h *MachineGroupManagementHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to write JSON response")
	}
}

// writeErrorResponse writes an error response
func (h *MachineGroupManagementHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}
