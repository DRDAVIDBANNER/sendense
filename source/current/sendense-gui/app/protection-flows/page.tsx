"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import { PageHeader } from "@/components/common/PageHeader";
import { FlowsTable, FlowDetailsPanel, JobLogPanel, CreateFlowModal, EditFlowModal, DeleteConfirmModal } from "@/components/features/protection-flows";
import { Flow } from "@/src/features/protection-flows/types";

const mockFlows: Flow[] = [
  {
    id: '1',
    name: 'Daily VM Backup - pgtest1',
    type: 'backup',
    status: 'success',
    lastRun: '2025-10-06T10:00:00Z',
    nextRun: '2025-10-07T10:00:00Z',
    source: 'vCenter-ESXi-01',
    destination: 'CloudStack-Primary',
    progress: 100
  },
  {
    id: '2',
    name: 'Hourly Replication - web-servers',
    type: 'replication',
    status: 'running',
    lastRun: '2025-10-06T09:00:00Z',
    nextRun: '2025-10-06T10:00:00Z',
    source: 'vCenter-ESXi-02',
    destination: 'CloudStack-DR',
    progress: 65
  },
  {
    id: '3',
    name: 'Weekly Archive - legacy-apps',
    type: 'backup',
    status: 'warning',
    lastRun: '2025-10-03T08:00:00Z',
    nextRun: '2025-10-10T08:00:00Z',
    source: 'vCenter-ESXi-01',
    destination: 'CloudStack-Archive',
    progress: 0
  },
  {
    id: '4',
    name: 'Critical DB Backup',
    type: 'backup',
    status: 'error',
    lastRun: '2025-10-06T02:00:00Z',
    nextRun: '2025-10-06T14:00:00Z',
    source: 'vCenter-ESXi-03',
    destination: 'CloudStack-Primary',
    progress: 0
  },
  {
    id: '5',
    name: 'Dev Environment Sync',
    type: 'replication',
    status: 'pending',
    lastRun: '2025-10-05T16:00:00Z',
    nextRun: '2025-10-06T12:00:00Z',
    source: 'vCenter-ESXi-02',
    destination: 'CloudStack-Dev',
    progress: 0
  }
];

export default function ProtectionFlowsPage() {
  const [selectedFlowId, setSelectedFlowId] = useState<string>();
  const [flows, setFlows] = useState<Flow[]>(mockFlows);

  // Modal states
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingFlow, setEditingFlow] = useState<Flow | null>(null);
  const [deletingFlow, setDeletingFlow] = useState<Flow | null>(null);

  const selectedFlow = flows.find(flow => flow.id === selectedFlowId);

  const handleCreateFlow = () => {
    setIsCreateModalOpen(true);
  };

  const handleSelectFlow = (flow: Flow) => {
    setSelectedFlowId(flow.id);
  };

  const handleCreateFlowSubmit = (newFlowData: Omit<Flow, 'id' | 'status' | 'lastRun' | 'progress'>) => {
    const newFlow: Flow = {
      ...newFlowData,
      id: Date.now().toString(),
      status: 'pending',
      lastRun: new Date().toISOString(),
      progress: 0
    };
    setFlows(prev => [...prev, newFlow]);
  };

  const handleEditFlow = (flow: Flow) => {
    setEditingFlow(flow);
  };

  const handleUpdateFlow = (flowId: string, updates: Partial<Flow>) => {
    setFlows(prev => prev.map(flow =>
      flow.id === flowId ? { ...flow, ...updates } : flow
    ));
    setEditingFlow(null);
  };

  const handleDeleteFlow = (flow: Flow) => {
    setDeletingFlow(flow);
  };

  const handleConfirmDelete = (flowId: string) => {
    setFlows(prev => prev.filter(flow => flow.id !== flowId));
    if (selectedFlowId === flowId) {
      setSelectedFlowId(undefined);
    }
    setDeletingFlow(null);
  };

  const handleRunNow = (flow: Flow) => {
    // TODO: Implement run now functionality
    console.log('Run flow now:', flow.id);
  };

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Protection Flows"
        breadcrumbs={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Protection Flows" }
        ]}
        actions={
          <Button onClick={handleCreateFlow} className="gap-2">
            <Plus className="h-4 w-4" />
            Create Flow
          </Button>
        }
      />

      <div className="flex-1 overflow-hidden flex">
        {/* Left Section: Table + Details Panel */}
        <div className="flex-1 flex flex-col min-w-0">
          {/* Table Section - takes remaining space */}
          <div className="flex-1 overflow-auto p-6">
            <div className="max-w-7xl mx-auto">
              <div className="mb-6">
                <h2 className="text-lg font-semibold text-foreground mb-2">
                  Backup & Replication Jobs
                </h2>
                <p className="text-muted-foreground">
                  Manage and monitor your protection flows across all environments
                </p>
              </div>

              <FlowsTable
                flows={flows}
                selectedFlowId={selectedFlowId}
                onSelectFlow={handleSelectFlow}
                onEdit={handleEditFlow}
                onDelete={handleDeleteFlow}
                onRunNow={handleRunNow}
              />
            </div>
          </div>

          {/* Details Panel - resizable */}
          <FlowDetailsPanel selectedFlow={selectedFlow} />
        </div>

        {/* Right Section: Job Log Panel */}
        <JobLogPanel />
      </div>

      {/* Modals */}
      <CreateFlowModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onCreate={handleCreateFlowSubmit}
      />

      <EditFlowModal
        isOpen={!!editingFlow}
        onClose={() => setEditingFlow(null)}
        flow={editingFlow}
        onUpdate={handleUpdateFlow}
      />

      <DeleteConfirmModal
        isOpen={!!deletingFlow}
        onClose={() => setDeletingFlow(null)}
        flow={deletingFlow}
        onConfirm={handleConfirmDelete}
      />
    </div>
  );
}
