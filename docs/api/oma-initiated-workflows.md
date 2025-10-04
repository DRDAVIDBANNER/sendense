# OMA-Initiated Workflows

## 🎯 **Overview**

This document describes the **OMA-initiated workflow** capability that enables the CloudStack appliance (OMA) to discover VMs and start replications by commanding the VMware appliance (VMA) through the tunnel.

## 🌐 Base URLs

- OMA API: `http://localhost:8082` (running now; use this for all OMA API calls)
- VMA API (via tunnel from OMA): `http://localhost:9081`

## 🔄 **Workflow Architecture**

### **Current State (VMA-initiated)**
```
VMA Client → vCenter Discovery → Send to OMA → Create Jobs → Start Migration
```

### **New Capability (OMA-initiated)**
```
OMA API → VMA API (via tunnel) → vCenter Discovery → Response → OMA Create Jobs → VMA API Start Replication
```

## 📋 **New VMA API Endpoints**

### **Discovery Endpoint**
```http
POST /api/v1/discover
```

**Purpose**: OMA can trigger VM discovery from any vCenter
**Implementation**: ✅ API endpoint added, VMware integration needed

### **Replication Endpoint** 
```http
POST /api/v1/replicate
```

**Purpose**: OMA can start replication of specific VMs
**Implementation**: ✅ API endpoint added, migratekit integration needed

## 🔄 **Complete OMA-Initiated Workflow**

### **Step 1: OMA Discovers VMs**
```bash
# OMA calls VMA via tunnel to discover VMs
curl -X POST http://localhost:9081/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{
    "vcenter": "quad-vcenter-01.quadris.local",
    "username": "administrator@vsphere.local",
    "password": "EmyGVoBFesGQc47-",
    "datacenter": "DatabanxDC",
    "filter": "PGWINTESTBIOS"
  }'
```

### **Step 2: OMA Creates Replication Jobs**
```bash
# OMA creates replication job with dynamic allocation
curl -X POST http://localhost:8082/api/v1/replications \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "source_vm": {
      "id": "vm-143233",
      "name": "PGWINTESTBIOS", 
      "path": "/DatabanxDC/vm/PGWINTESTBIOS"
    },
    "replication_type": "initial"
  }'
```

### **Step 3: OMA Starts VMA Replication**
```bash
# OMA tells VMA to start replication via tunnel
curl -X POST http://localhost:9081/api/v1/replicate \
  -H "Content-Type: application/json" \
  -d '{
    "job_id": "repl-vm-143233-1754407216",
    "vcenter": "quad-vcenter-01.quadris.local",
    "username": "administrator@vsphere.local",
    "password": "EmyGVoBFesGQc47-",
    "vm_paths": ["/DatabanxDC/vm/PGWINTESTBIOS"],
    "oma_url": "http://localhost:8082"
  }'
```

### **Step 4: VMA Executes Migration**
The VMA will:
1. Connect to vCenter with provided credentials
2. Use dynamic port allocation from OMA 
3. Start migratekit with the allocated NBD export
4. Report progress back to OMA

## 📊 **Implementation Status**

### **✅ Completed**
- VMA API endpoints for `/discover` and `/replicate` 
- Request/response data structures
- API documentation with examples
- Tunnel infrastructure for secure communication

### **🔧 In Progress** 
- VMware client integration for real discovery
- migratekit integration for real replication
- Progress reporting back to OMA

### **📋 Next Steps**
1. **Implement VMware Discovery**: Replace simple client with real vCenter integration
2. **Implement Replication Startup**: Integrate with existing migratekit workflow
3. **Test End-to-End**: Complete OMA → VMA → vCenter → Migration workflow

## 🔐 **Security & Network Compliance**

### **Tunnel Security**
- ✅ All communication via SSH tunnel (port 9081)
- ✅ No direct network connections between appliances
- ✅ Credentials passed securely through tunnel
- ✅ Network firewall rules maintained (only ports 22, 80, 443)

### **API Security**
- ✅ Tunnel-based authentication
- ✅ Request validation and error handling
- ✅ Secure credential handling

## 🧪 **Testing Current Implementation**

### **Test VMA API Tunnel**
```bash
# From OMA - test tunnel connectivity
curl -s http://localhost:9081/api/v1/health

# Test discovery endpoint (will show "not implemented" message)
curl -X POST http://localhost:9081/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{"vcenter":"test","username":"test","password":"test","datacenter":"test"}'

# Test replication endpoint (will show "not implemented" message)  
curl -X POST http://localhost:9081/api/v1/replicate \
  -H "Content-Type: application/json" \
  -d '{"job_id":"test","vcenter":"test","username":"test","password":"test","vm_paths":["test"]}'
```

## 🎯 **Benefits of OMA-Initiated Workflows**

### **Centralized Control**
- OMA becomes the orchestration hub
- Single point of control for all migrations
- Better resource management and scheduling

### **Scalability**
- Support multiple VMA appliances
- Concurrent migrations across different vCenters
- Load balancing and queue management

### **User Experience**
- Single web interface for all operations
- Real-time monitoring and control
- Simplified workflow management

## 📋 **API Compatibility**

### **Maintained Capabilities**
- ✅ Original VMA-initiated workflows still work
- ✅ Existing cleanup and status endpoints unchanged
- ✅ Backward compatibility with current tools

### **Enhanced Capabilities**
- ✅ OMA can discover VMs from any vCenter
- ✅ OMA can start replications on demand
- ✅ Full tunnel-based security maintained

---
**Status**: ✅ **API INFRASTRUCTURE READY** - VMware integration needed for complete functionality