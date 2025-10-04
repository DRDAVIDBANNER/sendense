# Claude 4.5 Handoff Prompt - Failover Visibility GUI Integration

**Copy this entire prompt to start a new Claude session for GUI work**

---

## 🎯 **YOUR MISSION**

Integrate failover/rollback visibility into the MigrateKit OSSEA GUI to match the UX quality of replication jobs. The backend is **100% complete** - you just need to consume the new APIs.

---

## 📚 **PROJECT CONTEXT**

**Project**: MigrateKit OSSEA - VMware to CloudStack migration platform  
**Working Directory**: `/home/pgrayson/migratekit-cloudstack`  
**Backend API**: OMA API running on `localhost:8082` (or test servers 10.245.246.147/.148)  
**Backend Version**: v2.32.0-unified-jobs-api ✅ Complete  

**Problem**: 
- Failover and rollback jobs have poor GUI visibility
- Error messages expose technical details ("virt-v2v", "VirtIO", device paths)
- Failed jobs disappear quickly from view
- Users don't know what failed or what to do

**Backend Solution** (Already Complete):
- ✅ Error sanitization (all technical terms converted to user-friendly language)
- ✅ Persistent operation summaries (failed jobs stored forever)
- ✅ Unified jobs API (combines replication + failover + rollback)
- ✅ Actionable steps (every error has user guidance)

**Your Task**:
Build GUI components to display this sanitized information beautifully.

---

## 🔌 **NEW API ENDPOINTS (Already Implemented & Live)**

### **Endpoint 1: Unified Recent Jobs**

```
GET /api/v1/vm-contexts/{context_id}/recent-jobs
```

**Returns**: All operations for a VM (replication + failover + rollback) in one list

**Example Response**:
```json
{
  "context_id": "ctx-pgtest1-20251003-140708",
  "count": 2,
  "jobs": [
    {
      "job_id": "e5de9b1b-5159-49e2-95e5-be644da2b7fb",
      "external_job_id": "unified-test-failover-pgtest1-1759510017",
      "job_type": "test_failover",
      "status": "failed",
      "progress": 60.0,
      "display_name": "Test Failover",
      "error_message": "KVM driver installation failed - compatibility issue",
      "error_category": "compatibility",
      "actionable_steps": [
        "Try live failover (no driver modification)",
        "Verify VM is Windows-based"
      ],
      "failed_step": "Preparing Drivers for Compatibility",
      "duration_seconds": 25,
      "steps_completed": 3,
      "steps_total": 5
    },
    {
      "job_type": "replication",
      "status": "completed",
      "display_name": "Replication Completed"
    }
  ]
}
```

---

### **Endpoint 2: VM Context (Enhanced)**

```
GET /api/v1/vm-contexts/{vm_name}
```

**Now includes**: `last_operation` field with persistent failure info

```json
{
  "context": { ... },
  "last_operation": {
    "status": "failed",
    "failed_step": "Preparing Drivers for Compatibility",
    "error_message": "KVM driver installation failed - compatibility issue",
    "actionable_steps": ["Try live failover (no driver modification)"],
    "progress": 60.0
  }
}
```

---

## 📊 **TEST DATA AVAILABLE**

**Server**: 10.245.246.148  
**VMs with Failed Operations**: pgtest1, pgtest3

**Test API**:
```bash
# Get unified jobs for pgtest1
curl "http://10.245.246.148:8082/api/v1/vm-contexts/ctx-pgtest1-20251003-140708/recent-jobs" | jq .

# Get VM context with last operation
curl "http://10.245.246.148:8082/api/v1/vm-contexts/pgtest1" | jq ".last_operation"
```

**Expected Error Message** (Sanitized):
- ✅ "KVM driver installation failed - compatibility issue"
- ✅ Step: "Preparing Drivers for Compatibility"
- ✅ Action: "Try live failover (no driver modification)"

**What You Should NEVER See**:
- ❌ "virt-v2v"
- ❌ "VirtIO"
- ❌ "/dev/vdc"
- ❌ "virtio-driver-injection"

---

## 🎨 **REQUIREMENTS**

### **Visual Design**

**Unified Job List** (Primary Requirement):
```
Recent Operations
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

❌ Test Failover - Failed (60%)
   Issue: KVM driver installation failed - compatibility issue
   
   What you can do:
   • Try live failover (no driver modification)
   • Verify VM is Windows-based
   
   Failed at: Preparing Drivers for Compatibility
   3 minutes ago

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ Replication Completed
   26.4 GB transferred
   3 hours ago

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Key Features**:
- Show ALL job types in one list (no separation)
- Failed jobs show error message + actionable steps
- Consistent visual style (colors, icons, spacing)
- Sorted by time (newest first)
- Persistent (don't disappear)

---

### **Persistent Error Banner** (Secondary Requirement):

Show at top of VM details when `last_operation.status === 'failed'`:

```
┌────────────────────────────────────────────────┐
│ ⚠️  Last Operation Failed                      │
├────────────────────────────────────────────────┤
│ Test Failover failed at step 3 of 5 (60%)      │
│                                                │
│ Issue: KVM driver installation failed          │
│                                                │
│ What you can do:                               │
│ • Try live failover (no driver modification)   │
│ • Verify VM is Windows-based                   │
│                                                │
│ [Dismiss] [Try Live Failover] [View Details]   │
└────────────────────────────────────────────────┘
```

---

## 🚨 **CRITICAL REQUIREMENTS**

### **MUST DO**:
1. Use ONLY sanitized error messages (never technical details)
2. Display actionable steps prominently
3. Show failover/rollback in same list as replications
4. Make failed jobs visible until dismissed
5. Use user-friendly step names

### **MUST NOT**:
1. ❌ NEVER show "virt-v2v", "VirtIO", or technical tool names
2. ❌ NEVER show device paths or file paths
3. ❌ NEVER use internal step names (use display_name from API)
4. ❌ NEVER hide errors from users
5. ❌ NEVER show technical_details field to non-admin users

---

## 📋 **IMPLEMENTATION CHECKLIST**

- [ ] **Task 1**: Create `UnifiedJobList.tsx` component
  - Fetch from `/api/v1/vm-contexts/{context_id}/recent-jobs`
  - Display all job types with consistent styling
  - Show sanitized errors for failed jobs
  - Display actionable steps
  
- [ ] **Task 2**: Create `OperationErrorBanner.tsx` component
  - Read `last_operation` from VM context
  - Show when status is "failed"
  - Display actionable steps
  - Add quick action buttons
  
- [ ] **Task 3**: Create `JobErrorDetailsModal.tsx` component
  - Show detailed error information
  - Display step progress
  - List all actionable steps
  - Optional admin technical details section
  
- [ ] **Task 4**: Integrate into existing VM views
  - Replace current failover job display
  - Add unified list to VM details page
  - Add error banner at top of page
  
- [ ] **Task 5**: Testing
  - Test with failed operations
  - Verify no technical terms visible
  - Verify actionable steps are helpful
  - Test persistence across refreshes

---

## 🧪 **HOW TO TEST**

### **Test on Server 10.245.246.148**

**Step 1**: Trigger a failed test failover (or use existing pgtest1/pgtest3 data)

**Step 2**: Check backend has sanitized data:
```bash
curl "http://10.245.246.148:8082/api/v1/vm-contexts/ctx-pgtest1-20251003-140708/recent-jobs" | jq ".jobs[0]"
```

**Expected**:
```json
{
  "error_message": "KVM driver installation failed - compatibility issue",
  "actionable_steps": ["Try live failover (no driver modification)"],
  "failed_step": "Preparing Drivers for Compatibility"
}
```

**Step 3**: Verify GUI displays this exactly as shown (no "virt-v2v", no "VirtIO")

---

## 📖 **KEY DOCUMENTATION TO READ**

**Before starting, read**:
1. `/home/pgrayson/migratekit-cloudstack/docs/api/UNIFIED_JOBS_API.md` - Complete API documentation
2. `/home/pgrayson/migratekit-cloudstack/AI_Helper/FAILOVER_VISIBILITY_ENHANCEMENT_JOB_SHEET.md` - Full requirements

**For reference**:
- Error sanitization logic: `source/current/oma/failover/error_sanitizer.go`
- Step name mapping: `source/current/oma/failover/step_display_names.go`

---

## 🎯 **EXPECTED OUTCOME**

### **Before** (Current State):
```
User: "My test failover failed"
GUI shows: Job disappeared from view
User thinks: "What happened? Where did it go?"
User action: Ask admin to check logs
```

### **After** (Your Implementation):
```
User: "My test failover failed"  
GUI shows: 
  "Test Failover Failed (60%)
   Issue: KVM driver installation failed - compatibility issue
   
   What you can do:
   • Try live failover (no driver modification)
   • Verify VM is Windows-based"

User thinks: "Oh, I should try live failover instead!"
User action: Clicks [Try Live Failover] button
```

---

## 🚀 **QUICK START**

```bash
# 1. Verify APIs work
curl "http://10.245.246.148:8082/api/v1/vm-contexts/ctx-pgtest1-20251003-140708/recent-jobs"

# 2. Find GUI codebase
ls /home/pgrayson/migratekit-cloudstack | grep -E "gui|frontend|dashboard|migration-dashboard"

# 3. Locate VM details/context components

# 4. Start implementing:
#    - UnifiedJobList component first (shows immediate value)
#    - Then OperationErrorBanner
#    - Then error detail modal
#    - Finally integrate everything

# 5. Test with pgtest1 and pgtest3 (both have failed operations)
```

---

## ⚡ **IMPORTANT NOTES**

### **Backend Team Has**:
- ✅ Sanitized ALL errors automatically
- ✅ Stored summaries in database
- ✅ Created unified API endpoint
- ✅ Tested on 2 servers (10.245.246.147/.148)
- ✅ Verified sanitization works (no technical leaks)

### **You Need To**:
- Build GUI components to display this data
- Use consistent styling with existing UI
- Test that users see actionable guidance
- Ensure failed jobs don't disappear

### **Zero Backend Work Required**:
All APIs are live and tested. Pure frontend integration task.

---

## 📞 **SUPPORT**

**If APIs aren't working**:
```bash
# Check service status
ssh oma_admin@10.245.246.148 'sudo systemctl status oma-api'

# Check health
curl "http://10.245.246.148:8082/health"
```

**If you need backend changes**:
Document what's needed - backend team can assist.

**If you find technical terms in API**:
This is a bug - report it. All errors should be sanitized.

---

**Ready to integrate! All backend work is complete.** 🎉

---

## 📝 **COPY-PASTE PROMPT FOR NEW CLAUDE SESSION**

```
I'm working on MigrateKit OSSEA GUI integration for failover visibility enhancement.

CONTEXT:
- Project: VMware to CloudStack migration platform with React/Next.js GUI
- Working directory: /home/pgrayson/migratekit-cloudstack
- Backend: OMA API v2.32.0 (fully complete with unified jobs API)
- Test servers: 10.245.246.147 and 10.245.246.148

BACKEND COMPLETE (Don't modify):
- New API: GET /api/v1/vm-contexts/{context_id}/recent-jobs
- Enhanced: GET /api/v1/vm-contexts/{vm_name} (includes last_operation)
- Error sanitization (no "virt-v2v", "VirtIO", device paths)
- Persistent operation summaries in database
- Actionable steps for every error

YOUR TASK:
Build GUI components to display failover/rollback jobs with sanitized errors:
1. UnifiedJobList component - shows ALL operations (replication + failover + rollback)
2. OperationErrorBanner - persistent failure notification at top of VM view
3. JobErrorDetailsModal - detailed error view with actionable steps
4. Integration into existing VM details/context views

REQUIREMENTS:
- Show failover/rollback jobs in same location as replications (unified list)
- Display sanitized error messages only (API provides these)
- Show actionable steps prominently for failed operations
- Make failed jobs visible persistently (don't disappear)
- NEVER show technical terms: virt-v2v, VirtIO, /dev/vdX, etc.

TEST DATA:
- Server: 10.245.246.148
- VMs: pgtest1 and pgtest3 both have failed test failovers
- API test: curl "http://10.245.246.148:8082/api/v1/vm-contexts/ctx-pgtest1-20251003-140708/recent-jobs"

DOCUMENTATION:
- Read: /home/pgrayson/migratekit-cloudstack/docs/api/UNIFIED_JOBS_API.md
- Read: /home/pgrayson/migratekit-cloudstack/docs/gui/FAILOVER_VISIBILITY_GUI_INTEGRATION_PROMPT.md
- Reference: /home/pgrayson/migratekit-cloudstack/AI_Helper/FAILOVER_VISIBILITY_ENHANCEMENT_JOB_SHEET.md

SUCCESS CRITERIA:
- Users see what failed without asking
- Users know what action to take next
- Failed jobs visible for days (not seconds)
- Consistent UX with replication jobs
- Zero technical jargon in error messages

Please start by:
1. Reading the UNIFIED_JOBS_API.md documentation
2. Testing the API endpoints to see the data structure
3. Finding the existing GUI codebase
4. Implementing the UnifiedJobList component first (quick win)
5. Then the error banner and modal

Let's make failover errors as clear and actionable as possible!
```

---

**This prompt is ready to copy-paste into a new Claude session** ✅


