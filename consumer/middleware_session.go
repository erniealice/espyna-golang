package consumer

import (
	"context"
	"net/http"
	"strings"

	sharedidentity "github.com/erniealice/espyna-golang/shared/identity"
)

// SessionContextKey is the context key type for session data.
type SessionContextKey string

const (
	// ContextKeyUserID is the context key for the authenticated user ID.
	ContextKeyUserID SessionContextKey = "uid"

	// ContextKeySessionToken is the context key for the session token.
	ContextKeySessionToken SessionContextKey = "session_token"

	// ContextKeyWorkspaceID is the context key for the workspace ID.
	ContextKeyWorkspaceID SessionContextKey = "workspace_id"

	// ContextKeyWorkspaceUserID is the context key for the workspace user ID.
	ContextKeyWorkspaceUserID SessionContextKey = "workspace_user_id"

	// DefaultSessionCookieName is the default cookie name for session tokens.
	DefaultSessionCookieName = "ichizen_session"
)

// SessionMiddleware validates session cookies on protected routes.
// If no valid session exists, it redirects to the login URL.
type SessionMiddleware struct {
	// AuthAdapter provides session validation via the active auth provider.
	AuthAdapter *AuthAdapter

	// LoginURL is where unauthenticated users are redirected (default: /auth/login).
	LoginURL string

	// ExcludePrefixes are URL path prefixes that skip session validation.
	// Common: "/auth/", "/assets/", "/health"
	ExcludePrefixes []string

	// CookieName is the session cookie name (default: "ichizen_session").
	CookieName string

	// CookieSecure sets the Secure flag on cookies (default: false, set true in production).
	CookieSecure bool

	// CookieMaxAge is the cookie max age in seconds (default: 604800 = 7 days).
	CookieMaxAge int
}

// NewSessionMiddleware creates a SessionMiddleware with sensible defaults.
func NewSessionMiddleware(authAdapter *AuthAdapter) *SessionMiddleware {
	return &SessionMiddleware{
		AuthAdapter: authAdapter,
		LoginURL:    "/auth/login",
		ExcludePrefixes: []string{
			"/auth/",
			"/assets/",
			"/health",
			"/favicon.ico",
		},
		CookieName:   DefaultSessionCookieName,
		CookieSecure: false,
		CookieMaxAge: 7 * 24 * 60 * 60, // 7 days
	}
}

// Handler returns an http.Handler middleware that wraps the given handler.
func (m *SessionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the path is excluded from session validation
		if m.isExcluded(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Read session token from cookie. This is the SINGLE source of binding
		// truth for the request: after a principal switch A→B the cookie holds
		// either the rotated token (workspace change) or the same token whose
		// session row was mutated in place (same workspace, new binding). Either
		// way, everything below is keyed by THIS token, so the request always
		// resolves B's binding — never a cached or carried-over copy of A's.
		token := m.getSessionCookie(r)
		if token == "" {
			m.redirectToLogin(w, r)
			return
		}

		// Resolve the user + workspace binding as one coherent snapshot keyed by
		// the single cookie token (P3-W3). ResolveSessionIdentity validates the
		// session first, then reads the workspace binding from the SAME token, so
		// the userID and the workspace context can never be torn across a
		// concurrent switch. There is no per-request identity cache: every
		// request re-resolves from the live cookie token.
		identity, err := m.AuthAdapter.ResolveSessionIdentity(r.Context(), token)
		if err != nil {
			// Invalid, expired, or rotated-out session — clear cookie and redirect.
			// (A token rotated out by a switch is now active=false and fails the
			// validate step, so a stale token can never resolve a live binding.)
			m.clearSessionCookie(w)
			m.redirectToLogin(w, r)
			return
		}

		// Inject full session identity (user, workspace, email) into request
		// context. An empty WorkspaceID/WorkspaceUserID here is a legitimate
		// pre-selection state for a valid session — preserved exactly, NOT a
		// denial, and NOT a carry-over from any prior binding.
		ctx := WithSessionIdentity(r.Context(), identity.UserID, identity.WorkspaceID, identity.WorkspaceUserID, "")
		ctx = context.WithValue(ctx, ContextKeySessionToken, token)

		// Stamp the session token onto the RequestIdentity struct that
		// WithSessionIdentity just stored. The struct is stored by pointer,
		// so mutating the retrieved pointer updates the context value in place.
		if rid, ok := sharedidentity.FromContext(ctx); ok {
			rid.SessionToken = token
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HandlerFunc is a convenience wrapper that returns http.HandlerFunc.
func (m *SessionMiddleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return m.Handler(next).ServeHTTP
}

// SetSessionCookie sets the session cookie on the response.
// Call this after a successful login to establish the session.
func (m *SessionMiddleware) SetSessionCookie(w http.ResponseWriter, token string) {
	cookieName := m.CookieName
	if cookieName == "" {
		cookieName = DefaultSessionCookieName
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   m.CookieMaxAge,
		HttpOnly: true,
		Secure:   m.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie removes the session cookie from the response.
// Call this after logout.
func (m *SessionMiddleware) ClearSessionCookie(w http.ResponseWriter) {
	m.clearSessionCookie(w)
}

// GetUserIDFromContext extracts the user ID from the request context.
// Returns empty string if not authenticated.
//
// Delegates to ExtractUserIDFromContext, which reads the typed key written by
// WithSessionIdentity (used by both the password-mode SessionMiddleware here
// and the dev MockSessionMiddleware). Reading only the legacy
// SessionContextKey("uid") here would miss the password-mode writer, which
// produced 403 "unauthorized" on POST /action/admin/switch-workspace and
// silently bounced authenticated change-password requests back to login.
func GetUserIDFromContext(ctx context.Context) string {
	return ExtractUserIDFromContext(ctx)
}

// GetSessionTokenFromContext extracts the session token from the request context.
func GetSessionTokenFromContext(ctx context.Context) string {
	if token, ok := ctx.Value(ContextKeySessionToken).(string); ok {
		return token
	}
	return ""
}

// --- Internal helpers ---

func (m *SessionMiddleware) isExcluded(path string) bool {
	for _, prefix := range m.ExcludePrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func (m *SessionMiddleware) getSessionCookie(r *http.Request) string {
	cookieName := m.CookieName
	if cookieName == "" {
		cookieName = DefaultSessionCookieName
	}

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (m *SessionMiddleware) clearSessionCookie(w http.ResponseWriter) {
	cookieName := m.CookieName
	if cookieName == "" {
		cookieName = DefaultSessionCookieName
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *SessionMiddleware) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	loginURL := m.LoginURL
	if loginURL == "" {
		loginURL = "/auth/login"
	}

	// For HTMX requests, use HX-Redirect header instead of 302
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", loginURL)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, loginURL, http.StatusFound)
}
