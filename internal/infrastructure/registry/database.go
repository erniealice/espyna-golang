package registry

import (
	"fmt"
	"sync"

	"leapfor.xyz/espyna/internal/application/ports"
	dbpb "leapfor.xyz/esqyma/golang/v1/infrastructure/database"
)

// =============================================================================
// Database Factory Registry Instance
// =============================================================================

var databaseRegistry = NewFactoryRegistry[ports.DatabaseProvider, *dbpb.DatabaseProviderConfig]("database")

// =============================================================================
// Database Provider Functions (delegates to generic registry)
// =============================================================================

func RegisterDatabaseProviderFactory(name string, factory func() ports.DatabaseProvider) {
	databaseRegistry.RegisterFactory(name, factory)
}

func GetDatabaseProviderFactory(name string) (func() ports.DatabaseProvider, bool) {
	return databaseRegistry.GetFactory(name)
}

func ListAvailableDatabaseProviderFactories() []string {
	return databaseRegistry.ListFactories()
}

// DatabaseConfigTransformer transforms raw config to DatabaseProviderConfig
type DatabaseConfigTransformer func(rawConfig map[string]any) (*dbpb.DatabaseProviderConfig, error)

func RegisterDatabaseConfigTransformer(name string, transformer DatabaseConfigTransformer) {
	databaseRegistry.RegisterConfigTransformer(name, transformer)
}

func GetDatabaseConfigTransformer(name string) (DatabaseConfigTransformer, bool) {
	return databaseRegistry.GetConfigTransformer(name)
}

func TransformDatabaseConfig(name string, rawConfig map[string]any) (*dbpb.DatabaseProviderConfig, error) {
	return databaseRegistry.TransformConfig(name, rawConfig)
}

func RegisterDatabaseBuildFromEnv(name string, builder func() (ports.DatabaseProvider, error)) {
	databaseRegistry.RegisterBuildFromEnv(name, builder)
}

func GetDatabaseBuildFromEnv(name string) (func() (ports.DatabaseProvider, error), bool) {
	return databaseRegistry.GetBuildFromEnv(name)
}

func BuildDatabaseProviderFromEnv(name string) (ports.DatabaseProvider, error) {
	return databaseRegistry.BuildFromEnv(name)
}

func ListAvailableDatabaseBuildFromEnv() []string {
	return databaseRegistry.ListBuildFromEnv()
}

// RegisterDatabaseProvider registers both factory and config transformer.
func RegisterDatabaseProvider(name string, factory func() ports.DatabaseProvider, transformer DatabaseConfigTransformer) {
	RegisterDatabaseProviderFactory(name, factory)
	if transformer != nil {
		RegisterDatabaseConfigTransformer(name, transformer)
	}
}

// =============================================================================
// Database Table Config Registry
// =============================================================================
//
// TableConfigRegistry allows database adapters to register their own table/collection
// name configuration builders. This moves table naming logic from the composition
// layer to the adapters where it belongs.
//
// =============================================================================

// DatabaseTableConfig holds database table/collection names.
// This is defined here to avoid circular imports with composition/config.
type DatabaseTableConfig struct {
	// Common
	Attribute, Category string
	// Entity
	Client, ClientAttribute, ClientCategory                            string
	Admin, Manager, Staff, StaffAttribute                              string
	Delegate, DelegateAttribute, DelegateClient                        string
	Group, GroupAttribute, Location, LocationAttribute                 string
	Permission, Role, RolePermission                                   string
	User, Workspace, WorkspaceClient, WorkspaceUser, WorkspaceUserRole string
	// Event
	Event, EventAttribute, EventClient, EventProduct, EventSettings string
	// Framework
	Framework, Objective, Task string
	// Payment
	Payment, PaymentAttribute, PaymentMethod, PaymentProfile string
	PaymentProfilePaymentMethod                              string
	// Integration
	IntegrationPayment string
	// Product
	Product, Collection, CollectionAttribute, CollectionParent, CollectionPlan string
	PriceProduct, ProductAttribute, ProductCollection, ProductPlan, Resource   string
	// Record
	Record string
	// Workflow
	Workflow, WorkflowTemplate, Stage, Activity, StageTemplate, ActivityTemplate string
	// Session
	Session string
	// Subscription
	Plan, PlanAttribute, PlanLocation, PlanSettings      string
	Balance, BalanceAttribute, Invoice, InvoiceAttribute string
	PricePlan, Subscription, SubscriptionAttribute       string
}

// TableConfigBuilder creates table config from environment variables.
type TableConfigBuilder func() *DatabaseTableConfig

// tableConfigRegistry holds registered table config builders
var tableConfigBuilders = struct {
	builders map[string]TableConfigBuilder
	mutex    sync.RWMutex
}{
	builders: make(map[string]TableConfigBuilder),
}

// RegisterDatabaseTableConfigBuilder registers a table config builder for a provider.
// This is called from init() in each database adapter.
func RegisterDatabaseTableConfigBuilder(providerName string, builder TableConfigBuilder) {
	tableConfigBuilders.mutex.Lock()
	defer tableConfigBuilders.mutex.Unlock()

	if builder == nil {
		panic(fmt.Sprintf("RegisterDatabaseTableConfigBuilder: builder is nil for %s", providerName))
	}
	tableConfigBuilders.builders[providerName] = builder
}

// GetDatabaseTableConfigBuilder retrieves a registered table config builder.
func GetDatabaseTableConfigBuilder(providerName string) (TableConfigBuilder, bool) {
	tableConfigBuilders.mutex.RLock()
	defer tableConfigBuilders.mutex.RUnlock()

	builder, exists := tableConfigBuilders.builders[providerName]
	return builder, exists
}

// BuildDatabaseTableConfig creates table config using the registered builder.
func BuildDatabaseTableConfig(providerName string) (*DatabaseTableConfig, error) {
	builder, exists := GetDatabaseTableConfigBuilder(providerName)
	if !exists {
		// Fallback to default config
		return DefaultDatabaseTableConfig(), nil
	}
	return builder(), nil
}

// DefaultDatabaseTableConfig returns sensible defaults for table names.
func DefaultDatabaseTableConfig() *DatabaseTableConfig {
	return &DatabaseTableConfig{
		// Common
		Attribute: "attribute",
		Category:  "category",
		// Entity
		Client: "client", ClientAttribute: "client_attribute", ClientCategory: "client_category",
		Admin: "admin", Manager: "manager",
		Staff: "staff", StaffAttribute: "staff_attribute",
		Delegate: "delegate", DelegateAttribute: "delegate_attribute", DelegateClient: "delegate_client",
		Group: "group", GroupAttribute: "group_attribute",
		Location: "location", LocationAttribute: "location_attribute",
		Permission: "permission", Role: "role", RolePermission: "role_permission",
		User: "user", Workspace: "workspace", WorkspaceClient: "workspace_client",
		WorkspaceUser: "workspace_user", WorkspaceUserRole: "workspace_user_role",
		// Event
		Event: "event", EventAttribute: "event_attribute", EventClient: "event_client",
		EventProduct: "event_product", EventSettings: "event_settings",
		// Framework
		Framework: "framework", Objective: "objective", Task: "task",
		// Payment
		Payment: "payment", PaymentAttribute: "payment_attribute",
		PaymentMethod: "payment_method", PaymentProfile: "payment_profile",
		PaymentProfilePaymentMethod: "payment_profile_payment_method",
		// Integration
		IntegrationPayment: "integration_payment",
		// Product
		Product: "product", Collection: "collection", CollectionAttribute: "collection_attribute",
		CollectionParent: "collection_parent", CollectionPlan: "collection_plan",
		PriceProduct: "price_product", ProductAttribute: "product_attribute",
		ProductCollection: "product_collection", ProductPlan: "product_plan", Resource: "resource",
		// Record
		Record: "record",
		// Workflow
		Workflow: "workflow", WorkflowTemplate: "workflow_template",
		Stage: "stage", Activity: "activity",
		StageTemplate: "stage_template", ActivityTemplate: "activity_template",
		// Session
		Session: "session",
		// Subscription
		Plan: "plan", PlanAttribute: "plan_attribute", PlanLocation: "plan_location", PlanSettings: "plan_settings",
		Balance: "balance", BalanceAttribute: "balance_attribute",
		Invoice: "invoice", InvoiceAttribute: "invoice_attribute",
		PricePlan: "price_plan", Subscription: "subscription", SubscriptionAttribute: "subscription_attribute",
	}
}

// =============================================================================
// Repository Factory Registry
// =============================================================================
//
// RepositoryFactoryRegistry provides self-registration for database repositories.
// This eliminates the giant switch statement in database providers by allowing
// each repository to register itself via init().
//
// Keys are composite: "provider:entity" (e.g., "firestore:client", "mock:user")
//
// =============================================================================

// RepositoryFactory creates a repository given a database connection and table name.
// The conn parameter type depends on the provider (e.g., *firestore.Client, *sql.DB).
type RepositoryFactory func(conn any, tableName string) (any, error)

// repositoryFactoryRegistry holds registered repository factories
type repositoryFactoryRegistry struct {
	factories map[string]RepositoryFactory
	mutex     sync.RWMutex
}

// Global repository factory registry
var repoRegistry = &repositoryFactoryRegistry{
	factories: make(map[string]RepositoryFactory),
}

// RegisterRepositoryFactory registers a repository factory for a provider:entity combination.
// This is called from init() in each repository file.
//
// Example:
//
//	func init() {
//	    registry.RegisterRepositoryFactory("firestore", "client", func(conn any, tableName string) (any, error) {
//	        client := conn.(*firestore.Client)
//	        ops := firestore.NewFirestoreOperations(client)
//	        return NewFirestoreClientRepository(ops, tableName), nil
//	    })
//	}
func RegisterRepositoryFactory(providerName, entityName string, factory RepositoryFactory) {
	repoRegistry.mutex.Lock()
	defer repoRegistry.mutex.Unlock()

	key := providerName + ":" + entityName
	if factory == nil {
		panic(fmt.Sprintf("RegisterRepositoryFactory: factory is nil for %s", key))
	}
	repoRegistry.factories[key] = factory
}

// GetRepositoryFactory retrieves a registered repository factory.
func GetRepositoryFactory(providerName, entityName string) (RepositoryFactory, bool) {
	repoRegistry.mutex.RLock()
	defer repoRegistry.mutex.RUnlock()

	key := providerName + ":" + entityName
	factory, exists := repoRegistry.factories[key]
	return factory, exists
}

// CreateRepository creates a repository using the registered factory.
// This is the replacement for the giant switch statement in database providers.
func CreateRepository(providerName, entityName string, conn any, tableName string) (any, error) {
	factory, exists := GetRepositoryFactory(providerName, entityName)
	if !exists {
		return nil, fmt.Errorf("no repository factory registered for %s:%s (available: %v)",
			providerName, entityName, ListRepositoryFactories(providerName))
	}
	return factory(conn, tableName)
}

// ListRepositoryFactories returns all registered entity names for a provider.
func ListRepositoryFactories(providerName string) []string {
	repoRegistry.mutex.RLock()
	defer repoRegistry.mutex.RUnlock()

	prefix := providerName + ":"
	var entities []string
	for key := range repoRegistry.factories {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			entities = append(entities, key[len(prefix):])
		}
	}
	return entities
}

// ListAllRepositoryFactories returns all registered provider:entity combinations.
func ListAllRepositoryFactories() []string {
	repoRegistry.mutex.RLock()
	defer repoRegistry.mutex.RUnlock()

	keys := make([]string, 0, len(repoRegistry.factories))
	for key := range repoRegistry.factories {
		keys = append(keys, key)
	}
	return keys
}

// =============================================================================
// Database Operations Factory Registry
// =============================================================================
//
// DatabaseOperationsFactory provides self-registration for DatabaseOperation creators.
// This allows creating technology-agnostic database operations from a provider connection.
//
// =============================================================================

// DatabaseOperationsFactory creates a DatabaseOperation from a database connection.
// The conn parameter type depends on the provider (e.g., *firestore.Client, *sql.DB).
type DatabaseOperationsFactory func(conn any) (any, error)

// databaseOpsRegistry holds registered database operations factories
type databaseOpsRegistry struct {
	factories map[string]DatabaseOperationsFactory
	mutex     sync.RWMutex
}

// Global database operations factory registry
var dbOpsRegistry = &databaseOpsRegistry{
	factories: make(map[string]DatabaseOperationsFactory),
}

// RegisterDatabaseOperationsFactory registers a DatabaseOperation factory for a provider.
// This is called from init() in each database adapter's core operations file.
//
// Example:
//
//	func init() {
//	    registry.RegisterDatabaseOperationsFactory("firestore", func(conn any) (any, error) {
//	        client := conn.(*firestore.Client)
//	        return NewFirestoreOperations(client), nil
//	    })
//	}
func RegisterDatabaseOperationsFactory(providerName string, factory DatabaseOperationsFactory) {
	dbOpsRegistry.mutex.Lock()
	defer dbOpsRegistry.mutex.Unlock()

	if factory == nil {
		panic(fmt.Sprintf("RegisterDatabaseOperationsFactory: factory is nil for %s", providerName))
	}
	dbOpsRegistry.factories[providerName] = factory
}

// GetDatabaseOperationsFactory retrieves a registered DatabaseOperation factory.
func GetDatabaseOperationsFactory(providerName string) (DatabaseOperationsFactory, bool) {
	dbOpsRegistry.mutex.RLock()
	defer dbOpsRegistry.mutex.RUnlock()

	factory, exists := dbOpsRegistry.factories[providerName]
	return factory, exists
}

// CreateDatabaseOperations creates a DatabaseOperation using the registered factory.
// Returns the technology-agnostic DatabaseOperation interface that provides
// Create, Read, Update, Delete, List, and Query methods.
func CreateDatabaseOperations(providerName string, conn any) (any, error) {
	factory, exists := GetDatabaseOperationsFactory(providerName)
	if !exists {
		return nil, fmt.Errorf("no database operations factory registered for %s (available: %v)",
			providerName, ListDatabaseOperationsFactories())
	}
	return factory(conn)
}

// ListDatabaseOperationsFactories returns all registered provider names.
func ListDatabaseOperationsFactories() []string {
	dbOpsRegistry.mutex.RLock()
	defer dbOpsRegistry.mutex.RUnlock()

	names := make([]string, 0, len(dbOpsRegistry.factories))
	for name := range dbOpsRegistry.factories {
		names = append(names, name)
	}
	return names
}
