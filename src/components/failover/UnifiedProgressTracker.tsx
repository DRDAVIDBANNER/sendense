'use client';

import React, { useState, useEffect, useRef } from 'react';
import { Card, Progress, Badge, Alert, Button } from 'flowbite-react';
import { 
  HiClock, 
  HiCheckCircle, 
  HiExclamationCircle, 
  HiRefresh,
  HiLightningBolt,
  HiBeaker,
  HiArrowLeft
} from 'react-icons/hi';

export interface UnifiedProgressTrackerProps {
  jobId: string | null;
  failoverType: 'live' | 'test';
  vmName: string;
  onComplete?: (success: boolean) => void;
  onRollbackRequest?: () => void;
}

interface ProgressPhase {
  phase: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  description: string;
  started_at?: string;
  completed_at?: string;
  error?: string;
}

interface UnifiedProgressData {
  job_id: string;
  status: string;
  progress: number;
  current_phase: string;
  phases: ProgressPhase[];
  estimated_completion?: string;
  elapsed_time?: string;
  metadata?: {
    failover_type: string;
    vm_name: string;
    configuration_summary?: string;
  };
}

export const UnifiedProgressTracker: React.FC<UnifiedProgressTrackerProps> = ({
  jobId,
  failoverType,
  vmName,
  onComplete,
  onRollbackRequest
}) => {
  // Detect if this is a rollback job from the job ID
  const isRollbackJob = jobId?.startsWith('rollback-');
  const actualFailoverType = isRollbackJob ? 'rollback' : failoverType;
  const [progressData, setProgressData] = useState<UnifiedProgressData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>('');
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());
  const isCompleteRef = useRef(false);

  // Poll for progress updates
  useEffect(() => {
    if (!jobId) return;

    // Reset completion status for new job
    isCompleteRef.current = false;
    let intervalId: NodeJS.Timeout | null = null;

    const pollProgress = async () => {
      try {
        setIsLoading(true);
        const response = await fetch(`/api/failover/progress/${jobId}`);
        const data = await response.json();
        
        if (data.success) {
          setProgressData(data.progress);
          setLastUpdate(new Date());
          setError('');
          
          // Check if job is complete
          if (data.progress.status === 'completed' || data.progress.status === 'failed') {
            // Stop polling when job is complete
            isCompleteRef.current = true;
            if (intervalId) {
              clearInterval(intervalId);
              intervalId = null;
            }
            onComplete?.(data.progress.status === 'completed');
          }
        } else {
          setError(data.message || 'Failed to get progress');
        }
      } catch (err) {
        setError('Network error getting progress');
        console.error('Progress polling error:', err);
      } finally {
        setIsLoading(false);
      }
    };

    // Initial poll
    pollProgress();
    
    // Set up polling interval (every 5 seconds) only if job is not complete
    intervalId = setInterval(() => {
      // Double-check completion status before polling
      if (isCompleteRef.current) {
        if (intervalId) {
          clearInterval(intervalId);
          intervalId = null;
        }
        return;
      }
      pollProgress();
    }, 5000);
    
    return () => {
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [jobId, onComplete]);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <HiCheckCircle className="w-5 h-5 text-green-500" />;
      case 'failed':
        return <HiExclamationCircle className="w-5 h-5 text-red-500" />;
      case 'running':
        return <HiClock className="w-5 h-5 text-blue-500 animate-spin" />;
      default:
        return <HiClock className="w-5 h-5 text-gray-500" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'success';
      case 'failed':
        return 'failure';
      case 'running':
        return 'warning';
      default:
        return 'gray';
    }
  };

  const getPhaseIcon = (phase: string) => {
    if (phase.includes('validation')) return <HiCheckCircle className="w-4 h-4" />;
    if (phase.includes('power')) return <HiLightningBolt className="w-4 h-4" />;
    if (phase.includes('sync')) return <HiRefresh className="w-4 h-4" />;
    if (phase.includes('snapshot')) return <HiBeaker className="w-4 h-4" />;
    if (phase.includes('vm')) return <HiLightningBolt className="w-4 h-4" />;
    return <HiClock className="w-4 h-4" />;
  };

  const formatDuration = (startTime: string, endTime?: string) => {
    const start = new Date(startTime);
    const end = endTime ? new Date(endTime) : new Date();
    const diff = end.getTime() - start.getTime();
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) {
      return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  };

  if (!jobId) {
    return (
      <Card>
        <div className="text-center py-8">
          <HiClock className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <p className="text-gray-500">No active failover job to track</p>
        </div>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {/* Main Progress Card */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center">
            {isRollbackJob ? (
              <HiArrowLeft className="w-6 h-6 text-orange-500 mr-2" />
            ) : failoverType === 'live' ? (
              <HiLightningBolt className="w-6 h-6 text-red-500 mr-2" />
            ) : (
              <HiBeaker className="w-6 h-6 text-purple-500 mr-2" />
            )}
            <h3 className="text-lg font-semibold">
              {isRollbackJob 
                ? `${failoverType.charAt(0).toUpperCase() + failoverType.slice(1)} Rollback Progress`
                : `${failoverType.charAt(0).toUpperCase() + failoverType.slice(1)} Failover Progress`
              }
            </h3>
          </div>
          <div className="flex items-center space-x-2">
            {progressData && (
              <Badge color={getStatusColor(progressData.status)}>
                {progressData.status.toUpperCase()}
              </Badge>
            )}
            {getStatusIcon(progressData?.status || 'pending')}
          </div>
        </div>

        {/* VM Information */}
        <div className="grid grid-cols-2 gap-4 mb-4 text-sm">
          <div>
            <span className="font-medium">VM Name:</span> {vmName}
          </div>
          <div>
            <span className="font-medium">Job ID:</span> {jobId}
          </div>
          <div>
            <span className="font-medium">Type:</span> {isRollbackJob ? `${failoverType} rollback` : failoverType}
          </div>
          <div>
            <span className="font-medium">Last Update:</span> {lastUpdate.toLocaleTimeString()}
          </div>
        </div>

        {/* Overall Progress */}
        {progressData && (
          <div className="mb-4">
            <div className="flex justify-between mb-2">
              <span className="text-sm font-medium">Overall Progress</span>
              <span className="text-sm text-gray-500">{Math.round(progressData.progress)}%</span>
            </div>
            {/* Custom progress bar as fallback */}
            <div className="w-full bg-gray-200 rounded-full h-3 mb-2">
              <div 
                className="bg-green-500 h-3 rounded-full transition-all duration-300 ease-in-out"
                style={{ width: `${Math.min(100, Math.max(0, Math.round(progressData.progress)))}%` }}
              ></div>
            </div>
            {progressData.current_phase && (
              <p className="text-sm text-gray-600 mt-2">
                Current Phase: {progressData.current_phase}
              </p>
            )}
          </div>
        )}

        {/* Error Display */}
        {error && (
          <Alert color="failure" icon={HiExclamationCircle} className="mb-4">
            <span className="font-medium">Error:</span> {error}
          </Alert>
        )}
      </Card>

      {/* Phase Details */}
      {progressData?.phases && progressData.phases.length > 0 && (
        <Card>
          <h4 className="text-lg font-semibold mb-4">Phase Details</h4>
          <div className="space-y-3">
            {progressData.phases.map((phase, index) => (
              <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                <div className="flex items-center">
                  <div className={`mr-3 ${phase.status === 'running' ? 'animate-pulse' : ''}`}>
                    {getPhaseIcon(phase.phase)}
                  </div>
                  <div>
                    <p className="font-medium">{phase.phase}</p>
                    <p className="text-sm text-gray-600">{phase.description}</p>
                    {phase.error && (
                      <p className="text-sm text-red-600 mt-1">Error: {phase.error}</p>
                    )}
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  {phase.started_at && (
                    <span className="text-xs text-gray-500">
                      {formatDuration(phase.started_at, phase.completed_at)}
                    </span>
                  )}
                  <Badge color={getStatusColor(phase.status)} size="sm">
                    {phase.status}
                  </Badge>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Action Buttons */}
      {progressData?.status === 'failed' && onRollbackRequest && (
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <h4 className="font-semibold text-red-600">Failover Failed</h4>
              <p className="text-sm text-gray-600">
                The {failoverType} failover has failed. You can initiate a rollback to restore the previous state.
              </p>
            </div>
            <Button color="warning" onClick={onRollbackRequest}>
              <HiArrowLeft className="w-4 h-4 mr-2" />
              Initiate Rollback
            </Button>
          </div>
        </Card>
      )}

      {/* Metadata */}
      {progressData?.metadata && (
        <Card>
          <h4 className="text-lg font-semibold mb-2">Configuration Summary</h4>
          <div className="text-sm space-y-1">
            {progressData.metadata.configuration_summary && (
              <p>{progressData.metadata.configuration_summary}</p>
            )}
            {progressData.estimated_completion && (
              <p>
                <span className="font-medium">Estimated Completion:</span> {progressData.estimated_completion}
              </p>
            )}
            {progressData.elapsed_time && (
              <p>
                <span className="font-medium">Elapsed Time:</span> {progressData.elapsed_time}
              </p>
            )}
          </div>
        </Card>
      )}
    </div>
  );
};
