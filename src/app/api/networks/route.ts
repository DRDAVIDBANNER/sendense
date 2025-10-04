// API route for network discovery - proxies to OMA API network endpoints
// This provides a bridge between Next.js frontend and Go OMA API for network management

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function GET(_request: NextRequest) {
  try {
    console.log('🌐 NETWORKS API: Fetching available OSSEA networks', {
      timestamp: new Date().toISOString()
    });

    // Forward request to OMA API networks endpoint
    const response = await fetch(`${OMA_API_BASE}/networks/available`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 NETWORKS API: Response from OMA', {
      status: response.status,
      success: data.success,
      network_count: data.data?.length || 0
    });

    if (!response.ok) {
      console.error('❌ NETWORKS API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to fetch networks',
          message: data.message || 'Unknown error occurred',
          networks: []
        },
        { status: response.status }
      );
    }

    console.log('✅ NETWORKS API: Networks retrieved successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      networks: data.data || [],
      total: data.data?.length || 0
    });

  } catch (error) {
    console.error('❌ NETWORKS API: Network or processing error', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API',
        networks: []
      },
      { status: 500 }
    );
  }
}
