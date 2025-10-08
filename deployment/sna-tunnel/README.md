# Sendense SSH Tunnel Deployment

**Version:** 1.0.0  
**Date:** 2025-10-07  
**Purpose:** Deploy SSH tunnel infrastructure to SNA (Sendense Node Appliance)

---

## 📋 Overview

This deployment package establishes a persistent SSH tunnel between:
- **SNA** (Sendense Node Appliance) @ VMware
- **SHA** (Sendense Hub Appliance) @ 10.245.246.134

### Tunnel Configuration:
- **NBD Ports:** 10100-10200 (101 ports for concurrent backups)
- **SHA API:** Port 8082 (control plane)
- **Reverse Tunnel:** Port 9081 (SHA → SNA API)
- **Security:** Ed25519 SSH key authentication
- **Management:** Systemd with auto-restart

---

## 📦 Package Contents

```
sna-tunnel/
├── sendense-tunnel.sh         # Tunnel management script
├── sendense-tunnel.service    # Systemd service definition
├── deploy-to-sna.sh           # Automated deployment script
└── README.md                  # This file
```

---

## 🚀 Quick Deployment

### Option 1: Automated Deployment (Recommended)

```bash
# From SHA machine (current location)
cd /home/oma_admin/sendense/deployment/sna-tunnel
./deploy-to-sna.sh
```

The script will:
1. SSH to SNA (10.0.100.231)
2. Copy files to correct locations
3. Set permissions
4. Enable and start systemd service
5. Verify tunnel status

### Option 2: Manual Deployment

```bash
# 1. Copy files to SNA
scp -i ~/.ssh/cloudstack_key sendense-tunnel.sh \
    pgrayson@10.0.100.231:/tmp/

scp -i ~/.ssh/cloudstack_key sendense-tunnel.service \
    pgrayson@10.0.100.231:/tmp/

# 2. SSH to SNA
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231

# 3. On SNA, install files
sudo mv /tmp/sendense-tunnel.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/sendense-tunnel.sh
sudo chown root:root /usr/local/bin/sendense-tunnel.sh

sudo mv /tmp/sendense-tunnel.service /etc/systemd/system/
sudo chmod 644 /etc/systemd/system/sendense-tunnel.service
sudo chown root:root /etc/systemd/system/sendense-tunnel.service

# 4. Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable sendense-tunnel
sudo systemctl start sendense-tunnel

# 5. Verify status
sudo systemctl status sendense-tunnel
sudo journalctl -u sendense-tunnel -f
```

---

## ✅ Verification

### Check Tunnel Status
```bash
# On SNA
sudo systemctl status sendense-tunnel

# Expected output:
● sendense-tunnel.service - Sendense SSH Tunnel Manager
   Loaded: loaded (/etc/systemd/system/sendense-tunnel.service; enabled)
   Active: active (running)
```

### Check Logs
```bash
# Real-time logs
sudo journalctl -u sendense-tunnel -f

# Recent logs
sudo journalctl -u sendense-tunnel --since "5 minutes ago"

# Log file
sudo tail -f /var/log/sendense-tunnel.log
```

### Test Port Forwards
```bash
# On SNA, test NBD port forward
nc -zv localhost 10105
# Expected: Connection succeeded

# Test SHA API forward
curl http://localhost:8082/api/v1/health
# Expected: JSON response from SHA

# On SHA, test reverse tunnel
curl http://localhost:9081/api/v1/health
# Expected: JSON response from SNA VMA API
```

---

## 🔧 Configuration

### Environment Variables

Edit `/etc/systemd/system/sendense-tunnel.service`:

```ini
Environment="SHA_HOST=10.245.246.134"     # SHA IP address
Environment="SHA_PORT=443"                # SSH port on SHA
Environment="SSH_KEY=/home/vma/.ssh/cloudstack_key"  # SSH key path
```

After changes:
```bash
sudo systemctl daemon-reload
sudo systemctl restart sendense-tunnel
```

---

## 🛠️ Management Commands

### Start/Stop/Restart
```bash
sudo systemctl start sendense-tunnel
sudo systemctl stop sendense-tunnel
sudo systemctl restart sendense-tunnel
```

### Enable/Disable Auto-Start
```bash
sudo systemctl enable sendense-tunnel   # Start on boot
sudo systemctl disable sendense-tunnel  # Don't start on boot
```

### Monitor
```bash
# Status
sudo systemctl status sendense-tunnel

# Logs (live)
sudo journalctl -u sendense-tunnel -f

# Logs (last 100 lines)
sudo journalctl -u sendense-tunnel -n 100

# Check connection
ps aux | grep sendense-tunnel
```

---

## 🐛 Troubleshooting

### Tunnel Not Starting

**Check SSH key permissions:**
```bash
ls -l /home/vma/.ssh/cloudstack_key
# Expected: -rw------- (600)

# Fix if needed:
chmod 600 /home/vma/.ssh/cloudstack_key
```

**Check SHA reachability:**
```bash
ping -c 3 10.245.246.134
ssh -i ~/.ssh/cloudstack_key vma_tunnel@10.245.246.134 -p 443
```

### Tunnel Keeps Disconnecting

**Check logs for errors:**
```bash
sudo journalctl -u sendense-tunnel --since "10 minutes ago" | grep ERROR
```

**Common issues:**
- Network instability (check `ServerAliveInterval` settings)
- Port conflicts (check if ports already in use)
- SSH key authentication failure (check key on SHA)

### Port Forwarding Not Working

**Test local port:**
```bash
# On SNA
netstat -tuln | grep 10105
# Expected: LISTEN on 127.0.0.1:10105
```

**Test end-to-end:**
```bash
# On SNA
nc -zv localhost 10105

# If fails, check:
sudo journalctl -u sendense-tunnel | grep "10105"
```

---

## 📊 Performance Monitoring

### Connection Stats
```bash
# SSH connection
ps aux | grep "ssh.*sendense"

# Port usage
netstat -tuln | grep -E "1010[0-9]|8082|9081"

# Resource usage
systemctl show sendense-tunnel --property=MemoryCurrent
```

### Bandwidth Monitoring
```bash
# Install iftop (if needed)
sudo apt-get install iftop

# Monitor traffic on SSH connection
sudo iftop -i eth0 -f "port 443"
```

---

## 🔐 Security Notes

- **SSH Key:** Ed25519, 600 permissions, never share
- **Tunnel User:** `vma_tunnel` (restricted, no shell)
- **Port Range:** Only 10100-10200, 8082, 9081 exposed
- **No PTY:** SSH runs with `-N` (no interactive shell)
- **Systemd Hardening:** NoNewPrivileges, PrivateTmp, ProtectSystem

---

## 📝 Architecture Diagram

```
┌─────────────────────────────────────┐
│  SNA (Sendense Node Appliance)      │
│  @ VMware (10.0.100.231)            │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ SendenseBackupClient (SBC)  │   │
│  │ Reads VMware → Writes NBD   │   │
│  └──────────┬──────────────────┘   │
│             │                       │
│             │ localhost:10105       │
│             ▼                       │
│  ┌─────────────────────────────┐   │
│  │ SSH Tunnel (sendense-tunnel)│   │
│  │ Forward: 10100-10200, 8082  │   │
│  │ Reverse: 9081               │   │
│  └──────────┬──────────────────┘   │
└─────────────┼────────────────────────┘
              │ SSH over port 443
              ▼
┌─────────────────────────────────────┐
│  SHA (Sendense Hub Appliance)       │
│  @ 10.245.246.134                   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ qemu-nbd (Port 10105)       │   │
│  │ Exposes QCOW2 via NBD       │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ SHA API (Port 8082)         │   │
│  │ Backup orchestration        │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ Reverse Tunnel (Port 9081)  │   │
│  │ Access to SNA VMA API       │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

---

## 📞 Support

- **Documentation:** https://sendense.io/docs
- **Issues:** File in project issue tracker
- **Logs:** Always include `/var/log/sendense-tunnel.log` and systemd journal

---

**Deployed:** $(date)  
**By:** $(whoami)@$(hostname)
