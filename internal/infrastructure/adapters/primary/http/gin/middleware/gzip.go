//go:build gin

package middleware

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// Gzip creates a Gin-specific gzip compression middleware
func Gzip() gin.HandlerFunc {
	return gzip.Gzip(gzip.DefaultCompression)
}
