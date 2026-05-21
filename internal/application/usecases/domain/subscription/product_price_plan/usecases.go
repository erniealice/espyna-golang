package product_price_plan

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	productplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product_plan"
	priceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/price_plan"
	productpriceplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/product_price_plan"
)

// ProductPricePlanRepositories groups all repository dependencies for product price plan use cases
type ProductPricePlanRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer // Primary entity repository
	PricePlan        priceplanpb.PricePlanDomainServiceServer               // Entity reference dependency
	ProductPlan      productplanpb.ProductPlanDomainServiceServer           // Entity reference dependency (Model D)
}

// ProductPricePlanServices groups all business service dependencies for product price plan use cases
type ProductPricePlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// ReadProductPricePlanRepositories groups all repository dependencies
type ReadProductPricePlanRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
}

// ReadProductPricePlanServices groups all business service dependencies
type ReadProductPricePlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// DeleteProductPricePlanRepositories groups all repository dependencies
type DeleteProductPricePlanRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
}

// DeleteProductPricePlanServices groups all business service dependencies
type DeleteProductPricePlanServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// ListProductPricePlansRepositories groups all repository dependencies
type ListProductPricePlansRepositories struct {
	ProductPricePlan productpriceplanpb.ProductPricePlanDomainServiceServer
}

// ListProductPricePlansServices groups all business service dependencies
type ListProductPricePlansServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// UseCases contains all product price plan-related use cases
type UseCases struct {
	CreateProductPricePlan          *CreateProductPricePlanUseCase
	ReadProductPricePlan            *ReadProductPricePlanUseCase
	UpdateProductPricePlan          *UpdateProductPricePlanUseCase
	DeleteProductPricePlan          *DeleteProductPricePlanUseCase
	ListProductPricePlans           *ListProductPricePlansUseCase
	GetProductPricePlanListPageData *GetProductPricePlanListPageDataUseCase
	GetProductPricePlanItemPageData *GetProductPricePlanItemPageDataUseCase
}

// NewUseCases creates a new collection of product price plan use cases
func NewUseCases(
	repositories ProductPricePlanRepositories,
	services ProductPricePlanServices,
) *UseCases {
	createRepos := CreateProductPricePlanRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
		PricePlan:        repositories.PricePlan,
		ProductPlan:      repositories.ProductPlan,
	}
	createServices := CreateProductPricePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	readRepos := ReadProductPricePlanRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	readServices := ReadProductPricePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	updateRepos := UpdateProductPricePlanRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
		PricePlan:        repositories.PricePlan,
		ProductPlan:      repositories.ProductPlan,
	}
	updateServices := UpdateProductPricePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	deleteRepos := DeleteProductPricePlanRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	deleteServices := DeleteProductPricePlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listRepos := ListProductPricePlansRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	listServices := ListProductPricePlansServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	listPageDataRepos := GetProductPricePlanListPageDataRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	listPageDataServices := GetProductPricePlanListPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	itemPageDataRepos := GetProductPricePlanItemPageDataRepositories{
		ProductPricePlan: repositories.ProductPricePlan,
	}
	itemPageDataServices := GetProductPricePlanItemPageDataServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		CreateProductPricePlan:          NewCreateProductPricePlanUseCase(createRepos, createServices),
		ReadProductPricePlan:            NewReadProductPricePlanUseCase(readRepos, readServices),
		UpdateProductPricePlan:          NewUpdateProductPricePlanUseCase(updateRepos, updateServices),
		DeleteProductPricePlan:          NewDeleteProductPricePlanUseCase(deleteRepos, deleteServices),
		ListProductPricePlans:           NewListProductPricePlansUseCase(listRepos, listServices),
		GetProductPricePlanListPageData: NewGetProductPricePlanListPageDataUseCase(listPageDataRepos, listPageDataServices),
		GetProductPricePlanItemPageData: NewGetProductPricePlanItemPageDataUseCase(itemPageDataRepos, itemPageDataServices),
	}
}
