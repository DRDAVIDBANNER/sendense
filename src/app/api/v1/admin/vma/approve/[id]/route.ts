// API route for VMA enrollment approval - proxies to OMA API
// This provides a bridge between Next.js frontend and Go OMA API for VMA enrollment

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function POST(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const { id } = params;
    const body = await request.json();

    console.log('✅ VMA APPROVE: Approving VMA enrollment', {
      enrollmentId: id,
      approvedBy: body.approved_by,
      timestamp: new Date().toISOString()
    });

    // Forward request to OMA API
    const response = await fetch(`${OMA_API_BASE}/admin/vma/approve/${id}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      },
      body: JSON.stringify(body)
    });

    const data = await response.json();

    console.log('📥 VMA APPROVE: Response from OMA', {
      status: response.status,
      success: data.success,
      enrollmentId: id
    });

    if (!response.ok) {
      console.error('❌ VMA APPROVE: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to approve enrollment',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ VMA APPROVE: Enrollment approved successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      status: data.status,
      enrollment_id: id
    });

  } catch (error) {
    console.error('❌ VMA APPROVE: Network or processing error', error);
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


