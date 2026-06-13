package infrastructure

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// IDProviderAdapter wraps an IDGenerator to implement the contracts.Provider interface
type IDProviderAdapter struct {
	idService ports.IDGenerator
	name      string
}

// NewIDProviderAdapter creates a new IDProviderAdapter
func NewIDProviderAdapter(service ports.IDGenerator, name string) *IDProviderAdapter {
	return &IDProviderAdapter{
		idService: service,
		name:      name,
	}
}

// Type returns the provider type
func (p *IDProviderAdapter) Type() contracts.ProviderType {
	return contracts.ProviderTypeID
}

// Name returns the provider name
func (p *IDProviderAdapter) Name() string {
	return p.name
}

// Initialize initializes the provider
func (p *IDProviderAdapter) Initialize(config interface{}) error {
	return nil
}

// Health checks provider health
func (p *IDProviderAdapter) Health(ctx context.Context) error {
	if p.idService == nil {
		return fmt.Errorf("ID service not initialized")
	}
	return nil
}

// Close closes the provider and releases resources
func (p *IDProviderAdapter) Close() error {
	return nil
}

// GetIDService returns the underlying ID service
func (p *IDProviderAdapter) GetIDService() ports.IDGenerator {
	return p.idService
}

// CreateIDProvider creates an ID provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_ID_PROVIDER environment variable to select which provider to use:
//   - "google_uuidv7" → Google UUID v7 provider (requires google_uuidv7 build tag)
//   - "noop" → NoOp provider (timestamp-based, for dev/test)
//
// Retired aliases (startup error): "uuidv7", "mock", "".
// A missing or unknown provider fails at startup — no silent fallback.
func CreateIDProvider() (contracts.Provider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_ID_PROVIDER"))

	// Reject retired aliases with a clear error message
	switch providerName {
	case "uuidv7":
		return nil, fmt.Errorf("CONFIG_ID_PROVIDER=%q is a retired alias — use \"google_uuidv7\" instead", providerName)
	case "mock":
		return nil, fmt.Errorf("CONFIG_ID_PROVIDER=%q is a retired alias — use \"noop\" instead", providerName)
	case "":
		return nil, fmt.Errorf("CONFIG_ID_PROVIDER is empty — set it explicitly to \"google_uuidv7\" or \"noop\"")
	}

	// Let the provider build and configure itself from environment
	idService, err := registry.BuildIDProviderFromEnv(providerName)
	if err != nil {
		return nil, fmt.Errorf("ID provider %q not registered (is the build tag present?): %w", providerName, err)
	}

	fmt.Printf("🆔 Created ID provider (config=%s): %s\n", providerName, idService.GetProviderInfo())

	// Wrap to satisfy composition contracts
	return NewIDProviderAdapter(idService, providerName), nil
}
