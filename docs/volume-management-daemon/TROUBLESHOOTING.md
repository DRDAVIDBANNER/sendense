# Volume Management Daemon - Troubleshooting Guide

**Comprehensive troubleshooting for common issues and diagnostic procedures**

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Common Issues](#common-issues)
3. [Diagnostic Commands](#diagnostic-commands)
4. [Log Analysis](#log-analysis)
5. [Database Issues](#database-issues)
6. [CloudStack Issues](#cloudstack-issues)
7. [Device Monitoring Issues](#device-monitoring-issues)
8. [Performance Issues](#performance-issues)
9. [Recovery Procedures](#recovery-procedures)
10. [Debug Mode](#debug-mode)

---

## Quick Diagnostics

### Health Check Sequence

Run these commands in order to quickly identify issues:

```bash
# 1. Service status
systemctl status volume-daemon

# 2. Basic connectivity  
curl -f http://localhost:8090/health || echo "HTTP server not responding"

# 3. Detailed health
curl -s http://localhost:8090/api/v1/health | jq '.cloudstack_health, .database_health, .device_monitor'

# 4. Recent logs
journalctl -u volume-daemon --since="5 minutes ago" | tail -20

# 5. Database connectivity
mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) FROM volume_operations;" 2>/dev/null || echo "Database connection failed"
```

### Service Status Interpretation

```bash
systemctl status volume-daemon
```

**Healthy Output**:
```
● volume-daemon.service - Volume Management Daemon
   Active: active (running) since Mon 2025-08-19 20:00:00 BST; 2h ago
   Main PID: 12345
```

**Problem Indicators**:
- `Active: failed` - Service crashed
- `Active: activating` - Service starting but not ready
- `Main PID: (none)` - Process not running

---

## Common Issues

### 1. Multi-Volume VM Correlation Issues (RESOLVED v1.2.0)

**CRITICAL FIX**: Multi-volume VMs like QUAD-AUVIK02 now work correctly.

#### Symptoms (BEFORE v1.2.0):
- First volume attaches successfully
- Second+ volumes timeout with `No fresh device detected during correlation timeout`
- VMs with multiple disks fail replication setup
- Logs show contemporary events being "kept" but correlation still times out

#### Root Cause (IDENTIFIED & FIXED):
- **Channel consumption bug**: Pre-draining logic consumed contemporary events before correlation
- **Event loss**: Events couldn't be "put back" into channel once consumed during drain
- **Race condition**: Timing between device detection and correlation start

#### Solution Implemented (v1.2.0):
```bash
# Check for new correlation logic in logs:
journalctl -u volume-daemon --since="5 minutes ago" | grep -E "correlation.*no pre-draining|contemporary.*device"

# Expected SUCCESS logs:
# "Starting device correlation with timestamp filtering (no pre-draining)"
# "Using contemporary/fresh device for correlation"
# NO MORE: "Drained stale device events" or "Keeping contemporary device event"
```

#### Verification Commands:
```bash
# 1. Check running version has the fix
ls -la /usr/local/bin/volume-daemon | grep "Sep  4 16:" # Should be v1.2.0 timestamp

# 2. Test multi-volume VM (like QUAD-AUVIK02)
# Both volumes should attach with unique device paths

# 3. Monitor correlation logs
journalctl -u volume-daemon -f | grep -E "correlation|device.*detected|skipping.*stale"
```

#### Multi-Volume Success Indicators:
- **Volume 1**: Attaches to `/dev/vdc` or `/dev/vdb` (depending on existing attachments)
- **Volume 2**: Attaches to next available device path (`/dev/vdd`, etc.)
- **Logs**: Show "Using contemporary/fresh device" for BOTH volumes
- **Database**: Both volumes have unique device_path entries
- **NBD Exports**: Both volumes get separate NBD exports

### 2. Service Won't Start

**Symptoms**:
- `systemctl start volume-daemon` fails
- Service exits immediately after start
- No HTTP server response

**Diagnostic Steps**:

```bash
# Check service logs
journalctl -u volume-daemon -n 50

# Check binary permissions
ls -la /usr/local/bin/volume-daemon

# Test manual startup
sudo /usr/local/bin/volume-daemon

# Check port availability
netstat -tlnp | grep :8090
```

**Common Causes & Solutions**:

#### Database Connection Failed
```
ERROR: Failed to initialize database: dial tcp: connection refused
```

**Solution**:
```bash
# Test database manually
mysql -u oma_user -poma_password migratekit_oma -e "SELECT 1"

# Check if database service is running
systemctl status mysql

# Verify database exists
mysql -u oma_user -poma_password -e "SHOW DATABASES" | grep migratekit_oma
```

#### Port Already in Use
```
ERROR: Failed to start server: listen tcp :8090: bind: address already in use
```

**Solution**:
```bash
# Find process using port
lsof -i :8090

# Kill conflicting process
sudo kill -9 <PID>

# Or change port in configuration
export PORT=8091
```

#### Permission Denied
```
ERROR: Failed to watch /sys/block: permission denied
```

**Solution**:
```bash
# Ensure running as root
sudo systemctl start volume-daemon

# Check /sys/block permissions
ls -la /sys/block/
```

### 2. CloudStack Operations Fail

**Symptoms**:
- All volume operations return CloudStack errors
- Health check shows `cloudstack_health: unhealthy`
- Operations stuck in `pending` status

**Diagnostic Steps**:

```bash
# Check CloudStack health
curl -s http://localhost:8090/api/v1/health | jq '.cloudstack_health'

# Test CloudStack config
go run cmd/test-disk-offerings/main.go

# Check database config
mysql -u oma_user -poma_password migratekit_oma -e "SELECT name, api_url, zone, is_active FROM ossea_configs WHERE is_active = 1"
```

**Common Causes & Solutions**:

#### Invalid API Credentials
```
CloudStack API error 401: unable to verify user credentials
```

**Solution**:
```sql
-- Update credentials in database
UPDATE ossea_configs 
SET api_key = 'new_key', secret_key = 'new_secret' 
WHERE is_active = 1;
```

#### Invalid Zone ID
```
CloudStack API error 431: Invalid parameter zoneid
```

**Solution**:
```bash
# Get correct zone ID
go run cmd/test-disk-offerings/main.go

# Update database
mysql -u oma_user -poma_password migratekit_oma -e "UPDATE ossea_configs SET zone = 'correct-zone-uuid' WHERE is_active = 1"
```

#### Network Connectivity
```
CloudStack API error: dial tcp: connection refused
```

**Solution**:
```bash
# Test API URL directly
curl -k "https://cloudstack.example.com/client/api?command=listZones&response=json"

# Check DNS resolution
nslookup cloudstack.example.com

# Check firewall rules
iptables -L | grep cloudstack
```

### 3. Device Detection Not Working

**Symptoms**:
- Volume attachments succeed but no device path detected
- `device_path` is empty in operation responses
- Device monitor shows 0 devices

**Diagnostic Steps**:

```bash
# Check current devices manually
ls -la /sys/block/vd*

# Test device monitor
go run cmd/test-polling-monitor/main.go

# Check daemon logs for device events
journalctl -u volume-daemon | grep -i device
```

**Common Causes & Solutions**:

#### No Virtio Devices Present
```
DEBUG: Found 0 virtio devices
```

**Solution**:
```bash
# Check if any block devices exist
ls -la /sys/block/

# Check for different device types
ls -la /dev/sd* /dev/xvd* /dev/nvme*

# Verify VM has virtio devices
lspci | grep -i virtio
```

#### Device Monitor Not Running
```
WARN: Device monitor not available for correlation
```

**Solution**:
```bash
# Check if polling monitor started
journalctl -u volume-daemon | grep "polling monitor"

# Restart daemon to reinitialize monitor
systemctl restart volume-daemon
```

#### Permission Issues
```
ERROR: Failed to read /sys/block: permission denied
```

**Solution**:
```bash
# Ensure daemon runs as root
sudo systemctl restart volume-daemon

# Check /sys/block accessibility
sudo ls -la /sys/block/
```

### 4. Database Corruption

**Symptoms**:
- Unique constraint violations
- Multiple volumes claiming same device path
- Inconsistent device mappings

**Diagnostic Steps**:

```sql
-- Check for duplicate device paths
SELECT device_path, COUNT(*) as count 
FROM device_mappings 
GROUP BY device_path 
HAVING COUNT(*) > 1;

-- Check for orphaned operations
SELECT COUNT(*) FROM volume_operations 
WHERE status IN ('pending', 'executing') 
AND created_at < DATE_SUB(NOW(), INTERVAL 1 HOUR);

-- Check for missing mappings
SELECT vo.volume_id 
FROM volume_operations vo 
WHERE vo.type = 'attach' AND vo.status = 'completed'
AND NOT EXISTS (
    SELECT 1 FROM device_mappings dm 
    WHERE dm.volume_id = vo.volume_id
);
```

**Recovery Procedures**:

#### Clean Duplicate Device Paths
```sql
-- Backup first
CREATE TABLE device_mappings_backup AS SELECT * FROM device_mappings;

-- Remove duplicates (keep newest)
DELETE dm1 FROM device_mappings dm1
INNER JOIN device_mappings dm2 
WHERE dm1.id < dm2.id 
AND dm1.device_path = dm2.device_path;
```

#### Reset Stuck Operations
```sql
-- Mark old pending operations as failed
UPDATE volume_operations 
SET status = 'failed', 
    error = 'Timeout - marked as failed during cleanup',
    updated_at = NOW(),
    completed_at = NOW()
WHERE status IN ('pending', 'executing') 
AND created_at < DATE_SUB(NOW(), INTERVAL 1 HOUR);
```

---

## Diagnostic Commands

### API Diagnostics

```bash
# Test all API endpoints
curl -f http://localhost:8090/health
curl -f http://localhost:8090/api/v1/health
curl -f http://localhost:8090/api/v1/metrics
curl -f http://localhost:8090/api/v1/operations

# Test volume creation (dry run)
curl -X POST http://localhost:8090/api/v1/volumes \
  -H "Content-Type: application/json" \
  -d '{"name":"test","size":1073741824,"disk_offering_id":"test","zone_id":"test"}' \
  && echo "API accepting requests" || echo "API rejecting requests"
```

### Database Diagnostics

```sql
-- Operation statistics
SELECT 
    type,
    status,
    COUNT(*) as count,
    AVG(TIMESTAMPDIFF(SECOND, created_at, completed_at)) as avg_duration_seconds
FROM volume_operations 
WHERE created_at > DATE_SUB(NOW(), INTERVAL 24 HOUR)
GROUP BY type, status;

-- Recent operations
SELECT 
    id,
    type,
    status,
    volume_id,
    created_at,
    TIMESTAMPDIFF(SECOND, created_at, NOW()) as age_seconds
FROM volume_operations 
ORDER BY created_at DESC 
LIMIT 10;

-- Device mapping status
SELECT 
    cloudstack_state,
    linux_state,
    COUNT(*) as count
FROM device_mappings 
GROUP BY cloudstack_state, linux_state;

-- Stale mappings
SELECT 
    volume_id,
    device_path,
    TIMESTAMPDIFF(HOUR, last_sync, NOW()) as hours_since_sync
FROM device_mappings 
WHERE last_sync < DATE_SUB(NOW(), INTERVAL 6 HOUR);
```

### System Diagnostics

```bash
# Memory and CPU usage
ps aux | grep volume-daemon | grep -v grep

# Open file descriptors
lsof -p $(pgrep volume-daemon)

# Network connections
netstat -tlnp | grep $(pgrep volume-daemon)

# Disk space
df -h /var/log
du -sh /var/log/volume-daemon*

# System resources
free -h
cat /proc/loadavg
```

### CloudStack Diagnostics

```bash
# Test CloudStack connectivity
go run cmd/test-disk-offerings/main.go

# Test device correlation
go run cmd/test-deviceid-correlation/main.go

# Manual CloudStack API test
curl -k "https://cloudstack-url/client/api?command=listZones&response=json&apikey=YOUR_KEY&signature=YOUR_SIGNATURE"
```

---

## Log Analysis

### Log Levels and Meanings

**INFO**: Normal operation events
```
INFO[0000] ✅ Linux device polling monitor started successfully
INFO[0015] Creating CloudStack volume
INFO[0023] 📎 New block device detected via polling
```

**WARN**: Issues that don't prevent operation
```
WARN[0045] Failed to get device info for added device
WARN[0060] Device event channel full, dropping event
WARN[0120] No device detected during correlation timeout
```

**ERROR**: Operation failures
```
ERROR[0030] CloudStack volume creation failed: Invalid parameter zoneid
ERROR[0090] Failed to create operation record: database connection lost
ERROR[0150] Failed to update operation with error status
```

### Key Log Patterns

#### Successful Volume Creation
```
INFO Creating CloudStack volume name=test-volume size=5368709120
INFO Volume creation completed volume_id=vol-12345 duration_ms=3240
```

#### Device Detection
```
INFO 📎 New block device detected via polling device_path=/dev/vdb size=5368717312 controller=virtio4
INFO ✅ Device detected during volume attachment volume_id=vol-12345 device_path=/dev/vdb
```

#### CloudStack Errors
```
ERROR CloudStack volume creation failed: CloudStack API error 431 (CSExceptionErrorCode: 9999): Invalid parameter zoneid
ERROR CloudStack volume attachment failed: CloudStack API error 432: Resource limit exceeded
```

#### Database Issues
```
ERROR Failed to create operation record: Error 1062: Duplicate entry 'op-12345' for key 'PRIMARY'
ERROR Failed to update operation: Error 2006: MySQL server has gone away
```

### Log Filtering Commands

```bash
# Recent errors only
journalctl -u volume-daemon --since="1 hour ago" -p err

# Volume operations
journalctl -u volume-daemon | grep -E "(Creating|completed|failed).*volume"

# Device events
journalctl -u volume-daemon | grep -E "(device|Device|📎|📌)"

# CloudStack interactions
journalctl -u volume-daemon | grep -i cloudstack

# Performance issues
journalctl -u volume-daemon | grep -E "(timeout|slow|duration|ms)"

# Follow live logs with filtering
journalctl -u volume-daemon -f | grep -E "(ERROR|WARN|device|volume)"
```

---

## Database Issues

### Connection Problems

#### Connection Pool Exhaustion
```
ERROR: database connection pool exhausted
```

**Diagnosis**:
```sql
SHOW PROCESSLIST;
SHOW STATUS LIKE 'Threads_connected';
SHOW VARIABLES LIKE 'max_connections';
```

**Solution**:
```sql
-- Increase connection limit
SET GLOBAL max_connections = 200;

-- Kill old connections
KILL <connection_id>;
```

#### Deadlocks
```
ERROR: Deadlock found when trying to get lock
```

**Diagnosis**:
```sql
SHOW ENGINE INNODB STATUS;
SELECT * FROM INFORMATION_SCHEMA.INNODB_LOCKS;
```

**Solution**:
- Review transaction order in code
- Add appropriate indexes
- Reduce transaction scope

### Data Corruption

#### Orphaned Records

**Find orphaned operations**:
```sql
SELECT vo.id, vo.volume_id, vo.status 
FROM volume_operations vo
LEFT JOIN device_mappings dm ON vo.volume_id = dm.volume_id
WHERE vo.type = 'attach' 
AND vo.status = 'completed' 
AND dm.volume_id IS NULL;
```

**Clean up orphans**:
```sql
-- Create device mappings for completed attach operations without mappings
INSERT INTO device_mappings (id, volume_id, vm_id, device_path, cloudstack_state, linux_state, size, last_sync, created_at, updated_at)
SELECT 
    CONCAT('cleanup-', UNIX_TIMESTAMP()),
    vo.volume_id,
    JSON_UNQUOTE(JSON_EXTRACT(vo.request, '$.vm_id')),
    'unknown',
    'attached',
    'unknown',
    0,
    NOW(),
    NOW(),
    NOW()
FROM volume_operations vo
LEFT JOIN device_mappings dm ON vo.volume_id = dm.volume_id
WHERE vo.type = 'attach' 
AND vo.status = 'completed' 
AND dm.volume_id IS NULL;
```

### Performance Issues

#### Slow Queries

**Enable query logging**:
```sql
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 2;
SET GLOBAL slow_query_log_file = '/var/log/mysql/slow-queries.log';
```

**Analyze slow queries**:
```bash
# Install pt-query-digest
sudo apt-get install percona-toolkit

# Analyze slow query log
pt-query-digest /var/log/mysql/slow-queries.log
```

**Common optimizations**:
```sql
-- Add missing indexes
CREATE INDEX idx_volume_operations_type_status ON volume_operations(type, status);
CREATE INDEX idx_device_mappings_last_sync ON device_mappings(last_sync);

-- Analyze table statistics
ANALYZE TABLE volume_operations;
ANALYZE TABLE device_mappings;
```

---

## CloudStack Issues

### Authentication Problems

#### Invalid Credentials
```
CloudStack API error 401: unable to verify user credentials
```

**Diagnosis**:
```bash
# Test credentials manually
go run cmd/test-disk-offerings/main.go

# Check database config
mysql -u oma_user -poma_password migratekit_oma -e "SELECT api_key, secret_key FROM ossea_configs WHERE is_active = 1"
```

**Solution**:
```sql
-- Update credentials
UPDATE ossea_configs 
SET api_key = 'new_api_key', secret_key = 'new_secret_key'
WHERE is_active = 1;

-- Restart daemon to pick up new config
```

```bash
systemctl restart volume-daemon
```

### API Rate Limiting

#### Too Many Requests
```
CloudStack API error 429: Rate limit exceeded
```

**Solution**:
- Implement exponential backoff
- Reduce concurrent operations
- Contact CloudStack administrator to increase limits

### Resource Limits

#### Quota Exceeded
```
CloudStack API error 432: Resource limit exceeded for account
```

**Diagnosis**:
```bash
# Check CloudStack quotas via API
# (Requires manual CloudStack API call)
```

**Solution**:
- Clean up unused volumes
- Request quota increase
- Implement volume lifecycle management

---

## Device Monitoring Issues

### No Devices Detected

**Symptoms**:
```
INFO[0000] 📋 Scanned existing block devices device_count=0
```

**Diagnosis**:
```bash
# Check for virtio devices manually
ls -la /sys/block/vd*

# Check all block devices
ls -la /sys/block/

# Test device detection logic
go run cmd/debug-device-scan/main.go
```

**Solutions**:

#### Wrong Device Type
```bash
# Check for other device types
ls -la /dev/sd* /dev/xvd* /dev/nvme*

# Update device filter if needed (code change required)
```

#### Permission Issues
```bash
# Ensure daemon runs as root
sudo systemctl restart volume-daemon

# Check permissions
ls -la /sys/block/
```

### Missed Device Events

**Symptoms**:
- Volume attachments succeed in CloudStack
- No device events generated
- Device correlation timeouts

**Diagnosis**:
```bash
# Test polling monitor
go run cmd/test-polling-monitor/main.go

# Check polling interval
journalctl -u volume-daemon | grep "poll_interval"

# Test manual polling
go run cmd/test-device-polling/main.go
```

**Solutions**:

#### Increase Polling Frequency
```go
// In polling_monitor.go
pollInterval: 1 * time.Second  // Reduce from 2 seconds
```

#### Increase Correlation Timeout
```go
// In volume_service.go
timeout := 60 * time.Second  // Increase from 30 seconds
```

---

## Performance Issues

### High CPU Usage

**Diagnosis**:
```bash
# Check CPU usage
top -p $(pgrep volume-daemon)

# Profile the daemon
go tool pprof http://localhost:8090/debug/pprof/profile
```

**Common Causes**:
- Polling too frequently
- Too many concurrent operations
- Inefficient database queries

**Solutions**:
```go
// Increase polling interval
pollInterval: 5 * time.Second

// Limit concurrent operations
semaphore := make(chan struct{}, 10)
```

### High Memory Usage

**Diagnosis**:
```bash
# Memory usage
ps aux | grep volume-daemon

# Memory profile
go tool pprof http://localhost:8090/debug/pprof/heap
```

**Common Causes**:
- Event channel buffer overflow
- Large device state cache
- Memory leaks in goroutines

**Solutions**:
```go
// Limit event buffer size
eventChan: make(chan service.DeviceEvent, 50)

// Periodic cache cleanup
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cleanupStaleDevices()
    }
}()
```

### Slow Response Times

**Diagnosis**:
```bash
# API response time test
time curl -s http://localhost:8090/api/v1/health > /dev/null

# Database query timing
mysql -u oma_user -poma_password migratekit_oma -e "
SELECT 
    ROUND(AVG(timer_wait/1000000000000),6) as avg_sec,
    count_star as calls,
    sql_text 
FROM performance_schema.events_statements_summary_by_digest 
WHERE sql_text LIKE '%volume_operations%' 
ORDER BY avg_sec DESC 
LIMIT 5;"
```

**Solutions**:
- Add database indexes
- Optimize query patterns  
- Implement caching
- Use connection pooling

---

## Recovery Procedures

### Complete System Recovery

#### 1. Service Recovery
```bash
# Stop daemon
systemctl stop volume-daemon

# Check for stuck processes
ps aux | grep volume-daemon
sudo kill -9 <any_stuck_pids>

# Start fresh
systemctl start volume-daemon

# Verify startup
journalctl -u volume-daemon --since="1 minute ago"
curl -f http://localhost:8090/health
```

#### 2. Database Recovery
```sql
-- Backup current state
mysqldump -u oma_user -poma_password migratekit_oma volume_operations device_mappings > backup_$(date +%Y%m%d_%H%M%S).sql

-- Reset stuck operations
UPDATE volume_operations 
SET status = 'failed', error = 'Reset during recovery', updated_at = NOW(), completed_at = NOW()
WHERE status IN ('pending', 'executing');

-- Clean duplicate mappings
-- (See database corruption section above)

-- Verify integrity
SELECT 
    (SELECT COUNT(*) FROM volume_operations WHERE status IN ('pending', 'executing')) as stuck_operations,
    (SELECT COUNT(*) FROM device_mappings GROUP BY device_path HAVING COUNT(*) > 1) as duplicate_paths;
```

#### 3. CloudStack State Sync
```bash
# Force synchronization
curl -X POST http://localhost:8090/api/v1/admin/force-sync

# Verify CloudStack connectivity
go run cmd/test-disk-offerings/main.go

# Check operation processing
curl -s http://localhost:8090/api/v1/operations | jq '.[] | select(.status == "pending")'
```

### Partial Recovery Procedures

#### Reset Single Volume Operation
```sql
-- Find the operation
SELECT id, type, status, volume_id, error FROM volume_operations WHERE volume_id = 'vol-12345';

-- Reset to failed state
UPDATE volume_operations 
SET status = 'failed', error = 'Manual reset', updated_at = NOW(), completed_at = NOW()
WHERE id = 'op-12345';

-- Clean up device mapping if needed
DELETE FROM device_mappings WHERE volume_id = 'vol-12345';
```

#### Rebuild Device Mappings
```bash
# Stop daemon
systemctl stop volume-daemon

# Clear all mappings
mysql -u oma_user -poma_password migratekit_oma -e "TRUNCATE TABLE device_mappings;"

# Start daemon (will scan current devices)
systemctl start volume-daemon

# Check current device state
curl -s http://localhost:8090/api/v1/metrics | jq '.active_mappings'
```

---

## Debug Mode

### Enable Debug Logging

#### Temporary Debug Mode
```bash
# Set log level via environment
export LOG_LEVEL=debug
systemctl restart volume-daemon

# Or modify service file
sudo systemctl edit volume-daemon
```

Add:
```ini
[Service]
Environment=LOG_LEVEL=debug
```

#### Permanent Debug Configuration
```go
// In main.go
log.SetLevel(log.DebugLevel)
```

### Debug Output Examples

#### Verbose Device Monitoring
```
DEBUG[0001] Filesystem event detected event=CREATE path=/sys/block/vdb device_name=vdb device_path=/dev/vdb
DEBUG[0001] Device size check device_size=5368717312 volume_size=5368709120 size_diff=8192 tolerance=3221225472 matches=true
DEBUG[0001] Device match score calculated device_path=/dev/vdb device_size=5368717312 volume_size=5368709120 score=0.95
```

#### Verbose CloudStack Interactions
```
DEBUG[0001] CloudStack API call operation=CreateVolume name=test-volume size=5368709120
DEBUG[0003] CloudStack response received volume_id=vol-12345 state=Allocated
DEBUG[0003] CloudStack volume state change volume_id=vol-12345 old_state=Allocated new_state=Ready
```

#### Verbose Database Operations
```
DEBUG[0001] Database operation type=CreateOperation id=op-12345 duration_ms=15
DEBUG[0002] Database query sql="SELECT * FROM volume_operations WHERE id = ?" params=["op-12345"] duration_ms=3
DEBUG[0002] Database result rows_affected=1 last_insert_id=0
```

### Testing Utilities

#### Isolated Component Tests
```bash
# Test only device monitoring
go run cmd/test-polling-monitor/main.go

# Test only CloudStack integration
go run cmd/test-disk-offerings/main.go

# Test only database operations
mysql -u oma_user -poma_password migratekit_oma < test_queries.sql
```

#### Load Testing
```bash
# Concurrent API calls
for i in {1..10}; do
    curl -X POST http://localhost:8090/api/v1/volumes -H "Content-Type: application/json" -d "{\"name\":\"test-$i\",\"size\":1073741824,\"disk_offering_id\":\"test\",\"zone_id\":\"test\"}" &
done
wait

# Monitor performance
curl -s http://localhost:8090/api/v1/metrics | jq '.average_response_time_ms, .error_rate_percent'
```

---

## Emergency Procedures

### Complete System Failure

1. **Stop all operations**:
```bash
systemctl stop volume-daemon
```

2. **Preserve evidence**:
```bash
# Copy logs
cp /var/log/volume-daemon.log /tmp/emergency-backup-$(date +%Y%m%d_%H%M%S).log
journalctl -u volume-daemon > /tmp/emergency-journal-$(date +%Y%m%d_%H%M%S).log

# Database backup
mysqldump -u oma_user -poma_password migratekit_oma > /tmp/emergency-db-$(date +%Y%m%d_%H%M%S).sql
```

3. **Contact support with**:
- Emergency backup files
- Description of what was happening when failure occurred
- Any recent configuration changes
- CloudStack and database status

### Data Loss Prevention

**Before making changes**:
```bash
# Always backup first
mysqldump -u oma_user -poma_password migratekit_oma volume_operations device_mappings > backup_before_fix.sql

# Test fixes on copy
mysql -u oma_user -poma_password -e "CREATE DATABASE migratekit_oma_test;"
mysql -u oma_user -poma_password migratekit_oma_test < backup_before_fix.sql
```

This troubleshooting guide covers the most common issues and recovery procedures. For issues not covered here, enable debug logging and examine the detailed logs to understand the failure mode.
