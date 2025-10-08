#!/bin/bash
# Deploy SNA API Server v1.12.0 with Change ID Fix
# Run this script manually to deploy to SNA at 10.0.100.231

set -e

echo "🚀 Deploying SNA API Server v1.12.0-changeid-fix"
echo ""

# Binary info
LOCAL_BINARY="/home/oma_admin/sendense/source/builds/sna-api-server-v1.12.0-changeid-fix"
SNA_HOST="10.0.100.231"
SNA_USER="vma"

# Check local binary exists
if [ ! -f "$LOCAL_BINARY" ]; then
    echo "❌ Binary not found: $LOCAL_BINARY"
    exit 1
fi

echo "✅ Binary found: $LOCAL_BINARY ($(du -h $LOCAL_BINARY | cut -f1))"
echo ""

# Step 1: Copy binary to SNA
echo "📦 Step 1: Copying binary to SNA..."
scp "$LOCAL_BINARY" ${SNA_USER}@${SNA_HOST}:/tmp/sna-api-server-new
if [ $? -ne 0 ]; then
    echo "❌ Failed to copy binary. Check SSH access."
    echo "   Try: ssh ${SNA_USER}@${SNA_HOST}"
    exit 1
fi
echo "✅ Binary copied to SNA:/tmp/sna-api-server-new"
echo ""

# Step 2: Deploy on SNA
echo "🔧 Step 2: Deploying on SNA..."
ssh ${SNA_USER}@${SNA_HOST} << 'ENDSSH'
    set -e
    
    echo "  🛑 Stopping SNA API server..."
    sudo systemctl stop sna-api-server 2>/dev/null || sudo pkill sna-api-server || true
    sleep 2
    
    echo "  💾 Backing up old binary..."
    if [ -f /usr/local/bin/sna-api-server ]; then
        sudo cp /usr/local/bin/sna-api-server /usr/local/bin/sna-api-server.backup
    fi
    
    echo "  📦 Installing new binary..."
    sudo mv /tmp/sna-api-server-new /usr/local/bin/sna-api-server
    sudo chmod +x /usr/local/bin/sna-api-server
    sudo chown root:root /usr/local/bin/sna-api-server
    
    echo "  ✅ Starting SNA API server..."
    sudo systemctl start sna-api-server 2>/dev/null || \
        nohup /usr/local/bin/sna-api-server --port 8081 --auto-cbt=true > /var/log/sna-api.log 2>&1 &
    
    sleep 2
    
    echo "  🔍 Verifying process..."
    if ps aux | grep -v grep | grep sna-api-server > /dev/null; then
        echo "  ✅ SNA API server is running"
        ps aux | grep -v grep | grep sna-api-server | awk '{print "     PID: "$2" | "$11" "$12" "$13}'
    else
        echo "  ❌ SNA API server is NOT running"
        exit 1
    fi
ENDSSH

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Deployment complete!"
    echo ""
    echo "📋 What Changed:"
    echo "   - Added MIGRATEKIT_JOB_ID environment variable to backup command"
    echo "   - Added MIGRATEKIT_PREVIOUS_CHANGE_ID for incremental backups"
    echo "   - sendense-backup-client will now record change_id in SHA database"
    echo ""
    echo "🧪 Next Steps:"
    echo "   1. Clean up old QCOW2 files: /home/oma_admin/sendense/scripts/cleanup-backup-environment.sh"
    echo "   2. Start full backup: curl -X POST http://localhost:8082/api/v1/backups ..."
    echo "   3. Check change_id recorded: SELECT change_id FROM backup_jobs ORDER BY created_at DESC LIMIT 1;"
else
    echo ""
    echo "❌ Deployment failed on SNA"
    echo "   Check SSH connection and try manual deployment"
    exit 1
fi

