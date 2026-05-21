package prepayment

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
)

// PrepaymentRepositories groups all repository dependencies for prepayment use cases
type PrepaymentRepositories struct {
	Prepayment prepaymentpb.PrepaymentDomainServiceServer // Primary entity repository
}

// PrepaymentServices groups all business service dependencies for prepayment use cases
type PrepaymentServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all prepayment-related use cases
type UseCases struct {
	CreatePrepayment          *CreatePrepaymentUseCase
	ReadPrepayment            *ReadPrepaymentUseCase
	ListPrepayments           *ListPrepaymentsUseCase
	GetPrepaymentListPageData *GetPrepaymentListPageDataUseCase
}

// NewUseCases creates a new collection of prepayment use cases
func NewUseCases(
	repositories PrepaymentRepositories,
	services PrepaymentServices,
) *UseCases {
	createRepos := CreatePrepaymentRepositories(repositories)
	createServices := CreatePrepaymentServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPrepaymentRepositories(repositories)
	readServices := ReadPrepaymentServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPrepaymentsRepositories(repositories)
	listServices := ListPrepaymentsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	getListPageDataRepos := GetPrepaymentListPageDataRepositories(repositories)
	getListPageDataServices := GetPrepaymentListPageDataServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreatePrepayment:          NewCreatePrepaymentUseCase(createRepos, createServices),
		ReadPrepayment:            NewReadPrepaymentUseCase(readRepos, readServices),
		ListPrepayments:           NewListPrepaymentsUseCase(listRepos, listServices),
		GetPrepaymentListPageData: NewGetPrepaymentListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
