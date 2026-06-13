package staff

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
)

// StaffRepositories groups all repository dependencies for staff use cases
type StaffRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// StaffServices groups all business service dependencies for staff use cases
type StaffServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all staff-related use cases
type UseCases struct {
	CreateStaff          *CreateStaffUseCase
	ReadStaff            *ReadStaffUseCase
	UpdateStaff          *UpdateStaffUseCase
	DeleteStaff          *DeleteStaffUseCase
	ListStaffs           *ListStaffsUseCase
	GetStaffListPageData *GetStaffListPageDataUseCase
	GetStaffItemPageData *GetStaffItemPageDataUseCase
}

// NewUseCases creates a new collection of staff use cases
func NewUseCases(
	repositories StaffRepositories,
	services StaffServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateStaffRepositories(repositories)
	createServices := CreateStaffServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadStaffRepositories(repositories)
	readServices := ReadStaffServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateStaffRepositories(repositories)
	updateServices := UpdateStaffServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteStaffRepositories(repositories)
	deleteServices := DeleteStaffServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListStaffsRepositories(repositories)
	listServices := ListStaffsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getStaffListPageDataRepos := GetStaffListPageDataRepositories(repositories)
	getStaffListPageDataServices := GetStaffListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	getStaffItemPageDataRepos := GetStaffItemPageDataRepositories(repositories)
	getStaffItemPageDataServices := GetStaffItemPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateStaff:          NewCreateStaffUseCase(createRepos, createServices),
		ReadStaff:            NewReadStaffUseCase(readRepos, readServices),
		UpdateStaff:          NewUpdateStaffUseCase(updateRepos, updateServices),
		DeleteStaff:          NewDeleteStaffUseCase(deleteRepos, deleteServices),
		ListStaffs:           NewListStaffsUseCase(listRepos, listServices),
		GetStaffListPageData: NewGetStaffListPageDataUseCase(getStaffListPageDataRepos, getStaffListPageDataServices),
		GetStaffItemPageData: NewGetStaffItemPageDataUseCase(getStaffItemPageDataRepos, getStaffItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of staff use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(staffRepo staffpb.StaffDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := StaffRepositories{
		Staff: staffRepo,
	}

	services := StaffServices{
		Authorizer: nil,
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
