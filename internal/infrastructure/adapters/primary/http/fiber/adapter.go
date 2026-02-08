//go:build fiber && !vanilla && !gin && !fiber_v3

package fiber

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
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
		"fiber",
		func() ports.ServerProvider {
			return NewFiberAdapter()
		},
		buildFromEnv,
	)
}

// buildFromEnv creates a Fiber adapter from environment variables.
func buildFromEnv() (ports.ServerProvider, error) {
	adapter := NewFiberAdapter()
	return adapter, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// FiberAdapter implements ServerProvider for the Fiber v2 HTTP framework.
type FiberAdapter struct {
	app       *fiber.App
	container *core.Container
	enabled   bool
}

// NewFiberAdapter creates a new Fiber server adapter.
func NewFiberAdapter() *FiberAdapter {
	return &FiberAdapter{}
}

// Name returns the provider name.
func (a *FiberAdapter) Name() string {
	return "fiber"
}

// Initialize sets up the Fiber app with the container.
// The container parameter should be *core.Container but is typed as any
// to satisfy the ports.ServerProvider interface and avoid import cycles.
func (a *FiberAdapter) Initialize(container any) error {
	if container == nil {
		return fmt.Errorf("fiber adapter requires a non-nil container")
	}

	// Type assert to *core.Container
	c, ok := container.(*core.Container)
	if !ok {
		return fmt.Errorf("fiber adapter requires *core.Container, got %T", container)
	}

	a.container = c

	app := fiber.New(fiber.Config{
		AppName: "Espyna API v1.0 (Fiber)",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
		},
	})

	// Add CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
		AllowHeaders: "Content-Type, Authorization",
	}))

	// Add compression middleware
	app.Use(compress.New())

	a.app = app
	a.enabled = true

	// Install espyna routes
	a.installRoutes()

	// Add default health endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"framework": "fiber",
		})
	})

	log.Printf("Fiber adapter initialized successfully")
	return nil
}

// installRoutes exports routes from container and installs them on the Fiber app
func (a *FiberAdapter) installRoutes() {
	customizer := customization.NewRouteCustomizer()
	baseRoutes := a.container.GetRouteManager().GetAllRoutes()
	routes := customizer.ApplyCustomizations(baseRoutes)

	log.Printf("INFO: Installing %d routes on Fiber router", len(routes))

	for _, route := range routes {
		a.installRouteOnFiber(route)
	}
}

// installRouteOnFiber installs a single route on the Fiber app
func (a *FiberAdapter) installRouteOnFiber(route *routing.Route) {
	handler := a.createFiberHandler(route)

	switch route.Method {
	case "POST":
		a.app.Post(route.Path, handler)
	case "GET":
		a.app.Get(route.Path, handler)
	case "PUT":
		a.app.Put(route.Path, handler)
	case "DELETE":
		a.app.Delete(route.Path, handler)
	case "PATCH":
		a.app.Patch(route.Path, handler)
	default:
		log.Printf("WARNING: Unsupported HTTP method: %s for route: %s", route.Method, route.Path)
	}
}

// createFiberHandler creates a Fiber handler from an espyna route
func (a *FiberAdapter) createFiberHandler(route *routing.Route) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()

		ctx = context.WithValue(ctx, "user_id", "consumer-app-user")
		ctx = context.WithValue(ctx, "workspace_id", "test-workspace")
		ctx = context.WithValue(ctx, "roles", []string{"admin", "user"})

		var req proto.Message
		var err error

		if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
			body := c.Body()

			if len(body) > 0 {
				if parser, ok := route.Handler.(contracts.ProtobufParser); ok {
					req, err = parser.ParseRequestFromJSON(body)
					if err != nil {
						return c.Status(400).JSON(fiber.Map{
							"error":      "Failed to parse request JSON",
							"details":    err.Error(),
							"route_name": route.Metadata.Name,
						})
					}
				} else {
					return c.Status(500).JSON(fiber.Map{
						"error":      "Handler does not support JSON parsing",
						"route_name": route.Metadata.Name,
					})
				}
			}
		}

		resp, err := route.Handler.Execute(ctx, req)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":      "Handler execution failed",
				"details":    err.Error(),
				"route_name": route.Metadata.Name,
			})
		}

		if resp != nil {
			return c.JSON(resp)
		}
		return c.JSON(fiber.Map{"message": "Success", "route_name": route.Metadata.Name})
	}
}

// Start starts the Fiber HTTP server on the specified address.
func (a *FiberAdapter) Start(addr string) error {
	if a.app == nil {
		return fmt.Errorf("fiber adapter not initialized - call Initialize() first")
	}

	printServerInfo("fiber", addr)
	return a.app.Listen(addr)
}

// IsHealthy checks if the server is healthy.
func (a *FiberAdapter) IsHealthy(ctx context.Context) error {
	if a.app == nil {
		return fmt.Errorf("fiber app not initialized")
	}
	return nil
}

// Close shuts down the Fiber server.
func (a *FiberAdapter) Close() error {
	if a.app != nil {
		log.Printf("Fiber adapter closing")
		return a.app.Shutdown()
	}
	return nil
}

// IsEnabled returns whether this adapter is enabled.
func (a *FiberAdapter) IsEnabled() bool {
	return a.enabled
}

// RegisterCustomHandler registers a custom HTTP handler for the given method and path.
// This allows consumer applications to add custom routes beyond the espyna-generated routes.
func (a *FiberAdapter) RegisterCustomHandler(method, path string, handler http.HandlerFunc) error {
	if a.app == nil {
		return fmt.Errorf("fiber adapter not initialized - call Initialize() first")
	}

	// Convert http.HandlerFunc to fiber.Handler
	wrappedHandler := func(c *fiber.Ctx) error {
		// Create http.ResponseWriter from fiber.Ctx
		w := &fiberResponseWriter{Ctx: c}

		// Create http.Request from fiber.Ctx
		r := c.Context().Request

		// Call the handler
		handler(w, r)

		return nil
	}

	switch method {
	case "GET":
		a.app.Get(path, wrappedHandler)
	case "POST":
		a.app.Post(path, wrappedHandler)
	case "PUT":
		a.app.Put(path, wrappedHandler)
	case "DELETE":
		a.app.Delete(path, wrappedHandler)
	case "PATCH":
		a.app.Patch(path, wrappedHandler)
	case "OPTIONS":
		a.app.Options(path, wrappedHandler)
	case "HEAD":
		a.app.Head(path, wrappedHandler)
	default:
		return fmt.Errorf("unsupported HTTP method: %s", method)
	}

	return nil
}

// GetApp returns the underlying Fiber app for advanced customization.
func (a *FiberAdapter) GetApp() *fiber.App {
	return a.app
}

// fiberResponseWriter implements http.ResponseWriter using fiber.Ctx
type fiberResponseWriter struct {
	Ctx *fiber.Ctx
}

func (w *fiberResponseWriter) Header() http.Header {
	// Fiber manages headers separately, return a basic header map
	header := make(http.Header)
	w.Ctx.Context().Request.Header.VisitAll(func(key, value []byte) {
		header.Set(string(key), string(value))
	})
	return header
}

func (w *fiberResponseWriter) Write(data []byte) (int, error) {
	return w.Ctx.Write(data)
}

func (w *fiberResponseWriter) WriteHeader(statusCode int) {
	w.Ctx.Status(statusCode)
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
var _ ports.ServerProvider = (*FiberAdapter)(nil)
