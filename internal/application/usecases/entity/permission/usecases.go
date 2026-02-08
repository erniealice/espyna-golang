package permission

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// PermissionRepositories groups all repository dependencies for permission use cases
type PermissionRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// PermissionServices groups all business service dependencies for permission use cases
type PermissionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all permission-related use cases
type UseCases struct {
	CreatePermission          *CreatePermissionUseCase
	ReadPermission            *ReadPermissionUseCase
	UpdatePermission          *UpdatePermissionUseCase
	DeletePermission          *DeletePermissionUseCase
	ListPermissions           *ListPermissionsUseCase
	GetPermissionListPageData *GetPermissionListPageDataUseCase
	GetPermissionItemPageData *GetPermissionItemPageDataUseCase
}

// NewUseCases creates a new collection of permission use cases
func NewUseCases(
	repositories PermissionRepositories,
	services PermissionServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePermissionRepositories(repositories)
	createServices := CreatePermissionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadPermissionRepositories(repositories)
	readServices := ReadPermissionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdatePermissionRepositories(repositories)
	updateServices := UpdatePermissionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeletePermissionRepositories(repositories)
	deleteServices := DeletePermissionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPermissionsRepositories(repositories)
	listServices := ListPermissionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetPermissionListPageDataRepositories(repositories)
	getListPageDataServices := GetPermissionListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetPermissionItemPageDataRepositories(repositories)
	getItemPageDataServices := GetPermissionItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePermission:          NewCreatePermissionUseCase(createRepos, createServices),
		ReadPermission:            NewReadPermissionUseCase(readRepos, readServices),
		UpdatePermission:          NewUpdatePermissionUseCase(updateRepos, updateServices),
		DeletePermission:          NewDeletePermissionUseCase(deleteRepos, deleteServices),
		ListPermissions:           NewListPermissionsUseCase(listRepos, listServices),
		GetPermissionListPageData: NewGetPermissionListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetPermissionItemPageData: NewGetPermissionItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of permission use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(permissionRepo permissionpb.PermissionDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := PermissionRepositories{
		Permission: permissionRepo,
	}

	services := PermissionServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
