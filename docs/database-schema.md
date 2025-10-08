# Database Schema Documentation

## 🎯 **MigrateKit OSSEA Database Schema**

Complete schema documentation for the **production-ready** MigrateKit OSSEA system with **database-integrated ChangeID storage**.

## 📊 **Core Migration Tables**

### **replication_jobs**
Primary job tracking table for migration operations.

| Column | Type | Description |
|--------|------|-------------|
| `id` | VARCHAR(255) | Primary key - format: `job-YYYYMMDD-HHMMSS` |
| `source_vm_id` | VARCHAR(255) | VMware VM ID |
| `source_vm_name` | VARCHAR(255) | VMware VM name |
| `source_vm_path` | VARCHAR(255) | **VMware inventory path** - used for ChangeID lookup |
| `vcenter_host` | VARCHAR(255) | vCenter server hostname |
| `datacenter` | VARCHAR(255) | VMware datacenter name |
| `replication_type` | VARCHAR(50) | `initial` or `incremental` |
| `target_network` | VARCHAR(255) | OSSEA target network |
| `status` | VARCHAR(50) | `initializing`, `replicating`, `completed`, `failed` |
| `progress_percent` | DOUBLE | Migration progress (0-100) |
| `ossea_config_id` | INT | FK to ossea_configs |
| `created_at` | DATETIME(3) | Job creation timestamp |
| `updated_at` | DATETIME(3) | Last status update |

### **vm_disks**
Individual disk tracking with **ChangeID storage**.

**🆕 Architecture Change (October 6, 2025):** Disk records now populated immediately during VM discovery, not just during replication. This enables backup operations without requiring a replication job.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGINT | Auto-increment primary key |
| `job_id` | VARCHAR(191) **NULLABLE** | 🆕 FK to replication_jobs.id - **NULL when populated from discovery**, set when replication starts |
| `vm_context_id` | VARCHAR(191) | 🆕 FK to vm_replication_contexts.context_id - VM-centric architecture |
| `disk_id` | LONGTEXT | Disk identifier (e.g., `disk-2000`) |
| `vm_dk_path` | LONGTEXT | **VMware disk file path** - actual .vmdk location |
| `size_gb` | BIGINT | Disk size in GB |
| `datastore` | LONGTEXT | VMware datastore name |
| `unit_number` | BIGINT | SCSI unit number |
| `label` | LONGTEXT | Disk label from VMware |
| `capacity_bytes` | BIGINT | Total disk capacity |
| `provisioning_type` | LONGTEXT | `thin`, `thick`, etc. |
| `ossea_volume_id` | BIGINT | FK to ossea_volumes.id |
| **`disk_change_id`** | LONGTEXT | **🔥 CBT ChangeID storage** |
| `sync_status` | VARCHAR(191) | `pending`, `syncing`, `completed` |
| `sync_progress_percent` | DOUBLE | Disk sync progress |
| `bytes_synced` | BIGINT | Bytes transferred |
| `created_at` | DATETIME(3) | Record creation |
| `updated_at` | DATETIME(3) | Last ChangeID update |

**Discovery vs Replication Flow:**
- **Discovery**: VM added to management → vm_disks created with `job_id = NULL`
- **Replication**: Job created → existing vm_disks updated with `job_id` value
- **Backup**: Can query vm_disks by `vm_context_id` regardless of replication status

### **cbt_history**
Complete CBT tracking and audit trail.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INT | Auto-increment primary key |
| `job_id` | VARCHAR(255) | FK to replication_jobs.id |
| `disk_id` | VARCHAR(255) | Disk identifier |
| **`change_id`** | VARCHAR(255) | **🔥 Current CBT ChangeID** |
| `previous_change_id` | VARCHAR(255) | Previous ChangeID for incremental |
| `sync_type` | VARCHAR(50) | `initial`, `incremental`, `completed` |
| `blocks_changed` | INT | Number of changed blocks |
| `bytes_transferred` | BIGINT | Actual bytes transferred |
| `sync_duration_seconds` | INT | Migration duration |
| **`sync_success`** | TINYINT(1) | **Success flag (0/1)** |
| `created_at` | DATETIME(3) | CBT record timestamp |

## 🔥 **ChangeID Storage Architecture**

### **Storage Flow**
```
migratekit → HTTP API call → OMA API (port 8082) → MariaDB
    ↓
vm_disks.disk_change_id ← OMA Database
    ↓  
cbt_history.change_id ← Audit Trail
```

### **ChangeID Format**
Real VMware CBT ChangeIDs follow this format:
```
52 3c ec 11 9e 2c 4c 3d-87 4a c3 4e 85 f2 ea 95/446
```

### **API Endpoints**
- **GET** `/api/v1/replications/changeid?vm_path=/DatabanxDC/vm/VMNAME`
- **POST** `/api/v1/replications/{job_id}/changeid`

### **Lookup Logic**
The system queries `vm_disks` joined with `replication_jobs` to find previous ChangeIDs:
```sql
SELECT vm_disks.* FROM vm_disks 
JOIN replication_jobs ON vm_disks.job_id = replication_jobs.id 
WHERE replication_jobs.source_vm_path = ? 
AND vm_disks.disk_change_id IS NOT NULL 
AND vm_disks.disk_change_id != ''
ORDER BY vm_disks.updated_at DESC
```

## 📦 **Volume Management Tables**

### **ossea_volumes**
CloudStack/OSSEA volume tracking.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INT | Auto-increment primary key |
| `volume_id` | VARCHAR(255) | CloudStack volume UUID |
| `volume_name` | VARCHAR(255) | Volume name pattern: `migration-{VMName}-{VMName}-disk-{N}` |
| `size_gb` | INT | Volume size in GB |
| `ossea_config_id` | INT | FK to ossea_configs |
| `volume_type` | VARCHAR(50) | `DATADISK`, `ROOT`, etc. |
| **`device_path`** | VARCHAR(255) | **Linux device path** (`/dev/vdb`, `/dev/vdc`) |
| `mount_point` | VARCHAR(255) | Mount point (usually empty for NBD) |
| `status` | VARCHAR(50) | `created`, `attached`, `detached` |
| `created_at` | DATETIME(3) | Volume creation |
| `updated_at` | DATETIME(3) | Last attachment update |

### **nbd_exports**
NBD export configuration tracking.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INT | Auto-increment primary key |
| `job_id` | VARCHAR(255) | FK to replication_jobs.id |
| `volume_id` | VARCHAR(255) | FK to ossea_volumes.volume_id |
| **`export_name`** | VARCHAR(255) | **NBD export name** (`migration-vm-{UUID}-disk{N}`) |
| `port` | INT | NBD port (10809) |
| **`device_path`** | VARCHAR(255) | **Target device** (`/dev/vdb`, `/dev/vdc`) |
| `config_path` | VARCHAR(255) | NBD config file path |
| `status` | VARCHAR(50) | `running`, `stopped` |
| `created_at` | DATETIME(3) | Export creation |
| `updated_at` | DATETIME(3) | Last status update |

## 🔗 **Critical Relationships**

### **Device Path Mapping**
```
ossea_volumes.device_path ←→ nbd_exports.device_path
     ↓
Migration Data Flow: VMware → NBD Export → Linux Device → CloudStack Volume
```

### **ChangeID Persistence Chain**
```
1. migratekit completes migration
2. Calls POST /api/v1/replications/{job_id}/changeid
3. OMA API updates vm_disks.disk_change_id
4. OMA API creates cbt_history record
5. Next migration: GET changeid API retrieves from vm_disks
```

## ⚠️ **Critical Schema Notes**

1. **Path Distinctions**:
   - `replication_jobs.source_vm_path` = VMware inventory path (`/DatabanxDC/vm/VMNAME`)
   - `vm_disks.vm_dk_path` = VMware disk file path (`[datastore] path/file.vmdk`)

2. **ChangeID Storage**:
   - **Primary**: `vm_disks.disk_change_id` (used for lookups)
   - **Audit**: `cbt_history.change_id` (historical tracking)

3. **Device Management**:
   - Volumes must be cleaned from `ossea_volumes` when physically detached
   - NBD exports automatically cleaned when jobs complete
   - Device paths auto-corrected if CloudStack API reports wrong paths

4. **Job Status Flow**:
   - `initializing` → `replicating` → `completed`
   - ChangeIDs only stored on successful completion
   - CBT history tracks both attempts and successes
