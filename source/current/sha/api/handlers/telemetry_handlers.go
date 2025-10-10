package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/services"
)

// TelemetryHandler handles real-time telemetry updates from SBC
type TelemetryHandler struct {
	db        database.Connection
	telemetry *services.TelemetryService
}

// NewTelemetryHandler creates a new telemetry handler
func NewTelemetryHandler(db database.Connection) *TelemetryHandler {
	return &TelemetryHandler{
		db:        db,
		telemetry: services.NewTelemetryService(db),
	}
}

// ReceiveTelemetry handles POST /api/v1/telemetry/{job_type}/{job_id}
// Receives real-time progress updates from SBC during backup/replication operations
func (th *TelemetryHandler) ReceiveTelemetry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobType := vars["job_type"]
	jobID := vars["job_id"]
	
	if jobType == "" || jobID == "" {
		http.Error(w, "Missing job_type or job_id parameter", http.StatusBadRequest)
		return
	}
	
	log.WithFields(log.Fields{
		"job_type": jobType,
		"job_id":   jobID,
	}).Info("📨 RECEIVED telemetry request from SBC")
	
	var update services.TelemetryUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"job_type": jobType,
			"job_id":   jobID,
		}).Error("Failed to decode telemetry update")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	log.WithFields(log.Fields{
		"job_id":   jobID,
		"bytes":    update.BytesTransferred,
		"progress": update.ProgressPercent,
		"phase":    update.CurrentPhase,
	}).Info("📊 Decoded telemetry data")
	
	// Process the telemetry update
	if err := th.telemetry.ProcessTelemetryUpdate(r.Context(), jobType, jobID, &update); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"job_id":  jobID,
			"status":  update.Status,
			"phase":   update.CurrentPhase,
			"progress": update.ProgressPercent,
		}).Error("Failed to process telemetry update")
		http.Error(w, "Failed to process telemetry", http.StatusInternalServerError)
		return
	}
	
	log.WithFields(log.Fields{
		"job_id":           jobID,
		"status":           update.Status,
		"phase":            update.CurrentPhase,
		"progress_percent": update.ProgressPercent,
		"bytes":            update.BytesTransferred,
		"speed_bps":        update.TransferSpeedBps,
	}).Info("✅ Telemetry update processed and persisted to database")
	
	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   "Telemetry received and processed",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// RegisterRoutes registers telemetry endpoints
func (th *TelemetryHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/telemetry/{job_type}/{job_id}", th.ReceiveTelemetry).Methods("POST")
	
	log.Info("✅ Telemetry API routes registered: POST /api/v1/telemetry/{job_type}/{job_id}")
}

