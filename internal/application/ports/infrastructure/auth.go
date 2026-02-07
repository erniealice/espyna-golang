package infrastructure

import (
	"context"

	authpb "leapfor.xyz/esqyma/golang/v1/infrastructure/auth"
)

// AuthProvider defines the contract for authentication providers
// This interface abstracts authentication services like Firebase Auth, JWT, etc.
//
// Uses proto-generated ProviderConfig for typed configuration, aligning with
// the storage provider pattern for consistency across infrastructure ports.
type AuthProvider interface {
	// Name returns the name of the auth provider (e.g., "firebase", "jwt", "mock")
	Name() string

	// Initialize sets up the auth provider with proto-based configuration
	// Uses authpb.ProviderConfig for type-safe, provider-specific settings
	Initialize(config *authpb.ProviderConfig) error

	// GetAuthService returns the authentication service instance
	GetAuthService() AuthService

	// IsHealthy checks if the auth service is available
	IsHealthy(ctx context.Context) error

	// Close cleans up auth provider resources
	Close() error

	// IsEnabled returns whether this provider is currently enabled
	IsEnabled() bool
}

// AuthService defines the contract for authentication services
// This is a domain-level interface that abstracts away specific auth providers
//
// The interface uses proto-generated types for data (ValidateJwtRequest/Response)
// while defining behavioral contracts for lifecycle management
type AuthService interface {
	// VerifyToken validates a JWT token and returns validation result
	// Uses proto types for request/response to ensure consistency across providers
	VerifyToken(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error)

	// IsEnabled returns whether authentication is enabled
	IsEnabled() bool

	// GetProviderName returns the name of the auth provider (for logging/debugging)
	GetProviderName() string
}

// Error codes for authentication errors (for backward compatibility)
const (
	ErrCodeMissingToken = "AUTH_MISSING_TOKEN"
	ErrCodeInvalidToken = "AUTH_INVALID_TOKEN"
	ErrCodeExpiredToken = "AUTH_EXPIRED_TOKEN"
	ErrCodeServiceDown  = "AUTH_SERVICE_DOWN"
	ErrCodeUnauthorized = "AUTH_UNAUTHORIZED"
)
