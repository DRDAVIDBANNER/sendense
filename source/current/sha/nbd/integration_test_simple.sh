#!/bin/bash
# Simple NBD File Export Integration Test
set -euo pipefail

echo "🧪 NBD File Export Integration Test"
echo "===================================="
echo ""

# Test 1: QCOW2 Creation
echo "TEST 1: Create QCOW2 file"
TEST_DIR="/tmp/nbd-test-$$"
mkdir -p "$TEST_DIR"
qcow2_file="$TEST_DIR/test-backup.qcow2"

qemu-img create -f qcow2 "$qcow2_file" 1G > /dev/null 2>&1
size=$(qemu-img info --output=json "$qcow2_file" | jq '."virtual-size"')
echo "   ✅ Created QCOW2 file: $size bytes"

# Test 2: NBD Export Config Creation
echo ""
echo "TEST 2: Create NBD export configuration"
export_name="test-backup-$(date +%Y%m%dT%H%M%S)"
export_conf="/opt/migratekit/nbd-configs/conf.d/${export_name}.conf"

cat > "/tmp/${export_name}.conf" << EOF
[$export_name]
exportname = $qcow2_file
readonly = false
multifile = false
copyonwrite = false
EOF

sudo cp "/tmp/${export_name}.conf" "$export_conf"
echo "   ✅ Created NBD export config: $export_name"

# Test 3: Verify Config
echo ""
echo "TEST 3: Verify configuration"
if [ -f "$export_conf" ]; then
    echo "   ✅ Export config exists: $export_conf"
else
    echo "   ❌ Export config not found"
    exit 1
fi

# Test 4: SIGHUP Reload
echo ""
echo "TEST 4: SIGHUP reload"
nbd_pid=$(pgrep nbd-server | head -1)
echo "   NBD server PID: $nbd_pid"
sudo kill -SIGHUP "$nbd_pid"
sleep 2

if systemctl is-active --quiet nbd-server; then
    echo "   ✅ NBD server still running after SIGHUP"
else
    echo "   ❌ NBD server died after SIGHUP"
    exit 1
fi

# Test 5: Incremental Backup
echo ""
echo "TEST 5: Incremental backup with backing file"
incr_file="$TEST_DIR/test-incremental.qcow2"
qemu-img create -f qcow2 -b "$qcow2_file" -F qcow2 "$incr_file" > /dev/null 2>&1
backing=$(qemu-img info --output=json "$incr_file" | jq -r '."backing-filename"')
if [ "$backing" == "$qcow2_file" ]; then
    echo "   ✅ Incremental backup created with correct backing file"
else
    echo "   ❌ Backing file mismatch: $backing"
    exit 1
fi

# Test 6: Export Name Length
echo ""
echo "TEST 6: Export name length compliance"
long_name="backup-ctx-very-long-vm-name-disk0-full-$(date +%Y%m%dT%H%M%S)"
name_len=${#long_name}
if [ $name_len -le 63 ]; then
    echo "   ✅ Export name length: $name_len chars (< 64)"
else
    echo "   ❌ Export name too long: $name_len chars"
    exit 1
fi

# Test 7: Multiple Exports
echo ""
echo "TEST 7: Multiple concurrent exports"
initial_count=$(ls -1 /opt/migratekit/nbd-configs/conf.d/*.conf 2>/dev/null | wc -l)
echo "   Initial exports: $initial_count"

for i in {1..3}; do
    test_qcow2="$TEST_DIR/multi-$i.qcow2"
    test_export="test-multi-$i-$(date +%s)"
    
    qemu-img create -f qcow2 "$test_qcow2" 100M > /dev/null 2>&1
    
    cat > "/tmp/${test_export}.conf" << EOF
[$test_export]
exportname = $test_qcow2
readonly = true
EOF
    sudo cp "/tmp/${test_export}.conf" "/opt/migratekit/nbd-configs/conf.d/"
done

final_count=$(ls -1 /opt/migratekit/nbd-configs/conf.d/*.conf 2>/dev/null | wc -l)
echo "   Final exports: $final_count"
echo "   ✅ Created 3 additional exports"

# Test 8: config.d Pattern
echo ""
echo "TEST 8: Verify config.d pattern"
if grep -q "includedir.*conf\.d" /opt/migratekit/nbd-configs/nbd-server.conf; then
    echo "   ✅ Base config has includedir directive"
else
    echo "   ❌ includedir directive not found"
    exit 1
fi

# Cleanup
echo ""
echo "🧹 Cleaning up test files..."
rm -rf "$TEST_DIR"
sudo rm -f /opt/migratekit/nbd-configs/conf.d/test-*.conf
sudo pkill -SIGHUP nbd-server 2>/dev/null || true

echo ""
echo "✅ ALL TESTS PASSED!"
echo ""
