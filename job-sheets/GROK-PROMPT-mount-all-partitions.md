# GROK: Mount All Partitions - File-Level Restore Enhancement

**Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-09-mount-all-partitions.md`

---

## 🎯 MISSION

**Current:** Mount service auto-selects only the LARGEST partition  
**Goal:** Mount ALL partitions simultaneously, show as folders in file browser

**User Experience:**
```
Before:
📁 Users/
📁 Program Files/
[Only main partition visible]

After:
📁 Partition 1 - Recovery (1.5GB)
📁 Partition 2 - EFI System (100MB)
📁 Partition 4 - Windows C: (100.4GB) ← Main data
   └── Click → Users, Program Files, Windows
[All partitions accessible]
```

---

## 🏗️ ARCHITECTURE

**Use Option B: Single Mount with Sub-Directories**

```
Mount Structure:
/mnt/sendense/restore/{mount_id}/
  ├── partition-1/  ← p1 mounted here
  ├── partition-2/  ← p2 mounted here
  ├── partition-4/  ← p4 mounted here
  └── partition-5/  ← p5 mounted here (if mountable)
```

**Benefits:**
- ONE database record
- Simple cleanup (unmount parent removes all)
- File browser sees natural hierarchy

---

## 🔧 IMPLEMENTATION

### Task 1: Database Migration

**File:** Create `sha/database/migrations/20251009160000_add_partition_metadata.up.sql`

```sql
ALTER TABLE restore_mounts 
ADD COLUMN partition_metadata JSON COMMENT 'Partition details for multi-partition mounts';
```

**File:** Create `sha/database/migrations/20251009160000_add_partition_metadata.down.sql`

```sql
ALTER TABLE restore_mounts 
DROP COLUMN partition_metadata;
```

**Execute:**
```bash
cd /home/oma_admin/sendense/source/current/sha
mysql -u oma_user -p'oma_password' -D migratekit_oma < database/migrations/20251009160000_add_partition_metadata.up.sql
```

---

### Task 2: Backend - Mount All Partitions

**File:** `sha/restore/mount_manager.go`

**Add struct:**
```go
type PartitionMount struct {
    PartitionName string
    DevicePath    string
    MountPath     string
    Size          int64
    Filesystem    string
    Label         string
}
```

**Add function (replace current `detectPartition`):**
```go
func (mm *MountManager) mountAllPartitions(nbdDevice, baseMountPath string) ([]*PartitionMount, error) {
    log.WithField("nbd_device", nbdDevice).Info("🔍 Detecting and mounting all partitions")
    
    // List all partitions
    cmd := exec.Command("lsblk", "-rno", "NAME,SIZE,FSTYPE,LABEL", nbdDevice)
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("failed to list partitions: %w", err)
    }
    
    var partitions []*PartitionMount
    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    partitionIndex := 1
    
    for _, line := range lines {
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }
        
        name := fields[0]
        // Skip base device (only process partitions)
        if !strings.Contains(name, "p") {
            continue
        }
        
        devicePath := "/dev/" + name
        size := mm.parseSizeToBytes(fields[1])
        
        // Skip tiny partitions (< 1MB)
        if size < 1024*1024 {
            continue
        }
        
        // Create mount subdirectory
        mountPath := filepath.Join(baseMountPath, fmt.Sprintf("partition-%d", partitionIndex))
        os.MkdirAll(mountPath, 0755)
        
        // Attempt to mount (skip if fails)
        if err := mm.mountFilesystem(devicePath, mountPath); err != nil {
            log.WithError(err).Warn("⚠️  Failed to mount partition - skipping")
            os.RemoveAll(mountPath)
            continue
        }
        
        fsType := ""
        label := ""
        if len(fields) >= 3 {
            fsType = fields[2]
        }
        if len(fields) >= 4 {
            label = strings.Join(fields[3:], " ")
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
            "partition": devicePath,
            "size":      mm.formatBytes(size),
        }).Info("✅ Partition mounted")
        
        partitionIndex++
    }
    
    if len(partitions) == 0 {
        return nil, fmt.Errorf("no mountable partitions found")
    }
    
    log.WithField("count", len(partitions)).Info("✅ All partitions mounted")
    return partitions, nil
}
```

**Modify `MountBackup()` method:**
```go
// OLD CODE (remove):
partition := mm.detectPartition(nbdDevice)
mm.mountFilesystem(partition, mountPath)

// NEW CODE (replace with):
partitions, err := mm.mountAllPartitions(nbdDevice, mountPath)
if err != nil {
    // Cleanup and return error
    mm.disconnectNBD(nbdDevice)
    return nil, fmt.Errorf("failed to mount partitions: %w", err)
}

// Store partition metadata as JSON
partitionInfo := make([]map[string]interface{}, len(partitions))
for i, p := range partitions {
    partitionInfo[i] = map[string]interface{}{
        "partition_name": p.PartitionName,
        "size":           p.Size,
        "filesystem":     p.Filesystem,
        "label":          p.Label,
        "mount_path":     filepath.Base(p.MountPath), // "partition-1"
    }
}
metadataJSON, _ := json.Marshal(map[string]interface{}{
    "partitions": partitionInfo,
})
metadataStr := string(metadataJSON)

// Pass to mount record creation
mount := &database.RestoreMount{
    // ... existing fields ...
    PartitionMetadata: &metadataStr, // NEW FIELD
}
```

---

### Task 3: Backend - Update Models

**File:** `sha/database/models.go`

**Modify `RestoreMount` struct:**
```go
type RestoreMount struct {
    ID                string     `db:"id" gorm:"primaryKey" json:"id"`
    BackupDiskID      int64      `db:"backup_disk_id" gorm:"column:backup_disk_id;not null" json:"backup_disk_id"`
    MountPath         string     `db:"mount_path" gorm:"column:mount_path" json:"mount_path"`
    NBDDevice         string     `db:"nbd_device" gorm:"column:nbd_device" json:"nbd_device"`
    FilesystemType    string     `db:"filesystem_type" gorm:"column:filesystem_type" json:"filesystem_type"`
    MountMode         string     `db:"mount_mode" gorm:"column:mount_mode" json:"mount_mode"`
    Status            string     `db:"status" gorm:"column:status" json:"status"`
    CreatedAt         time.Time  `db:"created_at" gorm:"column:created_at" json:"created_at"`
    LastAccessedAt    time.Time  `db:"last_accessed_at" gorm:"column:last_accessed_at" json:"last_accessed_at"`
    ExpiresAt         *time.Time `db:"expires_at" gorm:"column:expires_at" json:"expires_at,omitempty"`
    PartitionMetadata *string    `db:"partition_metadata" gorm:"column:partition_metadata;type:json" json:"partition_metadata,omitempty"` // NEW
}
```

---

### Task 4: Backend - File Browser Logic

**File:** `sha/restore/file_browser.go`

**Modify `ListFiles()` function:**
```go
func (fb *FileBrowser) ListFiles(ctx context.Context, req *ListFilesRequest) (*ListFilesResponse, error) {
    // ... existing validation (mount lookup, status check) ...
    
    // Check for multi-partition metadata
    if mount.PartitionMetadata != nil && *mount.PartitionMetadata != "" {
        var metadata map[string]interface{}
        json.Unmarshal([]byte(*mount.PartitionMetadata), &metadata)
        
        partitions, ok := metadata["partitions"].([]interface{})
        if ok && len(partitions) > 0 {
            // ROOT PATH: Show partition folders
            if req.Path == "/" {
                return fb.listPartitionFolders(mount, partitions), nil
            }
            
            // PARTITION PATH: Show files within partition
            if strings.HasPrefix(req.Path, "/partition-") {
                return fb.listFilesInPartition(mount, req.Path, req.Recursive)
            }
        }
    }
    
    // LEGACY: Single partition mount (backward compatibility)
    safePath, _ := fb.ValidateAndSanitizePath(mount.MountPath, req.Path)
    if req.Recursive {
        return fb.listFilesRecursive(safePath, req.Path)
    }
    return fb.listFilesSingle(safePath, req.Path)
}
```

**Add helper functions:**
```go
func (fb *FileBrowser) listPartitionFolders(mount *database.RestoreMount, partitions []interface{}) *ListFilesResponse {
    files := make([]*FileInfo, 0, len(partitions))
    
    for i, p := range partitions {
        partition := p.(map[string]interface{})
        partitionNum := i + 1
        size := int64(partition["size"].(float64))
        label := ""
        if partition["label"] != nil && partition["label"].(string) != "" {
            label = partition["label"].(string)
        }
        
        // Generate friendly name
        name := fmt.Sprintf("Partition %d", partitionNum)
        if label != "" {
            name = fmt.Sprintf("Partition %d - %s", partitionNum, label)
        }
        name = fmt.Sprintf("%s (%s)", name, fb.formatBytes(size))
        
        files = append(files, &FileInfo{
            Name:         name,
            Path:         fmt.Sprintf("/partition-%d", partitionNum),
            Type:         "directory",
            Size:         size,
            Mode:         "0755",
            ModifiedTime: mount.CreatedAt,
        })
    }
    
    return &ListFilesResponse{
        MountID:    mount.ID,
        Path:       "/",
        Files:      files,
        TotalCount: len(files),
    }
}

func (fb *FileBrowser) listFilesInPartition(mount *database.RestoreMount, path string, recursive bool) (*ListFilesResponse, error) {
    // Extract partition folder: "/partition-1/Users" → "partition-1"
    parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
    partitionFolder := parts[0] // "partition-1"
    
    // Build filesystem path
    fsPath := filepath.Join(mount.MountPath, partitionFolder)
    if len(parts) > 1 {
        fsPath = filepath.Join(fsPath, strings.Join(parts[1:], "/"))
    }
    
    // List files
    if recursive {
        return fb.listFilesRecursive(fsPath, path)
    }
    return fb.listFilesSingle(fsPath, path)
}

func (fb *FileBrowser) formatBytes(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGT"[exp])
}
```

---

### Task 5: Backend - Unmount All Partitions

**File:** `sha/restore/mount_manager.go`

**Add function:**
```go
func (mm *MountManager) unmountAllPartitions(baseMountPath string) error {
    log.WithField("base_path", baseMountPath).Info("🧹 Unmounting all partitions")
    
    entries, err := os.ReadDir(baseMountPath)
    if err != nil {
        return fmt.Errorf("failed to read mount directory: %w", err)
    }
    
    for _, entry := range entries {
        if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "partition-") {
            continue
        }
        
        partitionPath := filepath.Join(baseMountPath, entry.Name())
        
        // Unmount partition
        cmd := exec.Command("sudo", "umount", partitionPath)
        if output, err := cmd.CombinedOutput(); err != nil {
            log.WithError(err).Warn("⚠️  Failed to unmount partition")
        }
        
        // Remove directory
        os.RemoveAll(partitionPath)
    }
    
    return nil
}
```

**Update `UnmountBackup()` method:**
```go
// OLD CODE:
mm.unmountFilesystem(mountPath)

// NEW CODE (replace with):
mm.unmountAllPartitions(mountPath) // Unmounts all partition subdirectories
```

---

## 🧪 TESTING

### Test 1: Mount and List Partitions
```bash
# Start fresh
mysql -u oma_user -p'oma_password' -D migratekit_oma -e "DELETE FROM restore_mounts;"

# Mount via GUI
# Select pgtest1 → Oct 9 backup → Click "Mount Backup"

# Check API response
curl "http://localhost:8082/api/v1/restore/{mount_id}/files?path=/" | jq

# Expected: 3-4 partition folders
{
  "files": [
    {"name": "Partition 1 - Recovery (1.5GB)", "path": "/partition-1", "type": "directory"},
    {"name": "Partition 2 - EFI System (100MB)", "path": "/partition-2", "type": "directory"},
    {"name": "Partition 4 - Windows C: (100.4GB)", "path": "/partition-4", "type": "directory"}
  ]
}
```

### Test 2: Navigate Into Partition
```bash
curl "http://localhost:8082/api/v1/restore/{mount_id}/files?path=/partition-4" | jq

# Expected: Users, Program Files, Windows folders
{
  "files": [
    {"name": "Users", "path": "/partition-4/Users", "type": "directory"},
    {"name": "Program Files", "path": "/partition-4/Program Files", "type": "directory"}
  ]
}
```

### Test 3: GUI Navigation
1. Mount backup
2. See partition folders at root
3. Click "Partition 4 - Windows C:"
4. See Windows filesystem
5. Download a file
6. Navigate back to root (breadcrumb)
7. Click "Partition 1 - Recovery"
8. See recovery files

---

## ✅ DEFINITION OF DONE

- [ ] Database migration executed successfully
- [ ] `mountAllPartitions()` detects and mounts all partitions
- [ ] Partition metadata stored in JSON field
- [ ] File browser lists partitions at root path
- [ ] File browser lists files within partition paths
- [ ] Unmount removes all partition directories
- [ ] GUI shows partition folders with size labels
- [ ] Downloads work from any partition
- [ ] Backend restarted without errors
- [ ] End-to-end test passes

---

## 🚨 CRITICAL RULES

1. **Skip failing partitions** - Don't fail entire mount if one partition fails
2. **JSON metadata** - Store partition info in `partition_metadata` column
3. **Backward compatibility** - Single-partition mounts (no metadata) still work
4. **Read-only** - All partition mounts use `-o ro` flag
5. **Cleanup** - Unmount ALL partitions before NBD disconnect

---

## 📁 FILES TO MODIFY

**Backend (SHA):**
1. `database/migrations/20251009160000_add_partition_metadata.up.sql` (NEW)
2. `database/migrations/20251009160000_add_partition_metadata.down.sql` (NEW)
3. `database/models.go` (add PartitionMetadata field)
4. `restore/mount_manager.go` (add mountAllPartitions, unmountAllPartitions)
5. `restore/file_browser.go` (add listPartitionFolders, listFilesInPartition)

**Frontend (if needed):**
6. `sendense-gui/components/features/restore/FileBrowser.tsx` (breadcrumb logic)

---

**Database Credentials:**
- User: `oma_user`
- Password: `oma_password`
- Database: `migratekit_oma`

**Test VM:** pgtest1 (102GB Windows disk with 5 partitions)

---

**START HERE:** Read full job sheet, execute migration, modify backend, test, deploy.

**Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-09-mount-all-partitions.md` (COMPREHENSIVE)


