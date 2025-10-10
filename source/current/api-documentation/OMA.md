OMA API Endpoints (OMA)

Base: /api/v1 (router in `oma/api/server.go`)

Authentication
- POST /auth/login → `handlers.Auth.Login`
  - Description: Issue bearer token for OMA API
  - Callsites: GUI/auth expected; no direct backend calls detected
  - Classification: Key (gateway)

Health/Swagger
- GET /health → inline `handleHealth`
  - Classification: Auxiliary
- GET /swagger/* → swagger handler
  - Classification: Auxiliary

VM Inventory
- GET /vms → `handlers.VM.List`
- POST /vms/inventory → `handlers.VM.ReceiveInventory`
- GET /vms/{id} → `handlers.VM.GetByID`
  - Classification: Key (GUI-driven)

Replications
- GET /replications → `handlers.Replication.List`
- POST /replications → `handlers.Replication.Create`
- GET /replications/{id} → `handlers.Replication.GetByID`
- PUT /replications/{id} → `handlers.Replication.Update`
- DELETE /replications/{id} → `handlers.Replication.Delete`
  - Callsites: `vma/client/oma_client.go` (POST), scheduler service (POST), unified failover engine (POST)
  - Classification: Key

ChangeID (CBT)
- GET /replications/changeid?vm_path={}&disk_id={} → `handlers.Replication.GetPreviousChangeID`
  - Callsites: migratekit `internal/target/cloudstack.go` and `vma/client`
  - Classification: Key
- POST /replications/{job_id}/changeid → `handlers.Replication.StoreChangeID`
  - Callsites: migratekit `internal/target/cloudstack.go` stores after completion
  - Classification: Key

Progress Proxy and Job Status
- GET /progress/{job_id} → `handlers.Replication.GetVMAProgressProxy`
  - Callsites: GUI; OMA constructs `http://localhost:9081/api/v1/progress/{job_id}` to VMA
  - Classification: Key
- GET /replications/{job_id}/progress → `handlers.Replication.GetReplicationProgress`
  - Classification: Key (enhanced progress via DB + VMA fields)

VM-Centric Architecture
- GET /vm-contexts → `handlers.VMContext.ListVMContexts`
  - Description: List all VM contexts with group membership information (supports multi-group VMs)
  - Request: None
  - Response: { vm_contexts: [VMContextWithGroups], count: number }
  - VMContextWithGroups includes: VMReplicationContext fields PLUS groups: [GroupMembershipInfo], group_count: number
  - GroupMembershipInfo: { group_id, group_name, priority, enabled }
  - **Multi-Group Support**: VMs can be in multiple groups simultaneously
  - Database: Queries vm_replication_contexts + vm_group_memberships + vm_machine_groups
  - Classification: Key (GUI relies on this for VM table with group badges)
  
- GET /vm-contexts/{vm_name} → `handlers.VMContext.GetVMContext`
  - Classification: Key (individual VM details)
  
- GET /vm-contexts/by-id/{context_id} → `handlers.VMContext.GetVMContextByID`
  - Description: Get VM context by context_id (for individual VM flows)
  - Request: context_id in URL path
  - Response: { context: VMReplicationContext, job_history: [], disks: [], cbt_history: [] }
  - Classification: Key (Protection Flows GUI integration)
  
- GET /vm-contexts/{context_id}/recent-jobs → `handlers.VMContext.GetRecentJobs`
  - Classification: Key (job history)

Discovery (Enhanced) - VM Discovery Without Immediate Replication
- POST /discovery/discover-vms → `handlers.EnhancedDiscovery.DiscoverVMs`
  - Description: Discover VMs from vCenter via VMA with optional immediate context creation
  - Request: { credential_id?: number, vcenter?: string, username?: string, password?: string, datacenter?: string, filter?: string, selected_vms?: string[], create_context: boolean }
  - Response: { discovered_vms: [DiscoveredVMInfo], addition_result?: BulkAddResult, discovery_count, processing_time, status, message }
  - DiscoveredVMInfo includes: id, name, path, power_state, guest_os, memory_mb, num_cpu, vmx_version, **disks: [VMADiskInfo]**, **networks: [VMANetworkInfo]**, existing, context_id
  - VMADiskInfo: { id, label, path, size_gb, capacity_bytes, datastore }
  - VMANetworkInfo: { label, network_name, mac_address }
  - Authentication: Required
  - Callsites: GUI VMDiscoveryModal, scheduler service
  - Classification: **Key** (primary discovery endpoint)
  - Notes: Supports both saved credentials (credential_id) and manual entry; returns full disk/network metadata for GUI display

- POST /discovery/add-vms → `handlers.EnhancedDiscovery.AddVMs`
  - Description: Add specific VMs to management by name (creates vm_replication_contexts without jobs)
  - Request: { credential_id?: number, vcenter?: string, username?: string, password?: string, datacenter?: string, vm_names: string[] (required), added_by?: string }
  - Response: { success, message, vms_added, vms_failed, total_vms, added_at, processed_vms: [ProcessedVMInfo] }
  - ProcessedVMInfo: { vm_name, success, context_id?, error? }
  - Authentication: Required
  - Callsites: GUI "Add to Management" workflow
  - Classification: **Key** (preferred bulk add method)
  - ⚠️ **IMPORTANT:** This is the CORRECT endpoint for GUI bulk add operations
  - ✅ **Supports credential_id** - works with saved VMware credentials
  - ✅ Accepts VM names array (vm_names field)
  - ✅ Returns detailed per-VM success/failure
  - ✅ Creates vm_replication_contexts with auto_added=true flag
  - Database Impact: Writes vm_replication_contexts table with full VM metadata (cpu_count, memory_mb, os_type, power_state)

- POST /discovery/bulk-add → `handlers.EnhancedDiscovery.BulkAddVMs`
  - Description: **LEGACY** bulk add requiring manual vCenter credentials
  - Request: { vcenter: string (required), username: string (required), password: string (required), datacenter: string (required), filter?: string, selected_vms: string[] (required) }
  - Response: BulkAddResult { total_requested, successfully_added, skipped, failed, added_vms, skipped_vms, failed_vms, discovery_duration, processing_duration }
  - Authentication: Required
  - Classification: **Auxiliary/Legacy** (prefer /discovery/add-vms instead)
  - ⚠️ **DO NOT USE FROM GUI** - This endpoint does NOT support credential_id
  - ❌ Requires explicit vcenter/username/password in every request
  - ❌ No saved credential support
  - Notes: Kept for backward compatibility; new code should use /discovery/add-vms

- GET /discovery/ungrouped-vms → `handlers.EnhancedDiscovery.GetUngroupedVMs`
  - Description: List VMs added to management but not assigned to any protection group
  - Response: { vms: [UngroupedVMInfo], count, retrieved_at }
  - UngroupedVMInfo: { context_id, vm_name, vm_path, vcenter_host, datacenter, current_status, auto_added, scheduler_enabled, cpu_count, memory_mb, os_type, power_state, created_at, last_job_at }
  - Authentication: Required
  - Classification: Key
  - Database Query: Reads vm_replication_contexts WHERE context_id NOT IN (SELECT vm_context_id FROM vm_group_memberships)

- POST /discovery/preview → `handlers.EnhancedDiscovery.GetDiscoveryPreview`
  - Description: Preview VMs that would be discovered without creating contexts
  - Request: { vcenter, username, password, datacenter, filter? }
  - Response: { vms: [DiscoveredVMInfo], total_discovered, new_vms, existing_vms, processing_time, vcenter, datacenter, filter? }
  - Authentication: Required
  - Classification: Key
  - Notes: Read-only operation, no database writes

- GET /vm-contexts/ungrouped → alias to ungrouped-vms
  - Classification: Auxiliary (redirect endpoint)
  
  **Architecture Notes:**
  - All discovery operations use VMA `/api/v1/discover` endpoint via SSH tunnel (localhost:9081)
  - VM contexts created without replication jobs (no entries in replication_jobs table)
  - Supports incremental discovery: detects existing VMs and skips them
  - All added VMs have auto_added=true and scheduler_enabled=true flags
  - Full VM metadata captured: CPU, memory, OS type, power state, disk info, network info
  
  **Callsites:**
  - GUI: VMDiscoveryModal uses discover-vms (discovery) → add-vms (bulk add)
  - Scheduler Service: Uses discover-vms with create_context=true for automated discovery
  - Machine Group Management: Uses ungrouped-vms to show available VMs for grouping
  
  **Database Impact:**
  - Writes: vm_replication_contexts (with auto_added=1, scheduler_enabled=1, ossea_config_id auto-assigned)
  - Reads: Checks existing contexts by vm_name to prevent duplicates
  - Foreign Keys: ossea_config_id references ossea_configs table

VMware Credentials Management
- GET /vmware-credentials → list
- POST /vmware-credentials → create
- GET /vmware-credentials/{id} → get
- PUT /vmware-credentials/{id} → update
- DELETE /vmware-credentials/{id} → delete
- PUT /vmware-credentials/{id}/set-default → set default
- POST /vmware-credentials/{id}/test → test
- GET /vmware-credentials/default → get default
  - Handler: `handlers.VMwareCredentials.*`
  - Classification: Key (feeds discovery flows)

CloudStack (OSSEA) Settings
- POST /settings/cloudstack/test-connection → `handlers.CloudStackSettings.TestConnection`
- POST /settings/cloudstack/detect-oma-vm → `handlers.CloudStackSettings.DetectOMAVM`
- GET /settings/cloudstack/networks → `handlers.CloudStackSettings.ListNetworks`
- POST /settings/cloudstack/validate → `handlers.CloudStackSettings.ValidateSettings`
- POST /settings/cloudstack/discover-all → `handlers.CloudStackSettings.DiscoverAllResources`
  - Classification: Key/Auxiliary (setup flows)

OSSEA Config
- POST /ossea/config → `handlers.OSSEA.HandleConfig`
- POST /ossea/discover-resources → `handlers.StreamlinedOSSEA.DiscoverResources`
- POST /ossea/config-streamlined → `handlers.StreamlinedOSSEA.SaveStreamlinedConfig`
  - Classification: Key (platform setup)

Network Mapping and Service Offerings
- POST /network-mappings → create
- GET /network-mappings → list all
- GET /network-mappings/{vm_id} → get by VM
- GET /network-mappings/{vm_id}/status → status
- DELETE /network-mappings/{vm_id}/{source_network_name} → delete
- GET /networks/available → list OSSEA networks
- POST /networks/resolve → resolve network IDs
- GET /service-offerings/available → list offerings
  - Handlers: `handlers.NetworkMapping.*`
  - Classification: Key

Failover (Enhanced + Unified)
- POST /failover/live → `handlers.Failover.InitiateEnhancedLiveFailover`
- POST /failover/test → `handlers.Failover.InitiateEnhancedTestFailover`
- DELETE /failover/test/{job_id} → `handlers.Failover.EndTestFailover`
- POST /failover/cleanup/{vm_name} → `handlers.Failover.CleanupTestFailover`
- POST /failover/{vm_name}/cleanup-failed → `handlers.Failover.CleanupFailedExecution`
- GET /failover/{job_id}/status → `handlers.Failover.GetFailoverJobStatus`
- GET /failover/{vm_id}/readiness → `handlers.Failover.ValidateFailoverReadiness`
- GET /failover/jobs → `handlers.Failover.ListFailoverJobs`
- POST /failover/unified → unified orchestrator
- GET /failover/preflight/config/{failover_type}/{vm_name} → preflight config
- POST /failover/preflight/validate → preflight validate
- POST /failover/rollback → enhanced rollback
- GET /failover/rollback/decision/{failover_type}/{vm_name} → rollback decision
  - Callsites: unified engine invokes VMA `/discover` and OMA `/replications`; cleanup uses Volume Daemon APIs
  - Classification: Key (core), with some Auxiliary (preflight/rollback decision)

Scheduler Ecosystem
- POST /schedules, GET /schedules, GET/PUT/DELETE /schedules/{id}
- POST /schedules/{id}/enable, POST /schedules/{id}/trigger, GET /schedules/{id}/executions
  - Handlers: `handlers.ScheduleManagement.*`
  - Classification: Key (automation)

Machine Groups and VM Assignments
- CRUD /machine-groups and VM assignment endpoints
  - Handlers: `handlers.MachineGroupManagement.*`, `handlers.VMGroupAssignment.*`
  - Classification: Key

Validation
- GET /vms/{vm_id}/failover-readiness
- GET /vms/{vm_id}/sync-status
- GET /vms/{vm_id}/network-mapping-status
- GET /vms/{vm_id}/volume-status
- GET /vms/{vm_id}/active-jobs
- GET /vms/{vm_id}/configuration-check
  - Handlers: `handlers.Validation.*` (also exposes RegisterValidationRoutes)
  - Classification: Key/Auxiliary

Debug
- GET /debug/health, /debug/failover-jobs, /debug/endpoints, /debug/logs
  - Classification: Auxiliary

VMA Enrollment (OMA side)
- Admin: POST /admin/vma/pairing-code; GET /admin/vma/pending; POST /admin/vma/approve/{id}; GET /admin/vma/active; POST /admin/vma/reject/{id}; DELETE /admin/vma/revoke/{id}; GET /admin/vma/audit
- Public: POST /vma/enroll; POST /vma/enroll/verify; GET /vma/enroll/result
  - Handler: `handlers.VMAReal.*`
  - Callsites: VMA `services/enrollment_client.go` uses public endpoints via 443 proxy
  - Classification: Key

Backup Repository Management (Storage Monitoring Day 4 - Implemented 2025-10-05)
- POST /api/v1/repositories → create new backup repository (Local, NFS, or CIFS)
  - Request: `CreateRepositoryRequest` with name, type, config JSON, immutability settings
  - Response: `RepositoryResponse` with storage info
  - Handler: `handlers.Repository.CreateRepository`
  - Authentication: Required
- GET /api/v1/repositories → list all repositories
  - Query Params: type (filter), enabled (true/false filter)
  - Response: Array of `RepositoryResponse` with storage info for each
  - Handler: `handlers.Repository.ListRepositories`
  - Authentication: Required
- GET /api/v1/repositories/{id}/storage → force immediate storage capacity check
  - Response: Fresh `StorageInfo` (total_bytes, used_bytes, available_bytes, mount_point)
  - Handler: `handlers.Repository.GetRepositoryStorage`
  - Authentication: Required
- POST /api/v1/repositories/test → test repository configuration without saving
  - Request: `TestRepositoryRequest` with type and config JSON
  - Response: `TestRepositoryResponse` with success flag and error details if failed
  - Handler: `handlers.Repository.TestRepository`
  - Authentication: Required
- DELETE /api/v1/repositories/{id} → delete repository configuration
  - Behavior: Fails with HTTP 409 Conflict if backups exist
  - Handler: `handlers.Repository.DeleteRepository`
  - Authentication: Required
  - Classification: Key (backup infrastructure)
  - Repository Types: Local (disk_path), NFS (server, export_path, mount_point), CIFS/SMB (server, share_name, credentials)
  - Backend: Uses `storage.RepositoryManager`, `storage.ConfigRepository`, `storage.MountManager`

Backup Policy Management (Backup Copy Engine Day 5 - Implemented 2025-10-05)
- POST /policies → `handlers.Policy.CreatePolicy`
  - Description: Create backup policy with 3-2-1 backup rule support
  - Request: PolicyRequest with name, enabled, primary_repository_id, retention_days, copy_rules
  - Response: PolicyResponse with generated ID and timestamps
  - Classification: Key (enterprise 3-2-1 backup rule)
- GET /policies → `handlers.Policy.ListPolicies`
  - Description: List all backup policies with copy rules
  - Response: Array of PolicyResponse objects
  - Classification: Key
- GET /policies/{id} → `handlers.Policy.GetPolicy`
  - Description: Get specific policy with copy rules
  - Response: PolicyResponse
  - Classification: Key
- DELETE /policies/{id} → `handlers.Policy.DeletePolicy`
  - Description: Delete policy and associated copy rules
  - Response: Success message
  - Classification: Key
- GET /backups/{id}/copies → `handlers.Policy.GetBackupCopies`
  - Description: List all copies of a backup across repositories
  - Response: Array of BackupCopyResponse objects
  - Callsites: GUI backup details view, copy status monitoring
  - Classification: Key (multi-repository backup tracking)
- POST /backups/{id}/copy → `handlers.Policy.TriggerBackupCopy`
  - Description: Manually trigger backup copy to repository
  - Request: { repository_id: string }
  - Response: { copy_id: string, status: "pending" }
  - Callsites: GUI manual copy operation
  - Classification: Key (manual backup replication)
  - Handler: `handlers.Policy.*`
  - Database: backup_policies, backup_copy_rules, backup_copies tables
  - Enterprise Features: Multi-repository copies, immutable storage support, automatic replication

File-Level Restore (Task 4 - Implemented 2025-10-05, v2.16.0+ Refactored 2025-10-08)
- POST /restore/mount → `handlers.Restore.MountBackup`
  - Description: Mount QCOW2 backup disk for file browsing via qemu-nbd (v2.16.0+ multi-disk support)
  - Request: 
    ```json
    {
      "backup_id": "backup-pgtest1-1759947871",
      "disk_index": 0
    }
    ```
  - Request Fields:
    - backup_id (string, required): Parent backup job ID from backup_jobs table
    - disk_index (int, required): Which disk to mount (0, 1, 2...) from multi-disk VM backup
  - Response: 
    ```json
    {
      "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398",
      "backup_id": "backup-pgtest1-1759947871",
      "backup_disk_id": 44,
      "disk_index": 0,
      "mount_path": "/mnt/sendense/restore/e4805a6f-8ee7-4f3c-8309-2f12362c7398",
      "nbd_device": "/dev/nbd0",
      "filesystem_type": "ntfs",
      "status": "mounted",
      "created_at": "2025-10-08T21:19:37+01:00",
      "expires_at": "2025-10-08T22:19:37+01:00"
    }
    ```
  - Response Fields:
    - mount_id: UUID for this mount session
    - backup_disk_id: Foreign key to backup_disks.id (v2.16.0+ architecture)
    - disk_index: Which disk was mounted (0-based)
    - mount_path: Filesystem mount point for browsing
    - nbd_device: NBD device used (/dev/nbd0-7 pool for restore)
    - filesystem_type: Detected filesystem (ntfs, ext4, xfs, etc.)
    - status: "mounting" or "mounted"
    - expires_at: Automatic cleanup time (1 hour from mount)
  - Classification: Key (customer file recovery)
  - Security: Read-only mounts, automatic cleanup after 1 hour idle
  - Architecture Changes (v2.16.0+):
    - Queries backup_disks table directly (replaces RepositoryManager)
    - FK to backup_disks.id ensures CASCADE DELETE integration
    - Supports multi-disk VM backups (select specific disk by index)
    - Unique constraint on backup_disk_id (one mount per disk)
  - Handler: `api/handlers/restore_handlers.go:MountBackup`
  - Service: `restore/mount_manager.go:MountBackup`
  - Database: restore_mounts table with backup_disk_id FK → backup_disks.id

- DELETE /restore/{mount_id} → `handlers.Restore.UnmountBackup`
  - Description: Unmount backup and release NBD device
  - Response: { message: "backup unmounted successfully" }
  - Classification: Key
  - Handler: `api/handlers/restore_handlers.go:UnmountBackup`
  - Service: `restore/mount_manager.go:UnmountBackup`

- GET /restore/mounts → `handlers.Restore.ListMounts`
  - Description: List all active restore mounts
  - Response: 
    ```json
    {
      "mounts": [
        {
          "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398",
          "backup_disk_id": 44,
          "mount_path": "/mnt/sendense/restore/...",
          "nbd_device": "/dev/nbd0",
          "status": "mounted",
          "created_at": "2025-10-08T21:19:37+01:00",
          "expires_at": "2025-10-08T22:19:37+01:00"
        }
      ],
      "count": 1
    }
    ```
  - Classification: Key
  - Handler: `api/handlers/restore_handlers.go:ListMounts`
  - Service: `restore/mount_manager.go:ListMounts`

- GET /restore/{mount_id}/files → `handlers.Restore.ListFiles`
  - Description: Browse files and directories within mounted backup (hierarchical navigation for GUI)
  - Query Params: 
    - path (string, default: "/"): Directory path to list
    - recursive (boolean, optional): Recursive listing (not recommended for large dirs)
  - Response: 
    ```json
    {
      "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398",
      "path": "/Recovery/WindowsRE",
      "files": [
        {
          "name": "ReAgent.xml",
          "path": "/Recovery/WindowsRE/ReAgent.xml",
          "type": "file",
          "size": 1129,
          "mode": "0777",
          "modified_time": "2025-09-02T06:21:20.1985298+01:00",
          "is_symlink": false
        },
        {
          "name": "winre.wim",
          "path": "/Recovery/WindowsRE/winre.wim",
          "type": "file",
          "size": 505453500,
          "mode": "0777",
          "modified_time": "2024-01-29T12:59:01.5190276Z",
          "is_symlink": false
        }
      ],
      "total_count": 2
    }
    ```
  - Response Fields:
    - mount_id: UUID for this mount session
    - path: Current directory path being listed
    - files: Array of file/directory entries
      - name: Filename or directory name
      - path: Full path (use for download or navigation)
      - type: "file" or "directory" (GUI uses for icon selection)
      - size: File size in bytes (0 for directories)
      - mode: Unix file mode (permissions)
      - modified_time: ISO 8601 timestamp
      - is_symlink: Boolean indicating symbolic link
    - total_count: Number of items in current directory
  - Security: Path traversal protection, validates all paths against mount root
  - Classification: Key (file browsing)
  - GUI Usage: 
    - Display files with type-based icons (folder/file)
    - Click folder → request files?path={item.path}
    - Click file → download?path={item.path}
    - Show file metadata (size, modified_time)
  - Handler: `api/handlers/restore_handlers.go:ListFiles`
  - Service: `restore/file_browser.go:ListFiles`

- GET /restore/{mount_id}/file-info → `handlers.Restore.GetFileInfo`
  - Description: Get detailed file metadata (size, permissions, modified time)
  - Query Params: path (required)
  - Response: FileInfo object with complete metadata
  - Classification: Auxiliary
  - Handler: `api/handlers/restore_handlers.go:GetFileInfo`
  - Service: `restore/file_browser.go:GetFileInfo`

- GET /restore/{mount_id}/download → `handlers.Restore.DownloadFile`
  - Description: Download individual file via HTTP streaming
  - Query Params: path (required) - Full file path from files API response
  - Response: File stream with appropriate Content-Type header
  - Example: `GET /restore/e4805a6f-8ee7-4f3c-8309-2f12362c7398/download?path=/Recovery/WindowsRE/ReAgent.xml`
  - Classification: Key (file recovery)
  - Handler: `api/handlers/restore_handlers.go:DownloadFile`
  - Service: `restore/file_downloader.go:DownloadFile`

- GET /restore/{mount_id}/download-directory → `handlers.Restore.DownloadDirectory`
  - Description: Download directory as ZIP or TAR.GZ archive (bulk file recovery)
  - Query Params: 
    - path (required): Directory path to download
    - format ("zip" or "tar.gz", default: "zip"): Archive format
  - Response: Archive stream with appropriate Content-Type
  - Classification: Key (bulk recovery)
  - Handler: `api/handlers/restore_handlers.go:DownloadDirectory`
  - Service: `restore/file_downloader.go:DownloadDirectory`

- GET /restore/resources → `handlers.Restore.GetResourceStatus`
  - Description: Monitor restore resource utilization (NBD devices, mount slots)
  - Response: 
    ```json
    {
      "active_mounts": 1,
      "max_mounts": 8,
      "available_slots": 7,
      "allocated_devices": ["/dev/nbd0"],
      "device_utilization": "12.5%"
    }
    ```
  - Classification: Auxiliary (monitoring)
  - Handler: `api/handlers/restore_handlers.go:GetResourceStatus`

- GET /restore/cleanup-status → `handlers.Restore.GetCleanupStatus`
  - Description: Cleanup service status and statistics
  - Response: 
    ```json
    {
      "running": true,
      "cleanup_interval": "5m",
      "idle_timeout": "1h",
      "active_mount_count": 1,
      "expired_mount_count": 0,
      "last_cleanup": "2025-10-08T21:20:00+01:00"
    }
    ```
  - Classification: Auxiliary (monitoring)
  - Handler: `api/handlers/restore_handlers.go:GetCleanupStatus`
  - Service: `restore/cleanup_service.go`

**Architecture Notes (v2.16.0+ Restore System):**
- **Handler:** `api/handlers/restore_handlers.go`
- **Services:** `restore/mount_manager.go`, `restore/file_browser.go`, `restore/file_downloader.go`, `restore/cleanup_service.go`
- **Database:** restore_mounts table with FK to backup_disks.id (CASCADE DELETE chain)
- **NBD Devices:** /dev/nbd0-7 pool dedicated to restore operations (separate from backup pool)
- **Security:** Read-only mounts, path traversal protection, automatic cleanup after 1 hour idle
- **Multi-Disk Support:** Select specific disk from multi-disk VM backup via disk_index parameter
- **Filesystem Detection:** Automatic detection of ntfs, ext4, xfs, btrfs, etc.
- **Cleanup Service:** Automatic unmount and NBD release after 1 hour idle time
- **CASCADE DELETE:** Deleting backup_disks record automatically unmounts and cleans up restore mounts
- **Customer Value:** Individual file recovery without full VM restore, competitive advantage vs Veeam
- **GUI Integration:** JSON responses optimized for file browser UI (type field, full paths, metadata)

**Database Schema (v2.16.0+):**
```sql
CREATE TABLE restore_mounts (
  id VARCHAR(64) PRIMARY KEY,
  backup_disk_id BIGINT NOT NULL,
  mount_path VARCHAR(512),
  nbd_device VARCHAR(32),
  filesystem_type VARCHAR(32),
  status ENUM('mounting', 'mounted', 'unmounting', 'failed'),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_accessed_at TIMESTAMP,
  expires_at TIMESTAMP,
  error_message TEXT,
  INDEX idx_backup_disk (backup_disk_id),
  UNIQUE KEY uk_backup_disk (backup_disk_id),
  UNIQUE KEY uk_nbd_device (nbd_device),
  CONSTRAINT fk_restore_mount_disk 
    FOREIGN KEY (backup_disk_id) REFERENCES backup_disks(id) 
    ON DELETE CASCADE
);
```

**Callsites:**
- GUI File Browser (planned): Uses mount → files → download workflow
- Customer Support: Manual file recovery via API
- Automatic Cleanup Service: Monitors last_accessed_at for expired mounts

**Example Workflow:**
```bash
# 1. Mount disk 0 from multi-disk backup
curl -X POST http://sha:8082/api/v1/restore/mount \
  -H "Content-Type: application/json" \
  -d '{"backup_id":"backup-pgtest1-1759947871","disk_index":0}'
# → Returns mount_id

# 2. Browse root directory
curl "http://sha:8082/api/v1/restore/{mount_id}/files?path=/"
# → Returns list of files/folders with types

# 3. Navigate into folder
curl "http://sha:8082/api/v1/restore/{mount_id}/files?path=/Recovery/WindowsRE"
# → Returns files in subdirectory

# 4. Download specific file
curl "http://sha:8082/api/v1/restore/{mount_id}/download?path=/Recovery/WindowsRE/ReAgent.xml" \
  -o ReAgent.xml
# → Downloads file

# 5. Unmount (or wait for automatic cleanup after 1 hour)
curl -X DELETE "http://sha:8082/api/v1/restore/{mount_id}"
# → Cleanup completed
```

**Testing Status:** ✅ PRODUCTION READY (tested 2025-10-08 with pgtest1 102GB Windows disk)

Backup API Endpoints (Multi-Disk VM-Level Backups - Implemented October 2025)
- POST /api/v1/backups → `handlers.BackupHandler.StartBackup`
  - Description: Start VM-level backup (all disks simultaneously) - prevents data corruption from multiple snapshots
  - Request: BackupStartRequest
    ```json
    {
      "vm_name": "pgtest1",
      "repository_id": "1",
      "backup_type": "full"
    }
    ```
  - ⚠️  **CRITICAL**: NO disk_id field - backups are VM-level to maintain consistency
  - Response: BackupResponse with multi-disk results
    ```json
    {
      "backup_id": "backup-pgtest1-1759901593",
      "vm_context_id": "ctx-pgtest1-20251006-203401",
      "vm_name": "pgtest1",
      "disk_results": [
        {
          "disk_id": 0,
          "nbd_port": 10104,
          "nbd_export_name": "pgtest1-disk-2000",
          "qcow2_path": "/backup/repository/pgtest1-disk-2000.qcow2",
          "qemu_nbd_pid": 3956432,
          "status": "qemu_started"
        },
        {
          "disk_id": 1,
          "nbd_port": 10105,
          "nbd_export_name": "pgtest1-disk-2001",
          "qcow2_path": "/backup/repository/pgtest1-disk-2001.qcow2",
          "qemu_nbd_pid": 3956438,
          "status": "qemu_started"
        }
      ],
      "nbd_targets_string": "2000:nbd://127.0.0.1:10104/pgtest1-disk-2000,2001:nbd://127.0.0.1:10105/pgtest1-disk-2001",
      "backup_type": "full",
      "repository_id": "1",
      "status": "started",
      "bytes_transferred": 0,
      "total_bytes": 0,
      "created_at": "2025-10-08T06:33:13+01:00"
    }
    ```
  - Classification: **Key** (backup automation)
  - Multi-Disk Architecture:
    - Discovers ALL disks for VM from vm_disks table
    - Allocates unique NBD port per disk (10100-10200 range)
    - Creates separate QCOW2 file per disk in repository
    - Starts qemu-nbd process per disk with --shared 10 flag
    - Generates VMware disk keys: 2000, 2001, 2002... (loop index + 2000)
    - Calls SNA /api/v1/backup/start with multi-disk NBD targets string
  - Handler Location: `sha/api/handlers/backup_handlers.go` (StartBackup method)
  - Database: Creates backup_jobs entry, links to vm_replication_contexts
  - Cleanup: Comprehensive defer block stops qemu-nbd, releases ports, deletes QCOW2s on failure
  - Tested: October 8, 2025 - 2-disk VM (102GB + 5GB) confirmed working at 10 MB/s

- GET /api/v1/backups → `handlers.BackupHandler.ListBackups`
  - Description: List backups with optional filtering
  - Query Params: vm_name, vm_context_id, repository_id, backup_type, status
  - Response: { backups: [BackupResponse], total: number }
  - Classification: **Key** (backup discovery)
  - BackupResponse Structure (Updated 2025-10-10):
    ```json
    {
      "backup_id": "backup-pgtest1-1760099954",
      "vm_context_id": "ctx-pgtest1-20251006-203401",
      "vm_name": "pgtest1",
      "backup_type": "incremental",
      "type": "incremental",
      "repository_id": "repo-local-1760055634",
      "status": "completed",
      "bytes_transferred": 8455192576,
      "total_bytes": 8455192576,
      "current_phase": "completed",
      "progress_percent": 100.0,
      "transfer_speed_bps": 336392246,
      "last_telemetry_at": "2025-10-10T13:40:16Z",
      "created_at": "2025-10-10T13:39:14Z",
      "started_at": "2025-10-10T13:39:15Z",
      "completed_at": "2025-10-10T13:40:16Z",
      "error_message": null
    }
    ```
  - New Fields (Added v2.27.0):
    - `type`: Alias for `backup_type` (frontend compatibility)
    - `current_phase`: Current backup phase from telemetry
    - `progress_percent`: Real-time progress from telemetry (0-100)
    - `transfer_speed_bps`: Current transfer speed from telemetry
    - `last_telemetry_at`: Timestamp of last telemetry update

- GET /api/v1/backups/{backup_id} → `handlers.BackupHandler.GetBackupDetails`
  - Description: Get detailed information about a specific backup
  - Response: BackupResponse with complete metadata and timestamps (see structure above)
  - Classification: **Key** (backup monitoring)

- POST /api/v1/backups/{backup_id}/complete → `handlers.BackupHandler.CompleteBackup`
  - Description: Mark backup as complete and record change_id (called by sendense-backup-client)
  - Request:
    ```json
    {
      "change_id": "52 66 8c 2d a7 c5 c5 68-c5 d2 8d 04 79 f5 fd 7d/5440",
      "bytes_transferred": 102000000000
    }
    ```
  - Response: { status: "completed", backup_id, change_id, message, timestamp }
  - Classification: **Key** (backup completion, incremental enablement)
  - Purpose: Stores VMware CBT change_id for next incremental backup
  - Added: October 8, 2025 (v2.23.0)

- DELETE /api/v1/backups/{backup_id} → `handlers.BackupHandler.DeleteBackup`
  - Description: Delete a backup from repository and database
  - Response: { message, backup_id }
  - Protection: CASCADE DELETE handles related records
  - Classification: **Key** (backup lifecycle)

- GET /api/v1/backups/chain → `handlers.BackupHandler.GetBackupChain`
  - Description: Get complete backup chain (full + incrementals) for entire VM
  - Query Params: vm_context_id or vm_name (required)
  - Response: { chain_id, full_backup_id, backups: [], total_size_bytes, backup_count }
  - Classification: **Key** (backup chain management)
  - Handler: `sha/api/handlers/backup_handlers.go`
  - Architecture: Integrates with BackupEngine, RepositoryManager, QemuNBDManager, NBDPortAllocator
  - Database: backup_jobs table with VM-level backup entries (no per-disk entries)
  - Dependencies: services.QemuNBDManager, services.NBDPortAllocator, database.VMDiskRepository
  - Customer Value: Crash-consistent multi-disk VM backups, prevents data corruption

- GET /api/v1/backups/changeid → `handlers.BackupHandler.GetChangeID`
  - Description: Get last successful change_id for specific VM disk (used by sendense-backup-client for incremental backups)
  - Query Params: 
    - vm_name (required): VM name
    - disk_id (required): Disk index (0, 1, 2...)
  - Response:
    ```json
    {
      "vm_name": "pgtest1",
      "disk_id": "0",
      "change_id": "52 66 8c 2d a7 c5 c5 68-c5 d2 8d 04 79 f5 fd 7d/5530",
      "backup_id": "backup-pgtest1-1759947871",
      "backup_type": "full",
      "completed_at": "2025-10-08T19:54:00Z"
    }
    ```
  - Classification: **Critical** (enables incremental backups via VMware CBT)
  - Handler: `sha/api/handlers/backup_handlers.go` (GetChangeID method, added v2.11.0)
  - Database: Queries backup_disks table with JOIN to vm_backup_contexts
  - Purpose: Provides backup client with last change_id to enable VMware CBT incremental transfers
  - Architecture Change: Separates backup change_ids from replication change_ids (different tables/workflows)
  - Testing: October 8, 2025 - Multi-disk VM confirmed working (pgtest1: disk 0 + disk 1)
  - Note: Returns 404 if no completed backup found for VM+disk combination

Backup Job Telemetry (Real-Time Progress Tracking - Implemented October 10, 2025)
- POST /api/v1/telemetry/{job_type}/{job_id} → `handlers.Telemetry.ReceiveTelemetry`
  - Description: Receive real-time telemetry updates from sendense-backup-client (replaces polling-based progress tracking)
  - Path Parameters:
    - job_type (string, required): Type of job - "backup", "replication", "restore"
    - job_id (string, required): Unique job identifier (e.g., "backup-pgtest1-disk0-20251010-143522")
  - Request Body: TelemetryUpdateRequest
    ```json
    {
      "job_type": "backup",
      "status": "running",
      "current_phase": "transferring",
      "bytes_transferred": 32212254720,
      "total_bytes": 107374182400,
      "transfer_speed_bps": 3221225472,
      "eta_seconds": 23,
      "progress_percent": 30.0,
      "disks": [
        {
          "disk_index": 0,
          "bytes_transferred": 32212254720,
          "progress_percent": 30.0,
          "status": "transferring",
          "error_message": null
        }
      ],
      "error": null,
      "timestamp": "2025-10-10T14:35:42Z"
    }
    ```
  - Request Fields:
    - status (string): Job status - "running", "completed", "failed", "stalled"
    - current_phase (string): Current operation phase - "snapshot", "transferring", "finalizing"
    - bytes_transferred (int64): Total bytes transferred across all disks
    - total_bytes (int64): Total bytes to transfer
    - transfer_speed_bps (int64): Current transfer speed in bytes per second
    - eta_seconds (int): Estimated time to completion in seconds
    - progress_percent (float64): Overall progress percentage (0-100)
    - disks (array): Per-disk progress information (multi-disk VM support)
    - error (string, optional): Error message if job failed
    - timestamp (string): ISO8601 timestamp of telemetry update
  - Response: { status: "success", message: "Telemetry received and processed", timestamp: "..." }
  - Response Codes:
    - 200 OK: Telemetry processed successfully
    - 400 Bad Request: Invalid request body or missing parameters
    - 500 Internal Server Error: Database update failed
  - Classification: **Critical** (enables real-time GUI progress tracking)
  - Architecture: Push-based telemetry replaces old polling system
  - Cadence: Hybrid - sends updates when:
    1. Time-based: Every 5 seconds during active transfer
    2. Progress-based: Every 10% progress milestone
    3. State changes: Phase transitions, errors, completion
    4. Mandatory: Job start and completion always send
  - Handler: `sha/api/handlers/telemetry_handlers.go` (ReceiveTelemetry method)
  - Service: `sha/services/telemetry_service.go` (ProcessTelemetryUpdate)
  - Database Updates:
    - backup_jobs: bytes_transferred, current_phase, transfer_speed_bps, eta_seconds, progress_percent, last_telemetry_at
    - backup_disks: bytes_transferred, progress_percent, status (per-disk tracking)
  - Stale Job Detection: Background service marks jobs "stalled" (60s) or "failed" (5min) if no telemetry received
  - Client: sendense-backup-client sends via http://localhost:8082 (tunnel endpoint)
  - Callsites: sendense-backup-client internal/telemetry/client.go
  - Testing: October 10, 2025 - Implementation complete, integration testing pending
  - Migration: Requires database schema changes (see 20251010_telemetry_fields.sql)
  - Benefits:
    - Real-time progress updates (no polling delay)
    - Accurate bytes_transferred reporting (fixes machine modal display)
    - Rich telemetry data for GUI charts (speed, ETA, per-disk progress)
    - Automatic stale job detection and cleanup
    - Extensible to all job types (backup, replication, restore)

NBD Port Management (Task 7 - Planned for Implementation 2025-10-07)
- POST /api/v1/nbd/ports/allocate → `handlers.NBD.AllocatePort`
  - Description: Allocate NBD port from pool (10100-10200 range) for backup/replication job
  - Request: { job_id: string, export_name: string, disk_count: number }
  - Response: { job_id: string, allocated_ports: [{ port: number, export_name: string }], expires_at: timestamp }
  - Classification: **Key** (resource management)
  - Purpose: Dynamic port allocation for multi-disk backup jobs via SSH tunnel
  - Root Cause Fix: Discovered October 7, 2025 - qemu-nbd defaults to --shared=1 (single connection limit)
  - Solution: Pre-forward ports 10100-10200 through SSH tunnel, allocate dynamically per job
- POST /api/v1/nbd/ports/release → `handlers.NBD.ReleasePort`
  - Description: Release NBD port(s) after job completion
  - Request: { job_id: string, ports: [number] }
  - Response: { success: true, released_count: number }
  - Classification: **Key** (cleanup)
- GET /api/v1/nbd/ports/status → `handlers.NBD.GetPortStatus`
  - Description: Query allocated ports and availability
  - Response: { total_ports: 101, allocated: number, available: number, allocations: [{ port, job_id, export_name, allocated_at, expires_at }] }
  - Classification: **Auxiliary** (monitoring)
- POST /api/v1/nbd/qemu-nbd/start → `handlers.NBD.StartQemuNBD`
  - Description: Start qemu-nbd process for QCOW2 file export
  - Request: { export_name: string, port: number, qcow2_path: string, read_only: boolean, shared_connections: number }
  - Response: { pid: number, port: number, export_name: string, status: "running" }
  - Classification: **Key** (qemu-nbd process management)
  - Critical: Always use --shared=5 or higher (migratekit opens 2 connections per export)
- POST /api/v1/nbd/qemu-nbd/stop → `handlers.NBD.StopQemuNBD`
  - Description: Stop qemu-nbd process by PID or export name
  - Request: { pid: number } OR { export_name: string }
  - Response: { success: true, stopped_pid: number }
  - Classification: **Key** (cleanup)
- GET /api/v1/nbd/qemu-nbd/list → `handlers.NBD.ListQemuNBDProcesses`
  - Description: List all running qemu-nbd processes
  - Response: { processes: [{ pid, port, export_name, qcow2_path, uptime_seconds, connection_count, shared_limit }] }
  - Classification: **Auxiliary** (monitoring)
  - Handler: `handlers.NBD.*` (to be implemented)
  - Database: nbd_port_allocations, qemu_nbd_processes tables (to be created)
  - Architecture: Port pool manager (10100-10200), qemu-nbd process lifecycle management, SSH multi-port tunnel integration
  - Investigation: 10+ hours, 12+ tests, job sheet `2025-10-07-qemu-nbd-tunnel-investigation.md`
  - Solution Verified: October 7, 2025 - SSH tunnel + direct TCP both work with --shared flag configured

Protection Flows Engine (v2.25.2+ - October 9, 2025)
- POST /api/v1/protection-flows → `handlers.ProtectionFlow.CreateFlow`
  - Description: Create new backup or replication flow for VM or group
  - Request: { name, description?, flow_type: "backup"|"replication", target_type: "vm"|"group", target_id, repository_id, schedule_id?, policy_id?, enabled: boolean }
  - Response: ProtectionFlow object with auto-generated ID and status fields
  - Classification: **Key** (flow orchestration)
  - Handler: `sha/api/handlers/protection_flow_handlers.go`
  - Database: Writes protection_flows table
  - Purpose: Create scheduled backup/replication flows for automated protection

- GET /api/v1/protection-flows → `handlers.ProtectionFlow.ListFlows`
  - Description: List all protection flows with status information
  - Query Params: flow_type?, target_type?, enabled?
  - Response: { flows: [ProtectionFlow], count }
  - Classification: **Key** (GUI flow management)

- GET /api/v1/protection-flows/summary → `handlers.ProtectionFlow.GetFlowsSummary`
  - Description: Get summary statistics for all flows
  - Response: { total_flows, active_flows, total_executions, successful_executions, failed_executions }
  - Classification: **Key** (dashboard statistics)

- GET /api/v1/protection-flows/{id} → `handlers.ProtectionFlow.GetFlow`
  - Description: Get single flow with full status
  - Response: ProtectionFlow with nested status object
  - Classification: **Key** (flow details)

- PUT /api/v1/protection-flows/{id} → `handlers.ProtectionFlow.UpdateFlow`
  - Description: Update flow configuration (name, schedule, enabled state)
  - Request: Partial ProtectionFlow fields
  - Response: Updated ProtectionFlow
  - Classification: **Key** (flow management)

- DELETE /api/v1/protection-flows/{id} → `handlers.ProtectionFlow.DeleteFlow`
  - Description: Delete protection flow (CASCADE deletes executions)
  - Response: { message, deleted_flow_id }
  - Classification: **Key** (flow management)

- POST /api/v1/protection-flows/{id}/execute → `handlers.ProtectionFlow.ExecuteFlow`
  - Description: Manual execution of protection flow (backup or replication)
  - Response: ProtectionFlowExecution with created job IDs
  - Classification: **Critical** (manual "Run Now" trigger)
  - Purpose: Executes backup for target VM/group, auto-detects full vs incremental
  - Database: Creates protection_flow_executions record, triggers backup via existing backup API
  - Job Tracking: Uses job_type="scheduler" for compatibility

- POST /api/v1/protection-flows/bulk-execute → `handlers.ProtectionFlow.BulkExecuteFlows`
  - Description: Execute multiple flows simultaneously
  - Request: { flow_ids: [string] }
  - Response: { results: [ExecutionResult], total, succeeded, failed }
  - Classification: **Key** (bulk operations)

- GET /api/v1/protection-flows/{id}/executions → `handlers.ProtectionFlow.ListExecutions`
  - Description: Get execution history for flow
  - Query Params: limit?, offset?
  - Response: { executions: [ProtectionFlowExecution], count }
  - Classification: **Key** (execution history)

- GET /api/v1/protection-flows/{id}/status → `handlers.ProtectionFlow.GetFlowStatus`
  - Description: Get real-time flow status with last execution details
  - Response: FlowStatus with execution statistics
  - Classification: **Key** (status monitoring)

- POST /api/v1/protection-flows/bulk-enable → `handlers.ProtectionFlow.BulkEnableFlows`
  - Description: Enable multiple flows
  - Request: { flow_ids: [string] }
  - Response: { updated_count }
  - Classification: **Auxiliary** (bulk management)

- POST /api/v1/protection-flows/bulk-disable → `handlers.ProtectionFlow.BulkDisableFlows`
  - Description: Disable multiple flows
  - Request: { flow_ids: [string] }
  - Response: { updated_count }
  - Classification: **Auxiliary** (bulk management)

- POST /api/v1/protection-flows/bulk-delete → `handlers.ProtectionFlow.BulkDeleteFlows`
  - Description: Delete multiple flows (CASCADE)
  - Request: { flow_ids: [string] }
  - Response: { deleted_count }
  - Classification: **Auxiliary** (bulk management)

- Architecture Notes:
  - Protection Flows Engine integrates with existing scheduler service (SchedulerService.RegisterFlowSchedule)
  - Backup flows call existing POST /api/v1/backups endpoint with intelligent full/incremental detection
  - Replication flows planned for Phase 5 (currently returns "not implemented" error)
  - Database CASCADE DELETE: Deleting flow auto-removes all execution records
  - Job tracking uses "scheduler" job_type for compatibility with existing job_tracking ENUM
  - First execution per VM/repository always performs full backup, subsequent executions are incremental
  - Multi-disk VMs: backup API handles all disks automatically, GUI shows aggregated status

- Testing Status: October 9, 2025
  - ✅ Flow creation working (GUI + API)
  - ✅ Manual execution working (backup flows)
  - ✅ Incremental backup detection working (checks backup_jobs table for existing full backup)
  - ✅ Multi-disk VM support working (pgtest1: disk 0 + disk 1)
  - ✅ Flow status tracking working (nested status object)
  - ✅ GUI wiring complete (Protection Flows page functional)

- Database Tables:
  - protection_flows: Flow definitions with configuration
  - protection_flow_executions: Execution history with CASCADE DELETE FK
  - Foreign Keys: schedule_id → replication_schedules.id (deferred due to collation mismatch)

- Implementation: October 9, 2025
  - Backend: Grok (Tasks 1-5 complete)
  - GUI: Grok (wiring + theme fixes complete)
  - Job Sheet: `job-sheets/2025-10-09-protection-flows-engine.md`
  - Binary: sendense-hub-v2.25.2-backup-type-fix
  - Commits: Multiple (schema, service, API, GUI integration, bug fixes)

Legacy/Potentially Legacy Notes
- Original failover handlers exist alongside enhanced; enhanced/unified are primary. The `RegisterFailoverRoutes` exports classic paths; prefer enhanced/unified.
- `vma_simple.go` defines simple enrollment handlers but is not wired in `handlers.NewHandlers`; Classification: Legacy (unused).

