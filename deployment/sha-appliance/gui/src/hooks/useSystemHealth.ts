'use client';

import { useQuery } from '@tanstack/react-query';

export interface SystemHealth {
  active_jobs: number;
  vma_healthy: boolean;
  volume_daemon_healthy: boolean;
  oma_healthy: boolean;
  last_check: string;
  uptime_seconds: number;
}

export function useSystemHealth() {
  return useQuery<SystemHealth>({
    queryKey: ['systemHealth'],
    queryFn: async () => {
      const response = await fetch('/api/health');
      
      if (!response.ok) {
        throw new Error(`Failed to fetch system health: ${response.statusText}`);
      }
      
      return response.json();
    },
    refetchInterval: 60000, // Check health every minute
    staleTime: 30000,
    retry: 2,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10000),
  });
}










