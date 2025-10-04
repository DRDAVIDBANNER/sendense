'use client';

import React from 'react';
import { Card, Badge, Alert } from 'flowbite-react';
import { useSystemHealth } from '../../hooks/useRealTimeUpdates';

export function RealTimeSystemHealth() {
  const { systemInfo, isConnected, error } = useSystemHealth();

  const parseMemoryInfo = (memoryLine: string) => {
    // Parse free -m output: "Mem:    total    used    free  shared  buff/cache   available"
    const parts = memoryLine.trim().split(/\s+/);
    if (parts.length >= 4) {
      return {
        total: parseInt(parts[1]) || 0,
        used: parseInt(parts[2]) || 0,
        free: parseInt(parts[3]) || 0,
        available: parts.length >= 7 ? parseInt(parts[6]) || 0 : parseInt(parts[3]) || 0
      };
    }
    return null;
  };

  const formatUptime = (uptimeString: string) => {
    // Extract uptime from string like "11:35:29 up 4 days, 2:15, 1 user, load average: 0.1, 0.2, 0.3"
    const match = uptimeString.match(/up\s+(.+?),\s+\d+\s+user/);
    return match ? match[1].trim() : 'Unknown';
  };

  const getConnectionStatus = () => {
    if (error) return { color: 'failure' as const, text: 'Disconnected' };
    if (isConnected) return { color: 'success' as const, text: 'Live' };
    return { color: 'warning' as const, text: 'Connecting' };
  };

  const connectionStatus = getConnectionStatus();
  const memoryData = systemInfo?.memory_info ? parseMemoryInfo(systemInfo.memory_info) : null;

  return (
    <div className="space-y-4">
      {/* Connection Status */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white">
          ⚡ Real-Time System Health
        </h3>
        <div className="flex items-center space-x-2">
          <div className={`w-2 h-2 rounded-full ${
            connectionStatus.color === 'success' ? 'bg-green-500' : 
            connectionStatus.color === 'warning' ? 'bg-yellow-500' : 'bg-red-500'
          }`}></div>
          <Badge color={connectionStatus.color} size="sm">
            {connectionStatus.text}
          </Badge>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <Alert color="warning" className="text-sm">
          {error}
        </Alert>
      )}

      {/* System Metrics */}
      {systemInfo && !systemInfo.error && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* System Uptime */}
          <Card>
            <div className="flex items-center">
              <div className="h-8 w-8 bg-blue-100 dark:bg-blue-900 rounded-lg flex items-center justify-center mr-3">
                <span className="text-blue-500 text-sm font-bold">⏱️</span>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">System Uptime</p>
                <p className="text-lg font-bold text-gray-900 dark:text-white">
                  {formatUptime(systemInfo.uptime)}
                </p>
              </div>
            </div>
          </Card>

          {/* Memory Usage */}
          {memoryData && (
            <Card>
              <div className="flex items-center">
                <div className="h-8 w-8 bg-green-100 dark:bg-green-900 rounded-lg flex items-center justify-center mr-3">
                  <span className="text-green-500 text-sm font-bold">🧠</span>
                </div>
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Memory Usage</p>
                  <div className="flex items-center space-x-2">
                    <p className="text-lg font-bold text-gray-900 dark:text-white">
                      {Math.round((memoryData.used / memoryData.total) * 100)}%
                    </p>
                    <div className="flex-1 bg-gray-200 rounded-full h-2 dark:bg-gray-700">
                      <div 
                        className="bg-green-600 h-2 rounded-full transition-all duration-300" 
                        style={{ width: `${(memoryData.used / memoryData.total) * 100}%` }}
                      />
                    </div>
                  </div>
                  <p className="text-xs text-gray-500">
                    {memoryData.used}MB / {memoryData.total}MB used
                  </p>
                </div>
              </div>
            </Card>
          )}

          {/* Active Jobs Count */}
          <Card>
            <div className="flex items-center">
              <div className="h-8 w-8 bg-orange-100 dark:bg-orange-900 rounded-lg flex items-center justify-center mr-3">
                <span className="text-orange-500 text-sm font-bold">🔄</span>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Active Jobs</p>
                <p className="text-lg font-bold text-gray-900 dark:text-white">
                  {systemInfo.active_job_count}
                </p>
              </div>
            </div>
          </Card>

          {/* Last Update */}
          <Card>
            <div className="flex items-center">
              <div className="h-8 w-8 bg-purple-100 dark:bg-purple-900 rounded-lg flex items-center justify-center mr-3">
                <span className="text-purple-500 text-sm font-bold">📡</span>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Last Update</p>
                <p className="text-lg font-bold text-gray-900 dark:text-white">
                  {new Date(systemInfo.timestamp).toLocaleTimeString()}
                </p>
              </div>
            </div>
          </Card>
        </div>
      )}

      {/* Error State */}
      {systemInfo?.error && (
        <Alert color="failure">
          <span>System health monitoring error: {systemInfo.error}</span>
        </Alert>
      )}

      {/* No Data State */}
      {!systemInfo && !error && isConnected && (
        <Card>
          <div className="text-center py-4">
            <p className="text-gray-500 dark:text-gray-400">
              Waiting for system health data...
            </p>
          </div>
        </Card>
      )}
    </div>
  );
}
