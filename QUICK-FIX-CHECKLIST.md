# Quick Fix Checklist - Phase 1 Remediation

**Use this checklist to fix violations and unblock work**

---

## 🔴 CRITICAL (Do First)

### 1. Move Binaries Out of Source Tree (30 min)

```bash
cd /home/oma_admin/sendense

# Move sna-api-server binaries
mv source/current/sna-api-server-* source/builds/

# Move volume-daemon binaries  
mv source/current/volume-daemon/volume-daemon* source/builds/
mv source/current/volume-daemon/test* source/builds/

# Verify nothing left
find source/current -type f -executable -size +1M
# Should return nothing

# Update MANIFEST
cd source/builds
ls -1 > MANIFEST.txt
```

**Verification:** `find source/current -type f -size +1M -executable` returns empty

---

### 2. Debug qemu-nbd Startup (1-2 hours)

**File:** `source/current/sha/services/qemu_nbd_manager.go`

**Check these things:**

```bash
# 1. Verify QCOW2 file exists before qemu-nbd starts
ls -lh /backup/repository/pgtest1-disk-*.qcow2

# 2. Check file permissions
ls -lh /backup/repository/
# Should be writable by user running SHA

# 3. Test qemu-nbd manually
qemu-img create -f qcow2 /tmp/test.qcow2 10G
qemu-nbd -f qcow2 -x test -p 10150 -b 0.0.0.0 --shared=10 -t /tmp/test.qcow2 &
ps aux | grep qemu-nbd
lsof -i :10150
# Should show process running and port listening

# 4. Check qemu-nbd stderr/stdout
# Add to qemu_nbd_manager.go:
cmd.Stderr = os.Stderr
cmd.Stdout = os.Stdout
# Rebuild and check logs
```

**Likely Causes:**
- [ ] QCOW2 file doesn't exist when qemu-nbd starts
- [ ] File permissions wrong
- [ ] Path incorrect
- [ ] qemu-nbd version incompatible

**Fix:** Add proper file creation + verification before qemu-nbd start

---

### 3. Implement SNA Backup Endpoint (2-3 hours)

**File:** `source/current/sna/api/server.go`

**Add this endpoint:**

```go
// POST /api/v1/backup/start
func (s *Server) handleBackupStart(w http.ResponseWriter, r *http.Request) {
    var req struct {
        VMName        string `json:"vm_name"`
        VCenterHost   string `json:"vcenter_host"`
        VCenterUser   string `json:"vcenter_user"`
        VCenterPass   string `json:"vcenter_password"`
        VMPath        string `json:"vm_path"`
        NBDTargets    string `json:"nbd_targets"`
        JobID         string `json:"job_id"`
        BackupType    string `json:"backup_type"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    
    // Build sendense-backup-client command
    cmd := exec.Command("/usr/local/bin/sendense-backup-client",
        "migrate",
        "--vmware-endpoint", req.VCenterHost,
        "--vmware-username", req.VCenterUser,
        "--vmware-password", req.VCenterPass,
        "--vmware-path", req.VMPath,
        "--nbd-targets", req.NBDTargets,
        "--job-id", req.JobID,
    )
    
    // Start process
    if err := cmd.Start(); err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    
    // Return success
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": "started",
        "pid": cmd.Process.Pid,
        "job_id": req.JobID,
    })
}

// Add to router (in SetupRoutes or similar)
r.HandleFunc("/api/v1/backup/start", s.handleBackupStart).Methods("POST")
```

**Build and deploy:**

```bash
cd source/current/sna
go build -o ../../source/builds/sna-api-v1.4.2-backup-endpoint cmd/main.go
scp ../../source/builds/sna-api-v1.4.2-backup-endpoint vma@10.0.100.231:/tmp/
ssh vma@10.0.100.231 'sudo mv /tmp/sna-api-v1.4.2-backup-endpoint /usr/local/bin/sna-api && sudo systemctl restart sna-api'
```

**Test:**

```bash
curl -X POST http://localhost:9081/api/v1/backup/start \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"test","vcenter_host":"vcenter.local","vcenter_user":"admin","vcenter_password":"pass","vm_path":"/DC/vm/test","nbd_targets":"2000:nbd://127.0.0.1:10110/test","job_id":"test-123","backup_type":"full"}'
```

---

### 4. Update Status Documents (15 min)

**Files to update:**

1. **HANDOVER-2025-10-07-SENDENSE-BACKUP-CLIENT.md**
   - Line 4: Change "95% Complete" → "60% Complete - BLOCKED"
   - Line 8: Remove "Production Ready" claim
   - Add: "Status: IN PROGRESS - qemu-nbd blocker"

2. **job-sheets/2025-10-07-unified-nbd-architecture.md**
   - Line 8: Change "95% COMPLETE" → "60% COMPLETE - BLOCKED"
   - Line 405: Change "100% COMPLETE" → "80% COMPLETE - qemu blocker"
   - Line 825: Change "100% COMPLETE" → "85% COMPLETE - systemd bug"

3. **Create:** `job-sheets/CURRENT-ACTIVE-WORK.md`
   ```markdown
   # Current Active Work
   
   **Status:** BLOCKED - Critical qemu-nbd issue
   **Active Job Sheet:** 2025-10-07-unified-nbd-architecture.md
   **Blocker:** qemu-nbd processes die immediately after start
   **Next Step:** Debug qemu_nbd_manager.go
   ```

---

## 🟡 MAJOR (Do After Unblocked)

### 5. Refactor Commented Code (2 hours)

**File:** `source/current/sendense-backup-client/main.go`

**Remove lines 331-362 and 373-414**

**Replace with proper separation:**

Option A - Build Tags:
```go
//go:build !openstack
// +build !openstack

// NBD-only backup implementation
func runBackup(ctx context.Context) error {
    // Existing backup code without OpenStack
}
```

Option B - Separate Binary:
- Keep migratekit for migrations (with OpenStack)
- Keep sendense-backup-client for backups (without OpenStack)
- Don't share main.go

**Rebuild:**
```bash
cd source/current/sendense-backup-client
go build -o ../../source/builds/sendense-backup-client-v1.1.0-refactored
```

---

### 6. Update API Documentation (1 hour)

**File:** `source/current/api-documentation/API_REFERENCE.md`

**Add these changes from October 7:**

```markdown
## SHA Backup API Changes (v2.20.3)

### POST /api/v1/backups

**Changes:**
- Added multi-disk support
- Returns `disk_results` array (not single disk_id)
- Returns `nbd_targets_string` field
- Uses VMwareCredentialService for credential lookup

**New Response Format:**
{
  "backup_id": "backup-vm-123",
  "disk_results": [
    {
      "disk_id": 0,
      "nbd_port": 10110,
      "nbd_export_name": "vm-disk0",
      "qcow2_path": "/backup/repository/vm-disk0.qcow2",
      "qemu_nbd_pid": 12345,
      "status": "qemu_started"
    }
  ],
  "nbd_targets_string": "2000:nbd://127.0.0.1:10110/vm-disk0,2001:nbd://127.0.0.1:10111/vm-disk1"
}

### SNA Backup API Changes (v1.4.1)

### POST /api/v1/backup/start

**Flag Changes:**
- `--vcenter` → `--vmware-endpoint`
- `--username` → `--vmware-username`
- `--password` → `--vmware-password`
```

**Update version:**
- Change "v2.7.6" → "v2.20.3"
- Change "Last Updated: October 5" → "October 7"

---

### 7. Fix Naming Consistency (1 hour)

**Search and replace:**

```bash
cd /home/oma_admin/sendense

# Find remaining OMA references
grep -r "oma-api" source/current/ --include="*.go" --include="*.sh" --include="*.md"

# Should use: sendense-hub
# Should use: /usr/local/bin/sendense-hub

# Update deployment paths in docs
find . -name "*.md" -exec sed -i 's|/opt/migratekit/bin/oma-api|/usr/local/bin/sendense-hub|g' {} \;
```

---

### 8. Fix Systemd Services (1 hour)

**File:** `deployment/sna-tunnel/sendense-tunnel.sh`

**Change preflight check (line ~150):**

```bash
# OLD (WRONG):
if ! nc -z "$SHA_HOST" 22; then
    log "ERROR: Cannot reach SHA_HOST on port 22"
    exit 1
fi

# NEW (CORRECT):
if ! nc -z "$SHA_HOST" "$SHA_PORT"; then
    log "ERROR: Cannot reach $SHA_HOST on port $SHA_PORT"
    exit 1
fi
```

**Create SHA systemd service:**

```bash
cat > /etc/systemd/system/sendense-hub.service << 'EOF'
[Unit]
Description=Sendense Hub Appliance API
After=network-online.target mariadb.service
Wants=network-online.target

[Service]
Type=simple
User=oma_admin
WorkingDirectory=/home/oma_admin/sendense
Environment="DB_USER=oma_user"
Environment="DB_PASSWORD=oma_password"
Environment="DB_NAME=migratekit_oma"
ExecStart=/usr/local/bin/sendense-hub
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable sendense-hub
systemctl start sendense-hub
```

---

## 🔵 MODERATE (Do When Time Permits)

### 9. Link to Project Goals (30 min)

**File:** `project-goals/phases/phase-1-vmware-backup.md`

**Update tasks:**

```markdown
### Task 1: Backup Repository Abstraction ✅ COMPLETED
**Status:** Complete (October 7, 2025)
**Job Sheet:** job-sheets/2025-10-07-unified-nbd-architecture.md
**Evidence:** SHA backup API operational with QCOW2 storage

### Task 2: VMware Backup Workflow ⏳ IN PROGRESS
**Status:** 60% complete - BLOCKED by qemu-nbd issue
**Job Sheet:** job-sheets/2025-10-07-unified-nbd-architecture.md  
**Blocker:** qemu-nbd processes exit immediately
**Next:** Debug qemu_nbd_manager.go
```

---

### 10. Production Readiness Checklist (After All Fixes)

**Only claim "Production Ready" after ALL these pass:**

```markdown
## Production Readiness Checklist

**Functional:**
- [ ] One successful backup completes
- [ ] Multi-disk VM backups work
- [ ] Data transfers to QCOW2 files
- [ ] File sizes match VM disk sizes
- [ ] ONE VMware snapshot for multi-disk VMs

**Code Quality:**
- [ ] No binaries in source/current/
- [ ] No commented code blocks >10 lines
- [ ] Linter passes with zero errors
- [ ] Code follows project standards

**Testing:**
- [ ] Unit tests pass (100%)
- [ ] Integration tests pass (100%)
- [ ] End-to-end test succeeds
- [ ] 10+ concurrent backups tested
- [ ] Failure scenarios tested

**Documentation:**
- [ ] API docs current
- [ ] All endpoints documented
- [ ] Examples provided
- [ ] CHANGELOG updated

**Deployment:**
- [ ] Systemd services working
- [ ] Auto-restart functional
- [ ] Health checks implemented
- [ ] Monitoring configured

**Performance:**
- [ ] 3.0+ GiB/s throughput maintained
- [ ] Resource usage acceptable
- [ ] No memory leaks
- [ ] Benchmarks documented

**Security:**
- [ ] Security scan passing
- [ ] No hardcoded credentials
- [ ] SSH tunnel secure
- [ ] Credentials encrypted
```

**Evidence Required:**
- Test results with timestamps
- Performance benchmark results
- Security scan reports
- Deployment verification

---

## ✅ VERIFICATION COMMANDS

**Check binaries moved:**
```bash
find /home/oma_admin/sendense/source/current -type f -executable -size +1M
# Should return nothing
```

**Check qemu-nbd working:**
```bash
ps aux | grep qemu-nbd
lsof -i :10110-10200
# Should show processes and listening ports
```

**Check SNA endpoint exists:**
```bash
curl http://localhost:9081/api/v1/backup/start
# Should NOT return 404
```

**Check end-to-end:**
```bash
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}'
  
# Should return with disk_results array
# Check QCOW2 files created with data
ls -lh /backup/repository/pgtest1-*
```

---

## 📊 PROGRESS TRACKING

Use this to track fixes:

```markdown
## Fix Progress

**Critical (Must Do First):**
- [ ] 1. Binaries moved (30 min)
- [ ] 2. qemu-nbd debugged (1-2 hours)
- [ ] 3. SNA endpoint implemented (2-3 hours)
- [ ] 4. Status updated (15 min)
- [ ] 5. One successful backup (1 hour)

**Major (Do After Unblocked):**
- [ ] 6. Code refactored (2 hours)
- [ ] 7. API docs updated (1 hour)
- [ ] 8. Naming fixed (1 hour)
- [ ] 9. Systemd fixed (1 hour)

**Moderate (When Time Permits):**
- [ ] 10. Project goals linked (30 min)
- [ ] 11. Production checklist (varies)

**Total Time:** ~12-15 hours to complete
```

---

**Use this checklist to work through fixes systematically. Check off items as you complete them.**

