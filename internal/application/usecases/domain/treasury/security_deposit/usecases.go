package securitydeposit

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

// SecurityDepositRepositories groups all repository dependencies for security deposit use cases
type SecurityDepositRepositories struct {
	SecurityDeposit securitydepositpb.SecurityDepositDomainServiceServer // Primary entity repository
}

// SecurityDepositServices groups all business service dependencies for security deposit use cases
type SecurityDepositServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	listRepos := ListSecurityDepositsRepositories(repositories)
	listServices := ListSecurityDepositsServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetSecurityDepositListPageDataRepositories(repositories)
	getListPageDataServices := GetSecurityDepositListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateSecurityDeposit:          NewCreateSecurityDepositUseCase(createRepos, createServices),
		ListSecurityDeposits:           NewListSecurityDepositsUseCase(listRepos, listServices),
		GetSecurityDepositListPageData: NewGetSecurityDepositListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
