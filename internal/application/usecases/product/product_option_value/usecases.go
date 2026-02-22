package product_option_value

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	productoptionvaluepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_option_value"
)

// ProductOptionValueRepositories groups all repository dependencies for product option value use cases
type ProductOptionValueRepositories struct {
	ProductOptionValue productoptionvaluepb.ProductOptionValueDomainServiceServer // Primary entity repository
}

// ProductOptionValueServices groups all business service dependencies for product option value use cases
type ProductOptionValueServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductOptionValueRepositories(repositories)
	readServices := ReadProductOptionValueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductOptionValueRepositories(repositories)
	updateServices := UpdateProductOptionValueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductOptionValueRepositories(repositories)
	deleteServices := DeleteProductOptionValueServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductOptionValuesRepositories(repositories)
	listServices := ListProductOptionValuesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateProductOptionValue: NewCreateProductOptionValueUseCase(createRepos, createServices),
		ReadProductOptionValue:   NewReadProductOptionValueUseCase(readRepos, readServices),
		UpdateProductOptionValue: NewUpdateProductOptionValueUseCase(updateRepos, updateServices),
		DeleteProductOptionValue: NewDeleteProductOptionValueUseCase(deleteRepos, deleteServices),
		ListProductOptionValues:  NewListProductOptionValuesUseCase(listRepos, listServices),
	}
}
