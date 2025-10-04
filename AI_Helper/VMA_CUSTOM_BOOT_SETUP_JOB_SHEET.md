# 🖥️ **VMA CUSTOM BOOT SETUP JOB SHEET**

**Created**: September 26, 2025  
**Priority**: 🔥 **HIGH** - Deployment automation and user experience  
**Issue ID**: VMA-BOOT-SETUP-001  
**Status**: 📋 **PLANNING PHASE** - Custom VMA boot configuration design

---

## 🎯 **EXECUTIVE SUMMARY**

**Objective**: Create custom VMA boot experience that bypasses login prompt and presents OMA IP configuration interface for streamlined deployment.

**Business Value**: 
- ✅ **Simplified Deployment**: New VMA setups require only OMA IP input
- ✅ **User Experience**: No Linux knowledge required for basic setup
- ✅ **Automated Configuration**: Tunnel setup, service configuration, and connectivity testing
- ✅ **Professional Appearance**: Clean, branded interface for enterprise deployments

---

## 🏗️ **TECHNICAL ARCHITECTURE**

### **🔧 Boot Process Modification**

#### **Standard Boot Flow (Current):**
```
1. GRUB bootloader
2. Linux kernel load
3. systemd initialization
4. getty login prompt ❌ (requires manual login)
5. Manual configuration ❌ (requires Linux knowledge)
```

#### **Enhanced Boot Flow (Proposed):**
```
1. GRUB bootloader  
2. Linux kernel load
3. systemd initialization
4. Custom auto-login service ✅
5. VMA Setup Wizard ✅ (OMA IP configuration)
6. Automated tunnel setup ✅
7. Service validation ✅
```

### **📋 Implementation Components**

#### **Component 1: Auto-Login Service**
```bash
# File: /etc/systemd/system/vma-autologin.service
[Unit]
Description=VMA Auto-login for Setup Wizard
After=multi-user.target

[Service]
Type=idle
User=pgrayson
TTY=/dev/tty1
ExecStart=/bin/bash /opt/vma/setup-wizard.sh
StandardInput=tty
StandardOutput=tty
Restart=no

[Install]
WantedBy=multi-user.target
```

#### **Component 2: VMA Setup Wizard**
```bash
# File: /opt/vma/setup-wizard.sh
#!/bin/bash

# VMA Setup Wizard - OMA Connection Configuration
clear

cat << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║                    MigrateKit OSSEA - VMA Setup                  ║
║                  VMware Migration Appliance                      ║
╚══════════════════════════════════════════════════════════════════╝

🚀 Welcome to VMA (VMware Migration Appliance) Setup

This wizard will configure your VMA to connect to the OMA (OSSEA Migration Appliance).

EOF

# Get OMA IP address
while true; do
    echo ""
    read -p "📡 Enter OMA IP Address (e.g., 10.245.246.125): " OMA_IP
    
    # Validate IP format
    if [[ $OMA_IP =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        echo "✅ Valid IP format: $OMA_IP"
        break
    else
        echo "❌ Invalid IP format. Please enter a valid IP address."
    fi
done

# Test connectivity
echo ""
echo "🔍 Testing connectivity to OMA at $OMA_IP..."
if ping -c 3 $OMA_IP > /dev/null 2>&1; then
    echo "✅ Network connectivity to OMA successful"
else
    echo "⚠️  Warning: Cannot ping OMA at $OMA_IP (may be normal if firewall blocks ping)"
fi

# Configure tunnel
echo ""
echo "🔧 Configuring VMA-OMA tunnel..."
sudo /opt/vma/configure-oma-connection.sh "$OMA_IP"

# Start services
echo ""
echo "🚀 Starting VMA services..."
sudo systemctl enable vma-tunnel-enhanced-v2.service
sudo systemctl start vma-tunnel-enhanced-v2.service
sudo systemctl enable vma-api.service  
sudo systemctl start vma-api.service

echo ""
echo "✅ VMA Setup Complete!"
echo ""
echo "📊 Connection Status:"
echo "   - OMA IP: $OMA_IP"
echo "   - VMA API: http://localhost:8081"
echo "   - Tunnel Status: $(systemctl is-active vma-tunnel-enhanced-v2.service)"
echo ""
echo "🎯 Next Steps:"
echo "   1. Access OMA GUI at: http://$OMA_IP:3001"
echo "   2. Add this VMA in Discovery settings"
echo "   3. Begin VM discovery and migration"
echo ""
read -p "Press Enter to continue to normal shell..."

# Drop to normal shell
exec /bin/bash
EOF
```

#### **Component 3: OMA Connection Configuration**
```bash
# File: /opt/vma/configure-oma-connection.sh
#!/bin/bash

OMA_IP="$1"
VMA_IP=$(hostname -I | awk '{print $1}')

echo "🔧 Configuring VMA-OMA connection..."
echo "   VMA IP: $VMA_IP"
echo "   OMA IP: $OMA_IP"

# Update tunnel configuration
cat > /opt/vma/tunnel-config.conf << EOF
OMA_HOST=$OMA_IP
OMA_PORT=443
VMA_API_PORT=8081
TUNNEL_LOCAL_PORT=9081
EOF

# Update systemd service with OMA IP
sudo sed -i "s/OMA_HOST=.*/OMA_HOST=$OMA_IP/" /etc/systemd/system/vma-tunnel-enhanced-v2.service
sudo systemctl daemon-reload

echo "✅ VMA-OMA connection configured"
```

---

## 🎨 **USER EXPERIENCE DESIGN**

### **📺 Boot Screen Interface**

#### **Visual Design:**
```
╔══════════════════════════════════════════════════════════════════╗
║                    MigrateKit OSSEA - VMA Setup                  ║
║                  VMware Migration Appliance                      ║
║                                                                  ║
║  🚀 Quick Setup Wizard                                           ║
║                                                                  ║
║  📡 OMA IP Address: [_______________]                            ║
║                                                                  ║
║  [Test Connection]  [Configure & Start]                         ║
║                                                                  ║
║  Status: ⏳ Configuring tunnel...                                ║
╚══════════════════════════════════════════════════════════════════╝
```

#### **User Flow:**
1. **Boot**: VMA boots directly to setup wizard (no login)
2. **Input**: User enters OMA IP address with validation
3. **Test**: Automatic connectivity test to OMA
4. **Configure**: Automatic tunnel and service configuration
5. **Validate**: Service status verification and next steps
6. **Complete**: Drop to normal shell or continue to management interface

### **🔒 Security Considerations**

#### **Auto-Login Security:**
- **Limited Scope**: Auto-login only for initial setup wizard
- **Session Control**: Wizard exits to normal shell after configuration
- **Service Isolation**: Setup wizard runs with limited permissions
- **One-Time Use**: Can be disabled after initial setup

#### **Network Security:**
- **Input Validation**: IP address format validation
- **Connectivity Testing**: Verify OMA accessibility before configuration
- **Secure Tunnel**: Maintains existing SSH tunnel security model
- **Service Verification**: Validate tunnel establishment before completion

---

## 📋 **IMPLEMENTATION PHASES**

### **🔧 PHASE 1: Boot Process Modification (1 hour)**

#### **Task 1.1: Disable Standard Login**
```bash
# Disable getty on tty1
sudo systemctl disable getty@tty1.service

# Create custom auto-login service
sudo systemctl enable vma-autologin.service
```

#### **Task 1.2: Create Setup Wizard Service**
```bash
# Install setup wizard scripts
sudo mkdir -p /opt/vma
sudo cp setup-wizard.sh /opt/vma/
sudo cp configure-oma-connection.sh /opt/vma/
sudo chmod +x /opt/vma/*.sh
```

#### **Task 1.3: Configure Systemd Auto-Login**
```bash
# Create auto-login service that launches setup wizard
sudo systemctl enable vma-autologin.service
sudo systemctl set-default multi-user.target
```

### **🔧 PHASE 2: Setup Wizard Implementation (2 hours)**

#### **Task 2.1: Interactive Interface**
- **Input handling**: OMA IP address with validation
- **Connectivity testing**: Ping and port checks
- **Progress indicators**: Real-time status updates
- **Error handling**: Clear error messages and retry options

#### **Task 2.2: Configuration Automation**
- **Tunnel setup**: Automatic SSH tunnel configuration
- **Service management**: Enable and start required services
- **Validation**: Verify tunnel establishment and service health
- **Status reporting**: Clear success/failure indicators

#### **Task 2.3: Professional Interface**
- **Branded display**: MigrateKit OSSEA branding and layout
- **Clear instructions**: Step-by-step guidance for users
- **Status indicators**: Visual feedback for each configuration step
- **Next steps**: Clear guidance for post-setup operations

### **🔧 PHASE 3: Integration & Testing (1 hour)**

#### **Task 3.1: Boot Testing**
- **Fresh boot validation**: Test complete boot-to-wizard flow
- **Configuration testing**: Verify OMA connection setup works
- **Service validation**: Confirm all services start correctly
- **User experience**: Validate ease of use for non-technical users

#### **Task 3.2: Deployment Preparation**
- **Image creation**: Prepare VMA image with custom boot setup
- **Documentation**: User guide for VMA deployment and setup
- **Rollback capability**: Method to revert to standard login if needed

---

## 🎯 **ADVANCED FEATURES**

### **📊 Enhanced Setup Options**

#### **Advanced Configuration Mode:**
```bash
# Optional advanced settings
echo "🔧 Advanced Configuration (Optional):"
echo "   1. Custom VMA API port (default: 8081)"
echo "   2. SSH key configuration"
echo "   3. Network interface selection"
echo "   4. Service startup options"
```

#### **Status Dashboard:**
```bash
# Post-setup status display
echo "📊 VMA System Status:"
echo "   - Tunnel: $(systemctl is-active vma-tunnel-enhanced-v2.service)"
echo "   - VMA API: $(systemctl is-active vma-api.service)"
echo "   - OMA Connectivity: $(curl -s http://localhost:9081/health | jq -r '.status' 2>/dev/null || echo 'Not connected')"
```

#### **Troubleshooting Mode:**
```bash
# Built-in diagnostics
echo "🔍 Troubleshooting Options:"
echo "   1. Test OMA connectivity"
echo "   2. Restart tunnel service"
echo "   3. View service logs"
echo "   4. Reset configuration"
```

---

## 📊 **DEPLOYMENT STRATEGY**

### **🔄 Implementation Approach**

#### **Phase A: Development (Safe)**
1. **Create scripts** on existing VMA for testing
2. **Test setup wizard** functionality manually
3. **Validate configuration automation** works correctly

#### **Phase B: Service Integration (Controlled)**
1. **Install auto-login service** alongside existing login
2. **Test boot process** with fallback to standard login
3. **Validate service startup** and configuration automation

#### **Phase C: Production Deployment (Final)**
1. **Disable standard login** after validation
2. **Enable auto-login service** as primary boot experience
3. **Create deployment image** with custom boot setup

### **🔒 Safety Measures**

#### **Rollback Capability:**
```bash
# Emergency rollback to standard login
sudo systemctl disable vma-autologin.service
sudo systemctl enable getty@tty1.service
sudo systemctl set-default graphical.target
```

#### **Alternative Access:**
- **SSH access**: Always available for emergency access
- **TTY2-6**: Alternative terminals for troubleshooting
- **Recovery mode**: GRUB recovery options maintained

---

## 🎯 **SUCCESS CRITERIA**

### **✅ User Experience Goals**
- [ ] **Zero Linux Knowledge**: Users can configure VMA with only OMA IP
- [ ] **One-Step Setup**: Single IP input configures complete VMA-OMA connection
- [ ] **Clear Feedback**: Real-time status and next step guidance
- [ ] **Professional Interface**: Branded, clean, enterprise-appropriate display

### **✅ Technical Goals**
- [ ] **Automated Tunnel**: SSH tunnel configured and started automatically
- [ ] **Service Integration**: All VMA services started and validated
- [ ] **Connectivity Verification**: OMA connection tested and confirmed
- [ ] **Graceful Degradation**: Fallback to manual configuration if automation fails

### **✅ Deployment Goals**
- [ ] **Image Ready**: VMA image with custom boot setup deployable
- [ ] **Documentation**: Complete user guide for VMA deployment
- [ ] **Support Ready**: Troubleshooting and recovery procedures documented

---

**🎯 This enhancement transforms VMA from technical appliance to user-friendly plug-and-play OSSEA-Migrate solution.**
