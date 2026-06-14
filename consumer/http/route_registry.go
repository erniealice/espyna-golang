package http

// route_registry.go — the Server-owned route registry (Wave B D1).
//
// In Wave A the fluent Build() handed blocks a consumer.NoopRegistrar and the
// app still owned the real RouteRegistry + the HTTP-adapter finalize. D1 moves
// the framework-GENERIC finalize server-side: Build() now provides a REAL
// registry that blocks register their handlers/views into, reads it back after
// the opt loop, and feeds the routes into the HTTP adapter. This is the espyna
// twin of service-admin's composition.RouteRegistry — it satisfies pyeza's
// view.RouteRegistrar (GET/POST take a view.View) plus the broader
// HandleFunc/Redirect surface the auth module (D2) registers through.

import (
	"net/http"

	"github.com/erniealice/pyeza-golang/view"
)

// routeRegistry collects RouteConfig entries from domain blocks (and, after D2,
// the entydad auth module) during composition. The Server reads Routes() back
// after the opt loop to wire the HTTP adapter. It implements pyeza.RouteRegistrar
// (GET/POST) and the auth-module HandleFunc/Redirect surface.
type routeRegistry struct {
	routes []RouteConfig
}

// newRouteRegistry creates an empty Server-owned route registry.
func newRouteRegistry() *routeRegistry {
	return &routeRegistry{routes: make([]RouteConfig, 0)}
}

// GET registers a GET view route.
func (r *routeRegistry) GET(path string, v view.View, middlewares ...string) {
	r.routes = append(r.routes, RouteConfig{
		Method:      "GET",
		Path:        path,
		View:        v,
		Middlewares: middlewares,
	})
}

// POST registers a POST view route.
func (r *routeRegistry) POST(path string, v view.View, middlewares ...string) {
	r.routes = append(r.routes, RouteConfig{
		Method:      "POST",
		Path:        path,
		View:        v,
		Middlewares: middlewares,
	})
}

// HandleFunc registers a non-view handler route (used by the auth module).
func (r *routeRegistry) HandleFunc(method, path string, handler http.HandlerFunc, middlewares ...string) {
	r.routes = append(r.routes, RouteConfig{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Middlewares: middlewares,
	})
}

// Redirect registers a GET redirect route.
func (r *routeRegistry) Redirect(path, target string) {
	r.routes = append(r.routes, RouteConfig{
		Method: "GET",
		Path:   path,
		Handler: func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, target, http.StatusFound)
		},
	})
}

// Routes returns all registered routes in registration order.
func (r *routeRegistry) Routes() []RouteConfig {
	return r.routes
}
