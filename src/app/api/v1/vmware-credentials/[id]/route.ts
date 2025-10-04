import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = 'http://localhost:8082/api/v1';

// GET /api/v1/vmware-credentials/[id] - Get specific credentials
export async function GET(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const response = await fetch(`${OMA_API_BASE}/vmware-credentials/${params.id}`, {
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

// PUT /api/v1/vmware-credentials/[id] - Update credentials
export async function PUT(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const body = await request.json();
    
    const response = await fetch(`${OMA_API_BASE}/vmware-credentials/${params.id}`, {
      method: 'PUT',
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
    console.error('Failed to update VMware credentials:', error);
    return NextResponse.json(
      { error: 'Failed to connect to OMA API service' },
      { status: 500 }
    );
  }
}

// DELETE /api/v1/vmware-credentials/[id] - Delete credentials
export async function DELETE(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const response = await fetch(`${OMA_API_BASE}/vmware-credentials/${params.id}`, {
      method: 'DELETE',
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
    console.error('Failed to delete VMware credentials:', error);
    return NextResponse.json(
      { error: 'Failed to connect to OMA API service' },
      { status: 500 }
    );
  }
}
