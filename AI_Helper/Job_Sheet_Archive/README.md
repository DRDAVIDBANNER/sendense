# AI Helper - MigrateKit OSSEA Project Documentation

**Last Updated**: September 27, 2025  
**Status**: 🚀 **PRODUCTION READY SYSTEM + COMPLETE ENTERPRISE FAILURE RECOVERY + INTELLIGENT CLEANUP SYSTEM**  
**Purpose**: Single source of truth for AI assistant rules, project context, and current status

---

## 🎯 **PURPOSE**

This directory contains the essential rules, constraints, and context for all AI assistant interactions with the MigrateKit OSSEA project. It ensures consistency across sessions and prevents architectural violations.

---

## 📁 **DIRECTORY STRUCTURE**

```
AI_Helper/
├── README.md                           # This file - Overview and navigation
├── MASTER_PROMPT.md                   # 🚀 CONCISE master prompt for any AI assistant
├── RULES_AND_CONSTRAINTS.md           # Hard project rules (mandatory compliance)
├── VERIFIED_DATABASE_SCHEMA.md          # ✅ VERIFIED Complete schema reference (prevents field name assumptions)
├── CURRENT_PROJECT_STATUS.md          # Active project status and current issues
├── OMA_CONSOLIDATION_COMPLETION_REPORT.md  # ✅ September 2025 OMA consolidation achievement
├── VOLUME_DAEMON_ASSESSMENT.md        # 📋 Volume Daemon consolidation assessment
├── VOLUME_DAEMON_CONSOLIDATION_COMPLETION_REPORT.md  # ✅ September 2025 Volume Daemon consolidation achievement
├── LINSTOR_CLEANUP_AND_DUPLICATE_CODE_COMPLETION_REPORT.md  # ✅ September 2025 Linstor removal & duplicate code cleanup
├── CHATGPT_PROMPT.md                  # Drop-in context for ChatGPT-5 online
```

---

## 🚨 **CRITICAL AI ASSISTANT RULES**

### **1. MANDATORY READING ORDER**
When starting any session, AI assistants MUST read these files in order:
1. `MASTER_PROMPT.md` - 🚀 **CONCISE master prompt (START HERE)**
2. `RULES_AND_CONSTRAINTS.md` - Non-negotiable project rules (detailed reference)
3. `VERIFIED_DATABASE_SCHEMA.md` - ✅ VERIFIED Schema field names (no assumptions allowed)
4. `CURRENT_PROJECT_STATUS.md` - Current state and active issues
5. `OMA_CONSOLIDATION_COMPLETION_REPORT.md` - ✅ **September 2025 OMA consolidation achievement**
6. `VOLUME_DAEMON_CONSOLIDATION_COMPLETION_REPORT.md` - ✅ **September 2025 Volume Daemon consolidation achievement**
7. `ENHANCED_FAILOVER_REFACTORING_COMPLETION_REPORT.md` - ✅ **September 2025 Enhanced failover modular refactoring achievement**
8. `CLEANUP_SERVICE_REFACTORING_COMPLETION_REPORT.md` - ✅ **September 2025 Cleanup service modular refactoring achievement**
9. `LINSTOR_CLEANUP_AND_DUPLICATE_CODE_COMPLETION_REPORT.md` - ✅ **September 2025 Linstor removal & duplicate code cleanup**
10. `VOLUME_DAEMON_ASSESSMENT.md` - 📋 **Volume Daemon consolidation assessment (historical)**
11. `CHATGPT_PROMPT.md` - Full context for external AI systems

### **2. SOURCE CODE HIERARCHY**
- **ONLY** use code under `source/current/` as authoritative
- **NEVER** reference or modify archived code without explicit permission
- **ALWAYS** validate database field names against schema before queries/updates

### **3. ARCHITECTURAL GUARDRAILS**
- **Volume Operations**: MUST use Volume Daemon via `internal/common/volume_client.go`
- **Logging**: MUST use `internal/joblog` for all business logic
- **Networking**: ONLY port 443 tunnel; no direct connections
- **Versioning**: Respect `source/current/VERSION.txt`; no "latest" tags

### **4. MEMORY COMPLIANCE**
AI assistants must update project memories when rules change or discoveries are made. Never contradict established rules without explicit user override.

### **5. DEPLOYMENT INFORMATION** 🔥 **CRITICAL**
- **SSH Key**: `~/.ssh/cloudstack_key` for VMA access (`ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231`)
- **VMA Location**: `pgrayson@10.0.100.231` (VMware appliance)
- **OMA Location**: Local system (`localhost`) 

#### **Source Code Locations** 🔥 **UPDATED SEPTEMBER 25, 2025 - FINAL CONSOLIDATION**
- **MigrateKit Source**: `/home/pgrayson/migratekit-cloudstack/source/current/migratekit/` ✅ **AUTHORITATIVE** (v2.18.0-job-type-propagation)
- **VMA API Server Source**: `/home/pgrayson/migratekit-cloudstack/source/current/vma-api-server/` ✅ **AUTHORITATIVE** (v1.10.4-progress-fixed)
- **VMA Services Source**: `/home/pgrayson/migratekit-cloudstack/source/current/vma/` ✅ **AUTHORITATIVE** (CBT + Progress)
- **OMA Source**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/` ✅ **AUTHORITATIVE** (v2.22.0-polling-debug-enhanced)
- **Volume Daemon Source**: `/home/pgrayson/migratekit-cloudstack/source/current/volume-daemon/` ✅ **AUTHORITATIVE** (v1.2.3-multi-volume-snapshots)

#### **Binary Locations & Services** 🔥 **UPDATED SEPTEMBER 26, 2025**
- **MigrateKit** (ON VMA): `/home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel` → `migratekit-v2.20.1-chunk-size-fix` ✅ **SPARSE BLOCK OPTIMIZATION + NBD COMPATIBILITY**
- **VMA API** (ON VMA): `/home/pgrayson/migratekit-cloudstack/vma-api-server` → `vma-api-server-v1.10.4-progress-fixed` (systemd: `vma-api.service`)
- **OMA API** (LOCAL): `/opt/migratekit/bin/oma-api` → `oma-api-v2.28.0-intelligent-cleanup` (systemd: `oma-api.service`) ✅ **INTELLIGENT FAILED EXECUTION CLEANUP + ENTERPRISE PROTECTION**
- **Volume Daemon** (LOCAL): `/usr/local/bin/volume-daemon` → `volume-daemon-v1.3.2-persistent-naming-fixed` (systemd: `volume-daemon.service`) ✅ **PERSISTENT DEVICE NAMING + NBD MEMORY SYNC**

#### **VMA API Endpoints** 🔥 **ESSENTIAL**
- **Base URL**: `http://localhost:8081` (when connected to VMA) or via tunnel from OMA
- **Progress API**: 
  - `GET /api/v1/progress/{jobId}` - Get job progress
  - `POST /api/v1/progress/{jobId}/update` - Update progress (from migratekit)
- **Health Check**: `GET /api/v1/health` - Service health status
- **VM Operations**: 
  - `GET /api/v1/vms/{vmPath}/cbt-status` - Check CBT status
  - `POST /api/v1/vms/{vmPath}/enable-cbt` - Enable CBT

#### **Database Connection** 🔥 **ESSENTIAL**
- **Connection String**: `oma_user:oma_password@tcp(localhost:3306)/migratekit_oma`
- **Quick Access**: `mysql -u oma_user -poma_password migratekit_oma`
- **Replication Jobs Query**: `SELECT id, source_vm_name, status, created_at FROM replication_jobs WHERE source_vm_name = 'VMNAME' ORDER BY created_at DESC LIMIT 5;`

#### **Job Deletion Implementation** 🔥 **NEW FEATURE**
- **Endpoint**: `DELETE /api/v1/replications/{job_id}` - Complete job deletion with volume cleanup
- **Safety Protection**: Prevents deletion of attached volumes (CloudStack API error 431 expected behavior)
- **Database Schema**: Uses `VERIFIED_DATABASE_SCHEMA.md` for accurate field names
- **Volume Daemon Integration**: All volume operations via Volume Daemon API for consistency
- **JobLog Tracking**: Full audit trail for deletion operations
- **Repository Pattern**: Centralized database operations via `ReplicationJobRepository`

#### **Essential Commands**
```bash
# Database Access
mysql -u oma_user -poma_password migratekit_oma

# Job Deletion Testing
curl -X DELETE "http://localhost:8082/api/v1/replications/job-YYYYMMDD-HHMMSS" -v

# Build MigrateKit
cd source/current/migratekit && go build -o migratekit-v2.X.X-feature .

# Build VMA API  
cd source/current && go build -o vma-api-server-v1.X.X-feature ./vma-api-server/main.go

# Build OMA API
cd cmd/oma && go build -o oma-api-v2.X.X-feature .

# Deploy to VMA (example)
scp -i ~/.ssh/cloudstack_key migratekit-v2.X.X-feature pgrayson@10.0.100.231:/home/pgrayson/migratekit-cloudstack/

# Update symlinks & restart services (example)
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "ln -sf migratekit-v2.X.X-feature migratekit-tls-tunnel && sudo systemctl restart vma-api"

# Test VMA API
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "curl -s http://localhost:8081/api/v1/health"
```

---

## 📋 **QUICK REFERENCE CHECKLIST**

Before making ANY changes to the project:

- [ ] Read `RULES_AND_CONSTRAINTS.md` completely
- [ ] Validate field names in `VERIFIED_DATABASE_SCHEMA.md`
- [ ] Check current issues in `CURRENT_PROJECT_STATUS.md`
- [ ] Confirm git commit exists (user preference)
- [ ] Verify source location under `source/current/`
- [ ] Identify which services are affected
- [ ] Plan minimal, non-breaking changes

---

## 🔗 **EXTERNAL AI INTEGRATION**

### **🚀 RECOMMENDED: Use Master Prompt**
For ChatGPT-5 or other online AI systems:
1. Copy the entire contents of `MASTER_PROMPT.md`
2. Paste as the first message to establish context
3. Concise but complete project understanding

### **📖 ALTERNATIVE: Use Full Context**
For comprehensive context:
1. Copy the entire contents of `CHATGPT_PROMPT.md`
2. Paste as the first message to establish context
3. The external AI will have detailed project understanding

---

## 📚 **ARCHIVED DOCUMENTATION**

All previous project documentation has been archived under `archive/2025-09-03-project-cleanup/`:
- Project status documents
- Implementation guides  
- Bug tracking
- Session notes
- Technical specifications

**Rule**: Archived content is READ-ONLY reference material. Do not copy old patterns or rules from archived content without validating against current rules.

---

## 🔄 **MAINTENANCE**

This directory MUST be updated:
- After every significant session
- When project rules change
- When database schema evolves
- When architectural decisions are made

**Responsible Party**: Every AI assistant session must maintain these files current and accurate.

---

## 🚨 **EMERGENCY PROCEDURES**

If any file in this directory becomes corrupted or inconsistent:
1. STOP all development work immediately
2. Restore from the most recent archive
3. Validate all rules and constraints
4. Update project memories to reflect current state
5. Resume only after full consistency check

---

**🎯 Bottom Line**: This directory prevents the chaos of version mix-ups, rule violations, and architectural drift. Every AI assistant MUST respect these guardrails to maintain project integrity.