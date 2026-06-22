package subscription_group_workspace_user

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_workspace_user"
)

type UseCases struct {
	CreateSubscriptionGroupWorkspaceUser          *CreateSubscriptionGroupWorkspaceUserUseCase
	ReadSubscriptionGroupWorkspaceUser            *ReadSubscriptionGroupWorkspaceUserUseCase
	UpdateSubscriptionGroupWorkspaceUser          *UpdateSubscriptionGroupWorkspaceUserUseCase
	DeleteSubscriptionGroupWorkspaceUser          *DeleteSubscriptionGroupWorkspaceUserUseCase
	ListSubscriptionGroupWorkspaceUsers           *ListSubscriptionGroupWorkspaceUsersUseCase
	GetSubscriptionGroupWorkspaceUserListPageData *GetSubscriptionGroupWorkspaceUserListPageDataUseCase
	GetSubscriptionGroupWorkspaceUserItemPageData *GetSubscriptionGroupWorkspaceUserItemPageDataUseCase
}

type Repositories struct {
	SubscriptionGroupWorkspaceUser pb.SubscriptionGroupWorkspaceUserDomainServiceServer
}

type Services struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

func NewUseCases(r Repositories, s Services) *UseCases {
	repo := r.SubscriptionGroupWorkspaceUser
	return &UseCases{
		CreateSubscriptionGroupWorkspaceUser:          NewCreateSubscriptionGroupWorkspaceUserUseCase(CreateSubscriptionGroupWorkspaceUserRepositories{SubscriptionGroupWorkspaceUser: repo}, CreateSubscriptionGroupWorkspaceUserServices(s)),
		ReadSubscriptionGroupWorkspaceUser:            NewReadSubscriptionGroupWorkspaceUserUseCase(ReadSubscriptionGroupWorkspaceUserRepositories{SubscriptionGroupWorkspaceUser: repo}, ReadSubscriptionGroupWorkspaceUserServices(s)),
		UpdateSubscriptionGroupWorkspaceUser:          NewUpdateSubscriptionGroupWorkspaceUserUseCase(UpdateSubscriptionGroupWorkspaceUserRepositories{SubscriptionGroupWorkspaceUser: repo}, UpdateSubscriptionGroupWorkspaceUserServices(s)),
		DeleteSubscriptionGroupWorkspaceUser:          NewDeleteSubscriptionGroupWorkspaceUserUseCase(DeleteSubscriptionGroupWorkspaceUserRepositories{SubscriptionGroupWorkspaceUser: repo}, DeleteSubscriptionGroupWorkspaceUserServices(s)),
		ListSubscriptionGroupWorkspaceUsers:           NewListSubscriptionGroupWorkspaceUsersUseCase(ListSubscriptionGroupWorkspaceUsersRepositories{SubscriptionGroupWorkspaceUser: repo}, ListSubscriptionGroupWorkspaceUsersServices(s)),
		GetSubscriptionGroupWorkspaceUserListPageData: NewGetSubscriptionGroupWorkspaceUserListPageDataUseCase(GetSubscriptionGroupWorkspaceUserListPageDataRepositories{SubscriptionGroupWorkspaceUser: repo}, GetSubscriptionGroupWorkspaceUserListPageDataServices(s)),
		GetSubscriptionGroupWorkspaceUserItemPageData: NewGetSubscriptionGroupWorkspaceUserItemPageDataUseCase(GetSubscriptionGroupWorkspaceUserItemPageDataRepositories{SubscriptionGroupWorkspaceUser: repo}, GetSubscriptionGroupWorkspaceUserItemPageDataServices(s)),
	}
}
