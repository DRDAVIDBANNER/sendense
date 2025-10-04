// API route for network mapping management - proxies to OMA API network mapping endpoints
// This provides CRUD operations for VM network mappings

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const vm_id = searchParams.get('vm_id');

    console.log('📋 NETWORK MAPPINGS API: Getting network mappings', {
      vm_id,
      timestamp: new Date().toISOString()
    });

    // Build endpoint based on whether vm_id is provided
    const endpoint = vm_id 
      ? `${OMA_API_BASE}/network-mappings/${vm_id}`
      : `${OMA_API_BASE}/network-mappings`;

    console.log('📤 NETWORK MAPPINGS API: Requesting from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 NETWORK MAPPINGS API: Response from OMA', {
      status: response.status,
      success: data.success,
      mapping_count: data.data?.length || 0
    });

    if (!response.ok) {
      console.error('❌ NETWORK MAPPINGS API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to fetch network mappings',
          message: data.message || 'Unknown error occurred',
          mappings: []
        },
        { status: response.status }
      );
    }

    console.log('✅ NETWORK MAPPINGS API: Mappings retrieved successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      mappings: data.data || [],
      total: data.data?.length || 0
    });

  } catch (error) {
    console.error('❌ NETWORK MAPPINGS API: Network or processing error', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API',
        mappings: []
      },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { vm_id, source_network_name, destination_network_id, destination_network_name, is_test_network } = body;

    console.log('🔧 NETWORK MAPPINGS API: Creating network mapping', {
      vm_id,
      source_network_name,
      destination_network_id,
      destination_network_name,
      is_test_network,
      timestamp: new Date().toISOString()
    });

    // Prepare request payload
    const payload = {
      vm_id,
      source_network_name,
      destination_network_id,
      destination_network_name,
      is_test_network: is_test_network || false
    };

    console.log('📤 NETWORK MAPPINGS API: Sending to OMA', { payload });

    // Forward request to OMA API
    const response = await fetch(`${OMA_API_BASE}/network-mappings`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      },
      body: JSON.stringify(payload)
    });

    const data = await response.json();

    console.log('📥 NETWORK MAPPINGS API: Response from OMA', {
      status: response.status,
      success: data.success,
      mapping_id: data.data?.id
    });

    if (!response.ok) {
      console.error('❌ NETWORK MAPPINGS API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to create network mapping',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ NETWORK MAPPINGS API: Mapping created successfully', {
      mapping_id: data.data?.id
    });

    return NextResponse.json({
      success: true,
      message: data.message,
      mapping: data.data
    });

  } catch (error) {
    console.error('❌ NETWORK MAPPINGS API: Network or processing error', error);
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

export async function DELETE(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const vm_id = searchParams.get('vm_id');
    const source_network_name = searchParams.get('source_network_name');

    if (!vm_id || !source_network_name) {
      return NextResponse.json(
        { 
          success: false, 
          error: 'Missing required parameters',
          message: 'vm_id and source_network_name are required'
        },
        { status: 400 }
      );
    }

    console.log('🗑️ NETWORK MAPPINGS API: Deleting network mapping', {
      vm_id,
      source_network_name,
      timestamp: new Date().toISOString()
    });

    const endpoint = `${OMA_API_BASE}/network-mappings/${vm_id}/${encodeURIComponent(source_network_name)}`;

    console.log('📤 NETWORK MAPPINGS API: Deleting from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 NETWORK MAPPINGS API: Delete response from OMA', {
      status: response.status,
      success: data.success
    });

    if (!response.ok) {
      console.error('❌ NETWORK MAPPINGS API: OMA API delete error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to delete network mapping',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ NETWORK MAPPINGS API: Mapping deleted successfully');

    return NextResponse.json({
      success: true,
      message: data.message
    });

  } catch (error) {
    console.error('❌ NETWORK MAPPINGS API: Network or processing error', error);
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
