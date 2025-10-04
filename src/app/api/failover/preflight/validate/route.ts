// API route for pre-flight configuration validation - proxies to OMA API
// POST /api/failover/preflight/validate

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    console.log('🔍 PRE-FLIGHT VALIDATE API: Validating configuration', {
      context_id: body.context_id,
      failover_type: body.failover_type,
      vm_name: body.vm_name,
      timestamp: new Date().toISOString()
    });

    // Forward request to OMA API
    const endpoint = `${OMA_API_BASE}/failover/preflight/validate`;
    
    console.log('📤 PRE-FLIGHT VALIDATE API: Sending to OMA', { 
      endpoint,
      payload: { ...body, vm_name: body.vm_name }
    });

    const response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      },
      body: JSON.stringify(body)
    });

    const data = await response.json();

    console.log('📥 PRE-FLIGHT VALIDATE API: Response from OMA', {
      status: response.status,
      success: data.success,
      errors: data.errors?.length || 0
    });

    if (!response.ok) {
      console.error('❌ PRE-FLIGHT VALIDATE API: Validation failed', data);
      return NextResponse.json(
        { 
          success: false, 
          message: data.message || 'Configuration validation failed',
          errors: data.errors || ['Validation failed'],
          request: data.request
        },
        { status: response.status }
      );
    }

    console.log('✅ PRE-FLIGHT VALIDATE API: Configuration validation passed');

    return NextResponse.json({
      success: true,
      message: data.message || 'Configuration validation passed',
      request: data.request,
      metadata: data.metadata
    });

  } catch (error) {
    console.error('❌ PRE-FLIGHT VALIDATE API: Network or processing error', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API',
        errors: ['Network error during validation']
      },
      { status: 500 }
    );
  }
}
