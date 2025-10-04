# OMA Deployment Package - Quick Start Guide

**Version**: v2.33.0-health-monitor  
**Updated**: October 3, 2025  
**Status**: ✅ **Production Ready**  

---

## 📦 **PACKAGE CONTENTS**

```
oma-deployment-package/
├── binaries/
│   └── oma-api                    # Latest OMA API binary (v2.33.0)
├── migrations/
│   └── 20251003160000_add_operation_summary.up.sql
├── scripts/
│   ├── run-migrations.sh          # Database migration runner
│   └── inject-virtio-drivers.sh   # VirtIO injection script
├── configs/
│   └── (service files, configs)
├── database/
│   └── (initial schema if needed)
└── DEPLOYMENT_README.md           # This file
```

---

## 🚀 **QUICK DEPLOYMENT**

### **Option 1: Full Automated Deployment**

```bash
# 1. Copy package to server
scp -r oma-deployment-package oma_admin@<server_ip>:/tmp/

# 2. SSH to server
ssh oma_admin@<server_ip>

# 3. Run deployment
cd /tmp/oma-deployment-package

# 4. Run database migrations
sudo bash scripts/run-migrations.sh

# 5. Deploy binary
sudo cp binaries/oma-api /opt/migratekit/bin/oma-api-v2.33.0-health-monitor
sudo chmod +x /opt/migratekit/bin/oma-api-v2.33.0-health-monitor

# 6. Backup current
sudo cp /opt/migratekit/bin/oma-api /opt/migratekit/bin/oma-api.backup-$(date +%Y%m%d-%H%M%S)

# 7. Update symlink
sudo ln -sf /opt/migratekit/bin/oma-api-v2.33.0-health-monitor /opt/migratekit/bin/oma-api

# 8. Restart service
sudo systemctl restart oma-api

# 9. Verify
sudo systemctl status oma-api
curl http://localhost:8082/health
```

---

### **Option 2: Step-by-Step Deployment**

#### **Step 1: Database Migrations**

```bash
cd /tmp/oma-deployment-package
sudo bash scripts/run-migrations.sh
```

**Expected Output**:
```
🔄 Running OMA database migrations...
   Migration directory: ./migrations
   Database: migratekit_oma@localhost
✅ Database connection verified
  📥 Applying: 20251003160000_add_operation_summary
  ✅ Applied: 20251003160000_add_operation_summary

✅ Database migrations complete:
   Total: 1
   Applied: 1
   Skipped: 0
```

**If already applied**:
```
  ⏭️  Skipping (already applied): 20251003160000_add_operation_summary
```

---

#### **Step 2: Deploy Binary**

```bash
# Copy binary to deployment location
sudo cp binaries/oma-api /opt/migratekit/bin/oma-api-v2.33.0-health-monitor
sudo chmod +x /opt/migratekit/bin/oma-api-v2.33.0-health-monitor

# Backup current binary
sudo cp /opt/migratekit/bin/oma-api /opt/migratekit/bin/oma-api.backup-$(date +%Y%m%d-%H%M%S)

# Update symlink
sudo ln -sf /opt/migratekit/bin/oma-api-v2.33.0-health-monitor /opt/migratekit/bin/oma-api
```

---

#### **Step 3: Restart Service**

```bash
sudo systemctl restart oma-api
```

**Wait 3-5 seconds** for startup to complete.

---

#### **Step 4: Verification**

```bash
# Check service status
sudo systemctl status oma-api

# Check API health
curl http://localhost:8082/health

# Check job recovery logs
sudo journalctl -u oma-api --since "2 minutes ago" | grep "job recovery"

# Check health monitor logs
sudo journalctl -u oma-api --since "2 minutes ago" | grep "Health monitor"

# Verify database schema
mysql -u oma_user -poma_password migratekit_oma -e \
  "SHOW COLUMNS FROM vm_replication_contexts LIKE 'last_operation_summary';"
```

**Expected in Logs**:
```
✅ Job recovery completed successfully
✅ Health monitor started - will check for orphaned jobs every 2 minutes
```

---

## 🆕 **WHAT'S NEW IN v2.33.0**

### **Feature 1: Intelligent Job Recovery**
- Queries VMA before marking jobs as failed
- Automatically restarts polling for running jobs
- Detects completed jobs during downtime
- Smart age-based decisions

### **Feature 2: Continuous Health Monitor**
- Checks every 2 minutes for stale jobs
- Detects jobs that die after OMA startup
- Validates with VMA and recovers automatically
- Prevents jobs stuck in "replicating" forever

### **Feature 3: Failover Visibility Enhancement**
- Error message sanitization (no technical jargon)
- User-friendly step names
- Persistent operation summaries
- Actionable steps for every failure

### **Feature 4: Unified Jobs API**
- Single endpoint for replication + failover + rollback
- Sanitized errors throughout
- Combines all operation types

---

## 🗄️ **DATABASE MIGRATIONS**

### **20251003160000_add_operation_summary**

**Purpose**: Add operation summary storage for failover visibility

**Changes**:
```sql
ALTER TABLE vm_replication_contexts
ADD COLUMN last_operation_summary JSON NULL;
```

**Impact**: 
- Minimal (one column)
- No data migration needed
- Backward compatible
- No performance impact

**Rollback** (if needed):
```sql
ALTER TABLE vm_replication_contexts
DROP COLUMN IF EXISTS last_operation_summary;
```

---

## 🧪 **POST-DEPLOYMENT VALIDATION**

### **Test 1: Service Health**

```bash
curl http://localhost:8082/health
```

**Expected**:
```json
{"status":"healthy","database":"connected"}
```

---

### **Test 2: Job Recovery**

```bash
sudo journalctl -u oma-api --since "5 minutes ago" | grep "job recovery"
```

**Expected**:
```
✅ Job recovery completed successfully
OR
✅ No active jobs found - system is clean
```

---

### **Test 3: Health Monitor**

```bash
sudo journalctl -u oma-api --since "5 minutes ago" | grep "Health monitor"
```

**Expected**:
```
✅ Health monitor started - will check for orphaned jobs every 2 minutes
```

---

### **Test 4: Database Schema**

```bash
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT COUNT(*) as column_exists FROM information_schema.COLUMNS 
   WHERE TABLE_NAME='vm_replication_contexts' 
   AND COLUMN_NAME='last_operation_summary';"
```

**Expected**:
```
column_exists
1
```

---

### **Test 5: Migration Tracking**

```bash
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT * FROM schema_migrations ORDER BY applied_at DESC LIMIT 5;"
```

**Expected**: Shows applied migrations with timestamps

---

## 🔄 **ROLLBACK PROCEDURE**

If issues arise after deployment:

```bash
# 1. Stop service
sudo systemctl stop oma-api

# 2. Restore previous binary
sudo ln -sf /opt/migratekit/bin/oma-api.backup-YYYYMMDD-HHMMSS /opt/migratekit/bin/oma-api

# 3. Restart service
sudo systemctl start oma-api

# 4. Verify
sudo systemctl status oma-api
curl http://localhost:8082/health

# 5. If database issues, rollback migration:
mysql -u oma_user -poma_password migratekit_oma -e \
  "ALTER TABLE vm_replication_contexts DROP COLUMN IF EXISTS last_operation_summary;"
```

---

## 📊 **MIGRATION RUNNER FEATURES**

The `scripts/run-migrations.sh` script provides:

- ✅ **Idempotent**: Safe to run multiple times
- ✅ **Tracking**: Records applied migrations in `schema_migrations` table
- ✅ **Skip Applied**: Automatically skips already-applied migrations
- ✅ **Error Handling**: Ignores harmless "Duplicate column" errors
- ✅ **Logging**: Clear output of what was applied/skipped
- ✅ **Validation**: Tests database connectivity before running

**Environment Variables**:
- `MIGRATION_DIR` - Path to migrations (default: ../migrations)
- `DB_USER` - Database user (default: oma_user)
- `DB_PASS` - Database password (default: oma_password)
- `DB_NAME` - Database name (default: migratekit_oma)
- `DB_HOST` - Database host (default: localhost)

---

## 🎯 **DEPLOYMENT CHECKLIST**

Before deploying:
- [ ] Package contents verified (binaries/, migrations/, scripts/)
- [ ] Database credentials known
- [ ] Current OMA API backed up
- [ ] Maintenance window scheduled (if needed)

During deployment:
- [ ] Migrations run successfully
- [ ] Binary deployed to /opt/migratekit/bin/
- [ ] Symlink updated
- [ ] Service restarted
- [ ] Logs checked for errors

After deployment:
- [ ] Health endpoint responding
- [ ] Job recovery initialized
- [ ] Health monitor started
- [ ] Database schema verified
- [ ] No errors in logs for 5 minutes

---

## 🚨 **TROUBLESHOOTING**

### **Issue: Migration fails with "Duplicate column"**

**Cause**: Migration already applied  
**Solution**: This is expected - migration runner will skip it  
**Action**: No action needed - continue with deployment

---

### **Issue: Health monitor not starting**

**Cause**: VMA services not available  
**Check**:
```bash
sudo journalctl -u oma-api | grep "VMA"
```

**Solution**: Verify VMA connectivity, check environment variables

---

### **Issue: Job recovery finds no jobs**

**Cause**: System is clean (no active jobs)  
**Solution**: This is normal - recovery will activate when jobs exist  
**Action**: No action needed

---

## 📝 **VERSION COMPATIBILITY**

**Minimum Requirements**:
- MariaDB 10.3+ (JSON support)
- OMA API v2.30.0+ (prerequisite features)
- Volume Daemon v1.3.0+ (if using)

**Database Schema**: Requires all previous migrations applied

---

## 📞 **SUPPORT**

**Logs Location**:
```bash
sudo journalctl -u oma-api -f
```

**Database Check**:
```bash
mysql -u oma_user -poma_password migratekit_oma
```

**Binary Version**:
```bash
ls -l /opt/migratekit/bin/oma-api
```

---

**Deployment Package Ready**: ✅  
**Migration Runner**: ✅  
**Documentation**: ✅  
**Tested on 3 servers**: ✅


