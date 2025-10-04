# 🖥️ **OMA CUSTOM BOOT SETUP JOB SHEET**

**Created**: September 27, 2025  
**Priority**: 🔥 **HIGH** - Professional OMA deployment automation  
**Issue ID**: OMA-BOOT-SETUP-001  
**Status**: 📋 **PLANNING PHASE** - OMA deployment automation design

---

## 🎯 **EXECUTIVE SUMMARY**

**Objective**: Create custom OMA boot experience that presents network configuration interface and service status dashboard for streamlined deployment.

**Business Value**: 
- ✅ **Simplified Deployment**: New OMA setups require only network configuration
- ✅ **Service Monitoring**: Real-time status of all OSSEA-Migrate services
- ✅ **Professional Interface**: Branded configuration experience for enterprise deployments
- ✅ **Network Management**: Complete network configuration without Linux knowledge

---

## 🏗️ **TECHNICAL ARCHITECTURE**

### **🔧 Boot Process Enhancement**

#### **Enhanced Boot Flow:**
```
1. OMA boots → Custom auto-login service ✅
2. OSSEA-Migrate Configuration Wizard ✅
3. Network configuration interface ✅
4. Service status dashboard ✅
5. Access URL display ✅
```

### **📋 Configuration Interface Components**

#### **Component 1: Network Configuration**
```bash
# OMA Network Settings
Current IP: 10.245.246.125
Interface: ens18
Gateway: 10.245.246.1
DNS: 8.8.8.8
Configuration: DHCP

Options:
1. Keep current network configuration
2. Configure static IP address
3. View service status
4. Access OSSEA-Migrate GUI
```

#### **Component 2: Service Status Dashboard**
```bash
# OSSEA-Migrate Service Status
┌─────────────────────────────────────┐
│ 🟢 OMA API Service        [ACTIVE]  │
│ 🟢 Volume Daemon          [ACTIVE]  │
│ 🟢 NBD Server             [ACTIVE]  │
│ 🟢 MariaDB Database       [ACTIVE]  │
│ 🟢 Migration GUI          [ACTIVE]  │
│ 🟡 VMA Connection         [PENDING] │
└─────────────────────────────────────┘

Access URLs:
- GUI: http://10.245.246.125:3001
- API: http://10.245.246.125:8082
```

#### **Component 3: Professional Interface**
```bash
╔══════════════════════════════════════════════════════════════════╗
║                     OSSEA-Migrate - OMA Setup                   ║
║                OSSEA Migration Appliance Control                 ║
║                                                                  ║
║              🚀 Professional Migration Platform                  ║
╚══════════════════════════════════════════════════════════════════╝

Welcome to OSSEA-Migrate OMA (OSSEA Migration Appliance) Configuration
This interface provides network configuration and service management.

📡 Current Network Configuration:
   OMA IP Address: 10.245.246.125
   Network Interface: ens18
   Gateway: 10.245.246.1
   DNS Server: 8.8.8.8
   Configuration: DHCP

🚀 Service Status:
   OMA API: ✅ Active (Port 8082)
   Volume Daemon: ✅ Active (Port 8090)
   NBD Server: ✅ Active (Port 10809)
   MariaDB: ✅ Active (Port 3306)
   Migration GUI: ✅ Active (Port 3001)

🌐 Access Information:
   Web Interface: http://10.245.246.125:3001
   API Endpoint: http://10.245.246.125:8082
   Documentation: Available in GUI settings

🔧 Configuration Options:
   1. Configure network settings
   2. View detailed service status
   3. Access OSSEA-Migrate GUI
   4. Admin shell access
```

---

## 📋 **IMPLEMENTATION COMPONENTS**

### **🔧 OMA Setup Wizard Script**

#### **File: `/opt/ossea-migrate/oma-setup-wizard.sh`**
```bash
#!/bin/bash
# OMA Setup Wizard - Network Configuration and Service Status
# Professional deployment interface for OSSEA-Migrate OMA

# Network configuration functions
get_current_network_info() {
    OMA_IP=$(hostname -I | awk '{print $1}')
    OMA_INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    OMA_GATEWAY=$(ip route | grep default | awk '{print $3}' | head -1)
    OMA_DNS=$(cat /etc/resolv.conf | grep nameserver | head -1 | awk '{print $2}')
}

# Service status checking
check_service_status() {
    OMA_API_STATUS=$(systemctl is-active oma-api.service)
    VOLUME_DAEMON_STATUS=$(systemctl is-active volume-daemon.service)
    NBD_SERVER_STATUS=$(systemctl is-active nbd-server.service)
    MARIADB_STATUS=$(systemctl is-active mariadb.service)
    GUI_STATUS=$(systemctl is-active migration-gui.service)
}

# Service health checking
check_service_health() {
    # Test OMA API health
    if curl -s http://localhost:8082/health > /dev/null 2>&1; then
        OMA_API_HEALTH="✅ Healthy"
    else
        OMA_API_HEALTH="❌ Not responding"
    fi
    
    # Test Volume Daemon health
    if curl -s http://localhost:8090/api/v1/health > /dev/null 2>&1; then
        VOLUME_DAEMON_HEALTH="✅ Healthy"
    else
        VOLUME_DAEMON_HEALTH="❌ Not responding"
    fi
    
    # Test GUI health
    if curl -s http://localhost:3001 > /dev/null 2>&1; then
        GUI_HEALTH="✅ Healthy"
    else
        GUI_HEALTH="❌ Not responding"
    fi
}
```

### **🔧 Auto-Login Service**

#### **File: `/etc/systemd/system/oma-autologin.service`**
```ini
[Unit]
Description=OMA Auto-login Setup Wizard
Documentation=https://github.com/DRDAVIDBANNER/X-Vire
After=multi-user.target network.target mariadb.service
Wants=network.target

[Service]
Type=idle
User=pgrayson
Group=pgrayson
TTY=/dev/tty1
ExecStart=/opt/ossea-migrate/oma-setup-wizard.sh
StandardInput=tty
StandardOutput=tty
StandardError=tty
Restart=no
RemainAfterExit=yes

Environment=HOME=/home/pgrayson
Environment=USER=pgrayson
Environment=TERM=xterm-256color

[Install]
WantedBy=multi-user.target
```

### **🔧 Network Configuration Functions**

#### **Static IP Configuration:**
```bash
configure_static_ip() {
    local static_ip="$1"
    local netmask="$2" 
    local gateway="$3"
    local dns="$4"
    
    # Create netplan configuration
    cat > /tmp/01-static-config.yaml << EOF
network:
  version: 2
  renderer: networkd
  ethernets:
    $OMA_INTERFACE:
      dhcp4: false
      addresses:
        - $static_ip/$netmask
      gateway4: $gateway
      nameservers:
        addresses:
          - $dns
EOF
    
    # Apply configuration
    sudo cp /tmp/01-static-config.yaml /etc/netplan/01-netcfg.yaml
    sudo netplan apply
}
```

---

## 🎨 **USER INTERFACE DESIGN**

### **📺 Main Configuration Screen**
```
╔══════════════════════════════════════════════════════════════════╗
║                     OSSEA-Migrate - OMA Setup                   ║
║                OSSEA Migration Appliance Control                 ║
║                                                                  ║
║              🚀 Professional Migration Platform                  ║
╚══════════════════════════════════════════════════════════════════╝

📡 Network Configuration:
   IP Address: 10.245.246.125        Interface: ens18
   Gateway: 10.245.246.1             DNS: 8.8.8.8
   Configuration: DHCP               Status: ✅ Connected

🚀 Service Status:
   ┌─────────────────────────────────────┐
   │ 🟢 OMA API Service        [ACTIVE]  │
   │ 🟢 Volume Daemon          [ACTIVE]  │  
   │ 🟢 NBD Server             [ACTIVE]  │
   │ 🟢 MariaDB Database       [ACTIVE]  │
   │ 🟢 Migration GUI          [ACTIVE]  │
   └─────────────────────────────────────┘

🌐 Access Information:
   Web Interface: http://10.245.246.125:3001
   API Endpoint: http://10.245.246.125:8082/health
   System Status: All services operational

🔧 Configuration Options:
   1. Configure network settings
   2. View detailed service logs
   3. Access OSSEA-Migrate GUI
   4. Restart services
   5. Admin shell access
```

### **📊 Service Status Details**
```
🔍 Detailed Service Information:

OMA API Service (oma-api.service):
├── Status: ✅ Active (running)
├── Memory: 16.2M
├── Uptime: 2h 15m
├── Health: ✅ http://localhost:8082/health
└── Endpoints: 62 API routes configured

Volume Daemon (volume-daemon.service):
├── Status: ✅ Active (running)
├── Memory: 5.6M
├── Uptime: 2h 15m
├── Health: ✅ http://localhost:8090/api/v1/health
└── Volumes: 8 managed, 8 with persistent naming

Migration GUI (migration-gui.service):
├── Status: ✅ Active (running)
├── Memory: 178M
├── Uptime: 45m
├── Health: ✅ http://localhost:3001
└── Interface: Professional OSSEA-Migrate dashboard
```

---

## 📋 **IMPLEMENTATION PHASES**

### **🚀 PHASE 1: Boot Setup Creation (45 minutes)**

#### **Task 1.1: OMA Setup Wizard**
- Create professional OSSEA-Migrate branded interface
- Implement network configuration with validation
- Add service status monitoring dashboard
- Include access URL display and guidance

#### **Task 1.2: Auto-Login Service**
- Create systemd service for boot-to-wizard experience
- Configure proper dependencies and startup order
- Add security and environment configuration

#### **Task 1.3: Network Configuration**
- Implement static IP configuration with netplan
- Add DHCP/static detection and switching
- Include validation and connectivity testing

### **🔧 PHASE 2: Service Integration (30 minutes)**

#### **Task 2.1: Service Status Monitoring**
- Real-time status checking for all OSSEA-Migrate services
- Health endpoint testing and response validation
- Memory and uptime monitoring display

#### **Task 2.2: Professional Interface**
- Branded OSSEA-Migrate headers and styling
- Professional color scheme and layout
- Clear navigation and option selection

### **🔧 PHASE 3: Deployment & Testing (30 minutes)**

#### **Task 3.1: Installation Script**
- Automated deployment to OMA appliance
- Service configuration and enablement
- Testing and validation procedures

#### **Task 3.2: Production Validation**
- Test complete boot-to-configuration workflow
- Validate network configuration changes
- Verify service status accuracy

---

## 🎯 **SUCCESS CRITERIA**

### **Professional Deployment Goals:**
- [ ] ✅ **Boot-to-Configuration**: OMA boots directly to setup interface
- [ ] ✅ **Network Management**: Complete IP/DNS/gateway configuration
- [ ] ✅ **Service Monitoring**: Real-time status of all services
- [ ] ✅ **Access Guidance**: Clear URLs and next steps for users

### **Enterprise Features:**
- [ ] ✅ **Professional Branding**: OSSEA-Migrate branded interface
- [ ] ✅ **Status Dashboard**: Comprehensive service health monitoring
- [ ] ✅ **Network Configuration**: Both DHCP and static IP support
- [ ] ✅ **User Guidance**: Clear access instructions and next steps

### **Operational Benefits:**
- [ ] ✅ **Zero Linux Knowledge**: Network configuration without command line
- [ ] ✅ **Service Visibility**: Clear indication of system health
- [ ] ✅ **Professional Appearance**: Enterprise-appropriate deployment interface
- [ ] ✅ **Complete Automation**: Plug-and-play OMA deployment experience

---

**🎯 This creates a complete professional deployment experience for both VMA and OMA appliances, making OSSEA-Migrate truly plug-and-play for enterprise environments.**






