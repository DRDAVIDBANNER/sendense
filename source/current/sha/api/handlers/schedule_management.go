// Package handlers provides REST API endpoints for schedule management
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/services"
)

// ScheduleManagementHandler handles schedule CRUD API endpoints
type ScheduleManagementHandler struct {
	schedulerRepo    *database.SchedulerRepository
	schedulerService *services.SchedulerService
	tracker          *joblog.Tracker
}

// NewScheduleManagementHandler creates a new schedule management handler
func NewScheduleManagementHandler(schedulerRepo *database.SchedulerRepository, schedulerService *services.SchedulerService, tracker *joblog.Tracker) *ScheduleManagementHandler {
	return &ScheduleManagementHandler{
		schedulerRepo:    schedulerRepo,
		schedulerService: schedulerService,
		tracker:          tracker,
	}
}

// CreateScheduleRequest represents a request to create a new replication schedule
type CreateScheduleRequest struct {
	Name              string  `json:"name" binding:"required,min=1,max=255"`
	Description       *string `json:"description,omitempty"`
	CronExpression    string  `json:"cron_expression" binding:"required"`
	Timezone          string  `json:"timezone" binding:"required"`
	SkipIfRunning     bool    `json:"skip_if_running"`
	MaxConcurrentJobs int     `json:"max_concurrent_jobs" binding:"min=1,max=100"`
	ReplicationType   string  `json:"replication_type,omitempty"`
	RetryAttempts     int     `json:"retry_attempts" binding:"min=0"`
	RetryDelayMinutes int     `json:"retry_delay_minutes" binding:"min=0"`
	Enabled           bool    `json:"enabled"`
	CreatedBy         string  `json:"created_by,omitempty"`
}

// UpdateScheduleRequest represents a request to update an existing schedule
type UpdateScheduleRequest struct {
	Name              *string `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description       *string `json:"description,omitempty"`
	CronExpression    *string `json:"cron_expression,omitempty"`
	Timezone          *string `json:"timezone,omitempty"`
	SkipIfRunning     *bool   `json:"skip_if_running,omitempty"`
	MaxConcurrentJobs *int    `json:"max_concurrent_jobs,omitempty" binding:"omitempty,min=1,max=100"`
	ReplicationType   *string `json:"replication_type,omitempty"`
	RetryAttempts     *int    `json:"retry_attempts,omitempty" binding:"omitempty,min=0"`
	RetryDelayMinutes *int    `json:"retry_delay_minutes,omitempty" binding:"omitempty,min=0"`
	Enabled           *bool   `json:"enabled,omitempty"`
}

// ScheduleResponse represents a schedule in API responses
type ScheduleResponse struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       *string   `json:"description"`
	CronExpression    string    `json:"cron_expression"`
	Timezone          string    `json:"timezone"`
	SkipIfRunning     bool      `json:"skip_if_running"`
	MaxConcurrentJobs int       `json:"max_concurrent_jobs"`
	ReplicationType   string    `json:"replication_type"`
	RetryAttempts     int       `json:"retry_attempts"`
	RetryDelayMinutes int       `json:"retry_delay_minutes"`
	Enabled           bool      `json:"enabled"`
	CreatedBy         string    `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ScheduleListResponse represents a list of schedules with metadata
type ScheduleListResponse struct {
	Schedules     []ScheduleResponse `json:"schedules"`
	TotalCount    int                `json:"total_count"`
	EnabledCount  int                `json:"enabled_count"`
	DisabledCount int                `json:"disabled_count"`
	RetrievedAt   time.Time          `json:"retrieved_at"`
}

// ScheduleStatsResponse represents schedule statistics
type ScheduleStatsResponse struct {
	ScheduleID      string                 `json:"schedule_id"`
	ScheduleName    string                 `json:"schedule_name"`
	TotalExecutions int                    `json:"total_executions"`
	SuccessfulRuns  int                    `json:"successful_runs"`
	FailedRuns      int                    `json:"failed_runs"`
	AverageRuntime  time.Duration          `json:"average_runtime"`
	LastExecution   *time.Time             `json:"last_execution,omitempty"`
	NextExecution   *time.Time             `json:"next_execution,omitempty"`
	AdditionalStats map[string]interface{} `json:"additional_stats,omitempty"`
}

// EnableScheduleRequest represents a request to enable/disable a schedule
type EnableScheduleRequest struct {
	Enabled bool `json:"enabled"`
}

// TriggerScheduleRequest represents a request to manually trigger a schedule
type TriggerScheduleRequest struct {
	Reason    *string `json:"reason,omitempty"`
	CreatedBy string  `json:"created_by,omitempty"`
}

// TriggerScheduleResponse represents the response from manually triggering a schedule
type TriggerScheduleResponse struct {
	ExecutionID  string                 `json:"execution_id"`
	ScheduleID   string                 `json:"schedule_id"`
	ScheduleName string                 `json:"schedule_name"`
	Status       string                 `json:"status"`
	TriggeredAt  time.Time              `json:"triggered_at"`
	TriggeredBy  string                 `json:"triggered_by"`
	Reason       *string                `json:"reason,omitempty"`
	JobsCreated  int                    `json:"jobs_created"`
	VMsProcessed int                    `json:"vms_processed"`
	Summary      map[string]interface{} `json:"summary,omitempty"`
}

// ScheduleExecutionResponse represents a schedule execution with details
type ScheduleExecutionResponse struct {
	ExecutionID    string                 `json:"execution_id"`
	ScheduleID     string                 `json:"schedule_id"`
	ScheduleName   string                 `json:"schedule_name"`
	Status         string                 `json:"status"`
	ScheduledAt    time.Time              `json:"scheduled_at"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Duration       *int                   `json:"duration_seconds,omitempty"`
	JobsCreated    int                    `json:"jobs_created"`
	JobsSuccessful int                    `json:"jobs_successful"`
	JobsFailed     int                    `json:"jobs_failed"`
	VMsProcessed   int                    `json:"vms_processed"`
	ErrorMessage   *string                `json:"error_message,omitempty"`
	ExecutionType  string                 `json:"execution_type"` // "scheduled" or "manual"
	TriggeredBy    *string                `json:"triggered_by,omitempty"`
	Summary        map[string]interface{} `json:"summary,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ScheduleExecutionListResponse represents a list of schedule executions
type ScheduleExecutionListResponse struct {
	Executions  []ScheduleExecutionResponse `json:"executions"`
	TotalCount  int                         `json:"total_count"`
	Page        int                         `json:"page"`
	PageSize    int                         `json:"page_size"`
	HasMore     bool                        `json:"has_more"`
	RetrievedAt time.Time                   `json:"retrieved_at"`
}

// CreateSchedule creates a new replication schedule
// POST /api/v1/schedules
func (h *ScheduleManagementHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var request CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	log.WithFields(log.Fields{
		"name":                request.Name,
		"cron_expression":     request.CronExpression,
		"timezone":            request.Timezone,
		"max_concurrent_jobs": request.MaxConcurrentJobs,
		"enabled":             request.Enabled,
	}).Info("Creating new replication schedule")

	ctx := r.Context()

	// Start job tracking
	ctx, jobID, err := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "schedule_management",
		Operation: "create-schedule",
		Owner:     stringPtr("api"),
	})
	if err != nil {
		log.WithError(err).Error("Failed to start schedule creation job")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to start schedule creation")
		return
	}

	var createdSchedule *database.ReplicationSchedule

	err = h.tracker.RunStep(ctx, jobID, "create-schedule", func(ctx context.Context) error {
		log := h.tracker.Logger(ctx)

		// Validate cron expression format (basic validation)
		if err := h.validateCronExpression(request.CronExpression); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}

		// Validate timezone
		if err := h.validateTimezone(request.Timezone); err != nil {
			return fmt.Errorf("invalid timezone: %w", err)
		}

		// Create schedule object
		schedule := &database.ReplicationSchedule{
			Name:              request.Name,
			Description:       request.Description,
			CronExpression:    request.CronExpression,
			ScheduleType:      "cron",
			Timezone:          request.Timezone,
			SkipIfRunning:     request.SkipIfRunning,
			MaxConcurrentJobs: request.MaxConcurrentJobs,
			ReplicationType:   request.ReplicationType,
			RetryAttempts:     request.RetryAttempts,
			RetryDelayMinutes: request.RetryDelayMinutes,
			Enabled:           request.Enabled,
			CreatedBy:         request.CreatedBy,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if schedule.CreatedBy == "" {
			schedule.CreatedBy = "api"
		}

		// Create in database
		if err := h.schedulerRepo.CreateSchedule(schedule); err != nil {
			return fmt.Errorf("failed to create schedule in database: %w", err)
		}

		createdSchedule = schedule

		log.Info("Schedule created successfully",
			"schedule_id", schedule.ID,
			"name", schedule.Name,
			"enabled", schedule.Enabled)

		return nil
	})

	// End job tracking
	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		log.WithError(err).Error("Failed to create schedule")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create schedule: "+err.Error())
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	// ✅ DYNAMIC RELOAD: Refresh scheduler after schedule creation
	if createdSchedule.Enabled {
		if err := h.schedulerService.ReloadSchedules(ctx); err != nil {
			log.WithError(err).Warn("Failed to reload schedules after creation, schedule may not be active until restart")
		} else {
			log.WithField("schedule_id", createdSchedule.ID).Info("Scheduler reloaded successfully after schedule creation")
		}
	}

	// Convert to response format
	response := h.convertToScheduleResponse(createdSchedule)

	log.WithFields(log.Fields{
		"schedule_id": response.ID,
		"name":        response.Name,
		"enabled":     response.Enabled,
	}).Info("Schedule creation completed successfully")

	h.writeJSONResponse(w, http.StatusCreated, response)
}

// ListSchedules retrieves all replication schedules
// GET /api/v1/schedules?enabled_only=true
func (h *ScheduleManagementHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	enabledOnlyParam := r.URL.Query().Get("enabled_only")
	enabledOnly := enabledOnlyParam == "true"

	log.WithField("enabled_only", enabledOnly).Info("Listing replication schedules")

	schedules, err := h.schedulerRepo.ListSchedules(enabledOnly)
	if err != nil {
		log.WithError(err).Error("Failed to list schedules")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve schedules: "+err.Error())
		return
	}

	// Convert to response format
	scheduleResponses := make([]ScheduleResponse, 0, len(schedules))
	enabledCount := 0
	disabledCount := 0

	for _, schedule := range schedules {
		scheduleResponse := h.convertToScheduleResponse(&schedule)
		scheduleResponses = append(scheduleResponses, scheduleResponse)

		if schedule.Enabled {
			enabledCount++
		} else {
			disabledCount++
		}
	}

	response := ScheduleListResponse{
		Schedules:     scheduleResponses,
		TotalCount:    len(schedules),
		EnabledCount:  enabledCount,
		DisabledCount: disabledCount,
		RetrievedAt:   time.Now(),
	}

	log.WithFields(log.Fields{
		"total_count":    response.TotalCount,
		"enabled_count":  response.EnabledCount,
		"disabled_count": response.DisabledCount,
	}).Info("Schedule list retrieved successfully")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetScheduleByID retrieves a specific replication schedule by ID
// GET /api/v1/schedules/{id}
func (h *ScheduleManagementHandler) GetScheduleByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	if scheduleID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Schedule ID is required")
		return
	}

	log.WithField("schedule_id", scheduleID).Info("Getting schedule details")

	schedule, err := h.schedulerRepo.GetScheduleByID(scheduleID)
	if err != nil {
		log.WithError(err).Error("Failed to get schedule")
		h.writeErrorResponse(w, http.StatusNotFound, "Schedule not found: "+err.Error())
		return
	}

	response := h.convertToScheduleResponse(schedule)

	log.WithFields(log.Fields{
		"schedule_id": response.ID,
		"name":        response.Name,
		"enabled":     response.Enabled,
	}).Info("Schedule details retrieved successfully")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// UpdateSchedule updates an existing replication schedule
// PUT /api/v1/schedules/{id}
func (h *ScheduleManagementHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	if scheduleID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Schedule ID is required")
		return
	}

	var request UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	log.WithField("schedule_id", scheduleID).Info("Updating replication schedule")

	ctx := r.Context()

	// Start job tracking
	ctx, jobID, err := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "schedule_management",
		Operation: "update-schedule",
		Owner:     stringPtr("api"),
	})
	if err != nil {
		log.WithError(err).Error("Failed to start schedule update job")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to start schedule update")
		return
	}

	var updatedSchedule *database.ReplicationSchedule

	err = h.tracker.RunStep(ctx, jobID, "update-schedule", func(ctx context.Context) error {
		log := h.tracker.Logger(ctx)

		// Verify schedule exists
		_, err := h.schedulerRepo.GetScheduleByID(scheduleID)
		if err != nil {
			return fmt.Errorf("schedule not found: %w", err)
		}

		// Prepare updates map
		updates := make(map[string]interface{})

		if request.Name != nil {
			updates["name"] = *request.Name
		}
		if request.Description != nil {
			updates["description"] = *request.Description
		}
		if request.CronExpression != nil {
			if err := h.validateCronExpression(*request.CronExpression); err != nil {
				return fmt.Errorf("invalid cron expression: %w", err)
			}
			updates["cron_expression"] = *request.CronExpression
		}
		if request.Timezone != nil {
			if err := h.validateTimezone(*request.Timezone); err != nil {
				return fmt.Errorf("invalid timezone: %w", err)
			}
			updates["timezone"] = *request.Timezone
		}
		if request.SkipIfRunning != nil {
			updates["skip_if_running"] = *request.SkipIfRunning
		}
		if request.MaxConcurrentJobs != nil {
			updates["max_concurrent_jobs"] = *request.MaxConcurrentJobs
		}
		if request.ReplicationType != nil {
			updates["replication_type"] = *request.ReplicationType
		}
		if request.RetryAttempts != nil {
			updates["retry_attempts"] = *request.RetryAttempts
		}
		if request.RetryDelayMinutes != nil {
			updates["retry_delay_minutes"] = *request.RetryDelayMinutes
		}
		if request.Enabled != nil {
			updates["enabled"] = *request.Enabled
		}

		// Always update the updated_at timestamp
		updates["updated_at"] = time.Now()

		if len(updates) == 1 { // Only updated_at was added
			return fmt.Errorf("no fields to update")
		}

		// Update in database
		if err := h.schedulerRepo.UpdateSchedule(scheduleID, updates); err != nil {
			return fmt.Errorf("failed to update schedule: %w", err)
		}

		// Get updated schedule
		updatedSchedule, err = h.schedulerRepo.GetScheduleByID(scheduleID)
		if err != nil {
			return fmt.Errorf("failed to retrieve updated schedule: %w", err)
		}

		log.Info("Schedule updated successfully",
			"schedule_id", scheduleID,
			"updated_fields", len(updates)-1) // Subtract 1 for updated_at

		return nil
	})

	// End job tracking
	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		log.WithError(err).Error("Failed to update schedule")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update schedule: "+err.Error())
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	response := h.convertToScheduleResponse(updatedSchedule)

	// ✅ DYNAMIC RELOAD: Refresh scheduler after schedule update
	if err := h.schedulerService.ReloadSchedules(ctx); err != nil {
		log.WithError(err).Warn("Failed to reload schedules after update, changes may not be active until restart")
	} else {
		log.WithField("schedule_id", scheduleID).Info("Scheduler reloaded successfully after schedule update")
	}

	log.WithFields(log.Fields{
		"schedule_id": response.ID,
		"name":        response.Name,
		"enabled":     response.Enabled,
	}).Info("Schedule update completed successfully")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// DeleteSchedule removes a replication schedule
// DELETE /api/v1/schedules/{id}
func (h *ScheduleManagementHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	if scheduleID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Schedule ID is required")
		return
	}

	log.WithField("schedule_id", scheduleID).Info("Deleting replication schedule")

	ctx := r.Context()

	// Start job tracking
	ctx, jobID, err := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "schedule_management",
		Operation: "delete-schedule",
		Owner:     stringPtr("api"),
	})
	if err != nil {
		log.WithError(err).Error("Failed to start schedule deletion job")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to start schedule deletion")
		return
	}

	err = h.tracker.RunStep(ctx, jobID, "delete-schedule", func(ctx context.Context) error {
		log := h.tracker.Logger(ctx)

		// Verify schedule exists
		existingSchedule, err := h.schedulerRepo.GetScheduleByID(scheduleID)
		if err != nil {
			return fmt.Errorf("schedule not found: %w", err)
		}

		// Check if schedule has associated machine groups
		groups, err := h.schedulerRepo.ListGroups(&scheduleID)
		if err != nil {
			return fmt.Errorf("failed to check associated groups: %w", err)
		}

		if len(groups) > 0 {
			return fmt.Errorf("cannot delete schedule: %d machine groups are still using this schedule", len(groups))
		}

		// Delete the schedule
		if err := h.schedulerRepo.DeleteSchedule(scheduleID); err != nil {
			return fmt.Errorf("failed to delete schedule: %w", err)
		}

		log.Info("Schedule deleted successfully",
			"schedule_id", scheduleID,
			"name", existingSchedule.Name)

		return nil
	})

	// End job tracking
	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		log.WithError(err).Error("Failed to delete schedule")

		// Return appropriate status based on error type
		if strings.Contains(err.Error(), "not found") {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else if strings.Contains(err.Error(), "cannot delete") {
			h.writeErrorResponse(w, http.StatusConflict, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete schedule: "+err.Error())
		}
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	// ✅ DYNAMIC RELOAD: Refresh scheduler after schedule deletion
	if err := h.schedulerService.ReloadSchedules(ctx); err != nil {
		log.WithError(err).Warn("Failed to reload schedules after deletion, schedule may remain active until restart")
	} else {
		log.WithField("schedule_id", scheduleID).Info("Scheduler reloaded successfully after schedule deletion")
	}

	response := map[string]interface{}{
		"success":    true,
		"message":    "Schedule deleted successfully",
		"deleted_at": time.Now().UTC().Format(time.RFC3339),
	}

	log.WithField("schedule_id", scheduleID).Info("Schedule deletion completed successfully")
	h.writeJSONResponse(w, http.StatusOK, response)
}

// convertToScheduleResponse converts a database schedule to API response format
func (h *ScheduleManagementHandler) convertToScheduleResponse(schedule *database.ReplicationSchedule) ScheduleResponse {
	return ScheduleResponse{
		ID:                schedule.ID,
		Name:              schedule.Name,
		Description:       schedule.Description,
		CronExpression:    schedule.CronExpression,
		Timezone:          schedule.Timezone,
		SkipIfRunning:     schedule.SkipIfRunning,
		MaxConcurrentJobs: schedule.MaxConcurrentJobs,
		ReplicationType:   schedule.ReplicationType,
		RetryAttempts:     schedule.RetryAttempts,
		RetryDelayMinutes: schedule.RetryDelayMinutes,
		Enabled:           schedule.Enabled,
		CreatedBy:         schedule.CreatedBy,
		CreatedAt:         schedule.CreatedAt,
		UpdatedAt:         schedule.UpdatedAt,
	}
}

// validateCronExpression performs basic validation of cron expression format
func (h *ScheduleManagementHandler) validateCronExpression(cronExpr string) error {
	// Basic validation - ensure we have the expected number of fields
	// Cron expressions can be 5 or 6 fields (with optional seconds)
	fields := strings.Fields(cronExpr)
	if len(fields) != 5 && len(fields) != 6 {
		return fmt.Errorf("cron expression must have 5 or 6 fields, got %d", len(fields))
	}

	// Additional validation could be added here (e.g., parsing with cron library)
	return nil
}

// validateTimezone validates that the timezone string is valid
func (h *ScheduleManagementHandler) validateTimezone(tz string) error {
	_, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}
	return nil
}

// writeJSONResponse writes a JSON response
func (h *ScheduleManagementHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to write JSON response")
	}
}

// writeErrorResponse writes an error response
func (h *ScheduleManagementHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}

// EnableSchedule enables or disables a replication schedule
// POST /api/v1/schedules/{id}/enable
func (h *ScheduleManagementHandler) EnableSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	if scheduleID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Schedule ID is required")
		return
	}

	var request EnableScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx := context.Background()
	ctx, jobID, _ := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "schedule_control",
		Operation: "enable_schedule",
		Owner:     stringPtr("api"),
	})

	logger := h.tracker.Logger(ctx)

	err := h.tracker.RunStep(ctx, jobID, "enable-schedule", func(ctx context.Context) error {
		logger.Info("Enabling/disabling schedule", "schedule_id", scheduleID, "enabled", request.Enabled)

		// Check if schedule exists
		schedule, err := h.schedulerRepo.GetScheduleByID(scheduleID)
		if err != nil {
			return fmt.Errorf("failed to get schedule: %w", err)
		}
		if schedule == nil {
			return fmt.Errorf("schedule not found")
		}

		// Update schedule enabled status
		updates := map[string]interface{}{
			"enabled":    request.Enabled,
			"updated_at": time.Now().UTC(),
		}

		if err := h.schedulerRepo.UpdateSchedule(scheduleID, updates); err != nil {
			return fmt.Errorf("failed to update schedule: %w", err)
		}

		action := "disabled"
		if request.Enabled {
			action = "enabled"
		}
		logger.Info("Schedule successfully "+action, "schedule_id", scheduleID, "schedule_name", schedule.Name)

		return nil
	})

	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update schedule: "+err.Error())
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	// ✅ DYNAMIC RELOAD: Refresh scheduler after enable/disable
	if err := h.schedulerService.ReloadSchedules(ctx); err != nil {
		log.WithError(err).Warn("Failed to reload schedules after enable/disable, changes may not be active until restart")
	} else {
		log.WithField("schedule_id", scheduleID).Info("Scheduler reloaded successfully after enable/disable")
	}

	// Return updated schedule
	updatedSchedule, err := h.schedulerRepo.GetScheduleByID(scheduleID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve updated schedule: "+err.Error())
		return
	}

	response := h.convertToScheduleResponse(updatedSchedule)
	h.writeJSONResponse(w, http.StatusOK, response)
}

// TriggerSchedule manually triggers a replication schedule execution
// POST /api/v1/schedules/{id}/trigger
func (h *ScheduleManagementHandler) TriggerSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	if scheduleID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Schedule ID is required")
		return
	}

	var request TriggerScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx := context.Background()
	ctx, jobID, _ := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "schedule_control",
		Operation: "trigger_schedule",
		Owner:     stringPtr("api"),
	})

	logger := h.tracker.Logger(ctx)

	var triggerResponse *TriggerScheduleResponse

	err := h.tracker.RunStep(ctx, jobID, "trigger-schedule", func(ctx context.Context) error {
		logger.Info("Manually triggering schedule", "schedule_id", scheduleID, "triggered_by", request.CreatedBy)

		// Check if schedule exists and is enabled
		schedule, err := h.schedulerRepo.GetScheduleByID(scheduleID)
		if err != nil {
			return fmt.Errorf("failed to get schedule: %w", err)
		}
		if schedule == nil {
			return fmt.Errorf("schedule not found")
		}
		if !schedule.Enabled {
			return fmt.Errorf("cannot trigger disabled schedule")
		}

		// Trigger manual execution via scheduler service
		executionSummary, err := h.schedulerService.TriggerManualExecution(ctx, scheduleID, request.CreatedBy, request.Reason)
		if err != nil {
			return fmt.Errorf("failed to trigger schedule execution: %w", err)
		}

		triggerResponse = &TriggerScheduleResponse{
			ExecutionID:  executionSummary.ExecutionID,
			ScheduleID:   scheduleID,
			ScheduleName: schedule.Name,
			Status:       executionSummary.Status,
			TriggeredAt:  executionSummary.StartedAt,
			TriggeredBy:  request.CreatedBy,
			Reason:       request.Reason,
			JobsCreated:  executionSummary.JobsCreated,
			VMsProcessed: executionSummary.VMsProcessed,
			Summary:      executionSummary.Summary,
		}

		logger.Info("Schedule triggered successfully",
			"schedule_id", scheduleID,
			"execution_id", triggerResponse.ExecutionID,
			"jobs_created", triggerResponse.JobsCreated)

		return nil
	})

	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		if strings.Contains(err.Error(), "not found") {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else if strings.Contains(err.Error(), "disabled") {
			h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to trigger schedule: "+err.Error())
		}
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	h.writeJSONResponse(w, http.StatusOK, triggerResponse)
}

// GetScheduleExecutions retrieves execution history for a schedule
// GET /api/v1/schedules/{id}/executions
func (h *ScheduleManagementHandler) GetScheduleExecutions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	if scheduleID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Schedule ID is required")
		return
	}

	// Parse pagination parameters
	limit := 20 // default
	offset := 0 // default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			offset = (p - 1) * limit
		}
	}

	ctx := context.Background()
	ctx, jobID, _ := h.tracker.StartJob(ctx, joblog.JobStart{
		JobType:   "schedule_control",
		Operation: "get_executions",
		Owner:     stringPtr("api"),
	})

	logger := h.tracker.Logger(ctx)

	var listResponse *ScheduleExecutionListResponse

	err := h.tracker.RunStep(ctx, jobID, "get-executions", func(ctx context.Context) error {
		logger.Info("Retrieving schedule executions", "schedule_id", scheduleID, "limit", limit, "offset", offset)

		// Check if schedule exists
		schedule, err := h.schedulerRepo.GetScheduleByID(scheduleID)
		if err != nil {
			return fmt.Errorf("failed to get schedule: %w", err)
		}
		if schedule == nil {
			return fmt.Errorf("schedule not found")
		}

		// Get executions with pagination
		executions, err := h.schedulerRepo.GetScheduleExecutions(scheduleID, limit+1, offset) // +1 to check if there are more
		if err != nil {
			return fmt.Errorf("failed to get schedule executions: %w", err)
		}

		hasMore := len(executions) > limit
		if hasMore {
			executions = executions[:limit] // Remove the extra one
		}

		// Convert to response format
		executionResponses := make([]ScheduleExecutionResponse, len(executions))
		for i, exec := range executions {
			var duration *int
			if exec.ExecutionDurationSeconds != nil {
				duration = exec.ExecutionDurationSeconds
			} else if exec.StartedAt != nil && exec.CompletedAt != nil {
				d := int(exec.CompletedAt.Sub(*exec.StartedAt).Seconds())
				duration = &d
			}

			// Determine execution type from triggered_by field
			executionType := "scheduled"
			if exec.TriggeredBy != "scheduler" {
				executionType = "manual"
			}

			// Parse execution details as summary if available
			var summary map[string]interface{}
			if exec.ExecutionDetails != nil {
				// Try to parse JSON, fallback to simple map if not valid JSON
				if err := json.Unmarshal([]byte(*exec.ExecutionDetails), &summary); err != nil {
					summary = map[string]interface{}{"details": *exec.ExecutionDetails}
				}
			}

			executionResponses[i] = ScheduleExecutionResponse{
				ExecutionID:    exec.ID,
				ScheduleID:     exec.ScheduleID,
				ScheduleName:   schedule.Name,
				Status:         exec.Status,
				ScheduledAt:    exec.ScheduledAt,
				StartedAt:      exec.StartedAt,
				CompletedAt:    exec.CompletedAt,
				Duration:       duration,
				JobsCreated:    exec.JobsCreated,
				JobsSuccessful: exec.JobsCompleted,
				JobsFailed:     exec.JobsFailed,
				VMsProcessed:   exec.VMsEligible, // Using VMsEligible as closest equivalent
				ErrorMessage:   exec.ErrorMessage,
				ExecutionType:  executionType,
				TriggeredBy:    &exec.TriggeredBy,
				Summary:        summary,
				CreatedAt:      exec.CreatedAt,
				UpdatedAt:      exec.CreatedAt, // No UpdatedAt field in model, use CreatedAt
			}
		}

		page := (offset / limit) + 1
		listResponse = &ScheduleExecutionListResponse{
			Executions:  executionResponses,
			TotalCount:  len(executionResponses), // Note: This would need a separate count query for true total
			Page:        page,
			PageSize:    limit,
			HasMore:     hasMore,
			RetrievedAt: time.Now().UTC(),
		}

		logger.Info("Schedule executions retrieved successfully",
			"schedule_id", scheduleID,
			"count", len(executionResponses),
			"page", page)

		return nil
	})

	if err != nil {
		h.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		if strings.Contains(err.Error(), "not found") {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get schedule executions: "+err.Error())
		}
		return
	}

	h.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	h.writeJSONResponse(w, http.StatusOK, listResponse)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
