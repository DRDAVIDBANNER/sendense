'use client';

import React, { useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { HiCog, HiExclamationCircle, HiCheckCircle, HiInformationCircle } from 'react-icons/hi';

export interface PreFlightConfigurationProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (config: FailoverConfiguration) => void;
  vmName: string;
  failoverType: string;
  vmContext: {
    context_id: string;
    vmware_vm_id: string;
    vm_name: string;
    [key: string]: any;
  };
}

export interface FailoverConfiguration {
  context_id: string;
  vmware_vm_id: string;
  vm_name: string;
  failover_type: string;
  // Optional behaviors - match backend UnifiedFailoverRequest
  skip_validation?: boolean;
  skip_virtio?: boolean;
  // Advanced options
  test_duration?: string;
  custom_config?: Record<string, any>;
  network_mappings?: Record<string, string>;
}

interface ConfigurationOption {
  default: any;
  required: boolean;
  description: string;
  type: 'boolean' | 'string' | 'number' | 'select';
  options?: string[];
}

interface ConfigurationMetadata {
  configuration: Record<string, ConfigurationOption>;
  metadata: {
    description: string;
    version: string;
    timestamp: string;
  };
}

export const PreFlightConfiguration: React.FC<PreFlightConfigurationProps> = ({
  isOpen,
  onClose,
  onConfirm,
  vmName,
  failoverType,
  vmContext
}) => {
  const [configMetadata, setConfigMetadata] = useState<ConfigurationMetadata | null>(null);
  const [configuration, setConfiguration] = useState<FailoverConfiguration>({
    context_id: vmContext.context_id,
    vmware_vm_id: vmContext.vmware_vm_id,
    vm_name: vmContext.vm_name,
    failover_type: failoverType
  });
  const [isLoading, setIsLoading] = useState(false);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);
  const [isValidating, setIsValidating] = useState(false);

  // Load configuration metadata when modal opens or failover type changes
  useEffect(() => {
    if (isOpen) {
      // COMPLETE RESET - Clear everything and start fresh
      console.log('🔄 PRE-FLIGHT CONFIG: Modal opened with failoverType:', failoverType);
      console.log('🔄 PRE-FLIGHT CONFIG: vmContext:', vmContext);
      
      // Reset all state immediately
      setConfigMetadata(null);
      setValidationErrors([]);
      setIsValidating(false);
      
      // Set base configuration with current failoverType
      const baseConfig: FailoverConfiguration = {
        context_id: vmContext.context_id,
        vmware_vm_id: vmContext.vmware_vm_id,
        vm_name: vmContext.vm_name,
        failover_type: failoverType // Use the prop directly
      };
      
      console.log('🔄 PRE-FLIGHT CONFIG: Setting base config:', baseConfig);
      setConfiguration(baseConfig);
      
      // Load backend configuration after state reset
      loadConfigurationMetadata();
    }
  }, [isOpen, failoverType, vmContext.context_id, vmContext.vmware_vm_id, vmContext.vm_name]);

  const loadConfigurationMetadata = async () => {
    try {
      setIsLoading(true);
      const response = await fetch(`/api/failover/preflight/config/${failoverType}/${vmName}`);
      
      if (response.ok) {
        const data = await response.json();
        console.log('🔧 PRE-FLIGHT CONFIG: Configuration metadata loaded', data);
        setConfigMetadata(data);
        
        // Reset configuration to base fields and apply backend defaults
        if (data.configuration) {
          // Start with only the base required fields
          const baseConfig: FailoverConfiguration = {
            context_id: vmContext.context_id,
            vmware_vm_id: vmContext.vmware_vm_id,
            vm_name: vmContext.vm_name,
            failover_type: failoverType
          };
          
          // Apply only the defaults provided by the backend for this failover type
          Object.entries(data.configuration).forEach(([key, option]) => {
            (baseConfig as any)[key] = option.default;
          });
          
          console.log('🔧 PRE-FLIGHT CONFIG: Reset configuration for', failoverType, baseConfig);
          console.log('🔧 PRE-FLIGHT CONFIG: Backend data received:', data);
          setConfiguration(baseConfig);
        }
      }
    } catch (error) {
      console.error('Failed to load configuration metadata:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const validateConfiguration = async () => {
    try {
      setIsValidating(true);
      setValidationErrors([]);
      
      // Basic validation
      const errors: string[] = [];
      
      if (!configuration.context_id) {
        errors.push('Context ID is required');
      }
      
      if (!configuration.vmware_vm_id) {
        errors.push('VMware VM ID is required');
      }
      
      if (!configuration.vm_name) {
        errors.push('VM name is required');
      }
      
      if (configMetadata?.configuration) {
        Object.entries(configMetadata.configuration).forEach(([key, option]) => {
          if (option.required && !configuration[key as keyof FailoverConfiguration]) {
            errors.push(`${key} is required`);
          }
        });
      }
      
      setValidationErrors(errors);
      
      if (errors.length === 0) {
        console.log('✅ PRE-FLIGHT CONFIG: Configuration validation passed');
      }
    } catch (error) {
      console.error('Configuration validation failed:', error);
      setValidationErrors(['Configuration validation failed']);
    } finally {
      setIsValidating(false);
    }
  };

  const handleConfirm = async () => {
    if (validationErrors.length === 0) {
      try {
        setIsLoading(true);
        console.log('🚀 PRE-FLIGHT CONFIG: Starting failover with configuration', configuration);
        console.log('🚀 PRE-FLIGHT CONFIG: failoverType prop value:', failoverType);
        console.log('🚀 PRE-FLIGHT CONFIG: configuration.failover_type value:', configuration.failover_type);
        
        // Call the onConfirm handler which will initiate the failover
        await onConfirm(configuration);
        
        // Note: Don't set isLoading to false here as the parent component will close the modal
        // after successful job creation
      } catch (error) {
        console.error('❌ PRE-FLIGHT CONFIG: Error during failover initiation', error);
        setIsLoading(false); // Only reset loading on error
      }
    } else {
      console.warn('⚠️ PRE-FLIGHT CONFIG: Cannot start failover with validation errors', validationErrors);
    }
  };

  const renderConfigurationField = (key: string, option: ConfigurationOption) => {
    const value = configuration[key as keyof FailoverConfiguration];
    
    switch (option.type) {
      case 'boolean':
        return (
          <div key={key} className="flex items-center gap-3">
            <input
              type="checkbox"
              id={key}
              checked={Boolean(value)}
              onChange={(e) => setConfiguration(prev => ({
                ...prev,
                [key]: e.target.checked
              }))}
              className="h-4 w-4 text-cyan-500 focus:ring-cyan-500 bg-slate-700 border-slate-600 rounded"
            />
            <div>
              <label htmlFor={key} className="text-sm font-medium text-gray-300">
                {key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                {option.required && <span className="text-red-400 ml-1">*</span>}
              </label>
              <p className="text-xs text-gray-400">{option.description}</p>
            </div>
          </div>
        );
      
      case 'select':
        return (
          <div key={key}>
            <label htmlFor={key} className="block text-sm font-medium text-gray-300 mb-2">
              {key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
              {option.required && <span className="text-red-400 ml-1">*</span>}
            </label>
            <select
              id={key}
              value={String(value || '')}
              onChange={(e) => setConfiguration(prev => ({
                ...prev,
                [key]: e.target.value
              }))}
              className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
            >
              {option.options?.map(opt => (
                <option key={opt} value={opt}>{opt}</option>
              ))}
            </select>
            <p className="text-xs text-gray-400 mt-1">{option.description}</p>
          </div>
        );
      
      case 'number':
        return (
          <div key={key}>
            <label htmlFor={key} className="block text-sm font-medium text-gray-300 mb-2">
              {key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
              {option.required && <span className="text-red-400 ml-1">*</span>}
            </label>
            <input
              type="number"
              id={key}
              value={String(value || '')}
              onChange={(e) => setConfiguration(prev => ({
                ...prev,
                [key]: parseInt(e.target.value) || 0
              }))}
              className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
            />
            <p className="text-xs text-gray-400 mt-1">{option.description}</p>
          </div>
        );
      
      default:
        return (
          <div key={key}>
            <label htmlFor={key} className="block text-sm font-medium text-gray-300 mb-2">
              {key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
              {option.required && <span className="text-red-400 ml-1">*</span>}
            </label>
            <input
              type="text"
              id={key}
              value={String(value || '')}
              onChange={(e) => setConfiguration(prev => ({
                ...prev,
                [key]: e.target.value
              }))}
              className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
            />
            <p className="text-xs text-gray-400 mt-1">{option.description}</p>
          </div>
        );
    }
  };

  if (!isOpen) return null;

  const modalContent = (
    <div 
      className="fixed inset-0 z-[45] bg-black bg-opacity-50 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div 
        className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="bg-slate-800 px-6 pt-6 pb-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <HiCog className="w-6 h-6 text-blue-500 mr-2" />
              <h3 className="text-lg font-medium text-white">
                Pre-flight Configuration - {failoverType.charAt(0).toUpperCase() + failoverType.slice(1)} Failover
              </h3>
            </div>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-200 text-2xl font-bold leading-none"
            >
              ×
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="px-6 pb-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-cyan-500"></div>
              <span className="ml-3 text-gray-300">Loading configuration options...</span>
            </div>
          ) : (
            <div className="space-y-6">
              {/* VM Information */}
              <div className="bg-slate-700/50 border border-slate-600 rounded-lg p-4">
                <div className="flex items-center mb-3">
                  <HiInformationCircle className="w-5 h-5 text-blue-500 mr-2" />
                  <h4 className="text-lg font-semibold text-white">VM Information</h4>
                </div>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="font-medium text-gray-300">VM Name:</span>
                    <span className="ml-2 text-white">{vmName}</span>
                  </div>
                  <div>
                    <span className="font-medium text-gray-300">Failover Type:</span>
                    <span className="ml-2 text-white">{failoverType}</span>
                  </div>
                  <div>
                    <span className="font-medium text-gray-300">Context ID:</span>
                    <span className="ml-2 text-white font-mono text-xs">{vmContext.context_id}</span>
                  </div>
                  <div>
                    <span className="font-medium text-gray-300">VMware VM ID:</span>
                    <span className="ml-2 text-white font-mono text-xs">{vmContext.vmware_vm_id}</span>
                  </div>
                </div>
              </div>

              {/* Configuration Options */}
              {configMetadata && (
                <div className="bg-slate-700/50 border border-slate-600 rounded-lg p-4">
                  <div className="flex items-center mb-4">
                    <HiCog className="w-5 h-5 text-green-500 mr-2" />
                    <h4 className="text-lg font-semibold text-white">Configuration Options</h4>
                  </div>
                  <p className="text-sm text-gray-300 mb-4">{configMetadata.metadata.description}</p>
                  
                  <div className="space-y-4">
                    {Object.entries(configMetadata.configuration).map(([key, option]) =>
                      renderConfigurationField(key, option)
                    )}
                  </div>
                </div>
              )}

              {/* Validation Errors */}
              {validationErrors.length > 0 && (
                <div className="bg-red-500/20 border border-red-500/30 rounded-lg p-4">
                  <div className="flex items-center mb-3">
                    <HiExclamationCircle className="w-5 h-5 text-red-400 mr-2" />
                    <span className="font-medium text-red-300">Configuration Validation Failed:</span>
                  </div>
                  <ul className="list-disc list-inside space-y-1">
                    {validationErrors.map((error, index) => (
                      <li key={index} className="text-red-300 text-sm">{error}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="bg-slate-700/50 border-t border-slate-600 px-6 py-4 flex justify-between">
          <button
            onClick={onClose}
            disabled={isLoading || isValidating}
            className="px-4 py-2 bg-slate-600 text-gray-300 rounded-md hover:bg-slate-500 focus:outline-none focus:ring-2 focus:ring-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            Cancel
          </button>
          <div className="flex space-x-3">
            <button
              onClick={validateConfiguration}
              disabled={isLoading || isValidating}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isValidating ? (
                <>
                  <div className="inline-block animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                  Validating...
                </>
              ) : (
                'Validate Configuration'
              )}
            </button>
            <button
              onClick={handleConfirm}
              disabled={isLoading || isValidating}
              className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center"
            >
              {isLoading ? (
                <>
                  <div className="inline-block animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                  Initiating...
                </>
              ) : (
                <>
                  <HiCheckCircle className="w-4 h-4 mr-2" />
                  Start {failoverType.charAt(0).toUpperCase() + failoverType.slice(1)} Failover
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );

  // Use portal to render modal outside the component tree
  return typeof document !== 'undefined' 
    ? createPortal(modalContent, document.body)
    : null;
};