# VM Failover System Documentation

**Version**: 1.0  
**Date**: 2025-08-18  
**Status**: Live Failover ✅ Complete | Test Failover ⚠️ Requires Architectural Update

## 📋 **Overview**

The VM Failover System provides live and test failover capabilities for VMs that have been fully replicated from VMware to OSSEA (CloudStack). The system supports both permanent live failovers and reversible test failovers for validation purposes.

### **Key Features**
- ✅ **Live Failover**: Permanent VM migration to OSSEA with volume transfer
- ⚠️ **Test Failover**: Reversible VM testing (requires architectural update)
- ✅ **Pre-Failover Validation**: Comprehensive readiness checks
- ✅ **Network Mapping**: Source to destination network configuration
- ✅ **Progress Tracking**: Real-time job status and execution monitoring
- ✅ **API Integration**: Complete REST API with Swagger documentation

## 🔧 **Architecture Components**

### **Core Components**
1. **Pre-Failover Validator** (`internal/oma/failover/validator.go`)
2. **Live Failover Engine** (`internal/oma/failover/live_failover.go`)
3. **Test Failover Engine** (`internal/oma/failover/test_failover.go`)
4. **Failover Handler** (`internal/oma/api/handlers/failover.go`)
5. **Network Mapping Service** (`internal/oma/services/network_mapping_service.go`)
6. **VM Info Service** (`internal/oma/services/simple_database_vm_info_service.go`)

### **Database Components**
- **Failover Jobs**: Track all failover operations
- **Network Mappings**: Source to destination network mappings
- **VM Disks**: Extended with VM specifications for failover
- **OSSEA Volumes**: CloudStack volume tracking
- **CBT History**: Change tracking for sync validation

## 🌐 **API Endpoints**

### **Failover Management**
```
POST   /api/v1/failover/live           - Initiate live failover
POST   /api/v1/failover/test           - Initiate test failover  
DELETE /api/v1/failover/test/{job_id}  - End test failover
GET    /api/v1/failover/{job_id}/status - Get failover job status
GET    /api/v1/failover/{vm_id}/readiness - Check VM failover readiness
GET    /api/v1/failover/jobs           - List all failover jobs
```

### **VM Validation**
```
GET /api/v1/vms/{vm_id}/failover-readiness    - Comprehensive VM validation
GET /api/v1/vms/{vm_id}/sync-status           - VM sync status check
GET /api/v1/vms/{vm_id}/network-mapping-status - Network mapping validation
GET /api/v1/vms/{vm_id}/volume-status         - Volume state validation
GET /api/v1/vms/{vm_id}/active-jobs           - Check active jobs
GET /api/v1/vms/{vm_id}/configuration-check   - Complete configuration validation
```

### **Network Management**
```
POST   /api/v1/network-mappings        - Create network mapping
GET    /api/v1/network-mappings/{vm_id} - Get VM network mappings
PUT    /api/v1/network-mappings/{vm_id} - Update network mappings
DELETE /api/v1/network-mappings/{vm_id} - Delete network mappings
GET    /api/v1/network-mappings        - List all network mappings
```

### **Debug Endpoints**
```
GET /api/v1/debug/health          - System health check
GET /api/v1/debug/failover-jobs   - Detailed failover job debugging
GET /api/v1/debug/endpoints       - List all available endpoints
GET /api/v1/debug/logs           - Recent system logs
```

## 📊 **Database Schema**

### **Failover Jobs Table**
```sql
CREATE TABLE failover_jobs (
    id               INT PRIMARY KEY AUTO_INCREMENT,
    job_id           VARCHAR(255) UNIQUE NOT NULL,     -- failover-YYYYMMDD-HHMMSS
    vm_id            VARCHAR(255) NOT NULL,            -- VMware VM ID  
    replication_job_id VARCHAR(255),                   -- FK to replication_jobs.id
    job_type         VARCHAR(50) NOT NULL,             -- 'live', 'test'
    status           VARCHAR(50) DEFAULT 'pending',    -- Job execution status
    source_vm_name   VARCHAR(255) NOT NULL,            -- Original VM name
    source_vm_spec   TEXT,                             -- JSON VM specifications
    destination_vm_id VARCHAR(255),                    -- Created OSSEA VM ID
    ossea_snapshot_id VARCHAR(255),                    -- Snapshot ID for cleanup
    network_mappings TEXT,                             -- JSON network mappings
    error_message    TEXT,                             -- Error details if failed
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    started_at       TIMESTAMP NULL,
    completed_at     TIMESTAMP NULL,
    
    INDEX idx_failover_jobs_vm_id (vm_id),
    INDEX idx_failover_jobs_replication (replication_job_id),
    INDEX idx_failover_jobs_status (status)
);
```

**Status Values:**
- `pending` - Job created, waiting to start
- `validating` - Running pre-failover validation
- `snapshotting` - Creating backup snapshot
- `creating_vm` - Creating OSSEA VM instance
- `switching_volume` - Detaching/attaching volumes
- `powering_on` - Starting the OSSEA VM
- `completed` - Failover successful
- `failed` - Failover failed
- `cleanup` - Cleaning up test resources
- `reverting` - Reverting test failover

### **Network Mappings Table**
```sql
CREATE TABLE network_mappings (
    id                      INT PRIMARY KEY AUTO_INCREMENT,
    vm_id                   VARCHAR(255) NOT NULL,            -- VMware VM ID
    source_network_name     VARCHAR(255) NOT NULL,            -- Source network name
    destination_network_id  VARCHAR(255) NOT NULL,            -- OSSEA network ID  
    destination_network_name VARCHAR(255) NOT NULL,           -- OSSEA network name
    is_test_network         BOOLEAN DEFAULT FALSE,            -- Test Layer 2 network
    created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE INDEX idx_network_mappings_unique (vm_id, source_network_name),
    INDEX idx_network_mappings_vm_id (vm_id)
);
```

### **Extended VM Disks Table**
The `vm_disks` table has been extended with VM specification fields for failover:

```sql
-- Additional fields added to existing vm_disks table:
ALTER TABLE vm_disks ADD COLUMN cpu_count INT DEFAULT 0;
ALTER TABLE vm_disks ADD COLUMN memory_mb INT DEFAULT 0;
ALTER TABLE vm_disks ADD COLUMN os_type VARCHAR(255) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN vm_tools_version VARCHAR(255) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN network_config TEXT;
ALTER TABLE vm_disks ADD COLUMN display_name VARCHAR(255) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN annotation TEXT;
ALTER TABLE vm_disks ADD COLUMN power_state VARCHAR(50) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN vmware_uuid VARCHAR(255) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN bios_setup TEXT;
```

**Purpose**: Store VM specifications during migration for use in failover operations.

## 🔄 **Failover Processes**

### **Live Failover Process**
1. **Pre-Validation**: Check VM readiness, sync status, network mappings
2. **Snapshot Creation**: Create OSSEA volume snapshot for rollback capability
3. **VM Creation**: Create identical OSSEA VM with exact source specifications
4. **Volume Operations**: Detach volume from OMA, attach to OSSEA VM as root disk
5. **Network Configuration**: Apply network mappings to OSSEA VM
6. **VM Startup**: Power on OSSEA VM and validate successful boot
7. **Completion**: Mark job as completed, update status

### **Test Failover Process** ⚠️ **REQUIRES ARCHITECTURAL UPDATE**

**Current Issue**: CloudStack KVM volume snapshots are disabled by default for running VMs.
**Error**: `"KVM Snapshot is not supported for Running VMs. To enable it set global settings kvm.snapshot.enabled to True"`

**New Approach Required**:
1. **Create Test VM**: Identical specifications with test Layer 2 network
2. **Volume Detach**: Safely detach volume from OMA instance
3. **Volume Attach**: Attach volume to test VM as root disk
4. **VM Snapshot**: Take VM snapshot (instead of volume snapshot)
5. **Test Execution**: Power up test VM for validation
6. **Cleanup Process**: Shutdown VM → Revert snapshot → Detach volume → Reattach to OMA

## 📝 **Request/Response Models**

### **Live Failover Request**
```json
{
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
  "vm_name": "PGWINTESTBIOS",
  "skip_validation": false,
  "network_mappings": {
    "VM Network": "production-network-id"
  },
  "custom_config": {},
  "notification_config": {}
}
```

### **Test Failover Request**
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
  "custom_config": {},
  "notification_config": {}
}
```

### **Failover Response**
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

### **Job Status Response**
```json
{
  "success": true,
  "message": "Retrieved status for test failover job",
  "job_id": "test-failover-4205a841-0265-f4bd-39a6-39fd92196f53-1755530108",
  "status": "failed",
  "progress": 15.5,
  "start_time": "2025-08-18T16:15:08+01:00",
  "duration": 125000000000,
  "job_details": {
    "job_type": "test",
    "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
    "vm_name": "PGWINTESTBIOS",
    "destination_vm_id": "ossea-vm-12345",
    "snapshot_id": "snapshot-67890",
    "error_message": "Volume snapshot failed",
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

### **Validation Response**
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
        "check_name": "Active Jobs",
        "check_type": "critical", 
        "status": "pass",
        "message": "No active jobs found for VM",
        "execution_time": 23000000
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
      },
      {
        "check_name": "Network Mappings",
        "check_type": "warning",
        "status": "pass",
        "message": "All source networks mapped to OSSEA networks",
        "execution_time": 34000000,
        "details": {
          "mapped_networks": 1,
          "unmapped_networks": 0,
          "test_networks_available": true
        }
      },
      {
        "check_name": "Volume State",
        "check_type": "critical",
        "status": "pass",
        "message": "OSSEA volume available and accessible",
        "execution_time": 89000000,
        "details": {
          "volume_id": "e915ef05-ddf5-48d5-8352-a01300609717",
          "volume_status": "attached",
          "volume_size_gb": 54
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

## 🔧 **Configuration**

### **OSSEA Configuration**
Failover operations require active OSSEA configuration:

```json
{
  "name": "primary-ossea",
  "api_url": "https://ossea.example.com:8080/client/api", 
  "api_key": "your-api-key",
  "secret_key": "your-secret-key",
  "domain": "ROOT",
  "zone": "zone1",
  "template_id": "template-12345",
  "network_id": "network-67890", 
  "service_offering_id": "offering-abcde",
  "disk_offering_id": "disk-offering-fghij",
  "oma_vm_id": "oma-vm-instance-id"
}
```

### **Network Mapping Configuration**
Each VM requires network mappings before failover:

```json
{
  "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
  "source_network_name": "VM Network",
  "destination_network_id": "network-production-123",
  "destination_network_name": "Production Network",
  "is_test_network": false
}
```

## ⚠️ **Known Issues & Limitations**

### **Test Failover Architecture Issue**
**Problem**: CloudStack KVM volume snapshots disabled by default  
**Impact**: Test failover fails at snapshot creation step  
**Status**: Requires architectural rewrite  
**Solution**: Implement VM snapshot approach instead of volume snapshots

### **Volume ID Mapping**
**Problem**: Database stores integer volume IDs, but CloudStack expects UUIDs  
**Status**: ✅ **RESOLVED** - Now correctly maps database ID to CloudStack UUID  
**Solution**: Added OSSEA volume lookup to get real CloudStack volume ID

### **VM Specification Storage**
**Problem**: VM specifications not populated during migration  
**Status**: ✅ **RESOLVED** - VM specs now stored in vm_disks table  
**Solution**: Enhanced migration workflow to capture CPU, memory, OS, network config

## 🔄 **Current Implementation Status**

### **✅ Completed Components**
- Pre-Failover Validator (660+ lines)
- Live Failover Engine (750+ lines) 
- API Handlers with full endpoint coverage
- Database schema and repositories
- Swagger documentation
- VM specification collection during migration
- Real CloudStack volume ID mapping

### **⚠️ Requires Update**
- Test Failover Engine (690+ lines) - architectural changes needed
- Volume detach/attach operations for test failover
- VM snapshot creation/revert functionality
- Test failover cleanup process

### **📋 Integration Points**
- **Migration Workflow**: Populates VM specifications during replication
- **CBT System**: Provides ChangeID validation for sync status
- **OSSEA Client**: Handles CloudStack API operations
- **GUI Integration**: React frontend with real-time progress tracking

## 🔍 **Testing & Validation**

### **Successful Test Results** ✅
- VM specification gathering: 2 CPUs, 4096MB memory for PGWINTESTBIOS
- ChangeID validation: Working with database-based lookup  
- Volume mapping: Correctly using CloudStack UUID `e915ef05-ddf5-48d5-8352-a01300609717`
- Network mapping: Test network configuration applied
- CloudStack API: Successful communication and parameter passing

### **Current Test Limitations** ⚠️
- Test failover stops at snapshot creation due to CloudStack KVM limitations
- Requires implementation of new VM snapshot approach
- Cleanup process needs updating for volume detach/reattach

## 📚 **Related Documentation**
- [API Documentation](../api/README.md)
- [Database Schema](../database-schema.md)
- [OSSEA Integration](../ossea-integration.md)
- [VM Failover Implementation Plan](../../AI_Helper/VM_FAILOVER_IMPLEMENTATION_PLAN.md)

