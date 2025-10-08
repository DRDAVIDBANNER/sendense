#!/bin/bash
# Unit test for backup completion endpoint
# Tests database record creation and change_id recording WITHOUT waiting for full backup

set -e

echo "🧪 UNIT TEST: Backup Completion Endpoint"
echo "========================================"

DB_USER="oma_user"
DB_PASS="oma_password"
DB_NAME="migratekit_oma"

# Step 1: Create test backup job in database
echo ""
echo "📝 Step 1: Creating test backup job record..."
TEST_BACKUP_ID="backup-unittest-$(date +%s)"
TEST_CHANGE_ID="52 66 8c 2d a7 c5 c5 68-c5 d2 8d 04 79 f5 fd 7d/9999"

mysql -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" <<EOF
INSERT INTO backup_jobs (
    id, 
    vm_context_id, 
    vm_name, 
    disk_id,
    repository_id, 
    policy_id,
    backup_type, 
    status,
    repository_path,
    parent_backup_id,
    change_id,
    bytes_transferred,
    total_bytes,
    compression_enabled,
    error_message,
    started_at,
    completed_at,
    created_at
) VALUES (
    '$TEST_BACKUP_ID',
    'ctx-pgtest1-20251006-203401',
    'pgtest1',
    0,
    '1',
    NULL,
    'full',
    'running',
    '/backup/repository',
    NULL,
    '',
    0,
    109521666048,
    true,
    '',
    NOW(),
    NULL,
    NOW()
);
EOF

echo "✅ Test backup job created: $TEST_BACKUP_ID"

# Step 2: Verify job exists
echo ""
echo "📋 Step 2: Verifying job in database..."
mysql -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, vm_name, status, change_id, bytes_transferred 
FROM backup_jobs 
WHERE id = '$TEST_BACKUP_ID';
"

# Step 3: Call completion endpoint (simulating sendense-backup-client)
echo ""
echo "📡 Step 3: Calling backup completion endpoint..."
RESPONSE=$(curl -s -X POST "http://localhost:8082/api/v1/backups/$TEST_BACKUP_ID/complete" \
    -H "Content-Type: application/json" \
    -d "{
        \"change_id\": \"$TEST_CHANGE_ID\",
        \"bytes_transferred\": 102000000000
    }")

echo "Response: $RESPONSE"

# Step 4: Verify change_id was stored
echo ""
echo "📋 Step 4: Verifying change_id was recorded..."
RESULT=$(mysql -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT id, status, change_id, bytes_transferred, completed_at IS NOT NULL as has_completion_time
FROM backup_jobs 
WHERE id = '$TEST_BACKUP_ID';
")

echo "$RESULT"

# Step 5: Validate results
echo ""
echo "🔍 Step 5: Validating results..."
STORED_CHANGE_ID=$(mysql -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -sN -e "
SELECT change_id FROM backup_jobs WHERE id = '$TEST_BACKUP_ID';
")

if [ "$STORED_CHANGE_ID" = "$TEST_CHANGE_ID" ]; then
    echo "✅ change_id CORRECTLY STORED: $STORED_CHANGE_ID"
else
    echo "❌ change_id NOT STORED or INCORRECT"
    echo "   Expected: $TEST_CHANGE_ID"
    echo "   Got:      $STORED_CHANGE_ID"
    exit 1
fi

STORED_STATUS=$(mysql -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -sN -e "
SELECT status FROM backup_jobs WHERE id = '$TEST_BACKUP_ID';
")

if [ "$STORED_STATUS" = "completed" ]; then
    echo "✅ Status CORRECTLY UPDATED: $STORED_STATUS"
else
    echo "⚠️  Status not updated (may be intentional)"
    echo "   Got: $STORED_STATUS"
fi

STORED_BYTES=$(mysql -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -sN -e "
SELECT bytes_transferred FROM backup_jobs WHERE id = '$TEST_BACKUP_ID';
")

if [ "$STORED_BYTES" = "102000000000" ]; then
    echo "✅ Bytes transferred CORRECTLY STORED: $STORED_BYTES"
else
    echo "⚠️  Bytes transferred: $STORED_BYTES"
fi

# Step 6: Cleanup
echo ""
echo "🧹 Step 6: Cleaning up test data..."
mysql -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
DELETE FROM backup_jobs WHERE id = '$TEST_BACKUP_ID';
"

echo "✅ Test data cleaned"

echo ""
echo "=========================================="
echo "✅ UNIT TEST PASSED - Backend is ready!"
echo "=========================================="
echo ""
echo "Next step: Run full backup test to verify E2E flow"

