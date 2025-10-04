# Job Sheet: Example Task Linking to Project Goals

**Date Created:** 2025-10-04  
**Status:** ✅ **EXAMPLE - TEMPLATE FOR ALL WORK**  
**Project Goal Link:** [project-goals/phases/phase-1-vmware-backup.md → Task 1.2]

---

## 🎯 PROJECT GOALS INTEGRATION (MANDATORY)

### **Specific Project Goals Reference**
**Document:** `/project-goals/phases/phase-1-vmware-backup.md`  
**Task Section:** **Task 1: Backup Repository Abstraction** (Week 1)  
**Sub-Task:** **1.2. Implement Local QCOW2 Backend**  
**Business Value:** Enables Backup Edition tier ($10/VM revenue stream)  
**Success Criteria:** As defined in Phase 1 acceptance criteria

**Task Description from Project Goals:**
```
1.2. Implement Local QCOW2 Backend
   - QCOW2 file creation with backing files
   - Incremental chain management
   - Metadata storage (JSON sidecar files)
```

**Acceptance Criteria from Project Goals:**
```
- [ ] Can create QCOW2 file with backing file
- [ ] Can track backup chains in metadata
- [ ] Can list all backups for a VM
- [ ] Can calculate total chain size
```

---

## 📋 JOB BREAKDOWN (Detailed Implementation)

### **Technical Tasks (Must All Complete)**
- [x] **Research QCOW2 format** - Understand backing file mechanics
  - **Evidence:** QCOW2 format documentation reviewed
  - **Result:** qemu-img commands identified for implementation

- [x] **Design backup chain structure** - File organization and metadata
  - **Evidence:** File structure designed in `/var/lib/sendense/backups/`
  - **Result:** JSON metadata schema for backup chains

- [ ] **Implement QCOW2Manager struct** - Core QCOW2 operations
  - **File:** `source/current/hub-appliance/storage/qcow2_manager.go`
  - **Methods:** CreateQCOW2(), CreateWithBacking(), AttachViaQemuNBD()

- [ ] **Implement BackupChainManager** - Chain tracking and metadata
  - **File:** `source/current/hub-appliance/storage/chain_manager.go`  
  - **Methods:** CreateChain(), AddToChain(), GetChainMetadata()

- [ ] **Integration with NBD server** - File export capability
  - **File:** `source/current/hub-appliance/nbd/file_export.go`
  - **Method:** CreateFileExport() - export QCOW2 via NBD

- [ ] **Unit tests** - Test all QCOW2 and chain operations
  - **Coverage:** 80%+ for all new code
  - **File:** `source/current/hub-appliance/storage/qcow2_test.go`

- [ ] **Integration tests** - End-to-end backup creation
  - **Test:** Create full backup → create incremental → verify chain
  - **File:** `tests/integration/backup_storage_test.go`

### **Documentation Tasks (Must All Complete)**
- [ ] **API documentation** - Update with new backup storage endpoints
  - **File:** `source/current/api-documentation/API_REFERENCE.md`
  - **Add:** POST /api/v1/repositories/local/write endpoint

- [ ] **Schema documentation** - Update if new tables needed
  - **File:** `source/current/api-documentation/DB_SCHEMA.md`
  - **Check:** Any new tables for backup chain tracking

- [ ] **CHANGELOG.md** - Document backup storage feature addition
  - **Category:** Added
  - **Entry:** "Local QCOW2 backup repository with incremental chains"

### **Deployment Tasks (If Applicable)**
- [ ] **Binary updates** - If hub appliance binary changes
  - **Location:** `deployment/sha-appliance/binaries/`
  - **Update:** Build manifest with new version

- [ ] **Script updates** - If deployment procedure changes  
  - **File:** `deployment/sha-appliance/scripts/deploy-sha.sh`
  - **Update:** Configuration or dependency changes

---

## ✅ SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **Functional Testing:** QCOW2 backup creation working end-to-end
- [ ] **Performance Testing:** 3.2+ GiB/s throughput maintained
- [ ] **Integration Testing:** Works with existing SHA components
- [ ] **Code Quality:** Passes linting, security scan, code review
- [ ] **Documentation:** All required docs updated and accurate
- [ ] **Project Goals:** Task marked complete with evidence

### **Evidence Collection (Required)**
- **Functional Test:** [Link to working demo or test results]
- **Performance Test:** [Benchmark results showing throughput]
- **Integration Test:** [Test showing end-to-end backup workflow]
- **Code Review:** [Link to approved pull request]
- **Documentation:** [Links to updated API and schema docs]

---

## 📊 PROJECT GOALS COMPLETION MARKING

### **How to Mark Task Complete in Project Goals**

**Step 1: Update Project Goals Document**
```bash
# Edit the actual project goals file
vi project-goals/phases/phase-1-vmware-backup.md

# Find Task 1.2 and update:
### Task 1: Backup Repository Abstraction (Week 1)

1.2. **Implement Local QCOW2 Backend**
   - [x] **COMPLETED 2025-10-04** ✅
   - **Job Sheet:** job-sheets/2025-10-04-qcow2-storage.md
   - **Evidence:** QCOW2 backup storage operational, 3.2 GiB/s maintained
   - **Integration:** Successfully integrated with SHA NBD server
   - **Files:** hub-appliance/storage/qcow2_manager.go, chain_manager.go
   - **Tests:** All unit and integration tests passing
```

**Step 2: Update Overall Phase Progress**
```markdown
# Track overall phase completion
**Phase 1 Progress:** Task 1.2 complete (2/7 tasks, 29% complete)
```

**Step 3: Enable Dependent Tasks**
```markdown
# Mark dependencies satisfied
**Enables:** Task 2 (NBD File Export) can now proceed
**Blocks Removed:** QCOW2 storage dependency resolved
```

---

## 🔄 CONTEXT RELOAD EXAMPLE

### **When AI Context Becomes Unclear**

**AI Response When Triggered:**
```markdown
🔄 CONTEXT RELOAD REQUIRED

**Trigger:** Making assumption about backup_jobs table fields without validating against DB_SCHEMA.md

**Current Work Handoff:**
- **Task:** Implementing QCOW2 backup storage
- **Progress:** QCOW2Manager struct designed, need to implement CreateWithBacking()
- **Next Step:** Complete chain_manager.go implementation
- **Project Goal:** project-goals/phases/phase-1-vmware-backup.md → Task 1.2
- **Job Sheet:** job-sheets/2025-10-04-qcow2-storage.md

**Issues Detected:**
- Uncertain about backup_jobs table schema
- Need to validate field names against current schema
- Should not proceed without schema validation

**Next AI Session Must:**
1. Read start_here/MASTER_AI_PROMPT.md (mandatory)
2. Read source/current/api-documentation/DB_SCHEMA.md (validate schema)
3. Continue QCOW2 implementation from documented state
4. Do NOT assume any database field names

**SESSION ENDED - CONTEXT RELOAD REQUIRED**
```

---

**THIS SYSTEM ENSURES EVERY PIECE OF WORK CONNECTS TO THE BUSINESS PLAN AND IS TRACKED PROPERLY**

**NO MORE RANDOM CODING - EVERYTHING MUST SERVE THE GOAL OF DESTROYING VEEAM**

---

**Framework Complete:** ✅ **Ready for Implementation**  
**Next Step:** Team training and process implementation  
**Success Metric:** Zero unlinked work, zero rule violations  
**Review:** Weekly compliance monitoring
