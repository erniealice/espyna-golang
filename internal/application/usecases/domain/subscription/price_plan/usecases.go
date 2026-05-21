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
	Authorizer       ports.Authorizer // Current: RBAC and permissions
	Transactor       ports.Transactor // Current: Database transactions
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator      // Only for CreatePricePlan
	ReferenceChecker ports.ReferenceChecker // §3.5 — UpdatePricePlan multi-engagement confirm gate
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPricePlanRepositories{
		PricePlan: repositories.PricePlan,
	}
	readServices := ReadPricePlanServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdatePricePlanRepositories(repositories)
	updateServices := UpdatePricePlanServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ReferenceChecker: services.ReferenceChecker,
		IDGenerator:      services.IDGenerator,
	}

	deleteRepos := DeletePricePlanRepositories{
		PricePlan: repositories.PricePlan,
	}
	deleteServices := DeletePricePlanServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPricePlansRepositories{
		PricePlan: repositories.PricePlan,
	}
	listServices := ListPricePlansServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetPricePlanListPageDataRepositories{
		PricePlan: repositories.PricePlan,
	}
	listPageDataServices := GetPricePlanListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetPricePlanItemPageDataRepositories{
		PricePlan: repositories.PricePlan,
	}
	itemPageDataServices := GetPricePlanItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
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
