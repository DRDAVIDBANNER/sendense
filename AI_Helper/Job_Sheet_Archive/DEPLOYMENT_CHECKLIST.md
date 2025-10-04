# OMA Deployment Checklist - v6.18.0

**Last Updated**: October 4, 2025  
**Package Version**: v6.18.0  
**Critical Updates**: OSSEA Config Auto-Detection Fix

---

## 📋 **Pre-Deployment Checklist**

### **1. Prerequisites**
- [ ] Target server accessible via SSH
- [ ] Root/sudo access available
- [ ] MariaDB 10.5+ installed and running
- [ ] Ports 8082, 3001, 443 available
- [ ] Minimum 4GB RAM, 20GB disk space
- [ ] Valid OSSEA CloudStack environment

### **2. Backup Current System**
```bash
# Backup binaries
sudo cp /opt/migratekit/bin/oma-api /opt/migratekit/bin/oma-api.backup.$(date +%Y%m%d-%H%M%S)
sudo cp /usr/local/bin/volume-daemon /usr/local/bin/volume-daemon.backup.$(date +%Y%m%d-%H%M%S)

# Backup database
mysqldump -u oma_user -poma_password migratekit_oma > migratekit_oma_backup_$(date +%Y%m%d-%H%M%S).sql

# Backup GUI
sudo tar -czf /opt/migratekit/gui_backup_$(date +%Y%m%d-%H%M%S).tar.gz /opt/migratekit/gui
```

### **3. Package Contents Verification**
- [ ] `binaries/oma-api` (v2.40.5, 33MB)
- [ ] `binaries/volume-daemon` (latest)
- [ ] `database/production-schema.sql` (with ossea_config_id, credential_id)
- [ ] `gui/` directory (with UnifiedOSSEAConfiguration)
- [ ] Documentation files

---

## 🚀 **Deployment Steps**

### **Step 1: Stop Services**
```bash
sudo systemctl stop oma-api
sudo systemctl stop volume-daemon
sudo systemctl stop migration-gui
```

### **Step 2: Deploy Binaries**
```bash
# OMA API
sudo cp binaries/oma-api /opt/migratekit/bin/oma-api
sudo chmod +x /opt/migratekit/bin/oma-api

# Volume Daemon
sudo cp binaries/volume-daemon /usr/local/bin/volume-daemon
sudo chmod +x /usr/local/bin/volume-daemon
```

### **Step 3: Update Database Schema**
```bash
# Apply schema updates (if fresh deployment)
mysql -u oma_user -poma_password migratekit_oma < database/production-schema.sql

# Or manually add missing fields (if upgrading):
mysql -u oma_user -poma_password migratekit_oma << 'EOF'
-- Check if fields exist first
DESCRIBE vm_replication_contexts;

-- Add fields if missing (will error if they exist, that's OK)
ALTER TABLE vm_replication_contexts 
ADD COLUMN ossea_config_id INT NULL,
ADD COLUMN credential_id INT NULL,
ADD COLUMN last_operation_summary TEXT NULL;

-- Update existing VMs to use active config
UPDATE vm_replication_contexts 
SET ossea_config_id = (SELECT id FROM ossea_configs WHERE is_active = 1 LIMIT 1)
WHERE ossea_config_id IS NULL;
EOF
```

### **Step 4: Deploy GUI**
```bash
# Copy GUI files
sudo cp -r gui/* /opt/migratekit/gui/

# Build GUI
cd /opt/migratekit/gui
sudo npm install
sudo npm run build
```

### **Step 5: Start Services**
```bash
sudo systemctl start volume-daemon
sudo systemctl start oma-api
sudo systemctl start migration-gui
```

### **Step 6: Verify Services**
```bash
# Check services are running
sudo systemctl status volume-daemon
sudo systemctl status oma-api  
sudo systemctl status migration-gui

# Check health endpoints
curl -s http://localhost:8090/api/v1/health | jq
curl -s http://localhost:8082/health | jq
curl -s http://localhost:3001 | head -10
```

---

## ✅ **Post-Deployment Verification**

### **1. Database Verification**
```bash
# Check OSSEA configs
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT id, name, zone, oma_vm_id, disk_offering_id, is_active FROM ossea_configs;"

# Check VM contexts have configs
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT context_id, vm_name, ossea_config_id, credential_id FROM vm_replication_contexts LIMIT 5;"
```

### **2. Functional Testing**
- [ ] Access GUI: `http://<server-ip>:3001`
- [ ] Go to Settings → OSSEA Configuration
- [ ] Verify unified configuration loads
- [ ] Test "Add to Management" on discovery page
- [ ] Verify VM gets `ossea_config_id` assigned
- [ ] Start test replication
- [ ] Verify no "CloudStack prerequisites not met" error
- [ ] Check replication job creates successfully

### **3. Log Verification**
```bash
# OMA API logs - should show auto-detection
sudo journalctl -u oma-api --since "5 minutes ago" | grep -E "Auto-detected|config_id"

# Expected logs:
# ✅ Found active OSSEA config via auto-detection config_id=2
# 🔄 Auto-detected active OSSEA config for validation
# 🔄 Using auto-detected OSSEA config ID for migration
```

---

## 🐛 **Troubleshooting**

### **Issue: "Foreign key constraint fails"**
**Solution**: Ensure `ossea_configs` table has at least one row with `is_active = 1`

```bash
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT COUNT(*) FROM ossea_configs WHERE is_active = 1;"
```

### **Issue: "Module not found" in GUI**
**Solution**: Rebuild GUI

```bash
cd /opt/migratekit/gui
sudo rm -rf .next node_modules
sudo npm install
sudo npm run build
sudo systemctl restart migration-gui
```

### **Issue: Service won't start**
**Solution**: Check logs and permissions

```bash
# Check logs
sudo journalctl -u oma-api -n 50
sudo journalctl -u volume-daemon -n 50

# Check permissions
ls -l /opt/migratekit/bin/oma-api
ls -l /usr/local/bin/volume-daemon

# Fix permissions
sudo chmod +x /opt/migratekit/bin/oma-api
sudo chmod +x /usr/local/bin/volume-daemon
```

---

## 📊 **Version Information**

### **What's New in v6.18.0**:

**Critical Fix (v2.40.5)**:
- Fixed OSSEA config ID not being passed to migration workflow
- Added `validateAndGetConfigID()` method
- Properly updates `migrationReq.OSSEAConfigID` with detected value

**Major Features (v2.40.4)**:
- OSSEA config auto-detection in Add to Management
- Auto-assigns active config to new VMs
- Fallback logic for NULL/invalid config IDs
- Database model updates (OSSEAConfigID, CredentialID fields)

**UX Improvements (v2.40.3)**:
- Unified CloudStack configuration GUI
- Combined discovery endpoint
- Template discovery fixes

---

## 🔄 **Rollback Procedure**

If issues occur:

```bash
# Stop services
sudo systemctl stop oma-api volume-daemon migration-gui

# Restore binaries
sudo mv /opt/migratekit/bin/oma-api.backup.* /opt/migratekit/bin/oma-api
sudo mv /usr/local/bin/volume-daemon.backup.* /usr/local/bin/volume-daemon

# Restore database (if needed)
mysql -u oma_user -poma_password migratekit_oma < migratekit_oma_backup_*.sql

# Restore GUI (if needed)
sudo rm -rf /opt/migratekit/gui
sudo tar -xzf /opt/migratekit/gui_backup_*.tar.gz -C /

# Start services
sudo systemctl start volume-daemon oma-api migration-gui
```

---

## 📞 **Support**

**Documentation**:
- `OSSEA_CONFIG_AUTO_DETECTION_v2.40.5.md` - Detailed fix documentation
- `UNIFIED_CLOUDSTACK_CONFIG_v6.17.0.md` - Unified config system
- `binaries/MANIFEST.md` - Binary version history

**Logs Location**:
- OMA API: `journalctl -u oma-api`
- Volume Daemon: `journalctl -u volume-daemon`
- Migration GUI: `journalctl -u migration-gui`

---

**Deployment Status**: ✅ **READY FOR PRODUCTION**  
**Last Updated**: October 4, 2025  
**Package Version**: v6.18.0

---

**End of Checklist**

