# Phase 1 Context Helper
**Purpose:** Quick reference for AI sessions working on Phase 1 (VMware Backup Implementation)  
**Status Location:** See `project-goals/phases/phase-1-vmware-backup.md` for current state  
**Last Updated:** October 8, 2025

---

## 🎯 WHAT IS PHASE 1?

VMware virtual machine backup implementation using VMware CBT (Changed Block Tracking) to QCOW2 format on local storage.

**Business Goal:** Back up VMware VMs from vCenter to SHA local repository with incremental capability.

**Technical Goal:** Full + incremental backups using NBD protocol, QCOW2 storage format, VMware change tracking.

---

## 📊 CURRENT STATUS REFERENCES

**DO NOT rely on this document for current status. Always check:**

1. **Phase 1 Status:** `/home/oma_admin/sendense/project-goals/phases/phase-1-vmware-backup.md`
2. **Recent Job Sheets:** `/home/oma_admin/sendense/job-sheets/` (sort by date)
3. **Changelog:** `/home/oma_admin/sendense/start_here/CHANGELOG.md`
4. **Active Binaries:** Check symlinks in `/usr/local/bin/` and actual files in `source/builds/`

---

## 🏗️ ARCHITECTURE OVERVIEW

```
VMware vCenter (ESXi)
    ↓ (NBD export via snapshot)
SNA (10.0.100.231) - sendense-backup-client
    ↓ (SSH tunnel port 443 → local ports 10100-10200)
SHA (localhost/oma_admin) - qemu-nbd processes
    ↓ (Write QCOW2 files)
Local Repository (/backup/repository/)
```

### **Components:**

**SHA Side (OMA in code):**
- **API Handler:** `sha/api/handlers/backup_handlers.go` - REST API for backups
- **Backup Engine:** `sha/workflows/backup.go` - Orchestrates backup workflow
- **NBD Manager:** `sha/services/qemu_nbd_manager.go` - Manages qemu-nbd processes
- **Port Allocator:** `sha/services/nbd_port_allocator.go` - Allocates ports 10100-10200
- **QCOW2 Manager:** `sha/storage/qcow2_manager.go` - Creates full/incremental QCOW2s
- **Chain Manager:** `sha/storage/chain_manager.go` - Tracks backup chains
- **Repository:** `sha/storage/local_repository.go` - Storage interface
- **Binary:** `sendense-hub` (see `source/builds/sendense-hub-v*`)

**SNA Side (VMA in code):**
- **API Server:** `sna/api/server.go` - Receives backup requests from SHA
- **VMware Service:** `sna/vmware/service.go` - VMware operations
- **Backup Client:** `sendense-backup-client` binary - Performs actual data transfer
- **Binary:** `sna-api-server` (deployed on 10.0.100.231)

---

## 🗄️ DATABASE SCHEMA

**Main Tables (in `migratekit_oma` database):**

### **backup_jobs**
```sql
id VARCHAR(191) PRIMARY KEY    -- Format: "backup-{vm_name}-{timestamp}-{uuid}"
vm_context_id VARCHAR(191)     -- Context directory name
vm_name VARCHAR(255)
backup_type ENUM('full','incremental')
status ENUM('pending','running','completed','failed')
change_id VARCHAR(191)         -- VMware CBT change ID (for incrementals)
parent_backup_id VARCHAR(191)  -- FK to parent backup (for incrementals)
bytes_transferred BIGINT
created_at TIMESTAMP
completed_at TIMESTAMP
```

### **backup_chains**
```sql
id INT AUTO_INCREMENT PRIMARY KEY
vm_name VARCHAR(255) UNIQUE
full_backup_id VARCHAR(191)    -- FK to backup_jobs (first full)
latest_backup_id VARCHAR(191)  -- FK to backup_jobs (most recent)
created_at TIMESTAMP
updated_at TIMESTAMP
```

### **vm_disks**
```sql
id INT AUTO_INCREMENT PRIMARY KEY
vm_context_id VARCHAR(191)
disk_index INT                 -- 0, 1, 2...
capacity_bytes BIGINT
datastore VARCHAR(255)
vmdk_path TEXT                 -- Full VMware path
nbd_port INT                   -- Allocated NBD port (10100-10200)
qcow2_path TEXT                -- Path to QCOW2 file
```

### **vmware_credentials**
```sql
id INT AUTO_INCREMENT PRIMARY KEY
name VARCHAR(255) UNIQUE       -- e.g., "production-vcenter"
vcenter_host VARCHAR(255)
vcenter_port INT               -- Default 443
username VARCHAR(255)
password_encrypted TEXT
```

**Foreign Keys:**
- `backup_jobs.parent_backup_id` → `backup_jobs.id`
- `backup_chains.full_backup_id` → `backup_jobs.id`
- `backup_chains.latest_backup_id` → `backup_jobs.id`

---

## 🔌 API ENDPOINTS

**Base URL:** `http://localhost:8082/api/v1`

### **Start Backup**
```bash
POST /api/v1/backups
Content-Type: application/json

{
  "vm_name": "pgtest1",
  "repository_id": "1",
  "backup_type": "full"  # or "incremental"
}

# Response:
{
  "backup_id": "backup-pgtest1-20251008-a1b2c3",
  "vm_name": "pgtest1",
  "backup_type": "full",
  "status": "running",
  "change_id": "",
  "message": "Backup started successfully"
}
```

### **List Backups**
```bash
GET /api/v1/backups?vm_name=pgtest1

# Response:
{
  "backups": [
    {
      "backup_id": "...",
      "vm_name": "pgtest1",
      "backup_type": "full",
      "status": "completed",
      "created_at": "2025-10-08T06:00:00Z"
    }
  ]
}
```

### **Get Backup Details**
```bash
GET /api/v1/backups/{backup_id}
```

### **Get Backup Chain**
```bash
GET /api/v1/backups/chain/{vm_name}

# Returns: Full backup + all incrementals in order
```

**Code Location:** `sha/api/handlers/backup_handlers.go`  
**Documentation:** `source/current/api-documentation/OMA.md` (lines 336-393)

---

## 📁 FILE STRUCTURE

### **QCOW2 Storage**
```
/backup/repository/
└── ctx-{vm_name}-{timestamp}/
    ├── disk-0/
    │   ├── backup-{vm_name}-{timestamp}-full.qcow2
    │   └── backup-{vm_name}-{timestamp2}-incr.qcow2  # Backing file: full.qcow2
    └── disk-1/
        ├── backup-{vm_name}-{timestamp}-full.qcow2
        └── backup-{vm_name}-{timestamp2}-incr.qcow2
```

### **Source Code**
```
source/current/
├── sha/                        # SHA (OMA) code
│   ├── api/handlers/          # API handlers
│   ├── workflows/             # Backup orchestration
│   ├── services/              # qemu-nbd, port allocation
│   ├── storage/               # QCOW2, chains, repository
│   └── cmd/main.go           # Binary entry point
├── sna/                       # SNA (VMA) code
│   ├── api/server.go         # API server
│   └── vmware/service.go     # VMware operations
└── sendense-backup-client/    # Data transfer binary
    └── internal/target/nbd.go # NBD operations, change_id handling
```

### **Binaries**
```
source/builds/
├── sendense-hub-v2.*.0-*      # SHA binaries (sort by version)
├── sna-api-server-v1.*.0-*    # SNA binaries
└── sendense-backup-client-*   # Backup client binaries

/usr/local/bin/
├── sendense-hub → /home/oma_admin/sendense/source/builds/sendense-hub-v*
└── (sna-api-server on 10.0.100.231)
```

---

## 🔑 KEY CONCEPTS

### **Multi-Disk Architecture**
- Backups are **VM-level**, not per-disk
- Each disk gets separate NBD export (ports 10100+)
- Each disk gets separate QCOW2 file in same context directory
- VMware disk keys: 2000, 2001, 2002... (i + 2000)
- All disks backed up concurrently for consistency

### **NBD Protocol**
- **qemu-nbd** exports QCOW2 files as NBD targets
- Runs with `--shared=10` for concurrent access
- Dynamic port allocation (10100-10200 range)
- SNA connects via SSH tunnel (port 443 → local ports)

### **VMware CBT (Changed Block Tracking)**
- **Full backup:** No change_id, creates snapshot, transfers all data
- **Incremental:** Uses previous change_id, queries changed blocks only
- **change_id format:** `{uuid}/{sequence}` (e.g., "52d0eb97.../446")
- Stored in `backup_jobs.change_id` for next incremental

### **QCOW2 Backing Files**
- **Full backup:** Standalone QCOW2
- **Incremental:** QCOW2 with backing file pointing to parent
- Created via `qemu-img create -f qcow2 -b {parent} {new}`
- Chain tracked in database via `parent_backup_id`

### **SSH Tunnel**
- **All** SNA→SHA traffic over port 443
- SNA maintains persistent SSH tunnel
- Forwards NBD ports (10100-10200) to localhost
- Command: `ssh -L 10100:localhost:10100 -L 10101:localhost:10101...`

---

## 🧪 COMMON TESTING COMMANDS

### **Start Backup**
```bash
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}'
```

### **Monitor Progress**
```bash
# Watch QCOW2 files grow
watch 'ls -lh /backup/repository/*.qcow2 2>/dev/null || ls -lh /backup/repository/ctx-*/disk-*/*.qcow2 2>/dev/null'

# Check qemu-nbd processes
ps aux | grep qemu-nbd

# Check NBD ports
ss -tlnp | grep '10[01][0-9][0-9]'
```

### **Check Database**
```bash
mysql -u oma_user -p'oma_password' migratekit_oma -e \
  "SELECT id, vm_name, backup_type, status, change_id, created_at FROM backup_jobs ORDER BY created_at DESC LIMIT 5;"
```

### **Cleanup Test Environment**
```bash
/home/oma_admin/sendense/scripts/cleanup-backup-environment.sh
```

### **Check QCOW2 Details**
```bash
qemu-img info /backup/repository/ctx-pgtest1-*/disk-0/backup-*.qcow2
# Look for "backing file" line in incrementals
```

---

## 🐛 COMMON ISSUES & SOLUTIONS

### **Issue: qemu-nbd processes linger**
**Symptom:** `Failed to get "write" lock` on QCOW2 files  
**Solution:** 
```bash
pkill -9 qemu-nbd
rm -rf /backup/repository/ctx-*
```

### **Issue: change_id not recorded**
**Symptom:** `backup_jobs.change_id = NULL` after full backup  
**Cause:** `MIGRATEKIT_JOB_ID` env var not set in SNA  
**Solution:** ✅ FIXED in `sna-api-server-v1.12.0-changeid-fix`
- Added `cmd.Env` configuration in `sna/api/server.go` lines 691-701
- Binary deployed on SNA (10.0.100.231:8081)
- Verified working: log shows "Set progress tracking job ID from command line flag"
- Job sheet: `job-sheets/2025-10-08-changeid-recording-fix.md`

### **Issue: Disk keys wrong (both disks same)**
**Symptom:** Data corruption in multi-disk backups  
**Cause:** Old binary without disk key fix  
**Solution:** Deploy binary with `diskKey := i + 2000` fix

### **Issue: NBD ports not released**
**Symptom:** "Port already in use" errors  
**Cause:** `QemuNBDManager` not releasing ports on failure  
**Solution:** Integrated `NBDPortAllocator` into cleanup (completed)

---

## 🔧 DEPLOYMENT

### **SNA Access Credentials**
```bash
Host: vma@10.0.100.231
Password: Password1
```

### **Build SHA Binary**
```bash
cd /home/oma_admin/sendense/source/current/sha/cmd
go build -o /home/oma_admin/sendense/source/builds/sendense-hub-v2.X.0-description main.go
```

### **Deploy SHA Binary**
```bash
sudo systemctl stop sendense-hub || pkill sendense-hub
sudo ln -sf /home/oma_admin/sendense/source/builds/sendense-hub-v2.X.0-description /usr/local/bin/sendense-hub
nohup /usr/local/bin/sendense-hub -port=8082 -auth=false \
  -db-host=localhost -db-port=3306 -db-name=migratekit_oma \
  -db-user=oma_user -db-pass=oma_password >/tmp/sha.log 2>&1 &
```

### **Build SNA Binary**
```bash
cd /home/oma_admin/sendense/source/current/sna-api-server
go build -o /home/oma_admin/sendense/source/builds/sna-api-server-v1.X.0-description .
```

### **Deploy SNA Binary**
```bash
# Using password authentication
sshpass -p 'Password1' scp /home/oma_admin/sendense/source/builds/sna-api-server-v1.X.0-description vma@10.0.100.231:/tmp/sna-new

sshpass -p 'Password1' ssh vma@10.0.100.231 << 'EOF'
  sudo pkill -9 sna-api-server
  sudo mv /tmp/sna-new /usr/local/bin/sna-api-server
  sudo chmod +x /usr/local/bin/sna-api-server
  sudo /usr/local/bin/sna-api-server --port 8081 --auto-cbt=true > /tmp/sna-api.log 2>&1 &
  sleep 2
  ps aux | grep sna-api-server | grep -v grep
EOF
```

---

## 📚 KEY DOCUMENTATION FILES

**Phase 1 Goals:**  
`project-goals/phases/phase-1-vmware-backup.md`

**API Reference:**  
`source/current/api-documentation/API_REFERENCE.md` (index)  
`source/current/api-documentation/OMA.md` (backup endpoints)  
`source/current/api-documentation/API_DB_MAPPING.md` (database impact)

**Database Schema:**  
`source/current/api-documentation/DB_SCHEMA.md`

**Recent Job Sheets:**  
`job-sheets/2025-10-08-phase1-backup-completion.md` (multi-disk backup infrastructure)  
`job-sheets/2025-10-08-changeid-recording-fix.md` (✅ COMPLETE - incremental backups enabled)

**Cleanup Script:**  
`scripts/cleanup-backup-environment.sh`  
`scripts/README.md`

---

## 🎓 LEARNING RESOURCES

### **Understanding Code Flow:**
1. User calls `POST /api/v1/backups`
2. `backup_handlers.go::StartBackup()` validates request
3. `backup.go::ExecuteBackup()` orchestrates:
   - Get VMware credentials
   - Query VM disks from vCenter
   - Allocate NBD ports (one per disk)
   - Create context directory
   - Create QCOW2 files (full or incremental with backing file)
   - Start `qemu-nbd` processes (one per disk)
   - Call SNA API to start backup client
4. SNA receives request, starts `sendense-backup-client` process
5. `sendense-backup-client`:
   - Creates VMware snapshot
   - Opens NBD connections (one per disk)
   - Transfers data (full or changed blocks)
   - Records change_id (if `MIGRATEKIT_JOB_ID` set)
   - Removes snapshot
6. SHA cleanup:
   - Stop `qemu-nbd` processes
   - Release NBD ports
   - Update database status

### **Understanding Multi-Disk:**
- VMware API returns disk list: `vm.Config.Hardware.Device`
- Filter for `*types.VirtualDisk`
- Each disk: separate NBD port, separate QCOW2, concurrent transfer
- Consistency: Single VMware snapshot covers all disks

### **Understanding Incrementals:**
- Requires previous `change_id` from database
- VMware `QueryChangedDiskAreas()` API returns changed blocks
- Transfer only changed blocks (90%+ savings typical)
- QCOW2 references parent via backing file
- Chain integrity via database `parent_backup_id`

---

## ⚠️ IMPORTANT NOTES

1. **No "production ready" claims** - Always test before marking complete
2. **Binaries in `source/builds/` only** - Never in `source/current/`
3. **Update documentation with every API/schema change**
4. **Test incrementals** - Full backup alone doesn't prove Phase 1 complete
5. **Check active binaries** - Symlinks can point to old versions
6. **Follow .cursorrules** - Read before starting work

---

## 🚀 QUICK START FOR NEW SESSION

1. **Check Phase 1 status:** `cat /home/oma_admin/sendense/project-goals/phases/phase-1-vmware-backup.md | grep -A 5 "Status:"`
2. **Check recent work:** `ls -lt /home/oma_admin/sendense/job-sheets/ | head -5`
3. **Check active binary:** `ls -l /usr/local/bin/sendense-hub`
4. **Check last backup:** `mysql -u oma_user -p'oma_password' migratekit_oma -e "SELECT * FROM backup_jobs ORDER BY created_at DESC LIMIT 1\G"`
5. **Read .cursorrules:** `cat /home/oma_admin/sendense/.cursorrules | head -100`

---

**This is a reference document. Current status may differ. Always verify before proceeding.**

