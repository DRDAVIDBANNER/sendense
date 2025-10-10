"use client";

import { Table, TableHeader, TableHead, TableBody, TableRow, TableCell } from "@/components/ui/table";
import type { FlowMachineInfo } from "@/src/features/protection-flows/types";

interface FlowMachinesTableProps {
  machines: FlowMachineInfo[];
}

export function FlowMachinesTable({ machines }: FlowMachinesTableProps) {
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '—';
    const k = 1024;
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${['B','KB','MB','GB','TB'][i]}`;
  };

  const getOSIcon = (os: string): string => {
    if (os.toLowerCase().includes('windows')) return '🪟';
    if (os.toLowerCase().includes('linux')) return '🐧';
    return '💿';
  };

  const totalDisksGB = (disks: FlowMachineInfo['disks']): number =>
    disks.reduce((sum, d) => sum + d.size_gb, 0);

  if (machines.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        No machines found in this protection flow
      </div>
    );
  }

  return (
    <div className="bg-card border border-border rounded-lg overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-1/4">VM Name</TableHead>
            <TableHead className="w-1/6">OS</TableHead>
            <TableHead className="w-1/6">CPU/Memory</TableHead>
            <TableHead className="w-1/6">Disks</TableHead>
            <TableHead className="w-1/6">Backups</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {machines.map((machine) => (
            <TableRow key={machine.context_id}>
              <TableCell className="font-medium">
                <div className="flex items-center gap-2">
                  <span
                    className={`w-2 h-2 rounded-full ${
                      machine.power_state === 'poweredOn' ? 'bg-green-500' :
                      machine.power_state === 'poweredOff' ? 'bg-gray-400' :
                      'bg-red-500'
                    }`}
                  />
                  <span className="truncate">{machine.vm_name}</span>
                </div>
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <span className="text-lg">{getOSIcon(machine.os_type)}</span>
                  <span className="text-sm">{machine.os_type}</span>
                </div>
              </TableCell>
              <TableCell className="text-sm">
                {machine.cpu_count}c / {Math.round(machine.memory_mb / 1024)}GB
              </TableCell>
              <TableCell className="text-sm">
                {machine.disks.length} ({totalDisksGB(machine.disks)}GB)
              </TableCell>
              <TableCell className="text-sm">
                {machine.backup_stats.backup_count} ({formatBytes(machine.backup_stats.total_size_bytes)})
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
