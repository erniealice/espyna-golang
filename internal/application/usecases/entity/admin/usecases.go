package admin

import (
	"leapfor.xyz/espyna/internal/application/ports"
	adminpb "leapfor.xyz/esqyma/golang/v1/domain/entity/admin"
)

// AdminRepositories groups all repository dependencies for admin use cases
type AdminRepositories struct {
	Admin adminpb.AdminDomainServiceServer // Primary entity repository
}

// AdminServices groups all business service dependencies for admin use cases
type AdminServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadAdminRepositories(repositories)
	readServices := ReadAdminServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateAdminRepositories(repositories)
	updateServices := UpdateAdminServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteAdminRepositories(repositories)
	deleteServices := DeleteAdminServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListAdminsRepositories(repositories)
	listServices := ListAdminsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetAdminListPageDataRepositories(repositories)
	getListPageDataServices := GetAdminListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getItemPageDataRepos := GetAdminItemPageDataRepositories(repositories)
	getItemPageDataServices := GetAdminItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
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
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
