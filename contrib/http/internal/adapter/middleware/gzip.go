//go:build vanilla

package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipResponseWriter wraps http.ResponseWriter to provide gzip compression
type gzipResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

// Gzip creates a Vanilla HTTP-specific gzip compression middleware
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip encoding
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Check content type - only compress text-based content
		contentType := w.Header().Get("Content-Type")
		if contentType == "" {
			// Set a default content type to enable compression for JSON responses
			w.Header().Set("Content-Type", "application/json")
		}

		// Only compress certain content types
		shouldCompress := strings.Contains(contentType, "application/json") ||
			strings.Contains(contentType, "text/") ||
			strings.Contains(contentType, "application/javascript") ||
			strings.Contains(contentType, "application/xml")

		if !shouldCompress {
			next.ServeHTTP(w, r)
			return
		}

		// Set gzip headers
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		// Create gzip writer
		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()

		// Wrap response writer
		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			writer:         gzipWriter,
		}

		// Call next handler with gzip writer
		next.ServeHTTP(gzw, r)
	})
}
