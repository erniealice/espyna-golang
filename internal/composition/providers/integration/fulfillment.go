package integration

import (
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// CreateFulfillmentProvider creates a fulfillment provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_FULFILLMENT_PROVIDER environment variable to select which provider to use:
//   - "lalamove"      → Lalamove delivery service
//   - "grabexpress"   → GrabExpress delivery service
//   - "mock_fulfillment", "mock", or "" → Mock fulfillment provider (default)
func CreateFulfillmentProvider() (integration.FulfillmentProvider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_FULFILLMENT_PROVIDER"))

	// Normalize provider names
	switch providerName {
	case "mock", "":
		providerName = "mock_fulfillment"
	}

	// Let the provider build and configure itself from environment
	providerInstance, err := registry.BuildFulfillmentProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment provider '%s': %w", providerName, err)
	}

	return providerInstance, nil
}

// CreateFulfillmentProviders creates all fulfillment providers specified in CONFIG_FULFILLMENT_PROVIDER.
// Supports comma-separated values (e.g., "lalamove,grabexpress").
// All providers are active simultaneously — the domain layer picks per-operation.
// Returns a map keyed by provider name.
func CreateFulfillmentProviders() (map[string]integration.FulfillmentProvider, error) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("CONFIG_FULFILLMENT_PROVIDER")))
	if raw == "" || raw == "mock" {
		raw = "mock_fulfillment"
	}

	names := strings.Split(raw, ",")
	providers := make(map[string]integration.FulfillmentProvider)

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Normalize
		if name == "mock" {
			name = "mock_fulfillment"
		}

		provider, err := registry.BuildFulfillmentProviderFromEnv(name)
		if err != nil {
			fmt.Printf("warning: failed to initialize fulfillment provider '%s': %v\n", name, err)
			continue
		}
		if provider != nil {
			providers[name] = provider
		}
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no fulfillment providers could be initialized from CONFIG_FULFILLMENT_PROVIDER=%s", raw)
	}

	return providers, nil
}
