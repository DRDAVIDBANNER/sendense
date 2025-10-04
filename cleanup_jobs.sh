#!/bin/bash

# Script to clean up all existing replication jobs using the enhanced deletion API
# This prepares the database for the VM-centric architecture implementation

set -e  # Exit on any error

echo "=== MigrateKit OMA Job Cleanup Script ==="
echo "Starting systematic cleanup of all existing replication jobs..."
echo

# Get all job IDs
JOB_IDS=$(mysql -u oma_user -poma_password -D migratekit_oma -se "SELECT id FROM replication_jobs ORDER BY created_at ASC;")

if [ -z "$JOB_IDS" ]; then
    echo "No replication jobs found. Database is already clean."
    exit 0
fi

# Count total jobs
TOTAL_JOBS=$(echo "$JOB_IDS" | wc -l)
echo "Found $TOTAL_JOBS replication jobs to delete"
echo

# Delete each job
COUNT=0
DELETED=0
FAILED=0

for JOB_ID in $JOB_IDS; do
    COUNT=$((COUNT + 1))
    echo "[$COUNT/$TOTAL_JOBS] Deleting job: $JOB_ID"
    
    # Call the enhanced deletion API
    RESPONSE=$(curl -s -w "%{http_code}" -X DELETE "http://localhost:8082/api/v1/replications/$JOB_ID")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo "  ✅ Successfully deleted job $JOB_ID"
        DELETED=$((DELETED + 1))
    else
        echo "  ❌ Failed to delete job $JOB_ID (HTTP $HTTP_CODE)"
        if [ -n "$BODY" ]; then
            echo "     Response: $BODY"
        fi
        FAILED=$((FAILED + 1))
    fi
    
    # Small delay to avoid overwhelming the API
    sleep 0.5
done

echo
echo "=== Cleanup Summary ==="
echo "Total jobs processed: $TOTAL_JOBS"
echo "Successfully deleted: $DELETED"
echo "Failed deletions: $FAILED"
echo

# Verify cleanup
echo "=== Verification ==="
REMAINING=$(mysql -u oma_user -poma_password -D migratekit_oma -se "SELECT COUNT(*) FROM replication_jobs;")
echo "Remaining replication jobs: $REMAINING"

if [ "$REMAINING" = "0" ]; then
    echo "✅ Database cleanup completed successfully!"
else
    echo "⚠️  Warning: $REMAINING jobs remain in database"
    echo "Listing remaining jobs:"
    mysql -u oma_user -poma_password -D migratekit_oma -e "SELECT id, source_vm_name, status FROM replication_jobs;"
fi

echo
echo "Job cleanup script completed."
