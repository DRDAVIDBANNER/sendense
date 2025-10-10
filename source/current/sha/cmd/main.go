// SHA API Server - OSSEA Migration Appliance API
// Implements unified migration API following project rules: minimal endpoints, modular design
// This is the main entry point for the SHA (OSSEA Migration Appliance) API server

// @title SHA Migration API
// @version 1.1.0
// @description SHA (OSSEA Migration Appliance) API for VMware to OSSEA migration operations
// @description Following project rules: minimal endpoints, modular design, clean interfaces
// @termsOfService http://swagger.io/terms/

// @contact.name SHA API Support
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

	"github.com/vexxhost/migratekit-sha/api"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/services"
)

var (
	port        = flag.Int("port", 8080, "Port for SHA API server")
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
	}).Info("Starting SHA Migration API server")

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
		log.WithError(err).Fatal("Failed to create SHA API server")
	}

	// 🚨 REMOVED (2025-10-10): Old SNA progress-based job recovery
	// Replaced by telemetry-based stale job detection (see stale_job_detector.go)
	// Old job recovery code removed as part of polling → push telemetry migration
	log.Info("✅ Telemetry-based stale job detection will handle orphaned jobs")

	// 🚨 NEW: Stale job detector for telemetry-based progress tracking
	log.Info("🚨 Starting stale job detector for real-time telemetry monitoring")
	staleDetector := services.NewStaleJobDetector(db)
	go staleDetector.Start(context.Background())

	// 🆕 NEW: Execution monitor to update flow execution status when jobs complete
	log.Info("🔍 Starting execution monitor for flow completion tracking")
	flowRepo := database.NewFlowRepository(db)
	executionMonitor := services.NewExecutionMonitor(flowRepo, db)
	executionMonitor.Start()

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("🛑 Shutdown signal received, stopping SHA API server")
		// Database cleanup would go here when implemented
		os.Exit(0)
	}()

	// Start the server
	log.WithField("port", *port).Info("SHA API server started successfully")
	log.WithField("url", fmt.Sprintf("http://localhost:%d/swagger/index.html", *port)).Info("Swagger documentation available at")
	log.WithField("url", fmt.Sprintf("http://localhost:%d/health", *port)).Info("Health check available at")

	ctx := context.Background()
	if err := apiServer.Start(ctx); err != nil {
		log.WithError(err).Fatal("Failed to start SHA API server")
	}
}
