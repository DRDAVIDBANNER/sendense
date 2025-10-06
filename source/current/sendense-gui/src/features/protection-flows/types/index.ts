export type FlowType = 'backup' | 'replication';
export type FlowStatus = 'success' | 'running' | 'warning' | 'error' | 'pending';

export interface Flow {
  id: string;
  name: string;
  type: FlowType;
  status: FlowStatus;
  lastRun: string;
  nextRun: string;
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
  onEdit: (flow: Flow) => void;
  onDelete: (flow: Flow) => void;
  onRunNow: (flow: Flow) => void;
}
