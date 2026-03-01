package expenditurecategory

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

// ExpenditureCategoryRepositories groups all repository dependencies
type ExpenditureCategoryRepositories struct {
	ExpenditureCategory pb.ExpenditureCategoryDomainServiceServer
}

// ExpenditureCategoryServices groups all business service dependencies
type ExpenditureCategoryServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all expenditure category use cases
type UseCases struct {
	CreateExpenditureCategory *CreateExpenditureCategoryUseCase
	ReadExpenditureCategory   *ReadExpenditureCategoryUseCase
	UpdateExpenditureCategory *UpdateExpenditureCategoryUseCase
	DeleteExpenditureCategory *DeleteExpenditureCategoryUseCase
	ListExpenditureCategories *ListExpenditureCategoriesUseCase
}

// NewUseCases creates a new collection of expenditure category use cases
func NewUseCases(
	repositories ExpenditureCategoryRepositories,
	services ExpenditureCategoryServices,
) *UseCases {
	createRepos := CreateExpenditureCategoryRepositories(repositories)
	createServices := CreateExpenditureCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadExpenditureCategoryRepositories(repositories)
	readServices := ReadExpenditureCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateExpenditureCategoryRepositories(repositories)
	updateServices := UpdateExpenditureCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteExpenditureCategoryRepositories(repositories)
	deleteServices := DeleteExpenditureCategoryServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListExpenditureCategoriesRepositories(repositories)
	listServices := ListExpenditureCategoriesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateExpenditureCategory: NewCreateExpenditureCategoryUseCase(createRepos, createServices),
		ReadExpenditureCategory:   NewReadExpenditureCategoryUseCase(readRepos, readServices),
		UpdateExpenditureCategory: NewUpdateExpenditureCategoryUseCase(updateRepos, updateServices),
		DeleteExpenditureCategory: NewDeleteExpenditureCategoryUseCase(deleteRepos, deleteServices),
		ListExpenditureCategories: NewListExpenditureCategoriesUseCase(listRepos, listServices),
	}
}
