"use client";

import { useState, useEffect } from "react";
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
import { getUIStatus, FlowMachineInfo } from "@/src/features/protection-flows/types";
import { RestoreWorkflowModal } from "./RestoreWorkflowModal";
import { MachineDetailsModal } from "./MachineDetailsModal";
import { FlowMachinesTable } from "./FlowMachinesTable";
import { useFlowExecutions, useFlowMachines } from "@/src/features/protection-flows/hooks/useProtectionFlows";
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

interface FlowDetailsPanelProps {
  flow: Flow;
}


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

export function FlowDetailsPanel({ flow }: FlowDetailsPanelProps) {
  const [activeTab, setActiveTab] = useState('machines');
  const [isRestoreModalOpen, setIsRestoreModalOpen] = useState(false);
  const [selectedMachine, setSelectedMachine] = useState<FlowMachineInfo | null>(null);
  const [isMachineModalOpen, setIsMachineModalOpen] = useState(false);

  // Real API data
  const { data: executionsData } = useFlowExecutions(flow.id);
  const executions = executionsData?.executions || [];

  // Real machine data from flow
  const { data: machinesData, isLoading: machinesLoading } = useFlowMachines(flow.id);
  const flowMachines = machinesData?.machines || [];

  if (!flow) return null;

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running': return <Activity className="h-4 w-4 text-blue-500" />;
      case 'success': return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'error': return <XCircle className="h-4 w-4 text-red-500" />;
      case 'warning': return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      default: return <Clock className="h-4 w-4 text-muted-foreground" />;
    }
  };

  const getStatusBadge = (status: string) => {
    const variants = {
      running: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
      success: 'bg-green-500/10 text-green-400 border-green-500/20',
      error: 'bg-red-500/10 text-red-400 border-red-500/20',
      warning: 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20',
      stopped: 'bg-muted text-muted-foreground border-muted-foreground/20'
    };

    return (
      <Badge className={variants[status as keyof typeof variants] || variants.stopped}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Badge>
    );
  };

  const renderReplicationActions = () => {
    if (flow.flow_type !== 'replication') return null;

    const actions = [];
    const uiStatus = getUIStatus(flow);

    if (uiStatus === 'success' || uiStatus === 'warning') {
      actions.push(
        <Button key="replicate-now" onClick={() => {}} className="gap-2">
          <Play className="h-4 w-4" />
          Replicate Now
        </Button>
      );
    }

    if (uiStatus === 'running') {
      actions.push(
        <Button key="pause" variant="outline" onClick={() => {}} className="gap-2">
          <Square className="h-4 w-4" />
          Pause
        </Button>
      );
    }

    if (uiStatus === 'success' && flow.progress === 100) {
      actions.push(
        <Button key="failover" variant="destructive" onClick={() => {}} className="gap-2">
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
    if (flow.flow_type !== 'backup') return null;

    return (
      <div className="flex gap-2">
        <Button onClick={() => {}} className="gap-2">
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
    <div className="h-full flex flex-col bg-background">
      {/* Header with action buttons */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-border shrink-0">
        <div>
          <div className="flex items-center gap-3">
            <h3 className="text-xl font-semibold text-foreground">{flow.name}</h3>
            {getStatusBadge(getUIStatus(flow))}
          </div>
          <p className="text-sm text-muted-foreground mt-1">{flow.source} → {flow.destination}</p>
        </div>

        <div className="flex gap-2">
          {renderReplicationActions() || renderBackupActions()}
        </div>
          </div>

      {/* Tabs */}
          <div className="flex-1 overflow-auto">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="h-full flex flex-col">
          <TabsList className="grid w-full grid-cols-3 mx-6 mt-4 mb-2">
            <TabsTrigger value="machines">Machines ({flowMachines.length})</TabsTrigger>
            <TabsTrigger value="jobs">Jobs & Progress</TabsTrigger>
            <TabsTrigger value="performance">Performance</TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-hidden">
            <TabsContent value="machines" className="h-full mt-4">
              <ScrollArea className="h-full">
                <div className="px-6 pb-4">
                  {machinesLoading ? (
                    <div className="flex items-center justify-center py-8">
                      <div className="text-muted-foreground">Loading machines...</div>
                    </div>
                  ) : (
                    <FlowMachinesTable
                      machines={flowMachines}
                      onMachineClick={(machine) => {
                        setSelectedMachine(machine);
                        setIsMachineModalOpen(true);
                      }}
                    />
                  )}
                </div>
              </ScrollArea>
            </TabsContent>

            <TabsContent value="jobs" className="h-full mt-4">
              <ScrollArea className="h-full">
                <div className="space-y-6 px-6 pb-4">
                  {/* Active Jobs */}
                  {executions.filter(e => e.status === 'running').length > 0 && (
                    <div>
                      <h3 className="text-lg font-semibold mb-4 flex items-center gap-2 text-foreground">
                        <Activity className="h-5 w-5 text-primary" />
                        Active Jobs
                      </h3>
                      <div className="space-y-4">
                        {executions.filter(e => e.status === 'running').map((execution) => (
                          <Card key={execution.id}>
                            <CardHeader>
                              <div className="flex items-center justify-between">
                                <CardTitle className="text-base capitalize text-foreground">
                                  {flow.flow_type} Execution #{execution.id.slice(-4)}
                                </CardTitle>
                                {getStatusBadge(execution.status)}
                              </div>
                            </CardHeader>
                            <CardContent className="space-y-4">
                              <div>
                                <div className="flex justify-between text-sm mb-2">
                                  <span className="text-muted-foreground">Status</span>
                                  <span className="text-foreground">{execution.status}</span>
                                </div>
                              </div>

                              <div className="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                  <span className="text-muted-foreground">Started:</span>
                                  <div className="text-foreground">{format(new Date(execution.started_at), 'HH:mm')}</div>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">Duration:</span>
                                  <div className="text-foreground">{execution.duration_seconds || 0}s</div>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">Transferred:</span>
                                  <div className="text-foreground">{(execution.bytes_transferred || 0) / (1024 * 1024)} MB</div>
                                </div>
                              </div>
                            </CardContent>
                </Card>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Job History */}
                  {executions.filter(e => e.status !== 'running').length > 0 && (
                    <div>
                      <h3 className="text-lg font-semibold mb-4 flex items-center gap-2 text-foreground">
                        <Clock className="h-5 w-5 text-muted-foreground" />
                        Execution History
                      </h3>
                      <div className="space-y-4">
                        {executions.filter(e => e.status !== 'running').slice(0, 5).map((execution) => (
                          <Card key={execution.id}>
                            <CardHeader>
                              <div className="flex items-center justify-between">
                                <CardTitle className="text-base capitalize text-foreground">
                                  {flow.flow_type} Execution #{execution.id.slice(-4)}
                                </CardTitle>
                                {getStatusBadge(execution.status)}
                              </div>
                            </CardHeader>
                            <CardContent className="space-y-4">
                              <div className="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                  <span className="text-muted-foreground">Started:</span>
                                  <div className="text-foreground">{format(new Date(execution.started_at), 'MMM dd, HH:mm')}</div>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">Completed:</span>
                                  <div className="text-foreground">
                                    {execution.completed_at ? format(new Date(execution.completed_at), 'MMM dd, HH:mm') : 'In Progress'}
                                  </div>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">Duration:</span>
                                  <div className="text-foreground">{execution.duration_seconds || 0}s</div>
                                </div>
                                <div>
                                  <span className="text-muted-foreground">Transferred:</span>
                                  <div className="text-foreground">{(execution.bytes_transferred || 0) / (1024 * 1024)} MB</div>
                                </div>
                              </div>
                              {execution.error_message && (
                                <div className="text-sm">
                                  <span className="text-muted-foreground">Error:</span>
                                  <div className="text-destructive mt-1">{execution.error_message}</div>
                                </div>
                              )}
                            </CardContent>
                          </Card>
                        ))}
                      </div>
                    </div>
                  )}

                </div>
              </ScrollArea>
            </TabsContent>

            <TabsContent value="performance" className="h-full mt-4">
              <ScrollArea className="h-full">
                <div className="space-y-6 px-6 pb-4">
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2 text-foreground">
                        <BarChart3 className="h-5 w-5 text-primary" />
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
                        <CardTitle className="flex items-center gap-2 text-foreground">
                          <Cpu className="h-5 w-5 text-primary" />
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
                        <CardTitle className="flex items-center gap-2 text-foreground">
                          <HardDrive className="h-5 w-5 text-primary" />
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
      </div>

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

      {/* Machine Details Modal */}
      <MachineDetailsModal
        machine={selectedMachine}
        repositoryId={flow.repository_id || ''}
        isOpen={isMachineModalOpen}
        onClose={() => {
          setIsMachineModalOpen(false);
          setSelectedMachine(null); // Reset state on close
        }}
      />
    </div>
  );
}
