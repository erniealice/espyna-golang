package contracts

import (
	"context"
)

// ============================================================================
// Domain Constants
// ============================================================================

// Domain represents the business domains in the application
type Domain string

const (
	// Entity domain - user, client, group, role management
	DomainEntity Domain = "entity"

	// Event domain - event management and processing
	DomainEvent Domain = "event"

	// Payment domain - payments, billing, transactions
	DomainPayment Domain = "payment"

	// Product domain - products, collections, pricing
	DomainProduct Domain = "product"

	// Record domain - record management
	DomainRecord Domain = "record"

	// Subscription domain - plans, subscriptions, billing
	DomainSubscription Domain = "subscription"

	// Workflow domain - workflow templates, instances, execution
	DomainWorkflow Domain = "workflow"
)

// ============================================================================
// Core Service Interfaces
// ============================================================================

// Service defines the basic interface for all services in the container
type Service interface {
	// Name returns the service name
	Name() string

	// Start starts the service
	Start(ctx context.Context) error

	// Stop stops the service
	Stop(ctx context.Context) error

	// Health checks the service health
	Health(ctx context.Context) error
}

// UseCase defines the basic interface for all use cases
type UseCase interface {
	// Name returns the use case name
	Name() string

	// Domain returns the business domain
	Domain() Domain

	// Validate validates input parameters
	Validate(input interface{}) error

	// Execute executes the use case
	Execute(ctx context.Context, input interface{}) (interface{}, error)
}

// Repository defines the basic interface for all repositories
type Repository interface {
	// Name returns the repository name
	Name() string

	// Type returns the repository type
	Type() string

	// Health checks repository health
	Health(ctx context.Context) error
}

// ============================================================================
// Handler and Middleware Interfaces
// ============================================================================

// Handler defines the basic interface for HTTP handlers (component pattern)
type Handler interface {
	// Name returns the handler name
	Name() string

	// Method returns the HTTP method
	Method() string

	// Path returns the handler path
	Path() string

	// Handle handles the HTTP request
	Handle(ctx context.Context, request interface{}) (interface{}, error)
}

// Middleware defines the interface for HTTP middleware
type Middleware interface {
	// Name returns the middleware name
	Name() string

	// Execute executes the middleware
	Execute(ctx context.Context, request interface{}, next Handler) (interface{}, error)
}

// ============================================================================
// Provider Interfaces
// ============================================================================

// ProviderType represents different types of providers
type ProviderType string

const (
	ProviderTypeDatabase ProviderType = "database"
	ProviderTypeAuth     ProviderType = "auth"
	ProviderTypeStorage  ProviderType = "storage"
	ProviderTypeServer   ProviderType = "server"
	ProviderTypeID       ProviderType = "id" // ID generation provider
	ProviderTypeCache    ProviderType = "cache"
	ProviderTypeMessage  ProviderType = "message"
	ProviderTypeMetrics  ProviderType = "metrics"
	ProviderTypeLogger   ProviderType = "logger"
)

// Provider defines the interface for all providers
type Provider interface {
	// Type returns the provider type
	Type() ProviderType

	// Name returns the provider name
	Name() string

	// Initialize initializes the provider
	Initialize(config interface{}) error

	// Health checks provider health
	Health(ctx context.Context) error

	// Close closes the provider and releases resources
	Close() error
}

// RepositoryProvider defines the interface for providers that can create repositories and provide connections.
// This decouples domain repository creation from specific infrastructure implementations.
type RepositoryProvider interface {
	// CreateRepository creates a new repository instance based on entity name, connection, and table name.
	CreateRepository(entityName string, conn interface{}, tableName string) (interface{}, error)
	// GetConnection returns the underlying connection object used by the provider.
	GetConnection() any
}

// ConfigProvider defines the interface for configuration providers
type ConfigProvider interface {
	// Get gets a configuration value
	Get(key string) (interface{}, error)

	// Set sets a configuration value
	Set(key string, value interface{}) error

	// Watch watches for configuration changes
	Watch(key string, callback func(interface{})) error
}
