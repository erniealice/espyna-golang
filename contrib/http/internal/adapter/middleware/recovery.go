//go:build http

package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery recovers from panics and returns an internal server error
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered on %s %s: %v\n%s", r.Method, r.URL.Path, err, debug.Stack())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
