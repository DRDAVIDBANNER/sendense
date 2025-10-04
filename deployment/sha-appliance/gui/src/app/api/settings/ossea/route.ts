import { NextRequest, NextResponse } from 'next/server';
import fs from 'fs/promises';
import path from 'path';

const CONFIG_FILE = path.join(process.env.HOME || '/home/pgrayson', '.ossea_config.json');

// GET - Load existing configuration
export async function GET() {
  try {
    // First try to load from OMA API database (primary source)
    try {
      const omaResponse = await fetch('http://localhost:8082/api/v1/ossea/config', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
        },
        body: JSON.stringify({
          action: 'get' // Get all configurations (will get the most recent active one)
        })
      });

      if (omaResponse.ok) {
        const omaData = await omaResponse.json();
        if (omaData.success && omaData.configs && omaData.configs.length > 0) {
          // Use the first (most recent) configuration
          const config = omaData.configs[0];
          // Don't send the secret key in full for security
          if (config.api_key) {
            config.api_key = config.api_key.substring(0, 8) + '****';
          }
          if (config.secret_key) {
            config.secret_key = '****';
          }
          console.log('✅ Loaded OSSEA config from database:', config.name);
          return NextResponse.json(config);
        }
      }
    } catch (omaError) {
      console.warn('Failed to load from OMA database, trying file backup:', omaError);
    }

    // Fallback: Try to load from local file backup
    try {
      const data = await fs.readFile(CONFIG_FILE, 'utf8');
      const config = JSON.parse(data);
      // Don't send the secret key in full
      if (config.secret_key) {
        config.secret_key = config.secret_key.substring(0, 4) + '****';
      }
      console.log('📁 Loaded OSSEA config from file backup');
      return NextResponse.json(config);
    } catch (_fileError) {
      console.log('No existing configuration found, returning defaults');
    }

    // Return default empty config if neither database nor file exists
    return NextResponse.json({
      name: 'production-ossea',
      api_url: '',
      api_key: '',
      secret_key: '',
      zone: '',
      domain: '',
      template_id: '',
      network_id: '',
      service_offering_id: '',
      disk_offering_id: '',
      oma_vm_id: ''
    });
  } catch (_error) {
    return NextResponse.json(
      { error: 'Failed to load configuration' },
      { status: 500 }
    );
  }
}

// POST - Save configuration
export async function POST(request: NextRequest) {
  try {
    const config = await request.json();
    
    // Load existing config to preserve secret key if not changed
    let existingConfig: any = {};
    try {
      const data = await fs.readFile(CONFIG_FILE, 'utf8');
      existingConfig = JSON.parse(data);
    } catch (err) {
      // File doesn't exist yet, that's ok
    }
    
    // If secret key is masked, use the existing one
    if (config.secret_key && config.secret_key.includes('****')) {
      config.secret_key = existingConfig.secret_key;
    }
    
    // Validate required fields are IDs (not names)
    // Note: Zone field accepts names, not IDs, so it's excluded from ID validation
    const idFields = [
      { key: 'template_id', label: 'Template ID' },
      { key: 'service_offering_id', label: 'Service Offering ID' },
      { key: 'disk_offering_id', label: 'Disk Offering ID' },
      { key: 'network_id', label: 'Network ID' },
    ];
    for (const field of idFields) {
      if (config[field.key] && !/^[a-f0-9\-]{8,}$/.test(config[field.key])) {
        return NextResponse.json(
          { error: `${field.label} must be a valid CloudStack ID (not a name)` },
          { status: 400 }
        );
      }
    }

    // Check if configuration already exists to determine create vs update
    let existingConfigId = null;
    try {
      const checkResponse = await fetch('http://localhost:8082/api/v1/ossea/config', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
        },
        body: JSON.stringify({
          action: 'get'
        })
      });
      
      if (checkResponse.ok) {
        const checkData = await checkResponse.json();
        if (checkData.success && checkData.configs && checkData.configs.length > 0) {
          // Find existing config by name
          const existing = checkData.configs.find(c => c.name === config.name);
          if (existing) {
            existingConfigId = existing.id;
          }
        }
      }
    } catch (checkError) {
      console.warn('Failed to check existing config:', checkError);
    }

    // Save to database via OMA API (the source of truth)
    try {
      const isUpdate = existingConfigId !== null;
      const requestBody = isUpdate ? {
        action: 'update',
        id: existingConfigId,
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
      } : {
        action: 'create',
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
      };

      console.log(`${isUpdate ? 'Updating' : 'Creating'} OSSEA config in database:`, config.name);
      
      const omaResponse = await fetch('http://localhost:8082/api/v1/ossea/config', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent' // Long-lived token for OMA API
        },
        body: JSON.stringify(requestBody)
      });

      if (!omaResponse.ok) {
        throw new Error(`OMA API error: ${omaResponse.statusText}`);
      }
    } catch (omaError) {
      console.error('Failed to save to OMA database:', omaError);
      return NextResponse.json(
        { error: 'Failed to save to database: ' + omaError },
        { status: 500 }
      );
    }
    
    // Also save to file for backup
    await fs.writeFile(CONFIG_FILE, JSON.stringify(config, null, 2));
    
    // Also create environment variable export file
    const envContent = `#!/bin/bash
# OSSEA Configuration Environment Variables
# Generated by MigrateKit GUI

export OSSEA_API_URL="${config.api_url}"
export OSSEA_API_KEY="${config.api_key}"
export OSSEA_SECRET_KEY="${config.secret_key}"
export OSSEA_ZONE="${config.zone}"
export OSSEA_DOMAIN="${config.domain || ''}"
export OSSEA_TEMPLATE_ID="${config.template_id || ''}"
export OSSEA_NETWORK_ID="${config.network_id || ''}"
export OSSEA_SERVICE_OFFERING_ID="${config.service_offering_id || ''}"
export OSSEA_DISK_OFFERING_ID="${config.disk_offering_id || ''}"
export OSSEA_OMA_VM_ID="${config.oma_vm_id || ''}"

echo "OSSEA environment variables loaded"
`;

    const envFile = path.join(process.env.HOME || '/home/pgrayson', 'ossea_env.sh');
    await fs.writeFile(envFile, envContent);
    await fs.chmod(envFile, 0o755);
    
    return NextResponse.json({ success: true });
  } catch (error) {
    console.error('Failed to save configuration:', error);
    return NextResponse.json(
      { error: 'Failed to save configuration' },
      { status: 500 }
    );
  }
}