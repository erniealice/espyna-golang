package fiscalperiod

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
)

// FiscalPeriodRepositories groups all repository dependencies for fiscal period use cases
type FiscalPeriodRepositories struct {
	FiscalPeriod fiscalperiodpb.FiscalPeriodDomainServiceServer // Primary entity repository
}

// FiscalPeriodServices groups all business service dependencies for fiscal period use cases
type FiscalPeriodServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all fiscal-period-related use cases
type UseCases struct {
	CreateFiscalPeriod          *CreateFiscalPeriodUseCase
	ReadFiscalPeriod            *ReadFiscalPeriodUseCase
	ListFiscalPeriods           *ListFiscalPeriodsUseCase
	GetFiscalPeriodListPageData *GetFiscalPeriodListPageDataUseCase
	CloseFiscalPeriod           *CloseFiscalPeriodUseCase
}

// NewUseCases creates a new collection of fiscal period use cases
func NewUseCases(
	repositories FiscalPeriodRepositories,
	services FiscalPeriodServices,
) *UseCases {
	createRepos := CreateFiscalPeriodRepositories(repositories)
	createServices := CreateFiscalPeriodServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadFiscalPeriodRepositories(repositories)
	readServices := ReadFiscalPeriodServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListFiscalPeriodsRepositories(repositories)
	listServices := ListFiscalPeriodsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetFiscalPeriodListPageDataRepositories(repositories)
	getListPageDataServices := GetFiscalPeriodListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	closeRepos := CloseFiscalPeriodRepositories(repositories)
	closeServices := CloseFiscalPeriodServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateFiscalPeriod:          NewCreateFiscalPeriodUseCase(createRepos, createServices),
		ReadFiscalPeriod:            NewReadFiscalPeriodUseCase(readRepos, readServices),
		ListFiscalPeriods:           NewListFiscalPeriodsUseCase(listRepos, listServices),
		GetFiscalPeriodListPageData: NewGetFiscalPeriodListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		CloseFiscalPeriod:           NewCloseFiscalPeriodUseCase(closeRepos, closeServices),
	}
}
