'use client';

import React, { useState, useEffect } from 'react';
import { Card, Button, Badge, Alert, Spinner, Table } from 'flowbite-react';
import { HiOutlineGlobeAlt, HiOutlineCheck, HiOutlineExclamationCircle, HiOutlineCog, HiOutlineRefresh } from 'react-icons/hi';
import { LeftNavigation } from '@/components/layout/LeftNavigation';
import { useVMContexts } from '@/hooks/useVMContext';
import SimpleNetworkMappingModal from '@/components/network/SimpleNetworkMappingModal';

interface VMNetworkStatus {
  vm_name: string;
  context_id: string;
  status: 'ready' | 'configuring' | 'mapped' | 'error';
  vmware_networks: string[];
  ossea_networks: NetworkMapping[];
  has_mappings: boolean;
}

interface NetworkMapping {
  source_network_name: string;
  destination_network_name: string;
  destination_network_id: string;
  is_test_network: boolean;
  status: 'valid' | 'invalid' | 'pending';
}

interface OSSeaNetwork {
  id: string;
  name: string;
  zone_name: string;
  state: string;
  is_default: boolean;
}

export default function NetworkMappingPage() {
  const [vmNetworkStatus, setVmNetworkStatus] = useState<VMNetworkStatus[]>([]);
  const [availableNetworks, setAvailableNetworks] = useState<OSSeaNetwork[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedVM, setSelectedVM] = useState<VMNetworkStatus | null>(null);
  const [showConfigModal, setShowConfigModal] = useState(false);
  const [notification, setNotification] = useState<{type: 'success' | 'info' | 'warning'; message: string} | null>(null);
  
  const { data: vmContexts, isLoading: vmLoading, error: vmError } = useVMContexts();

  // Load VM network status and available OSSEA networks
  const loadNetworkData = async () => {
    if (!vmContexts) return;
    
    try {
      setLoading(true);
      setError(null);

      // Get available OSSEA networks
      const networksResponse = await fetch('/api/networks');
      const networksData = await networksResponse.json();
      
      if (networksData.success) {
        setAvailableNetworks(networksData.networks || []);
      }

      // Get VM network status for each VM
      const vmStatusPromises = vmContexts.map(async (context) => {
        try {
          // Discover real networks for this VM (Direct tunnel to VMA)
          let vmwareNetworks: string[] = [];
          
          // First, try to get existing mappings to see if we have real network names
          const existingMappingsResponse = await fetch(`/api/network-mappings?vm_id=${context.vm_name}`);
          let existingNetworkNames: string[] = [];
          if (existingMappingsResponse.ok) {
            const existingMappingsData = await existingMappingsResponse.json();
            if (existingMappingsData.success && existingMappingsData.mappings) {
              const networkNames = existingMappingsData.mappings.map((m: any) => m.source_network_name as string).filter(Boolean);
              existingNetworkNames = Array.from(new Set(networkNames));
            }
          }

          // Use existing mappings if available, otherwise VMA discovery, finally fallback
          if (existingNetworkNames.length > 0) {
            // Always use existing network names if we have them (real or synthetic)
            vmwareNetworks = existingNetworkNames;
          } else {
            // Try VMA discovery for VMs without any mappings
            try {
              const controller = new AbortController();
              const timeoutId = setTimeout(() => controller.abort(), 8000); // Shorter timeout for main page

              const discoveryResponse = await fetch('http://localhost:9081/api/v1/discover', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  vcenter: 'quad-vcenter-01.quadris.local',
                  username: 'administrator@vsphere.local',
                  password: 'EmyGVoBFesGQc47-',
                  datacenter: 'DatabanxDC',
                  filter: context.vm_name
                }),
                signal: controller.signal
              });

              clearTimeout(timeoutId);

              if (discoveryResponse.ok) {
                const discoveryData = await discoveryResponse.json();
                const discoveredVM = discoveryData.vms?.find((vm: any) => vm.name === context.vm_name);
                if (discoveredVM?.networks) {
                  vmwareNetworks = discoveredVM.networks
                    .map((net: any) => net.network_name)
                    .filter((name: string) => name && name.trim() !== '');
                }
              }
            } catch (error) {
              console.error(`VMA discovery failed for ${context.vm_name}:`, error);
            }

            // Fallback to synthetic network if all else fails
            if (vmwareNetworks.length === 0) {
              vmwareNetworks = [`${context.vm_name}-network`];
            }
          }

          // Get existing mappings for this VM
          const mappingsResponse = await fetch(`/api/network-mappings?vm_id=${context.vm_name}`);
          let existingMappings: NetworkMapping[] = [];
          if (mappingsResponse.ok) {
            const mappingsData = await mappingsResponse.json();
            // Fix: API returns 'mappings' not 'data'
            existingMappings = (mappingsData.mappings || []).map((mapping: any) => ({
              source_network_name: mapping.source_network_name,
              destination_network_name: mapping.destination_network_name,
              destination_network_id: mapping.destination_network_id || mapping.destination_network_name,
              is_test_network: mapping.is_test_network || false,
              status: 'valid' as const
            }));
          }

          // Calculate proper status based on dual mapping requirements
          const getNetworkMappingStatus = () => {
            if (vmwareNetworks.length === 0) return 'ready';
            
            const hasAllMappings = vmwareNetworks.every(vmwareNetwork => {
              const productionMapping = existingMappings.find(
                (m: any) => m.source_network_name === vmwareNetwork && !m.is_test_network
              );
              const testMapping = existingMappings.find(
                (m: any) => m.source_network_name === vmwareNetwork && m.is_test_network
              );
              return productionMapping && testMapping;
            });
            
            return hasAllMappings ? 'mapped' : (existingMappings.length > 0 ? 'configuring' : 'ready');
          };

          return {
            vm_name: context.vm_name,
            context_id: `ctx-${context.vm_name}-${Date.now()}`,
            status: getNetworkMappingStatus(),
            vmware_networks: vmwareNetworks,
            ossea_networks: existingMappings,
            has_mappings: existingMappings.length > 0
          } as VMNetworkStatus;
        } catch (error) {
          console.error(`Failed to load network data for ${context.vm_name}:`, error);
          return {
            vm_name: context.vm_name,
            context_id: `ctx-${context.vm_name}-error`,
            status: 'error',
            vmware_networks: [],
            ossea_networks: [],
            has_mappings: false
          } as VMNetworkStatus;
        }
      });

      const vmStatuses = await Promise.all(vmStatusPromises);
      setVmNetworkStatus(vmStatuses);
    } catch (error) {
      setError('Failed to load network configuration data');
      console.error('Network data loading error:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchVMDataForModal = async (vmName: string): Promise<VMNetworkStatus> => {
    // Find the VM context
    if (!vmContexts) {
      throw new Error('VM contexts not loaded');
    }
    
    const vmContext = vmContexts.find(ctx => ctx.vm_name === vmName);
    if (!vmContext) {
      throw new Error(`VM context not found for ${vmName}`);
    }

    // Get existing network mappings for this VM
    const mappingsResponse = await fetch(`/api/network-mappings?vm_id=${vmName}`);
    const mappingsData = await mappingsResponse.json();
    const existingMappings = mappingsData.success ? mappingsData.mappings : [];

    // Get VMware networks intelligently (prioritize existing mappings)
    let vmwareNetworks: string[] = [];
    
    // First, extract unique network names from existing mappings
    const networkNames = existingMappings.map((m: any) => m.source_network_name as string).filter(Boolean);
    const existingNetworkNames: string[] = Array.from(new Set(networkNames));
    
    // If we have existing mappings with real network names, use those
    if (existingNetworkNames.length > 0 && !existingNetworkNames.every(name => name.includes('-network'))) {
      vmwareNetworks = existingNetworkNames;
    } else {
      // Try VMA discovery as fallback
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 10000); // Shorter timeout

        const discoveryResponse = await fetch('http://localhost:9081/api/v1/discover', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            vcenter: 'quad-vcenter-01.quadris.local',
            username: 'administrator@vsphere.local', 
            password: 'EmyGVoBFesGQc47-',
            datacenter: 'DatabanxDC',
            filter: vmName
          }),
          signal: controller.signal
        });

        clearTimeout(timeoutId);

        if (discoveryResponse.ok) {
          const discoveryData = await discoveryResponse.json();
          if (discoveryData.success && discoveryData.vms && discoveryData.vms.length > 0) {
            const vm = discoveryData.vms[0];
            vmwareNetworks = vm.networks?.map((net: any) => net.network_name).filter(Boolean) || [];
          }
        }
      } catch (error) {
        console.error(`VMA discovery failed for ${vmName}:`, error);
      }

      // Fallback to synthetic network if all else fails
      if (vmwareNetworks.length === 0) {
        vmwareNetworks = [`${vmName}-network`];
      }
    }

    // Calculate proper status based on dual mapping requirements
    const getNetworkMappingStatus = () => {
      if (vmwareNetworks.length === 0) return 'ready';
      
      const hasAllMappings = vmwareNetworks.every(vmwareNetwork => {
        const productionMapping = existingMappings.find(
          (m: any) => m.source_network_name === vmwareNetwork && !m.is_test_network
        );
        const testMapping = existingMappings.find(
          (m: any) => m.source_network_name === vmwareNetwork && m.is_test_network
        );
        return productionMapping && testMapping;
      });
      
      return hasAllMappings ? 'mapped' : (existingMappings.length > 0 ? 'configuring' : 'ready');
    };

    return {
      vm_name: vmName,
      context_id: `ctx-${vmName}-${Date.now()}`,
      status: getNetworkMappingStatus(),
      vmware_networks: vmwareNetworks,
      ossea_networks: existingMappings,
      has_mappings: existingMappings.length > 0
    } as VMNetworkStatus;
  };

  const handleConfigureVM = async (vmName: string) => {
    try {
      setLoading(true);
      
      // Fetch fresh VM data directly
      const freshVmData = await fetchVMDataForModal(vmName);
      
      setSelectedVM(freshVmData);
      setShowConfigModal(true);
      
      // Also refresh the main data for consistency (but don't wait for it)
      loadNetworkData();
      
    } catch (error) {
      setError('Failed to load VM configuration data');
      console.error('Failed to configure VM:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleMappingSave = (mappings: NetworkMapping[]) => {
    if (selectedVM) {
    setNotification({
      type: 'success',
        message: `Successfully configured network mappings for ${selectedVM.vm_name}`
      });
      
      // Refresh the data to show updated status
      loadNetworkData();
    }
    setShowConfigModal(false);
    setSelectedVM(null);
  };

  const handleModalClose = () => {
    setShowConfigModal(false);
    setSelectedVM(null);
  };

  const handleRefresh = () => {
    loadNetworkData();
  };

  useEffect(() => {
    if (vmContexts && !vmLoading) {
      loadNetworkData();
    }
  }, [vmContexts, vmLoading]);

  useEffect(() => {
    if (notification) {
      const timer = setTimeout(() => setNotification(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [notification]);

  const getStatusColor = (status: VMNetworkStatus['status']) => {
    switch (status) {
      case 'mapped': return 'success';
      case 'ready': return 'warning';
      case 'configuring': return 'blue';
      case 'error': return 'failure';
      default: return 'gray';
    }
  };

  const getStatusText = (status: VMNetworkStatus['status']) => {
    switch (status) {
      case 'mapped': return 'Fully Configured';
      case 'ready': return 'Needs Setup';
      case 'configuring': return 'Partial Config';
      case 'error': return 'Error';
      default: return 'Unknown';
    }
  };

  if (vmLoading || loading) {
    return (
      <div className="flex min-h-screen bg-gray-50 dark:bg-gray-900">
        <div className="w-64 flex-shrink-0">
          <LeftNavigation 
            activeSection="network-mapping"
            onSectionChange={() => {}}
            collapsed={false}
            onToggle={() => {}}
          />
        </div>
        <main className="flex-1 overflow-auto">
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <Spinner size="xl" className="mb-4" />
              <p className="text-gray-500">Loading network configuration...</p>
            </div>
          </div>
        </main>
      </div>
    );
  }

  if (vmError || error) {
    return (
      <div className="flex min-h-screen bg-gray-50 dark:bg-gray-900">
        <div className="w-64 flex-shrink-0">
          <LeftNavigation 
            activeSection="network-mapping"
            onSectionChange={() => {}}
            collapsed={false}
            onToggle={() => {}}
          />
        </div>
        <main className="flex-1 overflow-auto p-6">
          <Alert color="failure">
            <div className="flex items-center justify-between">
              <span>Failed to load network data: {vmError?.message || error}</span>
              <Button size="sm" color="failure" onClick={handleRefresh}>
                <HiOutlineRefresh className="mr-2 h-4 w-4" />
                Retry
              </Button>
            </div>
          </Alert>
        </main>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen bg-gray-50 dark:bg-gray-900">
      <div className="w-64 flex-shrink-0">
        <LeftNavigation 
          activeSection="network-mapping"
          onSectionChange={() => {}}
          collapsed={false}
          onToggle={() => {}}
        />
      </div>
      <main className="flex-1 overflow-auto">
        <div className="space-y-6 p-6">
          {/* Simple, Clear Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center">
              <HiOutlineGlobeAlt className="mr-3 h-8 w-8 text-blue-500" />
                Network Configuration
            </h1>
            <p className="text-gray-500 dark:text-gray-400 mt-1">
                Map VMware networks to OSSEA networks for each virtual machine
            </p>
          </div>
          
          <div className="flex items-center space-x-3">
              <Button color="gray" size="sm" onClick={handleRefresh}>
                <HiOutlineRefresh className="mr-2 h-4 w-4" />
                Refresh
              </Button>
            <Badge color="blue" size="lg">
                {vmNetworkStatus.length} VMs
              </Badge>
          </div>
        </div>

        {/* Notification */}
        {notification && (
          <Alert color={notification.type} onDismiss={() => setNotification(null)}>
            {notification.message}
          </Alert>
        )}

          {/* Simple VM Network Status Table */}
        <Card>
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                VM Network Configuration Status
              </h2>
              <div className="text-sm text-gray-500">
                {vmNetworkStatus.filter(vm => vm.has_mappings).length} of {vmNetworkStatus.length} VMs configured
              </div>
            </div>

            {vmNetworkStatus.length === 0 ? (
              <div className="text-center py-12 text-gray-500">
                <HiOutlineGlobeAlt className="mx-auto h-16 w-16 mb-4 text-gray-300" />
                <h3 className="text-lg font-medium mb-2">No VMs Available</h3>
                <p>No virtual machines found for network configuration.</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <div className="min-w-full">
                  <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                    <thead className="bg-gray-50 dark:bg-gray-700">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Virtual Machine</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Status</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">VMware Networks</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">OSSEA Mappings</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                      {vmNetworkStatus.map((vm) => (
                        <tr key={vm.context_id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                          <td className="px-6 py-4 whitespace-nowrap font-medium text-gray-900 dark:text-white">
                            <div>
                              <div className="font-semibold">{vm.vm_name}</div>
                              <div className="text-xs text-gray-500">{vm.context_id}</div>
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <Badge color={getStatusColor(vm.status)} size="sm">
                              {getStatusText(vm.status)}
                            </Badge>
                          </td>
                          <td className="px-6 py-4">
                            <div className="space-y-1">
                              {vm.vmware_networks.length > 0 ? (
                                vm.vmware_networks.map((network, idx) => (
                                  <div key={idx} className="text-sm bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                                    {network}
                                  </div>
                                ))
                              ) : (
                                <span className="text-gray-500 text-sm italic">No networks discovered</span>
                              )}
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <div className="space-y-1">
                              {vm.ossea_networks.length > 0 ? (
                                vm.ossea_networks.map((mapping, idx) => (
                                  <div key={idx} className="text-sm">
            <div className="flex items-center space-x-2">
                                      <span className="text-blue-600 dark:text-blue-400">{mapping.destination_network_name}</span>
                                      {mapping.is_test_network && (
                                        <Badge color="purple" size="xs">Test</Badge>
              )}
            </div>
          </div>
                                ))
                              ) : (
                                <span className="text-orange-500 text-sm italic">Not configured</span>
                              )}
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <div className="flex items-center space-x-2">
                  <Button
                    size="xs"
                                color={vm.has_mappings ? "gray" : "blue"}
                                onClick={() => handleConfigureVM(vm.vm_name)}
                              >
                                <HiOutlineCog className="mr-1 h-3 w-3" />
                                {vm.has_mappings ? "Edit" : "Configure"}
                  </Button>
                              {vm.status === 'mapped' && (
                                <Badge color="success" size="xs">
                                  <HiOutlineCheck className="mr-1 h-3 w-3" />
                                  Ready
                                </Badge>
                              )}
                              {vm.status === 'error' && (
                                <Badge color="failure" size="xs">
                                  <HiOutlineExclamationCircle className="mr-1 h-3 w-3" />
                                  Error
                                </Badge>
                              )}
                </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
            </div>
          )}
        </Card>

          {/* Simple Next Steps Guide */}
          <Card className="bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800">
            <div className="flex items-start space-x-3">
              <HiOutlineGlobeAlt className="h-6 w-6 text-blue-500 mt-1" />
              <div>
                <h3 className="text-lg font-semibold text-blue-900 dark:text-blue-100 mb-2">
                  How to Configure Network Mapping
                </h3>
                <ol className="text-sm text-blue-800 dark:text-blue-200 space-y-1 list-decimal list-inside">
                  <li>Click <strong>"Configure"</strong> for any VM that shows "Needs Setup"</li>
                  <li>Map each VMware network to an OSSEA network for failover</li>
                  <li>Choose between production networks (live failover) or test networks (test failover)</li>
                  <li>Save your configuration - the VM will show as "Configured" when complete</li>
                </ol>
              </div>
            </div>
        </Card>

          {/* Network Configuration Modal */}
          <SimpleNetworkMappingModal
            isOpen={showConfigModal}
            onClose={handleModalClose}
            vmData={selectedVM}
            onSave={handleMappingSave}
        />
        </div>
      </main>
    </div>
  );
}
