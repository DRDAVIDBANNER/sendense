import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = process.env.OMA_API_BASE || 'http://localhost:8082';

export async function POST(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const scheduleId = params.id;
    const body = await request.json();
    
    // Forward the request to the OMA API
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/schedules/${scheduleId}/enable`, {
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
        { error: `Failed to enable/disable schedule: ${errorText || omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error enabling/disabling schedule:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}