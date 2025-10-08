# Sendense SSH Tunnel Configuration v1.1.0

**Release Date:** October 7, 2025  
**Status:** Production Ready  
**Architecture:** Simplified, reliable tunnel with 101 NBD ports

---

## 📋 **Overview**

This package contains the SSH tunnel configuration for Sendense distributed backup architecture. The tunnel enables secure communication between SNA (Node Appliance) and SHA (Hub Appliance) for multi-disk VM backups.

### **Architecture**

```
┌─────────────────────────────────────────────────────────────────┐
│                  SNA (Sendense Node Appliance)                  │
│                    10.0.100.231:443 (outbound)                  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ sendense-tunnel.service                                  │  │
│  │ - 101 NBD port forwards (10100-10200)                   │  │
│  │ - SHA API forward (8082)                                │  │
│  │ - Auto-restart, systemd managed                         │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              ↓ SSH Tunnel (port 443)
                              ↓ Ed25519 key auth
                              ↓ vma_tunnel user
┌─────────────────────────────────────────────────────────────────┐
│                  SHA (Sendense Hub Appliance)                   │
│                         10.245.246.134                          │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ sshd with Match User vma_tunnel                         │  │
│  │ - PermitOpen: 9081, 10809, 8082                         │  │
│  │ - AllowTcpForwarding: yes                               │  │
│  │ - Security: pubkey only, no PTY, no X11                 │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ qemu-nbd processes on ports 10100-10200                 │  │
│  │ - One per disk being backed up                          │  │
│  │ - --shared=10 for multiple connections                  │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📁 **Package Contents**

```
ssh-tunnel-v1.1.0/
├── README.md                           # This file
├── CHANGELOG.md                        # Version history
├── sha/                                # SHA (Hub) configuration
│   ├── sshd_config.snippet            # SSH server config
│   └── deploy-sha-ssh-config.sh       # Deployment script
└── sna/                                # SNA (Node) configuration
    ├── sendense-tunnel.sh              # Tunnel manager script
    ├── sendense-tunnel.service         # Systemd service
    └── deploy-sna-tunnel.sh            # Deployment script
```

---

## 🚀 **Quick Start**

### **Prerequisites**

1. **SHA Requirements:**
   - SSH server running on port 443
   - `vma_tunnel` user exists
   - Ed25519 public key configured

2. **SNA Requirements:**
   - SSH client installed
   - Ed25519 private key at `/home/vma/.ssh/cloudstack_key`
   - `vma` user exists

### **Step 1: Deploy SHA Configuration**

```bash
cd ssh-tunnel-v1.1.0/sha
sudo ./deploy-sha-ssh-config.sh
```

**What it does:**
- Backs up existing sshd_config
- Adds vma_tunnel Match block
- Tests configuration
- Reloads SSH daemon

### **Step 2: Deploy SNA Tunnel**

**Local deployment (on SNA):**
```bash
cd ssh-tunnel-v1.1.0/sna
sudo ./deploy-sna-tunnel.sh
```

**Remote deployment (from SHA):**
```bash
cd ssh-tunnel-v1.1.0/sna
sshpass -p 'Password1' ./deploy-sna-tunnel.sh 10.0.100.231
```

**What it does:**
- Installs tunnel script to `/usr/local/bin/`
- Installs systemd service
- Enables auto-start
- Starts tunnel

### **Step 3: Verify**

**On SNA:**
```bash
# Check service status
systemctl status sendense-tunnel

# Check tunnel process
ps aux | grep sendense-tunnel

# Verify forwarded ports
netstat -an | grep LISTEN | grep 1010
```

**On SHA:**
```bash
# Check vma_tunnel connections
sudo netstat -tnp | grep vma_tunnel

# Verify NBD ports accessible from SNA
# (run from SNA)
curl http://localhost:8082/health
```

---

## 🔧 **Configuration**

### **Port Forwarding**

**Forward Tunnels (SNA → SHA):**
- `10100-10200`: NBD data ports (101 concurrent backups)
- `8082`: SHA API endpoint

**Reverse Tunnel (SHA → SNA):**
- `9081`: SNA API endpoint
- **Status:** Disabled in v1.1.0 due to SSH config issues
- **Workaround:** Direct SNA:8081 access if needed

### **Environment Variables**

Set in systemd service file (`sendense-tunnel.service`):

```ini
Environment="SHA_HOST=10.245.246.134"
Environment="SHA_PORT=443"
Environment="SSH_KEY=/home/vma/.ssh/cloudstack_key"
```

**To customize:**
```bash
sudo systemctl edit sendense-tunnel
```

### **Security Settings**

**SHA sshd_config:**
- Authentication: Ed25519 public key only
- No PTY allocation
- No X11 forwarding
- Restricted port forwarding (PermitOpen)
- Dedicated vma_tunnel user (UID 997)

**SNA tunnel:**
- User: `vma` (non-root)
- systemd security hardening:
  - NoNewPrivileges=true
  - PrivateTmp=true
  - ProtectSystem=strict
  - ProtectHome=read-only

---

## 📊 **Monitoring**

### **Service Status**

```bash
# On SNA
systemctl status sendense-tunnel

# View logs
journalctl -u sendense-tunnel -f

# Check uptime
systemctl show sendense-tunnel --property=ActiveEnterTimestamp
```

### **Connection Health**

```bash
# On SHA: Check established tunnels
sudo ss -tnp | grep :443 | grep vma_tunnel

# On SNA: Test forwarded ports
for port in {10100..10105}; do
    nc -zv localhost $port && echo "Port $port: OK"
done

# Test SHA API access
curl http://localhost:8082/health
```

### **Performance Metrics**

```bash
# Tunnel bandwidth (on SNA)
watch -n 1 'cat /proc/net/dev | grep -A1 eth0'

# SSH process stats
ps aux | grep sendense-tunnel
```

---

## 🔍 **Troubleshooting**

### **Service Won't Start**

```bash
# Check logs
journalctl -u sendense-tunnel --since "5 minutes ago"

# Common issues:
# 1. SSH key missing
ls -la /home/vma/.ssh/cloudstack_key

# 2. SHA unreachable
ping -c 3 10.245.246.134
ssh -i /home/vma/.ssh/cloudstack_key -p 443 vma_tunnel@10.245.246.134 echo "OK"

# 3. Service file syntax
systemd-analyze verify sendense-tunnel.service
```

### **Tunnel Connects But Drops**

```bash
# Check ServerAlive settings
grep ServerAlive /usr/local/bin/sendense-tunnel.sh

# Monitor for drops
journalctl -u sendense-tunnel -f | grep -i "exit\|disconnect\|timeout"

# Check network stability
mtr -c 100 10.245.246.134
```

### **Port Forwarding Fails**

```bash
# On SHA: Check sshd config
sudo sshd -T | grep -i permitopen

# Verify vma_tunnel Match block
sudo grep -A10 "Match User vma_tunnel" /etc/ssh/sshd_config

# Test SSH config
sudo sshd -t
```

### **Performance Issues**

```bash
# Check tunnel CPU/memory
ps aux | grep sendense-tunnel

# Monitor connection count
netstat -an | grep ESTABLISHED | grep -E "1010[0-9]|1011[0-9]|1012[0-9]" | wc -l

# Check for packet loss
ping -c 100 10.245.246.134 | grep loss
```

---

## 🔄 **Maintenance**

### **Restart Tunnel**

```bash
sudo systemctl restart sendense-tunnel
```

### **Update Configuration**

```bash
# Edit script
sudo vi /usr/local/bin/sendense-tunnel.sh

# Edit service
sudo systemctl edit --full sendense-tunnel

# Apply changes
sudo systemctl daemon-reload
sudo systemctl restart sendense-tunnel
```

### **Upgrade**

```bash
# Download new version
cd /tmp
wget https://sendense.io/downloads/ssh-tunnel-v1.2.0.tar.gz
tar xzf ssh-tunnel-v1.2.0.tar.gz

# Deploy
cd ssh-tunnel-v1.2.0/sna
sudo ./deploy-sna-tunnel.sh
```

---

## 📈 **Version History**

### **v1.1.0** (October 7, 2025)
- ✅ Simplified tunnel script (30 lines vs 205)
- ✅ 101 NBD ports (10100-10200)
- ✅ Removed preflight checks (caused failures)
- ✅ Removed reverse tunnel (SSH config issues)
- ✅ Production-ready systemd service
- ✅ Auto-restart on failure

### **v1.0.0** (October 6, 2025)
- Initial complex implementation (205 lines)
- Preflight checks
- Comprehensive logging
- Reverse tunnel support
- **Issues:** Too complex, preflight false positives

---

## ⚠️ **Known Issues**

### **Issue 1: Reverse Tunnel Disabled**
- **Problem:** Port 9081 reverse tunnel fails with "remote port forwarding failed"
- **Impact:** SHA cannot access SNA API via tunnel
- **Workaround:** Direct SNA:8081 access (if network allows)
- **Status:** Under investigation
- **ETA:** v1.2.0

### **Issue 2: Port Range Limitation**
- **Current:** 101 ports (10100-10200)
- **Scalability:** Supports 101 concurrent backups max
- **Future:** Dynamic port allocation if needed

---

## 📚 **Additional Resources**

- **Project Documentation:** `/home/oma_admin/sendense/start_here/`
- **Architecture Docs:** `job-sheets/2025-10-07-unified-nbd-architecture.md`
- **Deployment Guides:** `DEPLOYMENT-DEV-CHECKLIST.md`
- **Testing:** `TESTING-PGTEST1-CHECKLIST.md`

---

## 🆘 **Support**

**For issues:**
1. Check logs: `journalctl -u sendense-tunnel -f`
2. Review troubleshooting section above
3. Check project documentation
4. Contact: support@sendense.io

---

**Version:** 1.1.0  
**Status:** Production Ready  
**Last Updated:** October 7, 2025

