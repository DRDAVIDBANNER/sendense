# 🚨 **SOURCE CODE AUTHORITY - MIGRATEKIT OSSEA PROJECT**

**Created**: September 25, 2025  
**Purpose**: **DEFINITIVE SOURCE CODE LOCATIONS** - NO MORE CONFUSION  
**Rule**: **ONLY these locations contain authoritative source code**

---

## 🚨 **ABSOLUTE SOURCE CODE AUTHORITY**

### **🎯 AUTHORITATIVE SOURCE LOCATIONS**

| **Component** | **Authoritative Source Location** | **Status** | **Version** |
|---------------|-----------------------------------|------------|-------------|
| **MigrateKit** | `/home/pgrayson/migratekit-cloudstack/source/current/migratekit/` | ✅ **WORKING** | v2.18.0-job-type-propagation |
| **VMA API Server** | `/home/pgrayson/migratekit-cloudstack/source/current/vma-api-server/` | ✅ **WORKING** | v1.10.4-progress-fixed |
| **VMA Services** | `/home/pgrayson/migratekit-cloudstack/source/current/vma/` | ✅ **WORKING** | Current |
| **OMA API** | `/home/pgrayson/migratekit-cloudstack/source/current/oma/` | ✅ **WORKING** | v2.22.0-polling-debug-enhanced |
| **Volume Daemon** | `/home/pgrayson/migratekit-cloudstack/source/current/volume-daemon/` | ✅ **WORKING** | v1.2.3-multi-volume-snapshots |

### **🚫 FORBIDDEN LOCATIONS**
- ❌ **Root directory**: `/home/pgrayson/migratekit-cloudstack/main.go` (ARCHIVED)
- ❌ **Root internal**: `/home/pgrayson/migratekit-cloudstack/internal/` (ARCHIVED)
- ❌ **Top-level binaries**: `/home/pgrayson/migratekit-cloudstack/migratekit-*` (RUNTIME ONLY)
- ❌ **Archive directories**: `/home/pgrayson/migratekit-cloudstack/archive/` (READ-ONLY)
- ❌ **Restored temp**: `/home/pgrayson/migratekit-cloudstack/source/restored-temp/` (ARCHIVED)

---

## 📂 **DETAILED SOURCE CODE INVENTORY**

### **🔧 MIGRATEKIT** - `/source/current/migratekit/`
```
migratekit/
├── main.go                           # ✅ Entry point with VMA progress integration
├── internal/
│   ├── vmware_nbdkit/
│   │   └── vmware_nbdkit.go         # ✅ libnbd integration + CBT delta calculation
│   ├── progress/
│   │   └── vma_client.go            # ✅ VMA HTTP progress client
│   ├── vmware/
│   │   ├── disk_info.go             # ✅ CBT delta calculation functions
│   │   └── change_id.go             # ✅ Change ID handling
│   ├── target/
│   │   └── cloudstack.go            # ✅ CloudStack NBD target with libnbd
│   └── nbdcopy/
│       └── nbdcopy.go               # ✅ nbdcopy fallback integration
├── go.mod                           # ✅ Module dependencies
└── VERSION.txt                      # ✅ v2.5.1-libnbd-connection-fix
```

### **🔧 VMA API SERVER** - `/source/current/vma-api-server/`
```
vma-api-server/
├── main.go                          # ✅ VMA API server entry point
└── VERSION.txt                      # ✅ Current
```

### **🔧 VMA SERVICES** - `/source/current/vma/`
```
vma/
├── api/
│   ├── server.go                    # ✅ VMA Control API with CBT endpoints
│   └── progress_handler.go          # ✅ VMA Progress API endpoints
├── services/
│   └── progress_service.go          # ✅ VMA Progress Service (in-memory)
├── vmware/
│   └── service.go                   # ✅ VMware service with environment setup
├── cbt/
│   └── cbt.go                       # ✅ CBT management with datacenter support
└── progress/
    ├── model.go                     # ✅ Progress data models
    └── parser.go                    # ✅ Multi-disk progress parsing
```

### **🔧 OMA API** - `/source/current/oma/`
```
oma/
├── cmd/main.go                      # ✅ OMA API entry point
├── api/handlers/                    # ✅ REST API handlers
├── services/
│   └── vma_progress_poller.go       # ✅ VMA Progress Poller with timing fixes
├── database/                        # ✅ Database models and repositories
├── failover/                        # ✅ Multi-volume snapshot system
├── workflows/                       # ✅ Migration orchestration
└── VERSION.txt                      # ✅ v2.7.6-api-uuid-correlation
```

### **🔧 VOLUME DAEMON** - `/source/current/volume-daemon/`
```
volume-daemon/
├── cmd/main.go                      # ✅ Volume Daemon entry point
├── service/volume_service.go        # ✅ Volume operations with snapshot tracking
├── models/volume.go                 # ✅ DeviceMapping with snapshot fields
├── database/                        # ✅ Repository with snapshot methods
└── VERSION.txt                      # ✅ v1.2.0-volume-daemon-consolidation
```

---

## 🚨 **WORKING BINARY LOCATIONS**

### **🎯 DEPLOYED WORKING BINARIES**
| **Component** | **Binary Location** | **Symlink** | **Version** |
|---------------|-------------------|-------------|-------------|
| **MigrateKit** | VMA: `/home/pgrayson/migratekit-cloudstack/migratekit-v2.18.0-job-type-propagation` | `/home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel` | v2.18.0 |
| **VMA API** | VMA: `/home/pgrayson/migratekit-cloudstack/vma-api-server-v1.10.4-progress-fixed` | `/home/pgrayson/migratekit-cloudstack/vma-api-server` | v1.10.4 |
| **OMA API** | OMA: `/opt/migratekit/bin/oma-api-v2.22.0-polling-debug-enhanced` | `/opt/migratekit/bin/oma-api` | v2.22.0 |
| **Volume Daemon** | OMA: `/usr/local/bin/volume-daemon-v1.2.3-multi-volume-snapshots` | `/usr/local/bin/volume-daemon` | v1.2.3 |

### **📦 BUILD COMMANDS**
```bash
# MigrateKit
cd /home/pgrayson/migratekit-cloudstack/source/current/migratekit
go build -o migratekit-v2.X.X-feature-name .

# VMA API Server  
cd /home/pgrayson/migratekit-cloudstack/source/current/vma-api-server
go build -o vma-api-server-v1.X.X-feature-name .

# OMA API
cd /home/pgrayson/migratekit-cloudstack/source/current/oma
go build -o oma-api-v2.X.X-feature-name ./cmd

# Volume Daemon
cd /home/pgrayson/migratekit-cloudstack/source/current/volume-daemon
go build -o volume-daemon-v1.X.X-feature-name ./cmd
```

---

## 🧹 **CLEANUP COMPLETED**

### **✅ ARCHIVED CONFUSING VERSIONS**
The following **confusing/duplicate versions** have been identified for archiving:

#### **Root Directory Cleanup**
- ❌ `/home/pgrayson/migratekit-cloudstack/main.go` → **ARCHIVE**
- ❌ `/home/pgrayson/migratekit-cloudstack/internal/` → **ARCHIVE** 
- ❌ `/home/pgrayson/migratekit-cloudstack/cmd/` → **ARCHIVE**

#### **Duplicate Binary Cleanup** 
- ❌ `/home/pgrayson/migratekit-cloudstack/migratekit-v2.13.*` (broken versions) → **ARCHIVE**
- ❌ `/home/pgrayson/migratekit-cloudstack/oma-api-*` (scattered versions) → **ARCHIVE**
- ❌ `/home/pgrayson/migratekit-cloudstack/vma-api-server-v1.9.*` (old versions) → **ARCHIVE**

#### **Restored Temp Cleanup**
- ❌ `/home/pgrayson/migratekit-cloudstack/source/restored-temp/` → **ARCHIVE**

---

## 🎯 **CURRENT WORKING FUNCTIONALITY**

### **✅ CONFIRMED WORKING FEATURES**
1. **✅ CBT Auto-Enablement**: Automatically enables CBT if disabled
2. **✅ VMA Progress Integration**: Real-time progress tracking restored
3. **✅ Job Type Propagation**: Incremental jobs properly tagged in database
4. **✅ Multi-Volume Snapshots**: Device_mappings snapshot tracking ready
5. **✅ libnbd Integration**: Direct libnbd operations with progress callbacks
6. **✅ CBT Delta Calculation**: calculateDeltaSize() function integrated
7. **✅ Final Progress Updates**: 100% completion sent to VMA

### **🎯 TESTING STATUS**
- **pgtest2**: Ready for failover testing (multi-disk, ready_for_failover)
- **pgtest1**: Ready for CBT auto-enablement testing  
- **pgtest3**: Ready for CBT auto-enablement testing

---

## 🚨 **AI ASSISTANT RULES FOR FUTURE SESSIONS**

### **📖 MANDATORY READING**
**Every AI assistant MUST read this document first** before making ANY changes to source code.

### **🚫 ABSOLUTE PROHIBITIONS**
1. **NEVER modify code outside `/source/current/`**
2. **NEVER build from root directory or scattered locations**
3. **NEVER reference archived/old source code**
4. **NEVER revert to older incomplete versions**
5. **NEVER lose working functionality during changes**

### **✅ REQUIRED WORKFLOW**
1. **Read this document** to understand source authority
2. **Verify component locations** before any modifications
3. **Build from authoritative source only**
4. **Test incrementally** to avoid functionality loss
5. **Update this document** if source locations change

### **🔧 BUILD VERIFICATION**
```bash
# Verify all components build from authoritative source
cd /home/pgrayson/migratekit-cloudstack/source/current/migratekit && go build . # ✅ MUST WORK
cd /home/pgrayson/migratekit-cloudstack/source/current/vma-api-server && go build . # ✅ MUST WORK  
cd /home/pgrayson/migratekit-cloudstack/source/current/oma && go build ./cmd # ✅ MUST WORK
cd /home/pgrayson/migratekit-cloudstack/source/current/volume-daemon && go build ./cmd # ✅ MUST WORK
```

### **📋 FUNCTIONALITY VERIFICATION**
```bash
# Verify working functionality (NEVER lose these)
✅ CBT Auto-Enablement: migratekit enables CBT if disabled
✅ VMA Progress Tracking: Real-time progress updates 
✅ Job Type Propagation: Incremental jobs tagged correctly
✅ Multi-Volume Snapshots: Device_mappings snapshot tracking
✅ libnbd Integration: Direct libnbd with progress callbacks
✅ CBT Delta Calculation: Actual changed data size calculation
```

---

## 🎯 **NEXT STEPS**

### **🧹 IMMEDIATE CLEANUP**
1. **Archive root directory source** to `archive/2025-09-25-root-source-cleanup/`
2. **Archive duplicate binaries** to `archive/2025-09-25-duplicate-binaries/`
3. **Archive restored-temp** to `archive/2025-09-25-restored-temp/`
4. **Update AI_Helper** with source authority rules

### **🛡️ PROTECTION MEASURES**
1. **Add .gitignore rules** for root directory source
2. **Create source validation scripts** 
3. **Add pre-commit hooks** to prevent source authority violations
4. **Document exact working binary builds** with source verification

---

**🚨 BOTTOM LINE**: `/source/current/` is the ONLY authoritative source location. Everything else is confusion that MUST be archived. No more lost functionality, no more scattered source code, no more chaos.

**Status**: 📋 **READY FOR SYSTEMATIC CLEANUP**

