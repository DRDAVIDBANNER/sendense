"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { RefreshCw, Activity, Database, Shield, TrendingUp, Clock, CheckCircle, AlertTriangle, XCircle } from "lucide-react";
import { PageHeader } from "@/components/common/PageHeader";
import { PerformanceChart } from "@/components/features/dashboard";

interface SystemHealthCard {
  title: string;
  value: string | number;
  status: 'good' | 'warning' | 'error';
  icon: React.ComponentType<{ className?: string }>;
  description: string;
}

interface ActivityItem {
  id: string;
  type: 'backup' | 'replication' | 'system';
  title: string;
  status: 'success' | 'running' | 'error';
  timestamp: string;
  details?: string;
}

const systemHealthCards: SystemHealthCard[] = [
  {
    title: "All Systems",
    value: "Operational",
    status: "good",
    icon: CheckCircle,
    description: "All backup systems running normally"
  },
  {
    title: "Protected VMs",
    value: 47,
    status: "good",
    icon: Shield,
    description: "Virtual machines under protection"
  },
  {
    title: "Active Jobs",
    value: 3,
    status: "good",
    icon: Activity,
    description: "Currently running backup jobs"
  },
  {
    title: "Storage Used",
    value: "2.4 TB",
    status: "warning",
    icon: Database,
    description: "Total backup storage consumption"
  }
];

const mockActivities: ActivityItem[] = [
  {
    id: '1',
    type: 'backup',
    title: 'Daily VM Backup - pgtest1',
    status: 'success',
    timestamp: '2025-10-06T10:00:00Z',
    details: 'Completed successfully in 12m 34s'
  },
  {
    id: '2',
    type: 'replication',
    title: 'Hourly Replication - web-servers',
    status: 'running',
    timestamp: '2025-10-06T09:45:00Z',
    details: '65% complete, transferring data...'
  },
  {
    id: '3',
    type: 'backup',
    title: 'Weekly Archive - legacy-apps',
    status: 'error',
    timestamp: '2025-10-06T08:30:00Z',
    details: 'Failed: Network timeout'
  },
  {
    id: '4',
    type: 'system',
    title: 'System Health Check',
    status: 'success',
    timestamp: '2025-10-06T08:00:00Z',
    details: 'All systems operational'
  },
  {
    id: '5',
    type: 'backup',
    title: 'Critical DB Backup',
    status: 'success',
    timestamp: '2025-10-06T06:00:00Z',
    details: 'Completed successfully in 8m 12s'
  }
];

export default function DashboardPage() {
  const [lastRefresh, setLastRefresh] = useState(new Date());
  const [isRefreshing, setIsRefreshing] = useState(false);

  const handleRefresh = async () => {
    setIsRefreshing(true);
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 1000));
    setLastRefresh(new Date());
    setIsRefreshing(false);
  };

  useEffect(() => {
    // Auto-refresh every 30 seconds
    const interval = setInterval(() => {
      handleRefresh();
    }, 30000);

    return () => clearInterval(interval);
  }, []);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success': return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'running': return <Activity className="h-4 w-4 text-blue-500" />;
      case 'error': return <XCircle className="h-4 w-4 text-red-500" />;
      default: return <Clock className="h-4 w-4 text-gray-500" />;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'success': return <Badge className="bg-green-500/10 text-green-400 border-green-500/20">Success</Badge>;
      case 'running': return <Badge className="bg-blue-500/10 text-blue-400 border-blue-500/20">Running</Badge>;
      case 'error': return <Badge className="bg-red-500/10 text-red-400 border-red-500/20">Error</Badge>;
      default: return <Badge variant="secondary">Unknown</Badge>;
    }
  };

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffHours = diffMs / (1000 * 60 * 60);

    if (diffHours < 1) {
      const diffMinutes = Math.floor(diffMs / (1000 * 60));
      return `${diffMinutes}m ago`;
    } else if (diffHours < 24) {
      return `${Math.floor(diffHours)}h ago`;
    } else {
      return date.toLocaleDateString();
    }
  };

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Dashboard"
        actions={
          <div className="flex items-center gap-4">
            <span className="text-sm text-muted-foreground">
              Last updated: {lastRefresh.toLocaleTimeString()}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={handleRefresh}
              disabled={isRefreshing}
              className="gap-2"
            >
              <RefreshCw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          </div>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="p-6 space-y-6">
          {/* System Health Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {systemHealthCards.map((card) => {
              const Icon = card.icon;
              return (
                <Card key={card.title}>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">{card.title}</CardTitle>
                    <Icon className={`h-4 w-4 ${
                      card.status === 'good' ? 'text-green-500' :
                      card.status === 'warning' ? 'text-yellow-500' :
                      'text-red-500'
                    }`} />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{card.value}</div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {card.description}
                    </p>
                    {card.title === "Storage Used" && (
                      <div className="mt-3">
                        <Progress value={75} className="h-2" />
                        <p className="text-xs text-muted-foreground mt-1">
                          75% of 3.2 TB capacity
                        </p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              );
            })}
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Performance Chart */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <TrendingUp className="h-5 w-5" />
                  Performance Overview
                </CardTitle>
              </CardHeader>
              <CardContent>
                <PerformanceChart />
              </CardContent>
            </Card>

            {/* Recent Activity Feed */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Activity className="h-5 w-5" />
                  Recent Activity
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {mockActivities.map((activity) => (
                    <div key={activity.id} className="flex items-start gap-3 p-3 rounded-lg bg-muted/20">
                      <div className="mt-0.5">
                        {getStatusIcon(activity.status)}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <p className="text-sm font-medium text-foreground truncate">
                            {activity.title}
                          </p>
                          {getStatusBadge(activity.status)}
                        </div>
                        {activity.details && (
                          <p className="text-xs text-muted-foreground mb-1">
                            {activity.details}
                          </p>
                        )}
                        <p className="text-xs text-muted-foreground">
                          {formatTimestamp(activity.timestamp)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}
