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

const (
	// LoginRateLimitPath is the path this middleware gates.
	LoginRateLimitPath = "/auth/login"

	// EnvKeyLoginRateLimit is the env var controlling max attempts per IP
	// per minute.
	EnvKeyLoginRateLimit = "LOGIN_RATE_LIMIT_PER_IP"

	defaultLoginRateLimit  = 20
	defaultLoginRateWindow = 1 * time.Minute
)

type loginRateLimitConfig struct {
	maxAttempts int
	window      time.Duration
}

type ipAttemptRecord struct {
	mu         sync.Mutex
	timestamps []time.Time
}

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

// LoginRateLimit returns a MiddlewareFunc that rate-limits POST /auth/login
// by client IP. Non-login requests pass through with zero overhead. The rate
// limit is configured via the LOGIN_RATE_LIMIT_PER_IP environment variable
// (default 20/minute).
func LoginRateLimit() MiddlewareFunc {
	mw, _ := newLoginRateLimitMiddleware(os.Getenv)
	return mw
}

// LoginRateLimitWithStop is like LoginRateLimit but also returns a stop
// function that halts the background cleanup goroutine (useful in tests).
func LoginRateLimitWithStop() (MiddlewareFunc, func()) {
	return newLoginRateLimitMiddleware(os.Getenv)
}

func newLoginRateLimitMiddleware(getenv func(string) string) (MiddlewareFunc, func()) {
	cfg := parseLoginRateLimitEnv(getenv)

	var store sync.Map
	done := make(chan struct{})

	// Background sweep of stale entries.
	go func() {
		ticker := time.NewTicker(cfg.window * 2)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case now := <-ticker.C:
				cutoff := now.Add(-cfg.window)
				store.Range(func(key, value any) bool {
					rec := value.(*ipAttemptRecord)
					rec.mu.Lock()
					fresh := rec.timestamps[:0]
					for _, t := range rec.timestamps {
						if t.After(cutoff) {
							fresh = append(fresh, t)
						}
					}
					rec.timestamps = fresh
					if len(fresh) == 0 {
						rec.mu.Unlock()
						store.Delete(key)
					} else {
						rec.mu.Unlock()
					}
					return true
				})
			}
		}
	}()

	stop := func() { close(done) }

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only gate POST /auth/login.
			if r.Method != http.MethodPost || r.URL.Path != LoginRateLimitPath {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractIP(r)
			now := time.Now()
			cutoff := now.Add(-cfg.window)

			val, _ := store.LoadOrStore(ip, &ipAttemptRecord{})
			rec := val.(*ipAttemptRecord)

			rec.mu.Lock()
			// Trim expired.
			fresh := rec.timestamps[:0]
			for _, t := range rec.timestamps {
				if t.After(cutoff) {
					fresh = append(fresh, t)
				}
			}
			rec.timestamps = fresh

			if len(rec.timestamps) >= cfg.maxAttempts {
				rec.mu.Unlock()
				retryAfter := rec.timestamps[0].Add(cfg.window).Sub(now)
				if retryAfter < time.Second {
					retryAfter = time.Second
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			rec.timestamps = append(rec.timestamps, now)
			rec.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}

	return mw, stop
}

func extractIP(r *http.Request) string {
	// X-Forwarded-For first entry.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	// X-Real-IP.
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// RemoteAddr.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
