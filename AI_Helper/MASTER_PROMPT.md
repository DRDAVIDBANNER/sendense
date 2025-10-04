# MigrateKit OSSEA - Master AI Prompt

**Copy this entire prompt for any AI assistant working on this project**

---

## PROJECT CONTEXT

You are working on MigrateKit OSSEA, a VMware → OSSEA migration platform with complete replication, failover, and cleanup workflows.

Your job: write and review code under strict architectural rules. You must respect repository structure, schema, logging, and volume operation guardrails. Never guess. Always stop and ask if unsure.

## 📂 SOURCE AUTHORITY

**Canonical source**: `source/current/**`

**Archives**: `source/archive/**` (read-only snapshots)

**Forbidden**: `oma-api-*`, `vma-api-server-*`, top-level versioned dirs, binaries in source trees

**VERSION**: Use `source/current/VERSION.txt` for active version

All build outputs go to `source/builds/` or `dist/`. Never commit binaries/logs.

## ⚙️ CORE COMPONENTS

### **VMA (VMware-side agent)**
- Discovery, CBT, replication orchestration
- Entrypoint: `source/current/vma-api-server/main.go`
- Libs: `source/current/vma/**`

### **OMA (OSSEA-side API)**
- Orchestration, failover, cleanup, job tracking
- Entrypoint: `cmd/oma/main.go`
- Logic: `internal/oma/**` (may consolidate into `source/current/oma/`)

### **Volume Daemon**
- Single source of truth for all OSSEA volume operations
- Entrypoint: `cmd/volume-daemon/main.go`
- Logic: `internal/volume/**`
- Shared client: `internal/common/volume_client.go`

### **TLS Tunnel**
- All traffic over port 443
- NBD single-port (10809), always tunneled
- No direct raw NBD connections

## 🛡️ NON-NEGOTIABLE RULES

### **1. Volume Operations**
- ✅ Always via `internal/common/volume_client.go`
- ❌ No direct SDK calls (`osseaClient.AttachVolume()`, etc.)

### **2. Logging / Job Tracking**
- ✅ All business logic logs via `internal/joblog` (`StartJob` → `RunStep` → `EndJob`)
- ❌ No `fmt.Printf`, `logrus`, or ad-hoc loggers in operation logic

### **3. Networking**
- ✅ All traffic tunneled over port 443
- ✅ NBD port 10809 inside tunnel
- ❌ Never expose raw NBD

### **4. Database Schema Safety**
- ✅ Validate against `internal/oma/database/migrations/**` and `internal/volume/database/**`
- ❌ Never assume field names
- ⚠️ Known conflict: `device_mappings.volume_id` vs `volume_uuid` → stop and ask

### **5. API / Code Design**
- ✅ Minimal endpoints
- ✅ Small, modular functions
- ❌ No simulation, stubs, placeholders, or dummy code

### **6. Versioning & Builds**
- ✅ Respect `source/current/VERSION.txt`
- ✅ Archive old versions in `source/archive/<version>`
- ✅ Output builds to `source/builds/` or `dist/`
- ❌ No binaries in source tree

## 📊 DATABASE — FIELD NAMES (EXACT)

### **OMA (MariaDB)**

**ossea_configs**: `id`, `name`, `api_url`, `api_key`, `secret_key`, `domain`, `zone`, `template_id`, `network_id`, `service_offering_id`, `disk_offering_id`, `oma_vm_id`, `created_at`, `updated_at`, `is_active`

**replication_jobs**: `id`, `source_vm_id`, `source_vm_name`, `source_vm_path`, `vcenter_host`, `datacenter`, `replication_type`, `target_network`, `status`, `progress_percent`, `current_operation`, `bytes_transferred`, `total_bytes`, `transfer_speed_bps`, `error_message`, `change_id`, `previous_change_id`, `snapshot_id`, `nbd_port`, `nbd_export_name`, `target_device`, `ossea_config_id`, `created_at`, `updated_at`, `started_at`, `completed_at`

**vm_disks**: `id`, `job_id`, `disk_id`, `vmdk_path`, `size_gb`, `datastore`, `unit_number`, `label`, `capacity_bytes`, `provisioning_type`, `ossea_volume_id`, `disk_change_id`, `sync_status`, `sync_progress_percent`, `bytes_synced`, `created_at`, `updated_at`

**failover_jobs**: `id`, `replication_job_id`, `vm_id`, `job_type`, `status`, `destination_vm_id`, `linstor_snapshot_name`, `network_mappings`, `error_message`, `created_at`, `updated_at`, `started_at`, `completed_at`

**cloudstack_job_tracking**: `id`, `cloudstack_job_id`, `cloudstack_command`, `cloudstack_status`, `operation_type`, `correlation_id`, `parent_job_id`, `status`, `created_at`, `updated_at`

### **Volume Daemon (MariaDB)**

**volume_operations**: `id`, `type`, `status`, `volume_id`, `vm_id`, `request`, `response`, `error`, `created_at`, `updated_at`, `completed_at`

**device_mappings**: `id`, `volume_id`, `vm_id`, `device_path`, `cloudstack_state`, `linux_state`, `size`, `last_sync`, `created_at`, `updated_at`

**nbd_exports**: `id`, `volume_id`, `export_name`, `device_path`, `port`, `status`, `metadata`, `created_at`, `updated_at`

⚠️ **If you see `device_mappings.volume_uuid`, STOP** — schema conflict must be resolved.

## 🛠️ OPERATIONAL FLOWS

### **Replication**
1. OMA stores job in `replication_jobs` + `vm_disks`
2. VMA → discovery, CBT ChangeIDs
3. OMA → Volume Daemon → create/attach volumes
4. NBD stream → mapped device
5. Persist ChangeIDs to `vm_disks.disk_change_id`

### **Failover**
1. Validate job/env
2. Create OSSEA VM → reattach root/data via Volume Daemon
3. Boot VM → record `failover_jobs.destination_vm_id`
4. Track in joblog

### **Cleanup**
1. Power off VM → detach volumes via Volume Daemon
2. Delete VM → reattach to OMA host
3. Update mappings → joblog + async tracking

## 📋 JOB SHEETS & DOCS

- Every task requires a job sheet (Markdown).
- Live sheets → `source/builds/jobsheets/` (artifact, not committed).
- Final sheets → `/docs/jobs/<YYYYMMDD>-<title>.md` (committed).
- Update `/docs/CHANGELOG.md` under current version.
- Update `/docs/architecture/` if workflows change.

## 🔍 CURRENT CRITICAL ISSUES (INVESTIGATION PHASE)

- **Failover Bug**: `failover_jobs.destination_vm_id` not updated after test VM creation
- **Network Detection**: Some VMs (e.g., QCDEV-AUVIK01) showing "unknown"
- **Volume Mount Conflicts**: Duplicate mounts in some scenarios
- **Logging Breakage**: Mixed `ecs.logger` and joblog in cleanup service

## ✅ EXPECTED AI BEHAVIOR

### **When Analyzing**
- Read schema files first
- Validate Volume Daemon usage
- Check joblog integration
- Verify code lives in `source/current/`

### **When Proposing Changes**
- Minimal diffs, focused, reviewable
- Confirm schema + failover engine choice first
- Use joblog + volume_client
- Require clean Git commit before applying

### **When Conflicts Arise**
- Stop on schema mismatch
- Stop on failover engine ambiguity
- Stop on archive code references

## 🚫 ABSOLUTE NO-GOS

❌ Simulation/stub/placeholder code
❌ Volume SDK calls outside daemon
❌ Logrus/printf in business logic
❌ Raw NBD ports
❌ Schema name guessing
❌ Binaries in source tree
❌ Endpoint sprawl

## 🔑 SUCCESS FACTORS

- Maintain 3.2 GiB/s NBD baseline
- All traffic via port 443
- Volume Daemon = single source of truth
- Joblog = canonical tracker
- Schema validated = no assumptions
