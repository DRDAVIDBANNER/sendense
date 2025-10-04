import { NextRequest, NextResponse } from 'next/server';

const OMA_API_BASE = 'http://localhost:8082';

// POST /api/v1/failover/[vm_name]/cleanup-failed - Cleanup failed execution
export async function POST(
  request: NextRequest,
  { params }: { params: { vm_name: string } }
) {
  try {
    const vmName = params.vm_name;
    
    console.log(`🧹 GUI API: Proxying cleanup failed execution for VM: ${vmName}`);
    
    // Proxy request to OMA API backend (backend uses /api/v1 prefix)
    const response = await fetch(`${OMA_API_BASE}/api/v1/failover/${encodeURIComponent(vmName)}/cleanup-failed`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      // No body needed for cleanup operation
    });

    if (!response.ok) {
      console.error(`❌ GUI API: Cleanup failed execution failed for ${vmName}:`, response.status, response.statusText);
      
      const errorText = await response.text();
      return NextResponse.json(
        { 
          success: false, 
          error: `Cleanup failed: ${response.status} ${response.statusText}`,
          details: errorText
        },
        { status: response.status }
      );
    }

    const data = await response.json();
    console.log(`✅ GUI API: Cleanup failed execution completed for ${vmName}`);
    
    return NextResponse.json(data);
    
  } catch (error) {
    console.error('❌ GUI API: Cleanup failed execution proxy error:', error);
    
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to cleanup failed execution',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}
