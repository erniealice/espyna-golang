//go:build fiber

package fiber

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/protobuf/proto"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/composition/routing"
)

// setupRoutes configures all HTTP routes using the route manager.
func (s *Server) setupRoutes() {
	fmt.Printf("🔍 Setting up Fiber routes...\n")

	// Get routes from the container route manager (composer-based routing)
	routeManager := s.container.GetRouteManager()
	fmt.Printf("📊 Route manager from container: %v\n", routeManager != nil)

	if routeManager != nil {
		// Get all routes from the route manager
		routes := routeManager.GetAllRoutes()
		fmt.Printf("📊 Total routes from route manager: %d\n", len(routes))

		// Install each route on the Fiber app
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

	fmt.Printf("✅ Fiber route setup completed\n")
}

// installRoute installs a single route on the Fiber app
func (s *Server) installRoute(route *routing.Route) {
	// Create handler function
	handler := s.createRouteHandler(route)

	// Register route with Fiber app
	switch route.Method {
	case "GET":
		s.app.Get(route.Path, handler)
	case "POST":
		s.app.Post(route.Path, handler)
	case "PUT":
		s.app.Put(route.Path, handler)
	case "DELETE":
		s.app.Delete(route.Path, handler)
	case "PATCH":
		s.app.Patch(route.Path, handler)
	default:
		// Unsupported method, skip
		fmt.Printf("⚠️ Unsupported HTTP method: %s for route %s\n", route.Method, route.Path)
	}
}

// createRouteHandler creates a Fiber handler function from a routing.Route
func (s *Server) createRouteHandler(route *routing.Route) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Enhanced logging: Request start
		fmt.Printf("🚀 [FIBER ROUTE HANDLER] Incoming request: %s %s\n", c.Method(), c.Path())
		fmt.Printf("📋 [ROUTE INFO] Expected method: %s, Path: %s, Domain: %s, Resource: %s, Operation: %s\n",
			route.Method, route.Path, route.Metadata.Domain, route.Metadata.Resource, route.Metadata.Operation)

		// Check HTTP method
		if c.Method() != route.Method {
			fmt.Printf("❌ [METHOD CHECK] Failed: got %s, expected %s\n", c.Method(), route.Method)
			c.Set("Content-Type", "application/json")
			return c.Status(http.StatusMethodNotAllowed).JSON(fiber.Map{
				"success": false,
				"message": "Method not allowed",
			})
		}

		fmt.Printf("✅ [METHOD CHECK] Passed: %s matches expected %s\n", c.Method(), route.Method)

		// Set content type
		c.Set("Content-Type", "application/json")

		// Execute the route handler if it exists
		if route.Handler != nil {
			fmt.Printf("🔧 [HANDLER CHECK] Handler exists, executing...\n")

			// Log request body details
			body := s.extractBody(c)
			fmt.Printf("📝 [REQUEST BODY] Raw body: %s\n", string(body))

			// Parse request body into appropriate protobuf based on route metadata
			fmt.Printf("🔄 [PARSER] Starting protobuf parsing...\n")
			protobufRequest, err := s.parseRequestToProtobuf(c, route)
			if err != nil {
				fmt.Printf("❌ [PARSER] Failed to parse request: %v\n", err)
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{
					"success": false,
					"message": "Failed to parse request: " + err.Error(),
				})
			}

			fmt.Printf("✅ [PARSER] Successfully parsed request to protobuf: %T\n", protobufRequest)

			// Call the use case handler directly with protobuf request
			fmt.Printf("🎯 [HANDLER EXEC] Executing route handler...\n")

			// Add mock user context for testing (since we're using mock_auth)
			ctx := c.UserContext()
			ctx = context.WithValue(ctx, "uid", "mock-user-12345")
			fmt.Printf("🔐 [AUTH] Added mock user context: mock-user-12345\n")

			response, err := route.Handler.Execute(ctx, protobufRequest)
			if err != nil {
				fmt.Printf("❌ [HANDLER EXEC] Handler execution failed: %v\n", err)
				fmt.Printf("🔍 [ERROR DETAILS] Error type: %T, Error: %s\n", err, err.Error())
				return c.Status(http.StatusInternalServerError).JSON(response)
			}

			fmt.Printf("✅ [HANDLER EXEC] Handler executed successfully\n")
			fmt.Printf("📦 [RESPONSE] Response type: %T\n", response)

			// Convert protobuf response to JSON
			fmt.Printf("🔄 [ENCODER] Encoding response to JSON...\n")
			return c.Status(http.StatusOK).JSON(response)
		}

		fmt.Printf("❌ [HANDLER CHECK] No handler found for route\n")
		// No handler found
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Handler not found",
		})
	}
}

// setupBasicRoutes sets up basic routes like health check
func (s *Server) setupBasicRoutes() {
	// Health check endpoint
	s.app.Get("/health", func(c *fiber.Ctx) error {
		fmt.Printf("🏥 [HEALTH] Health check requested\n")
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"success":   true,
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"transport": "fiber",
		})
	})

	// Root endpoint
	s.app.Get("/", func(c *fiber.Ctx) error {
		fmt.Printf("🏠 [ROOT] Root endpoint requested\n")
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "Espyna API (Fiber)",
			"version": "1.0.0",
			"transport": "fiber",
		})
	})
}

// Helper methods for extracting request data

// extractPathParams extracts path parameters from the Fiber request
func (s *Server) extractPathParams(c *fiber.Ctx) map[string]string {
	// Note: Fiber doesn't have a direct way to get all params as a map
	// This is a simplified implementation - you might need to implement
	// parameter extraction based on your route patterns
	pathParams := make(map[string]string)

	// For common path parameters, you would extract them based on route definitions
	// For now, this is a placeholder implementation
	return pathParams
}

// extractQueryParams extracts query parameters from the Fiber request
func (s *Server) extractQueryParams(c *fiber.Ctx) map[string]string {
	queryParams := make(map[string]string)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})
	return queryParams
}

// extractHeaders extracts headers from the Fiber request
func (s *Server) extractHeaders(c *fiber.Ctx) map[string]string {
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})
	return headers
}

// extractBody extracts and parses the request body
func (s *Server) extractBody(c *fiber.Ctx) []byte {
	body := c.Body()
	if len(body) == 0 {
		return []byte("{}")
	}
	return body
}

// parseRequestToProtobuf parses HTTP request body into appropriate protobuf using dynamic parsing
func (s *Server) parseRequestToProtobuf(c *fiber.Ctx, route *routing.Route) (proto.Message, error) {
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