# GROK URGENT FIX: Partition Path Mismatch

## **CRITICAL BUG - 500 ERROR WHEN BROWSING PARTITIONS**

### **Problem:**
Backend creates partition folders as: `/partition-1`, `/partition-2`, `/partition-3`  
But `listPartitionFolders()` is setting the `Path` field to friendly names like:
- `"/Partition 3 (100.4 GB)"`  
- `"/Partition 1 - New\x20Volume (1.5 GB)"`

When GUI clicks these, backend path validation fails because these paths don't exist.

### **Error from logs:**
```
time="2025-10-09T16:22:04+01:00" level=error msg="List files failed" 
error="path validation failed: path does not exist: /Partition 3 (100.4 GB)"
```

---

## **THE FIX:**

**File:** `source/current/sha/restore/file_browser.go`

**Function:** `listPartitionFolders()` (around line 452)

**Current broken code:**
```go
fileInfo := &FileInfo{
    Name:         name,  // "Partition 3 (100.4 GB)"
    Path:         fmt.Sprintf("/partition-%d", partitionNum),  // "/partition-3"
    Type:         "directory",
    // ...
}
```

Wait... actually looking at your diff, the code ALREADY sets `Path` correctly to `/partition-N`.

The bug must be in `listFilesInPartition()` - it's NOT properly handling the path conversion.

---

## **ACTUAL ROOT CAUSE:**

Looking at the logs, when GUI sends `"/Partition 3 (100.4 GB)"`, the backend is trying to validate that as a filesystem path instead of recognizing it's a virtual partition folder name.

**The issue is:** `listFilesInPartition()` is receiving the **display name** from GUI but needs to extract the **partition number** to map to the actual `/partition-N` folder.

---

## **CORRECT FIX:**

**File:** `source/current/sha/restore/file_browser.go`

**In function:** `ListFiles()` around line 119

**Current logic:**
```go
} else if strings.HasPrefix(requestPath, "/partition-") {
    // PARTITION PATH: List files within partition
    files, err = fb.listFilesInPartition(mount, requestPath, req.Recursive)
```

**Problem:** This check only works if the path is `/partition-1`, but GUI is sending `"/Partition 3 (100.4 GB)"`.

**Solution:** We need to parse the partition NUMBER from the display name and convert it to the actual path.

---

## **STEP-BY-STEP FIX:**

### **Step 1: Update `ListFiles()` logic**

Replace this section (around lines 105-134):

```go
// Handle multi-partition browsing
var files []*FileInfo
if len(partitionMetadata) > 0 && requestPath == "/" {
    // ROOT PATH: Show partition folders as virtual directories
    files = fb.listPartitionFolders(mount, partitionMetadata)
} else if strings.HasPrefix(requestPath, "/partition-") {
    // PARTITION PATH: List files within partition
    files, err = fb.listFilesInPartition(mount, requestPath, req.Recursive)
    if err != nil {
        return nil, fmt.Errorf("failed to list files in partition: %w", err)
    }
}
```

With:

```go
// Handle multi-partition browsing
var files []*FileInfo
if len(partitionMetadata) > 0 && requestPath == "/" {
    // ROOT PATH: Show partition folders as virtual directories
    files = fb.listPartitionFolders(mount, partitionMetadata)
} else if len(partitionMetadata) > 0 && strings.HasPrefix(requestPath, "/Partition") {
    // GUI sent display name like "/Partition 3 (100.4 GB)" - extract partition number
    // and convert to actual path "/partition-3"
    partitionNum := fb.extractPartitionNumber(requestPath)
    if partitionNum > 0 {
        actualPath := fmt.Sprintf("/partition-%d", partitionNum)
        files, err = fb.listFilesInPartition(mount, actualPath, req.Recursive)
        if err != nil {
            return nil, fmt.Errorf("failed to list files in partition: %w", err)
        }
    } else {
        return nil, fmt.Errorf("invalid partition path: %s", requestPath)
    }
} else if strings.HasPrefix(requestPath, "/partition-") {
    // Direct partition path (for API calls)
    files, err = fb.listFilesInPartition(mount, requestPath, req.Recursive)
    if err != nil {
        return nil, fmt.Errorf("failed to list files in partition: %w", err)
    }
}
```

### **Step 2: Add helper function**

Add this new function after `listPartitionFolders()`:

```go
// extractPartitionNumber extracts partition number from display name
// Example: "/Partition 3 (100.4 GB)" -> 3
// Example: "/Partition 1 - New Volume (1.5 GB)" -> 1
func (fb *FileBrowser) extractPartitionNumber(displayPath string) int {
	// Remove leading slash
	displayPath = strings.TrimPrefix(displayPath, "/")
	
	// Extract number after "Partition "
	if !strings.HasPrefix(displayPath, "Partition ") {
		return 0
	}
	
	// Remove "Partition " prefix
	remaining := strings.TrimPrefix(displayPath, "Partition ")
	
	// Extract first number
	var numStr string
	for _, ch := range remaining {
		if ch >= '0' && ch <= '9' {
			numStr += string(ch)
		} else {
			break
		}
	}
	
	if numStr == "" {
		return 0
	}
	
	partitionNum, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}
	
	return partitionNum
}
```

### **Step 3: Add missing import**

At the top of the file, ensure `strconv` is imported:

```go
import (
	// ... existing imports ...
	"strconv"
)
```

---

## **TESTING COMMANDS:**

After deployment, test:

```bash
# Check mount structure
ls -la /mnt/sendense/restore/e6e1dd09-05dd-4ba8-8344-cd885a07b7c5/

# Should show:
# partition-1/
# partition-2/
# partition-3/

# Test listing partition contents
curl -s "http://localhost:8082/api/v1/restore/e6e1dd09-05dd-4ba8-8344-cd885a07b7c5/files?path=/partition-1" | jq .

# GUI should now work - click partition folders and browse contents
```

---

## **FILES TO MODIFY:**

1. `source/current/sha/restore/file_browser.go` - Add helper function + update ListFiles logic

---

## **ACCEPTANCE CRITERIA:**

✅ Click "Partition 1 - New Volume (1.5 GB)" → Shows partition contents  
✅ Click "Partition 3 (100.4 GB)" → Shows Windows OS files  
✅ No 500 errors in browser console  
✅ Backend logs show successful file listings  

---

## **CRITICAL RULES:**

- DO NOT change the actual filesystem paths (`/partition-N`)
- DO NOT change how partitions are mounted
- ONLY fix the path parsing/mapping logic
- Backend must handle BOTH formats: display names from GUI and direct `/partition-N` paths


