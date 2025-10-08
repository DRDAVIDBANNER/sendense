# Task 2.4 Worker Prompt: Multi-Disk VM Backup Support

**Task:** Implement multi-disk VM backup support (CRITICAL FIX)  
**Job Sheet:** `2025-10-07-unified-nbd-architecture.md` (Task 2.4)  
**Priority:** 🚨 **CRITICAL** - Data corruption risk  
**Estimated Time:** 3-4 hours  
**File:** `sha/api/handlers/backup_handlers.go`

---

## 🚨 CRITICAL ISSUE

**Current Implementation BREAKS VMware Consistency:**
- Only backs up ONE disk per API call
- Requires 3 separate calls for 3-disk VM
- Creates 3 separate VMware snapshots at different times
- **RESULT:** Data corruption, inconsistent backups

**Example of BROKEN Behavior:**
```
POST /api/v1/backups {"vm_name": "db", "disk_id": 0}  → Snapshot at 10:00am
POST /api/v1/backups {"vm_name": "db", "disk_id": 1}  → Snapshot at 10:05am ❌
POST /api/v1/backups {"vm_name": "db", "disk_id": 2}  → Snapshot at 10:10am ❌

Result: Disk 0 has data from 10:00, disk 1 from 10:05, disk 2 from 10:10
        → DATABASE CORRUPTION!
```

---

## ✅ PROOF: System Already Supports This!

**SendenseBackupClient:**
- Line 426: Has `--nbd-targets` flag for multi-disk
- Format: `"disk_key:nbd_url,disk_key:nbd_url"`
- Creates ONE VMware snapshot, reads ALL disks from it

**Replication Workflow:**
- migration.go line 337: Loops through ALL disks
- migration.go line 496: Provisions volumes for ALL disks
- migration.go line 1025: Calls SNA once with multi-disk NBD targets map

**Your Job:** Make backup work like replication does!

---

## 🎯 OBJECTIVE

Change backup from **disk-level** to **VM-level** operations:
- Remove `disk_id` from request (backups are for entire VM)
- Loop through ALL disks for VM
- Allocate NBD port for EACH disk
- Start qemu-nbd for EACH disk  
- Build multi-disk NBD targets string
- Call SNA API once with ALL disks
- Return results for ALL disks

---

## 📋 IMPLEMENTATION STEPS

### **Step 1: Update Request Structure (Lines 56-64)**

**BEFORE (BROKEN):**
```go
type BackupStartRequest struct {
    VMName       string            `json:"vm_name"`
    DiskID       int               `json:"disk_id"`      // ← REMOVE THIS!
    BackupType   string            `json:"backup_type"`
    RepositoryID string            `json:"repository_id"`
    PolicyID     string            `json:"policy_id,omitempty"`
    Tags         map[string]string `json:"tags,omitempty"`
}
```

**AFTER (CORRECT):**
```go
type BackupStartRequest struct {
    VMName       string            `json:"vm_name"`       // VM name (ALL disks)
    BackupType   string            `json:"backup_type"`   // "full" or "incremental"
    RepositoryID string            `json:"repository_id"`
    PolicyID     string            `json:"policy_id,omitempty"`
    Tags         map[string]string `json:"tags,omitempty"`
    // NO disk_id field - backups are VM-level!
}
```

---

### **Step 2: Add New Structs for Multi-Disk Results**

**Add BEFORE BackupResponse (around line 66):**
```go
// DiskBackupResult represents backup result for a single disk
type DiskBackupResult struct {
    DiskID        int    `json:"disk_id"`
    NBDPort       int    `json:"nbd_port"`
    ExportName    string `json:"nbd_export_name"`
    QCOW2Path     string `json:"qcow2_path"`
    QemuNBDPID    int    `json:"qemu_nbd_pid"`
    Status        string `json:"status"`
    ErrorMessage  string `json:"error_message,omitempty"`
}
```

**Update BackupResponse (Lines 66-87):**
```go
type BackupResponse struct {
    BackupID         string              `json:"backup_id"`
    VMContextID      string              `json:"vm_context_id"`
    VMName           string              `json:"vm_name"`
    DiskResults      []DiskBackupResult  `json:"disk_results"`           // NEW: All disks
    NBDTargetsString string              `json:"nbd_targets_string"`     // NEW: For SBC
    BackupType       string              `json:"backup_type"`
    RepositoryID     string              `json:"repository_id"`
    PolicyID         string              `json:"policy_id,omitempty"`
    Status           string              `json:"status"`
    BytesTransferred int64               `json:"bytes_transferred"`
    TotalBytes       int64               `json:"total_bytes"`
    ErrorMessage     string              `json:"error_message,omitempty"`
    CreatedAt        string              `json:"created_at"`
    Tags             map[string]string   `json:"tags,omitempty"`
}
```

---

### **Step 3: Rewrite StartBackup() Method (MAJOR CHANGE)**

**Location:** Around line 126 in backup_handlers.go

**Current code processes ONE disk. New code must process ALL disks.**

**Key Changes:**
1. Query ALL disks for VM (not just one)
2. Loop to allocate ports for each disk
3. Loop to start qemu-nbd for each disk
4. Build NBD targets string
5. Cleanup logic handles ALL disks

**TEMPLATE (Fill in the details):**
```go
func (bh *BackupHandler) StartBackup(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    var req BackupStartRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        bh.sendError(w, http.StatusBadRequest, "invalid request body", err.Error())
        return
    }
    
    // Validate request
    if req.VMName == "" {
        bh.sendError(w, http.StatusBadRequest, "vm_name is required", "")
        return
    }
    
    log.WithFields(log.Fields{
        "vm_name":       req.VMName,
        "backup_type":   req.BackupType,
        "repository_id": req.RepositoryID,
    }).Info("🎯 Starting VM backup (multi-disk)")
    
    // ========================================================================
    // STEP 1: Get VM context
    // ========================================================================
    vmContext, err := bh.vmContextRepo.GetByVMName(req.VMName)
    if err != nil {
        log.WithError(err).Error("Failed to get VM context")
        bh.sendError(w, http.StatusNotFound, "VM not found", err.Error())
        return
    }
    
    // ========================================================================
    // STEP 2: Get ALL disks for VM
    // ========================================================================
    vmDisks, err := bh.vmDiskRepo.GetByVMContext(vmContext.ContextID)
    if err != nil {
        log.WithError(err).Error("Failed to get VM disks")
        bh.sendError(w, http.StatusInternalServerError, "failed to get VM disks", err.Error())
        return
    }
    
    if len(vmDisks) == 0 {
        log.Error("No disks found for VM")
        bh.sendError(w, http.StatusNotFound, "No disks found for VM", 
            "VM has no disks in database - ensure VM discovery completed")
        return
    }
    
    log.WithFields(log.Fields{
        "vm_name":    req.VMName,
        "disk_count": len(vmDisks),
    }).Info("📀 Found disks for multi-disk backup")
    
    // ========================================================================
    // STEP 3: Allocate NBD ports for ALL disks
    // ========================================================================
    diskResults := make([]DiskBackupResult, len(vmDisks))
    allocatedPorts := []int{}
    var allocationErr error
    
    // Cleanup function for failure scenarios
    defer func() {
        if allocationErr != nil {
            log.Info("🧹 Cleaning up allocated resources due to failure")
            // Release all allocated ports
            for _, port := range allocatedPorts {
                bh.portAllocator.Release(port)
            }
            // Stop all started qemu-nbd processes
            for i := range diskResults {
                if diskResults[i].QemuNBDPID > 0 {
                    bh.qemuManager.Stop(diskResults[i].NBDPort)
                }
            }
        }
    }()
    
    for i, vmDisk := range vmDisks {
        exportName := fmt.Sprintf("%s-disk%d", req.VMName, vmDisk.DiskID)
        backupJobID := fmt.Sprintf("backup-%s-disk%d-%d", req.VMName, vmDisk.DiskID, time.Now().Unix())
        
        // Allocate NBD port
        nbdPort, err := bh.portAllocator.Allocate(backupJobID, req.VMName, exportName)
        if err != nil {
            log.WithError(err).WithField("disk_id", vmDisk.DiskID).Error("❌ Failed to allocate NBD port")
            allocationErr = err
            bh.sendError(w, http.StatusServiceUnavailable, "no available NBD ports", err.Error())
            return
        }
        allocatedPorts = append(allocatedPorts, nbdPort)
        
        log.WithFields(log.Fields{
            "disk_id":     vmDisk.DiskID,
            "nbd_port":    nbdPort,
            "export_name": exportName,
        }).Info("✅ NBD port allocated for disk")
        
        diskResults[i] = DiskBackupResult{
            DiskID:     vmDisk.DiskID,
            NBDPort:    nbdPort,
            ExportName: exportName,
            Status:     "port_allocated",
        }
    }
    
    // ========================================================================
    // STEP 4: Start qemu-nbd for ALL disks
    // ========================================================================
    for i := range diskResults {
        vmDisk := vmDisks[i]
        result := &diskResults[i]
        
        // Determine QCOW2 file path
        qcow2Path := filepath.Join("/backup/repository", 
            fmt.Sprintf("%s-disk%d.qcow2", req.VMName, vmDisk.DiskID))
        
        // Start qemu-nbd process
        backupJobID := fmt.Sprintf("backup-%s-disk%d-%d", req.VMName, vmDisk.DiskID, time.Now().Unix())
        qemuProcess, err := bh.qemuManager.Start(
            result.NBDPort,
            result.ExportName,
            qcow2Path,
            backupJobID,
            req.VMName,
            vmDisk.DiskID,
        )
        
        if err != nil {
            log.WithError(err).WithField("disk_id", vmDisk.DiskID).Error("❌ Failed to start qemu-nbd")
            result.Status = "failed"
            result.ErrorMessage = err.Error()
            allocationErr = err
            bh.sendError(w, http.StatusInternalServerError, "failed to start qemu-nbd", err.Error())
            return
        }
        
        result.QCOW2Path = qcow2Path
        result.QemuNBDPID = qemuProcess.PID
        result.Status = "qemu_started"
        
        log.WithFields(log.Fields{
            "disk_id": vmDisk.DiskID,
            "port":    result.NBDPort,
            "pid":     result.QemuNBDPID,
            "qcow2":   qcow2Path,
        }).Info("✅ qemu-nbd started for disk")
    }
    
    // ========================================================================
    // STEP 5: Build NBD targets string for SendenseBackupClient
    // ========================================================================
    // Format: "vmware_disk_key:nbd://host:port/export,vmware_disk_key:nbd://..."
    nbdTargets := []string{}
    for i, result := range diskResults {
        vmDisk := vmDisks[i]
        // Calculate VMware disk key (unit_number + 2000 is VMware standard offset)
        diskKey := vmDisk.UnitNumber + 2000
        nbdURL := fmt.Sprintf("nbd://127.0.0.1:%d/%s", result.NBDPort, result.ExportName)
        nbdTargets = append(nbdTargets, fmt.Sprintf("%d:%s", diskKey, nbdURL))
    }
    nbdTargetsString := strings.Join(nbdTargets, ",")
    
    log.WithFields(log.Fields{
        "nbd_targets":  nbdTargetsString,
        "target_count": len(nbdTargets),
    }).Info("🎯 Built multi-disk NBD targets string")
    
    // ========================================================================
    // STEP 6: Call SNA VMA API (via reverse tunnel on port 9081)
    // ========================================================================
    backupJobID := fmt.Sprintf("backup-%s-%d", req.VMName, time.Now().Unix())
    
    snaReq := map[string]interface{}{
        "vm_name":     req.VMName,
        "nbd_host":    "127.0.0.1",        // Via SSH tunnel
        "nbd_targets": nbdTargetsString,   // ← Multi-disk NBD targets!
        "job_id":      backupJobID,
        "backup_type": req.BackupType,
    }
    
    jsonData, _ := json.Marshal(snaReq)
    snaURL := "http://localhost:9081/api/v1/backup/start"
    
    resp, err := http.Post(snaURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        log.WithError(err).Error("❌ Failed to call SNA VMA API")
        allocationErr = err
        bh.sendError(w, http.StatusInternalServerError, "failed to call SNA API", err.Error())
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        log.WithFields(log.Fields{
            "status": resp.StatusCode,
            "body":   string(body),
        }).Error("❌ SNA VMA API returned error")
        allocationErr = fmt.Errorf("SNA API error: %d", resp.StatusCode)
        bh.sendError(w, http.StatusInternalServerError, "SNA API error", string(body))
        return
    }
    
    log.WithFields(log.Fields{
        "vm_name":      req.VMName,
        "disk_count":   len(diskResults),
        "sna_url":      snaURL,
        "backup_job_id": backupJobID,
    }).Info("✅ SNA VMA API called successfully for multi-disk backup")
    
    // ========================================================================
    // STEP 7: Return response with ALL disk details
    // ========================================================================
    response := BackupResponse{
        BackupID:         backupJobID,
        VMContextID:      vmContext.ContextID,
        VMName:           req.VMName,
        DiskResults:      diskResults,
        NBDTargetsString: nbdTargetsString,
        BackupType:       req.BackupType,
        RepositoryID:     req.RepositoryID,
        PolicyID:         req.PolicyID,
        Status:           "started",
        CreatedAt:        time.Now().Format(time.RFC3339),
        Tags:             req.Tags,
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
    
    log.WithFields(log.Fields{
        "backup_id":  backupJobID,
        "vm_name":    req.VMName,
        "disk_count": len(diskResults),
    }).Info("🎉 Multi-disk VM backup started successfully")
}
```

---

## ✅ TESTING CHECKLIST

After implementation, verify:

**1. Code Compiles:**
```bash
cd /home/oma_admin/sendense/source/current/sha
go build ./cmd/main.go
# Should succeed with no errors
```

**2. Test Request Format:**
```json
POST /api/v1/backups
{
    "vm_name": "test-vm",
    "backup_type": "full",
    "repository_id": "repo-001"
}
```

**3. Expected Response Format:**
```json
{
    "backup_id": "backup-test-vm-1696713600",
    "vm_name": "test-vm",
    "disk_results": [
        {
            "disk_id": 0,
            "nbd_port": 10105,
            "nbd_export_name": "test-vm-disk0",
            "qcow2_path": "/backup/repository/test-vm-disk0.qcow2",
            "qemu_nbd_pid": 12345,
            "status": "qemu_started"
        },
        {
            "disk_id": 1,
            "nbd_port": 10106,
            "nbd_export_name": "test-vm-disk1",
            "qcow2_path": "/backup/repository/test-vm-disk1.qcow2",
            "qemu_nbd_pid": 12346,
            "status": "qemu_started"
        }
    ],
    "nbd_targets_string": "2000:nbd://127.0.0.1:10105/test-vm-disk0,2001:nbd://127.0.0.1:10106/test-vm-disk1",
    "backup_type": "full",
    "status": "started"
}
```

---

## 🚨 CRITICAL SUCCESS CRITERIA

Before reporting "complete", verify:

- [ ] ✅ `disk_id` field REMOVED from BackupStartRequest
- [ ] ✅ `DiskBackupResult` struct added
- [ ] ✅ `BackupResponse` has `disk_results` array
- [ ] ✅ `BackupResponse` has `nbd_targets_string` field
- [ ] ✅ Code queries `vmDiskRepo.GetByVMContext()` for ALL disks
- [ ] ✅ Loop allocates NBD port for EACH disk
- [ ] ✅ Loop starts qemu-nbd for EACH disk
- [ ] ✅ NBD targets string built correctly (format: `key:url,key:url`)
- [ ] ✅ SNA API called ONCE with `nbd_targets` (not per-disk)
- [ ] ✅ Cleanup logic releases ALL ports on failure
- [ ] ✅ Cleanup logic stops ALL qemu-nbd on failure
- [ ] ✅ SHA compiles cleanly (`go build ./cmd/main.go`)
- [ ] ✅ No linter errors
- [ ] ✅ Tested with multi-disk VM (or documented if no test VM available)

---

## ⚠️ COMMON MISTAKES TO AVOID

**1. Don't keep `disk_id` in request!**
   - Backups are VM-level now, not disk-level
   - Remove the field completely

**2. Don't call SNA API per-disk!**
   - ONE call for entire VM with `nbd_targets` string
   - Multiple calls = multiple snapshots = corruption!

**3. Don't forget cleanup!**
   - Use `defer` to cleanup on ANY error
   - Release ports AND stop qemu-nbd processes

**4. Don't hardcode disk keys!**
   - Calculate from `vmDisk.UnitNumber + 2000`
   - This matches VMware's disk key convention

**5. Test compilation!**
   - Don't report "complete" until `go build` succeeds
   - Read any error messages carefully

---

## 📝 REPORTING FORMAT

When complete, provide:

**Summary:**
```
✅ TASK 2.4 COMPLETE - Multi-Disk VM Backup Support

Changes Made:
- Removed disk_id from BackupStartRequest ✅
- Added DiskBackupResult struct ✅  
- Updated BackupResponse with disk_results array ✅
- Queries ALL disks via GetByVMContext() ✅
- Allocates NBD port for each disk ✅
- Starts qemu-nbd for each disk ✅
- Builds NBD targets string correctly ✅
- Calls SNA API once with multi-disk targets ✅
- Cleanup logic handles all disks ✅

Compilation: SUCCESS (34MB binary)
Linter: ZERO errors
Lines Modified: ~250 lines
```

**Evidence:**
- Paste compilation output (should show exit code 0)
- Confirm SHA main binary builds
- List any remaining issues

---

## 🚀 START NOW!

**Begin with Step 1** (Update Request Structure)

Work through steps 1-7 systematically.

Test compilation after each major change.

Report back when complete!

**GO!** 🚀
