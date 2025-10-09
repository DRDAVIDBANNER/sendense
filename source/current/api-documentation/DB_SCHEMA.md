Database Schema (migratekit_oma)

Source: `/home/oma_admin/sendense/oma-deployment-package/database/production-schema.sql`

Core VM-Centric Tables
- vm_replication_contexts
  - PK: context_id (varchar 64)
  - Unique: (vmware_vm_id, vcenter_host), (vm_name, vcenter_host)
  - Status tracking fields: current_status ENUM, current_job_id, last_successful_job_id, job counters, timestamps
  - Scheduling: scheduler_enabled, next_scheduled_at, last_scheduled_job_id
  - Credentials and configs: credential_id → vmware_credentials(id), ossea_config_id (FK absent in dump; referenced by other tables)
  - FKs:
    - current_job_id → replication_jobs(id) ON DELETE SET NULL
    - last_successful_job_id → replication_jobs(id) ON DELETE SET NULL

- replication_jobs
  - PK: id (varchar 191)
  - Context: vm_context_id → vm_replication_contexts(context_id) ON DELETE CASCADE
  - Source details: source_vm_id/name/path, v_center_host, datacenter
  - Progress: status, progress_percent, current_operation, bytes/total, transfer_speed_bps, error_message
  - CBT fields: change_id, previous_change_id
  - VMA integration: vma_* fields for phases/throughput/eta/last_poll/error
  - Scheduling: schedule_execution_id, scheduled_by, vm_group_id
  - FKs: ossea_config_id → ossea_configs(id); schedule_execution_id → schedule_executions(id); vm_group_id → vm_machine_groups(id)

- vm_disks
  - PK: id (auto)
  - Unique: (vm_context_id, disk_id)
  - FKs: vm_context_id → vm_replication_contexts(context_id) CASCADE; job_id → replication_jobs(id) CASCADE
  - Fields: disk_id, vm_dk_path, size_gb, datastore, unit_number, label, capacity_bytes, provisioning_type, ossea_volume_id, disk_change_id, per-disk progress

Failover and Networking
- failover_jobs
  - PK: id (auto)
  - Unique: job_id
  - Context: vm_context_id → vm_replication_contexts(context_id) CASCADE
  - Fields: job_type (live/test), status (enum lifecycle), destination_vm_id, linstor_snapshot_name, ossea_snapshot_id, linstror_config_id, network_mappings

- network_mappings
  - PK: id (auto)
  - Unique: (vm_id, source_network_name, is_test_network), (vm_context_id, source_network_name, is_test_network)
  - Context: vm_context_id → vm_replication_contexts(context_id) CASCADE
  - Fields: mapping_type, validation_status, strategy, last_validated

Storage and NBD
- ossea_configs
  - PK: id (auto). Unique name

- ossea_volumes
  - PK: id (auto)
  - Context: vm_context_id → vm_replication_contexts(context_id) CASCADE
  - Snapshot tracking: snapshot_id/status/created_at

- device_mappings
  - PK: id (uuid default)
  - Unique: volume_uuid, device_path
  - Context: vm_context_id → vm_replication_contexts(context_id) CASCADE
  - Fields: operation_mode ENUM('oma','failover'), vm_id, cloudstack/device/linux states, persistent_device_name, symlink_path, snapshot tracking

- nbd_exports
  - PK: id (auto)
  - Unique: export_name
  - FKs: device_mapping_uuid → device_mappings(volume_uuid) CASCADE; vm_disk_id → vm_disks(id) CASCADE; job_id → replication_jobs(id) CASCADE; vm_context_id → vm_replication_contexts(context_id) CASCADE

CBT
- cbt_history
  - PK: id (auto)
  - FKs: vm_context_id → vm_replication_contexts(context_id) CASCADE; job_id → replication_jobs(id) CASCADE
  - Fields: disk_id, change_id, previous_change_id, sync_type, blocks_changed, bytes_transferred, sync_success

Scheduling and Groups
- replication_schedules (self-FK chain)
- schedule_executions (FKs to replication_schedules, vm_machine_groups)
- vm_machine_groups (FK schedule_id → replication_schedules)
- vm_group_memberships (FKs to vm_machine_groups, vm_replication_contexts, replication_schedules for overrides)
- Views: active_schedules, schedule_execution_summary, vm_schedule_status

JobLog Infrastructure
- job_tracking (self-FK parent_job_id)
- job_steps (FK job_id → job_tracking)
- job_execution_log (FK job_id → job_tracking)
- log_events (FK job_id → job_tracking, step_id → job_steps)
- Views: active_jobs, job_tracking_hierarchy

VMA Enrollment and Connectivity
- vmware_credentials (centralized credentials)
- vma_enrollments (pairing workflow), vma_pairing_codes (code issuance), vma_active_connections (current tunnel), vma_connection_audit (events)

Volume Daemon Telemetry
- volume_operations, volume_operation_history, volume_mounts, volume_daemon_metrics

Foreign Key Summary
- Master context table: `vm_replication_contexts` with CASCADE DELETE propagating to: replication_jobs, vm_disks, ossea_volumes, device_mappings, nbd_exports, network_mappings, and linked views.

Views
- active_jobs, job_tracking_hierarchy, active_schedules, schedule_execution_summary, vm_schedule_status as defined at end of schema.

Backup Repository System (Phase 1 - Added 2025-10-04)
- backup_repositories
  - PK: id (varchar 64)
  - Unique: name
  - Fields: repository_type ENUM('local','nfs','cifs','smb','s3','azure'), enabled BOOLEAN, config JSON, is_immutable BOOLEAN, immutable_config JSON, min_retention_days INT
  - Storage tracking: total_size_bytes, used_size_bytes, available_size_bytes, last_check_at
  - Indexes: idx_type, idx_enabled, idx_immutable

- backup_policies
  - PK: id (varchar 64)
  - Unique: name
  - FK: primary_repository_id → backup_repositories(id)
  - Fields: enabled BOOLEAN, retention_days INT
  - Index: idx_enabled

- backup_copy_rules
  - PK: id (varchar 64)
  - FKs: policy_id → backup_policies(id) CASCADE; destination_repository_id → backup_repositories(id)
  - Fields: copy_mode ENUM('immediate','scheduled','manual'), priority INT, enabled BOOLEAN, verify_after_copy BOOLEAN
  - Indexes: idx_policy, idx_priority

- vm_backup_contexts (Added 2025-10-08 v2.16.0 - VM-Centric Backup Architecture)
  - PK: context_id (varchar 64)
  - Unique: (vm_name, repository_id)
  - FK: repository_id → backup_repositories(id) RESTRICT
  - Fields: vm_name, vmware_vm_id, vm_path, vcenter_host, datacenter
  - Counters: total_backups_run, successful_backups, failed_backups
  - Tracking: last_backup_id, last_backup_type ENUM('full','incremental'), last_backup_at
  - Timestamps: created_at, updated_at
  - Indexes: idx_vm_name, idx_last_backup
  - Purpose: Master context for backup VMs (eliminates fragile timestamp-window matching)

- backup_disks (Added 2025-10-08 v2.16.0 - Per-Disk Tracking)
  - PK: id (bigint auto)
  - Unique: (backup_job_id, disk_index)
  - FKs: vm_backup_context_id → vm_backup_contexts(context_id) CASCADE; backup_job_id → backup_jobs(id) CASCADE
  - Fields: disk_index (0,1,2...), vmware_disk_key (2000,2001...), size_gb, unit_number
  - CBT: disk_change_id (varchar 255) - VMware CBT change ID for incremental backups
  - Storage: qcow2_path (varchar 512), bytes_transferred
  - Status: status ENUM('pending','running','completed','failed'), error_message, completed_at
  - Timestamps: created_at
  - Indexes: idx_change_id_lookup (vm_backup_context_id, disk_index, status), idx_completion (backup_job_id, status)
  - Purpose: Per-disk backup tracking with individual change_ids (replaced time-window hack)

- backup_jobs
  - PK: id (varchar 64)
  - FKs: vm_context_id → vm_replication_contexts(context_id) CASCADE; vm_backup_context_id → vm_backup_contexts(context_id) CASCADE; repository_id → backup_repositories(id) RESTRICT; policy_id → backup_policies(id) SET NULL; parent_backup_id → backup_jobs(id) SET NULL
  - Fields: vm_name, backup_type ENUM('full','incremental','differential'), status ENUM('pending','running','completed','failed','cancelled'), repository_path, bytes_transferred, total_bytes, compression_enabled, error_message
  - Deprecated (v2.16.0+): disk_id, change_id - now stored in backup_disks table per-disk
  - Timestamps: created_at, started_at, completed_at
  - Indexes: idx_vm_context, idx_repository, idx_policy, idx_status, idx_created, idx_parent
  - Note: Parent job represents entire multi-disk backup; per-disk details in backup_disks table

- backup_copies
  - PK: id (varchar 64)
  - Unique: (source_backup_id, repository_id)
  - FKs: source_backup_id → backup_jobs(id) CASCADE; repository_id → backup_repositories(id); copy_rule_id → backup_copy_rules(id) SET NULL
  - Fields: status ENUM('pending','copying','verifying','completed','failed'), file_path, size_bytes, verification_status ENUM('pending','passed','failed'), error_message
  - Timestamps: copy_started_at, copy_completed_at, verified_at
  - Indexes: idx_source_backup, idx_repository, idx_status

- backup_chains
  - PK: id (varchar 64)
  - Unique: (vm_context_id, disk_id)
  - FK: vm_context_id → vm_backup_contexts(context_id) CASCADE (Changed 2025-10-08 - was vm_replication_contexts)
  - Fields: disk_id, full_backup_id, latest_backup_id, total_backups INT, total_size_bytes BIGINT
  - Index: idx_vm_context
  - Purpose: Tracks backup chain metadata (full + incrementals) per disk

File-Level Restore System (Task 4 - Added 2025-10-05, Updated 2025-10-09 v2.16.0+)
- restore_mounts
  - PK: id (varchar 64) - Mount UUID
  - FK: backup_disk_id → backup_disks(id) CASCADE (v2.16.0+: Changed from backup_id to support per-disk mounting)
  - Fields: mount_path (varchar 512), nbd_device (varchar 32), filesystem_type (varchar 32), mount_mode ENUM('read-only','read-write'), status ENUM('mounting','mounted','unmounting','failed','unmounted'), error_message TEXT, created_at, last_accessed_at, expires_at, unmounted_at
  - Indexes: idx_backup_disk, idx_status, idx_nbd_device, idx_expires, idx_last_accessed
  - Unique Constraints: uk_nbd_device (nbd_device), uk_backup_disk (backup_disk_id)
  - CASCADE DELETE Chain: vm_backup_contexts → backup_jobs → backup_disks → restore_mounts
  - Purpose: Track active QCOW2 backup mounts for file-level browsing and recovery (v2.16.0+ supports multi-disk VMs)

Protection Flows Engine (Phase 1 - Added 2025-10-09 v2.25.2)
- protection_flows
  - PK: id (varchar 64) - UUID flow ID
  - Fields: name (varchar 255), description TEXT, flow_type ENUM('backup','replication'), target_type ENUM('vm','group'), target_id (varchar 255), enabled BOOLEAN DEFAULT true
  - Configuration: repository_id → backup_repositories(id) (for backup flows), schedule_id → replication_schedules(id) (for scheduled execution), policy_id → backup_policies(id) (for retention policies)
  - Destination: destination_type ENUM('local','cluster','cloud') (for replication flows), destination_id (varchar 255) (for replication flows), destination_host (varchar 255) (for replication flows)
  - Status Tracking: last_execution_id (varchar 64), last_execution_status ENUM('pending','running','success','warning','error','cancelled'), last_execution_time DATETIME
  - Statistics: total_executions INT DEFAULT 0, successful_executions INT DEFAULT 0, failed_executions INT DEFAULT 0
  - Timestamps: created_at, updated_at
  - Indexes: idx_flow_type, idx_target, idx_enabled, idx_last_execution_status
  - Foreign Keys (deferred): schedule_id → replication_schedules(id) (blocked by collation mismatch utf8mb4_unicode_ci vs utf8mb4_general_ci)
  - Purpose: Define scheduled or manual backup/replication flows for VMs or groups with intelligent full/incremental detection

- protection_flow_executions
  - PK: id (varchar 64) - UUID execution ID
  - FK: flow_id → protection_flows(id) ON DELETE CASCADE
  - Fields: status ENUM('pending','running','success','warning','error','cancelled'), execution_type ENUM('manual','scheduled')
  - Job Tracking: created_job_ids TEXT (JSON array of backup_jobs.id or replication_jobs.id), jobs_created INT, jobs_completed INT, jobs_failed INT, jobs_skipped INT
  - VM Processing: vms_processed INT, bytes_transferred BIGINT
  - Timing: started_at DATETIME, completed_at DATETIME, execution_time_seconds INT
  - Audit: triggered_by (varchar 255), error_message TEXT
  - Timestamps: created_at, updated_at
  - Indexes: idx_flow_id, idx_status, idx_execution_type, idx_started_at
  - CASCADE DELETE: Deleting protection_flow auto-removes all execution records
  - Purpose: Track execution history for each protection flow with detailed job and VM processing statistics


