import { NextRequest, NextResponse } from 'next/server';

const OMA_API_URL = process.env.OMA_API_URL || 'http://localhost:8082';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    const response = await fetch(`${OMA_API_URL}/api/v1/settings/cloudstack/validate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch (error) {
    console.error('CloudStack validation proxy error:', error);
    return NextResponse.json(
      { 
        success: false, 
        message: 'Failed to validate CloudStack settings',
        result: {
          oma_vm_detection: { status: 'fail', message: 'Validation failed' },
          compute_offering: { status: 'fail', message: 'Validation failed' },
          account_match: { status: 'fail', message: 'Validation failed' },
          network_selection: { status: 'fail', message: 'Validation failed' },
          overall_status: 'fail' as const
        },
        error: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}


