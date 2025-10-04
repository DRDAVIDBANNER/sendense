# Failover API Documentation

**Version**: 1.0  
**Date**: 2025-08-18  
**Base URL**: `http://localhost:8082/api/v1`

## 📋 **Overview**

The Failover API provides comprehensive VM failover management capabilities including live failover, test failover, validation, and network mapping operations. All endpoints support JSON request/response format and include detailed error handling.

## 🔐 **Authentication**

Currently running with authentication disabled for troubleshooting. In production, all endpoints require proper API authentication.

## 📍 **Endpoint Categories**

### 1. Failover Management
### 2. VM Validation  
### 3. Network Management
### 4. Debug Operations

---

## 🚀 **1. Failover Management**

### **POST /api/v1/failover/live**
Initiates a permanent live VM failover to OSSEA.

**Request Body:**
```json
{
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
  "vm_name": "PGWINTESTBIOS",
  "skip_validation": false,
  "network_mappings": {
    "VM Network": "production-network-id"
  },
  "custom_config": {
    "cpu_overcommit": 1.5,
    "memory_overcommit": 1.2
  },
  "notification_config": {
    "email": "admin@company.com",
    "slack_webhook": "https://hooks.slack.com/..."
  }
}
```

**Response (Success):**
```json
{
  "success": true,
  "message": "Live failover execution started successfully",
  "job_id": "live-failover-4205a841-0265-f4bd-39a6-39fd92196f53-1755530108",
  "estimated_duration": "5-10 minutes",
  "data": {
    "job_type": "live",
    "status": "executing",
    "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
    "vm_name": "PGWINTESTBIOS"
  }
}
```

**Response (Error):**
```json
{
  "success": false,
  "message": "Validation failed",
  "error": "VM has active migration jobs - cannot perform failover",
  "required_actions": [
    "Wait for active migration to complete",
    "Cancel active migration job"
  ]
}
```

---

### **POST /api/v1/failover/test**
Initiates a reversible test VM failover to OSSEA.

**Request Body:**
```json
{
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
  "vm_name": "PGWINTESTBIOS", 
  "skip_validation": false,
  "test_duration": "2h",
  "auto_cleanup": true,
  "network_mappings": {
    "VM Network": "test-network-id"
  },
  "custom_config": {
    "test_environment": true,
    "debug_mode": false
  },
  "notification_config": {}
}
```

**Request Parameters:**
- `vm_id` (required): VMware VM identifier
- `vm_name` (optional): Display name for the VM
- `skip_validation` (optional): Skip pre-failover validation (default: false)
- `test_duration` (required): Test duration (e.g., "30m", "2h", "4h30m")
- `auto_cleanup` (optional): Automatically cleanup after test duration (default: false)
- `network_mappings` (optional): Network mapping overrides
- `custom_config` (optional): Additional configuration parameters
- `notification_config` (optional): Notification settings

**Response (Success):**
```json
{
  "success": true,
  "message": "Test failover execution started successfully", 
  "job_id": "test-failover-4205a841-0265-f4bd-39a6-39fd92196f53-1755530108",
  "estimated_duration": "3-10 minutes (test duration: 2h0s)",
  "data": {
    "job_type": "test",
    "status": "executing",
    "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
    "vm_name": "PGWINTESTBIOS",
    "test_duration": "2h",
    "auto_cleanup": true
  }
}
```

**Current Known Issue:**
```json
{
  "success": false,
  "message": "Test failover failed",
  "error": "Failed to create test snapshot: CloudStack API error 530: KVM Snapshot is not supported for Running VMs",
  "required_actions": [
    "Enable kvm.snapshot.enabled in CloudStack global settings",
    "OR implement new VM snapshot approach (architectural update required)"
  ]
}
```

---

### **DELETE /api/v1/failover/test/{job_id}**
Terminates a test failover and cleans up resources.

**URL Parameters:**
- `job_id`: Test failover job ID

**Response (Success):**
```json
{
  "success": true,
  "message": "Test failover cleanup initiated successfully",
  "job_id": "test-failover-4205a841-0265-f4bd-39a6-39fd92196f53-1755530108",
  "cleanup_status": "in_progress",
  "estimated_cleanup_time": "2-5 minutes"
}
```

**Response (Error):**
```json
{
  "success": false,
  "message": "Cannot cleanup test failover",
  "error": "Job not found or not in a state that allows cleanup",
  "job_status": "failed"
}
```

---

### **GET /api/v1/failover/{job_id}/status**
Retrieves the current status of a failover job.

**URL Parameters:**
- `job_id`: Failover job ID

**Response (Success):**
```json
{
  "success": true,
  "message": "Retrieved status for test failover job",
  "job_id": "test-failover-4205a841-0265-f4bd-39a6-39fd92196f53-1755530108",
  "status": "executing",
  "progress": 45.5,
  "start_time": "2025-08-18T16:15:08+01:00",
  "duration": 125000000000,
  "job_details": {
    "job_type": "test",
    "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
    "vm_name": "PGWINTESTBIOS",
    "destination_vm_id": "ossea-vm-12345",
    "snapshot_id": "snapshot-67890",
    "error_message": null,
    "started_at": "2025-08-18T16:15:08+01:00",
    "completed_at": null,
    "custom_config": {
      "test_duration": "2h0s",
      "auto_cleanup": true,
      "network_mappings": {
        "VM Network": "test-network-default"
      }
    }
  }
}
```

**Status Values:**
- `pending` - Job queued, waiting to start
- `validating` - Running pre-failover validation  
- `snapshotting` - Creating backup snapshot
- `creating_vm` - Creating OSSEA VM instance
- `switching_volume` - Detaching/attaching volumes
- `powering_on` - Starting the OSSEA VM
- `completed` - Failover successful
- `failed` - Failover failed
- `cleanup` - Cleaning up test resources  
- `reverting` - Reverting test failover

---

### **GET /api/v1/failover/{vm_id}/readiness**
Performs comprehensive VM failover readiness validation.

**URL Parameters:**
- `vm_id`: VMware VM identifier

**Response (Success):**
```json
{
  "success": true,
  "message": "VM failover readiness validation completed",
  "is_valid": true,
  "readiness_score": 100.0,
  "validation_result": {
    "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
    "is_ready": true,
    "readiness_score": 100.0,
    "estimated_duration": "5-8 minutes",
    "checks": [
      {
        "check_name": "VM Existence",
        "check_type": "critical",
        "status": "pass",
        "message": "VM exists and is accessible",
        "execution_time": 45000000
      },
      {
        "check_name": "Sync Status", 
        "check_type": "critical",
        "status": "pass",
        "message": "VM has valid ChangeID from successful sync",
        "execution_time": 67000000,
        "details": {
          "job_id": "job-20250818-153521",
          "change_id": "52 3c ec 11 9e 2c 4c 3d-87 4a c3 4e 85 f2 ea 95/5",
          "sync_time": "2025-08-18T15:35:21+01:00"
        }
      }
    ],
    "warnings": [],
    "errors": [],
    "execution_time": 258000000
  },
  "required_actions": []
}
```

**Response (Validation Failed):**
```json
{
  "success": true,
  "message": "VM failover readiness validation completed",
  "is_valid": false,
  "readiness_score": 25.0,
  "validation_result": {
    "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
    "is_ready": false,
    "readiness_score": 25.0,
    "estimated_duration": "Cannot estimate - validation failed",
    "checks": [
      {
        "check_name": "Active Jobs",
        "check_type": "critical",
        "status": "fail",
        "message": "VM has active migration job in progress",
        "execution_time": 23000000,
        "details": {
          "active_job_id": "migration-job-12345",
          "job_status": "replicating"
        }
      }
    ],
    "warnings": [
      "VM network 'Legacy Network' not mapped to OSSEA network"
    ],
    "errors": [
      "Active migration job must complete before failover",
      "Network mappings incomplete"
    ],
    "execution_time": 189000000
  },
  "required_actions": [
    "Wait for migration job migration-job-12345 to complete",
    "Create network mapping for 'Legacy Network'"
  ]
}
```

---

### **GET /api/v1/failover/jobs**
Lists all failover jobs with optional filtering.

**Query Parameters:**
- `status` (optional): Filter by job status
- `type` (optional): Filter by job type (live, test)
- `vm_id` (optional): Filter by VM ID
- `limit` (optional): Number of results to return (default: 50)
- `offset` (optional): Result offset for pagination (default: 0)

**Example Request:**
```
GET /api/v1/failover/jobs?status=completed&type=test&limit=10
```

**Response (Success):**
```json
{
  "success": true,
  "message": "Retrieved failover jobs list",
  "total_jobs": 25,
  "returned_jobs": 10,
  "jobs": [
    {
      "job_id": "test-failover-4205a841-0265-f4bd-39a6-39fd92196f53-1755530108",
      "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
      "vm_name": "PGWINTESTBIOS",
      "job_type": "test",
      "status": "completed",
      "created_at": "2025-08-18T16:15:08+01:00",
      "started_at": "2025-08-18T16:15:15+01:00",
      "completed_at": "2025-08-18T18:20:33+01:00",
      "duration": 7518000000000,
      "destination_vm_id": "ossea-vm-test-12345"
    }
  ]
}
```

---

## ✅ **2. VM Validation**

### **GET /api/v1/vms/{vm_id}/failover-readiness**
Comprehensive VM failover readiness validation (same as `/api/v1/failover/{vm_id}/readiness`).

### **GET /api/v1/vms/{vm_id}/sync-status**
Checks VM sync status and ChangeID validation.

**Response:**
```json
{
  "success": true,
  "message": "VM sync status retrieved",
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
  "sync_status": {
    "has_valid_change_id": true,
    "latest_change_id": "52 3c ec 11 9e 2c 4c 3d-87 4a c3 4e 85 f2 ea 95/5",
    "last_sync_time": "2025-08-18T15:35:21+01:00",
    "replication_job_id": "job-20250818-153521",
    "sync_percentage": 100.0,
    "is_ready_for_failover": true
  }
}
```

### **GET /api/v1/vms/{vm_id}/network-mapping-status**
Validates network mapping configuration for failover.

**Response:**
```json
{
  "success": true,
  "message": "Network mapping status retrieved",
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
  "network_status": {
    "total_networks": 1,
    "mapped_networks": 1,
    "unmapped_networks": 0,
    "test_networks_available": true,
    "mappings": [
      {
        "source_network_name": "VM Network",
        "destination_network_id": "test-network-default", 
        "destination_network_name": "Test Network",
        "is_test_network": true,
        "status": "mapped"
      }
    ],
    "unmapped": [],
    "is_ready_for_failover": true
  }
}
```

### **GET /api/v1/vms/{vm_id}/volume-status**
Validates OSSEA volume availability and status.

**Response:**
```json
{
  "success": true,
  "message": "Volume status retrieved",
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53", 
  "volume_status": {
    "volume_id": "e915ef05-ddf5-48d5-8352-a01300609717",
    "volume_name": "migration-PGWINTESTBIOS-PGWINTESTBIOS-disk-0",
    "volume_status": "attached",
    "size_gb": 54,
    "volume_type": "ROOT",
    "attachment_status": "attached_to_oma",
    "is_accessible": true,
    "is_ready_for_failover": true
  }
}
```

### **GET /api/v1/vms/{vm_id}/active-jobs**
Checks for active migration or failover jobs that would prevent failover.

**Response:**
```json
{
  "success": true,
  "message": "Active jobs status retrieved",
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
  "active_jobs": {
    "has_active_jobs": false,
    "migration_jobs": [],
    "failover_jobs": [],
    "total_active": 0,
    "is_ready_for_failover": true
  }
}
```

### **GET /api/v1/vms/{vm_id}/configuration-check**
Comprehensive configuration validation combining all checks.

---

## 🌐 **3. Network Management**

### **POST /api/v1/network-mappings**
Creates a new network mapping for a VM.

**Request Body:**
```json
{
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
  "source_network_name": "VM Network",
  "destination_network_id": "network-production-123",
  "destination_network_name": "Production Network",
  "is_test_network": false
}
```

**Response:**
```json
{
  "success": true,
  "message": "Network mapping created successfully",
  "mapping": {
    "id": 15,
    "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
    "source_network_name": "VM Network",
    "destination_network_id": "network-production-123", 
    "destination_network_name": "Production Network",
    "is_test_network": false,
    "created_at": "2025-08-18T16:30:00+01:00"
  }
}
```

### **GET /api/v1/network-mappings/{vm_id}**
Retrieves all network mappings for a specific VM.

**Response:**
```json
{
  "success": true,
  "message": "Network mappings retrieved",
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
  "mappings": [
    {
      "id": 15,
      "source_network_name": "VM Network",
      "destination_network_id": "network-production-123",
      "destination_network_name": "Production Network", 
      "is_test_network": false,
      "created_at": "2025-08-18T16:30:00+01:00",
      "updated_at": "2025-08-18T16:30:00+01:00"
    }
  ],
  "total_mappings": 1
}
```

### **PUT /api/v1/network-mappings/{vm_id}**
Updates network mappings for a VM.

### **DELETE /api/v1/network-mappings/{vm_id}**
Deletes network mappings for a VM.

### **GET /api/v1/network-mappings**
Lists all network mappings with optional filtering.

---

## 🔍 **4. Debug Operations**

### **GET /api/v1/debug/health**
System health check and status.

**Response:**
```json
{
  "success": true,
  "message": "System health check completed",
  "status": "healthy",
  "components": {
    "database": "connected",
    "ossea_client": "configured",
    "failover_engines": "initialized",
    "api_server": "running"
  },
  "metrics": {
    "active_failover_jobs": 2,
    "total_vms_discovered": 95,
    "ossea_volumes": 12,
    "network_mappings": 8
  }
}
```

### **GET /api/v1/debug/failover-jobs**
Detailed failover job debugging information.

### **GET /api/v1/debug/endpoints**
Lists all available API endpoints.

### **GET /api/v1/debug/logs**
Recent system logs for troubleshooting.

---

## ⚠️ **Error Handling**

### **Standard Error Response Format**
```json
{
  "success": false,
  "message": "Error description",
  "error": "Detailed error message",
  "error_code": "VALIDATION_FAILED",
  "timestamp": "2025-08-18T16:45:30+01:00",
  "request_id": "req-12345-67890"
}
```

### **Common Error Codes**
- `VALIDATION_FAILED` - Pre-failover validation failed
- `VM_NOT_FOUND` - VM does not exist or is not accessible
- `ACTIVE_JOBS` - VM has active jobs preventing failover
- `NETWORK_MAPPING_MISSING` - Required network mappings not configured
- `VOLUME_NOT_AVAILABLE` - OSSEA volume not accessible
- `SNAPSHOT_FAILED` - Snapshot creation failed
- `OSSEA_API_ERROR` - CloudStack API error
- `INTERNAL_ERROR` - Unexpected system error

### **HTTP Status Codes**
- `200` - Success
- `400` - Bad Request (invalid parameters)
- `404` - Resource Not Found (VM, job, etc.)
- `409` - Conflict (active jobs, invalid state)
- `500` - Internal Server Error
- `503` - Service Unavailable (OSSEA not configured)

---

## 📚 **Additional Resources**

- **Swagger Documentation**: `http://localhost:8082/swagger/index.html`
- **Interactive API Testing**: Available via Swagger UI
- **Database Schema**: [Database Documentation](../database-schema.md)
- **OSSEA Integration**: [OSSEA Documentation](../ossea-integration.md)

