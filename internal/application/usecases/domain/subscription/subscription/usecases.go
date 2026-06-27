package subscription

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	Authorizer              ports.Authorizer
	Transactor              ports.Transactor
	Translator              ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator             ports.IDGenerator // Only for CreateSubscription
	JobTemplateInstantiator JobTemplateInstantiator
}

// UseCases contains all subscription-related use cases.
//
// 20260518-hexagonal-strict-adherence Phase 3 — MaterializeJobs +
// MaterializeInstanceJobs (formerly flat fields on SubscriptionUseCases) now
// nest here. The parent subscription aggregator post-assigns them when the
// operation domain repos are available (the same lifecycle pattern as the
// treasury advance use cases). nil-safe.
type UseCases struct {
	CreateSubscription           *CreateSubscriptionUseCase
	ReadSubscription             *ReadSubscriptionUseCase
	UpdateSubscription           *UpdateSubscriptionUseCase
	DeleteSubscription           *DeleteSubscriptionUseCase
	ListSubscriptions            *ListSubscriptionsUseCase
	GetSubscriptionListPageData  *GetSubscriptionListPageDataUseCase
	GetSubscriptionItemPageData  *GetSubscriptionItemPageDataUseCase
	CountActiveByClientIds       *CountActiveByClientIdsUseCase
	ListSubscriptionsByPricePlan *ListSubscriptionsByPricePlanUseCase

	// Job-spawn use cases (Phase 3 F6 closure). Populated post-construction by
	// the parent aggregator (espyna/internal/composition/core/usecases.go)
	// after the operation domain repos resolve. nil-safe.
	MaterializeJobs         *MaterializeJobsForSubscriptionUseCase
	MaterializeInstanceJobs *MaterializeInstanceJobsForSubscriptionUseCase
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
		ActionGatekeeper:        services.ActionGatekeeper,
		Authorizer:              services.Authorizer,
		Transactor:              services.Transactor,
		Translator:              services.Translator,
		IDGenerator:             services.IDGenerator,
		JobTemplateInstantiator: services.JobTemplateInstantiator,
	}

	readRepos := ReadSubscriptionRepositories{
		Subscription: repositories.Subscription,
	}
	readServices := ReadSubscriptionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateSubscriptionRepositories{
		Subscription: repositories.Subscription,
		Client:       repositories.Client,
		PricePlan:    repositories.PricePlan,
	}
	updateServices := UpdateSubscriptionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteSubscriptionRepositories{
		Subscription: repositories.Subscription,
	}
	deleteServices := DeleteSubscriptionServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListSubscriptionsRepositories{
		Subscription: repositories.Subscription,
	}
	listServices := ListSubscriptionsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetSubscriptionListPageDataRepositories{
		Subscription: repositories.Subscription,
	}
	listPageDataServices := GetSubscriptionListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetSubscriptionItemPageDataRepositories{
		Subscription: repositories.Subscription,
	}
	itemPageDataServices := GetSubscriptionItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	countActiveRepos := CountActiveByClientIdsRepositories{
		Subscription: repositories.Subscription,
	}
	countActiveServices := CountActiveByClientIdsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Translator: services.Translator,
	}

	listByPricePlanRepos := ListSubscriptionsByPricePlanRepositories{
		Subscription: repositories.Subscription,
	}
	listByPricePlanServices := ListSubscriptionsByPricePlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateSubscription:           NewCreateSubscriptionUseCase(createRepos, createServices),
		ReadSubscription:             NewReadSubscriptionUseCase(readRepos, readServices),
		UpdateSubscription:           NewUpdateSubscriptionUseCase(updateRepos, updateServices),
		DeleteSubscription:           NewDeleteSubscriptionUseCase(deleteRepos, deleteServices),
		ListSubscriptions:            NewListSubscriptionsUseCase(listRepos, listServices),
		GetSubscriptionListPageData:  NewGetSubscriptionListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetSubscriptionItemPageData:  NewGetSubscriptionItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		CountActiveByClientIds:       NewCountActiveByClientIdsUseCase(countActiveRepos, countActiveServices),
		ListSubscriptionsByPricePlan: NewListSubscriptionsByPricePlanUseCase(listByPricePlanRepos, listByPricePlanServices),
	}
}
