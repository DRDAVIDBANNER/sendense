import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = process.env.OMA_API_BASE || 'http://localhost:8082';

export async function GET(request: NextRequest) {
  try {
    // Forward the request to the OMA API (using discovery endpoint)
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/discovery/ungrouped-vms`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!omaResponse.ok) {
      console.error(`OMA API error: ${omaResponse.status} ${omaResponse.statusText}`);
      return NextResponse.json(
        { error: `Failed to fetch ungrouped VM contexts: ${omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    
    // Transform response to match expected format
    const transformedData = {
      vm_contexts: data.vms || [],
      count: data.count || 0,
      retrieved_at: data.retrieved_at
    };
    
    return NextResponse.json(transformedData);
  } catch (error) {
    console.error('Error fetching ungrouped VM contexts:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
