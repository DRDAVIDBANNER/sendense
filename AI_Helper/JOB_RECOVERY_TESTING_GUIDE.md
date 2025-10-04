# Job Recovery Enhancement Testing Guide

**Version**: oma-api-v2.30.0-job-recovery-enhancement  
**Created**: October 3, 2025  
**Status**: Ready for Testing  

---

## 📋 **BUILD INFORMATION**

**Binary Locations**:
- **Builds Archive**: `/home/pgrayson/migratekit-cloudstack/source/builds/oma-api-v2.30.0-job-recovery-enhancement`
- **Deployment Directory**: `/opt/migratekit/bin/oma-api-v2.30.0-job-recovery-enhancement`
- **Current Symlink**: `/opt/migratekit/bin/oma-api` → `oma-api-v2.40.0-dynamic-oma-vm-id-fix`

**Binary Size**: 32M  
**Build Date**: October 3, 2025 14:09  

---

## 🚨 **PRE-DEPLOYMENT CHECKLIST**

Before activating the new binary:

- [ ] **Backup Current Binary**
  ```bash
  sudo cp /opt/migratekit/bin/oma-api /opt/migratekit/bin/oma-api.backup-$(date +%Y%m%d-%H%M%S)
  ```

- [ ] **Check Running Jobs**
  ```bash
  mysql -u oma_user -poma_password migratekit_oma -e \
    "SELECT id, source_vm_name, status, progress_percent, updated_at 
     FROM replication_jobs 
     WHERE status IN ('replicating', 'initializing') 
     ORDER BY updated_at DESC LIMIT 10;"
  ```

- [ ] **Check VMA Connectivity**
  ```bash
  curl -s http://localhost:9081/api/v1/health || echo "VMA unreachable"
  ```

- [ ] **Verify Database Connection**
  ```bash
  mysql -u oma_user -poma_password -e "SELECT 1" migratekit_oma
  ```

---

## 🧪 **TEST SCENARIOS**

### **Scenario 1: Normal Startup with No Active Jobs** (Baseline)

**Purpose**: Verify recovery doesn't break normal startup

**Steps**:
1. Stop OMA API:
   ```bash
   sudo systemctl stop oma-api
   ```

2. Verify no active jobs in database:
   ```bash
   mysql -u oma_user -poma_password migratekit_oma -e \
     "SELECT COUNT(*) as active_jobs FROM replication_jobs WHERE status = 'replicating';"
   ```

3. Update symlink to new binary:
   ```bash
   sudo ln -sf /opt/migratekit/bin/oma-api-v2.30.0-job-recovery-enhancement /opt/migratekit/bin/oma-api
   ```

4. Start OMA API:
   ```bash
   sudo systemctl start oma-api
   ```

5. Check startup logs:
   ```bash
   sudo journalctl -u oma-api --since "1 minute ago" | grep -E "job recovery|Starting intelligent|No active jobs"
   ```

**Expected Result**:
```
✅ Starting intelligent job recovery with VMA validation on OMA startup
✅ No active jobs found - system is clean
✅ Job recovery completed successfully
✅ OMA API server started successfully
```

**Success Criteria**:
- [ ] Service starts without errors
- [ ] Logs show "No active jobs found"
- [ ] API responds to health check: `curl http://localhost:8082/health`

---

### **Scenario 2: Job Still Running on VMA** (CRITICAL TEST)

**Purpose**: Verify polling automatically restarts for jobs running on VMA

**Steps**:
1. Start a replication job (use GUI or API):
   ```bash
   # Via API - adjust VM details as needed
   curl -X POST http://localhost:8082/api/v1/replications \
     -H "Content-Type: application/json" \
     -d '{
       "source_vm": {...},
       "ossea_config_id": 1,
       "start_replication": true
     }'
   ```

2. Wait for job to reach ~30% progress:
   ```bash
   # Monitor until progress > 20%
   watch -n 2 'mysql -u oma_user -poma_password migratekit_oma -e \
     "SELECT id, progress_percent, status FROM replication_jobs ORDER BY created_at DESC LIMIT 1;"'
   ```

3. Note the job ID and current progress

4. Stop OMA API:
   ```bash
   sudo systemctl stop oma-api
   ```

5. Verify job is still running on VMA:
   ```bash
   curl -s http://localhost:9081/api/v1/progress/{JOB_ID} | jq '.phase, .percentage'
   ```

6. Start OMA API:
   ```bash
   sudo systemctl start oma-api
   ```

7. Watch recovery logs:
   ```bash
   sudo journalctl -u oma-api --since "1 minute ago" -f | grep -E "job recovery|VMA status|restarting polling"
   ```

8. Verify polling restarted:
   ```bash
   # Check database updates resume
   watch -n 2 'mysql -u oma_user -poma_password migratekit_oma -e \
     "SELECT id, progress_percent, vma_last_poll_at, updated_at FROM replication_jobs WHERE id='\''JOB_ID'\'';"'
   ```

**Expected Logs**:
```
🔍 Found 1 active jobs requiring recovery validation
🔄 Processing job: {JOB_ID} ({VM_NAME}) - stagnant: X.X min, total age: X.X min
🔍 Checking VMA status for job {JOB_ID}
🔗 Found N NBD export names for job {JOB_ID}, trying progress API
✅ Got VMA response via NBD export name migration-vol-{UUID}
🎯 Recovery decision for job {JOB_ID}: VMA status=running, stagnant=X.X min
✅ Job {JOB_ID} still running on VMA (XX.X%) - restarting polling
🚀 Successfully restarted VMA progress polling for job {JOB_ID}
✅ Job recovery completed:
    Total processed: 1
    Still running (polling restarted): 1
```

**Success Criteria**:
- [ ] Job found during recovery scan
- [ ] VMA queried successfully
- [ ] Status detected as "running"
- [ ] Polling restarted automatically
- [ ] `vma_last_poll_at` updates every 5 seconds
- [ ] Progress continues from where it left off
- [ ] Job completes successfully

---

### **Scenario 3: Job Completed During OMA Downtime** (CRITICAL TEST)

**Purpose**: Verify jobs completed during downtime are properly finalized

**Steps**:
1. Start a very small VM replication (should complete in 2-3 minutes)

2. Immediately stop OMA API:
   ```bash
   sudo systemctl stop oma-api
   ```

3. Wait for job to complete on VMA:
   ```bash
   # Check VMA directly
   curl -s http://localhost:9081/api/v1/progress/{JOB_ID} | jq '.phase, .percentage, .status'
   ```

4. When VMA shows "completed", start OMA API:
   ```bash
   sudo systemctl start oma-api
   ```

5. Watch recovery logs:
   ```bash
   sudo journalctl -u oma-api --since "1 minute ago" | grep -E "job recovery|completed on VMA|marking as completed"
   ```

6. Check database:
   ```bash
   mysql -u oma_user -poma_password migratekit_oma -e \
     "SELECT id, status, progress_percent, current_operation, completed_at 
      FROM replication_jobs WHERE id='{JOB_ID}';"
   ```

**Expected Logs**:
```
🔍 Found 1 active jobs requiring recovery validation
🔄 Processing job: {JOB_ID}
✅ Got VMA response via NBD export name
🎯 Recovery decision: VMA status=completed
✅ Job {JOB_ID} completed on VMA (100.0%) - finalizing
✅ Job {JOB_ID} marked as completed
✅ Job recovery completed:
    Completed: 1
```

**Success Criteria**:
- [ ] Job detected as completed
- [ ] Status updated to "completed"
- [ ] progress_percent = 100.0
- [ ] current_operation = "Completed"
- [ ] completed_at IS NOT NULL
- [ ] VM context updated to "ready_for_failover"
- [ ] No polling started (job already done)

---

### **Scenario 4: VMA Unreachable During Recovery**

**Purpose**: Verify graceful handling when VMA is down

**Steps**:
1. Start a replication job

2. Stop VMA API:
   ```bash
   ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl stop vma-api'
   ```

3. Stop OMA API:
   ```bash
   sudo systemctl stop oma-api
   ```

4. Start OMA API (VMA still down):
   ```bash
   sudo systemctl start oma-api
   ```

5. Watch recovery logs:
   ```bash
   sudo journalctl -u oma-api --since "1 minute ago" | grep -E "job recovery|VMA unreachable|leaving for retry"
   ```

**For Recent Jobs** (< 30 minutes old):
```
Expected: Job left in "replicating" status for health monitor to retry later
❌ VMA unreachable for job {JOB_ID}
⏳ Job {JOB_ID} - VMA unreachable but job is recent - leaving for retry
```

**For Old Jobs** (> 30 minutes old):
```
Expected: Job marked as failed
❌ VMA unreachable for job {JOB_ID}
❌ Job {JOB_ID} - VMA unreachable and job is old - marking as failed
```

6. Restart VMA:
   ```bash
   ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl start vma-api'
   ```

**Success Criteria**:
- [ ] Recent jobs NOT marked as failed
- [ ] Old jobs marked as failed with appropriate error
- [ ] System recovers when VMA comes back online

---

### **Scenario 5: Job Failed on VMA**

**Purpose**: Verify VMA-reported failures are properly handled

**Steps**:
1. Start a replication job that will fail (e.g., invalid NBD connection)

2. Wait for VMA to report failure:
   ```bash
   curl -s http://localhost:9081/api/v1/progress/{JOB_ID} | jq '.status, .errors'
   ```

3. Stop and restart OMA API:
   ```bash
   sudo systemctl restart oma-api
   ```

4. Check recovery logs:
   ```bash
   sudo journalctl -u oma-api --since "1 minute ago" | grep -E "VMA reports job failed|marking as failed"
   ```

**Expected Logs**:
```
❌ VMA reports job failed: {ERROR_MESSAGE}
🎯 Recovery decision: VMA status=failed
❌ Job {JOB_ID} failed on VMA - marking as failed with VMA error
```

**Success Criteria**:
- [ ] Job detected as failed
- [ ] Error message from VMA stored in database
- [ ] vma_error_classification = "vma_reported_failure"
- [ ] VM context updated to allow new operations

---

### **Scenario 6: Job Not Found on VMA**

**Purpose**: Verify intelligent handling when job disappears from VMA

**Test 6A: High Progress Job (>90%)**
1. Find or create a job that reached >90% but is now missing from VMA
2. Restart OMA API
3. Expect: Marked as "completed" (likely finished and cleaned up)

**Test 6B: Low Progress Job (<90%)**
1. Find or create a job that only reached <90% and is now missing from VMA
2. Restart OMA API
3. Expect: Marked as "failed" with "job lost" classification

**Verification**:
```bash
sudo journalctl -u oma-api --since "1 minute ago" | grep -E "not found on VMA|assuming completed|marking as lost"
```

---

## 🔍 **MONITORING & VERIFICATION**

### **Real-time Recovery Monitoring**

```bash
# Watch recovery in real-time during startup
sudo journalctl -u oma-api -f | grep --line-buffered -E \
  "job recovery|VMA status|Recovery decision|restarting polling|marked as"
```

### **Database Verification Queries**

```sql
-- Check all jobs and their latest poll times
SELECT 
    id, 
    source_vm_name, 
    status, 
    progress_percent,
    vma_last_poll_at,
    TIMESTAMPDIFF(SECOND, vma_last_poll_at, NOW()) as seconds_since_poll,
    updated_at
FROM replication_jobs 
WHERE status IN ('replicating', 'initializing')
ORDER BY updated_at DESC;

-- Check VM contexts updated correctly
SELECT 
    context_id,
    vm_name,
    current_status,
    successful_jobs,
    failed_jobs,
    last_replication_at
FROM vm_replication_contexts
WHERE current_status IN ('ready_for_failover', 'replicating')
ORDER BY updated_at DESC;
```

### **VMA Polling Status Check**

Once OMA is running, you can check if polling is active:
```bash
# This will be available after we add the API endpoint (Phase 4)
# For now, check logs for "Successfully restarted VMA progress polling"
```

---

## 🚀 **DEPLOYMENT PROCEDURE**

### **Step 1: Deploy to Development (This System)**

```bash
# 1. Backup current binary
sudo cp /opt/migratekit/bin/oma-api /opt/migratekit/bin/oma-api.backup-$(date +%Y%m%d-%H%M%S)

# 2. Update symlink
sudo ln -sf /opt/migratekit/bin/oma-api-v2.30.0-job-recovery-enhancement /opt/migratekit/bin/oma-api

# 3. Restart service
sudo systemctl restart oma-api

# 4. Watch startup (Ctrl+C to exit)
sudo journalctl -u oma-api -f
```

**Expected Startup Sequence**:
```
Starting OMA Migration API server
Database connection established
🚀 VMA progress poller started successfully
🚀 Starting scheduler service
✅ Scheduler service started
🔍 Initializing intelligent job recovery system with VMA validation
🚀 Running intelligent job recovery scan with VMA validation...
[Recovery messages based on current state]
✅ Job recovery completed successfully
OMA API server started successfully
```

### **Step 2: Verify Service Health**

```bash
# Check service status
sudo systemctl status oma-api

# Check API health
curl http://localhost:8082/health

# Check logs for errors
sudo journalctl -u oma-api --since "5 minutes ago" | grep -i error
```

### **Step 3: Monitor for Issues**

```bash
# Watch for 5 minutes
sudo journalctl -u oma-api -f

# Look for:
# - Any error messages
# - Failed recovery attempts
# - VMA connection issues
# - Database errors
```

---

## 🔄 **ROLLBACK PROCEDURE**

If issues arise:

```bash
# 1. Stop service
sudo systemctl stop oma-api

# 2. Revert to previous binary
sudo ln -sf /opt/migratekit/bin/oma-api-v2.40.0-dynamic-oma-vm-id-fix /opt/migratekit/bin/oma-api

# 3. Restart service
sudo systemctl start oma-api

# 4. Verify
sudo systemctl status oma-api
curl http://localhost:8082/health
```

---

## 📊 **SUCCESS METRICS**

After deployment, monitor for 1 hour and verify:

- [ ] **Service Stability**: No crashes or restarts
- [ ] **Recovery Success**: Jobs properly recovered based on VMA status
- [ ] **Polling Active**: `vma_last_poll_at` updating every 5 seconds for active jobs
- [ ] **No False Failures**: No running jobs incorrectly marked as failed
- [ ] **Completion Detection**: Completed jobs properly finalized
- [ ] **Performance**: No degradation in API response times
- [ ] **Memory**: No memory leaks (check with `systemctl status oma-api`)

---

## 🐛 **TROUBLESHOOTING**

### **Issue: VMA Client Not Initialized**

**Symptom**: Logs show "VMA services not available - job recovery will be limited"

**Solution**:
```bash
# Check VMA environment variable
grep VMA_API_URL /etc/systemd/system/oma-api.service

# Should be: http://localhost:9081 (via SSH tunnel reverse proxy)
# If missing, add to service file and restart
```

### **Issue: Jobs Not Being Recovered**

**Symptom**: Jobs stuck in "replicating" after restart

**Debug**:
```bash
# Check recovery ran
sudo journalctl -u oma-api --since "5 minutes ago" | grep "job recovery"

# Check VMA reachability
curl http://localhost:9081/api/v1/health

# Check job details
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT id, status, progress_percent, vma_last_poll_at FROM replication_jobs WHERE status='replicating';"
```

### **Issue: Polling Not Restarting**

**Symptom**: Logs show job found but polling doesn't restart

**Debug**:
```bash
# Check for "Failed to restart polling" messages
sudo journalctl -u oma-api | grep "Failed to restart polling"

# Check VMA progress poller status
# (Will need API endpoint from Phase 4 - for now check logs)
```

---

## 📈 **PERFORMANCE BASELINE**

Track these metrics before and after deployment:

```bash
# API response time
time curl -s http://localhost:8082/health

# Memory usage
sudo systemctl status oma-api | grep Memory

# Active polling jobs
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT COUNT(*) FROM replication_jobs WHERE status='replicating';"
```

---

## ✅ **SIGN-OFF CHECKLIST**

Before considering deployment complete:

- [ ] All test scenarios passed
- [ ] Service stable for 1+ hours
- [ ] No errors in logs
- [ ] Active jobs properly tracked
- [ ] Job completion detected correctly
- [ ] VMA polling working
- [ ] Performance metrics acceptable
- [ ] Rollback procedure tested and works

---

## 📞 **NEXT STEPS AFTER VALIDATION**

Once this deployment is validated:

1. **Deploy to QC Environment** (45.130.45.65)
2. **Deploy to Production OMA** (10.245.246.121)
3. **Proceed to Phase 2**: Fix HTTP 200 error detection in poller
4. **Proceed to Phase 2**: Add health monitor for continuous monitoring

---

**Status**: Ready for Testing  
**Risk Level**: Medium - Changes core recovery logic  
**Rollback Time**: < 2 minutes  
**Testing Time Required**: 1-2 hours


