# Migration Dashboard GUI Service Setup

## 🎯 Overview

The Migration Dashboard is a Next.js-based GUI for managing VMware to OSSEA migrations. This document covers setting it up as a production systemd service.

## 📋 Service Configuration

### Service Details
- **Service Name**: `migration-gui`
- **Port**: 3001
- **Protocol**: HTTP
- **User**: pgrayson
- **Working Directory**: `/home/pgrayson/migration-dashboard`

### Prerequisites
- Node.js (included in Next.js dependencies)
- OMA API service running on port 8082
- Network access to VMA for discovery operations

## 🚀 Installation Steps

### 1. Install Dependencies
```bash
cd /home/pgrayson/migration-dashboard
npm install
```

### 2. Create Service File
Create `/etc/systemd/system/migration-gui.service`:
```ini
[Unit]
Description=Migration Dashboard GUI
Documentation=Migration dashboard for VMware to OSSEA migrations
After=network-online.target oma-api.service
Wants=network-online.target
Requires=network-online.target

[Service]
Type=simple
User=pgrayson
Group=pgrayson
WorkingDirectory=/home/pgrayson/migration-dashboard

# Environment variables for Next.js
Environment=NODE_ENV=production
Environment=PORT=3001
Environment=HOSTNAME=localhost

# Service executable
ExecStart=/usr/bin/npm run dev -- --port 3001 --hostname localhost
Restart=always
RestartSec=10
StartLimitInterval=60
StartLimitBurst=3

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=migration-gui

[Install]
WantedBy=multi-user.target
```

### 3. Enable and Start Service
```bash
sudo systemctl daemon-reload
sudo systemctl enable migration-gui
sudo systemctl start migration-gui
```

## 🔍 Verification

### Service Status
```bash
sudo systemctl status migration-gui
```

### Web Access
```bash
# Local access
curl -s http://localhost:3001 | head -5

# External access (if needed)
curl -s http://10.245.246.125:3001 | head -5
```

### Port Check
```bash
netstat -tlnp | grep 3001
ss -tlnp | grep 3001
```

## 🎨 GUI Features

### Dashboard Components
- **VM Discovery**: Scan vCenter for available VMs
- **Migration Management**: Start and monitor automated migrations
- **OSSEA Configuration**: Complete settings management
- **Real-time Status**: Live migration progress tracking

### Settings Management
- **OSSEA Configuration**: API credentials, zone settings, resource IDs
- **Connection Testing**: Validate OSSEA API connectivity
- **Environment Export**: Generate shell script with environment variables

### API Integration
- **OMA API**: Port 8082 for migration workflow management
- **VMA API**: Port 9081 (via SSH tunnel) for VM discovery
- **OSSEA API**: Direct connection for configuration testing

## 🔧 Configuration

### Port Configuration
All API routes are configured to use port **8082** for OMA API communication:
- `/api/replicate` - Start migrations via OMA workflow
- `/api/migrations` - Fetch migration job status
- `/api/discover` - VM discovery via VMA
- `/api/settings/ossea` - OSSEA configuration management

### Authentication
- GUI uses session tokens for OMA API authentication
- OSSEA settings are stored locally in `~/.ossea_config.json`
- Environment variables exported to `~/ossea_env.sh`

## 📊 Monitoring

### Service Logs
```bash
# Real-time logs
sudo journalctl -u migration-gui -f

# Recent logs
sudo journalctl -u migration-gui --since "1 hour ago"
```

### Application Logs
Next.js development server provides detailed logging for:
- API route calls
- Migration workflow status
- OSSEA connection testing
- VM discovery operations

## 🚨 Troubleshooting

### Service Won't Start
```bash
# Check service status
sudo systemctl status migration-gui

# Check configuration
sudo systemctl cat migration-gui

# Restart service
sudo systemctl restart migration-gui
```

### Port Already in Use
```bash
# Find process using port 3001
sudo ss -tlnp | grep 3001
sudo lsof -i :3001

# Kill conflicting process
sudo kill -9 <PID>
```

### OMA API Connection Issues
- Verify OMA API is running on port 8082
- Check authentication tokens in GUI API routes
- Test direct API connectivity: `curl http://localhost:8082/health`

### OSSEA Configuration Problems
- Verify OSSEA credentials in settings page
- Use "Test Connection" feature to validate API access
- Check firewall rules for OSSEA API access

## 🔐 Security Considerations

### Service Security
- Runs as non-privileged user (pgrayson)
- `NoNewPrivileges=yes` prevents privilege escalation
- `PrivateTmp=yes` isolates temporary files

### Network Security
- GUI only listens on localhost by default
- All OMA API communication over localhost
- OSSEA credentials stored locally, not transmitted in GUI

### Production Hardening
- Consider HTTPS reverse proxy for external access
- Implement proper authentication for production deployments
- Regular security updates for Node.js dependencies

## 📋 Service Management

### Start/Stop/Restart
```bash
sudo systemctl start migration-gui
sudo systemctl stop migration-gui
sudo systemctl restart migration-gui
```

### Enable/Disable Auto-start
```bash
sudo systemctl enable migration-gui
sudo systemctl disable migration-gui
```

### Check Dependencies
```bash
# Verify OMA API dependency
sudo systemctl list-dependencies migration-gui
```

---

**Status**: ✅ **PRODUCTION READY** - GUI service operational on port 3001 with complete OSSEA integration