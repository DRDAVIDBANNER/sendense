import { NextRequest, NextResponse } from 'next/server';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    
    // Forward to OMA Enhanced Discovery API
    const omaResponse = await fetch('http://localhost:8082/api/v1/discovery/discover-vms', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
      },
      body: JSON.stringify(body),
    });

    if (!omaResponse.ok) {
      const errorData = await omaResponse.json();
      return NextResponse.json(errorData, { status: omaResponse.status });
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Discovery proxy error:', error);
    return NextResponse.json(
      { error: 'Internal server error during discovery' },
      { status: 500 }
    );
  }
}

