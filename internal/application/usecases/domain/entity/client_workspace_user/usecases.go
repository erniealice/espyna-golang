package client_workspace_user

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"
)

// UseCases contains all client workspace user-related use cases
type UseCases struct {
	CreateClientWorkspaceUser *CreateClientWorkspaceUserUseCase
	ReadClientWorkspaceUser   *ReadClientWorkspaceUserUseCase
	UpdateClientWorkspaceUser *UpdateClientWorkspaceUserUseCase
	DeleteClientWorkspaceUser *DeleteClientWorkspaceUserUseCase
	ListClientWorkspaceUsers  *ListClientWorkspaceUsersUseCase
}

// ClientWorkspaceUserRepositories groups all repository dependencies for client workspace user use cases
type ClientWorkspaceUserRepositories struct {
	ClientWorkspaceUser clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer // Primary entity repository
	Client              clientpb.ClientDomainServiceServer                           // FK validation
}

// ClientWorkspaceUserServices groups all business service dependencies for client workspace user use cases
type ClientWorkspaceUserServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of client workspace user use cases
func NewUseCases(
	repositories ClientWorkspaceUserRepositories,
	services ClientWorkspaceUserServices,
) *UseCases {
	return &UseCases{
		CreateClientWorkspaceUser: NewCreateClientWorkspaceUserUseCase(
			CreateClientWorkspaceUserRepositories{
				ClientWorkspaceUser: repositories.ClientWorkspaceUser,
				Client:              repositories.Client,
			},
			CreateClientWorkspaceUserServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadClientWorkspaceUser: NewReadClientWorkspaceUserUseCase(
			ReadClientWorkspaceUserRepositories{ClientWorkspaceUser: repositories.ClientWorkspaceUser},
			ReadClientWorkspaceUserServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		UpdateClientWorkspaceUser: NewUpdateClientWorkspaceUserUseCase(
			UpdateClientWorkspaceUserRepositories{ClientWorkspaceUser: repositories.ClientWorkspaceUser},
			UpdateClientWorkspaceUserServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		DeleteClientWorkspaceUser: NewDeleteClientWorkspaceUserUseCase(
			DeleteClientWorkspaceUserRepositories{ClientWorkspaceUser: repositories.ClientWorkspaceUser},
			DeleteClientWorkspaceUserServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ListClientWorkspaceUsers: NewListClientWorkspaceUsersUseCase(
			ListClientWorkspaceUsersRepositories{ClientWorkspaceUser: repositories.ClientWorkspaceUser},
			ListClientWorkspaceUsersServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
