# Sendense Hub Appliance (SHA) Binary Manifest

**Build Date:** 20251005 003927  
**Git Commit:** 8095ce8  
**Builder:** oma_admin@localhost  
**Go Version:** go1.24.6

## Binaries

### SHA API Server
- **Version:** v2.7.6-api-uuid-correlation
- **Binary:** sendense-hub-v2.7.6-api-uuid-correlation-linux-amd64-20251005-8095ce8
- **Source:** source/current/oma/
- **Description:** Sendense Hub Appliance API server

### Volume Daemon
- **Version:** v2.1.0-dynamic-config-20251001-132544
- **Binary:** volume-daemon-v2.1.0-dynamic-config-20251001-132544-linux-amd64-20251005-8095ce8
- **Source:** source/current/volume-daemon/
- **Description:** Volume management daemon for OSSEA operations

## Installation

```bash
# Copy binaries to system
sudo cp sendense-hub-* /usr/local/bin/sendense-hub
sudo cp volume-daemon-* /usr/local/bin/volume-daemon
sudo chmod +x /usr/local/bin/sendense-hub /usr/local/bin/volume-daemon

# Verify installation
sendense-hub --version
volume-daemon --version
```

## Checksums

See CHECKSUMS.sha256 for binary checksums.

```bash
# Verify checksums
sha256sum -c CHECKSUMS.sha256
```
