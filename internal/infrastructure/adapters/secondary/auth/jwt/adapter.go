//go:build jwt_auth

package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
)

// TODO: JWTAuthAdapter does not implement ports.AuthProvider (no Initialize, Name,
// GetAuthService, IsHealthy, Close methods). It only implements ports.AuthService.
// Therefore it cannot be registered with registry.RegisterAuthProvider which
// requires a func() ports.AuthProvider factory. Self-registration is skipped.

// JWTAuthAdapter is an example implementation for JWT-based authentication
// This demonstrates how easy it is to swap authentication providers
type JWTAuthAdapter struct {
	enabled   bool
	jwtSecret string
	issuer    string
}

// NewJWTAuthAdapter creates a new JWT auth adapter
func NewJWTAuthAdapter(jwtSecret, issuer string, enabled bool) ports.AuthService {
	return &JWTAuthAdapter{
		enabled:   enabled,
		jwtSecret: jwtSecret,
		issuer:    issuer,
	}
}

// VerifyToken implements the AuthService interface for JWT tokens
func (a *JWTAuthAdapter) VerifyToken(ctx context.Context, token string) (*ports.AuthToken, error) {
	if !a.enabled {
		return nil, ports.NewAuthError(
			ports.ErrCodeServiceDown,
			"JWT authentication service is disabled",
			nil,
		)
	}

	if token == "" {
		return nil, ports.NewAuthError(
			ports.ErrCodeMissingToken,
			"JWT token is missing",
			nil,
		)
	}

	// TODO: Implement actual JWT verification logic here
	// This is just a placeholder to show the structure

	// For now, we'll just validate that we have a non-empty token
	// In a real implementation, you would:
	// 1. Parse the JWT token
	// 2. Validate the signature using jwtSecret
	// 3. Check the issuer matches
	// 4. Verify expiration
	// 5. Extract claims

	if len(token) < 10 {
		return nil, ports.NewAuthError(
			ports.ErrCodeInvalidToken,
			"JWT token appears to be invalid",
			nil,
		)
	}

	// Mock user data - replace with actual JWT parsing
	authToken := &ports.AuthToken{
		UID: "jwt_user_123",
		Claims: map[string]any{
			"iss":    a.issuer,
			"sub":    "jwt_user_123",
			"email":  "user@example.com",
			"name":   "JWT User",
			"custom": "jwt_specific_claim",
		},
		Expires: time.Now().Add(time.Hour), // 1 hour from now
		Email:   "user@example.com",
		Name:    "JWT User",
	}

	return authToken, nil
}

// IsEnabled implements the AuthService interface
func (a *JWTAuthAdapter) IsEnabled() bool {
	return a.enabled && a.jwtSecret != ""
}

// GetProviderName implements the AuthService interface
func (a *JWTAuthAdapter) GetProviderName() string {
	return fmt.Sprintf("JWT Auth (issuer: %s)", a.issuer)
}
