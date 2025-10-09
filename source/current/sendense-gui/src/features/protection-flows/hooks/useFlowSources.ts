import { useQuery } from '@tanstack/react-query';
import * as api from '../api/sourcesApi';

export function useProtectionGroups() {
  return useQuery({
    queryKey: ['protection-groups'],
    queryFn: api.listProtectionGroups,
    staleTime: 30000, // 30 seconds - groups don't change often
  });
}

export function useVMContexts() {
  return useQuery({
    queryKey: ['vm-contexts'],
    queryFn: api.listVMContexts,
    staleTime: 10000, // 10 seconds - VMs change more frequently
  });
}

export function useRepositories() {
  return useQuery({
    queryKey: ['repositories'],
    queryFn: api.listRepositories,
    staleTime: 60000, // 60 seconds - repos are relatively static
  });
}
