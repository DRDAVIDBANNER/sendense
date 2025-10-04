'use client';

import React from 'react';
import { Card, Badge, Alert } from 'flowbite-react';
import { useActiveJobProgress } from '../../hooks/useRealTimeUpdates';

export function RealTimeJobProgress() {
  const { activeJobs, isConnected, error } = useActiveJobProgress();

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed':
        return 'success';
      case 'failed':
        return 'failure';
      case 'replicating':
        return 'warning';
      case 'pending':
        return 'gray';
      default:
        return 'gray';
    }
  };

  const getProgressColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed':
        return 'bg-green-600';
      case 'failed':
        return 'bg-red-600';
      case 'replicating':
        return 'bg-blue-600';
      default:
        return 'bg-gray-600';
    }
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white">
          🚀 Live Migration Progress
        </h3>
        <div className="flex items-center space-x-2">
          <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}></div>
          <Badge color={isConnected ? 'success' : 'failure'} size="sm">
            {isConnected ? 'Live' : 'Offline'}
          </Badge>
          {activeJobs.length > 0 && (
            <Badge color="blue" size="sm">
              {activeJobs.length} Active
            </Badge>
          )}
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <Alert color="warning" className="text-sm">
          {error}
        </Alert>
      )}

      {/* Active Jobs */}
      {activeJobs.length > 0 ? (
        <div className="space-y-3">
          {activeJobs.map((job) => (
            <Card key={job.id}>
              <div className="space-y-3">
                {/* Job Header */}
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="font-medium text-gray-900 dark:text-white">
                      {job.vm_name}
                    </h4>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      Job ID: {job.id.substring(0, 16)}...
                    </p>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Badge color={getStatusColor(job.status)} size="sm">
                      {job.status}
                    </Badge>
                    {job.throughput_mbps > 0 && (
                      <span className="text-sm text-gray-500">
                        {job.throughput_mbps.toFixed(1)} MB/s
                      </span>
                    )}
                  </div>
                </div>

                {/* Progress Bar */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-gray-600 dark:text-gray-400">
                      {job.current_operation}
                    </span>
                    <span className="font-medium text-gray-900 dark:text-white">
                      {job.progress_percent.toFixed(1)}%
                    </span>
                  </div>
                  
                  <div className="w-full bg-gray-200 rounded-full h-3 dark:bg-gray-700">
                    <div 
                      className={`h-3 rounded-full transition-all duration-500 ${getProgressColor(job.status)}`}
                      style={{ width: `${Math.min(job.progress_percent, 100)}%` }}
                    />
                  </div>

                  {/* Transfer Details */}
                  {job.total_bytes > 0 && (
                    <div className="flex items-center justify-between text-xs text-gray-500">
                      <span>
                        {formatBytes(job.bytes_transferred)} / {formatBytes(job.total_bytes)}
                      </span>
                      <span>
                        Updated: {new Date(job.updated_at).toLocaleTimeString()}
                      </span>
                    </div>
                  )}
                </div>

                {/* Real-time Pulse Indicator */}
                {job.status === 'replicating' && (
                  <div className="flex items-center justify-center">
                    <div className="flex space-x-1">
                      <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse"></div>
                      <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" style={{ animationDelay: '0.2s' }}></div>
                      <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" style={{ animationDelay: '0.4s' }}></div>
                    </div>
                  </div>
                )}
              </div>
            </Card>
          ))}
        </div>
      ) : (
        /* No Active Jobs */
        <Card>
          <div className="text-center py-8">
            <div className="mx-auto h-12 w-12 text-gray-400 mb-4">
              <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2 2v-5m16 0h-2M4 13h2m13-8V4a1 1 0 00-1-1H7a1 1 0 00-1 1v1m8 0V4a1 1 0 00-1-1H9a1 1 0 00-1 1v1" />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
              No Active Migrations
            </h3>
            <p className="text-gray-600 dark:text-gray-400">
              All migration jobs are completed or idle.
            </p>
          </div>
        </Card>
      )}

      {/* Connection Info */}
      {!isConnected && (
        <div className="text-center">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Real-time updates are currently unavailable. Progress shown from last polling update.
          </p>
        </div>
      )}
    </div>
  );
}
