# VMA Dependencies - NBD Stack

## Contents

This directory contains the complete NBD stack required for VMware migrations:

### nbdkit-vddk-stack.tar.gz (40MB)

**Contents:**
- `/usr/bin/nbdkit` (147KB) - nbdkit v1.36.3 server
- `/usr/lib/x86_64-linux-gnu/nbdkit/plugins/nbdkit-vddk-plugin.so` (60KB) - VDDK plugin v1.42.6
- `/usr/lib/vmware-vix-disklib/` - Complete VMware VDDK libraries
  - `lib64/libvddkVimAccess.so*` - VMware vSphere access
  - `lib64/libvddkVac2.so*` - VDDK API
  - `lib64/libvddk-api-bindings.so` - API bindings
  - `lib64/libdiskLibPlugin.so` (3.9MB) - Disk library plugin
  - `lib64/libcrypto.so*` (4.5MB) - Cryptography
  - `lib64/libcurl.so*` (623KB) - HTTP client
  - All supporting libraries

## Source

Extracted from working VMA (10.0.100.231) with proven configuration.

## Deployment

This tarball is automatically deployed during VMA production build:
- Script: `scripts/deploy-production-vma-with-enrollment.sh`
- Phase: System Dependencies (Phase 1)

### Manual Deployment

```bash
# Copy to VMA
scp nbdkit-vddk-stack.tar.gz vma@VMA_IP:/tmp/

# Extract (as root)
sudo tar xzf /tmp/nbdkit-vddk-stack.tar.gz -C /

# Verify
nbdkit --version
nbdkit --dump-plugin vddk
ls -la /usr/lib/vmware-vix-disklib/
```

## Verification

After deployment, verify with:
```bash
# Check nbdkit
which nbdkit && nbdkit --version

# Check VDDK plugin
nbdkit --dump-plugin vddk | head -5

# Check libraries
ls -la /usr/lib/vmware-vix-disklib/lib64/
```

## Version Info

- **nbdkit**: 1.36.3-1ubuntu10
- **nbdkit-plugin-vddk**: 1.42.6-1
- **VMware VDDK**: Compatible with vSphere 8.0+
- **Extracted**: 2025-09-29
- **Source VMA**: 10.0.100.231

## Notes

- This stack is required for migratekit to read from VMware
- Without this, migrations will fail with "nbdkit: executable file not found"
- The VDDK libraries are proprietary VMware code
- This exact configuration is proven working in production
