import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as api from '../api/protectionFlowsApi';

// Use empty string to let Next.js rewrites proxy handle the routing
const API_BASE = '';

export function useProtectionFlows() {
  return useQuery({
    queryKey: ['protection-flows'],
    queryFn: api.listFlows,
    refetchInterval: 5000, // Refresh every 5 seconds for live updates
  });
}

export function useProtectionFlow(id: string) {
  return useQuery({
    queryKey: ['protection-flow', id],
    queryFn: () => api.getFlow(id),
    enabled: !!id,
  });
}

export function useFlowSummary() {
  return useQuery({
    queryKey: ['protection-flows', 'summary'],
    queryFn: api.getFlowSummary,
    refetchInterval: 10000, // Refresh every 10 seconds
  });
}

export function useFlowExecutions(flowId: string) {
  return useQuery({
    queryKey: ['protection-flow', flowId, 'executions'],
    queryFn: () => api.getFlowExecutions(flowId),
    enabled: !!flowId,
    refetchInterval: 3000, // Refresh every 3 seconds for active jobs
  });
}

export function useCreateFlow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.createFlow,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flows', 'summary'] });
    },
  });
}

export function useUpdateFlow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, flow }: { id: string; flow: Partial<api.ProtectionFlow> }) =>
      api.updateFlow(id, flow),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flow', variables.id] });
    },
  });
}

export function useDeleteFlow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.deleteFlow,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flows', 'summary'] });
    },
  });
}

export function useExecuteFlow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.executeFlow,
    onSuccess: (_, flowId) => {
      queryClient.invalidateQueries({ queryKey: ['protection-flow', flowId, 'executions'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flow', flowId] });
    },
  });
}

export function useEnableFlow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.enableFlow,
    onSuccess: (_, flowId) => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flow', flowId] });
    },
  });
}

export function useDisableFlow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.disableFlow,
    onSuccess: (_, flowId) => {
      queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      queryClient.invalidateQueries({ queryKey: ['protection-flow', flowId] });
    },
  });
}

export function useFlowMachines(flowId: string | null) {
  return useQuery({
    queryKey: ['protection-flow', flowId, 'machines'],
    queryFn: () => api.getFlowMachines(flowId!),
    enabled: !!flowId,
    staleTime: 30000, // Refresh every 30 seconds
  });
}

export function useMachineBackups(vmName: string | null, repositoryId: string) {
  return useQuery({
    queryKey: ['machine-backups', vmName, repositoryId],
    queryFn: async () => {
      if (!vmName) return null;

      const params = new URLSearchParams({
        vm_name: vmName,
        repository_id: repositoryId,
      });

      const response = await fetch(`${API_BASE}/api/v1/backups?${params.toString()}`);

      if (!response.ok) {
        throw new Error(`Failed to fetch backups: ${response.statusText}`);
      }

      const data = await response.json();
      return data.backups || [];
    },
    enabled: !!vmName && !!repositoryId,
    staleTime: 30000, // 30 seconds - reasonable for backup data
  });
}
