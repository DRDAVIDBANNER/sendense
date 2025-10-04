#!/bin/bash
# run-migrations.sh - Database migration runner for OMA deployment
# Applies all pending migrations in the migrations directory

set -e

# Configuration (can be overridden by environment variables)
MIGRATION_DIR="${MIGRATION_DIR:-$(dirname "$0")/../migrations}"
DB_USER="${DB_USER:-oma_user}"
DB_PASS="${DB_PASS:-oma_password}"
DB_NAME="${DB_NAME:-migratekit_oma}"
DB_HOST="${DB_HOST:-localhost}"

echo "🔄 Running OMA database migrations..."
echo "   Migration directory: $MIGRATION_DIR"
echo "   Database: $DB_NAME@$DB_HOST"

# Check database connectivity
if ! mysql -u $DB_USER -p$DB_PASS -h $DB_HOST -e "SELECT 1" $DB_NAME > /dev/null 2>&1; then
    echo "❌ Cannot connect to database"
    exit 1
fi
echo "✅ Database connection verified"

# Create migrations tracking table if it doesn't exist
mysql -u $DB_USER -p$DB_PASS -h $DB_HOST $DB_NAME << 'EOF'
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(14) PRIMARY KEY,
    description VARCHAR(255),
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
EOF

# Count migrations
MIGRATION_COUNT=0
APPLIED_COUNT=0
SKIPPED_COUNT=0

# Check if migration directory exists
if [ ! -d "$MIGRATION_DIR" ]; then
    echo "⚠️  Migration directory not found: $MIGRATION_DIR"
    echo "   No migrations to apply"
    exit 0
fi

# Run migrations in order
for migration in $(ls $MIGRATION_DIR/*.up.sql 2>/dev/null | sort); do
    MIGRATION_COUNT=$((MIGRATION_COUNT + 1))
    MIGRATION_NAME=$(basename $migration .up.sql)
    VERSION=$(echo $MIGRATION_NAME | grep -oP '^\d{14}')
    
    # Check if already applied
    ALREADY_APPLIED=$(mysql -u $DB_USER -p$DB_PASS -h $DB_HOST $DB_NAME -sN \
        -e "SELECT COUNT(*) FROM schema_migrations WHERE version='$VERSION'" 2>/dev/null || echo "0")
    
    if [ "$ALREADY_APPLIED" -eq "0" ]; then
        echo "  📥 Applying: $MIGRATION_NAME"
        
        # Run migration and capture output
        MIGRATION_OUTPUT=$(mysql -u $DB_USER -p$DB_PASS -h $DB_HOST $DB_NAME < $migration 2>&1)
        MIGRATION_EXIT=$?
        
        # Check for errors (but ignore "Duplicate column" which is harmless)
        if echo "$MIGRATION_OUTPUT" | grep -i "ERROR" | grep -v "Duplicate column" | grep -v "Duplicate key" > /dev/null; then
            echo "  ❌ Migration failed: $MIGRATION_NAME"
            echo "$MIGRATION_OUTPUT"
            exit 1
        fi
        
        # Record successful application
        mysql -u $DB_USER -p$DB_PASS -h $DB_HOST $DB_NAME \
            -e "INSERT INTO schema_migrations (version, description) VALUES ('$VERSION', '$MIGRATION_NAME')" 2>/dev/null || true
        
        APPLIED_COUNT=$((APPLIED_COUNT + 1))
        echo "  ✅ Applied: $MIGRATION_NAME"
    else
        SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
        echo "  ⏭️  Skipping (already applied): $MIGRATION_NAME"
    fi
done

if [ $MIGRATION_COUNT -eq 0 ]; then
    echo "ℹ️  No migration files found in $MIGRATION_DIR"
else
    echo ""
    echo "✅ Database migrations complete:"
    echo "   Total: $MIGRATION_COUNT"
    echo "   Applied: $APPLIED_COUNT"
    echo "   Skipped: $SKIPPED_COUNT"
fi


