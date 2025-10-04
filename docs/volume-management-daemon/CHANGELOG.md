# Volume Management Daemon - Changelog

**Production-ready centralized volume management for MigrateKit CloudStack integration**

---

## [v1.3.2] - 2025-09-26 🔗 PERSISTENT DEVICE NAMING + NBD MEMORY SYNCHRONIZATION

### 🔥 **CRITICAL ENHANCEMENT: Persistent Device Naming System**

**PRODUCTION MILESTONE**: Complete solution for NBD export memory synchronization issues.

#### **Problem Solved**
- **Issue**: NBD server accumulated stale exports in memory after volume operations (failover/failback/deletion)
- **Root Cause**: NBD export names changed during volume lifecycle, leading to memory desynchronization
- **Symptoms**: 
  - Post-failback replication jobs failed with "Access denied by server configuration"
  - NBD server memory contained stale exports pointing to invalid device paths
  - Manual NBD server restarts required to clear memory state

#### **Solution Implemented**
- **Persistent Device Naming**: Stable device names throughout volume lifecycle operations
- **Device Mapper Integration**: Automatic symlink creation (`/dev/mapper/vol[id]` → actual device)
- **NBD Export Stability**: All exports use persistent symlinks, eliminating export churn
- **Database Enhancement**: Added `persistent_device_name` and `symlink_path` tracking fields

#### **Technical Implementation**
```go
// Enhanced Volume Attachment with Persistent Naming:
persistentName := fmt.Sprintf("vol%s", volumeID[0:8])
symlinkPath := "/dev/mapper/" + persistentName

// Create device mapper symlink
dmsetup create persistentName --table "0 deviceSize linear actualDevice 0"

// NBD export uses persistent symlink (stable)
exportname = /dev/mapper/vol3106013a  // Never changes during operations
```

#### **Production Impact**
- **✅ Eliminated**: Post-failback replication failures caused by NBD memory issues
- **✅ Stable Operations**: NBD export names persist throughout volume lifecycle
- **✅ Zero Maintenance**: No manual NBD server restarts required
- **✅ Clear Diagnostics**: Human-readable persistent device names for troubleshooting

#### **Database Schema Changes**
```sql
ALTER TABLE device_mappings 
ADD COLUMN persistent_device_name VARCHAR(255) NULL,
ADD COLUMN symlink_path VARCHAR(255) NULL;
```

#### **System Coverage**
- **Complete Retrofit**: All existing VMs enhanced with persistent naming
- **Automatic Integration**: New volumes get persistent naming automatically
- **Production Validation**: Complete failover/rollback cycles successful with persistent naming

---

## [v1.2.0] - 2025-09-04 🎉 MAJOR BREAKTHROUGH RELEASE

### 🔥 **CRITICAL FIX: Multi-Volume VM Support**

**MASSIVE IMPROVEMENT**: Volume Daemon now reliably handles VMs with multiple disks.

#### **Problem Solved**
- **Issue**: Multi-volume VMs (like QUAD-AUVIK02 with 2 disks) failed consistently
- **Root Cause**: Channel consumption bug in device correlation logic
- **Symptoms**: 
  - First volume attached successfully
  - Second+ volumes timed out with "No fresh device detected during correlation timeout"
  - Logs showed contemporary events being "kept" but then lost

#### **Solution Implemented**
- **Eliminated pre-draining**: Removed `drainStaleDeviceEvents()` phase entirely
- **Direct timestamp filtering**: Skip stale events in correlation loop with `continue`
- **Contemporary window**: Events within 5 seconds of correlation start are accepted
- **No event loss**: All events processed in single correlation loop

#### **Technical Changes**
```go
// BEFORE (BROKEN):
vs.drainStaleDeviceEvents(ctx, correlationStartTime)  // Consumed events!
// ... correlation loop that never found events

// AFTER (WORKING):
// Single correlation loop with direct filtering
if event.Timestamp.Before(contemporaryThreshold) {
    continue // Skip stale, keep looking for fresh
}
return event.DevicePath, event.DeviceInfo.Size // Use immediately
```

#### **Production Validation**
- **Test Case**: QUAD-AUVIK02 VM (37GB + 5GB disks)
- **Result**: Both volumes successfully attached to unique device paths
- **Device Paths**: `/dev/vdc` (disk-0), `/dev/vdd` (disk-1)
- **Database**: Proper correlation tracking for both volumes
- **NBD Exports**: Automatic creation for both volumes
- **User Feedback**: *"Holy shit its working"* ✅

#### **Breaking Changes**
- None - backward compatible
- Removed unused `drainStaleDeviceEvents()` helper function
- Enhanced logging shows new correlation flow

#### **Enhanced Logging**
```
🕐 Starting device correlation with timestamp filtering (no pre-draining)
🚫 Skipping stale device event (>5s before correlation)
✅ Using contemporary/fresh device for correlation
```

#### **Files Changed**
- `internal/volume/service/volume_service.go` - Core correlation logic
- Enhanced error handling and logging

---

## [v1.1.1] - 2025-09-04 (SUPERSEDED)

### **Timing Precision Fix**
- **Issue**: 5-second contemporary window too strict
- **Fix**: Adjusted timing tolerance in drain logic
- **Status**: Partially resolved channel consumption, but root cause remained

---

## [v1.1.0] - 2025-09-04 (SUPERSEDED)

### **Initial Stale Event Fix Attempt**
- **Issue**: Stale events from previous attachments being used
- **Fix**: Added timestamp filtering and drain logic
- **Status**: Introduced channel consumption bug that was later fixed

---

## [v1.0.0] - 2025-09-01

### **Initial Production Release**
- **Features**: Centralized volume management, device correlation, NBD integration
- **Architecture**: Polling-based device monitoring, atomic operations
- **Limitations**: Multi-volume VM support had correlation bugs (fixed in v1.2.0)

---

## **Migration Guide**

### **To v1.2.0 from v1.1.x**
```bash
# 1. Stop service
sudo systemctl stop volume-daemon

# 2. Deploy new binary
sudo cp volume-daemon-v1.2.0-no-drain-fix /usr/local/bin/volume-daemon
sudo chmod +x /usr/local/bin/volume-daemon

# 3. Start service
sudo systemctl start volume-daemon

# 4. Verify deployment
curl -s http://localhost:8090/health | jq '.status'

# 5. Test multi-volume VM
# Start replication for VM with multiple disks
```

### **Verification Commands**
```bash
# Check version
ls -la /usr/local/bin/volume-daemon

# Monitor new correlation logs
journalctl -u volume-daemon -f | grep -E "correlation.*no pre-draining|contemporary.*device"

# Test multi-volume scenario
# Expected: Both volumes attach with unique device paths
```

---

## **Known Issues**

### **Resolved**
- ✅ Multi-volume VM correlation timeouts (v1.2.0)
- ✅ Stale event correlation bugs (v1.2.0)
- ✅ Channel consumption race condition (v1.2.0)

### **Active**
- None currently known

---

## **Future Roadmap**

### **Planned Enhancements**
- Enhanced correlation algorithms with size validation
- High availability with leader election
- Prometheus metrics integration
- Advanced NBD export management
- Container deployment support

### **Performance Improvements**
- Connection pooling optimization
- Correlation timeout tuning
- Memory usage optimization
- Background worker scaling

---

## **Support Information**

### **Documentation**
- `README.md` - Complete setup and usage guide
- `ARCHITECTURE.md` - Technical implementation details (updated with v1.2.0 fix)
- `TROUBLESHOOTING.md` - Diagnostic procedures (includes multi-volume section)
- `API_REFERENCE.md` - Complete API documentation
- `INTEGRATION_GUIDE.md` - Service integration procedures

### **Logging**
```bash
# Service logs
journalctl -u volume-daemon -f

# Correlation debugging
journalctl -u volume-daemon | grep -E "correlation|device.*detected"

# Health monitoring
curl -s http://localhost:8090/api/v1/health | jq
```

### **Contact**
- Technical issues: Check TROUBLESHOOTING.md first
- Bug reports: Include logs and reproduction steps
- Feature requests: Describe use case and requirements

---

**Volume Management Daemon v1.2.0** - *The multi-volume breakthrough release* 🚀
