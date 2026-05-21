package expenditure

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	paymenttermpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/payment_term"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
)

// ExpenditureRepositories groups all repository dependencies for expenditure use cases
type ExpenditureRepositories struct {
	Expenditure expenditurepb.ExpenditureDomainServiceServer
	PaymentTerm paymenttermpb.PaymentTermDomainServiceServer
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
	createRepos := CreateExpenditureRepositories{
		Expenditure: repositories.Expenditure,
		PaymentTerm: repositories.PaymentTerm,
	}
	createServices := CreateExpenditureServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadExpenditureRepositories{
		Expenditure: repositories.Expenditure,
	}
	readServices := ReadExpenditureServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateExpenditureRepositories{
		Expenditure: repositories.Expenditure,
	}
	updateServices := UpdateExpenditureServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteExpenditureRepositories{
		Expenditure: repositories.Expenditure,
	}
	deleteServices := DeleteExpenditureServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListExpendituresRepositories{
		Expenditure: repositories.Expenditure,
	}
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
