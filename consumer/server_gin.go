//go:build gin

package consumer

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	"leapfor.xyz/espyna/internal/composition/contracts"
)

/*
═ ESPYNA CONSUMER APP - Gin Utilities ═

Gin-specific utility functions for the Espyna consumer application.
This file is only compiled when the 'gin' build tag is specified.

These utilities provide Gin framework integration while keeping the server
implementation logic in the main application for easy customization.

Usage: go run -tags gin,firestore,mock_auth,mock_storage main.go
*/

// CreateGinHandler converts espyna Handler to gin.HandlerFunc
func CreateGinHandler(route *Route) func(*gin.Context) {
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
			// Read body
			body, err := io.ReadAll(c.Request.Body)
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
					// Fallback for handlers that don't implement ProtobufParser
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

// InstallRouteOnGin converts an Espyna route to a Gin handler and registers it
// This utility function simplifies route registration in consumer applications
func InstallRouteOnGin(router *gin.Engine, route *Route) {
	handler := CreateGinHandler(route)

	switch route.Method {
	case "POST":
		router.POST(route.Path, handler)
	case "GET":
		router.GET(route.Path, handler)
	case "PUT":
		router.PUT(route.Path, handler)
	case "DELETE":
		router.DELETE(route.Path, handler)
	case "PATCH":
		router.PATCH(route.Path, handler)
	default:
		log.Printf("WARNING: Unsupported HTTP method: %s for route: %s", route.Method, route.Path)
		return
	}

	// log.Printf("SUCCESS: Registered: %s %s -> %s", route.Method, route.Path, route.Metadata.Name)
}
