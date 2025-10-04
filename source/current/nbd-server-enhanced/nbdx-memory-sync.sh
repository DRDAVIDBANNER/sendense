#!/bin/bash
# NBDX Memory Synchronization Tool
# MigrateKit OSSEA Enhancement for NBD Server Memory Management
#
# PROBLEM: NBD server holds stale exports in memory after volume operations
# SOLUTION: Smart SIGHUP trigger that synchronizes memory with database state

set -euo pipefail

echo "🔄 NBDX Memory Sync - Synchronizing NBD server memory with database state"

# Get current NBD server PID
NBD_PID=$(pgrep nbd-server || echo "")
if [[ -z "$NBD_PID" ]]; then
    echo "❌ NBD server not running"
    exit 1
fi

echo "🔍 Found NBD server PID: $NBD_PID"

# Get current exports from NBD server memory
echo "📊 Current NBD server memory state:"
nbd-client -l localhost 10809 2>/dev/null | grep -v "Negotiation" || echo "No exports found"

# Count stale exports vs database exports
DB_EXPORTS=$(mysql -u oma_user -poma_password migratekit_oma -se "SELECT COUNT(*) FROM nbd_exports WHERE status = 'active';" 2>/dev/null)
MEMORY_EXPORTS=$(nbd-client -l localhost 10809 2>/dev/null | grep -c "migration-vol-" || echo "0")

echo "📋 Export count comparison:"
echo "   Database active exports: $DB_EXPORTS"
echo "   NBD memory exports: $MEMORY_EXPORTS"

if [[ "$MEMORY_EXPORTS" -gt "$DB_EXPORTS" ]]; then
    echo "⚠️  NBD server memory has stale exports - sending SIGHUP to sync"
    
    # Send SIGHUP to refresh NBD server memory
    sudo kill -HUP "$NBD_PID"
    
    # Wait for reload
    sleep 2
    
    # Check new state
    NEW_MEMORY_EXPORTS=$(nbd-client -l localhost 10809 2>/dev/null | grep -c "migration-vol-" || echo "0")
    echo "✅ After SIGHUP - NBD memory exports: $NEW_MEMORY_EXPORTS"
    
    if [[ "$NEW_MEMORY_EXPORTS" -eq "$DB_EXPORTS" ]]; then
        echo "🎉 NBD server memory synchronized successfully"
    else
        echo "⚠️  Memory still not synchronized - may need manual intervention"
    fi
else
    echo "✅ NBD server memory already synchronized"
fi

echo "🔄 NBDX Memory Sync completed"

