import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = process.env.OMA_API_BASE || 'http://localhost:8082';

export async function GET(_request: NextRequest) {
  try {
    // Forward the request to the OMA API
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/machine-groups`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!omaResponse.ok) {
      console.error(`OMA API error: ${omaResponse.status} ${omaResponse.statusText}`);
      return NextResponse.json(
        { error: `Failed to fetch machine groups: ${omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error fetching machine groups:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    
    // Forward the request to the OMA API
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/machine-groups`, {
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
        { error: `Failed to create machine group: ${errorText || omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error creating machine group:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

