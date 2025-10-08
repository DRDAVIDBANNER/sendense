# Task 1.4 Handover to Worker

**Date:** October 7, 2025  
**From:** Project Overseer  
**To:** Implementation Worker  
**Task:** VMA/OMA → SNA/SHA Rename (Task 1.4)

---

## 🎯 QUICK BRIEF

**What You're Doing:**
Renaming all VMA (VMware Migration Appliance) and OMA (OSSEA Migration Appliance) references to SNA (Sendense Node Appliance) and SHA (Sendense Hub Appliance) throughout the codebase.

**Why:**
Project branding consistency. This is the Sendense platform, not MigrateKit.

**Pattern:**
Same type of refactor you just did in Task 1.3 (cloudstack → nbd), but bigger scope (2 directories + 25+ binaries).

**Estimated Time:** 2-3 hours

---

## 📋 WHAT TO READ

1. **Main Prompt:** `TASK-1.4-WORKER-PROMPT.md` ← **READ THIS FIRST**
   - Complete step-by-step instructions
   - Lessons learned from Task 1.3
   - All the pitfalls to avoid
   - Success criteria checklist

2. **Job Sheet Reference:** `job-sheets/2025-10-07-unified-nbd-architecture.md`
   - Scroll to Task 1.4 section
   - See official task requirements

3. **Context from Task 1.3:** `TASK-1.3-COMPLETION-REPORT.md`
   - What you did before (similar pattern)
   - What Project Overseer had to fix (type assertions!)
   - Don't make the same mistakes

---

## 🚀 START COMMAND

```bash
# Go to work directory:
cd /home/oma_admin/sendense/source/current

# Read the full prompt:
cat /home/oma_admin/sendense/TASK-1.4-WORKER-PROMPT.md

# Begin with Phase A (Discovery):
grep -r "VMA" --include="*.go" . | wc -l
grep -ri "vma" --include="*.go" . | wc -l
```

---

## ⚠️ CRITICAL REMINDERS

1. **Type Assertions:** Don't miss them (Task 1.3 mistake!)
   - `if vmaClient, ok := client.(*vma.Client); ok {`
   - Must become: `if snaClient, ok := client.(*sna.Client); ok {`

2. **Test Compilation:** After EVERY phase
   - Don't claim "complete" until `go build` succeeds

3. **Backup Files:** Update them too
   - `*.working`, `*.backup` files

4. **Grep Verification:** Before claiming complete
   - `grep -ri "vma" --include="*.go" .` should show minimal results (comments only)

5. **Report Progress:** After each phase
   - Project Overseer will track your progress

---

## 📊 SCOPE OVERVIEW

**What You're Renaming:**

**Directories:** (3)
- `vma/` → `sna/`
- `vma-api-server/` → `sna-api-server/`
- `oma/` → `sha/` (if exists)

**Binaries:** (25+ files)
- All `vma-api-server-*` → `sna-api-server-*`

**Code References:** (100s)
- Import paths with vma/oma
- Struct names (VMA*/OMA*)
- Variable names (vma*/oma*)
- Type assertions (**CRITICAL!**)
- Function names
- Comments/logs

---

## ✅ SUCCESS = ALL THESE TRUE

- [ ] Directories renamed (verified with `ls`)
- [ ] Imports updated (no vma/oma in import statements)
- [ ] Structs renamed (VMA* → SNA*, OMA* → SHA*)
- [ ] Variables renamed (vma* → sna*, oma* → sha*)
- [ ] Type assertions updated (**verified with grep**)
- [ ] Binaries renamed (25+ files)
- [ ] Backup files updated
- [ ] SNA API Server compiles ✅
- [ ] SHA components compile ✅ (if applicable)
- [ ] Final grep shows <10 acceptable references (comments only)

---

## 🎯 GO TIME!

**Your job:** Complete Task 1.4 following the detailed prompt

**Expected outcome:** Clean compilation, proper renaming, ready for Project Overseer audit

**Time limit:** 2-3 hours (quality over speed)

**Questions?** Check the prompt first, it has everything you need

---

**START WITH:** `TASK-1.4-WORKER-PROMPT.md`

**REPORT BACK:** After each phase (A-F)

**FINISH WITH:** Summary statement + compilation evidence

---

**GOOD LUCK! 🚀**

---

*Project Overseer will audit your work upon completion*  
*Remember: Task 1.3 had missed type assertions - don't repeat that mistake!*
