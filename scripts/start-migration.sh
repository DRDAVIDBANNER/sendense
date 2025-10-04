#!/bin/bash

# Start Migration Script
# This script starts a migration job that can be controlled externally
# All credentials and settings are hardcoded for PGWINTESTBIOS

set -e

# Configuration - All hardcoded for PGWINTESTBIOS
VM_NAME="PGWINTESTBIOS"
VMWARE_ENDPOINT="192.168.17.159"
VMWARE_USERNAME="administrator@vsphere.local"
VMWARE_PASSWORD="EmyGVoBFesGQc47-"
VMWARE_PATH="/DatabanxDC/vm/PGWINTESTBIOS"

# CloudStack Configuration  
CLOUDSTACK_HOST="pgrayson@10.245.246.125"
CLOUDSTACK_DEVICE="/dev/vde"

echo "🚀 Starting Migration Job"
echo "======================================"
echo "VM Name: $VM_NAME (hardcoded)"
echo "VMware Endpoint: $VMWARE_ENDPOINT"
echo "VMware Path: $VMWARE_PATH"
echo "CloudStack Target: $CLOUDSTACK_HOST:$CLOUDSTACK_DEVICE"
echo "======================================"

# Set required environment variables
export CLOUDSTACK_API_URL="http://dummy.local"
export CLOUDSTACK_API_KEY="dummy-key"
export CLOUDSTACK_SECRET_KEY="dummy-secret"

# Create PID file for tracking
MIGRATION_PID_FILE="/tmp/migratekit-migration-${VM_NAME}.pid"
MIGRATION_LOG_FILE="/tmp/migratekit-migration-${VM_NAME}.log"

echo "📝 PID file: $MIGRATION_PID_FILE"
echo "📝 Log file: $MIGRATION_LOG_FILE"

# Function to cleanup on exit
cleanup() {
    echo "🧹 Cleaning up start script..."
    if [[ -f "$MIGRATION_PID_FILE" ]]; then
        rm -f "$MIGRATION_PID_FILE"
    fi
}
trap cleanup EXIT

# Start migration in background and capture PID
echo "🎯 Starting migratekit migration process..."

  # Use nohup to ensure process continues after script ends
  nohup ./migratekit-tls-tunnel migrate \
  --vmware-endpoint "$VMWARE_ENDPOINT" \
  --vmware-username "$VMWARE_USERNAME" \
  --vmware-password "$VMWARE_PASSWORD" \
  --vmware-path "$VMWARE_PATH" \
  > "$MIGRATION_LOG_FILE" 2>&1 &

MIGRATION_PID=$!

# Save PID for stop script
echo "$MIGRATION_PID" > "$MIGRATION_PID_FILE"

echo "✅ Migration started with PID: $MIGRATION_PID"
echo "📊 Monitor progress: tail -f $MIGRATION_LOG_FILE"
echo "🛑 Stop migration: ./scripts/stop-migration.sh $VM_NAME"

# Wait a moment to ensure process started successfully
sleep 3

if kill -0 "$MIGRATION_PID" 2>/dev/null; then
    echo "✅ Migration process is running successfully"
    echo "🔍 Check logs for snapshot creation and progress updates"
else
    echo "❌ Migration process failed to start"
    cat "$MIGRATION_LOG_FILE"
    exit 1
fi

echo ""
echo "🎯 Migration job started successfully!"
echo "   PID: $MIGRATION_PID"
echo "   Monitor: tail -f $MIGRATION_LOG_FILE"
echo "   Stop: ./scripts/stop-migration.sh"