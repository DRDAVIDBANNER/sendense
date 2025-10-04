'use client';

import React, { useState } from 'react';
import { OperationSummary } from '@/lib/types';
import { Button } from 'flowbite-react';
import { HiX } from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';

interface OperationErrorBannerProps {
  lastOperation?: OperationSummary;
  vmName: string;
  onDismiss?: () => void;
  onTryLiveFailover?: () => void;
  onViewDetails?: () => void;
}

export const OperationErrorBanner = React.memo(({ 
  lastOperation, 
  vmName,
  onDismiss, 
  onTryLiveFailover,
  onViewDetails
}: OperationErrorBannerProps) => {
  const [dismissed, setDismissed] = useState(false);

  // Only show if there's a failed operation and it hasn't been dismissed
  if (!lastOperation || lastOperation.status !== 'failed' || dismissed) {
    return null;
  }

  const handleDismiss = () => {
    setDismissed(true);
    onDismiss?.();
  };

  // Check if actionable steps suggest trying live failover
  const shouldShowLiveFailoverButton = 
    lastOperation.actionable_steps?.some(step => 
      step.toLowerCase().includes('live failover')
    ) || false;

  // Format operation type for display
  const operationName = lastOperation.operation_type
    .replace('_', ' ')
    .split(' ')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');

  return (
    <div className="mb-6 bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 rounded-r-lg shadow-md overflow-hidden">
      <div className="p-6">
        {/* Header */}
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <span className="text-3xl">⚠️</span>
            <div>
              <h3 className="text-lg font-bold text-red-800 dark:text-red-300">
                Last Operation Failed
              </h3>
              <p className="text-sm text-red-700 dark:text-red-400 mt-1">
                {operationName} for {vmName}
              </p>
            </div>
          </div>
          
          <button
            onClick={handleDismiss}
            className="p-1 rounded-lg text-red-500 hover:bg-red-100 dark:hover:bg-red-800/50 transition-colors"
            aria-label="Dismiss notification"
          >
            <ClientIcon className="w-5 h-5">
              <HiX />
            </ClientIcon>
          </button>
        </div>

        {/* Progress Information */}
        {lastOperation.steps_completed && lastOperation.steps_total && (
          <div className="mb-3 text-sm text-red-700 dark:text-red-400">
            Failed at step {lastOperation.steps_completed} of {lastOperation.steps_total} ({lastOperation.progress.toFixed(0)}%)
          </div>
        )}

        {/* Failed Step */}
        {lastOperation.failed_step && (
          <div className="mb-3">
            <span className="text-sm font-medium text-red-800 dark:text-red-300">Failed Step: </span>
            <span className="text-sm text-red-700 dark:text-red-400">{lastOperation.failed_step}</span>
          </div>
        )}

        {/* Error Message - SANITIZED */}
        <div className="mb-4 p-3 bg-white dark:bg-gray-800 rounded-lg border border-red-200 dark:border-red-700/50">
          <div className="text-sm font-medium text-red-800 dark:text-red-300 mb-1">
            Issue:
          </div>
          <div className="text-sm text-red-700 dark:text-red-400">
            {lastOperation.error_message}
          </div>
        </div>

        {/* Actionable Steps - PROMINENT DISPLAY */}
        {lastOperation.actionable_steps && lastOperation.actionable_steps.length > 0 && (
          <div className="mb-4 p-3 bg-white dark:bg-gray-800 rounded-lg border border-red-200 dark:border-red-700/50">
            <div className="text-sm font-medium text-red-800 dark:text-red-300 mb-2">
              What you can do:
            </div>
            <ul className="space-y-2">
              {lastOperation.actionable_steps.map((step, index) => (
                <li key={index} className="flex items-start gap-2 text-sm text-red-700 dark:text-red-400">
                  <span className="text-red-500 dark:text-red-400 mt-0.5">•</span>
                  <span>{step}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex gap-3 flex-wrap">
          <Button
            size="sm"
            color="light"
            onClick={handleDismiss}
          >
            Dismiss
          </Button>

          {shouldShowLiveFailoverButton && onTryLiveFailover && (
            <Button
              size="sm"
              color="failure"
              onClick={onTryLiveFailover}
            >
              Try Live Failover
            </Button>
          )}

          {onViewDetails && (
            <Button
              size="sm"
              color="light"
              onClick={onViewDetails}
            >
              View Details
            </Button>
          )}
        </div>

        {/* Metadata */}
        <div className="mt-4 pt-3 border-t border-red-200 dark:border-red-700/50">
          <div className="flex items-center justify-between text-xs text-red-600 dark:text-red-400">
            <span>
              {lastOperation.error_category && (
                <span className="font-medium">Category: {lastOperation.error_category}</span>
              )}
            </span>
            <span>
              {formatTimestamp(lastOperation.timestamp)}
              {lastOperation.duration_seconds && ` • Duration: ${formatDuration(lastOperation.duration_seconds)}`}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
});

OperationErrorBanner.displayName = 'OperationErrorBanner';

// Helper functions
function formatTimestamp(ts: string): string {
  const date = new Date(ts);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins} min ago`;
  if (diffMins < 1440) return `${Math.floor(diffMins / 60)} hours ago`;
  return `${Math.floor(diffMins / 1440)} days ago`;
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
}


