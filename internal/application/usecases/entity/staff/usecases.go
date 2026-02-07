package staff

import (
	"leapfor.xyz/espyna/internal/application/ports"
	staffpb "leapfor.xyz/esqyma/golang/v1/domain/entity/staff"
)

// StaffRepositories groups all repository dependencies for staff use cases
type StaffRepositories struct {
	Staff staffpb.StaffDomainServiceServer // Primary entity repository
}

// StaffServices groups all business service dependencies for staff use cases
type StaffServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadStaffRepositories(repositories)
	readServices := ReadStaffServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateStaffRepositories(repositories)
	updateServices := UpdateStaffServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteStaffRepositories(repositories)
	deleteServices := DeleteStaffServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListStaffsRepositories(repositories)
	listServices := ListStaffsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getStaffListPageDataRepos := GetStaffListPageDataRepositories(repositories)
	getStaffListPageDataServices := GetStaffListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getStaffItemPageDataRepos := GetStaffItemPageDataRepositories(repositories)
	getStaffItemPageDataServices := GetStaffItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
