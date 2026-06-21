package compose

import (
	"log"
	"net/http"

	"github.com/erniealice/pyeza-golang/view"
)

// handleFuncRegistrar extends RouteRegistrar with raw HTTP handler support.
type handleFuncRegistrar interface {
	view.RouteRegistrar
	HandleFunc(method, path string, handler http.HandlerFunc, middlewares ...string)
}

// HandleFunc registers a raw http.HandlerFunc route on the registrar if it
// supports HandleFunc. Silently skips when the handler is nil or the path is
// empty. Logs a warning when the registrar does not implement HandleFunc
// (matching fayna block.go's existing behavior).
func HandleFunc(r view.RouteRegistrar, method, path string, handler http.HandlerFunc) {
	if path == "" || handler == nil {
		return
	}
	if full, ok := r.(handleFuncRegistrar); ok {
		full.HandleFunc(method, path, handler)
		return
	}
	log.Printf("compose: RouteRegistrar does not support HandleFunc — skipping %s %s", method, path)
}
