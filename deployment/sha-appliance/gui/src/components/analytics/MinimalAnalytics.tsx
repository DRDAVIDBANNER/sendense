'use client';

import React, { useState, useEffect } from 'react';
import { Card, Button, Badge } from 'flowbite-react';

export default function MinimalAnalytics() {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await fetch('/api/analytics/overview');
        const result = await response.json();
        setData(result.data);
      } catch (error) {
        console.error('Error:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  if (loading) {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-bold mb-4">Analytics Dashboard</h1>
        <p>Loading...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
            📊 Analytics Dashboard
          </h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">
            System performance metrics and migration analytics
          </p>
        </div>
        
        <div className="flex items-center space-x-3">
          <Badge color="gray" size="sm">
            Last updated: {new Date().toLocaleTimeString()}
          </Badge>
          <Button size="sm" color="gray" onClick={() => window.location.reload()}>
            🔄 Refresh
          </Button>
        </div>
      </div>

      {/* System Health Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <Card>
          <div className="flex items-center">
            <div className="h-10 w-10 bg-blue-100 dark:bg-blue-900 rounded-lg flex items-center justify-center mr-4">
              <span className="text-blue-500 font-bold">💻</span>
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">CPU Cores</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {data?.system_health?.cpu_cores || 'N/A'}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="h-10 w-10 bg-green-100 dark:bg-green-900 rounded-lg flex items-center justify-center mr-4">
              <span className="text-green-500 font-bold">🧠</span>
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Memory Usage</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {data ? Math.round((data.system_health?.memory_usage_mb / data.system_health?.memory_total_mb) * 100) : 0}%
              </p>
              <p className="text-xs text-gray-500">
                {data?.system_health?.memory_usage_mb || 0}MB / {data?.system_health?.memory_total_mb || 0}MB
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="h-10 w-10 bg-purple-100 dark:bg-purple-900 rounded-lg flex items-center justify-center mr-4">
              <span className="text-purple-500 font-bold">⏱️</span>
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">System Uptime</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {data?.system_health?.uptime_hours ? Math.round(data.system_health.uptime_hours) + 'h' : 'N/A'}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="h-10 w-10 bg-yellow-100 dark:bg-yellow-900 rounded-lg flex items-center justify-center mr-4">
              <span className="text-yellow-500 font-bold">🗑️</span>
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">GC Runs</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {data?.system_health?.gc_runs?.toLocaleString() || 'N/A'}
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* VM Summary */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <h3 className="text-lg font-semibold mb-4">VM Summary</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="text-center">
              <div className="text-3xl font-bold text-blue-600">
                {data?.vm_summary?.total_vms || 0}
              </div>
              <div className="text-sm text-gray-500">Total VMs</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-orange-600">
                {data?.vm_summary?.active_jobs || 0}
              </div>
              <div className="text-sm text-gray-500">Active Jobs</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-green-600">
                {data?.vm_summary?.completed_jobs || 0}
              </div>
              <div className="text-sm text-gray-500">Completed</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-red-600">
                {data?.vm_summary?.failed_jobs || 0}
              </div>
              <div className="text-sm text-gray-500">Failed</div>
            </div>
          </div>
        </Card>

        <Card>
          <h3 className="text-lg font-semibold mb-4">Job Status Overview</h3>
          <div className="space-y-3">
            <div className="flex items-center justify-between p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
              <span className="text-sm font-medium">Active Jobs</span>
              <Badge color="blue" size="sm">
                {data?.vm_summary?.active_jobs || 0}
              </Badge>
            </div>
            <div className="flex items-center justify-between p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
              <span className="text-sm font-medium">Completed Jobs</span>
              <Badge color="success" size="sm">
                {data?.vm_summary?.completed_jobs || 0}
              </Badge>
            </div>
            <div className="flex items-center justify-between p-3 bg-red-50 dark:bg-red-900/20 rounded-lg">
              <span className="text-sm font-medium">Failed Jobs</span>
              <Badge color="failure" size="sm">
                {data?.vm_summary?.failed_jobs || 0}
              </Badge>
            </div>
          </div>
        </Card>
      </div>

      {/* Performance Summary */}
      <Card>
        <h3 className="text-lg font-semibold mb-4">Performance Summary</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-white">
              {data?.performance_summary?.success_rate_percent || 0}%
            </div>
            <div className="text-sm text-gray-500">Success Rate</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-white">
              {data?.performance_summary?.average_migration_speed_mbps || 'N/A'}
            </div>
            <div className="text-sm text-gray-500">Avg Speed (Mbps)</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-white">
              {data?.performance_summary?.total_data_migrated_gb || 'N/A'}
            </div>
            <div className="text-sm text-gray-500">Data Migrated (GB)</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-white">
              {data?.performance_summary?.average_completion_time_hours || 'N/A'}
            </div>
            <div className="text-sm text-gray-500">Avg Time (Hours)</div>
          </div>
        </div>
      </Card>

      {/* Recent Activity */}
      <Card>
        <h3 className="text-lg font-semibold mb-4">Recent Activity</h3>
        <div className="space-y-4">
          <div className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <span className="text-sm font-medium">Last Migration</span>
            <span className="text-sm text-gray-500">
              {data?.recent_activity?.last_migration || 'No recent migrations'}
            </span>
          </div>
          <div className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <span className="text-sm font-medium">Migrations (24h)</span>
            <Badge color="blue" size="sm">
              {data?.recent_activity?.migrations_last_24h || 0}
            </Badge>
          </div>
          <div className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <span className="text-sm font-medium">System Alerts</span>
            <Badge color={(data?.recent_activity?.system_alerts || 0) > 0 ? 'failure' : 'success'} size="sm">
              {data?.recent_activity?.system_alerts || 0}
            </Badge>
          </div>
        </div>
      </Card>
    </div>
  );
}
