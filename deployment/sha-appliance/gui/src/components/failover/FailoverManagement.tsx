'use client';

import React from 'react';
import { Card } from 'flowbite-react';

export interface FailoverManagementProps {
  onVMSelect: (vmName: string) => void;
}

export const FailoverManagement = React.memo(({ onVMSelect }: FailoverManagementProps) => {
  return (
    <div className="p-6">
      <Card>
        <div className="text-center p-8">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
            Failover Management
          </h2>
          <p className="text-gray-600 dark:text-gray-400">
            Failover controls implementation coming in Phase 2.
          </p>
        </div>
      </Card>
    </div>
  );
});

FailoverManagement.displayName = 'FailoverManagement';










