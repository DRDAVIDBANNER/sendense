'use client';

import React from 'react';
import { Card } from 'flowbite-react';

export interface JobListViewProps {
  onVMSelect: (vmName: string) => void;
}

export const JobListView = React.memo(({ onVMSelect }: JobListViewProps) => {
  return (
    <div className="p-6">
      <Card>
        <div className="text-center p-8">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
            Replication Jobs
          </h2>
          <p className="text-gray-600 dark:text-gray-400">
            Job-centric view implementation coming in Phase 3.
          </p>
        </div>
      </Card>
    </div>
  );
});

JobListView.displayName = 'JobListView';










