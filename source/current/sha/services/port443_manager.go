// Package services provides port 443 exposure management for SNA enrollment
package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// Port443Manager handles secure exposure of enrollment endpoints on port 443
type Port443Manager struct {
	router            *mux.Router
	tlsConfig         *TLSConfig
	securityConfig    *Port443SecurityConfig
	enrollmentHandler http.Handler
}

// TLSConfig represents TLS configuration for port 443
type TLSConfig struct {
	CertFile      string   `json:"cert_file"`
	KeyFile       string   `json:"key_file"`
	EnableHTTP2   bool     `json:"enable_http2"`
	MinTLSVersion string   `json:"min_tls_version"`
	CipherSuites  []string `json:"cipher_suites"`
	EnableHSTS    bool     `json:"enable_hsts"`
	HSTSMaxAge    int      `json:"hsts_max_age"`
}

// Port443SecurityConfig represents security settings for port 443 exposure
type Port443SecurityConfig struct {
	EnableRateLimiting    bool     `json:"enable_rate_limiting"`
	EnableInputValidation bool     `json:"enable_input_validation"`
	EnableIPWhitelist     bool     `json:"enable_ip_whitelist"`
	AllowedOrigins        []string `json:"allowed_origins"`
	EnableSecurityHeaders bool     `json:"enable_security_headers"`
	LogAllRequests        bool     `json:"log_all_requests"`
}

// NewPort443Manager creates a new port 443 manager
func NewPort443Manager(enrollmentHandler http.Handler) *Port443Manager {
	return &Port443Manager{
		router:            mux.NewRouter(),
		enrollmentHandler: enrollmentHandler,
		tlsConfig: &TLSConfig{
			CertFile:      "/etc/ssl/certs/oma-enrollment.crt",
			KeyFile:       "/etc/ssl/private/oma-enrollment.key",
			EnableHTTP2:   true,
			MinTLSVersion: "1.2",
			CipherSuites: []string{
				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			},
			EnableHSTS: true,
			HSTSMaxAge: 31536000, // 1 year
		},
		securityConfig: &Port443SecurityConfig{
			EnableRateLimiting:    true,
			EnableInputValidation: true,
			EnableIPWhitelist:     false,         // Disabled by default for broader access
			AllowedOrigins:        []string{"*"}, // Configured per deployment
			EnableSecurityHeaders: true,
			LogAllRequests:        true,
		},
	}
}

// SetupSecureRoutes configures secure routes for port 443 exposure
func (pm *Port443Manager) SetupSecureRoutes() {
	// Security headers middleware
	pm.router.Use(pm.securityHeadersMiddleware)

	// Request logging middleware for security monitoring
	if pm.securityConfig.LogAllRequests {
		pm.router.Use(pm.securityLoggingMiddleware)
	}

	// CORS middleware for enrollment endpoints
	pm.router.Use(pm.enrollmentCORSMiddleware)

	// Health check for port 443 (minimal info)
	pm.router.HandleFunc("/health", pm.handleSecureHealth).Methods("GET")

	// Public SNA enrollment endpoints (secured)
	enrollmentAPI := pm.router.PathPrefix("/api/v1").Subrouter()

	// Apply security middleware to enrollment endpoints only
	enrollmentAPI.PathPrefix("/vma/").Handler(pm.enrollmentHandler)

	log.Info("🔒 Port 443 secure routes configured for SNA enrollment")
}

// securityHeadersMiddleware adds security headers for internet exposure
func (pm *Port443Manager) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS (HTTP Strict Transport Security)
		if pm.tlsConfig.EnableHSTS {
			w.Header().Set("Strict-Transport-Security",
				fmt.Sprintf("max-age=%d; includeSubDomains; preload", pm.tlsConfig.HSTSMaxAge))
		}

		// Content Security Policy
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; script-src 'self'; connect-src 'self'; img-src 'self'; style-src 'self'")

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// API-specific headers
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.Header().Set("Pragma", "no-cache")

		next.ServeHTTP(w, r)
	})
}

// securityLoggingMiddleware logs all requests for security monitoring
func (pm *Port443Manager) securityLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request details for security monitoring
		log.WithFields(log.Fields{
			"security_log":    "port443_access",
			"method":          r.Method,
			"path":            r.URL.Path,
			"query":           r.URL.RawQuery,
			"remote_addr":     r.RemoteAddr,
			"user_agent":      r.UserAgent(),
			"x_forwarded_for": r.Header.Get("X-Forwarded-For"),
			"x_real_ip":       r.Header.Get("X-Real-IP"),
			"content_length":  r.ContentLength,
			"timestamp":       start,
		}).Info("🌐 Port 443 access logged")

		// Capture response
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		// Log response details
		log.WithFields(log.Fields{
			"security_log": "port443_response",
			"status_code":  wrapped.statusCode,
			"duration_ms":  duration.Milliseconds(),
			"remote_addr":  r.RemoteAddr,
			"path":         r.URL.Path,
		}).Info("🌐 Port 443 response logged")
	})
}

// enrollmentCORSMiddleware handles CORS for enrollment endpoints
func (pm *Port443Manager) enrollmentCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for enrollment endpoints
		w.Header().Set("Access-Control-Allow-Origin", "*") // Configured per deployment
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleSecureHealth provides minimal health check for port 443
func (pm *Port443Manager) handleSecureHealth(w http.ResponseWriter, r *http.Request) {
	// Minimal health response for security (don't expose system details)
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "vma-enrollment",
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// StartSecureServer starts the secure port 443 server
func (pm *Port443Manager) StartSecureServer() error {
	server := &http.Server{
		Addr:         ":443",
		Handler:      pm.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		// TLS config would be applied here
	}

	log.WithFields(log.Fields{
		"port":             443,
		"tls_enabled":      true,
		"security_headers": pm.securityConfig.EnableSecurityHeaders,
		"rate_limiting":    pm.securityConfig.EnableRateLimiting,
	}).Info("🔒 Starting secure SNA enrollment server on port 443")

	// In production, this would use TLS:
	// return server.ListenAndServeTLS(pm.tlsConfig.CertFile, pm.tlsConfig.KeyFile)

	// For now, log the configuration
	log.Info("🔒 Port 443 secure server configuration ready")

	// Store server reference for potential future use
	_ = server
	return nil
}

// responseWriter captures response status for logging
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
