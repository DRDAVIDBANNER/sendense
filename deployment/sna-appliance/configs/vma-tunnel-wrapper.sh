#!/bin/bash
set -euo pipefail

# VMA SSH Tunnel Wrapper Script - Pre-shared Key MVP
# Uses existing cloudstack_key as pre-shared key

source /opt/vma/vma-config.conf
SSH_KEY="/home/vma/.ssh/cloudstack_key"

echo "Starting VMA SSH tunnel to $OMA_HOST..."

exec /usr/bin/ssh -i "$SSH_KEY" -p 443 -N \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=3 \
    -o ExitOnForwardFailure=yes \
    -L 127.0.0.1:10808:127.0.0.1:10809 \
    -L 127.0.0.1:8082:127.0.0.1:8082 \
    -R 127.0.0.1:9081:127.0.0.1:8081 \
    vma_tunnel@$OMA_HOST
