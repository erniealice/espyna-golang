//go:build vanilla && !gin && !fiber && !fiber_v3

package vanilla

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/composition/core"
	"github.com/erniealice/espyna-golang/internal/composition/routing"
	"github.com/erniealice/espyna-golang/internal/composition/routing/customization"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterServerProvider(
		"vanilla",
		func() ports.ServerProvider {
			return NewVanillaAdapter()
		},
		buildFromEnv,
	)
}

// buildFromEnv creates a Vanilla adapter from environment variables.
func buildFromEnv() (ports.ServerProvider, error) {
	adapter := NewVanillaAdapter()
	return adapter, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// VanillaAdapter implements ServerProvider for vanilla net/http.
type VanillaAdapter struct {
	mux       *http.ServeMux
	container *core.Container
	enabled   bool
	server    *http.Server
}

// NewVanillaAdapter creates a new vanilla HTTP server adapter.
func NewVanillaAdapter() *VanillaAdapter {
	return &VanillaAdapter{}
}

// Name returns the provider name.
func (a *VanillaAdapter) Name() string {
	return "vanilla"
}

// Initialize sets up the HTTP mux with the container.
// The container parameter should be *core.Container but is typed as any
// to satisfy the ports.ServerProvider interface and avoid import cycles.
func (a *VanillaAdapter) Initialize(container any) error {
	if container == nil {
		return fmt.Errorf("vanilla adapter requires a non-nil container")
	}

	// Type assert to *core.Container
	c, ok := container.(*core.Container)
	if !ok {
		return fmt.Errorf("vanilla adapter requires *core.Container, got %T", container)
	}

	a.container = c
	a.mux = http.NewServeMux()
	a.enabled = true

	// Install espyna routes
	a.installRoutes()

	// Add default health endpoint
	a.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"framework": "vanilla",
		})
	})

	log.Printf("Vanilla adapter initialized successfully")
	return nil
}

// installRoutes exports routes from container and installs them on the HTTP mux
func (a *VanillaAdapter) installRoutes() {
	customizer := customization.NewRouteCustomizer()
	baseRoutes := a.container.GetRouteManager().GetAllRoutes()
	routes := customizer.ApplyCustomizations(baseRoutes)

	log.Printf("INFO: Installing %d routes on vanilla HTTP mux", len(routes))

	for _, route := range routes {
		a.installRouteOnMux(route)
	}
}

// installRouteOnMux installs a single route on the HTTP mux
func (a *VanillaAdapter) installRouteOnMux(route *routing.Route) {
	handler := a.createHTTPHandler(route)
	a.mux.HandleFunc(route.Path, handler)
}

// createHTTPHandler creates an HTTP handler from an espyna route
func (a *VanillaAdapter) createHTTPHandler(route *routing.Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check HTTP method
		if r.Method != route.Method && r.Method != "OPTIONS" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		// Set timeout context
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Add user context for mock auth
		ctx = context.WithValue(ctx, "user_id", "consumer-app-user")
		ctx = context.WithValue(ctx, "workspace_id", "test-workspace")
		ctx = context.WithValue(ctx, "roles", []string{"admin", "user"})

		var req proto.Message
		var err error

		// Parse request body for methods that typically have request data
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "Failed to read body", err.Error())
				return
			}
			defer r.Body.Close()

			// Only try to parse if there's actual content
			if len(body) > 0 {
				// Parse to protobuf using the handler's parser
				if parser, ok := route.Handler.(contracts.ProtobufParser); ok {
					req, err = parser.ParseRequestFromJSON(body)
					if err != nil {
						writeJSONError(w, http.StatusBadRequest, "Failed to parse request JSON", err.Error())
						return
					}
				} else {
					writeJSONError(w, http.StatusInternalServerError, "Handler does not support JSON parsing", "")
					return
				}
			}
		}

		// Execute handler
		resp, err := route.Handler.Execute(ctx, req)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Handler execution failed", err.Error())
			return
		}

		// Return response
		if resp != nil {
			json.NewEncoder(w).Encode(resp)
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message":    "Success",
				"route_name": route.Metadata.Name,
			})
		}
	}
}

// Start starts the vanilla HTTP server on the specified address.
func (a *VanillaAdapter) Start(addr string) error {
	if a.mux == nil {
		return fmt.Errorf("vanilla adapter not initialized - call Initialize() first")
	}

	printServerInfo("vanilla", addr)

	// Wrap the mux with CORS and Gzip middleware
	handler := corsMiddleware(gzipMiddleware(a.mux))

	a.server = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return a.server.ListenAndServe()
}

// IsHealthy checks if the server is healthy.
func (a *VanillaAdapter) IsHealthy(ctx context.Context) error {
	if a.mux == nil {
		return fmt.Errorf("vanilla mux not initialized")
	}
	return nil
}

// Close shuts down the vanilla server.
func (a *VanillaAdapter) Close() error {
	if a.server != nil {
		log.Printf("Vanilla adapter closing")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return a.server.Shutdown(ctx)
	}
	return nil
}

// IsEnabled returns whether this adapter is enabled.
func (a *VanillaAdapter) IsEnabled() bool {
	return a.enabled
}

// RegisterCustomHandler registers a custom HTTP handler for the given method and path.
// This allows consumer applications to add custom routes beyond the espyna-generated routes.
func (a *VanillaAdapter) RegisterCustomHandler(method, path string, handler http.HandlerFunc) error {
	if a.mux == nil {
		return fmt.Errorf("vanilla adapter not initialized - call Initialize() first")
	}

	// For vanilla HTTP, we create a wrapper that checks the method
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		// Check if the method matches
		if r.Method != method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}

	a.mux.HandleFunc(path, wrappedHandler)
	return nil
}

// GetMux returns the underlying HTTP mux for advanced customization.
func (a *VanillaAdapter) GetMux() *http.ServeMux {
	return a.mux
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, status int, message, details string) {
	w.WriteHeader(status)
	response := map[string]interface{}{
		"error": message,
	}
	if details != "" {
		response["details"] = details
	}
	json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// gzipMiddleware compresses responses when client accepts gzip
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Create gzip writer
		gz := gzip.NewWriter(w)
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}

// gzipResponseWriter wraps http.ResponseWriter for gzip compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// printServerInfo prints server startup information
func printServerInfo(framework, addr string) {
	fmt.Printf("\n")
	fmt.Printf("  Espyna Server\n")
	fmt.Printf("  Framework: %s\n", framework)
	fmt.Printf("  Address: %s\n", addr)
	fmt.Printf("  Database: %s\n", getEnv("CONFIG_DATABASE_PROVIDER", "mock_db"))
	fmt.Printf("  Auth: %s\n", getEnv("CONFIG_AUTH_PROVIDER", "mock_auth"))
	fmt.Printf("  ID: %s\n", getEnv("CONFIG_ID_PROVIDER", "noop"))
	fmt.Printf("  Storage: %s\n", getEnv("CONFIG_STORAGE_PROVIDER", "mock_storage"))
	fmt.Printf("\n")
}

// getEnv returns environment variable value or default if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Compile-time interface check
var _ ports.ServerProvider = (*VanillaAdapter)(nil)
