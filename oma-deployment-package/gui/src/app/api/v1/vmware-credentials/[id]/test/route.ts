import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = 'http://localhost:8082/api/v1';

// POST /api/v1/vmware-credentials/[id]/test - Test credentials connectivity
export async function POST(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const response = await fetch(`${OMA_API_BASE}/vmware-credentials/${params.id}/test`, {
      method: 'POST',
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
    console.error('Failed to test VMware credentials:', error);
    return NextResponse.json(
      { error: 'Failed to connect to OMA API service' },
      { status: 500 }
    );
  }
}
