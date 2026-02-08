//go:build gin

package gin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
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
		"gin",
		func() ports.ServerProvider {
			return NewGinAdapter()
		},
		buildFromEnv,
	)
}

// buildFromEnv creates and initializes a Gin adapter from environment variables.
// Note: This doesn't fully initialize the adapter - Initialize() must be called
// with a container before Start() can be used.
func buildFromEnv() (ports.ServerProvider, error) {
	adapter := NewGinAdapter()
	// Note: Initialize() requires a container, which is provided separately
	return adapter, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// GinAdapter implements ServerProvider for the Gin HTTP framework.
type GinAdapter struct {
	router    *gin.Engine
	container *core.Container
	enabled   bool
}

// NewGinAdapter creates a new Gin server adapter.
func NewGinAdapter() *GinAdapter {
	return &GinAdapter{}
}

// Name returns the provider name.
func (a *GinAdapter) Name() string {
	return "gin"
}

// Initialize sets up the Gin router with the container.
// The container parameter should be *core.Container but is typed as any
// to satisfy the ports.ServerProvider interface and avoid import cycles.
func (a *GinAdapter) Initialize(container any) error {
	if container == nil {
		return fmt.Errorf("gin adapter requires a non-nil container")
	}

	// Type assert to *core.Container
	c, ok := container.(*core.Container)
	if !ok {
		return fmt.Errorf("gin adapter requires *core.Container, got %T", container)
	}

	a.container = c

	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Add request logging
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	a.router = router
	a.enabled = true

	// Install espyna routes
	a.installRoutes()

	// Add default health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"framework": "gin",
		})
	})

	log.Printf("Gin adapter initialized successfully")
	return nil
}

// installRoutes exports routes from container and installs them on the Gin router
func (a *GinAdapter) installRoutes() {
	customizer := customization.NewRouteCustomizer()
	baseRoutes := a.container.GetRouteManager().GetAllRoutes()
	routes := customizer.ApplyCustomizations(baseRoutes)

	log.Printf("INFO: Installing %d routes on Gin router", len(routes))

	for _, route := range routes {
		a.installRouteOnGin(route)
	}
}

// installRouteOnGin installs a single route on the Gin router
func (a *GinAdapter) installRouteOnGin(route *routing.Route) {
	handler := a.createGinHandler(route)

	switch route.Method {
	case "POST":
		a.router.POST(route.Path, handler)
	case "GET":
		a.router.GET(route.Path, handler)
	case "PUT":
		a.router.PUT(route.Path, handler)
	case "DELETE":
		a.router.DELETE(route.Path, handler)
	case "PATCH":
		a.router.PATCH(route.Path, handler)
	default:
		log.Printf("WARNING: Unsupported HTTP method: %s for route: %s", route.Method, route.Path)
	}
}

// createGinHandler creates a Gin handler from an espyna route
func (a *GinAdapter) createGinHandler(route *routing.Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set timeout context
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		// Add user context for mock auth
		ctx = context.WithValue(ctx, "user_id", "consumer-app-user")
		ctx = context.WithValue(ctx, "workspace_id", "test-workspace")
		ctx = context.WithValue(ctx, "roles", []string{"admin", "user"})

		var req proto.Message
		var err error

		// Parse request body for methods that typically have request data
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			body, err := c.GetRawData()
			if err != nil {
				c.JSON(400, gin.H{
					"error":   "Failed to read body",
					"details": err.Error(),
				})
				return
			}

			// Only try to parse if there's actual content
			if len(body) > 0 {
				// Parse to protobuf using the handler's parser
				if parser, ok := route.Handler.(contracts.ProtobufParser); ok {
					req, err = parser.ParseRequestFromJSON(body)
					if err != nil {
						c.JSON(400, gin.H{
							"error":      "Failed to parse request JSON",
							"details":    err.Error(),
							"route_name": route.Metadata.Name,
						})
						return
					}
				} else {
					c.JSON(500, gin.H{
						"error":      "Handler does not support JSON parsing",
						"route_name": route.Metadata.Name,
					})
					return
				}
			}
		}

		// Execute handler
		resp, err := route.Handler.Execute(ctx, req)
		if err != nil {
			c.JSON(500, gin.H{
				"error":      "Handler execution failed",
				"details":    err.Error(),
				"route_name": route.Metadata.Name,
			})
			return
		}

		// Return response
		if resp != nil {
			c.JSON(200, resp)
		} else {
			c.JSON(200, gin.H{"message": "Success", "route_name": route.Metadata.Name})
		}
	}
}

// Start starts the Gin HTTP server on the specified address.
func (a *GinAdapter) Start(addr string) error {
	if a.router == nil {
		return fmt.Errorf("gin adapter not initialized - call Initialize() first")
	}

	printServerInfo("gin", addr)
	return a.router.Run(addr)
}

// IsHealthy checks if the server is healthy.
func (a *GinAdapter) IsHealthy(ctx context.Context) error {
	if a.router == nil {
		return fmt.Errorf("gin router not initialized")
	}
	return nil
}

// Close shuts down the Gin server.
func (a *GinAdapter) Close() error {
	log.Printf("Gin adapter closing")
	return nil
}

// IsEnabled returns whether this adapter is enabled.
func (a *GinAdapter) IsEnabled() bool {
	return a.enabled
}

// RegisterCustomHandler registers a custom HTTP handler for the given method and path.
// This allows consumer applications to add custom routes beyond the espyna-generated routes.
func (a *GinAdapter) RegisterCustomHandler(method, path string, handler http.HandlerFunc) error {
	if a.router == nil {
		return fmt.Errorf("gin adapter not initialized - call Initialize() first")
	}

	// Wrap http.HandlerFunc with gin.WrapH
	wrappedHandler := gin.WrapH(handler)

	switch method {
	case "GET":
		a.router.GET(path, wrappedHandler)
	case "POST":
		a.router.POST(path, wrappedHandler)
	case "PUT":
		a.router.PUT(path, wrappedHandler)
	case "DELETE":
		a.router.DELETE(path, wrappedHandler)
	case "PATCH":
		a.router.PATCH(path, wrappedHandler)
	case "OPTIONS":
		a.router.OPTIONS(path, wrappedHandler)
	case "HEAD":
		a.router.HEAD(path, wrappedHandler)
	default:
		return fmt.Errorf("unsupported HTTP method: %s", method)
	}

	return nil
}

// GetRouter returns the underlying Gin router for advanced customization.
func (a *GinAdapter) GetRouter() *gin.Engine {
	return a.router
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
var _ ports.ServerProvider = (*GinAdapter)(nil)
