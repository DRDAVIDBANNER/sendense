'use client';

import { useState, useEffect } from 'react';
import { Button, Card, Modal, TextInput, Label, Alert, Spinner } from 'flowbite-react';
import { HiPlus, HiCheck, HiX, HiTrash, HiRefresh, HiExclamation } from 'react-icons/hi';

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
}

interface ActiveVMA {
  id: string;
  vma_name: string;
  vma_fingerprint: string;
  connection_status: string;
  connected_at: string;
  last_seen_at?: string;
}

export default function VMAEnrollmentSettings() {
  const [pairingCode, setPairingCode] = useState<PairingCode | null>(null);
  const [pendingEnrollments, setPendingEnrollments] = useState<PendingEnrollment[]>([]);
  const [activeVMAs, setActiveVMAs] = useState<ActiveVMA[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  
  // Modal states
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [showApprovalModal, setShowApprovalModal] = useState(false);
  const [selectedEnrollment, setSelectedEnrollment] = useState<PendingEnrollment | null>(null);
  const [approvalNotes, setApprovalNotes] = useState('');

  const API_BASE = process.env.NODE_ENV === 'production' ? '' : 'http://localhost:8082';

  // Load data on component mount (disabled until API is deployed)
  useEffect(() => {
    // loadPendingEnrollments();
    // loadActiveVMAs();
    // TODO: Enable when VMA enrollment API is deployed
  }, []);

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
        throw new Error('VMA enrollment API not available yet');
      }

      const data = await response.json();
      setPairingCode(data);
      setSuccess('Pairing code generated successfully');
      setShowGenerateModal(false);
    } catch (err) {
      // For now, show a placeholder since the API isn't deployed yet
      setError('VMA enrollment system not deployed yet - this is Phase 1 UI preview');
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
      } else {
        // API endpoint not implemented yet - show placeholder
        setPendingEnrollments([]);
      }
    } catch (err) {
      console.error('VMA enrollment API not available yet:', err);
      setPendingEnrollments([]);
    }
  };

  const loadActiveVMAs = async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/admin/vma/active`);
      if (response.ok) {
        const data = await response.json();
        setActiveVMAs(data.connections || []);
      } else {
        // API endpoint not implemented yet - show placeholder
        setActiveVMAs([]);
      }
    } catch (err) {
      console.error('VMA enrollment API not available yet:', err);
      setActiveVMAs([]);
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
        throw new Error('Failed to approve enrollment');
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
        throw new Error('Failed to reject enrollment');
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
        throw new Error('Failed to revoke VMA access');
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

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setSuccess('Copied to clipboard');
  };

  return (
    <div className="space-y-6">
      {/* Development Notice */}
      <Alert color="info">
        <HiExclamation className="h-4 w-4" />
        <span className="font-medium">VMA Enrollment System - Phase 1 UI Preview</span>
        <div className="mt-2 text-sm">
          The VMA enrollment backend is currently in development. This interface shows the planned admin workflow for secure VMA-OMA pairing with operator approval.
        </div>
      </Alert>

      {/* Alerts */}
      {error && (
        <Alert color="failure" onDismiss={() => setError(null)}>
          <HiExclamation className="h-4 w-4" />
          {error}
        </Alert>
      )}
      {success && (
        <Alert color="success" onDismiss={() => setSuccess(null)}>
          <HiCheck className="h-4 w-4" />
          {success}
        </Alert>
      )}

      {/* Pairing Code Generation */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              VMA Pairing Code
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Generate secure pairing codes for VMA enrollment
            </p>
          </div>
          <Button
            onClick={() => setShowGenerateModal(true)}
            disabled={loading}
            size="sm"
          >
            <HiPlus className="h-4 w-4 mr-2" />
            Generate Code
          </Button>
        </div>

        {pairingCode && (
          <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-2xl font-mono font-bold text-blue-600 dark:text-blue-400">
                  {pairingCode.pairing_code}
                </div>
                <div className="text-sm text-gray-600 dark:text-gray-400">
                  Expires: {new Date(pairingCode.expires_at).toLocaleString()}
                </div>
              </div>
              <Button
                size="sm"
                onClick={() => copyToClipboard(pairingCode.pairing_code)}
              >
                Copy Code
              </Button>
            </div>
          </div>
        )}
      </Card>

      {/* Pending Enrollments */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Pending VMA Enrollments
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              VMAs awaiting approval for connection
            </p>
          </div>
          <Button
            onClick={loadPendingEnrollments}
            disabled={loading}
            size="sm"
            color="gray"
          >
            <HiRefresh className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>

        {pendingEnrollments.length === 0 ? (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            No pending VMA enrollments
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
                    Version
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    SSH Fingerprint
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Source IP
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
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                      {enrollment.vma_name || 'Unnamed VMA'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                        {enrollment.vma_version || 'Unknown'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <code className="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                        {enrollment.vma_fingerprint?.substring(0, 20)}...
                      </code>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                      {enrollment.vma_ip_address || 'Unknown'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                      {formatTimeAgo(enrollment.created_at)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
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
      </Card>

      {/* Active VMA Connections */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Active VMA Connections
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Currently connected VMAs
            </p>
          </div>
          <Button
            onClick={loadActiveVMAs}
            disabled={loading}
            size="sm"
            color="gray"
          >
            <HiRefresh className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>

        {activeVMAs.length === 0 ? (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            No active VMA connections
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
                    Status
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
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                      {vma.vma_name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        vma.connection_status === 'connected' 
                          ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                          : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                      }`}>
                        {vma.connection_status}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <code className="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                        {vma.vma_fingerprint.substring(0, 20)}...
                      </code>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                      {formatTimeAgo(vma.connected_at)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                      {vma.last_seen_at ? formatTimeAgo(vma.last_seen_at) : 'Never'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <button
                        onClick={() => revokeVMAAccess(vma.id)}
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
      </Card>

      {/* Generate Pairing Code Modal */}
      <Modal show={showGenerateModal} onClose={() => setShowGenerateModal(false)}>
        <Modal.Header>Generate VMA Pairing Code</Modal.Header>
        <Modal.Body>
          <div className="space-y-4">
            <p className="text-gray-600 dark:text-gray-400">
              Generate a secure pairing code for VMA enrollment. The code will be valid for 10 minutes.
            </p>
            <Alert color="info">
              <HiExclamation className="h-4 w-4" />
              Share this code securely with the VMA operator. It can only be used once.
            </Alert>
          </div>
        </Modal.Body>
        <Modal.Footer>
          <Button onClick={generatePairingCode} disabled={loading}>
            {loading && <Spinner size="sm" className="mr-2" />}
            Generate Code
          </Button>
          <Button color="gray" onClick={() => setShowGenerateModal(false)}>
            Cancel
          </Button>
        </Modal.Footer>
      </Modal>

      {/* Approval Modal */}
      <Modal show={showApprovalModal} onClose={() => setShowApprovalModal(false)}>
        <Modal.Header>Approve VMA Enrollment</Modal.Header>
        <Modal.Body>
          {selectedEnrollment && (
            <div className="space-y-4">
              <div>
                <h4 className="font-semibold text-gray-900 dark:text-white mb-2">
                  VMA Details
                </h4>
                <div className="bg-gray-50 dark:bg-gray-800 p-3 rounded">
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div><strong>Name:</strong> {selectedEnrollment.vma_name || 'Unnamed'}</div>
                    <div><strong>Version:</strong> {selectedEnrollment.vma_version || 'Unknown'}</div>
                    <div className="col-span-2">
                      <strong>SSH Fingerprint:</strong>
                      <code className="block mt-1 text-xs bg-gray-100 dark:bg-gray-700 p-1 rounded">
                        {selectedEnrollment.vma_fingerprint}
                      </code>
                    </div>
                    <div><strong>Source IP:</strong> {selectedEnrollment.vma_ip_address}</div>
                    <div><strong>Requested:</strong> {formatTimeAgo(selectedEnrollment.created_at)}</div>
                  </div>
                </div>
              </div>
              
              <div>
                <Label htmlFor="approval-notes" value="Approval Notes (Optional)" />
                <TextInput
                  id="approval-notes"
                  type="text"
                  placeholder="e.g., Approved for production migration project"
                  value={approvalNotes}
                  onChange={(e) => setApprovalNotes(e.target.value)}
                />
              </div>
            </div>
          )}
        </Modal.Body>
        <Modal.Footer>
          <Button 
            onClick={() => selectedEnrollment && approveEnrollment(selectedEnrollment.id)} 
            disabled={loading}
          >
            {loading && <Spinner size="sm" className="mr-2" />}
            <HiCheck className="h-4 w-4 mr-2" />
            Approve VMA
          </Button>
          <Button color="gray" onClick={() => setShowApprovalModal(false)}>
            Cancel
          </Button>
        </Modal.Footer>
      </Modal>
    </div>
  );
}
