# OSSEA Integration Guide

## 🎯 **Overview**

This guide explains the **production-ready** OSSEA integration in MigrateKit for complete volume lifecycle management during VM migrations. The system provides automated volume creation, attachment, mounting, and cleanup with CloudStack/OSSEA infrastructure.

## ✅ **Production Status**

**FULLY OPERATIONAL** - Complete OSSEA configuration management with database persistence:
- ✅ **MariaDB Integration** with GORM auto-migration
- ✅ **Unified API Endpoint** for all OSSEA configuration operations
- ✅ **Real Database Persistence** (no simulated responses)
- ✅ **Interactive Swagger Documentation** 
- ✅ **Production Service Configuration** with systemd
- ✅ **Volume Lifecycle Management** (create→attach→mount→unmount→detach→delete)

## 🔧 **Configuration**

### **1. Identifying the OMA VM in OSSEA**

The OMA appliance must be registered as a VM in OSSEA to attach volumes. You need to:

1. **Find your OMA VM ID in OSSEA**:
   ```bash
   # Using CloudStack/OSSEA CLI
   cs listVirtualMachines name=oma-appliance
   
   # Or via API
   curl "http://ossea.example.com/client/api?command=listVirtualMachines&name=oma-appliance&..."
   ```

2. **Note the VM ID** (format: `12345678-1234-1234-1234-123456789012`)

### **2. Configure OSSEA Connection**

#### **Option A: Using OMA API (Recommended)**

The OMA API now provides a unified endpoint for OSSEA configuration management with database persistence:

```bash
# Create OSSEA configuration via API
curl -X POST http://localhost:8082/api/v1/ossea/config \
  -H "Authorization: Bearer <session_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "create",
    "configuration": {
      "name": "production-ossea",
      "api_url": "http://10.245.241.101:8080/client/api",
      "api_key": "GdsWBVHco0OOW4vBlgfnyzjru65FV-U1l1kKJ7n2WwH0gn3soTaZQZZyXfgUsxX7PyP06WrOOOcNRKmhRWDSlA",
      "secret_key": "uTCroKUkHZaNybhBXkcQsCb_eKDvZKbhHaZK4I1nHrGJYLKN-j0O-t9EGUx9yBdHH3F8dN5wVelitvdpQjwdcQ",
      "domain": "/",
      "zone": "OSSEA-Zone",
      "oma_vm_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
    }
  }'
```

#### **Option B: Using Environment Variables (Legacy)**
```bash
export OSSEA_API_URL="http://ossea.example.com/client/api"
export OSSEA_API_KEY="your-api-key"
export OSSEA_SECRET_KEY="your-secret-key"
export OSSEA_ZONE="zone1"
export OSSEA_OMA_VM_ID="12345678-1234-1234-1234-123456789012"  # Your OMA VM ID
```

#### **Option C: Using YAML Configuration File (Legacy)**
```yaml
# configs/ossea-prod.yaml
name: "production-ossea"
api_url: "http://ossea.example.com/client/api"
api_key: "your-api-key"
secret_key: "your-secret-key"
zone: "zone1"
oma_vm_id: "12345678-1234-1234-1234-123456789012"  # Your OMA VM ID
```

### **3. Database Configuration**

The OSSEA configuration is stored in MariaDB. The `oma_vm_id` field identifies which VM in OSSEA this OMA instance represents.

```sql
-- Example: Update existing configuration
UPDATE ossea_configs 
SET oma_vm_id = '12345678-1234-1234-1234-123456789012' 
WHERE name = 'production-ossea';
```

## 📦 **Volume Lifecycle**

### **1. Volume Creation**
When a migration job starts, OMA creates volumes in OSSEA matching the source VM disk sizes:
- Volumes are tagged with migration metadata
- Named using pattern: `mig_<vm_name>_disk_<n>`

### **2. Volume Attachment**
Before mounting, volumes must be attached to the OMA VM in OSSEA:
- Uses the configured `oma_vm_id` to identify target VM
- Attaches volumes as virtio devices (`/dev/vdb`, `/dev/vdc`, etc.)
- Device assignment is automatic based on availability

### **3. Volume Mounting**
After attachment, volumes are mounted on the OMA filesystem:
- Mount points: `/mnt/migration/<job_id>/<disk_name>`
- Filesystem auto-detection
- Read-write access for NBD export

### **4. Volume Detachment**
After migration completion:
- Volumes are unmounted from filesystem
- Detached from OMA VM in OSSEA
- Can be attached to target VM or deleted

## 🔄 **Migration Workflow**

```
1. Create Job → 2. Create Volumes → 3. Attach to OMA → 4. Mount Locally
                                            ↓
8. Delete/Reuse ← 7. Detach from OMA ← 6. Unmount ← 5. Stream Data (NBD)
```

## 🚀 **Volume Management API**

### **Automatic Volume Lifecycle**

The OSSEA integration provides a complete, automated volume management system:

```go
// Volume lifecycle is handled automatically
service := volume_service.NewVolumeService(client, mountManager)

// Creates volumes for each VM disk, attaches to OMA, and mounts locally
err := service.AttachAndMountVolumesForJob(jobID)

// Migration data streams to mounted volumes via NBD
// ...

// Cleanup: unmounts, detaches from OMA, and deletes volumes
err = service.UnmountAndDetachVolumesForJob(jobID)
```

### **Key Features**

- **Auto Device Assignment**: CloudStack automatically assigns device IDs (no conflict management needed)
- **Async Job Handling**: Proper monitoring of CloudStack async operations
- **Error Recovery**: Automatic cleanup on any failure in the lifecycle
- **Admin Support**: Works with CloudStack admin accounts for full permissions
- **State Management**: Tracks volume states (Allocated → Ready → Mounted)

### **Volume Creation**

Volumes are automatically created with the same size as source VM disks:

```go
// Automatically determines size from VMware source
createReq := &ossea.CreateVolumeRequest{
    Name:           fmt.Sprintf("migration-%s-disk-%d", jobID, diskIndex),
    SizeGB:         sourceVMDiskSizeGB,
    Zone:           config.Zone,
    DiskOfferingID: config.DiskOfferingID,
}
```

### **Device Management**

Device ID conflicts are eliminated through CloudStack's auto-assignment:

```go
attachReq := &ossea.AttachVolumeRequest{
    VolumeID:         volume.ID,
    VirtualMachineID: config.OMAVMID,
    DeviceID:         0, // 0 = auto-assign next available device
}
```

## 🛠️ **Troubleshooting**

### **Database Connection Issues**
```
Error 1045 (28000): Access denied for user 'oma_user'@'localhost'
```
**Solution**: Ensure MariaDB user has proper permissions:
```bash
sudo mysql -u root -p -e "GRANT ALL PRIVILEGES ON migratekit_oma.* TO 'oma_user'@'localhost'; FLUSH PRIVILEGES;"
```

### **Missing Database Tables**
```
Error 1146 (42S02): Table 'migratekit_oma.ossea_configs' doesn't exist
```
**Solution**: Ensure auto-migration is enabled in the OMA API startup. The binary with migration support should be used.

### **Missing OMA VM ID Error**
```
Error: OMA VM ID not configured in OSSEA config
```
**Solution**: Set the `oma_vm_id` field when creating the configuration via the API.

### **Volume Attachment Failed**
```
Error: failed to attach volume: VM not found
```
**Solution**: Verify the OMA VM ID is correct and the VM exists in OSSEA.

### **Permission Denied**
```
Error: Caller does not have permission to operate with provided resource
```
**Solution**: Ensure using admin-level OSSEA API credentials with VM management permissions.

### **API Connection Test Failed**
```
OSSEA API error 0: 
```
**Solution**: This indicates authentication issues with OSSEA. Verify API credentials, network connectivity to OSSEA, and ensure proper HMAC-SHA1 signature generation. The API is now making real calls to OSSEA CloudStack.

## 📋 **Best Practices**

1. **Use the OMA API** for all OSSEA configuration management
2. **Always configure OMA VM ID** before starting migrations
3. **Test connections** using the API test endpoint
4. **Monitor database logs** for CRUD operation success
2. **Monitor volume attachments** to avoid device exhaustion
3. **Clean up volumes** after failed migrations
4. **Use consistent naming** for easy volume identification
5. **Tag volumes** with migration metadata for tracking

## 🔗 **API Reference**

For complete API documentation including interactive Swagger UI, see:
- **[OMA API Documentation](api/oma-api.md)** - Complete endpoint reference
- **Interactive Swagger**: `http://localhost:8082/swagger/` when API is running

## 📊 **Current Status**

**✅ PRODUCTION READY** - Database integration complete
- **Database Persistence**: All OSSEA configurations stored in MariaDB
- **Auto-Migration**: Schema automatically managed by GORM
- **Unified API**: Single endpoint for all CRUD operations
- **Interactive Documentation**: Complete Swagger integration
- **Service Ready**: Systemd configuration available

**✅ COMPLETE**: Real OSSEA connection test integrated - API now makes actual calls to OSSEA CloudStack for authentication and zone verification

## 🔒 **Security Considerations**

- API keys should be stored securely
- Use environment variables in production
- Restrict API key permissions to necessary operations
- Monitor volume creation/deletion for cost control

---

For more details on the OSSEA API, refer to the CloudStack API documentation.