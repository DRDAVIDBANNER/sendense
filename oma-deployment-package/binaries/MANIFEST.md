# OMA Deployment Package - Binary Manifest

**Last Updated**: October 4, 2025  
**Package Version**: v6.19.0

---

## 📦 **Binaries Included**

### **OMA API Server**

**Binary**: `oma-api`  
**Version**: v2.40.9-template-size-filter  
**Build Date**: October 4, 2025  
**Size**: 33 MB  
**SHA256**: `5ba07158a58355b68ddf0e0de86da8203862ba3e990e5ee4dbccdf959bc313f9`

**What's New in v2.40.9**:
- ✅ Template Size Filter (< 2 GB) for failover compatibility
- ✅ Template validation checks CloudStack size restrictions
- ✅ Prevents selection of incompatible templates with fixed root disk sizes
- ✅ Clear error messages explaining CloudStack template requirements

**Critical Fixes**:
1. **Template Discovery**: Only shows templates with Size < 2 GB (flexible templates)
2. **Template Validation**: Rejects templates >= 2 GB with clear error message
3. **Failover Compatibility**: Ensures CloudStack accepts dynamic root disk sizing
4. **User Guidance**: Filters out incompatible templates from dropdown

**Breaking Changes**: None - fully backward compatible

**Dependencies**:
- Go 1.21+
- MariaDB 10.5+
- Volume Daemon v1.2.1+
- Valid OSSEA configuration in database

---

## 🔧 **Installation**

```bash
# Stop OMA API
sudo systemctl stop oma-api

# Install binary
sudo cp oma-api /opt/migratekit/bin/oma-api
sudo chmod +x /opt/migratekit/bin/oma-api

# Start OMA API
sudo systemctl start oma-api

# Verify
sudo systemctl status oma-api
```

---

## 📊 **Version History**

### **v2.40.9-template-size-filter** (October 4, 2025)
- Template size filtering (< 2 GB) for failover compatibility
- Template validation check for CloudStack size restrictions
- Prevents test failover failures due to incompatible templates

### **v2.40.5-ossea-config-fix** (October 4, 2025)
- Fixed OSSEA config ID auto-detection in migration workflow
- Added `validateAndGetConfigID()` method
- Properly passes detected config ID to `migrationReq.OSSEAConfigID`

### **v2.40.4-ossea-config-auto-detect** (October 4, 2025)
- Added OSSEA config auto-detection to `EnhancedDiscoveryService`
- Added `OSSEAConfigID` field to `VMReplicationContext` model
- Fixed replication blocker for missing config IDs

### **v2.40.3-unified-cloudstack** (October 4, 2025)
- Unified CloudStack configuration GUI
- Combined discovery endpoint
- Template discovery fix

### **v2.39.0-gorm-field-fix** (October 3, 2025)
- Database field name corrections
- GORM tag fixes

---

## 🔍 **Verification**

After installation, verify the binary:

```bash
# Check service is running
sudo systemctl status oma-api

# Check logs for version
sudo journalctl -u oma-api --since "1 minute ago" | head -20

# Test health endpoint
curl -s http://localhost:8082/health | jq
```

---

## 📝 **Notes**

- This binary requires the unified CloudStack configuration to be completed
- Auto-detection only works when `is_active = 1` is set on an OSSEA config
- If no active config exists, VMs will be added but replication will require manual config assignment
- All volume operations still go through Volume Daemon (no changes)

---

**End of Manifest**
