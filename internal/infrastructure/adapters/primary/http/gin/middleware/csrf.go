//go:build gin

package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CSRF creates a Gin-specific CSRF middleware
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for safe methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			// For GET requests, generate and set CSRF token
			if c.Request.Method == "GET" {
				token := generateCSRFToken()
				c.SetCookie("csrf_token", token, 3600, "/", "", false, true)
				c.Header("X-Csrf-Token", token)
			}
			c.Next()
			return
		}

		// For unsafe methods, validate CSRF token
		cookieToken, err := c.Cookie("csrf_token")
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "CSRF token missing from cookie",
			})
			c.Abort()
			return
		}

		headerToken := c.GetHeader("X-Csrf-Token")
		if headerToken == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "CSRF token missing from header",
			})
			c.Abort()
			return
		}

		if cookieToken != headerToken {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "CSRF token validation failed",
			})
			c.Abort()
			return
		}

		c.Next()
	}
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