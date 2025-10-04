// VM Contexts API Route - List all VM contexts
// Proxies to OMA API /api/v1/vm-contexts

import { NextRequest, NextResponse } from 'next/server';

export async function GET(request: NextRequest) {
  try {
    // Proxy to OMA API
    const omaResponse = await fetch('http://localhost:8082/api/v1/vm-contexts', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        // TODO: Add authentication headers when implemented
      },
    });

    if (!omaResponse.ok) {
      console.error('OMA API Error:', omaResponse.status, omaResponse.statusText);
      return NextResponse.json(
        { error: 'Failed to fetch VM contexts from OMA API' },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();
    const vmContexts = data.vm_contexts || [];
    
    // For each VM context, fetch the current job progress if there's an active job
    const transformedData = await Promise.all(vmContexts.map(async (vm: any) => {
      let progressData = null;
      
      if (vm.current_job_id && vm.current_status === 'replicating') {
        try {
          // Fetch current job progress from replication jobs
          const progressResponse = await fetch(`http://localhost:8082/api/v1/replications/${vm.current_job_id}`);
          if (progressResponse.ok) {
            progressData = await progressResponse.json();
          }
        } catch (error) {
          console.warn(`Failed to fetch progress for job ${vm.current_job_id}:`, error);
        }
      }
      
      return {
        vm_name: vm.vm_name,
        status: vm.current_status || 'unknown',
        job_count: vm.total_jobs_run || 0,
        last_activity: vm.last_job_at || vm.updated_at || 'N/A',
        progress_percentage: progressData?.progress_percent || 0,
        current_job: vm.current_job_id ? {
          id: vm.current_job_id,
          source_vm_name: vm.vm_name,
          status: progressData?.status || vm.current_status || 'unknown',
          replication_type: progressData?.replication_type || 'unknown',
          current_operation: progressData?.current_operation || vm.current_status || 'unknown',
          progress_percentage: progressData?.progress_percent || 0,
          vma_sync_type: progressData?.vma_sync_type || 'unknown',
          vma_eta_seconds: progressData?.vma_eta_seconds || 0,
          created_at: progressData?.created_at || vm.last_job_at || vm.updated_at || new Date().toISOString()
        } : undefined
      };
    }));
    
    return NextResponse.json(transformedData);
  } catch (error) {
    console.error('VM Contexts API Error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
