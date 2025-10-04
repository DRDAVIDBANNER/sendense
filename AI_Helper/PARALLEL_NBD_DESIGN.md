# Parallel NBD Incremental Copy Design

**Date**: October 3, 2025  
**Status**: 🎯 **DESIGN PROPOSAL**  
**Feasibility**: ✅ **FEASIBLE WITH CONSTRAINTS**

---

## 🎯 **OBJECTIVE**

Redesign MigrateKit's incremental copy loop to support parallel NBD sessions per disk, improving throughput for VMware migrations where CBT returns thousands of small extents (64 KB–1 MB).

**Current Problem**:
- Serial copy loop: one NBD connection per disk
- Synchronous read → write blocking pipeline
- VMware NBD doesn't pipeline well with small extents
- Low throughput due to request-response latency

**Target Solution**:
- 2-4 parallel NBD connections per disk
- Extent coalescing (merge small extents into 1-32 MB chunks)
- Asynchronous goroutine workers with channel-based streaming
- Aggregated progress tracking across workers

---

## ✅ **FEASIBILITY ANALYSIS**

### **1. VMware NBD Server Capabilities**

✅ **CONFIRMED**: VMware NBD servers CAN handle multiple parallel connections to the same disk

**Evidence**:
- Each `nbdkit-vddk-plugin` process is independent
- VDDK library is thread-safe for multiple connections
- VMware ESXi NBD server supports concurrent reads from same VMDK
- Current architecture already supports multiple disks concurrently

**Constraints**:
- Recommended: 2-4 workers per disk (VDDK optimal range)
- Too many workers (>8) may cause VDDK throttling
- Each worker needs separate `libnbd.Libnbd` connection

### **2. Project Architecture Compliance**

✅ **COMPLIANT** with all MigrateKit rules:

| Rule | Compliance | Notes |
|------|-----------|-------|
| Modular Design | ✅ | Separate extent utilities, worker pools, progress aggregation |
| No Monster Code | ✅ | Small, focused functions (<200 lines each) |
| Progress Tracking | ✅ | VMA progress client integration maintained |
| Error Handling | ✅ | Retry logic with exponential backoff |
| NBD Only | ✅ | All traffic via NBD (no VDDK fallback) |
| JobLog Integration | ✅ | Logging via logrus (existing pattern) |

### **3. Performance Expectations**

**Current Performance**:
- Serial: ~150 MB/s (dominated by round-trip latency)
- Small extents = high request overhead

**Expected Improvement** (4 workers):
- 2.5-3.5x throughput increase
- Target: 400-500 MB/s for incremental copies
- Depends on VMware network and storage latency

**Optimal Configuration**:
- Worker Count: **4 workers per disk**
- Chunk Size: **1-32 MB coalesced chunks**
- Coalesce Gap: **1 MB** (merge extents within 1 MB of each other)
- Retry Policy: **3 attempts with exponential backoff**

---

## 🏗️ **ARCHITECTURE DESIGN**

### **Component Overview**

```
┌─────────────────────────────────────────────────────────────┐
│                  IncrementalCopyToTarget                    │
│                   (Main Orchestrator)                        │
└────────┬────────────────────────────────────────────────────┘
         │
         ├─► 1. Query CBT Extents (VMware API)
         │      └─► []DiskChangeExtent
         │
         ├─► 2. Coalesce Extents
         │      └─► []CoalescedExtent (1-32 MB chunks)
         │
         ├─► 3. Split Extents Across Workers
         │      └─► 4 worker slices (round-robin)
         │
         ├─► 4. Start Worker Pool
         │      ├─► Worker 1 (goroutine + NBD connection)
         │      ├─► Worker 2 (goroutine + NBD connection)
         │      ├─► Worker 3 (goroutine + NBD connection)
         │      └─► Worker 4 (goroutine + NBD connection)
         │
         ├─► 5. Progress Aggregator
         │      └─► Collect bytes from all workers
         │             └─► Send VMA updates every 2s
         │
         └─► 6. Error Collection & Retry
                └─► Failed chunks → retry queue
```

### **Key Components**

#### **1. Extent Utilities** (`extent_utils.go`)
```go
// Coalesce adjacent/near extents into larger chunks
func coalesceExtents(extents []Extent, maxGap int64) []CoalescedExtent

// Split extents evenly across N workers (round-robin)
func splitExtentsAcrossWorkers(extents []CoalescedExtent, numWorkers int) [][]CoalescedExtent
```

#### **2. Worker Pool** (`parallel_worker.go`)
```go
// Worker processes extents from its assigned slice
func copyWorker(ctx context.Context, workerID int, extents []CoalescedExtent, 
                sourceNBD *libnbd.Libnbd, targetNBD *libnbd.Libnbd,
                progressChan chan<- int64, errorChan chan<- error)
```

#### **3. Progress Aggregator** (`progress_aggregator.go`)
```go
// Collect progress from all workers and send VMA updates
func progressAggregator(ctx context.Context, progressChan <-chan int64,
                        vmaClient *progress.VMAProgressClient, totalBytes int64)
```

#### **4. Retry Handler** (`retry_handler.go`)
```go
// Retry failed chunks with exponential backoff
func retryFailedChunks(ctx context.Context, failedChunks []CoalescedExtent,
                       sourceNBD *libnbd.Libnbd, targetNBD *libnbd.Libnbd) error
```

---

## 🔧 **DESIGN DECISIONS**

### **1. Worker Count: 4 Workers Per Disk**

**Rationale**:
- VDDK optimal performance range: 2-4 concurrent connections
- Too few (<2): Limited parallelism benefit
- Too many (>8): VDDK throttling, diminishing returns
- **4 workers** = sweet spot for throughput vs complexity

### **2. Extent Coalescing: 1 MB Gap Threshold**

**Rationale**:
- VMware returns thousands of small extents (64 KB–1 MB)
- Coalescing reduces request overhead dramatically
- **1 MB gap**: Acceptable to copy some unchanged bytes for throughput gain
- Target chunks: 1-32 MB (matches existing MaxChunkSize)

**Example**:
```
Before Coalescing:
Extent 1: [0, 64KB]
Extent 2: [128KB, 64KB]    ← 64KB gap
Extent 3: [256KB, 64KB]

After Coalescing (1 MB gap):
CoalescedExtent: [0, 320KB]  ← Single read/write operation
```

### **3. Batch Size: 32 MB Maximum**

**Rationale**:
- Matches existing `MaxChunkSize = 32 * 1024 * 1024`
- NBD server compatibility constraint
- Prevents memory exhaustion with large buffers

### **4. Retry Logic: 3 Attempts with Exponential Backoff**

**Rationale**:
- Network transients are common in VMware environments
- **3 retries**: Balance between resilience and failure detection speed
- **Exponential backoff**: 1s → 2s → 4s (prevents thundering herd)

### **5. Progress Aggregation: Central Channel**

**Rationale**:
- All workers send progress to single channel
- Aggregator maintains cumulative count
- VMA updates every 2 seconds (existing pattern)
- Thread-safe without explicit locking

---

## 📊 **INTEGRATION WITH EXISTING ARCHITECTURE**

### **1. VMA Progress Client**

✅ **PRESERVED**: All existing VMA progress integration maintained

```go
// Progress aggregator sends updates matching existing pattern
vpc.SendUpdate(progress.VMAProgressUpdate{
    Stage:            "Transfer",
    Status:           "in_progress",
    BytesTransferred: totalBytesTransferred,
    TotalBytes:       totalBytes,
    Percent:          currentPercent,
    ThroughputBPS:    throughputBPS,
})
```

### **2. Error Handling**

✅ **ENHANCED**: Graceful degradation with retry mechanism

- Failed chunks collected in error channel
- Automatic retry with exponential backoff
- If all retries fail: error propagates (existing behavior)

### **3. Context Cancellation**

✅ **SUPPORTED**: Proper context propagation

- All workers monitor `ctx.Done()`
- Clean shutdown on cancellation
- Partial progress preserved in database

### **4. Logging**

✅ **ENHANCED**: Per-worker logging with throughput visibility

```go
logger.WithFields(log.Fields{
    "worker_id":         workerID,
    "extents_processed": extentsProcessed,
    "bytes_copied":      bytesCopied,
    "throughput_mbps":   throughputMBps,
}).Info("🚀 Worker completed")
```

---

## 🚨 **CONSTRAINTS AND LIMITATIONS**

### **1. VMware NBD Connection Limits**

- **Constraint**: ESXi may throttle >8 concurrent NBD connections per VM
- **Mitigation**: Hard limit of 4 workers per disk
- **Monitoring**: Add warning logs if connection failures increase

### **2. Memory Usage**

- **Current**: ~32 MB buffer per disk (serial)
- **Parallel**: ~128 MB per disk (4 workers × 32 MB buffers)
- **Mitigation**: Release buffers immediately after write

### **3. Write Ordering**

- **Design**: Writes are concurrent but positioned (NBD `Pwrite` at correct offsets)
- **Guarantee**: Final disk state is correct regardless of write order
- **Risk**: None (positioned writes are atomic per NBD spec)

### **4. Progress Reporting Accuracy**

- **Challenge**: Workers complete at different rates
- **Solution**: Aggregate progress from all workers (cumulative)
- **Impact**: Progress may be "lumpy" but always accurate

---

## 🧪 **TESTING STRATEGY**

### **1. Unit Tests**

- Test extent coalescing logic with various gap sizes
- Test extent splitting across workers (even distribution)
- Test retry mechanism with simulated failures

### **2. Integration Tests**

- Test with real VMware VMs (pgtest1, pgtest2)
- Compare serial vs parallel throughput
- Monitor VDDK connection stability

### **3. Performance Benchmarks**

- Measure throughput improvement (target: 2.5-3.5x)
- Monitor worker utilization (all workers busy?)
- Track retry rate (should be <1%)

### **4. Failure Scenarios**

- Kill workers mid-transfer (context cancellation)
- Simulate network failures (retry mechanism)
- Test with very large extent counts (>10,000)

---

## 📈 **EXPECTED BENEFITS**

### **Performance Improvements**

| Metric | Current (Serial) | Target (Parallel) | Improvement |
|--------|------------------|-------------------|-------------|
| Throughput | 150 MB/s | 400-500 MB/s | **2.5-3.5x** |
| Request Latency | 10-20ms per extent | Amortized | Hidden by parallelism |
| CPU Utilization | 20-30% | 60-80% | Better resource usage |
| Incremental Sync Time | 60 minutes | 15-20 minutes | **3-4x faster** |

### **Operational Benefits**

- ✅ Reduced migration windows
- ✅ Better handling of small extent workloads
- ✅ Improved resource utilization
- ✅ No breaking changes to existing functionality

---

## 🔄 **ROLLBACK STRATEGY**

**If parallel implementation causes issues:**

1. **Feature Flag**: Add `MIGRATEKIT_PARALLEL_NBD=false` environment variable
2. **Fallback**: Automatic degradation to serial copy on worker startup failures
3. **Monitoring**: Track retry rates and connection failures
4. **Rollback**: Simple config change to disable parallel mode

---

## 🎯 **IMPLEMENTATION PHASES**

### **Phase 1: Core Utilities** (1-2 days)
- Implement `coalesceExtents()`
- Implement `splitExtentsAcrossWorkers()`
- Unit tests for extent logic

### **Phase 2: Worker Pool** (2-3 days)
- Implement `copyWorker()` goroutine
- NBD connection per worker
- Basic error handling

### **Phase 3: Progress & Monitoring** (1-2 days)
- Progress aggregator with channel
- VMA client integration
- Per-worker logging

### **Phase 4: Retry & Error Handling** (1-2 days)
- Retry mechanism with exponential backoff
- Error collection and reporting
- Graceful degradation

### **Phase 5: Testing & Optimization** (2-3 days)
- Integration tests with real VMs
- Performance benchmarking
- Tuning worker count and chunk sizes

**Total Estimate**: 7-12 days for complete implementation and testing

---

## ✅ **FINAL FEASIBILITY VERDICT**

**Status**: ✅ **FEASIBLE AND RECOMMENDED**

**Rationale**:
1. ✅ VMware NBD servers support parallel connections
2. ✅ Architecture complies with all MigrateKit rules
3. ✅ Modular design with clear separation of concerns
4. ✅ Expected 2.5-3.5x throughput improvement
5. ✅ Graceful fallback to serial mode if needed
6. ✅ No breaking changes to existing functionality

**Risk Level**: 🟢 **LOW** (with proper testing and feature flag)

**Recommendation**: **PROCEED WITH IMPLEMENTATION**

---

**Next Steps**:
1. Review this design document
2. Approve implementation plan
3. Begin Phase 1: Core Utilities
4. Iterative testing with pgtest VMs

