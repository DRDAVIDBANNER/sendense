# Sendense Binary Management Rules

**Document Version:** 1.0  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **MANDATORY - NO BINARIES IN SOURCE**

---

## 🚨 BINARY MANAGEMENT RULES (ABSOLUTE)

### **Rule 1: NO BINARIES IN SOURCE CODE**
- ❌ **FORBIDDEN:** Any binary files in `source/current/`
- ❌ **FORBIDDEN:** Compiled executables in git repository
- ❌ **FORBIDDEN:** Libraries, dependencies, or archives in source
- ❌ **FORBIDDEN:** Test binaries, temporary builds, or debug files
- ✅ **REQUIRED:** All binaries in designated build directories only

### **Rule 2: EXPLICIT VERSION NUMBERS**
- ❌ **FORBIDDEN:** Ambiguous names (main, latest, final, new, old)
- ✅ **REQUIRED:** Semantic version numbers (v1.2.3)
- ✅ **REQUIRED:** Build date and commit hash in filename
- ✅ **REQUIRED:** Platform specification (linux, windows, arm64)

### **Rule 3: CENTRALIZED BUILD MANAGEMENT**
- ✅ **REQUIRED:** All builds in `source/builds/` directory
- ✅ **REQUIRED:** Build manifests with checksums
- ✅ **REQUIRED:** Build scripts and procedures documented
- ✅ **REQUIRED:** Clean build environment (no dependencies in source)

---

## 📁 BINARY DIRECTORY STRUCTURE

### **Required Directory Layout**

```
sendense/
├── source/builds/                    # ✅ ALL binaries go here
│   ├── control-plane/
│   │   ├── control-plane-v3.0.1-linux-amd64-20251004-abc123ef
│   │   ├── control-plane-v3.0.0-linux-amd64-20251001-def456ab
│   │   └── BUILD_MANIFEST.md        # Build history and checksums
│   ├── capture-agents/
│   │   ├── vmware/
│   │   │   ├── vmware-agent-v2.1.5-linux-amd64-20251004-ghi789cd
│   │   │   ├── vmware-agent-v2.1.4-linux-amd64-20251003-jkl012ef
│   │   │   └── BUILD_MANIFEST.md
│   │   ├── cloudstack/
│   │   │   ├── cloudstack-agent-v1.0.3-linux-amd64-20251004-mno345gh
│   │   │   └── BUILD_MANIFEST.md
│   │   └── hyperv/
│   │       ├── hyperv-agent-v1.0.1-windows-amd64-20251004-pqr678ij
│   │       └── BUILD_MANIFEST.md
│   ├── gui/
│   │   ├── sendense-cockpit-v1.2.0-20251004-stu901kl.tar.gz
│   │   └── BUILD_MANIFEST.md
│   └── deployment/
│       ├── sendense-full-v3.0.1-deployment-package.tar.gz
│       └── DEPLOYMENT_MANIFEST.md
├── source/current/                   # ✅ Source code only (NO binaries)
│   ├── control-plane/               # Go source code
│   ├── capture-agent/               # Go source code  
│   ├── gui/                         # React/Next.js source
│   └── build-scripts/               # Build automation scripts
└── dist/                            # ✅ Release distributions
    ├── v3.0.1/
    │   ├── sendense-v3.0.1-linux.tar.gz
    │   ├── sendense-v3.0.1-windows.zip
    │   ├── sendense-v3.0.1-checksums.sha256
    │   └── RELEASE_NOTES.md
    └── latest -> v3.0.1/             # Symlink to current release
```

---

## 🏗️ BUILD NAMING CONVENTIONS

### **Binary Naming Standard (MANDATORY)**

**Format:** `{component}-v{version}-{platform}-{arch}-{date}-{commit}`

**Examples:**
```bash
# Control Plane
control-plane-v3.0.1-linux-amd64-20251004-abc123ef

# Capture Agents
vmware-agent-v2.1.5-linux-amd64-20251004-def456ab
cloudstack-agent-v1.0.3-linux-amd64-20251004-ghi789cd
hyperv-agent-v1.0.1-windows-amd64-20251004-jkl012ef

# GUI Builds
sendense-cockpit-v1.2.0-20251004-mno345gh.tar.gz

# Deployment Packages
sendense-full-v3.0.1-deployment-package-20251004-pqr678ij.tar.gz
```

**Components:**
- `{component}`: control-plane, vmware-agent, cloudstack-agent, etc.
- `{version}`: Semantic version (v3.0.1)
- `{platform}`: linux, windows, darwin
- `{arch}`: amd64, arm64, 386
- `{date}`: YYYYMMDD build date
- `{commit}`: First 8 chars of git commit hash

---

## 📋 BUILD MANIFEST REQUIREMENTS

### **BUILD_MANIFEST.md Format**

**Required for Every Component:**
```markdown
# Control Plane Build Manifest

**Component:** Sendense Control Plane  
**Version:** v3.0.1  
**Build Date:** 2025-10-04 12:00:00 UTC  
**Git Commit:** abc123ef45678901  
**Builder:** GitHub Actions CI  

## Build Environment
- **Go Version:** 1.21.3
- **GOOS:** linux
- **GOARCH:** amd64  
- **CGO_ENABLED:** 0

## Build Configuration
```bash
go build -o control-plane-v3.0.1-linux-amd64-20251004-abc123ef \
  -ldflags "-X main.version=v3.0.1 -X main.commit=abc123ef -X main.buildTime=2025-10-04T12:00:00Z" \
  ./cmd/control-plane/
```

## Binary Details
- **File:** control-plane-v3.0.1-linux-amd64-20251004-abc123ef
- **Size:** 45.2 MB
- **SHA256:** a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456
- **Stripped:** Yes (debug symbols removed)
- **Compressed:** No (raw binary)

## Dependencies
- **External Dependencies:** None (static binary)
- **Runtime Dependencies:** systemd, OpenSSL, MariaDB client
- **Optional Dependencies:** qemu-img (for format conversion)

## Testing
- [ ] Unit tests: PASS (100% coverage)
- [ ] Integration tests: PASS 
- [ ] Security scan: PASS (no vulnerabilities)
- [ ] Performance test: 3.2 GiB/s throughput verified
- [ ] Load test: 50 concurrent operations verified

## Deployment Instructions
```bash
# Deploy to production
sudo systemctl stop sendense-control-plane
sudo cp control-plane-v3.0.1-linux-amd64-20251004-abc123ef /usr/local/bin/sendense-control-plane
sudo systemctl start sendense-control-plane

# Verify deployment
systemctl status sendense-control-plane
curl http://localhost:8082/health
```

## Rollback Instructions
```bash
# Rollback to previous version
sudo systemctl stop sendense-control-plane  
sudo cp control-plane-v3.0.0-linux-amd64-20251001-def456ab /usr/local/bin/sendense-control-plane
sudo systemctl start sendense-control-plane
```

## Known Issues
- None

## Changes from Previous Version
- Added backup repository support
- Improved error handling
- Performance optimization: 8% throughput improvement

---

**Build Approved By:** Engineering Lead  
**Deployment Approved By:** DevOps Lead  
**Release Date:** 2025-10-04
```

---

## 🔧 BUILD AUTOMATION

### **Required Build Scripts**

**Location:** `source/current/build-scripts/`

```bash
# build-control-plane.sh
#!/bin/bash
set -euo pipefail

VERSION="${1:-$(cat VERSION.txt)}"
COMMIT="$(git rev-parse --short HEAD)"
DATE="$(date +%Y%m%d)"
PLATFORM="linux"
ARCH="amd64"

BINARY_NAME="control-plane-v${VERSION}-${PLATFORM}-${ARCH}-${DATE}-${COMMIT}"
OUTPUT_DIR="../builds/control-plane"

echo "Building Control Plane v${VERSION}"
echo "Commit: ${COMMIT}"
echo "Output: ${OUTPUT_DIR}/${BINARY_NAME}"

# Clean build
go mod tidy
go mod verify

# Build with version information
go build -o "${OUTPUT_DIR}/${BINARY_NAME}" \
  -ldflags "-X main.version=v${VERSION} -X main.commit=${COMMIT} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -trimpath \
  ./cmd/control-plane/

# Generate checksum
cd "${OUTPUT_DIR}"
sha256sum "${BINARY_NAME}" > "${BINARY_NAME}.sha256"

# Update manifest
update-build-manifest.sh "control-plane" "${VERSION}" "${BINARY_NAME}"

echo "✅ Build completed: ${BINARY_NAME}"
echo "📊 Size: $(ls -lh ${BINARY_NAME} | awk '{print $5}')"
echo "🔒 SHA256: $(cat ${BINARY_NAME}.sha256)"
```

### **Version Management**

```bash
# bump-version.sh
#!/bin/bash
set -euo pipefail

CURRENT_VERSION="$(cat source/current/VERSION.txt)"
NEW_VERSION="${1}"

if [[ -z "${NEW_VERSION}" ]]; then
    echo "Usage: ./bump-version.sh v3.0.2"
    echo "Current version: ${CURRENT_VERSION}"
    exit 1
fi

echo "Bumping version: ${CURRENT_VERSION} → ${NEW_VERSION}"

# Update VERSION.txt
echo "${NEW_VERSION}" > source/current/VERSION.txt

# Update package.json (if GUI exists)
if [[ -f source/current/gui/package.json ]]; then
    sed -i "s/\"version\": \".*\"/\"version\": \"${NEW_VERSION#v}\"/" source/current/gui/package.json
fi

# Update CHANGELOG.md
sed -i "s/## \\[Unreleased\\]/## [Unreleased]\\n\\n## [${NEW_VERSION}] - $(date +%Y-%m-%d)/" CHANGELOG.md

# Commit version bump
git add source/current/VERSION.txt CHANGELOG.md
git commit -m "chore: bump version to ${NEW_VERSION}"
git tag -a "${NEW_VERSION}" -m "Release ${NEW_VERSION}"

echo "✅ Version bumped to ${NEW_VERSION}"
echo "🏷️ Git tag created: ${NEW_VERSION}"
```

---

## 📦 DEPLOYMENT PACKAGE MANAGEMENT

### **Deployment Package Structure**

```
sendense-full-v3.0.1-deployment-package/
├── DEPLOYMENT_GUIDE.md           # Installation instructions
├── SYSTEM_REQUIREMENTS.md        # Hardware/software requirements
├── binaries/
│   ├── control-plane-v3.0.1-linux-amd64-20251004-abc123ef
│   ├── vmware-agent-v2.1.5-linux-amd64-20251004-def456ab
│   ├── cloudstack-agent-v1.0.3-linux-amd64-20251004-ghi789cd
│   └── CHECKSUMS.sha256          # All binary checksums
├── configs/
│   ├── control-plane.service      # systemd service files
│   ├── capture-agent.service     
│   ├── config-templates/          # Configuration templates
│   └── ssl-certificates/          # SSL cert templates
├── database/
│   ├── schema.sql                 # Complete schema
│   ├── migrations/                # All migration files
│   └── seed-data.sql              # Initial data (if any)
├── scripts/
│   ├── install.sh                 # Automated installation
│   ├── upgrade.sh                 # Upgrade procedures
│   ├── backup.sh                  # Pre-upgrade backup
│   └── rollback.sh                # Rollback procedures
└── gui/
    ├── sendense-cockpit-v1.2.0.tar.gz
    ├── nginx.conf                 # Web server config
    └── ssl-setup.sh               # SSL configuration
```

### **Package Creation Script**

```bash
#!/bin/bash
# create-deployment-package.sh

VERSION="${1:-$(cat source/current/VERSION.txt)}"
DATE="$(date +%Y%m%d)"
PACKAGE_NAME="sendense-full-v${VERSION}-deployment-package-${DATE}"

echo "Creating deployment package: ${PACKAGE_NAME}"

# 1. Create package directory
mkdir -p "dist/${PACKAGE_NAME}"/{binaries,configs,database,scripts,gui}

# 2. Copy all binaries from builds/
cp source/builds/control-plane/control-plane-v${VERSION}-* "dist/${PACKAGE_NAME}/binaries/"
cp source/builds/capture-agents/**/agent-v*-* "dist/${PACKAGE_NAME}/binaries/" 

# 3. Generate checksums for all binaries
cd "dist/${PACKAGE_NAME}/binaries"
sha256sum * > CHECKSUMS.sha256
cd -

# 4. Copy configuration templates
cp -r deployment/configs/* "dist/${PACKAGE_NAME}/configs/"
cp -r deployment/database/* "dist/${PACKAGE_NAME}/database/"
cp -r deployment/scripts/* "dist/${PACKAGE_NAME}/scripts/"

# 5. Package GUI if built
if [[ -d "source/builds/gui/" ]]; then
    cp source/builds/gui/sendense-cockpit-v${VERSION}-*.tar.gz "dist/${PACKAGE_NAME}/gui/"
fi

# 6. Generate deployment manifest
generate-deployment-manifest.sh "${PACKAGE_NAME}" "${VERSION}"

# 7. Create deployment package
cd dist/
tar -czf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_NAME}/"
sha256sum "${PACKAGE_NAME}.tar.gz" > "${PACKAGE_NAME}.tar.gz.sha256"

echo "✅ Deployment package created: dist/${PACKAGE_NAME}.tar.gz"
echo "📊 Package size: $(ls -lh ${PACKAGE_NAME}.tar.gz | awk '{print $5}')"
echo "🔒 Package SHA256: $(cat ${PACKAGE_NAME}.tar.gz.sha256)"
```

---

## 🎯 BUILD PROCESS REQUIREMENTS

### **Automated Build Pipeline**

```yaml
# .github/workflows/build.yml
name: Sendense Build Pipeline

on:
  push:
    tags: ['v*']
  release:
    types: [published]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
          
      - name: Validate Source Structure
        run: |
          # Ensure no binaries in source/current/
          if find source/current/ -type f -executable -size +1M | grep -q .; then
            echo "ERROR: Binaries found in source code"
            exit 1
          fi
          
      - name: Build All Components
        run: |
          ./source/current/build-scripts/build-all.sh ${{ github.ref_name }}
          
      - name: Validate Build Quality
        run: |
          ./source/current/build-scripts/validate-builds.sh
          
      - name: Create Deployment Package
        run: |
          ./source/current/build-scripts/create-deployment-package.sh ${{ github.ref_name }}
          
      - name: Upload Build Artifacts
        uses: actions/upload-artifact@v3
        with:
          name: sendense-${{ github.ref_name }}
          path: dist/
```

### **Build Validation**

```bash
# validate-builds.sh
#!/bin/bash
set -euo pipefail

echo "🔍 Validating build quality..."

# 1. Check all binaries have version info
for binary in source/builds/**/*-v*; do
    if [[ -x "${binary}" ]]; then
        VERSION_OUTPUT="$(${binary} --version 2>/dev/null || echo 'NO_VERSION')"
        if [[ "${VERSION_OUTPUT}" == "NO_VERSION" ]]; then
            echo "❌ ERROR: Binary ${binary} missing version information"
            exit 1
        fi
        echo "✅ ${binary}: ${VERSION_OUTPUT}"
    fi
done

# 2. Verify checksums
for manifest in source/builds/**/BUILD_MANIFEST.md; do
    DIR="$(dirname ${manifest})"
    if [[ -f "${DIR}/CHECKSUMS.sha256" ]]; then
        cd "${DIR}"
        sha256sum -c CHECKSUMS.sha256 || exit 1
        cd -
    fi
done

# 3. Check for security vulnerabilities
if command -v govulncheck > /dev/null; then
    govulncheck ./source/current/...
fi

# 4. Validate no test dependencies in production builds
for binary in source/builds/**/*-v*; do
    if [[ -x "${binary}" ]]; then
        if ldd "${binary}" 2>/dev/null | grep -i test; then
            echo "❌ ERROR: Binary ${binary} contains test dependencies"
            exit 1
        fi
    fi
done

echo "✅ All build validations passed"
```

---

## 🔒 BINARY SECURITY

### **Security Requirements**

**Code Signing (Production):**
- ✅ **REQUIRED:** All production binaries must be code signed
- ✅ **REQUIRED:** Certificate validation in deployment scripts
- ✅ **REQUIRED:** Signature verification before execution

**Binary Hardening:**
```bash
# Security hardening during build
go build -o "${BINARY_NAME}" \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -trimpath \
  -buildmode=pie \
  ./cmd/control-plane/

# Strip debug symbols (-s -w)
# Remove build paths (-trimpath)  
# Position independent executable (-buildmode=pie)
```

**Supply Chain Security:**
```bash
# Verify dependency integrity
go mod verify
go mod download -json | jq -r .Sum | sort | uniq

# Check for known vulnerabilities  
govulncheck ./...

# Verify reproducible builds
go env GOPROXY GOSUMDB
```

---

## 🎯 DEPLOYMENT AUTOMATION

### **Clean Deployment Process**

```bash
# deploy-sendense.sh
#!/bin/bash
set -euo pipefail

VERSION="${1}"
DEPLOYMENT_PACKAGE="sendense-full-v${VERSION}-deployment-package"
BACKUP_DIR="/opt/sendense/backup/$(date +%Y%m%d-%H%M%S)"

# 1. Validate package integrity
echo "🔍 Validating deployment package..."
cd dist/
sha256sum -c "${DEPLOYMENT_PACKAGE}.tar.gz.sha256" || {
    echo "❌ Package integrity check failed"
    exit 1
}

# 2. Backup current installation
echo "💾 Backing up current installation..."
sudo mkdir -p "${BACKUP_DIR}"
sudo cp /usr/local/bin/sendense-* "${BACKUP_DIR}/" 2>/dev/null || true
sudo cp -r /etc/sendense/ "${BACKUP_DIR}/config/" 2>/dev/null || true

# 3. Extract and validate deployment package
tar -xzf "${DEPLOYMENT_PACKAGE}.tar.gz"
cd "${DEPLOYMENT_PACKAGE}"

# Verify all binaries present and valid
./scripts/verify-package.sh || {
    echo "❌ Package validation failed"
    exit 1
}

# 4. Stop services gracefully
echo "⏸️ Stopping Sendense services..."
sudo systemctl stop sendense-control-plane
sudo systemctl stop sendense-capture-agent

# 5. Deploy binaries
echo "📦 Deploying new binaries..."
sudo cp binaries/control-plane-v${VERSION}-* /usr/local/bin/sendense-control-plane
sudo cp binaries/*-agent-v*-* /usr/local/bin/

# 6. Update configuration (if needed)
if [[ -f "configs/migration.sql" ]]; then
    echo "🔄 Applying configuration updates..."
    mysql -u sendense -p < configs/migration.sql
fi

# 7. Start services
echo "🚀 Starting Sendense services..."
sudo systemctl start sendense-control-plane
sudo systemctl start sendense-capture-agent

# 8. Verify deployment
echo "✅ Verifying deployment..."
sleep 10
sudo systemctl is-active sendense-control-plane
curl -f http://localhost:8082/health

echo "✅ Deployment completed successfully"
echo "🔧 Backup location: ${BACKUP_DIR}"
echo "📋 Rollback: ./rollback.sh ${BACKUP_DIR}"
```

---

## 📊 BINARY LIFECYCLE MANAGEMENT

### **Retention Policy**

**Build Retention:**
- **Keep Last 5 Versions:** For each component
- **Keep All Major Versions:** For compatibility testing
- **Archive After 1 Year:** Move to long-term storage
- **Security Patches:** Keep indefinitely for audit trail

**Automated Cleanup:**
```bash
# cleanup-old-builds.sh (run monthly)
#!/bin/bash

for component_dir in source/builds/*/; do
    component="$(basename ${component_dir})"
    
    # Keep last 5 versions, remove older
    cd "${component_dir}"
    ls -t ${component}-v* | tail -n +6 | xargs rm -f
    
    # Update BUILD_MANIFEST.md
    echo "Cleaned up old builds for ${component}"
done
```

---

## 🎯 COMPLIANCE AND AUDITING

### **Build Audit Trail**

**Required Documentation:**
- Complete build environment specification
- All dependency versions and checksums
- Build configuration and flags used
- Security scanning results
- Performance testing results
- Deployment and rollback procedures

**Audit Questions (Must Be Answerable):**
- What exact commit was this binary built from?
- What dependencies were used and their versions?
- What build environment and configuration?
- What testing was performed before deployment?
- How can this build be reproduced exactly?
- What security scanning was performed?

### **Compliance Reporting**

**Monthly Binary Security Report:**
```markdown
# Binary Security Report - October 2025

## Build Summary
- Total Binaries: 47
- Security Scanned: 47 (100%)
- Vulnerabilities Found: 0 (Target: 0)
- Code Signed: 47 (100%)

## Dependency Analysis
- Go Dependencies: 127 (all verified)
- Known Vulnerabilities: 0
- Outdated Dependencies: 2 (non-critical)

## Compliance Status
- Build Reproducibility: 100% 
- Supply Chain Verification: Pass
- Binary Hardening: All binaries hardened
- Code Signing: All production binaries signed

## Action Items
- Update 2 non-critical dependencies
- Archive builds older than 6 months
- Renew code signing certificate (expires in 90 days)
```

---

**THIS ENSURES SENDENSE HAS ENTERPRISE-GRADE BUILD MANAGEMENT**

**NO SCATTERED BINARIES, NO MYSTERY BUILDS, NO SECURITY GAPS**

**PROFESSIONAL BUILD PROCESS THAT ENTERPRISES CAN TRUST**

---

**Document Owner:** DevOps Engineering Team  
**Enforcement:** Mandatory for all builds  
**Review Cycle:** Monthly audit, quarterly process review  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **MANDATORY COMPLIANCE**
