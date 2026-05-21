package category

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	categorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// CategoryRepositories groups all repository dependencies for category use cases
type CategoryRepositories struct {
	Category categorypb.CategoryDomainServiceServer // Primary entity repository
}

// CategoryServices groups all business service dependencies for category use cases
type CategoryServices struct {
	TransactionService ports.TransactionService
	TranslationService ports.TranslationService
	IDService          ports.IDService
}

// UseCases contains all category-related use cases
type UseCases struct {
	CreateCategory *CreateCategoryUseCase
	ReadCategory   *ReadCategoryUseCase
	UpdateCategory *UpdateCategoryUseCase
	DeleteCategory *DeleteCategoryUseCase
	ListCategories *ListCategoriesUseCase
}

// NewUseCases creates a new collection of category use cases
func NewUseCases(
	repositories CategoryRepositories,
	services CategoryServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateCategoryRepositories(repositories)
	createServices := CreateCategoryServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
		IDService:          services.IDService,
	}

	readRepos := ReadCategoryRepositories(repositories)
	readServices := ReadCategoryServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateCategoryRepositories(repositories)
	updateServices := UpdateCategoryServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteCategoryRepositories(repositories)
	deleteServices := DeleteCategoryServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListCategoriesRepositories(repositories)
	listServices := ListCategoriesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateCategory: NewCreateCategoryUseCase(createRepos, createServices),
		ReadCategory:   NewReadCategoryUseCase(readRepos, readServices),
		UpdateCategory: NewUpdateCategoryUseCase(updateRepos, updateServices),
		DeleteCategory: NewDeleteCategoryUseCase(deleteRepos, deleteServices),
		ListCategories: NewListCategoriesUseCase(listRepos, listServices),
	}
}

// NewUseCasesUngrouped creates a new collection of category use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(categoryRepo categorypb.CategoryDomainServiceServer) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := CategoryRepositories{
		Category: categoryRepo,
	}

	services := CategoryServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
