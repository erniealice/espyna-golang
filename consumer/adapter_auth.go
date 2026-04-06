package consumer

import (
	"context"
	"fmt"

	authpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/auth"
)

// authProviderOperations defines the operations interface for auth without conflicting Initialize
type authProviderOperations interface {
	Name() string
	GetAuthService() authServiceOperations
	IsHealthy(ctx context.Context) error
	Close() error
	IsEnabled() bool
}

// databaseAuthOperations defines the extended operations available with db_auth provider.
type databaseAuthOperations interface {
	Register(ctx context.Context, email, password, firstName, lastName, mobileNumber string) (string, error)
	Login(ctx context.Context, email, password string) (string, *authpb.Identity, error)
	RequestPasswordReset(ctx context.Context, email string) (string, error)
	ExecutePasswordReset(ctx context.Context, token, newPassword string) error
	CreateSession(ctx context.Context, userID string) (string, error)
	ValidateSession(ctx context.Context, token string) (string, error)
	InvalidateSession(ctx context.Context, token string) error
	GetSessionWorkspaceContext(ctx context.Context, token string) (wsUserID, wsID string)
}

// authServiceOperations defines the auth service operations
type authServiceOperations interface {
	VerifyToken(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error)
	IsEnabled() bool
	GetProviderName() string
}

/*
 ESPYNA CONSUMER APP - Technology-Agnostic Auth Adapter

Provides direct access to authentication operations without requiring
the full use cases/provider initialization chain.

This adapter works with ANY auth provider (Firebase, JWT, Mock)
based on your CONFIG_AUTH_PROVIDER environment variable.

Usage:

	// Option 1: Get from container (recommended)
	container := consumer.NewContainerFromEnv()
	adapter := consumer.NewAuthAdapterFromContainer(container)

	// Verify JWT token
	result, err := adapter.VerifyToken(ctx, "Bearer eyJ...")

	// Check if auth is enabled
	if adapter.IsEnabled() {
	    // Auth is available
	}
*/

// AuthAdapter provides technology-agnostic access to authentication services.
// It wraps the AuthProvider interface and works with Firebase, JWT, Mock, etc.
type AuthAdapter struct {
	provider  authProviderOperations
	service   authServiceOperations
	container *Container
}

// NewAuthAdapterFromContainer creates an AuthAdapter from an existing container.
// This is the recommended way to create the adapter as it reuses the container's provider.
func NewAuthAdapterFromContainer(container *Container) *AuthAdapter {
	if container == nil {
		return nil
	}

	// Get auth provider from container
	providerContract := container.GetAuthProvider()
	if providerContract == nil {
		return nil
	}

	// Cast to authProviderOperations interface (avoids Initialize method conflict)
	provider, ok := providerContract.(authProviderOperations)
	if !ok {
		return nil
	}

	// Get auth service from provider
	service := provider.GetAuthService()

	return &AuthAdapter{
		provider:  provider,
		service:   service,
		container: container,
	}
}

// Close closes the auth adapter.
// Note: If created from container, this does NOT close the container.
func (a *AuthAdapter) Close() error {
	// Don't close the container here - let the caller manage it
	return nil
}

// GetProvider returns the underlying auth provider for advanced operations.
func (a *AuthAdapter) GetProvider() authProviderOperations {
	return a.provider
}

// GetService returns the underlying auth service for direct access.
func (a *AuthAdapter) GetService() authServiceOperations {
	return a.service
}

// Name returns the name of the underlying auth provider (e.g., "firebase", "jwt", "mock")
func (a *AuthAdapter) Name() string {
	if a.provider == nil {
		return ""
	}
	return a.provider.Name()
}

// IsEnabled returns whether the auth provider is enabled
func (a *AuthAdapter) IsEnabled() bool {
	if a.service == nil {
		return false
	}
	return a.service.IsEnabled()
}

// --- Auth Operations ---

// VerifyToken validates a JWT token and returns the validation result.
// The token should be the full token string (with or without "Bearer " prefix).
func (a *AuthAdapter) VerifyToken(ctx context.Context, token string) (*authpb.ValidateJwtTokenResponse, error) {
	if a.service == nil {
		return nil, fmt.Errorf("auth service not initialized")
	}

	req := &authpb.ValidateJwtTokenRequest{
		Token: token,
	}

	return a.service.VerifyToken(ctx, req)
}

// VerifyTokenProto validates a JWT token using the protobuf request type directly.
// Use this for full control over all validation parameters.
func (a *AuthAdapter) VerifyTokenProto(ctx context.Context, req *authpb.ValidateJwtTokenRequest) (*authpb.ValidateJwtTokenResponse, error) {
	if a.service == nil {
		return nil, fmt.Errorf("auth service not initialized")
	}
	return a.service.VerifyToken(ctx, req)
}

// IsHealthy checks if the auth provider is healthy and available.
func (a *AuthAdapter) IsHealthy(ctx context.Context) error {
	if a.provider == nil {
		return fmt.Errorf("auth provider not initialized")
	}
	return a.provider.IsHealthy(ctx)
}

// GetProviderName returns the name of the auth provider (for logging/debugging).
func (a *AuthAdapter) GetProviderName() string {
	if a.service == nil {
		return ""
	}
	return a.service.GetProviderName()
}

// --- Convenience Methods ---

// ValidateAndExtractUserID validates a token and extracts the user ID if valid.
// Returns the user ID on success, or an error if validation fails.
func (a *AuthAdapter) ValidateAndExtractUserID(ctx context.Context, token string) (string, error) {
	resp, err := a.VerifyToken(ctx, token)
	if err != nil {
		return "", err
	}

	if !resp.IsValid {
		return "", fmt.Errorf("token validation failed: %s", resp.ErrorMessage)
	}

	if resp.Identity == nil {
		return "", fmt.Errorf("no identity in token")
	}

	return resp.Identity.Id, nil
}

// ValidateAndExtractIdentity validates a token and extracts the identity if valid.
// Returns the identity on success, or an error if validation fails.
func (a *AuthAdapter) ValidateAndExtractIdentity(ctx context.Context, token string) (*authpb.Identity, error) {
	resp, err := a.VerifyToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if !resp.IsValid {
		return nil, fmt.Errorf("token validation failed: %s", resp.ErrorMessage)
	}

	return resp.Identity, nil
}

// ValidateAndExtractToken validates a token and extracts the decoded token if valid.
// Returns the decoded token on success, or an error if validation fails.
func (a *AuthAdapter) ValidateAndExtractToken(ctx context.Context, token string) (*authpb.JwtToken, error) {
	resp, err := a.VerifyToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if !resp.IsValid {
		return nil, fmt.Errorf("token validation failed: %s", resp.ErrorMessage)
	}

	return resp.Token, nil
}

// --- Database Auth Methods ---

// Register creates a new user account with the given credentials.
// Only supported by db_auth provider. Returns ErrNotSupported for other providers.
func (a *AuthAdapter) Register(ctx context.Context, email, password, firstName, lastName, mobileNumber string) (string, error) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", fmt.Errorf("register not supported by %s provider", a.Name())
	}
	return dbAuth.Register(ctx, email, password, firstName, lastName, mobileNumber)
}

// Login authenticates a user with email/password and returns a session token + identity.
// Only supported by db_auth provider.
func (a *AuthAdapter) Login(ctx context.Context, email, password string) (string, *authpb.Identity, error) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", nil, fmt.Errorf("login not supported by %s provider", a.Name())
	}
	return dbAuth.Login(ctx, email, password)
}

// RequestPasswordReset generates a reset token for the given email.
// Returns the raw token (caller sends it via email). Only supported by db_auth provider.
func (a *AuthAdapter) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", fmt.Errorf("password reset not supported by %s provider", a.Name())
	}
	return dbAuth.RequestPasswordReset(ctx, email)
}

// ExecutePasswordReset validates a reset token and sets a new password.
// Only supported by db_auth provider.
func (a *AuthAdapter) ExecutePasswordReset(ctx context.Context, token, newPassword string) error {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return fmt.Errorf("password reset not supported by %s provider", a.Name())
	}
	return dbAuth.ExecutePasswordReset(ctx, token, newPassword)
}

// CreateSession creates a new session for the given user.
// Only supported by db_auth provider.
func (a *AuthAdapter) CreateSession(ctx context.Context, userID string) (string, error) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", fmt.Errorf("session management not supported by %s provider", a.Name())
	}
	return dbAuth.CreateSession(ctx, userID)
}

// ValidateSession checks if a session token is valid and returns the user ID.
// Only supported by db_auth provider.
func (a *AuthAdapter) ValidateSession(ctx context.Context, token string) (string, error) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", fmt.Errorf("session management not supported by %s provider", a.Name())
	}
	return dbAuth.ValidateSession(ctx, token)
}

// InvalidateSession marks a session as inactive.
// Only supported by db_auth provider.
func (a *AuthAdapter) InvalidateSession(ctx context.Context, token string) error {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return fmt.Errorf("session management not supported by %s provider", a.Name())
	}
	return dbAuth.InvalidateSession(ctx, token)
}

// GetSessionWorkspaceContext returns the workspace_user_id and workspace_id stored on the session.
// Only supported by db_auth provider. Returns empty strings for other providers.
func (a *AuthAdapter) GetSessionWorkspaceContext(ctx context.Context, token string) (wsUserID, wsID string) {
	dbAuth, ok := a.provider.(databaseAuthOperations)
	if !ok {
		return "", ""
	}
	return dbAuth.GetSessionWorkspaceContext(ctx, token)
}

// --- Re-export error codes for consumer convenience ---

const (
	// AuthErrCodeMissingToken indicates no token was provided
	AuthErrCodeMissingToken = "AUTH_MISSING_TOKEN"
	// AuthErrCodeInvalidToken indicates the token format is invalid
	AuthErrCodeInvalidToken = "AUTH_INVALID_TOKEN"
	// AuthErrCodeExpiredToken indicates the token has expired
	AuthErrCodeExpiredToken = "AUTH_EXPIRED_TOKEN"
	// AuthErrCodeServiceDown indicates the auth service is unavailable
	AuthErrCodeServiceDown = "AUTH_SERVICE_DOWN"
	// AuthErrCodeUnauthorized indicates authorization was denied
	AuthErrCodeUnauthorized = "AUTH_UNAUTHORIZED"
)
