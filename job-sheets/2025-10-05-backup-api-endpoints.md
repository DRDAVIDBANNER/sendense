# Job Sheet: Backup API Endpoints Implementation

**Date Created:** 2025-10-05  
**Status:** 🔴 **READY TO START**  
**Project Goal Link:** [project-goals/phases/phase-1-vmware-backup.md → Task 5: API Endpoints]  
**Duration:** 1 week  
**Priority:** High (GUI integration and automation capability)

---

## 🎯 PROJECT GOALS INTEGRATION (MANDATORY)

### **Specific Project Goals Reference**
**Document:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`  
**Task Section:** **Task 5: API Endpoints** (Lines 407-454)  
**Sub-Tasks:** **REST API for backup operations**  
**Business Value:** GUI integration and backup automation (customer self-service)  
**Success Criteria:** Complete backup workflow accessible via REST API

**Task Description (From Project Goals):**
```
Goal: Expose backup operations via REST API

Endpoints to Implement:
- POST /api/v1/backup/start (full and incremental backups)
- GET /api/v1/backup/list (list VM backups)  
- GET /api/v1/backup/{backup_id} (backup details)
- DELETE /api/v1/backup/{backup_id} (delete backup)
- GET /api/v1/backup/chain (get backup chain)
```

**Acceptance Criteria (From Project Goals):**
- [ ] All endpoints functional
- [ ] Proper error handling
- [ ] RBAC integration (existing system)
- [ ] API documentation (Swagger)

---

## 🔗 DEPENDENCY STATUS

### **Required Before Starting:**
- ✅ Task 1: Repository infrastructure (Local/NFS/CIFS/Immutable)
- ✅ Task 2: NBD file export (QCOW2 export capability)
- ✅ Task 3: BackupEngine workflow (complete orchestration)
- ✅ Task 4: File-level restore (complete customer workflow)

### **Enables These Features:**
- ⏸️ GUI backup operations (frontend integration)
- ⏸️ Backup automation scripts (API-driven workflows)
- ⏸️ Customer self-service backups
- ⏸️ Complete end-to-end backup solution

---

## 📋 JOB BREAKDOWN (Detailed Implementation)

### **Phase 1: Backup Handler Foundation (Days 1-2)**

- [ ] **Create Backup Handler Structure**
  - **File:** `source/current/oma/api/handlers/backup_handlers.go`
  - **Structure:** BackupHandler with BackupEngine integration
  - **Dependencies:** BackupEngine from Task 3, Repository pattern
  - **Evidence:** Handler skeleton with proper dependency injection

- [ ] **Request/Response Models**
  - **Structs:** BackupStartRequest, BackupResponse, BackupListResponse
  - **Validation:** Input validation with comprehensive error messages
  - **Integration:** Map to BackupEngine.BackupRequest from Task 3
  - **Evidence:** Complete API schemas for all endpoints

- [ ] **Handler Integration**
  - **File:** `source/current/oma/api/handlers/handlers.go`
  - **Addition:** Backup *BackupHandler field
  - **Initialization:** Wire up BackupHandler in NewHandlers()
  - **Evidence:** BackupHandler available to API router

### **Phase 2: Core Backup Endpoints (Days 2-4)**

- [ ] **Start Backup Endpoint**
  - **Route:** POST /api/v1/backup/start
  - **Function:** StartBackup() handler
  - **Integration:** Call BackupEngine.ExecuteBackup() from Task 3
  - **Evidence:** Can start full and incremental backups via API

- [ ] **List Backups Endpoint**
  - **Route:** GET /api/v1/backup/list
  - **Function:** ListBackups() handler  
  - **Filters:** vm_context_id, repository_id, backup_type, status
  - **Evidence:** Returns paginated backup list with metadata

- [ ] **Get Backup Details Endpoint**
  - **Route:** GET /api/v1/backup/{backup_id}
  - **Function:** GetBackupDetails() handler
  - **Integration:** BackupEngine.GetBackup() from Task 3
  - **Evidence:** Returns complete backup metadata and status

- [ ] **Delete Backup Endpoint**
  - **Route:** DELETE /api/v1/backup/{backup_id}
  - **Function:** DeleteBackup() handler
  - **Validation:** Respect immutability settings from Task 1
  - **Evidence:** Secure backup deletion with protection rules

### **Phase 3: Chain Management Endpoints (Days 4-5)**

- [ ] **Backup Chain Endpoint**
  - **Route:** GET /api/v1/backup/chain
  - **Function:** GetBackupChain() handler
  - **Query:** vm_context_id, disk_id parameters
  - **Evidence:** Returns complete backup chain with relationships

- [ ] **Chain Consolidation Endpoint**
  - **Route:** POST /api/v1/backup/consolidate
  - **Function:** ConsolidateChain() handler
  - **Operation:** Merge incremental backups (advanced feature)
  - **Evidence:** Chain consolidation via BackupEngine

### **Phase 4: Integration & Route Registration (Days 5-6)**

- [ ] **Route Registration**
  - **File:** `source/current/oma/api/server.go`
  - **Integration:** Register 6 backup endpoints with authentication
  - **Middleware:** requireAuth() on all backup endpoints
  - **Evidence:** Backup routes accessible via /api/v1/backup/*

- [ ] **Authentication Integration**
  - **Security:** All endpoints require bearer token
  - **Integration:** Use existing requireAuth middleware
  - **Validation:** Unauthorized requests rejected with 401
  - **Evidence:** Backup endpoints protected by authentication

- [ ] **Error Handling**
  - **Standards:** Consistent HTTP status codes (400, 404, 500)
  - **Messages:** Clear error messages for validation failures
  - **Context:** Structured error responses with details
  - **Evidence:** Proper error handling across all endpoints

### **Phase 5: Documentation & Testing (Days 6-7)**

- [ ] **API Documentation**
  - **File:** `source/current/api-documentation/OMA.md`
  - **Addition:** Complete backup endpoint documentation
  - **Schemas:** Request/response examples for all endpoints
  - **Evidence:** All backup endpoints documented with examples

- [ ] **Integration Testing**
  - **Workflow:** Start backup → monitor → list → delete cycle
  - **Validation:** Full and incremental backup workflows via API
  - **Error Testing:** Invalid requests handled gracefully
  - **Evidence:** Complete backup workflow operational via REST

- [ ] **Handler Testing**
  - **Unit Tests:** Test all handler methods
  - **Mock Integration:** Mock BackupEngine for isolated testing
  - **Coverage:** >80% test coverage for backup handlers
  - **Evidence:** Comprehensive test suite for API endpoints

---

## 🏗️ TECHNICAL ARCHITECTURE

### **Handler Integration with BackupEngine**
```go
type BackupHandler struct {
    backupEngine *workflows.BackupEngine  // Task 3 integration
    db           database.Connection      // Database access
}

func (bh *BackupHandler) StartBackup(w http.ResponseWriter, r *http.Request) {
    var req BackupStartRequest
    // Parse request, validate, call backupEngine.ExecuteBackup()
}
```

### **Request/Response Schemas**
```go
type BackupStartRequest struct {
    VMName       string `json:"vm_name"`       // VM to backup
    DiskID       int    `json:"disk_id"`       // Disk number  
    BackupType   string `json:"backup_type"`   // "full" or "incremental"
    RepositoryID string `json:"repository_id"` // Target repository
    PolicyID     string `json:"policy_id,omitempty"` // Optional policy
}

type BackupResponse struct {
    BackupID         string    `json:"backup_id"`
    Status           string    `json:"status"`           // pending, running, completed, failed
    BackupType       string    `json:"backup_type"`
    RepositoryID     string    `json:"repository_id"`
    FilePath         string    `json:"file_path"`
    NBDExportName    string    `json:"nbd_export_name"`
    BytesTransferred int64     `json:"bytes_transferred"`
    TotalBytes       int64     `json:"total_bytes"`
    CreatedAt        time.Time `json:"created_at"`
}
```

### **API Endpoint Specifications**
```bash
# Start backup
POST /api/v1/backup/start
Request: BackupStartRequest
Response: BackupResponse
Auth: Required

# List backups
GET /api/v1/backup/list?vm_name=pgtest2&repository_id=local-ssd&status=completed
Response: { backups: [BackupResponse], total: number }
Auth: Required

# Get backup details  
GET /api/v1/backup/{backup_id}
Response: BackupResponse with complete metadata
Auth: Required

# Delete backup
DELETE /api/v1/backup/{backup_id}
Response: { message: "backup deleted successfully" }
Auth: Required
Protection: Respects immutability settings

# Get backup chain
GET /api/v1/backup/chain?vm_context_id=ctx-pgtest2-20251005&disk_id=0
Response: { chain_id, full_backup_id, backups: [], total_size_bytes }
Auth: Required

# Consolidate chain (advanced)
POST /api/v1/backup/consolidate
Request: { chain_id: string }
Response: { consolidation_job_id: string, status: "running" }
Auth: Required
```

---

## ✅ SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **Start Backup API:** Can start full and incremental backups via REST
- [ ] **List Backups API:** Can query backups with filtering
- [ ] **Backup Details API:** Can retrieve complete backup metadata
- [ ] **Delete Backup API:** Can delete backups with immutability protection
- [ ] **Chain Management:** Can retrieve and manage backup chains
- [ ] **Authentication:** All endpoints require and validate bearer tokens
- [ ] **Error Handling:** Proper HTTP status codes and error messages

### **Integration Evidence Required**
- [ ] Start backup via API triggers BackupEngine workflow
- [ ] Backup progress visible through existing progress infrastructure
- [ ] Backup files accessible via file-level restore endpoints (Task 4)
- [ ] Repository integration works across all repository types
- [ ] Immutable backups properly protected from deletion
- [ ] API errors provide clear guidance for resolution

---

## 🚨 PROJECT RULES COMPLIANCE

### **Must Follow (No Exceptions):**
- ✅ **Source Authority:** All code in `source/current/` only
- ✅ **Repository Pattern:** Use existing database repositories 
- ✅ **BackupEngine Integration:** Use Task 3 infrastructure without modification
- ✅ **Authentication Required:** All endpoints use requireAuth middleware
- ✅ **Error Handling:** Graceful failures with structured responses
- ✅ **API Documentation:** Update OMA.md with all new endpoints
- ✅ **No Simulations:** Real backup operations only

### **Integration Requirements:**
- **Task 1 Integration:** Use RepositoryManager for repository operations
- **Task 2 Integration:** Coordinate with NBD file exports (no conflicts)
- **Task 3 Integration:** Use BackupEngine as primary orchestration layer
- **Task 4 Integration:** Ensure backups created are accessible for restore
- **Database:** Use repository pattern for all database operations

---

## 📊 DELIVERABLES

### **Code Deliverables**
- `source/current/oma/api/handlers/backup_handlers.go` - Complete backup API implementation
- Enhanced `source/current/oma/api/handlers/handlers.go` - BackupHandler integration
- Enhanced `source/current/oma/api/server.go` - Route registration for backup endpoints

### **API Endpoints (6 total)**
- `POST /api/v1/backup/start` - Start backup operations
- `GET /api/v1/backup/list` - List and filter backups
- `GET /api/v1/backup/{backup_id}` - Get backup details
- `DELETE /api/v1/backup/{backup_id}` - Delete backup with protection
- `GET /api/v1/backup/chain` - Retrieve backup chain
- `POST /api/v1/backup/consolidate` - Consolidate backup chain

### **Documentation Deliverables**
- Updated API documentation (OMA.md) with complete backup endpoint specs
- Request/response schemas and examples
- Error code documentation
- Authentication requirements

---

## 🔗 INTEGRATION POINTS

### **Task 3 Dependencies (BackupEngine)**
- **Primary Integration:** Use `workflows.BackupEngine` for all operations
- **Methods:** ExecuteBackup(), GetBackup(), ListBackups(), CompleteBackup()
- **No Modification:** Use BackupEngine as-is, no changes to Task 3 code

### **Task 1 Dependencies (Repository System)**
- **Repository Access:** Use RepositoryManager for backup location operations
- **Multi-Repository:** Support backup operations across Local/NFS/CIFS/Immutable
- **Validation:** Repository existence validation before backup start

### **Task 2 Dependencies (NBD File Export)**
- **Coordination:** Understand NBD export naming (no conflicts needed)
- **Integration:** BackupEngine already handles NBD export via Task 2

### **Task 4 Dependencies (File Restore)**
- **Compatibility:** Backups created must be mountable by restore endpoints
- **Workflow:** Complete customer journey: backup → restore files

### **Database Integration**
- **Repository Pattern:** Use existing BackupJobRepository from Task 3
- **No Direct SQL:** All database operations via repository interfaces
- **Consistency:** Maintain existing database schema and relationships

---

## 🎯 ENTERPRISE VALUE

### **Customer Capabilities Enabled**
- ✅ **API-Driven Backups** - Customers can automate backup operations
- ✅ **GUI Integration** - Frontend can control backup workflows
- ✅ **Backup Monitoring** - Real-time backup status via API
- ✅ **Self-Service Management** - Customers manage their own backups
- ✅ **Scriptable Operations** - DevOps teams can script backup workflows

### **Business Benefits**
- ✅ **MSP Integration** - Service providers can integrate backup APIs
- ✅ **Automation Ready** - Scheduled backups via API calls
- ✅ **Monitoring Integration** - Third-party monitoring systems
- ✅ **Customer Portal** - Self-service backup management
- ✅ **Competitive Position** - Complete API coverage vs limited competitors

---

## 📋 ACCEPTANCE CRITERIA

### **Functional Requirements**
- [ ] **Start Backup API:** Can trigger full and incremental backups
- [ ] **List Backups API:** Can query and filter backup collections  
- [ ] **Backup Details API:** Can retrieve complete backup information
- [ ] **Delete Backup API:** Can delete backups with immutability respect
- [ ] **Chain Management:** Can retrieve backup chain relationships
- [ ] **Status Monitoring:** Real-time backup status via API

### **Technical Requirements**
- [ ] **Authentication:** All endpoints require and validate bearer tokens
- [ ] **Error Handling:** Proper HTTP status codes (200, 400, 401, 404, 500)
- [ ] **Validation:** Input validation with clear error messages
- [ ] **Integration:** Clean use of BackupEngine without modification
- [ ] **Performance:** Endpoint response times <2 seconds

### **Documentation Requirements**
- [ ] **Complete API Docs:** All endpoints documented in OMA.md
- [ ] **Request Examples:** Working JSON examples for all requests
- [ ] **Response Examples:** Complete response schemas with field descriptions
- [ ] **Error Documentation:** Error codes and resolution guidance

---

## 🔧 IMPLEMENTATION DETAILS

### **Backup Start Integration**
```go
func (bh *BackupHandler) StartBackup(w http.ResponseWriter, r *http.Request) {
    var req BackupStartRequest
    // Parse and validate request
    
    // Convert to BackupEngine request format (Task 3 integration)
    backupReq := &workflows.BackupRequest{
        VMContextID:  req.VMContextID,
        VMName:       req.VMName,
        DiskID:       req.DiskID,
        RepositoryID: req.RepositoryID,
        BackupType:   storage.BackupType(req.BackupType),
        // ...
    }
    
    // Call BackupEngine (Task 3)
    result, err := bh.backupEngine.ExecuteBackup(ctx, backupReq)
    // Return API response
}
```

### **Repository Integration Pattern**
```go
// Find VM context for backup operations
vmContext, err := bh.vmContextRepo.GetByVMName(req.VMName)
if err != nil {
    return http.StatusNotFound, "VM not found"
}

// Validate repository exists (Task 1 integration)
repo, err := bh.repositoryManager.GetRepository(req.RepositoryID)
if err != nil {
    return http.StatusBadRequest, "Repository not available"
}
```

### **Backup List Filtering**
```go
type BackupListQuery struct {
    VMContextID  string `json:"vm_context_id,omitempty"`
    RepositoryID string `json:"repository_id,omitempty"` 
    BackupType   string `json:"backup_type,omitempty"`   // full, incremental
    Status       string `json:"status,omitempty"`        // pending, running, completed, failed
    Limit        int    `json:"limit,omitempty"`         // Pagination
    Offset       int    `json:"offset,omitempty"`        // Pagination
}
```

---

## 🔐 SECURITY CONSIDERATIONS

### **Authentication & Authorization**
- **Bearer Tokens:** All endpoints require valid authentication
- **User Validation:** Ensure user access to requested VMs and repositories
- **Role-Based Access:** Different permissions for backup vs admin operations

### **Input Validation**
- **VM Name Validation:** Prevent injection attacks
- **Path Validation:** Secure repository ID validation
- **Parameter Limits:** Prevent resource exhaustion attacks
- **Request Size:** Limit request payload sizes

### **Backup Protection**
- **Immutability Respect:** Cannot delete immutable backups before retention
- **Repository Protection:** Cannot delete backups if repository is immutable
- **Chain Integrity:** Prevent deletion of parent backups with children

---

## 🎯 API DESIGN PRINCIPLES

### **RESTful Design**
- **Resource-Based:** Backup as primary resource
- **HTTP Methods:** GET (read), POST (create), DELETE (remove)
- **Status Codes:** Meaningful HTTP status codes
- **Idempotency:** GET operations are safe and idempotent

### **Error Response Format**
```json
{
  "error": "validation_failed",
  "message": "VM name is required",
  "details": {
    "field": "vm_name",
    "code": "required"
  }
}
```

### **Success Response Format**
```json
{
  "backup_id": "backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000",
  "status": "running",
  "backup_type": "full",
  "repository_id": "local-ssd-primary",
  "vm_name": "pgtest2",
  "file_path": "/backups/pgtest2/disk0/backup-full-20251005T120000.qcow2",
  "nbd_export_name": "backup-ctx-pgtest2-disk0-full-20251005T120000",
  "total_bytes": 107374182400,
  "created_at": "2025-10-05T12:00:00Z"
}
```

---

## ✅ SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **All 6 Endpoints:** Start, list, details, delete, chain, consolidate
- [ ] **BackupEngine Integration:** All operations use Task 3 infrastructure
- [ ] **Authentication Working:** All endpoints require valid tokens
- [ ] **Error Handling:** Comprehensive validation and error responses
- [ ] **Documentation Complete:** All endpoints documented in OMA.md
- [ ] **Integration Testing:** Complete backup workflow via API

### **Testing Evidence Required**
- [ ] Start full backup via API successfully
- [ ] Start incremental backup with parent chain detection
- [ ] List backups with various filter combinations
- [ ] Get backup details for active and completed backups
- [ ] Delete backup with proper immutability protection
- [ ] Retrieve backup chain showing full → incremental relationships
- [ ] All endpoints return proper error responses for invalid inputs

---

## 🚨 PROJECT RULES COMPLIANCE

### **Must Follow (No Exceptions):**
- ✅ **Source Authority:** All code in `source/current/` only
- ✅ **Repository Pattern:** Use existing database repositories
- ✅ **BackupEngine Unchanged:** Use Task 3 infrastructure as-is
- ✅ **Authentication Required:** All endpoints use existing auth middleware  
- ✅ **Error Handling:** Structured error responses with clear messages
- ✅ **API Documentation:** Update OMA.md with complete endpoint specs
- ✅ **No Simulations:** Real backup operations via BackupEngine

### **Integration Constraints:**
- **Router Consistency:** Use gorilla/mux (existing pattern)
- **Handler Pattern:** Follow existing handler structure and initialization
- **Endpoint Naming:** Follow /api/v1/* pattern
- **Response Format:** Consistent with existing API responses
- **Authentication:** Use existing requireAuth middleware

---

## 🔗 READY FOR IMPLEMENTATION

### **Foundation Operational (Tasks 1-4)**
- ✅ **BackupEngine** - Complete workflow orchestration ready
- ✅ **Repository System** - Multi-repository backup support
- ✅ **NBD File Export** - QCOW2 backup file handling
- ✅ **File Restore** - Complete customer recovery workflow
- ✅ **Database Schema** - backup_jobs, backup_chains operational
- ✅ **API Infrastructure** - Handler pattern and authentication ready

### **Clear Implementation Path**
- ✅ **Scope Defined** - 6 specific endpoints with clear requirements  
- ✅ **Integration Clear** - Use BackupEngine methods directly
- ✅ **Patterns Established** - Follow existing handler architecture
- ✅ **Testing Strategy** - Build on Tasks 1-4 operational foundation

---

## 🎯 TASK 5 ENABLES GUI INTEGRATION

**What This Unlocks:**
- **Backup Dashboard** - GUI can start/monitor/manage backups
- **Customer Portal** - Self-service backup operations
- **DevOps Integration** - Scriptable backup workflows
- **MSP Automation** - Service provider backup automation

**Customer Journey Complete:**
```
API: Start Backup → API: Monitor Progress → API: List Backups → 
API: Mount for Recovery → API: Browse Files → API: Download Files
```

---

**THIS JOB COMPLETES THE BACKUP API LAYER**

**ENABLES COMPLETE CUSTOMER AUTOMATION**

---

**Job Owner:** Backend Engineering Team  
**Reviewer:** Architecture Lead + API Review  
**Status:** 🔴 Ready to Start  
**Last Updated:** 2025-10-05
