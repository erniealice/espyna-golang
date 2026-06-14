//go:build fiber

package middleware

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	// loginRateLimitPath is the path this middleware gates.
	loginRateLimitPath = "/auth/login"

	// envKeyLoginRateLimit is the env var controlling max attempts per IP
	// per minute.
	envKeyLoginRateLimit = "LOGIN_RATE_LIMIT_PER_IP"

	defaultLoginRateLimitVal    = 20
	defaultLoginRateLimitWindow = 1 * time.Minute
)

type fiberLoginRateLimitConfig struct {
	maxAttempts int
	window      time.Duration
}

type fiberIPAttemptRecord struct {
	mu         sync.Mutex
	timestamps []time.Time
}

func parseFiberLoginRateLimitEnv(getenv func(string) string) fiberLoginRateLimitConfig {
	cfg := fiberLoginRateLimitConfig{
		maxAttempts: defaultLoginRateLimitVal,
		window:      defaultLoginRateLimitWindow,
	}
	raw := strings.TrimSpace(getenv(envKeyLoginRateLimit))
	if raw == "" {
		return cfg
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		log.Printf("[LOGIN_RATE_LIMIT] ignoring invalid %s=%q (must be a positive integer); using default %d/min",
			envKeyLoginRateLimit, raw, defaultLoginRateLimitVal)
		return cfg
	}
	cfg.maxAttempts = n
	return cfg
}

// LoginRateLimit returns a Fiber middleware that rate-limits POST /auth/login
// by client IP. Non-login requests pass through with zero overhead. The rate
// limit is configured via the LOGIN_RATE_LIMIT_PER_IP environment variable
// (default 20/minute).
//
// Mirrors the vanilla net/http reference implementation
// (consumer/http/middleware/login_rate_limit.go): same per-IP sliding window,
// same env var, same default, same Retry-After header on 429.
func LoginRateLimit() fiber.Handler {
	mw, _ := newFiberLoginRateLimitMiddleware(os.Getenv)
	return mw
}

// LoginRateLimitWithStop is like LoginRateLimit but also returns a stop
// function that halts the background cleanup goroutine (useful in tests).
func LoginRateLimitWithStop() (fiber.Handler, func()) {
	return newFiberLoginRateLimitMiddleware(os.Getenv)
}

func newFiberLoginRateLimitMiddleware(getenv func(string) string) (fiber.Handler, func()) {
	cfg := parseFiberLoginRateLimitEnv(getenv)

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
					rec := value.(*fiberIPAttemptRecord)
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

	mw := func(c *fiber.Ctx) error {
		// Only gate POST /auth/login.
		if c.Method() != fiber.MethodPost || c.Path() != loginRateLimitPath {
			return c.Next()
		}

		ip := extractFiberIP(c)
		now := time.Now()
		cutoff := now.Add(-cfg.window)

		val, _ := store.LoadOrStore(ip, &fiberIPAttemptRecord{})
		rec := val.(*fiberIPAttemptRecord)

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
			c.Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			return c.Status(fiber.StatusTooManyRequests).SendString("Too Many Requests")
		}

		rec.timestamps = append(rec.timestamps, now)
		rec.mu.Unlock()

		return c.Next()
	}

	return mw, stop
}

func extractFiberIP(c *fiber.Ctx) string {
	// X-Forwarded-For first entry.
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	// X-Real-IP.
	if xri := c.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Fiber's IP() (RemoteAddr equivalent).
	host, _, err := net.SplitHostPort(c.IP())
	if err != nil {
		return c.IP()
	}
	return host
}
