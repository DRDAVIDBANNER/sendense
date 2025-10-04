import { NextRequest, NextResponse } from 'next/server';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    console.log('🚀 Starting automated migration workflow via OMA', body);
    
    // Validate required fields
    if (!body.ossea_config_id) {
      console.error('❌ Missing OSSEA config ID in request');
      return NextResponse.json(
        { error: 'OSSEA configuration ID is required' },
        { status: 400 }
      );
    }
    
    // Transform request to OMA automated workflow format
    const omaRequest = {
      source_vm: body.source_vm, // Complete VM data from VMA discovery
      ossea_config_id: body.ossea_config_id, // REQUIRED: Must be provided by GUI (no hardcoded fallback)
      replication_type: body.replication_type || "initial", // Default to full initial sync
      target_network: body.target_network || "", // Optional
      vcenter_host: body.vcenter_host || "quad-vcenter-01.quadris.local", // Default
      datacenter: body.datacenter || "DatabanxDC", // Default
      change_id: body.change_id || "", // Optional for incremental
      previous_change_id: body.previous_change_id || "", // Optional for incremental
      snapshot_id: body.snapshot_id || "", // Optional
      start_replication: body.start_replication // NEW: Pass through start_replication flag
    };

    console.log('📡 Calling OMA automated workflow', omaRequest);
    
    // Call OMA automated workflow instead of VMA direct replication
    const omaResponse = await fetch('http://localhost:8082/api/v1/replications', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent' // Long-lived dev token expires 2035
      },
      body: JSON.stringify(omaRequest),
    });

    const data = await omaResponse.json();
    
    if (omaResponse.ok) {
      // Handle both job creation and context-only responses
      if (data.job_id) {
        // Job creation response (start_replication: true or undefined)
        console.log('✅ OMA automated workflow started successfully', data);
        return NextResponse.json({
          job_id: data.job_id,
          status: data.status,
          progress_percent: data.progress_percent,
          created_volumes: data.created_volumes,
          mounted_volumes: data.mounted_volumes,
          started_at: data.started_at,
          message: data.message
        });
      } else {
        // Context-only response (start_replication: false)
        console.log('✅ VM added to management successfully', data);
        return NextResponse.json({
          context_id: data.context_id,
          vm_name: data.vm_name,
          vmware_vm_id: data.vmware_vm_id,
          current_status: data.current_status,
          message: data.message,
          created_at: data.created_at
        });
      }
    } else {
      console.error('❌ OMA automated workflow failed', data);
      return NextResponse.json(
        { error: data.error || 'Failed to start automated migration workflow' },
        { status: omaResponse.status }
      );
    }
  } catch (error) {
    console.error('❌ Automated Migration Error:', error);
    return NextResponse.json(
      { error: 'Internal server error during automated migration' },
      { status: 500 }
    );
  }
}
