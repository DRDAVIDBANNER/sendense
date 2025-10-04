import { NextResponse } from 'next/server';

const OMA_API_BASE = 'http://localhost:8082/api/v1';

// GET /api/v1/vmware-credentials/default - Get default credentials
export async function GET() {
  try {
    const response = await fetch(`${OMA_API_BASE}/vmware-credentials/default`, {
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
    console.error('Failed to fetch default VMware credentials:', error);
    return NextResponse.json(
      { error: 'Failed to connect to OMA API service' },
      { status: 500 }
    );
  }
}
