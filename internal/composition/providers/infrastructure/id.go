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

// IDProviderAdapter wraps an IDService to implement the contracts.Provider interface
type IDProviderAdapter struct {
	idService ports.IDService
	name      string
}

// NewIDProviderAdapter creates a new IDProviderAdapter
func NewIDProviderAdapter(service ports.IDService, name string) *IDProviderAdapter {
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
func (p *IDProviderAdapter) GetIDService() ports.IDService {
	return p.idService
}

// CreateIDProvider creates an ID provider using provider self-configuration.
// The provider reads its own environment variables - composition layer is provider-agnostic.
//
// Uses CONFIG_ID_PROVIDER environment variable to select which provider to use:
//   - "google_uuidv7" or "uuidv7" → Google UUID v7 provider
//   - "noop", "mock", or "" → NoOp provider (default)
func CreateIDProvider() (contracts.Provider, error) {
	providerName := strings.ToLower(os.Getenv("CONFIG_ID_PROVIDER"))

	// Normalize provider names
	switch providerName {
	case "uuidv7":
		providerName = "google_uuidv7"
	case "mock", "":
		providerName = "noop"
	}

	// Let the provider build and configure itself from environment
	idService, err := registry.BuildIDProviderFromEnv(providerName)
	if err != nil {
		// Fallback to noop for unknown providers
		fmt.Printf("🆔 ID provider '%s' not found, using fallback noop: %v\n", providerName, err)
		idService = ports.NewNoOpIDService()
		providerName = "noop"
	}

	fmt.Printf("🆔 Created ID provider (config=%s): %s\n", providerName, idService.GetProviderInfo())

	// Wrap to satisfy composition contracts
	return NewIDProviderAdapter(idService, providerName), nil
}
