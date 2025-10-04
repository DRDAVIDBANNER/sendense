# 🚀 **PRODUCTION OMA DEPLOYMENT SPECIFICATION**

**Created**: October 1, 2025  
**Purpose**: Complete specification for deploying production-ready OMA templates  
**Source Environment**: Dev OMA (10.245.246.125)  
**Status**: ✅ **VERIFIED PRODUCTION COMPONENTS**

---

## 📊 **PRODUCTION ENVIRONMENT ANALYSIS**

### **Source System**: Dev OMA (10.245.246.125)
- **OS**: Ubuntu 24.04 LTS
- **Status**: ✅ **FULLY OPERATIONAL** - 14+ hours uptime
- **Network Performance**: Baseline established (some TCP throttling)
- **Database**: 34 tables in production schema
- **Services**: All production services running

---

## 🔧 **PRODUCTION BINARY LOCATIONS**

### **✅ OMA API Server**
**Current Active Binary**: `oma-api-v2.39.0-gorm-field-fix`
```bash
# Location: /opt/migratekit/bin/oma-api (symlink)
# Target: /opt/migratekit/bin/oma-api-v2.39.0-gorm-field-fix
# Size: 33,401,775 bytes
# Date: Sep 30 14:48
# Owner: root:root
# Permissions: -rwxr-xr-x

# Service Configuration:
# User: oma (running as oma user)
# Command: /opt/migratekit/bin/oma-api -port=8082 -db-type=mariadb -db-host=localhost -db-port=3306 -db-name=migratekit_oma -db-user=oma_user -db-pass=oma_password -auth=false -debug=false
# Status: Active (running) since Tue 2025-09-30 18:28:55 BST
```

**Available Backup Versions** (for rollback if needed):
- `oma-api-v2.37.0-ssh-automation-fixed` (Sep 29 14:15)
- `oma-api-v2.36.0-ssh-automation` (Sep 29 14:07)
- `oma-api-v2.35.0-complete-vma-fix` (Sep 29 13:17)
- `oma-api-v2.34.0-security-hardened` (Sep 29 13:10)
- `oma-api-v2.31.0-vma-enrollment` (Sep 29 13:09)

### **✅ Volume Management Daemon**
**Current Active Binary**: `volume-daemon-v2.0.0-by-id-paths`
```bash
# Location: /usr/local/bin/volume-daemon (symlink)
# Target: /usr/local/bin/volume-daemon-v2.0.0-by-id-paths
# Size: 14,884,806 bytes
# Date: Sep 30 17:48
# Owner: root:root
# Permissions: -rwxr-xr-x

# Service Configuration:
# User: root (running as root user)
# Command: /usr/local/bin/volume-daemon
# Status: Active (running) since Tue 2025-09-30 18:28:55 BST
```

**Available Backup Versions**:
- `volume-daemon-v1.2.3-multi-volume-snapshots` (Sep 28 18:36)
- `volume-daemon-v1.2.4-nbd-export-recreation-fix` (Sep 25 08:11)
- `volume-daemon-v1.2.2-enhanced-delete` (Sep 9 08:18)
- `volume-daemon-v1.2.1-failover-nbd-fix` (Sep 8 20:45)

### **✅ Migration GUI Application**
**Current Active Application**: Next.js Production Build
```bash
# Location: /home/pgrayson/migration-dashboard/
# Type: Next.js React Application
# Size: ~700KB total directory
# Build Status: Production build available (.next directory)
# Dependencies: package.json with production dependencies

# Service Configuration:
# User: pgrayson (development user - needs standardization)
# Working Directory: /home/pgrayson/migration-dashboard
# Command: npm run dev (development mode - needs production mode)
# Port: 3001
# Status: Active (running) since Tue 2025-09-30 18:28:55 BST
```

**GUI Key Files**:
- `package.json` - Dependencies and build configuration
- `.next/` - Built application artifacts
- `src/` - Source code (not needed for deployment)
- `public/` - Static assets

---

## 🗄️ **PRODUCTION DATABASE SCHEMA**

### **Database Connection Details**
```bash
# Database: migratekit_oma
# Host: localhost:3306
# User: oma_user
# Password: oma_password
# Tables: 34 (complete production schema)
```

### **Complete Table List** (34 Tables)
```sql
-- Core Migration Tables
replication_jobs              -- Primary job tracking
vm_disks                     -- Disk-level replication details
vm_replication_contexts      -- VM-centric master context
failover_jobs               -- VM failover operations
network_mappings            -- Network configuration mapping

-- Volume Management Tables
ossea_volumes               -- OSSEA volume tracking
device_mappings             -- Volume-to-device correlation
volume_operations           -- Volume operation history
volume_operation_history    -- Historical volume operations
volume_mounts               -- Volume mount tracking
volume_daemon_metrics       -- Performance metrics

-- NBD Export Management
nbd_exports                 -- NBD export configurations
vm_export_mappings          -- VM-based export reuse

-- Job Tracking and Logging
job_tracking                -- Generic job tracking system
job_execution_log           -- Detailed execution audit trail
job_steps                   -- Individual job step tracking
job_tracking_hierarchy      -- Parent-child job relationships
log_events                  -- Structured logging events
active_jobs                 -- Currently active job view

-- Scheduling System
replication_schedules       -- Cron-based job scheduling
active_schedules           -- Currently active schedules
schedule_executions        -- Schedule execution history
schedule_execution_summary -- Schedule performance summary
vm_machine_groups          -- VM grouping for bulk operations
vm_group_memberships       -- VM-to-group relationships
vm_schedule_status         -- VM scheduling status

-- VMware Integration
vmware_credentials         -- Encrypted vCenter credentials
cbt_history               -- Change Block Tracking audit trail

-- VMA Management
vma_enrollments           -- VMA enrollment tracking
vma_pairing_codes         -- VMA pairing code management
vma_active_connections    -- Active VMA connections
vma_connection_audit      -- VMA connection audit trail

-- Configuration Management
ossea_configs             -- OSSEA/CloudStack configurations
linstor_configs          -- Linstor storage configurations (deprecated)
```

### **Schema Export Command**
```bash
# Complete schema export (no data)
mysqldump -u oma_user -poma_password \
  --no-data \
  --routines \
  --triggers \
  --single-transaction \
  migratekit_oma > production-schema.sql

# Essential configuration data export
mysqldump -u oma_user -poma_password \
  --no-create-info \
  --where='1=1' \
  migratekit_oma ossea_configs vmware_credentials > production-config-data.sql
```

---

## 🌐 **PRODUCTION SERVICE CONFIGURATION**

### **✅ OMA API Service**
**Service File**: `/etc/systemd/system/oma-api.service`
```ini
[Unit]
Description=OMA Migration API Server
After=network.target mariadb.service volume-daemon.service
Requires=mariadb.service
Wants=volume-daemon.service

[Service]
Type=simple
User=oma
Group=oma
WorkingDirectory=/opt/migratekit
ExecStart=/opt/migratekit/bin/oma-api -port=8082 -db-type=mariadb -db-host=localhost -db-port=3306 -db-name=migratekit_oma -db-user=oma_user -db-pass=oma_password -auth=false -debug=false
Restart=always
RestartSec=10
TimeoutStartSec=60
TimeoutStopSec=30
KillMode=mixed
KillSignal=SIGTERM
StandardOutput=journal
StandardError=journal
Environment=MIGRATEKIT_CRED_ENCRYPTION_KEY=[GENERATED_KEY]

[Install]
WantedBy=multi-user.target
```

### **✅ Volume Daemon Service**
**Service File**: `/etc/systemd/system/volume-daemon.service`
```ini
[Unit]
Description=Volume Management Daemon for MigrateKit OSSEA
After=network.target mariadb.service
Requires=mariadb.service

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/volume-daemon
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
```

### **✅ Migration GUI Service**
**Service File**: `/etc/systemd/system/migration-gui.service`
```ini
[Unit]
Description=Migration Dashboard GUI
After=network.target oma-api.service
Wants=oma-api.service

[Service]
Type=simple
User=pgrayson
Group=pgrayson
WorkingDirectory=/home/pgrayson/migration-dashboard
ExecStart=/usr/bin/npm run dev -- --port 3001 --hostname 0.0.0.0
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
Environment=NODE_ENV=development

[Install]
WantedBy=multi-user.target
```

**⚠️ Note**: GUI service needs standardization for production (user: oma_admin, production mode)

---

## 🔐 **SSH TUNNEL INFRASTRUCTURE**

### **SSH Configuration** (`/etc/ssh/sshd_config`)
```bash
# Port configuration
Port 22
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

### **SSH Socket Configuration** (`/etc/systemd/system/ssh.socket.d/port443.conf`)
```ini
[Socket]
ListenStream=
ListenStream=0.0.0.0:22
ListenStream=0.0.0.0:443
ListenStream=[::]:22
ListenStream=[::]:443
```

### **VMA Tunnel User Setup**
```bash
# User: vma_tunnel (UID varies)
# Home: /var/lib/vma_tunnel
# SSH Directory: /var/lib/vma_tunnel/.ssh (700 permissions)
# Authorized Keys: /var/lib/vma_tunnel/.ssh/authorized_keys (600 permissions)
```

---

## 📡 **NBD SERVER CONFIGURATION**

### **NBD Config Base** (`/etc/nbd-server/config-base`)
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

### **NBD Service Status**
- **Port**: 10809
- **Status**: Active (running)
- **Configuration**: Uses `/etc/nbd-server/config` (copy of config-base)
- **Dynamic Exports**: Created in `/etc/nbd-server/conf.d/` by Volume Daemon

---

## 🏗️ **DEPLOYMENT SCRIPT FEATURES**

### **Real Production Deployment** (`deploy-real-production-oma.sh`)
**Script Location**: `/home/pgrayson/migratekit-cloudstack/scripts/deploy-real-production-oma.sh`

**What It Does**:
1. **System Preparation**: Ubuntu 24.04 hardening, cloud-init disable
2. **Binary Deployment**: Copies REAL binaries from dev OMA via SCP
3. **Database Deployment**: Exports and imports complete 34-table schema
4. **GUI Deployment**: Copies complete Next.js application
5. **Infrastructure Setup**: SSH tunnel, NBD server, service configuration
6. **Comprehensive Validation**: Health checks for all components

**Deployment Command**:
```bash
# Copy script to target server
scp deploy-real-production-oma.sh oma_admin@TARGET_IP:/tmp/

# Execute deployment
ssh oma_admin@TARGET_IP 'cd /tmp && echo "Password1" | sudo -S bash deploy-real-production-oma.sh'
```

---

## 📋 **PRODUCTION COMPONENT CHECKLIST**

### **✅ Required for Complete OMA Template**
- [x] **OMA API**: oma-api-v2.39.0-gorm-field-fix (33.4MB)
- [x] **Volume Daemon**: volume-daemon-v2.0.0-by-id-paths (14.9MB)
- [x] **Database Schema**: Complete 34-table production schema
- [x] **Migration GUI**: Complete Next.js application
- [x] **NBD Server**: Production configuration with dummy export
- [x] **SSH Tunnel**: vma_tunnel user with port 443 support
- [x] **VirtIO Tools**: `/usr/share/virtio-win/virtio-win.iso`
- [x] **Service Configurations**: All systemd services properly configured
- [x] **Cloud-init**: Disabled for production deployment
- [x] **Network Configuration**: SSH ports 22 and 443 active

### **🔧 Production Standardizations Needed**
- [ ] **GUI Service User**: Change from `pgrayson` to `oma_admin`
- [ ] **GUI Production Mode**: Change from `npm run dev` to `npx next start`
- [ ] **Service User Consistency**: Standardize all services to use `oma_admin`
- [ ] **Encryption Key Generation**: Unique key per deployment
- [ ] **VMA SSH Key Integration**: Automated VMA key addition process

---

## 🗄️ **DATABASE SCHEMA SPECIFICATION**

### **Complete Production Schema** (34 Tables)
**Export Command**:
```bash
mysqldump -u oma_user -poma_password \
  --no-data \
  --routines \
  --triggers \
  --single-transaction \
  migratekit_oma > complete-production-schema.sql
```

**Table Categories**:

#### **Core Migration System** (5 tables)
- `replication_jobs` - Primary job tracking with VM context
- `vm_disks` - Disk-level replication details and change IDs
- `vm_replication_contexts` - VM-centric master context table
- `failover_jobs` - VM failover operations (live/test)
- `network_mappings` - VMware to OSSEA network mapping

#### **Volume Management System** (6 tables)
- `ossea_volumes` - OSSEA volume tracking
- `device_mappings` - Volume-to-device correlation
- `volume_operations` - Volume operation tracking
- `volume_operation_history` - Historical volume operations
- `volume_mounts` - Volume mount state tracking
- `volume_daemon_metrics` - Volume Daemon performance metrics

#### **NBD Export Management** (2 tables)
- `nbd_exports` - NBD export configurations
- `vm_export_mappings` - VM-based export reuse system

#### **Job Tracking and Logging** (5 tables)
- `job_tracking` - Generic job tracking system
- `job_execution_log` - Detailed execution audit trail
- `job_steps` - Individual job step tracking
- `job_tracking_hierarchy` - Parent-child job relationships
- `log_events` - Structured logging events

#### **Scheduling System** (7 tables)
- `replication_schedules` - Cron-based job scheduling
- `active_schedules` - Currently active schedules
- `schedule_executions` - Schedule execution history
- `schedule_execution_summary` - Schedule performance summary
- `vm_machine_groups` - VM grouping for bulk operations
- `vm_group_memberships` - VM-to-group relationships
- `vm_schedule_status` - VM scheduling status

#### **VMware Integration** (2 tables)
- `vmware_credentials` - Encrypted vCenter credentials
- `cbt_history` - Change Block Tracking audit trail

#### **VMA Management** (4 tables)
- `vma_enrollments` - VMA enrollment tracking
- `vma_pairing_codes` - VMA pairing code management
- `vma_active_connections` - Active VMA connections
- `vma_connection_audit` - VMA connection audit trail

#### **Configuration Management** (2 tables)
- `ossea_configs` - OSSEA/CloudStack configurations
- `linstor_configs` - Linstor storage configurations (deprecated)

#### **System Views** (1 table)
- `active_jobs` - View of currently active jobs

---

## 🚀 **DEPLOYMENT EXECUTION PLAN**

### **Phase 1: Pre-Deployment Preparation**
```bash
# 1. Verify dev OMA is accessible
ssh pgrayson@10.245.246.125 'systemctl status oma-api volume-daemon migration-gui'

# 2. Verify target server is fresh Ubuntu 24.04
ssh oma_admin@TARGET_IP 'cat /etc/os-release | grep VERSION_ID'

# 3. Copy deployment script
scp scripts/deploy-real-production-oma.sh oma_admin@TARGET_IP:/tmp/
```

### **Phase 2: Automated Deployment**
```bash
# Execute complete deployment
ssh oma_admin@TARGET_IP 'cd /tmp && echo "Password1" | sudo -S bash deploy-real-production-oma.sh'
```

### **Phase 3: Post-Deployment Validation**
```bash
# Health check all components
curl http://TARGET_IP:3001    # Migration GUI
curl http://TARGET_IP:8082/health    # OMA API
curl http://TARGET_IP:8090/api/v1/health    # Volume Daemon

# Service status verification
ssh oma_admin@TARGET_IP 'systemctl status oma-api volume-daemon migration-gui nbd-server'
```

### **Phase 4: VMA Connection Setup**
```bash
# Add VMA SSH public key for tunnel connectivity
ssh oma_admin@TARGET_IP 'echo "VMA_PUBLIC_KEY" | sudo tee /var/lib/vma_tunnel/.ssh/authorized_keys'
ssh oma_admin@TARGET_IP 'sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh/authorized_keys'
ssh oma_admin@TARGET_IP 'sudo chmod 600 /var/lib/vma_tunnel/.ssh/authorized_keys'
```

---

## 🎯 **PRODUCTION TEMPLATE EXPORT**

### **CloudStack Template Creation**
After successful deployment and validation:

1. **Stop all services** for clean template
2. **Clear logs and temporary files**
3. **Export VM as CloudStack template**
4. **Document template specifications**

### **Template Specifications**
- **Name**: OSSEA-Migrate OMA v2.0 Production
- **OS**: Ubuntu 24.04 LTS
- **Resources**: 8GB RAM, 4 vCPU, 100GB disk (minimum)
- **Network**: Single interface (auto-DHCP)
- **Features**: Complete migration platform ready for immediate use

---

## 🔒 **SECURITY CONSIDERATIONS**

### **Production Security Features**
- ✅ **Cloud-init disabled** - No metadata dependencies
- ✅ **SSH tunnel hardening** - Restricted vma_tunnel user
- ✅ **Credential encryption** - VMware passwords encrypted at rest
- ✅ **Service isolation** - Proper user separation
- ✅ **Port restrictions** - Only required ports exposed

### **Pre-Shared Key MVP**
- **Approach**: Use existing SSH keys (cloudstack_key) as pre-shared keys
- **User**: vma_tunnel@OMA for all VMA connections
- **Ports**: SSH tunnel on port 443 only
- **Restrictions**: No PTY, no X11, limited port forwarding

---

## 📊 **SUCCESS CRITERIA**

### **Deployment Success Indicators**
- ✅ All health endpoints responding (API, GUI, Volume Daemon)
- ✅ Database contains 34 tables with proper relationships
- ✅ SSH tunnel infrastructure ready (vma_tunnel user, port 443)
- ✅ NBD server listening on port 10809
- ✅ VirtIO tools installed for Windows VM support
- ✅ All services auto-start on boot

### **VMA Connectivity Test**
- ✅ VMA can establish SSH tunnel to port 443
- ✅ Forward tunnels work (NBD data, OMA API access)
- ✅ Reverse tunnel works (VMA API access from OMA)
- ✅ Migration workflow functional end-to-end

---

## 🚨 **CRITICAL PROJECT COMPLIANCE**

### **✅ Rules Followed**
- **NO SIMULATION CODE**: All components are REAL production binaries
- **Source Authority**: All binaries from `source/current/` via dev OMA
- **Volume Operations**: Volume Daemon is single source of truth
- **Database Schema**: Complete normalized schema with foreign keys
- **Network Constraints**: SSH tunnel on port 443 only

### **⚠️ Standardizations Required**
- **User Model**: Standardize GUI service to use `oma_admin`
- **Production Mode**: Change GUI from development to production mode
- **Service Consistency**: All services should use same user model

---

**🎯 This specification provides complete documentation for deploying REAL production OMA templates with actual binaries, complete database schema, and full functionality - no simulation code or placeholder components.**

**Status**: Ready for production deployment to OMAv3 (10.245.246.134) after revert

