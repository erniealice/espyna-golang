package price_product

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
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
	Authorizer  ports.Authorizer // Current: RBAC and permissions
	Transactor  ports.Transactor // Current: Database transactions
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
		Transactor:  services.Transactor,
		Authorizer:  services.Authorizer,
		Translator:  services.Translator,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadPriceProductRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	readServices := ReadPriceProductServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	updateRepos := UpdatePriceProductRepositories{
		PriceProduct: repositories.PriceProduct,
		Product:      repositories.Product,
	}
	updateServices := UpdatePriceProductServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	deleteRepos := DeletePriceProductRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	deleteServices := DeletePriceProductServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listRepos := ListPriceProductsRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	listServices := ListPriceProductsServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	listPageDataRepos := GetPriceProductListPageDataRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	listPageDataServices := GetPriceProductListPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
	}

	itemPageDataRepos := GetPriceProductItemPageDataRepositories{
		PriceProduct: repositories.PriceProduct,
	}
	itemPageDataServices := GetPriceProductItemPageDataServices{
		Transactor: services.Transactor,
		Translator: services.Translator,
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
