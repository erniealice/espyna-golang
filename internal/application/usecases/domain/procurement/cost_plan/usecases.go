package cost_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
	costplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/cost_plan"
)

// Repositories groups all repository dependencies for cost_plan use cases
type Repositories struct {
	CostPlan  costplanpb.CostPlanDomainServiceServer
	Workspace workspacepb.WorkspaceDomainServiceServer // Cross-domain: currency hard-block on create
}

// Services groups all business service dependencies
type Services struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all cost_plan-related use cases
type UseCases struct {
	CreateCostPlan          *CreateCostPlanUseCase
	ReadCostPlan            *ReadCostPlanUseCase
	UpdateCostPlan          *UpdateCostPlanUseCase
	DeleteCostPlan          *DeleteCostPlanUseCase
	ListCostPlans           *ListCostPlansUseCase
	GetCostPlanListPageData *GetCostPlanListPageDataUseCase
	GetCostPlanItemPageData *GetCostPlanItemPageDataUseCase
}

// NewUseCases creates a new collection of cost_plan use cases
func NewUseCases(repos Repositories, svcs Services) *UseCases {
	return &UseCases{
		CreateCostPlan: NewCreateCostPlanUseCase(
			CreateCostPlanRepositories{CostPlan: repos.CostPlan, Workspace: repos.Workspace},
			CreateCostPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService, IDService: svcs.IDService},
		),
		ReadCostPlan: NewReadCostPlanUseCase(
			ReadCostPlanRepositories{CostPlan: repos.CostPlan},
			ReadCostPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		UpdateCostPlan: NewUpdateCostPlanUseCase(
			UpdateCostPlanRepositories{CostPlan: repos.CostPlan},
			UpdateCostPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		DeleteCostPlan: NewDeleteCostPlanUseCase(
			DeleteCostPlanRepositories{CostPlan: repos.CostPlan},
			DeleteCostPlanServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		ListCostPlans: NewListCostPlansUseCase(
			ListCostPlansRepositories{CostPlan: repos.CostPlan},
			ListCostPlansServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetCostPlanListPageData: NewGetCostPlanListPageDataUseCase(
			GetCostPlanListPageDataRepositories{CostPlan: repos.CostPlan},
			GetCostPlanListPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
		GetCostPlanItemPageData: NewGetCostPlanItemPageDataUseCase(
			GetCostPlanItemPageDataRepositories{CostPlan: repos.CostPlan},
			GetCostPlanItemPageDataServices{AuthorizationService: svcs.AuthorizationService, TransactionService: svcs.TransactionService, TranslationService: svcs.TranslationService},
		),
	}
}
