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
  FailoverForm
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
}

// Export singleton instance
export const api = new APIClient();

// Export class for testing
export { APIClient };
