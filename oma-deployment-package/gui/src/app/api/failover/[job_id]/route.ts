// API route for individual failover job operations
// Handles job status, test failover cleanup, etc.

import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = 'http://localhost:8082/api/v1';

export async function GET(
  request: NextRequest,
  { params }: { params: { job_id: string } }
) {
  try {
    const { job_id } = await params;

    console.log('📊 FAILOVER STATUS API: Getting job status', {
      job_id,
      timestamp: new Date().toISOString()
    });

    const endpoint = `${OMA_API_BASE}/failover/${job_id}/status`;

    console.log('📤 FAILOVER STATUS API: Requesting status from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      }
    });

    const data = await response.json();

    console.log('📥 FAILOVER STATUS API: Status response from OMA', {
      job_id,
      status: response.status,
      success: data.success,
      job_status: data.status,
      progress: data.progress
    });

    if (!response.ok) {
      console.error('❌ FAILOVER STATUS API: Failed to get status', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to get job status'
        },
        { status: response.status }
      );
    }

    console.log('✅ FAILOVER STATUS API: Status retrieved successfully');

    return NextResponse.json({
      success: true,
      message: data.message,
      job_id: data.job_id,
      status: data.status,
      progress: data.progress,
      start_time: data.start_time,
      duration: data.duration,
      job_details: data.job_details
    });

  } catch (error) {
    console.error('❌ FAILOVER STATUS API: Error getting status', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API'
      },
      { status: 500 }
    );
  }
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: { job_id: string } }
) {
  try {
    const { job_id } = params;

    console.log('🧹 FAILOVER CLEANUP API: Ending test failover', {
      job_id,
      timestamp: new Date().toISOString()
    });

    const endpoint = `${OMA_API_BASE}/failover/test/${job_id}`;

    console.log('📤 FAILOVER CLEANUP API: Requesting cleanup from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      }
    });

    const data = await response.json();

    console.log('📥 FAILOVER CLEANUP API: Cleanup response from OMA', {
      job_id,
      status: response.status,
      success: data.success,
      message: data.message
    });

    if (!response.ok) {
      console.error('❌ FAILOVER CLEANUP API: Failed to cleanup', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to cleanup test failover'
        },
        { status: response.status }
      );
    }

    console.log('✅ FAILOVER CLEANUP API: Test failover cleanup completed');

    return NextResponse.json({
      success: true,
      message: data.message,
      job_id: data.job_id,
      data: data.data
    });

  } catch (error) {
    console.error('❌ FAILOVER CLEANUP API: Error during cleanup', error);
    return NextResponse.json(
      { 
        success: false, 
        error: 'Network error',
        message: 'Failed to communicate with OMA API'
      },
      { status: 500 }
    );
  }
}




