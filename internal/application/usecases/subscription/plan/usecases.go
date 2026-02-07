package plan

import (
	"leapfor.xyz/espyna/internal/application/ports"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
)

// PlanRepositories groups all repository dependencies for plan use cases
type PlanRepositories struct {
	Plan planpb.PlanDomainServiceServer // Primary entity repository
}

// PlanServices groups all business service dependencies for plan use cases
type PlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Only for CreatePlan
}

// UseCases contains all plan-related use cases
type UseCases struct {
	CreatePlan          *CreatePlanUseCase
	ReadPlan            *ReadPlanUseCase
	UpdatePlan          *UpdatePlanUseCase
	DeletePlan          *DeletePlanUseCase
	ListPlans           *ListPlansUseCase
	GetPlanListPageData *GetPlanListPageDataUseCase
	GetPlanItemPageData *GetPlanItemPageDataUseCase
}

// NewUseCases creates a new collection of plan use cases
func NewUseCases(
	repositories PlanRepositories,
	services PlanServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePlanRepositories(repositories)
	createServices := CreatePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPlanRepositories(repositories)
	readServices := ReadPlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePlanRepositories(repositories)
	updateServices := UpdatePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePlanRepositories(repositories)
	deleteServices := DeletePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPlansRepositories(repositories)
	listServices := ListPlansServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetPlanListPageDataRepositories{
		Plan: repositories.Plan,
	}
	listPageDataServices := GetPlanListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetPlanItemPageDataRepositories{
		Plan: repositories.Plan,
	}
	itemPageDataServices := GetPlanItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePlan:          NewCreatePlanUseCase(createRepos, createServices),
		ReadPlan:            NewReadPlanUseCase(readRepos, readServices),
		UpdatePlan:          NewUpdatePlanUseCase(updateRepos, updateServices),
		DeletePlan:          NewDeletePlanUseCase(deleteRepos, deleteServices),
		ListPlans:           NewListPlansUseCase(listRepos, listServices),
		GetPlanListPageData: NewGetPlanListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPlanItemPageData: NewGetPlanItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
