package infrastructure

import (
	"context"
	"time"

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

// PoolSizer is an optional capability for DatabaseProviders that maintain a
// bounded connection pool. Concurrency-sensitive callers (batch generators,
// fanout workers) type-assert their provider to this interface to clamp their
// parallelism against the available pool budget rather than picking an
// arbitrary fanout that could starve the pool.
//
// Providers without a meaningful pool concept (e.g., HTTP/gRPC clients like
// Firestore) simply do not implement it; callers should fall back to a
// conservative default when the assertion fails.
type PoolSizer interface {
	// MaxConns returns the configured maximum number of simultaneously open
	// connections in this provider's pool. Implementations should return the
	// effective value applied to the driver (e.g., *sql.DB.SetMaxOpenConns),
	// not the raw env value.
	MaxConns() int
}

// PoolStats is a driver-neutral snapshot of a bounded connection pool's
// health, mirroring database/sql.DBStats without leaking the driver type
// through the port.
//
// The load-bearing fields are WaitCount and WaitDuration: they grow only when
// a caller had to block waiting for a connection — i.e. when pool saturation
// was observed. Saturation says callers reached the configured cap, not why:
// an undersized cap, slow queries, lock contention, a slow server or a leaked
// transaction all produce it, so correlate with query/lock latency before
// raising a cap. A pool whose WaitCount stays ≈0 under peak load is not
// cap-bound.
//
// All counters are cumulative process totals (mirroring database/sql.DBStats).
// Note that WaitCount increments when a wait begins while WaitDuration is
// added when it ends, so a wait spanning two samples splits unevenly across
// intervals; the cumulative totals remain exact.
type PoolStats struct {
	MaxOpen int // pool cap (max simultaneously open connections)
	Open    int // currently established connections
	InUse   int // connections executing work right now
	Idle    int // warm connections awaiting reuse

	WaitCount    int64         // cumulative callers that blocked waiting for a connection
	WaitDuration time.Duration // cumulative time callers spent blocked

	MaxIdleClosed     int64 // connections closed because the idle count cap was exceeded
	MaxIdleTimeClosed int64 // connections pruned by the max idle-time policy
	MaxLifetimeClosed int64 // connections rotated out by the max lifetime policy
}

// PoolStatser is an optional capability for DatabaseProviders that maintain a
// bounded connection pool, sibling to PoolSizer. Observability surfaces
// (debug endpoints, health checks, load investigations) type-assert their
// provider to this interface; providers without a pool concept simply do not
// implement it.
type PoolStatser interface {
	// PoolStats returns a point-in-time snapshot of the pool. Implementations
	// must be safe to call concurrently with query traffic.
	PoolStats() PoolStats
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
