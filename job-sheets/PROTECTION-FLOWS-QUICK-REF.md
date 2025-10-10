# Protection Flows Engine - Quick Reference

**Status:** 🔴 Not Started  
**Phase:** Phase 1 Extension  
**Priority:** HIGH  
**Effort:** 3-4 days  
**Blocking:** GUI full functionality

---

## 📋 Overview

**What:** Build orchestration layer that ties together VMs, Protection Groups, Schedules, Repositories, and Policies into executable "flows"

**Why:** GUI Protection Flows page is 100% mock data. Backend has no flow concept. This blocks automated protection workflows.

**How:** Extend existing scheduler service, add flow abstraction layer, wire GUI to real APIs.

---

## 🎯 Quick Facts

### What Exists ✅
- Backup API (`POST /api/v1/backups`)
- Protection Groups API (`/api/v1/machine-groups`)
- Scheduler API (`/api/v1/schedules`)
- Repository API (`/api/v1/repositories`)
- Backup Policies API (`/api/v1/policies`)

### What's Missing ❌
- Protection Flow abstraction
- Flow-to-components binding
- Flow execution orchestration
- Flow status tracking
- GUI integration

### What We're Reusing ♻️
- Existing scheduler service (`robfig/cron/v3`)
- Existing JobLog tracking
- Existing backup engine
- Existing group management

---

## 📚 Key Documents

### Implementation Docs (Read in Order)
1. **Main Job Sheet:** `2025-10-09-protection-flows-engine.md` (BIBLE)
2. **Grok Prompt:** `GROK-PROMPT-protection-flows-engine.md` (Quick Guide)
3. **This File:** Quick reference

### Context Docs
- **Phase 1 Context:** `/sendense/start_here/PHASE_1_CONTEXT_HELPER.md`
- **Phase 1 Goals:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`
- **API Docs:** `/sendense/source/current/api-documentation/OMA.md`
- **Backup Integration:** `/sendense/HANDOVER-GUI-BACKUP-RESTORE-INTEGRATION.md`

### Reference Code
- **Scheduler Service:** `sha/services/scheduler_service.go` (REUSE PATTERNS)
- **Machine Group Service:** `sha/services/machine_group_service.go` (REFERENCE)
- **Group Handlers:** `sha/api/handlers/machine_group_management.go` (REFERENCE)
- **GUI Groups Page:** `sendense-gui/app/protection-groups/page.tsx` (REFERENCE)

---

## 🏗️ Architecture at a Glance

```
Protection Flow = Target + Schedule + Repository + Policy
                      ↓
              Flow Execution
                      ↓
         Resolve VMs → Execute Backups → Track Status
                      ↓
              Update Statistics
```

**Flow Types:**
- `backup` - Backup VMs to repository (Phase 1)
- `replication` - Replicate VMs to destination (Phase 5)

**Target Types:**
- `vm` - Single VM (by context_id)
- `group` - Protection group (all enabled VMs)

---

## 🗄️ Database Schema Summary

### Table 1: `protection_flows`
**Purpose:** Flow definitions

**Key Fields:**
- `id` - UUID primary key
- `name` - Unique flow name
- `flow_type` - 'backup' or 'replication'
- `target_type` - 'vm' or 'group'
- `target_id` - context_id or group_id
- `repository_id` - FK to repositories
- `schedule_id` - FK to schedules
- `enabled` - On/off switch
- `last_execution_status` - 'success', 'warning', 'error', etc.
- `last_execution_time` - Timestamp
- `next_execution_time` - Timestamp
- Statistics: `total_executions`, `successful_executions`, `failed_executions`

### Table 2: `protection_flow_executions`
**Purpose:** Execution history

**Key Fields:**
- `id` - UUID primary key
- `flow_id` - FK to flows (CASCADE DELETE)
- `status` - 'pending', 'running', 'success', 'warning', 'error', 'cancelled'
- `execution_type` - 'scheduled', 'manual', 'api'
- Job tracking: `jobs_created`, `jobs_completed`, `jobs_failed`, `jobs_skipped`
- `vms_processed` - Count of VMs processed
- `bytes_transferred` - Total bytes
- `created_job_ids` - JSON array of backup_job_id or replication_job_id
- Timing: `started_at`, `completed_at`, `execution_time_seconds`

---

## 🔌 API Endpoints Summary

### Core CRUD
```
POST   /api/v1/protection-flows              Create flow
GET    /api/v1/protection-flows              List flows (with filters)
GET    /api/v1/protection-flows/{id}         Get flow details
PUT    /api/v1/protection-flows/{id}         Update flow
DELETE /api/v1/protection-flows/{id}         Delete flow
```

### Control
```
PATCH  /api/v1/protection-flows/{id}/enable  Enable flow
PATCH  /api/v1/protection-flows/{id}/disable Disable flow
```

### Execution
```
POST   /api/v1/protection-flows/{id}/execute     Manual execution
GET    /api/v1/protection-flows/{id}/executions  Execution history
GET    /api/v1/protection-flows/{id}/status      Current status + next run
POST   /api/v1/protection-flows/{id}/test        Validate config (dry run)
```

### Summary
```
GET    /api/v1/protection-flows/summary      Aggregated statistics
```

---

## 🔨 Implementation Tasks

| # | Task | Effort | Files |
|---|------|--------|-------|
| 1 | Database Schema | 2-4h | `sha/database/migrations/20251009_*.sql` |
| 2 | Go Models & Repo | 3-4h | `sha/database/models.go`, `flow_repository.go` |
| 3 | Flow Service | 4-6h | `sha/services/protection_flow_service.go` |
| 4 | Scheduler Integration | 3-4h | `sha/services/scheduler_service.go` (modify) |
| 5 | API Handlers | 4-5h | `sha/api/handlers/protection_flow_handlers.go` |
| 6 | GUI Integration | 6-8h | `sendense-gui/src/services/protectionFlowsApi.ts` + components |
| 7 | Testing & Docs | 3-4h | Unit tests, integration tests, API docs |

**Total:** 25-35 hours (3-4 days)

---

## 🎯 MVP Success Criteria

### Must Work
- [ ] Create backup flow for single VM via GUI
- [ ] Flow executes manually (button click) and creates backup
- [ ] Flow executes automatically on schedule
- [ ] Execution history visible in GUI
- [ ] Flow statistics update correctly

### Must Not Break
- [ ] Existing backup API still works
- [ ] Existing scheduler still works for replications
- [ ] Protection Groups page unaffected
- [ ] No database corruption

---

## 🚨 Critical Integration Points

### 1. Scheduler Service
**File:** `sha/services/scheduler_service.go`

**Add:**
- `flowService *ProtectionFlowService` field
- `activeFlowSchedules map[string]cron.EntryID` tracking
- `RegisterFlowSchedule(flowID, scheduleID)` method
- `ExecuteScheduledFlow(ctx, flowID, scheduleID)` method

**When to register:**
- Flow created with schedule_id
- Flow enabled
- Schedule_id updated

**When to unregister:**
- Flow deleted
- Flow disabled
- Schedule_id removed

### 2. Backup Service
**File:** `sha/services/backup_service.go` (existing)

**Integration:**
- Flow service calls `backupService.StartBackup()` for each VM
- Flow tracks returned `backup_id` in `created_job_ids`
- Flow monitors backup status (optional for v1)

### 3. Machine Group Service
**File:** `sha/services/machine_group_service.go` (existing)

**Integration:**
- Flow service calls `GetGroupMembers(groupID)` to resolve VMs
- Filters to only enabled members
- Extracts `vm_context_id` for each member

### 4. GUI Components
**Files:** `sendense-gui/app/protection-flows/*.tsx`

**Replace:**
- `mockFlows` → `useState<ProtectionFlow[]>()` + API fetch
- Hardcoded actions → API calls (`createFlow`, `executeFlow`, etc.)
- Mock status → Real status from backend

---

## 📊 Data Flow Example

### Creating a Flow
```
GUI CreateFlowModal
    ↓ (POST /api/v1/protection-flows)
API Handler (CreateFlow)
    ↓
Flow Service (CreateFlow)
    ↓ (validates config)
Flow Repository (CreateFlow)
    ↓ (inserts row)
Database (protection_flows)
    ↓ (returns flow)
Scheduler Service (RegisterFlowSchedule)
    ↓ (adds cron job)
Cron Scheduler
```

### Executing a Flow (Scheduled)
```
Cron Trigger (time match)
    ↓
Scheduler Service (ExecuteScheduledFlow)
    ↓
Flow Service (ExecuteFlow)
    ↓
Flow Repository (CreateExecution with status='running')
    ↓
Flow Service (ProcessBackupFlow)
    ├→ Resolve target VMs (from group or single VM)
    └→ For each VM:
        ├→ Backup Service (StartBackup)
        └→ Track backup_job_id
    ↓
Flow Repository (UpdateExecutionStatus with status='success')
    ↓
Flow Repository (UpdateFlowStatistics)
```

### Viewing Flow in GUI
```
GUI Page Load
    ↓ (GET /api/v1/protection-flows)
API Handler (ListFlows)
    ↓
Flow Service (ListFlows)
    ↓
Flow Repository (ListFlows with joins)
    ↓ (loads schedule, repository, policy names)
Database (protection_flows + joins)
    ↓ (returns flows with relationships)
GUI (displays flows in table)
```

---

## 🧪 Testing Strategy

### Unit Tests (Go)
```go
// Test flow creation
TestCreateFlow_ValidData
TestCreateFlow_InvalidData
TestCreateFlow_DuplicateName

// Test flow execution
TestExecuteFlow_SingleVM
TestExecuteFlow_Group
TestExecuteFlow_EmptyGroup

// Test validation
TestValidateFlow_MissingRepository
TestValidateFlow_InvalidSchedule
```

### Integration Tests (curl)
```bash
# Create flow
curl -X POST http://localhost:8082/api/v1/protection-flows \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Flow","flow_type":"backup",...}'

# Execute flow
curl -X POST http://localhost:8082/api/v1/protection-flows/{id}/execute

# Check history
curl http://localhost:8082/api/v1/protection-flows/{id}/executions
```

### GUI Tests (Manual)
1. Create flow via modal
2. Verify flow appears in list
3. Click "Run Now" button
4. Verify execution history updates
5. Wait for scheduled execution
6. Verify automatic execution worked

---

## 🐛 Common Issues and Solutions

### Issue: Flow doesn't execute on schedule
**Check:**
- Schedule enabled? (`SELECT enabled FROM replication_schedules WHERE id=...`)
- Flow enabled? (`SELECT enabled FROM protection_flows WHERE id=...`)
- Cron registered? (Check `activeFlowSchedules` map)
- Cron running? (Check scheduler service logs)

### Issue: Execution status stuck on "running"
**Cause:** Flow execution crashed without updating status

**Fix:**
- Add `defer` block to update status even on panic
- Add timeout handling
- Implement execution watchdog

### Issue: GUI shows stale data
**Cause:** Not refreshing after operations

**Fix:**
- Call `fetchFlows()` after create/update/delete
- Implement polling for running executions
- Use WebSocket for real-time updates (v2)

### Issue: Foreign key constraint failure
**Cause:** Referenced entity doesn't exist

**Fix:**
- Validate repository_id exists before creating flow
- Validate schedule_id exists before creating flow
- Validate target_id exists (vm_context_id or group_id)

---

## 📈 Performance Considerations

### Database
- **Index on `enabled`** - Fast query for enabled flows
- **Index on `schedule_id`** - Fast lookup of flows by schedule
- **Index on `last_execution_time`** - Fast status queries
- **Composite index on `(target_type, target_id)`** - Fast target lookups

### Execution
- **Concurrent execution limits** - Reuse scheduler's `max_concurrent_jobs`
- **Batch VM processing** - Process group VMs in parallel (Phase 2)
- **Execution timeout** - Kill runaway executions after N minutes

### GUI
- **Pagination** - Don't load all flows at once if >100
- **Debounce filters** - Don't hammer API on every keystroke
- **Cache flow list** - Refresh on interval, not every render

---

## 🔐 Security Considerations

### Authentication
- All endpoints require authentication (existing middleware)
- User must have permission to access target VMs/groups
- Audit log all flow operations

### Validation
- Validate target exists (VM or group)
- Validate repository exists and accessible
- Validate schedule syntax (cron expression)
- Prevent SQL injection (use parameterized queries)

### Authorization (Phase 7 - Multi-Tenant)
- Add `tenant_id` to flows
- Filter flows by tenant
- Prevent cross-tenant access

---

## 🚀 Future Enhancements (Not in Scope)

### Phase 5 - Replication Flows
- Add `destination_type` field ('ossea', 'vmware', 'hyperv')
- Add `destination_config` JSON field
- Implement `ProcessReplicationFlow()` method
- Wire replication API

### Phase 7 - MSP Platform
- Add `tenant_id` field
- Multi-tenant flow isolation
- Tenant-specific repositories
- Cross-tenant reporting

### Advanced Features
- Flow dependencies (chain flows)
- Conditional execution (only if conditions met)
- Flow templates (pre-configured flows)
- Flow cloning (duplicate flows)
- Bulk operations (enable/disable multiple flows)

---

## 📞 Getting Help

### Stuck on Database?
- Check: `sha/database/migrations/` for existing patterns
- Check: `sha/database/models.go` for existing structs
- Check: GORM documentation for tag syntax

### Stuck on Service Layer?
- Reference: `sha/services/scheduler_service.go` (BEST EXAMPLE)
- Reference: `sha/services/machine_group_service.go`
- Pattern: JobLog for tracking, context for cancellation

### Stuck on API?
- Reference: `sha/api/handlers/machine_group_management.go` (BEST EXAMPLE)
- Pattern: Decode request → validate → call service → encode response
- Use: `h.writeErrorResponse()` and `h.writeJSONResponse()` helpers

### Stuck on GUI?
- Reference: `sendense-gui/app/protection-groups/page.tsx` (BEST EXAMPLE)
- Pattern: useState + useEffect + API calls + loading/error states
- Use: Existing components (Button, Card, Badge, etc.)

---

## ✅ Pre-Deployment Checklist

### Code Quality
- [ ] No commented-out code
- [ ] No debug print statements
- [ ] No hardcoded values (use config)
- [ ] No TODOs in critical paths
- [ ] Proper error messages (descriptive)

### Testing
- [ ] All unit tests passing
- [ ] Integration tests passing
- [ ] Manual testing complete
- [ ] Edge cases tested (empty groups, invalid targets, etc.)

### Documentation
- [ ] API docs updated (`api-documentation/OMA.md`)
- [ ] Architecture docs updated
- [ ] User guide created
- [ ] Code comments for complex logic

### Database
- [ ] Migrations tested (up and down)
- [ ] Indexes created
- [ ] Foreign keys validated
- [ ] Test data cleaned up

### Deployment
- [ ] Backup database before migration
- [ ] Test on staging first
- [ ] Monitor logs after deployment
- [ ] Verify flows execute correctly

---

## 🎓 Learning Resources

### Go Best Practices
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go by Example](https://gobyexample.com/)
- [GORM Documentation](https://gorm.io/docs/)

### Cron Expressions
- [Crontab Guru](https://crontab.guru/) - Cron expression tester
- [robfig/cron docs](https://pkg.go.dev/github.com/robfig/cron/v3)

### React/TypeScript
- [React TypeScript Cheatsheet](https://react-typescript-cheatsheet.netlify.app/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/handbook/intro.html)

---

**Version:** 1.0  
**Last Updated:** 2025-10-09  
**Maintainer:** Sendense Engineering Team


