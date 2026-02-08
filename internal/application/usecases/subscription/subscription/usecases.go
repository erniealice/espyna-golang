package subscription

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// SubscriptionRepositories groups all repository dependencies
type SubscriptionRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	PricePlan    priceplanpb.PricePlanDomainServiceServer
}

// SubscriptionServices groups all business service dependencies
type SubscriptionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Only for CreateSubscription
}

// UseCases contains all subscription-related use cases
type UseCases struct {
	CreateSubscription          *CreateSubscriptionUseCase
	ReadSubscription            *ReadSubscriptionUseCase
	UpdateSubscription          *UpdateSubscriptionUseCase
	DeleteSubscription          *DeleteSubscriptionUseCase
	ListSubscriptions           *ListSubscriptionsUseCase
	GetSubscriptionListPageData *GetSubscriptionListPageDataUseCase
	GetSubscriptionItemPageData *GetSubscriptionItemPageDataUseCase
}

// NewUseCases creates a new collection of subscription use cases
func NewUseCases(
	repositories SubscriptionRepositories,
	services SubscriptionServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateSubscriptionRepositories{
		Subscription: repositories.Subscription,
		Client:       repositories.Client,
		PricePlan:    repositories.PricePlan,
	}
	createServices := CreateSubscriptionServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
		IDService:          services.IDService,
	}

	readRepos := ReadSubscriptionRepositories{
		Subscription: repositories.Subscription,
	}
	readServices := ReadSubscriptionServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateSubscriptionRepositories{
		Subscription: repositories.Subscription,
		Client:       repositories.Client,
		PricePlan:    repositories.PricePlan,
	}
	updateServices := UpdateSubscriptionServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteSubscriptionRepositories{
		Subscription: repositories.Subscription,
	}
	deleteServices := DeleteSubscriptionServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListSubscriptionsRepositories{
		Subscription: repositories.Subscription,
	}
	listServices := ListSubscriptionsServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listPageDataRepos := GetSubscriptionListPageDataRepositories{
		Subscription: repositories.Subscription,
	}
	listPageDataServices := GetSubscriptionListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetSubscriptionItemPageDataRepositories{
		Subscription: repositories.Subscription,
	}
	itemPageDataServices := GetSubscriptionItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateSubscription:          NewCreateSubscriptionUseCase(createRepos, createServices),
		ReadSubscription:            NewReadSubscriptionUseCase(readRepos, readServices),
		UpdateSubscription:          NewUpdateSubscriptionUseCase(updateRepos, updateServices),
		DeleteSubscription:          NewDeleteSubscriptionUseCase(deleteRepos, deleteServices),
		ListSubscriptions:           NewListSubscriptionsUseCase(listRepos, listServices),
		GetSubscriptionListPageData: NewGetSubscriptionListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetSubscriptionItemPageData: NewGetSubscriptionItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
