//go:build vanilla

package vanilla

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/composition/routing"
)

// setupRoutes configures all HTTP routes using the route manager.
func (s *Server) setupRoutes() {
	fmt.Printf("🔍 Setting up routes...\n")

	// Get routes from the container route manager (composer-based routing)
	routeManager := s.container.GetRouteManager()
	fmt.Printf("📊 Route manager from container: %v\n", routeManager != nil)

	if routeManager != nil {
		// Get all routes from the route manager
		routes := routeManager.GetAllRoutes()
		fmt.Printf("📊 Total routes from route manager: %d\n", len(routes))

		// Install each route on the vanilla mux
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

	fmt.Printf("✅ Route setup completed\n")
}

// installRoute installs a single route on the vanilla mux
func (s *Server) installRoute(route *routing.Route) {
	// Create handler function
	handler := s.createRouteHandler(route)

	// Register route with mux with full path
	fullPath := route.Path
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}

	switch route.Method {
	case "GET":
		s.mux.HandleFunc(fullPath, handler)
	case "POST":
		s.mux.HandleFunc(fullPath, handler)
	case "PUT":
		s.mux.HandleFunc(fullPath, handler)
	case "DELETE":
		s.mux.HandleFunc(fullPath, handler)
	default:
		// Unsupported method, skip
	}
}

// createRouteHandler creates an HTTP handler function from a routing.Route
func (s *Server) createRouteHandler(route *routing.Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Enhanced logging: Request start
		fmt.Printf("🚀 [ROUTE HANDLER] Incoming request: %s %s\n", r.Method, r.URL.Path)
		fmt.Printf("📋 [ROUTE INFO] Expected method: %s, Path: %s, Domain: %s, Resource: %s, Operation: %s\n",
			route.Method, route.Path, route.Metadata.Domain, route.Metadata.Resource, route.Metadata.Operation)

		// Check HTTP method
		if r.Method != route.Method {
			fmt.Printf("❌ [METHOD CHECK] Failed: got %s, expected %s\n", r.Method, route.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"message": "Method not allowed",
			})
			return
		}

		fmt.Printf("✅ [METHOD CHECK] Passed: %s matches expected %s\n", r.Method, route.Method)

		// Set content type
		w.Header().Set("Content-Type", "application/json")

		// Execute the route handler if it exists
		if route.Handler != nil {
			fmt.Printf("🔧 [HANDLER CHECK] Handler exists, executing...\n")

			// Log request body details
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					fmt.Printf("❌ [BODY READ] Failed to read request body: %v\n", err)
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]any{
						"success": false,
						"message": "Failed to read request body: " + err.Error(),
					})
					return
				}
				fmt.Printf("📝 [REQUEST BODY] Raw body: %s\n", string(bodyBytes))
				// Restore the body for subsequent reads
				r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
			} else {
				fmt.Printf("📝 [REQUEST BODY] Body is nil\n")
			}

			// Parse request body into appropriate protobuf based on route metadata
			fmt.Printf("🔄 [PARSER] Starting protobuf parsing...\n")
			protobufRequest, err := s.parseRequestToProtobuf(r, route)
			if err != nil {
				fmt.Printf("❌ [PARSER] Failed to parse request: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{
					"success": false,
					"message": "Failed to parse request: " + err.Error(),
				})
				return
			}

			fmt.Printf("✅ [PARSER] Successfully parsed request to protobuf: %T\n", protobufRequest)

			// Call the use case handler directly with protobuf request
			fmt.Printf("🎯 [HANDLER EXEC] Executing route handler...\n")

			// Add mock user context for testing (since we're using mock_auth)
			ctx := r.Context()
			ctx = context.WithValue(ctx, "uid", "mock-user-12345")
			fmt.Printf("🔐 [AUTH] Added mock user context: mock-user-12345\n")

			response, err := route.Handler.Execute(ctx, protobufRequest)
			if err != nil {
				fmt.Printf("❌ [HANDLER EXEC] Handler execution failed: %v\n", err)
				fmt.Printf("🔍 [ERROR DETAILS] Error type: %T, Error: %s\n", err, err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(response)
				return
			}

			fmt.Printf("✅ [HANDLER EXEC] Handler executed successfully\n")
			fmt.Printf("📦 [RESPONSE] Response type: %T\n", response)

			// Convert protobuf response to JSON
			fmt.Printf("🔄 [ENCODER] Encoding response to JSON...\n")
			w.WriteHeader(http.StatusOK)
			err = json.NewEncoder(w).Encode(response)
			if err != nil {
				fmt.Printf("❌ [ENCODER] Failed to encode response: %v\n", err)
			} else {
				fmt.Printf("✅ [ENCODER] Response encoded successfully\n")
			}
			return
		}

		fmt.Printf("❌ [HANDLER CHECK] No handler found for route\n")
		// No handler found
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "Handler not found",
		})
	}
}

// setupBasicRoutes sets up basic routes like health check
func (s *Server) setupBasicRoutes() {
	// Health check endpoint
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"success":   true,
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		}
		jsonBytes, _ := json.Marshal(response)
		w.Write(jsonBytes)
	})

	// Root endpoint
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"success": true,
			"message": "Espyna API (Vanilla)",
			"version": "1.0.0",
		}
		jsonBytes, _ := json.Marshal(response)
		w.Write(jsonBytes)
	})
}

// Helper methods for extracting request data

// extractPathParams extracts path parameters from the HTTP request
func (s *Server) extractPathParams(r *http.Request, routePath string) map[string]string {
	// For simplicity, we'll implement basic path parameter extraction
	// In a real implementation, this would use a more sophisticated path matching algorithm
	pathParams := make(map[string]string)

	// Example: for route "/api/users/{id}", extract the ID from the URL
	// This is a simplified implementation
	return pathParams
}

// extractQueryParams extracts query parameters from the HTTP request
func (s *Server) extractQueryParams(r *http.Request) map[string]string {
	queryParams := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}
	return queryParams
}

// extractHeaders extracts headers from the HTTP request
func (s *Server) extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

// extractBody extracts and parses the request body
func (s *Server) extractBody(r *http.Request) []byte {
	if r.Body == nil {
		return []byte("{}")
	}
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Return empty JSON object on error
		return []byte("{}")
	}

	return body
}

// parseRequestToProtobuf parses HTTP request body into appropriate protobuf using dynamic parsing
func (s *Server) parseRequestToProtobuf(r *http.Request, route *routing.Route) (proto.Message, error) {
	body := s.extractBody(r)
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
