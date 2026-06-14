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
	"context"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/erniealice/espyna-golang/consumer"
	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/composition/core"
	"github.com/erniealice/espyna-golang/reference"

	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
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
	middleware     []MiddlewareFunc
	blocks         []BlockFunc
	assetsDir      string
	reservedSlugs  []string
	catchAllHandler http.Handler

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

// WithAssets sets the directory path for serving static assets at /assets/.
// Defaults to "assets" when not set.
func (s *Server) WithAssets(path string) *Server {
	s.assetsDir = path
	return s
}

// WithReservedSlugs sets workspace slug values that are reserved and cannot
// be used as workspace slugs (e.g., "auth", "me", "portal").
func (s *Server) WithReservedSlugs(slugs ...string) *Server {
	s.reservedSlugs = append(s.reservedSlugs, slugs...)
	return s
}

// WithCatchAll sets the catch-all handler (typically an httpAdapter.Handler())
// that serves all application routes registered by domain blocks. When nil,
// a default 404 handler is used.
func (s *Server) WithCatchAll(h http.Handler) *Server {
	s.catchAllHandler = h
	return s
}

// Handler constructs the full middleware stack and returns the composed
// http.Handler ready to serve. This is the primary entry point for consumer
// apps -- it builds the mux, applies all middleware, and returns a handler
// that container.go can store directly:
//
//	return &Container{Handler: srv.Handler(), Addr: srv.Addr()}
func (s *Server) Handler() http.Handler {
	// ── 1. Mux ──────────────────────────────────────────────────────────
	assetsDir := s.assetsDir
	if assetsDir == "" {
		assetsDir = "assets"
	}

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST "+consumermw.CSPReportPath(), consumermw.NewCSPReportHandler())
	mux.HandleFunc("GET /api/notifications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, `{"notifications":[]}`)
	})
	if s.catchAllHandler != nil {
		mux.Handle("/", s.catchAllHandler)
	} else {
		mux.Handle("/", http.NotFoundHandler())
	}

	// ── 2. Business type context injection ──────────────────────────────
	businessType := s.config.BusinessType
	businessTypeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "businessType", businessType)
		mux.ServeHTTP(w, r.WithContext(ctx))
	})

	// ── 3. Inner middleware (timezone → action guard → CSRF → workspace path → session → rate limit) ──
	// 3a. Timezone middleware config
	tzCfg := consumermw.TimezoneConfig{
		GetUserID: func(ctx context.Context) string {
			uid := consumer.GetUserIDFromContext(ctx)
			if uid == "" {
				uid = consumer.ExtractUserIDFromContext(ctx)
			}
			return uid
		},
		LookupTimezone: s.buildTimezoneLookup(),
	}

	// 3b. HMAC secret for CSRF + action guard
	hmacSecret := consumermw.SecretFromEnv(os.Getenv)

	// 3c. Boot guard: password provider requires an HMAC secret
	if guardProvider := os.Getenv("CONFIG_AUTH_PROVIDER"); guardProvider == "password" {
		if hmacSecret == "" {
			log.Fatalf("FATAL: CONFIG_AUTH_PROVIDER=%s requires an HMAC secret for the "+
				"workspace action-guard + CSRF middleware, but neither %s nor %s is set. "+
				"Set one before boot — refusing to start with /action/* mutations unprotected.",
				guardProvider, consumermw.EnvKeyWorkspaceFormHMAC, consumermw.EnvKeyFallbackHMAC)
		}
	}

	// 3d. CSRF middleware config
	csrfCfg := consumermw.CSRFConfig{
		Secret: []byte(hmacSecret),
	}
	log.Printf("  CSRF: middleware configured (secret len=%d)", len(hmacSecret))

	// 3e. Action guard middleware config (Signer is nil → pass-through
	// until the signer implementation moves to espyna; container.go can
	// override via WithMiddleware if needed).
	agCfg := consumermw.ActionGuardConfig{}
	if hmacSecret != "" {
		log.Printf("  ActionGuard: HMAC key available (will be active when signer wired)")
	}

	// 3f. Workspace path middleware config
	reservedSet := make(map[string]bool, len(s.reservedSlugs))
	for _, slug := range s.reservedSlugs {
		reservedSet[slug] = true
	}
	wpCfg := consumermw.WorkspacePathConfig{
		SessionLookup: func(r *http.Request) (userID, workspaceID, token string, ok bool) {
			ctx := r.Context()
			userID = consumer.GetUserIDFromContext(ctx)
			workspaceID = consumer.GetWorkspaceIDFromContext(ctx)
			token = consumer.GetSessionTokenFromContext(ctx)
			ok = userID != ""
			return
		},
		IsReservedSlug: func(slug string) bool {
			return reservedSet[slug]
		},
	}

	// Wire slug lookup if the Workspace use case is available.
	if s.useCases != nil && s.useCases.Entity != nil &&
		s.useCases.Entity.Workspace != nil &&
		s.useCases.Entity.Workspace.ResolveWorkspaceBySlug != nil {
		resolveUC := s.useCases.Entity.Workspace.ResolveWorkspaceBySlug
		wpCfg.SlugLookup = func(ctx context.Context, slug string) (string, error) {
			return resolveUC.Execute(ctx, slug)
		}
	}

	// Wire principal lookup if the Auth service is available.
	if s.useCases != nil && s.useCases.Service != nil && s.useCases.Service.Auth != nil &&
		s.useCases.Service.Auth.LookupSessionPrincipal != nil {
		lookupUC := s.useCases.Service.Auth.LookupSessionPrincipal
		wpCfg.PrincipalLookup = func(r *http.Request) (kind int32, principalID string) {
			token := consumer.GetSessionTokenFromContext(r.Context())
			if token == "" {
				return 0, ""
			}
			resp, err := lookupUC.Execute(r.Context(), &authpb.LookupSessionPrincipalRequest{Token: token})
			if err != nil || resp == nil {
				return 0, ""
			}
			return int32(resp.Kind), resp.PrincipalId
		}
	}

	// 3g. Session handler
	var sessionHandler consumermw.SessionHandler
	if os.Getenv("CONFIG_AUTH_PROVIDER") == "mock" {
		if s.useCases != nil && s.useCases.Service != nil && s.useCases.Service.Auth != nil &&
			s.useCases.Service.Auth.AuthenticateSession != nil &&
			s.useCases.Service.Auth.IssueSession != nil {
			testUserID := getEnv("TEST_USER_ID", "superadmin-001")
			testEmail := getEnv("TEST_USER_EMAIL", "admin@ichizen.leapfor.xyz")
			testWsUserID := getEnv("TEST_WORKSPACE_USER_ID", "ws-user-001")
			defaultWsID := getEnv("DEFAULT_WORKSPACE_ID", "default-workspace")
			mockMw := consumer.NewMockSessionMiddleware(
				s.container.GetUseCases(), testUserID, testEmail, testWsUserID, defaultWsID,
			)
			sessionHandler = consumermw.MockSessionHandler(mockMw.Handle)
			log.Printf("  Session: mock (auto-create for %s)", testUserID)
		} else {
			log.Printf("  Session: mock requested but Auth use cases unavailable — running without session middleware")
		}
	} else if s.sessionMw != nil {
		sessionHandler = s.sessionMw
		log.Printf("  Session: password session middleware active")
	}

	// ── 4. Compose the middleware stack ─────────────────────────────────
	// Order (outermost → innermost):
	//   SecurityHeaders → Gzip → Logger → Recovery → LoginRateLimit →
	//   Session → WorkspacePath → CSRF → ActionGuard → Timezone → mux
	secCfg := consumermw.SecurityHeadersConfigFromEnv(os.Getenv)
	cspMode := "report-only"
	if secCfg.CSPEnforce {
		cspMode = "ENFORCING (CONFIG_SECURITY_CSP_ENFORCE set)"
	}
	if secCfg.HSTSEnabled {
		log.Printf("  SecurityHeaders: CSP %s + HSTS enabled", cspMode)
	} else {
		log.Printf("  SecurityHeaders: CSP %s; HSTS disabled", cspMode)
	}

	handler := consumermw.SecurityHeaders(secCfg)(
		consumermw.Gzip()(
			consumermw.Logger()(
				consumermw.Recovery()(
					consumermw.LoginRateLimit()(
						consumermw.Session(sessionHandler)(
							consumermw.WorkspacePath(wpCfg)(
								consumermw.CSRF(csrfCfg)(
									consumermw.ActionGuard(agCfg)(
										consumermw.Timezone(tzCfg)(
											businessTypeHandler))))))))))

	// ── 5. Apply any extra middleware registered via WithMiddleware ──────
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}

	s.handler = handler
	return handler
}

// buildTimezoneLookup returns a closure that resolves a user's timezone
// preference from the User use case. Returns nil when the use case is
// unavailable (the Timezone middleware falls back to DefaultTimezone).
func (s *Server) buildTimezoneLookup() func(ctx context.Context, userID string) string {
	if s.useCases == nil || s.useCases.Entity == nil || s.useCases.Entity.User == nil ||
		s.useCases.Entity.User.ReadUser == nil {
		return nil
	}
	readUserUC := s.useCases.Entity.User.ReadUser
	return func(ctx context.Context, userID string) string {
		if userID == "" {
			return ""
		}
		resp, err := readUserUC.Execute(ctx, &userpb.ReadUserRequest{
			Data: &userpb.User{Id: userID},
		})
		if err != nil || resp == nil || len(resp.GetData()) == 0 {
			return ""
		}
		return resp.GetData()[0].GetTimezone()
	}
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
