//go:build postgres

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	dbpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/database"
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
}

// buildPgTableConfig creates table config from POSTGRES_TABLE_* environment variables.
// This allows PostgreSQL-specific table naming without the container knowing about it.
func buildPgTableConfig() *registry.DatabaseTableConfig {
	prefix := getEnv("POSTGRES_TABLE_PREFIX", "")
	return &registry.DatabaseTableConfig{
		// Common
		Attribute: prefix + getPostgresTableEnv("ATTRIBUTE", "attribute"),
		// Entity
		Client:            prefix + getPostgresTableEnv("CLIENT", "client"),
		ClientAttribute:   prefix + getPostgresTableEnv("CLIENT_ATTRIBUTE", "client_attribute"),
		Admin:             prefix + getPostgresTableEnv("ADMIN", "admin"),
		Manager:           prefix + getPostgresTableEnv("MANAGER", "manager"),
		Staff:             prefix + getPostgresTableEnv("STAFF", "staff"),
		StaffAttribute:    prefix + getPostgresTableEnv("STAFF_ATTRIBUTE", "staff_attribute"),
		Delegate:          prefix + getPostgresTableEnv("DELEGATE", "delegate"),
		DelegateAttribute: prefix + getPostgresTableEnv("DELEGATE_ATTRIBUTE", "delegate_attribute"),
		DelegateClient:    prefix + getPostgresTableEnv("DELEGATE_CLIENT", "delegate_client"),
		Group:             prefix + getPostgresTableEnv("GROUP", "group"),
		GroupAttribute:    prefix + getPostgresTableEnv("GROUP_ATTRIBUTE", "group_attribute"),
		Location:          prefix + getPostgresTableEnv("LOCATION", "location"),
		LocationAttribute: prefix + getPostgresTableEnv("LOCATION_ATTRIBUTE", "location_attribute"),
		Permission:        prefix + getPostgresTableEnv("PERMISSION", "permission"),
		Role:              prefix + getPostgresTableEnv("ROLE", "role"),
		RolePermission:    prefix + getPostgresTableEnv("ROLE_PERMISSION", "role_permission"),
		User:              prefix + getPostgresTableEnv("USER", "user"),
		Workspace:         prefix + getPostgresTableEnv("WORKSPACE", "workspace"),
		WorkspaceClient:   prefix + getPostgresTableEnv("WORKSPACE_CLIENT", "workspace_client"),
		WorkspaceUser:     prefix + getPostgresTableEnv("WORKSPACE_USER", "workspace_user"),
		WorkspaceUserRole: prefix + getPostgresTableEnv("WORKSPACE_USER_ROLE", "workspace_user_role"),
		// Event
		Event:          prefix + getPostgresTableEnv("EVENT", "event"),
		EventAttribute: prefix + getPostgresTableEnv("EVENT_ATTRIBUTE", "event_attribute"),
		EventClient:    prefix + getPostgresTableEnv("EVENT_CLIENT", "event_client"),
		EventProduct:   prefix + getPostgresTableEnv("EVENT_PRODUCT", "event_product"),
		EventSettings:  prefix + getPostgresTableEnv("EVENT_SETTINGS", "event_settings"),
		// Framework
		Framework: prefix + getPostgresTableEnv("FRAMEWORK", "framework"),
		Objective: prefix + getPostgresTableEnv("OBJECTIVE", "objective"),
		Task:      prefix + getPostgresTableEnv("TASK", "task"),
		// Payment
		Payment:                     prefix + getPostgresTableEnv("PAYMENT", "payment"),
		PaymentAttribute:            prefix + getPostgresTableEnv("PAYMENT_ATTRIBUTE", "payment_attribute"),
		PaymentMethod:               prefix + getPostgresTableEnv("PAYMENT_METHOD", "payment_method"),
		PaymentProfile:              prefix + getPostgresTableEnv("PAYMENT_PROFILE", "payment_profile"),
		PaymentProfilePaymentMethod: prefix + getPostgresTableEnv("PAYMENT_PROFILE_PAYMENT_METHOD", "payment_profile_payment_method"),
		// Product
		Product:             prefix + getPostgresTableEnv("PRODUCT", "product"),
		Collection:          prefix + getPostgresTableEnv("COLLECTION", "collection"),
		CollectionAttribute: prefix + getPostgresTableEnv("COLLECTION_ATTRIBUTE", "collection_attribute"),
		CollectionParent:    prefix + getPostgresTableEnv("COLLECTION_PARENT", "collection_parent"),
		CollectionPlan:      prefix + getPostgresTableEnv("COLLECTION_PLAN", "collection_plan"),
		PriceProduct:        prefix + getPostgresTableEnv("PRICE_PRODUCT", "price_product"),
		ProductAttribute:    prefix + getPostgresTableEnv("PRODUCT_ATTRIBUTE", "product_attribute"),
		ProductCollection:   prefix + getPostgresTableEnv("PRODUCT_COLLECTION", "product_collection"),
		ProductPlan:         prefix + getPostgresTableEnv("PRODUCT_PLAN", "product_plan"),
		Resource:            prefix + getPostgresTableEnv("RESOURCE", "resource"),
		// Record
		Record: prefix + getPostgresTableEnv("RECORD", "record"),
		// Workflow
		Workflow:         prefix + getPostgresTableEnv("WORKFLOW", "workflow"),
		WorkflowTemplate: prefix + getPostgresTableEnv("WORKFLOW_TEMPLATE", "workflow_template"),
		Stage:            prefix + getPostgresTableEnv("STAGE", "stage"),
		Activity:         prefix + getPostgresTableEnv("ACTIVITY", "activity"),
		StageTemplate:    prefix + getPostgresTableEnv("STAGE_TEMPLATE", "stage_template"),
		ActivityTemplate: prefix + getPostgresTableEnv("ACTIVITY_TEMPLATE", "activity_template"),
		// Session
		Session: prefix + getPostgresTableEnv("SESSION", "session"),
		// Subscription
		Plan:                  prefix + getPostgresTableEnv("PLAN", "plan"),
		PlanAttribute:         prefix + getPostgresTableEnv("PLAN_ATTRIBUTE", "plan_attribute"),
		PlanLocation:          prefix + getPostgresTableEnv("PLAN_LOCATION", "plan_location"),
		PlanSettings:          prefix + getPostgresTableEnv("PLAN_SETTINGS", "plan_settings"),
		Balance:               prefix + getPostgresTableEnv("BALANCE", "balance"),
		BalanceAttribute:      prefix + getPostgresTableEnv("BALANCE_ATTRIBUTE", "balance_attribute"),
		Invoice:               prefix + getPostgresTableEnv("INVOICE", "invoice"),
		InvoiceAttribute:      prefix + getPostgresTableEnv("INVOICE_ATTRIBUTE", "invoice_attribute"),
		PricePlan:             prefix + getPostgresTableEnv("PRICE_PLAN", "price_plan"),
		Subscription:          prefix + getPostgresTableEnv("SUBSCRIPTION", "subscription"),
		SubscriptionAttribute: prefix + getPostgresTableEnv("SUBSCRIPTION_ATTRIBUTE", "subscription_attribute"),
	}
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

	connStr := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		pgConfig.Host, pgConfig.Port, pgConfig.Name,
		pgConfig.User, pgConfig.Password, pgConfig.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	db.SetMaxOpenConns(pgConfig.MaxConns)
	db.SetMaxIdleConns(pgConfig.MaxConns / 2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	a.db = db
	a.enabled = config.Enabled
	a.connected = true

	log.Printf("✅ PostgreSQL adapter connected to %s:%s/%s", pgConfig.Host, pgConfig.Port, pgConfig.Name)
	return nil
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
var _ ports.RepositoryProvider = (*PostgresAdapter)(nil)
