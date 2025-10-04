package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/vexxhost/migratekit-volume-daemon/api"
	"github.com/vexxhost/migratekit-volume-daemon/cloudstack"
	"github.com/vexxhost/migratekit-volume-daemon/database"
	"github.com/vexxhost/migratekit-volume-daemon/device"
	"github.com/vexxhost/migratekit-volume-daemon/nbd"
	"github.com/vexxhost/migratekit-volume-daemon/repository"
	"github.com/vexxhost/migratekit-volume-daemon/service"
)

func main() {
	log.Println("🚀 Starting Volume Management Daemon...")

	// Initialize database connection
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repository
	repo := database.NewRepository(db)

	// Initialize CloudStack client factory
	csFactory := cloudstack.NewFactory(db)

	// Test CloudStack connectivity
	if err := csFactory.TestConnection(context.Background()); err != nil {
		log.Printf("⚠️  CloudStack connection test failed: %v", err)
		log.Println("📝 Volume daemon will start but CloudStack operations will fail until configuration is fixed")
	} else {
		log.Println("✅ CloudStack connectivity verified")
	}

	// Create CloudStack client (will be nil if no active config)
	var cloudStackClient service.CloudStackClient
	if client, err := csFactory.CreateClient(context.Background()); err != nil {
		log.Printf("⚠️  Failed to create CloudStack client: %v", err)
		log.Println("📝 Volume operations will fail until CloudStack is configured")
	} else {
		cloudStackClient = client
	}

	// Initialize device monitor (using polling instead of inotify for CloudStack compatibility)
	deviceMonitor, err := device.NewPollingMonitor()
	if err != nil {
		log.Printf("⚠️  Failed to create device monitor: %v", err)
		log.Println("📝 Volume daemon will continue without device monitoring")
		deviceMonitor = nil
	} else {
		// Start device monitoring
		if err := deviceMonitor.StartMonitoring(context.Background()); err != nil {
			log.Printf("⚠️  Failed to start device monitor: %v", err)
			deviceMonitor = nil
		} else {
			log.Println("✅ Device polling monitor started successfully")
		}
	}

	// Initialize NBD export manager using existing migratekit_oma database
	nbdConfigPath := "/etc/nbd-server/config-base"
	nbdConfDir := "/etc/nbd-server/conf.d"
	nbdExportRepo := repository.NewOMANBDRepository(db) // Use existing migratekit_oma nbd_exports table
	nbdExportManager := nbd.NewExportManager(nbdConfigPath, nbdConfDir, nbdExportRepo)

	log.Printf("✅ NBD Export Manager initialized (config: %s, conf.d: %s)", nbdConfigPath, nbdConfDir)

	// Initialize OSSEA volume repository for managing ossea_volumes table
	osseaVolumeRepo := repository.NewOSSEAVolumeRepository(db)
	log.Printf("✅ OSSEA Volume Repository initialized")

	// Initialize services
	volumeService := service.NewVolumeService(repo, cloudStackClient, deviceMonitor, nbdExportManager, osseaVolumeRepo)

	// Initialize NBD cleanup service
	cleanupService := service.NewNBDCleanupService(db, nbdExportManager)
	log.Printf("✅ NBD Cleanup Service initialized")

	// Initialize HTTP server
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"service":   "volume-management-daemon",
			"version":   "1.0.0",
		})
	})

	// Initialize API routes
	api.SetupRoutes(router, volumeService, cleanupService)

	// Setup graceful shutdown
	server := &http.Server{
		Addr:    ":8090",
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Println("🌐 Volume Management Daemon listening on :8090")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down Volume Management Daemon...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop device monitor first
	if deviceMonitor != nil {
		if err := deviceMonitor.StopMonitoring(ctx); err != nil {
			log.Printf("⚠️  Error stopping device monitor: %v", err)
		}
	}

	// Stop HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Volume Management Daemon stopped")
}

func initDatabase() (*sqlx.DB, error) {
	// For now, use environment variables or default values
	// TODO: Load from configuration file
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "oma_user:oma_password@tcp(localhost:3306)/migratekit_oma?parseTime=true&multiStatements=true"
	}

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err = db.Ping(); err != nil {
		return nil, err
	}

	log.Println("✅ Database connection established")
	return db, nil
}
