//go:build http

// login_rate_limit.go
//
// Per-IP rate limiter for POST /auth/login. This is the HTTP-layer
// brute-force gate: it prevents a single IP from hammering login across
// DIFFERENT accounts. The per-ACCOUNT lockout counter lives in espyna's
// password adapter (DB-side atomic increment); this middleware complements
// it at the network layer.
//
// Design:
//   - Sliding window: each IP tracks timestamped attempts in the current window.
//   - In-memory sync.Map -- no external dependencies for v1.
//   - Configurable via LOGIN_RATE_LIMIT_PER_IP env var (default 20/minute).
//   - Returns 429 Too Many Requests when exceeded.
//   - A background goroutine sweeps stale entries every 2x the window to
//     bound memory.
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
)

// loginRateLimitConfig holds the parsed config for the per-IP login throttle.
type loginRateLimitConfig struct {
	maxAttempts int
	window      time.Duration
}

// ipAttemptRecord tracks login attempts from a single IP within the
// sliding window. The mutex protects the timestamps slice; the
// outer sync.Map handles per-IP lookup without a global lock.
type ipAttemptRecord struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// LoginRateLimitPath is the path this middleware gates. Exported for tests.
const LoginRateLimitPath = "/auth/login"

// EnvKeyLoginRateLimit is the env var controlling max attempts per IP per minute.
const EnvKeyLoginRateLimit = "LOGIN_RATE_LIMIT_PER_IP"

// defaultLoginRateLimit is the default max login attempts per IP per window.
const defaultLoginRateLimit = 20

// defaultLoginRateWindow is the sliding window duration.
const defaultLoginRateWindow = 1 * time.Minute

// parseLoginRateLimitEnv reads LOGIN_RATE_LIMIT_PER_IP from the environment.
// Format: plain integer (attempts per minute), e.g. "20".
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
// The returned middleware is safe for concurrent use. Call the returned
// stop function to halt the background cleanup goroutine (useful in tests;
// in production the goroutine lives for the process lifetime).
func NewLoginRateLimitMiddleware() (mw func(http.Handler) http.Handler, stop func()) {
	return NewLoginRateLimitMiddlewareFromEnv(os.Getenv)
}

// NewLoginRateLimitMiddlewareFromEnv is the testable variant that accepts a
// getenv function instead of reading os.Getenv directly.
func NewLoginRateLimitMiddlewareFromEnv(getenv func(string) string) (mw func(http.Handler) http.Handler, stop func()) {
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
					// If all timestamps are older than the cutoff, delete the entry.
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

	mw = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Fast path: only gate POST /auth/login.
			if r.Method != http.MethodPost || r.URL.Path != LoginRateLimitPath {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractClientIP(r)
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
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter))
				http.Error(w, "Too many login attempts. Please try again later.", http.StatusTooManyRequests)
				return
			}

			// Record this attempt and proceed.
			rec.timestamps = append(rec.timestamps, now)
			rec.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}

	log.Printf("  LoginRateLimit: configured (%d attempts per %v per IP)", cfg.maxAttempts, cfg.window)
	return mw, stop
}

// extractClientIP returns the client's IP address from the request.
// It checks X-Forwarded-For (first entry, as set by a trusted reverse proxy),
// X-Real-Ip, and finally falls back to r.RemoteAddr.
func extractClientIP(r *http.Request) string {
	// X-Forwarded-For: client, proxy1, proxy2 -- take the leftmost.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	// X-Real-Ip (set by nginx / some proxies).
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (host:port or just host).
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port (e.g. Unix socket).
		return r.RemoteAddr
	}
	return host
}
