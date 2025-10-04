import { NextRequest } from 'next/server';

// Server-Sent Events implementation for real-time updates
export async function GET(request: NextRequest) {

  // Set up SSE headers
  const encoder = new TextEncoder();
  const stream = new ReadableStream({
    start(controller) {
      // Send initial connection message
      const data = `data: ${JSON.stringify({
        type: 'connection',
        message: 'Real-time monitoring connected',
        timestamp: new Date().toISOString()
      })}\n\n`;
      controller.enqueue(encoder.encode(data));

      // Set up interval for live updates
      const interval = setInterval(async () => {
        try {
          // Fetch current system status
          const systemData = await fetchSystemStatus();
          
          const message = `data: ${JSON.stringify({
            type: 'system_update',
            data: systemData,
            timestamp: new Date().toISOString()
          })}\n\n`;
          
          controller.enqueue(encoder.encode(message));
        } catch (error) {
          console.error('SSE update error:', error);
        }
      }, 5000); // Update every 5 seconds

      // Cleanup on close
      request.signal.addEventListener('abort', () => {
        clearInterval(interval);
        controller.close();
      });
    },
  });

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
    },
  });
}

async function fetchSystemStatus() {
  try {
    const { execSync } = require('child_process');
    
    // Get active migration progress
    const progressQuery = `mysql -u oma_user -p'oma_password' migratekit_oma --silent -e "
      SELECT 
        id,
        source_vm_name,
        status,
        progress_percent,
        current_operation,
        bytes_transferred,
        total_bytes,
        vma_throughput_mbps,
        updated_at
      FROM replication_jobs 
      WHERE status IN ('replicating', 'pending', 'validating')
      ORDER BY updated_at DESC
      LIMIT 10
    " 2>/dev/null`;

    const progressOutput = execSync(progressQuery, { encoding: 'utf8' });
    const progressLines = progressOutput.trim().split('\n').filter(line => line.trim());
    
    const activeJobs = progressLines.map(line => {
      const fields = line.split('\t');
      if (fields.length >= 8) {
        return {
          id: fields[0],
          vm_name: fields[1],
          status: fields[2],
          progress_percent: parseFloat(fields[3]) || 0,
          current_operation: fields[4] || 'Unknown',
          bytes_transferred: parseInt(fields[5]) || 0,
          total_bytes: parseInt(fields[6]) || 0,
          throughput_mbps: parseFloat(fields[7]) || 0,
          updated_at: fields[8]
        };
      }
      return null;
    }).filter(job => job !== null);

    // Get system metrics (basic)
    const uptimeOutput = execSync('uptime', { encoding: 'utf8' });
    const memoryOutput = execSync('free -m', { encoding: 'utf8' });
    
    return {
      active_jobs: activeJobs,
      system_info: {
        uptime: uptimeOutput.trim(),
        memory_info: memoryOutput.split('\n')[1], // Memory line
        timestamp: new Date().toISOString(),
        active_job_count: activeJobs.length
      }
    };
  } catch (error) {
    console.error('Error fetching system status:', error);
    return {
      active_jobs: [],
      system_info: {
        error: 'Failed to fetch system status',
        timestamp: new Date().toISOString(),
        active_job_count: 0
      }
    };
  }
}
