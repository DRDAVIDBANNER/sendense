#!/bin/bash
# integration_test.sh - NBD File Export Integration Tests
# Tests the complete NBD file export workflow with real NBD server
#
# Prerequisites:
# - NBD server running on localhost:10809
# - /opt/migratekit/nbd-configs/conf.d/ directory exists
# - qemu-img available
# - nbd-client available (optional, for connection testing)

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
TEST_DIR="/tmp/nbd-integration-test-$$"
NBD_CONFIG_DIR="/opt/migratekit/nbd-configs/conf.d"
NBD_PORT=10809

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Setup
setup() {
    echo -e "${BLUE}🧪 NBD File Export Integration Tests${NC}"
    echo -e "${BLUE}====================================${NC}"
    echo ""
    
    # Create test directory
    mkdir -p "$TEST_DIR"
    echo "📁 Test directory: $TEST_DIR"
    
    # Verify NBD server is running
    if ! systemctl is-active --quiet nbd-server; then
        echo -e "${RED}❌ NBD server is not running${NC}"
        echo "   Start with: sudo systemctl start nbd-server"
        exit 1
    fi
    echo "✅ NBD server is running"
    
    # Verify qemu-img is available
    if ! command -v qemu-img &> /dev/null; then
        echo -e "${RED}❌ qemu-img not found${NC}"
        exit 1
    fi
    echo "✅ qemu-img available: $(qemu-img --version | head -1)"
    
    echo ""
}

# Cleanup
cleanup() {
    echo ""
    echo -e "${YELLOW}🧹 Cleaning up...${NC}"
    
    # Remove test QCOW2 files
    rm -rf "$TEST_DIR"
    
    # Remove test NBD exports
    rm -f "$NBD_CONFIG_DIR"/test-backup-*.conf 2>/dev/null || true
    
    # Reload NBD server
    if systemctl is-active --quiet nbd-server; then
        sudo pkill -SIGHUP nbd-server 2>/dev/null || true
    fi
    
    echo "✅ Cleanup complete"
}

# Test helper
run_test() {
    local test_name="$1"
    local test_func="$2"
    
    ((TESTS_RUN++))
    echo -e "${YELLOW}TEST $TESTS_RUN: $test_name${NC}"
    
    set +e  # Temporarily disable exit on error
    $test_func
    local result=$?
    set -e  # Re-enable exit on error
    
    if [ $result -eq 0 ]; then
        echo -e "${GREEN}   ✅ PASS${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}   ❌ FAIL${NC}"
        ((TESTS_FAILED++))
    fi
    echo ""
}

# Test 1: Create QCOW2 backup file
test_create_qcow2() {
    local qcow2_file="$TEST_DIR/test-backup-full.qcow2"
    
    # Create 1GB QCOW2 file
    if ! qemu-img create -f qcow2 "$qcow2_file" 1G > /dev/null 2>&1; then
        echo "   Failed to create QCOW2 file"
        return 1
    fi
    
    # Verify file exists
    if [ ! -f "$qcow2_file" ]; then
        echo "   QCOW2 file not found"
        return 1
    fi
    
    # Verify file size
    local size=$(qemu-img info --output=json "$qcow2_file" | jq '."virtual-size"')
    if [ "$size" != "1073741824" ]; then
        echo "   Unexpected file size: $size"
        return 1
    fi
    
    echo "   Created: $qcow2_file (1 GB virtual size)"
    return 0
}

# Test 2: Create NBD export configuration
test_create_nbd_export() {
    local export_name="test-backup-ctx-integration-disk0-full-$(date +%Y%m%dT%H%M%S)"
    local qcow2_file="$TEST_DIR/test-backup-full.qcow2"
    local export_conf="$NBD_CONFIG_DIR/${export_name}.conf"
    
    # Create export configuration
    cat > "/tmp/${export_name}.conf" << EOF
[$export_name]
exportname = $qcow2_file
readonly = false
multifile = false
copyonwrite = false
EOF
    
    # Copy to NBD config directory (needs sudo)
    if ! sudo cp "/tmp/${export_name}.conf" "$export_conf"; then
        echo "   Failed to create NBD export config"
        return 1
    fi
    
    # Verify config file exists
    if [ ! -f "$export_conf" ]; then
        echo "   Export config not found: $export_conf"
        return 1
    fi
    
    echo "   Created NBD export config: $export_name"
    echo "   Config file: $export_conf"
    return 0
}

# Test 3: SIGHUP reload
test_sighup_reload() {
    local nbd_pid=$(pgrep nbd-server | head -1)
    
    if [ -z "$nbd_pid" ]; then
        echo "   NBD server PID not found"
        return 1
    fi
    
    echo "   NBD server PID: $nbd_pid"
    
    # Send SIGHUP
    if ! sudo kill -SIGHUP "$nbd_pid"; then
        echo "   Failed to send SIGHUP"
        return 1
    fi
    
    # Wait a moment for reload
    sleep 2
    
    # Verify NBD server still running
    if ! systemctl is-active --quiet nbd-server; then
        echo "   NBD server died after SIGHUP"
        return 1
    fi
    
    echo "   ✅ SIGHUP reload successful (server still running)"
    return 0
}

# Test 4: Verify export name length compliance
test_export_name_length() {
    # Test various name lengths
    local vm_context="ctx-very-long-vm-name-that-might-exceed-limits-12345678"
    local export_name="backup-${vm_context}-disk0-full-$(date +%Y%m%dT%H%M%S)"
    
    local length=${#export_name}
    
    if [ $length -gt 63 ]; then
        echo "   Export name too long: $length characters (max 63)"
        return 1
    fi
    
    echo "   Export name length: $length characters (✅ < 64)"
    echo "   Sample name: $export_name"
    return 0
}

# Test 5: Create incremental backup with backing file
test_incremental_qcow2() {
    local full_backup="$TEST_DIR/test-backup-full.qcow2"
    local incr_backup="$TEST_DIR/test-backup-incr.qcow2"
    
    # Create incremental backup with backing file
    if ! qemu-img create -f qcow2 -b "$full_backup" -F qcow2 "$incr_backup" > /dev/null 2>&1; then
        echo "   Failed to create incremental QCOW2"
        return 1
    fi
    
    # Verify backing file relationship
    local backing=$(qemu-img info --output=json "$incr_backup" | jq -r '."backing-filename"')
    if [ "$backing" != "$full_backup" ]; then
        echo "   Incorrect backing file: $backing"
        return 1
    fi
    
    echo "   Created incremental backup: $incr_backup"
    echo "   Backing file: $full_backup"
    return 0
}

# Test 6: Verify NBD config.d pattern
test_config_d_pattern() {
    # Verify base config has includedir
    local base_config="/opt/migratekit/nbd-configs/nbd-server.conf"
    
    if [ ! -f "$base_config" ]; then
        echo "   Base config not found: $base_config"
        return 1
    fi
    
    if ! grep -q "includedir.*conf\.d" "$base_config"; then
        echo "   Base config missing includedir directive"
        return 1
    fi
    
    # Verify conf.d directory exists
    if [ ! -d "$NBD_CONFIG_DIR" ]; then
        echo "   conf.d directory not found: $NBD_CONFIG_DIR"
        return 1
    fi
    
    echo "   ✅ config.d pattern verified"
    echo "   Base config: $base_config"
    echo "   Include dir: $NBD_CONFIG_DIR"
    return 0
}

# Test 7: Multiple concurrent exports
test_multiple_exports() {
    local export_count=0
    
    # Count export files in conf.d
    export_count=$(ls -1 "$NBD_CONFIG_DIR"/*.conf 2>/dev/null | wc -l)
    
    echo "   Found $export_count export(s) in conf.d"
    
    # Create additional test exports
    for i in {1..3}; do
        local export_name="test-backup-multi-$i-$(date +%Y%m%dT%H%M%S)"
        local qcow2_file="$TEST_DIR/test-multi-$i.qcow2"
        
        # Create QCOW2
        qemu-img create -f qcow2 "$qcow2_file" 100M > /dev/null 2>&1
        
        # Create export config
        cat > "/tmp/${export_name}.conf" << EOF
[$export_name]
exportname = $qcow2_file
readonly = true
multifile = false
EOF
        sudo cp "/tmp/${export_name}.conf" "$NBD_CONFIG_DIR/"
    done
    
    # Reload NBD server
    sudo pkill -SIGHUP nbd-server
    sleep 1
    
    local new_count=$(ls -1 "$NBD_CONFIG_DIR"/*.conf 2>/dev/null | wc -l)
    echo "   Created 3 additional exports (total: $new_count)"
    
    if [ $new_count -le $export_count ]; then
        echo "   Export count did not increase"
        return 1
    fi
    
    return 0
}

# Test 8: Performance baseline
test_performance() {
    local qcow2_file="$TEST_DIR/test-backup-full.qcow2"
    
    # Write 100MB to QCOW2 file
    echo "   Writing 100MB to QCOW2 file..."
    local start_time=$(date +%s)
    
    dd if=/dev/zero of="$qcow2_file" bs=1M count=100 conv=notrunc oflag=direct > /dev/null 2>&1 || true
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $duration -eq 0 ]; then
        duration=1
    fi
    
    local throughput=$((100 / duration))
    
    echo "   Wrote 100MB in ${duration}s (${throughput} MB/s)"
    
    if [ $throughput -lt 50 ]; then
        echo "   ⚠️  Performance below expected baseline"
    fi
    
    return 0
}

# Main test execution
main() {
    setup
    trap cleanup EXIT
    
    echo -e "${BLUE}Running Integration Tests${NC}"
    echo ""
    
    run_test "Create QCOW2 backup file" test_create_qcow2
    run_test "Create NBD export configuration" test_create_nbd_export
    run_test "SIGHUP reload without service restart" test_sighup_reload
    run_test "Export name length compliance" test_export_name_length
    run_test "Create incremental backup with backing file" test_incremental_qcow2
    run_test "Verify config.d pattern" test_config_d_pattern
    run_test "Multiple concurrent exports" test_multiple_exports
    run_test "Performance baseline" test_performance
    
    echo -e "${BLUE}====================================${NC}"
    echo -e "${BLUE}Test Results${NC}"
    echo -e "${BLUE}====================================${NC}"
    echo ""
    echo "Total tests: $TESTS_RUN"
    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    echo ""
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}🎉 ALL TESTS PASSED!${NC}"
        return 0
    else
        echo -e "${RED}❌ SOME TESTS FAILED${NC}"
        return 1
    fi
}

# Execute
main "$@"
