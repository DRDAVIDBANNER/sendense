# GUI Discovery System Fix - v6.16.0
**Date**: October 4, 2025  
**Status**: ✅ PRODUCTION READY  
**Deployment Package Updated**: YES

## Overview
Complete fix of the GUI discovery system to remove all hardcoded VMware credentials and migrate to the Enhanced Discovery API with database-managed credentials.

## Critical Changes

### 1. Removed Old Discovery Endpoint
**Deleted**: `src/app/api/discover/route.ts`
- Old endpoint that proxied directly to VMA with hardcoded credentials
- Replaced by Enhanced Discovery API

### 2. Added New Enhanced Discovery Proxy Routes
**Created**: `src/app/api/discovery/discover-vms/route.ts`
- Proxies to OMA backend `/api/v1/discovery/discover-vms`
- Uses `credential_id` from database instead of hardcoded credentials
- Returns `discovered_vms` array

**Created**: `src/app/api/discovery/add-vms/route.ts`
- Proxies to OMA backend `/api/v1/discovery/add-vms`
- Adds VMs to management using database credentials
- Proper VM context creation

### 3. Updated GUI Components

#### Discovery View (`src/components/discovery/DiscoveryView.tsx`)
**Before**: 
- Hardcoded vCenter credentials (vcenter, username, password, datacenter)
- Called old `/api/discover` endpoint

**After**:
- Loads VMware credentials from database (`/api/v1/vmware-credentials`)
- Credential selector dropdown
- Calls new `/api/discovery/discover-vms` endpoint
- Uses `credential_id` parameter

#### VM Context Panel (`src/components/layout/RightContextPanel.tsx`)
**Before**:
- Called old `/api/discover` with hardcoded credentials
- Parsed `vms` array

**After**:
- Calls new `/api/discovery/discover-vms` with `credential_id`
- Uses VM's credential from context or default
- Parses `discovered_vms` array
- Added `ossea_config_id: 1` to replication request

#### Failover Page (`src/app/failover/page.tsx`)
**Before**:
```typescript
fetch('/api/discover', {
  body: JSON.stringify({
    vcenter: 'quad-vcenter-01.quadris.local',
    username: 'administrator@vsphere.local',
    password: 'EmyGVoBFesGQc47-',
    datacenter: 'DatabanxDC'
  })
})
```

**After**:
```typescript
fetch('/api/discovery/discover-vms', {
  body: JSON.stringify({
    credential_id: 2, // Default credential
    create_context: false
  })
})
```

#### API Library (`src/lib/api.ts`)
**Before**:
```typescript
async discoverVMs(credentials: {
  vcenter: string;
  username: string;
  password: string;
  datacenter: string;
}): Promise<{ vms: VM[] }>
```

**After**:
```typescript
async discoverVMs(params: {
  credential_id: number;
  filter?: string;
  create_context?: boolean;
}): Promise<{ discovered_vms: VM[] }>
```

### 4. Fixed Replication Start
**Added**: `ossea_config_id: 1` to replication request in `RightContextPanel.tsx`
- Backend `/api/replicate` route requires this field
- Prevents "OSSEA configuration ID is required" error

## Security Improvements
✅ **NO hardcoded credentials** anywhere in GUI code  
✅ All credentials managed in database (`vmware_credentials` table)  
✅ Encrypted credentials using AES-256-GCM  
✅ Credentials selectable per-operation  

## Database Requirements
Ensure these tables exist with proper data:
1. `vmware_credentials` - At least one credential set as default
2. `ossea_configs` - At least one OSSEA configuration (ID=1)

## Deployment Notes

### Included in Deployment Package
✅ All updated GUI source files  
✅ New Next.js API proxy routes  
✅ Updated components (DiscoveryView, RightContextPanel, etc.)  
✅ Migration GUI systemd service file  

### Deployment Process
The `deploy-real-production-oma.sh` script will:
1. Copy GUI source from `oma-deployment-package/gui/` to target
2. Run `npm install` on target server
3. Run `npm run build` to create production `.next` directory
4. Deploy and start `migration-gui.service`

### Post-Deployment Configuration
After deployment, configure via GUI:
1. **Settings → VMware Credentials**: Add vCenter credentials
2. **Settings → OSSEA Configuration**: Configure CloudStack settings
3. **Discovery**: Test discovery with saved credentials

## Testing Checklist
- [x] Discovery page loads credentials from database
- [x] Discovery works with credential selector
- [x] Add to Management creates VM context
- [x] Start Replication includes ossea_config_id
- [x] No hardcoded credentials in any file
- [x] All `/api/discover` references removed
- [x] Failover page VM discovery works
- [x] Scheduler uses new discovery API

## Files Modified
```
src/app/api/discovery/discover-vms/route.ts    [NEW]
src/app/api/discovery/add-vms/route.ts         [NEW]
src/app/api/discover/route.ts                  [DELETED]
src/app/api/replicate/route.ts                 [MODIFIED]
src/app/failover/page.tsx                      [MODIFIED]
src/components/discovery/DiscoveryView.tsx     [MODIFIED]
src/components/layout/RightContextPanel.tsx    [MODIFIED]
src/lib/api.ts                                 [MODIFIED]
```

## Version History
- **v6.15.0**: Initial security cleanup (removed some hardcoded credentials)
- **v6.16.0**: Complete discovery system migration to Enhanced Discovery API

## Production Status
✅ **DEPLOYED**: 10.246.5.124 (October 4, 2025)  
✅ **TESTED**: Discovery, Add to Management, Start Replication all working  
✅ **PACKAGE UPDATED**: oma-deployment-package/gui/ synchronized with latest changes  

## Known Issues
None - all discovery endpoints updated and tested.

## Future Enhancements
- [ ] Multi-OSSEA config selector in GUI (currently hardcoded to ID=1)
- [ ] Credential test button in discovery view
- [ ] Credential auto-discovery from vCenter

