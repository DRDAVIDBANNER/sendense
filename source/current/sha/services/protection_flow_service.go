package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
)

// =============================================================================
// PROTECTION FLOW SERVICE - Flow Orchestration and Execution
// =============================================================================
// Provides unified abstraction for backup and replication operations
// Integrates with existing scheduler, backup engine, and group management

// ProtectionFlowService orchestrates protection flow execution
type ProtectionFlowService struct {
	flowRepo        *database.FlowRepository
	scheduleService *SchedulerService
	machineGroupSvc *MachineGroupService
	vmContextRepo   *database.VMReplicationContextRepository
	jobTracker      *joblog.Tracker
	db              database.Connection

	// HTTP client for backup API calls
	backupAPIClient *http.Client
	backupAPIURL    string
}

// Flow execution request types
type CreateFlowRequest struct {
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
	FlowType     string  `json:"flow_type"`     // "backup" or "replication"
	TargetType   string  `json:"target_type"`   // "vm" or "group"
	TargetID     string  `json:"target_id"`     // context_id or group_id
	RepositoryID *string `json:"repository_id,omitempty"`
	PolicyID     *string `json:"policy_id,omitempty"`
	ScheduleID   *string `json:"schedule_id,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
}

// Flow status response
type FlowStatus struct {
	LastExecutionID     *string    `json:"last_execution_id,omitempty"`
	LastExecutionStatus string     `json:"last_execution_status"`
	LastExecutionTime   *time.Time `json:"last_execution_time,omitempty"`
	NextExecutionTime   *time.Time `json:"next_execution_time,omitempty"`
	TotalExecutions     int        `json:"total_executions"`
	SuccessfulExecutions int       `json:"successful_executions"`
	FailedExecutions    int       `json:"failed_executions"`
}

// NewProtectionFlowService creates a new protection flow service
func NewProtectionFlowService(
	flowRepo *database.FlowRepository,
	scheduleService *SchedulerService,
	machineGroupSvc *MachineGroupService,
	vmContextRepo *database.VMReplicationContextRepository,
	jobTracker *joblog.Tracker,
	db database.Connection,
	backupAPIURL string,
) *ProtectionFlowService {
	return &ProtectionFlowService{
		flowRepo:        flowRepo,
		scheduleService: scheduleService,
		machineGroupSvc: machineGroupSvc,
		vmContextRepo:   vmContextRepo,
		jobTracker:      jobTracker,
		db:              db,
		backupAPIClient: &http.Client{Timeout: 30 * time.Second},
		backupAPIURL:    backupAPIURL,
	}
}

// SetSchedulerService sets the scheduler service (resolves circular dependency)
func (s *ProtectionFlowService) SetSchedulerService(schedulerService *SchedulerService) {
	s.scheduleService = schedulerService
}

// SetMachineGroupService sets the machine group service
func (s *ProtectionFlowService) SetMachineGroupService(machineGroupSvc *MachineGroupService) {
	s.machineGroupSvc = machineGroupSvc
}

// =============================================================================
// FLOW CRUD OPERATIONS
// =============================================================================

// CreateFlow creates and validates a new protection flow
func (s *ProtectionFlowService) CreateFlow(ctx context.Context, req CreateFlowRequest) (*database.ProtectionFlow, error) {
	logger := s.jobTracker.Logger(ctx)
	logger.Info("Creating protection flow", "name", req.Name, "type", req.FlowType)

	// Validate request
	if err := s.validateFlowRequest(req); err != nil {
		logger.Error("Flow validation failed", "error", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check for name conflicts
	_, err := s.flowRepo.GetFlowByName(ctx, req.Name)
	if err == nil {
		return nil, fmt.Errorf("flow with name '%s' already exists", req.Name)
	}

	// Set defaults
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	// Create flow
	flow := &database.ProtectionFlow{
		Name:        req.Name,
		Description: req.Description,
		FlowType:    req.FlowType,
		TargetType:  req.TargetType,
		TargetID:    req.TargetID,
		RepositoryID: req.RepositoryID,
		PolicyID:     req.PolicyID,
		ScheduleID:   req.ScheduleID,
		Enabled:      enabled,
		LastExecutionStatus: "pending",
		CreatedBy:           "system", // TODO: Get from context
	}

	if err := s.flowRepo.CreateFlow(ctx, flow); err != nil {
		logger.Error("Failed to create flow in database", "error", err)
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}

	logger.Info("Protection flow created successfully", "flow_id", flow.ID)
	return flow, nil
}

// GetFlow retrieves a flow by ID
func (s *ProtectionFlowService) GetFlow(ctx context.Context, id string) (*database.ProtectionFlow, error) {
	return s.flowRepo.GetFlowByID(ctx, id)
}

// ListFlows retrieves flows with optional filtering
func (s *ProtectionFlowService) ListFlows(ctx context.Context, filters database.FlowFilters) ([]*database.ProtectionFlow, error) {
	return s.flowRepo.ListFlows(ctx, filters)
}

// UpdateFlow updates a flow configuration
func (s *ProtectionFlowService) UpdateFlow(ctx context.Context, id string, updates map[string]interface{}) error {
	logger := s.jobTracker.Logger(ctx)
	logger.Info("Updating protection flow", "flow_id", id)

	return s.flowRepo.UpdateFlow(ctx, id, updates)
}

// DeleteFlow deletes a flow
func (s *ProtectionFlowService) DeleteFlow(ctx context.Context, id string) error {
	logger := s.jobTracker.Logger(ctx)
	logger.Info("Deleting protection flow", "flow_id", id)

	// TODO: Unregister from scheduler if scheduled

	return s.flowRepo.DeleteFlow(ctx, id)
}

// =============================================================================
// FLOW EXECUTION
// =============================================================================

// ExecuteFlow executes a protection flow (manual or scheduled)
func (s *ProtectionFlowService) ExecuteFlow(ctx context.Context, flowID string, executionType string) (*database.ProtectionFlowExecution, error) {
	logger := s.jobTracker.Logger(ctx)

	// Start job tracking
	owner := "system"
	jobCtx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "scheduler",
		Operation: fmt.Sprintf("execute_%s_flow", executionType),
		Owner:     &owner,
		Metadata:  map[string]interface{}{"flow_id": flowID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start job tracking: %w", err)
	}

	logger = s.jobTracker.Logger(jobCtx)
	logger.Info("Starting flow execution", "flow_id", flowID, "execution_type", executionType)

	defer func() {
		s.jobTracker.EndJob(jobCtx, jobID, joblog.StatusCompleted, nil)
	}()

	// 1. Load flow with relationships
	flow, err := s.flowRepo.GetFlowByID(jobCtx, flowID)
	if err != nil {
		logger.Error("Failed to load flow", "error", err)
		s.jobTracker.RunStep(jobCtx, jobID, "load_flow", func(ctx context.Context) error {
			return fmt.Errorf("failed to load flow: %w", err)
		})
		return nil, fmt.Errorf("flow not found: %w", err)
	}

	// 2. Create execution record
	triggeredBy := "system" // TODO: Get from context
	execution := &database.ProtectionFlowExecution{
		FlowID:        flowID,
		Status:        "running",
		ExecutionType: executionType,
		StartedAt:     &time.Time{},
		TriggeredBy:   triggeredBy,
	}
	*execution.StartedAt = time.Now()

	if err := s.flowRepo.CreateExecution(jobCtx, execution); err != nil {
		logger.Error("Failed to create execution record", "error", err)
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	logger.Info("Execution record created", "execution_id", execution.ID)

	// 3. Route to appropriate handler
	var execErr error
	switch flow.FlowType {
	case "backup":
		execErr = s.ProcessBackupFlow(jobCtx, flow, execution)
	case "replication":
		execErr = s.ProcessReplicationFlow(jobCtx, flow, execution)
	default:
		execErr = fmt.Errorf("unknown flow type: %s", flow.FlowType)
	}

	// 4. Update execution status
	// ✅ FIX: If no error, jobs are RUNNING (not completed!)
	// Execution stays in "running" status until background monitor updates it
	if execErr != nil {
		// Only mark as error/failed if job creation failed
		finalStatus := "error"
		errorMsg := execErr.Error()
		execution.ErrorMessage = &errorMsg
		execution.Status = finalStatus
		completedAt := time.Now()
		execution.CompletedAt = &completedAt
		execution.ExecutionTimeSeconds = int(completedAt.Sub(*execution.StartedAt).Seconds())

		logger.Error("Flow execution failed to start", "error", execErr)

		updateErr := s.flowRepo.UpdateExecutionStatus(jobCtx, execution.ID, finalStatus, map[string]interface{}{
			"completed_at":            execution.CompletedAt,
			"execution_time_seconds": execution.ExecutionTimeSeconds,
			"error_message":          execution.ErrorMessage,
			"jobs_created":           execution.JobsCreated,
			"jobs_completed":         execution.JobsCompleted,
			"jobs_failed":            execution.JobsFailed,
			"jobs_skipped":           execution.JobsSkipped,
			"vms_processed":          execution.VMsProcessed,
			"bytes_transferred":      execution.BytesTransferred,
		})
		if updateErr != nil {
			logger.Error("Failed to update execution status", "error", updateErr)
		}
	} else {
		// ✅ FIX: Jobs started successfully - keep status="running"
		// Don't set completed_at or mark as success yet!
		// Background monitor will update when jobs actually complete
		updateErr := s.flowRepo.UpdateExecutionStatus(jobCtx, execution.ID, "running", map[string]interface{}{
			"jobs_created":      execution.JobsCreated,
			"jobs_completed":    execution.JobsCompleted,
			"jobs_failed":       execution.JobsFailed,
			"jobs_skipped":      execution.JobsSkipped,
			"vms_processed":     execution.VMsProcessed,
			"created_job_ids":   execution.CreatedJobIDs,
		})
		if updateErr != nil {
			logger.Error("Failed to update execution status", "error", updateErr)
		}

		logger.Info("Flow execution started successfully - jobs running in background",
			"execution_id", execution.ID,
			"jobs_created", execution.JobsCreated,
			"status", "running")
	}

	// 5. Update flow statistics
	// ✅ FIX: Only update success/failed counts when execution actually completes
	// For running executions, just update last_execution_id and status
	statsErr := s.flowRepo.UpdateFlowStatistics(jobCtx, flowID, database.FlowStatistics{
		LastExecutionID:     &execution.ID,
		LastExecutionStatus: execution.Status,  // Use execution.Status (not finalStatus)
		LastExecutionTime:   execution.CompletedAt,  // Will be nil for running executions
		TotalExecutions:     flow.TotalExecutions + 1,
		SuccessfulExecutions: flow.SuccessfulExecutions + func() int {
			if execution.Status == "success" {
				return 1
			}
			return 0
		}(),
		FailedExecutions: flow.FailedExecutions + func() int {
			if execution.Status == "error" {
				return 1
			}
			return 0
		}(),
	})
	if statsErr != nil {
		logger.Error("Failed to update flow statistics", "error", statsErr)
	}

	logger.Info("Flow execution API call completed",
		"execution_id", execution.ID,
		"status", execution.Status,  // ✅ FIX: Use execution.Status
		"jobs_created", execution.JobsCreated,
		"vms_processed", execution.VMsProcessed)

	return execution, nil
}

// ProcessBackupFlow executes a backup-type flow
func (s *ProtectionFlowService) ProcessBackupFlow(ctx context.Context, flow *database.ProtectionFlow, execution *database.ProtectionFlowExecution) error {
	logger := s.jobTracker.Logger(ctx)
	logger.Info("Processing backup flow", "flow_id", flow.ID, "target_type", flow.TargetType)

	// 1. Resolve target VMs
	var vmContexts []string
	switch flow.TargetType {
	case "vm":
		vmContexts = []string{flow.TargetID}
	case "group":
		groupSummary, err := s.machineGroupSvc.GetGroup(ctx, flow.TargetID)
		if err != nil {
			return fmt.Errorf("failed to get group: %w", err)
		}
		for _, membership := range groupSummary.Memberships {
			if membership.Enabled {
				vmContexts = append(vmContexts, membership.VMContextID)
			}
		}
	default:
		return fmt.Errorf("unsupported target type: %s", flow.TargetType)
	}

	logger.Info("Resolved target VMs", "count", len(vmContexts), "contexts", vmContexts)

	if len(vmContexts) == 0 {
		logger.Warn("No VMs to process")
		return nil
	}

	// 2. Execute backup for each VM
	var createdJobIDs []string
	var jobsFailed, jobsSkipped int  // ✅ FIX: Removed jobsCompleted (not used - jobs run in background)
	var totalBytes int64

	for _, contextID := range vmContexts {
		// Query VM context directly since no GetByContextID method exists
		var vmCtx database.VMReplicationContext
		if err := s.db.GetGormDB().Where("context_id = ?", contextID).First(&vmCtx).Error; err != nil {
			logger.Error("Failed to load VM context", "context_id", contextID, "error", err)
			jobsSkipped++
			continue
		}

		// Determine backup type: check if COMPLETED full backup exists for this VM
		// CRITICAL: Only count backups with status='completed' - failed/running backups don't have valid change IDs
		backupType := "incremental"
		var existingBackup database.BackupJob
		if err := s.db.GetGormDB().Where("vm_name = ? AND repository_id = ? AND backup_type = ? AND status = ?", 
			vmCtx.VMName, *flow.RepositoryID, "full", "completed").First(&existingBackup).Error; err != nil {
			// No completed full backup exists, must do full backup first
			backupType = "full"
			logger.Info("No completed full backup found, will perform full backup", "vm_name", vmCtx.VMName)
		}

		// Call backup API
		backupResp, err := s.startBackup(ctx, &BackupStartRequest{
			VMName:       vmCtx.VMName,
			RepositoryID: *flow.RepositoryID,
			BackupType:   backupType,
			PolicyID:     stringPtrToString(flow.PolicyID),
		})
		if err != nil {
			logger.Error("Failed to start backup", "vm_name", vmCtx.VMName, "error", err)
			jobsFailed++
			continue
		}

		createdJobIDs = append(createdJobIDs, backupResp.BackupID)
		// ❌ REMOVED: jobsCompleted++ (backup just STARTED, not completed!)
		totalBytes += backupResp.TotalBytes

		logger.Info("Backup started successfully",
			"vm_name", vmCtx.VMName,
			"backup_id", backupResp.BackupID,
			"bytes", backupResp.TotalBytes)
	}

	// 3. Update execution with results
	execution.JobsCreated = len(createdJobIDs)
	// ✅ FIX: Don't set jobsCompleted here - jobs are still running!
	// Jobs will be counted as completed by background monitor
	execution.JobsCompleted = 0  // No jobs completed yet
	execution.JobsFailed = jobsFailed
	execution.JobsSkipped = jobsSkipped
	execution.VMsProcessed = len(vmContexts)
	execution.BytesTransferred = 0  // Will be updated when jobs complete

	// Store created job IDs as JSON
	if len(createdJobIDs) > 0 {
		jobIDsJSON, _ := json.Marshal(createdJobIDs)
		jobIDsStr := string(jobIDsJSON)
		execution.CreatedJobIDs = &jobIDsStr
	}

	if jobsFailed > 0 {
		return fmt.Errorf("%d of %d backups failed to start", jobsFailed, len(vmContexts))
	}

	logger.Info("Backup flow jobs created (running in background)",
		"jobs_created", execution.JobsCreated,
		"jobs_running", len(createdJobIDs))

	// ✅ FIX: Return nil to indicate jobs started successfully (not completed!)
	// Background monitor will update execution status when jobs actually complete
	return nil
}

// ProcessReplicationFlow executes a replication-type flow (Phase 5 placeholder)
func (s *ProtectionFlowService) ProcessReplicationFlow(ctx context.Context, flow *database.ProtectionFlow, execution *database.ProtectionFlowExecution) error {
	logger := s.jobTracker.Logger(ctx)
	logger.Warn("Replication flows not yet implemented", "flow_id", flow.ID)
	return fmt.Errorf("replication flows are not yet implemented (Phase 5)")
}

// =============================================================================
// FLOW VALIDATION
// =============================================================================

// ValidateFlowConfiguration validates a flow configuration
func (s *ProtectionFlowService) ValidateFlowConfiguration(ctx context.Context, flow *database.ProtectionFlow) error {
	// Validate flow type
	if flow.FlowType != "backup" && flow.FlowType != "replication" {
		return fmt.Errorf("invalid flow_type: must be 'backup' or 'replication'")
	}

	// Validate target type
	if flow.TargetType != "vm" && flow.TargetType != "group" {
		return fmt.Errorf("invalid target_type: must be 'vm' or 'group'")
	}

	// Validate target exists
	switch flow.TargetType {
	case "vm":
		var vmCtx database.VMReplicationContext
		if err := s.db.GetGormDB().Where("context_id = ?", flow.TargetID).First(&vmCtx).Error; err != nil {
			return fmt.Errorf("VM context not found: %s", flow.TargetID)
		}
	case "group":
		// TODO: Validate group exists when group service is available
	}

	// Validate backup-specific fields
	if flow.FlowType == "backup" {
		if flow.RepositoryID == nil {
			return fmt.Errorf("repository_id is required for backup flows")
		}
		// TODO: Validate repository exists
	}

	// Validate replication-specific fields (Phase 5)
	if flow.FlowType == "replication" {
		if flow.DestinationType == nil {
			return fmt.Errorf("destination_type is required for replication flows")
		}
	}

	// Validate schedule if provided
	if flow.ScheduleID != nil {
		// TODO: Validate schedule exists
	}

	return nil
}

// validateFlowRequest validates a flow creation request
func (s *ProtectionFlowService) validateFlowRequest(req CreateFlowRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.Name) > 255 {
		return fmt.Errorf("name must be 255 characters or less")
	}

	// Validate flow type
	if req.FlowType != "backup" && req.FlowType != "replication" {
		return fmt.Errorf("flow_type must be 'backup' or 'replication'")
	}

	// Validate target type
	if req.TargetType != "vm" && req.TargetType != "group" {
		return fmt.Errorf("target_type must be 'vm' or 'group'")
	}

	if req.TargetID == "" {
		return fmt.Errorf("target_id is required")
	}

	// Validate backup-specific requirements
	if req.FlowType == "backup" && req.RepositoryID == nil {
		return fmt.Errorf("repository_id is required for backup flows")
	}

	return nil
}

// =============================================================================
// FLOW STATUS AND STATISTICS
// =============================================================================

// GetFlowStatus returns current flow status with next execution time
func (s *ProtectionFlowService) GetFlowStatus(ctx context.Context, flowID string) (*FlowStatus, error) {
	flow, err := s.flowRepo.GetFlowByID(ctx, flowID)
	if err != nil {
		return nil, err
	}

	status := &FlowStatus{
		LastExecutionID:     flow.LastExecutionID,
		LastExecutionStatus: flow.LastExecutionStatus,
		LastExecutionTime:   flow.LastExecutionTime,
		NextExecutionTime:   flow.NextExecutionTime,
		TotalExecutions:     flow.TotalExecutions,
		SuccessfulExecutions: flow.SuccessfulExecutions,
		FailedExecutions:    flow.FailedExecutions,
	}

	// Calculate next execution time if scheduled
	if flow.ScheduleID != nil && flow.Enabled {
		// TODO: Calculate next run time from schedule
	}

	return status, nil
}

// GetFlowExecutions returns execution history for a flow
func (s *ProtectionFlowService) GetFlowExecutions(ctx context.Context, flowID string, limit int) ([]*database.ProtectionFlowExecution, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if limit > 100 {
		limit = 100 // Maximum limit
	}

	return s.flowRepo.ListExecutions(ctx, flowID, limit)
}

// =============================================================================
// BACKUP API INTEGRATION
// =============================================================================

// BackupStartRequest matches the backup API request structure
type BackupStartRequest struct {
	VMName       string            `json:"vm_name"`
	BackupType   string            `json:"backup_type"`
	RepositoryID string            `json:"repository_id"`
	PolicyID     string            `json:"policy_id,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// BackupStartResponse matches the backup API response structure
type BackupStartResponse struct {
	BackupID  string            `json:"backup_id"`
	Status    string            `json:"status"`
	Message   string            `json:"message,omitempty"`
	TotalBytes int64            `json:"total_bytes"`
	Disks     []BackupDiskInfo  `json:"disks,omitempty"`
}

// BackupDiskInfo represents disk information in backup response
type BackupDiskInfo struct {
	DiskID     int    `json:"disk_id"`
	NBDPort    int    `json:"nbd_port"`
	ExportName string `json:"nbd_export_name"`
	QCOW2Path  string `json:"qcow2_path"`
	Status     string `json:"status"`
}

// startBackup calls the backup API to start a backup
func (s *ProtectionFlowService) startBackup(ctx context.Context, req *BackupStartRequest) (*BackupStartResponse, error) {
	// Prepare request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request to backup API
	url := fmt.Sprintf("%s/api/v1/backups", s.backupAPIURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.backupAPIClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("backup API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("backup API returned status %d", resp.StatusCode)
	}

	// Parse response
	var backupResp BackupStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&backupResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &backupResp, nil
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// stringPtrToString converts a string pointer to string, returning empty string if nil
func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
