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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPrepaymentRepositories(repositories)
	readServices := ReadPrepaymentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListPrepaymentsRepositories(repositories)
	listServices := ListPrepaymentsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	getListPageDataRepos := GetPrepaymentListPageDataRepositories(repositories)
	getListPageDataServices := GetPrepaymentListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreatePrepayment:          NewCreatePrepaymentUseCase(createRepos, createServices),
		ReadPrepayment:            NewReadPrepaymentUseCase(readRepos, readServices),
		ListPrepayments:           NewListPrepaymentsUseCase(listRepos, listServices),
		GetPrepaymentListPageData: NewGetPrepaymentListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
	}
}
