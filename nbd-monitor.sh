#!/bin/bash
# Real-time NBD monitoring script

while true; do
    clear
    echo "=== NBD SERVER STATUS ($(date)) ==="
    NBD_PID=$(pgrep -f "nbd-server -C /etc/nbd-server/config-base")
    if [ -n "$NBD_PID" ]; then
        echo "✅ NBD Server RUNNING (PID: $NBD_PID)"
        echo "🌐 Port Status:"
        sudo ss -tlnp | grep 10809 | head -1
    else
        echo "❌ NBD Server NOT RUNNING"
    fi
    
    echo -e "\n=== CURRENT EXPORTS ==="
    sudo cat /etc/nbd-server/config-base | grep -A1 "^\[" | grep -E "^\[|^exportname"
    
    echo -e "\n=== RECENT NBD HELPER LOG ==="
    sudo tail -3 /var/log/oma-nbd-helper.log 2>/dev/null
    
    echo -e "\n=== PRESS CTRL+C TO STOP ==="
    sleep 3
done
