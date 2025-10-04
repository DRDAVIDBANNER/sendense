// API route for bulk network mapping preview - shows what mappings would be created
// This endpoint previews bulk mapping results without actually creating them

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

interface BulkMappingPreviewRequest {
  vm_ids: string[];
  mapping_rules: BulkMappingRule[];
}

export async function POST(request: NextRequest) {
  try {
    const body: BulkMappingPreviewRequest = await request.json();
    const { vm_ids, mapping_rules } = body;

    console.log('👀 BULK MAPPING PREVIEW API: Generating preview', {
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

    // Fetch existing mappings to avoid duplicates in preview
    const existingMappingsResponse = await fetch(`${OMA_API_BASE}/network-mappings`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    let existingMappings = [];
    if (existingMappingsResponse.ok) {
      const mappingsData = await existingMappingsResponse.json();
      existingMappings = mappingsData.data || [];
    }

    const previewResults = [];

    // Process each VM for preview
    for (const vmContext of selectedVMContexts) {
      console.log(`👀 Previewing mappings for VM: ${vmContext.vm_name}`);

      // Get real VM network information from VMA discovery
      let vmNetworks = [];
      try {
        console.log(`🔍 PREVIEW: Discovering real networks for VM: ${vmContext.vm_name}`);
        
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
            
            console.log(`✅ PREVIEW: Discovered ${vmNetworks.length} real networks for ${vmContext.vm_name}:`, vmNetworks);
          }
        }
      } catch (discoveryError) {
        console.warn(`⚠️ PREVIEW: VMA discovery failed for ${vmContext.vm_name}, using fallback:`, discoveryError);
      }

      // Fallback to basic networks if discovery failed or returned no networks
      if (vmNetworks.length === 0) {
        vmNetworks = [
          'VM Network', // Common VMware network name
          'Management Network',
          'Production Network'
        ];
        console.log(`📋 PREVIEW: Using fallback networks for ${vmContext.vm_name}:`, vmNetworks);
      }

      const vmMappings = [];

      // Apply mapping rules to each network (preview only)
      for (const networkName of vmNetworks) {
        // Check if mapping already exists
        const existingMapping = existingMappings.find(m => 
          m.vm_id === vmContext.vm_name && 
          m.source_network_name === networkName
        );

        if (existingMapping) {
          // Skip networks that already have mappings
          continue;
        }

        // Find matching rule based on priority
        const sortedRules = [...mapping_rules].sort((a, b) => a.priority - b.priority);
        
        for (const rule of sortedRules) {
          if (!rule.destination_network_id) continue;

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

          if (matches) {
            vmMappings.push({
              source_network: networkName,
              destination_network: rule.destination_network_name,
              rule_matched: `${rule.match_type}: "${rule.source_pattern}"`,
              is_test: rule.is_test_network,
              confidence: rule.match_type === 'exact' ? 'high' : 
                         rule.match_type === 'contains' ? 'medium' : 'low'
            });

            console.log(`📋 Preview: ${networkName} → ${rule.destination_network_name} (${rule.match_type})`);
            break; // Stop at first matching rule
          }
        }
      }

      previewResults.push({
        vm_id: vmContext.vm_name,
        vm_name: vmContext.vm_name,
        mappings: vmMappings
      });
    }

    // Calculate preview statistics
    const totalMappings = previewResults.reduce((sum, result) => sum + result.mappings.length, 0);
    const vmsWithMappings = previewResults.filter(result => result.mappings.length > 0).length;
    const productionMappings = previewResults.reduce((sum, result) => 
      sum + result.mappings.filter(m => !m.is_test).length, 0
    );
    const testMappings = previewResults.reduce((sum, result) => 
      sum + result.mappings.filter(m => m.is_test).length, 0
    );

    const response = {
      preview_results: previewResults,
      summary: {
        total_vms: selectedVMContexts.length,
        vms_with_mappings: vmsWithMappings,
        total_mappings: totalMappings,
        production_mappings: productionMappings,
        test_mappings: testMappings,
        rules_applied: mapping_rules.length
      },
      validation: {
        all_vms_have_mappings: vmsWithMappings === selectedVMContexts.length,
        has_production_mappings: productionMappings > 0,
        has_test_mappings: testMappings > 0,
        rules_coverage: mapping_rules.filter(r => r.destination_network_id).length / mapping_rules.length
      }
    };

    console.log('✅ BULK MAPPING PREVIEW API: Preview generated successfully', {
      total_vms: selectedVMContexts.length,
      vms_with_mappings: vmsWithMappings,
      total_mappings: totalMappings
    });

    return NextResponse.json(response);

  } catch (error) {
    console.error('❌ BULK MAPPING PREVIEW API: Error generating preview', error);
    return NextResponse.json(
      { 
        error: 'Failed to generate bulk mapping preview',
        message: error instanceof Error ? error.message : 'Unknown error occurred',
        preview_results: [],
        summary: {
          total_vms: 0,
          vms_with_mappings: 0,
          total_mappings: 0,
          production_mappings: 0,
          test_mappings: 0,
          rules_applied: 0
        }
      },
      { status: 500 }
    );
  }
}
