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
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	serviceauth "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
	"github.com/erniealice/espyna-golang/internal/composition/core"
	"github.com/erniealice/espyna-golang/reference"

	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
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
	middleware      []MiddlewareFunc
	blocks          []BlockFunc
	assetsDir       string
	reservedSlugs   []string
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

	// ── 2. Assemble the full fixed-order middleware chain via the seam ───
	// W2 (docs/plan/20260614-composition-model-a/w2-plan.md): the 11-middleware
	// fixed order is no longer hand-rolled here. finalizePreset fills the opaque
	// StandardAdmin() Preset's per-slot config closures from espyna's own use
	// cases + env; buildChain forwards it to the build-tag-selected assembler
	// (contrib/http/provider/chain_http.go under //go:build http, pass-through
	// stub otherwise) which realizes the EXACT order:
	//   SecurityHeaders → Gzip → Logger → Recovery → LoginRateLimit →
	//   Session → WorkspacePath → CSRF → ActionGuard → Timezone → businessType → mux
	// The businessType slot is owned by the chain now (it pins the same
	// "businessType" ctx key), so the inner handler passed in is the bare mux.
	preset := s.finalizePreset(consumermw.StandardAdmin())
	handler := buildChain(preset, mux)

	s.handler = handler
	return handler
}

// finalizePreset fills the opaque StandardAdmin() Preset's per-slot config
// closures from the Server's use cases + env. This is the MECHANICAL relocation
// of the pre-W2 inline middleware-config blocks: the SlugLookup / BindingResolver
// / ExecuteSwitch / PrincipalLookup / SetCSRFCookie / SessionLookup / timezone
// closures + the session-handler selection + the HMAC-secret boot guard are
// copied verbatim. The FIXED ORDER stays in the chain assembler; finalizePreset
// only fills the slots. The consumer app supplies NONE of these.
func (s *Server) finalizePreset(p consumermw.Preset) consumermw.Preset {
	// ── Cookie-secure boot resolution (W2 §5.5) ─────────────────────────
	// Resolved here (was app container.go); the chain prelude calls
	// SetSecureCookies(p.CookieSecure()) ONCE before any cookie writer is built.
	cookieSecure := consumermw.CookieSecureFromEnv(os.Getenv)

	// ── Timezone slot config ────────────────────────────────────────────
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

	// ── HMAC secret for CSRF + action guard ─────────────────────────────
	hmacSecret := consumermw.SecretFromEnv(os.Getenv)

	// ── Boot guard: password provider requires an HMAC secret ───────────
	// Fatal BEFORE serving, agnostic, ahead of the chain build. Stays here (not
	// in the chain assembler) so it fatals regardless of the server-provider tag.
	if guardProvider := os.Getenv("CONFIG_AUTH_PROVIDER"); guardProvider == "password" {
		if hmacSecret == "" {
			log.Fatalf("FATAL: CONFIG_AUTH_PROVIDER=%s requires an HMAC secret for the "+
				"workspace action-guard + CSRF middleware, but neither %s nor %s is set. "+
				"Set one before boot — refusing to start with /action/* mutations unprotected.",
				guardProvider, consumermw.EnvKeyWorkspaceFormHMAC, consumermw.EnvKeyFallbackHMAC)
		}
	}

	// ── CSRF slot config — workspace/session claim readers wired to the
	// consumer context so the v1 token claim validates against the live session.
	csrfCfg := consumermw.CSRFConfig{
		Secret: []byte(hmacSecret),
		SessionToken: func(r *http.Request) string {
			return consumer.GetSessionTokenFromContext(r.Context())
		},
		WorkspaceID: func(r *http.Request) string {
			return consumer.GetWorkspaceIDFromContext(r.Context())
		},
	}
	log.Printf("  CSRF: middleware configured (secret len=%d)", len(hmacSecret))

	// ── Action guard slot config — the HMAC secret + session workspace_id
	// reader drive the signed _workspace_id form-field check on /action/*
	// mutations. Empty secret → the impl is a pass-through (the boot guard above
	// already fatals for a real auth provider with no secret).
	agCfg := consumermw.ActionGuardConfig{
		Secret: []byte(hmacSecret),
		SessionWorkspaceID: func(ctx context.Context) string {
			return consumer.GetWorkspaceIDFromContext(ctx)
		},
	}
	if hmacSecret != "" {
		log.Printf("  ActionGuard: middleware configured (signed _workspace_id form-field guard active)")
	} else {
		log.Printf("  ActionGuard: DISABLED (no %s / %s in env)", consumermw.EnvKeyWorkspaceFormHMAC, consumermw.EnvKeyFallbackHMAC)
	}

	// ── Workspace path slot config — fully wired from espyna's OWN use cases
	// (ResolveWorkspaceBySlug, ResolveBinding, SwitchPrincipal,
	// LookupSessionPrincipal) + the contrib CSRF-cookie issuer. The consumer
	// app supplies NONE of these.
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
		// Q-WS-13: pin the URL-canonical workspace_id into ctx so downstream
		// guards (CSRF claim, action guard, view adapter) read the URL value,
		// not the stale session-injected one.
		WithWorkspaceID: consumer.WithWorkspaceID,
		SlugCacheTTL:    5 * time.Minute,
		// Per-user URL-driven rotation cap (matches the pre-migration value).
		RotationRateLimitPerMin: 10,
	}

	// Slug → workspace_id (Entity.Workspace.ResolveWorkspaceBySlug — skips
	// authcheck; this is a pre-auth middleware concern, the middleware's own
	// LRU sits above it).
	if s.useCases != nil && s.useCases.Entity != nil &&
		s.useCases.Entity.Workspace != nil &&
		s.useCases.Entity.Workspace.ResolveWorkspaceBySlug != nil {
		resolveUC := s.useCases.Entity.Workspace.ResolveWorkspaceBySlug
		wpCfg.SlugLookup = func(ctx context.Context, slug string) (string, error) {
			return resolveUC.Execute(ctx, slug)
		}
	}

	// Session principal hint + binding resolver + switch primitive — all from
	// service.Auth. PrincipalLookup surfaces the session's (kind, id) so the
	// BindingResolver stays in the session's lane (A3: no auto-elect by
	// privilege); ExecuteSwitch rotates atomically through SwitchPrincipal.
	if s.useCases != nil && s.useCases.Service != nil && s.useCases.Service.Auth != nil {
		auth := s.useCases.Service.Auth

		if auth.LookupSessionPrincipal != nil {
			lookupUC := auth.LookupSessionPrincipal
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

		// BindingResolver: ResolveBinding use case → neutral WorkspaceBinding.
		// The use case applies the A3 resolution policy and returns its own
		// sentinels (serviceauth.ErrAmbiguousBinding / serviceauth.ErrNoBinding).
		// We map them to the agnostic sentinels so the impl renders the CORRECT
		// fail-closed branch: ambiguous → picker (NO auto-elect by privilege —
		// security invariant A3); no binding → unified not-found. We call the
		// use case DIRECTLY rather than via consumer.BuildBindingResolveFn
		// because that helper collapses BOTH sentinels into one generic "no
		// binding" error, which would silently route the ambiguous case to
		// not-found instead of the picker (a fail-open of the A3 invariant).
		if auth.ResolveBinding != nil {
			resolveUC := auth.ResolveBinding
			wpCfg.BindingResolver = func(ctx context.Context, userID, workspaceID string, kind int32, principalID string) (*consumermw.WorkspaceBinding, error) {
				pb, err := resolveUC.Execute(ctx, userID, workspaceID, principaltypepb.PrincipalType(kind), principalID)
				if err != nil {
					switch {
					case errors.Is(err, serviceauth.ErrAmbiguousBinding):
						return nil, consumermw.ErrAmbiguousBinding
					case errors.Is(err, serviceauth.ErrNoBinding):
						return nil, consumermw.ErrNoBinding
					default:
						// Infrastructure failure → propagate so the impl returns
						// 500 (must NOT leak as a silent not-found / fail-open).
						return nil, err
					}
				}
				if pb == nil {
					return nil, consumermw.ErrNoBinding
				}
				pd := consumer.ProtoPrincipalToData(pb)
				return principalDataToBinding(&pd), nil
			}
		}

		// ExecuteSwitch: SwitchPrincipal use case via consumer.ExecutePrincipalSwitch.
		// URL-driven (UseCase derived from the rotation/in-place delta), audited,
		// audit-failure rolls back the rotation (red-team A-4).
		if auth.SwitchPrincipal != nil {
			switchUC := auth.SwitchPrincipal
			wpCfg.ExecuteSwitch = func(
				ctx context.Context,
				userID, token string,
				binding *consumermw.WorkspaceBinding,
				urlActingAs string,
				requestURL, referer, secFetchSite, userAgent string,
			) (*consumermw.WorkspaceSwitchResult, error) {
				if binding == nil {
					return nil, consumermw.ErrNoBinding
				}
				// Interpret the URL /as/{id} value as client vs supplier from
				// the resolved binding's kind (the load-bearing fix against
				// silent target-misrouting — without this the switch primitive
				// would default to ActingAsTargets[0]).
				var actingAsClientID, actingAsSupplierID string
				if urlActingAs != "" {
					switch binding.Kind {
					case consumermw.BindingKindClientDelegate:
						actingAsClientID = urlActingAs
					case consumermw.BindingKindSupplierDelegate:
						actingAsSupplierID = urlActingAs
					}
				}
				res, err := consumer.ExecutePrincipalSwitch(ctx, switchUC, consumer.PrincipalSwitchInput{
					UserID:             userID,
					Token:              token,
					TargetPrincipal:    bindingToPrincipalData(binding),
					ActingAsClientID:   actingAsClientID,
					ActingAsSupplierID: actingAsSupplierID,
					// URLDriven=true + empty UseCase → the adapter derives the
					// discriminator (switch_url_rotate / _acting_as_inplace /
					// _principal_inplace) from what actually changed.
					URLDriven:    true,
					RequestURL:   requestURL,
					Referer:      referer,
					SecFetchSite: secFetchSite,
					UserAgent:    userAgent,
					RequireAudit: true,
				})
				if err != nil {
					return nil, err
				}
				if res == nil {
					return nil, nil
				}
				return &consumermw.WorkspaceSwitchResult{NewToken: res.NewToken, RedirectURL: res.RedirectURL}, nil
			}
		}
	}

	// SetCSRFCookie: issue a fresh workspace-claim CSRF cookie whenever the
	// workspace_path middleware rotates the session (C2). Disabled (empty
	// secret) → IssueWorkspaceCSRFCookie no-ops on an opaque token; the impl
	// only calls this on a real rotation.
	if hmacSecret != "" {
		csrfSecret := []byte(hmacSecret)
		wpCfg.SetCSRFCookie = func(w http.ResponseWriter, newSessionToken, newWorkspaceID string) {
			issueWorkspaceCSRFCookie(w, csrfSecret, newSessionToken, newWorkspaceID)
		}
	}

	// ── Session handler selection ───────────────────────────────────────
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

	// ── SecurityHeaders slot config ─────────────────────────────────────
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

	// ── Fill the Preset slots ───────────────────────────────────────────
	// WithMiddleware extras wrap OUTSIDE the fixed core only (never spliced
	// between the security layers). Convert the consumer/http MiddlewareFunc
	// slice into the agnostic consumermw.MiddlewareFunc slice (identical
	// underlying signature).
	extras := make([]consumermw.MiddlewareFunc, 0, len(s.middleware))
	for _, mw := range s.middleware {
		extras = append(extras, consumermw.MiddlewareFunc(mw))
	}

	return p.
		WithCookieSecure(cookieSecure).
		WithSecurity(secCfg).
		WithSession(sessionHandler).
		WithWorkspace(wpCfg).
		WithCSRF(csrfCfg).
		WithActionGuard(agCfg).
		WithTimezone(tzCfg).
		WithBusinessType(s.config.BusinessType).
		WithExtras(extras)
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
