package infrastructure

import (
	"context"
	"net/http"
)

// ServerProvider defines the contract for HTTP server providers.
// This interface abstracts HTTP frameworks like Gin, Fiber, Vanilla HTTP, etc.
//
// Unlike other infrastructure providers (database, auth, storage), the server
// provider is a PRIMARY adapter that exposes the application to the outside world.
// It uses `any` for the container parameter to avoid import cycles between
// the ports layer and the composition layer.
type ServerProvider interface {
	// Name returns the name of the server provider (e.g., "gin", "fiber", "vanilla")
	Name() string

	// Initialize sets up the server with the application container.
	// The container parameter should be *core.Container but is typed as any
	// to avoid import cycles. The implementation should type-assert it.
	Initialize(container any) error

	// Start begins listening on the specified address (e.g., ":8080" or "localhost:8080")
	Start(addr string) error

	// IsHealthy checks if the server is available and ready to serve requests
	IsHealthy(ctx context.Context) error

	// Close shuts down the server gracefully
	Close() error

	// IsEnabled returns whether this provider is currently enabled
	IsEnabled() bool

	// RegisterCustomHandler registers a custom HTTP handler for the given method and path.
	// This allows consumer applications to add custom routes beyond the espyna-generated routes.
	// The handler is a standard http.HandlerFunc, which all frameworks can wrap.
	//
	// Parameters:
	//   - method: HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	//   - path: URL path (e.g., "/api/v1/custom", "/health")
	//   - handler: Standard http.HandlerFunc to register
	//
	// Returns an error if the registration fails or the provider is not initialized.
	RegisterCustomHandler(method, path string, handler http.HandlerFunc) error
}
