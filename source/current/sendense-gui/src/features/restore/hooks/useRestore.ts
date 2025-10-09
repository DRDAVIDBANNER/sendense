import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as restoreApi from '../api/restoreApi';

// Fetch active mounts
export const useActiveMounts = () => {
  return useQuery({
    queryKey: ['active-mounts'],
    queryFn: restoreApi.listActiveMounts,
    refetchInterval: 30000, // Refresh every 30 seconds
  });
};

// Fetch files for a mount
export const useFiles = (mountId: string, path: string) => {
  return useQuery({
    queryKey: ['files', mountId, path],
    queryFn: () => restoreApi.listFiles(mountId, path),
    enabled: !!mountId, // Only fetch if mountId exists
  });
};

// Fetch VMs for backup selection
export const useVMContexts = () => {
  return useQuery({
    queryKey: ['vm-contexts'],
    queryFn: restoreApi.listVMContexts,
    staleTime: 60000, // VMs don't change often
  });
};

// Fetch backups for a specific VM
export const useVMBackups = (vmName: string) => {
  return useQuery({
    queryKey: ['vm-backups', vmName],
    queryFn: () => restoreApi.getVMBackups(vmName),
    enabled: !!vmName,
    staleTime: 30000, // Backup history refreshes moderately
  });
};

// Mount backup mutation
export const useMountBackup = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: restoreApi.mountBackup,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['active-mounts'] });
    },
  });
};

// Unmount backup mutation
export const useUnmountBackup = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: restoreApi.unmountBackup,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['active-mounts'] });
    },
  });
};
