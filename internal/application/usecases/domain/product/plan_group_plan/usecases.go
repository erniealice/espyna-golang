package plan_group_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group_plan"
)

type UseCases struct {
	CreatePlanGroupPlan          *CreatePlanGroupPlanUseCase
	ReadPlanGroupPlan            *ReadPlanGroupPlanUseCase
	UpdatePlanGroupPlan          *UpdatePlanGroupPlanUseCase
	DeletePlanGroupPlan          *DeletePlanGroupPlanUseCase
	ListPlanGroupPlans           *ListPlanGroupPlansUseCase
	GetPlanGroupPlanListPageData *GetPlanGroupPlanListPageDataUseCase
	GetPlanGroupPlanItemPageData *GetPlanGroupPlanItemPageDataUseCase
}

type Repositories struct {
	PlanGroupPlan pb.PlanGroupPlanDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.PlanGroupPlan
	return &UseCases{
		CreatePlanGroupPlan:          NewCreatePlanGroupPlanUseCase(CreatePlanGroupPlanRepositories{PlanGroupPlan: repo}, CreatePlanGroupPlanServices(s)),
		ReadPlanGroupPlan:            NewReadPlanGroupPlanUseCase(ReadPlanGroupPlanRepositories{PlanGroupPlan: repo}, ReadPlanGroupPlanServices(s)),
		UpdatePlanGroupPlan:          NewUpdatePlanGroupPlanUseCase(UpdatePlanGroupPlanRepositories{PlanGroupPlan: repo}, UpdatePlanGroupPlanServices(s)),
		DeletePlanGroupPlan:          NewDeletePlanGroupPlanUseCase(DeletePlanGroupPlanRepositories{PlanGroupPlan: repo}, DeletePlanGroupPlanServices(s)),
		ListPlanGroupPlans:           NewListPlanGroupPlansUseCase(ListPlanGroupPlansRepositories{PlanGroupPlan: repo}, ListPlanGroupPlansServices(s)),
		GetPlanGroupPlanListPageData: NewGetPlanGroupPlanListPageDataUseCase(GetPlanGroupPlanListPageDataRepositories{PlanGroupPlan: repo}, GetPlanGroupPlanListPageDataServices(s)),
		GetPlanGroupPlanItemPageData: NewGetPlanGroupPlanItemPageDataUseCase(GetPlanGroupPlanItemPageDataRepositories{PlanGroupPlan: repo}, GetPlanGroupPlanItemPageDataServices(s)),
	}
}
