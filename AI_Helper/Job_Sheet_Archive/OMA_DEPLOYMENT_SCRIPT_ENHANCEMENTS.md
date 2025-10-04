# OMA Deployment Script Enhancements Needed

**Created**: October 3, 2025  
**Purpose**: Track changes needed to automate OMA deployment with new features  
**Status**: 📋 **DOCUMENTATION - Not Yet Implemented in Script**

---

## 🎯 **CURRENT DEPLOYMENT GAPS**

The OMA deployment scripts need to be updated to include:
1. Database migrations for new features
2. Job recovery system deployment
3. Failover visibility enhancements
4. SSH tunnel configuration (if not already included)

---

## 📝 **REQUIRED ENHANCEMENTS**

### **Enhancement 1: Database Migration Runner**

**Current Issue**: Migrations must be run manually

**Required Addition**:
```bash
# In OMA deployment script, add migration runner section

echo "🔄 Running database migrations..."
MIGRATION_DIR="/opt/migratekit/migrations"

# Copy migrations from deployment package
if [ -d "$DEPLOYMENT_PACKAGE/migrations" ]; then
    sudo mkdir -p $MIGRATION_DIR
    sudo cp $DEPLOYMENT_PACKAGE/migrations/*.sql $MIGRATION_DIR/
fi

# Run all pending migrations in order
for migration in $(ls $MIGRATION_DIR/*.up.sql 2>/dev/null | sort); do
    echo "  Applying: $(basename $migration)"
    mysql -u $DB_USER -p$DB_PASS $DB_NAME < $migration 2>&1 | grep -v "Duplicate column" || true
done

echo "✅ Database migrations complete"
```

**Migrations to Include in Package**:
- `20251003160000_add_operation_summary.up.sql` - Failover visibility
- Future migrations as they're created

---

### **Enhancement 2: VMA Services Configuration**

**Current Issue**: VMA client and poller need proper configuration

**Required Addition**:
```bash
# Add VMA service configuration section

echo "🔧 Configuring VMA services..."

# Set VMA API URL environment variable
VMA_API_URL="${VMA_API_URL:-http://localhost:9081}"  # Via SSH tunnel

# Update systemd service file
cat > /tmp/oma-api.service.env << EOF
VMA_API_URL=$VMA_API_URL
OMA_API_URL=http://localhost:8082
OMA_NBD_HOST=$OMA_IP
EOF

# Add environment file to service
sudo mkdir -p /etc/migratekit
sudo mv /tmp/oma-api.service.env /etc/migratekit/oma-api.env

# Update systemd service to use environment file
if ! grep -q "EnvironmentFile" /etc/systemd/system/oma-api.service; then
    sudo sed -i '/\[Service\]/a EnvironmentFile=/etc/migratekit/oma-api.env' /etc/systemd/system/oma-api.service
    sudo systemctl daemon-reload
fi

echo "✅ VMA services configured"
```

---

### **Enhancement 3: Job Recovery Validation**

**Current Issue**: No verification that job recovery is working after deployment

**Required Addition**:
```bash
# Add job recovery verification

echo "🔍 Verifying job recovery system..."

# Wait for service to fully start
sleep 5

# Check logs for job recovery
sudo journalctl -u oma-api --since "1 minute ago" | grep -E "job recovery" > /tmp/recovery_check.log

if grep -q "Job recovery completed successfully" /tmp/recovery_check.log; then
    echo "✅ Job recovery system operational"
elif grep -q "No active jobs found" /tmp/recovery_check.log; then
    echo "✅ Job recovery system operational (no jobs to recover)"
else
    echo "⚠️ Job recovery status unclear - check logs manually"
    sudo journalctl -u oma-api --since "1 minute ago" | grep "job recovery"
fi

rm -f /tmp/recovery_check.log
```

---

### **Enhancement 4: Migration Package Structure**

**Required Directory Structure**:
```
oma-deployment-package/
├── binaries/
│   ├── oma-api                          # Main binary
│   └── volume-daemon                    # If included
├── migrations/                          # NEW: Database migrations
│   ├── 20251003160000_add_operation_summary.up.sql
│   └── (other migrations as created)
├── scripts/
│   ├── deploy-oma.sh                    # Main deployment script
│   ├── setup-oma-ssh-tunnel.sh         # SSH tunnel setup
│   └── verify-deployment.sh            # Post-deployment verification
├── configs/
│   └── oma-api.service                  # Systemd service template
└── README.md                            # Deployment instructions
```

---

### **Enhancement 5: Post-Deployment Validation**

**Current Issue**: No automated validation that features are working

**Required Addition**:
```bash
# Add comprehensive validation section

echo "🧪 Running post-deployment validation..."

# Test 1: API Health
if curl -s -f http://localhost:8082/health > /dev/null; then
    echo "✅ API health check passed"
else
    echo "❌ API health check failed"
    exit 1
fi

# Test 2: Database Schema
COLUMN_CHECK=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -e \
    "SELECT COUNT(*) FROM information_schema.COLUMNS 
     WHERE TABLE_NAME='vm_replication_contexts' 
     AND COLUMN_NAME='last_operation_summary';" -sN)

if [ "$COLUMN_CHECK" -eq "1" ]; then
    echo "✅ Operation summary column present"
else
    echo "❌ Operation summary column missing"
    exit 1
fi

# Test 3: Job Recovery Logs
if sudo journalctl -u oma-api --since "2 minutes ago" | grep -q "Intelligent job recovery"; then
    echo "✅ Job recovery system initialized"
else
    echo "⚠️ Job recovery not detected in logs"
fi

# Test 4: VMA Progress Poller
if sudo journalctl -u oma-api --since "2 minutes ago" | grep -q "VMA progress poller started"; then
    echo "✅ VMA progress poller started"
else
    echo "⚠️ VMA progress poller not detected"
fi

echo "✅ Post-deployment validation complete"
```

---

## 📋 **DEPLOYMENT SCRIPT CHANGES SUMMARY**

### **Files to Modify**:
1. `oma-deployment-package/scripts/deploy-oma.sh` (main deployment script)
2. Create: `oma-deployment-package/migrations/` (directory)
3. Create: `oma-deployment-package/scripts/verify-deployment.sh` (validation script)

### **New Sections to Add**:
1. **Before Service Start**:
   - Run database migrations
   - Configure VMA services
   - Set environment variables

2. **After Service Start**:
   - Verify job recovery
   - Validate schema changes
   - Test API endpoints

---

## 🔧 **SPECIFIC MIGRATIONS TO INCLUDE**

### **Current Migrations** (as of October 3, 2025):

1. **20251003160000_add_operation_summary.up.sql**
   - Purpose: Failover visibility enhancement
   - Adds: `last_operation_summary` JSON column
   - Status: ✅ Required for v2.31.0+

2. **Future: Job recovery metadata** (if we add from job recovery job sheet)
   - Purpose: Enhanced job recovery tracking
   - Status: ⏳ Optional enhancement

---

## 📦 **DEPLOYMENT PACKAGE CHECKLIST**

Before releasing deployment package:

- [ ] Include all migration files in `migrations/` directory
- [ ] Update `deploy-oma.sh` with migration runner
- [ ] Add VMA service configuration
- [ ] Include post-deployment validation
- [ ] Test full deployment on clean system
- [ ] Document migration rollback procedure
- [ ] Update README with migration information

---

## 🚀 **AUTOMATED DEPLOYMENT FLOW**

**Proposed Enhanced Flow**:

```bash
#!/bin/bash
# deploy-oma.sh - Enhanced with automated migrations

# 1. Pre-flight checks
./scripts/preflight-check.sh

# 2. Backup current state
./scripts/backup-current.sh

# 3. Run database migrations
./scripts/run-migrations.sh

# 4. Deploy binaries
./scripts/deploy-binaries.sh

# 5. Configure services
./scripts/configure-services.sh

# 6. Restart services
systemctl restart oma-api
systemctl restart volume-daemon

# 7. Post-deployment validation
./scripts/verify-deployment.sh

# 8. Cleanup
./scripts/cleanup-temp-files.sh
```

---

## 📋 **MIGRATION RUNNER SCRIPT** (NEW)

**File**: `oma-deployment-package/scripts/run-migrations.sh`

```bash
#!/bin/bash
# run-migrations.sh - Database migration runner for OMA deployment

set -e

MIGRATION_DIR="${MIGRATION_DIR:-./migrations}"
DB_USER="${DB_USER:-oma_user}"
DB_PASS="${DB_PASS:-oma_password}"
DB_NAME="${DB_NAME:-migratekit_oma}"
DB_HOST="${DB_HOST:-localhost}"

echo "🔄 Running database migrations..."

# Check database connectivity
if ! mysql -u $DB_USER -p$DB_PASS -h $DB_HOST -e "SELECT 1" $DB_NAME > /dev/null 2>&1; then
    echo "❌ Cannot connect to database"
    exit 1
fi

# Create migrations tracking table if it doesn't exist
mysql -u $DB_USER -p$DB_PASS -h $DB_HOST $DB_NAME << 'EOF'
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(14) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description VARCHAR(255)
);
EOF

# Run migrations in order
MIGRATION_COUNT=0
APPLIED_COUNT=0

for migration in $(ls $MIGRATION_DIR/*.up.sql 2>/dev/null | sort); do
    MIGRATION_COUNT=$((MIGRATION_COUNT + 1))
    MIGRATION_NAME=$(basename $migration .up.sql)
    VERSION=$(echo $MIGRATION_NAME | grep -oP '^\d{14}')
    
    # Check if already applied
    ALREADY_APPLIED=$(mysql -u $DB_USER -p$DB_PASS -h $DB_HOST $DB_NAME -sN \
        -e "SELECT COUNT(*) FROM schema_migrations WHERE version='$VERSION'")
    
    if [ "$ALREADY_APPLIED" -eq "0" ]; then
        echo "  📥 Applying: $MIGRATION_NAME"
        
        # Run migration
        if mysql -u $DB_USER -p$DB_PASS -h $DB_HOST $DB_NAME < $migration 2>&1 | \
           grep -v "Duplicate column" | grep -v "Duplicate key"; then
            
            # Record successful application
            mysql -u $DB_USER -p$DB_PASS -h $DB_HOST $DB_NAME \
                -e "INSERT INTO schema_migrations (version, description) VALUES ('$VERSION', '$MIGRATION_NAME')" || true
            
            APPLIED_COUNT=$((APPLIED_COUNT + 1))
            echo "  ✅ Applied: $MIGRATION_NAME"
        else
            echo "  ⚠️  Migration may have partially failed: $MIGRATION_NAME"
        fi
    else
        echo "  ⏭️  Skipping (already applied): $MIGRATION_NAME"
    fi
done

echo "✅ Database migrations complete: $APPLIED_COUNT applied, $((MIGRATION_COUNT - APPLIED_COUNT)) skipped"
```

---

## 🎯 **CURRENT DEPLOYMENT STATUS**

**Server**: 10.245.246.147  
**Status**: ✅ **DEPLOYED AND RUNNING**

**What's Live**:
- ✅ OMA API v2.31.0 with failover visibility
- ✅ Database migration applied (last_operation_summary column exists)
- ✅ Job recovery working (1 job polling restarted)
- ✅ Service healthy and responding

**Logs Show**:
```
VMA progress poller started
Scheduler service started  
Intelligent job recovery system with VMA validation
Found 1 active jobs
Job still running on VMA - restarting polling ← WORKING!
Still running (polling restarted): 1
```

---

## 📝 **NEXT STEPS FOR DEPLOYMENT AUTOMATION**

1. **Create** `oma-deployment-package/scripts/run-migrations.sh`
2. **Update** `oma-deployment-package/scripts/deploy-oma.sh` to call it
3. **Add** migrations directory to deployment package
4. **Create** verification script
5. **Test** full automated deployment on clean system

---

**Deployment Log Saved**: `/home/pgrayson/migratekit-cloudstack/AI_Helper/OMA_DEPLOYMENT_SCRIPT_ENHANCEMENTS.md`


