# MigrateKit OSSEA - Project Rules and Constraints

**Last Updated**: 2025-09-22  
**Status**: 🚨 **MANDATORY COMPLIANCE**  
**Violations**: Immediately stop and consult user before proceeding

---

## 🚨 **ABSOLUTE PROJECT RULES**

### **1. OPERATIONAL SAFETY (CRITICAL)**
- **NEVER EXECUTE**: Failover operations (live/test) without explicit user approval
- **NEVER EXECUTE**: Cleanup operations without explicit user approval  
- **FORBIDDEN**: Any `curl` commands that trigger failover/cleanup endpoints
- **FORBIDDEN**: Any direct API calls that initiate VM state changes
- **VIOLATION**: Any failover/cleanup execution during testing or debugging
- **RULE**: Always ask user permission before ANY operation that affects VM state

### **2. SOURCE CODE AUTHORITY**
- **CANONICAL SOURCE**: Only `source/current/` contains authoritative code
- **FORBIDDEN**: Do not touch or reference `archive/`, `archived_*/`, or top-level versioned directories
- **VERSION CONTROL**: Respect `source/current/VERSION.txt` for active version
- **BINARY HYGIENE**: No binaries in source trees; use `source/builds/` or `dist/`

### **3. VOLUME OPERATIONS (CRITICAL)**
- **MANDATORY**: ALL OSSEA volume operations MUST use Volume Daemon via `internal/common/volume_client.go`
- **FORBIDDEN**: Direct OSSEA SDK calls for create/attach/detach/delete operations
- **EXCEPTION**: Volume Daemon internal operations under `internal/volume/` are permitted
- **VIOLATION**: Any `osseaClient.AttachVolume()`, `osseaClient.DetachVolume()`, etc. outside Volume Daemon

### **4. LOGGING AND JOB TRACKING**
- **MANDATORY**: ALL business logic operations MUST use `internal/joblog`
- **PATTERN**: `tracker.StartJob()` → `tracker.RunStep()` → `tracker.EndJob()`
- **FORBIDDEN**: Direct logrus/slog calls in operation logic
- **EXCEPTION**: HTTP middleware may bridge to centralized logging for request-scoped metadata

### **5. NETWORKING CONSTRAINTS**
- **ONLY PORT 443**: All VMA-OMA traffic via TLS tunnel on port 443
- **NO DIRECT CONNECTIONS**: No direct connections to NBD ports 10800-11000
- **VMA OUTBOUND-ONLY**: VMA establishes tunnel to OMA port 443
- **SINGLE NBD PORT**: All NBD traffic on port 10809 via tunnel

### **6. DATABASE SCHEMA SAFETY**
- **NEVER ASSUME FIELD NAMES**: Always validate against migrations and schema files
- **SOURCES**: `internal/oma/database/migrations/`, `internal/volume/database/`
- **KNOWN CONFLICT**: `device_mappings.volume_id` vs `device_mappings.volume_uuid` - must be resolved
- **RULE**: If schema mismatch detected, STOP and request resolution

---

## 🏗️ **ARCHITECTURAL STANDARDS**

### **1. Development Standards**
- **TLS NBD BASELINE**: Maintain 3.2 GiB/s performance; never break existing functionality
- **NO MONSTER CODE**: Small, focused functions with clear separation
- **MODULAR DESIGN**: Clean interfaces and pluggable design
- **NO SIMULATION CODE**: Only live data migrations; no synthetic/demo scenarios
- **DOCUMENTATION**: Document major logic as implemented

### **2. API Design Principles**
- **MINIMAL ENDPOINTS**: As few endpoints as possible to avoid sprawl
- **NO ENDPOINT PROLIFERATION**: Prefer refactoring over adding new endpoints
- **REUSE EXISTING**: Use existing API surfaces when possible

### **3. Migration Technology**
- **NBD ONLY**: Always use NBD for migrations; no VDDK fallback
- **REAL DATA ONLY**: No simulation test copies; only actual data migration
- **CBT CHANGEID**: Must be persisted to `vm_disks.disk_change_id` at completion

### **4. Naming Convention**
- **OSSEA NAMING**: Use "OSSEA" throughout; never "CloudStack" in code/docs
- **VERSION NUMBERS**: Explicit version numbers; never "latest" or "final" tags

---

## 🔧 **TECHNICAL REQUIREMENTS**

### **1. Failover System Rules**
- **SINGLE ENGINE**: Use only one active failover engine; archive others
- **DESTINATION VM ID**: Must be written to `failover_jobs.destination_vm_id` after test VM creation
- **LINSTOR SNAPSHOTS**: Mandatory before VirtIO injection in test failover
- **VIRTIO INJECTION**: Essential for Windows VMs using `/usr/share/virtio-win/virtio-win.iso`

### **2. Change Block Tracking (CBT)**
- **AUTO-ENABLEMENT**: VMs must have CBT enabled before migration
- **STORAGE**: ChangeIDs stored in `vm_disks.disk_change_id` field
- **AUDIT TRAIL**: Track in `cbt_history` table
- **NO TEMP FILES**: Never store ChangeIDs in `/tmp` files

### **3. Volume Management**
- **SINGLE SOURCE OF TRUTH**: Volume Daemon localhost:8090 for all operations
- **DEVICE CORRELATION**: Real device paths from polling; never arithmetic assumptions
- **OPERATION MODE**: `operation_mode` ENUM in device_mappings (OMA vs Failover)
- **ATOMIC OPERATIONS**: All volume operations must be atomic via daemon

---

## 🗄️ **DATABASE CONSTRAINTS**

### **1. Schema Validation Rules**
- **FIELD VERIFICATION**: Validate all field names against actual migrations
- **NO ASSUMPTIONS**: Query schema files before using field names
- **FOREIGN KEYS**: Respect CASCADE DELETE relationships

### **2. Known Schema Issues**
- **CRITICAL**: `device_mappings.volume_id` vs `volume_uuid` mismatch in NBD exports
- **TABLES AFFECTED**: `nbd_exports`, triggers, views reference `volume_uuid`
- **REQUIREMENT**: Resolve naming consistency before related work

### **3. Data Integrity**
- **FOREIGN KEY CONSTRAINTS**: 7 FK constraints enforce referential integrity
- **UNIQUE CONSTRAINTS**: 11 unique constraints prevent orphaned records
- **CASCADE DELETE**: Proper cleanup via foreign key relationships

---

## 🚧 **PROJECT STATUS CONSTRAINTS**

### **1. Current Development Focus**
- **PHASE**: Deep dive assessment of replication/migration function bugs
- **RULE**: No changes until assessment complete
- **METHODOLOGY**: Methodical investigation before proposing fixes

### **2. Known Critical Issues**
- **ENHANCED FAILOVER**: `destination_vm_id` not being set after test VM creation
- **NETWORK DETECTION**: Source network showing "unknown" for some VMs
- **VOLUME MOUNTING**: Duplicate volume mount issues in some scenarios
- **LOGGING INTEGRATION**: Mixed old/new logging patterns need cleanup

### **3. Component Status**
- **VOLUME DAEMON**: ✅ Production ready, fully integrated
- **NBD ARCHITECTURE**: ✅ Single port 10809 with SIGHUP reload operational
- **VM FAILOVER**: 95% complete (live=100%, test=95% awaiting snapshot debug)
- **VMA PROGRESS API**: ✅ Operational with proxy integration

---

## 📁 **DIRECTORY STRUCTURE RULES**

### **1. Canonical Layout**
```
source/
├── current/           # AUTHORITATIVE SOURCE
│   ├── vma-api-server/
│   ├── vma/
│   └── VERSION.txt
├── builds/            # Version-stamped binaries
└── archive/           # Read-only historical snapshots
```

### **2. Legacy Code Handling**
- **INTERNAL DIRS**: `internal/oma/`, `internal/volume/`, `internal/vma/` (current location)
- **TOP-LEVEL VERSIONED**: `oma-api-*`, `vma-api-server-*` (to be archived)
- **GO MODULES**: Consolidation needed between single vs multi-module approach

---

## 🔍 **VALIDATION PROCEDURES**

### **1. Before Making Changes**
- [ ] Confirm clean git commit exists
- [ ] Verify current working in `source/current/` or approved internal dirs
- [ ] Check `DATABASE_SCHEMA.md` for field names
- [ ] Validate Volume Daemon usage for volume operations
- [ ] Confirm joblog usage for operation tracking

### **2. Pre-Commit Checks**
- [ ] No binaries added to source trees
- [ ] All volume operations via Volume Daemon client
- [ ] All business logic uses joblog, not direct logging
- [ ] Database field names match schema files
- [ ] Network traffic only via port 443 tunnel

### **3. Testing Requirements**
- [ ] End-to-end testing for major functionality
- [ ] Error scenarios and rollback verification
- [ ] Volume Daemon integration confirmed
- [ ] JobLog correlation IDs working

---

## 🚨 **VIOLATION RESPONSE**

### **Immediate Actions if Rule Violated**
1. **STOP ALL WORK** immediately
2. **IDENTIFY VIOLATION** type and scope
3. **ASSESS IMPACT** on existing functionality
4. **CONSULT USER** before proceeding with fixes
5. **UPDATE MEMORY** to reflect any rule clarifications

### **Common Violations**
- **Volume Operations**: Direct OSSEA SDK calls instead of Volume Daemon
- **Source Code**: Editing archived code instead of current
- **Database**: Using assumed field names without schema validation
- **Logging**: Direct logging calls instead of joblog in business logic
- **Networking**: Attempting direct connections instead of tunnel

---

## 🎯 **SUCCESS CRITERIA**

### **Architectural Compliance**
- [ ] All volume operations via Volume Daemon
- [ ] All business operations use joblog tracking
- [ ] All traffic via port 443 tunnel only
- [ ] Database schema consistency maintained
- [ ] Source code authority respected

### **Quality Gates**
- [ ] No build errors or linter violations
- [ ] No broken foreign key relationships
- [ ] No orphaned volume records
- [ ] No simulation code in production paths
- [ ] Complete documentation for major changes

---

**🚨 CRITICAL**: These rules exist to prevent the architectural drift and version confusion that has plagued this project. Every AI assistant session MUST enforce these constraints without exception.
