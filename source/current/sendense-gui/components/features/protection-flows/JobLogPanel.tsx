"use client";

import { useState, useEffect, useRef } from "react";
import { ChevronLeft, ChevronRight, Filter } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";

interface LogEntry {
  id: string;
  timestamp: string;
  level: 'info' | 'warning' | 'error';
  message: string;
  source?: string;
}

interface JobLogPanelProps {
  className?: string;
}

const mockLogs: LogEntry[] = [
  { id: '1', timestamp: '2025-10-06T10:00:01Z', level: 'info', message: 'Starting backup job for pgtest1', source: 'Backup Engine' },
  { id: '2', timestamp: '2025-10-06T10:00:02Z', level: 'info', message: 'Connecting to vCenter server', source: 'VMware API' },
  { id: '3', timestamp: '2025-10-06T10:00:03Z', level: 'info', message: 'Snapshot created successfully', source: 'VMware API' },
  { id: '4', timestamp: '2025-10-06T10:00:04Z', level: 'info', message: 'Transferring data: 25% complete', source: 'NBD Transfer' },
  { id: '5', timestamp: '2025-10-06T10:00:05Z', level: 'warning', message: 'Network latency detected, adjusting buffer size', source: 'NBD Transfer' },
  { id: '6', timestamp: '2025-10-06T10:00:06Z', level: 'info', message: 'Transferring data: 50% complete', source: 'NBD Transfer' },
  { id: '7', timestamp: '2025-10-06T10:00:07Z', level: 'info', message: 'Transferring data: 75% complete', source: 'NBD Transfer' },
  { id: '8', timestamp: '2025-10-06T10:00:08Z', level: 'info', message: 'Transferring data: 100% complete', source: 'NBD Transfer' },
  { id: '9', timestamp: '2025-10-06T10:00:09Z', level: 'info', message: 'Verifying backup integrity', source: 'Validation Engine' },
  { id: '10', timestamp: '2025-10-06T10:00:10Z', level: 'info', message: 'Backup completed successfully', source: 'Backup Engine' },
];

export function JobLogPanel({ className }: JobLogPanelProps) {
  const [isExpanded, setIsExpanded] = useState(() => {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('jobLogPanelExpanded') !== 'false';
    }
    return true;
  });

  const [width, setWidth] = useState(() => {
    if (typeof window !== 'undefined') {
      return parseInt(localStorage.getItem('jobLogPanelWidth') || '420');
    }
    return 420;
  });

  const [filter, setFilter] = useState<'all' | 'info' | 'warning' | 'error'>('all');
  const [autoScroll, setAutoScroll] = useState(true);
  const [logs, setLogs] = useState<LogEntry[]>(mockLogs);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const [isDragging, setIsDragging] = useState(false);

  // Persist state to localStorage
  useEffect(() => {
    localStorage.setItem('jobLogPanelExpanded', isExpanded.toString());
  }, [isExpanded]);

  useEffect(() => {
    localStorage.setItem('jobLogPanelWidth', width.toString());
  }, [width]);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && scrollAreaRef.current) {
      const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]');
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight;
      }
    }
  }, [logs, autoScroll]);

  // Mock real-time log updates
  useEffect(() => {
    const interval = setInterval(() => {
      if (Math.random() < 0.3) { // 30% chance every 5 seconds
        const newLog: LogEntry = {
          id: Date.now().toString(),
          timestamp: new Date().toISOString(),
          level: Math.random() < 0.8 ? 'info' : Math.random() < 0.5 ? 'warning' : 'error',
          message: `System activity: ${Math.random().toString(36).substring(7)}`,
          source: 'System Monitor'
        };
        setLogs(prev => [...prev.slice(-19), newLog]); // Keep last 20 logs
      }
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  const filteredLogs = logs.filter(log => filter === 'all' || log.level === filter);

  const handleMouseDown = (e: React.MouseEvent) => {
    if (!isExpanded) return;

    setIsDragging(true);
    const startX = e.clientX;
    const startWidth = width;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const deltaX = startX - moveEvent.clientX;
      const newWidth = Math.max(200, Math.min(600, startWidth + deltaX));
      setWidth(newWidth);
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
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  };

  const toggleExpanded = () => {
    setIsExpanded(!isExpanded);
  };

  const getLogLevelColor = (level: string) => {
    switch (level) {
      case 'error': return 'text-red-400';
      case 'warning': return 'text-yellow-400';
      case 'info': return 'text-blue-400';
      default: return 'text-muted-foreground';
    }
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  return (
    <div className={`relative bg-card border-l border-border ${className}`}>
      {/* Collapse/Expand Toggle Button */}
      <Button
        variant="ghost"
        size="sm"
        onClick={toggleExpanded}
        className="absolute left-0 top-4 z-10 p-2 bg-card border border-border rounded-l shadow-sm hover:bg-muted"
        style={{ transform: 'translateX(-100%)' }}
      >
        {isExpanded ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
      </Button>

      {/* Drag Handle */}
      {isExpanded && (
        <div
          className={`absolute left-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary transition-colors ${
            isDragging ? 'bg-primary' : 'bg-border'
          }`}
          onMouseDown={handleMouseDown}
          style={{ transform: 'translateX(-2px)' }}
        />
      )}

      {/* Panel Content */}
      <div
        className="h-full flex flex-col"
        style={{ width: isExpanded ? `${width}px` : '0px' }}
      >
        {isExpanded && (
          <>
            {/* Header */}
            <div className="p-4 border-b border-border">
              <div className="flex items-center justify-between mb-3">
                <h3 className="font-medium text-foreground">Job Logs</h3>
                <Badge variant="secondary" className="text-xs">
                  Live
                </Badge>
              </div>

              {/* Filters */}
              <div className="flex items-center gap-2">
                <Select value={filter} onValueChange={(value: any) => setFilter(value)}>
                  <SelectTrigger className="h-8 w-24">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    <SelectItem value="info">Info</SelectItem>
                    <SelectItem value="warning">Warning</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                  </SelectContent>
                </Select>

                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setAutoScroll(!autoScroll)}
                  className={`h-8 px-2 ${autoScroll ? 'bg-primary/10 text-primary' : ''}`}
                >
                  Auto-scroll
                </Button>
              </div>
            </div>

            {/* Log Content */}
            <ScrollArea ref={scrollAreaRef} className="flex-1 p-4">
              <div className="space-y-1 font-mono text-xs">
                {filteredLogs.map((log) => (
                  <div key={log.id} className="flex gap-2 py-1 border-l-2 border-transparent hover:border-muted-foreground/20 hover:bg-muted/20 px-2 -mx-2 rounded">
                    <span className="text-muted-foreground shrink-0">
                      {formatTimestamp(log.timestamp)}
                    </span>
                    <span className={`font-medium uppercase shrink-0 ${getLogLevelColor(log.level)}`}>
                      [{log.level}]
                    </span>
                    <span className="text-foreground flex-1 break-all">
                      {log.message}
                    </span>
                    {log.source && (
                      <span className="text-muted-foreground shrink-0">
                        {log.source}
                      </span>
                    )}
                  </div>
                ))}
              </div>
            </ScrollArea>
          </>
        )}
      </div>
    </div>
  );
}
