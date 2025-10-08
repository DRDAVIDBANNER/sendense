// Package handlers provides HTTP handlers for SHA API endpoints
// Following project rules: modular design, small focused functions, clean separation
package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/ossea"
	"github.com/vexxhost/migratekit-sha/services"
	"github.com/vexxhost/migratekit-sha/volume"
	"github.com/vexxhost/migratekit-sha/workflows"
)

// Handlers contains all API endpoint handlers
// Follows project rules: clean interfaces, modular design
type Handlers struct {
	Auth                   *AuthHandler
	VM                     *VMHandler
	Replication            *ReplicationHandler
	OSSEA                  *OSSEAHandler
	Linstor                *LinstorHandler
	NetworkMapping         *NetworkMappingHandler
	Failover               *FailoverHandler
	Validation             *ValidationHandler
	Debug                  *DebugHandler
	VMContext              *VMContextHandler // VM-Centric Architecture GUI endpoints
	ScheduleManagement     *ScheduleManagementHandler
	MachineGroupManagement *MachineGroupManagementHandler
	VMGroupAssignment      *VMGroupAssignmentHandler
	EnhancedDiscovery      *EnhancedDiscoveryHandler
	VMwareCredentials      *VMwareCredentialsHandler      // 🆕 NEW: VMware credential management
	StreamlinedOSSEA       *StreamlinedOSSEAConfigHandler // 🆕 NEW: Streamlined OSSEA configuration
	SNAReal                *SNARealHandler                // 🆕 NEW: SNA enrollment system (real implementation)
	CloudStackSettings     *CloudStackSettingsHandler     // 🆕 NEW: CloudStack validation & settings
	Repository             *RepositoryHandler             // 🆕 NEW: Backup repository management (Storage Monitoring Day 4)
	Policy                 *PolicyHandler                 // 🆕 NEW: Backup policy management (Backup Copy Engine Day 5)
	Restore                *RestoreHandlers               // 🆕 NEW: File-level restore (Task 4 - 2025-10-05)
	Backup                 *BackupHandler                 // 🆕 NEW: Backup API endpoints (Task 5 - 2025-10-05)
	
	// Exposed services for job recovery integration
	SNAProgressClient *services.SNAProgressClient // SNA API client
	SNAProgressPoller *services.SNAProgressPoller // SNA progress poller
}

// NewHandlers creates a new handlers instance with database connection and mount manager
func NewHandlers(db database.Connection) (*Handlers, error) {
	// Initialize volume mount manager with default mount path
	// Note: We pass nil for database since mount manager can work without it for basic operations
	mountManager := volume.NewMountManager(nil, "/mnt/migration")

	// Initialize SNA progress services (via tunnel)
	snaProgressClient := services.NewVMAProgressClient("http://localhost:9081")
	repo := database.NewOSSEAConfigRepository(db)
	
	// 🆕 TASK 3: Initialize encryption service EARLY for OSSEA config repository
	encryptionService, err := services.NewCredentialEncryptionService()
	if err != nil {
		log.WithError(err).Warn("Credential encryption service unavailable - credentials will be stored in plaintext")
	} else {
		repo.SetEncryptionService(encryptionService)
		log.Info("✅ Credential encryption enabled for OSSEA configuration")
	}
	
	snaProgressPoller := services.NewVMAProgressPoller(snaProgressClient, repo)

	// Start SNA progress polling service
	ctx := context.Background()
	if err := snaProgressPoller.Start(ctx); err != nil {
		log.WithError(err).Warn("Failed to start SNA progress poller - continuing without real-time progress")
	} else {
		log.Info("🚀 SNA progress poller started successfully")
	}

	// Try to initialize OSSEA clients for network mapping handler
	var osseaClient *ossea.Client
	var networkClient *ossea.NetworkClient

	// Get active OSSEA configuration for network operations
	var configs []database.OSSEAConfig
	err = db.GetGormDB().Where("is_active = true").Find(&configs).Error
	if err == nil && len(configs) > 0 {
		config := configs[0]
		log.WithField("config_name", config.Name).Info("🔧 Initializing OSSEA clients for network operations")

		// Create OSSEA client
		osseaClient = ossea.NewClient(
			config.APIURL,
			config.APIKey,
			config.SecretKey,
			config.Domain,
			config.Zone,
		)
		networkClient = ossea.NewNetworkClient(osseaClient)
	} else {
		log.Warn("⚠️ No active OSSEA configuration found - network resolution will be limited")
	}

	// Initialize scheduler-related services and repositories
	schedulerRepo := database.NewSchedulerRepository(db)
	replicationRepo := database.NewReplicationJobRepository(db)
	vmContextRepo := database.NewVMReplicationContextRepository(db)

	// Initialize joblog tracker (mandatory for all operations)
	var jobTracker *joblog.Tracker
	if sqlDB, err := db.GetGormDB().DB(); err == nil {
		stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
		dbHandler := joblog.NewDBHandler(sqlDB, joblog.DefaultDBHandlerConfig())
		jobTracker = joblog.New(sqlDB, stdoutHandler, dbHandler)
	} else {
		log.WithError(err).Error("Failed to initialize JobLog tracker, scheduler services will have limited functionality")
		return nil, err
	}

	// ✅ UPDATED: Scheduler service now uses SNA discovery + SHA API (aligned with GUI workflow)
	// No longer needs direct Migration Engine access - uses same API path as GUI
	snaAPIEndpoint := "http://localhost:9081" // SNA API via tunnel
	schedulerService := services.NewSchedulerService(schedulerRepo, replicationRepo, jobTracker, snaAPIEndpoint)

	// 🚀 CRITICAL: Start the scheduler service to enable automatic job scheduling
	log.Info("🚀 Starting scheduler service for automatic job execution")
	if err := schedulerService.Start(context.Background()); err != nil {
		log.WithError(err).Error("Failed to start scheduler service")
		return nil, err
	}
	log.Info("✅ Scheduler service started - automatic jobs will now trigger at scheduled times")

	// Initialize machine group service
	machineGroupService := services.NewMachineGroupService(schedulerRepo, jobTracker)

	// Initialize enhanced discovery service with SNA API endpoint
	// 🆕 Pass main database connection (not schedulerRepo) for vm_disks creation
	enhancedDiscoveryService := services.NewEnhancedDiscoveryService(vmContextRepo, db, jobTracker, snaAPIEndpoint)

	// 🆕 NEW: Initialize VMware credentials management services
	// Note: encryptionService already initialized earlier for OSSEA config repository
	// Handle nil encryptionService gracefully
	if encryptionService == nil {
		log.Warn("⚠️ VMware credential service running without encryption (development mode)")
	}
	vmwareCredentialService := services.NewVMwareCredentialService(&db, encryptionService)

	// 🆕 NEW: Initialize SNA enrollment services (commented out for simple version)
	// snaEnrollmentRepo := database.NewVMAEnrollmentRepository(db)
	// snaAuditRepo := database.NewVMAAuditRepository(db)
	// snaCryptoService := services.NewVMACryptoService()
	// snaEnrollmentService := services.NewVMAEnrollmentService(db, snaEnrollmentRepo, snaAuditRepo, snaCryptoService)
	// snaAuditService := services.NewVMAAuditService(snaAuditRepo)

	handlers := &Handlers{
		Auth:                   NewAuthHandler(db),
		VM:                     NewVMHandler(db),
		Replication:            NewReplicationHandler(db, mountManager, snaProgressPoller),
		OSSEA:                  NewOSSEAHandler(db),
		Linstor:                NewLinstorHandler(db),
		NetworkMapping:         NewNetworkMappingHandler(db, osseaClient, networkClient),
		Failover:               NewEnhancedFailoverHandler(db), // Using enhanced failover with JobLog integration
		Validation:             NewValidationHandler(db),
		Debug:                  NewDebugHandler(db),
		VMContext:              NewVMContextHandler(db, jobTracker), // VM-Centric Architecture GUI endpoints with JobLog integration
		ScheduleManagement:     NewScheduleManagementHandler(schedulerRepo, schedulerService, jobTracker),
		MachineGroupManagement: NewMachineGroupManagementHandler(machineGroupService, schedulerRepo, jobTracker),
		VMGroupAssignment:      NewVMGroupAssignmentHandler(machineGroupService, schedulerRepo, vmContextRepo, jobTracker),
		EnhancedDiscovery:      NewEnhancedDiscoveryHandler(enhancedDiscoveryService, vmContextRepo, schedulerRepo, jobTracker, db), // 🆕 NEW: Pass db for credential lookup
		VMwareCredentials:      NewVMwareCredentialsHandler(vmwareCredentialService), // 🆕 NEW: VMware credential management
		StreamlinedOSSEA:       NewStreamlinedOSSEAConfigHandler(db),                 // 🆕 NEW: Streamlined OSSEA configuration
		SNAReal:                NewVMARealHandler(db),                                // 🆕 NEW: SNA enrollment system (real implementation)
		CloudStackSettings:     NewCloudStackSettingsHandler(db),                     // 🆕 NEW: CloudStack validation & settings
		
		// Expose SNA services for job recovery
		SNAProgressClient: snaProgressClient,
		SNAProgressPoller: snaProgressPoller,
	}

	// Initialize Repository handler (requires separate initialization due to error handling)
	sqlDB, err := handlers.extractSQLDB(db)
	if err != nil {
		log.WithError(err).Warn("Failed to get SQL DB from connection - repository management unavailable")
	} else {
		repositoryHandler, err := NewRepositoryHandler(sqlDB)
		if err != nil {
			log.WithError(err).Warn("Repository handler initialization failed - repository management endpoints unavailable")
		} else {
			handlers.Repository = repositoryHandler
		}

		// Initialize Policy handler (requires same SQL DB)
		policyHandler, err := NewPolicyHandler(sqlDB)
		if err != nil {
			log.WithError(err).Warn("Policy handler initialization failed - backup policy endpoints unavailable")
		} else {
			handlers.Policy = policyHandler
			log.Info("✅ Backup policy management enabled (Enterprise 3-2-1 backup rule support)")
		}

		// Initialize Restore handler (Task 4: File-Level Restore)
		restoreHandler := NewRestoreHandlers(db, repositoryHandler.repoManager)
		handlers.Restore = restoreHandler
		log.Info("✅ File-level restore enabled (Task 4: Mount backups, browse files, download)")

		// Initialize Backup handler (Task 5: Backup API Endpoints)
		// Requires BackupEngine integration with repository manager
		
		// 🆕 Initialize NBD Port Allocator (10100-10200 range for 100 concurrent backups)
		nbdPortAllocator := services.NewNBDPortAllocator(10100, 10200)
		
		// 🆕 Initialize qemu-nbd Process Manager with automatic port release
		qemuNBDManager := services.NewQemuNBDManager(nbdPortAllocator)
		
		// Initialize BackupEngine with NBD infrastructure
		backupEngine := workflows.NewBackupEngine(db, repositoryHandler.repoManager, nbdPortAllocator, qemuNBDManager, snaAPIEndpoint)
		
		backupHandler := NewBackupHandler(db, backupEngine, nbdPortAllocator, qemuNBDManager, vmwareCredentialService)
		handlers.Backup = backupHandler
		log.Info("✅ Backup API endpoints enabled (Task 5: Start, list, delete backups via REST API + Unified NBD Architecture)")
	}

	return handlers, nil
}

// extractSQLDB extracts *sql.DB from database.Connection
func (h *Handlers) extractSQLDB(conn database.Connection) (*sql.DB, error) {
	gormDB := conn.GetGormDB()
	if gormDB == nil {
		return nil, fmt.Errorf("GORM DB is nil")
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL DB: %w", err)
	}
	return sqlDB, nil
}
