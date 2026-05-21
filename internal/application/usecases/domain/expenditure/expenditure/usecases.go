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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadExpenditureRepositories{
		Expenditure: repositories.Expenditure,
	}
	readServices := ReadExpenditureServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateExpenditureRepositories{
		Expenditure: repositories.Expenditure,
	}
	updateServices := UpdateExpenditureServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteExpenditureRepositories{
		Expenditure: repositories.Expenditure,
	}
	deleteServices := DeleteExpenditureServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListExpendituresRepositories{
		Expenditure: repositories.Expenditure,
	}
	listServices := ListExpendituresServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateExpenditure: NewCreateExpenditureUseCase(createRepos, createServices),
		ReadExpenditure:   NewReadExpenditureUseCase(readRepos, readServices),
		UpdateExpenditure: NewUpdateExpenditureUseCase(updateRepos, updateServices),
		DeleteExpenditure: NewDeleteExpenditureUseCase(deleteRepos, deleteServices),
		ListExpenditures:  NewListExpendituresUseCase(listRepos, listServices),
	}
}
