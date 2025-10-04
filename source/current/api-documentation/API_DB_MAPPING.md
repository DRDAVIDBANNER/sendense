API ↔ Database Mapping

OMA Endpoints

Auth
- POST /auth/login → issues tokens only; no schema writes (may read users if implemented externally)

VM Contexts
- GET /vm-contexts → reads `vm_replication_contexts` (list, filter)
- GET /vm-contexts/{vm_name} → reads `vm_replication_contexts` (by vm_name/vcenter)
- GET /vm-contexts/{context_id}/recent-jobs → reads `replication_jobs`, `job_tracking`, `job_steps`

Replications
- POST /replications → creates `replication_jobs` (status=pending), creates `vm_disks` child rows, sets `vm_replication_contexts.current_job_id`; may affect `nbd_exports` later via workflows
- GET/PUT/DELETE /replications/{id} → reads/updates/removes `replication_jobs`; delete cascades to `vm_disks`, `nbd_exports`
- GET /replications/changeid → reads `vm_disks` (latest per VM or specific disk) for `disk_change_id`
- POST /replications/{job_id}/changeid → updates `vm_disks.disk_change_id`; inserts `cbt_history` row
- GET /replications/{job_id}/progress → reads `replication_jobs` vma_* fields (plus timing) consolidated with recent polls

Progress Proxy
- GET /progress/{job_id} → proxies VMA; no direct DB writes; used with `replication_jobs` vma_* updates via poller elsewhere

Discovery
- POST /discovery/discover-vms → writes to `vm_replication_contexts` when create_context true; may set `auto_added=1`, scheduler flags
- POST /discovery/add-vms|bulk-add → writes `vm_replication_contexts` rows without jobs (no `replication_jobs`)
- GET /discovery/ungrouped-vms → reads `vm_replication_contexts` where not in `vm_group_memberships`
- POST /discovery/preview → reads vCenter via VMA; read-only on DB

VMware Credentials
- CRUD /vmware-credentials → reads/writes `vmware_credentials`; may update `vm_replication_contexts.credential_id`
- GET /vmware-credentials/default → reads `vmware_credentials` with `is_default=1`

CloudStack Settings
- POST /settings/cloudstack/* → reads/writes `ossea_configs` (test/validate/discover resources); `ossea_configs.is_active` toggles
- Detect OMA VM may record `ossea_configs.oma_vm_id`

OSSEA Config
- POST /ossea/config → writes `ossea_configs`
- POST /ossea/discover-resources → reads OSSEA; may update `ossea_configs`
- POST /ossea/config-streamlined → writes `ossea_configs`

Network Mapping
- POST /network-mappings → inserts `network_mappings` (unique constraints per VM/context)
- GET /network-mappings* → reads `network_mappings`
- DELETE /network-mappings/{vm_id}/{source_network_name} → deletes row; uniqueness ensures isolation
- GET /service-offerings/available → reads OSSEA; no DB write

Failover
- POST /failover/live|test → inserts `failover_jobs` (status lifecycle), updates `vm_replication_contexts.current_status`, may create `ossea_volumes` and `device_mappings` via Volume Daemon; may delete/reattach volumes; records `linstor_snapshot_name` and `ossea_snapshot_id`
- DELETE /failover/test/{job_id} → updates `failover_jobs` and triggers rollback: updates `device_mappings.operation_mode`, cleans `ossea_volumes` snapshots, reattaches volumes to OMA; updates `vm_replication_contexts.current_status`
- POST /failover/cleanup/{vm_name} → same DB entities as above (rollback path)
- GET /failover/{job_id}/status → reads `failover_jobs`
- GET /failover/{vm_id}/readiness → reads `vm_replication_contexts`, `replication_jobs`, `vm_disks`, `network_mappings`, `device_mappings`
- GET /failover/jobs → reads `failover_jobs`
- Unified/preflight/rollback endpoints → read/write `failover_jobs`, `device_mappings`, `ossea_volumes`, `replication_jobs`, and `vm_replication_contexts`

Scheduler & Groups
- Schedules CRUD → reads/writes `replication_schedules`
- Executions → writes `schedule_executions`; reads summary views
- Machine groups CRUD → `vm_machine_groups`
- VM group assignment → `vm_group_memberships` (insert/delete/update)

Validation
- VM validation endpoints read: `vm_replication_contexts`, `replication_jobs`, `vm_disks`, `network_mappings`, `device_mappings`, `nbd_exports`

Debug
- Debug endpoints read from various tables and views; no writes

VMA Enrollment (OMA side)
- Admin endpoints → write/read: `vma_pairing_codes`, `vma_enrollments`, `vma_active_connections`, `vma_connection_audit`
- Public endpoints → `vma_enrollments` transitions, pairing code consumption

VMA Endpoints
- GET /progress/{jobId} → reads VMA in-memory progress; indirectly OMA poller writes `replication_jobs` vma_* fields
- POST /progress/{jobId}/update → no DB (VMA memory); OMA persists via poller
- POST /discover → no OMA DB; OMA writes contexts when using results
- POST /replicate → legacy; when used, OMA still creates `replication_jobs`
- Power endpoints → no OMA DB; used by failover engines which update `failover_jobs`
- CBT status → OMA reads result to manage `vm_disks.disk_change_id` and `cbt_history`
- Enrollment → OMA DB tables above

JobLog Correlation
- All business operations use `job_tracking`, `job_steps`, `job_execution_log`, `log_events` with `context_id` when applicable, surfacing to GUI via views.


