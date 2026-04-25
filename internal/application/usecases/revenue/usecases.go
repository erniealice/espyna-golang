package revenue

import (
	// Revenue use cases
	deferredRevenueUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/deferred_revenue"
	revenueUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/revenue"
	revenueAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/revenue_attribute"
	revenueCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/revenue_category"
	revenueLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/revenue/revenue_line_item"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services - Entity domain (cross-domain dependency)
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"

	// Protobuf domain services for revenue repositories
	deferredrevenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/deferred_revenue"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenueattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_attribute"
	revenuecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_category"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"

	// Cross-domain dependencies for the recognize-revenue use case
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
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
	// Cross-domain dependency: payment term lookup for due date computation
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer

	// Cross-domain dependencies for revenue recognition from a subscription.
	// All optional from a wiring standpoint — but all required for the
	// RecognizeRevenueFromSubscription use case to actually fire.
	Subscription     subscriptionpb.SubscriptionDomainServiceServer
	PricePlan        priceplanpb.PricePlanDomainServiceServer
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
	PriceSchedule    priceschedulepb.PriceScheduleDomainServiceServer
	Client           clientpb.ClientDomainServiceServer
}

// RevenueUseCases contains all revenue-related use cases
type RevenueUseCases struct {
	Revenue          *revenueUseCases.UseCases
	RevenueLineItem  *revenueLineItemUseCases.UseCases
	RevenueCategory  *revenueCategoryUseCases.UseCases
	RevenueAttribute *revenueAttributeUseCases.UseCases
	DeferredRevenue  *deferredRevenueUseCases.UseCases
}

// NewUseCases creates all revenue use cases with proper constructor injection
func NewUseCases(
	repos RevenueRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *RevenueUseCases {
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
		},
		revenueUseCases.RevenueServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
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

	return &RevenueUseCases{
		Revenue:          revenueUC,
		RevenueLineItem:  revenueLineItemUC,
		RevenueCategory:  revenueCategoryUC,
		RevenueAttribute: revenueAttributeUC,
		DeferredRevenue:  deferredRevenueUC,
	}
}
