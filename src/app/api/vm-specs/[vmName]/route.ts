import { NextRequest, NextResponse } from 'next/server';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ vmName: string }> }
) {
  try {
    const { vmName } = await params;
    
    if (!vmName) {
      return NextResponse.json(
        { error: 'VM name is required' },
        { status: 400 }
      );
    }

    // Get VM specifications from vm_disks table (most recent job)
    const { execSync } = require('child_process');
    const dbCommand = `mysql -u oma_user -p'oma_password' migratekit_oma -e "SELECT cpu_count, memory_mb, os_type, power_state, display_name, vm_tools_version FROM vm_disks WHERE job_id = (SELECT id FROM replication_jobs WHERE source_vm_name = '${vmName}' ORDER BY created_at DESC LIMIT 1) LIMIT 1;" 2>/dev/null`;
    
    try {
      const dbOutput = execSync(dbCommand, { encoding: 'utf8' });
      console.log('🔍 Raw database output:', dbOutput);
      
      const lines = dbOutput.trim().split('\n');
      if (lines.length >= 2) {
        const headers = lines[0].split('\t');
        const values = lines[1].split('\t');
        
        const specs = {
          cpu_count: values[0] && values[0] !== '0' ? parseInt(values[0]) || null : null,
          memory_mb: values[1] ? parseInt(values[1]) || null : null,
          memory_gb: values[1] ? Math.round(parseInt(values[1]) / 1024 * 10) / 10 : null,
          os_type: values[2] || null,
          power_state: values[3] || null,
          display_name: values[4] || null,
          vm_tools_version: values[5] || null
        };
        
        console.log('✅ VM specs for', vmName, ':', specs);
        return NextResponse.json(specs);
      } else {
        return NextResponse.json({
          cpu_count: null,
          memory_mb: null,
          memory_gb: null,
          os_type: null,
          power_state: null,
          display_name: null,
          vm_tools_version: null
        });
      }
    } catch (dbError) {
      console.error('Database query failed:', dbError);
      return NextResponse.json(
        { error: 'Failed to fetch VM specifications' },
        { status: 500 }
      );
    }
  } catch (error) {
    console.error('VM Specs API Error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
