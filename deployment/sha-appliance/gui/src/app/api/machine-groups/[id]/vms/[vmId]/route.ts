import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = process.env.OMA_API_BASE || 'http://localhost:8082';

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ id: string; vmId: string }> }
) {
  try {
    const { id: groupId, vmId } = await params;
    
    // Forward the request to the OMA API
    const omaResponse = await fetch(`${OMA_API_BASE}/api/v1/machine-groups/${groupId}/vms/${vmId}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!omaResponse.ok) {
      console.error(`OMA API error: ${omaResponse.status} ${omaResponse.statusText}`);
      const errorText = await omaResponse.text();
      return NextResponse.json(
        { error: `Failed to remove VM from group: ${errorText || omaResponse.statusText}` },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Error removing VM from group:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

