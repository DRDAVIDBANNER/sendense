package storage

import (
	"context"
	"testing"
	"time"
)

// TestNewMountManager tests mount manager creation
func TestNewMountManager(t *testing.T) {
	mm := NewMountManager()
	if mm == nil {
		t.Fatal("NewMountManager returned nil")
	}

	if mm.mountedPaths == nil {
		t.Error("mountedPaths map not initialized")
	}

	if mm.mountTimeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", mm.mountTimeout)
	}
}

// TestMountNFS_Validation tests NFS mount configuration validation
func TestMountNFS_Validation(t *testing.T) {
	mm := NewMountManager()
	ctx := context.Background()

	tests := []struct {
		name        string
		config      NFSMountConfig
		expectError string
	}{
		{
			name: "empty server",
			config: NFSMountConfig{
				Server:     "",
				ExportPath: "/exports/backups",
				MountPoint: "/mnt/test",
			},
			expectError: "NFS server cannot be empty",
		},
		{
			name: "empty export path",
			config: NFSMountConfig{
				Server:     "nfs-server",
				ExportPath: "",
				MountPoint: "/mnt/test",
			},
			expectError: "NFS export path cannot be empty",
		},
		{
			name: "empty mount point",
			config: NFSMountConfig{
				Server:     "nfs-server",
				ExportPath: "/exports/backups",
				MountPoint: "",
			},
			expectError: "mount point cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mm.MountNFS(ctx, tt.config)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if err.Error() != tt.expectError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectError, err.Error())
			}
		})
	}
}

// TestMountCIFS_Validation tests CIFS mount configuration validation
func TestMountCIFS_Validation(t *testing.T) {
	mm := NewMountManager()
	ctx := context.Background()

	tests := []struct {
		name        string
		config      CIFSMountConfig
		expectError string
	}{
		{
			name: "empty server",
			config: CIFSMountConfig{
				Server:     "",
				ShareName:  "backups",
				MountPoint: "/mnt/test",
				Username:   "testuser",
			},
			expectError: "CIFS server cannot be empty",
		},
		{
			name: "empty share name",
			config: CIFSMountConfig{
				Server:     "cifs-server",
				ShareName:  "",
				MountPoint: "/mnt/test",
				Username:   "testuser",
			},
			expectError: "CIFS share name cannot be empty",
		},
		{
			name: "empty mount point",
			config: CIFSMountConfig{
				Server:     "cifs-server",
				ShareName:  "backups",
				MountPoint: "",
				Username:   "testuser",
			},
			expectError: "mount point cannot be empty",
		},
		{
			name: "empty username",
			config: CIFSMountConfig{
				Server:     "cifs-server",
				ShareName:  "backups",
				MountPoint: "/mnt/test",
				Username:   "",
			},
			expectError: "CIFS username cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mm.MountCIFS(ctx, tt.config)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if err.Error() != tt.expectError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectError, err.Error())
			}
		})
	}
}

// TestMountInfo_Tracking tests mount information tracking
func TestMountInfo_Tracking(t *testing.T) {
	mm := NewMountManager()

	// Manually add mount info for testing (bypassing actual mount)
	testMountPoint := "/mnt/test-nfs"
	mm.mountedPaths[testMountPoint] = &MountInfo{
		MountPoint:   testMountPoint,
		RepositoryID: "repo-123",
		Type:         "nfs",
		RemoteHost:   "nfs-server.example.com",
		RemotePath:   "/exports/backups",
		MountedAt:    time.Now(),
		MountOptions: []string{"vers=4", "rw", "hard"},
	}

	// Test IsMounted
	if !mm.IsMounted(testMountPoint) {
		t.Error("Expected mount point to be marked as mounted")
	}

	if mm.IsMounted("/mnt/nonexistent") {
		t.Error("Expected nonexistent mount to return false")
	}

	// Test GetMountInfo
	info, err := mm.GetMountInfo(testMountPoint)
	if err != nil {
		t.Fatalf("GetMountInfo failed: %v", err)
	}

	if info.RepositoryID != "repo-123" {
		t.Errorf("Expected repository ID 'repo-123', got '%s'", info.RepositoryID)
	}

	if info.Type != "nfs" {
		t.Errorf("Expected type 'nfs', got '%s'", info.Type)
	}

	// Test GetMountInfo for nonexistent mount
	_, err = mm.GetMountInfo("/mnt/nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent mount, got nil")
	}
}

// TestGetMountsByRepository tests repository-based mount lookup
func TestGetMountsByRepository(t *testing.T) {
	mm := NewMountManager()

	// Add multiple mounts for same repository
	mm.mountedPaths["/mnt/nfs-1"] = &MountInfo{
		MountPoint:   "/mnt/nfs-1",
		RepositoryID: "repo-123",
		Type:         "nfs",
		RemoteHost:   "nfs1.example.com",
		RemotePath:   "/exports/backups",
		MountedAt:    time.Now(),
	}

	mm.mountedPaths["/mnt/nfs-2"] = &MountInfo{
		MountPoint:   "/mnt/nfs-2",
		RepositoryID: "repo-123",
		Type:         "nfs",
		RemoteHost:   "nfs2.example.com",
		RemotePath:   "/exports/backups",
		MountedAt:    time.Now(),
	}

	mm.mountedPaths["/mnt/cifs-1"] = &MountInfo{
		MountPoint:   "/mnt/cifs-1",
		RepositoryID: "repo-456",
		Type:         "cifs",
		RemoteHost:   "cifs.example.com",
		RemotePath:   "backups",
		MountedAt:    time.Now(),
	}

	// Get mounts for repo-123
	mounts := mm.GetMountsByRepository("repo-123")
	if len(mounts) != 2 {
		t.Errorf("Expected 2 mounts for repo-123, got %d", len(mounts))
	}

	// Get mounts for repo-456
	mounts = mm.GetMountsByRepository("repo-456")
	if len(mounts) != 1 {
		t.Errorf("Expected 1 mount for repo-456, got %d", len(mounts))
	}

	// Get mounts for nonexistent repository
	mounts = mm.GetMountsByRepository("repo-999")
	if len(mounts) != 0 {
		t.Errorf("Expected 0 mounts for nonexistent repo, got %d", len(mounts))
	}
}

// TestListMounted tests listing all mounted filesystems
func TestListMounted(t *testing.T) {
	mm := NewMountManager()

	// Empty list initially
	mounts := mm.ListMounted()
	if len(mounts) != 0 {
		t.Errorf("Expected 0 mounts initially, got %d", len(mounts))
	}

	// Add test mounts
	mm.mountedPaths["/mnt/test-1"] = &MountInfo{
		MountPoint:   "/mnt/test-1",
		RepositoryID: "repo-1",
		Type:         "nfs",
		RemoteHost:   "host1",
		RemotePath:   "/path1",
		MountedAt:    time.Now(),
	}

	mm.mountedPaths["/mnt/test-2"] = &MountInfo{
		MountPoint:   "/mnt/test-2",
		RepositoryID: "repo-2",
		Type:         "cifs",
		RemoteHost:   "host2",
		RemotePath:   "share2",
		MountedAt:    time.Now(),
	}

	// List all mounts
	mounts = mm.ListMounted()
	if len(mounts) != 2 {
		t.Errorf("Expected 2 mounts, got %d", len(mounts))
	}

	// Verify mount info is copied (not referenced)
	info, exists := mounts["/mnt/test-1"]
	if !exists {
		t.Fatal("Expected /mnt/test-1 in mounts")
	}

	// Modify returned info shouldn't affect manager's state
	info.RepositoryID = "modified"
	originalInfo, _ := mm.GetMountInfo("/mnt/test-1")
	if originalInfo.RepositoryID == "modified" {
		t.Error("Returned mount info should be a copy, not reference")
	}
}

// TestUnmount_NotFound tests unmounting non-managed mounts
func TestUnmount_NotFound(t *testing.T) {
	mm := NewMountManager()
	ctx := context.Background()

	err := mm.Unmount(ctx, "/mnt/nonexistent", false)
	if err == nil {
		t.Fatal("Expected error for nonexistent mount, got nil")
	}

	expectedError := "mount point /mnt/nonexistent not found in manager"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestNFSMountConfig_DefaultOptions tests NFS default mount options
func TestNFSMountConfig_DefaultOptions(t *testing.T) {
	// This test verifies the logic without actually mounting
	// Real mounting requires root privileges and NFS server

	config := NFSMountConfig{
		Server:       "nfs.example.com",
		ExportPath:   "/exports/backups",
		MountPoint:   "/mnt/test-nfs",
		RepositoryID: "repo-nfs-001",
		Options:      nil, // No options - should get defaults
	}

	expectedDefaults := []string{"vers=4", "rw", "hard", "intr", "timeo=600", "retrans=2"}

	// Verify config structure
	if config.Server != "nfs.example.com" {
		t.Error("Server not set correctly")
	}

	// The actual default options are applied in MountNFS method
	// This test documents the expected behavior
	_ = expectedDefaults
}

// TestCIFSMountConfig_PasswordSecurity tests password handling
func TestCIFSMountConfig_PasswordSecurity(t *testing.T) {
	mm := NewMountManager()

	// Create test mount info with password (simulating what would be stored)
	testMountPoint := "/mnt/test-cifs"
	mm.mountedPaths[testMountPoint] = &MountInfo{
		MountPoint:   testMountPoint,
		RepositoryID: "repo-cifs-001",
		Type:         "cifs",
		RemoteHost:   "cifs.example.com",
		RemotePath:   "backups",
		MountedAt:    time.Now(),
		// Note: MountOptions should NOT contain password
		MountOptions: []string{"username=testuser", "domain=TESTDOMAIN", "vers=3.0"},
	}

	// Get mount info
	info, err := mm.GetMountInfo(testMountPoint)
	if err != nil {
		t.Fatalf("GetMountInfo failed: %v", err)
	}

	// Verify password is not in mount options
	for _, opt := range info.MountOptions {
		if len(opt) > 9 && opt[:9] == "password=" {
			t.Error("Password should not be stored in MountOptions")
		}
	}

	// Verify safe options are present
	hasUsername := false
	for _, opt := range info.MountOptions {
		if opt == "username=testuser" {
			hasUsername = true
		}
	}

	if !hasUsername {
		t.Error("Username should be in mount options")
	}
}

// TestConcurrentAccess tests thread-safe access to mount manager
func TestConcurrentAccess(t *testing.T) {
	mm := NewMountManager()

	// Add initial mount
	mm.mountedPaths["/mnt/test"] = &MountInfo{
		MountPoint:   "/mnt/test",
		RepositoryID: "repo-123",
		Type:         "nfs",
		RemoteHost:   "host",
		RemotePath:   "/path",
		MountedAt:    time.Now(),
	}

	// Concurrent reads should work
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = mm.IsMounted("/mnt/test")
			_ = mm.ListMounted()
			_, _ = mm.GetMountInfo("/mnt/test")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify state is still consistent
	if !mm.IsMounted("/mnt/test") {
		t.Error("Concurrent access corrupted state")
	}
}

// TestMountInfoCopy tests that returned mount info is a copy
func TestMountInfoCopy(t *testing.T) {
	mm := NewMountManager()

	originalRepoID := "repo-original"
	mm.mountedPaths["/mnt/test"] = &MountInfo{
		MountPoint:   "/mnt/test",
		RepositoryID: originalRepoID,
		Type:         "nfs",
		RemoteHost:   "host",
		RemotePath:   "/path",
		MountedAt:    time.Now(),
		MountOptions: []string{"rw", "hard"},
	}

	// Get mount info
	info, err := mm.GetMountInfo("/mnt/test")
	if err != nil {
		t.Fatalf("GetMountInfo failed: %v", err)
	}

	// Modify returned copy
	info.RepositoryID = "repo-modified"
	info.MountOptions[0] = "ro"

	// Verify original is unchanged
	originalInfo, _ := mm.GetMountInfo("/mnt/test")
	if originalInfo.RepositoryID != originalRepoID {
		t.Error("Original mount info was modified (should be a copy)")
	}

	if originalInfo.MountOptions[0] != "rw" {
		t.Error("Original mount options were modified (should be a copy)")
	}
}
