// API route for enhanced rollback operations - proxies to OMA API
// POST /api/failover/rollback

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
      failover_type, 
      power_on_source,      // Legacy field name
      power_on_source_vm,   // New field name from enhanced component
      force_cleanup 
    } = body;

    // Support both field names for compatibility
    const powerOnSource = power_on_source_vm !== undefined ? power_on_source_vm : power_on_source;

    console.log('🔄 ENHANCED ROLLBACK API: Initiating enhanced rollback', {
      context_id,
      vm_id,
      vm_name,
      vmware_vm_id,
      failover_type,
      power_on_source: powerOnSource,
      force_cleanup,
      rollback_type: 'enhanced',
      timestamp: new Date().toISOString()
    });

    // Prepare rollback request payload (backend expects 'power_on_source')
    const payload = {
      context_id,
      vm_id,
      vm_name,
      vmware_vm_id,
      failover_type,
      power_on_source: powerOnSource || false,  // ✅ Map to backend field name
      force_cleanup: force_cleanup || false
    };

    console.log('📤 ENHANCED ROLLBACK API: Sending request to OMA', {
      endpoint: `${OMA_API_BASE}/failover/rollback`,
      payload
    });

    // Forward request to OMA enhanced rollback API
    const response = await fetch(`${OMA_API_BASE}/failover/rollback`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      },
      body: JSON.stringify(payload)
    });

    const data = await response.json();

    console.log('📥 ENHANCED ROLLBACK API: Response from OMA', {
      status: response.status,
      success: data.success,
      message: data.message
    });

    if (!response.ok) {
      console.error('❌ ENHANCED ROLLBACK API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Enhanced rollback request failed',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ ENHANCED ROLLBACK API: Enhanced rollback initiated successfully', {
      failover_type: failover_type,
      power_on_source: powerOnSource,
      operation_type: `${failover_type} failover rollback`
    });

    return NextResponse.json({
      success: true,
      message: data.message,
      job_id: data.job_id,
      operation_type: `${failover_type} failover rollback`,
      rollback_options: {
        failover_type,
        power_on_source: powerOnSource,
        force_cleanup
      }
    });

  } catch (error) {
    console.error('❌ ENHANCED ROLLBACK API: Network or processing error', error);
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
