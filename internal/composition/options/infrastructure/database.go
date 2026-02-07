package infrastructure

import (
	"fmt"
	"strings"

	appConfig "leapfor.xyz/espyna/internal/composition/config"
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
		Host:     GetEnv("POSTGRES_HOST", "localhost"),
		Port:     ParseInt(GetEnv("POSTGRES_PORT", "5432")),
		Name:     GetEnv("POSTGRES_NAME", "espyna"),
		User:     GetEnv("POSTGRES_USER", "postgres"),
		Password: GetEnv("POSTGRES_PASSWORD", ""),
		URL:      GetEnv("POSTGRES_URL", ""),
		SSLMode:  GetEnv("POSTGRES_SSL_MODE", "disable"),
	}
}

func createFirestoreConfigFromEnv() FirestoreDatabaseConfig {
	return FirestoreDatabaseConfig{
		ProjectID:       GetEnv("FIRESTORE_PROJECT_ID", ""),
		CredentialsPath: GetEnv("FIRESTORE_CREDENTIALS_PATH", ""),
		DatabaseID:      GetEnv("FIRESTORE_DATABASE", ""),
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
		case "postgres":
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

		// Load and set database table configuration from environment
		tableConfig := createTableConfig("POSTGRES")
		if tableSetter, ok := c.(DatabaseTableConfigSetter); ok {
			tableSetter.SetDatabaseTableConfig(&tableConfig)
		} else {
			return fmt.Errorf("container does not implement SetDatabaseTableConfig method")
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

		tableConfig := createTableConfig("FIRESTORE")
		if tableSetter, ok := c.(DatabaseTableConfigSetter); ok {
			tableSetter.SetDatabaseTableConfig(&tableConfig)
		} else {
			return fmt.Errorf("container does not implement SetDatabaseTableConfig method")
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

// WithDatabaseTableConfig sets the database table configuration
func WithDatabaseTableConfig(config appConfig.DatabaseTableConfig) ContainerOption {
	return func(c Container) error {
		if setter, ok := c.(DatabaseTableConfigSetter); ok {
			setter.SetDatabaseTableConfig(&config)
		} else {
			return fmt.Errorf("container does not implement SetDatabaseTableConfig method")
		}
		return nil
	}
}

// createTableConfig creates table/collection configuration for a specific database type
func createTableConfig(dbType string) appConfig.DatabaseTableConfig {
	var prefix string
	switch dbType {
	case "POSTGRES":
		prefix = "LEAPFOR_DATABASE_POSTGRES_TABLE_"
	case "FIRESTORE":
		prefix = "LEAPFOR_DATABASE_FIRESTORE_COLLECTION_"
	default:
		prefix = "LEAPFOR_DATABASE_DEFAULT_"
	}

	return appConfig.DatabaseTableConfig{
		// Common tables/collections
		Attribute: GetEnv(prefix+"ATTRIBUTE", "attribute"),

		// Entity tables/collections
		Client:            GetEnv(prefix+"CLIENT", "client"),
		ClientAttribute:   GetEnv(prefix+"CLIENT_ATTRIBUTE", "client_attribute"),
		Admin:             GetEnv(prefix+"ADMIN", "admin"),
		Manager:           GetEnv(prefix+"MANAGER", "manager"),
		Staff:             GetEnv(prefix+"STAFF", "staff"),
		StaffAttribute:    GetEnv(prefix+"STAFF_ATTRIBUTE", "staff_attribute"),
		Delegate:          GetEnv(prefix+"DELEGATE", "delegate"),
		DelegateAttribute: GetEnv(prefix+"DELEGATE_ATTRIBUTE", "delegate_attribute"),
		DelegateClient:    GetEnv(prefix+"DELEGATE_CLIENT", "delegate_client"),
		Group:             GetEnv(prefix+"GROUP", "group"),
		GroupAttribute:    GetEnv(prefix+"GROUP_ATTRIBUTE", "group_attribute"),
		Location:          GetEnv(prefix+"LOCATION", "location"),
		LocationAttribute: GetEnv(prefix+"LOCATION_ATTRIBUTE", "location_attribute"),
		Permission:        GetEnv(prefix+"PERMISSION", "permission"),
		Role:              GetEnv(prefix+"ROLE", "role"),
		RolePermission:    GetEnv(prefix+"ROLE_PERMISSION", "role_permission"),
		User:              GetEnv(prefix+"USER", "user"),
		Workspace:         GetEnv(prefix+"WORKSPACE", "workspace"),
		WorkspaceClient:   GetEnv(prefix+"WORKSPACE_CLIENT", "workspace_client"),
		WorkspaceUser:     GetEnv(prefix+"WORKSPACE_USER", "workspace_user"),
		WorkspaceUserRole: GetEnv(prefix+"WORKSPACE_USER_ROLE", "workspace_user_role"),

		// Event tables/collections
		Event:          GetEnv(prefix+"EVENT", "event"),
		EventAttribute: GetEnv(prefix+"EVENT_ATTRIBUTE", "event_attribute"),
		EventClient:    GetEnv(prefix+"EVENT_CLIENT", "event_client"),
		EventProduct:   GetEnv(prefix+"EVENT_PRODUCT", "event_product"),
		EventSettings:  GetEnv(prefix+"EVENT_SETTINGS", "event_settings"),

		// Framework tables/collections
		Framework: GetEnv(prefix+"FRAMEWORK", "framework"),
		Objective: GetEnv(prefix+"OBJECTIVE", "objective"),
		Task:      GetEnv(prefix+"TASK", "task"),

		// Payment tables/collections
		Payment:                     GetEnv(prefix+"PAYMENT", "payment"),
		PaymentAttribute:            GetEnv(prefix+"PAYMENT_ATTRIBUTE", "payment_attribute"),
		PaymentMethod:               GetEnv(prefix+"PAYMENT_METHOD", "payment_method"),
		PaymentProfile:              GetEnv(prefix+"PAYMENT_PROFILE", "payment_profile"),
		PaymentProfilePaymentMethod: GetEnv(prefix+"PAYMENT_PROFILE_PAYMENT_METHOD", "payment_profile_payment_method"),

		// Product tables/collections
		Product:             GetEnv(prefix+"PRODUCT", "product"),
		Collection:          GetEnv(prefix+"COLLECTION", "collection"),
		CollectionAttribute: GetEnv(prefix+"COLLECTION_ATTRIBUTE", "collection_attribute"),
		CollectionParent:    GetEnv(prefix+"COLLECTION_PARENT", "collection_parent"),
		CollectionPlan:      GetEnv(prefix+"COLLECTION_PLAN", "collection_plan"),
		PriceProduct:        GetEnv(prefix+"PRICE_PRODUCT", "price_product"),
		ProductAttribute:    GetEnv(prefix+"PRODUCT_ATTRIBUTE", "product_attribute"),
		ProductCollection:   GetEnv(prefix+"PRODUCT_COLLECTION", "product_collection"),
		ProductPlan:         GetEnv(prefix+"PRODUCT_PLAN", "product_plan"),
		Resource:            GetEnv(prefix+"RESOURCE", "resource"),

		// Record tables/collections
		Record: GetEnv(prefix+"RECORD", "record"),

		// Workflow tables/collections
		Workflow:         GetEnv(prefix+"WORKFLOW", "workflow"),
		WorkflowTemplate: GetEnv(prefix+"WORKFLOW_TEMPLATE", "workflow_template"),
		Stage:            GetEnv(prefix+"STAGE", "stage"),
		Activity:         GetEnv(prefix+"ACTIVITY", "activity"),
		StageTemplate:    GetEnv(prefix+"STAGE_TEMPLATE", "stage_template"),
		ActivityTemplate: GetEnv(prefix+"ACTIVITY_TEMPLATE", "activity_template"),

		// Subscription tables/collections
		Plan:                  GetEnv(prefix+"PLAN", "plan"),
		PlanAttribute:         GetEnv(prefix+"PLAN_ATTRIBUTE", "plan_attribute"),
		PlanLocation:          GetEnv(prefix+"PLAN_LOCATION", "plan_location"),
		PlanSettings:          GetEnv(prefix+"PLAN_SETTINGS", "plan_settings"),
		Balance:               GetEnv(prefix+"BALANCE", "balance"),
		BalanceAttribute:      GetEnv(prefix+"BALANCE_ATTRIBUTE", "balance_attribute"),
		Invoice:               GetEnv(prefix+"INVOICE", "invoice"),
		InvoiceAttribute:      GetEnv(prefix+"INVOICE_ATTRIBUTE", "invoice_attribute"),
		PricePlan:             GetEnv(prefix+"PRICE_PLAN", "price_plan"),
		Subscription:          GetEnv(prefix+"SUBSCRIPTION", "subscription"),
		SubscriptionAttribute: GetEnv(prefix+"SUBSCRIPTION_ATTRIBUTE", "subscription_attribute"),
	}
}
