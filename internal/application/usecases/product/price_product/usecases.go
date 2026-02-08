package price_product

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
)

// UseCases contains all price product-related use cases
type UseCases struct {
	CreatePriceProduct          *CreatePriceProductUseCase
	ReadPriceProduct            *ReadPriceProductUseCase
	UpdatePriceProduct          *UpdatePriceProductUseCase
	DeletePriceProduct          *DeletePriceProductUseCase
	ListPriceProducts           *ListPriceProductsUseCase
	GetPriceProductListPageData *GetPriceProductListPageDataUseCase
	GetPriceProductItemPageData *GetPriceProductItemPageDataUseCase
}

// PriceProductRepositories groups all repository dependencies for price product use cases
type PriceProductRepositories struct {
	PriceProduct priceproductpb.PriceProductDomainServiceServer // Primary entity repository
	Product      productpb.ProductDomainServiceServer           // Entity reference validation
}

// PriceProductServices groups all business service dependencies for price product use cases
type PriceProductServices struct {
	AuthorizationService ports.AuthorizationService // Current: RBAC and permissions
	TransactionService   ports.TransactionService   // Current: Database transactions
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of price product use cases
func NewUseCases(
	repositories PriceProductRepositories,
	services PriceProductServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreatePriceProductRepositories{
		PriceProduct: repositories.PriceProduct,
		Product:      repositories.Product,
	}
	createServices := CreatePriceProductServices{
		TransactionService:   services.TransactionService,
		AuthorizationService: services.AuthorizationService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadPriceProductRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	readServices := ReadPriceProductServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdatePriceProductRepositories{
		PriceProduct: repositories.PriceProduct,
		Product:      repositories.Product,
	}
	updateServices := UpdatePriceProductServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeletePriceProductRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	deleteServices := DeletePriceProductServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListPriceProductsRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	listServices := ListPriceProductsServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listPageDataRepos := GetPriceProductListPageDataRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	listPageDataServices := GetPriceProductListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	itemPageDataRepos := GetPriceProductItemPageDataRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	itemPageDataServices := GetPriceProductItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreatePriceProduct:          NewCreatePriceProductUseCase(createRepos, createServices),
		ReadPriceProduct:            NewReadPriceProductUseCase(readRepos, readServices),
		UpdatePriceProduct:          NewUpdatePriceProductUseCase(updateRepos, updateServices),
		DeletePriceProduct:          NewDeletePriceProductUseCase(deleteRepos, deleteServices),
		ListPriceProducts:           NewListPriceProductsUseCase(listRepos, listServices),
		GetPriceProductListPageData: NewGetPriceProductListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetPriceProductItemPageData: NewGetPriceProductItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
