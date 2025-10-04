# VMA Deployment Guide

**Version**: 2.0  
**Date**: September 30, 2025  
**Architecture**: Enhanced SSH Tunnel  

---

## 📖 **OVERVIEW**

This guide provides complete instructions for deploying production VMware Migration Appliances (VMAs) using the automated deployment script. The system creates fully functional VMA appliances with enhanced SSH tunnel architecture, professional setup wizards, and complete migration capabilities.

---

## 🎯 **DEPLOYMENT ARCHITECTURE**

### **Enhanced SSH Tunnel System**
```
┌─── VMA ─────────────────────────────────────┐
│                                              │
│  Enhanced SSH Tunnel Service                 │
│    → pgrayson@10.245.246.125:443           │
│                                              │
│  Forward Tunnels (VMA → OMA):               │
│    localhost:8082  → OMA API :8082          │  Change IDs
│    localhost:10809 → OMA NBD :10809         │  NBD Primary
│    localhost:10808 → OMA NBD :10809         │  NBD Alternate
│                                              │
│  Reverse Tunnel (OMA → VMA):                │
│    OMA:9081 → VMA API :8081                 │  Progress Polling
│                                              │
│  Auto-Recovery Features:                     │
│    • Health checks every 60 seconds         │
│    • Automatic tunnel restart on failure    │
│    • Connection monitoring and logging      │
│                                              │
└──────────────────────────────────────────────┘

┌─── OMA ─────────────────────────────────────┐
│                                              │
│  SSH Daemon                                  │
│    Listen: 10.245.246.125:443              │
│    User: pgrayson (development)             │
│                                              │
│  Services:                                   │
│    OMA API: localhost:8082                  │
│    NBD Server: localhost:10809              │
│    GUI: localhost:3001                      │
│                                              │
└──────────────────────────────────────────────┘
```

### **Key Features**
- **Single Port**: All traffic over port 443 (internet-safe)
- **Bidirectional**: Both forward and reverse tunnels
- **Auto-Recovery**: Enhanced tunnel service with health monitoring
- **Professional UI**: Setup wizard with network configuration
- **Production Ready**: Complete systemd integration

---

## 📦 **DEPLOYMENT PROCEDURE**

### **Prerequisites**
- Target VMA with Ubuntu 24.04+ and SSH access
- OMA with SSH daemon on port 443
- Network connectivity between VMA and OMA
- Required deployment files (see File Requirements)

### **File Requirements**
Copy these files to target VMA `/tmp/` directory before deployment:

| File | Source Location | Purpose |
|------|----------------|---------|
| `cloudstack_key*` | `~/.ssh/cloudstack_key*` | SSH keys for OMA access |
| `migratekit-v2.21.0-*` | `~/migratekit-cloudstack/source/current/migratekit/` | Migration binary |
| `nbdkit-vddk-stack.tar.gz` | `~/migratekit-cloudstack/vma-dependencies/` | NBD + VDDK libraries |
| `vma-api-server` | Copy from working VMA | VMA API server binary |
| `vma-setup-wizard.sh` | Copy from working VMA | Professional setup wizard |
| `deploy-production-vma-with-enrollment.sh` | `~/migratekit-cloudstack/scripts/` | Deployment script |

### **Deployment Steps**

#### **Step 1: Prepare Files on OMA**
```bash
cd /home/pgrayson

# Gather deployment files
cp ~/.ssh/cloudstack_key* /tmp/
cp migratekit-cloudstack/source/current/migratekit/migratekit-v2.21.0-hierarchical-sparse-optimization /tmp/
cp migratekit-cloudstack/vma-dependencies/nbdkit-vddk-stack.tar.gz /tmp/
cp migratekit-cloudstack/scripts/deploy-production-vma-with-enrollment.sh /tmp/

# Copy VMA API server from working VMA
scp -i ~/.ssh/cloudstack_key pgrayson@WORKING_VMA_IP:/home/pgrayson/migratekit-cloudstack/vma-api-server-v1.11.0-enrollment-system /tmp/vma-api-server

# Copy setup wizard from working VMA
scp -i ~/.ssh/cloudstack_key pgrayson@WORKING_VMA_IP:/opt/vma/setup-wizard.sh /tmp/vma-setup-wizard.sh
```

#### **Step 2: Deploy to Target VMA**
```bash
# Copy all files to target VMA
scp /tmp/{cloudstack_key*,migratekit-v2.21.0-*,nbdkit-vddk-stack.tar.gz,vma-api-server,vma-setup-wizard.sh,deploy-production-vma-with-enrollment.sh} vma@TARGET_VMA_IP:/tmp/

# Run deployment
ssh vma@TARGET_VMA_IP "sudo bash /tmp/deploy-production-vma-with-enrollment.sh"
```

#### **Step 3: Post-Deployment Validation**
```bash
# Reboot VMA - should auto-load setup wizard
ssh vma@TARGET_VMA_IP "sudo reboot"

# Wait for boot, then check console for wizard
# Configure OMA IP: 10.245.246.125
# Test connectivity through wizard interface

# Verify services
ssh vma@TARGET_VMA_IP "systemctl status vma-api vma-tunnel-enhanced-v2 vma-autologin"

# Test tunnel connectivity
curl http://127.0.0.1:9081/api/v1/health  # OMA → VMA reverse tunnel
```

---

## 🔧 **CONFIGURATION**

### **VMA Configuration File**
**Location**: `/opt/vma/vma-config.conf`
```ini
# VMA Configuration (created by setup wizard)
OMA_HOST=10.245.246.125
OMA_PORT=443
VMA_API_PORT=8081
TUNNEL_TYPE=ssh
SETUP_DATE=<timestamp>
SETUP_VERSION=v2.0.0
```

### **Enhanced Tunnel Configuration**
**Script**: `/opt/vma/scripts/enhanced-ssh-tunnel-remote.sh`
**Service**: `vma-tunnel-enhanced-v2.service`
**Log**: `/var/log/vma-tunnel-enhanced.log`

**Key Parameters**:
- Target: `pgrayson@10.245.246.125:443`
- SSH Key: `/home/vma/.ssh/cloudstack_key`
- Health Check Interval: 60 seconds
- Auto-restart on failure: 10 second delay

### **VMA API Server**
**Binary**: `/opt/vma/bin/vma-api-server`
**Service**: `vma-api.service`
**Port**: 8081
**Features**: Job management, progress tracking, NBD export verification

---

## 🚨 **TROUBLESHOOTING**

### **Common Issues**

#### **Deployment Fails - Missing Files**
```bash
# Error: Files not found in /tmp/
# Solution: Ensure all 6 required files copied to /tmp/ before deployment
ls -la /tmp/{cloudstack_key*,migratekit-v2.21.0-*,nbdkit-vddk-stack.tar.gz,vma-api-server,vma-setup-wizard.sh}
```

#### **Tunnel Service Fails**
```bash
# Check tunnel logs
sudo journalctl -u vma-tunnel-enhanced-v2 -f

# Check SSH key permissions
ls -la /home/vma/.ssh/cloudstack_key
# Should be: -rw------- vma vma

# Test SSH connectivity
ssh -i /home/vma/.ssh/cloudstack_key pgrayson@10.245.246.125
```

#### **VMA API Not Responding**
```bash
# Check VMA API service
sudo systemctl status vma-api

# Check if migratekit binary exists
ls -la /opt/vma/bin/migratekit

# Check compatibility paths
ls -la /home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel
```

#### **Wizard Not Loading on Console**
```bash
# Check auto-login service
sudo systemctl status vma-autologin

# Check if getty disabled
sudo systemctl status getty@tty1

# Manual wizard start
sudo /opt/vma/setup-wizard.sh
```

### **Log Files**
- **Tunnel**: `/var/log/vma-tunnel-enhanced.log`
- **VMA API**: `sudo journalctl -u vma-api`
- **Auto-login**: `sudo journalctl -u vma-autologin`
- **migratekit**: `/tmp/migratekit-job-*.log`

---

## 📈 **PERFORMANCE EXPECTATIONS**

### **Deployment Time**
- **Package Installation**: 2-5 minutes (depends on internet speed)
- **Binary Deployment**: 1-2 minutes
- **Service Configuration**: < 1 minute
- **Total Deployment**: 5-10 minutes

### **Runtime Performance**
- **Tunnel Establishment**: 5-15 seconds
- **Health Check Interval**: 60 seconds
- **Auto-restart Delay**: 10 seconds on failure
- **Migration Speed**: Depends on tunnel performance (testing required)

---

## 🔒 **SECURITY CONSIDERATIONS**

### **Current Security Model**
- **SSH Key**: RSA 2048-bit (cloudstack_key)
- **SSH User**: pgrayson (development user)
- **Tunnel Port**: 443 (standard HTTPS - internet-safe)
- **VMA User**: passwordless sudo (required for wizard)

### **Production Hardening (Future)**
- **SSH Key**: Ed25519 (stronger cryptography)
- **SSH User**: vma_tunnel (restricted user)
- **Minimal Privileges**: Remove passwordless sudo
- **Audit Logging**: Enhanced security monitoring

---

## 📋 **MAINTENANCE**

### **Regular Tasks**
- **Monitor tunnel health**: Check `/var/log/vma-tunnel-enhanced.log`
- **Service health**: `systemctl status vma-*`
- **Disk space**: Monitor `/opt/vma/` and `/var/log/`
- **SSH key rotation**: Update keys as needed

### **Updates**
- **migratekit**: Replace `/opt/vma/bin/migratekit` and restart services
- **VMA API**: Replace `/opt/vma/bin/vma-api-server` and restart
- **Tunnel Script**: Update `/opt/vma/scripts/enhanced-ssh-tunnel-remote.sh`

---

## ✅ **DEPLOYMENT CHECKLIST**

### **Pre-Deployment**
- [ ] All 6 required files copied to VMA `/tmp/`
- [ ] Target VMA has SSH access and sudo capability
- [ ] OMA SSH daemon listening on port 443
- [ ] Network connectivity verified

### **During Deployment**
- [ ] All phases complete successfully (1-11)
- [ ] No error messages in deployment output
- [ ] All services enabled and started

### **Post-Deployment**
- [ ] VMA reboots to setup wizard on console
- [ ] OMA IP configuration successful (10.245.246.125)
- [ ] Tunnel connectivity verified
- [ ] VMA API health check passes
- [ ] Ready for migration testing

---

**The VMA deployment system provides automated creation of production-ready migration appliances with enhanced tunnel architecture and professional management interfaces.**
