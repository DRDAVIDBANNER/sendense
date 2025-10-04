'use client';

import React from 'react';
import { Button, Badge, Spinner, Alert } from 'flowbite-react';
import { 
  HiRefresh, 
  HiPlay, 
  HiExclamationCircle, 
  HiServer,
  HiClock,
  HiLightningBolt,
  HiDatabase,
  HiChartBar,
  HiCheckCircle,
  HiXCircle
} from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';
import { useVMContexts, VMContextListItem } from '../../hooks/useVMContext';
import { useJobProgress, formatBytes, getProgressColor } from '../../hooks/useJobProgress';

export interface ModernVMTableProps {
  onVMSelect: (vmName: string) => void;
}

const ModernVMTable = React.memo(({ onVMSelect }: ModernVMTableProps) => {
  const { data: vmContexts, isLoading, error, refetch } = useVMContexts();
  const { data: allJobs } = useJobProgress();

  const handleVMSelect = React.useCallback((vmName: string) => {
    onVMSelect(vmName);
  }, [onVMSelect]);

  const handleRefresh = React.useCallback(() => {
    refetch();
  }, [refetch]);

  const getStatusBadge = React.useCallback((status: string | null | undefined) => {
    if (!status) {
      return { 
        color: 'bg-gray-500/20 text-gray-300 border-gray-500/30', 
        text: 'Unknown',
        icon: HiExclamationCircle
      };
    }
    
    switch (status.toLowerCase()) {
      case 'completed':
        return { 
          color: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30', 
          text: 'Completed',
          icon: HiCheckCircle
        };
      case 'failed':
        return { 
          color: 'bg-red-500/20 text-red-300 border-red-500/30', 
          text: 'Failed',
          icon: HiXCircle
        };
      case 'replicating':
        return { 
          color: 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30', 
          text: 'Replicating',
          icon: HiLightningBolt
        };
      case 'running':
        return { 
          color: 'bg-blue-500/20 text-blue-300 border-blue-500/30', 
          text: 'Running',
          icon: HiPlay
        };
      case 'pending':
        return { 
          color: 'bg-amber-500/20 text-amber-300 border-amber-500/30', 
          text: 'Pending',
          icon: HiClock
        };
      default:
        return { 
          color: 'bg-purple-500/20 text-purple-300 border-purple-500/30', 
          text: status,
          icon: HiDatabase
        };
    }
  }, []);

  const formatProgress = React.useCallback((job?: VMContextListItem['current_job']) => {
    if (!job || !job.progress_percentage) {
      return null;
    }
    
    const progress = Math.round(job.progress_percentage);
    const getProgressColorClass = (p: number) => {
      if (p >= 90) return 'from-emerald-500 to-green-400';
      if (p >= 70) return 'from-cyan-500 to-blue-400';
      if (p >= 40) return 'from-blue-500 to-indigo-400';
      return 'from-indigo-500 to-purple-400';
    };

    return (
      <div className="space-y-1">
        <div className="flex items-center justify-between text-xs">
          <span className="text-gray-300">{job.current_operation || 'Processing'}</span>
          <span className="text-cyan-300 font-mono">{progress}%</span>
        </div>
        <div className="w-full bg-gray-700/50 rounded-full h-1.5 backdrop-blur-sm">
          <div 
            className={`h-1.5 rounded-full bg-gradient-to-r ${getProgressColorClass(progress)} transition-all duration-500 ease-out`}
            style={{ width: `${progress}%` }}
          />
        </div>
        {job.vma_eta_seconds && (
          <div className="text-xs text-gray-400">
            ETA: {Math.round(job.vma_eta_seconds / 60)}m
          </div>
        )}
      </div>
    );
  }, []);

  const formatLastActivity = React.useCallback((dateString: string) => {
    try {
      const date = new Date(dateString);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / 60000);
      const diffHours = Math.floor(diffMins / 60);
      const diffDays = Math.floor(diffHours / 24);

      if (diffDays > 0) {
        return `${diffDays}d ago`;
      } else if (diffHours > 0) {
        return `${diffHours}h ago`;
      } else if (diffMins > 0) {
        return `${diffMins}m ago`;
      } else {
        return 'Just now';
      }
    } catch {
      return 'Unknown';
    }
  }, []);

  if (isLoading) {
    return (
      <div className="min-h-[600px] bg-gradient-to-br from-slate-900 via-gray-900 to-slate-900 rounded-2xl backdrop-blur-xl border border-gray-700/50 shadow-2xl">
        <div className="flex items-center justify-center h-96">
          <div className="text-center">
            <div className="relative">
              <Spinner size="xl" className="text-cyan-400" />
              <div className="absolute inset-0 animate-ping">
                <div className="w-8 h-8 bg-cyan-400/20 rounded-full"></div>
              </div>
            </div>
            <p className="mt-6 text-gray-300 font-medium">Loading virtual machines...</p>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-[600px] bg-gradient-to-br from-slate-900 via-gray-900 to-slate-900 rounded-2xl backdrop-blur-xl border border-gray-700/50 shadow-2xl">
        <div className="flex items-center justify-center h-96">
          <div className="text-center max-w-md">
            <div className="w-16 h-16 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-4 border border-red-500/30">
              <HiExclamationCircle className="w-8 h-8 text-red-400" />
            </div>
            <h3 className="text-xl font-semibold text-red-300 mb-2">Connection Error</h3>
            <p className="text-gray-400 mb-6">Failed to load virtual machines. Please check your connection.</p>
            <Button 
              onClick={handleRefresh} 
              className="bg-gradient-to-r from-red-600 to-red-500 hover:from-red-500 hover:to-red-400 border-0 text-white"
            >
              <HiRefresh className="mr-2 h-4 w-4" />
              Retry Connection
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (!vmContexts || vmContexts.length === 0) {
    return (
      <div className="min-h-[600px] bg-gradient-to-br from-slate-900 via-gray-900 to-slate-900 rounded-2xl backdrop-blur-xl border border-gray-700/50 shadow-2xl">
        <div className="flex items-center justify-center h-96">
          <div className="text-center max-w-md">
            <div className="w-20 h-20 bg-gray-700/30 rounded-full flex items-center justify-center mx-auto mb-6 border border-gray-600/30">
              <HiServer className="w-10 h-10 text-gray-400" />
            </div>
            <h3 className="text-2xl font-semibold text-gray-200 mb-3">No Virtual Machines</h3>
            <p className="text-gray-400 mb-8">No VMs are currently available for migration. Try discovering VMs first.</p>
            <Button 
              onClick={handleRefresh} 
              className="bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 border-0 text-white"
            >
              <HiRefresh className="mr-2 h-4 w-4" />
              Refresh List
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Modern Header */}
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h1 className="text-3xl font-bold bg-gradient-to-r from-cyan-400 via-blue-400 to-purple-400 bg-clip-text text-transparent">
            Virtual Machines
          </h1>
          <div className="flex items-center space-x-4">
            <span className="text-gray-400">
              {vmContexts.length} VM{vmContexts.length !== 1 ? 's' : ''} available
            </span>
            <div className="w-2 h-2 bg-emerald-400 rounded-full animate-pulse"></div>
            <span className="text-emerald-400 text-sm font-medium">Live Status</span>
          </div>
        </div>
        
        <Button 
          onClick={handleRefresh} 
          className="bg-gradient-to-r from-gray-700 to-gray-600 hover:from-gray-600 hover:to-gray-500 border border-gray-600/50 text-gray-200"
        >
          <HiRefresh className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      {/* Modern VM Grid */}
      <div className="grid gap-4">
        {vmContexts.map((vm) => {
          const statusInfo = getStatusBadge(vm.current_job?.status || vm.status);
          const vmJob = allJobs 
            ? allJobs
                .filter(job => job.vm_name === vm.vm_name)
                .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())[0]
            : null;
          
          return (
            <div 
              key={vm.vm_name}
              className="group relative bg-gradient-to-r from-slate-800/80 via-gray-800/80 to-slate-800/80 backdrop-blur-xl border border-gray-700/50 rounded-xl p-6 hover:border-cyan-500/50 hover:shadow-xl hover:shadow-cyan-500/10 cursor-pointer transition-all duration-300 hover:scale-[1.02]"
              onClick={() => handleVMSelect(vm.vm_name)}
            >
              {/* Glow effect on hover */}
              <div className="absolute inset-0 bg-gradient-to-r from-cyan-600/0 via-cyan-600/5 to-blue-600/0 rounded-xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
              
              <div className="relative z-10">
                <div className="flex items-center justify-between mb-4">
                  {/* VM Name and Icon */}
                  <div className="flex items-center space-x-3">
                    <div className="w-10 h-10 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-lg flex items-center justify-center shadow-lg">
                      <HiServer className="w-5 h-5 text-white" />
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-gray-100 group-hover:text-cyan-300 transition-colors">
                        {vm.vm_name}
                      </h3>
                      <div className="flex items-center space-x-2 text-sm text-gray-400">
                        <HiDatabase className="w-3 h-3" />
                        <span>{vm.job_count} job{vm.job_count !== 1 ? 's' : ''}</span>
                      </div>
                    </div>
                  </div>

                  {/* Status Badge */}
                  <div className={`px-3 py-1.5 rounded-lg border text-xs font-medium flex items-center space-x-1.5 ${statusInfo.color}`}>
                    <statusInfo.icon className="w-3 h-3" />
                    <span>{statusInfo.text}</span>
                  </div>
                </div>

                {/* Progress Section */}
                {vm.current_job && formatProgress(vm.current_job) && (
                  <div className="mb-4 p-3 bg-gray-900/40 rounded-lg border border-gray-700/30">
                    {formatProgress(vm.current_job)}
                  </div>
                )}

                {/* Stats Row */}
                <div className="grid grid-cols-3 gap-4 text-sm">
                  <div className="text-center">
                    <div className="text-gray-400 flex items-center justify-center mb-1">
                      <HiChartBar className="w-4 h-4 mr-1" />
                      Progress
                    </div>
                    <div className="text-cyan-300 font-mono">
                      {vm.current_job?.progress_percentage ? 
                        `${Math.round(vm.current_job.progress_percentage)}%` : 
                        '—'
                      }
                    </div>
                  </div>
                  
                  <div className="text-center">
                    <div className="text-gray-400 flex items-center justify-center mb-1">
                      <HiClock className="w-4 h-4 mr-1" />
                      Last Activity
                    </div>
                    <div className="text-gray-300">
                      {formatLastActivity(vm.last_activity)}
                    </div>
                  </div>
                  
                  <div className="text-center">
                    <div className="text-gray-400 flex items-center justify-center mb-1">
                      <HiLightningBolt className="w-4 h-4 mr-1" />
                      Actions
                    </div>
                    <Button 
                      size="xs" 
                      className="bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 border-0 text-white text-xs px-3"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleVMSelect(vm.vm_name);
                      }}
                    >
                      Manage
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Call to Action */}
      <div className="text-center py-6">
        <p className="text-gray-400 text-sm">
          Click on any VM to view detailed context, progress, and available actions
        </p>
      </div>
    </div>
  );
});

ModernVMTable.displayName = 'ModernVMTable';

export default ModernVMTable;
