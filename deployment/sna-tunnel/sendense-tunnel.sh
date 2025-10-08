#!/bin/bash
# Sendense SSH Tunnel Manager
# Establishes persistent SSH tunnel with full NBD port range
# 
# Purpose: Connect SNA (Sendense Node Appliance) to SHA (Sendense Hub Appliance)
# - Forwards NBD ports (10100-10200) for backup data transfer
# - Forwards SHA API port (8082) for control plane
# - Reverse tunnel (9081) for SNA API access from SHA
#
# Version: 1.0.0
# Date: 2025-10-07

set -e

# ========================================================================
# CONFIGURATION
# ========================================================================

SHA_HOST="${SHA_HOST:-10.245.246.134}"
SHA_PORT="${SHA_PORT:-443}"
SSH_KEY="${SSH_KEY:-/home/vma/.ssh/cloudstack_key}"
TUNNEL_USER="vma_tunnel"

# Port ranges
NBD_PORT_START=10100
NBD_PORT_END=10200
SNA_API_PORT=8081  # SNA VMA API (runs on SNA)
SHA_API_PORT=8082  # SHA API (runs on SHA)

# Logging
LOG_FILE="/var/log/sendense-tunnel.log"
MAX_LOG_SIZE=10485760  # 10MB

# ========================================================================
# LOGGING
# ========================================================================

log() {
    local timestamp=$(date +'%Y-%m-%d %H:%M:%S')
    local message="[$timestamp] $*"
    echo "$message" | tee -a "$LOG_FILE"
    
    # Rotate log if too large
    if [ -f "$LOG_FILE" ] && [ $(stat -f%z "$LOG_FILE" 2>/dev/null || stat -c%s "$LOG_FILE") -gt $MAX_LOG_SIZE ]; then
        mv "$LOG_FILE" "$LOG_FILE.old"
        log "Log rotated (size exceeded ${MAX_LOG_SIZE} bytes)"
    fi
}

log_error() {
    log "❌ ERROR: $*"
}

log_info() {
    log "ℹ️  INFO: $*"
}

log_success() {
    log "✅ SUCCESS: $*"
}

# ========================================================================
# PRE-FLIGHT CHECKS
# ========================================================================

preflight_checks() {
    log_info "Running pre-flight checks..."
    
    # Check SSH key exists
    if [ ! -f "$SSH_KEY" ]; then
        log_error "SSH key not found: $SSH_KEY"
        return 1
    fi
    
    # Check SSH key permissions
    local key_perms=$(stat -c %a "$SSH_KEY" 2>/dev/null || stat -f %A "$SSH_KEY")
    if [ "$key_perms" != "600" ] && [ "$key_perms" != "400" ]; then
        log_error "SSH key has incorrect permissions: $key_perms (should be 600 or 400)"
        return 1
    fi
    
    # Check if SHA host is reachable
    if ! ping -c 1 -W 2 "$SHA_HOST" >/dev/null 2>&1; then
        log_error "SHA host unreachable: $SHA_HOST"
        return 1
    fi
    
    log_success "Pre-flight checks passed"
    return 0
}

# ========================================================================
# PORT FORWARDING CONFIGURATION
# ========================================================================

build_port_forwards() {
    local forwards=""
    
    # Forward NBD port range (SNA localhost → SHA)
    # These ports carry backup data from qemu-nbd on SHA to SBC on SNA
    log_info "Building NBD port forwards ($NBD_PORT_START-$NBD_PORT_END)..."
    for port in $(seq $NBD_PORT_START $NBD_PORT_END); do
        forwards="$forwards -L $port:localhost:$port"
    done
    
    # Forward SHA API (SNA localhost → SHA)
    # This allows SNA to access SHA control plane
    forwards="$forwards -L $SHA_API_PORT:localhost:$SHA_API_PORT"
    log_info "Added SHA API forward: $SHA_API_PORT → SHA:$SHA_API_PORT"
    
    # Reverse tunnel for SNA API (SHA localhost → SNA)
    # This allows SHA to trigger backups on SNA
    forwards="$forwards -R 9081:localhost:$SNA_API_PORT"
    log_info "Added reverse tunnel: SHA:9081 → SNA:$SNA_API_PORT"
    
    echo "$forwards"
}

# ========================================================================
# TUNNEL MANAGEMENT
# ========================================================================

start_tunnel() {
    log_info "Building SSH tunnel configuration..."
    local port_forwards=$(build_port_forwards)
    
    log_info "Establishing SSH tunnel to $SHA_HOST:$SHA_PORT"
    log_info "Configuration:"
    log_info "  - NBD Ports: $NBD_PORT_START-$NBD_PORT_END (101 ports)"
    log_info "  - SHA API: $SHA_API_PORT"
    log_info "  - Reverse Tunnel: 9081 → $SNA_API_PORT"
    log_info "  - SSH Key: $SSH_KEY"
    log_info "  - Tunnel User: $TUNNEL_USER"
    
    # Start SSH tunnel with comprehensive options
    ssh -i "$SSH_KEY" \
        -p "$SHA_PORT" \
        -N \
        -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -o ServerAliveInterval=30 \
        -o ServerAliveCountMax=3 \
        -o ExitOnForwardFailure=yes \
        -o TCPKeepAlive=yes \
        -o Compression=yes \
        -o ConnectionAttempts=3 \
        -o ConnectTimeout=10 \
        $port_forwards \
        "$TUNNEL_USER@$SHA_HOST"
}

# ========================================================================
# SIGNAL HANDLING
# ========================================================================

cleanup() {
    log_info "Received shutdown signal, cleaning up..."
    exit 0
}

trap cleanup SIGTERM SIGINT

# ========================================================================
# MAIN LOOP
# ========================================================================

log_success "Sendense SSH Tunnel Manager starting..."
log_info "Version: 1.0.0"
log_info "PID: $$"

# Run pre-flight checks
if ! preflight_checks; then
    log_error "Pre-flight checks failed, exiting"
    exit 1
fi

# Main reconnection loop
RETRY_COUNT=0
MAX_RETRIES=3

while true; do
    log_info "Attempt $((RETRY_COUNT + 1)): Starting tunnel..."
    
    start_tunnel
    EXIT_CODE=$?
    
    if [ $EXIT_CODE -eq 0 ]; then
        log_info "Tunnel disconnected gracefully (exit code: 0)"
        RETRY_COUNT=0
    else
        log_error "Tunnel failed with exit code: $EXIT_CODE"
        RETRY_COUNT=$((RETRY_COUNT + 1))
        
        if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
            log_error "Max retries ($MAX_RETRIES) reached, performing extended backoff"
            RETRY_COUNT=0
            SLEEP_TIME=60
        else
            SLEEP_TIME=5
        fi
    fi
    
    log_info "Reconnecting in ${SLEEP_TIME} seconds..."
    sleep $SLEEP_TIME
done
