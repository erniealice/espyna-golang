//go:build gin

package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a Gin middleware that logs HTTP requests with method, path,
// status code, and duration. Mirrors the vanilla net/http Logger middleware
// (apps/service-admin/internal/infrastructure/input/http/middleware/logger.go)
// but uses gin.LogFormatterParams for richer output.
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s %s %d %v\n",
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
		)
	})
}

// LoggerWithConfig returns a Gin middleware that logs with the specified
// formatter. This is the configurable variant for callers that need custom
// log output (e.g. structured JSON logging). When formatter is nil, it
// falls back to the default Logger format.
func LoggerWithConfig(formatter gin.LogFormatter) gin.HandlerFunc {
	if formatter == nil {
		return Logger()
	}
	return gin.LoggerWithFormatter(formatter)
}

// LoggerVerbose returns a Gin middleware that logs HTTP requests with full
// detail: client IP, timestamp, method, path, protocol, status, latency,
// user agent, and any error message. This matches the format used by the
// GinAdapter.Initialize method in adapter.go.
func LoggerVerbose() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}
