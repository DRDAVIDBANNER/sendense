# 🔧 **NBD Server Enhanced Build Plan - Docker Approach**

**Created**: September 26, 2025  
**Priority**: 🔥 **CRITICAL** - Fixes post-failback replication failures  
**Issue**: NBD server memory state desynchronization after volume operations  
**Solution**: Docker-based build of enhanced NBD server with memory cache flush

---

## 🎯 **EXECUTIVE SUMMARY**

**Problem**: NBD server holds stale exports in memory after volume failover/failback operations, causing "Access denied" errors for subsequent replication jobs.

**Solution**: Build enhanced NBD server with SIGHUP cache flush capability using Docker build environment, deploy as drop-in replacement for production use.

**Outcome**: Proper NBD memory synchronization without production architecture changes.

---

## 🏗️ **IMPLEMENTATION PLAN**

### **Phase 1: Docker Build Environment (30 minutes)**

#### **Task 1.1: Create Dockerfile**
```dockerfile
FROM ubuntu:22.04

# Install NBD server build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    libglib2.0-dev \
    autotools-dev \
    autoconf \
    libtool \
    git \
    wget

# Create build directory
WORKDIR /build

# Download NBD server source (same version as current)
RUN wget http://archive.ubuntu.com/ubuntu/pool/universe/n/nbd/nbd_3.24.orig.tar.xz && \
    tar -xf nbd_3.24.orig.tar.xz && \
    cd nbd-3.24

# Copy our patch and source modifications
COPY nbd-server-cache-flush.patch /build/
COPY nbd-server.c /build/nbd-3.24/

# Build enhanced NBD server
WORKDIR /build/nbd-3.24
RUN ./autogen.sh && \
    ./configure --enable-syslog && \
    make nbd-server

# Create output directory
RUN mkdir -p /output && cp nbd-server /output/nbd-server-enhanced
```

#### **Task 1.2: Prepare Build Context**
- Copy existing patch: `nbd-server-cache-flush.patch`
- Copy enhanced source: `nbd-server.c` 
- Verify patch applies to target NBD version

### **Phase 2: Enhanced Binary Build (15 minutes)**

#### **Task 2.1: Execute Docker Build**
```bash
cd source/current/nbd-server-enhanced
docker build -t nbd-server-builder .
```

#### **Task 2.2: Extract Binary**
```bash
# Create container and copy binary out
docker create --name nbd-builder nbd-server-builder
docker cp nbd-builder:/output/nbd-server-enhanced ./nbd-server-enhanced
docker rm nbd-builder
```

#### **Task 2.3: Binary Validation**
```bash
# Test binary functionality
./nbd-server-enhanced --help
ldd ./nbd-server-enhanced  # Check dependencies
```

### **Phase 3: Production Deployment (15 minutes)**

#### **Task 3.1: Backup Current Binary**
```bash
sudo cp /usr/bin/nbd-server /usr/bin/nbd-server.backup-$(date +%Y%m%d)
```

#### **Task 3.2: Deploy Enhanced Binary**
```bash
sudo cp nbd-server-enhanced /usr/bin/nbd-server
sudo chmod +x /usr/bin/nbd-server
```

#### **Task 3.3: Test Deployment**
```bash
# Test new binary (no restart yet)
/usr/bin/nbd-server --help

# Verify version/functionality
nbd-server -V
```

### **Phase 4: Production Integration (30 minutes)**

#### **Task 4.1: Service Restart (Scheduled)**
```bash
# During maintenance window:
sudo systemctl restart nbd-server
sudo systemctl status nbd-server
```

#### **Task 4.2: SIGHUP Enhancement Testing**
```bash
# Test enhanced SIGHUP functionality
sudo kill -HUP $(pgrep nbd-server)

# Verify cache flush in logs
sudo journalctl -u nbd-server --since "1 minute ago" | grep MIGRATEKIT
```

#### **Task 4.3: Integration with Volume Daemon**
```bash
# Add SIGHUP trigger to Volume Daemon when needed
# Future enhancement: Volume Daemon calls SIGHUP after volume operations
```

---

## 🔧 **TECHNICAL SPECIFICATIONS**

### **Enhanced NBD Server Features**

#### **SIGHUP Cache Flush (New):**
- **Trigger**: Standard `kill -HUP` signal
- **Action**: Refreshes existing export configurations + clears client cache
- **Logging**: MigrateKit-specific log messages for tracking
- **Backward Compatibility**: All existing functionality preserved

#### **Memory Synchronization:**
- **Export Refresh**: Re-reads device paths from configuration files
- **Client Cache Clear**: Forces client re-negotiation on next connection
- **Stale Export Removal**: Cleans up exports with no backing configuration

### **Production Integration Points**

#### **Volume Daemon Integration (Future):**
```go
// After volume operations that change device paths:
func (vs *VolumeService) triggerNBDMemorySync() {
    cmd := exec.Command("sudo", "kill", "-HUP", "$(pgrep nbd-server)")
    cmd.Run()
}
```

#### **Cleanup Service Integration (Future):**
```go
// After failover/failback operations:
func (ecs *EnhancedCleanupService) syncNBDServerMemory(ctx context.Context) {
    // Trigger SIGHUP to clear stale exports
}
```

---

## 🎯 **SUCCESS CRITERIA**

### **Build Validation**
- [ ] Docker build completes successfully
- [ ] Enhanced binary extracted without errors
- [ ] Binary passes functionality tests
- [ ] Dependencies are properly linked

### **Deployment Validation**
- [ ] Enhanced binary replaces system binary  
- [ ] NBD server starts with enhanced binary
- [ ] All existing exports continue working
- [ ] SIGHUP triggers enhanced cache flush behavior

### **Production Validation**
- [ ] Post-failback replication jobs succeed without restart
- [ ] SIGHUP clears stale exports from memory
- [ ] No operational disruption to active migrations
- [ ] Memory state stays synchronized with database/config

---

## ⚠️ **RISK ASSESSMENT**

| **Risk Level** | **Description** | **Mitigation** |
|----------------|-----------------|----------------|
| 🟢 **LOW** | Docker build fails | Existing patch and source already tested |
| 🟢 **LOW** | Binary incompatibility | Test extensively before deployment |
| 🟡 **MEDIUM** | Service disruption during deployment | Deploy during maintenance window |
| 🟡 **MEDIUM** | Enhanced SIGHUP breaks existing functionality | Preserve all original SIGHUP behavior |

---

## 📅 **TIMELINE ESTIMATE**

| **Phase** | **Duration** | **Dependencies** | **Risk** |
|-----------|--------------|------------------|----------|
| **Phase 1**: Docker Build Environment | 30 min | Docker available | 🟢 Low |
| **Phase 2**: Enhanced Binary Build | 15 min | Phase 1 complete | 🟢 Low |
| **Phase 3**: Production Deployment | 15 min | Binary validated | 🟡 Medium |
| **Phase 4**: Production Integration | 30 min | Service restart | 🟡 Medium |
| **Total** | **~90 minutes** | No active migrations | 🟡 **MEDIUM** |

---

## 🚀 **DEPLOYMENT STRATEGY**

### **Safety-First Approach**
1. **Build and validate** enhanced binary completely
2. **Test functionality** thoroughly before deployment  
3. **Deploy during maintenance window** when no jobs are active
4. **Keep backup binary** for immediate rollback if needed
5. **Validate production behavior** before declaring success

### **Rollback Plan**
```bash
# If issues arise:
sudo systemctl stop nbd-server
sudo cp /usr/bin/nbd-server.backup-YYYYMMDD /usr/bin/nbd-server  
sudo systemctl start nbd-server
```

---

## 🎯 **EXPECTED OUTCOME**

**Post-Implementation:**
- ✅ **Failover/Failback cycles** won't break subsequent replication jobs
- ✅ **NBD memory state** stays synchronized with database/configuration  
- ✅ **SIGHUP capability** for memory cleanup without service restart
- ✅ **Production reliability** for enterprise multi-volume operations

**This resolves the final critical issue** blocking enterprise-grade multi-volume snapshot protection reliability.

---

**Ready to proceed with this plan?** 🚀

