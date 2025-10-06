# API Documentation Update - October 6, 2025

## Motivation
GUI integration work revealed confusion between multiple discovery endpoints with similar names but different capabilities. The `/discovery/bulk-add` endpoint was being incorrectly used instead of `/discovery/add-vms`, causing 400 Bad Request errors because:

1. **bulk-add does NOT support credential_id** (requires manual credentials every time)
2. **add-vms DOES support credential_id** (works with saved VMware credentials)
3. Payload structure differs: `vm_ids` vs `vm_names`

This confusion wasted development time and could easily happen again without proper documentation.

## Files Updated

### 1. `/source/current/api-documentation/OMA.md`
**Section:** Discovery (Enhanced) - VM Discovery Without Immediate Replication

**Changes:**
- Complete rewrite of discovery endpoints section with detailed documentation
- Added comprehensive request/response schemas for each endpoint
- Added data structure definitions: `DiscoveredVMInfo`, `VMADiskInfo`, `VMANetworkInfo`
- **Critical warnings** about bulk-add vs add-vms differences
- Documented disk and network metadata fields (was missing from GUI)
- Added architecture notes explaining VMA integration
- Added callsites documentation showing GUI → add-vms workflow
- Added database impact documentation

**Key Additions:**
```
⚠️ IMPORTANT: This is the CORRECT endpoint for GUI bulk add operations
✅ Supports credential_id - works with saved VMware credentials
✅ Accepts VM names array (vm_names field)
✅ Returns detailed per-VM success/failure
```

vs Legacy bulk-add:
```
⚠️ DO NOT USE FROM GUI - This endpoint does NOT support credential_id
❌ Requires explicit vcenter/username/password in every request
❌ No saved credential support
```

### 2. `/source/current/api-documentation/API_DB_MAPPING.md`
**Section:** Discovery

**Changes:**
- Expanded discovery endpoints with detailed database impact
- Documented which tables are written (vm_replication_contexts)
- Documented which tables are read (vmware_credentials for credential_id lookup)
- Documented which tables are NOT touched (replication_jobs, vm_disks, nbd_exports)
- Added field-level documentation of what gets written to vm_replication_contexts
- Documented foreign key relationships (ossea_config_id auto-assignment)
- Documented unique constraints preventing duplicate discovery

**Key Additions:**
```
**Database Impact Details:**
- Table: vm_replication_contexts
  - Writes: context_id, vm_name, vmware_vm_id, vm_path, vcenter_host, datacenter, 
    current_status='discovered', ossea_config_id, cpu_count, memory_mb, os_type, 
    power_state, vm_tools_version, auto_added=1, scheduler_enabled=1, 
    created_at, updated_at, last_status_change
  - FK: ossea_config_id references ossea_configs(id) - auto-assigned from active config
  - Unique Constraint: vm_name + vcenter_host (prevents duplicate discovery)
```

### 3. `/source/current/api-documentation/README.md`
**New Section:** ⚠️ Common API Gotchas & Best Practices

**Changes:**
- Added "Discovery Endpoints Confusion" section
- Added "Disk and Network Metadata" section
- Added "Database Schema Safety" section
- Added "Credential Management" section
- Documented common field name mismatches between backend and frontend
- Provided clear guidance on which endpoints to use for what purpose

**Purpose:**
- Prevent this exact confusion from happening again
- Help future developers (and AI assistants) make correct endpoint choices
- Document common frontend/backend field name mismatches
- Explain the credential_id workflow vs manual credentials

## Impact

### Immediate Benefits
1. ✅ Clear documentation of correct endpoint for GUI bulk add
2. ✅ Warning markers prevent wrong endpoint usage
3. ✅ Disk/network metadata fields documented (were missing from GUI)
4. ✅ Database impact fully documented for discovery operations
5. ✅ Common gotchas section prevents future confusion

### Future Benefits
1. Faster GUI development (clear request/response schemas)
2. Reduced debugging time (know which endpoint does what)
3. Better frontend/backend contract understanding
4. Clearer database transaction boundaries
5. Reduced risk of duplicate work or wrong assumptions

## Related Work

**GUI Fix Prompt:** `/home/oma_admin/sendense/GROK-FIX-DISKS-NETWORKS-BULKADD.md`

This Grok prompt was created alongside the documentation update to fix the three GUI issues:
1. Missing disk display (backend sends, GUI ignores)
2. Missing network display (backend sends, GUI ignores)
3. Wrong bulk-add endpoint usage (bulk-add → add-vms)

## Compliance

✅ Follows MAINTENANCE_RULES.md requirements
✅ Documents request/response schemas
✅ Documents database impacts
✅ Documents handler locations
✅ Documents callsites
✅ Documents authentication requirements
✅ Adds warnings for gotchas/confusion points

## Session Context

This documentation update was part of the October 6, 2025 session focused on:
- VMware VM discovery GUI integration
- Fixing VM names display issue (completed)
- Adding disk/network display (pending GUI fix)
- Fixing bulk add operation (pending GUI fix)

The documentation ensures this knowledge is preserved and prevents future confusion.
