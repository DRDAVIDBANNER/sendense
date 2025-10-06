# Sendense Terminology & Naming Conventions

**Document Version:** 1.0  
**Last Updated:** October 4, 2025

---

## 🎯 Core Concepts

### **Data Flow Operations**

#### **descend** 📥
- **Definition:** Data flowing DOWN into backup storage
- **Direction:** Production VM → Backup Repository
- **Use Cases:**
  - VMware VM → Local disk backup
  - CloudStack VM → S3 backup
  - Hyper-V VM → Azure Blob backup
- **Example:** "Descending pgtest2 VM to S3 repository"

#### **ascend** 📤
- **Definition:** Data flowing UP from backup storage
- **Direction:** Backup Repository → Running VM
- **Use Cases:**
  - S3 backup → VMware restore
  - Local disk backup → CloudStack restore
  - Azure Blob backup → Hyper-V restore
- **Example:** "Ascending database backup to production CloudStack"

#### **transcend** 🌉
- **Definition:** Data CROSSING platform boundaries (the premium feature)
- **Direction:** Platform A → Platform B (continuous replication)
- **Use Cases:**
  - VMware → CloudStack (live replication)
  - CloudStack → VMware (reverse replication)
  - Hyper-V → AWS EC2 (migration)
  - VMware → Azure (cloud migration)
- **Example:** "Transcending from VMware to CloudStack with near-live replication"
- **Note:** This is the $100/VM premium tier feature

---

## 🖥️ System Components

### **Legacy Names (Deprecated)**

- **VMA** (VMware Migration Appliance) ❌
  - Too VMware-specific
  - "Migration" doesn't cover backup use case
  
- **OMA** (OSSEA Migration Appliance) ❌
  - Too CloudStack-specific
  - "Migration" doesn't cover backup use case

### **New Names (Active)**

#### **Capture Agent** (or "Sendense Capture")
- **Abbreviation:** SCA
- **Purpose:** Runs on or near source systems to capture data
- **Locations:**
  - VMware: Runs on ESXi host or management network
  - CloudStack: Runs on KVM hypervisor
  - Hyper-V: Runs on Hyper-V host
  - Physical: Agent installed on server
  
- **CLI:** `sendense-capture` or `sca`
- **Binary:** `sendense-capture-agent`
- **Service:** `sendense-capture.service`

**Responsibilities:**
- Connect to source platform APIs (vCenter, libvirt, Hyper-V)
- Read changed blocks via CBT/dirty bitmaps
- Stream data via NBD protocol
- Track progress and report status
- Run source-side benchmarks

#### **Control Plane** (or "Sendense Control")
- **Abbreviation:** SCP
- **Purpose:** Central orchestration, storage, and management
- **Locations:**
  - On-prem (dedicated appliance or VM)
  - Cloud (AWS EC2, Azure VM)
  - Hybrid (multi-location for DR)

- **CLI:** `sendense-control` or `scp`
- **Binary:** `sendense-control-plane`
- **Service:** `sendense-control.service`

**Responsibilities:**
- Orchestrate backup/restore/replication jobs
- Manage backup repositories (local, S3, Azure)
- Volume lifecycle management (create/attach/detach)
- Job scheduling and retention policies
- GUI web server
- API endpoints
- Run target-side benchmarks

---

## 📊 Job Types

### **Backup Job**
- **Flow:** descend operation
- **Type:** Point-in-time snapshot
- **Frequency:** Hourly, daily, weekly (periodic)
- **Target:** Backup repository (QCOW2, S3, etc.)
- **RTO:** Hours (restore time + boot time)
- **RPO:** Backup interval (e.g., 24 hours)

### **Restore Job**
- **Flow:** ascend operation
- **Type:** Recover from backup
- **Frequency:** On-demand (disaster recovery)
- **Source:** Backup repository
- **Target:** Original or different platform (Enterprise tier)
- **Sub-Types:**
  - **Full VM Restore:** Boot entire VM
  - **File-Level Restore:** Extract specific files
  - **Application Restore:** Database/mailbox/AD object

### **Replication Job**
- **Flow:** transcend operation
- **Type:** Continuous sync (near-live)
- **Frequency:** Every 5-15 minutes (continuous)
- **Target:** Live VM on different platform
- **RTO:** 5-15 minutes (just boot target VM)
- **RPO:** 1-15 minutes (last incremental sync)
- **Premium Feature:** $100/VM/month

---

## 🔄 Change Tracking Technologies

### **CBT (Changed Block Tracking)**
- **Platform:** VMware vSphere
- **Mechanism:** VMware's built-in change tracking
- **Granularity:** Typically 256KB blocks
- **API:** vSphere VDDK
- **Status:** Production-ready in Sendense

### **Dirty Bitmaps**
- **Platform:** CloudStack/KVM (QEMU)
- **Mechanism:** QEMU block dirty tracking
- **Granularity:** Configurable (typically 64KB)
- **API:** libvirt + QMP (QEMU Machine Protocol)
- **Status:** Phase 2 implementation

### **RCT (Resilient Change Tracking)**
- **Platform:** Microsoft Hyper-V
- **Mechanism:** Hyper-V's built-in change tracking
- **Granularity:** 64KB blocks
- **API:** WMI / PowerShell
- **Status:** Planned (Phase 4)

### **EBS Changed Blocks**
- **Platform:** AWS EC2
- **Mechanism:** EBS snapshot diff API
- **Granularity:** 512KB blocks
- **API:** AWS SDK (ListChangedBlocks)
- **Status:** Planned (Phase 5)

---

## 💾 Storage Formats

### **QCOW2 (QEMU Copy-On-Write v2)**
- **Use:** Primary backup format for local storage
- **Benefits:**
  - Native backing file support (incremental chains)
  - Compression support
  - Encryption support
  - Industry standard (portable)
- **Backing Chain Example:**
  ```
  full-2025-10-01.qcow2 (40 GB)
    ↑ backing file
  incr-2025-10-02.qcow2 (2 GB, changes only)
    ↑ backing file
  incr-2025-10-03.qcow2 (1.5 GB, changes only)
  ```

### **VHD/VHDX**
- **Use:** Hyper-V native format
- **Benefits:**
  - Direct Hyper-V import
  - Differencing disk support (like backing files)

### **VMDK**
- **Use:** VMware native format
- **Benefits:**
  - Direct vCenter import
  - Sparse file support

### **Raw + Metadata**
- **Use:** AWS/Azure (no QCOW2 support)
- **Format:** Raw disk + JSON metadata for chain tracking

---

## 🏷️ Naming Conventions

### **Backup File Names**

```
{vm-name}-{type}-{timestamp}.qcow2

Examples:
- pgtest2-full-20251004-120000.qcow2
- pgtest2-incr-20251004-180000.qcow2
- database-prod-full-20251004-000000.qcow2
```

### **Repository Paths**

```
/var/lib/sendense/backups/{vm-uuid}/disk-{N}/{backup-file}

Example:
/var/lib/sendense/backups/
  ├─ 4205784a-098a-40f1-1f1e-a5cd2597fd59/
  │  ├─ metadata.json
  │  ├─ disk-0/
  │  │  ├─ pgtest2-full-20251001-120000.qcow2
  │  │  ├─ pgtest2-incr-20251002-120000.qcow2
  │  │  └─ chain.json
  │  └─ disk-1/
  │     └─ pgtest2-disk1-full-20251001-120000.qcow2
```

### **Job IDs**

```
{type}-{vm-name}-{timestamp}

Examples:
- backup-pgtest2-20251004120000
- restore-database-prod-20251004183000
- replication-exchange-server-20251004090000
```

---

## 🌐 Network Architecture

### **SSH Tunnel**
- **Port:** 443 (HTTPS, firewall-friendly)
- **Purpose:** All Capture Agent → Control Plane traffic
- **Security:** Ed25519 keys, restricted forwarding
- **Protocols Inside Tunnel:**
  - NBD (data streaming)
  - HTTP/REST (control/status)

### **NBD (Network Block Device)**
- **Port:** 10809 (inside SSH tunnel)
- **Purpose:** Block-level data streaming
- **Performance:** 3.2 GiB/s proven throughput
- **Protocol:** NBD protocol over TCP

---

## 📈 Performance Metrics

### **Backup Performance**
- **Full Backup:** Source disk size (e.g., 100 GB)
- **Incremental Backup:** Changed blocks only (e.g., 5 GB)
- **Compression Ratio:** Typically 1.5-3x (depends on data)
- **Deduplication:** Filesystem-level (ZFS, XFS)

### **Restore Performance**
- **Sequential Restore:** ~3.2 GiB/s (NBD throughput)
- **Random Restore:** Depends on target storage IOPS
- **Instant Restore:** Mount backup as live disk (QCOW2 NBD export)

### **Replication Performance**
- **Sync Interval:** 5-15 minutes (configurable)
- **Bandwidth:** Depends on change rate and network
- **RTO:** 5-15 minutes (target VM boot time)
- **RPO:** Sync interval (1-15 minutes typical)

---

## 🎯 Product Tiers (Quick Reference)

| Tier | Price | Key Feature | Flow Type |
|------|-------|-------------|-----------|
| **Backup Edition** | $10/VM/month | Same-platform restore | descend + ascend (same platform) |
| **Enterprise Edition** | $25/VM/month | Cross-platform restore | descend + ascend (any platform) |
| **Replication Edition** | $100/VM/month | Near-live replication | transcend (continuous) |

---

## 🔗 Relationship to Legacy Terms

**When reading old documentation:**

| Old Term | New Term | Notes |
|----------|----------|-------|
| VMA | Capture Agent | Source-side agent |
| OMA | Control Plane | Central orchestrator |
| Migration | Replication or Backup | Context-dependent |
| CloudStack | Any target platform | No longer specific to CloudStack |
| OSSEA | Target platform | Generic term now |

---

## 📚 Glossary

**Backup Chain**: Series of incremental backups linked to a full backup
**Backing File**: QCOW2 concept - parent image for incremental
**CBT**: Changed Block Tracking (VMware's term)
**Dirty Bitmap**: QEMU's change tracking mechanism
**RTO**: Recovery Time Objective (how fast can you restore)
**RPO**: Recovery Point Objective (how much data can you lose)
**Failover**: Switch from source to target (disaster event)
**Failback**: Switch from target back to source (after disaster resolved)
**Transcend**: Cross-platform replication (Sendense premium feature)

---

**Document Owner:** Documentation Team  
**Review Cycle:** Monthly or on terminology changes  
**Last Reviewed:** October 4, 2025


