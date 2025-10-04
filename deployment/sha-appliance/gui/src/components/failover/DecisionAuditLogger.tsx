'use client';

import React, { createContext, useContext, useState, useCallback } from 'react';

export interface DecisionAuditEntry {
  id: string;
  timestamp: string;
  decision_type: 'pre_flight_config' | 'rollback_decision' | 'mid_flight_decision';
  vm_name: string;
  vm_context_id: string;
  failover_type: 'live' | 'test';
  decision_data: {
    question?: string;
    selected_option?: string;
    configuration?: Record<string, any>;
    user_choices?: Record<string, any>;
  };
  metadata: {
    job_id?: string;
    phase?: string;
    user_agent: string;
    session_id?: string;
  };
}

interface DecisionAuditContextType {
  logDecision: (entry: Omit<DecisionAuditEntry, 'id' | 'timestamp' | 'metadata'>) => void;
  getDecisionHistory: (vmName?: string) => DecisionAuditEntry[];
  exportAuditLog: () => string;
}

const DecisionAuditContext = createContext<DecisionAuditContextType | null>(null);

export const useDecisionAudit = () => {
  const context = useContext(DecisionAuditContext);
  if (!context) {
    throw new Error('useDecisionAudit must be used within a DecisionAuditProvider');
  }
  return context;
};

export const DecisionAuditProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [auditLog, setAuditLog] = useState<DecisionAuditEntry[]>([]);

  const generateSessionId = useCallback(() => {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }, []);

  const logDecision = useCallback((entry: Omit<DecisionAuditEntry, 'id' | 'timestamp' | 'metadata'>) => {
    const auditEntry: DecisionAuditEntry = {
      ...entry,
      id: `audit_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      timestamp: new Date().toISOString(),
      metadata: {
        job_id: entry.decision_data.configuration?.job_id,
        phase: entry.decision_data.configuration?.current_phase,
        user_agent: navigator.userAgent,
        session_id: generateSessionId()
      }
    };

    setAuditLog(prev => [...prev, auditEntry]);

    // Log to console for debugging
    console.log('📋 DECISION AUDIT: Logged decision', {
      decision_type: auditEntry.decision_type,
      vm_name: auditEntry.vm_name,
      failover_type: auditEntry.failover_type,
      timestamp: auditEntry.timestamp
    });

    // Send to backend for persistent storage
    sendAuditToBackend(auditEntry);
  }, [generateSessionId]);

  const sendAuditToBackend = async (entry: DecisionAuditEntry) => {
    try {
      const response = await fetch('/api/failover/audit/decision', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(entry)
      });

      if (response.ok) {
        console.log('✅ DECISION AUDIT: Sent to backend successfully');
      } else {
        console.error('❌ DECISION AUDIT: Failed to send to backend', response.status);
      }
    } catch (error) {
      console.error('❌ DECISION AUDIT: Network error sending to backend', error);
    }
  };

  const getDecisionHistory = useCallback((vmName?: string) => {
    if (vmName) {
      return auditLog.filter(entry => entry.vm_name === vmName);
    }
    return auditLog;
  }, [auditLog]);

  const exportAuditLog = useCallback(() => {
    const exportData = {
      export_timestamp: new Date().toISOString(),
      total_entries: auditLog.length,
      entries: auditLog
    };
    return JSON.stringify(exportData, null, 2);
  }, [auditLog]);

  return (
    <DecisionAuditContext.Provider value={{ logDecision, getDecisionHistory, exportAuditLog }}>
      {children}
    </DecisionAuditContext.Provider>
  );
};

// Helper hooks for specific decision types
export const usePreFlightAudit = () => {
  const { logDecision } = useDecisionAudit();

  const logPreFlightConfiguration = useCallback((
    vmName: string,
    vmContextId: string,
    failoverType: 'live' | 'test',
    configuration: Record<string, any>
  ) => {
    logDecision({
      decision_type: 'pre_flight_config',
      vm_name: vmName,
      vm_context_id: vmContextId,
      failover_type: failoverType,
      decision_data: {
        configuration,
        user_choices: configuration
      }
    });
  }, [logDecision]);

  return { logPreFlightConfiguration };
};

export const useRollbackAudit = () => {
  const { logDecision } = useDecisionAudit();

  const logRollbackDecision = useCallback((
    vmName: string,
    vmContextId: string,
    failoverType: 'live' | 'test',
    question: string,
    selectedOption: string,
    rollbackOptions: Record<string, any>
  ) => {
    logDecision({
      decision_type: 'rollback_decision',
      vm_name: vmName,
      vm_context_id: vmContextId,
      failover_type: failoverType,
      decision_data: {
        question,
        selected_option: selectedOption,
        user_choices: rollbackOptions
      }
    });
  }, [logDecision]);

  return { logRollbackDecision };
};

export const useMidFlightAudit = () => {
  const { logDecision } = useDecisionAudit();

  const logMidFlightDecision = useCallback((
    vmName: string,
    vmContextId: string,
    failoverType: 'live' | 'test',
    phase: string,
    question: string,
    selectedOption: string,
    jobId?: string
  ) => {
    logDecision({
      decision_type: 'mid_flight_decision',
      vm_name: vmName,
      vm_context_id: vmContextId,
      failover_type: failoverType,
      decision_data: {
        question,
        selected_option: selectedOption,
        configuration: {
          job_id: jobId,
          current_phase: phase
        }
      }
    });
  }, [logDecision]);

  return { logMidFlightDecision };
};

// Audit Log Viewer Component
export const DecisionAuditViewer: React.FC<{ vmName?: string }> = ({ vmName }) => {
  const { getDecisionHistory, exportAuditLog } = useDecisionAudit();
  const history = getDecisionHistory(vmName);

  const handleExport = () => {
    const auditData = exportAuditLog();
    const blob = new Blob([auditData], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `failover_audit_log_${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  if (history.length === 0) {
    return (
      <div className="text-center py-4 text-gray-500">
        No decision audit entries {vmName ? `for ${vmName}` : 'found'}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold">
          Decision Audit Log {vmName && `- ${vmName}`}
        </h3>
        <button
          onClick={handleExport}
          className="px-3 py-1 bg-blue-500 text-white rounded text-sm hover:bg-blue-600"
        >
          Export Log
        </button>
      </div>
      
      <div className="space-y-2">
        {history.map(entry => (
          <div key={entry.id} className="border rounded p-3 bg-gray-50">
            <div className="flex justify-between items-start mb-2">
              <div>
                <span className="font-medium">{entry.decision_type.replace(/_/g, ' ').toUpperCase()}</span>
                <span className="ml-2 text-sm text-gray-600">
                  {entry.vm_name} ({entry.failover_type})
                </span>
              </div>
              <span className="text-xs text-gray-500">
                {new Date(entry.timestamp).toLocaleString()}
              </span>
            </div>
            
            {entry.decision_data.question && (
              <p className="text-sm mb-1">
                <span className="font-medium">Question:</span> {entry.decision_data.question}
              </p>
            )}
            
            {entry.decision_data.selected_option && (
              <p className="text-sm mb-1">
                <span className="font-medium">Selected:</span> {entry.decision_data.selected_option}
              </p>
            )}
            
            {entry.metadata.job_id && (
              <p className="text-xs text-gray-500">
                Job ID: {entry.metadata.job_id}
              </p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};
