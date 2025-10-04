#!/bin/bash
# Enhanced SSH tunnel with keep-alive and monitoring for VMA->OMA connection
# Production version with proper user permissions and enrollment support
# Preserves all tunnel recovery functionality from original system

set -euo pipefail

# Production configuration paths
VMA_CONFIG_DIR="${VMA_CONFIG_DIR:-/opt/vma/config}"
VMA_SSH_DIR="${VMA_SSH_DIR:-/opt/vma/ssh}"
VMA_LOG_DIR="${VMA_LOG_DIR:-/opt/vma/logs}"

# Configuration (environment variables override defaults)
OMA_HOST="${OMA_HOST:-10.245.246.125}"
SSH_KEY="${SSH_KEY:-$VMA_SSH_DIR/oma-server-key}"
VMA_API_PORT=8081
OMA_API_PORT=8082
OMA_REVERSE_PORT=9081
NBD_PORT=10809
NBD_PORT_ALT=10808
LOG_FILE="$VMA_LOG_DIR/vma-tunnel-enhanced.log"

# SSH keep-alive and connection settings (PRESERVED FROM ORIGINAL)
SERVER_ALIVE_INTERVAL=30        # Send keep-alive every 30 seconds
SERVER_ALIVE_COUNT_MAX=3        # Try 3 times before considering connection dead
CONNECT_TIMEOUT=30              # Connection timeout in seconds
HEALTH_CHECK_INTERVAL=60        # Health check every 60 seconds

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') [$$] $*" | tee -a "$LOG_FILE"
}

# Health check function - tests tunnel connectivity (PRESERVED)
health_check() {
    local test_url="http://localhost:$OMA_API_PORT/health"
    
    if curl --connect-timeout 5 --max-time 10 -s "$test_url" >/dev/null 2>&1; then
        return 0  # Tunnel is healthy
    else
        return 1  # Tunnel is broken
    fi
}

# Clean up function - kills existing SSH processes and cleans up stale connections (PRESERVED)
cleanup_tunnel() {
    log "🧹 Cleaning up existing tunnel processes..."
    
    # Kill any existing SSH tunnel processes
    pkill -f "ssh.*$OMA_HOST" || true
    
    # Wait for processes to exit
    sleep 2
    
    # Force kill if still running
    pkill -9 -f "ssh.*$OMA_HOST" || true
    
    # Clean up any lingering port bindings
    local pids=$(lsof -ti :$OMA_REVERSE_PORT 2>/dev/null || true)
    if [ -n "$pids" ]; then
        log "🧹 Killing processes using port $OMA_REVERSE_PORT: $pids"
        kill -9 $pids || true
    fi
    
    log "✅ Cleanup completed"
}

# Check for enrollment-generated SSH key
check_enrollment_key() {
    # Check for enrollment-specific SSH key for this OMA
    local enrollment_key="$VMA_SSH_DIR/enrollment-${OMA_HOST}.key"
    
    if [ -f "$enrollment_key" ]; then
        log "🔐 Using enrollment-generated SSH key: $enrollment_key"
        SSH_KEY="$enrollment_key"
        return 0
    fi
    
    # Fallback to default SSH key
    if [ -f "$SSH_KEY" ]; then
        log "🔑 Using default SSH key: $SSH_KEY"
        return 0
    fi
    
    log "❌ No SSH key found for OMA connection"
    return 1
}

# Establish SSH tunnel with enhanced settings (PRESERVED + ENROLLMENT SUPPORT)
establish_tunnel() {
    log "🔧 Establishing enhanced SSH tunnel to $OMA_HOST..."
    
    # Check SSH key availability
    if ! check_enrollment_key; then
        log "❌ SSH key not available - cannot establish tunnel"
        return 1
    fi
    
    # Determine SSH user based on key type
    local ssh_user="oma"
    if [[ "$SSH_KEY" == *"enrollment"* ]]; then
        ssh_user="vma_tunnel"  # Use enrollment user for enrolled connections
        log "🔐 Using enrollment authentication: $ssh_user@$OMA_HOST"
    else
        log "🔑 Using manual authentication: $ssh_user@$OMA_HOST"
    fi
    
    # Build SSH command with robust settings (PRESERVED)
    ssh -i "$SSH_KEY" \
        -R ${OMA_REVERSE_PORT}:localhost:${VMA_API_PORT} \
        -L ${OMA_API_PORT}:localhost:${OMA_API_PORT} \
        -L ${NBD_PORT}:localhost:${NBD_PORT} \
        -L ${NBD_PORT_ALT}:localhost:10809 \
        -N \
        -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -o ServerAliveInterval=$SERVER_ALIVE_INTERVAL \
        -o ServerAliveCountMax=$SERVER_ALIVE_COUNT_MAX \
        -o ConnectTimeout=$CONNECT_TIMEOUT \
        -o TCPKeepAlive=yes \
        -o ExitOnForwardFailure=yes \
        -o BatchMode=yes \
        "${ssh_user}@$OMA_HOST" &
    
    local ssh_pid=$!
    log "🚀 SSH tunnel started with PID: $ssh_pid"
    
    # Wait for tunnel to establish
    sleep 5
    
    # Verify SSH process is still running
    if ! kill -0 $ssh_pid 2>/dev/null; then
        log "❌ SSH tunnel process died immediately"
        return 1
    fi
    
    # Test tunnel connectivity
    local retries=0
    while [ $retries -lt 10 ]; do
        if health_check; then
            log "✅ SSH tunnel established and verified"
            return 0
        fi
        
        retries=$((retries + 1))
        log "⏳ Waiting for tunnel to be ready... (attempt $retries/10)"
        sleep 3
    done
    
    log "❌ SSH tunnel failed to become ready after 30 seconds"
    return 1
}

# Main tunnel management loop (PRESERVED)
main() {
    # Ensure required directories exist
    mkdir -p "$VMA_LOG_DIR"
    
    log "🎯 Starting Enhanced SSH Tunnel Manager (Production Version)"
    log "   Target: $OMA_HOST"
    log "   SSH Key: $SSH_KEY"
    log "   Forward: localhost:$OMA_API_PORT -> OMA:$OMA_API_PORT"
    log "   Reverse: OMA:$OMA_REVERSE_PORT -> localhost:$VMA_API_PORT"
    log "   Health Check Interval: ${HEALTH_CHECK_INTERVAL}s"
    log "   User: $(whoami)"
    log "   Working Dir: $(pwd)"
    
    # Initial cleanup
    cleanup_tunnel
    
    # Main recovery loop (PRESERVED)
    while true; do
        # Establish tunnel
        if establish_tunnel; then
            log "🔄 Tunnel established, starting health monitoring..."
            
            # Monitor tunnel health (PRESERVED)
            while true; do
                sleep $HEALTH_CHECK_INTERVAL
                
                if health_check; then
                    log "💚 Tunnel health check passed"
                else
                    log "💔 Tunnel health check failed - tunnel needs restart"
                    break
                fi
            done
        else
            log "❌ Failed to establish tunnel"
        fi
        
        # Clean up before retry
        cleanup_tunnel
        
        log "⏳ Waiting 10 seconds before retry..."
        sleep 10
    done
}

# Handle signals gracefully (PRESERVED)
trap 'log "🛑 Received termination signal"; cleanup_tunnel; exit 0' TERM INT

# Start main loop
main






