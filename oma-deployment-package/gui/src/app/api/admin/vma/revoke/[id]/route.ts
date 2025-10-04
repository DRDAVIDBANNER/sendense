import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = 'http://localhost:8082';

interface RouteParams {
  params: {
    id: string;
  };
}

export async function DELETE(request: NextRequest, { params }: RouteParams) {
  try {
    // Get authorization header from request
    const authHeader = request.headers.get('authorization');
    if (!authHeader) {
      return NextResponse.json({ error: 'Authorization header required' }, { status: 401 });
    }

    // Get enrollment ID from route params
    const { id } = params;

    // Forward request to OMA API server
    const response = await fetch(`${OMA_API_BASE}/api/v1/admin/vma/revoke/${id}`, {
      method: 'DELETE',
      headers: {
        'Authorization': authHeader,
      },
    });

    // Get response data
    const data = await response.text();

    // Return response with same status and content type
    return new NextResponse(data, {
      status: response.status,
      headers: {
        'Content-Type': response.headers.get('content-type') || 'application/json',
      },
    });

  } catch (error) {
    console.error('VMA access revocation failed:', error);
    return NextResponse.json(
      { error: 'Failed to revoke VMA access' },
      { status: 500 }
    );
  }
}

export async function OPTIONS() {
  return new NextResponse(null, {
    status: 200,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'DELETE, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    },
  });
}


