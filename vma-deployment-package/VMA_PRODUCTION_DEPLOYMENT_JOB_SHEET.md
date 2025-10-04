# 🚀 **VMA PRODUCTION DEPLOYMENT JOB SHEET**

**Created**: October 2, 2025  
**Session Focus**: Complete VMA production deployment package and automation  
**Status**: 📋 **PLANNING PHASE** - Comprehensive job breakdown  
**Priority**: 🔥 **CRITICAL** - Production VMA template creation

---

## 🎯 **PROJECT OBJECTIVES**

### **Primary Goal**: Create complete, self-contained VMA deployment package
### **Secondary Goal**: Bulletproof VMA deployment automation script
### **Success Criteria**: 100% automated VMA deployment with tunnel connectivity
### **Testing**: NO deployment testing until ALL tasks complete

---

## 📋 **JOB BREAKDOWN - TICK OFF AS COMPLETED**

### **🔧 PHASE 1: VMA BINARY COLLECTION AND PACKAGING**

#### **Task 1.1: Create VMA Package Directory Structure** ⏰ **15 min**
- [ ] **1.1.1**: Create `/home/pgrayson/vma-deployment-package/` directory
- [ ] **1.1.2**: Create subdirectories: `binaries/`, `configs/`, `scripts/`, `keys/`, `dependencies/`
- [ ] **1.1.3**: Set proper permissions on package directory
- [ ] **1.1.4**: Verify directory structure matches OMA package layout

#### **Task 1.2: Collect Current VMA Binaries** ⏰ **30 min**
- [ ] **1.2.1**: Copy MigrateKit binary from VMA 233 (`/opt/vma/bin/migratekit` → 20.9MB)
- [ ] **1.2.2**: Copy VMA API Server binary from VMA 233 (`/opt/vma/bin/vma-api-server` → 20.7MB)
- [ ] **1.2.3**: Verify binary sizes match source (20,933,808 bytes migratekit, 20,667,184 bytes vma-api)
- [ ] **1.2.4**: Set executable permissions on copied binaries
- [ ] **1.2.5**: Document exact binary versions and dates in package

#### **Task 1.3: Identify Latest Source Binaries** ⏰ **20 min**
- [ ] **1.3.1**: Compare VMA 233 binaries with latest in `/source/current/`
- [ ] **1.3.2**: Verify `migratekit-v2.21.1-chunk-size-fix` is newer than deployed version
- [ ] **1.3.3**: Verify `vma-api-server-multi-disk-debug` is newer than deployed version
- [ ] **1.3.4**: Copy latest binaries to package if newer versions exist
- [ ] **1.3.5**: Document version upgrade recommendations

#### **Task 1.4: Package Binary Validation** ⏰ **15 min**
- [ ] **1.4.1**: Verify all binaries are executable
- [ ] **1.4.2**: Check binary file types and architectures
- [ ] **1.4.3**: Calculate package binary directory size
- [ ] **1.4.4**: Create binary inventory manifest

### **🔧 PHASE 2: VMA CONFIGURATION PACKAGING**

#### **Task 2.1: Service Configuration Files** ⏰ **30 min**
- [ ] **2.1.1**: Copy VMA API service file from VMA 233 (`/etc/systemd/system/vma-api.service`)
- [ ] **2.1.2**: Copy SSH tunnel service file from VMA 233 (`/etc/systemd/system/vma-ssh-tunnel.service`)
- [ ] **2.1.3**: Copy tunnel wrapper script from VMA 233 (`/usr/local/bin/vma-tunnel-wrapper.sh`)
- [ ] **2.1.4**: Verify all service files reference correct binary paths
- [ ] **2.1.5**: Package service files in `configs/` directory

#### **Task 2.2: Fixed VMA Configuration Template** ⏰ **20 min**
- [ ] **2.2.1**: Create `vma-config.conf.template` with quoted SETUP_DATE
- [ ] **2.2.2**: Test template syntax with bash source command
- [ ] **2.2.3**: Verify template contains all required variables
- [ ] **2.2.4**: Document template usage and variable substitution
- [ ] **2.2.5**: Package template in `configs/` directory

#### **Task 2.3: SSH Key Collection** ⏰ **15 min**
- [ ] **2.3.1**: Copy VMA pre-shared key from VMA 233 (`/home/vma/.ssh/cloudstack_key`)
- [ ] **2.3.2**: Copy VMA public key (`/home/vma/.ssh/cloudstack_key.pub`)
- [ ] **2.3.3**: Verify key pair matches and is functional
- [ ] **2.3.4**: Package keys in `keys/` directory with proper permissions
- [ ] **2.3.5**: Document key usage and security requirements

### **🔧 PHASE 3: VMA WIZARD FIXES**

#### **Task 3.1: Update VMA Wizard Source** ⏰ **30 min**
- [ ] **3.1.1**: Verify wizard fix applied to `/home/pgrayson/migratekit-cloudstack/vma-setup-wizard.sh`
- [ ] **3.1.2**: Check both SETUP_DATE lines are quoted (lines 158 and 823)
- [ ] **3.1.3**: Test wizard config generation with syntax validation
- [ ] **3.1.4**: Copy fixed wizard to package `scripts/` directory
- [ ] **3.1.5**: Verify wizard restart logic is functional

#### **Task 3.2: Wizard Validation Enhancement** ⏰ **20 min**
- [ ] **3.2.1**: Add service status validation after restart in wizard
- [ ] **3.2.2**: Add config syntax validation before service restart
- [ ] **3.2.3**: Add tunnel connectivity test in wizard
- [ ] **3.2.4**: Enhance error reporting in wizard
- [ ] **3.2.5**: Test enhanced wizard validation logic

### **🔧 PHASE 4: VMA DEPLOYMENT SCRIPT MODERNIZATION**

#### **Task 4.1: Create New VMA Deployment Script** ⏰ **45 min**
- [ ] **4.1.1**: Create `deploy-vma-production.sh` based on OMA script structure
- [ ] **4.1.2**: Add remote deployment capability (TARGET_IP parameter)
- [ ] **4.1.3**: Add passwordless sudo setup (Phase 1)
- [ ] **4.1.4**: Add self-contained package usage (no external dependencies)
- [ ] **4.1.5**: Add comprehensive validation phase

#### **Task 4.2: Binary Deployment Logic** ⏰ **30 min**
- [ ] **4.2.1**: Add binary deployment from package (`$PACKAGE_DIR/binaries/`)
- [ ] **4.2.2**: Add fallback to source directory if package missing
- [ ] **4.2.3**: Add proper binary permissions and ownership
- [ ] **4.2.4**: Add binary validation and health checks
- [ ] **4.2.5**: Add symlink creation for compatibility

#### **Task 4.3: Service Configuration Deployment** ⏰ **30 min**
- [ ] **4.3.1**: Deploy VMA API service configuration
- [ ] **4.3.2**: Deploy SSH tunnel service configuration
- [ ] **4.3.3**: Deploy tunnel wrapper script
- [ ] **4.3.4**: Deploy fixed wizard with quoted SETUP_DATE
- [ ] **4.3.5**: Add service startup and validation logic

#### **Task 4.4: SSH Key and Authentication Setup** ⏰ **20 min**
- [ ] **4.4.1**: Deploy VMA SSH keys from package
- [ ] **4.4.2**: Set proper key permissions (600 for private, 644 for public)
- [ ] **4.4.3**: Configure SSH client for tunnel connectivity
- [ ] **4.4.4**: Add SSH key validation and testing
- [ ] **4.4.5**: Document key management procedures

### **🔧 PHASE 5: DEPENDENCY AND SYSTEM SETUP**

#### **Task 5.1: System Dependencies** ⏰ **20 min**
- [ ] **5.1.1**: Document all required packages from current deployment script
- [ ] **5.1.2**: Create package installation logic with proper error handling
- [ ] **5.1.3**: Add cloud-init disabling (production hardening)
- [ ] **5.1.4**: Add system user configuration (vma user setup)
- [ ] **5.1.5**: Add directory structure creation with proper permissions

#### **Task 5.2: NBD Stack Integration** ⏰ **25 min**
- [ ] **5.2.1**: Evaluate if NBD stack tar.gz is needed in package
- [ ] **5.2.2**: Package NBD stack if required by VMA deployment
- [ ] **5.2.3**: Add NBD stack extraction and setup logic
- [ ] **5.2.4**: Add VDDK library setup and symlinks
- [ ] **5.2.5**: Verify NBD stack functionality requirements

### **🔧 PHASE 6: SCRIPT INTEGRATION AND TESTING PREPARATION**

#### **Task 6.1: Script Integration** ⏰ **30 min**
- [ ] **6.1.1**: Update script to reference VMA deployment package
- [ ] **6.1.2**: Add package path detection and validation
- [ ] **6.1.3**: Add comprehensive error handling throughout script
- [ ] **6.1.4**: Add detailed logging for troubleshooting
- [ ] **6.1.5**: Add script version and package compatibility checks

#### **Task 6.2: Validation Framework** ⏰ **25 min**
- [ ] **6.2.1**: Add VMA service health checks (API, tunnel)
- [ ] **6.2.2**: Add tunnel connectivity validation
- [ ] **6.2.3**: Add binary functionality testing
- [ ] **6.2.4**: Add configuration syntax validation
- [ ] **6.2.5**: Add complete system status reporting

### **🔧 PHASE 7: DOCUMENTATION AND FINALIZATION**

#### **Task 7.1: Complete Package Documentation** ⏰ **30 min**
- [ ] **7.1.1**: Create VMA deployment package README
- [ ] **7.1.2**: Document all binary specifications and functions
- [ ] **7.1.3**: Document configuration templates and usage
- [ ] **7.1.4**: Document SSH key management and security
- [ ] **7.1.5**: Create troubleshooting guide

#### **Task 7.2: Script Documentation** ⏰ **20 min**
- [ ] **7.2.1**: Add comprehensive script header documentation
- [ ] **7.2.2**: Document all script phases and functions
- [ ] **7.2.3**: Add usage examples and parameter documentation
- [ ] **7.2.4**: Document error codes and troubleshooting
- [ ] **7.2.5**: Create script version changelog

#### **Task 7.3: Integration Documentation** ⏰ **15 min**
- [ ] **7.3.1**: Document VMA-OMA integration procedures
- [ ] **7.3.2**: Document tunnel setup and validation
- [ ] **7.3.3**: Update connection cheat sheet with VMA package info
- [ ] **7.3.4**: Create deployment workflow documentation
- [ ] **7.3.5**: Finalize package for production use

---

## 📊 **TASK SUMMARY**

### **Total Tasks**: 35 individual tasks across 7 phases
### **Estimated Time**: 5 hours 47 minutes total
### **Dependencies**: Each phase builds on previous phases
### **Validation**: Each task must be completed and verified before proceeding

---

## 🚨 **CRITICAL SUCCESS FACTORS**

### **✅ NO SHORTCUTS ALLOWED**
- Every binary must be verified and tested
- Every configuration must be validated
- Every script function must be implemented
- Every service must be properly configured

### **✅ NO SIMULATIONS**
- All binaries must be real production versions
- All configurations must be production-ready
- All tests must use real functionality
- All documentation must be complete

### **✅ COMPLETE BEFORE TESTING**
- Package must be 100% complete before any deployment testing
- Script must pass all validation checks before testing
- Documentation must be comprehensive before testing
- All tasks must be ticked off before deployment validation

---

## 🎯 **READY TO PROCEED**

**This job sheet provides complete task breakdown for creating a production-ready VMA deployment package.**

**Each task will be completed and verified before moving to the next.**

**No deployment testing until ALL 35 tasks are completed and verified.**

**Agreed to proceed with this comprehensive approach?** 🔥
