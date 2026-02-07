//go:build mock_db || mock_auth

package mock

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	authpb "leapfor.xyz/esqyma/golang/v1/infrastructure/auth"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterAuthProvider(
		"mock",
		func() ports.AuthProvider {
			return NewAdapter()
		},
		transformConfig,
	)
	registry.RegisterAuthBuildFromEnv("mock", buildFromEnv)
}

// buildFromEnv creates and initializes a Mock auth provider.
func buildFromEnv() (ports.AuthProvider, error) {
	protoConfig := &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		DisplayName: "Mock",
		Config: &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName: "mock",
			},
		},
	}
	p := NewAdapter()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("mock_auth: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to mock auth proto config.
func transformConfig(rawConfig map[string]any) (*authpb.ProviderConfig, error) {
	return &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		DisplayName: "Mock",
		Config: &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName: "mock",
			},
		},
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// MockAuthAdapter implements ports.AuthProvider and ports.AuthService
// This adapter provides mock authentication for testing and development
// Following the same pattern as FirebaseAuthAdapter for consistency
type MockAuthAdapter struct {
	config  *authpb.ProviderConfig
	enabled bool
}

// NewAdapter creates a new mock auth adapter
func NewAdapter() ports.AuthProvider {
	return &MockAuthAdapter{
		enabled: false,
	}
}

// Name returns the provider name
func (p *MockAuthAdapter) Name() string {
	return "mock"
}

// Initialize sets up mock auth with proto-based configuration
func (p *MockAuthAdapter) Initialize(config *authpb.ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	p.config = config
	p.enabled = config.Enabled

	if config.Enabled {
		log.Println("[OK] Mock Auth provider initialized")
	} else {
		log.Println("[AUTH] Mock Auth is disabled")
	}

	return nil
}

// GetAuthService returns the authentication service (returns itself)
func (p *MockAuthAdapter) GetAuthService() ports.AuthService {
	if !p.enabled {
		return nil
	}
	return p
}

// IsHealthy checks if mock auth is available
func (p *MockAuthAdapter) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("mock auth provider is not enabled")
	}
	return nil
}

// Close cleans up mock auth resources
func (p *MockAuthAdapter) Close() error {
	if p.enabled {
		log.Println("[AUTH] Closing Mock Auth provider")
		p.enabled = false
	}
	return nil
}

// IsEnabled returns whether mock auth is enabled
func (p *MockAuthAdapter) IsEnabled() bool {
	return p.enabled
}

// VerifyToken implements the AuthService interface with mock token verification
func (p *MockAuthAdapter) VerifyToken(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error) {
	if !p.enabled {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Authentication service is disabled",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
					Message: "Service disabled",
				},
			},
		}, nil
	}

	// For mock auth, always succeed when enabled - bypass all auth checks
	log.Println("[AUTH] Mock Auth: enabled - bypassing all auth checks")

	// Mock token validation logic
	if req.Token == "" {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Token is required",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_MALFORMED,
					Message: "Empty token",
				},
			},
		}, nil
	}

	if req.Token == "invalid" {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Invalid token",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_INVALID_SIGNATURE,
					Message: "Signature verification failed",
				},
			},
		}, nil
	}

	if req.Token == "expired" {
		return &authpb.ValidateJwtTokenResponse{
			IsValid:      false,
			ErrorMessage: "Token has expired",
			ValidationErrors: []*authpb.ValidationError{
				{
					Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_EXPIRED,
					Message: "Token expired",
				},
			},
		}, nil
	}

	// Mock successful verification
	identity := &authpb.Identity{
		Id:          "mock-user-123",
		Type:        authpb.IdentityType_IDENTITY_TYPE_USER,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		Email:       "mock@example.com",
		DisplayName: "Mock User",
		IsActive:    true,
	}

	expiresAt := time.Now().Add(time.Hour)
	jwtToken := &authpb.JwtToken{
		Token:     req.Token,
		TokenType: "Bearer",
		ExpiresAt: timestamppb.New(expiresAt),
		IssuedAt:  timestamppb.New(time.Now()),
		Subject:   "mock-user-123",
		Provider:  authpb.Provider_PROVIDER_CUSTOM,
		CustomClaims: map[string]string{
			"role": "user",
			"mock": "true",
		},
	}

	return &authpb.ValidateJwtTokenResponse{
		IsValid:  true,
		Token:    jwtToken,
		Identity: identity,
	}, nil
}

// GetProviderName implements the AuthService interface
func (p *MockAuthAdapter) GetProviderName() string {
	return "mock"
}

// Compile-time checks that MockAuthAdapter implements both interfaces
var _ ports.AuthProvider = (*MockAuthAdapter)(nil)
var _ ports.AuthService = (*MockAuthAdapter)(nil)
