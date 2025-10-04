// API route for bulk network mapping operations - apply network mappings to multiple VMs
// This endpoint processes bulk mapping rules and applies them to selected VMs

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL
const OMA_API_BASE = 'http://localhost:8082/api/v1';

interface BulkMappingRule {
  id: string;
  source_pattern: string;
  destination_network_id: string;
  destination_network_name: string;
  is_test_network: boolean;
  match_type: 'exact' | 'contains' | 'regex';
  priority: number;
}

interface BulkMappingRequest {
  vm_ids: string[];
  mapping_rules: BulkMappingRule[];
}

export async function POST(request: NextRequest) {
  try {
    const body: BulkMappingRequest = await request.json();
    const { vm_ids, mapping_rules } = body;

    console.log('📦 BULK NETWORK MAPPING API: Processing bulk mapping request', {
      vm_count: vm_ids.length,
      rule_count: mapping_rules.length,
      timestamp: new Date().toISOString()
    });

    if (!vm_ids.length || !mapping_rules.length) {
      return NextResponse.json(
        { 
          error: 'Missing required parameters',
          message: 'vm_ids and mapping_rules are required'
        },
        { status: 400 }
      );
    }

    // Fetch VM contexts to get network information
    const vmContextsResponse = await fetch(`${OMA_API_BASE}/vm-contexts`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    if (!vmContextsResponse.ok) {
      throw new Error('Failed to fetch VM contexts');
    }

    const vmContextsData = await vmContextsResponse.json();
    const allVMContexts = vmContextsData.vm_contexts || [];

    // Filter to selected VMs
    const selectedVMContexts = allVMContexts.filter(context => 
      vm_ids.includes(context.vm_name)
    );

    const results = [];
    const errors = [];

    // Process each VM
    for (const vmContext of selectedVMContexts) {
      try {
        console.log(`🔄 Processing VM: ${vmContext.vm_name}`);

        // Get real VM network information from VMA discovery
        let vmNetworks = [];
        try {
          console.log(`🔍 Discovering real networks for VM: ${vmContext.vm_name}`);
          
          const discoveryResponse = await fetch(`http://localhost:9081/api/v1/discover`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              vcenter: vmContext.vcenter_host || 'quad-vcenter-01.quadris.local',
              username: 'administrator@vsphere.local',
              password: 'EmyGVoBFesGQc47-',
              datacenter: vmContext.datacenter || 'DatabanxDC',
              filter: vmContext.vm_name
            })
          });

          if (discoveryResponse.ok) {
            const discoveryData = await discoveryResponse.json();
            const discoveredVM = discoveryData.vms?.find(vm => vm.name === vmContext.vm_name);
            
            if (discoveredVM && discoveredVM.networks) {
              // Extract real network names from VMA discovery
              vmNetworks = discoveredVM.networks
                .map(net => net.network_name)
                .filter(name => name && name.trim() !== ''); // Filter out empty names
              
              console.log(`✅ Discovered ${vmNetworks.length} real networks for ${vmContext.vm_name}:`, vmNetworks);
            }
          }
        } catch (discoveryError) {
          console.warn(`⚠️ VMA discovery failed for ${vmContext.vm_name}, using fallback:`, discoveryError);
        }

        // Fallback to basic networks if discovery failed or returned no networks
        if (vmNetworks.length === 0) {
          vmNetworks = ['VM Network', 'Management Network']; // Generic VMware defaults instead of synthetic names
          console.log(`📋 Using fallback networks for ${vmContext.vm_name}:`, vmNetworks);
        }

        const vmMappings = [];

        // Apply mapping rules to each network
        for (const networkName of vmNetworks) {
          // Find matching rule based on priority
          const sortedRules = [...mapping_rules].sort((a, b) => a.priority - b.priority);
          
          for (const rule of sortedRules) {
            let matches = false;

            switch (rule.match_type) {
              case 'exact':
                matches = networkName === rule.source_pattern;
                break;
              case 'contains':
                matches = networkName.toLowerCase().includes(rule.source_pattern.toLowerCase());
                break;
              case 'regex':
                try {
                  matches = new RegExp(rule.source_pattern, 'i').test(networkName);
                } catch {
                  console.warn(`Invalid regex pattern: ${rule.source_pattern}`);
                  matches = false;
                }
                break;
            }

            if (matches && rule.destination_network_id) {
              // Create the network mapping
              const mappingPayload = {
                vm_id: vmContext.vm_name,
                source_network_name: networkName,
                destination_network_id: rule.destination_network_id,
                destination_network_name: rule.destination_network_name,
                is_test_network: rule.is_test_network
              };

              console.log(`📤 Creating mapping for ${vmContext.vm_name}: ${networkName} → ${rule.destination_network_name}`);

              const mappingResponse = await fetch(`${OMA_API_BASE}/network-mappings`, {
                method: 'POST',
                headers: {
                  'Content-Type': 'application/json',
                  'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
                },
                body: JSON.stringify(mappingPayload)
              });

              if (mappingResponse.ok) {
                const mappingData = await mappingResponse.json();
                vmMappings.push({
                  source_network: networkName,
                  destination_network: rule.destination_network_name,
                  rule_matched: rule.source_pattern,
                  is_test: rule.is_test_network,
                  mapping_id: mappingData.data?.id
                });
                console.log(`✅ Mapping created successfully for ${networkName}`);
              } else {
                const errorData = await mappingResponse.json();
                console.error(`❌ Failed to create mapping for ${networkName}:`, errorData);
                errors.push({
                  vm_name: vmContext.vm_name,
                  network_name: networkName,
                  error: errorData.error || 'Failed to create mapping'
                });
              }

              break; // Stop at first matching rule
            }
          }
        }

        results.push({
          vm_id: vmContext.vm_name,
          vm_name: vmContext.vm_name,
          mappings_created: vmMappings.length,
          mappings: vmMappings
        });

      } catch (vmError) {
        console.error(`❌ Error processing VM ${vmContext.vm_name}:`, vmError);
        errors.push({
          vm_name: vmContext.vm_name,
          error: vmError instanceof Error ? vmError.message : 'Unknown error'
        });
      }
    }

    // Calculate summary
    const totalMappingsCreated = results.reduce((sum, result) => sum + result.mappings_created, 0);
    const successfulVMs = results.filter(result => result.mappings_created > 0).length;

    const response = {
      success: errors.length === 0,
      message: errors.length === 0 
        ? `Successfully applied bulk network mappings to ${successfulVMs} VMs`
        : `Bulk mapping completed with ${errors.length} errors`,
      summary: {
        total_vms_processed: selectedVMContexts.length,
        successful_vms: successfulVMs,
        total_mappings_created: totalMappingsCreated,
        errors_count: errors.length
      },
      results,
      errors
    };

    console.log('✅ BULK NETWORK MAPPING API: Bulk mapping completed', {
      successful_vms: successfulVMs,
      total_mappings: totalMappingsCreated,
      errors: errors.length
    });

    return NextResponse.json(response);

  } catch (error) {
    console.error('❌ BULK NETWORK MAPPING API: Error processing bulk mapping', error);
    return NextResponse.json(
      { 
        error: 'Failed to process bulk network mapping',
        message: error instanceof Error ? error.message : 'Unknown error occurred',
        success: false,
        results: [],
        errors: []
      },
      { status: 500 }
    );
  }
}
