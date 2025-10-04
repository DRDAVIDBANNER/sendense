'use client';

import React, { useState, useEffect } from 'react';
import { Card, Button, Badge, Spinner, Alert } from 'flowbite-react';

interface MigrationTrend {
  date: string;
  total_jobs: number;
  completed: number;
  failed: number;
  active: number;
  avg_progress: number;
}

interface PerformanceMetrics {
  avg_throughput_mbps: number;
  max_throughput_mbps: number;
  avg_completion_time_seconds: number;
  avg_data_transferred_gb: number;
  total_data_transferred_gb: number;
}

interface SuccessRate {
  os_type: string;
  total_jobs: number;
  successful: number;
  success_rate: number;
}

interface CBTEfficiency {
  operation_type: string;
  operations: number;
  avg_changes_mb: number;
  total_changes_gb: number;
}

interface VolumeMetrics {
  operation_type: string;
  operations: number;
  avg_duration_seconds: number;
  successful_ops: number;
}

interface AnalyticsData {
  migration_trends: MigrationTrend[];
  performance_metrics: PerformanceMetrics | null;
  success_rates: SuccessRate[];
  cbt_efficiency: CBTEfficiency[];
  volume_metrics: VolumeMetrics[];
  generated_at: string;
}

export default function HistoricalAnalytics() {
  const [data, setData] = useState<AnalyticsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchData = async () => {
    try {
      setLoading(true);
      setError('');
      const response = await fetch('/api/analytics/historical');
      const result = await response.json();

      if (response.ok) {
        setData(result);
      } else {
        setError(result.error || 'Failed to fetch analytics data');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Network error fetching analytics data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const formatBytes = (gb: number) => {
    if (gb < 1) return `${Math.round(gb * 1024)} MB`;
    if (gb < 1024) return `${gb.toFixed(1)} GB`;
    return `${(gb / 1024).toFixed(1)} TB`;
  };

  const formatDuration = (seconds: number) => {
    if (seconds < 60) return `${Math.round(seconds)}s`;
    if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
    return `${Math.round(seconds / 3600)}h ${Math.round((seconds % 3600) / 60)}m`;
  };

  const getSuccessColor = (rate: number) => {
    if (rate >= 90) return 'success';
    if (rate >= 70) return 'warning';
    return 'failure';
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
        <p className="ml-3 text-gray-500">Loading historical analytics...</p>
      </div>
    );
  }

  if (error) {
    return (
      <Alert color="failure" onDismiss={() => setError('')}>
        <span>{error}</span>
        <Button size="sm" color="failure" onClick={fetchData}>
          Retry
        </Button>
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
            📈 Historical Analytics
          </h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">
            Migration performance and trends over the last 30 days
          </p>
        </div>
        
        <div className="flex items-center space-x-3">
          <Badge color="gray" size="sm">
            Last updated: {data?.generated_at ? new Date(data.generated_at).toLocaleTimeString() : 'N/A'}
          </Badge>
          <Button size="sm" color="gray" onClick={fetchData}>
            🔄 Refresh
          </Button>
        </div>
      </div>

      {/* Performance Overview */}
      {data?.performance_metrics && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card>
            <div className="flex items-center">
              <div className="h-10 w-10 bg-blue-100 dark:bg-blue-900 rounded-lg flex items-center justify-center mr-4">
                <span className="text-blue-500 font-bold">⚡</span>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Avg Throughput</p>
                <p className="text-2xl font-bold text-gray-900 dark:text-white">
                  {data.performance_metrics.avg_throughput_mbps.toFixed(1)} MB/s
                </p>
              </div>
            </div>
          </Card>

          <Card>
            <div className="flex items-center">
              <div className="h-10 w-10 bg-green-100 dark:bg-green-900 rounded-lg flex items-center justify-center mr-4">
                <span className="text-green-500 font-bold">⏱️</span>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Avg Duration</p>
                <p className="text-2xl font-bold text-gray-900 dark:text-white">
                  {formatDuration(data.performance_metrics.avg_completion_time_seconds)}
                </p>
              </div>
            </div>
          </Card>

          <Card>
            <div className="flex items-center">
              <div className="h-10 w-10 bg-purple-100 dark:bg-purple-900 rounded-lg flex items-center justify-center mr-4">
                <span className="text-purple-500 font-bold">💾</span>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Total Data</p>
                <p className="text-2xl font-bold text-gray-900 dark:text-white">
                  {formatBytes(data.performance_metrics.total_data_transferred_gb)}
                </p>
              </div>
            </div>
          </Card>

          <Card>
            <div className="flex items-center">
              <div className="h-10 w-10 bg-orange-100 dark:bg-orange-900 rounded-lg flex items-center justify-center mr-4">
                <span className="text-orange-500 font-bold">🚀</span>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Peak Speed</p>
                <p className="text-2xl font-bold text-gray-900 dark:text-white">
                  {data.performance_metrics.max_throughput_mbps.toFixed(1)} MB/s
                </p>
              </div>
            </div>
          </Card>
        </div>
      )}

      {/* Migration Trends */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <h3 className="text-lg font-semibold mb-4">Migration Trends (Last 30 Days)</h3>
          {data?.migration_trends && data.migration_trends.length > 0 ? (
            <div className="space-y-3">
              {data.migration_trends.slice(0, 10).map((trend, index) => (
                <div key={trend.date} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <div>
                    <div className="font-medium text-sm">{new Date(trend.date).toLocaleDateString()}</div>
                    <div className="text-xs text-gray-500">
                      {trend.total_jobs} total jobs
                    </div>
                  </div>
                  <div className="flex space-x-2">
                    <Badge color="success" size="xs">{trend.completed} ✅</Badge>
                    <Badge color="failure" size="xs">{trend.failed} ❌</Badge>
                    {trend.active > 0 && <Badge color="warning" size="xs">{trend.active} 🔄</Badge>}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              No migration data available for the last 30 days
            </div>
          )}
        </Card>

        {/* Success Rates by OS */}
        <Card>
          <h3 className="text-lg font-semibold mb-4">Success Rates by OS Type</h3>
          {data?.success_rates && data.success_rates.length > 0 ? (
            <div className="space-y-3">
              {data.success_rates.map((rate, index) => (
                <div key={rate.os_type} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <div>
                    <div className="font-medium text-sm">{rate.os_type || 'Unknown'}</div>
                    <div className="text-xs text-gray-500">
                      {rate.successful}/{rate.total_jobs} jobs
                    </div>
                  </div>
                  <Badge color={getSuccessColor(rate.success_rate)} size="sm">
                    {rate.success_rate.toFixed(1)}%
                  </Badge>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              No OS success rate data available
            </div>
          )}
        </Card>
      </div>

      {/* CBT and Volume Operations */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <h3 className="text-lg font-semibold mb-4">CBT Efficiency</h3>
          {data?.cbt_efficiency && data.cbt_efficiency.length > 0 ? (
            <div className="space-y-3">
              {data.cbt_efficiency.map((cbt, index) => (
                <div key={cbt.operation_type} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <div>
                    <div className="font-medium text-sm">{cbt.operation_type}</div>
                    <div className="text-xs text-gray-500">
                      {cbt.operations} operations
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="font-medium text-sm">{formatBytes(cbt.total_changes_gb)}</div>
                    <div className="text-xs text-gray-500">
                      {cbt.avg_changes_mb.toFixed(1)} MB avg
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              No CBT data available
            </div>
          )}
        </Card>

        <Card>
          <h3 className="text-lg font-semibold mb-4">Volume Operations</h3>
          {data?.volume_metrics && data.volume_metrics.length > 0 ? (
            <div className="space-y-3">
              {data.volume_metrics.map((vol, index) => (
                <div key={vol.operation_type} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <div>
                    <div className="font-medium text-sm">{vol.operation_type}</div>
                    <div className="text-xs text-gray-500">
                      {vol.operations} operations
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="font-medium text-sm">{formatDuration(vol.avg_duration_seconds)}</div>
                    <div className="text-xs text-gray-500">
                      {((vol.successful_ops / vol.operations) * 100).toFixed(1)}% success
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              No volume operation data available
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}
