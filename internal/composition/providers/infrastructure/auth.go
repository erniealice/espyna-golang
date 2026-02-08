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
//   - "firebase_auth" or "firebase" → Firebase Auth provider
//   - "jwt_auth" or "jwt" → JWT Auth provider
//   - "mock_auth", "mock", "noop", or "" → Mock Auth provider (default)
func CreateAuthProvider() (contracts.Provider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_AUTH_PROVIDER"))

	// Normalize provider names
	switch providerName {
	case "firebase_auth":
		providerName = "firebase"
	case "jwt_auth":
		providerName = "jwt"
	case "mock_auth", "noop", "":
		providerName = "mock"
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildAuthProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth provider '%s': %w", providerName, err)
	}

	// Wrap to satisfy composition contracts
	return NewProviderWrapper(providerInstance, contracts.ProviderTypeAuth), nil
}
