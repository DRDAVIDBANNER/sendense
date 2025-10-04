#!/bin/bash
# build-sha-binaries.sh - Build Sendense Hub Appliance (SHA) binaries
# Version: v1.0.0
# Date: 2025-10-04

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SOURCE_DIR="$PROJECT_ROOT/source/current"
BUILD_OUTPUT_DIR="$PROJECT_ROOT/deployment/sha-appliance/binaries"

# Version info
BUILD_DATE=$(date +%Y%m%d)
BUILD_TIME=$(date +%H%M%S)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Binary versions (read from VERSION.txt files)
SHA_VERSION=$(cat "$SOURCE_DIR/oma/VERSION.txt" 2>/dev/null || echo "v3.0.1")
VOLUME_DAEMON_VERSION=$(cat "$SOURCE_DIR/volume-daemon/VERSION.txt" 2>/dev/null || echo "v1.2.1")

echo -e "${BLUE}🏗️  Building Sendense Hub Appliance (SHA) Binaries${NC}"
echo -e "${BLUE}=================================================${NC}"
echo ""
echo "📅 Build Date: $BUILD_DATE"
echo "🕐 Build Time: $BUILD_TIME"
echo "📝 Git Commit: $GIT_COMMIT"
echo "🎯 SHA API Version: $SHA_VERSION"
echo "⚙️  Volume Daemon Version: $VOLUME_DAEMON_VERSION"
echo ""

# Validation
validate_environment() {
    echo -e "${YELLOW}🔍 Validating build environment...${NC}"
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go is not installed${NC}"
        echo "   Install with: sudo snap install go --classic"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}')
    echo "   ✅ Go version: $GO_VERSION"
    
    # Check source directories exist
    if [[ ! -d "$SOURCE_DIR/oma" ]]; then
        echo -e "${RED}❌ SHA API source not found: $SOURCE_DIR/oma${NC}"
        exit 1
    fi
    
    if [[ ! -d "$SOURCE_DIR/volume-daemon" ]]; then
        echo -e "${RED}❌ Volume Daemon source not found: $SOURCE_DIR/volume-daemon${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}   ✅ Build environment validated${NC}"
    echo ""
}

# Build SHA API
build_sha_api() {
    echo -e "${YELLOW}🔨 Building SHA API...${NC}"
    
    cd "$SOURCE_DIR/oma"
    
    # Check for go.mod
    if [[ ! -f "go.mod" ]]; then
        echo -e "${RED}❌ go.mod not found in OMA directory${NC}"
        exit 1
    fi
    
    # Download dependencies
    echo "   📦 Downloading Go dependencies..."
    go mod download
    
    # Build binary
    BINARY_NAME="sendense-hub-${SHA_VERSION}-linux-amd64-${BUILD_DATE}-${GIT_COMMIT}"
    
    echo "   🔧 Compiling binary: $BINARY_NAME"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags="-X main.Version=${SHA_VERSION} -X main.BuildDate=${BUILD_DATE} -X main.GitCommit=${GIT_COMMIT} -w -s" \
        -o "$BINARY_NAME" \
        ./cmd/main.go
    
    # Verify binary was created
    if [[ ! -f "$BINARY_NAME" ]]; then
        echo -e "${RED}❌ Failed to create SHA API binary${NC}"
        exit 1
    fi
    
    # Calculate checksum
    SHA256_HASH=$(sha256sum "$BINARY_NAME" | awk '{print $1}')
    
    # Move to build output directory
    mkdir -p "$BUILD_OUTPUT_DIR"
    mv "$BINARY_NAME" "$BUILD_OUTPUT_DIR/"
    
    echo -e "${GREEN}   ✅ SHA API binary built successfully${NC}"
    echo "      Binary: $BINARY_NAME"
    echo "      Size: $(du -h "$BUILD_OUTPUT_DIR/$BINARY_NAME" | awk '{print $1}')"
    echo "      SHA256: $SHA256_HASH"
    echo ""
    
    # Create symlink for easy reference
    ln -sf "$BINARY_NAME" "$BUILD_OUTPUT_DIR/sendense-hub-latest"
}

# Build Volume Daemon
build_volume_daemon() {
    echo -e "${YELLOW}🔨 Building Volume Daemon...${NC}"
    
    cd "$SOURCE_DIR/volume-daemon"
    
    # Check for go.mod
    if [[ ! -f "go.mod" ]]; then
        echo -e "${RED}❌ go.mod not found in volume-daemon directory${NC}"
        exit 1
    fi
    
    # Download dependencies
    echo "   📦 Downloading Go dependencies..."
    go mod download
    
    # Build binary
    BINARY_NAME="volume-daemon-${VOLUME_DAEMON_VERSION}-linux-amd64-${BUILD_DATE}-${GIT_COMMIT}"
    
    echo "   🔧 Compiling binary: $BINARY_NAME"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags="-X main.Version=${VOLUME_DAEMON_VERSION} -X main.BuildDate=${BUILD_DATE} -X main.GitCommit=${GIT_COMMIT} -w -s" \
        -o "$BINARY_NAME" \
        ./cmd/*.go
    
    # Verify binary was created
    if [[ ! -f "$BINARY_NAME" ]]; then
        echo -e "${RED}❌ Failed to create Volume Daemon binary${NC}"
        exit 1
    fi
    
    # Calculate checksum
    SHA256_HASH=$(sha256sum "$BINARY_NAME" | awk '{print $1}')
    
    # Move to build output directory
    mv "$BINARY_NAME" "$BUILD_OUTPUT_DIR/"
    
    echo -e "${GREEN}   ✅ Volume Daemon binary built successfully${NC}"
    echo "      Binary: $BINARY_NAME"
    echo "      Size: $(du -h "$BUILD_OUTPUT_DIR/$BINARY_NAME" | awk '{print $1}')"
    echo "      SHA256: $SHA256_HASH"
    echo ""
    
    # Create symlink for easy reference
    ln -sf "$BINARY_NAME" "$BUILD_OUTPUT_DIR/volume-daemon-latest"
}

# Generate checksum file
generate_checksums() {
    echo -e "${YELLOW}🔐 Generating checksums...${NC}"
    
    cd "$BUILD_OUTPUT_DIR"
    
    # Generate SHA256 checksums for all binaries
    sha256sum sendense-hub-* volume-daemon-* | grep -v "latest" > CHECKSUMS.sha256
    
    echo -e "${GREEN}   ✅ Checksums generated${NC}"
    echo ""
}

# Create build manifest
create_manifest() {
    echo -e "${YELLOW}📝 Creating build manifest...${NC}"
    
    cat > "$BUILD_OUTPUT_DIR/BINARY_MANIFEST.md" << EOF
# Sendense Hub Appliance (SHA) Binary Manifest

**Build Date:** $BUILD_DATE $BUILD_TIME  
**Git Commit:** $GIT_COMMIT  
**Builder:** $(whoami)@$(hostname)  
**Go Version:** $(go version | awk '{print $3}')

## Binaries

### SHA API Server
- **Version:** $SHA_VERSION
- **Binary:** sendense-hub-${SHA_VERSION}-linux-amd64-${BUILD_DATE}-${GIT_COMMIT}
- **Source:** source/current/oma/
- **Description:** Sendense Hub Appliance API server

### Volume Daemon
- **Version:** $VOLUME_DAEMON_VERSION
- **Binary:** volume-daemon-${VOLUME_DAEMON_VERSION}-linux-amd64-${BUILD_DATE}-${GIT_COMMIT}
- **Source:** source/current/volume-daemon/
- **Description:** Volume management daemon for OSSEA operations

## Installation

\`\`\`bash
# Copy binaries to system
sudo cp sendense-hub-* /usr/local/bin/sendense-hub
sudo cp volume-daemon-* /usr/local/bin/volume-daemon
sudo chmod +x /usr/local/bin/sendense-hub /usr/local/bin/volume-daemon

# Verify installation
sendense-hub --version
volume-daemon --version
\`\`\`

## Checksums

See CHECKSUMS.sha256 for binary checksums.

\`\`\`bash
# Verify checksums
sha256sum -c CHECKSUMS.sha256
\`\`\`
EOF
    
    echo -e "${GREEN}   ✅ Build manifest created${NC}"
    echo ""
}

# Main build flow
main() {
    validate_environment
    build_sha_api
    build_volume_daemon
    generate_checksums
    create_manifest
    
    echo -e "${GREEN}🎉 Build completed successfully!${NC}"
    echo ""
    echo "📦 Binaries Location:"
    echo "   $BUILD_OUTPUT_DIR"
    echo ""
    echo "📄 Files Created:"
    ls -lh "$BUILD_OUTPUT_DIR" | grep -E "(sendense-hub|volume-daemon)" | grep -v "latest"
    echo ""
    echo "🚀 Next Steps:"
    echo "   1. Run deployment: ./deployment/sha-appliance/scripts/deploy-sha.sh"
    echo "   2. Or manually install: sudo cp $BUILD_OUTPUT_DIR/sendense-hub-latest /usr/local/bin/sendense-hub"
    echo ""
}

# Execute main build
main "$@"
