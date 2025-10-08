// Package main provides a port 443 proxy for SNA enrollment endpoints
// This allows NEW SNAs to reach enrollment API without existing tunnel
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var (
	port       = flag.Int("port", 443, "Port for enrollment proxy (443 for production)")
	backend    = flag.String("backend", "http://localhost:8082", "Backend SHA API server")
	debug      = flag.Bool("debug", false, "Enable debug logging")
	enableTLS  = flag.Bool("tls", false, "Enable TLS (requires certificates)")
	certFile   = flag.String("cert", "/etc/ssl/certs/oma-enrollment.crt", "TLS certificate file")
	keyFile    = flag.String("key", "/etc/ssl/private/oma-enrollment.key", "TLS private key file")
)

func main() {
	flag.Parse()

	// Parse backend URL
	backendURL, err := url.Parse(*backend)
	if err != nil {
		log.Fatalf("Invalid backend URL: %v", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Custom director to only allow enrollment endpoints
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		
		// Log all requests for security monitoring
		if *debug {
			log.Printf("🌐 Port 443 request: %s %s from %s", req.Method, req.URL.Path, req.RemoteAddr)
		}
	}

	// Create HTTP server with security timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      createSecureHandler(proxy),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("🔒 Starting SNA enrollment proxy on port %d", *port)
	log.Printf("🔗 Proxying to backend: %s", *backend)
	log.Printf("🛡️ TLS enabled: %v", *enableTLS)

	// Start server
	if *enableTLS {
		log.Printf("🔐 Starting HTTPS server with TLS certificates")
		if err := server.ListenAndServeTLS(*certFile, *keyFile); err != nil {
			log.Fatalf("Failed to start HTTPS server: %v", err)
		}
	} else {
		log.Printf("⚠️  Starting HTTP server (TLS disabled for testing)")
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}
}

// createSecureHandler wraps the proxy with security middleware
func createSecureHandler(proxy *httputil.ReverseProxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		// Add CORS headers for enrollment
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Only allow enrollment endpoints
		if !isEnrollmentEndpoint(r.URL.Path) {
			log.Printf("🚫 Blocked non-enrollment request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			http.Error(w, "Endpoint not available on this port", http.StatusNotFound)
			return
		}

		// Log enrollment access
		log.Printf("🔐 Enrollment API access: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		
		// Proxy to backend
		proxy.ServeHTTP(w, r)
	})
}

// isEnrollmentEndpoint checks if the path is an allowed enrollment endpoint
func isEnrollmentEndpoint(path string) bool {
	allowedPaths := []string{
		"/api/v1/vma/enroll",
		"/api/v1/vma/enroll/verify", 
		"/api/v1/vma/enroll/result",
		"/health",
	}
	
	for _, allowedPath := range allowedPaths {
		if path == allowedPath {
			return true
		}
	}
	
	return false
}






