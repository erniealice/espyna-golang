package subscription_workspace_user

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	clientworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_workspace_user"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
	subscriptionworkspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_workspace_user"
)

// UseCases contains all subscription workspace user-related use cases
type UseCases struct {
	CreateSubscriptionWorkspaceUser          *CreateSubscriptionWorkspaceUserUseCase
	ReadSubscriptionWorkspaceUser            *ReadSubscriptionWorkspaceUserUseCase
	UpdateSubscriptionWorkspaceUser          *UpdateSubscriptionWorkspaceUserUseCase
	DeleteSubscriptionWorkspaceUser          *DeleteSubscriptionWorkspaceUserUseCase
	ListSubscriptionWorkspaceUsers           *ListSubscriptionWorkspaceUsersUseCase
	GetSubscriptionWorkspaceUserListPageData *GetSubscriptionWorkspaceUserListPageDataUseCase
	GetSubscriptionWorkspaceUserItemPageData *GetSubscriptionWorkspaceUserItemPageDataUseCase
}

// SubscriptionWorkspaceUserRepositories groups all repository dependencies for subscription workspace user use cases
type SubscriptionWorkspaceUserRepositories struct {
	SubscriptionWorkspaceUser subscriptionworkspaceuserpb.SubscriptionWorkspaceUserDomainServiceServer // Primary entity repository
	Subscription              subscriptionpb.SubscriptionDomainServiceServer                           // client_id stamping + FK validation
	ClientWorkspaceUser       clientworkspaceuserpb.ClientWorkspaceUserDomainServiceServer             // composite-FK pre-check
}

// SubscriptionWorkspaceUserServices groups all business service dependencies for subscription workspace user use cases
type SubscriptionWorkspaceUserServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of subscription workspace user use cases
func NewUseCases(
	repositories SubscriptionWorkspaceUserRepositories,
	services SubscriptionWorkspaceUserServices,
) *UseCases {
	return &UseCases{
		CreateSubscriptionWorkspaceUser: NewCreateSubscriptionWorkspaceUserUseCase(
			CreateSubscriptionWorkspaceUserRepositories{
				SubscriptionWorkspaceUser: repositories.SubscriptionWorkspaceUser,
				Subscription:              repositories.Subscription,
				ClientWorkspaceUser:       repositories.ClientWorkspaceUser,
			},
			CreateSubscriptionWorkspaceUserServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadSubscriptionWorkspaceUser: NewReadSubscriptionWorkspaceUserUseCase(
			ReadSubscriptionWorkspaceUserRepositories{SubscriptionWorkspaceUser: repositories.SubscriptionWorkspaceUser},
			ReadSubscriptionWorkspaceUserServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		UpdateSubscriptionWorkspaceUser: NewUpdateSubscriptionWorkspaceUserUseCase(
			UpdateSubscriptionWorkspaceUserRepositories{SubscriptionWorkspaceUser: repositories.SubscriptionWorkspaceUser},
			UpdateSubscriptionWorkspaceUserServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		DeleteSubscriptionWorkspaceUser: NewDeleteSubscriptionWorkspaceUserUseCase(
			DeleteSubscriptionWorkspaceUserRepositories{SubscriptionWorkspaceUser: repositories.SubscriptionWorkspaceUser},
			DeleteSubscriptionWorkspaceUserServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		ListSubscriptionWorkspaceUsers: NewListSubscriptionWorkspaceUsersUseCase(
			ListSubscriptionWorkspaceUsersRepositories{SubscriptionWorkspaceUser: repositories.SubscriptionWorkspaceUser},
			ListSubscriptionWorkspaceUsersServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		GetSubscriptionWorkspaceUserListPageData: NewGetSubscriptionWorkspaceUserListPageDataUseCase(
			GetSubscriptionWorkspaceUserListPageDataRepositories{SubscriptionWorkspaceUser: repositories.SubscriptionWorkspaceUser},
			GetSubscriptionWorkspaceUserListPageDataServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
		GetSubscriptionWorkspaceUserItemPageData: NewGetSubscriptionWorkspaceUserItemPageDataUseCase(
			GetSubscriptionWorkspaceUserItemPageDataRepositories{SubscriptionWorkspaceUser: repositories.SubscriptionWorkspaceUser},
			GetSubscriptionWorkspaceUserItemPageDataServices{
				Authorizer: services.Authorizer,
				Transactor: services.Transactor,
				Translator: services.Translator,
			},
		),
	}
}
