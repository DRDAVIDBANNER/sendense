export type FlowType = 'backup' | 'replication';
export type FlowStatus = 'success' | 'running' | 'warning' | 'error' | 'pending';

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
  created_at: string;
  updated_at: string;
  last_execution?: string;
  next_execution?: string;
  execution_count: number;
  success_count: number;
  failure_count: number;
  // UI-specific fields (computed from API data)
  status?: FlowStatus;
  lastRun?: string;
  nextRun?: string;
  source?: string;
  destination?: string;
  progress?: number;
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
