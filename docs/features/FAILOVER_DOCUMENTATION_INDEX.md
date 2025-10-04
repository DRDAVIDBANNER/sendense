# VM Failover System Documentation Index

**Version**: 1.0  
**Date**: 2025-08-18  
**Last Updated**: 2025-08-18

## 📚 **Complete Documentation Suite**

This comprehensive documentation covers all aspects of the VM Failover System, from high-level architecture to detailed implementation specifications. All information is accurate as of the current implementation status.

## 📋 **Documentation Structure**

### **🎯 Core Documentation**

#### **1. [VM Failover System Overview](VM_FAILOVER_SYSTEM.md)**
**Primary Reference Document**
- Complete system overview and architecture
- Current implementation status  
- Known issues and limitations
- API endpoint summary
- Database schema overview
- Configuration requirements

#### **2. [Failover API Documentation](../api/FAILOVER_API.md)**
**Complete API Reference**
- All 17 API endpoints with examples
- Request/response models
- Error handling and status codes
- Query parameters and filtering
- Authentication requirements
- Swagger integration details

#### **3. [Failover Database Schema](../database/FAILOVER_SCHEMA.md)**
**Database Design & Implementation**
- Complete table schemas with field descriptions
- Indexes and performance optimization
- Critical queries for failover operations
- Database migrations and extensions
- Data relationships and foreign keys

#### **4. [Failover Process Flows](../architecture/FAILOVER_FLOWS.md)**
**Detailed Process Documentation**
- Step-by-step live failover process
- New test failover architecture (VM snapshot approach)
- Pre-failover validation flows
- Error handling and rollback procedures
- Performance metrics and monitoring

---

## 🔧 **Current Implementation Status**

### ✅ **Fully Implemented & Documented**
- **Pre-Failover Validation**: 100% complete with comprehensive validation checks
- **Live Failover Engine**: 750+ lines, fully operational
- **API Endpoints**: All 17 endpoints implemented with Swagger docs
- **Database Schema**: Complete schema with all required tables and indexes
- **GUI Integration**: React frontend with real-time progress tracking
- **VM Specification Storage**: Extended vm_disks table with VM specs
- **Volume ID Mapping**: Real CloudStack UUID resolution
- **Network Mapping Service**: Complete source-to-destination network mapping

### ⚠️ **Requires Architectural Update**
- **Test Failover Engine**: 690+ lines implemented but needs rewrite
- **Issue**: CloudStack KVM volume snapshots disabled by default
- **Solution**: New VM snapshot approach documented in process flows
- **Status**: Architecture fully documented, implementation pending

### 🔍 **Debugging Session Results (2025-08-18)**
- ✅ Fixed ChangeID validation with database-based VM specs
- ✅ Resolved volume ID mapping (database ID vs CloudStack UUID)
- ✅ Confirmed OSSEA API integration working correctly
- ✅ Test failover reaches snapshot creation (validates all prior steps)
- ⚠️ Identified CloudStack KVM limitation requiring new approach

---

## 📖 **How to Use This Documentation**

### **For Development Work**
1. **Start with**: [VM Failover System Overview](VM_FAILOVER_SYSTEM.md)
2. **API Implementation**: [Failover API Documentation](../api/FAILOVER_API.md)
3. **Database Work**: [Failover Database Schema](../database/FAILOVER_SCHEMA.md)
4. **Process Understanding**: [Failover Process Flows](../architecture/FAILOVER_FLOWS.md)

### **For System Integration**
1. **API Endpoints**: Complete endpoint reference with examples
2. **Database Queries**: Ready-to-use SQL for all validation operations
3. **Error Handling**: Comprehensive error scenarios and recovery procedures
4. **Monitoring**: Performance metrics and logging strategies

### **For Troubleshooting**
1. **Known Issues**: Documented in system overview
2. **Error Codes**: Complete error handling reference in API docs
3. **Database Queries**: Diagnostic queries in schema documentation
4. **Process Flows**: Step-by-step debugging in flow documentation

---

## 🎯 **Key Information for Future Development**

### **Test Failover Implementation Guidance**
**Current Challenge**: CloudStack KVM volume snapshots disabled  
**Required Solution**: VM snapshot approach instead of volume snapshots

**New Process (Documented)**:
1. Create test VM with identical specifications
2. Detach volume from OMA → Attach to test VM
3. Take VM snapshot (includes volume state)
4. Run test → Shutdown VM → Revert snapshot
5. Detach volume → Reattach to OMA → Cleanup

**Implementation Files to Update**:
- `internal/oma/failover/test_failover.go` (rewrite snapshot logic)
- Volume detach/attach operations
- VM snapshot creation/revert functions
- Test cleanup process

### **Critical Database Information**
**Volume ID Resolution**: Always use `ossea_volumes.volume_id` (CloudStack UUID) not `ossea_volumes.id` (database integer)

**VM Specifications**: Stored in first disk record of each VM in `vm_disks` table
```sql
SELECT vd.cpu_count, vd.memory_mb, vd.os_type, ov.volume_id
FROM vm_disks vd
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id  
JOIN replication_jobs rj ON vd.job_id = rj.id
WHERE rj.source_vm_id = ? AND vd.cpu_count > 0
ORDER BY vd.created_at DESC LIMIT 1;
```

**ChangeID Validation**: Query `cbt_history` with join to `replication_jobs`
```sql
SELECT cb.change_id FROM cbt_history cb
JOIN replication_jobs rj ON cb.job_id = rj.id
WHERE rj.source_vm_id = ? AND cb.sync_success = TRUE
ORDER BY cb.created_at DESC LIMIT 1;
```

### **API Testing Information**
**Base URL**: `http://localhost:8082/api/v1`  
**Swagger UI**: `http://localhost:8082/swagger/index.html`  
**Authentication**: Currently disabled for troubleshooting

**Test Failover Example**:
```bash
curl -X POST "http://localhost:8082/api/v1/failover/test" \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
    "vm_name": "PGWINTESTBIOS", 
    "test_duration": "30m",
    "auto_cleanup": true
  }'
```

---

## 🔄 **Documentation Maintenance**

### **When to Update Documentation**
- API endpoint changes or additions
- Database schema modifications
- Process flow updates or new approaches
- Error handling changes
- Performance optimizations

### **Documentation Versioning**
- Version numbers match system implementation
- Major changes require version increment
- Date stamps track when updates were made
- Status indicators show current implementation state

### **Accuracy Guarantee**
All documentation reflects the **actual current implementation** as of 2025-08-18. No assumptions or planned features are documented as complete. Future development should verify current state before making changes.

---

## 📞 **Support Information**

### **Debug Endpoints**
- `GET /api/v1/debug/health` - System health check
- `GET /api/v1/debug/failover-jobs` - Detailed job debugging
- `GET /api/v1/debug/endpoints` - Available API endpoints

### **Logging Locations**
- **API Logs**: `sudo journalctl -u oma-api`
- **Database Logs**: Check MariaDB error logs
- **OSSEA Integration**: CloudStack API response logs

### **Configuration Files**
- **Service**: `/etc/systemd/system/oma-api.service`
- **Binary**: `/opt/migratekit/bin/oma-api`
- **Database**: `migratekit_oma` (MariaDB)

---

## 📚 **Related Documentation**
- [Main Project Documentation](../README.md)
- [Database Schema Overview](../database-schema.md)
- [OSSEA Integration Guide](../ossea-integration.md)
- [API Documentation Index](../api/README.md)
- [VM Failover Implementation Plan](../../AI_Helper/VM_FAILOVER_IMPLEMENTATION_PLAN.md)

