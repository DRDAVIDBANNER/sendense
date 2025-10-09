// Restore Mount
export interface RestoreMount {
  mount_id: string;
  backup_id: string;
  backup_disk_id: number;
  disk_index: number;
  mount_path: string;
  nbd_device: string;
  filesystem_type: string;
  status: 'mounting' | 'mounted' | 'unmounting' | 'failed' | 'unmounted';
  created_at: string;
  expires_at: string;
  last_accessed_at: string;
  partition_metadata?: string;
  error_message?: string;
}

// Mount Request
export interface MountBackupRequest {
  backup_id: string;
  disk_index: number;
}

// File Info
export interface FileInfo {
  name: string;
  path: string;
  type: 'file' | 'directory';
  size: number;
  mode: string;
  modified_time: string;
  is_symlink: boolean;
}

// File List Response
export interface FileListResponse {
  mount_id: string;
  path: string;
  files: FileInfo[];
  total_count: number;
}

// Active Mounts Response
export interface ActiveMountsResponse {
  mounts: RestoreMount[];
  count: number;
}

// VM Context (for backup selection)
export interface VMContext {
  context_id: string;
  vm_name: string;
  vmware_vm_id: string;
  vcenter_host: string;
  power_state: string;
  os_type: string;
  groups?: Array<{
    group_id: string;
    group_name: string;
    enabled: boolean;
  }>;
}

// Backup Job (for backup history)
export interface BackupJob {
  backup_id: string;
  vm_name: string;
  backup_type: string;
  status: string;
  repository_id: string;
  total_bytes: number;        // ✅ Correct field name
  bytes_transferred: number;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  disks_count: number;        // ✅ Already included
}

// Backup Disk (for disk selection)
export interface BackupDisk {
  id: number;
  backup_job_id: string;
  disk_index: number;
  original_path: string;
  size_bytes: number;
  qcow2_path: string;
  created_at: string;
}

// VM Backups Response
export interface VMBackupsResponse {
  vm_context: VMContext;
  backups: BackupJob[];
}
