package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
)

// =============================================================================
// JOB CONFLICT DETECTOR - Intelligent Job Validation using context_id
// =============================================================================
// CRITICAL: All VM operations use vm_replication_contexts.context_id
// Prevents job conflicts and respects scheduling constraints

// JobConflictDetector handles job conflict detection and validation
type JobConflictDetector struct {
	replicationRepo *database.ReplicationJobRepository
	schedulerRepo   *database.SchedulerRepository
	jobTracker      *joblog.Tracker
}

// ConflictType represents different types of job conflicts
type ConflictType string

const (
	ConflictActiveJob        ConflictType = "active_job"        // VM has active job
	ConflictSkipIfRunning    ConflictType = "skip_if_running"   // Schedule setting prevents execution
	ConflictMaxConcurrent    ConflictType = "max_concurrent"    // Max concurrent jobs exceeded
	ConflictGroupConcurrent  ConflictType = "group_concurrent"  // Group concurrent limit exceeded
	ConflictVMDisabled       ConflictType = "vm_disabled"       // VM membership disabled
	ConflictScheduleDisabled ConflictType = "schedule_disabled" // Schedule is disabled
	ConflictVMInFailover     ConflictType = "vm_in_failover"    // VM is in failover state
)

// ConflictResult represents the result of conflict detection for a VM
type ConflictResult struct {
	VMContextID      string       `json:"vm_context_id"`
	VMName           string       `json:"vm_name"`
	HasConflict      bool         `json:"has_conflict"`
	ConflictType     ConflictType `json:"conflict_type,omitempty"`
	ConflictReason   string       `json:"conflict_reason,omitempty"`
	CurrentJobID     *string      `json:"current_job_id,omitempty"`
	CurrentJobStatus *string      `json:"current_job_status,omitempty"`
	CanSchedule      bool         `json:"can_schedule"`
	SkippedReason    *string      `json:"skipped_reason,omitempty"`
}

// ConflictScanSummary provides summary of conflict detection across multiple VMs
type ConflictScanSummary struct {
	TotalVMs       int                  `json:"total_vms"`
	EligibleVMs    int                  `json:"eligible_vms"`
	ConflictedVMs  int                  `json:"conflicted_vms"`
	DisabledVMs    int                  `json:"disabled_vms"`
	Results        []ConflictResult     `json:"results"`
	ConflictCounts map[ConflictType]int `json:"conflict_counts"`
	ScanDuration   time.Duration        `json:"scan_duration"`
}

// ScheduleConstraints represents schedule-level constraints for job validation
type ScheduleConstraints struct {
	ScheduleID        string `json:"schedule_id"`
	SkipIfRunning     bool   `json:"skip_if_running"`
	MaxConcurrentJobs int    `json:"max_concurrent_jobs"`
	Enabled           bool   `json:"enabled"`
}

// GroupConstraints represents group-level constraints for job validation
type GroupConstraints struct {
	GroupID          string `json:"group_id"`
	MaxConcurrentVMs int    `json:"max_concurrent_vms"`
	Priority         int    `json:"priority"`
}

// NewJobConflictDetector creates a new job conflict detector
func NewJobConflictDetector(
	replicationRepo *database.ReplicationJobRepository,
	schedulerRepo *database.SchedulerRepository,
	jobTracker *joblog.Tracker,
) *JobConflictDetector {
	return &JobConflictDetector{
		replicationRepo: replicationRepo,
		schedulerRepo:   schedulerRepo,
		jobTracker:      jobTracker,
	}
}

// CheckVMConflicts analyzes a list of VM contexts for scheduling conflicts
func (d *JobConflictDetector) CheckVMConflicts(
	ctx context.Context,
	vmContexts []*database.VMReplicationContext,
	scheduleConstraints *ScheduleConstraints,
	groupConstraints *GroupConstraints,
) (*ConflictScanSummary, error) {
	startTime := time.Now()

	// Start job tracking for conflict detection
	owner := "job-conflict-detector"
	ctx, conflictJobID, err := d.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "conflict-detection",
		Operation: "check-vm-conflicts",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"vm_count":    len(vmContexts),
			"schedule_id": scheduleConstraints.ScheduleID,
			"group_id":    groupConstraints.GroupID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start conflict detection job tracking: %w", err)
	}

	logger := d.jobTracker.Logger(ctx)
	logger.Info("🔍 Starting VM conflict detection",
		"vm_count", len(vmContexts),
		"schedule_id", scheduleConstraints.ScheduleID,
		"group_id", groupConstraints.GroupID,
	)

	summary := &ConflictScanSummary{
		TotalVMs:       len(vmContexts),
		Results:        make([]ConflictResult, 0, len(vmContexts)),
		ConflictCounts: make(map[ConflictType]int),
	}

	// Get current active jobs for conflict checking
	var activeJobs []database.ReplicationJob
	err = d.jobTracker.RunStep(ctx, conflictJobID, "fetch-active-jobs", func(ctx context.Context) error {
		activeJobs, err = d.replicationRepo.GetJobsByStatus([]string{"pending", "replicating", "provisioning"})
		if err != nil {
			return fmt.Errorf("failed to fetch active jobs: %w", err)
		}

		logger.Info("Fetched active jobs for conflict detection", "active_jobs", len(activeJobs))
		return nil
	})

	if err != nil {
		d.jobTracker.EndJob(ctx, conflictJobID, joblog.StatusFailed, err)
		return nil, err
	}

	// Create active job map by VM context ID for fast lookup
	activeJobMap := make(map[string]*database.ReplicationJob)
	for i := range activeJobs {
		job := &activeJobs[i]
		if job.VMContextID != "" {
			activeJobMap[job.VMContextID] = job
		}
	}

	// Count current running jobs for schedule-level limits
	currentScheduleJobs := 0
	currentGroupJobs := 0
	for _, job := range activeJobs {
		if job.ScheduleExecutionID != nil && job.Status != "failed" && job.Status != "completed" {
			currentScheduleJobs++
		}
		if job.VMGroupID != nil && *job.VMGroupID == groupConstraints.GroupID {
			currentGroupJobs++
		}
	}

	// Check each VM for conflicts
	err = d.jobTracker.RunStep(ctx, conflictJobID, "analyze-vm-conflicts", func(ctx context.Context) error {
		for _, vmCtx := range vmContexts {
			result := d.analyzeVMConflict(
				vmCtx,
				activeJobMap,
				scheduleConstraints,
				groupConstraints,
				currentScheduleJobs,
				currentGroupJobs,
			)

			summary.Results = append(summary.Results, result)

			if result.HasConflict {
				summary.ConflictedVMs++
				summary.ConflictCounts[result.ConflictType]++
			} else if result.CanSchedule {
				summary.EligibleVMs++
			} else {
				summary.DisabledVMs++
			}

			// If this VM would be scheduled, increment counters for next iterations
			if result.CanSchedule && !result.HasConflict {
				currentScheduleJobs++
				currentGroupJobs++
			}
		}

		return nil
	})

	if err != nil {
		d.jobTracker.EndJob(ctx, conflictJobID, joblog.StatusFailed, err)
		return nil, err
	}

	summary.ScanDuration = time.Since(startTime)

	logger.Info("Conflict detection completed",
		"total_vms", summary.TotalVMs,
		"eligible", summary.EligibleVMs,
		"conflicted", summary.ConflictedVMs,
		"disabled", summary.DisabledVMs,
		"duration", summary.ScanDuration,
	)

	d.jobTracker.EndJob(ctx, conflictJobID, joblog.StatusCompleted, nil)
	return summary, nil
}

// analyzeVMConflict performs conflict analysis for a single VM
func (d *JobConflictDetector) analyzeVMConflict(
	vmCtx *database.VMReplicationContext,
	activeJobMap map[string]*database.ReplicationJob,
	scheduleConstraints *ScheduleConstraints,
	groupConstraints *GroupConstraints,
	currentScheduleJobs int,
	currentGroupJobs int,
) ConflictResult {
	result := ConflictResult{
		VMContextID: vmCtx.ContextID,
		VMName:      vmCtx.VMName,
		HasConflict: false,
		CanSchedule: true,
	}

	// Check 1: Schedule enabled
	if !scheduleConstraints.Enabled {
		result.HasConflict = true
		result.ConflictType = ConflictScheduleDisabled
		result.ConflictReason = "Schedule is disabled"
		result.CanSchedule = false
		return result
	}

	// Check 2: VM scheduler enabled
	if !vmCtx.SchedulerEnabled {
		result.HasConflict = true
		result.ConflictType = ConflictVMDisabled
		result.ConflictReason = "VM scheduler is disabled"
		result.CanSchedule = false
		result.SkippedReason = &result.ConflictReason
		return result
	}

	// Check 3: VM current status (failover state detection)
	// CRITICAL: Skip VMs in failover states to prevent data corruption and conflicts
	if vmCtx.CurrentStatus == "failed_over_test" ||
		vmCtx.CurrentStatus == "failed_over_live" ||
		vmCtx.CurrentStatus == "cleanup_required" {
		result.HasConflict = true
		result.ConflictType = ConflictVMInFailover
		result.ConflictReason = fmt.Sprintf("VM is in failover state: %s (cannot replicate while failed over)", vmCtx.CurrentStatus)
		result.CanSchedule = false
		result.SkippedReason = &result.ConflictReason
		return result
	}

	// Check 4: Active job exists for this VM
	if activeJob, hasActiveJob := activeJobMap[vmCtx.ContextID]; hasActiveJob {
		result.CurrentJobID = &activeJob.ID
		result.CurrentJobStatus = &activeJob.Status

		if scheduleConstraints.SkipIfRunning {
			result.HasConflict = true
			result.ConflictType = ConflictSkipIfRunning
			result.ConflictReason = fmt.Sprintf("VM has active job (ID: %s, Status: %s) and skip_if_running is enabled",
				activeJob.ID, activeJob.Status)
			result.CanSchedule = false
			return result
		} else {
			result.HasConflict = true
			result.ConflictType = ConflictActiveJob
			result.ConflictReason = fmt.Sprintf("VM has active job (ID: %s, Status: %s)",
				activeJob.ID, activeJob.Status)
			result.CanSchedule = false
			return result
		}
	}

	// Check 5: Schedule-level max concurrent jobs
	if currentScheduleJobs >= scheduleConstraints.MaxConcurrentJobs {
		result.HasConflict = true
		result.ConflictType = ConflictMaxConcurrent
		result.ConflictReason = fmt.Sprintf("Schedule max concurrent jobs limit reached (%d/%d)",
			currentScheduleJobs, scheduleConstraints.MaxConcurrentJobs)
		result.CanSchedule = false
		return result
	}

	// Check 6: Group-level max concurrent VMs
	if currentGroupJobs >= groupConstraints.MaxConcurrentVMs {
		result.HasConflict = true
		result.ConflictType = ConflictGroupConcurrent
		result.ConflictReason = fmt.Sprintf("Group max concurrent VMs limit reached (%d/%d)",
			currentGroupJobs, groupConstraints.MaxConcurrentVMs)
		result.CanSchedule = false
		return result
	}

	// No conflicts found - VM can be scheduled
	result.ConflictReason = "No conflicts detected - VM eligible for scheduling"
	return result
}

// CheckSingleVMConflict checks conflicts for a single VM context
func (d *JobConflictDetector) CheckSingleVMConflict(
	ctx context.Context,
	vmContextID string,
	scheduleID string,
	groupID string,
) (*ConflictResult, error) {
	// Get VM context
	vmCtx, err := d.schedulerRepo.GetVMContextByID(vmContextID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM context: %w", err)
	}

	// Get schedule details
	schedule, err := d.schedulerRepo.GetScheduleByID(scheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	// Get group details
	group, err := d.schedulerRepo.GetGroupByID(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	scheduleConstraints := &ScheduleConstraints{
		ScheduleID:        schedule.ID,
		SkipIfRunning:     schedule.SkipIfRunning,
		MaxConcurrentJobs: schedule.MaxConcurrentJobs,
		Enabled:           schedule.Enabled,
	}

	groupConstraints := &GroupConstraints{
		GroupID:          group.ID,
		MaxConcurrentVMs: group.MaxConcurrentVMs,
		Priority:         group.Priority,
	}

	// Analyze single VM
	summary, err := d.CheckVMConflicts(ctx, []*database.VMReplicationContext{vmCtx}, scheduleConstraints, groupConstraints)
	if err != nil {
		return nil, err
	}

	if len(summary.Results) == 0 {
		return nil, fmt.Errorf("no results returned from conflict check")
	}

	return &summary.Results[0], nil
}

// GetEligibleVMs filters VMs that can be scheduled based on conflict detection
func (d *JobConflictDetector) GetEligibleVMs(
	ctx context.Context,
	vmContexts []*database.VMReplicationContext,
	scheduleConstraints *ScheduleConstraints,
	groupConstraints *GroupConstraints,
) ([]*database.VMReplicationContext, error) {
	summary, err := d.CheckVMConflicts(ctx, vmContexts, scheduleConstraints, groupConstraints)
	if err != nil {
		return nil, err
	}

	// Build list of eligible VMs
	eligible := make([]*database.VMReplicationContext, 0)
	for i, result := range summary.Results {
		if result.CanSchedule && !result.HasConflict {
			eligible = append(eligible, vmContexts[i])
		}
	}

	return eligible, nil
}
