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
	"database/sql"
	"fmt"
	"os"

	"github.com/erniealice/espyna-golang/internal/composition/providers"

	// Application use cases aggregate
	"github.com/erniealice/espyna-golang/internal/application/usecases"

	// Application ports (for service interfaces)
	"github.com/erniealice/espyna-golang/internal/application/ports"
	securityports "github.com/erniealice/espyna-golang/internal/application/ports/security"

	// Infrastructure adapters for mock services
	mockAuth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/mock"
	// Production (non-mock) RBAC Authorizer — the Layer-4 use-case backstop.
	rbacauth "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/auth/rbac"
	internalregistry "github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	// Domain use cases (for proper initialization)
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/asset"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/common"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/communication"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/document"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/entity"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/event"
	eventdashboard "github.com/erniealice/espyna-golang/internal/application/usecases/domain/event/dashboard"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/expenditure"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/finance"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/fulfillment"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/funding"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/integration"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/inventory"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/ledger"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation"
	jobUseCase "github.com/erniealice/espyna-golang/internal/application/usecases/domain/operation/job"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/payroll"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/procurement"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/product"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/revenue"
	"github.com/erniealice/espyna-golang/internal/application/usecases/service"
	servicetax "github.com/erniealice/espyna-golang/internal/application/usecases/service/tax"

	// service/registrar blank-import: triggers init() of every
	// dynamically-registered service-driven candidate (currently
	// tax_compute; future dashboards/reporting). MUST be loaded before
	// initservice.InitializeAll so service.Register has populated
	// the factory map by the time service.NewServiceUseCases iterates it.
	// See docs/wiki/articles/hexagonal-rules.md §8 (tax_compute worked
	// example) for the canonical shape.
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription"
	subscriptionUseCase "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/subscription"
	_ "github.com/erniealice/espyna-golang/internal/application/usecases/service/registrar"

	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax"
	computepkg "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/compute_taxes_for_revenue"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/tenancy"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/workflow"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"

	repodomain "github.com/erniealice/espyna-golang/internal/composition/providers/domain"

	// Composition initializers (sub-packages mirroring proto/v1/{domain,service}/)
	"github.com/erniealice/espyna-golang/internal/composition/core/initializers/domain"
	initservice "github.com/erniealice/espyna-golang/internal/composition/core/initializers/service"
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

	documentUC, err := uci.initializeDocumentUseCases(container)
	if err != nil {
		documentUC = &document.UseCases{}
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

	communicationUC, err := uci.initializeCommunicationUseCases(container)
	if err != nil {
		// Only Communication domain fails - use empty struct for this domain only
		communicationUC = &communication.CommunicationUseCases{}
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
	if subscriptionUC != nil && subscriptionUC.Subscription != nil &&
		subscriptionUC.Subscription.MaterializeInstanceJobs != nil &&
		revenueUC != nil && revenueUC.Revenue != nil &&
		revenueUC.Revenue.RecognizeRevenueFromSubscription != nil {
		revenueUC.Revenue.RecognizeRevenueFromSubscription.SetMaterializeInstanceJobsForSubscription(
			&materializeInstanceJobsAdapter{uc: subscriptionUC.Subscription.MaterializeInstanceJobs},
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
	//
	// 2026-05-20 Plan 2 / Q-SDM-TAX — also capture the entity-layer compute
	// into the service/tax registry slot so that NewServiceUseCases (called
	// later in this method) builds a working service-driven Tax sub-aggregate.
	// RecognizeRevenueFromSubscription is then rewired BELOW (after serviceUC
	// is built) to route through that proto-shaped wrapper instead of the
	// entity-layer use case directly, satisfying the Q-SDM-TAX direction
	// (service-driven domain category) without changing failure semantics —
	// the wrapper's ExecuteForRevenue is a thin pass-through that calls the
	// captured entity compute with the same 3-arg shape.
	var entityCompute *computepkg.ComputeTaxesForRevenueUseCase
	if taxUC != nil && taxUC.ComputeTaxes != nil && revenueUC != nil && revenueUC.Revenue != nil {
		entityCompute = taxUC.ComputeTaxes.ComputeTaxesForRevenue
		if entityCompute != nil {
			// Wire into RecomputeTaxes admin use case (Phase E).
			if revenueUC.Revenue.RecomputeTaxes != nil {
				revenueUC.Revenue.RecomputeTaxes.SetComputeTaxes(entityCompute)
				fmt.Printf("Tax compute wired into revenue domain (ComputeTaxesForRevenue → RecomputeTaxes)\n")
			}
			// Wire into CreateRevenue post-persist hook (Phase D).
			if revenueUC.Revenue.CreateRevenue != nil {
				revenueUC.Revenue.CreateRevenue.SetComputeTaxes(entityCompute)
				fmt.Printf("Tax compute wired into revenue domain (ComputeTaxesForRevenue → CreateRevenue)\n")
			}
		}
	}

	// Capture entity-layer compute into the service/tax registry BEFORE
	// initializeServiceUseCases runs. The registered factory reads the
	// captured value at construction time; passing nil leaves the
	// service.Tax sub-aggregate unresolved and the RecognizeRevenueFromSubscription
	// invoker rewire below degrades to a no-op (no warning, no panic).
	servicetax.SetEntityCompute(entityCompute)

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

	// 20260518-hexagonal-strict-adherence Phase 1.D — service-driven
	// use cases (audit query; reporting; auth; security per Q7).
	// Resolves the raw *sql.DB from the database provider so the audit
	// service factory can plug in; degrades to a no-op service when the
	// connection isn't SQL-backed (mock/firestore). Option B: entity-auth
	// is built internally by the service initializer via txSvc + idSvc.
	serviceUC, err := uci.initializeServiceUseCases(container)
	if err != nil {
		serviceUC = &service.ServiceUseCases{}
	}

	// 2026-05-20 Plan 2 / Q-SDM-TAX — wire RecognizeRevenueFromSubscription's
	// ComputeTaxes hook through the service/tax wrapper instead of the
	// entity-layer use case directly. The proto-shaped wrapper satisfies the
	// narrow ComputeTaxesForRevenueInvoker contract via its ExecuteForRevenue
	// pass-through (defined in service/tax/compute_taxes_for_revenue.go) — no
	// failure-semantics change. serviceUC.Tax resolves via the dynamic
	// registry (servicetax.From == service.Get[*servicetax.UseCases]); when
	// the entity-layer compute is nil (no SQL provider), From returns nil and
	// SetComputeTaxes(nil) below leaves the hook disabled per its nil-safe
	// contract.
	if serviceUC != nil && revenueUC != nil && revenueUC.Revenue != nil &&
		revenueUC.Revenue.RecognizeRevenueFromSubscription != nil {
		if taxSub := servicetax.From(serviceUC); taxSub != nil && taxSub.ComputeTaxesForRevenue != nil {
			revenueUC.Revenue.RecognizeRevenueFromSubscription.SetComputeTaxes(taxSub.ComputeTaxesForRevenue)
			fmt.Printf("Tax compute wired into revenue domain via service/tax (ComputeTaxesForRevenue → RecognizeRevenueFromSubscription [service-driven path])\n")
		}
	}

	// Create aggregate with successfully initialized domains
	aggregate := usecases.NewAggregate(
		commonUC,
		documentUC,
		entityUC,
		eventUC,
		communicationUC,
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
		serviceUC,
	)
	container.useCases = aggregate

	return nil
}

// Domain initializer methods - one for each of the 7 domains

// initializeCommonUseCases initializes Common domain use cases (1 entity: Attribute)
// Common domain provides cross-domain dependencies used by Entity, Subscription, Product, and Treasury domains
func (uci *UseCaseInitializer) initializeCommonUseCases(container *Container) (*common.CommonUseCases, error) {
	fmt.Printf("🔧 Initializing Common use cases...\n")

	repos, err := repodomain.NewCommonRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
	commonUseCases, err := domain.InitializeCommon(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewEntityRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
	entityUseCases, err := domain.InitializeEntity(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewEventRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
	eventUseCases, err := domain.InitializeEvent(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize event use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Event domain initialized successfully: %v\n", eventUseCases != nil)

	return eventUseCases, nil
}

// initializeCommunicationUseCases initializes Communication domain use cases
// (conversation, conversation_post, conversation_read_receipt; participant is a
// v2-queried seam with no use cases).
func (uci *UseCaseInitializer) initializeCommunicationUseCases(container *Container) (*communication.CommunicationUseCases, error) {
	repos, err := repodomain.NewCommunicationRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		return nil, err
	}

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		return nil, err
	}

	return domain.InitializeCommunication(repos, authSvc, txSvc, i18nSvc, idSvc)
}

// initializeLedgerUseCases initializes Ledger domain use cases (document template)
func (uci *UseCaseInitializer) initializeLedgerUseCases(container *Container) (*ledger.LedgerUseCases, error) {
	fmt.Printf("📄 Initializing Ledger use cases...\n")

	repos, err := repodomain.NewLedgerRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	ledgerUseCases, err := domain.InitializeLedger(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize ledger use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Ledger domain initialized successfully: %v\n", ledgerUseCases != nil)

	return ledgerUseCases, nil
}

// initializeDocumentUseCases initializes Document domain use cases (attachment + template sub-aggregates).
// Per docs/plan/20260522-usecases-realignment Q-UR4 LOCK — bundled into one initializer
// because the document/ proto has both attachment and template sub-categories under it.
func (uci *UseCaseInitializer) initializeDocumentUseCases(container *Container) (*document.UseCases, error) {
	fmt.Printf("📄 Initializing Document use cases...\n")

	ledgerRepos, err := repodomain.NewLedgerRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Document repositories not available (Ledger unavailable): %v\n", err)
		return &document.UseCases{}, nil
	}

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return nil, err
	}

	documentUseCases, err := domain.InitializeDocument(ledgerRepos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("❌ Failed to initialize document use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("✅ Document domain initialized successfully: %v\n", documentUseCases != nil)
	return documentUseCases, nil
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

	repos, err := repodomain.NewOperationRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Operation database provider not available: %v\n", err)
		return &operation.OperationUseCases{}, nil
	}
	fmt.Printf("✅ Got operation repositories\n")

	subRepos, subErr := repodomain.NewSubscriptionRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	operationUseCases, err := domain.InitializeOperation(repos, subRepos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewTreasuryRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
	if revRepos, revErr := repodomain.NewRevenueRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig()); revErr == nil && revRepos != nil {
		repos.Revenue = revRepos.Revenue
	} else if revErr != nil {
		fmt.Printf("⚠️  Treasury: revenue repo unavailable for AmortizeAdvanceCollection: %v\n", revErr)
	}
	if expRepos, expErr := repodomain.NewExpenditureRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig()); expErr == nil && expRepos != nil {
		repos.ExpenseRecognition = expRepos.ExpenseRecognition
	} else if expErr != nil {
		fmt.Printf("⚠️  Treasury: expense_recognition repo unavailable for AmortizeAdvanceDisbursement: %v\n", expErr)
	}

	// 20260517-advance-cash-events Plan B Phase 7 — pull BillingEvent from the
	// subscription provider block so the selling-side MILESTONE recognize use
	// case (RecognizeMilestoneAdvanceCollection) is actually constructed.
	// Without this the four-AND nil-guard in treasury.NewUseCases sees
	// repos.BillingEvent == nil and silently leaves the use case as nil.
	if subRepos, subErr := repodomain.NewSubscriptionRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig()); subErr == nil && subRepos != nil {
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
	treasuryUseCases, err := domain.InitializeTreasury(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewProductRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
	productUseCases, err := domain.InitializeProduct(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewRevenueRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	revenueUseCases, err := domain.InitializeRevenue(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewExpenditureRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Expenditure database provider not available: %v\n", err)
		return &expenditure.ExpenditureUseCases{}, nil
	}
	fmt.Printf("✅ Got expenditure repositories\n")

	// Cross-domain: procurement repos for SupplierSubscription workspace validation
	// on RecognizeFromExpenditure (buying/selling parity 2026-05-09). Non-fatal.
	procurementRepos, procErr := repodomain.NewProcurementRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
	treasuryRepos, trErr := repodomain.NewTreasuryRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	expenditureUseCases, err := domain.InitializeExpenditure(repos, authSvc, txSvc, i18nSvc, idSvc, treasuryUseCases)
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

	repos, err := repodomain.NewInventoryRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
	inventoryUseCases, err := domain.InitializeInventory(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	subscriptionRepos, err := repodomain.NewSubscriptionRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
	operationRepos, opErr := repodomain.NewOperationRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idSvc,
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
				Authorizer:                     authSvc,
				Transactor:                     txSvc,
				Translator:                     i18nSvc,
				IDGenerator:                    idSvc,
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
				Authorizer:  authSvc,
				Transactor:  txSvc,
				Translator:  i18nSvc,
				IDGenerator: idSvc,
			},
		)
		fmt.Printf("✅ MaterializeInstanceJobsForSubscription wired\n")
	}

	// Use composition initializer to wire everything together. The reference
	// checker is plumbed through ports.NewNoOpReferenceChecker by default —
	// the application owner (service-admin) wires the postgres-backed
	// reference.Checker via the container path when running on postgres.
	subscriptionUseCases, err := domain.InitializeSubscription(subscriptionRepos, authSvc, txSvc, i18nSvc, idSvc, jobTemplateInstantiator, ports.NewNoOpReferenceChecker())
	if err != nil {
		fmt.Printf("❌ Failed to initialize subscription use cases: %v\n", err)
		return nil, err
	}
	// 2026-04-29 auto-spawn-jobs-from-subscription Phase D — expose the
	// concrete use case so centymo's create-form opt-out + retroactive
	// spawn handler can call it directly.
	// 20260518-hexagonal-strict-adherence Phase 3 F6 closure — the two
	// materialize use cases now nest under .Subscription.MaterializeJobs /
	// .Subscription.MaterializeInstanceJobs (entity sub-aggregate).
	if subscriptionUseCases != nil && opErr == nil && subscriptionUseCases.Subscription != nil {
		// `mjfs` is in scope only when operation repos resolved. Capture
		// from the instantiator wrapper to keep the nil branches clean.
		if inst, ok := jobTemplateInstantiator.(*subscriptionUseCase.MaterializeJobsForSubscriptionInstantiator); ok && inst != nil {
			subscriptionUseCases.Subscription.MaterializeJobs = inst.UseCase
		}
		// 2026-04-30 cyclic-subscription-jobs Phase B — expose the cyclic
		// instance Job spawner alongside MaterializeJobs so:
		//   - recognize-revenue piggyback (espyna's own Phase C wiring) can
		//     call it after successful revenue recognition.
		//   - Future Operations tab "Spawn this cycle now" / "Backfill" CTAs
		//     can invoke it directly via the consumer surface.
		if instanceJobsUC != nil {
			subscriptionUseCases.Subscription.MaterializeInstanceJobs = instanceJobsUC
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

	repos, err := repodomain.NewWorkflowRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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
	workflowUseCases, err := domain.InitializeWorkflow(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewPayrollRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if err != nil {
		fmt.Printf("⚠️  Payroll database provider not available: %v\n", err)
		return &payroll.PayrollUseCases{}, nil
	}
	fmt.Printf("✅ Got payroll repositories\n")

	// Cross-domain repos for the orchestrator. Failure to load these is non-fatal:
	// payroll CRUD still works; only Calculate/GeneratePayCycles will be unavailable.
	entityRepos, eErr := repodomain.NewEntityRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	if eErr != nil {
		fmt.Printf("⚠️  Payroll: entity repos unavailable for orchestrator: %v\n", eErr)
		entityRepos = nil
	}
	expenditureRepos, xErr := repodomain.NewExpenditureRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	payrollUseCases, err := domain.InitializePayroll(repos, entityRepos, expenditureRepos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewFulfillmentRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	fulfillmentUseCases, err := domain.InitializeFulfillment(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewAssetRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	assetUseCases, err := domain.InitializeAsset(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewTaxRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	taxUseCases, err := domain.InitializeTax(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewFinanceRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	financeUseCases, err := domain.InitializeFinance(repos, authSvc, txSvc, i18nSvc, idSvc)
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

	repos, err := repodomain.NewProcurementRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	procurementUseCases, err := domain.InitializeProcurement(repos, authSvc, txSvc, i18nSvc, idSvc)
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
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	err error,
) {
	// Services are optional - translation service handles nil gracefully by returning fallback strings
	// Auth and transaction services should be checked by use cases before use

	// Authorization selection (W0 — Q-AWS2 = A+C; see
	// docs/plan/20260530-authz-workspace-hardening/w0-design.md §2.5).
	//
	// Resolution order:
	//   1. Provider already an Authorizer (e.g. an injected real provider).
	//   2. The real RBAC Authorizer built from the registered PermissionQuery
	//      — the SAME chain the UI permission loader uses (no second SQL path).
	//   3. AllowAll ONLY when provably a dev/mock build
	//      (allowAllFallbackPermitted); otherwise a non-nil error is returned
	//      so getServices propagates a hard boot-fail (Q-AWS2 = C). This makes
	//      the "silently booted with allow-all" state impossible for a
	//      password / non-dev build.

	// 1. Provider already an Authorizer?
	if authProvider := uci.providerManager.GetAuthProvider(); authProvider != nil {
		if authService, ok := authProvider.(ports.Authorizer); ok {
			authSvc = authService
			fmt.Printf("🔐 Using authorization service from provider: %T\n", authSvc)
		}
	}

	// 2. Build the real RBAC Authorizer from the registered PermissionQuery.
	if authSvc == nil {
		if pq := uci.resolvePermissionQuery(); pq != nil {
			authSvc = rbacauth.NewPermissionAuthorizer(pq)
			fmt.Printf("🔐 Using RBAC Authorizer (PermissionQuery-backed): %T\n", authSvc)
		}
	}

	// 3. Fallback to AllowAll ONLY when provably a dev/mock build; else boot-fail.
	if authSvc == nil {
		if allowAllFallbackPermitted() {
			authSvc = mockAuth.NewAllowAllAuth()
			fmt.Printf("🔓 AllowAll authorization (dev/mock build — CONFIG_AUTH_PROVIDER=mock_auth)\n")
		} else {
			return nil, nil, nil, nil, fmt.Errorf(
				"refusing to boot: no RBAC Authorizer available (PermissionQuery unregistered or no *sql.DB) " +
					"and AllowAll fallback is forbidden for non-dev builds (set CONFIG_AUTH_PROVIDER=mock_auth for dev/mock)")
		}
	}

	// Get ID service from provider manager
	if idProvider := uci.providerManager.GetIDProvider(); idProvider != nil {
		// Check if the provider has a GetIDService method (IDProviderWrapper)
		if idWrapper, ok := idProvider.(interface{ GetIDService() ports.IDGenerator }); ok {
			idSvc = idWrapper.GetIDService()
			// fmt.Printf("🆔 Using ID service from provider: %T - %s\n", idSvc, idSvc.GetProviderInfo())
		} else if idService, ok := idProvider.(ports.IDGenerator); ok {
			// Fallback: provider directly implements IDGenerator
			idSvc = idService
			fmt.Printf("🆔 Using ID service (direct): %T - %s\n", idSvc, idSvc.GetProviderInfo())
		} else {
			// Fallback to noop if provider doesn't implement the interface
			idSvc = ports.NewNoOpIDGenerator()
			fmt.Printf("🆔 Created NoOp ID service (provider fallback): %T\n", idSvc)
		}
	} else {
		// No ID provider configured, use noop service
		idSvc = ports.NewNoOpIDGenerator()
		fmt.Printf("🆔 Created NoOp ID service (no provider): %T\n", idSvc)
	}

	txSvc, _ = container.services.Transaction.(ports.Transactor)

	// Extract Translator from wrapper or direct assignment
	if wrapper, ok := container.services.Translation.(*translationServiceWrapper); ok {
		i18nSvc = wrapper.svc
	} else {
		i18nSvc, _ = container.services.Translation.(ports.Translator)
	}

	return authSvc, txSvc, i18nSvc, idSvc, nil
}

// resolvePermissionQuery returns the registered PermissionQuery backed by the
// database provider's raw *sql.DB, or nil when no SQL connection or no RBAC
// factory is available (e.g. non-postgres / non-mock builds).
//
// It mirrors initializers/service/security.go:permissionQueryFromDB and reuses
// the same GetConnection()→*sql.DB extraction shape already present in
// initializeServiceUseCases. This is the W0 v1 choice (OD-3): a standalone
// helper that accepts the duplicate DB-handle extraction, rather than threading
// the security-initializer's single PermissionQuery instance into getServices
// (which runs per-domain, before/independent of the service use cases — the
// reordering is the tradeoff flagged as the dedup follow-up).
func (uci *UseCaseInitializer) resolvePermissionQuery() securityports.PermissionQuery {
	var sqlDB *sql.DB
	if dbProvider := uci.providerManager.GetDatabaseProvider(); dbProvider != nil {
		if connHolder, ok := dbProvider.(interface{ GetConnection() any }); ok {
			if conn := connHolder.GetConnection(); conn != nil {
				if db, ok := conn.(*sql.DB); ok {
					sqlDB = db
				}
			}
		}
	}
	if sqlDB == nil {
		return nil
	}
	factory, ok := internalregistry.GetPermissionQueryFactory()
	if !ok || factory == nil {
		return nil
	}
	// The factory takes `any` to dodge the cyclic import (see
	// registry/permission_query.go); the postgres adapter expects a *sql.DB and
	// returns a *PostgresPermissionQuery, the mock adapter ignores db.
	if pq, ok := factory(sqlDB).(securityports.PermissionQuery); ok {
		return pq
	}
	return nil
}

// allowAllFallbackPermitted reports whether selecting the AllowAll authorization
// service is permitted (Q-AWS2 = C boot-fail guard, w0-design.md §2.7). It
// returns true ONLY when the build is provably dev/mock — keyed on the same
// CONFIG_AUTH_PROVIDER == "mock_auth" signal the app already discriminates on at
// apps/service-admin/internal/composition/container.go:236,1496, so the dev/prod
// boundary stays consistent.
//
// When this returns false and no RBAC Authorizer could be built, getServices
// returns a non-nil error which propagates to the app entrypoint's log.Fatalf —
// a hard boot fail. This is the W0 RUNTIME guard; the compile-time build-tag
// flip of AllowAllAuthService to //go:build mock_auth is deferred to audit
// Wave-0.1 (OD-5 / R4) after the NewAllowAllAuth-caller audit.
func allowAllFallbackPermitted() bool {
	return os.Getenv("CONFIG_AUTH_PROVIDER") == "mock_auth"
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
	var integrationPaymentRepo repodomain.IntegrationPaymentRepository
	dbProvider := uci.providerManager.GetDatabaseProvider()
	tableConfig := uci.providerManager.GetDBTableConfig()
	if dbProvider != nil && tableConfig != nil {
		repo, err := repodomain.NewIntegrationPaymentRepository(dbProvider, tableConfig)
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

	repos, err := repodomain.NewTenancyRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	tenancyUseCases, err := domain.InitializeTenancy(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize tenancy use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("Tenancy domain initialized successfully: %v\n", tenancyUseCases != nil)

	return tenancyUseCases, nil
}

// initializeServiceUseCases initializes the service-driven domain
// sub-aggregate (audit query; reporting; security per Q7). The
// audit sub-aggregate needs a raw *sql.DB so the audit service factory
// can plug in; on non-SQL providers (mock/firestore) the AuditService
// resolves to nil and the use cases degrade gracefully.
// txSvc and idSvc are threaded through to InitializeAll (Option B:
// entity-auth is built internally by the service initializer).
func (uci *UseCaseInitializer) initializeServiceUseCases(container *Container) (*service.ServiceUseCases, error) {
	fmt.Printf("🧩 Initializing Service-driven use cases (audit, security, auth)...\n")

	authSvc, txSvc, i18nSvc, idSvc, err := uci.getServices(container)
	if err != nil {
		fmt.Printf("❌ Failed to get services: %v\n", err)
		return &service.ServiceUseCases{}, err
	}

	// Resolve the raw *sql.DB from the database provider (only postgres
	// builds yield a concrete *sql.DB). Non-SQL providers degrade to nil
	// — the audit use cases still wire up but ListByEntity returns empty.
	var sqlDB *sql.DB
	if dbProvider := uci.providerManager.GetDatabaseProvider(); dbProvider != nil {
		if connHolder, ok := dbProvider.(interface{ GetConnection() any }); ok {
			if conn := connHolder.GetConnection(); conn != nil {
				if db, ok := conn.(*sql.DB); ok {
					sqlDB = db
				}
			}
		}
	}

	// Wave B P1.C.1+: dashboards under service.Dashboard read across
	// entity repos via extension interfaces; resolve them once here so
	// InitializeService can thread per-candidate Deps fields through to
	// the umbrella factory.
	//
	// Round 2a (Location P1.C.2, Equity P1.C.4, Payroll P1.C.6) aborted
	// 2026-05-20 — the ledger/payroll repo resolution for the equity +
	// payroll dashboards is omitted here until the proto regen + use
	// cases re-land. See docs/plan/20260520-service-domain-migration/
	// progress.md.
	entityRepos, _ := repodomain.NewEntityRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())

	// Wave B P1.C.4 Equity — dashboards under service.Dashboard.Equity read
	// across the ledger equity_account + equity_transaction aggregates.
	// Resolved here so InitializeService can thread typed Deps fields into
	// the umbrella factory. Nil under non-postgres builds — equity use case
	// tolerates nil repositories.
	ledgerReposForSvc, _ := repodomain.NewLedgerRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())

	// Wave B P1.C.6 Payroll — dashboards under service.Dashboard.Payroll read
	// across the payroll_run + payroll_remittance aggregates. Resolved here
	// so InitializeService can thread typed Deps fields into the umbrella
	// factory. Nil under non-postgres builds — payroll dashboard use case
	// tolerates nil repositories.
	payrollReposForSvc, _ := repodomain.NewPayrollRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())

	// Wave B P1.C.5 Treasury (unified Loan+Cash) — dashboards under
	// service.Dashboard.Treasury read across loan + loan_payment (Loan slice)
	// and collection (Cash slice). Resolved here so InitializeService can
	// thread typed Deps fields into the umbrella factory. Nil under non-
	// postgres builds — treasury dashboard use cases tolerate nil repos.
	treasuryReposForSvc, _ := repodomain.NewTreasuryRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())

	// Wave C P1.C.8 Expenditure, P1.C.9 Job (source aggregate `operation`),
	// P1.C.11 Product, P1.C.12 Fulfillment — dashboards under
	// service.Dashboard.{Expenditure,Job,Product,Fulfillment} read from the
	// expenditure / operation (job+job_activity) / product / fulfillment
	// aggregates. Resolved here so InitializeService can thread typed Deps
	// fields into the umbrella factory. Nil under non-postgres builds — each
	// dashboard use case tolerates nil repositories.
	expenditureReposForSvc, _ := repodomain.NewExpenditureRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	operationReposForSvc, _ := repodomain.NewOperationRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	productReposForSvc, _ := repodomain.NewProductRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	fulfillmentReposForSvc, _ := repodomain.NewFulfillmentRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())

	// Wave B P1.C.7 Schedule (event dashboard): build the entity-layer
	// schedule-dashboard use case here so the service-layer wrapper can
	// wrap it without re-coupling event/usecases.go to the dashboard
	// proto. The event repo only satisfies EventDashboardRepository under
	// the postgres adapter; type assertion fails harmlessly on mock builds
	// and the wrapper degrades to empty Response.
	eventRepos, _ := repodomain.NewEventRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
	var scheduleEntityDash *eventdashboard.GetScheduleDashboardPageDataUseCase
	if eventRepos != nil && eventRepos.Event != nil {
		if eq, ok := eventRepos.Event.(eventdashboard.EventDashboardRepository); ok {
			scheduleEntityDash = eventdashboard.NewGetScheduleDashboardPageDataUseCase(eq)
		}
	}

	// Wave B P1.E.1 AR aging — the espyna composition root cannot build the
	// concrete ledger reporting svc here (the table config lives in the
	// app's composition root, inlined at apps/service-admin/internal/composition/
	// container.go ~line 187 (struct) and ~line 759 (factory call)). Pass nil; the AR aging Reporter on the
	// umbrella stays nil and the use cases degrade to empty responses
	// until the app's composition root rewires through a different code
	// path. The actual wiring happens in apps/service-admin via a setter
	// pattern (see ar_aging.SetReporter in service/reporting/ar_aging/).
	var ledgerReportingSvcForARAging any = nil

	svcUC, err := initservice.InitializeAll(sqlDB, authSvc, i18nSvc, txSvc, idSvc, entityRepos, ledgerReposForSvc, payrollReposForSvc, treasuryReposForSvc, expenditureReposForSvc, operationReposForSvc, productReposForSvc, fulfillmentReposForSvc, scheduleEntityDash, ledgerReportingSvcForARAging)
	if err != nil {
		fmt.Printf("❌ Failed to initialize service-driven use cases: %v\n", err)
		return &service.ServiceUseCases{}, err
	}
	fmt.Printf("✅ Service-driven use cases initialized (audit: %v, security: %v, auth: %v)\n",
		svcUC != nil && svcUC.Audit != nil,
		svcUC != nil && svcUC.Security != nil,
		svcUC != nil && svcUC.Auth != nil)
	return svcUC, nil
}

// initializeFundingUseCases initializes Funding domain use cases (3 entities: Fund, FundAllocation, FundTransaction).
// Graceful degradation: returns empty struct on failure so the app starts without funding.
func (uci *UseCaseInitializer) initializeFundingUseCases(container *Container) (*funding.FundingUseCases, error) {
	fmt.Printf("Initializing Funding use cases...\n")

	repos, err := repodomain.NewFundingRepositories(uci.providerManager.GetDatabaseProvider(), uci.providerManager.GetDBTableConfig())
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

	fundingUseCases, err := domain.InitializeFunding(repos, authSvc, txSvc, i18nSvc, idSvc)
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize funding use cases: %v\n", err)
		return nil, err
	}
	fmt.Printf("Funding domain initialized successfully: %v\n", fundingUseCases != nil)

	return fundingUseCases, nil
}
