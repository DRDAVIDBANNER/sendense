// OMA API Server - OSSEA Migration Appliance API
// Implements unified migration API following project rules: minimal endpoints, modular design
// This is the main entry point for the OMA (OSSEA Migration Appliance) API server

// @title OMA Migration API
// @version 1.1.0
// @description OMA (OSSEA Migration Appliance) API for VMware to OSSEA migration operations
// @description Following project rules: minimal endpoints, modular design, clean interfaces
// @termsOfService http://swagger.io/terms/

// @contact.name OMA API Support
// @contact.email support@company.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8082
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/api"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/services"
)

var (
	port        = flag.Int("port", 8080, "Port for OMA API server")
	debug       = flag.Bool("debug", false, "Enable debug logging")
	dbType      = flag.String("db-type", "mariadb", "Database type: mariadb or memory")
	dbHost      = flag.String("db-host", "localhost", "Database host")
	dbPort      = flag.Int("db-port", 3306, "Database port")
	dbName      = flag.String("db-name", "migratekit", "Database name")
	dbUser      = flag.String("db-user", "migratekit", "Database user")
	dbPass      = flag.String("db-pass", "migratekit123", "Database password")
	authEnabled = flag.Bool("auth", true, "Enable authentication")
)

func main() {
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.WithFields(log.Fields{
		"version":   "1.1.0",
		"port":      *port,
		"db_type":   *dbType,
		"auth":      *authEnabled,
		"log_level": log.GetLevel().String(),
	}).Info("Starting OMA Migration API server")

	// Initialize database connection
	var db *database.MariaDBConnection
	var err error

	if *dbType == "mariadb" {
		config := &database.MariaDBConfig{
			Host:     *dbHost,
			Port:     *dbPort,
			Database: *dbName,
			Username: *dbUser,
			Password: *dbPass,
		}
		db, err = database.NewMariaDBConnection(config)
		if err != nil {
			log.WithError(err).Fatal("Failed to connect to database")
		}
		log.Info("Database connection established")
	} else {
		log.Info("Using in-memory storage (no persistence)")
	}

	// Create and configure the API server
	serverConfig := &api.Config{
		Port:        *port,
		AuthEnabled: *authEnabled,
		Database:    db,
	}

	apiServer, err := api.NewServer(serverConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to create OMA API server")
	}

	// 🎯 PRODUCTION ENHANCEMENT: Intelligent job recovery with VMA validation on startup
	log.Info("🔍 Initializing intelligent job recovery system with VMA validation")
	
	// Get VMA services from API server handlers
	vmaClient := apiServer.GetHandlers().VMAProgressClient
	vmaPoller := apiServer.GetHandlers().VMAProgressPoller
	
	if vmaClient == nil || vmaPoller == nil {
		log.Warn("⚠️ VMA services not available - job recovery will be limited")
	}
	
	// Create job recovery with VMA validation capabilities
	jobRecovery := services.NewProductionJobRecovery(db, vmaClient, vmaPoller)
	
	// Run intelligent recovery on startup
	log.Info("🚀 Running intelligent job recovery scan with VMA validation...")
	if err := jobRecovery.RecoverOrphanedJobsOnStartup(context.Background()); err != nil {
		log.WithError(err).Warn("⚠️ Job recovery failed during startup - continuing with normal operation")
	} else {
		log.Info("✅ Job recovery completed successfully")
	}
	
	// 🏥 PRODUCTION ENHANCEMENT: Continuous health monitoring for orphaned jobs
	log.Info("🏥 Starting VMA polling health monitor for continuous job monitoring")
	healthMonitor := services.NewVMAPollingHealthMonitor(db, vmaClient, vmaPoller)
	if err := healthMonitor.Start(context.Background()); err != nil {
		log.WithError(err).Warn("⚠️ Failed to start health monitor - continuing without continuous monitoring")
	} else {
		log.Info("✅ Health monitor started - will check for orphaned jobs every 2 minutes")
	}

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("🛑 Shutdown signal received, stopping OMA API server")
		// Database cleanup would go here when implemented
		os.Exit(0)
	}()

	// Start the server
	log.WithField("port", *port).Info("OMA API server started successfully")
	log.WithField("url", fmt.Sprintf("http://localhost:%d/swagger/index.html", *port)).Info("Swagger documentation available at")
	log.WithField("url", fmt.Sprintf("http://localhost:%d/health", *port)).Info("Health check available at")

	ctx := context.Background()
	if err := apiServer.Start(ctx); err != nil {
		log.WithError(err).Fatal("Failed to start OMA API server")
	}
}
