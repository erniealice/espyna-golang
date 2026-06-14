package user

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// UseCases contains all user-related use cases
type UseCases struct {
	CreateUser          *CreateUserUseCase
	ReadUser            *ReadUserUseCase
	UpdateUser          *UpdateUserUseCase
	DeleteUser          *DeleteUserUseCase
	ListUsers           *ListUsersUseCase
	GetUserListPageData *GetUserListPageDataUseCase
	GetUserItemPageData *GetUserItemPageDataUseCase
	ResolveUserByEmail  *ResolveUserByEmailUseCase
}

// UserRepositories groups all repository dependencies for user use cases
type UserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// UserServices groups all business service dependencies for user use cases
type UserServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// NewUseCases creates a new collection of user use cases
func NewUseCases(
	repositories UserRepositories,
	services UserServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateUserRepositories(repositories)
	createServices := CreateUserServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadUserRepositories(repositories)
	readServices := ReadUserServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateUserRepositories(repositories)
	updateServices := UpdateUserServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteUserRepositories(repositories)
	deleteServices := DeleteUserServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListUsersRepositories(repositories)
	listServices := ListUsersServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getUserListPageDataRepos := GetUserListPageDataRepositories(repositories)
	getUserListPageDataServices := GetUserListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getUserItemPageDataRepos := GetUserItemPageDataRepositories(repositories)
	getUserItemPageDataServices := GetUserItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	resolveByEmailRepos := ResolveUserByEmailRepositories(repositories)
	resolveByEmailServices := ResolveUserByEmailServices{
		Translator: services.Translator,
	}

	return &UseCases{
		CreateUser:          NewCreateUserUseCase(createRepos, createServices),
		ReadUser:            NewReadUserUseCase(readRepos, readServices),
		UpdateUser:          NewUpdateUserUseCase(updateRepos, updateServices),
		DeleteUser:          NewDeleteUserUseCase(deleteRepos, deleteServices),
		ListUsers:           NewListUsersUseCase(listRepos, listServices),
		GetUserListPageData: NewGetUserListPageDataUseCase(getUserListPageDataRepos, getUserListPageDataServices),
		GetUserItemPageData: NewGetUserItemPageDataUseCase(getUserItemPageDataRepos, getUserItemPageDataServices),
		ResolveUserByEmail:  NewResolveUserByEmailUseCase(resolveByEmailRepos, resolveByEmailServices),
	}
}

// NewUseCasesUngrouped creates a new collection of user use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(userRepo userpb.UserDomainServiceServer, authorizationService ports.Authorizer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := UserRepositories{
		User: userRepo,
	}

	services := UserServices{
		Authorizer: authorizationService,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
