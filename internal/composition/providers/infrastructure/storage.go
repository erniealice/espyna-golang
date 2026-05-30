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
// Uses CONFIG_STORAGE_PROVIDER environment variable to select which provider to use.
// The accepted (authoritative) values match the registered factory names:
//   - "gcs"           → Google Cloud Storage provider (build tag: google && gcp_storage, env: GOOGLE_CLOUD_*)
//   - "s3"            → AWS S3 provider (build tag: aws && s3)
//   - "azure"         → Azure Blob Storage provider (build tag: azure && azure_blob)
//   - "local_storage" → Local filesystem storage provider (build tag: local_storage)
//   - "mock_storage", "mock", or "" → Mock storage provider (default)
//
// Legacy literals are normalized to the authoritative name so existing dev/test
// configs keep booting after the registry-key rename (Q-ST-C2):
//   - "local"       → "local_storage"
//   - "gcp_storage" → "gcs"
func CreateStorageProvider() (contracts.Provider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_STORAGE_PROVIDER"))

	// Normalize provider names: defaults + legacy-literal aliases.
	switch providerName {
	case "mock", "":
		providerName = "mock_storage"
	case "local":
		// Legacy dev literal — registry now keys this provider as "local_storage".
		providerName = "local_storage"
	case "gcp_storage":
		// Legacy literal — registry now keys this provider as "gcs".
		providerName = "gcs"
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildStorageProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provider '%s': %w", providerName, err)
	}

	// Wrap to satisfy composition contracts
	return NewProviderWrapper(providerInstance, contracts.ProviderTypeStorage), nil
}
