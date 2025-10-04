# OMA Deployment Package Checklist

**Updated**: October 3, 2025  
**Package Location**: `/home/pgrayson/oma-deployment-package`  
**Status**: ✅ **Ready for Distribution**  

---

## ✅ **PACKAGE CONTENTS VERIFIED**

### **Binaries** ✅
- [x] `binaries/oma-api` (v2.33.0-health-monitor, 33M)
- [x] `binaries/volume-daemon` (v1.3.0+, 15M)

### **Database Migrations** ✅
- [x] `migrations/20251003160000_add_operation_summary.up.sql`
- Future migrations will be added here as created

### **Scripts** ✅
- [x] `scripts/run-migrations.sh` - Automated migration runner
- [x] `scripts/inject-virtio-drivers.sh` - VirtIO injection

### **Documentation** ✅
- [x] `DEPLOYMENT_README.md` - Complete deployment guide
- [x] `BINARY_MANIFEST.md` - Binary version tracking

---

## 🎯 **DEPLOYMENT PACKAGE REQUIREMENTS**

### **Before Each Release**

1. **Update Binary**:
   ```bash
   cp /home/pgrayson/migratekit-cloudstack/source/current/oma/cmd/oma-api-vX.X.X \
      /home/pgrayson/oma-deployment-package/binaries/oma-api
   ```

2. **Add New Migrations**:
   ```bash
   cp /home/pgrayson/migratekit-cloudstack/source/current/oma/database/migrations/*.up.sql \
      /home/pgrayson/oma-deployment-package/migrations/
   ```

3. **Update Documentation**:
   - Update DEPLOYMENT_README.md with version number
   - Update BINARY_MANIFEST.md with binary details
   - Document any new features or breaking changes

4. **Test Migration Runner**:
   ```bash
   cd /home/pgrayson/oma-deployment-package
   bash scripts/run-migrations.sh
   # Should show: migrations applied or skipped
   ```

---

## 📋 **CURRENT PACKAGE STATE**

**Binary Version**: v2.33.0-health-monitor  
**Size**: 33M  
**Build Date**: October 3, 2025 18:13  

**Migrations Included**:
1. `20251003160000_add_operation_summary.up.sql` ✅

**Scripts**:
1. `run-migrations.sh` - Database migration automation ✅
2. `inject-virtio-drivers.sh` - VirtIO driver injection ✅

---

## 🚀 **DEPLOYMENT WORKFLOW**

### **Standard Deployment to New Server**

```bash
# 1. Copy entire package
scp -r /home/pgrayson/oma-deployment-package oma_admin@<server>:/tmp/

# 2. SSH to server
ssh oma_admin@<server>

# 3. Run migrations
cd /tmp/oma-deployment-package
sudo bash scripts/run-migrations.sh

# 4. Deploy binary
sudo cp binaries/oma-api /opt/migratekit/bin/oma-api-v2.33.0-health-monitor
sudo chmod +x /opt/migratekit/bin/oma-api-v2.33.0-health-monitor
sudo ln -sf /opt/migratekit/bin/oma-api-v2.33.0-health-monitor /opt/migratekit/bin/oma-api

# 5. Restart
sudo systemctl restart oma-api

# 6. Verify
curl http://localhost:8082/health
sudo journalctl -u oma-api --since "1 minute ago" | grep -E "Health monitor|job recovery"
```

---

## 🧪 **TESTED ON SERVERS**

### **Server: 10.245.246.147**
- [x] Migration applied
- [x] Binary deployed (v2.33.0)
- [x] Health monitor running
- [x] Job recovery operational
- [x] Service healthy

### **Server: 10.245.246.148**
- [x] Migration applied
- [x] Binary deployed (v2.33.0)
- [x] Health monitor running
- [x] Job recovery operational  
- [x] Service healthy

### **Server: 10.246.5.153**
- [x] Migration applied
- [x] Binary deployed (v2.33.0)
- [x] Health monitor running
- [x] Job recovery operational
- [x] Service healthy
- [x] Caught and fixed stale job (QUAD-DCMGMT-01)

---

## 📝 **MAINTENANCE NOTES**

### **Adding New Migrations**

When creating new migrations:

1. Create migration file in source:
   ```
   source/current/oma/database/migrations/YYYYMMDDHHMMSS_description.up.sql
   ```

2. Copy to deployment package:
   ```bash
   cp source/current/oma/database/migrations/YYYYMMDDHHMMSS*.up.sql \
      oma-deployment-package/migrations/
   ```

3. Test migration runner:
   ```bash
   cd oma-deployment-package
   bash scripts/run-migrations.sh
   ```

4. Commit to git:
   ```bash
   git add source/current/oma/database/migrations/
   git commit -m "Add migration: description"
   ```

---

### **Updating Binary**

When releasing new version:

1. Build binary:
   ```bash
   cd source/current/oma/cmd
   go build -o oma-api-vX.X.X-feature-name .
   ```

2. Test locally

3. Copy to deployment package:
   ```bash
   cp oma-api-vX.X.X-feature-name \
      /home/pgrayson/oma-deployment-package/binaries/oma-api
   ```

4. Update version in DEPLOYMENT_README.md

---

## 🔒 **SECURITY NOTES**

### **Migration Runner**
- Requires database credentials
- Runs with elevated privileges (sudo)
- Creates tracking table if needed
- Safe to run multiple times

### **Binary Deployment**
- Requires sudo for /opt/migratekit/bin/
- Previous binary automatically backed up
- Symlink ensures atomic switchover

---

## ✅ **PACKAGE VALIDATION**

To verify package is complete before distribution:

```bash
cd /home/pgrayson/oma-deployment-package

# Check structure
ls -R

# Verify binary executable
file binaries/oma-api | grep "executable"

# Verify migration syntax
mysql -u oma_user -poma_password migratekit_oma --execute "source migrations/20251003160000_add_operation_summary.up.sql" --force --verbose

# Test migration runner
bash scripts/run-migrations.sh
```

**Expected**: All checks pass without errors

---

## 🎯 **CURRENT STATUS**

**Package Version**: v2.33.0-health-monitor  
**Migration Count**: 1  
**Scripts Count**: 2  
**Tested Servers**: 3  
**Production Ready**: YES ✅  

**Last Updated**: October 3, 2025  
**Package Location**: `/home/pgrayson/oma-deployment-package`  
**Ready for Distribution**: YES ✅


