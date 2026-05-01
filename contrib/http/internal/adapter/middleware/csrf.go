//go:build vanilla

package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

// CSRF creates a Vanilla HTTP-specific CSRF middleware
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for safe methods
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			// For GET requests, generate and set CSRF token
			if r.Method == "GET" {
				token := generateCSRFToken()
				cookie := &http.Cookie{
					Name:     "csrf_token",
					Value:    token,
					Path:     "/",
					MaxAge:   3600, // 1 hour
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				}
				http.SetCookie(w, cookie)
				w.Header().Set("X-Csrf-Token", token)
			}
			next.ServeHTTP(w, r)
			return
		}

		// For unsafe methods, validate CSRF token
		cookieToken := ""
		if cookie, err := r.Cookie("csrf_token"); err == nil {
			cookieToken = cookie.Value
		}

		if cookieToken == "" {
			writeCSRFError(w, "CSRF token missing from cookie")
			return
		}

		headerToken := r.Header.Get("X-Csrf-Token")
		if headerToken == "" {
			writeCSRFError(w, "CSRF token missing from header")
			return
		}

		if cookieToken != headerToken {
			writeCSRFError(w, "CSRF token validation failed")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// generateCSRFToken generates a cryptographically secure random token
func generateCSRFToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based token if random generation fails
		return base64.URLEncoding.EncodeToString([]byte(time.Now().String()))
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// writeCSRFError writes a standardized CSRF error response
func writeCSRFError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	
	json.NewEncoder(w).Encode(response)
}