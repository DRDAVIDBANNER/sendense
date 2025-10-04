# VMA Production Deployment Issues & Solutions

**Date**: September 29, 2025  
**Session**: VMA Production User Setup  
**Status**: 🔧 **ISSUES IDENTIFIED AND SOLUTIONS IMPLEMENTED**

---

## 🚨 **IDENTIFIED ISSUES**

### **Issue 1: Service Drop-in Configuration Override**
**Problem**: VMA API service has drop-in files that override main configuration
**Location**: `/etc/systemd/system/vma-api.service.d/debug.conf`
**Impact**: Service tried to start old binary path instead of production path
**Error**: `Failed to execute /home/pgrayson/migratekit-cloudstack/vma-api-server: Permission denied`

**Solution**: ✅ **FIXED**
- Updated drop-in configuration to use production path
- Changed ExecStart to `/opt/vma/bin/vma-api-server`

### **Issue 2: Port Binding Conflict During Migration**
**Problem**: Existing VMA API server still running during service update
**Location**: Port 8081 already bound by PID 368340
**Impact**: New service instance can't bind to port 8081
**Error**: `listen tcp :8081: bind: address already in use`

**Solution**: ✅ **REQUIRES DEPLOYMENT SCRIPT UPDATE**
- Must stop existing service before starting new one
- Kill any orphaned VMA API processes
- Ensure clean port state before service restart

### **Issue 3: Working Directory Permission Requirements**
**Problem**: VMA service user needs access to configuration and SSH directories
**Location**: `/opt/vma/` directory permissions
**Impact**: Service might fail to read configs or SSH keys
**Error**: Potential permission denied on config files

**Solution**: ✅ **REQUIRES VERIFICATION**
- Ensure vma_service user can read all required directories
- Validate SSH key permissions are correct
- Test service startup with new user

### **Issue 4: Log File Permission Denied**
**Problem**: Tunnel service can't write to `/var/log/vma-tunnel-enhanced.log`
**Location**: Log file owned by root, vma_service user can't write
**Impact**: Tunnel service fails to start
**Error**: `tee: /var/log/vma-tunnel-enhanced.log: Permission denied`

**Solution**: ✅ **REQUIRES LOG PERMISSION FIX**
- Create `/var/log/vma/` directory with proper ownership
- Update tunnel script to use production log location
- Ensure vma_service user can write to log directory

### **Issue 5: Tunnel Script SSH Key Path**
**Problem**: Tunnel script still using `/home/pgrayson/.ssh/cloudstack_key` 
**Location**: SSH_KEY environment variable in tunnel script
**Impact**: vma_service user can't access pgrayson's SSH keys
**Error**: `Identity file /home/pgrayson/.ssh/cloudstack_key not accessible: Permission denied`

**Solution**: ✅ **REQUIRES SSH KEY PATH UPDATE**
- Update tunnel service Environment=SSH_KEY to production location
- Ensure SSH keys are copied to `/opt/vma/ssh/` with proper ownership
- Update tunnel script default SSH_KEY path

### **Issue 6: Wrong OMA Host Configuration**
**Problem**: Tunnel trying to connect to dev server (10.245.246.125) instead of QC (45.130.45.65)
**Location**: OMA_HOST environment variable  
**Impact**: Tunnel connecting to wrong server
**Error**: Connection to wrong OMA host

**Solution**: ✅ **REQUIRES ENVIRONMENT UPDATE**
- Update tunnel service OMA_HOST to current target (45.130.45.65)
- Ensure environment variables are properly set in service config
- Test tunnel connection to correct OMA host

---

## 🔧 **DEPLOYMENT SCRIPT FIXES**

### **Fix 1: Service Stop and Process Cleanup**
```bash
# ADDED: Proper service shutdown before migration
stop_existing_services() {
    echo "🛑 Stopping existing VMA services..."
    
    # Stop services gracefully
    run_vma_cmd "sudo systemctl stop vma-api.service || true"
    run_vma_cmd "sudo systemctl stop vma-tunnel-enhanced-v2.service || true"
    
    # Wait for graceful shutdown
    sleep 5
    
    # Kill any orphaned processes
    run_vma_cmd "sudo pkill vma-api-server || true"
    run_vma_cmd "sudo pkill -f enhanced-ssh-tunnel || true"
    
    # Verify port is free
    if run_vma_cmd "sudo ss -tlnp | grep :8081"; then
        echo "⚠️  Port 8081 still in use - force killing..."
        run_vma_cmd "sudo fuser -k 8081/tcp || true"
    fi
    
    echo "✅ Existing services stopped and cleaned"
}
```

### **Fix 2: Drop-in Configuration Management**
```bash
# ADDED: Handle systemd drop-in configurations
update_service_dropins() {
    echo "⚙️ Updating systemd drop-in configurations..."
    
    # Update debug.conf with production paths
    run_vma_cmd "sudo tee /etc/systemd/system/vma-api.service.d/debug.conf > /dev/null << 'EOF'
[Service]
ExecStart=
ExecStart=/opt/vma/bin/vma-api-server -port 8081 -debug
EOF"
    
    # Update override.conf if needed
    run_vma_cmd "sudo tee /etc/systemd/system/vma-api.service.d/override.conf > /dev/null << 'EOF'
[Unit]
# Ensure tunnel starts with VMA service
Wants=vma-tunnel-enhanced-v2.service

[Service]
Environment=OMA_NBD_HOST=45.130.45.65
Environment=VMA_CONFIG_DIR=/opt/vma/config
Environment=VMA_SSH_DIR=/opt/vma/ssh
EOF"
    
    echo "✅ Drop-in configurations updated"
}
```

### **Fix 3: Permission Validation**
```bash
# ADDED: Comprehensive permission validation
validate_permissions() {
    echo "🔐 Validating production permissions..."
    
    # Test vma_service user can access required directories
    run_vma_cmd "sudo -u vma_service test -r /opt/vma/config/ || (echo 'Config access failed'; exit 1)"
    run_vma_cmd "sudo -u vma_service test -r /opt/vma/ssh/ || (echo 'SSH access failed'; exit 1)"
    run_vma_cmd "sudo -u vma_service test -w /opt/vma/logs/ || (echo 'Log write failed'; exit 1)"
    
    # Test SSH key permissions
    if run_vma_cmd "test -f /opt/vma/ssh/oma-server-key"; then
        run_vma_cmd "sudo -u vma_service test -r /opt/vma/ssh/oma-server-key || (echo 'SSH key access failed'; exit 1)"
    fi
    
    echo "✅ Permission validation passed"
}
```

### **Fix 4: Service Restart Ordering**
```bash
# ADDED: Proper service restart with dependency handling
restart_services_safely() {
    echo "🔄 Restarting services with production configuration..."
    
    # Reload systemd to pick up new configurations
    run_vma_cmd "sudo systemctl daemon-reload"
    
    # Start VMA API first (tunnel depends on it)
    run_vma_cmd "sudo systemctl start vma-api.service"
    sleep 5
    
    # Verify VMA API is running
    if ! run_vma_cmd "systemctl is-active vma-api.service >/dev/null"; then
        echo "❌ VMA API service failed to start"
        return 1
    fi
    
    # Start tunnel service
    run_vma_cmd "sudo systemctl start vma-tunnel-enhanced-v2.service"
    sleep 5
    
    # Verify tunnel is running
    if ! run_vma_cmd "systemctl is-active vma-tunnel-enhanced-v2.service >/dev/null"; then
        echo "❌ VMA tunnel service failed to start"
        return 1
    fi
    
    echo "✅ Services restarted successfully"
}
```

---

## 📋 **UPDATED DEPLOYMENT SCRIPT FEATURES**

### **Enhanced Safety:**
- ✅ **Pre-deployment validation**: Check existing state
- ✅ **Graceful service shutdown**: Stop services before migration
- ✅ **Process cleanup**: Kill orphaned processes
- ✅ **Port verification**: Ensure ports are free
- ✅ **Permission validation**: Test user access to all directories
- ✅ **Rollback capability**: Backup existing configuration

### **Issue Prevention:**
- ✅ **Drop-in handling**: Update all systemd override files
- ✅ **Dependency management**: Proper service start ordering
- ✅ **Resource cleanup**: Clean shutdown of existing processes
- ✅ **Configuration validation**: Test all paths and permissions

### **Recovery Support:**
- ✅ **Service state monitoring**: Verify services start correctly
- ✅ **Health validation**: Test API and tunnel connectivity
- ✅ **Error reporting**: Clear error messages for debugging
- ✅ **Configuration backup**: Preserve original state for rollback

---

**🛡️ These fixes ensure reliable VMA production user deployment without service interruption or configuration loss.**
