# Sendense - Project Goals & Roadmap

**Vision:** Universal backup and replication platform that breaks vendor lock-in

**Mission:** Backup from anything, restore to anything, replicate across any platform

---

## 📂 Documentation Structure

```
project-goals/
├── README.md (this file)
├── TERMINOLOGY.md                     # 🔥 NEW: descend/ascend/transcend naming
├── architecture/
│   ├── 01-system-overview.md          # High-level system architecture
│   └── 02-msp-cloud-architecture.md   # 🔥 NEW: MSP cloud + bulletproof licensing
├── editions/
│   ├── 01-product-tiers.md            # Backup ($10), Enterprise ($25), Replication ($100)
│   └── 02-competitive-analysis.md     # 🔥 NEW: vs Veeam, PlateSpin, Carbonite
├── modules/
│   ├── 01-vmware-source.md            # ✅ COMPLETE: VMware source (CBT)
│   ├── 02-cloudstack-source.md        # CloudStack source (libvirt dirty bitmaps)
│   ├── 03-backup-repository.md        # 🔥 NEW: Storage abstraction (QCOW2, S3, immutable)
│   ├── 04-restore-engine.md           # 🔥 NEW: Cross-platform restore (Enterprise enabler)
│   ├── 05-nutanix-source.md           # Nutanix AHV source (Prism API)
│   ├── 06-hyperv-source.md            # 🔥 NEW: Hyper-V source (RCT)
│   ├── 07-aws-source.md               # 🔥 NEW: AWS EC2 source (EBS CBT)
│   ├── 08-performance-benchmarking.md # Source vs target performance validation
│   ├── 09-backup-validation.md        # 🔥 NEW: Auto-validate backups by booting VMs
│   ├── 10-azure-source.md             # 🔥 NEW: Azure VM source (managed disk snapshots)
│   └── 12-licensing-system.md         # 🔥 NEW: Bulletproof licensing + anti-piracy
└── phases/
    ├── phase-1-vmware-backup.md       # 🔴 START HERE: VMware backups (6 weeks)
    ├── phase-2-cloudstack-backup.md   # CloudStack/KVM backups
    ├── phase-3-gui-redesign.md        # Modern UI overhaul (waiting for your GUI plan)
    ├── phase-4-cross-platform-restore.md # Enterprise tier unlock ($25/VM)
    ├── phase-5-multi-platform-replication.md # Premium tier ($100/VM) 💰
    ├── phase-6-application-aware-restores.md # SQL/AD/Exchange granular
    └── phase-7-msp-platform.md        # MSP control plane (scalable business)
```

---

## 🎯 Current Status

**Base Platform:** MigrateKit OSSEA (VMware → CloudStack migration)
- ✅ VMware source integration (VDDK, CBT tracking)
- ✅ NBD streaming (3.2 GiB/s encrypted over SSH tunnel)
- ✅ CloudStack target (Volume Daemon integration)
- ✅ Volume lifecycle management
- ✅ SSH tunnel infrastructure (port 443)
- ✅ Progress tracking and monitoring
- ✅ Multi-disk VM support
- ✅ Database schema (VM-centric architecture)

**Reusable Components:**
- NBD streaming engine
- SSH tunnel security
- Volume management
- Job tracking system
- Database schema
- API framework

---

## 🚀 Execution Order

### **Immediate (Q4 2025)**
1. ✅ **Phase 1: VMware Backups** - START HERE
   - File-based backup repository
   - Full + incremental backups (CBT-based)
   - Local storage backend
   - File-level restore capability

2. **Phase 2: CloudStack Backups**
   - Libvirt dirty bitmap integration
   - Agent deployment on KVM hosts
   - Incremental backup support

3. **Phase 3: GUI Redesign**
   - Modern backup dashboard
   - Job monitoring and management
   - Restore wizard
   - Repository configuration

### **Near-Term (Q1 2026)**
4. **Phase 4: Restore Engine**
   - Cross-platform restore (VMware → CloudStack, etc.)
   - Storage backend expansion (S3, Azure Blob)
   - Immutable storage support

5. **Phase 5: Replication Engine**
   - Hyper-V replication target
   - AWS EC2 replication target
   - Failover automation

### **Mid-Term (Q2-Q3 2026)**
6. **Phase 6: Application-Aware**
   - SQL Server restores (database/table level)
   - Active Directory restores (object level)
   - Exchange Server restores (mailbox level)

7. **Phase 7: MSP Platform**
   - Multi-tenant control plane
   - White-label portal
   - Billing integration
   - Usage metering

---

## 💰 Revenue Projections

**Target Market Segments:**
1. **VMware → CloudStack Migrations** (Unique advantage)
2. **Multi-Platform Enterprises** (Premium tier)
3. **MSPs** (Recurring revenue model)
4. **Anti-Veeam Market** (Price + vendor neutrality)

**Pricing Tiers:**
- **Backup Edition**: $10/VM/month (Veeam replacement)
- **Enterprise Edition**: $25/VM/month (Cross-platform restore)
- **Replication Edition**: $100/VM/month (Near-live replication) 💰
- **MSP Platform**: $5/VM/month + $200/month base

**Conservative Year 1 Target:**
- 50 MSP customers
- Average 50 VMs per customer (2,500 VMs total)
- Mix: 60% Backup, 30% Enterprise, 10% Replication
- **ARR: ~$600K-800K**

---

## 🎯 Success Metrics

### **Phase 1 (VMware Backup)**
- ✅ Backup job completes successfully
- ✅ Incremental backup uses CBT (90%+ data reduction)
- ✅ File-level restore works
- ✅ Performance: 3.2 GiB/s throughput maintained

### **Phase 2 (CloudStack Backup)**
- ✅ Dirty bitmap tracking operational
- ✅ Incremental backups working
- ✅ Agent deployed successfully on KVM hosts

### **Phase 3 (GUI Redesign)**
- ✅ Modern dashboard replacing current UI
- ✅ Intuitive backup job creation
- ✅ Restore wizard for non-technical users

---

## 📚 Key Documentation

- **Architecture**: High-level system design and technical decisions
- **Editions**: Product tiers, pricing, and positioning
- **Modules**: Deep dives into each technical component
- **Phases**: Detailed project plans with tasks and timelines

---

## 🔗 Related Documents

- Current codebase: `source/current/`
- Existing architecture: `docs/architecture/`
- API documentation: `docs/api/`
- Database schema: `AI_Helper/VERIFIED_DATABASE_SCHEMA.md`

---

**Last Updated:** October 4, 2025  
**Current Phase:** Phase 1 - VMware Backups  
**Next Milestone:** Phase 2 - CloudStack Backups

