# MigrateKit OSSEA - Architecture Overview

## 🎯 **System Purpose**

MigrateKit OSSEA is a **production-ready VMware-to-CloudStack migration platform** featuring **CBT-based progress tracking**, **sparse block optimization**, and **high-performance libnbd engine** with TLS-encrypted data transfer and real-time monitoring.

## 🏗️ **High-Level Architecture**

```
┌─────────────────────────────────────────────────────────────────┐
│                    Production Network Boundary                  │
│                                                                 │
│  VMA (VMware Appliance)              OMA (OSSEA Appliance)      │
│  ┌─────────────────────┐             ┌─────────────────────────┐ │
│  │ 🚀 Outbound Only    │   SSH/TLS   │ 🎯 Inbound Hub          │ │
│  │                     │◄────────────┤                         │ │
│  │ • VMA API :8081     │  Port 443   │ • OMA API :8082         │ │
│  │ • stunnel :10808    │  Port 22    │ • NBD Server :10809     │ │
│  │ • migratekit        │  Port 80    │ • SSH Tunnel Endpoint   │ │
│  │                     │             │ • Multiple Exports      │ │
│  └─────────────────────┘             └─────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 🔥 **CBT-Based Migration Architecture** 

### **Current Production Architecture (September 2025)**
- **CBT Progress Tracking**: Accurate progress based on actual data transfer, not disk capacity
- **Sparse Block Optimization**: Automatic zero-block detection and skipping for bandwidth efficiency
- **libnbd Performance Engine**: Native libnbd integration for maximum throughput
- **Single NBD Port**: All migrations use port 10809 with shared `nbd-server`
- **Real-time Monitoring**: VMA progress service with OMA polling and database integration
- **Database-Backed Mappings**: `vm_export_mappings` table tracks VM-to-export relationships
- **Intelligent SIGHUP Management**: Only create new exports when needed, reuse existing exports without SIGHUP
- **Multi-Disk Support**: Separate exports for each disk within the same VM (disk0, disk1, etc.)
- **Concurrent Migrations**: Multiple VMs migrate simultaneously with optimal resource utilization

### **Fixed Issues**
- ✅ **NBD Server Restart Problem**: Fixed SIGHUP targeting multiple PIDs causing server restarts
- ✅ **Export Reuse Implementation**: VM-based export persistence preventing unnecessary SIGHUP operations
- ✅ **Export Verification**: Added NBD server state verification to handle missing exports
- ✅ **Database Optimization**: Export reuse skips duplicate database record creation
- ✅ **MigrateKit Integration**: Dynamic export names work end-to-end
- ✅ **Permission Management**: `oma` user has sudo access to NBD helper script
- ✅ **Tunnel Configuration**: Verified VMA:10808 → OMA:443 → OMA:10809 path

### **Key Benefits**
- ✅ **Simplified Networking**: Only one NBD port to manage
- ✅ **Concurrent Operations**: Unlimited concurrent migrations with zero conflicts
- ✅ **Export Reuse Efficiency**: Existing VM exports reused without any server operations
- ✅ **Multi-Migration Support**: Same VM can have multiple concurrent migration jobs
- ✅ **Resource Efficiency**: Single NBD daemon handles all jobs optimally
- ✅ **Stability**: Zero NBD server restarts during normal operations

## 🔄 **Data Flow Architecture**

### **1. Migration Data Path**
```
VMware vCenter → VDDK/nbdkit → stunnel TLS → OMA:443 → NBD Server:10809 → /dev/vdX
                                                            ↓
                                          Export: migration-vm-{vmID}-disk{N}
                                          (Reused across multiple jobs)
```

### **2. Control Command Path**  
```
OMA API → SSH Tunnel :9081 → VMA API :8081 → VMware Operations
```

### **3. VM-Based Export Management**
```
New Job → Check VM Export Mappings → Existing Export? → Reuse (No SIGHUP)
                                  ↓
                                  New Export? → Update /etc/nbd-server/config-base → SIGHUP → Active Export
```

## 🏭 **Component Architecture**

### **VMware Migration Appliance (VMA)**
- **Purpose**: Reads from VMware vSphere, coordinates migrations
- **Network**: Outbound-only connections  
- **Key Services**:
  - VMA Control API (port 8081) - 4 minimal endpoints
  - stunnel client (port 10808) - TLS tunnel to OMA
  - migratekit-tls-tunnel - Migration execution engine
  - SSH tunnel client - Bidirectional communication

### **OSSEA Migration Appliance (OMA)**
- **Purpose**: Orchestrates migrations, manages OSSEA volumes
- **Network**: Inbound hub for all communication
- **Key Services**:
  - OMA Control API (port 8082) - Unified migration API
  - NBD Server (port 10809) - Single port with multiple exports
  - stunnel server (port 443) - TLS termination
  - SSH tunnel server - Bidirectional communication
  - Migration GUI (port 3001) - Web dashboard

## 🔧 **NBD Server Management**

### **VM-Based Export Design**
```
Port 10809 (Single NBD Daemon)
├── Export: migration-vm-4205784a-098a-40f1-1f1e-a5cd2597fd59-disk0 → /dev/vdc (40GB) [REUSED]
├── Export: migration-vm-4205a841-0265-f4bd-39a6-39fd92196f53-disk0 → /dev/vdd (110GB) [REUSED]
└── Export: migration-vm-{vmID3}-disk1 → /dev/vde (80GB) [NEW]
```

### **Intelligent Export Management**
1. **New Job Request** → OMA API (with VM ID + Disk Unit Number)
2. **Volume Creation** → CloudStack/OSSEA
3. **Device Attachment** → Linux block device
4. **Check VM Mappings** → Query `vm_export_mappings` table
5. **Existing Export?** → Reuse without any NBD operations
6. **New Export?** → Append to `/etc/nbd-server/config-base` + SIGHUP + Create mapping
7. **Migration Start** → VMA connects to VM-persistent export

## 🌐 **Network Security**

### **TLS Tunnel Architecture**
```
VMA stunnel:10808 ←→ TLS:443 ←→ OMA stunnel → NBD:10809
                                      ↓
                              VM-Persistent Exports:
                              • migration-vm-{vmID1}-disk0 → /dev/vdc [REUSED]
                              • migration-vm-{vmID2}-disk0 → /dev/vdd [REUSED]
                              • migration-vm-{vmID3}-disk1 → /dev/vde [NEW]
```

### **Security Features**
- **Port 443 Only**: All NBD traffic via single TLS tunnel
- **Bidirectional SSH Tunnel**: API communication via port 22 with auto-recovery
- **Enhanced Tunnel Service**: `vma-tunnel-enhanced-v2.service` with health monitoring
- **No Direct Connections**: VMA cannot directly access NBD ports
- **Outbound Only**: VMA initiates all connections with automatic failure recovery

## 📊 **Proven VM Export Reuse**

**LIVE VALIDATION** (2025-08-12):
- ✅ **pgtest2**: VM ID `4205784a-098a-40f1-1f1e-a5cd2597fd59` → export: `migration-vm-4205784a-098a-40f1-1f1e-a5cd2597fd59-disk0`
- ✅ **PGWINTESTBIOS**: VM ID `4205a841-0265-f4bd-39a6-39fd92196f53` → export: `migration-vm-4205a841-0265-f4bd-39a6-39fd92196f53-disk0`
- ✅ **Export Reuse**: Subsequent jobs for same VM reuse existing exports without SIGHUP
- ✅ **NBD Stability**: Zero server restarts during concurrent and sequential operations
- ✅ **Multi-Disk Support**: Individual exports created per disk unit number within same VM

## 🎉 **Production Status**

**Status**: ✅ **PRODUCTION READY**

**Architecture Validated**:
- VM-Based Export Reuse with intelligent SIGHUP management operational
- Export reuse prevents NBD server restarts and enhances stability  
- Concurrent and sequential migrations proven working
- Multi-disk support within single VM validated
- Network compliance verified
- Database persistence with `vm_export_mappings` functional
- Volume lifecycle complete

---
**Last Updated**: 2025-08-12  
**Architecture**: VM-Based NBD Export Reuse with Intelligent SIGHUP Management  
**Major Achievement**: NBD server restart issue completely resolved
**Documentation Status**: Complete and current