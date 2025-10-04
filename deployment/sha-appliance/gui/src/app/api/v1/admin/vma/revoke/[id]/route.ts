// API route for VMA access revocation - proxies to OMA API
// This provides a bridge between Next.js frontend and Go OMA API for VMA enrollment

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function DELETE(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const { id } = params;
    const { searchParams } = new URL(request.url);
    const revokedBy = searchParams.get('revoked_by') || 'admin';

    console.log('🔐 VMA REVOKE: Revoking VMA access', {
      enrollmentId: id,
      revokedBy,
      timestamp: new Date().toISOString()
    });

    // Forward request to OMA API
    const response = await fetch(`${OMA_API_BASE}/admin/vma/revoke/${id}?revoked_by=${encodeURIComponent(revokedBy)}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 VMA REVOKE: Response from OMA', {
      status: response.status,
      success: data.success,
      enrollmentId: id
    });

    if (!response.ok) {
      console.error('❌ VMA REVOKE: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to revoke VMA access',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ VMA REVOKE: VMA access revoked successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      enrollment_id: id,
      revoked_by: revokedBy
    });

  } catch (error) {
    console.error('❌ VMA REVOKE: Network or processing error', error);
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


