export type FlowType = 'backup' | 'replication';
export type FlowStatus = 'success' | 'running' | 'warning' | 'error' | 'pending';

export interface FlowStatusData {
  last_execution_status: 'pending' | 'running' | 'completed' | 'success' | 'failed' | 'cancelled';  // ✅ FIX: Add 'success' type
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
  schedule_name?: string;
  schedule_cron?: string;
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
  if (status === 'completed' || status === 'success') return 'success';  // ✅ FIX: API returns 'success' not 'completed'
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
  onEdit?: (flow: Flow) => void;      // Optional: for opening edit modal
  onDelete?: (flow: Flow) => void;    // Optional: for opening delete modal
  // Note: onRunNow is NOT here - FlowsTable uses its own with optimistic UI
}

export interface FlowRowProps {
  flow: Flow & { isOptimisticallyRunning?: boolean };
  isSelected: boolean;
  onSelect: (flow: Flow) => void;
  onViewDetails?: (flow: Flow) => void;
  onEdit?: (flow: Flow) => void;
  onDelete?: (flow: Flow) => void;
  onRunNow?: (flow: Flow) => void;
}

// Flow Machines Panel types
export interface FlowMachineInfo {
  context_id: string;
  vm_name: string;
  cpu_count: number;
  memory_mb: number;
  os_type: string;
  power_state: string;
  disks: VMDiskInfo[];
  backup_stats: VMBackupStats;
}

export interface VMDiskInfo {
  disk_id: string;
  size_gb: number;
}

export interface VMBackupStats {
  backup_count: number;
  total_size_bytes: number;
  last_backup_at?: string;
}
