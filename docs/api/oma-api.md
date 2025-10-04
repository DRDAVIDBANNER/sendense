# OMA (OSSEA Migration Appliance) API Documentation

## 🎯 **API Overview**

The OMA API provides **unified OSSEA configuration management** and **migration job orchestration** for VMware to OSSEA migrations with **VM-Based Export Reuse Architecture**. Following the **anti-sprawl principle**, it uses minimal endpoints with multi-functional operations and **intelligent export management** that reuses VM exports without unnecessary SIGHUP operations.

## 🌐 **API Access**

- **Base URL**: `http://localhost:8082` (OMA local)
- **VMA Access**: `http://localhost:8082` (via SSH forward tunnel)
- **External Access**: `http://10.245.246.125:8082` (if needed)
- **Authentication**: Session-based with Bearer tokens
- **Protocol**: HTTP (TLS via reverse proxy in production)
- **Documentation**: `http://localhost:8082/swagger/` (Interactive Swagger UI)

## 🏗️ **Architecture**

### **Directory Structure** (Go Standard Layout)
```
cmd/oma-api/main.go              # Main entry point
internal/oma/api/
├── server.go                    # HTTP server with gorilla/mux
├── handlers/
│   ├── handlers.go             # Centralized handler struct
│   ├── auth.go                 # Authentication endpoints
│   ├── ossea.go                # OSSEA configuration (unified endpoint)
│   ├── vm.go                   # VM inventory endpoints
│   └── replication.go          # Replication job endpoints
└── database/
    ├── connection.go           # Database abstraction interface
    ├── models.go              # GORM models
    └── repository.go          # Repository pattern implementation
```

### **Database Integration**
- **Database**: MariaDB with GORM ORM
- **Auto-Migration**: Schema automatically created/updated on startup
- **Repository Pattern**: Clean data access layer with VM export mapping repository
- **Connection Interface**: Supports both MariaDB and in-memory modes
- **VM Export Mappings**: Persistent `vm_export_mappings` table for export reuse tracking

## 🔐 **Authentication**

### **Login**
```http
POST /api/v1/auth/login
Content-Type: application/json
```

**Request**:
```json
{
  "appliance_id": "test-vma",
  "token": "vma_test_token_abc123def456789012345678",
  "version": "1.0.0"
}
```

**Response**:
```json
{
  "success": true,
  "message": "Authentication successful",
  "session_token": "oma_session_xyz789abc456def123",
  "expires_at": "2025-08-06T12:00:00Z"
}
```

### **Logout**
```http
POST /api/v1/auth/logout
Authorization: Bearer oma_session_xyz789abc456def123
```

## 🔧 **OSSEA Configuration Management**

### **Unified OSSEA Configuration Endpoint**

**Single POST endpoint handles ALL operations** (following anti-sprawl rule):

```http
POST /api/v1/ossea/config
Authorization: Bearer <session_token>
Content-Type: application/json
```

#### **Get All Configurations**
```json
{
  "action": "get"
}
```

**Response**:
```json
{
  "success": true,
  "configurations": [
    {
      "id": 1,
      "name": "production-ossea",
      "api_url": "http://10.245.241.101:8080/client/api",
      "api_key": "GdsWBVHco0OOW4vBlgfnyzjru65FV-U1l1kKJ7n2WwH0gn3soTaZQZZyXfgUsxX7PyP06WrOOOcNRKmhRWDSlA",
      "secret_key": "[REDACTED]",
      "domain": "/",
      "zone": "OSSEA-Zone",
      "oma_vm_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "is_active": true,
      "created_at": "2025-08-06T11:25:47Z",
      "updated_at": "2025-08-06T11:26:17Z"
    }
  ]
}
```

#### **Get Configuration by ID**
```json
{
  "action": "get",
  "id": 1
}
```

#### **Create Configuration**
```json
{
  "action": "create",
  "configuration": {
    "name": "production-ossea",
    "api_url": "http://10.245.241.101:8080/client/api",
    "api_key": "GdsWBVHco0OOW4vBlgfnyzjru65FV-U1l1kKJ7n2WwH0gn3soTaZQZZyXfgUsxX7PyP06WrOOOcNRKmhRWDSlA",
    "secret_key": "uTCroKUkHZaNybhBXkcQsCb_eKDvZKbhHaZK4I1nHrGJYLKN-j0O-t9EGUx9yBdHH3F8dN5wVelitvdpQjwdcQ",
    "domain": "/",
    "zone": "OSSEA-Zone",
    "oma_vm_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
  }
}
```

#### **Update Configuration**
```json
{
  "action": "update",
  "id": 1,
  "configuration": {
    "name": "production-ossea-updated",
    "zone": "OSSEA-Zone-Updated"
  }
}
```

#### **Delete Configuration**
```json
{
  "action": "delete",
  "id": 1
}
```

#### **Test OSSEA Connection**
```json
{
  "action": "test",
  "configuration": {
    "api_url": "http://10.245.241.101:8080/client/api",
    "api_key": "GdsWBVHco0OOW4vBlgfnyzjru65FV-U1l1kKJ7n2WwH0gn3soTaZQZZyXfgUsxX7PyP06WrOOOcNRKmhRWDSlA",
    "secret_key": "uTCroKUkHZaNybhBXkcQsCb_eKDvZKbhHaZK4I1nHrGJYLKN-j0O-t9EGUx9yBdHH3F8dN5wVelitvdpQjwdcQ",
    "domain": "/",
    "zone": "OSSEA-Zone"
  }
}
```

**Response**:
```json
{
  "success": true,
  "message": "OSSEA connection test successful",
  "details": {
    "zones_found": 1,
    "zone_verified": "OSSEA-Zone",
    "api_accessible": true
  }
}
```

## 📊 **VM Inventory Management**

### **Discover VMs**
```http
GET /api/v1/vms/discover?vcenter=vcenter.example.com
Authorization: Bearer <session_token>
```

### **List VMs**
```http
GET /api/v1/vms
Authorization: Bearer <session_token>
```

## 🔄 **Replication Job Management**

### **Create Automated Migration Job**
```http
POST /api/v1/replications
Authorization: Bearer <session_token>
Content-Type: application/json
```

**Request** (Simplified - all volume creation/mounting automated):
```json
{
  "source_vm": {
    "id": "vm-143233",
    "name": "PGWINTESTBIOS",
    "path": "/DatabanxDC/vm/PGWINTESTBIOS",
    "datacenter": "DatabanxDC",
    "cpus": 2,
    "memory_mb": 4096,
    "disks": [
      {
        "id": "disk-001",
        "path": "[datastore1] PGWINTESTBIOS/PGWINTESTBIOS.vmdk",
        "size_gb": 40,
        "datastore": "datastore1",
        "capacity_bytes": 42949672960
      }
    ]
  },
  "ossea_config_id": 1,
  "replication_type": "initial"
}
```

**Response** (Complete automation with VM export reuse):
```json
{
  "job_id": "job-20250812-080802",
  "status": "ready_for_sync",
  "progress_percent": 100.0,
  "source_vm": { "..." },
  "created_volumes": [
    {
      "volume_id": "vol-123456",
      "volume_name": "migration-pgtest2-pgtest2-disk-0",
      "size_gb": 40,
      "status": "created"
    }
  ],
  "nbd_exports": [
    {
      "export_name": "migration-vm-4205784a-098a-40f1-1f1e-a5cd2597fd59-disk0",
      "device_path": "/dev/vdc",
      "port": 10809,
      "reused": true,
      "vm_id": "4205784a-098a-40f1-1f1e-a5cd2597fd59",
      "disk_unit_number": 0
    }
  ],
  "started_at": "2025-08-12T08:08:02Z",
  "message": "Migration workflow started - VM export reused without SIGHUP operation"
}
```

### **List Replication Jobs**
```http
GET /api/v1/replication/jobs
Authorization: Bearer <session_token>
```

### **Get Job Status**
```http
GET /api/v1/replication/jobs/{job_id}
Authorization: Bearer <session_token>
```

## 🏥 **Health and Monitoring**

### **Health Check**
```http
GET /health
```

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-08-06T11:18:06Z",
  "database": "connected",
  "version": "1.0.0"
}
```

## 🚀 **Deployment and Service Management**

### **Starting the API Server**

**MariaDB Mode (Production)**:
```bash
./bin/oma-api -port 8082 -db-type mariadb \
  -db-host localhost -db-port 3306 \
  -db-name migratekit_oma -db-user oma_user \
  -db-pass oma_password -debug
```

**Memory Mode (Development)**:
```bash
./bin/oma-api -port 8082 -db-type memory -debug
```

### **Systemd Service**

The API can be deployed as a systemd service:

```bash
# Install service
sudo cp scripts/oma-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable oma-api
sudo systemctl start oma-api

# Check status
sudo systemctl status oma-api
```

### **Configuration Flags**

| Flag | Description | Default |
|------|-------------|---------|
| `-port` | HTTP server port | 8080 |
| `-debug` | Enable debug logging | false |
| `-db-type` | Database type (mariadb/memory) | memory |
| `-db-host` | MariaDB hostname | localhost |
| `-db-port` | MariaDB port | 3306 |
| `-db-name` | Database name | migratekit_oma |
| `-db-user` | Database username | oma_user |
| `-db-pass` | Database password | - |
| `-auth-enabled` | Enable authentication | true |

## 📚 **Interactive Documentation**

The API includes complete Swagger documentation:

- **URL**: `http://localhost:8082/swagger/`
- **Features**: Interactive API testing, complete model definitions, example requests/responses
- **Auto-generated**: Documentation stays current with code changes

## 🔧 **Database Schema**

### **OSSEA Configurations Table**
```sql
CREATE TABLE ossea_configs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    api_url VARCHAR(500) NOT NULL,
    api_key VARCHAR(500) NOT NULL,
    secret_key VARCHAR(500) NOT NULL,
    domain VARCHAR(255) DEFAULT '/',
    zone VARCHAR(255) NOT NULL,
    oma_vm_id VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### **VM Export Mappings Table**
```sql
CREATE TABLE vm_export_mappings (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    vm_id VARCHAR(36) NOT NULL,               -- VMware UUID
    disk_unit_number INT NOT NULL,            -- SCSI unit number (0,1,2...)
    vm_name VARCHAR(255) NOT NULL,            -- VMware VM name  
    export_name VARCHAR(255) NOT NULL UNIQUE, -- NBD export name
    device_path VARCHAR(255) NOT NULL,        -- /dev/vdb, /dev/vdc, /dev/vdd
    status ENUM('active', 'inactive') DEFAULT 'active',
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY unique_vm_disk (vm_id, disk_unit_number),
    INDEX idx_vm_id (vm_id),
    INDEX idx_export_name (export_name),
    INDEX idx_device_path (device_path),
    INDEX idx_status (status)
);
```

### **Auto-Migration**
- Schema automatically created/updated on startup using GORM AutoMigrate
- No manual migration required
- Supports schema evolution for future updates

## 🚨 **Error Handling**

### **Standard Error Response**
```json
{
  "success": false,
  "error": "Configuration not found",
  "details": "No OSSEA configuration found with ID: 999"
}
```

### **Common HTTP Status Codes**
- `200` - Success
- `201` - Created
- `400` - Bad Request (invalid JSON, missing fields)
- `401` - Unauthorized (invalid session token)
- `404` - Not Found (configuration/resource not found)
- `500` - Internal Server Error (database connection, etc.)

## 🎯 **VM Context API - GUI Integration**

The VM Context API provides comprehensive VM-centric data access for frontend applications, implementing the VM-Centric Architecture with minimal, powerful endpoints.

### **Endpoints**

#### **List All VM Contexts**
```
GET /api/v1/vm-contexts
```
- **Purpose**: Dashboard overview of all VMs
- **Response**: Array of VM contexts with essential metadata
- **Features**: VM status, job statistics, current operation

#### **Detailed VM Context**
```
GET /api/v1/vm-contexts/{vm_name}
```
- **Purpose**: Complete VM details for drilling down
- **Response**: Comprehensive VM data including:
  - VM context metadata and status
  - Current job with live progress tracking
  - Job history (last 10 jobs)
  - VM disk information and specifications
  - CBT change tracking history (last 20 records)

### **Key Features**
- ✅ **Single Source of Truth**: All VM data linked via `vm_context_id`
- ✅ **Real-time Progress**: Live job status and transfer metrics
- ✅ **Historical Data**: Complete job and change tracking history
- ✅ **VM Specifications**: CPU, memory, disks, network configuration
- ✅ **Performance Optimized**: Limited queries with efficient database indexes
- ✅ **Authentication Required**: Bearer token authentication on all endpoints

### **Example Response (Detailed)**
```json
{
  "context": {
    "context_id": "ctx-pgtest1-20250909-113839",
    "vm_name": "pgtest1",
    "current_status": "replicating",
    "total_jobs_run": 2,
    "successful_jobs": 0,
    "failed_jobs": 0
  },
  "current_job": {
    "id": "job-20250909-114850",
    "status": "replicating",
    "progress_percent": 49.31,
    "current_operation": "Transferring Data",
    "vma_throughput_mbps": 11.46,
    "vma_eta_seconds": 1631
  },
  "job_history": [/* Last 10 jobs */],
  "disks": [/* VM disk configuration */],
  "cbt_history": [/* Change tracking records */]
}
```

### **GUI Integration**
- **Dashboard View**: Use `/vm-contexts` for VM list with status overview
- **Detail View**: Use `/vm-contexts/{vm_name}` for comprehensive VM data
- **Real-time Updates**: Poll detail endpoint for live progress tracking
- **Minimal API**: Two endpoints provide complete functionality

**📖 Full Documentation**: See `docs/api/VM_CONTEXT_API.md` for complete API reference, examples, and integration patterns.

---

## 📈 **Performance and Monitoring**

### **Logging**
- **Structured logging** with logrus
- **Request timing** for all endpoints
- **Database query logging** in debug mode
- **Authentication events** logged

### **Metrics**
- API response times logged
- Database connection status monitored
- Auto-migration success/failure tracked

## 🔒 **Security**

### **Authentication**
- Session-based authentication with bearer tokens
- Token validation on all protected endpoints
- Configurable authentication (can be disabled for development)

### **Database Security**
- Dedicated database user with minimal privileges
- Connection strings not logged
- Secret keys handled securely

### **Network Security**
- Designed for tunnel-based access (SSH forward from VMA)
- No direct external exposure required
- TLS termination handled by reverse proxy in production

---

**Last Updated**: 2025-08-12  
**API Version**: 1.0.0  
**Database**: MariaDB with GORM ORM + VM Export Mappings  
**Major Feature**: VM-Based Export Reuse preventing NBD server restarts
**Status**: ✅ **PRODUCTION READY**