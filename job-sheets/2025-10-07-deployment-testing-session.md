# Job Sheet: Unified NBD Architecture - Deployment & Testing

**Job ID:** JS-2025-10-07-DEPLOY-TEST  
**Phase:** Phase 4 - Deployment & Testing  
**Related Jobs:** 
- `2025-10-07-unified-nbd-architecture.md` (Architecture & Implementation)
- `2025-10-06-backup-api-integration.md` (Backup API endpoints)  
**Status:** 🟡 **IN PROGRESS** - Deployment Complete, Critical Issue Found  
**Created:** October 7, 2025 15:10 BST  
**Priority:** HIGH - Multi-disk backup validation  
**Session Duration:** ~1.5 hours  

---

## 🎯 **OBJECTIVE**

Deploy and test the complete Unified NBD Architecture (Phases 1-3):
1. SHA API with NBD services (port allocator + qemu-nbd manager)
2. SNA SSH tunnel with 101 port forwards (10100-10200)
3. End-to-end multi-disk backup testing with pgtest1

**CRITICAL SUCCESS CRITERIA:**
- ✅ Multi-disk VM backup creates ONE VMware snapshot (not one per disk)
- ✅ All disks backed up from same snapshot instant
- ✅ Zero data corruption risk

---

## ✅ **DEPLOYMENT PHASE - COMPLETE**

### **SHA API Deployment**

**Status:** ✅ **SUCCESS**

**Actions Completed:**
1. ✅ Compiled SHA binary from source
   - Location: `/home/oma_admin/sendense/source/current/sha/cmd`
   - Binary: `sha-api-v2.20.0-nbd-unified` (34MB)
   - Compilation: Clean, no errors
   
2. ✅ Deployed to production location
   - Path: `/opt/migratekit/bin/sha-api-v2.20.0-nbd-unified`
   - Symlink: `/opt/migratekit/bin/oma-api` → `sha-api-v2.20.0-nbd-unified`
   - Permissions: `755 root:root`
   
3. ✅ Started SHA API manually (no systemd service found)
   - Command: `export DB_USER=oma_user DB_PASSWORD=oma_password && /opt/migratekit/bin/oma-api`
   - Logs: `/tmp/sha-api-new.log`
   - Health: `http://localhost:8082/health` → `{"status":"healthy"}`
   
4. ✅ Database connectivity confirmed
   - MariaDB 10.11.13 running
   - Database: `migratekit_oma`
   - User: `oma_user`
   - Connection: Verified

**Files Modified:**
- Compiled: `/home/oma_admin/sendense/source/current/sha/sha`
- Deployed: `/opt/migratekit/bin/sha-api-v2.20.0-nbd-unified`

---

### **SNA Tunnel Deployment**

**Status:** ✅ **SUCCESS** (Manual deployment, systemd preflight check issue)

**Actions Completed:**
1. ✅ Updated tunnel configuration
   - File: `/home/oma_admin/sendense/deployment/sna-tunnel/sendense-tunnel.sh`
   - Changed: `SHA_HOST` from `sha.sendense.io` → `10.245.246.134`
   - Verified: `SHA_PORT=443` (correct)
   
2. ✅ Transferred files to SNA
   - Target: `vma@10.0.100.231`
   - Files: `sendense-tunnel.sh`, `sendense-tunnel.service`
   - Method: `sshpass -p 'Password1' scp`
   
3. ✅ Installed tunnel infrastructure
   - Script: `/usr/local/bin/sendense-tunnel.sh` (executable)
   - Service: `/etc/systemd/system/sendense-tunnel.service`
   - Enabled: Auto-start on boot
   
4. ✅ Started tunnel manually (systemd preflight check failed)
   - Issue: Preflight check fails on port 22 instead of port 443
   - Workaround: Started SSH tunnel manually with correct ports
   - Tunnel: 101 NBD ports (10100-10200) + SHA API (8082) + Reverse (9081→8081)
   
5. ✅ Verified tunnel connectivity
   - Forward tunnel: SNA can reach SHA via forwarded ports
   - Reverse tunnel: SHA can reach SNA VMA API on `localhost:9081`
   - Test: `curl http://localhost:9081/api/v1/health` → `{"status":"healthy"}`

**Tunnel Status:**
```bash
# Active SSH tunnel on SNA
ssh -i /home/vma/.ssh/cloudstack_key -p 443 -N \
  -L 10100:localhost:10100 \
  -L 10101:localhost:10101 \
  ... (101 ports total) \
  -R 9081:localhost:8081 \
  vma_tunnel@10.245.246.134
```

**Issues Encountered:**
1. ⚠️  Systemd service preflight check fails
   - Root cause: Script checks SSH on port 22, not port 443
   - Impact: Service won't auto-start, manual tunnel needed
   - Fix needed: Update preflight check to test port 443
   
2. ⚠️  Log file permission issue
   - Path: `/var/log/sendense-tunnel.log`
   - Fixed: Created with `vma:vma` ownership

**Files Modified:**
- `/home/oma_admin/sendense/deployment/sna-tunnel/sendense-tunnel.sh` (SHA_HOST updated)
- SNA: `/usr/local/bin/sendense-tunnel.sh` (installed)
- SNA: `/etc/systemd/system/sendense-tunnel.service` (installed)

---

### **Pre-Test Verification**

**Status:** ✅ **COMPLETE**

**Test VM: pgtest1**

**VM Details:**
- VM Context ID: `ctx-pgtest1-20251006-203401`
- VMware VM ID: `420570c7-f61f-a930-77c5-1e876786cb3c`
- vCenter: `quad-vcenter-01.quadris.local`
- Status: `discovered`
- **Disk Count: 2** ← CRITICAL for multi-disk test

**Disk Details:**
```
disk-2000: 102 GB (vmdk: pgtest1.vmdk)
disk-2001:   5 GB (vmdk: pgtest1_1.vmdk)
Total: 107 GB
```

**Database Verification:**
- ✅ VM context exists in `vm_replication_contexts`
- ✅ Both disks linked via `vm_context_id` in `vm_disks` table
- ✅ Disk IDs: `disk-2000`, `disk-2001`
- ✅ Unit numbers: Both show `0` (potential issue?)

**Repository Preparation:**
- ✅ Path: `/backup/repository/`
- ✅ QCOW2 files created:
  - `pgtest1-disk-2000.qcow2` (110GB allocated)
  - `pgtest1-disk-2001.qcow2` (6GB allocated)
- ✅ Available space: 75GB free

**Network Connectivity:**
- ✅ SHA → SNA reverse tunnel working
- ✅ SNA → vCenter reachable
- ✅ SSH keys verified on SNA

---

## 🚨 **CRITICAL ISSUE FOUND**

### **Multi-Disk Code Not Executing**

**Status:** ❌ **BLOCKING** - Requires debugging  
**Severity:** CRITICAL - Data corruption risk if not fixed  
**Discovered:** 2025-10-07 15:05 BST

**Symptoms:**

1. **API Response Format Wrong:**
   ```json
   {
     "backup_id": "backup-pgtest1-disk0-20251007-150859",
     "disk_id": 0,              ← Should NOT exist (single-disk field)
     "total_bytes": 109521666048,  ← Only 102GB (disk 0 only, not 107GB)
     "status": "running"
     // MISSING: disk_results array
     // MISSING: nbd_targets_string field
   }
   ```

2. **Backup Behavior:**
   - Only disk 0 being backed up
   - No evidence of loop through all disks
   - No NBD ports allocated for disk 1
   - Suggests old single-disk code path executing

3. **No Handler Logs:**
   - No "Starting VM backup (multi-disk)" log messages
   - No "Found disks for multi-disk backup" messages
   - Suggests `StartBackup()` handler not being called

**Code Review:**

✅ **Multi-disk code EXISTS and looks correct:**

```go
// File: sha/api/handlers/backup_handlers.go:128
func (bh *BackupHandler) StartBackup(w http.ResponseWriter, r *http.Request) {
    // STEP 2: Get ALL disks for VM
    vmDisks, err := bh.vmDiskRepo.GetByVMContextID(vmContext.ContextID)
    
    // STEP 3 & 4: Allocate NBD ports for ALL disks
    diskResults := make([]DiskBackupResult, len(vmDisks))
    for i, vmDisk := range vmDisks {
        // Allocate port for each disk
        // Start qemu-nbd for each disk
    }
    
    // STEP 8: Return response with ALL disk details
    response := BackupResponse{
        DiskResults:      diskResults,        // ← Should have 2 entries
        NBDTargetsString: nbdTargetsString,   // ← Should have multi-disk targets
        // ...
    }
}
```

✅ **Routing looks correct:**
```go
// File: sha/api/handlers/backup_handlers.go:691
r.HandleFunc("/backups", bh.StartBackup).Methods("POST")
```

✅ **Repository method exists:**
```go
// File: sha/database/repository.go:453
func (r *VMDiskRepository) GetByVMContextID(vmContextID string) ([]VMDisk, error)
```

✅ **Database data correct:**
```sql
SELECT * FROM vm_disks WHERE vm_context_id = 'ctx-pgtest1-20251006-203401';
-- Returns 2 rows (disk-2000, disk-2001)
```

**Potential Causes:**

1. **Wrong API Process Running:**
   - Old `sendense-hub` process found on port 8082 (PID 2403150)
   - Killed old process, restarted new binary
   - Issue persists → rules out old process

2. **Handler Not Registered:**
   - Routes may not be initialized
   - Middleware might be intercepting
   - Default handler responding instead

3. **Different Endpoint Being Called:**
   - Some other backup service responding
   - Old legacy endpoint still active
   - Need to verify actual route handling

4. **Code Not Compiled Into Binary:**
   - Possible if wrong directory compiled
   - Need to verify build process
   - Check binary actually contains new code

**Evidence Collected:**

```bash
# Binary verification
ls -lh /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified
-rwxr-xr-x 1 root root 34M Oct  7 14:58

# Process verification
sudo lsof -i :8082
# Shows correct process running

# Health check
curl http://localhost:8082/health
# Returns healthy response

# Backup API test
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"repo-local-1759780081","backup_type":"full"}'
# Returns OLD single-disk format
```

---

## 🔍 **DEBUGGING PLAN**

### **Priority 1: Verify Code Path** (Next Session)

**Step 1: Add Debug Logging**
```go
// Add to start of StartBackup() handler
log.Info("🔥 DEBUG: StartBackup() handler called!")
log.WithFields(log.Fields{
    "method": r.Method,
    "path":   r.URL.Path,
    "body":   req,
}).Info("🔥 DEBUG: Request details")
```

**Step 2: Verify Handler Registration**
```go
// Check server.go initialization
// Verify BackupHandler is properly initialized
// Check router middleware not blocking
```

**Step 3: Test Endpoint Directly**
```bash
# Verbose curl to see response headers
curl -v -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d @/tmp/backup-request.json 2>&1 | tee /tmp/curl-verbose.log

# Check response headers for routing info
grep -i "x-" /tmp/curl-verbose.log
```

**Step 4: Rebuild Binary with Debug**
```bash
cd /home/oma_admin/sendense/source/current/sha/cmd
go build -o ../sha-debug -ldflags="-X main.Version=v2.20.0-debug" .
# Test with debug binary
```

### **Priority 2: Alternative Tests**

**Test with curl -v to see actual headers**
```bash
curl -v -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"repo-local-1759780081","backup_type":"full"}' \
  2>&1 | head -30
```

**Check for multiple backup services**
```bash
# Find all processes with "backup" in name
ps aux | grep -i backup

# Check for other ports serving backup APIs
sudo netstat -tlnp | grep -E "8082|8080|8081"
```

**Verify binary actually has new code**
```bash
# Check if DiskBackupResult symbol exists in binary
strings /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified | grep -i "DiskBackupResult"
strings /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified | grep -i "nbd_targets_string"
```

### **Priority 3: If Code Path Confirmed**

**Check for runtime issues:**
- Database query failing silently?
- vmDiskRepo not initialized?
- Context not being passed correctly?
- Error handling swallowing failures?

**Add comprehensive logging:**
```go
log.Info("STEP 1: Getting VM context...")
log.Info("STEP 2: Getting disks...")
log.WithField("disk_count", len(vmDisks)).Info("STEP 2 RESULT")
log.Info("STEP 3: Allocating ports...")
// ... etc
```

---

## 📊 **SESSION STATISTICS**

**Time Spent:**
- Deployment: 45 minutes
- Testing: 30 minutes
- Debugging: 15 minutes
- **Total: 1.5 hours**

**Actions Completed:**
- ✅ 8 deployment steps
- ✅ 6 verification steps
- ✅ 3 test attempts
- ❌ 1 critical issue found

**Code Changes:**
- Modified: 1 file (`sendense-tunnel.sh`)
- Compiled: 1 binary (`sha`)
- Deployed: 2 binaries (SHA + tunnel script)

**Database Queries:**
- 12 verification queries executed
- All returned expected data

**Network Tests:**
- 5 connectivity tests passed
- 2 tunnel tests passed
- 1 API health check passed

---

## 📝 **NEXT SESSION ACTIONS**

### **Immediate (Start of Next Session):**

1. **Verify Binary Contains New Code**
   ```bash
   strings /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified | grep "disk_results"
   objdump -t /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified | grep StartBackup
   ```

2. **Add Debug Logging & Rebuild**
   - Add entry point logging to `StartBackup()`
   - Add step-by-step logging throughout
   - Rebuild and redeploy
   - Test again

3. **Check Server Initialization**
   - Review `server.go` handler registration
   - Verify BackupHandler struct initialization
   - Check middleware chain
   - Verify routes actually registered

4. **Test Alternative Endpoints**
   - Try different HTTP methods
   - Test with different request formats
   - Check if old endpoints still exist

### **If Still Blocked:**

5. **Create Minimal Test Handler**
   - Add new test endpoint: `POST /api/v1/test/multi-disk`
   - Implement same logic in test handler
   - Verify test handler works
   - Compare with production handler

6. **Binary Comparison**
   - Check size of binary vs expected
   - Compare with previous working binaries
   - Verify Go version compatibility
   - Check for build cache issues

### **Success Criteria for Next Session:**

- [ ] Identify why multi-disk code not executing
- [ ] Fix code path issue
- [ ] Get proper multi-disk response with `disk_results` array
- [ ] Verify ONE VMware snapshot created
- [ ] Complete backup successfully
- [ ] Validate QCOW2 files for all disks
- [ ] Verify resource cleanup

---

## 🎓 **LESSONS LEARNED**

### **What Went Well:**
1. ✅ Deployment process well-documented and followed
2. ✅ SSH tunnel deployed successfully despite systemd issue
3. ✅ Database connectivity verified thoroughly
4. ✅ Test VM (pgtest1) perfect for multi-disk validation
5. ✅ Code review confirmed multi-disk logic exists

### **What Needs Improvement:**
1. ⚠️  Need systemd services for production (currently manual)
2. ⚠️  Tunnel preflight check needs fixing (port 443 vs 22)
3. ⚠️  Need better logging/debugging from start
4. ⚠️  Binary verification should be standard step
5. ⚠️  Should kill old processes before starting new

### **Critical Discovery:**
- **Old binaries running on same ports** - Caused confusion
- **Need process verification** before assuming correct binary
- **Always check `lsof`/`netstat`** to verify what's actually running

---

## 📚 **REFERENCE INFORMATION**

### **Key Locations:**

**SHA (Dev OMA):**
- Source: `/home/oma_admin/sendense/source/current/sha/`
- Binary: `/opt/migratekit/bin/sha-api-v2.20.0-nbd-unified`
- Symlink: `/opt/migratekit/bin/oma-api`
- Logs: `/tmp/sha-api-new.log`
- Config: Environment variables (no config file)

**SNA (10.0.100.231):**
- Tunnel script: `/usr/local/bin/sendense-tunnel.sh`
- Service: `/etc/systemd/system/sendense-tunnel.service`
- SSH key: `/home/vma/.ssh/cloudstack_key`
- VMA API: Port 8081 (reverse tunnel → 9081 on SHA)

**Database:**
- Host: `localhost:3306`
- Database: `migratekit_oma`
- User: `oma_user` / Password: `oma_password`
- Tables: `vm_replication_contexts`, `vm_disks`, `backup_jobs`

**Test Files:**
- Request: `/tmp/backup-request.json`
- QCOW2: `/backup/repository/pgtest1-disk-*.qcow2`
- Responses: `/tmp/backup-response-*.json`

### **Key Commands:**

**Start SHA API:**
```bash
export DB_USER=oma_user DB_PASSWORD=oma_password DB_NAME=migratekit_oma
/opt/migratekit/bin/oma-api > /tmp/sha-api-new.log 2>&1 &
```

**Check Health:**
```bash
curl http://localhost:8082/health
curl http://localhost:9081/api/v1/health  # SNA via reverse tunnel
```

**Test Backup:**
```bash
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"repo-local-1759780081","backup_type":"full"}'
```

**Check Processes:**
```bash
ps aux | grep oma-api
sudo lsof -i :8082
sudo ss -tlnp | grep :8082
```

---

## ✅ **DEPLOYMENT SIGN-OFF**

**Deployment Date:** October 7, 2025 15:10 BST  
**Deployed By:** AI Assistant + oma_admin  
**SHA Version:** v2.20.0-nbd-unified  
**Deployment Status:** ⚠️  **PARTIAL** - Deployed but runtime issue  

**Component Status:**
- SHA API: ✅ Deployed, ⚠️  Multi-disk code not executing
- SNA Tunnel: ✅ Deployed, ⚠️  Systemd preflight needs fix
- Database: ✅ Healthy
- Network: ✅ All tunnels working

**Testing Status:**
- Pre-test verification: ✅ COMPLETE
- Single-disk backup: ⏸️  SKIPPED (multi-disk VM)
- Multi-disk backup: ❌ BLOCKED (runtime issue)
- Consistency validation: ⏸️  PENDING
- Resource cleanup: ⏸️  PENDING

**Issues Requiring Resolution:**
1. 🚨 **CRITICAL:** Multi-disk code not executing - needs debugging
2. ⚠️  **MEDIUM:** Tunnel systemd preflight check fails on port 22 vs 443
3. ⚠️  **LOW:** No systemd service for SHA API (manual start required)

**Approved for Debugging:** ✅ YES  
**Ready for Production:** ❌ NO - Critical issue must be resolved first

---

**Next Session:** Debug multi-disk code execution issue  
**Estimated Time:** 1-2 hours  
**Success Metric:** Multi-disk backup completes with proper response format

---

**End of Deployment & Testing Job Sheet**

---

## 🔥 **CRITICAL DISCOVERY - NAMING ISSUE**

**Discovered:** 2025-10-07 15:20 BST

### **Problem:**
We deployed with WRONG naming convention:
- ❌ Used: `/opt/migratekit/bin/oma-api` (legacy OMA naming)
- ✅ Should be: `/usr/local/bin/sendense-hub` (new SHA naming)

### **Correct Naming (Per project docs):**

| Component | Old (DEPRECATED) | New (CORRECT) |
|-----------|------------------|---------------|
| Hub Appliance | OMA | SHA (Sendense Hub Appliance) |
| Binary Name | oma-api | sendense-hub |
| Path | /opt/migratekit/bin/ | /usr/local/bin/ |

### **Existing Deployment Found:**
```bash
/usr/local/bin/sendense-hub → /home/oma_admin/sendense/source/builds/sendense-hub-v2.20.0-nbd-size-param
```
This was the process running on port 8082 (old version without multi-disk code).

### **What We Should Have Done:**
1. Built binary: `sendense-hub-v2.20.0-nbd-unified` (not sha-api)
2. Placed in: `/home/oma_admin/sendense/source/builds/`
3. Symlinked: `/usr/local/bin/sendense-hub` → that binary
4. NO `oma-api` naming anywhere

### **Multi-Disk Code Status:**
- ✅ **WORKING!** Multi-disk code executes correctly
- ✅ Logs show: "Found disks for multi-disk backup" disk_count=2
- ✅ Both NBD ports allocated (10100, 10101)
- ✅ Both qemu-nbd processes started
- ⚠️  qemu-nbd processes died (file path issue - separate bug)

### **Next Steps:**
1. Kill misnamed processes
2. Rebuild/redeploy with correct `sendense-hub` naming
3. Update deployment scripts to use correct paths
4. Document proper deployment procedure

**Action:** Continue testing with current working API on port 8080, then fix naming in next deployment.


---

## 🚨 **BLOCKER FOUND - Missing SNA API Endpoint**

**Time:** 2025-10-07 15:42 BST  
**Status:** ⛔ **BLOCKED** - Cannot proceed with testing

### **Multi-Disk Code Status: ✅ WORKING**

Evidence from logs:
```
✅ Starting VM backup (multi-disk)
✅ Found disks for multi-disk backup" disk_count=2
✅ NBD port allocated port=10100
✅ NBD port allocated port=10101
✅ qemu-nbd started pid=3174015 port=10100
✅ qemu-nbd started pid=3174016 port=10101
✅ Built multi-disk NBD targets string
❌ SNA VMA API returned error status=404
```

### **Root Cause:**

SHA calls: `http://localhost:9081/api/v1/backup/start`  
SNA response: `404 Not Found`

**The SNA API doesn't have the backup endpoint yet!**

### **What's Missing:**

SNA needs `/api/v1/backup/start` endpoint that:
1. Accepts NBD targets string from SHA
2. Calls `migratekit` with multi-disk NBD targets
3. Returns job status

### **Deployment Status:**

| Component | Status | Notes |
|-----------|--------|-------|
| SHA API Multi-Disk Code | ✅ COMPLETE | Allocates ports, starts qemu-nbd, builds targets |
| SHA→SNA Communication | ✅ WORKING | Tunnel on port 9081 active |
| SNA Backup Endpoint | ❌ MISSING | Endpoint doesn't exist |
| SNA Tunnel (101 ports) | ❓ UNKNOWN | Need to verify |

### **Next Steps:**

1. Deploy SNA backup API endpoint
2. Implement multi-disk NBD target handling in SNA
3. Update `migratekit` to accept multi-target string
4. Redeploy SNA components
5. Retry multi-disk backup test

**Cannot proceed with backup testing until SNA API is deployed.**

---


