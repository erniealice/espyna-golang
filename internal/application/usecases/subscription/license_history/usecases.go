package licensehistory

import (
	"leapfor.xyz/espyna/internal/application/ports"
	licensehistorypb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license_history"
)

// LicenseHistoryRepositories groups all repository dependencies for license history use cases
type LicenseHistoryRepositories struct {
	LicenseHistory licensehistorypb.LicenseHistoryDomainServiceServer // Primary entity repository
}

// LicenseHistoryServices groups all business service dependencies for license history use cases
type LicenseHistoryServices struct {
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
	IDService            ports.IDService            // UUID generation
}

// UseCases contains all license history-related use cases
type UseCases struct {
	CreateLicenseHistory          *CreateLicenseHistoryUseCase
	ReadLicenseHistory            *ReadLicenseHistoryUseCase
	ListLicenseHistory            *ListLicenseHistoryUseCase
	GetLicenseHistoryListPageData *GetLicenseHistoryListPageDataUseCase
}

// NewUseCases creates a new collection of license history use cases
func NewUseCases(
	repositories LicenseHistoryRepositories,
	services LicenseHistoryServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateLicenseHistoryRepositories(repositories)
	createServices := CreateLicenseHistoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadLicenseHistoryRepositories(repositories)
	readServices := ReadLicenseHistoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListLicenseHistoryRepositories(repositories)
	listServices := ListLicenseHistoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetLicenseHistoryListPageDataRepositories(repositories)
	getListPageDataServices := GetLicenseHistoryListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateLicenseHistory:          NewCreateLicenseHistoryUseCase(createRepos, createServices),
		ReadLicenseHistory:            NewReadLicenseHistoryUseCase(readRepos, readServices),
		ListLicenseHistory:            NewListLicenseHistoryUseCase(listRepos, listServices),
		GetLicenseHistoryListPageData: NewGetLicenseHistoryListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
