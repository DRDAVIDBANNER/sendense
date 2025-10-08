# MigrateKit NBD Hang Investigation - Session Handover

**Date:** October 7, 2025 05:35 UTC  
**Status:** BLOCKING - migratekit hangs after NBD connection, preventing backup workflow testing  
**Priority:** HIGH - Final blocker for Phase 1 VMware Backup API testing

---

## 🎯 **Problem Statement**

**migratekit** on the SNA (10.0.100.231) successfully connects to qemu-nbd on SHA (via SSH tunnel), retrieves correct QCOW2 virtual disk size, but then **hangs indefinitely** after logging "✅ NBD metadata context enabled for sparse optimization" - preventing any actual data transfer.

---

## 🔍 **Symptoms**

### Observable Behavior
- ✅ SNA successfully connects to SHA qemu-nbd export via SSH tunnel (localhost:10808 → 10809)
- ✅ libnbd negotiation completes successfully
- ✅ Export size correctly reported as 109521666048 bytes (102GB)
- ✅ VMware CBT analysis completes (36GB actual usage calculated)
- ✅ "NBD metadata context enabled for sparse optimization" logged
- ❌ **HANG** - No further progress, no worker startup, no data transfer
- ❌ Process state: `S (sleeping)`, 10 threads, 11 open file descriptors
- ❌ No errors in logs, no timeouts, just silence

### Log Evidence (from SNA)
```
time="2025-10-07T04:29:47Z" level=info msg="📊 Using CBT-calculated disk usage: 36 GB used of 102 GB total"
time="2025-10-07T04:29:47Z" level=info msg="✅ NBD metadata context enabled for sparse optimization"
[NOTHING AFTER THIS - HUNG]
```

**Expected next log:** `"🔧 Using 3 parallel workers for full copy"` (line 73 of parallel_full_copy.go)

---

## 🧪 **What We've Tested and Ruled Out**

### ✅ Confirmed Working
1. **SSH Tunnel:** VMA → SHA tunnel operational (vma-ssh-tunnel.service running)
2. **qemu-nbd Server:** Runs successfully, listens on 127.0.0.1:10809
3. **NBD Export Creation:** QCOW2 file exported correctly with proper virtual size
4. **Connection Establishment:** libnbd successfully connects and negotiates
5. **Size Detection:** qemu-nbd reports correct 109GB QCOW2 virtual size
6. **Metadata Context Support:** qemu-nbd supports `base:allocation` (has `-A` flag)

### ❌ Fixes Attempted (All Failed)
1. **NBD server `allowlist`:** Changed to `false` - no change
2. **NBD server `listenaddr`:** Set to `127.0.0.1` - no change
3. **SSH tunnel `PermitOpen`:** Changed to `any` - no change
4. **NBD export `readonly`:** Changed to writable for backups - no change
5. **NBD config `size` parameter:** Fixed from `filesize` to `size` - no change
6. **qemu-nbd `-A` flag:** Added allocation-depth flag - **SAME HANG**
7. **Multiple test jobs:** All hang at identical point (75 lines in log)

### 🔬 Ruled Out Issues
- ❌ Not a connection issue (libnbd debug shows successful connect)
- ❌ Not a permissions issue (qemu-nbd serving correctly)
- ❌ Not a size mismatch (correct 109GB reported)
- ❌ Not an SSH tunnel issue (tunnel stable, no errors)
- ❌ Not an NBD server config issue (allowlist/size/readonly all tested)
- ❌ Not a qemu-nbd startup issue (process running, port listening)

---

## 📍 **Current System State**

### SNA (Source - 10.0.100.231)
- **SSH:** vma@10.0.100.231 (Password: `Password1`)
- **Tunnel:** vma-ssh-tunnel.service running (Forward: 10808→10809, Reverse: 9081→8081)
- **migratekit:** `/opt/vma/bin/migratekit` (version v2.22.2)
- **Test Job:** `test-with-allocation-flag` currently hung (PID 314342)
- **Log:** `/tmp/migratekit-test-with-allocation-flag.log` (75 lines, stopped at metadata context)

### SHA (Target - 10.245.246.134)
- **Current Session:** oma_admin@10.245.246.134
- **qemu-nbd:** Running on 127.0.0.1:10809 with `-A` flag
  ```bash
  qemu-nbd -f qcow2 -x test-with-allocation -p 10809 -b 127.0.0.1 -t -A \
    /mnt/sendense-backups/ctx-pgtest1-20251006-203401/disk-0/backup-pgtest1-disk0-20251007-043253.qcow2
  ```
- **QCOW2 File:** `/mnt/sendense-backups/ctx-pgtest1-20251006-203401/disk-0/backup-pgtest1-disk0-20251007-043253.qcow2`
- **QCOW2 Virtual Size:** 109521666048 bytes (102GB)
- **NBD Config:** `/etc/nbd-server/config` (port 10809, localhost only)
- **SSH Config:** `/etc/ssh/sshd_config` (AllowTcpForwarding yes, PermitOpen any for vma_tunnel)

---

## 🔎 **Investigation Findings**

### Code Analysis - Exact Hang Location

**File:** `/home/oma_admin/sendense/source/current/migratekit/internal/vmware_nbdkit/parallel_full_copy.go`

**Execution Flow:**
```go
// Line 33: ✅ EXECUTES
logger.Info("🚀 Starting parallel full copy")

// Line 65: ✅ EXECUTES (function completes successfully)
nbdTarget, err := s.connectToNBDTarget(ctx, path)
if err != nil {
    return fmt.Errorf("failed to connect to target: %w", err)
}

// Line 69: ✅ EXECUTES (defer registered)
defer nbdTarget.Close()

// Line 72: ❌ NEVER EXECUTES
numWorkers := determineWorkerCount(100)

// Line 73: ❌ NEVER LOGGED
logger.Infof("🔧 Using %d parallel workers for full copy", numWorkers)
```

**Inside `connectToNBDTarget()` (lines 476-490):**
```go
// Line 476: ✅ EXECUTES
err = nbdTarget.AddMetaContext("base:allocation")
if err != nil {
    log.WithError(err).Warn("Failed to add metadata context")
} else {
    // Line 480: ✅ LOGGED (last thing we see)
    log.Info("✅ NBD metadata context enabled for sparse optimization")
}

// Line 484: ✅ EXECUTES (libnbd debug shows successful connection)
err = nbdTarget.ConnectTcp(u.Hostname(), u.Port())
if err != nil {
    nbdTarget.Close()
    return nil, fmt.Errorf("failed to connect to target NBD: %w", err)
}

// Line 490: ❌ SHOULD EXECUTE BUT SEEMS NOT TO
return nbdTarget, nil
```

### The Mystery

**Between line 69 (defer) and line 72 (determineWorkerCount) there is NOTHING but blank space.**

Yet the log shows:
1. ✅ "Starting parallel full copy" (line 33)
2. ✅ VMware CBT analysis (lines 35-62)
3. ✅ "NBD metadata context enabled" (line 480 inside connectToNBDTarget)
4. ✅ libnbd debug shows `exportsize: 109521666048 eflags: 0xced` (connection complete)
5. ❌ **NEVER** "Using N parallel workers" (line 73)

**Possible Explanations:**
1. **Goroutine deadlock** - Some background goroutine is blocking
2. **libnbd internal hang** - AddMetaContext + ConnectTcp combination triggers deadlock
3. **qemu-nbd compatibility** - Known issue with qemu-nbd + libnbd metadata contexts
4. **Deferred cleanup blocking** - Something in defer stack is triggering premature hang
5. **Context cancellation** - ctx is being cancelled somewhere unexpected

---

## 🧬 **Key Code Files**

### migratekit Source (SNA - needs recompilation for debug)
- **Main hang location:** `/home/oma_admin/sendense/source/current/migratekit/internal/vmware_nbdkit/parallel_full_copy.go`
  - Lines 64-76: The mystery zone between NBD connection and worker startup
  - Lines 460-491: `connectToNBDTarget()` function (metadata context + connection)
  
- **Worker determination:** `/home/oma_admin/sendense/source/current/migratekit/internal/vmware_nbdkit/parallel_incremental.go`
  - Lines 250-263: `determineWorkerCount()` function (simple, shouldn't block)

- **Build command (on SNA):**
  ```bash
  cd /opt/vma/migratekit-source  # Need to locate actual source directory
  go build -o /opt/vma/bin/migratekit-debug ./cmd/migratekit
  ```

### SHA Backup Workflow (working, not the issue)
- **Backup orchestration:** `/home/oma_admin/sendense/source/current/oma/workflows/backup.go`
- **NBD export creation:** `/home/oma_admin/sendense/source/current/oma/nbd/nbd_config_manager.go`
- **QCOW2 repository:** `/home/oma_admin/sendense/source/current/oma/storage/local_repository.go`

---

## 🎬 **Next Steps - Detailed Instructions**

### Option 1: Add Debug Logging (RECOMMENDED FIRST)

**Goal:** Determine if hang is before/after `connectToNBDTarget` returns

**Steps:**
1. **Locate migratekit source on SNA:**
   ```bash
   ssh vma@10.0.100.231  # Password: Password1
   find /home/vma /opt/vma -type d -name "migratekit*" 2>/dev/null | grep -v ".git"
   # OR check: /opt/vma/migratekit-source or /home/vma/sendense/source/current/migratekit
   ```

2. **Add debug logs to `parallel_full_copy.go`:**
   ```go
   // After line 65 (immediately after connectToNBDTarget):
   logger.Info("🐛 DEBUG: connectToNBDTarget returned successfully")
   
   // After line 69 (immediately after defer):
   logger.Info("🐛 DEBUG: defer registered, about to call determineWorkerCount")
   
   // Before line 72 (immediately before determineWorkerCount):
   logger.Info("🐛 DEBUG: calling determineWorkerCount with value 100")
   
   // Inside connectToNBDTarget() after line 484 (after ConnectTcp):
   log.Info("🐛 DEBUG: ConnectTcp completed, about to return nbdTarget")
   
   // Immediately before line 490 (right before return):
   log.Info("🐛 DEBUG: returning from connectToNBDTarget")
   ```

3. **Rebuild migratekit:**
   ```bash
   cd /path/to/migratekit/source
   go build -o /opt/vma/bin/migratekit-debug ./cmd/migratekit
   chmod +x /opt/vma/bin/migratekit-debug
   ```

4. **Test with debug binary:**
   ```bash
   # On SHA, restart qemu-nbd (already running)
   # On SNA:
   /opt/vma/bin/migratekit-debug migrate \
     --vmware-endpoint quad-vcenter-01.quadris.local \
     --vmware-username administrator@vsphere.local \
     --vmware-password EmyGVoBFesGQc47- \
     --vmware-path /DatabanxDC/vm/pgtest1 \
     --nbd-export-name test-with-allocation \
     --job-id test-debug-logging \
     --debug > /tmp/migratekit-test-debug-logging.log 2>&1 &
   
   # Monitor:
   tail -f /tmp/migratekit-test-debug-logging.log
   ```

5. **Analyze results:**
   - If you see "DEBUG: returning from connectToNBDTarget" → Issue is AFTER function returns (defer/context?)
   - If you DON'T see "DEBUG: ConnectTcp completed" → Issue is INSIDE ConnectTcp (libnbd hang)
   - If you see "DEBUG: defer registered" → Issue is with determineWorkerCount or earlier

### Option 3: Check qemu-nbd + libnbd Compatibility

**Goal:** Determine if this is a known issue

**Research queries:**
```
1. "qemu-nbd libnbd metadata context hang"
2. "libnbd AddMetaContext base:allocation deadlock"
3. "qemu-nbd structured replies hang"
4. "libnbd ConnectTcp blocks after AddMetaContext"
5. Check qemu-nbd version: qemu-nbd --version
6. Check libnbd version: pkg-config --modversion libnbd
```

**Key questions:**
- Does qemu-nbd require specific flags when using metadata contexts?
- Is there a known incompatibility between qemu-nbd QCOW2 exports and libnbd metadata contexts?
- Does libnbd require specific initialization order for metadata contexts?

### Option 2: Skip Metadata Context (LAST RESORT)

**Why dangerous:** Metadata context enables sparse optimization - skipping it means:
- ❌ Every zero block gets copied (massive waste)
- ❌ 102GB full copy instead of 36GB sparse copy
- ❌ 3x longer backup times
- ❌ Defeats the entire purpose of CBT + QCOW2 sparse backups

**Only do this if:**
1. Debug logging shows hang is specifically in AddMetaContext
2. Research confirms qemu-nbd incompatibility
3. We accept 3x performance penalty for initial testing

---

## 🛠️ **Quick Recovery Commands**

### Kill Hung Test Job (SNA)
```bash
ssh vma@10.0.100.231
sudo pkill -f "migratekit.*test-with-allocation"
rm /tmp/migratekit-test-with-allocation-flag.log
```

### Restart qemu-nbd (SHA)
```bash
sudo pkill qemu-nbd
sudo qemu-nbd -f qcow2 -x test-export -p 10809 -b 127.0.0.1 -t -A \
  /mnt/sendense-backups/ctx-pgtest1-20251006-203401/disk-0/backup-pgtest1-disk0-20251007-043253.qcow2 &
```

### Check SSH Tunnel Status (SNA)
```bash
ssh vma@10.0.100.231
systemctl status vma-ssh-tunnel.service
ss -tlnp | grep 10808  # Should show stunnel listening
```

### Test NBD Connection Manually (SHA)
```bash
# Install nbdinfo if not present:
apt-get install -y libnbd-bin

# Query export:
nbdinfo --map nbd://localhost:10809/test-with-allocation
```

---

## 📚 **Context Documents**

### Related Files
- **Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-06-backup-api-integration.md`
- **Architecture:** `/home/oma_admin/sendense/VM_DISKS_ARCHITECTURE_ASSESSMENT.md`
- **Session Progress:** `/home/oma_admin/sendense/BACKUP-API-SESSION-PROGRESS.md`

### Completed Tasks
- ✅ vm_disks table populated at discovery time
- ✅ Backup API endpoints RESTful and functional
- ✅ Backend workflow FK constraint bug fixed
- ✅ 500GB volume repository configured
- ✅ NBD server installed and configured
- ✅ SSH tunnel operational
- ✅ qemu-nbd successfully exports QCOW2

### Blocked Tasks
- ❌ Test backup start → **BLOCKED** by migratekit hang
- ❌ Test backup details endpoint → **BLOCKED**
- ❌ Test backup chain endpoint → **BLOCKED**
- ❌ Test backup delete endpoint → **BLOCKED**
- ❌ E2E integration test → **BLOCKED**

---

## 🎯 **Success Criteria**

**Immediate Goal:** Determine WHERE migratekit hangs (inside ConnectTcp, after return, or in determineWorkerCount)

**Resolution Path:**
1. Add debug logging → Pinpoint exact hang location
2. Research compatibility → Check for known issues
3. Apply fix (code patch, config change, or workaround)
4. Verify data transfer starts (see "Using N workers" log)
5. Complete end-to-end backup test

**Victory Condition:** See migratekit log progress beyond "metadata context enabled" to worker startup and actual data transfer.

---

## 🔥 **Critical Notes**

1. **DO NOT skip metadata context testing** - This defeats sparse optimization
2. **DO NOT modify production migratekit** - Use `-debug` suffix for test binaries
3. **DO verify SSH tunnel health** - Restart vma-ssh-tunnel.service if needed
4. **DO check qemu-nbd logs** - May have errors we haven't seen
5. **DO consider strace** - Last resort for syscall-level debugging

---

## 🤝 **Handover Complete**

**Next session should:**
1. Start with Option 1 (debug logging)
2. Build test binary on SNA
3. Run test and analyze log output
4. Proceed to Option 3 (research) based on findings
5. Report back with detailed results

**Key insight:** The hang is precisely between "metadata context enabled" (line 480) and "Using N workers" (line 73), with only ~10 lines of simple code between them. The mystery is WHERE in those 10 lines the hang occurs.

Good hunting! 🎯
