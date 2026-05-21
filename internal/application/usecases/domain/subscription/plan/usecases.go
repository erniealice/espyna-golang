package plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	priceschedulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// PlanRepositories groups all repository dependencies for plan use cases.
//
// Cross-domain reads were added in 2026-04-27 to support the
// CustomizePlanForClient use case, which clones a Plan tree (Plan,
// ProductPlan, PricePlan, ProductPricePlan, PriceSchedule, optionally
// repointing a Subscription) into a target client's namespace. The legacy
// CRUD use cases continue to use only the Plan field.
type PlanRepositories struct {
	Plan             planpb.PlanDomainServiceServer                         // Primary entity repository
	PricePlan        priceplanpb.PricePlanDomainServiceServer               // Cascade target for client_id sync (§3.2) + customize clone
	ProductPlan      productplanpb.ProductPlanDomainServiceServer           // Customize clone
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer // Customize clone
	PriceSchedule    priceschedulepb.PriceScheduleDomainServiceServer       // Customize resolve-or-create
	Subscription     subscriptionpb.SubscriptionDomainServiceServer         // Customize optional repoint
	Client           clientpb.ClientDomainServiceServer                     // Customize: client existence + display name
}

// PlanServices groups all business service dependencies for plan use cases
type PlanServices struct {
	Authorizer       ports.Authorizer // Current: RBAC and permissions
	Transactor       ports.Transactor // Current: Database transactions
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator      // Only for CreatePlan / CustomizePlanForClient
	ReferenceChecker ports.ReferenceChecker // §3.1 — UpdatePlan client_id reassignment guard
}

// UseCases contains all plan-related use cases
type UseCases struct {
	CreatePlan             *CreatePlanUseCase
	ReadPlan               *ReadPlanUseCase
	UpdatePlan             *UpdatePlanUseCase
	DeletePlan             *DeletePlanUseCase
	ListPlans              *ListPlansUseCase
	GetPlanListPageData    *GetPlanListPageDataUseCase
	GetPlanItemPageData    *GetPlanItemPageDataUseCase
	SearchPlansByName      *SearchPlansByNameUseCase
	CustomizePlanForClient *CustomizePlanForClientUseCase
}

// NewUseCases creates a new collection of plan use cases
func NewUseCases(
	repositories PlanRepositories,
	services PlanServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePlanRepositories{Plan: repositories.Plan}
	createServices := CreatePlanServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPlanRepositories{Plan: repositories.Plan}
	readServices := ReadPlanServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdatePlanRepositories{
		Plan:      repositories.Plan,
		PricePlan: repositories.PricePlan,
	}
	updateServices := UpdatePlanServices{
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		ReferenceChecker: services.ReferenceChecker,
	}

	deleteRepos := DeletePlanRepositories{Plan: repositories.Plan}
	deleteServices := DeletePlanServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPlansRepositories{Plan: repositories.Plan}
	listServices := ListPlansServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetPlanListPageDataRepositories{
		Plan: repositories.Plan,
	}
	listPageDataServices := GetPlanListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetPlanItemPageDataRepositories{
		Plan: repositories.Plan,
	}
	itemPageDataServices := GetPlanItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	searchByNameRepos := SearchPlansByNameRepositories{
		Plan: repositories.Plan,
	}
	searchByNameServices := SearchPlansByNameServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	customizeRepos := CustomizePlanForClientRepositories{
		Plan:             repositories.Plan,
		PricePlan:        repositories.PricePlan,
		ProductPlan:      repositories.ProductPlan,
		ProductPricePlan: repositories.ProductPricePlan,
		PriceSchedule:    repositories.PriceSchedule,
		Subscription:     repositories.Subscription,
		Client:           repositories.Client,
	}
	customizeServices := CustomizePlanForClientServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	return &UseCases{
		CreatePlan:             NewCreatePlanUseCase(createRepos, createServices),
		ReadPlan:               NewReadPlanUseCase(readRepos, readServices),
		UpdatePlan:             NewUpdatePlanUseCase(updateRepos, updateServices),
		DeletePlan:             NewDeletePlanUseCase(deleteRepos, deleteServices),
		ListPlans:              NewListPlansUseCase(listRepos, listServices),
		GetPlanListPageData:    NewGetPlanListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPlanItemPageData:    NewGetPlanItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
		SearchPlansByName:      NewSearchPlansByNameUseCase(searchByNameRepos, searchByNameServices),
		CustomizePlanForClient: NewCustomizePlanForClientUseCase(customizeRepos, customizeServices),
	}
}
