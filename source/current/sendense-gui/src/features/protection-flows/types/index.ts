export type FlowType = 'backup' | 'replication';
export type FlowStatus = 'success' | 'running' | 'warning' | 'error' | 'pending';

export interface FlowStatusData {
  last_execution_status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  total_executions: number;
  successful_executions: number;
  failed_executions: number;
}

export interface Flow {
  id: string;
  name: string;
  flow_type: 'backup' | 'replication';
  target_type: 'vm' | 'group';
  target_id: string;
  repository_id?: string;
  schedule_id?: string;
  policy_id?: string;
  enabled: boolean;
  status: FlowStatusData;
  created_at: string;
  updated_at: string;
  created_by?: string;
  last_execution?: string;
  next_execution?: string;
  // UI-specific fields (computed from API data)
  lastRun?: string;
  nextRun?: string;
  source?: string;
  destination?: string;
  progress?: number;
}

// Helper to get simple UI status from API status
export function getUIStatus(flow: Flow): FlowStatus {
  const status = flow.status?.last_execution_status;
  if (!status || status === 'pending') return 'pending';
  if (status === 'running') return 'running';
  if (status === 'completed') return 'success';
  if (status === 'failed') return 'error';
  if (status === 'cancelled') return 'warning';
  return 'pending';
}

export interface FlowsTableProps {
  flows: Flow[];
  onSelectFlow: (flow: Flow) => void;
  selectedFlowId?: string;
  onSort?: (column: string, direction: 'asc' | 'desc') => void;
  sortColumn?: string;
  sortDirection?: 'asc' | 'desc';
}

export interface FlowRowProps {
  flow: Flow;
  isSelected: boolean;
  onSelect: (flow: Flow) => void;
  onViewDetails?: (flow: Flow) => void;
  onEdit?: (flow: Flow) => void;
  onDelete?: (flow: Flow) => void;
  onRunNow?: (flow: Flow) => void;
}
