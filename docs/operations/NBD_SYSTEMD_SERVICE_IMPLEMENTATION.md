# NBD Server Systemd Service Implementation

**Date**: 2025-09-01  
**Status**: ✅ **PRODUCTION READY**  
**Related**: [NBD Export Management Tasks](./NBD_EXPORT_MANAGEMENT_TASKS.md)

## 🎯 **Overview**

This document covers the complete implementation of NBD server as a systemd service and the resolution of dynamic export management issues that were discovered during production testing.

## 🚨 **Critical Issues Resolved**

### **Issue 1: NBD Server Process Management**
- **Problem**: NBD server was running as manual process, no service management
- **Impact**: No automatic startup, no logging, difficult troubleshooting
- **Solution**: Created full systemd service with logging integration

### **Issue 2: SIGHUP Mechanism Failure**  
- **Problem**: Volume Daemon couldn't send SIGHUP to reload NBD exports
- **Root Cause**: Volume Daemon used `pgrep` which failed with systemd process
- **Impact**: New exports created but never visible to VMA
- **Solution**: Fixed PID detection to read `/run/nbd-server.pid` file

### **Issue 3: Export Naming Conflicts**
- **Problem**: Multiple volumes generated identical export names
- **Root Cause**: Using `migration-vm-{VM_ID}-disk{NUMBER}` format
- **Impact**: Duplicate export errors, failed replication jobs  
- **Solution**: Changed to `migration-vol-{VOLUME_ID}` for guaranteed uniqueness

## 🔧 **Implementation Details**

### **Systemd Service Configuration**

**Service File**: `/etc/systemd/system/nbd-server.service`
```ini
[Unit]
Description=Network Block Device Server
Documentation=man:nbd-server(1) man:nbd-server(5)
After=network.target

[Service]
Type=forking
ExecStart=/usr/bin/nbd-server -C /etc/nbd-server/config
ExecReload=/bin/kill -HUP $MAINPID
PIDFile=/run/nbd-server.pid
Restart=always
RestartSec=5
TimeoutStartSec=30
TimeoutStopSec=30

# Logging configuration
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nbd-server

# Security settings
User=root
Group=root
ProtectSystem=false
ProtectHome=true
NoNewPrivileges=false
PrivateTmp=false

# Working directory
WorkingDirectory=/etc/nbd-server

# Environment
Environment=NBD_DEBUG=1

[Install]
WantedBy=multi-user.target
```

### **SIGHUP Mechanism Fix**

**Problem Code** (Volume Daemon):
```go
// Old: Failed with systemd
cmd := exec.Command("pgrep", "-f", "nbd-server.*"+filepath.Base(cm.configPath))
```

**Fixed Code**:
```go
// New: Reads systemd PID file with fallback
func (cm *ConfigManager) GetNBDServerPID() (int, error) {
    // First try reading systemd PID file
    pidFile := "/run/nbd-server.pid"
    if pidData, err := os.ReadFile(pidFile); err == nil {
        var pid int
        if _, err := fmt.Sscanf(strings.TrimSpace(string(pidData)), "%d", &pid); err == nil {
            // Verify the process is actually running
            if err := syscall.Kill(pid, 0); err == nil {
                return pid, nil
            }
        }
    }
    
    // Fallback to pgrep for backward compatibility
    cmd := exec.Command("pgrep", "-f", "nbd-server.*"+filepath.Base(cm.configPath))
    // ... rest of fallback logic
}
```

### **Export Naming Scheme Update**

**Old Format**: `migration-vm-{VM_ID}-disk{NUMBER}`
- Problem: Multiple volumes attached to same VM created duplicates
- Example: `migration-vm-8a4400e5-c92a-4bc4-8bff-4b6b0b6a018c-disk0`

**New Format**: `migration-vol-{VOLUME_ID}`  
- Solution: Each volume gets unique export name
- Example: `migration-vol-42e8fd44-028a-4169-a0d7-82ed70add839`

## 📊 **Operational Benefits**

### **Service Management**
```bash
# Service control
sudo systemctl start nbd-server
sudo systemctl stop nbd-server
sudo systemctl restart nbd-server
sudo systemctl reload nbd-server  # SIGHUP for export reload

# Status and logging
sudo systemctl status nbd-server
sudo journalctl -u nbd-server --follow
sudo journalctl -u nbd-server --since="1 hour ago"
```

### **Export Management**
- **Automatic Creation**: NBD exports created during volume attachment
- **Dynamic Loading**: SIGHUP reloads new exports without downtime
- **Unique Naming**: No more export name conflicts
- **Cleanup**: Exports removed when volumes detached

### **Monitoring Integration**
- **Systemd Status**: `systemctl status nbd-server`
- **Logs**: `journalctl -u nbd-server`
- **Health Checks**: NBD server responds to connections
- **Export Listing**: `nbd-client -l localhost`

## 🧪 **Testing Validation**

### **Concurrent Job Testing**
- ✅ **pgtest1**: Working with old naming scheme  
- ✅ **pgtest2**: Working with new naming scheme
- ✅ **PGWINTESTBIOS**: Working with new naming scheme
- ✅ **All 4 jobs running simultaneously**

### **SIGHUP Mechanism Validation**
- ✅ Volume Daemon can find NBD server PID from `/run/nbd-server.pid`
- ✅ SIGHUP sent successfully for new exports
- ✅ New exports visible immediately in `nbd-client -l localhost`
- ✅ No manual NBD server restart required

### **Export Naming Validation**
- ✅ No duplicate export name errors
- ✅ Each volume gets unique export
- ✅ Backward compatibility maintained for existing exports

## 🚀 **Production Impact**

### **Before Implementation**
- ❌ Manual NBD server process management
- ❌ No centralized logging
- ❌ SIGHUP mechanism broken with systemd
- ❌ Export naming conflicts causing job failures
- ❌ Manual intervention required for new exports

### **After Implementation**  
- ✅ Full systemd service integration
- ✅ Centralized logging via journalctl
- ✅ SIGHUP mechanism working with systemd
- ✅ Unique export naming prevents conflicts
- ✅ Automatic export management, zero manual intervention

## 📋 **Maintenance Procedures**

### **Service Management**
```bash
# Check service status
sudo systemctl status nbd-server

# View recent logs
sudo journalctl -u nbd-server --since="1 hour ago"

# Restart if needed (rare)
sudo systemctl restart nbd-server

# Reload for configuration changes
sudo systemctl reload nbd-server
```

### **Export Troubleshooting**
```bash
# List all exports
nbd-client -l localhost

# Check export config files
ls -la /etc/nbd-server/conf.d/

# Check Volume Daemon logs for export creation
sudo journalctl -u volume-daemon | grep "NBD export"
```

### **Database Monitoring**
```sql
-- Check current exports
SELECT export_name, volume_id, status, created_at 
FROM nbd_exports 
ORDER BY created_at DESC;

-- Check active volume attachments
SELECT volume_id, device_path, operation_mode 
FROM device_mappings 
WHERE operation_mode = 'oma';
```

## 🎯 **Key Achievements**

1. **🔧 Systemd Integration**: NBD server now managed as proper system service
2. **📊 Logging Infrastructure**: Complete logging via journalctl integration  
3. **🔄 Dynamic Export Management**: SIGHUP mechanism fixed and operational
4. **🎯 Unique Export Naming**: Volume-based naming eliminates conflicts
5. **⚡ Zero Downtime Operations**: New exports added without service interruption
6. **🚀 Production Ready**: System validated with concurrent migrations

---

**Status**: ✅ **FULLY OPERATIONAL**  
**Last Updated**: 2025-09-01  
**Next Review**: Production monitoring and performance optimization
