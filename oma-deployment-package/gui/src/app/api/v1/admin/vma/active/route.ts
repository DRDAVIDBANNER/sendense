// API route for VMA active connections - proxies to OMA API
// This provides a bridge between Next.js frontend and Go OMA API for VMA enrollment

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const status = searchParams.get('status');
    const limit = searchParams.get('limit');

    console.log('🔗 VMA ACTIVE: Fetching active VMA connections', {
      status, limit,
      timestamp: new Date().toISOString()
    });

    // Build query parameters
    const params = new URLSearchParams();
    if (status) params.append('status', status);
    if (limit) params.append('limit', limit);

    const endpoint = `${OMA_API_BASE}/admin/vma/active?${params.toString()}`;

    console.log('📤 VMA ACTIVE: Requesting active VMAs from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 VMA ACTIVE: Response from OMA', {
      status: response.status,
      success: data.success,
      total_connections: data.connections?.length || 0
    });

    if (!response.ok) {
      console.error('❌ VMA ACTIVE: Failed to get active VMAs', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to get active VMAs',
          connections: []
        },
        { status: response.status }
      );
    }

    console.log('✅ VMA ACTIVE: Active VMAs retrieved successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      total: data.total,
      connections: data.connections || []
    });

  } catch (error) {
    console.error('❌ VMA ACTIVE: Error fetching active VMAs', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API',
        connections: []
      },
      { status: 500 }
    );
  }
}


