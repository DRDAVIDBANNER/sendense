# Progress Tracking API Reference

**Version**: v1.8.1 (VMA), v2.6.0 (OMA)  
**Protocol**: HTTP/1.1 with JSON payloads  
**Authentication**: None (internal network communication)

## 🔗 **VMA Progress API**

### **Base URL**
```
http://localhost:8081/api/v1/progress/
```

### **POST /api/v1/progress/{jobId}/update**
**Purpose**: Receive progress updates from migratekit  
**Used by**: migratekit libnbd callbacks

#### **Request**
```http
POST /api/v1/progress/migration-vol-c290646c-41ba-4d50-a31f-f497320ca0bd/update
Content-Type: application/json

{
    "stage": "Transfer",
    "status": "Queued", 
    "bytes_transferred": 933232640,
    "total_bytes": 42949672960,
    "throughput_bps": 0,
    "percent": 2.1728515625,
    "disk_id": "disk-2000"
}
```

#### **Response**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
    "status": "success",
    "message": "Progress updated successfully"
}
```

#### **Error Responses**
```http
HTTP/1.1 404 Not Found
{
    "error": "job not found",
    "job_id": "migration-vol-invalid-uuid"
}

HTTP/1.1 400 Bad Request  
{
    "error": "invalid progress data",
    "details": "bytes_transferred must be >= 0"
}
```

### **GET /api/v1/progress/{jobId}**
**Purpose**: Retrieve current progress for OMA polling  
**Used by**: OMA VMAProgressPoller

#### **Request**
```http
GET /api/v1/progress/migration-vol-c290646c-41ba-4d50-a31f-f497320ca0bd
```

#### **Response**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
    "job_id": "migration-vol-c290646c-41ba-4d50-a31f-f497320ca0bd",
    "status": "Queued",
    "sync_type": "full",
    "phase": "Transfer", 
    "percentage": 2.1728515625,
    "current_operation": "Transfer",
    "bytes_transferred": 933232640,
    "total_bytes": 42949672960,
    "throughput": {
        "current_mbps": 0,
        "average_mbps": 0,
        "peak_mbps": 0,
        "last_update": "2025-09-05T15:26:15Z"
    },
    "timing": {
        "start_time": "2025-09-05T15:24:43Z",
        "last_update": "2025-09-05T15:26:15Z", 
        "elapsed_ms": 91984,
        "phase_start": "2025-09-05T15:24:43Z",
        "phase_elapsed_ms": 91984,
        "eta_seconds": 0
    },
    "vm_info": {
        "name": "unknown",
        "path": "unknown", 
        "disk_size_gb": 40,
        "disk_size_bytes": 42949672960,
        "cbt_enabled": false
    },
    "phases": [
        {
            "name": "Transfer",
            "status": "Queued",
            "start_time": "2025-09-05T15:24:43Z",
            "end_time": "",
            "duration_ms": 91984
        }
    ],
    "errors": [],
    "last_error": null
}
```

#### **Error Responses**
```http
HTTP/1.1 404 Not Found
{
    "error": "job not found",
    "job_id": "migration-vol-invalid-uuid"
}
```

## 🔗 **OMA Progress API** 

### **Base URL**
```
http://localhost:8082/api/v1/
```

### **Internal Endpoints** (VMAProgressPoller)

The OMA API doesn't expose external progress endpoints but uses internal services:

#### **VMAProgressPoller Service**
```go
// Service initialization in handlers.go
vmaProgressClient := services.NewVMAProgressClient("http://localhost:9081")
vmaProgressPoller := services.NewVMAProgressPoller(vmaProgressClient, repo)

// Start background polling
ctx := context.Background()
if err := vmaProgressPoller.Start(ctx); err != nil {
    log.WithError(err).Warn("Failed to start VMA progress poller")
}
```

#### **Database Updates**
```sql
-- Progress data flows to these tables
UPDATE replication_jobs SET
    progress_percent = 2.17,
    current_operation = 'Transfer',
    bytes_transferred = 933232640,
    total_bytes = 42949672960, 
    transfer_speed_bps = 0,
    updated_at = NOW()
WHERE id = 'job-20250905-162427'
```

## 📊 **Data Models**

### **VMA Progress Update (POST)**
```go
type VMAProgressUpdate struct {
    Stage            string  `json:"stage"`             // Transfer, Discovery, etc.
    Status           string  `json:"status,omitempty"`  // Queued, Running, Completed
    BytesTransferred int64   `json:"bytes_transferred"` // Actual bytes transferred
    TotalBytes       int64   `json:"total_bytes,omitempty"` // Total volume size
    ThroughputBPS    int64   `json:"throughput_bps"`    // Current throughput (bytes/sec)
    Percent          float64 `json:"percent,omitempty"` // Percentage complete
    DiskID           string  `json:"disk_id,omitempty"` // VM disk identifier
}
```

### **VMA Progress Response (GET)**
```go
type VMAProgressResponse struct {
    JobID            string `json:"job_id"`
    Status           string `json:"status"` 
    SyncType         string `json:"sync_type"`
    Phase            string `json:"phase"`
    Percentage       float64 `json:"percentage"`
    CurrentOperation string `json:"current_operation"`
    BytesTransferred int64  `json:"bytes_transferred"`
    TotalBytes       int64  `json:"total_bytes"`
    
    Throughput struct {
        CurrentMBps float64 `json:"current_mbps"`
        AverageMBps float64 `json:"average_mbps"`
        PeakMBps    float64 `json:"peak_mbps"`
        LastUpdate  string  `json:"last_update"`
    } `json:"throughput"`
    
    Timing struct {
        StartTime       string `json:"start_time"`
        LastUpdate      string `json:"last_update"`
        ElapsedMs       int64  `json:"elapsed_ms"`
        PhaseStart      string `json:"phase_start"`
        PhaseElapsedMs  int64  `json:"phase_elapsed_ms"`
        ETASeconds      int64  `json:"eta_seconds"`
    } `json:"timing"`
    
    VMInfo struct {
        Name           string `json:"name"`
        Path           string `json:"path"`
        DiskSizeGB     int64  `json:"disk_size_gb"`
        DiskSizeBytes  int64  `json:"disk_size_bytes"`
        CBTEnabled     bool   `json:"cbt_enabled"`
    } `json:"vm_info"`
    
    Phases []struct {
        Name        string `json:"name"`
        Status      string `json:"status"`
        StartTime   string `json:"start_time"`
        EndTime     string `json:"end_time"`
        DurationMs  int64  `json:"duration_ms"`
    } `json:"phases"`
    
    Errors    []string    `json:"errors"`
    LastError interface{} `json:"last_error"`
}
```

## 🔄 **Progress Stages**

### **Stage Lifecycle**
1. **StageDiscover**: Initial VM discovery and validation
2. **StageTransfer**: Data transfer via NBD/libnbd 
3. **StageComplete**: Transfer completed successfully

### **Status Values**
- **StatusQueued**: Job queued for processing
- **StatusRunning**: Active data transfer in progress
- **StatusCompleted**: All data transferred successfully  
- **StatusFailed**: Error occurred during transfer

## 🚨 **Error Handling**

### **Error Categories**
1. **Validation Errors** (400): Invalid request data
2. **Not Found Errors** (404): Job/progress not found
3. **Server Errors** (500): Internal VMA/OMA failures

### **Error Response Format**
```json
{
    "error": "descriptive error message",
    "job_id": "problematic-job-id", 
    "details": "additional error context",
    "timestamp": "2025-09-05T15:26:15Z"
}
```

### **Retry Logic**
- **migratekit**: Continue on progress update failures (non-blocking)
- **OMA Poller**: Exponential backoff: 5s → 10s → 20s → 40s → stop
- **Database**: Automatic reconnection on connection loss

## 🧪 **Testing Examples**

### **Test Progress Update**
```bash
# Send progress update to VMA
curl -X POST http://localhost:8081/api/v1/progress/migration-vol-test-uuid/update \
  -H "Content-Type: application/json" \
  -d '{
    "stage": "Transfer",
    "bytes_transferred": 1073741824,
    "total_bytes": 10737418240,
    "throughput_bps": 104857600,
    "percent": 10.0
  }'
```

### **Test Progress Retrieval**
```bash
# Get progress from VMA  
curl http://localhost:8081/api/v1/progress/migration-vol-test-uuid | jq

# Check OMA database updates
mysql -u oma_user -p migratekit_oma \
  -e "SELECT progress_percent, bytes_transferred FROM replication_jobs WHERE id='job-test-123'"
```

### **Test Job Discovery**
```bash
# Check active jobs being polled
sudo journalctl -u oma-api.service --since="1 minute ago" | grep -E "Started.*polling|Stopped.*polling"
```

## 📈 **Performance**

### **Throughput Calculations**
- **Current Throughput**: Bytes transferred since last update / time elapsed
- **Average Throughput**: Total bytes transferred / total elapsed time
- **Peak Throughput**: Maximum throughput observed during transfer

### **Update Frequency**
- **migratekit → VMA**: Every 10MB transferred (configurable)
- **OMA Poller → VMA**: Every 5 seconds per active job
- **Database Updates**: On each successful progress retrieval

### **Memory Usage**
- **VMA Progress Storage**: In-memory map, auto-cleanup on completion
- **OMA Polling Context**: Per-job state tracking with error limits
- **HTTP Connections**: Keep-alive connections with connection pooling

---

**Implementation Status**: VMA API complete, OMA Poller Phase 1 ready for implementation.
