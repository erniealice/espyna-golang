package infrastructure

import (
	"context"

	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
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

	// ChangePassword updates the password for an authenticated user.
	// oldPassword must match the stored hash; newPassword is validated and hashed.
	// The caller's current session is preserved — only the password_hash is updated.
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error

	// Admin user-lifecycle effects at the IdP. Each is provider-config-driven:
	// the firebase adapter performs the Admin SDK effect; the password adapter
	// treats the DB (user.active / password_hash / session rows) as authoritative
	// and no-ops where the IdP is not the source of truth; mock no-ops.
	// userID is the DB user id; the firebase adapter resolves DB-uid→firebase by email.

	// DisableUserAtProvider disables the account at the IdP (firebase: UpdateUser{Disabled:true}).
	DisableUserAtProvider(ctx context.Context, userID string) error

	// EnableUserAtProvider re-enables the account at the IdP (firebase: UpdateUser{Disabled:false}).
	EnableUserAtProvider(ctx context.Context, userID string) error

	// UpdateEmailAtProvider syncs the account email at the IdP (firebase: UpdateUser{Email}).
	UpdateEmailAtProvider(ctx context.Context, userID, newEmail string) error

	// AdminSetPassword sets a new password without the old one (admin reset).
	// firebase: UpdateUser{Password}; password: bcrypt + write user.password_hash.
	AdminSetPassword(ctx context.Context, userID, newPassword string) error

	// GeneratePasswordResetLink returns a provider-issued reset link for the user
	// (firebase: PasswordResetLink(email)). password: returns "" + error (use AdminSetPassword).
	GeneratePasswordResetLink(ctx context.Context, userID string) (string, error)

	// RevokeUserTokens revokes the user's outstanding refresh tokens at the IdP
	// (firebase: RevokeRefreshTokens(uid)). password: no-op (the user.active guard
	// and session rows are authoritative).
	RevokeUserTokens(ctx context.Context, userID string) error
}

// Error codes for authentication errors (for backward compatibility)
const (
	ErrCodeMissingToken = "AUTH_MISSING_TOKEN"
	ErrCodeInvalidToken = "AUTH_INVALID_TOKEN"
	ErrCodeExpiredToken = "AUTH_EXPIRED_TOKEN"
	ErrCodeServiceDown  = "AUTH_SERVICE_DOWN"
	ErrCodeUnauthorized = "AUTH_UNAUTHORIZED"
)
