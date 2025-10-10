import axios from 'axios';

// Use empty string to let Next.js rewrites proxy handle the routing
// next.config.ts proxies /api/v1/* to http://localhost:8082/api/v1/*
const API_BASE = '';

export interface ProtectionFlowStatus {
  last_execution_id?: string;
  last_execution_status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  last_execution_time?: string;
  next_execution_time?: string;
  total_executions: number;
  successful_executions: number;
  failed_executions: number;
}

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
  status: ProtectionFlowStatus;
  created_at: string;
  updated_at: string;
  created_by?: string;
  last_execution?: string;
  next_execution?: string;
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
  jobs_created?: number;
  jobs_completed?: number;
  jobs_failed?: number;
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

// Transform backend flow data to frontend format
function transformFlowResponse(apiFlow: any): ProtectionFlow {
  return {
    ...apiFlow,
    status: {
      ...apiFlow.status,
      // Ensure the status fields are properly mapped
      last_execution_status: apiFlow.status?.last_execution_status || 'pending',
      last_execution_time: apiFlow.status?.last_execution_time,
      next_execution_time: apiFlow.status?.next_execution_time,
    },
  };
}

// Calculate next run time from cron expression if available
function calculateNextRun(flow: any): string | undefined {
  // If next_execution_time exists, use it
  if (flow.status?.next_execution_time) {
    return flow.status.next_execution_time;
  }

  // If flow has schedule_cron, calculate next run
  if (flow.schedule_cron) {
    try {
      // Use a simple calculation for next run time
      // For now, just return a placeholder - in production you'd use cron-parser
      const now = new Date();
      // Simple calculation: assume daily if no specific time
      const nextRun = new Date(now);
      nextRun.setDate(nextRun.getDate() + 1);
      nextRun.setHours(2, 0, 0, 0); // Default to 2 AM
      return nextRun.toISOString();
    } catch (e) {
      return undefined;
    }
  }

  return undefined;
}

// GET /api/v1/protection-flows
export async function listFlows(): Promise<{ flows: ProtectionFlow[]; total: number }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/protection-flows`);
  return {
    flows: data.flows.map(transformFlowResponse),
    total: data.total
  };
}

// GET /api/v1/protection-flows/{id}
export async function getFlow(id: string): Promise<ProtectionFlow> {
  const { data } = await axios.get(`${API_BASE}/api/v1/protection-flows/${id}`);
  return transformFlowResponse(data);
}

// POST /api/v1/protection-flows
export async function createFlow(flow: any): Promise<ProtectionFlow> {
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

// GET flow machines (VMs in the flow with specs and backup stats)
export async function getFlowMachines(flowId: string): Promise<{
  machines: {
    context_id: string;
    vm_name: string;
    cpu_count: number;
    memory_mb: number;
    os_type: string;
    power_state: string;
    disks: { disk_id: string; size_gb: number }[];
    backup_stats: { backup_count: number; total_size_bytes: number; last_backup_at?: string };
  }[]
}> {
  // 1. Get flow details
  const flow = await getFlow(flowId);

  // 2. Get VMs (from group or single VM)
  let vms: any[] = [];
  if (flow.target_type === 'group') {
    const response = await axios.get(`${API_BASE}/api/v1/vm-groups/${flow.target_id}/members`);
    vms = response.data.members;
  } else {
    // Individual VM flow - get single VM context by context_id
    const response = await axios.get(`${API_BASE}/api/v1/vm-contexts/by-id/${flow.target_id}`);
    vms = [response.data.context]; // Extract the context from the response
  }

  // 3. Get disks and backup stats for each VM
  const enriched = await Promise.all(vms.map(async (vm: any) => {
    try {
      // Get disks
      const disksResponse = await axios.get(`${API_BASE}/api/v1/vm-contexts/${vm.context_id}/disks`);
      const disks = disksResponse.data.disks || [];

      // Get backup stats
      const statsResponse = await axios.get(`${API_BASE}/api/v1/backups/stats?vm_name=${vm.vm_name}&repository_id=${flow.repository_id}`);
      const backupStats = statsResponse.data;

      return {
        ...vm,
        disks,
        backup_stats: backupStats
      };
    } catch (error) {
      // Return VM with empty data if enrichment fails
      console.warn(`Failed to enrich VM ${vm.vm_name}:`, error);
      return {
        ...vm,
        disks: [],
        backup_stats: { backup_count: 0, total_size_bytes: 0 }
      };
    }
  }));

  return { machines: enriched };
}
