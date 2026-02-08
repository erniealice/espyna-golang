package infrastructure

import (
	"context"

	dbpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/database"
)

// DatabaseProvider defines the contract for database providers
// This interface abstracts the database connection and initialization logic
type DatabaseProvider interface {
	// Name returns the name of the provider (e.g., "postgresql", "firestore", "mock")
	Name() string

	// Initialize sets up the database connection with the given configuration
	Initialize(config *dbpb.DatabaseProviderConfig) error

	// GetConnection returns the database connection
	// Returns *sql.DB for SQL-based providers, or a provider-specific client for others
	GetConnection() any

	// IsHealthy checks if the database connection is healthy
	IsHealthy(ctx context.Context) error

	// Close closes the database connection and cleans up resources
	Close() error

	// IsEnabled returns whether this provider is currently enabled
	IsEnabled() bool
}

// RepositoryProvider defines the simplified contract for data source providers
// This interface enables direct repository creation from database providers.
type RepositoryProvider interface {
	// Name returns the provider name (e.g., "postgresql", "mock", "firestore")
	Name() string

	// Initialize sets up the provider with the given configuration
	Initialize(config *dbpb.DatabaseProviderConfig) error

	// CreateRepository creates a single repository by entity name
	// This method enables metadata-driven repository creation by mapping
	// entity names to their corresponding repository constructors.
	//
	// Parameters:
	//   - entityName: Repository entity name (e.g., "client", "product", "subscription")
	//   - conn: Database connection (type depends on provider: *sql.DB, *firestore.Client, etc.)
	//   - tableName: Table/collection name (can be business-type specific)
	//
	// Returns:
	//   - Repository instance (must be cast to appropriate interface type)
	//   - Error if entity name is unknown or repository creation fails
	//
	// This method is used by CreateRepositories to loop through metadata
	// and create all 40 repositories dynamically, eliminating boilerplate.
	CreateRepository(entityName string, conn any, tableName string) (any, error)

	// IsEnabled returns whether this provider is currently enabled
	IsEnabled() bool

	// HealthCheck verifies the provider's health status
	HealthCheck(ctx context.Context) error

	// GetConnection returns the underlying connection (for compatibility)
	GetConnection() any

	// Close cleans up provider resources
	Close() error
}

// RepositoryConfig interface provides configuration for repository creation
// This interface abstracts configuration access for different provider types
type RepositoryConfig interface {
	GetTableName(entityName string) string
	GetBusinessType() string
	GetProviderConfig() map[string]any
}

// ConcreteRepositoryConfig provides a concrete implementation of RepositoryConfig
type ConcreteRepositoryConfig struct {
	// TablePrefix is prepended to all table/collection names
	TablePrefix string

	// TableSuffix is appended to all table/collection names
	TableSuffix string

	// TableMappings provides explicit name overrides for specific entities
	// Key: entity type (e.g., "product", "user"), Value: actual table/collection name
	TableMappings map[string]string

	// SchemaName specifies the database schema (for PostgreSQL)
	SchemaName string

	// BusinessType specifies the business type (e.g., "education", "fitness_center")
	BusinessType string

	// ProviderConfig holds provider-specific configuration
	ProviderConfig map[string]any
}

// GetTableName resolves the final table/collection name for an entity
func (c ConcreteRepositoryConfig) GetTableName(entityType string) string {
	// Check for explicit mapping first
	if mapped, exists := c.TableMappings[entityType]; exists {
		return mapped
	}

	// Use prefix/suffix pattern
	tableName := c.TablePrefix + entityType + c.TableSuffix

	// Add schema prefix if specified (for PostgreSQL)
	if c.SchemaName != "" {
		return c.SchemaName + "." + tableName
	}

	return tableName
}

// GetBusinessType returns the business type
func (c ConcreteRepositoryConfig) GetBusinessType() string {
	return c.BusinessType
}

// GetProviderConfig returns the provider-specific configuration
func (c ConcreteRepositoryConfig) GetProviderConfig() map[string]any {
	return c.ProviderConfig
}
