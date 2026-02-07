package role

import (
	"leapfor.xyz/espyna/internal/application/ports"
	rolepb "leapfor.xyz/esqyma/golang/v1/domain/entity/role"
)

// RoleRepositories groups all repository dependencies for role use cases
type RoleRepositories struct {
	Role rolepb.RoleDomainServiceServer // Primary entity repository
}

// RoleServices groups all business service dependencies for role use cases
type RoleServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all role-related use cases
type UseCases struct {
	CreateRole          *CreateRoleUseCase
	ReadRole            *ReadRoleUseCase
	UpdateRole          *UpdateRoleUseCase
	DeleteRole          *DeleteRoleUseCase
	ListRoles           *ListRolesUseCase
	GetRoleListPageData *GetRoleListPageDataUseCase
	GetRoleItemPageData *GetRoleItemPageDataUseCase
}

// NewUseCases creates a new collection of role use cases
func NewUseCases(
	repositories RoleRepositories,
	services RoleServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateRoleRepositories(repositories)
	createServices := CreateRoleServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadRoleRepositories(repositories)
	readServices := ReadRoleServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateRoleRepositories(repositories)
	updateServices := UpdateRoleServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteRoleRepositories(repositories)
	deleteServices := DeleteRoleServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListRolesRepositories(repositories)
	listServices := ListRolesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetRoleListPageDataRepositories(repositories)
	getListPageDataServices := GetRoleListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetRoleItemPageDataRepositories(repositories)
	getItemPageDataServices := GetRoleItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateRole:          NewCreateRoleUseCase(createRepos, createServices),
		ReadRole:            NewReadRoleUseCase(readRepos, readServices),
		UpdateRole:          NewUpdateRoleUseCase(updateRepos, updateServices),
		DeleteRole:          NewDeleteRoleUseCase(deleteRepos, deleteServices),
		ListRoles:           NewListRolesUseCase(listRepos, listServices),
		GetRoleListPageData: NewGetRoleListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetRoleItemPageData: NewGetRoleItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of role use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(roleRepo rolepb.RoleDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := RoleRepositories{
		Role: roleRepo,
	}

	services := RoleServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
