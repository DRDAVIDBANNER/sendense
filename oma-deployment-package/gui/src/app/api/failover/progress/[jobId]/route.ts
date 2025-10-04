// API route for failover progress tracking - proxies to OMA API
// GET /api/failover/progress/{job_id}

import { NextRequest, NextResponse } from 'next/server';

// OMA API base URL - adjust port as needed
const OMA_API_BASE = 'http://localhost:8082/api/v1';

// Helper function to determine current phase from status and step progress
function determineCurrentPhase(status: string, completedSteps: number, totalSteps: number): string {
  if (status === 'completed') return 'Completed';
  if (status === 'failed') return 'Failed';
  if (status === 'pending') return 'Initializing';
  if (status === 'running' || status === 'in_progress') {
    const progress = totalSteps > 0 ? (completedSteps / totalSteps) * 100 : 0;
    if (progress < 20) return 'Starting';
    if (progress < 40) return 'Power Management';
    if (progress < 60) return 'Data Transfer';
    if (progress < 80) return 'VM Creation';
    return 'Finalizing';
  }
  return 'Unknown';
}

export async function GET(
  request: NextRequest,
  { params }: { params: { jobId: string } }
) {
  try {
    const { jobId } = await params;

    console.log('📊 PROGRESS API: Getting failover progress', {
      jobId,
      timestamp: new Date().toISOString()
    });

    // Forward request to OMA API
    const endpoint = `${OMA_API_BASE}/failover/${jobId}/status`;
    
    console.log('📤 PROGRESS API: Requesting from OMA', { endpoint });

    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent',
      }
    });

    const data = await response.json();

    console.log('📥 PROGRESS API: Response from OMA', {
      status: response.status,
      success: data.success,
      job_status: data.status,
      progress: data.progress
    });

    if (!response.ok) {
      console.error('❌ PROGRESS API: OMA API error', data);
      return NextResponse.json(
        { 
          success: false, 
          error: data.error || 'Failed to get progress',
          message: data.message || 'Unknown error occurred'
        },
        { status: response.status }
      );
    }

    console.log('✅ PROGRESS API: Progress retrieved successfully');

    // Transform OMA response to unified progress format
    // OMA API returns: {success, job_id, status, progress, job_details: {completed_steps, total_steps, metadata}}
    const jobDetails = data.job_details || {};
    const metadata = jobDetails.metadata || {};
    
    const progressData = {
      job_id: data.job_id || jobId,
      status: data.status || 'unknown',
      progress: data.progress || 0,
      current_phase: determineCurrentPhase(data.status, jobDetails.completed_steps, jobDetails.total_steps),
      phases: [], // Not using detailed phases for now, just % progress
      estimated_completion: data.estimated_completion,
      elapsed_time: data.elapsed_time,
      metadata: {
        failover_type: metadata.failover_type || 'unknown',
        vm_name: metadata.vm_name || 'Unknown VM',
        configuration_summary: `${jobDetails.completed_steps || 0}/${jobDetails.total_steps || 0} steps completed`
      }
    };

    return NextResponse.json({
      success: true,
      message: data.message || 'Progress retrieved successfully',
      progress: progressData
    });

  } catch (error) {
    console.error('❌ PROGRESS API: Network or processing error', error);
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
