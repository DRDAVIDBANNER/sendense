// API route for pre-flight configuration discovery - proxies to OMA API
// GET /api/failover/preflight/config/{failover_type}/{vm_name}

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function GET(
  request: NextRequest,
  { params }: { params: { failoverType: string; vmName: string } }
) {
  try {
    const { failoverType, vmName } = await params;

    console.log('🔧 PRE-FLIGHT CONFIG API: Getting configuration options', {
      failoverType,
      vmName,
      timestamp: new Date().toISOString()
    });

    // Validate failover type
    if (!['live', 'test'].includes(failoverType)) {
      return NextResponse.json(
        { 
          success: false, 
          error: 'Invalid failover type',
          message: 'Failover type must be "live" or "test"'
        },
        { status: 400 }
      );
    }

    // Forward request to OMA API
    const endpoint = `${OMA_API_BASE}/failover/preflight/config/${failoverType}/${vmName}`;
    
    console.log('📤 PRE-FLIGHT CONFIG API: Requesting from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 PRE-FLIGHT CONFIG API: Response from OMA', {
      status: response.status,
      success: data.success,
      failover_type: data.failover_type
    });

    if (!response.ok) {
      console.error('❌ PRE-FLIGHT CONFIG API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to get configuration options',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ PRE-FLIGHT CONFIG API: Configuration options retrieved successfully');

    return NextResponse.json({
      success: true,
      failover_type: data.failover_type,
      vm_name: data.vm_name,
      configuration: data.configuration,
      metadata: data.metadata
    });

  } catch (error) {
    console.error('❌ PRE-FLIGHT CONFIG API: Network or processing error', error);
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
