"use client";

import { useState, useEffect } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Flow } from "./types";

interface FlowDetailsPanelProps {
  selectedFlow?: Flow;
  className?: string;
}

export function FlowDetailsPanel({ selectedFlow, className }: FlowDetailsPanelProps) {
  const [height, setHeight] = useState(() => {
    if (typeof window !== 'undefined') {
      return parseInt(localStorage.getItem('flowDetailsPanelHeight') || '400');
    }
    return 400;
  });

  const [isDragging, setIsDragging] = useState(false);

  useEffect(() => {
    localStorage.setItem('flowDetailsPanelHeight', height.toString());
  }, [height]);

  const handleMouseDown = (e: React.MouseEvent) => {
    setIsDragging(true);
    const startY = e.clientY;
    const startHeight = height;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const deltaY = startY - moveEvent.clientY;
      const newHeight = Math.max(100, Math.min(
        window.innerHeight * 0.6,
        startHeight + deltaY
      ));
      setHeight(newHeight);
    };

    const handleMouseUp = () => {
      setIsDragging(false);
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
    document.body.style.cursor = 'row-resize';
    document.body.style.userSelect = 'none';
  };

  if (!selectedFlow) {
    return (
      <div className={`border-t border-border bg-card ${className}`} style={{ height: `${height}px` }}>
        <div className="flex items-center justify-center h-full text-muted-foreground">
          Select a flow to view details
        </div>
      </div>
    );
  }

  return (
    <div className={`border-t border-border bg-card ${className}`} style={{ height: `${height}px` }}>
      {/* Drag Handle */}
      <div
        className={`h-1 bg-border cursor-row-resize hover:bg-primary transition-colors ${
          isDragging ? 'bg-primary' : ''
        }`}
        onMouseDown={handleMouseDown}
      />

      {/* Content */}
      <div className="h-full overflow-hidden">
        <Tabs defaultValue="overview" className="h-full flex flex-col">
          <div className="px-6 py-4 border-b border-border">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="volumes">Volumes</TabsTrigger>
              <TabsTrigger value="history">History</TabsTrigger>
            </TabsList>
          </div>

          <div className="flex-1 overflow-auto">
            <TabsContent value="overview" className="p-6">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-medium">Flow Information</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">Name</label>
                      <p className="text-sm font-medium">{selectedFlow.name}</p>
                    </div>
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">Type</label>
                      <div className="mt-1">
                        <Badge variant="outline" className="capitalize">
                          {selectedFlow.type}
                        </Badge>
                      </div>
                    </div>
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">Status</label>
                      <div className="mt-1 flex items-center gap-2">
                        <div className={`w-2 h-2 rounded-full ${
                          selectedFlow.status === 'success' ? 'bg-green-500' :
                          selectedFlow.status === 'running' ? 'bg-blue-500' :
                          selectedFlow.status === 'warning' ? 'bg-yellow-500' :
                          selectedFlow.status === 'error' ? 'bg-red-500' :
                          'bg-gray-400'
                        }`} />
                        <span className="text-sm capitalize">{selectedFlow.status}</span>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-medium">Schedule</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">Last Run</label>
                      <p className="text-sm">
                        {new Date(selectedFlow.lastRun).toLocaleString()}
                      </p>
                    </div>
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">Next Run</label>
                      <p className="text-sm">
                        {new Date(selectedFlow.nextRun).toLocaleString()}
                      </p>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-medium">Progress</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div>
                      <div className="flex justify-between text-sm mb-2">
                        <span>Completion</span>
                        <span>{selectedFlow.progress || 0}%</span>
                      </div>
                      <Progress value={selectedFlow.progress || 0} className="h-2" />
                    </div>
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">Source</label>
                      <p className="text-sm">{selectedFlow.source}</p>
                    </div>
                    <div>
                      <label className="text-xs font-medium text-muted-foreground">Destination</label>
                      <p className="text-sm">{selectedFlow.destination}</p>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </TabsContent>

            <TabsContent value="volumes" className="p-6">
              <div className="space-y-4">
                <h3 className="text-lg font-medium">Volumes in Flow</h3>
                <div className="text-muted-foreground">
                  Volume details will be displayed here
                </div>
                {/* TODO: Add volume list */}
              </div>
            </TabsContent>

            <TabsContent value="history" className="p-6">
              <div className="space-y-4">
                <h3 className="text-lg font-medium">Execution History</h3>
                <div className="text-muted-foreground">
                  Recent executions will be displayed here
                </div>
                {/* TODO: Add history list */}
              </div>
            </TabsContent>
          </div>
        </Tabs>
      </div>
    </div>
  );
}
