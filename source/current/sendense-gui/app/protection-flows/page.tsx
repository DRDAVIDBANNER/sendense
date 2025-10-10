"use client";

import { useState, useEffect } from "react";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
import { Button } from "@/components/ui/button";
import { Plus, ClipboardList } from "lucide-react";
import { PageHeader } from "@/components/common/PageHeader";
import { FlowsTable, FlowDetailsPanel, JobLogsDrawer, CreateFlowModal, EditFlowModal, DeleteConfirmModal } from "@/components/features/protection-flows";
import { Flow } from "@/src/features/protection-flows/types";
import { useProtectionFlows, useCreateFlow, useUpdateFlow, useDeleteFlow, useExecuteFlow } from "@/src/features/protection-flows/hooks/useProtectionFlows";

export default function ProtectionFlowsPage() {
  const [selectedFlowId, setSelectedFlowId] = useState<string>();
  const { data: flowsData, isLoading, error } = useProtectionFlows();
  const flows = flowsData?.flows || [];

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

  // API mutations
  const createFlowMutation = useCreateFlow();
  const updateFlowMutation = useUpdateFlow();
  const deleteFlowMutation = useDeleteFlow();
  const executeFlowMutation = useExecuteFlow();

  const selectedFlow = flows.find(flow => flow.id === selectedFlowId);

  const handleCreateFlow = () => {
    setIsCreateModalOpen(true);
  };

  const handleSelectFlow = (flow: Flow) => {
    setSelectedFlowId(flow.id);
  };

  const handleCreateFlowSubmit = async (newFlowData: Omit<Flow, 'id' | 'status' | 'lastRun' | 'progress'>) => {
    // Ensure required fields are provided
    const flowData = {
      ...newFlowData,
      repository_id: newFlowData.flow_type === 'backup' ? (newFlowData.repository_id || '') : undefined,
      enabled: newFlowData.enabled ?? true,
      target_type: newFlowData.target_type || 'vm',
    };
    await createFlowMutation.mutateAsync(flowData);
    setIsCreateModalOpen(false);
  };

  const handleEditFlow = (flow: Flow) => {
    setEditingFlow(flow);
  };

  const handleUpdateFlow = async (flowId: string, updates: Partial<Flow>) => {
    await updateFlowMutation.mutateAsync({ id: flowId, flow: updates });
    setEditingFlow(null);
  };

  const handleDeleteFlow = (flow: Flow) => {
    setDeletingFlow(flow);
  };

  const handleConfirmDelete = async (flowId: string) => {
    await deleteFlowMutation.mutateAsync(flowId);
    if (selectedFlowId === flowId) {
      setSelectedFlowId(undefined);
    }
    setDeletingFlow(null);
  };

  // ❌ REMOVED: handleRunNow from page - FlowsTable has its own with optimistic UI

  return (
    <div className="h-screen bg-background">
      <PanelGroup direction="vertical">
        {/* Top Panel: Flows Table */}
        <Panel defaultSize={50} minSize={30}>
          <div className="flex flex-col h-full bg-background">
            {/* Compact header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-border shrink-0">
              <div>
                <h2 className="text-lg font-semibold text-foreground">
                  Backup & Replication Jobs
                </h2>
                <p className="text-xs text-muted-foreground">
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
                      ? 'bg-primary/20 text-primary'
                      : 'bg-muted text-muted-foreground hover:bg-muted/80'
                  }`}
                  title="Job Logs (Ctrl+L)"
                >
                  <ClipboardList className="h-5 w-5" />
                </button>
              </div>
            </div>

            {/* Table (no extra container) */}
            {/* FlowsTable uses page handlers for edit/delete (modals) but its own handleRunNow (optimistic UI) */}
            <div className="flex-1 overflow-auto">
              <FlowsTable
                flows={flows}
                selectedFlowId={selectedFlowId}
                onSelectFlow={handleSelectFlow}
                onEdit={handleEditFlow}
                onDelete={handleDeleteFlow}
              />
            </div>
          </div>
        </Panel>

        {/* Resize Handle */}
        <PanelResizeHandle className="h-1 bg-border hover:bg-primary transition-colors cursor-ns-resize" />

        {/* Lower Panel: Flow Details */}
        <Panel defaultSize={40} minSize={20}>
          <div className="h-full bg-background border-t border-border overflow-auto">
            {selectedFlow ? (
              <FlowDetailsPanel flow={selectedFlow} />
            ) : (
              <div className="flex items-center justify-center h-full">
                <p className="text-muted-foreground text-center">
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
