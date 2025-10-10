# Phase 9: Integration Testing Guide
## Backup Job Telemetry Framework

**Date:** October 10, 2025  
**Status:** Ready for Testing  
**Prerequisites:** Database schema applied ✅, Binaries built (pending)

---

## Pre-Testing Checklist

### 1. Database Schema Validation

✅ **Schema Status:** All telemetry fields already exist

```bash
# Verify backup_jobs telemetry fields
mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT COLUMN_NAME, COLUMN_TYPE, COLUMN_DEFAULT 
  FROM information_schema.COLUMNS 
  WHERE TABLE_SCHEMA='migratekit_oma' 
  AND TABLE_NAME='backup_jobs' 
  AND COLUMN_NAME IN ('current_phase', 'transfer_speed_bps', 'eta_seconds', 'progress_percent', 'last_telemetry_at');"
```

**Expected Output:**
```
current_phase         | varchar(50)    | pending
transfer_speed_bps    | bigint(20)     | 0
eta_seconds           | int(11)        | 0
progress_percent      | decimal(5,2)   | 0.00
last_telemetry_at     | datetime       | NULL
```

```bash
# Verify backup_disks progress field
mysql -u oma_admin -poma_password migratekit_oma -e "
  SELECT COLUMN_NAME, COLUMN_TYPE, COLUMN_DEFAULT 
  FROM information_schema.COLUMNS 
  WHERE TABLE_SCHEMA='migratekit_oma' 
  AND TABLE_NAME='backup_disks' 
  AND COLUMN_NAME='progress_percent';"
```

**Expected Output:**
```
progress_percent | decimal(5,2) | 0.00
```

```bash
# Verify telemetry index for stale detection
mysql -u oma_user -poma_password migratekit_oma -e "
  SHOW INDEXES FROM backup_jobs WHERE Key_name LIKE '%telemetry%';"
```

**Expected:** Index `idx_last_telemetry` on (status, last_telemetry_at)

---

### 2. Build New Binaries

```bash
cd /home/oma_admin/sendense

# Build SHA with telemetry framework
cd source/current/sha
go build -o ../../../../builds/sendense-hub-v2.26.0-telemetry cmd/main.go
echo "✅ SHA binary built: $(ls -lh ../../../../builds/sendense-hub-v2.26.0-telemetry)"

# Build SBC with telemetry sender
cd ../../sendense-backup-client
go build -o ../../../builds/sendense-backup-client-v1.0.2-telemetry main.go
echo "✅ SBC binary built: $(ls -lh ../../../builds/sendense-backup-client-v1.0.2-telemetry)"
```

---

### 3. Deploy SHA Binary

```bash
# Stop current SHA service
sudo systemctl stop sendense-hub

# Backup current binary
sudo cp /usr/local/bin/sendense-hub /usr/local/bin/sendense-hub.backup-$(date +%Y%m%d)

# Deploy new binary
sudo cp /home/oma_admin/sendense/builds/sendense-hub-v2.26.0-telemetry /usr/local/bin/sendense-hub
sudo chmod +x /usr/local/bin/sendense-hub

# Start SHA service
sudo systemctl start sendense-hub

# Verify service started
sudo systemctl status sendense-hub

# Watch logs for telemetry handler and stale detector
sudo journalctl -u sendense-hub -f --since="1 minute ago" | grep -E "Telemetry|Stale|telemetry"
```

**Expected Log Messages:**
```
✅ Telemetry API endpoints enabled (Real-time SBC progress tracking)
✅ Telemetry API routes registered: POST /api/v1/telemetry/{job_type}/{job_id}
🚨 Stale job detector started
```

---

### 4. Deploy SBC Binary to SNA

```bash
# Copy to SNA (replace with actual SNA IP)
scp /home/oma_admin/sendense/builds/sendense-backup-client-v1.0.2-telemetry sna:/usr/local/bin/sendense-backup-client

# SSH to SNA and verify
ssh sna "ls -lh /usr/local/bin/sendense-backup-client"
```

---

## Test Scenarios

### Test 1: bytes_transferred Fix Verification

**Objective:** Verify that `bytes_transferred` now populates correctly in `backup_jobs` table

**Steps:**
1. Start a backup job via GUI or API
2. Wait for backup to complete
3. Query database to verify `bytes_transferred` is populated

**Test Commands:**
```bash
# Wait for a backup to complete, then check:
mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    id,
    vm_name,
    status,
    bytes_transferred,
    total_bytes,
    ROUND(bytes_transferred / (1024*1024*1024), 2) AS gb_transferred,
    created_at,
    completed_at
  FROM backup_jobs 
  WHERE status='completed' 
  ORDER BY completed_at DESC 
  LIMIT 5;" --vertical
```

**Expected Result:**
- `bytes_transferred` > 0 (not zero!)
- `bytes_transferred` matches sum of disk sizes
- GUI machine modal shows correct backup size

**Validation Query:**
```bash
# Verify aggregation from backup_disks
mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    bj.id AS backup_job_id,
    bj.bytes_transferred AS parent_bytes,
    SUM(bd.bytes_transferred) AS sum_disk_bytes,
    (bj.bytes_transferred = SUM(bd.bytes_transferred)) AS aggregation_correct
  FROM backup_jobs bj
  JOIN backup_disks bd ON bd.backup_job_id = bj.id
  WHERE bj.status = 'completed'
  GROUP BY bj.id
  ORDER BY bj.completed_at DESC
  LIMIT 5;" --vertical
```

**Success Criteria:**
- ✅ `parent_bytes` > 0
- ✅ `aggregation_correct` = 1 (true)
- ✅ Machine modal displays accurate sizes

---

### Test 2: Real-Time Telemetry Updates

**Objective:** Verify telemetry updates arrive in real-time during backup

**Steps:**
1. Start a backup job
2. Monitor database updates in real-time
3. Verify updates arrive every ~5 seconds

**Test Commands:**

**Terminal 1 - Real-Time Monitor:**
```bash
watch -n 2 'mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    id,
    status,
    current_phase,
    ROUND(progress_percent, 1) AS progress,
    ROUND(bytes_transferred / (1024*1024), 0) AS mb_transferred,
    ROUND(transfer_speed_bps / (1024*1024), 2) AS mbps,
    eta_seconds,
    last_telemetry_at
  FROM backup_jobs 
  WHERE status IN (\"running\", \"stalled\") 
  ORDER BY created_at DESC 
  LIMIT 1;" --vertical'
```

**Terminal 2 - SHA Telemetry Logs:**
```bash
sudo journalctl -u sendense-hub -f | grep -E "telemetry|Telemetry"
```

**Terminal 3 - SBC Telemetry Logs (on SNA):**
```bash
ssh sna "tail -f /var/log/sendense-backup-client.log | grep -E 'SHA telemetry|Telemetry'"
```

**Expected Behavior:**
- `progress_percent` increases from 0 to 100
- `last_telemetry_at` updates every ~5 seconds
- `transfer_speed_bps` shows realistic values (e.g., 100-500 MB/s)
- `eta_seconds` decreases as backup progresses
- `current_phase` transitions: "snapshot" → "transferring" → "finalizing"

**Success Criteria:**
- ✅ Updates arrive every 5 seconds (±1s)
- ✅ Progress increases smoothly
- ✅ No gaps > 10 seconds in `last_telemetry_at`
- ✅ SHA logs show "Telemetry received and processed"
- ✅ SBC logs show "SHA telemetry update sent"

---

### Test 3: Per-Disk Progress Tracking

**Objective:** Verify per-disk progress updates for multi-disk VMs

**Test Commands:**
```bash
# Monitor per-disk progress during backup
watch -n 2 'mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    backup_job_id,
    disk_index,
    ROUND(progress_percent, 1) AS progress,
    status,
    ROUND(bytes_transferred / (1024*1024), 0) AS mb_transferred,
    ROUND(size_gb, 1) AS disk_size_gb
  FROM backup_disks 
  WHERE backup_job_id IN (
    SELECT id FROM backup_jobs WHERE status=\"running\" ORDER BY created_at DESC LIMIT 1
  );"'
```

**Expected Behavior:**
- Each disk shows individual `progress_percent`
- Disks can complete at different times
- Parent job progress is aggregate of all disks

**Success Criteria:**
- ✅ All disks tracked independently
- ✅ Per-disk progress matches parent progress
- ✅ Disk status transitions correctly

---

### Test 4: Stale Job Detection

**Objective:** Verify automatic detection and marking of stale jobs

**Steps:**
1. Start a backup job
2. Kill SBC process mid-backup (simulate client crash)
3. Wait 60 seconds → job marked "stalled"
4. Wait 5 minutes → job marked "failed"

**Test Commands:**

**Start Backup:**
```bash
# Via GUI or API - get the job_id
JOB_ID=$(mysql -u oma_user -poma_password migratekit_oma -N -e "SELECT id FROM backup_jobs WHERE status='running' ORDER BY created_at DESC LIMIT 1;")
echo "Monitoring job: $JOB_ID"
```

**Kill SBC (on SNA):**
```bash
ssh sna "sudo pkill -9 sendense-backup-client"
```

**Monitor Stale Detection:**
```bash
# Watch for status transitions
watch -n 5 'mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    id,
    status,
    TIMESTAMPDIFF(SECOND, last_telemetry_at, NOW()) AS seconds_since_update,
    error_message,
    last_telemetry_at,
    NOW() AS current_time
  FROM backup_jobs 
  WHERE id=\"$JOB_ID\";" --vertical'
```

**Check Stale Detector Logs:**
```bash
sudo journalctl -u sendense-hub --since="5 minutes ago" | grep -E "stale|Stale|failed"
```

**Expected Timeline:**
- T+0s: Job running, telemetry updates every 5s
- T+0s: Kill SBC, telemetry stops
- T+60s: Job marked "stalled", error_message = "Job stalled due to no telemetry updates for 60s"
- T+300s: Job marked "failed", error_message = "Job failed due to no telemetry updates for 5m0s"

**Success Criteria:**
- ✅ Job marked "stalled" after 60s
- ✅ Job marked "failed" after 300s
- ✅ `error_message` contains stale detection reason
- ✅ `completed_at` timestamp set when marked failed
- ✅ SHA logs show stale detection messages

---

### Test 5: Backward Compatibility

**Objective:** Verify old SBC (without telemetry) still works

**Steps:**
1. Use old SBC binary (v1.0.1-port-fix)
2. Start backup
3. Verify backup completes successfully
4. Confirm no telemetry data (expected)

**Test Commands:**
```bash
# Deploy old SBC binary on SNA
ssh sna "sudo cp /usr/local/bin/sendense-backup-client.old /usr/local/bin/sendense-backup-client"

# Start backup via GUI/API

# Monitor - should see no telemetry updates
mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    id,
    status,
    bytes_transferred,
    last_telemetry_at,
    progress_percent
  FROM backup_jobs 
  WHERE status='running' 
  ORDER BY created_at DESC 
  LIMIT 1;" --vertical
```

**Expected Behavior:**
- Backup completes successfully
- `last_telemetry_at` = NULL (no telemetry)
- `progress_percent` = 0.0 (no progress updates)
- `bytes_transferred` = 0 until completion (then aggregated via CompleteBackup fix)

**Success Criteria:**
- ✅ Backup completes without errors
- ✅ No telemetry data (expected)
- ✅ `bytes_transferred` still aggregates at completion (Phase 1 fix works)

---

## Validation Queries

### Overall System Health

```bash
# Check telemetry coverage
mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    status,
    COUNT(*) AS count,
    SUM(CASE WHEN last_telemetry_at IS NOT NULL THEN 1 ELSE 0 END) AS with_telemetry,
    SUM(CASE WHEN last_telemetry_at IS NULL THEN 1 ELSE 0 END) AS without_telemetry
  FROM backup_jobs
  WHERE created_at > DATE_SUB(NOW(), INTERVAL 24 HOUR)
  GROUP BY status;"
```

### Telemetry Update Frequency

```bash
# Check telemetry update intervals
mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    id,
    status,
    TIMESTAMPDIFF(SECOND, created_at, last_telemetry_at) AS total_duration_sec,
    ROUND(bytes_transferred / (1024*1024*1024), 2) AS gb_transferred,
    ROUND(transfer_speed_bps / (1024*1024), 2) AS avg_mbps
  FROM backup_jobs 
  WHERE last_telemetry_at IS NOT NULL
  AND created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)
  ORDER BY created_at DESC;"
```

### Stale Job Statistics

```bash
# Check stale/failed jobs
mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    status,
    COUNT(*) AS count,
    error_message
  FROM backup_jobs
  WHERE error_message LIKE '%stalled%' OR error_message LIKE '%no telemetry%'
  GROUP BY status, error_message;"
```

---

## Performance Benchmarks

### Expected Performance Metrics

**Telemetry Overhead:**
- Network: < 1 KB per update (JSON payload)
- Frequency: 1 update per 5 seconds
- Database: < 10ms per update
- CPU: Negligible

**Backup Performance:**
- Should match baseline (no telemetry overhead)
- Transfer speed: 100-500 MB/s (hardware dependent)
- Telemetry should NOT slow down transfer

**Comparison Test:**
```bash
# Test backup WITH telemetry
time ssh sna "sendense-backup-client --vm pgtest1 --job-id test-telemetry-1"

# Test backup WITHOUT telemetry (old SBC)
time ssh sna "sendense-backup-client.old --vm pgtest1 --job-id test-no-telemetry-1"

# Compare times - should be within 1-2%
```

---

## Troubleshooting

### Issue: Telemetry updates not arriving

**Symptoms:**
- `last_telemetry_at` = NULL during running backup
- No progress updates in GUI
- No telemetry logs in SHA

**Diagnosis:**
```bash
# Check SHA telemetry handler registered
curl -s http://localhost:8082/api/v1/health | jq

# Check SBC can reach SHA API
ssh sna "curl -X POST http://localhost:8082/api/v1/telemetry/backup/test-123 \
  -H 'Content-Type: application/json' \
  -d '{\"job_type\":\"backup\",\"status\":\"running\"}'"

# Expected: 200 OK or 400 Bad Request (but NOT connection refused)

# Check SHA logs for errors
sudo journalctl -u sendense-hub -n 100 | grep -i error

# Check SBC telemetry initialization
ssh sna "grep 'SHA telemetry' /var/log/sendense-backup-client.log | tail -20"
```

**Common Causes:**
- SHA not restarted after deploy
- Tunnel not working (check ssh tunnel health)
- SBC using wrong URL (should be http://localhost:8082)

---

### Issue: bytes_transferred still zero

**Symptoms:**
- Completed backups show `bytes_transferred` = 0
- Machine modal shows incorrect sizes

**Diagnosis:**
```bash
# Check if CompleteBackup aggregation is running
sudo journalctl -u sendense-hub | grep -A 5 "Aggregated bytes_transferred"

# Manually verify aggregation logic
mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    bd.backup_job_id,
    SUM(bd.bytes_transferred) AS should_be,
    bj.bytes_transferred AS actual,
    (SUM(bd.bytes_transferred) = bj.bytes_transferred) AS correct
  FROM backup_disks bd
  JOIN backup_jobs bj ON bj.id = bd.backup_job_id
  WHERE bj.status = 'completed'
  GROUP BY bd.backup_job_id
  HAVING correct = 0;" --vertical
```

**Fix:**
- Verify SHA binary is v2.26.0-telemetry
- Check CompleteBackup code has aggregation logic (line ~622)
- May need to manually fix old jobs:
```sql
UPDATE backup_jobs bj
SET bytes_transferred = (
  SELECT SUM(IFNULL(bytes_transferred, 0))
  FROM backup_disks bd
  WHERE bd.backup_job_id = bj.id
)
WHERE bj.status = 'completed' AND bj.bytes_transferred = 0;
```

---

### Issue: Stale detector not working

**Symptoms:**
- Jobs stuck in "running" status forever
- No automatic "stalled"/"failed" transitions

**Diagnosis:**
```bash
# Check stale detector started
sudo journalctl -u sendense-hub --since boot | grep "Stale job detector"

# Expected: "🚨 Stale job detector started"

# Check detector loop running
sudo journalctl -u sendense-hub --since="5 minutes ago" | grep -i stale

# Manually test stale detection logic
mysql -u oma_user -poma_password migratekit_oma -e "
  SELECT 
    id,
    status,
    last_telemetry_at,
    TIMESTAMPDIFF(SECOND, last_telemetry_at, NOW()) AS seconds_stale,
    CASE
      WHEN TIMESTAMPDIFF(SECOND, last_telemetry_at, NOW()) > 300 THEN 'should_be_failed'
      WHEN TIMESTAMPDIFF(SECOND, last_telemetry_at, NOW()) > 60 THEN 'should_be_stalled'
      ELSE 'OK'
    END AS detection_status
  FROM backup_jobs 
  WHERE status = 'running'
  AND last_telemetry_at IS NOT NULL;"
```

**Fix:**
- Restart SHA service
- Check for goroutine panics in logs
- Verify stale detector code deployed

---

## Success Criteria Summary

| Test | Pass Criteria | Status |
|------|---------------|--------|
| 1. bytes_transferred Fix | `bytes_transferred` > 0 for completed jobs | ⏳ |
| 2. Real-Time Telemetry | Updates every 5s, progress increases smoothly | ⏳ |
| 3. Per-Disk Progress | All disks tracked independently | ⏳ |
| 4. Stale Detection | Jobs marked stalled (60s) and failed (5min) | ⏳ |
| 5. Backward Compatibility | Old SBC works without telemetry | ⏳ |

---

## Next Steps After Testing

1. **If all tests pass:**
   - Mark Phase 9 as complete ✅
   - Update CHANGELOG.md with test results
   - Proceed to Phase 10 (final documentation)
   - Deploy to production

2. **If tests fail:**
   - Document failures in detail
   - Create bug fixes
   - Re-test after fixes
   - Do NOT deploy to production

3. **Production Rollout:**
   - Deploy SHA binary to production
   - Deploy SBC binary to all SNAs
   - Monitor telemetry coverage (aim for 100%)
   - Remove old SNA progress poller (final cleanup)

---

**End of Testing Guide**

