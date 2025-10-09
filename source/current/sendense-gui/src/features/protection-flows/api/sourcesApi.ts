import axios from 'axios';

const API_BASE = '';  // Uses Next.js proxy

// Protection Group (Machine Group)
export interface ProtectionGroup {
  id: string;
  name: string;
  description: string;
  total_vms: number;
  enabled_vms: number;
  disabled_vms: number;
  created_at: string;
}

// VM Context
export interface VMContext {
  context_id: string;
  vm_name: string;
  vmware_vm_id: string;
  vm_path: string;
  vcenter_host: string;
  datacenter: string;
  current_status: string;
  os_type: string;
  power_state: string;
  groups?: Array<{
    group_id: string;
    group_name: string;
    enabled: boolean;
  }>;
}

// Repository
export interface Repository {
  id: string;
  name: string;
  type: string;  // 'local', 'nfs', 'cifs', 's3', 'azure'
  enabled: boolean;
  storage_info?: {
    total_bytes: number;
    available_bytes: number;
    used_percent: number;
    backup_count: number;
  };
}

// GET /api/v1/machine-groups
export async function listProtectionGroups(): Promise<{ groups: ProtectionGroup[]; total: number }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/machine-groups`);
  return data;
}

// GET /api/v1/vm-contexts
export async function listVMContexts(): Promise<{ vm_contexts: VMContext[] }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/vm-contexts`);
  return data;
}

// GET /api/v1/repositories
export async function listRepositories(): Promise<{ repositories: Repository[]; total: number }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/repositories`);
  return data;
}
