'use client';

import React from 'react';
import ModernVMTable from '../vm/ModernVMTable';
import ModernVMDetailTabs from '../vm/ModernVMDetailTabs';
import { DashboardOverview } from '../DashboardOverview';
import { JobListView } from '../jobs/JobListView';
import { FailoverManagement } from '../failover/FailoverManagement';
import { NetworkMappingView } from '../network/NetworkMappingView';
import { SystemLogs } from '../logs/SystemLogs';
import { SettingsView } from '../settings/SettingsView';
import { DiscoveryView } from '../discovery/DiscoveryView';

export interface MainContentAreaProps {
  section: string;
  selectedVM: string | null;
  onVMSelect: (vmName: string | null) => void;
  children?: React.ReactNode;
}

export const MainContentArea = React.memo(({ 
  section, 
  selectedVM, 
  onVMSelect, 
  children 
}: MainContentAreaProps) => {
  
  const renderContent = React.useCallback(() => {
    // If custom children are provided (for backward compatibility), render them
    if (children) {
      return children;
    }

    switch (section) {
      case 'dashboard':
        return <DashboardOverview onVMSelect={onVMSelect} />;
      
      case 'discovery':
        return <DiscoveryView onVMSelect={onVMSelect} />;
      
      case 'virtual-machines':
        return (
          <div className="h-full flex flex-col">
            {selectedVM ? (
              <ModernVMDetailTabs 
                vmName={selectedVM} 
                onBack={() => onVMSelect(null)} 
              />
            ) : (
              <ModernVMTable onVMSelect={onVMSelect} />
            )}
          </div>
        );
      
      case 'replication-jobs':
        return <JobListView onVMSelect={onVMSelect} />;
      
      case 'failover':
        return <FailoverManagement onVMSelect={onVMSelect} />;
      
      case 'network-mapping':
        return <NetworkMappingView />;
      
      case 'logs':
        return <SystemLogs />;
      
      case 'settings':
        return <SettingsView />;
      
      default:
        return (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
                Section Not Found
              </h2>
              <p className="text-gray-600 dark:text-gray-400">
                The requested section &quot;{section}&quot; is not implemented yet.
              </p>
            </div>
          </div>
        );
    }
  }, [section, selectedVM, onVMSelect, children]);

  return (
    <div className="h-full bg-gray-50 dark:bg-gray-900">
      {renderContent()}
    </div>
  );
});

MainContentArea.displayName = 'MainContentArea';


