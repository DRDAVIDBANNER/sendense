# TESTING CHECKLIST - PGTEST1 BACKUP
## Unified NBD Architecture - Backup Testing

**Date:** October 7, 2025  
**Test VM:** pgtest1  
**SHA (Dev OMA):** localhost  
**SNA:** 10.0.100.231 (vma@10.0.100.231)  
**Version:** v2.20.0-nbd-unified

---

## 🎯 TESTING OVERVIEW

**What we're testing:**
1. ✅ Single VM backup workflow (pgtest1)
2. ✅ NBD port allocation (dynamic)
3. ✅ qemu-nbd process management (--shared=10)
4. ✅ SSH tunnel forwarding (NBD traffic)
5. ✅ Multi-disk consistency (if applicable)
6. ✅ VM-level backup (ONE snapshot for all disks)

**Success criteria:**
- Backup job starts successfully
- NBD port allocated from pool (10100-10200)
- qemu-nbd starts with `--shared=10` flag
- SNA receives NBD connection details
- VMware snapshot created (ONE for entire VM)
- Data transfers through tunnel
- Backup completes successfully
- All resources cleaned up

---

## 📋 PRE-TEST VERIFICATION

### **Step 0.1: Verify Deployment Complete**

```bash
# Verify you completed deployment checklist
cat /home/oma_admin/sendense/DEPLOYMENT-DEV-CHECKLIST.md | grep "Deployment Status:"

# Confirm:
# - SHA API running: systemctl status oma-api
# - SNA tunnel running: sshpass -p 'Password1' ssh vma@10.0.100.231 "systemctl status sendense-tunnel"
```

- [ ] ✅ Deployment checklist complete
- [ ] ✅ SHA API running and healthy
- [ ] ✅ SNA tunnel connected

---

### **Step 0.2: Verify pgtest1 in Database**

```bash
# Get pgtest1 context
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT id, vm_name, vmware_vm_id, vcenter_host, current_status 
   FROM vm_replication_contexts 
   WHERE vm_name LIKE '%pgtest1%';" \
  | tee /tmp/pgtest1-context.txt

# Get pgtest1 disks
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT d.id, d.vm_context_id, d.disk_key, d.disk_size_bytes, d.vmdk_path, d.datastore_name 
   FROM vm_disks d 
   JOIN vm_replication_contexts v ON d.vm_context_id = v.id 
   WHERE v.vm_name LIKE '%pgtest1%';" \
  | tee /tmp/pgtest1-disks.txt

# Count disks
DISK_COUNT=$(mysql -u oma_user -poma_password migratekit_oma -N -e \
  "SELECT COUNT(*) FROM vm_disks d 
   JOIN vm_replication_contexts v ON d.vm_context_id = v.id 
   WHERE v.vm_name LIKE '%pgtest1%';")

echo "pgtest1 has $DISK_COUNT disk(s)"
```

- [ ] ✅ pgtest1 found in database
- [ ] **VM Context ID:** _______________
- [ ] **VMware VM ID:** _______________
- [ ] **Number of disks:** _______________
- [ ] **Current status:** _______________

**Record disk details:**
- Disk 1: Key: _______ Size: _______ Path: _______
- Disk 2: Key: _______ Size: _______ Path: _______ (if applicable)
- Disk 3: Key: _______ Size: _______ Path: _______ (if applicable)

---

### **Step 0.3: Check Repository Storage**

```bash
# Check repository location
df -h /backup 2>/dev/null || df -h /var/lib/sendense 2>/dev/null || df -h /

# Create QCOW2 files for pgtest1 disks (if not exists)
REPO_PATH="/backup/repository"
sudo mkdir -p "$REPO_PATH"

# Get pgtest1 disk sizes from database
mysql -u oma_user -poma_password migratekit_oma -N -e \
  "SELECT d.disk_key, d.disk_size_bytes 
   FROM vm_disks d 
   JOIN vm_replication_contexts v ON d.vm_context_id = v.id 
   WHERE v.vm_name LIKE '%pgtest1%';" \
  | while read disk_key disk_size; do
    qcow_file="$REPO_PATH/pgtest1-${disk_key}.qcow2"
    if [ ! -f "$qcow_file" ]; then
        echo "Creating QCOW2 for disk $disk_key (size: $disk_size bytes)"
        # Add ~10% for metadata overhead
        size_gb=$(( ($disk_size / 1073741824) + 5 ))
        qemu-img create -f qcow2 "$qcow_file" "${size_gb}G"
        echo "✅ Created: $qcow_file"
    else
        echo "✅ Exists: $qcow_file"
    fi
done

# List QCOW2 files
ls -lh "$REPO_PATH"/pgtest1-*.qcow2
```

- [ ] ✅ Repository location identified
- [ ] **Repository path:** _______________
- [ ] **Available space:** _______________
- [ ] ✅ QCOW2 files created for all disks
- [ ] **QCOW2 files:** _______________

---

### **Step 0.4: Verify VMware Connectivity**

```bash
# Get vCenter details from pgtest1
VC_HOST=$(mysql -u oma_user -poma_password migratekit_oma -N -e \
  "SELECT vcenter_host FROM vm_replication_contexts WHERE vm_name LIKE '%pgtest1%';")

echo "vCenter host: $VC_HOST"

# Test vCenter connectivity (from SNA)
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "curl -k -s -o /dev/null -w '%{http_code}' https://$VC_HOST 2>/dev/null" \
  && echo "✅ vCenter reachable from SNA" \
  || echo "❌ vCenter NOT reachable from SNA"
```

- [ ] ✅ vCenter host identified
- [ ] **vCenter:** _______________
- [ ] ✅ vCenter reachable from SNA

---

## 🧪 TEST 1: SINGLE-DISK BACKUP (IF APPLICABLE)

**Skip this if pgtest1 has multiple disks - go to Test 2**

### **Test 1.1: Prepare Single-Disk Backup Request**

```bash
# Get VM context ID
VM_CONTEXT_ID=$(mysql -u oma_user -poma_password migratekit_oma -N -e \
  "SELECT id FROM vm_replication_contexts WHERE vm_name LIKE '%pgtest1%';")

echo "VM Context ID: $VM_CONTEXT_ID"

# Prepare backup request JSON
cat > /tmp/backup-request.json <<EOF
{
  "vm_context_id": "$VM_CONTEXT_ID",
  "backup_type": "full",
  "repository_path": "/backup/repository"
}
EOF

cat /tmp/backup-request.json
```

- [ ] ✅ Backup request prepared
- [ ] **VM Context ID:** _______________

---

### **Test 1.2: Start Backup via API**

```bash
# Start backup
echo "Starting backup..."
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d @/tmp/backup-request.json \
  2>&1 | tee /tmp/backup-response.json

# Pretty print response
cat /tmp/backup-response.json | jq '.'

# Extract job ID and NBD details
JOB_ID=$(cat /tmp/backup-response.json | jq -r '.job_id // .backup_job_id // empty')
NBD_PORT=$(cat /tmp/backup-response.json | jq -r '.nbd_port // empty')

echo "Job ID: $JOB_ID"
echo "NBD Port: $NBD_PORT"
```

**Expected response:**
```json
{
  "job_id": "backup-xxxxx",
  "vm_context_id": "xxx",
  "nbd_port": 10100,
  "nbd_targets_string": "disk0:nbd://127.0.0.1:10100/pgtest1-disk0",
  "disk_results": [
    {
      "disk_key": "disk0",
      "nbd_port": 10100,
      "qcow2_path": "/backup/repository/pgtest1-disk0.qcow2",
      "status": "started"
    }
  ],
  "status": "started",
  "message": "Backup started successfully"
}
```

- [ ] ✅ API call successful (HTTP 200)
- [ ] **Job ID:** _______________
- [ ] **NBD Port:** _______________
- [ ] **Status:** _______________

---

### **Test 1.3: Verify NBD Port Allocated**

```bash
# Check port allocation
curl -s http://localhost:8082/api/v1/nbd/ports | jq '.'

# Expected: Port 10100-10200 allocated to job
```

- [ ] ✅ NBD port allocated
- [ ] **Allocated port:** _______________
- [ ] **Job ID:** _______________

---

### **Test 1.4: Verify qemu-nbd Process**

```bash
# Check qemu-nbd process
curl -s http://localhost:8082/api/v1/nbd/processes | jq '.'

# Check process on system
ps aux | grep qemu-nbd | grep -v grep

# Verify --shared flag
ps aux | grep qemu-nbd | grep -v grep | grep -o "\-\-shared=[0-9]*"
```

**Expected:**
- qemu-nbd process running
- Port matches allocated port
- `--shared=10` flag present

- [ ] ✅ qemu-nbd process running
- [ ] **Process ID:** _______________
- [ ] **Port:** _______________
- [ ] ✅ `--shared=10` flag present

---

### **Test 1.5: Verify SNA Received Request**

```bash
# Check SNA VMA API logs
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '2 minutes ago' | grep -i 'backup\|nbd' | tail -20"

# Check if SBC process started
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "ps aux | grep sendense-backup-client | grep -v grep"
```

- [ ] ✅ SNA received backup request
- [ ] ✅ SBC process started
- [ ] **SBC process:** _______________

---

### **Test 1.6: Monitor Backup Progress**

```bash
# Watch backup progress (refresh every 5 seconds)
watch -n 5 "curl -s http://localhost:8082/api/v1/backups/$JOB_ID/status | jq '.'"

# OR check logs
journalctl -u oma-api -f | grep -i "backup\|progress\|nbd"

# Check QCOW2 file growing
watch -n 5 "ls -lh /backup/repository/pgtest1-*.qcow2"
```

- [ ] ✅ Backup progressing
- [ ] **Progress:** _______________
- [ ] **QCOW2 size growing:** Yes / No

---

### **Test 1.7: Verify Backup Completion**

```bash
# Check final status
curl -s http://localhost:8082/api/v1/backups/$JOB_ID/status | jq '.'

# Check QCOW2 file final size
ls -lh /backup/repository/pgtest1-*.qcow2

# Verify QCOW2 integrity
qemu-img check /backup/repository/pgtest1-*.qcow2

# Check qemu-nbd stopped
ps aux | grep qemu-nbd | grep -v grep || echo "✅ qemu-nbd stopped"

# Check port released
curl -s http://localhost:8082/api/v1/nbd/ports | jq '.allocated_ports'
```

**Expected:**
- Status: "completed"
- QCOW2 file exists with reasonable size
- QCOW2 integrity check passes
- qemu-nbd process stopped
- Port released back to pool

- [ ] ✅ Backup completed successfully
- [ ] **Final status:** _______________
- [ ] **Backup duration:** _______________
- [ ] **QCOW2 size:** _______________
- [ ] ✅ QCOW2 integrity OK
- [ ] ✅ qemu-nbd stopped
- [ ] ✅ Port released

---

## 🧪 TEST 2: MULTI-DISK BACKUP (CRITICAL TEST)

**This tests the CRITICAL Task 2.4 fix - ONE snapshot for ALL disks**

### **Test 2.1: Verify Multi-Disk VM**

```bash
# Count disks
DISK_COUNT=$(mysql -u oma_user -poma_password migratekit_oma -N -e \
  "SELECT COUNT(*) FROM vm_disks d 
   JOIN vm_replication_contexts v ON d.vm_context_id = v.id 
   WHERE v.vm_name LIKE '%pgtest1%';")

echo "pgtest1 has $DISK_COUNT disk(s)"

if [ "$DISK_COUNT" -lt 2 ]; then
    echo "⚠️  pgtest1 only has 1 disk - multi-disk test not applicable"
    exit 0
else
    echo "✅ pgtest1 has $DISK_COUNT disks - multi-disk test applicable"
fi

# List all disks
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT d.disk_key, d.disk_size_bytes, d.vmdk_path 
   FROM vm_disks d 
   JOIN vm_replication_contexts v ON d.vm_context_id = v.id 
   WHERE v.vm_name LIKE '%pgtest1%';"
```

- [ ] ✅ pgtest1 has multiple disks
- [ ] **Disk count:** _______________

---

### **Test 2.2: Prepare Multi-Disk Backup Request**

```bash
# Get VM context ID
VM_CONTEXT_ID=$(mysql -u oma_user -poma_password migratekit_oma -N -e \
  "SELECT id FROM vm_replication_contexts WHERE vm_name LIKE '%pgtest1%';")

# Prepare VM-level backup request (NO disk_id!)
cat > /tmp/backup-multi-request.json <<EOF
{
  "vm_context_id": "$VM_CONTEXT_ID",
  "backup_type": "full",
  "repository_path": "/backup/repository"
}
EOF

echo "Multi-disk backup request:"
cat /tmp/backup-multi-request.json | jq '.'
```

- [ ] ✅ Multi-disk backup request prepared
- [ ] **VM Context ID:** _______________

---

### **Test 2.3: Start Multi-Disk Backup**

```bash
# Start backup
echo "Starting multi-disk backup..."
START_TIME=$(date +%s)

curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d @/tmp/backup-multi-request.json \
  2>&1 | tee /tmp/backup-multi-response.json

# Pretty print
cat /tmp/backup-multi-response.json | jq '.'

# Extract details
JOB_ID=$(cat /tmp/backup-multi-response.json | jq -r '.job_id // .backup_job_id // empty')
DISK_RESULTS=$(cat /tmp/backup-multi-response.json | jq '.disk_results')

echo ""
echo "Job ID: $JOB_ID"
echo "Disk Results:"
echo "$DISK_RESULTS" | jq '.'
```

**Expected response:**
```json
{
  "job_id": "backup-xxxxx",
  "vm_context_id": "xxx",
  "nbd_targets_string": "disk0:nbd://127.0.0.1:10100/pgtest1-disk0,disk1:nbd://127.0.0.1:10101/pgtest1-disk1",
  "disk_results": [
    {
      "disk_key": "disk0",
      "nbd_port": 10100,
      "qcow2_path": "/backup/repository/pgtest1-disk0.qcow2",
      "status": "started"
    },
    {
      "disk_key": "disk1",
      "nbd_port": 10101,
      "qcow2_path": "/backup/repository/pgtest1-disk1.qcow2",
      "status": "started"
    }
  ],
  "status": "started",
  "message": "Backup started successfully"
}
```

- [ ] ✅ API call successful
- [ ] **Job ID:** _______________
- [ ] **Number of disks in response:** _______________
- [ ] ✅ Multiple NBD targets in response

---

### **Test 2.4: Verify Multiple NBD Ports Allocated**

```bash
# Check port allocations
curl -s http://localhost:8082/api/v1/nbd/ports | jq '.'

# Count allocated ports for this job
ALLOCATED=$(curl -s http://localhost:8082/api/v1/nbd/ports | jq ".allocated_ports | map(select(.job_id == \"$JOB_ID\")) | length")

echo "Ports allocated for job $JOB_ID: $ALLOCATED"
```

**Expected:** Number of ports = number of disks (e.g., 2 disks = 2 ports)

- [ ] ✅ Multiple ports allocated
- [ ] **Ports allocated:** _______________
- [ ] ✅ Matches disk count

---

### **Test 2.5: Verify Multiple qemu-nbd Processes**

```bash
# Check qemu-nbd processes
curl -s http://localhost:8082/api/v1/nbd/processes | jq '.'

# Count processes
ps aux | grep qemu-nbd | grep -v grep | wc -l

# Verify all have --shared=10
ps aux | grep qemu-nbd | grep -v grep
```

**Expected:** One qemu-nbd process per disk, all with `--shared=10`

- [ ] ✅ Multiple qemu-nbd processes
- [ ] **Process count:** _______________
- [ ] ✅ All have `--shared=10` flag
- [ ] **Ports:** _______________

---

### **Test 2.6: CRITICAL - Verify ONE VMware Snapshot**

**This is the CRITICAL test - verifies Task 2.4 fix**

```bash
# Check SNA logs for snapshot creation
echo "Checking VMware snapshot events..."
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '2 minutes ago' | grep -i 'snapshot' | head -20"

# Look for snapshot name
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '2 minutes ago' | grep -i 'snapshot' | grep -o 'snap-[a-zA-Z0-9-]*' | head -1" \
  | tee /tmp/snapshot-name.txt

SNAPSHOT_NAME=$(cat /tmp/snapshot-name.txt)
echo "Snapshot name: $SNAPSHOT_NAME"

# Count snapshot creation events (should be 1, not N)
SNAPSHOT_COUNT=$(sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '2 minutes ago' | grep -c 'Creating snapshot' || echo 0")

echo "Snapshot creation events: $SNAPSHOT_COUNT"

if [ "$SNAPSHOT_COUNT" -eq 1 ]; then
    echo "✅ PASS: Only ONE snapshot created (correct!)"
else
    echo "❌ FAIL: $SNAPSHOT_COUNT snapshots created (should be 1!)"
fi
```

**CRITICAL VALIDATION:**
- **Expected:** ONE snapshot event (for entire VM)
- **NOT:** Multiple snapshot events (one per disk)

- [ ] ✅ **CRITICAL: Only ONE snapshot created**
- [ ] **Snapshot count:** _______________
- [ ] **Snapshot name:** _______________

---

### **Test 2.7: Verify SNA Received Multi-Disk Targets**

```bash
# Check SNA logs for NBD targets string
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '2 minutes ago' | grep -i 'nbd-targets' | tail -5"

# Check SBC command line
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "ps aux | grep sendense-backup-client | grep -v grep"

# Look for --nbd-targets flag with multiple disks
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "ps aux | grep 'nbd-targets' | grep -v grep"
```

**Expected:** SBC process with `--nbd-targets disk0:nbd://...,disk1:nbd://...`

- [ ] ✅ SBC received multi-disk targets
- [ ] ✅ Multiple disks in --nbd-targets flag
- [ ] **SBC command:** _______________

---

### **Test 2.8: Monitor Multi-Disk Backup Progress**

```bash
# Watch progress
watch -n 5 "curl -s http://localhost:8082/api/v1/backups/$JOB_ID/status | jq '.'"

# Watch all QCOW2 files growing
watch -n 5 "ls -lh /backup/repository/pgtest1-*.qcow2"

# Check SHA logs
journalctl -u oma-api -f | grep -i "backup\|progress\|nbd"
```

- [ ] ✅ All disks progressing
- [ ] **Progress:** _______________

---

### **Test 2.9: Verify Multi-Disk Completion**

```bash
# Check final status
curl -s http://localhost:8082/api/v1/backups/$JOB_ID/status | jq '.'

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
echo "Backup duration: $DURATION seconds"

# Check all QCOW2 files
ls -lh /backup/repository/pgtest1-*.qcow2

# Verify all QCOW2 integrity
for qcow in /backup/repository/pgtest1-*.qcow2; do
    echo "Checking: $qcow"
    qemu-img check "$qcow"
done

# Verify all qemu-nbd stopped
ps aux | grep qemu-nbd | grep -v grep || echo "✅ All qemu-nbd processes stopped"

# Verify all ports released
curl -s http://localhost:8082/api/v1/nbd/ports | jq '.allocated_ports'
```

**Expected:**
- Status: "completed"
- All QCOW2 files exist
- All integrity checks pass
- All qemu-nbd processes stopped
- All ports released

- [ ] ✅ Backup completed successfully
- [ ] **Final status:** _______________
- [ ] **Duration:** _______________ seconds
- [ ] ✅ All QCOW2 files created
- [ ] **File sizes:**
  - Disk 0: _______________
  - Disk 1: _______________
  - Disk 2: _______________ (if applicable)
- [ ] ✅ All QCOW2 integrity checks passed
- [ ] ✅ All qemu-nbd processes stopped
- [ ] ✅ All ports released

---

### **Test 2.10: Verify Snapshot Deleted**

```bash
# Check SNA logs for snapshot deletion
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '5 minutes ago' | grep -i 'delete.*snapshot\|remove.*snapshot' | tail -10"

# Count delete events (should be 1)
DELETE_COUNT=$(sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '5 minutes ago' | grep -c 'Deleting snapshot' || echo 0")

echo "Snapshot deletion events: $DELETE_COUNT"

if [ "$DELETE_COUNT" -eq 1 ]; then
    echo "✅ PASS: Snapshot deleted cleanly"
else
    echo "⚠️  WARNING: Expected 1 delete event, found $DELETE_COUNT"
fi
```

- [ ] ✅ Snapshot deleted
- [ ] **Delete count:** _______________

---

## 🎯 TEST 3: CONSISTENCY VALIDATION (MULTI-DISK)

**This validates that all disks were backed up from the SAME instant**

### **Test 3.1: Check QCOW2 Creation Times**

```bash
# Get creation times of all QCOW2 files
echo "QCOW2 file timestamps:"
stat /backup/repository/pgtest1-*.qcow2 | grep -E "File:|Modify:"

# Calculate time difference between first and last file modification
FIRST_TIME=$(stat -c %Y /backup/repository/pgtest1-*.qcow2 | sort -n | head -1)
LAST_TIME=$(stat -c %Y /backup/repository/pgtest1-*.qcow2 | sort -n | tail -1)
TIME_DIFF=$((LAST_TIME - FIRST_TIME))

echo "Time difference between first and last QCOW2 modification: $TIME_DIFF seconds"

if [ "$TIME_DIFF" -lt 10 ]; then
    echo "✅ PASS: All files created within 10 seconds (consistent)"
else
    echo "⚠️  WARNING: $TIME_DIFF second difference (may indicate separate snapshots)"
fi
```

- [ ] ✅ All QCOW2 files created within 10 seconds
- [ ] **Time difference:** _______________ seconds

---

### **Test 3.2: Verify Snapshot Timing in Logs**

```bash
# Extract snapshot timestamps from logs
echo "Snapshot timing analysis:"
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '10 minutes ago' --output=short-precise | grep -i 'snapshot'" \
  | tee /tmp/snapshot-timing.txt

# Count unique snapshot names created
UNIQUE_SNAPSHOTS=$(cat /tmp/snapshot-timing.txt | grep -o 'snap-[a-zA-Z0-9-]*' | sort -u | wc -l)

echo "Unique snapshots created: $UNIQUE_SNAPSHOTS"

if [ "$UNIQUE_SNAPSHOTS" -eq 1 ]; then
    echo "✅ PASS: Only ONE unique snapshot (consistent backup)"
else
    echo "❌ FAIL: $UNIQUE_SNAPSHOTS snapshots created (data corruption risk!)"
fi
```

**CRITICAL:**
- **Expected:** 1 unique snapshot
- **NOT:** Multiple snapshots (would indicate broken multi-disk consistency)

- [ ] ✅ **CRITICAL: Only ONE unique snapshot**
- [ ] **Unique snapshot count:** _______________

---

## 🧪 TEST 4: STRESS TEST (OPTIONAL)

### **Test 4.1: Multiple Concurrent Backups**

```bash
# Start 3 concurrent backups (if you have 3+ VMs)
for vm in pgtest1 pgtest2 pgtest3; do
    VM_CONTEXT_ID=$(mysql -u oma_user -poma_password migratekit_oma -N -e \
      "SELECT id FROM vm_replication_contexts WHERE vm_name = '$vm';")
    
    if [ -n "$VM_CONTEXT_ID" ]; then
        echo "Starting backup for $vm..."
        curl -X POST http://localhost:8082/api/v1/backups \
          -H "Content-Type: application/json" \
          -d "{\"vm_context_id\": \"$VM_CONTEXT_ID\", \"backup_type\": \"full\"}" \
          &
    fi
done

wait

# Check port allocations
curl -s http://localhost:8082/api/v1/nbd/ports | jq '.'

# Check process count
ps aux | grep qemu-nbd | grep -v grep | wc -l
```

- [ ] ✅ Multiple concurrent backups started
- [ ] **Concurrent count:** _______________
- [ ] ✅ Unique ports allocated for each

---

## 📊 TESTING SUMMARY

### **Test Results**

| Test | Status | Notes |
|------|--------|-------|
| **Pre-Test Verification** | ✅ / ❌ | _______________ |
| **Single-Disk Backup** | ✅ / ❌ / N/A | _______________ |
| **Multi-Disk Backup** | ✅ / ❌ / N/A | _______________ |
| **ONE Snapshot (CRITICAL)** | ✅ / ❌ | _______________ |
| **NBD Port Allocation** | ✅ / ❌ | _______________ |
| **qemu-nbd Management** | ✅ / ❌ | _______________ |
| **SSH Tunnel Traffic** | ✅ / ❌ | _______________ |
| **Consistency Validation** | ✅ / ❌ | _______________ |
| **Resource Cleanup** | ✅ / ❌ | _______________ |

---

### **Performance Metrics**

**Single-Disk Backup (if tested):**
- Duration: _______________ seconds
- Data size: _______________ GB
- Throughput: _______________ MB/s
- NBD port: _______________

**Multi-Disk Backup (if tested):**
- Duration: _______________ seconds
- Total data size: _______________ GB
- Throughput: _______________ MB/s
- NBD ports: _______________
- Disk count: _______________

---

### **Critical Validations**

**Task 2.4 Fix Validation (Multi-Disk Consistency):**
- [ ] ✅ **PASS:** Only ONE VMware snapshot created
- [ ] ✅ **PASS:** All disks backed up from SAME instant
- [ ] ✅ **PASS:** VM-level backup API (not disk-level)
- [ ] ✅ **PASS:** Multi-disk NBD targets string used

**If ANY of the above are FAIL:** **DATA CORRUPTION RISK - DO NOT USE IN PRODUCTION**

---

### **Issues Encountered**

1. **Issue:** _______________  
   **Resolution:** _______________

2. **Issue:** _______________  
   **Resolution:** _______________

3. **Issue:** _______________  
   **Resolution:** _______________

---

## ✅ TESTING SIGN-OFF

**Test Date:** _______________  
**Tested By:** _______________  
**Test Environment:** Dev OMA + SNA (10.0.100.231)  
**Test VM:** pgtest1  

**Overall Result:** ✅ PASS / ❌ FAIL

**Critical Tests:**
- Multi-Disk Consistency: ✅ / ❌
- ONE Snapshot Validation: ✅ / ❌
- Resource Cleanup: ✅ / ❌

**Recommendation:**
- [ ] ✅ Approve for production pilot
- [ ] ⚠️  Additional testing required
- [ ] ❌ Block production - critical issues found

**Notes:**
_______________________________________________
_______________________________________________
_______________________________________________

**Approved By:** _______________ **Date:** _______________

---

## 🚀 NEXT STEPS

**If testing PASSED:**
1. Document test results
2. Update CHANGELOG.md
3. Prepare for production rollout
4. Train operations team
5. Set up monitoring/alerting

**If testing FAILED:**
1. Review logs for errors
2. Identify root cause
3. File bug report
4. Roll back if necessary
5. Fix and re-test

---

**TESTING COMPLETE!** 🎉

**For questions:** Refer to comprehensive documentation in `/home/oma_admin/sendense/`

---

**End of Testing Checklist**
