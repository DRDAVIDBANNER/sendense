# Production Deployment Guide

## 🎯 **Deployment Overview**

This guide covers deploying the **production-ready** MigrateKit OSSEA system with **dynamic port allocation** and **TLS encryption**.

## 🏗️ **Architecture Components**

### **Required Appliances**
- **VMA (VMware Migration Appliance)**: `10.0.100.231`
- **OMA (OpenStack Migration Appliance)**: `10.245.246.125`

### **Network Requirements**
- **Firewall Ports**: Only 22, 80, 443 open between appliances
- **SSH Access**: Key-based authentication required
- **DNS/IP**: Direct IP connectivity or DNS resolution

## 🚀 **OMA (OSSEA Appliance) Setup**

### **1. Core Services Installation**
```bash
# Install required packages
sudo apt-get update
sudo apt-get install -y nbd-server stunnel4 openssh-server

# Create migratekit user
sudo useradd -m -s /bin/bash pgrayson
sudo usermod -aG sudo pgrayson
```

### **2. MariaDB Database Setup**
```bash
# Install MariaDB
sudo apt-get install -y mariadb-server mariadb-client

# Secure installation
sudo mysql_secure_installation

# Create database and user
sudo mysql -u root -p <<EOF
CREATE DATABASE migratekit_oma;
CREATE USER 'oma_user'@'localhost' IDENTIFIED BY 'oma_password';
GRANT ALL PRIVILEGES ON migratekit_oma.* TO 'oma_user'@'localhost';
FLUSH PRIVILEGES;
EOF
```

### **3. OMA API Server Deployment**
```bash
# Deploy OMA API binary
sudo cp bin/oma-api /usr/local/bin/oma-api
sudo chmod +x /usr/local/bin/oma-api

# Use setup script for service deployment
sudo ./scripts/setup-oma-service.sh

# OR manually create systemd service
sudo cp scripts/oma-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable oma-api
sudo systemctl start oma-api

# Verify service
sudo systemctl status oma-api
```

### **4. TLS Certificate Setup**
```bash
# Generate self-signed certificate for stunnel
sudo mkdir -p /etc/stunnel
sudo openssl req -x509 -newkey rsa:2048 \
    -keyout /etc/stunnel/nbd.pem \
    -out /etc/stunnel/nbd.pem \
    -days 365 -nodes \
    -subj "/CN=oma-$(hostname)"
sudo chmod 600 /etc/stunnel/nbd.pem
```

### **5. stunnel Server Configuration**
```bash
# Create stunnel config
sudo tee /etc/stunnel/nbd-server.conf <<EOF
[nbd-tls-server]
accept = 443
connect = 127.0.0.1:10810
cert = /etc/stunnel/nbd.pem
debug = 3
EOF

# Start stunnel server
sudo stunnel /etc/stunnel/nbd-server.conf
```

### **6. Target Device Preparation**
```bash
# Prepare target devices for migrations
sudo mkdir -p /mnt/migration-targets

# Create device files (example)
sudo fallocate -l 50G /mnt/migration-targets/vde.img
sudo fallocate -l 50G /mnt/migration-targets/vdf.img
sudo fallocate -l 50G /mnt/migration-targets/vdg.img

# Setup loop devices
sudo losetup /dev/loop1 /mnt/migration-targets/vde.img
sudo losetup /dev/loop2 /mnt/migration-targets/vdf.img
sudo losetup /dev/loop3 /mnt/migration-targets/vdg.img

# Create symlinks for consistency
sudo ln -sf /dev/loop1 /dev/vde
sudo ln -sf /dev/loop2 /dev/vdf  
sudo ln -sf /dev/loop3 /dev/vdg
```

## 🔧 **VMA (VMware Appliance) Setup**

### **1. Core Services Installation**
```bash
# Install required packages
sudo apt-get update
sudo apt-get install -y stunnel4 openssh-client

# Create migratekit user
sudo useradd -m -s /bin/bash pgrayson
sudo usermod -aG sudo pgrayson
```

### **2. Migration Binaries Deployment**
```bash
# Deploy binaries
sudo cp migratekit-tls-tunnel /usr/local/bin/
sudo cp vma-api-server /usr/local/bin/
sudo cp vma-client /usr/local/bin/
sudo chmod +x /usr/local/bin/migratekit-tls-tunnel
sudo chmod +x /usr/local/bin/vma-api-server
sudo chmod +x /usr/local/bin/vma-client
```

### **3. VMA API Server Service**
```bash
# Create VMA API service
sudo tee /etc/systemd/system/vma-api.service <<EOF
[Unit]
Description=VMA Control API Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=pgrayson
ExecStart=/usr/local/bin/vma-api-server -port 8081
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl enable vma-api
sudo systemctl start vma-api
```

### **4. SSH Key Setup**
```bash
# Generate SSH key for OMA connection
ssh-keygen -t rsa -b 2048 -f ~/.ssh/cloudstack_key -N ""

# Copy public key to OMA
ssh-copy-id -i ~/.ssh/cloudstack_key.pub pgrayson@10.245.246.125
```

### **5. Enhanced Tunnel Service**
```bash
# Create enhanced tunnel service
sudo tee /etc/systemd/system/vma-tunnel-enhanced.service <<EOF
[Unit]
Description=VMA Enhanced SSH Tunnel to OMA (Bidirectional + API Access)
After=network-online.target vma-api.service
Wants=network-online.target

[Service]
Type=simple
User=pgrayson
ExecStart=/usr/bin/ssh -i /home/pgrayson/.ssh/cloudstack_key \\
    -R 9081:localhost:8081 \\
    -L 8082:localhost:8082 \\
    -N -o StrictHostKeyChecking=no \\
    pgrayson@10.245.246.125
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl enable vma-tunnel-enhanced
sudo systemctl start vma-tunnel-enhanced
```

### **6. stunnel Client Certificate**
```bash
# Generate client certificate
sudo mkdir -p /etc/stunnel
sudo openssl req -x509 -newkey rsa:2048 \
    -keyout /etc/stunnel/client.pem \
    -out /etc/stunnel/client.pem \
    -days 365 -nodes \
    -subj "/CN=vma-client-$(hostname)"
sudo chmod 600 /etc/stunnel/client.pem
```

## 🔍 **Deployment Verification**

### **1. Service Status Check**
```bash
# OMA services
ssh pgrayson@10.245.246.125 "
  sudo systemctl status oma-api migration-gui
  curl -s http://localhost:8082/health | jq
  curl -s http://localhost:3001 | head -5
  sudo ss -tlnp | grep -E '8082|3001|9081|443'
"

# VMA services  
ssh pgrayson@10.0.100.231 "
  sudo systemctl status vma-api vma-tunnel-enhanced
  curl -s http://localhost:8081/api/v1/health | jq
"
```

### **2. Tunnel Connectivity Test**
```bash
# Test forward tunnel (VMA → OMA API)
ssh pgrayson@10.0.100.231 "curl -s http://localhost:8082/health | jq"

# Test reverse tunnel (OMA → VMA API)  
ssh pgrayson@10.245.246.125 "curl -s http://localhost:9081/api/v1/health | jq"
```

### **3. End-to-End Migration Test**
```bash
# Run VM discovery
ssh pgrayson@10.0.100.231 "
  cd /home/pgrayson/migratekit-cloudstack
  ./vma-client -oma http://localhost:8082 \\
    -vcenter quad-vcenter-01.quadris.local \\
    -user administrator@vsphere.local \\
    -password 'EmyGVoBFesGQc47-' \\
    -datacenter DatabanxDC \\
    -cmd discover \\
    -vm-filter PGWINTESTBIOS
"

# Start test migration
ssh pgrayson@10.0.100.231 "
  cd /home/pgrayson/migratekit-cloudstack  
  sudo ./vma-client -oma http://localhost:8082 \\
    -vcenter quad-vcenter-01.quadris.local \\
    -user administrator@vsphere.local \\
    -password 'EmyGVoBFesGQc47-' \\
    -datacenter DatabanxDC \\
    -cmd replicate \\
    -vm-filter PGWINTESTBIOS
"
```

## 📊 **Monitoring & Operations**

### **Log Locations**
```bash
# OMA logs
sudo journalctl -u oma-api -f

# VMA logs  
ssh pgrayson@10.0.100.231 "sudo journalctl -u vma-api -u vma-tunnel-enhanced -f"

# Migration logs
ssh pgrayson@10.0.100.231 "tail -f /home/pgrayson/migratekit-cloudstack/migration-*.log"
```

### **Performance Monitoring**
```bash
# NBD server activity
ssh pgrayson@10.245.246.125 "sudo ss -tlnp | grep nbd-server"

# Target device I/O
ssh pgrayson@10.245.246.125 "sudo iotop -a -o -d 5"

# Network throughput
ssh pgrayson@10.245.246.125 "sudo iftop -i eth0"
```

### **Resource Usage**
```bash
# Active replication jobs
curl -s http://10.245.246.125:8082/api/v1/replications | jq

# System resource status
curl -s http://10.245.246.125:8082/api/v1/system/status | jq
```

## 🚨 **Troubleshooting**

### **Common Issues**

#### **Tunnel Connection Failed**
```bash
# Check SSH connectivity
ssh -i ~/.ssh/cloudstack_key pgrayson@10.245.246.125 "echo 'SSH OK'"

# Check tunnel ports
ssh pgrayson@10.0.100.231 "ss -tlnp | grep -E '8082|10808'"
ssh pgrayson@10.245.246.125 "ss -tlnp | grep -E '9081|443'"
```

#### **NBD Server Not Starting**
```bash
# Check port availability
ssh pgrayson@10.245.246.125 "sudo ss -tlnp | grep 10813"

# Check NBD configuration
ssh pgrayson@10.245.246.125 "sudo cat /etc/nbd-server/config-dynamic-10813"

# Manual NBD server start
ssh pgrayson@10.245.246.125 "sudo nbd-server -C /etc/nbd-server/config-dynamic-10813"
```

#### **Migration Fails to Connect**
```bash
# Test NBD connection path
ssh pgrayson@10.0.100.231 "nc -z localhost 10808"  # stunnel client
ssh pgrayson@10.245.246.125 "nc -z localhost 10813"  # NBD server

# Check migration logs  
ssh pgrayson@10.0.100.231 "tail -100 /home/pgrayson/migratekit-cloudstack/migration-*.log"
```

## 🔐 **Security Considerations**

### **Production Hardening**
- **SSH Keys**: Use unique keys per deployment
- **TLS Certificates**: Use CA-signed certificates in production
- **Firewall Rules**: Restrict source IPs where possible
- **User Accounts**: Use dedicated service accounts with minimal privileges

### **Network Security**
- **Port Restrictions**: Verify only 22, 80, 443 are accessible
- **Tunnel Isolation**: Ensure tunnels are properly isolated
- **API Authentication**: Enable stronger authentication in production

## 📋 **Deployment Checklist**

- [ ] **OMA Services**: oma-api running on port 8082, migration-gui on port 3001
- [ ] **VMA Services**: vma-api + vma-tunnel-enhanced running  
- [ ] **SSH Tunnel**: Bidirectional connectivity verified
- [ ] **TLS Tunnel**: stunnel client/server operational
- [ ] **NBD Infrastructure**: Dynamic server startup working
- [ ] **Target Devices**: Block devices available for allocation
- [ ] **VM Discovery**: VMA can discover VMware inventory
- [ ] **Migration Test**: End-to-end migration successful
- [ ] **Monitoring**: Logging and metrics collection active

---
**Status**: ✅ **PRODUCTION DEPLOYMENT READY** - All components operational with real migration proven