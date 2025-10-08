# Sendense Deployment Packages

This directory contains versioned deployment packages for Sendense infrastructure components.

---

## 📦 **Available Packages**

### **SSH Tunnel Configuration**

| Version | Date | Status | Description |
|---------|------|--------|-------------|
| [v1.1.0](ssh-tunnel-v1.1.0/) | 2025-10-07 | ✅ Production | Simplified, reliable tunnel (30 lines) |
| v1.0.0 | 2025-10-06 | ⚠️ Deprecated | Complex implementation (205 lines) |

**Current Recommendation:** Use v1.1.0

---

## 🚀 **Quick Start**

### **SSH Tunnel Deployment**

**For SHA (Hub Appliance):**
```bash
cd ssh-tunnel-v1.1.0/sha
sudo ./deploy-sha-ssh-config.sh
```

**For SNA (Node Appliance):**
```bash
cd ssh-tunnel-v1.1.0/sna

# Local deployment
sudo ./deploy-sna-tunnel.sh

# Remote deployment
sshpass -p 'Password1' ./deploy-sna-tunnel.sh 10.0.100.231
```

**Full Documentation:** [ssh-tunnel-v1.1.0/README.md](ssh-tunnel-v1.1.0/README.md)

---

## 📁 **Directory Structure**

```
deployment/
├── README.md                           # This file
├── ssh-tunnel-v1.1.0/                  # Current SSH tunnel package
│   ├── README.md                       # Complete documentation
│   ├── CHANGELOG.md                    # Version history
│   ├── sha/                            # SHA configuration
│   │   ├── sshd_config.snippet        # SSH server config
│   │   └── deploy-sha-ssh-config.sh   # Deployment script
│   └── sna/                            # SNA configuration
│       ├── sendense-tunnel.sh          # Tunnel manager
│       ├── sendense-tunnel.service     # Systemd service
│       └── deploy-sna-tunnel.sh        # Deployment script
└── [future packages]/
```

---

## 📋 **Component Status**

### **SSH Tunnel v1.1.0**
- ✅ SHA configuration: Ready
- ✅ SNA tunnel: Ready
- ✅ 101 NBD ports (10100-10200)
- ✅ Auto-restart systemd service
- ⚠️ Reverse tunnel: Disabled (known issue)

### **Coming Soon**
- SNA API with backup endpoint
- Multi-disk backup coordination
- Monitoring and alerting
- High availability configurations

---

## 🔄 **Version Policy**

### **Semantic Versioning**
We use semantic versioning (MAJOR.MINOR.PATCH):
- **MAJOR**: Breaking changes, incompatible API changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

### **Support Policy**
- **Current version (v1.1.0):** Full support
- **Previous version (v1.0.0):** Security fixes only
- **Older versions:** Unsupported, upgrade recommended

### **Release Cycle**
- **Production releases:** Tested and stable
- **Beta releases:** Feature testing
- **Dev releases:** Experimental, not for production

---

## 📊 **Deployment Environments**

### **Development**
- **SHA:** 10.245.246.134
- **SNA:** 10.0.100.231
- **Status:** Active, daily updates

### **Production** (Future)
- **SHA Cluster:** TBD
- **SNA Fleet:** TBD
- **Status:** Planned

---

## 🔍 **Troubleshooting**

### **General Issues**

**Package not found:**
```bash
cd /home/oma_admin/sendense/deployment
ls -la
```

**Permission denied:**
```bash
sudo chmod +x ssh-tunnel-v1.1.0/*/deploy-*.sh
```

**Component-specific issues:**
See package README:
- [SSH Tunnel v1.1.0](ssh-tunnel-v1.1.0/README.md#troubleshooting)

---

## 📚 **Additional Resources**

### **Project Documentation**
- **Start Here:** `/home/oma_admin/sendense/start_here/`
- **Architecture:** `job-sheets/2025-10-07-unified-nbd-architecture.md`
- **Testing:** `TESTING-PGTEST1-CHECKLIST.md`
- **Session Summary:** `SESSION-SUMMARY-2025-10-07.md`

### **Development Guides**
- **Master Prompt:** `start_here/MASTER_AI_PROMPT.md`
- **Project Rules:** `start_here/PROJECT_RULES.md`
- **Binary Management:** `start_here/BINARY_MANAGEMENT.md`

---

## 🆘 **Support**

**For deployment issues:**
1. Check package-specific README
2. Review troubleshooting section
3. Check system logs: `journalctl -xe`
4. Contact: support@sendense.io

**For development:**
1. Review project documentation
2. Check job sheets for context
3. Follow project rules strictly

---

## 📈 **Changelog**

### October 7, 2025
- ✅ Added SSH Tunnel v1.1.0 (simplified, production-ready)
- ✅ Created deployment package structure
- ✅ Added comprehensive documentation

### October 6, 2025
- ⚠️ SSH Tunnel v1.0.0 (deprecated, too complex)

---

**Last Updated:** October 7, 2025  
**Maintainer:** Sendense Engineering Team

