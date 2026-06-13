package product_option_value

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	productoptionvaluepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option_value"
)

// ProductOptionValueRepositories groups all repository dependencies for product option value use cases
type ProductOptionValueRepositories struct {
	ProductOptionValue productoptionvaluepb.ProductOptionValueDomainServiceServer // Primary entity repository
}

// ProductOptionValueServices groups all business service dependencies for product option value use cases
type ProductOptionValueServices struct {
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all product option value-related use cases
type UseCases struct {
	CreateProductOptionValue *CreateProductOptionValueUseCase
	ReadProductOptionValue   *ReadProductOptionValueUseCase
	UpdateProductOptionValue *UpdateProductOptionValueUseCase
	DeleteProductOptionValue *DeleteProductOptionValueUseCase
	ListProductOptionValues  *ListProductOptionValuesUseCase
}

// NewUseCases creates a new collection of product option value use cases
func NewUseCases(
	repositories ProductOptionValueRepositories,
	services ProductOptionValueServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductOptionValueRepositories(repositories)
	createServices := CreateProductOptionValueServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadProductOptionValueRepositories(repositories)
	readServices := ReadProductOptionValueServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdateProductOptionValueRepositories(repositories)
	updateServices := UpdateProductOptionValueServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeleteProductOptionValueRepositories(repositories)
	deleteServices := DeleteProductOptionValueServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListProductOptionValuesRepositories(repositories)
	listServices := ListProductOptionValuesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	return &UseCases{
		CreateProductOptionValue: NewCreateProductOptionValueUseCase(createRepos, createServices),
		ReadProductOptionValue:   NewReadProductOptionValueUseCase(readRepos, readServices),
		UpdateProductOptionValue: NewUpdateProductOptionValueUseCase(updateRepos, updateServices),
		DeleteProductOptionValue: NewDeleteProductOptionValueUseCase(deleteRepos, deleteServices),
		ListProductOptionValues:  NewListProductOptionValuesUseCase(listRepos, listServices),
	}
}
