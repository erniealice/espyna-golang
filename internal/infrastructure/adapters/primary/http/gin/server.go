//go:build gin

package gin

import (
	"github.com/gin-gonic/gin"

	"github.com/erniealice/espyna-golang/internal/composition/core"
	ginMiddleware "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/gin/middleware"
)

// Server represents a Gin HTTP server with all dependencies
type Server struct {
	router    *gin.Engine
	container *core.Container
}

// NewServer creates a new Gin HTTP server with dependencies
func NewServer(container *core.Container) *Server {
	router := gin.Default()

	// TODO: Initialize authentication middleware when services are available
	// authenticationMw := ginMiddleware.NewAuthenticationMiddleware(container.services.Auth, ...)

	// Add middleware in order: CORS -> Gzip
	// TODO: Add authentication middleware when services are available
	router.Use(ginMiddleware.CORS())
	router.Use(ginMiddleware.Gzip())

	server := &Server{
		router:    router,
		container: container,
	}

	server.setupRoutes()
	return server
}

// GetRouter returns the Gin router
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// Start starts the HTTP server on the specified address
func (s *Server) Start(addr string) error {
	return s.router.Run(addr)
}
