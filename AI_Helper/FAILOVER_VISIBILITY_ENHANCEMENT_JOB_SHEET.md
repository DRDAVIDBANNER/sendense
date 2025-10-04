# Failover & Rollback Visibility Enhancement Job Sheet

**Created**: October 3, 2025  
**Status**: 🔴 **NOT STARTED**  
**Priority**: 🟡 **HIGH** - UX/Visibility Issue  
**Estimated Effort**: 1-2 days  
**Current Phase**: Phase 1 - Error Sanitization  

---

## ✅ **PROGRESS TRACKER**

**Overall Completion**: 0/5 phases complete

- [ ] **Phase 1**: Error Message Sanitization (0/6 tasks)
- [ ] **Phase 2**: Job Summary Persistence (0/5 tasks)
- [ ] **Phase 3**: Unified Recent Jobs API (0/4 tasks)
- [ ] **Phase 4**: Enhanced Progress Responses (0/3 tasks)
- [ ] **Phase 5**: Testing & Validation (0/4 tasks)

---

## 📋 **EXECUTIVE SUMMARY**

**Problem**: Failover and rollback jobs have poor visibility compared to replication jobs:
- ❌ No "recent job" tracking like replications
- ❌ Jobs disappear from view quickly after completion/failure
- ❌ Different UI location (sidebar bottom vs. top)
- ❌ Technical error messages expose internal details (virt-v2v, driver paths, etc.)
- ❌ When failures occur, unclear where and why without diving into logs

**Solution**: Unify failover/rollback visibility with replication jobs WITHOUT major schema changes:
- ✅ Use existing `vm_replication_contexts` fields to track failover jobs
- ✅ Add sanitized, user-friendly error messages
- ✅ Create unified "recent jobs" view combining replication + failover
- ✅ Persistent job summaries for post-completion visibility
- ✅ Clear failure stage indication with actionable information

---

## 🎯 **KEY REQUIREMENTS**

### **Must Haves**
1. **Unified Job History** - Failover jobs shown alongside replication jobs
2. **Persistent Visibility** - Jobs remain visible after completion/failure
3. **Clear Error Location** - Know which step failed
4. **Sanitized Messages** - No "virt-v2v", "VirtIO", etc. → "Driver Injection"
5. **No Major Schema Changes** - Use existing fields creatively

### **Technical Constraints**
- ❌ No new tables or major migrations
- ✅ Can add JSON columns for metadata
- ✅ Can use existing vm_replication_contexts fields
- ✅ Can enhance API responses
- ✅ Can improve error message formatting

---

## 🏗️ **PROPOSED ARCHITECTURE**

### **Approach: Lightweight Job Summary Tracking**

Instead of creating new tables, store job summaries in `vm_replication_contexts`:

```sql
-- Minimal schema addition (lightweight)
ALTER TABLE vm_replication_contexts
ADD COLUMN last_operation_summary JSON NULL COMMENT 'Summary of most recent operation (replication/failover/rollback)';

CREATE INDEX idx_vm_contexts_last_operation ON vm_replication_contexts(
    (CAST(last_operation_summary->>'$.timestamp' AS DATETIME))
);
```

**JSON Structure**:
```json
{
  "job_id": "abc-123-uuid",
  "external_job_id": "unified-test-failover-pgtest1-1234567890",
  "operation_type": "test_failover",
  "status": "failed",
  "progress": 70.0,
  "failed_step": "driver_injection",
  "error_message": "Driver injection failed - Windows drivers could not be installed",
  "error_category": "compatibility",
  "timestamp": "2025-10-03T14:22:00Z",
  "duration_seconds": 120,
  "steps_completed": 7,
  "steps_total": 10
}
```

---

## 📊 **DETAILED TASK BREAKDOWN**

---

### **Phase 1: Error Message Sanitization** 🚨 **START HERE**

**Goal**: Convert all technical error messages to user-friendly, actionable guidance

#### **Task 1.1: Create Error Sanitizer Module**

**File**: `source/current/oma/failover/error_sanitizer.go` (NEW)

File: `source/current/oma/failover/error_sanitizer.go` (NEW)

```go
package failover

// SanitizedMessage provides user-friendly error messaging
type SanitizedMessage struct {
    UserMessage    string   // Clean message for GUI
    TechnicalDetails string // Full details for logs only
    Category       string   // Error category: compatibility, network, storage, configuration
    ActionableSteps []string // What user can do about it
}

// SanitizeFailoverError converts technical errors to user-friendly messages
func SanitizeFailoverError(step string, err error) SanitizedMessage {
    errorStr := err.Error()
    
    switch step {
    case "virtio-driver-injection":
        return sanitizeDriverInjectionError(errorStr)
    case "vm-creation":
        return sanitizeVMCreationError(errorStr)
    case "network-configuration":
        return sanitizeNetworkError(errorStr)
    case "volume-attachment":
        return sanitizeVolumeError(errorStr)
    default:
        return SanitizedMessage{
            UserMessage: "Operation failed - see logs for details",
            TechnicalDetails: errorStr,
            Category: "unknown",
        }
    }
}

// sanitizeDriverInjectionError converts VirtIO/virt-v2v errors
func sanitizeDriverInjectionError(errMsg string) SanitizedMessage {
    // Pattern matching for common issues
    switch {
    case strings.Contains(errMsg, "virt-v2v"):
        return SanitizedMessage{
            UserMessage: "Driver injection failed - compatibility tool error",
            TechnicalDetails: errMsg,
            Category: "compatibility",
            ActionableSteps: []string{
                "Verify VM is Windows-based",
                "Check VM disk is accessible",
                "Ensure Windows is in a bootable state",
            },
        }
    case strings.Contains(errMsg, "VirtIO"):
        return SanitizedMessage{
            UserMessage: "Driver injection failed - cannot install required drivers",
            TechnicalDetails: errMsg,
            Category: "compatibility",
            ActionableSteps: []string{
                "VM may not be compatible with KVM virtualization",
                "Try live failover instead (no driver modification)",
            },
        }
    case strings.Contains(errMsg, "device"):
        return SanitizedMessage{
            UserMessage: "Driver injection failed - disk access error",
            TechnicalDetails: errMsg,
            Category: "storage",
            ActionableSteps: []string{
                "Verify volumes are attached",
                "Check disk is not corrupted",
            },
        }
    default:
        return SanitizedMessage{
            UserMessage: "Driver injection failed",
            TechnicalDetails: errMsg,
            Category: "compatibility",
        }
    }
}

// Additional sanitizers for other error types...
```

**Subtasks**:
- [ ] **1.1.1** Create `error_sanitizer.go` file structure
- [ ] **1.1.2** Define `SanitizedMessage` struct with all fields
- [ ] **1.1.3** Implement `SanitizeFailoverError()` main function
- [ ] **1.1.4** Implement `sanitizeDriverInjectionError()` (virt-v2v → "Driver injection")
- [ ] **1.1.5** Implement `sanitizeVMCreationError()` (CloudStack errors)
- [ ] **1.1.6** Implement `sanitizeNetworkError()` (network config issues)
- [ ] **1.1.7** Implement `sanitizeVolumeError()` (storage errors)
- [ ] **1.1.8** Implement `sanitizeVMAConnectivityError()` (VMA unreachable)
- [ ] **1.1.9** Add pattern matching for common error strings
- [ ] **1.1.10** Create actionable steps database

**Testing Criteria**:
- [ ] All "virt-v2v" references replaced
- [ ] All "VirtIO" references replaced
- [ ] No device paths exposed
- [ ] All errors have actionable steps

---

---

#### **Task 1.2: Create Step Name Mapper**

**File**: `source/current/oma/failover/step_display_names.go` (NEW)

Create user-friendly step name mapper:

```go
// GetUserFriendlyStepName converts internal step names to GUI-friendly names
func GetUserFriendlyStepName(internalStep string) string {
    stepNames := map[string]string{
        "validation":                    "Pre-flight Validation",
        "source-vm-power-off":           "Powering Off Source VM",
        "final-sync":                    "Final Data Synchronization",
        "multi-volume-snapshot-creation": "Creating Backup Snapshots",
        "virtio-driver-injection":       "Preparing Drivers for KVM", // ✅ Sanitized
        "vm-creation":                   "Creating Destination VM",
        "volume-attachment":             "Attaching Storage Volumes",
        "vm-startup-and-validation":     "Starting and Validating VM",
        "network-configuration":         "Configuring Network Adapters",
        "status-update":                 "Finalizing Operation",
        
        // Rollback steps
        "test-vm-shutdown":              "Shutting Down Test VM",
        "volume-detachment":             "Detaching Storage Volumes",
        "cloudstack-snapshot-rollback":  "Rolling Back to Backup",
        "test-vm-deletion":              "Removing Test VM",
    }
    
    if friendly, exists := stepNames[internalStep]; exists {
        return friendly
    }
    return internalStep // Fallback to original
}
```

**Subtasks**:
- [ ] **1.2.1** Create `step_display_names.go` file
- [ ] **1.2.2** Define complete step name mapping table (20+ steps)
- [ ] **1.2.3** Implement `GetUserFriendlyStepName()` function
- [ ] **1.2.4** Add step category tags (setup/execution/cleanup)
- [ ] **1.2.5** Create step description helper
- [ ] **1.2.6** Test all step names are mapped

**Testing Criteria**:
- [ ] All internal step names have friendly versions
- [ ] No technical jargon in step names
- [ ] Names are consistent across operations

---

---

### **Phase 2: Job Summary Persistence** 🟡 **HIGH PRIORITY**

**Goal**: Store operation summaries for persistent visibility

#### **Task 2.1: Schema Enhancement (Minimal)**

**Option A: Single JSON Column** (Recommended - minimal change)

```sql
-- Add one column to store operation summaries
ALTER TABLE vm_replication_contexts
ADD COLUMN last_operation_summary JSON NULL COMMENT 'Summary of most recent operation for GUI visibility';

-- Index for querying recent operations
CREATE INDEX idx_vm_contexts_last_op_time ON vm_replication_contexts(
    (CAST(last_operation_summary->>'$.timestamp' AS DATETIME))
);
```

**Option B: Reuse Existing Fields** (Zero schema change)

Use existing fields creatively:
- `current_job_id` - Store failover job ID (like we do for replication)
- Use JSON in existing text fields (if available)
- Store in metadata table that might already exist

**Subtasks**:
- [ ] **2.1.1** Review existing vm_replication_contexts schema
- [ ] **2.1.2** Decide: JSON column OR reuse existing fields
- [ ] **2.1.3** Create migration `20251003160000_add_operation_summary.up.sql`
- [ ] **2.1.4** Create down migration `20251003160000_add_operation_summary.down.sql`
- [ ] **2.1.5** Test migration on dev database
- [ ] **2.1.6** Document JSON structure spec
- [ ] **2.1.7** Test JSON query performance

**Testing Criteria**:
- [ ] Migration runs without errors
- [ ] Can store and retrieve JSON
- [ ] Index works for time-based queries
- [ ] No impact on existing queries

---

#### **Task 2.2: Update Failover Completion to Store Summary**

File: `source/current/oma/failover/unified_failover_engine.go`

Add method to store operation summary:

```go
// storeOperationSummary saves a summary of the failover/rollback operation to VM context
func (ufe *UnifiedFailoverEngine) storeOperationSummary(
    ctx context.Context,
    config *UnifiedFailoverConfig,
    jobID string,
    externalJobID string,
    status string,
    progress float64,
    failedStep string,
    err error,
) error {
    logger := ufe.jobTracker.Logger(ctx)
    
    // Get job details
    jobSummary, summaryErr := ufe.jobTracker.FindJobByAnyID(jobID)
    if summaryErr != nil {
        logger.Warn("Could not get job summary", "error", summaryErr)
    }
    
    // Build sanitized summary
    summary := map[string]interface{}{
        "job_id":          jobID,
        "external_job_id": externalJobID,
        "operation_type":  config.FailoverType, // "test" or "live"
        "status":          status,
        "progress":        progress,
        "timestamp":       time.Now(),
    }
    
    if jobSummary != nil {
        summary["steps_completed"] = jobSummary.Progress.CompletedSteps
        summary["steps_total"] = jobSummary.Progress.TotalSteps
        summary["duration_seconds"] = jobSummary.Progress.RuntimeSeconds
    }
    
    if failedStep != "" {
        // Sanitize the step name and error
        friendlyStepName := GetUserFriendlyStepName(failedStep)
        summary["failed_step"] = friendlyStepName
        
        if err != nil {
            sanitized := SanitizeFailoverError(failedStep, err)
            summary["error_message"] = sanitized.UserMessage
            summary["error_category"] = sanitized.Category
            summary["actionable_steps"] = sanitized.ActionableSteps
            // Don't include technical details in summary!
        }
    }
    
    // Store in VM context
    summaryJSON, _ := json.Marshal(summary)
    updates := map[string]interface{}{
        "last_operation_summary": string(summaryJSON),
        "updated_at": time.Now(),
    }
    
    err = ufe.db.GetGormDB().Model(&database.VMReplicationContext{}).
        Where("context_id = ?", config.ContextID).
        Updates(updates).Error
        
    if err != nil {
        logger.Error("Failed to store operation summary", "error", err)
        return err
    }
    
    logger.Info("✅ Stored operation summary for GUI visibility",
        "context_id", config.ContextID,
        "operation", config.FailoverType,
        "status", status)
    
    return nil
}
```

**Subtasks**:
- [ ] **2.2.1** Create `storeOperationSummary()` method in unified_failover_engine.go
- [ ] **2.2.2** Add helper to extract job details from JobLog
- [ ] **2.2.3** Integrate sanitization layer (use error_sanitizer.go)
- [ ] **2.2.4** Call on successful completion (ExecuteUnifiedFailover success path)
- [ ] **2.2.5** Call on failure (error handling in all phases)
- [ ] **2.2.6** Add to enhanced_cleanup_service.go for rollbacks
- [ ] **2.2.7** Test summary storage with real failover
- [ ] **2.2.8** Verify summaries persist across page refreshes

**Testing Criteria**:
- [ ] Summary stored on success
- [ ] Summary stored on failure with sanitized error
- [ ] Summary stored on rollback
- [ ] Can query summaries via API

---

#### **Task 2.3: Update VM Context API to Include Operation Summary**

File: `source/current/oma/api/handlers/vm_context.go` (or wherever VM context endpoints are)

Enhance VM context response to include operation summary:

```go
type VMContextResponse struct {
    ContextID string `json:"context_id"`
    VMName    string `json:"vm_name"`
    // ... existing fields
    
    // Recent operation summary
    LastOperation *OperationSummary `json:"last_operation,omitempty"`
}

type OperationSummary struct {
    JobID           string   `json:"job_id"`
    ExternalJobID   string   `json:"external_job_id"`
    OperationType   string   `json:"operation_type"` // "replication", "test_failover", "live_failover", "rollback"
    Status          string   `json:"status"`
    Progress        float64  `json:"progress"`
    FailedStep      string   `json:"failed_step,omitempty"`
    ErrorMessage    string   `json:"error_message,omitempty"`
    ErrorCategory   string   `json:"error_category,omitempty"`
    ActionableSteps []string `json:"actionable_steps,omitempty"`
    Timestamp       time.Time `json:"timestamp"`
    DurationSeconds int64    `json:"duration_seconds"`
}

// Parse last_operation_summary JSON and include in response
if vmContext.LastOperationSummary != "" {
    var opSummary OperationSummary
    if err := json.Unmarshal([]byte(vmContext.LastOperationSummary), &opSummary); err == nil {
        response.LastOperation = &opSummary
    }
}
```

**Subtasks**:
- [ ] **2.3.1** Add `LastOperationSummary` field to VMReplicationContext model
- [ ] **2.3.2** Create `OperationSummary` struct for API responses
- [ ] **2.3.3** Add JSON parsing in GetVMContext endpoint
- [ ] **2.3.4** Add to ListVMContexts endpoint
- [ ] **2.3.5** Include in VM context details API
- [ ] **2.3.6** Test API response structure
- [ ] **2.3.7** Update API documentation

**Testing Criteria**:
- [ ] JSON parses correctly
- [ ] API returns operation summary
- [ ] Null handling works (no summary = no error)
- [ ] Structure matches GUI expectations

---

---

### **Phase 3: Unified Recent Jobs API** 🟡 **HIGH PRIORITY**

**Goal**: Single API endpoint showing ALL operations (replication + failover + rollback)

#### **Task 3.1: Create Unified Jobs API Endpoint**

**New Endpoint**: `GET /api/v1/vm-contexts/{context_id}/recent-jobs`

Returns **all** recent operations (replication, failover, rollback) in one list:

```go
type UnifiedJobItem struct {
    JobID         string    `json:"job_id"`
    ExternalJobID string    `json:"external_job_id,omitempty"`
    JobType       string    `json:"job_type"` // "replication", "test_failover", "live_failover", "rollback"
    Status        string    `json:"status"`
    Progress      float64   `json:"progress"`
    StartedAt     time.Time `json:"started_at"`
    CompletedAt   *time.Time `json:"completed_at"`
    
    // User-friendly display
    DisplayName   string `json:"display_name"` // "Incremental Replication", "Test Failover", "Rollback"
    CurrentStep   string `json:"current_step"` // "Preparing Drivers for KVM" (sanitized)
    ErrorMessage  string `json:"error_message,omitempty"` // Sanitized
    
    // Source indicator
    DataSource    string `json:"data_source"` // "replication_jobs", "job_tracking"
}

func (h *VMContextHandler) GetRecentJobs(w http.ResponseWriter, r *http.Request) {
    contextID := mux.Vars(r)["context_id"]
    
    var allJobs []UnifiedJobItem
    
    // 1. Get replication jobs from replication_jobs table
    var replJobs []database.ReplicationJob
    h.db.GetGormDB().
        Where("vm_context_id = ?", contextID).
        Order("created_at DESC").
        Limit(10).
        Find(&replJobs)
    
    for _, job := range replJobs {
        allJobs = append(allJobs, UnifiedJobItem{
            JobID:       job.ID,
            JobType:     "replication",
            Status:      job.Status,
            Progress:    job.ProgressPercent,
            StartedAt:   job.CreatedAt,
            CompletedAt: job.CompletedAt,
            DisplayName: getReplicationDisplayName(job.ReplicationType),
            CurrentStep: job.CurrentOperation,
            ErrorMessage: sanitizeReplicationError(job.ErrorMessage),
            DataSource: "replication_jobs",
        })
    }
    
    // 2. Get failover/rollback jobs from job_tracking via JobLog
    if h.jobTracker != nil {
        jobSummaries, err := h.jobTracker.GetJobByContextID(contextID)
        if err == nil {
            for _, summary := range jobSummaries {
                if summary.Job.JobType == "failover" || summary.Job.JobType == "cleanup" {
                    allJobs = append(allJobs, UnifiedJobItem{
                        JobID:         summary.Job.ID,
                        ExternalJobID: getStringOrEmpty(summary.Job.ExternalJobID),
                        JobType:       getDisplayJobType(summary.Job.Operation),
                        Status:        string(summary.Job.Status),
                        Progress:      summary.Progress.StepCompletion,
                        StartedAt:     summary.Job.StartedAt,
                        CompletedAt:   summary.Job.CompletedAt,
                        DisplayName:   getFailoverDisplayName(summary.Job.Operation),
                        CurrentStep:   getCurrentStepFriendly(summary.Steps),
                        ErrorMessage:  sanitizeJobLogError(summary.Job.ErrorMessage),
                        DataSource:    "job_tracking",
                    })
                }
            }
        }
    }
    
    // 3. Sort by timestamp (most recent first)
    sort.Slice(allJobs, func(i, j int) bool {
        return allJobs[i].StartedAt.After(allJobs[j].StartedAt)
    })
    
    // 4. Limit to 20 most recent
    if len(allJobs) > 20 {
        allJobs = allJobs[:20]
    }
    
    respondJSON(w, http.StatusOK, allJobs)
}
```

**Subtasks**:
- [ ] **3.1.1** Create `GetRecentJobs()` handler in vm_context.go
- [ ] **3.1.2** Query replication_jobs table for context_id
- [ ] **3.1.3** Query job_tracking table via JobLog for context_id
- [ ] **3.1.4** Define `UnifiedJobItem` response struct
- [ ] **3.1.5** Implement job type detection (replication vs failover vs rollback)
- [ ] **3.1.6** Add display name generation (operation_type → friendly name)
- [ ] **3.1.7** Merge and sort by timestamp
- [ ] **3.1.8** Limit to 20 most recent
- [ ] **3.1.9** Apply sanitization to all error messages
- [ ] **3.1.10** Register endpoint in server.go

**Testing Criteria**:
- [ ] Returns jobs from both sources
- [ ] Sorted correctly (newest first)
- [ ] All job types included
- [ ] Errors are sanitized
- [ ] Performance acceptable (<100ms)

---

#### **Task 3.2: Add Operation Summary to VM Context**

Update failover engines to store summaries:

**On Success**:
```go
// At end of ExecuteUnifiedFailover
if err == nil {
    ufe.storeOperationSummary(ctx, config, jobID, externalJobID, "completed", 100.0, "", nil)
}
```

**On Failure**:
```go
// In error handling
if stepErr != nil {
    ufe.storeOperationSummary(ctx, config, jobID, externalJobID, "failed", 
        currentProgress, currentStepName, stepErr)
}
```

**Subtasks**:
- [ ] **3.2.1** Add summary storage call in ExecuteUnifiedFailover (success)
- [ ] **3.2.2** Add summary storage call in error handlers (failure)
- [ ] **3.2.3** Add summary storage call in EnhancedCleanupService (rollback)
- [ ] **3.2.4** Pass sanitized errors to storage
- [ ] **3.2.5** Include progress percentage at failure point
- [ ] **3.2.6** Test with successful operations
- [ ] **3.2.7** Test with failed operations
- [ ] **3.2.8** Verify summaries queryable via unified API

**Testing Criteria**:
- [ ] Summaries stored on all paths
- [ ] Includes sanitized error messages
- [ ] Includes failed step information
- [ ] Persists across OMA restarts

---

---

### **Phase 4: Enhanced Progress API Response** 🟢 **MEDIUM PRIORITY**

**Goal**: Real-time progress includes sanitized, actionable information

#### **Task 4.1: Enhance GetFailoverJobStatus Response**

File: `source/current/oma/api/handlers/failover.go:570`

Add sanitized information to existing response:

```go
response = JobStatusResponse{
    Success:   true,
    Message:   "...",
    JobID:     jobID,
    Status:    status,
    Progress:  progress.StepCompletion,
    StartTime: jobSummary.Job.StartedAt,
    Duration:  duration,
    
    // ✅ NEW: User-friendly additions
    CurrentStepFriendly: GetUserFriendlyStepName(getCurrentStep(jobSummary.Steps)),
    StepsCompleted:      progress.CompletedSteps,
    StepsTotal:          progress.TotalSteps,
    
    JobDetails: map[string]interface{}{
        // Existing fields...
        "total_steps":     progress.TotalSteps,
        "completed_steps": progress.CompletedSteps,
        
        // ✅ NEW: Sanitized error info
        "error_message_user": sanitizeErrorForUser(jobSummary.Job.ErrorMessage),
        "error_category":     categorizeError(jobSummary.Job.ErrorMessage),
        "actionable_steps":   getActionableSteps(jobSummary.Job.ErrorMessage),
        
        // ✅ NEW: Step details with friendly names
        "current_step_friendly": GetUserFriendlyStepName(getCurrentStep(jobSummary.Steps)),
        "failed_step_friendly":  getFailedStepFriendly(jobSummary.Steps),
        
        // Keep technical details for debugging (admin only)
        "technical_details": jobSummary.Job.ErrorMessage, // Full error
    },
}
```

**Subtasks**:
- [ ] **4.1.1** Update `JobStatusResponse` struct with new fields
- [ ] **4.1.2** Add `CurrentStepFriendly` field
- [ ] **4.1.3** Add `ErrorMessageUser` field (sanitized)
- [ ] **4.1.4** Add `ErrorCategory` field
- [ ] **4.1.5** Add `ActionableSteps` array field
- [ ] **4.1.6** Integrate error_sanitizer in GetFailoverJobStatus
- [ ] **4.1.7** Apply step name mapping to current step
- [ ] **4.1.8** Keep technical details in separate field (admin only)
- [ ] **4.1.9** Test response structure with GUI polling

**Testing Criteria**:
- [ ] Response includes sanitized messages
- [ ] Current step is user-friendly
- [ ] Actionable steps present for failures
- [ ] Technical details hidden by default
- [ ] GUI can display all fields

---

#### **Task 4.2: Add Step Progress Details**

Return individual step statuses with friendly names:

```go
"steps": [
    {
        "name": "virtio-driver-injection",
        "display_name": "Preparing Drivers for KVM",
        "status": "failed",
        "started_at": "2025-10-03T14:22:00Z",
        "error_message": "Driver injection failed - compatibility tool error",
        "error_category": "compatibility",
        "actionable_steps": [
            "Verify VM is Windows-based",
            "Try live failover instead"
        ]
    },
    // Other steps...
]
```

**Subtasks**:
- [ ] **4.2.1** Add `steps` array to JobStatusResponse
- [ ] **4.2.2** Include display_name for each step
- [ ] **4.2.3** Sanitize error messages per step
- [ ] **4.2.4** Add error_category per step
- [ ] **4.2.5** Include actionable_steps per failed step
- [ ] **4.2.6** Test with multi-step failure scenarios

**Testing Criteria**:
- [ ] All steps have friendly names
- [ ] Failed steps clearly indicated
- [ ] Error messages sanitized per step
- [ ] Actionable steps included

---

---

### **Phase 5: Testing & Validation** 🎯 **CRITICAL**

**Goal**: Comprehensive testing of all enhancements

#### **Task 5.1: Error Sanitization Testing**

**Subtasks**:
- [ ] **5.1.1** Test virt-v2v error sanitization
- [ ] **5.1.2** Test VirtIO error sanitization
- [ ] **5.1.3** Test CloudStack API error sanitization
- [ ] **5.1.4** Test network error sanitization
- [ ] **5.1.5** Test volume error sanitization
- [ ] **5.1.6** Test VMA connectivity error sanitization
- [ ] **5.1.7** Verify no technical terms exposed in any scenario
- [ ] **5.1.8** Verify all errors have actionable steps

**Testing Criteria**:
- [ ] 100% of errors sanitized
- [ ] Zero technical leaks
- [ ] All categories covered
- [ ] Actionable steps helpful

---

#### **Task 5.2: Job Summary Persistence Testing**

**Subtasks**:
- [ ] **5.2.1** Test summary storage on successful test failover
- [ ] **5.2.2** Test summary storage on failed test failover
- [ ] **5.2.3** Test summary storage on successful live failover
- [ ] **5.2.4** Test summary storage on failed live failover
- [ ] **5.2.5** Test summary storage on successful rollback
- [ ] **5.2.6** Test summary storage on failed rollback
- [ ] **5.2.7** Verify summaries persist across OMA restart
- [ ] **5.2.8** Test JSON query performance (< 50ms)

**Testing Criteria**:
- [ ] All operation types store summaries
- [ ] Summaries include all required fields
- [ ] Data persists correctly
- [ ] No performance degradation

---

#### **Task 5.3: Unified API Testing**

**Subtasks**:
- [ ] **5.3.1** Test recent jobs with only replication jobs
- [ ] **5.3.2** Test recent jobs with only failover jobs
- [ ] **5.3.3** Test recent jobs with mixed job types
- [ ] **5.3.4** Test sorting (newest first)
- [ ] **5.3.5** Test limit (max 20 jobs)
- [ ] **5.3.6** Test with empty job history
- [ ] **5.3.7** Verify sanitization applied to all jobs
- [ ] **5.3.8** Test response time (< 100ms)

**Testing Criteria**:
- [ ] All job types included
- [ ] Correct sort order
- [ ] Proper error sanitization
- [ ] Performance acceptable

---

#### **Task 5.4: End-to-End UX Testing**

**Subtasks**:
- [ ] **5.4.1** Test complete successful test failover workflow
- [ ] **5.4.2** Test failed driver injection scenario
- [ ] **5.4.3** Test failed VM creation scenario
- [ ] **5.4.4** Test failed network configuration scenario
- [ ] **5.4.5** Test rollback operation visibility
- [ ] **5.4.6** Verify job persistence in GUI
- [ ] **5.4.7** Test error detail modal display
- [ ] **5.4.8** Verify actionable steps are helpful

**Testing Criteria**:
- [ ] Users can see what failed
- [ ] Users know what to do next
- [ ] Failed jobs remain visible
- [ ] UX is consistent with replications

---

### **Phase 6: Documentation & Deployment** 📚 **REQUIRED**

#### **Task 6.1: Update Documentation**

**Subtasks**:
- [ ] **6.1.1** Update PROJECT_STATUS.md with visibility enhancements
- [ ] **6.1.2** Document error sanitization rules
- [ ] **6.1.3** Document step name mappings
- [ ] **6.1.4** Create operator troubleshooting guide
- [ ] **6.1.5** Update API documentation (Swagger)
- [ ] **6.1.6** Document JSON summary structure

**Testing Criteria**:
- [ ] All docs updated
- [ ] Examples tested
- [ ] Clear for operators

---

#### **Task 6.2: Build & Deploy**

**Subtasks**:
- [ ] **6.2.1** Run database migration
- [ ] **6.2.2** Build OMA API binary
- [ ] **6.2.3** Deploy to dev environment
- [ ] **6.2.4** Monitor for 24 hours
- [ ] **6.2.5** Deploy to QC environment
- [ ] **6.2.6** Deploy to production
- [ ] **6.2.7** Update deployment package

**Testing Criteria**:
- [ ] Migration successful
- [ ] No service disruption
- [ ] Zero regressions
- [ ] Performance stable

---

## 🎨 **QUICK WIN: Phase 1 Can Be Done Immediately**

The error sanitization (Phase 1) requires **ZERO database changes**:

```bash
# Just add two new files:
1. source/current/oma/failover/error_sanitizer.go
2. source/current/oma/failover/step_display_names.go

# Then use them in existing code:
3. Update unified_failover_engine.go to sanitize errors
4. Update enhanced_cleanup_service.go to sanitize errors
5. Update GetFailoverJobStatus to use friendly names

# Deploy and immediately see user-friendly errors!
```

This gives immediate value while we work on persistent summaries (Phase 2-3).

---

### **GUI Components Needed** 🎨

*Note: Listed for completeness, but backend work comes first*

#### **Task 5.1: Unified Job List Component**

**Create**: Component that shows ALL operations in one list:

```typescript
// Pseudo-structure
interface UnifiedJob {
  jobId: string;
  jobType: 'replication' | 'test_failover' | 'live_failover' | 'rollback';
  displayName: string;
  status: string;
  progress: number;
  startedAt: Date;
  currentStep?: string;
  errorMessage?: string;
  errorCategory?: string;
  actionableSteps?: string[];
}

// Shows:
// [Icon] Incremental Replication - 100% ✅ (2 hours ago)
// [Icon] Test Failover - Failed at 70% ❌ (1 hour ago)
//        → Driver injection failed - compatibility tool error
//        → Suggested: Try live failover instead
// [Icon] Incremental Replication - 26% ⏳ (Running)
```

**Tasks**:
- [ ] Create UnifiedJobList component
- [ ] Add job type icons
- [ ] Show sanitized errors
- [ ] Display actionable steps
- [ ] Sort by recency

---

#### **Task 5.2: Enhanced Error Display Modal**

When user clicks on failed job:

```
╔══════════════════════════════════════════════════╗
║  Test Failover Failed                            ║
╠══════════════════════════════════════════════════╣
║                                                  ║
║  VM: pgtest1                                     ║
║  Started: 2 hours ago                            ║
║  Failed at: 70% (Step 7 of 10)                   ║
║                                                  ║
║  Failed Step: Preparing Drivers for KVM          ║
║                                                  ║
║  Issue:                                          ║
║  Driver injection failed - compatibility tool    ║
║  error. Windows drivers could not be installed.  ║
║                                                  ║
║  Category: Compatibility                         ║
║                                                  ║
║  What You Can Do:                                ║
║  • Verify VM is Windows-based                    ║
║  • Check VM disk is accessible                   ║
║  • Try live failover instead (no drivers needed) ║
║                                                  ║
║  [View Technical Details] [Try Live Failover]    ║
╚══════════════════════════════════════════════════╝
```

**Tasks**:
- [ ] Create error detail modal
- [ ] Show sanitized messages
- [ ] Include actionable steps
- [ ] Add "View Technical Details" (admin only)
- [ ] Quick action buttons

---

### **Phase 6: Error Message Sanitization Rules** 📝 **REFERENCE**

**Mapping Table for Sanitization**:

| Internal Term | User-Friendly Term | Example Message |
|--------------|-------------------|-----------------|
| `virt-v2v` | "Driver injection tool" | "Driver injection failed - compatibility tool error" |
| `VirtIO` | "KVM drivers" | "Required drivers could not be installed" |
| `virtio-win.iso` | "Driver package" | "Driver package not accessible" |
| `/dev/vdX` | "Storage volume" | "Storage volume access error" |
| `snapshot creation failed` | "Backup creation failed" | "Could not create backup snapshot" |
| `attachment failed` | "Storage attachment failed" | "Could not attach storage to VM" |
| `network not found` | "Network configuration error" | "Specified network not available" |
| `CloudStack API error` | "Platform error" | "Platform operation failed" |
| `VMA unreachable` | "Source connection lost" | "Cannot connect to source environment" |

**Error Categories**:
- `compatibility` - VM not compatible with operation
- `network` - Network configuration issues
- `storage` - Volume/disk issues
- `platform` - CloudStack/OSSEA errors
- `connectivity` - VMA/network connectivity
- `configuration` - Missing or invalid configuration

**Tasks**:
- [ ] Document all sanitization rules
- [ ] Create comprehensive mapping table
- [ ] Test with real error scenarios
- [ ] Update as new errors are discovered

---

## 🧪 **TESTING SCENARIOS**

### **Scenario 1: Failed Test Failover - Driver Injection**

**Setup**:
1. Start test failover on Windows VM
2. Simulate driver injection failure

**Expected GUI Display**:
```
Recent Jobs:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔴 Test Failover - Failed at 70%
   VM: pgtest1
   Step: Preparing Drivers for KVM
   Error: Driver injection failed - compatibility tool error
   Suggested: Try live failover instead
   2 hours ago
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Incremental Replication - Completed
   26.4 GB transferred
   3 hours ago
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Verify**:
- [ ] Job visible in recent jobs
- [ ] No technical jargon
- [ ] Clear actionable steps
- [ ] Persists across page refreshes

---

### **Scenario 2: Successful Test Failover**

**Expected Summary**:
```json
{
  "last_operation": {
    "operation_type": "test_failover",
    "status": "completed",
    "progress": 100.0,
    "steps_completed": 10,
    "steps_total": 10,
    "duration_seconds": 180
  }
}
```

**Verify**:
- [ ] Summary stored in vm_replication_contexts
- [ ] Shows in unified job list
- [ ] Success indicator clear
- [ ] Duration displayed

---

### **Scenario 3: Rollback Operation**

**Expected Display**:
```
🔄 Rollback - Completed
   Test VM removed successfully
   Snapshots cleaned up
   1 hour ago
```

**Verify**:
- [ ] Rollback jobs visible
- [ ] Clear what happened
- [ ] No technical details exposed

---

## 📝 **SANITIZATION EXAMPLES**

### **Before (Technical)**
```
Error: virt-v2v-in-place failed with exit code 1
Details: VirtIO driver injection failed at /dev/vdc
Log: /var/log/migratekit/virtv2v-virtio-12345.log
```

### **After (User-Friendly)**
```
Error: Driver injection failed - compatibility tool error
Reason: Windows drivers could not be installed on VM
Actions:
  • Verify VM is Windows-based
  • Ensure VM disk is accessible
  • Try live failover (no driver modification required)
```

---

### **Before (Technical)**
```
Error: CloudStack API returned 431 - volume is attached
Stack: volume_operations.go:223 AttachVolume()
```

### **After (User-Friendly)**
```
Error: Storage attachment failed
Reason: Volume is already in use
Actions:
  • Wait for previous operation to complete
  • Check if test VM is still running
  • Contact administrator if issue persists
```

---

## 🎯 **IMPLEMENTATION CHECKLIST**

### **Phase 1: Error Sanitization**
- [ ] Create error_sanitizer.go
- [ ] Map all technical terms
- [ ] Create user message templates
- [ ] Add actionable steps database
- [ ] Test with real failures

### **Phase 2: Job Summaries**
- [ ] Add last_operation_summary column (or reuse existing)
- [ ] Implement storeOperationSummary()
- [ ] Call on all completion paths
- [ ] Test JSON storage

### **Phase 3: Unified API**
- [ ] Create recent jobs endpoint
- [ ] Combine replication + failover jobs
- [ ] Sort and filter properly
- [ ] Test performance

### **Phase 4: GUI Integration**
- [ ] Create unified job list component
- [ ] Show sanitized errors
- [ ] Add error detail modal
- [ ] Test UX flow

### **Phase 5: Validation**
- [ ] Test all failure scenarios
- [ ] Verify no technical leaks
- [ ] Confirm actionable steps help users
- [ ] Validate performance

---

## 🎨 **USER EXPERIENCE GOALS**

### **Before**
```
User: "My test failover failed"
Problem: 
  - Can't see what failed
  - Job disappeared from view
  - No idea what to do next
  - Have to ask admin to check logs
```

### **After**
```
User: "My test failover failed"
Solution:
  - Sees: "Failed at step 7/10: Preparing Drivers for KVM"
  - Reads: "Driver injection failed - compatibility tool error"
  - Actions: "Try live failover instead (no driver modification)"
  - Result: User knows exactly what to do!
```

---

## 🚨 **CRITICAL REQUIREMENTS**

### **Security/Information Hiding**
- ❌ **NEVER show**: virt-v2v, virtio-win.iso, device paths, script names
- ✅ **ALWAYS show**: User-friendly operation names, clear categories, next steps

### **Consistency**
- Same visual style for all job types
- Consistent error messaging patterns
- Unified job list across replication + failover

### **Persistence**
- Failed jobs stay visible
- Can review past operations
- Error details accessible for troubleshooting

---

## 📊 **SUCCESS METRICS**

- [ ] **User Comprehension**: Non-technical users understand what failed
- [ ] **Actionability**: 80%+ of errors have clear next steps
- [ ] **Visibility**: Failed jobs visible for 7+ days
- [ ] **Unification**: Replication and failover jobs in same list
- [ ] **No Technical Leaks**: Zero internal implementation details in GUI

---

## 🔄 **IMPLEMENTATION PRIORITY**

**Immediate** (Do First):
1. Error message sanitization (Phase 1)
2. Store operation summaries (Phase 2)
3. Enhanced status endpoint (Phase 4)

**Next** (After Core Works):
4. Unified job list API (Phase 3)
5. GUI integration (Phase 5)

**Optional Enhancements**:
- Historical operation analytics
- Failure trend analysis
- Automated troubleshooting suggestions

---

---

## 📈 **TASK SUMMARY BY PHASE**

### **Phase 1: Error Sanitization** (6 tasks, 16 subtasks)
- Task 1.1: Error Sanitizer Module - 10 subtasks
- Task 1.2: Step Name Mapper - 6 subtasks

### **Phase 2: Job Summaries** (3 tasks, 22 subtasks)
- Task 2.1: Schema Enhancement - 7 subtasks
- Task 2.2: Store Summaries - 8 subtasks  
- Task 2.3: VM Context API - 7 subtasks

### **Phase 3: Unified API** (2 tasks, 18 subtasks)
- Task 3.1: Recent Jobs Endpoint - 10 subtasks
- Task 3.2: Integration Calls - 8 subtasks

### **Phase 4: Enhanced Responses** (2 tasks, 15 subtasks)
- Task 4.1: Status Response Enhancement - 9 subtasks
- Task 4.2: Step Details - 6 subtasks

### **Phase 5: Testing** (4 tasks, 32 subtasks)
- Task 5.1: Error Sanitization Tests - 8 subtasks
- Task 5.2: Persistence Tests - 8 subtasks
- Task 5.3: API Tests - 8 subtasks
- Task 5.4: UX Tests - 8 subtasks

### **Phase 6: Deployment** (2 tasks, 13 subtasks)
- Task 6.1: Documentation - 6 subtasks
- Task 6.2: Build & Deploy - 7 subtasks

**Total**: 19 tasks, 116 subtasks

---

## ✅ **COMPLETION CHECKLIST**

### **Phase 1 Complete When:**
- [ ] error_sanitizer.go created and tested
- [ ] step_display_names.go created and tested
- [ ] All technical terms mapped to user-friendly versions
- [ ] All actionable steps database complete
- [ ] Zero technical leaks in any error message

### **Phase 2 Complete When:**
- [ ] Database migration deployed
- [ ] Operation summaries storing correctly
- [ ] Summaries persist across restarts
- [ ] VM context API returns summaries
- [ ] Performance impact < 5ms per query

### **Phase 3 Complete When:**
- [ ] Unified recent jobs API working
- [ ] Returns all job types
- [ ] Sorting and limiting correct
- [ ] Sanitization applied throughout
- [ ] Response time < 100ms

### **Phase 4 Complete When:**
- [ ] Status responses enhanced
- [ ] Friendly step names included
- [ ] Sanitized errors in responses
- [ ] Step details available
- [ ] GUI can consume all fields

### **Phase 5 Complete When:**
- [ ] All test scenarios pass
- [ ] No regressions
- [ ] UX validated by users
- [ ] Performance metrics met
- [ ] Edge cases handled

### **Phase 6 Complete When:**
- [ ] All documentation updated
- [ ] Deployed to all environments
- [ ] Monitoring shows stable
- [ ] User feedback positive

---

## 🎯 **SUCCESS CRITERIA**

**User Experience**:
- [ ] Users understand what failed without asking
- [ ] Users know what action to take next
- [ ] Failed jobs visible for 7+ days
- [ ] Consistent UX with replication jobs

**Technical**:
- [ ] Zero technical implementation details exposed
- [ ] All errors sanitized
- [ ] Performance impact < 5%
- [ ] No breaking changes

**Operational**:
- [ ] Reduced support tickets for "what failed?"
- [ ] Operators can troubleshoot from GUI
- [ ] Clear audit trail of all operations

---

## 📊 **IMPLEMENTATION ORDER**

**Week 1**:
- Days 1-2: Phase 1 (Error Sanitization) - Immediate value
- Days 3-4: Phase 2 (Job Summaries) - Persistence
- Day 5: Phase 3 (Unified API) - Complete integration

**Week 2**:
- Days 1-2: Phase 4 (Enhanced Responses) - Polish
- Days 3-4: Phase 5 (Testing) - Validation
- Day 5: Phase 6 (Deployment) - Production

---

## 🚀 **QUICK START GUIDE**

To begin implementation:

1. **Start with Task 1.1.1**: Create `error_sanitizer.go` file
2. **Reference this job sheet**: Check off subtasks as you complete them
3. **Test incrementally**: Don't wait until everything is done
4. **Deploy Phase 1 separately**: Get user-friendly errors live ASAP
5. **Then do Phase 2-3**: Add persistence and unified view

---

**Status**: Ready for Implementation  
**Dependencies**: None (all internal changes)  
**Risk Level**: Low - Additive changes only  
**Testing Time**: 4-6 hours  
**Total Subtasks**: 116  
**Current Progress**: 0/116 (0%)

---

**Last Updated**: October 3, 2025  
**Next Review**: After Phase 1 completion

