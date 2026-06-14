package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery returns a MiddlewareFunc that recovers from panics in downstream
// handlers and responds with a 500 Internal Server Error.
func Recovery() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Panic recovered on %s %s: %v\n%s",
						r.Method, r.URL.Path, err, debug.Stack())
					http.Error(w, "Internal Server Error",
						http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
