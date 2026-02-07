package collection_plan

import (
	"leapfor.xyz/espyna/internal/application/ports"
	collectionpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection"
	collectionplanpb "leapfor.xyz/esqyma/golang/v1/domain/product/collection_plan"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
)

// CollectionPlanRepositories groups all repository dependencies for collection plan use cases
type CollectionPlanRepositories struct {
	CollectionPlan collectionplanpb.CollectionPlanDomainServiceServer // Primary entity repository
	Collection     collectionpb.CollectionDomainServiceServer         // Entity reference: collection_plan.collection_id -> collection.id
	Plan           planpb.PlanDomainServiceServer                     // Entity reference: collection_plan.plan_id -> plan.id
}

// CollectionPlanServices groups all business service dependencies for collection plan use cases
type CollectionPlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadCollectionPlanRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	readServices := ReadCollectionPlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateCollectionPlanRepositories(repositories)
	updateServices := UpdateCollectionPlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteCollectionPlanRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	deleteServices := DeleteCollectionPlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListCollectionPlansRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	listServices := ListCollectionPlansServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetCollectionPlanListPageDataRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	listPageDataServices := GetCollectionPlanListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetCollectionPlanItemPageDataRepositories{
		CollectionPlan: repositories.CollectionPlan,
	}
	itemPageDataServices := GetCollectionPlanItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
