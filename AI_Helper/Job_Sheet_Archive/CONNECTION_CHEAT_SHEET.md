# 🔗 **CONNECTION CHEAT SHEET**

**Last Updated**: September 30, 2025  
**Purpose**: Quick reference for all system connections and access details

---

## 🖥️ **PRODUCTION OMA SERVERS**

### **Server 121 - Production OMA (FULLY OPERATIONAL)**
- **IP Address**: `10.245.246.121`
- **SSH Access**: `ssh -i ~/.ssh/ossea-appliance-build oma_admin@10.245.246.121`
- **SSH Key**: `~/.ssh/ossea-appliance-build`
- **User**: `oma_admin` (sudo with Password1)
- **OS**: Ubuntu 24.04.3 LTS
- **Uptime**: 3+ days (stable)
- **Status**: ✅ **100% OPERATIONAL**

**Services Status**:
- ✅ **OMA API**: `http://10.245.246.121:8082` (working)
- ✅ **Migration GUI**: `http://10.245.246.121:3001` (working)
- ✅ **Volume Daemon**: `http://10.245.246.121:8090` (working)
- ✅ **MariaDB**: localhost:3306 (active)
- ✅ **NBD Server**: port 10809 (active)
- ✅ **Custom Boot**: oma-autologin.service (active)

**Missing Components**:
- ❌ **SSH Tunnel**: No vma_tunnel user configured
- ❌ **VirtIO Tooling**: Not verified
- ❌ **Pre-shared Keys**: Not configured

### **OMAv6 - Latest Production Deployment (NEWLY DEPLOYED)**
- **IP Address**: `10.245.246.147`
- **SSH Access**: `sshpass -p 'Password1' ssh oma_admin@10.245.246.147`
- **User**: `oma_admin` (sudo with Password1)
- **OS**: Ubuntu 24.04 LTS
- **Status**: ✅ **DEPLOYED** with all fixes
- **Binary**: oma-api-v2.40.2-production-oma-vm-id-fix

**Services Status**:
- ✅ **OMA API**: `http://10.245.246.147:8082` (working)
- ✅ **Migration GUI**: `http://10.245.246.147:3001` (working)
- ✅ **Volume Daemon**: `http://10.245.246.147:8090` (working)
- ✅ **SSH Tunnel**: vma_tunnel user configured
- ✅ **VirtIO Tools**: Complete Windows VM support

**Issues Identified**:
- ❌ **FK Constraint**: ossea_config_id mismatch (expects 1, has 2)
- ❌ **CloudStack Auth**: API authentication failure

### **Server 120 - Production OMA (FULLY OPERATIONAL)**
- **IP Address**: `10.245.246.120`
- **SSH Access**: `ssh -i ~/.ssh/oma-v2-server oma_admin@10.245.246.120`
- **SSH Key**: `~/.ssh/oma-v2-server`
- **User**: `oma_admin` (sudo with Password1)
- **OS**: Ubuntu 24.04.2 LTS
- **Uptime**: 2+ days (stable)
- **Status**: ✅ **100% OPERATIONAL**

**Services Status**:
- ✅ **OMA API**: `http://10.245.246.120:8082` (working)
- ✅ **Migration GUI**: `http://10.245.246.120:3001` (working)
- ✅ **Volume Daemon**: `http://10.245.246.120:8090` (working)
- ✅ **MariaDB**: localhost:3306 (active)
- ✅ **NBD Server**: port 10809 (active)
- ✅ **Custom Boot**: oma-autologin.service (active)

**Missing Components**:
- ❌ **SSH Tunnel**: Not verified
- ❌ **VirtIO Tooling**: Not verified
- ❌ **Pre-shared Keys**: Not configured

---

## 🖥️ **DEVELOPMENT OMA SERVER**

### **Dev OMA - Current Development System**
- **IP Address**: `10.245.246.125`
- **SSH Access**: Local system (no SSH needed)
- **User**: `pgrayson` (development user)
- **OS**: Ubuntu (development environment)
- **Status**: ✅ **DEVELOPMENT ACTIVE**

**Services Status**:
- ✅ **Migration GUI**: `http://10.245.246.125:3001` (working)
- ✅ **OMA API**: `http://10.245.246.125:8082` (working)
- ✅ **Volume Daemon**: `http://10.245.246.125:8090` (working)
- ✅ **MariaDB**: localhost:3306 (active)
- ✅ **iperf3 Server**: port 8888 (for testing)

**Network Issues**:
- ⚠️ **TCP Throttling**: Upload speed limited to ~0.06 MB/s
- ✅ **UDP Performance**: 11.9 MB/s (works fine)
- 🔍 **Root Cause**: Network infrastructure TCP throttling

---

## 🖥️ **VMA SERVERS**

### **VMA 231 - Primary Development VMA**
- **IP Address**: `10.0.100.231`
- **SSH Access**: `ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231`
- **SSH Key**: `~/.ssh/cloudstack_key`
- **User**: `pgrayson` (development user)
- **Hostname**: `pg-migrationdev`
- **Status**: ✅ **OPERATIONAL**

**Network Performance**:
- ❌ **TCP Upload**: 0.06 MB/s (throttled)
- ✅ **TCP Download**: 2.33 MB/s (usable)
- 🔍 **Issue**: Same network throttling as other VMAs

### **VMA 232 - SSH Tunnel Test VMA**
- **IP Address**: `10.0.100.232`
- **SSH Access**: `ssh -i ~/.ssh/vma_232_key vma@10.0.100.232`
- **SSH Key**: `~/.ssh/vma_232_key`
- **User**: `vma` (production user model)
- **Status**: ✅ **SSH TUNNEL READY**

**Features**:
- ✅ **SSH Tunnel**: Configured for port 443
- ✅ **Production User**: Uses vma user model
- ✅ **Systemd Services**: Proper service configuration

### **VMA 233 - Performance Test VMA**
- **IP Address**: `10.0.100.233`
- **SSH Access**: `ssh -i ~/.ssh/vma_233_key vma@10.0.100.233`
- **SSH Key**: `~/.ssh/vma_233_key`
- **User**: `vma` (production user model)
- **Hostname**: `vma`
- **Status**: ✅ **OPERATIONAL**

**Network Performance**:
- ❌ **TCP Upload**: 0.07 MB/s (throttled)
- ✅ **TCP Download**: 5.24 MB/s (better than 231)
- ✅ **UDP**: 11.9 MB/s (perfect)

---

## 🔑 **SSH KEY REFERENCE**

### **Available SSH Keys**
```bash
# Production OMA servers
~/.ssh/ossea-appliance-build      # Server 121 access
~/.ssh/oma-v2-server             # Server 120 access
~/.ssh/remote-oma-server         # Alternative OMA access

# VMA servers
~/.ssh/cloudstack_key            # VMA 231 (development)
~/.ssh/vma_232_key              # VMA 232 (SSH tunnel ready)
~/.ssh/vma_233_key              # VMA 233 (performance testing)
```

### **Key Permissions Check**
```bash
# Ensure proper permissions
chmod 600 ~/.ssh/ossea-appliance-build
chmod 600 ~/.ssh/oma-v2-server
chmod 600 ~/.ssh/vma_*_key
```

---

## 🌐 **QUICK ACCESS COMMANDS**

### **Health Checks**
```bash
# Server 121 health
curl -s http://10.245.246.121:8082/health && echo "✅ API OK" || echo "❌ API Failed"
curl -s http://10.245.246.121:3001 > /dev/null && echo "✅ GUI OK" || echo "❌ GUI Failed"

# Server 120 health  
curl -s http://10.245.246.120:8082/health && echo "✅ API OK" || echo "❌ API Failed"
curl -s http://10.245.246.120:3001 > /dev/null && echo "✅ GUI OK" || echo "❌ GUI Failed"

# Dev OMA health
curl -s http://10.245.246.125:8082/health && echo "✅ API OK" || echo "❌ API Failed"
```

### **Service Status Checks**
```bash
# Server 121 services
ssh -i ~/.ssh/ossea-appliance-build oma_admin@10.245.246.121 'systemctl status oma-api volume-daemon migration-gui'

# Server 120 services
ssh -i ~/.ssh/oma-v2-server oma_admin@10.245.246.120 'systemctl status oma-api volume-daemon migration-gui'
```

### **Database Access**
```bash
# Server 121 database
ssh -i ~/.ssh/ossea-appliance-build oma_admin@10.245.246.121 'mysql -u oma_user -poma_password migratekit_oma'

# Server 120 database
ssh -i ~/.ssh/oma-v2-server oma_admin@10.245.246.120 'mysql -u oma_user -poma_password migratekit_oma'
```

---

## 🚀 **DEPLOYMENT TESTING PLAN**

### **Server 121 - Enhancement Target**
**Missing Components to Add**:
1. **SSH Tunnel Infrastructure** - Add vma_tunnel user and SSH hardening
2. **VirtIO Tooling** - Install `/usr/share/virtio-win/` for Windows failover
3. **Pre-shared Key System** - Configure VMA enrollment capability

### **Server 120 - Template Creation Target**
**Use Case**: Perfect for creating production template
1. **Assess Current State** - Document all deployed components
2. **Add Missing Components** - SSH tunnel, VirtIO, pre-shared keys
3. **Create Template** - Export as production-ready template

### **Testing Workflow**
1. **Use Server 121** - Test deployment script enhancements
2. **Validate on Server 120** - Confirm template readiness
3. **Export Template** - Create repeatable deployment

---

## 📊 **CURRENT ASSESSMENT SUMMARY**

### **✅ What's Working**
- **Both production servers**: 100% core services operational
- **All health endpoints**: API, GUI, Volume Daemon responding
- **Database systems**: MariaDB active with proper schemas
- **Professional boot**: Custom OSSEA-Migrate boot experience
- **Service management**: Proper systemd integration

### **❌ What's Missing**
- **SSH Tunnel Infrastructure**: vma_tunnel user setup
- **VirtIO Tooling**: Windows VM failover support
- **Pre-shared Key System**: VMA enrollment capability
- **Network Performance**: TCP throttling affects all systems

### **🎯 Next Steps**
1. **Complete Server 121** - Add missing SSH tunnel components
2. **Test Full Workflow** - End-to-end migration testing
3. **Create Production Template** - Repeatable deployment from Server 120
4. **Network Issue Escalation** - Address TCP throttling with network team

---

**Status**: Ready for production deployment testing and template creation! 🚀


## 📦 **VMA DEPLOYMENT PACKAGE**

### **Package Location**: `/home/pgrayson/vma-deployment-package/`
- **Binaries**: Latest production VMA binaries (40MB)
- **Configurations**: Service files and templates
- **SSH Keys**: Pre-shared key for tunnel authentication
- **Scripts**: Fixed wizard and deployment automation
- **Dependencies**: System package requirements

### **Deployment Usage**:
```bash
cd /home/pgrayson/vma-deployment-package
./scripts/deploy-vma-production.sh <TARGET_IP>
```

### **Package Features**:
- ✅ Self-contained (no external dependencies)
- ✅ Real production binaries (no simulation)
- ✅ Fixed wizard (no config syntax errors)
- ✅ Complete automation
