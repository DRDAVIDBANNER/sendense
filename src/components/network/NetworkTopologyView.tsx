'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Card, Button, Badge, Alert, Spinner } from 'flowbite-react';
import { HiOutlineGlobeAlt, HiOutlineRefresh, HiOutlineEye, HiOutlineCog } from 'react-icons/hi';

interface NetworkNode {
  id: string;
  name: string;
  type: 'source' | 'destination';
  zone?: string;
  state: string;
  connected_vms: number;
  is_default?: boolean;
}

interface NetworkMapping {
  id: number;
  source_network_name: string;
  destination_network_id: string;
  destination_network_name: string;
  is_test_network: boolean;
  vm_count: number;
}

interface NetworkTopologyData {
  source_networks: NetworkNode[];
  destination_networks: NetworkNode[];
  mappings: NetworkMapping[];
  unmapped_source_networks: string[];
}

interface NetworkTopologyViewProps {
  vmId?: string;
  onMappingChange?: () => void;
}

export default function NetworkTopologyView({ vmId, onMappingChange }: NetworkTopologyViewProps) {
  const [topologyData, setTopologyData] = useState<NetworkTopologyData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedMapping, setSelectedMapping] = useState<NetworkMapping | null>(null);
  const [viewMode, setViewMode] = useState<'topology' | 'table'>('topology');

  const fetchTopologyData = useCallback(async () => {
    try {
      setLoading(true);
      setError('');

      // Fetch network topology data
      const endpoint = vmId 
        ? `/api/networks/topology?vm_id=${vmId}`
        : '/api/networks/topology';

      const response = await fetch(endpoint);
      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to fetch network topology');
      }

      setTopologyData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load network topology');
    } finally {
      setLoading(false);
    }
  }, [vmId]);

  useEffect(() => {
    fetchTopologyData();
  }, [fetchTopologyData]);

  const handleRefresh = () => {
    fetchTopologyData();
  };

  const getNetworkStatusColor = (state: string) => {
    switch (state.toLowerCase()) {
      case 'implemented':
      case 'active':
        return 'success';
      case 'allocated':
      case 'pending':
        return 'warning';
      case 'destroyed':
      case 'error':
        return 'failure';
      default:
        return 'gray';
    }
  };

  const getMappingTypeColor = (isTest: boolean) => {
    return isTest ? 'purple' : 'blue';
  };

  const renderTopologyView = () => {
    if (!topologyData) return null;

    return (
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Source Networks */}
        <Card className="h-fit">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Source Networks (VMware)
            </h3>
            <Badge color="gray" size="sm">
              {topologyData.source_networks.length}
            </Badge>
          </div>
          
          <div className="space-y-3">
            {topologyData.source_networks.map((network) => (
              <div
                key={network.id}
                className="p-3 border rounded-lg bg-gray-50 dark:bg-gray-800 dark:border-gray-700"
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="font-medium text-sm">{network.name}</span>
                  <Badge color={getNetworkStatusColor(network.state)} size="xs">
                    {network.state}
                  </Badge>
                </div>
                <div className="text-xs text-gray-500">
                  <div>VMs: {network.connected_vms}</div>
                  {network.zone && <div>Zone: {network.zone}</div>}
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* Network Mappings */}
        <Card className="h-fit">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Network Mappings
            </h3>
            <Badge color="blue" size="sm">
              {topologyData.mappings.length}
            </Badge>
          </div>

          <div className="space-y-3">
            {topologyData.mappings.length > 0 ? (
              topologyData.mappings.map((mapping) => (
                <div
                  key={mapping.id}
                  className={`p-3 border rounded-lg cursor-pointer transition-colors ${
                    selectedMapping?.id === mapping.id
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800'
                  }`}
                  onClick={() => setSelectedMapping(mapping)}
                >
                  <div className="flex items-center justify-between mb-2">
                    <Badge color={getMappingTypeColor(mapping.is_test_network)} size="xs">
                      {mapping.is_test_network ? 'Test' : 'Production'}
                    </Badge>
                    <span className="text-xs text-gray-500">
                      {mapping.vm_count} VMs
                    </span>
                  </div>
                  
                  <div className="text-sm">
                    <div className="font-medium text-gray-900 dark:text-white truncate">
                      {mapping.source_network_name}
                    </div>
                    <div className="text-gray-500 text-xs mt-1">
                      ↓ {mapping.destination_network_name}
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center py-8 text-gray-500">
                <HiOutlineGlobeAlt className="mx-auto h-8 w-8 mb-2" />
                <p className="text-sm">No network mappings configured</p>
              </div>
            )}
          </div>
        </Card>

        {/* Destination Networks */}
        <Card className="h-fit">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Destination Networks (OSSEA)
            </h3>
            <Badge color="gray" size="sm">
              {topologyData.destination_networks.length}
            </Badge>
          </div>
          
          <div className="space-y-3">
            {topologyData.destination_networks.map((network) => (
              <div
                key={network.id}
                className={`p-3 border rounded-lg ${
                  network.is_default
                    ? 'border-green-200 bg-green-50 dark:bg-green-900/20 dark:border-green-700'
                    : 'border-gray-200 bg-gray-50 dark:bg-gray-800 dark:border-gray-700'
                }`}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="font-medium text-sm">{network.name}</span>
                  <div className="flex items-center space-x-1">
                    {network.is_default && (
                      <Badge color="success" size="xs">Default</Badge>
                    )}
                    <Badge color={getNetworkStatusColor(network.state)} size="xs">
                      {network.state}
                    </Badge>
                  </div>
                </div>
                <div className="text-xs text-gray-500">
                  <div>Zone: {network.zone || 'Unknown'}</div>
                  <div>Mapped VMs: {network.connected_vms}</div>
                </div>
              </div>
            ))}
          </div>
        </Card>
      </div>
    );
  };

  const renderTableView = () => {
    if (!topologyData) return null;

    return (
      <Card>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Source Network
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Destination Network
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  VM Count
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                  Status
                </th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {topologyData.mappings.map((mapping) => {
                const destNetwork = topologyData.destination_networks.find(
                  n => n.id === mapping.destination_network_id
                );
                
                return (
                  <tr key={mapping.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                      {mapping.source_network_name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">
                      {mapping.destination_network_name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <Badge color={getMappingTypeColor(mapping.is_test_network)} size="sm">
                        {mapping.is_test_network ? 'Test' : 'Production'}
                      </Badge>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">
                      {mapping.vm_count}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <Badge 
                        color={getNetworkStatusColor(destNetwork?.state || 'unknown')} 
                        size="sm"
                      >
                        {destNetwork?.state || 'Unknown'}
                      </Badge>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </Card>
    );
  };

  const renderUnmappedNetworks = () => {
    if (!topologyData?.unmapped_source_networks.length) return null;

    return (
      <Alert color="warning" className="mb-6">
        <div className="flex items-center">
          <HiOutlineGlobeAlt className="mr-2 h-4 w-4" />
          <span className="font-medium">Unmapped Source Networks</span>
        </div>
        <div className="mt-2">
          <p className="text-sm">
            The following source networks are not mapped to any destination networks:
          </p>
          <div className="flex flex-wrap gap-2 mt-2">
            {topologyData.unmapped_source_networks.map((networkName, index) => (
              <Badge key={index} color="warning" size="sm">
                {networkName}
              </Badge>
            ))}
          </div>
        </div>
      </Alert>
    );
  };

  if (loading) {
    return (
      <Card>
        <div className="flex items-center justify-center py-12">
          <Spinner size="lg" />
          <span className="ml-3 text-gray-500">Loading network topology...</span>
        </div>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <Alert color="failure">
          <div className="flex items-center justify-between">
            <span>{error}</span>
            <Button size="sm" color="failure" onClick={handleRefresh}>
              <HiOutlineRefresh className="mr-2 h-4 w-4" />
              Retry
            </Button>
          </div>
        </Alert>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
            Network Topology
          </h2>
          <p className="text-gray-500 dark:text-gray-400">
            Visualize and manage network mappings between VMware and OSSEA environments
          </p>
        </div>
        
        <div className="flex items-center space-x-2">
          <div className="flex rounded-md shadow-sm">
            <Button
              size="sm"
              color={viewMode === 'topology' ? 'blue' : 'gray'}
              onClick={() => setViewMode('topology')}
              className="rounded-r-none"
            >
              <HiOutlineEye className="mr-2 h-4 w-4" />
              Topology
            </Button>
            <Button
              size="sm"
              color={viewMode === 'table' ? 'blue' : 'gray'}
              onClick={() => setViewMode('table')}
              className="rounded-l-none border-l-0"
            >
              <HiOutlineCog className="mr-2 h-4 w-4" />
              Table
            </Button>
          </div>
          
          <Button size="sm" color="gray" onClick={handleRefresh}>
            <HiOutlineRefresh className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Unmapped Networks Alert */}
      {renderUnmappedNetworks()}

      {/* Main Content */}
      {viewMode === 'topology' ? renderTopologyView() : renderTableView()}

      {/* Selected Mapping Details */}
      {selectedMapping && (
        <Card>
          <h3 className="text-lg font-semibold mb-4">Mapping Details</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Source Network
              </label>
              <div className="text-sm text-gray-900 dark:text-white">
                {selectedMapping.source_network_name}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Destination Network
              </label>
              <div className="text-sm text-gray-900 dark:text-white">
                {selectedMapping.destination_network_name}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Mapping Type
              </label>
              <Badge color={getMappingTypeColor(selectedMapping.is_test_network)} size="sm">
                {selectedMapping.is_test_network ? 'Test Environment' : 'Production Environment'}
              </Badge>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Associated VMs
              </label>
              <div className="text-sm text-gray-900 dark:text-white">
                {selectedMapping.vm_count} VMs
              </div>
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}
