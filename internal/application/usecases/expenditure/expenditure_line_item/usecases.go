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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadExpenditureLineItemRepositories(repositories)
	readServices := ReadExpenditureLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateExpenditureLineItemRepositories(repositories)
	updateServices := UpdateExpenditureLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteExpenditureLineItemRepositories(repositories)
	deleteServices := DeleteExpenditureLineItemServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListExpenditureLineItemsRepositories(repositories)
	listServices := ListExpenditureLineItemsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateExpenditureLineItem: NewCreateExpenditureLineItemUseCase(createRepos, createServices),
		ReadExpenditureLineItem:   NewReadExpenditureLineItemUseCase(readRepos, readServices),
		UpdateExpenditureLineItem: NewUpdateExpenditureLineItemUseCase(updateRepos, updateServices),
		DeleteExpenditureLineItem: NewDeleteExpenditureLineItemUseCase(deleteRepos, deleteServices),
		ListExpenditureLineItems:  NewListExpenditureLineItemsUseCase(listRepos, listServices),
	}
}
