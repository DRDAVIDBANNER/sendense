"use client";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { AlertCircle, Server, HardDrive, Cpu, Zap } from "lucide-react";
import { format } from "date-fns";
import type { FlowMachineInfo } from "@/src/features/protection-flows/types";
import { useMachineBackups } from "@/src/features/protection-flows/hooks/useProtectionFlows";
import { formatBytes, formatDuration, getDuration, formatTimestamp } from "@/src/features/protection-flows/utils/formatters";

interface MachineDetailsModalProps {
  machine: FlowMachineInfo | null;
  repositoryId: string;
  isOpen: boolean;
  onClose: () => void;
}

interface BackupRecord {
  backup_id: string;
  type: "full" | "incremental";
  status: "completed" | "failed" | "running";
  bytes_transferred: number;
  progress_percent: number;
  transfer_speed_bps: number;
  current_phase: string;
  created_at: string;
  started_at: string | null;
  completed_at: string | null;
  error_message?: string;
  last_telemetry_at: string | null;
}

export function MachineDetailsModal({ machine, repositoryId, isOpen, onClose }: MachineDetailsModalProps) {
  const { data: backups, isLoading, isError, error } = useMachineBackups(
    machine?.vm_name || null,
    repositoryId
  );

  if (!machine) return null;

  // Calculate KPIs
  const totalBackups = backups?.length || 0;
  const completedBackups = backups?.filter((b: any) => b.status === 'completed') || [];
  const successRate = totalBackups > 0
    ? ((completedBackups.length / totalBackups) * 100).toFixed(0) + '%'
    : 'N/A';

  // Average size (completed backups only, using bytes_transferred)
  const avgSize = completedBackups.length > 0
    ? completedBackups.reduce((sum: number, b: any) => sum + b.bytes_transferred, 0) / completedBackups.length
    : 0;
  const avgSizeFormatted = formatBytes(avgSize);

  // Average duration (completed backups only)
  const completedWithTime = completedBackups.filter((b: any) =>
    b.started_at && b.completed_at
  );
  const avgDuration = completedWithTime.length > 0
    ? completedWithTime.reduce((sum: number, b: any) => {
        const duration = getDuration(b.started_at, b.completed_at);
        return sum + duration;
      }, 0) / completedWithTime.length
    : 0;
  const avgDurationFormatted = formatDuration(avgDuration);

  const getOSIcon = (os: string): string => {
    if (os.toLowerCase().includes('windows')) return '🪟';
    if (os.toLowerCase().includes('linux')) return '🐧';
    return '💿';
  };

  const totalDisksGB = machine.disks.reduce((sum, d) => sum + d.size_gb, 0);

  const statusStyles = {
    completed: "bg-green-500/10 text-green-400 border-green-500/20",
    failed: "bg-red-500/10 text-red-400 border-red-500/20",
    running: "bg-blue-500/10 text-blue-400 border-blue-500/20",
  };

  const typeStyles = {
    full: "bg-blue-500/10 text-blue-400 border-blue-500/20",
    incremental: "bg-green-500/10 text-green-400 border-green-500/20",
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-xl">
            <Server className="h-5 w-5" />
            {machine.vm_name}
          </DialogTitle>
        </DialogHeader>

        <div className="flex flex-col h-full max-h-[calc(90vh-8rem)] overflow-hidden">
          {/* VM Summary Card */}
          <Card className="mb-6">
            <CardContent className="pt-6">
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4 text-sm">
                <div className="flex items-center gap-2">
                  <Cpu className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">CPU:</span>
                  <span className="font-medium">{machine.cpu_count} cores</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">Memory:</span>
                  <span className="font-medium">{Math.round(machine.memory_mb / 1024)} GB</span>
                </div>
                <div className="flex items-center gap-2">
                  <HardDrive className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Disks:</span>
                  <span className="font-medium">{machine.disks.length} ({totalDisksGB} GB total)</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-lg">{getOSIcon(machine.os_type)}</span>
                  <span className="text-muted-foreground">OS:</span>
                  <span className="font-medium">{machine.os_type}</span>
                </div>
              </div>
              <div className="mt-3 flex items-center gap-2">
                <span
                  className={`w-2 h-2 rounded-full ${
                    machine.power_state === 'poweredOn' ? 'bg-green-500' :
                    machine.power_state === 'poweredOff' ? 'bg-gray-400' :
                    'bg-red-500'
                  }`}
                />
                <span className="text-sm text-muted-foreground">Power State:</span>
                <span className="text-sm font-medium capitalize">{machine.power_state}</span>
              </div>
            </CardContent>
          </Card>

          {/* KPI Cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">Total Backups</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{totalBackups}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">Success Rate</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{successRate}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">Avg Size</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{avgSizeFormatted}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">Avg Duration</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{avgDurationFormatted}</div>
              </CardContent>
            </Card>
          </div>

          {/* Backup History */}
          <Card className="flex-1 overflow-hidden">
            <CardHeader>
              <CardTitle className="text-lg">Backup History</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <ScrollArea className="h-96">
                {isLoading ? (
                  <div className="flex items-center justify-center py-12">
                    <div className="text-muted-foreground">Loading backup history...</div>
                  </div>
                ) : isError ? (
                  <div className="text-center py-12">
                    <div className="flex items-center justify-center gap-2 text-red-500 mb-2">
                      <AlertCircle className="h-5 w-5" />
                      <span className="font-medium">Failed to load backup history</span>
                    </div>
                    <div className="text-sm text-muted-foreground">{error?.message}</div>
                  </div>
                ) : backups && backups.length > 0 ? (
                  <div className="overflow-x-auto">
                    <table className="w-full">
                      <thead className="border-b border-border">
                        <tr className="text-left">
                          <th className="px-4 py-3 text-sm font-medium text-muted-foreground">Type</th>
                          <th className="px-4 py-3 text-sm font-medium text-muted-foreground">Size</th>
                          <th className="px-4 py-3 text-sm font-medium text-muted-foreground">Duration</th>
                          <th className="px-4 py-3 text-sm font-medium text-muted-foreground">Status</th>
                          <th className="px-4 py-3 text-sm font-medium text-muted-foreground">Date</th>
                        </tr>
                      </thead>
                      <tbody>
                        {backups.map((backup: any) => (
                          <tr key={backup.backup_id} className="border-b border-border/50 hover:bg-muted/50">
                            <td className="px-4 py-3">
                              <Badge
                                variant="outline"
                                className={typeStyles[backup.type as keyof typeof typeStyles] || typeStyles.full}
                              >
                                {backup.type.charAt(0).toUpperCase() + backup.type.slice(1)}
                              </Badge>
                            </td>
                            <td className="px-4 py-3 text-sm">
                              {formatBytes(backup.bytes_transferred)}
                            </td>
                            <td className="px-4 py-3 text-sm">
                              {formatDuration(getDuration(backup.started_at, backup.completed_at))}
                            </td>
                            <td className="px-4 py-3">
                              <Badge
                                className={statusStyles[backup.status as keyof typeof statusStyles] || statusStyles.completed}
                              >
                                {backup.status === 'completed' ? '✅ Success' :
                                 backup.status === 'failed' ? '❌ Failed' :
                                 '🔄 Running'}
                              </Badge>
                            </td>
                            <td className="px-4 py-3 text-sm">
                              {formatTimestamp(backup.created_at)}
                            </td>
                          </tr>
                        ))}
                        {/* Show error messages for failed backups */}
                        {backups.filter((b: any) => b.status === 'failed' && b.error_message).map((backup: any) => (
                          <tr key={`${backup.backup_id}-error`} className="border-b border-border/50 bg-red-50/50 dark:bg-red-950/20">
                            <td colSpan={5} className="px-4 py-2 text-sm text-red-600 dark:text-red-400 italic">
                              └─ Error: {backup.error_message}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <div className="text-center py-12 text-muted-foreground">
                    <p>No backups found for this machine.</p>
                    <p className="text-sm mt-2">Backups will appear here once protection runs.</p>
                  </div>
                )}
              </ScrollArea>
            </CardContent>
          </Card>
        </div>
      </DialogContent>
    </Dialog>
  );
}
