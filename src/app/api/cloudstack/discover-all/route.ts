import { NextRequest, NextResponse } from 'next/server';

// POST /api/cloudstack/discover-all
// Combined endpoint that tests connection, detects OMA VM, and discovers all resources
export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    
    // Forward to OMA API
    const omaResponse = await fetch('http://localhost:8082/api/v1/settings/cloudstack/discover-all', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
      },
      body: JSON.stringify(body),
    });

    const data = await omaResponse.json();
    
    if (omaResponse.ok) {
      return NextResponse.json(data);
    } else {
      return NextResponse.json(
        { error: data.error || 'Failed to discover CloudStack resources' },
        { status: omaResponse.status }
      );
    }
  } catch (error) {
    console.error('CloudStack Discover All Error:', error);
    return NextResponse.json(
      { error: 'Internal server error during CloudStack discovery' },
      { status: 500 }
    );
  }
}

