import { NextRequest, NextResponse } from 'next/server';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ vmName: string }> }
) {
  try {
    const { vmName } = await params;
    
    // First get the context_id from the VM name
    const vmContextResponse = await fetch(
      `http://localhost:8082/api/v1/vm-contexts/${vmName}`,
      {
        headers: {
          'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
          'Content-Type': 'application/json'
        }
      }
    );

    if (!vmContextResponse.ok) {
      return NextResponse.json(
        { error: 'Failed to fetch VM context' },
        { status: vmContextResponse.status }
      );
    }

    const vmContextData = await vmContextResponse.json();
    const contextId = vmContextData.context?.context_id;

    if (!contextId) {
      return NextResponse.json(
        { error: 'No context ID found for VM' },
        { status: 404 }
      );
    }

    // Now fetch recent jobs using the context_id
    const response = await fetch(
      `http://localhost:8082/api/v1/vm-contexts/${contextId}/recent-jobs`,
      {
        headers: {
          'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
          'Content-Type': 'application/json'
        }
      }
    );

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to fetch recent jobs' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Recent jobs API error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}


