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
- GET /vm-contexts/{vm_name} → `handlers.VMContext.GetVMContext`
- GET /vm-contexts/{context_id}/recent-jobs → `handlers.VMContext.GetRecentJobs`
  - Classification: Key (GUI relies on these)

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

Backup API Endpoints (Task 5 - Implemented 2025-10-05)
- POST /api/v1/backup/start → `handlers.Backup.StartBackup`
  - Description: Start a full or incremental backup of a VM disk
  - Request: { vm_name, disk_id, backup_type: "full"|"incremental", repository_id, policy_id?, tags? }
  - Response: BackupResponse with backup_id, status, file_path, progress
  - Classification: **Key** (backup automation)
  - Integration: Calls BackupEngine workflow, creates NBD export, triggers VMA replication

- GET /api/v1/backup/list → `handlers.Backup.ListBackups`
  - Description: List backups with optional filtering
  - Query Params: vm_name, vm_context_id, repository_id, backup_type, status
  - Response: { backups: [BackupResponse], total: number }
  - Classification: **Key** (backup discovery)

- GET /api/v1/backup/{backup_id} → `handlers.Backup.GetBackupDetails`
  - Description: Get detailed information about a specific backup
  - Response: BackupResponse with complete metadata and timestamps
  - Classification: **Key** (backup monitoring)

- DELETE /api/v1/backup/{backup_id} → `handlers.Backup.DeleteBackup`
  - Description: Delete a backup from repository and database
  - Response: { message, backup_id }
  - Protection: CASCADE DELETE handles related records
  - Classification: **Key** (backup lifecycle)

- GET /api/v1/backup/chain → `handlers.Backup.GetBackupChain`
  - Description: Get complete backup chain (full + incrementals) for a VM disk
  - Query Params: vm_context_id or vm_name (required), disk_id (default: 0)
  - Response: { chain_id, full_backup_id, backups: [], total_size_bytes, backup_count }
  - Classification: **Key** (backup chain management)
  - Handler: `handlers.Backup.*`
  - Architecture: Integrates with BackupEngine (Task 3), Repository Manager (Task 1), NBD Export (Task 2)
  - Database: backup_jobs table with backup chain relationships
  - Customer Value: API-driven backup automation, GUI integration, scheduled backups

Legacy/Potentially Legacy Notes
- Original failover handlers exist alongside enhanced; enhanced/unified are primary. The `RegisterFailoverRoutes` exports classic paths; prefer enhanced/unified.
- `vma_simple.go` defines simple enrollment handlers but is not wired in `handlers.NewHandlers`; Classification: Legacy (unused).

