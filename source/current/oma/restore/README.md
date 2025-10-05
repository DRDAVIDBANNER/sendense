# File-Level Restore System

**Task 4: File-Level Restore Implementation**  
**Status:** Phase 1 Complete ✅  
**Date:** 2025-10-05

## Overview

The restore system provides file-level recovery from QCOW2 backup files. Customers can mount backups, browse filesystem contents, and download individual files or directories without full VM restoration.

## Architecture

### Components

1. **MountManager** (`mount_manager.go`)
   - Core QCOW2 mount operations via qemu-nbd
   - NBD device allocation (/dev/nbd0-7 for restore operations)
   - Filesystem detection and mounting (read-only)
   - Automatic resource management

2. **RestoreMountRepository** (`database/restore_mount_repository.go`)
   - Database operations via repository pattern (PROJECT_RULES compliance)
   - Mount tracking and lifecycle management
   - NBD device allocation tracking
   - Idle timeout detection

3. **Database Schema** (`migrations/20251005120000_add_restore_tables.up.sql`)
   - `restore_mounts` table with mount metadata
   - Foreign key to `backup_jobs` with CASCADE DELETE
   - Indexes for performance optimization

## NBD Device Allocation Strategy

Following Task 2 integration requirements:

- **Restore Operations:** `/dev/nbd0-7` (8 concurrent mounts)
- **Backup Operations:** `/dev/nbd8+` (separate allocation pool)

This prevents conflicts between restore mounts and backup exports.

## Mount Workflow

```
1. Validate backup exists in repository (Task 1 integration)
   ↓
2. Check mount limits (max 8 concurrent mounts)
   ↓
3. Allocate NBD device from restore pool (/dev/nbd0-7)
   ↓
4. Export QCOW2 via qemu-nbd --read-only
   ↓
5. Wait for NBD device availability
   ↓
6. Detect partition (usually nbdXp1)
   ↓
7. Detect filesystem type (ext4, xfs, ntfs, etc.)
   ↓
8. Mount filesystem to /mnt/sendense/restore/{uuid}
   ↓
9. Track mount in restore_mounts table
   ↓
10. Return mount_id and mount_path
```

## Usage Example

```go
// Initialize mount manager
mountRepo := database.NewRestoreMountRepository(db)
mountManager := restore.NewMountManager(mountRepo, repositoryManager)

// Mount a backup
req := &restore.MountRequest{
    BackupID: "backup-pgtest2-20251004120000",
}

mountInfo, err := mountManager.MountBackup(ctx, req)
if err != nil {
    log.Fatalf("Mount failed: %v", err)
}

fmt.Printf("Mounted at: %s\n", mountInfo.MountPath)
// Output: Mounted at: /mnt/sendense/restore/a1b2c3d4-...

// Browse files (Phase 2)
// files, err := fileBrowser.ListFiles(ctx, mountInfo.MountID, "/var/www")

// Unmount when done
err = mountManager.UnmountBackup(ctx, mountInfo.MountID)
```

## Security

- **Read-Only Mounts:** All backups mounted read-only (backup integrity)
- **Mount Isolation:** Each mount in separate directory (`/mnt/sendense/restore/{uuid}`)
- **Resource Limits:** Maximum 8 concurrent mounts (NBD device pool size)
- **Automatic Cleanup:** 1-hour idle timeout (configurable)
- **Path Validation:** File browser (Phase 2) validates paths against mount root

## Integration Points

### Task 1: Repository Infrastructure
- Uses `RepositoryManager.GetBackupFromAnyRepository()` to locate backups
- Supports backups in Local, NFS, CIFS repositories
- Handles backup file path resolution

### Task 2: NBD File Export
- Coordinates with NBD backup exports (separate device pools)
- Uses qemu-nbd for QCOW2 file exports
- No conflicts with backup operations

### Task 3: Backup Workflow
- Mounts QCOW2 files created by BackupEngine
- Works with full and incremental backups (via backing chain)
- Accesses backup_jobs table for metadata

## Database Schema

```sql
CREATE TABLE restore_mounts (
    id VARCHAR(64) PRIMARY KEY,              -- Mount UUID
    backup_id VARCHAR(64) NOT NULL,          -- FK to backup_jobs
    mount_path VARCHAR(512) NOT NULL,        -- /mnt/sendense/restore/{uuid}
    nbd_device VARCHAR(32) NOT NULL,         -- /dev/nbd0-7
    filesystem_type VARCHAR(32),             -- ext4, xfs, ntfs, etc.
    mount_mode ENUM('read-only'),            -- Always read-only
    status ENUM('mounting', 'mounted', 'unmounting', 'failed'),
    created_at TIMESTAMP,
    last_accessed_at TIMESTAMP,              -- For idle detection
    expires_at TIMESTAMP,                    -- created_at + 1 hour
    
    FOREIGN KEY (backup_id) REFERENCES backup_jobs(id) ON DELETE CASCADE
);
```

## Phase Status

### Phase 1: QCOW2 Mount Management ✅ COMPLETE
- [x] Mount Manager implementation
- [x] Database migration and schema
- [x] Repository pattern for database operations
- [x] NBD device allocation strategy
- [x] qemu-nbd integration
- [x] Filesystem detection and mounting

### Phase 2: File Browser API (Next)
- [ ] File listing service
- [ ] Directory traversal with security
- [ ] File metadata extraction
- [ ] Path validation and sanitization

### Phase 3: File Download & Extraction (Pending)
- [ ] HTTP streaming downloads
- [ ] Directory downloads as archives
- [ ] Progress tracking

### Phase 4: Safety & Cleanup (Pending)
- [ ] Automatic idle timeout cleanup
- [ ] Mount conflict resolution
- [ ] Resource monitoring

### Phase 5: API Integration (Pending)
- [ ] REST API handlers
- [ ] API documentation
- [ ] End-to-end integration testing

## Configuration

```go
const (
    RestoreNBDDeviceStart = 0              // /dev/nbd0
    RestoreNBDDeviceEnd   = 7              // /dev/nbd7
    RestoreMountBaseDir   = "/mnt/sendense/restore"
    DefaultIdleTimeout    = 1 * time.Hour  // Auto-cleanup after 1 hour
    DefaultMaxMounts      = 8              // Maximum concurrent mounts
)
```

## Error Handling

All operations use comprehensive error handling:

- **Mount Failures:** Automatic cleanup of partial mounts
- **NBD Exhaustion:** Clear error when device pool full
- **Filesystem Detection:** Continues with "unknown" if detection fails
- **Database Errors:** Repository pattern handles all DB operations

## Logging

Structured logging with log levels:

- **Info:** Mount/unmount operations, major milestones
- **Debug:** Detailed operation steps, device allocation
- **Warn:** Non-fatal errors, cleanup warnings
- **Error:** Critical failures requiring attention

## Testing

### Manual Testing
```bash
# 1. Mount a backup
curl -X POST http://localhost:8082/api/v1/restore/mount \
  -H "Content-Type: application/json" \
  -d '{"backup_id": "backup-pgtest2-20251004120000"}'

# 2. Verify mount
ls -la /mnt/sendense/restore/{mount-uuid}/

# 3. Unmount
curl -X DELETE http://localhost:8082/api/v1/restore/{mount-uuid}
```

### Integration Testing
- Mount multiple backups concurrently (up to 8)
- Verify NBD device allocation (no conflicts)
- Test automatic cleanup after idle timeout
- Verify mount reuse for same backup

## Future Enhancements

- **LVM Support:** Detect and mount LVM volumes
- **Multi-Partition:** Mount multiple partitions from same backup
- **Snapshot Browsing:** Browse incremental backup chains
- **Performance:** NBD caching for faster file access

---

**Phase 1 Implementation:** 2025-10-05  
**Next Phase:** Phase 2 - File Browser API  
**Project:** Sendense Phase 1 - VMware Backups
