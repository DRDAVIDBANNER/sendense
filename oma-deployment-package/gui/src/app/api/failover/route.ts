// API route for VM failover operations - proxies to OMA API
// This provides a bridge between Next.js frontend and Go OMA API

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { context_id, vm_id, failover_type, ...options } = body;

    console.log('🚀 FAILOVER API: Initiating failover', {
      context_id,
      vm_id,
      failover_type,
      timestamp: new Date().toISOString()
    });

    // Determine endpoint based on failover type
    const endpoint = failover_type === 'test' 
      ? `${OMA_API_BASE}/failover/test`
      : `${OMA_API_BASE}/failover/live`;

    // Prepare request payload with VM-centric identifiers
    const payload = {
      context_id,
      vm_id,
      vm_name: options.vm_name || `VM-${vm_id}`,
      skip_validation: options.skip_validation || false,
      network_mappings: options.network_mappings || {},
      custom_config: options.custom_config || {},
      notification_config: options.notification_config || {},
      ...(failover_type === 'test' && {
        test_duration: options.test_duration || '2h',
        auto_cleanup: options.auto_cleanup || true
      })
    };

    console.log('📤 FAILOVER API: Sending request to OMA', {
      endpoint,
      payload: { ...payload, vm_id, failover_type }
    });

    // Forward request to OMA API
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      },
      body: JSON.stringify(payload)
    });

    const data = await response.json();

    console.log('📥 FAILOVER API: Response from OMA', {
      status: response.status,
      success: data.success,
      job_id: data.job_id,
      message: data.message
    });

    if (!response.ok) {
      console.error('❌ FAILOVER API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failover request failed',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ FAILOVER API: Failover initiated successfully', {
      job_id: data.job_id,
      estimated_duration: data.estimated_duration
    });

    return NextResponse.json({
      success: true,
      message: data.message,
      job_id: data.job_id,
      estimated_duration: data.estimated_duration,
      data: data.data
    });

  } catch (error) {
    console.error('❌ FAILOVER API: Network or processing error', error);
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

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const type = searchParams.get('type');
    const status = searchParams.get('status');
    const vm_id = searchParams.get('vm_id');

    console.log('📋 FAILOVER API: Listing failover jobs', {
      type, status, vm_id,
      timestamp: new Date().toISOString()
    });

    // Build query parameters
    const params = new URLSearchParams();
    if (type) params.append('type', type);
    if (status) params.append('status', status);
    if (vm_id) params.append('vm_id', vm_id);

    const endpoint = `${OMA_API_BASE}/failover/jobs?${params.toString()}`;

    console.log('📤 FAILOVER API: Requesting job list from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 FAILOVER API: Job list response from OMA', {
      status: response.status,
      success: data.success,
      total_jobs: data.total || 0
    });

    if (!response.ok) {
      console.error('❌ FAILOVER API: Failed to get job list', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to get failover jobs',
          jobs: []
        },
        { status: response.status }
      );
    }

    console.log('✅ FAILOVER API: Job list retrieved successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      total: data.total,
      jobs: data.jobs || [],
      filters: data.filters || {}
    });

  } catch (error) {
    console.error('❌ FAILOVER API: Error fetching job list', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API',
        jobs: []
      },
      { status: 500 }
    );
  }
}



