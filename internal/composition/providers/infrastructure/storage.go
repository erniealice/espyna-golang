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
//   - "mock_storage"    → Mock storage provider
//
// No aliases — the canonical token must be used verbatim. Unknown or empty values
// produce an error directing the user to the canonical token.
func CreateStorageProvider() (contracts.Provider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_STORAGE_PROVIDER"))

	// Reject retired aliases with a clear migration message.
	switch providerName {
	case "mock":
		return nil, fmt.Errorf("storage provider 'mock' is retired - use CONFIG_STORAGE_PROVIDER=mock_storage")
	case "local":
		return nil, fmt.Errorf("storage provider 'local' is retired - use CONFIG_STORAGE_PROVIDER=local_storage")
	case "gcp_storage":
		return nil, fmt.Errorf("storage provider 'gcp_storage' is retired - use CONFIG_STORAGE_PROVIDER=gcs")
	case "s3":
		return nil, fmt.Errorf("storage provider 's3' is retired - use CONFIG_STORAGE_PROVIDER=aws_storage")
	case "azure", "azure_blob":
		return nil, fmt.Errorf("storage provider '%s' is retired - use CONFIG_STORAGE_PROVIDER=azure_storage", providerName)
	case "":
		return nil, fmt.Errorf("CONFIG_STORAGE_PROVIDER is empty - set it explicitly (mock_storage, local_storage, gcs, aws_storage, azure_storage)")
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildStorageProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provider '%s': %w", providerName, err)
	}

	// Wrap to satisfy composition contracts
	return NewProviderWrapper(providerInstance, contracts.ProviderTypeStorage), nil
}
