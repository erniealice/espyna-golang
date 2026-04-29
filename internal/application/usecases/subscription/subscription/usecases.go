package subscription

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// JobTemplateInstantiator creates job hierarchies from templates linked to a plan.
// Optional — if nil, no jobs are created on subscription creation.
//
// 2026-04-29 auto-spawn-jobs-from-subscription plan §5.1 — the spawnJobs bool
// carries the operator's "Spawn Jobs on Create" toggle decision from the
// centymo view layer through CreateSubscriptionUseCase. When false, the
// implementor must skip the spawn entirely (the new
// MaterializeJobsForSubscriptionUseCase short-circuits on this flag with
// SkipReasonOperatorOptOut). When true, normal spawn proceeds.
type JobTemplateInstantiator interface {
	InstantiateJobsFromPlan(ctx context.Context, planID, clientID, subscriptionID, workspaceID string, spawnJobs bool) error
}

// SubscriptionRepositories groups all repository dependencies
type SubscriptionRepositories struct {
	Subscription subscriptionpb.SubscriptionDomainServiceServer
	Client       clientpb.ClientDomainServiceServer
	PricePlan    priceplanpb.PricePlanDomainServiceServer
}

// SubscriptionServices groups all business service dependencies
type SubscriptionServices struct {
	AuthorizationService    ports.AuthorizationService
	TransactionService      ports.TransactionService
	TranslationService      ports.TranslationService
	IDService               ports.IDService // Only for CreateSubscription
	JobTemplateInstantiator JobTemplateInstantiator
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
		AuthorizationService:    services.AuthorizationService,
		TransactionService:      services.TransactionService,
		TranslationService:      services.TranslationService,
		IDService:               services.IDService,
		JobTemplateInstantiator: services.JobTemplateInstantiator,
	}

	readRepos := ReadSubscriptionRepositories{
		Subscription: repositories.Subscription,
	}
	readServices := ReadSubscriptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateSubscriptionRepositories{
		Subscription: repositories.Subscription,
		Client:       repositories.Client,
		PricePlan:    repositories.PricePlan,
	}
	updateServices := UpdateSubscriptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteSubscriptionRepositories{
		Subscription: repositories.Subscription,
	}
	deleteServices := DeleteSubscriptionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListSubscriptionsRepositories{
		Subscription: repositories.Subscription,
	}
	listServices := ListSubscriptionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetSubscriptionListPageDataRepositories{
		Subscription: repositories.Subscription,
	}
	listPageDataServices := GetSubscriptionListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetSubscriptionItemPageDataRepositories{
		Subscription: repositories.Subscription,
	}
	itemPageDataServices := GetSubscriptionItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
