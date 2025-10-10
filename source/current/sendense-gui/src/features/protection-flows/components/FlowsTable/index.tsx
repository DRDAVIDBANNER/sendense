"use client";

import { useState, useMemo } from "react";
import { ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Flow, FlowsTableProps, FlowRowProps } from "../../types";
import { FlowRow } from "./FlowRow";
import { useAllFlowsProgress } from "../../hooks/useFlowProgress";
import { useExecuteFlow } from "../../hooks/useProtectionFlows";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

export function FlowsTable({ flows, onSelectFlow, selectedFlowId, onEdit, onDelete }: FlowsTableProps) {
  const [sortColumn, setSortColumn] = useState<string>('name');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  // Optimistic UI state for immediate feedback
  const [optimisticRunning, setOptimisticRunning] = useState<Set<string>>(new Set());

  // Get progress data for all flows
  const flowIds = flows.map(f => f.id);
  const { data: progressData } = useAllFlowsProgress(flowIds, flows.length > 0);

  // Mutations and query client
  const executeFlowMutation = useExecuteFlow();
  const queryClient = useQueryClient();

  const sortedFlows = useMemo(() => {
    return [...flows].sort((a, b) => {
      const aValue = a[sortColumn as keyof Flow];
      const bValue = b[sortColumn as keyof Flow];

      // Handle undefined/null values
      if (aValue == null && bValue == null) return 0;
      if (aValue == null) return sortDirection === 'asc' ? -1 : 1;
      if (bValue == null) return sortDirection === 'asc' ? 1 : -1;

      if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1;
      if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });
  }, [flows, sortColumn, sortDirection]);

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  const getSortIcon = (column: string) => {
    if (sortColumn !== column) return <ArrowUpDown className="h-4 w-4" />;
    return sortDirection === 'asc' ? <ArrowUp className="h-4 w-4" /> : <ArrowDown className="h-4 w-4" />;
  };

  const handleEdit = (flow: Flow) => {
    if (onEdit) {
      onEdit(flow);  // Use parent handler if provided
    } else {
      console.log('Edit flow:', flow.id);  // Fallback
    }
  };

  const handleDelete = (flow: Flow) => {
    if (onDelete) {
      onDelete(flow);  // Use parent handler if provided
    } else {
      console.log('Delete flow:', flow.id);  // Fallback
    }
  };

  const handleRunNow = async (flow: Flow) => {
    console.log('🔥 RunNow clicked for flow:', flow.id, flow.name);
    // 1. Optimistic UI update - immediate feedback
    setOptimisticRunning(prev => new Set(prev).add(flow.id));
    console.log('✅ Optimistic state set for flow:', flow.id);

    try {
      // 2. Execute the flow
      await executeFlowMutation.mutateAsync(flow.id);

      // 3. Show success toast
      toast.success(`Starting backup for ${flow.name}`, {
        description: "Backup execution has begun",
        duration: 3000,
      });

      // 4. DELAYED poll for progress - give user time to see optimistic state first
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['all-flows-progress'] });
        queryClient.invalidateQueries({ queryKey: ['protection-flows'] });
      }, 1500); // Wait 1.5 seconds so user can see optimistic state

      // 5. Remove optimistic state after delay (let real data take over)
      setTimeout(() => {
        setOptimisticRunning(prev => {
          const next = new Set(prev);
          next.delete(flow.id);
          return next;
        });
      }, 3000);

    } catch (error: any) {
      // On error, remove optimistic state and show error
      setOptimisticRunning(prev => {
        const next = new Set(prev);
        next.delete(flow.id);
        return next;
      });

      toast.error(`Failed to start backup for ${flow.name}`, {
        description: error?.message || "An unexpected error occurred",
        duration: 5000,
      });
    }
  };

  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow className="bg-muted/50">
            <TableHead className="w-[300px]">
              <button
                className="flex items-center gap-2 hover:text-primary transition-colors"
                onClick={() => handleSort('name')}
              >
                Name
                {getSortIcon('name')}
              </button>
            </TableHead>
            <TableHead className="w-[120px]">
              <button
                className="flex items-center gap-2 hover:text-primary transition-colors"
                onClick={() => handleSort('type')}
              >
                Type
                {getSortIcon('type')}
              </button>
            </TableHead>
            <TableHead className="w-[120px]">
              <button
                className="flex items-center gap-2 hover:text-primary transition-colors"
                onClick={() => handleSort('status')}
              >
                Status
                {getSortIcon('status')}
              </button>
            </TableHead>
            <TableHead className="w-[160px]">
              <button
                className="flex items-center gap-2 hover:text-primary transition-colors"
                onClick={() => handleSort('lastRun')}
              >
                Last Run
                {getSortIcon('lastRun')}
              </button>
            </TableHead>
            <TableHead className="w-[160px]">
              <button
                className="flex items-center gap-2 hover:text-primary transition-colors"
                onClick={() => handleSort('nextRun')}
              >
                Next Run
                {getSortIcon('nextRun')}
              </button>
            </TableHead>
            <TableHead className="w-[100px]">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sortedFlows.length === 0 ? (
            <TableRow>
              <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                No protection flows found
              </TableCell>
            </TableRow>
          ) : (
            sortedFlows.map((flow) => {
              const flowProgress = progressData?.[flow.id];
              const isOptRunning = optimisticRunning.has(flow.id);

              if (flow.name.includes('pgtest')) {
                console.log('🎯 Rendering FlowRow for:', flow.name, {
                  progress: flowProgress?.progress,
                  isOptimisticallyRunning: isOptRunning,
                  optimisticRunningSet: Array.from(optimisticRunning)
                });
              }

              return (
                <FlowRow
                  key={flow.id}
                  flow={{
                    ...flow,
                    progress: flowProgress?.progress, // Add progress to flow object
                    isOptimisticallyRunning: isOptRunning // Add optimistic state
                  }}
                  isSelected={selectedFlowId === flow.id}
                  onSelect={onSelectFlow}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                  onRunNow={handleRunNow}
                />
              );
            })
          )}
        </TableBody>
      </Table>
    </div>
  );
}
