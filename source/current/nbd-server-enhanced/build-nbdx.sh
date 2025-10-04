#!/bin/bash
# NBDX Build Script - Docker-based Enhanced NBD Server Build
# MigrateKit OSSEA Enhancement for NBD Memory Synchronization

set -euo pipefail

echo "🚀 Building NBDX (Enhanced NBD Server) using Docker..."

# Change to NBDX build directory
cd "$(dirname "$0")"

# Verify required files exist
if [[ ! -f "nbd-server-cache-flush.patch" ]]; then
    echo "❌ Error: nbd-server-cache-flush.patch not found"
    exit 1
fi

if [[ ! -f "nbd-server.c" ]]; then
    echo "❌ Error: nbd-server.c not found"
    exit 1
fi

echo "✅ Build files verified"

# Build NBDX container
echo "🔧 Building NBDX Docker container..."
docker build -t nbdx-builder:latest .

# Extract NBDX binary
echo "📦 Extracting NBDX binary..."
docker create --name nbdx-extract nbdx-builder:latest
docker cp nbdx-extract:/output/nbdx ./nbdx
docker cp nbdx-extract:/output/nbdx-version.txt ./nbdx-version.txt
docker rm nbdx-extract

# Verify binary
echo "✅ NBDX binary extracted:"
ls -la nbdx
cat nbdx-version.txt

echo "🎉 NBDX build complete!"
echo "📁 Binary location: $(pwd)/nbdx"
echo "🚀 Ready for deployment as enhanced NBD server"

# Instructions
echo ""
echo "📋 Next steps:"
echo "1. Test binary: ./nbdx --help"
echo "2. Backup current: sudo cp /usr/bin/nbd-server /usr/bin/nbd-server.backup"
echo "3. Deploy: sudo cp nbdx /usr/bin/nbd-server"
echo "4. Restart service: sudo systemctl restart nbd-server"
echo "5. Test SIGHUP: sudo kill -HUP \$(pgrep nbd-server)"

