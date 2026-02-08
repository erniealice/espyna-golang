package price_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
)

// PricePlanRepositories groups all repository dependencies for price plan use cases
type PricePlanRepositories struct {
	PricePlan priceplanpb.PricePlanDomainServiceServer // Primary entity repository
	Plan      planpb.PlanDomainServiceServer           // Entity reference dependency
}

// PricePlanServices groups all business service dependencies for price plan use cases
type PricePlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService // Only for CreatePricePlan
}

// UseCases contains all price_plan-related use cases
type UseCases struct {
	CreatePricePlan          *CreatePricePlanUseCase
	ReadPricePlan            *ReadPricePlanUseCase
	UpdatePricePlan          *UpdatePricePlanUseCase
	DeletePricePlan          *DeletePricePlanUseCase
	ListPricePlans           *ListPricePlansUseCase
	GetPricePlanListPageData *GetPricePlanListPageDataUseCase
	GetPricePlanItemPageData *GetPricePlanItemPageDataUseCase
}

// NewUseCases creates a new collection of price_plan use cases
func NewUseCases(
	repositories PricePlanRepositories,
	services PricePlanServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePricePlanRepositories(repositories)
	createServices := CreatePricePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPricePlanRepositories{
		PricePlan: repositories.PricePlan,
	}
	readServices := ReadPricePlanServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdatePricePlanRepositories(repositories)
	updateServices := UpdatePricePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePricePlanRepositories{
		PricePlan: repositories.PricePlan,
	}
	deleteServices := DeletePricePlanServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListPricePlansRepositories{
		PricePlan: repositories.PricePlan,
	}
	listServices := ListPricePlansServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listPageDataRepos := GetPricePlanListPageDataRepositories{
		PricePlan: repositories.PricePlan,
	}
	listPageDataServices := GetPricePlanListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetPricePlanItemPageDataRepositories{
		PricePlan: repositories.PricePlan,
	}
	itemPageDataServices := GetPricePlanItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePricePlan:          NewCreatePricePlanUseCase(createRepos, createServices),
		ReadPricePlan:            NewReadPricePlanUseCase(readRepos, readServices),
		UpdatePricePlan:          NewUpdatePricePlanUseCase(updateRepos, updateServices),
		DeletePricePlan:          NewDeletePricePlanUseCase(deleteRepos, deleteServices),
		ListPricePlans:           NewListPricePlansUseCase(listRepos, listServices),
		GetPricePlanListPageData: NewGetPricePlanListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPricePlanItemPageData: NewGetPricePlanItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
