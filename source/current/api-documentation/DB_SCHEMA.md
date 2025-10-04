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

- backup_jobs
  - PK: id (varchar 64)
  - FKs: vm_context_id → vm_replication_contexts(context_id) CASCADE; repository_id → backup_repositories(id) RESTRICT; policy_id → backup_policies(id) SET NULL; parent_backup_id → backup_jobs(id) SET NULL
  - Fields: vm_name, backup_type ENUM('full','incremental','differential'), status ENUM('pending','running','completed','failed','cancelled'), repository_path, change_id, bytes_transferred, total_bytes, compression_enabled, error_message
  - Timestamps: created_at, started_at, completed_at
  - Indexes: idx_vm_context, idx_repository, idx_policy, idx_status, idx_created, idx_parent

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
  - FK: vm_context_id → vm_replication_contexts(context_id) CASCADE
  - Fields: disk_id, full_backup_id, latest_backup_id, total_backups INT, total_size_bytes BIGINT
  - Index: idx_vm_context


