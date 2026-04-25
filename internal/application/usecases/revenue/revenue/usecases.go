package revenue

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"

	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
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
}

// RevenueServices groups all business service dependencies for revenue use cases
type RevenueServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
	}
	recognizeServices := RecognizeRevenueFromSubscriptionServices{
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
		RecognizeRevenueFromSubscription: NewRecognizeRevenueFromSubscriptionUseCase(recognizeRepos, recognizeServices),
	}
}
