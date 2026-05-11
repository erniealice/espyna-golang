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
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService        // Only for CreatePlan / CustomizePlanForClient
	ReferenceChecker     ports.ReferenceChecker // §3.1 — UpdatePlan client_id reassignment guard
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPlanRepositories{Plan: repositories.Plan}
	readServices := ReadPlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePlanRepositories{
		Plan:      repositories.Plan,
		PricePlan: repositories.PricePlan,
	}
	updateServices := UpdatePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		ReferenceChecker:     services.ReferenceChecker,
	}

	deleteRepos := DeletePlanRepositories{Plan: repositories.Plan}
	deleteServices := DeletePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPlansRepositories{Plan: repositories.Plan}
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

	searchByNameRepos := SearchPlansByNameRepositories{
		Plan: repositories.Plan,
	}
	searchByNameServices := SearchPlansByNameServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
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
