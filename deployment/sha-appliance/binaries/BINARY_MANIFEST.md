# OMA Deployment Package - Binary Manifest

**Last Updated**: October 3, 2025 22:04 BST  
**Package Version**: v2.33.1  
**Status**: ✅ **Production Ready**

---

## 📦 **INCLUDED BINARIES**

### **oma-api** ✅ **UPDATED**

**Binary**: `oma-api`  
**Version**: v2.33.1-current-job-clear-fix  
**Size**: 33M (33,558,528 bytes)  
**Build Date**: October 3, 2025 22:01 BST  
**Source**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/cmd`  
**Target Location**: `/opt/migratekit/bin/oma-api`  
**Service**: `oma-api.service`  

**Key Features in v2.33.1**:
- ✅ **Intelligent Job Recovery** - Queries VMA before marking jobs as failed
- ✅ **Continuous Health Monitor** - Detects stale jobs every 2 minutes
- ✅ **Error Sanitization** - Converts technical errors to user-friendly messages
- ✅ **Unified Jobs API** - Single endpoint for replication + failover + rollback
- ✅ **Operation Summary Storage** - Persistent visibility of failures
- ✅ **Current Job Clear Fix** - Clears current_job_id when marking jobs as failed (allows new jobs)

**What's New in v2.33.1**:
- 🆕 **Critical Fix**: Clears `current_job_id` from VM context when job fails
- 🆕 Previously failed jobs would block new operations with "job already in progress"
- 🆕 Now failed jobs properly release the VM for new operations

**Version History**:
- v2.33.1 (Oct 3, 22:01) - Current job clear fix
- v2.33.0 (Oct 3, 18:13) - Health monitor
- v2.32.0 (Oct 3, 17:18) - Unified jobs API
- v2.31.0 (Oct 3, 17:18) - Failover visibility
- v2.30.1 (Oct 3, 14:25) - Job recovery enhancement

---

### **volume-daemon**

**Binary**: `volume-daemon`  
**Version**: v1.3.2-persistent-naming-fixed  
**Size**: 15M (14,885,875 bytes)  
**Build Date**: October 1, 2025 18:19  
**Target Location**: `/usr/local/bin/volume-daemon`  
**Service**: `volume-daemon.service`  

**Key Features**:
- ✅ Persistent device naming with device mapper symlinks
- ✅ NBD memory synchronization
- ✅ Dynamic OMA VM ID from database
- ✅ Volume lifecycle management
- ✅ Device correlation with CloudStack

---

## 🗄️ **DATABASE MIGRATIONS**

### **Included Migrations**:

**20251003160000_add_operation_summary.up.sql**:
- Purpose: Add operation summary storage for failover visibility
- Schema: Adds `last_operation_summary` JSON column to `vm_replication_contexts`
- Impact: Minimal (one column), backward compatible
- Required for: v2.31.0+

---

## 📜 **SCRIPTS INCLUDED**

### **run-migrations.sh** ✅

**Purpose**: Automated database migration runner  
**Features**:
- Idempotent (safe to run multiple times)
- Tracks applied migrations in `schema_migrations` table
- Skips already-applied migrations
- Validates database connectivity
- Clear logging

**Usage**:
```bash
cd oma-deployment-package
sudo bash scripts/run-migrations.sh
```

### **inject-virtio-drivers.sh** ✅

**Purpose**: VirtIO driver injection for Windows VM compatibility  
**Location**: Deployed to `/opt/migratekit/bin/`

---

## 🚀 **DEPLOYMENT PROCEDURE**

### **Complete Deployment** (New Server):

```bash
# 1. Copy package to server
scp -r oma-deployment-package server:/tmp/

# 2. SSH to server
ssh server

# 3. Run migrations
cd /tmp/oma-deployment-package
sudo bash scripts/run-migrations.sh

# 4. Deploy OMA API
sudo cp binaries/oma-api /opt/migratekit/bin/oma-api-v2.33.1-current-job-clear-fix
sudo chmod +x /opt/migratekit/bin/oma-api-v2.33.1-current-job-clear-fix
sudo ln -sf /opt/migratekit/bin/oma-api-v2.33.1-current-job-clear-fix /opt/migratekit/bin/oma-api

# 5. Deploy Volume Daemon
sudo cp binaries/volume-daemon /usr/local/bin/volume-daemon
sudo chmod +x /usr/local/bin/volume-daemon

# 6. Restart services
sudo systemctl restart oma-api
sudo systemctl restart volume-daemon

# 7. Verify
curl http://localhost:8082/health
sudo journalctl -u oma-api --since "1 minute ago" | grep "Health monitor"
```

---

## ✅ **TESTED ON SERVERS**

**Deployment Verified**:
- ✅ 10.245.246.147 - Running v2.33.1
- ✅ 10.245.246.148 - Running v2.33.1
- ✅ 10.246.5.153 - Running v2.33.1

**Features Verified**:
- ✅ Job recovery with VMA validation working
- ✅ Health monitor catching stale jobs (tested with QUAD-DCMGMT-01)
- ✅ Error sanitization working (no "virt-v2v" or "VirtIO" in errors)
- ✅ Unified jobs API returning sanitized data
- ✅ Current job cleared on failure (allows new jobs to start)

---

## 🔒 **CHECKSUMS**

```bash
# Verify binary integrity after transfer
md5sum binaries/oma-api
# Should match on both source and destination
```

---

## 📊 **WHAT'S NEW IN THIS RELEASE**

### **🔧 Job Recovery System**
**Problem Solved**: Jobs stuck in "replicating" after OMA restart  
**Solution**: Intelligent recovery with VMA validation

**Features**:
- Queries VMA for actual job status
- Restarts polling for running jobs
- Detects completed jobs during downtime
- Age-based decisions for VMA unreachable

### **🏥 Health Monitor**
**Problem Solved**: Jobs that die after OMA startup remain stuck  
**Solution**: Continuous monitoring every 2 minutes

**Features**:
- Detects jobs with stale polling (>30 seconds)
- Validates with VMA
- Restarts polling or marks as failed
- Prevents indefinite stuck jobs

### **🎨 Failover Visibility**
**Problem Solved**: Technical error messages, jobs disappear from view  
**Solution**: Error sanitization + persistent summaries

**Features**:
- No "virt-v2v", "VirtIO", device paths in errors
- User-friendly step names
- Actionable guidance for every failure
- Failed operations visible indefinitely

### **🔗 Unified Jobs API**
**Problem Solved**: Failover/rollback jobs separate from replications  
**Solution**: Single API for all operation types

**Features**:
- Combines replication + failover + rollback
- Sorted by time, sanitized errors
- Consistent UX for all job types

### **🚨 Critical Fixes**
- ✅ Clears `current_job_id` when job fails (allows new operations)
- ✅ Health monitor catches jobs that die after startup
- ✅ VMA validation prevents false failures
- ✅ Persistent error summaries for GUI visibility

---

## 📋 **DEPLOYMENT CHECKLIST**

Before deploying:
- [ ] Backup current binary
- [ ] Database accessible (oma_user/oma_password)
- [ ] MariaDB 10.3+ (JSON support)

During deployment:
- [ ] Run migrations first (`bash scripts/run-migrations.sh`)
- [ ] Deploy binary to /opt/migratekit/bin/
- [ ] Update symlink
- [ ] Restart service

After deployment:
- [ ] Health endpoint responds (curl http://localhost:8082/health)
- [ ] Logs show "Health monitor started"
- [ ] Logs show "Job recovery completed"
- [ ] Database column exists (last_operation_summary)

---

## 🎯 **SUPPORT**

**Logs**:
```bash
sudo journalctl -u oma-api -f
```

**Database**:
```bash
mysql -u oma_user -poma_password migratekit_oma
```

**Version Check**:
```bash
ls -l /opt/migratekit/bin/oma-api
```

---

**Package Ready**: ✅  
**Production Validated**: ✅  
**Deployment Tested**: 3 servers  
**Status**: Ready for distribution
