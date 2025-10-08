# SNA Cleanup and Deployment Plan

**Date:** October 7, 2025 15:50 BST  
**Status:** 🔴 **CRITICAL** - SNA needs cleanup and new endpoint deployment

---

## 🚨 **CURRENT SNA STATE (MESSY)**

### **Services Running:**
- ❌ `vma-api.service` (OLD, port 8081)
- ❌ `vma-autologin.service` (OLD)
- ❌ `vma-ssh-tunnel.service` (exists, OLD)
- ⚠️ `sendense-tunnel.service` (exists but NOT running)

### **Processes:**
- ❌ OLD `/opt/vma/bin/vma-api-server` on port 8081
- ⚠️ TWO SSH tunnels running (confused state)

### **Binaries:**
- ❌ 14+ old VMA binaries in `source/current/sna-api-server/` (RULE VIOLATION!)
- ✅ NEW `sendense-tunnel.sh` deployed at `/usr/local/bin/`

### **Missing:**
- ❌ NO new SNA API binary deployed
- ❌ NO `/api/v1/backup/start` endpoint

---

## 📋 **CLEANUP TASKS**

### **Task 1: Stop Old Services**
```bash
sshpass -p 'Password1' ssh vma@10.0.100.231 '
  sudo systemctl stop vma-api
  sudo systemctl disable vma-api
  sudo systemctl stop vma-ssh-tunnel 2>/dev/null
  sudo systemctl disable vma-ssh-tunnel 2>/dev/null
'
```

### **Task 2: Clean Up Old Processes**
```bash
sshpass -p 'Password1' ssh vma@10.0.100.231 '
  sudo pkill -f vma-api-server
  sudo pkill -f vma-ssh-tunnel
'
```

### **Task 3: Remove Binaries from Source Tree**
```bash
# On SHA (where source is)
cd /home/oma_admin/sendense/source/current/sna-api-server/
rm -f vma-api-server* test-vma-build
# Keep only main.go
```

### **Task 4: Start Sendense Services**
```bash
sshpass -p 'Password1' ssh vma@10.0.100.231 '
  sudo systemctl enable sendense-tunnel
  sudo systemctl start sendense-tunnel
  sudo systemctl status sendense-tunnel
'
```

---

## 🔧 **DEVELOPMENT NEEDED**

### **Required: Add Backup Endpoint to SNA API**

**File:** `source/current/sna/api/server.go`

**New Endpoint:**
```go
api.HandleFunc("/backup/start", s.handleBackupStart).Methods("POST")
```

**Handler Implementation:**
```go
func (s *Server) handleBackupStart(w http.ResponseWriter, r *http.Request) {
    // 1. Parse multi-disk NBD targets from SHA
    // 2. Call migratekit with all targets
    // 3. Return job ID and status
}
```

**Request Format (from SHA):**
```json
{
  "vm_name": "pgtest1",
  "vcenter_host": "vcenter.example.com",
  "vcenter_user": "user@vsphere.local",
  "vcenter_password": "password",
  "nbd_targets": "2000:nbd://127.0.0.1:10100/pgtest1-disk0,2001:nbd://127.0.0.1:10101/pgtest1-disk1",
  "snapshot_name": "sendense-backup-20251007-154500"
}
```

**Response Format:**
```json
{
  "job_id": "backup-job-uuid",
  "status": "started",
  "disks": [
    {"disk_key": "2000", "nbd_port": 10100},
    {"disk_key": "2001", "nbd_port": 10101}
  ]
}
```

---

## 🚀 **DEPLOYMENT SEQUENCE**

### **Phase 1: Cleanup (5 minutes)**
1. Stop old VMA services
2. Clean up processes
3. Remove binaries from source
4. Start sendense-tunnel

### **Phase 2: Develop Backup Endpoint (30-60 minutes)**
1. Add `/api/v1/backup/start` handler
2. Implement multi-disk NBD target parsing
3. Call migratekit with multi-target string
4. Build new SNA API binary

### **Phase 3: Deploy New SNA API (10 minutes)**
1. Build: `sna-api-v1.4.0-backup-endpoint`
2. Copy to SNA: `/opt/vma/bin/sna-api`
3. Create systemd service: `sna-api.service`
4. Start service on port 8081
5. Verify health and endpoints

### **Phase 4: Test Multi-Disk Backup (15 minutes)**
1. Retry pgtest1 backup from SHA
2. Verify SNA receives request
3. Confirm migratekit called with 2 NBD targets
4. Check QCOW2 files created on SHA

---

## 🎯 **SUCCESS CRITERIA**

- [ ] Old VMA services stopped and disabled
- [ ] sendense-tunnel service running
- [ ] No binaries in source tree
- [ ] New SNA API with `/backup/start` endpoint
- [ ] SHA → SNA backup call succeeds (200 OK)
- [ ] Multi-disk backup creates 2 QCOW2 files
- [ ] VMware snapshot created/deleted properly

---

## ⚠️ **BLOCKERS**

1. **SNA API Development:** Need to implement `/backup/start` endpoint
2. **migratekit Update:** May need to support multi-target format
3. **Testing:** Need working SNA environment to test

---

**Next Step:** Clean up SNA services, then develop backup endpoint

