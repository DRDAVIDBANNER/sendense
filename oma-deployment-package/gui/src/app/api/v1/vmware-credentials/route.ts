import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = 'http://localhost:8082/api/v1';

// GET /api/v1/vmware-credentials - List all credentials
export async function GET() {
  try {
    const response = await fetch(`${OMA_API_BASE}/vmware-credentials`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const errorText = await response.text();
      console.error('OMA API error:', response.status, errorText);
      return NextResponse.json(
        { error: `OMA API error: ${errorText}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Failed to fetch VMware credentials:', error);
    return NextResponse.json(
      { error: 'Failed to connect to OMA API service' },
      { status: 500 }
    );
  }
}

// POST /api/v1/vmware-credentials - Create new credentials
export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    
    const response = await fetch(`${OMA_API_BASE}/vmware-credentials`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const errorText = await response.text();
      console.error('OMA API error:', response.status, errorText);
      return NextResponse.json(
        { error: `OMA API error: ${errorText}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Failed to create VMware credentials:', error);
    return NextResponse.json(
      { error: 'Failed to connect to OMA API service' },
      { status: 500 }
    );
  }
}
