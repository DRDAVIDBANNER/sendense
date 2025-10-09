"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { HardDrive, Server, Clock } from "lucide-react";
import { useVMContexts, useVMBackups, useMountBackup } from "@/src/features/restore/hooks/useRestore";
import type { VMContext, BackupJob } from "@/src/features/restore/types";

interface BackupSelectorProps {
  onMount: (mountId: string) => void;
}

export function BackupSelector({ onMount }: BackupSelectorProps) {
  const [selectedVM, setSelectedVM] = useState<string>("");
  const [selectedBackup, setSelectedBackup] = useState<string>("");
  const [selectedDisk, setSelectedDisk] = useState<number>(0);

  const { data: vmsData, isLoading: loadingVMs } = useVMContexts();
  const { data: backupsData, isLoading: loadingBackups } = useVMBackups(selectedVM);
  const mountMutation = useMountBackup();

  const vms = vmsData?.vm_contexts || [];
  const backups = backupsData?.backups || [];

  const handleMount = async () => {
    if (!selectedBackup) return;

    try {
      const mountRequest = {
        backup_id: selectedBackup,
        disk_index: selectedDisk
      };

      const result = await mountMutation.mutateAsync(mountRequest);
      onMount(result.mount_id);
    } catch (error) {
      console.error('Failed to mount backup:', error);
    }
  };

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return dateString;
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <HardDrive className="h-5 w-5" />
          Select VM and Backup
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* VM Selection */}
        <div className="space-y-2">
          <Label htmlFor="vm">Virtual Machine</Label>
          <Select value={selectedVM} onValueChange={(value) => {
            setSelectedVM(value);
            setSelectedBackup("");
            setSelectedDisk(0);
          }}>
            <SelectTrigger disabled={loadingVMs}>
              <SelectValue placeholder={
                loadingVMs ? "Loading VMs..." : "Select a VM"
              } />
            </SelectTrigger>
            <SelectContent>
              {vms.map((vm) => (
                <SelectItem key={vm.context_id} value={vm.context_id}>
                  <div className="flex flex-col">
                    <div className="flex items-center gap-2">
                      <div className={`w-2 h-2 rounded-full ${
                        vm.power_state === 'poweredOn' ? 'bg-green-500' :
                        vm.power_state === 'poweredOff' ? 'bg-gray-400' : 'bg-red-500'
                      }`} />
                      <span className="font-medium">{vm.vm_name}</span>
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {vm.vcenter_host} • {vm.os_type}
                    </span>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Backup Selection */}
        {selectedVM && (
          <div className="space-y-2">
            <Label htmlFor="backup">Backup</Label>
            <Select
              value={selectedBackup}
              onValueChange={(value) => {
                setSelectedBackup(value);
                setSelectedDisk(0);
              }}
              disabled={loadingBackups}
            >
              <SelectTrigger>
                <SelectValue placeholder={
                  loadingBackups ? "Loading backups..." : "Select a backup"
                } />
              </SelectTrigger>
              <SelectContent>
                {backups
                  .filter(backup => backup.status === 'completed')
                  .map((backup) => (
                    <SelectItem key={backup.id} value={backup.backup_id}>
                      <div className="flex flex-col">
                        <div className="flex items-center gap-2">
                          <span className="font-medium">
                            {backup.backup_type === 'full' ? 'Full' : 'Incremental'} Backup
                          </span>
                          <span className={`px-2 py-0.5 text-xs rounded ${
                            backup.backup_type === 'full'
                              ? 'bg-blue-500/20 text-blue-400'
                              : 'bg-green-500/20 text-green-400'
                          }`}>
                            {backup.backup_type}
                          </span>
                        </div>
                        <span className="text-xs text-muted-foreground">
                          {formatDate(backup.completed_at || backup.started_at)}
                          {' • '}
                          {formatSize(backup.total_size_bytes)}
                          {' • '}
                          {backup.disks_count} disk{backup.disks_count !== 1 ? 's' : ''}
                        </span>
                      </div>
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
        )}

        {/* Disk Selection */}
        {selectedBackup && backups.length > 0 && (
          <div className="space-y-2">
            <Label htmlFor="disk">Disk to Mount</Label>
            <Select
              value={selectedDisk.toString()}
              onValueChange={(value) => setSelectedDisk(parseInt(value))}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Array.from({ length: backups.find(b => b.backup_id === selectedBackup)?.disks_count || 1 }, (_, i) => (
                  <SelectItem key={i} value={i.toString()}>
                    Disk {i} {i === 0 ? '(System Disk)' : '(Data Disk)'}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        {/* Mount Button */}
        <Button
          onClick={handleMount}
          disabled={!selectedBackup || mountMutation.isPending}
          className="w-full"
        >
          {mountMutation.isPending ? (
            <>
              <Clock className="h-4 w-4 mr-2 animate-spin" />
              Mounting Backup...
            </>
          ) : (
            <>
              <Server className="h-4 w-4 mr-2" />
              Mount Backup
            </>
          )}
        </Button>
      </CardContent>
    </Card>
  );
}
