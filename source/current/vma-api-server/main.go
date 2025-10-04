// VMA API Server - Minimal control API for OMA communication
// Implements the 4-endpoint design for bidirectional tunnel architecture
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit/source/current/vma/api"
	"github.com/vexxhost/migratekit/source/current/vma/client"
	"github.com/vexxhost/migratekit/source/current/vma/services"
	"github.com/vexxhost/migratekit/source/current/vma/vmware"
)

var (
	port    = flag.Int("port", 8081, "Port for VMA control API server")
	debug   = flag.Bool("debug", false, "Enable debug logging")
	autoCBT = flag.Bool("auto-cbt", true, "Enable automatic CBT enablement before migration")
)

func main() {
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.WithFields(log.Fields{
		"version":  "1.3.2",
		"port":     *port,
		"auto_cbt": *autoCBT,
	}).Info("🚀 VMA Control API Server starting")

	// Create real VMware client for actual vCenter integration
	omaClient := client.NewClient(client.Config{
		BaseURL:     "http://10.245.246.125:8082",
		AuthToken:   "vma_test_token_abc123def456789012345678",
		ApplianceID: "vma-01",
		Timeout:     30 * time.Second,
	})

	// Configure service with CBT auto-enablement option
	serviceConfig := vmware.ServiceConfig{
		AutoCBTEnabled: *autoCBT,
	}
	vmwareClient := vmware.NewRealVMwareClientWithConfig(omaClient, serviceConfig)

	// Create progress service for robust replication tracking
	progressSvc := services.NewProgressService()
	progressHandler := api.NewProgressHandler(progressSvc)

	// Create and configure the API server
	server := api.NewVMAControlServer(*port, vmwareClient)
	
	// Register the new progress endpoint
	progressHandler.RegisterRoutes(server.GetRouter())

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("🛑 Shutdown signal received, stopping VMA API server")
		os.Exit(0)
	}()

	// Start the server
	log.WithField("address", fmt.Sprintf(":%d", *port)).Info("VMA Control API listening")
	if err := server.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start VMA API server")
	}
}
