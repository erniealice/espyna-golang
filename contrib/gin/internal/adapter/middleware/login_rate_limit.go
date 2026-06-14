//go:build gin

// login_rate_limit.go — per-IP rate limiter for POST /auth/login.
//
// Mirrors the vanilla net/http implementation
// (apps/service-admin/internal/infrastructure/input/http/middleware/login_rate_limit.go).
//
// This is the HTTP-layer brute-force gate: it prevents a single IP from
// hammering login across DIFFERENT accounts. The per-ACCOUNT lockout counter
// lives in espyna's password adapter (DB-side atomic increment); this middleware
// complements it at the network layer.
//
// Design:
//   - Sliding window: each IP tracks timestamped attempts in the current window.
//   - In-memory sync.Map — no external dependencies for v1.
//   - Configurable via LOGIN_RATE_LIMIT_PER_IP env var (default 20/minute).
//   - Returns 429 Too Many Requests when exceeded.
//   - Background goroutine sweeps stale entries every 2x the window.
//   - Only applies to POST /auth/login; all other paths pass through.
package middleware

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// loginRateLimitConfig holds the parsed config for the per-IP login throttle.
type loginRateLimitConfig struct {
	maxAttempts int
	window      time.Duration
}

// ipAttemptRecord tracks login attempts from a single IP within the sliding
// window.
type ipAttemptRecord struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// LoginRateLimitPath is the path this middleware gates.
const LoginRateLimitPath = "/auth/login"

// EnvKeyLoginRateLimit is the env var controlling max attempts per IP per minute.
const EnvKeyLoginRateLimit = "LOGIN_RATE_LIMIT_PER_IP"

const (
	defaultLoginRateLimit  = 20
	defaultLoginRateWindow = 1 * time.Minute
)

// parseLoginRateLimitEnv reads LOGIN_RATE_LIMIT_PER_IP from the environment.
// Returns the default (20/minute) on missing, empty, or unparseable values.
func parseLoginRateLimitEnv(getenv func(string) string) loginRateLimitConfig {
	cfg := loginRateLimitConfig{
		maxAttempts: defaultLoginRateLimit,
		window:      defaultLoginRateWindow,
	}
	raw := strings.TrimSpace(getenv(EnvKeyLoginRateLimit))
	if raw == "" {
		return cfg
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		log.Printf("[LOGIN_RATE_LIMIT] ignoring invalid %s=%q (must be a positive integer); using default %d/min",
			EnvKeyLoginRateLimit, raw, defaultLoginRateLimit)
		return cfg
	}
	cfg.maxAttempts = n
	return cfg
}

// NewLoginRateLimitMiddleware creates a per-IP rate limiter for POST /auth/login.
// It reads config from the environment via os.Getenv. Non-login requests pass
// through with zero overhead (a path+method check only).
//
// The returned stop function halts the background cleanup goroutine (useful in
// tests; in production the goroutine lives for the process lifetime).
func NewLoginRateLimitMiddleware() (mw gin.HandlerFunc, stop func()) {
	return NewLoginRateLimitMiddlewareFromEnv(os.Getenv)
}

// NewLoginRateLimitMiddlewareFromEnv is the testable variant that accepts a
// getenv function instead of reading os.Getenv directly.
func NewLoginRateLimitMiddlewareFromEnv(getenv func(string) string) (mw gin.HandlerFunc, stop func()) {
	cfg := parseLoginRateLimitEnv(getenv)

	var store sync.Map // IP string -> *ipAttemptRecord
	done := make(chan struct{})

	// Background sweeper: evicts entries whose newest timestamp is older
	// than 2x the window. Runs every window duration.
	go func() {
		ticker := time.NewTicker(cfg.window)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case now := <-ticker.C:
				cutoff := now.Add(-2 * cfg.window)
				store.Range(func(key, value any) bool {
					rec := value.(*ipAttemptRecord)
					rec.mu.Lock()
					allStale := true
					for _, t := range rec.timestamps {
						if t.After(cutoff) {
							allStale = false
							break
						}
					}
					rec.mu.Unlock()
					if allStale {
						store.Delete(key)
					}
					return true
				})
			}
		}
	}()

	stop = func() { close(done) }

	mw = func(c *gin.Context) {
		// Fast path: only gate POST /auth/login.
		if c.Request.Method != http.MethodPost || c.Request.URL.Path != LoginRateLimitPath {
			c.Next()
			return
		}

		ip := extractClientIP(c)
		now := time.Now()
		windowStart := now.Add(-cfg.window)

		// Load-or-store the per-IP record.
		val, _ := store.LoadOrStore(ip, &ipAttemptRecord{})
		rec := val.(*ipAttemptRecord)

		rec.mu.Lock()
		// Prune timestamps outside the current window.
		pruned := rec.timestamps[:0]
		for _, t := range rec.timestamps {
			if t.After(windowStart) {
				pruned = append(pruned, t)
			}
		}
		rec.timestamps = pruned

		if len(rec.timestamps) >= cfg.maxAttempts {
			rec.mu.Unlock()
			log.Printf("[LOGIN_RATE_LIMIT] 429 for IP %s (%d attempts in last %v)",
				ip, len(pruned), cfg.window)
			retryAfter := cfg.window.Seconds()
			c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter))
			c.String(http.StatusTooManyRequests, "Too many login attempts. Please try again later.")
			c.Abort()
			return
		}

		// Record this attempt and proceed.
		rec.timestamps = append(rec.timestamps, now)
		rec.mu.Unlock()

		c.Next()
	}

	log.Printf("  LoginRateLimit: configured (%d attempts per %v per IP)", cfg.maxAttempts, cfg.window)
	return mw, stop
}

// extractClientIP returns the client's IP address from the request.
// Checks X-Forwarded-For (first entry), X-Real-Ip, and falls back to
// r.RemoteAddr. Trusts X-Forwarded-For unconditionally (appropriate when
// behind a trusted reverse proxy).
func extractClientIP(c *gin.Context) string {
	// Gin's c.ClientIP() already handles X-Forwarded-For / X-Real-Ip, but we
	// replicate the vanilla logic for exact behavioral parity.
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	if xri := c.GetHeader("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}

	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}
