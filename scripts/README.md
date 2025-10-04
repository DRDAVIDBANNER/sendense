# Migration Control Scripts

These scripts provide external control over migratekit migration jobs, allowing you to start, monitor, and gracefully stop migrations for testing and troubleshooting.

## 🚀 Quick Start (No Parameters Needed!)

### 1. Test Prerequisites
```bash
./scripts/test-migration.sh
```

### 2. Start a Migration
```bash
./scripts/start-migration.sh
```

### 3. Monitor Progress
```bash
tail -f /tmp/migratekit-migration-PGWINTESTBIOS.log
```

### 4. Check Status
```bash
./scripts/status-migration.sh
```

### 5. Stop Migration Gracefully
```bash
./scripts/stop-migration.sh
```

## 📋 Scripts Overview

### `start-migration.sh`
Starts a migration job in the background with full control.

**Usage:**
```bash
./scripts/start-migration.sh
```

**Hardcoded Configuration:**
- VM_NAME: `PGWINTESTBIOS`
- VMWARE_ENDPOINT: `192.168.17.159`
- USERNAME: `administrator@vsphere.local`
- PASSWORD: `EmyGVoBFesGQc47-`
- VM_PATH: `/DatabanxDC/vm/PGWINTESTBIOS`
- CLOUDSTACK_HOST: `pgrayson@10.245.246.125`
- DEVICE: `/dev/vde`

**What it does:**
- Creates PID file for tracking
- Starts migration in background with nohup
- Provides monitoring and stop instructions
- Validates process started successfully

### `stop-migration.sh`
Gracefully stops a running migration and cleans up resources.

**Usage:**
```bash
./scripts/stop-migration.sh [FORCE]
```

**Parameters:**
- FORCE: Set to `true` for immediate force stop (default: `false`)
- VM is hardcoded to: `PGWINTESTBIOS`

**What it does:**
- Sends SIGTERM for graceful shutdown (30 second timeout)
- Falls back to SIGKILL if needed
- Cleans up NBD processes and named pipes
- Shows final log entries
- Checks for ChangeID completion indicator
- Reports CloudStack disk status

### `test-migration.sh`
Validates prerequisites and connectivity before migration.

**Usage:**
```bash
./scripts/test-migration.sh
```

**What it checks:**
- Script files exist and are executable
- migratekit binary is available
- CloudStack SSH connectivity
- Target device exists on appliance

## 📊 Monitoring and Logging

### Log Files
- **Location:** `/tmp/migratekit-migration-{VM_NAME}.log`
- **Content:** Full migration output including progress, errors, and completion status

### PID Files
- **Location:** `/tmp/migratekit-migration-{VM_NAME}.pid`
- **Purpose:** Track running migration process for stop script

### ChangeID Files
- **Location:** `/tmp/migratekit_changeid_{VM_NAME}_disk_2000`
- **Purpose:** CBT tracking for incremental migrations

## 🎯 Testing Scenarios

### 1. Full Migration Test
```bash
# Start migration
./scripts/start-migration.sh

# Monitor in real-time
tail -f /tmp/migratekit-migration-PGWINTESTBIOS.log

# Let it complete naturally or stop when ready
./scripts/stop-migration.sh
```

### 2. Interruption Test
```bash
# Start migration
./scripts/start-migration.sh

# Wait for snapshot creation (look for "Creating snapshot" in logs)
tail -f /tmp/migratekit-migration-PGWINTESTBIOS.log

# Stop at specific point to test cleanup
./scripts/stop-migration.sh
```

### 3. Force Stop Test
```bash
# Start migration
./scripts/start-migration.sh

# Force stop immediately
./scripts/stop-migration.sh true
```

## 🔍 Troubleshooting

### Migration Won't Start
1. Check prerequisites: `./scripts/test-migration.sh`
2. Verify VMware connectivity
3. Check migratekit binary exists: `ls -la ./migratekit`
4. Review error in log file

### Migration Won't Stop
1. Try force stop: `./scripts/stop-migration.sh VM_NAME true`
2. Manual cleanup:
   ```bash
   pkill -f "nbdkit.*vddk"
   pkill -f "nbdcopy"
   rm -f /tmp/cloudstack_stream_*
   ```

### Process Status Check
```bash
# Check if migration is running
ps aux | grep migratekit

# Check NBD processes
ps aux | grep -E "(nbdkit|nbdcopy)"

# Check PID file
cat /tmp/migratekit-migration-PGWINTESTBIOS.pid
```

## 🧹 Manual Cleanup

If scripts fail to clean up properly:

```bash
# Stop all migration processes
pkill -f migratekit
pkill -f nbdkit
pkill -f nbdcopy

# Remove temporary files
rm -f /tmp/migratekit-migration-*.pid
rm -f /tmp/migratekit-migration-*.log
rm -f /tmp/cloudstack_stream_*

# Check VMware snapshots (manual removal may be needed)
# Connect to vCenter and remove any "migration-snapshot-*" snapshots
```

## 📋 Expected Process Flow

1. **Snapshot Creation:** VM snapshot created for consistent access
2. **NBD Setup:** nbdkit server started with VDDK plugin
3. **Data Transfer:** nbdcopy streams data to CloudStack appliance
4. **Progress Tracking:** Real-time progress updates in log
5. **Cleanup:** Snapshot removed, ChangeID written, NBD stopped

## 🎯 Success Indicators

- **ChangeID File:** Indicates successful completion
- **Log Message:** "Migration completed" 
- **No Processes:** No nbdkit/nbdcopy processes running
- **Clean Snapshot:** VMware snapshot removed

Use these scripts to gain full control over your migration testing and troubleshooting process!