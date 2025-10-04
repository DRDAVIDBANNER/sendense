import { NextRequest, NextResponse } from 'next/server';
import crypto from 'crypto';

// POST - Test OSSEA connection
export async function POST(request: NextRequest) {
  try {
    const config = await request.json();
    
    // Basic validation
    if (!config.api_url || !config.api_key || !config.secret_key || !config.zone) {
      return NextResponse.json(
        { error: 'Missing required configuration fields' },
        { status: 400 }
      );
    }
    
    // Validate URL format
    let apiUrl: URL;
    try {
      apiUrl = new URL(config.api_url);
    } catch (err) {
      return NextResponse.json(
        { error: 'Invalid API URL format' },
        { status: 400 }
      );
    }
    
    // First test OMA API connection
    try {
      const omaResponse = await fetch('http://localhost:8082/api/v1/ossea/config', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
        },
        body: JSON.stringify({
          action: 'test',
          config: {
            name: config.name,
            api_url: config.api_url,
            api_key: config.api_key,
            secret_key: config.secret_key,
            domain: config.domain,
            zone: config.zone,
            template_id: config.template_id,
            network_id: config.network_id,
            service_offering_id: config.service_offering_id,
            disk_offering_id: config.disk_offering_id,
            oma_vm_id: config.oma_vm_id
          }
        })
      });

      if (!omaResponse.ok) {
        const errorText = await omaResponse.text();
        return NextResponse.json(
          { error: `OMA API error: ${omaResponse.status} - ${errorText}` },
          { status: 400 }
        );
      }

      // If OMA test succeeds, also test direct OSSEA API connection
      const omaTestResult = await omaResponse.json();
      if (!omaTestResult.success) {
        return NextResponse.json(
          { error: omaTestResult.message || 'OMA API test failed' },
          { status: 400 }
        );
      }

      // Now test direct OSSEA API connection
      // Build API request with HMAC-SHA1 signature (CloudStack/OSSEA standard)
      const params = new URLSearchParams({
        command: 'listZones',
        response: 'json',
        apikey: config.api_key
      });
      
      // Sort parameters for signature
      const sortedParams = Array.from(params.entries())
        .sort((a, b) => a[0].localeCompare(b[0]))
        .map(([key, value]) => `${key}=${encodeURIComponent(value).toLowerCase()}`)
        .join('&');
      
      // Generate signature
      const signature = crypto
        .createHmac('sha1', config.secret_key)
        .update(sortedParams)
        .digest('base64');
      
      params.append('signature', signature);
      
      // Make the API call
      const testUrl = `${config.api_url}?${params.toString()}`;
      const response = await fetch(testUrl, {
        method: 'GET',
        headers: {
          'Accept': 'application/json'
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        // Check if we got a valid response structure
        if (data.listzonesresponse) {
          return NextResponse.json({ 
            success: true, 
            message: '✅ Both OMA and OSSEA connections successful! All APIs are reachable and authentication is valid.' 
          });
        } else {
          return NextResponse.json(
            { error: 'Invalid API response format - may not be a valid OSSEA/CloudStack endpoint' },
            { status: 400 }
          );
        }
      } else {
        const errorText = await response.text();
        return NextResponse.json(
          { error: `OSSEA API error: ${response.status} - ${errorText}` },
          { status: 400 }
        );
      }
    } catch (err: any) {
      // Network error or other issue
      return NextResponse.json(
        { error: `Connection failed: ${err.message || 'Unable to reach API'}` },
        { status: 400 }
      );
    }
  } catch (error) {
    console.error('Failed to test connection:', error);
    return NextResponse.json(
      { error: 'Failed to test connection' },
      { status: 500 }
    );
  }
}