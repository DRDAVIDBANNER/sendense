'use client';

import React, { useEffect, useState, useCallback } from 'react';
import { Card, Spinner, Button } from 'flowbite-react';
import { UnifiedJob, UnifiedJobsResponse } from '@/lib/types';
import { HiExclamationCircle, HiChevronDown, HiChevronRight, HiCheckCircle, HiClock, HiXCircle, HiLightningBolt } from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';

interface UnifiedJobListProps {
  vmName: string;
  onJobClick?: (job: UnifiedJob) => void;
}

export const UnifiedJobList = React.memo(({ vmName, onJobClick }: UnifiedJobListProps) => {
  const [jobs, setJobs] = useState<UnifiedJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadJobs = useCallback(async () => {
    console.log('🔍 UnifiedJobList loadJobs called with vmName:', vmName);
    
    if (!vmName) {
      console.log('❌ No vmName provided to UnifiedJobList');
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);
      
      console.log('📡 Fetching unified jobs from:', `/api/vm-contexts/${vmName}/recent-jobs`);
      
      const response = await fetch(`/api/vm-contexts/${vmName}/recent-jobs`);
      
      console.log('📡 UnifiedJobList API response status:', response.status);

      if (!response.ok) {
        throw new Error(`Failed to fetch recent jobs: ${response.statusText}`);
      }

      const data: UnifiedJobsResponse = await response.json();
      console.log('✅ UnifiedJobList data received:', data);
      console.log('✅ Jobs count:', data.jobs?.length || 0);
      setJobs(data.jobs || []);
    } catch (err) {
      console.error('❌ Error loading unified jobs:', err);
      setError(err instanceof Error ? err.message : 'Failed to load recent operations');
    } finally {
      setLoading(false);
      console.log('🏁 UnifiedJobList loading complete, jobs:', jobs.length);
    }
  }, [vmName]);

  useEffect(() => {
    loadJobs();
  }, [loadJobs]);

  if (loading) {
    return (
      <Card>
        <div className="flex items-center justify-center p-8">
          <Spinner size="lg" />
          <span className="ml-3 text-gray-600 dark:text-gray-400">
            Loading recent operations...
          </span>
        </div>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <div className="text-center p-4">
          <ClientIcon className="w-12 h-12 text-red-500 mx-auto mb-2">
            <HiExclamationCircle />
          </ClientIcon>
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
            Error Loading Operations
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {error}
          </p>
        </div>
      </Card>
    );
  }

  if (jobs.length === 0) {
    return (
      <Card>
        <div className="text-center p-8">
          <p className="text-gray-500 dark:text-gray-400">
            No recent operations
          </p>
        </div>
      </Card>
    );
  }

  return (
    <Card>
      <h4 className="text-md font-medium text-gray-900 dark:text-white mb-3">
        Recent Operations
      </h4>
      
      {jobs.length === 0 ? (
        <p className="text-sm text-gray-500 text-center py-4">
          No recent operations
        </p>
      ) : (
        <div className="space-y-2">
          {jobs.map((job) => (
            <UnifiedJobCard 
              key={job.job_id} 
              job={job}
            />
          ))}
        </div>
      )}
    </Card>
  );
});

UnifiedJobList.displayName = 'UnifiedJobList';

interface UnifiedJobCardProps {
  job: UnifiedJob;
}

const UnifiedJobCard = React.memo(({ job }: UnifiedJobCardProps) => {
  const [expanded, setExpanded] = useState(false);
  
  const getStatusIcon = () => {
    switch (job.status) {
      case 'completed':
        return HiCheckCircle;
      case 'failed':
        return HiXCircle;
      case 'running':
        return HiClock;
      default:
        return HiClock;
    }
  };

  const getStatusIconColor = () => {
    switch (job.status) {
      case 'completed':
        return 'text-green-500';
      case 'failed':
        return 'text-red-500';
      case 'running':
        return 'text-yellow-500';
      default:
        return 'text-gray-500';
    }
  };

  const StatusIcon = getStatusIcon();
  const iconColor = getStatusIconColor();

  const formatTimestamp = (ts: string): string => {
    const date = new Date(ts);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins} min ago`;
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)} hours ago`;
    return `${Math.floor(diffMins / 1440)} days ago`;
  };

  const formatDuration = (seconds: number): string => {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    const hours = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${mins}m`;
  };

  // Compact view matching original design
  return (
    <div className={`p-2 bg-gray-50 dark:bg-gray-700 rounded ${job.status === 'failed' && !expanded ? 'hover:bg-gray-100 dark:hover:bg-gray-600 cursor-pointer' : ''}`}>
      {/* Compact Header - Always Visible */}
      <div 
        className="flex items-center justify-between"
        onClick={() => job.status === 'failed' && setExpanded(!expanded)}
      >
        <div className="flex items-center space-x-2 flex-1">
          <ClientIcon className={`w-4 h-4 ${iconColor}`}>
            <StatusIcon />
          </ClientIcon>
          <div className="flex-1">
            <p className="text-sm font-medium text-gray-900 dark:text-white">
              {job.display_name}
              {job.status === 'failed' && ` - Failed (${job.progress.toFixed(0)}%)`}
            </p>
            <p className="text-xs text-gray-500">
              {formatTimestamp(job.started_at)}
              {job.duration_seconds && ` • ${formatDuration(job.duration_seconds)}`}
            </p>
          </div>
        </div>
        
        {/* Status badge and expand icon for failed jobs */}
        <div className="flex items-center gap-2">
          {job.status === 'failed' && (
            <ClientIcon className="w-4 h-4 text-gray-400">
              {expanded ? <HiChevronDown /> : <HiChevronRight />}
            </ClientIcon>
          )}
        </div>
      </div>

      {/* Expanded Details - Only for Failed Jobs */}
      {job.status === 'failed' && expanded && job.error_message && (
        <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600 space-y-3">
          {/* Sanitized Error Message */}
          <div className="text-sm">
            <span className="text-gray-500 dark:text-gray-400">Issue: </span>
            <span className="text-red-600 dark:text-red-400">{job.error_message}</span>
          </div>

          {/* Actionable Steps */}
          {job.actionable_steps && job.actionable_steps.length > 0 && (
            <div>
              <div className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                What you can do:
              </div>
              <ul className="text-xs text-gray-600 dark:text-gray-300 space-y-1 ml-4">
                {job.actionable_steps.map((step, i) => (
                  <li key={i} className="flex items-start gap-1">
                    <span className="text-red-500">•</span>
                    <span>{step}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Quick Action - Try Live Failover if suggested */}
          {job.actionable_steps?.some(s => s.toLowerCase().includes('live failover')) && (
            <Button
              size="xs"
              color="failure"
              className="w-full"
              onClick={(e) => {
                e.stopPropagation();
                // TODO: Trigger live failover
                console.log('Try live failover for', job.job_id);
              }}
            >
              <ClientIcon className="w-3 h-3 mr-1">
                <HiLightningBolt />
              </ClientIcon>
              Try Live Failover
            </Button>
          )}
        </div>
      )}
    </div>
  );
});

UnifiedJobCard.displayName = 'UnifiedJobCard';

