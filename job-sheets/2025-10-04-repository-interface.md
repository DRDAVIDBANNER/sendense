# Job Sheet: Repository Interface & Configuration System

**Date Created:** 2025-10-04  
**Status:** 🔴 **ACTIVE - IN PROGRESS**  
**Project Goal Link:** [project-goals/phases/phase-1-vmware-backup.md → Task 1: Repository Abstraction]  
**Duration:** 3-4 days  
**Priority:** Critical (Foundation for all backup functionality)

---

## 🎯 PROJECT GOALS INTEGRATION (MANDATORY)

### **Specific Project Goals Reference**
**Document:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`  
**Task Section:** **Task 1: Backup Repository Abstraction** (Week 1)  
**Sub-Tasks:** **1.1 Design Repository Interface** + **1.2 Implement Local QCOW2 Backend**  
**Business Value:** Foundation for $10/VM Backup Edition tier  
**Success Criteria:** Generic storage interface supporting multiple backends

**Task Description from Project Goals:**
```
1.1. Design Repository Interface
   - Define Go interface for backup storage
   - Support metadata operations (list, query, delete)
   - Version and chain management
   
1.2. Implement Local QCOW2 Backend
   - QCOW2 file creation with backing files
   - Incremental chain management
   - Metadata storage (JSON sidecar files)
```

**Acceptance Criteria from Project Goals:**
```
- [ ] Can create QCOW2 file with backing file
- [ ] Can track backup chains in metadata
- [ ] Can list all backups for a VM
- [ ] Can calculate total chain size
- [ ] Repository interface extensible for future backends (S3, Azure)
```

---

## 📋 JOB BREAKDOWN (Detailed Implementation)

### **Core Interface Design (Day 1)** ✅

- [x] **Design Repository interface** - Clean Go interface for any storage backend
  - **File:** `source/current/oma/storage/interface.go` ✅
  - **Methods:** CreateBackup(), GetBackup(), ListBackups(), DeleteBackup(), GetStorageInfo()
  - **Evidence:** Interface definition complete with documentation

- [x] **Design metadata structures** - Backup, BackupChain, RepositoryConfig structs
  - **File:** `source/current/oma/storage/metadata.go` ✅
  - **Structs:** Backup, BackupChain, BackupRequest, StorageInfo
  - **Evidence:** All structs with JSON tags and validation

- [x] **Define error types** - Specific errors for backup operations
  - **File:** `source/current/oma/storage/errors.go` ✅
  - **Errors:** ErrBackupNotFound, ErrInsufficientSpace, ErrCorruptChain
  - **Evidence:** Error types with helpful messages

### **Repository Configuration (Day 1-2)** ✅

- [x] **Implement RepositoryConfig system** - Support multiple repository types
  - **File:** `source/current/oma/storage/repository_config.go` ✅
  - **Types:** Local, NFS, CIFS, SMB (S3/Azure future)
  - **Evidence:** Config structs for each type + ImmutableConfig

- [x] **Implement RepositoryManager** - Manage multiple repositories
  - **File:** `source/current/oma/storage/repository_manager.go` ✅
  - **Methods:** RegisterRepository(), GetRepository(), ListRepositories(), TestRepository()
  - **Evidence:** Manager can handle multiple active repositories

- [x] **Database integration** - Store repository configurations
  - **Migration:** `source/current/oma/database/migrations/20251004120000_add_backup_tables.up.sql` ✅
  - **Tables:** `backup_repositories`, `backup_policies`, `backup_copy_rules`, `backup_jobs`, `backup_copies`, `backup_chains`
  - **Evidence:** Complete schema with FK constraints and immutability support

### **QCOW2 Implementation (Day 2-3)** ✅

- [x] **Implement QCOW2Manager** - Core QCOW2 file operations
  - **File:** `source/current/oma/storage/qcow2_manager.go` ✅
  - **Methods:** CreateFull(), CreateIncremental(), GetInfo(), Verify()
  - **Evidence:** Can create QCOW2 with backing files

- [x] **Implement LocalRepository** - Concrete implementation of Repository interface
  - **File:** `source/current/oma/storage/local_repository.go` ✅
  - **Methods:** Full implementation of Repository interface
  - **Evidence:** Creates backups on local filesystem

- [x] **Implement ChainManager** - Track backup chains
  - **File:** `source/current/oma/storage/chain_manager.go` ✅
  - **Methods:** CreateChain(), AddToChain(), ValidateChain(), GetChain()
  - **Evidence:** Properly tracks full → incremental relationships

### **Testing (Day 3-4)**

- [ ] **Unit tests** - Test each component independently
  - **Files:** `*_test.go` for all components
  - **Coverage:** >80% for all new code
  - **Evidence:** All unit tests passing

- [ ] **Integration tests** - End-to-end repository operations
  - **File:** `storage/integration_test.go`
  - **Tests:** Create repo → create backup → verify → delete
  - **Evidence:** Integration tests passing

### **Documentation (Day 4)** ✅

- [x] **API documentation** - Repository management endpoints
  - **File:** `source/current/api-documentation/OMA.md` ✅
  - **Add:** 27 repository/backup endpoints (repositories, policies, jobs, copies)
  - **Evidence:** All endpoints documented with handler references

- [x] **Database schema documentation** - Document new tables
  - **File:** `source/current/api-documentation/DB_SCHEMA.md` ✅
  - **Add:** 6 backup tables (repositories, policies, copy_rules, jobs, copies, chains)
  - **Evidence:** Schema documented with FK relationships and indexes

- [x] **GUI Integration document** - How frontend interacts with backup system
  - **File:** `source/current/api-documentation/BACKUP_REPOSITORY_GUI_INTEGRATION.md` ✅
  - **Add:** Complete GUI component specs, WebSocket patterns, React Query keys
  - **Evidence:** Comprehensive integration guide with 19 API endpoints detailed

---

## 🗄️ DATABASE SCHEMA CHANGES

### **Migration File:**
`source/current/control-plane/database/migrations/20251004000001_add_backup_tables.up.sql`

### **Tables to Create:**
```sql
-- Repository configurations
CREATE TABLE backup_repositories (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    repository_type ENUM('local', 'nfs', 'cifs', 'smb', 's3', 'azure') NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    config JSON NOT NULL,
    total_size_bytes BIGINT DEFAULT 0,
    used_size_bytes BIGINT DEFAULT 0,
    available_size_bytes BIGINT DEFAULT 0,
    last_check_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_name (name),
    INDEX idx_type (repository_type),
    INDEX idx_enabled (enabled)
);

-- Backup jobs
CREATE TABLE backup_jobs (
    id VARCHAR(64) PRIMARY KEY,
    vm_context_id VARCHAR(191) NOT NULL,
    vm_name VARCHAR(255) NOT NULL,
    repository_id VARCHAR(64) NOT NULL,
    backup_type ENUM('full', 'incremental', 'differential') NOT NULL,
    status ENUM('pending', 'running', 'completed', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
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
    FOREIGN KEY (repository_id) REFERENCES backup_repositories(id) ON DELETE RESTRICT,
    FOREIGN KEY (parent_backup_id) REFERENCES backup_jobs(id) ON DELETE SET NULL,
    INDEX idx_vm_context (vm_context_id),
    INDEX idx_repository (repository_id),
    INDEX idx_status (status)
);

-- Backup chains
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
    UNIQUE KEY unique_vm_disk (vm_context_id, disk_id)
);
```

---

## ✅ SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **Interface Design:** Repository interface clean and extensible
- [ ] **QCOW2 Creation:** Can create full and incremental QCOW2 files
- [ ] **Chain Tracking:** Backup chains tracked correctly
- [ ] **Multi-Repository:** Can configure multiple repositories
- [ ] **Database:** All tables created and migrations working
- [ ] **Testing:** >80% code coverage, all tests passing
- [ ] **Documentation:** API and schema docs updated
- [ ] **Code Quality:** Passes linting, no security issues
- [ ] **Architecture Compliance:** Uses JobLog, follows project rules

### **Evidence Collection (Required)**
- **Functional Test:** QCOW2 files created with correct backing file structure
- **Chain Test:** Full backup → 3 incrementals → chain validated
- **Performance Test:** File operations don't degrade NBD throughput
- **Integration Test:** Works with existing SHA infrastructure
- **Code Review:** Pull request approved by architecture team
- **Documentation:** Links to updated API_REFERENCE.md and DB_SCHEMA.md

---

## 🚨 CRITICAL PROJECT RULES COMPLIANCE

### **Must Follow (No Exceptions):**
- ✅ **Source Authority:** All code in `source/current/` only
- ✅ **No Simulations:** Real QCOW2 operations, no placeholders
- ✅ **Volume Daemon:** Not applicable (this is storage layer)
- ✅ **JobLog:** All operations use `internal/joblog` for tracking
- ✅ **Database Fields:** Validate ALL field names against DB_SCHEMA.md
- ✅ **No Direct Logging:** Use JobLog, not logrus/slog
- ✅ **API Documentation:** Update API_REFERENCE.md with changes
- ✅ **No Assumptions:** Validate configurations before operations

### **Architecture Patterns:**
- Repository pattern for storage abstraction
- Interface-based design for extensibility
- JSON sidecar files for metadata
- Database as source of truth for configuration
- Atomic operations with proper error handling

---

## 📊 PROJECT GOALS COMPLETION MARKING

### **When This Job Sheet Completes:**

**Step 1: Update Project Goals Document**
```bash
vi project-goals/phases/phase-1-vmware-backup.md

# Mark tasks complete:
### Task 1: Backup Repository Abstraction (Week 1)

1.1. **Design Repository Interface**
   - [x] **COMPLETED 2025-10-XX** ✅
   - **Job Sheet:** job-sheets/2025-10-04-repository-interface.md
   - **Evidence:** Repository interface operational, supports QCOW2
   - **Files:** control-plane/storage/interface.go, repository_manager.go
   
1.2. **Implement Local QCOW2 Backend**
   - [x] **COMPLETED 2025-10-XX** ✅
   - **Job Sheet:** job-sheets/2025-10-04-repository-interface.md
   - **Evidence:** QCOW2 backups working, chain management operational
   - **Files:** control-plane/storage/qcow2_manager.go, local_repository.go
```

**Step 2: Update Phase Progress**
```markdown
**Phase 1 Progress:** Task 1 complete (1/7 tasks, 14% complete)
**Next Task:** Task 2 - Modify NBD Server for File Export
```

**Step 3: Move Job Sheet to Archive**
```bash
# After completion, archive the job sheet
mkdir -p job-sheets/archive/2025/10/
mv job-sheets/2025-10-04-repository-interface.md job-sheets/archive/2025/10/
```

---

## 🔗 RELATED JOB SHEETS

**Depends On:** None (foundation work)  
**Enables:** 
- `2025-10-04-storage-monitoring.md` (multi-backend support)
- `2025-10-04-backup-copy-engine.md` (copy operations)

**Blocks:**
- Task 2: NBD File Export (needs repository interface)
- Task 3: Backup Workflow (needs repository implementation)

---

## 📝 NOTES & DECISIONS

### **Design Decisions:**
- **QCOW2 Format:** Chosen for native Linux support, incremental backing files
- **JSON Metadata:** Sidecar files for backup metadata (easy debugging)
- **Interface First:** Clean abstraction allows future S3/Azure backends
- **Repository Manager:** Central registry for multiple repositories

### **Technical Constraints:**
- Requires `qemu-img` installed on SHA
- Filesystem must support large files (XFS or ext4)
- Backing file paths are relative to parent

### **Future Enhancements (Not This Job):**
- S3 backend implementation
- Azure Blob backend implementation
- Compression options
- Deduplication

---

**THIS JOB SHEET ENSURES REPOSITORY FOUNDATION IS SOLID AND EXTENSIBLE**

**NO SHORTCUTS - PROPER INTERFACE DESIGN THAT SCALES TO ENTERPRISE NEEDS**

---

**Job Owner:** Backend Engineering Team  
**Reviewer:** Architecture Lead  
**Status:** 🔴 Ready to Start  
**Last Updated:** 2025-10-04
