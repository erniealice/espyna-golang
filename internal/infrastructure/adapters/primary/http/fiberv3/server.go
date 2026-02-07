//go:build fiber_v3

package fiberv3

import (
	"github.com/gofiber/fiber/v3"

	"leapfor.xyz/espyna/internal/composition/core"
	fiberMiddleware "leapfor.xyz/espyna/internal/infrastructure/adapters/primary/http/fiberv3/middleware"
)

// Server represents a Fiber v3 HTTP server with all dependencies
type Server struct {
	app       *fiber.App
	container *core.Container
}

// NewServer creates a new Fiber v3 HTTP server with dependencies
func NewServer(container *core.Container) *Server {
	app := fiber.New(fiber.Config{
		AppName: "Espyna API v1.0 (Fiber v3)",
		ErrorHandler: func(c fiber.Ctx, err error) error {
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

	// TODO: Initialize authentication middleware when services are available
	// authenticationMw := fiberMiddleware.NewAuthenticationMiddleware(container.services.Auth, ...)

	// Add middleware in order: CORS -> Gzip
	// TODO: Add authentication middleware when services are available
	app.Use(fiberMiddleware.CORS())
	app.Use(fiberMiddleware.Gzip())

	server := &Server{
		app:       app,
		container: container,
	}

	server.setupRoutes()
	return server
}

// GetApp returns the Fiber app
func (s *Server) GetApp() *fiber.App {
	return s.app
}

// Start starts the HTTP server on the specified address
func (s *Server) Start(addr string) error {
	return s.app.Listen(addr)
}
