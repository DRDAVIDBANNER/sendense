#!/bin/bash

# Test script to validate the enhanced cleanup logic
# Usage: ./test_cleanup_logic.sh <job_id>

if [ $# -lt 1 ]; then
    echo "Usage: $0 <job_id>"
    echo "Example: $0 job-20250821-090824"
    exit 1
fi

JOB_ID="$1"
DB_USER="oma_user"
DB_PASS="oma_password"  
DB_NAME="migratekit_oma"

echo "🔍 Testing cleanup logic for job: $JOB_ID"
echo "============================================"

echo "📊 vm_disks records for this job:"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT id, job_id, ossea_volume_id, cloudstack_volume_uuid 
FROM vm_disks 
WHERE job_id = '$JOB_ID'
"

echo ""
echo "📦 ossea_volumes referenced by this job:"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT ov.id, ov.volume_id, ov.volume_name, ov.status
FROM vm_disks vd
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
WHERE vd.job_id = '$JOB_ID'
"

echo ""
echo "🔗 device_mappings for volumes in this job:"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT dm.volume_uuid, dm.device_path, dm.cloudstack_state
FROM vm_disks vd
LEFT JOIN device_mappings dm ON vd.cloudstack_volume_uuid = dm.volume_uuid
WHERE vd.job_id = '$JOB_ID'
"

echo ""
echo "🧹 VOLUMES THAT WOULD BE CLEANED UP:"
echo "Method 1 (Direct cloudstack_volume_uuid):"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "
SELECT DISTINCT vd.cloudstack_volume_uuid as volume_uuid
FROM vm_disks vd 
WHERE vd.job_id = '$JOB_ID' 
AND vd.cloudstack_volume_uuid IS NOT NULL
"

echo ""
echo "Method 2 (Via ossea_volumes table):"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "
SELECT DISTINCT ov.volume_id as volume_uuid
FROM vm_disks vd
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
WHERE vd.job_id = '$JOB_ID'
"

echo ""
echo "Method 3 (Via device_mappings):"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "
SELECT DISTINCT dm.volume_uuid 
FROM vm_disks vd 
JOIN device_mappings dm ON vd.cloudstack_volume_uuid = dm.volume_uuid 
WHERE vd.job_id = '$JOB_ID'
"

echo ""
echo "📋 COMBINED CLEANUP LIST (what the fixed script would find):"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "
SELECT DISTINCT vd.cloudstack_volume_uuid as volume_uuid
FROM vm_disks vd 
WHERE vd.job_id = '$JOB_ID' 
AND vd.cloudstack_volume_uuid IS NOT NULL

UNION

SELECT DISTINCT ov.volume_id as volume_uuid
FROM vm_disks vd
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
WHERE vd.job_id = '$JOB_ID'

UNION

SELECT DISTINCT dm.volume_uuid 
FROM vm_disks vd 
JOIN device_mappings dm ON vd.cloudstack_volume_uuid = dm.volume_uuid 
WHERE vd.job_id = '$JOB_ID'
" | grep -v '^$' | sort

echo ""
echo "✅ Testing completed!"
