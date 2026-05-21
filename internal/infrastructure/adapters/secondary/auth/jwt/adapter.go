//go:build jwt_auth

package jwt

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
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
func (a *JWTAuthAdapter) VerifyToken(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error) {
	if !a.enabled {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "JWT authentication service is disabled",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
					Message: "service disabled",
				},
			},
		}, nil
	}

	token := ""
	if req != nil {
		token = req.GetToken()
	}
	if token == "" {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "JWT token is missing",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
					Message: "missing token",
				},
			},
		}, nil
	}

	// TODO: Implement actual JWT verification logic here
	// This is just a placeholder to show the structure
	//
	// In a real implementation, you would:
	// 1. Parse the JWT token
	// 2. Validate the signature using jwtSecret
	// 3. Check the issuer matches
	// 4. Verify expiration
	// 5. Extract claims into ValidateJwtTokenResponse.Claims

	if len(token) < 10 {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "JWT token appears to be invalid",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
					Message: "invalid token",
				},
			},
		}, nil
	}

	// Mock validation - replace with actual JWT parsing
	return &authpb.ValidateJwtTokenResponse{
		IsValid: true,
		Identity: &authpb.Identity{
			Id:          "jwt_user_123",
			Email:       "user@example.com",
			DisplayName: "JWT User",
			Provider:    authpb.Provider_PROVIDER_CUSTOM,
			Type:        authpb.IdentityType_IDENTITY_TYPE_USER,
		},
	}, nil
}

// IsEnabled implements the AuthService interface
func (a *JWTAuthAdapter) IsEnabled() bool {
	return a.enabled && a.jwtSecret != ""
}

// GetProviderName implements the AuthService interface
func (a *JWTAuthAdapter) GetProviderName() string {
	return fmt.Sprintf("JWT Auth (issuer: %s)", a.issuer)
}

// ChangePassword is unsupported by the JWT auth provider — JWT is a stateless
// token format and does not own a password store. The password provider owns
// password lifecycle; JWT-only deployments cannot rotate passwords here.
func (a *JWTAuthAdapter) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	return fmt.Errorf("change password not supported by jwt auth provider (use password provider for credential rotation)")
}

// Compile-time check that JWTAuthAdapter implements the AuthService interface.
var _ ports.AuthService = (*JWTAuthAdapter)(nil)
