# Sendense Product Editions & Tiers

**Document Version:** 1.0  
**Last Updated:** October 4, 2025

---

## 🎯 Product Strategy

Sendense offers three distinct product tiers, each targeting different customer needs and price points:

1. **Backup Edition** - Standard backup/restore (Veeam replacement)
2. **Enterprise Edition** - Cross-platform disaster recovery flexibility
3. **Replication Edition** - Near-live cross-platform replication (PREMIUM 💰)

**Key Principle:** **Replication is separate from backup/restore**
- Backup/restore = Point-in-time recovery (hours RTO, backup interval RPO)
- Replication = Continuous sync (minutes RTO, seconds/minutes RPO)

---

## 📦 Tier 1: Backup Edition

**Tagline:** *"Protect your VMs, restore them back where they came from"*

**Price:** $10/VM/month

### **Features**

**Backup Sources:**
- ✅ VMware vSphere (CBT-based incrementals)
- ✅ CloudStack/KVM (Dirty bitmap incrementals)
- 🔜 Hyper-V (RCT-based incrementals)
- 🔜 AWS EC2 (EBS snapshot-based)
- 🔜 Azure VMs (Managed disk snapshots)
- 🔜 Physical servers (Agent-based)

**Restore Targets:**
- ✅ **Same platform only** (VMware → VMware, CloudStack → CloudStack, etc.)
- ❌ **NO cross-platform restore** (locked in Enterprise tier)

**Storage Backends:**
- Local disk (QCOW2 format with backing chains)
- S3-compatible (AWS S3, Wasabi, Backblaze B2, MinIO)
- Azure Blob Storage (Hot/Cool/Archive tiers)
- NFS/SMB network storage

**Restore Capabilities:**
- Full VM restore to original platform
- File-level restore (mount backup, browse, extract files)
- Application-aware snapshots (VSS for Windows)
- Point-in-time recovery (from any backup in chain)

**Management:**
- Backup scheduling (hourly, daily, weekly)
- Retention policies (days, weeks, months)
- Basic job monitoring
- Email notifications
- Single-tenant only

**Support:**
- Email support (24-48 hour response)
- Community forum
- Documentation and knowledge base

### **Target Customers**

- Small to medium businesses (10-100 VMs)
- Companies wanting to replace Veeam
- Single-platform environments (all VMware or all Hyper-V)
- Cost-conscious buyers
- Standard backup/DR requirements

### **Competitive Positioning**

**vs Veeam Community Edition:**
- ✅ Modern UI (not Windows 95 style)
- ✅ Linux-first architecture
- ✅ Cloud storage backends
- ✅ Same price point ($10/VM)

**vs Acronis:**
- ✅ More flexible storage options
- ✅ Open architecture
- ✅ Better performance (3.2 GiB/s)

---

## 📦 Tier 2: Enterprise Edition

**Tagline:** *"Disaster recovery flexibility - restore anywhere"*

**Price:** $25/VM/month

### **Features**

**Everything in Backup Edition PLUS:**

**Cross-Platform Restore 🔓:**
- VMware backup → CloudStack restore ✅
- VMware backup → Hyper-V restore
- VMware backup → AWS EC2 restore
- Hyper-V backup → VMware restore
- Hyper-V backup → CloudStack restore
- CloudStack backup → VMware restore
- **ANY backup → ANY platform restore**

**Important Distinction:**
- This is **point-in-time restore**, NOT continuous replication
- Uses last backup taken (hourly, daily, etc.)
- RTO: Hours (time to restore + boot VM)
- RPO: Backup interval (e.g., 4 hours if backing up every 4 hours)

**Advanced Storage:**
- Immutable backup storage (S3 Object Lock, WORM)
- Encryption at rest (AES-256)
- Geo-replication (multi-region)
- Backup verification and testing

**Compliance & Governance:**
- Compliance reporting (HIPAA, SOC2, GDPR)
- Audit logging and trail
- Legal hold capabilities
- Data residency controls

**Advanced RBAC:**
- Role-based access control
- Custom roles and permissions
- Per-VM access control
- Delegation

**Support:**
- Email + phone support (8-hour response)
- SLA options available
- Dedicated account manager (25+ VMs)

### **Target Customers**

- Medium to large enterprises (100-1,000 VMs)
- Multi-platform environments (VMware + CloudStack + Hyper-V)
- Companies planning platform migrations
- Disaster recovery flexibility requirements
- Compliance-driven organizations (healthcare, finance)

### **Use Cases**

**Scenario 1: Platform Flexibility**
- Production: VMware vSphere
- Backups: Daily to S3
- Disaster: vCenter failure, no spare hardware
- Solution: Restore to CloudStack or AWS EC2 from backup
- RTO: 2-4 hours (restore time + boot)

**Scenario 2: Cloud Migration Testing**
- Production: VMware on-prem
- Testing: Restore backups to AWS EC2 for testing
- No impact to production
- Plan migration with confidence

**Scenario 3: Cost Optimization**
- Production: Expensive VMware licensing
- DR: Cheaper CloudStack or Hyper-V for DR site
- Restore to cheaper platform during disasters

### **Competitive Positioning**

**vs Veeam Enterprise Plus:**
- ✅ True cross-platform restore (Veeam is limited)
- ✅ Lower cost ($25 vs $40-60/VM)
- ✅ Modern architecture

**vs Commvault:**
- ✅ Much simpler to deploy and manage
- ✅ Lower cost ($25 vs $50+/VM)
- ✅ No enterprise bloatware

---

## 📦 Tier 3: Replication Edition 💰

**Tagline:** *"Near-live cross-platform replication - zero downtime"*

**Price:** $100+/VM/month (premium tier)

### **Features**

**Everything in Enterprise Edition PLUS:**

**Continuous Cross-Platform Replication 🚀:**
- **VMware → CloudStack** ✅ (WORKING, production ready!)
- VMware → Hyper-V
- VMware → AWS EC2
- VMware → Azure
- Hyper-V → VMware
- Hyper-V → CloudStack
- Physical → Virtual (any target)

**Key Capabilities:**
- **CBT/Dirty Bitmap-based incremental sync** (every 5-15 minutes)
- **Target VM kept up-to-date continuously** (not just periodic backups)
- **One-click failover** with minimal data loss
- **Failback capability** (reverse replication after failover)
- **Test failover** without affecting production replication
- **RTO: 5-15 minutes** (just boot the already-synced target VM)
- **RPO: 1-15 minutes** (last incremental sync)

**Advanced Features:**
- Automated failover orchestration
- Network remapping during failover
- Application-consistent failovers
- Bandwidth throttling and scheduling
- WAN optimization
- Migration projects (permanent moves)

**Support:**
- 24/7 phone support
- 4-hour response SLA
- Dedicated technical account manager
- Migration assistance (professional services)

### **Replication vs Restore - Critical Difference**

#### **Cross-Platform Restore (Enterprise Tier)**
```
Backup Schedule: Every 4 hours

12:00 AM ─┐
04:00 AM  ├─ Backups taken periodically
08:00 AM  │
12:00 PM ─┘

Disaster at 3:00 PM:
  ├─ Data loss: 3 hours (since last backup at 12:00 PM)
  ├─ Restore from 12:00 PM backup
  ├─ Convert backup format to target platform (1-2 hours)
  └─ Boot VM on target platform

RTO: 2-4 hours
RPO: Backup interval (4 hours in this example)
```

#### **Cross-Platform Replication (Replication Tier)**
```
Continuous Sync: Every 5 minutes

12:00 AM ─┐
12:05 AM  │
12:10 AM  ├─ Incremental syncs (changed blocks only)
12:15 AM  │
  ...     │
02:55 PM  │
03:00 PM ─┘ ← Disaster happens

Target VM is already up-to-date (last sync 2:55 PM):
  ├─ Data loss: 5 minutes maximum
  ├─ No restore needed (target already synced)
  ├─ One-click failover (just boot target VM)
  └─ VM running in 5-15 minutes

RTO: 5-15 minutes
RPO: 1-15 minutes (sync interval)
```

### **Target Customers**

- Large enterprises (1,000+ VMs)
- Mission-critical workloads (finance, healthcare, e-commerce)
- Companies with strict RTO/RPO requirements
- Platform migration projects (VMware → CloudStack, VMware → AWS)
- MSPs offering premium DR services
- High availability requirements

### **Use Cases**

**Scenario 1: VMware to CloudStack Migration**
- Production: VMware vSphere (expensive licensing)
- Goal: Migrate to CloudStack (open-source, cost savings)
- Solution:
  1. Set up replication VMware → CloudStack
  2. Continuous sync with near-zero downtime
  3. Planned failover during maintenance window
  4. 5-10 minutes downtime for cutover
  5. Save 60%+ on licensing costs

**Scenario 2: Zero-Downtime DR**
- Production: VMware on-prem (primary datacenter)
- DR: CloudStack in secondary datacenter
- Continuous replication (every 5 minutes)
- Primary datacenter failure:
  - Failover in 10 minutes
  - Data loss: 5 minutes maximum
  - Business continuity maintained

**Scenario 3: Cloud Burst**
- Production: On-prem VMware (normal load)
- Peak season: Need extra capacity
- Solution: Replicate select VMs to AWS EC2
- Scale up in cloud during peak
- Scale back down after peak
- Pay cloud costs only when needed

### **Competitive Positioning**

**vs PlateSpin Migrate:**
- ✅ Better pricing ($100 vs $150/VM)
- ✅ Modern UI and architecture
- ✅ CloudStack support (PlateSpin doesn't have this)
- ✅ More platform combinations

**vs Carbonite Migrate:**
- ✅ Competitive pricing ($100 vs $80-120/VM)
- ✅ Better performance (3.2 GiB/s proven)
- ✅ Open architecture

**vs Zerto:**
- ✅ True cross-platform (Zerto is VMware-centric)
- ✅ Lower cost
- ✅ Not locked to specific storage vendors

**UNIQUE ADVANTAGE:**
- **VMware → CloudStack is EXCLUSIVE to Sendense**
- No other vendor offers this combination
- Massive market (VMware shops looking to escape Broadcom)

---

## 💰 Pricing Comparison Matrix

| Feature | Backup | Enterprise | Replication |
|---------|--------|------------|-------------|
| **Price/VM/Month** | $10 | $25 | $100 |
| **Same-Platform Restore** | ✅ | ✅ | ✅ |
| **Cross-Platform Restore** | ❌ | ✅ | ✅ |
| **Cross-Platform Replication** | ❌ | ❌ | ✅ |
| **RTO** | Hours | Hours | Minutes |
| **RPO** | Backup interval | Backup interval | Minutes |
| **Immutable Storage** | ❌ | ✅ | ✅ |
| **Compliance Reporting** | ❌ | ✅ | ✅ |
| **24/7 Support** | ❌ | ❌ | ✅ |
| **Failback Capability** | ❌ | ❌ | ✅ |
| **Test Failover** | ❌ | ❌ | ✅ |

---

## 🎯 Market Segmentation

### **Backup Edition Targets**
- **Size:** 10-100 VMs
- **Budget:** $1,000-10,000/month
- **Need:** Standard backup/restore
- **Competition:** Veeam Community, Acronis

### **Enterprise Edition Targets**
- **Size:** 100-1,000 VMs
- **Budget:** $10,000-100,000/month
- **Need:** Platform flexibility, compliance
- **Competition:** Veeam Enterprise, Commvault

### **Replication Edition Targets**
- **Size:** 50-5,000 VMs (but only replicate critical subset)
- **Budget:** $50,000-500,000/month
- **Need:** Near-zero RTO/RPO, migrations
- **Competition:** PlateSpin, Carbonite, Zerto

---

## 🚀 Go-To-Market Strategy

### **Phase 1: Launch with Replication (Premium First)**

**Why start premium?**
- Higher margins ($100/VM vs $10/VM)
- VMware → CloudStack is unique (no competition)
- Targets enterprises with budget
- Builds reputation with complex deployments

**Initial Target:** VMware → CloudStack migrations
- Broadcom price increases driving migration
- Sendense is ONLY solution for this
- Charge premium for unique capability

### **Phase 2: Expand to Enterprise Tier**

Once replication proven:
- Add cross-platform restore (from backups)
- Expand storage backends (S3, Azure, immutable)
- Target DR-focused customers

### **Phase 3: Volume with Backup Tier**

After premium tiers established:
- Launch Backup Edition for volume
- Compete directly with Veeam Community
- Cross-sell to Replication customers

---

## 📊 Revenue Model Example

**Scenario: 100-VM Customer**

### **Option 1: Backup Only**
- 100 VMs × $10/month = $1,000/month
- Annual: $12,000

### **Option 2: Backup + Cross-Platform Restore**
- 100 VMs × $25/month = $2,500/month
- Annual: $30,000

### **Option 3: Hybrid (Backup + Selective Replication)**
- 80 VMs × $10/month (Backup) = $800/month
- 20 VMs × $100/month (Replication) = $2,000/month
- **Total: $2,800/month**
- **Annual: $33,600**

**Most common scenario:** Hybrid
- Backup all VMs for standard DR
- Replicate only mission-critical VMs for zero downtime

---

## 🎁 Free Tier (Growth Strategy)

**Sendense Community Edition** (Free)

**Purpose:** Hook developers, small businesses, create ecosystem

**Features:**
- Up to 5 VMs protected
- Local backup storage only
- Same-platform restore only
- Community support (forum)
- Limited to single OMA instance

**Upgrade Path:**
- Start free, grow to paid tiers
- Credit card required after 5 VMs
- Frictionless upgrade

---

## 📚 Related Documents

- **Pricing Strategy**: `02-pricing-strategy.md`
- **Competitive Analysis**: `03-competitive-analysis.md`
- **System Architecture**: `../architecture/01-system-overview.md`

---

**Document Owner:** Product Management  
**Review Cycle:** Quarterly  
**Last Reviewed:** October 4, 2025


