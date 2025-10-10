import { useQuery } from '@tanstack/react-query';
import axios from 'axios';
import * as api from '../api/protectionFlowsApi';

const API_BASE = '';

export interface FlowProgress {
  flowId: string;
  isRunning: boolean;
  progress: number;
  currentExecution?: {
    id: string;
    jobs_created: number;
    jobs_completed: number;
    jobs_failed: number;
  };
}

export function useFlowProgress(flowId: string, enabled: boolean = true) {
  return useQuery({
    queryKey: ['flow-progress', flowId],
    queryFn: async (): Promise<FlowProgress> => {
      // 1. Get latest execution for this flow
      const executionsResult = await api.getFlowExecutions(flowId);
      const latestExecution = executionsResult.executions?.[0];

      // 2. Check if running
      if (!latestExecution || latestExecution.status !== 'running') {
        return {
          flowId,
          isRunning: false,
          progress: 0,
        };
      }

      // 3. Calculate progress from execution job counts
      const jobsCreated = latestExecution.jobs_created || 0;
      const jobsCompleted = latestExecution.jobs_completed || 0;

      let progress = 0;
      if (jobsCreated > 0) {
        progress = Math.round((jobsCompleted / jobsCreated) * 100);
      }

      return {
        flowId,
        isRunning: true,
        progress: Math.min(progress, 99), // Never show 100% until execution completes
        currentExecution: {
          id: latestExecution.id,
          jobs_created: jobsCreated,
          jobs_completed: jobsCompleted,
          jobs_failed: latestExecution.jobs_failed || 0,
        },
      };
    },
    enabled: enabled && !!flowId,
    refetchInterval: (query) => {
      // Poll every 2 seconds if running, otherwise don't poll
      return query.state.data?.isRunning ? 2000 : false;
    },
    staleTime: 1000, // Consider data fresh for 1 second
  });
}

// Bulk progress hook for all flows in the table
export function useAllFlowsProgress(flowIds: string[], enabled: boolean = true) {
  return useQuery({
    queryKey: ['all-flows-progress', flowIds],
    queryFn: async (): Promise<Record<string, FlowProgress>> => {
      // Fetch progress for all flows in parallel
      const results = await Promise.allSettled(
        flowIds.map(async (flowId) => {
          try {
            const executionsResult = await api.getFlowExecutions(flowId);
            const latestExecution = executionsResult.executions?.[0];

            if (!latestExecution || latestExecution.status !== 'running') {
              return { flowId, isRunning: false, progress: 0 };
            }

            const jobsCreated = latestExecution.jobs_created || 0;
            const jobsCompleted = latestExecution.jobs_completed || 0;
            const progress = jobsCreated > 0
              ? Math.min(Math.round((jobsCompleted / jobsCreated) * 100), 99)
              : 0;

            return {
              flowId,
              isRunning: true,
              progress,
              currentExecution: {
                id: latestExecution.id,
                jobs_created: jobsCreated,
                jobs_completed: jobsCompleted,
                jobs_failed: latestExecution.jobs_failed || 0,
              },
            };
          } catch (error) {
            // Return default progress if API call fails
            return { flowId, isRunning: false, progress: 0 };
          }
        })
      );

      // Build record of flowId -> progress
      const progressMap: Record<string, FlowProgress> = {};
      results.forEach((result, index) => {
        if (result.status === 'fulfilled') {
          progressMap[flowIds[index]] = result.value;
        } else {
          // If failed, set default progress
          progressMap[flowIds[index]] = {
            flowId: flowIds[index],
            isRunning: false,
            progress: 0
          };
        }
      });

      return progressMap;
    },
    enabled: enabled && flowIds.length > 0,
    refetchInterval: 2000, // Poll every 2 seconds for real-time updates
    refetchOnMount: 'always',  // Always refetch on mount for immediate updates
    refetchOnWindowFocus: true,  // Refetch when window focused
    staleTime: 1000, // Consider data fresh for 1 second
  });
}
