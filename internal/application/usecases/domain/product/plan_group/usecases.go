package plan_group

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/plan_group"
)

type UseCases struct {
	CreatePlanGroup          *CreatePlanGroupUseCase
	ReadPlanGroup            *ReadPlanGroupUseCase
	UpdatePlanGroup          *UpdatePlanGroupUseCase
	DeletePlanGroup          *DeletePlanGroupUseCase
	ListPlanGroups           *ListPlanGroupsUseCase
	GetPlanGroupListPageData *GetPlanGroupListPageDataUseCase
	GetPlanGroupItemPageData *GetPlanGroupItemPageDataUseCase
}

type Repositories struct {
	PlanGroup pb.PlanGroupDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.PlanGroup
	return &UseCases{
		CreatePlanGroup:          NewCreatePlanGroupUseCase(CreatePlanGroupRepositories{PlanGroup: repo}, CreatePlanGroupServices(s)),
		ReadPlanGroup:            NewReadPlanGroupUseCase(ReadPlanGroupRepositories{PlanGroup: repo}, ReadPlanGroupServices(s)),
		UpdatePlanGroup:          NewUpdatePlanGroupUseCase(UpdatePlanGroupRepositories{PlanGroup: repo}, UpdatePlanGroupServices(s)),
		DeletePlanGroup:          NewDeletePlanGroupUseCase(DeletePlanGroupRepositories{PlanGroup: repo}, DeletePlanGroupServices(s)),
		ListPlanGroups:           NewListPlanGroupsUseCase(ListPlanGroupsRepositories{PlanGroup: repo}, ListPlanGroupsServices(s)),
		GetPlanGroupListPageData: NewGetPlanGroupListPageDataUseCase(GetPlanGroupListPageDataRepositories{PlanGroup: repo}, GetPlanGroupListPageDataServices(s)),
		GetPlanGroupItemPageData: NewGetPlanGroupItemPageDataUseCase(GetPlanGroupItemPageDataRepositories{PlanGroup: repo}, GetPlanGroupItemPageDataServices(s)),
	}
}
