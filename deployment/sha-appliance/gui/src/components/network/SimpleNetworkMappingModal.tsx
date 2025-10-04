'use client';

import React, { useState, useEffect } from 'react';
import { Button, Badge, Spinner } from 'flowbite-react';
import { HiOutlineGlobeAlt, HiOutlineCheck, HiOutlineX, HiOutlineExclamationCircle, HiOutlineExclamationTriangle } from 'react-icons/hi';

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

interface SimpleNetworkMappingModalProps {
  isOpen: boolean;
  onClose: () => void;
  vmData: VMNetworkStatus | null;
  onSave: (mappings: NetworkMapping[]) => void;
}

export default function SimpleNetworkMappingModal({ 
  isOpen, 
  onClose, 
  vmData, 
  onSave 
}: SimpleNetworkMappingModalProps) {
  const [availableNetworks, setAvailableNetworks] = useState<OSSeaNetwork[]>([]);
  const [mappings, setMappings] = useState<Record<string, {
    production: {destination_id: string, destination_name: string} | null,
    test: {destination_id: string, destination_name: string} | null
  }>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load available OSSEA networks when modal opens
  useEffect(() => {
    if (isOpen && vmData) {
      loadAvailableNetworks();
      initializeMappings();
    }
  }, [isOpen, vmData]);

  const loadAvailableNetworks = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch('/api/networks');
      const data = await response.json();
      
      if (data.success) {
        setAvailableNetworks(data.networks || []);
      } else {
        setError('Failed to load available OSSEA networks');
      }
    } catch (err) {
      setError('Network error loading OSSEA networks');
      console.error('Failed to load networks:', err);
    } finally {
      setLoading(false);
    }
  };

  const initializeMappings = () => {
    if (!vmData) return;

    const initialMappings: Record<string, {
      production: {destination_id: string, destination_name: string} | null,
      test: {destination_id: string, destination_name: string} | null
    }> = {};
    
    // Initialize empty mappings for each VMware network
    vmData.vmware_networks.forEach(networkName => {
      initialMappings[networkName] = {
        production: null,
        test: null
      };
    });

    // Pre-populate with existing mappings
    vmData.ossea_networks.forEach(mapping => {
      if (initialMappings[mapping.source_network_name]) {
        const mappingType = mapping.is_test_network ? 'test' : 'production';
        initialMappings[mapping.source_network_name][mappingType] = {
          destination_id: mapping.destination_network_id,
          destination_name: mapping.destination_network_name
        };
      }
    });

    setMappings(initialMappings);
  };

  const handleNetworkMapping = (sourceNetwork: string, mappingType: 'production' | 'test', destinationNetworkId: string) => {
    const destinationNetwork = availableNetworks.find(net => net.id === destinationNetworkId);
    if (!destinationNetwork) return;

    setMappings(prev => ({
      ...prev,
      [sourceNetwork]: {
        ...prev[sourceNetwork],
        [mappingType]: {
          destination_id: destinationNetworkId,
          destination_name: destinationNetwork.name
        }
      }
    }));
  };

  const removeMapping = (sourceNetwork: string, mappingType: 'production' | 'test') => {
    setMappings(prev => ({
      ...prev,
      [sourceNetwork]: {
        ...prev[sourceNetwork],
        [mappingType]: null
      }
    }));
  };

  const handleSave = async () => {
    if (!vmData) return;

    try {
      setSaving(true);
      setError(null);

      // Save each mapping individually (API expects one mapping per call)
      const mappingPromises: Promise<any>[] = [];
      
      Object.entries(mappings).forEach(([sourceNetwork, networkMappings]) => {
        // Save production mapping if exists
        if (networkMappings.production) {
          mappingPromises.push(
            fetch('/api/network-mappings', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                vm_id: vmData.vm_name,
                source_network_name: sourceNetwork,
                destination_network_id: networkMappings.production.destination_id,
                destination_network_name: networkMappings.production.destination_name,
                is_test_network: false
              })
            }).then(async (response) => {
              if (!response.ok) {
                const errorData = await response.json();
                throw new Error(`Failed to save production mapping for ${sourceNetwork}: ${errorData.message || 'Unknown error'}`);
              }
              return response.json();
            })
          );
        }

        // Save test mapping if exists
        if (networkMappings.test) {
          mappingPromises.push(
            fetch('/api/network-mappings', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                vm_id: vmData.vm_name,
                source_network_name: sourceNetwork,
                destination_network_id: networkMappings.test.destination_id,
                destination_network_name: networkMappings.test.destination_name,
                is_test_network: true
              })
            }).then(async (response) => {
              if (!response.ok) {
                const errorData = await response.json();
                throw new Error(`Failed to save test mapping for ${sourceNetwork}: ${errorData.message || 'Unknown error'}`);
              }
              return response.json();
            })
          );
        }
      });

      // Wait for all mappings to be saved
      await Promise.all(mappingPromises);

      // Convert mappings to the format expected by parent component
      const mappingList: NetworkMapping[] = [];
      Object.entries(mappings).forEach(([sourceNetwork, networkMappings]) => {
        if (networkMappings.production) {
          mappingList.push({
            source_network_name: sourceNetwork,
            destination_network_name: networkMappings.production.destination_name,
            destination_network_id: networkMappings.production.destination_id,
            is_test_network: false,
            status: 'valid' as const
          });
        }
        if (networkMappings.test) {
          mappingList.push({
            source_network_name: sourceNetwork,
            destination_network_name: networkMappings.test.destination_name,
            destination_network_id: networkMappings.test.destination_id,
            is_test_network: true,
            status: 'valid' as const
          });
        }
      });

      onSave(mappingList);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save network mappings');
      console.error('Failed to save mappings:', err);
    } finally {
      setSaving(false);
    }
  };

  const getMappingStatus = () => {
    if (!vmData) return { mapped: 0, total: 0, production: 0, test: 0 };
    
    const total = vmData.vmware_networks.length;
    let production = 0;
    let test = 0;
    
    Object.values(mappings).forEach(networkMapping => {
      if (networkMapping.production) production++;
      if (networkMapping.test) test++;
    });
    
    const mapped = Math.min(production, test); // Both production and test needed for complete mapping
    return { mapped, total, production, test };
  };

  const isComplete = () => {
    const { production, test, total } = getMappingStatus();
    return production === total && test === total && total > 0;
  };

  if (!isOpen || !vmData) return null;

  const { mapped, total, production, test } = getMappingStatus();

  return (
    <div className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50 flex items-center justify-center">
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
        {/* Modal Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center space-x-3">
            <HiOutlineGlobeAlt className="h-6 w-6 text-blue-500" />
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Configure Network Mapping</h3>
              <p className="text-sm text-gray-500 dark:text-gray-400">{vmData.vm_name}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 bg-transparent hover:bg-gray-200 hover:text-gray-900 rounded-lg text-sm p-1.5 ml-auto inline-flex items-center dark:hover:bg-gray-600 dark:hover:text-white"
          >
            <HiOutlineX className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-6">
        <div className="space-y-6">
          {/* Error Alert */}
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded relative dark:bg-red-900/20 dark:border-red-800 dark:text-red-400">
              <div className="flex items-center space-x-2">
                <HiOutlineExclamationCircle className="h-4 w-4" />
                <span>{error}</span>
                <button
                  onClick={() => setError(null)}
                  className="absolute top-0 bottom-0 right-0 px-4 py-3"
                >
                  <HiOutlineX className="h-4 w-4" />
                </button>
              </div>
            </div>
          )}

          {/* Progress Indicator */}
          <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
            <div className="flex items-center justify-between mb-3">
              <span className="text-sm font-medium text-blue-900 dark:text-blue-100">
                Network Mapping Progress
              </span>
              <Badge color={isComplete() ? "success" : "warning"} size="sm">
                {mapped} of {total} networks fully mapped
              </Badge>
            </div>
            
            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="text-blue-800 dark:text-blue-200">Production Networks:</span>
                <span className="font-medium">{production}/{total}</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-1.5 dark:bg-gray-700">
                <div 
                  className="bg-green-500 h-1.5 rounded-full transition-all duration-300" 
                  style={{ width: total > 0 ? `${(production / total) * 100}%` : '0%' }}
                ></div>
              </div>
              
              <div className="flex items-center justify-between text-xs">
                <span className="text-blue-800 dark:text-blue-200">Test Networks:</span>
                <span className="font-medium">{test}/{total}</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-1.5 dark:bg-gray-700">
                <div 
                  className="bg-purple-500 h-1.5 rounded-full transition-all duration-300" 
                  style={{ width: total > 0 ? `${(test / total) * 100}%` : '0%' }}
                ></div>
              </div>
            </div>
          </div>

          {/* Network Mappings */}
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <Spinner size="lg" />
              <span className="ml-3 text-gray-500">Loading available networks...</span>
            </div>
          ) : (
            <div className="space-y-4">
              <h4 className="font-medium text-gray-900 dark:text-white">
                Map VMware Networks to OSSEA Networks
              </h4>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Each VMware network needs both a production network (for live failover) and a test network (for test failover).
              </p>
              
              {vmData.vmware_networks.map((sourceNetwork, index) => (
                <div key={index} className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                  <div className="mb-4">
                    <div className="font-medium text-gray-900 dark:text-white">{sourceNetwork}</div>
                    <div className="text-xs text-gray-500">VMware Source Network</div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {/* Production Network Mapping */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <label className="block text-sm font-medium text-green-700 dark:text-green-400">
                          Production Network:
                        </label>
                        {mappings[sourceNetwork]?.production && (
                          <button
                            onClick={() => removeMapping(sourceNetwork, 'production')}
                            className="text-red-500 hover:text-red-700 p-1"
                            title="Remove production mapping"
                          >
                            <HiOutlineX className="h-3 w-3" />
                          </button>
                        )}
                      </div>
                      
                      <select
                        value={mappings[sourceNetwork]?.production?.destination_id || ''}
                        onChange={(e) => handleNetworkMapping(sourceNetwork, 'production', e.target.value)}
                        className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-green-500 focus:border-green-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-green-500 dark:focus:border-green-500"
                      >
                        <option value="">Select production network...</option>
                        {availableNetworks.map((network) => (
                          <option key={network.id} value={network.id}>
                            {network.name} ({network.zone_name})
                          </option>
                        ))}
                      </select>

                      {mappings[sourceNetwork]?.production && (
                        <div className="flex items-center space-x-2 mt-1">
                          <Badge color="success" size="xs">Live Failover</Badge>
                          <span className="text-xs text-gray-500">
                            → {mappings[sourceNetwork].production.destination_name}
                          </span>
                        </div>
                      )}
                    </div>

                    {/* Test Network Mapping */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <label className="block text-sm font-medium text-purple-700 dark:text-purple-400">
                          Test Network:
                        </label>
                        {mappings[sourceNetwork]?.test && (
                          <button
                            onClick={() => removeMapping(sourceNetwork, 'test')}
                            className="text-red-500 hover:text-red-700 p-1"
                            title="Remove test mapping"
                          >
                            <HiOutlineX className="h-3 w-3" />
                          </button>
                        )}
                      </div>
                      
                      <select
                        value={mappings[sourceNetwork]?.test?.destination_id || ''}
                        onChange={(e) => handleNetworkMapping(sourceNetwork, 'test', e.target.value)}
                        className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-purple-500 focus:border-purple-500 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white dark:focus:ring-purple-500 dark:focus:border-purple-500"
                      >
                        <option value="">Select test network...</option>
                        {availableNetworks.map((network) => (
                          <option key={network.id} value={network.id}>
                            {network.name} ({network.zone_name})
                          </option>
                        ))}
                      </select>

                      {mappings[sourceNetwork]?.test && (
                        <div className="flex items-center space-x-2 mt-1">
                          <Badge color="purple" size="xs">Test Failover</Badge>
                          <span className="text-xs text-gray-500">
                            → {mappings[sourceNetwork].test.destination_name}
                          </span>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}

              {vmData.vmware_networks.length === 0 && (
                <div className="text-center py-6 text-gray-500">
                  <HiOutlineGlobeAlt className="mx-auto h-12 w-12 mb-3 text-gray-300" />
                  <p>No VMware networks discovered for this VM</p>
                </div>
              )}
            </div>
          )}
        </div>
        </div>

        {/* Modal Footer */}
        <div className="flex items-center justify-between p-6 border-t border-gray-200 dark:border-gray-700">
          <div className="text-sm text-gray-500 dark:text-gray-400">
            {isComplete() ? (
              <span className="text-green-600 dark:text-green-400 font-medium">✓ All networks fully mapped for live and test failovers</span>
            ) : (
              <span>Each VMware network needs both production and test network mappings</span>
            )}
          </div>
          
          <div className="flex items-center space-x-3">
            <Button color="gray" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            <Button 
              color="blue" 
              onClick={handleSave} 
              disabled={!isComplete() || saving || loading}
            >
              {saving ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Saving...
                </>
              ) : (
                'Save Configuration'
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
