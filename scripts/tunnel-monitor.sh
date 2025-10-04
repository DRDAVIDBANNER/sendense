#!/bin/bash
# SSH Tunnel Health Monitor
# Provides simple health monitoring and manual recovery commands

set -euo pipefail

OMA_HOST="${OMA_HOST:-10.245.246.125}"
VMA_HOST="${VMA_HOST:-10.0.100.231}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/cloudstack_key}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') $*"
}

# Test tunnel health from OMA perspective
test_from_oma() {
    echo -e "${BLUE}🔍 Testing tunnel from OMA perspective...${NC}"
    
    # Test forward tunnel (VMA API via reverse port)
    if curl --connect-timeout 5 --max-time 10 -s "http://localhost:9081/api/v1/health" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Forward tunnel (OMA->VMA): HEALTHY${NC}"
        return 0
    else
        echo -e "${RED}❌ Forward tunnel (OMA->VMA): FAILED${NC}"
        return 1
    fi
}

# Test tunnel health from VMA perspective  
test_from_vma() {
    echo -e "${BLUE}🔍 Testing tunnel from VMA perspective...${NC}"
    
    # Test reverse tunnel (OMA API via local port)
    if ssh -i "$SSH_KEY" "pgrayson@$VMA_HOST" "curl --connect-timeout 5 --max-time 10 -s http://localhost:8082/health" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Reverse tunnel (VMA->OMA): HEALTHY${NC}"
        return 0
    else
        echo -e "${RED}❌ Reverse tunnel (VMA->OMA): FAILED${NC}"
        return 1
    fi
}

# Show service status
show_status() {
    echo -e "${BLUE}📊 SSH Tunnel Service Status${NC}"
    echo "----------------------------------------"
    
    # VMA service status
    echo -e "${YELLOW}VMA Service Status:${NC}"
    ssh -i "$SSH_KEY" "pgrayson@$VMA_HOST" "sudo systemctl status vma-tunnel-enhanced-v2 --no-pager -l" 2>/dev/null || {
        echo -e "${RED}❌ Failed to get VMA service status${NC}"
    }
    
    echo ""
    echo -e "${YELLOW}Active SSH Processes on VMA:${NC}"
    ssh -i "$SSH_KEY" "pgrayson@$VMA_HOST" "ps aux | grep ssh | grep $OMA_HOST | grep -v grep" 2>/dev/null || {
        echo -e "${RED}❌ No SSH tunnel processes found${NC}"
    }
    
    echo ""
    echo -e "${YELLOW}Port Bindings on OMA:${NC}"
    ss -tlnp | grep -E "(9081|8082)" || echo "No tunnel ports bound"
}

# Show recent logs
show_logs() {
    echo -e "${BLUE}📋 Recent SSH Tunnel Logs${NC}"
    echo "----------------------------------------"
    
    ssh -i "$SSH_KEY" "pgrayson@$VMA_HOST" "sudo journalctl -u vma-tunnel-enhanced-v2 --no-pager -l --since='10 minutes ago'" 2>/dev/null || {
        echo -e "${RED}❌ Failed to get VMA service logs${NC}"
    }
}

# Restart tunnel service
restart_tunnel() {
    echo -e "${YELLOW}🔄 Restarting SSH tunnel service...${NC}"
    
    ssh -i "$SSH_KEY" "pgrayson@$VMA_HOST" "sudo systemctl restart vma-tunnel-enhanced-v2" || {
        echo -e "${RED}❌ Failed to restart tunnel service${NC}"
        return 1
    }
    
    echo "⏳ Waiting 15 seconds for tunnel to establish..."
    sleep 15
    
    if test_from_oma && test_from_vma; then
        echo -e "${GREEN}✅ Tunnel restart successful${NC}"
    else
        echo -e "${RED}❌ Tunnel restart failed - check logs${NC}"
        return 1
    fi
}

# Main health check
health_check() {
    echo -e "${BLUE}🏥 SSH Tunnel Health Check${NC}"
    echo "========================================"
    
    local oma_ok=false
    local vma_ok=false
    
    if test_from_oma; then
        oma_ok=true
    fi
    
    if test_from_vma; then
        vma_ok=true
    fi
    
    echo ""
    if $oma_ok && $vma_ok; then
        echo -e "${GREEN}🎉 Overall Tunnel Status: HEALTHY${NC}"
        return 0
    else
        echo -e "${RED}💔 Overall Tunnel Status: DEGRADED${NC}"
        return 1
    fi
}

# Usage information
usage() {
    echo "SSH Tunnel Monitor - Health checking and management"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  health    - Run full health check (default)"
    echo "  status    - Show service and process status"
    echo "  logs      - Show recent tunnel logs"
    echo "  restart   - Restart tunnel service"
    echo "  monitor   - Continuous monitoring (every 30 seconds)"
    echo ""
    echo "Examples:"
    echo "  $0                 # Quick health check"
    echo "  $0 status          # Show detailed status"
    echo "  $0 restart         # Restart tunnel"
    echo "  $0 monitor         # Continuous monitoring"
}

# Continuous monitoring mode
monitor_mode() {
    echo -e "${BLUE}🔄 Starting continuous tunnel monitoring...${NC}"
    echo "Press Ctrl+C to stop"
    echo ""
    
    while true; do
        if health_check; then
            log "✅ Tunnel healthy"
        else
            log "❌ Tunnel unhealthy"
        fi
        
        echo ""
        sleep 30
    done
}

# Main script logic
main() {
    local command="${1:-health}"
    
    case "$command" in
        "health")
            health_check
            ;;
        "status")
            show_status
            ;;
        "logs")
            show_logs
            ;;
        "restart")
            restart_tunnel
            ;;
        "monitor")
            monitor_mode
            ;;
        "-h"|"--help"|"help")
            usage
            ;;
        *)
            echo -e "${RED}❌ Unknown command: $command${NC}"
            echo ""
            usage
            exit 1
            ;;
    esac
}

# Run main with all arguments
main "$@"
