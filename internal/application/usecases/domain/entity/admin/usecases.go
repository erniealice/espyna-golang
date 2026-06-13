package admin

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	adminpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/admin"
)

// AdminRepositories groups all repository dependencies for admin use cases
type AdminRepositories struct {
	Admin adminpb.AdminDomainServiceServer // Primary entity repository
}

// AdminServices groups all business service dependencies for admin use cases
type AdminServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all admin-related use cases
type UseCases struct {
	CreateAdmin          *CreateAdminUseCase
	ReadAdmin            *ReadAdminUseCase
	UpdateAdmin          *UpdateAdminUseCase
	DeleteAdmin          *DeleteAdminUseCase
	ListAdmins           *ListAdminsUseCase
	GetAdminListPageData *GetAdminListPageDataUseCase
	GetAdminItemPageData *GetAdminItemPageDataUseCase
}

// NewUseCases creates a new collection of admin use cases
func NewUseCases(
	repositories AdminRepositories,
	services AdminServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateAdminRepositories(repositories)
	createServices := CreateAdminServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadAdminRepositories(repositories)
	readServices := ReadAdminServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateAdminRepositories(repositories)
	updateServices := UpdateAdminServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteAdminRepositories(repositories)
	deleteServices := DeleteAdminServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListAdminsRepositories(repositories)
	listServices := ListAdminsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getListPageDataRepos := GetAdminListPageDataRepositories(repositories)
	getListPageDataServices := GetAdminListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getItemPageDataRepos := GetAdminItemPageDataRepositories(repositories)
	getItemPageDataServices := GetAdminItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateAdmin:          NewCreateAdminUseCase(createRepos, createServices),
		ReadAdmin:            NewReadAdminUseCase(readRepos, readServices),
		UpdateAdmin:          NewUpdateAdminUseCase(updateRepos, updateServices),
		DeleteAdmin:          NewDeleteAdminUseCase(deleteRepos, deleteServices),
		ListAdmins:           NewListAdminsUseCase(listRepos, listServices),
		GetAdminListPageData: NewGetAdminListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetAdminItemPageData: NewGetAdminItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of admin use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(adminRepo adminpb.AdminDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := AdminRepositories{
		Admin: adminRepo,
	}

	services := AdminServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator:       ports.NewNoOpTranslator(),
		ActionGatekeeper: actiongate.NewActionGatekeeper(nil, ports.NewNoOpTranslator()),
	}

	return NewUseCases(repositories, services)
}
