import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = process.env.OMA_API_BASE || 'http://localhost:8082';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id: groupId } = await params;
    
    // Forward the request to the OMA API
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/machine-groups/${groupId}/vms`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!omaResponse.ok) {
      console.error(`OMA API error: ${omaResponse.status} ${omaResponse.statusText}`);
      return NextResponse.json(
        { error: `Failed to fetch group VMs: ${omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching group VMs:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id: groupId } = await params;
    const body = await request.json();
    
    // Forward the request to the OMA API
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/machine-groups/${groupId}/vms`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!omaResponse.ok) {
      console.error(`OMA API error: ${omaResponse.status} ${omaResponse.statusText}`);
      const errorText = await omaResponse.text();
      return NextResponse.json(
        { error: `Failed to assign VM to group: ${errorText || omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error assigning VM to group:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

