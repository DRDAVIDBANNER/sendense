package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
)

// =============================================================================
// ENHANCED BULK OPERATIONS SERVICE - Advanced VM Group Management
// =============================================================================
// CRITICAL: All VM operations use vm_replication_contexts.context_id
// Provides advanced bulk operations beyond basic group management

// EnhancedBulkOperationsService handles advanced bulk operations across groups
type EnhancedBulkOperationsService struct {
	schedulerRepo   *database.SchedulerRepository
	machineGroupSvc *MachineGroupService
	jobTracker      *joblog.Tracker
}

// BulkScheduleChangeRequest represents a request to change schedules for multiple groups
type BulkScheduleChangeRequest struct {
	GroupIDs      []string `json:"group_ids" validate:"required,min=1"`
	NewScheduleID *string  `json:"new_schedule_id,omitempty"`
	ValidateOnly  bool     `json:"validate_only"`
	ForceChange   bool     `json:"force_change"`
}

// CrossGroupVMRequest represents a request to move VMs between groups
type CrossGroupVMRequest struct {
	VMContextIDs  []string `json:"vm_context_ids" validate:"required,min=1"`
	SourceGroupID string   `json:"source_group_id" validate:"required"`
	TargetGroupID string   `json:"target_group_id" validate:"required"`
	Priority      int      `json:"priority" validate:"min=0"`
	Enabled       bool     `json:"enabled"`
	CopySettings  bool     `json:"copy_settings"` // Copy priority/enabled from source
}

// BulkValidationRequest represents a request to validate multiple VM operations
type BulkValidationRequest struct {
	Operations      []BulkOperation `json:"operations" validate:"required,min=1"`
	ValidateOnly    bool            `json:"validate_only"`
	ContinueOnError bool            `json:"continue_on_error"`
}

// BulkOperation represents a single operation in a bulk request
type BulkOperation struct {
	Type          string                 `json:"type"` // add_vm, remove_vm, move_vm, change_priority
	GroupID       string                 `json:"group_id"`
	VMContextID   *string                `json:"vm_context_id,omitempty"`
	TargetGroupID *string                `json:"target_group_id,omitempty"`
	Priority      *int                   `json:"priority,omitempty"`
	Enabled       *bool                  `json:"enabled,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AdvancedBulkResult provides detailed results of advanced bulk operations
type AdvancedBulkResult struct {
	TotalOperations  int               `json:"total_operations"`
	SuccessfulOps    int               `json:"successful_ops"`
	FailedOps        int               `json:"failed_ops"`
	SkippedOps       int               `json:"skipped_ops"`
	ValidationErrors []string          `json:"validation_errors,omitempty"`
	OperationResults []OperationResult `json:"operation_results"`
	ExecutionTime    time.Duration     `json:"execution_time"`
	AffectedGroups   []string          `json:"affected_groups"`
	AffectedVMs      []string          `json:"affected_vms"`
}

// OperationResult represents the result of a single bulk operation
type OperationResult struct {
	Operation        BulkOperation `json:"operation"`
	Success          bool          `json:"success"`
	ErrorMessage     *string       `json:"error_message,omitempty"`
	ExecutionTime    time.Duration `json:"execution_time"`
	AffectedEntities []string      `json:"affected_entities,omitempty"`
}

// GroupScheduleSummary provides summary of schedule changes across groups
type GroupScheduleSummary struct {
	GroupID        string  `json:"group_id"`
	GroupName      string  `json:"group_name"`
	OldScheduleID  *string `json:"old_schedule_id,omitempty"`
	NewScheduleID  *string `json:"new_schedule_id,omitempty"`
	VMCount        int     `json:"vm_count"`
	ActiveJobs     int     `json:"active_jobs"`
	ChangeAllowed  bool    `json:"change_allowed"`
	BlockingReason *string `json:"blocking_reason,omitempty"`
}

// NewEnhancedBulkOperationsService creates a new enhanced bulk operations service
func NewEnhancedBulkOperationsService(
	schedulerRepo *database.SchedulerRepository,
	machineGroupSvc *MachineGroupService,
	jobTracker *joblog.Tracker,
) *EnhancedBulkOperationsService {
	return &EnhancedBulkOperationsService{
		schedulerRepo:   schedulerRepo,
		machineGroupSvc: machineGroupSvc,
		jobTracker:      jobTracker,
	}
}

// BulkChangeGroupSchedules changes schedules for multiple groups
func (s *EnhancedBulkOperationsService) BulkChangeGroupSchedules(ctx context.Context, req *BulkScheduleChangeRequest) (*AdvancedBulkResult, error) {
	startTime := time.Now()

	// Start job tracking for bulk schedule change
	owner := "enhanced-bulk-operations"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "bulk-operations",
		Operation: "bulk-change-schedules",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"group_count":     len(req.GroupIDs),
			"new_schedule_id": req.NewScheduleID,
			"validate_only":   req.ValidateOnly,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start bulk schedule change job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("🔄 Starting bulk schedule change",
		"group_count", len(req.GroupIDs),
		"new_schedule_id", req.NewScheduleID,
		"validate_only", req.ValidateOnly,
	)

	result := &AdvancedBulkResult{
		TotalOperations:  len(req.GroupIDs),
		OperationResults: make([]OperationResult, 0),
		AffectedGroups:   make([]string, 0),
		ValidationErrors: make([]string, 0),
	}

	var summaries []GroupScheduleSummary
	err = s.jobTracker.RunStep(ctx, jobID, "validate-schedule-changes", func(ctx context.Context) error {
		// Validate new schedule exists if provided
		if req.NewScheduleID != nil {
			_, err := s.schedulerRepo.GetScheduleByID(*req.NewScheduleID)
			if err != nil {
				result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("Invalid new_schedule_id: %s", err.Error()))
				return fmt.Errorf("invalid new_schedule_id: %w", err)
			}
		}

		// Analyze each group
		for _, groupID := range req.GroupIDs {
			summary, err := s.analyzeGroupScheduleChange(ctx, groupID, req.NewScheduleID, req.ForceChange)
			if err != nil {
				result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("Group %s: %s", groupID, err.Error()))
				if !req.ForceChange {
					continue
				}
			}
			summaries = append(summaries, *summary)
		}

		return nil
	})

	if err != nil && !req.ForceChange {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return result, err
	}

	// If validation only, return results without making changes
	if req.ValidateOnly {
		result.ExecutionTime = time.Since(startTime)
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
		return result, nil
	}

	// Execute schedule changes
	err = s.jobTracker.RunStep(ctx, jobID, "execute-schedule-changes", func(ctx context.Context) error {
		for _, summary := range summaries {
			if !summary.ChangeAllowed && !req.ForceChange {
				opResult := OperationResult{
					Operation: BulkOperation{
						Type:    "change_schedule",
						GroupID: summary.GroupID,
					},
					Success:      false,
					ErrorMessage: summary.BlockingReason,
				}
				result.OperationResults = append(result.OperationResults, opResult)
				result.SkippedOps++
				continue
			}

			// Perform the schedule change
			opStart := time.Now()
			err := s.machineGroupSvc.ChangeGroupSchedule(ctx, summary.GroupID, req.NewScheduleID)
			opDuration := time.Since(opStart)

			opResult := OperationResult{
				Operation: BulkOperation{
					Type:    "change_schedule",
					GroupID: summary.GroupID,
				},
				ExecutionTime:    opDuration,
				AffectedEntities: []string{summary.GroupID},
			}

			if err != nil {
				opResult.Success = false
				errMsg := err.Error()
				opResult.ErrorMessage = &errMsg
				result.FailedOps++

				logger.Error("Failed to change group schedule",
					"group_id", summary.GroupID,
					"error", err,
				)
			} else {
				opResult.Success = true
				result.SuccessfulOps++
				result.AffectedGroups = append(result.AffectedGroups, summary.GroupID)

				logger.Info("Successfully changed group schedule",
					"group_id", summary.GroupID,
					"old_schedule", summary.OldScheduleID,
					"new_schedule", req.NewScheduleID,
				)
			}

			result.OperationResults = append(result.OperationResults, opResult)
		}

		return nil
	})

	result.ExecutionTime = time.Since(startTime)

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return result, err
	}

	logger.Info("Bulk schedule change completed",
		"successful", result.SuccessfulOps,
		"failed", result.FailedOps,
		"skipped", result.SkippedOps,
		"duration", result.ExecutionTime,
	)

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return result, nil
}

// CrossGroupVMMovement moves VMs between groups
func (s *EnhancedBulkOperationsService) CrossGroupVMMovement(ctx context.Context, req *CrossGroupVMRequest) (*AdvancedBulkResult, error) {
	startTime := time.Now()

	// Start job tracking for cross-group movement
	owner := "enhanced-bulk-operations"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "bulk-operations",
		Operation: "cross-group-vm-movement",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"vm_count":     len(req.VMContextIDs),
			"source_group": req.SourceGroupID,
			"target_group": req.TargetGroupID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start cross-group movement job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("🔄 Starting cross-group VM movement",
		"vm_count", len(req.VMContextIDs),
		"source_group", req.SourceGroupID,
		"target_group", req.TargetGroupID,
	)

	result := &AdvancedBulkResult{
		TotalOperations:  len(req.VMContextIDs) * 2, // Remove + Add operations
		OperationResults: make([]OperationResult, 0),
		AffectedGroups:   []string{req.SourceGroupID, req.TargetGroupID},
		AffectedVMs:      make([]string, 0),
	}

	// Validate groups exist
	err = s.jobTracker.RunStep(ctx, jobID, "validate-groups", func(ctx context.Context) error {
		_, err := s.schedulerRepo.GetGroupByID(req.SourceGroupID)
		if err != nil {
			return fmt.Errorf("invalid source_group_id: %w", err)
		}

		_, err = s.schedulerRepo.GetGroupByID(req.TargetGroupID)
		if err != nil {
			return fmt.Errorf("invalid target_group_id: %w", err)
		}

		return nil
	})

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return result, err
	}

	// Process each VM movement
	err = s.jobTracker.RunStep(ctx, jobID, "process-vm-movements", func(ctx context.Context) error {
		for _, vmContextID := range req.VMContextIDs {
			// Get current membership settings if copying
			var priority int = req.Priority
			var enabled bool = req.Enabled

			if req.CopySettings {
				membership, err := s.getMembershipSettings(vmContextID, req.SourceGroupID)
				if err == nil {
					priority = membership.Priority
					enabled = membership.Enabled
				}
			}

			// Remove from source group
			removeStart := time.Now()
			removeErr := s.machineGroupSvc.RemoveVMFromGroup(ctx, req.SourceGroupID, vmContextID)
			removeDuration := time.Since(removeStart)

			removeResult := OperationResult{
				Operation: BulkOperation{
					Type:        "remove_vm",
					GroupID:     req.SourceGroupID,
					VMContextID: &vmContextID,
				},
				ExecutionTime:    removeDuration,
				AffectedEntities: []string{vmContextID, req.SourceGroupID},
			}

			if removeErr != nil {
				removeResult.Success = false
				errMsg := removeErr.Error()
				removeResult.ErrorMessage = &errMsg
				result.FailedOps++
			} else {
				removeResult.Success = true
				result.SuccessfulOps++
			}

			result.OperationResults = append(result.OperationResults, removeResult)

			// Add to target group (only if removal succeeded or if continuing on error)
			if removeErr == nil {
				addRequest := &VMMembershipRequest{
					VMContextID: vmContextID,
					Priority:    priority,
					Enabled:     enabled,
				}

				addStart := time.Now()
				_, addErr := s.machineGroupSvc.AddVMToGroup(ctx, req.TargetGroupID, addRequest)
				addDuration := time.Since(addStart)

				addResult := OperationResult{
					Operation: BulkOperation{
						Type:        "add_vm",
						GroupID:     req.TargetGroupID,
						VMContextID: &vmContextID,
					},
					ExecutionTime:    addDuration,
					AffectedEntities: []string{vmContextID, req.TargetGroupID},
				}

				if addErr != nil {
					addResult.Success = false
					errMsg := addErr.Error()
					addResult.ErrorMessage = &errMsg
					result.FailedOps++
				} else {
					addResult.Success = true
					result.SuccessfulOps++
					result.AffectedVMs = append(result.AffectedVMs, vmContextID)
				}

				result.OperationResults = append(result.OperationResults, addResult)
			} else {
				result.SkippedOps++ // Skip add operation due to failed remove
			}
		}

		return nil
	})

	result.ExecutionTime = time.Since(startTime)

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return result, err
	}

	logger.Info("Cross-group VM movement completed",
		"successful", result.SuccessfulOps,
		"failed", result.FailedOps,
		"skipped", result.SkippedOps,
		"moved_vms", len(result.AffectedVMs),
		"duration", result.ExecutionTime,
	)

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return result, nil
}

// ExecuteBulkOperations executes a series of bulk operations
func (s *EnhancedBulkOperationsService) ExecuteBulkOperations(ctx context.Context, req *BulkValidationRequest) (*AdvancedBulkResult, error) {
	startTime := time.Now()

	// Start job tracking for bulk operations
	owner := "enhanced-bulk-operations"
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "bulk-operations",
		Operation: "execute-bulk-operations",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"operation_count": len(req.Operations),
			"validate_only":   req.ValidateOnly,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start bulk operations job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("🔄 Starting bulk operations execution",
		"operation_count", len(req.Operations),
		"validate_only", req.ValidateOnly,
	)

	result := &AdvancedBulkResult{
		TotalOperations:  len(req.Operations),
		OperationResults: make([]OperationResult, 0),
		AffectedGroups:   make([]string, 0),
		AffectedVMs:      make([]string, 0),
	}

	// Execute each operation
	err = s.jobTracker.RunStep(ctx, jobID, "execute-operations", func(ctx context.Context) error {
		for _, operation := range req.Operations {
			opResult := s.executeOperation(ctx, operation, req.ValidateOnly)
			result.OperationResults = append(result.OperationResults, opResult)

			if opResult.Success {
				result.SuccessfulOps++
			} else {
				result.FailedOps++
				if !req.ContinueOnError {
					return fmt.Errorf("operation failed: %s", *opResult.ErrorMessage)
				}
			}

			// Track affected entities
			if !contains(result.AffectedGroups, operation.GroupID) {
				result.AffectedGroups = append(result.AffectedGroups, operation.GroupID)
			}

			if operation.VMContextID != nil && !contains(result.AffectedVMs, *operation.VMContextID) {
				result.AffectedVMs = append(result.AffectedVMs, *operation.VMContextID)
			}
		}

		return nil
	})

	result.ExecutionTime = time.Since(startTime)

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return result, err
	}

	logger.Info("Bulk operations completed",
		"successful", result.SuccessfulOps,
		"failed", result.FailedOps,
		"duration", result.ExecutionTime,
	)

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return result, nil
}

// analyzeGroupScheduleChange analyzes whether a group can have its schedule changed
func (s *EnhancedBulkOperationsService) analyzeGroupScheduleChange(ctx context.Context, groupID string, newScheduleID *string, force bool) (*GroupScheduleSummary, error) {
	group, err := s.schedulerRepo.GetGroupByID(groupID, "Schedule", "Memberships")
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	summary := &GroupScheduleSummary{
		GroupID:       groupID,
		GroupName:     group.Name,
		NewScheduleID: newScheduleID,
		VMCount:       len(group.Memberships),
		ChangeAllowed: true,
	}

	if group.Schedule != nil {
		summary.OldScheduleID = &group.Schedule.ID
	}

	// Check for active jobs (would require ReplicationJobRepository integration)
	// For now, assume no active jobs
	summary.ActiveJobs = 0

	// Determine if change is allowed
	if summary.ActiveJobs > 0 && !force {
		summary.ChangeAllowed = false
		reason := fmt.Sprintf("Group has %d active jobs", summary.ActiveJobs)
		summary.BlockingReason = &reason
	}

	return summary, nil
}

// executeOperation executes a single bulk operation
func (s *EnhancedBulkOperationsService) executeOperation(ctx context.Context, operation BulkOperation, validateOnly bool) OperationResult {
	startTime := time.Now()

	result := OperationResult{
		Operation:        operation,
		ExecutionTime:    0,
		AffectedEntities: make([]string, 0),
	}

	if validateOnly {
		result.Success = true
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	switch operation.Type {
	case "add_vm":
		if operation.VMContextID == nil {
			result.Success = false
			errMsg := "vm_context_id required for add_vm operation"
			result.ErrorMessage = &errMsg
		} else {
			req := &VMMembershipRequest{
				VMContextID: *operation.VMContextID,
				Priority:    *operation.Priority,
				Enabled:     *operation.Enabled,
			}
			_, err := s.machineGroupSvc.AddVMToGroup(ctx, operation.GroupID, req)
			if err != nil {
				result.Success = false
				errMsg := err.Error()
				result.ErrorMessage = &errMsg
			} else {
				result.Success = true
				result.AffectedEntities = []string{*operation.VMContextID, operation.GroupID}
			}
		}

	case "remove_vm":
		if operation.VMContextID == nil {
			result.Success = false
			errMsg := "vm_context_id required for remove_vm operation"
			result.ErrorMessage = &errMsg
		} else {
			err := s.machineGroupSvc.RemoveVMFromGroup(ctx, operation.GroupID, *operation.VMContextID)
			if err != nil {
				result.Success = false
				errMsg := err.Error()
				result.ErrorMessage = &errMsg
			} else {
				result.Success = true
				result.AffectedEntities = []string{*operation.VMContextID, operation.GroupID}
			}
		}

	case "change_priority":
		if operation.VMContextID == nil || operation.Priority == nil {
			result.Success = false
			errMsg := "vm_context_id and priority required for change_priority operation"
			result.ErrorMessage = &errMsg
		} else {
			err := s.machineGroupSvc.UpdateVMMembership(ctx, operation.GroupID, *operation.VMContextID, operation.Priority, nil)
			if err != nil {
				result.Success = false
				errMsg := err.Error()
				result.ErrorMessage = &errMsg
			} else {
				result.Success = true
				result.AffectedEntities = []string{*operation.VMContextID, operation.GroupID}
			}
		}

	default:
		result.Success = false
		errMsg := fmt.Sprintf("unknown operation type: %s", operation.Type)
		result.ErrorMessage = &errMsg
	}

	result.ExecutionTime = time.Since(startTime)
	return result
}

// getMembershipSettings gets current membership settings for a VM in a group
func (s *EnhancedBulkOperationsService) getMembershipSettings(vmContextID, groupID string) (*database.VMGroupMembership, error) {
	// This would require a new repository method to get specific membership
	// For now, return default values
	return &database.VMGroupMembership{
		VMContextID: vmContextID,
		GroupID:     groupID,
		Priority:    0,
		Enabled:     true,
	}, nil
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
