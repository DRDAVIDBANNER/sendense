# VM Context API Documentation

## Overview

The VM Context API provides comprehensive VM-centric data access for GUI integration, implementing the VM-Centric Architecture with minimal, powerful endpoints. This API follows the project's "tidy" design principle with just two endpoints that provide complete VM migration context.

## Endpoints

### 1. List All VM Contexts

**Endpoint:** `GET /api/v1/vm-contexts`

**Description:** Retrieves a summary list of all VM contexts with essential metadata for dashboard views.

**Authentication:** Required (Bearer token)

**Request:**
```bash
curl -H "Authorization: Bearer <token>" \
     "http://localhost:8082/api/v1/vm-contexts"
```

**Response:**
```json
{
  "count": 2,
  "vm_contexts": [
    {
      "context_id": "ctx-pgtest1-20250909-113839",
      "vm_name": "pgtest1",
      "vmware_vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
      "vm_path": "/DatabanxDC/vm/pgtest1",
      "vcenter_host": "quad-vcenter-01.quadris.local",
      "datacenter": "DatabanxDC",
      "current_status": "replicating",
      "current_job_id": "job-20250909-114850",
      "total_jobs_run": 2,
      "successful_jobs": 0,
      "failed_jobs": 0,
      "last_successful_job_id": null,
      "cpu_count": 2,
      "memory_mb": 8192,
      "os_type": "windows",
      "power_state": "poweredOn",
      "vm_tools_version": "11.3.5",
      "created_at": "2025-09-09T11:38:39+01:00",
      "updated_at": "2025-09-09T11:48:50+01:00",
      "first_job_at": "2025-09-09T11:38:39+01:00",
      "last_job_at": "2025-09-09T11:48:50+01:00",
      "last_status_change": "2025-09-09T11:48:50+01:00"
    }
  ]
}
```

**Status Codes:**
- `200 OK` - Success
- `401 Unauthorized` - Authentication required
- `500 Internal Server Error` - Database or system error

### 2. Get Detailed VM Context

**Endpoint:** `GET /api/v1/vm-contexts/{vm_name}`

**Description:** Retrieves complete VM context with current job details, historical data, disk information, and CBT tracking for detailed VM views.

**Authentication:** Required (Bearer token)

**Parameters:**
- `vm_name` (path) - The name of the VM (e.g., "pgtest1")

**Request:**
```bash
curl -H "Authorization: Bearer <token>" \
     "http://localhost:8082/api/v1/vm-contexts/pgtest1"
```

**Response Structure:**
```json
{
  "context": {
    "context_id": "ctx-pgtest1-20250909-113839",
    "vm_name": "pgtest1",
    "vmware_vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
    "vm_path": "/DatabanxDC/vm/pgtest1",
    "vcenter_host": "quad-vcenter-01.quadris.local",
    "datacenter": "DatabanxDC",
    "current_status": "replicating",
    "current_job_id": "job-20250909-114850",
    "total_jobs_run": 2,
    "successful_jobs": 0,
    "failed_jobs": 0,
    "last_successful_job_id": null,
    "cpu_count": 2,
    "memory_mb": 8192,
    "os_type": "windows",
    "power_state": "poweredOn",
    "vm_tools_version": "11.3.5",
    "created_at": "2025-09-09T11:38:39+01:00",
    "updated_at": "2025-09-09T11:48:50+01:00",
    "first_job_at": "2025-09-09T11:38:39+01:00",
    "last_job_at": "2025-09-09T11:48:50+01:00",
    "last_status_change": "2025-09-09T11:48:50+01:00"
  },
  "current_job": {
    "id": "job-20250909-114850",
    "vm_context_id": "ctx-pgtest1-20250909-113839",
    "source_vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
    "source_vm_name": "pgtest1",
    "status": "replicating",
    "replication_type": "initial",
    "progress_percent": 49.30555555555556,
    "current_operation": "Transferring Data",
    "bytes_transferred": 19058917376,
    "total_bytes": 38654705664,
    "transfer_speed_bps": 12013439,
    "error_message": "",
    "change_id": "",
    "previous_change_id": "",
    "nbd_export_name": "migration-vol-5dbff3d8-531a-4a80-9977-c5ae25b9c4ae",
    "vma_sync_type": "full",
    "vma_current_phase": "Transfer",
    "vma_throughput_mbps": 11.46,
    "vma_eta_seconds": 1631,
    "vma_last_poll_at": "2025-09-09T12:12:08+01:00",
    "created_at": "2025-09-09T11:48:50.709+01:00",
    "updated_at": "2025-09-09T12:12:08.204+01:00"
  },
  "job_history": [
    {
      "id": "job-20250909-113839",
      "status": "completed",
      "replication_type": "initial",
      "progress_percent": 100.0,
      "bytes_transferred": 38654705664,
      "total_bytes": 38654705664,
      "created_at": "2025-09-09T11:38:39.716+01:00",
      "completed_at": "2025-09-09T11:45:23.122+01:00"
    }
  ],
  "disks": [
    {
      "id": 573,
      "job_id": "job-20250909-114850",
      "vm_context_id": "ctx-pgtest1-20250909-113839",
      "disk_id": "disk-2000",
      "vmdk_path": "[vsanDatastore] 285ea568-64bc-07e9-4bc3-000af7864054/pgtest1.vmdk",
      "size_gb": 102,
      "datastore": "vsanDatastore",
      "unit_number": 0,
      "label": "Hard disk 1",
      "capacity_bytes": 109521666048,
      "provisioning_type": "thin",
      "ossea_volume_id": 160,
      "cpu_count": 2,
      "memory_mb": 8192,
      "os_type": "windows",
      "vm_tools_version": "",
      "network_config": "[{\"name\":\"\",\"type\":\"\",\"connected\":false,\"mac_address\":\"00:50:56:85:3c:59\",\"label\":\"Network adapter 1\",\"network_name\":\"VLAN 253 - QUADRIS_CLOUD-DMZ\",\"adapter_type\":\"vmxnet3\"}]",
      "display_name": "",
      "annotation": "",
      "power_state": "poweredOn",
      "vmware_uuid": "420570c7-f61f-a930-77c5-1e876786cb3c",
      "disk_change_id": "",
      "sync_status": "pending",
      "sync_progress_percent": 0,
      "bytes_synced": 0,
      "created_at": "2025-09-09T11:48:50.717+01:00",
      "updated_at": "2025-09-09T11:48:50.723+01:00"
    }
  ],
  "cbt_history": [
    {
      "id": 552,
      "job_id": "job-20250909-114850",
      "vm_context_id": "ctx-pgtest1-20250909-113839",
      "disk_id": "disk-2000",
      "change_id": "",
      "previous_change_id": "",
      "sync_type": "initial",
      "blocks_changed": 0,
      "bytes_transferred": 0,
      "sync_duration_seconds": 0,
      "sync_success": false,
      "created_at": "2025-09-09T11:48:50.734+01:00"
    }
  ]
}
```

**Status Codes:**
- `200 OK` - Success
- `401 Unauthorized` - Authentication required
- `404 Not Found` - VM context not found
- `500 Internal Server Error` - Database or system error

## Data Models

### VM Context

Core VM metadata and status tracking:

| Field | Type | Description |
|-------|------|-------------|
| `context_id` | string | Unique VM context identifier |
| `vm_name` | string | VM name (unique identifier) |
| `vmware_vm_id` | string | VMware vCenter VM UUID |
| `vm_path` | string | Full VMware inventory path |
| `vcenter_host` | string | vCenter server hostname |
| `datacenter` | string | VMware datacenter name |
| `current_status` | enum | Current VM status (discovered, replicating, ready_for_failover, etc.) |
| `current_job_id` | string | Active job ID (if any) |
| `total_jobs_run` | integer | Total number of jobs executed |
| `successful_jobs` | integer | Number of successful jobs |
| `failed_jobs` | integer | Number of failed jobs |
| `last_successful_job_id` | string | ID of last successful job |
| `cpu_count` | integer | VM CPU count |
| `memory_mb` | integer | VM memory in MB |
| `os_type` | string | Operating system type |
| `power_state` | string | VM power state |
| `vm_tools_version` | string | VMware Tools version |

### Current Job

Live job progress and details:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Job identifier |
| `status` | enum | Job status (pending, replicating, completed, failed) |
| `replication_type` | enum | Type of replication (initial, incremental) |
| `progress_percent` | float | Current progress percentage |
| `current_operation` | string | Current operation description |
| `bytes_transferred` | integer | Bytes transferred so far |
| `total_bytes` | integer | Total bytes to transfer |
| `transfer_speed_bps` | integer | Current transfer speed in bytes/second |
| `vma_throughput_mbps` | float | VMA reported throughput in Mbps |
| `vma_eta_seconds` | integer | Estimated time to completion |
| `nbd_export_name` | string | NBD export identifier |
| `change_id` | string | VMware change ID for incremental sync |

### VM Disks

Disk configuration and sync status:

| Field | Type | Description |
|-------|------|-------------|
| `disk_id` | string | VMware disk identifier |
| `vmdk_path` | string | VMDK file path in datastore |
| `size_gb` | integer | Disk size in GB |
| `capacity_bytes` | integer | Exact capacity in bytes |
| `provisioning_type` | string | Disk provisioning (thin, thick) |
| `ossea_volume_id` | integer | Associated OSSEA volume ID |
| `network_config` | string | JSON network configuration |
| `sync_status` | string | Disk sync status |
| `sync_progress_percent` | float | Disk-specific sync progress |

### CBT History

Change Block Tracking history:

| Field | Type | Description |
|-------|------|-------------|
| `change_id` | string | VMware change ID |
| `previous_change_id` | string | Previous change ID for incremental |
| `sync_type` | enum | Sync type (initial, incremental) |
| `blocks_changed` | integer | Number of changed blocks |
| `bytes_transferred` | integer | Bytes transferred for this sync |
| `sync_duration_seconds` | integer | Sync duration |
| `sync_success` | boolean | Whether sync was successful |

## Architecture Integration

### VM-Centric Design

These endpoints implement the VM-Centric Architecture where:
- All data is linked via `vm_context_id`
- VM context serves as the master record
- CASCADE DELETE ensures data consistency
- Single source of truth for VM migration state

### Database Integration

The API leverages the complete VM-Centric schema:
- `vm_replication_contexts` (master table)
- `replication_jobs` (job tracking)
- `vm_disks` (disk configuration)
- `ossea_volumes` (volume management)
- `device_mappings` (Volume Daemon integration)
- `nbd_exports` (NBD export tracking)
- `cbt_history` (change tracking)

### Performance Optimizations

- **Limited Queries**: Job history limited to 10 records, CBT history to 20
- **Indexed Lookups**: Efficient queries using database indexes
- **Single Transaction**: Comprehensive data in single API call
- **Minimal Endpoints**: Two endpoints provide complete functionality

## Usage Examples

### Dashboard Integration

```javascript
// Get all VM contexts for dashboard
fetch('/api/v1/vm-contexts')
  .then(response => response.json())
  .then(data => {
    data.vm_contexts.forEach(vm => {
      console.log(`${vm.vm_name}: ${vm.current_status} (${vm.total_jobs_run} jobs)`);
    });
  });
```

### Detailed VM View

```javascript
// Get detailed VM context for specific VM
fetch('/api/v1/vm-contexts/pgtest1')
  .then(response => response.json())
  .then(data => {
    const { context, current_job, job_history, disks, cbt_history } = data;
    
    if (current_job) {
      console.log(`Current Job: ${current_job.progress_percent}% complete`);
      console.log(`Transfer Speed: ${current_job.vma_throughput_mbps} Mbps`);
      console.log(`ETA: ${current_job.vma_eta_seconds} seconds`);
    }
    
    console.log(`Job History: ${job_history.length} previous jobs`);
    console.log(`Disks: ${disks.length} disk(s) configured`);
    console.log(`CBT Records: ${cbt_history.length} change tracking entries`);
  });
```

### Real-time Progress Monitoring

```javascript
// Poll for live progress updates
const pollProgress = async (vmName) => {
  const response = await fetch(`/api/v1/vm-contexts/${vmName}`);
  const data = await response.json();
  
  if (data.current_job) {
    const job = data.current_job;
    updateProgressBar(job.progress_percent);
    updateTransferRate(job.vma_throughput_mbps);
    updateETA(job.vma_eta_seconds);
    
    if (job.status === 'replicating') {
      setTimeout(() => pollProgress(vmName), 5000); // Poll every 5 seconds
    }
  }
};
```

## Error Handling

### Common Error Responses

**404 Not Found:**
```json
{
  "error": "VM context not found for: nonexistent-vm",
  "timestamp": "2025-09-09T12:15:30Z"
}
```

**401 Unauthorized:**
```json
{
  "error": "Authentication required",
  "timestamp": "2025-09-09T12:15:30Z"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Failed to retrieve VM context",
  "timestamp": "2025-09-09T12:15:30Z"
}
```

## Production Deployment

**Current Version:** `oma-api-v2.18.0-vm-context-endpoints`

**Service:** `oma-api.service`

**Port:** 8082 (internal), 443 (external via tunnel)

**Authentication:** Bearer token required for all endpoints

**Testing:**
```bash
# Test list endpoint
curl -H "Authorization: Bearer <token>" "http://localhost:8082/api/v1/vm-contexts"

# Test detailed endpoint
curl -H "Authorization: Bearer <token>" "http://localhost:8082/api/v1/vm-contexts/pgtest1"
```

## Future Enhancements

Potential future additions (following minimal endpoint principle):
- WebSocket support for real-time progress streaming
- Filtering/pagination for large VM lists
- Bulk operations via single endpoint
- Custom field selection for optimized responses

---

**Note:** This API follows the project's "tidy" design principle with minimal, comprehensive endpoints that provide complete functionality without endpoint sprawl.
