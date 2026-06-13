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
//   - "gcs"             → Google Cloud Storage provider (build tag: gcs)
//   - "aws_storage"     → AWS S3 provider (build tag: aws_storage)
//   - "azure_storage"   → Azure Blob Storage provider (build tag: azure_storage)
//   - "local_storage"   → Local filesystem storage provider (build tag: local_storage)
//   - "mock_storage", "mock", or "" → Mock storage provider (default)
//
// Legacy literals are normalized to the authoritative name so existing dev/test
// configs keep booting after the registry-key rename:
//   - "local"       → "local_storage"
//   - "gcp_storage" → "gcs"
//   - "s3"          → "aws_storage"
//   - "azure", "azure_blob" → "azure_storage"
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
	case "s3":
		// Legacy literal — registry now keys this provider as "aws_storage".
		providerName = "aws_storage"
	case "azure", "azure_blob":
		// Legacy literal — registry now keys this provider as "azure_storage".
		providerName = "azure_storage"
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildStorageProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provider '%s': %w", providerName, err)
	}

	// Wrap to satisfy composition contracts
	return NewProviderWrapper(providerInstance, contracts.ProviderTypeStorage), nil
}
