// +build integration

package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Integration tests require:
// 1. Running MariaDB instance
// 2. qemu-img installed
// 3. Writable test directory
// Run with: go test -tags=integration -v

func setupTestDB(t *testing.T) *sql.DB {
	// Connect to test database
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(localhost:3306)/sendense_test"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Create test schema
	schema := `
		CREATE TABLE IF NOT EXISTS backup_repositories (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			repository_type ENUM('local', 'nfs', 'cifs', 'smb', 's3', 'azure') NOT NULL,
			enabled BOOLEAN DEFAULT TRUE,
			config JSON NOT NULL,
			is_immutable BOOLEAN DEFAULT FALSE,
			immutable_config JSON NULL,
			min_retention_days INT DEFAULT 0,
			total_size_bytes BIGINT DEFAULT 0,
			used_size_bytes BIGINT DEFAULT 0,
			available_size_bytes BIGINT DEFAULT 0,
			last_check_at TIMESTAMP NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_name (name)
		);

		CREATE TABLE IF NOT EXISTS backup_jobs (
			id VARCHAR(64) PRIMARY KEY,
			vm_context_id VARCHAR(191) NOT NULL,
			vm_name VARCHAR(255) NOT NULL,
			repository_id VARCHAR(64) NOT NULL,
			backup_type ENUM('full', 'incremental', 'differential') NOT NULL,
			status ENUM('pending', 'running', 'completed', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
			repository_path VARCHAR(512) NOT NULL,
			parent_backup_id VARCHAR(64) NULL,
			change_id VARCHAR(191) NULL,
			bytes_transferred BIGINT DEFAULT 0,
			total_bytes BIGINT DEFAULT 0,
			compression_enabled BOOLEAN DEFAULT TRUE,
			error_message TEXT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP NULL,
			completed_at TIMESTAMP NULL,
			FOREIGN KEY (repository_id) REFERENCES backup_repositories(id)
		);

		CREATE TABLE IF NOT EXISTS backup_chains (
			id VARCHAR(64) PRIMARY KEY,
			vm_context_id VARCHAR(191) NOT NULL,
			disk_id INT NOT NULL,
			full_backup_id VARCHAR(64) NOT NULL,
			latest_backup_id VARCHAR(64) NOT NULL,
			total_backups INT DEFAULT 0,
			total_size_bytes BIGINT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_vm_disk (vm_context_id, disk_id)
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	return db
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
	_, _ = db.Exec("DROP TABLE IF EXISTS backup_chains")
	_, _ = db.Exec("DROP TABLE IF EXISTS backup_jobs")
	_, _ = db.Exec("DROP TABLE IF EXISTS backup_repositories")
	db.Close()
}

func TestEndToEndRepositoryLifecycle(t *testing.T) {
	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping integration test")
	}

	// Setup
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tmpDir, err := os.MkdirTemp("", "sendense-integration-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Create RepositoryManager
	rm, err := NewRepositoryManager(db)
	if err != nil {
		t.Fatalf("failed to create RepositoryManager: %v", err)
	}

	// Register a local repository
	config := &RepositoryConfig{
		ID:      "test-repo-1",
		Name:    "Test Repository",
		Type:    RepositoryTypeLocal,
		Enabled: true,
		Config: LocalRepositoryConfig{
			Path: tmpDir,
		},
	}

	err = rm.RegisterRepository(ctx, config)
	if err != nil {
		t.Fatalf("failed to register repository: %v", err)
	}

	// Get repository
	repo, err := rm.GetRepository(ctx, "test-repo-1")
	if err != nil {
		t.Fatalf("failed to get repository: %v", err)
	}

	// Create full backup
	backupReq := BackupRequest{
		VMContextID:  "ctx-test-vm-20251004-120000",
		VMName:       "test-vm",
		RepositoryID: "test-repo-1",
		BackupType:   BackupTypeFull,
		DiskID:       0,
		DiskSize:     1073741824, // 1 GB
	}

	backup, err := repo.CreateBackup(ctx, backupReq)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	if backup.ID == "" {
		t.Error("backup ID should not be empty")
	}
	if backup.Status != BackupStatusCompleted {
		t.Errorf("backup status = %v, want %v", backup.Status, BackupStatusCompleted)
	}

	// Verify backup file exists
	if _, err := os.Stat(backup.RepositoryPath); os.IsNotExist(err) {
		t.Error("backup file should exist")
	}

	// Get backup
	retrievedBackup, err := repo.GetBackup(ctx, backup.ID)
	if err != nil {
		t.Fatalf("failed to get backup: %v", err)
	}

	if retrievedBackup.ID != backup.ID {
		t.Errorf("retrieved backup ID = %v, want %v", retrievedBackup.ID, backup.ID)
	}

	// List backups
	backups, err := repo.ListBackups(ctx, "ctx-test-vm-20251004-120000")
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("backups count = %v, want 1", len(backups))
	}

	// Get storage info
	storageInfo, err := repo.GetStorageInfo(ctx)
	if err != nil {
		t.Fatalf("failed to get storage info: %v", err)
	}

	if storageInfo.TotalBytes == 0 {
		t.Error("total bytes should not be zero")
	}

	// Delete backup
	err = repo.DeleteBackup(ctx, backup.ID)
	if err != nil {
		t.Fatalf("failed to delete backup: %v", err)
	}

	// Verify backup file is deleted
	if _, err := os.Stat(backup.RepositoryPath); !os.IsNotExist(err) {
		t.Error("backup file should be deleted")
	}

	// Delete repository
	err = rm.DeleteRepository(ctx, "test-repo-1")
	if err != nil {
		t.Fatalf("failed to delete repository: %v", err)
	}
}

func TestIncrementalBackupChain(t *testing.T) {
	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping integration test")
	}

	// Setup
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tmpDir, err := os.MkdirTemp("", "sendense-integration-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Create repository
	rm, err := NewRepositoryManager(db)
	if err != nil {
		t.Fatalf("failed to create RepositoryManager: %v", err)
	}

	config := &RepositoryConfig{
		ID:      "test-repo-2",
		Name:    "Test Repository 2",
		Type:    RepositoryTypeLocal,
		Enabled: true,
		Config: LocalRepositoryConfig{
			Path: tmpDir,
		},
	}

	err = rm.RegisterRepository(ctx, config)
	if err != nil {
		t.Fatalf("failed to register repository: %v", err)
	}

	repo, err := rm.GetRepository(ctx, "test-repo-2")
	if err != nil {
		t.Fatalf("failed to get repository: %v", err)
	}

	vmContextID := "ctx-test-vm-20251004-130000"

	// Create full backup
	fullReq := BackupRequest{
		VMContextID:  vmContextID,
		VMName:       "test-vm-2",
		RepositoryID: "test-repo-2",
		BackupType:   BackupTypeFull,
		DiskID:       0,
		DiskSize:     1073741824, // 1 GB
	}

	fullBackup, err := repo.CreateBackup(ctx, fullReq)
	if err != nil {
		t.Fatalf("failed to create full backup: %v", err)
	}

	// Create first incremental backup
	incr1Req := BackupRequest{
		VMContextID:    vmContextID,
		VMName:         "test-vm-2",
		RepositoryID:   "test-repo-2",
		BackupType:     BackupTypeIncremental,
		ParentBackupID: fullBackup.ID,
		DiskID:         0,
		DiskSize:       1073741824,
	}

	incr1Backup, err := repo.CreateBackup(ctx, incr1Req)
	if err != nil {
		t.Fatalf("failed to create incremental backup 1: %v", err)
	}

	if incr1Backup.ParentBackupID != fullBackup.ID {
		t.Errorf("incremental backup parent = %v, want %v", incr1Backup.ParentBackupID, fullBackup.ID)
	}

	// Create second incremental backup
	incr2Req := BackupRequest{
		VMContextID:    vmContextID,
		VMName:         "test-vm-2",
		RepositoryID:   "test-repo-2",
		BackupType:     BackupTypeIncremental,
		ParentBackupID: incr1Backup.ID,
		DiskID:         0,
		DiskSize:       1073741824,
	}

	incr2Backup, err := repo.CreateBackup(ctx, incr2Req)
	if err != nil {
		t.Fatalf("failed to create incremental backup 2: %v", err)
	}

	// Get backup chain
	chain, err := repo.GetBackupChain(ctx, vmContextID, 0)
	if err != nil {
		t.Fatalf("failed to get backup chain: %v", err)
	}

	if chain.FullBackupID != fullBackup.ID {
		t.Errorf("chain full backup ID = %v, want %v", chain.FullBackupID, fullBackup.ID)
	}
	if chain.LatestBackupID != incr2Backup.ID {
		t.Errorf("chain latest backup ID = %v, want %v", chain.LatestBackupID, incr2Backup.ID)
	}
	if chain.TotalBackups != 3 {
		t.Errorf("chain total backups = %v, want 3", chain.TotalBackups)
	}

	// List all backups for VM
	backups, err := repo.ListBackups(ctx, vmContextID)
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("backups count = %v, want 3", len(backups))
	}
}

func TestMultipleRepositories(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tmpDir1, err := os.MkdirTemp("", "sendense-repo1-*")
	if err != nil {
		t.Fatalf("failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tmpDir1)

	tmpDir2, err := os.MkdirTemp("", "sendense-repo2-*")
	if err != nil {
		t.Fatalf("failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tmpDir2)

	ctx := context.Background()

	// Create RepositoryManager
	rm, err := NewRepositoryManager(db)
	if err != nil {
		t.Fatalf("failed to create RepositoryManager: %v", err)
	}

	// Register first repository
	config1 := &RepositoryConfig{
		ID:      "test-repo-primary",
		Name:    "Primary Repository",
		Type:    RepositoryTypeLocal,
		Enabled: true,
		Config: LocalRepositoryConfig{
			Path: tmpDir1,
		},
	}

	err = rm.RegisterRepository(ctx, config1)
	if err != nil {
		t.Fatalf("failed to register repository 1: %v", err)
	}

	// Register second repository
	config2 := &RepositoryConfig{
		ID:      "test-repo-secondary",
		Name:    "Secondary Repository",
		Type:    RepositoryTypeLocal,
		Enabled: true,
		Config: LocalRepositoryConfig{
			Path: tmpDir2,
		},
	}

	err = rm.RegisterRepository(ctx, config2)
	if err != nil {
		t.Fatalf("failed to register repository 2: %v", err)
	}

	// List repositories
	repos, err := rm.ListRepositories(ctx)
	if err != nil {
		t.Fatalf("failed to list repositories: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("repositories count = %v, want 2", len(repos))
	}

	// Get each repository
	repo1, err := rm.GetRepository(ctx, "test-repo-primary")
	if err != nil {
		t.Fatalf("failed to get repository 1: %v", err)
	}
	if repo1 == nil {
		t.Fatal("repository 1 should not be nil")
	}

	repo2, err := rm.GetRepository(ctx, "test-repo-secondary")
	if err != nil {
		t.Fatalf("failed to get repository 2: %v", err)
	}
	if repo2 == nil {
		t.Fatal("repository 2 should not be nil")
	}

	// Refresh storage info
	err = rm.RefreshStorageInfo(ctx)
	if err != nil {
		t.Fatalf("failed to refresh storage info: %v", err)
	}
}

func TestRepositoryErrorHandling(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	ctx := context.Background()

	rm, err := NewRepositoryManager(db)
	if err != nil {
		t.Fatalf("failed to create RepositoryManager: %v", err)
	}

	// Test getting non-existent repository
	_, err = rm.GetRepository(ctx, "nonexistent-repo")
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}

	// Test registering repository with invalid path
	config := &RepositoryConfig{
		ID:      "test-invalid",
		Name:    "Invalid Repository",
		Type:    RepositoryTypeLocal,
		Enabled: true,
		Config: LocalRepositoryConfig{
			Path: "/nonexistent/path/that/does/not/exist",
		},
	}

	err = rm.RegisterRepository(ctx, config)
	if err == nil {
		t.Error("RegisterRepository should fail for invalid path")
	}
}

func TestConcurrentBackupCreation(t *testing.T) {
	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping integration test")
	}

	// Setup
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tmpDir, err := os.MkdirTemp("", "sendense-concurrent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	rm, err := NewRepositoryManager(db)
	if err != nil {
		t.Fatalf("failed to create RepositoryManager: %v", err)
	}

	config := &RepositoryConfig{
		ID:      "test-repo-concurrent",
		Name:    "Concurrent Test Repository",
		Type:    RepositoryTypeLocal,
		Enabled: true,
		Config: LocalRepositoryConfig{
			Path: tmpDir,
		},
	}

	err = rm.RegisterRepository(ctx, config)
	if err != nil {
		t.Fatalf("failed to register repository: %v", err)
	}

	repo, err := rm.GetRepository(ctx, "test-repo-concurrent")
	if err != nil {
		t.Fatalf("failed to get repository: %v", err)
	}

	// Create multiple backups concurrently
	numBackups := 3
	errChan := make(chan error, numBackups)

	for i := 0; i < numBackups; i++ {
		go func(index int) {
			req := BackupRequest{
				VMContextID:  filepath.Join("ctx-test-vm", time.Now().Format("20060102-150405")),
				VMName:       filepath.Join("test-vm", string(rune(index))),
				RepositoryID: "test-repo-concurrent",
				BackupType:   BackupTypeFull,
				DiskID:       index,
				DiskSize:     1073741824,
			}

			_, err := repo.CreateBackup(ctx, req)
			errChan <- err
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numBackups; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("concurrent backup %d failed: %v", i, err)
		}
	}
}

