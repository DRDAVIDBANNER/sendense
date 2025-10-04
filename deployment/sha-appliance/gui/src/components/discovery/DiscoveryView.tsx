'use client';

import React, { useState, useCallback } from 'react';
import { Card, Button, Badge, Spinner, Alert, TextInput, Label } from 'flowbite-react';
import { HiRefresh, HiServer, HiCloud, HiDatabase, HiPlus } from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';

export interface DiscoveryViewProps {
  onVMSelect: (vmName: string) => void;
}

interface VMData {
  id: string;
  name: string;
  power_state: string;
  guest_os: string;
  num_cpu: number;
  memory_mb: number;
  disks: Array<{
    label: string;
    size_gb: number;
    provisioning_type: string;
    datastore: string;
  }>;
  networks: Array<{
    name: string;
    vlan_id?: string;
  }>;
  path: string;
  datacenter: string;
}

interface VMwareCredential {
  id: number;
  credential_name: string;
  vcenter_host: string;
  username: string;
  datacenter: string;
  is_default: boolean;
}

export const DiscoveryView = React.memo(({ onVMSelect }: DiscoveryViewProps) => {
  const [vms, setVms] = useState<VMData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState<string | null>(null);
  const [filter, setFilter] = useState('');
  
  // Enhanced Discovery state
  const [credentials, setCredentials] = useState<VMwareCredential[]>([]);
  const [selectedCredentialId, setSelectedCredentialId] = useState<number | null>(null);
  const [osseaConfigId] = useState(1); // Default to production OSSEA
  const [loadingCredentials, setLoadingCredentials] = useState(true);

  // Load VMware credentials on mount
  React.useEffect(() => {
    async function loadCredentials() {
      try {
        setLoadingCredentials(true);
        const response = await fetch('/api/v1/vmware-credentials');
        
        if (!response.ok) throw new Error('Failed to load credentials');
        
        const data = await response.json();
        setCredentials(data.credentials || []);
        
        // Auto-select default credential
        const defaultCred = (data.credentials || []).find((c: VMwareCredential) => c.is_default);
        if (defaultCred) {
          setSelectedCredentialId(defaultCred.id);
        } else if (data.credentials && data.credentials.length > 0) {
          setSelectedCredentialId(data.credentials[0].id);
        }
      } catch (err) {
        console.error('Failed to load credentials:', err);
        setError('Failed to load VMware credentials');
      } finally {
        setLoadingCredentials(false);
      }
    }
    loadCredentials();
  }, []);

  const discoverVMs = useCallback(async () => {
    if (!selectedCredentialId) {
      setError('Please select VMware credentials first');
      return;
    }

    setLoading(true);
    setError('');
    
    try {
      // Use Enhanced Discovery API via Next.js proxy
      const response = await fetch('/api/discovery/discover-vms', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          credential_id: selectedCredentialId,
          filter: filter || undefined,
          create_context: false  // Just discover, don't add to management yet
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Discovery failed');
      }

      const data = await response.json();
      setVms(data.discovered_vms || []);
    } catch (err) {
      console.error('Discovery error:', err);
      setError(err instanceof Error ? err.message : 'Failed to discover VMs');
      setVms([]);
    } finally {
      setLoading(false);
    }
  }, [selectedCredentialId, filter]);

  const addToManagement = useCallback(async (vm: VMData) => {
    if (!selectedCredentialId) {
      setError('Please select VMware credentials first');
      return;
    }

    // Show loading state
    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      // Use Enhanced Discovery add-vms endpoint via Next.js proxy
      const response = await fetch('/api/discovery/add-vms', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          credential_id: selectedCredentialId,
          vm_names: [vm.name],
          added_by: 'discovery-gui'
        })
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to add VM to management');
      }

      const result = await response.json();
      console.log('VM added to management successfully:', result);
      
      // Show success message
      setSuccess(`✅ ${vm.name} added to management successfully! Redirecting in 3 seconds...`);
      
      // Wait longer for user to see success message, then navigate
      setTimeout(() => {
        onVMSelect(vm.name);
      }, 3000);
    } catch (err) {
      console.error('Add to management error:', err);
      setError(err instanceof Error ? err.message : 'Failed to add VM to management');
    } finally {
      setLoading(false);
    }
  }, [selectedCredentialId, onVMSelect]);

  const formatBytes = useCallback((bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }, []);

  return (
    <div className="p-4 space-y-4">
      {/* Compact Header with Inline Stats */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            🔍 VM Discovery
          </h1>
        </div>
        <div className="flex items-center space-x-6 text-sm">
          <div className="flex items-center space-x-2">
            <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
            <span className="text-gray-600 dark:text-gray-400">VMs: {vms.length}</span>
          </div>
          <div className="flex items-center space-x-2">
            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
            <span className="text-gray-600 dark:text-gray-400">Running: {vms.filter(vm => vm.power_state === 'poweredOn').length}</span>
          </div>
          <div className="flex items-center space-x-2">
            <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
            <span className="text-gray-600 dark:text-gray-400">Storage: {formatBytes(vms.reduce((total, vm) => 
              total + vm.disks.reduce((diskTotal, disk) => 
                diskTotal + (disk.size_gb * 1024 * 1024 * 1024), 0), 0)
            )}</span>
          </div>
        </div>
      </div>

      {/* Compact Discovery Configuration */}
      <Card>
        <div className="flex items-center space-x-3">
          <div className="flex-1">
            <select
              className="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2 text-sm text-gray-900 focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:border-blue-500 dark:focus:ring-blue-500"
              value={selectedCredentialId || ''}
              onChange={(e) => setSelectedCredentialId(parseInt(e.target.value))}
              disabled={loadingCredentials}
            >
              <option value="">Select VMware Credentials...</option>
              {credentials.map(cred => (
                <option key={cred.id} value={cred.id}>
                  {cred.credential_name} ({cred.vcenter_host}){cred.is_default ? ' [Default]' : ''}
                </option>
              ))}
            </select>
          </div>
          <div className="flex-1">
            <TextInput
              size="sm"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="VM Filter (optional)"
            />
          </div>
          <Button 
            onClick={discoverVMs} 
            disabled={loading || !selectedCredentialId || loadingCredentials}
            color="blue"
            size="sm"
            className="flex-shrink-0"
          >
            <ClientIcon className="mr-1 h-4 w-4">
              <HiRefresh />
            </ClientIcon>
            {loadingCredentials ? 'Loading...' : loading ? 'Discovering...' : 'Discover'}
          </Button>
        </div>
      </Card>

      {/* Error Display */}
      {error && (
        <Alert color="failure" onDismiss={() => setError('')}>
          <span className="text-sm">{error}</span>
        </Alert>
      )}

      {/* Success Display */}
      {success && (
        <Alert color="success" onDismiss={() => setSuccess(null)}>
          <span className="text-sm">{success}</span>
        </Alert>
      )}

      {/* Loading State */}
      {loading && (
        <div className="flex items-center justify-center py-4">
          <Spinner size="md" />
          <span className="ml-2 text-sm text-gray-600 dark:text-gray-400">
            Discovering VMs...
          </span>
        </div>
      )}

      {/* Compact VM Table */}
      {vms.length > 0 && !loading && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <div className="overflow-auto max-h-[calc(100vh-200px)]">
            <table className="w-full text-xs text-left text-gray-500 dark:text-gray-400">
              <thead className="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400 sticky top-0">
                <tr>
                  <th scope="col" className="px-3 py-2 font-medium">VM Name</th>
                  <th scope="col" className="px-3 py-2 font-medium">State</th>
                  <th scope="col" className="px-3 py-2 font-medium">OS</th>
                  <th scope="col" className="px-3 py-2 font-medium">Resources</th>
                  <th scope="col" className="px-3 py-2 font-medium">Storage</th>
                  <th scope="col" className="px-3 py-2 font-medium">Networks</th>
                  <th scope="col" className="px-3 py-2 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {vms.map((vm) => (
                  <tr key={vm.id} className="bg-white border-b dark:bg-gray-800 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-3 py-2 font-medium text-gray-900 dark:text-white">
                      {vm.name}
                    </td>
                    <td className="px-3 py-2">
                      <div className="flex items-center space-x-1">
                        <div className={`w-2 h-2 rounded-full ${vm.power_state === 'poweredOn' ? 'bg-green-500' : 'bg-gray-400'}`}></div>
                        <span className="text-xs">{vm.power_state === 'poweredOn' ? 'On' : 'Off'}</span>
                      </div>
                    </td>
                    <td className="px-3 py-2 text-xs">
                      {vm.guest_os ? vm.guest_os.substring(0, 12) + (vm.guest_os.length > 12 ? '...' : '') : 'Unknown'}
                    </td>
                    <td className="px-3 py-2 text-xs">
                      <div>
                        <div>{vm.num_cpu} vCPU</div>
                        <div className="text-gray-500">{Math.round(vm.memory_mb / 1024)} GB RAM</div>
                      </div>
                    </td>
                    <td className="px-3 py-2 text-xs">
                      <div>
                        <div>{vm.disks.length} disk{vm.disks.length > 1 ? 's' : ''}</div>
                        <div className="text-gray-500">
                          {vm.disks.reduce((total, disk) => total + disk.size_gb, 0)} GB
                        </div>
                      </div>
                    </td>
                    <td className="px-3 py-2 text-xs">
                      <div>
                        <div>{vm.networks.length} network{vm.networks.length > 1 ? 's' : ''}</div>
                        {vm.networks.length > 0 && (
                          <div className="text-gray-500 truncate max-w-[80px]">
                            {vm.networks[0].name}
                            {vm.networks.length > 1 && ` +${vm.networks.length - 1}`}
                          </div>
                        )}
                      </div>
                    </td>
                    <td className="px-3 py-2">
                      <div className="flex space-x-1">
                        <Button 
                          size="xs" 
                          color="green"
                          onClick={() => addToManagement(vm)}
                          disabled={loading}
                          className="text-xs px-2 py-1"
                        >
                          {loading ? (
                            <>
                              <svg className="animate-spin h-3 w-3 mr-1" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                              </svg>
                              Adding...
                            </>
                          ) : (
                            <>
                              <ClientIcon className="mr-1 h-3 w-3">
                                <HiPlus />
                              </ClientIcon>
                              Add to Management
                            </>
                          )}
                        </Button>
                        <Button 
                          size="xs" 
                          color="gray"
                          onClick={() => onVMSelect(vm.name)}
                          className="text-xs px-2 py-1"
                        >
                          View
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Compact Empty State */}
      {!loading && vms.length === 0 && !error && (
        <div className="text-center py-8">
          <ClientIcon className="mx-auto h-8 w-8 text-gray-400 mb-3">
            <HiServer />
          </ClientIcon>
          <h3 className="text-sm font-medium text-gray-900 dark:text-white mb-2">
            No VMs Found
          </h3>
          <p className="text-xs text-gray-600 dark:text-gray-400 mb-3">
            Configure vCenter settings above and click "Discover" to find VMs.
          </p>
        </div>
      )}
    </div>
  );
});

DiscoveryView.displayName = 'DiscoveryView';










