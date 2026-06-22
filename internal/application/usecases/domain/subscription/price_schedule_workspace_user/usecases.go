package price_schedule_workspace_user

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_schedule_workspace_user"
)

type UseCases struct {
	CreatePriceScheduleWorkspaceUser          *CreatePriceScheduleWorkspaceUserUseCase
	ReadPriceScheduleWorkspaceUser            *ReadPriceScheduleWorkspaceUserUseCase
	UpdatePriceScheduleWorkspaceUser          *UpdatePriceScheduleWorkspaceUserUseCase
	DeletePriceScheduleWorkspaceUser          *DeletePriceScheduleWorkspaceUserUseCase
	ListPriceScheduleWorkspaceUsers           *ListPriceScheduleWorkspaceUsersUseCase
	GetPriceScheduleWorkspaceUserListPageData *GetPriceScheduleWorkspaceUserListPageDataUseCase
	GetPriceScheduleWorkspaceUserItemPageData *GetPriceScheduleWorkspaceUserItemPageDataUseCase
}

type Repositories struct {
	PriceScheduleWorkspaceUser pb.PriceScheduleWorkspaceUserDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.PriceScheduleWorkspaceUser
	return &UseCases{
		CreatePriceScheduleWorkspaceUser:          NewCreatePriceScheduleWorkspaceUserUseCase(CreatePriceScheduleWorkspaceUserRepositories{PriceScheduleWorkspaceUser: repo}, CreatePriceScheduleWorkspaceUserServices(s)),
		ReadPriceScheduleWorkspaceUser:            NewReadPriceScheduleWorkspaceUserUseCase(ReadPriceScheduleWorkspaceUserRepositories{PriceScheduleWorkspaceUser: repo}, ReadPriceScheduleWorkspaceUserServices(s)),
		UpdatePriceScheduleWorkspaceUser:          NewUpdatePriceScheduleWorkspaceUserUseCase(UpdatePriceScheduleWorkspaceUserRepositories{PriceScheduleWorkspaceUser: repo}, UpdatePriceScheduleWorkspaceUserServices(s)),
		DeletePriceScheduleWorkspaceUser:          NewDeletePriceScheduleWorkspaceUserUseCase(DeletePriceScheduleWorkspaceUserRepositories{PriceScheduleWorkspaceUser: repo}, DeletePriceScheduleWorkspaceUserServices(s)),
		ListPriceScheduleWorkspaceUsers:           NewListPriceScheduleWorkspaceUsersUseCase(ListPriceScheduleWorkspaceUsersRepositories{PriceScheduleWorkspaceUser: repo}, ListPriceScheduleWorkspaceUsersServices(s)),
		GetPriceScheduleWorkspaceUserListPageData: NewGetPriceScheduleWorkspaceUserListPageDataUseCase(GetPriceScheduleWorkspaceUserListPageDataRepositories{PriceScheduleWorkspaceUser: repo}, GetPriceScheduleWorkspaceUserListPageDataServices(s)),
		GetPriceScheduleWorkspaceUserItemPageData: NewGetPriceScheduleWorkspaceUserItemPageDataUseCase(GetPriceScheduleWorkspaceUserItemPageDataRepositories{PriceScheduleWorkspaceUser: repo}, GetPriceScheduleWorkspaceUserItemPageDataServices(s)),
	}
}
