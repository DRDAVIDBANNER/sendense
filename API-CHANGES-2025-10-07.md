# API Changes - October 7, 2025 Session

## 📡 **Modified APIs**

### **1. SHA Backup API**
**Endpoint:** `POST /api/v1/backups`  
**File:** `/home/oma_admin/sendense/source/current/sha/api/handlers/backup_handlers.go`  
**Binary:** `sendense-hub-v2.20.3-credential-service`

**Changes:**
- Added `credentialService *services.VMwareCredentialService` to BackupHandler struct
- Retrieves decrypted vCenter credentials using `credential_id` from `vm_replication_contexts`
- Passes credentials to SNA API

**Request (unchanged):**
```json
{
  "vm_name": "pgtest1",
  "repository_id": "1",
  "backup_type": "full"
}
```

**Response (unchanged):**
```json
{
  "backup_id": "backup-pgtest1-1759855798",
  "vm_context_id": "ctx-pgtest1-20251006-203401",
  "vm_name": "pgtest1",
  "disk_results": [
    {
      "disk_id": 0,
      "nbd_port": 10110,
      "nbd_export_name": "pgtest1-disk0",
      "qcow2_path": "/backup/repository/pgtest1-disk0.qcow2",
      "qemu_nbd_pid": 3294943,
      "status": "qemu_started"
    }
  ],
  "nbd_targets_string": "2000:nbd://127.0.0.1:10110/pgtest1-disk0,2000:nbd://127.0.0.1:10111/pgtest1-disk0",
  "status": "started"
}
```

**New Logic:**
```go
// Get credential_id from vm_replication_contexts
if vmContext.CredentialID == nil {
    return error("VM context missing credential_id")
}

// Use credential service for decryption
creds, err := bh.credentialService.GetCredentials(r.Context(), *vmContext.CredentialID)

// Pass to SNA
snaReq := map[string]interface{}{
    "vm_name":           req.VMName,
    "vcenter_host":      creds.VCenterHost,
    "vcenter_user":      creds.Username,
    "vcenter_password":  creds.Password,  // Decrypted
    "vm_path":           vmContext.VMPath,
    "nbd_host":          "127.0.0.1",
    "nbd_targets":       nbdTargetsString,
    "job_id":            backupJobID,
    "backup_type":       req.BackupType,
}
```

---

### **2. SNA Backup API**
**Endpoint:** `POST /api/v1/backup/start`  
**File:** `/home/oma_admin/sendense/source/current/sna/api/server.go`  
**Binary:** `sna-api-v1.4.1-migratekit-flags`

**Changes:**
- Fixed migratekit command-line flags

**Request (unchanged):**
```json
{
  "vm_name": "pgtest1",
  "vcenter_host": "quad-vcenter-01.quadris.local",
  "vcenter_user": "administrator@vsphere.local",
  "vcenter_password": "EmyGVoBFesGQc47-",
  "vm_path": "/DatabanxDC/vm/pgtest1",
  "nbd_targets": "2000:nbd://127.0.0.1:10110/pgtest1-disk0,2000:nbd://127.0.0.1:10111/pgtest1-disk0",
  "job_id": "backup-pgtest1-1759855798",
  "backup_type": "full"
}
```

**Fixed Command:**
```bash
# BEFORE (WRONG):
/usr/local/bin/migratekit migrate \
  --vcenter quad-vcenter-01.quadris.local \     # ❌ Invalid
  --username administrator@vsphere.local \      # ❌ Invalid
  --password xxx \                              # ❌ Invalid

# AFTER (CORRECT):
/usr/local/bin/sendense-backup-client migrate \
  --vmware-endpoint quad-vcenter-01.quadris.local \  # ✅
  --vmware-username administrator@vsphere.local \    # ✅
  --vmware-password xxx \                            # ✅
  --vmware-path /DatabanxDC/vm/pgtest1 \
  --nbd-targets 2000:nbd://127.0.0.1:10110/pgtest1-disk0,2000:nbd://... \
  --job-id backup-pgtest1-1759855798
```

---

## 🔧 **New Binary: sendense-backup-client**

**Location:** `/usr/local/bin/sendense-backup-client` (on SNA)  
**Version:** v1.0.1-port-fix  
**Size:** 20MB  
**Source:** `/home/oma_admin/sendense/source/current/sendense-backup-client/`

**Key Features:**
1. ✅ **No environment variables required** (OpenStack/CloudStack disabled)
2. ✅ **NBD port extraction** from URLs (e.g., `nbd://127.0.0.1:10106/export`)
3. ✅ **Multi-disk support** via `--nbd-targets` flag
4. ✅ **Works for backups only** (VM shutdown/OpenStack creation disabled)

**Command Format:**
```bash
sendense-backup-client migrate \
  --vmware-endpoint <vcenter-host> \
  --vmware-username <username> \
  --vmware-password <password> \
  --vmware-path <vm-path> \
  --nbd-targets <disk_key>:nbd://<host>:<port>/<export>,... \
  --job-id <job-id>
```

**Example:**
```bash
sendense-backup-client migrate \
  --vmware-endpoint quad-vcenter-01.quadris.local \
  --vmware-username administrator@vsphere.local \
  --vmware-password EmyGVoBFesGQc47- \
  --vmware-path /DatabanxDC/vm/pgtest1 \
  --nbd-targets 2000:nbd://127.0.0.1:10110/pgtest1-disk0,2000:nbd://127.0.0.1:10111/pgtest1-disk0 \
  --job-id backup-pgtest1-1759855798
```

**What It Does:**
1. Connects to VMware vCenter
2. Creates quiesced snapshot
3. Reads disk data via VMware NBD
4. Writes to NBD targets (qemu-nbd on SHA)
5. Removes snapshot
6. **Exits (no VM shutdown, no OpenStack creation)**

**Logs:** `/var/log/sendense/backup-<job-id>.log`

---

## 🗄️ **Database Changes**

### **Required for pgtest1:**
```sql
-- vm_replication_contexts needs credential_id set
UPDATE vm_replication_contexts 
SET credential_id = 35 
WHERE vm_name = 'pgtest1';
```

**Verification:**
```sql
SELECT context_id, vm_name, vcenter_host, credential_id 
FROM vm_replication_contexts 
WHERE vm_name = 'pgtest1';
```

**Expected:**
```
ctx-pgtest1-20251006-203401 | pgtest1 | quad-vcenter-01.quadris.local | 35
```

---

## 🔌 **SSH Tunnel Configuration**

**SNA → SHA Tunnel:** `sendense-tunnel.service`

**Ports:**
```bash
# Forward (SNA → SHA):
-L 10100:localhost:10100    # NBD port 1
-L 10101:localhost:10101    # NBD port 2
...
-L 10200:localhost:10200    # NBD port 101
-L 8082:localhost:8082      # SHA API

# Reverse (SHA → SNA):
-R 9081:localhost:8081      # SNA API (for SHA to call)
```

**Status Check:**
```bash
# On SNA
sudo systemctl status sendense-tunnel

# On SHA
sudo lsof -i :9081  # Should show SSH process listening
```

---

## 📊 **Service Locations**

### **SNA (10.0.100.231)**
- Binary: `/usr/local/bin/sendense-backup-client` (v1.0.1-port-fix)
- Binary: `/usr/local/bin/sna-api` (v1.4.1-migratekit-flags)
- Service: `sna-api.service` (port 8081)
- Service: `sendense-tunnel.service` (SSH tunnel)
- Logs: `/var/log/sendense/backup-*.log`

### **SHA (10.245.246.134)**
- Binary: `/usr/local/bin/sendense-hub` → `sendense-hub-v2.20.3-credential-service`
- Service: Running on port 8082 (no systemd yet)
- Logs: `/tmp/sha-api.log` (stdout/stderr redirect)
- Repository: `/backup/repository/` (QCOW2 files)

---

## 🚨 **Known Issues**

### **BLOCKER: qemu-nbd Processes Exit Immediately**

**Symptom:**
- SHA API reports qemu PID in response
- PID doesn't exist when checked
- No ports listening (10110, 10111, etc.)
- sendense-backup-client gets "server disconnected unexpectedly"

**Next Steps:**
1. Read `/home/oma_admin/sendense/source/current/sha/services/qemu_nbd_manager.go`
2. Check how qemu-nbd is spawned
3. Verify QCOW2 file creation
4. Capture qemu-nbd stderr/stdout
5. Test manually

**Test Command:**
```bash
# Manual qemu-nbd test
qemu-img create -f qcow2 /backup/repository/test.qcow2 100G
qemu-nbd -f qcow2 -x test-export -p 10150 -b 0.0.0.0 --shared=10 -t /backup/repository/test.qcow2
```

---

**Document Created:** 2025-10-07 17:52 UTC  
**For Session:** Next session (qemu-nbd debugging)
