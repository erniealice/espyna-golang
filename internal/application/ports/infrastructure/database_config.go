package infrastructure

import (
	"fmt"

	pb "leapfor.xyz/esqyma/golang/v1/infrastructure/database"
)

// DatabaseConfigAdapter provides helpers to convert between map[string]any config
// and proto DatabaseProviderConfig
//
// This adapter serves as the bridge between the application configuration layer
// (which loads from environment variables/files as map[string]any) and the
// database providers (which use strongly-typed proto contracts).
//
// Benefits of this approach:
// - Configuration remains flexible (can load from various sources)
// - Database providers get type-safe proto configs
// - Clear separation between config loading and business logic
// - Easy to validate and transform configuration
type DatabaseConfigAdapter struct{}

// NewDatabaseConfigAdapter creates a new database config adapter
func NewDatabaseConfigAdapter() *DatabaseConfigAdapter {
	return &DatabaseConfigAdapter{}
}

// ConvertMapToProtoConfig converts map[string]any config to proto DatabaseProviderConfig
// This allows ProviderManager to pass map configs while adapters use proto types
func (a *DatabaseConfigAdapter) ConvertMapToProtoConfig(
	providerName string,
	config map[string]any,
) (*pb.DatabaseProviderConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Determine provider type and convert
	switch providerName {
	case "postgresql", "postgres":
		return a.convertPostgreSQLConfig(config)
	case "firestore":
		return a.convertFirestoreConfig(config)
	case "mock", "mock_db":
		return a.convertMockConfig(config)
	case "mongodb", "mongo":
		return a.convertMongoDBConfig(config)
	case "mysql":
		return a.convertMySQLConfig(config)
	case "sqlite":
		return a.convertSQLiteConfig(config)
	default:
		return nil, fmt.Errorf("unknown database provider: %s", providerName)
	}
}

// convertPostgreSQLConfig converts PostgreSQL map config to proto
func (a *DatabaseConfigAdapter) convertPostgreSQLConfig(config map[string]any) (*pb.DatabaseProviderConfig, error) {
	pgConfig := &pb.PostgreSQLConfig{
		Host:                         getString(config, "host", "localhost"),
		Port:                         getString(config, "port", "5432"),
		Database:                     getString(config, "name", ""),
		Username:                     getString(config, "user", ""),
		Password:                     getString(config, "password", ""),
		SslMode:                      getString(config, "ssl_mode", "disable"),
		SslCertPath:                  getString(config, "ssl_cert_path", ""),
		SslKeyPath:                   getString(config, "ssl_key_path", ""),
		SslRootCertPath:              getString(config, "ssl_root_cert_path", ""),
		MaxConnections:               getInt32(config, "max_connections", 25),
		MaxIdleConnections:           getInt32(config, "max_idle_connections", 5),
		ConnectionMaxLifetimeSeconds: getInt32(config, "connection_max_lifetime_seconds", 3600),
		ConnectionMaxIdleTimeSeconds: getInt32(config, "connection_max_idle_time_seconds", 600),
		MigrationsPath:               getString(config, "migrations_path", ""),
		Schema:                       getString(config, "schema", "public"),
		AutoMigrate:                  getBool(config, "auto_migrate", false),
		StatementTimeoutSeconds:      getInt32(config, "statement_timeout_seconds", 30),
		Timezone:                     getString(config, "timezone", "UTC"),
		LogQueries:                   getBool(config, "log_queries", false),
		LogSlowQueries:               getBool(config, "log_slow_queries", false),
		SlowQueryThresholdMs:         getInt32(config, "slow_query_threshold_ms", 1000),
	}

	// Validation
	if pgConfig.Database == "" {
		return nil, fmt.Errorf("postgresql config: database name is required")
	}
	if pgConfig.Username == "" {
		return nil, fmt.Errorf("postgresql config: username is required")
	}

	return &pb.DatabaseProviderConfig{
		Provider: pb.DatabaseProvider_DATABASE_PROVIDER_POSTGRESQL,
		Enabled:  getBool(config, "enabled", true),
		Config: &pb.DatabaseProviderConfig_Postgresql{
			Postgresql: pgConfig,
		},
	}, nil
}

// convertFirestoreConfig converts Firestore map config to proto
func (a *DatabaseConfigAdapter) convertFirestoreConfig(config map[string]any) (*pb.DatabaseProviderConfig, error) {
	firestoreConfig := &pb.FirestoreConfig{
		ProjectId:                getString(config, "project_id", ""),
		DatabaseId:               getString(config, "database_id", "(default)"),
		CredentialsPath:          getString(config, "credentials_path", ""),
		UseServiceAccountJson:    getBool(config, "use_service_account_json", false),
		UseEmulator:              getBool(config, "use_emulator", false),
		EmulatorHost:             getString(config, "emulator_host", ""),
		MaxPoolSize:              getInt32(config, "max_pool_size", 100),
		MaxIdleConnections:       getInt32(config, "max_idle_connections", 10),
		ConnectionTimeoutSeconds: getInt32(config, "connection_timeout_seconds", 30),
		RequestTimeoutSeconds:    getInt32(config, "request_timeout_seconds", 30),
		LogRequests:              getBool(config, "log_requests", false),
	}

	// Validation
	if firestoreConfig.ProjectId == "" {
		return nil, fmt.Errorf("firestore config: project_id is required")
	}

	return &pb.DatabaseProviderConfig{
		Provider: pb.DatabaseProvider_DATABASE_PROVIDER_FIRESTORE,
		Enabled:  getBool(config, "enabled", true),
		Config: &pb.DatabaseProviderConfig_Firestore{
			Firestore: firestoreConfig,
		},
	}, nil
}

// convertMockConfig converts Mock map config to proto
func (a *DatabaseConfigAdapter) convertMockConfig(config map[string]any) (*pb.DatabaseProviderConfig, error) {
	mockConfig := &pb.MockConfig{
		Name:               getString(config, "name", "mock"),
		SimulateFailures:   getBool(config, "simulate_failures", false),
		FailureRatePercent: getInt32(config, "failure_rate_percent", 0),
		InitialDataPath:    getString(config, "initial_data_path", ""),
		LatencyMs:          getInt32(config, "latency_ms", 0),
		LatencyVarianceMs:  getInt32(config, "latency_variance_ms", 0),
		LogOperations:      getBool(config, "log_operations", false),
		Verbose:            getBool(config, "verbose", false),
	}

	// Handle initial_data map
	if initialData, ok := config["initial_data"].(map[string]any); ok {
		mockConfig.InitialData = make(map[string]string)
		for k, v := range initialData {
			if str, ok := v.(string); ok {
				mockConfig.InitialData[k] = str
			}
		}
	}

	return &pb.DatabaseProviderConfig{
		Provider: pb.DatabaseProvider_DATABASE_PROVIDER_MOCK,
		Enabled:  getBool(config, "enabled", true),
		Config: &pb.DatabaseProviderConfig_Mock{
			Mock: mockConfig,
		},
	}, nil
}

// convertMongoDBConfig converts MongoDB map config to proto
func (a *DatabaseConfigAdapter) convertMongoDBConfig(config map[string]any) (*pb.DatabaseProviderConfig, error) {
	mongoConfig := &pb.MongoDBConfig{
		ConnectionString:              getString(config, "connection_string", ""),
		Database:                      getString(config, "database", ""),
		MaxPoolSize:                   getInt32(config, "max_pool_size", 100),
		MinPoolSize:                   getInt32(config, "min_pool_size", 10),
		ConnectionTimeoutSeconds:      getInt32(config, "connection_timeout_seconds", 30),
		SocketTimeoutSeconds:          getInt32(config, "socket_timeout_seconds", 30),
		ServerSelectionTimeoutSeconds: getInt32(config, "server_selection_timeout_seconds", 30),
		ReadPreference:                getString(config, "read_preference", "primary"),
		WriteConcern:                  getString(config, "write_concern", "majority"),
		LogCommands:                   getBool(config, "log_commands", false),
	}

	// Validation
	if mongoConfig.ConnectionString == "" {
		return nil, fmt.Errorf("mongodb config: connection_string is required")
	}
	if mongoConfig.Database == "" {
		return nil, fmt.Errorf("mongodb config: database is required")
	}

	return &pb.DatabaseProviderConfig{
		Provider: pb.DatabaseProvider_DATABASE_PROVIDER_MONGODB,
		Enabled:  getBool(config, "enabled", true),
		Config: &pb.DatabaseProviderConfig_Mongodb{
			Mongodb: mongoConfig,
		},
	}, nil
}

// convertMySQLConfig converts MySQL map config to proto
func (a *DatabaseConfigAdapter) convertMySQLConfig(config map[string]any) (*pb.DatabaseProviderConfig, error) {
	mysqlConfig := &pb.MySQLConfig{
		Host:                         getString(config, "host", "localhost"),
		Port:                         getString(config, "port", "3306"),
		Database:                     getString(config, "database", ""),
		Username:                     getString(config, "username", ""),
		Password:                     getString(config, "password", ""),
		EnableTls:                    getBool(config, "enable_tls", false),
		TlsCertPath:                  getString(config, "tls_cert_path", ""),
		TlsKeyPath:                   getString(config, "tls_key_path", ""),
		TlsCaPath:                    getString(config, "tls_ca_path", ""),
		MaxConnections:               getInt32(config, "max_connections", 25),
		MaxIdleConnections:           getInt32(config, "max_idle_connections", 5),
		ConnectionMaxLifetimeSeconds: getInt32(config, "connection_max_lifetime_seconds", 3600),
		Charset:                      getString(config, "charset", "utf8mb4"),
		Collation:                    getString(config, "collation", "utf8mb4_unicode_ci"),
		Timezone:                     getString(config, "timezone", "UTC"),
		LogQueries:                   getBool(config, "log_queries", false),
	}

	// Validation
	if mysqlConfig.Database == "" {
		return nil, fmt.Errorf("mysql config: database is required")
	}
	if mysqlConfig.Username == "" {
		return nil, fmt.Errorf("mysql config: username is required")
	}

	return &pb.DatabaseProviderConfig{
		Provider: pb.DatabaseProvider_DATABASE_PROVIDER_MYSQL,
		Enabled:  getBool(config, "enabled", true),
		Config: &pb.DatabaseProviderConfig_Mysql{
			Mysql: mysqlConfig,
		},
	}, nil
}

// convertSQLiteConfig converts SQLite map config to proto
func (a *DatabaseConfigAdapter) convertSQLiteConfig(config map[string]any) (*pb.DatabaseProviderConfig, error) {
	sqliteConfig := &pb.SQLiteConfig{
		FilePath:           getString(config, "file_path", ""),
		InMemory:           getBool(config, "in_memory", false),
		JournalMode:        getString(config, "journal_mode", "WAL"),
		Synchronous:        getString(config, "synchronous", "NORMAL"),
		CacheSizeKb:        getInt32(config, "cache_size_kb", 2000),
		PageSizeBytes:      getInt32(config, "page_size_bytes", 4096),
		MaxOpenConnections: getInt32(config, "max_open_connections", 1),
		EnableForeignKeys:  getBool(config, "enable_foreign_keys", true),
		EnableTriggers:     getBool(config, "enable_triggers", true),
		MigrationsPath:     getString(config, "migrations_path", ""),
		AutoMigrate:        getBool(config, "auto_migrate", false),
		LogQueries:         getBool(config, "log_queries", false),
	}

	// Validation
	if !sqliteConfig.InMemory && sqliteConfig.FilePath == "" {
		return nil, fmt.Errorf("sqlite config: file_path is required when not using in_memory mode")
	}

	return &pb.DatabaseProviderConfig{
		Provider: pb.DatabaseProvider_DATABASE_PROVIDER_SQLITE,
		Enabled:  getBool(config, "enabled", true),
		Config: &pb.DatabaseProviderConfig_Sqlite{
			Sqlite: sqliteConfig,
		},
	}, nil
}

// Helper functions getString, getBool, getInt32 are defined in helpers.go
