import { NextRequest, NextResponse } from 'next/server';

const OMA_API_URL = process.env.OMA_API_URL || 'http://localhost:8082';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    const response = await fetch(`${OMA_API_URL}/api/v1/settings/cloudstack/detect-oma-vm`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch (error) {
    console.error('CloudStack detect OMA VM proxy error:', error);
    return NextResponse.json(
      { 
        success: false, 
        message: 'Failed to detect OMA VM',
        error: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}


