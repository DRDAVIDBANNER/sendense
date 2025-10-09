import axios from 'axios';

// Use empty string to let Next.js rewrites proxy handle the routing
// next.config.ts proxies /api/v1/* to http://localhost:8082/api/v1/*
const API_BASE = '';

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
