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
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
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
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadExpenditureCategoryRepositories(repositories)
	readServices := ReadExpenditureCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateExpenditureCategoryRepositories(repositories)
	updateServices := UpdateExpenditureCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteExpenditureCategoryRepositories(repositories)
	deleteServices := DeleteExpenditureCategoryServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListExpenditureCategoriesRepositories(repositories)
	listServices := ListExpenditureCategoriesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateExpenditureCategory: NewCreateExpenditureCategoryUseCase(createRepos, createServices),
		ReadExpenditureCategory:   NewReadExpenditureCategoryUseCase(readRepos, readServices),
		UpdateExpenditureCategory: NewUpdateExpenditureCategoryUseCase(updateRepos, updateServices),
		DeleteExpenditureCategory: NewDeleteExpenditureCategoryUseCase(deleteRepos, deleteServices),
		ListExpenditureCategories: NewListExpenditureCategoriesUseCase(listRepos, listServices),
	}
}
