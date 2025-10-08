package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
)

// =============================================================================
// MACHINE GROUP SERVICE - VM Group Management using context_id
// =============================================================================
// CRITICAL: All VM operations use vm_replication_contexts.context_id
// Provides comprehensive group CRUD, membership management, and schedule assignment

// MachineGroupService handles machine group operations and VM membership
type MachineGroupService struct {
	schedulerRepo *database.SchedulerRepository
	jobTracker    *joblog.Tracker
}

// GroupCreateRequest represents a request to create a new machine group
type GroupCreateRequest struct {
	Name             string  `json:"name" validate:"required,min=1,max=255"`
	Description      *string `json:"description,omitempty"`
	ScheduleID       *string `json:"schedule_id,omitempty"`
	MaxConcurrentVMs int     `json:"max_concurrent_vms" validate:"min=1,max=100"`
	Priority         int     `json:"priority" validate:"min=0"`
	CreatedBy        string  `json:"created_by,omitempty"`
}

// GroupUpdateRequest represents a request to update an existing machine group
type GroupUpdateRequest struct {
	Name             *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description      *string `json:"description,omitempty"`
	ScheduleID       *string `json:"schedule_id,omitempty"`
	MaxConcurrentVMs *int    `json:"max_concurrent_vms,omitempty" validate:"omitempty,min=1,max=100"`
	Priority         *int    `json:"priority,omitempty" validate:"omitempty,min=0"`
}

// VMMembershipRequest represents a request to manage VM group membership
type VMMembershipRequest struct {
	VMContextID string `json:"vm_context_id" validate:"required"`
	Priority    int    `json:"priority" validate:"min=0"`
	Enabled     bool   `json:"enabled"`
}

// BulkMembershipRequest represents a request for bulk VM membership operations
type BulkMembershipRequest struct {
	VMContextIDs []string `json:"vm_context_ids" validate:"required,min=1"`
	Priority     int      `json:"priority" validate:"min=0"`
	Enabled      bool     `json:"enabled"`
}

// GroupSummary provides summary information about a machine group
type GroupSummary struct {
	Group           *database.VMMachineGroup    `json:"group"`
	TotalVMs        int                         `json:"total_vms"`
	EnabledVMs      int                         `json:"enabled_vms"`
	DisabledVMs     int                         `json:"disabled_vms"`
	ActiveJobs      int                         `json:"active_jobs"`
	LastExecution   *time.Time                  `json:"last_execution,omitempty"`
	Schedule        *database.ReplicationSchedule `json:"schedule,omitempty"`
	Memberships     []database.VMGroupMembership  `json:"memberships,omitempty"`
}

// BulkOperationResult provides results of bulk operations
type BulkOperationResult struct {
	TotalRequested int      `json:"total_requested"`
	Successful     int      `json:"successful"`
	Failed         int      `json:"failed"`
	SuccessfulIDs  []string `json:"successful_ids"`
	FailedIDs      []string `json:"failed_ids"`
	ErrorMessages  []string `json:"error_messages,omitempty"`
	Duration       time.Duration `json:"duration"`
}

// NewMachineGroupService creates a new machine group service
func NewMachineGroupService(
	schedulerRepo *database.SchedulerRepository,
	jobTracker *joblog.Tracker,
) *MachineGroupService {
	return &MachineGroupService{
		schedulerRepo: schedulerRepo,
		jobTracker:    jobTracker,
	}
}

// CreateGroup creates a new machine group
func (s *MachineGroupService) CreateGroup(ctx context.Context, req *GroupCreateRequest) (*database.VMMachineGroup, error) {
	// Start job tracking for group creation
	owner := "machine-group-service"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "create-group",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"group_name": req.Name,
			"schedule_id": req.ScheduleID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start group creation job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("🆕 Creating machine group",
		"name", req.Name,
		"schedule_id", req.ScheduleID,
		"max_concurrent", req.MaxConcurrentVMs,
	)

	var group *database.VMMachineGroup
	err = s.jobTracker.RunStep(ctx, jobID, "create-group", func(ctx context.Context) error {
		// Validate schedule exists if provided
		if req.ScheduleID != nil {
			_, err := s.schedulerRepo.GetScheduleByID(*req.ScheduleID)
			if err != nil {
				return fmt.Errorf("invalid schedule_id: %w", err)
			}
		}

		// Create the group
		group = &database.VMMachineGroup{
			Name:             req.Name,
			Description:      req.Description,
			ScheduleID:       req.ScheduleID,
			MaxConcurrentVMs: req.MaxConcurrentVMs,
			Priority:         req.Priority,
			CreatedBy:        req.CreatedBy,
		}

		if group.CreatedBy == "" {
			group.CreatedBy = "machine-group-service"
		}

		if err := s.schedulerRepo.CreateGroup(group); err != nil {
			return fmt.Errorf("failed to create group: %w", err)
		}

		logger.Info("Successfully created machine group",
			"group_id", group.ID,
			"name", group.Name,
		)

		return nil
	})

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return nil, err
	}

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return group, nil
}

// UpdateGroup updates an existing machine group
func (s *MachineGroupService) UpdateGroup(ctx context.Context, groupID string, req *GroupUpdateRequest) (*database.VMMachineGroup, error) {
	// Start job tracking for group update
	owner := "machine-group-service"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "update-group",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"group_id": groupID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start group update job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("📝 Updating machine group", "group_id", groupID)

	var updatedGroup *database.VMMachineGroup
	err = s.jobTracker.RunStep(ctx, jobID, "update-group", func(ctx context.Context) error {
		// Validate schedule exists if provided
		if req.ScheduleID != nil {
			_, err := s.schedulerRepo.GetScheduleByID(*req.ScheduleID)
			if err != nil {
				return fmt.Errorf("invalid schedule_id: %w", err)
			}
		}

		// Build updates map
		updates := make(map[string]interface{})
		if req.Name != nil {
			updates["name"] = *req.Name
		}
		if req.Description != nil {
			updates["description"] = *req.Description
		}
		if req.ScheduleID != nil {
			updates["schedule_id"] = *req.ScheduleID
		}
		if req.MaxConcurrentVMs != nil {
			updates["max_concurrent_vms"] = *req.MaxConcurrentVMs
		}
		if req.Priority != nil {
			updates["priority"] = *req.Priority
		}
		updates["updated_at"] = time.Now()

		if err := s.schedulerRepo.UpdateGroup(groupID, updates); err != nil {
			return fmt.Errorf("failed to update group: %w", err)
		}

		// Get updated group
		updatedGroup, err = s.schedulerRepo.GetGroupByID(groupID, "Schedule")
		if err != nil {
			return fmt.Errorf("failed to get updated group: %w", err)
		}

		logger.Info("Successfully updated machine group",
			"group_id", groupID,
			"updates", len(updates),
		)

		return nil
	})

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return nil, err
	}

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return updatedGroup, nil
}

// GetGroup retrieves a machine group with full details
func (s *MachineGroupService) GetGroup(ctx context.Context, groupID string) (*GroupSummary, error) {
	group, err := s.schedulerRepo.GetGroupByID(groupID, "Schedule", "Memberships", "Memberships.VMContext")
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Calculate statistics
	summary := &GroupSummary{
		Group:       group,
		TotalVMs:    len(group.Memberships),
		Schedule:    group.Schedule,
		Memberships: group.Memberships,
	}

	for _, membership := range group.Memberships {
		if membership.Enabled {
			summary.EnabledVMs++
		} else {
			summary.DisabledVMs++
		}
	}

	// Get last execution (if any)
	executions, err := s.schedulerRepo.GetGroupExecutions(groupID, 1) // Get latest execution
	if err == nil && len(executions) > 0 {
		summary.LastExecution = &executions[0].ScheduledAt
	}

	// TODO: Calculate active jobs count by querying replication jobs
	// This would require integration with ReplicationJobRepository

	return summary, nil
}

// ListGroups lists all machine groups with optional filtering
func (s *MachineGroupService) ListGroups(ctx context.Context, scheduleID *string) ([]*GroupSummary, error) {
	groups, err := s.schedulerRepo.ListGroups(scheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	summaries := make([]*GroupSummary, 0, len(groups))
	for _, group := range groups {
		summary, err := s.GetGroup(ctx, group.ID)
		if err != nil {
			// Log error but continue with other groups
			continue
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// DeleteGroup deletes a machine group and all its memberships
func (s *MachineGroupService) DeleteGroup(ctx context.Context, groupID string) error {
	// Start job tracking for group deletion
	owner := "machine-group-service"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "delete-group",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"group_id": groupID,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start group deletion job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("🗑️ Deleting machine group", "group_id", groupID)

	err = s.jobTracker.RunStep(ctx, jobID, "delete-group", func(ctx context.Context) error {
		// Check if group has active executions
		executions, err := s.schedulerRepo.GetGroupExecutions(groupID, 10)
		if err != nil {
			logger.Warn("Failed to check group executions before deletion", "error", err)
		} else {
			for _, exec := range executions {
				if exec.Status == "running" || exec.Status == "pending" {
					return fmt.Errorf("cannot delete group with active executions")
				}
			}
		}

		// Delete the group (CASCADE DELETE will handle memberships)
		if err := s.schedulerRepo.DeleteGroup(groupID); err != nil {
			return fmt.Errorf("failed to delete group: %w", err)
		}

		logger.Info("Successfully deleted machine group", "group_id", groupID)
		return nil
	})

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return err
	}

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return nil
}

// AddVMToGroup adds a VM to a machine group
func (s *MachineGroupService) AddVMToGroup(ctx context.Context, groupID string, req *VMMembershipRequest) (*database.VMGroupMembership, error) {
	// Start job tracking for VM addition
	owner := "machine-group-service"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "add-vm-to-group",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"group_id": groupID,
			"vm_context_id": req.VMContextID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start VM addition job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("➕ Adding VM to group",
		"group_id", groupID,
		"vm_context_id", req.VMContextID,
		"priority", req.Priority,
	)

	var membership *database.VMGroupMembership
	err = s.jobTracker.RunStep(ctx, jobID, "add-vm-membership", func(ctx context.Context) error {
		// Validate group exists
		_, err := s.schedulerRepo.GetGroupByID(groupID)
		if err != nil {
			return fmt.Errorf("invalid group_id: %w", err)
		}

		// Validate VM context exists
		_, err = s.schedulerRepo.GetVMContextByID(req.VMContextID)
		if err != nil {
			return fmt.Errorf("invalid vm_context_id: %w", err)
		}

		// Create membership
		membership = &database.VMGroupMembership{
			GroupID:     groupID,
			VMContextID: req.VMContextID,
			Priority:    req.Priority,
			Enabled:     req.Enabled,
		}

		if err := s.schedulerRepo.CreateMembership(membership); err != nil {
			return fmt.Errorf("failed to create membership: %w", err)
		}

		logger.Info("Successfully added VM to group",
			"membership_id", membership.ID,
			"group_id", groupID,
			"vm_context_id", req.VMContextID,
		)

		return nil
	})

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return nil, err
	}

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return membership, nil
}

// RemoveVMFromGroup removes a VM from a machine group
func (s *MachineGroupService) RemoveVMFromGroup(ctx context.Context, groupID string, vmContextID string) error {
	// Start job tracking for VM removal
	owner := "machine-group-service"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "remove-vm-from-group",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"group_id": groupID,
			"vm_context_id": vmContextID,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start VM removal job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("➖ Removing VM from group",
		"group_id", groupID,
		"vm_context_id", vmContextID,
	)

	err = s.jobTracker.RunStep(ctx, jobID, "remove-vm-membership", func(ctx context.Context) error {
		if err := s.schedulerRepo.DeleteMembership(groupID, vmContextID); err != nil {
			return fmt.Errorf("failed to delete membership: %w", err)
		}

		logger.Info("Successfully removed VM from group",
			"group_id", groupID,
			"vm_context_id", vmContextID,
		)

		return nil
	})

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return err
	}

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return nil
}

// UpdateVMMembership updates VM membership settings
func (s *MachineGroupService) UpdateVMMembership(ctx context.Context, groupID string, vmContextID string, priority *int, enabled *bool) error {
	// Start job tracking for membership update
	owner := "machine-group-service"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "update-vm-membership",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"group_id": groupID,
			"vm_context_id": vmContextID,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start membership update job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("📝 Updating VM membership",
		"group_id", groupID,
		"vm_context_id", vmContextID,
	)

	err = s.jobTracker.RunStep(ctx, jobID, "update-membership", func(ctx context.Context) error {
		updates := make(map[string]interface{})
		if priority != nil {
			updates["priority"] = *priority
		}
		if enabled != nil {
			updates["enabled"] = *enabled
		}
		updates["updated_at"] = time.Now()

		if err := s.schedulerRepo.UpdateMembership(groupID, vmContextID, updates); err != nil {
			return fmt.Errorf("failed to update membership: %w", err)
		}

		logger.Info("Successfully updated VM membership",
			"group_id", groupID,
			"vm_context_id", vmContextID,
			"updates", len(updates),
		)

		return nil
	})

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return err
	}

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return nil
}

// BulkAddVMs adds multiple VMs to a group
func (s *MachineGroupService) BulkAddVMs(ctx context.Context, groupID string, req *BulkMembershipRequest) (*BulkOperationResult, error) {
	startTime := time.Now()

	// Start job tracking for bulk addition
	owner := "machine-group-service"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "bulk-add-vms",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"group_id": groupID,
			"vm_count": len(req.VMContextIDs),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start bulk addition job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("📦 Bulk adding VMs to group",
		"group_id", groupID,
		"vm_count", len(req.VMContextIDs),
	)

	result := &BulkOperationResult{
		TotalRequested: len(req.VMContextIDs),
		SuccessfulIDs:  make([]string, 0),
		FailedIDs:      make([]string, 0),
		ErrorMessages:  make([]string, 0),
	}

	err = s.jobTracker.RunStep(ctx, jobID, "process-bulk-additions", func(ctx context.Context) error {
		for _, vmContextID := range req.VMContextIDs {
			membershipReq := &VMMembershipRequest{
				VMContextID: vmContextID,
				Priority:    req.Priority,
				Enabled:     req.Enabled,
			}

			_, err := s.AddVMToGroup(ctx, groupID, membershipReq)
			if err != nil {
				result.Failed++
				result.FailedIDs = append(result.FailedIDs, vmContextID)
				result.ErrorMessages = append(result.ErrorMessages, err.Error())
				logger.Error("Failed to add VM to group",
					"vm_context_id", vmContextID,
					"error", err,
				)
			} else {
				result.Successful++
				result.SuccessfulIDs = append(result.SuccessfulIDs, vmContextID)
			}
		}

		return nil
	})

	result.Duration = time.Since(startTime)

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return result, err
	}

	logger.Info("Bulk VM addition completed",
		"successful", result.Successful,
		"failed", result.Failed,
		"duration", result.Duration,
	)

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return result, nil
}

// BulkRemoveVMs removes multiple VMs from a group
func (s *MachineGroupService) BulkRemoveVMs(ctx context.Context, groupID string, vmContextIDs []string) (*BulkOperationResult, error) {
	startTime := time.Now()

	// Start job tracking for bulk removal
	owner := "machine-group-service"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "group-management",
		Operation: "bulk-remove-vms",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"group_id": groupID,
			"vm_count": len(vmContextIDs),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start bulk removal job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("📦 Bulk removing VMs from group",
		"group_id", groupID,
		"vm_count", len(vmContextIDs),
	)

	result := &BulkOperationResult{
		TotalRequested: len(vmContextIDs),
		SuccessfulIDs:  make([]string, 0),
		FailedIDs:      make([]string, 0),
		ErrorMessages:  make([]string, 0),
	}

	err = s.jobTracker.RunStep(ctx, jobID, "process-bulk-removals", func(ctx context.Context) error {
		for _, vmContextID := range vmContextIDs {
			err := s.RemoveVMFromGroup(ctx, groupID, vmContextID)
			if err != nil {
				result.Failed++
				result.FailedIDs = append(result.FailedIDs, vmContextID)
				result.ErrorMessages = append(result.ErrorMessages, err.Error())
				logger.Error("Failed to remove VM from group",
					"vm_context_id", vmContextID,
					"error", err,
				)
			} else {
				result.Successful++
				result.SuccessfulIDs = append(result.SuccessfulIDs, vmContextID)
			}
		}

		return nil
	})

	result.Duration = time.Since(startTime)

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return result, err
	}

	logger.Info("Bulk VM removal completed",
		"successful", result.Successful,
		"failed", result.Failed,
		"duration", result.Duration,
	)

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return result, nil
}

// ChangeGroupSchedule changes the schedule assignment for a group
func (s *MachineGroupService) ChangeGroupSchedule(ctx context.Context, groupID string, scheduleID *string) error {
	updateReq := &GroupUpdateRequest{
		ScheduleID: scheduleID,
	}
	
	_, err := s.UpdateGroup(ctx, groupID, updateReq)
	return err
}
