# Protection Flows Table - Complete Wiring Implementation

**Date:** October 10, 2025  
**Priority:** HIGH  
**Complexity:** Medium

---

## 🎯 Objective

Wire up the Protection Flows table (Backup & Replication Jobs) to display real-time, accurate information:

1. **Progress Bar** - Show aggregate progress for flows with running jobs
2. **Status** - Display correct current status (not stuck on "Pending")
3. **Last Run** - Show actual last execution time
4. **Next Run** - Show next scheduled run time (if scheduled)

---

## 🐛 Current Issues

### Issue 1: Status Shows "Pending" for All Flows
**Problem:** All flows show status as "Pending" even when they have completed jobs.

**Root Cause:** The backend returns this structure:
```json
{
  "status": {
    "last_execution_status": "success",  // ← This exists
    "last_execution_time": "2025-10-10T14:28:18Z",
    "next_execution_time": null,
    "total_executions": 14,
    "successful_executions": 14,
    "failed_executions": 0
  }
}
```

But the frontend `getUIStatus()` function in `src/features/protection-flows/types/index.ts` is checking `flow.status?.last_execution_status` correctly, so there might be a data transformation issue.

**Location:**
- Backend: `/home/oma_admin/sendense/source/current/sha/api/handlers/protection_flow_handlers.go` (line 588-636)
- Frontend Type: `/home/oma_admin/sendense/source/current/sendense-gui/src/features/protection-flows/types/index.ts` (line 36-44)
- Frontend Display: `/home/oma_admin/sendense/source/current/sendense-gui/src/features/protection-flows/components/FlowsTable/FlowRow.tsx` (line 52-61)

---

### Issue 2: Last Run Shows "Never"
**Problem:** All flows show "Never" for Last Run even when `last_execution_time` exists.

**Root Cause:** Backend sends `status.last_execution_time: "2025-10-10T14:28:18Z"` but frontend is looking for `flow.lastRun` (which doesn't exist).

**Current Code (FlowRow.tsx line 64):**
```typescript
{formatDate(flow.lastRun)}  // ← Looking for flow.lastRun
```

**Backend Sends:**
```json
{
  "status": {
    "last_execution_time": "2025-10-10T14:28:18Z"  // ← This is what we have
  }
}
```

**Fix Needed:** Map `status.last_execution_time` to `lastRun` during data transformation.

---

### Issue 3: Next Run Shows "Never"
**Problem:** All flows show "Never" for Next Run even when scheduled.

**Root Cause:** Backend sends `status.next_execution_time` but frontend expects `flow.nextRun`.

**Fix Needed:** 
1. Map `status.next_execution_time` to `nextRun` during data transformation
2. If null and flow has `schedule_id`, calculate next run from cron expression

**Schedule Integration:**
Backend also provides:
```json
{
  "schedule_id": "schedule-uuid",
  "schedule_name": "Daily Backup",
  "schedule_cron": "0 2 * * *"  // ← Can calculate next run from this
}
```

---

### Issue 4: Progress Bar Missing
**Problem:** No progress bar shown for flows with running jobs.

**Current:** The table shows status dot + text but no progress indicator.

**Needed:** 
1. Detect if flow has running execution
2. Query running jobs for that flow
3. Calculate aggregate progress:
   - If flow targets single VM: Show single job progress
   - If flow targets group: Show aggregate (completed jobs / total jobs)
4. Display progress bar with percentage

**Progress Calculation Logic:**
```typescript
// For running flows:
// 1. Get current execution from /api/v1/protection-flows/{id}/executions
// 2. If execution.status === 'running':
//    - jobs_completed / jobs_created = % complete
//    - OR get live job progress from backup_jobs table
// 3. Display progress bar with current %
```

---

## 📊 Data Flow Analysis

### Backend API Response (ACTUAL)
```json
{
  "flows": [
    {
      "id": "92057168-a502-11f0-b62d-020200cc0023",
      "name": "pgtest1",
      "flow_type": "backup",
      "target_type": "vm",
      "target_id": "ctx-pgtest1-20251006-203401",
      "repository_id": "repo-local-1759780872",
      "schedule_id": null,
      "schedule_name": null,
      "schedule_cron": null,
      "enabled": true,
      "status": {
        "last_execution_id": "exec-uuid",
        "last_execution_status": "success",
        "last_execution_time": "2025-10-10T14:28:18Z",
        "next_execution_time": null,
        "total_executions": 14,
        "successful_executions": 14,
        "failed_executions": 0
      },
      "created_at": "2025-10-06T20:34:01Z",
      "updated_at": "2025-10-10T14:28:18Z",
      "created_by": "system"
    }
  ],
  "total": 2
}
```

### Frontend Type (EXPECTED)
```typescript
interface Flow {
  id: string;
  name: string;
  flow_type: 'backup' | 'replication';
  target_type: 'vm' | 'group';
  target_id: string;
  repository_id?: string;
  schedule_id?: string;
  enabled: boolean;
  status: FlowStatusData;
  
  // 🔥 These are missing and need to be computed:
  lastRun?: string;      // ← Map from status.last_execution_time
  nextRun?: string;      // ← Map from status.next_execution_time OR calculate from schedule_cron
  progress?: number;     // ← Calculate from running jobs (0-100)
  source?: string;       // ← Resolve target_id to VM/group name
  destination?: string;  // ← Resolve repository_id to repository name
}
```

---

## 🛠️ Implementation Plan

### Task 1: Add Data Transformation Layer
**File:** `src/features/protection-flows/api/protectionFlowsApi.ts`

**Add transformation function:**
```typescript
function transformFlowResponse(apiFlow: any): Flow {
  return {
    ...apiFlow,
    // Map backend fields to frontend fields
    lastRun: apiFlow.status?.last_execution_time || undefined,
    nextRun: apiFlow.status?.next_execution_time || calculateNextRun(apiFlow),
    progress: undefined, // Will be populated by separate query for running flows
    source: apiFlow.target_name || apiFlow.target_id,
    destination: apiFlow.repository_name || apiFlow.repository_id,
  };
}

function calculateNextRun(flow: any): string | undefined {
  // If next_execution_time exists, use it
  if (flow.status?.next_execution_time) {
    return flow.status.next_execution_time;
  }
  
  // If flow has schedule_cron, calculate next run
  if (flow.schedule_cron) {
    // Use cron-parser library to calculate next run time
    // npm install cron-parser
    const parser = require('cron-parser');
    try {
      const interval = parser.parseExpression(flow.schedule_cron);
      return interval.next().toISOString();
    } catch (e) {
      return undefined;
    }
  }
  
  return undefined;
}
```

**Update listFlows:**
```typescript
export async function listFlows(): Promise<{ flows: Flow[]; total: number }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/protection-flows`);
  return {
    flows: data.flows.map(transformFlowResponse),
    total: data.total
  };
}
```

---

### Task 2: Add Progress Calculation
**New File:** `src/features/protection-flows/hooks/useFlowProgress.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import axios from 'axios';

const API_BASE = '';

interface FlowProgress {
  flowId: string;
  isRunning: boolean;
  progress: number;
  currentExecution?: {
    id: string;
    jobs_created: number;
    jobs_completed: number;
    jobs_failed: number;
  };
}

export function useFlowProgress(flowId: string, enabled: boolean = true) {
  return useQuery({
    queryKey: ['flow-progress', flowId],
    queryFn: async (): Promise<FlowProgress> => {
      // 1. Get latest execution
      const { data } = await axios.get(
        `${API_BASE}/api/v1/protection-flows/${flowId}/executions?limit=1`
      );
      
      const latestExecution = data.executions?.[0];
      
      // 2. Check if running
      if (!latestExecution || latestExecution.status !== 'running') {
        return {
          flowId,
          isRunning: false,
          progress: 0,
        };
      }
      
      // 3. Calculate progress from execution
      const jobsCreated = latestExecution.jobs_created || 0;
      const jobsCompleted = latestExecution.jobs_completed || 0;
      
      let progress = 0;
      if (jobsCreated > 0) {
        progress = Math.round((jobsCompleted / jobsCreated) * 100);
      }
      
      // 4. For more accurate progress, we could also query individual backup jobs
      // and get their real-time progress_percent from telemetry data
      
      return {
        flowId,
        isRunning: true,
        progress: Math.min(progress, 99), // Never show 100% until execution completes
        currentExecution: {
          id: latestExecution.id,
          jobs_created: jobsCreated,
          jobs_completed: jobsCompleted,
          jobs_failed: latestExecution.jobs_failed || 0,
        },
      };
    },
    enabled: enabled,
    refetchInterval: (data) => {
      // Poll every 2 seconds if running, otherwise don't poll
      return data?.isRunning ? 2000 : false;
    },
  });
}

// Bulk progress hook for all flows in the table
export function useAllFlowsProgress(flowIds: string[], enabled: boolean = true) {
  return useQuery({
    queryKey: ['all-flows-progress', flowIds],
    queryFn: async (): Promise<Record<string, FlowProgress>> => {
      // Fetch progress for all flows in parallel
      const results = await Promise.allSettled(
        flowIds.map(async (flowId) => {
          const { data } = await axios.get(
            `${API_BASE}/api/v1/protection-flows/${flowId}/executions?limit=1`
          );
          
          const latestExecution = data.executions?.[0];
          
          if (!latestExecution || latestExecution.status !== 'running') {
            return { flowId, isRunning: false, progress: 0 };
          }
          
          const jobsCreated = latestExecution.jobs_created || 0;
          const jobsCompleted = latestExecution.jobs_completed || 0;
          const progress = jobsCreated > 0 
            ? Math.min(Math.round((jobsCompleted / jobsCreated) * 100), 99)
            : 0;
          
          return {
            flowId,
            isRunning: true,
            progress,
            currentExecution: {
              id: latestExecution.id,
              jobs_created: jobsCreated,
              jobs_completed: jobsCompleted,
              jobs_failed: latestExecution.jobs_failed || 0,
            },
          };
        })
      );
      
      // Build record of flowId -> progress
      const progressMap: Record<string, FlowProgress> = {};
      results.forEach((result, index) => {
        if (result.status === 'fulfilled') {
          progressMap[flowIds[index]] = result.value;
        }
      });
      
      return progressMap;
    },
    enabled: enabled && flowIds.length > 0,
    refetchInterval: 2000, // Poll every 2 seconds for real-time updates
  });
}
```

---

### Task 3: Update FlowsTable Component
**File:** `src/features/protection-flows/components/FlowsTable/index.tsx`

**Add progress hook:**
```typescript
import { useAllFlowsProgress } from '../../hooks/useFlowProgress';

export function FlowsTable({ flows, onSelectFlow, selectedFlowId }: FlowsTableProps) {
  // ... existing state ...
  
  // 🆕 NEW: Get progress for all flows
  const flowIds = flows.map(f => f.id);
  const { data: progressData } = useAllFlowsProgress(flowIds, flows.length > 0);
  
  // ... rest of existing code ...
  
  // Pass progress to FlowRow
  sortedFlows.map((flow) => {
    const flowProgress = progressData?.[flow.id];
    
    return (
      <FlowRow
        key={flow.id}
        flow={{
          ...flow,
          progress: flowProgress?.progress // ✅ Add progress to flow object
        }}
        isSelected={selectedFlowId === flow.id}
        onSelect={onSelectFlow}
        onEdit={handleEdit}
        onDelete={handleDelete}
        onRunNow={handleRunNow}
      />
    );
  })
}
```

---

### Task 4: Update FlowRow Component
**File:** `src/features/protection-flows/components/FlowsTable/FlowRow.tsx`

**Add progress bar display:**
```typescript
import { Progress } from "@/components/ui/progress"; // Assuming shadcn/ui Progress component

export function FlowRow({ flow, isSelected, onSelect, onEdit, onDelete, onRunNow }: FlowRowProps) {
  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Never';
    try {
      return format(new Date(dateString), 'MMM dd, yyyy HH:mm');
    } catch {
      return dateString;
    }
  };
  
  const uiStatus = getUIStatus(flow);
  const isRunning = uiStatus === 'running';
  const hasProgress = flow.progress !== undefined && flow.progress > 0;

  return (
    <tr
      className={`border-b border-border hover:bg-muted/50 cursor-pointer transition-colors ${
        isSelected ? 'bg-primary/5 border-primary/20' : ''
      }`}
      onClick={() => onSelect(flow)}
    >
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
          <div className="w-2 h-2 rounded-full bg-primary flex-shrink-0" />
          <div className="flex-1">
            <div className="font-medium text-foreground">{flow.name}</div>
            {flow.source && flow.destination && (
              <div className="text-sm text-muted-foreground">
                {flow.source} → {flow.destination}
              </div>
            )}
            {/* 🆕 NEW: Progress bar for running flows */}
            {isRunning && hasProgress && (
              <div className="mt-2 flex items-center gap-2">
                <Progress value={flow.progress} className="h-1.5 flex-1" />
                <span className="text-xs text-muted-foreground min-w-[3ch]">
                  {flow.progress}%
                </span>
              </div>
            )}
          </div>
        </div>
      </td>
      <td className="px-4 py-3">
        <Badge variant="outline" className="capitalize">
          {flow.flow_type}
        </Badge>
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center gap-2">
          <div className={`w-2 h-2 rounded-full ${
            uiStatus === 'success' ? 'bg-green-500' :
            uiStatus === 'running' ? 'bg-blue-500 animate-pulse' :  // ✅ Add pulse for running
            uiStatus === 'warning' ? 'bg-yellow-500' :
            uiStatus === 'error' ? 'bg-red-500' :
            'bg-muted-foreground'
          }`} />
          <span className="capitalize text-sm">{uiStatus}</span>
        </div>
      </td>
      <td className="px-4 py-3 text-sm text-muted-foreground">
        {formatDate(flow.lastRun)}
      </td>
      <td className="px-4 py-3 text-sm text-muted-foreground">
        {formatDate(flow.nextRun)}
      </td>
      {/* ... rest of existing code ... */}
    </tr>
  );
}
```

---

### Task 5: Install Required Dependencies
```bash
# If cron-parser is needed for next run calculation
npm install cron-parser
npm install --save-dev @types/cron-parser
```

---

## 🧪 Testing Checklist

### Status Display
- [ ] Create a test flow and run it to completion
- [ ] Verify status shows "Success" (green) when completed
- [ ] Start a backup and verify status shows "Running" (blue, pulsing)
- [ ] Cancel a backup and verify status shows "Warning" (yellow)
- [ ] Force a backup to fail and verify status shows "Error" (red)

### Last Run
- [ ] Flow with completed backup shows "Oct 10, 2025 14:28" format
- [ ] Brand new flow shows "Never"
- [ ] Time updates after each backup completes

### Next Run
- [ ] Unscheduled flow shows "Never"
- [ ] Scheduled flow shows calculated next run time
- [ ] Next run updates after scheduled execution completes

### Progress Bar
- [ ] Start a backup on a single-VM flow
- [ ] Verify progress bar appears below flow name
- [ ] Progress bar shows realistic % (not stuck at 0%)
- [ ] Progress bar updates smoothly every 2-3 seconds
- [ ] Progress bar disappears when backup completes
- [ ] Multi-VM group flow shows aggregate progress (e.g., "3/10 VMs complete")

---

## 📁 Files to Modify

### Frontend Files (sendense-gui)
1. ✅ `/src/features/protection-flows/api/protectionFlowsApi.ts`
   - Add `transformFlowResponse()` function
   - Add `calculateNextRun()` helper
   - Update `listFlows()` to transform data

2. ✅ `/src/features/protection-flows/hooks/useFlowProgress.ts` (NEW FILE)
   - Create `useFlowProgress()` hook
   - Create `useAllFlowsProgress()` bulk hook

3. ✅ `/src/features/protection-flows/components/FlowsTable/index.tsx`
   - Import `useAllFlowsProgress`
   - Fetch progress data
   - Pass progress to FlowRow

4. ✅ `/src/features/protection-flows/components/FlowsTable/FlowRow.tsx`
   - Add Progress component import
   - Display progress bar for running flows
   - Add pulse animation to running status dot

5. ✅ `/src/features/protection-flows/types/index.ts`
   - Verify `Flow` interface includes `lastRun`, `nextRun`, `progress` fields
   - Update if needed

### Backend Files (NO CHANGES NEEDED)
The backend API already provides all necessary data:
- ✅ `status.last_execution_time` 
- ✅ `status.next_execution_time`
- ✅ `schedule_cron` for calculation
- ✅ `/api/v1/protection-flows/{id}/executions` endpoint for progress

---

## 🎯 Success Criteria

1. **Status** - Shows correct color and text ("Success", "Running", "Error", etc.)
2. **Last Run** - Shows actual timestamp or "Never"
3. **Next Run** - Shows calculated next run time or "Never"
4. **Progress** - Visible progress bar with % for running flows
5. **Real-time Updates** - Progress updates every 2-3 seconds during execution
6. **Performance** - No UI lag when fetching progress for multiple flows

---

## 💡 Optional Enhancements (Future)

1. **Detailed Progress Tooltip**
   - Hover over progress bar to see: "3/10 VMs complete, 2 failed, 5 pending"
   
2. **ETA Display**
   - Calculate and show estimated completion time based on current speed
   
3. **Live Job Details**
   - Click progress bar to open modal with per-VM job details
   
4. **Schedule Preview**
   - Hover over Next Run to see next 5 scheduled executions

---

## 📝 Notes

- Progress calculation is **estimation-based** for group flows (jobs completed / jobs created)
- For **single-VM flows**, we could query the actual `backup_jobs.progress_percent` from telemetry for more accuracy
- The `useAllFlowsProgress` hook uses **parallel fetching** to minimize latency
- Consider adding a **loading skeleton** for the progress bar during first fetch
- The 2-second polling interval is a balance between **real-time feel** and **API load**

---

## 🚀 Implementation Order

1. **Phase 1:** Data transformation (Task 1) - Fixes Last Run / Next Run immediately
2. **Phase 2:** Progress hooks (Task 2) - Adds progress calculation logic
3. **Phase 3:** UI updates (Tasks 3-4) - Displays progress bars
4. **Phase 4:** Testing (Task 5) - Verify everything works end-to-end

**Estimated Time:** 2-3 hours total

---

## 🎨 UI Mockup Reference

**Current (Broken):**
```
Name             Type    Status    Last Run    Next Run    Actions
pgtest1          Backup  Pending   Never       Never       ⋯
```

**Target (Fixed):**
```
Name                          Type    Status      Last Run           Next Run           Actions
pgtest1                       Backup  ● Running   Oct 10, 14:28     Oct 11, 02:00     ⋯
  [████████░░░░░░] 63%
  
pgtest3                       Backup  ● Success   Oct 10, 04:50     Never             ⋯
```

---

**END OF JOB SHEET**

