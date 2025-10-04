# 🔧 **DEPLOYMENT SCRIPT FIXES REQUIRED**

**Created**: October 1, 2025  
**Purpose**: Document all fixes needed for bulletproof OMA deployment script  
**Status**: ✅ **FIXES IDENTIFIED** - Ready for script integration

---

## 🚨 **CRITICAL FIXES DISCOVERED DURING OMAV3 DEPLOYMENT**

### **✅ WORKING PRODUCTION ENVIRONMENT ACHIEVED:**
- **OMAv3 (10.245.246.134)**: Complete production OMA deployed and operational
- **All Services**: GUI, OMA API, Volume Daemon, NBD Server, Database working
- **Real Components**: Actual production binaries and 34-table schema
- **VMA Pre-shared Key**: Real cloudstack_key.pub configured for tunnel

---

## 🔧 **SCRIPT FIXES REQUIRED**

### **1. PASSWORDLESS SUDO SETUP (CRITICAL)**
**Issue**: Multiple sudo password prompts cause script failures
**Fix**: Add passwordless sudo setup as Phase 1
```bash
# Early in script (before any other sudo commands)
echo "$SUDO_PASSWORD" | sudo -S sh -c 'echo "oma_admin ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/oma_admin'

# Update run_sudo function to use passwordless after setup
run_sudo() {
    if [ -f "/etc/sudoers.d/oma_admin" ]; then
        sudo "$@"
    else
        echo "$SUDO_PASSWORD" | sudo -S "$@"
    fi
}
```

### **2. MISSING SERVICE CREATION (CRITICAL)**
**Issue**: Script doesn't create systemd service files
**Fix**: Add service file creation for all components

#### **Migration GUI Service** (MISSING):
```ini
[Unit]
Description=Migration Dashboard GUI
After=network.target oma-api.service
Wants=oma-api.service

[Service]
Type=simple
User=oma_admin
Group=oma_admin
WorkingDirectory=/opt/migratekit/gui
ExecStart=/usr/bin/npx next start --port 3001 --hostname 0.0.0.0
Restart=always
RestartSec=10
TimeoutStartSec=60
StandardOutput=journal
StandardError=journal
Environment=NODE_ENV=production

[Install]
WantedBy=multi-user.target
```

#### **OMA API Service** (NEEDS ENCRYPTION KEY):
```ini
# Add unique encryption key generation
ENCRYPTION_KEY=$(openssl rand -base64 32)
Environment=MIGRATEKIT_CRED_ENCRYPTION_KEY=$ENCRYPTION_KEY
```

#### **Volume Daemon Service** (NEEDS STANDARDIZATION):
```ini
# Ensure consistent user model
User=oma_admin
Group=oma_admin
Environment=GIN_MODE=release
```

### **3. SSH TUNNEL INFRASTRUCTURE (CRITICAL)**
**Issue**: SSH port 443 and tunnel configuration not properly set up
**Fix**: Complete SSH tunnel infrastructure setup

#### **SSH Configuration**:
```bash
# Add to /etc/ssh/sshd_config
Port 443

# VMA Tunnel User Configuration - Production
Match User vma_tunnel
    AuthenticationMethods publickey
    PubkeyAuthentication yes
    PasswordAuthentication no
    PermitTTY no
    X11Forwarding no
    AllowTcpForwarding remote
    PermitOpen 127.0.0.1:10809 127.0.0.1:8082
    PermitListen 127.0.0.1:9081
```

#### **SSH Socket Configuration**:
```bash
# Create /etc/systemd/system/ssh.socket.d/port443.conf
[Socket]
ListenStream=
ListenStream=0.0.0.0:22
ListenStream=0.0.0.0:443
ListenStream=[::]:22
ListenStream=[::]:443
```

#### **VMA Pre-shared Key Setup**:
```bash
# Use VMA's real cloudstack_key.pub (not generated keys)
# Copy from deployment package: keys/vma-preshared-key.pub
sudo cp /tmp/deployment-package/keys/vma-preshared-key.pub /var/lib/vma_tunnel/.ssh/authorized_keys
```

### **4. NBD SERVER CONFIGURATION (CRITICAL)**
**Issue**: NBD server doesn't start without proper config
**Fix**: Deploy production NBD configuration

#### **NBD Config Base**:
```ini
[generic]
port = 10809
allowlist = true
includedir = /etc/nbd-server/conf.d

# Dummy export required for NBD server to start
[dummy]
exportname = /dev/null
readonly = true
```

#### **NBD Deployment**:
```bash
# Copy config-base to both locations
sudo cp /tmp/deployment-package/configs/config-base /etc/nbd-server/
sudo cp /tmp/deployment-package/configs/config-base /etc/nbd-server/config
```

### **5. SERVICE STARTUP ORDER (IMPORTANT)**
**Issue**: Services need proper dependency order and timing
**Fix**: Start services with dependencies and delays

```bash
# Proper startup sequence
sudo systemctl enable mariadb volume-daemon oma-api migration-gui nbd-server
sudo systemctl start volume-daemon
sleep 5
sudo systemctl start nbd-server
sleep 3
sudo systemctl start oma-api
sleep 8
sudo systemctl start migration-gui
sleep 5
sudo systemctl restart ssh.socket
```

### **6. COMPREHENSIVE VALIDATION (IMPORTANT)**
**Issue**: Script needs to validate all components work
**Fix**: Add complete health check validation

```bash
# Test all health endpoints
curl -s http://$TARGET_IP:8082/health    # OMA API
curl -s http://$TARGET_IP:8090/api/v1/health    # Volume Daemon  
curl -s http://$TARGET_IP:3001           # Migration GUI

# Test infrastructure
ss -tlnp | grep :10809    # NBD Server
ss -tlnp | grep :443      # SSH Port 443
id vma_tunnel             # Tunnel user
```

---

## 📦 **SELF-CONTAINED PACKAGE REQUIREMENTS**

### **Package Structure** (ALREADY CREATED):
```
/home/pgrayson/oma-deployment-package/
├── binaries/
│   ├── oma-api (33.4MB - real v2.39.0-gorm-field-fix)
│   └── volume-daemon (14.9MB - real v2.0.0-by-id-paths)
├── database/
│   └── production-schema.sql (62KB - complete 34 tables)
├── gui/
│   └── migration-gui-built.tar.gz (99MB - pre-built Next.js)
├── keys/
│   └── vma-preshared-key.pub (VMA's real cloudstack_key.pub)
├── configs/
│   └── config-base (NBD configuration)
└── deploy-real-production-oma.sh (NEEDS UPDATES)
```

### **Package Usage**:
```bash
# Self-contained deployment (no external dependencies)
./oma-deployment-package/deploy-real-production-oma.sh 10.245.246.134
```

---

## ✅ **PROVEN WORKING CONFIGURATION**

### **OMAv3 (10.245.246.134) - FULLY OPERATIONAL:**
- ✅ **OMA API**: Real binary working on port 8082
- ✅ **Volume Daemon**: Real binary working on port 8090
- ✅ **Migration GUI**: Working on port 3001 (accessible)
- ✅ **Database**: 34 tables imported and operational
- ✅ **NBD Server**: Listening on port 10809
- ✅ **SSH Tunnel**: vma_tunnel user with VMA pre-shared key
- ✅ **SSH Port 443**: Listening and configured
- ✅ **Network Performance**: 12.1 MB/s (200x better than other servers)

### **VMA Integration Ready:**
- ✅ **Pre-shared Key**: VMA's cloudstack_key.pub in vma_tunnel authorized_keys
- ✅ **SSH Restrictions**: Proper tunnel restrictions configured
- ✅ **Port Permissions**: PermitOpen and PermitListen configured correctly

---

## 🎯 **NEXT STEPS**

### **Script Updates Needed:**
1. **Integrate passwordless sudo setup** as Phase 1
2. **Add missing service file creation** (migration-gui, proper oma-api, volume-daemon)
3. **Add complete SSH tunnel infrastructure** setup
4. **Add NBD server configuration** deployment
5. **Add comprehensive validation** phase
6. **Use self-contained package** approach (no external dependencies)

### **Testing Plan:**
1. **Update script** with all identified fixes
2. **Test on fresh OMAv3** after revert
3. **Validate complete automation** (no manual fixes needed)
4. **Test VMA tunnel connectivity** with real pre-shared key
5. **Export as production template** for customer deployment

---

## 📋 **DEPLOYMENT SCRIPT TEMPLATE STRUCTURE**

### **Recommended Script Flow:**
```
Phase 1: Passwordless Sudo Setup (eliminate password prompts)
Phase 2: System Preparation (cloud-init disable, packages)
Phase 3: Database Setup (MariaDB, 34-table schema import)
Phase 4: Binary Deployment (real OMA API, Volume Daemon)
Phase 5: GUI Deployment (pre-built Next.js application)
Phase 6: Service Configuration (create all systemd services)
Phase 7: SSH Tunnel Infrastructure (vma_tunnel user, port 443, pre-shared key)
Phase 8: NBD Server Configuration (config-base with dummy export)
Phase 9: Service Startup (proper dependency order)
Phase 10: Comprehensive Validation (all health checks)
Phase 11: VMA Tunnel Test (optional connectivity test)
```

---

**🎯 This document captures all the fixes needed to make the deployment script completely bulletproof for future deployments. The working OMAv3 proves all these components work together correctly.**

**Status**: Ready to integrate these fixes into the deployment script for fully automated production deployments.

