package line_workspace_user

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line_workspace_user"
)

type UseCases struct {
	CreateLineWorkspaceUser          *CreateLineWorkspaceUserUseCase
	ReadLineWorkspaceUser            *ReadLineWorkspaceUserUseCase
	UpdateLineWorkspaceUser          *UpdateLineWorkspaceUserUseCase
	DeleteLineWorkspaceUser          *DeleteLineWorkspaceUserUseCase
	ListLineWorkspaceUsers           *ListLineWorkspaceUsersUseCase
	GetLineWorkspaceUserListPageData *GetLineWorkspaceUserListPageDataUseCase
	GetLineWorkspaceUserItemPageData *GetLineWorkspaceUserItemPageDataUseCase
}

type Repositories struct {
	LineWorkspaceUser pb.LineWorkspaceUserDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.LineWorkspaceUser
	return &UseCases{
		CreateLineWorkspaceUser:          NewCreateLineWorkspaceUserUseCase(CreateLineWorkspaceUserRepositories{LineWorkspaceUser: repo}, CreateLineWorkspaceUserServices(s)),
		ReadLineWorkspaceUser:            NewReadLineWorkspaceUserUseCase(ReadLineWorkspaceUserRepositories{LineWorkspaceUser: repo}, ReadLineWorkspaceUserServices(s)),
		UpdateLineWorkspaceUser:          NewUpdateLineWorkspaceUserUseCase(UpdateLineWorkspaceUserRepositories{LineWorkspaceUser: repo}, UpdateLineWorkspaceUserServices(s)),
		DeleteLineWorkspaceUser:          NewDeleteLineWorkspaceUserUseCase(DeleteLineWorkspaceUserRepositories{LineWorkspaceUser: repo}, DeleteLineWorkspaceUserServices(s)),
		ListLineWorkspaceUsers:           NewListLineWorkspaceUsersUseCase(ListLineWorkspaceUsersRepositories{LineWorkspaceUser: repo}, ListLineWorkspaceUsersServices(s)),
		GetLineWorkspaceUserListPageData: NewGetLineWorkspaceUserListPageDataUseCase(GetLineWorkspaceUserListPageDataRepositories{LineWorkspaceUser: repo}, GetLineWorkspaceUserListPageDataServices(s)),
		GetLineWorkspaceUserItemPageData: NewGetLineWorkspaceUserItemPageDataUseCase(GetLineWorkspaceUserItemPageDataRepositories{LineWorkspaceUser: repo}, GetLineWorkspaceUserItemPageDataServices(s)),
	}
}
