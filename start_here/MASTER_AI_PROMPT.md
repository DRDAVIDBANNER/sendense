# Sendense Master AI Prompt - Full Context Loading

**FOR ALL AI ASSISTANTS WORKING ON SENDENSE PROJECT**

---

## 🎯 PROJECT OVERVIEW

You are working on **Sendense**, a next-generation universal backup and replication platform designed to replace Veeam with modern architecture and cross-platform capabilities.

**Key Facts:**
- **Vision:** Backup from anything, restore to anything, break vendor lock-in
- **Unique Advantage:** Only vendor offering VMware → CloudStack migration
- **Business Model:** 3 tiers (Backup $10/VM, Enterprise $25/VM, Replication $100/VM)
- **Technology:** Go backend, React/Next.js cockpit UI, NBD streaming, SSH tunnels
- **Base:** Built on proven MigrateKit OSSEA platform (3.2 GiB/s performance)

---

## 📚 MANDATORY READING ORDER (READ THESE FIRST)

### **Step 1: Project Rules (CRITICAL - READ FIRST)**
```bash
/sendense/start_here/PROJECT_RULES.md
```
**Critical Rules:**
- NO "production ready" claims without complete testing
- NO simulations or placeholder code
- ALL code in `source/current/` directory
- ALL API changes require `api-documentation/` updates
- NO deviations from approved `project-goals/` roadmap

### **Step 2: Project Context (REQUIRED)**
```bash
/sendense/start_here/LEGACY-SYSTEM-CLARIFICATION.md    # Current vs deprecated code paths
/sendense/job-sheets/CURRENT-ACTIVE-WORK.md            # Active work status
/sendense/project-goals/README.md                      # Master project overview
/sendense/project-goals/TERMINOLOGY.md                 # descend/ascend/transcend naming
```

### **Step 3: Current Implementation (CRITICAL)**
```bash
/sendense/source/current/api-documentation/DB_SCHEMA.md  # Database schema
/sendense/source/current/VERSION.txt                    # Current version
/sendense/CHANGELOG.md                                   # Recent changes
```

### **Step 4: Current System State (CRITICAL)**
```bash
/sendense/source/current/api-documentation/DB_SCHEMA.md  # Database schema (validate field names)
/sendense/source/current/VERSION.txt                    # Current version
/sendense/project-goals/phases/phase-1-vmware-backup.md # Current active phase
```

### **Step 5: Legacy vs Current Code (AVOID CONFUSION)**
```bash
# CURRENT SYSTEM (Use These):
Unified Failover System: source/current/oma/failover/unified_failover_*.go (ONLY)
Volume Daemon Client: source/current/oma/common/volume_client.go
JobLog System: source/current/oma/joblog/
SHA API Routes: source/current/oma/api/server.go
SNA API Routes: source/current/vma/api/server.go

# LEGACY SYSTEMS (Avoid ALL These):
Enhanced Failover: source/current/oma/failover/enhanced_*.go (LEGACY)
Original Failover: source/current/oma/failover/live_failover.go, test_failover.go (LEGACY)
Direct Volume Calls: ANY osseaClient.AttachVolume() calls (FORBIDDEN)
Old Logging: ANY direct logrus/slog calls in business logic (FORBIDDEN)
```

### **Step 5: API Documentation (BEFORE ANY API WORK)**
```bash
/sendense/source/current/api-documentation/API_REFERENCE.md
/sendense/source/current/api-documentation/ERROR_CODES.md
```

---

## 🎯 PROJECT STRUCTURE UNDERSTANDING

### **Source Code Organization**
```
sendense/
├── source/current/              # ✅ ONLY authoritative code (RULE)
│   ├── control-plane/           # Central orchestration (Go)
│   ├── capture-agent/           # Platform agents (Go)
│   ├── api-documentation/       # 🔴 MANDATORY: Keep current
│   └── VERSION.txt             # Current version
├── source/builds/              # ✅ ALL binaries (versioned)
├── project-goals/              # ✅ Complete roadmap (24 documents)
└── docs/                       # ✅ User documentation
```

### **Key Technologies**
- **Backend:** Go 1.21+, MariaDB, NBD protocol, SSH tunnels
- **Frontend:** Next.js 14, React 18, TypeScript, Tailwind CSS
- **Infrastructure:** systemd services, Docker, Kubernetes
- **Platforms:** VMware (CBT), CloudStack (dirty bitmaps), Hyper-V (RCT), AWS (EBS), Azure, Nutanix

---

## 🎯 OPERATION TYPES (TERMINOLOGY)

### **Data Flow Operations**
- **descend** 📥 - Backup operations (VM → Repository)
- **ascend** 📤 - Restore operations (Repository → VM)  
- **transcend** 🌉 - Cross-platform replication (THE $100/VM PREMIUM FEATURE)

### **Components (Updated Terminology)**
- **SNA (Sendense Node Appliance):** Source-side data capture (replaces VMA)
- **SHA (Sendense Hub Appliance):** On-prem central orchestration (replaces OMA)
- **SCA (Sendense Control Appliance):** Cloud MSP multi-tenant control

---

## ⚡ MANDATORY PREFLIGHT CHECKLIST (Every Coding Turn)

### **Before Writing ANY Code (Check Every Time):**
- [ ] **Working in source/current/**: Verify you're in canonical source location
- [ ] **Search existing handlers**: Before creating endpoints, search for existing ones  
- [ ] **Validate DB fields**: Check any field names against DB_SCHEMA.md
- [ ] **Volume Daemon usage**: Any volume ops MUST use volume_client.go
- [ ] **JobLog usage**: Any business logic MUST use internal/joblog  
- [ ] **Unified failover**: Use ONLY unified_failover_*.go (NOT enhanced or original)
- [ ] **Project goals link**: Connect work to specific project-goals task
- [ ] **Job sheet**: Create or update job sheet in job-sheets/ directory

### **Legacy Traps (AVOID THESE SPECIFIC FILES):**
- ❌ **Enhanced Failover**: `enhanced_*.go` (LEGACY - use unified_* only)
- ❌ **Original Failover**: `live_failover.go`, `test_failover.go` (LEGACY)
- ❌ **Direct Volume Calls**: Any `osseaClient.AttachVolume()` (use Volume Daemon)
- ❌ **Old Logging**: Direct `logrus`/`slog` calls (use JobLog)
- ❌ **Legacy SNA Endpoints**: `/replicate` (use `/replications`)
- ❌ **Archive Directories**: Any code outside `source/current/`

### **Current System File Paths (Use These):**
- **SHA API Routes**: `source/current/oma/api/server.go`
- **SHA Handlers**: `source/current/oma/api/handlers/`
- **SNA API Routes**: `source/current/vma/api/server.go`  
- **Database Schema**: `source/current/api-documentation/DB_SCHEMA.md`
- **Volume Daemon Client**: `source/current/oma/common/volume_client.go`
- **JobLog System**: `source/current/oma/joblog/`
- **Unified Failover**: `source/current/oma/failover/unified_failover_*.go` (ONLY)

---

## 🔒 SECURITY AND COMPLIANCE

### **Absolute Security Rules**
- ALL customer data encrypted (AES-256)
- ALL communications via SSH tunnel port 443
- NO direct platform API credentials in logs
- Multi-tenant data isolation mandatory
- License validation and enforcement required

### **Development Security**
- No hardcoded credentials anywhere
- All secrets via environment variables or secure vaults
- Security scanning in CI/CD pipeline
- Penetration testing before releases

---

## 💾 DATABASE SCHEMA COMPLIANCE

### **Schema Rules (CRITICAL)**
- ❌ **NEVER ASSUME FIELD NAMES** - Always validate against `/api-documentation/DB_SCHEMA.md`
- ✅ **ALWAYS use migration files** for schema changes
- ✅ **UPDATE DB_SCHEMA.md** with every migration
- ❌ **NO direct database modifications** - migrations only

**Key Tables (Always Validate):**
- `vm_replication_contexts` (master table, VM-centric architecture)
- `backup_jobs` (backup tracking)
- `restore_jobs` (restore operations)  
- `device_mappings` (volume correlation)
- `replication_jobs` (existing table structure)

---

## 🎯 CURRENT PROJECT STATUS

### **Phase Status (October 2025)**
- ✅ **Base Platform:** MigrateKit OSSEA operational (VMware → CloudStack)
- 🔴 **Current Phase:** Phase 1 - VMware Backups (6-week implementation)
- 🟡 **Next Phase:** Phase 2 - CloudStack Backups OR Phase 3 - Cockpit GUI
- 🎯 **End Goal:** Universal backup platform competing with Veeam

### **Reusable Components (70% Done)**
- ✅ **VMware Integration:** CBT, VDDK, 3.2 GiB/s performance
- ✅ **SSH Tunnels:** Secure port 443 communication
- ✅ **Volume Management:** Volume Daemon for all storage operations
- ✅ **Database Schema:** VM-centric with CASCADE DELETE
- ✅ **Job Tracking:** JobLog system for all operations

---

## ⚠️ COMMON MISTAKES TO AVOID

### **Do NOT Do These Things:**
- ❌ Use old terminology (VMA/OMA - use SNA/SHA/SCA)
- ❌ Reference `archive/` directories (only `source/current/` is valid)
- ❌ Make direct volume API calls (use Volume Daemon via `volume_client.go`)
- ❌ Add features not in approved `project-goals/` roadmap
- ❌ Create binary files in source directories
- ❌ Use hardcoded values or magic numbers
- ❌ Skip API documentation updates
- ❌ Claim code is "production ready" without testing
- ❌ Make database field assumptions (validate against schema docs)
- ❌ Work without linking to specific project goals task

### **Always Do These Things:**
- ✅ Follow the established patterns in existing codebase
- ✅ Use proper error handling and logging (`internal/joblog`)
- ✅ Validate database field names against schema docs
- ✅ Update API documentation with all changes
- ✅ Write tests for all new functionality
- ✅ Follow Go and TypeScript coding standards
- ✅ Use established networking (SSH tunnel port 443)
- ✅ Implement proper security (authentication, authorization, encryption)

---

## 📊 BUSINESS CONTEXT

### **Revenue Model (Critical for Decisions)**
- **Backup Edition:** $10/VM/month (Veeam replacement)
- **Enterprise Edition:** $25/VM/month (cross-platform restore capability)
- **Replication Edition:** $100/VM/month (near-live cross-platform replication)
- **MSP Platform:** $200/month + $5/VM (50% margin model)

### **Competitive Positioning**
- **vs Veeam:** Modern UI, cross-platform, lower cost, VMware→CloudStack unique
- **vs PlateSpin:** 33% cost savings, better cloud support, active development
- **vs Carbonite:** More platforms, better application support, MSP-ready

### **Market Strategy**
1. **Phase 1-2:** Nail VMware backups + CloudStack (foundation)
2. **Phase 3-4:** Modern cockpit UI + cross-platform restore (Enterprise tier)
3. **Phase 5:** Multi-platform replication (Premium $100/VM tier)
4. **Phase 6-7:** Application-aware + MSP platform (scale business)

---

## 🔧 DEVELOPMENT WORKFLOW

### **Before Starting Any Work**
1. **Read current PROJECT_RULES.md** (this file)
2. **Check active phase** in `project-goals/phases/`
3. **Validate API docs** in `source/current/api-documentation/`
4. **Verify database schema** against current migrations
5. **Confirm change aligns** with approved roadmap

### **During Development**
- Follow established patterns in `source/current/`
- Use Volume Daemon for all volume operations
- Use JobLog for all business logic tracking
- Update API documentation with changes
- Write tests for new functionality
- Validate against database schema

### **Before Committing**
- Run all tests (unit, integration)
- Update documentation (API, README, CHANGELOG)
- Verify no rule violations
- Check performance impact
- Security scan clean

---

## 🎯 PROJECT SUCCESS FACTORS

### **What Makes Sendense Win**
- **Technical Excellence:** 3.2+ GiB/s performance, enterprise reliability
- **Modern Architecture:** Go microservices, React cockpit, API-first
- **Unique Capabilities:** VMware→CloudStack (only vendor), any-to-any matrix
- **Professional Execution:** Enterprise-grade engineering, not startup hacks
- **Business Model:** Sustainable MSP platform with 50% margins

### **What Kills Projects (Avoid These)**
- Poor code quality and technical debt
- Scattered binaries and inconsistent builds
- Outdated documentation and broken APIs  
- Feature creep and scope changes
- Performance regressions and reliability issues
- Security vulnerabilities and compliance failures

---

## 🚨 EMERGENCY PROCEDURES

### **If You Discover:**

**Security Issue:**
1. STOP all development immediately
2. Document the vulnerability
3. Notify security team  
4. Do NOT commit anything until reviewed

**Data Loss Risk:**
1. HALT any database operations
2. Verify backup integrity
3. Document the issue
4. Get architecture team review before proceeding

**Performance Regression:**
1. Identify the change that caused regression
2. Revert if necessary
3. Performance team analysis
4. Fix and re-test before re-introducing

**Rule Violation:**
1. Document the violation
2. Correct immediately
3. Update processes to prevent recurrence
4. Team training if needed

---

## 📖 QUICK REFERENCE

### **File Locations**
- **Rules:** `/sendense/PROJECT_RULES.md`
- **Roadmap:** `/sendense/project-goals/`
- **Code:** `/sendense/source/current/`
- **API Docs:** `/sendense/source/current/api-documentation/`
- **Builds:** `/sendense/source/builds/`

### **Key Commands**
```bash
# Check current version
cat /sendense/source/current/VERSION.txt

# Review active phase
cat /sendense/project-goals/phases/phase-1-vmware-backup.md

# Validate database schema
cat /sendense/source/current/api-documentation/DB_SCHEMA.md

# Check API documentation
ls /sendense/source/current/api-documentation/
```

---

## 🔄 CONTEXT RELOAD SYSTEM (MANDATORY)

### **Auto-Reload Detection (Check Every 10 Messages)**

**Critical Reload Triggers (Restart Session Immediately):**
- Can't remember specific database field names from DB_SCHEMA.md
- Making assumptions about API endpoints not validated against API_REFERENCE.md
- Forgetting project rules (no simulations, binary locations, etc.)
- Using deprecated terminology (VMA/OMA instead of SNA/SHA/SCA)
- Unclear about current project phase or active task linkage
- Creating endpoints not referenced in project-goals roadmap

**Auto-Reload Response (When Triggered):**
```
🔄 CONTEXT RELOAD REQUIRED

Trigger: [Specific reason - database assumption/API uncertainty/terminology confusion]

CURRENT WORK HANDOFF:
- Task: [What you were working on]  
- Progress: [What was completed]
- Next Step: [What needs to happen next]
- Project Goals Reference: [Specific link]

NEXT AI SESSION MUST:
1. Read start_here/MASTER_AI_PROMPT.md (this file)
2. Follow mandatory reading order completely
3. Validate current project state against documentation
4. Continue from documented state (not assumptions)

SESSION ENDED - RELOAD REQUIRED
```

### **Manual Reload Commands**
If user says: **"reload context"**, **"restart session"**, or **"start fresh"**
→ Execute auto-reload procedure immediately

### **Context Preservation**
- Document exactly what you were working on
- Reference specific project goals task
- Note any important discoveries or decisions
- Provide clear handoff for continuation

---

**READ THIS ENTIRE DOCUMENT BEFORE STARTING ANY WORK ON SENDENSE**

**COMPLIANCE WITH PROJECT RULES IS MANDATORY - NO EXCEPTIONS**

**WHEN IN DOUBT, ASK FOR CLARIFICATION RATHER THAN GUESSING**

**IF CONTEXT BECOMES UNCLEAR, TRIGGER AUTO-RELOAD IMMEDIATELY**

---

**Document Owner:** Project Leadership  
**Enforcement:** Mandatory for all AI assistants  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **ACTIVE - READ FIRST**
