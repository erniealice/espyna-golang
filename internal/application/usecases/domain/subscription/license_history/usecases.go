package licensehistory

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
)

// LicenseHistoryRepositories groups all repository dependencies for license history use cases
type LicenseHistoryRepositories struct {
	LicenseHistory licensehistorypb.LicenseHistoryDomainServiceServer // Primary entity repository
}

// LicenseHistoryServices groups all business service dependencies for license history use cases
type LicenseHistoryServices struct {
	Authorizer  ports.Authorizer  // RBAC and permissions
	Transactor  ports.Transactor  // Database transactions
	Translator  ports.Translator  // i18n error messages
	IDGenerator ports.IDGenerator // UUID generation
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadLicenseHistoryRepositories(repositories)
	readServices := ReadLicenseHistoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListLicenseHistoryRepositories(repositories)
	listServices := ListLicenseHistoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetLicenseHistoryListPageDataRepositories(repositories)
	getListPageDataServices := GetLicenseHistoryListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateLicenseHistory:          NewCreateLicenseHistoryUseCase(createRepos, createServices),
		ReadLicenseHistory:            NewReadLicenseHistoryUseCase(readRepos, readServices),
		ListLicenseHistory:            NewListLicenseHistoryUseCase(listRepos, listServices),
		GetLicenseHistoryListPageData: NewGetLicenseHistoryListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
