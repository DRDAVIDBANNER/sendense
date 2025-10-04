import { NextRequest, NextResponse } from 'next/server';

export async function GET(_request: NextRequest) {
  try {
    console.log('📊 Fetching migration jobs from database');
    
    // Try OMA API first for active jobs
    let omaJobs: unknown[] = [];
    try {
      const omaResponse = await fetch('http://localhost:8082/api/v1/replications', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
        }
      });
      
      if (omaResponse.ok) {
        omaJobs = await omaResponse.json();
        console.log(`📡 OMA API returned ${omaJobs.length} active jobs`);
      } else {
        console.log(`⚠️ OMA API returned ${omaResponse.status}, falling back to database query`);
      }
    } catch (error) {
      console.log('⚠️ OMA API not available, using database query:', error);
    }

    // Query database for recent jobs (last 24 hours) using exec
    // Include jobs in transition states (created within last 15 seconds) to avoid grace period issues
    const { execSync } = await import('child_process');
    const dbCommand = `mysql -u oma_user -p'oma_password' migratekit_oma --silent -e "SELECT id, source_vm_name, status, progress_percent, current_operation, bytes_transferred, total_bytes, transfer_speed_bps, vma_throughput_mbps, vma_eta_seconds, replication_type, created_at, started_at, completed_at, updated_at, error_message FROM replication_jobs WHERE updated_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR) ORDER BY updated_at DESC LIMIT 20" 2>/dev/null`;
    
    let dbJobs: unknown[] = [];
    try {
      const dbOutput = execSync(dbCommand, { encoding: 'utf8' });
      const lines = dbOutput.trim().split('\n').filter(line => line.trim());
      
      dbJobs = lines.map(line => {
        const fields = line.split('\t');
        if (fields.length >= 16) {
          return {
            id: fields[0],
            vm_name: fields[1], 
            status: fields[2],
            progress_percent: parseFloat(fields[3]) || 0,
            current_operation: fields[4] || fields[2],
            bytes_transferred: parseInt(fields[5]) || 0,
            total_bytes: parseInt(fields[6]) || 0,
            transfer_speed_bps: parseInt(fields[7]) || 0,
            vma_throughput_mbps: parseFloat(fields[8]) || 0,
            vma_eta_seconds: fields[9] !== 'NULL' ? parseInt(fields[9]) : null,
            replication_type: fields[10],
            created_at: fields[11],
            started_at: fields[12] !== 'NULL' ? fields[12] : null,
            completed_at: fields[13] !== 'NULL' ? fields[13] : null,
            updated_at: fields[14],
            error_message: fields[15] !== 'NULL' ? fields[15] : null
          };
        }
        return null;
      }).filter(job => job !== null);
    } catch (error) {
      console.error('Database query failed:', error);
    }

    // Combine OMA active jobs with recent database jobs
    const allJobs = [...omaJobs, ...dbJobs];
    
    // Remove duplicates (prefer OMA data for active jobs)
    const uniqueJobs = allJobs.reduce((acc: unknown[], job: unknown) => {
      const existingIndex = acc.findIndex(j => j.id === job.id);
      if (existingIndex === -1) {
        acc.push(job);
      } else if (omaJobs.some(omaJob => omaJob.id === job.id)) {
        // Prefer OMA data for active jobs
        acc[existingIndex] = job;
      }
      return acc;
    }, []);

    console.log(`✅ Retrieved ${uniqueJobs.length} migration jobs (${omaJobs.length} active + ${dbJobs.length} recent)`);
    return NextResponse.json(uniqueJobs);
  } catch (error) {
    console.error('❌ Error fetching migrations:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}