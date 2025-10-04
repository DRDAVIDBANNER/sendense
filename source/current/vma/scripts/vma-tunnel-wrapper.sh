#!/bin/bash
set -euo pipefail

# VMA SSH Tunnel Wrapper Script
# Simple wrapper for systemd service

OMA_IP="${1:-10.245.246.125}"

echo "Starting VMA SSH tunnel to $OMA_IP..."

exec /usr/bin/ssh -i /opt/vma/enrollment/vma_enrollment_key -p 443 -N \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=3 \
    -o ExitOnForwardFailure=yes \
    -L 127.0.0.1:10808:127.0.0.1:10809 \
    -R 127.0.0.1:9081:127.0.0.1:8081 \
    vma_tunnel@$OMA_IP
