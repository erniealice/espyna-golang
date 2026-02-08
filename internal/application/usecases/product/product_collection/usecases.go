package product_collection

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	productcollectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_collection"
)

// ProductCollectionRepositories groups all repository dependencies for product collection use cases
type ProductCollectionRepositories struct {
	ProductCollection productcollectionpb.ProductCollectionDomainServiceServer // Primary entity repository
	Product           productpb.ProductDomainServiceServer
	Collection        collectionpb.CollectionDomainServiceServer
}

// ProductCollectionServices groups all business service dependencies for product collection use cases
type ProductCollectionServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all product collection-related use cases
type UseCases struct {
	CreateProductCollection          *CreateProductCollectionUseCase
	ReadProductCollection            *ReadProductCollectionUseCase
	UpdateProductCollection          *UpdateProductCollectionUseCase
	DeleteProductCollection          *DeleteProductCollectionUseCase
	ListProductCollections           *ListProductCollectionsUseCase
	GetProductCollectionListPageData *GetProductCollectionListPageDataUseCase
	GetProductCollectionItemPageData *GetProductCollectionItemPageDataUseCase
}

// NewUseCases creates a new collection of product collection use cases
func NewUseCases(
	repositories ProductCollectionRepositories,
	services ProductCollectionServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateProductCollectionRepositories{
		ProductCollection: repositories.ProductCollection,
		Product:           repositories.Product,
		Collection:        repositories.Collection,
	}
	createServices := CreateProductCollectionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductCollectionRepositories{
		ProductCollection: repositories.ProductCollection,
	}
	readServices := ReadProductCollectionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductCollectionRepositories{
		ProductCollection: repositories.ProductCollection,
		Product:           repositories.Product,
		Collection:        repositories.Collection,
	}
	updateServices := UpdateProductCollectionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductCollectionRepositories{
		ProductCollection: repositories.ProductCollection,
	}
	deleteServices := DeleteProductCollectionServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductCollectionsRepositories{
		ProductCollection: repositories.ProductCollection,
	}
	listServices := ListProductCollectionsServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetProductCollectionListPageDataRepositories{
		ProductCollection: repositories.ProductCollection,
	}
	listPageDataServices := GetProductCollectionListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetProductCollectionItemPageDataRepositories{
		ProductCollection: repositories.ProductCollection,
	}
	itemPageDataServices := GetProductCollectionItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateProductCollection:          NewCreateProductCollectionUseCase(createRepos, createServices),
		ReadProductCollection:            NewReadProductCollectionUseCase(readRepos, readServices),
		UpdateProductCollection:          NewUpdateProductCollectionUseCase(updateRepos, updateServices),
		DeleteProductCollection:          NewDeleteProductCollectionUseCase(deleteRepos, deleteServices),
		ListProductCollections:           NewListProductCollectionsUseCase(listRepos, listServices),
		GetProductCollectionListPageData: NewGetProductCollectionListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetProductCollectionItemPageData: NewGetProductCollectionItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
