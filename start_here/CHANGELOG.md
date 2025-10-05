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
- Cockpit-style GUI design system adapted from original plan
- **Repository Management API** (Storage Monitoring Day 4 - 2025-10-05):
  - 5 REST endpoints for backup repository CRUD operations (POST/GET/DELETE /api/v1/repositories)
  - Support for Local, NFS, and CIFS/SMB repository types
  - Test repository configuration endpoint for validation before saving
  - Real-time storage capacity monitoring via /api/v1/repositories/{id}/storage
  - Composition-based NFSRepository and CIFSRepository implementations
  - Full integration with MountManager for network storage operations
  - Protection against deleting repositories with existing backups

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