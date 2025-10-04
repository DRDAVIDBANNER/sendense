import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = process.env.OMA_API_BASE || 'http://localhost:8082';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    
    // Forward request to OMA API
    const response = await fetch(`${OMA_API_BASE}/api/v1/ossea/discover-resources`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const errorText = await response.text();
      return NextResponse.json(
        { error: errorText },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('OSSEA resource discovery proxy error:', error);
    return NextResponse.json(
      { error: 'Failed to discover resources' },
      { status: 500 }
    );
  }
}






