# Grok Prompt: Backup & Restore Improvements

## 🎯 **MISSION**
Continue improving the Sendense backup and restore platform with focus on automation, performance, and user experience enhancements.

## 📋 **CURRENT STATUS**
- ✅ **Individual VM Flows Machines Panel** - FIXED (SHA v2.25.7)
- ✅ **Protection Flows Engine** - Complete backend + GUI integration
- ✅ **File-Level Restore** - Multi-partition mounting + navigation
- ✅ **Backup Chain Management** - Incremental detection + cleanup
- ✅ **VM Discovery** - credential_id storage + multi-vCenter support

## 🚀 **RECOMMENDED FOCUS AREAS**

### **Option 1: Backup Job Cleanup Automation** (HIGH PRIORITY)
**Problem:** Stuck/failed backup jobs leave orphaned processes and incomplete files
**Impact:** Manual cleanup required, storage waste, user confusion

**Tasks:**
1. **Auto-cleanup stuck jobs** - Detect timeout, kill processes, clean database
2. **Orphaned process cleanup** - Kill `qemu-nbd` processes from failed jobs
3. **Incomplete file cleanup** - Remove partial QCOW2 files
4. **Database cleanup** - Mark failed `backup_disk` records as failed
5. **User feedback** - Better error messages and cleanup status

**Files to modify:**
- `source/current/sha/services/protection_flow_service.go`
- `source/current/sha/services/backup_service.go`
- `source/current/sha/restore/mount_manager.go`

### **Option 2: Performance Optimizations** (MEDIUM PRIORITY)
**Problem:** VM discovery slow (17s for 98 VMs), API response times could be better

**Tasks:**
1. **VM discovery optimization** - Parallel processing, caching
2. **Database query optimization** - Indexes, query improvements
3. **API response caching** - Cache frequently accessed data
4. **Frontend optimization** - Lazy loading, pagination

### **Option 3: GUI Enhancements** (MEDIUM PRIORITY)
**Problem:** User experience could be improved with better feedback and controls

**Tasks:**
1. **Real-time progress updates** - WebSocket or polling for live updates
2. **Better error handling** - User-friendly error messages
3. **Backup management** - Retention policies, storage analytics
4. **Loading states** - Better UX during operations

## 🔧 **TECHNICAL CONTEXT**

### **Key Files**
- `source/current/sha/` - Backend Go services
- `source/current/sendense-gui/` - Next.js frontend
- `source/current/api-documentation/` - API docs
- `start_here/CHANGELOG.md` - Change history

### **Active Services**
- **SHA API:** `sha-api-v2.25.7-vm-context-by-id` (port 8082)
- **Frontend:** Next.js dev server (port 3000)
- **Database:** MariaDB with normalized schema

### **Test Environment**
- **VMs:** pgtest1, pgtest2, pgtest3 available for testing
- **Protection Flows:** Individual and group-based flows working
- **File Restore:** Multi-partition mounting operational

## 📚 **DOCUMENTATION**
- **Handover:** `HANDOVER-2025-10-10-BACKUP-RESTORE-IMPROVEMENTS.md`
- **API Docs:** `source/current/api-documentation/OMA.md`
- **Database Schema:** `source/current/api-documentation/DB_SCHEMA.md`
- **Project Status:** `start_here/PHASE_1_CONTEXT_HELPER.md`

## 🎯 **SUCCESS CRITERIA**
1. **Automation:** Stuck jobs auto-cleanup without manual intervention
2. **Performance:** VM discovery under 10s for 100+ VMs
3. **User Experience:** Clear feedback, better error handling
4. **Reliability:** Robust error handling and recovery

## 🚨 **CRITICAL RULES**
- **NO SIMULATION CODE** - Only real, production-ready implementations
- **FOLLOW CURSORULES** - Update documentation, commit changes
- **TEST THOROUGHLY** - Verify all changes work end-to-end
- **MAINTAIN COMPATIBILITY** - Don't break existing functionality

## 🎉 **READY TO START**
The system is in excellent shape with all major components working. Choose your focus area and let's make it even better!

**Current commit:** `3294900` - "Fix individual VM flows machines panel"
**Git status:** All changes committed and pushed to main branch
