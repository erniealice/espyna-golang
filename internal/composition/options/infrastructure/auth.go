package infrastructure

import (
	"fmt"
	"strings"
)

// =============================================================================
// AUTH CONFIGURATION TYPES
// =============================================================================

// AuthConfig is a union type that can hold any auth configuration
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

// =============================================================================
// ENVIRONMENT CONFIGURATION LOADERS
// =============================================================================

func createFirebaseAuthConfigFromEnv() FirebaseAuthConfig {
	return FirebaseAuthConfig{
		ProjectID:       GetEnv("FIREBASE_AUTH_PROJECT_ID", GetEnv("FIRESTORE_PROJECT_ID", "")),
		CredentialsPath: GetEnv("FIREBASE_AUTH_CREDENTIALS_PATH", GetEnv("FIRESTORE_CREDENTIALS_PATH", "")),
		TenantID:        GetEnv("FIREBASE_AUTH_TENANT_ID", ""),
	}
}

func createJWTAuthConfigFromEnv() JWTAuthConfig {
	return JWTAuthConfig{
		SecretKey:     GetEnv("JWT_SECRET_KEY", ""),
		TokenExpiry:   GetEnv("JWT_TOKEN_EXPIRY", "24h"),
		RefreshExpiry: GetEnv("JWT_REFRESH_EXPIRY", "168h"),
		Issuer:        GetEnv("JWT_ISSUER", "espyna"),
	}
}

// =============================================================================
// AUTHENTICATION PROVIDER OPTIONS
// =============================================================================

// WithAuthFromEnv dynamically selects auth provider based on CONFIG_AUTH_PROVIDER
func WithAuthFromEnv() ContainerOption {
	return func(c Container) error {
		authProvider := strings.ToLower(GetEnv("CONFIG_AUTH_PROVIDER", "mock_auth"))

		switch authProvider {
		case "firebase_auth":
			return WithFirebaseAuth(createFirebaseAuthConfigFromEnv())(c)
		case "jwt_auth":
			return WithJWTAuth(createJWTAuthConfigFromEnv())(c)
		case "mock_auth", "noop", "":
			return WithMockAuth()(c)
		default:
			return fmt.Errorf("unsupported auth provider: %s", authProvider)
		}
	}
}

// WithFirebaseAuth configures Firebase Authentication
func WithFirebaseAuth(config FirebaseAuthConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid firebase auth configuration: %w", err)
		}

		if setter, ok := c.(AuthConfigSetter); ok {
			setter.SetAuthConfig(AuthConfig{Firebase: &config})
		} else {
			return fmt.Errorf("container does not implement SetAuthConfig method")
		}

		fmt.Printf("🔐 Configured Firebase Auth: %s\n", config.ProjectID)
		return nil
	}
}

// WithJWTAuth configures JWT authentication
func WithJWTAuth(config JWTAuthConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid jwt auth configuration: %w", err)
		}

		if setter, ok := c.(AuthConfigSetter); ok {
			setter.SetAuthConfig(AuthConfig{JWT: &config})
		} else {
			return fmt.Errorf("container does not implement SetAuthConfig method")
		}

		fmt.Printf("🔐 Configured JWT Auth\n")
		return nil
	}
}

// WithMockAuth configures mock authentication for testing
func WithMockAuth() ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(AuthConfigSetter); ok {
			setter.SetAuthConfig(AuthConfig{Mock: true})
		} else {
			return fmt.Errorf("container does not implement SetAuthConfig method")
		}

		fmt.Printf("🧪 Configured mock authentication\n")
		return nil
	}
}
