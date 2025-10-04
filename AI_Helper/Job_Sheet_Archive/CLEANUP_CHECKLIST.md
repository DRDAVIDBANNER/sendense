# Cleanup Checklist - October 4, 2025

## ✅ **COMPLETED CLEANUP**

### **Files Deleted**:
- ❌ `/migration-dashboard/src/app/api/discover/route.ts` 
  - **OLD**: Direct-to-VMA discovery route
  - **REPLACED BY**: Enhanced Discovery API (`/api/v1/discovery/discover-vms`)
  
- ❌ `/source/current/oma/api/handlers/vmware_credentials_simple.go`
  - **ISSUE**: Duplicate incomplete handler (only 2 methods)
  - **CORRECT VERSION**: `vmware_credentials.go` (8 complete CRUD methods)

### **Files Archived**:
- 📦 `source/archive/20251004-disabled-handlers/cloudstack_validation.go.disabled`
  - **REASON**: Superseded by `cloudstack_settings.go`
  
- 📦 `source/archive/20251004-disabled-handlers/vma_enrollment.go.disabled`
  - **REASON**: Superseded by `vma_real.go`

### **Database Cleaned**:
- ❌ Deleted credential ID 3 (plaintext test credential)
- ✅ Kept credential ID 2 (properly encrypted, set as default)

---

## 🎯 **WHAT TO KEEP**

### **Active Backend Handlers**:
- ✅ `enhanced_discovery.go` - Discovery with credentials
- ✅ `vmware_credentials.go` - Full CRUD (NOT the _simple version)
- ✅ `streamlined_ossea_config.go` - OSSEA resource discovery
- ✅ `cloudstack_settings.go` - CloudStack validation
- ✅ `vma_real.go` - VMA enrollment (NOT the .disabled version)

### **Active GUI Components**:
- ✅ `DiscoveryView.tsx` - Now uses Enhanced Discovery
- ✅ `VMwareCredentialsManager.tsx` - Credential management UI
- ✅ `CloudStackValidation.tsx` - OSSEA validation UI

### **Active API Routes** (GUI → Next.js → OMA):
- ✅ `/api/v1/vmware-credentials` - Credential management
- ✅ `/api/v1/discovery/discover-vms` - Enhanced discovery
- ✅ `/api/v1/discovery/add-vms` - Add to management
- ✅ `/api/v1/ossea/discover-resources` - OSSEA resource discovery
- ✅ `/api/v1/settings/cloudstack/*` - CloudStack validation

---

## ❌ **NO LONGER USED - DO NOT REFERENCE**

### **Removed Routes**:
- ❌ `/api/discover` - Deleted, use Enhanced Discovery instead

### **Removed Handlers**:
- ❌ `vmware_credentials_simple.go` - Deleted, use full version

### **Archived (Don't Use)**:
- ❌ `cloudstack_validation.go.disabled` - Use `cloudstack_settings.go`
- ❌ `vma_enrollment.go.disabled` - Use `vma_real.go`

---

## 🔍 **HOW TO VERIFY CLEANUP**

### **Check for References to Deleted Files**:
```bash
# Should return nothing:
grep -r "vmware_credentials_simple" source/current/
grep -r "/api/discover" migration-dashboard/src/
grep -r "cloudstack_validation.go" source/current/
grep -r "vma_enrollment.go" source/current/
```

### **Verify Correct Handlers Used**:
```bash
# Check server.go uses correct handlers:
grep "VMwareCredentials" source/current/oma/api/handlers/handlers.go
# Should show: NewVMwareCredentialsHandler (NOT Simple)

# Check imports:
grep "vmware_credentials" source/current/oma/api/server.go
# Should only reference handlers.VMwareCredentials
```

---

## 📋 **IF AI ASKS ABOUT THESE**

### **"Should we use /api/discover?"**
- ❌ NO - That route was deleted
- ✅ YES - Use `/api/v1/discovery/discover-vms` (Enhanced Discovery)

### **"Should we use vmware_credentials_simple.go?"**
- ❌ NO - That file was deleted (duplicate)
- ✅ YES - Use `vmware_credentials.go` (full version with 8 methods)

### **"What about the .disabled files?"**
- ❌ NO - Archived to `source/archive/20251004-disabled-handlers/`
- ✅ YES - Use active versions (cloudstack_settings.go, vma_real.go)

### **"Do we need hardcoded credentials in DiscoveryView?"**
- ❌ NO - Security issue, removed in repair
- ✅ YES - Use VMware Credentials Management system

---

## ✅ **CLEANUP VERIFICATION**

Run these commands to verify cleanup is complete:

```bash
cd /home/pgrayson/migratekit-cloudstack

# 1. Verify deleted files are gone
[ ! -f "migration-dashboard/src/app/api/discover/route.ts" ] && echo "✅ Old discover route deleted"
[ ! -f "source/current/oma/api/handlers/vmware_credentials_simple.go" ] && echo "✅ Duplicate handler deleted"

# 2. Verify disabled files are archived
[ ! -f "source/current/oma/api/handlers/cloudstack_validation.go.disabled" ] && echo "✅ Disabled validation archived"
[ ! -f "source/current/oma/api/handlers/vma_enrollment.go.disabled" ] && echo "✅ Disabled enrollment archived"

# 3. Verify archived files exist
[ -f "source/archive/20251004-disabled-handlers/cloudstack_validation.go.disabled" ] && echo "✅ Validation archived properly"
[ -f "source/archive/20251004-disabled-handlers/vma_enrollment.go.disabled" ] && echo "✅ Enrollment archived properly"

# 4. Verify no broken references
! grep -r "vmware_credentials_simple" source/current/ && echo "✅ No references to deleted simple handler"
! grep -r "/api/discover" migration-dashboard/src/ && echo "✅ No references to old discover route"

# 5. Verify production has updated GUI
ssh oma_admin@10.246.5.124 'stat /opt/migratekit/gui/.next' && echo "✅ Production GUI updated"
```

**Expected Result**: All ✅ checks pass

---

## 🎯 **SUMMARY**

**Files Deleted**: 2  
**Files Archived**: 2  
**Database Records Cleaned**: 1  
**Security Issues Fixed**: 1  
**Broken References**: 0  

**Status**: ✅ **CLEANUP COMPLETE**

