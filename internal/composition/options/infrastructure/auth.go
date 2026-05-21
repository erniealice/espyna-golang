package infrastructure

import (
	"fmt"
)

// AuthConfig is a union type that can hold any auth configuration.
//
// Per 20260521-composition-reshape Q-CR4 + Q-CR6 LOCK: the With*Auth
// option-setter functions (WithFirebaseAuth, WithJWTAuth, WithAuthFromEnv,
// WithMockAuth) were deleted — espyna-golang is internal-only, so no
// deprecation path was required. AuthConfig + AuthConfigSetter remain
// because ManagerConfig.Auth still references them; deferred to a future
// ManagerConfig refactor.
type AuthConfig struct {
	Firebase *FirebaseAuthConfig
	JWT      *JWTAuthConfig
	Mock     bool
}

// FirebaseAuthConfig defines configuration for Firebase Authentication
type FirebaseAuthConfig struct {
	ProjectID       string `json:"project_id"`
	CredentialsPath string `json:"credentials_path,omitempty"`
	TenantID        string `json:"tenant_id,omitempty"`
}

// Validate validates the firebase auth configuration
func (c FirebaseAuthConfig) Validate() error {
	if c.ProjectID == "" {
		return fmt.Errorf("firebase auth project ID is required")
	}
	return nil
}

// JWTAuthConfig defines configuration for JWT authentication
type JWTAuthConfig struct {
	SecretKey     string `json:"secret_key"`
	TokenExpiry   string `json:"token_expiry"`
	RefreshExpiry string `json:"refresh_expiry"`
	Issuer        string `json:"issuer"`
}

// Validate validates the JWT auth configuration
func (c JWTAuthConfig) Validate() error {
	if c.SecretKey == "" {
		return fmt.Errorf("JWT secret key is required")
	}
	if c.TokenExpiry == "" {
		c.TokenExpiry = "24h"
	}
	if c.RefreshExpiry == "" {
		c.RefreshExpiry = "168h"
	}
	if c.Issuer == "" {
		c.Issuer = "espyna"
	}
	return nil
}
