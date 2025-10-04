# VMA Deployment Package - Binary Manifest

**Last Updated:** October 3, 2025 15:26 UTC  
**Status:** ✅ VERIFIED CORRECT BINARIES  

---

## 🎯 Current Binaries

### vma-api-server
- **MD5:** `200fd75e80bc13c14f45427044c1e0e9`
- **Version:** 1.3.2 (from source/current)
- **Build Date:** October 3, 2025
- **Source:** `/home/pgrayson/migratekit-cloudstack/source/current/vma-api-server/main.go`
- **Status:** ✅ VERIFIED WORKING - Contains correct multi-disk NBD target mapping logic
- **Deployed To:** VMA 232 (10.0.100.232) - Tested and working correctly

### migratekit  
- **MD5:** `0a2e773653c47b8923809ee5df6e6ffa`
- **Build Date:** October 2, 2025
- **Status:** ✅ VERIFIED WORKING - Correct migratekit with multi-disk support

---

## 🚨 CRITICAL INCIDENT RESOLVED

**Date:** October 3, 2025  
**Problem:** VMA 232 was causing data corruption on multi-disk VMs

**Root Cause:**  
VMA 232's deployment on Oct 2 received a WRONG/CORRUPTED vma-api-server binary (MD5: `def32662a78fb9a0fb7de29a574ae4d1`) that had buggy disk-to-NBD-target mapping logic. This caused all disks to write to the SAME destination, resulting in the second disk overwriting the first disk's partition table.

**Evidence:**  
Screenshots showed VMA 232 writing `vdc` (10G) partition table to `vdb` (115G), corrupting the larger disk.

**Fix Applied:**  
1. Built correct vma-api-server from `source/current/` (MD5: `200fd75e80bc13c14f45427044c1e0e9`)
2. Deployed to VMA 232 on October 3, 2025 14:18 UTC
3. Verified correct disk mapping in logs: Each disk maps to separate NBD target with proper VMware disk keys (2000, 2001)

**Verification:**  
- VMA API logs show: `"🎯 Added NBD target with VMware disk key"` for each disk separately
- Command line shows: `--nbd-targets 2000:nbd://...vol-101328f1...,2001:nbd://...vol-ec4b3a8d...`
- Test replication of pgtest1 multi-disk VM confirmed working correctly

---

## 📦 Archived Binaries

Old/duplicate binaries moved to:  
`/home/pgrayson/migratekit-cloudstack/source/archive/vma-binaries-old-20251003/`

- `vma-api-server-vma233-working-6a34a934` (MD5: `6a34a93484cd4622fceba73965d7fbc5`) - Older but working version from VMA 233
- `vma-api-server-vma233-duplicate` - Duplicate copy
- `migratekit-duplicate` - Duplicate copy

---

## ✅ Deployment Instructions

**For Future VMA Deployments:**

1. **Use ONLY these binaries** from this directory
2. **Verify MD5 hashes** before deployment:
   ```bash
   md5sum vma-api-server migratekit
   ```
3. **Expected hashes:**
   - vma-api-server: `200fd75e80bc13c14f45427044c1e0e9`
   - migratekit: `0a2e773653c47b8923809ee5df6e6ffa`

4. **Deploy to VMA:**
   ```bash
   scp vma-api-server vma@VMA_IP:/tmp/
   scp migratekit vma@VMA_IP:/tmp/
   ssh vma@VMA_IP "sudo mv /tmp/vma-api-server /opt/vma/bin/ && sudo mv /tmp/migratekit /opt/vma/bin/"
   ```

---

**⚠️ NEVER deploy unverified binaries to production VMAs!**
