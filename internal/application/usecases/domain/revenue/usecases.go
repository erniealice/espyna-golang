package revenue

import (
	// Revenue use cases
	deferredRevenueUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/revenue/deferred_revenue"
	revenueUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/revenue/revenue"
	revenueAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/revenue/revenue_attribute"
	revenueCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/revenue/revenue_category"
	revenueLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/revenue/revenue_line_item"
	revenueTaxLineUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/revenue/revenue_tax_line"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"
	computepkg "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/compute_taxes_for_revenue"
	treasurycollectionpkg "github.com/erniealice/espyna-golang/internal/application/usecases/domain/treasury/collection"

	// Protobuf domain services - Entity domain (cross-domain dependency)
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"

	// Protobuf domain services for revenue repositories
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
	revenuecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	revenuerunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_run"
	revenuetaxlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_tax_line"

	// Cross-domain dependencies for the recognize-revenue use case
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"

	// Milestone-billing branch — operation domain reads.
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
)

// RevenueRepositories contains all revenue domain repositories.
//
// The recognize-revenue use case requires cross-domain reads from
// Subscription, PricePlan, ProductPricePlan, PriceSchedule, and Client. These
// are passed through unchanged when wired but may be nil — the use case
// gracefully degrades (returns appropriate validation errors) when a
// repository is missing.
type RevenueRepositories struct {
	Revenue          revenuepb.RevenueDomainServiceServer
	RevenueLineItem  revenuelineitempb.RevenueLineItemDomainServiceServer
	RevenueCategory  revenuecategorypb.RevenueCategoryDomainServiceServer
	RevenueAttribute revenueattributepb.RevenueAttributeDomainServiceServer
	DeferredRevenue  deferredrevenuepb.DeferredRevenueDomainServiceServer
	RevenueTaxLine   revenuetaxlinepb.RevenueTaxLineDomainServiceServer
	// Cross-domain dependency: payment term lookup for due date computation
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer

	// RevenueRun repo — used by ListRevenueRunCandidates and GenerateRevenueRun.
	RevenueRun revenuerunpb.RevenueRunDomainServiceServer

	// Cross-domain dependencies for revenue recognition from a subscription.
	// All optional from a wiring standpoint — but all required for the
	// RecognizeRevenueFromSubscription use case to actually fire.
	Subscription     subscriptionpb.SubscriptionDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
	PriceSchedule    priceschedulepb.PriceScheduleDomainServiceServer
	Client           clientpb.ClientDomainServiceServer

	// TreasuryCollection — used by ListRevenueRunCandidates (advance branch)
	// and indirectly by GenerateRevenueRun (via AmortizeAdvanceCollection).
	// Optional; when nil, advance-Collection candidates are skipped.
	// Plan B Phase 5a.
	TreasuryCollection collectionpb.CollectionDomainServiceServer

	// Workspace repo — used by ListRevenueRunCandidates to resolve the
	// workspace timezone for billing-cycle math. Optional; falls back to UTC.
	Workspace workspacepb.WorkspaceDomainServiceServer

	// Milestone-billing branch (Phase C — milestone-billing plan §3).
	// Optional — only required when MILESTONE plans are billed.
	BillingEvent     billingeventpb.BillingEventDomainServiceServer
	JobTemplatePhase jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	Job              jobpb.JobDomainServiceServer
}

// RevenueUseCases contains all revenue-related use cases
type RevenueUseCases struct {
	Revenue          *revenueUseCases.UseCases
	RevenueLineItem  *revenueLineItemUseCases.UseCases
	RevenueCategory  *revenueCategoryUseCases.UseCases
	RevenueAttribute *revenueAttributeUseCases.UseCases
	DeferredRevenue  *deferredRevenueUseCases.UseCases
	RevenueTaxLine   *revenueTaxLineUseCases.UseCases
}

// NewUseCases creates all revenue use cases with proper constructor injection.
// computeTaxes is optional; pass nil to disable tax-compute integration points.
//
// To wire the advance-Collection amortizer (Plan B Phase 5c), call
// WithAmortizeAdvanceCollection on the returned aggregator after construction.
func NewUseCases(
	repos RevenueRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
	computeTaxes ...*computepkg.ComputeTaxesForRevenueUseCase,
) *RevenueUseCases {
	// Accept optional computeTaxes as variadic for backward-compatibility.
	var computeUC *computepkg.ComputeTaxesForRevenueUseCase
	if len(computeTaxes) > 0 {
		computeUC = computeTaxes[0]
	}

	revenueUC := revenueUseCases.NewUseCases(
		revenueUseCases.RevenueRepositories{
			Revenue:          repos.Revenue,
			RevenueLineItem:  repos.RevenueLineItem,
			Subscription:     repos.Subscription,
			PricePlan:        repos.PricePlan,
			ProductPricePlan: repos.ProductPricePlan,
			PriceSchedule:    repos.PriceSchedule,
			Client:           repos.Client,
			PaymentTerm:      repos.PaymentTerm,
			Workspace:        repos.Workspace,
			RevenueRun:       repos.RevenueRun,

			// Milestone-billing branch reads (Phase C).
			BillingEvent:     repos.BillingEvent,
			JobTemplatePhase: repos.JobTemplatePhase,
			Job:              repos.Job,

			// Plan B Phase 5a — advance-Collection branch.
			TreasuryCollection: repos.TreasuryCollection,
		},
		revenueUseCases.RevenueServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
			ComputeTaxes:         computeUC,
		},
	)

	revenueLineItemUC := revenueLineItemUseCases.NewUseCases(
		revenueLineItemUseCases.RevenueLineItemRepositories{
			RevenueLineItem: repos.RevenueLineItem,
		},
		revenueLineItemUseCases.RevenueLineItemServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	revenueCategoryUC := revenueCategoryUseCases.NewUseCases(
		revenueCategoryUseCases.RevenueCategoryRepositories{
			RevenueCategory: repos.RevenueCategory,
		},
		revenueCategoryUseCases.RevenueCategoryServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	revenueAttributeUC := revenueAttributeUseCases.NewUseCases(
		revenueAttributeUseCases.RevenueAttributeRepositories{
			RevenueAttribute: repos.RevenueAttribute,
		},
		revenueAttributeUseCases.RevenueAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	deferredRevenueUC := deferredRevenueUseCases.NewUseCases(
		deferredRevenueUseCases.DeferredRevenueRepositories{
			DeferredRevenue: repos.DeferredRevenue,
		},
		deferredRevenueUseCases.DeferredRevenueServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	revenueTaxLineUC := revenueTaxLineUseCases.NewUseCases(
		revenueTaxLineUseCases.RevenueTaxLineRepositories{
			RevenueTaxLine: repos.RevenueTaxLine,
		},
		revenueTaxLineUseCases.RevenueTaxLineServices{
			AuthorizationService: authSvc,
			TranslationService:   i18nSvc,
		},
	)

	return &RevenueUseCases{
		Revenue:          revenueUC,
		RevenueLineItem:  revenueLineItemUC,
		RevenueCategory:  revenueCategoryUC,
		RevenueAttribute: revenueAttributeUC,
		DeferredRevenue:  deferredRevenueUC,
		RevenueTaxLine:   revenueTaxLineUC,
	}
}

// WithAmortizeAdvanceCollection wires Plan B's AmortizeAdvanceCollection use
// case into GenerateRevenueRun so ADVANCE_COLLECTION selections dispatch
// correctly. Call after NewUseCases — the use case isn't constructed by this
// aggregator (lives in a sibling package).
//
// Returns the receiver for builder-style chaining. Safe to call with nil.
func (uc *RevenueUseCases) WithAmortizeAdvanceCollection(a *treasurycollectionpkg.AmortizeAdvanceCollectionUseCase) *RevenueUseCases {
	if uc == nil || uc.Revenue == nil || uc.Revenue.GenerateRevenueRun == nil {
		return uc
	}
	uc.Revenue.GenerateRevenueRun.WithAdvanceCollectionAmortizer(a)
	return uc
}
