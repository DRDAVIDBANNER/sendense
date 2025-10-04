# VMA Tunnel OMA Switching Procedure

**Last Updated**: September 28, 2025  
**Purpose**: Document exact steps to switch VMA tunnel connection from one OMA to another  
**Status**: ✅ **PRODUCTION TESTED** - Successfully switched from dev OMA (10.245.246.125) to remote OMA (45.130.45.65)

---

## 🎯 **OVERVIEW**

The VMA (VMware Migration Appliance) uses an enhanced SSH tunnel service to connect to the OMA (OSSEA Migration Appliance). This tunnel provides:
- **Reverse Control Channel**: OMA can reach VMA API via `localhost:9081` → VMA `localhost:8081`
- **Forward Data Channel**: VMA can reach OMA API (if needed)
- **Health Monitoring**: Automatic tunnel recovery and keep-alive

---

## 📋 **SWITCHING PROCEDURE**

### **Prerequisites**
- SSH access to VMA: `ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231`
- SSH key for target OMA server (e.g., `~/.ssh/oma-server-key` for remote servers)
- Target OMA server details (IP address, SSH user)

### **Step 1: Stop Current Tunnel Service**
```bash
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo systemctl stop vma-tunnel-enhanced-v2.service"
```

### **Step 2: Update Service Configuration**
Update the tunnel service environment variables:

```bash
# For Dev OMA (10.245.246.125)
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's/Environment=OMA_HOST=.*/Environment=OMA_HOST=10.245.246.125/' /etc/systemd/system/vma-tunnel-enhanced-v2.service"
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's|Environment=SSH_KEY=.*|Environment=SSH_KEY=/home/pgrayson/.ssh/cloudstack_key|' /etc/systemd/system/vma-tunnel-enhanced-v2.service"

# For Remote OMA (45.130.45.65)  
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's/Environment=OMA_HOST=.*/Environment=OMA_HOST=45.130.45.65/' /etc/systemd/system/vma-tunnel-enhanced-v2.service"
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's|Environment=SSH_KEY=.*|Environment=SSH_KEY=/home/pgrayson/.ssh/oma-server-key|' /etc/systemd/system/vma-tunnel-enhanced-v2.service"
```

### **Step 3: Update Tunnel Script for SSH User**
The tunnel script must use the correct SSH user for the target OMA:

**For Dev OMA (user: pgrayson):**
```bash
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's|ExecStart=.*|ExecStart=/home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel.sh|' /etc/systemd/system/vma-tunnel-enhanced-v2.service"
```

**For Remote OMA (user: oma):**
```bash
# Create remote-specific tunnel script if it doesn't exist
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sed 's/pgrayson@\$OMA_HOST/oma@\$OMA_HOST/g' /home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel.sh > /home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel-remote.sh && chmod +x /home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel-remote.sh"

# Update service to use remote script
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's|ExecStart=.*|ExecStart=/home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel-remote.sh|' /etc/systemd/system/vma-tunnel-enhanced-v2.service"
```

### **Step 4: Update VMA API Configuration**
Update VMA API service to point to new OMA:

```bash
# For Dev OMA
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's/Environment=OMA_NBD_HOST=.*/Environment=OMA_NBD_HOST=10.245.246.125/' /etc/systemd/system/vma-api.service.d/override.conf"

# For Remote OMA
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's/Environment=OMA_NBD_HOST=.*/Environment=OMA_NBD_HOST=45.130.45.65/' /etc/systemd/system/vma-api.service.d/override.conf"
```

### **Step 5: Update Stunnel Configuration (Optional)**
If using stunnel for NBD data channel:

```bash
# For Dev OMA
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's/connect = .*/connect = 10.245.246.125:443/' /etc/stunnel/nbd-client-bidirectional.conf"

# For Remote OMA
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo sed -i 's/connect = .*/connect = 45.130.45.65:443/' /etc/stunnel/nbd-client-bidirectional.conf"
```

### **Step 6: Restart Services**
```bash
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo systemctl daemon-reload"
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo systemctl restart vma-api.service"
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo systemctl start vma-tunnel-enhanced-v2.service"
```

### **Step 7: Verify Connection**
```bash
# Check tunnel service status
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "sudo systemctl status vma-tunnel-enhanced-v2.service --no-pager | tail -10"

# Test reverse tunnel from target OMA
ssh -i ~/.ssh/target-oma-key user@target-oma-ip "curl -s http://localhost:9081/api/v1/health"
```

---

## 🔧 **CONFIGURATION FILES REFERENCE**

### **Key Files Modified:**
1. **`/etc/systemd/system/vma-tunnel-enhanced-v2.service`** - Main tunnel service
2. **`/etc/systemd/system/vma-api.service.d/override.conf`** - VMA API OMA endpoint
3. **`/etc/stunnel/nbd-client-bidirectional.conf`** - Stunnel NBD data channel
4. **`/home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel-remote.sh`** - Remote OMA tunnel script

### **SSH Keys:**
- **Dev OMA**: `/home/pgrayson/.ssh/cloudstack_key` → `pgrayson@10.245.246.125`
- **Remote OMA**: `/home/pgrayson/.ssh/oma-server-key` → `oma@45.130.45.65`

### **Service Dependencies:**
- **`vma-tunnel-enhanced-v2.service`** - Main tunnel service
- **`vma-api.service`** - VMA API server (depends on tunnel)

---

## ✅ **VALIDATION CHECKLIST**

After switching, verify:
- [ ] Tunnel service shows "✅ SSH tunnel established and verified"
- [ ] Tunnel service shows "🔄 Tunnel established, starting health monitoring"
- [ ] Target OMA can reach VMA API: `curl http://localhost:9081/api/v1/health`
- [ ] VMA API service is running and healthy
- [ ] Discovery from target OMA works properly

---

## 🚨 **COMMON ISSUES**

### **Permission Denied (publickey,password)**
- **Cause**: Wrong SSH key or SSH user
- **Fix**: Verify SSH key path and ensure tunnel script uses correct user

### **Connection Refused**
- **Cause**: Target OMA not accepting connections on port 22
- **Fix**: Verify target OMA SSH service and firewall settings

### **Tunnel Keeps Restarting**
- **Cause**: Health check failing or SSH connection unstable
- **Fix**: Check SSH connectivity and target OMA port availability

---

## 📝 **EXAMPLE: Complete Switch from Dev to Remote**

```bash
# Complete procedure to switch from dev (10.245.246.125) to remote (45.130.45.65)
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 "
sudo systemctl stop vma-tunnel-enhanced-v2.service &&
sudo sed -i 's/Environment=OMA_HOST=.*/Environment=OMA_HOST=45.130.45.65/' /etc/systemd/system/vma-tunnel-enhanced-v2.service &&
sudo sed -i 's|Environment=SSH_KEY=.*|Environment=SSH_KEY=/home/pgrayson/.ssh/oma-server-key|' /etc/systemd/system/vma-tunnel-enhanced-v2.service &&
sudo sed -i 's|ExecStart=.*|ExecStart=/home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel-remote.sh|' /etc/systemd/system/vma-tunnel-enhanced-v2.service &&
sudo sed -i 's/Environment=OMA_NBD_HOST=.*/Environment=OMA_NBD_HOST=45.130.45.65/' /etc/systemd/system/vma-api.service.d/override.conf &&
sudo sed -i 's/connect = .*/connect = 45.130.45.65:443/' /etc/stunnel/nbd-client-bidirectional.conf &&
sudo systemctl daemon-reload &&
sudo systemctl restart vma-api.service &&
sudo systemctl start vma-tunnel-enhanced-v2.service
"
```

---

**🎯 This procedure ensures clean switching between OMA environments with proper SSH authentication and tunnel configuration.**






