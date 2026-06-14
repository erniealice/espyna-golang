// Package http provides a declarative builder API for assembling an espyna-backed
// HTTP server. It absorbs the infrastructure wiring that previously lived in
// service-admin's composition layer (newAppBuilder + build) into a reusable
// Server struct.
//
// The target consumer API:
//
//	server := consumer.NewServer().
//	    WithMiddleware(mw...).
//	    WithBlocks(block1, block2, ...)
//	container := &Container{Handler: server.Handler(), Addr: server.Addr()}
//
// The Server owns the espyna container, DB/auth/storage adapters, use cases,
// and all infrastructure. Consumer apps (service-admin) only provide domain
// blocks and middleware -- they never construct adapters or use cases directly.
package http

import (
	"log"
	"net/http"
	"os"

	"github.com/erniealice/espyna-golang/consumer"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/composition/core"
	"github.com/erniealice/espyna-golang/reference"
)

// MiddlewareFunc is a standard HTTP middleware signature.
type MiddlewareFunc func(http.Handler) http.Handler

// BlockFunc configures a domain module using the shared server context.
// This is the espyna-side equivalent of pyeza.AppOption -- consumer apps
// bridge from pyeza.AppOption to BlockFunc in their composition layer.
type BlockFunc func(ctx *BlockContext) error

// BlockContext provides shared infrastructure to domain blocks during
// composition. It mirrors the fields of pyeza.AppContext but lives in
// espyna so the framework can populate it without importing pyeza.
//
// Consumer apps bridge between this and pyeza.AppContext: they create a
// pyeza.AppContext from the BlockContext fields and pass it to their
// pyeza.AppOption functions.
type BlockContext struct {
	// Container is the espyna DI container (typed, not opaque).
	Container *core.Container

	// UseCases is the use case aggregate from the container.
	UseCases *usecases.Aggregate

	// DB is the database adapter for CRUD operations.
	DB *consumer.DatabaseAdapter

	// AuthAdapter wraps the auth provider for session/login operations.
	AuthAdapter *consumer.AuthAdapter

	// StorageAdapter wraps the storage provider for file operations.
	StorageAdapter *consumer.StorageAdapter

	// EmailAdapter wraps the email provider for sending emails.
	EmailAdapter *consumer.EmailAdapter

	// RefChecker provides reference checking for deletable-state validation.
	RefChecker reference.Checker

	// Config holds application configuration read from env vars.
	Config *ServerConfig

	// SessionMiddleware is the real session middleware (non-nil for password provider).
	SessionMiddleware *consumer.SessionMiddleware

	// Routes is an app-provided route registrar. Blocks register their
	// HTTP handlers here. The Server reads them back when building the
	// final http.Handler.
	Routes RouteRegistrar
}

// RouteRegistrar is the interface that route collectors must implement.
// It matches the subset of service-admin's RouteRegistry that domain
// blocks actually call.
type RouteRegistrar interface {
	GET(path string, handler http.Handler, middlewares ...string)
	POST(path string, handler http.Handler, middlewares ...string)
	HandleFunc(method, path string, handler http.HandlerFunc, middlewares ...string)
	Redirect(path, target string)
}

// ServerConfig holds application configuration read from environment variables.
type ServerConfig struct {
	Host         string
	Port         string
	Theme        string
	Font         string
	CacheVersion string
	BusinessType string
}

// Server is the declarative builder for an espyna-backed HTTP server.
// It owns all infrastructure and exposes a fluent API for composition.
type Server struct {
	// Infrastructure (populated by NewServer)
	container      *core.Container
	useCases       *usecases.Aggregate
	db             *consumer.DatabaseAdapter
	authAdapter    *consumer.AuthAdapter
	storageAdapter *consumer.StorageAdapter
	emailAdapter   *consumer.EmailAdapter
	refChecker     reference.Checker
	sessionMw      *consumer.SessionMiddleware
	config         *ServerConfig

	// Builder state (populated by With* methods)
	middleware []MiddlewareFunc
	blocks     []BlockFunc

	// Built state
	handler http.Handler
	built   bool
}

// NewServer creates a new Server by reading all configuration from environment
// variables and initializing the espyna container with DB, auth, storage, email,
// and ID providers.
//
// godotenv must already be loaded before calling this (typically via a blank
// import in main.go: _ "github.com/joho/godotenv/autoload").
//
// This function does everything that service-admin's newAppBuilder() does in
// its "OUTPUT LAYER" phase: it creates the espyna container, extracts the
// DB/auth/storage/email adapters, and reads config from env.
func NewServer() (*Server, error) {
	cfg := loadServerConfig()

	log.Printf("Initializing espyna HTTP server")
	log.Printf("  Port: %s", cfg.Port)
	log.Printf("  Theme: %s", cfg.Theme)
	log.Printf("  Business Type: %s", cfg.BusinessType)

	// 1. Espyna container (DB + auth + storage + ID + email providers)
	espynaContainer, err := consumer.NewContainerFromEnv()
	if err != nil {
		return nil, err
	}

	// 2. Adapters
	db := consumer.NewDatabaseAdapterFromContainer(espynaContainer)
	authAdapter := consumer.NewAuthAdapterFromContainer(espynaContainer)
	if authAdapter != nil {
		log.Printf("  Auth provider: %s", authAdapter.Name())
	}
	storageAdapter := consumer.NewStorageAdapterFromContainer(espynaContainer)
	emailAdapter := consumer.NewEmailAdapterFromContainer(espynaContainer)
	if emailAdapter != nil && emailAdapter.IsEnabled() {
		log.Printf("  Email: %s provider enabled", emailAdapter.Name())
	} else {
		log.Printf("  Email: disabled")
	}

	// 3. Session middleware (password provider only)
	var sessionMw *consumer.SessionMiddleware
	if os.Getenv("CONFIG_AUTH_PROVIDER") == "password" && authAdapter != nil {
		sessionMw = consumer.NewSessionMiddleware(authAdapter)
	}

	// 4. Use cases
	useCases := espynaContainer.GetUseCases()
	if useCases == nil || useCases.Entity == nil || useCases.Entity.User == nil || useCases.Entity.Client == nil {
		log.Fatalf("Entity use cases not initialized -- check database connection (POSTGRES_HOST=%s POSTGRES_PORT=%s POSTGRES_NAME=%s)",
			os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_NAME"))
	}

	// 5. Reference checker
	refChecker := espynaContainer.RefChecker()

	return &Server{
		container:      espynaContainer,
		useCases:       useCases,
		db:             db,
		authAdapter:    authAdapter,
		storageAdapter: storageAdapter,
		emailAdapter:   emailAdapter,
		refChecker:     refChecker,
		sessionMw:      sessionMw,
		config:         cfg,
	}, nil
}

// WithMiddleware appends middleware functions to the server's middleware chain.
// Middleware is applied outermost-first: the first middleware in the list wraps
// all subsequent middleware and the final handler.
//
// Example:
//
//	server.WithMiddleware(
//	    middleware.SecurityHeaders(),
//	    middleware.Gzip,
//	    middleware.Logger,
//	    middleware.Recovery,
//	)
func (s *Server) WithMiddleware(mw ...MiddlewareFunc) *Server {
	s.middleware = append(s.middleware, mw...)
	return s
}

// WithBlocks applies domain blocks to the server. Each block receives a
// BlockContext populated with the server's shared infrastructure and
// configures its routes, labels, and views.
//
// Example:
//
//	server.WithBlocks(
//	    centymo.Block(),
//	    entydad.Block(),
//	)
func (s *Server) WithBlocks(blocks ...BlockFunc) *Server {
	s.blocks = append(s.blocks, blocks...)
	return s
}

// Build applies all blocks and middleware to produce the final http.Handler.
// This must be called after WithMiddleware and WithBlocks. It is called
// automatically by Handler() if not called explicitly.
func (s *Server) Build(routes RouteRegistrar) error {
	if s.built {
		return nil
	}

	// Create the block context that blocks will use
	ctx := &BlockContext{
		Container:         s.container,
		UseCases:          s.useCases,
		DB:                s.db,
		AuthAdapter:       s.authAdapter,
		StorageAdapter:    s.storageAdapter,
		EmailAdapter:      s.emailAdapter,
		RefChecker:        s.refChecker,
		Config:            s.config,
		SessionMiddleware: s.sessionMw,
		Routes:            routes,
	}

	// Apply blocks
	for _, block := range s.blocks {
		if err := block(ctx); err != nil {
			return err
		}
	}

	s.built = true
	return nil
}

// Container returns the underlying espyna container for advanced use cases
// (e.g., accessing providers directly, getting the database provider).
func (s *Server) Container() *core.Container {
	return s.container
}

// UseCases returns the use case aggregate.
func (s *Server) UseCases() *usecases.Aggregate {
	return s.useCases
}

// DatabaseAdapter returns the database adapter.
func (s *Server) DatabaseAdapter() *consumer.DatabaseAdapter {
	return s.db
}

// AuthAdapter returns the auth adapter (nil for providers without auth).
func (s *Server) AuthAdapter() *consumer.AuthAdapter {
	return s.authAdapter
}

// StorageAdapter returns the storage adapter.
func (s *Server) StorageAdapter() *consumer.StorageAdapter {
	return s.storageAdapter
}

// EmailAdapter returns the email adapter.
func (s *Server) EmailAdapter() *consumer.EmailAdapter {
	return s.emailAdapter
}

// RefChecker returns the reference checker.
func (s *Server) RefChecker() reference.Checker {
	return s.refChecker
}

// SessionMW returns the session middleware (nil when not using password auth).
func (s *Server) SessionMW() *consumer.SessionMiddleware {
	return s.sessionMw
}

// Config returns the server configuration.
func (s *Server) Config() *ServerConfig {
	return s.config
}

// Middleware returns the registered middleware chain.
func (s *Server) Middleware() []MiddlewareFunc {
	return s.middleware
}

// ApplyMiddleware wraps a handler with the registered middleware chain.
// Middleware is applied in reverse order so that the first registered
// middleware is the outermost wrapper.
func (s *Server) ApplyMiddleware(handler http.Handler) http.Handler {
	// Apply in reverse so first-registered is outermost
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}
	return handler
}

// Addr returns the server listen address (e.g., ":8080").
func (s *Server) Addr() string {
	return ":" + s.config.Port
}

// loadServerConfig reads application configuration from environment variables.
func loadServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:         getEnv("SERVER_HOST", "localhost"),
		Port:         getEnv("SERVER_PORT", "8080"),
		Theme:        getEnv("APP_THEME", "corporate-steel"),
		Font:         getEnv("APP_FONT", "default"),
		CacheVersion: getEnv("APP_CACHE_VERSION", "dev"),
		BusinessType: getEnv("BUSINESS_TYPE", "general"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
