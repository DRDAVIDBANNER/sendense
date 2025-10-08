# Module 04: Cross-Platform Restore Engine

**Module ID:** MOD-04  
**Status:** 🟡 **PLANNED** (Phase 4)  
**Priority:** Critical (Enterprise Tier Enabler)  
**Dependencies:** Module 01 (VMware), Module 02 (CloudStack), Module 03 (Storage)  
**Owner:** Platform Engineering Team

---

## 🎯 Module Purpose

Universal restore engine that can take any backup and restore it to any supported platform (cross-platform "ascend" operations).

**Key Capabilities:**
- **Format Conversion:** VMDK ↔ QCOW2 ↔ VHD ↔ RAW conversion pipeline
- **Metadata Translation:** VM specs between platforms (CPU, RAM, network, storage)
- **Driver Injection:** Platform-specific drivers (VirtIO, VMware Tools, Hyper-V IS)
- **Target Platform APIs:** Native integration with all target platforms
- **Compatibility Matrix:** Automatic validation of source→target compatibility

**Strategic Value:**
- **Enterprise Tier Unlock:** Enables $25/VM pricing tier
- **Vendor Lock-in Breaker:** True platform independence
- **Competitive Moat:** Few vendors offer true cross-platform restore

---

## 🏗️ Restore Engine Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ CROSS-PLATFORM RESTORE ENGINE                               │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Source: Any Platform Backup                                │
│  ┌─────────┬─────────┬─────────┬─────────┬─────────┐       │
│  │ VMware  │CloudStck│ Hyper-V │ AWS EC2 │ Nutanix │       │
│  │ Backup  │ Backup  │ Backup  │ Backup  │ Backup  │       │
│  │(QCOW2)  │(QCOW2)  │(QCOW2)  │(QCOW2)  │(QCOW2)  │       │
│  └─────────┴─────────┴─────────┴─────────┴─────────┘       │
│                        ↓                                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                RESTORE ORCHESTRATOR                    │ │
│  │                                                        │ │
│  │  Step 1: Source Analysis                               │ │
│  │  ├─ Parse backup metadata                              │ │
│  │  ├─ Extract VM specifications                          │ │
│  │  ├─ Identify source platform                          │ │
│  │  └─ Validate backup integrity                          │ │
│  │                                                        │ │
│  │  Step 2: Target Validation                             │ │
│  │  ├─ Check target platform capabilities                │ │
│  │  ├─ Validate resource requirements                    │ │
│  │  ├─ Verify network/storage mapping                     │ │
│  │  └─ Estimate restore time and cost                     │ │
│  │                                                        │ │
│  │  Step 3: Format Conversion                             │ │
│  │  ├─ Convert disk format (qemu-img)                    │ │
│  │  ├─ Inject platform drivers                           │ │
│  │  ├─ Update boot configuration                          │ │
│  │  └─ Optimize for target platform                       │ │
│  │                                                        │ │
│  │  Step 4: Target Deployment                             │ │
│  │  ├─ Create VM on target platform                      │ │
│  │  ├─ Configure networking                               │ │
│  │  ├─ Attach converted storage                          │ │
│  │  └─ Start and validate VM                             │ │
│  └────────────────────────────────────────────────────────┘ │
│                        ↓                                     │
│  Target: Any Platform VM                                    │
│  ┌─────────┬─────────┬─────────┬─────────┬─────────┐       │
│  │ VMware  │CloudStck│ Hyper-V │ AWS EC2 │ Azure   │       │
│  │   VM    │   VM    │   VM    │Instance │   VM    │       │
│  │(Running)│(Running)│(Running)│(Running)│(Running)│       │
│  └─────────┴─────────┴─────────┴─────────┴─────────┘       │
└──────────────────────────────────────────────────────────────┘
```

**This is the crown jewel - the module that enables true platform independence and unlocks the Enterprise pricing tier.**

---

**Module Owner:** Cross-Platform Engineering Team  
**Last Updated:** October 8, 2025  
**Status:** 🟡 Planned - Critical Business Enabler (Enterprise Tier $25/VM)

---

## 🔍 CRITICAL: Phase 1 Backup System Context (October 2025)

**⚠️ MUST READ BEFORE IMPLEMENTING RESTORE ENGINE**

Phase 1 VMware backups are now **PRODUCTION READY** with a complete VM-centric backup architecture. Understanding this system is **CRITICAL** for restore implementation.

### **Database Architecture** 📊

**Master Context Table:** `vm_backup_contexts`
```sql
-- One record per VM+repository combination
CREATE TABLE vm_backup_contexts (
  context_id VARCHAR(64) PRIMARY KEY,          -- ctx-backup-{vm_name}-{timestamp}
  vm_name VARCHAR(255),
  vmware_vm_id VARCHAR(255),                   -- Original VMware UUID
  vm_path VARCHAR(500),                        -- VMware inventory path
  vcenter_host VARCHAR(255),                   -- Source vCenter
  datacenter VARCHAR(255),                     -- Source datacenter
  repository_id VARCHAR(64),                   -- FK to backup_repositories
  total_backups_run INT,
  successful_backups INT,
  last_backup_id VARCHAR(64),
  last_backup_type ENUM('full','incremental'),
  last_backup_at TIMESTAMP,
  UNIQUE KEY (vm_name, repository_id)
);
```

**Per-Disk Tracking:** `backup_disks`
```sql
-- One record per backup per disk (multi-disk support)
CREATE TABLE backup_disks (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  vm_backup_context_id VARCHAR(64),            -- FK to vm_backup_contexts
  backup_job_id VARCHAR(64),                   -- FK to backup_jobs (parent)
  disk_index INT,                              -- 0, 1, 2... (array index)
  vmware_disk_key INT,                         -- 2000, 2001, 2002... (VMware key)
  size_gb BIGINT,
  unit_number INT,
  disk_change_id VARCHAR(255),                 -- VMware CBT change ID (CRITICAL!)
  qcow2_path VARCHAR(512),                     -- /mnt/.../disk-0/backup-xxx.qcow2
  bytes_transferred BIGINT,
  status ENUM('pending','running','completed','failed'),
  error_message TEXT,
  completed_at TIMESTAMP,
  UNIQUE KEY (backup_job_id, disk_index),
  INDEX idx_change_id_lookup (vm_backup_context_id, disk_index, status)
);
```

**Parent Job Tracker:** `backup_jobs`
```sql
-- One record per multi-disk backup operation
CREATE TABLE backup_jobs (
  id VARCHAR(64) PRIMARY KEY,                  -- backup-{vm_name}-{timestamp}
  vm_backup_context_id VARCHAR(64),            -- FK to vm_backup_contexts
  vm_name VARCHAR(255),
  backup_type ENUM('full','incremental','differential'),
  status ENUM('pending','running','completed','failed','cancelled'),
  repository_id VARCHAR(64),
  repository_path VARCHAR(512),
  -- DEPRECATED (use backup_disks instead):
  -- disk_id, change_id (moved to backup_disks for per-disk tracking)
  created_at TIMESTAMP,
  completed_at TIMESTAMP
);
```

**Backup Chain Metadata:** `backup_chains`
```sql
-- Tracks full + incrementals per disk
CREATE TABLE backup_chains (
  id VARCHAR(64) PRIMARY KEY,                  -- chain-{context_id}-disk{N}
  vm_context_id VARCHAR(64),                   -- FK to vm_backup_contexts
  disk_id INT,                                 -- Disk index
  full_backup_id VARCHAR(64),                  -- First full backup
  latest_backup_id VARCHAR(64),                -- Most recent backup
  total_backups INT,                           -- Count of backups in chain
  total_size_bytes BIGINT,                     -- Sum of all QCOW2 file sizes
  UNIQUE KEY (vm_context_id, disk_id),
  FOREIGN KEY (vm_context_id) REFERENCES vm_backup_contexts(context_id) ON DELETE CASCADE
);
```

### **QCOW2 Backing Chain Structure** 💾

**File Layout:**
```
/mnt/sendense-backups/
└── ctx-pgtest1-20251006-203401/                 # VM backup context directory
    ├── disk-0/                                   # First disk
    │   ├── backup-pgtest1-disk0-20251008-192431.qcow2  # Full backup (19GB)
    │   ├── backup-pgtest1-disk0-20251008-200751.qcow2  # Incremental 1 (58MB)
    │   ├── backup-pgtest1-disk0-20251008-201358.qcow2  # Incremental 2 (30MB)
    │   └── backup-pgtest1-disk0-20251008-201743.qcow2  # Incremental 3 (34MB)
    └── disk-1/                                   # Second disk
        ├── backup-pgtest1-disk1-20251008-192431.qcow2  # Full backup (97MB)
        ├── backup-pgtest1-disk1-20251008-200751.qcow2  # Incremental 1 (97MB)
        ├── backup-pgtest1-disk1-20251008-201358.qcow2  # Incremental 2 (97MB)
        └── backup-pgtest1-disk1-20251008-201743.qcow2  # Incremental 3 (97MB)
```

**Backing Chain Verification:**
```bash
# All incrementals point to full backup as backing file
qemu-img info backup-pgtest1-disk0-20251008-200751.qcow2
  # backing file: .../backup-pgtest1-disk0-20251008-192431.qcow2

qemu-img info backup-pgtest1-disk0-20251008-201358.qcow2
  # backing file: .../backup-pgtest1-disk0-20251008-192431.qcow2
```

**🚨 CRITICAL FOR RESTORE:**
- ALL incrementals in a chain point to the SAME full backup (no chain of chains)
- To restore ANY point in time, you need: `full_backup.qcow2` + `specific_incremental.qcow2`
- QCOW2 overlay mechanism handles the merge automatically when you mount the incremental

### **Key APIs for Restore Implementation** 🔌

**1. List Backups for VM:**
```http
GET /api/v1/backups?vm_name=pgtest1&status=completed
Response: {
  "backups": [
    {
      "backup_id": "backup-pgtest1-1759947871",
      "vm_name": "pgtest1",
      "backup_type": "full",
      "status": "completed",
      "created_at": "2025-10-08T19:24:31Z",
      "disk_results": [
        { "disk_id": 0, "qcow2_path": ".../disk-0/backup-xxx.qcow2" },
        { "disk_id": 1, "qcow2_path": ".../disk-1/backup-xxx.qcow2" }
      ]
    }
  ]
}
```

**2. Get Backup Chain:**
```http
GET /api/v1/backups/chain?vm_name=pgtest1
Response: {
  "chain_id": "chain-ctx-backup-pgtest1-...-disk0",
  "full_backup_id": "backup-pgtest1-disk0-20251008-192431",
  "backups": [
    { "id": "backup-...-192431", "type": "full", "size_bytes": 19643564032 },
    { "id": "backup-...-200751", "type": "incremental", "size_bytes": 60555264 },
    { "id": "backup-...-201358", "type": "incremental", "size_bytes": 31129600 },
    { "id": "backup-...-201743", "type": "incremental", "size_bytes": 34734080 }
  ],
  "total_backups": 4,
  "total_size_bytes": 19769982976
}
```

**3. Query Per-Disk Backup Details:**
```sql
-- Get all backup disks for a specific backup job
SELECT 
  disk_index,
  vmware_disk_key,
  size_gb,
  disk_change_id,
  qcow2_path,
  status
FROM backup_disks 
WHERE backup_job_id = 'backup-pgtest1-1759947871'
ORDER BY disk_index;

-- Result:
-- disk_index=0, qcow2_path=.../disk-0/backup-pgtest1-disk0-20251008-192431.qcow2
-- disk_index=1, qcow2_path=.../disk-1/backup-pgtest1-disk1-20251008-192431.qcow2
```

**4. Get VM Context (Source VM Metadata):**
```sql
SELECT 
  vm_name,
  vmware_vm_id,
  vm_path,
  vcenter_host,
  datacenter,
  last_backup_type,
  last_backup_at
FROM vm_backup_contexts 
WHERE vm_name = 'pgtest1';
```

### **Restore Implementation Guidelines** 🛠️

**Step 1: Query Backup Chain**
```go
// Find the backup to restore
backups, err := backupAPI.ListBackups(vmName, "completed")
selectedBackup := backups[0] // User selects which backup point

// Get ALL disks for this backup (multi-disk support!)
var disks []BackupDisk
db.Where("backup_job_id = ?", selectedBackup.ID).
   Order("disk_index").
   Find(&disks)

// For each disk, get QCOW2 path
for _, disk := range disks {
    qcow2Path := disk.QCOW2Path
    // If incremental, need parent too (handled by QCOW2 overlay)
}
```

**Step 2: Mount QCOW2 for Access**
```bash
# For incremental, QCOW2 automatically merges with backing file
qemu-nbd -r -c /dev/nbd0 /path/to/incremental-backup.qcow2

# Mount filesystem
mount -o ro /dev/nbd0p1 /mnt/restore

# Access files or convert to target format
```

**Step 3: Multi-Disk Restore Considerations**
- Each disk has its own QCOW2 chain (independent)
- Must restore ALL disks to recreate original VM
- Disk order matters: `disk_index` 0, 1, 2... corresponds to original VM disk order
- `vmware_disk_key` (2000, 2001, 2002) maps to VMware's internal disk numbering

**Step 4: Target Platform Conversion**
```bash
# Convert QCOW2 to target format
qemu-img convert -f qcow2 -O vmdk backup.qcow2 output.vmdk      # VMware
qemu-img convert -f qcow2 -O qcow2 backup.qcow2 output.qcow2    # CloudStack (native)
qemu-img convert -f qcow2 -O vpc backup.qcow2 output.vhd        # Hyper-V
qemu-img convert -f qcow2 -O raw backup.qcow2 output.img        # Raw/AWS
```

### **Performance Considerations** ⚡

**Proven Results (Phase 1 Testing):**
- Full backup: ~19GB for 102GB thin-provisioned VMware disk
- Incremental efficiency: **99.7% size reduction** (58MB vs 19GB!)
- Multi-disk: 5 backups × 2 disks = 10 QCOW2 files, ~18.8GB total
- QCOW2 overhead: Minimal (sparse allocation, efficient storage)

**Restore Speed Estimates:**
- Local repository restore: Limited by disk I/O (~500 MB/s)
- Network restore: Limited by SSH tunnel bandwidth (~100-150 MB/s)
- Format conversion: CPU-bound (qemu-img ~200 MB/s)

### **Common Pitfalls to Avoid** ⚠️

1. **DON'T assume single-disk VMs** - Multi-disk is the norm
2. **DON'T query deprecated fields** - Use `backup_disks.disk_change_id` NOT `backup_jobs.change_id`
3. **DON'T break QCOW2 chains** - Moving/renaming backing files breaks incrementals
4. **DON'T ignore disk_index** - Order matters for proper VM reconstruction
5. **DON'T forget vm_backup_contexts** - Links to source VM metadata (vCenter, datacenter, etc.)

### **Documentation References** 📚

- **API Reference:** `source/current/api-documentation/API_REFERENCE.md`
- **Database Schema:** `source/current/api-documentation/DB_SCHEMA.md`
- **Phase 1 Status:** `project-goals/phases/phase-1-vmware-backup.md`
- **Context Helper:** `start_here/PHASE_1_CONTEXT_HELPER.md`
- **Changelog:** `start_here/CHANGELOG.md` (v2.16.0 - v2.22.0)

---

