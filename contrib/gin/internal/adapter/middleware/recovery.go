//go:build gin

package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery returns a Gin middleware that recovers from panics and returns an
// internal server error. Mirrors the vanilla net/http Recovery middleware
// (apps/service-admin/internal/infrastructure/input/http/middleware/recovery.go).
//
// Gin has a built-in gin.Recovery() middleware, but this wrapper provides
// consistent logging format (method, path, stack trace) matching the net/http
// reference implementation, and ensures all server providers produce identical
// panic-recovery behavior.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered on %s %s: %v\n%s",
					c.Request.Method,
					c.Request.URL.Path,
					err,
					debug.Stack(),
				)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
