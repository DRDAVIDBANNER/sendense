// API route for VMA pending enrollments - proxies to OMA API
// This provides a bridge between Next.js frontend and Go OMA API for VMA enrollment

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const status = searchParams.get('status');
    const limit = searchParams.get('limit');

    console.log('📋 VMA PENDING: Fetching pending enrollments', {
      status, limit,
      timestamp: new Date().toISOString()
    });

    // Build query parameters
    const params = new URLSearchParams();
    if (status) params.append('status', status);
    if (limit) params.append('limit', limit);

    const endpoint = `${OMA_API_BASE}/admin/vma/pending?${params.toString()}`;

    console.log('📤 VMA PENDING: Requesting pending enrollments from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 VMA PENDING: Response from OMA', {
      status: response.status,
      success: data.success,
      total_enrollments: data.enrollments?.length || 0
    });

    if (!response.ok) {
      console.error('❌ VMA PENDING: Failed to get pending enrollments', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to get pending enrollments',
          enrollments: []
        },
        { status: response.status }
      );
    }

    console.log('✅ VMA PENDING: Pending enrollments retrieved successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      total: data.total,
      enrollments: data.enrollments || []
    });

  } catch (error) {
    console.error('❌ VMA PENDING: Error fetching pending enrollments', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API',
        enrollments: []
      },
      { status: 500 }
    );
  }
}


