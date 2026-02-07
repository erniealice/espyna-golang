package consumer

import (
	"context"
	"fmt"

	authpb "leapfor.xyz/esqyma/golang/v1/infrastructure/auth"
)

// authProviderOperations defines the operations interface for auth without conflicting Initialize
type authProviderOperations interface {
	Name() string
	GetAuthService() authServiceOperations
	IsHealthy(ctx context.Context) error
	Close() error
	IsEnabled() bool
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
