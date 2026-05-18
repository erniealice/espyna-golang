// usecases.go defines UseCaseInitializer, the orchestrator that wires domain
// repositories into use cases during application startup.
//
// For each domain, it:
//  1. Calls domain.New{Domain}Repositories() to create repos from the registry
//     (no switch statements — repos come from self-registered factory functions).
//  2. Calls initializers.Initialize{Domain}() to build use cases from repos + services.
//
// The providerManager supplies: database provider, table config, auth, ID, and
// translation services.
//
// Initialization order: Common domain is initialized first because it provides
// cross-domain dependencies (e.g., Attribute). All other domains are independent.
//
// Graceful degradation: most domains fall back to empty structs on failure, so the
// app can start with partial functionality. The Product domain is the exception —
// it panics on failure because the application cannot operate without it.
package core

import (
	"context"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/providers"

	// Application use cases aggregate
	"github.com/erniealice/espyna-golang/internal/application/usecases"

	// Application ports (for service interfaces)
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Infrastructure adapters for mock services
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"

	// Domain use cases (for proper initialization)
	"github.com/erniealice/espyna-golang/internal/application/usecases/asset"
	"github.com/erniealice/espyna-golang/internal/application/usecases/auth"
	"github.com/erniealice/espyna-golang/internal/application/usecases/common"
	"github.com/erniealice/espyna-golang/internal/application/usecases/entity"
	"github.com/erniealice/espyna-golang/internal/application/usecases/event"
	"github.com/erniealice/espyna-golang/internal/application/usecases/expenditure"
	"github.com/erniealice/espyna-golang/internal/application/usecases/finance"
	"github.com/erniealice/espyna-golang/internal/application/usecases/fulfillment"
	"github.com/erniealice/espyna-golang/internal/application/usecases/funding"
	"github.com/erniealice/espyna-golang/internal/application/usecases/integration"
	"github.com/erniealice/espyna-golang/internal/application/usecases/inventory"
	"github.com/erniealice/espyna-golang/internal/application/usecases/ledger"
	"github.com/erniealice/espyna-golang/internal/application/usecases/operation"
	jobUseCase "github.com/erniealice/espyna-golang/internal/application/usecases/operation/job"
	"github.com/erniealice/espyna-golang/internal/application/usecases/payroll"
	"github.com/erniealice/espyna-golang/internal/application/usecases/procurement"
	"github.com/erniealice/espyna-golang/internal/application/usecases/product"
	"github.com/erniealice/espyna-golang/internal/application/usecases/revenue"
	"github.com/erniealice/espyna-golang/internal/application/usecases/subscription"
	subscriptionUseCase "github.com/erniealice/espyna-golang/internal/application/usecases/subscription/subscription"

	"github.com/erniealice/espyna-golang/internal/application/usecases/tax"
	"github.com/erniealice/espyna-golang/internal/application/usecases/tenancy"
	"github.com/erniealice/espyna-golang/internal/application/usecases/treasury"
	"github.com/erniealice/espyna-golang/internal/application/usecases/workflow"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"

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

	expenditureUC, err := uci.initializeExpenditureUseCases(container, treasuryUC)
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

	// 2026-04-30 cyclic-subscription-jobs plan §5.2 — wire the recognize-
	// revenue piggyback. After both subscriptionUC and revenueUC have been
	// initialized, install the MaterializeInstanceJobsForSubscriptionInvoker
	// adapter onto the RecognizeRevenueFromSubscription use case so that
	// successful revenue recognition for cyclic plans triggers cycle-Job
	// materialisation.
	//
	// Failure semantics — non-fatal: see RecognizeRevenueFromSubscriptionServices
	// .MaterializeInstanceJobsForSubscription doc for the full contract.
	if subscriptionUC != nil && subscriptionUC.MaterializeInstanceJobsForSubscription != nil &&
		revenueUC != nil && revenueUC.Revenue != nil &&
		revenueUC.Revenue.RecognizeRevenueFromSubscription != nil {
		revenueUC.Revenue.RecognizeRevenueFromSubscription.SetMaterializeInstanceJobsForSubscription(
			&materializeInstanceJobsAdapter{uc: subscriptionUC.MaterializeInstanceJobsForSubscription},
		)
		fmt.Printf("✅ Recognize-revenue piggyback wired (cycle-Job spawn on successful recognition)\n")
	}

	workflowUC, err := uci.initializeWorkflowUseCases(container)
	if err != nil {
		workflowUC = &workflow.WorkflowUseCases{}
	}

	payrollUC, err := uci.initializePayrollUseCases(container)
	if err != nil {
		payrollUC = &payroll.PayrollUseCases{}
	}

	procurementUC, err := uci.initializeProcurementUseCases(container)
	if err != nil {
		procurementUC = &procurement.ProcurementUseCases{}
	}

	fulfillmentUC, err := uci.initializeFulfillmentUseCases(container)
	if err != nil {
		fulfillmentUC = &fulfillment.UseCases{}
	}

	assetUC, err := uci.initializeAssetUseCases(container)
	if err != nil {
		assetUC = &asset.AssetUseCases{}
	}

	taxUC, err := uci.initializeTaxUseCases(container)
	if err != nil {
		taxUC = &tax.TaxUseCases{}
	}

	// 2026-05-10 tax-integration Phase 4 — wire ComputeTaxesForRevenue into the
	// revenue domain. Both revenueUC and taxUC must be initialized first.
	// Non-fatal: if either is nil/empty, the respective hook remains a no-op.
	if taxUC != nil && taxUC.ComputeTaxes != nil && revenueUC != nil && revenueUC.Revenue != nil {
		computeUC := taxUC.ComputeTaxes.ComputeTaxesForRevenue
		if computeUC != nil {
			// Wire into RecomputeTaxes admin use case (Phase E).
			if revenueUC.Revenue.RecomputeTaxes != nil {
				revenueUC.Revenue.RecomputeTaxes.SetComputeTaxes(computeUC)
				fmt.Printf("Tax compute wired into revenue domain (ComputeTaxesForRevenue → RecomputeTaxes)\n")
			}
			// Wire into RecognizeRevenueFromSubscription post-persist hook (Phase C).
			if revenueUC.Revenue.RecognizeRevenueFromSubscription != nil {
				revenueUC.Revenue.RecognizeRevenueFromSubscription.SetComputeTaxes(computeUC)
				fmt.Printf("Tax compute wired into revenue domain (ComputeTaxesForRevenue → RecognizeRevenueFromSubscription)\n")
			}
			// Wire into CreateRevenue post-persist hook (Phase D).
			if revenueUC.Revenue.CreateRevenue != nil {
				revenueUC.Revenue.CreateRevenue.SetComputeTaxes(computeUC)
				fmt.Printf("Tax compute wired into revenue domain (ComputeTaxesForRevenue → CreateRevenue)\n")
			}
		}
	}

	financeUC, err := uci.initializeFinanceUseCases(container)
	if err != nil {
		financeUC = &finance.FinanceUseCases{}
	}

	tenancyUC, err := uci.initializeTenancyUseCases(container)
	if err != nil {
		tenancyUC = &tenancy.TenancyUseCases{}
	}

	fundingUC, err := uci.initializeFundingUseCases(container)
	if err != nil {
		fundingUC = &funding.FundingUseCases{}
	}

	// Initialize integration use cases (email, payment providers, etc.)
	// These are provider-based use cases, not domain-based
	integrationUC := uci.initializeIntegrationUseCases(container)

	// Identity-lifecycle use cases — must run after Entity so Session + User
	// repos are already registered with the database provider.
	authUC, err := uci.initializeAuthUseCases(container)
	if err != nil {
		authUC = &auth.UseCases{}
	}

	// Create aggregate with successfully initialized domains
	aggregate := usecases.NewAggregate(
		authUC,
		commonUC,
		entityUC,
		eventUC,
		expenditureUC,
		financeUC,
		fulfillmentUC,
		fundingUC,
		inventoryUC,
		ledgerUC,
		operationUC,
		payrollUC,
		procurementUC,
		taxUC,
		tenancyUC,
		treasuryUC,
		productUC,
		revenueUC,
		subscriptionUC,
		workflowUC,
		integrationUC,
		assetUC,
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

// initializeAuthUseCases initializes identity-lifecycle use cases
// (authenticate_session, issue_session, invalidate_session).
//
// These share the Session + User proto repositories used by the Entity
// domain, but produce distinct orchestration-level use cases that
// middleware/handlers depend on instead of poking at session rows.
func (uci *UseCaseInitializer) initializeAuthUseCases(container *Container) (*auth.UseCases, error) {
	fmt.Printf("🔐 Initializing Auth use cases...\n")

	repos, err := domain.NewEntityRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Auth repositories not available (Entity unavailable): %v\n", err)
		return &auth.UseCases{}, nil
	}

	_, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}

	authUseCases, err := initializers.InitializeAuth(repos, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize auth use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Auth domain initialized successfully: %v\n", authUseCases != nil)

	return authUseCases, nil
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
//
// Cross-domain reads (Subscription / PricePlan / ProductPricePlan / BillingEvent)
// are sourced from the Subscription domain provider so the
// MaterializeBillingEventsForJob use case and the OnJobPhaseCompleted hook
// can fire (milestone-billing plan §3). Failure to load the subscription
// provider degrades to nil — the operation use cases still wire up, just
// without the cross-domain branches.
func (uci *UseCaseInitializer) initializeOperationUseCases(container *Container) (*operation.OperationUseCases, error) {
	fmt.Printf("⚙️ Initializing Operation use cases...\n")

	repos, err := domain.NewOperationRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Operation database provider not available: %v\n", err)
		return &operation.OperationUseCases{}, nil
	}
	fmt.Printf("✅ Got operation repositories\n")

	subRepos, subErr := domain.NewSubscriptionRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if subErr != nil {
		fmt.Printf("⚠️  Subscription provider unavailable for operation cross-domain wiring: %v\n", subErr)
		subRepos = nil
	}

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	operationUseCases, err := initializers.InitializeOperation(repos, subRepos, authSvc, txSvc, i18nSvc, idSvc)
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

	// 20260517-advance-cash-events Plan B Phase 2 — Treasury aggregator needs
	// the Revenue + ExpenseRecognition repos for AmortizeAdvance* use cases.
	// We construct the Revenue + Expenditure providers here just to pull those
	// two repositories — the full aggregates are built independently
	// downstream. Non-fatal: when unavailable, AmortizeAdvance* still wire but
	// return errors at call time.
	if revRepos, revErr := domain.NewRevenueRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig()); revErr == nil && revRepos != nil {
		repos.Revenue = revRepos.Revenue
	} else if revErr != nil {
		fmt.Printf("⚠️  Treasury: revenue repo unavailable for AmortizeAdvanceCollection: %v\n", revErr)
	}
	if expRepos, expErr := domain.NewExpenditureRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig()); expErr == nil && expRepos != nil {
		repos.ExpenseRecognition = expRepos.ExpenseRecognition
	} else if expErr != nil {
		fmt.Printf("⚠️  Treasury: expense_recognition repo unavailable for AmortizeAdvanceDisbursement: %v\n", expErr)
	}

	// 20260517-advance-cash-events Plan B Phase 7 — pull BillingEvent from the
	// subscription provider block so the selling-side MILESTONE recognize use
	// case (RecognizeMilestoneAdvanceCollection) is actually constructed.
	// Without this the four-AND nil-guard in treasury.NewUseCases sees
	// repos.BillingEvent == nil and silently leaves the use case as nil.
	if subRepos, subErr := domain.NewSubscriptionRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig()); subErr == nil && subRepos != nil {
		repos.BillingEvent = subRepos.BillingEvent
	} else if subErr != nil {
		fmt.Printf("⚠️  Treasury: billing_event repo unavailable for RecognizeMilestoneAdvanceCollection: %v\n", subErr)
	}

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

// initializeExpenditureUseCases initializes Expenditure domain use cases (4 entities).
//
// treasuryUseCases is the already-constructed treasury aggregate (built earlier
// in InitializeAllUseCases). Plan A Phase 4 GenerateExpenseRun composes the
// cross-domain AmortizeAdvanceDisbursement from it. Pass nil to degrade.
func (uci *UseCaseInitializer) initializeExpenditureUseCases(container *Container, treasuryUseCases *treasury.TreasuryUseCases) (*expenditure.ExpenditureUseCases, error) {
	fmt.Printf("💸 Initializing Expenditure use cases...\n")

	repos, err := domain.NewExpenditureRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Expenditure database provider not available: %v\n", err)
		return &expenditure.ExpenditureUseCases{}, nil
	}
	fmt.Printf("✅ Got expenditure repositories\n")

	// Cross-domain: procurement repos for SupplierSubscription workspace validation
	// on RecognizeFromExpenditure (buying/selling parity 2026-05-09). Non-fatal.
	procurementRepos, procErr := domain.NewProcurementRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if procErr != nil {
		fmt.Printf("⚠️  Expenditure: procurement repos unavailable for supplier subscription validation: %v\n", procErr)
		procurementRepos = nil
	} else {
		repos.SupplierSubscription = procurementRepos.SupplierSubscription
		// 20260517-expense-run Plan A Phase 2 — CostPlan + SupplierProductCostPlan
		// for RecognizeExpenseFromSupplierSubscription.
		repos.CostPlan = procurementRepos.CostPlan
		repos.SupplierProductCostPlan = procurementRepos.SupplierProductCostPlan
	}

	// 20260517-expense-run Plan A Phase 2 — TreasuryDisbursement for advance
	// enumeration in ListExpenseRunCandidates. Non-fatal; the candidate list
	// degrades to subscription-only when unavailable.
	treasuryRepos, trErr := domain.NewTreasuryRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if trErr != nil {
		fmt.Printf("⚠️  Expenditure: treasury repos unavailable for advance enumeration: %v\n", trErr)
	} else if treasuryRepos != nil {
		repos.TreasuryDisbursement = treasuryRepos.Disbursement
	}

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	expenditureUseCases, err := initializers.InitializeExpenditure(repos, authSvc, txSvc, i18nSvc, idSvc, treasuryUseCases)
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

	// Build the JobTemplateInstantiator port from
	// MaterializeJobsForSubscriptionUseCase per
	// docs/plan/20260429-auto-spawn-jobs-from-subscription/ Phases A-C.
	// Best-effort: when operation repos are unavailable (mock-only test
	// runs, environments without the operation domain), the instantiator
	// stays nil and subscription create proceeds without spawning Jobs.
	var jobTemplateInstantiator subscriptionUseCase.JobTemplateInstantiator
	operationRepos, opErr := domain.NewOperationRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if opErr != nil {
		fmt.Printf("⚠️  MaterializeJobsForSubscription unavailable (operation repos: %v)\n", opErr)
	} else {
		// Build the milestone-billing invoker from the operation/job use case.
		mbeFor := jobUseCase.NewMaterializeBillingEventsForJobUseCase(
			jobUseCase.MaterializeBillingEventsForJobRepositories{
				Job:              operationRepos.Job,
				JobTemplatePhase: operationRepos.JobTemplatePhase,
				JobPhase:         operationRepos.JobPhase,
				BillingEvent:     subscriptionRepos.BillingEvent,
				Subscription:     subscriptionRepos.Subscription,
				PricePlan:        subscriptionRepos.PricePlan,
				ProductPricePlan: subscriptionRepos.ProductPricePlan,
			},
			jobUseCase.MaterializeBillingEventsForJobServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idSvc,
			},
		)
		mjfs := subscriptionUseCase.NewMaterializeJobsForSubscriptionUseCase(
			subscriptionUseCase.MaterializeJobsForSubscriptionRepositories{
				Subscription:        subscriptionRepos.Subscription,
				PricePlan:           subscriptionRepos.PricePlan,
				Plan:                subscriptionRepos.Plan,
				JobTemplate:         operationRepos.JobTemplate,
				JobTemplatePhase:    operationRepos.JobTemplatePhase,
				JobTemplateTask:     operationRepos.JobTemplateTask,
				JobTemplateRelation: operationRepos.JobTemplateRelation,
				Job:                 operationRepos.Job,
				JobPhase:            operationRepos.JobPhase,
				JobTask:             operationRepos.JobTask,
			},
			subscriptionUseCase.MaterializeJobsForSubscriptionServices{
				AuthorizationService:           authSvc,
				TransactionService:             txSvc,
				TranslationService:             i18nSvc,
				IDService:                      idSvc,
				MaterializeBillingEventsForJob: &materializeBillingEventsAdapter{uc: mbeFor},
			},
		)
		jobTemplateInstantiator = &subscriptionUseCase.MaterializeJobsForSubscriptionInstantiator{UseCase: mjfs}
		fmt.Printf("✅ MaterializeJobsForSubscription wired\n")
	}

	// 2026-04-30 cyclic-subscription-jobs plan §3 — wire the
	// MaterializeInstanceJobsForSubscriptionUseCase. Shares the same repos
	// + services as MaterializeJobsForSubscription. Captured here so the
	// downstream assignment onto the subscriptionUseCases aggregator
	// (after InitializeSubscription) can bind it.
	var instanceJobsUC *subscriptionUseCase.MaterializeInstanceJobsForSubscriptionUseCase
	if opErr == nil {
		instanceJobsUC = subscriptionUseCase.NewMaterializeInstanceJobsForSubscriptionUseCase(
			subscriptionUseCase.MaterializeInstanceJobsForSubscriptionRepositories{
				Subscription:        subscriptionRepos.Subscription,
				PricePlan:           subscriptionRepos.PricePlan,
				Plan:                subscriptionRepos.Plan,
				JobTemplate:         operationRepos.JobTemplate,
				JobTemplatePhase:    operationRepos.JobTemplatePhase,
				JobTemplateTask:     operationRepos.JobTemplateTask,
				JobTemplateRelation: operationRepos.JobTemplateRelation,
				Job:                 operationRepos.Job,
				JobPhase:            operationRepos.JobPhase,
				JobTask:             operationRepos.JobTask,
				// AD_HOC × PER_OCCURRENCE spawns paired BillingEvents.
				// See ad-hoc-subscription-billing plan §3.2.
				BillingEvent: subscriptionRepos.BillingEvent,
			},
			subscriptionUseCase.MaterializeInstanceJobsForSubscriptionServices{
				AuthorizationService: authSvc,
				TransactionService:   txSvc,
				TranslationService:   i18nSvc,
				IDService:            idSvc,
			},
		)
		fmt.Printf("✅ MaterializeInstanceJobsForSubscription wired\n")
	}

	// Use composition initializer to wire everything together. The reference
	// checker is plumbed through ports.NewNoOpReferenceChecker by default —
	// the application owner (service-admin) wires the postgres-backed
	// reference.Checker via the container path when running on postgres.
	subscriptionUseCases, err := initializers.InitializeSubscription(subscriptionRepos, authSvc, txSvc, i18nSvc, idSvc, jobTemplateInstantiator, ports.NewNoOpReferenceChecker())
	if err != nil {
		fmt.Printf("❌ Failed to initialize subscription use cases: %v\n", err)
		return nil, err
	}
	// 2026-04-29 auto-spawn-jobs-from-subscription Phase D — expose the
	// concrete use case so centymo's create-form opt-out + retroactive
	// spawn handler can call it directly.
	if subscriptionUseCases != nil && opErr == nil {
		// `mjfs` is in scope only when operation repos resolved. Capture
		// from the instantiator wrapper to keep the nil branches clean.
		if inst, ok := jobTemplateInstantiator.(*subscriptionUseCase.MaterializeJobsForSubscriptionInstantiator); ok && inst != nil {
			subscriptionUseCases.MaterializeJobsForSubscription = inst.UseCase
		}
		// 2026-04-30 cyclic-subscription-jobs Phase B — expose the cyclic
		// instance Job spawner alongside MaterializeJobsForSubscription so:
		//   - recognize-revenue piggyback (espyna's own Phase C wiring) can
		//     call it after successful revenue recognition.
		//   - Future Operations tab "Spawn this cycle now" / "Backfill" CTAs
		//     can invoke it directly via the consumer surface.
		if instanceJobsUC != nil {
			subscriptionUseCases.MaterializeInstanceJobsForSubscription = instanceJobsUC
		}
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

// initializePayrollUseCases initializes Payroll domain use cases.
// Loads payroll, entity, and expenditure repos so the orchestrator can
// fan out to employees and write payslip Expenditures.
func (uci *UseCaseInitializer) initializePayrollUseCases(container *Container) (*payroll.PayrollUseCases, error) {
	fmt.Printf("💼 Initializing Payroll use cases...\n")

	repos, err := domain.NewPayrollRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Payroll database provider not available: %v\n", err)
		return &payroll.PayrollUseCases{}, nil
	}
	fmt.Printf("✅ Got payroll repositories\n")

	// Cross-domain repos for the orchestrator. Failure to load these is non-fatal:
	// payroll CRUD still works; only Calculate/GeneratePayCycles will be unavailable.
	entityRepos, eErr := domain.NewEntityRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if eErr != nil {
		fmt.Printf("⚠️  Payroll: entity repos unavailable for orchestrator: %v\n", eErr)
		entityRepos = nil
	}
	expenditureRepos, xErr := domain.NewExpenditureRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if xErr != nil {
		fmt.Printf("⚠️  Payroll: expenditure repos unavailable for orchestrator: %v\n", xErr)
		expenditureRepos = nil
	}

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	payrollUseCases, err := initializers.InitializePayroll(repos, entityRepos, expenditureRepos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize payroll use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Payroll domain initialized successfully: %v\n", payrollUseCases != nil)

	return payrollUseCases, nil
}

// initializeFulfillmentUseCases initializes Fulfillment domain use cases.
func (uci *UseCaseInitializer) initializeFulfillmentUseCases(container *Container) (*fulfillment.UseCases, error) {
	fmt.Printf("📦 Initializing Fulfillment use cases...\n")

	repos, err := domain.NewFulfillmentRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Fulfillment database provider not available: %v\n", err)
		return &fulfillment.UseCases{}, nil
	}
	fmt.Printf("✅ Got fulfillment repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	fulfillmentUseCases, err := initializers.InitializeFulfillment(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize fulfillment use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Fulfillment domain initialized successfully: %v\n", fulfillmentUseCases != nil)

	return fulfillmentUseCases, nil
}

// initializeAssetUseCases initializes Asset domain use cases (Asset, AssetCategory).
func (uci *UseCaseInitializer) initializeAssetUseCases(container *Container) (*asset.AssetUseCases, error) {
	fmt.Printf("📦 Initializing Asset use cases...\n")

	repos, err := domain.NewAssetRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Asset database provider not available: %v\n", err)
		return &asset.AssetUseCases{}, nil
	}
	fmt.Printf("✅ Got asset repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Got services (auth: %v, tx: %v, i18n: %v, id: %v)\n", authSvc != nil, txSvc != nil, i18nSvc != nil, idSvc != nil)

	assetUseCases, err := initializers.InitializeAsset(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize asset use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Asset domain initialized successfully: %v\n", assetUseCases != nil)

	return assetUseCases, nil
}

// initializeTaxUseCases initializes Tax domain use cases (6 entities).
// Graceful degradation: returns empty struct on failure so the app starts without tax.
func (uci *UseCaseInitializer) initializeTaxUseCases(container *Container) (*tax.TaxUseCases, error) {
	fmt.Printf("Initializing Tax use cases...\n")

	repos, err := domain.NewTaxRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("WARNING: Tax database provider not available: %v\n", err)
		return &tax.TaxUseCases{}, nil
	}
	fmt.Printf("Got tax repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("ERROR: Failed to get services: %v\n", err)
		return nil, err
	}

	taxUseCases, err := initializers.InitializeTax(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize tax use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("Tax domain initialized successfully: %v\n", taxUseCases != nil)

	return taxUseCases, nil
}

// initializeFinanceUseCases initializes Finance domain use cases (1 entity: ForexRate).
// Graceful degradation: returns empty struct on failure so the app starts without finance.
func (uci *UseCaseInitializer) initializeFinanceUseCases(container *Container) (*finance.FinanceUseCases, error) {
	fmt.Printf("Initializing Finance use cases...\n")

	repos, err := domain.NewFinanceRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("WARNING: Finance database provider not available: %v\n", err)
		return &finance.FinanceUseCases{}, nil
	}
	fmt.Printf("Got finance repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("ERROR: Failed to get services: %v\n", err)
		return nil, err
	}

	financeUseCases, err := initializers.InitializeFinance(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize finance use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("Finance domain initialized successfully: %v\n", financeUseCases != nil)

	return financeUseCases, nil
}

// initializeProcurementUseCases initializes Procurement domain use cases (6 entities).
// Graceful degradation: returns empty struct on failure so the app starts without procurement.
func (uci *UseCaseInitializer) initializeProcurementUseCases(container *Container) (*procurement.ProcurementUseCases, error) {
	fmt.Printf("Initializing Procurement use cases...\n")

	repos, err := domain.NewProcurementRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("WARNING: Procurement database provider not available: %v\n", err)
		return &procurement.ProcurementUseCases{}, nil
	}
	fmt.Printf("Got procurement repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("ERROR: Failed to get services: %v\n", err)
		return nil, err
	}

	procurementUseCases, err := initializers.InitializeProcurement(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize procurement use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("Procurement domain initialized successfully: %v\n", procurementUseCases != nil)

	return procurementUseCases, nil
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

// materializeBillingEventsAdapter adapts the MaterializeBillingEventsForJob
// use case to the narrow MaterializeBillingEventsForJobInvoker interface
// consumed by MaterializeJobsForSubscription (plan §3.7). The adapter
// translates the (ctx, jobID, subscriptionID) shape into the use case's
// request struct.
type materializeBillingEventsAdapter struct {
	uc *jobUseCase.MaterializeBillingEventsForJobUseCase
}

// Execute satisfies subscriptionUseCase.MaterializeBillingEventsForJobInvoker.
func (a *materializeBillingEventsAdapter) Execute(ctx context.Context, jobID, subscriptionID string) error {
	if a == nil || a.uc == nil {
		return nil
	}
	_, err := a.uc.Execute(ctx, jobUseCase.MaterializeBillingEventsForJobRequest{
		JobID:          jobID,
		SubscriptionID: subscriptionID,
	})
	return err
}

// materializeInstanceJobsAdapter adapts the
// MaterializeInstanceJobsForSubscriptionUseCase to the narrow
// MaterializeInstanceJobsForSubscriptionInvoker interface consumed by the
// recognize-revenue piggyback (cyclic-subscription-jobs plan §5.2). The
// adapter translates the (ctx, subID, periodStart) shape into the use case's
// request struct.
type materializeInstanceJobsAdapter struct {
	uc *subscriptionUseCase.MaterializeInstanceJobsForSubscriptionUseCase
}

// Execute satisfies revenueUseCase.MaterializeInstanceJobsForSubscriptionInvoker.
//
// Errors propagate to the recognize-revenue use case which converts them into
// a non-fatal warning on the response (plan §5.2). nil-safe.
func (a *materializeInstanceJobsAdapter) Execute(ctx context.Context, subscriptionID, cyclePeriodStart string) error {
	if a == nil || a.uc == nil {
		return nil
	}
	pbReq := &subscriptionpb.MaterializeInstanceJobsForSubscriptionRequest{
		SubscriptionId: subscriptionID,
	}
	if cyclePeriodStart != "" {
		pbReq.CyclePeriodStart = &cyclePeriodStart
	}
	_, err := a.uc.Execute(ctx, pbReq)
	return err
}

// initializeTenancyUseCases initializes Tenancy domain use cases (3 entities: TenantSubscription, TenantPaymentMethod, TenantInvoice).
// Graceful degradation: returns empty struct on failure so the app starts without tenancy.
func (uci *UseCaseInitializer) initializeTenancyUseCases(container *Container) (*tenancy.TenancyUseCases, error) {
	fmt.Printf("Initializing Tenancy use cases...\n")

	repos, err := domain.NewTenancyRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("WARNING: Tenancy database provider not available: %v\n", err)
		return &tenancy.TenancyUseCases{}, nil
	}
	fmt.Printf("Got tenancy repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("ERROR: Failed to get services: %v\n", err)
		return nil, err
	}

	tenancyUseCases, err := initializers.InitializeTenancy(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize tenancy use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("Tenancy domain initialized successfully: %v\n", tenancyUseCases != nil)

	return tenancyUseCases, nil
}

// initializeFundingUseCases initializes Funding domain use cases (3 entities: Fund, FundAllocation, FundTransaction).
// Graceful degradation: returns empty struct on failure so the app starts without funding.
func (uci *UseCaseInitializer) initializeFundingUseCases(container *Container) (*funding.FundingUseCases, error) {
	fmt.Printf("Initializing Funding use cases...\n")

	repos, err := domain.NewFundingRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("WARNING: Funding database provider not available: %v\n", err)
		return &funding.FundingUseCases{}, nil
	}
	fmt.Printf("Got funding repositories\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("ERROR: Failed to get services: %v\n", err)
		return nil, err
	}

	fundingUseCases, err := initializers.InitializeFunding(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize funding use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("Funding domain initialized successfully: %v\n", fundingUseCases != nil)

	return fundingUseCases, nil
}
