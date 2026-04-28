package price_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
)

// PricePlanRepositories groups all repository dependencies for price plan use cases.
//
// PriceSchedule + Client refs are required for the auto-resolve-or-create
// client-scoped PriceSchedule path on CreatePricePlan / UpdatePricePlan
// (plan §3.2 / §4.4 of 20260427-plan-client-scope, wired 2026-04-28).
type PricePlanRepositories struct {
	PricePlan     priceplanpb.PricePlanDomainServiceServer
	Plan          planpb.PlanDomainServiceServer
	PriceSchedule priceschedulepb.PriceScheduleDomainServiceServer
	Client        clientpb.ClientDomainServiceServer
}

// PricePlanServices groups all business service dependencies for price plan use cases
type PricePlanServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService        // Only for CreatePricePlan
	ReferenceChecker     ports.ReferenceChecker // §3.5 — UpdatePricePlan multi-engagement confirm gate
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePricePlanRepositories(repositories)
	updateServices := UpdatePricePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		ReferenceChecker:     services.ReferenceChecker,
		IDService:            services.IDService,
	}

	deleteRepos := DeletePricePlanRepositories{
		PricePlan: repositories.PricePlan,
	}
	deleteServices := DeletePricePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPricePlansRepositories{
		PricePlan: repositories.PricePlan,
	}
	listServices := ListPricePlansServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
