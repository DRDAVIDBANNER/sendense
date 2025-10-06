/**
 * Discovery API Proxy Route
 * 
 * Custom proxy for discovery endpoints that can take 15-30 seconds
 * (vCenter queries for 98+ VMs). Next.js rewrites have limited timeout
 * control, so we use custom API routes with explicit fetch timeouts.
 * 
 * Handles:
 * - POST /api/v1/discovery/discover-vms (15-30s)
 * - POST /api/v1/discovery/add-vms (15-30s)
 * - GET /api/v1/discovery/ungrouped-vms (fast)
 * - POST /api/v1/discovery/preview (15-30s)
 */

import { NextRequest, NextResponse } from 'next/server';

const BACKEND_URL = 'http://localhost:8082';
const DISCOVERY_TIMEOUT = 60000; // 60 seconds for vCenter operations

export async function GET(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  const path = params.path.join('/');
  const url = `${BACKEND_URL}/api/v1/discovery/${path}`;
  
  try {
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      signal: AbortSignal.timeout(DISCOVERY_TIMEOUT),
    });

    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch (error: any) {
    if (error.name === 'TimeoutError' || error.name === 'AbortError') {
      return NextResponse.json(
        { error: 'Discovery operation timed out after 60 seconds' },
        { status: 504 }
      );
    }
    
    console.error(`Discovery API error [GET /${path}]:`, error);
    return NextResponse.json(
      { error: error.message || 'Discovery API request failed' },
      { status: 500 }
    );
  }
}

export async function POST(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  const path = params.path.join('/');
  const url = `${BACKEND_URL}/api/v1/discovery/${path}`;
  
  try {
    const body = await request.json();
    
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
      signal: AbortSignal.timeout(DISCOVERY_TIMEOUT),
    });

    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch (error: any) {
    if (error.name === 'TimeoutError' || error.name === 'AbortError') {
      return NextResponse.json(
        { error: 'Discovery operation timed out after 60 seconds' },
        { status: 504 }
      );
    }
    
    console.error(`Discovery API error [POST /${path}]:`, error);
    return NextResponse.json(
      { error: error.message || 'Discovery API request failed' },
      { status: 500 }
    );
  }
}

// Set custom runtime config for this route
export const runtime = 'nodejs'; // Use Node.js runtime (not edge)
export const dynamic = 'force-dynamic'; // Always run dynamically, never static
export const maxDuration = 60; // Maximum execution time in seconds

