// API route for unified failover operations - proxies to OMA API
// POST /api/failover/unified

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { context_id, vmware_vm_id, vm_name, failover_type, ...options } = body;

    console.log('🚀 UNIFIED FAILOVER API: Initiating unified failover', {
      context_id,
      vmware_vm_id,
      vm_name,
      failover_type,
      timestamp: new Date().toISOString()
    });

    // Prepare unified failover request payload
    const payload = {
      context_id,
      vmware_vm_id,
      vm_name,
      failover_type,
      
      // Optional behaviors for live failover
      ...(options.power_off_source !== undefined && { power_off_source: options.power_off_source }),
      ...(options.perform_final_sync !== undefined && { perform_final_sync: options.perform_final_sync }),
      
      // Optional behaviors for both types
      ...(options.skip_validation !== undefined && { skip_validation: options.skip_validation }),
      ...(options.skip_virtio !== undefined && { skip_virtio: options.skip_virtio }),
      
      // Network and VM naming options
      ...(options.network_strategy && { network_strategy: options.network_strategy }),
      ...(options.vm_naming && { vm_naming: options.vm_naming }),
      
      // Advanced options
      ...(options.test_duration && { test_duration: options.test_duration }),
      ...(options.custom_config && { custom_config: options.custom_config }),
      ...(options.network_mappings && { network_mappings: options.network_mappings })
    };

    console.log('📤 UNIFIED FAILOVER API: Sending request to OMA', {
      endpoint: `${OMA_API_BASE}/failover/unified`,
      payload: { ...payload, failover_type }
    });

    // Forward request to OMA unified failover API
    const response = await fetch(`${OMA_API_BASE}/failover/unified`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      },
      body: JSON.stringify(payload)
    });

    const data = await response.json();

    console.log('📥 UNIFIED FAILOVER API: Response from OMA', {
      status: response.status,
      success: data.success,
      job_id: data.job_id,
      message: data.message
    });

    if (!response.ok) {
      console.error('❌ UNIFIED FAILOVER API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Unified failover request failed',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ UNIFIED FAILOVER API: Unified failover initiated successfully', {
      job_id: data.job_id,
      failover_type: failover_type
    });

    return NextResponse.json({
      success: true,
      message: data.message,
      job_id: data.job_id,
      data: data.data
    });

  } catch (error) {
    console.error('❌ UNIFIED FAILOVER API: Network or processing error', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API'
      },
      { status: 500 }
    );
  }
}
