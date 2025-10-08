# Incremental Backup Test - October 8, 2025

**Purpose:** Validate incremental backup functionality using recorded change_id from full backup

**Prerequisite:** Full backup completed with change_id recorded  
**Full Backup:** `backup-pgtest1-1759913694`  
**change_id:** `52 ed 45 cf 23 2c 6a f0-a5 26 59 71 b7 9f 1f b3/4442`

---

## Test Plan

### 1. Make Changes to VM
**Action:** Modify some files in VM to create changed blocks  
**Expected:** VMware CBT will track these changes

### 2. Start Incremental Backup
```bash
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"incremental"}'
```

### 3. Expected Behavior
- SHA queries database for latest change_id
- Passes change_id to SNA as `MIGRATEKIT_PREVIOUS_CHANGE_ID`
- sendense-backup-client uses VMware QueryChangedDiskAreas() API
- Only changed blocks transferred
- New change_id recorded for next incremental

### 4. Success Criteria
- ✅ Incremental backup completes successfully
- ✅ New change_id recorded (different from previous)
- ✅ Transfer size much smaller than full backup (~90% reduction expected)
- ✅ New QCOW2 references parent backup (backing file)
- ✅ Backup chain properly linked in database

---

## Test Execution Log

**Started:** [timestamp]  
**Job ID:** [will be recorded]  
**Status:** Running...

