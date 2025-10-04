'use client';

import React, { useState, useEffect } from 'react';
import { Card, Badge, Button, Spinner } from 'flowbite-react';
import { 
  HiPlay, 
  HiLightningBolt, 
  HiBeaker, 
  HiRefresh,
  HiExclamationCircle,
  HiCheckCircle,
  HiClock,
  HiX
} from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';
import { useVMContext } from '../../hooks/useVMContext';
import { useSystemHealth } from '../../hooks/useSystemHealth';
import { useVMJobProgress, formatBytes, formatSpeed, formatETA, getProgressColor, formatDuration, formatStartTime } from '../../hooks/useJobProgress';
import { useNotifications } from '../ui/NotificationSystem';
import { PreFlightConfiguration, FailoverConfiguration } from '../failover/PreFlightConfiguration';
import { RollbackDecision, RollbackOptions } from '../failover/RollbackDecision';
import { UnifiedProgressTracker } from '../failover/UnifiedProgressTracker';
import { DecisionAuditProvider, usePreFlightAudit, useRollbackAudit } from '../failover/DecisionAuditLogger';
import { UnifiedJobList } from '../jobs/UnifiedJobList';
import { JobErrorDetailsModal } from '../jobs/JobErrorDetailsModal';
import { UnifiedJob } from '@/lib/types';

export interface RightContextPanelProps {
  selectedVM: string | null;
  onVMSelect: (vmName: string | null) => void;
}

interface QuickAction {
  id: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  color: 'success' | 'failure' | 'purple' | 'warning' | 'blue';
  onClick: () => void;
  disabled?: boolean;
}

const RightContextPanelInner = React.memo(({ selectedVM, onVMSelect }: RightContextPanelProps) => {
  const { data: vmContext, isLoading: vmLoading, error: vmError } = useVMContext(selectedVM);
  const { data: systemHealth, isLoading: healthLoading } = useSystemHealth();
  const { data: jobProgress, isLoading: jobLoading } = useVMJobProgress(selectedVM);
  const { success, error: showError, warning, info } = useNotifications();
  
  // Enhanced failover state
  const [preFlightModalOpen, setPreFlightModalOpen] = useState(false);
  const [rollbackModalOpen, setRollbackModalOpen] = useState(false);
  const [currentFailoverType, setCurrentFailoverType] = useState<'live' | 'test'>('test');
  const [activeJobId, setActiveJobId] = useState<string | null>(null);
  
  // Job error details modal state
  const [selectedJob, setSelectedJob] = useState<UnifiedJob | null>(null);
  const [isJobErrorModalOpen, setIsJobErrorModalOpen] = useState(false);
  
  // Audit logging hooks
  const { logPreFlightConfiguration } = usePreFlightAudit();
  const { logRollbackDecision } = useRollbackAudit();
  
  // Restore active job from localStorage on component mount (persistent state)
  useEffect(() => {
    const savedJobId = localStorage.getItem('ossea-migrate-active-job');
    if (savedJobId && savedJobId !== activeJobId) {
      console.log('🔄 Restoring active job from localStorage:', savedJobId);
      setActiveJobId(savedJobId);
    }
  }, [activeJobId]);
  
  // Persist active job to localStorage when it changes
  const setActiveJobWithPersistence = React.useCallback((jobId: string | null) => {
    console.log('💾 Persisting active job to localStorage:', jobId);
    setActiveJobId(jobId);
    
    if (jobId) {
      localStorage.setItem('ossea-migrate-active-job', jobId);
    } else {
      localStorage.removeItem('ossea-migrate-active-job');
    }
  }, []);
  

  const handleStartReplication = React.useCallback(async () => {
    console.log('🔥 BUTTON CLICKED - handleStartReplication called');
    console.log('🔥 selectedVM:', selectedVM);
    console.log('🔥 vmContext:', vmContext);
    
    if (!selectedVM || !vmContext) {
      console.log('❌ Missing selectedVM or vmContext');
      showError('Error', 'Please select a VM first');
      return;
    }
    
    try {
      info('Starting replication', `Discovering fresh VM data for ${selectedVM}...`);
      console.log('🔍 Starting discovery for', selectedVM);
      
      // Step 1: Get fresh VM data with timeout to avoid hanging
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 30000); // 30 second timeout
      
      // Use Enhanced Discovery API with credential_id from VM context
      const discoveryResponse = await fetch('/api/discovery/discover-vms', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          credential_id: vmContext.context.credential_id || 2, // Use VM's credential or default
          filter: selectedVM,
          create_context: false
        }),
        signal: controller.signal
      });
      
      clearTimeout(timeoutId);

      if (!discoveryResponse.ok) {
        const discoveryError = await discoveryResponse.json();
        throw new Error(`Discovery failed: ${discoveryError.error || 'Unknown error'}`);
      }

      const discoveryData = await discoveryResponse.json();
      console.log('✅ Fresh VM data discovered:', discoveryData);
      
      // Find the VM in discovery results (Enhanced Discovery returns discovered_vms array)
      const discoveredVM = discoveryData.discovered_vms?.find((vm: any) => vm.name === selectedVM);
      if (!discoveredVM) {
        throw new Error(`VM ${selectedVM} not found in discovery results`);
      }

      if (!discoveredVM.disks || discoveredVM.disks.length === 0) {
        throw new Error(`VM ${selectedVM} has no disks configured`);
      }

      info('Starting replication', `Using fresh VM data with ${discoveredVM.disks.length} disk(s)`);
      console.log('🚀 Starting replication with fresh VM data:', discoveredVM);
      
      // Step 2: Start replication with fresh, complete VM data including disks
      const response = await fetch('/api/replicate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          source_vm: {
            // Use fresh discovery data (complete VM specs)
            id: discoveredVM.id,
            name: discoveredVM.name,
            path: discoveredVM.path,
            vm_id: discoveredVM.id,
            vm_name: discoveredVM.name,
            vm_path: discoveredVM.path,
            datacenter: discoveredVM.datacenter,
            vcenter_host: vmContext.context.vcenter_host || 'quad-vcenter-01.quadris.local',
            cpus: discoveredVM.num_cpu || discoveredVM.cpus || 2,
            memory_mb: discoveredVM.memory_mb || 4096,
            power_state: discoveredVM.power_state || "poweredOn",
            os_type: discoveredVM.guest_os || "otherGuest",
            vmx_version: discoveredVM.vmx_version,
            disks: discoveredVM.disks, // This is critical!
            networks: discoveredVM.networks
          },
          ossea_config_id: 1, // REQUIRED: Production OSSEA configuration
          replication_type: 'initial',
          vcenter_host: vmContext.context.vcenter_host || 'quad-vcenter-01.quadris.local',
          datacenter: vmContext.context.datacenter || 'DatabanxDC'
        })
      });
      
      console.log('📡 Replication API response status:', response.status);
      const result = await response.json();
      console.log('📡 Replication API result:', result);
      
      if (response.ok) {
        console.log('✅ Replication started successfully:', result);
        success('Replication Started', `Successfully started replication for ${selectedVM} with ${discoveredVM.disks.length} disk(s). Job ID: ${result.job_id || 'Unknown'}`);
        
        // Add info notification about grace period to help with debugging
        setTimeout(() => {
          info('Grace Period Active', 'Job progress polling will suppress errors for the next 15 seconds during transition period');
        }, 1000);
      } else {
        console.error('❌ Replication failed:', result);
        showError('Replication Failed', result.error || 'Failed to start replication');
      }
    } catch (error) {
      console.error('❌ Replication error:', error);
      if (error.name === 'AbortError') {
        showError('Discovery Timeout', 'VM discovery took too long. Please try again.');
      } else {
        showError('Replication Error', error instanceof Error ? error.message : 'Network error while starting replication');
      }
    }
  }, [selectedVM, vmContext, success, showError, info]);

  // Enhanced unified failover handlers
  const handleUnifiedFailover = React.useCallback(async (config: FailoverConfiguration) => {
    if (!selectedVM || !vmContext) return;
    
    try {
      info('Starting unified failover', `Initiating ${config.failover_type} failover for ${selectedVM} with enhanced configuration`);
      console.log('🚀 Starting unified failover with config:', config);
      
      // Log pre-flight configuration decision
      logPreFlightConfiguration(
        config.vm_name,
        config.context_id,
        config.failover_type,
        config
      );
      
      const response = await fetch('/api/failover/unified', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      });
      
      const data = await response.json();
      
      if (data.success) {
        success('Unified failover started', `${config.failover_type.charAt(0).toUpperCase() + config.failover_type.slice(1)} failover initiated successfully! Job ID: ${data.job_id}`);
        setActiveJobWithPersistence(data.job_id);
        console.log('✅ Unified failover started successfully', { job_id: data.job_id });
        
        // Close the pre-flight modal after successful job creation
        setTimeout(() => {
          setPreFlightModalOpen(false);
        }, 2000); // Allow user to see success message for 2 seconds
        
      } else {
        showError('Unified failover failed', data.message || 'Failed to start unified failover');
        console.error('❌ Unified failover failed', data);
      }
    } catch (error) {
      showError('Network error', 'Failed to communicate with the server');
      console.error('❌ Network error during unified failover', error);
    }
  }, [selectedVM, vmContext, info, success, showError, logPreFlightConfiguration]);

  const handleRollback = React.useCallback(async (options: RollbackOptions) => {
    if (!selectedVM || !vmContext) return;
    
    try {
      info('Starting rollback', `Initiating rollback for ${selectedVM} (this may take several minutes)`);
      console.log('🔄 Starting rollback with options:', options);
      
      // Log rollback decision  
      logRollbackDecision(
        options.vm_name,
        options.context_id,
        options.failover_type,
        'Rollback decision',
        `Power on source: ${options.power_on_source_vm}, Force cleanup: ${options.force_cleanup}`,
        options
      );
      
      // Create AbortController for timeout handling
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 300000); // 5 minute timeout for rollback
      
      console.log('🔍 ROLLBACK DEBUG: About to start fetch request to /api/failover/rollback');
      console.log('🔍 ROLLBACK DEBUG: Request options:', options);
      
      const response = await fetch('/api/failover/rollback', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(options),
        signal: controller.signal
      });
      
      console.log('🔍 ROLLBACK DEBUG: Fetch completed, response status:', response.status);
      
      let data;
      try {
        data = await response.json();
        console.log('🔍 ROLLBACK DEBUG: JSON parsing successful');
      } catch (jsonError) {
        console.error('🔍 ROLLBACK DEBUG: JSON parsing failed:', jsonError);
        throw new Error('Failed to parse response JSON');
      }
      
      clearTimeout(timeoutId);
      
      console.log('🔍 ROLLBACK DEBUG: Full response data:', data);
      console.log('🔍 ROLLBACK DEBUG: data.success =', data.success);
      console.log('🔍 ROLLBACK DEBUG: data.job_id =', data.job_id);
      
      if (data.success) {
        success('Rollback started', `Rollback initiated successfully! Job ID: ${data.job_id || 'N/A'}`);
        setActiveJobWithPersistence(data.job_id);
        console.log('✅ Rollback started successfully', { job_id: data.job_id });
        console.log('🔍 ROLLBACK DEBUG: Setting activeJobId to:', data.job_id);
        console.log('🔍 ROLLBACK DEBUG: Setting timeout to close modal in 2 seconds...');
        
        // Close the rollback modal after successful job creation
        setTimeout(() => {
          console.log('🔍 ROLLBACK DEBUG: Timeout triggered, closing modal...');
          setRollbackModalOpen(false);
        }, 2000); // Allow user to see success message for 2 seconds
        
      } else {
        showError('Rollback failed', data.message || 'Failed to start rollback');
        console.error('❌ Rollback failed', data);
      }
    } catch (error: any) {
      console.error('🔍 ROLLBACK DEBUG: Caught error in rollback handler:', error);
      console.error('🔍 ROLLBACK DEBUG: Error name:', error.name);
      console.error('🔍 ROLLBACK DEBUG: Error message:', error.message);
      
      if (error.name === 'AbortError') {
        showError('Rollback timeout', 'Rollback operation timed out after 5 minutes');
      } else {
        showError('Network error', 'Failed to communicate with the server during rollback');
      }
      console.error('❌ Error during rollback', error);
    }
  }, [selectedVM, vmContext, info, success, showError, logRollbackDecision]);

  const handleLiveFailover = React.useCallback(async () => {
    if (!selectedVM || !vmContext) return;
    
    try {
      info('Starting live failover', `Initiating live failover for ${selectedVM}`);
      console.log('⚡ Starting live failover for', selectedVM);
      console.log('🔍 VM Context:', vmContext.context);
      const response = await fetch('/api/failover', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          context_id: vmContext.context.context_id,
          vm_id: vmContext.context.vmware_vm_id,
          vm_name: vmContext.context.vm_name,
          failover_type: 'live',
          skip_validation: false,
          network_mappings: {},
          custom_config: {}
        })
      });
      
      const result = await response.json();
      if (response.ok) {
        console.log('✅ Live failover started successfully:', result);
        success('Live Failover Started', `Successfully started live failover for ${selectedVM}. Job ID: ${result.job_id}`);
        // TODO: Optionally redirect to failover page for monitoring
      } else {
        console.error('❌ Live failover failed:', result);
        showError('Live Failover Failed', result.error || 'Failed to start live failover');
      }
    } catch (error) {
      console.error('❌ Live failover error:', error);
      showError('Live Failover Error', 'Network error while starting live failover');
    }
  }, [selectedVM, vmContext, success, showError, info]);

  const handleTestFailover = React.useCallback(async () => {
    if (!selectedVM || !vmContext) return;
    
    try {
      info('Starting test failover', `Initiating test failover for ${selectedVM}`);
      console.log('🧪 Starting test failover for', selectedVM);
      console.log('🔍 VM Context:', vmContext.context);
      const response = await fetch('/api/failover', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          context_id: vmContext.context.context_id,
          vm_id: vmContext.context.vmware_vm_id,
          vm_name: vmContext.context.vm_name,
          failover_type: 'test',
          skip_validation: false,
          test_duration: '2h',
          auto_cleanup: true,
          network_mappings: {},
          custom_config: {}
        })
      });
      
      const result = await response.json();
      if (response.ok) {
        console.log('✅ Test failover started successfully:', result);
        success('Test Failover Started', `Successfully started test failover for ${selectedVM}. Duration: 2h with auto-cleanup.`);
        // TODO: Optionally redirect to failover page for monitoring
      } else {
        console.error('❌ Test failover failed:', result);
        showError('Test Failover Failed', result.error || 'Failed to start test failover');
      }
    } catch (error) {
      console.error('❌ Test failover error:', error);
      showError('Test Failover Error', 'Network error while starting test failover');
    }
  }, [selectedVM, vmContext, success, showError, info]);

  const handleCleanup = React.useCallback(async () => {
    if (!selectedVM || !vmContext) return;
    
    try {
      info('Starting cleanup', `Initiating cleanup for ${selectedVM} (this may take 1-2 minutes)`);
      console.log('🧹 Starting cleanup for', selectedVM);
      
      // Create AbortController for timeout handling
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 180000); // 3 minute timeout
      
      const response = await fetch('/api/cleanup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          context_id: vmContext.context.context_id,
          vm_id: vmContext.context.vmware_vm_id,
          vm_name: vmContext.context.vm_name,
          cleanup_type: 'test_failover'
        }),
        signal: controller.signal
      });
      
      clearTimeout(timeoutId);
      
      const result = await response.json();
      if (response.ok) {
        console.log('✅ Cleanup started successfully:', result);
        success('Cleanup Started', `Successfully started cleanup for ${selectedVM}`);
      } else {
        console.error('❌ Cleanup failed:', result);
        const errorMessage = result.error || result.message || 'Failed to start cleanup';
        showError('Cleanup Failed', errorMessage);
      }
    } catch (error) {
      console.error('❌ Cleanup error:', error);
      if (error.name === 'AbortError') {
        showError('Cleanup Timeout', 'Cleanup operation timed out after 3 minutes. Check logs to verify completion.');
      } else {
        const errorMessage = error.message || 'Network error while starting cleanup';
        showError('Cleanup Error', errorMessage);
      }
    }
  }, [selectedVM, vmContext, success, showError, info]);

  // Handle failed execution cleanup
  const handleCleanupFailedExecution = React.useCallback(async () => {
    if (!selectedVM) {
      showError('Error', 'Please select a VM first');
      return;
    }

    try {
      const response = await fetch(`/api/v1/failover/${encodeURIComponent(selectedVM)}/cleanup-failed`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'Cleanup failed');
      }

      const result = await response.json();
      success('Cleanup Completed', `Failed execution cleanup completed successfully for ${selectedVM}`);
      
      // Refresh VM context after cleanup
      if (onVMSelect) {
        onVMSelect(selectedVM);
      }
    } catch (error) {
      console.error('Failed execution cleanup error:', error);
      showError('Cleanup Failed', `Failed to cleanup failed execution: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }, [selectedVM, onVMSelect, success, showError]);

  const quickActions = React.useMemo((): QuickAction[] => {
    if (!selectedVM || !vmContext) return [];

    return [
      {
        id: 'replicate',
        label: 'Start Replication',
        icon: HiPlay,
        color: 'success',
        onClick: handleStartReplication,
        disabled: false // Temporarily disabled for testing stale jobs
      },
      {
        id: 'live-failover',
        label: 'Live Failover',
        icon: HiLightningBolt,
        color: 'failure',
        onClick: () => {
          setCurrentFailoverType('live');
          setPreFlightModalOpen(true);
        },
        disabled: false
      },
      {
        id: 'test-failover',
        label: 'Test Failover',
        icon: HiBeaker,
        color: 'purple',
        onClick: () => {
          setCurrentFailoverType('test');
          setPreFlightModalOpen(true);
        }
      },
      {
        id: 'cleanup',
        label: 'Rollback',
        icon: HiX,
        color: 'warning',
        onClick: () => {
          // Determine failover type based on current VM status
          const failoverType = vmContext?.context?.current_status === 'failed_over_live' ? 'live' : 'test';
          setCurrentFailoverType(failoverType);
          setRollbackModalOpen(true);
        }
      },
      {
        id: 'cleanup-failed',
        label: 'Cleanup Failed Job',
        icon: HiRefresh,
        color: 'gray',
        onClick: handleCleanupFailedExecution,
        disabled: false
      }
    ];
  }, [selectedVM, vmContext, handleStartReplication, handleCleanupFailedExecution]);

  const formatJobStatus = React.useCallback((status: string | null | undefined) => {
    if (!status) {
      return { color: 'gray' as const, icon: HiClock };
    }
    switch (status.toLowerCase()) {
      case 'completed':
        return { color: 'success' as const, icon: HiCheckCircle };
      case 'failed':
        return { color: 'failure' as const, icon: HiExclamationCircle };
      case 'replicating':
      case 'running':
        return { color: 'warning' as const, icon: HiClock };
      default:
        return { color: 'gray' as const, icon: HiClock };
    }
  }, []);

  const getModernStatusBadge = React.useCallback((status: string | null | undefined) => {
    if (!status) {
      return 'bg-gray-500/20 text-gray-300 border-gray-500/30';
    }
    
    switch (status.toLowerCase()) {
      case 'completed':
      case 'ready':
        return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30';
      case 'failed':
        return 'bg-red-500/20 text-red-300 border-red-500/30';
      case 'replicating':
      case 'transferring data':
        return 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30';
      case 'running':
        return 'bg-blue-500/20 text-blue-300 border-blue-500/30';
      case 'pending':
        return 'bg-amber-500/20 text-amber-300 border-amber-500/30';
      default:
        return 'bg-purple-500/20 text-purple-300 border-purple-500/30';
    }
  }, []);

  const formatProgress = React.useCallback((progress: number) => {
    return Math.round(Math.max(0, Math.min(100, progress || 0)));
  }, []);

  const formatETA = React.useCallback((etaSeconds: number) => {
    if (!etaSeconds || etaSeconds <= 0) return 'Unknown';
    
    const hours = Math.floor(etaSeconds / 3600);
    const minutes = Math.floor((etaSeconds % 3600) / 60);
    
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }, []);

  return (
    <div className="h-full bg-gradient-to-b from-slate-900/95 to-gray-900/95 backdrop-blur-xl border-l border-gray-700/50 flex flex-col">
      <div className="p-6 border-b border-gray-700/50">
        <div className="flex items-center space-x-3 mb-2">
          <div className="w-8 h-8 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-lg flex items-center justify-center">
            <HiLightningBolt className="w-4 h-4 text-white" />
          </div>
          <h2 className="text-xl font-bold bg-gradient-to-r from-cyan-400 to-blue-400 bg-clip-text text-transparent">
            VM Context
          </h2>
        </div>
        <p className="text-sm text-gray-400">
          {selectedVM ? 'Selected VM details' : 'Select a VM to view details'}
        </p>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Selected VM Info */}
        {selectedVM ? (
          <>
            {vmLoading ? (
              <Card>
                <div className="flex items-center justify-center p-8">
                  <Spinner size="lg" />
                  <span className="ml-3 text-gray-600 dark:text-gray-400">
                    Loading VM context...
                  </span>
                </div>
              </Card>
            ) : vmError ? (
              <Card>
                <div className="text-center p-4">
                  <ClientIcon className="w-12 h-12 text-red-500 mx-auto mb-2">
                    <HiExclamationCircle />
                  </ClientIcon>
                  <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                    Error Loading VM
                  </h3>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                    Failed to load context for {selectedVM}
                  </p>
                  <Button size="sm" color="blue">
                    <ClientIcon className="w-4 h-4 mr-2">
                      <HiRefresh />
                    </ClientIcon>
                    Retry
                  </Button>
                </div>
              </Card>
            ) : vmContext ? (
              <>
                {/* VM Quick Info */}
                <Card>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <h3 className="text-xl font-bold bg-gradient-to-r from-cyan-400 to-blue-400 bg-clip-text text-transparent">
                        {selectedVM}
                      </h3>
                      <div className={`px-3 py-1.5 rounded-lg border text-xs font-medium ${getModernStatusBadge(vmContext.context.current_status || 'Ready')}`}>
                        {vmContext.context.current_status || 'Ready'}
                      </div>
                    </div>

                    {/* Real-time Job Progress */}
                    {jobProgress ? (
                      <div className="space-y-3">
                        <div className="flex items-center justify-between text-sm">
                          <span className="text-gray-600 dark:text-gray-400">Job ID</span>
                          <span className="font-medium font-mono text-xs">
                            {jobProgress.id}
                          </span>
                        </div>
                        
                        <div className="flex items-center justify-between text-sm">
                          <span className="text-gray-600 dark:text-gray-400">Status</span>
                          <span className="font-medium">
                            {jobProgress.current_operation || jobProgress.status}
                          </span>
                        </div>

                        {/* Job Timing Information */}
                        {(jobProgress.started_at || jobProgress.created_at) && (
                          <div className="flex items-center justify-between text-sm">
                            <span className="text-gray-600 dark:text-gray-400">Started At</span>
                            <span className="font-medium text-gray-200 font-mono text-xs">
                              {formatStartTime(jobProgress.started_at || jobProgress.created_at)}
                            </span>
                          </div>
                        )}
                        
                        {(jobProgress.started_at || jobProgress.created_at) && (
                          <div className="flex items-center justify-between text-sm">
                            <span className="text-gray-600 dark:text-gray-400">Duration</span>
                            <span className="font-medium text-cyan-400 font-mono text-xs">
                              {formatDuration(jobProgress.started_at || jobProgress.created_at, jobProgress.completed_at)}
                            </span>
                          </div>
                        )}

                        {/* Dynamic Progress Bar */}
                        <div className="space-y-2">
                          <div className="flex items-center justify-between text-sm">
                            <span className="text-gray-600 dark:text-gray-400">Progress</span>
                            <span className="font-medium text-blue-600">
                              {jobProgress.progress_percent.toFixed(1)}%
                            </span>
                          </div>
                          <div className="w-full bg-gray-200 rounded-full h-3 dark:bg-gray-700">
                            <div 
                              className={`h-3 rounded-full transition-all duration-500 ${getProgressColor(jobProgress.status)}`}
                              style={{ width: `${Math.min(jobProgress.progress_percent, 100)}%` }}
                            />
                          </div>
                        </div>

                        {/* Transfer Details */}
                        {jobProgress.total_bytes > 0 && (
                          <div className="space-y-2 text-xs">
                            <div className="flex items-center justify-between">
                              <span className="text-gray-400">Transferred:</span>
                              <span className="text-gray-200 font-mono text-right">
                                {formatBytes(jobProgress.bytes_transferred)} / {formatBytes(jobProgress.total_bytes)}
                              </span>
                            </div>
                            {jobProgress.vma_throughput_mbps > 0 && (
                              <div className="flex items-center justify-between">
                                <span className="text-gray-400">Speed:</span>
                                <span className="text-gray-200 font-mono">
                                  {jobProgress.vma_throughput_mbps.toFixed(1)} MB/s
                                </span>
                              </div>
                            )}
                            {jobProgress.vma_eta_seconds && jobProgress.vma_eta_seconds > 0 && (
                              <div className="flex items-center justify-between">
                                <span className="text-gray-400">ETA:</span>
                                <span className="text-gray-200 font-mono">
                                  {formatETA(jobProgress.vma_eta_seconds)}
                                </span>
                              </div>
                            )}
                          </div>
                        )}
                        
                        <div className="flex items-center justify-between text-xs pt-2 border-t border-gray-700/50">
                          <span className="text-gray-400">Last Update:</span>
                          <span className="text-gray-200 font-mono text-right">
                            {new Date(jobProgress.updated_at).toLocaleString()}
                          </span>
                        </div>
                      </div>
                    ) : vmContext.context.current_job_id && (
                      <div className="space-y-2">
                        <div className="flex items-center justify-between text-sm">
                          <span className="text-gray-600 dark:text-gray-400">Job ID</span>
                          <span className="font-medium font-mono text-xs">
                            {vmContext.context.current_job_id}
                          </span>
                        </div>
                        <div className="flex items-center justify-between text-sm">
                          <span className="text-gray-600 dark:text-gray-400">Status</span>
                          <span className="font-medium">
                            {vmContext.context.current_status}
                          </span>
                        </div>
                        
                        {vmContext.context.last_job_at && (
                          <div className="flex items-center justify-between text-xs">
                            <span className="text-gray-500">Last Activity:</span>
                            <span>{new Date(vmContext.context.last_job_at).toLocaleString()}</span>
                          </div>
                        )}
                      </div>
                    )}

                    {/* VM Specs */}
                    <div className="space-y-2 text-sm">
                      <div className="flex items-center justify-between">
                        <span className="text-gray-400">CPU:</span>
                        <span className="font-medium text-gray-200">
                          {vmContext.context.cpu_count ? `${vmContext.context.cpu_count} vCPUs` : 'N/A'}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-gray-400">RAM:</span>
                        <span className="font-medium text-gray-200">
                          {vmContext.context.memory_mb 
                            ? `${Math.round(vmContext.context.memory_mb / 1024)} GB` 
                            : 'N/A'
                          }
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-gray-400">Power:</span>
                        <span className="font-medium text-gray-200">
                          {vmContext.context.power_state || 'N/A'}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-gray-400">OS:</span>
                        <span className="font-medium text-gray-200 truncate ml-2">
                          {vmContext.context.os_type || 'N/A'}
                        </span>
                      </div>
                    </div>
                  </div>
                </Card>

                {/* Unified Recent Operations - Shows ALL job types */}
                {selectedVM && (
                  <UnifiedJobList 
                    vmName={selectedVM}
                    onJobClick={(job) => {
                      if (job.status === 'failed') {
                        setSelectedJob(job);
                        setIsJobErrorModalOpen(true);
                      }
                    }}
                  />
                )}

                {/* Quick Actions */}
                <Card>
                  <h4 className="text-md font-medium text-gray-900 dark:text-white mb-3">
                    Quick Actions
                  </h4>
                  
                  <div className="space-y-2">
                    {quickActions.map((action) => (
                      <Button
                        key={action.id}
                        size="sm"
                        color={action.color}
                        className="w-full justify-start"
                        onClick={action.onClick}
                        disabled={action.disabled}
                      >
                        <ClientIcon className="w-4 h-4 mr-2">
                          <action.icon />
                        </ClientIcon>
                        {action.label}
                      </Button>
                    ))}
                  </div>
                </Card>
              </>
            ) : null}
          </>
        ) : (
          /* No VM Selected */
          <div className="bg-gradient-to-br from-slate-800/60 to-gray-800/60 backdrop-blur-sm border border-gray-700/50 rounded-xl p-8">
            <div className="text-center">
              <div className="w-16 h-16 bg-gradient-to-br from-gray-600 to-gray-700 rounded-full flex items-center justify-center mx-auto mb-6 border border-gray-600/30">
                <HiCheckCircle className="w-8 h-8 text-gray-400" />
              </div>
              <h3 className="text-xl font-semibold text-gray-200 mb-3">
                No VM Selected
              </h3>
              <p className="text-gray-400 leading-relaxed">
                Select a virtual machine from the main view to see detailed context, progress, and available actions.
              </p>
            </div>
          </div>
        )}

        {/* System Health */}
        <div className="bg-gradient-to-br from-slate-800/60 to-gray-800/60 backdrop-blur-sm border border-gray-700/50 rounded-xl p-6">
          <div className="flex items-center space-x-2 mb-4">
            <div className="w-6 h-6 bg-gradient-to-br from-emerald-500 to-green-600 rounded-lg flex items-center justify-center">
              <HiCheckCircle className="w-3 h-3 text-white" />
            </div>
            <h4 className="text-lg font-semibold text-gray-200">
              System Health
            </h4>
          </div>
          
          {healthLoading ? (
            <div className="flex items-center justify-center p-4">
              <Spinner size="sm" />
              <span className="ml-2 text-gray-600 dark:text-gray-400 text-sm">
                Loading...
              </span>
            </div>
          ) : systemHealth ? (
            <div className="space-y-2 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-gray-600 dark:text-gray-400">Active Jobs:</span>
                <div className="px-2 py-1 rounded text-xs font-medium bg-blue-500/20 text-blue-300 border-blue-500/30 border">
                  {systemHealth.active_jobs || 0}
                </div>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-gray-600 dark:text-gray-400">VMA Status:</span>
                <div className={`px-2 py-1 rounded text-xs font-medium ${systemHealth.vma_healthy ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30' : 'bg-red-500/20 text-red-300 border-red-500/30'} border`}>
                  {systemHealth.vma_healthy ? 'Healthy' : 'Unhealthy'}
                </div>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-gray-600 dark:text-gray-400">Volume Daemon:</span>
                <div className={`px-2 py-1 rounded text-xs font-medium ${systemHealth.volume_daemon_healthy ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30' : 'bg-red-500/20 text-red-300 border-red-500/30'} border`}>
                  {systemHealth.volume_daemon_healthy ? 'Healthy' : 'Unhealthy'}
                </div>
              </div>
            </div>
          ) : (
            <p className="text-sm text-gray-400 text-center py-4">
              Health data unavailable
            </p>
          )}
        </div>
      </div>

      {/* Enhanced Failover Modals */}
      {selectedVM && vmContext && (
        <>
          {/* Pre-flight Configuration Modal */}
          <PreFlightConfiguration
            isOpen={preFlightModalOpen}
            onClose={() => setPreFlightModalOpen(false)}
            onConfirm={handleUnifiedFailover}
            vmName={selectedVM}
            failoverType={currentFailoverType}
            vmContext={{
              context_id: vmContext.context.context_id,
              vmware_vm_id: vmContext.context.vmware_vm_id,
              vm_name: vmContext.context.vm_name
            }}
          />

          {/* Rollback Decision Modal */}
          <RollbackDecision
            isOpen={rollbackModalOpen}
            onClose={() => setRollbackModalOpen(false)}
            onConfirm={handleRollback}
            vmName={selectedVM}
            failoverType={currentFailoverType}
            vmContext={vmContext}
          />

          {/* Integrated Active Operation Progress - Part of VM Context */}
          {activeJobId && selectedVM && activeJobId.includes(selectedVM) && (
            <div className="mb-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Active Operation
                </h3>
                <button
                  onClick={() => setActiveJobWithPersistence(null)}
                  className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 text-sm"
                >
                  Dismiss
                </button>
              </div>
              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
                <UnifiedProgressTracker
                  jobId={activeJobId}
                  failoverType={currentFailoverType}
                  vmName={selectedVM}
                  onComplete={(isSuccess) => {
                    const isRollbackJob = activeJobId?.startsWith('rollback-');
                    if (isSuccess) {
                      if (isRollbackJob) {
                        success('Rollback completed', 'Rollback operation completed successfully');
                      } else {
                        success('Failover completed', 'Failover operation completed successfully');
                      }
                    } else {
                      if (isRollbackJob) {
                        showError('Rollback failed', 'Rollback operation failed');
                      } else {
                        showError('Failover failed', 'Failover operation failed');
                      }
                    }
                    setActiveJobWithPersistence(null);
                  }}
                  onRollbackRequest={() => {
                    setRollbackModalOpen(true);
                  }}
                />
              </div>
            </div>
          )}
        </>
      )}

      {/* Job Error Details Modal */}
      {selectedJob && (
        <JobErrorDetailsModal
          isOpen={isJobErrorModalOpen}
          onClose={() => {
            setIsJobErrorModalOpen(false);
            setSelectedJob(null);
          }}
          job={selectedJob}
          vmName={selectedVM || 'Unknown'}
          onTryLiveFailover={() => {
            setCurrentFailoverType('live');
            setPreFlightModalOpen(true);
            setIsJobErrorModalOpen(false);
          }}
          showAdminDetails={false}
        />
      )}

    </div>
  );
});

RightContextPanelInner.displayName = 'RightContextPanelInner';

// Export wrapped with DecisionAuditProvider
export const RightContextPanel: React.FC<RightContextPanelProps> = (props) => {
  return (
    <DecisionAuditProvider>
      <RightContextPanelInner {...props} />
    </DecisionAuditProvider>
  );
};