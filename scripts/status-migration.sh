#!/bin/bash

# Migration Status Script
# Shows the status of running migrations

VM_NAME="PGWINTESTBIOS"

echo "📊 Migration Status Check"
echo "======================================"
echo "VM Name: $VM_NAME"
echo "Time: $(date)"
echo "======================================"

# Files to check
MIGRATION_PID_FILE="/tmp/migratekit-migration-${VM_NAME}.pid"
MIGRATION_LOG_FILE="/tmp/migratekit-migration-${VM_NAME}.log"
CHANGEID_FILE="/tmp/migratekit_changeid_${VM_NAME}_disk_2000"

# Check if migration PID file exists
if [[ -f "$MIGRATION_PID_FILE" ]]; then
    MIGRATION_PID=$(cat "$MIGRATION_PID_FILE")
    echo "📁 PID File: Found ($MIGRATION_PID)"
    
    # Check if process is running
    if kill -0 "$MIGRATION_PID" 2>/dev/null; then
        echo "🟢 Process Status: RUNNING (PID: $MIGRATION_PID)"
        
        # Show process info
        echo "📋 Process Info:"
        ps -p "$MIGRATION_PID" -o pid,ppid,cmd,etime,pcpu,pmem 2>/dev/null || echo "   Unable to get process info"
        
    else
        echo "🔴 Process Status: NOT RUNNING (PID file exists but process dead)"
    fi
else
    echo "📁 PID File: Not found"
    echo "🔴 Process Status: NOT RUNNING"
fi

# Check for related processes
echo ""
echo "🔍 Related Processes:"
NBDKIT_PROCS=$(pgrep -f "nbdkit.*vddk" 2>/dev/null | wc -l)
NBDCOPY_PROCS=$(pgrep -f "nbdcopy" 2>/dev/null | wc -l)
MIGRATEKIT_PROCS=$(pgrep -f "migratekit" 2>/dev/null | wc -l)

echo "   nbdkit processes: $NBDKIT_PROCS"
echo "   nbdcopy processes: $NBDCOPY_PROCS"
echo "   migratekit processes: $MIGRATEKIT_PROCS"

if [[ $NBDKIT_PROCS -gt 0 ]]; then
    echo "   NBD Details:"
    ps aux | grep -E "nbdkit.*vddk" | grep -v grep | head -3
fi

# Check log file and show recent entries
echo ""
echo "📝 Log Status:"
if [[ -f "$MIGRATION_LOG_FILE" ]]; then
    LOG_SIZE=$(stat -c%s "$MIGRATION_LOG_FILE" 2>/dev/null || echo "0")
    LOG_LINES=$(wc -l < "$MIGRATION_LOG_FILE" 2>/dev/null || echo "0")
    echo "   File: $MIGRATION_LOG_FILE"
    echo "   Size: $((LOG_SIZE / 1024)) KB ($LOG_LINES lines)"
    echo "   Modified: $(stat -c %y "$MIGRATION_LOG_FILE" 2>/dev/null || echo "unknown")"
    
    echo ""
    echo "📊 Recent Log Entries (last 10):"
    echo "------------------------------------"
    tail -10 "$MIGRATION_LOG_FILE" 2>/dev/null || echo "   Unable to read log file"
    echo "------------------------------------"
else
    echo "   No log file found"
fi

# Check ChangeID status
echo ""
echo "🔄 CBT Status:"
if [[ -f "$CHANGEID_FILE" ]]; then
    CHANGEID_CONTENT=$(cat "$CHANGEID_FILE" 2>/dev/null || echo "unable to read")
    echo "   ChangeID File: Found"
    echo "   Content: $CHANGEID_CONTENT"
    echo "   Modified: $(stat -c %y "$CHANGEID_FILE" 2>/dev/null || echo "unknown")"
else
    echo "   ChangeID File: Not found"
    echo "   Status: No previous migration or in progress"
fi

# Check CloudStack appliance status
echo ""
echo "💾 CloudStack Appliance Status:"
if ssh -o ConnectTimeout=5 pgrayson@10.245.246.125 "echo 'Connected'" 2>/dev/null >/dev/null; then
    echo "   Connection: ✅ OK"
    
    # Check disk usage
    echo "   Disk Status:"
    ssh pgrayson@10.245.246.125 "df -h | grep -E '/dev/vd|Filesystem'" 2>/dev/null || echo "   Unable to check disk status"
    
    # Check for active dd processes (receiving data)
    DD_PROCS=$(ssh pgrayson@10.245.246.125 "pgrep -f 'dd.*of=/dev/vd' | wc -l" 2>/dev/null || echo "0")
    echo "   Active dd processes: $DD_PROCS"
    
else
    echo "   Connection: ❌ Failed"
fi

# Check named pipes
echo ""
echo "🔗 Named Pipes:"
PIPE_COUNT=$(ls /tmp/cloudstack_stream_* 2>/dev/null | wc -l)
if [[ $PIPE_COUNT -gt 0 ]]; then
    echo "   Found $PIPE_COUNT pipe(s):"
    ls -la /tmp/cloudstack_stream_* 2>/dev/null | head -3
else
    echo "   No named pipes found"
fi

# Summary
echo ""
echo "📋 SUMMARY"
echo "======================================"

if [[ -f "$MIGRATION_PID_FILE" ]]; then
    MIGRATION_PID=$(cat "$MIGRATION_PID_FILE")
    if kill -0 "$MIGRATION_PID" 2>/dev/null; then
        echo "🟢 Migration ACTIVE (PID: $MIGRATION_PID)"
        if [[ $NBDCOPY_PROCS -gt 0 ]]; then
            echo "📊 Data transfer in progress"
        else
            echo "⏳ Migration process running (may be in setup/cleanup phase)"
        fi
    else
        echo "🟡 Migration STOPPED (PID file exists but process dead)"
    fi
else
    if [[ $MIGRATEKIT_PROCS -gt 0 ]] || [[ $NBDKIT_PROCS -gt 0 ]]; then
        echo "🟡 Migration-related processes detected without PID file"
        echo "   May need manual cleanup"
    else
        echo "🔴 No migration detected"
    fi
fi

if [[ -f "$CHANGEID_FILE" ]]; then
    echo "✅ CBT tracking available for incremental sync"
fi

echo ""
echo "🎯 Control Commands:"
echo "   Monitor: tail -f $MIGRATION_LOG_FILE"
echo "   Stop: ./scripts/stop-migration.sh"
echo "   Force stop: ./scripts/stop-migration.sh true"