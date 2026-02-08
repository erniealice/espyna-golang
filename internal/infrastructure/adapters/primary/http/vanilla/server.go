//go:build vanilla

package vanilla

import (
	"net/http"

	// Composition layer
	"github.com/erniealice/espyna-golang/internal/composition/core"
	vanillaMiddleware "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/primary/http/vanilla/middleware"
)

// Server represents a vanilla HTTP server with all dependencies
type Server struct {
	mux            *http.ServeMux
	container      *core.Container
	businessTypeMw *vanillaMiddleware.BusinessTypeMiddleware
}

// NewServer creates a new vanilla HTTP server with dependencies
func NewServer(container *core.Container) *Server {
	// Initialize business type middleware with default
	defaultBusinessType := "education" // fallback
	businessTypeMw := vanillaMiddleware.NewBusinessTypeMiddleware(defaultBusinessType)

	server := &Server{
		mux:            http.NewServeMux(),
		container:      container,
		businessTypeMw: businessTypeMw,
	}

	server.setupRoutes()
	return server
}

// GetHandler returns the HTTP handler
func (s *Server) GetHandler() http.Handler {
	return s.mux
}

// Start starts the HTTP server on the specified address
func (s *Server) Start(addr string) error {
	// Wrap the mux with BusinessType, CORS, and Gzip middleware
	// Apply middleware in order: BusinessType -> CORS -> Gzip -> Mux
	handler := s.businessTypeMw.SetBusinessType(
		vanillaMiddleware.CORS(
			vanillaMiddleware.Gzip(s.mux)))
	return http.ListenAndServe(addr, handler)
}
