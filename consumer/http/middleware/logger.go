package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWrapper captures the status code written by downstream handlers so
// the logger can report it after the request completes.
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Logger returns a MiddlewareFunc that logs every HTTP request with method,
// path, status code, and duration.
func Logger() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)
			log.Printf("%s %s %d %v",
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				time.Since(start),
			)
		})
	}
}
