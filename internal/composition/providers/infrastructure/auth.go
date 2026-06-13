package infrastructure

import (
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// CreateAuthProvider creates an auth provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_AUTH_PROVIDER environment variable to select which provider to use:
//   - "password" → Password/session auth provider
//   - "mock"     → Mock auth provider (dev/test)
//   - "firebase" → Firebase Auth provider
//
// Retired aliases (db_auth, mock_auth, noop, jwt, etc.) return an error
// directing the operator to the canonical token.
func CreateAuthProvider() (contracts.Provider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_AUTH_PROVIDER"))

	// Canonical tokens only — no alias normalization.
	switch providerName {
	case "password", "mock", "firebase":
		// accepted as-is
	case "db_auth":
		// One-release deprecation: map to "password" with a warning.
		fmt.Fprintf(os.Stderr, "DEPRECATED: CONFIG_AUTH_PROVIDER=db_auth is retired — use 'password' instead\n")
		providerName = "password"
	case "password_auth", "firebase_auth", "mock_auth", "noop", "jwt", "jwt_auth":
		return nil, fmt.Errorf("CONFIG_AUTH_PROVIDER=%q is retired; use one of: password, mock, firebase", providerName)
	case "":
		return nil, fmt.Errorf("CONFIG_AUTH_PROVIDER is empty; set one of: password, mock, firebase")
	default:
		// Future providers (oidc, etc.) pass through to the registry.
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildAuthProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth provider '%s': %w", providerName, err)
	}

	// Wrap to satisfy composition contracts
	return NewProviderWrapper(providerInstance, contracts.ProviderTypeAuth), nil
}
