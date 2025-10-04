# Legacy System Clarification - What to Use vs Avoid

**Document Version:** 1.0  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **CRITICAL - PREVENT AI CONFUSION**

---

## 🎯 PURPOSE

Prevent AI assistants from using legacy/deprecated code paths that still exist in the codebase but should NOT be used for new development.

**The Problem:**
- Multiple implementations exist (original vs enhanced)
- Legacy code still functional but deprecated
- AI assistants may choose wrong implementation
- Can cause architectural violations and performance issues

---

## 🚨 FAILOVER SYSTEM (CRITICAL CLARIFICATION)

### **✅ CURRENT/ACTIVE SYSTEM (Use This)**

**Unified Failover System:**
- **Files:** `source/current/oma/failover/unified_failover_*.go`
- **Status:** ✅ **PRODUCTION READY** - This is the live system
- **Features:** JobLog integration, Volume Daemon compliance, modular architecture
- **Handler:** Uses unified engine for all failover operations
- **API Endpoints:** Uses standard `/api/v1/failover/*` endpoints

**Key Files (ACTIVE - UNIFIED SYSTEM):**
```bash
source/current/oma/failover/
├── unified_failover_engine.go       # ✅ LIVE SYSTEM (PRIMARY)
├── unified_failover_config.go       # ✅ LIVE SYSTEM
└── validation.go                    # ✅ LIVE SYSTEM
```

### **❌ LEGACY SYSTEMS (Avoid All These)**

**Enhanced Failover System (LEGACY):**
- **Files:** `enhanced_*.go`
- **Status:** ❌ **LEGACY** - Replaced by unified system
- **Issues:** Superseded by unified approach

**Original Failover System (LEGACY):**  
- **Files:** `live_failover.go`, `test_failover.go`
- **Status:** ❌ **LEGACY** - Old monolithic system
- **Issues:** Volume Daemon violations, no JobLog integration

**Legacy Files (DO NOT USE ANY):**
```bash
source/current/oma/failover/
├── enhanced_live_failover.go        # ❌ LEGACY (superseded by unified)
├── enhanced_test_failover.go        # ❌ LEGACY (superseded by unified)
├── enhanced_cleanup_service.go      # ❌ LEGACY (superseded by unified)
├── enhanced_failover_wrapper.go     # ❌ LEGACY (superseded by unified)
├── live_failover.go                 # ❌ LEGACY (original system)
└── test_failover.go                 # ❌ LEGACY (original system)
```

**Why Legacy Systems Exist:**
- Transition safety (keep during unified system validation)  
- Backward compatibility for existing integrations
- Will be removed in future cleanup phase

**Critical Rule:** **ONLY use unified_failover_* files for any failover work**

---

## 🔧 VOLUME OPERATIONS (CRITICAL)

### **✅ CURRENT SYSTEM (Mandatory)**

**Volume Daemon (ONLY Approved Method):**
- **File:** `source/current/oma/common/volume_client.go`
- **Status:** ✅ **MANDATORY** - All volume operations must use this
- **Pattern:**
  ```go
  volumeClient := common.NewVolumeClient("http://localhost:8090")
  operation, err := volumeClient.AttachVolume(ctx, volumeID, vmID)
  ```

### **❌ LEGACY SYSTEM (Forbidden)**

**Direct CloudStack/Platform Calls:**
- **Pattern:** `osseaClient.AttachVolume()`, `osseaClient.DetachVolume()`
- **Status:** ❌ **FORBIDDEN** - Violates architecture
- **Issues:** No centralized management, device correlation failures, race conditions

**Critical Rule:** **NEVER use direct platform volume calls outside Volume Daemon**

---

## 📊 LOGGING SYSTEM

### **✅ CURRENT SYSTEM (Mandatory)**

**JobLog System:**
- **Files:** `source/current/oma/joblog/`
- **Status:** ✅ **MANDATORY** - All business logic must use this
- **Pattern:**
  ```go
  tracker := joblog.New(db, stdoutHandler, dbHandler)
  ctx, jobID, _ := tracker.StartJob(ctx, joblog.JobStart{...})
  tracker.RunStep(ctx, jobID, "step-name", func(ctx context.Context) error {...})
  tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
  ```

### **❌ LEGACY SYSTEM (Forbidden in Business Logic)**

**Direct Logging:**
- **Patterns:** `logrus.Info()`, `slog.Info()`, `fmt.Printf()`
- **Status:** ❌ **FORBIDDEN** in operation logic
- **Allowed:** Only in HTTP middleware for request-scoped metadata

**Critical Rule:** **ALL business logic operations MUST use JobLog**

---

## 🌐 API ENDPOINTS

### **✅ CURRENT SYSTEM (Active)**

**OMA (SHA) API Endpoints:**
- **File:** `source/current/oma/api/server.go`
- **Base:** `/api/v1/*`
- **Handlers:** `source/current/oma/api/handlers/*`

**VMA (SNA) API Endpoints:**
- **File:** `source/current/vma/api/server.go`  
- **Progress:** `source/current/vma/api/progress_handler.go`

### **❌ LEGACY ENDPOINTS (Check Before Using)**

**Potentially Legacy VMA Endpoints:**
- `/replicate` (legacy) → Use `/replications` (current)
- `/enable-cbt` (may not be implemented) → Check before using

**Rule:** **Search existing handlers before creating new endpoints**

---

## 📁 SOURCE AUTHORITY (CRITICAL)

### **✅ AUTHORITATIVE SOURCE**

**ONLY Canonical Location:**
- `source/current/` - ✅ **ONLY location for active development**

### **❌ DEPRECATED LOCATIONS (Avoid)**

**Do NOT Use:**
- `archive/` directories
- `archived_*/` directories  
- Top-level versioned directories
- Any code outside `source/current/`

**Rule:** **If code exists outside source/current/, it's archived/deprecated**

---

## 🔍 HOW TO IDENTIFY CURRENT vs LEGACY

### **Identification Rules**

**Current System Indicators:**
- ✅ Located in `source/current/`
- ✅ Uses Volume Daemon (`volume_client.go`)
- ✅ Uses JobLog (`internal/joblog`)
- ✅ Filename contains `enhanced_` or `unified_`
- ✅ Has recent git commits and active development

**Legacy System Indicators:**
- ❌ Direct platform SDK calls (`osseaClient.*`)
- ❌ Direct logging (`logrus`, `slog` in business logic)
- ❌ Filename without `enhanced_` prefix
- ❌ Missing JobLog integration
- ❌ No recent commits or marked as deprecated

### **When in Doubt**

**Quick Validation:**
```bash
# Check if file uses Volume Daemon
grep -l "volume_client.go\|NewVolumeClient" source/current/oma/failover/*.go

# Check if file uses JobLog
grep -l "joblog\|StartJob\|RunStep" source/current/oma/failover/*.go

# Check git activity (active vs stagnant)
git log --oneline --since="30 days ago" source/current/oma/failover/
```

**If Still Unclear:**
- ❌ **DO NOT GUESS** - Ask for clarification
- ❌ **DO NOT USE** potentially legacy code
- ✅ **VERIFY** with project leadership before proceeding

---

## 🎯 ENFORCEMENT CHECKLIST

### **Before Using Any Existing Code:**
- [ ] **Confirm location**: Is it in `source/current/`?
- [ ] **Check architecture**: Does it use Volume Daemon?
- [ ] **Check logging**: Does it use JobLog?
- [ ] **Check naming**: Does it follow current patterns?
- [ ] **Check activity**: Has it been updated recently?

### **Red Flags (Stop Immediately):**
- 🚨 **Direct volume calls**: Any `osseaClient.AttachVolume()`
- 🚨 **Direct logging**: `logrus.Info()` in business logic
- 🚨 **Non-enhanced failover**: Files without `enhanced_` prefix
- 🚨 **Archive references**: Any code outside `source/current/`

---

## 🔄 SAFE MIGRATION STRATEGY

### **Legacy Code Removal Plan**

**Phase 1: Enhanced System Validation (Current)**
- Keep legacy code for safety during enhanced system testing
- Enhanced system proven in production
- No new development uses legacy paths

**Phase 2: Legacy Deprecation (Future)**
- Mark legacy files with deprecation warnings
- Update all references to use enhanced system
- Remove legacy endpoint routing

**Phase 3: Legacy Removal (Future)**
- Move legacy files to archive/
- Clean up unused handlers
- Complete architecture consolidation

**Current Status:** **Phase 1** - Enhanced system is live, legacy kept for transition safety

---

## 🎯 AI ASSISTANT GUIDANCE

### **Safe Development Approach**

**When Working on Failover:**
1. ✅ **Use unified_failover_*.go files ONLY**
2. ✅ **Verify Volume Daemon usage**
3. ✅ **Verify JobLog integration**
4. ❌ **Never touch enhanced_*.go files (legacy)**
5. ❌ **Never touch original live_failover.go or test_failover.go (legacy)**

**When Working on API Endpoints:**
1. ✅ **Search existing handlers first**
2. ✅ **Use consistent routing patterns**
3. ✅ **Follow established authentication**
4. ❌ **Never create duplicate endpoints**

**When Working with Database:**
1. ✅ **Validate field names against DB_SCHEMA.md**
2. ✅ **Use repository patterns**
3. ✅ **Follow migration-based schema changes**
4. ❌ **Never assume field names exist**

### **Emergency Context Reload Triggers**

**Trigger Immediate Reload If:**
- Using any file mentioned in "Legacy System" section
- Making database field assumptions
- Creating endpoints without searching existing handlers
- Forgetting Volume Daemon or JobLog requirements
- Using deprecated terminology (VMA/OMA)

---

**THIS DOCUMENT PREVENTS AI SESSIONS FROM USING THE WRONG CODE PATHS AND VIOLATING SENDENSE ARCHITECTURE**

---

**Document Owner:** Architecture Team  
**Purpose:** Prevent legacy system usage  
**Maintenance:** Updated when legacy systems removed  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **ACTIVE GUIDANCE**
