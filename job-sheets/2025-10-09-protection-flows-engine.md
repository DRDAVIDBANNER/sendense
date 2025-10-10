# Protection Flows Engine - Job Sheet

**Created:** 2025-10-09  
**Status:** 🔴 NOT STARTED  
**Phase:** Phase 1 Extension - VMware Backup Automation  
**Priority:** HIGH (Blocks GUI full functionality)  
**Estimated Effort:** 2-3 days backend + 1 day GUI integration  
**Assignee:** TBD

---

## 🎯 Objective

Build a **Protection Flow Engine** that provides a unified abstraction layer for scheduled backup and replication operations. This engine ties together existing components (VMs, Protection Groups, Schedules, Repositories, Policies) into user-friendly "flows" that can be created, managed, and executed from the GUI.

**Business Value:**
- Customers define "what to protect" + "when" + "where" in a single UI flow
- Supports both backup flows (Phase 1) and replication flows (Phase 5)
- Reuses existing scheduler infrastructure (no wheel reinvention)
- Foundation for MSP multi-tenancy (Phase 7)

---

## 📊 Current State Assessment

### ✅ What Exists (Working Components)

1. **Backup Engine** (`/api/v1/backups`) ✅
   - POST /backups (start backup)
   - Multi-disk VM support
   - CBT change ID tracking
   - Repository storage
   
2. **Protection Groups** (`/api/v1/machine-groups`) ✅
   - CRUD operations
   - VM membership management
   - Multi-group support (VMs in multiple groups)
   - Priority/concurrency settings
   
3. **Scheduler System** (`/api/v1/schedules`) ✅
   - Cron-based scheduling
   - Schedule executions tracking
   - Group-level scheduling
   - Using `robfig/cron/v3` library
   
4. **Repository API** (`/api/v1/repositories`) ✅
   - Local/NFS/CIFS storage
   - Storage capacity monitoring
   - Immutability settings
   
5. **Backup Policies** (`/api/v1/policies`) ✅
   - 3-2-1 backup rules
   - Copy rules for multi-repository
   - Retention settings

### ❌ What's Missing (The Gap)

**NO unified "Protection Flow" concept exists in backend.**

The GUI has `Flow` types with mock data, but backend has no:
- Flow creation API
- Flow-to-components binding (VM/Group + Schedule + Repository + Policy)
- Flow execution orchestration
- Flow status/history tracking
- Flow type abstraction (backup vs replication)

**Current Reality:**
- Protection Groups page: ✅ Fully functional with real backend
- Protection Flows page: ❌ Pure mock data, no backend integration

---

## 🏗️ Architecture Design

### Database Schema

```sql
-- Protection flows table (new)
CREATE TABLE protection_flows (
    id VARCHAR(64) PRIMARY KEY DEFAULT (UUID()),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    
    -- Flow type and target
    flow_type ENUM('backup', 'replication') NOT NULL,
    target_type ENUM('vm', 'group') NOT NULL,
    target_id VARCHAR(64) NOT NULL,  -- vm_context_id OR group_id
    
    -- Backup-specific configuration
    repository_id INT,  -- FK to backup_repositories
    policy_id INT,      -- FK to backup_policies
    
    -- Replication-specific configuration (Phase 5)
    destination_type ENUM('ossea', 'vmware', 'hyperv') DEFAULT NULL,
    destination_config JSON DEFAULT NULL,  -- Extensible for future
    
    -- Scheduling
    schedule_id VARCHAR(64),  -- FK to replication_schedules
    
    -- Control flags
    enabled BOOLEAN DEFAULT true,
    
    -- Statistics (denormalized for performance)
    last_execution_id VARCHAR(64),
    last_execution_status ENUM('success', 'warning', 'error', 'running', 'pending') DEFAULT 'pending',
    last_execution_time TIMESTAMP,
    next_execution_time TIMESTAMP,
    total_executions INT DEFAULT 0,
    successful_executions INT DEFAULT 0,
    failed_executions INT DEFAULT 0,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system',
    
    -- Foreign keys
    FOREIGN KEY (schedule_id) REFERENCES replication_schedules(id) ON DELETE SET NULL,
    FOREIGN KEY (repository_id) REFERENCES backup_repositories(id) ON DELETE RESTRICT,
    FOREIGN KEY (policy_id) REFERENCES backup_policies(id) ON DELETE SET NULL,
    
    -- Indexes for common queries
    INDEX idx_flow_type (flow_type),
    INDEX idx_target (target_type, target_id),
    INDEX idx_enabled (enabled),
    INDEX idx_schedule (schedule_id),
    INDEX idx_last_execution (last_execution_time),
    
    -- Constraints
    CHECK (
        (flow_type = 'backup' AND repository_id IS NOT NULL) OR
        (flow_type = 'replication' AND destination_type IS NOT NULL)
    )
);

-- Flow executions table (new) - tracks individual flow runs
CREATE TABLE protection_flow_executions (
    id VARCHAR(64) PRIMARY KEY DEFAULT (UUID()),
    flow_id VARCHAR(64) NOT NULL,
    
    -- Execution details
    status ENUM('pending', 'running', 'success', 'warning', 'error', 'cancelled') NOT NULL DEFAULT 'pending',
    execution_type ENUM('scheduled', 'manual', 'api') NOT NULL,
    
    -- Job tracking
    jobs_created INT DEFAULT 0,
    jobs_completed INT DEFAULT 0,
    jobs_failed INT DEFAULT 0,
    jobs_skipped INT DEFAULT 0,
    
    -- Timing
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    execution_time_seconds INT,
    
    -- Results
    vms_processed INT DEFAULT 0,
    bytes_transferred BIGINT DEFAULT 0,
    error_message TEXT,
    execution_metadata JSON,  -- Flexible for execution details
    
    -- Links to actual work
    created_job_ids JSON,      -- Array of backup_job_id or replication_job_id
    schedule_execution_id VARCHAR(64),  -- FK to schedule_executions if scheduled
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    triggered_by VARCHAR(255) DEFAULT 'system',
    
    FOREIGN KEY (flow_id) REFERENCES protection_flows(id) ON DELETE CASCADE,
    FOREIGN KEY (schedule_execution_id) REFERENCES schedule_executions(id) ON DELETE SET NULL,
    
    INDEX idx_flow_executions (flow_id, started_at DESC),
    INDEX idx_status (status),
    INDEX idx_execution_time (started_at)
);
```

### API Endpoints (New)

#### Core CRUD Operations

```
POST   /api/v1/protection-flows              Create new protection flow
GET    /api/v1/protection-flows              List all flows (with filters)
GET    /api/v1/protection-flows/{id}         Get flow details
PUT    /api/v1/protection-flows/{id}         Update flow configuration
DELETE /api/v1/protection-flows/{id}         Delete flow
PATCH  /api/v1/protection-flows/{id}/enable  Enable flow
PATCH  /api/v1/protection-flows/{id}/disable Disable flow
```

#### Execution Operations

```
POST   /api/v1/protection-flows/{id}/execute     Manually trigger flow execution
GET    /api/v1/protection-flows/{id}/executions  Get execution history
GET    /api/v1/protection-flows/{id}/status      Get current status + next run
POST   /api/v1/protection-flows/{id}/test        Test flow configuration (dry run)
```

#### Bulk Operations

```
POST   /api/v1/protection-flows/bulk-enable      Enable multiple flows
POST   /api/v1/protection-flows/bulk-disable     Disable multiple flows
POST   /api/v1/protection-flows/bulk-delete      Delete multiple flows
GET    /api/v1/protection-flows/summary          Get aggregated statistics
```

### Request/Response Schemas

#### CreateFlowRequest

```json
{
  "name": "Daily VM Backup - Production",
  "description": "Automated daily backup of production VMs",
  "flow_type": "backup",
  "target_type": "group",
  "target_id": "group-uuid-here",
  "repository_id": 1,
  "policy_id": 2,
  "schedule_id": "schedule-uuid-here",
  "enabled": true
}
```

#### FlowResponse

```json
{
  "id": "flow-uuid-here",
  "name": "Daily VM Backup - Production",
  "description": "Automated daily backup of production VMs",
  "flow_type": "backup",
  "target_type": "group",
  "target_id": "group-uuid-here",
  "target_name": "Production Servers",
  "repository_id": 1,
  "repository_name": "Primary NFS Storage",
  "policy_id": 2,
  "policy_name": "3-2-1 Enterprise",
  "schedule_id": "schedule-uuid-here",
  "schedule_name": "Daily 2 AM",
  "schedule_cron": "0 2 * * *",
  "enabled": true,
  "status": {
    "last_execution_id": "exec-uuid-here",
    "last_execution_status": "success",
    "last_execution_time": "2025-10-09T02:00:00Z",
    "next_execution_time": "2025-10-10T02:00:00Z",
    "total_executions": 45,
    "successful_executions": 43,
    "failed_executions": 2
  },
  "created_at": "2025-09-01T10:00:00Z",
  "updated_at": "2025-10-09T02:15:00Z",
  "created_by": "admin@sendense.io"
}
```

#### ExecutionResponse

```json
{
  "id": "exec-uuid-here",
  "flow_id": "flow-uuid-here",
  "flow_name": "Daily VM Backup - Production",
  "status": "success",
  "execution_type": "scheduled",
  "started_at": "2025-10-09T02:00:00Z",
  "completed_at": "2025-10-09T02:45:32Z",
  "execution_time_seconds": 2732,
  "jobs_created": 12,
  "jobs_completed": 12,
  "jobs_failed": 0,
  "jobs_skipped": 0,
  "vms_processed": 12,
  "bytes_transferred": 45678901234,
  "created_job_ids": [
    "backup-pgtest1-1759901593",
    "backup-pgtest2-1759901594",
    "..."
  ],
  "triggered_by": "scheduler",
  "created_at": "2025-10-09T02:00:00Z"
}
```

---

## 🔨 Implementation Tasks

### Task 1: Database Schema and Migrations ⏱️ 2-4 hours

**Files to Create:**
- `sha/database/migrations/20251009_create_protection_flows.up.sql`
- `sha/database/migrations/20251009_create_protection_flows.down.sql`

**Deliverables:**
1. Create `protection_flows` table with all fields
2. Create `protection_flow_executions` table with CASCADE DELETE
3. Add foreign key constraints to existing tables
4. Create indexes for query optimization
5. Test migrations on dev database

**Acceptance Criteria:**
- [ ] Migrations run cleanly on fresh database
- [ ] Rollback migrations work correctly
- [ ] Foreign key constraints validated
- [ ] Indexes created successfully

---

### Task 2: Go Models and Repository ⏱️ 3-4 hours

**Files to Create:**
- `sha/database/models.go` (add new structs)
- `sha/database/flow_repository.go` (new file)

**Go Structs:**

```go
// ProtectionFlow represents a configured backup or replication flow
type ProtectionFlow struct {
    ID          string  `json:"id" gorm:"primaryKey;type:varchar(64);default:uuid()"`
    Name        string  `json:"name" gorm:"not null;uniqueIndex;type:varchar(255)"`
    Description *string `json:"description" gorm:"type:text"`
    
    // Flow configuration
    FlowType   string `json:"flow_type" gorm:"type:enum('backup','replication');not null"`
    TargetType string `json:"target_type" gorm:"type:enum('vm','group');not null"`
    TargetID   string `json:"target_id" gorm:"type:varchar(64);not null;index"`
    
    // Backup configuration
    RepositoryID *int `json:"repository_id" gorm:"index"`
    PolicyID     *int `json:"policy_id" gorm:"index"`
    
    // Replication configuration (Phase 5)
    DestinationType  *string `json:"destination_type" gorm:"type:enum('ossea','vmware','hyperv')"`
    DestinationConfig *string `json:"destination_config" gorm:"type:json"`
    
    // Scheduling
    ScheduleID *string `json:"schedule_id" gorm:"type:varchar(64);index"`
    
    // Control
    Enabled bool `json:"enabled" gorm:"default:true;index"`
    
    // Statistics
    LastExecutionID     *string    `json:"last_execution_id" gorm:"type:varchar(64)"`
    LastExecutionStatus string     `json:"last_execution_status" gorm:"type:enum('success','warning','error','running','pending');default:'pending'"`
    LastExecutionTime   *time.Time `json:"last_execution_time"`
    NextExecutionTime   *time.Time `json:"next_execution_time"`
    TotalExecutions     int        `json:"total_executions" gorm:"default:0"`
    SuccessfulExecutions int       `json:"successful_executions" gorm:"default:0"`
    FailedExecutions    int        `json:"failed_executions" gorm:"default:0"`
    
    // Metadata
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
    CreatedBy string    `json:"created_by" gorm:"default:'system';type:varchar(255)"`
    
    // Relationships (loaded with joins)
    Schedule   *ReplicationSchedule `json:"schedule,omitempty" gorm:"foreignKey:ScheduleID;references:ID"`
    Repository *BackupRepository    `json:"repository,omitempty" gorm:"foreignKey:RepositoryID;references:ID"`
    Policy     *BackupPolicy        `json:"policy,omitempty" gorm:"foreignKey:PolicyID;references:ID"`
    Executions []ProtectionFlowExecution `json:"executions,omitempty" gorm:"foreignKey:FlowID;references:ID"`
}

// ProtectionFlowExecution tracks individual flow execution runs
type ProtectionFlowExecution struct {
    ID        string `json:"id" gorm:"primaryKey;type:varchar(64);default:uuid()"`
    FlowID    string `json:"flow_id" gorm:"type:varchar(64);not null;index"`
    
    // Execution details
    Status        string `json:"status" gorm:"type:enum('pending','running','success','warning','error','cancelled');not null;default:'pending';index"`
    ExecutionType string `json:"execution_type" gorm:"type:enum('scheduled','manual','api');not null"`
    
    // Job tracking
    JobsCreated   int `json:"jobs_created" gorm:"default:0"`
    JobsCompleted int `json:"jobs_completed" gorm:"default:0"`
    JobsFailed    int `json:"jobs_failed" gorm:"default:0"`
    JobsSkipped   int `json:"jobs_skipped" gorm:"default:0"`
    
    // Timing
    StartedAt           *time.Time `json:"started_at"`
    CompletedAt         *time.Time `json:"completed_at"`
    ExecutionTimeSeconds int       `json:"execution_time_seconds"`
    
    // Results
    VMsProcessed      int     `json:"vms_processed" gorm:"default:0"`
    BytesTransferred  int64   `json:"bytes_transferred" gorm:"default:0"`
    ErrorMessage      *string `json:"error_message" gorm:"type:text"`
    ExecutionMetadata *string `json:"execution_metadata" gorm:"type:json"`
    
    // Links
    CreatedJobIDs        *string `json:"created_job_ids" gorm:"type:json"`
    ScheduleExecutionID  *string `json:"schedule_execution_id" gorm:"type:varchar(64)"`
    
    // Metadata
    CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
    TriggeredBy string    `json:"triggered_by" gorm:"default:'system';type:varchar(255)"`
    
    // Relationships
    Flow *ProtectionFlow `json:"flow,omitempty" gorm:"foreignKey:FlowID;references:ID"`
}
```

**Repository Methods:**

```go
type FlowRepository interface {
    // CRUD
    CreateFlow(ctx context.Context, flow *ProtectionFlow) error
    GetFlowByID(ctx context.Context, id string) (*ProtectionFlow, error)
    GetFlowByName(ctx context.Context, name string) (*ProtectionFlow, error)
    ListFlows(ctx context.Context, filters FlowFilters) ([]*ProtectionFlow, error)
    UpdateFlow(ctx context.Context, id string, updates map[string]interface{}) error
    DeleteFlow(ctx context.Context, id string) error
    
    // Enable/Disable
    EnableFlow(ctx context.Context, id string) error
    DisableFlow(ctx context.Context, id string) error
    BulkEnableFlows(ctx context.Context, ids []string) error
    BulkDisableFlows(ctx context.Context, ids []string) error
    
    // Status updates
    UpdateFlowStatistics(ctx context.Context, id string, stats FlowStatistics) error
    
    // Execution tracking
    CreateExecution(ctx context.Context, execution *ProtectionFlowExecution) error
    GetExecution(ctx context.Context, id string) (*ProtectionFlowExecution, error)
    ListExecutions(ctx context.Context, flowID string, limit int) ([]*ProtectionFlowExecution, error)
    UpdateExecutionStatus(ctx context.Context, id string, status string, updates map[string]interface{}) error
    
    // Queries
    GetFlowsByTarget(ctx context.Context, targetType, targetID string) ([]*ProtectionFlow, error)
    GetFlowsBySchedule(ctx context.Context, scheduleID string) ([]*ProtectionFlow, error)
    GetEnabledFlows(ctx context.Context, flowType string) ([]*ProtectionFlow, error)
    GetFlowsWithNextRun(ctx context.Context, before time.Time) ([]*ProtectionFlow, error)
}
```

**Deliverables:**
1. Go structs with GORM tags
2. Repository interface definition
3. Repository implementation with all methods
4. Unit tests for repository methods

**Acceptance Criteria:**
- [ ] Structs match database schema exactly
- [ ] All CRUD operations functional
- [ ] Foreign key relationships loaded with GORM
- [ ] Repository tests pass

---

### Task 3: Protection Flow Service (Business Logic) ⏱️ 4-6 hours

**Files to Create:**
- `sha/services/protection_flow_service.go` (new file)

**Service Responsibilities:**
1. Validate flow configurations
2. Coordinate with existing services (scheduler, backup, replication)
3. Execute flows (manual or scheduled)
4. Track execution status
5. Update flow statistics

**Key Service Methods:**

```go
type ProtectionFlowService struct {
    flowRepo        *database.FlowRepository
    scheduleService *SchedulerService
    backupService   *BackupService
    machineGroupSvc *MachineGroupService
    vmContextRepo   *database.VMContextRepository
    tracker         *joblog.Tracker
}

// CreateFlow validates and creates a new protection flow
func (s *ProtectionFlowService) CreateFlow(ctx context.Context, req CreateFlowRequest) (*ProtectionFlow, error)

// ExecuteFlow triggers flow execution (manual or scheduled)
func (s *ProtectionFlowService) ExecuteFlow(ctx context.Context, flowID string, executionType string) (*ProtectionFlowExecution, error)

// ValidateFlowConfiguration checks if flow config is valid
func (s *ProtectionFlowService) ValidateFlowConfiguration(ctx context.Context, flow *ProtectionFlow) error

// ProcessBackupFlow executes a backup-type flow
func (s *ProtectionFlowService) ProcessBackupFlow(ctx context.Context, execution *ProtectionFlowExecution) error

// ProcessReplicationFlow executes a replication-type flow (Phase 5)
func (s *ProtectionFlowService) ProcessReplicationFlow(ctx context.Context, execution *ProtectionFlowExecution) error

// GetFlowStatus returns current flow status with next execution time
func (s *ProtectionFlowService) GetFlowStatus(ctx context.Context, flowID string) (*FlowStatus, error)

// SyncFlowWithSchedule updates next_execution_time from scheduler
func (s *ProtectionFlowService) SyncFlowWithSchedule(ctx context.Context, flowID string) error
```

**Flow Execution Logic:**

```go
// ExecuteFlow - Main orchestration method
func (s *ProtectionFlowService) ExecuteFlow(ctx context.Context, flowID string, executionType string) (*ProtectionFlowExecution, error) {
    // 1. Load flow with relationships
    flow, err := s.flowRepo.GetFlowByID(ctx, flowID)
    if err != nil {
        return nil, fmt.Errorf("flow not found: %w", err)
    }
    
    // 2. Create execution record
    execution := &ProtectionFlowExecution{
        FlowID:        flowID,
        Status:        "running",
        ExecutionType: executionType,
        StartedAt:     time.Now(),
        TriggeredBy:   getContextUser(ctx),
    }
    if err := s.flowRepo.CreateExecution(ctx, execution); err != nil {
        return nil, err
    }
    
    // 3. Route to appropriate handler
    var execErr error
    switch flow.FlowType {
    case "backup":
        execErr = s.ProcessBackupFlow(ctx, flow, execution)
    case "replication":
        execErr = s.ProcessReplicationFlow(ctx, flow, execution)
    default:
        execErr = fmt.Errorf("unknown flow type: %s", flow.FlowType)
    }
    
    // 4. Update execution status
    finalStatus := "success"
    if execErr != nil {
        finalStatus = "error"
        execution.ErrorMessage = execErr.Error()
    } else if execution.JobsFailed > 0 {
        finalStatus = "warning"
    }
    
    execution.Status = finalStatus
    execution.CompletedAt = time.Now()
    execution.ExecutionTimeSeconds = int(execution.CompletedAt.Sub(*execution.StartedAt).Seconds())
    
    s.flowRepo.UpdateExecutionStatus(ctx, execution.ID, finalStatus, map[string]interface{}{
        "completed_at":            execution.CompletedAt,
        "execution_time_seconds": execution.ExecutionTimeSeconds,
        "error_message":          execution.ErrorMessage,
        "jobs_created":           execution.JobsCreated,
        "jobs_completed":         execution.JobsCompleted,
        "jobs_failed":            execution.JobsFailed,
        "vms_processed":          execution.VMsProcessed,
        "bytes_transferred":      execution.BytesTransferred,
    })
    
    // 5. Update flow statistics
    s.flowRepo.UpdateFlowStatistics(ctx, flowID, FlowStatistics{
        LastExecutionID:     execution.ID,
        LastExecutionStatus: finalStatus,
        LastExecutionTime:   execution.CompletedAt,
        TotalExecutions:     flow.TotalExecutions + 1,
        SuccessfulExecutions: flow.SuccessfulExecutions + (finalStatus == "success" ? 1 : 0),
        FailedExecutions:    flow.FailedExecutions + (finalStatus == "error" ? 1 : 0),
    })
    
    return execution, nil
}

// ProcessBackupFlow - Backup-specific execution
func (s *ProtectionFlowService) ProcessBackupFlow(ctx context.Context, flow *ProtectionFlow, execution *ProtectionFlowExecution) error {
    logger := s.tracker.Logger(ctx)
    
    // 1. Resolve target VMs
    var vmContexts []string
    switch flow.TargetType {
    case "vm":
        vmContexts = []string{flow.TargetID}
    case "group":
        members, err := s.machineGroupSvc.GetGroupMembers(ctx, flow.TargetID)
        if err != nil {
            return fmt.Errorf("failed to get group members: %w", err)
        }
        for _, member := range members {
            if member.Enabled {
                vmContexts = append(vmContexts, member.VMContextID)
            }
        }
    }
    
    logger.Info("Executing backup flow", "flow_id", flow.ID, "vms_count", len(vmContexts))
    
    // 2. Execute backup for each VM
    var createdJobIDs []string
    var jobsCompleted, jobsFailed, jobsSkipped int
    var totalBytes int64
    
    for _, contextID := range vmContexts {
        vmCtx, err := s.vmContextRepo.GetByContextID(ctx, contextID)
        if err != nil {
            logger.Error("Failed to load VM context", "context_id", contextID, "error", err)
            jobsSkipped++
            continue
        }
        
        // Trigger backup via existing backup API
        backupReq := &BackupStartRequest{
            VMName:       vmCtx.VMName,
            RepositoryID: *flow.RepositoryID,
            BackupType:   "auto", // Will detect full vs incremental
        }
        
        backupResp, err := s.backupService.StartBackup(ctx, backupReq)
        if err != nil {
            logger.Error("Failed to start backup", "vm_name", vmCtx.VMName, "error", err)
            jobsFailed++
            continue
        }
        
        createdJobIDs = append(createdJobIDs, backupResp.BackupID)
        jobsCompleted++
        totalBytes += backupResp.TotalBytes
        
        logger.Info("Backup started", "vm_name", vmCtx.VMName, "backup_id", backupResp.BackupID)
    }
    
    // 3. Update execution with results
    execution.JobsCreated = len(createdJobIDs)
    execution.JobsCompleted = jobsCompleted
    execution.JobsFailed = jobsFailed
    execution.JobsSkipped = jobsSkipped
    execution.VMsProcessed = len(vmContexts)
    execution.BytesTransferred = totalBytes
    execution.CreatedJobIDs = createdJobIDs
    
    if jobsFailed > 0 {
        return fmt.Errorf("%d of %d backups failed", jobsFailed, len(vmContexts))
    }
    
    return nil
}
```

**Deliverables:**
1. Complete service implementation
2. Flow validation logic
3. Backup flow execution
4. Replication flow execution (stub for Phase 5)
5. Integration with existing scheduler service
6. JobLog integration for tracking

**Acceptance Criteria:**
- [ ] Flow creation validates all dependencies
- [ ] Manual flow execution works end-to-end
- [ ] Backup flows trigger actual backups
- [ ] Execution history tracked correctly
- [ ] Flow statistics updated accurately
- [ ] JobLog integration complete

---

### Task 4: Scheduler Integration ⏱️ 3-4 hours

**Files to Modify:**
- `sha/services/scheduler_service.go` (extend existing)

**Integration Points:**

1. **Add Flow Support to Scheduler:**
   - Scheduler already handles cron expressions
   - Extend to trigger protection flows instead of just replications
   - Reuse existing `ScheduleExecution` tracking

2. **Schedule Callback for Flows:**

```go
// Add to SchedulerService
func (s *SchedulerService) RegisterFlowSchedule(flowID string, scheduleID string) error {
    schedule, err := s.repository.GetScheduleByID(scheduleID)
    if err != nil {
        return err
    }
    
    // Add cron job that executes the flow
    entryID, err := s.cron.AddFunc(schedule.CronExpression, func() {
        ctx := context.Background()
        s.ExecuteScheduledFlow(ctx, flowID, scheduleID)
    })
    
    if err != nil {
        return fmt.Errorf("failed to register flow schedule: %w", err)
    }
    
    // Track cron entry
    s.activeFlowSchedules[flowID] = entryID
    return nil
}

func (s *SchedulerService) ExecuteScheduledFlow(ctx context.Context, flowID string, scheduleID string) {
    logger := s.jobTracker.Logger(ctx)
    logger.Info("Executing scheduled flow", "flow_id", flowID, "schedule_id", scheduleID)
    
    // Delegate to ProtectionFlowService
    execution, err := s.flowService.ExecuteFlow(ctx, flowID, "scheduled")
    if err != nil {
        logger.Error("Flow execution failed", "error", err)
        return
    }
    
    logger.Info("Flow execution completed", "execution_id", execution.ID, "status", execution.Status)
}
```

**Deliverables:**
1. Extend scheduler to support protection flows
2. Flow-to-schedule binding
3. Automatic flow execution on schedule
4. Schedule execution tracking linked to flow executions

**Acceptance Criteria:**
- [ ] Flows execute automatically on schedule
- [ ] Schedule changes reflected in flow execution
- [ ] Execution history linked correctly
- [ ] No interference with existing replication schedules

---

### Task 5: API Handlers ⏱️ 4-5 hours

**Files to Create:**
- `sha/api/handlers/protection_flow_handlers.go` (new file)

**Handler Implementation:**

```go
type ProtectionFlowHandler struct {
    flowService *services.ProtectionFlowService
    tracker     *joblog.Tracker
}

// CreateFlow - POST /api/v1/protection-flows
func (h *ProtectionFlowHandler) CreateFlow(w http.ResponseWriter, r *http.Request)

// ListFlows - GET /api/v1/protection-flows
func (h *ProtectionFlowHandler) ListFlows(w http.ResponseWriter, r *http.Request)

// GetFlow - GET /api/v1/protection-flows/{id}
func (h *ProtectionFlowHandler) GetFlow(w http.ResponseWriter, r *http.Request)

// UpdateFlow - PUT /api/v1/protection-flows/{id}
func (h *ProtectionFlowHandler) UpdateFlow(w http.ResponseWriter, r *http.Request)

// DeleteFlow - DELETE /api/v1/protection-flows/{id}
func (h *ProtectionFlowHandler) DeleteFlow(w http.ResponseWriter, r *http.Request)

// EnableFlow - PATCH /api/v1/protection-flows/{id}/enable
func (h *ProtectionFlowHandler) EnableFlow(w http.ResponseWriter, r *http.Request)

// DisableFlow - PATCH /api/v1/protection-flows/{id}/disable
func (h *ProtectionFlowHandler) DisableFlow(w http.ResponseWriter, r *http.Request)

// ExecuteFlow - POST /api/v1/protection-flows/{id}/execute
func (h *ProtectionFlowHandler) ExecuteFlow(w http.ResponseWriter, r *http.Request)

// GetFlowExecutions - GET /api/v1/protection-flows/{id}/executions
func (h *ProtectionFlowHandler) GetFlowExecutions(w http.ResponseWriter, r *http.Request)

// GetFlowStatus - GET /api/v1/protection-flows/{id}/status
func (h *ProtectionFlowHandler) GetFlowStatus(w http.ResponseWriter, r *http.Request)

// TestFlow - POST /api/v1/protection-flows/{id}/test
func (h *ProtectionFlowHandler) TestFlow(w http.ResponseWriter, r *http.Request)

// GetFlowSummary - GET /api/v1/protection-flows/summary
func (h *ProtectionFlowHandler) GetFlowSummary(w http.ResponseWriter, r *http.Request)
```

**Route Registration:**

```go
// Add to server.go
func (s *Server) RegisterProtectionFlowRoutes() {
    flowHandler := handlers.NewProtectionFlowHandler(s.flowService, s.tracker)
    
    r := s.router.PathPrefix("/api/v1/protection-flows").Subrouter()
    r.Use(s.authMiddleware)
    
    // CRUD
    r.HandleFunc("", flowHandler.CreateFlow).Methods("POST")
    r.HandleFunc("", flowHandler.ListFlows).Methods("GET")
    r.HandleFunc("/{id}", flowHandler.GetFlow).Methods("GET")
    r.HandleFunc("/{id}", flowHandler.UpdateFlow).Methods("PUT")
    r.HandleFunc("/{id}", flowHandler.DeleteFlow).Methods("DELETE")
    
    // Control
    r.HandleFunc("/{id}/enable", flowHandler.EnableFlow).Methods("PATCH")
    r.HandleFunc("/{id}/disable", flowHandler.DisableFlow).Methods("PATCH")
    
    // Execution
    r.HandleFunc("/{id}/execute", flowHandler.ExecuteFlow).Methods("POST")
    r.HandleFunc("/{id}/executions", flowHandler.GetFlowExecutions).Methods("GET")
    r.HandleFunc("/{id}/status", flowHandler.GetFlowStatus).Methods("GET")
    r.HandleFunc("/{id}/test", flowHandler.TestFlow).Methods("POST")
    
    // Summary
    r.HandleFunc("/summary", flowHandler.GetFlowSummary).Methods("GET")
}
```

**Deliverables:**
1. All handler methods implemented
2. Request validation
3. Response formatting
4. Error handling
5. Route registration
6. Authentication middleware

**Acceptance Criteria:**
- [ ] All endpoints return correct HTTP status codes
- [ ] Request validation works
- [ ] Responses match defined schemas
- [ ] Error messages are descriptive
- [ ] Authentication enforced

---

### Task 6: GUI Integration ⏱️ 6-8 hours

**Files to Modify/Create:**

1. **API Service Layer:**
   - `sendense-gui/src/services/protectionFlowsApi.ts` (new file)

2. **Components to Wire:**
   - `app/protection-flows/page.tsx` - Replace mock data
   - `components/features/protection-flows/CreateFlowModal.tsx` - Real API calls
   - `components/features/protection-flows/EditFlowModal.tsx` - Real API calls
   - `components/features/protection-flows/FlowDetailsPanel.tsx` - Real data display
   - `components/features/protection-flows/FlowsTable.tsx` - Real data source

3. **API Integration Code:**

```typescript
// src/services/protectionFlowsApi.ts
import { apiClient } from './api';

export interface ProtectionFlow {
  id: string;
  name: string;
  description?: string;
  flow_type: 'backup' | 'replication';
  target_type: 'vm' | 'group';
  target_id: string;
  target_name?: string;
  repository_id?: number;
  repository_name?: string;
  policy_id?: number;
  policy_name?: string;
  schedule_id?: string;
  schedule_name?: string;
  schedule_cron?: string;
  enabled: boolean;
  status: {
    last_execution_id?: string;
    last_execution_status: 'success' | 'warning' | 'error' | 'running' | 'pending';
    last_execution_time?: string;
    next_execution_time?: string;
    total_executions: number;
    successful_executions: number;
    failed_executions: number;
  };
  created_at: string;
  updated_at: string;
  created_by: string;
}

export interface CreateFlowRequest {
  name: string;
  description?: string;
  flow_type: 'backup' | 'replication';
  target_type: 'vm' | 'group';
  target_id: string;
  repository_id?: number;
  policy_id?: number;
  schedule_id?: string;
  enabled?: boolean;
}

export interface FlowExecution {
  id: string;
  flow_id: string;
  flow_name: string;
  status: 'pending' | 'running' | 'success' | 'warning' | 'error' | 'cancelled';
  execution_type: 'scheduled' | 'manual' | 'api';
  started_at?: string;
  completed_at?: string;
  execution_time_seconds?: number;
  jobs_created: number;
  jobs_completed: number;
  jobs_failed: number;
  jobs_skipped: number;
  vms_processed: number;
  bytes_transferred: number;
  error_message?: string;
  created_job_ids?: string[];
  triggered_by: string;
  created_at: string;
}

export const protectionFlowsApi = {
  // CRUD operations
  listFlows: async (filters?: { 
    flow_type?: string; 
    target_type?: string;
    enabled?: boolean;
  }): Promise<{ flows: ProtectionFlow[]; total: number }> => {
    const params = new URLSearchParams();
    if (filters?.flow_type) params.append('flow_type', filters.flow_type);
    if (filters?.target_type) params.append('target_type', filters.target_type);
    if (filters?.enabled !== undefined) params.append('enabled', String(filters.enabled));
    
    const response = await apiClient.get(`/protection-flows?${params.toString()}`);
    return response.data;
  },
  
  getFlow: async (id: string): Promise<ProtectionFlow> => {
    const response = await apiClient.get(`/protection-flows/${id}`);
    return response.data;
  },
  
  createFlow: async (data: CreateFlowRequest): Promise<ProtectionFlow> => {
    const response = await apiClient.post('/protection-flows', data);
    return response.data;
  },
  
  updateFlow: async (id: string, data: Partial<CreateFlowRequest>): Promise<ProtectionFlow> => {
    const response = await apiClient.put(`/protection-flows/${id}`, data);
    return response.data;
  },
  
  deleteFlow: async (id: string): Promise<void> => {
    await apiClient.delete(`/protection-flows/${id}`);
  },
  
  // Control operations
  enableFlow: async (id: string): Promise<ProtectionFlow> => {
    const response = await apiClient.patch(`/protection-flows/${id}/enable`);
    return response.data;
  },
  
  disableFlow: async (id: string): Promise<ProtectionFlow> => {
    const response = await apiClient.patch(`/protection-flows/${id}/disable`);
    return response.data;
  },
  
  // Execution operations
  executeFlow: async (id: string): Promise<FlowExecution> => {
    const response = await apiClient.post(`/protection-flows/${id}/execute`);
    return response.data;
  },
  
  getFlowExecutions: async (id: string, limit?: number): Promise<{ executions: FlowExecution[]; total: number }> => {
    const params = limit ? `?limit=${limit}` : '';
    const response = await apiClient.get(`/protection-flows/${id}/executions${params}`);
    return response.data;
  },
  
  getFlowStatus: async (id: string): Promise<ProtectionFlow['status']> => {
    const response = await apiClient.get(`/protection-flows/${id}/status`);
    return response.data;
  },
  
  testFlow: async (id: string): Promise<{ valid: boolean; errors?: string[] }> => {
    const response = await apiClient.post(`/protection-flows/${id}/test`);
    return response.data;
  },
  
  // Summary
  getSummary: async (): Promise<{
    total_flows: number;
    enabled_flows: number;
    backup_flows: number;
    replication_flows: number;
    total_executions_today: number;
    failed_executions_today: number;
  }> => {
    const response = await apiClient.get('/protection-flows/summary');
    return response.data;
  },
};
```

4. **Update CreateFlowModal:**

```typescript
// components/features/protection-flows/CreateFlowModal.tsx
import { protectionFlowsApi } from '@/src/services/protectionFlowsApi';

export function CreateFlowModal({ isOpen, onClose, onCreate }: CreateFlowModalProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const handleSubmit = async (formData: CreateFlowFormData) => {
    setIsSubmitting(true);
    setError(null);
    
    try {
      const flow = await protectionFlowsApi.createFlow({
        name: formData.name,
        description: formData.description,
        flow_type: formData.type,
        target_type: formData.targetType,
        target_id: formData.targetId,
        repository_id: formData.repositoryId,
        policy_id: formData.policyId,
        schedule_id: formData.scheduleId,
        enabled: true,
      });
      
      onCreate(flow);
      onClose();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create flow');
    } finally {
      setIsSubmitting(false);
    }
  };
  
  // ... rest of modal implementation
}
```

5. **Update Page to Use Real Data:**

```typescript
// app/protection-flows/page.tsx
import { protectionFlowsApi, ProtectionFlow } from '@/src/services/protectionFlowsApi';

export default function ProtectionFlowsPage() {
  const [flows, setFlows] = useState<ProtectionFlow[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  // Fetch flows on mount
  useEffect(() => {
    fetchFlows();
  }, []);
  
  const fetchFlows = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await protectionFlowsApi.listFlows();
      setFlows(data.flows);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load flows');
    } finally {
      setIsLoading(false);
    }
  };
  
  const handleCreateFlow = async (flowData: CreateFlowFormData) => {
    // API call handled in modal, just refresh list
    await fetchFlows();
  };
  
  const handleRunNow = async (flow: ProtectionFlow) => {
    try {
      await protectionFlowsApi.executeFlow(flow.id);
      // Refresh to show updated status
      await fetchFlows();
    } catch (err: any) {
      alert(`Failed to execute flow: ${err.response?.data?.error || 'Unknown error'}`);
    }
  };
  
  // ... rest of page implementation
}
```

**Deliverables:**
1. Complete API service layer
2. All modals wired to real APIs
3. Real data loading with loading states
4. Error handling and user feedback
5. Flow execution from GUI
6. Status polling for running flows

**Acceptance Criteria:**
- [ ] GUI creates real flows via API
- [ ] Flow list loads from backend
- [ ] Flow execution works from GUI
- [ ] Error messages displayed to user
- [ ] Loading states shown appropriately
- [ ] No mock data remaining

---

### Task 7: Testing and Documentation ⏱️ 3-4 hours

**Testing Checklist:**

1. **Unit Tests:**
   - [ ] Repository methods
   - [ ] Service business logic
   - [ ] Flow validation
   
2. **Integration Tests:**
   - [ ] End-to-end flow creation
   - [ ] Scheduled execution
   - [ ] Manual execution
   - [ ] Group-level backups
   - [ ] Individual VM backups
   
3. **Manual Testing:**
   - [ ] Create backup flow for single VM
   - [ ] Create backup flow for protection group
   - [ ] Schedule executes flow automatically
   - [ ] Manual execution from GUI
   - [ ] Flow statistics update correctly
   - [ ] Execution history accurate
   - [ ] Enable/disable flows
   - [ ] Delete flows

**Documentation Deliverables:**

1. **API Documentation:**
   - Update `api-documentation/OMA.md` with new endpoints
   - Add request/response examples
   - Document error codes
   
2. **Architecture Documentation:**
   - Document flow execution logic
   - Diagram flow state machine
   - Explain scheduler integration
   
3. **User Guide:**
   - How to create protection flows
   - How to configure schedules
   - How to monitor flow execution

**Acceptance Criteria:**
- [ ] All tests passing
- [ ] API docs updated
- [ ] Architecture docs complete
- [ ] User guide written

---

## 🔄 Reusable Components

### Existing Scheduler Service ✅

The scheduler service (`sha/services/scheduler_service.go`) already provides:
- Cron-based scheduling using `robfig/cron/v3`
- Schedule execution tracking
- Concurrent execution limits
- Group-level operations
- JobLog integration

**What We're Reusing:**
```go
type SchedulerService struct {
    repository       *database.SchedulerRepository
    replicationRepo  *database.ReplicationJobRepository
    jobTracker       *joblog.Tracker
    cron             *cron.Cron
    // ... existing fields
}
```

**Extension Pattern:**
- Add `flowService *ProtectionFlowService` field
- Add `activeFlowSchedules map[string]cron.EntryID` for tracking
- Extend `RegisterSchedule` to support both replications and flows
- Route scheduled executions to appropriate service

**No Wheel Reinvention:**
- Use existing cron library ✅
- Use existing execution tracking ✅
- Use existing JobLog integration ✅
- Use existing concurrency controls ✅

---

## 🎯 Success Criteria

### Backend Success (Core Functionality)
- [ ] Database migrations run successfully
- [ ] All CRUD endpoints functional
- [ ] Flow execution works for single VM
- [ ] Flow execution works for protection groups
- [ ] Scheduled flows execute automatically
- [ ] Manual flow execution works
- [ ] Execution history tracked accurately
- [ ] Flow statistics updated correctly

### GUI Integration Success
- [ ] Create Flow modal works with real API
- [ ] Flows list displays real data
- [ ] Flow details panel shows accurate info
- [ ] "Run Now" button executes flows
- [ ] Enable/disable toggles work
- [ ] Execution history displays correctly
- [ ] No console errors
- [ ] All loading states functional

### Quality Gates
- [ ] All unit tests passing
- [ ] Integration tests passing
- [ ] Manual testing complete
- [ ] API documentation updated
- [ ] Code reviewed
- [ ] No TypeScript errors
- [ ] No Go compilation errors
- [ ] JobLog integration verified

---

## 📋 Phase Dependencies

### Phase 1 (Current) - VMware Backup ✅
- Backup API operational
- Protection Groups functional
- Scheduler system working
- Repository management ready

### Phase 5 - Multi-Platform Replication 🔜
- Reuse same Protection Flow engine
- Add `destination_type` support
- Extend `ProcessReplicationFlow` method
- Add replication-specific validation

### Phase 7 - MSP Platform 🔜
- Extend flows with tenant_id
- Multi-tenant flow isolation
- Tenant-specific repositories
- Tenant-specific policies

---

## 🚀 Deployment Plan

### Development Testing
1. Run migrations on dev database
2. Start SHA API with new endpoints
3. Test CRUD operations via curl/Postman
4. Test manual flow execution
5. Test scheduled flow execution
6. Verify JobLog integration

### Staging Deployment
1. Backup staging database
2. Run migrations
3. Deploy new SHA binary
4. Restart SHA service
5. Deploy new GUI build
6. Run smoke tests
7. Monitor for 24 hours

### Production Deployment
1. Schedule maintenance window
2. Backup production database
3. Run migrations
4. Deploy SHA binary
5. Deploy GUI build
6. Restart services
7. Verify flows execute correctly
8. Monitor execution logs

---

## 📚 Related Documentation

- **Phase 1 Context:** `/sendense/start_here/PHASE_1_CONTEXT_HELPER.md`
- **Project Goals:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`
- **API Reference:** `/sendense/source/current/api-documentation/OMA.md`
- **Backup Integration:** `/sendense/HANDOVER-GUI-BACKUP-RESTORE-INTEGRATION.md`
- **Scheduler Service:** `/sendense/source/current/sha/services/scheduler_service.go`
- **Protection Groups:** `/sendense/source/current/sha/services/machine_group_service.go`

---

## 🛠️ Development Environment Setup

### Backend Prerequisites
- Go 1.21+
- MariaDB/MySQL running
- SHA API development environment
- Access to test vCenter (for end-to-end testing)

### Frontend Prerequisites
- Node.js 18+
- npm/yarn
- GUI development environment
- API proxy configured to SHA

### Test Data Setup
```bash
# Create test protection group
curl -X POST http://localhost:8082/api/v1/machine-groups \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Group",
    "max_concurrent_vms": 5,
    "priority": 50,
    "created_by": "test@sendense.io"
  }'

# Create test schedule
curl -X POST http://localhost:8082/api/v1/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily 2 AM",
    "cron_expression": "0 2 * * *",
    "enabled": true
  }'
```

---

## ⚠️ Known Risks and Mitigations

### Risk 1: Scheduler Conflicts
**Risk:** New flow scheduling might conflict with existing replication schedules  
**Mitigation:** Use separate cron entry tracking maps, test thoroughly

### Risk 2: Database Performance
**Risk:** Flow execution tracking creates many rows  
**Mitigation:** Add indexes, implement execution history pruning

### Risk 3: Circular Dependencies
**Risk:** FlowService depends on BackupService which might need FlowService  
**Mitigation:** Clear dependency hierarchy, use interfaces

### Risk 4: GUI Breaking Changes
**Risk:** Changing Flow interface breaks existing GUI components  
**Mitigation:** Version API responses, use adapter pattern in GUI

---

## 📊 Estimated Timeline

| Task | Effort | Dependencies | Status |
|------|--------|--------------|--------|
| 1. Database Schema | 2-4h | None | 🔴 Not Started |
| 2. Go Models & Repository | 3-4h | Task 1 | 🔴 Not Started |
| 3. Flow Service | 4-6h | Task 2 | 🔴 Not Started |
| 4. Scheduler Integration | 3-4h | Task 3 | 🔴 Not Started |
| 5. API Handlers | 4-5h | Task 3 | 🔴 Not Started |
| 6. GUI Integration | 6-8h | Task 5 | 🔴 Not Started |
| 7. Testing & Docs | 3-4h | Tasks 1-6 | 🔴 Not Started |

**Total Estimated Effort:** 25-35 hours (3-4 days for experienced developer)

---

## ✅ Completion Checklist

### Backend Completion
- [ ] Migrations created and tested
- [ ] Models and repository implemented
- [ ] Service layer complete with validation
- [ ] Scheduler integration working
- [ ] All API endpoints functional
- [ ] Unit tests passing
- [ ] Integration tests passing
- [ ] API documentation updated

### GUI Completion
- [ ] API service layer created
- [ ] All modals wired to real APIs
- [ ] Page components using real data
- [ ] Loading states implemented
- [ ] Error handling complete
- [ ] No TypeScript errors
- [ ] No mock data remaining
- [ ] User guide updated

### Quality Assurance
- [ ] Code review completed
- [ ] Manual testing passed
- [ ] Performance testing done
- [ ] Security review done
- [ ] Documentation reviewed
- [ ] Deployment plan reviewed

---

## 📞 Support and Questions

- **Technical Questions:** Check `/sendense/start_here/PHASE_1_CONTEXT_HELPER.md`
- **API Questions:** See `/sendense/source/current/api-documentation/OMA.md`
- **Architecture Questions:** Review scheduler service and backup service implementations

---

**Job Sheet Version:** 1.0  
**Last Updated:** 2025-10-09  
**Next Review:** After Task 3 completion


