package registry

import (
	"context"
	"fmt"
	"log"
	"sync"

	"leapfor.xyz/espyna/internal/application/ports"
	authpb "leapfor.xyz/esqyma/golang/v1/infrastructure/auth"
	dbpb "leapfor.xyz/esqyma/golang/v1/infrastructure/database"
	storagepb "leapfor.xyz/esqyma/golang/v1/infrastructure/storage"
	emailpb "leapfor.xyz/esqyma/golang/v1/integration/email"
	paymentpb "leapfor.xyz/esqyma/golang/v1/integration/payment"
)

// =============================================================================
// Generic Instance Registry
// =============================================================================
//
// InstanceRegistry manages active provider instances at runtime.
// Uses Go generics to eliminate duplication across provider types.
//
// =============================================================================

// ProviderInstance is a constraint for provider types that have common methods.
type ProviderInstance interface {
	Name() string
	IsEnabled() bool
	IsHealthy(ctx context.Context) error
	Close() error
}

// InstanceRegistry manages active provider instances of a specific type.
type InstanceRegistry[T ProviderInstance] struct {
	instances map[string]T
	active    string // for single-active-provider scenarios
	mutex     sync.RWMutex
	typeName  string // for error messages
}

// NewInstanceRegistry creates a new instance registry.
func NewInstanceRegistry[T ProviderInstance](typeName string) *InstanceRegistry[T] {
	return &InstanceRegistry[T]{
		instances: make(map[string]T),
		typeName:  typeName,
	}
}

// Register adds a provider instance to the registry.
func (r *InstanceRegistry[T]) Register(provider T) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("%s provider must have a non-empty name", r.typeName)
	}

	r.instances[name] = provider
	return nil
}

// Get retrieves a provider by name.
func (r *InstanceRegistry[T]) Get(name string) (T, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	provider, exists := r.instances[name]
	if !exists {
		var zero T
		return zero, fmt.Errorf("%s provider '%s' not found", r.typeName, name)
	}
	return provider, nil
}

// SetActive sets the active provider by name (for single-active scenarios).
func (r *InstanceRegistry[T]) SetActive(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	provider, exists := r.instances[name]
	if !exists {
		return fmt.Errorf("%s provider '%s' is not registered", r.typeName, name)
	}

	if !provider.IsEnabled() {
		return fmt.Errorf("%s provider '%s' is not enabled", r.typeName, name)
	}

	r.active = name
	log.Printf("Set '%s' as active %s provider", name, r.typeName)
	return nil
}

// GetActive returns the active provider.
func (r *InstanceRegistry[T]) GetActive() (T, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.active == "" {
		var zero T
		return zero, false
	}
	return r.instances[r.active], true
}

// List returns all registered provider names.
func (r *InstanceRegistry[T]) List() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.instances))
	for name := range r.instances {
		names = append(names, name)
	}
	return names
}

// HealthCheck checks health of all enabled providers.
func (r *InstanceRegistry[T]) HealthCheck(ctx context.Context) map[string]bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	status := make(map[string]bool)
	for name, provider := range r.instances {
		if provider.IsEnabled() {
			err := provider.IsHealthy(ctx)
			status[name] = err == nil
		}
	}
	return status
}

// Close closes all providers.
func (r *InstanceRegistry[T]) Close() []error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errs []error
	for name, provider := range r.instances {
		if err := provider.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s provider %s: %w", r.typeName, name, err))
		}
	}
	return errs
}

// =============================================================================
// Registry - Main Runtime Instance Manager
// =============================================================================

// Registry manages all active provider instances
type Registry struct {
	database  *InstanceRegistry[ports.DatabaseProvider]
	auth      *InstanceRegistry[ports.AuthProvider]
	storage   *InstanceRegistry[ports.StorageProvider]
	email     *InstanceRegistry[ports.EmailProvider]
	payment   *InstanceRegistry[ports.PaymentProvider]
	migration map[string]ports.MigrationService
	mutex     sync.RWMutex
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		database:  NewInstanceRegistry[ports.DatabaseProvider]("database"),
		auth:      NewInstanceRegistry[ports.AuthProvider]("auth"),
		storage:   NewInstanceRegistry[ports.StorageProvider]("storage"),
		email:     NewInstanceRegistry[ports.EmailProvider]("email"),
		payment:   NewInstanceRegistry[ports.PaymentProvider]("payment"),
		migration: make(map[string]ports.MigrationService),
	}
}

// =============================================================================
// Database Provider Methods
// =============================================================================

func (r *Registry) RegisterDatabaseProvider(provider ports.DatabaseProvider) error {
	if provider == nil {
		return fmt.Errorf("database provider cannot be nil")
	}
	return r.database.Register(provider)
}

func (r *Registry) GetDatabaseProvider(name string) (ports.DatabaseProvider, error) {
	return r.database.Get(name)
}

func (r *Registry) CreateAndRegisterDatabaseProviderWithProto(name string, config *dbpb.DatabaseProviderConfig) error {
	factory, exists := GetDatabaseProviderFactory(name)
	if !exists {
		return fmt.Errorf("no factory registered for database provider: %s (available: %v)", name, ListAvailableDatabaseProviderFactories())
	}

	provider := factory()
	if err := provider.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize provider %s: %w", name, err)
	}

	return r.RegisterDatabaseProvider(provider)
}

// =============================================================================
// Auth Provider Methods
// =============================================================================

func (r *Registry) RegisterAuthProvider(provider ports.AuthProvider) error {
	if provider == nil {
		return fmt.Errorf("auth provider cannot be nil")
	}
	return r.auth.Register(provider)
}

func (r *Registry) GetAuthProvider(name string) (ports.AuthProvider, error) {
	return r.auth.Get(name)
}

func (r *Registry) CreateAndRegisterAuthProviderWithProto(name string, config *authpb.ProviderConfig) error {
	factory, exists := GetAuthProviderFactory(name)
	if !exists {
		return fmt.Errorf("no factory registered for auth provider: %s (available: %v)", name, ListAvailableAuthProviderFactories())
	}

	provider := factory()
	if err := provider.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize auth provider %s: %w", name, err)
	}

	return r.RegisterAuthProvider(provider)
}

// =============================================================================
// Storage Provider Methods
// =============================================================================

func (r *Registry) RegisterStorageProvider(provider ports.StorageProvider) error {
	if provider == nil {
		return fmt.Errorf("storage provider cannot be nil")
	}
	return r.storage.Register(provider)
}

func (r *Registry) GetStorageProvider(name string) (ports.StorageProvider, error) {
	return r.storage.Get(name)
}

func (r *Registry) CreateAndRegisterStorageProviderWithProto(name string, config *storagepb.StorageProviderConfig) error {
	factory, exists := GetStorageProviderFactory(name)
	if !exists {
		return fmt.Errorf("no factory registered for storage provider: %s (available: %v)", name, ListAvailableStorageProviderFactories())
	}

	provider := factory()
	if err := provider.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize storage provider %s: %w", name, err)
	}

	return r.RegisterStorageProvider(provider)
}

// =============================================================================
// Email Provider Methods
// =============================================================================

func (r *Registry) RegisterEmailProvider(provider ports.EmailProvider) error {
	if provider == nil {
		return fmt.Errorf("email provider cannot be nil")
	}
	if err := r.email.Register(provider); err != nil {
		return err
	}

	// Auto-set as active if first enabled provider
	if active, ok := r.email.GetActive(); !ok || active == nil {
		if provider.IsEnabled() {
			_ = r.email.SetActive(provider.Name())
		}
	}
	return nil
}

func (r *Registry) GetEmailProvider(name string) (ports.EmailProvider, error) {
	return r.email.Get(name)
}

func (r *Registry) SetActiveEmailProvider(name string) error {
	return r.email.SetActive(name)
}

func (r *Registry) GetActiveEmailProvider() ports.EmailProvider {
	provider, _ := r.email.GetActive()
	return provider
}

func (r *Registry) CreateAndRegisterEmailProviderWithProto(name string, config *emailpb.EmailProviderConfig) error {
	factory, exists := GetEmailProviderFactory(name)
	if !exists {
		return fmt.Errorf("no factory registered for email provider: %s (available: %v)", name, ListAvailableEmailProviderFactories())
	}

	provider := factory()
	if err := provider.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize email provider %s: %w", name, err)
	}

	return r.RegisterEmailProvider(provider)
}

// =============================================================================
// Payment Provider Methods
// =============================================================================

func (r *Registry) RegisterPaymentProvider(provider ports.PaymentProvider) error {
	if provider == nil {
		return fmt.Errorf("payment provider cannot be nil")
	}
	if err := r.payment.Register(provider); err != nil {
		return err
	}

	// Auto-set as active if first enabled provider
	if active, ok := r.payment.GetActive(); !ok || active == nil {
		if provider.IsEnabled() {
			_ = r.payment.SetActive(provider.Name())
		}
	}
	return nil
}

func (r *Registry) GetPaymentProvider(name string) (ports.PaymentProvider, error) {
	return r.payment.Get(name)
}

func (r *Registry) SetActivePaymentProvider(name string) error {
	return r.payment.SetActive(name)
}

func (r *Registry) GetActivePaymentProvider() ports.PaymentProvider {
	provider, _ := r.payment.GetActive()
	return provider
}

func (r *Registry) CreateAndRegisterPaymentProviderWithProto(name string, config *paymentpb.PaymentProviderConfig) error {
	factory, exists := GetPaymentProviderFactory(name)
	if !exists {
		return fmt.Errorf("no factory registered for payment provider: %s (available: %v)", name, ListAvailablePaymentProviderFactories())
	}

	provider := factory()
	if err := provider.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize payment provider %s: %w", name, err)
	}

	return r.RegisterPaymentProvider(provider)
}

// =============================================================================
// Migration Service Methods
// =============================================================================

func (r *Registry) RegisterMigrationService(providerName string, service ports.MigrationService) error {
	if service == nil {
		return fmt.Errorf("migration service cannot be nil")
	}
	if providerName == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.migration[providerName] = service
	return nil
}

func (r *Registry) GetMigrationService(providerName string) (ports.MigrationService, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	service, exists := r.migration[providerName]
	if !exists {
		return nil, fmt.Errorf("migration service for provider '%s' not found", providerName)
	}
	return service, nil
}

// =============================================================================
// Registry-wide Methods
// =============================================================================

// ProviderLists contains lists of available providers by category
type ProviderLists struct {
	DatabaseProviders []string `json:"database_providers"`
	AuthProviders     []string `json:"auth_providers"`
	StorageProviders  []string `json:"storage_providers"`
	EmailProviders    []string `json:"email_providers"`
	PaymentProviders  []string `json:"payment_providers"`
}

// ListAvailableProviders returns lists of all registered providers
func (r *Registry) ListAvailableProviders() *ProviderLists {
	return &ProviderLists{
		DatabaseProviders: r.database.List(),
		AuthProviders:     r.auth.List(),
		StorageProviders:  r.storage.List(),
		EmailProviders:    r.email.List(),
		PaymentProviders:  r.payment.List(),
	}
}

// RegistryHealthStatus contains health status for all providers
type RegistryHealthStatus struct {
	DatabaseProviders map[string]bool `json:"database_providers"`
	AuthProviders     map[string]bool `json:"auth_providers"`
	StorageProviders  map[string]bool `json:"storage_providers"`
	EmailProviders    map[string]bool `json:"email_providers"`
	PaymentProviders  map[string]bool `json:"payment_providers"`
}

// HealthCheck performs health checks on all enabled providers
func (r *Registry) HealthCheck(ctx context.Context) (*RegistryHealthStatus, error) {
	return &RegistryHealthStatus{
		DatabaseProviders: r.database.HealthCheck(ctx),
		AuthProviders:     r.auth.HealthCheck(ctx),
		StorageProviders:  r.storage.HealthCheck(ctx),
		EmailProviders:    r.email.HealthCheck(ctx),
		PaymentProviders:  r.payment.HealthCheck(ctx),
	}, nil
}

// Close closes all providers and cleans up resources
func (r *Registry) Close() error {
	var allErrs []error

	allErrs = append(allErrs, r.database.Close()...)
	allErrs = append(allErrs, r.auth.Close()...)
	allErrs = append(allErrs, r.storage.Close()...)
	allErrs = append(allErrs, r.email.Close()...)
	allErrs = append(allErrs, r.payment.Close()...)

	// Close migration services
	r.mutex.Lock()
	for name, service := range r.migration {
		if err := service.Close(); err != nil {
			allErrs = append(allErrs, fmt.Errorf("failed to close migration service %s: %w", name, err))
		}
	}
	r.mutex.Unlock()

	if len(allErrs) > 0 {
		return fmt.Errorf("multiple errors occurred during cleanup: %v", allErrs)
	}
	return nil
}
