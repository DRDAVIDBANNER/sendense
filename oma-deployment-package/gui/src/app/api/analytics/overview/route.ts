import { NextResponse } from 'next/server';

interface SystemHealthMetrics {
  cpu_cores: number;
  memory_usage_mb: number;
  memory_total_mb: number;
  uptime_hours: number;
  gc_runs: number;
  goroutines: number;
}

interface AnalyticsOverview {
  system_health: SystemHealthMetrics;
  vm_summary: {
    total_vms: number;
    active_jobs: number;
    completed_jobs: number;
    failed_jobs: number;
  };
  performance_summary: {
    average_migration_speed_mbps: number;
    total_data_migrated_gb: number;
    success_rate_percent: number;
    average_completion_time_hours: number;
  };
  recent_activity: {
    last_migration: string | null;
    migrations_last_24h: number;
    system_alerts: number;
  };
}

export async function GET(): Promise<NextResponse> {
  try {
    // Fetch system health from OMA API
    const healthResponse = await fetch('http://localhost:8082/api/v1/debug/health');
    
    if (!healthResponse.ok) {
      throw new Error('Failed to fetch system health data');
    }
    
    const healthData = await healthResponse.json();

    // Fetch VM contexts
    const vmResponse = await fetch('http://localhost:8082/api/v1/vm-contexts');
    
    if (!vmResponse.ok) {
      throw new Error('Failed to fetch VM context data');
    }
    
    const vmResponseData = await vmResponse.json();
    const vmData = vmResponseData.vm_contexts || [];

    // Fetch replication jobs
    const replicationResponse = await fetch('http://localhost:8082/api/v1/replications');
    
    if (!replicationResponse.ok) {
      throw new Error('Failed to fetch replication data');
    }
    
    const replicationData = await replicationResponse.json();

    // Extract system health metrics
    const systemInfo = healthData.system_info;
    const debugData = healthData.debug_data;
    
    // Parse uptime (e.g., "24h0m0.000000301s")
    const uptimeString = systemInfo.uptime;
    const uptimeHours = parseFloat(uptimeString.replace(/h.*/, '')) || 0;

    // Calculate memory usage
    const memoryUsageMB = Math.round(systemInfo.memory_stats.HeapAlloc / 1024 / 1024);
    const memoryTotalMB = Math.round(systemInfo.memory_stats.Sys / 1024 / 1024);

    // Analyze VM data
    const totalVMs = Array.isArray(vmData) ? vmData.length : 0;
    
    // Count jobs by status using current_status field
    let activeJobs = 0;
    let completedJobs = 0;
    let failedJobs = 0;

    if (Array.isArray(vmData)) {
      vmData.forEach((vm: { current_status?: string }) => {
        const status = vm.current_status?.toLowerCase() || 'unknown';
        if (status === 'replicating' || status === 'in_progress' || status === 'syncing') {
          activeJobs++;
        } else if (status === 'completed' || status === 'success' || status === 'idle') {
          completedJobs++;
        } else if (status === 'failed' || status === 'error') {
          failedJobs++;
        }
      });
    }

    // Calculate performance metrics (using available data)
    const totalReplications = Array.isArray(replicationData) ? replicationData.length : 0;
    const successRate = totalReplications > 0 ? 
      ((completedJobs / (completedJobs + failedJobs)) * 100) || 0 : 0;

    // Prepare analytics overview
    const analyticsOverview: AnalyticsOverview = {
      system_health: {
        cpu_cores: debugData.cpu_cores || 0,
        memory_usage_mb: memoryUsageMB,
        memory_total_mb: memoryTotalMB,
        uptime_hours: uptimeHours,
        gc_runs: debugData.gc_runs || 0,
        goroutines: debugData.goroutines || 0
      },
      vm_summary: {
        total_vms: totalVMs,
        active_jobs: activeJobs,
        completed_jobs: completedJobs,
        failed_jobs: failedJobs
      },
      performance_summary: {
        average_migration_speed_mbps: 0, // Will be calculated when historical data is available
        total_data_migrated_gb: 0, // Will be calculated when historical data is available
        success_rate_percent: Math.round(successRate * 100) / 100,
        average_completion_time_hours: 0 // Will be calculated when historical data is available
      },
      recent_activity: {
        last_migration: null, // Will be populated when historical data is available
        migrations_last_24h: 0,
        system_alerts: 0
      }
    };

    return NextResponse.json({
      success: true,
      message: 'Analytics overview retrieved successfully',
      data: analyticsOverview,
      timestamp: new Date().toISOString()
    });

  } catch (error) {
    console.error('Analytics overview error:', error);
    
    return NextResponse.json(
      {
        success: false,
        error: 'Failed to retrieve analytics overview',
        message: error instanceof Error ? error.message : 'Unknown error occurred'
      },
      { status: 500 }
    );
  }
}
