package revenue

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"

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
}

// RevenueServices groups all business service dependencies for revenue use cases
type RevenueServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService

	// 2026-04-30 cyclic-subscription-jobs plan §5.2 — recognize-piggyback
	// invoker. Optional; when nil the piggyback hook is skipped (no warning).
	// See RecognizeRevenueFromSubscriptionServices.MaterializeInstanceJobsForSubscription
	// for the full failure-semantics contract.
	MaterializeInstanceJobsForSubscription MaterializeInstanceJobsForSubscriptionInvoker
}

// UseCases contains all revenue-related use cases
type UseCases struct {
	CreateRevenue                     *CreateRevenueUseCase
	ReadRevenue                       *ReadRevenueUseCase
	UpdateRevenue                     *UpdateRevenueUseCase
	DeleteRevenue                     *DeleteRevenueUseCase
	ListRevenues                      *ListRevenuesUseCase
	GetRevenueListPageData            *GetRevenueListPageDataUseCase
	RecognizeRevenueFromSubscription  *RecognizeRevenueFromSubscriptionUseCase
	ListRevenueRunCandidates          *ListRevenueRunCandidatesUseCase
	GenerateRevenueRun                *GenerateRevenueRunUseCase
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadRevenueRepositories{
		Revenue: repositories.Revenue,
	}
	readServices := ReadRevenueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateRevenueRepositories{
		Revenue: repositories.Revenue,
	}
	updateServices := UpdateRevenueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteRevenueRepositories{
		Revenue: repositories.Revenue,
	}
	deleteServices := DeleteRevenueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListRevenuesRepositories{
		Revenue: repositories.Revenue,
	}
	listServices := ListRevenuesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetRevenueListPageDataRepositories{
		Revenue: repositories.Revenue,
	}
	getListPageDataServices := GetRevenueListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
		AuthorizationService:                   services.AuthorizationService,
		TransactionService:                     services.TransactionService,
		TranslationService:                     services.TranslationService,
		IDService:                              services.IDService,
		MaterializeInstanceJobsForSubscription: services.MaterializeInstanceJobsForSubscription,
	}

	recognizeUC := NewRecognizeRevenueFromSubscriptionUseCase(recognizeRepos, recognizeServices)

	listCandidatesRepos := ListRevenueRunCandidatesRepositories{
		Revenue:      repositories.Revenue,
		Subscription: repositories.Subscription,
		PricePlan:    repositories.PricePlan,
		Workspace:    repositories.Workspace,
	}
	listCandidatesServices := ListRevenueRunCandidatesServices{
		AuthorizationService: services.AuthorizationService,
		TranslationService:   services.TranslationService,
	}

	generateRunRepos := GenerateRevenueRunRepositories{
		Revenue:      repositories.Revenue,
		Subscription: repositories.Subscription,
		RevenueRun:   repositories.RevenueRun,
	}
	generateRunServices := GenerateRevenueRunServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	return &UseCases{
		CreateRevenue:                    NewCreateRevenueUseCase(createRepos, createServices),
		ReadRevenue:                      NewReadRevenueUseCase(readRepos, readServices),
		UpdateRevenue:                    NewUpdateRevenueUseCase(updateRepos, updateServices),
		DeleteRevenue:                    NewDeleteRevenueUseCase(deleteRepos, deleteServices),
		ListRevenues:                     NewListRevenuesUseCase(listRepos, listServices),
		GetRevenueListPageData:           NewGetRevenueListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		RecognizeRevenueFromSubscription: recognizeUC,
		ListRevenueRunCandidates:         NewListRevenueRunCandidatesUseCase(listCandidatesRepos, listCandidatesServices, recognizeUC),
		GenerateRevenueRun:               NewGenerateRevenueRunUseCase(generateRunRepos, generateRunServices, recognizeUC),
	}
}
