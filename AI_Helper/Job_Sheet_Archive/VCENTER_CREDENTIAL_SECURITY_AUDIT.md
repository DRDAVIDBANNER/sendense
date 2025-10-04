# vCenter Credential Security Audit - CRITICAL SECURITY ISSUE

**Date**: 2025-09-22  
**Status**: 🚨 **CRITICAL SECURITY VIOLATION**  
**Priority**: **IMMEDIATE REMEDIATION REQUIRED**  
**Discovered By**: AI Assistant during VMA Power Management Integration  

---

## 🚨 **CRITICAL SECURITY ISSUE IDENTIFIED**

### **Hard-coded vCenter Credentials Found Across Multiple Services**

During VMA power management integration, a comprehensive audit revealed **hard-coded vCenter administrator credentials** scattered across multiple files in the OMA codebase. This represents a **critical security vulnerability** that must be addressed immediately.

---

## 📍 **CONFIRMED HARD-CODED CREDENTIAL LOCATIONS**

### **1. Migration Workflow Service**
**File**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/workflows/migration.go`  
**Lines**: 951-952  
**Code**:
```go
vmaRequest := map[string]interface{}{
    "job_id":      req.JobID,
    "vcenter":     req.VCenterHost,
    "username":    "administrator@vsphere.local", // TODO: Get from config
    "password":    "EmyGVoBFesGQc47-",            // TODO: Get from config
    "vm_paths":    []string{req.SourceVM.Path},
    "oma_url":     omaURL,
    "nbd_targets": nbd_targets,
}
```
**Impact**: All replication jobs use these hard-coded credentials

### **2. Scheduler Service**
**File**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/services/scheduler_service.go`  
**Line**: 1101  
**Code**:
```go
Username:   "administrator@vsphere.local",
```
**Context**: VM discovery from VMA  
**Impact**: All scheduled operations use hard-coded username

### **3. VMA Client Implementation (Newly Added)**
**File**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/failover/vma_client.go`  
**Lines**: Various (currently being refactored)  
**Code**: Multiple references to hard-coded credentials  
**Impact**: Failover power management operations

### **4. Enhanced Discovery Handler**
**File**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/api/handlers/enhanced_discovery.go`  
**Lines**: 146-147, 243-244, 420-421  
**Code**:
```go
if request.VCenter == "" || request.Username == "" || request.Password == "" || request.Datacenter == "" {
    h.writeErrorResponse(w, http.StatusBadRequest, "vCenter, username, password, and datacenter are required")
    return
}
```
**Impact**: API expects credentials in requests but may use hard-coded fallbacks

---

## 🔍 **SECURITY VULNERABILITY ANALYSIS**

### **Risk Level**: 🚨 **CRITICAL**

**Vulnerabilities Identified**:
1. **Credentials in Source Code**: Administrator passwords committed to git repository
2. **Multiple Duplication**: Same credentials hard-coded in 3+ locations
3. **TODO Comments**: Code shows awareness but no implementation of secure config
4. **Version Control Exposure**: Credentials visible in git history
5. **Production Deployment**: These credentials are used in production systems

### **Attack Vectors**:
- **Code Repository Access**: Anyone with repository access has admin vCenter credentials
- **Binary Analysis**: Credentials could be extracted from compiled binaries
- **Log Exposure**: Credentials might appear in debug logs or error messages
- **Memory Dumps**: Credentials stored in process memory as plain text

### **Compliance Issues**:
- Violates security best practices for credential management
- May violate organizational security policies
- Creates audit trail concerns for production environments

---

## 📋 **IMMEDIATE REMEDIATION REQUIREMENTS**

### **Phase 1: Urgent Security Fix (CRITICAL)**

**1. Credential Externalization**
- [ ] Create secure configuration management system
- [ ] Move ALL hard-coded credentials to environment variables or secure config files
- [ ] Implement credential encryption at rest

**2. Repository Cleanup**
- [ ] Remove all hard-coded credentials from source code
- [ ] Update git history to purge credential exposure (if required by security policy)
- [ ] Add pre-commit hooks to prevent future credential commits

**3. Immediate File Updates Required**:
- [ ] `oma/workflows/migration.go` - Lines 951-952
- [ ] `oma/services/scheduler_service.go` - Line 1101  
- [ ] `oma/failover/vma_client.go` - Multiple locations
- [ ] Any fallback/default credential references

### **Phase 2: Secure Architecture Implementation**

**1. Configuration Management**
- [ ] Implement centralized credential management
- [ ] Create secure credential storage (HashiCorp Vault, AWS Secrets Manager, etc.)
- [ ] Environment-based configuration with secure defaults

**2. Code Refactoring**
- [ ] Create `VCenterCredentialManager` service
- [ ] Update all services to use credential manager
- [ ] Remove all hard-coded credential references

**3. Security Hardening**
- [ ] Implement credential rotation capabilities
- [ ] Add audit logging for credential access
- [ ] Create secure credential distribution for deployments

---

## 🔧 **RECOMMENDED IMPLEMENTATION APPROACH**

### **Immediate Fix (Deploy Today)**
```go
// Environment-based credentials
type VCenterConfig struct {
    Host     string
    Username string
    Password string
}

func GetVCenterConfig() (*VCenterConfig, error) {
    host := os.Getenv("VCENTER_HOST")
    username := os.Getenv("VCENTER_USERNAME") 
    password := os.Getenv("VCENTER_PASSWORD")
    
    if host == "" || username == "" || password == "" {
        return nil, fmt.Errorf("vCenter credentials not configured")
    }
    
    return &VCenterConfig{
        Host:     host,
        Username: username,
        Password: password,
    }, nil
}
```

### **Secure Deployment Configuration**
```bash
# Production environment variables
export VCENTER_HOST="192.168.17.159"
export VCENTER_USERNAME="administrator@vsphere.local"
export VCENTER_PASSWORD="EmyGVoBFesGQc47-"
```

### **Service Integration Pattern**
```go
// Update all services to use:
vcenterConfig, err := GetVCenterConfig()
if err != nil {
    return fmt.Errorf("vCenter configuration error: %w", err)
}

// Use vcenterConfig.Host, vcenterConfig.Username, vcenterConfig.Password
```

---

## 🚨 **CRITICAL ACTION ITEMS**

### **Immediate (Next Session)**
1. **STOP**: No new deployments until credentials are secured
2. **AUDIT**: Review all recent git commits for credential exposure
3. **IMPLEMENT**: Environment-based credential management
4. **UPDATE**: All identified files with secure credential access
5. **TEST**: Ensure all services work with new credential management
6. **DEPLOY**: Updated binaries with secure configuration

### **Follow-up (Within 24 Hours)**
1. **ROTATE**: Change vCenter administrator password
2. **AUDIT**: Review access logs for potential credential compromise
3. **DOCUMENT**: Security incident and remediation steps
4. **PREVENT**: Implement pre-commit security scanning

---

## 📊 **IMPACT ASSESSMENT**

### **Current Exposure**
- **Production Systems**: ✅ Currently functional but insecure
- **Development Environment**: ✅ Working but credentials exposed
- **Git Repository**: ❌ Credentials visible in source code
- **Deployment Binaries**: ❌ Credentials embedded in executables

### **Post-Remediation Security**
- **Production Systems**: ✅ Secure environment-based credentials
- **Development Environment**: ✅ Local environment configuration
- **Git Repository**: ✅ Clean of credential references
- **Deployment Binaries**: ✅ No embedded credentials

---

## 🎯 **SUCCESS CRITERIA**

### **Security Compliance Achieved When**:
- [ ] **Zero hard-coded credentials** in source code
- [ ] **Environment-based configuration** implemented across all services
- [ ] **All identified files updated** with secure credential access
- [ ] **Production deployment** using secure credential management
- [ ] **Git repository cleaned** of credential references
- [ ] **Security testing** confirms no credential leakage

---

**🚨 CRITICAL**: This security issue requires **immediate attention** before any production deployments. The hard-coded credentials represent a **serious security vulnerability** that must be addressed in the next development session.

**NEXT SESSION PRIORITY**: Credential security remediation takes precedence over all other development tasks.


