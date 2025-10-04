// API route for network topology visualization - provides comprehensive network mapping data
// This endpoint aggregates source networks, destination networks, and existing mappings

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const vm_id = searchParams.get('vm_id');

    console.log('🌐 NETWORK TOPOLOGY API: Fetching network topology data', {
      vm_id,
      timestamp: new Date().toISOString()
    });

    // Fetch available OSSEA networks
    const networksResponse = await fetch(`${OMA_API_BASE}/networks/available`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    if (!networksResponse.ok) {
      throw new Error('Failed to fetch OSSEA networks');
    }

    const networksData = await networksResponse.json();
    const osseaNetworks = networksData.data || [];

    // Fetch existing network mappings
    const mappingsEndpoint = vm_id 
      ? `${OMA_API_BASE}/network-mappings/${vm_id}`
      : `${OMA_API_BASE}/network-mappings`;

    const mappingsResponse = await fetch(mappingsEndpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    let existingMappings = [];
    if (mappingsResponse.ok) {
      const mappingsData = await mappingsResponse.json();
      existingMappings = mappingsData.data || [];
    }

    // Fetch VM contexts to get source networks
    const vmContextsResponse = await fetch(`${OMA_API_BASE}/vm-contexts`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    let sourceNetworks = [];
    let unmappedSourceNetworks = [];

    if (vmContextsResponse.ok) {
      const vmContextsData = await vmContextsResponse.json();
      const vmContexts = vmContextsData.vm_contexts || [];

      // Extract unique source networks from VM contexts
      const sourceNetworkMap = new Map();
      
      for (const context of vmContexts) {
        // If filtering by VM ID, only include that VM's networks
        if (vm_id && context.vm_name !== vm_id) {
          continue;
        }

        // Get real VM network information from VMA discovery
        let vmNetworks = [];
        try {
          console.log(`🔍 TOPOLOGY: Discovering real networks for VM: ${context.vm_name}`);
          
          const discoveryResponse = await fetch(`http://localhost:9081/api/v1/discover`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              vcenter: context.vcenter_host || 'quad-vcenter-01.quadris.local',
              username: 'administrator@vsphere.local',
              password: 'EmyGVoBFesGQc47-',
              datacenter: context.datacenter || 'DatabanxDC',
              filter: context.vm_name
            })
          });

          if (discoveryResponse.ok) {
            const discoveryData = await discoveryResponse.json();
            const discoveredVM = discoveryData.vms?.find(vm => vm.name === context.vm_name);
            
            if (discoveredVM && discoveredVM.networks) {
              // Extract real network names from VMA discovery
              vmNetworks = discoveredVM.networks
                .map(net => net.network_name)
                .filter(name => name && name.trim() !== ''); // Filter out empty names
              
              console.log(`✅ TOPOLOGY: Discovered ${vmNetworks.length} real networks for ${context.vm_name}:`, vmNetworks);
            }
          }
        } catch (discoveryError) {
          console.warn(`⚠️ TOPOLOGY: VMA discovery failed for ${context.vm_name}, using fallback:`, discoveryError);
        }

        // Fallback to basic networks if discovery failed or returned no networks
        if (vmNetworks.length === 0) {
          vmNetworks = ['VM Network', 'Management Network']; // Generic VMware defaults instead of synthetic names
          console.log(`📋 TOPOLOGY: Using fallback networks for ${context.vm_name}:`, vmNetworks);
        }

        // Add each discovered network to the source network map
        for (const networkName of vmNetworks) {
          if (!sourceNetworkMap.has(networkName)) {
            sourceNetworkMap.set(networkName, {
              id: networkName,
              name: networkName,
              type: 'source',
              zone: context.datacenter || 'Unknown',
              state: 'active',
              connected_vms: 1
            });
          } else {
            const existing = sourceNetworkMap.get(networkName);
            existing.connected_vms += 1;
          }
        }
      }

      sourceNetworks = Array.from(sourceNetworkMap.values());

      // Identify unmapped source networks
      const mappedSourceNetworks = new Set(existingMappings.map(m => m.source_network_name));
      unmappedSourceNetworks = sourceNetworks
        .map(n => n.name)
        .filter(name => !mappedSourceNetworks.has(name));
    }

    // Transform OSSEA networks to include mapping counts
    const destinationNetworks = osseaNetworks.map(network => {
      const mappingCount = existingMappings.filter(
        m => m.destination_network_id === network.id
      ).length;

      return {
        id: network.id,
        name: network.name,
        type: 'destination',
        zone: network.zone_name,
        state: network.state,
        connected_vms: mappingCount,
        is_default: network.is_default || false
      };
    });

    // Enhance mappings with VM counts
    const enhancedMappings = existingMappings.map(mapping => ({
      ...mapping,
      vm_count: 1 // In a real implementation, count VMs using this mapping
    }));

    const topologyData = {
      source_networks: sourceNetworks,
      destination_networks: destinationNetworks,
      mappings: enhancedMappings,
      unmapped_source_networks: unmappedSourceNetworks,
      summary: {
        total_source_networks: sourceNetworks.length,
        total_destination_networks: destinationNetworks.length,
        total_mappings: enhancedMappings.length,
        unmapped_count: unmappedSourceNetworks.length
      }
    };

    console.log('✅ NETWORK TOPOLOGY API: Topology data compiled successfully', {
      source_networks: sourceNetworks.length,
      destination_networks: destinationNetworks.length,
      mappings: enhancedMappings.length,
      unmapped: unmappedSourceNetworks.length
    });

    return NextResponse.json(topologyData);

  } catch (error) {
    console.error('❌ NETWORK TOPOLOGY API: Error fetching topology data', error);
    return NextResponse.json(
      { 
        error: 'Failed to fetch network topology data',
        message: error instanceof Error ? error.message : 'Unknown error occurred',
        source_networks: [],
        destination_networks: [],
        mappings: [],
        unmapped_source_networks: []
      },
      { status: 500 }
    );
  }
}
