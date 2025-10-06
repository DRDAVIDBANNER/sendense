"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Download,
  TrendingUp,
  TrendingDown,
  BarChart3,
  Calendar as CalendarIcon,
  FileText,
  Database,
  Clock,
  CheckCircle,
  AlertTriangle,
  XCircle
} from "lucide-react";
import { PageHeader } from "@/components/common/PageHeader";
import { format, subDays, startOfMonth, endOfMonth } from "date-fns";
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell
} from 'recharts';

// Mock data for charts
const backupSuccessData = [
  { date: '2025-09-25', success: 95, total: 100 },
  { date: '2025-09-26', success: 98, total: 105 },
  { date: '2025-09-27', success: 92, total: 98 },
  { date: '2025-09-28', success: 96, total: 102 },
  { date: '2025-09-29', success: 94, total: 99 },
  { date: '2025-09-30', success: 97, total: 103 },
  { date: '2025-10-01', success: 99, total: 104 },
];

const storageGrowthData = [
  { month: 'Jan', used: 120, total: 500 },
  { month: 'Feb', used: 145, total: 500 },
  { month: 'Mar', used: 180, total: 500 },
  { month: 'Apr', used: 220, total: 500 },
  { month: 'May', used: 280, total: 500 },
  { month: 'Jun', used: 320, total: 500 },
];

const vmBackupData = [
  { vm: 'web-server-01', backups: 28, size: 25 },
  { vm: 'database-01', backups: 15, size: 150 },
  { vm: 'app-server-01', backups: 22, size: 40 },
  { vm: 'dev-server-01', backups: 8, size: 15 },
  { vm: 'legacy-app-01', backups: 5, size: 80 },
];

const statusDistribution = [
  { name: 'Successful', value: 85, color: '#10b981' },
  { name: 'Failed', value: 10, color: '#ef4444' },
  { name: 'Running', value: 5, color: '#3b82f6' },
];

const kpiData = {
  totalBackups: 1247,
  successRate: 94.2,
  avgDuration: '12m 34s',
  storageUsed: '2.4 TB',
  activeJobs: 3,
  totalVMs: 47,
  protectedVMs: 42,
  dataTransferred: '156.8 GB'
};

export default function ReportCenterPage() {
  const [dateRange, setDateRange] = useState<{
    from?: Date;
    to?: Date;
  }>({
    from: startOfMonth(new Date()),
    to: endOfMonth(new Date())
  });

  const [timeRange, setTimeRange] = useState('30d');

  const handleExport = (format: 'csv' | 'pdf') => {
    // TODO: Implement export functionality
    console.log(`Exporting report as ${format.toUpperCase()}`);
  };

  const getTimeRangeLabel = (range: string) => {
    switch (range) {
      case '7d': return 'Last 7 days';
      case '30d': return 'Last 30 days';
      case '90d': return 'Last 90 days';
      case '1y': return 'Last year';
      default: return 'Custom range';
    }
  };

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Report Center"
        breadcrumbs={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Report Center" }
        ]}
        actions={
          <div className="flex items-center gap-3">
            <Select value={timeRange} onValueChange={setTimeRange}>
              <SelectTrigger className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="7d">Last 7 days</SelectItem>
                <SelectItem value="30d">Last 30 days</SelectItem>
                <SelectItem value="90d">Last 90 days</SelectItem>
                <SelectItem value="1y">Last year</SelectItem>
              </SelectContent>
            </Select>

            <Popover>
              <PopoverTrigger asChild>
                <Button variant="outline" className="gap-2">
                  <CalendarIcon className="h-4 w-4" />
                  {dateRange.from && dateRange.to
                    ? `${format(dateRange.from, 'MMM dd')} - ${format(dateRange.to, 'MMM dd')}`
                    : 'Select dates'
                  }
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-auto p-0" align="end">
                <Calendar
                  initialFocus
                  mode="range"
                  required
                  defaultMonth={dateRange.from}
                  selected={{
                    from: dateRange.from,
                    to: dateRange.to
                  }}
                  onSelect={setDateRange}
                  numberOfMonths={2}
                />
              </PopoverContent>
            </Popover>

            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => handleExport('csv')} className="gap-2">
                <Download className="h-4 w-4" />
                CSV
              </Button>
              <Button variant="outline" size="sm" onClick={() => handleExport('pdf')} className="gap-2">
                <FileText className="h-4 w-4" />
                PDF
              </Button>
            </div>
          </div>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="p-6 space-y-6">
          {/* KPI Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Backups</CardTitle>
                <Database className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{kpiData.totalBackups.toLocaleString()}</div>
                <div className="flex items-center text-xs text-muted-foreground mt-1">
                  <TrendingUp className="h-3 w-3 mr-1 text-green-500" />
                  +12% from last month
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
                <CheckCircle className="h-4 w-4 text-green-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{kpiData.successRate}%</div>
                <div className="flex items-center text-xs text-muted-foreground mt-1">
                  <TrendingUp className="h-3 w-3 mr-1 text-green-500" />
                  +2.1% from last month
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Avg Duration</CardTitle>
                <Clock className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{kpiData.avgDuration}</div>
                <div className="flex items-center text-xs text-muted-foreground mt-1">
                  <TrendingDown className="h-3 w-3 mr-1 text-green-500" />
                  -8% from last month
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Storage Used</CardTitle>
                <BarChart3 className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{kpiData.storageUsed}</div>
                <div className="flex items-center text-xs text-muted-foreground mt-1">
                  <TrendingUp className="h-3 w-3 mr-1 text-yellow-500" />
                  +15% from last month
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Charts Grid */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Backup Success Trend */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <TrendingUp className="h-5 w-5" />
                  Backup Success Rate Trend
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={backupSuccessData}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
                      <XAxis
                        dataKey="date"
                        stroke="#9ca3af"
                        fontSize={12}
                        tickFormatter={(value) => format(new Date(value), 'MMM dd')}
                      />
                      <YAxis stroke="#9ca3af" fontSize={12} />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: '#12172a',
                          border: '1px solid #374151',
                          borderRadius: '8px',
                          color: '#e4e7eb'
                        }}
                        labelFormatter={(value) => format(new Date(value), 'MMM dd, yyyy')}
                      />
                      <Line
                        type="monotone"
                        dataKey="success"
                        stroke="#10b981"
                        strokeWidth={2}
                        dot={{ fill: '#10b981', strokeWidth: 2, r: 4 }}
                        name="Successful Backups"
                      />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>

            {/* Storage Growth */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Database className="h-5 w-5" />
                  Storage Growth Trend
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={storageGrowthData}>
                      <defs>
                        <linearGradient id="storageGradient" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                          <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.1}/>
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
                      <XAxis dataKey="month" stroke="#9ca3af" fontSize={12} />
                      <YAxis stroke="#9ca3af" fontSize={12} />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: '#12172a',
                          border: '1px solid #374151',
                          borderRadius: '8px',
                          color: '#e4e7eb'
                        }}
                      />
                      <Area
                        type="monotone"
                        dataKey="used"
                        stroke="#3b82f6"
                        fill="url(#storageGradient)"
                        strokeWidth={2}
                        name="Storage Used (GB)"
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>

            {/* VM Backup Distribution */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <BarChart3 className="h-5 w-5" />
                  VM Backup Distribution
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={vmBackupData} layout="horizontal">
                      <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
                      <XAxis type="number" stroke="#9ca3af" fontSize={12} />
                      <YAxis
                        dataKey="vm"
                        type="category"
                        stroke="#9ca3af"
                        fontSize={12}
                        width={80}
                        tickFormatter={(value) => value.split('-')[0]}
                      />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: '#12172a',
                          border: '1px solid #374151',
                          borderRadius: '8px',
                          color: '#e4e7eb'
                        }}
                      />
                      <Bar dataKey="backups" fill="#8b5cf6" name="Backup Count" />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </CardContent>
            </Card>

            {/* Status Distribution */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <CheckCircle className="h-5 w-5" />
                  Backup Status Distribution
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="h-64 flex items-center justify-center">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={statusDistribution}
                        cx="50%"
                        cy="50%"
                        innerRadius={60}
                        outerRadius={80}
                        paddingAngle={5}
                        dataKey="value"
                      >
                        {statusDistribution.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} />
                        ))}
                      </Pie>
                      <Tooltip
                        contentStyle={{
                          backgroundColor: '#12172a',
                          border: '1px solid #374151',
                          borderRadius: '8px',
                          color: '#e4e7eb'
                        }}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
                <div className="flex justify-center gap-4 mt-4">
                  {statusDistribution.map((item, index) => (
                    <div key={index} className="flex items-center gap-2 text-sm">
                      <div
                        className="w-3 h-3 rounded-full"
                        style={{ backgroundColor: item.color }}
                      />
                      <span className="text-muted-foreground">{item.name}</span>
                      <span className="font-medium">{item.value}%</span>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Additional Metrics */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">System Health</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Active Jobs</span>
                  <Badge variant="secondary">{kpiData.activeJobs}</Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">VM Protection</span>
                  <Badge className="bg-green-500/10 text-green-400 border-green-500/20">
                    {kpiData.protectedVMs}/{kpiData.totalVMs} VMs
                  </Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Data Transferred</span>
                  <span className="font-medium">{kpiData.dataTransferred}</span>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Performance Metrics</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Peak Throughput</span>
                  <span className="font-medium">220 MB/s</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Avg Throughput</span>
                  <span className="font-medium">160 MB/s</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Network Efficiency</span>
                  <span className="font-medium text-green-400">98.5%</span>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Recent Alerts</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex items-start gap-3 p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/20">
                  <AlertTriangle className="h-4 w-4 text-yellow-500 mt-0.5" />
                  <div className="flex-1">
                    <p className="text-sm font-medium">Storage capacity warning</p>
                    <p className="text-xs text-muted-foreground">75% of allocated storage used</p>
                  </div>
                </div>
                <div className="flex items-start gap-3 p-3 rounded-lg bg-red-500/10 border border-red-500/20">
                  <XCircle className="h-4 w-4 text-red-500 mt-0.5" />
                  <div className="flex-1">
                    <p className="text-sm font-medium">Backup failure detected</p>
                    <p className="text-xs text-muted-foreground">3 jobs failed in the last hour</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}
