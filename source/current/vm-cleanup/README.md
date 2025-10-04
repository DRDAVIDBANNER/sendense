# VM Cleanup Utility

A robust Go utility for safely cleaning up VMs from the MigrateKit OSSEA system. This tool performs proper volume detachment via the Volume Daemon and cascade deletion of VM contexts.

## Features

- 🔍 **VM Discovery**: Finds VM contexts and associated volumes
- 📤 **Safe Volume Detachment**: Uses Volume Daemon API for proper detachment
- 🗑️ **Cascade Deletion**: Automatically cleans up all related records
- 🧪 **Dry Run Mode**: Preview what would be deleted without making changes
- ⚠️ **Safety Checks**: Warns about active VMs and requires confirmation
- 🔍 **Verification**: Confirms cleanup completed successfully
- 📋 **Verbose Logging**: Detailed progress information

## Usage

```bash
# Basic cleanup with confirmation
./vm-cleanup -vm <vm_name>

# Dry run to see what would be deleted
./vm-cleanup -vm <vm_name> -dry-run

# Force cleanup without confirmation
./vm-cleanup -vm <vm_name> -force

# Verbose output
./vm-cleanup -vm <vm_name> -verbose
```

## Examples

```bash
# Preview cleanup for pgtest1
./vm-cleanup -vm pgtest1 -dry-run

# Clean up PGWINTESTBIOS with verbose output
./vm-cleanup -vm PGWINTESTBIOS -verbose

# Force cleanup of failed VM without confirmation
./vm-cleanup -vm broken-vm -force
```

## What Gets Deleted

The utility performs a **cascade delete** that removes:

- ✅ VM replication context
- ✅ All replication jobs and history  
- ✅ All disk records and metadata
- ✅ All volume mappings
- ✅ All CBT history
- ✅ All network mappings
- ✅ All failover job records

## Safety Features

1. **Active State Detection**: Warns if VM is actively replicating
2. **Confirmation Prompt**: Requires explicit 'yes' confirmation
3. **Dry Run Mode**: Preview operations without making changes
4. **Force Flag**: Skip confirmations for automation
5. **Volume Detachment**: Proper CloudStack volume detachment first
6. **Verification**: Confirms all records were properly cleaned up

## Build Instructions

```bash
cd /home/pgrayson/migratekit-cloudstack/source/current/vm-cleanup
go mod tidy
go build -o vm-cleanup .
```

## Installation

```bash
# Build and install
go build -o vm-cleanup .
sudo cp vm-cleanup /usr/local/bin/
sudo chmod +x /usr/local/bin/vm-cleanup

# Or install to project bin directory
cp vm-cleanup /opt/migratekit/bin/
```

## Configuration

The utility uses these default configurations:

- **Volume Daemon**: `http://localhost:8090`
- **Database**: `oma_user:oma_password@tcp(localhost:3306)/migratekit_oma`

## Error Handling

The utility includes comprehensive error handling:

- Database connectivity issues
- Volume Daemon communication failures
- Operation timeout handling
- Orphaned record detection
- Invalid VM name handling

## Integration

This utility can be integrated into:

- 🖥️ **GUI**: Add "Delete VM" button with confirmation modal
- 📅 **Scheduler**: Automated cleanup of failed migrations
- 🔧 **Admin Scripts**: Bulk cleanup operations
- 📊 **Monitoring**: Cleanup stale VMs based on status

## Dependencies

- Go 1.21+
- MySQL driver (`github.com/go-sql-driver/mysql`)
- Access to Volume Daemon on `localhost:8090`
- Database access to `migratekit_oma`

## Exit Codes

- `0`: Success
- `1`: Error (invalid arguments, VM not found, operation failed)

## Logging

The utility provides clear, emoji-enhanced output:

```
🔍 Looking up VM context for 'pgtest1'...
📋 Found VM: pgtest1 (Context: ctx-pgtest1-20250922-210037, Status: failed_over_live)
📦 Found 1 volume(s):
   - migration-pgtest1-pgtest1-disk-0 (a67a3725-ab7a-4db7-8658-8a3012500233)
📤 Detaching volume: migration-pgtest1-pgtest1-disk-0...
⏳ Waiting for detach operation to complete...
✅ Volume detached successfully
🗑️ Performing cascade delete of VM context...
✅ Cleanup completed successfully for VM 'pgtest1'
```

