'use client';

import React, { useState, useEffect } from 'react';
import { HiPencil, HiTrash, HiPlus, HiKey } from 'react-icons/hi';

interface VMwareCredential {
  id: number;
  credential_name: string;
  vcenter_host: string;
  username: string;
  datacenter: string;
  is_active: boolean;
  is_default: boolean;
  created_at: string;
  updated_at: string;
  last_used?: string;
  usage_count: number;
}

interface CreateCredentialForm {
  credential_name: string;
  vcenter_host: string;
  username: string;
  password: string;
  datacenter: string;
  is_active: boolean;
  is_default: boolean;
}

export const VMwareCredentialsManager = React.memo(() => {
  const [credentials, setCredentials] = useState<VMwareCredential[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [testing, setTesting] = useState<{[key: number]: boolean}>({});
  const [testResults, setTestResults] = useState<{[key: number]: {success: boolean, message: string}}>({});

  const [createForm, setCreateForm] = useState<CreateCredentialForm>({
    credential_name: '',
    vcenter_host: '',
    username: '',
    password: '',
    datacenter: '',
    is_active: true,
    is_default: false,
  });

  // Load credentials from API
  const loadCredentials = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/v1/vmware-credentials');
      if (!response.ok) throw new Error('Failed to load credentials');
      
      const data = await response.json();
      setCredentials(data.credentials || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load credentials');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCredentials();
  }, []);

  // Create new credential set
  const handleCreateCredential = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch('/api/v1/vmware-credentials', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(createForm),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to create credentials');
      }

      setShowCreateModal(false);
      setCreateForm({
        credential_name: '',
        vcenter_host: '',
        username: '',
        password: '',
        datacenter: '',
        is_active: true,
        is_default: false,
      });
      await loadCredentials();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create credentials');
    }
  };

  // Delete credential set
  const handleDeleteCredential = async (id: number) => {
    if (!confirm('Are you sure you want to delete this credential set?')) return;

    try {
      const response = await fetch(`/api/v1/vmware-credentials/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) throw new Error('Failed to delete credentials');
      await loadCredentials();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete credentials');
    }
  };

  // Set credential as default
  const handleSetDefault = async (id: number) => {
    try {
      const response = await fetch(`/api/v1/vmware-credentials/${id}/set-default`, {
        method: 'PUT',
      });

      if (!response.ok) throw new Error('Failed to set default credentials');
      await loadCredentials();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to set default credentials');
    }
  };

  // Test connectivity
  const handleTestConnectivity = async (id: number) => {
    try {
      setTesting({...testing, [id]: true});
      const response = await fetch(`/api/v1/vmware-credentials/${id}/test`, {
        method: 'POST',
      });

      const result = await response.json();
      setTestResults({
        ...testResults,
        [id]: {
          success: response.ok,
          message: result.message || (response.ok ? 'Connection successful' : 'Connection failed')
        }
      });
    } catch (err) {
      setTestResults({
        ...testResults,
        [id]: {
          success: false,
          message: 'Test failed: Network error'
        }
      });
    } finally {
      setTesting({...testing, [id]: false});
    }
  };

  // Format date for display
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  if (loading) {
    return (
      <div className="bg-white dark:bg-gray-800 shadow rounded-lg">
        <div className="text-center p-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto"></div>
          <p className="mt-2 text-gray-600 dark:text-gray-400">Loading VMware credentials...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">VMware Credentials</h2>
          <p className="text-gray-600 dark:text-gray-400">Manage vCenter authentication credentials</p>
        </div>
        <button 
          onClick={() => setShowCreateModal(true)} 
          className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-lg flex items-center transition-colors"
        >
          <HiPlus className="w-4 h-4 mr-2" />
          Add Credentials
        </button>
      </div>

      {/* Error Alert */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded relative" role="alert">
          <span className="block sm:inline">{error}</span>
          <button 
            className="absolute top-0 bottom-0 right-0 px-4 py-3"
            onClick={() => setError(null)}
          >
            <svg className="fill-current h-6 w-6 text-red-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20">
              <path d="M14.348 14.849a1.2 1.2 0 0 1-1.697 0L10 11.819l-2.651 3.029a1.2 1.2 0 1 1-1.697-1.697l2.758-3.15-2.759-3.152a1.2 1.2 0 1 1 1.697-1.697L10 8.183l2.651-3.031a1.2 1.2 0 1 1 1.697 1.697l-2.758 3.152 2.758 3.15a1.2 1.2 0 0 1 0 1.698z"/>
            </svg>
          </button>
        </div>
      )}

      {/* Credentials Table */}
      <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">vCenter Host</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Username</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Datacenter</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Last Used</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {credentials.map((cred) => (
                <tr key={cred.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                    <div className="flex items-center space-x-2">
                      <span>{cred.credential_name}</span>
                      {cred.is_default && (
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                          DEFAULT
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-300">{cred.vcenter_host}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-300">{cred.username}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-300">{cred.datacenter}</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      cred.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                    }`}>
                      {cred.is_active ? "Active" : "Inactive"}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-300">
                    {cred.last_used ? (
                      <div>
                        <div>{formatDate(cred.last_used)}</div>
                        <div className="text-xs text-gray-500">
                          Used {cred.usage_count} time{cred.usage_count !== 1 ? 's' : ''}
                        </div>
                      </div>
                    ) : (
                      <span className="text-gray-400">Never used</span>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    <div className="flex space-x-2">
                      <button
                        onClick={() => handleTestConnectivity(cred.id)}
                        disabled={testing[cred.id]}
                        className="text-indigo-600 hover:text-indigo-900 disabled:opacity-50"
                      >
                        {testing[cred.id] ? 'Testing...' : 'Test'}
                      </button>
                      {!cred.is_default && (
                        <button
                          onClick={() => handleSetDefault(cred.id)}
                          className="text-green-600 hover:text-green-900"
                        >
                          Set Default
                        </button>
                      )}
                      <button
                        onClick={() => handleDeleteCredential(cred.id)}
                        disabled={cred.is_default}
                        className="text-red-600 hover:text-red-900 disabled:opacity-50"
                      >
                        <HiTrash className="w-4 h-4" />
                      </button>
                    </div>
                    {/* Test Results */}
                    {testResults[cred.id] && (
                      <div className={`mt-1 text-xs ${testResults[cred.id].success ? 'text-green-600' : 'text-red-600'}`}>
                        {testResults[cred.id].message}
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Create Credential Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg max-w-lg w-full max-h-screen overflow-y-auto">
            <div className="p-6">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Add VMware Credentials</h3>
              <form onSubmit={handleCreateCredential} className="space-y-4">
                <div>
                  <label htmlFor="credential_name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    Credential Name
                  </label>
                  <input
                    id="credential_name"
                    type="text"
                    placeholder="e.g. Production vCenter"
                    value={createForm.credential_name}
                    onChange={(e) => setCreateForm({...createForm, credential_name: e.target.value})}
                    required
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  />
                </div>

                <div>
                  <label htmlFor="vcenter_host" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    vCenter Host
                  </label>
                  <input
                    id="vcenter_host"
                    type="text"
                    placeholder="vcenter.example.com"
                    value={createForm.vcenter_host}
                    onChange={(e) => setCreateForm({...createForm, vcenter_host: e.target.value})}
                    required
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  />
                </div>

                <div>
                  <label htmlFor="username" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    Username
                  </label>
                  <input
                    id="username"
                    type="text"
                    placeholder="administrator@vsphere.local"
                    value={createForm.username}
                    onChange={(e) => setCreateForm({...createForm, username: e.target.value})}
                    required
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  />
                </div>

                <div>
                  <label htmlFor="password" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    Password
                  </label>
                  <input
                    id="password"
                    type="password"
                    placeholder="••••••••"
                    value={createForm.password}
                    onChange={(e) => setCreateForm({...createForm, password: e.target.value})}
                    required
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  />
                </div>

                <div>
                  <label htmlFor="datacenter" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    Default Datacenter
                  </label>
                  <input
                    id="datacenter"
                    type="text"
                    placeholder="Datacenter1"
                    value={createForm.datacenter}
                    onChange={(e) => setCreateForm({...createForm, datacenter: e.target.value})}
                    required
                    className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  />
                </div>

                <div className="flex items-center">
                  <input
                    id="is_default"
                    type="checkbox"
                    checked={createForm.is_default}
                    onChange={(e) => setCreateForm({...createForm, is_default: e.target.checked})}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label htmlFor="is_default" className="ml-2 block text-sm text-gray-900 dark:text-gray-300">
                    Set as default credentials
                  </label>
                </div>

                <div className="flex justify-end space-x-3 pt-4">
                  <button
                    type="button"
                    onClick={() => setShowCreateModal(false)}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors"
                  >
                    Create Credentials
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* System Info Card */}
      <div className="bg-white dark:bg-gray-800 shadow rounded-lg">
        <div className="p-6">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
            <HiKey className="w-5 h-5 mr-2" />
            Security Information
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div>
              <span className="font-medium text-gray-700 dark:text-gray-300">Encryption:</span>
              <span className="ml-2 text-gray-600 dark:text-gray-400">AES-256-GCM</span>
            </div>
            <div>
              <span className="font-medium text-gray-700 dark:text-gray-300">Storage:</span>
              <span className="ml-2 text-gray-600 dark:text-gray-400">Database Encrypted</span>
            </div>
            <div>
              <span className="font-medium text-gray-700 dark:text-gray-300">Total Credentials:</span>
              <span className="ml-2 text-gray-600 dark:text-gray-400">{credentials.length}</span>
            </div>
            <div>
              <span className="font-medium text-gray-700 dark:text-gray-300">Active Credentials:</span>
              <span className="ml-2 text-gray-600 dark:text-gray-400">
                {credentials.filter(c => c.is_active).length}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
});

VMwareCredentialsManager.displayName = 'VMwareCredentialsManager';