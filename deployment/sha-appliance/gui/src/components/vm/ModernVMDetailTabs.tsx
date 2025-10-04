'use client';

import React, { useState, useCallback } from 'react';
import { Button, Spinner, Alert, Badge, Tabs } from 'flowbite-react';
import { 
  HiArrowLeft, 
  HiExclamationCircle, 
  HiRefresh,
  HiServer,
  HiClock,
  HiLightningBolt,
  HiDatabase,
  HiChartBar,
  HiCheckCircle,
  HiXCircle,
  HiPlay
} from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';
import { useVMContext } from '../../hooks/useVMContext';
import { useVMJobProgress } from '../../hooks/useJobProgress';

export interface ModernVMDetailTabsProps {
  vmName: string;
  onBack: () => void;
}

export const ModernVMDetailTabs = React.memo(({ vmName, onBack }: ModernVMDetailTabsProps) => {
  const { data: vmContext, isLoading, error, refetch } = useVMContext(vmName);
  const { data: jobProgress, isLoading: jobLoading } = useVMJobProgress(vmName);

  const handleBack = React.useCallback(() => {
    onBack();
  }, [onBack]);

  const handleRefresh = React.useCallback(() => {
    refetch();
  }, [refetch]);

  const formatBytes = React.useCallback((bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }, []);

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
      default:
        return { 
          color: 'bg-purple-500/20 text-purple-300 border-purple-500/30', 
          text: status,
          icon: HiDatabase
        };
    }
  }, []);

  const getProgressColor = (progress: number) => {
    if (progress >= 90) return 'from-emerald-500 to-green-400';
    if (progress >= 70) return 'from-cyan-500 to-blue-400';
    if (progress >= 40) return 'from-blue-500 to-indigo-400';
    return 'from-indigo-500 to-purple-400';
  };

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
            <p className="mt-6 text-gray-300 font-medium">Loading VM details...</p>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center space-x-4">
          <Button 
            onClick={handleBack}
            className="bg-gradient-to-r from-gray-700 to-gray-600 hover:from-gray-600 hover:to-gray-500 border border-gray-600/50 text-gray-200"
          >
            <HiArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>
          <div>
            <h1 className="text-3xl font-bold bg-gradient-to-r from-red-400 via-red-400 to-red-500 bg-clip-text text-transparent">
              Error Loading VM
            </h1>
            <p className="text-gray-400">Failed to load VM details</p>
          </div>
        </div>

        {/* Error Content */}
        <div className="bg-gradient-to-br from-red-900/20 via-red-800/20 to-red-900/20 backdrop-blur-xl border border-red-700/50 rounded-2xl p-8">
          <div className="text-center">
            <div className="w-16 h-16 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-6 border border-red-500/30">
              <HiExclamationCircle className="w-8 h-8 text-red-400" />
            </div>
            <h3 className="text-xl font-semibold text-red-300 mb-3">Connection Error</h3>
            <p className="text-gray-400 mb-6">Failed to load VM details: {error.message}</p>
            <div className="flex justify-center space-x-3">
              <Button 
                onClick={handleBack}
                className="bg-gradient-to-r from-gray-700 to-gray-600 hover:from-gray-600 hover:to-gray-500 border-0 text-white"
              >
                <HiArrowLeft className="mr-2 h-4 w-4" />
                Back
              </Button>
              <Button 
                onClick={handleRefresh}
                className="bg-gradient-to-r from-red-600 to-red-500 hover:from-red-500 hover:to-red-400 border-0 text-white"
              >
                <HiRefresh className="mr-2 h-4 w-4" />
                Try Again
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (!vmContext) {
    return (
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center space-x-4">
          <Button 
            onClick={handleBack}
            className="bg-gradient-to-r from-gray-700 to-gray-600 hover:from-gray-600 hover:to-gray-500 border border-gray-600/50 text-gray-200"
          >
            <HiArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>
          <div>
            <h1 className="text-3xl font-bold bg-gradient-to-r from-gray-400 via-gray-400 to-gray-500 bg-clip-text text-transparent">
              VM Not Found
            </h1>
            <p className="text-gray-400">Virtual machine not found</p>
          </div>
        </div>

        {/* Not Found Content */}
        <div className="bg-gradient-to-br from-slate-800/60 to-gray-800/60 backdrop-blur-sm border border-gray-700/50 rounded-xl p-8">
          <div className="text-center">
            <div className="w-16 h-16 bg-gray-700/30 rounded-full flex items-center justify-center mx-auto mb-6 border border-gray-600/30">
              <HiServer className="w-8 h-8 text-gray-400" />
            </div>
            <h3 className="text-xl font-semibold text-gray-200 mb-3">VM Not Found</h3>
            <p className="text-gray-400 mb-6">The virtual machine "{vmName}" was not found.</p>
            <Button 
              onClick={handleBack}
              className="bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 border-0 text-white"
            >
              <HiArrowLeft className="mr-2 h-4 w-4" />
              Back to VM List
            </Button>
          </div>
        </div>
      </div>
    );
  }

  const currentJob = jobProgress || vmContext.current_job;
  const statusInfo = getStatusBadge(currentJob?.status || vmContext.context.current_status);

  return (
    <div className="space-y-6">
      {/* Modern Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button 
            onClick={handleBack}
            className="bg-gradient-to-r from-gray-700 to-gray-600 hover:from-gray-600 hover:to-gray-500 border border-gray-600/50 text-gray-200"
          >
            <HiArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>
          <div className="flex items-center space-x-3">
            <div className="w-12 h-12 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-xl flex items-center justify-center shadow-lg">
              <HiServer className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-cyan-400 via-blue-400 to-purple-400 bg-clip-text text-transparent">
                {vmName}
              </h1>
              <p className="text-gray-400">Virtual Machine Details</p>
            </div>
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

      {/* Status Overview Card */}
      <div className="bg-gradient-to-r from-slate-800/80 via-gray-800/80 to-slate-800/80 backdrop-blur-xl border border-gray-700/50 rounded-xl p-6">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {/* Status */}
          <div className="text-center">
            <div className="flex justify-center mb-3">
              <div className={`px-4 py-2 rounded-lg border text-sm font-medium flex items-center space-x-2 ${statusInfo.color}`}>
                <statusInfo.icon className="w-4 h-4" />
                <span>{statusInfo.text}</span>
              </div>
            </div>
            <p className="text-gray-400 text-sm">Current Status</p>
          </div>

          {/* Progress */}
          {currentJob && (
            <div className="text-center">
              <div className="mb-3">
                <div className="text-2xl font-bold text-cyan-300">
                  {Math.round(currentJob.progress_percent || 0)}%
                </div>
                <div className="w-full bg-gray-700/50 rounded-full h-2 mt-2">
                  <div 
                    className={`h-2 rounded-full bg-gradient-to-r ${getProgressColor(currentJob.progress_percent || 0)} transition-all duration-500`}
                    style={{ width: `${Math.min(currentJob.progress_percent || 0, 100)}%` }}
                  />
                </div>
              </div>
              <p className="text-gray-400 text-sm">Progress</p>
            </div>
          )}

          {/* Operation */}
          <div className="text-center">
            <div className="mb-3">
              <div className="text-lg font-semibold text-gray-200">
                {currentJob?.current_operation || 'Idle'}
              </div>
              {currentJob?.vma_eta_seconds && (
                <div className="text-sm text-cyan-400">
                  ETA: {Math.floor(currentJob.vma_eta_seconds / 60)}m
                </div>
              )}
            </div>
            <p className="text-gray-400 text-sm">Current Operation</p>
          </div>
        </div>
      </div>

      {/* VM Specifications */}
      <div className="bg-gradient-to-br from-slate-800/60 to-gray-800/60 backdrop-blur-sm border border-gray-700/50 rounded-xl p-6">
        <div className="flex items-center space-x-3 mb-6">
            <div className="w-8 h-8 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-lg flex items-center justify-center">
              <HiServer className="w-4 h-4 text-white" />
            </div>
          <h3 className="text-xl font-bold text-gray-200">Virtual Machine Specifications</h3>
        </div>
        
        <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
          {/* CPU */}
          <div className="text-center">
            <div className="w-12 h-12 bg-blue-500/20 rounded-lg flex items-center justify-center mx-auto mb-3 border border-blue-500/30">
              <HiLightningBolt className="w-6 h-6 text-blue-400" />
            </div>
            <div className="text-xl font-bold text-gray-200">
              {vmContext.context.cpu_count || 'N/A'}
            </div>
            <div className="text-sm text-gray-400">vCPUs</div>
          </div>

          {/* Memory */}
          <div className="text-center">
            <div className="w-12 h-12 bg-green-500/20 rounded-lg flex items-center justify-center mx-auto mb-3 border border-green-500/30">
              <HiDatabase className="w-6 h-6 text-green-400" />
            </div>
            <div className="text-xl font-bold text-gray-200">
              {vmContext.context.memory_mb 
                ? `${(vmContext.context.memory_mb / 1024).toFixed(1)} GB`
                : 'N/A'
              }
            </div>
            <div className="text-sm text-gray-400">Memory</div>
          </div>

          {/* OS */}
          <div className="text-center">
            <div className="w-12 h-12 bg-purple-500/20 rounded-lg flex items-center justify-center mx-auto mb-3 border border-purple-500/30">
              <HiServer className="w-6 h-6 text-purple-400" />
            </div>
            <div className="text-lg font-bold text-gray-200">
              {vmContext.context.guest_os || vmContext.context.os_type || 'N/A'}
            </div>
            <div className="text-sm text-gray-400">Guest OS</div>
          </div>

          {/* Power State */}
          <div className="text-center">
            <div className={`w-12 h-12 rounded-lg flex items-center justify-center mx-auto mb-3 border ${
              vmContext.context.power_state === 'poweredOn' 
                ? 'bg-emerald-500/20 border-emerald-500/30' 
                : 'bg-red-500/20 border-red-500/30'
            }`}>
              <HiLightningBolt className={`w-6 h-6 ${
                vmContext.context.power_state === 'poweredOn' ? 'text-emerald-400' : 'text-red-400'
              }`} />
            </div>
            <div className="text-lg font-bold text-gray-200">
              {vmContext.context.power_state === 'poweredOn' ? 'On' : 'Off'}
            </div>
            <div className="text-sm text-gray-400">Power State</div>
          </div>
        </div>
      </div>

      {/* Performance Data */}
      {currentJob && (
        <div className="bg-gradient-to-br from-slate-800/60 to-gray-800/60 backdrop-blur-sm border border-gray-700/50 rounded-xl p-6">
          <div className="flex items-center space-x-3 mb-6">
            <div className="w-8 h-8 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-lg flex items-center justify-center">
              <HiChartBar className="w-4 h-4 text-white" />
            </div>
            <h3 className="text-xl font-bold text-gray-200">Migration Performance</h3>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Transferred */}
            <div className="text-center">
              <div className="text-2xl font-bold text-cyan-300">
                {formatBytes(currentJob.bytes_transferred || 0)}
              </div>
              <div className="text-sm text-gray-400">Transferred</div>
              {currentJob.total_bytes && (
                <div className="text-xs text-gray-500 mt-1">
                  of {formatBytes(currentJob.total_bytes)}
                </div>
              )}
            </div>

            {/* Throughput */}
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-300">
                {currentJob.vma_throughput_mbps ? `${currentJob.vma_throughput_mbps.toFixed(1)}` : '0'} 
              </div>
              <div className="text-sm text-gray-400">MB/s</div>
            </div>

            {/* ETA */}
            <div className="text-center">
              <div className="text-2xl font-bold text-purple-300">
                {currentJob.vma_eta_seconds ? Math.floor(currentJob.vma_eta_seconds / 60) : '—'}
              </div>
              <div className="text-sm text-gray-400">Minutes Remaining</div>
            </div>
          </div>
        </div>
      )}

    </div>
  );
});

ModernVMDetailTabs.displayName = 'ModernVMDetailTabs';

export default ModernVMDetailTabs;
