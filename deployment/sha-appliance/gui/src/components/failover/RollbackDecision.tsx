'use client';

import React, { useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { HiArrowLeft, HiExclamationCircle, HiCheckCircle, HiInformationCircle, HiLightningBolt, HiTrash } from 'react-icons/hi';

export interface RollbackDecisionProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (options: RollbackOptions) => void;
  failoverType: 'test' | 'live';
  vmName: string;
  vmContext?: {
    context: {
      context_id: string;
      vmware_vm_id: string;
      vm_name: string;
    };
  };
}

export interface RollbackOptions {
  rollback_type: string;
  context_id: string;
  vm_name: string;
  vmware_vm_id: string;
  failover_type: 'test' | 'live';
  power_on_source_vm: boolean;  // ✅ Updated to match backend field name
  force_cleanup: boolean;
  preserve_snapshots?: boolean;
  cleanup_target?: boolean;
  [key: string]: any;
}

interface DecisionData {
  vm_info?: {
    status: string;
    created_at: string;
  };
  available_options?: Array<{
    value: string;
    label: string;
    description?: string;
    warning?: string;
  }>;
  current_config?: any;
  default_value?: string;
}

export const RollbackDecision: React.FC<RollbackDecisionProps> = ({
  isOpen,
  onClose,
  onConfirm,
  failoverType,
  vmName,
  vmContext
}) => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [powerOnSource, setPowerOnSource] = useState(true); // Default to true for live failover
  const [forceCleanup, setForceCleanup] = useState(false);

  // Reset state when modal opens
  useEffect(() => {
    if (isOpen) {
      setError('');
      setIsLoading(false);
      // Set default power-on option based on failover type
      setPowerOnSource(failoverType === 'live');
      setForceCleanup(false);
    }
  }, [isOpen, failoverType]);

  const handleConfirm = async () => {
    if (!vmContext?.context) return;
    
    try {
      setIsLoading(true);
      
      const options: RollbackOptions = {
        rollback_type: 'unified-rollback', // Use unified rollback type
        context_id: vmContext.context.context_id,
        vm_name: vmContext.context.vm_name,
        vmware_vm_id: vmContext.context.vmware_vm_id,
        failover_type: failoverType,
        power_on_source_vm: failoverType === 'live' ? powerOnSource : false, // Only for live failover
        force_cleanup: forceCleanup
      };

      console.log('🚀 Confirming enhanced rollback with options:', options);
      
      // Call the onConfirm handler which will initiate the rollback
      await onConfirm(options);
      
      // Note: Don't set isLoading to false here as the parent component will close the modal
      // after successful rollback creation
    } catch (error) {
      console.error('❌ ROLLBACK DECISION: Error during rollback initiation', error);
      setIsLoading(false); // Only reset loading on error
    }
  };

  const getDecisionIcon = () => {
    switch (failoverType) {
      case 'live':
        return <HiExclamationCircle className="w-6 h-6 text-orange-500" />;
      case 'test':
        return <HiInformationCircle className="w-6 h-6 text-blue-500" />;
      default:
        return <HiInformationCircle className="w-6 h-6 text-gray-500" />;
    }
  };

  const getDecisionColor = () => {
    switch (failoverType) {
      case 'live':
        return 'warning';
      case 'test':
        return 'info';
      default:
        return 'gray';
    }
  };

  if (!isOpen) return null;

  const modalContent = (
    <div 
      className="fixed inset-0 z-[45] bg-black bg-opacity-50 flex items-end justify-end p-4"
      onClick={onClose}
    >
      <div 
        className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-96 max-h-[80vh] overflow-y-auto mr-4 mb-4"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <div className="flex items-center">
            {getDecisionIcon()}
            <span className="ml-2 text-white text-lg font-semibold">
              Rollback Decision - {failoverType.charAt(0).toUpperCase() + failoverType.slice(1)} Failover
            </span>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors"
          >
            ✕
          </button>
        </div>

        {/* Body */}
        <div className="p-6 space-y-6">
          {error ? (
            <div className="bg-red-900/50 border border-red-700 text-red-200 px-4 py-3 rounded flex items-center">
              <HiExclamationCircle className="w-5 h-5 mr-2 flex-shrink-0" />
              <div>
                <span className="font-medium">Error:</span> {error}
              </div>
            </div>
          ) : (
            <div className="space-y-6">
              {/* VM Information */}
              <div className="bg-slate-700 border border-slate-600 rounded-lg p-4">
                <h5 className="mb-3 text-lg font-bold text-white flex items-center">
                  {failoverType === 'live' ? (
                    <HiLightningBolt className="w-5 h-5 text-orange-500 mr-2" />
                  ) : (
                    <HiTrash className="w-5 h-5 text-blue-500 mr-2" />
                  )}
                  {failoverType === 'live' ? 'Live Failover Rollback' : 'Test Failover Cleanup'}
                </h5>
                <div className="text-slate-300 space-y-1">
                  <div><strong className="text-slate-200">VM Name:</strong> {vmName}</div>
                  <div><strong className="text-slate-200">Context ID:</strong> {vmContext?.context?.context_id}</div>
                  {vmContext?.context?.vmware_vm_id && (
                    <div><strong className="text-slate-200">VMware VM ID:</strong> {vmContext.context.vmware_vm_id}</div>
                  )}
                </div>
              </div>

              {/* Live Failover Specific Options */}
              {failoverType === 'live' && (
                <div className="bg-orange-900/30 border border-orange-700 rounded-lg p-4">
                  <h5 className="mb-4 text-lg font-bold text-orange-200 flex items-center">
                    <HiExclamationCircle className="w-5 h-5 mr-2" />
                    Source VM Power Management
                  </h5>
                  <div className="text-orange-100 mb-4">
                    <p className="mb-2">The source VM was powered off during live failover.</p>
                    <p className="text-sm text-orange-200">Choose whether to power it back on during rollback:</p>
                  </div>
                  
                  <div className="space-y-3">
                    <div className="flex items-start">
                      <input
                        type="radio"
                        id="power-on-yes"
                        name="power-source"
                        checked={powerOnSource}
                        onChange={() => setPowerOnSource(true)}
                        className="mt-1 h-4 w-4 text-orange-600 bg-slate-100 border-slate-300 focus:ring-orange-500"
                      />
                      <div className="ml-3 flex-1">
                        <label htmlFor="power-on-yes" className="font-medium text-orange-100 cursor-pointer">
                          Yes, power on source VM
                        </label>
                        <p className="mt-1 text-sm text-orange-200">
                          Recommended: Powers on the source VM after cleaning up the failed-over resources.
                        </p>
                      </div>
                    </div>
                    
                    <div className="flex items-start">
                      <input
                        type="radio"
                        id="power-on-no"
                        name="power-source"
                        checked={!powerOnSource}
                        onChange={() => setPowerOnSource(false)}
                        className="mt-1 h-4 w-4 text-orange-600 bg-slate-100 border-slate-300 focus:ring-orange-500"
                      />
                      <div className="ml-3 flex-1">
                        <label htmlFor="power-on-no" className="font-medium text-orange-100 cursor-pointer">
                          No, leave source VM powered off
                        </label>
                        <p className="mt-1 text-sm text-orange-200">
                          The source VM will remain powered off. You can manually power it on later.
                        </p>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Test Failover Specific Options */}
              {failoverType === 'test' && (
                <div className="bg-blue-900/30 border border-blue-700 rounded-lg p-4">
                  <h5 className="mb-4 text-lg font-bold text-blue-200 flex items-center">
                    <HiInformationCircle className="w-5 h-5 mr-2" />
                    Test Cleanup Process
                  </h5>
                  <div className="text-blue-100 mb-4">
                    <p className="mb-2">This will clean up all test failover resources:</p>
                    <ul className="list-disc list-inside text-sm text-blue-200 space-y-1">
                      <li>Remove test VM from OSSEA</li>
                      <li>Detach volumes from test VM</li>
                      <li>Reattach volumes to OMA appliance</li>
                      <li>Update VM context status to ready for failover</li>
                      <li>Clean up any snapshots or temporary resources</li>
                    </ul>
                  </div>
                  <div className="bg-blue-800/50 border border-blue-600 rounded p-3">
                    <p className="text-sm text-blue-100">
                      <strong>Note:</strong> The source VM was not affected during test failover and will remain in its current state.
                    </p>
                  </div>
                </div>
              )}

              {/* Advanced Options */}
              <div className="bg-slate-700 border border-slate-600 rounded-lg p-4">
                <h5 className="mb-4 text-lg font-bold text-white">
                  Advanced Options
                </h5>
                <div className="flex items-start">
                  <input
                    type="checkbox"
                    id="force-cleanup"
                    checked={forceCleanup}
                    onChange={(e) => setForceCleanup(e.target.checked)}
                    className="mt-1 h-4 w-4 text-blue-600 bg-slate-100 border-slate-300 rounded focus:ring-blue-500"
                  />
                  <div className="ml-3 flex-1">
                    <label htmlFor="force-cleanup" className="font-medium text-white cursor-pointer">
                      Force cleanup on errors
                    </label>
                    <p className="mt-1 text-sm text-slate-300">
                      Continue cleanup process even if some operations fail. Use with caution.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-between p-4 border-t border-slate-700">
          <button
            onClick={onClose}
            disabled={isLoading}
            className="flex items-center px-4 py-2 text-slate-300 bg-slate-600 hover:bg-slate-500 disabled:opacity-50 disabled:cursor-not-allowed rounded transition-colors"
          >
            <HiArrowLeft className="mr-2 h-4 w-4" />
            Cancel
          </button>
          <button
            onClick={handleConfirm}
            disabled={isLoading}
            className={`flex items-center px-4 py-2 rounded transition-colors ${
              !isLoading
                ? failoverType === 'live'
                  ? 'text-white bg-orange-600 hover:bg-orange-700'
                  : 'text-white bg-blue-600 hover:bg-blue-700'
                : 'text-slate-400 bg-slate-600 cursor-not-allowed'
            }`}
          >
            {isLoading ? (
              <>
                <div className="inline-block animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                Initiating...
              </>
            ) : (
              <>
                {failoverType === 'live' ? (
                  <HiLightningBolt className="mr-2 h-4 w-4" />
                ) : (
                  <HiTrash className="mr-2 h-4 w-4" />
                )}
                {failoverType === 'live' ? 'Execute Rollback' : 'Cleanup Test Failover'}
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );

  return typeof document !== 'undefined' 
    ? createPortal(modalContent, document.body)
    : null;
};