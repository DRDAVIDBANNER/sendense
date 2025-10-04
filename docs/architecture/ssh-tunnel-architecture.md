# SSH Tunnel Architecture - Enhanced Design

**Version**: 2.0  
**Date**: September 30, 2025  
**Status**: ✅ **PRODUCTION READY**  
**Architecture**: Enhanced SSH Tunnel with Auto-Recovery  

---

## 🎯 **ARCHITECTURE OVERVIEW**

The Enhanced SSH Tunnel Architecture provides secure, reliable communication between VMware Migration Appliances (VMAs) and OSSEA Management Appliances (OMAs) using hardened SSH tunnels over port 443. This architecture replaces previous designs and provides enterprise-grade reliability with auto-recovery capabilities.

---

## 📊 **NETWORK TOPOLOGY**

### **Complete Architecture Diagram**
```
┌─── VMA (Migration Appliance) ───────────────────────────────┐
│                                                              │
│  ┌─ Enhanced SSH Tunnel Service ─────────────────────────┐  │
│  │                                                        │  │
│  │  Target: pgrayson@10.245.246.125:443                 │  │
│  │  SSH Key: /home/vma/.ssh/cloudstack_key              │  │
│  │  Health Monitoring: 60-second intervals               │  │
│  │  Auto-Recovery: Restart on failure                    │  │
│  │                                                        │  │
│  │  Forward Tunnels (VMA → OMA):                         │  │
│  │    localhost:8082  → OMA API :8082                   │  │  Change IDs
│  │    localhost:10809 → OMA NBD :10809                  │  │  NBD Primary  
│  │    localhost:10808 → OMA NBD :10809                  │  │  NBD Alternate
│  │                                                        │  │
│  │  Reverse Tunnel (OMA → VMA):                          │  │
│  │    OMA:9081 → VMA API :8081                          │  │  Progress
│  │                                                        │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌─ VMA Services ─────────────────────────────────────────┐  │
│  │                                                        │  │
│  │  VMA API Server: localhost:8081                       │  │
│  │  migratekit: /opt/vma/bin/migratekit                  │  │
│  │  Setup Wizard: /opt/vma/setup-wizard.sh              │  │
│  │  NBD Tools: nbdkit + nbd-client + libnbd             │  │
│  │                                                        │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
└──────────────────────────────────────────────────────────────┘

┌─── OMA (Management Appliance) ──────────────────────────────┐
│                                                              │
│  ┌─ SSH Daemon ───────────────────────────────────────────┐  │
│  │                                                        │  │
│  │  Listen: 10.245.246.125:443                          │  │
│  │  User: pgrayson (development access)                  │  │
│  │  Restrictions: Standard SSH access                    │  │
│  │                                                        │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌─ Backend Services ─────────────────────────────────────┐  │
│  │                                                        │  │
│  │  OMA API: localhost:8082                              │  │
│  │  NBD Server: localhost:10809                          │  │
│  │  Migration GUI: localhost:3001                        │  │
│  │  Database: MariaDB (migratekit_oma)                   │  │
│  │                                                        │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

## 🔧 **ENHANCED TUNNEL COMPONENTS**

### **VMA Enhanced SSH Tunnel Service**
**Service**: `vma-tunnel-enhanced-v2.service`  
**Script**: `/opt/vma/scripts/enhanced-ssh-tunnel-remote.sh`  
**User**: vma  
**Restart**: Automatic with 10-second delay  

**Features**:
- **Health Monitoring**: Tests OMA API connectivity every 60 seconds
- **Auto-Recovery**: Automatically restarts failed tunnels
- **Process Management**: Cleans up stale SSH processes
- **Comprehensive Logging**: Detailed tunnel status and error reporting
- **Signal Handling**: Graceful shutdown on termination

### **SSH Tunnel Configuration**
```bash
ssh -i /home/vma/.ssh/cloudstack_key \
    -p 443 \
    -R 9081:localhost:8081 \          # OMA → VMA API (reverse)
    -L 8082:localhost:8082 \          # VMA → OMA API (forward)
    -L 10809:localhost:10809 \        # VMA → OMA NBD (forward)
    -L 10808:localhost:10809 \        # VMA → OMA NBD (alternate)
    -N \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=3 \
    -o ConnectTimeout=30 \
    -o TCPKeepAlive=yes \
    -o ExitOnForwardFailure=yes \
    -o BatchMode=yes \
    "pgrayson@10.245.246.125"
```

### **Health Check Logic**
```bash
health_check() {
    local test_url="http://localhost:8082/health"
    if curl --connect-timeout 5 --max-time 10 -s "$test_url" >/dev/null 2>&1; then
        return 0  # Tunnel healthy
    else
        return 1  # Tunnel failed
    fi
}
```

---

## 📋 **TUNNEL MANAGEMENT**

### **Service Management**
```bash
# Start tunnel
sudo systemctl start vma-tunnel-enhanced-v2

# Stop tunnel  
sudo systemctl stop vma-tunnel-enhanced-v2

# Restart tunnel
sudo systemctl restart vma-tunnel-enhanced-v2

# Check status
sudo systemctl status vma-tunnel-enhanced-v2

# View logs
sudo tail -f /var/log/vma-tunnel-enhanced.log
```

### **Manual Tunnel Control**
```bash
# Kill all SSH tunnels
sudo pkill -f "ssh.*10.245.246.125"

# Start tunnel manually
sudo -u vma /opt/vma/scripts/enhanced-ssh-tunnel-remote.sh

# Test tunnel connectivity
curl http://localhost:8082/health  # VMA → OMA API
curl http://localhost:8081/health  # VMA API (via reverse tunnel from OMA)
```

### **Troubleshooting**
```bash
# Check SSH key permissions
ls -la /home/vma/.ssh/cloudstack_key
# Should be: -rw------- vma vma

# Test SSH connectivity
sudo -u vma ssh -i /home/vma/.ssh/cloudstack_key pgrayson@10.245.246.125

# Check port bindings
sudo ss -tlnp | grep -E ':8082|:10808|:10809'

# Monitor tunnel process
ps aux | grep ssh | grep 10.245.246.125
```

---

## 🔒 **SECURITY MODEL**

### **Current Security Configuration**
- **SSH Port**: 443 (standard HTTPS port - internet-safe)
- **Authentication**: RSA key-based (cloudstack_key)
- **SSH User**: pgrayson (development user with standard access)
- **Tunnel Binding**: localhost only (not exposed to network)
- **Process User**: vma (restricted service user)

### **Security Features**
- **No PTY**: SSH tunnel runs without terminal access
- **Batch Mode**: Non-interactive SSH execution
- **Key-based Auth**: No password authentication
- **Local Binding**: Tunnel ports bound to localhost only
- **Process Isolation**: Services run as vma user

---

## 📊 **MONITORING & LOGGING**

### **Log Files**
- **Tunnel Activity**: `/var/log/vma-tunnel-enhanced.log`
- **Service Status**: `journalctl -u vma-tunnel-enhanced-v2`
- **VMA API**: `journalctl -u vma-api`
- **Migration Jobs**: `/tmp/migratekit-job-*.log`

### **Monitoring Points**
- **Tunnel Health**: Health checks every 60 seconds
- **Service Status**: Systemd monitoring with auto-restart
- **Connection State**: SSH process monitoring
- **Port Availability**: Tunnel port binding verification

### **Alert Conditions**
- **Tunnel Failure**: Health check fails 3 consecutive times
- **Service Crash**: VMA API or tunnel service exits
- **SSH Authentication**: Key-based authentication failures
- **Port Conflicts**: Tunnel port binding failures

---

## 🔄 **RECOVERY PROCEDURES**

### **Automatic Recovery**
The enhanced tunnel service automatically handles:
- **SSH Process Death**: Detects and restarts tunnel
- **Network Interruption**: Reconnects on network recovery
- **OMA Unavailability**: Retries connection every 10 seconds
- **Port Conflicts**: Cleans up stale processes

### **Manual Recovery**
```bash
# Full tunnel reset
sudo systemctl stop vma-tunnel-enhanced-v2
sudo pkill -f "ssh.*10.245.246.125"
sudo systemctl start vma-tunnel-enhanced-v2

# Service recovery
sudo systemctl daemon-reload
sudo systemctl restart vma-api vma-tunnel-enhanced-v2

# SSH key recovery
sudo chown vma:vma /home/vma/.ssh/cloudstack_key
sudo chmod 600 /home/vma/.ssh/cloudstack_key
```

---

## 📚 **RELATED DOCUMENTATION**

- **Deployment Guide**: `docs/deployment/vma-deployment-guide.md`
- **Workarounds**: `AI_Helper/VMA_DEPLOYMENT_WORKAROUNDS_TO_FIX.md`
- **Network Topology**: `docs/architecture/network-topology.md`
- **Project Status**: `AI_Helper/PROJECT_STATUS.md`

---

## 🎯 **ARCHITECTURE STATUS**

**Current State**: ✅ **PRODUCTION FUNCTIONAL**  
**Performance**: Proven working (adapted from QC OMA)  
**Reliability**: Auto-recovery with health monitoring  
**Security**: Port 443 only, key-based authentication  
**Management**: Professional setup wizard interface  

**The Enhanced SSH Tunnel Architecture provides enterprise-grade VMA-OMA connectivity with comprehensive auto-recovery and monitoring capabilities.**

---

**Last Updated**: September 30, 2025  
**Tested Environment**: VMA 233 → Dev OMA (10.245.246.125)  
**Status**: ✅ **OPERATIONAL**
