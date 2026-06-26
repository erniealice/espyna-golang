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
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/consumer"
	consumerapp "github.com/erniealice/espyna-golang/consumer/app"
	compose "github.com/erniealice/espyna-golang/consumer/compose"
	consumermw "github.com/erniealice/espyna-golang/consumer/http/middleware"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	serviceauth "github.com/erniealice/espyna-golang/internal/application/usecases/service/auth"
	"github.com/erniealice/espyna-golang/internal/composition/core"
	"github.com/erniealice/espyna-golang/ports"

	principaltypepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/principal_type"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/service/auth"
)

// MiddlewareFunc is a standard HTTP middleware signature. (Demote-to-internal
// candidate per A.1; still referenced by the legacy Handler() extras conversion
// until Wave B retires the old surface.)
type MiddlewareFunc func(http.Handler) http.Handler

// BlockFunc configures a domain module using the shared server context.
// This is the espyna-side equivalent of consumerapp.AppOption -- consumer apps
// bridge from consumerapp.AppOption to BlockFunc in their composition layer.
//
// DEAD (A.1 — slated for Wave B/C deletion): the fluent WithBlocks now takes
// consumerapp.AppOption directly. Retained ONLY so the legacy Build(routes) path and
// any not-yet-migrated caller keep compiling; not used by the fluent chain.
type BlockFunc func(ctx *BlockContext) error

// BlockContext provides shared infrastructure to domain blocks during
// composition. It mirrors the fields of consumerapp.AppContext but lives in
// espyna so the framework can populate it without importing pyeza.
//
// DEAD (A.1 — slated for Wave B/C deletion). espyna consumer/ already imports
// pyeza (nav_resolver etc.), so the fluent path uses *consumerapp.AppContext directly;
// this hand-rolled mirror is retained only for the legacy Build(routes) path.
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
	RefChecker ports.Checker

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
//
// DEAD (A.1 — slated for Wave B/C deletion): routes are no longer app-owned in
// the fluent path; blocks register into the consumerapp.AppContext.Routes the Server
// provides. Retained only for the legacy Build(routes) / BlockContext path.
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
	refChecker     ports.Checker
	sessionMw      *consumer.SessionMiddleware
	config         *ServerConfig

	// Builder state (populated by With* methods)
	middleware      []MiddlewareFunc
	blocks          []BlockFunc
	assetsDir       string
	reservedSlugs   []string
	catchAllHandler http.Handler

	// Fluent-API builder state (A.1). Populated by WithApp / WithMiddleware /
	// WithBlocks; consumed by Build / MustBuild. Kept SEPARATE from the legacy
	// fields above so the old Handler()-based app path (service-admin until Wave
	// B) and the new fluent path coexist without interference.
	appConfig    AppConfig
	appConfigSet bool
	presetSet    bool
	preset       consumermw.Preset
	appBlocks    []consumerapp.AppOption

	// Wave B D1: the COMPLETE app-supplied UI/labels bundle (renderer, labels,
	// sidebars, translations, route-rewriter, auth labels). Populated by
	// WithUI; stamped into appCtx.UI by Build() and read + fail-loud-asserted
	// by finalizeHTTPAdapter. NOT used by the legacy Handler() path (the app
	// still owns its own renderer/labels until D2).
	appUI *consumerapp.AppUIBundle

	// Wave B D2c: the app injection hook(s). Registered by WithAppContext;
	// invoked by Build() AFTER espyna stamps the auth-chain slots it OWNS
	// (SessionManager/AuthAdapter/SecureWorkspaceSwitch*) and BEFORE the opt
	// loop. The app uses it to stamp its ~22 cross-cutting closures (SqlDB,
	// UploadFile, SendEmail, GetDashboardData, doc-template CRUD, etc.) — the
	// build-tag-selected adapters espyna does NOT know about — onto the same
	// *consumerapp.AppContext the domain EngineBlocks read. NOT invoked by the legacy
	// Handler() path (these run only inside Build()).
	appContextHooks []func(*consumerapp.AppContext)

	// Wave B D1: memoized security resolution (HMAC secret + cookie-secure
	// policy + CSRF issuer + the non-mock-empty-secret boot fatal). Resolved
	// ONCE by resolveSecurity() and shared by BOTH Build() and the legacy
	// finalizePreset()/Handler() path (codex C2 — single source of truth).
	security *securityResolution

	// Built state
	handler http.Handler
	built   bool
}

// securityResolution carries the once-resolved HMAC secret + cookie-secure
// policy + agnostic CSRF-cookie issuer closure (Wave B D1.0). resolveSecurity()
// populates it (running the non-mock-empty-secret boot fatal as a side effect);
// both Build() and finalizePreset() consume it so the env is read once and the
// values can never drift between the two paths.
type securityResolution struct {
	secret       []byte
	cookieSecure bool
	issuer       func(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string
}

// resolveSecurity resolves the HMAC secret + cookie-secure policy ONCE, runs the
// non-mock-empty-secret boot fatal, and builds the agnostic CSRFIssuer closure.
// Idempotent + memoized on the Server so Build() (pre-opt-loop) and the legacy
// finalizePreset()/Handler() path resolve the SAME values and BOTH fatal on a
// non-mock empty secret. (Wave B D1.0 / codex C2 — the legacy guard is
// PRESERVED: this is the exact behaviour previously inlined at server.go
// finalizePreset, lifted into a shared helper so neither path can boot non-mock
// with /action/* CSRF unprotected.)
func (s *Server) resolveSecurity() securityResolution {
	if s.security != nil {
		return *s.security // memoized — single source of truth
	}
	cookieSecure := consumermw.CookieSecureFromEnv(os.Getenv)
	hmacSecret := consumermw.SecretFromEnv(os.Getenv)

	// Boot guard: ANY non-mock provider requires an HMAC secret. The middleware
	// chain is FIXED for every provider (one StandardAdmin() Preset), but
	// BuildCSRF + BuildActionGuard SHORT-CIRCUIT to pass-throughs on an empty
	// secret — so a non-mock production provider booted with no HMAC secret
	// would start with /action/* workspace-claim mutations UNPROTECTED (a
	// verified fail-OPEN). The provider token is normalized like the auth
	// provider factory (strings.ToLower) so accepted dev input like "Mock"
	// does not wrongly fatal. mock (dev-only) stays bootable.
	guardProvider := strings.ToLower(os.Getenv("CONFIG_AUTH_PROVIDER"))
	if guardProvider != "mock" && hmacSecret == "" {
		log.Fatalf("FATAL: CONFIG_AUTH_PROVIDER=%q runs the workspace action-guard + CSRF "+
			"chain but no HMAC secret is set (neither %s nor %s). Refusing to start with "+
			"/action/* mutations unprotected. Set one before boot, or use "+
			"CONFIG_AUTH_PROVIDER=mock for dev.",
			guardProvider, consumermw.EnvKeyWorkspaceFormHMAC, consumermw.EnvKeyFallbackHMAC)
	}

	var issuer func(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string
	if hmacSecret != "" {
		issuer = func(w http.ResponseWriter, secret []byte, sessionToken, workspaceID string) string {
			return consumermw.IssueWorkspaceCSRFCookie(w, secret, sessionToken, workspaceID, cookieSecure)
		}
	}

	s.security = &securityResolution{
		secret:       []byte(hmacSecret),
		cookieSecure: cookieSecure,
		issuer:       issuer,
	}
	return *s.security
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

	// 3. Session middleware (server-side session providers: password + firebase).
	// Both manage server-side sessions (password = DB-backed session; firebase =
	// session cookie minted after VerifyIDToken), so both need the real
	// SessionMiddleware. mock uses its own NewMockSessionMiddleware path below.
	// entydad.EngineBlock FATALs on a nil session manager, so any auth provider
	// booted alongside it MUST wire one here.
	var sessionMw *consumer.SessionMiddleware
	if ap := os.Getenv("CONFIG_AUTH_PROVIDER"); (ap == "password" || ap == "firebase") && authAdapter != nil {
		sessionMw = consumer.NewSessionMiddleware(authAdapter)
		// Wave B D2c — COOKIE FIX: stamp the resolved Secure-flag policy from env
		// onto the session middleware so the ichizen_session cookie carries the
		// COOKIE_SECURE-derived Secure flag. NewSessionMiddleware defaults
		// CookieSecure=false; without this the password-auth session cookie
		// would drop the Secure flag in production. Previously the app set this
		// in newAppBuilder (container.go:84-89) AFTER NewServer; doing it here
		// makes the Build()/fluent path correct on its own. ADDITIVE + SAFE for
		// the legacy Handler() path: the app still overrides this same value to
		// the identical CookieSecureFromEnv result until the collapse lands, so
		// the legacy path sees no behaviour change. LoginURL already defaults to
		// "/auth/login" (== entydad.AuthLoginURL), so no override is needed.
		sessionMw.CookieSecure = consumermw.CookieSecureFromEnv(os.Getenv)
	}

	// 4. Use cases
	useCases := espynaContainer.GetUseCases()
	if useCases == nil || useCases.Entity == nil || useCases.Entity.User == nil || useCases.Entity.Client == nil {
		log.Fatalf("Entity use cases not initialized -- check database connection (DATABASE_POSTGRES_HOST=%s DATABASE_POSTGRES_PORT=%s DATABASE_POSTGRES_DBNAME=%s)",
			os.Getenv("DATABASE_POSTGRES_HOST"), os.Getenv("DATABASE_POSTGRES_PORT"), os.Getenv("DATABASE_POSTGRES_DBNAME"))
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

// MustNewServer is the fluent-chain entry: it boots the espyna container and
// log.Fatals on any boot error, returning a non-nil *Server so the
// .WithApp(...).WithUI(...).WithMiddleware(...).WithBlocks(...).MustBuild()
// chain is buildable as one expression (Wave B D1.3 / codex C3). NewServer()
// keeps its (*Server, error) signature for callers that handle the error.
func MustNewServer() *Server {
	s, err := NewServer()
	if err != nil {
		log.Fatalf("FATAL: espyna NewServer: %v", err)
	}
	return s
}

// WithApp records the application-level configuration (A.1 #3). Fluent: returns
// the Server for chaining. Read by Build() when finalizing the chain (BusinessType,
// reserved slugs, asset root, feature flags).
func (s *Server) WithApp(cfg AppConfig) *Server {
	s.appConfig = cfg
	s.appConfigSet = true
	if len(cfg.ReservedWorkspaceSlugs) > 0 {
		s.reservedSlugs = append(s.reservedSlugs, cfg.ReservedWorkspaceSlugs...)
	}
	if cfg.AssetRoot != "" {
		s.assetsDir = cfg.AssetRoot
	}
	if cfg.DefaultBusinessType != "" && s.config != nil {
		s.config.BusinessType = cfg.DefaultBusinessType
	}
	return s
}

// WithMiddleware records the opaque middleware Preset (A.1 #4). The Preset is the
// fixed-order security chain (StandardAdmin()); the consumer app cannot reorder
// or splice between its security layers — finalizePreset fills the per-slot
// config from espyna's own use cases. Fluent: returns the Server for chaining.
//
// (A.1 replaces the legacy WithMiddleware(...MiddlewareFunc); no caller used the
// raw-func variant. The legacy extras slice is still threaded by finalizePreset
// for the old Handler() path, but it is never populated now.)
func (s *Server) WithMiddleware(preset consumermw.Preset) *Server {
	s.preset = preset
	s.presetSet = true
	return s
}

// WithBlocks records the domain blocks as consumerapp.AppOption values (A.1 #5).
// espyna consumer/ already imports pyeza, so blocks register directly into the
// consumerapp.AppContext.Routes the Server provides — no BlockFunc/BlockContext mirror.
// Fluent: returns the Server for chaining.
func (s *Server) WithBlocks(opts ...consumerapp.AppOption) *Server {
	s.appBlocks = append(s.appBlocks, opts...)
	return s
}

// WithUI records the COMPLETE app-supplied UI/labels bundle (Wave B D1.1). The
// app builds it once from its own renderer / labels / sidebars / translations /
// route-rewriter / auth labels and passes it here; Build() stamps it into
// appCtx.UI (and mirrors Common/Table/Translations into the existing typed
// slots), and finalizeHTTPAdapter reads + fail-loud-asserts every field. Fluent:
// returns the Server for chaining. The renderer / sidebars / labels never enter
// espyna — they import app + domain template FSes and app-specific sidebar code
// (codex C4 boundary).
func (s *Server) WithUI(ui *consumerapp.AppUIBundle) *Server {
	s.appUI = ui
	return s
}

// WithAppContext registers an app injection hook (Wave B D2c). Fluent: returns
// the Server for chaining. The hook is invoked by Build() AFTER espyna stamps
// the auth-chain slots it OWNS onto the *consumerapp.AppContext (SessionManager,
// AuthAdapter, and the espyna-derivable SecureWorkspaceSwitch* siblings) and
// BEFORE the domain-block opt loop runs — so the app can stamp its ~22
// cross-cutting closures (SqlDB / UploadFile / SendEmail / GetDashboardData /
// doc-template CRUD / GetUsersByRoleID / HashPassword / ... captured from the
// app's build-tag-selected adapters) onto the SAME context the domain
// EngineBlocks read. espyna does NOT know providers; the hook is the seam where
// the app injects them. Multiple hooks are invoked in registration order. nil
// hooks are skipped. NOT invoked by the legacy Handler() path.
func (s *Server) WithAppContext(fn func(*consumerapp.AppContext)) *Server {
	if fn != nil {
		s.appContextHooks = append(s.appContextHooks, fn)
	}
	return s
}

// Build boots the chain and produces the *Container (A.1 #6). It:
//  1. builds a *consumerapp.AppContext from the Server's already-booted infra,
//  2. applies each consumerapp.AppOption (domain block) to it,
//  3. finalizes the opaque Preset's per-slot config from espyna's use cases,
//  4. assembles the fixed-order middleware chain via the existing BuildChain seam.
//
// Infra boot already happened in NewServer (A.1 NOTE: NewServer keeps its
// (*Server, error) signature so the service-admin app — which still calls the old
// surface until Wave B — keeps compiling; the fluent methods are layered on top).
func (s *Server) Build() (*Container, error) {
	// 1. Resolve CSRF secret + cookie-secure + issuer + run the non-mock-empty-
	//    secret boot fatal via the ONE shared helper (D1.0) BEFORE the opt loop,
	//    so blocks (entydad auth, D2) read the SAME values from the AppContext
	//    slots and the legacy Handler() path shares the guard.
	sec := s.resolveSecurity()

	// 2. The COMPLETE app-supplied UI/labels bundle is required to finalize the
	//    HTTP adapter (codex round-2). A non-mock binary must NEVER boot with a
	//    blank UI, so a missing WithUI(...) is a fail-loud boot fatal here.
	if s.appUI == nil {
		log.Fatalf("FATAL espyna Build(): WithUI(...) was not called — the consumer/http " +
			"adapter cannot be finalized without the app-supplied renderer/labels/translations " +
			"bundle. Refusing to boot with a blank UI.")
	}

	// 3. AppContext from the Server's booted infra. Routes = the Server's REAL
	//    registry (it satisfies pyeza's view.RouteRegistrar + the auth-module
	//    HandleFunc/Redirect surface). CSRF slots pre-stamped from
	//    resolveSecurity(); the app-supplied UI bundle is stamped from WithUI,
	//    and the three pre-existing typed slots (Common/Table/Translations) are
	//    mirrored from the bundle so blocks reading them get the SAME values the
	//    adapter does. Translations comes from appCtx.UI.Translations (the
	//    Server has NO s.translations field).
	routes := newRouteRegistry()
	appCtx := &consumerapp.AppContext{
		Routes:        routes,
		BusinessType:  s.config.BusinessType,
		Container:     s.container,
		UseCases:      s.useCases,
		DB:            s.db,
		RefChecker:    s.refChecker,
		ComposeResult: compose.NewResult(),
		CSRFSecret:    sec.secret,
		CookieSecure:  sec.cookieSecure,
		CSRFIssuer:    sec.issuer,
		UI:            s.appUI,
		Common:        mustCommonLabels(s.appUI.CommonLabels),
		Table:         mustTableLabels(s.appUI.TableLabels),
		Translations:  s.appUI.Translations,
	}

	// 3b. STAMP the auth-chain slots espyna OWNS onto appCtx, BEFORE the opt
	//     loop (Wave B D2c). The entydad EngineBlock reconstructs the auth.Deps
	//     from these — it reads ctx.SessionManager + ctx.AuthAdapter and derives
	//     its Infra.SecureSwitch from the SAME session middleware instance. By
	//     stamping s.sessionMw / s.authAdapter here (the SAME instances espyna
	//     booted in NewServer, with the COOKIE_SECURE-stamped CookieSecure), the
	//     block reuses them — no cookie/config drift between the middleware chain
	//     and the auth chain. CSRFSecret/CookieSecure/CSRFIssuer were already
	//     pre-stamped above from resolveSecurity(); these add the session/credential
	//     half. The SecureWorkspaceSwitch* siblings espyna can derive (ResolveUserID
	//     from the request, SetSessionCookie from s.sessionMw) are stamped too for
	//     parity with the documented prescription — the live EngineBlock path
	//     derives its own SecureSwitch internally so these are additive (only the
	//     legacy identity.go Block() path reads them). The SecureWorkspaceSwitch
	//     closure itself is NOT stamped: it requires entydad's switch primitive +
	//     use cases, which the block builds internally (espyna does not import
	//     entydad).
	appCtx.SessionManager = s.sessionMw
	appCtx.AuthAdapter = s.authAdapter
	if s.sessionMw != nil {
		appCtx.SecureWorkspaceSwitchResolveUserID = consumer.SecureSidebarResolveUserID
		sessionMw := s.sessionMw
		appCtx.SecureWorkspaceSwitchSetSessionCookie = func(w http.ResponseWriter, token string) {
			sessionMw.SetSessionCookie(w, token)
		}
	}

	// 3c. INVOKE the registered app injection hooks (Wave B D2c). The app stamps
	//     its ~22 cross-cutting closures (SqlDB / UploadFile / SendEmail / doc
	//     template CRUD / dashboard / role-user / hash-password / ... captured
	//     from its build-tag-selected adapters) onto the SAME appCtx the domain
	//     EngineBlocks read below. Runs AFTER the auth-chain stamps (so the app
	//     can read them if needed) and BEFORE the opt loop (so blocks see the
	//     stamped closures). espyna does NOT know providers — the app owns this.
	for _, hook := range s.appContextHooks {
		if hook != nil {
			hook(appCtx)
		}
	}

	// 4. Apply domain blocks (consumerapp.AppOption). Each block registers routes into
	//    appCtx.Routes, merges Nav into appCtx.ComposeResult, and (entydad, D2)
	//    sets appCtx.WorkspaceLoader / AuthDeps / SecureWorkspaceSwitch*.
	for _, opt := range s.appBlocks {
		if opt == nil {
			continue
		}
		if err := opt(appCtx); err != nil {
			return nil, err
		}
	}

	// 5. Finalize the HTTP adapter from the now-populated AppContext (the
	//    framework-generic relocation of the app's old build(); the renderer /
	//    sidebars / labels are read from the APP-SUPPLIED bundle).
	handler, err := s.finalizeHTTPAdapter(appCtx, routes)
	if err != nil {
		return nil, err
	}
	return &Container{handler: handler, addr: s.Addr()}, nil
}

// MustBuild is the panic-on-error convenience over Build (A.1 #7).
func (s *Server) MustBuild() *Container {
	c, err := s.Build()
	if err != nil {
		log.Fatalf("Server.Build failed: %v", err)
	}
	return c
}

// assembleHandler builds the mux and applies the fixed-order chain via the seam.
// Shared by the legacy Handler() and the fluent Build(). When a Preset was set
// via WithMiddleware it is used as the base; otherwise StandardAdmin() is used.
func (s *Server) assembleHandler() http.Handler {
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

	base := consumermw.StandardAdmin()
	if s.presetSet {
		base = s.preset
	}
	preset := s.finalizePreset(base)
	return buildChain(preset, mux)
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
func (s *Server) RefChecker() ports.Checker {
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
	// ── Assemble the mux + the full fixed-order middleware chain via the seam.
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
	// Shares assembleHandler() with the fluent Build() path.
	handler := s.assembleHandler()
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
	// ── Cookie-secure + HMAC secret + CSRF issuer + the non-mock-empty-secret
	// boot fatal: resolved ONCE via the shared helper (Wave B D1.0 / codex C2).
	// The fatal NO LONGER lives inline here — it MOVED INTO resolveSecurity(),
	// which BOTH this legacy Handler() path AND the new Build() invoke, so
	// neither path can boot non-mock with /action/* CSRF unprotected. The
	// chain prelude calls SetSecureCookies(p.CookieSecure()) ONCE before any
	// cookie writer is built.
	sec := s.resolveSecurity()
	cookieSecure := sec.cookieSecure
	hmacSecret := string(sec.secret)

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
	// workspace_path middleware rotates the session (C2 — A.3.3 trigger #2). The
	// closure keeps its agnostic signature func(w http.ResponseWriter, ...) and
	// calls the AGNOSTIC no-tag consumermw.IssueWorkspaceCSRFCookie directly (it
	// computes the HMAC inline + http.SetCookie — no build-tag dispatch, no ""
	// stub). secure is captured from the resolved cookieSecure policy (single
	// source of truth). The impl only calls this on a real rotation.
	if hmacSecret != "" {
		csrfSecret := []byte(hmacSecret)
		wpCfg.SetCSRFCookie = func(w http.ResponseWriter, newSessionToken, newWorkspaceID string) {
			consumermw.IssueWorkspaceCSRFCookie(w, csrfSecret, newSessionToken, newWorkspaceID, cookieSecure)
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
