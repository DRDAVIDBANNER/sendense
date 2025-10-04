// VM Context Detail API Route - Get specific VM context
// Proxies to OMA API /api/v1/vm-contexts/{vm_name}

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

    // Proxy to OMA API
    const omaResponse = await fetch(`http://localhost:8082/api/v1/vm-contexts/${encodeURIComponent(vmName)}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        // TODO: Add authentication headers when implemented
      },
    });

    if (!omaResponse.ok) {
      console.error('OMA API Error:', omaResponse.status, omaResponse.statusText);
      
      if (omaResponse.status === 404) {
        return NextResponse.json(
          { error: `VM context not found: ${vmName}` },
          { status: 404 }
        );
      }
      
      return NextResponse.json(
        { error: 'Failed to fetch VM context from OMA API' },
        { status: omaResponse.status }
      );
    }

    const data = await omaResponse.json();

    // Enhance with VM specifications from vm_disks table
    try {
      const { execSync } = require('child_process');
      const dbCommand = `mysql -u oma_user -p'oma_password' migratekit_oma -e "SELECT cpu_count, memory_mb, os_type, power_state FROM vm_disks WHERE job_id = (SELECT id FROM replication_jobs WHERE source_vm_name = '${vmName}' ORDER BY created_at DESC LIMIT 1) LIMIT 1;" 2>/dev/null`;
      
      const dbOutput = execSync(dbCommand, { encoding: 'utf8' });
      const lines = dbOutput.trim().split('\n');
      
      if (lines.length >= 2) {
        // Skip header line, get data line
        const values = lines[1].split('\t');
        
        if (values.length >= 4) {
          // Enhance the context with VM specifications
          data.context = {
            ...data.context,
            cpu_count: values[0] ? parseInt(values[0]) : null,
            memory_mb: values[1] ? parseInt(values[1]) : null,
            os_type: values[2] || null,
            power_state: values[3] || null
          };
          console.log('✅ Enhanced VM context with specifications from vm_disks for:', vmName);
        }
      }
    } catch (dbError) {
      console.warn('⚠️ Could not fetch VM specifications from database:', dbError);
    }
    
    return NextResponse.json(data);
  } catch (error) {
    console.error('VM Context Detail API Error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
