# Telemetry Data Collection Fix - Verification Guide
## Date: 2025-10-10
## Binary: migratekit-telemetry-fix

---

## 🎯 What Was Fixed

### Problem
- Telemetry infrastructure (API, service, database) worked perfectly
- BUT: SBC was sending empty/zeroed telemetry data
- Result: `bytes_transferred`, `progress_percent`, `transfer_speed_bps`, `current_phase` all remained `0`

### Root Cause
1. **Incomplete TelemetryUpdate construction** in `tracker.go`:
   - `ProgressPercent` was hardcoded to `0.0`
   - Missing fields: `JobID`, `JobType`, `CurrentPhase`, `TotalBytes`, `Timestamp`
   
2. **Missing parameters** in `progress_aggregator.go`:
   - Didn't pass `totalBytes` to tracker
   - Didn't pass `currentPhase` to tracker

### Fix Applied
1. **`tracker.go` - UpdateProgress() method:**
   ```go
   // OLD: Only 3 parameters, hardcoded 0 progress
   func UpdateProgress(ctx, bytesTransferred, transferSpeedBps, etaSeconds)
   
   // NEW: 5 parameters, calculates progress
   func UpdateProgress(ctx, bytesTransferred, totalBytes, transferSpeedBps, etaSeconds, currentPhase)
   
   // Now calculates:
   progressPercent = (bytesTransferred / totalBytes) * 100.0
   
   // And includes all required fields:
   JobID, JobType, CurrentPhase, TotalBytes, Timestamp, Disks[]
   ```

2. **`progress_aggregator.go` - maybeUpdateVMA():**
   ```go
   // OLD: Missing parameters
   pa.telemetryTracker.UpdateProgress(ctx, currentBytes, throughputBPS, etaSeconds)
   
   // NEW: Complete parameters
   pa.telemetryTracker.UpdateProgress(ctx, currentBytes, pa.totalBytes, throughputBPS, etaSeconds, "transferring")
   ```

3. **Enhanced logging:**
   - Changed from `Warn` to `Error` for failed telemetry sends
   - Added INFO-level logs for successful sends (more visible)
   - Added detailed field logging before each send

---

## ✅ Verification Steps

### Step 1: Start a Backup
```bash
# From GUI: Click "Run Now" on any protection flow
# Or use API:
curl -X POST http://localhost:8082/api/v1/protection-flows/{flow_id}/execute
```

### Step 2: Monitor SHA Logs (Real-time Telemetry Arrival)
```bash
# Watch for telemetry POST requests:
sudo journalctl -u sendense-hub -f | grep -E "(POST|telemetry|✅)"

# Expected output:
# ✅ Telemetry update processed
# job_id=backup-xxx
# bytes=12345678
# progress_percent=45.2
# speed_bps=104857600
```

### Step 3: Monitor Database Updates (Every 5 Seconds)
```bash
# Watch backup_jobs table in real-time:
watch -n 2 'mysql -u oma_user -poma_password migratekit_oma -e "
SELECT 
  id,
  status,
  ROUND(bytes_transferred / (1024*1024*1024), 2) AS gb_transferred,
  ROUND(progress_percent, 2) AS percent,
  ROUND(transfer_speed_bps / (1024*1024), 2) AS speed_mbps,
  current_phase,
  last_telemetry_at
FROM backup_jobs 
WHERE status IN (\"running\", \"queued\")
ORDER BY created_at DESC 
LIMIT 1" --vertical'
```

**Expected behavior:**
- `gb_transferred` increases in real-time
- `percent` increases from 0 → 100
- `speed_mbps` shows actual transfer rate
- `current_phase` shows "transferring"
- `last_telemetry_at` updates every ~5 seconds

### Step 4: Check SBC Logs on SNA
```bash
# SSH to SNA and check migratekit logs:
sshpass -p "Password1" ssh vma@10.0.100.231 "sudo journalctl -n 200 --no-pager | grep -E '(🚀|telemetry|SHA)'"

# Expected output:
# 🚀 Sending telemetry update to SHA
# job_id=backup-xxx
# bytes=12345678
# total=50000000
# percent=24.69
# speed_bps=104857600
# phase=transferring
```

### Step 5: Verify Per-Disk Progress
```bash
# Check backup_disks table:
mysql -u oma_user -poma_password migratekit_oma -e "
SELECT 
  backup_job_id,
  disk_index,
  status,
  ROUND(bytes_transferred / (1024*1024), 0) AS mb_transferred,
  ROUND(progress_percent, 2) AS percent,
  qcow2_path
FROM backup_disks 
WHERE backup_job_id = 'backup-xxx-xxx'
ORDER BY disk_index;" --vertical
```

**Expected:**
- Each disk shows real `mb_transferred` values
- `percent` shows real progress per disk

### Step 6: Verify Completion
```bash
# After backup completes, check final state:
mysql -u oma_user -poma_password migratekit_oma -e "
SELECT 
  id,
  status,
  ROUND(bytes_transferred / (1024*1024*1024), 2) AS gb_transferred,
  progress_percent,
  completed_at,
  error_message
FROM backup_jobs 
WHERE id = 'backup-xxx-xxx';" --vertical
```

**Expected:**
- `status` = "completed"
- `gb_transferred` > 0 (actual backup size)
- `progress_percent` = 100.00
- `completed_at` populated

---

## 🔍 Troubleshooting

### If telemetry still not working:

#### Problem: No POST requests in SHA logs
**Possible causes:**
1. SBC not using fixed binary
2. Wrong SHA_API_URL in SBC environment
3. Network/tunnel issue

**Debug:**
```bash
# Check SBC binary on SNA:
sshpass -p "Password1" ssh vma@10.0.100.231 "md5sum /home/vma/migratekit"
# Should match: 4f66a071a846efe9bddbe3f71b132dcd

# Check SBC process environment:
sshpass -p "Password1" ssh vma@10.0.100.231 "ps aux | grep migratekit | grep -v grep"
```

#### Problem: POST requests arrive but data still 0
**Possible causes:**
1. Old binary still running (not restarted)
2. Cache/old code issue

**Debug:**
```bash
# Check SBC logs for new INFO-level logs:
sshpass -p "Password1" ssh vma@10.0.100.231 "sudo journalctl -n 500 | grep '🚀 Sending telemetry'"

# Should see detailed field logs with actual values
```

#### Problem: Data in logs but not in database
**Possible causes:**
1. SHA TelemetryService error
2. Database permission issue

**Debug:**
```bash
# Check SHA logs for service errors:
sudo journalctl -u sendense-hub -n 200 | grep -i error

# Check telemetry processing:
sudo journalctl -u sendense-hub -n 200 | grep "process telemetry"
```

---

## 📊 Success Criteria

✅ **All of these must be true:**

1. SHA logs show `POST /api/v1/telemetry/backup/...` requests every ~5 seconds
2. SHA logs show "✅ Telemetry update processed" with actual values (not 0)
3. Database `backup_jobs.bytes_transferred` increases in real-time
4. Database `backup_jobs.progress_percent` increases from 0 → 100
5. Database `backup_jobs.transfer_speed_bps` shows actual speed (not 0)
6. Database `backup_jobs.current_phase` shows "transferring"
7. Database `backup_jobs.last_telemetry_at` updates every ~5 seconds
8. SBC logs show "🚀 Sending telemetry update to SHA" with actual values
9. On completion: `bytes_transferred` > 0, `progress_percent` = 100
10. Per-disk progress populates in `backup_disks` table

---

## 🎉 Expected Outcome

After this fix, the telemetry framework is **fully operational**:
- Real-time progress tracking works
- GUI can display live backup progress
- Machine modal will show accurate backup sizes
- Charts can be built from rich telemetry data
- Stale job detection works properly

This unblocks the Machine Backup Details Modal feature!

