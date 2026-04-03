package firestore

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/erniealice/espyna-golang/contrib/google/internal/database/firestore/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry"
	dbpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/database"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterDatabaseProvider(
		"firestore",
		func() ports.DatabaseProvider {
			return NewFirestoreAdapter()
		},
		transformConfig,
	)
	registry.RegisterDatabaseBuildFromEnv("firestore", buildFromEnv)
	registry.RegisterDatabaseTableConfigBuilder("firestore", buildTableConfig)
}

// buildTableConfig creates table config from FIRESTORE_TABLE_* environment variables.
// This allows Firestore-specific collection naming without the container knowing about it.
func buildTableConfig() *registry.TableConfig {
	prefix := os.Getenv("FIRESTORE_TABLE_PREFIX")
	overrides := make(map[string]string)

	// Map of entityid constant → env var suffix → default value
	// Only entries with non-default env overrides will be added.
	entityEnvMap := map[string]string{
		// Common
		"attribute": "ATTRIBUTE",
		// Entity
		"client":              "CLIENT",
		"client_attribute":    "CLIENT_ATTRIBUTE",
		"admin":               "ADMIN",
		"manager":             "MANAGER",
		"staff":               "STAFF",
		"staff_attribute":     "STAFF_ATTRIBUTE",
		"delegate":            "DELEGATE",
		"delegate_attribute":  "DELEGATE_ATTRIBUTE",
		"delegate_client":     "DELEGATE_CLIENT",
		"group":               "GROUP",
		"group_attribute":     "GROUP_ATTRIBUTE",
		"location":            "LOCATION",
		"location_attribute":  "LOCATION_ATTRIBUTE",
		"permission":          "PERMISSION",
		"role":                "ROLE",
		"role_permission":     "ROLE_PERMISSION",
		"user":                "USER",
		"workspace":           "WORKSPACE",
		"workspace_client":    "WORKSPACE_CLIENT",
		"workspace_user":      "WORKSPACE_USER",
		"workspace_user_role": "WORKSPACE_USER_ROLE",
		// Event
		"event":           "EVENT",
		"event_attribute": "EVENT_ATTRIBUTE",
		"event_client":    "EVENT_CLIENT",
		"event_product":   "EVENT_PRODUCT",
		"event_settings":  "EVENT_SETTINGS",
		// Framework
		"framework": "FRAMEWORK",
		"objective": "OBJECTIVE",
		"task":      "TASK",
		// Payment
		"payment":                        "PAYMENT",
		"payment_attribute":              "PAYMENT_ATTRIBUTE",
		"payment_method":                 "PAYMENT_METHOD",
		"payment_profile":                "PAYMENT_PROFILE",
		"payment_profile_payment_method": "PAYMENT_PROFILE_PAYMENT_METHOD",
		// Integration
		"integration_payment": "INTEGRATION_PAYMENT",
		// Product
		"product":              "PRODUCT",
		"collection":           "COLLECTION",
		"collection_attribute": "COLLECTION_ATTRIBUTE",
		"collection_parent":    "COLLECTION_PARENT",
		"collection_plan":      "COLLECTION_PLAN",
		"price_product":        "PRICE_PRODUCT",
		"product_attribute":    "PRODUCT_ATTRIBUTE",
		"product_collection":   "PRODUCT_COLLECTION",
		"product_plan":         "PRODUCT_PLAN",
		"resource":             "RESOURCE",
		// Record
		"record": "RECORD",
		// Workflow
		"workflow":          "WORKFLOW",
		"workflow_template": "WORKFLOW_TEMPLATE",
		"stage":             "STAGE",
		"activity":          "ACTIVITY",
		"stage_template":    "STAGE_TEMPLATE",
		"activity_template": "ACTIVITY_TEMPLATE",
		// Subscription
		"plan":                   "PLAN",
		"plan_attribute":         "PLAN_ATTRIBUTE",
		"plan_location":          "PLAN_LOCATION",
		"plan_settings":          "PLAN_SETTINGS",
		"balance":                "BALANCE",
		"balance_attribute":      "BALANCE_ATTRIBUTE",
		"invoice":                "INVOICE",
		"invoice_attribute":      "INVOICE_ATTRIBUTE",
		"price_plan":             "PRICE_PLAN",
		"subscription":           "SUBSCRIPTION",
		"subscription_attribute": "SUBSCRIPTION_ATTRIBUTE",
	}

	for entity, envSuffix := range entityEnvMap {
		tableName := getFirestoreTableEnv(envSuffix, entity)
		if tableName != entity {
			overrides[entity] = tableName
		}
	}

	return registry.NewTableConfig(prefix, overrides)
}

// getFirestoreTableEnv reads collection name from environment variables.
// Checks in order: LEAPFOR_DATABASE_FIRESTORE_COLLECTION_{suffix} (legacy), FIRESTORE_TABLE_{suffix} (new)
func getFirestoreTableEnv(suffix, defaultValue string) string {
	// Check legacy env var first (used by existing apps like tph-unlock-golang-v2)
	if value := os.Getenv("LEAPFOR_DATABASE_FIRESTORE_COLLECTION_" + suffix); value != "" {
		return value
	}
	// Check new env var format
	if value := os.Getenv("FIRESTORE_TABLE_" + suffix); value != "" {
		return value
	}
	return defaultValue
}

// buildFromEnv creates and initializes a Firestore adapter from environment variables.
//
// Environment variables:
//   - FIRESTORE_PROJECT_ID (required)
//   - FIRESTORE_CREDENTIALS_PATH (optional, uses ADC if not set)
//   - FIRESTORE_DATABASE (optional, defaults to "(default)")
func buildFromEnv() (ports.DatabaseProvider, error) {
	projectID := os.Getenv("FIRESTORE_PROJECT_ID")
	credentialsPath := os.Getenv("FIRESTORE_CREDENTIALS_PATH")
	databaseID := os.Getenv("FIRESTORE_DATABASE")

	if projectID == "" {
		return nil, fmt.Errorf("firestore: FIRESTORE_PROJECT_ID is required")
	}

	protoConfig := &dbpb.DatabaseProviderConfig{
		Provider: dbpb.DatabaseProvider_DATABASE_PROVIDER_FIRESTORE,
		Enabled:  true,
		Config: &dbpb.DatabaseProviderConfig_Firestore{
			Firestore: &dbpb.FirestoreConfig{
				ProjectId:       projectID,
				CredentialsPath: credentialsPath,
				DatabaseId:      databaseID,
			},
		},
	}

	adapter := NewFirestoreAdapter()
	if err := adapter.Initialize(protoConfig); err != nil {
		return nil, fmt.Errorf("firestore: failed to initialize: %w", err)
	}
	return adapter, nil
}

// transformConfig converts raw config map to Firestore proto config.
func transformConfig(rawConfig map[string]any) (*dbpb.DatabaseProviderConfig, error) {
	protoConfig := &dbpb.DatabaseProviderConfig{
		Provider: dbpb.DatabaseProvider_DATABASE_PROVIDER_FIRESTORE,
		Enabled:  true,
	}

	fsConfig := &dbpb.FirestoreConfig{}

	if projectID, ok := rawConfig["project_id"].(string); ok && projectID != "" {
		fsConfig.ProjectId = projectID
	} else {
		return nil, fmt.Errorf("firestore: project_id is required")
	}

	if credPath, ok := rawConfig["credentials_path"].(string); ok {
		fsConfig.CredentialsPath = credPath
	}

	if dbID, ok := rawConfig["database_id"].(string); ok {
		fsConfig.DatabaseId = dbID
	}

	protoConfig.Config = &dbpb.DatabaseProviderConfig_Firestore{
		Firestore: fsConfig,
	}

	return protoConfig, nil
}

// =============================================================================
// Adapter Implementation
// =============================================================================

// FirestoreAdapter implements DatabaseProvider and RepositoryProvider for Firestore.
// This adapter follows the same pattern as Gmail/AsiaPay adapters - it handles
// connection initialization and delegates repository creation to the registry.
type FirestoreAdapter struct {
	client    *firestore.Client
	projectID string
	enabled   bool
}

// NewFirestoreAdapter creates a new Firestore database adapter.
func NewFirestoreAdapter() *FirestoreAdapter {
	return &FirestoreAdapter{}
}

// Name returns the provider name.
func (a *FirestoreAdapter) Name() string {
	return "firestore"
}

// Initialize sets up the Firestore connection.
func (a *FirestoreAdapter) Initialize(config *dbpb.DatabaseProviderConfig) error {
	fsConfig := config.GetFirestore()
	if fsConfig == nil {
		return fmt.Errorf("firestore adapter requires firestore configuration")
	}

	projectID := fsConfig.ProjectId
	if projectID == "" {
		return fmt.Errorf("firestore adapter requires 'project_id' in configuration")
	}
	a.projectID = projectID

	databaseID := fsConfig.DatabaseId

	ctx := context.Background()
	var client *firestore.Client
	var err error

	if fsConfig.CredentialsPath != "" {
		log.Printf("📄 Using Firestore credentials from: %s", fsConfig.CredentialsPath)
		if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", fsConfig.CredentialsPath); err != nil {
			return fmt.Errorf("failed to set Google credentials: %w", err)
		}
	} else {
		log.Printf("🔑 Using Application Default Credentials for Firestore")
	}

	if databaseID != "" && databaseID != "(default)" {
		log.Printf("🔥 Connecting to named Firestore database: %s (project: %s)", databaseID, projectID)
		client, err = firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	} else {
		log.Printf("🔥 Connecting to (default) Firestore database (project: %s)", projectID)
		client, err = firestore.NewClient(ctx, projectID)
	}

	if err != nil {
		return fmt.Errorf("failed to create firestore client: %w", err)
	}

	a.client = client
	a.enabled = config.Enabled

	log.Printf("✅ Firestore adapter initialized successfully")
	return nil
}

// GetConnection returns the Firestore client connection.
func (a *FirestoreAdapter) GetConnection() any {
	return a.client
}

// Close closes the Firestore connection.
func (a *FirestoreAdapter) Close() error {
	if a.client != nil {
		log.Printf("🔌 Firestore adapter closing connection")
		return a.client.Close()
	}
	return nil
}

// IsHealthy performs a health check on the Firestore connection.
func (a *FirestoreAdapter) IsHealthy(ctx context.Context) error {
	if a.client == nil {
		return fmt.Errorf("firestore client not initialized")
	}

	_, err := a.client.Collection("_health").Doc("_check").Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil // NotFound is fine - means connection works
		}
		return fmt.Errorf("firestore health check failed: %w", err)
	}
	return nil
}

// IsEnabled returns whether this adapter is currently enabled.
func (a *FirestoreAdapter) IsEnabled() bool {
	return a.enabled
}

// =============================================================================
// RepositoryProvider Implementation - Delegates to Registry
// =============================================================================

// CreateRepository creates a repository by looking up the registered factory.
// This replaces the giant switch statement by delegating to self-registered factories.
func (a *FirestoreAdapter) CreateRepository(entityName string, conn any, collectionName string) (any, error) {
	return registry.CreateRepository("firestore", entityName, conn, collectionName)
}

// GetTransactionManager returns the Firestore transaction manager.
func (a *FirestoreAdapter) GetTransactionManager() interfaces.TransactionManager {
	if a.client == nil || !a.enabled {
		return nil
	}
	return core.NewFirestoreTransactionManager(a.client)
}

// HealthCheck checks if the Firestore adapter is healthy.
func (a *FirestoreAdapter) HealthCheck(ctx context.Context) error {
	return a.IsHealthy(ctx)
}

// Compile-time interface checks
var _ ports.DatabaseProvider = (*FirestoreAdapter)(nil)
var _ ports.RepositoryProvider = (*FirestoreAdapter)(nil)
