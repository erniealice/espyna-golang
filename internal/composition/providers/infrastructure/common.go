package infrastructure

import (
	"context"
	"fmt"

	"leapfor.xyz/espyna/internal/composition/contracts"
)

// ProviderWrapper adapts existing infrastructure providers to the contracts.Provider interface
type ProviderWrapper struct {
	provider     interface{}
	providerType contracts.ProviderType
}

// NewProviderWrapper creates a wrapper with explicit type
func NewProviderWrapper(provider interface{}, providerType contracts.ProviderType) *ProviderWrapper {
	return &ProviderWrapper{
		provider:     provider,
		providerType: providerType,
	}
}

// Provider returns the underlying provider
func (p *ProviderWrapper) Provider() interface{} {
	return p.provider
}

// Type returns the provider type
func (p *ProviderWrapper) Type() contracts.ProviderType {
	return p.providerType
}

// Name returns the provider name
func (p *ProviderWrapper) Name() string {
	// Try Name() first (used by ports.AuthProvider, etc.)
	if prov, ok := p.provider.(interface{ Name() string }); ok {
		return prov.Name()
	}
	// Try GetName() as fallback
	if prov, ok := p.provider.(interface{ GetName() string }); ok {
		return prov.GetName()
	}
	// Generate unique name based on type to avoid duplicate registration
	return fmt.Sprintf("wrapped-%s-%p", p.Type(), p.provider)
}

// Initialize initializes the provider
func (p *ProviderWrapper) Initialize(config interface{}) error {
	if prov, ok := p.provider.(interface{ Initialize(map[string]any) error }); ok {
		// Convert config to map if needed
		if cfg, ok := config.(map[string]any); ok {
			return prov.Initialize(cfg)
		}
		if config == nil {
			return prov.Initialize(nil)
		}
	}
	// Also check for the other type of Initialize
	if prov, ok := p.provider.(interface{ Initialize(interface{}) error }); ok {
		return prov.Initialize(config)
	}
	return nil
}

// Health checks provider health
func (p *ProviderWrapper) Health(ctx context.Context) error {
	if prov, ok := p.provider.(interface{ Health(context.Context) error }); ok {
		return prov.Health(ctx)
	}
	return nil
}

// Close closes the provider and releases resources
func (p *ProviderWrapper) Close() error {
	if prov, ok := p.provider.(interface{ Close() error }); ok {
		return prov.Close()
	}
	return nil
}

// CreateRepository implements the contracts.RepositoryProvider interface by delegating to the wrapped provider.
func (p *ProviderWrapper) CreateRepository(entityName string, conn interface{}, tableName string) (interface{}, error) {
	if repoCreator, ok := p.provider.(interface {
		CreateRepository(entityName string, conn interface{}, tableName string) (interface{}, error)
	}); ok {
		return repoCreator.CreateRepository(entityName, conn, tableName)
	}
	return nil, fmt.Errorf("wrapped provider does not support CreateRepository method")
}

// GetConnection implements the contracts.RepositoryProvider interface by delegating to the wrapped provider.
func (p *ProviderWrapper) GetConnection() any {
	if repoCreator, ok := p.provider.(interface {
		GetConnection() any
	}); ok {
		return repoCreator.GetConnection()
	}
	return nil // Return nil if the wrapped provider does not support GetConnection
}
