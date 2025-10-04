// API route for VMA audit log - proxies to OMA API
// This provides a bridge between Next.js frontend and Go OMA API for VMA enrollment

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const eventType = searchParams.get('event_type');
    const enrollmentId = searchParams.get('enrollment_id');
    const limit = searchParams.get('limit');
    const offset = searchParams.get('offset');

    console.log('📊 VMA AUDIT: Fetching VMA audit events', {
      eventType, enrollmentId, limit, offset,
      timestamp: new Date().toISOString()
    });

    // Build query parameters
    const params = new URLSearchParams();
    if (eventType) params.append('event_type', eventType);
    if (enrollmentId) params.append('enrollment_id', enrollmentId);
    if (limit) params.append('limit', limit);
    if (offset) params.append('offset', offset);

    const endpoint = `${OMA_API_BASE}/admin/vma/audit?${params.toString()}`;

    console.log('📤 VMA AUDIT: Requesting audit events from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 VMA AUDIT: Response from OMA', {
      status: response.status,
      success: data.success,
      total_events: data.events?.length || 0
    });

    if (!response.ok) {
      console.error('❌ VMA AUDIT: Failed to get audit events', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to get audit events',
          events: []
        },
        { status: response.status }
      );
    }

    console.log('✅ VMA AUDIT: Audit events retrieved successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      total: data.total,
      events: data.events || []
    });

  } catch (error) {
    console.error('❌ VMA AUDIT: Error fetching audit events', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API',
        events: []
      },
      { status: 500 }
    );
  }
}


