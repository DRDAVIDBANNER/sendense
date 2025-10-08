// Package middleware provides input validation for SNA enrollment security
package middleware

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	log "github.com/sirupsen/logrus"
)

// InputValidator implements strict input validation for security
type InputValidator struct {
	// Validation patterns
	pairingCodePattern *regexp.Regexp
	versionPattern     *regexp.Regexp
	sshKeyPattern      *regexp.Regexp

	// Security patterns to detect attacks
	sqlInjectionPatterns     []*regexp.Regexp
	xssPatterns              []*regexp.Regexp
	commandInjectionPatterns []*regexp.Regexp
}

// NewInputValidator creates a new input validator with security patterns
func NewInputValidator() *InputValidator {
	return &InputValidator{
		// Valid input patterns
		pairingCodePattern: regexp.MustCompile(`^[ABCDEFGHJKMNPQRSTVWXYZ23456789]{4}-[ABCDEFGHJKMNPQRSTVWXYZ23456789]{4}-[ABCDEFGHJKMNPQRSTVWXYZ23456789]{4}$`),
		versionPattern:     regexp.MustCompile(`^v\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?$`),
		sshKeyPattern:      regexp.MustCompile(`^ssh-ed25519 [A-Za-z0-9+/]+=* .+$`),

		// Attack detection patterns
		sqlInjectionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|exec|script)`),
			regexp.MustCompile(`[';\"]\s*(or|and)\s*[';\"]*\s*[=<>]`),
			regexp.MustCompile(`-{2,}|\*{2,}|/{2,}`),
		},

		xssPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)<script[^>]*>`),
			regexp.MustCompile(`(?i)javascript:`),
			regexp.MustCompile(`(?i)on\w+\s*=`),
		},

		commandInjectionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`[;&|]{1,2}`),
			regexp.MustCompile(`\$\([^)]*\)`),
			regexp.MustCompile("`[^`]*`"),
		},
	}
}

// ValidateEnrollmentRequest validates SNA enrollment request
func (iv *InputValidator) ValidateEnrollmentRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PairingCode    string `json:"pairing_code"`
			SNAPublicKey   string `json:"vma_public_key"`
			SNAName        string `json:"vma_name"`
			SNAVersion     string `json:"vma_version"`
			SNAFingerprint string `json:"vma_fingerprint"`
		}

		// Decode request for validation
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			iv.writeValidationError(w, "Invalid JSON format", err.Error())
			return
		}

		// Validate pairing code format
		if !iv.validatePairingCode(req.PairingCode) {
			iv.logSuspiciousInput(r.RemoteAddr, "invalid_pairing_code", req.PairingCode)
			iv.writeValidationError(w, "Invalid pairing code format", "Must be XXXX-XXXX-XXXX format")
			return
		}

		// Validate SSH public key
		if !iv.validateSSHKey(req.SNAPublicKey) {
			iv.logSuspiciousInput(r.RemoteAddr, "invalid_ssh_key", "SSH key validation failed")
			iv.writeValidationError(w, "Invalid SSH public key", "Must be valid Ed25519 SSH public key")
			return
		}

		// Validate SNA name
		if !iv.validateVMAName(req.SNAName) {
			iv.logSuspiciousInput(r.RemoteAddr, "invalid_vma_name", req.SNAName)
			iv.writeValidationError(w, "Invalid SNA name", "Must be alphanumeric with spaces only, max 64 characters")
			return
		}

		// Validate version string
		if !iv.validateVersion(req.SNAVersion) {
			iv.logSuspiciousInput(r.RemoteAddr, "invalid_version", req.SNAVersion)
			iv.writeValidationError(w, "Invalid version format", "Must be semantic version format (v1.2.3)")
			return
		}

		// Check for attack patterns in all fields
		allFields := []string{req.PairingCode, req.SNAPublicKey, req.SNAName, req.SNAVersion, req.SNAFingerprint}
		for _, field := range allFields {
			if iv.detectAttackPatterns(field) {
				iv.logSecurityThreat(r.RemoteAddr, "attack_pattern_detected", field)
				iv.writeValidationError(w, "Request rejected", "Suspicious content detected")
				return
			}
		}

		log.WithFields(log.Fields{
			"ip":          r.RemoteAddr,
			"vma_name":    req.SNAName,
			"vma_version": req.SNAVersion,
		}).Debug("✅ Enrollment request validation passed")

		next.ServeHTTP(w, r)
	})
}

// validatePairingCode validates pairing code format and characters
func (iv *InputValidator) validatePairingCode(code string) bool {
	// Check basic format
	if !iv.pairingCodePattern.MatchString(code) {
		return false
	}

	// Additional security checks
	if len(code) != 14 {
		return false
	}

	// Ensure proper dash placement
	if code[4] != '-' || code[9] != '-' {
		return false
	}

	return true
}

// validateSSHKey validates SSH Ed25519 public key format and security
func (iv *InputValidator) validateSSHKey(key string) bool {
	// Check basic SSH key format
	if !iv.sshKeyPattern.MatchString(key) {
		return false
	}

	// Ensure it starts with ssh-ed25519
	if !strings.HasPrefix(key, "ssh-ed25519 ") {
		return false
	}

	// Check key length constraints
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return false
	}

	// Validate base64 key data length (Ed25519 keys have specific length)
	keyData := parts[1]
	if len(keyData) < 50 || len(keyData) > 80 {
		return false
	}

	// Check for suspicious characters in comment
	if len(parts) > 2 {
		comment := strings.Join(parts[2:], " ")
		if iv.detectAttackPatterns(comment) {
			return false
		}
	}

	return true
}

// validateVMAName validates SNA name for security and format
func (iv *InputValidator) validateVMAName(name string) bool {
	// Length check
	if len(name) == 0 || len(name) > 64 {
		return false
	}

	// Character validation: alphanumeric, spaces, hyphens, underscores only
	for _, char := range name {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) &&
			char != ' ' && char != '-' && char != '_' {
			return false
		}
	}

	// Check for suspicious patterns
	lowerName := strings.ToLower(name)
	suspiciousWords := []string{"admin", "root", "system", "test", "debug", "hack", "exploit"}
	for _, word := range suspiciousWords {
		if strings.Contains(lowerName, word) {
			// Allow common words but log for monitoring
			log.WithField("vma_name", name).Debug("SNA name contains monitoring keyword")
		}
	}

	return true
}

// validateVersion validates semantic version format
func (iv *InputValidator) validateVersion(version string) bool {
	if version == "" {
		return true // Optional field
	}

	return iv.versionPattern.MatchString(version)
}

// detectAttackPatterns checks for common attack patterns
func (iv *InputValidator) detectAttackPatterns(input string) bool {
	// Check SQL injection patterns
	for _, pattern := range iv.sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	// Check XSS patterns
	for _, pattern := range iv.xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	// Check command injection patterns
	for _, pattern := range iv.commandInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	// Check for encoded attack attempts
	if iv.detectEncodedAttacks(input) {
		return true
	}

	return false
}

// detectEncodedAttacks detects URL-encoded or base64-encoded attacks
func (iv *InputValidator) detectEncodedAttacks(input string) bool {
	// Common URL-encoded attack patterns
	encodedPatterns := []string{
		"%3Cscript",   // <script
		"%22%3E",      // ">
		"%27%20or%20", // ' or
		"%3B",         // ;
		"%7C",         // |
		"%26%26",      // &&
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range encodedPatterns {
		if strings.Contains(lowerInput, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// logSuspiciousInput logs suspicious input attempts
func (iv *InputValidator) logSuspiciousInput(ip string, inputType string, value string) {
	log.WithFields(log.Fields{
		"security_event": "suspicious_input",
		"ip":             ip,
		"input_type":     inputType,
		"value":          value,
		"timestamp":      time.Now(),
	}).Warn("🚨 Suspicious input detected")
}

// logSecurityThreat logs potential security threats
func (iv *InputValidator) logSecurityThreat(ip string, threatType string, payload string) {
	log.WithFields(log.Fields{
		"security_event": "potential_attack",
		"ip":             ip,
		"threat_type":    threatType,
		"payload":        payload,
		"timestamp":      time.Now(),
	}).Error("🚨 Potential security attack detected")
}

// writeValidationError writes input validation error response
func (iv *InputValidator) writeValidationError(w http.ResponseWriter, error string, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := map[string]interface{}{
		"error":         error,
		"details":       details,
		"security_info": "Input validation failed - request logged for security monitoring",
		"timestamp":     time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}






