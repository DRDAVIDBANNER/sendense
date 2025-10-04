'use client';

import React, { useState, useEffect } from 'react';
import { HiPlus, HiCheck, HiX, HiTrash, HiRefresh, HiExclamation, HiClock, HiClipboard, HiServer } from 'react-icons/hi';

interface PairingCode {
  pairing_code: string;
  expires_at: string;
  valid_for: number;
}

interface PendingEnrollment {
  id: string;
  vma_name?: string;
  vma_version?: string;
  vma_fingerprint?: string;
  vma_ip_address?: string;
  created_at: string;
  expires_at: string;
  status: string;
}

interface ActiveVMA {
  id: string;
  enrollment_id: string;
  vma_name: string;
  vma_fingerprint: string;
  connection_status: string;
  connected_at: string;
  last_seen_at?: string;
}

interface AuditEvent {
  id: number;
  enrollment_id?: string;
  event_type: string;
  vma_fingerprint?: string;
  source_ip?: string;
  approved_by?: string;
  event_details?: string;
  created_at: string;
}

interface AuditFilter {
  event_type: string;
  limit: number;
}

export const VMAEnrollmentManager = React.memo(() => {
  const [pairingCode, setPairingCode] = useState<PairingCode | null>(null);
  const [pendingEnrollments, setPendingEnrollments] = useState<PendingEnrollment[]>([]);
  const [activeVMAs, setActiveVMAs] = useState<ActiveVMA[]>([]);
  const [auditEvents, setAuditEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  
  // Modal states
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [showApprovalModal, setShowApprovalModal] = useState(false);
  const [showAuditModal, setShowAuditModal] = useState(false);
  const [showRevocationModal, setShowRevocationModal] = useState(false);
  const [selectedEnrollment, setSelectedEnrollment] = useState<PendingEnrollment | null>(null);
  const [selectedVMAForRevocation, setSelectedVMAForRevocation] = useState<ActiveVMA | null>(null);
  const [approvalNotes, setApprovalNotes] = useState('');
  const [revocationReason, setRevocationReason] = useState('');
  
  // Audit filtering
  const [auditFilter, setAuditFilter] = useState<AuditFilter>({
    event_type: '',
    limit: 50
  });

  const API_BASE = ''; // Always use Next.js proxy routes for consistent API access

  // Auto-refresh pending enrollments every 30 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      if (!loading) {
        loadPendingEnrollments();
        loadActiveVMAs();
      }
    }, 30000);

    return () => clearInterval(interval);
  }, [loading]);

  const generatePairingCode = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`${API_BASE}/api/v1/admin/vma/pairing-code`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          generated_by: 'admin', // TODO: Get from auth context
          valid_for: 600 // 10 minutes
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Failed to generate pairing code');
      }

      const data = await response.json();
      setPairingCode(data);
      setSuccess('Pairing code generated successfully');
      setShowGenerateModal(false);
      
      // Auto-expire the displayed code after the timeout
      setTimeout(() => {
        setPairingCode(null);
      }, data.valid_for * 1000);
      
    } catch (err) {
      setError(err instanceof Error ? err.message : 'VMA enrollment API not available yet');
      setShowGenerateModal(false);
    } finally {
      setLoading(false);
    }
  };

  const loadPendingEnrollments = async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/admin/vma/pending`);
      if (response.ok) {
        const data = await response.json();
        setPendingEnrollments(data.enrollments || []);
      }
    } catch (err) {
      console.debug('VMA enrollment API not available yet');
      setPendingEnrollments([]);
    }
  };

  const loadActiveVMAs = async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/admin/vma/active`);
      if (response.ok) {
        const data = await response.json();
        setActiveVMAs(data.connections || []);
      }
    } catch (err) {
      console.debug('VMA enrollment API not available yet');
      setActiveVMAs([]);
    }
  };

  const loadAuditLog = async () => {
    try {
      const params = new URLSearchParams({
        limit: auditFilter.limit.toString(),
        ...(auditFilter.event_type && { event_type: auditFilter.event_type })
      });
      
      const response = await fetch(`${API_BASE}/api/v1/admin/vma/audit?${params}`);
      if (response.ok) {
        const data = await response.json();
        setAuditEvents(data.events || []);
      }
    } catch (err) {
      console.debug('VMA audit API not available yet');
      setAuditEvents([]);
    }
  };

  const approveEnrollment = async (enrollmentId: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/api/v1/admin/vma/approve/${enrollmentId}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          approved_by: 'admin', // TODO: Get from auth context
          notes: approvalNotes
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Failed to approve enrollment');
      }

      setSuccess('VMA enrollment approved successfully');
      setShowApprovalModal(false);
      setApprovalNotes('');
      loadPendingEnrollments();
      loadActiveVMAs();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve enrollment');
    } finally {
      setLoading(false);
    }
  };

  const rejectEnrollment = async (enrollmentId: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/api/v1/admin/vma/reject/${enrollmentId}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          rejected_by: 'admin', // TODO: Get from auth context
          reason: 'Rejected by administrator'
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Failed to reject enrollment');
      }

      setSuccess('VMA enrollment rejected');
      loadPendingEnrollments();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reject enrollment');
    } finally {
      setLoading(false);
    }
  };

  const revokeVMAAccess = async (enrollmentId: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/api/v1/admin/vma/revoke/${enrollmentId}?revoked_by=admin`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Failed to revoke VMA access');
      }

      setSuccess('VMA access revoked successfully');
      loadActiveVMAs();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revoke VMA access');
    } finally {
      setLoading(false);
    }
  };

  const formatTimeAgo = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    
    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
    return `${Math.floor(diffMins / 1440)}d ago`;
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setSuccess('Copied to clipboard');
    } catch (err) {
      setError('Failed to copy to clipboard');
    }
  };

  const getTimeRemaining = (expiresAt: string) => {
    const now = new Date();
    const expiry = new Date(expiresAt);
    const diffMs = expiry.getTime() - now.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    
    if (diffMins <= 0) return 'Expired';
    if (diffMins < 60) return `${diffMins}m remaining`;
    return `${Math.floor(diffMins / 60)}h ${diffMins % 60}m remaining`;
  };

  return (
    <div className="space-y-6">
      {/* Alerts */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <div className="flex">
            <HiExclamation className="h-5 w-5 text-red-400 mr-3 mt-0.5" />
            <div className="text-red-800 dark:text-red-200">
              <strong>Error:</strong> {error}
            </div>
            <button
              onClick={() => setError(null)}
              className="ml-auto text-red-400 hover:text-red-600"
            >
              <HiX className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {success && (
        <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
          <div className="flex">
            <HiCheck className="h-5 w-5 text-green-400 mr-3 mt-0.5" />
            <div className="text-green-800 dark:text-green-200">
              <strong>Success:</strong> {success}
            </div>
            <button
              onClick={() => setSuccess(null)}
              className="ml-auto text-green-400 hover:text-green-600"
            >
              <HiX className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {/* Pairing Code Generation */}
      <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              VMA Pairing Code Generator
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Generate secure pairing codes for VMA enrollment (10-minute expiry)
            </p>
          </div>
          <button
            onClick={() => setShowGenerateModal(true)}
            disabled={loading}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
          >
            <HiPlus className="h-4 w-4 mr-2" />
            Generate Code
          </button>
        </div>

        {pairingCode && (
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <div className="text-3xl font-mono font-bold text-blue-600 dark:text-blue-400 mb-2">
                  {pairingCode.pairing_code}
                </div>
                <div className="flex items-center text-sm text-gray-600 dark:text-gray-400">
                  <HiClock className="h-4 w-4 mr-1" />
                  {getTimeRemaining(pairingCode.expires_at)}
                </div>
              </div>
              <button
                onClick={() => copyToClipboard(pairingCode.pairing_code)}
                className="inline-flex items-center px-3 py-2 border border-gray-300 dark:border-gray-600 text-sm font-medium rounded-md text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                <HiClipboard className="h-4 w-4 mr-2" />
                Copy Code
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Pending Enrollments */}
      <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Pending VMA Enrollments
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              VMAs awaiting administrator approval
            </p>
          </div>
          <button
            onClick={loadPendingEnrollments}
            disabled={loading}
            className="inline-flex items-center px-3 py-2 border border-gray-300 dark:border-gray-600 text-sm font-medium rounded-md text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
          >
            <HiRefresh className="h-4 w-4 mr-2" />
            Refresh
          </button>
        </div>

        {pendingEnrollments.length === 0 ? (
          <div className="text-center py-12 text-gray-500 dark:text-gray-400">
            <HiServer className="h-12 w-12 mx-auto mb-4 text-gray-300 dark:text-gray-600" />
            <h4 className="text-lg font-medium mb-2">No Pending Enrollments</h4>
            <p>VMAs will appear here when they request enrollment using a pairing code.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    VMA Details
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    SSH Key Fingerprint
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Source
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Requested
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                {pendingEnrollments.map((enrollment) => (
                  <tr key={enrollment.id} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                    <td className="px-6 py-4">
                      <div>
                        <div className="text-sm font-medium text-gray-900 dark:text-white">
                          {enrollment.vma_name || 'Unnamed VMA'}
                        </div>
                        <div className="text-sm text-gray-500 dark:text-gray-400">
                          Version: {enrollment.vma_version || 'Unknown'}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <code className="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded font-mono">
                        {enrollment.vma_fingerprint?.substring(0, 24)}...
                      </code>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
                      {enrollment.vma_ip_address || 'Unknown'}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
                      {formatTimeAgo(enrollment.created_at)}
                    </td>
                    <td className="px-6 py-4 text-sm">
                      <div className="flex space-x-2">
                        <button
                          onClick={() => {
                            setSelectedEnrollment(enrollment);
                            setShowApprovalModal(true);
                          }}
                          disabled={loading}
                          className="inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50"
                        >
                          <HiCheck className="h-3 w-3 mr-1" />
                          Approve
                        </button>
                        <button
                          onClick={() => rejectEnrollment(enrollment.id)}
                          disabled={loading}
                          className="inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
                        >
                          <HiX className="h-3 w-3 mr-1" />
                          Reject
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Active VMA Connections */}
      <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Active VMA Connections
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Currently connected VMAs with tunnel status
            </p>
          </div>
          <button
            onClick={loadActiveVMAs}
            disabled={loading}
            className="inline-flex items-center px-3 py-2 border border-gray-300 dark:border-gray-600 text-sm font-medium rounded-md text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
          >
            <HiRefresh className="h-4 w-4 mr-2" />
            Refresh
          </button>
        </div>

        {activeVMAs.length === 0 ? (
          <div className="text-center py-12 text-gray-500 dark:text-gray-400">
            <HiServer className="h-12 w-12 mx-auto mb-4 text-gray-300 dark:text-gray-600" />
            <h4 className="text-lg font-medium mb-2">No Active Connections</h4>
            <p>Approved VMAs will appear here when they establish tunnel connections.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    VMA Name
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Connection Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    SSH Fingerprint
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Connected
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Last Seen
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                {activeVMAs.map((vma) => (
                  <tr key={vma.id} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                    <td className="px-6 py-4 text-sm font-medium text-gray-900 dark:text-white">
                      {vma.vma_name}
                    </td>
                    <td className="px-6 py-4">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        vma.connection_status === 'connected' 
                          ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                          : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                      }`}>
                        {vma.connection_status}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <code className="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded font-mono">
                        {vma.vma_fingerprint.substring(0, 24)}...
                      </code>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
                      {formatTimeAgo(vma.connected_at)}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
                      {vma.last_seen_at ? formatTimeAgo(vma.last_seen_at) : 'Never'}
                    </td>
                    <td className="px-6 py-4 text-sm">
                      <button
                        onClick={() => {
                          setSelectedVMAForRevocation(vma);
                          setShowRevocationModal(true);
                        }}
                        disabled={loading}
                        className="inline-flex items-center px-3 py-1 border border-transparent text-xs font-medium rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
                      >
                        <HiTrash className="h-3 w-3 mr-1" />
                        Revoke
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Audit Log Section */}
      <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Security Audit Log
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Complete audit trail of VMA enrollment and connection events
            </p>
          </div>
          <div className="flex space-x-3">
            <select
              value={auditFilter.event_type}
              onChange={(e) => setAuditFilter(prev => ({ ...prev, event_type: e.target.value }))}
              className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
            >
              <option value="">All Events</option>
              <option value="enrollment">Enrollments</option>
              <option value="verification">Verifications</option>
              <option value="approval">Approvals</option>
              <option value="rejection">Rejections</option>
              <option value="connection">Connections</option>
              <option value="disconnection">Disconnections</option>
              <option value="revocation">Revocations</option>
            </select>
            <button
              onClick={() => setShowAuditModal(true)}
              className="inline-flex items-center px-3 py-2 border border-gray-300 dark:border-gray-600 text-sm font-medium rounded-md text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              View Full Log
            </button>
          </div>
        </div>

        {/* Recent audit events preview */}
        <div className="space-y-2">
          {auditEvents.slice(0, 5).map((event) => (
            <div key={event.id} className="flex items-center justify-between py-2 px-3 bg-gray-50 dark:bg-gray-800 rounded">
              <div className="flex items-center space-x-3">
                <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                  event.event_type === 'approval' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' :
                  event.event_type === 'rejection' ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200' :
                  event.event_type === 'enrollment' ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200' :
                  'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200'
                }`}>
                  {event.event_type}
                </span>
                <span className="text-sm text-gray-900 dark:text-white">
                  {event.vma_fingerprint?.substring(0, 16)}...
                </span>
                {event.approved_by && (
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    by {event.approved_by}
                  </span>
                )}
              </div>
              <span className="text-xs text-gray-500 dark:text-gray-400">
                {formatTimeAgo(event.created_at)}
              </span>
            </div>
          ))}
          
          {auditEvents.length === 0 && (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">
              No audit events available
            </div>
          )}
        </div>
      </div>

      {/* Generate Pairing Code Modal */}
      {showGenerateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-900 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              Generate VMA Pairing Code
            </h3>
            <div className="space-y-4">
              <p className="text-gray-600 dark:text-gray-400">
                Generate a secure pairing code for VMA enrollment. The code will be valid for 10 minutes and can only be used once.
              </p>
              <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                <div className="flex items-center">
                  <HiExclamation className="h-4 w-4 text-blue-500 mr-2" />
                  <span className="text-blue-800 dark:text-blue-200 text-sm">
                    Share this code securely with the VMA operator.
                  </span>
                </div>
              </div>
            </div>
            <div className="flex justify-end space-x-3 mt-6">
              <button
                onClick={() => setShowGenerateModal(false)}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={generatePairingCode}
                disabled={loading}
                className="inline-flex items-center px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
              >
                {loading && <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>}
                Generate Code
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Approval Modal */}
      {showApprovalModal && selectedEnrollment && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-900 rounded-lg p-6 max-w-lg w-full mx-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              Approve VMA Enrollment
            </h3>
            
            <div className="space-y-4">
              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
                <h4 className="font-medium text-gray-900 dark:text-white mb-3">VMA Details</h4>
                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">Name:</span>
                    <div className="font-medium">{selectedEnrollment.vma_name || 'Unnamed VMA'}</div>
                  </div>
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">Version:</span>
                    <div className="font-medium">{selectedEnrollment.vma_version || 'Unknown'}</div>
                  </div>
                  <div className="col-span-2">
                    <span className="text-gray-500 dark:text-gray-400">SSH Fingerprint:</span>
                    <code className="block mt-1 text-xs bg-gray-100 dark:bg-gray-700 p-2 rounded font-mono">
                      {selectedEnrollment.vma_fingerprint}
                    </code>
                  </div>
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">Source IP:</span>
                    <div className="font-medium">{selectedEnrollment.vma_ip_address || 'Unknown'}</div>
                  </div>
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">Requested:</span>
                    <div className="font-medium">{formatTimeAgo(selectedEnrollment.created_at)}</div>
                  </div>
                </div>
              </div>
              
              <div>
                <label htmlFor="approval-notes" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Approval Notes (Optional)
                </label>
                <input
                  id="approval-notes"
                  type="text"
                  placeholder="e.g., Approved for production migration project"
                  value={approvalNotes}
                  onChange={(e) => setApprovalNotes(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-800 dark:text-white"
                />
              </div>
            </div>
            
            <div className="flex justify-end space-x-3 mt-6">
              <button
                onClick={() => setShowApprovalModal(false)}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={() => approveEnrollment(selectedEnrollment.id)}
                disabled={loading}
                className="inline-flex items-center px-4 py-2 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50"
              >
                {loading && <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>}
                <HiCheck className="h-4 w-4 mr-2" />
                Approve VMA
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Audit Log Modal */}
      {showAuditModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-900 rounded-lg p-6 max-w-4xl w-full mx-4 max-h-[80vh] overflow-hidden">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                VMA Security Audit Log
              </h3>
              <button
                onClick={() => setShowAuditModal(false)}
                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              >
                <HiX className="h-5 w-5" />
              </button>
            </div>
            
            {/* Audit filter controls */}
            <div className="flex items-center space-x-4 mb-4 pb-4 border-b border-gray-200 dark:border-gray-700">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Event Type
                </label>
                <select
                  value={auditFilter.event_type}
                  onChange={(e) => setAuditFilter(prev => ({ ...prev, event_type: e.target.value }))}
                  className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                >
                  <option value="">All Events</option>
                  <option value="enrollment">Enrollments</option>
                  <option value="verification">Verifications</option>
                  <option value="approval">Approvals</option>
                  <option value="rejection">Rejections</option>
                  <option value="connection">Connections</option>
                  <option value="disconnection">Disconnections</option>
                  <option value="revocation">Revocations</option>
                </select>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Limit
                </label>
                <select
                  value={auditFilter.limit}
                  onChange={(e) => setAuditFilter(prev => ({ ...prev, limit: parseInt(e.target.value) }))}
                  className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                >
                  <option value="25">25 events</option>
                  <option value="50">50 events</option>
                  <option value="100">100 events</option>
                  <option value="200">200 events</option>
                </select>
              </div>
              
              <button
                onClick={loadAuditLog}
                className="mt-6 inline-flex items-center px-3 py-2 border border-gray-300 dark:border-gray-600 text-sm font-medium rounded-md text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                <HiRefresh className="h-4 w-4 mr-2" />
                Refresh
              </button>
            </div>

            {/* Audit events table */}
            <div className="overflow-y-auto max-h-96">
              {auditEvents.length === 0 ? (
                <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                  No audit events found
                </div>
              ) : (
                <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                  <thead className="bg-gray-50 dark:bg-gray-800 sticky top-0">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Event
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        VMA Fingerprint
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Source IP
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Admin
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Time
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                    {auditEvents.map((event) => (
                      <tr key={event.id} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                        <td className="px-4 py-3">
                          <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                            event.event_type === 'approval' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' :
                            event.event_type === 'rejection' ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200' :
                            event.event_type === 'enrollment' ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200' :
                            event.event_type === 'revocation' ? 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200' :
                            'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200'
                          }`}>
                            {event.event_type}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <code className="text-xs bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded font-mono">
                            {event.vma_fingerprint?.substring(0, 20)}...
                          </code>
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-white">
                          {event.source_ip || '-'}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-white">
                          {event.approved_by || '-'}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                          {formatTimeAgo(event.created_at)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
            
            <div className="flex justify-end mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <button
                onClick={() => setShowAuditModal(false)}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Revocation Confirmation Modal */}
      {showRevocationModal && selectedVMAForRevocation && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-900 rounded-lg p-6 max-w-md w-full mx-4">
            <div className="flex items-center mb-4">
              <div className="flex-shrink-0">
                <HiExclamation className="h-6 w-6 text-red-600" />
              </div>
              <div className="ml-3">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Revoke VMA Access
                </h3>
              </div>
            </div>
            
            <div className="space-y-4">
              <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                <h4 className="font-medium text-red-900 dark:text-red-100 mb-2">
                  ⚠️ This action cannot be undone
                </h4>
                <p className="text-red-700 dark:text-red-300 text-sm">
                  Revoking access will immediately terminate the VMA's SSH tunnel connection and remove its authentication credentials.
                </p>
              </div>

              <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
                <h4 className="font-medium text-gray-900 dark:text-white mb-3">VMA Details</h4>
                <div className="space-y-2 text-sm">
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">Name:</span>
                    <span className="ml-2 font-medium">{selectedVMAForRevocation.vma_name}</span>
                  </div>
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">SSH Fingerprint:</span>
                    <code className="ml-2 text-xs bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded font-mono">
                      {selectedVMAForRevocation.vma_fingerprint.substring(0, 24)}...
                    </code>
                  </div>
                  <div>
                    <span className="text-gray-500 dark:text-gray-400">Connected:</span>
                    <span className="ml-2">{formatTimeAgo(selectedVMAForRevocation.connected_at)}</span>
                  </div>
                </div>
              </div>
              
              <div>
                <label htmlFor="revocation-reason" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Revocation Reason
                </label>
                <input
                  id="revocation-reason"
                  type="text"
                  placeholder="e.g., VMA decommissioned, security policy change"
                  value={revocationReason}
                  onChange={(e) => setRevocationReason(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-red-500 focus:border-red-500 dark:bg-gray-800 dark:text-white"
                />
              </div>
            </div>
            
            <div className="flex justify-end space-x-3 mt-6">
              <button
                onClick={() => {
                  setShowRevocationModal(false);
                  setRevocationReason('');
                  setSelectedVMAForRevocation(null);
                }}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  revokeVMAAccess(selectedVMAForRevocation.enrollment_id);
                  setShowRevocationModal(false);
                  setRevocationReason('');
                  setSelectedVMAForRevocation(null);
                }}
                disabled={loading}
                className="inline-flex items-center px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
              >
                {loading && <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>}
                <HiTrash className="h-4 w-4 mr-2" />
                Revoke Access
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
});
