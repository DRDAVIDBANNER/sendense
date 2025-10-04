'use client';

import { useState, useEffect } from 'react';
import { Button, Card, Badge, Table, Modal, TextInput, Label, Alert, Spinner } from 'flowbite-react';
import { HiPlus, HiPencil, HiTrash, HiRefresh, HiCheck, HiExclamation } from 'react-icons/hi';

interface VMwareCredential {
  id: number;
  name: string;
  vcenter_host: string;
  username: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export default function VMwareCredentialsSettings() {
  const [credentials, setCredentials] = useState<VMwareCredential[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const API_BASE = process.env.NODE_ENV === 'production' ? '' : 'http://localhost:8082';

  // Load credentials on component mount
  useEffect(() => {
    loadCredentials();
  }, []);

  const loadCredentials = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE}/api/v1/vmware-credentials`);
      if (response.ok) {
        const data = await response.json();
        setCredentials(data.credentials || []);
      } else {
        throw new Error('Failed to load VMware credentials');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load credentials');
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

  return (
    <div className="space-y-6">
      {/* Development Notice */}
      <Alert color="info">
        <HiExclamation className="h-4 w-4" />
        <span className="font-medium">VMware Credentials Management</span>
        <div className="mt-2 text-sm">
          VMware credentials are managed through the existing VMware Credentials Manager component. This placeholder will be replaced with the full interface.
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

      {/* VMware Credentials */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              VMware Credentials
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Manage vCenter connection credentials
            </p>
          </div>
          <div className="flex space-x-2">
            <Button
              onClick={loadCredentials}
              disabled={loading}
              size="sm"
              color="gray"
            >
              <HiRefresh className="h-4 w-4 mr-2" />
              Refresh
            </Button>
            <Button
              onClick={() => {/* TODO: Add create modal */}}
              disabled={loading}
              size="sm"
            >
              <HiPlus className="h-4 w-4 mr-2" />
              Add Credentials
            </Button>
          </div>
        </div>

        {loading ? (
          <div className="flex justify-center py-8">
            <Spinner size="lg" />
          </div>
        ) : credentials.length === 0 ? (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            No VMware credentials configured
          </div>
        ) : (
          <Table>
            <Table.Head>
              <Table.HeadCell>Name</Table.HeadCell>
              <Table.HeadCell>vCenter Host</Table.HeadCell>
              <Table.HeadCell>Username</Table.HeadCell>
              <Table.HeadCell>Status</Table.HeadCell>
              <Table.HeadCell>Updated</Table.HeadCell>
              <Table.HeadCell>Actions</Table.HeadCell>
            </Table.Head>
            <Table.Body>
              {credentials.map((cred) => (
                <Table.Row key={cred.id}>
                  <Table.Cell className="font-medium">
                    {cred.name}
                    {cred.is_default && (
                      <Badge color="success" size="sm" className="ml-2">
                        Default
                      </Badge>
                    )}
                  </Table.Cell>
                  <Table.Cell>{cred.vcenter_host}</Table.Cell>
                  <Table.Cell>{cred.username}</Table.Cell>
                  <Table.Cell>
                    <Badge color="success" size="sm">
                      Active
                    </Badge>
                  </Table.Cell>
                  <Table.Cell>{formatTimeAgo(cred.updated_at)}</Table.Cell>
                  <Table.Cell>
                    <div className="flex space-x-2">
                      <Button
                        size="xs"
                        color="gray"
                        onClick={() => {/* TODO: Add edit modal */}}
                      >
                        <HiPencil className="h-3 w-3 mr-1" />
                        Edit
                      </Button>
                      <Button
                        size="xs"
                        color="failure"
                        onClick={() => {/* TODO: Add delete confirmation */}}
                      >
                        <HiTrash className="h-3 w-3 mr-1" />
                        Delete
                      </Button>
                    </div>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
        )}
      </Card>
    </div>
  );
}
