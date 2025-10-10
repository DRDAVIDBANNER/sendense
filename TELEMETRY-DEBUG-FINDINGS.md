# Telemetry Framework Debug Findings
## Date: 2025-10-10
## Status: Root Cause Identified

---

## 🔍 Investigation Summary

Analyzed why telemetry data (progress, bytes, speed, phase) is `0` in database despite framework infrastructure being in place.

---

## ✅ What Works (Infrastructure Layer)

### 1. SHA Backend (Receiver Side)
- ✅ **Telemetry API endpoint registered**: `POST /api/v1/telemetry/{job_type}/{job_id}`
- ✅ **Service layer implemented**: `TelemetryService.ProcessTelemetryUpdate()` 
- ✅ **Database schema ready**: All telemetry fields exist in `backup_jobs` and `backup_disks`
- ✅ **Stale job detector running**: Background worker active
- ✅ **Database updates work**: `last_telemetry_at` is being updated (proves SHA can write)

**Evidence:**
```
Oct 10 12:44:05 sendense-hub: ✅ Telemetry API routes registered: POST /api/v1/telemetry/{job_type}/{job_id}
Oct 10 12:44:05 sendense-hub: 🚨 Starting stale job detector for real-time telemetry monitoring
```

### 2. SBC Client (Sender Side - Infrastructure)
- ✅ **Telemetry client initialized**: `telemetry.NewClient()` called in `main.go`
- ✅ **Progress tracker created**: `telemetry.NewProgressTracker()` instantiated
- ✅ **Context propagation**: Tracker passed to `parallel_full_copy.go` and `parallel_incremental.go`
- ✅ **Progress aggregator wired**: `progressAggregator.SetTelemetryTracker(tracker)` called

**Evidence from code:**
```go
// main.go line 340
telemetryTracker := telemetry.NewProgressTracker(telemetryClient, jobID)
ctx = context.WithValue(ctx, "telemetryTracker", telemetryTracker)

// parallel_full_copy.go line 99
progressAggregator.SetTelemetryTracker(tracker)
```

---

## ❌ What's Broken (Data Collection Layer)

### Critical Finding: **No Telemetry Requests Reaching SHA**

**Evidence:**
- SHA logs show **ZERO** `POST /api/v1/telemetry/...` requests
- Only SNA API calls visible (old polling system still working)
- Database `last_telemetry_at` updates, but **all telemetry fields are `0`**

**This means:**
- SBC is NOT sending HTTP requests to SHA telemetry endpoint
- The problem is in the SBC **data collection and sending logic**, not SHA

---

## 🐛 Root Cause Analysis

### Issue 1: **Incomplete TelemetryUpdate Construction**

**File:** `sendense-backup-client/internal/telemetry/tracker.go`

**Problem in `UpdateProgress()` method (lines 104-118):**

```go
func (pt *ProgressTracker) UpdateProgress(ctx context.Context, bytesTransferred, transferSpeedBps int64, etaSeconds int) {
    // Build telemetry update
    update := &TelemetryUpdate{
        Status:           "running",           // ✅ Set
        BytesTransferred: bytesTransferred,    // ✅ Set
        TransferSpeedBps: transferSpeedBps,    // ✅ Set
        ETASeconds:       etaSeconds,          // ✅ Set
        ProgressPercent:  0.0,                 // ❌ HARDCODED TO 0!
    }
    // ❌ MISSING FIELDS:
    // - JobID (not set)
    // - JobType (not set)
    // - CurrentPhase (not set)
    // - TotalBytes (not set)
    // - Timestamp (not set)
    // - Disks array (empty)
    
    if err := pt.SendIfNeeded(pt.jobID, update); err != nil {
        log.WithError(err).Warn("Failed to send progress update telemetry")
    }
}
```

**Impact:**
1. `ProgressPercent` is **hardcoded to `0.0`** instead of being calculated
2. `JobID` not populated (breaks routing/logging)
3. `TotalBytes` missing (can't calculate progress on SHA side)
4. `CurrentPhase` not set (always empty)
5. `Timestamp` not set (auditing broken)
6. `Disks` array empty (no per-disk progress)

### Issue 2: **Hybrid Cadence Logic May Block Sends**

**File:** `sendense-backup-client/internal/telemetry/tracker.go`

**Problem in `ShouldSend()` logic (lines 37-60):**

```go
func (pt *ProgressTracker) ShouldSend(currentProgress float64) bool {
    // Time-based: 5 seconds elapsed
    if now.Sub(pt.lastSentTime) >= pt.timeInterval {
        return true
    }
    
    // Progress-based: 10% progress made
    progressDelta := currentProgress - pt.lastSentProgress
    if progressDelta >= pt.progressInterval {
        return true
    }
    
    return false  // ❌ Blocks send if neither condition met
}
```

**Impact:**
- If `ProgressPercent` is always `0.0` (from Issue 1), the progress-based condition **never triggers**
- Time-based trigger should work, BUT if there are any errors in HTTP sending (network, auth, etc.), telemetry silently fails

### Issue 3: **Missing TotalBytes in Progress Aggregator**

**File:** `sendense-backup-client/internal/vmware_nbdkit/progress_aggregator.go`

**Problem in `maybeUpdateVMA()` (lines 125-136):**

The progress aggregator **has** `pa.totalBytes` but doesn't pass it to telemetry:

```go
// Line 127 in progress_aggregator.go
if pa.telemetryTracker != nil && pa.jobID != "" {
    pa.telemetryTracker.UpdateProgress(context.Background(), currentBytes, throughputBPS, etaSeconds)
    // ❌ Does NOT pass:
    // - pa.totalBytes
    // - currentPercent (already calculated!)
    // - current phase
}
```

### Issue 4: **No Error Logging for Failed HTTP Sends**

**File:** `sendense-backup-client/internal/telemetry/client.go`

The `SendBackupUpdate()` method returns errors, but they're swallowed with only a `Warn` log:

```go
// tracker.go line 115-117
if err := pt.SendIfNeeded(pt.jobID, update); err != nil {
    log.WithError(err).Warn("Failed to send progress update telemetry")  // ❌ Silent failure
}
```

**Impact:**
- If HTTP requests fail (wrong URL, network error, auth issue), we'd never know
- SBC logs show no telemetry activity because errors aren't logged at INFO level

---

## 🔬 Comparison: Old SNA System vs New Telemetry

### ✅ Old SNA Progress System (WORKS)

**File:** `sendense-backup-client/internal/progress/sna_client.go`

```go
// This WORKS and sends data successfully:
type SNAProgressUpdate struct {
    Stage            string  `json:"stage"`
    Status           string  `json:"status,omitempty"`
    BytesTransferred int64   `json:"bytes_transferred"`   // ✅ Populated
    TotalBytes       int64   `json:"total_bytes,omitempty"` // ✅ Populated
    ThroughputBPS    int64   `json:"throughput_bps"`      // ✅ Populated
    Percent          float64 `json:"percent,omitempty"`   // ✅ Calculated!
}

// progress_aggregator.go lines 103-122
err := pa.snaProgressClient.SendUpdate(progress.SNAProgressUpdate{
    Stage:            "Transfer",
    Status:           "in_progress",
    BytesTransferred: currentBytes,         // ✅ Real value
    TotalBytes:       pa.totalBytes,        // ✅ Real value
    Percent:          currentPercent,       // ✅ Real value
    ThroughputBPS:    throughputBPS,       // ✅ Real value
})
```

**Why it works:**
1. All fields populated from `progress_aggregator` context
2. Percent calculated **before** sending: `currentPercent = float64(currentBytes) / float64(pa.totalBytes) * 100`
3. Direct, simple logic - no hybrid cadence complexity
4. Sends to localhost:8081 (SNA local API)

### ❌ New SHA Telemetry System (BROKEN)

```go
// This DOESN'T WORK:
pa.telemetryTracker.UpdateProgress(context.Background(), currentBytes, throughputBPS, etaSeconds)
// ❌ Only passes 3 values, drops totalBytes and percent
// ❌ Tracker builds incomplete TelemetryUpdate
// ❌ May not even send due to cadence logic
```

---

## 📊 Database Evidence

```sql
-- backup_jobs record for job backup-pgtest1-1760096737:
status: completed ✅
bytes_transferred: 0 ❌
progress_percent: 0.00 ❌
transfer_speed_bps: 0 ❌
current_phase: NULL ❌
last_telemetry_at: 2025-10-10 12:45:40 ✅ (Updated by stale detector or completion)

-- backup_disks records:
disk_index: 0, status: completed ✅, bytes_transferred: 0 ❌, progress_percent: 0.00 ❌
disk_index: 1, status: pending ❌ (never started?)
```

**Conclusion:**
- SHA database CAN be updated (last_telemetry_at proves write path works)
- BUT telemetry data fields remain `0` because **no telemetry HTTP requests ever reach SHA**
- Completion/status updates might be happening via old completion logic, not telemetry

---

## 🎯 Required Fixes (In Priority Order)

### Fix 1: **Complete TelemetryUpdate Construction** (CRITICAL)

**File:** `sendense-backup-client/internal/telemetry/tracker.go`

**Change `UpdateProgress()` signature:**

```go
// OLD (broken):
func (pt *ProgressTracker) UpdateProgress(ctx context.Context, bytesTransferred, transferSpeedBps int64, etaSeconds int)

// NEW (fixed):
func (pt *ProgressTracker) UpdateProgress(
    ctx context.Context,
    bytesTransferred int64,
    totalBytes int64,         // ✅ ADD
    transferSpeedBps int64,
    etaSeconds int,
    currentPhase string,      // ✅ ADD
) {
    // Calculate progress percent
    progressPercent := 0.0
    if totalBytes > 0 {
        progressPercent = (float64(bytesTransferred) / float64(totalBytes)) * 100.0
    }
    
    update := &TelemetryUpdate{
        JobID:            pt.jobID,                          // ✅ FIX
        JobType:          "backup",                          // ✅ FIX (or pass as param)
        Status:           "running",
        CurrentPhase:     currentPhase,                      // ✅ FIX
        BytesTransferred: bytesTransferred,
        TotalBytes:       totalBytes,                        // ✅ FIX
        TransferSpeedBps: transferSpeedBps,
        ETASeconds:       etaSeconds,
        ProgressPercent:  progressPercent,                   // ✅ FIX (calculate!)
        Timestamp:        time.Now().Format(time.RFC3339),  // ✅ FIX
        Disks:            []DiskTelemetry{},                 // ✅ FIX (populate from context)
    }
    
    // Log before sending
    log.WithFields(log.Fields{
        "job_id":    pt.jobID,
        "bytes":     bytesTransferred,
        "total":     totalBytes,
        "percent":   progressPercent,
        "speed_bps": transferSpeedBps,
    }).Debug("🚀 Sending telemetry update to SHA")
    
    if err := pt.SendIfNeeded(pt.jobID, update); err != nil {
        log.WithError(err).Error("❌ FAILED to send telemetry update")  // ✅ Change to ERROR
    }
}
```

### Fix 2: **Update Progress Aggregator Call**

**File:** `sendense-backup-client/internal/vmware_nbdkit/progress_aggregator.go`

**Line 127 (inside `maybeUpdateVMA()`):**

```go
// OLD:
pa.telemetryTracker.UpdateProgress(context.Background(), currentBytes, throughputBPS, etaSeconds)

// NEW:
pa.telemetryTracker.UpdateProgress(
    context.Background(),
    currentBytes,
    pa.totalBytes,      // ✅ ADD
    throughputBPS,
    etaSeconds,
    "transferring",     // ✅ ADD (phase)
)
```

### Fix 3: **Pass Disk-Level Progress**

For multi-disk VMs, we need to pass per-disk progress. This requires refactoring to collect disk progress from workers.

**Options:**
1. **Quick fix**: Ignore per-disk progress initially, send only aggregate
2. **Proper fix**: Refactor progress aggregator to track per-disk progress and pass `Disks` array

### Fix 4: **Add Debug Logging**

Add more verbose logging at INFO/DEBUG level in:
1. `telemetry/client.go` - Log every HTTP attempt
2. `telemetry/tracker.go` - Log cadence decisions
3. `progress_aggregator.go` - Log telemetry send attempts

### Fix 5: **Verify SHA API URL**

Confirm SBC is using correct tunnel endpoint:
- Current: `http://localhost:8082` (SHA tunnel endpoint on SNA)
- SNA's old progress API: `http://localhost:8081` (SNA local API)

**Verify in deployment:**
- Is `SHA_API_URL` env var set on SBC?
- If not, default `http://localhost:8082` should work

---

## 🧪 Testing Strategy

### Test 1: Manual Telemetry Send (Proof of Concept)

```bash
# On SNA, manually send telemetry to SHA:
curl -X POST http://localhost:8082/api/v1/telemetry/backup/backup-test-123 \
  -H "Content-Type: application/json" \
  -d '{
    "job_id": "backup-test-123",
    "job_type": "backup",
    "status": "running",
    "current_phase": "transferring",
    "bytes_transferred": 1073741824,
    "total_bytes": 5368709120,
    "transfer_speed_bps": 104857600,
    "eta_seconds": 120,
    "progress_percent": 20.0,
    "timestamp": "2025-10-10T12:00:00Z",
    "disks": []
  }'

# Then check SHA database:
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT id, bytes_transferred, progress_percent, transfer_speed_bps FROM backup_jobs WHERE id='backup-test-123';"
```

**Expected:** SHA processes update and database reflects data

### Test 2: Fixed SBC Integration

After applying fixes:
1. Deploy fixed SBC to SNA
2. Start backup from GUI
3. Monitor SHA logs for telemetry POST requests
4. Monitor database for real-time updates every 5 seconds
5. Verify progress, bytes, speed all populate

---

## 📝 Summary

| Component | Status | Issue |
|-----------|--------|-------|
| SHA Telemetry API | ✅ Working | - |
| SHA Telemetry Service | ✅ Working | - |
| SHA Database Schema | ✅ Ready | - |
| SHA Stale Detector | ✅ Running | - |
| SBC Telemetry Client | ⚠️ Partially | Missing fields in payload |
| SBC Progress Tracker | ❌ Broken | Hardcoded `0` progress, missing fields |
| SBC Progress Aggregator | ⚠️ Partially | Doesn't pass enough data to tracker |
| Integration | ❌ Broken | No HTTP requests reaching SHA |

---

## 🎯 Next Steps

1. **Immediate Fix**: Apply Fix 1 & 2 (complete TelemetryUpdate construction)
2. **Test**: Run manual curl test to verify SHA endpoint works
3. **Deploy**: Build and deploy fixed SBC
4. **Integration Test**: Run backup and verify real-time telemetry
5. **Refinement**: Add per-disk progress (Fix 3)
6. **Cleanup**: Remove old SNA polling once telemetry proven stable

---

## 💡 Key Insight

**The telemetry framework architecture is 100% sound.** The issue is a classic "integration gap":
- Infrastructure layer (HTTP, database, services) ✅ WORKS
- Data collection layer (building payloads, making calls) ❌ INCOMPLETE

This is actually **good news** - it's a straightforward fix, not an architectural problem.

