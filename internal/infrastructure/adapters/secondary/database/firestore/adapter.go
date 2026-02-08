//go:build firestore

package firestore

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/erniealice/espyna-golang/internal/application/ports"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/firestore/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	dbpb "github.com/erniealice/esqyma/pkg/schema/v1/infrastructure/database"
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
func buildTableConfig() *registry.DatabaseTableConfig {
	prefix := os.Getenv("FIRESTORE_TABLE_PREFIX")
	return &registry.DatabaseTableConfig{
		// Common
		Attribute: prefix + getFirestoreTableEnv("ATTRIBUTE", "attribute"),
		// Entity
		Client:            prefix + getFirestoreTableEnv("CLIENT", "client"),
		ClientAttribute:   prefix + getFirestoreTableEnv("CLIENT_ATTRIBUTE", "client_attribute"),
		Admin:             prefix + getFirestoreTableEnv("ADMIN", "admin"),
		Manager:           prefix + getFirestoreTableEnv("MANAGER", "manager"),
		Staff:             prefix + getFirestoreTableEnv("STAFF", "staff"),
		StaffAttribute:    prefix + getFirestoreTableEnv("STAFF_ATTRIBUTE", "staff_attribute"),
		Delegate:          prefix + getFirestoreTableEnv("DELEGATE", "delegate"),
		DelegateAttribute: prefix + getFirestoreTableEnv("DELEGATE_ATTRIBUTE", "delegate_attribute"),
		DelegateClient:    prefix + getFirestoreTableEnv("DELEGATE_CLIENT", "delegate_client"),
		Group:             prefix + getFirestoreTableEnv("GROUP", "group"),
		GroupAttribute:    prefix + getFirestoreTableEnv("GROUP_ATTRIBUTE", "group_attribute"),
		Location:          prefix + getFirestoreTableEnv("LOCATION", "location"),
		LocationAttribute: prefix + getFirestoreTableEnv("LOCATION_ATTRIBUTE", "location_attribute"),
		Permission:        prefix + getFirestoreTableEnv("PERMISSION", "permission"),
		Role:              prefix + getFirestoreTableEnv("ROLE", "role"),
		RolePermission:    prefix + getFirestoreTableEnv("ROLE_PERMISSION", "role_permission"),
		User:              prefix + getFirestoreTableEnv("USER", "user"),
		Workspace:         prefix + getFirestoreTableEnv("WORKSPACE", "workspace"),
		WorkspaceClient:   prefix + getFirestoreTableEnv("WORKSPACE_CLIENT", "workspace_client"),
		WorkspaceUser:     prefix + getFirestoreTableEnv("WORKSPACE_USER", "workspace_user"),
		WorkspaceUserRole: prefix + getFirestoreTableEnv("WORKSPACE_USER_ROLE", "workspace_user_role"),
		// Event
		Event:          prefix + getFirestoreTableEnv("EVENT", "event"),
		EventAttribute: prefix + getFirestoreTableEnv("EVENT_ATTRIBUTE", "event_attribute"),
		EventClient:    prefix + getFirestoreTableEnv("EVENT_CLIENT", "event_client"),
		EventProduct:   prefix + getFirestoreTableEnv("EVENT_PRODUCT", "event_product"),
		EventSettings:  prefix + getFirestoreTableEnv("EVENT_SETTINGS", "event_settings"),
		// Framework
		Framework: prefix + getFirestoreTableEnv("FRAMEWORK", "framework"),
		Objective: prefix + getFirestoreTableEnv("OBJECTIVE", "objective"),
		Task:      prefix + getFirestoreTableEnv("TASK", "task"),
		// Payment
		Payment:                     prefix + getFirestoreTableEnv("PAYMENT", "payment"),
		PaymentAttribute:            prefix + getFirestoreTableEnv("PAYMENT_ATTRIBUTE", "payment_attribute"),
		PaymentMethod:               prefix + getFirestoreTableEnv("PAYMENT_METHOD", "payment_method"),
		PaymentProfile:              prefix + getFirestoreTableEnv("PAYMENT_PROFILE", "payment_profile"),
		PaymentProfilePaymentMethod: prefix + getFirestoreTableEnv("PAYMENT_PROFILE_PAYMENT_METHOD", "payment_profile_payment_method"),
		// Integration
		IntegrationPayment: prefix + getFirestoreTableEnv("INTEGRATION_PAYMENT", "integration_payment"),
		// Product
		Product:             prefix + getFirestoreTableEnv("PRODUCT", "product"),
		Collection:          prefix + getFirestoreTableEnv("COLLECTION", "collection"),
		CollectionAttribute: prefix + getFirestoreTableEnv("COLLECTION_ATTRIBUTE", "collection_attribute"),
		CollectionParent:    prefix + getFirestoreTableEnv("COLLECTION_PARENT", "collection_parent"),
		CollectionPlan:      prefix + getFirestoreTableEnv("COLLECTION_PLAN", "collection_plan"),
		PriceProduct:        prefix + getFirestoreTableEnv("PRICE_PRODUCT", "price_product"),
		ProductAttribute:    prefix + getFirestoreTableEnv("PRODUCT_ATTRIBUTE", "product_attribute"),
		ProductCollection:   prefix + getFirestoreTableEnv("PRODUCT_COLLECTION", "product_collection"),
		ProductPlan:         prefix + getFirestoreTableEnv("PRODUCT_PLAN", "product_plan"),
		Resource:            prefix + getFirestoreTableEnv("RESOURCE", "resource"),
		// Record
		Record: prefix + getFirestoreTableEnv("RECORD", "record"),
		// Workflow
		Workflow:         prefix + getFirestoreTableEnv("WORKFLOW", "workflow"),
		WorkflowTemplate: prefix + getFirestoreTableEnv("WORKFLOW_TEMPLATE", "workflow_template"),
		Stage:            prefix + getFirestoreTableEnv("STAGE", "stage"),
		Activity:         prefix + getFirestoreTableEnv("ACTIVITY", "activity"),
		StageTemplate:    prefix + getFirestoreTableEnv("STAGE_TEMPLATE", "stage_template"),
		ActivityTemplate: prefix + getFirestoreTableEnv("ACTIVITY_TEMPLATE", "activity_template"),
		// Subscription
		Plan:                  prefix + getFirestoreTableEnv("PLAN", "plan"),
		PlanAttribute:         prefix + getFirestoreTableEnv("PLAN_ATTRIBUTE", "plan_attribute"),
		PlanLocation:          prefix + getFirestoreTableEnv("PLAN_LOCATION", "plan_location"),
		PlanSettings:          prefix + getFirestoreTableEnv("PLAN_SETTINGS", "plan_settings"),
		Balance:               prefix + getFirestoreTableEnv("BALANCE", "balance"),
		BalanceAttribute:      prefix + getFirestoreTableEnv("BALANCE_ATTRIBUTE", "balance_attribute"),
		Invoice:               prefix + getFirestoreTableEnv("INVOICE", "invoice"),
		InvoiceAttribute:      prefix + getFirestoreTableEnv("INVOICE_ATTRIBUTE", "invoice_attribute"),
		PricePlan:             prefix + getFirestoreTableEnv("PRICE_PLAN", "price_plan"),
		Subscription:          prefix + getFirestoreTableEnv("SUBSCRIPTION", "subscription"),
		SubscriptionAttribute: prefix + getFirestoreTableEnv("SUBSCRIPTION_ATTRIBUTE", "subscription_attribute"),
	}
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
