# Phase 1: VMware Backup Implementation

**Phase ID:** PHASE-01  
**Status:** 🔴 **CURRENT PHASE - START HERE**  
**Priority:** Critical  
**Timeline:** 4-6 weeks  
**Team Size:** 2-3 developers

---

## 🎯 Phase Objectives

**Primary Goal:** Implement file-based backups for VMware VMs with incremental support

**Success Criteria:**
- ✅ Full backup of VMware VM to QCOW2 file
- ✅ Incremental backup using VMware CBT
- ✅ Backup chain management (full + incrementals)
- ✅ File-level restore (mount backup, extract files)
- ✅ 90%+ data reduction on incrementals vs full
- ✅ Performance: Maintain 3.2 GiB/s throughput

**Deliverables:**
1. Backup repository abstraction layer
2. VMware backup workflow (reuse existing Capture Agent)
3. QCOW2 backup storage implementation
4. File-level restore capability
5. Basic API endpoints
6. Command-line tools for testing

---

## 🏗️ Architecture Overview

### **What We're Building**

```
┌──────────────────────────────────────────────────────────────┐
│ PHASE 1: VMWARE BACKUP ARCHITECTURE                         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  VMware vCenter                                              │
│       ↓                                                      │
│  Capture Agent (existing VMA)                                │
│   ├─ CBT change tracking (existing) ✅                      │
│   ├─ VDDK/nbdkit read (existing) ✅                         │
│   └─ NBD stream (existing) ✅                               │
│       ↓ SSH Tunnel (port 443)                               │
│  Control Plane (existing OMA)                                │
│   ├─ NEW: Backup Repository Interface                       │
│   ├─ NEW: QCOW2 Storage Backend                            │
│   ├─ NEW: Backup Chain Manager                              │
│   └─ NEW: File Restore Engine                               │
│       ↓                                                      │
│  /var/lib/sendense/backups/                                  │
│   └─ {vm-uuid}/disk-0/                                      │
│      ├─ full-20251004-120000.qcow2   (40 GB)                │
│      ├─ incr-20251004-180000.qcow2   (2 GB)                 │
│      └─ incr-20251005-000000.qcow2   (1.5 GB)               │
└──────────────────────────────────────────────────────────────┘
```

### **What We're Reusing** ✅

From existing MigrateKit OSSEA platform:
- **Capture Agent** (VMA) - VMware source connector
- **CBT Tracking** - Change block tracking
- **NBD Streaming** - 3.2 GiB/s data transfer
- **SSH Tunnel** - Secure port 443 communication
- **Database Schema** - VM-centric architecture
- **JobLog System** - Operation tracking
- **Progress Tracking** - Real-time job monitoring

**Key Insight:** ~70% of the hard work is already done!

---

## 📋 Task Breakdown

### **Task 1: Backup Repository Abstraction** (Week 1)

**Goal:** Create generic storage interface for any backup target

**Sub-Tasks:**
1.1. **Design Repository Interface**
   - Define Go interface for backup storage
   - Support metadata operations (list, query, delete)
   - Version and chain management
   
1.2. **Implement Local QCOW2 Backend**
   - QCOW2 file creation with backing files
   - Incremental chain management
   - Metadata storage (JSON sidecar files)
   
1.3. **Backup Chain Manager**
   - Track full → incr → incr relationships
   - Handle chain consolidation (merge incrementals)
   - Prune old backups based on retention policy

**Files to Create:**
```
source/current/control-plane/storage/
├── interface.go              # Repository interface definition
├── local_repository.go       # Local disk implementation
├── qcow2_manager.go          # QCOW2 file operations
├── chain_manager.go          # Backup chain tracking
└── metadata.go               # JSON metadata structs
```

**Acceptance Criteria:**
- [ ] Can create QCOW2 file with backing file
- [ ] Can track backup chains in metadata
- [ ] Can list all backups for a VM
- [ ] Can calculate total chain size

---

### **Task 2: Modify NBD Server for File Export** (Week 1-2)

**Goal:** Extend NBD server to export files (not just block devices)

**Current State:** NBD server exports `/dev/vdX` block devices  
**New State:** NBD server can also export QCOW2 files

**Architecture Decision:** Follow Volume Daemon pattern with config.d + SIGHUP

**Sub-Tasks:**
2.1. **Migrate to config.d Pattern**
   - Update OMA NBD to use config.d directory structure (like Volume Daemon)
   - Base NBD config with `includedir = /opt/migratekit/nbd-configs/conf.d`
   - Individual export files in conf.d directory
   - Implement SIGHUP reload functionality
   
2.2. **Add File Export Support**
   - Modify `internal/oma/nbd/server.go`
   - Add `CreateFileExport()` method for QCOW2 files
   - Support both block device and file exports in same server
   
2.3. **Handle QCOW2-specific Options**
   - Set proper filesize from QCOW2 metadata using `qemu-img info`
   - Support read-write for incremental writes
   - Use SIGHUP reload after adding/removing exports (no service restart)

**Files to Modify:**
```
source/current/oma/nbd/
├── server.go                 # Add CreateFileExport method  
├── config.go                 # Migrate to config.d pattern with SIGHUP
└── models.go                 # Add FileExport type, update Export struct
```

**Export Naming Strategy (Collision Avoidance):**

Current migration exports use: `migration-vm-{vmID}-disk{diskNumber}`  
New backup exports will use: `backup-{vmContextID}-disk{diskID}-{backupType}-{timestamp}`

**Examples:**
- `backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000`
- `backup-ctx-pgtest2-20251005-120000-disk0-incr-20251005T130000`  
- `backup-ctx-pgtest2-20251005-120000-disk1-full-20251005T120000`

**Collision Prevention:**
- ✅ **Unique VM Context ID** - No VM name collisions
- ✅ **Backup prefix** - Distinguished from `migration-` exports
- ✅ **Disk ID** - Multi-disk VM support
- ✅ **Backup type** - full/incr distinction
- ✅ **Timestamp** - Multiple backup chain support
- ✅ **Length limit** - NBD export names <64 chars

**Implementation Notes:**
- **Pattern Consistency:** Align OMA NBD with proven Volume Daemon architecture
- **Dynamic Exports:** Add/remove exports without NBD server restart (SIGHUP only)
- **File Size Detection:** Use `qemu-img info --output=json` for accurate QCOW2 size
- **Backward Compatibility:** Existing block device exports continue working
- **Config Structure:**
  ```
  /opt/migratekit/nbd-configs/
  ├── nbd-server.conf          # Base config with includedir
  └── conf.d/                  # Individual export files
      ├── backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000.conf
      ├── backup-ctx-pgtest2-20251005-120000-disk0-incr-20251005T130000.conf
      └── migration-vm-a1b2c3d4-e5f6-7890-abcd-ef1234567890-disk0.conf
  ```

**Benefits:**
- ✅ **No Service Restarts** - SIGHUP reload for backup exports
- ✅ **Atomic Operations** - Individual export files prevent corruption
- ✅ **Proven Architecture** - Reuses Volume Daemon's working pattern
- ✅ **Clean Separation** - Block device vs file exports managed consistently

**Export Naming Implementation:**
```go
// BuildBackupExportName generates unique NBD export name for backup
func BuildBackupExportName(vmContextID string, diskID int, backupType string, timestamp time.Time) string {
    // Format: backup-{vmContextID}-disk{diskID}-{backupType}-{timestamp}
    // Example: backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000
    
    timestampStr := timestamp.Format("20060102T150405")
    exportName := fmt.Sprintf("backup-%s-disk%d-%s-%s", 
        vmContextID, diskID, backupType, timestampStr)
    
    // Ensure name length < 64 characters (NBD limit)
    if len(exportName) > 63 {
        // Truncate vmContextID if needed, preserve other components
        maxContextLen := 63 - len(fmt.Sprintf("backup--disk%d-%s-%s", diskID, backupType, timestampStr))
        if maxContextLen > 0 {
            truncatedContext := vmContextID[:maxContextLen]
            exportName = fmt.Sprintf("backup-%s-disk%d-%s-%s", 
                truncatedContext, diskID, backupType, timestampStr)
        }
    }
    
    return exportName
}
```

**Acceptance Criteria:**
- [ ] NBD server migrated to config.d pattern with SIGHUP reload
- [ ] NBD server can export QCOW2 files alongside block devices
- [ ] Backup exports use unique naming scheme (no collisions with migrations)
- [ ] Export names support multi-disk VMs and multiple backup types
- [ ] Export names remain under 64 character NBD limit
- [ ] Capture Agent can connect to file exports (same as block device exports)
- [ ] Data writes to QCOW2 file correctly with proper file locking
- [ ] No regression on existing block device exports
- [ ] Export add/remove operations use SIGHUP (no service restart)

---

### **Task 3: Backup Workflow Implementation** (Week 2-3)

**Goal:** Orchestrate backup jobs from Control Plane

**Sub-Tasks:**
3.1. **Full Backup Workflow**
   - Create new QCOW2 file for VM
   - Generate NBD export for file
   - Call Capture Agent to start replication
   - Monitor progress via existing JobLog
   - Mark backup as complete in database
   
3.2. **Incremental Backup Workflow**
   - Query last backup's change ID
   - Create QCOW2 with backing file (previous backup)
   - Generate NBD export
   - Call Capture Agent with previous change ID (existing CBT support)
   - Only changed blocks transferred
   
3.3. **Database Integration**
   - Create `backup_jobs` table
   - Create `backup_chains` table
   - Track backup metadata

**Files to Create:**
```
source/current/control-plane/workflows/
├── backup.go                 # Main backup workflow
├── full_backup.go            # Full backup logic
├── incremental_backup.go    # Incremental backup logic
└── backup_job_tracker.go     # Database operations
```

**Database Schema:**
```sql
CREATE TABLE backup_jobs (
    id VARCHAR(64) PRIMARY KEY,
    vm_context_id VARCHAR(191),
    vm_name VARCHAR(255),
    backup_type ENUM('full', 'incremental'),
    status ENUM('pending', 'running', 'completed', 'failed'),
    repository_path VARCHAR(512),
    parent_backup_id VARCHAR(64),
    change_id VARCHAR(191),
    bytes_transferred BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE
);

CREATE TABLE backup_chains (
    id VARCHAR(64) PRIMARY KEY,
    vm_context_id VARCHAR(191),
    disk_id INT,
    full_backup_id VARCHAR(64),
    latest_backup_id VARCHAR(64),
    total_backups INT DEFAULT 0,
    total_size_bytes BIGINT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (full_backup_id) REFERENCES backup_jobs(id),
    FOREIGN KEY (latest_backup_id) REFERENCES backup_jobs(id)
);
```

**Acceptance Criteria:**
- [ ] Full backup completes successfully
- [ ] Incremental backup only transfers changed blocks
- [ ] Backup chain tracked in database
- [ ] Progress visible in logs/GUI
- [ ] Performance: 3.2 GiB/s maintained

---

### **Task 4: File-Level Restore** (Week 3-4)

**Goal:** Mount backups and extract individual files

**Sub-Tasks:**
4.1. **QCOW2 Mount via qemu-nbd**
   - Use `qemu-nbd` to export QCOW2 as block device
   - Mount filesystem from block device
   - Implement safe mount/umount wrapper
   
4.2. **File Browser API**
   - List files/directories in mounted backup
   - Download individual files
   - Support recursive directory downloads
   
4.3. **Safety & Cleanup**
   - Automatic umount after timeout
   - Handle mount conflicts
   - Clean up NBD devices properly

**Files to Create:**
```
source/current/control-plane/restore/
├── mount_manager.go          # qemu-nbd mount operations
├── file_browser.go           # File listing and extraction
└── cleanup.go                # Automatic umount
```

**API Endpoints:**
```bash
# Mount a backup for browsing
POST /api/v1/restore/mount
{
  "backup_id": "backup-pgtest2-20251004120000",
  "mode": "read-only"
}
Response: { "mount_id": "mount-uuid-123", "mount_path": "/mnt/sendense/mount-uuid-123" }

# List files in mounted backup
GET /api/v1/restore/mount-uuid-123/files?path=/var/www/html

# Download a file
GET /api/v1/restore/mount-uuid-123/download?path=/var/www/html/index.php

# Umount backup
DELETE /api/v1/restore/mount-uuid-123
```

**Acceptance Criteria:**
- [ ] Can mount QCOW2 backup
- [ ] Can browse files via API
- [ ] Can download individual files
- [ ] Automatic cleanup after 1 hour idle
- [ ] Multiple concurrent mounts supported

---

### **Task 5: API Endpoints** (Week 4)

**Goal:** Expose backup operations via REST API

**Endpoints to Implement:**

```bash
# Start full backup
POST /api/v1/backup/start
{
  "vm_name": "pgtest2",
  "backup_type": "full",
  "repository": "local"
}

# Start incremental backup
POST /api/v1/backup/start
{
  "vm_name": "pgtest2",
  "backup_type": "incremental",
  "repository": "local"
}

# List backups for a VM
GET /api/v1/backup/list?vm_name=pgtest2

# Get backup details
GET /api/v1/backup/{backup_id}

# Delete backup
DELETE /api/v1/backup/{backup_id}

# Get backup chain
GET /api/v1/backup/chain?vm_name=pgtest2
```

**Files to Create:**
```
source/current/control-plane/api/handlers/
└── backup_handlers.go        # Backup API endpoints
```

**Acceptance Criteria:**
- [ ] All endpoints functional
- [ ] Proper error handling
- [ ] RBAC integration (existing system)
- [ ] API documentation (Swagger)

---

### **Task 6: CLI Tools** (Week 4)

**Goal:** Command-line tools for testing and admin

**Tools to Create:**

```bash
# Backup a VM
sendense-ctl backup start --vm pgtest2 --type full

# List backups
sendense-ctl backup list --vm pgtest2

# Mount backup for browsing
sendense-ctl backup mount --backup-id backup-pgtest2-20251004 --path /tmp/restore

# Extract a file
sendense-ctl backup extract --backup-id backup-pgtest2-20251004 --file /var/www/index.php --output ./index.php

# Show backup chain
sendense-ctl backup chain --vm pgtest2
```

**Files to Create:**
```
source/current/control-plane/cmd/sendense-ctl/
├── main.go
└── commands/
    ├── backup.go
    ├── mount.go
    └── restore.go
```

**Acceptance Criteria:**
- [ ] CLI commands work end-to-end
- [ ] User-friendly output
- [ ] Progress indicators
- [ ] Error messages clear

---

### **Task 7: Testing & Validation** (Week 5-6)

**Goal:** Comprehensive testing of backup functionality

**Test Scenarios:**

7.1. **Full Backup Test**
   - Backup small VM (10 GB)
   - Backup large VM (500 GB)
   - Validate QCOW2 file integrity
   - Verify all data present

7.2. **Incremental Backup Test**
   - Full backup → change 5% of data → incremental
   - Verify only 5% transferred
   - Mount incremental, verify files present
   - Test chain of 5 incrementals

7.3. **File Restore Test**
   - Mount backup
   - Extract files
   - Verify file contents match original
   - Test large files (>1 GB)

7.4. **Performance Test**
   - Measure full backup speed
   - Measure incremental backup speed
   - Verify 3.2 GiB/s throughput maintained
   - Test concurrent backups (5+ VMs)

7.5. **Failure Scenarios**
   - Disk full during backup
   - Network interruption mid-backup
   - Corrupt QCOW2 file detection
   - Capture Agent crash during backup

**Acceptance Criteria:**
- [ ] All test scenarios pass
- [ ] No regressions in existing functionality
- [ ] Performance targets met
- [ ] Edge cases handled gracefully

---

## 📊 Database Schema Changes

### **New Tables**

```sql
-- Migration file: 20251004000001_add_backup_tables.up.sql

CREATE TABLE backup_jobs (
    id VARCHAR(64) PRIMARY KEY,
    vm_context_id VARCHAR(191) NOT NULL,
    vm_name VARCHAR(255) NOT NULL,
    backup_type ENUM('full', 'incremental', 'differential') NOT NULL,
    status ENUM('pending', 'running', 'completed', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
    repository_type VARCHAR(50) NOT NULL,
    repository_path VARCHAR(512) NOT NULL,
    parent_backup_id VARCHAR(64) NULL,
    change_id VARCHAR(191) NULL,
    bytes_transferred BIGINT DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    compression_enabled BOOLEAN DEFAULT TRUE,
    error_message TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE,
    FOREIGN KEY (parent_backup_id) REFERENCES backup_jobs(id) ON DELETE SET NULL,
    INDEX idx_vm_context (vm_context_id),
    INDEX idx_status (status),
    INDEX idx_created (created_at)
);

CREATE TABLE backup_chains (
    id VARCHAR(64) PRIMARY KEY,
    vm_context_id VARCHAR(191) NOT NULL,
    disk_id INT NOT NULL,
    full_backup_id VARCHAR(64) NOT NULL,
    latest_backup_id VARCHAR(64) NOT NULL,
    total_backups INT DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE,
    FOREIGN KEY (full_backup_id) REFERENCES backup_jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (latest_backup_id) REFERENCES backup_jobs(id) ON DELETE CASCADE,
    UNIQUE KEY unique_vm_disk (vm_context_id, disk_id),
    INDEX idx_vm_context (vm_context_id)
);

CREATE TABLE backup_repositories (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    repository_type ENUM('local', 's3', 'azure', 'nfs') NOT NULL,
    config JSON NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    total_size_bytes BIGINT DEFAULT 0,
    available_size_bytes BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_name (name),
    INDEX idx_type (repository_type)
);
```

---

## 🎯 Success Metrics

### **Functional Metrics**
- ✅ Full backup completes without errors
- ✅ Incremental backup uses <20% of full backup data
- ✅ File-level restore extracts correct files
- ✅ Backup chains tracked accurately
- ✅ No data loss or corruption

### **Performance Metrics**
- ✅ Throughput: 3.2 GiB/s (maintained from existing system)
- ✅ Full backup: ~5 minutes for 100 GB VM
- ✅ Incremental backup: ~30 seconds for 5 GB changes
- ✅ File restore mount: <5 seconds
- ✅ Concurrent backups: 5+ VMs simultaneously

### **Quality Metrics**
- ✅ Code coverage: >80%
- ✅ No critical bugs in production
- ✅ All API endpoints documented
- ✅ CLI tools user-tested

---

## 🚀 Deployment Plan

### **Week 6: Production Deployment**

1. **Build & Package**
   - Compile Control Plane with new backup module
   - Update Capture Agent (if needed)
   - Package CLI tools

2. **Database Migration**
   - Apply schema changes to production
   - Verify migration successful
   - Backup existing database first

3. **Service Deployment**
   - Deploy new Control Plane binary
   - Restart services with zero downtime
   - Verify health checks pass

4. **Testing in Production**
   - Run backup on test VM
   - Validate results
   - Monitor for issues

5. **Documentation**
   - User guide: How to backup VMs
   - Admin guide: Backup management
   - API documentation
   - Troubleshooting guide

---

## 📚 Dependencies & Risks

### **Dependencies**
- ✅ Existing Capture Agent (VMA) - No changes needed
- ✅ SSH tunnel infrastructure - Operational
- ✅ NBD streaming - Working at 3.2 GiB/s
- ✅ Database schema - VM-centric design
- ⚠️ QCOW2 tooling - Need `qemu-img`, `qemu-nbd` installed
- ⚠️ Filesystem support - Need XFS or ext4 for large files

### **Risks & Mitigation**

**Risk 1: QCOW2 Performance**
- **Risk:** QCOW2 overhead might reduce throughput
- **Mitigation:** Test early, use no compression initially, optimize later
- **Fallback:** Use raw format with metadata sidecar

**Risk 2: Disk Space**
- **Risk:** Backup chains can grow large
- **Mitigation:** Implement retention policies, chain consolidation
- **Fallback:** Warn users before disk full, automatic cleanup

**Risk 3: NBD Server Complexity**
- **Risk:** Adding file exports might break block device exports
- **Mitigation:** Extensive testing, maintain backward compatibility
- **Fallback:** Keep old NBD logic intact, add new path

---

## 🎓 Learning & Documentation

### **Documentation to Create**
1. **Architecture Document:** Backup system design
2. **API Reference:** All backup endpoints
3. **User Guide:** How to backup and restore
4. **Admin Guide:** Managing backup repositories
5. **Troubleshooting:** Common issues and fixes

### **Internal Knowledge Sharing**
- Architecture review meeting (Week 1)
- Mid-phase demo (Week 3)
- Final demo and handoff (Week 6)

---

## ✅ Phase 1 Completion Checklist

**Before declaring Phase 1 complete:**
- [ ] Full VMware VM backup working
- [ ] Incremental backup using CBT working
- [ ] Backup chains tracked in database
- [ ] File-level restore functional
- [ ] API endpoints documented
- [ ] CLI tools user-tested
- [ ] Performance targets met (3.2 GiB/s)
- [ ] All tests passing
- [ ] Production deployment successful
- [ ] Documentation complete
- [ ] Zero regressions in existing features

**Sign-off Required:**
- [ ] Engineering Lead
- [ ] QA Lead
- [ ] Product Manager

---

## 🔗 Next Steps

**After Phase 1 Completion:**
→ **Phase 2: CloudStack Backup** (libvirt dirty bitmaps)
→ **Phase 3: GUI Redesign** (modern backup dashboard)

---

**Phase Owner:** Backend Engineering Team  
**Last Updated:** October 4, 2025  
**Status:** 🔴 Active - Ready to Start

