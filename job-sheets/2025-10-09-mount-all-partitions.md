# Mount All Partitions - File-Level Restore Enhancement

**Date:** October 9, 2025  
**Priority:** HIGH  
**Complexity:** Medium (30-45 minutes)  
**Estimated Impact:** High user value  
**Status:** Ready for Implementation

---

## 🎯 OBJECTIVE

**Current Behavior:** Mount service auto-selects and mounts only the LARGEST partition
- Windows: Mounts p4 (100GB C: drive), skips p1 (recovery), p2 (EFI), p3 (MSR)
- User sees only main partition, cannot access other partitions

**Desired Behavior:** Mount ALL partitions simultaneously
- Mount p1, p2, p3, p4, p5 (all partitions on the disk)
- File browser shows partitions as top-level folders
- User can browse and download from ANY partition

**User Experience:**
```
File Browser Root:
📁 Partition 1 - Recovery (1.5GB)
   ├── $RECYCLE.BIN
   ├── $WINRE_BACKUP_PARTITION.MARKER
   └── Recovery/
📁 Partition 2 - EFI System (100MB)
   └── EFI/
       └── Microsoft/
📁 Partition 4 - Windows C: (100.4GB) ← Main data
   ├── Users/
   ├── Program Files/
   └── Windows/
📁 Partition 5 - System Reserved (256KB)
```

---

## 🏗️ ARCHITECTURE OVERVIEW

### Current Architecture (Single Partition)
```
Mount Request → Detect Partitions → Select Largest → Mount One → Return Mount ID
```

### New Architecture (All Partitions)
```
Mount Request → Detect Partitions → Mount Each → Create Virtual Root → Return Mount ID
                                         ↓
                        p1 → /mnt/.../partition-1/
                        p2 → /mnt/.../partition-2/
                        p4 → /mnt/.../partition-4/
                        p5 → /mnt/.../partition-5/
```

### Key Architectural Decisions

**Option A: Multiple Mount Records** (Database-heavy)
- Create separate `restore_mounts` record for each partition
- Each partition gets its own mount_id
- File browser needs to aggregate multiple mount_ids

**Option B: Single Mount with Sub-Directories** ✅ **RECOMMENDED**
- Create ONE `restore_mounts` record for the backup
- Mount all partitions under subdirectories: `/mnt/restore/{mount_id}/partition-{N}/`
- File browser sees single mount path with partition folders
- Simpler database, simpler cleanup, better UX

**We'll use Option B** for simplicity and better UX.

---

## 📐 TECHNICAL DESIGN

### 1. Mount Manager Changes

**File:** `restore/mount_manager.go`

**Current `detectPartition()` function:**
- Returns: Single partition path (string)
- Logic: Finds largest partition

**New `detectAndMountAllPartitions()` function:**
- Returns: Map of partition paths (map[string]string)
- Logic: Detects all partitions, mounts each to subdirectory

**Implementation:**

```go
// PartitionMount represents a mounted partition
type PartitionMount struct {
    PartitionName string // "nbd0p1", "nbd0p4", etc.
    DevicePath    string // "/dev/nbd0p1"
    MountPath     string // "/mnt/restore/{mount_id}/partition-1"
    Size          int64  // Partition size in bytes
    Filesystem    string // "ntfs", "ext4", "vfat", etc.
    Label         string // Optional: partition label
}

// mountAllPartitions mounts all partitions from an NBD device
func (mm *MountManager) mountAllPartitions(nbdDevice, baseMountPath string) ([]*PartitionMount, error) {
    log.WithField("nbd_device", nbdDevice).Info("🔍 Detecting and mounting all partitions")
    
    // Step 1: List all partitions using lsblk
    cmd := exec.Command("lsblk", "-rno", "NAME,SIZE,FSTYPE,LABEL", nbdDevice)
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("failed to list partitions: %w", err)
    }
    
    // Step 2: Parse partitions
    var partitions []*PartitionMount
    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    
    partitionIndex := 1
    for _, line := range lines {
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }
        
        name := fields[0]
        sizeStr := fields[1]
        fsType := ""
        label := ""
        
        if len(fields) >= 3 {
            fsType = fields[2]
        }
        if len(fields) >= 4 {
            label = strings.Join(fields[3:], " ")
        }
        
        // Skip the base device (only process partitions)
        if !strings.Contains(name, "p") {
            continue
        }
        
        devicePath := "/dev/" + name
        size := mm.parseSizeToBytes(sizeStr)
        
        // Skip very small partitions (< 1MB) - usually reserved/alignment
        if size < 1024*1024 {
            log.WithField("partition", devicePath).Debug("⏭️  Skipping tiny partition")
            continue
        }
        
        // Create mount subdirectory
        mountPath := filepath.Join(baseMountPath, fmt.Sprintf("partition-%d", partitionIndex))
        if err := os.MkdirAll(mountPath, 0755); err != nil {
            log.WithError(err).Warn("Failed to create partition mount directory")
            continue
        }
        
        // Attempt to mount partition
        if err := mm.mountFilesystem(devicePath, mountPath); err != nil {
            log.WithFields(log.Fields{
                "partition": devicePath,
                "error":     err,
            }).Warn("⚠️  Failed to mount partition - skipping")
            
            // Clean up failed mount directory
            os.RemoveAll(mountPath)
            continue
        }
        
        partition := &PartitionMount{
            PartitionName: name,
            DevicePath:    devicePath,
            MountPath:     mountPath,
            Size:          size,
            Filesystem:    fsType,
            Label:         label,
        }
        
        partitions = append(partitions, partition)
        
        log.WithFields(log.Fields{
            "partition":  devicePath,
            "mount_path": mountPath,
            "size":       mm.formatBytes(size),
            "filesystem": fsType,
            "label":      label,
        }).Info("✅ Partition mounted successfully")
        
        partitionIndex++
    }
    
    if len(partitions) == 0 {
        return nil, fmt.Errorf("no mountable partitions found")
    }
    
    log.WithField("partition_count", len(partitions)).Info("✅ All partitions mounted")
    return partitions, nil
}
```

**Modification to `MountBackup()` method:**
```go
// OLD: Single partition mount
partition := mm.detectPartition(nbdDevice)
mm.mountFilesystem(partition, mountPath)

// NEW: Multi-partition mount
partitions, err := mm.mountAllPartitions(nbdDevice, mountPath)
if err != nil {
    return nil, fmt.Errorf("failed to mount partitions: %w", err)
}

// Store partition info in mount record (JSON metadata)
partitionInfo := make([]map[string]interface{}, len(partitions))
for i, p := range partitions {
    partitionInfo[i] = map[string]interface{}{
        "partition_name": p.PartitionName,
        "size":           p.Size,
        "filesystem":     p.Filesystem,
        "label":          p.Label,
        "mount_path":     p.MountPath,
    }
}

// Store in restore_mounts.metadata field (JSON)
metadataJSON, _ := json.Marshal(map[string]interface{}{
    "partitions": partitionInfo,
})
```

---

### 2. Database Schema Changes

**Table:** `restore_mounts`

**Current Schema:**
```sql
CREATE TABLE restore_mounts (
    id VARCHAR(64) PRIMARY KEY,
    backup_disk_id BIGINT NOT NULL,
    mount_path VARCHAR(512) NOT NULL,
    nbd_device VARCHAR(32) NOT NULL,
    filesystem_type VARCHAR(32),
    mount_mode VARCHAR(16),
    status VARCHAR(32),
    created_at TIMESTAMP,
    last_accessed_at TIMESTAMP,
    expires_at TIMESTAMP
);
```

**Option 1: Add metadata column** ✅ **RECOMMENDED**
```sql
ALTER TABLE restore_mounts 
ADD COLUMN partition_metadata JSON COMMENT 'Partition details for multi-partition mounts';
```

**Option 2: New table (more complex, not needed)**
```sql
CREATE TABLE restore_mount_partitions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    mount_id VARCHAR(64) NOT NULL,
    partition_name VARCHAR(32),
    device_path VARCHAR(64),
    mount_path VARCHAR(512),
    size_bytes BIGINT,
    filesystem VARCHAR(32),
    label VARCHAR(128),
    FOREIGN KEY (mount_id) REFERENCES restore_mounts(id) ON DELETE CASCADE
);
```

**Decision:** Use Option 1 (metadata column) - simpler, cleaner, less joins

---

### 3. File Browser Changes

**File:** `restore/file_browser.go`

**Current `ListFiles()` behavior:**
- Lists files directly in mount path
- Returns flat file list

**New `ListFiles()` behavior:**
- If path == "/", list partition folders as virtual directories
- If path starts with "/partition-N/", list files in that partition
- Seamless navigation between partitions

**Implementation:**

```go
func (fb *FileBrowser) ListFiles(ctx context.Context, req *ListFilesRequest) (*ListFilesResponse, error) {
    // ... existing validation ...
    
    // Check if mount has multi-partition metadata
    var partitionMetadata []map[string]interface{}
    if mount.PartitionMetadata != nil {
        json.Unmarshal([]byte(*mount.PartitionMetadata), &partitionMetadata)
    }
    
    // ROOT PATH: Show partition folders
    if req.Path == "/" && len(partitionMetadata) > 0 {
        return fb.listPartitionFolders(mount, partitionMetadata), nil
    }
    
    // PARTITION PATH: Show files within partition
    if strings.HasPrefix(req.Path, "/partition-") {
        return fb.listFilesInPartition(mount, req.Path, req.Recursive)
    }
    
    // LEGACY: Single partition mount (backward compatibility)
    return fb.listFilesSingle(mount.MountPath, req.Path)
}

func (fb *FileBrowser) listPartitionFolders(mount *RestoreMount, metadata []map[string]interface{}) *ListFilesResponse {
    files := make([]*FileInfo, 0, len(metadata))
    
    for i, partition := range metadata {
        partitionNum := i + 1
        size := int64(partition["size"].(float64))
        fsType := partition["filesystem"].(string)
        label := ""
        if partition["label"] != nil {
            label = partition["label"].(string)
        }
        
        // Generate friendly name
        name := fmt.Sprintf("Partition %d", partitionNum)
        if label != "" {
            name = fmt.Sprintf("Partition %d - %s", partitionNum, label)
        }
        name = fmt.Sprintf("%s (%s)", name, fb.formatBytes(size))
        
        fileInfo := &FileInfo{
            Name:         name,
            Path:         fmt.Sprintf("/partition-%d", partitionNum),
            Type:         "directory",
            Size:         size,
            Mode:         "0755",
            ModifiedTime: mount.CreatedAt,
            IsSymlink:    false,
        }
        
        files = append(files, fileInfo)
    }
    
    return &ListFilesResponse{
        MountID:    mount.ID,
        Path:       "/",
        Files:      files,
        TotalCount: len(files),
    }
}

func (fb *FileBrowser) listFilesInPartition(mount *RestoreMount, path string, recursive bool) (*ListFilesResponse, error) {
    // Extract partition number from path: "/partition-1/Users" → 1
    parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
    partitionPath := "/" + parts[0] // "/partition-1"
    
    // Build actual filesystem path
    fsPath := filepath.Join(mount.MountPath, partitionPath)
    if len(parts) > 1 {
        fsPath = filepath.Join(fsPath, strings.Join(parts[1:], "/"))
    }
    
    // List files using existing logic
    if recursive {
        return fb.listFilesRecursive(fsPath, path)
    }
    return fb.listFilesSingle(fsPath, path)
}
```

---

### 4. Unmount Changes

**Current:** Unmount single partition, disconnect NBD

**New:** Unmount ALL partitions, then disconnect NBD

**Implementation:**

```go
func (mm *MountManager) unmountAllPartitions(baseMountPath string) error {
    log.WithField("base_path", baseMountPath).Info("🧹 Unmounting all partitions")
    
    // Find all partition mount directories
    entries, err := os.ReadDir(baseMountPath)
    if err != nil {
        return fmt.Errorf("failed to read mount directory: %w", err)
    }
    
    errors := []error{}
    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }
        
        if strings.HasPrefix(entry.Name(), "partition-") {
            partitionPath := filepath.Join(baseMountPath, entry.Name())
            
            // Unmount partition
            cmd := exec.Command("sudo", "umount", partitionPath)
            if output, err := cmd.CombinedOutput(); err != nil {
                log.WithFields(log.Fields{
                    "partition": partitionPath,
                    "error":     err,
                    "output":    string(output),
                }).Warn("⚠️  Failed to unmount partition")
                errors = append(errors, err)
            } else {
                log.WithField("partition", partitionPath).Info("✅ Partition unmounted")
            }
            
            // Remove mount directory
            os.RemoveAll(partitionPath)
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("failed to unmount %d partitions", len(errors))
    }
    
    return nil
}
```

---

### 5. Frontend Changes

**File:** `components/features/restore/FileBrowser.tsx`

**Current:** Shows files directly

**New:** Shows partitions at root level, files within partitions

**Implementation:**

```typescript
// FileBrowser component logic
const handleNavigate = (file: FileInfo) => {
  if (file.type === 'directory') {
    // Navigation to partition or subdirectory
    setCurrentPath(file.path);
  }
};

// Breadcrumb navigation
const getBreadcrumbs = () => {
  const parts = currentPath.split('/').filter(Boolean);
  const breadcrumbs = [{ name: 'Root', path: '/' }];
  
  let accumulatedPath = '';
  for (const part of parts) {
    accumulatedPath += '/' + part;
    
    // Friendly name for partitions
    let displayName = part;
    if (part.startsWith('partition-')) {
      displayName = `Partition ${part.split('-')[1]}`;
    }
    
    breadcrumbs.push({
      name: displayName,
      path: accumulatedPath,
    });
  }
  
  return breadcrumbs;
};
```

**Visual Design:**

```
┌─────────────────────────────────────────────────────┐
│ Restore > pgtest1 > Oct 9 Backup                    │
├─────────────────────────────────────────────────────┤
│ 📁 Root                                              │
├─────────────────────────────────────────────────────┤
│ 📁 Partition 1 - Recovery (1.5GB)                    │
│ 📁 Partition 2 - EFI System (100MB)                  │
│ 📁 Partition 4 - Windows C: (100.4GB)                │
│ 📁 Partition 5 - System Reserved (256KB)             │
└─────────────────────────────────────────────────────┘

Click on "Partition 4" →

┌─────────────────────────────────────────────────────┐
│ Root > Partition 4                                   │
├─────────────────────────────────────────────────────┤
│ 📁 Users                                             │
│ 📁 Program Files                                     │
│ 📁 Windows                                           │
│ 📄 pagefile.sys                                      │
└─────────────────────────────────────────────────────┘
```

---

## 🧪 TESTING PLAN

### Test 1: Multi-Partition Detection
```bash
# Expected: Detect all 5 partitions
lsblk -rno NAME,SIZE,FSTYPE,LABEL /dev/nbd0

Expected output:
nbd0 102G
nbd0p1 1.5G ntfs "New Volume"
nbd0p2 100M vfat 
nbd0p3 15.8M
nbd0p4 100.4G ntfs
nbd0p5 256K
```

### Test 2: All Partitions Mounted
```bash
# After mount, check mount points
ls -la /mnt/sendense/restore/{mount_id}/

Expected directories:
partition-1/  (1.5GB NTFS)
partition-2/  (100MB FAT32)
partition-3/  (15.8MB - might fail to mount)
partition-4/  (100GB NTFS)
partition-5/  (256KB - too small, skipped)
```

### Test 3: API - List Root (Partitions)
```bash
curl "http://localhost:8082/api/v1/restore/{mount_id}/files?path=/"

Expected response:
{
  "mount_id": "...",
  "path": "/",
  "files": [
    {
      "name": "Partition 1 - Recovery (1.5GB)",
      "path": "/partition-1",
      "type": "directory",
      "size": 1610612736
    },
    {
      "name": "Partition 2 - EFI System (100MB)",
      "path": "/partition-2",
      "type": "directory",
      "size": 104857600
    },
    {
      "name": "Partition 4 - Windows C: (100.4GB)",
      "path": "/partition-4",
      "type": "directory",
      "size": 107374182400
    }
  ],
  "total_count": 3
}
```

### Test 4: API - List Files Within Partition
```bash
curl "http://localhost:8082/api/v1/restore/{mount_id}/files?path=/partition-4"

Expected response:
{
  "mount_id": "...",
  "path": "/partition-4",
  "files": [
    {
      "name": "Users",
      "path": "/partition-4/Users",
      "type": "directory",
      ...
    },
    {
      "name": "Program Files",
      "path": "/partition-4/Program Files",
      "type": "directory",
      ...
    }
  ]
}
```

### Test 5: GUI Navigation
1. Mount pgtest1 backup
2. See 3-4 partition folders at root
3. Click "Partition 4 - Windows C:"
4. See Users, Program Files, Windows folders
5. Navigate into Users/Administrator/Desktop
6. Download a file
7. Navigate back to root
8. Click "Partition 1 - Recovery"
9. See recovery files
10. Unmount - all partitions unmounted

### Test 6: Download from Multiple Partitions
1. Download file from Partition 1 (recovery partition)
2. Download file from Partition 4 (main partition)
3. Both downloads should work independently

### Test 7: Unmount Cleanup
```bash
# After unmount, verify cleanup
ls /mnt/sendense/restore/

Expected: No directories left
```

---

## ✅ ACCEPTANCE CRITERIA

**Backend:**
- [ ] `mountAllPartitions()` detects all partitions using lsblk
- [ ] Each partition mounted to `/mnt/restore/{mount_id}/partition-{N}/`
- [ ] Partitions with mount failures are skipped (not blocking)
- [ ] Partition metadata stored in `restore_mounts.partition_metadata` (JSON)
- [ ] File browser lists partition folders at root path "/"
- [ ] File browser lists files within partition paths "/partition-N/..."
- [ ] Unmount removes all partition mounts
- [ ] NBD device disconnected after all partitions unmounted

**Frontend:**
- [ ] Root displays partition folders with size and label
- [ ] Clicking partition folder navigates into it
- [ ] Breadcrumb shows: Root > Partition 4 > Users > ...
- [ ] Downloads work from any partition
- [ ] Navigation back to root shows partition list again

**Error Handling:**
- [ ] If no partitions can be mounted → error message
- [ ] If some partitions fail → mount succeeds with available partitions
- [ ] Unmount handles partial mount failures gracefully

**Backward Compatibility:**
- [ ] Single-partition mounts (if metadata missing) still work
- [ ] Existing mount records continue to function

---

## 🚨 IMPORTANT NOTES

### Skip Unmountable Partitions
- Some partitions (MSR, reserved) cannot be mounted
- **Don't fail the entire mount** if one partition fails
- Log warnings and continue with other partitions

### Partition Labeling
- Use `lsblk` LABEL field if available
- Fallback to generic "Partition N" if no label
- Include size in display name for clarity

### Filesystem Detection
- Use `lsblk FSTYPE` to detect filesystem
- Common types: ntfs, ext4, vfat, xfs, btrfs
- Some may not be supported - skip gracefully

### Performance Considerations
- Mounting 4-5 partitions adds ~1-2 seconds to mount time
- Acceptable tradeoff for better functionality
- NBD device is already connected (partitions visible immediately)

### Security
- All mounts remain read-only (`-o ro`)
- Path validation still applies within partitions
- No privilege escalation risk

---

## 📁 FILES TO MODIFY

### Backend
1. **`sha/restore/mount_manager.go`**
   - Add: `mountAllPartitions()` function (~80 lines)
   - Add: `unmountAllPartitions()` function (~30 lines)
   - Modify: `MountBackup()` to use multi-partition mount
   - Add: `formatBytes()` helper (already exists)

2. **`sha/database/models.go`**
   - Add: `PartitionMetadata *string` field to `RestoreMount` struct
   - GORM tag: `gorm:"column:partition_metadata;type:json"`

3. **`sha/restore/file_browser.go`**
   - Add: `listPartitionFolders()` function (~40 lines)
   - Add: `listFilesInPartition()` function (~30 lines)
   - Modify: `ListFiles()` to handle partition paths (~20 lines)

4. **`sha/database/migrations/`**
   - Create: `20251009160000_add_partition_metadata.up.sql`
   - Create: `20251009160000_add_partition_metadata.down.sql`

### Frontend
5. **`sendense-gui/components/features/restore/FileBrowser.tsx`**
   - Modify: Navigation logic for partitions
   - Modify: Breadcrumb generation
   - Add: Partition folder styling (icons, badges)

6. **`sendense-gui/src/features/restore/types/index.ts`**
   - Add: `PartitionInfo` interface
   - Modify: `RestoreMount` interface to include `partition_metadata`

---

## 🎯 EXPECTED OUTCOME

**Before:**
```
File Browser:
📁 $RECYCLE.BIN
📁 Recovery
📄 $WINRE_BACKUP_PARTITION.MARKER
[Only recovery partition visible]
```

**After:**
```
File Browser:
📁 Partition 1 - Recovery (1.5GB)
📁 Partition 2 - EFI System (100MB)
📁 Partition 4 - Windows C: (100.4GB)
   └── Click → Users, Program Files, Windows
📁 Partition 5 - System Reserved (256KB)
[All partitions accessible]
```

---

## 🚀 ROLLOUT PLAN

1. **Database Migration** (5 min)
   - Add `partition_metadata` column to `restore_mounts`
   - Test migration on development database

2. **Backend Implementation** (20-30 min)
   - Implement `mountAllPartitions()` function
   - Implement `listPartitionFolders()` function
   - Update `ListFiles()` logic
   - Test with pgtest1 (Windows multi-partition disk)

3. **Frontend Implementation** (10-15 min)
   - Update navigation logic
   - Update breadcrumb display
   - Add partition folder styling

4. **Testing** (10-15 min)
   - End-to-end test: mount → browse partitions → download
   - Test unmount cleanup
   - Verify backward compatibility

5. **Documentation** (5 min)
   - Update CHANGELOG.md
   - Update PHASE_1_CONTEXT_HELPER.md

**Total Time:** 50-70 minutes (conservative estimate)

---

## 📚 REFERENCE DOCUMENTS

- Current mount manager: `sha/restore/mount_manager.go` (detectPartition function)
- File browser API: `sha/restore/file_browser.go` (ListFiles function)
- Database schema: `sha/database/migrations/`
- Frontend file browser: `sendense-gui/components/features/restore/FileBrowser.tsx`

---

**READY FOR IMPLEMENTATION** 🚀


