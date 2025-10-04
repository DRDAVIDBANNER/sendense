# VMA Deployment Package - Binary Manifest

**Last Updated:** October 3, 2025 16:43 UTC  
**Status:** ✅ PRODUCTION READY - PARALLEL NBD IMPLEMENTATION  

---

## 🎯 Current Binaries

### vma-api-server
- **MD5:** `200fd75e80bc13c14f45427044c1e0e9`
- **Version:** 1.3.2 (from source/current)
- **Build Date:** October 3, 2025
- **Source:** `/home/pgrayson/migratekit-cloudstack/source/current/vma-api-server/main.go`
- **Status:** ✅ VERIFIED WORKING - Contains correct multi-disk NBD target mapping logic
- **Deployed To:** VMA 232 (10.0.100.232), VMA 233 (10.0.100.233) - Tested and working correctly

### migratekit  
- **MD5:** `918b3b312c606cc3d3f3d50b7197b964`
- **Version:** v2.22.2-parallel-nbd-chunk-limit-fix
- **Build Date:** October 3, 2025 17:47 UTC
- **Status:** ✅ **PRODUCTION READY** - Parallel NBD with all fixes
- **Features:**
  - Parallel NBD workers for full and incremental copies (3-4 workers per disk)
  - Extent coalescing for reduced request overhead
  - Intelligent worker distribution with 512-byte alignment
  - Per-worker progress tracking and retry logic
  - Sparse optimization maintained across all workers
  - **FIXED:** Completion status sent to VMA even when zero changed blocks (prevents frontend timeout)
  - **FIXED:** Large extent chunking to respect NBD 32 MB limit (handles VMware CBT large extents)
- **Performance:** 260-350 MB/s aggregate throughput (vs 70-80 MB/s serial)
- **Tested:** VMA 233 (10.0.100.233) - QCLOUD-JUMP04 (incremental), pgtest3 (full copy), pgtest1 (zero-extent)
- **Results:** 
  - Incremental: 1.7-1.9x improvement (11.8 MB/s aggregate)
  - Full Copy: 3.5-4.5x improvement (260-350 MB/s aggregate)
  - Zero-extent: Proper completion status (100%) sent to frontend
- **Deployed To:** VMA 233 (10.0.100.233) - Production validated

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
