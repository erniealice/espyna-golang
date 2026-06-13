package integration

import (
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// CreateTabularProvider creates a tabular provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_TABULAR_PROVIDER environment variable to select which provider to use:
//   - "google_sheets" -> Google Sheets provider
//   - "csv" -> CSV file provider
//   - "mock_tabular" -> Mock tabular provider
//
// No aliases — the canonical token must be used verbatim. Unknown or empty values
// produce an error directing the user to the canonical token.
func CreateTabularProvider() (integration.TabularSourceProvider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_TABULAR_PROVIDER"))

	// Reject retired aliases with a clear migration message.
	switch providerName {
	case "googlesheets":
		return nil, fmt.Errorf("tabular provider 'googlesheets' is retired - use CONFIG_TABULAR_PROVIDER=google_sheets")
	case "mock":
		return nil, fmt.Errorf("tabular provider 'mock' is retired - use CONFIG_TABULAR_PROVIDER=mock_tabular")
	case "":
		// Tabular is optional — not configured means skip.
		return nil, nil
	}

	// Check if provider is registered
	if _, exists := registry.GetTabularProviderFactory(providerName); !exists {
		// Try to list available providers for better error message
		available := registry.ListAvailableTabularProviderFactories()
		return nil, fmt.Errorf("tabular provider '%s' not available. Available providers: %v", providerName, available)
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildTabularProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create tabular provider '%s': %w", providerName, err)
	}

	return providerInstance, nil
}
