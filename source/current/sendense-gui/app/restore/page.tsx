"use client";

import { useState } from "react";
import { BackupSelector } from "@/components/features/restore/BackupSelector";
import { FileBrowser } from "@/components/features/restore/FileBrowser";
import { ActiveMountsPanel } from "@/components/features/restore/ActiveMountsPanel";
import { RotateCcw } from "lucide-react";

export default function RestorePage() {
  const [activeMountId, setActiveMountId] = useState<string | null>(null);
  const [currentPath, setCurrentPath] = useState("/");

  const handleMount = (mountId: string) => {
    setActiveMountId(mountId);
    setCurrentPath("/");
  };

  const handleNavigate = (path: string) => {
    setCurrentPath(path);
  };

  return (
    <div className="h-screen bg-background">
      <div className="container mx-auto px-6 py-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-border bg-card">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <RotateCcw className="h-6 w-6 text-primary" />
              <h1 className="text-2xl font-bold text-foreground">File-Level Restore</h1>
            </div>
            <p className="text-muted-foreground">
              Mount VM backups and download individual files or directories
            </p>
          </div>
        </div>

        {/* Main Content Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Left Column: Backup Selection and File Browser */}
          <div className="lg:col-span-2 space-y-6">
            {/* Backup Selector */}
            <BackupSelector onMount={handleMount} />

            {/* File Browser - Only show when we have an active mount */}
            {activeMountId && (
              <FileBrowser
                mountId={activeMountId}
                currentPath={currentPath}
                onNavigate={handleNavigate}
              />
            )}
          </div>

          {/* Right Column: Active Mounts Panel */}
          <div className="space-y-6">
            <ActiveMountsPanel />

            {/* Stats Card */}
            <div className="grid grid-cols-2 gap-4">
              <div className="bg-card border border-border rounded-lg p-4 text-center">
                <div className="text-2xl font-bold text-foreground">
                  {activeMountId ? "1" : "0"}
                </div>
                <div className="text-sm text-muted-foreground">Active Mount</div>
              </div>
              <div className="bg-card border border-border rounded-lg p-4 text-center">
                <div className="text-2xl font-bold text-foreground">47</div>
                <div className="text-sm text-muted-foreground">Total Restores</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
