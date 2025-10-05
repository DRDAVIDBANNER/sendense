# Sendense Hub Appliance (SHA) Binary Manifest

**Build Date:** 20251005 114100  
**Git Commit:** 2cf590d  
**Builder:** oma_admin@localhost  
**Go Version:** $(go version | awk '{print $3}')  
**Platform:** linux/amd64

## Current Production Binaries (v2.8.1)

### Sendense Hub API Server
- **Version:** v2.8.1-nbd-progress-tracking
- **Binary:** sendense-hub-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d
- **Symlink:** sendense-hub-latest → sendense-hub-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d
- **Source:** source/current/oma/cmd/main.go
- **Size:** 33 MB (statically linked)
- **Description:** Sendense Hub Appliance API server with backup capabilities
- **Features:**
  - Complete OMA API (replication, failover, scheduling)
  - Backup repository management
  - Backup job orchestration
  - Copy engine integration
  - Volume management via Volume Daemon
  - VMA enrollment and SSH tunnel management
  - JobLog correlation tracking
- **Build Flags:**
  ```
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64
  -ldflags "-X main.version=v2.8.1-nbd-progress-tracking 
            -X main.commit=2cf590d 
            -X main.buildTime=2025-10-05T11:41:00Z"
  ```

### Volume Management Daemon
- **Version:** v2.8.1-nbd-progress-tracking
- **Binary:** volume-daemon-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d
- **Symlink:** volume-daemon-latest → volume-daemon-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d
- **Source:** source/current/volume-daemon/
- **Size:** 2.2 MB (statically linked)
- **Description:** Volume management daemon for OSSEA/CloudStack operations
- **Features:**
  - Centralized volume lifecycle management
  - Device correlation and mapping
  - NBD export management
  - Atomic volume operations
  - Polling-based device detection
  - Operation mode management (oma/failover)
- **Build Flags:**
  ```
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64
  -ldflags "-X main.version=v2.8.1-nbd-progress-tracking 
            -X main.commit=2cf590d 
            -X main.buildTime=2025-10-05T11:41:00Z"
  ```

## Installation

### Via Deployment Script (Recommended)
```bash
cd /home/oma_admin/sendense/deployment/sha-appliance/scripts
./deploy-sha-remote.sh <TARGET_IP>
```

### Manual Installation
```bash
# Copy binaries to system
sudo cp sendense-hub-latest /opt/sendense/bin/sendense-hub
sudo cp volume-daemon-latest /usr/local/bin/volume-daemon
sudo chmod +x /opt/sendense/bin/sendense-hub /usr/local/bin/volume-daemon

# Verify installation
/opt/sendense/bin/sendense-hub --version
/usr/local/bin/volume-daemon --version
```

## Checksums

See CHECKSUMS-v2.8.1.sha256 for binary checksums.

```bash
# Verify checksums
sha256sum -c CHECKSUMS-v2.8.1.sha256
```

**SHA256 Checksums:**
```
21b5fa75bbb622787578b5d26b7d800c676454e8e8d904a4db4e99d81c923fd7  sendense-hub-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d
a7e9352ab862a8ead86583fa6e7fbbd00882da86d6d6385982ef4a6c85ccb5ef  volume-daemon-v2.8.1-nbd-progress-tracking-linux-amd64-20251005-2cf590d
```

## Version History

### v2.8.1-nbd-progress-tracking (Current - 2025-10-05)
- **Commit:** 2cf590d
- **Changes:** 
  - Added initial backup export helpers test suite (Task 2.3 prep)
  - NBD progress tracking enhancements
  - Backup system integration (6 new database tables)
  - Enhanced job correlation for GUI visibility
- **Database Schema:** Unified SHA schema (41 tables: 35 OMA + 6 backup)
- **Status:** ✅ Production ready
- **Deployment:** Remote deployment via deploy-sha-remote.sh

### v2.7.6-api-uuid-correlation (Previous - 2025-10-05)
- **Commit:** 8095ce8
- **Status:** Archived (replaced by v2.8.1)
- **Binaries:** Kept for rollback capability

## Binary Properties

### Sendense Hub
- **Type:** ELF 64-bit LSB executable
- **Architecture:** x86-64
- **Linking:** Statically linked (no external dependencies)
- **Debug Info:** Included (not stripped)
- **BuildID:** dccc66b4892c7bdaef1fbb8a3c7e27278ddabdeb

### Volume Daemon
- **Type:** ELF 64-bit LSB executable
- **Architecture:** x86-64
- **Linking:** Statically linked (no external dependencies)
- **Debug Info:** Included (not stripped)
- **BuildID:** 6811b70c458ffef2853ddf82ca1fa3e188d0e00f

## Database Compatibility

### Sendense Hub Requirements
- **Database:** migratekit_oma (kept for compatibility)
- **User:** oma_user / oma_password
- **Schema Version:** Unified SHA schema (41 tables)
- **Required Tables:** 
  - 35 OMA tables (replication, failover, volumes, etc.)
  - 6 Backup tables (repositories, policies, jobs, chains, copies, rules)

### Volume Daemon Requirements
- **Database:** migratekit_oma (same as Sendense Hub)
- **User:** oma_user / oma_password
- **Required Tables:**
  - volume_operations
  - device_mappings
  - ossea_volumes
  - volume_daemon_metrics

## Service Configuration

### Sendense Hub Service
- **Service Name:** sendense-hub.service
- **Binary Path:** /opt/sendense/bin/sendense-hub
- **Config:** Via command-line flags (-port=8082, -db-name=migratekit_oma, etc.)
- **User:** oma_admin
- **Dependencies:** mariadb.service, volume-daemon.service

### Volume Daemon Service
- **Service Name:** volume-daemon.service
- **Binary Path:** /usr/local/bin/volume-daemon
- **Config:** Environment variables + database
- **User:** oma_admin
- **Dependencies:** mariadb.service

## Deployment Targets

### Production Requirements
- Ubuntu 24.04 LTS
- MariaDB 10.11+
- 8GB+ RAM
- 100GB+ disk space
- NBD server installed
- Port 443 (SSH), 8082 (API), 8090 (Volume Daemon), 10809 (NBD) available

### Tested Platforms
- ✅ Ubuntu 24.04 LTS (Production)
- ✅ Ubuntu 22.04 LTS (Development)

## Build Information

### Source Code
- **Repository:** /home/oma_admin/sendense/source/current/
- **OMA Source:** source/current/oma/
- **Volume Daemon Source:** source/current/volume-daemon/
- **Version File:** source/current/VERSION.txt

### Build Environment
- **Go Version:** 1.24.6
- **CGO:** Disabled (static linking)
- **Target OS:** Linux
- **Target Arch:** AMD64 (x86-64)

### Build Reproducibility
All builds include:
- Version string from VERSION.txt
- Git commit hash (short form, 7 chars)
- Build timestamp (RFC3339 format)
- Static linking (CGO_ENABLED=0)

## Rollback Procedure

To rollback to v2.7.6:

```bash
cd /home/oma_admin/sendense/deployment/sha-appliance/binaries

# Update symlinks to previous version
rm sendense-hub-latest volume-daemon-latest
ln -s sendense-hub-v2.7.6-api-uuid-correlation-linux-amd64-20251005-8095ce8 sendense-hub-latest
ln -s volume-daemon-v2.1.0-dynamic-config-20251001-132544-linux-amd64-20251005-8095ce8 volume-daemon-latest

# Redeploy
cd ../scripts
./deploy-sha-remote.sh <TARGET_IP>
```

## Support

For binary issues:
- Check logs: `journalctl -u sendense-hub -n 100`
- Check logs: `journalctl -u volume-daemon -n 100`
- Verify checksums: `sha256sum -c CHECKSUMS-v2.8.1.sha256`
- Test API: `curl http://localhost:8082/health`
- Test Volume Daemon: `curl http://localhost:8090/api/v1/health`

---

**Last Updated:** 2025-10-05 11:41:00 UTC  
**Manifest Version:** 2.0
