# 🐛 CRITICAL BUG FIX: Protection Flow Execution - Invalid job_type ENUM

**Date:** 2025-10-09  
**Priority:** CRITICAL  
**Status:** Blocking user testing

---

## 🔥 The Problem

**User clicked "Execute" on a protection flow → 500 error**

**Backend Error:**
```
Failed to execute protection flow: failed to start job tracking: 
failed to create job record: Error 1265 (01000): Data truncated for column 'job_type' at row 1
```

**Root Cause:** `protection_flow_service.go` line 181:
```go
JobType:   "protection_flow",  // ❌ NOT IN DATABASE ENUM!
```

**Database ENUM Values (job_tracking table):**
```sql
'cleanup','failover','migration','cloudstack','volume_daemon','linstor','virtio',
'ossea','scheduler','discovery','bulk-operations','group-management',
'conflict-detection','phantom-detection','schedule_management','schedule_control'
```

**Missing:** `'protection_flow'` and `'backup'`

---

## 🎯 The Fix (Choose ONE approach)

### Option 1: Add to Database ENUM (Proper Solution) ✅ RECOMMENDED

**File:** Create new migration file:
`/source/current/sha/database/migrations/20251009130000_add_protection_flow_job_types.up.sql`

```sql
-- Add protection flow related job types to job_tracking.job_type ENUM
-- Context: Protection Flows Engine needs dedicated job types for tracking

ALTER TABLE job_tracking 
MODIFY COLUMN job_type ENUM(
    'cleanup',
    'failover',
    'migration',
    'cloudstack',
    'volume_daemon',
    'linstor',
    'virtio',
    'ossea',
    'scheduler',
    'discovery',
    'bulk-operations',
    'group-management',
    'conflict-detection',
    'phantom-detection',
    'schedule_management',
    'schedule_control',
    'protection_flow',      -- NEW: Protection flow orchestration
    'backup_execution'       -- NEW: Individual backup job execution
) DEFAULT NULL;
```

**Rollback file:**
`/source/current/sha/database/migrations/20251009130000_add_protection_flow_job_types.down.sql`

```sql
-- Rollback protection flow job types
-- WARNING: Will fail if any rows exist with these types

ALTER TABLE job_tracking 
MODIFY COLUMN job_type ENUM(
    'cleanup',
    'failover',
    'migration',
    'cloudstack',
    'volume_daemon',
    'linstor',
    'virtio',
    'ossea',
    'scheduler',
    'discovery',
    'bulk-operations',
    'group-management',
    'conflict-detection',
    'phantom-detection',
    'schedule_management',
    'schedule_control'
) DEFAULT NULL;
```

**Execute Migration:**
```bash
mysql -u oma_user -poma_password migratekit_oma < /home/oma_admin/sendense/source/current/sha/database/migrations/20251009130000_add_protection_flow_job_types.up.sql
```

**Verification:**
```bash
mysql -u oma_user -poma_password migratekit_oma -e "SHOW COLUMNS FROM job_tracking WHERE Field='job_type';" | grep protection_flow
```

---

### Option 2: Use Existing ENUM Value (Quick Workaround)

**File:** `/source/current/sha/services/protection_flow_service.go` line 181

**Change from:**
```go
jobCtx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
    JobType:   "protection_flow",  // ❌ NOT IN ENUM
    Operation: fmt.Sprintf("execute_%s_flow", executionType),
    Owner:     &owner,
    Metadata:  map[string]interface{}{"flow_id": flowID},
})
```

**Change to:**
```go
jobCtx, jobID, err := s.jobTracker.StartJob(ctx, joblog.JobStart{
    JobType:   "scheduler",  // ✅ EXISTS IN ENUM - flows are scheduled operations
    Operation: fmt.Sprintf("execute_%s_flow", executionType),
    Owner:     &owner,
    Metadata:  map[string]interface{}{
        "flow_id": flowID,
        "flow_component": "protection_flow_service",  // Track actual component
    },
})
```

---

## 🔧 RECOMMENDED: Option 1 (Proper Fix)

**Why:**
1. ✅ Protection Flows is a new system component - deserves its own job_type
2. ✅ Makes job tracking queries cleaner and more meaningful
3. ✅ Consistent with how other components have dedicated types
4. ✅ Future-proof for analytics and monitoring

**Steps:**
1. Create migration files (provided above)
2. Execute migration with correct database credentials
3. Verify ENUM was updated
4. Restart backend service: `sudo systemctl restart sendense-hub.service`
5. Test flow execution

---

## 🧪 Testing

**After Fix:**

1. **Create Test Flow:**
   - Name: "Quick Test Backup"
   - Type: Backup
   - Source: Any VM or group
   - Destination: Any repository

2. **Execute Flow:**
   - Click "Run Now" or use API: `POST /api/v1/protection-flows/{id}/execute`
   - Should return: `{"execution_id": "...", "message": "Flow execution started"}`

3. **Verify in Logs:**
```bash
sudo journalctl -u sendense-hub.service --since "1 minute ago" | grep "protection_flow"
```

Should see:
```
level=info msg="Starting flow execution" flow_id=... execution_type=manual
level=info msg="Processing backup flow" flow_id=... target_type=...
```

4. **Verify Job Tracking:**
```bash
mysql -u oma_user -poma_password migratekit_oma -e "SELECT id, job_type, operation, status FROM job_tracking WHERE job_type='protection_flow' ORDER BY created_at DESC LIMIT 3;"
```

5. **Verify Flow Execution:**
```bash
curl -s http://localhost:8082/api/v1/protection-flows/{FLOW_ID}/executions | jq '.'
```

Should show execution with status "running" or "completed".

---

## 📁 Files Involved

**Migration Files (CREATE THESE):**
- `/source/current/sha/database/migrations/20251009130000_add_protection_flow_job_types.up.sql`
- `/source/current/sha/database/migrations/20251009130000_add_protection_flow_job_types.down.sql`

**OR Workaround File (EDIT THIS):**
- `/source/current/sha/services/protection_flow_service.go` (line 181)

---

## 🚨 CRITICAL NOTES

1. **This blocks ALL flow execution** - user can't test anything until fixed
2. **Option 1 is proper** - adds the ENUM value to database
3. **Option 2 is quick workaround** - uses existing "scheduler" type
4. **After fix, restart service:** `sudo systemctl restart sendense-hub.service`
5. **User is waiting** - wants to test pgtest1 backup today

---

## 📊 Additional Context

**Database Credentials:**
```
User: oma_user
Password: oma_password
Database: migratekit_oma
```

**Service Control:**
```bash
sudo systemctl status sendense-hub.service
sudo systemctl restart sendense-hub.service
sudo journalctl -u sendense-hub.service -f  # Watch logs
```

**Backend Location:**
```
Binary: /usr/local/bin/sendense-hub (symlink to /home/oma_admin/sendense/source/builds/sendense-hub-v2.25.1-route-fix)
Source: /home/oma_admin/sendense/source/current/sha/
```

---

## ✅ Success Criteria

- [ ] Migration executed successfully (Option 1) OR code updated (Option 2)
- [ ] Backend restarted
- [ ] Test flow execution returns 200 (not 500)
- [ ] Execution record created in database
- [ ] Job tracking record created with correct job_type
- [ ] Backend logs show "Processing backup flow"
- [ ] User can test pgtest1 backup

---

**Next:** Choose an option and implement. Option 1 is recommended for proper solution.

