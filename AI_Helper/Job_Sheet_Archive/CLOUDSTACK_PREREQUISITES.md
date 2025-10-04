# CloudStack Prerequisites for OMA Deployment

**Document**: CloudStack Environment Requirements  
**Created**: October 2, 2025  
**Purpose**: Ensure CloudStack environment is properly configured before OMA deployment  
**Based on**: Real deployment testing and troubleshooting results  

---

## 🎯 **OVERVIEW**

This document outlines the **mandatory CloudStack environment requirements** discovered through production deployment testing. These prerequisites must be met for the unified failover system to function correctly.

---

## 🔑 **CRITICAL CLOUDSTACK REQUIREMENTS**

### **1. API User Permissions** 🚨 **MANDATORY**

**Issue Discovered**: CloudStack API error 530 "Internal error executing command" during snapshot creation

**Root Cause**: Insufficient API user permissions for snapshot operations

**Required Permissions**:
```
✅ Volume Management:
   - CreateVolume
   - AttachVolume  
   - DetachVolume
   - DeleteVolume

✅ Snapshot Operations: ⭐ CRITICAL
   - CreateSnapshot
   - DeleteSnapshot
   - ListSnapshots
   - RevertSnapshot (if rollback needed)

✅ Virtual Machine Operations:
   - CreateVirtualMachine
   - DestroyVirtualMachine
   - StartVirtualMachine
   - StopVirtualMachine
   - ListVirtualMachines

✅ Network Operations:
   - ListNetworks
   - ListNics
```

**Verification**:
```bash
# Test snapshot creation permission
curl -X POST "http://CLOUDSTACK_IP:8080/client/api" \
  -d "command=createSnapshot&volumeid=TEST_VOLUME_ID&apikey=YOUR_API_KEY&signature=SIGNATURE"
```

### **2. VM Templates Configuration** 🚨 **MANDATORY**

**Issue Discovered**: CloudStack API error 431 "rootdisksize override (5 GB) is smaller than template size (100.00 GB)"

**Root Cause**: VM template has minimum disk size larger than requested failover VM root disk

**Requirements**:
```
✅ Template Minimum Size: ≤ 20 GB (recommended)
✅ Template OS Type: Linux (Ubuntu/CentOS recommended)  
✅ Template Architecture: x86_64
✅ Template State: Ready
✅ Template Availability: Public or accessible to API user account
```

**Template Validation**:
```bash
# Check template details
curl "http://CLOUDSTACK_IP:8080/client/api?command=listTemplates&templatefilter=executable&apikey=YOUR_API_KEY&signature=SIGNATURE"

# Look for:
# - "size": Should be ≤ 21474836480 (20GB in bytes)
# - "ostypename": Linux variant
# - "isready": true
```

**Recommended Templates**:
- **Ubuntu 20.04/22.04 LTS** (minimal installation)
- **CentOS 7/8** (minimal installation)  
- **Avoid**: Windows templates (large minimum sizes)

### **3. Disk Offerings Configuration** 🚨 **MANDATORY**

**Requirements**:
```
✅ Disk Offering Type: Custom or Fixed
✅ Minimum Size: Flexible (no hard minimum)
✅ Storage Type: Compatible with snapshot operations
✅ Zone Availability: Available in target zone
```

### **4. Network Configuration** ✅ **REQUIRED**

**Requirements**:
```
✅ Network ID: Valid network UUID in target zone
✅ Network Type: Isolated, Shared, or VPC
✅ Network State: Enabled and accessible
✅ DHCP: Enabled (for automatic IP assignment)
```

### **5. Zone and Domain Configuration** ✅ **REQUIRED**

**Requirements**:
```
✅ Zone ID: Valid zone UUID
✅ Zone State: Enabled
✅ Domain: Valid domain (numeric ID or string name)
✅ Account: API user has access to target domain/zone
```

---

## 🛠️ **CLOUDSTACK ENVIRONMENT VALIDATION**

### **Pre-Deployment Checklist**

Before deploying OMA, validate CloudStack environment:

```bash
# 1. Test API connectivity
curl -s "http://CLOUDSTACK_IP:8080/client/api?command=listCapabilities&apikey=YOUR_API_KEY&signature=SIGNATURE"

# 2. Verify snapshot permissions (use existing volume)
curl -X POST "http://CLOUDSTACK_IP:8080/client/api" \
  -d "command=createSnapshot&volumeid=EXISTING_VOLUME_ID&name=test-permission-check&apikey=YOUR_API_KEY&signature=SIGNATURE"

# 3. Check template size
curl "http://CLOUDSTACK_IP:8080/client/api?command=listTemplates&templatefilter=executable&id=YOUR_TEMPLATE_ID&apikey=YOUR_API_KEY&signature=SIGNATURE"

# 4. Verify network availability  
curl "http://CLOUDSTACK_IP:8080/client/api?command=listNetworks&zoneid=YOUR_ZONE_ID&apikey=YOUR_API_KEY&signature=SIGNATURE"
```

### **Common Configuration Issues**

| **Issue** | **Symptom** | **Solution** |
|-----------|-------------|--------------|
| **Snapshot permissions** | Error 530 "Internal error" | Grant CreateSnapshot permission to API user |
| **Template too large** | Error 431 "rootdisksize override smaller than template" | Use template ≤20GB or configure larger root disks |
| **Invalid zone/domain** | Error 431 "Invalid parameter" | Verify zone/domain IDs exist and are accessible |
| **Network not found** | VM creation fails | Verify network ID exists in target zone |

---

## 🔧 **CLOUDSTACK GUI CONFIGURATION**

### **After OMA Deployment**

1. **Access Migration GUI**: `http://OMA_IP:3001`

2. **Configure CloudStack Settings**:
   ```
   Settings → CloudStack Configuration
   
   ✅ API URL: http://CLOUDSTACK_IP:8080/client/api
   ✅ API Key: [User API key with snapshot permissions]
   ✅ Secret Key: [User secret key]
   ✅ Domain: [Domain ID or name]
   ✅ Zone: [Target zone UUID]
   ✅ Template ID: [Template UUID ≤20GB]
   ✅ Network ID: [Target network UUID]
   ✅ Service Offering: [VM sizing template]
   ✅ Disk Offering: [Disk sizing template]
   ```

3. **Test Configuration**:
   - Use "Test Connection" button in GUI
   - Verify all settings show green checkmarks
   - Test volume creation/deletion

---

## 📊 **ENVIRONMENT EXAMPLES**

### **Working Configuration Example**

```yaml
CloudStack Environment:
  API URL: http://10.245.241.101:8080/client/api
  Domain: 151 (numeric)
  Zone: 057e86db-c726-4d8c-ab1f-75c5f55d1881
  Template: 07515c1a-0d20-425a-bf82-14cc1ffd6d86 (≤20GB)
  
API User Permissions:
  - Full volume operations ✅
  - Snapshot creation/deletion ✅  
  - VM lifecycle management ✅
  - Network access ✅

Result: Snapshots succeed, failover works
```

### **Problematic Configuration Example**

```yaml
CloudStack Environment:
  API URL: http://10.245.242.102:8080/client/api
  Domain: OSSEA (string)
  Zone: 73525212-589c-465f-9662-4001c9e2835c
  Template: 69d3070a-1675-47a9-9ec8-4e3c47250c4d (100GB minimum)
  
API User Permissions:
  - Volume operations ✅
  - Snapshot creation ❌ (missing permission)
  - VM lifecycle ✅
  - Network access ✅

Result: Error 530 during snapshot creation
```

---

## 🚨 **TROUBLESHOOTING**

### **Common Errors and Solutions**

#### **Snapshot Creation Fails (Error 530)**
```
Symptom: "Internal error executing command" during snapshot creation
Cause: API user lacks CreateSnapshot permission
Solution: Grant snapshot permissions to CloudStack API user
```

#### **VM Creation Fails (Error 431 - Template Size)**
```
Symptom: "rootdisksize override smaller than template size"
Cause: Template minimum size > requested VM root disk size  
Solution: Use template ≤20GB or increase VM root disk size
```

#### **VM Creation Fails (Error 431 - Invalid Parameter)**
```
Symptom: "Invalid parameter zoneid/domainid"
Cause: Zone or domain ID doesn't exist or isn't accessible
Solution: Verify IDs exist and API user has access
```

---

## 📋 **VALIDATION COMMANDS**

### **Quick CloudStack Health Check**

```bash
# Replace with your CloudStack details
CLOUDSTACK_IP="10.245.241.101"
API_KEY="your-api-key"
SECRET_KEY="your-secret-key"

# 1. Basic connectivity
curl -s "http://$CLOUDSTACK_IP:8080/client/api?command=listCapabilities"

# 2. API authentication  
curl -s "http://$CLOUDSTACK_IP:8080/client/api?command=listAccounts&apikey=$API_KEY&signature=CALCULATED_SIGNATURE"

# 3. Template validation
curl -s "http://$CLOUDSTACK_IP:8080/client/api?command=listTemplates&templatefilter=executable&apikey=$API_KEY&signature=CALCULATED_SIGNATURE"
```

---

## 🎯 **SUCCESS CRITERIA**

**Before proceeding with OMA failover testing:**

- ✅ **API Connectivity**: CloudStack API responds to basic queries
- ✅ **Snapshot Permissions**: Can create/delete test snapshots  
- ✅ **Template Validation**: Template size ≤20GB and ready
- ✅ **Network Access**: Target network exists and is accessible
- ✅ **Zone/Domain**: Valid IDs with proper access permissions

**When all requirements are met**: OMA failover system will work correctly with snapshots, VirtIO injection, and VM creation.

---

**This document is based on real deployment issues encountered and resolved during October 2025 production testing.**
