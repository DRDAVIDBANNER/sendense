#!/bin/bash
# Automated Deployment Script for Sendense SSH Tunnel
# Deploys tunnel infrastructure to SNA (Sendense Node Appliance)
#
# Version: 1.0.0
# Date: 2025-10-07

set -e

# ========================================================================
# CONFIGURATION
# ========================================================================

SNA_HOST="${SNA_HOST:-10.0.100.231}"
SNA_USER="${SNA_USER:-pgrayson}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/cloudstack_key}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TUNNEL_SCRIPT="sendense-tunnel.sh"
SYSTEMD_SERVICE="sendense-tunnel.service"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ========================================================================
# HELPER FUNCTIONS
# ========================================================================

log_info() {
    echo -e "${BLUE}ℹ️  INFO:${NC} $*"
}

log_success() {
    echo -e "${GREEN}✅ SUCCESS:${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}⚠️  WARNING:${NC} $*"
}

log_error() {
    echo -e "${RED}❌ ERROR:${NC} $*"
}

log_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$*${NC}"
    echo -e "${BLUE}========================================${NC}"
}

check_file() {
    local file=$1
    if [ ! -f "$file" ]; then
        log_error "Required file not found: $file"
        return 1
    fi
    log_success "Found: $file"
    return 0
}

check_ssh_connectivity() {
    log_info "Testing SSH connectivity to $SNA_USER@$SNA_HOST..."
    if ssh -i "$SSH_KEY" -o ConnectTimeout=5 -o BatchMode=yes "$SNA_USER@$SNA_HOST" "echo 'SSH OK'" >/dev/null 2>&1; then
        log_success "SSH connectivity verified"
        return 0
    else
        log_error "Cannot connect to SNA via SSH"
        log_info "Attempted: ssh -i $SSH_KEY $SNA_USER@$SNA_HOST"
        return 1
    fi
}

# ========================================================================
# PRE-DEPLOYMENT CHECKS
# ========================================================================

log_header "Pre-Deployment Checks"

# Check SSH key exists
log_info "Checking SSH key..."
if [ ! -f "$SSH_KEY" ]; then
    log_error "SSH key not found: $SSH_KEY"
    log_info "Please ensure the SSH key exists and is accessible"
    exit 1
fi
log_success "SSH key found: $SSH_KEY"

# Check required files
log_info "Checking deployment files..."
check_file "$SCRIPT_DIR/$TUNNEL_SCRIPT" || exit 1
check_file "$SCRIPT_DIR/$SYSTEMD_SERVICE" || exit 1

# Check SSH connectivity
check_ssh_connectivity || exit 1

log_success "All pre-deployment checks passed!"

# ========================================================================
# DEPLOYMENT
# ========================================================================

log_header "Deploying to SNA ($SNA_HOST)"

# Copy files to SNA /tmp
log_info "Copying files to SNA..."
scp -i "$SSH_KEY" \
    "$SCRIPT_DIR/$TUNNEL_SCRIPT" \
    "$SCRIPT_DIR/$SYSTEMD_SERVICE" \
    "$SNA_USER@$SNA_HOST:/tmp/" || {
    log_error "Failed to copy files to SNA"
    exit 1
}
log_success "Files copied to SNA:/tmp/"

# Install and configure on SNA
log_info "Installing tunnel infrastructure on SNA..."
ssh -i "$SSH_KEY" "$SNA_USER@$SNA_HOST" bash <<'ENDSSH'
    set -e
    
    echo "Installing tunnel script..."
    sudo mv /tmp/sendense-tunnel.sh /usr/local/bin/
    sudo chmod +x /usr/local/bin/sendense-tunnel.sh
    sudo chown root:root /usr/local/bin/sendense-tunnel.sh
    echo "✅ Tunnel script installed to /usr/local/bin/sendense-tunnel.sh"
    
    echo "Installing systemd service..."
    sudo mv /tmp/sendense-tunnel.service /etc/systemd/system/
    sudo chmod 644 /etc/systemd/system/sendense-tunnel.service
    sudo chown root:root /etc/systemd/system/sendense-tunnel.service
    echo "✅ Systemd service installed to /etc/systemd/system/sendense-tunnel.service"
    
    echo "Reloading systemd daemon..."
    sudo systemctl daemon-reload
    echo "✅ Systemd daemon reloaded"
    
    echo "Enabling sendense-tunnel service..."
    sudo systemctl enable sendense-tunnel
    echo "✅ Service enabled (will start on boot)"
    
    echo "Starting sendense-tunnel service..."
    sudo systemctl start sendense-tunnel
    echo "✅ Service started"
    
    sleep 2
    
    echo "Checking service status..."
    if sudo systemctl is-active --quiet sendense-tunnel; then
        echo "✅ Service is active and running"
    else
        echo "❌ Service failed to start!"
        echo "Status:"
        sudo systemctl status sendense-tunnel --no-pager
        exit 1
    fi
ENDSSH

if [ $? -eq 0 ]; then
    log_success "Tunnel infrastructure deployed successfully!"
else
    log_error "Deployment failed!"
    exit 1
fi

# ========================================================================
# POST-DEPLOYMENT VERIFICATION
# ========================================================================

log_header "Post-Deployment Verification"

# Check service status
log_info "Verifying service status..."
ssh -i "$SSH_KEY" "$SNA_USER@$SNA_HOST" "sudo systemctl status sendense-tunnel --no-pager" || true

# Show recent logs
log_info "Recent logs (last 10 lines)..."
ssh -i "$SSH_KEY" "$SNA_USER@$SNA_HOST" "sudo journalctl -u sendense-tunnel -n 10 --no-pager" || true

# Test port forwarding
log_info "Testing port forwarding..."
sleep 3
ssh -i "$SSH_KEY" "$SNA_USER@$SNA_HOST" bash <<'ENDSSH'
    echo "Testing NBD port 10105..."
    if nc -zv -w2 localhost 10105 2>&1 | grep -q succeeded; then
        echo "✅ Port 10105 accessible"
    else
        echo "⚠️  Port 10105 not yet accessible (may take a few seconds)"
    fi
ENDSSH

# ========================================================================
# DEPLOYMENT SUMMARY
# ========================================================================

log_header "Deployment Summary"

echo ""
log_success "Sendense SSH Tunnel deployed successfully!"
echo ""
echo "Configuration:"
echo "  - SNA Host: $SNA_HOST"
echo "  - Tunnel Script: /usr/local/bin/sendense-tunnel.sh"
echo "  - Systemd Service: /etc/systemd/system/sendense-tunnel.service"
echo "  - Log File: /var/log/sendense-tunnel.log"
echo ""
echo "Service Management:"
echo "  - Status:  ssh -i $SSH_KEY $SNA_USER@$SNA_HOST 'sudo systemctl status sendense-tunnel'"
echo "  - Logs:    ssh -i $SSH_KEY $SNA_USER@$SNA_HOST 'sudo journalctl -u sendense-tunnel -f'"
echo "  - Restart: ssh -i $SSH_KEY $SNA_USER@$SNA_HOST 'sudo systemctl restart sendense-tunnel'"
echo ""
log_info "Tunnel forwards:"
echo "  - NBD Ports: 10100-10200 (101 ports)"
echo "  - SHA API: 8082"
echo "  - Reverse tunnel: 9081 (SHA can access SNA VMA API)"
echo ""
log_success "Deployment complete! 🚀"
echo ""
