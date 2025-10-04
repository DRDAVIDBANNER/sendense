// API route for VM cleanup operations - proxies to OMA API
// This provides a bridge between Next.js frontend and Go OMA API for cleanup operations

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { 
      context_id, 
      vm_id, 
      vm_name, 
      vmware_vm_id,
      cleanup_type, 
      rollback_type,
      failover_type,
      power_on_source_vm,
      force_cleanup 
    } = body;

    // Support both old format (vm_name only) and new VM-centric format
    if (!vm_name && !context_id) {
      return NextResponse.json(
        { 
          success: false, 
          error: 'VM identifier is required',
          message: 'Please provide either vm_name or context_id for cleanup'
        },
        { status: 400 }
      );
    }

    console.log('🧹 CLEANUP API: Initiating enhanced rollback for VM', {
      context_id,
      vm_id,
      vm_name,
      vmware_vm_id,
      cleanup_type,
      rollback_type,
      failover_type,
      power_on_source_vm,
      force_cleanup,
      timestamp: new Date().toISOString()
    });

    // Determine endpoint based on rollback type
    const isUnifiedRollback = rollback_type === 'unified-rollback';
    const endpoint = isUnifiedRollback 
      ? `${OMA_API_BASE}/failover/rollback`  // Use unified rollback endpoint
      : `${OMA_API_BASE}/failover/cleanup/${vm_name || 'unknown'}`; // Legacy cleanup endpoint

    console.log('📤 CLEANUP API: Sending rollback request to OMA', {
      endpoint,
      context_id,
      vm_name,
      rollback_type: rollback_type || 'legacy-cleanup',
      failover_type,
      power_on_source_vm
    });

    // Prepare enhanced request payload
    const requestPayload = isUnifiedRollback ? {
      // Enhanced unified rollback request (matches backend RollbackRequest struct)
      context_id,
      vm_id,
      vm_name,
      vmware_vm_id,
      failover_type: failover_type || 'test',
      power_on_source: power_on_source_vm || false,  // ✅ Backend expects 'power_on_source'
      force_cleanup: force_cleanup || false
    } : {
      // Legacy cleanup request format
      context_id,
      vm_id,
      vm_name,
      cleanup_type
    };

    // Forward request to OMA API
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      },
      body: JSON.stringify(requestPayload)
    });

    const data = await response.json();

    console.log('📥 CLEANUP API: Response from OMA', {
      status: response.status,
      success: data.success,
      message: data.message
    });

    if (!response.ok) {
      console.error('❌ CLEANUP API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Cleanup request failed',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    const operationType = isUnifiedRollback 
      ? `${failover_type} failover rollback` 
      : 'cleanup operation';
    
    console.log('✅ CLEANUP API: Enhanced rollback initiated successfully', {
      vm_name,
      failover_type,
      power_on_source_vm,
      operation_type: operationType,
      message: data.message
    });

    return NextResponse.json({
      success: true,
      message: data.message || `${operationType} initiated successfully`,
      vm_name: vm_name,
      operation_type: operationType,
      rollback_options: isUnifiedRollback ? {
        failover_type,
        power_on_source_vm,
        force_cleanup
      } : undefined,
      data: data.data
    });

  } catch (error) {
    console.error('❌ CLEANUP API: Network or processing error', error);
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








