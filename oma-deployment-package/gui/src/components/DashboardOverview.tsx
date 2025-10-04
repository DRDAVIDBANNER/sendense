'use client';

import React from 'react';
import { Card, Badge } from 'flowbite-react';
import { 
  HiServer, 
  HiCloud, 
  HiLightningBolt, 
  HiDocumentText
} from 'react-icons/hi';
import { ClientIcon } from './ClientIcon';
import { useVMContexts } from '../hooks/useVMContext';
import { useSystemHealth } from '../hooks/useSystemHealth';

export interface DashboardOverviewProps {
  onVMSelect: (vmName: string) => void;
}

export const DashboardOverview = React.memo(({ onVMSelect }: DashboardOverviewProps) => {
  const { data: vmContexts } = useVMContexts();
  const { data: systemHealth } = useSystemHealth();

  const stats = React.useMemo(() => {
    if (!vmContexts) return { total: 0, active: 0, completed: 0, failed: 0 };
    
    return {
      total: vmContexts.length,
      active: vmContexts.filter(vm => vm.status === 'replicating').length,
      completed: vmContexts.filter(vm => vm.status === 'completed').length,
      failed: vmContexts.filter(vm => vm.status === 'failed').length
    };
  }, [vmContexts]);

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
          🚀 MigrateKit OSSEA Dashboard
        </h1>
        <p className="text-gray-600 dark:text-gray-300">
          VMware to OSSEA Migration Management - System Overview
        </p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <div className="flex items-center">
            <div className="p-3 rounded-full bg-blue-100 dark:bg-blue-900 mr-4">
              <ClientIcon className="w-6 h-6 text-blue-600 dark:text-blue-300">
                <HiServer />
              </ClientIcon>
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                Total VMs
              </h3>
              <p className="text-2xl font-bold text-blue-600 dark:text-blue-400">
                {stats.total}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="p-3 rounded-full bg-yellow-100 dark:bg-yellow-900 mr-4">
              <ClientIcon className="w-6 h-6 text-yellow-600 dark:text-yellow-300">
                <HiCloud />
              </ClientIcon>
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                Active Jobs
              </h3>
              <p className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">
                {stats.active}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="p-3 rounded-full bg-green-100 dark:bg-green-900 mr-4">
              <ClientIcon className="w-6 h-6 text-green-600 dark:text-green-300">
                <HiLightningBolt />
              </ClientIcon>
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                Completed
              </h3>
              <p className="text-2xl font-bold text-green-600 dark:text-green-400">
                {stats.completed}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="p-3 rounded-full bg-red-100 dark:bg-red-900 mr-4">
              <ClientIcon className="w-6 h-6 text-red-600 dark:text-red-300">
                <HiDocumentText />
              </ClientIcon>
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                System Health
              </h3>
              <Badge color={systemHealth?.oma_healthy ? 'success' : 'failure'}>
                {systemHealth?.oma_healthy ? 'Healthy' : 'Issues'}
              </Badge>
            </div>
          </div>
        </Card>
      </div>

      {/* Recent Activity */}
      <Card>
        <h2 className="text-xl font-bold text-gray-900 dark:text-white mb-4">
          Recent VM Activity
        </h2>
        
        {vmContexts && vmContexts.length > 0 ? (
          <div className="space-y-3">
            {vmContexts.slice(0, 5).map((vm) => (
              <div 
                key={vm.vm_name}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
                onClick={() => onVMSelect(vm.vm_name)}
              >
                <div className="flex items-center space-x-3">
                  <ClientIcon className="w-5 h-5 text-gray-500 dark:text-gray-400">
                    <HiServer />
                  </ClientIcon>
                  <div>
                    <p className="font-medium text-gray-900 dark:text-white">
                      {vm.vm_name}
                    </p>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                      {vm.job_count} jobs • Last activity: {vm.last_activity ? new Date(vm.last_activity).toLocaleString() : 'Unknown'}
                    </p>
                  </div>
                </div>
                <Badge color={vm.status === 'replicating' ? 'warning' : vm.status === 'completed' ? 'success' : 'gray'}>
                  {vm.status}
                </Badge>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8">
            <ClientIcon className="w-12 h-12 text-gray-400 mx-auto mb-4">
              <HiServer />
            </ClientIcon>
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
              No VMs Available
            </h3>
            <p className="text-gray-600 dark:text-gray-400">
              No virtual machines are currently available for migration.
            </p>
          </div>
        )}
      </Card>
    </div>
  );
});

DashboardOverview.displayName = 'DashboardOverview';










