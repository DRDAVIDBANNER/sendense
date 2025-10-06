# Sendense Changelog

All notable changes to the Sendense platform will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
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