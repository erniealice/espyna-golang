package expenditureattribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

// ExpenditureAttributeRepositories groups all repository dependencies
type ExpenditureAttributeRepositories struct {
	ExpenditureAttribute pb.ExpenditureAttributeDomainServiceServer
}

// ExpenditureAttributeServices groups all business service dependencies
type ExpenditureAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all expenditure attribute use cases
type UseCases struct {
	CreateExpenditureAttribute *CreateExpenditureAttributeUseCase
	ReadExpenditureAttribute   *ReadExpenditureAttributeUseCase
	UpdateExpenditureAttribute *UpdateExpenditureAttributeUseCase
	DeleteExpenditureAttribute *DeleteExpenditureAttributeUseCase
	ListExpenditureAttributes  *ListExpenditureAttributesUseCase
}

// NewUseCases creates a new collection of expenditure attribute use cases
func NewUseCases(
	repositories ExpenditureAttributeRepositories,
	services ExpenditureAttributeServices,
) *UseCases {
	createRepos := CreateExpenditureAttributeRepositories(repositories)
	createServices := CreateExpenditureAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadExpenditureAttributeRepositories(repositories)
	readServices := ReadExpenditureAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateExpenditureAttributeRepositories(repositories)
	updateServices := UpdateExpenditureAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteExpenditureAttributeRepositories(repositories)
	deleteServices := DeleteExpenditureAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListExpenditureAttributesRepositories(repositories)
	listServices := ListExpenditureAttributesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateExpenditureAttribute: NewCreateExpenditureAttributeUseCase(createRepos, createServices),
		ReadExpenditureAttribute:   NewReadExpenditureAttributeUseCase(readRepos, readServices),
		UpdateExpenditureAttribute: NewUpdateExpenditureAttributeUseCase(updateRepos, updateServices),
		DeleteExpenditureAttribute: NewDeleteExpenditureAttributeUseCase(deleteRepos, deleteServices),
		ListExpenditureAttributes:  NewListExpenditureAttributesUseCase(listRepos, listServices),
	}
}
