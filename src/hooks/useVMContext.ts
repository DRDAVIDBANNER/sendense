'use client';

import { useQuery } from '@tanstack/react-query';

export interface VMReplicationContext {
  id: string;
  source_vm_name: string;
  source_vm_path: string;
  datacenter: string;
  cpu_count: number;
  memory_mb: number;
  power_state: string;
  guest_os: string;
  created_at: string;
  updated_at: string;
}

export interface ReplicationJob {
  id: string;
  source_vm_name: string;
  status: string;
  replication_type: string;
  current_operation: string;
  progress_percentage: number;
  vma_sync_type: string;
  vma_eta_seconds: number;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface VMDisk {
  id: string;
  disk_label: string;
  disk_path: string;
  vmdk_path: string;
  size_gb: number;
  capacity_bytes: number;
  datastore: string;
  provisioning_type: string;
  unit_number: number;
  disk_change_id?: string;
}

export interface CBTHistory {
  id: string;
  change_id: string;
  captured_at: string;
  bytes_changed: number;
  operation_type: string;
}

export interface VMContextDetails {
  context: VMReplicationContext;
  current_job?: ReplicationJob;
  job_history: ReplicationJob[];
  disks: VMDisk[];
  cbt_history: CBTHistory[];
}

export interface VMContextListItem {
  vm_name: string;
  status: string;
  job_count: number;
  last_activity: string;
  current_job?: ReplicationJob;
}

export function useVMContext(vmName: string | null) {
  return useQuery<VMContextDetails>({
    queryKey: ['vmContext', vmName],
    queryFn: async () => {
      if (!vmName) {
        throw new Error('VM name is required');
      }
      
      const response = await fetch(`/api/vm-contexts/${encodeURIComponent(vmName)}`);
      
      if (!response.ok) {
        if (response.status === 404) {
          throw new Error(`VM context not found: ${vmName}`);
        }
        throw new Error(`Failed to fetch VM context: ${response.statusText}`);
      }
      
      return response.json();
    },
    enabled: !!vmName,
    refetchInterval: vmName ? 5000 : false, // Real-time updates for selected VM
    staleTime: 2000,
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
}

export function useVMContexts() {
  return useQuery<VMContextListItem[]>({
    queryKey: ['vmContexts'],
    queryFn: async () => {
      const response = await fetch('/api/vm-contexts');
      
      if (!response.ok) {
        throw new Error(`Failed to fetch VM contexts: ${response.statusText}`);
      }
      
      const data = await response.json();
      
      // API now returns transformed data
      return Array.isArray(data) ? data : [];
    },
    refetchInterval: 30000, // Background updates for all VMs
    staleTime: 15000,
    retry: 2,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10000),
  });
}