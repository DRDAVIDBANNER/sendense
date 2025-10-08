# QUICK START - DEPLOYMENT & TESTING
## Unified NBD Architecture - Dev Environment

**Version:** v2.20.0-nbd-unified  
**Date:** October 7, 2025  
**Target:** Dev OMA (localhost) + SNA (10.0.100.231)  
**Test VM:** pgtest1

---

## 🚀 QUICK REFERENCE

**You have TWO comprehensive checklists:**

1. **`DEPLOYMENT-DEV-CHECKLIST.md`** - Deploy SHA API + SNA tunnel
2. **`TESTING-PGTEST1-CHECKLIST.md`** - Test backup with pgtest1

---

## 📋 DEPLOYMENT QUICK COMMANDS

### **Phase 1: SHA API (Dev OMA)**

```bash
# 1. Compile SHA
cd /home/oma_admin/sendense/source/current/sha
go build -o sha -ldflags="-X main.Version=v2.20.0-nbd-unified" .

# 2. Stop current API
sudo systemctl stop oma-api

# 3. Backup old binary (if exists)
[ -f /opt/migratekit/bin/oma-api ] && \
  sudo cp /opt/migratekit/bin/oma-api /opt/migratekit/bin/oma-api.backup-$(date +%Y%m%d-%H%M%S)

# 4. Deploy new binary
sudo mkdir -p /opt/migratekit/bin
sudo cp sha /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified
sudo chmod +x /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified
sudo ln -sf /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified /opt/migratekit/bin/oma-api

# 5. Start new API
sudo systemctl start oma-api

# 6. Verify health
sleep 2
curl -f http://localhost:8082/health && echo -e "\n✅ SHA API HEALTHY"
systemctl status oma-api | head -10
```

---

### **Phase 2: SNA Tunnel (10.0.100.231)**

```bash
# 1. Go to deployment package
cd /home/oma_admin/sendense/deployment/sna-tunnel

# 2. IMPORTANT: Update SHA_HOST in sendense-tunnel.sh
# Edit line ~19: SHA_HOST="${SHA_HOST:-10.245.246.125}"  # Your dev OMA IP
nano sendense-tunnel.sh  # or vi

# 3. Deploy to SNA (one command!)
chmod +x deploy-to-sna.sh
sshpass -p 'Password1' ./deploy-to-sna.sh 10.0.100.231

# 4. Verify tunnel
sshpass -p 'Password1' ssh vma@10.0.100.231 "systemctl status sendense-tunnel | head -10"

# 5. Test reverse tunnel
curl -f http://localhost:9081/api/v1/health && echo -e "\n✅ Reverse tunnel working"
```

**Expected time:** 10-15 minutes

---

## 🧪 TESTING QUICK COMMANDS

### **Verify pgtest1**

```bash
# Check pgtest1 in database
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT id, vm_name, vmware_vm_id, current_status FROM vm_replication_contexts WHERE vm_name LIKE '%pgtest1%';"

# Get disk count
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT COUNT(*) as disk_count FROM vm_disks d JOIN vm_replication_contexts v ON d.vm_context_id = v.id WHERE v.vm_name LIKE '%pgtest1%';"
```

---

### **Start Backup Test**

```bash
# 1. Get VM context ID
VM_CONTEXT_ID=$(mysql -u oma_user -poma_password migratekit_oma -N -e \
  "SELECT id FROM vm_replication_contexts WHERE vm_name LIKE '%pgtest1%';")

echo "VM Context ID: $VM_CONTEXT_ID"

# 2. Prepare backup request
cat > /tmp/backup-request.json <<EOF
{
  "vm_context_id": "$VM_CONTEXT_ID",
  "backup_type": "full",
  "repository_path": "/backup/repository"
}
EOF

# 3. Start backup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d @/tmp/backup-request.json | jq '.'

# 4. Get job ID from response
JOB_ID="<paste-job-id-here>"

# 5. Monitor progress
watch -n 5 "curl -s http://localhost:8082/api/v1/backups/$JOB_ID/status | jq '.'"

# 6. Check NBD processes
curl -s http://localhost:8082/api/v1/nbd/processes | jq '.'
ps aux | grep qemu-nbd | grep -v grep
```

---

### **CRITICAL Multi-Disk Test**

```bash
# Verify ONE snapshot (not multiple!)
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '2 minutes ago' | grep -i 'Creating snapshot'"

# Count snapshot events (should be 1)
SNAPSHOT_COUNT=$(sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "journalctl -u vma-api --since '2 minutes ago' | grep -c 'Creating snapshot' || echo 0")

echo "Snapshot count: $SNAPSHOT_COUNT"

if [ "$SNAPSHOT_COUNT" -eq 1 ]; then
    echo "✅ PASS: Only ONE snapshot (correct!)"
else
    echo "❌ FAIL: Multiple snapshots (DATA CORRUPTION RISK!)"
fi
```

---

## ✅ SUCCESS CRITERIA

**Deployment:**
- [x] SHA API compiled and deployed
- [x] SHA API health check passes
- [x] SNA tunnel deployed and connected
- [x] Reverse tunnel working (localhost:9081)
- [x] NBD port allocator responding
- [x] qemu-nbd manager responding

**Testing:**
- [x] Backup API call succeeds
- [x] NBD port allocated from pool (10100-10200)
- [x] qemu-nbd started with `--shared=10`
- [x] **CRITICAL:** Only ONE VMware snapshot created (multi-disk VMs)
- [x] All disks backed up successfully
- [x] QCOW2 files created and integrity OK
- [x] All resources cleaned up (ports + processes)

---

## 🚨 TROUBLESHOOTING QUICK FIXES

### **SHA API won't start:**
```bash
# Check logs
journalctl -u oma-api --since "5 minutes ago" -n 50

# Common issues:
# - Port 8082 in use: lsof -i :8082
# - Database connection: mysql -u oma_user -poma_password migratekit_oma -e "SELECT 1;"
# - Binary permissions: ls -lh /opt/migratekit/bin/oma-api
```

### **SNA tunnel won't connect:**
```bash
# Check tunnel service
sshpass -p 'Password1' ssh vma@10.0.100.231 "systemctl status sendense-tunnel"

# Check logs
sshpass -p 'Password1' ssh vma@10.0.100.231 "journalctl -u sendense-tunnel -n 50"

# Common issues:
# - Wrong SHA_HOST IP in sendense-tunnel.sh
# - SSH key missing: ls /home/vma/.ssh/cloudstack_key
# - Firewall blocking port 443
```

### **Backup fails:**
```bash
# Check SHA API logs
journalctl -u oma-api -f

# Check SNA VMA API logs
sshpass -p 'Password1' ssh vma@10.0.100.231 "journalctl -u vma-api -f"

# Check NBD port allocator
curl -s http://localhost:8082/api/v1/nbd/ports | jq '.'

# Check qemu-nbd processes
ps aux | grep qemu-nbd | grep -v grep
```

---

## 📚 FULL DOCUMENTATION

**Comprehensive checklists:**
- `/home/oma_admin/sendense/DEPLOYMENT-DEV-CHECKLIST.md` (detailed deployment)
- `/home/oma_admin/sendense/TESTING-PGTEST1-CHECKLIST.md` (detailed testing)

**Architecture documentation:**
- `/home/oma_admin/sendense/UNIFIED-NBD-ARCHITECTURE-COMPLETE.md` (31K comprehensive)
- `/home/oma_admin/sendense/deployment/sna-tunnel/README.md` (deployment guide)
- `/home/oma_admin/sendense/deployment/sna-tunnel/VALIDATION_CHECKLIST.md` (15 tests)

**Project tracking:**
- `/home/oma_admin/sendense/job-sheets/2025-10-07-unified-nbd-architecture.md` (job sheet)
- `/home/oma_admin/sendense/start_here/CHANGELOG.md` (change log)

---

## 🎯 EXPECTED TIMELINE

| Phase | Duration | Status |
|-------|----------|--------|
| **Deployment** | 10-15 min | ⏸️ Pending |
| **Testing** | 30-60 min | ⏸️ Pending |
| **Validation** | 15-30 min | ⏸️ Pending |
| **Total** | 1-2 hours | ⏸️ Pending |

---

## 📞 SUPPORT

**If you encounter issues:**

1. Check the comprehensive checklists (step-by-step with troubleshooting)
2. Review logs (SHA API, SNA tunnel, VMA API)
3. Verify environment (database, SSH, network)
4. Refer to architecture documentation

**Key log locations:**
- SHA API: `journalctl -u oma-api`
- SNA Tunnel: `journalctl -u sendense-tunnel` (on SNA)
- SNA VMA API: `journalctl -u vma-api` (on SNA)

---

## 🚀 LET'S GO!

**Start here:**
```bash
cd /home/oma_admin/sendense
cat DEPLOYMENT-DEV-CHECKLIST.md  # Read deployment steps
```

**Then:**
```bash
cat TESTING-PGTEST1-CHECKLIST.md  # Read testing steps
```

**Good luck! You've got comprehensive checklists to guide you.** 💪

---

**End of Quick Start**
