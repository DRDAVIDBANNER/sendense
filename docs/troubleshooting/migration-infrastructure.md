# Migration Infrastructure Troubleshooting Guide

**Last Updated**: 2025-08-07  
**Status**: Infrastructure Operational  

## 🎯 **Quick Health Check**

### **1. Service Status Check**
```bash
# On OMA (10.245.246.125)
sudo systemctl status oma-api
sudo systemctl status migratekit-gui

# On VMA (10.0.100.231)  
ssh pgrayson@10.0.100.231 "sudo systemctl status vma-api"
ssh pgrayson@10.0.100.231 "sudo systemctl status vma-tunnel-enhanced"
```

### **2. SSH Tunnel Verification**
```bash
# From OMA - should return VMA health status
curl --connect-timeout 5 --max-time 10 -s http://localhost:9081/api/v1/health
# Expected: {"status":"healthy","timestamp":"...","uptime":"..."}
```

### **3. NBD Server Check**
```bash
# Check running NBD servers
ps aux | grep nbd-server | grep dynamic
# Should show multiple nbd-server processes with dynamic configs
```

## 🚨 **Common Issues & Solutions**

### Concurrent jobs collide on port 10808
- Symptom: Second job kills or conflicts with an existing bridge; migratekit connections flap
- Cause: A single global bridge on 10808 for all jobs
- Fix: Use per-job stunnel local ports and pass `NBD_LOCAL_PORT` to migratekit; no global 10808 bridge

### stunnel: Address already in use (per-job accept port)
- Symptom: `stunnel ... Address already in use (98)` in VMA logs
- Causes:
  - Port allocator race or lingering stunnel process
  - Port actually bound by another process
- Fixes:
  - Ensure allocator verifies port with a bind probe (implemented)
  - Kill stunnel by PID and retry a new port

### **SSH Tunnel Issues**

#### **Problem**: `curl localhost:9081` fails with connection refused
```bash
# Symptoms
curl: (7) Failed to connect to localhost port 9081: Connection refused
```

**Solution**:
```bash
# 1. Check for port conflicts
sudo ss -tlnp | grep 9081

# 2. Kill conflicting processes
sudo kill <PID_USING_PORT_9081>

# 3. Restart VMA tunnel service
ssh pgrayson@10.0.100.231 "sudo systemctl restart vma-tunnel-enhanced"

# 4. Verify tunnel working
curl --connect-timeout 5 -s http://localhost:9081/api/v1/health
```

#### **Problem**: VMA logs show "remote port forwarding failed"
```bash
# VMA logs show
Warning: remote port forwarding failed for listen port 9081
```

**Root Cause**: Port 9081 already in use on OMA side  
**Solution**: Same as above - kill processes using port 9081

### **Volume Attachment Issues**

#### **Problem**: "No more available PCI slots" error
```bash
# OMA logs show
Failed to attach volume... No more available PCI slots
```

**Solution**:
```bash
# 1. List attached volumes
curl -s http://localhost:8082/api/v1/ossea/volumes | head -20

# 2. Manually detach old volumes via CloudStack UI or API
# 3. Or restart OMA VM to reset PCI slots (if safe to do)

# 4. Retry migration job
```

### **NBD Server Issues**

#### **Problem**: NBD server not starting
```bash
# Check NBD server processes
ps aux | grep nbd-server | grep dynamic
# No processes found
```

**Solution**:
```bash
# 1. Check NBD configuration directory permissions
ls -la /opt/migratekit/nbd-configs/
sudo chown -R oma-api:oma-api /opt/migratekit/nbd-configs/

# 2. Check systemd service permissions
sudo systemctl cat oma-api | grep ReadWritePaths
# Should include /opt/migratekit/nbd-configs

# 3. Restart OMA API service
sudo systemctl restart oma-api
```

#### **Problem**: NBD configuration files not created
**Root Cause**: Systemd ProtectSystem=strict blocking writes  
**Solution**: Already fixed - ReadWritePaths includes nbd-configs directory

### **VMA-OMA Communication Issues**

#### **Problem**: VMA logs show "failed to call VMA API: connection refused"
```bash
# VMA logs show
failed to call VMA API: Post "http://localhost:8080/api/v1/replications": dial tcp 127.0.0.1:8080: connect: connection refused
```

**Root Cause**: VMA trying to call wrong port (8080 instead of 8082)  
**Solution**: Already fixed in codebase - VMA now uses port 8082

#### **Problem**: VMA shows "No NBD target provided for VM"
**Root Cause**: NBD target format mismatch  
**Solution**: Already fixed - OMA now sends NBD URLs instead of device paths

### **Migratekit Execution Issues**

#### **Problem**: Migratekit exits with status 1
```bash
# VMA logs show
level=info msg="Migratekit process completed" error="exit status 1"
```

**Status**: Known issue tracked in BUG-001  
**Investigation Steps**:
```bash
# 1. Manual test on VMA
ssh pgrayson@10.0.100.231
cd /home/pgrayson/migratekit-cloudstack
./migratekit-tls-tunnel migrate --help

# 2. Test NBD connectivity
nbd-client 10.245.246.125 10811 /tmp/test-nbd -name <export-name>

# 3. Check environment variables
env | grep -i cloudstack
env | grep -i ossea
```

## 🔍 **Diagnostic Commands**

### **Infrastructure Health Check**
```bash
#!/bin/bash
echo "=== MigrateKit Infrastructure Health Check ==="

echo "1. SSH Tunnel Status:"
curl --connect-timeout 3 --max-time 5 -s http://localhost:9081/api/v1/health || echo "❌ SSH tunnel failed"

echo -e "\n2. NBD Servers:"
ps aux | grep nbd-server | grep dynamic | wc -l | xargs echo "Active NBD servers:"

echo -e "\n3. Service Status:"
sudo systemctl is-active oma-api
sudo systemctl is-active migratekit-gui

echo -e "\n4. Recent Migration Jobs:"
curl -s http://localhost:8082/api/v1/replications | head -5

echo -e "\n5. Available PCI Slots (volume attachments):"
ls /dev/vd* | wc -l | xargs echo "Attached volumes:"
```

### **Log Analysis**
```bash
# OMA API logs (last 20 lines)
sudo journalctl -u oma-api --no-pager | tail -20

# VMA API logs (last 20 lines) 
ssh pgrayson@10.0.100.231 "sudo journalctl -u vma-api --no-pager | tail -20"

# NBD configuration files
ls -la /opt/migratekit/nbd-configs/
```

### **Network Connectivity**
```bash
# Test VMA connectivity
ping -c 3 10.0.100.231

# Test OMA API from VMA
ssh pgrayson@10.0.100.231 "curl --connect-timeout 5 -s http://localhost:8082/api/v1/health"

# Test NBD port connectivity from VMA
ssh pgrayson@10.0.100.231 "telnet 10.245.246.125 10811"
```

## 📊 **Performance Monitoring**

### **Response Time Benchmarks**
```bash
# OMA API response time
time curl -s http://localhost:8082/api/v1/health

# VMA API response time (via tunnel)
time curl -s http://localhost:9081/api/v1/health

# Migration job creation time
time curl -X POST http://localhost:8082/api/v1/replications -H "Content-Type: application/json" -d '{...}'
```

### **Resource Usage**
```bash
# Memory usage
ps aux | grep -E "(oma-api|nbd-server)" | awk '{sum+=$6} END {print "Memory usage:", sum/1024, "MB"}'

# Disk usage (NBD configs)
du -sh /opt/migratekit/nbd-configs/

# Network connections
sudo ss -tlnp | grep -E "(8082|9081|108[0-9][0-9])"
```

## 🛠️ **Maintenance Tasks**

### **Regular Cleanup**
```bash
# Clean up old NBD configurations (older than 1 day)
find /opt/migratekit/nbd-configs/ -name "config-dynamic-*" -mtime +1 -delete

# Clean up completed migration logs
find /home/pgrayson/migratekit-cloudstack/ -name "migration-*.log" -mtime +7 -delete

# Restart services weekly (if needed)
sudo systemctl restart oma-api
ssh pgrayson@10.0.100.231 "sudo systemctl restart vma-api"
```

### **Backup Important Configs**
```bash
# Backup systemd service files
sudo cp /etc/systemd/system/oma-api.service /backup/
sudo cp /etc/systemd/system/migratekit-gui.service /backup/

# Backup database (if applicable)
sudo mysqldump migratekit > /backup/migratekit-$(date +%Y%m%d).sql
```

---

## 📞 **Escalation Path**

1. **Infrastructure Issues**: Check this troubleshooting guide
2. **Service Failures**: Restart services and check logs  
3. **Persistent Problems**: Review recent code changes
4. **Migratekit Issues**: Refer to BUG-001 for current investigation status

**Remember**: The migration infrastructure is operational. Most issues are configuration or connectivity related and can be resolved with the solutions above.
