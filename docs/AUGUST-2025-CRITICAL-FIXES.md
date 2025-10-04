# August 2025 Critical Fixes - MigrateKit OSSEA

**Date**: August 11-12, 2025  
**Status**: ✅ **ALL ISSUES RESOLVED - NBD RESTART ISSUE COMPLETELY FIXED**

## 🎯 **Overview**

This document details the critical issues discovered and resolved in August 2025 that were preventing successful migrations. All fixes have been implemented, tested, and deployed.

## 🔧 **Critical Issues Fixed**

### **Issue 1: Hardcoded Export Names in MigrateKit**

**Symptoms**: 
- VMA logs showed: `nbd://127.0.0.1:10808/migration` (hardcoded)
- NBD connection errors: "server has no export named 'migration'"
- Migrations failed with "Connection reset by peer"

**Root Cause**:
- `migratekit` CloudStack target in `internal/target/cloudstack.go` was hardcoding `/migration` export name
- `--nbd-export-name` parameter was being set but ignored in `GetPath()` method
- VMA correctly passed dynamic names like `migration-job-20250811-094935` but migratekit used `/migration`

**Solution**:
```go
// BEFORE (hardcoded):
nbdPath := fmt.Sprintf("nbd://%s:%s/migration", t.nbdHost, t.nbdPort)

// AFTER (dynamic):
type CloudStack struct {
    // ... other fields ...
    nbdExportName string  // Added field to store export name
}

// Store export name in Connect():
t.nbdExportName = exportName

// Use dynamic export name in GetPath():
nbdPath := fmt.Sprintf("nbd://%s:%s/%s", t.nbdHost, t.nbdPort, t.nbdExportName)
```

**Files Modified**:
- `/home/pgrayson/migratekit-cloudstack/internal/target/cloudstack.go`

**Deployment**:
- Built with Go 1.23.4 on VMA: `/home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel`
- ✅ **VERIFIED**: Dynamic export names now working end-to-end

---

### **Issue 2: Conflicting NBD Systems**

**Symptoms**:
- Permission denied errors during NBD export creation
- "Connection reset by peer" errors during migration
- Old `config-dynamic-*` files being created alongside `config-base`

**Root Cause**:
- OMA workflow was using **two conflicting NBD systems simultaneously**:
  - **OLD**: `nbd.Service` → individual servers with `config-dynamic-10809` files
  - **NEW**: Shared server with `/etc/nbd-server/config-base` + SIGHUP
- Migration engine called `nbd.Service.CreateAndStartExport()` which tried to start individual NBD servers on port 10809
- Conflict with existing shared NBD server caused connection failures

**Solution**:
```go
// BEFORE (conflicting systems):
type MigrationEngine struct {
    nbdService *nbd.Service  // Old individual server approach
}
exportInfo, err := m.nbdService.CreateAndStartExport(req.JobID, attachResult.DevicePath)

// AFTER (unified system):
type MigrationEngine struct {
    // Removed nbdService field - use package-level function
}
exportInfo, err := nbd.AddDynamicExport(req.JobID, attachResult.DevicePath)

// New function uses shared server + oma-nbd-helper:
func AddDynamicExport(jobID, devicePath string) (*ExportInfo, error) {
    exportName := fmt.Sprintf("migration-%s", jobID)
    return addExportToSharedServer(exportName, devicePath)
}
```

**Files Modified**:
- `/home/pgrayson/migratekit-cloudstack/internal/oma/workflows/migration.go`
- `/home/pgrayson/migratekit-cloudstack/internal/oma/nbd/server.go`

**Deployment**:
- Built and deployed to OMA: `/opt/migratekit/bin/oma-api`
- ✅ **VERIFIED**: Single shared NBD server architecture working

---

### **Issue 3: Permission Denied for NBD Configuration**

**Symptoms**:
- Error: `/etc/nbd-server/config-base: Permission denied`
- Migration workflow failing at NBD export creation step
- `oma` user unable to modify NBD configuration files

**Root Cause**:
- `/etc/nbd-server/config-base` owned by `root:root` 
- `oma-nbd-helper` script running as `oma` user without sudo privileges
- NBD configuration operations require root access for file modification

**Solution**:
```bash
# 1. Grant sudo permissions to oma user for NBD helper:
echo "oma ALL=(ALL) NOPASSWD: /usr/local/bin/oma-nbd-helper" | sudo tee /etc/sudoers.d/oma-nbd

# 2. Update all helper script calls to use sudo:
appendCmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper", "append-config", "/etc/nbd-server/config-base")
sighupCmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper", "sighup-nbd", "/etc/nbd-server/config-base")
```

**Files Modified**:
- `/etc/sudoers.d/oma-nbd` (new file)
- `/home/pgrayson/migratekit-cloudstack/internal/oma/nbd/server.go`

**Deployment**:
- Updated OMA API with sudo calls: `/opt/migratekit/bin/oma-api`
- ✅ **VERIFIED**: `oma` user can now manage NBD configurations

---

### **Issue 4: NBD Server Restart Problem (August 12, 2025)**

**Symptoms**:
- Adding new NBD exports caused NBD server to restart instead of graceful SIGHUP reload
- Existing migration jobs interrupted when new jobs started  
- Multiple NBD server PIDs detected during monitoring
- Export conflicts and connection failures

**Root Cause**:
- `oma-nbd-helper` script's `pgrep` command returned multiple PIDs (including stale/zombie processes)
- `kill -HUP $NBD_PID` attempted to signal all PIDs, causing server instability
- Rapid consecutive SIGHUP operations from job-based exports overwhelmed the system

**Solution**:
```bash
# BEFORE (multiple PIDs):
NBD_PID=$(pgrep -f "nbd-server -C ${CONFIG_PATH}")

# AFTER (single PID):
NBD_PID=$(pgrep -f "nbd-server -C ${CONFIG_PATH}" | head -1)
```

**VM-Based Export Reuse Implementation**:
- Created `vm_export_mappings` database table for persistent VM-to-export relationships
- Implemented export reuse logic that checks existing mappings before creating new exports
- Added multi-disk support with `disk_unit_number` for VMs with multiple disks
- Export names now use VM ID: `migration-vm-{vmID}-disk{N}` instead of job ID
- Subsequent jobs for same VM reuse existing exports without any SIGHUP operations

**Files Modified**:
- `/usr/local/bin/oma-nbd-helper` - Fixed SIGHUP PID targeting
- `scripts/migrations/vm_export_mappings.sql` - New database table
- `internal/oma/database/vm_export_mapping.go` - New GORM model and repository
- `internal/oma/nbd/server.go` - VM-based export reuse logic
- `internal/oma/workflows/migration.go` - Integration with export reuse system

**Deployment**:
- Database migration applied automatically
- Updated OMA API binary: `/opt/migratekit/bin/oma-api`
- ✅ **VERIFIED**: NBD server remains stable, zero restarts during operations

---

## 🎯 **Current Architecture Status**

### **VM-Based Export Reuse Flow**:
1. **GUI** → OMA API creates migration job with VM ID + disk unit number
2. **OMA API** → Creates OSSEA volume and attaches to `/dev/vdc`
3. **OMA API** → Calls `nbd.AddDynamicExport("job-20250812-080802", "pgtest2", "4205784a-098a-40f1-1f1e-a5cd2597fd59", 0, vmExportRepo)`
4. **Export Logic** → Checks `vm_export_mappings` for existing VM export
5a. **Existing Export** → Reuses `migration-vm-4205784a-098a-40f1-1f1e-a5cd2597fd59-disk0` without SIGHUP
5b. **New Export** → Creates mapping, appends to config-base, SIGHUP (single PID only)
6. **OMA API** → Calls VMA API with VM-persistent export name
7. **VMA API** → Starts `migratekit --nbd-export-name=migration-vm-4205784a-098a-40f1-1f1e-a5cd2597fd59-disk0`
8. **MigrateKit** → Uses VM-persistent export: `nbd://127.0.0.1:10808/migration-vm-4205784a-098a-40f1-1f1e-a5cd2597fd59-disk0`
9. **Tunnel** → VMA:10808 → stunnel → OMA:443 → OMA:10809 → Reused Export → `/dev/vdc`

### **Key Components Working**:
- ✅ **VM-Based Export Reuse**: VM-persistent exports prevent NBD server restarts
- ✅ **Shared NBD Server**: Single stable `nbd-server` on port 10809
- ✅ **Smart Export Management**: Only SIGHUP when truly needed (new VMs)
- ✅ **Multi-Disk Support**: Individual exports per disk within same VM
- ✅ **Database Persistence**: `vm_export_mappings` tracks all VM-to-export relationships
- ✅ **Fixed SIGHUP**: Single PID targeting prevents server instability
- ✅ **Permission System**: `oma` user has sudo access to NBD operations
- ✅ **Export Name Chain**: GUI → OMA → VMA → MigrateKit → NBD (end-to-end)
- ✅ **Tunnel Infrastructure**: VMA:10808 ↔ OMA:443 ↔ OMA:10809

## 🚀 **Production Readiness**

**Status**: ✅ **READY FOR PRODUCTION USE**

All critical blocking issues have been resolved. The system now supports:
- **Zero NBD server restarts** during normal operations
- **VM-based export reuse** preventing unnecessary SIGHUP operations
- **Multi-disk VM support** with individual exports per disk
- **Concurrent migrations** with optimal resource utilization
- **Stable SIGHUP operations** with single PID targeting
- **Database-backed persistence** for VM export mappings
- **Dynamic export names** throughout the entire pipeline  
- **Proper permission management** for all operations
- **Reliable tunnel infrastructure** with TLS encryption
- **Complete workflow** from GUI through to actual data migration

**Next Steps**: System ready for production migrations with full stability.

---

**Last Updated**: August 12, 2025  
**Major Achievement**: NBD server restart issue completely resolved with VM export reuse
**Verified By**: End-to-end testing with zero server restarts during concurrent operations
**Status**: All systems operational and production-ready with enhanced stability




