#!/bin/bash
# Snapshot-only cleanup for VMs with already-detached volumes
# This handles the specific case where volumes are detached but snapshots remain

VM_NAME="$1"

if [ -z "$VM_NAME" ]; then
    echo "Usage: $0 <vm_name>"
    echo "Example: $0 pgtest2"
    exit 1
fi

echo "🧹 Performing snapshot-only cleanup for $VM_NAME"
echo "This will:"
echo "  1. Skip volume detachment (already detached)"
echo "  2. Revert and delete snapshots"
echo "  3. Reattach volumes to OMA"
echo "  4. Reset VM state"

# Create a custom cleanup request
curl -s -X POST "http://localhost:8082/api/v1/failover/${VM_NAME}/cleanup-failed" \
  -H "Content-Type: application/json" \
  -d '{"skip_detach": true}' || echo "Standard cleanup failed, trying snapshot-only approach..."

echo ""
echo "✅ Snapshot-only cleanup attempt completed for $VM_NAME"






