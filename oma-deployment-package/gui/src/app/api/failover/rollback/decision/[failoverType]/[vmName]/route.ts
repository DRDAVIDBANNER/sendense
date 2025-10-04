// API route for rollback decision options - proxies to OMA API
// GET /api/failover/rollback/decision/{failover_type}/{vm_name}

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function GET(
  request: NextRequest,
  { params }: { params: { failoverType: string; vmName: string } }
) {
  try {
    const { failoverType, vmName } = await params;

    console.log('🔄 ROLLBACK DECISION API: Getting decision options', {
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
    const endpoint = `${OMA_API_BASE}/failover/rollback/decision/${failoverType}/${vmName}`;
    
    console.log('📤 ROLLBACK DECISION API: Requesting from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 ROLLBACK DECISION API: Response from OMA', {
      status: response.status,
      decision_id: data.decision_id,
      options: data.options?.length || 0
    });

    if (!response.ok) {
      console.error('❌ ROLLBACK DECISION API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to get rollback decision options',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ ROLLBACK DECISION API: Decision options retrieved successfully');

    return NextResponse.json({
      decision_id: data.decision_id,
      question: data.question,
      options: data.options,
      default_value: data.default_value,
      required: data.required
    });

  } catch (error) {
    console.error('❌ ROLLBACK DECISION API: Network or processing error', error);
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
