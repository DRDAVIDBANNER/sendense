# Sendense Appliance Deployment Manifest

**Document Version:** 1.0  
**Last Updated:** October 4, 2025  
**Status:** ✅ **CURRENT**

---

## 🏗️ Appliance Deployment Overview

### **Sendense Three-Appliance Architecture**

```
┌─────────────────────────────────────────────────────────────┐
│ SENDENSE APPLIANCE DEPLOYMENT ARCHITECTURE                  │ 
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ SCA - Sendense Control Appliance (Cloud MSP)       │   │
│  │ • Multi-tenant control plane                        │   │
│  │ • MSP dashboard and billing                         │   │
│  │ • Customer white-label portals                      │   │
│  │ • Bulletproof licensing server                      │   │
│  │ Deployment: AWS/Azure/GCP Kubernetes               │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↕ License Management API             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ SHA - Sendense Hub Appliance (Customer On-Prem)    │   │
│  │ • Central orchestration and storage                 │   │
│  │ • Backup repository management                      │   │
│  │ • Cross-platform restore engine                     │   │
│  │ • Customer cockpit GUI                              │   │
│  │ Deployment: Customer datacenter VM/server          │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↕ SSH Tunnel Port 443                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ SNA - Sendense Node Appliance (Source Platform)    │   │
│  │ • Platform-specific data capture                    │   │
│  │ • Change tracking (CBT, RCT, Dirty Bitmaps)        │   │
│  │ • NBD streaming and tunneling                       │   │
│  │ • Real-time progress reporting                      │   │
│  │ Deployment: Near source systems (VMware, etc.)     │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 📦 CURRENT DEPLOYMENT PACKAGES

### **SHA - Sendense Hub Appliance** (On-Prem Customer)

**Location:** `deployment/sha-appliance/`

**Current Version:** v2.11.1-vm-disks-null-fix (October 6, 2025)
**Binaries:**
- `sendense-hub-v2.11.1-vm-disks-null-fix` (34 MB) - Latest
- `volume-daemon-v1.2.1-linux-amd64-20251004-def456ab` (15.3 MB)
- `sendense-cockpit-v1.2.0-gui.tar.gz` (12.1 MB)

**Components:**
- Hub Appliance (central orchestration - renamed from OMA)
- Volume Management Daemon (storage operations)
- Sendense Cockpit GUI (aviation-inspired interface)
- Database schema and migrations (6 migrations)
- Configuration templates

**Database Migrations:**
- `20251003160000` - Add operation_summary column
- `20251004120000` - Add backup_jobs and backup_chains tables
- `20251004120001` - Fix backup_jobs table structure
- `20251005120000` - Add restore_mounts table
- `20251005130000` - Add disk_id to backup_jobs
- `20251006200000` - **NEW:** Make vm_disks.job_id nullable (VM discovery fix)

**Latest Update (October 6, 2025):**
- ✅ vm_disks table now populated during VM discovery
- ✅ Backup operations work without replication jobs
- ✅ Migration: 20251006200000_make_vm_disks_job_id_nullable
- ✅ Binary: sendense-hub-v2.11.1-vm-disks-null-fix deployed
- 📄 Details: `deployment/sha-appliance/VM-DISKS-DISCOVERY-DEPLOYMENT-UPDATE.md`

**Deployment Script:** `deployment/sha-appliance/scripts/deploy-sha.sh`

### **SNA - Sendense Node Appliance** (Source Platform)

**Location:** `deployment/sna-appliance/`

**Current Version:** v2.1.5 (VMware variant)
**Binaries:**
- `sendense-node-vmware-v2.1.5-linux-amd64-20251004-ghi789cd` (20.1 MB)
- `sendense-node-cloudstack-v1.0.3-linux-amd64-20251004-jkl012ef` (18.7 MB)

**Components:**
- Node Appliance (data capture - renamed from VMA)
- Platform-specific capture agents (VMware, CloudStack)
- SSH tunnel client configuration
- Platform dependencies (VDDK, libvirt, etc.)

**Deployment Scripts:**
- `deployment/sna-appliance/scripts/deploy-sna.sh` (general)
- `deployment/sna-appliance/scripts/setup-vmware-node.sh` (VMware-specific)
- `deployment/sna-appliance/scripts/setup-cloudstack-node.sh` (CloudStack-specific)

### **SCA - Sendense Control Appliance** (Cloud MSP) 

**Location:** `deployment/sca-appliance/`

**Current Version:** v1.0.0 (planned - Phase 7)
**Components:**
- Control Appliance (MSP multi-tenant control)
- License server and validation
- MSP dashboard and customer portals
- Billing automation and usage tracking
- White-label portal generator

**Deployment Scripts:**
- `deployment/sca-appliance/scripts/deploy-sca-aws.sh` (AWS deployment)
- `deployment/sca-appliance/scripts/deploy-sca-k8s.sh` (Kubernetes)

---

## 🔄 DEPLOYMENT SCRIPT MAINTENANCE

### **Script Update Requirements (MANDATORY)**

**Triggers for Deployment Script Updates:**
- ✅ **New binary release** → Update script with new version numbers
- ✅ **Configuration changes** → Update config templates
- ✅ **Dependency changes** → Update system requirements
- ✅ **Security updates** → Update SSH keys, certificates, permissions
- ✅ **Database migrations** → Update schema deployment steps

**Update Checklist (For Each Script Update):**
- [ ] Binary version numbers updated to match latest builds
- [ ] Configuration templates reflect current settings
- [ ] System dependencies list is current
- [ ] Security configurations are current
- [ ] Rollback procedures tested
- [ ] Documentation updated with any changes

### **Deployment Script Versioning**

**Scripts Must Include:**
```bash
#!/bin/bash
# deploy-sha.sh - Sendense Hub Appliance Deployment
# Version: v3.0.1
# Last Updated: 2025-10-04
# Compatible with: SHA v3.0.1, Volume Daemon v1.2.1
# Tested on: Ubuntu 22.04 LTS, RHEL 9

set -euo pipefail

SCRIPT_VERSION="v3.0.1"
REQUIRED_SHA_VERSION="v3.0.1"
REQUIRED_VOLUME_DAEMON_VERSION="v1.2.1"

echo "🚀 Sendense Hub Appliance Deployment Script ${SCRIPT_VERSION}"
echo "📦 Target SHA Version: ${REQUIRED_SHA_VERSION}"
echo "⚙️ Volume Daemon Version: ${REQUIRED_VOLUME_DAEMON_VERSION}"
```

---

## 📋 BINARY PLACEMENT REQUIREMENTS

### **Binary Deployment Locations (MANDATORY)**

**SHA Binaries:**
```
deployment/sha-appliance/binaries/
├── sendense-hub-v3.0.1-linux-amd64-20251004-abc123ef
├── volume-daemon-v1.2.1-linux-amd64-20251004-def456ab
├── CHECKSUMS.sha256                    # SHA256 sums for all binaries
└── BUILD_MANIFEST.md                   # Build details and dependencies
```

**SNA Binaries:**
```
deployment/sna-appliance/binaries/
├── sendense-node-vmware-v2.1.5-linux-amd64-20251004-ghi789cd
├── sendense-node-cloudstack-v1.0.3-linux-amd64-20251004-jkl012ef
├── sendense-node-hyperv-v1.0.1-windows-amd64-20251004-mno345gh
├── CHECKSUMS.sha256
└── BUILD_MANIFEST.md
```

**Build Pipeline Integration:**
```bash
# Automated binary deployment (part of CI/CD)
#!/bin/bash
# deploy-binaries-to-packages.sh

VERSION="${1:-$(cat source/current/VERSION.txt)}"

echo "📦 Deploying binaries to deployment packages for version ${VERSION}"

# 1. Copy SHA binaries
cp source/builds/control-plane/sendense-hub-v${VERSION}-* \
   deployment/sha-appliance/binaries/

cp source/builds/volume-daemon/volume-daemon-v*-* \
   deployment/sha-appliance/binaries/

# 2. Copy SNA binaries  
cp source/builds/capture-agents/vmware/sendense-node-vmware-v*-* \
   deployment/sna-appliance/binaries/

cp source/builds/capture-agents/cloudstack/sendense-node-cloudstack-v*-* \
   deployment/sna-appliance/binaries/

# 3. Generate checksums for each appliance
cd deployment/sha-appliance/binaries/
sha256sum sendense-hub-v* volume-daemon-v* > CHECKSUMS.sha256

cd ../../sna-appliance/binaries/
sha256sum sendense-node-*-v* > CHECKSUMS.sha256

# 4. Update deployment scripts with new versions
./update-deployment-scripts.sh "${VERSION}"

echo "✅ Binaries deployed to all appliance packages"
```

---

## 🎯 JOB SHEET LINKING SYSTEM

### **Job Sheet Creation (MANDATORY for All Work)**

**Format:** `job-sheets/YYYY-MM-DD-[task-description].md`

**Example Job Sheet:**
```markdown
# Job Sheet: Implement QCOW2 Backup Storage

**Date:** 2025-10-04  
**Project Goal Link:** [project-goals/phases/phase-1-vmware-backup.md → Task 1.2]  
**Business Value:** Enables Backup Edition tier ($10/VM revenue)  
**Assigned:** [Developer Name]  
**AI Session:** [Session ID if AI-assisted]

## Project Goals Integration
**Phase:** Phase 1 - VMware Backups (Week 1 of 6)
**Module:** Module 03 - Backup Repository
**Task Reference:** Task 1.2 - Local QCOW2 Backend Implementation
**Success Criteria:** [As defined in project goals]

## Technical Requirements
- [ ] **Reuse existing:** NBD streaming, SSH tunnels, database schema  
- [ ] **New implementation:** QCOW2 backup file creation and management
- [ ] **Integration:** Connect to existing SHA (Hub Appliance) infrastructure
- [ ] **Testing:** Maintain 3.2 GiB/s throughput benchmark

## Completion Tracking
**Project Goals Task Status:**
- [ ] Read project-goals/phases/phase-1-vmware-backup.md Task 1.2
- [ ] Mark task as [IN PROGRESS] in project goals document
- [ ] Complete implementation with all acceptance criteria
- [ ] Mark task as [x] COMPLETED in project goals document with date

## Evidence Required
- [ ] Working QCOW2 file creation from NBD stream
- [ ] Backup chain management (full + incrementals)
- [ ] Performance test: 3.2+ GiB/s maintained
- [ ] Integration test: Works with existing SHA components
- [ ] Documentation: API docs updated, schema docs current

**Completion Date:** [When marked complete in project goals]
**Project Goals Updated:** [Confirmation that task marked off]
```

### **Progress Tracking Integration**

**Required Actions:**
1. **Before starting work:** Create job sheet linking to specific project goals task
2. **During work:** Update job sheet with progress and discoveries
3. **Mark project goals:** Update task status in actual project goals document
4. **Complete work:** Mark both job sheet and project goals task as complete

**Project Goals Task Marking:**
```markdown
# In project-goals/phases/phase-1-vmware-backup.md

### Task 1.2: Local QCOW2 Backend Implementation  
- [x] **COMPLETED 2025-10-04** - QCOW2 storage backend operational
  - **Job Sheet:** job-sheets/2025-10-04-qcow2-storage.md
  - **Evidence:** Performance test: 3.2 GiB/s maintained
  - **Integration:** Successfully integrated with SHA components
```

---

## 📁 PROJECT STRUCTURE SUMMARY

### **Complete Organization**
```
sendense/
├── start_here/                     # 🔴 MANDATORY READING
│   ├── README.md                   # This file - project orientation
│   ├── PROJECT_RULES.md            # Absolute rules and standards
│   ├── MASTER_AI_PROMPT.md         # AI context loading procedure
│   ├── CHANGELOG.md                # Change tracking standards
│   ├── BINARY_MANAGEMENT.md        # Binary organization rules
│   └── PROJECT_GOVERNANCE_SUMMARY.md # Complete framework
│
├── project-goals/                  # 🎯 MASTER ROADMAP (24 docs)
│   ├── README.md                   # Project overview
│   ├── phases/                     # 7 implementation phases
│   ├── modules/                    # 11 technical modules
│   ├── architecture/               # System design documents
│   └── editions/                   # Product tiers and competition
│
├── source/current/                 # ✅ AUTHORITATIVE SOURCE CODE
│   ├── hub-appliance/              # SHA (renamed from oma/)
│   ├── node-appliance/             # SNA (renamed from vma/)
│   ├── control-appliance/          # SCA (new for MSP)
│   └── api-documentation/          # 🔴 MUST BE CURRENT
│
├── deployment/                     # 🚀 APPLIANCE DEPLOYMENTS
│   ├── sha-appliance/              # Hub Appliance (customer on-prem)
│   ├── sna-appliance/              # Node Appliance (source capture)
│   └── sca-appliance/              # Control Appliance (MSP cloud)
│
├── source/builds/                  # ✅ VERSIONED BINARIES
├── job-sheets/                     # 📋 WORK TRACKING
└── docs/                          # 📚 USER DOCUMENTATION
```

---

## 🎯 SUCCESS CHECKLIST

### **Project Governance Complete When:**
- [x] **start_here/ directory** created with all governance docs
- [x] **PROJECT_RULES.md** established (mandatory compliance)
- [x] **MASTER_AI_PROMPT.md** created (AI context loading)
- [x] **Appliance terminology** updated (SNA/SHA/SCA)
- [x] **Deployment structure** organized (3 appliances)
- [x] **Job sheet system** established (link to project goals)
- [x] **Auto-reload system** created (prevent AI context loss)
- [x] **Binary management** rules established (no scattered files)

### **Development Process Complete When:**
- [ ] **Team training** on new governance completed
- [ ] **Automated checks** implemented (pre-commit hooks, CI/CD gates)
- [ ] **API documentation** audited and confirmed current
- [ ] **Deployment scripts** updated with new terminology
- [ ] **First job sheet** created for active work
- [ ] **Context reload** system tested with AI assistant

---

## 🔗 INTEGRATION WITH EXISTING WORK

### **Existing MigrateKit OSSEA Integration**

**What We Keep (Proven Components):**
- ✅ **VMware integration** (CBT, VDDK, 3.2 GiB/s proven)
- ✅ **SSH tunnel infrastructure** (port 443, Ed25519 keys)
- ✅ **Volume Daemon** (centralized volume management)  
- ✅ **Database schema** (VM-centric, CASCADE DELETE)
- ✅ **JobLog system** (structured operation tracking)

**What We Rename:**
- VMA → **SNA (Sendense Node Appliance)**
- OMA → **SHA (Sendense Hub Appliance)**
- Add: **SCA (Sendense Control Appliance)** for MSP

**What We Extend:**
- File-based backup storage (Phase 1)
- Cross-platform restore (Phase 4)
- Multi-platform replication (Phase 5)
- MSP multi-tenant control (Phase 7)

---

## 🚀 IMMEDIATE ACTION PLAN

### **This Week (Setup Phase)**
1. **✅ Team reads start_here/README.md** (this document)
2. **✅ Team acknowledges PROJECT_RULES.md** compliance
3. **🔄 Update all code** with SNA/SHA/SCA terminology
4. **🔄 Update deployment scripts** with new binary names
5. **📋 Create first job sheet** for any active development

### **Next Week (Implementation)**
1. **🎯 Start Phase 1** with proper job sheet linking
2. **📊 Implement automated checks** (pre-commit, CI/CD)
3. **🔍 API documentation audit** (ensure currency)
4. **🧪 Test context reload system** with AI assistant

---

**THIS GOVERNANCE FRAMEWORK ENSURES SENDENSE ACHIEVES ENTERPRISE-GRADE ENGINEERING EXCELLENCE REQUIRED TO DESTROY VEEAM AND BUILD A BILLION-DOLLAR PLATFORM**

---

**Document Owner:** Engineering Leadership  
**Maintenance:** Updated with every deployment  
**Compliance:** Mandatory for all development work  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **ACTIVE - IMMEDIATE IMPLEMENTATION**

