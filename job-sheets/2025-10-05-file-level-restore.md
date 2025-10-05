# Job Sheet: File-Level Restore Implementation

**Date Created:** 2025-10-05  
**Status:** 🔴 **READY TO START**  
**Project Goal Link:** [project-goals/phases/phase-1-vmware-backup.md → Task 4: File-Level Restore]  
**Duration:** 1-2 weeks  
**Priority:** Critical (Core customer file recovery capability)

---

## 🎯 PROJECT GOALS INTEGRATION (MANDATORY)

### **Specific Project Goals Reference**
**Document:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`  
**Task Section:** **Task 4: File-Level Restore** (Lines 304-358)  
**Sub-Tasks:** **4.1 QCOW2 Mount, 4.2 File Browser API, 4.3 Safety & Cleanup**  
**Business Value:** Customer file recovery from VMware backups (critical differentiator)  
**Success Criteria:** Mount backup, browse files, download files, automatic cleanup

**Task Description (From Project Goals):**
```
Goal: Mount backups and extract individual files

Sub-Tasks:
4.1. QCOW2 Mount via qemu-nbd
   - Use qemu-nbd to export QCOW2 as block device
   - Mount filesystem from block device
   - Implement safe mount/umount wrapper
   
4.2. File Browser API
   - List files/directories in mounted backup
   - Download individual files
   - Support recursive directory downloads
   
4.3. Safety & Cleanup
   - Automatic umount after timeout
   - Handle mount conflicts
   - Clean up NBD devices properly
```

**Acceptance Criteria (From Project Goals):**
- [ ] Can mount QCOW2 backup
- [ ] Can browse files via API
- [ ] Can download individual files
- [ ] Automatic cleanup after 1 hour idle
- [ ] Multiple concurrent mounts supported

---

## 🔗 DEPENDENCY STATUS

### **Required Before Starting:**
- ✅ Task 1: Repository infrastructure (Local/NFS/CIFS/Immutable repositories)
- ✅ Task 2: NBD file export (QCOW2 files exportable via NBD)
- ✅ Task 3: Backup workflow (QCOW2 backup files created)

### **Enables These Features:**
- ⏸️ Task 5: API Endpoints (can expose file restore via REST)
- ⏸️ Customer file recovery workflows
- ⏸️ Individual file browsing and extraction from backups

---

## 📋 JOB BREAKDOWN (Detailed Implementation)

### **Phase 1: QCOW2 Mount Management (Days 1-3)**

- [ ] **Implement Mount Manager** - Core QCOW2 mount operations
  - **File:** `source/current/oma/restore/mount_manager.go`
  - **Methods:** MountBackup(), UnmountBackup(), ListMounts()
  - **Evidence:** Can mount QCOW2 files via qemu-nbd

- [ ] **qemu-nbd Integration** - Export QCOW2 as block device
  - **Command:** `qemu-nbd --connect=/dev/nbd0 --format=qcow2 backup.qcow2`
  - **Management:** NBD device allocation and cleanup
  - **Evidence:** QCOW2 exported as accessible block device

- [ ] **Filesystem Mount** - Mount block device as filesystem
  - **Detection:** Automatic filesystem type detection
  - **Mount Point:** `/mnt/sendense/restore/{mount-uuid}`
  - **Evidence:** Can access backup files via standard filesystem

- [ ] **Mount Tracking** - Database tracking for active mounts
  - **Table:** `restore_mounts` with mount metadata
  - **Fields:** mount_id, backup_id, mount_path, nbd_device, created_at, expires_at
  - **Evidence:** All mounts tracked in database

### **Phase 2: File Browser API (Days 4-6)**

- [ ] **File Listing Service** - Browse backup contents
  - **File:** `source/current/oma/restore/file_browser.go`
  - **Methods:** ListFiles(), GetFileInfo(), ValidatePath()
  - **Evidence:** Can list directories and files in mounted backup

- [ ] **Directory Traversal** - Navigate backup filesystem structure
  - **Security:** Prevent path traversal attacks
  - **Support:** Recursive directory listing
  - **Evidence:** Safe directory navigation implemented

- [ ] **File Metadata** - Extract file information
  - **Data:** Size, permissions, modified time, type
  - **Format:** JSON response with file details
  - **Evidence:** File metadata accessible via API

- [ ] **Path Validation** - Secure path handling
  - **Validation:** Prevent access outside mount point
  - **Sanitization:** Clean malicious path inputs
  - **Evidence:** Security tested against path traversal

### **Phase 3: File Download & Extraction (Days 5-7)**

- [ ] **File Download Handler** - Individual file extraction
  - **Method:** HTTP streaming download
  - **Support:** Binary and text files
  - **Evidence:** Can download files from mounted backup

- [ ] **Directory Download** - Recursive directory extraction
  - **Format:** ZIP or TAR archive creation
  - **Streaming:** Large directory streaming support
  - **Evidence:** Can download entire directories as archives

- [ ] **Download Security** - Secure file access
  - **Validation:** Ensure file exists and is accessible
  - **Limits:** File size limits and timeout protection
  - **Evidence:** Download operations are secure and bounded

- [ ] **Progress Tracking** - Download progress monitoring
  - **Support:** Large file download progress
  - **Cancellation:** Ability to cancel long downloads
  - **Evidence:** Download progress visible and manageable

### **Phase 4: Safety & Cleanup (Days 6-8)**

- [ ] **Automatic Cleanup** - Idle mount cleanup
  - **Timeout:** 1 hour idle timeout (configurable)
  - **Background:** Cleanup service monitoring
  - **Evidence:** Unused mounts automatically cleaned up

- [ ] **Mount Conflict Resolution** - Handle concurrent access
  - **Detection:** Multiple mount attempts for same backup
  - **Resolution:** Reuse existing mount or queue
  - **Evidence:** No mount conflicts under concurrent access

- [ ] **NBD Device Management** - Proper NBD cleanup
  - **Allocation:** Dynamic NBD device allocation (/dev/nbd0-15)
  - **Cleanup:** Proper qemu-nbd disconnect
  - **Evidence:** NBD devices properly allocated and released

- [ ] **Resource Monitoring** - Track mount resource usage
  - **Limits:** Maximum concurrent mounts (default: 8)
  - **Monitoring:** Disk space and memory usage
  - **Evidence:** System resources protected from exhaustion

### **Phase 5: API Integration (Days 8-10)**

- [ ] **REST API Handlers** - Mount/browse/download endpoints
  - **File:** `source/current/oma/api/handlers/restore_handlers.go`
  - **Endpoints:** 4 REST endpoints for file-level restore
  - **Evidence:** Complete API for backup file access

- [ ] **API Documentation** - Complete endpoint documentation
  - **Update:** `source/current/api-documentation/OMA.md`
  - **Schemas:** Request/response documentation
  - **Evidence:** API endpoints documented with examples

- [ ] **Integration Testing** - End-to-end restore workflow
  - **Test:** Mount → browse → download → cleanup cycle
  - **Validation:** Multiple concurrent restore sessions
  - **Evidence:** Complete workflow operational

---

## 🏗️ TECHNICAL ARCHITECTURE

### **Mount Workflow**
```
1. Receive mount request (backup_id, mode)
       ↓
2. Validate backup exists in repository
       ↓  
3. Allocate NBD device (/dev/nbdX)
       ↓
4. Export QCOW2 via qemu-nbd
       ↓
5. Detect filesystem type
       ↓
6. Mount filesystem to /mnt/sendense/restore/{uuid}
       ↓
7. Track mount in restore_mounts table
       ↓
8. Return mount_id and mount_path
```

### **File Access Workflow**
```
1. Receive file/directory request (mount_id, path)
       ↓
2. Validate mount exists and active
       ↓
3. Sanitize and validate file path
       ↓
4. Access file via mounted filesystem
       ↓
5. Stream file data or metadata
       ↓
6. Update mount last_accessed timestamp
```

### **Cleanup Workflow**
```
1. Background cleanup service (every 15 minutes)
       ↓
2. Find mounts idle > 1 hour
       ↓
3. Umount filesystem
       ↓
4. Disconnect qemu-nbd
       ↓
5. Release NBD device
       ↓
6. Remove mount tracking record
```

### **Database Schema Integration**
```sql
CREATE TABLE restore_mounts (
    id VARCHAR(64) PRIMARY KEY,
    backup_id VARCHAR(64) NOT NULL,
    mount_path VARCHAR(512) NOT NULL,
    nbd_device VARCHAR(32) NOT NULL,
    filesystem_type VARCHAR(32),
    mount_mode ENUM('read-only') DEFAULT 'read-only',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    status ENUM('mounting', 'mounted', 'unmounting', 'failed') DEFAULT 'mounting',
    FOREIGN KEY (backup_id) REFERENCES backup_jobs(id) ON DELETE CASCADE
);

CREATE INDEX idx_restore_mounts_backup_id ON restore_mounts(backup_id);
CREATE INDEX idx_restore_mounts_expires_at ON restore_mounts(expires_at);
CREATE INDEX idx_restore_mounts_status ON restore_mounts(status);
```

---

## ✅ SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **QCOW2 Mount Success:** Can mount backup files via qemu-nbd
- [ ] **File Browser Functional:** Can navigate backup filesystem structure
- [ ] **Download Operations:** Can extract individual files and directories
- [ ] **Security Validated:** Path traversal protection operational
- [ ] **Resource Management:** Automatic cleanup and NBD device management
- [ ] **Concurrent Support:** Multiple simultaneous restore sessions
- [ ] **API Integration:** Complete REST endpoints for all operations

### **Testing Evidence Required**
- [ ] Mount a backup QCOW2 file successfully
- [ ] Browse backup directory structure via API
- [ ] Download individual files from mounted backup
- [ ] Download directory as archive (ZIP/TAR)
- [ ] Automatic cleanup after idle timeout
- [ ] Multiple concurrent mounts without conflicts
- [ ] NBD device allocation and proper cleanup

---

## 🚨 PROJECT RULES COMPLIANCE

### **Must Follow (No Exceptions):**
- ✅ **Source Authority:** All code in `source/current/` only
- ✅ **Repository Pattern:** Database operations via repository interfaces
- ✅ **Integration Points:** Use Tasks 1-3 infrastructure cleanly
- ✅ **Error Handling:** Graceful failures with comprehensive logging
- ✅ **Security:** Prevent path traversal, validate all inputs
- ✅ **Resource Management:** Proper NBD device allocation and cleanup
- ✅ **No Simulations:** Real qemu-nbd operations only
- ✅ **API Documentation:** Update API_REFERENCE.md and OMA.md

### **Integration Requirements:**
- **Task 1 Integration:** Use `storage.Repository` interface to find backup files
- **Task 2 Integration:** No conflicts with NBD file export operations  
- **Task 3 Integration:** Access backup files created by BackupEngine
- **Database:** Use repository pattern for all database operations
- **NBD Coordination:** Coordinate with existing NBD server usage

---

## 📊 DELIVERABLES

### **Code Deliverables**
- `source/current/oma/restore/mount_manager.go` - Core mount operations
- `source/current/oma/restore/file_browser.go` - File navigation and metadata
- `source/current/oma/restore/cleanup_service.go` - Automatic resource cleanup
- `source/current/oma/api/handlers/restore_handlers.go` - REST API endpoints
- `source/current/oma/database/restore_mount_repository.go` - Database operations

### **Database Schema**
- `restore_mounts` table with mount tracking
- Foreign key relationships to backup_jobs
- Indexes for performance optimization

### **API Endpoints**
- `POST /api/v1/restore/mount` - Mount backup for browsing
- `GET /api/v1/restore/{mount_id}/files` - List files/directories
- `GET /api/v1/restore/{mount_id}/download` - Download files
- `DELETE /api/v1/restore/{mount_id}` - Unmount backup

### **Documentation Deliverables**
- Updated API documentation (OMA.md, API_REFERENCE.md)
- Mount manager usage documentation
- Security considerations for file access
- Performance characteristics and limitations

---

## 🔗 INTEGRATION POINTS

### **Task 1 Dependencies (Repository Infrastructure)**
- **Repository Access:** Find backup files using `storage.Repository.GetBackup()`
- **Multi-Repository:** Support backups in Local/NFS/CIFS repositories
- **Path Resolution:** Get actual backup file paths from repository

### **Task 2 Dependencies (NBD File Export)**
- **NBD Coordination:** Ensure no conflicts with backup NBD exports
- **Device Allocation:** Use different NBD devices (/dev/nbd0-7 for restore, /dev/nbd8+ for backups)
- **QCOW2 Compatibility:** Same QCOW2 files, different access pattern

### **Task 3 Dependencies (Backup Workflow)**
- **Backup Files:** Access QCOW2 files created by BackupEngine
- **Metadata Integration:** Use backup_jobs table for backup validation
- **Chain Integration:** Support incremental backup mounting

### **Database Integration**
- **Repository Pattern:** All database operations via repository interfaces
- **Foreign Keys:** Proper relationships to existing backup tables
- **Transaction Safety:** Atomic mount/unmount operations

---

## 🎯 ENTERPRISE VALUE

### **Customer Capabilities Enabled**
- ✅ **Individual File Recovery** - Customers can recover single files without full VM restore
- ✅ **Selective Directory Recovery** - Extract specific application directories
- ✅ **File Browsing** - Navigate backup contents before recovery decision
- ✅ **Bulk Downloads** - Download multiple files/directories as archives
- ✅ **Self-Service Recovery** - Customers can recover files independently

### **Competitive Advantages**
- ✅ **Faster Recovery** - File-level restore vs full VM restore
- ✅ **Storage Efficiency** - Only download needed files
- ✅ **User Experience** - Browse backups like file explorer
- ✅ **Compliance** - Granular recovery for audit requirements
- ✅ **Cost Effective** - Reduces restore bandwidth and time

---

## 📋 ACCEPTANCE CRITERIA

### **Functional Requirements**
- [ ] **Mount QCOW2 Backup:** Successfully mount backup files via qemu-nbd
- [ ] **File Browser:** Navigate backup filesystem via REST API
- [ ] **File Download:** Extract individual files via HTTP streaming
- [ ] **Directory Download:** Recursive directory extraction as archives
- [ ] **Multiple Mounts:** Support concurrent restore sessions (8+ simultaneous)
- [ ] **Automatic Cleanup:** Idle mounts cleaned up after 1 hour timeout

### **Security Requirements**
- [ ] **Path Validation:** Prevent path traversal attacks
- [ ] **Mount Isolation:** Each mount isolated to unique directory
- [ ] **Access Control:** Validate user access to backup before mount
- [ ] **Resource Limits:** Prevent resource exhaustion attacks

### **Performance Requirements**
- [ ] **Mount Speed:** <10 seconds to mount typical backup
- [ ] **File Listing:** <2 seconds to list directory with 1000 files
- [ ] **Download Performance:** Streaming downloads at disk speed
- [ ] **Cleanup Efficiency:** Background cleanup without performance impact

### **Integration Requirements**
- [ ] **Repository Integration:** Works with all repository types (Local/NFS/CIFS)
- [ ] **Backup Integration:** Can mount any backup created by Task 3 workflow
- [ ] **NBD Coordination:** No conflicts with existing NBD export operations
- [ ] **Database Consistency:** All operations tracked via repository pattern

---

## 🔧 IMPLEMENTATION NOTES

### **NBD Device Allocation Strategy**
```
Restore Usage:  /dev/nbd0-7   (8 devices for concurrent mounts)
Backup Usage:   /dev/nbd8+    (Dedicated for backup export operations)

Prevents conflicts between restore mounts and backup exports
```

### **Mount Path Structure**
```
/mnt/sendense/restore/
├── {mount-uuid-1}/     # Individual mount directories
├── {mount-uuid-2}/     # Isolated from each other
└── {mount-uuid-N}/     # Up to 8 concurrent mounts
```

### **Security Considerations**
- **Path Sanitization:** All file paths validated against mount root
- **Mount Isolation:** Each mount in separate directory
- **Read-Only Access:** All mounts are read-only (backup integrity)
- **Timeout Protection:** Automatic cleanup prevents resource leaks

### **Performance Optimization**
- **Mount Caching:** Keep recently accessed mounts available
- **Lazy Cleanup:** Background cleanup service (non-blocking)
- **Streaming Downloads:** HTTP streaming for large files
- **Efficient Archives:** On-demand archive creation for directories

---

## 🎯 TASK 4 READY FOR IMPLEMENTATION

**Foundation Complete:** Tasks 1-3 provide all necessary infrastructure
**Architecture Clear:** Mount → browse → download → cleanup workflow  
**Integration Planned:** Clean separation from backup operations
**Documentation Ready:** Complete specification with acceptance criteria

---

**THIS JOB ENABLES CUSTOMER FILE RECOVERY**

**ENTERPRISE FILE-LEVEL RESTORE CAPABILITY**

---

**Job Owner:** Backend Engineering Team  
**Reviewer:** Architecture Lead + UX Review (file browsing)  
**Status:** 🔴 Ready to Start  
**Last Updated:** 2025-10-05
