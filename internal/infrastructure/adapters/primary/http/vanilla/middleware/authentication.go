//go:build vanilla

package middleware

import (
	"context"
	"net/http"
	"os"
	"slices"
	"strings"

	"leapfor.xyz/espyna/internal/application/ports"
	authpb "leapfor.xyz/esqyma/golang/v1/infrastructure/auth"
)

// AuthenticationMiddleware provides authentication middleware for vanilla HTTP requests
type AuthenticationMiddleware struct {
	authService ports.AuthService
}

// NewAuthenticationMiddleware creates a new authentication middleware instance
func NewAuthenticationMiddleware(authService ports.AuthService) *AuthenticationMiddleware {
	return &AuthenticationMiddleware{
		authService: authService,
	}
}

// RequireAuth is a middleware that validates authentication tokens
func (m *AuthenticationMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if disabled or service unavailable
		if m.authService == nil || !m.authService.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Check for routes that don't require authentication
		if m.isPublicRoute(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check for API key authentication
		if m.isAuthorizedAPIKey(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header or cookie
		token := m.extractToken(r)
		if token == "" {
			http.Error(w, "Missing or invalid authorization token", http.StatusUnauthorized)
			return
		}

		// Verify the authentication token using proto types
		req := &authpb.ValidateJwtTokenRequest{
			Token:    token,
			Provider: authpb.Provider_PROVIDER_GCP, // Default provider, could be configured
		}

		resp, err := m.authService.VerifyToken(r.Context(), req)
		if err != nil {
			http.Error(w, "Authentication failed", http.StatusInternalServerError)
			return
		}

		if !resp.IsValid {
			http.Error(w, resp.ErrorMessage, http.StatusUnauthorized)
			return
		}

		// Add user information to request context
		ctx := context.WithValue(r.Context(), "uid", resp.Identity.Id)
		ctx = context.WithValue(ctx, "email", resp.Identity.Email)
		ctx = context.WithValue(ctx, "identity", resp.Identity)
		if resp.Token != nil && resp.Token.ExpiresAt != nil {
			ctx = context.WithValue(ctx, "expires", resp.Token.ExpiresAt.AsTime().Unix())
		}

		// Continue with authenticated request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractToken extracts the token from Authorization header or cookie
func (m *AuthenticationMiddleware) extractToken(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}

	// Try token cookie as fallback
	if cookie, err := r.Cookie("token"); err == nil {
		return cookie.Value
	}

	return ""
}

// isPublicRoute checks if the route is public (no auth required)
func (m *AuthenticationMiddleware) isPublicRoute(path string) bool {
	publicRoutes := []string{
		"/health",
		"/api/ping", // Health check endpoints
	}

	return slices.Contains(publicRoutes, path)
}

// isAuthorizedAPIKey checks for valid API keys
func (m *AuthenticationMiddleware) isAuthorizedAPIKey(r *http.Request) bool {
	// Check X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" && apiKey == os.Getenv("X_API_KEY") {
		return true
	}

	// Check X-API-Key-Scheduler header
	schedulerKey := r.Header.Get("X-API-Key-Scheduler")
	if schedulerKey != "" && schedulerKey == os.Getenv("X_API_KEY_SCHEDULER") {
		return true
	}

	return false
}

// GetUserFromContext extracts user information from request context
func GetUserFromContext(ctx context.Context) (uid string, email string, ok bool) {
	uidVal := ctx.Value("uid")
	emailVal := ctx.Value("email")

	if uidVal == nil {
		return "", "", false
	}

	uid, uidOk := uidVal.(string)
	email, emailOk := emailVal.(string)

	return uid, email, uidOk && emailOk
}

// GetIdentityFromContext extracts the full identity from request context
func GetIdentityFromContext(ctx context.Context) (*authpb.Identity, bool) {
	identityVal := ctx.Value("identity")
	if identityVal == nil {
		return nil, false
	}

	identity, ok := identityVal.(*authpb.Identity)
	return identity, ok
}