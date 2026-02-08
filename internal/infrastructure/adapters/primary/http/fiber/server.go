//go:build fiber

package fiber

import (
	"github.com/gofiber/fiber/v2"

	"github.com/erniealice/espyna-golang/internal/composition/core"
	fiberMiddleware "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/fiber/middleware"
)

// Server represents a Fiber HTTP server with all dependencies
type Server struct {
	app       *fiber.App
	container *core.Container
}

// NewServer creates a new Fiber HTTP server with dependencies
func NewServer(container *core.Container) *Server {
	app := fiber.New(fiber.Config{
		AppName: "Espyna API v1.0",
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
