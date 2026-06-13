package permission

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	permissionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/permission"
)

// PermissionRepositories groups all repository dependencies for permission use cases
type PermissionRepositories struct {
	Permission permissionpb.PermissionDomainServiceServer // Primary entity repository
}

// PermissionServices groups all business service dependencies for permission use cases
type PermissionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPermissionRepositories(repositories)
	readServices := ReadPermissionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdatePermissionRepositories(repositories)
	updateServices := UpdatePermissionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeletePermissionRepositories(repositories)
	deleteServices := DeletePermissionServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListPermissionsRepositories(repositories)
	listServices := ListPermissionsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetPermissionListPageDataRepositories(repositories)
	getListPageDataServices := GetPermissionListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetPermissionItemPageDataRepositories(repositories)
	getItemPageDataServices := GetPermissionItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
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
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
