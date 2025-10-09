# GROK TASK 6: Protection Flows GUI Wiring + Theme Fix

**Date:** 2025-10-09  
**Status:** 🟡 Ready to Start  
**Prerequisites:** Backend Tasks 1-5 Complete ✅  
**Backend API:** http://localhost:8082/api/v1/protection-flows

---

## 🎯 Dual Objectives

This task has TWO critical objectives that must both be completed:

### Objective 1: Wire GUI to Backend API (Task 6 from job sheet)
Replace all mock data with real API calls to the Protection Flows Engine backend

### Objective 2: Fix Light Mode Support (Regression introduced during earlier refactoring)
Replace all hardcoded dark mode colors with semantic color tokens to restore light/dark mode compatibility

---

## 🚨 CRITICAL CONTEXT

### What Happened
During the modal-to-panel refactoring, the Protection Flows page was hardcoded with dark mode colors (e.g., `bg-gray-900`, `text-white`, `border-gray-700`). This broke light mode support and made the page inconsistent with the rest of the application.

### Why This Matters
**All other pages in the Sendense GUI support light/dark mode switching.** The Protection Flows page is the ONLY page with this regression. This is a **critical UX inconsistency** that must be fixed.

### How The Theme System Works
The Sendense GUI uses CSS custom properties defined in `app/globals.css`:

**Light Mode (`:root`):**
```css
--background: oklch(1 0 0);        /* white */
--foreground: oklch(0.145 0 0);    /* near-black */
--card: oklch(1 0 0);              /* white */
--border: oklch(0.922 0 0);        /* light gray */
```

**Dark Mode (`.dark`):**
```css
--background: var(--sendense-bg);  /* #0a0e17 */
--foreground: var(--sendense-text); /* #e4e7eb */
--card: var(--sendense-surface);   /* #12172a */
--border: #2a3441;                 /* dark gray */
```

**Semantic Tailwind Classes:**
- `bg-background` → Uses `--background` (white in light, dark in dark)
- `text-foreground` → Uses `--foreground` (black in light, white in dark)
- `bg-card` → Uses `--card` (white in light, dark surface in dark)
- `border-border` → Uses `--border` (light gray in light, dark gray in dark)
- `text-muted-foreground` → Secondary text color
- etc.

---

## 📝 TASK 1: Fix Theme Support (Do This FIRST)

### Files to Fix

#### 1. `/source/current/sendense-gui/app/protection-flows/page.tsx`

**Current Hardcoded Dark Mode:**
```typescript
<div className="h-screen bg-gray-900">  // ❌ WRONG
  <div className="flex flex-col h-full bg-gray-900">  // ❌ WRONG
    <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700">  // ❌ WRONG
      <h2 className="text-lg font-semibold text-white">  // ❌ WRONG
      <p className="text-xs text-gray-400">  // ❌ WRONG
      <PanelResizeHandle className="h-1 bg-gray-700 hover:bg-blue-500" />  // ❌ WRONG
      <div className="h-full bg-gray-900 border-t border-gray-700">  // ❌ WRONG
        <p className="text-gray-400">  // ❌ WRONG
```

**Fixed with Semantic Colors:**
```typescript
<div className="h-screen bg-background">  // ✅ CORRECT
  <div className="flex flex-col h-full bg-background">  // ✅ CORRECT
    <div className="flex items-center justify-between px-4 py-3 border-b border-border">  // ✅ CORRECT
      <h2 className="text-lg font-semibold text-foreground">  // ✅ CORRECT
      <p className="text-xs text-muted-foreground">  // ✅ CORRECT
      <PanelResizeHandle className="h-1 bg-border hover:bg-primary transition-colors" />  // ✅ CORRECT
      <div className="h-full bg-background border-t border-border">  // ✅ CORRECT
        <p className="text-muted-foreground">  // ✅ CORRECT
```

**Button Theme (lines 177-183):**
```typescript
// ❌ CURRENT (hardcoded dark)
<button
  onClick={() => setIsLogsOpen(!isLogsOpen)}
  className={`p-2 rounded-lg transition-colors ${
    isLogsOpen
      ? 'bg-blue-500/20 text-blue-400'
      : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
  }`}
>

// ✅ FIXED (semantic colors)
<button
  onClick={() => setIsLogsOpen(!isLogsOpen)}
  className={`p-2 rounded-lg transition-colors ${
    isLogsOpen
      ? 'bg-primary/20 text-primary'
      : 'bg-muted text-muted-foreground hover:bg-muted/80'
  }`}
>
```

#### 2. `/source/current/sendense-gui/components/features/protection-flows/FlowDetailsPanel.tsx`

**Check for hardcoded colors in:**
- All `className` props with `bg-gray-*`, `text-gray-*`, `text-white`, `border-gray-*`
- Replace with semantic equivalents:
  - `bg-gray-800` → `bg-card`
  - `bg-gray-900` → `bg-background`
  - `text-white` → `text-foreground`
  - `text-gray-400` → `text-muted-foreground`
  - `text-gray-300` → `text-foreground`
  - `border-gray-700` → `border-border`
  - `border-gray-600` → `border-border`
  - `hover:bg-gray-700` → `hover:bg-muted`

#### 3. `/source/current/sendense-gui/components/features/protection-flows/FlowsTable.tsx`

Check and fix any hardcoded colors in table styling.

#### 4. `/source/current/sendense-gui/components/features/protection-flows/JobLogsDrawer.tsx`

**Check especially:**
- Drawer background colors
- Text colors
- Border colors
- Header styling
- Log entry backgrounds

---

## 📝 TASK 2: Wire GUI to Backend API

### Overview
Replace all mock data with real API calls using React Query.

### API Service Layer Location
`/source/current/sendense-gui/src/features/protection-flows/api/`

### Required API Methods

Create `/source/current/sendense-gui/src/features/protection-flows/api/protectionFlowsApi.ts`:

```typescript
import axios from 'axios';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082';

export interface ProtectionFlow {
  id: string;
  name: string;
  flow_type: 'backup' | 'replication';
  target_type: 'vm' | 'group';
  target_id: string;
  repository_id: string;
  schedule_id?: string;
  policy_id?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  last_execution?: string;
  next_execution?: string;
  execution_count: number;
  success_count: number;
  failure_count: number;
}

export interface FlowExecution {
  id: string;
  flow_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  started_at: string;
  completed_at?: string;
  error_message?: string;
  bytes_transferred?: number;
  duration_seconds?: number;
}

export interface FlowSummary {
  total_flows: number;
  enabled_flows: number;
  disabled_flows: number;
  backup_flows: number;
  replication_flows: number;
  total_executions_today: number;
  successful_executions_today: number;
  failed_executions_today: number;
}

// GET /api/v1/protection-flows
export async function listFlows(): Promise<{ flows: ProtectionFlow[]; total: number }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/protection-flows`);
  return data;
}

// GET /api/v1/protection-flows/{id}
export async function getFlow(id: string): Promise<ProtectionFlow> {
  const { data } = await axios.get(`${API_BASE}/api/v1/protection-flows/${id}`);
  return data;
}

// POST /api/v1/protection-flows
export async function createFlow(flow: Omit<ProtectionFlow, 'id' | 'created_at' | 'updated_at' | 'execution_count' | 'success_count' | 'failure_count'>): Promise<ProtectionFlow> {
  const { data } = await axios.post(`${API_BASE}/api/v1/protection-flows`, flow);
  return data;
}

// PUT /api/v1/protection-flows/{id}
export async function updateFlow(id: string, flow: Partial<ProtectionFlow>): Promise<ProtectionFlow> {
  const { data } = await axios.put(`${API_BASE}/api/v1/protection-flows/${id}`, flow);
  return data;
}

// DELETE /api/v1/protection-flows/{id}
export async function deleteFlow(id: string): Promise<void> {
  await axios.delete(`${API_BASE}/api/v1/protection-flows/${id}`);
}

// PATCH /api/v1/protection-flows/{id}/enable
export async function enableFlow(id: string): Promise<ProtectionFlow> {
  const { data } = await axios.patch(`${API_BASE}/api/v1/protection-flows/${id}/enable`);
  return data;
}

// PATCH /api/v1/protection-flows/{id}/disable
export async function disableFlow(id: string): Promise<ProtectionFlow> {
  const { data } = await axios.patch(`${API_BASE}/api/v1/protection-flows/${id}/disable`);
  return data;
}

// POST /api/v1/protection-flows/{id}/execute
export async function executeFlow(id: string): Promise<{ execution_id: string; message: string }> {
  const { data } = await axios.post(`${API_BASE}/api/v1/protection-flows/${id}/execute`);
  return data;
}

// GET /api/v1/protection-flows/{id}/executions
export async function getFlowExecutions(id: string): Promise<{ executions: FlowExecution[]; total: number }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/protection-flows/${id}/executions`);
  return data;
}

// GET /api/v1/protection-flows/{id}/status
export async function getFlowStatus(id: string): Promise<{ flow: ProtectionFlow; last_execution?: FlowExecution; next_run?: string }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/protection-flows/${id}/status`);
  return data;
}

// GET /api/v1/protection-flows/summary
export async function getFlowSummary(): Promise<FlowSummary> {
  const { data } = await axios.get(`${API_BASE}/api/v1/protection-flows/summary`);
  return data;
}

// POST /api/v1/protection-flows/bulk-enable
export async function bulkEnableFlows(flow_ids: string[]): Promise<{ successful: number; failed: number; errors: Record<string, string> }> {
  const { data } = await axios.post(`${API_BASE}/api/v1/protection-flows/bulk-enable`, { flow_ids });
  return data;
}

// POST /api/v1/protection-flows/bulk-disable
export async function bulkDisableFlows(flow_ids: string[]): Promise<{ successful: number; failed: number; errors: Record<string, string> }> {
  const { data } = await axios.post(`${API_BASE}/api/v1/protection-flows/bulk-disable`, { flow_ids });
  return data;
}

// POST /api/v1/protection-flows/bulk-delete
export async function bulkDeleteFlows(flow_ids: string[]): Promise<{ successful: number; failed: number; errors: Record<string, string> }> {
  const { data } = await axios.post(`${API_BASE}/api/v1/protection-flows/bulk-delete`, { flow_ids });
  return data;
}
```

### React Query Hooks

Create `/source/current/sendense-gui/src/features/protection-flows/hooks/useProtectionFlows.ts`:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as api from '../api/protectionFlowsApi';

export function useProtectionFlows() {
  return useQuery({
    queryKey: ['protection-flows'],
    queryFn: api.listFlows,
    refetchInterval: 5000, // Refresh every 5 seconds for live updates
  });
}

export function useProtectionFlow(id: string) {
  return useQuery({
    queryKey: ['protection-flow', id],
    queryFn: () => api.getFlow(id),
    enabled: !!id,
  });
}

export function useFlowSummary() {
  return useQuery({
    queryKey: ['protection-flows', 'summary'],
    queryFn: api.getFlowSummary,
    refetchInterval: 10000, // Refresh every 10 seconds
  });
}

export function useFlowExecutions(flowId: string) {
  return useQuery({
    queryKey: ['protection-flow', flowId, 'executions'],
    queryFn: () => api.getFlowExecutions(flowId),
    enabled: !!flowId,
    refetchInterval: 3000, // Refresh every 3 seconds for active jobs
  });
}

export function useCreateFlow() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: api.createFlow,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flows', 'summary'] });
    },
  });
}

export function useUpdateFlow() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, flow }: { id: string; flow: Partial<api.ProtectionFlow> }) => 
      api.updateFlow(id, flow),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flow', variables.id] });
    },
  });
}

export function useDeleteFlow() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: api.deleteFlow,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flows', 'summary'] });
    },
  });
}

export function useExecuteFlow() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: api.executeFlow,
    onSuccess: (_, flowId) => {
      queryClient.invalidateQueries({ queryKey: ['protection-flow', flowId, 'executions'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flow', flowId] });
    },
  });
}

export function useEnableFlow() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: api.enableFlow,
    onSuccess: (_, flowId) => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flow', flowId] });
    },
  });
}

export function useDisableFlow() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: api.disableFlow,
    onSuccess: (_, flowId) => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flow', flowId] });
    },
  });
}
```

### Component Updates

#### 1. Update `page.tsx` to use real API:

```typescript
// Remove mockFlows, replace with:
const { data: flowsData, isLoading, error } = useProtectionFlows();
const flows = flowsData?.flows || [];

// Update create handler:
const createFlowMutation = useCreateFlow();
const handleCreateFlowSubmit = async (newFlowData: Omit<Flow, 'id' | 'status' | 'lastRun' | 'progress'>) => {
  await createFlowMutation.mutateAsync(newFlowData);
};

// Update update handler:
const updateFlowMutation = useUpdateFlow();
const handleUpdateFlow = async (flowId: string, updates: Partial<Flow>) => {
  await updateFlowMutation.mutateAsync({ id: flowId, flow: updates });
  setEditingFlow(null);
};

// Update delete handler:
const deleteFlowMutation = useDeleteFlow();
const handleConfirmDelete = async (flowId: string) => {
  await deleteFlowMutation.mutateAsync(flowId);
  if (selectedFlowId === flowId) {
    setSelectedFlowId(undefined);
  }
  setDeletingFlow(null);
};

// Update run now handler:
const executeFlowMutation = useExecuteFlow();
const handleRunNow = async (flow: Flow) => {
  await executeFlowMutation.mutateAsync(flow.id);
};
```

#### 2. Update `FlowDetailsPanel.tsx` to use real execution data:

```typescript
const { data: executionsData } = useFlowExecutions(flow.id);
const executions = executionsData?.executions || [];
```

#### 3. Update `JobLogsDrawer.tsx` to show real logs:

This will need integration with the job logs system (separate task, but wire up the basic structure).

---

## ✅ Completion Checklist

### Theme Fix
- [ ] `page.tsx` - All hardcoded dark colors replaced with semantic tokens
- [ ] `FlowDetailsPanel.tsx` - All hardcoded dark colors replaced
- [ ] `FlowsTable.tsx` - All hardcoded dark colors replaced
- [ ] `JobLogsDrawer.tsx` - All hardcoded dark colors replaced
- [ ] Test in light mode (remove `className="dark"` from `app/layout.tsx` line 28)
- [ ] Test in dark mode (restore `className="dark"`)
- [ ] Verify theme toggle works (if implemented)

### API Wiring
- [ ] API service created with all 15 endpoints
- [ ] React Query hooks created with proper invalidation
- [ ] `page.tsx` using real API data (not mock)
- [ ] Create flow working with real API
- [ ] Update flow working with real API
- [ ] Delete flow working with real API
- [ ] Execute flow working with real API
- [ ] Flow executions loading in details panel
- [ ] Loading states showing during API calls
- [ ] Error handling implemented for all operations
- [ ] Auto-refresh working (5s for flows, 3s for executions)

### Quality Gates
- [ ] No TypeScript errors (`npm run build`)
- [ ] No console errors in browser
- [ ] No mock data remaining in code
- [ ] All API calls use proper error handling
- [ ] Loading spinners show during operations
- [ ] Success/error toasts show after mutations
- [ ] Light mode looks professional (not broken)
- [ ] Dark mode still looks professional (not changed)

---

## 🔧 Testing Instructions

### 1. Test Theme Support
```bash
# Test light mode
# Edit app/layout.tsx line 28: change className="dark" to className=""
# Reload page - should look professional in light mode

# Test dark mode
# Restore className="dark"
# Reload page - should look professional (unchanged)
```

### 2. Test API Integration
```bash
# Backend should be running at http://localhost:8082
curl http://localhost:8082/api/v1/protection-flows/summary

# In GUI:
# 1. Create a new flow - should POST and appear in list
# 2. Edit a flow - should PUT and update in list
# 3. Execute a flow - should POST to execute endpoint
# 4. Delete a flow - should DELETE and remove from list
# 5. Check executions tab - should show real execution history
```

---

## 📚 Reference Files

- **Backend API:** `/source/current/sha/api/handlers/protection_flow_handlers.go`
- **Backend Service:** `/source/current/sha/services/protection_flow_service.go`
- **Database Models:** `/source/current/sha/database/models.go`
- **Theme System:** `/source/current/sendense-gui/app/globals.css`
- **Example API Usage:** Check other pages like Dashboard, Appliances, Repositories

---

## 🚨 CRITICAL RULES

1. **Theme First:** Fix theme support BEFORE wiring API
2. **No Hardcoded Colors:** NEVER use `bg-gray-*`, `text-white`, etc.
3. **Use Semantic Tokens:** ALWAYS use `bg-background`, `text-foreground`, `border-border`, etc.
4. **Test Both Modes:** MUST test in both light and dark mode
5. **No Mock Data:** ALL mock data must be replaced with real API calls
6. **Error Handling:** ALL mutations must have proper error handling
7. **Loading States:** ALL async operations must show loading indicators
8. **Auto-Refresh:** Use React Query's refetchInterval for live updates

---

**Ready to Start?** Yes, backend is deployed and operational. Proceed with Task 1 (theme fix) first, then Task 2 (API wiring).

