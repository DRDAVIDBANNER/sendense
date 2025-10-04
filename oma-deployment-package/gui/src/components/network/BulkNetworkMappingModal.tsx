'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Modal, Button, Select, Alert, Badge, Spinner, TextInput } from 'flowbite-react';
import { HiOutlineGlobeAlt, HiOutlineCheck, HiOutlineX, HiOutlineSearch } from 'react-icons/hi';

interface VM {
  id: string;
  name: string;
  networks: NetworkInfo[];
  current_mappings: NetworkMapping[];
}

interface NetworkInfo {
  label: string;
  network_name: string;
  adapter_type: string;
  mac_address: string;
  connected: boolean;
}

interface NetworkMapping {
  id: number;
  source_network_name: string;
  destination_network_id: string;
  destination_network_name: string;
  is_test_network: boolean;
}

interface OSSEANetwork {
  id: string;
  name: string;
  zone_name: string;
  type: string;
  state: string;
  is_default: boolean;
}

interface BulkMappingRule {
  id: string;
  source_pattern: string;
  destination_network_id: string;
  destination_network_name: string;
  is_test_network: boolean;
  match_type: 'exact' | 'contains' | 'regex';
  priority: number;
}

interface BulkNetworkMappingModalProps {
  isOpen: boolean;
  onClose: () => void;
  selectedVMs: VM[];
  onSave: () => void;
}

export default function BulkNetworkMappingModal({ 
  isOpen, 
  onClose, 
  selectedVMs, 
  onSave 
}: BulkNetworkMappingModalProps) {
  const [osseaNetworks, setOSSEANetworks] = useState<OSSEANetwork[]>([]);
  const [mappingRules, setMappingRules] = useState<BulkMappingRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [previewMode, setPreviewMode] = useState(false);
  const [previewResults, setPreviewResults] = useState<{
    vm_id: string;
    vm_name: string;
    mappings: {
      source_network: string;
      destination_network: string;
      rule_matched: string;
      is_test: boolean;
    }[];
  }[]>([]);
  const [searchTerm, setSearchTerm] = useState('');

  const fetchOSSEANetworks = useCallback(async () => {
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
  }, []);

  useEffect(() => {
    if (isOpen) {
      fetchOSSEANetworks();
      initializeDefaultRules();
    }
  }, [isOpen, fetchOSSEANetworks]);

  const initializeDefaultRules = () => {
    // Extract unique source networks from selected VMs
    const sourceNetworks = new Set<string>();
    selectedVMs.forEach(vm => {
      vm.networks.forEach(network => {
        sourceNetworks.add(network.network_name || network.label);
      });
    });

    // Create default rules for each source network
    const defaultRules: BulkMappingRule[] = Array.from(sourceNetworks).map((networkName, index) => ({
      id: `rule-${index}`,
      source_pattern: networkName,
      destination_network_id: '',
      destination_network_name: '',
      is_test_network: false,
      match_type: 'exact',
      priority: index + 1
    }));

    setMappingRules(defaultRules);
  };

  const addMappingRule = () => {
    const newRule: BulkMappingRule = {
      id: `rule-${Date.now()}`,
      source_pattern: '',
      destination_network_id: '',
      destination_network_name: '',
      is_test_network: false,
      match_type: 'contains',
      priority: mappingRules.length + 1
    };
    setMappingRules([...mappingRules, newRule]);
  };

  const updateMappingRule = (ruleId: string, updates: Partial<BulkMappingRule>) => {
    setMappingRules(rules => 
      rules.map(rule => 
        rule.id === ruleId ? { ...rule, ...updates } : rule
      )
    );
  };

  const removeMappingRule = (ruleId: string) => {
    setMappingRules(rules => rules.filter(rule => rule.id !== ruleId));
  };

  const handleNetworkSelection = (ruleId: string, networkId: string) => {
    const network = osseaNetworks.find(n => n.id === networkId);
    if (network) {
      updateMappingRule(ruleId, {
        destination_network_id: networkId,
        destination_network_name: network.name
      });
    }
  };

  const generatePreview = async () => {
    try {
      setPreviewMode(true);
      setError('');

      const response = await fetch('/api/networks/bulk-mapping-preview', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          vm_ids: selectedVMs.map(vm => vm.id),
          mapping_rules: mappingRules.filter(rule => rule.destination_network_id)
        })
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to generate preview');
      }

      setPreviewResults(data.preview_results || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate preview');
    }
  };

  const applyBulkMapping = async () => {
    try {
      setLoading(true);
      setError('');
      setSuccess('');

      const response = await fetch('/api/networks/bulk-mapping', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          vm_ids: selectedVMs.map(vm => vm.id),
          mapping_rules: mappingRules.filter(rule => rule.destination_network_id)
        })
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to apply bulk mapping');
      }

      setSuccess(`Successfully applied network mappings to ${selectedVMs.length} VMs`);
      onSave();
      
      // Auto-close after success
      setTimeout(() => {
        onClose();
      }, 2000);

    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply bulk mapping');
    } finally {
      setLoading(false);
    }
  };

  const getMatchingNetworks = (pattern: string, matchType: string) => {
    const sourceNetworks = new Set<string>();
    selectedVMs.forEach(vm => {
      vm.networks.forEach(network => {
        const networkName = network.network_name || network.label;
        let matches = false;
        
        switch (matchType) {
          case 'exact':
            matches = networkName === pattern;
            break;
          case 'contains':
            matches = networkName.toLowerCase().includes(pattern.toLowerCase());
            break;
          case 'regex':
            try {
              matches = new RegExp(pattern, 'i').test(networkName);
            } catch {
              matches = false;
            }
            break;
        }
        
        if (matches) {
          sourceNetworks.add(networkName);
        }
      });
    });
    return Array.from(sourceNetworks);
  };

  const filteredNetworks = osseaNetworks.filter(network =>
    network.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    network.zone_name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Modal show={isOpen} onClose={onClose} size="6xl">
      <Modal.Header>
        <div className="flex items-center">
          <HiOutlineGlobeAlt className="mr-2 h-5 w-5" />
          Bulk Network Mapping - {selectedVMs.length} VMs Selected
        </div>
      </Modal.Header>
      
      <Modal.Body>
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

          {!previewMode ? (
            <>
              {/* Mapping Rules Configuration */}
              <div>
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold">Network Mapping Rules</h3>
                  <Button size="sm" color="blue" onClick={addMappingRule}>
                    Add Rule
                  </Button>
                </div>
                
                <div className="space-y-4">
                  {mappingRules.map((rule, index) => (
                    <div key={rule.id} className="p-4 border rounded-lg bg-gray-50 dark:bg-gray-800 dark:border-gray-700">
                      <div className="grid grid-cols-1 lg:grid-cols-6 gap-4 items-end">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Source Pattern
                          </label>
                          <TextInput
                            value={rule.source_pattern}
                            onChange={(e) => updateMappingRule(rule.id, { source_pattern: e.target.value })}
                            placeholder="Network name or pattern"
                            size="sm"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Match Type
                          </label>
                          <Select
                            value={rule.match_type}
                            onChange={(e) => updateMappingRule(rule.id, { match_type: e.target.value as any })}
                            size="sm"
                          >
                            <option value="exact">Exact Match</option>
                            <option value="contains">Contains</option>
                            <option value="regex">Regex Pattern</option>
                          </Select>
                        </div>

                        <div className="lg:col-span-2">
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                            Destination Network
                          </label>
                          <Select
                            value={rule.destination_network_id}
                            onChange={(e) => handleNetworkSelection(rule.id, e.target.value)}
                            size="sm"
                          >
                            <option value="">Select Network</option>
                            {filteredNetworks
                              .filter(n => n.state === 'Implemented')
                              .map(network => (
                                <option key={network.id} value={network.id}>
                                  {network.name} ({network.type}) - {network.zone_name}
                                </option>
                              ))
                            }
                          </Select>
                        </div>

                        <div>
                          <label className="flex items-center">
                            <input
                              type="checkbox"
                              checked={rule.is_test_network}
                              onChange={(e) => updateMappingRule(rule.id, { is_test_network: e.target.checked })}
                              className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded mr-2"
                            />
                            <span className="text-sm">Test Network</span>
                          </label>
                        </div>

                        <div>
                          <Button
                            size="sm"
                            color="failure"
                            onClick={() => removeMappingRule(rule.id)}
                          >
                            <HiOutlineX className="h-4 w-4" />
                          </Button>
                        </div>
                      </div>

                      {/* Show matching networks preview */}
                      {rule.source_pattern && (
                        <div className="mt-3">
                          <div className="text-sm text-gray-500 mb-2">
                            Matching networks ({getMatchingNetworks(rule.source_pattern, rule.match_type).length}):
                          </div>
                          <div className="flex flex-wrap gap-1">
                            {getMatchingNetworks(rule.source_pattern, rule.match_type).map((networkName, idx) => (
                              <Badge key={idx} color="gray" size="xs">
                                {networkName}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Network Search */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Search Available Networks
                </label>
                <div className="relative">
                  <TextInput
                    icon={HiOutlineSearch}
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    placeholder="Search networks by name or zone..."
                  />
                </div>
              </div>

              {/* Available Networks Summary */}
              <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
                <h4 className="font-semibold mb-2">Available OSSEA Networks ({filteredNetworks.length})</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2 text-sm max-h-32 overflow-y-auto">
                  {filteredNetworks.slice(0, 12).map(network => (
                    <div key={network.id} className="flex items-center justify-between">
                      <span className="truncate mr-2">{network.name}</span>
                      <Badge color={network.state === 'Implemented' ? 'success' : 'gray'} size="xs">
                        {network.type}
                      </Badge>
                    </div>
                  ))}
                  {filteredNetworks.length > 12 && (
                    <div className="text-gray-500 italic">
                      ... and {filteredNetworks.length - 12} more networks
                    </div>
                  )}
                </div>
              </div>
            </>
          ) : (
            <>
              {/* Preview Results */}
              <div>
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold">Mapping Preview</h3>
                  <Button size="sm" color="gray" onClick={() => setPreviewMode(false)}>
                    Back to Rules
                  </Button>
                </div>

                <div className="space-y-4">
                  {previewResults.map((vmResult) => (
                    <div key={vmResult.vm_id} className="border rounded-lg p-4">
                      <h4 className="font-semibold text-lg mb-3">{vmResult.vm_name}</h4>
                      
                      {vmResult.mappings.length > 0 ? (
                        <div className="overflow-x-auto">
                          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead className="bg-gray-50 dark:bg-gray-700">
                              <tr>
                                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                                  Source Network
                                </th>
                                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                                  Destination Network
                                </th>
                                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                                  Rule Matched
                                </th>
                                <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">
                                  Type
                                </th>
                              </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                              {vmResult.mappings.map((mapping, idx) => (
                                <tr key={idx}>
                                  <td className="px-4 py-2 text-sm font-medium text-gray-900 dark:text-white">
                                    {mapping.source_network}
                                  </td>
                                  <td className="px-4 py-2 text-sm text-gray-500 dark:text-gray-300">
                                    {mapping.destination_network}
                                  </td>
                                  <td className="px-4 py-2 text-sm text-gray-500 dark:text-gray-300">
                                    {mapping.rule_matched}
                                  </td>
                                  <td className="px-4 py-2">
                                    <Badge color={mapping.is_test ? 'purple' : 'blue'} size="xs">
                                      {mapping.is_test ? 'Test' : 'Production'}
                                    </Badge>
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      ) : (
                        <div className="text-center py-4 text-gray-500">
                          No mappings will be created for this VM
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}
        </div>
      </Modal.Body>

      <Modal.Footer>
        <div className="flex justify-between w-full">
          <Button color="gray" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          
          <div className="flex space-x-2">
            {!previewMode && (
              <Button 
                color="gray" 
                onClick={generatePreview}
                disabled={loading || mappingRules.filter(r => r.destination_network_id).length === 0}
              >
                <HiOutlineCheck className="mr-2 h-4 w-4" />
                Preview Mappings
              </Button>
            )}
            
            {previewMode && (
              <Button 
                color="blue" 
                onClick={applyBulkMapping}
                disabled={loading || previewResults.length === 0}
              >
                {loading ? (
                  <>
                    <Spinner size="sm" className="mr-2" />
                    Applying...
                  </>
                ) : (
                  <>
                    <HiOutlineCheck className="mr-2 h-4 w-4" />
                    Apply Bulk Mapping
                  </>
                )}
              </Button>
            )}
          </div>
        </div>
      </Modal.Footer>
    </Modal>
  );
}
