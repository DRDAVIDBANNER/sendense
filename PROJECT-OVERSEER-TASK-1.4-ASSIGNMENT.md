# Project Overseer: Task 1.4 Assignment

**Date:** October 7, 2025  
**Assigned By:** Project Overseer  
**Task:** VMA/OMA → SNA/SHA Terminology Rename  
**Status:** 🔴 **READY FOR WORKER**

---

## 📋 WHAT WAS DONE

### **Job Sheet Updated**
✅ **File:** `job-sheets/2025-10-07-unified-nbd-architecture.md`  
✅ **Task 1.4:** Completely rewritten with new scope  
✅ **Old scope:** VMA API call format updates (deferred)  
✅ **New scope:** Complete VMA/OMA → SNA/SHA rename across codebase

**Changes:**
- Full task breakdown (Phases A-E)
- Pattern applied from Task 1.3 (cloudstack → nbd refactor)
- Critical lessons emphasized (type assertions, backup files, compilation testing)
- Estimated complexity documented (2-3 hours, 2 directories + 25+ binaries)

---

### **Worker Prompt Created**
✅ **File:** `TASK-1.4-WORKER-PROMPT.md` (7 KB, comprehensive)  
✅ **Content:** Complete step-by-step implementation guide

**Includes:**
- 6 implementation phases (Discovery → Testing)
- Bash commands for each step
- Common pitfalls from Task 1.3
- Success criteria checklist
- Reporting format requirements
- Troubleshooting guide

**Key Sections:**
1. **Phase A:** Discovery & Assessment (grep all references)
2. **Phase B:** Directory Rename (vma→sna, vma-api-server→sna-api-server, oma→sha)
3. **Phase C:** Import Path Updates (critical for compilation)
4. **Phase D:** Code Reference Updates (structs, variables, **type assertions**)
5. **Phase E:** Binary Rename (25+ files)
6. **Phase F:** Compilation & Testing (verification)

---

### **Quick Handover Created**
✅ **File:** `TASK-1.4-HANDOVER-TO-WORKER.md` (3 KB, quick brief)  
✅ **Content:** Fast-start guide for worker

**Includes:**
- Quick brief (what, why, pattern, time)
- Reading list (priorities)
- Start commands
- Critical reminders
- Scope overview
- Success checklist

---

### **CHANGELOG Updated**
✅ **File:** `start_here/CHANGELOG.md`  
✅ **Entry:** Task 1.4 Redefined (VMA/OMA → SNA/SHA)

**Details:**
- Scope expanded from VMA API updates to full terminology rename
- 3 directories, 25+ binaries, 100s of code references
- Pattern similar to Task 1.3 but larger scope
- Status: 🔴 IN PROGRESS - Worker assigned
- Purpose: Complete Sendense branding consistency

---

## 🎯 SCOPE SUMMARY

**What Worker Will Rename:**

### **Directories (3):**
```
vma/               → sna/
vma-api-server/    → sna-api-server/
oma/               → sha/ (if exists)
```

### **Binaries (25+):**
```
vma-api-server-*   → sna-api-server-*
```

### **Code References (100s):**
- Import paths: `"...vma..."` → `"...sna..."`
- Struct names: `VMAClient` → `SNAClient`
- Variables: `vmaClient` → `snaClient`
- Type assertions: `(*vma.Client)` → `(*sna.Client)` ⚠️ **CRITICAL**
- Functions: `GetVMAStatus()` → `GetSNAStatus()`
- Comments and logs

---

## ⚠️ CRITICAL LESSONS FROM TASK 1.3

**What Went Wrong:**
1. Worker claimed "complete" with compilation errors
2. Missed 2 type assertions (parallel_incremental.go, vmware_nbdkit.go)
3. Project Overseer had to fix and revalidate

**What Worker Must Do Differently:**
1. ✅ **Grep FIRST** - Find all references before starting
2. ✅ **Test compilation OFTEN** - After each phase
3. ✅ **Update backup files** - *.working, *.backup
4. ✅ **Verify type assertions** - grep for `(*vma.`, `(*oma.`
5. ✅ **Don't claim complete** - Until `go build` succeeds

**Project Overseer Will Check:**
- Compilation success ✅
- Type assertion correctness ✅
- Complete grep verification ✅
- Backup file updates ✅
- Documentation updates ✅

---

## 📊 SUCCESS CRITERIA

Worker must achieve ALL of these:
- [ ] ✅ Directories renamed (verified with `ls`)
- [ ] ✅ Imports updated (no vma/oma in import paths)
- [ ] ✅ Structs renamed (VMA* → SNA*, OMA* → SHA*)
- [ ] ✅ Variables renamed (vma* → sna*, oma* → sha*)
- [ ] ✅ Type assertions updated (**critical!**)
- [ ] ✅ Binaries renamed (25+ files)
- [ ] ✅ Backup files updated
- [ ] ✅ SNA API Server compiles cleanly
- [ ] ✅ SHA components compile (if applicable)
- [ ] ✅ Final grep shows <10 references (comments only)

---

## 🚀 WORKER INSTRUCTIONS

**Give worker this command:**

> "Read **TASK-1.4-HANDOVER-TO-WORKER.md** for quick brief, then follow **TASK-1.4-WORKER-PROMPT.md** for step-by-step instructions. Start with Phase A (Discovery), report back after each phase. Goal: Complete VMA/OMA → SNA/SHA rename with clean compilation. Estimated time: 2-3 hours."

**Or simpler:**

> "Execute Task 1.4: VMA/OMA → SNA/SHA rename. Start by reading TASK-1.4-WORKER-PROMPT.md. Report progress after each phase."

---

## 📁 FILES FOR WORKER

Worker needs to read these (in order):
1. `TASK-1.4-HANDOVER-TO-WORKER.md` - Quick brief
2. `TASK-1.4-WORKER-PROMPT.md` - Detailed instructions
3. `job-sheets/2025-10-07-unified-nbd-architecture.md` - Task 1.4 section
4. `TASK-1.3-COMPLETION-REPORT.md` - Learn from previous mistakes

---

## 🎯 EXPECTED OUTCOME

**When Complete:**
- All VMA/OMA terminology renamed to SNA/SHA
- SNA API Server compiles cleanly (20MB binary)
- SHA components compile (if applicable)
- No compilation errors
- Minimal grep references (<10, comments only)
- Ready for Project Overseer audit

**What Happens Next:**
1. Worker completes Task 1.4
2. Project Overseer audits work
3. If approved: Phase 1 complete! ✅
4. If issues: Overseer fixes and documents
5. Then: Proceed to Phase 2 (SHA API enhancements)

---

## 📝 PROJECT OVERSEER NOTES

**Task Assignment Decision:**
- Original Task 1.4 (VMA API updates) was too narrow
- Discovery showed VMA/OMA naming throughout codebase
- Expanded scope to complete terminology rename
- Better architectural alignment (consistent branding)
- Leverages Task 1.3 pattern (proven approach)

**Risk Assessment:**
- Similar complexity to Task 1.3
- Larger scope (2 directories vs 1 file)
- Higher risk of missed references
- Type assertions are critical pain point
- Worker has been warned extensively

**Mitigation Strategy:**
- Comprehensive prompt with bash commands
- Phase-by-phase reporting requirement
- Compilation testing after each phase
- Grep verification before claiming complete
- Project Overseer will audit rigorously

**Estimated Timeline:**
- Worker: 2-3 hours execution
- Overseer: 30 minutes audit
- Total: ~3 hours to complete Task 1.4
- Then: Phase 1 done! 🎉

---

## ✅ READY TO ASSIGN

**Status:** 🟢 **READY FOR WORKER**  
**Priority:** HIGH (blocks Phase 1 completion)  
**Complexity:** MEDIUM-HIGH (similar to Task 1.3, larger scope)  
**Documentation:** ✅ Complete (prompt, handover, job sheet, changelog)  
**Worker Guidance:** ✅ Comprehensive (lessons from Task 1.3 applied)

---

**ASSIGN TASK NOW!** 🚀

---

*Project Overseer ready to audit upon worker completion*  
*All documentation updated and compliance verified*  
*Task 1.4 ready for execution*
