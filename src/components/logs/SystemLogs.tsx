'use client';

import React from 'react';
import { Card } from 'flowbite-react';

export const SystemLogs = React.memo(() => {
  return (
    <div className="p-6">
      <Card>
        <div className="text-center p-8">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
            System Logs
          </h2>
          <p className="text-gray-600 dark:text-gray-400">
            Log viewing and troubleshooting tools coming in Phase 3.
          </p>
        </div>
      </Card>
    </div>
  );
});

SystemLogs.displayName = 'SystemLogs';










