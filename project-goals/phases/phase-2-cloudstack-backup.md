# Phase 2: CloudStack Backup Implementation

**Phase ID:** PHASE-02  
**Status:** 🟡 **PLANNED**  
**Priority:** High  
**Timeline:** 4-5 weeks  
**Team Size:** 2-3 developers  
**Dependencies:** Phase 1 Complete

---

## 🎯 Phase Objectives

**Primary Goal:** Implement CloudStack/KVM VM backups using libvirt dirty bitmaps

**Success Criteria:**
- ✅ Deploy Capture Agent on KVM hosts
- ✅ Libvirt dirty bitmap change tracking operational
- ✅ Full + incremental backups from CloudStack VMs
- ✅ Integration with existing backup repository (QCOW2)
- ✅ 90%+ data reduction on incrementals
- ✅ Performance: 3+ GiB/s throughput

**Strategic Value:**
- **Bidirectional Capability:** Can backup FROM CloudStack AND replicate TO CloudStack
- **Complete Platform Coverage:** VMware ↔ CloudStack full support
- **Competitive Advantage:** Few vendors handle CloudStack well

---

## 🏗️ Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│ PHASE 2: CLOUDSTACK BACKUP ARCHITECTURE                     │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  CloudStack Management                                       │
│       ↓ (VM snapshots for consistency)                       │
│  KVM Hypervisor Host                                         │
│   ├─ NEW: Sendense Capture Agent (CloudStack variant)       │
│   ├─ libvirt connection (qemu:///system)                     │
│   ├─ Dirty bitmap management (QEMU/QMP)                      │
│   └─ NBD server (export VM disks)                            │
│       ↓ SSH Tunnel (port 443)                               │
│  Control Plane                                               │
│   ├─ REUSE: Backup Repository Interface ✅                  │
│   ├─ REUSE: QCOW2 Storage Backend ✅                        │
│   ├─ NEW: CloudStack API integration                         │
│   └─ NEW: Dirty bitmap workflow                              │
│       ↓                                                      │
│  /var/lib/sendense/backups/                                  │
│   └─ {cloudstack-vm-uuid}/disk-0/                           │
│      ├─ full-20251101-120000.qcow2   (50 GB)                │
│      ├─ incr-20251101-180000.qcow2   (3 GB)                 │
│      └─ incr-20251102-000000.qcow2   (2 GB)                 │
└──────────────────────────────────────────────────────────────┘
```

---

## 🔧 Technical Implementation

### **Dirty Bitmap Workflow**

```go
// Phase 2 core implementation
func CreateCloudStackBackup(vmUUID string, backupType string) error {
    // 1. Create CloudStack snapshot for consistency
    snapshot := cloudstackAPI.CreateVMSnapshot(vmUUID)
    defer cloudstackAPI.DeleteVMSnapshot(snapshot.ID)
    
    // 2. Connect to libvirt on KVM host
    conn := libvirt.NewConnect("qemu:///system")
    dom := conn.LookupDomainByName(vmUUID)
    
    // 3. Setup dirty bitmap (if incremental)
    if backupType == "incremental" {
        dom.BlockDirtyBitmapAdd("vda", "sendense-bitmap-" + timestamp)
    }
    
    // 4. Export disk via NBD
    diskPath := "/var/lib/libvirt/images/" + vmUUID + ".qcow2"
    nbdExport := qemuNBD.Export(diskPath, 10808)
    
    // 5. Create backup file on Control Plane
    backupFile := createBackupTarget(vmUUID, backupType)
    
    // 6. Stream data via SSH tunnel (reuse existing infrastructure)
    return streamViaSSHTunnel(nbdExport, backupFile)
}
```

---

## 📋 Task Breakdown

### **Task 1: CloudStack API Integration** (Week 1)

**Goal:** Control CloudStack VMs for consistent snapshots

**Sub-Tasks:**
1.1. **CloudStack Client Library**
   - VM lifecycle operations (start/stop/snapshot)
   - Volume management
   - Host discovery (find which KVM host runs VM)
   
1.2. **VM Consistency Points**
   - Create VM snapshot before backup
   - Handle guest tools integration (VSS/fsfreeze)
   - Delete snapshot after backup completes
   
1.3. **Host SSH Management**
   - Discover KVM hosts from CloudStack
   - SSH key management for host access
   - Execute commands on remote KVM hosts

**Files to Create:**
```
source/current/capture-agent/cloudstack/
├── client.go                 # CloudStack API client
├── vm_operations.go          # VM snapshot/control
├── host_discovery.go         # Find KVM hosts for VMs
└── ssh_manager.go            # Remote KVM host access
```

**Acceptance Criteria:**
- [ ] Can create/delete CloudStack VM snapshots
- [ ] Can identify which KVM host runs a VM
- [ ] Can SSH to KVM hosts and execute commands

---

### **Task 2: Capture Agent (CloudStack Variant)** (Week 1-2)

**Goal:** Deploy agent on KVM hosts for libvirt access

**Sub-Tasks:**
2.1. **Libvirt Integration**
   - Connect to local libvirt daemon
   - VM domain discovery and management
   - Block device path resolution
   
2.2. **Dirty Bitmap Management**
   - Create/manage QEMU dirty bitmaps
   - Query changed blocks
   - Clear bitmaps after successful backup
   
2.3. **NBD Export Service**
   - Export VM disks via qemu-nbd
   - Handle multiple concurrent exports
   - Automatic cleanup on completion

**Agent Architecture:**
```
Capture Agent (CloudStack) = 
├─ libvirt client (local connection)
├─ QEMU/QMP interface (dirty bitmaps)
├─ qemu-nbd server (disk export)
├─ SSH tunnel client (connect to Control Plane)
└─ Local API server (Control Plane commands)
```

**Files to Create:**
```
source/current/capture-agent/cloudstack-agent/
├── main.go                   # Agent entry point
├── libvirt_client.go         # libvirt operations
├── dirty_bitmap.go           # QEMU bitmap management
├── nbd_server.go             # qemu-nbd wrapper
├── api_server.go             # Local API for Control Plane
└── ssh_tunnel.go             # SSH tunnel management
```

**Acceptance Criteria:**
- [ ] Agent deploys on KVM host successfully
- [ ] Can create and query dirty bitmaps
- [ ] Can export VM disk via NBD
- [ ] SSH tunnel connects to Control Plane

---

### **Task 3: Backup Workflow (CloudStack)** (Week 2-3)

**Goal:** Orchestrate CloudStack backups from Control Plane

**Sub-Tasks:**
3.1. **VM Discovery**
   - Query CloudStack for VM list
   - Get VM specifications and disk info
   - Identify KVM host for each VM
   
3.2. **Full Backup Workflow**
   - Create CloudStack snapshot
   - Deploy/configure Capture Agent on KVM host
   - Create NBD export on agent
   - Stream to Control Plane backup repository
   - Clean up snapshot and exports
   
3.3. **Incremental Backup Workflow**
   - Query dirty bitmap for changed blocks
   - Create incremental QCOW2 with backing file
   - Transfer only changed blocks
   - Update backup chain metadata

**Workflow Comparison:**

| Step | VMware (Phase 1) | CloudStack (Phase 2) |
|------|------------------|----------------------|
| Discovery | vCenter API | CloudStack API |
| Change Tracking | CBT | Dirty Bitmaps |
| Consistency | VMware Tools snapshot | CloudStack VM snapshot |
| Agent Location | ESXi host or vCenter network | KVM hypervisor |
| NBD Export | From VMDK files | From QCOW2 files |
| SSH Tunnel | VMA → Control Plane | KVM Agent → Control Plane |

**Files to Create:**
```
source/current/control-plane/workflows/
├── cloudstack_backup.go      # Main CloudStack workflow
├── cloudstack_discovery.go   # VM and host discovery
└── dirty_bitmap_tracker.go   # Change tracking logic
```

**Acceptance Criteria:**
- [ ] Can discover CloudStack VMs
- [ ] Full backup transfers all data correctly
- [ ] Incremental backup only transfers changed blocks
- [ ] Backup chains tracked properly

---

### **Task 4: Multi-Platform Repository** (Week 3)

**Goal:** Extend backup repository for multiple source platforms

**Sub-Tasks:**
4.1. **Platform-Agnostic Metadata**
   - Extend VM metadata to include source platform
   - Track platform-specific identifiers (VM UUID, instance ID)
   - Handle different disk formats (VMDK vs QCOW2)
   
4.2. **Mixed Platform Support**
   - VMware VMs and CloudStack VMs in same repository
   - Platform-aware restore (know original source)
   - Cross-platform restore planning
   
4.3. **Repository Enhancements**
   - Storage usage by platform
   - Performance metrics by source type
   - Platform-specific retention policies

**Extended Metadata Schema:**
```json
{
  "vm_uuid": "4205784a-098a-40f1-1f1e-a5cd2597fd59",
  "vm_name": "database-prod",
  "source_platform": "cloudstack",
  "platform_metadata": {
    "cloudstack": {
      "vm_id": "4205784a-098a-40f1-1f1e-a5cd2597fd59",
      "service_offering": "Medium Instance",
      "zone": "zone1",
      "template": "Ubuntu 22.04"
    }
  },
  "disks": [
    {
      "disk_id": 0,
      "size_gb": 50,
      "format": "qcow2",
      "device_path": "/var/lib/libvirt/images/vm-disk0.qcow2"
    }
  ]
}
```

**Acceptance Criteria:**
- [ ] Can store VMware and CloudStack backups
- [ ] Metadata includes platform information
- [ ] GUI shows source platform for each backup

---

### **Task 5: Agent Deployment System** (Week 4)

**Goal:** Automated deployment of Capture Agent on KVM hosts

**Sub-Tasks:**
5.1. **Discovery & Deployment**
   - Auto-discover KVM hosts from CloudStack
   - Check agent status on each host
   - Deploy agent binary if needed
   - Update agent to newer versions
   
5.2. **Configuration Management**
   - Generate SSH keys for Control Plane access
   - Configure libvirt permissions
   - Set up systemd service
   - Health monitoring
   
5.3. **Multi-Host Management**
   - Deploy to multiple KVM hosts simultaneously
   - Load balancing (route VMs to least busy agents)
   - Failover (if one host agent fails)

**Deployment Script:**
```bash
#!/bin/bash
# deploy-cloudstack-agent.sh

KVM_HOST=$1
CONTROL_PLANE_IP=$2

# 1. Copy agent binary
scp sendense-capture-agent root@$KVM_HOST:/usr/local/bin/

# 2. Install systemd service
scp cloudstack-agent.service root@$KVM_HOST:/etc/systemd/system/

# 3. Generate SSH keys for tunnel
ssh root@$KVM_HOST "ssh-keygen -t ed25519 -f /opt/sendense/tunnel_key"

# 4. Configure libvirt access
ssh root@$KVM_HOST "usermod -a -G libvirt sendense"

# 5. Start service
ssh root@$KVM_HOST "systemctl enable --now sendense-capture-agent"
```

**Files to Create:**
```
source/current/control-plane/deployment/
├── cloudstack_deployment.go   # Auto-deployment logic
├── agent_manager.go           # Multi-agent management
└── health_monitor.go          # Agent health checking
```

**Acceptance Criteria:**
- [ ] Can deploy agent to multiple KVM hosts
- [ ] Automatic SSH key setup
- [ ] Agent health monitoring
- [ ] Graceful handling of agent failures

---

### **Task 6: Testing & Validation** (Week 4-5)

**Goal:** Comprehensive testing with CloudStack environments

**Test Scenarios:**

6.1. **CloudStack Integration Test**
   - Test with Apache CloudStack 4.18+
   - Test with various hypervisors (KVM/RHEL, KVM/Ubuntu)
   - Test with different storage types (NFS, local, Ceph)
   
6.2. **Multi-VM Backup Test**
   - Backup 10+ VMs simultaneously
   - Mix of small (10 GB) and large (500 GB) VMs
   - Verify no resource conflicts
   
6.3. **Incremental Efficiency Test**
   - Full backup → 10% data change → incremental
   - Verify only 10% transferred
   - Test incremental chain of 10+ backups
   
6.4. **Cross-Platform Repository Test**
   - VMware backups and CloudStack backups in same repo
   - Verify metadata separation
   - Test GUI shows both platform types

**Performance Targets:**
- Full backup: 3+ GiB/s throughput
- Incremental backup: 95%+ data reduction
- Agent overhead: <5% CPU on KVM host
- Concurrent backups: 10+ VMs per KVM host

**Acceptance Criteria:**
- [ ] All integration tests pass
- [ ] Performance targets met
- [ ] No impact on running CloudStack VMs
- [ ] Backup data integrity verified

---

## 💾 Database Schema Extensions

### **CloudStack-Specific Tables**

```sql
-- Migration: 20251101000001_add_cloudstack_support.up.sql

-- Extend backup_jobs for CloudStack
ALTER TABLE backup_jobs 
ADD COLUMN source_platform ENUM('vmware', 'cloudstack', 'hyperv', 'aws', 'azure', 'nutanix') DEFAULT 'vmware',
ADD COLUMN platform_vm_id VARCHAR(255) NULL,
ADD COLUMN kvm_host_id VARCHAR(255) NULL,
ADD INDEX idx_source_platform (source_platform);

-- CloudStack-specific metadata
CREATE TABLE cloudstack_vms (
    id VARCHAR(64) PRIMARY KEY,
    vm_context_id VARCHAR(191) NOT NULL,
    cloudstack_vm_id VARCHAR(255) NOT NULL,
    vm_name VARCHAR(255) NOT NULL,
    service_offering VARCHAR(255),
    template VARCHAR(255),
    zone VARCHAR(255),
    state ENUM('Running', 'Stopped', 'Destroyed') NOT NULL,
    kvm_host_id VARCHAR(255),
    kvm_host_ip VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE,
    UNIQUE KEY unique_cloudstack_vm (cloudstack_vm_id),
    INDEX idx_vm_context (vm_context_id)
);

-- KVM host agent management
CREATE TABLE cloudstack_agents (
    id VARCHAR(64) PRIMARY KEY,
    host_id VARCHAR(255) NOT NULL,
    host_ip VARCHAR(45) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    agent_version VARCHAR(50),
    status ENUM('active', 'inactive', 'error') NOT NULL DEFAULT 'inactive',
    last_heartbeat TIMESTAMP NULL,
    ssh_key_fingerprint VARCHAR(255),
    capabilities JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_host (host_id),
    INDEX idx_status (status),
    INDEX idx_heartbeat (last_heartbeat)
);
```

---

## 🌟 Strategic Benefits

### **1. Bidirectional CloudStack Support**

**Before Phase 2:**
- VMware → CloudStack (replication only)
- No CloudStack backup capability

**After Phase 2:**
- CloudStack → Backup Repository (descend)
- CloudStack ← Backup Repository (ascend)
- **Future:** CloudStack → VMware (transcend - reverse replication)

### **2. Complete Platform Coverage**

| Operation | VMware | CloudStack |
|-----------|--------|------------|
| **Backup** (descend) | ✅ Phase 1 | ✅ Phase 2 |
| **Restore** (ascend) | ✅ Phase 1 | ✅ Phase 2 |
| **Replication** (transcend) | VMware → CloudStack ✅ | CloudStack → VMware 🔜 |

### **3. Competitive Advantage**

**Market Gap:**
- Veeam: No CloudStack support
- Bacula/Bareos: Complex, no GUI
- Native CloudStack backup: Basic, no enterprise features

**Sendense Advantage:**
- Modern GUI for CloudStack backup
- Enterprise features (retention, encryption, compliance)
- Cross-platform capability (CloudStack ↔ VMware)

---

## 🎯 Success Metrics

### **Functional Success**
- ✅ CloudStack VM backup completes
- ✅ Dirty bitmap incrementals work
- ✅ Multi-KVM host deployment successful
- ✅ Mixed platform repository functional

### **Performance Success**
- ✅ Throughput: 3+ GiB/s
- ✅ Incremental efficiency: 95%+ reduction
- ✅ Agent overhead: <5% CPU
- ✅ Concurrent backups: 10+ VMs

### **Business Success**
- ✅ Can demo complete VMware ↔ CloudStack story
- ✅ Unique market position (only vendor with both)
- ✅ Foundation for reverse replication (CloudStack → VMware)

---

## 🚀 Deployment Strategy

### **Phase 2A: Single KVM Host** (Week 3)
- Deploy to one KVM host
- Test basic backup functionality
- Validate integration

### **Phase 2B: Multi-Host** (Week 4)
- Deploy to 3-5 KVM hosts
- Test load balancing
- Validate concurrent operations

### **Phase 2C: Production** (Week 5)
- Full CloudStack cluster deployment
- Customer testing and feedback
- Performance optimization

---

## 📚 Documentation Deliverables

1. **CloudStack Integration Guide**
   - KVM host requirements
   - Agent deployment procedures
   - Troubleshooting steps

2. **API Documentation**
   - CloudStack-specific endpoints
   - Dirty bitmap management APIs
   - Multi-platform backup APIs

3. **Admin Guide**
   - Managing CloudStack agents
   - Multi-platform repositories
   - Performance tuning

---

## 🔗 Dependencies & Next Steps

**Dependencies:**
- Phase 1 (VMware Backup) completed ✅
- CloudStack test environment available
- KVM hosts with libvirt accessible

**Enables Future Phases:**
- **Phase 4:** Cross-platform restore (CloudStack backup → VMware restore)
- **Phase 5:** Reverse replication (CloudStack → VMware transcend)
- **Phase 6:** Application-aware restores for Linux VMs

**Next Phase:**
→ **Phase 3: GUI Redesign** (Modern backup dashboard for multiple platforms)

---

**Phase Owner:** Platform Engineering Team  
**Last Updated:** October 4, 2025  
**Status:** 🟡 Planned - Awaiting Phase 1 Completion

