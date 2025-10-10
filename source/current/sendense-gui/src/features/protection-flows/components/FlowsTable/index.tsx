"use client";

import { useState, useMemo } from "react";
import { ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Flow, FlowsTableProps, FlowRowProps } from "../../types";
import { FlowRow } from "./FlowRow";
import { useAllFlowsProgress } from "../../hooks/useFlowProgress";

export function FlowsTable({ flows, onSelectFlow, selectedFlowId }: FlowsTableProps) {
  const [sortColumn, setSortColumn] = useState<string>('name');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  // Get progress data for all flows
  const flowIds = flows.map(f => f.id);
  const { data: progressData } = useAllFlowsProgress(flowIds, flows.length > 0);

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
    // TODO: Open edit modal
    console.log('Edit flow:', flow.id);
  };

  const handleDelete = (flow: Flow) => {
    // TODO: Open delete confirmation modal
    console.log('Delete flow:', flow.id);
  };

  const handleRunNow = (flow: Flow) => {
    // TODO: Start flow execution
    console.log('Run flow now:', flow.id);
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
              return (
                <FlowRow
                  key={flow.id}
                  flow={{
                    ...flow,
                    progress: flowProgress?.progress // Add progress to flow object
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
