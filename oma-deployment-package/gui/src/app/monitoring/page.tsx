'use client';

import React from 'react';
import { LeftNavigation } from '@/components/layout/LeftNavigation';
import { RealTimeSystemHealth } from '@/components/monitoring/RealTimeSystemHealth';
import { RealTimeJobProgress } from '@/components/monitoring/RealTimeJobProgress';

export default function MonitoringPage() {
  return (
    <div className="flex min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Left Navigation */}
      <div className="w-64 flex-shrink-0">
        <LeftNavigation currentPage="monitoring" />
      </div>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        <div className="p-6 space-y-6">
          {/* Header */}
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
              ⚡ Real-Time Monitoring
            </h1>
            <p className="text-gray-500 dark:text-gray-400 mt-1">
              Live system health and migration progress updates
            </p>
          </div>

          {/* Real-time Grid Layout */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Live Migration Progress */}
            <div>
              <RealTimeJobProgress />
            </div>

            {/* System Health Monitor */}
            <div>
              <RealTimeSystemHealth />
            </div>
          </div>

          {/* Usage Instructions */}
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-lg p-4">
            <h3 className="text-sm font-medium text-blue-800 dark:text-blue-200 mb-2">
              💡 Real-Time Monitoring Features
            </h3>
            <ul className="text-sm text-blue-700 dark:text-blue-300 space-y-1">
              <li>• <strong>Live Progress Updates</strong>: Active migration jobs update every 5 seconds</li>
              <li>• <strong>System Health Metrics</strong>: Real-time memory usage, uptime, and job counts</li>
              <li>• <strong>Auto-Reconnection</strong>: Automatic reconnection if connection is lost</li>
              <li>• <strong>Visual Indicators</strong>: Live connection status and pulsing progress bars</li>
              <li>• <strong>Graceful Fallback</strong>: Falls back to polling if real-time updates fail</li>
            </ul>
          </div>
        </div>
      </main>
    </div>
  );
}
