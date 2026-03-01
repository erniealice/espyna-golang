package expenditure

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

// ExpenditureRepositories groups all repository dependencies for expenditure use cases
type ExpenditureRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
}

// ExpenditureServices groups all business service dependencies for expenditure use cases
type ExpenditureServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all expenditure-related use cases
type UseCases struct {
	CreateExpenditure *CreateExpenditureUseCase
	ReadExpenditure   *ReadExpenditureUseCase
	UpdateExpenditure *UpdateExpenditureUseCase
	DeleteExpenditure *DeleteExpenditureUseCase
	ListExpenditures  *ListExpendituresUseCase
}

// NewUseCases creates a new collection of expenditure use cases
func NewUseCases(
	repositories ExpenditureRepositories,
	services ExpenditureServices,
) *UseCases {
	createRepos := CreateExpenditureRepositories(repositories)
	createServices := CreateExpenditureServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadExpenditureRepositories(repositories)
	readServices := ReadExpenditureServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateExpenditureRepositories(repositories)
	updateServices := UpdateExpenditureServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteExpenditureRepositories(repositories)
	deleteServices := DeleteExpenditureServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListExpendituresRepositories(repositories)
	listServices := ListExpendituresServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateExpenditure: NewCreateExpenditureUseCase(createRepos, createServices),
		ReadExpenditure:   NewReadExpenditureUseCase(readRepos, readServices),
		UpdateExpenditure: NewUpdateExpenditureUseCase(updateRepos, updateServices),
		DeleteExpenditure: NewDeleteExpenditureUseCase(deleteRepos, deleteServices),
		ListExpenditures:  NewListExpendituresUseCase(listRepos, listServices),
	}
}
