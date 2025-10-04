import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = 'http://localhost:8082';

export async function GET(request: NextRequest) {
  try {
    // Get authorization header from request
    const authHeader = request.headers.get('authorization');
    if (!authHeader) {
      return NextResponse.json({ error: 'Authorization header required' }, { status: 401 });
    }

    // Forward request to OMA API server with query parameters
    const url = new URL(request.url);
    const queryParams = url.searchParams.toString();
    const apiUrl = `${OMA_API_BASE}/api/v1/admin/vma/audit${queryParams ? `?${queryParams}` : ''}`;

    const response = await fetch(apiUrl, {
      method: 'GET',
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
    console.error('Failed to fetch VMA audit log:', error);
    return NextResponse.json(
      { error: 'Failed to fetch audit log' },
      { status: 500 }
    );
  }
}

export async function OPTIONS() {
  return new NextResponse(null, {
    status: 200,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    },
  });
}


