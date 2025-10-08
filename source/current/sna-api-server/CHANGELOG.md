# SNA API Changelog

All notable changes to the Sendense Node Appliance (SNA) API will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.4.0] - 2025-10-07

### Added
- **Backup Endpoint:** POST `/api/v1/backup/start` for multi-disk VMware backups
- Multi-disk NBD targets support via comma-separated target string
- Job tracking integration for backup operations
- sendense-backup-client process management
- Automatic fallback to migratekit binary for backwards compatibility
- Log file management at `/var/log/sendense/backup-{job_id}.log`
- Comprehensive request validation with clear error messages

### Changed
- SNA API now supports 12 endpoints (was 11)
- Enhanced job tracking with backup-specific progress monitoring

### Fixed
- End-to-end backup workflow now functional (SHA → SNA communication working)
- Backup endpoint resolves 404 error that was blocking backup operations

### Technical Details
- Binary: `sna-api-v1.4.0-backup-endpoint`
- Deployment: `/opt/vma/bin/sna-api` on SNA appliances
- Service: `sna-api.service` (systemd managed)
- Port: 8081
- Integration: Works with SHA Backup API via SSH tunnel or direct connection

---

## [1.3.2] - Previous

### Features
- VM discovery endpoint
- Replication endpoint
- VM specification change detection
- CBT status checking
- Power management endpoints
- Enrollment system

---

**Maintainer:** Sendense Engineering Team  
**Last Updated:** October 7, 2025

