#!/bin/bash

# Stop Migration Script
# This script gracefully stops a running migration job and cleans up resources

set -e

# Configuration - Hardcoded for PGWINTESTBIOS
VM_NAME="PGWINTESTBIOS"
FORCE_STOP="${1:-false}"

echo "🛑 Stopping Migration Job"
echo "======================================"
echo "VM Name: $VM_NAME (hardcoded)"
echo "Force Stop: $FORCE_STOP"
echo "======================================"

# Files to check
MIGRATION_PID_FILE="/tmp/migratekit-migration-${VM_NAME}.pid"
MIGRATION_LOG_FILE="/tmp/migratekit-migration-${VM_NAME}.log"
CHANGEID_FILE="/tmp/migratekit_changeid_${VM_NAME}_disk_2000"

# Check if migration is running
if [[ ! -f "$MIGRATION_PID_FILE" ]]; then
    echo "⚠️  No migration PID file found for $VM_NAME"
    echo "   Expected: $MIGRATION_PID_FILE"
    echo "   Either migration is not running or already stopped"
    exit 1
fi

MIGRATION_PID=$(cat "$MIGRATION_PID_FILE")

echo "🔍 Found migration PID: $MIGRATION_PID"

# Check if process is still running
if ! kill -0 "$MIGRATION_PID" 2>/dev/null; then
    echo "⚠️  Migration process $MIGRATION_PID is not running"
    echo "🧹 Cleaning up PID file..."
    rm -f "$MIGRATION_PID_FILE"
    exit 0
fi

echo "✅ Migration process $MIGRATION_PID is running"

# Function to stop gracefully
stop_gracefully() {
    echo "🛑 Sending SIGTERM to migration process..."
    kill -TERM "$MIGRATION_PID" 2>/dev/null || true
    
    # Wait for graceful shutdown (up to 30 seconds)
    echo "⏳ Waiting for graceful shutdown..."
    for i in {1..30}; do
        if ! kill -0 "$MIGRATION_PID" 2>/dev/null; then
            echo "✅ Migration process stopped gracefully"
            return 0
        fi
        echo "   Waiting... ($i/30)"
        sleep 1
    done
    
    return 1
}

# Function to force stop
force_stop() {
    echo "💥 Force stopping migration process..."
    kill -KILL "$MIGRATION_PID" 2>/dev/null || true
    sleep 2
}

# Stop the migration process
if [[ "$FORCE_STOP" == "true" ]]; then
    force_stop
else
    if ! stop_gracefully; then
        echo "⚠️  Graceful shutdown timeout, force stopping..."
        force_stop
    fi
fi

# Verify process is stopped
if kill -0 "$MIGRATION_PID" 2>/dev/null; then
    echo "❌ Failed to stop migration process $MIGRATION_PID"
    exit 1
fi

echo "✅ Migration process stopped successfully"

# Clean up NBD processes
echo "🧹 Cleaning up NBD processes..."
pkill -f "nbdkit.*vddk" || echo "   No nbdkit processes found"
pkill -f "nbdcopy" || echo "   No nbdcopy processes found"

# Clean up any named pipes
echo "🧹 Cleaning up named pipes..."
rm -f /tmp/cloudstack_stream_* || true

# Show final log entries
echo ""
echo "📊 Final log entries:"
echo "======================================"
if [[ -f "$MIGRATION_LOG_FILE" ]]; then
    tail -10 "$MIGRATION_LOG_FILE"
else
    echo "   No log file found"
fi
echo "======================================"

# Clean up PID file
rm -f "$MIGRATION_PID_FILE"

# Check for ChangeID (this indicates successful completion)
if [[ -f "$CHANGEID_FILE" ]]; then
    echo "✅ ChangeID file found - migration appears to have completed"
    CHANGEID_CONTENT=$(cat "$CHANGEID_FILE" 2>/dev/null || echo "unable to read")
    echo "   ChangeID: $CHANGEID_CONTENT"
else
    echo "⚠️  No ChangeID file found - migration may have been interrupted"
    echo "   Expected: $CHANGEID_FILE"
fi

echo ""
echo "🎯 Migration stop completed!"
echo "   Process PID $MIGRATION_PID has been terminated"
echo "   NBD resources cleaned up"
echo "   Check CloudStack appliance for data integrity"

# Optional: Check CloudStack appliance disk usage
echo ""
echo "💾 CloudStack appliance disk status:"
echo "======================================"
ssh pgrayson@10.245.246.125 "df -h | grep -E '/dev/vd|Filesystem'" || echo "   Unable to check remote disk status"