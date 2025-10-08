package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewChainManager(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	cm := NewChainManager(db)
	if cm == nil {
		t.Fatal("NewChainManager returned nil")
	}
	if cm.db == nil {
		t.Error("ChainManager.db should not be nil")
	}
}

func TestCreateChain(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	cm := NewChainManager(db)
	ctx := context.Background()

	vmContextID := "ctx-vm1-20251004-153000"
	diskID := 0
	fullBackupID := "backup-full-123"

	// Expect INSERT query
	mock.ExpectExec("INSERT INTO backup_chains").
		WithArgs(sqlmock.AnyArg(), vmContextID, diskID, fullBackupID, fullBackupID, 1, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	chain, err := cm.CreateChain(ctx, vmContextID, diskID, fullBackupID, 42949672960)
	if err != nil {
		t.Fatalf("CreateChain failed: %v", err)
	}

	if chain.VMContextID != vmContextID {
		t.Errorf("chain.VMContextID = %v, want %v", chain.VMContextID, vmContextID)
	}
	if chain.DiskID != diskID {
		t.Errorf("chain.DiskID = %v, want %v", chain.DiskID, diskID)
	}
	if chain.FullBackupID != fullBackupID {
		t.Errorf("chain.FullBackupID = %v, want %v", chain.FullBackupID, fullBackupID)
	}
	if chain.LatestBackupID != fullBackupID {
		t.Errorf("chain.LatestBackupID = %v, want %v", chain.LatestBackupID, fullBackupID)
	}
	if chain.TotalBackups != 1 {
		t.Errorf("chain.TotalBackups = %v, want %v", chain.TotalBackups, 1)
	}
	if chain.TotalSizeBytes != 42949672960 {
		t.Errorf("chain.TotalSizeBytes = %v, want %v", chain.TotalSizeBytes, 42949672960)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAddToChain(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	cm := NewChainManager(db)
	ctx := context.Background()

	chainID := "chain-vm1-disk0"
	newBackupID := "backup-inc-456"
	newBackupSize := int64(4294967296)

	// Expect SELECT query for current chain
	rows := sqlmock.NewRows([]string{"total_backups", "total_size_bytes"}).
		AddRow(2, 47244640256)
	mock.ExpectQuery("SELECT total_backups, total_size_bytes FROM backup_chains").
		WithArgs(chainID).
		WillReturnRows(rows)

	// Expect UPDATE query
	mock.ExpectExec("UPDATE backup_chains").
		WithArgs(newBackupID, 3, 51539607552, sqlmock.AnyArg(), chainID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = cm.AddToChain(ctx, chainID, newBackupID, newBackupSize)
	if err != nil {
		t.Fatalf("AddToChain failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetChain(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	cm := NewChainManager(db)
	ctx := context.Background()

	vmContextID := "ctx-vm1-20251004-153000"
	diskID := 0
	now := time.Now()

	// Expect SELECT query
	rows := sqlmock.NewRows([]string{
		"id", "vm_context_id", "disk_id", "full_backup_id", "latest_backup_id",
		"total_backups", "total_size_bytes", "created_at", "updated_at",
	}).AddRow(
		"chain-vm1-disk0", vmContextID, diskID, "backup-full-123", "backup-inc-456",
		3, int64(51539607552), now, now,
	)
	mock.ExpectQuery("SELECT (.+) FROM backup_chains").
		WithArgs(vmContextID, diskID).
		WillReturnRows(rows)

	chain, err := cm.GetChain(ctx, vmContextID, diskID)
	if err != nil {
		t.Fatalf("GetChain failed: %v", err)
	}

	if chain.ID != "chain-vm1-disk0" {
		t.Errorf("chain.ID = %v, want %v", chain.ID, "chain-vm1-disk0")
	}
	if chain.VMContextID != vmContextID {
		t.Errorf("chain.VMContextID = %v, want %v", chain.VMContextID, vmContextID)
	}
	if chain.DiskID != diskID {
		t.Errorf("chain.DiskID = %v, want %v", chain.DiskID, diskID)
	}
	if chain.TotalBackups != 3 {
		t.Errorf("chain.TotalBackups = %v, want %v", chain.TotalBackups, 3)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetChainNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	cm := NewChainManager(db)
	ctx := context.Background()

	// Expect SELECT query that returns no rows
	mock.ExpectQuery("SELECT (.+) FROM backup_chains").
		WithArgs("ctx-nonexistent", 0).
		WillReturnError(sql.ErrNoRows)

	_, err = cm.GetChain(ctx, "ctx-nonexistent", 0)
	if err == nil {
		t.Fatal("GetChain should return error for non-existent chain")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestValidateChain(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	cm := NewChainManager(db)
	ctx := context.Background()

	chainID := "chain-vm1-disk0"

	// Expect query for chain info
	chainRows := sqlmock.NewRows([]string{"full_backup_id"}).
		AddRow("backup-full-123")
	mock.ExpectQuery("SELECT full_backup_id FROM backup_chains").
		WithArgs(chainID).
		WillReturnRows(chainRows)

	// Expect query for all backups in chain
	backupRows := sqlmock.NewRows([]string{"id", "parent_backup_id", "repository_path"}).
		AddRow("backup-full-123", nil, "/backups/vm1/backup-full-123.qcow2").
		AddRow("backup-inc-456", "backup-full-123", "/backups/vm1/backup-inc-456.qcow2").
		AddRow("backup-inc-789", "backup-inc-456", "/backups/vm1/backup-inc-789.qcow2")
	mock.ExpectQuery("SELECT id, parent_backup_id, repository_path FROM backup_jobs").
		WillReturnRows(backupRows)

	valid, err := cm.ValidateChain(ctx, chainID)
	if err != nil {
		t.Fatalf("ValidateChain failed: %v", err)
	}

	if !valid {
		t.Error("ValidateChain should return true for valid chain")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestValidateChainBroken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	cm := NewChainManager(db)
	ctx := context.Background()

	chainID := "chain-vm1-disk0"

	// Expect query for chain info
	chainRows := sqlmock.NewRows([]string{"full_backup_id"}).
		AddRow("backup-full-123")
	mock.ExpectQuery("SELECT full_backup_id FROM backup_chains").
		WithArgs(chainID).
		WillReturnRows(chainRows)

	// Expect query for backups with broken chain (missing parent)
	backupRows := sqlmock.NewRows([]string{"id", "parent_backup_id", "repository_path"}).
		AddRow("backup-full-123", nil, "/backups/vm1/backup-full-123.qcow2").
		AddRow("backup-inc-789", "backup-inc-456", "/backups/vm1/backup-inc-789.qcow2") // Parent missing!
	mock.ExpectQuery("SELECT id, parent_backup_id, repository_path FROM backup_jobs").
		WillReturnRows(backupRows)

	valid, err := cm.ValidateChain(ctx, chainID)
	if err != nil {
		t.Fatalf("ValidateChain failed: %v", err)
	}

	if valid {
		t.Error("ValidateChain should return false for broken chain")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDeleteChain(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	cm := NewChainManager(db)
	ctx := context.Background()

	chainID := "chain-vm1-disk0"

	// Expect DELETE query
	mock.ExpectExec("DELETE FROM backup_chains").
		WithArgs(chainID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = cm.DeleteChain(ctx, chainID)
	if err != nil {
		t.Fatalf("DeleteChain failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestChainManagerEdgeCases(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	cm := NewChainManager(db)
	ctx := context.Background()

	t.Run("AddToChain with zero size", func(t *testing.T) {
		// Expect SELECT query
		rows := sqlmock.NewRows([]string{"total_backups", "total_size_bytes"}).
			AddRow(1, 42949672960)
		mock.ExpectQuery("SELECT total_backups, total_size_bytes FROM backup_chains").
			WithArgs("chain-test").
			WillReturnRows(rows)

		// Expect UPDATE query
		mock.ExpectExec("UPDATE backup_chains").
			WithArgs("backup-new", 2, int64(42949672960), sqlmock.AnyArg(), "chain-test").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := cm.AddToChain(ctx, "chain-test", "backup-new", 0)
		if err != nil {
			t.Errorf("AddToChain with zero size failed: %v", err)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

