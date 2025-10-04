'use client';

import React from 'react';
import { Modal, Button, Badge } from 'flowbite-react';
import { UnifiedJob, OperationSummary } from '@/lib/types';
import { HiX, HiLightningBolt, HiExclamationCircle, HiInformationCircle } from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';

interface JobErrorDetailsModalProps {
  isOpen: boolean;
  onClose: () => void;
  job?: UnifiedJob | OperationSummary;
  vmName: string;
  onTryLiveFailover?: () => void;
  showAdminDetails?: boolean;
}

export const JobErrorDetailsModal = React.memo(({ 
  isOpen, 
  onClose, 
  job,
  vmName,
  onTryLiveFailover,
  showAdminDetails = false
}: JobErrorDetailsModalProps) => {
  if (!job) return null;

  // Determine if this is a UnifiedJob or OperationSummary
  const isUnifiedJob = 'display_name' in job;
  const displayName = isUnifiedJob ? job.display_name : formatOperationType(job.operation_type);
  const failedStep = isUnifiedJob ? job.current_step : job.failed_step;
  const isFailed = job.status === 'failed';

  // Check if we should show live failover button
  const shouldShowLiveFailoverButton = 
    job.actionable_steps?.some(step => 
      step.toLowerCase().includes('live failover')
    ) || false;

  return (
    <Modal show={isOpen} onClose={onClose} size="lg">
      <Modal.Header>
        <div className="flex items-center gap-3">
          <ClientIcon className={`w-8 h-8 ${isFailed ? 'text-red-500' : 'text-blue-500'}`}>
            {isFailed ? <HiExclamationCircle /> : <HiInformationCircle />}
          </ClientIcon>
          <div>
            <h3 className="text-xl font-bold text-gray-900 dark:text-white">
              {displayName}
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              {vmName}
            </p>
          </div>
        </div>
      </Modal.Header>

      <Modal.Body>
        <div className="space-y-4">
          {/* Job Metadata */}
          <div className="grid grid-cols-2 gap-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <div>
              <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Status</div>
              <Badge color={getStatusBadgeColor(job.status)} size="sm">
                {job.status.toUpperCase()}
              </Badge>
            </div>
            
            <div>
              <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Progress</div>
              <div className="font-medium text-gray-900 dark:text-white">
                {job.progress.toFixed(0)}%
              </div>
            </div>

            {job.duration_seconds !== undefined && (
              <div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Duration</div>
                <div className="font-medium text-gray-900 dark:text-white">
                  {formatDuration(job.duration_seconds)}
                </div>
              </div>
            )}

            {'steps_completed' in job && job.steps_completed !== undefined && job.steps_total !== undefined && (
              <div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Steps</div>
                <div className="font-medium text-gray-900 dark:text-white">
                  {job.steps_completed} of {job.steps_total}
                </div>
              </div>
            )}
          </div>

          {/* Failed Step Information */}
          {isFailed && failedStep && (
            <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-700/50">
              <div className="text-sm font-medium text-red-800 dark:text-red-300 mb-2">
                Failed Step:
              </div>
              <div className="text-base font-semibold text-red-900 dark:text-red-200">
                {failedStep}
              </div>
            </div>
          )}

          {/* Error Message - SANITIZED ONLY */}
          {isFailed && job.error_message && (
            <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-700/50">
              <div className="text-sm font-medium text-red-800 dark:text-red-300 mb-2">
                Issue:
              </div>
              <div className="text-base text-red-900 dark:text-red-200">
                {job.error_message}
              </div>
            </div>
          )}

          {/* Error Category */}
          {isFailed && job.error_category && (
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-600 dark:text-gray-400">Category:</span>
              <Badge color="red" size="sm">
                {job.error_category}
              </Badge>
              {('error_severity' in job) && job.error_severity && (
                <Badge color={getSeverityBadgeColor(job.error_severity)} size="sm">
                  {job.error_severity}
                </Badge>
              )}
            </div>
          )}

          {/* Actionable Steps - PROMINENT DISPLAY */}
          {isFailed && job.actionable_steps && job.actionable_steps.length > 0 && (
            <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-700/50">
              <div className="text-sm font-bold text-blue-800 dark:text-blue-300 mb-3">
                What You Can Do:
              </div>
              <ul className="space-y-2">
                {job.actionable_steps.map((step, index) => (
                  <li key={index} className="flex items-start gap-3">
                    <span className="text-blue-600 dark:text-blue-400 font-bold mt-0.5">•</span>
                    <span className="text-sm text-blue-900 dark:text-blue-200 flex-1">
                      {step}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Admin Technical Details - HIDDEN BY DEFAULT */}
          {showAdminDetails && isFailed && ('failed_step_internal' in job) && job.failed_step_internal && (
            <details className="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
              <summary className="cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-300">
                Technical Details (Admin Only)
              </summary>
              <div className="mt-3 space-y-2 text-xs font-mono text-gray-600 dark:text-gray-400">
                <div>
                  <span className="font-semibold">Internal Step:</span> {job.failed_step_internal}
                </div>
                <div>
                  <span className="font-semibold">Job ID:</span> {job.job_id}
                </div>
                {'external_job_id' in job && job.external_job_id && (
                  <div>
                    <span className="font-semibold">External Job ID:</span> {job.external_job_id}
                  </div>
                )}
              </div>
            </details>
          )}
        </div>
      </Modal.Body>

      <Modal.Footer>
        <div className="flex gap-2 w-full justify-end">
          <Button color="gray" onClick={onClose}>
            Close
          </Button>
          
          {isFailed && shouldShowLiveFailoverButton && onTryLiveFailover && (
            <Button color="failure" onClick={() => {
              onTryLiveFailover();
              onClose();
            }}>
              <ClientIcon className="w-4 h-4 mr-2">
                <HiLightningBolt />
              </ClientIcon>
              Try Live Failover
            </Button>
          )}
        </div>
      </Modal.Footer>
    </Modal>
  );
});

JobErrorDetailsModal.displayName = 'JobErrorDetailsModal';

// Helper functions
function getStatusBadgeColor(status: string): 'success' | 'failure' | 'warning' | 'gray' {
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
}

function getSeverityBadgeColor(severity: string): 'info' | 'warning' | 'failure' | 'purple' {
  switch (severity) {
    case 'critical':
      return 'failure';
    case 'error':
      return 'failure';
    case 'warning':
      return 'warning';
    case 'info':
      return 'info';
    default:
      return 'purple';
  }
}

function formatOperationType(opType: string): string {
  return opType
    .replace('_', ' ')
    .split(' ')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${mins}m`;
}


