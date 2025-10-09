"use client";

import { useState, useEffect } from "react";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
import { Button } from "@/components/ui/button";
import { Plus, ClipboardList } from "lucide-react";
import { PageHeader } from "@/components/common/PageHeader";
import { FlowsTable, FlowDetailsPanel, JobLogsDrawer, CreateFlowModal, EditFlowModal, DeleteConfirmModal } from "@/components/features/protection-flows";
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

  // Job Logs Drawer state
  const [isLogsOpen, setIsLogsOpen] = useState(false);

  // Load drawer state from localStorage on mount
  useEffect(() => {
    const savedState = localStorage.getItem('jobLogsOpen')
    if (savedState) {
      setIsLogsOpen(JSON.parse(savedState))
    }
  }, [])

  // Save drawer state to localStorage when it changes
  useEffect(() => {
    localStorage.setItem('jobLogsOpen', JSON.stringify(isLogsOpen))
  }, [isLogsOpen])

  // Keyboard shortcut for toggling drawer
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === 'l') {
        e.preventDefault()
        setIsLogsOpen(prev => !prev)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

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
    <div className="h-screen bg-gray-900">
      <PanelGroup direction="vertical">
        {/* Top Panel: Flows Table */}
        <Panel defaultSize={50} minSize={30}>
          <div className="flex flex-col h-full bg-gray-900">
            {/* Compact header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700 shrink-0">
              <div>
                <h2 className="text-lg font-semibold text-white">
                  Backup & Replication Jobs
                </h2>
                <p className="text-xs text-gray-400">
                  Manage and monitor your protection flows across all environments
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Button onClick={handleCreateFlow} color="blue" size="sm" className="gap-2">
                  <Plus className="h-4 w-4" />
                  Create Flow
                </Button>
                <button
                  onClick={() => setIsLogsOpen(!isLogsOpen)}
                  className={`p-2 rounded-lg transition-colors ${
                    isLogsOpen
                      ? 'bg-blue-500/20 text-blue-400'
                      : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
                  }`}
                  title="Job Logs (Ctrl+L)"
                >
                  <ClipboardList className="h-5 w-5" />
                </button>
              </div>
            </div>

            {/* Table (no extra container) */}
            <div className="flex-1 overflow-auto">
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
        </Panel>

        {/* Resize Handle */}
        <PanelResizeHandle className="h-1 bg-gray-700 hover:bg-blue-500 transition-colors cursor-ns-resize" />

        {/* Lower Panel: Flow Details */}
        <Panel defaultSize={40} minSize={20}>
          <div className="h-full bg-gray-900 border-t border-gray-700 overflow-auto">
            {selectedFlow ? (
              <FlowDetailsPanel flow={selectedFlow} />
            ) : (
              <div className="flex items-center justify-center h-full">
                <p className="text-gray-400 text-center">
                  Select a flow to view details
                </p>
              </div>
            )}
          </div>
        </Panel>
      </PanelGroup>

      {/* Job Logs Drawer */}
      <JobLogsDrawer isOpen={isLogsOpen} onClose={() => setIsLogsOpen(false)} />

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
