import { NextResponse } from 'next/server';
import { execSync } from 'child_process';

export async function GET() {
  try {
    console.log('📊 Fetching historical analytics data');
    
    // Query 1: Migration trends over time (all available data, last 90 days max)
    const migrationTrendsQuery = `mysql -u oma_user -p'oma_password' migratekit_oma --silent -e "
      SELECT 
        DATE(created_at) as date,
        COUNT(*) as total_jobs,
        SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
        SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
        SUM(CASE WHEN status = 'replicating' THEN 1 ELSE 0 END) as active,
        AVG(progress_percent) as avg_progress
      FROM replication_jobs 
      WHERE created_at >= DATE_SUB(NOW(), INTERVAL 90 DAY)
      GROUP BY DATE(created_at)
      ORDER BY date DESC
      LIMIT 30
    " 2>/dev/null`;

    // Query 2: Performance metrics (use all completed jobs since throughput data may be sparse)
    const performanceQuery = `mysql -u oma_user -p'oma_password' migratekit_oma --silent -e "
      SELECT 
        AVG(CASE WHEN vma_throughput_mbps > 0 THEN vma_throughput_mbps ELSE NULL END) as avg_throughput,
        MAX(vma_throughput_mbps) as max_throughput,
        AVG(CASE 
          WHEN completed_at IS NOT NULL AND started_at IS NOT NULL 
          THEN TIMESTAMPDIFF(SECOND, started_at, completed_at) 
          ELSE NULL 
        END) as avg_completion_time_seconds,
        AVG(bytes_transferred / (1024*1024*1024)) as avg_data_transferred_gb,
        SUM(bytes_transferred) / (1024*1024*1024) as total_data_transferred_gb
      FROM replication_jobs 
      WHERE status = 'completed'
    " 2>/dev/null`;

    // Query 3: Success rates by VM type
    const successRatesQuery = `mysql -u oma_user -p'oma_password' migratekit_oma --silent -e "
      SELECT 
        vd.os_type,
        COUNT(*) as total_jobs,
        SUM(CASE WHEN rj.status = 'completed' THEN 1 ELSE 0 END) as successful,
        ROUND((SUM(CASE WHEN rj.status = 'completed' THEN 1 ELSE 0 END) / COUNT(*)) * 100, 2) as success_rate
      FROM replication_jobs rj
      LEFT JOIN vm_disks vd ON rj.id = vd.job_id
      WHERE rj.created_at >= DATE_SUB(NOW(), INTERVAL 90 DAY)
      GROUP BY vd.os_type
      HAVING COUNT(*) > 0
      ORDER BY success_rate DESC
    " 2>/dev/null`;

    // Query 4: CBT efficiency metrics (use correct field names)
    const cbtEfficiencyQuery = `mysql -u oma_user -p'oma_password' migratekit_oma --silent -e "
      SELECT 
        ch.sync_type as operation_type,
        COUNT(*) as operations,
        AVG(ch.bytes_transferred / (1024*1024)) as avg_changes_mb,
        SUM(ch.bytes_transferred) / (1024*1024*1024) as total_changes_gb
      FROM cbt_history ch
      WHERE ch.created_at IS NOT NULL
      GROUP BY ch.sync_type
      ORDER BY operations DESC
    " 2>/dev/null`;

    // Query 5: Volume operation metrics (use correct field name 'type')
    const volumeMetricsQuery = `mysql -u oma_user -p'oma_password' migratekit_oma --silent -e "
      SELECT 
        type as operation_type,
        COUNT(*) as operations,
        AVG(CASE 
          WHEN completed_at IS NOT NULL AND created_at IS NOT NULL 
          THEN TIMESTAMPDIFF(SECOND, created_at, completed_at) 
          ELSE NULL 
        END) as avg_duration_seconds,
        SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as successful_ops
      FROM volume_operations
      WHERE created_at IS NOT NULL
      GROUP BY type
      ORDER BY operations DESC
      LIMIT 10
    " 2>/dev/null`;

    // Execute all queries with error handling
    let migrationTrends = '';
    let performance = '';
    let successRates = '';
    let cbtEfficiency = '';
    let volumeMetrics = '';

    try {
      migrationTrends = execSync(migrationTrendsQuery, { encoding: 'utf8' });
    } catch (error) {
      console.warn('Migration trends query failed:', error);
    }

    try {
      performance = execSync(performanceQuery, { encoding: 'utf8' });
    } catch (error) {
      console.warn('Performance query failed:', error);
    }

    try {
      successRates = execSync(successRatesQuery, { encoding: 'utf8' });
    } catch (error) {
      console.warn('Success rates query failed:', error);
    }

    try {
      cbtEfficiency = execSync(cbtEfficiencyQuery, { encoding: 'utf8' });
    } catch (error) {
      console.warn('CBT efficiency query failed:', error);
    }

    try {
      volumeMetrics = execSync(volumeMetricsQuery, { encoding: 'utf8' });
    } catch (error) {
      console.warn('Volume metrics query failed:', error);
    }

    // Parse migration trends
    const trendsLines = migrationTrends.trim().split('\n').filter(line => line.trim());
    const trendsData = trendsLines.map(line => {
      const fields = line.split('\t');
      return {
        date: fields[0],
        total_jobs: parseInt(fields[1]) || 0,
        completed: parseInt(fields[2]) || 0,
        failed: parseInt(fields[3]) || 0,
        active: parseInt(fields[4]) || 0,
        avg_progress: parseFloat(fields[5]) || 0
      };
    });

    // Parse performance metrics
    const perfLines = performance.trim().split('\n').filter(line => line.trim());
    const perfData = perfLines.length > 0 ? (() => {
      const fields = perfLines[0].split('\t');
      return {
        avg_throughput_mbps: parseFloat(fields[0]) || 0,
        max_throughput_mbps: parseFloat(fields[1]) || 0,
        avg_completion_time_seconds: parseFloat(fields[2]) || 0,
        avg_data_transferred_gb: parseFloat(fields[3]) || 0,
        total_data_transferred_gb: parseFloat(fields[4]) || 0
      };
    })() : null;

    // Parse success rates
    const successLines = successRates.trim().split('\n').filter(line => line.trim());
    const successData = successLines.map(line => {
      const fields = line.split('\t');
      return {
        os_type: fields[0] || 'Unknown',
        total_jobs: parseInt(fields[1]) || 0,
        successful: parseInt(fields[2]) || 0,
        success_rate: parseFloat(fields[3]) || 0
      };
    });

    // Parse CBT efficiency
    const cbtLines = cbtEfficiency.trim().split('\n').filter(line => line.trim());
    const cbtData = cbtLines.map(line => {
      const fields = line.split('\t');
      return {
        operation_type: fields[0] || 'Unknown',
        operations: parseInt(fields[1]) || 0,
        avg_changes_mb: parseFloat(fields[2]) || 0,
        total_changes_gb: parseFloat(fields[3]) || 0
      };
    });

    // Parse volume metrics
    const volumeLines = volumeMetrics.trim().split('\n').filter(line => line.trim());
    const volumeData = volumeLines.map(line => {
      const fields = line.split('\t');
      return {
        operation_type: fields[0] || 'Unknown',
        operations: parseInt(fields[1]) || 0,
        avg_duration_seconds: parseFloat(fields[2]) || 0,
        successful_ops: parseInt(fields[3]) || 0
      };
    });

    const analyticsData = {
      migration_trends: trendsData,
      performance_metrics: perfData,
      success_rates: successData,
      cbt_efficiency: cbtData,
      volume_metrics: volumeData,
      generated_at: new Date().toISOString()
    };

    console.log(`✅ Retrieved historical analytics data: ${trendsData.length} trend points, ${successData.length} OS types`);
    return NextResponse.json(analyticsData);
  } catch (error) {
    console.error('❌ Error fetching historical analytics:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
