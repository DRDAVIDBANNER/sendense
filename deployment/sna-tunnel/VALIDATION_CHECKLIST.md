# Sendense SSH Tunnel - Validation Checklist

**Version:** 1.0.0  
**Date:** 2025-10-07  
**Phase:** 3 - SNA SSH Tunnel Updates

---

## ✅ Pre-Deployment Validation

### Files Created
- [x] `/deployment/sna-tunnel/sendense-tunnel.sh` - Tunnel management script (180 lines)
- [x] `/deployment/sna-tunnel/sendense-tunnel.service` - Systemd service definition
- [x] `/deployment/sna-tunnel/deploy-to-sna.sh` - Automated deployment script (executable)
- [x] `/deployment/sna-tunnel/README.md` - Comprehensive documentation
- [x] `/deployment/sna-tunnel/VALIDATION_CHECKLIST.md` - This file

### Script Validation
- [x] Bash syntax check passed (`bash -n sendense-tunnel.sh`)
- [x] Deployment script syntax check passed (`bash -n deploy-to-sna.sh`)
- [x] Deployment script made executable (`chmod +x`)

### Configuration Review
- [x] NBD port range: 10100-10200 (101 ports)
- [x] SHA API port: 8082
- [x] SNA API port: 8081
- [x] Reverse tunnel port: 9081
- [x] SSH key path: `/home/vma/.ssh/cloudstack_key`
- [x] Tunnel user: `vma_tunnel`

---

## 🚀 Deployment Validation

### Run Automated Deployment
```bash
cd /home/oma_admin/sendense/deployment/sna-tunnel
./deploy-to-sna.sh
```

### Expected Output
- [ ] Pre-deployment checks pass
- [ ] SSH connectivity verified
- [ ] Files copied to SNA successfully
- [ ] Tunnel script installed to `/usr/local/bin/`
- [ ] Systemd service installed to `/etc/systemd/system/`
- [ ] Service enabled and started
- [ ] Service status shows "active (running)"

### Manual Verification (if automated deployment used)
```bash
# SSH to SNA
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231

# Check installed files
ls -l /usr/local/bin/sendense-tunnel.sh
ls -l /etc/systemd/system/sendense-tunnel.service

# Check service status
sudo systemctl status sendense-tunnel

# Check logs
sudo journalctl -u sendense-tunnel -n 20
```

---

## 🧪 Functional Testing

### Test 1: Service Status
```bash
# On SNA
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231

sudo systemctl status sendense-tunnel
```
**Expected:** Active (running), no errors

**Result:** [ ] PASS [ ] FAIL

---

### Test 2: Log Output
```bash
# On SNA
sudo tail -f /var/log/sendense-tunnel.log
```
**Expected:** 
- "Pre-flight checks passed"
- "Establishing SSH tunnel to..."
- No ERROR messages

**Result:** [ ] PASS [ ] FAIL

---

### Test 3: NBD Port Forwarding
```bash
# On SNA
nc -zv localhost 10105
nc -zv localhost 10150
nc -zv localhost 10200
```
**Expected:** All connections succeed

**Result:** [ ] PASS [ ] FAIL

---

### Test 4: SHA API Forward
```bash
# On SNA
curl -s http://localhost:8082/api/v1/health | jq
```
**Expected:** JSON response from SHA health check

**Result:** [ ] PASS [ ] FAIL

---

### Test 5: Reverse Tunnel (SNA API Access from SHA)
```bash
# On SHA (10.245.246.134)
curl -s http://localhost:9081/api/v1/health | jq
```
**Expected:** JSON response from SNA VMA API

**Result:** [ ] PASS [ ] FAIL

---

### Test 6: Auto-Restart on Failure
```bash
# On SNA
# Kill the tunnel process
sudo pkill -f sendense-tunnel.sh

# Wait 10 seconds
sleep 10

# Check if service restarted
sudo systemctl status sendense-tunnel
```
**Expected:** Service automatically restarted, status "active (running)"

**Result:** [ ] PASS [ ] FAIL

---

### Test 7: Boot Persistence
```bash
# On SNA
# Check if service is enabled
systemctl is-enabled sendense-tunnel
```
**Expected:** "enabled"

**Result:** [ ] PASS [ ] FAIL

---

## 🔧 Integration Testing

### Test 8: End-to-End Backup Flow
```bash
# On SHA
# Start a backup via API
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{
    "vm_name": "test-vm",
    "backup_type": "full",
    "repository_id": "repo-001"
  }'
```

**Expected:**
- SHA allocates NBD ports (e.g., 10105, 10106)
- SHA starts qemu-nbd processes on those ports
- SHA calls SNA VMA API via reverse tunnel (localhost:9081)
- SNA connects to NBD ports via tunnel (localhost:10105, etc.)
- Backup proceeds successfully

**Result:** [ ] PASS [ ] FAIL

---

### Test 9: Concurrent Operations
```bash
# On SHA
# Start multiple backups simultaneously
for i in {1..5}; do
  curl -X POST http://localhost:8082/api/v1/backups \
    -H "Content-Type: application/json" \
    -d "{\"vm_name\":\"test-vm-$i\",\"backup_type\":\"full\",\"repository_id\":\"repo-001\"}" &
done
wait
```

**Expected:**
- All 5 backups start successfully
- Different NBD ports allocated (10105, 10106, 10107, 10108, 10109)
- All qemu-nbd processes running
- All tunneled connections working
- No port conflicts

**Result:** [ ] PASS [ ] FAIL

---

### Test 10: Tunnel Reconnection
```bash
# On SHA
# Block SNA temporarily (simulate network issue)
sudo iptables -A INPUT -s 10.0.100.231 -j DROP

# Wait 30 seconds
sleep 30

# Unblock
sudo iptables -D INPUT -s 10.0.100.231 -j DROP

# On SNA, check if tunnel recovered
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 \
  "sudo journalctl -u sendense-tunnel --since '2 minutes ago' | grep -i reconnect"
```

**Expected:**
- Tunnel detects disconnect
- Automatic reconnection within 5 seconds
- Service remains running

**Result:** [ ] PASS [ ] FAIL

---

## 📊 Performance Validation

### Test 11: Resource Usage
```bash
# On SNA
# Check tunnel process resources
ps aux | grep sendense-tunnel
systemctl show sendense-tunnel --property=MemoryCurrent
```

**Expected:**
- Memory < 50MB
- CPU < 5% (idle)
- No memory leaks

**Result:** [ ] PASS [ ] FAIL

---

### Test 12: Connection Count
```bash
# On SNA
netstat -tuln | grep -E "1010[0-9]|8082|9081" | wc -l
```

**Expected:** 103 listening ports (101 NBD + 1 SHA API + 1 reverse tunnel)

**Result:** [ ] PASS [ ] FAIL

---

## 🔐 Security Validation

### Test 13: SSH Key Permissions
```bash
# On SNA
ls -l /home/vma/.ssh/cloudstack_key
```
**Expected:** `-rw------- (600)` or `-r-------- (400)`

**Result:** [ ] PASS [ ] FAIL

---

### Test 14: Process User
```bash
# On SNA
ps aux | grep sendense-tunnel | grep -v grep
```
**Expected:** Running as user `vma`

**Result:** [ ] PASS [ ] FAIL

---

### Test 15: Systemd Hardening
```bash
# On SNA
systemctl show sendense-tunnel | grep -E "NoNewPrivileges|PrivateTmp|ProtectSystem"
```
**Expected:**
- NoNewPrivileges=yes
- PrivateTmp=yes
- ProtectSystem=strict

**Result:** [ ] PASS [ ] FAIL

---

## 📝 Documentation Validation

### Checklist
- [x] README.md includes all deployment options
- [x] README.md includes troubleshooting section
- [x] README.md includes architecture diagram
- [x] Automated deployment script includes verbose output
- [x] Scripts include inline comments
- [x] Error messages are clear and actionable

---

## ✅ Final Approval

### Phase 3 Tasks Completed
- [x] Task 3.1: Multi-Port Tunnel Script created and validated
- [x] Task 3.2: Systemd Service created and validated
- [x] Task 3.3: Documentation and deployment tools created

### Sign-Off

**Deployment Package Ready:** [ ] YES [ ] NO

**Ready for Production:** [ ] YES [ ] NO

**Approved By:** ___________________

**Date:** ___________________

---

## 🚀 Next Steps

After validation:
1. [ ] Deploy to SNA using automated script
2. [ ] Run all functional tests
3. [ ] Monitor for 24 hours
4. [ ] Move to Phase 4 (Testing & Validation)
5. [ ] Update project documentation

---

**Validation Completed:** $(date)  
**By:** $(whoami)@$(hostname)
