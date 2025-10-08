// Package middleware provides security middleware for SNA enrollment system
package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// RateLimiter implements rate limiting for SNA enrollment endpoints
type RateLimiter struct {
	mu               sync.RWMutex
	enrollmentLimits map[string]*EnrollmentLimitTracker
	challengeLimits  map[string]*ChallengeLimitTracker
	blockedIPs       map[string]*BlockedIP
	config           *RateLimitConfig
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	// Enrollment limits
	MaxEnrollmentsPerHour  int           `json:"max_enrollments_per_hour"`
	MaxValidationsPerHour  int           `json:"max_validations_per_hour"`
	MaxFailuresBeforeBlock int           `json:"max_failures_before_block"`
	BlockDuration          time.Duration `json:"block_duration"`

	// Challenge limits
	MaxChallengeAttemptsPerEnrollment int `json:"max_challenge_attempts_per_enrollment"`
	MaxChallengeRequestsPerHour       int `json:"max_challenge_requests_per_hour"`

	// Backoff configuration
	EnableExponentialBackoff bool          `json:"enable_exponential_backoff"`
	BaseBackoffDelay         time.Duration `json:"base_backoff_delay"`
	MaxBackoffDelay          time.Duration `json:"max_backoff_delay"`
}

// EnrollmentLimitTracker tracks enrollment attempts per IP
type EnrollmentLimitTracker struct {
	IP                 string               `json:"ip"`
	EnrollmentAttempts []time.Time          `json:"enrollment_attempts"`
	ValidationAttempts []time.Time          `json:"validation_attempts"`
	FailureCount       int                  `json:"failure_count"`
	LastFailure        time.Time            `json:"last_failure"`
	NextAllowedTime    time.Time            `json:"next_allowed_time"`
	Violations         []RateLimitViolation `json:"violations"`
}

// ChallengeLimitTracker tracks challenge attempts per enrollment
type ChallengeLimitTracker struct {
	EnrollmentID      string      `json:"enrollment_id"`
	IP                string      `json:"ip"`
	AttemptCount      int         `json:"attempt_count"`
	ChallengeRequests []time.Time `json:"challenge_requests"`
	FirstAttempt      time.Time   `json:"first_attempt"`
	LastAttempt       time.Time   `json:"last_attempt"`
}

// BlockedIP tracks IPs that are temporarily blocked
type BlockedIP struct {
	IP          string               `json:"ip"`
	BlockedAt   time.Time            `json:"blocked_at"`
	UnblockedAt time.Time            `json:"unblocked_at"`
	Reason      string               `json:"reason"`
	Violations  []RateLimitViolation `json:"violations"`
}

// RateLimitViolation records rate limit violations for audit
type RateLimitViolation struct {
	Timestamp     time.Time `json:"timestamp"`
	ViolationType string    `json:"violation_type"`
	Details       string    `json:"details"`
	Endpoint      string    `json:"endpoint"`
}

// NewRateLimiter creates a new rate limiter with security-focused defaults
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		enrollmentLimits: make(map[string]*EnrollmentLimitTracker),
		challengeLimits:  make(map[string]*ChallengeLimitTracker),
		blockedIPs:       make(map[string]*BlockedIP),
		config: &RateLimitConfig{
			// Conservative limits for internet exposure
			MaxEnrollmentsPerHour:  5,              // Very low to prevent abuse
			MaxValidationsPerHour:  10,             // Allow some retries
			MaxFailuresBeforeBlock: 20,             // Block persistent attackers
			BlockDuration:          24 * time.Hour, // 24-hour blocks

			// Challenge security
			MaxChallengeAttemptsPerEnrollment: 3,  // Limit per enrollment
			MaxChallengeRequestsPerHour:       50, // Reasonable for legitimate use

			// Exponential backoff for failed attempts
			EnableExponentialBackoff: true,
			BaseBackoffDelay:         1 * time.Second,
			MaxBackoffDelay:          60 * time.Second,
		},
	}
}

// EnrollmentRateLimit middleware for enrollment endpoints
func (rl *RateLimiter) EnrollmentRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.extractClientIP(r)

		// Check if IP is blocked
		if rl.isIPBlocked(ip) {
			rl.logSecurityViolation(ip, "blocked_ip_access", fmt.Sprintf("Blocked IP attempted access: %s", r.URL.Path))
			rl.writeSecurityError(w, http.StatusTooManyRequests, "IP temporarily blocked", "Too many failed attempts")
			return
		}

		// Check enrollment rate limits
		if !rl.checkEnrollmentLimit(ip, r.URL.Path) {
			rl.recordRateLimitViolation(ip, "enrollment_rate_limit", r.URL.Path)
			rl.writeSecurityError(w, http.StatusTooManyRequests, "Rate limit exceeded", "Too many enrollment attempts")
			return
		}

		// Record successful request
		rl.recordEnrollmentAttempt(ip)

		// Apply exponential backoff if enabled
		if rl.config.EnableExponentialBackoff {
			rl.applyExponentialBackoff(ip, w)
		}

		next.ServeHTTP(w, r)
	})
}

// ChallengeRateLimit middleware for challenge verification endpoints
func (rl *RateLimiter) ChallengeRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.extractClientIP(r)

		// Extract enrollment ID from request
		enrollmentID := rl.extractEnrollmentID(r)

		// Check if IP is blocked
		if rl.isIPBlocked(ip) {
			rl.logSecurityViolation(ip, "blocked_ip_challenge", fmt.Sprintf("Blocked IP attempted challenge: %s", enrollmentID))
			rl.writeSecurityError(w, http.StatusTooManyRequests, "IP temporarily blocked", "Too many failed attempts")
			return
		}

		// Check challenge-specific rate limits
		if !rl.checkChallengeLimit(ip, enrollmentID) {
			rl.recordRateLimitViolation(ip, "challenge_rate_limit", r.URL.Path)
			rl.writeSecurityError(w, http.StatusTooManyRequests, "Challenge rate limit exceeded", "Too many challenge attempts")
			return
		}

		// Record challenge attempt
		rl.recordChallengeAttempt(ip, enrollmentID)

		next.ServeHTTP(w, r)
	})
}

// extractClientIP extracts the real client IP from request
func (rl *RateLimiter) extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxy setups)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to direct connection IP
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// extractEnrollmentID extracts enrollment ID from request
func (rl *RateLimiter) extractEnrollmentID(r *http.Request) string {
	// Try query parameter first
	if enrollmentID := r.URL.Query().Get("enrollment_id"); enrollmentID != "" {
		return enrollmentID
	}

	// Try JSON body
	var req struct {
		EnrollmentID string `json:"enrollment_id"`
	}

	if r.Body != nil {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err == nil {
			return req.EnrollmentID
		}
	}

	return ""
}

// checkEnrollmentLimit validates enrollment rate limits for IP
func (rl *RateLimiter) checkEnrollmentLimit(ip string, endpoint string) bool {
	rl.mu.RLock()
	tracker, exists := rl.enrollmentLimits[ip]
	rl.mu.RUnlock()

	if !exists {
		return true // No limits recorded yet
	}

	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	// Count recent enrollment attempts
	recentEnrollments := 0
	for _, attempt := range tracker.EnrollmentAttempts {
		if attempt.After(hourAgo) {
			recentEnrollments++
		}
	}

	// Count recent validations
	recentValidations := 0
	for _, attempt := range tracker.ValidationAttempts {
		if attempt.After(hourAgo) {
			recentValidations++
		}
	}

	// Check limits based on endpoint
	if strings.Contains(endpoint, "enroll") && recentEnrollments >= rl.config.MaxEnrollmentsPerHour {
		return false
	}

	if recentValidations >= rl.config.MaxValidationsPerHour {
		return false
	}

	// Check if still in backoff period
	if now.Before(tracker.NextAllowedTime) {
		return false
	}

	return true
}

// checkChallengeLimit validates challenge rate limits
func (rl *RateLimiter) checkChallengeLimit(ip string, enrollmentID string) bool {
	rl.mu.RLock()
	tracker, exists := rl.challengeLimits[enrollmentID]
	rl.mu.RUnlock()

	if !exists {
		return true // No limits recorded yet
	}

	// Check per-enrollment limits
	if tracker.AttemptCount >= rl.config.MaxChallengeAttemptsPerEnrollment {
		return false
	}

	// Check per-IP hourly limits
	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	recentRequests := 0
	for _, request := range tracker.ChallengeRequests {
		if request.After(hourAgo) {
			recentRequests++
		}
	}

	return recentRequests < rl.config.MaxChallengeRequestsPerHour
}

// isIPBlocked checks if an IP is currently blocked
func (rl *RateLimiter) isIPBlocked(ip string) bool {
	rl.mu.RLock()
	blocked, exists := rl.blockedIPs[ip]
	rl.mu.RUnlock()

	if !exists {
		return false
	}

	// Check if block has expired
	if time.Now().After(blocked.UnblockedAt) {
		rl.mu.Lock()
		delete(rl.blockedIPs, ip)
		rl.mu.Unlock()
		return false
	}

	return true
}

// recordEnrollmentAttempt records an enrollment attempt for rate limiting
func (rl *RateLimiter) recordEnrollmentAttempt(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if tracker, exists := rl.enrollmentLimits[ip]; exists {
		tracker.EnrollmentAttempts = append(tracker.EnrollmentAttempts, time.Now())
	} else {
		rl.enrollmentLimits[ip] = &EnrollmentLimitTracker{
			IP:                 ip,
			EnrollmentAttempts: []time.Time{time.Now()},
			ValidationAttempts: []time.Time{},
		}
	}
}

// recordChallengeAttempt records a challenge attempt
func (rl *RateLimiter) recordChallengeAttempt(ip string, enrollmentID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if tracker, exists := rl.challengeLimits[enrollmentID]; exists {
		tracker.AttemptCount++
		tracker.ChallengeRequests = append(tracker.ChallengeRequests, time.Now())
		tracker.LastAttempt = time.Now()
	} else {
		rl.challengeLimits[enrollmentID] = &ChallengeLimitTracker{
			EnrollmentID:      enrollmentID,
			IP:                ip,
			AttemptCount:      1,
			ChallengeRequests: []time.Time{time.Now()},
			FirstAttempt:      time.Now(),
			LastAttempt:       time.Now(),
		}
	}
}

// recordRateLimitViolation records a rate limit violation
func (rl *RateLimiter) recordRateLimitViolation(ip string, violationType string, endpoint string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	violation := RateLimitViolation{
		Timestamp:     time.Now(),
		ViolationType: violationType,
		Details:       fmt.Sprintf("Rate limit exceeded for %s", violationType),
		Endpoint:      endpoint,
	}

	if tracker, exists := rl.enrollmentLimits[ip]; exists {
		tracker.FailureCount++
		tracker.LastFailure = time.Now()
		tracker.Violations = append(tracker.Violations, violation)

		// Check if IP should be blocked
		if tracker.FailureCount >= rl.config.MaxFailuresBeforeBlock {
			rl.blockIP(ip, "excessive_rate_limit_violations", tracker.Violations)
		}

		// Apply exponential backoff
		if rl.config.EnableExponentialBackoff {
			backoffDelay := rl.calculateBackoffDelay(tracker.FailureCount)
			tracker.NextAllowedTime = time.Now().Add(backoffDelay)
		}
	}

	rl.logSecurityViolation(ip, violationType, fmt.Sprintf("Rate limit violation: %s on %s", violationType, endpoint))
}

// blockIP blocks an IP address for the configured duration
func (rl *RateLimiter) blockIP(ip string, reason string, violations []RateLimitViolation) {
	blockedUntil := time.Now().Add(rl.config.BlockDuration)

	rl.blockedIPs[ip] = &BlockedIP{
		IP:          ip,
		BlockedAt:   time.Now(),
		UnblockedAt: blockedUntil,
		Reason:      reason,
		Violations:  violations,
	}

	log.WithFields(log.Fields{
		"ip":              ip,
		"reason":          reason,
		"blocked_until":   blockedUntil,
		"violation_count": len(violations),
	}).Warn("🚫 IP address blocked for security violations")
}

// calculateBackoffDelay calculates exponential backoff delay
func (rl *RateLimiter) calculateBackoffDelay(failureCount int) time.Duration {
	// Exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s, 60s (max)
	delay := rl.config.BaseBackoffDelay

	for i := 1; i < failureCount && delay < rl.config.MaxBackoffDelay; i++ {
		delay *= 2
	}

	if delay > rl.config.MaxBackoffDelay {
		delay = rl.config.MaxBackoffDelay
	}

	return delay
}

// applyExponentialBackoff applies backoff delay if needed
func (rl *RateLimiter) applyExponentialBackoff(ip string, w http.ResponseWriter) {
	rl.mu.RLock()
	tracker, exists := rl.enrollmentLimits[ip]
	rl.mu.RUnlock()

	if !exists || tracker.FailureCount == 0 {
		return
	}

	if time.Now().Before(tracker.NextAllowedTime) {
		remainingDelay := time.Until(tracker.NextAllowedTime)

		log.WithFields(log.Fields{
			"ip":              ip,
			"remaining_delay": remainingDelay,
			"failure_count":   tracker.FailureCount,
		}).Debug("🐌 Applying exponential backoff delay")

		// Add delay header for client information
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", remainingDelay.Seconds()))
	}
}

// logSecurityViolation logs security violations for monitoring
func (rl *RateLimiter) logSecurityViolation(ip string, violationType string, details string) {
	log.WithFields(log.Fields{
		"security_event": "rate_limit_violation",
		"ip":             ip,
		"violation_type": violationType,
		"details":        details,
		"timestamp":      time.Now(),
	}).Warn("🚨 Security violation detected")
}

// writeSecurityError writes a security-related error response
func (rl *RateLimiter) writeSecurityError(w http.ResponseWriter, statusCode int, error string, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":         error,
		"details":       details,
		"security_info": "This request has been logged for security monitoring",
		"timestamp":     time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

// GetStatistics returns rate limiting statistics for monitoring
func (rl *RateLimiter) GetStatistics() *RateLimitStatistics {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := &RateLimitStatistics{
		TrackedIPs:        len(rl.enrollmentLimits),
		BlockedIPs:        len(rl.blockedIPs),
		ActiveEnrollments: len(rl.challengeLimits),
		TotalViolations:   0,
		LastViolationTime: time.Time{},
	}

	// Count total violations
	for _, tracker := range rl.enrollmentLimits {
		stats.TotalViolations += len(tracker.Violations)
		for _, violation := range tracker.Violations {
			if violation.Timestamp.After(stats.LastViolationTime) {
				stats.LastViolationTime = violation.Timestamp
			}
		}
	}

	return stats
}

// RateLimitStatistics represents rate limiting statistics
type RateLimitStatistics struct {
	TrackedIPs        int       `json:"tracked_ips"`
	BlockedIPs        int       `json:"blocked_ips"`
	ActiveEnrollments int       `json:"active_enrollments"`
	TotalViolations   int       `json:"total_violations"`
	LastViolationTime time.Time `json:"last_violation_time"`
}

// CleanupExpiredData removes old tracking data to prevent memory leaks
func (rl *RateLimiter) CleanupExpiredData() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	dayAgo := now.Add(-24 * time.Hour)

	// Clean up old enrollment trackers
	for ip, tracker := range rl.enrollmentLimits {
		// Remove attempts older than 24 hours
		var recentAttempts []time.Time
		for _, attempt := range tracker.EnrollmentAttempts {
			if attempt.After(dayAgo) {
				recentAttempts = append(recentAttempts, attempt)
			}
		}
		tracker.EnrollmentAttempts = recentAttempts

		// Remove tracker if no recent activity
		if len(recentAttempts) == 0 && tracker.LastFailure.Before(dayAgo) {
			delete(rl.enrollmentLimits, ip)
		}
	}

	// Clean up expired blocks
	for ip, blocked := range rl.blockedIPs {
		if now.After(blocked.UnblockedAt) {
			delete(rl.blockedIPs, ip)
		}
	}

	// Clean up old challenge trackers
	for enrollmentID, tracker := range rl.challengeLimits {
		if tracker.LastAttempt.Before(dayAgo) {
			delete(rl.challengeLimits, enrollmentID)
		}
	}
}






