# OMA Progress Poller Implementation

**Component**: `VMAProgressPoller` service in OMA API  
**Purpose**: Poll VMA Progress API and update OMA database with migration progress  
**Status**: Service running, Phase 1 fix ready for implementation

## 🏗️ **Architecture**

### **Service Overview**
```go
type VMAProgressPoller struct {
    vmaClient     *VMAProgressClient    // HTTP client for VMA API
    repository    *database.OSSEAConfigRepository
    pollInterval  time.Duration         // 5 seconds
    maxConcurrent int                   // 10 jobs max
    activeJobs    map[string]*PollingContext
}
```

### **Polling Context**
```go
type PollingContext struct {
    JobID             string
    StartedAt         time.Time
    LastPoll          time.Time
    ConsecutiveErrors int
    MaxErrors         int         // 5 consecutive failures → stop polling
    StopChan          chan struct{}
}
```

## 🔄 **Current Implementation**

### **Polling Loop**
1. **Job Discovery**: Query `replication_jobs` WHERE `status IN ('queued', 'running', 'replicating')`
2. **Active Job Management**: Start/stop polling contexts for discovered jobs  
3. **Progress Retrieval**: HTTP GET to VMA API per active job
4. **Database Update**: UPDATE replication_jobs with progress data
5. **Error Handling**: Exponential backoff on failures, stop after 5 consecutive errors

### **Database Query (Current)**
```sql
SELECT id, status 
FROM replication_jobs 
WHERE status IN ('queued', 'running', 'replicating') 
ORDER BY created_at DESC
```

## 🚨 **Current Issue: ID Mismatch**

### **Problem**
- **OMA Database**: Jobs stored with IDs like `job-20250905-162427`
- **VMA Progress API**: Progress stored with NBD export names like `migration-vol-c290646c-41ba-4d50-a31f-f497320ca0bd`
- **Result**: VMAProgressPoller polls VMA with job IDs → `404 job not found`

### **Evidence**
```bash
# This works (NBD export name)
curl http://localhost:9081/api/v1/progress/migration-vol-c290646c-41ba-4d50-a31f-f497320ca0bd
# Returns: {"job_id":"migration-vol-...", "percentage":2.17, ...}

# This fails (OMA job ID)  
curl http://localhost:9081/api/v1/progress/job-20250905-162427
# Returns: job not found
```

## 🛠️ **Phase 1 Fix: Dynamic NBD Export Name Construction**

### **New Method: getNBDExportNameForJob()**
```go
func (vpp *VMAProgressPoller) getNBDExportNameForJob(jobID string) ([]string, error) {
    query := `
        SELECT ov.volume_id 
        FROM replication_jobs rj
        JOIN vm_disks vd ON rj.id = vd.job_id  
        JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
        WHERE rj.id = ?`
    
    var volumeIDs []string
    err := vpp.repository.DB().Select(&volumeIDs, query, jobID)
    if err != nil {
        return nil, fmt.Errorf("failed to get volume IDs for job %s: %w", jobID, err)
    }
    
    var nbdExportNames []string
    for _, volumeID := range volumeIDs {
        nbdExportNames = append(nbdExportNames, fmt.Sprintf("migration-vol-%s", volumeID))
    }
    
    return nbdExportNames, nil
}
```

### **Updated Polling Logic**
```go
func (vpp *VMAProgressPoller) pollSingleJob(jobID string, pollingCtx *PollingContext) {
    logger := log.WithField("job_id", jobID)
    
    // Try NBD export names first (Phase 1 fix)
    nbdExportNames, err := vpp.getNBDExportNameForJob(jobID)
    if err == nil && len(nbdExportNames) > 0 {
        for _, nbdExportName := range nbdExportNames {
            progressData, err := vpp.vmaClient.GetProgress(nbdExportName)
            if err == nil {
                logger.WithField("nbd_export_name", nbdExportName).Debug("✅ Found progress via NBD export name")
                vpp.updateJobWithVMAData(jobID, progressData)
                return
            }
        }
        logger.Debug("⚠️ No progress found for any NBD export names, trying job ID")
    }
    
    // Fallback to traditional job ID (backward compatibility)
    progressData, err := vpp.vmaClient.GetProgress(jobID)
    if err != nil {
        vpp.handlePollingError(jobID, pollingCtx, err, logger)
        return
    }
    
    logger.Debug("✅ Found progress via job ID (legacy)")
    vpp.updateJobWithVMAData(jobID, progressData)
}
```

### **Multi-Disk Support**
- Single replication job may have multiple `vm_disks` entries
- Each disk gets its own NBD export name: `migration-vol-{volume_uuid}`  
- Polling tries all NBD export names for the job
- Returns progress from first successful response
- Aggregates progress across multiple disks if needed

## 📊 **Database Update Logic**

### **Progress Data Mapping**
```go
type VMAProgressResponse struct {
    JobID           string  `json:"job_id"`
    Status          string  `json:"status"`
    Phase           string  `json:"phase"`
    Percentage      float64 `json:"percentage"`
    BytesTransferred int64  `json:"bytes_transferred"`
    TotalBytes      int64   `json:"total_bytes"`
    Throughput      struct {
        CurrentMBps float64 `json:"current_mbps"`
        AverageMBps float64 `json:"average_mbps"`
        PeakMBps    float64 `json:"peak_mbps"`
    } `json:"throughput"`
}
```

### **Database Update Query**
```sql
UPDATE replication_jobs SET
    progress_percent = ?,
    current_operation = ?,
    bytes_transferred = ?,
    total_bytes = ?,
    transfer_speed_bps = ?,
    updated_at = NOW()
WHERE id = ?
```

## 🧪 **Testing Strategy**

### **Unit Tests**
```go
func TestGetNBDExportNameForJob(t *testing.T) {
    // Test single disk job
    // Test multi-disk job  
    // Test job not found
    // Test database connection failure
}

func TestPollSingleJobWithNBDExportNames(t *testing.T) {
    // Test successful NBD export name resolution
    // Test NBD export name fallback to job ID
    // Test all export names failing
}
```

### **Integration Tests**
```bash
# Test with known job ID
curl -X POST http://localhost:8082/api/test/start-polling/job-20250905-162427

# Verify database updates
mysql -e "SELECT progress_percent FROM replication_jobs WHERE id='job-20250905-162427'"

# Check polling logs
sudo journalctl -u oma-api.service -f | grep "job-20250905-162427"
```

## 📈 **Performance Considerations**

### **Database Query Optimization**
- **Query Frequency**: Every 5 seconds per active job
- **Index Requirements**: `vm_disks(job_id)`, `ossea_volumes(id)`
- **Connection Pooling**: Reuse database connections for polling queries
- **Query Caching**: Consider caching job ID → NBD export name mappings

### **Memory Management**
- **Active Jobs Map**: Limited to replication jobs in active states
- **Polling Context**: Cleanup on job completion or consecutive errors
- **HTTP Client**: Connection reuse with Keep-Alive

### **Error Recovery**
- **Consecutive Error Limit**: 5 failures → stop polling job
- **Exponential Backoff**: 5s → 10s → 20s → 40s → stop
- **Database Reconnection**: Automatic retry on connection loss
- **VMA API Timeout**: 10-second timeout per request

## 🔧 **Configuration**

### **Environment Variables**
```bash
VMA_PROGRESS_POLL_INTERVAL=5s      # Polling frequency
VMA_PROGRESS_MAX_CONCURRENT=10     # Max concurrent job polling
VMA_PROGRESS_MAX_ERRORS=5          # Stop polling after N consecutive errors
VMA_PROGRESS_HTTP_TIMEOUT=10s      # HTTP request timeout
VMA_PROGRESS_CLIENT_URL=http://localhost:9081  # VMA API base URL
```

### **Database Connection**
```go
// Uses existing OMA database connection
repository := database.NewOSSEAConfigRepository(db)
```

## 🚨 **Troubleshooting**

### **Common Issues**
1. **"Job not found" errors**: Check NBD export name construction
2. **Database update failures**: Verify job ID exists in replication_jobs
3. **High error rates**: Check VMA API availability and network connectivity
4. **Memory leaks**: Monitor active jobs map cleanup

### **Debug Logging**
```bash
# Enable debug logging
export LOG_LEVEL=debug

# Monitor specific job polling
sudo journalctl -u oma-api.service -f | grep "job-20250905-162427"

# Check NBD export name resolution
curl -X GET "http://localhost:8082/api/debug/nbd-export-names/job-20250905-162427"
```

### **Metrics**
- **Active Jobs Count**: Number of jobs being polled
- **Success Rate**: Percentage of successful progress retrievals
- **Average Response Time**: VMA API response latency
- **Database Update Rate**: Updates per second
- **Error Distribution**: Types and frequency of errors

---

**Next**: Implement Phase 1 fix and validate end-to-end progress tracking.
