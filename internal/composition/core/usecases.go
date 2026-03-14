package core

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/providers"

	// Application use cases aggregate
	"github.com/erniealice/espyna-golang/internal/application/usecases"

	// Application ports (for service interfaces)
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Infrastructure adapters for mock services
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"

	// Domain use cases (for proper initialization)
	"github.com/erniealice/espyna-golang/internal/application/usecases/common"
	"github.com/erniealice/espyna-golang/internal/application/usecases/entity"
	"github.com/erniealice/espyna-golang/internal/application/usecases/event"
	"github.com/erniealice/espyna-golang/internal/application/usecases/expenditure"
	"github.com/erniealice/espyna-golang/internal/application/usecases/integration"
	"github.com/erniealice/espyna-golang/internal/application/usecases/inventory"
	"github.com/erniealice/espyna-golang/internal/application/usecases/ledger"
	"github.com/erniealice/espyna-golang/internal/application/usecases/operation"
	"github.com/erniealice/espyna-golang/internal/application/usecases/product"
	"github.com/erniealice/espyna-golang/internal/application/usecases/revenue"
	"github.com/erniealice/espyna-golang/internal/application/usecases/subscription"
	"github.com/erniealice/espyna-golang/internal/application/usecases/treasury"
	"github.com/erniealice/espyna-golang/internal/application/usecases/workflow"

	domain "github.com/erniealice/espyna-golang/internal/composition/providers/domain"

	// Composition initializers (for domain-specific wiring)
	"github.com/erniealice/espyna-golang/internal/composition/core/initializers"
)

// UseCaseInitializer handles the initialization of all use cases across different domains
type UseCaseInitializer struct {
	providerManager *providers.Manager
}

// NewUseCaseInitializer creates a new use case initializer
func NewUseCaseInitializer(providerManager *providers.Manager) *UseCaseInitializer {
	return &UseCaseInitializer{
		providerManager: providerManager,
	}
}

// InitializeAll initializes all use cases across all 7 domains
// Each domain is initialized independently - if one fails, only that domain gets an empty struct
func (uci *UseCaseInitializer) InitializeAll(container *Container) error {
	// Initialize each domain independently with graceful degradation per domain

	// Common domain must be initialized first as it provides cross-domain dependencies (Attribute)
	commonUC, err := uci.initializeCommonUseCases(container)
	if err != nil {
		// Only Common domain fails - use empty struct for this domain only
		commonUC = &common.CommonUseCases{}
	}

	entityUC, err := uci.initializeEntityUseCases(container)
	if err != nil {
		// Only Entity domain fails - use empty struct for this domain only
		entityUC = &entity.EntityUseCases{}
	}

	eventUC, err := uci.initializeEventUseCases(container)
	if err != nil {
		// Only Event domain fails - use empty struct for this domain only
		eventUC = &event.EventUseCases{}
	}

	ledgerUC, err := uci.initializeLedgerUseCases(container)
	if err != nil {
		ledgerUC = &ledger.LedgerUseCases{}
	}

	operationUC, err := uci.initializeOperationUseCases(container)
	if err != nil {
		operationUC = &operation.OperationUseCases{}
	}

	treasuryUC, err := uci.initializeTreasuryUseCases(container)
	if err != nil {
		treasuryUC = &treasury.TreasuryUseCases{}
	}

	productUC, err := uci.initializeProductUseCases(container)
	if err != nil {
		fmt.Printf("FATAL: Product domain initialization failed: %v\n", err)
		fmt.Printf("  The application cannot start without Product use cases.\n")
		fmt.Printf("  Check that PostgreSQL is running and the database schema includes product tables.\n")
		panic(fmt.Sprintf("product domain initialization failed: %v", err))
	}

	revenueUC, err := uci.initializeRevenueUseCases(container)
	if err != nil {
		revenueUC = &revenue.RevenueUseCases{}
	}

	expenditureUC, err := uci.initializeExpenditureUseCases(container)
	if err != nil {
		expenditureUC = &expenditure.ExpenditureUseCases{}
	}

	inventoryUC, err := uci.initializeInventoryUseCases(container)
	if err != nil {
		inventoryUC = &inventory.InventoryUseCases{}
	}

	subscriptionUC, err := uci.initializeSubscriptionUseCases(container)
	if err != nil {
		subscriptionUC = &subscription.SubscriptionUseCases{}
	}

	workflowUC, err := uci.initializeWorkflowUseCases(container)
	if err != nil {
		workflowUC = &workflow.WorkflowUseCases{}
	}

	// Initialize integration use cases (email, payment providers, etc.)
	// These are provider-based use cases, not domain-based
	integrationUC := uci.initializeIntegrationUseCases(container)

	// Create aggregate with successfully initialized domains
	aggregate := usecases.NewAggregate(
		commonUC,
		entityUC,
		eventUC,
		expenditureUC,
		inventoryUC,
		ledgerUC,
		operationUC,
		treasuryUC,
		productUC,
		revenueUC,
		subscriptionUC,
		workflowUC,
		integrationUC,
	)
	container.useCases = aggregate

	return nil
}

// Domain initializer methods - one for each of the 7 domains

// initializeCommonUseCases initializes Common domain use cases (1 entity: Attribute)
// Common domain provides cross-domain dependencies used by Entity, Subscription, Product, and Treasury domains
func (uci *UseCaseInitializer) initializeCommonUseCases(container *Container) (*common.CommonUseCases, error) {
	fmt.Printf("🔧 Initializing Common use cases...\n")

	repos, err := domain.NewCommonRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Common repositories not yet implemented: %v\n", err)
		fmt.Printf("📋 Common domain includes: Attribute (cross-domain dependency)\n")
		// Don't return error - return empty struct for graceful degradation
		return &common.CommonUseCases{}, nil
	}
	fmt.Printf("✅ Got common repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	// Use composition initializer to wire everything together
	commonUseCases, err := initializers.InitializeCommon(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize common use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Common domain initialized successfully: %v\n", commonUseCases != nil)

	return commonUseCases, nil
}

// initializeEntityUseCases initializes Entity domain use cases (16 entities)
func (uci *UseCaseInitializer) initializeEntityUseCases(container *Container) (*entity.EntityUseCases, error) {
	fmt.Printf("👥 Initializing Entity use cases...\n")

	repos, err := domain.NewEntityRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Entity database provider not available: %v\n", err)
		// Don't return error - return empty struct for graceful degradation
		return &entity.EntityUseCases{}, nil
	}
	fmt.Printf("✅ Got entity repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	// Use composition initializer to wire everything together
	entityUseCases, err := initializers.InitializeEntity(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize entity use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Entity domain initialized successfully: %v\n", entityUseCases != nil)

	return entityUseCases, nil
}

// initializeEventUseCases initializes Event domain use cases (2 entities)
func (uci *UseCaseInitializer) initializeEventUseCases(container *Container) (*event.EventUseCases, error) {
	fmt.Printf("🗓️  Initializing Event use cases...\n")

	repos, err := domain.NewEventRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("❌ Failed to get event repositories: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got event repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	// Use composition initializer to wire everything together
	eventUseCases, err := initializers.InitializeEvent(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize event use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Event domain initialized successfully: %v\n", eventUseCases != nil)

	return eventUseCases, nil
}

// initializeLedgerUseCases initializes Ledger domain use cases (document template)
func (uci *UseCaseInitializer) initializeLedgerUseCases(container *Container) (*ledger.LedgerUseCases, error) {
	fmt.Printf("📄 Initializing Ledger use cases...\n")

	repos, err := domain.NewLedgerRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Ledger database provider not available: %v\n", err)
		return &ledger.LedgerUseCases{}, nil
	}
	fmt.Printf("✅ Got ledger repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	ledgerUseCases, err := initializers.InitializeLedger(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize ledger use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Ledger domain initialized successfully: %v\n", ledgerUseCases != nil)

	return ledgerUseCases, nil
}

// initializeOperationUseCases initializes Operation domain use cases.
func (uci *UseCaseInitializer) initializeOperationUseCases(container *Container) (*operation.OperationUseCases, error) {
	fmt.Printf("⚙️ Initializing Operation use cases...\n")

	repos, err := domain.NewOperationRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Operation database provider not available: %v\n", err)
		return &operation.OperationUseCases{}, nil
	}
	fmt.Printf("✅ Got operation repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	operationUseCases, err := initializers.InitializeOperation(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize operation use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Operation domain initialized successfully: %v\n", operationUseCases != nil)

	return operationUseCases, nil
}

// initializeTreasuryUseCases initializes Treasury domain use cases (legacy entities removed)
func (uci *UseCaseInitializer) initializeTreasuryUseCases(container *Container) (*treasury.TreasuryUseCases, error) {
	fmt.Printf("💳 Initializing Treasury use cases...\n")

	repos, err := domain.NewTreasuryRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("❌ Failed to get treasury repositories: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got treasury repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	// Use composition initializer to wire everything together
	treasuryUseCases, err := initializers.InitializeTreasury(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize treasury use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Treasury domain initialized successfully: %v\n", treasuryUseCases != nil)

	return treasuryUseCases, nil
}

// initializeProductUseCases initializes Product domain use cases (8 entities)
func (uci *UseCaseInitializer) initializeProductUseCases(container *Container) (*product.ProductUseCases, error) {
	fmt.Printf("🛍️  Initializing Product use cases...\n")

	repos, err := domain.NewProductRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("❌ Failed to get product repositories: %v\n", err)
		fmt.Printf("  Database provider: %v, Table config: %v\n", uci.providerManager.GetDatabaseProvider() != nil, uci.providerManager.GetDBTableConfig() != nil)
		return nil, fmt.Errorf("product repository creation failed (database unavailable): %w", err)
	}
	fmt.Printf("✅ Got product repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	// Use composition initializer to wire everything together
	productUseCases, err := initializers.InitializeProduct(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize product use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Product domain initialized successfully: %v\n", productUseCases != nil)

	return productUseCases, nil
}

// initializeRevenueUseCases initializes Revenue domain use cases
func (uci *UseCaseInitializer) initializeRevenueUseCases(container *Container) (*revenue.RevenueUseCases, error) {
	fmt.Printf("💰 Initializing Revenue use cases...\n")

	repos, err := domain.NewRevenueRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Revenue database provider not available: %v\n", err)
		return &revenue.RevenueUseCases{}, nil
	}
	fmt.Printf("✅ Got revenue repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	revenueUseCases, err := initializers.InitializeRevenue(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize revenue use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Revenue domain initialized successfully: %v\n", revenueUseCases != nil)

	return revenueUseCases, nil
}

// initializeExpenditureUseCases initializes Expenditure domain use cases (4 entities)
func (uci *UseCaseInitializer) initializeExpenditureUseCases(container *Container) (*expenditure.ExpenditureUseCases, error) {
	fmt.Printf("💸 Initializing Expenditure use cases...\n")

	repos, err := domain.NewExpenditureRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Expenditure database provider not available: %v\n", err)
		return &expenditure.ExpenditureUseCases{}, nil
	}
	fmt.Printf("✅ Got expenditure repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	expenditureUseCases, err := initializers.InitializeExpenditure(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize expenditure use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Expenditure domain initialized successfully: %v\n", expenditureUseCases != nil)

	return expenditureUseCases, nil
}

// initializeInventoryUseCases initializes Inventory domain use cases (6 entities)
func (uci *UseCaseInitializer) initializeInventoryUseCases(container *Container) (*inventory.InventoryUseCases, error) {
	fmt.Printf("📦 Initializing Inventory use cases...\n")

	repos, err := domain.NewInventoryRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Inventory database provider not available: %v\n", err)
		// Don't return error - return empty struct for graceful degradation
		return &inventory.InventoryUseCases{}, nil
	}
	fmt.Printf("✅ Got inventory repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	// Use composition initializer to wire everything together
	inventoryUseCases, err := initializers.InitializeInventory(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize inventory use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Inventory domain initialized successfully: %v\n", inventoryUseCases != nil)

	return inventoryUseCases, nil
}

// initializeSubscriptionUseCases initializes Subscription domain use cases (6 entities)
func (uci *UseCaseInitializer) initializeSubscriptionUseCases(container *Container) (*subscription.SubscriptionUseCases, error) {
	fmt.Printf("💰 Initializing Subscription use cases...\n")

	subscriptionRepos, err := domain.NewSubscriptionRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("❌ Failed to get subscription repositories: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got subscription repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	// Use composition initializer to wire everything together
	subscriptionUseCases, err := initializers.InitializeSubscription(subscriptionRepos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize subscription use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Subscription domain initialized successfully: %v\n", subscriptionUseCases != nil)

	return subscriptionUseCases, nil
}

// initializeWorkflowUseCases initializes workflow domain use cases (6 entities)
// Note: The workflow engine is NOT initialized here - it's an orchestration concern,
// not a domain concern. The Container handles engine initialization separately.
func (uci *UseCaseInitializer) initializeWorkflowUseCases(container *Container) (*workflow.WorkflowUseCases, error) {
	fmt.Printf("🔄 Initializing Workflow use cases...\n")

	repos, err := domain.NewWorkflowRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("❌ Failed to get workflow repositories: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got workflow repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	// Use composition initializer to wire everything together
	workflowUseCases, err := initializers.InitializeWorkflow(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize workflow use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Workflow domain initialized successfully: %v\n", workflowUseCases != nil)

	return workflowUseCases, nil
}

// getServices is a helper to extract services from container with proper type checking
// Returns nil services gracefully when not initialized (services are optional for now)
func (uci *UseCaseInitializer) getServices(container *Container) (
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
	err error,
) {
	// Services are optional - translation service handles nil gracefully by returning fallback strings
	// Auth and transaction services should be checked by use cases before use

	// Get auth service from provider manager or create mock service as fallback
	if authProvider := uci.providerManager.GetAuthProvider(); authProvider != nil {
		if authService, ok := authProvider.(ports.AuthorizationService); ok {
			authSvc = authService
			fmt.Printf("🔐 Using authorization service from provider: %T\n", authSvc)
		} else {
			// Fallback to mock auth service if provider doesn't implement the interface
			authSvc = mockAuth.NewAllowAllAuth()
			// fmt.Printf("🔓 Created allow-all authorization service (provider fallback): %T\n", authSvc)
		}
	} else {
		// No auth provider configured, use mock service
		authSvc = mockAuth.NewAllowAllAuth()
		// fmt.Printf("🔓 Created allow-all authorization service (no provider): %T\n", authSvc)
	}

	// Get ID service from provider manager
	if idProvider := uci.providerManager.GetIDProvider(); idProvider != nil {
		// Check if the provider has a GetIDService method (IDProviderWrapper)
		if idWrapper, ok := idProvider.(interface{ GetIDService() ports.IDService }); ok {
			idSvc = idWrapper.GetIDService()
			// fmt.Printf("🆔 Using ID service from provider: %T - %s\n", idSvc, idSvc.GetProviderInfo())
		} else if idService, ok := idProvider.(ports.IDService); ok {
			// Fallback: provider directly implements IDService
			idSvc = idService
			fmt.Printf("🆔 Using ID service (direct): %T - %s\n", idSvc, idSvc.GetProviderInfo())
		} else {
			// Fallback to noop if provider doesn't implement the interface
			idSvc = ports.NewNoOpIDService()
			fmt.Printf("🆔 Created NoOp ID service (provider fallback): %T\n", idSvc)
		}
	} else {
		// No ID provider configured, use noop service
		idSvc = ports.NewNoOpIDService()
		fmt.Printf("🆔 Created NoOp ID service (no provider): %T\n", idSvc)
	}

	txSvc, _ = container.services.Transaction.(ports.TransactionService)

	// Extract TranslationService from wrapper or direct assignment
	if wrapper, ok := container.services.Translation.(*translationServiceWrapper); ok {
		i18nSvc = wrapper.svc
	} else {
		i18nSvc, _ = container.services.Translation.(ports.TranslationService)
	}

	return authSvc, txSvc, i18nSvc, idSvc, nil
}

// initializeIntegrationUseCases initializes integration use cases (email, payment, scheduler providers)
// These are external provider integrations, not domain-based use cases
func (uci *UseCaseInitializer) initializeIntegrationUseCases(container *Container) *integration.IntegrationUseCases {
	fmt.Printf("🔌 Initializing Integration use cases...\n")

	// Get email provider from container (already typed as ports.EmailProvider)
	emailProvider := container.services.Email
	if emailProvider != nil {
		fmt.Printf("📧 Got email provider: %s\n", emailProvider.Name())
	}

	// Get payment provider from container (already typed as ports.PaymentProvider)
	paymentProvider := container.services.Payment
	if paymentProvider != nil {
		fmt.Printf("💳 Got payment provider: %s\n", paymentProvider.Name())
	}

	// Get scheduler provider from container (already typed as ports.SchedulerProvider)
	schedulerProvider := container.services.Scheduler
	if schedulerProvider != nil {
		fmt.Printf("📅 Got scheduler provider: %s\n", schedulerProvider.Name())
	}

	// Get tabular provider from container (already typed as ports.TabularSourceProvider)
	tabularProvider := container.services.Tabular
	if tabularProvider != nil {
		fmt.Printf("📊 Got tabular provider: %s\n", tabularProvider.Name())
	}

	// Get integration payment repository from database provider
	var integrationPaymentRepo domain.IntegrationPaymentRepository
	dbProvider := uci.providerManager.GetDatabaseProvider()
	tableConfig := uci.providerManager.GetDBTableConfig()
	if dbProvider != nil && tableConfig != nil {
		repo, err := domain.NewIntegrationPaymentRepository(dbProvider, tableConfig)
		if err != nil {
			fmt.Printf("⚠️  Failed to create integration payment repository: %v\n", err)
		} else {
			integrationPaymentRepo = repo
			fmt.Printf("📝 Got integration payment repository\n")
		}
	}

	// Create integration use cases with available providers
	integrationUC := integration.NewIntegrationUseCases(paymentProvider, emailProvider, schedulerProvider, tabularProvider, integrationPaymentRepo)

	if integrationUC != nil {
		routeCount := 0
		if integrationUC.Email != nil {
			routeCount += 3 // send, health, capabilities
		}
		if integrationUC.Payment != nil {
			routeCount += 6 // webhook, log, checkout, status, health, capabilities
		}
		if integrationUC.Scheduler != nil {
			routeCount += 10 // create, cancel, get, list, availability, webhook, eventTypes, getEventType, health, capabilities
		}
		if integrationUC.Tabular != nil {
			routeCount += 12 // read, write, write-simple, update, delete, search, schema, source, tables, batch, health, capabilities
		}
		fmt.Printf("✅ Integration use cases initialized (email: %v, payment: %v, scheduler: %v, tabular: %v, routes: %d)\n",
			integrationUC.Email != nil, integrationUC.Payment != nil, integrationUC.Scheduler != nil, integrationUC.Tabular != nil, routeCount)
	} else {
		fmt.Printf("⚠️ No integration providers available\n")
	}

	return integrationUC
}
