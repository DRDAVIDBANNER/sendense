# Replication Progress Tracking Architecture

**Last Updated**: 2025-09-04  
**Version**: v1.0  
**Status**: IMPLEMENTED  

---

## 📋 **OVERVIEW**

This document describes the robust replication progress tracking system that provides real-time progress monitoring for VMware to OSSEA migrations via VMA → OMA communication over TLS tunnel.

### **Key Features**
- Real VMware API integration for planned bytes calculation
- 2-second polling with intelligent throttling
- 5-minute timeout with automatic job failure
- JobLog integration with correlation IDs
- Database state machine enforcement
- CBT history tracking for audit trails

---

## 🏗️ **ARCHITECTURE DIAGRAM**

```
┌─────────────────┐    TLS Tunnel     ┌─────────────────┐
│      VMA        │    Port 443       │      OMA        │
│                 │ ◄────────────────► │                 │
│ ┌─────────────┐ │                   │ ┌─────────────┐ │
│ │ Progress    │ │   GET /progress/  │ │ Progress    │ │
│ │ Service     │ │     {jobId}       │ │ Poller      │ │
│ │             │ │                   │ │             │ │
│ │ - VMware    │ │                   │ │ - 2s polls  │ │
│ │   API calls │ │                   │ │ - Timeout   │ │
│ │ - CBT calc  │ │                   │ │   detection │ │
│ │ - Real-time │ │                   │ │ - HTTP      │ │
│ │   tracking  │ │                   │ │   client    │ │
│ └─────────────┘ │                   │ └─────────────┘ │
│                 │                   │        │        │
│ ┌─────────────┐ │                   │ ┌─────────────┐ │
│ │ Progress    │ │                   │ │ Progress    │ │
│ │ Handler     │ │                   │ │ Updater     │ │
│ │             │ │                   │ │             │ │
│ │ - HTTP      │ │                   │ │ - Database  │ │
│ │   endpoint  │ │                   │ │   updates   │ │
│ │ - JSON      │ │                   │ │ - Throttling│ │
│ │   response  │ │                   │ │ - JobLog    │ │
│ └─────────────┘ │                   │ └─────────────┘ │
└─────────────────┘                   └─────────────────┘
```

---

## 🔧 **COMPONENT SPECIFICATIONS**

### **VMA Progress Endpoint**

**Path**: `GET /api/v1/progress/{jobId}`

**Response Schema**:
```json
{
  "job_id": "<string>",
  "stage": "Transfer",
  "status": "Streaming",
  "started_at": "2025-09-04T12:00:00Z",
  "updated_at": "2025-09-04T12:00:10Z",
  "aggregate": {
    "total_bytes": 1234567890,
    "bytes_transferred": 456789012,
    "throughput_bps": 98765432,
    "percent": 37.0
  },
  "cbt": {
    "type": "incremental",
    "previous_change_id": "123-abc",
    "change_id": "456-def"
  },
  "nbd": {
    "exports": [
      {
        "name": "job-123-disk-0",
        "device": "/dev/nbdX",
        "connected": true,
        "started_at": "2025-09-04T12:00:02Z"
      }
    ]
  },
  "disks": [
    {
      "id": "disk-0",
      "label": "Hard disk 1",
      "planned_bytes": 987654321,
      "bytes_transferred": 345678901,
      "throughput_bps": 54321098,
      "percent": 35.0,
      "status": "Streaming"
    }
  ]
}
```

### **OMA Progress Poller**

**Polling Frequency**: Every 2 seconds  
**Timeout Rule**: >5 minutes no-contact → job failure  
**Failure Handling**: Max 3 consecutive failures → reduce frequency  

**Features**:
- Cancellable per-job polling contexts
- Automatic job completion detection
- Communication timeout enforcement
- JobLog integration for correlation

### **OMA Progress Updater**

**Throttling Rules**:
- Write if progress changed ≥1%
- Write if ≥2 seconds since last write
- Write if current_operation changed
- Write if status changed

**Database Updates**:
- `replication_jobs`: status, current_operation, progress_percent, bytes_transferred, total_bytes, transfer_speed_bps
- `vm_disks`: sync_status, sync_progress_percent, bytes_synced, disk_change_id (on completion)
- `cbt_history`: Complete audit trail per disk with change IDs, sync type, duration, success status

---

## 🔄 **STATE MACHINE**

### **Job-Level Progression**
```
replication_jobs.current_operation:
Discover → EnableCBT → QueryCBT → Snapshot → PrepareVolumes → StartExports → Transfer → Finalize → PersistChangeIDs

replication_jobs.status:
Queued | Preparing | Snapshotting | Streaming | Finalizing | Succeeded | Failed
```

### **Disk-Level Progression**
```
vm_disks.sync_status:
Queued | Snapshotting | Streaming | Completed | Failed
```

### **Status Mapping**
| VMA Status | DB Status | Description |
|------------|-----------|-------------|
| Queued | pending | Waiting to start |
| Preparing | preparing | Initial setup |
| Snapshotting | snapshotting | VM snapshot creation |
| Streaming | streaming | Data transfer active |
| Finalizing | finalizing | Cleanup and finalization |
| Succeeded | completed | Successfully completed |
| Failed | failed | Failed with error |

---

## 📊 **DATABASE INTEGRATION**

### **Tables Used**

**replication_jobs**:
- `status` VARCHAR(50) - Job status (pending, streaming, completed, failed, etc.)
- `current_operation` VARCHAR(255) - Current stage (Discover, Transfer, etc.)
- `progress_percent` DECIMAL(5,2) - Overall completion percentage
- `bytes_transferred` BIGINT - Total bytes transferred
- `total_bytes` BIGINT - Total bytes to transfer
- `transfer_speed_bps` BIGINT - Current transfer speed
- `error_message` TEXT - Error details on failure
- `updated_at` TIMESTAMP - Last progress update

**vm_disks**:
- `sync_status` VARCHAR(50) - Disk sync status (pending, syncing, completed, failed)
- `sync_progress_percent` DECIMAL(5,2) - Disk completion percentage
- `bytes_synced` BIGINT - Bytes transferred for this disk
- `disk_change_id` VARCHAR(255) - CBT ChangeID (set on completion)

**cbt_history**:
- `job_id` VARCHAR(255) - Reference to replication job
- `disk_id` VARCHAR(255) - Disk identifier
- `change_id` VARCHAR(255) - VMware CBT ChangeID
- `previous_change_id` VARCHAR(255) - Previous ChangeID for incrementals
- `sync_type` VARCHAR(50) - "full" or "incremental"
- `blocks_changed` INT - Number of changed blocks (if available)
- `bytes_transferred` BIGINT - Bytes transferred for this disk
- `sync_duration_seconds` INT - Time taken for sync
- `sync_success` BOOLEAN - Whether sync completed successfully

---

## ⚡ **PERFORMANCE OPTIMIZATIONS**

### **Throttled Database Writes**
- Prevents database overload with high-frequency polling
- Ensures critical updates (status/stage changes) are never missed
- Balances real-time visibility with system performance

### **Intelligent Failure Handling**
- Consecutive failure detection with backoff
- Communication timeout detection
- Graceful degradation under network issues

### **Efficient HTTP Communication**
- 10-second HTTP timeouts per request
- Connection reuse with persistent HTTP client
- Structured error handling with retry logic

---

## 🔍 **MONITORING AND DEBUGGING**

### **JobLog Integration**
Every replication job automatically gets:
- Correlation IDs for end-to-end tracing
- Stage transition logging
- Progress milestone tracking
- Panic recovery and error context
- Database audit trail

### **Key Log Messages**
```
🔄 Starting progress polling for replication job
⏹️ Stopped progress polling for job
✅ Job completed, stopping progress polling
🚨 VMA communication timeout exceeded
```

### **Monitoring Queries**
```sql
-- Active replication jobs
SELECT id, status, current_operation, progress_percent, updated_at 
FROM replication_jobs 
WHERE status IN ('pending', 'streaming', 'preparing');

-- Recent CBT history
SELECT job_id, disk_id, sync_type, bytes_transferred, sync_success, created_at
FROM cbt_history 
ORDER BY created_at DESC 
LIMIT 10;

-- Progress tracking health
SELECT 
  COUNT(*) as total_jobs,
  SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
  SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
FROM replication_jobs 
WHERE created_at > DATE_SUB(NOW(), INTERVAL 24 HOUR);
```

---

## 🛡️ **ERROR HANDLING**

### **Communication Failures**
- HTTP timeouts: 10-second per-request timeout
- Network issues: Graceful retry with exponential backoff
- VMA unavailable: Continue polling until timeout threshold

### **Database Failures**
- Transaction rollback on partial updates
- Non-fatal errors logged but don't stop polling
- Disk update failures don't affect job-level updates

### **Timeout Scenarios**
- >5 minutes no successful VMA contact → job marked as failed
- JobLog tracking ends with failure status
- Automatic cleanup of polling resources

---

## 🚀 **DEPLOYMENT CONSIDERATIONS**

### **Network Requirements**
- VMA accessible via TLS tunnel on port 443
- HTTP client configured for tunnel proxy if needed
- 10-second timeout appropriate for tunnel latency

### **Database Performance**
- Throttling prevents excessive write load
- Indexes on frequently queried columns (job_id, status, updated_at)
- CBT history table can grow large over time - consider archival strategy

### **Resource Management**
- One goroutine per active replication job
- Automatic cleanup on job completion
- Memory usage scales linearly with concurrent jobs

---

## 📈 **METRICS AND ANALYTICS**

### **Available Metrics**
- Real-time throughput (bytes/second)
- Progress percentage accuracy
- Stage transition timing
- Communication timeout incidents
- Database write frequency

### **Performance Baselines**
- Target: 2-second polling with <500ms response time
- Database writes: <100ms for throttled updates
- Memory: <10MB per 100 concurrent jobs

---

## 🔄 **FUTURE ENHANCEMENTS**

### **Planned Improvements**
1. **WebSocket Support**: Real-time push updates instead of polling
2. **Progress Prediction**: ETA calculation based on historical data
3. **Bandwidth Throttling**: QoS integration with network policies
4. **Advanced Analytics**: Machine learning for failure prediction

### **Integration Opportunities**
1. **Grafana Dashboards**: Real-time progress visualization
2. **AlertManager**: Automated failure notifications
3. **Prometheus Metrics**: Time-series performance data
4. **Elastic Stack**: Advanced log analysis and search

---

**🎯 Architecture Summary**: This robust progress tracking system provides production-ready replication monitoring with real-time updates, intelligent throttling, comprehensive error handling, and full audit trails while maintaining the strict architectural constraints of port 443 tunnel-only communication.
