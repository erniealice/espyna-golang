package expenditurelineitem

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

// ExpenditureLineItemRepositories groups all repository dependencies
type ExpenditureLineItemRepositories struct {
	ExpenditureLineItem pb.ExpenditureLineItemDomainServiceServer
}

// ExpenditureLineItemServices groups all business service dependencies
type ExpenditureLineItemServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// UseCases contains all expenditure line item use cases
type UseCases struct {
	CreateExpenditureLineItem *CreateExpenditureLineItemUseCase
	ReadExpenditureLineItem   *ReadExpenditureLineItemUseCase
	UpdateExpenditureLineItem *UpdateExpenditureLineItemUseCase
	DeleteExpenditureLineItem *DeleteExpenditureLineItemUseCase
	ListExpenditureLineItems  *ListExpenditureLineItemsUseCase
}

// NewUseCases creates a new collection of expenditure line item use cases
func NewUseCases(
	repositories ExpenditureLineItemRepositories,
	services ExpenditureLineItemServices,
) *UseCases {
	createRepos := CreateExpenditureLineItemRepositories(repositories)
	createServices := CreateExpenditureLineItemServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadExpenditureLineItemRepositories(repositories)
	readServices := ReadExpenditureLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateExpenditureLineItemRepositories(repositories)
	updateServices := UpdateExpenditureLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteExpenditureLineItemRepositories(repositories)
	deleteServices := DeleteExpenditureLineItemServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListExpenditureLineItemsRepositories(repositories)
	listServices := ListExpenditureLineItemsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateExpenditureLineItem: NewCreateExpenditureLineItemUseCase(createRepos, createServices),
		ReadExpenditureLineItem:   NewReadExpenditureLineItemUseCase(readRepos, readServices),
		UpdateExpenditureLineItem: NewUpdateExpenditureLineItemUseCase(updateRepos, updateServices),
		DeleteExpenditureLineItem: NewDeleteExpenditureLineItemUseCase(deleteRepos, deleteServices),
		ListExpenditureLineItems:  NewListExpenditureLineItemsUseCase(listRepos, listServices),
	}
}
