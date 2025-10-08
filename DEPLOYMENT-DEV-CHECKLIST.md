# DEPLOYMENT CHECKLIST - DEV ENVIRONMENT
## Unified NBD Architecture Deployment

**Date:** October 7, 2025  
**Target SHA:** Dev OMA (localhost - oma_admin)  
**Target SNA:** 10.0.100.231 (vma@10.0.100.231, Password: Password1)  
**Test VM:** pgtest1 (already in database)  
**Version:** v2.20.0-nbd-size-param

---

## 🎯 DEPLOYMENT OVERVIEW

**What we're deploying:**
1. ✅ SHA API with NBD services (Phase 2 code)
2. ✅ SNA SSH tunnel infrastructure (Phase 3 code)

**Expected outcome:**
- 101 NBD ports available (10100-10200)
- Auto-reconnecting SSH tunnel
- qemu-nbd process management
- VM-level multi-disk backup support

---

## 📋 PRE-DEPLOYMENT CHECKLIST

### **On SHA (Dev OMA - localhost)**

**Environment Verification:**
```bash
# Verify you're on the correct system
hostname                              # Should be: localhost or dev OMA hostname
whoami                                # Should be: oma_admin
pwd                                   # Should be: /home/oma_admin/sendense
```

- [ ] ✅ Confirmed: On dev OMA as oma_admin

---

**Database Verification:**
```bash
# Check MariaDB is running
systemctl status mariadb | grep "Active:"

# Test database connection
mysql -u oma_user -poma_password migratekit_oma -e "SELECT VERSION();"

# Check pgtest1 exists
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT id, vm_name, vmware_vm_id, current_status FROM vm_replication_contexts WHERE vm_name LIKE '%pgtest1%';"
```

- [ ] ✅ MariaDB running
- [ ] ✅ Database connection successful
- [ ] ✅ pgtest1 found in database
- [ ] **Record pgtest1 details:**
  - `vm_context_id`: _______________
  - `vmware_vm_id`: _______________
  - `current_status`: _______________

---

**Source Code Verification:**
```bash
cd /home/oma_admin/sendense/source/current

# Verify SHA code exists
ls -lh sha/

# Check for compiled binary
ls -lh sha/sha 2>/dev/null || echo "Binary needs compilation"

# Check services exist
ls -lh sha/services/nbd_port_allocator.go
ls -lh sha/services/qemu_nbd_manager.go
ls -lh sha/api/handlers/backup_handlers.go
```

- [ ] ✅ SHA source code present
- [ ] ✅ NBD Port Allocator service exists
- [ ] ✅ qemu-nbd Manager service exists
- [ ] ✅ Backup handlers updated

---

**Deployment Package Verification:**
```bash
cd /home/oma_admin/sendense/deployment/sna-tunnel

# Verify all files exist
ls -lh sendense-tunnel.sh
ls -lh sendense-tunnel.service
ls -lh deploy-to-sna.sh
ls -lh README.md
ls -lh VALIDATION_CHECKLIST.md

# Verify syntax
bash -n sendense-tunnel.sh && echo "✅ Tunnel script: VALID"
bash -n deploy-to-sna.sh && echo "✅ Deploy script: VALID"
```

- [ ] ✅ All 5 deployment files present
- [ ] ✅ Tunnel script syntax valid
- [ ] ✅ Deploy script syntax valid

---

### **On SNA (10.0.100.231)**

**SSH Connectivity Test:**
```bash
# Test SSH with password (using sshpass)
sshpass -p 'Password1' ssh -o StrictHostKeyChecking=no vma@10.0.100.231 "hostname && whoami"

# Expected output:
# <SNA hostname>
# vma
```

- [ ] ✅ SSH connection successful
- [ ] **Record SNA hostname:** _______________

---

**SNA Environment Check:**
```bash
# Check current VMA API status
sshpass -p 'Password1' ssh vma@10.0.100.231 "systemctl status vma-api 2>&1 | head -5"

# Check if old tunnel exists
sshpass -p 'Password1' ssh vma@10.0.100.231 "systemctl status sendense-tunnel 2>&1 | head -5"

# Check /usr/local/bin/
sshpass -p 'Password1' ssh vma@10.0.100.231 "ls -lh /usr/local/bin/sendense* 2>/dev/null || echo 'No existing sendense scripts'"

# Check current SSH keys
sshpass -p 'Password1' ssh vma@10.0.100.231 "ls -lh /home/vma/.ssh/cloudstack_key 2>/dev/null || echo 'SSH key not found'"
```

- [ ] ✅ SNA accessible
- [ ] ✅ VMA API status: _______________
- [ ] ✅ Existing tunnel status: _______________
- [ ] ✅ SSH key exists: Yes / No

---

## 🚀 PHASE 1: SHA API DEPLOYMENT

### **Step 1.1: Compile SHA Binary**

```bash
cd /home/oma_admin/sendense/source/current/sha

# Compile SHA with new NBD services
echo "Building SHA binary..."
go build -o sha -ldflags="-X main.Version=v2.20.0-nbd-unified" . 2>&1 | tee /tmp/sha-build.log

# Check compilation
if [ $? -eq 0 ]; then
    echo "✅ SHA COMPILED SUCCESSFULLY"
    ls -lh sha
    ./sha --version 2>/dev/null || echo "SHA binary ready"
else
    echo "❌ SHA COMPILATION FAILED"
    tail -20 /tmp/sha-build.log
    exit 1
fi
```

**Expected result:** Binary ~30-35MB

- [ ] ✅ SHA compiled successfully
- [ ] **Binary size:** _______________
- [ ] **Compilation time:** _______________

---

### **Step 1.2: Stop Current SHA API**

```bash
# Check current SHA/OMA API status
systemctl status oma-api 2>&1 | grep "Active:" || echo "Service not found"

# Stop current API (if running)
sudo systemctl stop oma-api 2>/dev/null || echo "No service to stop"

# Verify stopped
ps aux | grep -E "(oma-api|sha)" | grep -v grep || echo "No SHA/OMA processes running"
```

- [ ] ✅ Current API stopped
- [ ] **Previous version:** _______________

---

### **Step 1.3: Backup Current Binary**

```bash
# Backup current binary if exists
if [ -f /opt/migratekit/bin/oma-api ]; then
    sudo cp /opt/migratekit/bin/oma-api /opt/migratekit/bin/oma-api.backup-$(date +%Y%m%d-%H%M%S)
    echo "✅ Backed up current binary"
else
    echo "⚠️  No existing binary to backup"
fi

# List backups
ls -lh /opt/migratekit/bin/*.backup* 2>/dev/null || echo "No backups found"
```

- [ ] ✅ Current binary backed up (if existed)

---

### **Step 1.4: Deploy New SHA Binary**

```bash
cd /home/oma_admin/sendense/source/current/sha

# Copy new binary to production location
sudo mkdir -p /opt/migratekit/bin
sudo cp sha /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified
sudo chmod +x /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified

# Create symlink (or replace oma-api)
sudo ln -sf /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified /opt/migratekit/bin/oma-api

# Verify
ls -lh /opt/migratekit/bin/oma-api
ls -lh /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified
```

- [ ] ✅ New binary deployed
- [ ] ✅ Symlink created
- [ ] **Deploy location:** /opt/migratekit/bin/sha-api-v2.20.0-nbd-unified

---

### **Step 1.5: Update Systemd Service (if needed)**

```bash
# Check current service file
cat /etc/systemd/system/oma-api.service 2>/dev/null || echo "Service file not found"

# Verify ExecStart points to correct binary
grep "ExecStart" /etc/systemd/system/oma-api.service 2>/dev/null

# Reload systemd if service file exists
if [ -f /etc/systemd/system/oma-api.service ]; then
    sudo systemctl daemon-reload
    echo "✅ Systemd reloaded"
fi
```

- [ ] ✅ Service file verified
- [ ] ✅ Systemd reloaded (if needed)

---

### **Step 1.6: Start New SHA API**

```bash
# Start SHA API
sudo systemctl start oma-api

# Check status
sleep 2
systemctl status oma-api | head -15

# Check logs for errors
journalctl -u oma-api --since "1 minute ago" -n 50
```

- [ ] ✅ SHA API started
- [ ] ✅ No errors in logs
- [ ] **Service status:** _______________

---

### **Step 1.7: Verify SHA API Health**

```bash
# Test health endpoint
curl -f http://localhost:8082/health 2>/dev/null && echo -e "\n✅ SHA API HEALTHY" || echo "❌ SHA API UNHEALTHY"

# Check NBD services initialized
journalctl -u oma-api --since "1 minute ago" | grep -i "NBD\|port allocator\|qemu"

# Test NBD port allocation endpoint (if available)
curl -s http://localhost:8082/api/v1/nbd/ports 2>/dev/null | jq '.' || echo "Endpoint not available yet"
```

- [ ] ✅ Health check passed
- [ ] ✅ NBD services initialized
- [ ] **Health response:** _______________

---

## 🔌 PHASE 2: SNA TUNNEL DEPLOYMENT

### **Step 2.1: Review Deployment Script**

```bash
cd /home/oma_admin/sendense/deployment/sna-tunnel

# Review what the script will do
cat deploy-to-sna.sh | grep -A 5 "^deploy_"

# Review tunnel configuration
grep -E "^SHA_HOST|^SHA_PORT|^NBD_PORT" sendense-tunnel.sh
```

- [ ] ✅ Reviewed deployment script
- [ ] ✅ Reviewed tunnel configuration

---

### **Step 2.2: Customize Tunnel Configuration**

**Edit sendense-tunnel.sh if needed:**
```bash
# Check current SHA_HOST setting
grep "SHA_HOST=" sendense-tunnel.sh

# IMPORTANT: Update SHA_HOST to dev OMA IP
# Default: sha.sendense.io
# Change to: 10.245.246.134 (this dev OMA server)

# To edit:
nano sendense-tunnel.sh
# or
vi sendense-tunnel.sh

# Find line ~19:
# SHA_HOST="${SHA_HOST:-sha.sendense.io}"
# 
# Change to:
# SHA_HOST="${SHA_HOST:-10.245.246.134}"  # Dev OMA IP
```

- [ ] ✅ SHA_HOST updated to 10.245.246.134
- [ ] **SHA_HOST value:** 10.245.246.134
- [ ] **SHA_PORT value:** 443 (default)

---

### **Step 2.3: Deploy to SNA (Automated)**

```bash
cd /home/oma_admin/sendense/deployment/sna-tunnel

# Make deploy script executable
chmod +x deploy-to-sna.sh

# Deploy using automated script
# NOTE: Script will prompt for password or use sshpass
sshpass -p 'Password1' ./deploy-to-sna.sh 10.0.100.231

# OR if you prefer manual password entry:
# ./deploy-to-sna.sh 10.0.100.231
```

**What this does:**
1. Validates local files
2. Tests SSH connectivity
3. Transfers files to SNA
4. Installs script to `/usr/local/bin/`
5. Installs systemd service
6. Enables auto-start
7. Starts service
8. Verifies connectivity

**Expected output:**
```
[INFO] Starting deployment to SNA: 10.0.100.231
[OK] SSH connectivity confirmed
[OK] Files transferred
[OK] Script installed to /usr/local/bin/sendense-tunnel.sh
[OK] Systemd service installed
[OK] Service enabled for auto-start
[OK] Service started successfully
[OK] Tunnel connectivity verified
[SUCCESS] Deployment completed successfully!
```

- [ ] ✅ Deployment script ran successfully
- [ ] ✅ No errors during deployment
- [ ] **Deployment time:** _______________

---

### **Step 2.4: Verify Tunnel Service on SNA**

```bash
# Check service status
sshpass -p 'Password1' ssh vma@10.0.100.231 "systemctl status sendense-tunnel | head -15"

# Check if tunnel is connected
sshpass -p 'Password1' ssh vma@10.0.100.231 "ps aux | grep ssh | grep sendense | head -3"

# Check logs
sshpass -p 'Password1' ssh vma@10.0.100.231 "journalctl -u sendense-tunnel --since '2 minutes ago' -n 50"
```

- [ ] ✅ Service running
- [ ] ✅ SSH tunnel connected
- [ ] ✅ No errors in logs
- [ ] **Service status:** _______________

---

### **Step 2.5: Verify Port Forwards**

```bash
# On SNA: Check NBD ports are listening (forwarded)
sshpass -p 'Password1' ssh vma@10.0.100.231 "netstat -an | grep :10100 | head -3 || ss -an | grep :10100 | head -3"

# Check reverse tunnel (9081 → 8081)
sshpass -p 'Password1' ssh vma@10.0.100.231 "netstat -an | grep :8081 | head -3 || ss -an | grep :8081 | head -3"

# From SHA: Test forward tunnel to SNA API
curl -f http://localhost:9081/api/v1/health 2>/dev/null && echo "✅ Reverse tunnel working" || echo "❌ Reverse tunnel NOT working"
```

- [ ] ✅ NBD ports forwarded (10100-10200)
- [ ] ✅ SHA API port forwarded (8082)
- [ ] ✅ Reverse tunnel working (9081 → 8081)

---

## 🧪 PHASE 3: POST-DEPLOYMENT VERIFICATION

### **Step 3.1: Verify NBD Port Allocator**

```bash
# Check NBD port allocator status
curl -s http://localhost:8082/api/v1/nbd/ports 2>/dev/null | jq '.'

# Expected: Empty list or current allocations
# {
#   "allocated_ports": [],
#   "available_count": 101,
#   "total_ports": 101
# }
```

- [ ] ✅ NBD port allocator responding
- [ ] **Available ports:** _______________

---

### **Step 3.2: Verify qemu-nbd Manager**

```bash
# Check qemu-nbd manager status
curl -s http://localhost:8082/api/v1/nbd/processes 2>/dev/null | jq '.'

# Expected: Empty list or current processes
# {
#   "processes": [],
#   "process_count": 0
# }
```

- [ ] ✅ qemu-nbd manager responding
- [ ] **Process count:** _______________

---

### **Step 3.3: Verify Database Connectivity**

```bash
# Test SHA API can connect to database
mysql -u oma_user -poma_password migratekit_oma -e "SHOW TABLES;" | head -10

# Check vm_disks table (for multi-disk support)
mysql -u oma_user -poma_password migratekit_oma -e "DESCRIBE vm_disks;"
```

- [ ] ✅ Database accessible
- [ ] ✅ Tables exist
- [ ] ✅ vm_disks table present

---

### **Step 3.4: Verify Repository Storage**

```bash
# Check backup repository location
df -h /backup 2>/dev/null || df -h /var/lib/sendense 2>/dev/null || df -h /

# Create test directory for backups if needed
sudo mkdir -p /backup/repository
sudo chown oma_admin:oma_admin /backup/repository
ls -lhd /backup/repository
```

- [ ] ✅ Repository location identified
- [ ] **Repository path:** _______________
- [ ] **Available space:** _______________

---

## ✅ DEPLOYMENT COMPLETE CHECKLIST

### **SHA (Dev OMA)**
- [ ] ✅ SHA API binary compiled (v2.20.0-nbd-unified)
- [ ] ✅ SHA API deployed to /opt/migratekit/bin/
- [ ] ✅ SHA API service started
- [ ] ✅ Health check passing
- [ ] ✅ NBD Port Allocator service initialized
- [ ] ✅ qemu-nbd Manager service initialized
- [ ] ✅ Database connectivity confirmed

### **SNA (10.0.100.231)**
- [ ] ✅ SSH tunnel deployed
- [ ] ✅ Systemd service installed
- [ ] ✅ Service started and enabled
- [ ] ✅ 101 NBD ports forwarded (10100-10200)
- [ ] ✅ SHA API port forwarded (8082)
- [ ] ✅ Reverse tunnel working (9081 → 8081)
- [ ] ✅ No errors in tunnel logs

### **Integration**
- [ ] ✅ SHA can reach SNA API (reverse tunnel)
- [ ] ✅ Port allocator responding
- [ ] ✅ qemu-nbd manager responding
- [ ] ✅ Database tables ready
- [ ] ✅ Repository storage ready

---

## 🚨 ROLLBACK PROCEDURE (IF NEEDED)

### **If deployment fails:**

**Rollback SHA API:**
```bash
# Stop new API
sudo systemctl stop oma-api

# Restore old binary
BACKUP=$(ls -t /opt/migratekit/bin/*.backup* | head -1)
if [ -n "$BACKUP" ]; then
    sudo cp "$BACKUP" /opt/migratekit/bin/oma-api
    echo "✅ Restored from: $BACKUP"
fi

# Start old API
sudo systemctl start oma-api
systemctl status oma-api
```

**Rollback SNA Tunnel:**
```bash
# Stop and disable service
sshpass -p 'Password1' ssh vma@10.0.100.231 "sudo systemctl stop sendense-tunnel && sudo systemctl disable sendense-tunnel"

# Remove files
sshpass -p 'Password1' ssh vma@10.0.100.231 "sudo rm /usr/local/bin/sendense-tunnel.sh && sudo rm /etc/systemd/system/sendense-tunnel.service"

# Reload systemd
sshpass -p 'Password1' ssh vma@10.0.100.231 "sudo systemctl daemon-reload"
```

---

## 📊 DEPLOYMENT SIGN-OFF

**Deployment Date:** _______________  
**Deployed By:** _______________  
**SHA Version:** v2.20.0-nbd-unified  
**Deployment Status:** ✅ SUCCESS / ❌ FAILED  

**Post-Deployment Status:**
- SHA API: _______________
- SNA Tunnel: _______________
- Integration Tests: _______________

**Issues Encountered:**
1. _______________
2. _______________
3. _______________

**Notes:**
_______________________________________________
_______________________________________________
_______________________________________________

**Approved for Testing:** ✅ YES / ❌ NO

**Signature:** _______________ **Date:** _______________

---

**DEPLOYMENT COMPLETE!** 🎉

**Next Step:** Proceed to `TESTING-PGTEST1-CHECKLIST.md` for backup testing.

---

**End of Deployment Checklist**
