# VMA Progress Polling & Job Recovery Enhancement Job Sheet

**Created**: October 3, 2025  
**Status**: 🔴 **NOT STARTED**  
**Priority**: 🚨 **CRITICAL** - Production Reliability Issue  
**Estimated Effort**: 2-3 days  

---

## 📋 **EXECUTIVE SUMMARY**

**Problem**: When OMA API restarts, jobs remain stuck in "replicating" status indefinitely. The VMA progress poller loses all state (activeJobs map) and never resumes polling, causing complete loss of job tracking for active migrations.

**Impact**: 
- Jobs that are actively running on VMA appear stuck forever
- Operators must manually intervene to recover job state
- No visibility into actual job progress after OMA restart
- Production reliability severely compromised

**Solution**: Implement smart job recovery that validates VMA status and restarts polling for active jobs.

---

## 🚨 **PROBLEM STATEMENT**

### **Current Broken Flow**

```
OMA API Restart
   ↓
JobRecovery.RecoverOrphanedJobsOnStartup() runs
   ↓
Finds jobs in "replicating" status > 30 minutes old
   ↓
❌ Marks ALL as "failed" without checking VMA
   ↓
VMAProgressPoller starts with EMPTY activeJobs map
   ↓
❌ No polling happens for ANY jobs
   ↓
RESULT: Jobs stuck forever OR falsely marked as failed
```

### **Root Causes**

1. **In-Memory State Loss**: `VMAProgressPoller.activeJobs` is a map that's lost on restart
2. **No VMA Validation**: Job recovery marks jobs as failed without checking if they're still running
3. **No Polling Restart**: Recovery system doesn't notify poller to resume tracking
4. **Insufficient Status Detection**: Only checks "replicating" status, misses other active states
5. **HTTP 200 Bug**: VMA returns HTTP 200 with "job not found" text instead of proper 404

---

## 🎯 **OBJECTIVES**

- [ ] **O1**: Restore polling for jobs that are still running on VMA after OMA restart
- [ ] **O2**: Detect and properly finalize jobs that completed during OMA downtime
- [ ] **O3**: Prevent false failures by validating VMA status before marking jobs as failed
- [ ] **O4**: Implement health monitoring to detect polling drift over time
- [ ] **O5**: Fix VMA API error detection for proper failure handling

---

## 📊 **TASK BREAKDOWN**

### **Phase 1: Core Recovery System Enhancement** 🚨 **CRITICAL**

#### **Task 1.1: Add VMA Status Validation to Job Recovery**
- [ ] **1.1.1** Add `VMAProgressClient` to `ProductionJobRecovery` struct
  - File: `source/current/oma/services/job_recovery_production.go`
  - Pass client in constructor: `NewProductionJobRecovery(db, vmaClient)`
  
- [ ] **1.1.2** Implement `checkVMAStatus()` method
  ```go
  func (pjr *ProductionJobRecovery) checkVMAStatus(jobID string) (*VMAStatusResult, error)
  ```
  - Try NBD export name method first (query DB for export names)
  - Fallback to job_id method
  - Return structured status: running/completed/failed/not_found
  
- [ ] **1.1.3** Create helper to query NBD export names from database
  ```go
  func (pjr *ProductionJobRecovery) getNBDExportNamesForJob(jobID string) ([]string, error)
  ```
  - Query: `replication_jobs → vm_disks → ossea_volumes → volume_id`
  - Construct: `migration-vol-{volume_id}`

- [ ] **1.1.4** Add `VMAStatusResult` struct
  ```go
  type VMAStatusResult struct {
      Status       string  // running, completed, failed, not_found
      Percentage   float64
      ErrorMessage string
      IsReachable  bool
  }
  ```

**Testing Criteria**:
- [ ] Can detect running jobs on VMA
- [ ] Can detect completed jobs on VMA
- [ ] Can detect failed jobs on VMA
- [ ] Handles VMA unreachable gracefully
- [ ] Uses NBD export names as primary method

---

#### **Task 1.2: Enhance Recovery Logic with Smart Decision Making**
- [ ] **1.2.1** Modify `RecoverOrphanedJobsOnStartup()` to query VMA before marking as failed
  - File: `source/current/oma/services/job_recovery_production.go:29`
  - Replace blanket "mark as failed" with smart validation
  
- [ ] **1.2.2** Implement recovery decision tree:
  ```
  For each active job:
    ├─ Query VMA status
    │
    ├─ If VMA unreachable:
    │  ├─ Job < 30 min old: Continue polling (VMA might be starting)
    │  └─ Job > 30 min old: Mark as failed (VMA likely down)
    │
    ├─ If VMA responds "running":
    │  └─ Restart polling (Task 1.3)
    │
    ├─ If VMA responds "completed":
    │  └─ Update job to completed with final progress
    │
    ├─ If VMA responds "failed":
    │  └─ Mark as failed with VMA error message
    │
    └─ If VMA responds "not_found":
       ├─ Progress > 90%: Likely completed, mark as completed
       └─ Progress < 90%: Likely lost, mark as failed
  ```

- [ ] **1.2.3** Update `recoverOrphanedJob()` signature
  ```go
  func (pjr *ProductionJobRecovery) recoverOrphanedJob(
      job *database.ReplicationJob,
      vmaStatus *VMAStatusResult,
  ) error
  ```

- [ ] **1.2.4** Add new recovery methods:
  ```go
  func (pjr *ProductionJobRecovery) markAsCompleted(job *database.ReplicationJob, vmaStatus *VMAStatusResult) error
  func (pjr *ProductionJobRecovery) updateWithVMAProgress(job *database.ReplicationJob, vmaStatus *VMAStatusResult) error
  ```

**Testing Criteria**:
- [ ] Jobs running on VMA are not marked as failed
- [ ] Jobs completed on VMA are properly finalized
- [ ] Jobs with VMA errors get proper error messages
- [ ] Old jobs with unreachable VMA are marked as failed
- [ ] Recent jobs with unreachable VMA are left alone

---

#### **Task 1.3: Integrate Polling Restart into Recovery**
- [ ] **1.3.1** Pass `VMAProgressPoller` reference to job recovery
  - File: `source/current/oma/cmd/main.go:104`
  - Modify constructor: `NewProductionJobRecovery(db, vmaClient, vmaProgressPoller)`
  
- [ ] **1.3.2** Add `VMAProgressPoller` field to `ProductionJobRecovery`
  ```go
  type ProductionJobRecovery struct {
      db                database.Connection
      vmaClient         *services.VMAProgressClient
      vmaProgressPoller *services.VMAProgressPoller  // NEW
      maxJobAge         time.Duration
      recoveryEnabled   bool
  }
  ```

- [ ] **1.3.3** Implement polling restart for active jobs
  ```go
  func (pjr *ProductionJobRecovery) restartPollingForJob(jobID string) error {
      if err := pjr.vmaProgressPoller.StartPolling(jobID); err != nil {
          return fmt.Errorf("failed to restart polling: %w", err)
      }
      log.WithField("job_id", jobID).Info("✅ Restarted VMA progress polling")
      return nil
  }
  ```

- [ ] **1.3.4** Call `restartPollingForJob()` for running jobs in recovery loop

- [ ] **1.3.5** Update `cmd/main.go` to wire everything together:
  ```go
  // Create VMA client
  vmaClient := services.NewVMAProgressClient(vmaAPIURL)
  
  // VMA poller is created in api.NewServer - need to extract it
  // Pass to job recovery
  jobRecovery := services.NewProductionJobRecovery(db, vmaClient, vmaProgressPoller)
  ```

**Testing Criteria**:
- [ ] Polling automatically restarts for running jobs after OMA restart
- [ ] activeJobs map repopulates correctly
- [ ] Progress updates resume within 5 seconds
- [ ] No duplicate polling for same job

---

#### **Task 1.4: Expand Active Job Detection**
- [ ] **1.4.1** Update query to find ALL active states, not just "replicating"
  - File: `source/current/oma/services/job_recovery_production.go:40`
  - Current: `WHERE status = 'replicating'`
  - New: `WHERE status IN ('replicating', 'initializing', 'ready_for_sync', 'attaching', 'configuring')`

- [ ] **1.4.2** Create helper method:
  ```go
  func (pjr *ProductionJobRecovery) findAllActiveJobs(ctx context.Context) ([]database.ReplicationJob, error)
  ```

- [ ] **1.4.3** Add status validation for edge cases:
  - Jobs in "initializing" < 5 minutes: Let them initialize
  - Jobs in "ready_for_sync" < 2 minutes: Normal startup delay
  - Jobs in "attaching" < 5 minutes: Volume operations take time

**Testing Criteria**:
- [ ] Detects jobs stuck in "initializing"
- [ ] Detects jobs stuck in "ready_for_sync"
- [ ] Doesn't flag recently created jobs as orphaned
- [ ] Handles all active states correctly

---

### **Phase 2: VMA Progress Poller Enhancements** 🟡 **HIGH PRIORITY**

#### **Task 2.1: Add Polling State Query Methods**
- [ ] **2.1.1** Add `IsPolling()` public method
  - File: `source/current/oma/services/vma_progress_poller.go`
  ```go
  func (vpp *VMAProgressPoller) IsPolling(jobID string) bool {
      vpp.jobsMutex.RLock()
      defer vpp.jobsMutex.RUnlock()
      _, exists := vpp.activeJobs[jobID]
      return exists
  }
  ```

- [ ] **2.1.2** Add `GetActiveJobIDs()` method
  ```go
  func (vpp *VMAProgressPoller) GetActiveJobIDs() []string {
      vpp.jobsMutex.RLock()
      defer vpp.jobsMutex.RUnlock()
      
      jobIDs := make([]string, 0, len(vpp.activeJobs))
      for jobID := range vpp.activeJobs {
          jobIDs = append(jobIDs, jobID)
      }
      return jobIDs
  }
  ```

- [ ] **2.1.3** Enhance `GetPollingStatus()` with more details
  - Add: job age, consecutive errors, last successful poll time
  - Add: VMA API health status
  - Add: total polls attempted, success rate

**Testing Criteria**:
- [ ] Can query if specific job is being polled
- [ ] Can get list of all actively polled jobs
- [ ] Status includes comprehensive debugging info

---

#### **Task 2.2: Fix VMA API Error Detection**
- [ ] **2.2.1** Fix "job not found" detection bug
  - File: `source/current/oma/services/vma_progress_poller.go:292`
  - Current: Only checks HTTP 404
  - Issue: VMA returns HTTP 200 with "job not found" text
  
- [ ] **2.2.2** Update `handlePollingError()` to parse response body
  ```go
  func (vpp *VMAProgressPoller) handlePollingError(...) {
      // Check HTTP 404
      if vmaErr, ok := err.(*VMAProgressError); ok && vmaErr.StatusCode == 404 {
          // ... existing logic
      }
      
      // NEW: Check HTTP 200 with "job not found" in message
      if vmaErr, ok := err.(*VMAProgressError); ok && vmaErr.StatusCode == 200 {
          if strings.Contains(strings.ToLower(vmaErr.Message), "job not found") ||
             strings.Contains(strings.ToLower(vmaErr.Message), "not found") {
              // Treat as job completion/not found
              jobAge := time.Since(pollingCtx.StartedAt)
              if jobAge < pollingCtx.StartupGracePeriod {
                  logger.Debug("Job not found during startup grace period")
                  return
              }
              logger.Info("Job not found in VMA (HTTP 200) - likely completed")
              vpp.StopPolling(jobID)
              return
          }
      }
  }
  ```

- [ ] **2.2.3** Update `VMAProgressError` to include response body
  ```go
  type VMAProgressError struct {
      StatusCode   int
      Message      string
      ResponseBody string  // NEW: Full response for debugging
      JobID        string
  }
  ```

**Testing Criteria**:
- [ ] Detects HTTP 200 "job not found" responses
- [ ] Stops polling appropriately
- [ ] Respects grace period
- [ ] Logs clear debugging information

---

#### **Task 2.3: Implement Health Monitor for Polling Drift**
- [ ] **2.3.1** Create background health monitor goroutine
  ```go
  func (vpp *VMAProgressPoller) StartHealthMonitor(ctx context.Context, db database.Connection) {
      ticker := time.NewTicker(1 * time.Minute)
      defer ticker.Stop()
      
      for {
          select {
          case <-ctx.Done():
              return
          case <-ticker.C:
              vpp.checkForOrphanedJobs(ctx, db)
          }
      }
  }
  ```

- [ ] **2.3.2** Implement orphaned job detection
  ```go
  func (vpp *VMAProgressPoller) checkForOrphanedJobs(ctx context.Context, db database.Connection) {
      // Query database for jobs in "replicating" status
      var activeJobs []database.ReplicationJob
      db.GetGormDB().Where("status = ?", "replicating").Find(&activeJobs)
      
      // Check if each is being polled
      for _, job := range activeJobs {
          if !vpp.IsPolling(job.ID) {
              logger.Warn("Found job not being polled - restarting",
                  "job_id", job.ID,
                  "age_minutes", time.Since(job.UpdatedAt).Minutes())
              
              // Restart polling
              if err := vpp.StartPolling(job.ID); err != nil {
                  logger.Error("Failed to restart polling for orphaned job",
                      "job_id", job.ID,
                      "error", err)
              }
          }
      }
  }
  ```

- [ ] **2.3.3** Start health monitor from `cmd/main.go`
  ```go
  // After VMAProgressPoller.Start()
  go vmaProgressPoller.StartHealthMonitor(ctx, db)
  ```

- [ ] **2.3.4** Add metrics to track health monitor effectiveness
  - Counter: orphaned jobs detected
  - Counter: polling successfully restarted
  - Gauge: drift between DB and polling state

**Testing Criteria**:
- [ ] Detects jobs that should be polled but aren't
- [ ] Automatically restarts polling for orphaned jobs
- [ ] Runs every minute without performance impact
- [ ] Logs clear debugging information

---

### **Phase 3: Database Schema Enhancements** 🟢 **MEDIUM PRIORITY**

#### **Task 3.1: Add Job Recovery Metadata to replication_jobs**
- [ ] **3.1.1** Create migration: `20251003140000_add_job_recovery_fields.up.sql`
  ```sql
  ALTER TABLE replication_jobs ADD COLUMN recovery_attempted BOOLEAN DEFAULT FALSE;
  ALTER TABLE replication_jobs ADD COLUMN recovery_attempted_at TIMESTAMP NULL;
  ALTER TABLE replication_jobs ADD COLUMN recovery_method VARCHAR(50) NULL;
  ALTER TABLE replication_jobs ADD COLUMN vma_validated_at TIMESTAMP NULL;
  ALTER TABLE replication_jobs ADD COLUMN polling_restarted_at TIMESTAMP NULL;
  
  CREATE INDEX idx_replication_jobs_recovery ON replication_jobs(recovery_attempted, status);
  ```

- [ ] **3.1.2** Create down migration: `20251003140000_add_job_recovery_fields.down.sql`
  ```sql
  DROP INDEX idx_replication_jobs_recovery ON replication_jobs;
  ALTER TABLE replication_jobs DROP COLUMN polling_restarted_at;
  ALTER TABLE replication_jobs DROP COLUMN vma_validated_at;
  ALTER TABLE replication_jobs DROP COLUMN recovery_method;
  ALTER TABLE replication_jobs DROP COLUMN recovery_attempted_at;
  ALTER TABLE replication_jobs DROP COLUMN recovery_attempted;
  ```

- [ ] **3.1.3** Update `database.ReplicationJob` struct
  ```go
  type ReplicationJob struct {
      // ... existing fields
      
      // Recovery tracking
      RecoveryAttempted   bool       `json:"recovery_attempted" gorm:"default:false"`
      RecoveryAttemptedAt *time.Time `json:"recovery_attempted_at"`
      RecoveryMethod      string     `json:"recovery_method"`      // vma_validated, timeout, manual
      VMAValidatedAt      *time.Time `json:"vma_validated_at"`
      PollingRestartedAt  *time.Time `json:"polling_restarted_at"`
  }
  ```

- [ ] **3.1.4** Update recovery code to populate these fields

**Testing Criteria**:
- [ ] Migration runs successfully
- [ ] Fields populate correctly during recovery
- [ ] Can query recovery history
- [ ] Helps with debugging recovery issues

---

#### **Task 3.2: Optional - Persistent Polling State Table**
*Note: This is a nice-to-have enhancement for future resilience*

- [ ] **3.2.1** Create migration: `20251003150000_add_vma_polling_state.up.sql`
  ```sql
  CREATE TABLE vma_polling_state (
      job_id VARCHAR(64) PRIMARY KEY,
      started_polling_at TIMESTAMP NOT NULL,
      last_poll_at TIMESTAMP NOT NULL,
      consecutive_errors INT DEFAULT 0,
      last_error_message TEXT NULL,
      is_active BOOLEAN DEFAULT TRUE,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      
      FOREIGN KEY (job_id) REFERENCES replication_jobs(id) ON DELETE CASCADE,
      INDEX idx_active_polling (is_active, last_poll_at)
  );
  ```

- [ ] **3.2.2** Update VMAProgressPoller to persist state
  - Write to DB when polling starts
  - Update last_poll_at on each successful poll
  - Mark is_active=false when polling stops

- [ ] **3.2.3** Restore from DB on startup
  ```go
  func (vpp *VMAProgressPoller) RestoreFromDatabase(ctx context.Context, db database.Connection) error
  ```

**Testing Criteria**:
- [ ] Polling state persists across restarts
- [ ] Can recover exact polling status
- [ ] Performance impact is minimal
- [ ] Cleanup happens on job completion

---

### **Phase 4: API Endpoints for Observability** 🟢 **LOW PRIORITY**

#### **Task 4.1: Add Job Recovery Status Endpoint**
- [ ] **4.1.1** Create endpoint: `GET /api/v1/job-recovery/status`
  ```go
  func (h *JobRecoveryHandler) GetRecoveryStatus(w http.ResponseWriter, r *http.Request) {
      status := h.jobRecovery.GetRecoveryStatus()
      // Returns:
      // - Total jobs recovered
      // - Last recovery time
      // - Jobs currently in recovery
      // - Recovery success rate
  }
  ```

- [ ] **4.1.2** Create endpoint: `POST /api/v1/job-recovery/trigger`
  - Manually trigger recovery scan
  - Useful for debugging

- [ ] **4.1.3** Add to handler registration in `api/server.go`

**Testing Criteria**:
- [ ] Endpoint returns comprehensive recovery status
- [ ] Manual trigger works correctly
- [ ] Useful for operational debugging

---

#### **Task 4.2: Add Polling Status Endpoint**
- [ ] **4.2.1** Create endpoint: `GET /api/v1/vma-polling/status`
  ```go
  func (h *PollingHandler) GetPollingStatus(w http.ResponseWriter, r *http.Request) {
      status := h.vmaProgressPoller.GetPollingStatus()
      // Returns enhanced status with:
      // - Active job count
      // - Job details with ages, errors
      // - VMA health status
      // - Polling performance metrics
  }
  ```

- [ ] **4.2.2** Create endpoint: `GET /api/v1/vma-polling/jobs/{job_id}`
  - Get detailed status for specific job
  - Includes full polling history

- [ ] **4.2.3** Add to handler registration

**Testing Criteria**:
- [ ] Can query overall polling status
- [ ] Can get details for specific job
- [ ] Useful for debugging stuck jobs

---

### **Phase 5: Testing & Validation** 🎯 **CRITICAL**

#### **Task 5.1: Unit Tests**
- [ ] **5.1.1** Test `ProductionJobRecovery.checkVMAStatus()`
  - Mock VMA responses (running, completed, failed, not_found)
  - Test VMA unreachable scenario
  - Test NBD export name fallback

- [ ] **5.1.2** Test `ProductionJobRecovery.RecoverOrphanedJobsOnStartup()`
  - Test with running jobs on VMA
  - Test with completed jobs on VMA
  - Test with failed jobs on VMA
  - Test with VMA unreachable

- [ ] **5.1.3** Test `VMAProgressPoller` error detection
  - Test HTTP 404 handling
  - Test HTTP 200 "job not found" handling
  - Test grace period logic

- [ ] **5.1.4** Test health monitor
  - Test orphaned job detection
  - Test polling restart logic

**Testing Criteria**:
- [ ] All unit tests pass
- [ ] Code coverage > 80%
- [ ] Edge cases handled

---

#### **Task 5.2: Integration Tests**
- [ ] **5.2.1** Test OMA restart with running jobs
  - Start job on VMA
  - Restart OMA API
  - Verify polling resumes
  - Verify progress continues updating

- [ ] **5.2.2** Test OMA restart with completed jobs
  - Complete job on VMA during OMA downtime
  - Restart OMA API
  - Verify job marked as completed

- [ ] **5.2.3** Test VMA unreachable scenario
  - Start jobs
  - Stop VMA
  - Restart OMA
  - Verify appropriate error handling

- [ ] **5.2.4** Test orphaned job detection
  - Manually remove job from polling map
  - Wait for health monitor cycle
  - Verify polling restarts

**Testing Criteria**:
- [ ] Jobs survive OMA restart
- [ ] Progress tracking resumes automatically
- [ ] No false failures
- [ ] No stuck jobs

---

#### **Task 5.3: Production Testing Checklist**
- [ ] **5.3.1** Deploy to QC environment (45.130.45.65)
  - Test with real VMA at 10.0.100.232
  - Observe for 24 hours

- [ ] **5.3.2** Simulate OMA restart scenarios:
  ```bash
  # Scenario 1: Quick restart (< 1 minute)
  systemctl restart oma-api
  # Verify: Jobs continue normally
  
  # Scenario 2: Long downtime (> 5 minutes)
  systemctl stop oma-api
  sleep 300
  systemctl start oma-api
  # Verify: Jobs recover based on VMA status
  
  # Scenario 3: VMA unreachable
  systemctl restart oma-api  # with VMA down
  # Verify: Appropriate error handling
  ```

- [ ] **5.3.3** Monitor metrics:
  - Recovery success rate
  - False failure rate
  - Polling restart success rate
  - Time to recovery

**Testing Criteria**:
- [ ] Zero false failures
- [ ] 100% polling restart for active jobs
- [ ] < 10 second recovery time
- [ ] No manual intervention needed

---

### **Phase 6: Documentation & Deployment** 📚 **REQUIRED**

#### **Task 6.1: Update Documentation**
- [ ] **6.1.1** Update `PROJECT_STATUS.md`
  - Add "Job Recovery System" section
  - Document architecture changes

- [ ] **6.1.2** Update `RULES_AND_CONSTRAINTS.md`
  - Add job recovery rules
  - Document polling state management

- [ ] **6.1.3** Create operator guide: `docs/OPERATIONAL_RUNBOOK.md`
  - Troubleshooting stuck jobs
  - Manual recovery procedures
  - Using job recovery endpoints

- [ ] **6.1.4** Update API documentation
  - Document new endpoints
  - Update Swagger specs

**Testing Criteria**:
- [ ] Documentation is complete
- [ ] Examples are tested
- [ ] Operators can follow procedures

---

#### **Task 6.2: Build & Deploy**
- [ ] **6.2.1** Build new OMA API binary
  ```bash
  cd source/current/oma/cmd
  go build -o oma-api-v2.30.0-job-recovery-enhancement .
  ```

- [ ] **6.2.2** Deploy to QC environment
  ```bash
  scp oma-api-v2.30.0-job-recovery-enhancement qc-server:/opt/migratekit/bin/
  systemctl restart oma-api
  ```

- [ ] **6.2.3** Update symlink
  ```bash
  ln -sf oma-api-v2.30.0-job-recovery-enhancement /opt/migratekit/bin/oma-api
  ```

- [ ] **6.2.4** Monitor startup logs
  ```bash
  journalctl -u oma-api -f
  ```

- [ ] **6.2.5** Deploy to production (10.245.246.125) after QC validation

**Testing Criteria**:
- [ ] Deployment succeeds without errors
- [ ] Service starts cleanly
- [ ] Recovery runs on startup
- [ ] No regressions

---

## 🔍 **TESTING SCENARIOS**

### **Scenario 1: OMA Restart with Active Jobs**
```
Setup:
1. Start replication job for VM (job-20251003-140000-abc123)
2. Wait for progress to reach 30%
3. Restart OMA API

Expected Result:
✅ Job recovery validates with VMA
✅ Finds job still running
✅ Restarts polling automatically
✅ Progress updates continue from 30% → 100%
✅ Job completes successfully

Verify:
- SELECT * FROM replication_jobs WHERE id = 'job-20251003-140000-abc123';
  - recovery_attempted = TRUE
  - recovery_method = 'vma_validated'
  - polling_restarted_at IS NOT NULL
- Check logs for "Restarted VMA progress polling"
- Check logs for "Job still active on VMA"
```

### **Scenario 2: Job Completed During OMA Downtime**
```
Setup:
1. Start quick replication job (small VM, CBT enabled)
2. Immediately stop OMA API
3. Wait for job to complete on VMA
4. Start OMA API

Expected Result:
✅ Job recovery queries VMA
✅ Finds job completed
✅ Updates status to 'completed' with 100% progress
✅ Updates vm_replication_contexts
✅ No polling started (job already done)

Verify:
- Job status = 'completed'
- completed_at IS NOT NULL
- progress_percent = 100
- VM context status updated appropriately
```

### **Scenario 3: VMA Unreachable During Recovery**
```
Setup:
1. Start replication job
2. Stop VMA API
3. Restart OMA API

Expected Result:
If job < 30 minutes:
  ✅ Recovery logs warning
  ✅ Job remains in 'replicating'
  ✅ Will retry on next health monitor cycle

If job > 30 minutes:
  ✅ Recovery marks as failed
  ✅ Error message: "VMA unreachable after restart"
  ✅ VM context updated to allow new operations

Verify:
- Appropriate error logging
- Correct decision based on job age
```

### **Scenario 4: Health Monitor Detects Orphaned Job**
```
Setup:
1. Start replication job
2. Manually remove from polling map (simulated bug):
   - In debugger or via injected fault
3. Wait 1 minute for health monitor

Expected Result:
✅ Health monitor detects job not being polled
✅ Logs warning about orphaned job
✅ Automatically restarts polling
✅ Progress updates resume

Verify:
- Check logs for "Found job not being polled - restarting"
- Verify polling actually restarted
- Progress updates resume within 5 seconds
```

---

## 📈 **SUCCESS METRICS**

### **Critical Metrics**
- [ ] **Zero Stuck Jobs**: No jobs remain in "replicating" status > 1 hour after OMA restart
- [ ] **100% Recovery Rate**: All running jobs automatically resume polling after OMA restart
- [ ] **Zero False Failures**: No running jobs incorrectly marked as failed
- [ ] **< 10 Second Recovery**: Polling restarts within 10 seconds of OMA startup

### **Quality Metrics**
- [ ] **Test Coverage**: > 80% code coverage on new code
- [ ] **No Regressions**: All existing tests still pass
- [ ] **Documentation Complete**: Operator runbook is clear and tested

### **Operational Metrics**
- [ ] **Monitoring**: Can query job recovery status via API
- [ ] **Alerting**: Clear logs for recovery events
- [ ] **Manual Intervention**: Zero manual interventions needed after 1 week

---

## 🚨 **KNOWN RISKS & MITIGATIONS**

| Risk | Impact | Mitigation |
|------|--------|------------|
| **VMA API changes break detection** | High | Version VMA API responses, add schema validation |
| **Performance impact from health monitor** | Medium | Limit to 1-minute intervals, optimize queries |
| **Race condition during concurrent restarts** | Medium | Add locking around polling map operations |
| **False positives from VMA temporary failures** | Low | Grace periods and retry logic |

---

## 📝 **NOTES**

### **Implementation Order**
1. **Phase 1 (Critical)** - Do this first, enables basic recovery
2. **Phase 2 (High)** - Error detection fixes, essential for reliability
3. **Phase 3 (Medium)** - Database enhancements, helpful for debugging
4. **Phase 4 (Low)** - API endpoints, nice-to-have observability
5. **Phase 5 (Critical)** - Testing must be thorough
6. **Phase 6 (Required)** - Documentation and deployment

### **Dependencies**
- No external dependencies
- All changes internal to OMA codebase
- VMA API remains unchanged
- Compatible with existing database schema (Phase 1-2)

### **Rollback Plan**
If issues arise:
1. Revert to previous OMA API binary
2. Manual job recovery may be needed for in-flight jobs
3. Document any stuck jobs for manual cleanup

---

## ✅ **COMPLETION CHECKLIST**

### **Phase 1 Complete**
- [ ] All Task 1.1 items completed
- [ ] All Task 1.2 items completed
- [ ] All Task 1.3 items completed
- [ ] All Task 1.4 items completed
- [ ] Basic recovery working in dev environment

### **Phase 2 Complete**
- [ ] All Task 2.1 items completed
- [ ] All Task 2.2 items completed
- [ ] All Task 2.3 items completed
- [ ] Error detection working correctly

### **Phase 3 Complete**
- [ ] Database migrations created and tested
- [ ] Schema changes deployed to dev

### **Phase 4 Complete**
- [ ] API endpoints implemented
- [ ] Swagger documentation updated

### **Phase 5 Complete**
- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] Production testing scenarios validated

### **Phase 6 Complete**
- [ ] All documentation updated
- [ ] Deployed to QC and tested
- [ ] Deployed to production

### **Project Complete**
- [ ] All objectives met
- [ ] All success metrics achieved
- [ ] Zero regressions
- [ ] Operators trained on new features

---

**Status Tracking**: Update this section as work progresses

**Current Phase**: 🔴 Phase 1 - Not Started  
**Blockers**: None  
**Next Steps**: Begin Task 1.1.1 - Add VMAProgressClient to ProductionJobRecovery

---

**Last Updated**: October 3, 2025  
**Next Review**: After Phase 1 completion


