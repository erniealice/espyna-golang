package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/usecases"
	"leapfor.xyz/espyna/internal/composition/contracts"
	"leapfor.xyz/espyna/internal/composition/core/initializers"
	infraopts "leapfor.xyz/espyna/internal/composition/options/infrastructure"
	"leapfor.xyz/espyna/internal/composition/providers"
	"leapfor.xyz/espyna/internal/composition/providers/domain"
	infraProviders "leapfor.xyz/espyna/internal/composition/providers/infrastructure"
	"leapfor.xyz/espyna/internal/composition/providers/integration"
	"leapfor.xyz/espyna/internal/composition/routing"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
	orchcontracts "leapfor.xyz/espyna/internal/orchestration/contracts"
	workflowregistry "leapfor.xyz/espyna/internal/orchestration/workflow"
)

// RouteManager defines the interface for route management to avoid import cycles
type RouteManager interface {
	GetAllRoutes() []*routing.Route
	GetConfig() *routing.Config
	Close() error
	// Add other required methods as needed
}

// Services holds all core infrastructure services with mock defaults
type Services struct {
	Auth           contracts.Service           // Authentication/Authorization service
	Storage        contracts.Service           // Storage service (files, uploads)
	Metrics        contracts.Service           // Metrics and monitoring service
	Logger         contracts.Service           // Logging service
	Cache          contracts.Service           // Caching service
	Translation    contracts.Service           // Translation/i18n service
	Transaction    contracts.Service           // Transaction management service
	IDGen          contracts.Service           // ID generation service (UUID v7, etc.)
	Email          ports.EmailProvider          // Email provider service (Gmail, SendGrid, etc.)
	Payment        ports.PaymentProvider        // Payment provider service (AsiaPay, Stripe, etc.)
	Scheduler      ports.SchedulerProvider      // Scheduler provider service (Calendly, etc.)
	Tabular        ports.TabularSourceProvider  // Tabular data provider (Google Sheets, etc.)
	WorkflowEngine ports.WorkflowEngineService  // Orchestration engine service
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

// translationServiceWrapper wraps ports.TranslationService to implement contracts.Service
type translationServiceWrapper struct {
	svc ports.TranslationService
}

func (w *translationServiceWrapper) Name() string {
	return "translation"
}

func (w *translationServiceWrapper) Start(ctx context.Context) error {
	return nil
}

func (w *translationServiceWrapper) Stop(ctx context.Context) error {
	return nil
}

func (w *translationServiceWrapper) Health(ctx context.Context) error {
	return nil
}

// NewDefaultServices creates a Services struct with mock defaults
func NewDefaultServices() *Services {
	return &Services{
		Auth:        NewMockService("mock-auth"), // Placeholder - actual auth service created separately
		Storage:     NewMockService("mock-storage"),
		Metrics:     NewMockService("mock-metrics"),
		Logger:      NewMockService("mock-logger"),
		Cache:       NewMockService("mock-cache"),
		Translation: NewMockService("mock-translation"),
		Transaction: NewMockService("mock-transaction"),
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
	services Services

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
		services: *NewDefaultServices(),
	}
}

// NewContainerFromEnv creates a container directly from environment variables.
// This is the recommended way to create a container - providers self-configure
// by reading their own CONFIG_* and provider-specific environment variables.
//
// Environment variables (provider selection):
//   - CONFIG_DATABASE_PROVIDER: mock_db, postgres, firestore (default: mock_db)
//   - CONFIG_AUTH_PROVIDER: mock_auth, firebase_auth (default: mock_auth)
//   - CONFIG_ID_PROVIDER: noop, google_uuidv7 (default: noop)
//   - CONFIG_STORAGE_PROVIDER: mock_storage, local, gcs (default: mock_storage)
//   - CONFIG_EMAIL_PROVIDER: mock_email, gmail, microsoft (default: mock_email)
//   - CONFIG_PAYMENT_PROVIDER: mock_payment, asiapay, stripe (default: mock_payment)
//   - CONFIG_WORKFLOW_ENGINE_MODE: eager, late, lazy (default: late)
//
// Each provider reads its own configuration from environment variables.
// See provider implementations for provider-specific variables.
func NewContainerFromEnv() *Container {
	container := NewContainer()

	// Log which providers are configured (providers self-configure from env)
	fmt.Printf("📦 Creating container from environment...\n")
	fmt.Printf("   Database:  %s\n", strings.ToLower(getEnv("CONFIG_DATABASE_PROVIDER", "mock_db")))
	fmt.Printf("   Auth:      %s\n", strings.ToLower(getEnv("CONFIG_AUTH_PROVIDER", "mock_auth")))
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
		fmt.Printf("❌ Container initialization failed: %v\n", err)
		fmt.Printf("⚠️  Falling back to basic initialization\n")
		routingConfig := routing.DefaultConfig()
		routeManager := routing.NewRouteManager(routingConfig)

		container.mu.Lock()
		container.routing = routeManager
		container.initialized = true
		container.mu.Unlock()
	} else {
		fmt.Printf("✅ Container initialized successfully\n")
	}

	return container
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
	fmt.Printf("✅ Provider manager initialized (table config: %s)\n", providerManager.GetDBTableConfig().Client)

	// Initialize email provider from environment
	fmt.Printf("📧 Initializing email provider...\n")
	if provider, err := integration.CreateEmailProvider(); err != nil {
		fmt.Printf("⚠️ Failed to initialize email provider: %v\n", err)
	} else if provider != nil {
		c.services.Email = provider
		fmt.Printf("✅ Email provider initialized: %s\n", provider.Name())
	}

	// Initialize payment provider from environment
	fmt.Printf("💳 Initializing payment provider...\n")
	if provider, err := integration.CreatePaymentProvider(); err != nil {
		fmt.Printf("⚠️ Failed to initialize payment provider: %v\n", err)
	} else if provider != nil {
		c.services.Payment = provider
		fmt.Printf("✅ Payment provider initialized: %s\n", provider.Name())
	}

	// Initialize scheduler provider from environment
	fmt.Printf("📅 Initializing scheduler provider...\n")
	if provider, err := integration.CreateSchedulerProvider(); err != nil {
		fmt.Printf("⚠️ Failed to initialize scheduler provider: %v\n", err)
	} else if provider != nil {
		c.services.Scheduler = provider
		fmt.Printf("✅ Scheduler provider initialized: %s\n", provider.Name())
	}

	// Initialize tabular provider from environment (Google Sheets, etc.)
	fmt.Printf("📊 Initializing tabular provider...\n")
	if provider, err := integration.CreateTabularProvider(); err != nil {
		fmt.Printf("⚠️ Failed to initialize tabular provider: %v\n", err)
	} else if provider != nil {
		c.services.Tabular = provider
		fmt.Printf("✅ Tabular provider initialized: %s\n", provider.Name())
	}

	// Initialize translation provider from environment (default: lyngua)
	fmt.Printf("🌐 Initializing translation provider...\n")
	if translationSvc, err := infraProviders.CreateTranslationService(); err != nil {
		fmt.Printf("⚠️ Failed to initialize translation provider: %v\n", err)
	} else if translationSvc != nil {
		c.services.Translation = &translationServiceWrapper{svc: translationSvc}
		fmt.Printf("✅ Translation provider initialized\n")
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
	// We need access to the initializers.InitializeWorkflowEngine function
	// and the required repositories.
	// This requires moving some logic from usecases.go to here.

	// First, get the workflow repositories
	workflowRepos, err := domain.NewWorkflowRepositories(c.providers.GetDatabaseProvider(), c.providers.GetDBTableConfig())
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
		engineUC, err := initializers.InitializeWorkflowEngine(workflowRepos, authSvc, txSvc, i18nSvc, idSvc, executorRegistry)
		if err != nil {
			return err
		}
		c.services.WorkflowEngine = engineUC
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
			engineUC, err := initializers.InitializeWorkflowEngine(workflowRepos, authSvc, txSvc, i18nSvc, idSvc, executorRegistry)
			if err != nil {
				return err
			}
			c.services.WorkflowEngine = engineUC
			fmt.Printf("✅ Workflow Engine initialized (lazily)\n")
			return nil
		}
	default:
		return fmt.Errorf("unknown Workflow Engine Mode: %s", c.config.WorkflowEngineMode)
	}
	return nil
}

// getServicesForInitializers is a new helper similar to the one in usecases.go
func (c *Container) getServicesForInitializers() (
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
	err error,
) {
	// This logic is duplicated from the UseCaseInitializer, which is a sign it should
	// probably be centralized, but for now we copy it here to make this work.

	// Get auth service from provider manager
	if authProvider := c.providers.GetAuthProvider(); authProvider != nil {
		if authService, ok := authProvider.(ports.AuthorizationService); ok {
			authSvc = authService
		}
	}
	// Fallback to mock if needed
	if authSvc == nil {
		authSvc, _ = c.services.Auth.(ports.AuthorizationService)
	}

	// Get ID service from provider manager
	if idProvider := c.providers.GetIDProvider(); idProvider != nil {
		if idWrapper, ok := idProvider.(interface{ GetIDService() ports.IDService }); ok {
			idSvc = idWrapper.GetIDService()
		}
	}
	if idSvc == nil {
		idSvc, _ = c.services.IDGen.(ports.IDService)
	}

	txSvc, _ = c.services.Transaction.(ports.TransactionService)

	// Extract TranslationService from wrapper or direct assignment
	if wrapper, ok := c.services.Translation.(*translationServiceWrapper); ok {
		i18nSvc = wrapper.svc
	} else {
		i18nSvc, _ = c.services.Translation.(ports.TranslationService)
	}

	return authSvc, txSvc, i18nSvc, idSvc, nil
}

// GetProviderManager returns the provider manager
func (c *Container) GetProviderManager() *providers.Manager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.providers
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

// GetDBTableConfig returns the database table configuration directly
func (c *Container) GetDBTableConfig() *registry.DatabaseTableConfig {
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

	return ops
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
