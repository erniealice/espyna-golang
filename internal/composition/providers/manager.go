package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

// Manager is the unified provider management system that handles all external dependencies.
// Providers read their own configuration from environment variables.
type Manager struct {
	mu sync.RWMutex

	// Core providers
	databaseProvider contracts.Provider
	authProvider     contracts.Provider
	storageProvider  contracts.Provider
	serverProvider   contracts.Provider
	idProvider       contracts.Provider // ID generation provider (UUID v7, etc.)

	// Additional providers
	cacheProvider   contracts.Provider
	metricsProvider contracts.Provider
	loggerProvider  contracts.Provider
	tracingProvider contracts.Provider

	// Provider registry
	providerRegistry *Registry

	// Database table configuration (obtained from registry based on active provider)
	dbTableConfig *registry.DatabaseTableConfig

	// Health monitoring
	healthCheckInterval time.Duration
	healthChecker       map[contracts.ProviderType]contracts.HealthChecker

	// State
	initialized bool
	closed      bool
}

// NewManager creates a new unified provider manager.
// All providers read their configuration from environment variables.
// Table configuration is obtained from the registry based on the active database provider.
func NewManager() (*Manager, error) {
	manager := &Manager{
		providerRegistry:    NewRegistry(),
		healthChecker:       make(map[contracts.ProviderType]contracts.HealthChecker),
		healthCheckInterval: time.Minute * 5,
	}

	// Initialize core providers from environment
	if err := manager.initializeCoreProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize core providers: %w", err)
	}

	// Get table config from registry based on active database provider
	if manager.databaseProvider != nil {
		providerName := manager.databaseProvider.Name()
		tableConfig, err := registry.BuildDatabaseTableConfig(providerName)
		if err != nil {
			return nil, fmt.Errorf("failed to build table config for %s: %w", providerName, err)
		}
		manager.dbTableConfig = tableConfig
	} else {
		// Fallback to default if no database provider
		manager.dbTableConfig = registry.DefaultDatabaseTableConfig()
	}

	return manager, nil
}

// initializeCoreProviders initializes the core providers from environment
func (m *Manager) initializeCoreProviders() error {
	// Initialize database provider via the provider registry
	dbProvider, err := m.providerRegistry.CreateAndRegisterDatabaseProvider()
	if err != nil {
		return fmt.Errorf("failed to create database provider: %w", err)
	}
	m.databaseProvider = dbProvider

	// Initialize auth provider via the provider registry
	authProvider, err := m.providerRegistry.CreateAndRegisterAuthProvider()
	if err != nil {
		return fmt.Errorf("failed to create auth provider: %w", err)
	}
	m.authProvider = authProvider

	// Initialize storage provider via the provider registry
	storageProvider, err := m.providerRegistry.CreateAndRegisterStorageProvider()
	if err != nil {
		return fmt.Errorf("failed to create storage provider: %w", err)
	}
	m.storageProvider = storageProvider

	// Initialize ID provider via the provider registry
	idProvider, err := m.providerRegistry.CreateAndRegisterIDProvider()
	if err != nil {
		return fmt.Errorf("failed to create ID provider: %w", err)
	}
	m.idProvider = idProvider

	return nil
}

// Initialize initializes all providers
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return fmt.Errorf("provider manager already initialized")
	}

	// Initialize all providers
	// Database, Auth, Storage, and ID providers are already initialized and registered by their respective
	// CreateAndRegister methods within the registry, so they are excluded from this loop.
	providers := []contracts.Provider{
		m.serverProvider,
		m.cacheProvider,
		m.metricsProvider,
		m.loggerProvider,
		m.tracingProvider,
	}

	for _, provider := range providers {
		if provider == nil {
			continue
		}

		if err := provider.Initialize(nil); err != nil {
			return fmt.Errorf("failed to initialize provider %s: %w", provider.Name(), err)
		}

		// Register provider in provider registry
		m.providerRegistry.Register(provider)

		// Setup health checking
		if healthChecker, ok := provider.(contracts.HealthChecker); ok {
			m.healthChecker[provider.Type()] = healthChecker
		}
	}

	// Start health monitoring
	go m.startHealthMonitoring()

	m.initialized = true
	return nil
}

// SetDatabaseProvider sets the database provider
func (m *Manager) SetDatabaseProvider(provider contracts.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.databaseProvider = provider
}

// SetAuthProvider sets the auth provider
func (m *Manager) SetAuthProvider(provider contracts.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authProvider = provider
}

// SetStorageProvider sets the storage provider
func (m *Manager) SetStorageProvider(provider contracts.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storageProvider = provider
}

// SetServerProvider sets the server provider
func (m *Manager) SetServerProvider(provider contracts.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serverProvider = provider
}

// GetDatabaseProvider returns the database provider
func (m *Manager) GetDatabaseProvider() contracts.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.databaseProvider
}

// GetAuthProvider returns the auth provider
func (m *Manager) GetAuthProvider() contracts.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.authProvider
}

// GetStorageProvider returns the storage provider
func (m *Manager) GetStorageProvider() contracts.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storageProvider
}

// GetServerProvider returns the server provider
func (m *Manager) GetServerProvider() contracts.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.serverProvider
}

// GetIDProvider returns the ID provider
func (m *Manager) GetIDProvider() contracts.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.idProvider
}

// SetIDProvider sets the ID provider
func (m *Manager) SetIDProvider(provider contracts.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idProvider = provider
}

// GetDBTableConfig returns the database table configuration
func (m *Manager) GetDBTableConfig() *registry.DatabaseTableConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dbTableConfig
}

// CheckHealth checks the health of all providers
func (m *Manager) CheckHealth(ctx context.Context) map[string]contracts.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]contracts.HealthStatus)

	for providerType, healthChecker := range m.healthChecker {
		status := healthChecker.Check(ctx)
		results[string(providerType)] = status
	}

	return results
}

// GetRegistry returns the provider registry
func (m *Manager) GetRegistry() *Registry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.providerRegistry
}

// Close closes all providers and releases resources
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	var errors []error

	// Close all providers
	providers := []contracts.Provider{
		m.databaseProvider,
		m.authProvider,
		m.storageProvider,
		m.serverProvider,
		m.idProvider,
		m.cacheProvider,
		m.metricsProvider,
		m.loggerProvider,
		m.tracingProvider,
	}

	for _, provider := range providers {
		if provider == nil {
			continue
		}

		if err := provider.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close provider %s: %w", provider.Name(), err))
		}
	}

	m.closed = true

	if len(errors) > 0 {
		return fmt.Errorf("errors during provider shutdown: %v", errors)
	}

	return nil
}

// IsInitialized returns whether the manager has been initialized
func (m *Manager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

// IsClosed returns whether the manager has been closed
func (m *Manager) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// startHealthMonitoring starts the health monitoring routine
func (m *Manager) startHealthMonitoring() {
	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

// performHealthCheck performs health check on all providers
func (m *Manager) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	healthResults := m.CheckHealth(ctx)

	// Log health results (implement logging as needed)
	for providerType, status := range healthResults {
		if status.Status != "healthy" {
			// Log unhealthy provider
			fmt.Printf("Provider %s is %s: %s\n", providerType, status.Status, status.Message)
		}
	}
}
