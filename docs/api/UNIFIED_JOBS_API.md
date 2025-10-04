# Unified Jobs API - Backend Implementation Complete

**Version**: v2.32.0  
**Created**: October 3, 2025  
**Status**: ✅ **BACKEND COMPLETE - Ready for GUI Integration**  

---

## 📋 **OVERVIEW**

The Unified Jobs API provides a consistent interface for tracking ALL VM operations (replication, failover, rollback) with sanitized error messages and actionable user guidance. This eliminates the previous disconnect where failover/rollback jobs had poor visibility compared to replication jobs.

**Key Features**:
- ✅ Combines replication_jobs + job_tracking (JobLog) in single API
- ✅ Sanitizes all technical error messages
- ✅ Provides user-friendly step names
- ✅ Includes actionable steps for every failure
- ✅ Persistent visibility (failed jobs don't disappear)

---

## 🔌 **API ENDPOINTS**

### **1. Get Unified Recent Jobs**

**Endpoint**: `GET /api/v1/vm-contexts/{context_id}/recent-jobs`

**Purpose**: Retrieve ALL recent operations for a VM (replication + failover + rollback) in one unified list

**Parameters**:
- `context_id` (path) - VM context identifier (e.g., "ctx-pgtest1-20251003-140708")

**Response**:
```json
{
  "context_id": "ctx-pgtest1-20251003-140708",
  "count": 5,
  "jobs": [
    {
      "job_id": "e5de9b1b-5159-49e2-95e5-be644da2b7fb",
      "external_job_id": "unified-test-failover-pgtest1-1759510017",
      "job_type": "test_failover",
      "status": "failed",
      "progress": 60.0,
      "started_at": "2025-10-03T17:46:57+01:00",
      "completed_at": "2025-10-03T17:47:22+01:00",
      "display_name": "Test Failover",
      "current_step": "Preparing Drivers for Compatibility",
      "error_message": "KVM driver installation failed - compatibility issue",
      "error_category": "compatibility",
      "actionable_steps": [
        "Try live failover (no driver modification)",
        "Verify VM is Windows-based"
      ],
      "data_source": "job_tracking",
      "duration_seconds": 25
    },
    {
      "job_id": "job-20251003-140728",
      "job_type": "replication",
      "status": "completed",
      "progress": 100.0,
      "started_at": "2025-10-03T14:07:28Z",
      "completed_at": "2025-10-03T14:40:48Z",
      "display_name": "Replication Completed",
      "current_step": "Completed",
      "data_source": "replication_jobs",
      "duration_seconds": 1999
    }
  ]
}
```

**Job Types**:
- `replication` - Replication jobs
- `test_failover` - Test failover operations
- `live_failover` - Live failover operations
- `rollback` - Rollback/cleanup operations

**Sort Order**: Most recent first (by started_at timestamp)  
**Limit**: 20 most recent jobs  

---

### **2. Get VM Context (Enhanced)**

**Endpoint**: `GET /api/v1/vm-contexts/{vm_name}`

**Enhancement**: Now includes `last_operation` field with sanitized operation summary

**Response Addition**:
```json
{
  "context": { ... },
  "current_job": { ... },
  "job_history": [ ... ],
  "last_operation": {
    "job_id": "e5de9b1b-5159-49e2-95e5-be644da2b7fb",
    "external_job_id": "unified-test-failover-pgtest1-1759510017",
    "operation_type": "test_failover",
    "status": "failed",
    "progress": 60.0,
    "failed_step": "Preparing Drivers for Compatibility",
    "error_message": "KVM driver installation failed - compatibility issue",
    "error_category": "compatibility",
    "error_severity": "error",
    "actionable_steps": [
      "Try live failover (no driver modification)",
      "Verify VM is Windows-based"
    ],
    "timestamp": "2025-10-03T17:47:22Z",
    "duration_seconds": 25,
    "steps_completed": 3,
    "steps_total": 5
  }
}
```

**Null Handling**: `last_operation` is omitted if no operation summary exists (backward compatible)

---

## 🎨 **ERROR SANITIZATION RULES**

### **Technical Terms → User-Friendly**

| Technical Term | User-Friendly Term |
|---------------|-------------------|
| `virt-v2v` | "Driver injection tool" |
| `virt-v2v-in-place` | "Driver preparation" |
| `VirtIO` | "KVM drivers" |
| `virtio-win.iso` | "Driver package" |
| `/dev/vdX` | "Storage volume" |
| `virtio-driver-injection` | "Preparing Drivers for Compatibility" |
| `cloudstack-snapshot-creation` | "Creating Backup Snapshots" |
| `vm-creation` | "Creating Destination VM" |
| `volume-attachment` | "Attaching Storage Volumes" |
| `network-configuration` | "Configuring Network Adapters" |

### **Step Name Mapping**

**All internal step names have user-friendly equivalents**:
- `validation` → "Pre-flight Validation"
- `source-vm-power-off` → "Powering Off Source VM"
- `final-sync` → "Final Data Synchronization"
- `multi-volume-snapshot-creation` → "Creating Backup Snapshots"
- `virtio-driver-injection` → "Preparing Drivers for Compatibility"
- `vm-startup-and-validation` → "Starting and Validating VM"
- `test-vm-deletion` → "Removing Test VM"
- `cloudstack-snapshot-rollback` → "Rolling Back to Backup"

**See**: `source/current/oma/failover/step_display_names.go` for complete list

---

## 📂 **ERROR CATEGORIES**

All errors are categorized for consistent handling:

- **`compatibility`** - VM not compatible with operation
  - Example: "Driver injection failed - VM may not be compatible"
  - Actions: "Try live failover instead"

- **`network`** - Network configuration issues
  - Example: "Network not found - target network is not available"
  - Actions: "Configure network mapping"

- **`storage`** - Volume/disk issues
  - Example: "Storage volume is already in use"
  - Actions: "Wait for previous operation to complete"

- **`platform`** - CloudStack/OSSEA errors
  - Example: "Insufficient resources on destination platform"
  - Actions: "Check available resources"

- **`connectivity`** - VMA/network connectivity
  - Example: "Cannot connect to source environment"
  - Actions: "Verify source environment is online"

- **`configuration`** - Missing or invalid configuration
  - Example: "Pre-flight validation failed"
  - Actions: "Review VM configuration and status"

---

## 🔧 **BACKEND IMPLEMENTATION**

### **Error Sanitization Module**

**File**: `source/current/oma/failover/error_sanitizer.go`

**Main Function**: `SanitizeFailoverError(stepName string, err error) SanitizedMessage`

**Returns**:
```go
type SanitizedMessage struct {
    UserMessage     string   // Clean message for GUI
    TechnicalDetail string   // Full details (logs only)
    Category        string   // Error category
    ActionableSteps []string // What user can do
    Severity        string   // info, warning, error, critical
}
```

**Sanitizers Available**:
- `sanitizeDriverInjectionError()` - VirtIO/virt-v2v errors
- `sanitizeVMCreationError()` - VM creation failures
- `sanitizeNetworkError()` - Network configuration
- `sanitizeVolumeError()` - Storage operations
- `sanitizeVMPowerError()` - Power management
- `sanitizeSnapshotError()` - Snapshot operations
- `sanitizeValidationError()` - Pre-flight validation

---

### **Step Name Mapping**

**File**: `source/current/oma/failover/step_display_names.go`

**Functions**:
- `GetUserFriendlyStepName(internalStep string) string`
- `GetStepCategory(internalStep string) string` - Returns: setup, preparation, execution, cleanup, finalization
- `GetStepDescription(internalStep string) string` - Detailed description
- `GetStepIcon(internalStep, status string) string` - Emoji for visual display
- `FormatStepForDisplay(...)` - Complete formatting

---

### **Operation Summary Storage**

**Database**:
```sql
-- Column added to vm_replication_contexts
last_operation_summary JSON NULL
```

**Stored Automatically**:
- On failover completion (success or failure)
- On rollback completion (success or failure)
- Includes sanitized errors and actionable steps
- Persists indefinitely for GUI visibility

**JSON Structure**:
```json
{
  "job_id": "uuid",
  "external_job_id": "unified-test-failover-vm-timestamp",
  "operation_type": "test_failover|live_failover|rollback",
  "status": "completed|failed",
  "progress": 0-100,
  "failed_step": "User-friendly step name",
  "error_message": "Sanitized error message",
  "error_category": "compatibility|network|storage|platform|connectivity|configuration",
  "actionable_steps": ["Action 1", "Action 2"],
  "timestamp": "ISO8601",
  "duration_seconds": 123,
  "steps_completed": 3,
  "steps_total": 10
}
```

---

## 🧪 **TESTING EXAMPLES**

### **Test 1: Get Recent Jobs for pgtest1**

```bash
curl -s "http://localhost:8082/api/v1/vm-contexts/ctx-pgtest1-20251003-140708/recent-jobs" | jq .
```

**Expected**: List of all operations (replication + failover) with sanitized errors

### **Test 2: Get VM Context with Last Operation**

```bash
curl -s "http://localhost:8082/api/v1/vm-contexts/pgtest1" | jq ".last_operation"
```

**Expected**: Sanitized summary of most recent operation

### **Test 3: Verify Error Sanitization**

**Trigger a failed test failover**, then:
```sql
SELECT JSON_PRETTY(last_operation_summary) 
FROM vm_replication_contexts 
WHERE vm_name = 'pgtest1';
```

**Expected**:
- No "virt-v2v" references
- No "VirtIO" references
- User-friendly step names
- Clear actionable steps

---

## 📊 **DATA SOURCES**

### **Replication Jobs**
**Table**: `replication_jobs`  
**Fields**: id, status, progress_percent, error_message, created_at, completed_at  
**Filter**: WHERE vm_context_id = ?  

### **Failover/Rollback Jobs**
**Table**: `job_tracking` (via JobLog)  
**Fields**: id, external_job_id, status, started_at, completed_at  
**Related**: `job_steps` (for step-by-step progress)  
**Filter**: WHERE context_id = ? AND job_type IN ('failover', 'cleanup')  

### **Merging Logic**
1. Query both sources independently
2. Convert to unified `UnifiedJobItem` structure
3. Apply sanitization to all error messages
4. Sort by started_at (newest first)
5. Limit to 20 most recent

---

## 🎯 **USAGE GUIDELINES**

### **For Frontend Developers**

**Use Case 1: Show Recent Activity**
```typescript
// Fetch all recent operations for a VM
const response = await fetch(`/api/v1/vm-contexts/${contextId}/recent-jobs`);
const data = await response.json();

// Display each job with sanitized info
data.jobs.forEach(job => {
  console.log(`${job.display_name}: ${job.status}`);
  if (job.error_message) {
    console.log(`Error: ${job.error_message}`);
    console.log(`Actions: ${job.actionable_steps.join(', ')}`);
  }
});
```

**Use Case 2: Show Last Operation Status**
```typescript
// Get VM context with last operation
const vm = await fetch(`/api/v1/vm-contexts/${vmName}`).then(r => r.json());

if (vm.last_operation && vm.last_operation.status === 'failed') {
  // Show persistent failure notification
  showError({
    message: vm.last_operation.error_message,
    actions: vm.last_operation.actionable_steps
  });
}
```

**Use Case 3: Unified Job History Component**
```typescript
// Single component shows ALL job types
<UnifiedJobList contextId={contextId}>
  {jobs.map(job => (
    <JobItem
      key={job.job_id}
      type={job.job_type}
      displayName={job.display_name}
      status={job.status}
      progress={job.progress}
      error={job.error_message}
      actions={job.actionable_steps}
    />
  ))}
</UnifiedJobList>
```

---

## ⚠️ **IMPORTANT SECURITY NOTES**

### **What's Hidden from Users**

The sanitization layer ensures these NEVER appear in GUI:
- ❌ `virt-v2v` tool references
- ❌ `VirtIO` driver internal names
- ❌ Device paths (`/dev/vdX`)
- ❌ Script paths or names
- ❌ Internal implementation details

### **What's Available for Admins**

Technical details are still logged and stored:
- Full error messages in `job_tracking.error_message`
- Full step details in `job_steps.error_message`
- Complete logs in `log_events` table
- Technical details available via admin-only endpoints (if implemented)

**GUI should NOT display technical_detail field** - it's for admin/debugging only.

---

## 📊 **RESPONSE FIELD REFERENCE**

### **UnifiedJobItem Fields**

| Field | Type | Description | Always Present |
|-------|------|-------------|----------------|
| `job_id` | string | Internal job identifier | Yes |
| `external_job_id` | string | GUI-constructed job ID | For failover/rollback |
| `job_type` | string | Type: replication, test_failover, live_failover, rollback | Yes |
| `status` | string | Job status: running, completed, failed, cancelled | Yes |
| `progress` | number | Progress percentage (0-100) | Yes |
| `started_at` | timestamp | When job started | Yes |
| `completed_at` | timestamp | When job finished | If completed |
| `display_name` | string | User-friendly operation name | Yes |
| `current_step` | string | Current/last step (sanitized) | If available |
| `error_message` | string | **SANITIZED** error message | If failed |
| `error_category` | string | Error category | If failed |
| `actionable_steps` | array | What user can do | If failed |
| `data_source` | string | replication_jobs or job_tracking | Yes |
| `duration_seconds` | number | Total duration | If completed |

### **OperationSummary Fields** (last_operation)

Same as UnifiedJobItem, plus:
- `failed_step` - Which step failed (user-friendly)
- `failed_step_internal` - Internal step name (for debugging)
- `error_severity` - info, warning, error, critical
- `steps_completed` - How many steps completed
- `steps_total` - Total steps in operation
- `timestamp` - When summary was created

---

## 🎨 **GUI INTEGRATION EXAMPLES**

### **Example 1: Unified Job List**

```jsx
function UnifiedJobList({ contextId }) {
  const [jobs, setJobs] = useState([]);
  
  useEffect(() => {
    fetch(`/api/v1/vm-contexts/${contextId}/recent-jobs`)
      .then(r => r.json())
      .then(data => setJobs(data.jobs));
  }, [contextId]);
  
  return (
    <div className="job-list">
      <h3>Recent Operations</h3>
      {jobs.map(job => (
        <JobCard key={job.job_id} job={job} />
      ))}
    </div>
  );
}

function JobCard({ job }) {
  const getStatusIcon = (status) => {
    switch(status) {
      case 'completed': return '✅';
      case 'failed': return '❌';
      case 'running': return '⏳';
      default: return '📋';
    }
  };
  
  return (
    <div className={`job-card job-${job.status}`}>
      <div className="job-header">
        {getStatusIcon(job.status)} {job.display_name}
      </div>
      
      {job.status === 'running' && (
        <div className="progress-bar">
          <div style={{width: `${job.progress}%`}}>
            {job.progress.toFixed(0)}%
          </div>
        </div>
      )}
      
      {job.status === 'failed' && (
        <div className="error-info">
          <div className="error-message">{job.error_message}</div>
          <div className="actionable-steps">
            <strong>What you can do:</strong>
            <ul>
              {job.actionable_steps.map((step, i) => (
                <li key={i}>{step}</li>
              ))}
            </ul>
          </div>
        </div>
      )}
      
      <div className="job-meta">
        {new Date(job.started_at).toLocaleString()}
        {job.duration_seconds && ` • ${formatDuration(job.duration_seconds)}`}
      </div>
    </div>
  );
}
```

---

### **Example 2: Persistent Error Display**

```jsx
function VMStatusBanner({ vmContext }) {
  const lastOp = vmContext.last_operation;
  
  if (!lastOp || lastOp.status === 'completed') {
    return null; // No persistent error to show
  }
  
  if (lastOp.status === 'failed') {
    return (
      <Alert severity="error">
        <AlertTitle>
          {lastOp.operation_type.replace('_', ' ')} Failed
        </AlertTitle>
        <p><strong>Issue:</strong> {lastOp.error_message}</p>
        <p><strong>Failed at:</strong> {lastOp.failed_step} 
           ({lastOp.progress}%)</p>
        
        <div className="actions">
          <strong>What you can do:</strong>
          <ul>
            {lastOp.actionable_steps.map((step, i) => (
              <li key={i}>{step}</li>
            ))}
          </ul>
        </div>
        
        <Button onClick={() => dismissError()}>Dismiss</Button>
        <Button onClick={() => viewDetails(lastOp)}>View Details</Button>
      </Alert>
    );
  }
  
  return null;
}
```

---

## 🔒 **SECURITY CONSIDERATIONS**

### **What to NEVER Show in GUI**

1. ❌ Technical error stack traces
2. ❌ File paths or script names
3. ❌ Internal tool names (virt-v2v, libguestfs, etc.)
4. ❌ Device paths (/dev/vdX)
5. ❌ Database queries or SQL errors
6. ❌ API endpoint paths or internal service names

### **What to ALWAYS Show**

1. ✅ Sanitized error messages
2. ✅ User-friendly step names
3. ✅ Actionable guidance
4. ✅ Error categories
5. ✅ Operation progress
6. ✅ Time information

---

## 📊 **BACKEND COMPONENTS**

### **Files Modified**:
1. `source/current/oma/failover/error_sanitizer.go` - Error sanitization logic
2. `source/current/oma/failover/step_display_names.go` - Step name mapping
3. `source/current/oma/failover/unified_failover_engine.go` - Stores summaries
4. `source/current/oma/failover/enhanced_cleanup_service.go` - Rollback summaries
5. `source/current/oma/api/handlers/vm_contexts.go` - Unified jobs API
6. `source/current/oma/database/models.go` - Added LastOperationSummary field
7. `source/current/oma/database/repository.go` - Enhanced VMContextDetails

### **Database Migration**:
`source/current/oma/database/migrations/20251003160000_add_operation_summary.up.sql`

---

## 🎯 **NEXT STEPS FOR GUI**

**The backend is 100% complete.** GUI needs to:

1. **Use Unified Jobs API** - Call `/api/v1/vm-contexts/{context_id}/recent-jobs`
2. **Display Sanitized Errors** - Show error_message and actionable_steps
3. **Show Persistent Failures** - Display last_operation from VM context
4. **Unify Job Display** - Show replication + failover in same list
5. **Remove Technical Terms** - Don't show any unsanitized data

**See**: Handoff prompt for GUI developer in next section

---

## 📝 **VERSION HISTORY**

- **v2.31.0** - Error sanitization + operation summary storage
- **v2.32.0** - Unified recent jobs API endpoint
- **v2.30.1** - Job recovery with VMA validation (prerequisite)

---

**Backend Status**: ✅ **COMPLETE**  
**Deployed Servers**: 10.245.246.147, 10.245.246.148  
**Ready for GUI Integration**: YES  
**Documentation**: Complete


