# MigrateKit Source Code - libnbd Progress Integration

**Version**: v2.7.0-libnbd-progress-fixed  
**Date**: 2025-09-06  
**Status**: ✅ **WORKING** - libnbd integration with VMA progress tracking

## 🎯 **CRITICAL PROJECT RULE COMPLIANCE**

### **✅ SOURCE CODE AUTHORITY**
- **Authoritative Location**: `/source/current/migratekit/`
- **All source code consolidated** from scattered locations
- **No duplicate code** in `/internal/` or `/cmd/` directories

### **🚨 PREVIOUS VIOLATIONS FIXED**
1. **Working libnbd source** was in `/internal/vmware_nbdkit/` (WRONG)
2. **Broken nbdcopy source** was in `/source/current/migratekit/` (WRONG)
3. **Fixed**: Copied working libnbd source to proper `/source/current/migratekit/` location

## 📁 **SOURCE CODE STRUCTURE**

### **Core Components**
```
source/current/migratekit/
├── main.go                           # Entry point with --job-id flag
├── internal/
│   ├── vmware_nbdkit/
│   │   └── vmware_nbdkit.go         # 🎯 WORKING libnbd integration + progress tracking
│   ├── progress/
│   │   └── vma_client.go            # VMA progress HTTP client
│   ├── target/
│   │   └── cloudstack.go            # CloudStack NBD target
│   └── vmware/
│       └── *.go                     # VMware API integration
├── go.mod                           # Module dependencies (includes libnbd)
└── VERSION.txt                      # Current version
```

### **Key Files**
- **`internal/vmware_nbdkit/vmware_nbdkit.go`**: Contains working libnbd integration with real-time progress callbacks
- **`internal/progress/vma_client.go`**: HTTP client for sending progress updates to VMA API
- **`main.go`**: Command-line interface with `--job-id` flag for progress tracking

## 🔧 **TECHNICAL IMPLEMENTATION**

### **libnbd Integration Features**
1. **Real-time Progress Tracking**: Uses libnbd callbacks during incremental copy
2. **VMA API Integration**: Sends HTTP POST updates to VMA every 2 seconds
3. **Multi-disk Support**: Includes `disk_id` in progress payloads
4. **Job ID Support**: Reads `MIGRATEKIT_PROGRESS_JOB_ID` environment variable
5. **Stage Progression**: Reports proper migration stages (Discover → Transfer → Complete)

### **Progress Update Flow**
```
migratekit (libnbd callbacks) 
    ↓ HTTP POST every 2s
VMA API (/api/v1/progress/{jobId}/update)
    ↓ Auto-initialize & store
VMA Progress Service (in-memory)
    ↓ Poll every 30s  
OMA Progress Poller
    ↓ Database update
replication_jobs & vm_disks tables
```

## 🚨 **INVALID BINARIES ARCHIVED**

### **Moved to `/source/archive/invalid-binaries-20250906/`**
- `migratekit-v2.6.0-multi-disk-job-id` (20MB) - ❌ Lost libnbd, uses nbdcopy
- `migratekit-v2.6.1-progress-tracking` (20MB) - ❌ Lost libnbd, uses nbdcopy

### **Working Binaries**
- `migratekit-v2.5.0-libnbd-callbacks` (21MB) - ✅ Working libnbd (legacy)
- `migratekit-v2.5.1-libnbd-connection-fix` (21MB) - ✅ Working libnbd (legacy)
- `migratekit-v2.7.0-libnbd-progress-fixed` (20MB) - ✅ **CURRENT** - libnbd + progress tracking

## 🎯 **USAGE**

### **Command Line**
```bash
./migratekit-v2.7.0-libnbd-progress-fixed migrate \
  --vmware-endpoint vcenter.example.com \
  --vmware-username admin@vsphere.local \
  --vmware-password password \
  --vmware-path "/Datacenter/vm/MyVM" \
  --nbd-export-name migration-vol-uuid \
  --job-id job-20250906-123456 \
  --debug
```

### **Environment Variables**
- `MIGRATEKIT_PROGRESS_JOB_ID`: Job ID for progress tracking (set by --job-id flag)
- `MIGRATEKIT_JOB_ID`: CBT ChangeID storage (separate from progress tracking)

## 🔍 **VERIFICATION**

### **Confirm libnbd Integration**
```bash
# Check binary contains libnbd (not nbdcopy)
strings migratekit-v2.7.0-libnbd-progress-fixed | grep libnbd
# Should show: libnbd.so.0, *libnbd.Libnbd, etc.

# Check for progress tracking
strings migratekit-v2.7.0-libnbd-progress-fixed | grep "Progress update sent"
# Should show: "📊 Progress update sent to VMA"
```

### **Test Progress Tracking**
1. Start incremental migration with `--job-id` flag
2. Monitor VMA logs for progress updates: `journalctl -u vma-api.service -f`
3. Check VMA API: `curl http://localhost:8081/api/v1/progress/job-id`
4. Verify database updates in `replication_jobs` table

## 📋 **BUILD INSTRUCTIONS**

### **From Source**
```bash
cd /home/pgrayson/migratekit-cloudstack/source/current/migratekit
go build -o migratekit-v2.7.x-description .
```

### **Dependencies**
- Go 1.21+
- libnbd development libraries
- VMware vSphere API libraries
- CloudStack integration libraries

## 🚨 **CRITICAL RULES**

1. **NEVER build from `/internal/` or `/cmd/` directories**
2. **ALWAYS use `/source/current/migratekit/` as authoritative source**
3. **NEVER revert to nbdcopy** - libnbd is required for progress tracking
4. **ALWAYS test with `--job-id` flag** to verify progress integration
5. **ALWAYS check binary size** - libnbd versions should be 20MB+

## 📊 **VERSION HISTORY**

- **v2.7.0-libnbd-progress-fixed**: Current working version with libnbd + progress tracking
- **v2.5.1-libnbd-connection-fix**: Previous working libnbd version (legacy)
- **v2.6.x**: ❌ INVALID - Lost libnbd integration, archived

---

**BOTTOM LINE**: This is the authoritative source code location. All migratekit development must happen here. The libnbd integration with VMA progress tracking is working and properly consolidated.
