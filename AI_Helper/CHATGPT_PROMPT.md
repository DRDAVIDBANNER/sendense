# ChatGPT-5 Context: MigrateKit OSSEA Project

**Copy this entire document and paste as your first message to ChatGPT-5 to establish full project context**

---

## 🎯 PROJECT OVERVIEW

You are assisting with MigrateKit OSSEA, a production VMware-to-OSSEA migration platform with complete VM failover capabilities. The project is 95% production ready with critical infrastructure operational.

### **Core Components**
- **VMA (VMware-side agent)**: Discovery, CBT, replication orchestration against vCenter
- **OMA (OSSEA-side API/services)**: Failover engines, replication orchestration, job tracking  
- **Volume Management Daemon**: Single source of truth for all OSSEA volume operations
- **TLS Tunnel**: All traffic via port 443 between VMA and OMA

### **Current Phase**
📋 **PROJECT CONSOLIDATION AND ASSESSMENT** - Deep dive investigation of replication/migration bugs with NO CHANGES until assessment complete.

---

## 🚨 ABSOLUTE PROJECT RULES (NON-NEGOTIABLE)

### **1. SOURCE CODE AUTHORITY**
- **ONLY** use code under `source/current/` as authoritative
- **NEVER** reference archived code without explicit permission (`archive/`, `archived_*/`)
- **FORBIDDEN**: Top-level versioned directories (`oma-api-*`, `vma-api-server-*`)
- **VERSION**: Respect `source/current/VERSION.txt` for active version

### **2. VOLUME OPERATIONS (CRITICAL)**
- **MANDATORY**: ALL OSSEA volume operations via Volume Daemon (`internal/common/volume_client.go`)
- **FORBIDDEN**: Direct OSSEA SDK calls (`osseaClient.AttachVolume()`, `osseaClient.DetachVolume()`, etc.)
- **EXCEPTION**: Volume Daemon internal operations under `internal/volume/`

### **3. LOGGING AND JOB TRACKING**
- **MANDATORY**: ALL business logic uses `internal/joblog` (tracker.StartJob → RunStep → EndJob)
- **FORBIDDEN**: Direct logrus/slog in operation logic
- **EXCEPTION**: HTTP middleware may bridge to centralized logging

### **4. NETWORKING CONSTRAINTS**
- **ONLY PORT 443**: All VMA-OMA traffic via TLS tunnel
- **NO DIRECT CONNECTIONS**: No direct NBD ports; everything tunneled
- **SINGLE NBD PORT**: All NBD traffic on port 10809 via tunnel

### **5. DATABASE SCHEMA SAFETY**
- **NEVER ASSUME FIELD NAMES**: Always validate against actual schema
- **CRITICAL CONFLICT**: `device_mappings.volume_id` vs `volume_uuid` mismatch in NBD exports
- **RULE**: If schema mismatch detected, STOP and request resolution

---

## 📊 DATABASE SCHEMA (EXACT FIELD NAMES)

### **OMA Database (MariaDB)**
**Core Tables**:
- `ossea_configs`: id, name, api_url, api_key, secret_key, domain, zone, template_id, network_id, service_offering_id, disk_offering_id, oma_vm_id, created_at, updated_at, is_active
- `replication_jobs`: id, source_vm_id, source_vm_name, source_vm_path, vcenter_host, datacenter, replication_type, target_network, status, progress_percent, current_operation, bytes_transferred, total_bytes, transfer_speed_bps, error_message, change_id, previous_change_id, snapshot_id, nbd_port, nbd_export_name, target_device, ossea_config_id, created_at, updated_at, started_at, completed_at
- `vm_disks`: id, job_id, disk_id, vmdk_path, size_gb, datastore, unit_number, label, capacity_bytes, provisioning_type, ossea_volume_id, disk_change_id, sync_status, sync_progress_percent, bytes_synced, created_at, updated_at
- `failover_jobs`: id, replication_job_id, vm_id, job_type, status, destination_vm_id, linstor_snapshot_name, network_mappings, error_message, created_at, updated_at, started_at, completed_at
- `cloudstack_job_tracking`: id, cloudstack_job_id, cloudstack_command, cloudstack_status, operation_type, correlation_id, parent_job_id, status, created_at, updated_at

### **Volume Daemon Database (MariaDB)**
- `volume_operations`: id, type, status, volume_id, vm_id, request, response, error, created_at, updated_at, completed_at
- `device_mappings`: id, volume_id, vm_id, device_path, cloudstack_state, linux_state, size, last_sync, created_at, updated_at
- `nbd_exports`: id, volume_id, export_name, device_path, port, status, metadata, created_at, updated_at

**⚠️ CRITICAL SCHEMA CONFLICT**: `device_mappings.volume_id` vs `device_mappings.volume_uuid` in FK references

---

## 🏗️ CURRENT ARCHITECTURE STATUS

### **✅ OPERATIONAL (PRODUCTION READY)**
- **Volume Management Daemon**: Fully integrated, single source of truth
- **Single Port NBD**: Port 10809 with SIGHUP reload, concurrent migrations validated
- **VMA Progress API**: Restored and operational via OMA proxy
- **CloudStack SDK Workarounds**: Complete workarounds for JSON unmarshaling issues
- **CBT Auto-Enablement**: Database-integrated ChangeID storage
- **Live Failover**: Production ready VM failover capability

### **🚧 NEEDS ATTENTION**
- **Test Failover**: 95% complete, `destination_vm_id` database update bug
- **Enhanced Cleanup**: Broken centralized logging integration
- **Schema Conflict**: `device_mappings` column naming mismatch
- **Logging Systems**: Mixed joblog vs old patterns

---

## 🚨 CURRENT CRITICAL ISSUES (INVESTIGATION PHASE)

### **Issue 1: Enhanced Failover Destination VM ID Bug**
- **Status**: 🚨 CONFIRMED - Database write issue
- **Symptom**: `failover_jobs.destination_vm_id` not updated after test VM creation
- **Impact**: Cleanup service cannot find test VMs
- **Root Cause**: Enhanced failover logic works but missing DB record update

### **Issue 2: Source Network Detection Failures**
- **Status**: 🔍 Under investigation
- **Symptom**: QCDEV-AUVIK01 VM showing "unknown" for source network
- **Impact**: Network mapping validation failures

### **Issue 3: Volume Mounting Conflicts**
- **Status**: 🔍 Under investigation  
- **Symptom**: Duplicate volume mount issues in some scenarios
- **Impact**: Volume operation failures

### **Issue 4: Centralized Logging Integration**
- **Status**: 🚨 Broken in enhanced cleanup service
- **Symptom**: Undefined `ecs.logger` references, mixed logging patterns
- **Impact**: Service failures and inconsistent logging

---

## 📁 SOURCE CODE LOCATIONS

### **Current Layout (NEEDS CONSOLIDATION)**
```
PROJECT_ROOT/
├── source/current/          # ✅ VMA code consolidated here
│   ├── vma-api-server/     # VMA entry point
│   ├── vma/                # VMA libraries  
│   └── VERSION.txt         # Active version
├── internal/oma/           # 🔄 OMA services (needs consolidation)
├── internal/volume/        # 🔄 Volume Daemon (needs consolidation)
├── cmd/oma/               # 🔄 OMA entry point
├── cmd/volume-daemon/     # 🔄 Volume Daemon entry point
├── oma-api-*              # ❌ Scattered versioned dirs (archive these)
├── vma-api-server-*       # ❌ Scattered binaries (archive these)
└── bin/                   # ❌ Mixed binaries (clean up)
```

### **Key Entry Points**
- **VMA**: `source/current/vma-api-server/main.go`
- **OMA**: `cmd/oma/main.go` 
- **Volume Daemon**: `cmd/volume-daemon/main.go`
- **Shared Volume Client**: `internal/common/volume_client.go`

---

## 🔧 OPERATIONAL ENVIRONMENT

### **Network Topology**
- **VMA**: 10.0.100.231 (VMware appliance, outbound-only)
- **OMA**: 10.245.246.125 (OSSEA appliance)
- **Tunnel**: VMA → OMA port 443 (all traffic)
- **NBD**: Single port 10809 via tunnel with unique exports

### **Testing VMs Available**
- **pgtest1**: Currently in failed-over state (good cleanup test candidate)
- **pgtest2**: Available for testing
- **PGWINTESTBIOS**: Available for testing
- **QCDEV-AUVIK01**: Network detection issues

### **Database Access**
- **OMA Database**: MariaDB on OMA appliance
- **Volume Daemon Database**: MariaDB on OMA appliance
- **Connection**: Via OMA appliance access

---

## 🎯 BEHAVIORAL EXPECTATIONS

### **When Analyzing Issues**
1. **Read schema files first** - Never assume field names
2. **Validate Volume Daemon usage** - Check for direct OSSEA SDK violations
3. **Check joblog integration** - Identify old logging patterns
4. **Verify source code location** - Only reference `source/current/` or approved internal dirs

### **When Proposing Changes**
1. **Assessment first** - Current phase is investigation only
2. **Minimal changes** - Small, focused, reviewable diffs
3. **Schema validation** - Confirm field names before database operations
4. **Volume Daemon compliance** - All volume ops via daemon
5. **Git commit required** - User preference for clean commits before changes

### **When Encountering Conflicts**
1. **Schema mismatch** - Stop and request resolution
2. **Volume operations** - Enforce Volume Daemon usage
3. **Logging patterns** - Standardize on joblog for business logic
4. **Source code confusion** - Stick to `source/current/` authority

---

## 🔍 TROUBLESHOOTING QUICK REFERENCE

### **Volume Daemon Issues**
```bash
# Check daemon health
curl -f http://localhost:8090/health

# Check volume operations
curl -s http://localhost:8090/api/v1/operations | jq

# Check device mappings
curl -s http://localhost:8090/api/v1/volumes | jq
```

### **Database Field Validation**
```sql
-- Check actual schema
DESCRIBE device_mappings;
DESCRIBE nbd_exports;
DESCRIBE failover_jobs;

-- Validate FK constraints
SELECT * FROM information_schema.KEY_COLUMN_USAGE 
WHERE TABLE_NAME = 'nbd_exports';
```

### **Service Status**
```bash
# OMA API
systemctl status oma-api

# Volume Daemon  
systemctl status volume-daemon

# VMA API (on VMA appliance)
ssh pgrayson@10.0.100.231 "ps aux | grep vma-api-server"
```

---

## 🚨 CRITICAL SUCCESS FACTORS

### **Must Maintain**
- **3.2 GiB/s NBD baseline** - Never break existing functionality
- **Port 443 only** - All traffic via tunnel
- **Volume Daemon authority** - All volume ops via daemon
- **JobLog consistency** - Business operations tracked properly

### **Must Avoid**
- **Schema assumptions** - Always validate field names
- **Volume SDK calls** - Outside of Volume Daemon
- **Simulation code** - Only live data operations
- **Endpoint sprawl** - Minimal API surfaces

### **Must Resolve**
- **Schema conflicts** before dependent work
- **Logging inconsistencies** for service stability  
- **Destination VM ID bug** for cleanup functionality
- **Source code consolidation** for AI consistency

---

**🎯 Context Summary**: You're working with a 95% production-ready VMware migration platform in assessment phase. Focus on investigation over changes, enforce Volume Daemon usage, validate schema field names, and maintain the port 443 tunnel architecture. The project has complex history but clear current rules - follow them strictly to avoid architectural violations.
