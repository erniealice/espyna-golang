package pettycash

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
)

// PettyCashRepositories groups all repository dependencies for petty cash use cases
type PettyCashRepositories struct {
	PettyCashFund pettycashfundpb.PettyCashFundDomainServiceServer // Primary entity repository
}

// PettyCashServices groups all business service dependencies for petty cash use cases
type PettyCashServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all petty cash-related use cases
type UseCases struct {
	CreatePettyCashFund          *CreatePettyCashFundUseCase
	ListPettyCashFunds           *ListPettyCashFundsUseCase
	GetPettyCashFundListPageData *GetPettyCashFundListPageDataUseCase
}

// NewUseCases creates a new collection of petty cash use cases
func NewUseCases(
	repositories PettyCashRepositories,
	services PettyCashServices,
) *UseCases {
	createRepos := CreatePettyCashFundRepositories(repositories)
	createServices := CreatePettyCashFundServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	listRepos := ListPettyCashFundsRepositories(repositories)
	listServices := ListPettyCashFundsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetPettyCashFundListPageDataRepositories(repositories)
	getListPageDataServices := GetPettyCashFundListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePettyCashFund:          NewCreatePettyCashFundUseCase(createRepos, createServices),
		ListPettyCashFunds:           NewListPettyCashFundsUseCase(listRepos, listServices),
		GetPettyCashFundListPageData: NewGetPettyCashFundListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
