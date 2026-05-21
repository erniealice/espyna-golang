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
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadCategoryRepositories(repositories)
	readServices := ReadCategoryServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateCategoryRepositories(repositories)
	updateServices := UpdateCategoryServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteCategoryRepositories(repositories)
	deleteServices := DeleteCategoryServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListCategoriesRepositories(repositories)
	listServices := ListCategoriesServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
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
		Transactor: ports.NewNoOpTransactor(),
		Translator: ports.NewNoOpTranslator(),
	}

	return NewUseCases(repositories, services)
}
