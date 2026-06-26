package infrastructure

import (
	"fmt"
	"strings"
)

// =============================================================================
// DATABASE CONFIGURATION TYPES
// =============================================================================

// DatabaseConfig is a union type that can hold any database configuration
type DatabaseConfig struct {
	Postgres  *PostgresDatabaseConfig
	Firestore *FirestoreDatabaseConfig
	Mock      *MockDatabaseConfig
}

// PostgresDatabaseConfig defines configuration for PostgreSQL database
type PostgresDatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	URL      string `json:"url,omitempty"`
	SSLMode  string `json:"ssl_mode"`
}

// Validate validates the postgres configuration
func (c PostgresDatabaseConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("postgres host is required")
	}
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.User == "" {
		return fmt.Errorf("postgres user is required")
	}
	if c.Name == "" {
		c.Name = "espyna"
	}
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	return nil
}

// ToMap converts the config to a map for provider initialization
func (c PostgresDatabaseConfig) ToMap() map[string]any {
	return map[string]any{
		"host":     c.Host,
		"port":     c.Port,
		"name":     c.Name,
		"user":     c.User,
		"password": c.Password,
		"url":      c.URL,
		"ssl_mode": c.SSLMode,
	}
}

// FirestoreDatabaseConfig defines configuration for Firestore database
type FirestoreDatabaseConfig struct {
	ProjectID       string `json:"project_id"`
	CredentialsPath string `json:"credentials_path,omitempty"`
	DatabaseID      string `json:"database_id,omitempty"`
}

// Validate validates the firestore configuration
func (c FirestoreDatabaseConfig) Validate() error {
	if c.ProjectID == "" {
		return fmt.Errorf("firestore project ID is required")
	}
	return nil
}

// ToMap converts the config to a map for provider initialization
func (c FirestoreDatabaseConfig) ToMap() map[string]any {
	return map[string]any{
		"project_id":       c.ProjectID,
		"credentials_path": c.CredentialsPath,
		"database_id":      c.DatabaseID,
	}
}

// MockDatabaseConfig defines configuration for mock database
type MockDatabaseConfig struct {
	DefaultBusinessType string `json:"default_business_type"`
}

// Validate validates the mock database configuration
func (c MockDatabaseConfig) Validate() error {
	if c.DefaultBusinessType == "" {
		c.DefaultBusinessType = "education"
	}
	return nil
}

// ToMap converts the config to a map for provider initialization
func (c MockDatabaseConfig) ToMap() map[string]any {
	return map[string]any{
		"default_business_type": c.DefaultBusinessType,
	}
}

// =============================================================================
// ENVIRONMENT CONFIGURATION LOADERS
// =============================================================================

func createPostgresConfigFromEnv() PostgresDatabaseConfig {
	return PostgresDatabaseConfig{
		Host:     GetEnv("DATABASE_POSTGRES_HOST", "localhost"),
		Port:     ParseInt(GetEnv("DATABASE_POSTGRES_PORT", "5432")),
		Name:     GetEnv("DATABASE_POSTGRES_DBNAME", "espyna"),
		User:     GetEnv("DATABASE_POSTGRES_USER", "postgres"),
		Password: GetEnv("DATABASE_POSTGRES_PASSWORD", ""),
		URL:      GetEnv("DATABASE_POSTGRES_URL", ""),
		SSLMode:  GetEnv("DATABASE_POSTGRES_SSLMODE", "disable"),
	}
}

func createFirestoreConfigFromEnv() FirestoreDatabaseConfig {
	return FirestoreDatabaseConfig{
		ProjectID:       GetEnv("DATABASE_FIRESTORE_PROJECT_ID", ""),
		CredentialsPath: GetEnv("DATABASE_FIRESTORE_CREDENTIALS_FILE", ""),
		DatabaseID:      GetEnv("DATABASE_FIRESTORE_DATABASE", ""),
	}
}

func createMockConfigFromEnv() MockDatabaseConfig {
	return MockDatabaseConfig{
		DefaultBusinessType: GetEnv("BUSINESS_TYPE", "education"),
	}
}

// =============================================================================
// DATABASE PROVIDER OPTIONS
// =============================================================================

// WithDatabaseFromEnv dynamically selects database provider based on CONFIG_DATABASE_PROVIDER
func WithDatabaseFromEnv() ContainerOption {
	return func(c Container) error {
		dbProvider := strings.ToLower(GetEnv("CONFIG_DATABASE_PROVIDER", "mock_db"))

		switch dbProvider {
		case "postgresql":
			return WithPostgresDatabase(createPostgresConfigFromEnv())(c)
		case "firestore":
			return WithFirestoreDatabase(createFirestoreConfigFromEnv())(c)
		case "mock_db", "":
			return WithMockDatabase(createMockConfigFromEnv())(c)
		default:
			return fmt.Errorf("unsupported database provider: %s", dbProvider)
		}
	}
}

// WithPostgresDatabase configures PostgreSQL as the database provider
func WithPostgresDatabase(config PostgresDatabaseConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid postgres configuration: %w", err)
		}

		if setter, ok := c.(DatabaseConfigSetter); ok {
			setter.SetDatabaseConfig(DatabaseConfig{Postgres: &config})
		} else {
			return fmt.Errorf("container does not implement SetDatabaseConfig method")
		}

		fmt.Printf("🐘 Configured PostgreSQL database: %s:%d\n", config.Host, config.Port)
		return nil
	}
}

// WithFirestoreDatabase configures Firestore as the database provider
func WithFirestoreDatabase(config FirestoreDatabaseConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid firestore configuration: %w", err)
		}

		if setter, ok := c.(DatabaseConfigSetter); ok {
			setter.SetDatabaseConfig(DatabaseConfig{Firestore: &config})
		} else {
			return fmt.Errorf("container does not implement SetDatabaseConfig method")
		}

		fmt.Printf("🔥 Configured Firestore database: %s\n", config.ProjectID)
		return nil
	}
}

// WithMockDatabase configures mock database for testing/development
func WithMockDatabase(config MockDatabaseConfig) ContainerOption {
	return func(c Container) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid mock database configuration: %w", err)
		}

		if setter, ok := c.(DatabaseConfigSetter); ok {
			setter.SetDatabaseConfig(DatabaseConfig{Mock: &config})
		} else {
			return fmt.Errorf("container does not implement SetDatabaseConfig method")
		}

		fmt.Printf("🧪 Configured mock database: %s\n", config.DefaultBusinessType)
		return nil
	}
}
