package cost_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
			CreateCostPlanServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator, IDGenerator: svcs.IDGenerator},
		),
		ReadCostPlan: NewReadCostPlanUseCase(
			ReadCostPlanRepositories{CostPlan: repos.CostPlan},
			ReadCostPlanServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		UpdateCostPlan: NewUpdateCostPlanUseCase(
			UpdateCostPlanRepositories{CostPlan: repos.CostPlan},
			UpdateCostPlanServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		DeleteCostPlan: NewDeleteCostPlanUseCase(
			DeleteCostPlanRepositories{CostPlan: repos.CostPlan},
			DeleteCostPlanServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		ListCostPlans: NewListCostPlansUseCase(
			ListCostPlansRepositories{CostPlan: repos.CostPlan},
			ListCostPlansServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetCostPlanListPageData: NewGetCostPlanListPageDataUseCase(
			GetCostPlanListPageDataRepositories{CostPlan: repos.CostPlan},
			GetCostPlanListPageDataServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
		GetCostPlanItemPageData: NewGetCostPlanItemPageDataUseCase(
			GetCostPlanItemPageDataRepositories{CostPlan: repos.CostPlan},
			GetCostPlanItemPageDataServices{ActionGatekeeper: svcs.ActionGatekeeper, Authorizer: svcs.Authorizer, Transactor: svcs.Transactor, Translator: svcs.Translator},
		),
	}
}
