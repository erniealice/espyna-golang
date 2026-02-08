package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"
	"github.com/erniealice/espyna-golang/internal/composition/providers/infrastructure"
	"github.com/erniealice/espyna-golang/internal/composition/providers/integration"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// Registry orchestrates all provider sub-registries
type Registry struct {
	mu sync.RWMutex

	// Sub-registries
	infrastructure *infrastructure.Registry
	domain         *domain.Registry
	integration    *integration.Registry

	// Legacy compatibility - for providers that need direct access
	providers map[contracts.ProviderType]map[string]contracts.Provider
	instances map[string]contracts.Provider
	metadata  map[string]*ProviderMetadata
}

// ProviderMetadata contains metadata about a provider
type ProviderMetadata struct {
	Name        string                 `json:"name"`
	Type        contracts.ProviderType `json:"type"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags"`
	Config      map[string]interface{} `json:"config"`
	Status      string                 `json:"status"` // "active", "inactive", "error"
	LastError   string                 `json:"last_error,omitempty"`
	CreatedAt   int64                  `json:"created_at"`
	UpdatedAt   int64                  `json:"updated_at"`
}

// ProviderDefinition defines a provider that can be registered
type ProviderDefinition struct {
	Type         contracts.ProviderType `json:"type"`
	Name         string                 `json:"name"`
	Factory      ProviderFactory        `json:"-"`
	Config       map[string]interface{} `json:"config"`
	Metadata     ProviderMetadata       `json:"metadata"`
	Dependencies []string               `json:"dependencies"`
}

// ProviderFactory creates provider instances
type ProviderFactory func(config map[string]interface{}) (contracts.Provider, error)

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		infrastructure: infrastructure.NewRegistry(),
		integration:    integration.NewRegistry(),
		providers:      make(map[contracts.ProviderType]map[string]contracts.Provider),
		instances:      make(map[string]contracts.Provider),
		metadata:       make(map[string]*ProviderMetadata),
	}
}

// InitializeAll initializes all sub-registries from environment.
// Each provider reads its own configuration from environment variables.
func (r *Registry) InitializeAll(dbTableConfig *registry.DatabaseTableConfig) error {
	// Initialize infrastructure providers first (database, auth, storage, id)
	if err := r.infrastructure.InitializeAll(); err != nil {
		return fmt.Errorf("failed to initialize infrastructure providers: %w", err)
	}

	// Initialize domain registry with database provider
	dbProvider := r.infrastructure.GetDatabase()
	if dbProvider != nil {
		r.domain = domain.NewRegistry(dbProvider, dbTableConfig)
		if err := r.domain.InitializeAll(); err != nil {
			return fmt.Errorf("failed to initialize domain repositories: %w", err)
		}
	}

	// Initialize integration providers (email, payment)
	if err := r.integration.InitializeAll(); err != nil {
		return fmt.Errorf("failed to initialize integration providers: %w", err)
	}

	// Register all providers in the legacy maps for compatibility
	r.registerInfrastructureProviders()

	return nil
}

// registerInfrastructureProviders registers infrastructure providers in legacy maps
func (r *Registry) registerInfrastructureProviders() {
	for _, provider := range r.infrastructure.GetAll() {
		if provider != nil {
			_ = r.Register(provider)
		}
	}
}

// =============================================================================
// PROVIDER CREATION METHODS (from environment)
// =============================================================================

// CreateAndRegisterDatabaseProvider creates and registers a database provider.
// Delegates to infrastructure sub-registry. Provider reads config from env.
func (r *Registry) CreateAndRegisterDatabaseProvider() (contracts.Provider, error) {
	provider, err := infrastructure.CreateDatabaseProvider()
	if err != nil {
		return nil, err
	}
	r.infrastructure.SetDatabase(provider)
	if err := r.Register(provider); err != nil {
		return nil, err
	}
	return provider, nil
}

// CreateAndRegisterAuthProvider creates and registers an auth provider.
// Delegates to infrastructure sub-registry. Provider reads config from env.
func (r *Registry) CreateAndRegisterAuthProvider() (contracts.Provider, error) {
	provider, err := infrastructure.CreateAuthProvider()
	if err != nil {
		return nil, err
	}
	r.infrastructure.SetAuth(provider)
	if err := r.Register(provider); err != nil {
		return nil, err
	}
	return provider, nil
}

// CreateAndRegisterStorageProvider creates and registers a storage provider.
// Delegates to infrastructure sub-registry. Provider reads config from env.
func (r *Registry) CreateAndRegisterStorageProvider() (contracts.Provider, error) {
	provider, err := infrastructure.CreateStorageProvider()
	if err != nil {
		return nil, err
	}
	r.infrastructure.SetStorage(provider)
	if err := r.Register(provider); err != nil {
		return nil, err
	}
	return provider, nil
}

// CreateAndRegisterIDProvider creates and registers an ID provider.
// Delegates to infrastructure sub-registry. Provider reads config from env.
func (r *Registry) CreateAndRegisterIDProvider() (contracts.Provider, error) {
	provider, err := infrastructure.CreateIDProvider()
	if err != nil {
		return nil, err
	}
	r.infrastructure.SetID(provider)
	if err := r.Register(provider); err != nil {
		return nil, err
	}
	return provider, nil
}

// =============================================================================
// INFRASTRUCTURE PROVIDER ACCESS
// =============================================================================

// GetDatabase returns the database provider
func (r *Registry) GetDatabase() contracts.Provider {
	return r.infrastructure.GetDatabase()
}

// GetAuth returns the auth provider
func (r *Registry) GetAuth() contracts.Provider {
	return r.infrastructure.GetAuth()
}

// GetStorage returns the storage provider
func (r *Registry) GetStorage() contracts.Provider {
	return r.infrastructure.GetStorage()
}

// GetID returns the ID provider
func (r *Registry) GetID() contracts.Provider {
	return r.infrastructure.GetID()
}

// GetIDService returns the underlying ID service from the ID provider
func (r *Registry) GetIDService() ports.IDService {
	idProvider := r.infrastructure.GetID()
	if idProvider == nil {
		return nil
	}
	if adapter, ok := idProvider.(*infrastructure.IDProviderAdapter); ok {
		return adapter.GetIDService()
	}
	return nil
}

// =============================================================================
// DOMAIN REGISTRY ACCESS
// =============================================================================

// GetDomain returns the domain registry for accessing repository collections
func (r *Registry) GetDomain() *domain.Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.domain
}

// =============================================================================
// INTEGRATION PROVIDER ACCESS
// =============================================================================

// GetEmail returns the email provider
func (r *Registry) GetEmail() ports.EmailProvider {
	return r.integration.GetEmail()
}

// GetPayment returns the payment provider
func (r *Registry) GetPayment() ports.PaymentProvider {
	return r.integration.GetPayment()
}

// =============================================================================
// LEGACY PROVIDER REGISTRATION (for compatibility)
// =============================================================================

// Register registers a provider in the registry
func (r *Registry) Register(provider contracts.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	providerType := provider.Type()
	providerName := provider.Name()

	// Initialize provider type map if not exists
	if r.providers[providerType] == nil {
		r.providers[providerType] = make(map[string]contracts.Provider)
	}

	// Check for duplicate registration
	if _, exists := r.providers[providerType][providerName]; exists {
		return fmt.Errorf("provider %s of type %s already registered", providerName, providerType)
	}

	// Register provider
	r.providers[providerType][providerName] = provider
	r.instances[providerName] = provider

	// Create metadata
	metadata := &ProviderMetadata{
		Name:        providerName,
		Type:        providerType,
		Version:     "1.0.0",
		Description: fmt.Sprintf("%s provider for %s", providerType, providerName),
		Tags:        []string{string(providerType)},
		Config:      make(map[string]interface{}),
		Status:      "active",
		CreatedAt:   getCurrentTimestamp(),
		UpdatedAt:   getCurrentTimestamp(),
	}

	r.metadata[providerName] = metadata

	return nil
}

// Get gets a provider by type and name
func (r *Registry) Get(providerType contracts.ProviderType, providerName string) (contracts.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.providers[providerType] == nil {
		return nil, fmt.Errorf("provider type %s not found", providerType)
	}

	provider, exists := r.providers[providerType][providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s of type %s not found", providerName, providerType)
	}

	return provider, nil
}

// GetByType gets all providers of a specific type
func (r *Registry) GetByType(providerType contracts.ProviderType) (map[string]contracts.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider type %s not found", providerType)
	}

	// Return a copy to prevent external modification
	result := make(map[string]contracts.Provider)
	for name, provider := range providers {
		result[name] = provider
	}

	return result, nil
}

// GetAll gets all registered providers
func (r *Registry) GetAll() map[contracts.ProviderType]map[string]contracts.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[contracts.ProviderType]map[string]contracts.Provider)

	for providerType, providers := range r.providers {
		result[providerType] = make(map[string]contracts.Provider)
		for name, provider := range providers {
			result[providerType][name] = provider
		}
	}

	return result
}

// List returns a list of all registered provider names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.instances {
		names = append(names, name)
	}

	return names
}

// GetMetadata gets metadata for a provider
func (r *Registry) GetMetadata(providerName string) (*ProviderMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[providerName]
	if !exists {
		return nil, fmt.Errorf("metadata for provider %s not found", providerName)
	}

	// Return a copy to prevent external modification
	metadataCopy := *metadata
	return &metadataCopy, nil
}

// Stats returns registry statistics
func (r *Registry) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]interface{})

	typeCounts := make(map[string]int)
	statusCounts := make(map[string]int)

	for providerType, providers := range r.providers {
		typeCounts[string(providerType)] = len(providers)
	}

	for _, metadata := range r.metadata {
		statusCounts[metadata.Status]++
	}

	stats["total_providers"] = len(r.instances)
	stats["providers_by_type"] = typeCounts
	stats["providers_by_status"] = statusCounts

	return stats
}

// HealthCheck checks health of all providers
func (r *Registry) HealthCheck(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errs []error

	for name, provider := range r.instances {
		if err := provider.Health(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("health check failures: %v", errs)
	}
	return nil
}

// Close closes all providers and sub-registries
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error

	// Close integration providers
	if r.integration != nil {
		if err := r.integration.Close(); err != nil {
			errs = append(errs, fmt.Errorf("integration: %w", err))
		}
	}

	// Close infrastructure providers
	if r.infrastructure != nil {
		if err := r.infrastructure.Close(); err != nil {
			errs = append(errs, fmt.Errorf("infrastructure: %w", err))
		}
	}

	// Clear legacy maps
	r.providers = make(map[contracts.ProviderType]map[string]contracts.Provider)
	r.instances = make(map[string]contracts.Provider)
	r.metadata = make(map[string]*ProviderMetadata)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing registry: %v", errs)
	}
	return nil
}

// Helper function to get current timestamp
func getCurrentTimestamp() int64 {
	// In a real implementation, you'd use time.Now().Unix()
	return 1234567890
}
