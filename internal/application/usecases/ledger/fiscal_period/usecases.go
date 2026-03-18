package fiscalperiod

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	fiscalperiodpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/fiscal_period"
)

// FiscalPeriodRepositories groups all repository dependencies for fiscal period use cases
type FiscalPeriodRepositories struct {
	FiscalPeriod fiscalperiodpb.FiscalPeriodDomainServiceServer // Primary entity repository
}

// FiscalPeriodServices groups all business service dependencies for fiscal period use cases
type FiscalPeriodServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadFiscalPeriodRepositories(repositories)
	readServices := ReadFiscalPeriodServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListFiscalPeriodsRepositories(repositories)
	listServices := ListFiscalPeriodsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetFiscalPeriodListPageDataRepositories(repositories)
	getListPageDataServices := GetFiscalPeriodListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	closeRepos := CloseFiscalPeriodRepositories(repositories)
	closeServices := CloseFiscalPeriodServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateFiscalPeriod:          NewCreateFiscalPeriodUseCase(createRepos, createServices),
		ReadFiscalPeriod:            NewReadFiscalPeriodUseCase(readRepos, readServices),
		ListFiscalPeriods:           NewListFiscalPeriodsUseCase(listRepos, listServices),
		GetFiscalPeriodListPageData: NewGetFiscalPeriodListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		CloseFiscalPeriod:           NewCloseFiscalPeriodUseCase(closeRepos, closeServices),
	}
}
