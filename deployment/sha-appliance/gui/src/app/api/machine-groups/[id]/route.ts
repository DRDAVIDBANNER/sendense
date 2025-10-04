import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = process.env.OMA_API_BASE || 'http://localhost:8082';

export async function GET(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const groupId = params.id;
    
    // Forward the request to the OMA API
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/machine-groups/${groupId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!omaResponse.ok) {
      console.error(`OMA API error: ${omaResponse.status} ${omaResponse.statusText}`);
      return NextResponse.json(
        { error: `Failed to fetch machine group: ${omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching machine group:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

export async function PUT(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const groupId = params.id;
    const body = await request.json();
    
    // Forward the request to the OMA API
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/machine-groups/${groupId}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!omaResponse.ok) {
      console.error(`OMA API error: ${omaResponse.status} ${omaResponse.statusText}`);
      const errorText = await omaResponse.text();
      return NextResponse.json(
        { error: `Failed to update machine group: ${errorText || omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error updating machine group:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const groupId = params.id;
    
    // Forward the request to the OMA API
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/machine-groups/${groupId}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!omaResponse.ok) {
      console.error(`OMA API error: ${omaResponse.status} ${omaResponse.statusText}`);
      const errorText = await omaResponse.text();
      return NextResponse.json(
        { error: `Failed to delete machine group: ${errorText || omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error deleting machine group:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

