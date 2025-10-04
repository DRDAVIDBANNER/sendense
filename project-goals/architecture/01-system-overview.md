# Sendense System Architecture Overview

**Document Version:** 1.0  
**Last Updated:** October 4, 2025

---

## 🎯 System Purpose

Sendense is a universal backup and replication platform designed to:
1. **Backup** from any virtualization platform or cloud
2. **Restore** to any target platform (cross-platform capability)
3. **Replicate** VMs with near-zero RTO/RPO across platforms
4. **Break vendor lock-in** with open architecture and standards

---

## 🏗️ High-Level Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                     SENDENSE PLATFORM                              │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                   WEB UI LAYER                               │ │
│  │  • Modern React/Next.js Dashboard                            │ │
│  │  • Backup Job Management                                     │ │
│  │  • Restore Wizard (Cross-Platform)                           │ │
│  │  • Repository Configuration                                  │ │
│  │  • Monitoring & Reporting                                    │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                          ↕ REST API                                │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │              ORCHESTRATION LAYER (OMA)                       │ │
│  │  • Job Scheduler & Policy Engine                             │ │
│  │  • Retention Management                                      │ │
│  │  • Volume Management Daemon Integration                      │ │
│  │  • Progress Tracking (JobLog)                                │ │
│  │  • Multi-Tenancy & RBAC                                      │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                          ↕                                         │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                  CONNECTOR LAYER                             │ │
│  │                                                              │ │
│  │  ┌────────────────┐  ┌──────────────┐  ┌─────────────────┐ │ │
│  │  │ SOURCE         │  │ STORAGE      │  │ TARGET          │ │ │
│  │  │ CONNECTORS     │  │ BACKENDS     │  │ CONNECTORS      │ │ │
│  │  │                │  │              │  │                 │ │ │
│  │  │ • VMware ✅    │  │ • Local Disk │  │ • CloudStack ✅ │ │ │
│  │  │ • CloudStack   │  │ • S3/Wasabi  │  │ • VMware        │ │ │
│  │  │ • Hyper-V      │  │ • Azure Blob │  │ • Hyper-V       │ │ │
│  │  │ • AWS EC2      │  │ • Backblaze  │  │ • AWS EC2       │ │ │
│  │  │ • Azure VM     │  │ • Immutable  │  │ • Azure         │ │ │
│  │  │ • Physical     │  │ • NFS/iSCSI  │  │ • Physical      │ │ │
│  │  └────────────────┘  └──────────────┘  └─────────────────┘ │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                          ↕                                         │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │              APPLICATION-AWARE LAYER                         │ │
│  │  • SQL Server (Database/Table/Transaction level)             │ │
│  │  • Active Directory (Domain Controller/Object level)         │ │
│  │  • Exchange Server (Mailbox/Item level)                      │ │
│  │  • Oracle, MySQL, PostgreSQL, MongoDB                        │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

---

## 🔄 Data Flow Architecture

### **Backup Flow**

```
┌──────────────┐
│ Source VM    │ (VMware, CloudStack, Hyper-V, etc.)
└──────┬───────┘
       │ Change tracking (CBT/Dirty Bitmap)
       ↓
┌──────────────┐
│ Source Agent │ (VMA for VMware, KVM agent for CloudStack)
└──────┬───────┘
       │ Block-level read (changed blocks only)
       ↓
┌──────────────┐
│ NBD Stream   │ (Encrypted SSH tunnel, port 443)
└──────┬───────┘
       │ 3.2 GiB/s throughput
       ↓
┌──────────────┐
│ OMA          │ (Orchestration Management Appliance)
└──────┬───────┘
       │ Repository abstraction
       ↓
┌──────────────┐
│ Backup Store │ (QCOW2, S3, Azure Blob, etc.)
└──────────────┘
```

### **Restore Flow**

```
┌──────────────┐
│ Backup Store │ (Source: QCOW2 chain, S3 object, etc.)
└──────┬───────┘
       │ Read backup chain
       ↓
┌──────────────┐
│ OMA          │ (Format conversion if needed)
└──────┬───────┘
       │ Stream or mount
       ↓
┌──────────────┐
│ Target       │ (CloudStack volume, VMDK, VHDX, EBS, etc.)
│ Connector    │
└──────┬───────┘
       │ Platform-specific import
       ↓
┌──────────────┐
│ Running VM   │ (Original platform OR different platform)
└──────────────┘
```

### **Replication Flow (Near-Live)**

```
┌──────────────┐
│ Source VM    │ (Running in production)
└──────┬───────┘
       │ Continuous change tracking (5-15 min sync)
       ↓
┌──────────────┐
│ Source Agent │ (Captures changed blocks in real-time)
└──────┬───────┘
       │ Incremental NBD stream
       ↓
┌──────────────┐
│ Target VM    │ (Replica kept up-to-date, ready for failover)
└──────────────┘

Failover Event:
  → Stop source VM (optional)
  → Final sync (changed blocks since last sync)
  → Power on target VM
  → RTO: 5-15 minutes, RPO: 1-15 minutes
```

---

## 🧩 Core Components

### **1. Source Connectors**

**Purpose:** Read data from source platforms using native change tracking

**VMware Connector (Existing ✅):**
- VDDK/nbdkit integration
- CBT (Changed Block Tracking) for incrementals
- VSS integration for application consistency
- Location: `source/current/vma/`

**CloudStack Connector (Phase 2):**
- Libvirt API integration
- Dirty bitmap tracking (QEMU)
- Agent on KVM hosts
- Location: `source/current/cloudstack-agent/` (new)

**Future Connectors:**
- Hyper-V: RCT (Resilient Change Tracking)
- AWS EC2: EBS snapshots + changed block tracking
- Azure VM: Managed disk snapshots
- Physical servers: Agent-based (block or file level)

### **2. Storage Backends**

**Purpose:** Abstract storage layer - any backup target

**Interface:**
```go
type BackupRepository interface {
    Write(backupID string, data io.Reader) error
    Read(backupID string) (io.Reader, error)
    Delete(backupID string) error
    List(filters BackupFilters) ([]BackupMetadata, error)
    SetImmutable(backupID string, duration time.Duration) error
}
```

**Implementations:**
- **Local Repository**: QCOW2 files with backing chains
- **S3 Repository**: AWS S3, Wasabi, Backblaze B2, MinIO
- **Azure Repository**: Azure Blob Storage (Hot/Cool/Archive)
- **Immutable Repository**: Wrapper for WORM compliance

**Location:** `source/current/oma/storage/` (new)

### **3. Target Connectors**

**Purpose:** Write/import backups to target platforms

**CloudStack Target (Existing ✅):**
- Volume Daemon integration
- CloudStack API for VM creation
- Device correlation and management
- Location: `source/current/oma/` (Volume Daemon integrated)

**Future Targets:**
- VMware: vCenter API, OVF/VMDK import
- Hyper-V: WMI/PowerShell, VHDX import
- AWS EC2: AMI creation, EC2 launch
- Azure: VHD upload, managed disk, VM creation

### **4. Orchestration Layer (OMA)**

**Purpose:** Central control plane for all operations

**Components:**
- **Job Scheduler**: Cron-based scheduling, retention policies
- **Volume Management**: Volume Daemon integration (existing)
- **Progress Tracking**: JobLog system (existing)
- **API Layer**: REST API for GUI and integrations
- **Database**: VM-centric schema with CASCADE DELETE (existing)

**Location:** `source/current/oma/`

### **5. Application-Aware Processing**

**Purpose:** Deep integration for application-level restores

**Capabilities:**
- **SQL Server**: Database/table/transaction log restores
- **Active Directory**: DC restore, object-level recovery
- **Exchange**: Mailbox/item-level recovery
- **General**: VSS integration, quiesced snapshots

**Location:** `source/current/oma/application-aware/` (new)

---

## 🛡️ Security Architecture

### **Data in Transit**
- **SSH Tunnel**: All traffic over port 443 (existing)
- **TLS Encryption**: End-to-end encryption
- **Ed25519 Keys**: Modern cryptography
- **Restricted SSH**: No PTY, no X11, port forwarding only

### **Data at Rest**
- **Encryption**: Repository-level encryption (AES-256)
- **Immutability**: S3 Object Lock, WORM storage
- **Retention**: Automated retention policies
- **Compliance**: HIPAA, SOC2, GDPR support

### **Access Control**
- **RBAC**: Role-based access control
- **Multi-Tenancy**: Customer isolation for MSP mode
- **Audit Logging**: Complete audit trail (existing JobLog)
- **API Authentication**: Token-based auth with expiration

---

## 📊 Database Architecture

### **VM-Centric Schema (Existing)**

```
vm_replication_contexts (Master table)
  ├─ replication_jobs (Migration jobs)
  ├─ backup_jobs (NEW - Backup jobs)
  ├─ vm_disks (Disk metadata + change IDs)
  ├─ failover_jobs (Failover tracking)
  └─ job_tracking (JobLog integration)

backup_specific tables (NEW):
  ├─ backup_chains (Incremental backup relationships)
  ├─ backup_repositories (Storage backend configs)
  ├─ restore_jobs (Restore operation tracking)
  └─ application_restores (SQL/AD/Exchange restores)

CASCADE DELETE ensures cleanup when VM context is removed
```

**Key Features:**
- VM-centric architecture (existing)
- CASCADE DELETE relationships (existing)
- Foreign key constraints (existing)
- JobLog integration (existing)

---

## 🔧 Technology Stack

### **Backend**
- **Language**: Go 1.21+
- **APIs**: REST (existing), future GraphQL for GUI
- **Database**: MariaDB (existing schema)
- **Orchestration**: systemd services (existing)
- **Tunneling**: SSH (existing infrastructure)

### **Frontend**
- **Framework**: React 18+ / Next.js 14+
- **UI Library**: Tailwind CSS + Shadcn/ui (modern)
- **State Management**: React Query + Zustand
- **Charts**: Recharts or Chart.js
- **Real-time**: WebSocket for live updates

### **Storage**
- **File Format**: QCOW2 (native backing files)
- **Cloud**: S3 SDK (AWS SDK v3)
- **Compression**: zstd or lz4
- **Deduplication**: Filesystem-level (ZFS, XFS)

### **Virtualization APIs**
- **VMware**: VDDK, vCenter SOAP/REST (existing)
- **CloudStack**: CloudStack API + libvirt (partial existing)
- **Hyper-V**: WMI, PowerShell, Hyper-V WMI Provider
- **Cloud**: AWS SDK, Azure SDK

---

## 🚀 Scalability

### **Horizontal Scaling**
- Multiple OMA instances (load balancing)
- Multiple VMA/agent instances per source
- Distributed backup repositories
- Cloud-native architecture

### **Performance Targets**
- **Throughput**: 3.2 GiB/s (existing, maintain)
- **Concurrent Jobs**: 50+ per OMA
- **VM Scale**: 1,000+ VMs per OMA
- **Repository Size**: Unlimited (cloud storage)

---

## 🎯 Design Principles

### **1. Reuse Existing Architecture**
Don't reinvent the wheel. Leverage:
- NBD streaming engine (3.2 GiB/s proven)
- SSH tunnel infrastructure (secure, tested)
- Volume Daemon (centralized volume management)
- JobLog system (tracking and auditing)
- Database schema (VM-centric, normalized)

### **2. Abstraction Layers**
- **Storage Backend Interface**: Write once, support many targets
- **Source Connector Interface**: Standardized read operations
- **Target Connector Interface**: Standardized write operations
- **Repository Interface**: Any storage backend

### **3. Modularity**
- Independent modules (source, storage, target)
- Plugin architecture for new platforms
- API-first design
- Microservices where appropriate

### **4. Open Standards**
- QCOW2 (not proprietary formats)
- S3 API (not vendor-locked storage)
- Standard protocols (NBD, SSH, HTTP)
- Open-source friendly

### **5. MSP-Ready**
- Multi-tenant from day one
- White-label capability
- Usage metering and billing hooks
- Centralized management

---

## 📚 Related Documents

- **Technical Stack**: `02-technical-stack.md`
- **Storage Abstraction**: `03-storage-abstraction.md`
- **Product Editions**: `../editions/01-product-tiers.md`
- **Phase 1 Plan**: `../phases/phase-1-vmware-backup.md`

---

**Document Owner:** Architecture Team  
**Review Cycle:** Quarterly or on major changes  
**Last Reviewed:** October 4, 2025

