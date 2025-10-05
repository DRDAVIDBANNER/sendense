// API Types for MigrateKit GUI
// Following our TypeScript best practices - 100% typed, no any types

// VM Context API Response Types (from our VM Context API)
export interface VMReplicationContext {
  context_id: string;
  vm_name: string;
  vmware_vm_id: string;
  vm_path: string;
  vcenter_host: string;
  datacenter: string;
  current_status: 'discovered' | 'replicating' | 'ready_for_failover' | 'failed_over_test' | 'failed_over_live' | 'completed' | 'failed' | 'cleanup_required';
  current_job_id?: string;
  total_jobs_run: number;
  successful_jobs: number;
  failed_jobs: number;
  last_successful_job_id?: string;
  cpu_count?: number;
  memory_mb?: number;
  os_type?: string;
  power_state?: string;
  vm_tools_version?: string;
  created_at: string;
  updated_at: string;
  first_job_at?: string;
  last_job_at?: string;
  last_status_change: string;
}

export interface ReplicationJob {
  id: string;
  vm_context_id: string;
  source_vm_id: string;
  source_vm_name: string;
  source_vm_path: string;
  vcenter_host: string;
  datacenter: string;
  replication_type: 'initial' | 'incremental';
  target_network: string;
  status: 'pending' | 'replicating' | 'completed' | 'failed' | 'stopped';
  progress_percent: number;
  current_operation: string;
  bytes_transferred: number;
  total_bytes: number;
  transfer_speed_bps: number;
  error_message: string;
  change_id: string;
  previous_change_id: string;
  snapshot_id: string;
  nbd_port: number;
  nbd_export_name: string;
  target_device: string;
  ossea_config_id: number;
  vma_sync_type: string;
  vma_current_phase: string;
  vma_throughput_mbps: number;
  vma_eta_seconds: number;
  vma_last_poll_at: string;
  vma_error_classification: string;
  vma_error_details: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface VMDisk {
  id: number;
  job_id: string;
  vm_context_id: string;
  disk_id: string;
  vmdk_path: string;
  size_gb: number;
  datastore: string;
  unit_number: number;
  label: string;
  capacity_bytes: number;
  provisioning_type: string;
  ossea_volume_id: number;
  cpu_count: number;
  memory_mb: number;
  os_type: string;
  vm_tools_version: string;
  network_config: string;
  display_name: string;
  annotation: string;
  power_state: string;
  vmware_uuid: string;
  bios_setup: string;
  disk_change_id: string;
  sync_status: string;
  sync_progress_percent: number;
  bytes_synced: number;
  created_at: string;
  updated_at: string;
}

export interface CBTHistory {
  id: number;
  job_id: string;
  vm_context_id: string;
  disk_id: string;
  change_id: string;
  previous_change_id: string;
  sync_type: 'initial' | 'incremental';
  blocks_changed: number;
  bytes_transferred: number;
  sync_duration_seconds: number;
  sync_success: boolean;
  created_at: string;
}

export interface VMContextDetails {
  context: VMReplicationContext;
  current_job?: ReplicationJob;
  job_history: ReplicationJob[];
  disks: VMDisk[];
  cbt_history: CBTHistory[];
}

export interface VMContextListResponse {
  vm_contexts: VMReplicationContext[];
  count: number;
}

// Legacy VM Discovery Types (VMA API)
export interface VM {
  id: string;
  name: string;
  path: string;
  datacenter: string;
  power_state: string;
  guest_os: string;
  memory_mb: number;
  num_cpu: number;
  vmx_version?: string;
  disks?: DiskInfo[];
  networks?: NetworkInfo[];
}

export interface DiskInfo {
  id: string;
  label: string;
  path: string;
  vmdk_path: string;
  size_gb: number;
  capacity_bytes: number;
  datastore: string;
  provisioning_type: string;
  unit_number: number;
}

export interface NetworkInfo {
  label: string;
  network_name: string;
  adapter_type: string;
  mac_address: string;
  connected: boolean;
}

export interface Migration {
  id: string;
  vm_name: string;
  status: string;
  started_at?: string;
  progress?: number;
  job_type?: string;
}

export interface FailoverJob {
  job_id: string;
  vm_id: string;
  vm_name: string;
  job_type: 'live' | 'test';
  status: string;
  progress: number;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  duration?: string;
  error_message?: string;
}

// Navigation Types
export type NavigationSection = 
  | 'dashboard' 
  | 'discovery' 
  | 'virtual-machines' 
  | 'replication-jobs' 
  | 'failover' 
  | 'network-mapping' 
  | 'logs' 
  | 'settings';

export interface NavigationItem {
  id: NavigationSection;
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
}

// UI State Types
export interface UIState {
  selectedVM: string | null;
  sidebarCollapsed: boolean;
  activePanel: 'progress' | 'history' | 'details';
  loading: boolean;
  error: string | null;
}

// API Response Types
export interface APIResponse<T> {
  data?: T;
  error?: string;
  success: boolean;
}

export interface ErrorResponse {
  error: string;
  timestamp: string;
}

// Form Types
export interface NetworkMappingForm {
  vm_id: string;
  source_network_name: string;
  destination_network_id: string;
  is_test_network: boolean;
}

export interface ReplicationForm {
  source_vm: VM;
  ossea_config_id: number;
  replication_type: 'initial' | 'incremental';
}

export interface FailoverForm {
  vm_id: string;
  vm_name: string;
  failover_type: 'live' | 'test';
  skip_validation: boolean;
  network_mappings: Record<string, string>;
  test_duration?: string;
  auto_cleanup?: boolean;
}

// Unified Jobs API Types (v2.32.0)
export type UnifiedJobType = 'replication' | 'test_failover' | 'live_failover' | 'rollback';
export type UnifiedJobStatus = 'running' | 'completed' | 'failed' | 'cancelled';
export type ErrorCategory = 'compatibility' | 'network' | 'storage' | 'platform' | 'connectivity' | 'configuration';
export type ErrorSeverity = 'info' | 'warning' | 'error' | 'critical';

export interface UnifiedJob {
  job_id: string;
  external_job_id?: string;
  job_type: UnifiedJobType;
  status: UnifiedJobStatus;
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

export interface UnifiedJobsResponse {
  context_id: string;
  count: number;
  jobs: UnifiedJob[];
}

export interface OperationSummary {
  job_id: string;
  external_job_id?: string;
  operation_type: string;
  status: UnifiedJobStatus;
  progress: number;
  failed_step?: string;
  failed_step_internal?: string;
  error_message?: string;
  error_category?: ErrorCategory;
  error_severity?: ErrorSeverity;
  actionable_steps?: string[];
  timestamp: string;
  duration_seconds: number;
  steps_completed?: number;
  steps_total?: number;
}

// Enhanced VMContextDetails with last_operation
export interface EnhancedVMContextDetails extends VMContextDetails {
  last_operation?: OperationSummary;
}

// ============================================================================
// BACKUP API TYPES (Task 5 Integration)
// ============================================================================

export interface BackupJob {
  backup_id: string;
  vm_context_id: string;
  vm_name: string;
  disk_id: number;
  backup_type: 'full' | 'incremental';
  repository_id: string;
  policy_id?: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  file_path?: string;
  nbd_export_name?: string;
  bytes_transferred: number;
  total_bytes: number;
  change_id?: string;
  error_message?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  tags?: Record<string, string>;
}

export interface BackupListResponse {
  backups: BackupJob[];
  total: number;
}

export interface BackupChainResponse {
  chain_id: string;
  vm_context_id: string;
  vm_name: string;
  disk_id: number;
  repository_id: string;
  full_backup_id: string;
  backups: BackupJob[];
  total_size_bytes: number;
  backup_count: number;
}

export interface StartBackupRequest {
  vm_name: string;
  disk_id: number;
  backup_type: 'full' | 'incremental';
  repository_id: string;
  policy_id?: string;
  tags?: Record<string, string>;
}

// ============================================================================
// FILE-LEVEL RESTORE API TYPES (Task 4 Integration)
// ============================================================================

export interface RestoreMount {
  mount_id: string;
  backup_id: string;
  mount_path: string;
  nbd_device: string;
  filesystem_type: string;
  status: 'mounting' | 'mounted' | 'unmounting' | 'failed';
  created_at: string;
  last_accessed_at: string;
  expires_at: string;
}

export interface RestoreMountsListResponse {
  mounts: RestoreMount[];
  count: number;
}

export interface FileInfo {
  name: string;
  path: string;
  type: 'file' | 'directory';
  size: number;
  modified: string;
  permissions: string;
}

export interface FileListResponse {
  files: FileInfo[];
  total_count: number;
}

export interface RestoreResourceStatus {
  active_mounts: number;
  max_mounts: number;
  available_slots: number;
  allocated_devices: string[];
  device_utilization: number;
}

export interface RestoreCleanupStatus {
  running: boolean;
  cleanup_interval: string;
  idle_timeout: string;
  active_mount_count: number;
  expired_mount_count: number;
}
