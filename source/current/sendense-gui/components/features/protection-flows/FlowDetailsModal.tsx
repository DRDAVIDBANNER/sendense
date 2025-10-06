"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Progress } from "@/components/ui/progress";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Server,
  Activity,
  Clock,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Play,
  RotateCcw,
  Square,
  Download,
  Upload,
  BarChart3,
  Cpu,
  HardDrive,
  Wifi,
  Zap
} from "lucide-react";
import { format } from "date-fns";
import { Flow } from "./types";
import { RestoreWorkflowModal } from "./RestoreWorkflowModal";
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell
} from 'recharts';

interface Machine {
  id: string;
  name: string;
  status: 'running' | 'stopped' | 'suspended' | 'error';
  host: string;
  os: string;
  cpu: number;
  memory: number;
  storage: number;
  network: string;
  lastActivity: string;
  performance: {
    cpuUsage: number;
    memoryUsage: number;
    networkThroughput: number;
    diskIOPS: number;
  };
}

interface ActiveJob {
  id: string;
  type: 'backup' | 'replication';
  status: 'running' | 'paused' | 'error';
  progress: number;
  startTime: string;
  estimatedCompletion: string;
  currentPhase: string;
  throughput: number;
  machines: string[]; // machine IDs
}

interface JobHistory {
  id: string;
  type: 'backup' | 'replication';
  status: 'success' | 'error' | 'cancelled';
  startTime: string;
  endTime: string;
  duration: string;
  size: number;
  machines: string[];
}


interface FlowDetailsModalProps {
  flow: Flow | null;
  isOpen: boolean;
  onClose: () => void;
  onReplicationAction?: (action: 'start' | 'pause' | 'resume' | 'failover' | 'rollback' | 'cleanup') => void;
  onBackupAction?: (action: 'start' | 'restore') => void;
}

const mockMachines: Machine[] = [
  {
    id: 'vm1',
    name: 'web-server-01',
    status: 'running',
    host: 'esxi-01',
    os: 'Ubuntu 22.04',
    cpu: 2,
    memory: 4,
    storage: 100,
    network: '192.168.1.10',
    lastActivity: '2025-10-06T14:30:00Z',
    performance: {
      cpuUsage: 45,
      memoryUsage: 62,
      networkThroughput: 1250,
      diskIOPS: 850
    }
  },
  {
    id: 'vm2',
    name: 'database-01',
    status: 'running',
    host: 'esxi-02',
    os: 'Windows Server 2022',
    cpu: 4,
    memory: 16,
    storage: 500,
    network: '192.168.1.11',
    lastActivity: '2025-10-06T14:25:00Z',
    performance: {
      cpuUsage: 78,
      memoryUsage: 85,
      networkThroughput: 2100,
      diskIOPS: 1250
    }
  },
  {
    id: 'vm3',
    name: 'app-server-01',
    status: 'running',
    host: 'esxi-01',
    os: 'CentOS 8',
    cpu: 2,
    memory: 8,
    storage: 200,
    network: '192.168.1.12',
    lastActivity: '2025-10-06T14:20:00Z',
    performance: {
      cpuUsage: 32,
      memoryUsage: 45,
      networkThroughput: 890,
      diskIOPS: 620
    }
  }
];

const mockActiveJobs: ActiveJob[] = [
  {
    id: 'job1',
    type: 'replication',
    status: 'running',
    progress: 65,
    startTime: '2025-10-06T13:00:00Z',
    estimatedCompletion: '2025-10-06T15:30:00Z',
    currentPhase: 'Transferring data',
    throughput: 1250,
    machines: ['vm1', 'vm2']
  }
];

const mockJobHistory: JobHistory[] = [
  {
    id: 'hist1',
    type: 'backup',
    status: 'success',
    startTime: '2025-10-05T02:00:00Z',
    endTime: '2025-10-05T02:45:00Z',
    duration: '45m',
    size: 250,
    machines: ['vm1', 'vm2', 'vm3']
  },
  {
    id: 'hist2',
    type: 'replication',
    status: 'success',
    startTime: '2025-10-04T14:00:00Z',
    endTime: '2025-10-04T16:30:00Z',
    duration: '2h 30m',
    size: 800,
    machines: ['vm1', 'vm2']
  }
];

// Performance data for charts
const performanceData = [
  { time: '10:00', throughput: 800, cpu: 45, memory: 62 },
  { time: '10:30', throughput: 1200, cpu: 52, memory: 68 },
  { time: '11:00', throughput: 1100, cpu: 48, memory: 65 },
  { time: '11:30', throughput: 1350, cpu: 55, memory: 70 },
  { time: '12:00', throughput: 1250, cpu: 50, memory: 67 },
  { time: '12:30', throughput: 1400, cpu: 58, memory: 72 },
  { time: '13:00', throughput: 1300, cpu: 53, memory: 69 },
  { time: '13:30', throughput: 1450, cpu: 60, memory: 75 },
  { time: '14:00', throughput: 1250, cpu: 55, memory: 71 },
];

export function FlowDetailsModal({ flow, isOpen, onClose, onReplicationAction, onBackupAction }: FlowDetailsModalProps) {
  const [activeTab, setActiveTab] = useState('machines');
  const [isRestoreModalOpen, setIsRestoreModalOpen] = useState(false);

  if (!flow) return null;

  const flowMachines = mockMachines.filter(machine =>
    flow.type === 'replication' ? ['vm1', 'vm2'].includes(machine.id) : true
  );

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running': return <Activity className="h-4 w-4 text-blue-500" />;
      case 'success': return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'error': return <XCircle className="h-4 w-4 text-red-500" />;
      case 'warning': return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      default: return <Clock className="h-4 w-4 text-gray-500" />;
    }
  };

  const getStatusBadge = (status: string) => {
    const variants = {
      running: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
      success: 'bg-green-500/10 text-green-400 border-green-500/20',
      error: 'bg-red-500/10 text-red-400 border-red-500/20',
      warning: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20',
      stopped: 'bg-gray-500/10 text-gray-400 border-gray-500/20'
    };

    return (
      <Badge className={variants[status as keyof typeof variants] || variants.stopped}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Badge>
    );
  };

  const renderReplicationActions = () => {
    if (flow.type !== 'replication') return null;

    const actions = [];

    if (flow.status === 'success' || flow.status === 'warning') {
      actions.push(
        <Button key="replicate-now" onClick={() => onReplicationAction?.('start')} className="gap-2">
          <Play className="h-4 w-4" />
          Replicate Now
        </Button>
      );
    }

    if (flow.status === 'running') {
      actions.push(
        <Button key="pause" variant="outline" onClick={() => onReplicationAction?.('pause')} className="gap-2">
          <Square className="h-4 w-4" />
          Pause
        </Button>
      );
    }

    if (flow.status === 'success' && flow.progress === 100) {
      actions.push(
        <Button key="failover" variant="destructive" onClick={() => onReplicationAction?.('failover')} className="gap-2">
          <Zap className="h-4 w-4" />
          Test Failover
        </Button>
      );
    }

    return actions.length > 0 ? (
      <div className="flex gap-2">
        {actions}
      </div>
    ) : null;
  };

  const renderBackupActions = () => {
    if (flow.type !== 'backup') return null;

    return (
      <div className="flex gap-2">
        <Button onClick={() => onBackupAction?.('start')} className="gap-2">
          <Play className="h-4 w-4" />
          Backup Now
        </Button>
        <Button variant="outline" onClick={() => setIsRestoreModalOpen(true)} className="gap-2">
          <Download className="h-4 w-4" />
          Restore
        </Button>
      </div>
    );
  };

  const handleRestore = (config: any) => {
    console.log('Starting restore with config:', config);
    // TODO: Implement actual restore logic
    setIsRestoreModalOpen(false);
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="
        max-w-[90vw] w-[90vw]
        max-h-[85vh] h-[85vh]
        min-w-[900px]
        p-6
        overflow-hidden flex flex-col
      ">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              {getStatusIcon(flow.status)}
              <span>{flow.name}</span>
            </div>
            {getStatusBadge(flow.status)}
          </DialogTitle>
          <DialogDescription>
            {flow.type === 'backup' ? 'Backup' : 'Replication'} flow details and operational controls
          </DialogDescription>
        </DialogHeader>

        {/* Action Buttons */}
        <div className="flex justify-between items-center border-b pb-4">
          <div className="text-sm text-muted-foreground">
            Last run: {format(new Date(flow.lastRun), 'MMM dd, yyyy HH:mm')} |
            Next run: {format(new Date(flow.nextRun), 'MMM dd, yyyy HH:mm')}
          </div>
          {renderReplicationActions() || renderBackupActions()}
        </div>

        {/* Main Content */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 overflow-hidden flex flex-col">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="machines">Machines ({flowMachines.length})</TabsTrigger>
            <TabsTrigger value="jobs">Jobs & Progress</TabsTrigger>
            <TabsTrigger value="performance">Performance</TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-hidden">
            <TabsContent value="machines" className="h-full mt-4">
              <ScrollArea className="h-full">
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {flowMachines.map((machine) => (
                    <Card key={machine.id} className="relative">
                      <CardHeader className="pb-3">
                        <div className="flex items-center justify-between">
                          <CardTitle className="text-base flex items-center gap-2">
                            <Server className="h-4 w-4" />
                            {machine.name}
                          </CardTitle>
                          {getStatusBadge(machine.status)}
                        </div>
                      </CardHeader>
                      <CardContent className="space-y-3">
                        <div className="grid grid-cols-2 gap-2 text-sm">
                          <div>
                            <span className="text-muted-foreground">Host:</span>
                            <div className="font-medium">{machine.host}</div>
                          </div>
                          <div>
                            <span className="text-muted-foreground">OS:</span>
                            <div className="font-medium">{machine.os}</div>
                          </div>
                          <div>
                            <span className="text-muted-foreground">CPU:</span>
                            <div className="font-medium">{machine.cpu} cores</div>
                          </div>
                          <div>
                            <span className="text-muted-foreground">Memory:</span>
                            <div className="font-medium">{machine.memory} GB</div>
                          </div>
                        </div>

                        <div className="space-y-2">
                          <div className="flex justify-between text-sm">
                            <span>CPU Usage</span>
                            <span>{machine.performance.cpuUsage}%</span>
                          </div>
                          <Progress value={machine.performance.cpuUsage} className="h-2" />

                          <div className="flex justify-between text-sm">
                            <span>Memory Usage</span>
                            <span>{machine.performance.memoryUsage}%</span>
                          </div>
                          <Progress value={machine.performance.memoryUsage} className="h-2" />
                        </div>

                        <div className="text-xs text-muted-foreground">
                          Last activity: {format(new Date(machine.lastActivity), 'MMM dd, HH:mm')}
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </ScrollArea>
            </TabsContent>

            <TabsContent value="jobs" className="h-full mt-4">
              <ScrollArea className="h-full">
                <div className="space-y-6">
                  {/* Active Jobs */}
                  {mockActiveJobs.length > 0 && (
                    <div>
                      <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                        <Activity className="h-5 w-5" />
                        Active Jobs
                      </h3>
                      <div className="space-y-4">
                        {mockActiveJobs.map((job) => (
                          <Card key={job.id}>
                            <CardHeader>
                              <div className="flex items-center justify-between">
                                <CardTitle className="text-base capitalize">
                                  {job.type} Job #{job.id.slice(-4)}
                                </CardTitle>
                                {getStatusBadge(job.status)}
                              </div>
                            </CardHeader>
                            <CardContent className="space-y-4">
                              <div>
                                <div className="flex justify-between text-sm mb-2">
                                  <span>Progress</span>
                                  <span>{job.progress}%</span>
                                </div>
                                <Progress value={job.progress} className="h-3" />
                              </div>

                              <div className="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                  <span className="text-muted-foreground">Started:</span>
                                  <div>{format(new Date(job.startTime), 'HH:mm')}</div>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">ETA:</span>
                                  <div>{format(new Date(job.estimatedCompletion), 'HH:mm')}</div>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">Phase:</span>
                                  <div>{job.currentPhase}</div>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">Throughput:</span>
                                  <div>{job.throughput} MB/s</div>
                                </div>
                              </div>

                              <div>
                                <span className="text-muted-foreground text-sm">Machines:</span>
                                <div className="flex gap-1 mt-1">
                                  {job.machines.map(machineId => {
                                    const machine = mockMachines.find(m => m.id === machineId);
                                    return machine ? (
                                      <Badge key={machineId} variant="outline" className="text-xs">
                                        {machine.name}
                                      </Badge>
                                    ) : null;
                                  })}
                                </div>
                              </div>
                            </CardContent>
                          </Card>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Job History */}
                  <div>
                    <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                      <Clock className="h-5 w-5" />
                      Job History
                    </h3>
                    <div className="space-y-3">
                      {mockJobHistory.map((job) => (
                        <Card key={job.id} className="hover:shadow-md transition-shadow">
                          <CardContent className="pt-4">
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-3">
                                {getStatusIcon(job.status)}
                                <div>
                                  <div className="font-medium capitalize">
                                    {job.type} Job #{job.id.slice(-4)}
                                  </div>
                                  <div className="text-sm text-muted-foreground">
                                    {format(new Date(job.startTime), 'MMM dd, yyyy HH:mm')} • {job.duration}
                                  </div>
                                </div>
                              </div>
                              <div className="text-right">
                                <div className="font-medium">{job.size} GB</div>
                                <div className="text-sm text-muted-foreground">
                                  {job.machines.length} machines
                                </div>
                              </div>
                            </div>
                          </CardContent>
                        </Card>
                      ))}
                    </div>
                  </div>
                </div>
              </ScrollArea>
            </TabsContent>

            <TabsContent value="performance" className="h-full mt-4">
              <ScrollArea className="h-full">
                <div className="space-y-6">
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <BarChart3 className="h-5 w-5" />
                        Throughput Over Time
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="h-64">
                        <ResponsiveContainer width="100%" height="100%">
                          <AreaChart data={performanceData}>
                            <defs>
                              <linearGradient id="throughputGradient" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                                <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.1}/>
                              </linearGradient>
                            </defs>
                            <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
                            <XAxis dataKey="time" stroke="#9ca3af" fontSize={12} />
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
                              dataKey="throughput"
                              stroke="#3b82f6"
                              fill="url(#throughputGradient)"
                              strokeWidth={2}
                              name="Throughput (MB/s)"
                            />
                          </AreaChart>
                        </ResponsiveContainer>
                      </div>
                    </CardContent>
                  </Card>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <Card>
                      <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                          <Cpu className="h-5 w-5" />
                          CPU Usage Trend
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="h-48">
                          <ResponsiveContainer width="100%" height="100%">
                            <LineChart data={performanceData}>
                              <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
                              <XAxis dataKey="time" stroke="#9ca3af" fontSize={12} />
                              <YAxis stroke="#9ca3af" fontSize={12} domain={[0, 100]} />
                              <Tooltip
                                contentStyle={{
                                  backgroundColor: '#12172a',
                                  border: '1px solid #374151',
                                  borderRadius: '8px',
                                  color: '#e4e7eb'
                                }}
                              />
                              <Line
                                type="monotone"
                                dataKey="cpu"
                                stroke="#10b981"
                                strokeWidth={2}
                                name="CPU Usage (%)"
                              />
                            </LineChart>
                          </ResponsiveContainer>
                        </div>
                      </CardContent>
                    </Card>

                    <Card>
                      <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                          <HardDrive className="h-5 w-5" />
                          Memory Usage Trend
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="h-48">
                          <ResponsiveContainer width="100%" height="100%">
                            <LineChart data={performanceData}>
                              <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
                              <XAxis dataKey="time" stroke="#9ca3af" fontSize={12} />
                              <YAxis stroke="#9ca3af" fontSize={12} domain={[0, 100]} />
                              <Tooltip
                                contentStyle={{
                                  backgroundColor: '#12172a',
                                  border: '1px solid #374151',
                                  borderRadius: '8px',
                                  color: '#e4e7eb'
                                }}
                              />
                              <Line
                                type="monotone"
                                dataKey="memory"
                                stroke="#f59e0b"
                                strokeWidth={2}
                                name="Memory Usage (%)"
                              />
                            </LineChart>
                          </ResponsiveContainer>
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                </div>
              </ScrollArea>
            </TabsContent>
          </div>
        </Tabs>

        {/* Restore Workflow Modal */}
        <RestoreWorkflowModal
          isOpen={isRestoreModalOpen}
          onClose={() => setIsRestoreModalOpen(false)}
          onRestore={handleRestore}
          availableDestinations={[
            {
              id: 'source-vm',
              name: 'Original Source VM',
              type: 'source',
              available: true,
              description: 'Restore directly to the original virtual machine'
            },
            {
              id: 'local-download',
              name: 'Local Download',
              type: 'local',
              available: true,
              description: 'Download files to local storage'
            }
          ]}
          licenseFeatures={{
            backup_edition: true,
            enterprise_edition: false,
            replication_edition: false
          }}
        />
      </DialogContent>
    </Dialog>
  );
}
