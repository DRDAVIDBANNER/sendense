import axios from 'axios';
import type {
  RestoreMount,
  MountBackupRequest,
  FileListResponse,
  ActiveMountsResponse,
  VMContext,
  VMBackupsResponse
} from '../types';

const API_BASE = ''; // Uses Next.js proxy

// 1. Mount backup disk
export const mountBackup = async (request: MountBackupRequest): Promise<RestoreMount> => {
  const response = await axios.post(`${API_BASE}/api/v1/restore/mount`, request);

  if (response.status !== 200) {
    throw new Error(response.data?.error || 'Failed to mount backup');
  }

  return response.data;
};

// 2. Browse files
export const listFiles = async (mountId: string, path: string): Promise<FileListResponse> => {
  const response = await axios.get(
    `${API_BASE}/api/v1/restore/${mountId}/files?path=${encodeURIComponent(path)}`
  );

  if (response.status !== 200) {
    throw new Error('Failed to list files');
  }

  return response.data;
};

// 3. List active mounts
export const listActiveMounts = async (): Promise<ActiveMountsResponse> => {
  const response = await axios.get(`${API_BASE}/api/v1/restore/mounts`);

  if (response.status !== 200) {
    throw new Error('Failed to list active mounts');
  }

  return response.data;
};

// 4. Unmount backup
export const unmountBackup = async (mountId: string): Promise<void> => {
  const response = await axios.delete(`${API_BASE}/api/v1/restore/${mountId}`);

  if (response.status !== 200) {
    throw new Error('Failed to unmount backup');
  }
};

// 5. Get download URL (file)
export const getDownloadFileUrl = (mountId: string, path: string): string => {
  return `${API_BASE}/api/v1/restore/${mountId}/download?path=${encodeURIComponent(path)}`;
};

// 6. Get download URL (directory)
export const getDownloadDirectoryUrl = (mountId: string, path: string, format: 'zip' | 'tar.gz' = 'zip'): string => {
  return `${API_BASE}/api/v1/restore/${mountId}/download-directory?path=${encodeURIComponent(path)}&format=${format}`;
};

// 7. List VMs for backup selection
export const listVMContexts = async (): Promise<{ vm_contexts: VMContext[] }> => {
  const response = await axios.get(`${API_BASE}/api/v1/vm-contexts`);

  if (response.status !== 200) {
    throw new Error('Failed to list VMs');
  }

  return response.data;
};

// 8. Get backups for a specific VM
export const getVMBackups = async (vmName: string): Promise<VMBackupsResponse> => {
  const response = await axios.get(`${API_BASE}/api/v1/backups?vm_name=${encodeURIComponent(vmName)}&status=completed`);

  if (response.status !== 200) {
    throw new Error('Failed to get VM backups');
  }

  return response.data;
};
