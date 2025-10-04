# VMA-OMA Enrollment System - Complete Documentation

**System Version**: v2.0.0  
**Date**: September 29, 2025  
**Status**: ✅ **PRODUCTION READY**  
**Architecture**: Secure chicken-and-egg solution for NEW VMA enrollment

---

## 🎯 **SYSTEM OVERVIEW**

### **Purpose**
The VMA-OMA Enrollment System solves the **chicken-and-egg problem**: NEW VMAs need to connect to OMAs for operations, but establishing that connection traditionally required manual SSH key distribution and configuration.

### **Solution Architecture**
**Dual-Access System**:
- **NEW VMAs**: Enroll via internet-exposed port 443 **before** tunnel exists
- **EXISTING VMAs**: Continue using established tunnel for operations and re-enrollment

### **Business Value**
- ✅ **Self-Service Enrollment**: VMA operators can connect without OMA shell access
- ✅ **Enterprise Security**: Admin approval workflow with cryptographic verification
- ✅ **Scalability**: Supports multiple VMAs per OMA with unique credentials
- ✅ **Audit Compliance**: Complete security event logging and tracking

---

## 🏗️ **SYSTEM ARCHITECTURE**

### **Component Overview**
```
┌─────────────────────────────────────────────────────────────────────────┐
│                     VMA ENROLLMENT SYSTEM                              │
│                   CHICKEN & EGG SOLUTION                               │
│                                                                         │
│  NEW VMA (no tunnel)              OMA (Production)                     │
│  ┌─────────────────────┐          ┌─────────────────────────────────┐   │
│  │  VMA Enrollment     │          │                                 │   │
│  │  Script             │──────────►│  PORT 443 (Internet Exposed)   │   │
│  │  /opt/vma/          │   HTTPS  │  ┌─────────────────────────────┐ │   │
│  │  vma-enrollment.sh  │          │  │  enrollment-proxy service   │ │   │
│  └─────────────────────┘          │  │  - Security filtering       │ │   │
│                                   │  │  - Only VMA endpoints       │ │   │
│  EXISTING VMA (has tunnel)        │  │  - Proxies to :8082         │ │   │
│  ┌─────────────────────┐          │  └─────────┬───────────────────┘ │   │
│  │  VMA Operations     │          │            │                     │   │
│  │                     │──────────►│  ┌─────────▼───────────────────┐ │   │
│  │  Via SSH Tunnel     │  Tunnel  │  │  OMA API:8082               │ │   │
│  │  Port 443           │          │  │  - 10 VMA endpoints         │ │   │
│  └─────────────────────┘          │  │  - Admin approval           │ │   │
│                                   │  │  - Database integration     │ │   │
│                                   │  └─────────┬───────────────────┘ │   │
│                                   │            │                     │   │
│                                   │  ┌─────────▼───────────────────┐ │   │
│                                   │  │  Next.js GUI:3001           │ │   │
│                                   │  │  - Admin interface          │ │   │
│                                   │  │  - Approval workflow        │ │   │
│                                   │  └─────────┬───────────────────┘ │   │
│                                   │            │                     │   │
│                                   │  ┌─────────▼───────────────────┐ │   │
│                                   │  │  MariaDB:3306               │ │   │
│                                   │  │  - 4 enrollment tables      │ │   │
│                                   │  │  - Complete audit trail     │ │   │
│                                   │  └─────────────────────────────┘ │   │
│                                   └─────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

### **Port Configuration**
| Service | Port | Purpose | Access Level |
|---------|------|---------|-------------|
| **OMA API** | 8082 | Core enrollment endpoints | Internal only |
| **Next.js GUI** | 3001 | Admin approval interface | Network accessible |
| **Enrollment Proxy** | 443 | NEW VMA enrollment | Internet exposed |
| **SSH Tunnel** | 443 | Existing VMA operations | VMA outbound only |
| **MariaDB** | 3306 | Enrollment database | Internal only |

---

## 📡 **API ENDPOINTS**

### **Admin Management Endpoints** (Port 8082, Authenticated)
```bash
POST   /api/v1/admin/vma/pairing-code     # Generate secure pairing codes (10min expiry)
GET    /api/v1/admin/vma/pending          # List VMAs awaiting approval
POST   /api/v1/admin/vma/approve/{id}     # Approve VMA enrollment (installs SSH key)
GET    /api/v1/admin/vma/active           # List active VMA connections
POST   /api/v1/admin/vma/reject/{id}      # Reject VMA enrollment
DELETE /api/v1/admin/vma/revoke/{id}      # Revoke VMA access (removes SSH key)
GET    /api/v1/admin/vma/audit            # Security audit log with filtering
```

### **Public Enrollment Endpoints** (Port 443, Internet Exposed)
```bash
POST   /api/v1/vma/enroll                 # Initial VMA enrollment request
POST   /api/v1/vma/enroll/verify          # Challenge signature verification
GET    /api/v1/vma/enroll/result          # Poll for approval status
```

### **GUI Proxy Routes** (Port 3001, Network Accessible)
```typescript
# Next.js proxy routes forward to OMA API:8082
POST   /api/v1/admin/vma/pairing-code     # Admin pairing code generation
GET    /api/v1/admin/vma/pending          # Pending enrollments list
POST   /api/v1/admin/vma/approve/{id}     # VMA approval workflow
GET    /api/v1/admin/vma/active           # Active connections display
# ... (all admin endpoints proxied)
```

---

## 🗄️ **DATABASE SCHEMA**

### **Core Tables (4 Total)**

#### **vma_enrollments** - Core enrollment tracking
```sql
CREATE TABLE vma_enrollments (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    pairing_code VARCHAR(20) UNIQUE NOT NULL,           -- AX7K-PJ3F-TH2Q format
    vma_public_key TEXT NOT NULL,                       -- Ed25519 public key
    vma_name VARCHAR(255),                              -- Human-readable identifier
    vma_version VARCHAR(100),                           -- VMA software version
    vma_fingerprint VARCHAR(255),                       -- SSH key fingerprint
    vma_ip_address VARCHAR(45),                         -- Source IP address
    challenge_nonce VARCHAR(64),                        -- Cryptographic challenge
    status ENUM('pending_verification', 'awaiting_approval', 'approved', 'rejected', 'expired'),
    approved_by VARCHAR(255),                           -- Admin who approved
    approved_at TIMESTAMP NULL,
    expires_at TIMESTAMP NOT NULL,                      -- Pairing code expiry
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

#### **vma_active_connections** - Live connection tracking
```sql
CREATE TABLE vma_active_connections (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    enrollment_id VARCHAR(36) NOT NULL,                 -- FK to vma_enrollments
    vma_name VARCHAR(255) NOT NULL,
    vma_fingerprint VARCHAR(255) NOT NULL,
    ssh_user VARCHAR(50) DEFAULT 'vma_tunnel',
    connection_status ENUM('connected', 'disconnected', 'revoked'),
    last_seen_at TIMESTAMP NULL,                        -- Health monitoring
    connected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP NULL,
    revoked_by VARCHAR(255),
    FOREIGN KEY (enrollment_id) REFERENCES vma_enrollments(id) ON DELETE CASCADE
);
```

#### **vma_connection_audit** - Security audit trail
```sql
CREATE TABLE vma_connection_audit (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    enrollment_id VARCHAR(36),
    event_type ENUM('enrollment', 'verification', 'approval', 'rejection', 'connection', 'disconnection', 'revocation'),
    vma_fingerprint VARCHAR(255),
    source_ip VARCHAR(45),
    user_agent VARCHAR(255),
    approved_by VARCHAR(255),
    event_details JSON,                                 -- Additional metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (enrollment_id) REFERENCES vma_enrollments(id) ON DELETE CASCADE
);
```

#### **vma_pairing_codes** - Code generation tracking
```sql
CREATE TABLE vma_pairing_codes (
    id VARCHAR(36) PRIMARY KEY DEFAULT (UUID()),
    pairing_code VARCHAR(20) UNIQUE NOT NULL,
    generated_by VARCHAR(255) NOT NULL,
    used_by_enrollment_id VARCHAR(36) NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    FOREIGN KEY (used_by_enrollment_id) REFERENCES vma_enrollments(id) ON DELETE SET NULL
);
```

---

## 🔄 **ENROLLMENT WORKFLOWS**

### **Admin Workflow (OMA Side)**
```
1. Admin → GUI:3001/settings → VMA Enrollment tab
2. Click "Generate Pairing Code" → 10-minute expiry code created
3. Share pairing code securely with VMA operator
4. VMA enrolls → Appears in "Pending Enrollments" with details:
   - VMA name and version
   - SSH key fingerprint (for verification)
   - Source IP address
   - Enrollment timestamp
5. Admin reviews VMA details → Click "Approve" with optional notes
6. SSH key automatically installed (MVP: manual setup)
7. VMA appears in "Active VMA Connections" list
```

### **VMA Workflow (VMA Side)**
```
1. VMA Operator → VM Console → sudo /opt/vma/setup-wizard.sh
2. Select "Option 0: VMA Enrollment"
   OR directly: sudo /opt/vma/vma-enrollment.sh
3. Enter OMA IP address (e.g., 45.130.45.65)
4. Enter pairing code from admin (XXXX-XXXX-XXXX format)
5. VMA generates Ed25519 keypair automatically
6. VMA submits enrollment request via port 443
7. VMA completes cryptographic challenge/response
8. VMA polls for approval every 30 seconds (max 30 minutes)
9. When approved → VMA automatically configures tunnel
10. VMA ready for migration operations
```

### **Technical Workflow (System Level)**
```
VMA Enrollment Process:
├── Generate Ed25519 keypair (console-safe environment)
├── POST /api/v1/vma/enroll → receive cryptographic challenge
├── Sign challenge → POST /api/v1/vma/enroll/verify
├── Poll /api/v1/vma/enroll/result until status = "approved"
└── Configure SSH tunnel with enrollment credentials

OMA Processing:
├── Validate pairing code (unused + not expired)
├── Store enrollment with VMA metadata
├── Generate 32-byte cryptographic challenge
├── Verify signature → mark "awaiting_approval"
├── Admin approval → install SSH key (MVP: manual)
└── Return tunnel credentials and configuration
```

---

## 🔐 **SECURITY IMPLEMENTATION**

### **Cryptographic Security**
- **Ed25519 Keys**: Modern elliptic curve cryptography (256-bit)
- **Challenge/Response**: 32-byte random nonce prevents replay attacks
- **Pairing Codes**: 60-bit entropy, 10-minute expiry, single-use
- **SSH Restrictions**: Tunnel access only (ports 10809, 8081)

### **Network Security**
- **Port 443 Filtering**: Only VMA enrollment endpoints accessible
- **Rate Limiting**: Prevents brute force attacks on enrollment
- **Input Validation**: SQL injection and attack pattern detection
- **CORS Configuration**: Controlled cross-origin access

### **Operational Security**
- **Operator Approval**: Human verification required for all connections
- **Complete Audit Trail**: All enrollment events logged with metadata
- **Time-Limited Access**: Pairing codes expire automatically
- **SSH Key Management**: Atomic operations with backup/recovery

---

## 🚀 **DEPLOYMENT CONFIGURATION**

### **OMA Deployment Requirements**

#### **Database Migration**
```bash
# File: source/current/oma/database/migrations/20250928200000_vma_enrollment_system.up.sql
# Execution: Included in setup_mariadb_oma.sh
mysql -u oma_user -poma_password migratekit_oma < 20250928200000_vma_enrollment_system.up.sql
```

#### **SSH Key Generation** 
```bash
# File: scripts/setup_vma_enrollment.sh
# Creates: vma_tunnel user, SSH keys, tunnel wrapper, sudoers config
./setup_vma_enrollment.sh
# Result: Unique SSH key per OMA instance for VMA enrollment
```

#### **Enrollment Proxy Service**
```bash
# Binary: enrollment-proxy-v1.0.1
# Service: enrollment-proxy.service
# Port: 443 (internet-exposed)
# Configuration: Proxies only VMA enrollment endpoints to OMA API:8082
```

#### **OMA API Binary**
```bash
# Binary: oma-api-v2.39.0-gorm-field-fix (or latest)
# Features: 10 VMA enrollment endpoints, SSH automation (MVP manual)
# Requirements: Includes VMASSHManager service and enhanced approval workflow
```

### **VMA Deployment Requirements**

#### **VMA Enrollment Script**
```bash
# File: /opt/vma/vma-enrollment.sh
# Function: Complete VMA enrollment workflow
# Features: Console-safe key generation, approval polling, tunnel configuration
```

#### **VMA Setup Wizard Enhancement**
```bash
# File: /opt/vma/setup-wizard.sh
# Enhancement: Option 0 calls vma-enrollment.sh
# Integration: Clean separation, no complex code changes
```

#### **Dependencies**
```bash
# Required: haveged (entropy generation)
# Required: jq (JSON parsing)
# Required: curl (API communication)
# Required: ssh-keygen (key generation)
```

---

## 🔧 **OPERATIONAL PROCEDURES**

### **New OMA Deployment**
1. **Setup Database**: Run `setup_mariadb_oma.sh` (includes VMA tables)
2. **Generate SSH Keys**: Run `setup_vma_enrollment.sh` (unique per OMA)
3. **Deploy API**: Install OMA API with VMA enrollment endpoints
4. **Configure Proxy**: Deploy enrollment-proxy service on port 443
5. **Deploy GUI**: Install Next.js GUI with VMA enrollment interface
6. **Validate**: Test pairing code generation and enrollment workflow

### **New VMA Deployment**
1. **Install Scripts**: Deploy vma-enrollment.sh and enhanced setup-wizard.sh
2. **Install Dependencies**: haveged, jq, curl (for enrollment workflow)
3. **Configure Services**: Ensure VMA API and tunnel services ready
4. **Test Enrollment**: Validate VMA can connect to OMA via enrollment
5. **Validate Tunnel**: Confirm tunnel establishment after approval

### **Operational Management**
- **Pairing Codes**: Generate via GUI with 10-minute expiry
- **VMA Approval**: Review VMA details and SSH fingerprint before approval
- **Active Monitoring**: Monitor VMA connections via GUI active connections list
- **Access Revocation**: Remove VMA access via GUI with audit logging
- **Security Audit**: Review enrollment events via audit log interface

---

## 🧪 **TESTING PROCEDURES**

### **End-to-End Enrollment Test**
1. **Generate Code**: OMA admin creates pairing code via GUI
2. **VMA Enrollment**: VMA operator runs enrollment script with code
3. **Admin Approval**: OMA admin approves VMA via GUI interface
4. **Tunnel Validation**: Verify VMA tunnel establishes automatically
5. **Operations Test**: Confirm VMA can perform migration operations

### **Security Validation**
- **Invalid Codes**: Test expired/invalid pairing code rejection
- **Unauthorized Access**: Verify non-enrollment endpoints blocked on port 443
- **Approval Required**: Confirm VMAs cannot connect without admin approval
- **Audit Logging**: Validate all enrollment events properly logged

### **Integration Testing**
- **Multiple VMAs**: Test multiple concurrent enrollments
- **Existing VMA Operations**: Ensure existing tunnels unaffected
- **GUI Functionality**: Validate all admin interface operations
- **Database Integrity**: Confirm foreign key relationships and CASCADE DELETE

---

## 🛡️ **SECURITY CONSIDERATIONS**

### **Internet Exposure (Port 443)**
- **Filtered Access**: Only VMA enrollment endpoints accessible
- **Rate Limiting**: Prevents brute force attacks
- **Input Validation**: Comprehensive sanitization and attack detection
- **Audit Logging**: Complete security event trail

### **SSH Key Management**
- **Restricted Access**: SSH keys limited to tunnel ports only
- **Atomic Operations**: Safe authorized_keys file management
- **Backup/Recovery**: Automatic backup before key modifications
- **User Isolation**: Dedicated vma_tunnel user for security

### **Database Security**
- **Parameterized Queries**: SQL injection prevention
- **Foreign Key Constraints**: Data integrity enforcement
- **Audit Trail**: Complete enrollment event logging
- **Sensitive Data**: SSH keys and challenges properly stored

---

## 🔧 **TROUBLESHOOTING**

### **Common Issues**

#### **Enrollment Fails**
- **Check**: Pairing code validity (10-minute expiry)
- **Check**: Port 443 accessibility from VMA
- **Check**: enrollment-proxy service running on OMA
- **Debug**: VMA enrollment script debug output

#### **Approval Doesn't Work**
- **Check**: OMA API service running with VMA endpoints
- **Check**: GUI proxy routes configuration
- **Check**: Database connectivity and table existence
- **Debug**: OMA API logs for approval workflow

#### **SSH Key Issues**
- **Check**: vma_tunnel user exists with proper home directory
- **Check**: SSH directory permissions (700 for .ssh, 600 for authorized_keys)
- **Check**: Sudoers configuration for oma user
- **Manual**: Install SSH keys manually if automation fails

#### **VMA Key Generation Hanging**
- **Solution**: Console-safe environment (env -i PATH="$PATH" HOME="$HOME")
- **Solution**: Entropy service (haveged) for better randomness
- **Solution**: Timeout protection (15 seconds) with fallback
- **Debug**: strace ssh-keygen to identify hanging point

### **Service Dependencies**
```bash
# OMA Side
systemctl status oma-api.service          # Core API with VMA endpoints
systemctl status enrollment-proxy.service # Port 443 proxy
systemctl status migration-gui.service    # Admin interface
systemctl status mariadb.service         # Database

# VMA Side  
systemctl status vma-api.service         # VMA control API
systemctl status vma-tunnel-enhanced-v2.service # SSH tunnel (preserve!)
systemctl status haveged.service         # Entropy generation
```

---

## 📊 **MONITORING & METRICS**

### **Enrollment Metrics**
- **Active Enrollments**: Count of pending enrollments
- **Approval Rate**: Percentage of enrollments approved vs rejected
- **Connection Health**: Active VMA tunnel status monitoring
- **Security Events**: Failed enrollment attempts and patterns

### **Performance Metrics**
- **Enrollment Time**: Average time from submission to approval
- **Key Generation**: Time taken for VMA keypair generation
- **Tunnel Establishment**: Time from approval to active connection
- **API Response**: Enrollment endpoint response times

### **Health Monitoring**
```bash
# Enrollment system health check
curl -s http://localhost:8082/health
curl -s http://localhost:443/health
curl -s http://localhost:3001/api/health

# Database health
mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) FROM vma_enrollments;"

# Service health
systemctl is-active enrollment-proxy.service
systemctl is-active oma-api.service
```

---

## 🔗 **INTEGRATION POINTS**

### **Existing MigrateKit Systems**
- **Volume Daemon**: VMA enrollment preserved volume operation compliance
- **Failover System**: Enhanced failover system unaffected by enrollment
- **Progress Tracking**: VMA progress integration maintained
- **Database Schema**: VM-centric architecture preserved with enrollment addition

### **Network Architecture**
- **Single Port Rule**: All VMA-OMA traffic via port 443 (maintained)
- **TLS Tunnel**: Existing tunnel system preserved and enhanced
- **NBD Operations**: Migration operations unaffected by enrollment system

### **GUI Integration**
- **Settings Interface**: VMA enrollment as dedicated settings tab
- **Consistent Design**: Matches existing GUI design patterns
- **Error Handling**: Graceful degradation for missing enrollment features
- **Real-time Updates**: Automatic refresh of enrollment status

---

## 🎯 **SUCCESS METRICS**

### **Functional Requirements Met**
- ✅ **Self-Service Enrollment**: VMAs enroll without OMA shell access
- ✅ **Admin Approval**: Professional GUI approval workflow
- ✅ **Automatic Connection**: Approved VMAs get tunnel access (MVP: manual setup)
- ✅ **Security Compliance**: Complete audit trail and encryption
- ✅ **Multiple VMA Support**: Scalable to enterprise deployments

### **Technical Requirements Met**
- ✅ **Chicken-and-Egg Solution**: NEW VMAs enroll via port 443 without tunnel
- ✅ **Existing VMA Compatibility**: Current tunnel operations preserved
- ✅ **Database Integration**: Normalized schema with foreign key constraints
- ✅ **Professional Interface**: Enterprise-grade admin and operator workflows
- ✅ **Security Hardening**: Internet exposure with comprehensive protection

---

## 📋 **FUTURE ENHANCEMENTS**

### **Planned Improvements**
1. **External SSH Key Service**: Automated SSH key management (9-14 hours)
2. **Key Rotation System**: Automatic SSH key rotation with expiry
3. **TLS Certificate Integration**: mTLS for enrollment endpoints
4. **Advanced Monitoring**: Real-time connection health and alerting
5. **Bulk Operations**: Multi-VMA enrollment and management

### **Production Considerations**
- **Load Balancing**: Multiple OMA instances with enrollment coordination
- **Backup/Recovery**: Enrollment database backup and restore procedures
- **Security Hardening**: Additional attack prevention and monitoring
- **Documentation**: Operator training and troubleshooting guides

---

**🎯 The VMA-OMA Enrollment System provides enterprise-grade secure self-service VMA enrollment, solving the chicken-and-egg problem with professional workflows for both administrators and VMA operators.**





