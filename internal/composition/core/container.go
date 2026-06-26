package core

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	infraports "github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/usecases"
	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/composition/core/initializers/domain"
	infraopts "github.com/erniealice/espyna-golang/internal/composition/options/infrastructure"
	"github.com/erniealice/espyna-golang/internal/composition/providers"
	repodomain "github.com/erniealice/espyna-golang/internal/composition/providers/domain"
	"github.com/erniealice/espyna-golang/internal/composition/providers/integration"
	"github.com/erniealice/espyna-golang/internal/composition/routing"
	dbifaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	txbridge "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/transactions"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	orchcontracts "github.com/erniealice/espyna-golang/internal/orchestration/contracts"
	workflowregistry "github.com/erniealice/espyna-golang/internal/orchestration/workflow"
	"github.com/erniealice/espyna-golang/shared/database/schema"
)

// RouteManager defines the interface for route management to avoid import cycles
type RouteManager interface {
	GetAllRoutes() []*routing.Route
	GetConfig() *routing.Config
	Close() error
	// Add other required methods as needed
}

// Platform holds all core infrastructure services with mock defaults
type Platform struct {
	Auth           contracts.Service           // Authentication/Authorization service
	Storage        contracts.Service           // Storage service (files, uploads)
	Metrics        contracts.Service           // Metrics and monitoring service
	Logger         contracts.Service           // Logging service
	Cache          contracts.Service           // Caching service
	Transaction    ports.Transactor            // Transaction port (real DB-backed, NoOp when no DB)
	IDGen          contracts.Service           // ID generation service (UUID v7, etc.)
	Email          ports.EmailProvider         // Email provider service (Gmail, SendGrid, etc.)
	Payment        ports.PaymentProvider       // Payment provider service (AsiaPay, Stripe, etc.)
	Scheduler      ports.SchedulerProvider     // Scheduler provider service (Calendly, etc.)
	Tabular        ports.TabularSourceProvider // Tabular data provider (Google Sheets, etc.)
	WorkflowEngine        ports.WorkflowEngineService        // Orchestration engine service
	WorkflowAssigneeQuery ports.WorkflowAssigneeQueryService // Engine identity bridge (read-only)

	// Multi-provider registries — all configured providers are active simultaneously.
	// Legacy single fields above are set to the first provider for backwards compat.
	PaymentProviders     map[string]ports.PaymentProvider
	SchedulerProviders   map[string]ports.SchedulerProvider
	FulfillmentProviders map[string]ports.FulfillmentProvider
}

// MockService provides a default mock implementation of the Service interface
type MockService struct {
	name string
}

func NewMockService(name string) *MockService {
	return &MockService{name: name}
}

func (m *MockService) Name() string {
	return m.name
}

func (m *MockService) Start(ctx context.Context) error {
	return nil
}

func (m *MockService) Stop(ctx context.Context) error {
	return nil
}

func (m *MockService) Health(ctx context.Context) error {
	return nil
}

// NewDefaultPlatform creates a Platform struct with mock defaults
func NewDefaultPlatform() *Platform {
	return &Platform{
		Auth:        NewMockService("mock-auth"), // Placeholder - actual auth service created separately
		Storage:     NewMockService("mock-storage"),
		Metrics:     NewMockService("mock-metrics"),
		Logger:      NewMockService("mock-logger"),
		Cache:       NewMockService("mock-cache"),
		// Transaction is a real ports.Transactor (NOT a MockService). The NoOp
		// fallback reports SupportsTransactions()==false so use cases take their
		// executeCore (no-tx) branch — identical to the pre-wiring dormant
		// behavior — until Initialize() replaces it with a DB-backed adapter.
		Transaction: ports.NewNoOpTransactor(),
		IDGen:       NewMockService("mock-idgen"), // Placeholder - actual ID service created by provider
	}
}

// Container is the main dependency injection container that manages all
// application components following hexagonal architecture principles.
type Container struct {
	mu        sync.RWMutex
	config    *Config
	providers *providers.Manager
	routing   RouteManager

	// Organized component groups
	useCases *usecases.Aggregate
	services Platform

	initialized bool
	closed      bool

	// workflowEngineFactory creates engine on first use (lazy mode only)
	workflowEngineFactory func() error
}

// Config holds the main container configuration.
// Provider-specific configurations are now handled by providers themselves via
// the registry pattern - each provider reads its own CONFIG_* environment variables.
// Database table configuration is now handled by the registry - adapters register their
// own table config builders and the Manager retrieves it based on the active provider.
type Config struct {
	// Application identity
	Name        string
	Version     string
	Environment string

	// Runtime configuration
	BusinessType       string
	WorkflowEngineMode string

	// Routing configuration
	RoutingConfig *routing.Config
}

// NewContainer creates a new container instance with default configuration and mock services
func NewContainer() *Container {
	return &Container{
		config: &Config{
			Name:        "espyna",
			Version:     "1.0.0",
			Environment: "development",
		},
		services: *NewDefaultPlatform(),
	}
}

// NewContainerFromEnv creates a container directly from environment variables.
// This is the recommended way to create a container - providers self-configure
// by reading their own CONFIG_* and provider-specific environment variables.
//
// Environment variables (provider selection):
//   - CONFIG_DATABASE_PROVIDER: mock_db, postgres, firestore (default: mock_db)
//   - CONFIG_AUTH_PROVIDER: mock, password, firebase (default: mock)
//   - CONFIG_ID_PROVIDER: noop, google_uuidv7 (default: noop)
//   - CONFIG_STORAGE_PROVIDER: mock_storage, local, gcs (default: mock_storage)
//   - CONFIG_EMAIL_PROVIDER: mock_email, google_email, microsoft_email (default: mock_email)
//   - CONFIG_PAYMENT_PROVIDER: mock_payment, asiapay, stripe (default: mock_payment)
//   - CONFIG_WORKFLOW_ENGINE_MODE: eager, late, lazy (default: late)
//
// Each provider reads its own configuration from environment variables.
// See provider implementations for provider-specific variables.
func NewContainerFromEnv() (*Container, error) {
	container := NewContainer()

	// Log which providers are configured (providers self-configure from env)
	fmt.Printf("📦 Creating container from environment...\n")
	fmt.Printf("   Database:  %s\n", strings.ToLower(getEnv("CONFIG_DATABASE_PROVIDER", "mock_db")))
	fmt.Printf("   Auth:      %s\n", strings.ToLower(getEnv("CONFIG_AUTH_PROVIDER", "mock")))
	fmt.Printf("   ID:        %s\n", strings.ToLower(getEnv("CONFIG_ID_PROVIDER", "noop")))
	fmt.Printf("   Storage:   %s\n", strings.ToLower(getEnv("CONFIG_STORAGE_PROVIDER", "mock_storage")))
	fmt.Printf("   Email:     %s\n", strings.ToLower(getEnv("CONFIG_EMAIL_PROVIDER", "mock_email")))
	fmt.Printf("   Payment:   %s\n", strings.ToLower(getEnv("CONFIG_PAYMENT_PROVIDER", "mock_payment")))
	fmt.Printf("   Scheduler: %s\n", strings.ToLower(getEnv("CONFIG_SCHEDULER_PROVIDER", "mock_scheduler")))

	// Set runtime configuration
	container.config.BusinessType = getEnv("BUSINESS_TYPE", "education")
	container.config.WorkflowEngineMode = strings.ToLower(getEnv("CONFIG_WORKFLOW_ENGINE_MODE", "late"))
	fmt.Printf("   Workflow:  %s\n", container.config.WorkflowEngineMode)

	// Note: Database table config is now handled by the registry - adapters register their
	// own table config builders and the Manager retrieves it based on the active provider.

	// Initialize the container (providers self-configure via registry)
	fmt.Printf("🔄 Initializing container...\n")
	if err := container.Initialize(); err != nil {
		return nil, fmt.Errorf("container initialization failed: %w", err)
	}

	fmt.Printf("✅ Container initialized successfully\n")
	return container, nil
}

// getEnv returns environment variable value or default if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvWithFallback tries primary key first, then fallback key, then default
func getEnvWithFallback(primaryKey, fallbackKey, defaultValue string) string {
	if value := os.Getenv(primaryKey); value != "" {
		return value
	}
	if value := os.Getenv(fallbackKey); value != "" {
		return value
	}
	return defaultValue
}

// parseInt converts string to int with default value
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// NewContainerWithOptions creates a new container instance with functional options
func NewContainerWithOptions(opts ...infraopts.ContainerOption) (*Container, error) {
	container := NewContainer()

	// Apply functional options
	for _, opt := range opts {
		if err := opt(container); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Initialize container
	if err := container.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize container: %w", err)
	}

	return container, nil
}

// Initialize sets up all container components
func (c *Container) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return fmt.Errorf("container already initialized")
	}

	fmt.Printf("📦 Starting container initialization...\n")

	// Initialize provider manager (providers read their own config from env)
	// Table configuration is now obtained from the registry based on active database provider
	fmt.Printf("🔧 Initializing provider manager...\n")
	providerManager, err := providers.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create provider manager: %w", err)
	}
	c.providers = providerManager
	fmt.Printf("✅ Provider manager initialized (table config: %s)\n", providerManager.GetDBTableConfig().TableName("client"))

	// Plan 2 (reflectionless CRUD) — descriptor registry build + boot-shot schema
	// validator. This runs AFTER every adapter init() has populated
	// protoregistry.GlobalTypes (the provider manager above triggered the provider
	// factory, whose package's transitive imports register the pb messages). The
	// container is dialect-neutral, so it cannot import the postgresql-tagged
	// validator directly: it resolves the active provider's validator via the
	// registry hook (mirroring RegisterDatabaseTableConfigBuilder) and calls it with
	// the *sql.DB from the provider connection. SHADOW mode — the validator
	// fails-fast on drift but does NOT flip operations.go's unknown-column behavior.
	if err := c.runSchemaBootShot(); err != nil {
		return fmt.Errorf("schema boot-shot validation failed: %w", err)
	}

	// Initialize email provider from environment
	fmt.Printf("📧 Initializing email provider...\n")
	if provider, err := integration.CreateEmailProvider(); err != nil {
		return fmt.Errorf("failed to initialize email provider: %w", err)
	} else if provider != nil {
		c.services.Email = provider
		fmt.Printf("✅ Email provider initialized: %s\n", provider.Name())
	}

	// Initialize payment providers from environment (supports multiple comma-separated)
	fmt.Printf("💳 Initializing payment providers...\n")
	if providers, err := integration.CreatePaymentProviders(); err != nil {
		fmt.Printf("⚠️ Failed to initialize payment providers: %v\n", err)
	} else if len(providers) > 0 {
		c.services.PaymentProviders = providers
		// Set legacy single field to first provider for backwards compat
		for _, p := range providers {
			c.services.Payment = p
			break
		}
		names := make([]string, 0, len(providers))
		for name := range providers {
			names = append(names, name)
		}
		fmt.Printf("✅ Payment providers initialized: %v\n", names)
	}

	// Initialize scheduler providers from environment (supports multiple comma-separated)
	fmt.Printf("📅 Initializing scheduler providers...\n")
	if providers, err := integration.CreateSchedulerProviders(); err != nil {
		fmt.Printf("⚠️ Failed to initialize scheduler providers: %v\n", err)
	} else if len(providers) > 0 {
		c.services.SchedulerProviders = providers
		for _, p := range providers {
			c.services.Scheduler = p
			break
		}
		names := make([]string, 0, len(providers))
		for name := range providers {
			names = append(names, name)
		}
		fmt.Printf("✅ Scheduler providers initialized: %v\n", names)
	}

	// Initialize fulfillment providers from environment (supports multiple comma-separated)
	fmt.Printf("🚚 Initializing fulfillment providers...\n")
	if providers, err := integration.CreateFulfillmentProviders(); err != nil {
		fmt.Printf("⚠️ Failed to initialize fulfillment providers: %v\n", err)
	} else if len(providers) > 0 {
		c.services.FulfillmentProviders = providers
		names := make([]string, 0, len(providers))
		for name := range providers {
			names = append(names, name)
		}
		fmt.Printf("✅ Fulfillment providers initialized: %v\n", names)
	}

	// Initialize tabular provider from environment (Google Sheets, etc.)
	fmt.Printf("📊 Initializing tabular provider...\n")
	if provider, err := integration.CreateTabularProvider(); err != nil {
		fmt.Printf("⚠️ Failed to initialize tabular provider: %v\n", err)
	} else if provider != nil {
		c.services.Tabular = provider
		fmt.Printf("✅ Tabular provider initialized: %s\n", provider.Name())
	}

	// Initialize the transaction port from the active DB adapter (provider-agnostic).
	//
	// This is the ONE Platform service NewDefaultPlatform leaves as a NoOp:
	// the use-case `if Transactor != nil && Transactor.SupportsTransactions()`
	// branches stay dormant until a real, DB-backed Transactor is wired here.
	// We must run BEFORE InitializeAll (whose getServices casts services.Transaction
	// to ports.Transactor at usecases.go) and before initializeWorkflowEngine
	// (whose getServicesForInitializers does the same cast), so both casts see the
	// real adapter.
	//
	// Access is provider-agnostic: GetDatabaseProvider() returns a *ProviderWrapper
	// (contracts.Provider) that does NOT delegate GetTransactionManager — we must
	// unwrap via its Provider() escape hatch to reach the concrete adapter
	// (postgres / firestore / mock), all of which expose
	// GetTransactionManager() dbifaces.TransactionManager. If anything is absent
	// (no DB configured, adapter not connected, manager nil), we keep the NoOp
	// fallback so boot never breaks — use cases simply run their no-tx branch.
	fmt.Printf("🔁 Initializing transaction port...\n")
	if dbProvider := c.providers.GetDatabaseProvider(); dbProvider != nil {
		if w, ok := dbProvider.(interface{ Provider() any }); ok {
			if tmProvider, ok := w.Provider().(interface {
				GetTransactionManager() dbifaces.TransactionManager
			}); ok {
				if mgr := tmProvider.GetTransactionManager(); mgr != nil {
					c.services.Transaction = txbridge.NewTransactionServiceAdapter(mgr)
					fmt.Printf("✅ Transaction port wired from DB adapter (supports tx: %v)\n", c.services.Transaction.SupportsTransactions())
				} else {
					fmt.Printf("⚠️ DB adapter returned a nil transaction manager — keeping NoOp transaction port\n")
				}
			} else {
				fmt.Printf("⚠️ DB adapter does not expose GetTransactionManager — keeping NoOp transaction port\n")
			}
		} else {
			fmt.Printf("⚠️ DB provider is not unwrappable — keeping NoOp transaction port\n")
		}
	} else {
		fmt.Printf("⚠️ No DB provider configured — keeping NoOp transaction port\n")
	}

	// Initialize use cases FIRST (before routing and orchestration)
	fmt.Printf("🔧 Initializing use cases...\n")
	usecaseInitializer := NewUseCaseInitializer(c.providers)
	if err := usecaseInitializer.InitializeAll(c); err != nil {
		return fmt.Errorf("failed to initialize use cases: %w", err)
	}
	fmt.Printf("✅ Use cases initialized: %v\n", c.useCases != nil)

	// Initialize workflow engine AFTER use cases are ready
	if err := c.initializeWorkflowEngine(); err != nil {
		// Log as a warning, not a fatal error, as the app might run without the engine
		fmt.Printf("⚠️  Workflow Engine initialization failed: %v\n", err)
	}

	// Initialize routing manager with default config if not provided
	if c.config.RoutingConfig == nil {
		c.config.RoutingConfig = routing.DefaultConfig()
	}

	// Unlock before creating routing composer since it calls GetUseCases() which needs a read lock
	c.mu.Unlock()

	// Create routing composer that will manage all routes
	// This MUST happen AFTER use cases are initialized
	fmt.Printf("🔧 Creating routing composer (use cases available: %v)...\n", c.useCases != nil)
	composer, err := routing.NewComposer(&routing.ComposerConfig{
		Config:    c.config.RoutingConfig,
		Container: c,
	})
	if err != nil {
		c.mu.Lock() // Re-lock before returning error
		return fmt.Errorf("failed to create routing composer: %w", err)
	}

	// Re-lock to set final fields
	c.mu.Lock()
	c.routing = composer.GetRouteManager()
	fmt.Printf("✅ Routing composer created, routes registered: %d\n", len(c.routing.GetAllRoutes()))

	c.initialized = true
	fmt.Printf("✅ Container initialization complete!\n")
	return nil
}

func (c *Container) initializeWorkflowEngine() error {
	// The UsecaseInitializer is no longer the right place.
	// We need access to the domain.InitializeWorkflowEngine function
	// and the required repositories.
	// This requires moving some logic from usecases.go to here.

	// First, get the workflow repositories
	workflowRepos, err := repodomain.NewWorkflowRepositories(c.providers.GetDatabaseProvider(), c.providers.GetDBTableConfig())
	if err != nil {
		return fmt.Errorf("cannot initialize engine, failed to get workflow repositories: %w", err)
	}

	// Second, get the required services
	authSvc, txSvc, i18nSvc, idSvc, err := c.getServicesForInitializers()
	if err != nil {
		return fmt.Errorf("cannot initialize engine, failed to get services: %w", err)
	}

	// Third, create the executor registry
	executorRegistry := workflowregistry.NewRegistry(c.useCases)

	// Fourth, initialize the engine based on lifecycle mode
	switch orchcontracts.WorkflowEngineMode(c.config.WorkflowEngineMode) {
	case orchcontracts.ModeLate, orchcontracts.ModeEager, "": // Eager and Late are now the same
		fmt.Printf("🚀 Initializing Workflow Engine (%s binding mode)...\n", c.config.WorkflowEngineMode)
		engineUC, err := domain.InitializeWorkflowEngine(workflowRepos, authSvc, txSvc, i18nSvc, idSvc,
			executorRegistry)
		if err != nil {
			return err
		}
		c.services.WorkflowEngine = engineUC
		// Wire the engine identity bridge (Q-EIB-BRIDGE) if a DB connection is available.
		c.wireAssigneeQuery(engineUC)
		fmt.Printf("✅ Workflow Engine initialized\n")

	case orchcontracts.ModeLazy:
		fmt.Printf("😴 Workflow Engine deferred (lazy binding mode)...\n")
		// Store factory for lazy initialization
		c.workflowEngineFactory = func() error {
			// This lock is important to prevent race conditions on lazy init
			c.mu.Lock()
			defer c.mu.Unlock()
			if c.services.WorkflowEngine != nil {
				return nil // Already initialized
			}
			engineUC, err := domain.InitializeWorkflowEngine(workflowRepos, authSvc, txSvc, i18nSvc, idSvc,
				executorRegistry)
			if err != nil {
				return err
			}
			c.services.WorkflowEngine = engineUC
			// Wire the engine identity bridge (Q-EIB-BRIDGE) if a DB connection is available.
			c.wireAssigneeQuery(engineUC)
			fmt.Printf("✅ Workflow Engine initialized (lazily)\n")
			return nil
		}

	case orchcontracts.ModeNone:
		fmt.Printf("⏭️ Workflow Engine disabled (none mode)\n")

	default:
		return fmt.Errorf("unknown Workflow Engine Mode: %s", c.config.WorkflowEngineMode)
	}
	return nil
}

// getServicesForInitializers is a new helper similar to the one in usecases.go
func (c *Container) getServicesForInitializers() (
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	err error,
) {
	// This logic is duplicated from the UseCaseInitializer, which is a sign it should
	// probably be centralized, but for now we copy it here to make this work.

	// Get auth service from provider manager
	if authProvider := c.providers.GetAuthProvider(); authProvider != nil {
		if authService, ok := authProvider.(ports.Authorizer); ok {
			authSvc = authService
		}
	}
	// Fallback to mock if needed
	if authSvc == nil {
		authSvc, _ = c.services.Auth.(ports.Authorizer)
	}

	// Get ID service from provider manager
	if idProvider := c.providers.GetIDProvider(); idProvider != nil {
		if idWrapper, ok := idProvider.(interface{ GetIDService() ports.IDGenerator }); ok {
			idSvc = idWrapper.GetIDService()
		}
	}
	if idSvc == nil {
		idSvc, _ = c.services.IDGen.(ports.IDGenerator)
	}

	txSvc, _ = c.services.Transaction.(ports.Transactor)

	// P6 (E5): translation provider system retired. Use the port-level NoOp
	// translator directly. Track 1 (authcheck error-message translation) has
	// always run noop in production; Track 2 (lyngua label loading) is wired
	// directly in service-admin composition and does not flow through this path.
	i18nSvc = ports.NewNoOpTranslator()

	return authSvc, txSvc, i18nSvc, idSvc, nil
}

// GetProviderManager returns the provider manager
func (c *Container) GetProviderManager() *providers.Manager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.providers
}

// runSchemaBootShot builds the dialect-neutral descriptor registry and runs the
// active provider's registered boot-shot schema validator (Plan 2). It is a no-op
// for providers without a registered validator (mock, firestore) and for providers
// whose connection is not a *sql.DB — keeping the dialect-neutral container free of
// any SQL-dialect knowledge.
//
// Ordering: called from Initialize() immediately after the provider manager is set,
// so every adapter init() has already populated protoregistry.GlobalTypes.
func (c *Container) runSchemaBootShot() error {
	// Build the descriptor registry from protoregistry.GlobalTypes (idempotent).
	if err := schema.Build(); err != nil {
		return fmt.Errorf("descriptor registry build: %w", err)
	}
	// Enforce the total-table-count floor only in the fully-linked binary; a
	// collapse of the adapter import graph fails the boot loud.
	if err := schema.AssertMinimumCoverage(); err != nil {
		return err
	}

	dbProvider := c.GetDatabaseProvider()
	if dbProvider == nil {
		return nil
	}

	validator, ok := registry.GetSchemaValidator(dbProvider.Name())
	if !ok {
		// No validator registered for this provider (mock / firestore). Shadow
		// boot-shot is postgres-only this wave; nothing to reconcile.
		return nil
	}

	connHolder, ok := dbProvider.(interface{ GetConnection() any })
	if !ok {
		return nil
	}
	sqlDB, ok := connHolder.GetConnection().(*sql.DB)
	if !ok || sqlDB == nil {
		// Provider has a validator but a non-SQL connection — should not happen for
		// postgresql, but stay defensive rather than panic at boot.
		return nil
	}

	return validator(context.Background(), sqlDB)
}

// ─────────────────────────────────────────────────────────────────────────────
// Direct Provider Access - convenience methods for cleaner consumer API
// ─────────────────────────────────────────────────────────────────────────────

// GetDatabaseProvider returns the database provider directly
func (c *Container) GetDatabaseProvider() contracts.Provider {
	if c.providers == nil {
		return nil
	}
	return c.providers.GetDatabaseProvider()
}

// GetAuthProvider returns the auth provider directly
func (c *Container) GetAuthProvider() contracts.Provider {
	if c.providers == nil {
		return nil
	}
	return c.providers.GetAuthProvider()
}

// GetStorageProvider returns the storage provider directly
func (c *Container) GetStorageProvider() contracts.Provider {
	if c.providers == nil {
		return nil
	}
	return c.providers.GetStorageProvider()
}

// GetIDProvider returns the ID generation provider directly
func (c *Container) GetIDProvider() contracts.Provider {
	if c.providers == nil {
		return nil
	}
	return c.providers.GetIDProvider()
}

// GetPaymentProvider returns the payment provider directly
func (c *Container) GetPaymentProvider() ports.PaymentProvider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services.Payment
}

// GetEmailProvider returns the email provider directly
func (c *Container) GetEmailProvider() ports.EmailProvider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services.Email
}

// GetSchedulerProvider returns the scheduler provider directly
func (c *Container) GetSchedulerProvider() ports.SchedulerProvider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services.Scheduler
}

// GetPaymentProviders returns all registered payment providers
func (c *Container) GetPaymentProviders() map[string]ports.PaymentProvider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services.PaymentProviders
}

// GetPaymentProviderByName returns a specific payment provider by name
func (c *Container) GetPaymentProviderByName(name string) ports.PaymentProvider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.services.PaymentProviders == nil {
		return nil
	}
	return c.services.PaymentProviders[name]
}

// GetSchedulerProviders returns all registered scheduler providers
func (c *Container) GetSchedulerProviders() map[string]ports.SchedulerProvider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services.SchedulerProviders
}

// GetSchedulerProviderByName returns a specific scheduler provider by name
func (c *Container) GetSchedulerProviderByName(name string) ports.SchedulerProvider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.services.SchedulerProviders == nil {
		return nil
	}
	return c.services.SchedulerProviders[name]
}

// GetFulfillmentProviders returns all registered fulfillment providers
func (c *Container) GetFulfillmentProviders() map[string]ports.FulfillmentProvider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services.FulfillmentProviders
}

// GetFulfillmentProviderByName returns a specific fulfillment provider by name
func (c *Container) GetFulfillmentProviderByName(name string) ports.FulfillmentProvider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.services.FulfillmentProviders == nil {
		return nil
	}
	return c.services.FulfillmentProviders[name]
}

// GetDBTableConfig returns the database table configuration directly
func (c *Container) GetDBTableConfig() *registry.TableConfig {
	if c.providers == nil {
		return nil
	}
	return c.providers.GetDBTableConfig()
}

// GetDatabaseOperations returns a technology-agnostic database operations interface.
// This provides Create, Read, Update, Delete, List, and Query operations
// that work regardless of the underlying database (Firestore, Postgres, Mock).
//
// Usage:
//
//	ops := container.GetDatabaseOperations()
//	if ops != nil {
//	    doc, err := ops.Read(ctx, "my_collection", "doc-id")
//	    result, err := ops.Create(ctx, "my_collection", data)
//	}
func (c *Container) GetDatabaseOperations() interface{} {
	dbProvider := c.GetDatabaseProvider()
	if dbProvider == nil {
		return nil
	}

	// Get the connection from the provider
	conn := dbProvider.(interface{ GetConnection() any }).GetConnection()
	if conn == nil {
		return nil
	}

	// Use the registry to create operations based on provider type
	providerName := dbProvider.Name()
	ops, err := registry.CreateDatabaseOperations(providerName, conn)
	if err != nil {
		fmt.Printf("⚠️ Failed to create database operations: %v\n", err)
		return nil
	}

	// 20260518-hexagonal-strict-adherence Phase 1.D + 20260521-composition-reshape
	// Q-CR1 — transparently wrap the raw ops with any registered
	// composition-root decorator (audit today; future encryption / CDC /
	// webhook fan-out land in the same chain). Apps consume the decorated
	// ops via consumer.NewDatabaseAdapterFromContainer without needing to
	// know decoration happened.
	if sqlDB, ok := conn.(*sql.DB); ok && sqlDB != nil {
		ops = applyRegisteredOperationsDecorators(ops, sqlDB)
	}

	return ops
}

// applyRegisteredOperationsDecorators returns a DatabaseOperations impl
// wrapped with every registered composition-root decorator. Audit is the
// only entry today; future cross-cutting write-time concerns (encryption,
// CDC, webhook fan-out) land in this chain.
//
// Returns ops unchanged when no decorator is registered for the given db.
func applyRegisteredOperationsDecorators(ops any, db *sql.DB) any {
	if ops == nil || db == nil {
		return ops
	}
	if auditSvc := auditServiceFromDB(db); auditSvc != nil {
		if decorated := decorateWithAudit(ops, db, auditSvc); decorated != nil {
			ops = decorated
		}
	}
	return ops
}

// decorateWithAudit returns ops wrapped with audit-decorated ops when the
// audit-enabled operations factory has been registered (e.g. via
// contrib/postgres init()). Returns the original ops unchanged when the
// factory is unregistered or returns nil.
func decorateWithAudit(ops any, db *sql.DB, auditSvc infraports.AuditService) any {
	if ops == nil || db == nil || auditSvc == nil {
		return ops
	}
	factory, ok := registry.GetAuditEnabledOperationsFactory()
	if !ok || factory == nil {
		return ops
	}
	decorated := factory(db, auditSvc)
	if decorated == nil {
		return ops
	}
	return decorated
}

// auditServiceFromDB resolves the registered audit service from the DB.
// Keep behaviorally identical with the twin in
// composition/core/initializers/service/audit.go; cannot dedupe until
// core no longer imports core/initializers (Codex round-1 §6).
func auditServiceFromDB(db *sql.DB) infraports.AuditService {
	if db == nil {
		return nil
	}
	factory, ok := registry.GetAuditServiceFactory()
	if !ok || factory == nil {
		return nil
	}
	result := factory(db)
	if result == nil {
		return nil
	}
	if svc, ok := result.(infraports.AuditService); ok {
		return svc
	}
	return nil
}

// GetUseCases returns the use case aggregate
func (c *Container) GetUseCases() *usecases.Aggregate {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.useCases
}

// GetWorkflowEngine returns the workflow engine service.
// This is the orchestration engine, managed as a first-class container service.
func (c *Container) GetWorkflowEngine() ports.WorkflowEngineService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services.WorkflowEngine
}

// GetWorkflowEngineService is an alias for GetWorkflowEngine for routing compatibility
func (c *Container) GetWorkflowEngineService() ports.WorkflowEngineService {
	return c.GetWorkflowEngine()
}

// GetWorkflowAssigneeQueryService returns the engine identity bridge service.
// Returns nil if the bridge has not been wired (e.g. no DB connection or
// workflow engine not initialized).
func (c *Container) GetWorkflowAssigneeQueryService() ports.WorkflowAssigneeQueryService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.services.WorkflowAssigneeQuery
}

// wireAssigneeQuery resolves the registered AssigneeQueryRepository factory
// from the registry and wires it into the engine use cases so it can serve
// WorkflowAssigneeQueryService.
//
// This is best-effort — if no factory is registered (e.g. mock DB, firestore),
// or the DB connection is unavailable, the bridge is silently skipped. The
// engine remains functional; only the assignee query surface is absent.
func (c *Container) wireAssigneeQuery(engineSvc ports.WorkflowEngineService) {
	// Resolve the factory from the registry (registered by the postgres adapter
	// in its init() via registry.RegisterAssigneeQueryFactory).
	factory, ok := registry.GetAssigneeQueryFactory()
	if !ok || factory == nil {
		return
	}

	// Get the DB connection from the provider.
	dbProvider := c.providers.GetDatabaseProvider()
	if dbProvider == nil {
		return
	}
	connHolder, ok := dbProvider.(interface{ GetConnection() any })
	if !ok {
		return
	}
	conn := connHolder.GetConnection()
	if conn == nil {
		return
	}

	// Create the adapter via the factory.
	repoAny := factory(conn)
	if repoAny == nil {
		return
	}

	// Type-assert to access SetAssigneeQueryRepository on the concrete engine.
	// The setter takes `any` so that this structural assertion works without
	// importing the engine package.
	type assigneeWirer interface {
		SetAssigneeQueryRepository(repo any)
	}

	wirer, ok := engineSvc.(assigneeWirer)
	if !ok {
		fmt.Printf("⚠️ Engine does not support assignee query wiring — skipping identity bridge\n")
		return
	}

	wirer.SetAssigneeQueryRepository(repoAny)

	// Also store on Platform so consumers can access it directly.
	if querySvc, ok := engineSvc.(ports.WorkflowAssigneeQueryService); ok {
		c.services.WorkflowAssigneeQuery = querySvc
	}
	fmt.Printf("✅ Engine identity bridge wired (WorkflowAssigneeQueryService)\n")
}

// GetConfig returns the container configuration (implements infraopts.Container interface)
func (c *Container) GetConfig() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// SetConfig sets the container configuration (implements infraopts.Container interface)
func (c *Container) SetConfig(cfg interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if config, ok := cfg.(*Config); ok {
		c.config = config
	}
}

// GetRouteManager returns the route manager
func (c *Container) GetRouteManager() RouteManager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.routing
}

// Close closes all container resources including providers, routing, and services
func (c *Container) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close route manager
	if c.routing != nil {
		if err := c.routing.Close(); err != nil {
			return fmt.Errorf("failed to close route manager: %w", err)
		}
	}

	// Close provider manager (which closes database, auth, etc.)
	if c.providers != nil {
		if err := c.providers.Close(); err != nil {
			return fmt.Errorf("failed to close provider manager: %w", err)
		}
	}

	// Close email provider
	if c.services.Email != nil {
		if err := c.services.Email.Close(); err != nil {
			return fmt.Errorf("failed to close email provider: %w", err)
		}
	}

	// Close payment provider
	if c.services.Payment != nil {
		if err := c.services.Payment.Close(); err != nil {
			return fmt.Errorf("failed to close payment provider: %w", err)
		}
	}

	// Close scheduler provider
	if c.services.Scheduler != nil {
		if err := c.services.Scheduler.Close(); err != nil {
			return fmt.Errorf("failed to close scheduler provider: %w", err)
		}
	}

	return nil
}
