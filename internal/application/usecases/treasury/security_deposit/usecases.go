package securitydeposit

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

// SecurityDepositRepositories groups all repository dependencies for security deposit use cases
type SecurityDepositRepositories struct {
	SecurityDeposit securitydepositpb.SecurityDepositDomainServiceServer // Primary entity repository
}

// SecurityDepositServices groups all business service dependencies for security deposit use cases
type SecurityDepositServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all security deposit-related use cases
type UseCases struct {
	CreateSecurityDeposit          *CreateSecurityDepositUseCase
	ListSecurityDeposits           *ListSecurityDepositsUseCase
	GetSecurityDepositListPageData *GetSecurityDepositListPageDataUseCase
}

// NewUseCases creates a new collection of security deposit use cases
func NewUseCases(
	repositories SecurityDepositRepositories,
	services SecurityDepositServices,
) *UseCases {
	createRepos := CreateSecurityDepositRepositories(repositories)
	createServices := CreateSecurityDepositServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	listRepos := ListSecurityDepositsRepositories(repositories)
	listServices := ListSecurityDepositsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetSecurityDepositListPageDataRepositories(repositories)
	getListPageDataServices := GetSecurityDepositListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateSecurityDeposit:          NewCreateSecurityDepositUseCase(createRepos, createServices),
		ListSecurityDeposits:           NewListSecurityDepositsUseCase(listRepos, listServices),
		GetSecurityDepositListPageData: NewGetSecurityDepositListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
