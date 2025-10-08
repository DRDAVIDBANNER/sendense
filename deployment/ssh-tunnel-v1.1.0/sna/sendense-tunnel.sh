#!/bin/bash
# Sendense SSH Tunnel Manager for SNA (Node Appliance)
# Version: 1.1.0
# Date: 2025-10-07
# 
# Establishes persistent SSH tunnel from SNA to SHA with:
# - 101 NBD data ports (10100-10200) for concurrent backups
# - SHA API access (port 8082)
# - Simplified, reliable design (no reverse tunnel for now)
#
# DEPLOYMENT:
# - Copy to: /usr/local/bin/sendense-tunnel.sh
# - Permissions: chmod +x /usr/local/bin/sendense-tunnel.sh
# - Service: Install sendense-tunnel.service to /etc/systemd/system/

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================

SHA_HOST="${SHA_HOST:-10.245.246.134}"
SHA_PORT="${SHA_PORT:-443}"
SSH_KEY="/home/vma/.ssh/cloudstack_key"

# ============================================================================
# MAIN
# ============================================================================

echo "Starting Sendense SSH tunnel (forward ports only)..."

# Build NBD port forwards (10100-10200)
PORT_FORWARDS=""
for port in {10100..10200}; do
    PORT_FORWARDS="$PORT_FORWARDS -L $port:localhost:$port"
done

# Add SHA API forward
PORT_FORWARDS="$PORT_FORWARDS -L 8082:localhost:8082"

# NOTE: Reverse tunnel -R 9081:localhost:8081 disabled
# SHA can directly access SNA API on SNA:8081 if needed
# Will enable after resolving SSH PermitListen configuration

# Establish SSH tunnel with all port forwards
exec /usr/bin/ssh -i "$SSH_KEY" -p "$SHA_PORT" -N \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=3 \
    -o ExitOnForwardFailure=yes \
    $PORT_FORWARDS \
    vma_tunnel@$SHA_HOST
