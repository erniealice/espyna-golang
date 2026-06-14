//go:build http

// workspace_path.go
//
// WorkspacePathMiddleware parses the workspace slug from /w/{slug}/* URLs,
// resolves the slug to a workspace_id, validates the user's active binding,
// optionally rotates the session when the URL workspace differs from the
// session workspace (URL is canonical for workspace_id), strips the prefix,
// and dispatches to the downstream handler.
//
// ## Steps
//
//  1. Match /w/{workspace_slug}[/as/{client_id}][/rest...] via regex.
//  2. Validate slug format (^[a-z0-9]+(?:-[a-z0-9]+)*$ length 3-30).
//  3. CSRF preflight -- gate session rotation behind Sec-Fetch-Site +
//     Sec-Fetch-Mode (Referer fallback for older browsers).
//  4. Slug -> workspace_id resolution via a process-local LRU (5 min TTL).
//  5. Read session context from upstream session middleware.
//  6. Active-binding validation. Per-request memoization prevents N+1.
//  7. Slug-not-found and binding-missing collapse to the same response
//     shape (303 to /auth/select-workspace-role).
//  8. Optional /as/{client_id} extraction.
//  9. Per-user rotation rate limit (10/min default).
//  10. URL-driven rotation when URL workspace differs from session workspace.
//  11. SameSite=Strict on rotated session cookies.
//  12. Bind workspace_id into request context.
//  13. Strip prefix and dispatch.
//
// SECURITY-CRITICAL.
package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/erniealice/espyna-golang/consumer/http/httpctx"
	"github.com/erniealice/pyeza-golang/render"
)

// defaultSessionCookieName mirrors consumer.DefaultSessionCookieName.
// Duplicated here to avoid the import cycle (consumer -> contrib/http ->
// consumer). The wiring layer should pass the CookieName explicitly.
const defaultSessionCookieName = "ichizen_session"

// --- Principal types (decoupled from service-admin) ---
//
// These types mirror the adapthttp.Principal/PrincipalType types from
// service-admin. The wiring layer (container.go) bridges between these
// and the concrete types via closures.

// PrincipalType represents the kind of principal binding.
//
// The integer values are PROTO-ALIGNED: they match esqyma's
// domain.entity.principal_type.PrincipalType enum (and pyeza render's mirror)
// VERBATIM, so the int32 binding kind flows across the consumer/http <-> contrib
// boundary with no translation table. This is load-bearing for wsCanActAs:
// SUPPLIER_DELEGATE is 6 (NOT 5) in the proto, so an iota-based enum would
// silently misclassify supplier delegates and break /as/{id} for them.
type PrincipalType int

const (
	// PrincipalTypeUnspecified is the zero value (no hint). (proto 0)
	PrincipalTypeUnspecified PrincipalType = 0
	// PrincipalTypeOperatorOwner is the workspace owner / top-level admin. (proto 1)
	PrincipalTypeOperatorOwner PrincipalType = 1
	// PrincipalTypeOperatorStaff is an operator staff binding. (proto 2)
	PrincipalTypeOperatorStaff PrincipalType = 2
	// PrincipalTypeClient is a client binding. (proto 3)
	PrincipalTypeClient PrincipalType = 3
	// PrincipalTypeClientDelegate is a delegate-for-client binding. (proto 4)
	PrincipalTypeClientDelegate PrincipalType = 4
	// PrincipalTypeSupplier is a supplier binding. (proto 5)
	PrincipalTypeSupplier PrincipalType = 5
	// PrincipalTypeSupplierDelegate is a delegate-for-supplier binding. (proto 6)
	PrincipalTypeSupplierDelegate PrincipalType = 6
)

// ActingAsTarget is one target a delegate may act for.
type ActingAsTarget struct {
	ID          string
	WorkspaceID string
}

// Principal represents a user's binding in a workspace.
type Principal struct {
	Type            PrincipalType
	ID              string
	WorkspaceID     string
	ActingAsTargets []ActingAsTarget
}

// --- Sentinel errors ---

var (
	// ErrSlugFormatInvalid is returned when the slug fails format validation.
	ErrSlugFormatInvalid = errors.New("workspace_path: slug format invalid")

	// ErrSlugReserved is returned when the slug collides with a reserved segment.
	ErrSlugReserved = errors.New("workspace_path: slug reserved")

	// ErrSlugNotFound is returned when the slug doesn't resolve to a workspace.
	ErrSlugNotFound = errors.New("workspace_path: slug not found")

	// ErrNoBindingInWorkspace is returned when the user has no active binding.
	ErrNoBindingInWorkspace = errors.New("workspace_path: user has no active binding in workspace")

	// ErrAmbiguousBinding is returned when the user has multiple bindings
	// and no session hint can disambiguate.
	ErrAmbiguousBinding = errors.New("workspace_path: ambiguous binding")

	// ErrCSRFPreflightFailed is returned when cross-site mutation is detected.
	ErrCSRFPreflightFailed = errors.New("workspace_path: cross-site GET cannot mutate session")

	// ErrRateLimited is returned when rotation rate limit is exceeded.
	ErrRateLimited = errors.New("workspace_path: rotation rate limit exceeded")
)

// --- Regex + format constants ---

var slugFormat = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// pathPattern matches /w/{slug}[/as/{client_id}][/rest...].
var pathPattern = regexp.MustCompile(`^/w/([a-z0-9-]+)(?:/as/([a-zA-Z0-9_-]+))?(/.*)?$`)

const (
	defaultSlugCacheTTL      = 5 * time.Minute
	defaultRotationRateLimit = 10
	defaultRateLimitWindow   = time.Minute
	rateLimiterGCInterval    = 5 * time.Minute
	pickerRedirectURL        = "/auth/select-workspace-role"
	loginRedirectURL         = "/auth/login"
)

// --- Context keys ---

type wsCtxKey string

const (
	// CtxKeyURLWorkspaceID carries the URL-canonical workspace_id.
	CtxKeyURLWorkspaceID wsCtxKey = "workspace_path.url_workspace_id"

	// CtxKeyURLWorkspaceSlug carries the original slug from the URL.
	CtxKeyURLWorkspaceSlug wsCtxKey = "workspace_path.url_workspace_slug"

	// CtxKeyActingAsClientID carries the path-param acting_as_client_id.
	CtxKeyActingAsClientID wsCtxKey = "workspace_path.acting_as_client_id"
)

// --- Post-rotation banner ---
//
// The post-rotation banner ctx-key is owned by pyeza render
// (render.WithPostRotationBanner / render.PostRotationBannerFromContext /
// render.PostRotationBannerData), the canonical reader in the render pipeline.
// The espyna→pyeza import arrow makes render the single source of this key:
// this writer (WorkspacePath, on a URL-driven rotation) writes render's key;
// the render pipeline reads it. Do NOT re-declare a private banner key here —
// that fork is exactly the silent ctx-key drift the leaf convergence (W3)
// eliminated (and which had left the banner inert before W3).

// --- Public contract ---

// WorkspaceSwitchResult is the outcome of a URL-driven principal switch.
type WorkspaceSwitchResult struct {
	NewToken    string
	RedirectURL string
}

// BindingResolverFunc resolves the user's binding in a specific workspace.
// Returns (nil, ErrNoBindingInWorkspace) when no active binding exists.
// Returns (nil, ErrAmbiguousBinding) when multiple bindings exist without
// a disambiguating hint.
type BindingResolverFunc func(
	ctx context.Context,
	userID, workspaceID string,
	sessionPrincipalKind PrincipalType,
	sessionPrincipalID string,
) (*Principal, error)

// PrincipalLookupFunc reads the session's current principal kind and id.
type PrincipalLookupFunc func(r *http.Request) (PrincipalType, string)

// ExecuteSwitchFunc performs the atomic session update for a URL-driven
// workspace navigation.
type ExecuteSwitchFunc func(
	ctx context.Context,
	userID, token string,
	target *Principal,
	urlActingAs string,
	requestURL, referer, secFetchSite, userAgent string,
) (*WorkspaceSwitchResult, error)

// SessionLookupFunc reads (userID, workspaceID, token) from the request.
type SessionLookupFunc func(r *http.Request) (userID, workspaceID, token string, ok bool)

// WorkspacePathConfig configures the WorkspacePathMiddleware.
type WorkspacePathConfig struct {
	// SlugLookup resolves a workspace slug to a workspace_id.
	SlugLookup func(ctx context.Context, slug string) (string, error)

	// SessionLookup reads the current session identity. Required.
	SessionLookup SessionLookupFunc

	// BindingResolver validates the user's binding in the URL workspace. Required.
	BindingResolver BindingResolverFunc

	// PrincipalLookup reads the session's current principal kind + id. Optional.
	PrincipalLookup PrincipalLookupFunc

	// ExecuteSwitch performs atomic session rotation. Required.
	ExecuteSwitch ExecuteSwitchFunc

	// SetSessionCookie writes the rotated session cookie. When nil the
	// middleware uses writeStrictSessionCookie.
	SetSessionCookie func(w http.ResponseWriter, token string)

	// SetCSRFCookie issues a fresh workspace-claim CSRF cookie alongside
	// the rotated session cookie. When nil no CSRF cookie is issued.
	SetCSRFCookie func(w http.ResponseWriter, newSessionToken, newWorkspaceID string)

	// CookieName is the session cookie name. Default
	// consumer.DefaultSessionCookieName.
	CookieName string

	// IsReservedSlug reports whether a slug is reserved.
	IsReservedSlug func(slug string) bool

	// WithWorkspaceID stores the workspace_id in the context for downstream
	// consumers. Typically wired to consumer.WithWorkspaceID. When nil, the
	// middleware uses context.WithValue with CtxKeyURLWorkspaceID only.
	WithWorkspaceID func(ctx context.Context, workspaceID string) context.Context

	// AppOrigin is the canonical origin (scheme://host[:port]) used as
	// the Referer-fallback comparator when Sec-Fetch-* headers are absent.
	AppOrigin string

	// SlugCacheTTL is how long a slug->workspace_id mapping is cached.
	// Default 5 minutes.
	SlugCacheTTL time.Duration

	// RotationRateLimitPerMin is the maximum URL-driven rotations per
	// user per minute. Default 10.
	RotationRateLimitPerMin int
}

// WorkspacePathMiddleware parses /w/{slug}/* paths, validates workspace
// binding, and optionally rotates sessions on cross-workspace navigation.
type WorkspacePathMiddleware struct {
	cfg     WorkspacePathConfig
	cache   *wsSlugCache
	limiter *wsRotationRateLimiter
}

// NewWorkspacePathMiddleware constructs the middleware from config.
// Panics if SessionLookup, BindingResolver, or ExecuteSwitch are nil.
func NewWorkspacePathMiddleware(cfg WorkspacePathConfig) func(next http.Handler) http.Handler {
	if cfg.SessionLookup == nil {
		panic("workspace_path middleware: SessionLookup is required")
	}
	if cfg.BindingResolver == nil {
		panic("workspace_path middleware: BindingResolver is required")
	}
	if cfg.ExecuteSwitch == nil {
		panic("workspace_path middleware: ExecuteSwitch is required")
	}
	if cfg.SlugCacheTTL <= 0 {
		cfg.SlugCacheTTL = defaultSlugCacheTTL
	}
	if cfg.RotationRateLimitPerMin <= 0 {
		cfg.RotationRateLimitPerMin = defaultRotationRateLimit
	}
	if cfg.CookieName == "" {
		cfg.CookieName = defaultSessionCookieName
	}
	if cfg.IsReservedSlug == nil {
		cfg.IsReservedSlug = func(string) bool { return false }
	}
	if cfg.SlugLookup == nil {
		cfg.SlugLookup = func(ctx context.Context, slug string) (string, error) {
			return "", nil
		}
	}

	m := &WorkspacePathMiddleware{
		cfg:     cfg,
		cache:   newWSSlugCache(cfg.SlugCacheTTL),
		limiter: newWSRotationRateLimiter(cfg.RotationRateLimitPerMin, defaultRateLimitWindow),
	}
	return m.handle
}

func (m *WorkspacePathMiddleware) handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 0. Fast path -- non-/w/ requests bypass entirely.
		if !strings.HasPrefix(r.URL.Path, "/w/") {
			next.ServeHTTP(w, r)
			return
		}

		// 1. Parse the path.
		match := pathPattern.FindStringSubmatch(r.URL.Path)
		if match == nil {
			wsUnifiedNotFoundOrUnauthorized(w, r)
			return
		}
		slug := match[1]
		actingAsClientID := match[2]
		rest := match[3]
		if rest == "" {
			rest = "/"
		}

		// 2. Slug format + reserved-word validation.
		slugLower := strings.ToLower(slug)
		if !isValidSlugFormat(slugLower) || m.cfg.IsReservedSlug(slugLower) {
			wsUnifiedNotFoundOrUnauthorized(w, r)
			return
		}

		// 3. CSRF preflight.
		if !wsIsPermittedRequest(r, m.cfg.AppOrigin) {
			http.Error(w, "Forbidden: cross-site mutation blocked", http.StatusForbidden)
			return
		}

		// 4. Resolve slug -> workspace_id.
		workspaceID, ok := m.cache.get(slugLower)
		if !ok {
			resolved, err := m.cfg.SlugLookup(r.Context(), slugLower)
			if err != nil {
				log.Printf("[workspace_path] slug lookup error slug=%s: %v", slugLower, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if resolved == "" {
				wsUnifiedNotFoundOrUnauthorized(w, r)
				return
			}
			workspaceID = resolved
			m.cache.set(slugLower, workspaceID)
		}

		// 5. Session lookup.
		userID, sessionWsID, token, sessionOK := m.cfg.SessionLookup(r)
		if !sessionOK || userID == "" {
			wsRedirectTo(w, r, loginRedirectURL)
			return
		}

		// 5b. Session principal hint.
		var (
			sessionPrincipalKind PrincipalType
			sessionPrincipalID   string
		)
		if m.cfg.PrincipalLookup != nil {
			sessionPrincipalKind, sessionPrincipalID = m.cfg.PrincipalLookup(r)
		}

		// 6. Active-binding validation with per-request memoization.
		ctx := r.Context()
		binding, berr := wsResolveBindingMemo(
			ctx, userID, workspaceID,
			sessionPrincipalKind, sessionPrincipalID,
			m.cfg.BindingResolver,
		)
		if errors.Is(berr, ErrAmbiguousBinding) {
			wsRedirectTo(w, r, pickerRedirectURL)
			return
		}
		if errors.Is(berr, ErrNoBindingInWorkspace) || binding == nil {
			wsUnifiedNotFoundOrUnauthorized(w, r)
			return
		}
		if berr != nil {
			log.Printf("[workspace_path] binding resolve error user=%s ws=%s: %v",
				userID, workspaceID, berr)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Pin workspace + binding into context. The slug + acting-as are also
		// written under the canonical framework-agnostic keys (httpctx leaf) so
		// any consumer — incl. the app's workspace route rewriter — reads ONE key
		// regardless of server provider. The provider-local CtxKey* values are
		// retained for in-package readers / backward compat.
		ctx = context.WithValue(ctx, CtxKeyURLWorkspaceID, workspaceID)
		ctx = context.WithValue(ctx, CtxKeyURLWorkspaceSlug, slugLower)
		ctx = httpctx.WithURLWorkspaceSlug(ctx, slugLower)
		if actingAsClientID != "" {
			ctx = context.WithValue(ctx, CtxKeyActingAsClientID, actingAsClientID)
			ctx = httpctx.WithActingAsClientID(ctx, actingAsClientID)
		}
		ctx = wsWithBindingMemo(ctx, userID, workspaceID, binding)
		// Override the espyna-owned workspace_id key so the legacy
		// session-injected workspace_id doesn't dominate after rotation.
		if m.cfg.WithWorkspaceID != nil {
			ctx = m.cfg.WithWorkspaceID(ctx, workspaceID)
		}

		// 7. Acting-as plausibility check.
		if actingAsClientID != "" && !wsCanActAs(binding) {
			wsRedirectTo(w, r, pickerRedirectURL)
			return
		}

		// 8. URL-driven rotation.
		rotationNeeded := sessionWsID != workspaceID
		if actingAsClientID != "" {
			rotationNeeded = true
		}

		if rotationNeeded {
			// 8a. Per-user rate limit.
			if !m.limiter.allow(userID) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			// 8b. Build target with URL-derived acting_as.
			target := *binding
			if actingAsClientID != "" {
				target.ActingAsTargets = []ActingAsTarget{{
					ID:          actingAsClientID,
					WorkspaceID: workspaceID,
				}}
			}

			// 8c. Invoke the rotation primitive.
			result, err := m.cfg.ExecuteSwitch(
				ctx,
				userID,
				token,
				&target,
				actingAsClientID,
				r.URL.Path,
				r.Header.Get("Referer"),
				r.Header.Get("Sec-Fetch-Site"),
				r.Header.Get("User-Agent"),
			)
			if err != nil {
				log.Printf("[workspace_path] rotation failed user=%s url_ws=%s: %v",
					userID, workspaceID, err)
				wsRedirectTo(w, r, pickerRedirectURL)
				return
			}

			// 8d. Cookie write.
			if result != nil && result.NewToken != "" {
				if m.cfg.SetSessionCookie != nil {
					m.cfg.SetSessionCookie(w, result.NewToken)
				} else {
					writeStrictSessionCookie(w, m.cfg.CookieName, result.NewToken)
				}
				if m.cfg.SetCSRFCookie != nil {
					m.cfg.SetCSRFCookie(w, result.NewToken, workspaceID)
				}

				// 8e. Banner data. Written under pyeza render's canonical key so
				// the render pipeline (the reader) round-trips it.
				prevSlug, _ := m.cache.getSlugByID(sessionWsID)
				ctx = render.WithPostRotationBanner(ctx, &render.PostRotationBannerData{
					TargetSlug:   slugLower,
					PreviousSlug: prevSlug,
				})
			}
		}

		// 9. Strip prefix and dispatch.
		r2 := r.Clone(ctx)
		r2.URL.Path = rest
		r2.URL.RawPath = ""

		next.ServeHTTP(w, r2)
	})
}

// --- CSRF preflight ---

func wsIsPermittedRequest(r *http.Request, appOrigin string) bool {
	site := r.Header.Get("Sec-Fetch-Site")
	mode := r.Header.Get("Sec-Fetch-Mode")

	switch site {
	case "same-origin":
		return true
	case "none":
		if mode == "navigate" || mode == "" {
			return true
		}
		return false
	case "cross-site", "same-site":
		return false
	case "":
		// Old browser path -- fall through to Referer check.
	default:
		return false
	}

	referer := r.Header.Get("Referer")
	origin := r.Header.Get("Origin")

	if appOrigin == "" {
		return false
	}

	if origin != "" {
		return wsOriginMatches(origin, appOrigin)
	}
	if referer == "" {
		return true
	}
	return wsOriginMatches(referer, appOrigin)
}

func wsOriginMatches(headerValue, appOrigin string) bool {
	hv, err := url.Parse(headerValue)
	if err != nil || hv.Host == "" {
		return false
	}
	ao, err := url.Parse(appOrigin)
	if err != nil || ao.Host == "" {
		return false
	}
	return strings.EqualFold(hv.Host, ao.Host) && strings.EqualFold(hv.Scheme, ao.Scheme)
}

// --- Unified responses ---

func wsUnifiedNotFoundOrUnauthorized(w http.ResponseWriter, r *http.Request) {
	wsRedirectTo(w, r, pickerRedirectURL)
}

func wsRedirectTo(w http.ResponseWriter, r *http.Request, target string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// --- Slug LRU cache ---

type wsSlugCache struct {
	mu  sync.RWMutex
	ttl time.Duration
	m   map[string]wsSlugCacheEntry
}

type wsSlugCacheEntry struct {
	workspaceID string
	expiresAt   time.Time
}

func newWSSlugCache(ttl time.Duration) *wsSlugCache {
	return &wsSlugCache{
		ttl: ttl,
		m:   make(map[string]wsSlugCacheEntry, 16),
	}
}

func (c *wsSlugCache) get(slug string) (string, bool) {
	c.mu.RLock()
	entry, ok := c.m[slug]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.workspaceID, true
}

func (c *wsSlugCache) set(slug, workspaceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[slug] = wsSlugCacheEntry{
		workspaceID: workspaceID,
		expiresAt:   time.Now().Add(c.ttl),
	}
}

func (c *wsSlugCache) getSlugByID(workspaceID string) (string, bool) {
	if workspaceID == "" {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := time.Now()
	for slug, entry := range c.m {
		if entry.workspaceID == workspaceID && !now.After(entry.expiresAt) {
			return slug, true
		}
	}
	return "", false
}

// --- Rotation rate limiter ---

type wsRotationRateLimiter struct {
	mu          sync.Mutex
	budget      int
	window      time.Duration
	state       map[string]*wsRotationBucket
	lastSweep   time.Time
	sweepWindow time.Duration
}

type wsRotationBucket struct {
	count       int
	windowStart time.Time
}

func newWSRotationRateLimiter(budget int, window time.Duration) *wsRotationRateLimiter {
	return &wsRotationRateLimiter{
		budget:      budget,
		window:      window,
		state:       make(map[string]*wsRotationBucket, 64),
		lastSweep:   time.Now(),
		sweepWindow: rateLimiterGCInterval,
	}
}

func (l *wsRotationRateLimiter) allow(userID string) bool {
	if userID == "" {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.lastSweep) > l.sweepWindow {
		l.sweepLocked(now)
	}

	bucket, ok := l.state[userID]
	if !ok || now.Sub(bucket.windowStart) > l.window {
		l.state[userID] = &wsRotationBucket{count: 1, windowStart: now}
		return true
	}
	if bucket.count >= l.budget {
		return false
	}
	bucket.count++
	return true
}

func (l *wsRotationRateLimiter) sweepLocked(now time.Time) {
	for k, b := range l.state {
		if now.Sub(b.windowStart) > l.window {
			delete(l.state, k)
		}
	}
	l.lastSweep = now
}

// --- Binding memoization ---

type wsBindingMemoMap map[wsBindingMemoKey]*Principal

type wsBindingMemoKey struct {
	UserID      string
	WorkspaceID string
}

type wsBindingMemoCtxKey struct{}

func wsResolveBindingMemo(
	ctx context.Context,
	userID, workspaceID string,
	sessionPrincipalKind PrincipalType,
	sessionPrincipalID string,
	resolver BindingResolverFunc,
) (*Principal, error) {
	if memo, ok := ctx.Value(wsBindingMemoCtxKey{}).(wsBindingMemoMap); ok {
		if cached, hit := memo[wsBindingMemoKey{UserID: userID, WorkspaceID: workspaceID}]; hit {
			if cached == nil {
				return nil, ErrNoBindingInWorkspace
			}
			return cached, nil
		}
	}
	if resolver == nil {
		return nil, ErrNoBindingInWorkspace
	}
	return resolver(ctx, userID, workspaceID, sessionPrincipalKind, sessionPrincipalID)
}

func wsWithBindingMemo(
	ctx context.Context,
	userID, workspaceID string,
	binding *Principal,
) context.Context {
	memo, ok := ctx.Value(wsBindingMemoCtxKey{}).(wsBindingMemoMap)
	if !ok {
		memo = make(wsBindingMemoMap, 4)
		ctx = context.WithValue(ctx, wsBindingMemoCtxKey{}, memo)
	}
	memo[wsBindingMemoKey{UserID: userID, WorkspaceID: workspaceID}] = binding
	return ctx
}

// --- Helpers ---

func isValidSlugFormat(slug string) bool {
	if len(slug) < 3 || len(slug) > 30 {
		return false
	}
	return slugFormat.MatchString(slug)
}

func wsCanActAs(p *Principal) bool {
	if p == nil {
		return false
	}
	switch p.Type {
	case PrincipalTypeClientDelegate,
		PrincipalTypeSupplierDelegate:
		return true
	}
	return false
}

// writeStrictSessionCookie writes the rotated session cookie with
// SameSite=Strict. URL-driven rotation upgrades SameSite from the espyna
// primitive's default Lax to Strict.
func writeStrictSessionCookie(w http.ResponseWriter, cookieName, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

// GetURLWorkspaceIDFromContext reads the URL-canonical workspace_id set by
// WorkspacePathMiddleware.
func GetURLWorkspaceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyURLWorkspaceID).(string)
	return v
}

// GetURLWorkspaceSlugFromContext reads the URL slug set by
// WorkspacePathMiddleware.
func GetURLWorkspaceSlugFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyURLWorkspaceSlug).(string)
	return v
}

// GetActingAsClientIDFromContext reads the /as/{client_id} parameter.
func GetActingAsClientIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyActingAsClientID).(string)
	return v
}
