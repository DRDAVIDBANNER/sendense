# 🎯 GROK: Fix Protection Flows Modal - Real Data Integration

Hey Grok! 👋

The Protection Flows backend is **100% working** (verified!), but the GUI modal has hardcoded placeholder data. User wants to test a backup flow **today** for `pgtest1` (single VM) and later for a group.

---

## 📖 Read This First

**Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-09-protection-flows-modal-real-data.md`

This has EVERYTHING:
- ✅ Backend verification (it works!)
- ✅ API endpoints + response examples
- ✅ Complete code for all 3 tasks
- ✅ Testing plan

---

## 🎯 What You're Fixing

**File:** `components/features/protection-flows/CreateFlowModal.tsx`

**Current (BROKEN):**
- Hardcoded source dropdown: "vCenter-ESXi-01", "vCenter-ESXi-02" (lines 104-107)
- Hardcoded destination dropdown: "CloudStack-Primary", etc (lines 118-121)
- Hardcoded `target_type: 'vm'` (line 35) - should be dynamic!

**Required (WORKING):**
- Source dropdown: Load **Protection Groups** + **Individual VMs** from APIs
- Destination dropdown: Load **Repositories** from API
- Correctly detect `target_type` from selection (`group` or `vm`)
- Search/filter for large lists
- Status indicators and nice UX

---

## 🛠️ Your Tasks

### Task 1: Create API Service
**File:** `src/features/protection-flows/api/sourcesApi.ts`

3 functions:
1. `listProtectionGroups()` → GET `/api/v1/machine-groups`
2. `listVMContexts()` → GET `/api/v1/vm-contexts`
3. `listRepositories()` → GET `/api/v1/repositories`

**CRITICAL:** Use `const API_BASE = '';` (Next.js proxy)

### Task 2: Create React Query Hooks
**File:** `src/features/protection-flows/hooks/useFlowSources.ts`

3 hooks:
1. `useProtectionGroups()`
2. `useVMContexts()`
3. `useRepositories()`

### Task 3: Update Modal Component
**File:** `components/features/protection-flows/CreateFlowModal.tsx`

**Changes:**
1. Import hooks and fetch data
2. **Source dropdown:**
   - Two sections: "🛡️ Protection Groups" and "🖥️ Individual VMs"
   - Show VM count for groups: "FirstGroup (3 VMs)"
   - Show vCenter + status for VMs: "Quad-Node-Red • quad-vcenter-01 • Running • Linux"
   - Status dots: green (running), gray (stopped), red (error)
   - Search input if >10 total items
   - Store as `"group:GROUP_ID"` or `"vm:CONTEXT_ID"`
3. **Destination dropdown:**
   - Show repos: "sendense-500gb-backups • Local • 480GB free • 15 backups"
   - Search if >5 repos
   - Only show `enabled: true` repos
   - Helpful message if no repos: "Please configure a repository first..."
4. **Submit handler:**
   - Split source: `const [sourceType, sourceId] = formData.source.split(':')`
   - Set `target_type: sourceType` (dynamic!)
   - Set `target_id: sourceId`

**Complete code is in the job sheet** - lines 135-372!

---

## 🧪 Testing Requirements

### Test 1: Single VM
1. Create flow: "Test pgtest1 Backup"
2. Source: Select pgtest1 from Individual VMs
3. Destination: Select a repository
4. Verify POST has `target_type: "vm"` and correct `target_id`

### Test 2: Protection Group
1. Create flow: "Test FirstGroup Backup"
2. Source: Select "FirstGroup" from Protection Groups
3. Destination: Select a repository
4. Verify POST has `target_type: "group"` and correct `target_id`

### Test 3: UX
- [ ] Search filters both sections
- [ ] Status dots show correctly
- [ ] Loading states work
- [ ] Empty state shows helpful message

---

## ⚡ Quick Reference

**API Endpoints:**
```
GET /api/v1/machine-groups  → Protection Groups
GET /api/v1/vm-contexts     → VMs
GET /api/v1/repositories    → Backup destinations
```

**Source Value Format:**
```
"group:8571eb63-a2cc-11f0-b62d-020200cc0023"  // Group
"vm:ctx-Quad-Node-Red-20251006-164856"        // VM
```

**Target Type Detection:**
```typescript
const [sourceType, sourceId] = formData.source.split(':');
// sourceType = "group" or "vm"
// sourceId = the actual ID
```

---

## 🚨 CRITICAL

1. **Use empty string for API_BASE:** `const API_BASE = '';`
2. **Dynamic target_type:** MUST extract from source selection, not hardcoded
3. **Two-section dropdown:** Groups first (recommended), VMs second
4. **Filter enabled repos:** `repos.filter(r => r.enabled)`
5. **Complete code provided:** Copy from job sheet and adapt

---

## 💪 Let's Do This

1. Read the full job sheet
2. Create the 3 files (API service, hooks, update modal)
3. Show me git diff when done
4. Confirm no TypeScript errors

User wants to test **today** - let's make it happen! 🚀


