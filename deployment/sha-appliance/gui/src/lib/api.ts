// API Client for MigrateKit GUI
// Following our best practices: type-safe, error handling, no any types

import { 
  VMContextDetails, 
  VMContextListResponse, 
  VM, 
  Migration, 
  FailoverJob,
  APIResponse,
  NetworkMappingForm,
  ReplicationForm,
  FailoverForm,
  BackupJob,
  BackupListResponse,
  BackupChainResponse,
  StartBackupRequest,
  RestoreMount,
  RestoreMountsListResponse,
  FileInfo,
  FileListResponse,
  RestoreResourceStatus,
  RestoreCleanupStatus
} from './types';

class APIClient {
  private baseURL: string;

  constructor() {
    // API endpoints - GUI proxies to OMA API
    this.baseURL = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:3001';
  }

  // VM Context API (New - Using our VM Context endpoints)
  async getVMContexts(): Promise<VMContextListResponse> {
    const response = await fetch(`${this.baseURL}/api/vm-contexts`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch VM contexts: ${response.statusText}`);
    }
    
    return response.json();
  }

  async getVMContext(vmName: string): Promise<VMContextDetails> {
    const response = await fetch(`${this.baseURL}/api/vm-contexts/${encodeURIComponent(vmName)}`);
    
    if (!response.ok) {
      if (response.status === 404) {
        throw new Error(`VM context not found: ${vmName}`);
      }
      throw new Error(`Failed to fetch VM context: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Enhanced Discovery API (OMA)
  async discoverVMs(params: {
    credential_id: number;
    filter?: string;
    create_context?: boolean;
  }): Promise<{ discovered_vms: VM[] }> {
    const response = await fetch(`${this.baseURL}/api/discovery/discover-vms`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        credential_id: params.credential_id,
        filter: params.filter,
        create_context: params.create_context || false
      })
    });
    
    if (!response.ok) {
      throw new Error(`Failed to discover VMs: ${response.statusText}`);
    }
    
    return response.json();
  }

  // Migration Management
  async getMigrations(): Promise<Migration[]> {
    const response = await fetch(`${this.baseURL}/api/migrations`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch migrations: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data || [];
  }

  async startMigration(form: ReplicationForm): Promise<APIResponse<{ job_id: string; status: string; started_at?: string; message?: string }>> {
    const response = await fetch(`${this.baseURL}/api/replicate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form)
    });
    
    const data = await response.json();
    
    if (!response.ok) {
      return { success: false, error: data.error || 'Failed to start migration' };
    }
    
    return { success: true, data };
  }

  // Failover Management
  async getFailoverJobs(): Promise<FailoverJob[]> {
    const response = await fetch(`${this.baseURL}/api/failover`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch failover jobs: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.jobs || [];
  }

  async startFailover(form: FailoverForm): Promise<APIResponse<{ job_id: string }>> {
    const response = await fetch(`${this.baseURL}/api/failover`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form)
    });
    
    const data = await response.json();
    
    if (!response.ok) {
      return { success: false, error: data.error || 'Failed to start failover' };
    }
    
    return { success: true, data };
  }

  async cleanupFailover(vmName: string): Promise<APIResponse<{ message: string }>> {
    const response = await fetch(`${this.baseURL}/api/cleanup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ vm_name: vmName })
    });
    
    const data = await response.json();
    
    if (!response.ok) {
      return { success: false, error: data.error || 'Failed to cleanup' };
    }
    
    return { success: true, data };
  }

  // Network Mapping Management
  async getNetworkMappings(vmId: string): Promise<{ mappings: NetworkMappingForm[] }> {
    const response = await fetch(`${this.baseURL}/api/network-mappings?vm_id=${encodeURIComponent(vmId)}`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch network mappings: ${response.statusText}`);
    }
    
    const data = await response.json();
    return { mappings: data.mappings || [] };
  }

  async saveNetworkMapping(mapping: NetworkMappingForm): Promise<APIResponse<{ message: string }>> {
    const response = await fetch(`${this.baseURL}/api/network-mappings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(mapping)
    });
    
    const data = await response.json();
    
    if (!response.ok) {
      return { success: false, error: data.error || 'Failed to save network mapping' };
    }
    
    return { success: true, data };
  }

  async getAvailableNetworks(): Promise<{ networks: Array<{ id: string; name: string; type: string }> }> {
    const response = await fetch(`${this.baseURL}/api/networks`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch available networks: ${response.statusText}`);
    }
    
    return response.json();
  }

  // CloudStack Validation API
  async testCloudStackConnection(credentials: {
    api_url: string;
    api_key: string;
    secret_key: string;
  }): Promise<{ success: boolean; message: string; error?: string }> {
    const response = await fetch(`${this.baseURL}/api/cloudstack/test-connection`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials)
    });
    
    return response.json();
  }

  async detectOMAVM(credentials: {
    api_url: string;
    api_key: string;
    secret_key: string;
  }): Promise<{
    success: boolean;
    oma_info?: {
      vm_id: string;
      vm_name: string;
      mac_address: string;
      ip_address: string;
      account: string;
    };
    message: string;
    error?: string;
  }> {
    const response = await fetch(`${this.baseURL}/api/cloudstack/detect-oma-vm`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials)
    });
    
    return response.json();
  }

  async getCloudStackNetworks(): Promise<{
    success: boolean;
    networks: Array<{
      id: string;
      name: string;
      zone_id: string;
      zone_name: string;
      state: string;
    }>;
    count: number;
    error?: string;
  }> {
    const response = await fetch(`${this.baseURL}/api/cloudstack/networks`);
    
    return response.json();
  }

  async validateCloudStackSettings(config: {
    api_url: string;
    api_key: string;
    secret_key: string;
    oma_vm_id?: string;
    service_offering_id?: string;
    network_id?: string;
  }): Promise<{
    success: boolean;
    result: {
      oma_vm_detection: { status: string; message: string; details?: any };
      compute_offering: { status: string; message: string; details?: any };
      account_match: { status: string; message: string; details?: any };
      network_selection: { status: string; message: string; details?: any };
      overall_status: 'pass' | 'warning' | 'fail';
    };
    message: string;
  }> {
    const response = await fetch(`${this.baseURL}/api/cloudstack/validate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    });
    
    return response.json();
  }

  // Error handling helper
  private handleError(error: unknown): never {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error('An unknown error occurred');
  }

  // ============================================================================
  // BACKUP API METHODS (Task 5 Integration)
  // ============================================================================

  async listBackups(params?: {
    vm_name?: string;
    vm_context_id?: string;
    repository_id?: string;
    status?: string;
    backup_type?: string;
  }): Promise<BackupListResponse> {
    const searchParams = new URLSearchParams();
    if (params?.vm_name) searchParams.append('vm_name', params.vm_name);
    if (params?.vm_context_id) searchParams.append('vm_context_id', params.vm_context_id);
    if (params?.repository_id) searchParams.append('repository_id', params.repository_id);
    if (params?.status) searchParams.append('status', params.status);
    if (params?.backup_type) searchParams.append('backup_type', params.backup_type);

    const url = searchParams.toString() 
      ? `${this.baseURL}/api/v1/backup/list?${searchParams}`
      : `${this.baseURL}/api/v1/backup/list`;

    const response = await fetch(url);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch backups: ${response.statusText}`);
    }
    
    return response.json();
  }

  async getBackupDetails(backupId: string): Promise<BackupJob> {
    const response = await fetch(`${this.baseURL}/api/v1/backup/${encodeURIComponent(backupId)}`);
    
    if (!response.ok) {
      if (response.status === 404) {
        throw new Error(`Backup not found: ${backupId}`);
      }
      throw new Error(`Failed to fetch backup details: ${response.statusText}`);
    }
    
    return response.json();
  }

  async startBackup(request: StartBackupRequest): Promise<BackupJob> {
    const response = await fetch(`${this.baseURL}/api/v1/backup/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request)
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to start backup');
    }
    
    return response.json();
  }

  async deleteBackup(backupId: string): Promise<{ message: string; backup_id: string }> {
    const response = await fetch(`${this.baseURL}/api/v1/backup/${encodeURIComponent(backupId)}`, {
      method: 'DELETE'
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to delete backup');
    }
    
    return response.json();
  }

  async getBackupChain(params: {
    vm_context_id?: string;
    vm_name?: string;
    disk_id?: number;
  }): Promise<BackupChainResponse> {
    const searchParams = new URLSearchParams();
    if (params.vm_context_id) searchParams.append('vm_context_id', params.vm_context_id);
    if (params.vm_name) searchParams.append('vm_name', params.vm_name);
    if (params.disk_id !== undefined) searchParams.append('disk_id', params.disk_id.toString());

    const response = await fetch(`${this.baseURL}/api/v1/backup/chain?${searchParams}`);
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch backup chain');
    }
    
    return response.json();
  }

  // ============================================================================
  // FILE-LEVEL RESTORE API METHODS (Task 4 Integration)
  // ============================================================================

  async mountBackup(backupId: string): Promise<RestoreMount> {
    const response = await fetch(`${this.baseURL}/api/v1/restore/mount`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ backup_id: backupId })
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to mount backup');
    }
    
    return response.json();
  }

  async unmountBackup(mountId: string): Promise<{ message: string }> {
    const response = await fetch(`${this.baseURL}/api/v1/restore/${encodeURIComponent(mountId)}`, {
      method: 'DELETE'
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to unmount backup');
    }
    
    return response.json();
  }

  async listRestoreMounts(): Promise<RestoreMountsListResponse> {
    const response = await fetch(`${this.baseURL}/api/v1/restore/mounts`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch restore mounts: ${response.statusText}`);
    }
    
    return response.json();
  }

  async listFiles(mountId: string, path: string = '/', recursive: boolean = false): Promise<FileListResponse> {
    const searchParams = new URLSearchParams();
    searchParams.append('path', path);
    if (recursive) searchParams.append('recursive', 'true');

    const response = await fetch(
      `${this.baseURL}/api/v1/restore/${encodeURIComponent(mountId)}/files?${searchParams}`
    );
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to list files');
    }
    
    return response.json();
  }

  async getFileInfo(mountId: string, path: string): Promise<FileInfo> {
    const searchParams = new URLSearchParams();
    searchParams.append('path', path);

    const response = await fetch(
      `${this.baseURL}/api/v1/restore/${encodeURIComponent(mountId)}/file-info?${searchParams}`
    );
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to get file info');
    }
    
    return response.json();
  }

  getDownloadFileUrl(mountId: string, path: string): string {
    const searchParams = new URLSearchParams();
    searchParams.append('path', path);
    return `${this.baseURL}/api/v1/restore/${encodeURIComponent(mountId)}/download?${searchParams}`;
  }

  getDownloadDirectoryUrl(mountId: string, path: string, format: 'zip' | 'tar.gz' = 'zip'): string {
    const searchParams = new URLSearchParams();
    searchParams.append('path', path);
    searchParams.append('format', format);
    return `${this.baseURL}/api/v1/restore/${encodeURIComponent(mountId)}/download-directory?${searchParams}`;
  }

  async getRestoreResourceStatus(): Promise<RestoreResourceStatus> {
    const response = await fetch(`${this.baseURL}/api/v1/restore/resources`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch restore resource status: ${response.statusText}`);
    }
    
    return response.json();
  }

  async getRestoreCleanupStatus(): Promise<RestoreCleanupStatus> {
    const response = await fetch(`${this.baseURL}/api/v1/restore/cleanup-status`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch restore cleanup status: ${response.statusText}`);
    }
    
    return response.json();
  }
}

// Export singleton instance
export const api = new APIClient();

// Export class for testing
export { APIClient };
