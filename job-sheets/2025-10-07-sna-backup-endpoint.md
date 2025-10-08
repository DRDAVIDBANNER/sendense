# Job Sheet: SNA Backup Endpoint Implementation

**Job ID:** JS-2025-10-07-SNA-BACKUP-ENDPOINT  
**Project Goal Reference:** `/project-goals/phases/phase-1-vmware-backup.md` (Task 7.4)  
**Related Job:** `2025-10-07-unified-nbd-architecture.md` (Unified NBD Architecture)  
**Status:** 🔴 **READY TO START** - All context researched  
**Created:** October 7, 2025  
**Priority:** CRITICAL - Blocks end-to-end testing  
**Estimated Duration:** 3-4 hours  
**Complexity:** Medium

---

## 🎯 **Objective**

Implement `/api/v1/backup/start` endpoint in SNA API to accept multi-disk backup requests from SHA, orchestrate VMware backup operations, and return job tracking information.

**Success Criteria:**
- ✅ Endpoint accepts multi-disk NBD targets from SHA
- ✅ Parses VMware credentials and VM information
- ✅ Launches `sendense-backup-client` (SBC) with correct parameters
- ✅ Returns job ID for progress tracking
- ✅ Integrates with existing job tracking system
- ✅ NO simulation code - only real backup operations
- ✅ Follows project rules (no CloudStack dependencies, clean code)

---

## 📚 **Context & Background**

### **Current State (October 7, 2025)**

**SHA (Hub) Side:** ✅ **100% COMPLETE**
- Multi-disk backup API working
- NBD Port Allocator operational (10100-10200)
- qemu-nbd Process Manager with `--shared=10`
- Repository management (467GB available)
- Builds multi-disk NBD targets string: `"2000:nbd://127.0.0.1:10100/export1,2001:nbd://127.0.0.1:10101/export2"`

**SNA (Node) Side:** ⚠️ **BLOCKED**
- SSH tunnel working (101 NBD ports forwarded)
- Old services cleaned up
- **MISSING:** `/api/v1/backup/start` endpoint causes 404 error
- **Impact:** SHA cannot initiate backups, end-to-end testing blocked

**Evidence from Session:**
```bash
# SHA log shows successful multi-disk detection and port allocation:
INFO[0057] Allocated NBD ports for 2 disks              
INFO[0057] Starting qemu-nbd on port 10100 for /backup/repository/pgtest1-disk-2000.qcow2
INFO[0057] Starting qemu-nbd on port 10101 for /backup/repository/pgtest1-disk-2001.qcow2
INFO[0057] Built NBD targets: 2000:nbd://127.0.0.1:10100/...,2001:nbd://127.0.0.1:10101/...

# SHA attempts to call SNA API:
POST http://localhost:9081/api/v1/backup/start
Response: 404 Not Found ❌

# Root cause: SNA API server.go has NO /backup/start handler
```

### **Architecture Context**

```
┌─────────────────────────────────────────────────────────┐
│                 SHA (Hub Appliance)                      │
│                                                          │
│  POST /api/v1/backup/start                              │
│  {                                                       │
│    "vm_name": "pgtest1",                                │
│    "vcenter_host": "10.0.100.10",                       │
│    "nbd_targets": "2000:nbd://...10100,2001:nbd://10101"│
│  }                                                       │
└────────────────────┬────────────────────────────────────┘
                     │ Reverse tunnel (port 9081)
                     │ OR Direct SNA API (port 8081)
         ════════════▼════════════════════════════════════
┌─────────────────────────────────────────────────────────┐
│                 SNA (Node Appliance)                     │
│                                                          │
│  ┌──────────────────────────────────────────────┐      │
│  │ SNA API (port 8081)                          │      │
│  │ POST /api/v1/backup/start ◄─── NEW ENDPOINT │      │
│  │ - handleBackupStart()                        │      │
│  │ - Parse request                              │      │
│  │ - Launch sendense-backup-client              │      │
│  │ - Return job ID                              │      │
│  └──────────────────┬───────────────────────────┘      │
│                     │                                    │
│  ┌──────────────────▼───────────────────────────┐      │
│  │ sendense-backup-client (SBC)                 │      │
│  │ - Connects to VMware via VDDK                │      │
│  │ - Reads disk data (CBT-aware)                │      │
│  │ - Writes to NBD targets via tunnel           │      │
│  │ - Multi-disk coordination                    │      │
│  └──────────────────┬───────────────────────────┘      │
│                     │                                    │
│                     │ Connect via tunnel                 │
│  ┌──────────────────▼───────────────────────────┐      │
│  │ SSH Tunnel (port 443)                        │      │
│  │ Forward: 10100-10200 → SHA NBD ports         │      │
│  └──────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────┘
```

### **Project Rules Compliance**

**From `/start_here/PROJECT_RULES.md`:**
- ✅ NO SIMULATIONS: Real VMware backup operations only
- ✅ SOURCE AUTHORITY: All code in `source/current/sna/`
- ✅ DOCUMENTATION: Update API docs after implementation
- ✅ NO DEVIATIONS: Follows approved Phase 1 Task 7.4 plan

**From `/start_here/MASTER_AI_PROMPT.md`:**
- ✅ TERMINOLOGY: SNA (not VMA), SHA (not OMA)
- ✅ NAMING: `sendense-backup-client` (not migratekit)
- ✅ STRUCTURE: Go standard project layout

**From `/project-goals/phases/phase-1-vmware-backup.md`:**
- ✅ PHASE 1 TASK 7.4: "SHA Backup API Updates" (dependencies complete)
- ✅ UNIFIED NBD ARCHITECTURE: Port-based NBD (10100-10200)
- ✅ MULTI-DISK SUPPORT: Accept NBD targets string format

---

## 📋 **Implementation Tasks**

### **Task 1: Define Request/Response Structures** ⏱️ 15 min

**File:** `source/current/sna/api/server.go`  
**Location:** After existing type definitions (~line 196)

**Add these types:**
```go
// BackupRequest represents a backup job request from SHA
type BackupRequest struct {
    JobID          string `json:"job_id"`           // SHA-generated job ID
    VMName         string `json:"vm_name"`          // VM name for identification
    VCenterHost    string `json:"vcenter_host"`     // vCenter hostname
    VCenterUser    string `json:"vcenter_user"`     // vCenter username
    VCenterPass    string `json:"vcenter_password"` // vCenter password
    VMPath         string `json:"vm_path"`          // VMware VM path (e.g., "/DC1/vm/pgtest1")
    NBDTargets     string `json:"nbd_targets"`      // Multi-disk NBD targets string
    BackupType     string `json:"backup_type"`      // "full" or "incremental"
    PreviousChangeID string `json:"previous_change_id,omitempty"` // For incremental backups
}

// BackupResponse represents the response from starting a backup
type BackupResponse struct {
    JobID     string `json:"job_id"`      // Echo back job ID
    Status    string `json:"status"`      // "started", "failed"
    Message   string `json:"message"`     // Success/error message
    StartedAt string `json:"started_at"`  // ISO 8601 timestamp
    PID       int    `json:"pid"`         // SBC process ID
}
```

**Acceptance Criteria:**
- [ ] Types added after line 196 (near other request/response structs)
- [ ] Field tags match JSON naming convention
- [ ] Comments explain each field's purpose
- [ ] No linter errors

---

### **Task 2: Add Route Registration** ⏱️ 5 min

**File:** `source/current/sna/api/server.go`  
**Location:** `setupRoutes()` method (~line 235)

**Add route:**
```go
// setupRoutes configures the SNA Control API endpoints
func (s *SNAControlServer) setupRoutes() {
    api := s.router.PathPrefix("/api/v1").Subrouter()

    // Core control endpoints
    api.HandleFunc("/cleanup", s.handleCleanup).Methods("POST")
    api.HandleFunc("/status/{job_id}", s.handleStatus).Methods("GET")
    api.HandleFunc("/config", s.handleConfig).Methods("PUT")
    api.HandleFunc("/health", s.handleHealth).Methods("GET")
    api.HandleFunc("/vms/{vm_path:.*}/cbt-status", s.handleCBTStatus).Methods("GET")

    // SHA-initiated workflow endpoints
    api.HandleFunc("/discover", s.handleDiscover).Methods("POST")
    api.HandleFunc("/replicate", s.handleReplicate).Methods("POST")
    api.HandleFunc("/backup/start", s.handleBackupStart).Methods("POST") // ← NEW
    api.HandleFunc("/vm-spec-changes", s.handleVMSpecChanges).Methods("POST")

    // ... rest of routes ...
}
```

**Acceptance Criteria:**
- [ ] Route added in logical position (after `/replicate`, before `/vm-spec-changes`)
- [ ] Method restricted to POST only
- [ ] Endpoint count updated in log message
- [ ] No linter errors

---

### **Task 3: Implement Handler Logic** ⏱️ 90 min

**File:** `source/current/sna/api/server.go`  
**Location:** After existing handlers (~line 500)

**Implementation:**
```go
// handleBackupStart processes backup job requests from SHA
// This endpoint accepts multi-disk NBD targets and launches sendense-backup-client
func (s *SNAControlServer) handleBackupStart(w http.ResponseWriter, r *http.Request) {
    var req BackupRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.WithError(err).Error("Invalid backup request")
        http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
        return
    }

    log.WithFields(log.Fields{
        "job_id":     req.JobID,
        "vm_name":    req.VMName,
        "vcenter":    req.VCenterHost,
        "nbd_targets": req.NBDTargets,
        "backup_type": req.BackupType,
    }).Info("🎬 Received backup request from SHA")

    // Validate required fields
    if err := s.validateBackupRequest(&req); err != nil {
        log.WithError(err).Error("Backup request validation failed")
        http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
        return
    }

    // Build sendense-backup-client command
    cmd, err := s.buildBackupCommand(&req)
    if err != nil {
        log.WithError(err).Error("Failed to build backup command")
        http.Error(w, fmt.Sprintf("Command build failed: %v", err), http.StatusInternalServerError)
        return
    }

    // Start backup process
    if err := cmd.Start(); err != nil {
        log.WithError(err).Error("Failed to start backup process")
        http.Error(w, fmt.Sprintf("Process start failed: %v", err), http.StatusInternalServerError)
        return
    }

    // Add job to tracker for status monitoring
    s.AddJobWithProgress(req.JobID, req.VMPath)

    // Create response
    response := BackupResponse{
        JobID:     req.JobID,
        Status:    "started",
        Message:   fmt.Sprintf("Backup started for %s", req.VMName),
        StartedAt: time.Now().UTC().Format(time.RFC3339),
        PID:       cmd.Process.Pid,
    }

    log.WithFields(log.Fields{
        "job_id":  req.JobID,
        "vm_name": req.VMName,
        "pid":     response.PID,
    }).Info("✅ Backup process started successfully")

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

// validateBackupRequest validates the backup request fields
func (s *SNAControlServer) validateBackupRequest(req *BackupRequest) error {
    if req.JobID == "" {
        return fmt.Errorf("job_id is required")
    }
    if req.VMName == "" {
        return fmt.Errorf("vm_name is required")
    }
    if req.VCenterHost == "" {
        return fmt.Errorf("vcenter_host is required")
    }
    if req.VCenterUser == "" {
        return fmt.Errorf("vcenter_user is required")
    }
    if req.VCenterPass == "" {
        return fmt.Errorf("vcenter_password is required")
    }
    if req.VMPath == "" {
        return fmt.Errorf("vm_path is required")
    }
    if req.NBDTargets == "" {
        return fmt.Errorf("nbd_targets is required")
    }
    if req.BackupType != "full" && req.BackupType != "incremental" {
        return fmt.Errorf("backup_type must be 'full' or 'incremental'")
    }
    if req.BackupType == "incremental" && req.PreviousChangeID == "" {
        return fmt.Errorf("previous_change_id is required for incremental backups")
    }
    return nil
}

// buildBackupCommand constructs the sendense-backup-client command
func (s *SNAControlServer) buildBackupCommand(req *BackupRequest) (*exec.Cmd, error) {
    // sendense-backup-client binary path
    sbcBinary := "/usr/local/bin/sendense-backup-client"

    // Check if binary exists
    if _, err := os.Stat(sbcBinary); os.IsNotExist(err) {
        // Fallback to migratekit for backwards compatibility
        sbcBinary = "/usr/local/bin/migratekit"
        if _, err := os.Stat(sbcBinary); os.IsNotExist(err) {
            return nil, fmt.Errorf("sendense-backup-client binary not found")
        }
        log.Warn("Using migratekit binary (legacy) - upgrade to sendense-backup-client recommended")
    }

    // Build command arguments
    args := []string{
        "migrate",
        "--vcenter", req.VCenterHost,
        "--username", req.VCenterUser,
        "--password", req.VCenterPass,
        "--vm", req.VMPath,
        "--nbd-targets", req.NBDTargets,
        "--job-id", req.JobID,
    }

    // Add incremental backup parameters if needed
    if req.BackupType == "incremental" && req.PreviousChangeID != "" {
        args = append(args, "--change-id", req.PreviousChangeID)
    }

    // Create command
    cmd := exec.Command(sbcBinary, args...)

    // Set up logging to /var/log/sendense/
    logDir := "/var/log/sendense"
    if err := os.MkdirAll(logDir, 0755); err != nil {
        log.WithError(err).Warn("Failed to create log directory, using /tmp")
        logDir = "/tmp"
    }

    logPath := filepath.Join(logDir, fmt.Sprintf("backup-%s.log", req.JobID))
    logFile, err := os.Create(logPath)
    if err != nil {
        return nil, fmt.Errorf("failed to create log file: %w", err)
    }

    cmd.Stdout = logFile
    cmd.Stderr = logFile

    log.WithFields(log.Fields{
        "binary":   sbcBinary,
        "job_id":   req.JobID,
        "log_path": logPath,
    }).Info("Built backup command")

    return cmd, nil
}
```

**Acceptance Criteria:**
- [ ] Handler follows existing code style (similar to `handleReplicate`)
- [ ] All required fields validated with clear error messages
- [ ] sendense-backup-client command built correctly
- [ ] NBD targets string passed correctly
- [ ] Job added to tracker for status monitoring
- [ ] Process started asynchronously (non-blocking)
- [ ] Logs written to `/var/log/sendense/backup-{job_id}.log`
- [ ] Response includes job_id, status, message, timestamp, PID
- [ ] No linter errors
- [ ] NO simulation code - real VMware operations only

---

### **Task 4: Add Required Imports** ⏱️ 5 min

**File:** `source/current/sna/api/server.go`  
**Location:** Top of file (~line 7)

**Add imports:**
```go
import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "os"              // ← ADD for os.Stat, os.MkdirAll
    "os/exec"         // ← ADD for exec.Command
    "path/filepath"   // ← ADD for filepath.Join
    "sync"
    "time"

    "github.com/gorilla/mux"
    log "github.com/sirupsen/logrus"
    // ... rest of imports ...
)
```

**Acceptance Criteria:**
- [ ] Imports added in alphabetical order (Go convention)
- [ ] No unused imports
- [ ] No linter errors

---

### **Task 5: Build and Deploy** ⏱️ 30 min

**Build:**
```bash
cd /home/oma_admin/sendense/source/current/sna
go build -o sna-api-v1.4.0-backup-endpoint cmd/main.go
```

**Deploy to SNA (10.0.100.231):**
```bash
# Copy binary to SNA
sshpass -p 'Password1' scp sna-api-v1.4.0-backup-endpoint \
    vma@10.0.100.231:/tmp/

# SSH to SNA and install
sshpass -p 'Password1' ssh vma@10.0.100.231 'sudo bash -s' <<'DEPLOY'
    # Stop old service
    sudo systemctl stop sna-api 2>/dev/null || true
    
    # Install new binary
    sudo mv /tmp/sna-api-v1.4.0-backup-endpoint /opt/vma/bin/sna-api
    sudo chmod +x /opt/vma/bin/sna-api
    sudo chown root:root /opt/vma/bin/sna-api
    
    # Create systemd service if not exists
    if [ ! -f /etc/systemd/system/sna-api.service ]; then
        cat > /tmp/sna-api.service <<'SERVICE'
[Unit]
Description=Sendense Node Appliance (SNA) API Server
Documentation=https://sendense.io/docs/sna-api
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/vma
ExecStart=/opt/vma/bin/sna-api --port=8081
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=sna-api

# Security hardening
NoNewPrivileges=true
PrivateTmp=true

# Resource limits
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
SERVICE
        sudo mv /tmp/sna-api.service /etc/systemd/system/
    fi
    
    # Start service
    sudo systemctl daemon-reload
    sudo systemctl enable sna-api
    sudo systemctl start sna-api
    
    # Wait and check
    sleep 3
    sudo systemctl status sna-api --no-pager
DEPLOY
```

**Acceptance Criteria:**
- [ ] Binary builds without errors
- [ ] Binary copied to SNA successfully
- [ ] Service starts without errors
- [ ] Health check responds: `curl http://localhost:8081/api/v1/health`
- [ ] New endpoint available: `curl -X POST http://localhost:8081/api/v1/backup/start`
- [ ] Service logs show no errors: `journalctl -u sna-api -n 50`

---

### **Task 6: Integration Testing** ⏱️ 45 min

**Test 1: Endpoint Availability**
```bash
# On SHA
curl -X POST http://localhost:9081/api/v1/backup/start \
    -H "Content-Type: application/json" \
    -d '{
        "job_id": "test-001",
        "vm_name": "test",
        "vcenter_host": "10.0.100.10",
        "vcenter_user": "user",
        "vcenter_password": "pass",
        "vm_path": "/DC1/vm/test",
        "nbd_targets": "2000:nbd://127.0.0.1:10100/test",
        "backup_type": "full"
    }'

# Expected: 200 OK with job_id, status, started_at, pid
```

**Test 2: Validation Errors**
```bash
# Missing required field
curl -X POST http://localhost:9081/api/v1/backup/start \
    -H "Content-Type: application/json" \
    -d '{
        "job_id": "test-002"
    }'

# Expected: 400 Bad Request with "vm_name is required" message
```

**Test 3: End-to-End Backup (pgtest1)**
```bash
# On SHA
curl -X POST http://localhost:8082/api/v1/backup/start \
    -H "Content-Type: application/json" \
    -d '{
        "vm_name": "pgtest1",
        "repository_id": "1",
        "backup_type": "full"
    }'

# Expected:
# 1. SHA allocates ports (10100, 10101)
# 2. SHA starts qemu-nbd processes
# 3. SHA calls SNA API → 200 OK (not 404!)
# 4. SNA launches sendense-backup-client
# 5. SBC connects to VMware and NBD targets
# 6. Backup completes with 2 QCOW2 files created
```

**Test 4: Job Status Tracking**
```bash
# Check job status on SNA
curl http://localhost:9081/api/v1/status/backup-job-123

# Expected: Job status with progress_percent, current_operation
```

**Acceptance Criteria:**
- [ ] Endpoint responds (not 404)
- [ ] Validation errors return 400 with clear messages
- [ ] Valid requests return 200 with job details
- [ ] sendense-backup-client process starts
- [ ] Logs written to `/var/log/sendense/backup-{job_id}.log`
- [ ] End-to-end backup completes successfully
- [ ] Multi-disk VMs create multiple QCOW2 files
- [ ] Job status tracking works

---

### **Task 7: Documentation Updates** ⏱️ 20 min

**File 1:** `source/current/api-documentation/SNA-API.md`  
**Add endpoint documentation:**
```markdown
### POST /api/v1/backup/start

**Description:** Initiates a VMware VM backup to NBD targets.

**Request Body:**
\`\`\`json
{
  "job_id": "backup-job-123",
  "vm_name": "pgtest1",
  "vcenter_host": "10.0.100.10",
  "vcenter_user": "administrator@vsphere.local",
  "vcenter_password": "password",
  "vm_path": "/Datacenter/vm/pgtest1",
  "nbd_targets": "2000:nbd://127.0.0.1:10100/export1,2001:nbd://127.0.0.1:10101/export2",
  "backup_type": "full",
  "previous_change_id": ""
}
\`\`\`

**Response (200 OK):**
\`\`\`json
{
  "job_id": "backup-job-123",
  "status": "started",
  "message": "Backup started for pgtest1",
  "started_at": "2025-10-07T16:30:00Z",
  "pid": 12345
}
\`\`\`

**Error Responses:**
- 400: Invalid request (missing required fields)
- 500: Process start failed

**Multi-Disk Support:**  
The `nbd_targets` field contains a comma-separated list of disk mappings in the format:  
`{vmware_disk_key}:nbd://{host}:{port}/{export_name}`

Example for 2-disk VM:  
`"2000:nbd://127.0.0.1:10100/pgtest1-disk0,2001:nbd://127.0.0.1:10101/pgtest1-disk1"`
```

**File 2:** Update `CHANGELOG.md`
```markdown
## [1.4.0] - 2025-10-07

### Added
- **Backup Endpoint:** POST /api/v1/backup/start for multi-disk VMware backups
- Multi-disk NBD targets support
- Job tracking integration for backup operations
- sendense-backup-client process management

### Changed
- SNA API now supports 12 endpoints (was 11)

### Fixed
- End-to-end backup workflow now functional (SHA → SNA communication working)
```

**Acceptance Criteria:**
- [ ] API documentation updated with full endpoint details
- [ ] Request/response examples provided
- [ ] Multi-disk NBD targets format documented
- [ ] CHANGELOG.md updated with version 1.4.0
- [ ] Error codes documented

---

## 📊 **Completion Checklist**

### **Implementation**
- [ ] Request/Response types defined (Task 1)
- [ ] Route registered in setupRoutes() (Task 2)
- [ ] Handler implemented with validation (Task 3)
- [ ] Required imports added (Task 4)
- [ ] No linter errors
- [ ] No simulation code (real operations only)

### **Deployment**
- [ ] Binary built successfully (Task 5)
- [ ] Binary deployed to SNA at /opt/vma/bin/sna-api
- [ ] Systemd service created and running
- [ ] Health check responds correctly

### **Testing**
- [ ] Endpoint availability test passes (Task 6.1)
- [ ] Validation error test passes (Task 6.2)
- [ ] End-to-end backup test passes (Task 6.3)
- [ ] Job status tracking works (Task 6.4)
- [ ] Multi-disk VM backup creates all QCOW2 files
- [ ] Logs written to correct location

### **Documentation**
- [ ] SNA-API.md updated with endpoint docs (Task 7)
- [ ] CHANGELOG.md updated with version 1.4.0
- [ ] Error responses documented
- [ ] Multi-disk format documented

### **Project Rules Compliance**
- [ ] NO simulation code
- [ ] Source in source/current/sna/
- [ ] Follows unified NBD architecture
- [ ] SNA/SHA terminology used (not VMA/OMA)
- [ ] No CloudStack dependencies

---

## 🎯 **Success Metrics**

### **Functional**
- ✅ Endpoint responds 200 OK (not 404)
- ✅ sendense-backup-client process starts
- ✅ Multi-disk backups create all QCOW2 files
- ✅ Job tracking integration works
- ✅ Validation errors return clear messages

### **Quality**
- ✅ No linter errors
- ✅ Code follows existing patterns
- ✅ Comprehensive error handling
- ✅ Logging at appropriate levels
- ✅ API documentation complete

### **Integration**
- ✅ SHA → SNA communication works
- ✅ SSH tunnel forwards NBD traffic
- ✅ qemu-nbd processes accessible from SNA
- ✅ End-to-end backup completes
- ✅ Progress tracking operational

---

## 📚 **Dependencies & Prerequisites**

### **Code Dependencies**
- ✅ Existing SNA API structure (server.go)
- ✅ Job tracking system (JobTracker)
- ✅ Progress parser system
- ✅ sendense-backup-client binary (Task 7.1 complete)

### **Infrastructure Dependencies**
- ✅ SSH tunnel operational (101 NBD ports)
- ✅ SHA qemu-nbd processes on ports 10100-10200
- ✅ VMware vCenter accessible from SNA
- ✅ Systemd on SNA for service management

### **Testing Dependencies**
- ✅ pgtest1 VM available (2 disks: 102GB + 5GB)
- ✅ SHA repository with 467GB available
- ✅ vCenter credentials configured

---

## 🔗 **Related Files**

**To Modify:**
- `source/current/sna/api/server.go` (main implementation)
- `source/current/api-documentation/SNA-API.md` (endpoint docs)
- `CHANGELOG.md` (version 1.4.0 entry)

**To Reference:**
- `source/current/sha/api/handlers/backup_handlers.go` (SHA side implementation)
- `source/current/sna/api/server.go` (existing handlers for pattern matching)
- `project-goals/phases/phase-1-vmware-backup.md` (Task 7 details)
- `job-sheets/2025-10-07-unified-nbd-architecture.md` (architecture context)
- `SESSION-SUMMARY-2025-10-07.md` (current state)

**To Test With:**
- `TESTING-PGTEST1-CHECKLIST.md` (comprehensive test plan)

---

## ⚠️ **Risks & Mitigation**

**Risk 1: Binary Name Mismatch**
- **Risk:** sendense-backup-client not found on SNA
- **Mitigation:** Fallback to /usr/local/bin/migratekit for backwards compatibility
- **Evidence:** Code includes binary existence check

**Risk 2: Process Management**
- **Risk:** Orphaned sendense-backup-client processes
- **Mitigation:** Use job tracker to monitor process lifecycle
- **Future:** Implement cleanup on API shutdown

**Risk 3: Log File Permissions**
- **Risk:** Cannot write to /var/log/sendense/
- **Mitigation:** Create directory with proper permissions, fallback to /tmp
- **Evidence:** Code includes MkdirAll with error handling

**Risk 4: Reverse Tunnel Issue**
- **Risk:** SHA cannot reach SNA API on port 9081
- **Current:** Reverse tunnel disabled due to SSH config issue
- **Workaround:** SHA can directly access SNA:8081 if network allows
- **Future:** Fix SSH PermitListen configuration in next session

---

## 📝 **Notes for Implementation**

1. **Binary Compatibility:** Code checks for both sendense-backup-client and migratekit binaries
2. **Job Tracking:** Reuses existing AddJobWithProgress() method for consistency
3. **Log Location:** /var/log/sendense/ preferred, /tmp fallback
4. **Error Handling:** All errors logged with structured fields
5. **Process Management:** Asynchronous start, PID returned in response
6. **NBD Targets:** Passed as-is from SHA, no parsing required
7. **Incremental Backups:** previous_change_id parameter supported
8. **Security:** Credentials passed in request body, not logged
9. **Testing:** Start with validation tests, then end-to-end with pgtest1
10. **Deployment:** SNA service restart required after deployment

---

## 🎓 **Knowledge Captured**

**What We Learned:**
- SHA multi-disk backup code already working perfectly
- NBD port allocation and qemu-nbd management operational
- SSH tunnel forwards all 101 ports successfully
- **ONLY BLOCKER:** Missing SNA API endpoint

**Architecture Validated:**
- Unified NBD architecture design is correct
- Port-based approach (10100-10200) works
- Multi-disk NBD targets string format effective
- sendense-backup-client accepts --nbd-targets flag

**Next Phase Ready:**
- After this endpoint: Phase 1 Task 7 COMPLETE
- Ready for Task 8: Comprehensive testing & validation
- Ready for production deployment

---

**Phase Owner:** Backend Engineering Team  
**Implementation Lead:** AI Assistant (with user validation)  
**Reviewer:** User (code review before deployment)  
**Last Updated:** October 7, 2025  
**Status:** 🔴 **READY TO START** - All prerequisites complete

