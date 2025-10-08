package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/models"
)

// =============================================================================
// SCHEDULER SERVICE - VM-Centric Automation using context_id
// =============================================================================
// ALL VM operations in this service use vm_replication_contexts.context_id
// Following project rules: joblog integration, modular design, no monster code

// SchedulerService manages automated replication scheduling
// CRITICAL: All VM references use context_id, never vm_name or other identifiers
// ALIGNED: Uses same workflow as GUI with fresh SNA discovery + SHA API calls
type SchedulerService struct {
	repository       *database.SchedulerRepository
	replicationRepo  *database.ReplicationJobRepository
	jobTracker       *joblog.Tracker
	phantomDetector  *PhantomJobDetector
	conflictDetector *JobConflictDetector
	cron             *cron.Cron

	// ✅ NEW: SNA Discovery Integration (aligned with GUI workflow)
	snaAPIEndpoint string
	snaClient      *http.Client
	shaAPIEndpoint string
	shaClient      *http.Client

	// Concurrent execution tracking
	runningMutex    sync.RWMutex
	activeSchedules map[string]*ScheduleContext
	maxConcurrent   int

	// Service lifecycle
	stopChan     chan struct{}
	isRunning    bool
	runningCount int
}

// ScheduleContext tracks execution state for individual schedules
type ScheduleContext struct {
	ScheduleID       string                        `json:"schedule_id"`
	Schedule         *database.ReplicationSchedule `json:"schedule"`
	NextRun          time.Time                     `json:"next_run"`
	LastRun          time.Time                     `json:"last_run"`
	IsRunning        bool                          `json:"is_running"`
	CurrentExecution *database.ScheduleExecution   `json:"current_execution,omitempty"`
	JobCount         int                           `json:"job_count"`
	CronEntryID      cron.EntryID                  `json:"cron_entry_id"`
}

// ExecutionSummary provides execution results using context_id references
type ExecutionSummary struct {
	ExecutionID         string                 `json:"execution_id"`
	ScheduleID          string                 `json:"schedule_id"`
	GroupID             *string                `json:"group_id"`
	Status              string                 `json:"status"`
	StartedAt           time.Time              `json:"started_at"`
	VMsEligible         int                    `json:"vms_eligible"`
	VMsProcessed        int                    `json:"vms_processed"`
	JobsCreated         int                    `json:"jobs_created"`
	JobsCompleted       int                    `json:"jobs_completed"`
	JobsFailed          int                    `json:"jobs_failed"`
	JobsSkipped         int                    `json:"jobs_skipped"`
	VMContextsProcessed []string               `json:"vm_contexts_processed"` // context_ids
	CreatedJobIDs       []string               `json:"created_job_ids"`
	ExecutionTime       time.Duration          `json:"execution_time"`
	Summary             map[string]interface{} `json:"summary,omitempty"`
	ErrorMessage        *string                `json:"error_message,omitempty"`
}

// ✅ SNA Discovery Integration (aligned with GUI workflow)
// These structures match the SNA API response format used by GUI

// VMDiscoveryRequest matches SNA API discovery request format
type VMDiscoveryRequest struct {
	VCenter    string `json:"vcenter"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Datacenter string `json:"datacenter"`
	Filter     string `json:"filter"` // VM name filter
}

// VMDiscoveryResponse matches SNA API discovery response format
type VMDiscoveryResponse struct {
	VMs []VMDiscoveryData `json:"vms"`
}

// VMDiscoveryData matches the VM data structure from SNA discovery
type VMDiscoveryData struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Path       string               `json:"path"`
	Datacenter string               `json:"datacenter"`
	NumCPU     int                  `json:"num_cpu"`
	CPUs       int                  `json:"cpus"` // Alternative field name
	MemoryMB   int                  `json:"memory_mb"`
	PowerState string               `json:"power_state"`
	GuestOS    string               `json:"guest_os"`
	VMXVersion string               `json:"vmx_version"`
	Disks      []models.DiskInfo    `json:"disks"`
	Networks   []models.NetworkInfo `json:"networks"`
}

// CreateMigrationRequest matches SHA API replication request format
type CreateMigrationRequest struct {
	SourceVM         models.VMInfo `json:"source_vm"`
	OSSEAConfigID    int           `json:"ossea_config_id"`
	ReplicationType  string        `json:"replication_type"`
	TargetNetwork    string        `json:"target_network"`
	VCenterHost      string        `json:"vcenter_host"`
	Datacenter       string        `json:"datacenter"`
	ChangeID         string        `json:"change_id"`
	PreviousChangeID string        `json:"previous_change_id"`
	SnapshotID       string        `json:"snapshot_id"`
	// ✅ NEW: Scheduler metadata
	ScheduleExecutionID string `json:"schedule_execution_id,omitempty"`
	VMGroupID           string `json:"vm_group_id,omitempty"`
	ScheduledBy         string `json:"scheduled_by,omitempty"`
}

// MigrationResult matches actual SHA API replication response format
type MigrationResult struct {
	JobID           string     `json:"job_id"`
	Status          string     `json:"status"`
	ProgressPercent float64    `json:"progress_percent"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`

	// Nested objects from actual response
	SourceVM      SourceVMInfo     `json:"source_vm"`
	Configuration ConfigInfo       `json:"configuration"`
	Disks         []DiskInfo       `json:"disks"`
	Mounts        []interface{}    `json:"mounts"` // Can be empty array
	CBTHistory    []CBTHistoryInfo `json:"cbt_history"`
}

// Supporting structs for MigrationResult
type SourceVMInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	VCenterHost string `json:"vcenter_host"`
	Datacenter  string `json:"datacenter"`
}

type ConfigInfo struct {
	ReplicationType string `json:"replication_type"`
	TargetNetwork   string `json:"target_network"`
	OSSeaConfigID   int    `json:"ossea_config_id"`
}

type DiskInfo struct {
	ID                  int     `json:"id"`
	DiskID              string  `json:"disk_id"`
	VMDKPath            string  `json:"vmdk_path"`
	SizeGB              int     `json:"size_gb"`
	SyncStatus          string  `json:"sync_status"`
	SyncProgressPercent float64 `json:"sync_progress_percent"`
	BytesSynced         int64   `json:"bytes_synced"`
	ChangeID            string  `json:"change_id"`
	OSSeaVolumeID       int     `json:"ossea_volume_id"`
}

type CBTHistoryInfo struct {
	ID                  int       `json:"id"`
	DiskID              string    `json:"disk_id"`
	ChangeID            string    `json:"change_id"`
	SyncType            string    `json:"sync_type"`
	SyncSuccess         bool      `json:"sync_success"`
	BlocksChanged       int       `json:"blocks_changed"`
	BytesTransferred    int64     `json:"bytes_transferred"`
	SyncDurationSeconds int       `json:"sync_duration_seconds"`
	CreatedAt           time.Time `json:"created_at"`
}

// NewSchedulerService creates a new scheduler service
// ✅ UPDATED: Now includes SNA discovery and SHA API integration (aligned with GUI workflow)
func NewSchedulerService(
	schedulerRepo *database.SchedulerRepository,
	replicationRepo *database.ReplicationJobRepository,
	jobTracker *joblog.Tracker,
	snaAPIEndpoint string,
) *SchedulerService {
	phantomDetector := NewPhantomJobDetector(replicationRepo, jobTracker, snaAPIEndpoint)
	conflictDetector := NewJobConflictDetector(replicationRepo, schedulerRepo, jobTracker)

	return &SchedulerService{
		repository:       schedulerRepo,
		replicationRepo:  replicationRepo,
		jobTracker:       jobTracker,
		phantomDetector:  phantomDetector,
		conflictDetector: conflictDetector,
		cron:             cron.New(cron.WithSeconds()), // Support second-level precision

		// ✅ NEW: SNA Discovery Integration (same as GUI)
		snaAPIEndpoint: snaAPIEndpoint,
		snaClient:      &http.Client{Timeout: 30 * time.Second},
		shaAPIEndpoint: "http://localhost:8082", // Same endpoint as GUI
		shaClient:      &http.Client{Timeout: 60 * time.Second},

		activeSchedules: make(map[string]*ScheduleContext),
		maxConcurrent:   10, // Maximum concurrent schedule executions
		stopChan:        make(chan struct{}),
	}
}

// Start initializes and starts the scheduler service
func (s *SchedulerService) Start(ctx context.Context) error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler service already running")
	}

	log.Info("🚀 Starting scheduler service")

	// Initialize job tracker for scheduler operations
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "scheduler",
		Operation: "service-start",
		Owner:     stringPtr("system"),
	})
	if err != nil {
		return fmt.Errorf("failed to start scheduler job tracking: %w", err)
	}

	// Load and register all enabled schedules
	err = s.jobTracker.RunStep(ctx, jobID, "load-schedules", func(ctx context.Context) error {
		logger := s.jobTracker.Logger(ctx)

		schedules, err := s.repository.ListSchedules(true) // enabledOnly = true
		if err != nil {
			return fmt.Errorf("failed to load schedules: %w", err)
		}

		logger.Info("Loading enabled schedules", "count", len(schedules))

		for _, schedule := range schedules {
			if err := s.registerSchedule(&schedule); err != nil {
				logger.Error("Failed to register schedule", "error", err, "schedule_id", schedule.ID)
				continue
			}
		}

		logger.Info("Successfully registered schedules", "registered", len(s.activeSchedules))
		return nil
	})

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return err
	}

	// Start the cron scheduler
	err = s.jobTracker.RunStep(ctx, jobID, "start-cron", func(ctx context.Context) error {
		s.cron.Start()
		s.isRunning = true
		return nil
	})

	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return err
	}

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	log.WithField("active_schedules", len(s.activeSchedules)).Info("✅ Scheduler service started successfully")
	return nil
}

// Stop gracefully stops the scheduler service
func (s *SchedulerService) Stop(ctx context.Context) error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler service not running")
	}

	log.Info("🛑 Stopping scheduler service")

	// Signal stop and wait for current executions to complete
	close(s.stopChan)

	// Stop cron scheduler
	cronCtx := s.cron.Stop()

	// Wait for current executions with timeout
	select {
	case <-cronCtx.Done():
		log.Info("All cron jobs completed")
	case <-time.After(30 * time.Second):
		log.Warn("Timeout waiting for cron jobs to complete")
	}

	// Wait for running executions to complete
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		s.runningMutex.RLock()
		running := s.runningCount
		s.runningMutex.RUnlock()

		if running == 0 {
			break
		}

		select {
		case <-timeout:
			log.WithField("running_executions", running).Warn("Timeout waiting for executions to complete")
			goto stopComplete
		case <-ticker.C:
			log.WithField("running_executions", running).Debug("Waiting for executions to complete")
		}
	}

stopComplete:
	s.isRunning = false
	s.activeSchedules = make(map[string]*ScheduleContext)

	log.Info("✅ Scheduler service stopped")
	return nil
}

// registerSchedule adds a schedule to the cron scheduler
func (s *SchedulerService) registerSchedule(schedule *database.ReplicationSchedule) error {
	if schedule.ScheduleType != "cron" {
		return fmt.Errorf("only cron schedules supported currently, got: %s", schedule.ScheduleType)
	}

	log.WithFields(log.Fields{
		"schedule_id":   schedule.ID,
		"schedule_name": schedule.Name,
		"cron_expr":     schedule.CronExpression,
	}).Info("Registering schedule")

	// Create execution function with context_id operations
	executionFunc := func() {
		ctx := context.Background()
		s.executeSchedule(ctx, schedule.ID)
	}

	// Add to cron with timezone support
	entryID, err := s.cron.AddFunc(schedule.CronExpression, executionFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job for schedule %s: %w", schedule.ID, err)
	}

	// Create schedule context
	scheduleCtx := &ScheduleContext{
		ScheduleID:  schedule.ID,
		Schedule:    schedule,
		CronEntryID: entryID,
		IsRunning:   false,
		JobCount:    0,
	}

	s.activeSchedules[schedule.ID] = scheduleCtx

	log.WithFields(log.Fields{
		"schedule_id": schedule.ID,
		"entry_id":    entryID,
	}).Info("Successfully registered schedule")

	return nil
}

// ReloadSchedules dynamically reloads schedules from database
// Adds new enabled schedules and removes disabled ones without service restart
func (s *SchedulerService) ReloadSchedules(ctx context.Context) error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler service not running")
	}

	log.Info("🔄 Reloading schedules dynamically")

	// Start job tracking for reload operation
	ctx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "scheduler",
		Operation: "dynamic-reload",
		Owner:     stringPtr("scheduler-service"),
	})
	if err != nil {
		return fmt.Errorf("failed to start reload job tracking: %w", err)
	}

	logger := s.jobTracker.Logger(ctx)

	// Get current enabled schedules from database
	dbSchedules, err := s.repository.ListSchedules(true) // enabledOnly = true
	if err != nil {
		s.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
		return fmt.Errorf("failed to load schedules: %w", err)
	}

	logger.Info("Retrieved schedules from database", "enabled_count", len(dbSchedules))

	// Create maps for comparison
	dbScheduleMap := make(map[string]*database.ReplicationSchedule)
	for i := range dbSchedules {
		dbScheduleMap[dbSchedules[i].ID] = &dbSchedules[i]
	}

	// Track changes
	added := 0
	removed := 0
	kept := 0

	// Remove schedules that are no longer enabled or don't exist
	for scheduleID, scheduleCtx := range s.activeSchedules {
		if _, exists := dbScheduleMap[scheduleID]; !exists {
			// Schedule no longer enabled or deleted - remove from cron
			logger.Info("Removing disabled/deleted schedule", "schedule_id", scheduleID, "entry_id", scheduleCtx.CronEntryID)
			s.cron.Remove(scheduleCtx.CronEntryID)
			delete(s.activeSchedules, scheduleID)
			removed++
		} else {
			kept++
		}
	}

	// Add new enabled schedules
	for _, schedule := range dbSchedules {
		if _, exists := s.activeSchedules[schedule.ID]; !exists {
			// New schedule to register
			logger.Info("Adding new enabled schedule", "schedule_id", schedule.ID, "name", schedule.Name)
			if err := s.registerSchedule(&schedule); err != nil {
				logger.Error("Failed to register new schedule", "error", err, "schedule_id", schedule.ID)
				continue
			}
			added++
		}
	}

	logger.Info("Schedule reload completed",
		"added", added,
		"removed", removed,
		"kept", kept,
		"total_active", len(s.activeSchedules))

	s.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
	return nil
}

// unregisterSchedule removes a schedule from the cron scheduler
func (s *SchedulerService) unregisterSchedule(scheduleID string) error {
	scheduleCtx, exists := s.activeSchedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found in active schedules: %s", scheduleID)
	}

	log.WithFields(log.Fields{
		"schedule_id": scheduleID,
		"entry_id":    scheduleCtx.CronEntryID,
	}).Info("Unregistering schedule")

	// Remove from cron
	s.cron.Remove(scheduleCtx.CronEntryID)

	// Remove from active schedules
	delete(s.activeSchedules, scheduleID)

	log.WithField("schedule_id", scheduleID).Info("Successfully unregistered schedule")
	return nil
}

// executeSchedule runs a scheduled execution using VM context_ids
func (s *SchedulerService) executeSchedule(ctx context.Context, scheduleID string) {
	// ✅ EARLY ENABLED CHECK - Prevent job tracker creation for disabled schedules
	schedule, err := s.repository.GetScheduleByID(scheduleID)
	if err != nil {
		log.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to get schedule for execution")
		return
	}
	if schedule == nil || !schedule.Enabled {
		log.WithField("schedule_id", scheduleID).Warn("⚠️ Schedule is disabled or not found, skipping execution")
		return
	}

	// Check concurrency limits
	s.runningMutex.Lock()
	if s.runningCount >= s.maxConcurrent {
		s.runningMutex.Unlock()
		log.WithField("schedule_id", scheduleID).Warn("⚠️ Max concurrent executions reached, skipping")
		return
	}
	s.runningCount++
	s.runningMutex.Unlock()

	defer func() {
		s.runningMutex.Lock()
		s.runningCount--
		s.runningMutex.Unlock()
	}()

	// Start job tracking for this execution (only after enabled check)
	ctx, executionJobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "scheduler",
		Operation: "schedule-execution",
		Owner:     stringPtr("scheduler-service"),
		Metadata: map[string]interface{}{
			"schedule_id": scheduleID,
		},
	})
	if err != nil {
		log.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to start execution job tracking")
		return
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("🎯 Starting scheduled execution", "schedule_id", scheduleID)

	// Execute the schedule
	summary, err := s.runScheduleExecution(ctx, scheduleID)
	if err != nil {
		logger.Error("❌ Schedule execution failed", "error", err)
		s.jobTracker.EndJob(ctx, executionJobID, joblog.StatusFailed, err)
		return
	}

	logger.Info("✅ Schedule execution completed",
		"execution_id", summary.ExecutionID,
		"vms_eligible", summary.VMsEligible,
		"jobs_created", summary.JobsCreated,
		"jobs_skipped", summary.JobsSkipped,
		"execution_time", summary.ExecutionTime,
	)

	s.jobTracker.EndJob(ctx, executionJobID, joblog.StatusCompleted, nil)
}

// runScheduleExecution performs the actual schedule execution logic
func (s *SchedulerService) runScheduleExecution(ctx context.Context, scheduleID string) (*ExecutionSummary, error) {
	startTime := time.Now()

	// Get schedule details
	schedule, err := s.repository.GetScheduleByID(scheduleID, "Groups", "Groups.Memberships", "Groups.Memberships.VMContext")
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if !schedule.Enabled {
		return nil, fmt.Errorf("schedule is disabled: %s", scheduleID)
	}

	logger := s.jobTracker.Logger(ctx)
	logger.Info("Executing schedule",
		"schedule_name", schedule.Name,
		"groups_count", len(schedule.Groups),
	)

	// Create schedule execution record
	execution := &database.ScheduleExecution{
		ScheduleID:  scheduleID,
		ScheduledAt: startTime,
		StartedAt:   &startTime,
		Status:      "running",
		TriggeredBy: "scheduler-service",
	}

	if err := s.repository.CreateScheduleExecution(execution); err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	summary := &ExecutionSummary{
		ExecutionID:         execution.ID,
		ScheduleID:          scheduleID,
		Status:              "running",
		StartedAt:           time.Now().UTC(),
		VMsEligible:         0,
		VMsProcessed:        0,
		JobsCreated:         0,
		JobsCompleted:       0,
		JobsFailed:          0,
		JobsSkipped:         0,
		VMContextsProcessed: make([]string, 0),
		CreatedJobIDs:       make([]string, 0),
		Summary:             make(map[string]interface{}),
	}

	// Process each group in the schedule
	for _, group := range schedule.Groups {
		groupSummary, err := s.executeGroup(ctx, execution, &group, schedule)
		if err != nil {
			logger.Error("Group execution failed", "error", err, "group_id", group.ID)
			continue
		}

		// Aggregate results
		summary.VMsEligible += groupSummary.VMsEligible
		summary.VMsProcessed += groupSummary.VMsProcessed
		summary.JobsCreated += groupSummary.JobsCreated
		summary.JobsCompleted += groupSummary.JobsCompleted
		summary.JobsFailed += groupSummary.JobsFailed
		summary.JobsSkipped += groupSummary.JobsSkipped
		summary.VMContextsProcessed = append(summary.VMContextsProcessed, groupSummary.VMContextsProcessed...)
		summary.CreatedJobIDs = append(summary.CreatedJobIDs, groupSummary.CreatedJobIDs...)
	}

	// Complete execution
	endTime := time.Now()
	summary.ExecutionTime = endTime.Sub(startTime)
	summary.Status = "completed"

	// Update execution record
	executionDetails, _ := json.Marshal(summary)
	updates := map[string]interface{}{
		"completed_at":               endTime,
		"status":                     "completed",
		"vms_eligible":               summary.VMsEligible,
		"jobs_created":               summary.JobsCreated,
		"jobs_completed":             summary.JobsCompleted,
		"jobs_failed":                summary.JobsFailed,
		"jobs_skipped":               summary.JobsSkipped,
		"execution_duration_seconds": int(summary.ExecutionTime.Seconds()),
		"execution_details":          string(executionDetails),
	}

	if err := s.repository.UpdateScheduleExecution(execution.ID, updates); err != nil {
		logger.Error("Failed to update execution record", "error", err)
	}

	return summary, nil
}

// executeGroup processes all VMs in a group using context_id
func (s *SchedulerService) executeGroup(ctx context.Context, execution *database.ScheduleExecution, group *database.VMMachineGroup, schedule *database.ReplicationSchedule) (*ExecutionSummary, error) {
	logger := s.jobTracker.Logger(ctx)
	logger.Info("Processing machine group",
		"group_id", group.ID,
		"group_name", group.Name,
		"vms_count", len(group.Memberships),
	)

	// Get enabled memberships ordered by priority
	memberships, err := s.repository.GetGroupMemberships(group.ID, true) // enabledOnly = true
	if err != nil {
		return nil, fmt.Errorf("failed to get group memberships: %w", err)
	}

	summary := &ExecutionSummary{
		ExecutionID:         execution.ID,
		ScheduleID:          execution.ScheduleID,
		GroupID:             &group.ID,
		Status:              "running",
		StartedAt:           time.Now().UTC(),
		VMsEligible:         len(memberships),
		VMsProcessed:        0,
		JobsCreated:         0,
		JobsCompleted:       0,
		JobsFailed:          0,
		JobsSkipped:         0,
		VMContextsProcessed: make([]string, 0),
		CreatedJobIDs:       make([]string, 0),
		Summary:             make(map[string]interface{}),
	}

	// Extract VM contexts for conflict detection
	vmContexts := make([]*database.VMReplicationContext, 0, len(memberships))
	membershipMap := make(map[string]*database.VMGroupMembership)

	for _, membership := range memberships {
		if membership.VMContext == nil {
			logger.Warn("VM context not loaded, skipping", "vm_context_id", membership.VMContextID)
			summary.JobsSkipped++
			continue
		}
		vmContexts = append(vmContexts, membership.VMContext)
		membershipMap[membership.VMContext.ContextID] = &membership
	}

	// Prepare constraint objects for conflict detection
	scheduleConstraints := &ScheduleConstraints{
		ScheduleID:        execution.ScheduleID,
		SkipIfRunning:     schedule.SkipIfRunning,
		MaxConcurrentJobs: schedule.MaxConcurrentJobs,
		Enabled:           schedule.Enabled,
	}

	groupConstraints := &GroupConstraints{
		GroupID:          group.ID,
		MaxConcurrentVMs: group.MaxConcurrentVMs,
		Priority:         group.Priority,
	}

	// Run conflict detection on all VMs
	conflictSummary, err := s.conflictDetector.CheckVMConflicts(ctx, vmContexts, scheduleConstraints, groupConstraints)
	if err != nil {
		logger.Error("Failed to run conflict detection", "error", err)
		// Fall back to processing all VMs without conflict detection
	} else {
		logger.Info("Conflict detection completed",
			"total_vms", conflictSummary.TotalVMs,
			"eligible", conflictSummary.EligibleVMs,
			"conflicted", conflictSummary.ConflictedVMs,
		)
	}

	// Process VMs based on conflict detection results
	for i, vmCtx := range vmContexts {
		membership := membershipMap[vmCtx.ContextID]

		logger.Info("Processing VM",
			"vm_context_id", vmCtx.ContextID,
			"vm_name", vmCtx.VMName,
			"priority", membership.Priority,
		)

		// Check conflict detection results
		var canSchedule bool = true
		var skipReason string = ""

		if conflictSummary != nil && i < len(conflictSummary.Results) {
			result := conflictSummary.Results[i]
			canSchedule = result.CanSchedule && !result.HasConflict
			if !canSchedule {
				skipReason = result.ConflictReason
			}
		}

		// Skip if conflicts detected
		if !canSchedule {
			logger.Info("Skipping VM due to conflict",
				"vm_context_id", vmCtx.ContextID,
				"skip_reason", skipReason,
			)
			summary.JobsSkipped++
			summary.VMsProcessed++
			summary.VMContextsProcessed = append(summary.VMContextsProcessed, vmCtx.ContextID)
			continue
		}

		// Create replication job for this VM
		jobID, err := s.createReplicationJob(ctx, execution, group, vmCtx, schedule)
		if err != nil {
			logger.Error("Failed to create replication job",
				"error", err,
				"vm_context_id", vmCtx.ContextID,
				"vm_name", vmCtx.VMName,
			)
			summary.JobsFailed++
		} else {
			logger.Info("Successfully created replication job",
				"vm_context_id", vmCtx.ContextID,
				"job_id", jobID,
			)
			summary.JobsCreated++
			summary.CreatedJobIDs = append(summary.CreatedJobIDs, jobID)
		}

		summary.VMsProcessed++
		summary.VMContextsProcessed = append(summary.VMContextsProcessed, vmCtx.ContextID)

		// Respect group concurrency limits
		if summary.JobsCreated >= group.MaxConcurrentVMs {
			logger.Info("Reached group concurrency limit", "max_concurrent", group.MaxConcurrentVMs)
			break
		}
	}

	logger.Info("Completed group processing",
		"group_id", group.ID,
		"vms_eligible", summary.VMsEligible,
		"jobs_created", summary.JobsCreated,
		"jobs_skipped", summary.JobsSkipped,
	)

	return summary, nil
}

// ✅ ALIGNED: createReplicationJob now uses EXACT same workflow as GUI
// 1. Fresh SNA discovery API call for latest VM specifications
// 2. SHA replication API call using same endpoint as GUI
// 3. Let Migration Engine handle VM context updates automatically
func (s *SchedulerService) createReplicationJob(
	ctx context.Context,
	execution *database.ScheduleExecution,
	group *database.VMMachineGroup,
	vmCtx *database.VMReplicationContext,
	schedule *database.ReplicationSchedule,
) (string, error) {
	logger := s.jobTracker.Logger(ctx)

	logger.Info("Starting aligned replication job creation (same workflow as GUI)",
		"vm_context_id", vmCtx.ContextID,
		"vm_name", vmCtx.VMName,
		"vcenter_host", vmCtx.VCenterHost,
		"schedule_id", schedule.ID,
		"execution_id", execution.ID,
	)

	// ✅ STEP 1: Fresh VM Discovery (CRITICAL ALIGNMENT WITH GUI)
	// Always get latest VM specifications from vCenter before job creation
	discoveredVM, err := s.discoverVMFromVMA(ctx, vmCtx.VMName, vmCtx.VCenterHost, vmCtx.Datacenter)
	if err != nil {
		logger.Error("Fresh VM discovery failed", "error", err)
		return "", fmt.Errorf("discovery failed for VM %s: %w", vmCtx.VMName, err)
	}

	logger.Info("Successfully discovered fresh VM data",
		"vm_name", discoveredVM.Name,
		"vm_id", discoveredVM.ID,
		"cpus", discoveredVM.NumCPU,
		"memory_mb", discoveredVM.MemoryMB,
		"disk_count", len(discoveredVM.Disks),
		"power_state", discoveredVM.PowerState)

	// ✅ STEP 2: Transform to SHA API format (EXACT field mapping as GUI)
	// Use fresh discovery data instead of stale database data
	shaRequest := CreateMigrationRequest{
		SourceVM: models.VMInfo{
			// ✅ EXACT FIELD MAPPING (from GUI workflow documentation)
			ID:         discoveredVM.ID,
			Name:       discoveredVM.Name,
			Path:       discoveredVM.Path,
			Datacenter: discoveredVM.Datacenter,
			CPUs:       discoveredVM.NumCPU,     // ✅ Fresh from vCenter
			MemoryMB:   discoveredVM.MemoryMB,   // ✅ Fresh from vCenter
			PowerState: discoveredVM.PowerState, // ✅ Fresh from vCenter
			OSType:     discoveredVM.GuestOS,    // ✅ Fresh from vCenter
			Disks:      discoveredVM.Disks,      // ✅ CRITICAL: Fresh disk specs
			Networks:   discoveredVM.Networks,   // ✅ Fresh network config
		},
		OSSEAConfigID:   s.getActiveOSSEAConfigID(ctx), // Dynamic lookup
		ReplicationType: schedule.ReplicationType,
		TargetNetwork:   "default", // TODO: Add target network to schedule config
		VCenterHost:     vmCtx.VCenterHost,
		Datacenter:      vmCtx.Datacenter,
		// CBT fields (let backend determine incremental vs initial)
		ChangeID:         "",
		PreviousChangeID: "",
		SnapshotID:       "",
		// ✅ NEW: Scheduler metadata (passed to Migration Engine via SHA API)
		ScheduleExecutionID: execution.ID,
		VMGroupID:           group.ID,
		ScheduledBy:         "scheduler-service",
	}

	// ✅ STEP 3: Call SHA API (SAME endpoint and workflow as GUI)
	// This ensures identical behavior between manual and scheduled jobs
	result, err := s.callOMAReplicationAPI(ctx, shaRequest)
	if err != nil {
		logger.Error("SHA replication API call failed", "error", err)
		return "", fmt.Errorf("failed to start replication via SHA API: %w", err)
	}

	logger.Info("Successfully started replication via SHA API (aligned with GUI)",
		"job_id", result.JobID,
		"status", result.Status,
		"progress_percent", result.ProgressPercent,
		"disks_count", len(result.Disks),
		"vm_context_id", vmCtx.ContextID)

	// ✅ STEP 4: Scheduler metadata now handled by Migration Engine
	// No more direct database updates - metadata passed via SHA API request
	logger.Info("Scheduler metadata passed to Migration Engine via SHA API",
		"job_id", result.JobID,
		"schedule_execution_id", execution.ID,
		"vm_group_id", group.ID,
		"scheduled_by", "scheduler-service")

	logger.Info("Completed aligned replication job creation",
		"job_id", result.JobID,
		"vm_context_id", vmCtx.ContextID,
		"vm_name", vmCtx.VMName,
		"fresh_cpus", discoveredVM.NumCPU,
		"fresh_memory_mb", discoveredVM.MemoryMB,
		"fresh_disk_count", len(discoveredVM.Disks),
		"alignment", "✅ SAME_AS_GUI")

	return result.JobID, nil
}

// ✅ REMOVED STALE DATABASE HELPERS
// The following methods were removed as part of GUI workflow alignment:
// - valueOrDefault() - No longer needed (using fresh discovery data)
// - stringPtrToString() - No longer needed (using fresh discovery data)
// - getVMDisksForContext() - REPLACED by fresh SNA discovery API calls
//
// All VM specifications now come from fresh vCenter discovery, not stale database data

// GetActiveSchedules returns currently registered schedules
func (s *SchedulerService) GetActiveSchedules() map[string]*ScheduleContext {
	s.runningMutex.RLock()
	defer s.runningMutex.RUnlock()

	// Create copy to avoid race conditions
	result := make(map[string]*ScheduleContext)
	for k, v := range s.activeSchedules {
		result[k] = v
	}
	return result
}

// GetServiceStatus returns current service status
func (s *SchedulerService) GetServiceStatus() map[string]interface{} {
	s.runningMutex.RLock()
	defer s.runningMutex.RUnlock()

	return map[string]interface{}{
		"is_running":         s.isRunning,
		"active_schedules":   len(s.activeSchedules),
		"running_executions": s.runningCount,
		"max_concurrent":     s.maxConcurrent,
	}
}

// ScanForPhantomJobs scans for phantom jobs using the integrated detector
func (s *SchedulerService) ScanForPhantomJobs(ctx context.Context) (*PhantomScanSummary, error) {
	return s.phantomDetector.ScanForPhantomJobs(ctx)
}

// CleanupPhantomJobs scans for and automatically cleans up phantom jobs
func (s *SchedulerService) CleanupPhantomJobs(ctx context.Context) (*PhantomScanSummary, error) {
	// Scan for phantom jobs
	summary, err := s.phantomDetector.ScanForPhantomJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan for phantom jobs: %w", err)
	}

	// Mark phantom jobs as failed if any were found
	if len(summary.PhantomJobIDs) > 0 {
		if err := s.phantomDetector.MarkPhantomJobsAsFailed(ctx, summary.PhantomJobIDs); err != nil {
			return summary, fmt.Errorf("failed to mark phantom jobs as failed: %w", err)
		}
	}

	return summary, nil
}

// CheckVMConflicts exposes conflict detection for external use
func (s *SchedulerService) CheckVMConflicts(
	ctx context.Context,
	vmContexts []*database.VMReplicationContext,
	scheduleID string,
	groupID string,
) (*ConflictScanSummary, error) {
	// Get schedule and group details
	schedule, err := s.repository.GetScheduleByID(scheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	group, err := s.repository.GetGroupByID(groupID)
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

	return s.conflictDetector.CheckVMConflicts(ctx, vmContexts, scheduleConstraints, groupConstraints)
}

// CheckSingleVMConflict checks conflicts for a single VM
func (s *SchedulerService) CheckSingleVMConflict(
	ctx context.Context,
	vmContextID string,
	scheduleID string,
	groupID string,
) (*ConflictResult, error) {
	return s.conflictDetector.CheckSingleVMConflict(ctx, vmContextID, scheduleID, groupID)
}

// TriggerManualExecution manually triggers a schedule execution
func (s *SchedulerService) TriggerManualExecution(
	ctx context.Context,
	scheduleID string,
	triggeredBy string,
	reason *string,
) (*ExecutionSummary, error) {
	logger := s.jobTracker.Logger(ctx)
	logger.Info("Triggering manual schedule execution", "schedule_id", scheduleID, "triggered_by", triggeredBy)

	// Get schedule to validate it exists and is enabled
	schedule, err := s.repository.GetScheduleByID(scheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	if schedule == nil {
		return nil, fmt.Errorf("schedule not found")
	}
	if !schedule.Enabled {
		return nil, fmt.Errorf("schedule is disabled")
	}

	// Create a manual execution record
	executionID := fmt.Sprintf("manual-%d", time.Now().Unix())
	startedAt := time.Now().UTC()
	execution := &database.ScheduleExecution{
		ID:            executionID,
		ScheduleID:    scheduleID,
		Status:        "running",
		ScheduledAt:   time.Now().UTC(),
		StartedAt:     &startedAt,
		JobsCreated:   0,
		JobsCompleted: 0,
		JobsFailed:    0,
		JobsSkipped:   0,
		VMsEligible:   0,
		TriggeredBy:   triggeredBy,
	}

	// Add reason to execution details if provided
	if reason != nil {
		reasonDetails := fmt.Sprintf(`{"trigger_reason": "%s"}`, *reason)
		execution.ExecutionDetails = &reasonDetails
	}

	if err := s.repository.CreateScheduleExecution(execution); err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	logger.Info("Created manual execution record", "execution_id", executionID)

	// Execute the schedule using the existing runScheduleExecution logic
	summary, err := s.runScheduleExecution(ctx, scheduleID)
	if err != nil {
		// Update execution with error
		s.repository.UpdateScheduleExecution(executionID, map[string]interface{}{
			"status":        "failed",
			"error_message": err.Error(),
			"completed_at":  time.Now().UTC(),
			"updated_at":    time.Now().UTC(),
		})

		return nil, fmt.Errorf("manual execution failed: %w", err)
	}

	// Update execution with success results
	completedAt := time.Now().UTC()
	duration := int(completedAt.Sub(*execution.StartedAt).Seconds())

	s.repository.UpdateScheduleExecution(executionID, map[string]interface{}{
		"status":                     "completed",
		"completed_at":               completedAt,
		"jobs_created":               summary.JobsCreated,
		"jobs_completed":             summary.JobsCompleted,
		"jobs_failed":                summary.JobsFailed,
		"jobs_skipped":               summary.JobsSkipped,
		"vms_eligible":               summary.VMsEligible,
		"execution_duration_seconds": duration,
	})

	// Update the summary with execution details
	summary.ExecutionID = executionID
	summary.StartedAt = *execution.StartedAt

	logger.Info("Manual execution completed successfully",
		"execution_id", executionID,
		"jobs_created", summary.JobsCreated,
		"vms_processed", summary.VMsProcessed)

	return summary, nil
}

// ✅ SNA DISCOVERY INTEGRATION (aligned with GUI workflow)
// These methods implement the exact same workflow as the GUI for consistency

// discoverVMFromVMA calls SNA discovery API to get fresh VM specifications
// This ensures we always have the latest VM data from vCenter before job creation
func (s *SchedulerService) discoverVMFromVMA(ctx context.Context, vmName, vCenterHost, datacenter string) (*VMDiscoveryData, error) {
	logger := s.jobTracker.Logger(ctx)

	// Create discovery request (same format as GUI)
	// 🆕 ENHANCED: Use credential service for scheduler operations
	var username, password string

	// Get credentials from credential service
	// Note: SchedulerService needs database access for credential service
	// For now, falling back to hardcoded until service architecture is enhanced
	// TODO: Modify SchedulerService constructor to include database connection
	logger.Warn("Credential service integration pending for SchedulerService - using fallback credentials")
	username = "administrator@vsphere.local"
	password = "EmyGVoBFesGQc47-"

	discoveryRequest := VMDiscoveryRequest{
		VCenter:    vCenterHost,
		Username:   username, // Using credential service
		Password:   password, // Using credential service
		Datacenter: datacenter,
		Filter:     vmName, // Get specific VM only
	}

	jsonData, err := json.Marshal(discoveryRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal discovery request: %w", err)
	}

	// Call SNA discovery API (same endpoint as GUI)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.snaAPIEndpoint+"/api/v1/discover", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	logger.Info("Calling SNA discovery API for fresh VM data",
		"vm_name", vmName,
		"vcenter_host", vCenterHost,
		"datacenter", datacenter,
		"endpoint", s.snaAPIEndpoint+"/api/v1/discover")

	resp, err := s.snaClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call SNA discovery API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SNA discovery API returned status %d", resp.StatusCode)
	}

	var discoveryResponse VMDiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&discoveryResponse); err != nil {
		return nil, fmt.Errorf("failed to decode discovery response: %w", err)
	}

	// Find the specific VM in discovery results
	if len(discoveryResponse.VMs) == 0 {
		return nil, fmt.Errorf("VM %s not found in discovery results", vmName)
	}

	discoveredVM := &discoveryResponse.VMs[0] // Should be filtered to single VM
	if discoveredVM.Name != vmName {
		return nil, fmt.Errorf("discovered VM name %s does not match requested VM %s", discoveredVM.Name, vmName)
	}

	// Validate disk information (critical for migration)
	if len(discoveredVM.Disks) == 0 {
		return nil, fmt.Errorf("VM %s has no disks configured", vmName)
	}

	logger.Info("Successfully discovered fresh VM data",
		"vm_name", discoveredVM.Name,
		"vm_id", discoveredVM.ID,
		"cpus", discoveredVM.NumCPU,
		"memory_mb", discoveredVM.MemoryMB,
		"disk_count", len(discoveredVM.Disks),
		"network_count", len(discoveredVM.Networks))

	return discoveredVM, nil
}

// callOMAReplicationAPI calls SHA replication API using the same endpoint as GUI
// This ensures consistent behavior between manual GUI jobs and scheduled jobs
func (s *SchedulerService) callOMAReplicationAPI(ctx context.Context, req CreateMigrationRequest) (*MigrationResult, error) {
	logger := s.jobTracker.Logger(ctx)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal replication request: %w", err)
	}

	// Call same API endpoint as GUI
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.shaAPIEndpoint+"/api/v1/replications", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create replication request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer sess_longlived_dev_token_2025_2035_permanent") // Same auth as GUI

	logger.Info("Calling SHA replication API",
		"vm_name", req.SourceVM.Name,
		"replication_type", req.ReplicationType,
		"endpoint", s.shaAPIEndpoint+"/api/v1/replications")

	resp, err := s.shaClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call SHA replication API: %w", err)
	}
	defer resp.Body.Close()

	var result MigrationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode replication response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("SHA replication API returned status %d", resp.StatusCode)
	}

	logger.Info("Successfully started replication via SHA API",
		"job_id", result.JobID,
		"status", result.Status,
		"progress_percent", result.ProgressPercent,
		"disks_count", len(result.Disks))

	return &result, nil
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}

// getActiveOSSEAConfigID retrieves the active OSSEA config ID from database
func (s *SchedulerService) getActiveOSSEAConfigID(ctx context.Context) int {
	// Use hardcoded value for now - SchedulerService doesn't have direct database access
	// TODO: Add proper database connection to SchedulerService or pass config ID as parameter
	return 1 // Default fallback - should be updated when proper database access is added
}
