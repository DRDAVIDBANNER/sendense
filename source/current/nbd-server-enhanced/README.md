# Enhanced NBD Server for MigrateKit OSSEA

## Problem Statement

The standard NBD server maintains in-memory export-to-device mappings that persist even after SIGHUP configuration reloads. This causes **stale device correlation** in multi-disk VM scenarios where:

1. Volumes are detached and reattached during failover operations
2. Device paths change (e.g., `/dev/vdc` → `/dev/vdb`)
3. NBD server continues using **cached device mappings** instead of current ones
4. Result: VMware disks write to **wrong target volumes**, causing data corruption

## Solution: Enhanced SIGHUP Handler

This enhancement adds **memory cache flushing** to the NBD server SIGHUP handler to prevent stale device correlation.

### Key Enhancements

#### 1. **refresh_existing_server_configs()**
- Re-parses configuration files during SIGHUP
- Updates existing SERVER objects with fresh device paths
- Prevents stale device path caching

#### 2. **clear_export_client_cache()**
- Clears any cached export-to-device mappings
- Forces clients to re-negotiate export configurations
- Ensures fresh device correlation on next connection

#### 3. **Enhanced SIGHUP Processing**
- Maintains backward compatibility with existing SIGHUP workflow
- Adds cache flush operations before processing new exports
- Provides detailed logging for troubleshooting

## Implementation

### Current SIGHUP Handler (BEFORE):
```c
if (is_sighup_caught) {
    msg(LOG_INFO, "reconfiguration request received");
    is_sighup_caught = 0;
    
    // Only appends new servers - doesn't refresh existing ones
    n = append_new_servers(servers, genconf, &gerror);
}
```

### Enhanced SIGHUP Handler (AFTER):
```c
if (is_sighup_caught) {
    msg(LOG_INFO, "🔄 MIGRATEKIT: Enhanced reconfiguration request received");
    is_sighup_caught = 0;
    
    // NEW: Refresh existing server configurations
    refresh_existing_server_configs(servers, genconf);
    
    // NEW: Clear export client cache
    clear_export_client_cache();
    
    // EXISTING: Append new servers (preserve original functionality)  
    n = append_new_servers(servers, genconf, &gerror);
    
    msg(LOG_INFO, "✅ MIGRATEKIT: Enhanced reconfiguration completed with cache flush");
}
```

## Benefits

- ✅ **Fixes multi-disk correlation**: Prevents stale device mappings
- ✅ **Backward compatible**: Preserves existing SIGHUP workflow
- ✅ **Production ready**: Minimal change to proven NBD server
- ✅ **Maintains architecture**: No networking or tunnel changes required
- ✅ **Enhanced logging**: Clear visibility into cache flush operations

## Deployment

### Option A: Patch and Build
1. Apply patch to NBD server 3.26.1 source
2. Build enhanced NBD server binary
3. Replace system NBD server with enhanced version
4. Test with multi-disk VM operations

### Option B: Alternative Implementation
If building custom NBD server is not feasible:
1. Implement NBD export recreation instead of SIGHUP
2. Delete and recreate export config files during volume operations
3. Use full NBD server restart for critical operations (with job coordination)

## Testing

Test the enhancement by:
1. Creating multi-disk VM replication
2. Performing volume detach/reattach operations (failover/rollback)
3. Verifying NBD exports point to correct devices after SIGHUP
4. Confirming no data corruption occurs during multi-disk operations

## Impact

This enhancement enables **reliable multi-disk VM migration and failover** without the operational complexity of:
- Dynamic port management (NBDKit approach)
- Device path consistency enforcement (Ubuntu limitations)  
- Complete NBD server replacement (architectural changes)

**Result**: Production-ready multi-disk enterprise VM migration platform.








