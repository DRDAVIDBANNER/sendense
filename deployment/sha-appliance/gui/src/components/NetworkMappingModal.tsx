'use client';

import { useState, useEffect } from 'react';
import { Modal, Button, Select, Alert, Badge } from 'flowbite-react';
import { HiCog } from 'react-icons/hi';

interface VM {
  id: string;
  name: string;
  networks?: NetworkInfo[];
}

interface NetworkInfo {
  label: string;
  network_name: string;
  adapter_type: string;
  mac_address: string;
  connected: boolean;
}

interface OSSEANetwork {
  id: string;
  name: string;
  zone_name: string;
  type: string;
  state: string;
  is_default: boolean;
}

interface NetworkMapping {
  id: number;
  source_network_name: string;
  destination_network_id: string;
  destination_network_name: string;
  is_test_network: boolean;
}

interface NetworkMappingModalProps {
  isOpen: boolean;
  onClose: () => void;
  vm: VM | null;
  onSave: () => void;
}

export default function NetworkMappingModal({ isOpen, onClose, vm, onSave }: NetworkMappingModalProps) {
  const [osseaNetworks, setOSSEANetworks] = useState<OSSEANetwork[]>([]);
  const [existingMappings, setExistingMappings] = useState<NetworkMapping[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [pendingMappings, setPendingMappings] = useState<{[key: string]: { production: string; test: string }}>({});

  // Fetch OSSEA networks and existing mappings when modal opens
  useEffect(() => {
    if (isOpen && vm) {
      fetchOSSEANetworks();
      fetchExistingMappings();
      initializePendingMappings();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, vm]);

  const fetchOSSEANetworks = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/networks');
      const data = await response.json();
      
      if (data.success) {
        setOSSEANetworks(data.networks || []);
      } else {
        setError('Failed to load OSSEA networks');
      }
    } catch (err) {
      setError('Network error loading OSSEA networks');
    } finally {
      setLoading(false);
    }
  };

  const fetchExistingMappings = async () => {
    if (!vm) return;
    
    try {
      const response = await fetch(`/api/network-mappings?vm_id=${vm.id}`);
      const data = await response.json();
      
      if (data.success) {
        setExistingMappings(data.mappings || []);
      }
    } catch (err) {
      console.error('Failed to fetch existing mappings:', err);
    }
  };

  const initializePendingMappings = () => {
    if (!vm?.networks) return;

    const initialized: {[key: string]: { production: string; test: string }} = {};
    
    vm.networks.forEach(network => {
      const networkName = network.network_name || network.label;
      
      // Find existing mappings for this network
      const productionMapping = existingMappings.find(m => 
        m.source_network_name === networkName && !m.is_test_network
      );
      const testMapping = existingMappings.find(m => 
        m.source_network_name === networkName && m.is_test_network
      );
      
      initialized[networkName] = {
        production: productionMapping?.destination_network_id || '',
        test: testMapping?.destination_network_id || ''
      };
    });
    
    setPendingMappings(initialized);
  };

  const handleMappingChange = (sourceNetwork: string, type: 'production' | 'test', destinationNetworkId: string) => {
    setPendingMappings(prev => ({
      ...prev,
      [sourceNetwork]: {
        ...prev[sourceNetwork],
        [type]: destinationNetworkId
      }
    }));
  };

  const saveNetworkMappings = async () => {
    if (!vm) return;

    try {
      setLoading(true);
      setError('');
      setSuccess('');

      // Process each source network
      for (const [sourceNetwork, mappings] of Object.entries(pendingMappings)) {
        // Save production mapping if selected
        if (mappings.production) {
          const osseaNetwork = osseaNetworks.find(n => n.id === mappings.production);
          await saveMapping(sourceNetwork, mappings.production, osseaNetwork?.name || 'Unknown', false);
        }

        // Save test mapping if selected
        if (mappings.test) {
          const osseaNetwork = osseaNetworks.find(n => n.id === mappings.test);
          await saveMapping(sourceNetwork, mappings.test, osseaNetwork?.name || 'Unknown', true);
        }
      }

      setSuccess('Network mappings saved successfully!');
      fetchExistingMappings(); // Refresh mappings
      onSave(); // Notify parent
      
      // Auto-close after success
      setTimeout(() => {
        onClose();
      }, 2000);

    } catch (err) {
      setError('Failed to save network mappings');
    } finally {
      setLoading(false);
    }
  };

  const saveMapping = async (sourceNetwork: string, destinationId: string, destinationName: string, isTest: boolean) => {
    const response = await fetch('/api/network-mappings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        vm_id: vm?.id,
        source_network_name: sourceNetwork,
        destination_network_id: destinationId,
        destination_network_name: destinationName,
        is_test_network: isTest
      })
    });

    if (!response.ok) {
      const data = await response.json();
      throw new Error(data.error || 'Failed to save mapping');
    }
  };

  // deleteMapping function removed as it's not currently used in the UI

  if (!vm) return null;

  return (
    <Modal show={isOpen} onClose={onClose} size="4xl">
      <div className="p-6">
        <div className="flex items-center mb-6">
          <HiCog className="mr-2 h-5 w-5" />
          <h2 className="text-xl font-semibold">Network Mapping Configuration - {vm.name}</h2>
        </div>
        <div className="space-y-6">
          {error && (
            <Alert color="failure" onDismiss={() => setError('')}>
              {error}
            </Alert>
          )}

          {success && (
            <Alert color="success" onDismiss={() => setSuccess('')}>
              {success}
            </Alert>
          )}

          <div>
            <h3 className="text-lg font-semibold mb-3">VM Network Configuration</h3>
            <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
              Configure network mappings for both production (live failover) and test environments.
            </p>
          </div>

          {/* VM Networks Table */}
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-700">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Source Network</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Adapter Info</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Production Mapping</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Test Mapping</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Status</th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                {vm.networks && vm.networks.length > 0 ? (
                  vm.networks.map((network, index) => {
                    const networkName = network.network_name || network.label;
                    const existingProd = existingMappings.find(m => 
                      m.source_network_name === networkName && !m.is_test_network
                    );
                    const existingTest = existingMappings.find(m => 
                      m.source_network_name === networkName && m.is_test_network
                    );

                    return (
                      <tr key={index} className="bg-white dark:border-gray-700 dark:bg-gray-800">
                        <td className="px-6 py-4 whitespace-nowrap font-medium text-gray-900 dark:text-white">
                          <div>
                            <div className="font-semibold">{networkName}</div>
                            <div className="text-xs text-gray-500">{network.label}</div>
                          </div>
                        </td>
                        
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="text-sm">
                            <Badge color={network.connected ? 'success' : 'gray'} size="xs">
                              {network.adapter_type}
                            </Badge>
                            <div className="text-xs text-gray-500 mt-1">
                              MAC: {network.mac_address}
                            </div>
                            <div className="text-xs text-gray-500">
                              {network.connected ? 'Connected' : 'Disconnected'}
                            </div>
                          </div>
                        </td>

                        {/* Production Mapping */}
                        <td className="px-6 py-4 whitespace-nowrap">
                          <Select
                            value={pendingMappings[networkName]?.production || ''}
                            onChange={(e) => handleMappingChange(networkName, 'production', e.target.value)}
                            sizing="sm"
                            disabled={loading}
                          >
                            <option value="">Select Production Network</option>
                            {osseaNetworks
                              .filter(n => n.state === 'Implemented' && !n.name.toLowerCase().includes('test'))
                              .map(network => (
                                <option key={network.id} value={network.id}>
                                  {network.name} ({network.type})
                                </option>
                              ))
                            }
                          </Select>
                          {existingProd && (
                            <div className="text-xs text-green-600 mt-1">
                              ✓ Mapped to: {existingProd.destination_network_name}
                            </div>
                          )}
                        </td>

                        {/* Test Mapping */}
                        <td className="px-6 py-4 whitespace-nowrap">
                          <Select
                            value={pendingMappings[networkName]?.test || ''}
                            onChange={(e) => handleMappingChange(networkName, 'test', e.target.value)}
                            sizing="sm"
                            disabled={loading}
                          >
                            <option value="">Select Test Network</option>
                            {osseaNetworks
                              .filter(n => n.state === 'Implemented')
                              .map(network => (
                                <option key={network.id} value={network.id}>
                                  {network.name} ({network.type})
                                </option>
                              ))
                            }
                          </Select>
                          {existingTest && (
                            <div className="text-xs text-blue-600 mt-1">
                              ✓ Mapped to: {existingTest.destination_network_name}
                            </div>
                          )}
                        </td>

                        {/* Status */}
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex flex-col space-y-1">
                            {(existingProd || pendingMappings[networkName]?.production) && (
                              <Badge color="success" size="xs">Live Ready</Badge>
                            )}
                            {(existingTest || pendingMappings[networkName]?.test) && (
                              <Badge color="purple" size="xs">Test Ready</Badge>
                            )}
                            {!existingProd && !existingTest && !pendingMappings[networkName]?.production && !pendingMappings[networkName]?.test && (
                              <Badge color="gray" size="xs">Not Mapped</Badge>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })
                ) : (
                  <tr>
                    <td colSpan={5} className="px-6 py-4 text-center text-gray-500">
                      No network adapters found for this VM
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Available OSSEA Networks Summary */}
          <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
            <h4 className="font-semibold mb-2">Available OSSEA Networks ({osseaNetworks.length})</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-sm">
              {osseaNetworks.slice(0, 6).map(network => (
                <div key={network.id} className="flex items-center justify-between">
                  <span className="truncate mr-2">{network.name}</span>
                  <Badge color={network.state === 'Implemented' ? 'success' : 'gray'} size="xs">
                    {network.type}
                  </Badge>
                </div>
              ))}
              {osseaNetworks.length > 6 && (
                <div className="text-gray-500 italic">
                  ... and {osseaNetworks.length - 6} more networks
                </div>
              )}
            </div>
          </div>
        </div>
        
        {/* Footer */}
        <div className="flex justify-between items-center mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
          <div className="text-sm text-gray-500">
            {existingMappings.length > 0 && (
              <span>Existing mappings: {existingMappings.length}</span>
            )}
          </div>
          <div className="flex space-x-2">
            <Button color="gray" onClick={onClose} disabled={loading}>
              Cancel
            </Button>
            <Button 
              onClick={saveNetworkMappings} 
              disabled={loading || Object.values(pendingMappings).every(m => !m.production && !m.test)}
            >
              {loading ? 'Saving...' : 'Save Network Mappings'}
            </Button>
          </div>
        </div>
      </div>
    </Modal>
  );
}
