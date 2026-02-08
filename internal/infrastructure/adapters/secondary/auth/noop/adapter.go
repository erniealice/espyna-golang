//go:build noop || mock_auth || mock_db

package noop

import (
	"context"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterAuthProvider(
		"noop",
		func() ports.AuthProvider {
			return NewAdapter()
		},
		transformConfig,
	)
	registry.RegisterAuthBuildFromEnv("noop", buildFromEnv)
}

// buildFromEnv creates and initializes a No-Op auth provider.
func buildFromEnv() (ports.AuthProvider, error) {
	protoConfig := &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		DisplayName: "NoOp",
		Config: &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName: "noop",
			},
		},
	}
	p := NewAdapter()
	if err := p.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("noop_auth: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig converts raw config map to no-op auth proto config.
func transformConfig(rawConfig map[string]any) (*authpb.ProviderConfig, error) {
	return &authpb.ProviderConfig{
		Enabled:     true,
		Provider:    authpb.Provider_PROVIDER_CUSTOM,
		DisplayName: "NoOp",
		Config: &authpb.ProviderConfig_CustomConfig{
			CustomConfig: &authpb.CustomProviderConfig{
				ProviderName: "noop",
			},
		},
	}, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// NoOpAuthAdapter implements ports.AuthProvider and ports.AuthService
// This adapter provides disabled authentication (no-op)
// Following the same pattern as other auth adapters for consistency
type NoOpAuthAdapter struct {
	config  *authpb.ProviderConfig
	enabled bool
}

// NewAdapter creates a new no-op auth adapter
func NewAdapter() ports.AuthProvider {
	return &NoOpAuthAdapter{
		enabled: false,
	}
}

// Name returns the provider name
func (p *NoOpAuthAdapter) Name() string {
	return "noop"
}

// Initialize sets up no-op auth with proto-based configuration
func (p *NoOpAuthAdapter) Initialize(config *authpb.ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	p.config = config
	p.enabled = config.Enabled

	if config.Enabled {
		log.Println("[AUTH] No-Op Auth provider initialized (authentication disabled)")
	}

	return nil
}

// GetAuthService returns the authentication service (returns itself)
func (p *NoOpAuthAdapter) GetAuthService() ports.AuthService {
	if !p.enabled {
		return nil
	}
	return p
}

// IsHealthy always returns nil for no-op provider
func (p *NoOpAuthAdapter) IsHealthy(ctx context.Context) error {
	return nil
}

// Close cleans up no-op auth resources (none)
func (p *NoOpAuthAdapter) Close() error {
	if p.enabled {
		log.Println("[AUTH] Closing No-Op Auth provider")
		p.enabled = false
	}
	return nil
}

// IsEnabled returns whether no-op auth is enabled
func (p *NoOpAuthAdapter) IsEnabled() bool {
	return p.enabled
}

// VerifyToken always returns an unauthorized error using proto types
func (p *NoOpAuthAdapter) VerifyToken(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error) {
	return &authpb.ValidateJwtTokenResponse{
		IsValid:      false,
		ErrorMessage: "Authentication is disabled",
		ValidationErrors: []*authpb.ValidationError{
			{
				Type:    authpb.ValidationErrorType_VALIDATION_ERROR_TYPE_UNSPECIFIED,
				Message: "No-op authentication provider",
			},
		},
	}, nil
}

// GetProviderName implements the AuthService interface
func (p *NoOpAuthAdapter) GetProviderName() string {
	return "noop"
}

// Compile-time checks that NoOpAuthAdapter implements both interfaces
var _ ports.AuthProvider = (*NoOpAuthAdapter)(nil)
var _ ports.AuthService = (*NoOpAuthAdapter)(nil)
