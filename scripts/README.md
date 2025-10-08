# Sendense Scripts

**Location:** `/home/oma_admin/sendense/scripts/`  
**Purpose:** Utility scripts for Sendense platform operations and maintenance  
**Last Updated:** October 8, 2025

---

## 📜 Available Scripts

### **cleanup-backup-environment.sh**

**Purpose:** Clean all backup-related processes and files before testing

**Usage:**
```bash
./scripts/cleanup-backup-environment.sh
```

**What It Does:**
1. **Kills all qemu-nbd processes** - Stops any running NBD servers
2. **Deletes all QCOW2 files** - Removes backup files from /backup/repository/
3. **Kills SNA backup processes** - Stops sendense-backup-client on remote SNA
4. **Checks for file locks** - Verifies no QCOW2 files are locked
5. **Restarts SHA service** - Clears port allocations (if systemd service exists)
6. **Verifies cleanup** - Reports final environment state

**When to Use:**
- Before running backup tests
- After failed backup operations
- When cleaning up corrupted test environment
- When qemu-nbd processes are orphaned

**Exit Codes:**
- `0` - Cleanup successful (or completed with warnings)
- Color-coded output indicates issues (red = errors, yellow = warnings)

**Requirements:**
- sudo access (for pkill, lsof)
- SSH access to SNA (vma@10.0.100.231) with password authentication
- /backup/repository/ directory exists

**Example Output:**
```
==========================================
🧹 Sendense Backup Environment Cleanup
==========================================

Step 1: Killing qemu-nbd processes...
  Found 2 qemu-nbd processes
  ✅ All qemu-nbd processes killed

Step 2: Deleting QCOW2 files from /backup/repository/...
  Found 2 QCOW2 files
  ✅ All QCOW2 files deleted

Step 3: Killing sendense-backup-client processes on SNA...
  SNA accessible at 10.0.100.231
  ✅ Sent kill signal to SNA backup processes

Step 4: Checking for QCOW2 file locks...
  ✅ No QCOW2 file locks detected

Step 5: Restarting sendense-hub service...
  ⚠️  sendense-hub service not found or not active

Step 6: Final verification...
  Process verification:
    - qemu-nbd processes: 0
  File verification:
    - QCOW2 files: 0
  Service verification:
    - sendense-hub: inactive

==========================================
🎉 Environment cleanup completed successfully
✅ Ready for backup testing
==========================================
```

**Notes:**
- If sendense-hub is running manually (not as systemd service), you'll see a warning in Step 5
- This is normal for development environments
- Manual SHA restart: Kill existing process and restart with desired flags
- Script will still report success even with warnings

**Troubleshooting:**
- If qemu-nbd processes won't die: Check for hung NBD connections
- If QCOW2 files won't delete: Check file locks with `sudo lsof | grep qcow2`
- If SNA unreachable: Check SSH tunnel and network connectivity
- If file locks persist: May need to unmount filesystems or restart system

---

## 🔧 Script Maintenance

### **Adding New Scripts**

When adding scripts to this directory:

1. **Create the script** with proper shebang (`#!/bin/bash`)
2. **Make executable:** `chmod +x scripts/new-script.sh`
3. **Add documentation** to this README.md
4. **Test thoroughly** before committing
5. **Update CHANGELOG.md** with script addition

### **Script Standards**

All scripts in this directory should follow these standards:

- ✅ Use `set -e` for error handling
- ✅ Provide color-coded output (GREEN=success, RED=error, YELLOW=warning)
- ✅ Include comprehensive logging
- ✅ Verify operations completed successfully
- ✅ Exit with appropriate exit codes
- ✅ Include usage documentation in script header
- ✅ Handle errors gracefully

---

## 📊 Script Testing Checklist

Before committing any script:

- [ ] Script runs without errors
- [ ] All operations complete successfully
- [ ] Error handling works correctly
- [ ] Output is clear and informative
- [ ] Exit codes are correct
- [ ] Documentation is complete
- [ ] Script is executable (`chmod +x`)

---

## 🚀 Quick Reference

### **Backup Testing Workflow**

1. **Clean environment:** `./scripts/cleanup-backup-environment.sh`
2. **Start backup test:** Use SHA API to initiate backup
3. **Monitor progress:** Check SHA logs and QCOW2 file growth
4. **Verify completion:** Check backup_jobs table in database
5. **Clean up:** Run cleanup script again if needed

---

**Last Updated:** October 8, 2025  
**Maintainer:** Sendense Engineering Team  
**Related:** Phase 1 VMware Backup Completion