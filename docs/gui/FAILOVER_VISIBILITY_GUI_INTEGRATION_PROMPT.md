# GUI Integration Prompt - Failover Visibility Enhancement

**For**: Next Claude 4.5 Session  
**Project**: MigrateKit OSSEA - Failover Visibility Enhancement  
**Backend Status**: ✅ **COMPLETE** (v2.32.0)  
**Your Task**: Integrate unified jobs API into GUI for seamless UX  

---

## 🎯 **YOUR MISSION**

Integrate the newly created Unified Jobs API into the MigrateKit OSSEA GUI to provide seamless visibility for failover and rollback operations, matching the UX quality of replication jobs.

**Problem Solved by Backend**:
- ✅ Failover/rollback errors are now sanitized (no technical jargon)
- ✅ Operation summaries persist in database (don't disappear)
- ✅ Unified API combines replication + failover + rollback jobs
- ✅ Every error has actionable steps for users

**Your Task**:
- Integrate these APIs into the GUI
- Display failover/rollback jobs alongside replication jobs
- Show sanitized errors with actionable guidance
- Make failed operations persistently visible

---

## 📚 **ESSENTIAL CONTEXT**

### **Project**: MigrateKit OSSEA
VMware → OSSEA (CloudStack) migration platform with GUI dashboard.

### **Current Issue**:
Failover and rollback operations have poor GUI visibility:
- Jobs disappear quickly after completion/failure
- Error messages expose technical implementation ("virt-v2v", "VirtIO")
- No persistent failure tracking
- Different UI location than replication jobs (disjointed UX)

### **Backend Solution** (Already Complete):
1. **Error Sanitization**: Converts all technical errors to user-friendly messages
2. **Operation Summaries**: Stores sanitized summaries in database (persistent)
3. **Unified Jobs API**: Single endpoint for all operation types
4. **Actionable Guidance**: Every error includes what user can do

---

## 🔌 **NEW API ENDPOINTS AVAILABLE**

### **Endpoint 1: Unified Recent Jobs**

```
GET /api/v1/vm-contexts/{context_id}/recent-jobs
```

**Returns**: All recent operations (replication + failover + rollback) in one list

**Example Response**:
```json
{
  "context_id": "ctx-pgtest1-20251003-140708",
  "count": 3,
  "jobs": [
    {
      "job_id": "uuid-123",
      "external_job_id": "unified-test-failover-pgtest1-1759510017",
      "job_type": "test_failover",
      "status": "failed",
      "progress": 60.0,
      "started_at": "2025-10-03T17:46:57Z",
      "completed_at": "2025-10-03T17:47:22Z",
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
      "display_name": "Replication Completed",
      "data_source": "replication_jobs"
    }
  ]
}
```

**Job Types**:
- `replication` - Incremental/initial replication
- `test_failover` - Test failover operations
- `live_failover` - Live failover operations
- `rollback` - Rollback/cleanup operations

---

### **Endpoint 2: VM Context with Last Operation**

```
GET /api/v1/vm-contexts/{vm_name}
```

**Enhancement**: Now includes `last_operation` field with persistent failure info

**New Field in Response**:
```json
{
  "context": { ... },
  "last_operation": {
    "operation_type": "test_failover",
    "status": "failed",
    "progress": 60.0,
    "failed_step": "Preparing Drivers for Compatibility",
    "error_message": "KVM driver installation failed - compatibility issue",
    "error_category": "compatibility",
    "actionable_steps": [
      "Try live failover (no driver modification)",
      "Verify VM is Windows-based"
    ],
    "timestamp": "2025-10-03T17:47:22Z",
    "steps_completed": 3,
    "steps_total": 5
  }
}
```

**Null Handling**: `last_operation` is omitted if no summary exists (backward compatible)

---

## 🎨 **GUI REQUIREMENTS**

### **Requirement 1: Unified Job List Component**

**Location**: Integrate into VM details view (wherever replication jobs are shown)

**Display**:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Recent Operations

❌ Test Failover - Failed (60%)
   Issue: KVM driver installation failed - compatibility issue
   
   What you can do:
   • Try live failover (no driver modification)
   • Verify VM is Windows-based
   
   25 seconds ago

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ Replication Completed
   26.4 GB transferred
   3 hours ago

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Features**:
- Shows ALL job types in one list (no separation)
- Sorted by time (newest first)
- Failed jobs show sanitized errors
- Actionable steps displayed prominently
- Consistent visual style

---

### **Requirement 2: Persistent Error Banner**

**Location**: Top of VM details view (like a sticky notification)

**When to Show**: When `last_operation.status === 'failed'`

**Display**:
```
╔══════════════════════════════════════════════════╗
║  ⚠️  Last Operation Failed                       ║
╠══════════════════════════════════════════════════╣
║  Test Failover failed at step 3 of 5 (60%)       ║
║                                                  ║
║  Failed Step: Preparing Drivers for Compatibility║
║                                                  ║
║  Issue:                                          ║
║  KVM driver installation failed - compatibility  ║
║  issue.                                          ║
║                                                  ║
║  What you can do:                                ║
║  • Try live failover (no driver modification)    ║
║  • Verify VM is Windows-based                    ║
║                                                  ║
║  [Dismiss] [Try Live Failover] [View Details]    ║
╚══════════════════════════════════════════════════╝
```

**Features**:
- Persists across page refreshes
- User can dismiss (clear from state, not database)
- Quick action buttons based on suggestions
- Clean, non-technical language

---

### **Requirement 3: Enhanced Error Modal**

**When**: User clicks on failed job or "View Details"

**Display**:
```jsx
<Modal>
  <ModalHeader>
    ❌ Test Failover Failed
  </ModalHeader>
  
  <ModalBody>
    <Section>
      <Label>VM:</Label> pgtest1
      <Label>Started:</Label> 2 hours ago
      <Label>Duration:</Label> 25 seconds
      <Label>Progress:</Label> 60% (3 of 5 steps completed)
    </Section>
    
    <Section>
      <Label>Failed Step:</Label>
      <StepName>Preparing Drivers for Compatibility</StepName>
    </Section>
    
    <Section>
      <Label>Issue:</Label>
      <ErrorMessage>
        KVM driver installation failed - compatibility issue
      </ErrorMessage>
    </Section>
    
    <Section>
      <Label>Category:</Label> Compatibility
    </Section>
    
    <Section>
      <Label>What You Can Do:</Label>
      <ActionableSteps>
        <li>Try live failover (no driver modification)</li>
        <li>Verify VM is Windows-based</li>
      </ActionableSteps>
    </Section>
    
    {/* Optional: Admin-only technical details */}
    {isAdmin && (
      <Accordion>
        <AccordionTitle>Technical Details (Admin)</AccordionTitle>
        <AccordionContent>
          {/* Show job.technical_details if needed */}
        </AccordionContent>
      </Accordion>
    )}
  </ModalBody>
  
  <ModalFooter>
    <Button onClick={close}>Close</Button>
    <Button onClick={tryLiveFailover}>Try Live Failover</Button>
  </ModalFooter>
</Modal>
```

---

## 🚨 **CRITICAL REQUIREMENTS**

### **MUST DO**:
1. ✅ Use sanitized error messages ONLY (never show technical details to regular users)
2. ✅ Display actionable steps prominently
3. ✅ Show failover/rollback jobs in same list as replications
4. ✅ Make failed jobs visible until explicitly dismissed
5. ✅ Use consistent visual styling for all job types

### **MUST NOT DO**:
1. ❌ NEVER show "virt-v2v", "VirtIO", or other technical tool names
2. ❌ NEVER show device paths or file paths
3. ❌ NEVER show internal step names (use display_name from API)
4. ❌ NEVER expose technical_details field to non-admin users
5. ❌ NEVER silently fail to display errors (users must see what went wrong)

---

## 📋 **IMPLEMENTATION CHECKLIST**

### **Task 1: Create Unified Job List Component**
- [ ] Create `UnifiedJobList.tsx` component
- [ ] Fetch from `/api/v1/vm-contexts/{context_id}/recent-jobs`
- [ ] Display all job types with consistent styling
- [ ] Show sanitized errors for failed jobs
- [ ] Display actionable steps for failures
- [ ] Sort by timestamp (newest first)
- [ ] Handle loading and error states

### **Task 2: Add Persistent Error Banner**
- [ ] Create `OperationErrorBanner.tsx` component
- [ ] Read `last_operation` from VM context response
- [ ] Show banner when `last_operation.status === 'failed'`
- [ ] Display sanitized error message
- [ ] Display actionable steps
- [ ] Add quick action buttons
- [ ] Persist across page refreshes
- [ ] Allow user to dismiss

### **Task 3: Enhance Error Display Modal**
- [ ] Create/enhance `JobErrorDetailsModal.tsx`
- [ ] Show failed step name (user-friendly)
- [ ] Show error message (sanitized)
- [ ] Show progress at failure point
- [ ] List actionable steps
- [ ] Add quick action buttons (e.g., "Try Live Failover")
- [ ] Optional admin-only technical details section

### **Task 4: Update VM List/Context Views**
- [ ] Replace separate failover/rollback job displays with unified list
- [ ] Move failover jobs to same location as replication jobs
- [ ] Use consistent visual indicators (icons, colors)
- [ ] Show last operation status in VM summary cards

### **Task 5: Testing**
- [ ] Test with failed test failover
- [ ] Test with failed live failover
- [ ] Test with failed rollback
- [ ] Test with successful operations
- [ ] Test with mix of job types
- [ ] Verify no technical terms visible
- [ ] Verify actionable steps are helpful

---

## 🧪 **TESTING DATA**

### **Test Server**: 10.245.246.148

**Available Test Data**:
```bash
# pgtest1 has a failed test failover with sanitized error
curl "http://10.245.246.148:8082/api/v1/vm-contexts/ctx-pgtest1-20251003-140708/recent-jobs"

# Expected: List with failed test failover showing:
# - "KVM driver installation failed - compatibility issue"
# - Actionable: "Try live failover (no driver modification)"
```

### **Expected Sanitization Examples**:

**If you see these in GUI - IT'S WRONG**:
- ❌ "virt-v2v-in-place failed"
- ❌ "VirtIO driver error"
- ❌ "Failed at /dev/vdc"

**You should see**:
- ✅ "Driver installation failed - compatibility issue"
- ✅ "Preparing Drivers for Compatibility"
- ✅ "Storage volume access error"

---

## 📖 **REFERENCE DOCUMENTATION**

**Complete API Documentation**:
`/home/pgrayson/migratekit-cloudstack/docs/api/UNIFIED_JOBS_API.md`

**Backend Implementation Details**:
- Error sanitizer: `source/current/oma/failover/error_sanitizer.go`
- Step names: `source/current/oma/failover/step_display_names.go`
- API handler: `source/current/oma/api/handlers/vm_contexts.go`

**Job Sheet** (Backend - for reference):
`AI_Helper/FAILOVER_VISIBILITY_ENHANCEMENT_JOB_SHEET.md`

---

## 🎨 **DESIGN GUIDELINES**

### **Visual Consistency**

**Job Status Colors**:
- ✅ Completed: Green
- ❌ Failed: Red
- ⏳ Running: Blue/Yellow
- 🔲 Pending: Gray

**Job Type Icons**:
- 🔄 Replication
- 🚀 Test Failover
- ⚡ Live Failover
- ↩️ Rollback

### **Error Display Priority**

1. **Primary**: Sanitized error message
2. **Secondary**: Actionable steps (bullet list)
3. **Tertiary**: Failed step name and progress
4. **Hidden**: Technical details (admin only)

### **Information Hierarchy**

```
Operation Name (Test Failover)          ← Most prominent
   ↓
Status (Failed) + Progress (60%)        ← Clear indication
   ↓
Error Message (sanitized)               ← What went wrong
   ↓
Actionable Steps                        ← What user can do
   ↓
Metadata (time, duration)               ← Context
```

---

## 🛠️ **IMPLEMENTATION GUIDE**

### **Step 1: Create API Client Functions**

```typescript
// api/vmContexts.ts

export async function getRecentJobs(contextId: string) {
  const response = await fetch(
    `/api/v1/vm-contexts/${contextId}/recent-jobs`
  );
  if (!response.ok) throw new Error('Failed to fetch recent jobs');
  return response.json();
}

export async function getVMContext(vmName: string) {
  const response = await fetch(`/api/v1/vm-contexts/${vmName}`);
  if (!response.ok) throw new Error('Failed to fetch VM context');
  return response.json();
}
```

---

### **Step 2: Create Type Definitions**

```typescript
// types/jobs.ts

export type JobType = 'replication' | 'test_failover' | 'live_failover' | 'rollback';

export type JobStatus = 'running' | 'completed' | 'failed' | 'cancelled';

export type ErrorCategory = 
  | 'compatibility' 
  | 'network' 
  | 'storage' 
  | 'platform' 
  | 'connectivity' 
  | 'configuration';

export interface UnifiedJob {
  job_id: string;
  external_job_id?: string;
  job_type: JobType;
  status: JobStatus;
  progress: number;
  started_at: string;
  completed_at?: string;
  display_name: string;
  current_step?: string;
  error_message?: string;
  error_category?: ErrorCategory;
  actionable_steps?: string[];
  data_source: 'replication_jobs' | 'job_tracking';
  duration_seconds?: number;
}

export interface OperationSummary {
  job_id: string;
  external_job_id?: string;
  operation_type: string;
  status: JobStatus;
  progress: number;
  failed_step?: string;
  error_message?: string;
  error_category?: ErrorCategory;
  actionable_steps?: string[];
  timestamp: string;
  duration_seconds: number;
  steps_completed?: number;
  steps_total?: number;
}
```

---

### **Step 3: Create Unified Job List Component**

```typescript
// components/UnifiedJobList.tsx

import React, { useEffect, useState } from 'react';
import { UnifiedJob } from '@/types/jobs';
import { getRecentJobs } from '@/api/vmContexts';

interface Props {
  contextId: string;
}

export function UnifiedJobList({ contextId }: Props) {
  const [jobs, setJobs] = useState<UnifiedJob[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadJobs();
  }, [contextId]);

  const loadJobs = async () => {
    try {
      const data = await getRecentJobs(contextId);
      setJobs(data.jobs);
    } catch (error) {
      console.error('Failed to load recent jobs:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <LoadingSpinner />;

  return (
    <div className="unified-job-list">
      <h3 className="text-lg font-semibold mb-4">Recent Operations</h3>
      
      {jobs.length === 0 && (
        <div className="text-gray-500">No recent operations</div>
      )}
      
      {jobs.map(job => (
        <JobCard key={job.job_id} job={job} />
      ))}
    </div>
  );
}

function JobCard({ job }: { job: UnifiedJob }) {
  const statusIcon = {
    completed: '✅',
    failed: '❌',
    running: '⏳',
    cancelled: '🚫'
  }[job.status] || '📋';

  const statusColor = {
    completed: 'text-green-600 bg-green-50',
    failed: 'text-red-600 bg-red-50',
    running: 'text-blue-600 bg-blue-50',
    cancelled: 'text-gray-600 bg-gray-50'
  }[job.status] || 'text-gray-600 bg-gray-50';

  return (
    <div className={`job-card p-4 mb-3 rounded-lg border ${statusColor}`}>
      {/* Header */}
      <div className="flex justify-between items-start mb-2">
        <div className="flex items-center gap-2">
          <span className="text-2xl">{statusIcon}</span>
          <div>
            <div className="font-semibold">{job.display_name}</div>
            <div className="text-sm text-gray-600">
              {formatTimestamp(job.started_at)}
              {job.duration_seconds && ` • ${formatDuration(job.duration_seconds)}`}
            </div>
          </div>
        </div>
        
        {job.status === 'running' && (
          <div className="text-sm font-medium">{job.progress.toFixed(0)}%</div>
        )}
      </div>
      
      {/* Progress bar for running jobs */}
      {job.status === 'running' && (
        <div className="w-full bg-gray-200 rounded h-2 mb-2">
          <div 
            className="bg-blue-500 h-2 rounded"
            style={{ width: `${job.progress}%` }}
          />
        </div>
      )}
      
      {/* Error information for failed jobs */}
      {job.status === 'failed' && job.error_message && (
        <div className="mt-3 p-3 bg-white rounded border border-red-200">
          <div className="font-medium text-red-700 mb-2">
            {job.error_message}
          </div>
          
          {job.actionable_steps && job.actionable_steps.length > 0 && (
            <div className="mt-2">
              <div className="text-sm font-medium text-gray-700 mb-1">
                What you can do:
              </div>
              <ul className="text-sm text-gray-600 list-disc list-inside space-y-1">
                {job.actionable_steps.map((step, i) => (
                  <li key={i}>{step}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
      
      {/* Current step for running jobs */}
      {job.status === 'running' && job.current_step && (
        <div className="text-sm text-gray-600 mt-2">
          Current: {job.current_step}
        </div>
      )}
    </div>
  );
}

function formatTimestamp(ts: string): string {
  const date = new Date(ts);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  
  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins} min ago`;
  if (diffMins < 1440) return `${Math.floor(diffMins / 60)} hours ago`;
  return `${Math.floor(diffMins / 1440)} days ago`;
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
}
```

---

### **Step 4: Create Persistent Error Banner**

```typescript
// components/OperationErrorBanner.tsx

import React, { useState } from 'react';
import { OperationSummary } from '@/types/jobs';

interface Props {
  lastOperation?: OperationSummary;
  onDismiss?: () => void;
  onTryLiveFailover?: () => void;
}

export function OperationErrorBanner({ lastOperation, onDismiss, onTryLiveFailover }: Props) {
  const [dismissed, setDismissed] = useState(false);

  if (!lastOperation || lastOperation.status !== 'failed' || dismissed) {
    return null;
  }

  const handleDismiss = () => {
    setDismissed(true);
    onDismiss?.();
  };

  const showLiveFailoverButton = 
    lastOperation.actionable_steps?.some(step => 
      step.toLowerCase().includes('live failover')
    );

  return (
    <div className="bg-red-50 border-l-4 border-red-500 p-4 mb-4">
      <div className="flex items-start">
        <div className="flex-shrink-0">
          <span className="text-2xl">⚠️</span>
        </div>
        
        <div className="ml-3 flex-1">
          <h3 className="text-red-800 font-semibold">
            {lastOperation.operation_type.replace('_', ' ')} Failed
          </h3>
          
          <div className="mt-2 text-sm text-red-700">
            <p className="font-medium">Failed at: {lastOperation.failed_step}</p>
            <p className="mt-1">{lastOperation.error_message}</p>
          </div>
          
          {lastOperation.actionable_steps && lastOperation.actionable_steps.length > 0 && (
            <div className="mt-3">
              <p className="text-sm font-medium text-red-800">What you can do:</p>
              <ul className="mt-1 text-sm text-red-700 list-disc list-inside space-y-1">
                {lastOperation.actionable_steps.map((step, i) => (
                  <li key={i}>{step}</li>
                ))}
              </ul>
            </div>
          )}
          
          <div className="mt-4 flex gap-2">
            <button
              onClick={handleDismiss}
              className="px-3 py-1 text-sm bg-white border border-red-300 rounded hover:bg-red-50"
            >
              Dismiss
            </button>
            
            {showLiveFailoverButton && onTryLiveFailover && (
              <button
                onClick={onTryLiveFailover}
                className="px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700"
              >
                Try Live Failover
              </button>
            )}
            
            <button
              onClick={() => {/* Open details modal */}}
              className="px-3 py-1 text-sm text-red-700 hover:text-red-900"
            >
              View Details
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
```

---

## 🔍 **WHERE TO INTEGRATE**

### **Current GUI Structure** (Assumed):

```
app/
├── virtual-machines/
│   └── [vmName]/
│       └── page.tsx          ← Add UnifiedJobList here
├── components/
│   ├── JobCard.tsx           ← May already exist for replications
│   ├── UnifiedJobList.tsx    ← NEW: Create this
│   ├── OperationErrorBanner.tsx  ← NEW: Create this
│   └── JobErrorDetailsModal.tsx  ← NEW: Create this
└── api/
    └── vm-contexts/
        └── route.ts          ← May need proxy to backend
```

**Integration Points**:
1. **VM Details Page**: Add `<UnifiedJobList>` where job history is shown
2. **VM Details Page**: Add `<OperationErrorBanner>` at top
3. **Sidebar/Cards**: Show last operation status indicator
4. **Modals**: Use sanitized errors in all failure dialogs

---

## ⚡ **QUICK START FOR GUI SESSION**

```bash
# 1. Read essential context
Read: /home/pgrayson/migratekit-cloudstack/docs/api/UNIFIED_JOBS_API.md
Read: /home/pgrayson/migratekit-cloudstack/AI_Helper/FAILOVER_VISIBILITY_ENHANCEMENT_JOB_SHEET.md

# 2. Test the API
curl "http://10.245.246.148:8082/api/v1/vm-contexts/ctx-pgtest1-20251003-140708/recent-jobs"

# 3. Find GUI codebase
cd /home/pgrayson/migratekit-cloudstack
# Look for: migration-dashboard, frontend, gui, etc.

# 4. Implement components as documented above

# 5. Test with real data on 10.245.246.148
```

---

## 🎯 **SUCCESS CRITERIA**

### **User Experience Goals**:
- [ ] Users can see what failed without asking
- [ ] Users know what action to take next
- [ ] Failed jobs visible for 7+ days
- [ ] Consistent UX with replication jobs
- [ ] Failover/rollback jobs in same location as replications

### **Technical Requirements**:
- [ ] Zero technical implementation details exposed
- [ ] All errors show sanitized messages
- [ ] Actionable steps displayed for all failures
- [ ] Unified job list shows all operation types
- [ ] Failed operations persist across page refreshes

### **Visual Requirements**:
- [ ] Consistent styling for all job types
- [ ] Clear status indicators
- [ ] Prominent error messages
- [ ] Easy-to-read actionable steps
- [ ] Mobile-responsive design

---

## 📝 **EXAMPLE TEST SCENARIO**

**Backend has stored** (for pgtest1):
```json
{
  "error_message": "KVM driver installation failed - compatibility issue",
  "failed_step": "Preparing Drivers for Compatibility",
  "actionable_steps": [
    "Try live failover (no driver modification)",
    "Verify VM is Windows-based"
  ]
}
```

**GUI should display**:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
❌ Test Failover - Failed (60%)

Issue: KVM driver installation failed - 
compatibility issue

What you can do:
• Try live failover (no driver modification)
• Verify VM is Windows-based

3 minutes ago
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**User clicks "View Details"**:
```
Modal shows:
- Failed Step: Preparing Drivers for Compatibility
- Progress: 60% (3 of 5 steps)
- Duration: 25 seconds
- Category: Compatibility
- Full actionable steps list
- [Try Live Failover] button
```

---

## 🚨 **VALIDATION CHECKLIST**

Before considering GUI work complete:

- [ ] Can see failed failover in job list
- [ ] Error message is user-friendly (no "virt-v2v" or "VirtIO")
- [ ] Step name is user-friendly (no "virtio-driver-injection")
- [ ] Actionable steps are displayed
- [ ] Failed job persists across page refresh
- [ ] Failover jobs shown in same location as replications
- [ ] Can trigger live failover from error banner
- [ ] Admin can see technical details if needed
- [ ] Mobile responsive
- [ ] Tested with: failed test failover, failed live failover, failed rollback

---

## 📚 **BACKEND FILES TO REFERENCE**

**For understanding sanitization logic**:
- `source/current/oma/failover/error_sanitizer.go` - All sanitization rules
- `source/current/oma/failover/step_display_names.go` - All step name mappings

**For API structure**:
- `source/current/oma/api/handlers/vm_contexts.go` - Unified jobs endpoint
- `source/current/oma/database/repository.go` - VMContextDetails structure

**Don't modify these** - they're backend files. Just reference for understanding.

---

## 🎯 **YOUR DELIVERABLES**

1. ✅ Unified job list component
2. ✅ Persistent error banner component
3. ✅ Enhanced error detail modal
4. ✅ Integration into VM details views
5. ✅ Consistent styling with existing UI
6. ✅ Testing with real failed operations

---

**Backend Status**: ✅ Complete  
**API Endpoints**: ✅ Live on 10.245.246.147 & .148  
**Test Data**: ✅ Available (pgtest1 and pgtest3 have failed operations)  
**Your Mission**: Integrate into GUI for seamless UX  

**Good luck! The backend team has provided everything you need.** 🚀


