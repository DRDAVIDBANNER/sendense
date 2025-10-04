'use client';

import React, { useState, useEffect } from 'react';
import { Card, Button, Badge, Alert, Spinner } from 'flowbite-react';
import { HiOutlineChartBar, HiOutlineRefresh, HiOutlineCpu, HiOutlineServer } from 'react-icons/hi';

interface SystemHealthMetrics {
  cpu_cores: number;
  memory_usage_mb: number;
  memory_total_mb: number;
  uptime_hours: number;
  gc_runs: number;
  goroutines: number;
}

interface AnalyticsOverview {
  system_health: SystemHealthMetrics;
  vm_summary: {
    total_vms: number;
    active_jobs: number;
    completed_jobs: number;
    failed_jobs: number;
  };
  performance_summary: {
    average_migration_speed_mbps: number;
    total_data_migrated_gb: number;
    success_rate_percent: number;
    average_completion_time_hours: number;
  };
  recent_activity: {
    last_migration: string | null;
    migrations_last_24h: number;
    system_alerts: number;
  };
}

export default function AnalyticsDashboard() {
  const [analyticsData, setAnalyticsData] = useState<AnalyticsOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  const fetchAnalyticsData = async () => {
    try {
      setLoading(true);
      setError('');

      const response = await fetch('/api/analytics/overview');
      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.error || 'Failed to fetch analytics data');
      }

      setAnalyticsData(result.data);
      setLastRefresh(new Date());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load analytics data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAnalyticsData();
    
    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchAnalyticsData, 30000);
    return () => clearInterval(interval);
  }, []);

  const getMemoryUsagePercent = () => {
    if (!analyticsData) return 0;
    const { memory_usage_mb, memory_total_mb } = analyticsData.system_health;
    return memory_total_mb > 0 ? Math.round((memory_usage_mb / memory_total_mb) * 100) : 0;
  };

  const formatUptime = (hours: number) => {
    if (hours < 1) return `${Math.round(hours * 60)}m`;
    if (hours < 24) return `${Math.round(hours)}h`;
    return `${Math.round(hours / 24)}d`;
  };

  if (loading && !analyticsData) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
        <span className="ml-3 text-gray-500">Loading analytics dashboard...</span>
      </div>
    );
  }

  if (error && !analyticsData) {
    return (
      <Alert color="failure">
        <div className="flex items-center justify-between">
          <span>{error}</span>
          <Button size="sm" color="failure" onClick={fetchAnalyticsData}>
            <HiOutlineRefresh className="mr-2 h-4 w-4" />
            Retry
          </Button>
        </div>
      </Alert>
    );
  }

  if (!analyticsData) {
    return (
      <Alert color="warning">
        No analytics data available
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center">
            <HiOutlineChartBar className="mr-3 h-8 w-8 text-blue-500" />
            Analytics Dashboard
          </h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">
            System performance metrics and migration analytics
          </p>
        </div>
        
        <div className="flex items-center space-x-3">
          <Badge color="gray" size="sm">
            Last updated: {lastRefresh.toLocaleTimeString()}
          </Badge>
          <Button size="sm" color="gray" onClick={fetchAnalyticsData} disabled={loading}>
            <HiOutlineRefresh className="mr-2 h-4 w-4" />
            {loading ? 'Refreshing...' : 'Refresh'}
          </Button>
        </div>
      </div>

      {/* Error Banner */}
      {error && (
        <Alert color="warning" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      {/* System Health Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <Card>
          <div className="flex items-center">
            <HiOutlineCpu className="h-10 w-10 text-blue-500" />
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">CPU Cores</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {analyticsData.system_health.cpu_cores}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <HiOutlineServer className="h-10 w-10 text-green-500" />
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Memory Usage</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {getMemoryUsagePercent()}%
              </p>
              <p className="text-xs text-gray-500">
                {analyticsData.system_health.memory_usage_mb}MB / {analyticsData.system_health.memory_total_mb}MB
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="h-10 w-10 bg-purple-100 dark:bg-purple-900 rounded-lg flex items-center justify-center">
              <span className="text-purple-500 font-bold">UP</span>
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">System Uptime</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {formatUptime(analyticsData.system_health.uptime_hours)}
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="h-10 w-10 bg-yellow-100 dark:bg-yellow-900 rounded-lg flex items-center justify-center">
              <span className="text-yellow-500 font-bold">GC</span>
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">GC Runs</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {analyticsData.system_health.gc_runs.toLocaleString()}
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
                {analyticsData.vm_summary.total_vms}
              </div>
              <div className="text-sm text-gray-500">Total VMs</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-orange-600">
                {analyticsData.vm_summary.active_jobs}
              </div>
              <div className="text-sm text-gray-500">Active Jobs</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-green-600">
                {analyticsData.vm_summary.completed_jobs}
              </div>
              <div className="text-sm text-gray-500">Completed</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-red-600">
                {analyticsData.vm_summary.failed_jobs}
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
                {analyticsData.vm_summary.active_jobs}
              </Badge>
            </div>
            <div className="flex items-center justify-between p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
              <span className="text-sm font-medium">Completed Jobs</span>
              <Badge color="success" size="sm">
                {analyticsData.vm_summary.completed_jobs}
              </Badge>
            </div>
            <div className="flex items-center justify-between p-3 bg-red-50 dark:bg-red-900/20 rounded-lg">
              <span className="text-sm font-medium">Failed Jobs</span>
              <Badge color="failure" size="sm">
                {analyticsData.vm_summary.failed_jobs}
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
              {analyticsData.performance_summary.success_rate_percent}%
            </div>
            <div className="text-sm text-gray-500">Success Rate</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-white">
              {analyticsData.performance_summary.average_migration_speed_mbps || 'N/A'}
            </div>
            <div className="text-sm text-gray-500">Avg Speed (Mbps)</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-white">
              {analyticsData.performance_summary.total_data_migrated_gb || 'N/A'}
            </div>
            <div className="text-sm text-gray-500">Data Migrated (GB)</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-white">
              {analyticsData.performance_summary.average_completion_time_hours || 'N/A'}
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
              {analyticsData.recent_activity.last_migration || 'No recent migrations'}
            </span>
          </div>
          <div className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <span className="text-sm font-medium">Migrations (24h)</span>
            <Badge color="blue" size="sm">
              {analyticsData.recent_activity.migrations_last_24h}
            </Badge>
          </div>
          <div className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <span className="text-sm font-medium">System Alerts</span>
            <Badge color={analyticsData.recent_activity.system_alerts > 0 ? 'failure' : 'success'} size="sm">
              {analyticsData.recent_activity.system_alerts}
            </Badge>
          </div>
        </div>
      </Card>
    </div>
  );
}