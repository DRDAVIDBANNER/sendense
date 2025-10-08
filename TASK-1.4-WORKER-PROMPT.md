# Task 1.4 Worker Prompt: VMA/OMA → SNA/SHA Rename

**Task:** Rename all VMA/OMA references to SNA/SHA across the entire codebase  
**Job Sheet:** `2025-10-07-unified-nbd-architecture.md` (Task 1.4)  
**Priority:** HIGH (Naming consistency for project branding)  
**Estimated Time:** 2-3 hours  
**Pattern:** Similar to Task 1.3 (cloudstack → nbd refactor)

---

## 🎯 OBJECTIVE

Rename all appliance terminology from legacy names to new Sendense branding:
- **VMA** (VMware Migration Appliance) → **SNA** (Sendense Node Appliance)
- **OMA** (OSSEA Migration Appliance) → **SHA** (Sendense Hub Appliance)

**Why:** Project branding consistency. The platform is now "Sendense", not MigrateKit.

---

## 📋 WHAT YOU LEARNED FROM TASK 1.3

**Critical Lessons:**
1. ⚠️ **Type Assertions are Easy to Miss** - Grep for ALL references first
2. ⚠️ **Backup Files Too** - Don't forget `*.working`, `*.backup` files
3. ✅ **Test Compilation Often** - After each major phase
4. ✅ **Systematic Approach** - Don't claim "complete" until grep shows zero matches
5. ⚠️ **Document Acceptable Debt** - Some legacy references in comments are OK

**Project Overseer Will Check:**
- Compilation success
- Type assertion correctness
- Complete grep verification
- Documentation updates

---

## 📂 CURRENT STATE (What Exists)

**Directories:**
```
source/current/
├── vma/                    # ← Rename to sna/
├── vma-api-server/         # ← Rename to sna-api-server/
├── oma/                    # ← Rename to sha/
└── sendense-backup-client/ # ← Already refactored (Phase 1)
```

**Binaries (25+ files in source/current/):**
```
vma-api-server-fixed
vma-api-server-multi-disk-debug
vma-api-server-v1.10.0-power-management
... (20+ more)
```

---

## 🚀 IMPLEMENTATION PLAN

### **Phase A: Discovery & Assessment** (15 minutes)

**Step 1: Find ALL VMA references**
```bash
cd /home/oma_admin/sendense/source/current

# Case-sensitive VMA (likely struct/type names):
grep -r "VMA" --include="*.go" . | wc -l

# Case-insensitive vma (variables, functions, imports):
grep -ri "vma" --include="*.go" . | wc -l

# Save detailed list:
grep -ri "vma" --include="*.go" . > /tmp/vma-references.txt
```

**Step 2: Find ALL OMA references**
```bash
# Case-sensitive OMA:
grep -r "OMA" --include="*.go" . | wc -l

# Case-insensitive oma:
grep -ri "oma" --include="*.go" . | wc -l

# Save detailed list:
grep -ri "oma" --include="*.go" . > /tmp/oma-references.txt
```

**Step 3: Review the lists**
```bash
# Check what you're dealing with:
head -50 /tmp/vma-references.txt
head -50 /tmp/oma-references.txt

# Estimate: How many files? How many references?
```

**Report back:** "Found X VMA references in Y files, Z OMA references in W files"

---

### **Phase B: Directory Rename** (10 minutes)

**Step 1: Rename VMA directories**
```bash
cd /home/oma_admin/sendense/source/current

# Rename vma/ to sna/:
mv vma/ sna/

# Rename vma-api-server/ to sna-api-server/:
mv vma-api-server/ sna-api-server/

# Verify:
ls -la | grep -E "sna|vma"
# Should see: sna/, sna-api-server/ (no vma)
```

**Step 2: Check OMA directory**
```bash
# Does oma/ directory exist?
ls -la | grep oma

# If exists, rename to sha/:
mv oma/ sha/

# If it doesn't exist, note that and continue
```

---

### **Phase C: Import Path Updates** (30 minutes)

**This is the CRITICAL phase - take your time!**

**Step 1: Update VMA imports**
```bash
# Find all files with VMA imports:
grep -r "vma" --include="*.go" . | grep import | cut -d: -f1 | sort -u

# For EACH file with VMA imports:
# 1. Open the file
# 2. Change import paths:
#    "...vma..." → "...sna..."
#    "...vma-api-server..." → "...sna-api-server..."

# Example (use your editor):
# Before: import "github.com/vexxhost/migratekit/internal/vma/client"
# After:  import "github.com/vexxhost/migratekit/internal/sna/client"
```

**Step 2: Update OMA imports (if applicable)**
```bash
# Find all files with OMA imports:
grep -r "oma" --include="*.go" . | grep import | cut -d: -f1 | sort -u

# For EACH file with OMA imports:
# Change import paths:
#    "...oma..." → "...sha..."
```

**Step 3: Test compilation after imports**
```bash
# Test SNA API server:
cd /home/oma_admin/sendense/source/current/sna-api-server
go build -o test-sna-api 2>&1 | tee /tmp/sna-build-errors.txt

# If errors, read them:
cat /tmp/sna-build-errors.txt

# Fix import errors before continuing
```

---

### **Phase D: Code Reference Updates** (45 minutes)

**This is where Task 1.3 missed type assertions - BE THOROUGH!**

**Step 1: Find struct definitions**
```bash
# Find all VMA structs:
grep -r "type.*VMA" --include="*.go" .

# Find all OMA structs:
grep -r "type.*OMA" --include="*.go" .
```

**Step 2: Find variable names**
```bash
# Find vma variables:
grep -r "vma[A-Z]" --include="*.go" . | head -20

# Example: vmaClient, vmaAPI, vmaService
```

**Step 3: Find function names**
```bash
# Find VMA functions:
grep -r "func.*VMA" --include="*.go" .
grep -r "func.*Vma" --include="*.go" .
```

**Step 4: ⚠️ CRITICAL: Find type assertions**
```bash
# This is what Task 1.3 missed!
# Find all type assertions with VMA/OMA:
grep -r "\*vma\." --include="*.go" .
grep -r "\*oma\." --include="*.go" .
grep -r "\.(*vma\." --include="*.go" .
grep -r "\.(*oma\." --include="*.go" .

# Example pattern that MUST be updated:
# if vmaClient, ok := client.(*vma.Client); ok {
# Should become:
# if snaClient, ok := client.(*sna.Client); ok {
```

**Step 5: Systematic replacement**

For EACH file with VMA/OMA references:
1. Open the file
2. Replace struct names: `VMAClient` → `SNAClient`
3. Replace variables: `vmaClient` → `snaClient`
4. Replace functions: `GetVMAStatus()` → `GetSNAStatus()`
5. Replace type assertions: `(*vma.Client)` → `(*sna.Client)`
6. Update comments and logs

**Step 6: Update backup files**
```bash
# Don't forget these!
find . -name "*.working" -o -name "*.backup" | grep -E "vma|oma"

# Update any backup files found
```

---

### **Phase E: Binary Rename** (15 minutes)

**Step 1: Rename VMA binaries**
```bash
cd /home/oma_admin/sendense/source/current

# List all vma-api-server binaries:
ls -la | grep vma-api-server

# Rename each one:
for file in vma-api-server-*; do
    newname="${file/vma-api-server/sna-api-server}"
    mv "$file" "$newname"
    echo "Renamed: $file → $newname"
done

# Verify:
ls -la | grep -E "vma-api-server|sna-api-server"
# Should see ONLY sna-api-server-* files
```

**Step 2: Update build scripts (if they exist)**
```bash
# Check for Makefile or build scripts:
find . -name "Makefile" -o -name "build.sh" -o -name "*.mk"

# Update any references to vma-api-server in build scripts
```

---

### **Phase F: Final Compilation & Testing** (20 minutes)

**Step 1: Clean build SNA API Server**
```bash
cd /home/oma_admin/sendense/source/current/sna-api-server

# Clean previous builds:
go clean

# Build with verbose output:
go build -v -o sna-api-server-test-build 2>&1 | tee /tmp/final-build.log

# Check result:
if [ -f sna-api-server-test-build ]; then
    echo "✅ SUCCESS: Binary built"
    ls -lh sna-api-server-test-build
else
    echo "❌ FAILED: Check /tmp/final-build.log"
    cat /tmp/final-build.log
fi
```

**Step 2: Test binary (if applicable)**
```bash
# Try --help:
./sna-api-server-test-build --help 2>&1 | head -20

# Look for any remaining VMA references in output
```

**Step 3: Build SHA components (if oma/ was renamed)**
```bash
cd /home/oma_admin/sendense/source/current/sha

# If this directory exists, build it:
go build -v 2>&1 | tee /tmp/sha-build.log
```

**Step 4: Verify zero VMA/OMA references**
```bash
cd /home/oma_admin/sendense/source/current

# Final grep verification:
echo "=== VMA References Remaining ==="
grep -ri "vma" --include="*.go" . | wc -l

echo "=== OMA References Remaining ==="
grep -ri "oma" --include="*.go" . | wc -l

# Acceptable: Comments/logs mentioning "VMA" for historical context
# NOT acceptable: Import paths, struct names, type assertions with VMA/OMA
```

**Step 5: Clean up test binaries**
```bash
rm sna-api-server-test-build
```

---

## 📊 SUCCESS CRITERIA

Before reporting "Task 1.4 Complete":
- [ ] ✅ All directories renamed (vma→sna, vma-api-server→sna-api-server, oma→sha)
- [ ] ✅ All imports updated (no "vma" or "oma" in import paths)
- [ ] ✅ All struct names updated (VMA* → SNA*, OMA* → SHA*)
- [ ] ✅ All variables updated (vma* → sna*, oma* → sha*)
- [ ] ✅ All type assertions updated (verified with grep)
- [ ] ✅ All binaries renamed (sna-api-server-*)
- [ ] ✅ Backup files updated (*.working, *.backup)
- [ ] ✅ SNA API Server compiles cleanly
- [ ] ✅ SHA components compile (if applicable)
- [ ] ✅ Final grep shows minimal references (comments only)

---

## ⚠️ COMMON PITFALLS (From Task 1.3)

**1. Missing Type Assertions**
```go
// ❌ MISSED in Task 1.3:
if vmaClient, ok := client.(*vma.Client); ok {

// ✅ Should be:
if snaClient, ok := client.(*sna.Client); ok {
```

**2. Forgetting Backup Files**
```
vmware_nbdkit.go.working-libnbd-backup  # ← Must be updated too!
```

**3. Claiming "Complete" Too Early**
- Task 1.3 was marked complete but had 2 compilation errors
- Project Overseer had to fix them
- **Don't let this happen again!**

**4. Not Testing Compilation**
- Always test `go build` before claiming complete
- Read the error messages carefully

---

## 📝 REPORTING FORMAT

When you complete each phase, report:

**Phase A Complete:**
```
✅ Discovery complete:
- Found X VMA references in Y files
- Found Z OMA references in W files
- Saved reference lists to /tmp/
```

**Phase B Complete:**
```
✅ Directories renamed:
- vma/ → sna/ ✅
- vma-api-server/ → sna-api-server/ ✅
- oma/ → sha/ (if existed) ✅
```

**Phase C Complete:**
```
✅ Imports updated:
- Updated X files with VMA imports
- Updated Y files with OMA imports
- SNA API Server test build: [SUCCESS/FAILED]
```

**Phase D Complete:**
```
✅ Code references updated:
- Struct names: N changes
- Variables: M changes
- Type assertions: P changes
- Functions: Q changes
```

**Phase E Complete:**
```
✅ Binaries renamed:
- Renamed 25+ vma-api-server-* → sna-api-server-*
- Build scripts updated (if applicable)
```

**Phase F Complete:**
```
✅ Final verification:
- SNA API Server: COMPILES ✅ (20MB binary)
- SHA components: COMPILES ✅ (if applicable)
- Final grep: X VMA refs (all in comments), Y OMA refs (all in comments)
```

---

## 🎯 FINAL DELIVERABLE

When fully complete, provide:

1. **Summary Statement:**
   ```
   ✅ TASK 1.4 COMPLETE - VMA/OMA → SNA/SHA Refactor Done!
   
   Directories renamed: vma/ → sna/, vma-api-server/ → sna-api-server/, oma/ → sha/
   Binaries renamed: 25+ sna-api-server-* files
   Code updated: X structs, Y variables, Z type assertions, W functions
   Compilation: SNA API Server builds cleanly (20MB)
   Verification: Final grep shows only acceptable legacy references in comments
   ```

2. **Files Modified Count:**
   - Number of Go files changed
   - Number of directories renamed
   - Number of binaries renamed

3. **Compilation Evidence:**
   - SNA API Server binary size
   - SHA components status (if applicable)
   - Zero compilation errors

---

## 🚨 IF YOU GET STUCK

**Common Issues & Solutions:**

**1. Compilation errors after import updates:**
- Read the error message carefully
- Check for missed import path updates
- Look for type assertion mismatches

**2. Too many references to update:**
- Work file by file, don't try to do everything at once
- Use find/replace in your editor (but be careful!)
- Test compilation after each file

**3. Unsure if a reference should be updated:**
- Import paths: YES, always update
- Struct names: YES, always update
- Type assertions: YES, always update (CRITICAL!)
- Comments mentioning "VMA" historically: NO, acceptable debt
- Log messages: Update if easy, document if skipped

**4. Can't find a reference source:**
- Use: `grep -rn "exact_string" --include="*.go" .`
- The `-n` shows line numbers

---

## ✅ START NOW!

Begin with **Phase A: Discovery & Assessment**

Report back with your findings, then proceed through phases B-F systematically.

**Remember:** Quality over speed. Project Overseer WILL check your work!

**GO!** 🚀
