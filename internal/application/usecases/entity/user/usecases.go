package user

import (
	"leapfor.xyz/espyna/internal/application/ports"
	userpb "leapfor.xyz/esqyma/golang/v1/domain/entity/user"
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
}

// UserRepositories groups all repository dependencies for user use cases
type UserRepositories struct {
	User userpb.UserDomainServiceServer // Primary entity repository
}

// UserServices groups all business service dependencies for user use cases
type UserServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of user use cases
func NewUseCases(
	repositories UserRepositories,
	services UserServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateUserRepositories(repositories)
	createServices := CreateUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadUserRepositories(repositories)
	readServices := ReadUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateUserRepositories(repositories)
	updateServices := UpdateUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteUserRepositories(repositories)
	deleteServices := DeleteUserServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListUsersRepositories(repositories)
	listServices := ListUsersServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getUserListPageDataRepos := GetUserListPageDataRepositories(repositories)
	getUserListPageDataServices := GetUserListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getUserItemPageDataRepos := GetUserItemPageDataRepositories(repositories)
	getUserItemPageDataServices := GetUserItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateUser:          NewCreateUserUseCase(createRepos, createServices),
		ReadUser:            NewReadUserUseCase(readRepos, readServices),
		UpdateUser:          NewUpdateUserUseCase(updateRepos, updateServices),
		DeleteUser:          NewDeleteUserUseCase(deleteRepos, deleteServices),
		ListUsers:           NewListUsersUseCase(listRepos, listServices),
		GetUserListPageData: NewGetUserListPageDataUseCase(getUserListPageDataRepos, getUserListPageDataServices),
		GetUserItemPageData: NewGetUserItemPageDataUseCase(getUserItemPageDataRepos, getUserItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of user use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(userRepo userpb.UserDomainServiceServer, authorizationService ports.AuthorizationService) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := UserRepositories{
		User: userRepo,
	}

	services := UserServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
