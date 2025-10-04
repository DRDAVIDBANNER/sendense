# SSH Tunnel Architecture - Production Grade

## Overview

MigrateKit OSSEA uses a **surgically restricted SSH tunnel on port 443** for bidirectional communication between VMA and OMA appliances. This architecture provides enterprise-grade security while maintaining complete functionality for VMA-OMA communication.

**Date:** September 29, 2025  
**Status:** ✅ PRODUCTION READY  
**Deployment Script:** `/home/pgrayson/migratekit-cloudstack/scripts/deploy-production-ssh-tunnel.sh`

---

## Architecture Summary

```
VMA (10.0.100.232) ←→ SSH Tunnel (Port 443) ←→ OMA (10.245.246.125)
                       
Security Layer:
├── Port 443 only (internet-exposed)
├── Ed25519 public key authentication
├── Surgical SSH restrictions (no PTY, no X11, no agent forwarding)
├── No interactive shell access
├── Limited port forwarding (port 9081 only)
└── Forced command execution (/bin/true)

Application Layer:
└── Reverse Tunnel: OMA:9081 → VMA API:8081
    ├── VMA progress polling
    ├── Job management
    ├── Replication control
    └── Real-time status updates
```

---

## Key Features

### 🔒 Security Features

1. **Public Key Authentication Only**
   - Ed25519 keys generated via VMA enrollment
   - No password authentication
   - Keys stored securely in `/opt/vma/enrollment/`

2. **Surgical SSH Restrictions**
   - No PTY allocation (no interactive terminal)
   - No X11 forwarding
   - No SSH agent forwarding
   - No user RC file execution
   - Forced command execution (`/bin/true`)
   - Limited port listening (127.0.0.1:9081 only)

3. **SSH Daemon Restrictions**
   - AuthenticationMethods: publickey only
   - PasswordAuthentication: disabled
   - KbdInteractiveAuthentication: disabled
   - PermitTTY: disabled
   - AllowTcpForwarding: remote only
   - PermitOpen: none
   - PermitListen: 127.0.0.1:9081 only

### 🚀 Operational Features

1. **Systemd Service Management**
   - Auto-start on boot
   - Auto-restart on failure
   - 10-second restart delay
   - Full journal logging

2. **Connection Resilience**
   - ServerAliveInterval: 30 seconds
   - ServerAliveCountMax: 3 attempts
   - ExitOnForwardFailure: yes (fail fast)
   - Automatic reconnection via systemd

3. **Monitoring & Management**
   - Full systemd integration
   - Journal logging
   - Status checking commands
   - Health endpoint verification

---

## Components

### OMA Side

**User:** `vma_tunnel`
- System user (UID 995)
- Home directory: `/var/lib/vma_tunnel`
- Shell: `/bin/bash`
- Purpose: Accept SSH reverse tunnels from VMAs

**SSH Configuration:**
- Port 443 enabled in `/etc/ssh/sshd_config`
- Match User block for `vma_tunnel` with restrictions
- Authorized keys in `/var/lib/vma_tunnel/.ssh/authorized_keys`

**Setup Script:**
```bash
/home/pgrayson/migratekit-cloudstack/source/current/oma/scripts/setup-oma-ssh-tunnel.sh
```

### VMA Side

**User:** `vma`
- Standard VMA user
- SSH keys in `/opt/vma/enrollment/`
- Generated via enrollment wizard

**Systemd Service:** `vma-ssh-tunnel.service`
- Location: `/etc/systemd/system/vma-ssh-tunnel.service`
- Wrapper script: `/usr/local/bin/vma-tunnel-wrapper.sh`
- Auto-enabled and auto-started

**Setup Script:**
```bash
/home/pgrayson/migratekit-cloudstack/source/current/vma/scripts/setup-vma-ssh-tunnel.sh
```

---

## Deployment

### One-Command Production Deployment

**Location:** `/home/pgrayson/migratekit-cloudstack/scripts/deploy-production-ssh-tunnel.sh`

**Usage:**
```bash
cd /home/pgrayson/migratekit-cloudstack
./scripts/deploy-production-ssh-tunnel.sh <OMA_IP> <VMA_IP>
```

**Example:**
```bash
./scripts/deploy-production-ssh-tunnel.sh 10.245.246.125 10.0.100.232
```

**What it does:**
1. ✅ Cleans existing VMA deployment
2. ✅ Sets up OMA SSH infrastructure
3. ✅ Verifies VMA enrollment keys exist
4. ✅ Configures hardened SSH authentication
5. ✅ Deploys VMA tunnel wrapper script
6. ✅ Creates VMA systemd service
7. ✅ Enables and starts service
8. ✅ Tests bidirectional connectivity
9. ✅ Reports complete status

---

## Management

### Status Checking

```bash
# Check VMA tunnel service status
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl status vma-ssh-tunnel'

# Check if port 9081 is listening on OMA
ss -tlnp | grep 9081

# Test VMA API via reverse tunnel
curl http://127.0.0.1:9081/api/v1/health
```

### Log Monitoring

```bash
# View VMA tunnel logs (live)
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo journalctl -u vma-ssh-tunnel -f'

# View last 50 lines of logs
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo journalctl -u vma-ssh-tunnel -n 50'
```

### Service Management

```bash
# Restart VMA tunnel
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl restart vma-ssh-tunnel'

# Stop VMA tunnel
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl stop vma-ssh-tunnel'

# Start VMA tunnel
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl start vma-ssh-tunnel'

# Disable auto-start
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl disable vma-ssh-tunnel'

# Enable auto-start
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl enable vma-ssh-tunnel'
```

---

## Security Configuration

### OMA: /etc/ssh/sshd_config

```
Port 443

# VMA Tunnel Security - Production Hardening
Match User vma_tunnel
    AuthenticationMethods publickey
    PubkeyAuthentication yes
    PasswordAuthentication no
    KbdInteractiveAuthentication no
    PermitTTY no
    X11Forwarding no
    AllowTcpForwarding remote
    AllowStreamLocalForwarding no
    GatewayPorts no
    PermitOpen none
    PermitListen 127.0.0.1:9081
```

### OMA: /var/lib/vma_tunnel/.ssh/authorized_keys

```
no-pty,no-X11-forwarding,no-agent-forwarding,no-user-rc,permitlisten="127.0.0.1:9081",command="/bin/true" ssh-ed25519 AAAAC3Nza... VMA-vma-10.245.246.125-202509291728
```

### VMA: /etc/systemd/system/vma-ssh-tunnel.service

```ini
[Unit]
Description=VMA SSH Tunnel to OMA
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=vma
Group=vma
ExecStart=/usr/local/bin/vma-tunnel-wrapper.sh 10.245.246.125
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### VMA: /usr/local/bin/vma-tunnel-wrapper.sh

```bash
#!/bin/bash
set -euo pipefail

OMA_IP="${1:-10.245.246.125}"

echo "Starting VMA SSH tunnel to $OMA_IP..."

exec /usr/bin/ssh -i /opt/vma/enrollment/vma_enrollment_key -p 443 -N \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=3 \
    -o ExitOnForwardFailure=yes \
    -R 127.0.0.1:9081:127.0.0.1:8081 \
    vma_tunnel@$OMA_IP
```

---

## Troubleshooting

### Tunnel Not Establishing

**Check VMA service status:**
```bash
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl status vma-ssh-tunnel'
```

**Check VMA service logs:**
```bash
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo journalctl -u vma-ssh-tunnel -n 50'
```

**Common errors:**
- `Permission denied (publickey)` - SSH key mismatch, check authorized_keys
- `remote port forwarding failed for listen port 9081` - Port already in use
- `Connection refused` - SSH not listening on port 443

### Port 9081 Already in Use

```bash
# Find what's using port 9081
sudo lsof -i :9081

# Kill the process
sudo kill <PID>

# Restart VMA tunnel service
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl restart vma-ssh-tunnel'
```

### SSH Key Issues

**Verify VMA enrollment key exists:**
```bash
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'ls -l /opt/vma/enrollment/vma_enrollment_key*'
```

**Check key permissions:**
```bash
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'stat -c "%a %U:%G %n" /opt/vma/enrollment/vma_enrollment_key*'
```

**Expected permissions:**
- Private key: 600 (vma:vma)
- Public key: 644 (vma:vma)

### Re-deploy from Scratch

```bash
cd /home/pgrayson/migratekit-cloudstack
./scripts/deploy-production-ssh-tunnel.sh 10.245.246.125 10.0.100.232
```

---

## Testing

### Health Check

```bash
curl -s http://127.0.0.1:9081/api/v1/health | jq
```

**Expected output:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-29T19:47:37Z",
  "uptime": "460ns"
}
```

### Port Listening Check

```bash
ss -tlnp | grep 9081
```

**Expected output:**
```
LISTEN 0      128        127.0.0.1:9081       0.0.0.0:*
```

### Service Status Check

```bash
ssh -i ~/.ssh/vma_232_key vma@10.0.100.232 'sudo systemctl is-active vma-ssh-tunnel'
```

**Expected output:**
```
active
```

---

## Future Enhancements

### ⚠️ TODO: Auto-Recovery Monitoring Service

**Requirement:** VMA needs a monitoring service to detect and recover from tunnel failures.

**Current State:**
- ✅ Systemd auto-restart on process failure
- ❌ No monitoring for network-level failures
- ❌ No OMA-side health checks

**Proposed Enhancement:**
Create a VMA monitoring service that:
1. Periodically checks tunnel connectivity to OMA
2. Monitors SSH process health
3. Detects network-level failures
4. Triggers manual recovery if systemd restart fails
5. Sends alerts on repeated failures

**Implementation Plan:**
- Service name: `vma-tunnel-monitor.service`
- Check interval: 60 seconds
- Failure threshold: 3 consecutive failures
- Action: Force restart tunnel service
- Logging: Full failure context to journal

---

## Migration Notes

### Replaced Technologies

**Previous:** stunnel (TLS tunneling)
- **Reason for change:** Complexity, port conflicts with enrollment-proxy
- **Migration:** Complete replacement with SSH tunnel
- **Status:** All stunnel references removed from codebase

**Previous:** enrollment-proxy on port 443
- **Reason for change:** Port conflict with SSH tunnel
- **Migration:** VMA enrollment uses SSH key generation
- **Status:** Enrollment system integrated with SSH tunnel

### Breaking Changes

- ⚠️ Port 443 now used exclusively for SSH tunnel
- ⚠️ VMA enrollment must complete before tunnel setup
- ⚠️ Old stunnel configurations no longer supported

---

## Production Checklist

Before deploying to production VMA:

- [ ] VMA enrollment wizard completed
- [ ] SSH keys generated in `/opt/vma/enrollment/`
- [ ] OMA has `vma_tunnel` user configured
- [ ] OMA SSH daemon listening on port 443
- [ ] VMA can reach OMA on port 443
- [ ] Deployment script tested on dev environment
- [ ] Health checks passing
- [ ] Auto-restart verified

---

## Support

For issues or questions:
1. Check this documentation
2. Review systemd logs: `journalctl -u vma-ssh-tunnel`
3. Test connectivity: `curl http://127.0.0.1:9081/api/v1/health`
4. Re-deploy if needed: `./scripts/deploy-production-ssh-tunnel.sh`

---

**Last Updated:** September 29, 2025  
**Version:** 1.0 (Production Ready)  
**Maintained By:** MigrateKit OSSEA Team


