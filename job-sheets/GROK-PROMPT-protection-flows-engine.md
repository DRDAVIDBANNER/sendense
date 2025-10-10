# GROK PROMPT: Protection Flows Engine Implementation

**Context:** Sendense Phase 1 VMware Backup - Protection Flows Automation  
**Priority:** HIGH - Blocks GUI full functionality  
**Complexity:** Medium-High (3-4 days work)  
**Related Job Sheet:** `2025-10-09-protection-flows-engine.md`

---

## 🎯 Your Mission

Build a **Protection Flow Engine** that provides a unified abstraction for scheduled backup and replication operations. You're creating the orchestration layer that ties together existing components (VMs, Protection Groups, Schedules, Repositories, Policies) into user-friendly "flows" that customers can create and manage from the GUI.

**Why This Matters:**
- The GUI has a beautiful Protection Flows page, but it's 100% mock data
- We have all the pieces (backup engine, scheduler, groups, repositories) but no glue layer
- This blocks customers from using automated protection workflows
- This is the foundation for future replication flows (Phase 5)

---

## 📚 Read These Files FIRST

**CRITICAL - Read in this order:**

1. **Job Sheet (THIS IS YOUR BIBLE):**
   - `/home/oma_admin/sendense/job-sheets/2025-10-09-protection-flows-engine.md`
   - Contains complete architecture, database schema, API specs, everything

2. **Phase 1 Context:**
   - `/home/oma_admin/sendense/start_here/PHASE_1_CONTEXT_HELPER.md`
   - Understand backup architecture and existing components

3. **Existing Scheduler Service (REUSE THIS):**
   - `/home/oma_admin/sendense/source/current/sha/services/scheduler_service.go`
   - Shows cron integration, execution tracking patterns

4. **Existing Machine Group Service (REFERENCE):**
   - `/home/oma_admin/sendense/source/current/sha/services/machine_group_service.go`
   - Shows group management patterns

5. **API Documentation:**
   - `/home/oma_admin/sendense/source/current/api-documentation/OMA.md`
   - Understand existing endpoints and patterns

6. **GUI Handover:**
   - `/home/oma_admin/sendense/HANDOVER-GUI-BACKUP-RESTORE-INTEGRATION.md`
   - API integration patterns

---

## 🏗️ What You're Building

### The Problem

**Current State:**
```
Protection Groups ✅ → No connection ❌ → Backups ✅
Scheduler ✅        → No connection ❌ → Backups ✅
VMs ✅              → No connection ❌ → Backups ✅
```

**Desired State:**
```
Protection Flow = VM/Group + Schedule + Repository + Policy
    ↓
Scheduler executes flow automatically
    ↓
Flow orchestrates backups for all VMs in target
    ↓
GUI shows flow status and execution history
```

### The Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   PROTECTION FLOW ENGINE                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Protection Flow (new abstraction)                          │
│    ├─ flow_type: backup | replication                      │
│    ├─ target: VM or Protection Group                       │
│    ├─ repository: Where to store backups                   │
│    ├─ policy: Retention and copy rules                     │
│    ├─ schedule: When to run (cron)                         │
│    └─ enabled: On/off switch                               │
│         ↓                                                   │
│  Flow Execution                                             │
│    ├─ Resolve target VMs                                   │
│    ├─ Call existing backup API for each VM                 │
│    ├─ Track execution status                               │
│    └─ Update flow statistics                               │
│         ↓                                                   │
│  Scheduler Integration (reuse existing)                     │
│    ├─ Register cron job for each flow                      │
│    ├─ Trigger flow execution on schedule                   │
│    └─ Link to schedule_executions table                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 📋 Implementation Order (7 Tasks)

### Task 1: Database Schema (2-4 hours)
**Create:** Migration files in `sha/database/migrations/`

Two tables:
1. `protection_flows` - Flow definitions
2. `protection_flow_executions` - Execution history

**Complete schema in job sheet** - Follow it EXACTLY.

**Key Points:**
- Use VARCHAR(64) for IDs with UUID() default
- ENUM types for flow_type, target_type, status
- Foreign keys to existing tables (schedules, repositories, policies)
- Indexes on commonly queried fields
- CASCADE DELETE for executions

**Testing:**
```bash
# Run migration
mysql -u sendense -p sendense_db < migrations/20251009_create_protection_flows.up.sql

# Verify tables
mysql -u sendense -p sendense_db -e "SHOW TABLES LIKE 'protection_flow%'"
mysql -u sendense -p sendense_db -e "DESCRIBE protection_flows"

# Test rollback
mysql -u sendense -p sendense_db < migrations/20251009_create_protection_flows.down.sql
```

---

### Task 2: Go Models and Repository (3-4 hours)
**Modify:** `sha/database/models.go`  
**Create:** `sha/database/flow_repository.go`

**Models:**
```go
type ProtectionFlow struct {
    ID                   string     `json:"id" gorm:"primaryKey"`
    Name                 string     `json:"name" gorm:"uniqueIndex"`
    FlowType             string     `json:"flow_type"`
    TargetType           string     `json:"target_type"`
    TargetID             string     `json:"target_id"`
    RepositoryID         *int       `json:"repository_id"`
    ScheduleID           *string    `json:"schedule_id"`
    Enabled              bool       `json:"enabled"`
    LastExecutionStatus  string     `json:"last_execution_status"`
    // ... see job sheet for complete fields
}

type ProtectionFlowExecution struct {
    ID               string    `json:"id" gorm:"primaryKey"`
    FlowID           string    `json:"flow_id"`
    Status           string    `json:"status"`
    JobsCreated      int       `json:"jobs_created"`
    VMsProcessed     int       `json:"vms_processed"`
    BytesTransferred int64     `json:"bytes_transferred"`
    // ... see job sheet for complete fields
}
```

**Repository Interface:**
```go
type FlowRepository interface {
    CreateFlow(ctx context.Context, flow *ProtectionFlow) error
    GetFlowByID(ctx context.Context, id string) (*ProtectionFlow, error)
    ListFlows(ctx context.Context, filters FlowFilters) ([]*ProtectionFlow, error)
    UpdateFlow(ctx context.Context, id string, updates map[string]interface{}) error
    DeleteFlow(ctx context.Context, id string) error
    // ... see job sheet for all methods
}
```

**CRITICAL:** Follow existing patterns from `scheduler_repository.go` and `machine_group_service.go`

---

### Task 3: Protection Flow Service (4-6 hours) ⚠️ MOST IMPORTANT
**Create:** `sha/services/protection_flow_service.go`

This is the brain of the system. The service:

1. **Validates flow configurations**
2. **Executes flows (manual or scheduled)**
3. **Coordinates with existing services**
4. **Tracks execution status**
5. **Updates statistics**

**Core Method - ExecuteFlow:**

```go
func (s *ProtectionFlowService) ExecuteFlow(ctx context.Context, flowID string, executionType string) (*ProtectionFlowExecution, error) {
    // 1. Load flow with all relationships
    flow, err := s.flowRepo.GetFlowByID(ctx, flowID)
    
    // 2. Create execution record (status: "running")
    execution := &ProtectionFlowExecution{...}
    s.flowRepo.CreateExecution(ctx, execution)
    
    // 3. Route to handler based on flow type
    switch flow.FlowType {
    case "backup":
        err = s.ProcessBackupFlow(ctx, flow, execution)
    case "replication":
        err = s.ProcessReplicationFlow(ctx, flow, execution)
    }
    
    // 4. Update execution status (success/error/warning)
    s.flowRepo.UpdateExecutionStatus(...)
    
    // 5. Update flow statistics
    s.flowRepo.UpdateFlowStatistics(...)
    
    return execution, nil
}
```

**ProcessBackupFlow Logic:**

```go
func (s *ProtectionFlowService) ProcessBackupFlow(ctx context.Context, flow *ProtectionFlow, execution *ProtectionFlowExecution) error {
    // 1. Resolve target VMs
    var vmContexts []string
    if flow.TargetType == "group" {
        // Get all enabled VMs in group
        members := s.machineGroupSvc.GetGroupMembers(ctx, flow.TargetID)
        for _, member := range members {
            if member.Enabled {
                vmContexts = append(vmContexts, member.VMContextID)
            }
        }
    } else {
        // Single VM
        vmContexts = []string{flow.TargetID}
    }
    
    // 2. Execute backup for each VM
    var createdJobIDs []string
    for _, contextID := range vmContexts {
        vmCtx := s.vmContextRepo.GetByContextID(ctx, contextID)
        
        // Call existing backup API
        backupResp := s.backupService.StartBackup(ctx, &BackupStartRequest{
            VMName:       vmCtx.VMName,
            RepositoryID: *flow.RepositoryID,
            BackupType:   "auto", // Will auto-detect full vs incremental
        })
        
        createdJobIDs = append(createdJobIDs, backupResp.BackupID)
    }
    
    // 3. Update execution with results
    execution.JobsCreated = len(createdJobIDs)
    execution.VMsProcessed = len(vmContexts)
    execution.CreatedJobIDs = createdJobIDs
    
    return nil
}
```

**CRITICAL PATTERNS:**
- Use JobLog for tracking: `s.tracker.StartJob()`, `s.tracker.RunStep()`
- Handle errors gracefully: partial success = "warning" status
- Update statistics: increment success/failure counts
- Link executions: store created backup_job_ids

---

### Task 4: Scheduler Integration (3-4 hours)
**Modify:** `sha/services/scheduler_service.go`

**Goal:** Extend existing scheduler to support protection flows.

**Add to SchedulerService:**

```go
type SchedulerService struct {
    // ... existing fields
    flowService *ProtectionFlowService  // NEW
    activeFlowSchedules map[string]cron.EntryID  // NEW
}

// NEW METHOD
func (s *SchedulerService) RegisterFlowSchedule(flowID string, scheduleID string) error {
    schedule := s.repository.GetScheduleByID(scheduleID)
    
    // Add cron job
    entryID, err := s.cron.AddFunc(schedule.CronExpression, func() {
        s.ExecuteScheduledFlow(context.Background(), flowID, scheduleID)
    })
    
    s.activeFlowSchedules[flowID] = entryID
    return nil
}

// NEW METHOD
func (s *SchedulerService) ExecuteScheduledFlow(ctx context.Context, flowID string, scheduleID string) {
    logger := s.jobTracker.Logger(ctx)
    logger.Info("Executing scheduled flow", "flow_id", flowID)
    
    // Delegate to FlowService
    execution, err := s.flowService.ExecuteFlow(ctx, flowID, "scheduled")
    if err != nil {
        logger.Error("Flow execution failed", "error", err)
    }
}
```

**When to Register:**
- When flow is created with schedule_id
- When flow schedule_id is updated
- When flow is enabled

**When to Unregister:**
- When flow is deleted
- When flow is disabled
- When schedule_id is removed

---

### Task 5: API Handlers (4-5 hours)
**Create:** `sha/api/handlers/protection_flow_handlers.go`

Implement 12 endpoints (see job sheet for complete list):

**Core Endpoints:**
```go
// POST /api/v1/protection-flows - Create flow
func (h *ProtectionFlowHandler) CreateFlow(w http.ResponseWriter, r *http.Request) {
    var req CreateFlowRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Validate request
    if err := h.validateFlowRequest(&req); err != nil {
        h.writeError(w, 400, err.Error())
        return
    }
    
    // Create flow
    flow, err := h.flowService.CreateFlow(ctx, &req)
    if err != nil {
        h.writeError(w, 500, err.Error())
        return
    }
    
    // Register schedule if provided
    if flow.ScheduleID != nil {
        h.schedulerService.RegisterFlowSchedule(flow.ID, *flow.ScheduleID)
    }
    
    h.writeJSON(w, 201, flow)
}

// POST /api/v1/protection-flows/{id}/execute - Manual execution
func (h *ProtectionFlowHandler) ExecuteFlow(w http.ResponseWriter, r *http.Request) {
    flowID := mux.Vars(r)["id"]
    
    execution, err := h.flowService.ExecuteFlow(ctx, flowID, "manual")
    if err != nil {
        h.writeError(w, 500, err.Error())
        return
    }
    
    h.writeJSON(w, 200, execution)
}
```

**Register Routes:**
```go
// In server.go
r := router.PathPrefix("/api/v1/protection-flows").Subrouter()
r.HandleFunc("", handler.CreateFlow).Methods("POST")
r.HandleFunc("", handler.ListFlows).Methods("GET")
r.HandleFunc("/{id}", handler.GetFlow).Methods("GET")
r.HandleFunc("/{id}", handler.UpdateFlow).Methods("PUT")
r.HandleFunc("/{id}", handler.DeleteFlow).Methods("DELETE")
r.HandleFunc("/{id}/execute", handler.ExecuteFlow).Methods("POST")
// ... more routes
```

**Response Format (CRITICAL):**
- Match schemas in job sheet EXACTLY
- Include relationships (schedule_name, repository_name, target_name)
- Use proper HTTP status codes (201 for create, 404 for not found, etc.)
- Return descriptive error messages

---

### Task 6: GUI Integration (6-8 hours)

**Create API Service:**  
`sendense-gui/src/services/protectionFlowsApi.ts`

```typescript
export const protectionFlowsApi = {
  listFlows: async (filters?: FlowFilters): Promise<{ flows: ProtectionFlow[]; total: number }> => {
    const response = await apiClient.get('/protection-flows', { params: filters });
    return response.data;
  },
  
  createFlow: async (data: CreateFlowRequest): Promise<ProtectionFlow> => {
    const response = await apiClient.post('/protection-flows', data);
    return response.data;
  },
  
  executeFlow: async (id: string): Promise<FlowExecution> => {
    const response = await apiClient.post(`/protection-flows/${id}/execute`);
    return response.data;
  },
  
  // ... more methods (see job sheet)
};
```

**Wire Components:**

1. **app/protection-flows/page.tsx:**
   - Replace `mockFlows` with `useState<ProtectionFlow[]>`
   - Add `useEffect` to fetch flows on mount
   - Call `protectionFlowsApi.listFlows()` in `fetchFlows()`
   - Update `handleRunNow` to call `protectionFlowsApi.executeFlow()`

2. **CreateFlowModal.tsx:**
   - Add form fields: name, description, type, target, repository, policy, schedule
   - Call `protectionFlowsApi.createFlow()` on submit
   - Show loading state while submitting
   - Show error message on failure

3. **FlowDetailsPanel.tsx:**
   - Display real flow data from props
   - Show execution history from `protectionFlowsApi.getFlowExecutions()`
   - Add "Run Now" button that calls `executeFlow`

**CRITICAL:**
- Remove ALL mock data
- Add loading states (skeletons)
- Add error handling (display to user)
- Use TypeScript strict mode (no `any` types)
- Follow existing GUI patterns from Protection Groups page

---

### Task 7: Testing and Documentation (3-4 hours)

**Unit Tests:**
```go
// flow_repository_test.go
func TestCreateFlow(t *testing.T) {
    repo := NewFlowRepository(db)
    flow := &ProtectionFlow{
        Name:         "Test Flow",
        FlowType:     "backup",
        TargetType:   "vm",
        TargetID:     "ctx-test-vm",
        RepositoryID: intPtr(1),
        Enabled:      true,
    }
    err := repo.CreateFlow(context.Background(), flow)
    assert.NoError(t, err)
    assert.NotEmpty(t, flow.ID)
}

// protection_flow_service_test.go
func TestExecuteBackupFlow(t *testing.T) {
    // Mock dependencies
    // Create test flow
    // Execute flow
    // Assert backup API called
    // Assert execution recorded
}
```

**Integration Tests:**
```bash
# Create flow via API
curl -X POST http://localhost:8082/api/v1/protection-flows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Backup Flow",
    "flow_type": "backup",
    "target_type": "vm",
    "target_id": "ctx-pgtest1-20251006-203401",
    "repository_id": 1,
    "schedule_id": "schedule-daily-2am",
    "enabled": true
  }'

# Execute manually
curl -X POST http://localhost:8082/api/v1/protection-flows/{id}/execute

# Check execution history
curl http://localhost:8082/api/v1/protection-flows/{id}/executions
```

**Update Documentation:**
1. Add new endpoints to `api-documentation/OMA.md`
2. Update `start_here/PHASE_1_CONTEXT_HELPER.md` with flow info
3. Create user guide: "How to Create Protection Flows"

---

## ⚠️ CRITICAL RULES

### Database Rules
1. **Use VARCHAR(64) for all IDs** (consistent with existing schema)
2. **Use ENUM types** for status/type fields
3. **Add indexes** on foreign keys and commonly queried fields
4. **CASCADE DELETE** for child records (executions)
5. **Use GORM tags** that match existing patterns

### Go Code Rules
1. **Follow existing patterns** from scheduler_service.go and machine_group_service.go
2. **Use JobLog** for all tracking (`tracker.StartJob()`, `tracker.RunStep()`)
3. **Use context.Context** for all service methods
4. **Handle errors gracefully** (partial success = warning)
5. **No monster code** - keep functions under 100 lines

### API Rules
1. **Match response schemas** in job sheet EXACTLY
2. **Use proper HTTP status codes** (201, 404, 500, etc.)
3. **Include relationships** in responses (schedule_name, etc.)
4. **Validate requests** before processing
5. **Return descriptive errors** to GUI

### GUI Rules
1. **Remove ALL mock data** - use real API
2. **Add loading states** - skeletons for tables/cards
3. **Add error handling** - display errors to user
4. **TypeScript strict mode** - no `any` types
5. **Follow existing patterns** from Protection Groups page

---

## 🎯 Success Criteria

### Minimum Viable Product (MVP)
- [ ] Create backup flow for single VM via GUI
- [ ] Flow executes manually and creates backup
- [ ] Flow executes automatically on schedule
- [ ] Execution history visible in GUI
- [ ] Flow statistics update correctly

### Full Feature Set
- [ ] All CRUD operations work
- [ ] Group-level flows work
- [ ] Enable/disable flows
- [ ] Flow validation works
- [ ] Error handling complete
- [ ] Tests passing
- [ ] Documentation complete

---

## 🚫 What NOT To Do

1. **Don't reinvent scheduler** - reuse existing SchedulerService
2. **Don't create new cron logic** - extend existing
3. **Don't bypass JobLog** - use it for all tracking
4. **Don't leave mock data** - wire everything to real APIs
5. **Don't skip validation** - validate flows before creating
6. **Don't ignore errors** - handle gracefully and inform user

---

## 🔍 Testing Checklist

### Backend Tests
- [ ] Create flow with valid data
- [ ] Create flow with invalid data (should fail)
- [ ] Execute flow for single VM
- [ ] Execute flow for protection group
- [ ] Scheduled execution works
- [ ] Manual execution works
- [ ] Enable/disable works
- [ ] Delete flow removes schedule

### GUI Tests
- [ ] Flow list loads from API
- [ ] Create flow modal works
- [ ] Flow execution button works
- [ ] Loading states show
- [ ] Errors display to user
- [ ] No console errors
- [ ] No TypeScript errors

### End-to-End Test
1. Create protection group with 2 VMs
2. Create schedule (every 5 minutes)
3. Create backup flow for group
4. Wait 5 minutes
5. Verify flow executed
6. Verify 2 backups created
7. Check execution history
8. Verify statistics updated

---

## 📞 When You're Stuck

1. **Database issues?** Check existing migrations in `sha/database/migrations/`
2. **Service patterns?** Look at `scheduler_service.go` and `machine_group_service.go`
3. **API patterns?** Check `machine_group_management.go` handlers
4. **GUI patterns?** Look at Protection Groups page (`app/protection-groups/page.tsx`)
5. **Backup API?** See `HANDOVER-GUI-BACKUP-RESTORE-INTEGRATION.md`

---

## 🚀 Getting Started

**Step 1:** Read the job sheet top to bottom  
**Step 2:** Study the existing scheduler service  
**Step 3:** Start with Task 1 (database schema)  
**Step 4:** Build incrementally, test each layer  
**Step 5:** Wire GUI last (backend must be solid first)  

**Remember:** You have all the pieces. This is the glue layer that ties them together. The hard work (backup engine, scheduler, groups) is already done. You're just connecting the dots.

---

**GO BUILD SOMETHING AWESOME! 🚀**

The Sendense team is counting on you. This is the key to making Phase 1 fully operational. No pressure! 😎


