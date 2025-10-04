// API route for VMA pairing code generation - proxies to OMA API
// This provides a bridge between Next.js frontend and Go OMA API for VMA enrollment

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    console.log('🔐 VMA PAIRING CODE: Generating new pairing code', {
      timestamp: new Date().toISOString(),
      requestData: body
    });

    // Forward request to OMA API
    const response = await fetch(`${OMA_API_BASE}/admin/vma/pairing-code`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      },
      body: JSON.stringify(body)
    });

    const data = await response.json();

    console.log('📥 VMA PAIRING CODE: Response from OMA', {
      status: response.status,
      success: data.success,
      pairingCode: data.pairing_code ? 'Generated' : 'None'
    });

    if (!response.ok) {
      console.error('❌ VMA PAIRING CODE: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to generate pairing code',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ VMA PAIRING CODE: Pairing code generated successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      pairing_code: data.pairing_code,
      expires_at: data.expires_at,
      valid_for: data.valid_for
    });

  } catch (error) {
    console.error('❌ VMA PAIRING CODE: Network or processing error', error);
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


