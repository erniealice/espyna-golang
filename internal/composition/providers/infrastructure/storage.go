package infrastructure

import (
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// CreateStorageProvider creates a storage provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_STORAGE_PROVIDER environment variable to select which provider to use:
//   - "gcp_storage" → Google Cloud Storage provider (build tag: google && gcp_storage, env: GOOGLE_CLOUD_*)
//   - "local" → Local filesystem storage provider
//   - "mock_storage", "mock", or "" → Mock storage provider (default)
func CreateStorageProvider() (contracts.Provider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_STORAGE_PROVIDER"))

	// Normalize provider names
	switch providerName {
	case "mock", "":
		providerName = "mock_storage"
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildStorageProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provider '%s': %w", providerName, err)
	}

	// Wrap to satisfy composition contracts
	return NewProviderWrapper(providerInstance, contracts.ProviderTypeStorage), nil
}
