"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Unlink, Clock, HardDrive, Server } from "lucide-react";
import { useActiveMounts, useUnmountBackup } from "@/src/features/restore/hooks/useRestore";
import type { RestoreMount } from "@/src/features/restore/types";

export function ActiveMountsPanel() {
  const { data: mountsData, isLoading } = useActiveMounts();
  const unmountMutation = useUnmountBackup();
  const mounts = mountsData?.mounts || [];

  const getTimeRemaining = (expiresAt: string) => {
    const now = new Date();
    const expiry = new Date(expiresAt);
    const diff = expiry.getTime() - now.getTime();

    if (diff <= 0) return "Expired";

    const minutes = Math.floor(diff / 60000);
    if (minutes < 60) {
      return `Expires in ${minutes} min`;
    }

    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    return `Expires in ${hours}h ${remainingMinutes}m`;
  };

  const handleUnmount = async (mountId: string) => {
    try {
      await unmountMutation.mutateAsync(mountId);
    } catch (error) {
      console.error('Failed to unmount:', error);
    }
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Active Mounts
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center text-muted-foreground py-4">
            Loading active mounts...
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Server className="h-5 w-5" />
          Active Mounts ({mounts.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        {mounts.length === 0 ? (
          <div className="text-center text-muted-foreground py-4">
            No active mounts
          </div>
        ) : (
          <div className="space-y-3">
            {mounts.map((mount) => (
              <div key={mount.mount_id} className="flex items-center justify-between p-3 border rounded-lg">
                <div className="flex items-center gap-3">
                  <div className={`w-2 h-2 rounded-full ${
                    mount.status === 'mounted' ? 'bg-green-500' :
                    mount.status === 'mounting' ? 'bg-yellow-500' :
                    'bg-red-500'
                  }`} />
                  <div>
                    <div className="font-medium">
                      {mount.backup_id.split('-')[1] || mount.backup_id} • Disk {mount.disk_index}
                    </div>
                    <div className="text-sm text-muted-foreground">
                      {mount.filesystem_type} • {getTimeRemaining(mount.expires_at)}
                    </div>
                  </div>
                </div>

                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleUnmount(mount.mount_id)}
                  disabled={unmountMutation.isPending}
                  className="gap-2"
                >
                  {unmountMutation.isPending ? (
                    <>
                      <Clock className="h-4 w-4 animate-spin" />
                      Unmounting...
                    </>
                  ) : (
                    <>
                      <Unlink className="h-4 w-4" />
                      Unmount
                    </>
                  )}
                </Button>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
