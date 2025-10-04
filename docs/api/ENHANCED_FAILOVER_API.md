# Enhanced Failover API Reference

## 🎯 **OVERVIEW**

The Enhanced Failover API provides the **ONLY** supported failover endpoints in MigrateKit OSSEA. All failover operations must use these enhanced endpoints.

> **⚠️ DEPRECATED**: Original failover endpoints are no longer supported.

## 🔗 **API ENDPOINTS**

### **Enhanced Test Failover**

**Endpoint**: `POST /api/v1/failover/test`

**Description**: Initiates an enhanced test failover with Linstor snapshots, VirtIO injection, and complete audit trails.

#### **Request**

```json
{
  "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
  "vm_name": "pgtest1",
  "auto_cleanup": false,
  "test_duration": "30m",
  "skip_validation": false,
  "skip_snapshot": false,
  "skip_virtio_injection": false,
  "network_mappings": {
    "source_network_id": "target_network_id"
  },
  "custom_config": {
    "key": "value"
  }
}
```

#### **Request Parameters**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `vm_id` | string | ✅ | **VMware VM UUID** (not VM name) |
| `vm_name` | string | ✅ | VM display name |
| `auto_cleanup` | boolean | ❌ | Auto-cleanup after test_duration |
| `test_duration` | string | ❌ | Duration to keep test running (e.g., "30m", "2h") |
| `skip_validation` | boolean | ❌ | Skip pre-failover validation |
| `skip_snapshot` | boolean | ❌ | Skip Linstor snapshot creation |
| `skip_virtio_injection` | boolean | ❌ | Skip VirtIO driver injection |
| `network_mappings` | object | ❌ | Network mapping overrides |
| `custom_config` | object | ❌ | Additional configuration |

#### **Response**

```json
{
  "success": true,
  "message": "Test failover initiated successfully with snapshot protection and VirtIO injection",
  "job_id": "enhanced-test-failover-420570c7-f61f-a930-77c5-1e876786cb3c-1756882765",
  "estimated_duration": "30m",
  "data": {
    "auto_cleanup": false,
    "correlation_id": "1bcd3a2c-1547-45fe-a4ce-af66094969a6",
    "job_type": "test",
    "snapshot_protection": true,
    "status": "executing",
    "test_duration": "30m",
    "virtio_injection": true,
    "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
    "vm_name": "pgtest1"
  }
}
```

#### **Error Response**

```json
{
  "success": false,
  "error": "Validation failed: VM not found or not replicated",
  "details": {
    "error_code": "VM_NOT_FOUND",
    "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c"
  }
}
```

### **Enhanced Live Failover**

**Endpoint**: `POST /api/v1/failover/live`

**Description**: Initiates a permanent enhanced live failover operation.

#### **Request**

```json
{
  "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
  "vm_name": "pgtest1",
  "skip_validation": false,
  "skip_snapshot": false,
  "skip_virtio_injection": false,
  "network_mappings": {
    "source_network_id": "target_network_id"
  },
  "notification_config": {
    "email": "admin@company.com"
  }
}
```

#### **Response**

```json
{
  "success": true,
  "message": "Live failover initiated successfully",
  "job_id": "enhanced-live-failover-420570c7-f61f-a930-77c5-1e876786cb3c-1756882900",
  "data": {
    "correlation_id": "2bcd3a2c-1547-45fe-a4ce-af66094969a7",
    "job_type": "live",
    "snapshot_protection": true,
    "status": "executing",
    "virtio_injection": true,
    "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
    "vm_name": "pgtest1"
  }
}
```

## 📊 **MONITORING ENDPOINTS**

### **List Failover Jobs**

**Endpoint**: `GET /api/v1/failover/jobs`

**Query Parameters**:
- `status`: Filter by status (pending, executing, completed, failed)
- `type`: Filter by type (test, live)
- `vm_id`: Filter by VM ID

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "job_id": "enhanced-test-failover-420570c7-f61f-a930-77c5-1e876786cb3c-1756882765",
      "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
      "vm_name": "pgtest1",
      "job_type": "test",
      "status": "completed",
      "linstor_snapshot_name": "test-420570c7-1756882765",
      "destination_vm_id": "8b5400e5-c92a-4bc4-8bff-4b6b0b6a018d",
      "created_at": "2025-09-03T07:59:25Z",
      "completed_at": "2025-09-03T08:05:30Z"
    }
  ],
  "total": 1
}
```

### **Get Failover Job Details**

**Endpoint**: `GET /api/v1/failover/jobs/{job_id}`

**Response**:
```json
{
  "success": true,
  "data": {
    "job_id": "enhanced-test-failover-420570c7-f61f-a930-77c5-1e876786cb3c-1756882765",
    "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
    "vm_name": "pgtest1",
    "job_type": "test",
    "status": "completed",
    "source_vm_spec": "{\"cpu_count\":2,\"memory_mb\":4096,\"vm_id\":\"420570c7-f61f-a930-77c5-1e876786cb3c\",\"vm_name\":\"pgtest1\"}",
    "destination_vm_id": "8b5400e5-c92a-4bc4-8bff-4b6b0b6a018d",
    "linstor_snapshot_name": "test-420570c7-1756882765",
    "network_mappings": "{\"vm_network\":\"test_network\"}",
    "created_at": "2025-09-03T07:59:25Z",
    "started_at": "2025-09-03T07:59:26Z",
    "completed_at": "2025-09-03T08:05:30Z",
    "steps": [
      {
        "step_name": "validation",
        "status": "completed",
        "started_at": "2025-09-03T07:59:26Z",
        "completed_at": "2025-09-03T07:59:27Z"
      },
      {
        "step_name": "snapshot",
        "status": "completed",
        "started_at": "2025-09-03T07:59:27Z",
        "completed_at": "2025-09-03T08:01:15Z"
      }
    ]
  }
}
```

## 🔄 **JOB LIFECYCLE**

### **Status Flow**

```
pending → validating → executing → [step-specific statuses] → completed
                                                           → failed
```

### **Test Failover Steps**

1. `validation` - Pre-failover validation
2. `snapshot` - Linstor snapshot creation
3. `virtio-injection` - VirtIO driver injection
4. `test-vm-creation` - Test VM creation
5. `volume-attachment` - Volume attachment
6. `test-vm-startup` - VM startup and validation

### **Live Failover Steps**

1. `validation` - Pre-failover validation
2. `snapshot` - Linstor snapshot creation
3. `virtio-injection` - VirtIO driver injection
4. `vm-creation` - Production VM creation
5. `volume-switch` - Volume switching
6. `vm-startup` - VM startup

## 🛡️ **ERROR HANDLING**

### **Common Error Codes**

| Error Code | Description | Resolution |
|------------|-------------|------------|
| `VM_NOT_FOUND` | VM UUID not found in replication jobs | Verify VM is replicated |
| `VALIDATION_FAILED` | Pre-failover validation failed | Check validation details |
| `SNAPSHOT_FAILED` | Linstor snapshot creation failed | Check Linstor connectivity |
| `VIRTIO_FAILED` | VirtIO injection failed | Check driver availability |
| `VM_CREATE_FAILED` | CloudStack VM creation failed | Check CloudStack config |
| `VOLUME_ATTACH_FAILED` | Volume attachment failed | Check Volume Daemon |

### **Error Response Format**

```json
{
  "success": false,
  "error": "Human readable error message",
  "error_code": "MACHINE_READABLE_CODE",
  "details": {
    "step": "validation",
    "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
    "correlation_id": "1bcd3a2c-1547-45fe-a4ce-af66094969a6"
  },
  "troubleshooting": {
    "suggestions": [
      "Check VM replication status",
      "Verify CloudStack connectivity"
    ],
    "documentation": "/docs/failover/troubleshooting.md"
  }
}
```

## 🔐 **AUTHENTICATION**

Currently, the API operates without authentication for development. Production deployments should implement proper authentication mechanisms.

## 📈 **RATE LIMITING**

- **Concurrent Failovers**: Maximum 5 concurrent failover operations per VM
- **Rate Limit**: 10 requests per minute per client IP
- **Burst Limit**: 20 requests in 30 seconds

## 🔍 **DEBUGGING**

### **Correlation IDs**

All API responses include `correlation_id` for tracking operations across logs:

```bash
# Search logs by correlation ID
sudo journalctl -u oma-api | grep "1bcd3a2c-1547-45fe-a4ce-af66094969a6"
```

### **Database Monitoring**

```sql
-- Check failover job status
SELECT job_id, status, linstor_snapshot_name, error_message 
FROM failover_jobs 
WHERE vm_id = '420570c7-f61f-a930-77c5-1e876786cb3c';

-- Check job tracking details
SELECT operation, status, error_message 
FROM job_tracking 
WHERE correlation_id = '1bcd3a2c-1547-45fe-a4ce-af66094969a6';
```

## 📝 **EXAMPLES**

### **cURL Examples**

#### **Test Failover**
```bash
curl -X POST http://localhost:8082/api/v1/failover/test \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
    "vm_name": "pgtest1",
    "test_duration": "1h",
    "auto_cleanup": true
  }'
```

#### **Live Failover**
```bash
curl -X POST http://localhost:8082/api/v1/failover/live \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
    "vm_name": "pgtest1"
  }'
```

#### **Monitor Progress**
```bash
curl http://localhost:8082/api/v1/failover/jobs/enhanced-test-failover-420570c7-f61f-a930-77c5-1e876786cb3c-1756882765
```

---

**This API documentation is current as of September 2025 and represents the only supported failover API.**
