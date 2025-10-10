"use client";

import { format } from "date-fns";
import { MoreHorizontal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { FlowRowProps, getUIStatus } from "../../types";

export function FlowRow({ flow, isSelected, onSelect, onEdit, onDelete, onRunNow }: FlowRowProps) {
  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Never';
    try {
      return format(new Date(dateString), 'MMM dd, yyyy HH:mm');
    } catch {
      return dateString;
    }
  };

  const uiStatus = getUIStatus(flow);

  // Check for optimistic running state
  const isOptimisticallyRunning = flow.isOptimisticallyRunning || false;
  const isRunning = uiStatus === 'running' || isOptimisticallyRunning;
  const hasProgress = flow.progress !== undefined && flow.progress >= 0; // Changed to >= 0 to show 0%

  // Show 0% progress if optimistically running
  const displayProgress = isOptimisticallyRunning ? 0 : (flow.progress || 0);

  return (
    <tr
      className={`border-b border-border hover:bg-muted/50 cursor-pointer transition-colors ${
        isSelected ? 'bg-primary/5 border-primary/20' : ''
      }`}
      onClick={() => onSelect(flow)}
    >
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
          <div className="w-2 h-2 rounded-full bg-primary flex-shrink-0" />
          <div className="flex-1">
            <div className="font-medium text-foreground">{flow.name}</div>
            {flow.source && flow.destination && (
              <div className="text-sm text-muted-foreground">
                {flow.source} → {flow.destination}
              </div>
            )}
            {/* Progress bar for running OR optimistically running flows */}
            {isRunning && (
              <div className="mt-2 flex items-center gap-2">
                <Progress
                  value={displayProgress}
                  className={`h-1.5 flex-1 ${isOptimisticallyRunning ? 'animate-pulse' : ''}`}
                />
                <span className="text-xs text-muted-foreground min-w-[3ch]">
                  {isOptimisticallyRunning ? '0%' : `${displayProgress}%`}
                </span>
              </div>
            )}
          </div>
        </div>
      </td>
      <td className="px-4 py-3">
        <Badge variant="outline" className="capitalize">
          {flow.flow_type}
        </Badge>
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center gap-2">
          <div className={`w-2 h-2 rounded-full ${
            isOptimisticallyRunning ? 'bg-blue-500 animate-pulse' :  // Blue pulse for starting
            uiStatus === 'success' ? 'bg-green-500' :
            uiStatus === 'running' ? 'bg-blue-500 animate-pulse' :
            uiStatus === 'warning' ? 'bg-yellow-500' :
            uiStatus === 'error' ? 'bg-red-500' :
            'bg-muted-foreground'
          }`} />
          <span className="capitalize text-sm">
            {isOptimisticallyRunning ? 'Starting' : uiStatus}
          </span>
        </div>
      </td>
      <td className="px-4 py-3 text-sm text-muted-foreground">
        {formatDate(flow.lastRun)}
      </td>
      <td className="px-4 py-3 text-sm text-muted-foreground">
        {formatDate(flow.nextRun)}
      </td>
      <td className="px-4 py-3">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              onClick={(e) => e.stopPropagation()}
              className="h-8 w-8 p-0"
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuItem
              onClick={(e) => { e.stopPropagation(); onEdit?.(flow); }}
              disabled={isOptimisticallyRunning || uiStatus === 'running'}
            >
              Edit Flow
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={(e) => { e.stopPropagation(); onRunNow?.(flow); }}
              disabled={isOptimisticallyRunning || uiStatus === 'running'}
            >
              {isOptimisticallyRunning || uiStatus === 'running' ? 'Running...' : 'Run Now'}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={(e) => { e.stopPropagation(); onDelete?.(flow); }}
              className="text-destructive focus:text-destructive"
              disabled={isOptimisticallyRunning || uiStatus === 'running'}
            >
              Delete Flow
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </td>
    </tr>
  );
}
