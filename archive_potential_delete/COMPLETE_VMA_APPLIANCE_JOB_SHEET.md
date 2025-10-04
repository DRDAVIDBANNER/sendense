# Complete VMA Appliance Deployment - Job Sheet

**Created**: September 29, 2025  
**Priority**: 🔥 **HIGH** - Complete functional VMA appliance with enrollment system  
**Target**: 10.0.100.232 → Fresh VMA deployment → VMA template  
**Estimated Duration**: 4-6 hours  

---

## 🎯 **PROJECT OVERVIEW**

### **Current Status**
- **Target VMA**: 10.0.100.232 (Ubuntu VM with enrollment system)
- **Enrollment**: ✅ Working (scripts deployed, connects to dev OMA)
- **VMA Services**: ❌ Missing (no VMA API, no tunnel, no migration engine)
- **Goal**: Complete functional VMA appliance

### **Success Criteria**
- ✅ **Complete VMA functionality**: Migration engine, API, tunnel working
- ✅ **Enrollment integration**: NEW VMA enrollment via wizard
- ✅ **Auto-boot wizard**: Professional appliance experience
- ✅ **Documentation**: Complete template requirements
- ✅ **Deployment script**: Reproducible VMA appliance creation

---

## 📋 **IMPLEMENTATION PHASES**

### **🔧 PHASE 1: Complete Current VMA Appliance (10.0.100.232)** ⏱️ *Est: 2 hours*

#### **Task 1.1: Deploy VMA Core Binaries**
- [✅] **Copy MigrateKit binary** from working VMA (pg-migrationdev)
  - Source: `/usr/local/bin/migratekit-multidisk-incremental-fix` (20MB)
  - Target: `/opt/vma/bin/migratekit`
  - Symlink: `/usr/local/bin/migratekit`
  - **FIX DOCUMENTED**: File transfer issues - need fallback paths in script

- [✅] **Copy VMA API server** from working VMA
  - Source: `vma-api-server-v1.11.0-enrollment-system` (20MB)
  - Target: `/opt/vma/bin/vma-api-server` ✅ **DEPLOYED**
  - Symlink: `/usr/local/bin/vma-api-server`
  - **STATUS**: VMA API responding on port 8081

#### **Task 1.2: Configure VMA Services**
- [✅] **Deploy VMA API service** ✅ **RUNNING**
  ```ini
  [Unit]
  Description=VMA Control API Server
  [Service]
  User=vma
  ExecStart=/opt/vma/bin/vma-api-server -port 8081
  Restart=always
  ```

- [🔧] **Deploy VMA tunnel service** ✅ **CONFIGURED** ❌ **NOT CONNECTING**
  ```ini
  [Unit] 
  Description=VMA Enhanced SSH Tunnel to OMA
  [Service]
  User=vma
  ExecStart=/opt/vma/scripts/enhanced-ssh-tunnel.sh
  Environment=OMA_HOST=10.245.246.125
  Environment=SSH_KEY=/opt/vma/enrollment/vma_enrollment_key
  ```
  - **ISSUE**: SSH key not accessible by VMA user
  - **FIX NEEDED**: Copy SSH private key from OMA to VMA

#### **Task 1.3: Configure Tunnel System**
- [ ] **Create enhanced tunnel script**
  - Bidirectional tunnel: R 9081:8081, L 8082:8082, L 10809:10809
  - Health monitoring: 60-second checks
  - Auto-recovery: Restart on failure
  - Logging: Connection events and health

- [ ] **Copy SSH key for tunnel**
  - Use manual SSH key from dev OMA authorized_keys
  - Copy to VMA for tunnel authentication
  - Set proper permissions (600)

#### **Task 1.4: Test Complete VMA Functionality**
- [ ] **Start VMA API service** and test port 8081
- [ ] **Start tunnel service** and test OMA connectivity
- [ ] **Test enrollment workflow** end-to-end
- [ ] **Validate auto-boot wizard** works

### **🚀 PHASE 2: Document Working Deployment** ⏱️ *Est: 1 hour*

#### **Task 2.1: Create Complete Deployment Script**
- [ ] **Enhanced deployment script** with all components
  - VMA binary deployment (MigrateKit + VMA API)
  - Service configuration (API + tunnel + autologin)
  - Enrollment system integration
  - Dependencies and system setup

#### **Task 2.2: Validate Deployment Script**
- [ ] **Test script syntax** and execution
- [ ] **Document script requirements** and dependencies
- [ ] **Add error handling** and validation steps
- [ ] **Commit enhanced script** to git repository

### **🧪 PHASE 3: Fresh VMA Appliance Deployment** ⏱️ *Est: 1-2 hours*

#### **Task 3.1: Deploy Fresh VMA Appliance**
- [ ] **Create new Ubuntu VM** for testing
- [ ] **Run complete deployment script** 
- [ ] **Validate all components** deployed correctly
- [ ] **Test enrollment workflow** on fresh appliance

#### **Task 3.2: Integration Testing**
- [ ] **Auto-boot wizard**: Test appliance boots to wizard
- [ ] **VMA enrollment**: Test NEW VMA enrollment workflow
- [ ] **Migration functionality**: Test basic VM discovery
- [ ] **Tunnel connectivity**: Test OMA connection and operations

### **📋 PHASE 4: VMA Template Documentation** ⏱️ *Est: 1 hour*

#### **Task 4.1: Document VMA Template Requirements**
- [ ] **Complete component list**: All binaries, services, scripts
- [ ] **System requirements**: Ubuntu version, dependencies
- [ ] **Configuration files**: Service configs, environment setup
- [ ] **Deployment procedure**: Step-by-step template creation

#### **Task 4.2: Create Template Creation Guide**
- [ ] **VMA image preparation**: OS setup and hardening
- [ ] **Component installation**: Binary and service deployment
- [ ] **Template finalization**: Image creation and validation
- [ ] **Deployment validation**: Template testing procedures

---

## 🧪 **VALIDATION CHECKLIST**

### **Current VMA Appliance (10.0.100.232)**
- [ ] ✅ **MigrateKit binary**: Working migration engine
- [ ] ✅ **VMA API**: Port 8081 responding with VM discovery
- [ ] ✅ **SSH tunnel**: Connected to dev OMA with health monitoring
- [ ] ✅ **Enrollment system**: Working wizard and enrollment workflow
- [ ] ✅ **Auto-boot**: Boots directly to setup wizard
- [ ] ✅ **Services**: All VMA services enabled and running

### **Deployment Script Validation**
- [ ] ✅ **Complete deployment**: All VMA components included
- [ ] ✅ **Error handling**: Proper validation and rollback
- [ ] ✅ **Reproducible**: Creates identical VMA appliances
- [ ] ✅ **Documentation**: Clear requirements and procedures

### **Fresh VMA Testing**
- [ ] ✅ **Clean deployment**: Script deploys complete VMA successfully
- [ ] ✅ **Enrollment workflow**: NEW VMA enrollment works end-to-end
- [ ] ✅ **Migration operations**: Basic VM discovery and operations
- [ ] ✅ **Production ready**: Appliance ready for enterprise deployment

---

## 📊 **COMPONENT BREAKDOWN**

### **VMA Binaries (~40MB)**
- **MigrateKit**: 20MB (migration engine)
- **VMA API**: 20MB (control and discovery)

### **System Dependencies**
- **libnbd-bin**: NBD client operations
- **haveged**: Entropy generation
- **jq, curl**: API communication
- **openssh-client**: Tunnel operations

### **VMA Scripts (<1MB)**
- **vma-enrollment.sh**: Enrollment workflow
- **setup-wizard.sh**: Professional interface
- **enhanced-ssh-tunnel.sh**: Tunnel with monitoring

### **VMA Services**
- **vma-api.service**: VMA control API
- **vma-tunnel-enhanced-v2.service**: SSH tunnel with recovery
- **vma-autologin.service**: Auto-boot to wizard

---

## 🎯 **EXECUTION PLAN**

### **Step 1: Fix Current VMA (10.0.100.232)**
Deploy missing binaries and services to make it a complete VMA appliance

### **Step 2: Document Working Deployment**
Create deployment script that reproduces the working VMA

### **Step 3: Test Fresh Deployment**
Deploy new VMA appliance from script to validate

### **Step 4: Create VMA Template**
Document template requirements for production VMA image creation

---

**🚀 Ready to execute systematic VMA appliance deployment with task tracking**
