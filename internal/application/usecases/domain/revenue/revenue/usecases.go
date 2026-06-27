package revenue

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	computepkg "github.com/erniealice/espyna-golang/internal/application/usecases/domain/tax/compute_taxes_for_revenue"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	jobpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job"
	jobtemplatephasepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_phase"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
	revenuerunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_run"
	billingeventpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/billing_event"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

// RevenueRepositories groups all repository dependencies for revenue use cases.
// Cross-domain reads (Subscription, PricePlan, ProductPricePlan, PriceSchedule,
// Client, RevenueLineItem, PaymentTerm) are required by the
// RecognizeRevenueFromSubscription use case — see plan §5 Phase B.
type RevenueRepositories struct {
	Revenue          revenuepb.RevenueDomainServiceServer
	RevenueLineItem  revenuelineitempb.RevenueLineItemDomainServiceServer
	Subscription     subscriptionpb.SubscriptionDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
	PriceSchedule    priceschedulepb.PriceScheduleDomainServiceServer
	Client           clientpb.ClientDomainServiceServer
	PaymentTerm      paymenttermpb.PaymentTermDomainServiceServer
	// Workspace repo — used by ListRevenueRunCandidates to resolve the
	// workspace timezone for billing-cycle math. Optional; falls back to UTC.
	Workspace workspacepb.WorkspaceDomainServiceServer

	// RevenueRun repo — used by ListRevenueRunCandidates and GenerateRevenueRun.
	RevenueRun revenuerunpb.RevenueRunDomainServiceServer

	// Milestone-billing branch (Phase C — milestone-billing plan §3).
	// Optional — only required when MILESTONE PricePlans are billed.
	BillingEvent     billingeventpb.BillingEventDomainServiceServer
	JobTemplatePhase jobtemplatephasepb.JobTemplatePhaseDomainServiceServer
	Job              jobpb.JobDomainServiceServer

	// TreasuryCollection — used by ListRevenueRunCandidates (advance-Collection
	// branch) and indirectly by GenerateRevenueRun (via AmortizeAdvanceCollection).
	// Optional; when nil, advance-Collection candidates and dispatch are skipped.
	// Plan B Phase 5a.
	TreasuryCollection collectionpb.CollectionDomainServiceServer
}

// RevenueServices groups all business service dependencies for revenue use cases
type RevenueServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator

	// 2026-04-30 cyclic-subscription-jobs plan §5.2 — recognize-piggyback
	// invoker. Optional; when nil the piggyback hook is skipped (no warning).
	// See RecognizeRevenueFromSubscriptionServices.MaterializeInstanceJobsForSubscription
	// for the full failure-semantics contract.
	MaterializeInstanceJobsForSubscription MaterializeInstanceJobsForSubscriptionInvoker

	// ComputeTaxes wires the ComputeTaxesForRevenue use case into the revenue
	// domain. Optional; when nil, tax-compute integration points are skipped.
	ComputeTaxes *computepkg.ComputeTaxesForRevenueUseCase

	// AmortizeAdvanceCollection wires Plan B's selling-side amortization use
	// case for the GenerateRevenueRun dispatcher. Optional; when nil,
	// ADVANCE_COLLECTION selections error out with "amortize_advance_unavailable".
	// Plan B Phase 5c.
	AmortizeAdvanceCollection AdvanceCollectionAmortizer
}

// UseCases contains all revenue-related use cases
type UseCases struct {
	CreateRevenue                    *CreateRevenueUseCase
	ReadRevenue                      *ReadRevenueUseCase
	UpdateRevenue                    *UpdateRevenueUseCase
	DeleteRevenue                    *DeleteRevenueUseCase
	ListRevenues                     *ListRevenuesUseCase
	GetRevenueListPageData           *GetRevenueListPageDataUseCase
	RecognizeRevenueFromSubscription *RecognizeRevenueFromSubscriptionUseCase
	ListRevenueRunCandidates         *ListRevenueRunCandidatesUseCase
	GenerateRevenueRun               *GenerateRevenueRunUseCase
	RecomputeTaxes                   *RecomputeTaxesUseCase
}

// NewUseCases creates a new collection of revenue use cases
func NewUseCases(
	repositories RevenueRepositories,
	services RevenueServices,
) *UseCases {
	createRepos := CreateRevenueRepositories{
		Revenue:     repositories.Revenue,
		PaymentTerm: repositories.PaymentTerm,
	}
	createServices := CreateRevenueServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadRevenueRepositories{
		Revenue: repositories.Revenue,
	}
	readServices := ReadRevenueServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateRevenueRepositories{
		Revenue: repositories.Revenue,
	}
	updateServices := UpdateRevenueServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteRevenueRepositories{
		Revenue: repositories.Revenue,
	}
	deleteServices := DeleteRevenueServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListRevenuesRepositories{
		Revenue: repositories.Revenue,
	}
	listServices := ListRevenuesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetRevenueListPageDataRepositories{
		Revenue: repositories.Revenue,
	}
	getListPageDataServices := GetRevenueListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	recognizeRepos := RecognizeRevenueFromSubscriptionRepositories{
		Revenue:          repositories.Revenue,
		RevenueLineItem:  repositories.RevenueLineItem,
		Subscription:     repositories.Subscription,
		PricePlan:        repositories.PricePlan,
		ProductPricePlan: repositories.ProductPricePlan,
		PriceSchedule:    repositories.PriceSchedule,
		Client:           repositories.Client,
		PaymentTerm:      repositories.PaymentTerm,

		BillingEvent:     repositories.BillingEvent,
		JobTemplatePhase: repositories.JobTemplatePhase,
	}
	recognizeServices := RecognizeRevenueFromSubscriptionServices{
		ActionGatekeeper:                        services.ActionGatekeeper,
		Authorizer:                             services.Authorizer,
		Transactor:                             services.Transactor,
		Translator:                             services.Translator,
		IDGenerator:                            services.IDGenerator,
		MaterializeInstanceJobsForSubscription: services.MaterializeInstanceJobsForSubscription,
		ComputeTaxes:                           services.ComputeTaxes,
	}

	recognizeUC := NewRecognizeRevenueFromSubscriptionUseCase(recognizeRepos, recognizeServices)

	listCandidatesRepos := ListRevenueRunCandidatesRepositories{
		Revenue:      repositories.Revenue,
		Subscription: repositories.Subscription,
		PricePlan:    repositories.PricePlan,
		Workspace:    repositories.Workspace,
	}
	// Plan B Phase 5a — thread TreasuryCollection through for the advance branch.
	listCandidatesRepos.TreasuryCollection = repositories.TreasuryCollection
	listCandidatesServices := ListRevenueRunCandidatesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	generateRunRepos := GenerateRevenueRunRepositories{
		Revenue:      repositories.Revenue,
		Subscription: repositories.Subscription,
		RevenueRun:   repositories.RevenueRun,
		Workspace:    repositories.Workspace,
	}
	generateRunServices := GenerateRevenueRunServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	recomputeTaxesUC := NewRecomputeTaxesUseCase(
		RecomputeTaxesRepositories{Revenue: repositories.Revenue},
		RecomputeTaxesServices{
			ActionGatekeeper: services.ActionGatekeeper,
			Authorizer: services.Authorizer,
			Translator: services.Translator,
		},
		services.ComputeTaxes,
	)

	return &UseCases{
		CreateRevenue:                    NewCreateRevenueUseCase(createRepos, createServices),
		ReadRevenue:                      NewReadRevenueUseCase(readRepos, readServices),
		UpdateRevenue:                    NewUpdateRevenueUseCase(updateRepos, updateServices),
		DeleteRevenue:                    NewDeleteRevenueUseCase(deleteRepos, deleteServices),
		ListRevenues:                     NewListRevenuesUseCase(listRepos, listServices),
		GetRevenueListPageData:           NewGetRevenueListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		RecognizeRevenueFromSubscription: recognizeUC,
		ListRevenueRunCandidates:         NewListRevenueRunCandidatesUseCase(listCandidatesRepos, listCandidatesServices, recognizeUC),
		GenerateRevenueRun:               NewGenerateRevenueRunUseCase(generateRunRepos, generateRunServices, recognizeUC).WithAdvanceCollectionAmortizer(services.AmortizeAdvanceCollection),
		RecomputeTaxes:                   recomputeTaxesUC,
	}
}
