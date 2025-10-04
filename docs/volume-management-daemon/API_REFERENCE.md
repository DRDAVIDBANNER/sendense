# Volume Management Daemon - API Reference

**Complete REST API documentation with examples**

## Base Information

- **Base URL**: `http://localhost:8090/api/v1`
- **Content-Type**: `application/json`
- **Authentication**: None (internal service)

---

## Volume Operations

### Create Volume

Creates a new volume in CloudStack with real-time tracking.

**Endpoint**: `POST /volumes`

**Request Body**:
```json
{
  "name": "string (required)",
  "size": "integer (required, bytes)",
  "disk_offering_id": "string (required, UUID)",
  "zone_id": "string (required, UUID)",
  "metadata": "object (optional)"
}
```

**Example Request**:
```bash
curl -X POST http://localhost:8090/api/v1/volumes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "migration-volume-001",
    "size": 5368709120,
    "disk_offering_id": "c813c642-d946-49e1-9289-c616dd70206a",
    "zone_id": "057e86db-c726-4d8c-ab1f-75c5f55d1881",
    "metadata": {
      "purpose": "vm_migration",
      "vm_name": "test-vm-01"
    }
  }'
```

**Response**: `201 Created`
```json
{
  "id": "op-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "type": "create",
  "status": "pending",
  "volume_id": "",
  "vm_id": null,
  "request": {
    "name": "migration-volume-001",
    "size": 5368709120,
    "disk_offering_id": "c813c642-d946-49e1-9289-c616dd70206a",
    "zone_id": "057e86db-c726-4d8c-ab1f-75c5f55d1881",
    "metadata": {
      "purpose": "vm_migration",
      "vm_name": "test-vm-01"
    }
  },
  "created_at": "2025-08-19T20:30:00Z",
  "updated_at": "2025-08-19T20:30:00Z"
}
```

**Error Responses**:
- `400 Bad Request`: Invalid request parameters
- `500 Internal Server Error`: CloudStack API error

---

### Attach Volume

Attaches a volume to a VM with automatic device correlation.

**Endpoint**: `POST /volumes/{volume_id}/attach`

**Path Parameters**:
- `volume_id`: CloudStack volume UUID

**Request Body**:
```json
{
  "vm_id": "string (required, UUID)"
}
```

**Example Request**:
```bash
curl -X POST http://localhost:8090/api/v1/volumes/vol-12345678-1234-1234-1234-123456789012/attach \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "vm-87654321-4321-4321-4321-210987654321"
  }'
```

**Response**: `201 Created`
```json
{
  "id": "op-b2c3d4e5-f6g7-8901-bcde-f12345678901",
  "type": "attach", 
  "status": "pending",
  "volume_id": "vol-12345678-1234-1234-1234-123456789012",
  "vm_id": "vm-87654321-4321-4321-4321-210987654321",
  "request": {
    "volume_id": "vol-12345678-1234-1234-1234-123456789012",
    "vm_id": "vm-87654321-4321-4321-4321-210987654321"
  },
  "created_at": "2025-08-19T20:31:00Z",
  "updated_at": "2025-08-19T20:31:00Z"
}
```

---

### Attach Volume as Root Disk

Attach a CloudStack volume to a VM as the root disk (device ID 0). This is specifically used for test failover scenarios where the test VM needs to boot from the attached volume.

**Endpoint**: `POST /volumes/{volume_id}/attach-root`

**Path Parameters**:
- `volume_id`: CloudStack volume UUID

**Request Body**:
```json
{
  "vm_id": "string (required, UUID)"
}
```

**Example Request**:
```bash
curl -X POST http://localhost:8090/api/v1/volumes/vol-12345678-1234-1234-1234-123456789012/attach-root \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "vm-87654321-4321-4321-4321-210987654321"
  }'
```

**Response**: `201 Created`
```json
{
  "id": "op-c3d4e5f6-g7h8-9012-cdef-g12345678901",
  "type": "attach",
  "status": "pending",
  "volume_id": "vol-12345678-1234-1234-1234-123456789012",
  "vm_id": "vm-87654321-4321-4321-4321-210987654321",
  "request": {
    "volume_id": "vol-12345678-1234-1234-1234-123456789012",
    "vm_id": "vm-87654321-4321-4321-4321-210987654321",
    "device_id": 0,
    "attach_as": "root"
  },
  "created_at": "2025-08-20T10:00:00Z",
  "updated_at": "2025-08-20T10:00:00Z"
}
```

**Key Differences from Regular Attach**:
- Volume attached as **device ID 0** (root disk)
- Test VM can boot from the attached volume
- Used specifically for test failover scenarios
- CloudStack `SetDeviceid(0)` ensures proper root disk assignment

---

### Detach Volume

Detaches a volume from its VM and updates device mappings.

**Endpoint**: `POST /volumes/{volume_id}/detach`

**Path Parameters**:
- `volume_id`: CloudStack volume UUID

**Example Request**:
```bash
curl -X POST http://localhost:8090/api/v1/volumes/vol-12345678-1234-1234-1234-123456789012/detach
```

**Response**: `201 Created`
```json
{
  "id": "op-c3d4e5f6-g7h8-9012-cdef-123456789012",
  "type": "detach",
  "status": "pending", 
  "volume_id": "vol-12345678-1234-1234-1234-123456789012",
  "request": {
    "volume_id": "vol-12345678-1234-1234-1234-123456789012"
  },
  "created_at": "2025-08-19T20:32:00Z",
  "updated_at": "2025-08-19T20:32:00Z"
}
```

---

### Delete Volume

Deletes a volume from CloudStack and cleans up mappings.

**Endpoint**: `DELETE /volumes/{volume_id}`

**Path Parameters**:
- `volume_id`: CloudStack volume UUID

**Example Request**:
```bash
curl -X DELETE http://localhost:8090/api/v1/volumes/vol-12345678-1234-1234-1234-123456789012
```

**Response**: `201 Created`
```json
{
  "id": "op-d4e5f6g7-h8i9-0123-defg-234567890123",
  "type": "delete",
  "status": "pending",
  "volume_id": "vol-12345678-1234-1234-1234-123456789012",
  "request": {
    "volume_id": "vol-12345678-1234-1234-1234-123456789012"
  },
  "created_at": "2025-08-19T20:33:00Z",
  "updated_at": "2025-08-19T20:33:00Z"
}
```

---

## Status & Information

### Get Volume Status

Retrieves current status and device information for a volume.

**Endpoint**: `GET /volumes/{volume_id}`

**Example Request**:
```bash
curl http://localhost:8090/api/v1/volumes/vol-12345678-1234-1234-1234-123456789012
```

**Response**: `200 OK`
```json
{
  "volume_id": "vol-12345678-1234-1234-1234-123456789012",
  "vm_id": "vm-87654321-4321-4321-4321-210987654321",
  "device_path": "/dev/vdb",
  "state": "Ready",
  "size": 5368709120,
  "cloudstack_data": {
    "id": "vol-12345678-1234-1234-1234-123456789012",
    "name": "migration-volume-001",
    "state": "Ready",
    "type": "DATADISK",
    "zonename": "OSSEA-Zone",
    "attached": "2025-08-19T20:31:15Z",
    "deviceid": 2
  },
  "linux_data": {
    "device_path": "/dev/vdb",
    "size": 5368717312,
    "controller": "virtio4"
  },
  "last_operation": {
    "id": "op-b2c3d4e5-f6g7-8901-bcde-f12345678901",
    "type": "attach",
    "status": "completed"
  }
}
```

---

### Get Device Mapping

Returns device mapping information for a volume.

**Endpoint**: `GET /volumes/{volume_id}/device`

**Example Request**:
```bash
curl http://localhost:8090/api/v1/volumes/vol-12345678-1234-1234-1234-123456789012/device
```

**Response**: `200 OK`
```json
{
  "id": "mapping-1629834567890",
  "volume_id": "vol-12345678-1234-1234-1234-123456789012",
  "vm_id": "vm-87654321-4321-4321-4321-210987654321",
  "device_path": "/dev/vdb",
  "cloudstack_state": "attached",
  "linux_state": "detected",
  "size": 5368717312,
  "last_sync": "2025-08-19T20:31:16Z",
  "created_at": "2025-08-19T20:31:16Z",
  "updated_at": "2025-08-19T20:31:16Z"
}
```

---

### Get Volume for Device

Returns volume information for a specific device path.

**Endpoint**: `GET /devices/{device_path}/volume`

**Path Parameters**:
- `device_path`: Linux device path (encode as needed, e.g., `dev%2Fvdb`)

**Example Request**:
```bash
curl http://localhost:8090/api/v1/devices/dev%2Fvdb/volume
```

**Response**: `200 OK`
```json
{
  "id": "mapping-1629834567890",
  "volume_id": "vol-12345678-1234-1234-1234-123456789012",
  "vm_id": "vm-87654321-4321-4321-4321-210987654321", 
  "device_path": "/dev/vdb",
  "cloudstack_state": "attached",
  "linux_state": "detected",
  "size": 5368717312,
  "last_sync": "2025-08-19T20:31:16Z"
}
```

---

### List VM Volumes

Lists all volumes attached to a specific VM.

**Endpoint**: `GET /vms/{vm_id}/volumes`

**Example Request**:
```bash
curl http://localhost:8090/api/v1/vms/vm-87654321-4321-4321-4321-210987654321/volumes
```

**Response**: `200 OK`
```json
[
  {
    "volume_id": "vol-12345678-1234-1234-1234-123456789012",
    "vm_id": "vm-87654321-4321-4321-4321-210987654321",
    "device_path": "/dev/vdb",
    "state": "Ready",
    "size": 5368709120,
    "last_operation": {
      "type": "attach",
      "status": "completed"
    }
  },
  {
    "volume_id": "vol-23456789-2345-2345-2345-234567890123",
    "vm_id": "vm-87654321-4321-4321-4321-210987654321",
    "device_path": "/dev/vdc",
    "state": "Ready", 
    "size": 10737418240,
    "last_operation": {
      "type": "attach",
      "status": "completed"
    }
  }
]
```

---

## Operation Tracking

### Get Operation Status

Retrieves detailed status for a specific operation.

**Endpoint**: `GET /operations/{operation_id}`

**Example Request**:
```bash
curl http://localhost:8090/api/v1/operations/op-b2c3d4e5-f6g7-8901-bcde-f12345678901
```

**Response**: `200 OK`
```json
{
  "id": "op-b2c3d4e5-f6g7-8901-bcde-f12345678901",
  "type": "attach",
  "status": "completed",
  "volume_id": "vol-12345678-1234-1234-1234-123456789012",
  "vm_id": "vm-87654321-4321-4321-4321-210987654321",
  "request": {
    "volume_id": "vol-12345678-1234-1234-1234-123456789012",
    "vm_id": "vm-87654321-4321-4321-4321-210987654321"
  },
  "response": {
    "volume_id": "vol-12345678-1234-1234-1234-123456789012",
    "device_path": "/dev/vdb",
    "vm_id": "vm-87654321-4321-4321-4321-210987654321",
    "message": "Volume attached successfully"
  },
  "created_at": "2025-08-19T20:31:00Z",
  "updated_at": "2025-08-19T20:31:16Z",
  "completed_at": "2025-08-19T20:31:16Z"
}
```

**Operation Status Values**:
- `pending`: Operation queued but not started
- `executing`: Operation in progress  
- `completed`: Operation finished successfully
- `failed`: Operation failed with error
- `cancelled`: Operation was cancelled

---

### List Operations

Lists operations with optional filtering.

**Endpoint**: `GET /operations`

**Query Parameters**:
- `type`: Filter by operation type (`create`, `attach`, `detach`, `delete`)
- `status`: Filter by status (`pending`, `executing`, `completed`, `failed`)
- `volume_id`: Filter by volume ID
- `vm_id`: Filter by VM ID
- `limit`: Limit number of results (default: all)

**Example Requests**:
```bash
# All operations
curl http://localhost:8090/api/v1/operations

# Recent failed operations
curl "http://localhost:8090/api/v1/operations?status=failed&limit=10"

# Operations for specific volume
curl "http://localhost:8090/api/v1/operations?volume_id=vol-12345678-1234-1234-1234-123456789012"

# Pending attach operations
curl "http://localhost:8090/api/v1/operations?type=attach&status=pending"
```

**Response**: `200 OK`
```json
[
  {
    "id": "op-b2c3d4e5-f6g7-8901-bcde-f12345678901",
    "type": "attach",
    "status": "completed",
    "volume_id": "vol-12345678-1234-1234-1234-123456789012",
    "vm_id": "vm-87654321-4321-4321-4321-210987654321",
    "created_at": "2025-08-19T20:31:00Z",
    "completed_at": "2025-08-19T20:31:16Z"
  },
  {
    "id": "op-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "type": "create",
    "status": "completed", 
    "volume_id": "vol-12345678-1234-1234-1234-123456789012",
    "created_at": "2025-08-19T20:30:00Z",
    "completed_at": "2025-08-19T20:30:05Z"
  }
]
```

---

## Administrative

### Force Synchronization

Forces a synchronization between CloudStack state and device mappings.

**Endpoint**: `POST /admin/force-sync`

**Example Request**:
```bash
curl -X POST http://localhost:8090/api/v1/admin/force-sync
```

**Response**: `200 OK`
```json
{
  "message": "Force sync completed successfully",
  "timestamp": "2025-08-19T20:35:00Z"
}
```

---

## Health & Monitoring

### Health Check

Returns overall health status of the daemon.

**Endpoint**: `GET /health`

**Example Request**:
```bash
curl http://localhost:8090/api/v1/health
```

**Response**: `200 OK`
```json
{
  "status": "healthy",
  "timestamp": "2025-08-19T20:35:00Z",
  "cloudstack_health": "healthy",
  "database_health": "healthy",
  "device_monitor": "healthy",
  "details": {
    "implementation_status": "production_ready",
    "polling_interval": "2s",
    "active_operations": 0,
    "device_count": 4
  }
}
```

**Health Status Values**:
- `healthy`: All systems operational
- `degraded`: Some issues but functional  
- `unhealthy`: Critical issues affecting operation

---

### Service Metrics

Returns detailed operational metrics.

**Endpoint**: `GET /metrics`

**Example Request**:
```bash
curl http://localhost:8090/api/v1/metrics
```

**Response**: `200 OK`
```json
{
  "timestamp": "2025-08-19T20:35:00Z",
  "total_operations": 1547,
  "pending_operations": 2,
  "active_mappings": 12,
  "operations_by_type": {
    "create": 423,
    "attach": 512, 
    "detach": 489,
    "delete": 123
  },
  "operations_by_status": {
    "pending": 2,
    "executing": 1,
    "completed": 1523,
    "failed": 21,
    "cancelled": 0
  },
  "average_response_time_ms": 2340.5,
  "error_rate_percent": 1.36,
  "details": {
    "cloudstack_api_calls": 3094,
    "device_events_processed": 847,
    "database_transactions": 1547,
    "uptime_seconds": 86400
  }
}
```

---

## Error Handling

### Standard Error Format

All API errors follow this format:

```json
{
  "error": "Human readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "Additional context",
    "operation_id": "op-12345...",
    "timestamp": "2025-08-19T20:35:00Z"
  }
}
```

### Common Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `INVALID_REQUEST` | Request validation failed |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `CONFLICT` | Resource conflict (e.g., device already mapped) |
| 422 | `CLOUDSTACK_ERROR` | CloudStack API error |
| 500 | `INTERNAL_ERROR` | Internal server error |
| 503 | `SERVICE_UNAVAILABLE` | Service dependencies unavailable |

### Example Error Responses

**400 Bad Request**:
```json
{
  "error": "Invalid volume size: must be at least 1GB",
  "code": "INVALID_REQUEST",
  "details": {
    "field": "size",
    "provided_value": 536870912,
    "minimum_value": 1073741824
  }
}
```

**422 CloudStack Error**:
```json
{
  "error": "CloudStack volume creation failed: Invalid parameter zoneid",
  "code": "CLOUDSTACK_ERROR", 
  "details": {
    "cloudstack_error_code": 431,
    "cloudstack_message": "Invalid parameter zoneid value=invalid-zone",
    "operation_id": "op-a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }
}
```

---

## Rate Limiting

Currently no rate limiting is implemented as this is an internal service. Future versions may include:

- **Per-IP limits**: Protect against abuse
- **Operation type limits**: Prevent resource exhaustion
- **Burst allowances**: Handle traffic spikes

---

## API Versioning

- **Current Version**: `v1`
- **Version Header**: Not required (URL-based versioning)
- **Backward Compatibility**: Breaking changes will increment version number
- **Deprecation Policy**: 6 months notice for deprecated endpoints

---

## Client Libraries

### Go Client Example

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type VolumeRequest struct {
    Name           string            `json:"name"`
    Size           int64             `json:"size"`
    DiskOfferingID string            `json:"disk_offering_id"`
    ZoneID         string            `json:"zone_id"`
    Metadata       map[string]string `json:"metadata,omitempty"`
}

func createVolume(req VolumeRequest) error {
    jsonData, _ := json.Marshal(req)
    
    resp, err := http.Post(
        "http://localhost:8090/api/v1/volumes",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 201 {
        return fmt.Errorf("API error: %d", resp.StatusCode)
    }
    
    return nil
}
```

### Python Client Example

```python
import requests
import json

class VolumeManagementClient:
    def __init__(self, base_url="http://localhost:8090/api/v1"):
        self.base_url = base_url
    
    def create_volume(self, name, size, disk_offering_id, zone_id, metadata=None):
        data = {
            "name": name,
            "size": size,
            "disk_offering_id": disk_offering_id,
            "zone_id": zone_id,
            "metadata": metadata or {}
        }
        
        response = requests.post(
            f"{self.base_url}/volumes",
            json=data,
            headers={"Content-Type": "application/json"}
        )
        
        response.raise_for_status()
        return response.json()
    
    def get_operation_status(self, operation_id):
        response = requests.get(f"{self.base_url}/operations/{operation_id}")
        response.raise_for_status()
        return response.json()

# Usage
client = VolumeManagementClient()
operation = client.create_volume(
    name="test-volume",
    size=5368709120,
    disk_offering_id="c813c642-d946-49e1-9289-c616dd70206a", 
    zone_id="057e86db-c726-4d8c-ab1f-75c5f55d1881"
)
print(f"Operation ID: {operation['id']}")
```

---

## OpenAPI Specification

A complete OpenAPI 3.0 specification is available for code generation and testing tools. The specification includes:

- **Complete endpoint definitions**
- **Request/response schemas**
- **Error response formats**
- **Example requests and responses**

Access the specification at: `http://localhost:8090/api/v1/openapi.json` (when implemented).


## 🔗 **Persistent Device Naming Enhancement (v1.3.2)**

### New Database Fields

The Volume Management Daemon now supports persistent device naming to eliminate NBD export memory synchronization issues:

```sql
-- Enhanced device_mappings table:
persistent_device_name VARCHAR(255) NULL  -- Stable device name (e.g., vol3106013a)
symlink_path VARCHAR(255) NULL           -- Device mapper symlink (/dev/mapper/vol3106013a)
```

### Device Mapper Integration

**Persistent Device Creation:**
- Automatic generation of stable device names during volume attachment
- Device mapper symlinks (`/dev/mapper/vol[id]`) redirect to actual devices
- NBD exports use persistent symlinks for stable export names throughout volume lifecycle

### Enhanced Attachment Process

**Volume Attachment with Persistent Naming:**
1. Standard CloudStack volume attachment
2. Device correlation to actual Linux device path
3. Persistent device name generation (vol[first8chars])
4. Device mapper symlink creation (`/dev/mapper/vol[id]` → actual device)
5. Database update with persistent naming metadata
6. NBD export creation using persistent symlink path

### Production Benefits

- **NBD Export Stability**: Export names never change during volume operations
- **Memory Synchronization**: Eliminates NBD server stale export accumulation
- **Operational Reliability**: Post-failback replication jobs succeed consistently
- **Clear Troubleshooting**: Human-readable persistent device names
