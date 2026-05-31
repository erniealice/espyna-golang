//go:build postgresql

// Package postgres is the PostgreSQL adapter's self-registration entry point.
//
// init() registers three things with the espyna registry:
//   - Provider factory (NewPostgresAdapter)
//   - BuildFromEnv builder (reads POSTGRES_* env vars, returns initialized adapter)
//   - TableConfigBuilder (buildPgTableConfig — scans POSTGRES_TABLE_* env vars via entityid.All)
//
// The 145+ entity adapters in subdirectories (entity/, product/, revenue/, etc.)
// each have their own init() that calls registry.RegisterRepositoryFactory to
// register a "postgresql:<entityid>" factory.
//
// Adding a new PostgreSQL entity adapter:
//  1. Create an adapter file in the appropriate subdomain directory.
//  2. Add an init() that calls registry.RegisterRepositoryFactory("postgresql", entityid.X, factory).
//  3. Blank-import the adapter package in the consumer binary so init() fires.
//
// Table name resolution: buildPgTableConfig() iterates entityid.All, checking
// POSTGRES_TABLE_{ENTITY} env vars for overrides, and stores them in TableConfig.
//
// Import order matters: adapter packages must be blank-imported in the consumer
// binary for their init() registrations to execute.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	dbpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/database"
	_ "github.com/lib/pq"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterDatabaseProvider(
		"postgresql",
		func() ports.DatabaseProvider {
			return NewPostgresAdapter()
		},
		transformConfig,
	)
	registry.RegisterDatabaseBuildFromEnv("postgresql", buildFromEnv)
	registry.RegisterDatabaseTableConfigBuilder("postgresql", buildPgTableConfig)
	// Plan 2 (reflectionless CRUD): register the boot-shot schema validator so the
	// dialect-neutral container can resolve and run it for the postgresql provider
	// without importing this postgresql-tagged package directly. Mirrors the
	// RegisterDatabaseTableConfigBuilder hook above.
	registry.RegisterSchemaValidator("postgresql", core.ValidateSchema)
}

// buildPgTableConfig creates table config from POSTGRES_TABLE_* environment variables.
// This allows PostgreSQL-specific table naming without the container knowing about it.
func buildPgTableConfig() *registry.TableConfig {
	prefix := getEnv("POSTGRES_TABLE_PREFIX", "")
	overrides := make(map[string]string)
	for _, entity := range entityid.All {
		envKey := "POSTGRES_TABLE_" + toEnvKey(entity)
		if val := os.Getenv(envKey); val != "" {
			overrides[entity] = val
		}
	}
	return registry.NewTableConfig(prefix, overrides)
}

// toEnvKey converts a snake_case entity ID to UPPER_CASE for env var lookup.
func toEnvKey(entity string) string {
	return strings.ToUpper(entity)
}

// getPostgresTableEnv reads POSTGRES_TABLE_{suffix} or returns the default.
func getPostgresTableEnv(suffix, defaultValue string) string {
	if value := os.Getenv("POSTGRES_TABLE_" + suffix); value != "" {
		return value
	}
	return defaultValue
}

// buildFromEnv creates and initializes a PostgreSQL adapter from environment variables.
func buildFromEnv() (ports.DatabaseProvider, error) {
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	name := getEnv("POSTGRES_NAME", "espyna")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "")
	sslMode := getEnv("POSTGRES_SSL_MODE", "disable")
	maxConns := getEnvInt("POSTGRES_MAX_CONNECTIONS", 25)

	if host == "" {
		return nil, fmt.Errorf("postgresql: POSTGRES_HOST is required")
	}
	if user == "" {
		return nil, fmt.Errorf("postgresql: POSTGRES_USER is required")
	}

	protoConfig := &dbpb.DatabaseProviderConfig{
		Provider: dbpb.DatabaseProvider_DATABASE_PROVIDER_POSTGRESQL,
		Enabled:  true,
		Config: &dbpb.DatabaseProviderConfig_Postgresql{
			Postgresql: &dbpb.PostgreSQLConfig{
				Host:           host,
				Port:           port,
				Database:       name,
				Username:       user,
				Password:       password,
				SslMode:        sslMode,
				MaxConnections: int32(maxConns),
			},
		},
	}

	adapter := NewPostgresAdapter()
	if err := adapter.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("postgresql: failed to initialize: %w", err)
	}
	return adapter, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// transformConfig converts raw config map to PostgreSQL proto config.
func transformConfig(rawConfig map[string]any) (*dbpb.DatabaseProviderConfig, error) {
	protoConfig := &dbpb.DatabaseProviderConfig{
		Provider: dbpb.DatabaseProvider_DATABASE_PROVIDER_POSTGRESQL,
		Enabled:  true,
	}

	pgConfig := &dbpb.PostgreSQLConfig{}

	if host, ok := rawConfig["host"].(string); ok && host != "" {
		pgConfig.Host = host
	} else {
		return nil, fmt.Errorf("postgresql: host is required")
	}

	switch p := rawConfig["port"].(type) {
	case int:
		pgConfig.Port = fmt.Sprintf("%d", p)
	case string:
		pgConfig.Port = p
	default:
		pgConfig.Port = "5432"
	}

	if name, ok := rawConfig["name"].(string); ok && name != "" {
		pgConfig.Database = name
	} else if name, ok := rawConfig["database"].(string); ok && name != "" {
		pgConfig.Database = name
	} else {
		return nil, fmt.Errorf("postgresql: name/database is required")
	}

	if user, ok := rawConfig["user"].(string); ok && user != "" {
		pgConfig.Username = user
	} else if user, ok := rawConfig["username"].(string); ok && user != "" {
		pgConfig.Username = user
	} else {
		return nil, fmt.Errorf("postgresql: user/username is required")
	}

	if password, ok := rawConfig["password"].(string); ok {
		pgConfig.Password = password
	} else {
		return nil, fmt.Errorf("postgresql: password is required")
	}

	if sslMode, ok := rawConfig["ssl_mode"].(string); ok && sslMode != "" {
		pgConfig.SslMode = sslMode
	} else {
		pgConfig.SslMode = "disable"
	}

	if maxConns, ok := rawConfig["max_connections"].(int); ok && maxConns > 0 {
		pgConfig.MaxConnections = int32(maxConns)
	}

	protoConfig.Config = &dbpb.DatabaseProviderConfig_Postgresql{
		Postgresql: pgConfig,
	}

	return protoConfig, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// PostgresAdapter implements DatabaseProvider and RepositoryProvider for PostgreSQL.
// This adapter follows the same self-registration pattern as Firestore/Mock.
type PostgresAdapter struct {
	db        *sql.DB
	config    *PostgresConfig
	enabled   bool
	connected bool
}

// PostgresConfig holds PostgreSQL-specific configuration.
type PostgresConfig struct {
	Host           string
	Port           string
	Name           string
	User           string
	Password       string
	SSLMode        string
	MaxConns       int
	MigrationsPath string
}

// NewPostgresAdapter creates a new PostgreSQL database adapter.
func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{
		enabled: true,
	}
}

// Name returns the provider name.
func (a *PostgresAdapter) Name() string {
	return "postgresql"
}

// Initialize sets up the PostgreSQL connection.
func (a *PostgresAdapter) Initialize(config *dbpb.DatabaseProviderConfig) error {
	pgProto := config.GetPostgresql()
	if pgProto == nil {
		return fmt.Errorf("postgresql adapter requires postgresql configuration")
	}

	pgConfig := &PostgresConfig{
		Host:           pgProto.Host,
		Port:           pgProto.Port,
		Name:           pgProto.Database,
		User:           pgProto.Username,
		Password:       pgProto.Password,
		SSLMode:        pgProto.SslMode,
		MaxConns:       int(pgProto.MaxConnections),
		MigrationsPath: "./migrations",
	}

	if pgConfig.SSLMode == "" {
		pgConfig.SSLMode = "disable"
	}
	if pgConfig.MaxConns <= 0 {
		pgConfig.MaxConns = 25
	}

	a.config = pgConfig

	connParts := []string{
		"host=" + pgConfig.Host,
		"port=" + pgConfig.Port,
		"dbname=" + pgConfig.Name,
		"user=" + pgConfig.User,
		"sslmode=" + pgConfig.SSLMode,
		"connect_timeout=5",
	}
	if pgConfig.Password != "" {
		connParts = append(connParts, "password="+pgConfig.Password)
	}
	connStr := strings.Join(connParts, " ")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Keep idle connections at ~1/5 of the open cap (floor-divided, min 1) so
	// bursty workloads don't hold every connection warm at all times while
	// still keeping enough hot for typical concurrent reads. The previous
	// idle == open setting kept the entire pool warm even at idle — fine for
	// steady traffic, wasteful otherwise.
	maxIdle := pgConfig.MaxConns / 5
	if maxIdle < 1 {
		maxIdle = 1
	}
	db.SetMaxOpenConns(pgConfig.MaxConns)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	a.db = db
	a.enabled = config.Enabled
	a.connected = true

	log.Printf("✅ PostgreSQL adapter connected to %s:%s/%s (pool max=%d idle=%d)",
		pgConfig.Host, pgConfig.Port, pgConfig.Name, pgConfig.MaxConns, maxIdle)
	return nil
}

// MaxConns returns the effective max-open-connections cap configured on the
// underlying *sql.DB pool. Implements the optional ports.PoolSizer capability
// so concurrency-sensitive callers can clamp their fanout to the pool budget.
//
// Returns 0 when Initialize has not been called or the adapter is in a zero
// state; callers should treat 0 as "unknown" and fall back to a conservative
// default rather than dividing by it.
func (a *PostgresAdapter) MaxConns() int {
	if a == nil || a.config == nil {
		return 0
	}
	return a.config.MaxConns
}

// GetConnection returns the PostgreSQL database connection.
func (a *PostgresAdapter) GetConnection() any {
	return a.db
}

// Close closes the PostgreSQL connection.
func (a *PostgresAdapter) Close() error {
	if a.db != nil {
		err := a.db.Close()
		a.db = nil
		a.connected = false
		if err != nil {
			return fmt.Errorf("failed to close PostgreSQL connection: %w", err)
		}
		log.Println("✅ PostgreSQL adapter closed")
	}
	return nil
}

// IsHealthy checks if the PostgreSQL connection is healthy.
func (a *PostgresAdapter) IsHealthy(ctx context.Context) error {
	if !a.enabled {
		return fmt.Errorf("postgresql adapter is disabled")
	}
	if a.db == nil {
		return fmt.Errorf("postgresql connection is nil")
	}
	if err := a.db.PingContext(ctx); err != nil {
		a.connected = false
		return fmt.Errorf("postgresql health check failed: %w", err)
	}
	a.connected = true
	return nil
}

// IsEnabled returns whether this adapter is currently enabled.
func (a *PostgresAdapter) IsEnabled() bool {
	return a.enabled
}

// =============================================================================
// RepositoryProvider Implementation - Delegates to Registry
// =============================================================================

// CreateRepository creates a repository by looking up the registered factory.
// This replaces the giant switch statement by delegating to self-registered factories.
func (a *PostgresAdapter) CreateRepository(entityName string, conn any, tableName string) (any, error) {
	return registry.CreateRepository("postgresql", entityName, conn, tableName)
}

// GetTransactionManager returns the PostgreSQL transaction manager.
func (a *PostgresAdapter) GetTransactionManager() interfaces.TransactionManager {
	if a.db == nil || !a.connected {
		return nil
	}
	return core.NewPostgreSQLTransactionManager(a.db)
}

// HealthCheck checks if the PostgreSQL adapter is healthy.
func (a *PostgresAdapter) HealthCheck(ctx context.Context) error {
	return a.IsHealthy(ctx)
}

// Compile-time interface checks
var _ ports.DatabaseProvider = (*PostgresAdapter)(nil)
var _ ports.PoolSizer = (*PostgresAdapter)(nil)
var _ ports.RepositoryProvider = (*PostgresAdapter)(nil)
