# Parallel NBD Integration Guide

**Date**: October 3, 2025  
**Status**: 🎯 **IMPLEMENTATION COMPLETE - READY FOR TESTING**

---

## 📦 **FILES CREATED**

### **Core Implementation**

1. **`extent_utils.go`** - Extent coalescing and splitting logic
   - `coalesceExtents()` - Merges adjacent extents
   - `splitExtentsAcrossWorkers()` - Distributes work across workers
   - `calculateTotalBytes()` - Progress calculation helper

2. **`parallel_worker.go`** - Worker pool implementation
   - `copyWorker()` - Goroutine worker processing extents
   - `copyExtentWithRetry()` - Retry logic with exponential backoff
   - `copyExtent()` - Single extent copy with sparse optimization

3. **`progress_aggregator.go`** - Multi-worker progress tracking
   - `ProgressAggregator` - Centralized progress collection
   - `Run()` - Background goroutine collecting worker updates
   - `SendFinalUpdate()` - 100% completion notification

4. **`parallel_incremental.go`** - Main orchestrator
   - `ParallelIncrementalCopyToTarget()` - New parallel copy method
   - `IncrementalCopyToTargetAutoSelect()` - Auto-fallback wrapper
   - `ParallelIncrementalCopyEnabled()` - Feature flag check

---

## 🔧 **INTEGRATION STEPS**

### **Step 1: Enable Parallel Copy**

Set environment variable on VMA:

```bash
# Enable parallel NBD copy
export MIGRATEKIT_PARALLEL_NBD=true

# Or in systemd service file:
Environment="MIGRATEKIT_PARALLEL_NBD=true"
```

### **Step 2: Modify SyncToTarget Method**

Replace the incremental copy call in `vmware_nbdkit.go:SyncToTarget()`:

**Current Code** (line 922):
```go
} else {
    err = s.IncrementalCopyToTarget(ctx, t, path)
    if err != nil {
        return err
    }
}
```

**New Code with Auto-Fallback**:
```go
} else {
    // Use auto-select method for safe parallel copy with fallback
    err = s.IncrementalCopyToTargetAutoSelect(ctx, t, path)
    if err != nil {
        return err
    }
}
```

### **Step 3: Build and Deploy**

```bash
cd /home/pgrayson/migratekit-cloudstack/source/current/migratekit
go build -o migratekit-v2.22.0-parallel-nbd .

# Deploy to VMA
scp -i ~/.ssh/cloudstack_key migratekit-v2.22.0-parallel-nbd pgrayson@10.0.100.231:/home/pgrayson/migratekit-cloudstack/

# Update symlink
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 \
    "sudo ln -sf /home/pgrayson/migratekit-cloudstack/migratekit-v2.22.0-parallel-nbd \
                 /home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel"

# Restart VMA API (if needed)
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo systemctl restart vma-api"
```

---

## 🧪 **TESTING STRATEGY**

### **Phase 1: Feature Flag Disabled (Baseline)**

Test with existing serial copy to establish baseline:

```bash
# Ensure feature flag is disabled
unset MIGRATEKIT_PARALLEL_NBD

# Run incremental sync on pgtest1
# Expected: Uses serial copy (existing behavior)
# Expected: ~150 MB/s throughput
```

### **Phase 2: Parallel Copy with Small VM**

Enable parallel copy and test with small VM:

```bash
# Enable parallel copy
export MIGRATEKIT_PARALLEL_NBD=true

# Run incremental sync on pgtest1 (small VM, ~100GB)
# Expected: 2-4 workers spawned
# Expected: 300-400 MB/s throughput (2-2.5x improvement)
# Expected: All workers complete successfully
```

### **Phase 3: Parallel Copy with Large VM**

Test with larger multi-disk VM:

```bash
# Enable parallel copy
export MIGRATEKIT_PARALLEL_NBD=true

# Run incremental sync on pgtest2 (larger VM with multiple disks)
# Expected: Per-disk parallelization
# Expected: 400-500 MB/s throughput per disk
# Expected: No VDDK connection errors
```

### **Phase 4: Failure Recovery Testing**

Test automatic fallback on errors:

```bash
# Simulate failure conditions:
# 1. Kill VMA API mid-transfer (context cancellation)
# 2. Network interruption (retry mechanism)
# 3. Disable feature flag mid-run (next run uses serial)

# Expected: Graceful degradation to serial copy
# Expected: Error logging but no data corruption
```

---

## 📊 **MONITORING AND METRICS**

### **Key Log Messages**

Look for these log entries to confirm parallel operation:

```
🚀 Starting parallel incremental copy
🔗 Extent coalescing completed
🔧 Using 4 parallel workers
📦 Worker extent allocation (per worker)
🚀 Worker started (per worker)
📊 Worker progress (every 10 extents per worker)
✅ Worker completed successfully (per worker)
✅ Parallel incremental copy completed successfully
```

### **Performance Metrics to Track**

1. **Throughput Improvement**:
   - Serial baseline: ~150 MB/s
   - Parallel target: 400-500 MB/s
   - Improvement ratio: 2.5-3.5x

2. **Worker Utilization**:
   - All 4 workers should be active
   - Balanced extent distribution
   - Similar completion times

3. **Error Rates**:
   - Retry rate: <1% of extents
   - Worker failures: 0
   - Context cancellations: Handled gracefully

4. **Memory Usage**:
   - Serial: ~32 MB per disk
   - Parallel: ~128 MB per disk (4 workers × 32 MB)
   - Expected increase: 4x (acceptable)

---

## 🎯 **CONFIGURATION OPTIONS**

### **Environment Variables**

| Variable | Values | Default | Description |
|----------|--------|---------|-------------|
| `MIGRATEKIT_PARALLEL_NBD` | `true`, `1`, `enabled` | `false` | Enable parallel NBD copy |

### **Code Constants**

Tune these in `parallel_incremental.go` if needed:

```go
const (
    DefaultNumWorkers      = 4                 // Workers per disk
    CoalesceGapThreshold   = 1 * 1024 * 1024   // 1 MB gap merging
    MaxRetries             = 3                 // Retry attempts
    InitialRetryDelay      = 1 * time.Second   // Base retry delay
)
```

---

## 🔄 **ROLLBACK PROCEDURE**

### **Option 1: Disable Feature Flag**

Fastest rollback - no code changes:

```bash
# Unset environment variable
unset MIGRATEKIT_PARALLEL_NBD

# Or in systemd:
# Remove "Environment=MIGRATEKIT_PARALLEL_NBD=true"
sudo systemctl daemon-reload
sudo systemctl restart vma-api
```

### **Option 2: Revert Code Changes**

If parallel copy causes issues:

```bash
# Revert to previous binary version
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 \
    "sudo ln -sf /home/pgrayson/migratekit-cloudstack/migratekit-v2.21.1-chunk-size-fix \
                 /home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel"

# Restart VMA API
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo systemctl restart vma-api"
```

### **Option 3: Code Modification**

Change `SyncToTarget()` to always use serial copy:

```go
} else {
    // Force serial copy (rollback from parallel)
    err = s.IncrementalCopyToTarget(ctx, t, path)
    if err != nil {
        return err
    }
}
```

---

## 🐛 **TROUBLESHOOTING**

### **Issue: Workers Not Starting**

**Symptoms**:
- Log shows "Using 1 parallel workers" instead of 4
- Throughput same as serial copy

**Causes**:
- Too few extents for parallelization
- `determineWorkerCount()` choosing serial mode

**Solution**:
- Lower thresholds in `determineWorkerCount()`
- Or force 4 workers for testing

### **Issue: VDDK Connection Errors**

**Symptoms**:
- "worker N: failed to connect to source NBD"
- Workers failing immediately

**Causes**:
- Too many concurrent VDDK connections
- VMware ESXi throttling

**Solution**:
- Reduce `DefaultNumWorkers` from 4 to 2
- Check VMware NBD connection limits

### **Issue: Progress Not Updating**

**Symptoms**:
- VMA progress stuck at 0%
- No progress updates in logs

**Causes**:
- Progress channel full (blocking workers)
- Progress aggregator not running

**Solution**:
- Increase `progressChan` buffer size (currently 1000)
- Check for aggregator errors in logs

### **Issue: Memory Exhaustion**

**Symptoms**:
- OOM errors during large transfers
- System slowdown

**Causes**:
- Too many workers or too large buffers
- Memory not being released

**Solution**:
- Reduce `DefaultNumWorkers`
- Reduce `MaxChunkSize` if needed
- Check for buffer leaks

---

## ✅ **PRODUCTION READINESS CHECKLIST**

Before deploying to production:

- [ ] Tested with feature flag disabled (baseline performance)
- [ ] Tested with feature flag enabled (parallel performance)
- [ ] Verified 2-3x throughput improvement
- [ ] Tested automatic fallback on errors
- [ ] Monitored memory usage (acceptable increase)
- [ ] Checked VMware VDDK connection stability
- [ ] Tested with multiple concurrent VMs
- [ ] Verified VMA progress updates working
- [ ] Tested context cancellation (clean shutdown)
- [ ] Documented rollback procedure
- [ ] Created monitoring alerts for worker failures

---

## 📈 **EXPECTED RESULTS**

### **Before (Serial Copy)**

```
Incremental sync: 45 minutes
Throughput: ~150 MB/s
CPU utilization: 20-30%
Memory: 32 MB per disk
```

### **After (Parallel Copy - 4 Workers)**

```
Incremental sync: 12-15 minutes (3-4x faster)
Throughput: 400-500 MB/s (2.5-3.5x improvement)
CPU utilization: 60-80%
Memory: 128 MB per disk (4x increase, acceptable)
```

### **Operational Benefits**

- ✅ Reduced migration windows by 3-4x
- ✅ Better handling of small extent workloads
- ✅ Improved resource utilization
- ✅ Zero breaking changes (feature flag controlled)
- ✅ Automatic fallback on errors

---

## 🎓 **NEXT STEPS**

1. **Phase 1**: Test with feature flag disabled (baseline)
2. **Phase 2**: Enable feature flag and test pgtest1 (small VM)
3. **Phase 3**: Test with pgtest2 (larger multi-disk VM)
4. **Phase 4**: Performance benchmarking and tuning
5. **Phase 5**: Production deployment (gradual rollout)

---

**Status**: ✅ **READY FOR TESTING**  
**Risk Level**: 🟢 **LOW** (feature flag controlled with automatic fallback)  
**Recommendation**: **PROCEED WITH TESTING PLAN**

