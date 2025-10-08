# 🚨 CRITICAL: Multi-Disk VM Backup Architecture Plan

**Date:** October 7, 2025  
**Issue:** Task 2.3 implementation INCOMPLETE - Single-disk backups break VMware consistency  
**Severity:** **CRITICAL** - Can cause data corruption for multi-disk VMs  
**Status:** 🔴 **REQUIRES IMMEDIATE FIX**

---

## 🎯 PROBLEM STATEMENT

### **Current Implementation (BROKEN):**
```go
// Current backup API accepts SINGLE disk
type BackupStartRequest struct {
    VMName     string  `json:"vm_name"`
    DiskID     int     `json:"disk_id"`      // ← SINGLE DISK ONLY!
    BackupType string  `json:"backup_type"`
}

// For 3-disk VM, requires 3 separate API calls:
POST /api/v1/backups { "vm_name": "db-server", "disk_id": 0 }  // Snapshot at T0
POST /api/v1/backups { "vm_name": "db-server", "disk_id": 1 }  // Snapshot at T1
POST /api/v1/backups { "vm_name": "db-server", "disk_id": 2 }  // Snapshot at T2
```

**Why This Is BROKEN:**
1. ❌ **Three separate VMware snapshots** (T0, T1, T2)
2. ❌ **Inconsistent data** - disk 0 is from 10:00am, disk 1 from 10:05am, disk 2 from 10:10am
3. ❌ **Database corruption** - application sees inconsistent state
4. ❌ **Violates VMware design** - snapshots are VM-level, not disk-level

### **VMware Architecture (CORRECT):**
```
VM Snapshot (SINGLE point in time)
├── Disk 0 (consistent state at T0)
├── Disk 1 (consistent state at T0)
└── Disk 2 (consistent state at T0)
```

**VMware guarantees:**
- ✅ All disks captured at SAME instant
- ✅ Application consistency (with quiesce)
- ✅ Crash consistency (minimum)
- ✅ Safe restore point

---

## ✅ PROOF: SendenseBackupClient ALREADY SUPPORTS MULTI-DISK

**Evidence from code review:**

### **1. Multi-Disk CLI Flag (main.go:426)**
```go
rootCmd.PersistentFlags().StringVar(&nbdTargets, "nbd-targets", "", 
    "NBD targets for multi-disk VMs (format: vm_disk_id:nbd_url,vm_disk_id:nbd_url)")
```

### **2. Multi-Disk NBD Export Determination (nbd.go:260-277)**
```go
func (t *NBDTarget) determineNBDExportForDisk(ctx context.Context) (string, error) {
    // Check if multi-disk targets are provided
    nbdTargetsStr := ctx.Value("nbdTargets")
    if nbdTargetsStr != nil && nbdTargetsStr.(string) != "" {
        // Parse multi-disk NBD targets: "vm_disk_id:nbd_url,vm_disk_id:nbd_url"
        return t.parseMultiDiskNBDTargets(ctx, nbdTargetsStr.(string))
    }
    // Fallback to single-disk mode
}
```

### **3. Multi-Disk Target Parser (nbd.go:280-291)**
```go
func (t *NBDTarget) parseMultiDiskNBDTargets(ctx context.Context, nbdTargetsStr string) (string, error) {
    currentDiskID := t.getCurrentDiskID()
    log.Printf("🎯 Multi-disk mode: Looking for NBD target for disk %s (VMware key: %d)", 
        currentDiskID, t.Disk.Key)
    
    // Parse NBD targets: "2000:nbd://...,2001:nbd://..." (VMware disk keys)
    targetPairs := strings.Split(nbdTargetsStr, ",")
    // ... matches disk to correct NBD export
}
```

### **4. Single VM Snapshot for All Disks (vmware_nbdkit.go:66)**
```go
func (s *NbdkitServers) createSnapshot(ctx context.Context) error {
    task, err := s.VirtualMachine.CreateSnapshot(ctx, "migratekit", 
        "Ephemeral snapshot for MigrateKit", false, s.VddkConfig.Quiesce)
    // ← ONE snapshot for ENTIRE VM (all disks)
}
```

**Conclusion:** SendenseBackupClient is READY for multi-disk. SHA API is NOT.

---

## ✅ PROOF: Replication ALREADY HANDLES MULTI-DISK CORRECTLY

**Evidence from migration.go:**

### **1. Loop Through All Disks (migration.go:337)**
```go
for i, disk := range req.SourceVM.Disks {
    vmDisk := &database.VMDisk{
        JobID:       &req.JobID,
        VMContextID: vmContextID,
        DiskID:      disk.ID,
        VMDKPath:    disk.Path,
        // ... stores ALL disks
    }
    m.vmDiskRepo.Create(vmDisk)
}
```

### **2. Provision All OSSEA Volumes (migration.go:496-620)**
```go
func (m *MigrationEngine) provisionOSSEAVolumes(ctx context.Context, req *MigrationRequest) ([]VolumeProvisionResult, error) {
    vmDisks, err := m.vmDiskRepo.GetByJobID(req.JobID)  // Get ALL disks
    
    results := []VolumeProvisionResult{}
    for _, vmDisk := range vmDisks {  // Loop ALL disks
        // Create volume for EACH disk
        operation, err := volumeClient.CreateVolume(ctx, createReq)
        // ...
        results = append(results, result)
    }
    return results, nil  // Returns ALL disk results
}
```

### **3. Create NBD Exports for All Disks (migration.go:1200-1334)**
```go
func (m *MigrationEngine) getOrCreateNBDExports(req *MigrationRequest, attachResults []VolumeMountResult) ([]*nbd.ExportInfo, error) {
    // Queries Volume Daemon for ALL NBD exports (auto-created during attach)
    nbdExports, err := volumeClient.ListNBDExports(ctx)
    
    // Returns exports for ALL disks
    return nbdExports, nil
}
```

### **4. Call SNA with ALL Disk Details (migration.go:1025-1093)**
```go
func (m *MigrationEngine) initiateVMwareReplication(req *MigrationRequest, nbdExports []*nbd.ExportInfo) error {
    // Builds NBD targets map for ALL disks
    nbdTargetsMap := make(map[string]string)
    for _, export := range nbdExports {  // ALL exports
        diskKey := // ... VMware disk key
        nbdTargetsMap[diskKey] = fmt.Sprintf("nbd://%s:%d/%s", ...)
    }
    
    // Sends to SNA with multi-disk targets
    snaReq := SNAReplicationRequest{
        NBDTargets: nbdTargetsMap,  // ← Multi-disk NBD map!
    }
    
    http.Post(snaAPIURL, ...)  // One call for ALL disks
}
```

**Conclusion:** Replication correctly handles multi-disk in ONE operation from ONE snapshot.

---

## 🔧 REQUIRED FIX: Multi-Disk Backup API

### **Architecture Change: VM-Level Backup, Not Disk-Level**

**New API Contract:**
```go
// ❌ OLD (BROKEN):
POST /api/v1/backups
{
    "vm_name": "db-server",
    "disk_id": 0,                    // ← Single disk only!
    "backup_type": "full"
}

// ✅ NEW (CORRECT):
POST /api/v1/backups/vm
{
    "vm_name": "db-server",          // VM name (all disks)
    "backup_type": "full",
    "repository_id": "repo-001"
}

// Returns:
{
    "backup_job_id": "backup-12345",
    "vm_name": "db-server",
    "disk_results": [
        {
            "disk_id": 0,
            "nbd_port": 10105,
            "nbd_export_name": "db-server-disk0",
            "qcow2_path": "/repo/db-server-disk0.qcow2",
            "status": "started"
        },
        {
            "disk_id": 1,
            "nbd_port": 10106,
            "nbd_export_name": "db-server-disk1",
            "qcow2_path": "/repo/db-server-disk1.qcow2",
            "status": "started"
        },
        {
            "disk_id": 2,
            "nbd_port": 10107,
            "nbd_export_name": "db-server-disk2",
            "qcow2_path": "/repo/db-server-disk2.qcow2",
            "status": "started"
        }
    ],
    "nbd_targets_string": "2000:nbd://127.0.0.1:10105/db-server-disk0,2001:nbd://127.0.0.1:10106/db-server-disk1,2002:nbd://127.0.0.1:10107/db-server-disk2",
    "status": "started"
}
```

---

## 📋 IMPLEMENTATION PLAN

### **Task 2.4: Multi-Disk VM Backup (NEW - CRITICAL)**

**File:** `sha/api/handlers/backup_handlers.go`

**Changes Required:**

#### **1. Update Request Structure**
```go
// Remove disk_id field (it's VM-level now)
type BackupStartRequest struct {
    VMName       string            `json:"vm_name"`       // Required: VM name (ALL disks)
    BackupType   string            `json:"backup_type"`   // Required: "full" or "incremental"
    RepositoryID string            `json:"repository_id"` // Required: Target repository
    PolicyID     string            `json:"policy_id,omitempty"`
    Tags         map[string]string `json:"tags,omitempty"`
}
```

#### **2. Update Response Structure**
```go
type DiskBackupResult struct {
    DiskID        int    `json:"disk_id"`
    NBDPort       int    `json:"nbd_port"`
    ExportName    string `json:"nbd_export_name"`
    QCOW2Path     string `json:"qcow2_path"`
    QemuNBDPID    int    `json:"qemu_nbd_pid"`
    Status        string `json:"status"`
    ErrorMessage  string `json:"error_message,omitempty"`
}

type BackupResponse struct {
    BackupID         string              `json:"backup_id"`
    VMContextID      string              `json:"vm_context_id"`
    VMName           string              `json:"vm_name"`
    DiskResults      []DiskBackupResult  `json:"disk_results"`           // NEW: All disks
    NBDTargetsString string              `json:"nbd_targets_string"`     // NEW: For SBC
    BackupType       string              `json:"backup_type"`
    RepositoryID     string              `json:"repository_id"`
    Status           string              `json:"status"`
    CreatedAt        string              `json:"created_at"`
    Tags             map[string]string   `json:"tags,omitempty"`
}
```

#### **3. Core Logic Changes**
```go
func (bh *BackupHandler) StartBackup(w http.ResponseWriter, r *http.Request) {
    // ... validate request, get VM context ...
    
    // ========================================
    // STEP 1: Get ALL disks for VM
    // ========================================
    vmDisks, err := bh.vmDiskRepo.GetByVMContext(vmContext.ContextID)
    if err != nil || len(vmDisks) == 0 {
        bh.sendError(w, http.StatusNotFound, "No disks found for VM", err.Error())
        return
    }
    
    log.WithFields(log.Fields{
        "vm_name":    req.VMName,
        "disk_count": len(vmDisks),
    }).Info("📀 Found disks for multi-disk backup")
    
    // ========================================
    // STEP 2: Allocate NBD ports for ALL disks
    // ========================================
    diskResults := []DiskBackupResult{}
    allocatedPorts := []int{}
    
    // Cleanup function for failure scenarios
    defer func() {
        if err != nil {
            log.Info("🧹 Cleaning up allocated resources due to failure")
            for i, port := range allocatedPorts {
                bh.portAllocator.Release(port)
                if i < len(diskResults) && diskResults[i].QemuNBDPID > 0 {
                    bh.qemuManager.Stop(port)
                }
            }
        }
    }()
    
    for _, vmDisk := range vmDisks {
        exportName := fmt.Sprintf("%s-disk%d", req.VMName, vmDisk.DiskID)
        backupJobID := fmt.Sprintf("backup-%s-disk%d-%d", req.VMName, vmDisk.DiskID, time.Now().Unix())
        
        // Allocate port
        nbdPort, err := bh.portAllocator.Allocate(backupJobID, req.VMName, exportName)
        if err != nil {
            log.WithError(err).Error("❌ Failed to allocate NBD port for disk")
            return // Defer will clean up
        }
        allocatedPorts = append(allocatedPorts, nbdPort)
        
        log.WithFields(log.Fields{
            "disk_id":  vmDisk.DiskID,
            "nbd_port": nbdPort,
            "export":   exportName,
        }).Info("✅ NBD port allocated for disk")
        
        diskResults = append(diskResults, DiskBackupResult{
            DiskID:     vmDisk.DiskID,
            NBDPort:    nbdPort,
            ExportName: exportName,
            Status:     "port_allocated",
        })
    }
    
    // ========================================
    // STEP 3: Start qemu-nbd for ALL disks
    // ========================================
    for i := range diskResults {
        vmDisk := vmDisks[i]
        result := &diskResults[i]
        
        qcow2Path := filepath.Join("/backup/repository", 
            fmt.Sprintf("%s-disk%d.qcow2", req.VMName, vmDisk.DiskID))
        
        // Start qemu-nbd
        qemuProcess, err := bh.qemuManager.Start(
            result.NBDPort,
            result.ExportName,
            qcow2Path,
            backupJobID,
            req.VMName,
            vmDisk.DiskID,
        )
        
        if err != nil {
            log.WithError(err).Error("❌ Failed to start qemu-nbd for disk")
            result.Status = "failed"
            result.ErrorMessage = err.Error()
            return // Defer will clean up
        }
        
        result.QCOW2Path = qcow2Path
        result.QemuNBDPID = qemuProcess.PID
        result.Status = "qemu_started"
        
        log.WithFields(log.Fields{
            "disk_id":  vmDisk.DiskID,
            "port":     result.NBDPort,
            "pid":      result.QemuNBDPID,
            "qcow2":    qcow2Path,
        }).Info("✅ qemu-nbd started for disk")
    }
    
    // ========================================
    // STEP 4: Build NBD targets string for SendenseBackupClient
    // ========================================
    // Format: "vmware_disk_key:nbd://host:port/export,..."
    nbdTargets := []string{}
    for i, result := range diskResults {
        vmDisk := vmDisks[i]
        // Use VMware disk key (from vm_disks.unit_number or calculated)
        diskKey := vmDisk.UnitNumber + 2000  // VMware standard offset
        nbdURL := fmt.Sprintf("nbd://127.0.0.1:%d/%s", result.NBDPort, result.ExportName)
        nbdTargets = append(nbdTargets, fmt.Sprintf("%d:%s", diskKey, nbdURL))
    }
    nbdTargetsString := strings.Join(nbdTargets, ",")
    
    log.WithField("nbd_targets", nbdTargetsString).Info("🎯 Built multi-disk NBD targets string")
    
    // ========================================
    // STEP 5: Call SNA VMA API (via reverse tunnel)
    // ========================================
    snaReq := map[string]interface{}{
        "vm_name":     req.VMName,
        "nbd_host":    "127.0.0.1",      // Via SSH tunnel
        "nbd_targets": nbdTargetsString, // ← Multi-disk!
        "job_id":      backupJobID,
        "backup_type": req.BackupType,
    }
    
    snaURL := "http://localhost:9081/api/v1/backup/start"
    jsonData, _ := json.Marshal(snaReq)
    
    resp, err := http.Post(snaURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        log.WithError(err).Error("❌ Failed to call SNA VMA API")
        return // Defer will clean up
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        log.WithField("status", resp.StatusCode).Error("❌ SNA VMA API returned error")
        return // Defer will clean up
    }
    
    log.WithFields(log.Fields{
        "vm_name":    req.VMName,
        "disk_count": len(diskResults),
        "sna_url":    snaURL,
    }).Info("✅ SNA VMA API called successfully for multi-disk backup")
    
    // ========================================
    // STEP 6: Return response with ALL disk details
    // ========================================
    response := BackupResponse{
        BackupID:         backupJobID,
        VMContextID:      vmContext.ContextID,
        VMName:           req.VMName,
        DiskResults:      diskResults,
        NBDTargetsString: nbdTargetsString,
        BackupType:       req.BackupType,
        RepositoryID:     req.RepositoryID,
        Status:           "started",
        CreatedAt:        time.Now().Format(time.RFC3339),
        Tags:             req.Tags,
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}
```

---

## ✅ SUCCESS CRITERIA

**Before Approval:**
- [ ] API accepts VM name only (no disk_id in request)
- [ ] Loops through ALL disks for VM
- [ ] Allocates NBD port for EACH disk
- [ ] Starts qemu-nbd for EACH disk
- [ ] Builds multi-disk NBD targets string
- [ ] Calls SNA VMA API once with ALL disk targets
- [ ] Returns disk results for ALL disks
- [ ] Cleanup on failure releases ALL ports and stops ALL qemu-nbd
- [ ] Compiles cleanly
- [ ] Database queries work (GetByVMContext)

**VMware Consistency Guarantee:**
- [ ] SNA creates ONE VM snapshot
- [ ] ALL disks backed up from SAME snapshot
- [ ] Application consistency maintained
- [ ] No data corruption risk

---

## 📊 COMPARISON: Before vs After

### **Before (BROKEN):**
```
API Call 1: Backup disk 0 → Snapshot at 10:00am → Port 10105
API Call 2: Backup disk 1 → Snapshot at 10:05am → Port 10106  ← Inconsistent!
API Call 3: Backup disk 2 → Snapshot at 10:10am → Port 10107  ← Inconsistent!

Result: Database corruption, inconsistent backup
```

### **After (CORRECT):**
```
API Call: Backup VM (all disks) → ONE Snapshot at 10:00am
  ├── Disk 0 → Port 10105 → qemu-nbd → db-server-disk0.qcow2
  ├── Disk 1 → Port 10106 → qemu-nbd → db-server-disk1.qcow2
  └── Disk 2 → Port 10107 → qemu-nbd → db-server-disk2.qcow2

SBC receives: "2000:nbd://127.0.0.1:10105/db-server-disk0,2001:nbd://127.0.0.1:10106/db-server-disk1,2002:nbd://127.0.0.1:10107/db-server-disk2"

SBC creates: ONE VMware snapshot, reads ALL disks from SAME snapshot

Result: Consistent backup, safe to restore
```

---

## 🚨 CRITICAL IMPACT

**Data Loss Risk:** HIGH  
**Customer Impact:** HIGH  
**Complexity:** MEDIUM  
**Priority:** **CRITICAL** - Must be fixed before Task 2.3 approval

**Why This Matters:**
1. Veeam does this correctly (VM-level backups)
2. VMware designed snapshots for VM-level consistency
3. Database/application workloads REQUIRE this
4. Current implementation can cause silent data corruption
5. Customers will lose data during restores

---

## ✅ APPROVAL PLAN

1. **REJECT Task 2.3 as incomplete** (missing multi-disk)
2. **Create Task 2.4** (Multi-Disk VM Backup)
3. **Update job sheet** with new task
4. **Implement fix** per plan above
5. **Test with multi-disk VM**
6. **Re-approve Phase 2** when fixed

---

**STATUS:** 🔴 **AWAITING USER APPROVAL TO PROCEED**

**Next Steps:**
1. User reviews this plan
2. User approves approach
3. Update job sheet with Task 2.4
4. Implement multi-disk support
5. Test and validate
6. Final approval

---

**Project Overseer Signature:** Awaiting approval on October 7, 2025
