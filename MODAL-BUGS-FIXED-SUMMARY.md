# Machine Backup Details Modal - Bug Fixes Summary

**Date:** October 10, 2025  
**Status:** ✅ IMMEDIATE BUGS FIXED

---

## 🐛 Issues Fixed

### 1. Duration Showing 0s for All Backups ✅ FIXED

**Root Cause:** The `started_at` timestamp was never being set when backup jobs were created.

**Location:** `sha/api/handlers/backup_handlers.go` line 242-251

**Fix Applied:**
```go
// BEFORE (Missing started_at):
INSERT INTO backup_jobs (
    id, vm_backup_context_id, vm_context_id, vm_name, repository_id,
    backup_type, status, repository_path, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)

// AFTER (started_at included):
INSERT INTO backup_jobs (
    id, vm_backup_context_id, vm_context_id, vm_name, repository_id,
    backup_type, status, repository_path, created_at, started_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)  // ✅ started_at = created_at
```

**Result:** All NEW backups will now have proper `started_at` timestamps, enabling accurate duration calculations.

**Note:** Existing backups in the database still have `started_at = NULL`, so they'll continue to show 0s duration. New backups will show correct durations.

---

### 2. Modal API Fields Missing ✅ FIXED (Previous Session)

**Issue:** Frontend expected `type`, telemetry fields (`current_phase`, `progress_percent`, etc.)

**Fix:** SHA API now returns all required fields

**Binary:** `sendense-hub-v2.27.0-backup-modal-api-fix`

---

## 📦 Deployment

**Current Binary:** `sendense-hub-v2.28.0-started-at-fix`  
**Service:** `sendense-hub.service` (restarted and running)  
**Test:** Next backup will have started_at set automatically

---

## 📊 Remaining Issues for Discussion

### 1. Modal Sizing / Layout

**Issue:** Modal doesn't optimally display the data
- Columns may be too narrow/wide
- Table height could be adjusted
- KPI cards could be better sized

**Solution:** This is a frontend CSS/layout issue for Grok to address

---

### 2. Error Messages Not Displayed

**Issue:** No dedicated area in modal to show job error messages
- Failed backups have `error_message` in database
- Currently showing inline under backup (Grok implemented this)
- Could be more prominent

**Recommendation:** Keep current inline display, it's actually good!

---

### 3. Telemetry History / Performance Charts (MAJOR FEATURE)

**Current State:**
- Telemetry data is stored as single row that gets overwritten
- Only shows final state (100% complete, last speed)
- **No historical performance data** for charts

**User Request:**
- Tab 1: Summary (current view)
- Tab 2: Performance Chart (histogram of transfer speed over time)
- Tab 3: Job Details (logs, errors, technical info)
- Donut chart: Success vs Failed ratio
- Ability to select individual jobs and see their performance

**The Big Question:**
> Is it feasible to store telemetry snapshots for charting without database bloat?

---

## 💡 Proposed Solution: Hybrid Telemetry History

### Architecture

**Table: `job_telemetry_snapshots`**
```sql
CREATE TABLE job_telemetry_snapshots (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL,
    job_type VARCHAR(50) NOT NULL,  -- backup, restore, replication
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Snapshot metrics
    bytes_transferred BIGINT DEFAULT 0,
    transfer_speed_bps BIGINT DEFAULT 0,
    progress_percent DECIMAL(5,2) DEFAULT 0.00,
    current_phase VARCHAR(100),
    
    -- Indexes for fast queries
    INDEX idx_job_id (job_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_job_timestamp (job_id, timestamp)
);
```

### Strategy

**During Job Execution:**
- Store snapshot every 30 seconds in `job_telemetry_snapshots`
- Current single-row telemetry in `backup_jobs` keeps updating (for real-time display)

**On Job Completion:**
- Calculate summary metrics (peak speed, avg speed, min speed)
- Store rolled-up summary in `backup_jobs.performance_metrics` JSON field:
  ```json
  {
    "peak_speed_bps": 500000000,
    "avg_speed_bps": 350000000,
    "min_speed_bps": 200000000,
    "duration_seconds": 300,
    "snapshots": 10,
    "chart_data": [
      {"t": 0, "s": 300000000, "p": 0},
      {"t": 30, "s": 350000000, "p": 15},
      {"t": 60, "s": 400000000, "p": 30}
      // ... etc
    ]
  }
  ```
- **Delete snapshots from `job_telemetry_snapshots`** (keep database lean)

### Storage Impact

**Example:**
- 5-minute backup: 10 snapshots × 100 bytes = 1 KB (deleted after completion)
- 1-hour backup: 120 snapshots × 100 bytes = 12 KB (deleted after completion)
- Final JSON in `backup_jobs`: ~2-5 KB per job (kept forever)

**1000 backups = 2-5 MB total** (trivial storage cost)

### Benefits

1. **Real-time charts** while job running (from snapshots table)
2. **Historical charts** after completion (from JSON field)
3. **Minimal storage** (delete snapshots after completion)
4. **Fast queries** (indexed snapshot lookups, single JSON read for history)
5. **Standard SQL** (no special database features needed)

---

## 🎯 Recommendations for Next Steps

### Immediate (Grok can do now):

1. **Modal layout tweaks** - Adjust column widths, spacing, card sizing
2. **Success vs Failed donut chart** - Can be implemented NOW with existing data:
   ```sql
   SELECT status, COUNT(*) as count 
   FROM backup_jobs 
   WHERE vm_name = 'pgtest1' AND repository_id = 'xxx'
   GROUP BY status
   ```
3. **Job selector dropdown** - Add dropdown to modal to switch between machines

### Medium Term (Requires Backend Work):

1. **Implement telemetry snapshots system** (SHA modifications):
   - Create `job_telemetry_snapshots` table
   - Modify telemetry receiver to store snapshots every 30s
   - Add completion handler to roll up metrics and delete snapshots
   - Add API endpoint: `GET /api/v1/telemetry/history/{job_id}`

2. **Implement chart tabs in modal** (Grok frontend work):
   - Tab 1: Summary (current view)
   - Tab 2: Performance Chart (speed histogram using chart library)
   - Tab 3: Job Details (logs, metadata)

---

## ✅ Current Status

- ✅ `started_at` bug fixed (v2.28.0)
- ✅ API fields fixed (v2.27.0)
- ✅ Telemetry data collection working (SBC → SHA)
- ✅ Modal displaying backup history
- 🔄 Modal layout needs tweaks
- 🔄 Telemetry history system needs implementation for charts

**Bottom Line:** The immediate bugs are fixed. The chart/history feature is a bigger architectural addition but totally feasible with a hybrid snapshot approach.

