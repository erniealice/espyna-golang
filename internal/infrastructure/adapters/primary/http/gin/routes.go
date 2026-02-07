//go:build gin

package gin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/composition/routing"
)

// setupRoutes configures all HTTP routes using the route manager.
func (s *Server) setupRoutes() {
	fmt.Printf("🔍 Setting up Gin routes...\n")

	// Get routes from the container route manager (composer-based routing)
	routeManager := s.container.GetRouteManager()
	fmt.Printf("📊 Route manager from container: %v\n", routeManager != nil)

	if routeManager != nil {
		// Get all routes from the route manager
		routes := routeManager.GetAllRoutes()
		fmt.Printf("📊 Total routes from route manager: %d\n", len(routes))

		// Install each route on the Gin router
		for _, route := range routes {
			fmt.Printf("🔄 Installing route: %s %s\n", route.Method, route.Path)
			s.installRoute(route)
		}
	} else {
		fmt.Printf("⚠️ No route manager found\n")
	}

	// Always set up basic routes
	fmt.Printf("🔧 Setting up basic routes...\n")
	s.setupBasicRoutes()

	fmt.Printf("✅ Gin route setup completed\n")
}

// installRoute installs a single route on the Gin router
func (s *Server) installRoute(route *routing.Route) {
	// Create handler function
	handler := s.createRouteHandler(route)

	// Register route with Gin
	switch route.Method {
	case "GET":
		s.router.GET(route.Path, handler)
	case "POST":
		s.router.POST(route.Path, handler)
	case "PUT":
		s.router.PUT(route.Path, handler)
	case "DELETE":
		s.router.DELETE(route.Path, handler)
	case "PATCH":
		s.router.PATCH(route.Path, handler)
	default:
		// Unsupported method, skip
		fmt.Printf("⚠️ Unsupported HTTP method: %s for route %s\n", route.Method, route.Path)
	}
}

// createRouteHandler creates a Gin handler function from a routing.Route
func (s *Server) createRouteHandler(route *routing.Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Enhanced logging: Request start
		fmt.Printf("🚀 [GIN ROUTE HANDLER] Incoming request: %s %s\n", c.Request.Method, c.Request.URL.Path)
		fmt.Printf("📋 [ROUTE INFO] Expected method: %s, Path: %s, Domain: %s, Resource: %s, Operation: %s\n",
			route.Method, route.Path, route.Metadata.Domain, route.Metadata.Resource, route.Metadata.Operation)

		// Check HTTP method
		if c.Request.Method != route.Method {
			fmt.Printf("❌ [METHOD CHECK] Failed: got %s, expected %s\n", c.Request.Method, route.Method)
			c.Header("Content-Type", "application/json")
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"success": false,
				"message": "Method not allowed",
			})
			return
		}

		fmt.Printf("✅ [METHOD CHECK] Passed: %s matches expected %s\n", c.Request.Method, route.Method)

		// Set content type
		c.Header("Content-Type", "application/json")

		// Execute the route handler if it exists
		if route.Handler != nil {
			fmt.Printf("🔧 [HANDLER CHECK] Handler exists, executing...\n")

			// Log request body details
			if c.Request.Body != nil {
				bodyBytes, err := io.ReadAll(c.Request.Body)
				if err != nil {
					fmt.Printf("❌ [BODY READ] Failed to read request body: %v\n", err)
					c.JSON(http.StatusBadRequest, gin.H{
						"success": false,
						"message": "Failed to read request body: " + err.Error(),
					})
					return
				}
				fmt.Printf("📝 [REQUEST BODY] Raw body: %s\n", string(bodyBytes))
				// Restore the body for subsequent reads
				c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
			} else {
				fmt.Printf("📝 [REQUEST BODY] Body is nil\n")
			}

			// Parse request body into appropriate protobuf based on route metadata
			fmt.Printf("🔄 [PARSER] Starting protobuf parsing...\n")
			protobufRequest, err := s.parseRequestToProtobuf(c, route)
			if err != nil {
				fmt.Printf("❌ [PARSER] Failed to parse request: %v\n", err)
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "Failed to parse request: " + err.Error(),
				})
				return
			}

			fmt.Printf("✅ [PARSER] Successfully parsed request to protobuf: %T\n", protobufRequest)

			// Call the use case handler directly with protobuf request
			fmt.Printf("🎯 [HANDLER EXEC] Executing route handler...\n")

			// Add mock user context for testing (since we're using mock_auth)
			ctx := c.Request.Context()
			ctx = context.WithValue(ctx, "uid", "mock-user-12345")
			fmt.Printf("🔐 [AUTH] Added mock user context: mock-user-12345\n")

			response, err := route.Handler.Execute(ctx, protobufRequest)
			if err != nil {
				fmt.Printf("❌ [HANDLER EXEC] Handler execution failed: %v\n", err)
				fmt.Printf("🔍 [ERROR DETAILS] Error type: %T, Error: %s\n", err, err.Error())
				c.JSON(http.StatusInternalServerError, response)
				return
			}

			fmt.Printf("✅ [HANDLER EXEC] Handler executed successfully\n")
			fmt.Printf("📦 [RESPONSE] Response type: %T\n", response)

			// Convert protobuf response to JSON
			fmt.Printf("🔄 [ENCODER] Encoding response to JSON...\n")
			c.JSON(http.StatusOK, response)
			fmt.Printf("✅ [ENCODER] Response encoded successfully\n")
			return
		}

		fmt.Printf("❌ [HANDLER CHECK] No handler found for route\n")
		// No handler found
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Handler not found",
		})
	}
}

// setupBasicRoutes sets up basic routes like health check
func (s *Server) setupBasicRoutes() {
	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		fmt.Printf("🏥 [HEALTH] Health check requested\n")
		c.JSON(http.StatusOK, gin.H{
			"success":   true,
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"transport": "gin",
		})
	})

	// Root endpoint
	s.router.GET("/", func(c *gin.Context) {
		fmt.Printf("🏠 [ROOT] Root endpoint requested\n")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Espyna API (Gin)",
			"version": "1.0.0",
			"transport": "gin",
		})
	})
}

// Helper methods for extracting request data

// extractPathParams extracts path parameters from the Gin request
func (s *Server) extractPathParams(c *gin.Context) map[string]string {
	pathParams := make(map[string]string)
	for _, param := range c.Params {
		pathParams[param.Key] = param.Value
	}
	return pathParams
}

// extractQueryParams extracts query parameters from the Gin request
func (s *Server) extractQueryParams(c *gin.Context) map[string]string {
	queryParams := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}
	return queryParams
}

// extractHeaders extracts headers from the Gin request
func (s *Server) extractHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

// extractBody extracts and parses the request body
func (s *Server) extractBody(c *gin.Context) []byte {
	if c.Request.Body == nil {
		return []byte("{}")
	}

	body, err := c.GetRawData()
	if err != nil {
		// Return empty JSON object on error
		return []byte("{}")
	}

	return body
}

// parseRequestToProtobuf parses HTTP request body into appropriate protobuf using dynamic parsing
func (s *Server) parseRequestToProtobuf(c *gin.Context, route *routing.Route) (proto.Message, error) {
	body := s.extractBody(c)
	if len(body) == 0 {
		return nil, nil // Empty request is valid for some operations
	}

	// Check if the handler supports dynamic protobuf parsing
	if parser, ok := route.Handler.(contracts.ProtobufParser); ok {
		// Use the handler's native protobuf parsing
		return parser.ParseRequestFromJSON(body)
	}

	// Fallback: return empty request for non-protobuf handlers
	return nil, fmt.Errorf("route handler does not support protobuf parsing")
}