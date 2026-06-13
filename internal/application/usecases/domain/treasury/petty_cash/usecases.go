package pettycash

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
)

// PettyCashRepositories groups all repository dependencies for petty cash use cases
type PettyCashRepositories struct {
	PettyCashFund pettycashfundpb.PettyCashFundDomainServiceServer // Primary entity repository
}

// PettyCashServices groups all business service dependencies for petty cash use cases
type PettyCashServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	listRepos := ListPettyCashFundsRepositories(repositories)
	listServices := ListPettyCashFundsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetPettyCashFundListPageDataRepositories(repositories)
	getListPageDataServices := GetPettyCashFundListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreatePettyCashFund:          NewCreatePettyCashFundUseCase(createRepos, createServices),
		ListPettyCashFunds:           NewListPettyCashFundsUseCase(listRepos, listServices),
		GetPettyCashFundListPageData: NewGetPettyCashFundListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
