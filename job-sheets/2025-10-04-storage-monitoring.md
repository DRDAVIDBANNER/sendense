# Job Sheet: Storage Monitoring & Multi-Backend Support

**Date Created:** 2025-10-04  
**Status:** 🟡 **PENDING** (Blocked by repository-interface job)  
**Project Goal Link:** [project-goals/phases/phase-1-vmware-backup.md → Task 1: Repository Abstraction]  
**Duration:** 3-4 days  
**Priority:** High (Required for production repository management)

---

## 🎯 PROJECT GOALS INTEGRATION (MANDATORY)

### **Specific Project Goals Reference**
**Document:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`  
**Task Section:** **Task 1: Backup Repository Abstraction** (Week 1-2)  
**Sub-Tasks:** **Storage monitoring** + **Multi-backend support (NFS/CIFS)**  
**Business Value:** Enterprise repository management for $10-25/VM tiers  
**Success Criteria:** Multiple repositories with capacity monitoring

**Task Description (Extended from Project Goals):**
```
Repository Configuration & Monitoring:
   - Support local, NFS, CIFS/SMB storage
   - Real-time capacity monitoring
   - Storage health checks
   - Multiple repository management
   - Mount/unmount automation
```

**Acceptance Criteria:**
```
- [ ] Can configure NFS repository
- [ ] Can configure CIFS/SMB repository
- [ ] Storage capacity reported accurately
- [ ] Background monitoring every 5 minutes
- [ ] Alerts when storage >90% full
- [ ] Can mount/unmount network storage safely
```

---

## 🔗 DEPENDENCIES

**Blocks This Job:**
- `2025-10-04-repository-interface.md` - MUST complete first

**This Job Enables:**
- `2025-10-04-backup-copy-engine.md` - Needs multi-repository support
- Task 3: Backup Workflow - Needs repository selection

**Required Before Starting:**
- ✅ Repository interface defined
- ✅ LocalRepository implementation complete
- ✅ backup_repositories table created

---

## 📋 JOB BREAKDOWN (Detailed Implementation)

### **Mount Management (Day 1-2)**

- [ ] **Implement MountManager** - Handle network storage mounting
  - **File:** `source/current/control-plane/storage/mount_manager.go`
  - **Methods:** MountNFS(), MountCIFS(), Unmount(), IsMounted()
  - **Evidence:** Can mount/unmount NFS and CIFS shares

- [ ] **NFS mount support** - Network File System mounting
  - **Implementation:** Execute `mount -t nfs` with proper options
  - **Config:** NFSRepositoryConfig with server, export path, options
  - **Evidence:** NFS share successfully mounted and accessible

- [ ] **CIFS/SMB mount support** - Windows/Samba share mounting
  - **Implementation:** Execute `mount -t cifs` with credentials
  - **Config:** CIFSRepositoryConfig with username, password, domain
  - **Evidence:** CIFS share successfully mounted with authentication

- [ ] **Auto-mount on startup** - Mount configured repositories at service start
  - **Integration:** RepositoryManager initialization
  - **Feature:** Auto-mount enabled repositories
  - **Evidence:** Repositories mounted after service restart

- [ ] **Safe unmount** - Graceful unmount with active job checking
  - **Check:** No active backup jobs using repository
  - **Method:** Unmount() with force option for admin
  - **Evidence:** Cannot unmount with active jobs

### **Storage Monitoring (Day 2-3)**

- [ ] **Implement StorageMonitor** - Background storage monitoring
  - **File:** `source/current/control-plane/storage/storage_monitor.go`
  - **Methods:** CheckRepository(), GetStorageInfo(), checkAllRepositories()
  - **Evidence:** Monitor runs every 5 minutes

- [ ] **Capacity detection** - Use syscall.Statfs for filesystem stats
  - **Implementation:** Get total, used, available bytes
  - **Calculation:** Used percentage for alerts
  - **Evidence:** Accurate capacity reporting

- [ ] **Database updates** - Store storage info in database
  - **Table:** `backup_repositories` (total_size_bytes, used_size_bytes, available_size_bytes)
  - **Update:** Every monitoring cycle
  - **Evidence:** Database reflects current storage state

- [ ] **Alert system** - Warn when storage getting full
  - **Thresholds:** 90% warning, 95% critical
  - **Action:** Log warning, disable repository at 95%
  - **Evidence:** Alerts triggered at correct thresholds

- [ ] **Background goroutine** - Continuous monitoring service
  - **Start:** Monitor.Start(ctx) in main service
  - **Interval:** 5 minutes (configurable)
  - **Evidence:** Monitor runs continuously without memory leaks

### **Repository Types Implementation (Day 3-4)**

- [ ] **NFSRepository** - NFS-backed repository
  - **Implementation:** Wrapper around LocalRepository with mount management
  - **Config:** Server, export path, mount options
  - **Evidence:** Can create backups on NFS mount

- [ ] **CIFSRepository** - CIFS/SMB-backed repository
  - **Implementation:** Wrapper around LocalRepository with credential management
  - **Config:** Server, share name, credentials
  - **Evidence:** Can create backups on CIFS mount

- [ ] **Repository factory** - Create correct repository type
  - **Method:** NewRepository(config) returns appropriate type
  - **Factory:** Switch on repository_type field
  - **Evidence:** Returns correct implementation for each type

### **API Endpoints (Day 4)**

- [ ] **POST /api/v1/repositories** - Create new repository
  - **Handler:** `api/handlers/repository_handlers.go`
  - **Validation:** Test connection before saving
  - **Evidence:** Can register repositories via API

- [ ] **GET /api/v1/repositories** - List all repositories
  - **Response:** Include storage info for each
  - **Filter:** Optional filter by type or status
  - **Evidence:** Returns all configured repositories

- [ ] **GET /api/v1/repositories/{id}/storage** - Force storage check
  - **Action:** Immediate storage capacity check
  - **Response:** Real-time storage information
  - **Evidence:** Returns fresh storage stats

- [ ] **POST /api/v1/repositories/test** - Test configuration
  - **Action:** Validate config without saving
  - **Check:** Mount, test write, get capacity, unmount
  - **Evidence:** Catches configuration errors before saving

- [ ] **DELETE /api/v1/repositories/{id}** - Delete repository
  - **Check:** No existing backups
  - **Action:** Unmount and remove configuration
  - **Evidence:** Cannot delete with active backups

### **Testing (Day 4)**

- [ ] **Unit tests** - Test mount manager and monitor
  - **Coverage:** >80% for new code
  - **Mock:** Filesystem operations for testing
  - **Evidence:** All unit tests passing

- [ ] **Integration tests** - End-to-end repository configuration
  - **Scenario:** Create NFS repo → mount → create backup → unmount
  - **Validation:** Storage monitoring updates correctly
  - **Evidence:** Integration tests passing

### **Documentation (Day 4)**

- [ ] **API documentation** - Repository management endpoints
  - **File:** `source/current/api-documentation/API_REFERENCE.md`
  - **Add:** All repository CRUD endpoints with examples
  - **Evidence:** Complete endpoint documentation

- [ ] **GUI integration document** - How GUI uses repositories
  - **File:** `docs/gui/backup-repository-integration.md`
  - **Content:** Repository configuration UI flows
  - **Evidence:** Complete GUI integration guide

- [ ] **CHANGELOG.md** - Document multi-backend support
  - **Entry:** "Added NFS and CIFS repository support with monitoring"
  - **Evidence:** Changelog updated

---

## 🗄️ DATABASE SCHEMA CHANGES

### **No New Tables** (uses existing backup_repositories)

### **Field Additions:**
```sql
-- Already included in initial migration
-- Verify these fields exist in backup_repositories:
- total_size_bytes BIGINT
- used_size_bytes BIGINT
- available_size_bytes BIGINT
- last_check_at TIMESTAMP
```

---

## ✅ SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **NFS Support:** Can mount NFS shares and create backups
- [ ] **CIFS Support:** Can mount CIFS shares with authentication
- [ ] **Monitoring:** Storage capacity updated every 5 minutes
- [ ] **Alerts:** Warnings triggered at 90% capacity
- [ ] **Auto-mount:** Repositories mounted on service start
- [ ] **API Endpoints:** All repository management endpoints working
- [ ] **Testing:** >80% coverage, all tests passing
- [ ] **Documentation:** API and GUI docs complete

### **Evidence Collection (Required)**
- **NFS Test:** NFS repository configured and backup created
- **CIFS Test:** CIFS repository with credentials working
- **Monitoring Test:** Storage stats updated in database
- **Alert Test:** Warning logged when storage >90%
- **API Test:** All endpoints return correct responses
- **GUI Doc:** Complete GUI integration document

---

## 🚨 CRITICAL PROJECT RULES COMPLIANCE

### **Must Follow (No Exceptions):**
- ✅ **Source Authority:** All code in `source/current/` only
- ✅ **JobLog:** All operations use `internal/joblog`
- ✅ **Database Fields:** Validate against DB_SCHEMA.md
- ✅ **No Assumptions:** Test configurations before saving
- ✅ **Error Handling:** Proper errors for mount failures
- ✅ **Security:** Encrypt credentials in database
- ✅ **API Documentation:** Update API_REFERENCE.md

### **Security Considerations:**
- **Credentials:** Encrypt CIFS passwords in database
- **Mount Options:** Validate mount options for security
- **Permissions:** Restrict repository deletion to admins
- **Secrets:** Never log passwords or sensitive data

---

## 📊 PROJECT GOALS COMPLETION MARKING

### **When This Job Sheet Completes:**

**Step 1: Update Project Goals Document**
```bash
vi project-goals/phases/phase-1-vmware-backup.md

# Mark extended features complete:
### Task 1: Backup Repository Abstraction (Week 1-2)

**Multi-Backend Support**
   - [x] **COMPLETED 2025-10-XX** ✅
   - **Job Sheet:** job-sheets/2025-10-04-storage-monitoring.md
   - **Evidence:** NFS and CIFS repositories operational
   - **Monitoring:** Real-time capacity tracking working
   - **Files:** storage/mount_manager.go, storage_monitor.go
```

**Step 2: Archive Job Sheet**
```bash
mkdir -p job-sheets/archive/2025/10/
mv job-sheets/2025-10-04-storage-monitoring.md job-sheets/archive/2025/10/
```

---

## 📝 NOTES & DECISIONS

### **Design Decisions:**
- **Mount Management:** Separate MountManager for clean abstraction
- **Monitoring Interval:** 5 minutes balances accuracy vs overhead
- **Alert Thresholds:** 90% warning, 95% critical (disable)
- **Auto-mount:** Optional per repository for flexibility

### **Technical Constraints:**
- Requires `nfs-common` package for NFS mounts
- Requires `cifs-utils` package for CIFS mounts
- Must run as root or have CAP_SYS_ADMIN for mounting
- Credentials stored encrypted in database

### **Future Enhancements (Not This Job):**
- S3 backend with lifecycle policies
- Azure Blob backend
- Immutable storage support
- Multi-tenant repository isolation

---

**THIS JOB ENABLES ENTERPRISE REPOSITORY MANAGEMENT**

**MULTIPLE STORAGE TYPES WITH PROFESSIONAL MONITORING**

---

**Job Owner:** Backend Engineering Team  
**Reviewer:** Architecture Lead + Security Review (credentials)  
**Status:** 🟡 Pending (waiting for repository-interface)  
**Last Updated:** 2025-10-04
