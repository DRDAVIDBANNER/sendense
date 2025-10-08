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

File-Level Restore (Task 4 - Implemented 2025-10-05)
- POST /restore/mount → `handlers.Restore.MountBackup`
  - Description: Mount QCOW2 backup for file browsing via qemu-nbd
  - Request: { backup_id: string }
  - Response: { mount_id: string, mount_path: string, nbd_device: string, filesystem_type: string, status: string, expires_at: timestamp }
  - Classification: Key (customer file recovery)
  - Security: Read-only mounts, automatic cleanup after 1 hour idle
- DELETE /restore/{mount_id} → `handlers.Restore.UnmountBackup`
  - Description: Unmount backup and release NBD device
  - Response: { message: "backup unmounted successfully" }
  - Classification: Key
- GET /restore/mounts → `handlers.Restore.ListMounts`
  - Description: List all active restore mounts
  - Response: { mounts: [], count: number }
  - Classification: Key
- GET /restore/{mount_id}/files → `handlers.Restore.ListFiles`
  - Description: Browse files and directories within mounted backup
  - Query Params: path (default: "/"), recursive (boolean)
  - Response: { files: [], total_count: number }
  - Security: Path traversal protection, validates all paths against mount root
  - Classification: Key (file browsing)
- GET /restore/{mount_id}/file-info → `handlers.Restore.GetFileInfo`
  - Description: Get detailed file metadata (size, permissions, modified time)
  - Query Params: path (required)
  - Response: FileInfo object with complete metadata
  - Classification: Auxiliary
- GET /restore/{mount_id}/download → `handlers.Restore.DownloadFile`
  - Description: Download individual file via HTTP streaming
  - Query Params: path (required)
  - Response: File stream with appropriate Content-Type
  - Classification: Key (file recovery)
- GET /restore/{mount_id}/download-directory → `handlers.Restore.DownloadDirectory`
  - Description: Download directory as ZIP or TAR.GZ archive
  - Query Params: path (required), format ("zip" or "tar.gz", default: "zip")
  - Response: Archive stream
  - Classification: Key (bulk recovery)
- GET /restore/resources → `handlers.Restore.GetResourceStatus`
  - Description: Monitor restore resource utilization (NBD devices, mount slots)
  - Response: { active_mounts, max_mounts, available_slots, allocated_devices, device_utilization }
  - Classification: Auxiliary (monitoring)
- GET /restore/cleanup-status → `handlers.Restore.GetCleanupStatus`
  - Description: Cleanup service status and statistics
  - Response: { running, cleanup_interval, idle_timeout, active_mount_count, expired_mount_count }
  - Classification: Auxiliary (monitoring)
  - Handler: `handlers.Restore.*`
  - Architecture: qemu-nbd on /dev/nbd0-7, automatic cleanup service, path traversal protection
  - Database: restore_mounts table with mount tracking
  - Customer Value: Individual file recovery without full VM restore

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

- GET /api/v1/backups/{backup_id} → `handlers.BackupHandler.GetBackupDetails`
  - Description: Get detailed information about a specific backup
  - Response: BackupResponse with complete metadata and timestamps
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

Legacy/Potentially Legacy Notes
- Original failover handlers exist alongside enhanced; enhanced/unified are primary. The `RegisterFailoverRoutes` exports classic paths; prefer enhanced/unified.
- `vma_simple.go` defines simple enrollment handlers but is not wired in `handlers.NewHandlers`; Classification: Legacy (unused).

