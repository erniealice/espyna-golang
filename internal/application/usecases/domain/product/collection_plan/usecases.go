package collection_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	collectionplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_plan"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
)

// CollectionPlanRepositories groups all repository dependencies for collection plan use cases
type CollectionPlanRepositories struct {
	CollectionPlan collectionplanpb.CollectionPlanDomainServiceServer // Primary entity repository
	Collection     collectionpb.CollectionDomainServiceServer         // Entity reference: collection_plan.collection_id -> collection.id
	Plan           planpb.PlanDomainServiceServer                     // Entity reference: collection_plan.plan_id -> plan.id
}

// CollectionPlanServices groups all business service dependencies for collection plan use cases
type CollectionPlanServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all collection plan-related use cases
type UseCases struct {
	CreateCollectionPlan          *CreateCollectionPlanUseCase
	ReadCollectionPlan            *ReadCollectionPlanUseCase
	UpdateCollectionPlan          *UpdateCollectionPlanUseCase
	DeleteCollectionPlan          *DeleteCollectionPlanUseCase
	ListCollectionPlans           *ListCollectionPlansUseCase
	GetCollectionPlanListPageData *GetCollectionPlanListPageDataUseCase
	GetCollectionPlanItemPageData *GetCollectionPlanItemPageDataUseCase
}

// NewUseCases creates a new collection of collection plan use cases with entity reference dependencies
func NewUseCases(
	repositories CollectionPlanRepositories,
	services CollectionPlanServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateCollectionPlanRepositories(repositories)
	createServices := CreateCollectionPlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadCollectionPlanRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	readServices := ReadCollectionPlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateCollectionPlanRepositories(repositories)
	updateServices := UpdateCollectionPlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteCollectionPlanRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	deleteServices := DeleteCollectionPlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListCollectionPlansRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	listServices := ListCollectionPlansServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetCollectionPlanListPageDataRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	listPageDataServices := GetCollectionPlanListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetCollectionPlanItemPageDataRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	itemPageDataServices := GetCollectionPlanItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateCollectionPlan:          NewCreateCollectionPlanUseCase(createRepos, createServices),
		ReadCollectionPlan:            NewReadCollectionPlanUseCase(readRepos, readServices),
		UpdateCollectionPlan:          NewUpdateCollectionPlanUseCase(updateRepos, updateServices),
		DeleteCollectionPlan:          NewDeleteCollectionPlanUseCase(deleteRepos, deleteServices),
		ListCollectionPlans:           NewListCollectionPlansUseCase(listRepos, listServices),
		GetCollectionPlanListPageData: NewGetCollectionPlanListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetCollectionPlanItemPageData: NewGetCollectionPlanItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
