# Sendense Changelog

All notable changes to the Sendense platform will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Fixed
- **Repository API Response Format** (October 6, 2025):
  - Fixed ListRepositories handler to return `{ success: true, repositories: [] }` format
  - Backend was returning bare array `[]`, frontend expected wrapped object
  - Matches documented API response format in BACKUP_REPOSITORY_GUI_INTEGRATION.md
  - Resolves "Failed to load repositories" error on Repositories page
  - Binary: sendense-hub-v2.10.1-repo-api-fix deployed
  
### Added
- **Repository Management GUI Integration Job Sheet** (October 6, 2025):
  - Comprehensive Grok prompt for wiring up Repositories page to backend API
  - Complete data transformation guide (bytes→GB, enabled→status mappings)
  - 7 detailed implementation tasks for repository CRUD operations
  - Test connection integration with `/api/v1/repositories/test` endpoint
  - Safety checks for repository deletion (blocks if backups exist)
  - UI/UX enhancements: loading skeletons, error states, success toasts
  - 30+ test scenarios covering all repository operations
  - Location extraction logic for all repository types (Local/NFS/CIFS/S3/Azure)
  - Config building helpers to transform GUI form data to backend structure
- **Multi-Group VM Membership Support** (Protection Groups - October 6, 2025):
  - VMs can now belong to multiple protection groups simultaneously
  - Enhanced `/api/v1/vm-contexts` endpoint to include group membership arrays
  - Compact group display in GUI: single badge, or "GroupName +N more" for multiple groups
  - Backend: VMContextWithGroups type with groups array and group_count fields
  - Frontend: GroupMembership interface tracking group_id, group_name, priority, enabled status
  - Database: Confirmed unique_vm_group constraint allows multi-group membership
  - GUI Protection Groups page shows ALL VMs (not just ungrouped) with group status
  - Binary: sendense-hub-v2.10.0-vm-multi-group
- **Protection Groups GUI Fixes** (October 6, 2025):
  - Fixed EditGroupModal SelectItem empty value error (changed to 'none' value)
  - Fixed ManageVMsModal bulk assignment (loop through VMs individually with singular vm_context_id)
  - Fixed payload mismatch in Add to Group (vm_context_id vs vm_context_ids)
  - Added per-VM error handling with success/fail counts and user feedback alerts
  - Schedule now optional for group creation (manual backups only mode)
- Complete project roadmap and documentation (24 documents)
- PROJECT_RULES.md with mandatory development standards
- MASTER_AI_PROMPT.md for AI assistant context loading
- Multi-platform architecture planning (VMware, CloudStack, Hyper-V, AWS, Azure, Nutanix)
- Terminology framework (descend/ascend/transcend operations)
- MSP cloud platform architecture with bulletproof licensing
- **Sendense Professional GUI (Phase 3 - October 6, 2025):**
  - Complete 8-phase enterprise GUI implementation + major enhancements
  - Next.js 15 + shadcn/ui + TypeScript strict mode
  - 9 functional pages: Dashboard, Protection Flows, Groups, Reports, Settings, Users, Support, Appliances, Repositories
  - Three-panel layout with draggable panels and professional styling
  - Production build successful (15/15 pages static generated)
  - Major enhancements: Appliance fleet management, repository management, flow operational controls
  - Complete deployment guide and troubleshooting documentation
  - Enterprise-grade interface that exceeds Veeam capabilities professionally
- **Repository Management API** (Storage Monitoring Day 4 - 2025-10-05):
  - 5 REST endpoints for backup repository CRUD operations (POST/GET/DELETE /api/v1/repositories)
  - Support for Local, NFS, and CIFS/SMB repository types
  - Test repository configuration endpoint for validation before saving
  - Real-time storage capacity monitoring via /api/v1/repositories/{id}/storage
  - Composition-based NFSRepository and CIFSRepository implementations
  - Full integration with MountManager for network storage operations
  - Protection against deleting repositories with existing backups
- **Backup Policy Management API** (Backup Copy Engine Day 5 - 2025-10-05):
  - 6 REST endpoints for enterprise 3-2-1 backup rule support (POST/GET/DELETE /api/v1/policies, /api/v1/backups/{id}/copies, /api/v1/backups/{id}/copy)
  - Multi-repository backup copy rules with automatic replication
  - Policy-based backup distribution across multiple storage locations
  - Manual backup copy triggering for ad-hoc replication needs
  - Copy rule management with retention periods and copy modes (full/incremental)
  - Integration with immutable storage for ransomware protection
  - BackupCopyEngine with worker pool for concurrent copy operations
  - Checksum verification for backup integrity validation (sha256sum)
  - Database tracking: backup_policies, backup_copy_rules, backup_copies tables
- **Backup Workflow Orchestration** (Task 3 - 2025-10-05):
  - Full and incremental backup workflow implementation (481 lines workflows/backup.go)
  - BackupEngine orchestrates complete backup lifecycle (QCOW2 creation → NBD export → VMA replication → status tracking)
  - BackupJobRepository for database operations (262 lines database/backup_job_repository.go)
  - Integration with NBD file export system for QCOW2 backup files
  - VMA API client integration for triggering Capture Agent replication
  - CBT (Changed Block Tracking) support for incremental backups with change ID tracking
  - Full integration with storage repository layer (Task 1) and NBD server (Task 2)
  - Database tracking: backup_jobs table with status, progress, and error tracking
  - Foundation complete for Phase 1 VMware backup workflows
- **NBD File Export Testing & Validation** (Task 2.3 - 2025-10-05):
  - Complete unit test suite for backup export helpers (285 lines backup_export_helpers_test.go)
  - Comprehensive integration tests (8 scenarios) validated on deployed server (10.245.246.136)
  - SIGHUP reload functionality verified (dynamic export management without service restarts)
  - QCOW2 file creation, validation, and incremental backup testing with qemu-img
  - Export name generation with collision-proof naming and length compliance (<64 chars)
  - Multiple concurrent exports tested (block devices + QCOW2 files)
  - config.d pattern operational and verified
  - Fixed QCOW2 validation logic (handle "no errors" message correctly)
  - Task 2 NBD File Export: 100% COMPLETE (Phases 2.1, 2.2, 2.3 all done)

### Changed
- **VM Contexts Endpoint Enhancement** (October 6, 2025):
  - GET /api/v1/vm-contexts now returns group membership information for all VMs
  - Response structure changed: VMReplicationContext → VMContextWithGroups
  - Added fields: groups (array), group_count (number)
  - Breaking change: GUI must handle new response structure (backward compatible with undefined checks)
  - Impact: Protection Groups page now shows comprehensive VM-to-group relationships
- Component naming: VMA/OMA → Capture Agent/Control Plane
- Project scope: Migration tool → Universal backup platform
- Navigation design: Simple menu → Aviation-inspired cockpit interface

### Architecture
- Cross-platform restore engine design (Enterprise tier enabler)
- Multi-platform replication matrix (Premium tier $100/VM)
- Storage abstraction layer (local, S3, Azure, immutable)
- Performance benchmarking system (source vs target validation)
- Automatic backup validation (boot VMs to test backups)

---

## [2.19.0] - 2025-10-04 (Base Platform - MigrateKit OSSEA)

### Platform Foundation
- ✅ VMware source integration (CBT, VDDK, 3.2 GiB/s performance)
- ✅ CloudStack target integration (Volume Daemon, device correlation)
- ✅ SSH tunnel infrastructure (port 443, Ed25519 keys)
- ✅ Database schema (VM-centric, CASCADE DELETE)
- ✅ JobLog system (structured logging and tracking)
- ✅ Progress tracking (VMA progress service + OMA polling)

### Performance
- Proven 3.2 GiB/s encrypted NBD streaming
- Multi-disk VM support operational
- Concurrent migrations validated
- Single-port NBD architecture (port 10809)

### Infrastructure  
- SSH tunnel system (complete stunnel replacement)
- Volume Management Daemon (centralized operations)
- Enhanced failover system (modular architecture)
- Professional GUI foundation (Next.js dashboard)

---

## Change Categories

### Added
New features, capabilities, or components added to the platform.

### Changed
Modifications to existing functionality or behavior.

### Fixed
Bug fixes and issue resolutions.

### Removed
Features, components, or functionality removed from the platform.

### Security
Security improvements, vulnerability fixes, or security-related changes.

### Performance
Performance improvements, optimizations, or benchmark updates.

### Architecture
Architectural changes, design pattern updates, or structural modifications.

### Documentation
Documentation additions, updates, or improvements.

### Testing
Test additions, test infrastructure improvements, or testing methodology changes.

---

## Version Numbering

Sendense follows [Semantic Versioning](https://semver.org/):

- **MAJOR.MINOR.PATCH** (e.g., 3.1.2)
- **MAJOR:** Breaking changes or major new capabilities
- **MINOR:** New features, backward-compatible additions
- **PATCH:** Bug fixes, small improvements

### Version History Context

**v1.x:** Initial MigrateKit development (legacy)
**v2.x:** MigrateKit OSSEA platform (current base)
**v3.x:** Sendense platform launch (planned)
- v3.0.0: VMware backups + modern GUI
- v3.1.0: CloudStack backups
- v3.2.0: Cross-platform restore (Enterprise tier)
- v3.3.0: Multi-platform replication (Premium tier)
- v3.4.0: Application-aware restores
- v3.5.0: MSP platform launch

---

## Changelog Maintenance Rules

### When to Update
- **EVERY commit** with user-visible changes
- **EVERY API modification** (new endpoints, changed responses)
- **EVERY feature addition** or removal
- **EVERY performance improvement** or regression
- **EVERY security change** or vulnerability fix

### How to Update
1. Add entries to `[Unreleased]` section during development
2. Move to versioned section when releasing
3. Include issue/PR references where applicable
4. Use clear, non-technical language for user-facing changes
5. Be specific about impact and scope

### Required Information
- **What changed:** Clear description of the change
- **Why it changed:** Business or technical reason
- **Impact:** Who is affected (users, admins, developers)
- **Action required:** Any required actions for users
- **Breaking changes:** Clearly marked with migration guide

---

## Examples

### Good Changelog Entries
```markdown
### Added
- VMware backup support with CBT incremental tracking (#SEND-001)
- Cross-platform restore wizard with compatibility validation (#SEND-045)
- S3 repository backend with lifecycle management (#SEND-067)

### Changed
- Improved backup performance by 25% through optimized block transfer (#SEND-089)
- Enhanced error messages for failed platform connections (#SEND-092)

### Fixed
- Resolved race condition in concurrent backup jobs (#SEND-098)
- Fixed memory leak in long-running replication operations (#SEND-101)

### Security
- Updated SSH tunnel to use Ed25519 keys exclusively (#SEND-105)
- Added per-customer encryption key isolation (#SEND-108)
```

### Bad Changelog Entries
```markdown
### Changed
- Fixed stuff
- Updated things
- Made improvements
- Various bug fixes
```

---

**Document Owner:** Engineering Leadership  
**Maintenance:** Updated with every release  
**Format Standard:** Keep a Changelog v1.0.0  
**Last Updated:** October 4, 2025