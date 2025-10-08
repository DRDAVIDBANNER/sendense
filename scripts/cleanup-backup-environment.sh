#!/bin/bash
# Sendense Backup Environment Cleanup Script
# Purpose: Clean all backup-related processes and files before testing
# Created: October 8, 2025
# Project: Phase 1 VMware Backup Completion

set -e  # Exit on error

echo "=========================================="
echo "🧹 Sendense Backup Environment Cleanup"
echo "=========================================="
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track cleanup success
CLEANUP_ERRORS=0

# ==========================================
# STEP 1: Kill all qemu-nbd processes
# ==========================================
echo "Step 1: Killing qemu-nbd processes..."
QEMU_COUNT=$(pgrep -f "qemu-nbd" | wc -l || true)
if [ "$QEMU_COUNT" -gt 0 ]; then
    echo "  Found $QEMU_COUNT qemu-nbd processes"
    sudo pkill -9 -f qemu-nbd || true
    sleep 1
    REMAINING=$(pgrep -f "qemu-nbd" | wc -l || true)
    if [ "$REMAINING" -eq 0 ]; then
        echo -e "  ${GREEN}✅ All qemu-nbd processes killed${NC}"
    else
        echo -e "  ${RED}❌ Warning: $REMAINING qemu-nbd processes still running${NC}"
        CLEANUP_ERRORS=$((CLEANUP_ERRORS + 1))
    fi
else
    echo -e "  ${GREEN}✅ No qemu-nbd processes running${NC}"
fi
echo ""

# ==========================================
# STEP 2: Delete all QCOW2 files
# ==========================================
echo "Step 2: Deleting QCOW2 files from /backup/repository/..."
if [ ! -d "/backup/repository" ]; then
    echo -e "  ${YELLOW}⚠️  Directory /backup/repository does not exist${NC}"
else
    QCOW2_COUNT=$(find /backup/repository -name "*.qcow2" 2>/dev/null | wc -l || true)
    if [ "$QCOW2_COUNT" -gt 0 ]; then
        echo "  Found $QCOW2_COUNT QCOW2 files"
        rm -f /backup/repository/*.qcow2 || true
        rm -f /backup/repository/**/*.qcow2 || true
        REMAINING=$(find /backup/repository -name "*.qcow2" 2>/dev/null | wc -l || true)
        if [ "$REMAINING" -eq 0 ]; then
            echo -e "  ${GREEN}✅ All QCOW2 files deleted${NC}"
        else
            echo -e "  ${RED}❌ Warning: $REMAINING QCOW2 files remain${NC}"
            CLEANUP_ERRORS=$((CLEANUP_ERRORS + 1))
        fi
    else
        echo -e "  ${GREEN}✅ No QCOW2 files found${NC}"
    fi
fi
echo ""

# ==========================================
# STEP 3: Kill sendense-backup-client on SNA
# ==========================================
echo "Step 3: Killing sendense-backup-client processes on SNA..."
SNA_HOST="10.0.100.231"
SNA_USER="vma"

# Check if we can reach SNA
if timeout 2 bash -c "echo > /dev/tcp/$SNA_HOST/22" 2>/dev/null; then
    echo "  SNA accessible at $SNA_HOST"
    # Kill processes (use password authentication)
    ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 "$SNA_USER@$SNA_HOST" \
        "pkill -f sendense-backup-client || true" 2>/dev/null || true
    echo -e "  ${GREEN}✅ Sent kill signal to SNA backup processes${NC}"
else
    echo -e "  ${YELLOW}⚠️  Cannot reach SNA at $SNA_HOST - skipping${NC}"
fi
echo ""

# ==========================================
# STEP 4: Check for file locks on QCOW2
# ==========================================
echo "Step 4: Checking for QCOW2 file locks..."
if command -v lsof &> /dev/null; then
    LOCKS=$(sudo lsof 2>/dev/null | grep -c "\.qcow2" || true)
    if [ "$LOCKS" -gt 0 ]; then
        echo -e "  ${RED}❌ WARNING: $LOCKS QCOW2 file locks still present${NC}"
        echo "  Locked files:"
        sudo lsof 2>/dev/null | grep "\.qcow2" || true
        CLEANUP_ERRORS=$((CLEANUP_ERRORS + 1))
    else
        echo -e "  ${GREEN}✅ No QCOW2 file locks detected${NC}"
    fi
else
    echo -e "  ${YELLOW}⚠️  lsof not available - cannot check locks${NC}"
fi
echo ""

# ==========================================
# STEP 5: Restart SHA to clear port allocations
# ==========================================
echo "Step 5: Restarting sendense-hub service..."
if systemctl is-active --quiet sendense-hub 2>/dev/null; then
    echo "  Stopping sendense-hub..."
    sudo systemctl stop sendense-hub
    sleep 2
    echo "  Starting sendense-hub..."
    sudo systemctl start sendense-hub
    sleep 2
    
    if systemctl is-active --quiet sendense-hub; then
        echo -e "  ${GREEN}✅ sendense-hub restarted successfully${NC}"
    else
        echo -e "  ${RED}❌ sendense-hub failed to restart${NC}"
        CLEANUP_ERRORS=$((CLEANUP_ERRORS + 1))
    fi
else
    echo -e "  ${YELLOW}⚠️  sendense-hub service not found or not active${NC}"
fi
echo ""

# ==========================================
# STEP 6: Verify environment clean
# ==========================================
echo "Step 6: Final verification..."
echo ""

echo "  Process verification:"
QEMU_FINAL=$(pgrep -f "qemu-nbd" | wc -l || true)
echo "    - qemu-nbd processes: $QEMU_FINAL"

echo "  File verification:"
if [ -d "/backup/repository" ]; then
    QCOW2_FINAL=$(find /backup/repository -name "*.qcow2" 2>/dev/null | wc -l || true)
    echo "    - QCOW2 files: $QCOW2_FINAL"
else
    echo "    - /backup/repository: not found"
fi

echo "  Service verification:"
if systemctl is-active --quiet sendense-hub 2>/dev/null; then
    echo -e "    - sendense-hub: ${GREEN}active${NC}"
else
    echo -e "    - sendense-hub: ${RED}inactive${NC}"
fi
echo ""

# ==========================================
# CLEANUP SUMMARY
# ==========================================
echo "=========================================="
if [ "$CLEANUP_ERRORS" -eq 0 ]; then
    echo -e "${GREEN}🎉 Environment cleanup completed successfully${NC}"
    echo -e "${GREEN}✅ Ready for backup testing${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Cleanup completed with $CLEANUP_ERRORS warnings${NC}"
    echo -e "${YELLOW}   Review warnings above before proceeding${NC}"
    exit 0  # Still exit 0 as partial cleanup is useful
fi
echo "=========================================="
